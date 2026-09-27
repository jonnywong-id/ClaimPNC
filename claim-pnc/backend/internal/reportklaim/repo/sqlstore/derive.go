package sqlstore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

// deriveBuilder menyusun penghitung kolom turunan untuk satu kali jalan laporan.
//
// # Kenapa pembangun, bukan fungsi biasa
//
// Karena sebagian kolom turunan membutuhkan BAHAN yang diambil sekali di awal, bukan
// sekali per baris. Kalender libur laporan TAT contohnya: ia dibaca sekali untuk seluruh
// rentang, lalu dipakai ratusan ribu kali. Menjadikannya fungsi biasa berarti setiap
// baris membaca ulang kalender yang sama.
type deriveBuilder func(ctx context.Context, r *Repo, f reportklaim.Filter) (func(reportklaim.Row), error)

// staticDerive membungkus penghitung yang tidak butuh bahan apa pun.
func staticDerive(fn func(reportklaim.Row)) deriveBuilder {
	return func(context.Context, *Repo, reportklaim.Filter) (func(reportklaim.Row), error) {
		return fn, nil
	}
}

// tanggalCSV adalah bentuk tanggal yang ditulis ke berkas — sama dengan yang dihasilkan
// fungsi text pada reportklaim.go.
const tanggalCSV = "02/01/2006"

// bacaTanggalCSV mengurai kembali sel tanggal menjadi nilai waktu.
//
// # Kenapa diurai kembali, bukan dibawa sebagai waktu
//
// Baris hasil kueri sudah berupa teks saat sampai ke sini — itu bentuk yang ditulis ke
// CSV, dan menyimpannya dua kali (sebagai teks dan sebagai waktu) berarti dua nilai yang
// dapat berbeda. Menguraikannya kembali hanya terjadi pada kolom yang benar-benar dipakai
// berhitung, dan bentuknya tetap satu.
func bacaTanggalCSV(nilai string) time.Time {
	nilai = strings.TrimSpace(nilai)
	if nilai == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(tanggalCSV, nilai, clock.ZoneWIB)
	if err != nil {
		return time.Time{}
	}
	return t
}

// tanggalDariTeksPadat mengubah teks berbentuk YYYYMMDD menjadi dd/mm/yyyy.
//
// Sistem lama melakukannya dengan memotong teks — `@substring(.PRODKE,6,8)+"/"+…` —
// bukan dengan fungsi tanggal, karena nilainya memang tersimpan sebagai teks dan bukan
// sebagai kolom tanggal. Perlakuan itu ditiru: yang tidak berbentuk delapan angka
// dikembalikan sebagai sel kosong, bukan dipaksa menjadi tanggal.
func tanggalDariTeksPadat(nilai string) string {
	nilai = strings.TrimSpace(nilai)
	if len(nilai) != 8 {
		return ""
	}
	for _, r := range nilai {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return nilai[6:8] + "/" + nilai[4:6] + "/" + nilai[0:4]
}

// bulanSingkat adalah singkatan bulan tiga huruf pada kolom "Bulan Close" dan
// "Bulan Reject".
//
// # Bentuknya ditiru, caranya tidak
//
// Kueri aslinya menulis `SUBSTR(a.closeclaimdate, 4, 3)` — memotong tiga huruf dari
// tanggal yang diubah menjadi teks memakai format bawaan basis data. Hasilnya bergantung
// pada pengaturan NLS server, dan itu tidak dapat ditiru secara portabel maupun diuji.
//
// Yang dipakai di sini adalah singkatan bulan bahasa Inggris huruf besar — bentuk yang
// dihasilkan Oracle pada pengaturan bawaannya, dan bentuk yang selama ini diterima
// pembaca berkasnya.
func bulanSingkat(tanggal time.Time) string {
	if tanggal.IsZero() {
		return ""
	}
	return strings.ToUpper(tanggal.Format("Jan"))
}

// desimalKoma menulis angka dengan koma sebagai pemisah desimal.
//
// Sistem lama melakukannya dengan `@replaceAll(nilai,".",",")` pada kolom nilai uang
// tertentu. Ia ditiru apa adanya, termasuk keterbatasannya: yang diganti hanya titik,
// sehingga angka yang sudah memakai titik sebagai pemisah ribuan akan ikut berubah.
// Memperbaikinya berarti mengubah bentuk angka pada berkas yang dibaca ulang berkas kerja
// penggunanya.
func desimalKoma(nilai string) string {
	return strings.ReplaceAll(nilai, ".", ",")
}

// usiaPadaTanggal menghitung usia peserta dalam tahun penuh pada tanggal kejadian.
//
// # Kenapa dihitung di sini, bukan di SQL
//
// Kueri aslinya memakai `TRUNC(MONTHS_BETWEEN(TRUNC(dateofloss), dob) / 12)`, dan
// `MONTHS_BETWEEN` termasuk padanan wajib `09-DATABASE-STRATEGY.md` §4 yang harus
// dihitung di Go — ia tidak ada di PostgreSQL.
//
// Hasilnya sama: tahun penuh yang sudah dilewati, bukan pembulatan ke tahun terdekat.
// Seseorang yang berulang tahun sehari setelah kejadian tetap tercatat berusia setahun
// lebih muda, dan itu memang yang dimaksud — usia PADA SAAT kejadian.
//
// Tanggal lahir atau tanggal kejadian yang kosong menghasilkan sel kosong, bukan 0.
// Angka nol di kolom usia terbaca sebagai bayi, bukan sebagai "tidak diketahui".
func usiaPadaTanggal(lahir, kejadian time.Time) string {
	if lahir.IsZero() || kejadian.IsZero() {
		return ""
	}

	lahirWIB := clock.DateWIB(lahir)
	kejadianWIB := clock.DateWIB(kejadian)
	if kejadianWIB.Before(lahirWIB) {
		return ""
	}

	usia := kejadianWIB.Year() - lahirWIB.Year()
	// Ulang tahun pada tahun kejadian belum lewat bila bulannya lebih besar, atau
	// bulannya sama tetapi tanggalnya lebih besar.
	if kejadianWIB.Month() < lahirWIB.Month() ||
		(kejadianWIB.Month() == lahirWIB.Month() && kejadianWIB.Day() < lahirWIB.Day()) {
		usia--
	}
	return strconv.Itoa(usia)
}

// ---------------------------------------------------------------------------
// REPORT TAT
// ---------------------------------------------------------------------------

// tatWorkingDayColumn memetakan empat kolom "Lama proses" ke pasangan tanggalnya.
//
// Keempatnya berisi `Param.CountBusiness` di sistem lama — hasil
// `GCNMTimeDifferenceWorkCalender_Act` atas dua tanggal yang berbeda-beda. Pasangannya
// dibaca dari JUDUL kolomnya, yang menyebutkan keduanya apa adanya.
var tatWorkingDayColumn = []struct {
	// field adalah nama properti kolomnya pada berkas CSV.
	field string

	// judul adalah judul kolom di berkas, disebut di sini supaya pasangan tanggalnya
	// dapat diperiksa terhadap berkas keluaran tanpa membuka katalog.
	judul string

	// dari dan sampai adalah nama properti kolom tanggal pada baris yang sama.
	dari   string
	sampai string

	// nolkanBilaTakDiketahui menirukan `@If(Param.CountBusiness=="-1","0",…)`.
	//
	// Tiga dari empat kolom memakainya; satu tidak, dan perbedaan itu ada di export —
	// bukan kelalaian. Lihat keterangan pada tatDerive.
	nolkanBilaTakDiketahui bool
}{
	{
		field: "CloseClaimNote", judul: "Lama proses regis-tf ke teknik",
		dari: "City", sampai: "NewNoKTP", nolkanBilaTakDiketahui: true,
	},
	{
		field: "CityID", judul: "Lama regis - tanggal trf kasir",
		dari: "City", sampai: "NewEmail", nolkanBilaTakDiketahui: false,
	},
	{
		field: "OccupationCode", judul: "Lama aksep - trf kasir",
		dari: "CountryID", sampai: "NewEmail", nolkanBilaTakDiketahui: true,
	},
	{
		field: "Conveyance", judul: "Lama akseptasi - tanggal bayar",
		dari: "CountryID", sampai: "TanggalBayar", nolkanBilaTakDiketahui: true,
	},
}

// tatHelperColumn adalah kolom bantu yang dipakai berhitung dan TIDAK ikut ke berkas.
var tatHelperColumn = []string{
	"TanggalTerimaDokumenTeks",
	"TanggalLaporanTeks",
	"TanggalBayar",
}

// tatDerive menyusun penghitung kolom turunan laporan TAT.
//
// Ia membaca kalender libur SEKALI untuk seluruh rentang laporan, lalu memakainya pada
// setiap baris. Kalender yang tidak tersedia — karena portalnya belum punya koneksi kedua,
// atau karena pembacaannya gagal — membuat keempat kolom "Lama proses" menjadi sel
// kosong, bukan angka yang dihitung tanpa hari libur.
//
// Perbedaan itu penting: angka yang dihitung tanpa hari libur akan LEBIH BESAR dari yang
// sebenarnya, dan tidak ada apa pun di berkas yang menandakannya.
func tatDerive(ctx context.Context, r *Repo, f reportklaim.Filter) (func(reportklaim.Row), error) {
	calendar := r.holidayCalendar(ctx, f.From, f.To)

	return func(row reportklaim.Row) {
		row["CommentKomiteClosecase"] = tanggalDariTeksPadat(row.Value("TanggalTerimaDokumenTeks"))
		row["CASEDB"] = tanggalDariTeksPadat(row.Value("TanggalLaporanTeks"))
		row["USIA"] = usiaPadaTanggal(
			bacaTanggalCSV(row.Value("DOB")),
			bacaTanggalCSV(row.Value("DaftarObjek")),
		)

		for _, kolom := range tatWorkingDayColumn {
			hari := reportklaim.WorkingDaysBetween(
				bacaTanggalCSV(row.Value(kolom.dari)),
				bacaTanggalCSV(row.Value(kolom.sampai)),
				calendar,
			)
			switch {
			case hari != reportklaim.WorkingDaysUnknown:
				row[kolom.field] = strconv.Itoa(hari)
			case !calendar.Available():
				// Kalender tidak terbaca: sel dikosongkan. Menulis "0" di sini berarti
				// menyatakan tidak ada jeda sama sekali.
				row[kolom.field] = ""
			case kolom.nolkanBilaTakDiketahui:
				row[kolom.field] = "0"
			default:
				row[kolom.field] = strconv.Itoa(reportklaim.WorkingDaysUnknown)
			}
		}

		for _, bantu := range tatHelperColumn {
			delete(row, bantu)
		}
	}, nil
}

// ---------------------------------------------------------------------------
// Bulan Close / Bulan Reject
// ---------------------------------------------------------------------------

// bulanCloseDerive mengisi kolom singkatan bulan dari tanggal tutup klaim.
//
// Dipakai laporan Reject Klaim, Close Klaim, dan Temporary Close Klaim — ketiganya
// memakai properti `FlagASO` dengan judul "Bulan Reject" atau "Bulan Close".
func bulanCloseDerive(row reportklaim.Row) {
	row["FlagASO"] = bulanSingkat(bacaTanggalCSV(row.Value("BulanCloseTanggal")))
	delete(row, "BulanCloseTanggal")
}

// rejectDerive menambahkan pemformatan desimal koma pada tiga kolom nilai.
//
// Ketiganya dibungkus `@replaceAll(x,".",",")` di export: Estimasi Value, Persen OR, dan
// Fac_Out.
func rejectDerive(row reportklaim.Row) {
	bulanCloseDerive(row)
	for _, field := range []string{"Resources", "OwnRisk", "AlasanDokterRejectRCL"} {
		row[field] = desimalKoma(row.Value(field))
	}
}

// ---------------------------------------------------------------------------
// REPORT PRODUKSI KLAIM PA
// ---------------------------------------------------------------------------

// periodeProduksiPA adalah kelima nama properti kolom "Tahun" pada laporan Produksi
// Klaim PA.
//
// Kelimanya berisi NILAI YANG SAMA — satu per kelompok risiko. Itu memang bentuk berkas
// lama: setiap kelompok membawa kolom periodenya sendiri.
var periodeProduksiPA = []string{
	"CityID",         // Resiko A
	"CaseID",         // Resiko B
	"City",           // Resiko D
	"AnaylstRemarks", // Resiko MC
	"ReporterName",   // Resiko LAINNYA
}

// derivePeriodeProduksiPA menyusun periode berbentuk "2026-09" dari tahun dan bulan.
//
// Bentuknya disalin dari `TO_CHAR(A.CREATEDATETIME,'YYYY-MM')` pada kueri asli: empat
// digit tahun, tanda hubung, dua digit bulan dengan nol di depan.
func derivePeriodeProduksiPA(row reportklaim.Row) {
	tahun := row.Value("TahunPeriode")
	bulan := row.Value("BulanPeriode")

	periode := ""
	if tahun != "" && bulan != "" {
		if n, err := strconv.Atoi(bulan); err == nil {
			periode = fmt.Sprintf("%s-%02d", tahun, n)
		}
	}
	for _, name := range periodeProduksiPA {
		row[name] = periode
	}

	delete(row, "TahunPeriode")
	delete(row, "BulanPeriode")
}

// ---------------------------------------------------------------------------
// Susunan RINCI Non-MBU — Close Klaim, Temporary Close Klaim, Data Komite
// ---------------------------------------------------------------------------

// bulanDuaDigit menirukan `TO_CHAR(tanggal,'mm')`: dua angka dengan nol di depan.
func bulanDuaDigit(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("01")
}

// tahunEmpatDigit menirukan `TO_CHAR(tanggal,'yyyy')` dan `TO_CHAR(tanggal,'rrrr')`.
//
// Keduanya menghasilkan empat angka tahun yang sama saat MENULIS; `RR` hanya berbeda
// perilakunya saat MEMBACA teks menjadi tanggal.
func tahunEmpatDigit(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006")
}

// tanggalDuaDigitTahun menirukan `TO_CHAR(tanggal,'DD/MM/YY')`.
//
// Hanya satu kolom yang memakainya — "Tgl Terakhir Update Progress" — dan bentuknya
// ditiru apa adanya meski berbeda dari seluruh kolom tanggal lain di berkas yang sama.
// Menyeragamkannya berarti mengubah kolom yang selama ini dibaca berkas kerja penggunanya.
func tanggalDuaDigitTahun(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("02/01/06")
}

// selisihHariKalender menghitung selisih dua tanggal dalam hari penuh.
//
// Ia BUKAN hari kerja. Sumbernya `TO_DATE(TO_CHAR(tgl_proses,'dd/mm/yyyy'),'dd/mm/yyyy')
// - TO_DATE(TO_CHAR(tgl_aksep,…),…)` — pengurangan dua DATE Oracle yang sudah dipangkas
// ke tanggal, tanpa menyentuh kalender libur sama sekali.
//
// Salah satu tanggal yang kosong menghasilkan sel kosong, bukan 0.
func selisihHariKalender(dari, sampai time.Time) string {
	if dari.IsZero() || sampai.IsZero() {
		return ""
	}
	hari := int(clock.DateWIB(sampai).Sub(clock.DateWIB(dari)).Hours() / 24)
	return strconv.Itoa(hari)
}

// angka menulis bilangan pecahan sama seperti fungsi text menulis nilai dari basis data.
//
// Keduanya harus sama: satu kolom yang dihitung di Go dan satu kolom yang datang dari
// basis data berdampingan di berkas yang sama, dan bentuk angka yang berbeda di antara
// keduanya akan terbaca sebagai dua satuan yang berbeda.
func angka(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// bacaAngka mengurai sel angka kembali menjadi bilangan.
//
// Sel kosong menghasilkan (0, false) — dibedakan dari nol yang sungguh-sungguh, karena
// keduanya berbeda arti pada kolom nilai uang.
func bacaAngka(nilai string) (float64, bool) {
	nilai = strings.TrimSpace(nilai)
	if nilai == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(nilai, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// closeNonMBUHelperColumn adalah kolom bantu susunan rinci yang tidak ikut ke berkas.
var closeNonMBUHelperColumn = []string{
	"FeeDasarTotalClaim",
	"FeeLangsung",
	"FeeJumlahBaris",
	"FeeJumlahBarisInterpolasi",
	"ProgresJSON",
	"TglUpdateProgres",
	"TanggalRegistrasiTeks",
	"KunciKlaimDominan",
}

// closeNonMBUSalinanNilaiOR adalah tiga kolom yang isinya SAMA dengan `ReportDescription`.
//
// Keempatnya — "Nilai Klaim yang Diajukan Tertanggung OR", "Adjustment Klaim (Self
// Adjustment) OR", "Adjustment Klaim (Adjuster) OR", dan "Selisih … OR" — di sumbernya
// adalah empat ekspresi yang ditulis ULANG kata per kata dengan isi yang persis sama.
//
// Judulnya menjanjikan empat angka berbeda; yang keluar satu angka empat kali. Itu cacat
// sistem lama, dan ia DITIRU: memperbaikinya menuntut mengetahui angka apa yang
// seharusnya, dan tidak ada apa pun di export yang menyebutkannya.
//
// Yang tidak ditiru hanyalah cara menghitungnya — satu anak-kueri, bukan empat yang sama.
var closeNonMBUSalinanNilaiOR = []string{"StatusClaim", "TelpTertanggung", "UserAdmin"}

// closeDerive menyusun penghitung kolom turunan panel Close dan Temporary Close.
//
// Ia bercabang pada lini bisnis, karena kedua susunannya tidak berbagi satu pun kolom
// turunan: susunan ringkas hanya punya singkatan bulan, susunan rinci punya sebelas.
func closeDerive(ctx context.Context, r *Repo, f reportklaim.Filter) (func(reportklaim.Row), error) {
	if f.BusinessLine != reportklaim.BusinessLineNonMBU {
		return bulanCloseDerive, nil
	}

	// Kedua master dibaca SEKALI untuk seluruh laporan. Keduanya kecil — 17 pita fee dan
	// beberapa puluh tahapan progres — dan dipakai sekali per baris.
	fee := r.feeScale(ctx)
	progress := r.progressNames(ctx)
	dominan := r.dominantFactors(ctx, f.From, f.To)

	return func(row reportklaim.Row) {
		dol := bacaTanggalCSV(row.Value("StatusWork"))
		row["ClaimNo"] = bulanDuaDigit(dol)
		row["CloseClaimNote"] = tahunEmpatDigit(dol)

		row["RefNo"] = tahunEmpatDigit(bacaTanggalCSV(row.Value("UserTeknisGroup")))

		aksep := bacaTanggalCSV(row.Value("DollarCurrencyVal"))
		row["CompliancePosAuditByr"] = bulanDuaDigit(aksep)
		row["ComplianceRemark"] = tahunEmpatDigit(aksep)

		reject := bacaTanggalCSV(row.Value("UserTeknis"))
		row["Conveyance"] = bulanDuaDigit(reject)
		row["Country"] = tahunEmpatDigit(reject)

		// Kolom "TAT": selisih tanggal proses dengan tanggal akseptasi, dalam hari penuh.
		row["UserName"] = selisihHariKalender(aksep, bacaTanggalCSV(row.Value("UserTeknisEmail")))

		row["ReporterName"] = feeAdjuster(row, fee)

		nilaiOR := row.Value("ReportDescription")
		for _, name := range closeNonMBUSalinanNilaiOR {
			row[name] = nilaiOR
		}

		row["AlasanTerlambat"] = progress.Positions(row.Value("ProgresJSON"))
		row["ReceiverClaim"] = tanggalDuaDigitTahun(bacaTanggalCSV(row.Value("TglUpdateProgres")))
		row["NewNoKTP"] = tanggalDariTeksPadat(row.Value("TanggalRegistrasiTeks"))
		row["DominanName"] = dominan.Names(row.Value("KunciKlaimDominan"))

		for _, bantu := range closeNonMBUHelperColumn {
			delete(row, bantu)
		}
	}, nil
}

// feeAdjuster menghitung kolom "Adjuster Fee".
//
// # Rumusnya, dibaca dari kueri asli
//
// Sumbernya menjumlahkan satu CASE atas setiap baris settlement milik nomor akseptasi:
//
//	baris bertipe Adjuster Fee ('4')  → grossvalue × currencyvalue
//	baris lainnya                     → Get_InterpolasiPNC(jumlah TOTAL_CLAIM akseptasi itu)
//
// Perhatikan cabang kedua: yang diinterpolasi adalah JUMLAH seluruh akseptasi, bukan nilai
// baris itu — sehingga angka yang sama ikut terjumlah SEKALI PER BARIS bukan-'4'. Itu
// ditiru apa adanya; mengubahnya berarti mengubah nilai uang yang dilaporkan.
//
// # Yang dikembalikan saat tidak dapat dihitung
//
// Sel kosong, pada dua keadaan: tidak ada satu pun baris settlement untuk nomor akseptasi
// itu (sumbernya menghasilkan NULL), dan tangga fee yang tidak dapat menjawab. Yang kedua
// disengaja — menuliskan hanya bagian yang bertipe '4' akan menghasilkan angka yang lebih
// kecil dari seharusnya tanpa satu pun tanda.
func feeAdjuster(row reportklaim.Row, fee *reportklaim.FeeScale) string {
	jumlahBaris, ada := bacaAngka(row.Value("FeeJumlahBaris"))
	if !ada || jumlahBaris == 0 {
		return ""
	}

	total, _ := bacaAngka(row.Value("FeeLangsung"))

	interpolasi, _ := bacaAngka(row.Value("FeeJumlahBarisInterpolasi"))
	if interpolasi == 0 {
		return angka(total)
	}

	dasar, adaDasar := bacaAngka(row.Value("FeeDasarTotalClaim"))
	if !adaDasar {
		return ""
	}
	satuan, dapat := fee.Fee(dasar)
	if !dapat {
		return ""
	}
	return angka(total + interpolasi*satuan)
}

// komiteNonMBUHelperColumn adalah kolom bantu susunan rinci Data Komite.
var komiteNonMBUHelperColumn = []string{
	"ProgresJSON",
	"TanggalCloseUntukTAT",
	"TanggalTerimaLOD",
	"TanggalTransferKasir",
}

// komiteDerive menyusun penghitung kolom turunan panel Data Komite.
//
// Susunan ringkasnya tidak punya kolom turunan sama sekali, sehingga cabang itu
// mengembalikan nil — dan Stream memakai barisnya apa adanya.
func komiteDerive(ctx context.Context, r *Repo, f reportklaim.Filter) (func(reportklaim.Row), error) {
	if f.BusinessLine != reportklaim.BusinessLineNonMBU {
		// Susunan ringkas memakai kueri yang SAMA dengan panel Close Klaim, dan karena itu
		// kolom singkatan bulannya pun sama. Sebelumnya cabang ini tidak punya penghitung
		// sama sekali, sehingga kolom "Bulan Close" kosong — ditemukan
		// TestSetiapKolomBerkasPunyaSumber, bukan oleh pembacaan ulang.
		return bulanCloseDerive, nil
	}

	calendar := r.holidayCalendar(ctx, f.From, f.To)
	progress := r.progressNames(ctx)

	return func(row reportklaim.Row) {
		dol := bacaTanggalCSV(row.Value("CountryID"))
		row["AreaClaimId"] = bulanDuaDigit(dol)
		row["ContractNo"] = tahunEmpatDigit(dol)

		row["Remark"] = tahunEmpatDigit(bacaTanggalCSV(row.Value("RCV_ID")))

		// "BULAN CLOSE" dan "TAHUN CLOSE" TIDAK diisi. Alias sumbernya salah ketik
		// (`month CopyFrom`, `year CreateFrom`), sehingga keduanya tidak pernah terisi di
		// berkas lama — dan itu dibiarkan (keputusan Work Owner 2026-09-25).

		row["ExGratiaNote"] = hariKerjaSel(
			row.Value("Country"), row.Value("TanggalCloseUntukTAT"), calendar)
		row["FlagASO"] = hariKerjaSel(
			row.Value("TanggalTerimaLOD"), row.Value("TanggalTransferKasir"), calendar)

		row["AlasanTerlambat"] = progress.Positions(row.Value("ProgresJSON"))

		// Dua kolom yang memang tidak punya sumber — "OR ASM" dan "CLOSE CLAIM NOTE" —
		// TIDAK ditulis di sini. Ketiadaannya dinyatakan satu tempat saja, di
		// kolomTanpaSumber, supaya tidak ada dua daftar yang dapat berselisih.

		for _, bantu := range komiteNonMBUHelperColumn {
			delete(row, bantu)
		}
	}, nil
}

// hariKerjaSel menghitung jumlah hari kerja antara dua sel tanggal.
//
// Kalender yang tidak tersedia menghasilkan sel KOSONG, bukan angka yang dihitung tanpa
// hari libur — angka seperti itu selalu lebih besar dari yang sebenarnya, dan tidak ada
// apa pun di berkas yang menandakannya.
func hariKerjaSel(dari, sampai string, calendar *reportklaim.HolidayCalendar) string {
	hari := reportklaim.WorkingDaysBetween(
		bacaTanggalCSV(dari), bacaTanggalCSV(sampai), calendar)
	if hari == reportklaim.WorkingDaysUnknown {
		return ""
	}
	return strconv.Itoa(hari)
}

// ---------------------------------------------------------------------------
// Kolom yang di sistem lama pun SELALU kosong
// ---------------------------------------------------------------------------

// kolomTanpaSumber mendaftar kolom berkas yang TIDAK punya ekspresi apa pun di kueri
// aslinya, dan karena itu selalu kosong sejak dulu.
//
// # Kenapa didaftarkan, bukan dibiarkan
//
// Karena tiga hal yang berbeda terlihat sama di berkas keluaran: kolom yang datanya
// memang kosong, kolom yang aliasnya salah ketik, dan kolom yang tidak pernah punya
// sumber. Yang kedua adalah cacat yang harus diperbaiki; yang ketiga adalah keadaan yang
// harus diketahui pemilik laporan.
//
// Daftar ini memisahkan ketiganya. Ia dipakai uji TestSetiapKolomBerkasPunyaSumber sebagai
// satu-satunya pengecualian yang diterima, dan uji itu juga menolak nama yang ternyata
// SUDAH punya sumber — sehingga daftar ini tidak dapat menjadi catatan usang.
//
// # Bagaimana isinya ditetapkan
//
// Setiap nama di sini diperiksa terhadap rule SQL aslinya di export, dengan perbandingan
// yang MENGABAIKAN huruf besar-kecil — properti Pega tidak peka huruf, sehingga
// `RESPONSENOTE` dan `ResponseNote` sebenarnya cocok dan BUKAN kolom tanpa sumber.
//
// Yang tersisa di bawah adalah nama yang benar-benar tidak punya padanan apa pun.
//
// # Dua sebab yang berbeda, dan keduanya dibiarkan
//
// Dua puluh satu kolom pada delapan panel tidak pernah berisi data, karena dua sebab:
//
//	SALAH KETIK ALIAS  ekspresinya ADA, tetapi namanya meleset satu-dua huruf sehingga
//	                   tidak pernah bertemu properti berkasnya — `IsCFS` versus
//	                   `IsCFS_PNC`, `month CopyFrom` versus `CopyFrom`
//
//	TIDAK ADA SUMBER   tidak ada ekspresi apa pun yang dimaksudkan mengisinya —
//	                   "OR ASM", "Dokumen Penalti", "Dominan Factor"
//
// Kelompok pertama sempat diselaraskan supaya terisi. Penyelarasan itu DICABUT: Work Owner
// menetapkan 2026-09-25 bahwa cacat yang berasal dari Pega dibiarkan seperti Pega, dan yang
// diperbaiki hanya cacat pemindahan.
//
// Kelompok kedua memang tidak dapat diperbaiki dari export — menebak ekspresinya berarti
// mengarang angka.
//
// Yang berubah dari sistem lama karena itu bukan isinya, melainkan bahwa ketiadaannya kini
// TERCATAT dan terkunci uji. Sebelumnya tidak ada apa pun yang menyatakan kolom-kolom ini
// memang tidak akan pernah terisi. Ini diangkat ke Work Owner sebagai TEMUAN.
var kolomTanpaSumber = map[reportklaim.Code][]string{
	// "Remark" diisi `.Notes` di activity-nya, dan tidak ada satu pun langkah yang
	// mengisi `.Notes` — penugasan itu menyalin kekosongan.
	reportklaim.CodeTAT: {
		"Status", // "Remark"
	},
	reportklaim.CodeKlaimHarian: {
		"Location", // "Location"
	},
	// Alias sumbernya ` QQNAME` — dengan SPASI DI DEPAN.
	reportklaim.CodeRegistSimasOnline: {
		"QQNAME", // "Nama Tertanggung"
	},
	reportklaim.CodeAIKlaim: {
		"NoKTP",        // "Tgl Kejadian"
		"NamaSurveyor", // "Tipe Note AI"
	},
	// Alias sumbernya `IsCFS_PNC` sementara propertinya `IsCFS` — arah yang TERBALIK dari
	// Close Klaim di bawah. Keduanya cacat Pega, keduanya dibiarkan.
	reportklaim.CodeRejectKlaim: {
		"IsCFS", // "ER2"
	},
	// "Dokumen Penalti" diisi `.ReporterTelp` — penugasan ke dirinya sendiri dari nama
	// yang tidak pernah dihasilkan kueri, sehingga menyalin kekosongan. Sama dengan
	// "ER2" di bawahnya.
	reportklaim.CodeCloseKlaim: {
		"IsCFS_PNC",    // "ER2" — alias sumbernya `IsCFS`
		"ReporterTelp", // "Dokumen Penalti"
	},
	reportklaim.CodeTemporaryCloseKlaim: {
		"IsCFS_PNC",
		"ReporterTelp",
	},
	reportklaim.CodeKomite: {
		"CopyFrom",    // "BULAN CLOSE"  — alias sumbernya `month CopyFrom`
		"CreateFrom",  // "TAHUN CLOSE"  — alias sumbernya `year CreateFrom`
		"Note Komite", // "KomiteAccepted" — alias sumbernya `KomiteAccepted`
		"Location",    // "OR ASM"       — tidak ada ekspresi apa pun
		"NoteKasir",   // "CLOSE CLAIM NOTE" — tidak ada ekspresi apa pun
	},
	reportklaim.CodeAkseptasi: {
		"Location",     // "OR"
		"ReporterName", // "SURPLUS"
		"RefNo",        // "OTHER"
	},
	reportklaim.CodeOSKomite: {
		"Location",
		"ReporterName",
		"RefNo",
		"DominanName", // "DOMINAN FACTOR"
	},
	reportklaim.CodeOSBelumKomite: {
		"Location",
		"ReporterName",
		"RefNo",
	},
}

// osKomiteDerive mengisi kolom UW YEAR dari tanggal mulai periode polis.
//
// Kolomnya di sumbernya `to_char(c.begindate,'rrrr')`; tanggalnya sendiri sudah ada di
// baris yang sama sebagai kolom START DATE, sehingga tahunnya dihitung dari situ alih-alih
// mengambil kolom yang sama dua kali.
func osKomiteDerive(row reportklaim.Row) {
	row["Remark"] = tahunEmpatDigit(bacaTanggalCSV(row.Value("RCV_ID")))
}
