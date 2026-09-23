package inboxlaporanklaim

import (
	"strings"
	"time"
)

// ReportNumberPrefix menandai laporan yang diterbitkan aplikasi ini.
//
// Bentuknya `RCVN.YY.xxxx`, mengikuti keputusan `D-71` untuk nomor klaim
// (`PNCN.YY.xxxx`) beserta alasannya: selama masa paralel, asal sebuah baris harus
// terbaca langsung dari nomornya tanpa tabel pemetaan. Laporan warisan Pega memakai
// `pyID` berbentuk lain dan tidak pernah diterbitkan ulang.
const ReportNumberPrefix = "RCVN"

// Money adalah nilai rupiah dalam SEN. Rp 1.000 = 100_000.
//
// # Kenapa bilangan bulat, dan kenapa tidak mengimpornya dari modul lain
//
// `ADR-0016` menuntut nilai uang presisi penuh dan pembulatan hanya saat ditampilkan.
// Tipe pecahan biner tidak dapat memenuhi itu: 0,1 tidak punya wakil yang tepat.
//
// Modul `registrasi` mendeklarasikan tipe yang sama untuk alasan yang sama, dan tipe itu
// TIDAK diimpor ke sini: paket domain satu modul tidak boleh bergantung pada paket domain
// modul lain — ketergantungan itu membuat kedua modul tidak dapat dipindahkan
// sendiri-sendiri. Tipe uang bersama adalah milik `TKT-U2-004` yang belum ada; sampai ia
// ada, pengulangan satu baris ini lebih murah daripada kopling antarmodul.
type Money int64

// Rupiah membentuk Money dari jumlah rupiah bulat.
func Rupiah(n int64) Money { return Money(n * 100) }

// Origin menyatakan sistem mana yang menerbitkan sebuah baris.
//
// Ia BUKAN kolom di basis data mana pun; ia diturunkan dari tabel asal barisnya saat
// dibaca. Dibawa sampai ke layar dengan sengaja — petugas yang melihat selisih antara
// aplikasi ini dan Pega harus dapat mengetahui baris mana yang ditulis siapa, dan itulah
// pertanyaan pertama pada setiap rekonsiliasi harian masa paralel.
type Origin string

const (
	// OriginLegacy: berkas warisan, dikenali dari nomornya yang TIDAK berawalan RCVN.
	// Dibaca dari POOLDATA.T_CLAIMLIST_ADMIN, dan hanya dibaca.
	OriginLegacy Origin = "pega"

	// OriginNew: baris dibaca dari POOLDATA.CPNC_LAPORAN_KLAIM, tabel milik aplikasi ini.
	OriginNew Origin = "claimpnc"
)

// Position adalah keadaan sebuah laporan pada alurnya.
//
// Ketiga nilainya DISALIN APA ADANYA dari `RDB List/ViewAllCase-SQL.xml`, yang
// menurunkannya lewat CASE WHEN atas dua kolom saja — `pnccaseid` dan `statuslock_1`:
//
//	pnccaseid IS NOT NULL AND statuslock_1 IS NOT NULL  ->  Outstanding
//	statuslock_1 IS NOT NULL AND pnccaseid IS NULL      ->  Not Registered
//	pnccaseid IS NULL AND statuslock_1 IS NULL          ->  Not Transferred
//
// Teksnya tidak diterjemahkan: `D-13` menetapkan tampilan meniru Pega supaya pengguna
// tidak belajar ulang, dan inilah teks yang selama ini mereka baca di kolom "Position".
type Position string

const (
	// PositionOutstanding: laporan sudah menjadi klaim bernomor dan masih berjalan.
	PositionOutstanding Position = "Outstanding"

	// PositionNotRegistered: berkas sudah diserahkan ke klaim tetapi belum diregistrasi.
	PositionNotRegistered Position = "Not Registered"

	// PositionNotTransferred: laporan baru masuk, belum diserahkan ke petugas klaim.
	PositionNotTransferred Position = "Not Transferred"
)

// DerivePosition menurunkan Position dari dua penanda yang dipakai kueri lama.
//
// Ia dipakai KEDUA pengisi seam — sqlstore maupun memory — supaya keduanya tidak dapat
// berbeda tafsir. Kueri lama menghitungnya di SQL; di sini ia dihitung sekali di Go,
// sehingga aturannya dapat diuji tanpa basis data dan tidak tersebar di sembilan kueri.
func DerivePosition(hasClaimNumber, transferred bool) Position {
	switch {
	case hasClaimNumber && transferred:
		return PositionOutstanding
	case transferred:
		return PositionNotRegistered
	default:
		return PositionNotTransferred
	}
}

// ClaimReport adalah satu baris pada Inbox Laporan Klaim.
//
// # Alias menyesatkan yang TIDAK dibawa (D-19)
//
// Kesembilan kueri lama mengaliaskan kolomnya ke nama properti Pega yang sudah ada,
// bukan ke nama yang mencerminkan isinya — persis utang teknis §4.2
// `03-CURRENT-ARCHITECTURE.md`. Lima di antaranya berbahaya karena namanya menyebut hal
// yang sama sekali lain:
//
//	BUSINESSNAME    AS "Kurir"              -> BusinessName   (nama bisnis, BUKAN kurir)
//	BRANCHNAME      AS "UserAdmin"          -> BranchName     (nama cabang, BUKAN nama user)
//	pxcreateoperator AS "KodeCabang"        -> CreatedBy      (operator, BUKAN kode cabang)
//	kodecabang_1    AS "StatusKomunikasi"   -> BranchCode     (kode cabang, BUKAN status)
//	KETERANGAN_1    AS "SIM"                -> Reason         (alasan, BUKAN nomor SIM)
//
// Membawa alias itu berarti membawa kesalahpahamannya. Nama di sini menyebut apa yang
// benar-benar ada di kolomnya; pemetaan balik ke kolom aslinya ada di berkas `.sql`.
type ClaimReport struct {
	// ID adalah nomor register laporan — `pyid` pada kueri lama, dialiaskan "RCVID".
	// Ia kunci baris yang dilihat pengguna di kolom "Case ID".
	ID string

	// ClaimNumber adalah nomor klaim yang terbit dari laporan ini — `pnccaseid`,
	// kolom "Case PNC". KOSONG berarti laporan belum diregistrasi, dan kekosongan itu
	// bukan data yang hilang: ia salah satu dari dua penanda yang menentukan Position.
	ClaimNumber string

	// Transferred menandai berkas sudah diserahkan ke petugas klaim — `statuslock_1`
	// yang TIDAK NULL pada kueri lama.
	//
	// Kolomnya bernama "lock" tetapi tidak mengunci apa pun; yang diisi di sana adalah
	// `pzInsKey` penugasan. Yang dibawa ke sini hanyalah ADA atau TIDAKNYA nilai itu,
	// karena hanya itu yang dipakai kesembilan kueri untuk membedakan tab.
	Transferred bool

	// AssignmentRef adalah rujukan penugasan warisan, isi `statuslock_1` apa adanya.
	//
	// Ia dibawa hanya supaya layar dapat membuka berkasnya di Pega selama masa paralel —
	// `SetListRCV_Act` menyusunnya menjadi
	// "ASSIGN-WORKLIST " + trim(StatusLock) + "!ReceiveDocument_Flow". Kosong untuk
	// laporan yang diterbitkan aplikasi ini.
	AssignmentRef string

	PolicyNumber string // POLICYNO — kolom "Polis no"
	InsuredName  string // qqname — kolom "Insured Name"

	// BusinessName adalah nama lini bisnis polis — BUSINESSNAME, kolom "Business Name".
	BusinessName string

	// ReferenceNumber adalah nomor rujukan berkas masuk — `BookNo_1`, kolom
	// "Reference no", yang kueri lama aliaskan "Sender".
	ReferenceNumber string

	// ReporterName adalah nama pelapor — properti `Sender` pada berkas Pega, yang
	// `CreateNewCaseRCV` isi dengan nama petugas pembuatnya.
	//
	// Ia TIDAK digambar di sembilan grid mana pun, dan tetap dibawa: ia bagian dari isi
	// berkas, dan membuangnya berarti berkas yang dibuat aplikasi ini kehilangan sesuatu
	// yang berkas Pega punya.
	//
	// Kosong untuk SELURUH baris warisan, dan itu bukan data yang hilang: tidak satu pun
	// dari kesembilan kueri lama membaca kolomnya, sehingga nama kolomnya di tabel Pega
	// tidak diketahui (R-08). Menebaknya berarti kueri yang gagal saat dijalankan
	// pertama kali di produksi.
	ReporterName string

	// DateOfLoss adalah tanggal kejadian — `DATEOFLOSS_1`, kolom "Date of loss".
	// Nol berarti belum diisi; laporan yang baru masuk memang sering belum memilikinya.
	DateOfLoss time.Time

	// CreatedAt adalah saat laporan masuk — `pxcreatedatetime`, kolom "Input Date".
	CreatedAt time.Time

	// CreatedBy adalah operator yang memasukkan laporan — `pxcreateoperator`,
	// kolom "Creator".
	CreatedBy string

	// UpdatedBy dan UpdatedAt mencatat penyimpanan terakhir lewat form Input Receive
	// Document.
	//
	// Keduanya TIDAK ada di sistem lama — objek kerja Pega mencatatnya sendiri di tabel
	// engine yang tidak kita bawa. Ia BUKAN jejak audit: jejak audit adalah `S-5` yang
	// mencatat nilai sebelum dan sesudah (`D-28`), dan modul itu belum ada. Yang ini
	// hanya menjawab "siapa terakhir menyentuh berkas ini", dan itu pun sudah lebih
	// banyak daripada yang dapat dijawab hari ini.
	//
	// Kosong untuk baris warisan dan untuk berkas yang belum pernah disunting.
	UpdatedBy string
	UpdatedAt time.Time

	// BranchCode dan BranchName menyebut cabang klaim yang menangani — `kodecabang_1`
	// dan hasil lookup POOLDATA.BRANCH, kolom "Cabang Klaim".
	BranchCode string
	BranchName string

	// AgingAt adalah titik hitung umur berkas — `DateForAging_1`.
	// Ia juga kunci pengurutan seluruh kueri lama: ORDER BY DateForAging_1 DESC.
	//
	// Darinya dihitung kolom "Total Aging" pada grid. Ia BUKAN kolom "Aging" —
	// lihat AgingValue.
	AgingAt time.Time

	// AgingValue adalah isi kolom `AGING` pada POOLDATA.T_CLAIMLIST_ADMIN, digambar apa
	// adanya pada kolom "Aging" di grid.
	//
	// # Kenapa teks, bukan angka
	//
	// Kolomnya NUMBER dan nullable, dan layar membedakan "kosong" dari "nol". Angka
	// bertipe int tidak dapat membedakan keduanya — 0 akan tergambar sebagai "0" padahal
	// kolomnya memang belum diisi, dan pada layar contoh ia justru kosong di seluruh
	// baris. Teks membawa keduanya apa adanya tanpa satu pun tafsiran.
	//
	// # Kenapa ia DIPISAH dari AgingAt
	//
	// Layar lama menggambar "Aging" dan "Total Aging" BERDAMPINGAN sebagai dua kolom.
	// Yang kedua dihitung dari AgingAt oleh aplikasi; yang pertama dibaca dari kolom ini.
	// Menyatukannya akan menghapus satu kolom yang memang ada di layar.
	//
	// **Artinya belum diketahui.** Tidak ada satu pun rule di export yang menyentuh kolom
	// ini, dan pada layar contoh ia kosong di seluruh baris yang terlihat. Ia karena itu
	// diteruskan apa adanya, bukan ditafsirkan.
	AgingValue string

	// Reason adalah keterangan kenapa berkas belum berpindah — `KETERANGAN_1`,
	// kolom "Alasan" pada grid, dan isian "Keterangan Belum Transfer" pada form.
	//
	// Ketiganya satu kolom yang sama: form mengikatnya ke `.ReceiveDocument.Keterangan`,
	// dan procedure menyimpannya sebagai `ALASANBLMTRANSFER`.
	Reason string

	// # Isian form Input Receive Document
	//
	// Kesepuluh field berikut TIDAK digambar di sembilan grid mana pun; ia isi berkas
	// yang dikumpulkan form `InputReceiveDocument` (`Flow/InputReceiveDocument.xml`).
	//
	// Seluruhnya KOSONG untuk baris warisan, dan itu bukan data yang hilang: tidak satu
	// pun dari kesembilan kueri lama membaca kolomnya, sehingga nama kolomnya di tabel
	// Pega tidak diketahui (`R-08`). Menebaknya menghasilkan kueri yang gagal saat
	// pertama dijalankan di produksi — dan berkas warisan memang tidak dapat disunting
	// dari sini (lihat ErrReadOnlyOrigin).

	// ReceivedDate adalah Tanggal Terima Dokumen — `.ReceiveDocument.ReceivedDate`,
	// disimpan procedure sebagai `TANGGALTERIMADOKUMEN`.
	ReceivedDate time.Time

	ReporterEmail string // .ReceiveDocument.EmailPengirim  — "Email Pengirim"
	ReporterPhone string // .ReceiveDocument.TelpPengirim   — "No. HP Pengirim"
	CourierName   string // .ReceiveDocument.Kurir          — "Nama Kurir ASM"

	// EstimateValue adalah Estimasi Kerugian — `.ReceiveDocument.Estimasi`.
	//
	// Ia BUKAN nilai klaim: nilai klaim lahir di `B-5` setelah registrasi. Yang ada di
	// sini angka yang disebut pelapor saat berkasnya masuk, dan ia tidak pernah dipakai
	// menghitung apa pun di modul ini.
	EstimateValue Money

	LossLocation     string // .ReceiveDocument.LokasiKejadian     — "Lokasi Kejadian"
	Chronology       string // .ReceiveDocument.KronologisKejadian — "Kronologis Kejadian"
	DamageDetail     string // .ReceiveDocument.RincianKerusakan   — "Rincian Kerusakan"
	NotRegisteredNote string // .ReceiveDocument.NotRegistNote     — "Keterangan Belum Registrasi"

	// DocumentCount adalah Total Jumlah Dokumen — `.ReceiveDocument.NumberOfDocument`.
	//
	// Ia hanya ANGKA. Rincian dokumennya — nama, jenis, jumlah per baris, dan tautan
	// melihatnya — terikat page list `.ReceiveDocument.DocumentList` dan menuntut
	// penyimpanan dokumen (`S-1`) yang belum ada. Angkanya dibawa; daftarnya tidak.
	DocumentCount int

	// EmailSubject adalah judul surel laporan masuk — `SubjectEmail_1`,
	// kolom "Subject Email".
	EmailSubject string

	// LastMessage adalah pesan terakhir pada percakapan cabang–pusat, kolom
	// "Last message". Hanya terisi pada ketiga kategori komunikasi; kosong di tab lain,
	// dan kekosongan itu bukan data yang hilang.
	LastMessage string

	// Position diturunkan DerivePosition, bukan disimpan. Lihat Position.
	Position Position

	// Origin menyebut tabel asal baris ini. Lihat Origin.
	Origin Origin
}

// AgingDays menghitung umur berkas dalam hari penuh terhadap waktu acuan.
//
// Kolom "Total Aging" di sistem lama menghitungnya di sisi tampilan atas `DateForAging`.
// Di sini ia dihitung di Go, sejalan dengan aturan Steering bahwa pemformatan dan
// perhitungan tampilan dikeluarkan dari SQL (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// Acuannya diminta sebagai parameter, bukan diambil dari jam sistem di dalam fungsi ini:
// nilai yang bergantung pada "sekarang" tidak dapat diuji secara deterministik, dan itu
// persis kelas cacat yang seam Clock (`F-5`) ada untuk mencegahnya.
//
// Berkas yang tanggal acuannya belum terisi mengembalikan 0 — bukan umur raksasa yang
// terhitung dari tahun nol.
func (r ClaimReport) AgingDays(now time.Time) int {
	if r.AgingAt.IsZero() {
		return 0
	}
	// Selisihnya dihitung atas TANGGAL, bukan atas timestamp: berkas yang masuk kemarin
	// sore berumur satu hari, bukan nol hari karena belum genap 24 jam.
	from := time.Date(r.AgingAt.Year(), r.AgingAt.Month(), r.AgingAt.Day(), 0, 0, 0, 0, time.UTC)
	to := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	days := int(to.Sub(from).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// Registered menyatakan laporan sudah menjadi klaim bernomor.
func (r ClaimReport) Registered() bool { return strings.TrimSpace(r.ClaimNumber) != "" }

// LegacyAssignmentKey menyusun rujukan penugasan warisan yang dipakai layar Pega.
//
// Bentuknya disalin dari `Activity/SetListRCV_Act-Act.xml`:
//
//	.Keterangan := "ASSIGN-WORKLIST " + @trim(.StatusLock) + "!ReceiveDocument_Flow"
//
// Kosong bila barisnya bukan milik Pega atau belum diserahkan — memanggil Pega dengan
// kunci separuh jadi akan membuka layar galat, bukan berkasnya.
func (r ClaimReport) LegacyAssignmentKey() string {
	reference := strings.TrimSpace(r.AssignmentRef)
	if reference == "" || r.Origin != OriginLegacy {
		return ""
	}
	return "ASSIGN-WORKLIST " + reference + "!ReceiveDocument_Flow"
}
