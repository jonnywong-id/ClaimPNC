package komite

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/platform/money"
)

// Rincian "Lihat Detail Transfer" — isi `Section/ShowTransferDetail`.
//
// ============================================================================
// KENAPA IA TIDAK DIBACA DARI TEMPAT YANG SAMA DENGAN PEGA
// ============================================================================
//
// Di sistem lama, seluruh medan di bawah datang dari CLIPBOARD objek kerja: menekan nomor
// case menjalankan `SetAssignmentKomite`, yang membuka objeknya dengan `OBJ-OPEN-BY-HANDLE`,
// dan `ShowTransferDetail` membaca `.Adjustment.*`, `.Komite.*`, dan `.Policy.*` dari sana.
//
// Isi clipboard itu tersimpan di BLOB Pega, dan blob tidak dapat dibaca SQL. Diverifikasi
// ke `ALL_TAB_COLUMNS`: dari 33 properti yang `ShowTransferDetail` tampilkan, hanya **6**
// yang benar-benar kolom pada `PC_ASM_FW_GCNMFW_WORK` — POLICYNO, QQNAME, BUSINESSNAME,
// BRANCHNAME, SOBNAME, dan DATEOFLOSS. Dua puluh tujuh sisanya tidak ada.
//
// Karena itu nilainya diambil dari tabel POOLDATA yang memuat data yang sama:
//
//	POOLDATA.T_CLAIM_ADJUSTMENT    nilai uang dan keadaan akseptasinya
//	POOLDATA.T_CLAIM_KOMITE_LIST   keputusan komite menurut Pega
//
// ============================================================================
// KENAPA CASEIDKOMITE, BUKAN CLAIMID
// ============================================================================
//
// `T_CLAIM_ADJUSTMENT` punya keduanya, dan keduanya berarti hal yang BERBEDA. Diukur pada
// basis data ASM atas dua belas case yang punya keduanya:
//
//	lewat CASEIDKOMITE   1 baris   pada dua belas-duanya
//	lewat CLAIMID        4–8 baris pada dua belas-duanya
//
// Tidak satu pun sama. `CLAIMID` mengembalikan SELURUH baris adjustment milik klaimnya —
// termasuk milik case komite lain, objek lain, dan coverage lain. Memakainya berarti
// menaruh empat sampai delapan nilai uang yang bukan urusan case ini di layar tempat orang
// menyetujui uang.
//
// `CASEIDKOMITE` namanya sendiri sudah menyatakannya: case komite mana yang memutuskan
// baris ini. Ia terisi pada baris ber-tanggal akseptasi 2019 sampai 2026, sehingga ia bukan
// kolom baru yang belum terpakai.
//
// # Yang harus disadari: tidak semua case punya barisnya
//
// Dari 189 case yang dapat muncul di inbox, **41** punya baris adjustment lewat kunci ini.
// Itu bukan cacat pembacaan melainkan keadaan datanya: baris adjustment lahir pada tahap
// tertentu, dan komite jenis Survey maupun Liable Klaim memang tidak memilikinya.
//
// Ketiadaannya karena itu ditampilkan sebagai KETERANGAN, bukan sebagai nol rupiah. Nol
// yang tidak dapat dibedakan dari "belum ada" adalah kesalahan paling mahal yang bisa ada
// pada layar ini.
type TransferDetail struct {
	// Lines adalah baris adjustment yang diputuskan case komite ini.
	//
	// Hampir selalu satu; bentuknya senarai karena kuncinya tidak menjamin demikian, dan
	// menganggapnya tunggal akan membuang baris kedua tanpa satu pun tanda.
	Lines []AdjustmentLine

	// Committee adalah keputusan komite MENURUT PEGA, dari `T_CLAIM_KOMITE_LIST`.
	Committee CommitteeRecord

	// HasCommitteeRecord membedakan "tidak ada barisnya" dari "ada tetapi kosong".
	HasCommitteeRecord bool

	// GroupPanel adalah `GROUPPANEL_1` pada baris kerja case komite ini.
	//
	// Ia ada di sini semata-mata karena judul layar membutuhkannya: `ShowTransfer`
	// menyembunyikan akhiran "- ADJUSTMENT" ketika `IsTravel` benar, dan `IsTravel`
	// tidak lain adalah `GroupPanel = "005"`.
	//
	// Terisi pada 610 dari 610 case komite sejak 2024, jadi cabang itu dapat dinilai.
	GroupPanel string

	// Claim adalah data klaim yang `ShowTransferDetail` tampilkan lewat
	// `.KomiteClaimData.*` — tanggal kejadian, tanggal register, lokasi, kronologi.
	//
	// Terbaca pada 189 dari 189 case inbox.
	Claim    ClaimSummary
	HasClaim bool

	// Coverages adalah blok analisis komite: `.COVERAGE`, `.Komite.ExtentOfLoss`,
	// `.Komite.CircumtansesCouseOfLoss`, `.Komite.Remarks`, dan TSI objeknya.
	//
	// Senarai, karena satu klaim dapat punya beberapa coverage. Diukur: 1 sampai 3 baris,
	// rerata 1,4 — tidak satu pun case melampaui tiga, sehingga seluruhnya dapat
	// ditampilkan tanpa paginasi.
	Coverages []CoverageAnalysis

	// BusinessType adalah `BUSINESSTYPE`, yang `IsHE` bandingkan terhadap "HE" dan
	// "ContractorsPM" untuk memunculkan `ShowTransferDetailHE`.
	//
	// # Ia KOSONG pada seluruh case komite
	//
	// Diukur pada basis data ASM: terisi 0 dari 610. Kolomnya ada, tetapi Pega tidak
	// pernah menulisinya untuk kelas `Work-Komite` — nilainya hidup di clipboard.
	//
	// Medan ini tetap dibaca supaya cabang HE menjadi benar dengan sendirinya bila kolom
	// itu kelak terisi, dan supaya layar dapat menyatakan dengan jujur bahwa cabang itu
	// TIDAK DAPAT dinilai, alih-alih diam-diam menganggapnya salah.
	BusinessType string
}

// Judul menyusun judul layar persis seperti `Section/ShowTransfer` menyusunnya.
//
// # Bentuknya di Pega
//
// `ShowTransfer` memuat tujuh sel label berdampingan: satu tanpa syarat berisi
// "CLAIM COMMITTEE", dan enam lagi masing-masing dengan syaratnya sendiri. Yang syaratnya
// terpenuhi ikut tampil, berurutan. Karena itu fungsi ini MENGGABUNG, bukan memilih satu —
// dua akhiran yang syaratnya sama-sama benar memang tampil berdua di sistem lama.
//
//	"- SURVEY"       .TransferType == '1'
//	"- ADJUSTMENT"   .TransferType == 2 && !IsTravel && .Adjustment.PaymentType != '6'
//	"- REJECT"       .TransferType == '3'
//	"- LIABILITY"    .TransferType == '4'
//	"- FINAL"        .TransferType == '5'
//	"- TOLAK KLAIM"  .Adjustment.PaymentType == '6'
//
// # Kenapa TYPEKOMITE dipakai sebagai ganti .TransferType
//
// `.TransferType` hidup di clipboard, jadi ia tidak terbaca SQL. `T_CLAIM_KOMITE_LIST`
// menyimpannya sebagai `TYPEKOMITE` — nilai yang ditulis `InsertUpdateKomiteList` dari
// `TempKomite.TransferType`, dan yang `ShowKomiteTerimaTolakNonMBU` baca dengan domain kode
// yang sama persis (1..5). Diukur: tidak satu pun case punya dua `TYPEKOMITE` berbeda,
// sehingga agregasi `MAX` pada kueri tidak menyembunyikan pilihan apa pun.
//
// # Satu label yang SENGAJA berbeda dari "Tipe Komite"
//
// Kode `3` di sini berbunyi "- REJECT", sementara [CommitteeKindOf] menyebutnya
// "Ex Gratia". Keduanya benar: keduanya rule yang berbeda dengan caption yang berbeda atas
// kode yang sama. Menyeragamkannya berarti mengarang salah satunya (`P-5`).
func (d TransferDetail) Judul() string {
	judul := "CLAIM COMMITTEE"
	for _, akhiran := range d.akhiranJudul() {
		judul += " " + akhiran
	}
	return judul
}

func (d TransferDetail) akhiranJudul() []string {
	if !d.HasCommitteeRecord {
		return nil
	}

	tipe := kodeRingkas(d.Committee.TransferTypeCode)
	bayar := kodeRingkas(d.Committee.PaymentTypeCode)
	travel := strings.TrimSpace(d.GroupPanel) == "005"

	var akhiran []string
	if tipe == "1" {
		akhiran = append(akhiran, "- SURVEY")
	}
	if tipe == "2" && !travel && bayar != "6" {
		akhiran = append(akhiran, "- ADJUSTMENT")
	}
	if tipe == "3" {
		akhiran = append(akhiran, "- REJECT")
	}
	if tipe == "4" {
		akhiran = append(akhiran, "- LIABILITY")
	}
	if tipe == "5" {
		akhiran = append(akhiran, "- FINAL")
	}
	if bayar == "6" {
		akhiran = append(akhiran, "- TOLAK KLAIM")
	}
	return akhiran
}

// kodeRingkas membuang spasi dan nol di depan sebuah kode.
//
// Nilai yang teramati di basis data tidak berimbuh nol ("1".."6"), tetapi [paymentKind]
// sudah memangkasnya sejak semula. Keduanya dibuat memangkas dengan cara yang sama supaya
// satu baris data tidak pernah menghasilkan jenis komite dan judul yang saling bertentangan.
func kodeRingkas(kode string) string {
	ringkas := strings.TrimLeft(strings.TrimSpace(kode), "0")
	if ringkas == "" {
		return strings.TrimSpace(kode)
	}
	return ringkas
}

// Empty menyatakan tidak ada satu pun rincian yang dapat ditampilkan.
func (d TransferDetail) Empty() bool {
	return len(d.Lines) == 0 && !d.HasCommitteeRecord &&
		!d.HasClaim && len(d.Coverages) == 0
}

// MoneyEmpty menyatakan tidak ada satu pun NILAI UANG yang dapat ditampilkan.
//
// Dipisahkan dari Empty sejak blok klaim dan coverage masuk: sebuah case dapat punya
// data klaim lengkap tanpa satu pun baris adjustment, dan menyamakan keduanya akan
// menyembunyikan seluruh layar hanya karena angkanya belum ada.
func (d TransferDetail) MoneyEmpty() bool {
	return len(d.Lines) == 0 && !d.HasCommitteeRecord
}

// ClaimSummary adalah baris `POOLDATA.T_CLAIM_PNC` milik klaim yang dikomitekan.
//
// # Kuncinya membawa prefix kelas Pega, dan itu WAJIB
//
// `T_CLAIM_PNC.CLAIMID` berbunyi `ASM-FW-GCNMFW-WORK PNC-1670`, bukan `PNC-1670`. Menjoin
// dengan nomor yang sudah dipangkas menghasilkan **nol baris pada seluruh 189 case** —
// tanpa galat apa pun, hanya layar kosong. Itu persis utang teknis 4.1 pada Steering:
// nama kelas internal Pega tertanam di dalam kunci data bisnis.
type ClaimSummary struct {
	DateOfLoss   time.Time
	RegisterDate time.Time

	Location    string
	Chronology  string
	ClaimStatus string

	// Recommendation adalah `REMARKRECOMENDATION` — `.KomiteClaimData.RemarkRecommendation`.
	// Terisi 3 dari 72 klaim, jadi kosongnya adalah keadaan wajar.
	Recommendation string

	ASMShare  string
	CoinsName string
	Currency  string
	ExGratia  string
}

// CoverageAnalysis adalah satu baris `POOLDATA.T_CLAIM_OBJECTCOVERAGE`.
//
// # Di sinilah blok `.Komite.*` ShowTransferDetail sebenarnya tinggal
//
// Properti yang di Pega berbunyi `.Komite.ExtentOfLoss`, `.Komite.CircumtansesCouseOfLoss`,
// dan `.Komite.Remarks` adalah kolom pada tabel ini — bukan pada tabel kerja, dan bukan
// pada `T_CLAIM_KOMITE_LIST`.
//
// # Yang terisi, dan yang tidak — keduanya diukur
//
// Dari 79 baris coverage milik 189 case inbox:
//
//	COVERAGENAME · CAUSEOFLOSS · SUMTSI    79  (seluruhnya)
//	EXTENTOFLOSS                           21
//	CURICUMOFLOSS · REMARKS                11
//	LEGALLIABILITY · DIAGNOSE ·
//	INITIALNAME · TANGGALCOMITEE            0
//
// Keempat yang terakhir tetap dibaca. Biayanya nol, dan ia membuat layar benar dengan
// sendirinya bila kolomnya kelak terisi — sementara menghapusnya berarti menebak bahwa ia
// tidak akan pernah terisi.
type CoverageAnalysis struct {
	ObjectID   string
	CoverageID string

	ObjectName   string
	CoverageName string
	CauseOfLoss  string

	SumInsured money.Money
	Currency   string

	Circumstances  string
	ExtentOfLoss   string
	LegalLiability string
	Remarks        string
	Diagnose       string
	InitialName    string

	CommitteeDate time.Time
}

// Filled menyatakan baris ini punya keterangan analisis, bukan hanya nama coverage.
//
// Dipakai layar untuk memutuskan apakah blok analisisnya digambar sama sekali — 58 dari 79
// baris tidak punya satu pun, dan judul blok kosong lebih buruk daripada tidak ada blok.
func (c CoverageAnalysis) Filled() bool {
	for _, teks := range []string{
		c.Circumstances, c.ExtentOfLoss, c.LegalLiability,
		c.Remarks, c.Diagnose, c.InitialName,
	} {
		if strings.TrimSpace(teks) != "" {
			return true
		}
	}
	return false
}

// AdjustmentLine adalah satu baris `POOLDATA.T_CLAIM_ADJUSTMENT`.
//
// Penamaannya mengikuti arti kolomnya, bukan nama kolomnya (`D-19`). Pemetaan ke nama
// aslinya hanya ada di repo/sqlstore.
type AdjustmentLine struct {
	ClaimNumber string
	ObjectID    string
	CoverageID  string

	// AcceptanceNo dan AcceptedAt terisi setelah akseptasi; kosong sebelum itu.
	AcceptanceNo string
	AcceptedAt   time.Time

	Currency    string
	PaymentType string

	// Keenam nilai uang di bawah `money.Money`, tidak pernah float — `I-12` menetapkan
	// nilai uang disimpan presisi penuh dan dibulatkan hanya saat ditampilkan.
	GrossValue     money.Money
	ProposeValue   money.Money
	AcceptedValue  money.Money
	SalvageValue   money.Money
	ASMShareValue  money.Money
	IndividualRisk money.Money

	// ASMSharePercent adalah persentase, bukan nilai uang.
	ASMSharePercent string

	ExGratia bool
	Notes    string

	// CauseOfLoss adalah `CIRCUMCAUSEOFLOSS` — keterangan sebab kerugian yang dibaca
	// komite, bukan kode Penyebab Kerugian pada master.
	CauseOfLoss string
}

// CommitteeRecord adalah satu baris `POOLDATA.T_CLAIM_KOMITE_LIST`.
type CommitteeRecord struct {
	// MemberName adalah `NAMAKOMITE` — anggota komite yang tercatat di Pega.
	MemberName string

	// Tier adalah `KOMITEKE`, jenjang menurut Pega.
	Tier int

	// Kind adalah "Tipe Komite" yang dilihat pengguna, hasil penurunan `TYPEKOMITE` dan
	// `PAYMENTTYPE` — lihat CommitteeKindOf.
	Kind string

	// TransferTypeCode dan PaymentTypeCode adalah `TYPEKOMITE` dan `PAYMENTTYPE` MENTAH.
	//
	// Keduanya disimpan berdampingan dengan Kind yang sudah diturunkan darinya karena dua
	// rule Pega menurunkan hal yang berbeda dari kode yang sama: `ShowKomiteTerimaTolakNonMBU`
	// menghasilkan "Tipe Komite", `ShowTransfer` menghasilkan akhiran judul, dan kode `3`
	// berbunyi "Ex Gratia" pada yang pertama serta "REJECT" pada yang kedua.
	//
	// Membuang kode mentahnya berarti salah satu dari keduanya harus menebak dari hasil
	// penurunan yang lain.
	TransferTypeCode string
	PaymentTypeCode  string

	// Note adalah `NOTEKOMITE`, catatan komite saat memutuskan.
	Note string

	// ClaimValue dan ASMShare adalah nilai yang dibandingkan terhadap ambang penjenjangan.
	ClaimValue money.Money
	ASMShare   string

	DecidedAt time.Time
	Outcome   Outcome
}

// CommitteeKindOf menurunkan "Tipe Komite" dari dua kolom `T_CLAIM_KOMITE_LIST`.
//
// # Aturannya disalin apa adanya dari SQL lama
//
// `RDB List/ShowKomiteTerimaTolakNonMBU-SQL.xml` menurunkannya lewat CASE bersarang atas
// `MAX(TYPEKOMITE)` dan `MAX(PAYMENTTYPE)`, lalu — inilah bagian yang menyesatkan —
// menaruh hasilnya pada property bernama `StatusKlaim`, yang di layar diberi caption
// "Tipe Komite". Ia sama sekali bukan Status Klaim dalam arti `D-18`.
//
// # Kenapa dihitung di Go, bukan dibiarkan di SQL
//
// Bentuk aslinya menjalankan ENAM subkueri berkorelasi ke tabel yang sama untuk satu
// baris — `(select max(PAYMENTTYPE) …)` diulang sekali per cabang. Di sini ia satu tabel
// keputusan yang dapat diuji tanpa basis data.
//
// Nilai yang tidak dikenali jatuh ke "Survey Komite", persis cabang `else` terakhir pada
// rule aslinya — ditiru, bukan diperbaiki, karena menebak apa yang SEHARUSNYA terjadi pada
// tipe yang tidak dikenal berarti mengarang aturan.
func CommitteeKindOf(committeeType, paymentType string) string {
	switch strings.TrimSpace(committeeType) {
	case "1":
		return "Survey Komite"
	case "2":
		return paymentKind(paymentType)
	case "3":
		return "Ex Gratia"
	case "4":
		return "Liable Klaim"
	case "5":
		return "Final"
	default:
		return "Survey Komite"
	}
}

// paymentKind menerjemahkan PAYMENTTYPE, yang hanya bermakna saat TYPEKOMITE = 2.
//
// Angkanya tersimpan sebagai bilangan dan dibandingkan sebagai bilangan di rule aslinya,
// sehingga "01" dan "1" adalah hal yang sama. Perbandingan teks di sini akan membuat
// keduanya berbeda — karena itu ia diurai lebih dulu.
func paymentKind(paymentType string) string {
	switch kodeRingkas(paymentType) {
	case "1":
		return "Final"
	case "2":
		return "Interim"
	case "3":
		return "Salvage"
	case "4":
		return "Adjuster Fee"
	case "5":
		return "Adjustment"
	case "6":
		return "Tolak Klaim"
	default:
		return "Collection Fee"
	}
}

// TransferRepo adalah seam ke rincian transfer.
//
// Ia TERPISAH dari InboxRepo meski keduanya membaca tabel warisan, dan pemisahannya
// mengikuti apa yang dibaca: InboxRepo melayani DAFTAR dan dibaca setiap kali layar
// dibuka; ini melayani SATU kasus dan hanya dibaca ketika seseorang menekannya.
//
// Menyatukannya akan memaksa setiap pemakai daftar ikut menyediakan bahan yang tidak
// pernah ia pakai.
type TransferRepo interface {
	// FindTransfer mengembalikan rincian transfer satu case komite.
	//
	// Case yang belum punya baris adjustment BUKAN galat — ia keadaan yang wajar, dan
	// dikembalikan sebagai TransferDetail kosong. Menjadikannya galat akan membuat 148
	// dari 189 case gagal dibuka.
	FindTransfer(ctx context.Context, caseID string) (TransferDetail, error)
}
