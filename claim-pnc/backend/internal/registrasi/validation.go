package registrasi

import (
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// # Gerbang validasi Input Register
//
// Berkas ini adalah pembacaan ulang 137 langkah `Activity/InputRegister_act-Act.xml`.
// Setiap aturan menyebut langkah asalnya, supaya siapa pun dapat memeriksa apakah
// pembacaannya benar tanpa mempercayai berkas ini.
//
// # Tiga hal yang sengaja berbeda dari sistem lama
//
//  1. SELURUH pelanggaran dikumpulkan, bukan berhenti pada yang pertama. Urutannya tetap
//     urutan langkah Pega, sehingga pelanggaran pertama selalu sama (lihat ValidationError).
//  2. TIDAK ADA penambahan tujuh jam. Sistem lama menggeser sebagian nilai tanggal
//     sebesar tujuh jam sebelum membandingkannya — dan pada langkah 21 pergeseran itu
//     diterapkan hanya pada SATU sisi perbandingan. Di sini seluruh perbandingan tanggal
//     memakai tanggal kalender WIB lewat `platform/waktu`.
//  3. Objek tanpa coverage DITOLAK. Sistem lama membuangnya diam-diam (langkah 4.4:
//     `Page-Remove` bila `SizeOfPropertyList(.ObjectCoverageList)` bernilai nol),
//     sehingga petugas menyimpan klaim lalu mendapati objeknya hilang tanpa pesan.
//     `TKT-B02-004` menuntut penolakan.

// DuplicateClaim adalah klaim lain yang sudah terdaftar dengan kunci yang sama.
type DuplicateClaim struct {
	Number      string
	InsuredItem string
}

// Parts adalah keterangan di luar klaim yang dibutuhkan validasi.
//
// Ia sengaja berupa nilai, bukan seam: validasi harus dapat dijalankan tanpa basis data
// dan tanpa jaringan. Pengambilan datanya adalah urusan usecase.
type Parts struct {
	// Now adalah waktu berjalan, dalam UTC. Ia datang dari seam Jam, bukan dari
	// time.Now, supaya kasus batas tengah malam WIB dapat diuji.
	Now time.Time

	// Duplicates adalah klaim yang sudah ada dengan kunci duplikasi yang sama.
	Duplicates []DuplicateClaim
}

// TravelPAPeriodTolerance adalah jumlah hari setelah polis berakhir yang masih
// membolehkan klaim Travel dan Personal Accident DIDAFTARKAN hari ini.
//
// PENTING — angka ini BUKAN toleransi tanggal kejadian. `BRD §11.1` mencatatnya sebagai
// toleransi periode polis terhadap tanggal kejadian; source berkata lain. Langkah 18
// membandingkan HARI INI, bukan tanggal kejadian:
//
//	IsTravelPA AND ( hari_ini > akhir_polis + 90  OR  awal_polis > hari_ini )
//
// Tanggal kejadian tetap diperiksa terhadap periode polis TANPA toleransi apa pun oleh
// langkah 21, yang berlaku untuk seluruh lini termasuk Travel dan PA.
const TravelPAPeriodTolerance = 90

// TravelDocumentReceiptLimit adalah jarak hari terbesar antara tanggal kejadian dan
// tanggal terima dokumen untuk lini Travel (langkah 19).
const TravelDocumentReceiptLimit = 90

// ReportAfterLossLimit adalah jarak hari yang membuat tanggal lapor ditolak
// (langkah 20).
//
// PERHATIKAN OPERATORNYA. Pesan di sistem lama berbunyi "tidak boleh lebih dari 7 hari",
// tetapi kondisinya menolak sejak hari ke-7, bukan setelahnya:
//
//	skip bila  kejadian + 7 > lapor      →  galat bila  lapor >= kejadian + 7
//
// Perilakunya dibawa apa adanya (`P-5`); selisih antara pesan dan aturan dicatat
// sebagai calon perbaikan, bukan diperbaiki diam-diam.
const ReportAfterLossLimit = 7

// Validate menjalankan seluruh aturan tahap Input Register.
//
// Ia mengembalikan *ValidationError bila ada aturan yang dilanggar, dan nil bila klaim
// dapat disimpan.
func Validate(k Claim, b Parts) error {
	v := &collector{}

	validateSLIKNumber(v, k)
	validateDeclarationPolicy(v, k)
	validatePolicyPeriodActive(v, k, b.Now)
	validateDates(v, k, b.Now)
	validateReporterStatus(v, k)
	validateDuplicateClaim(v, b.Duplicates)
	validateItemsAndCoverage(v, k)
	validateSpreading(v, k)
	validateEstimateAgainstTSI(v, k)

	if len(v.violations) == 0 {
		return nil
	}
	return &ValidationError{Violation: v.violations}
}

type collector struct{ violations []Violation }

func (p *collector) add(code ViolationCode, field, messages string) {
	p.violations = append(p.violations, Violation{Code: code, Field: field, Message: messages})
}

// validateSLIKNumber — langkah 4.2 dan 5.
//
// Prasyaratnya rangkap tiga: aplikasi bernama `PEGAKREDIT`, rule `IsAsuransiKredit`, dan
// `NoSLIK` kosong. Rule `IsAsuransiKredit` TIDAK ADA di export (`R-16`); penandanya di
// sini adalah `Polis.CreditGuarantee`, yang diisi seam PolicyRepo. Bila kelak rule itu
// tiba dan ternyata menguji sesuatu yang lain, yang berubah hanya pengisian field ini.
func validateSLIKNumber(v *collector, k Claim) {
	if !k.Policy.CreditGuarantee {
		return
	}
	if strings.TrimSpace(k.SLIKNumber) != "" {
		return
	}
	v.add(ViolationSLIKNumberEmpty, "nomor_slik", "Nomor SLIK Harus Diisi")
}

// validateDeclarationPolicy — langkah 10.
func validateDeclarationPolicy(v *collector, k Claim) {
	if !k.Policy.Declaration {
		return
	}
	v.add(ViolationDeclarationPolicy, "nomor_polis", "Polis Deklarasi tidak dapat diklaim")
}

// validatePolicyPeriodActive — langkah 18, khusus lini Travel dan Personal Accident.
func validatePolicyPeriodActive(v *collector, k Claim, now time.Time) {
	if k.Policy.Line != LineTravel && k.Policy.Line != LinePersonalAccident {
		return
	}
	today := clock.DateWIB(now)
	start := clock.DateWIB(k.Policy.CoverageStart)
	limit := clock.AddDays(k.Policy.CoverageEnd, TravelPAPeriodTolerance)

	if today.Before(start) || today.After(limit) {
		v.add(ViolationPolicyPeriodEnded, "tanggal_kejadian", "Periode Polis sudah berakhir")
	}
}

// validateDates menjalankan aturan tanggal pada urutan yang sama dengan sistem lama:
// langkah 19, 20, 21, 22, 23, 24, 26, dan 27.
//
// # Langkah 25 sengaja tidak ada di sini
//
// Langkah 25 memeriksa tanggal kejadian terhadap `akhir_polis + 30 hari` dengan
// prasyarat `IsBonding`, dan `BRD §11.1` menyebutnya toleransi 30 hari untuk Bonding.
// Ia TIDAK PERNAH dapat menjadi aturan yang menentukan, karena:
//
//   - langkah 21 memeriksa batas yang LEBIH KETAT (`akhir_polis`, tanpa toleransi),
//   - langkah 21 tidak punya prasyarat lini sama sekali sehingga berlaku juga untuk
//     Bonding, dan
//   - transisi langkah 21 adalah `exit activity`, sehingga langkah 25 tidak pernah
//     dijalankan pada kasus yang seharusnya ia toleransi.
//
// Dengan kata lain toleransi 30 hari Bonding **mati** — ini menjawab pertanyaan terbuka
// pada `TKT-B02-002`. Membawanya ke sistem baru berarti menambahkan aturan yang tidak
// pernah berlaku di produksi.
func validateDates(v *collector, k Claim, now time.Time) {
	today := clock.DateWIB(now)
	lossDate := clock.DateWIB(k.DateOfLoss)
	reportDate := clock.DateWIB(k.ReportDate)
	receivedDate := clock.DateWIB(k.DateReceived)

	// Langkah 19 — hanya Travel.
	if k.Policy.Line == LineTravel {
		if clock.DaysBetween(lossDate, receivedDate) > TravelDocumentReceiptLimit {
			v.add(ViolationReceivedAfter90Days, "tanggal_terima_dokumen",
				fmt.Sprintf("Tanggal Terima Dokumen tidak boleh lebih dari %d hari setelah Tanggal Kejadian.",
					TravelDocumentReceiptLimit))
		}
	}

	// Langkah 20 — seluruh lini KECUALI Personal Accident.
	if k.Policy.Line != LinePersonalAccident {
		if clock.DaysBetween(lossDate, reportDate) >= ReportAfterLossLimit {
			v.add(ViolationReportedAfter7Days, "tanggal_lapor",
				fmt.Sprintf("Tanggal Lapor tidak boleh lebih dari %d hari setelah Tanggal Kejadian.",
					ReportAfterLossLimit))
		}
	}

	// Langkah 21 — tanggal kejadian di dalam periode polis.
	//
	// Di sistem lama sisi kiri perbandingan memakai nilai mentah dan sisi kanan memakai
	// nilai yang sudah digeser tujuh jam, di dalam SATU kondisi yang sama. Akibatnya
	// klaim yang tanggal kejadiannya tepat di batas periode bisa lolos atau ditolak
	// tergantung jam berapa datanya tersimpan. Kedua sisi di sini dipotong ke tanggal
	// WIB lebih dulu, sehingga hasilnya sama untuk seluruh jam pada hari yang sama.
	// Inilah butir 3 dari 13 perbaikan `P-5` (`D-49`, `ADR-0017`).
	policyStart := clock.DateWIB(k.Policy.CoverageStart)
	policyEnd := clock.DateWIB(k.Policy.CoverageEnd)
	if lossDate.Before(policyStart) || lossDate.After(policyEnd) {
		v.add(ViolationLossOutsidePolicyPeriod, "tanggal_kejadian",
			"Tanggal Kejadian tidak dalam range polis.")
	}

	// Langkah 22 — kejadian tidak boleh setelah lapor. Tanggal yang sama tidak masalah.
	if lossDate.After(reportDate) {
		v.add(ViolationReportDateBeforeLoss, "tanggal_lapor",
			"Tanggal Lapor harus setelah Tanggal Kejadian.")
	}

	// Langkah 23 — lapor tidak boleh setelah terima dokumen.
	if reportDate.After(receivedDate) {
		v.add(ViolationReceivedBeforeReport, "tanggal_terima_dokumen",
			"Tanggal Terima Dokumen harus setelah Tanggal Lapor")
	}

	// Langkah 24, 26, 27 — ketiganya tidak boleh melewati hari ini.
	if lossDate.After(today) {
		v.add(ViolationLossDateInFuture, "tanggal_kejadian",
			"Tanggal Kejadian tidak boleh lebih dari tanggal hari ini.")
	}
	if reportDate.After(today) {
		v.add(ViolationReportDateInFuture, "tanggal_lapor",
			"Tanggal Lapor tidak boleh lebih dari tanggal hari ini.")
	}
	if receivedDate.After(today) {
		v.add(ViolationReceivedInFuture, "tanggal_terima_dokumen",
			"Tanggal Terima Dokumen tidak boleh lebih dari tanggal hari ini.")
	}
}

// validateReporterStatus — langkah 28.
func validateReporterStatus(v *collector, k Claim) {
	if k.Reporter.Relation != RelationOther {
		return
	}
	if strings.TrimSpace(k.Reporter.OtherRelation) != "" {
		return
	}
	v.add(ViolationReporterStatusEmpty, "hubungan_lainnya", "Pilih Status Pelapor.")
}

// validateDuplicateClaim — langkah 32 dan 33.
//
// Kunci duplikasinya disusun usecase, bukan di sini: ia menyentuh basis data. Yang ada
// di sini hanya perlakuan atas hasilnya — dan perlakuan itu adalah menyebut NOMOR KLAIM
// yang sudah ada, supaya petugas dapat memeriksanya alih-alih sekadar ditolak.
func validateDuplicateClaim(v *collector, duplicate []DuplicateClaim) {
	for _, g := range duplicate {
		v.add(ViolationDuplicateClaim, "nomor_polis",
			"Nomor Polis Sudah Terdaftar dengan no Klaim "+g.Number)
	}
}

// validateItemsAndCoverage — langkah 4.4 dan 37.3.3.
func validateItemsAndCoverage(v *collector, k Claim) {
	for _, o := range k.InsuredItem {
		if len(o.Coverage) == 0 {
			v.add(ViolationItemWithoutCoverage, "objek",
				"Objek "+itemName(o)+" tidak memiliki coverage.")
			continue
		}
		// Penyebab kerugian wajib untuk seluruh lini KECUALI Travel.
		if k.Policy.Line == LineTravel {
			continue
		}
		for _, c := range o.Coverage {
			if strings.TrimSpace(c.CauseOfLoss) == "" {
				v.add(ViolationCauseOfLossEmpty, "penyebab_kerugian",
					"Penyebab Kerugian harus di isi")
			}
		}
	}
}

func itemName(o InsuredItem) string {
	if strings.TrimSpace(o.Name) != "" {
		return o.Name
	}
	return o.ID
}

// spreadingLowerBound dan spreadingUpperBound adalah toleransi total share yang
// `ADR-0016` tetapkan: `ROUND(SUM(share), 4) BETWEEN 99.9999 AND 100.0001`.
//
// Ia menggantikan pencocokan substring pada langkah 37.3.6 — `@contains(total, 100.0) ||
// total == 100 || @contains(total, 99.99)` — yang meloloskan `199.99` dan `1100.0`
// karena keduanya MEMUAT teks yang dicari. Itu butir pertama daftar perbaikan `D-49`.
const (
	spreadingLowerBound Percent = 999_999
	spreadingUpperBound Percent = 1_000_001
)

// validateSpreading — langkah 37.3.2, 37.3.5.3, 37.3.5.4, dan 37.3.6.
func validateSpreading(v *collector, k Claim) {
	if k.SpreadingCount() == 0 {
		v.add(ViolationNoSpreading, "spreading",
			"Tidak ada Spreading dalam No Polis ini. Silahkan cek nopolis kembali dan koordinasikan ke bagian UW/MKT")
		return
	}

	for _, c := range k.AllCoverages() {
		for _, s := range c.Spreading {
			if s.Removed || s.TreatyKind != TreatyFacOut {
				continue
			}
			if strings.TrimSpace(s.FacOfferItem) == "" {
				v.add(ViolationFacOfferIncomplete, "spreading",
					"Data Spreading Facout : Object Name belum lengkap.")
			}
		}
	}

	total := k.TotalSpreading()
	if total < spreadingLowerBound || total > spreadingUpperBound {
		v.add(ViolationSpreadingTotalNot100, "spreading",
			"Total share spreading tidak 100%. Silakan cek spreading kembali")
	}
}

// validateEstimateAgainstTSI — pesan `local.errorEstimation` pada langkah 7, dipakai
// activity `RegisterObjectValidation` yang dipanggil langkah 30.
//
// Activity itu ada di export tetapi rincian perbandingannya berada di 63 langkahnya
// sendiri; yang dibawa di sini adalah aturan yang `TKT-B02-004` nyatakan sebagai
// kriteria penerimaan: nilai estimasi tidak boleh melebihi TSI, dan pesannya menyebut
// TSI-nya.
func validateEstimateAgainstTSI(v *collector, k Claim) {
	tsi := k.HighestTSI()
	if tsi == 0 || k.EstimateValue <= tsi {
		return
	}
	v.add(ViolationEstimateExceedsTSI, "nilai_estimasi",
		fmt.Sprintf("Nilai Estimasi Klaim harus lebih kecil dari Nilai TSI. TSI: %s", FormatRupiah(tsi)))
}

// FormatRupiah menuliskan nilai uang dengan pemisah ribuan titik dan dua desimal koma,
// sesuai kebiasaan penulisan Indonesia.
//
// Pembulatan hanya terjadi di sini — saat ditampilkan — sesuai `ADR-0016`.
func FormatRupiah(u Money) string {
	negative := u < 0
	if negative {
		u = -u
	}
	rupiah := int64(u) / 100
	cents := int64(u) % 100

	number := fmt.Sprintf("%d", rupiah)
	var b strings.Builder
	for i, r := range number {
		if i > 0 && (len(number)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	result := fmt.Sprintf("Rp %s,%02d", b.String(), cents)
	if negative {
		return "-" + result
	}
	return result
}
