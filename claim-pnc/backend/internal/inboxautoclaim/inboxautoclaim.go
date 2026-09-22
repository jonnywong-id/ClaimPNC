// Package inboxautoclaim adalah inti modul Inbox Auto Claim.
//
// # Apa yang dimodelkan di sini
//
// Perusahaan rekanan — bank dan lembaga pembiayaan — mengirimkan klaimnya secara
// BORONGAN, bukan satu per satu lewat layar Register. Berkas berisi banyak baris klaim
// diunggah sekaligus sebagai satu **Batch**, lalu sistem memprosesnya menjadi case klaim
// satu per satu. Layar ini tempat petugas melihat setiap batch beserta berapa baris yang
// berhasil dan berapa yang gagal.
//
//	Satu baris di layar = satu pasangan (Kode Perusahaan × Nomor Batch)
//
// Di sistem lama ia Harness/InboxAutoClaim-Harness.xml, dan datanya tinggal di
// POOLDATA.TMP_BATCH_AUTO_CLAIM yang dijoin ke POOLDATA.M_AUTO_CLAIM_PNC.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxAutoClaim-Harness.xml             layar "Inbox Auto claim"
//	Section/Inbox_AS_KREDIT_Sect-Section.xml       judul, grid 8 kolom, 7 tombol
//	Section/ButtonPagingInbox-Section.xml          First/Previous/Next/Last + Total Data
//	Activity/INBOX_AS_KREDIT_ACT_AUTOCLAIM-Act.xml pemuat daftar batch + penyaringnya
//	Activity/DETAIL_AUTO_CLAIM-Act.xml             rincian satu batch, PageSize = 15
//	Activity/REPORT_AUTO_CLAIM_ACT-Act.xml         kedua CSV beserta judul kolomnya
//	Activity/CreateCasePNCAgent_AutoClaim-Act.xml  alur "Proses Klaim"
//	Activity/CreateCasePNC_AutoClaim-Act.xml       pembuatan case per baris
//	RDB List/GroupingAutoClaim-SQL.xml             kolom TMP_BATCH_AUTO_CLAIM
//	RDB List/GroupingAutoClaim2-SQL.xml            penanda batch yang belum diproses
//	RDB List/InsertUpdateAutoClaim-SQL.xml         parameter POOLDATA.INSERT_AUTOCLAIM
//	RDB List/BrowseAutoKlaim-SQL.xml               kolom M_AUTO_CLAIM_PNC
//	RDB List/GetReceiverClaimAsuransiKredit-SQL.xml syarat perusahaan boleh diklaim
//
// # ENAM KUERI LAYAR INI TIDAK ADA DI EXPORT
//
// BrowseClaimSPKAutoClaim, BrowseClaimSPK1_AutoClaim, BrowseClaimSPK_COUNT_AutoClaim,
// BrowseAutoClaim_COUNT, BrowseClaimSPK_detail_AutoClaim, dan
// BrowseReportClaimSPK_AutoClaim dirujuk keempat activity di atas tetapi TIDAK ADA di
// seluruh 2.634 berkas export. Ini R-16.
//
// Kuerinya karena itu DIREKONSTRUKSI dari tabel dan dari activity yang memanggilnya —
// bukan disalin. Setiap kolom yang dipakai punya bukti berkas:baris-nya sendiri di
// repo/sqlstore/inboxautoclaim.sql. Konsekuensinya dicatat tegas: sampai keenam kueri
// aslinya tiba dari Tim Pega, modul ini TIDAK DAPAT dinyatakan lulus gerbang 1, karena
// tidak ada yang dapat dibandingkan (D-42).
//
// # Penamaan ulang yang disengaja (D-19)
//
// Grid lama mengikat kolomnya ke property yang namanya tidak ada hubungannya dengan
// isinya — persis utang teknis §4.2 03-CURRENT-ARCHITECTURE.md. Alias itu TIDAK dibawa:
//
//	.CaseID                  -> CompanyCode  (bukan nomor kasus sama sekali)
//	.AlasanTerlambat         -> CompanyName  (bukan alasan keterlambatan)
//	.CauseOfLoss             -> BatchNumber  (bukan penyebab kerugian)
//	.ChronologicalOfIncodent -> Uploaded     (bukan kronologi kejadian)
//	.City                    -> Processed    (bukan nama kota)
//	.CityID                  -> Succeeded    (bukan kode kota)
//	.ClaimID                 -> Failed       (bukan nomor klaim)
//	.AnaylstRemarks          -> UploadedBy   (bukan catatan analis; salah ketiknya pun ikut)
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package inboxautoclaim

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// MessageSuccess adalah isi kolom TMP_MESSAGE untuk baris yang berhasil menjadi klaim.
//
// # Kenapa nilai ini boleh berada di dalam kode, padahal D-15 melarang hardcode
//
// D-15 melarang NILAI KEBIJAKAN BISNIS di-hardcode — ambang uang, penerima notifikasi,
// pemetaan peran — karena semuanya berubah tanpa merilis aplikasi. Nilai ini bukan
// kebijakan: ia ISI PROTOKOL tabel warisan. Yang menuliskannya adalah
// POOLDATA.INSERT_AUTOCLAIM, dan yang membacanya di sistem lama menuliskannya apa adanya:
//
//	Activity/REPORT_AUTO_CLAIM_ACT-Act.xml   "AND TMP_MESSAGE='Sukses Klaim'"
//	                                         "AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null"
//	RDB List/GetHasilAutoClaim-SQL.xml       tmp_message='Sukses Klaim'
//	RDB List/GetHasilAutoClaimGagal-SQL.xml  tmp_message !='Sukses Klaim'
//
// Menjadikannya konfigurasi justru berbahaya: mengubahnya di sini tidak akan mengubah
// apa yang ditulis procedure, sehingga layar dan data berselisih diam-diam.
//
// Ia tinggal di SATU tempat ini supaya keempat kueri yang memakainya tidak dapat berbeda
// satu sama lain — persis cacat "satu ambang, tiga operator" pada D-49 #2.
const MessageSuccess = "Sukses Klaim"

// DefaultPageSize adalah jumlah baris per halaman.
//
// Angka 15 dibaca dari Activity/DETAIL_AUTO_CLAIM-Act.xml yang menetapkan
// `.PageSize = 15`. Daftar batch memakai angka yang sama: activity pemuatnya
// (INBOX_AS_KREDIT_ACT_AUTOCLAIM) memakai mekanisme paginasi yang sama
// (`.LastRow = (.CurrentIndex * .PageSize)`) tetapi nilainya tidak ikut terekam di
// export, dan dua angka berbeda untuk dua daftar pada satu layar hanya membingungkan.
const DefaultPageSize = 15

// MaxPageSize membatasi ukuran halaman yang boleh diminta klien.
//
// 10-API-STRATEGY.md §4: permintaan yang lebih besar DITOLAK, bukan dipenuhi diam-diam —
// halaman raksasa adalah cara termurah membuat satu permintaan menghabiskan memori.
const MaxPageSize = 100

// Batch adalah satu baris grid inbox: satu unggahan milik satu perusahaan.
type Batch struct {
	// CompanyCode adalah kolom INISIALID — kode singkat perusahaan rekanan.
	CompanyCode string

	// CompanyName adalah M_AUTO_CLAIM_PNC.NAMA_PENERIMA.
	//
	// Dapat KOSONG. Join-nya sengaja LEFT, bukan INNER: batch yang kode perusahaannya
	// tidak ada di master tetap harus terlihat petugas. Menyembunyikannya membuat baris
	// yang tidak akan pernah berhasil diproses menghilang tanpa seorang pun tahu —
	// kegagalan senyap yang justru paling mahal.
	CompanyName string

	// BatchNumber adalah kolom BATCH — nomor unggahan, unik di dalam satu perusahaan.
	BatchNumber string

	// ProcessedDate adalah TANGGAL kolom TGLPROSES, tanpa jamnya.
	//
	// Ia ikut menjadi kunci baris grid, bukan sekadar keterangan: kueri asli
	// (`BrowseClaimSPKAutoClaim`) mengelompokkan dengan
	// `to_date(to_char(a.TGLPROSES,'dd/mm/yyyy'),'dd/mm/yyyy')`, sehingga satu batch yang
	// barisnya masuk pada dua tanggal berbeda tampil sebagai DUA baris.
	//
	// Versi pertama modul ini meringkasnya menjadi satu baris per (perusahaan, batch) —
	// itu MENYEMBUNYIKAN pemrosesan bertahap, dan diperbaiki setelah kueri aslinya tiba.
	ProcessedDate string

	// Uploaded adalah jumlah seluruh baris pada batch ini.
	//
	// Perhatikan: hitungannya menghitung SELURUH baris (perusahaan, batch) — tidak
	// disaring tanggal proses, persis seperti subkueri pada kueri asli. Jadi bila satu
	// batch tampil sebagai dua baris tanggal, keempat angkanya SAMA pada kedua baris.
	// Itu perilaku aslinya, bukan cacat perhitungan.
	Uploaded int

	// Processed adalah baris yang sudah selesai diproses, berhasil maupun gagal.
	// Dasarnya TMP_MESSAGE sudah terisi.
	Processed int

	// Succeeded adalah baris ber-TMP_MESSAGE = MessageSuccess.
	Succeeded int

	// Failed adalah baris yang TMP_MESSAGE-nya terisi tetapi bukan MessageSuccess.
	Failed int

	// UploadedBy adalah kolom USERINPUT — pengguna yang mengunggah batch ini.
	UploadedBy string
}

// Pending adalah baris yang belum tersentuh proses sama sekali.
//
// Ia DIHITUNG, bukan disimpan: menyimpannya berarti ada dua sumber untuk satu angka, dan
// keduanya dapat berselisih. Kolomnya pun tidak ada di layar lama — yang ada hanya
// "di upload" dan "telah diproses" — tetapi selisih keduanya itulah yang sebenarnya
// dilihat petugas saat memutuskan menekan Proses Klaim.
func (b Batch) Pending() int {
	pending := b.Uploaded - b.Processed
	if pending < 0 {
		// Tidak mungkin secara aturan, tetapi mungkin secara data: kedua angka datang
		// dari satu kueri agregat atas tabel yang ditulis sistem lain. Mengembalikan
		// angka negatif ke layar hanya memindahkan kebingungan ke pengguna.
		return 0
	}
	return pending
}

// Line adalah satu baris klaim di dalam sebuah batch — isi layar DETAIL.
//
// # Kenapa seluruh nilainya bertipe teks
//
// Bukan kemalasan. Kolom tanggal pada TMP_BATCH_AUTO_CLAIM menyimpan TEKS berformat
// dd/mm/yyyy, dan itu terbaca dari Activity/CreateCasePNC_AutoClaim-Act.xml yang
// menyusun ulang timestamp-nya dengan pemotongan karakter:
//
//	Local.dol = @substring(.DateOfLoss,6,10)+@substring(.DateOfLoss,3,5)+@substring(.DateOfLoss,0,2)+"T000000.000 GMT"
//
// Membacanya sebagai tanggal lalu menuliskannya kembali akan MENGUBAH isinya — dan pada
// tabel yang masih ditulis serta dibaca Pega, itu melanggar P-5. Ia juga mengaktifkan
// kembali seluruh kelas cacat zona waktu yang F-5 justru tutup.
//
// ClaimAmount bertipe teks karena alasan berbeda dan sama kerasnya: I-12 menuntut nilai
// uang disimpan PRESISI PENUH dan hanya dibulatkan saat ditampilkan. Modul ini tidak
// menghitung apa pun atas nilai itu — ia hanya memindahkannya — sehingga mengubahnya
// menjadi float hanya menambah kesempatan kehilangan presisi.
type Line struct {
	CompanyCode string
	BatchNumber string

	// PolicyNo dan ProductSeq adalah NOPOLIS dan PRODKE — pasangan yang menunjuk satu
	// polis pada satu masa pertanggungan.
	PolicyNo   string
	ProductSeq string

	// ClaimID adalah IDPEGA — nomor klaim yang terbit bila baris ini berhasil diproses.
	// Kosong selama baris belum diproses.
	ClaimID string

	// AcceptanceNo adalah NOAKSEPTASI.
	//
	// Pada baris yang GAGAL, kolom ini berisi TEKS GALAT, bukan nomor akseptasi —
	// `Activity/InsertKlaimToTable_Other-Act.xml` mengisi `IDPEGA`, `NOAKSEPTASI`, dan
	// `TMP_MESSAGE` ketiganya dengan pesan yang sama.
	AcceptanceNo string

	// Currency adalah isi kolom CURRENCY, yang berupa **ID**, bukan kode mata uang.
	Currency string

	// CurrencyCode adalah kode yang dibaca pengguna, hasil lookup ke POOLDATA.CURRENCY.
	//
	// Keduanya disimpan karena keduanya dipakai: ID untuk menelusuri data, kode untuk
	// ditampilkan. Versi pertama modul ini hanya punya yang pertama dan menampilkannya
	// apa adanya — layar memperlihatkan angka, bukan "IDR".
	CurrencyCode string

	ClaimAmount string

	// CauseOfLoss adalah COL_ID. Boleh kosong saat diunggah; bila kosong, pemrosesan
	// menandai barisnya gagal dengan "Penyebab kerugian tidak ditemukan"
	// (Activity/CreateCasePNC_AutoClaim-Act.xml).
	CauseOfLoss string

	DateOfLoss string
	ReportDate string

	// ProcessedDate adalah TGLPROSES — kolom yang ditampilkan layar DETAIL Pega.
	//
	// Ia bagian PRIMARY KEY `(NOPOLIS, TGLPROSES, TGLKEJADIAN)`, dan itulah yang dipakai
	// mengurutkan halaman rincian supaya OFFSET tidak melewatkan baris.
	ProcessedDate string

	// Note dan Keyword adalah NOTE dan KEYWORD, dibawa apa adanya ke case klaim.
	Note    string
	Keyword string

	// ObjectName dan FlagNoPayout adalah OBJECTNAME dan FLAGTIDAKBAYAR — dua kolom yang
	// tidak diketahui sampai DDL-nya diterima 2026-09-19.
	ObjectName   string
	FlagNoPayout string

	// Message adalah TMP_MESSAGE — hasil pemrosesan baris ini. Kosong berarti belum
	// diproses.
	Message string

	// BankRefNo adalah kolom "No Ref Bank" pada berkas ekspor — dan ia SELALU KOSONG.
	//
	// Ia bukan kolom TMP_BATCH_AUTO_CLAIM. Kueri ekspor mengambilnya lewat DB Link:
	//
	//	(select no_ref_bank from gl.t_claim_asuransi_credit@asmd.sinarmas.co.id
	//	  where no_aksep = a.noakseptasi)
	//
	// D-25 mengganti seluruh DB Link dengan API, dan API penggantinya belum ada (R-03).
	// Field-nya tetap ada supaya kolomnya tidak hilang dari berkas dan supaya tempat
	// pengisiannya kelak sudah jelas — bukan supaya ada yang mengisinya sekarang.
	BankRefNo string

	// UploadedBy adalah kolom USERINPUT.
	//
	// Ia ada di tingkat BARIS walau layar menampilkannya di tingkat batch, karena
	// begitulah tabelnya: kolom USERINPUT terisi pada setiap baris, dan grid
	// meringkasnya. Memodelkannya di tingkat batch saja akan membuat penyimpanan memori
	// tidak dapat menghitung ringkasan itu dari barisnya — dan penyimpanan yang
	// menyimpan ringkasan alih-alih menghitungnya bukan lagi tiruan tabel.
	UploadedBy string
}

// Succeeded menyatakan baris ini berhasil menjadi klaim.
func (l Line) Succeeded() bool { return l.Message == MessageSuccess }

// Processed menyatakan baris ini sudah disentuh pemrosesan, berhasil maupun gagal.
func (l Line) Processed() bool { return strings.TrimSpace(l.Message) != "" }

// Company adalah satu pilihan pada penyaring Nama Perusahaan.
type Company struct {
	Code string
	Name string
}

// CompanySummary adalah satu irisan ringkasan: berapa BATCH milik satu perusahaan.
//
// # Yang dihitung adalah BATCH, bukan baris klaim
//
// Satuannya sengaja sama dengan satu baris grid, yaitu satu kelompok
// `(INISIALID, BATCH, USERINPUT, tanggal TGLPROSES)`. Dengan begitu angka pada ringkasan
// dan jumlah baris yang tampil setelah disaring **selalu cocok** — pengguna yang melihat
// "3" lalu mengeklik perusahaan itu mendapat tepat tiga baris.
//
// Menghitung baris klaim akan terlihat lebih ramai tetapi tidak dapat diperiksa pengguna
// dari layar mana pun, karena tidak ada layar yang menampilkan seluruh baris klaim satu
// perusahaan sekaligus.
type CompanySummary struct {
	Code string

	// Name dapat kosong bila kodenya tidak ada di Master Auto Claim. Barisnya tetap ikut,
	// dengan alasan yang sama seperti pada grid: batch yang perusahaannya tidak terdaftar
	// justru yang tidak akan pernah berhasil diproses.
	Name string

	BatchCount int
}

// Summary adalah seluruh ringkasan beserta jumlah keseluruhannya.
//
// Total dihitung server, bukan dijumlahkan layar. Sebabnya bukan kenyamanan: begitu
// ringkasannya kelak dipotong (misalnya "10 perusahaan teratas"), penjumlahan di layar
// akan diam-diam menghasilkan angka yang lebih kecil dari yang sebenarnya — dan barisnya
// tetap bertuliskan "All".
type Summary struct {
	Company []CompanySummary
	Total   int
}

// BatchPage adalah satu halaman daftar batch beserta jumlah seluruh barisnya.
//
// Total ikut dikirim karena layar lama menampilkan "Total Data :" dan tombol Last
// (Section/ButtonPagingInbox-Section.xml); keduanya mustahil tanpa jumlah keseluruhan.
type BatchPage struct {
	Item  []Batch
	Total int
}

// LinePage adalah satu halaman rincian batch.
type LinePage struct {
	Item  []Line
	Total int
}

// PageRequest adalah halaman yang diminta pemanggil.
type PageRequest struct {
	// Number dimulai dari 1, mengikuti apa yang dilihat pengguna di layar.
	Number int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Clean memperbaiki nilai di luar batas alih-alih menolaknya.
//
// Halaman ke-0 dan ukuran negatif datang dari URL yang diketik tangan atau dari tautan
// lama, dan menolaknya dengan galat hanya menampilkan layar rusak untuk kesalahan yang
// jelas maksudnya. Ukuran yang melewati MaxPageSize DIPANGKAS, sesuai 10-API-STRATEGY.md §4.
func (p PageRequest) Clean() PageRequest {
	clean := p
	if clean.Number < 1 {
		clean.Number = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah jumlah baris yang dilewati untuk mencapai halaman ini.
func (p PageRequest) Offset() int {
	clean := p.Clean()
	return (clean.Number - 1) * clean.Size
}

// Source adalah TAB pada layar Inbox Auto Claim.
//
// # Ketiganya tab dari satu harness, bukan tiga layar
//
// `InboxAutoClaim/InboxAutoClaim-Harness.xml` memuat tiga tab berlabel **ANEKA**,
// **Asuransi Kredit**, dan **Travel** (`pyCaption` pada harness itu). Ketiganya berbentuk
// sama persis — grid batch, rincian, dua berkas ekspor, dan ringkasan per perusahaan —
// dan hanya berbeda pada TABEL serta NAMA KOLOM perusahaannya.
//
// # Kenapa nilainya "aneka", bukan "auto-claim"
//
// Nama rule-nya `*_AutoClaim`, tetapi label yang dilihat pengguna adalah **ANEKA**. Yang
// dipakai di sini adalah yang dilihat pengguna: Work Owner menyebutnya "Aneka", dan
// istilah yang dipakai orang yang memesannya adalah istilah yang benar (`D-81`).
type Source string

const (
	// SourceAneka adalah tab ANEKA — rule-nya bernama `*_AutoClaim`.
	SourceAneka Source = "aneka"

	// SourceKredit adalah tab Asuransi Kredit.
	SourceKredit Source = "kredit"

	// SourceTravel adalah tab Travel.
	SourceTravel Source = "travel"
)

// DefaultSource adalah tab yang terbuka lebih dulu.
//
// Asuransi Kredit, bukan ANEKA yang disebut lebih dulu di harness Pega: Work Owner
// meminta urutannya dibalik pada 2026-09-20, dan tab pertama adalah tab yang terbuka.
// Ia juga tab dengan isi terbanyak — 100 batch berbanding 11 dan 13 pada basis data
// yang dipakai sekarang.
const DefaultSource = SourceKredit

// SourceInfo menyebut apa yang membedakan satu tab dari yang lain.
//
// Hanya DUA hal yang berbeda: tabel batch-nya dan nama kolom perusahaannya. Seluruh
// bentuk kueri, hitungan, dan aturannya sama persis — dan itulah sebabnya kuerinya
// ditulis SEKALI lalu diisi dari sini, bukan disalin tiga kali.
//
// Menyalinnya tiga kali persis anti-pola yang `03-CURRENT-ARCHITECTURE.md` §4.6 catat
// sebagai utang teknis sistem lama: "satu perubahan aturan harus diterapkan di empat
// tempat — dan sering hanya diterapkan di sebagian".
type SourceInfo struct {
	// Label adalah teks tab yang dilihat pengguna, disalin dari pyCaption harness.
	Label string

	// Table adalah tabel batch milik tab ini.
	Table string

	// CompanyColumn adalah kolom kode perusahaan DI DALAM tabel itu.
	//
	// Perhatikan tab Kredit: kolomnya `AGENID`, bukan `INISIALID`. Master yang dijoin
	// tetap sama untuk ketiganya — `POOLDATA.M_AUTO_CLAIM_PNC.INISIALID`.
	CompanyColumn string
}

// sourceInfo memetakan tab ke tabel dan kolomnya.
//
// Nilainya diambil langsung dari kueri Pega:
//
//	ANEKA           BrowseClaimSPKAutoClaim   POOLDATA.TMP_BATCH_AUTO_CLAIM   INISIALID
//	Asuransi Kredit BrowseClaimSPKClaimKredit POOLDATA.TMP_BATCH_CLAIM_KREDIT AGENID
//	Travel          BrowseClaimSPKTravel      POOLDATA.TMP_BATCH_AUTO_TRAVEL  INISIALID
var sourceInfo = map[Source]SourceInfo{
	SourceAneka: {
		Label:         "ANEKA",
		Table:         "POOLDATA.TMP_BATCH_AUTO_CLAIM",
		CompanyColumn: "INISIALID",
	},
	SourceKredit: {
		Label:         "Asuransi Kredit",
		Table:         "POOLDATA.TMP_BATCH_CLAIM_KREDIT",
		CompanyColumn: "AGENID",
	},
	SourceTravel: {
		Label:         "Travel",
		Table:         "POOLDATA.TMP_BATCH_AUTO_TRAVEL",
		CompanyColumn: "INISIALID",
	},
}

// AllSource menyebut ketiga tab menurut urutan tampilnya.
//
// Urutannya ditetapkan Work Owner (2026-09-20), bukan disalin dari harness Pega yang
// menyebut ANEKA lebih dulu. Yang pertama di sini juga yang terbuka saat layar dibuka —
// lihat DefaultSource.
func AllSource() []Source { return []Source{SourceKredit, SourceAneka, SourceTravel} }

// Info mengembalikan tabel dan kolom milik tab ini.
//
// Tab yang tidak dikenal mengembalikan false, dan pemanggil WAJIB menolak permintaannya.
// Jatuh ke tab bawaan akan membaca tabel yang berbeda dari yang diminta tanpa satu pun
// tanda di layar — kelas kesalahan yang sama dengan jatuh ke portal default (`R-20`).
func (s Source) Info() (SourceInfo, bool) {
	info, exists := sourceInfo[s]
	return info, exists
}

// Label adalah teks tab yang dilihat pengguna.
func (s Source) Label() string {
	if info, exists := sourceInfo[s]; exists {
		return info.Label
	}
	return string(s)
}

// ParseSource membaca nilai tab dari permintaan.
//
// Kosong berarti tab bawaan — layar yang baru dibuka belum memilih apa pun. Nilai yang
// TIDAK dikenal ditolak, bukan dibulatkan ke bawaan: `?sumber=kredi` yang salah ketik
// akan menampilkan data ANEKA sambil menyorot tab Kredit, dan tidak ada apa pun di layar
// yang menandakannya.
func ParseSource(value string) (Source, error) {
	clean := Source(strings.ToLower(strings.TrimSpace(value)))
	if clean == "" {
		return DefaultSource, nil
	}
	if _, exists := sourceInfo[clean]; !exists {
		return "", fmt.Errorf("%w: tab %q tidak dikenal; pilihannya aneka, kredit, atau travel",
			ErrUnknownSource, value)
	}
	return clean, nil
}

// BatchFilter menyaring daftar batch.
type BatchFilter struct {
	// Source adalah TAB yang sedang dibuka. Ia menentukan tabel mana yang dibaca.
	Source Source

	// CompanyCode menyaring pada satu perusahaan; kosong berarti seluruh perusahaan.
	//
	// # Kenapa menyaring pada KODE, bukan pada nama seperti sistem lama
	//
	// Penyaring lama merangkai teks SQL dari nama perusahaan lalu menyisipkannya
	// mentah-mentah:
	//
	//	TemporaryInboxKasirAutoClaim.CaseID = "and B.NAMA_PENERIMA='"+TempSimpan1.CauseOfLoss+"'"
	//	(Activity/INBOX_AS_KREDIT_ACT_AUTOCLAIM-Act.xml, disisipkan lewat {ASIS:...})
	//
	// Itu celah SQL injection yang 08-TECHNICAL-STRATEGY.md §4.3 tutup tanpa perkecualian.
	// Menyaring pada KODE sekaligus memperbaiki cacat kedua: nama perusahaan tidak
	// dijamin unik, dan dua perusahaan bernama sama akan tercampur. Yang dilihat
	// pengguna tetap namanya — kodenya hanya nilai di balik pilihan.
	CompanyCode string

	Page PageRequest
}

// LineQuery meminta rincian satu batch.
type LineQuery struct {
	// Source adalah TAB yang sedang dibuka.
	Source Source

	CompanyCode string
	BatchNumber string

	// Result menyaring berdasarkan hasil pemrosesan; kosong berarti seluruh baris.
	Result Result

	Page PageRequest
}

// Result adalah penyaring hasil pemrosesan satu baris.
type Result string

// Ketiga nilai penyaring hasil.
const (
	// ResultAll tidak menyaring apa pun.
	ResultAll Result = ""

	// ResultSucceeded: TMP_MESSAGE = MessageSuccess.
	ResultSucceeded Result = "berhasil"

	// ResultFailed: TMP_MESSAGE terisi tetapi bukan MessageSuccess.
	//
	// Perhatikan "terisi": baris yang belum diproses BUKAN baris gagal. Kueri export
	// sistem lama pun membedakannya —
	// "AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null".
	ResultFailed Result = "gagal"
)

// ParseResult membaca nilai penyaring hasil dari permintaan.
//
// Nilai yang tidak dikenal DITOLAK, bukan diperlakukan sebagai "semua": penyaring yang
// salah ketik lalu diam-diam diabaikan akan mengunduh berkas berisi seluruh baris
// padahal pengguna meminta yang gagal saja.
func ParseResult(value string) (Result, error) {
	switch Result(strings.ToLower(strings.TrimSpace(value))) {
	case ResultAll:
		return ResultAll, nil
	case ResultSucceeded:
		return ResultSucceeded, nil
	case ResultFailed:
		return ResultFailed, nil
	default:
		return ResultAll, fmt.Errorf("%w: hasil %q tidak dikenal", ErrUnknownResult, value)
	}
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrBatchNotFound: batch yang diminta tidak ada pada portal ini.
	ErrBatchNotFound = errors.New("inboxautoclaim: batch tidak ditemukan")

	// ErrUnknownResult: penyaring hasil di luar ketiga nilai yang dikenal.
	ErrUnknownResult = errors.New("inboxautoclaim: penyaring hasil tidak dikenal")

	// ErrUnknownSource: tab di luar ketiga tab yang dikenal.
	//
	// Ia ditolak, tidak dibulatkan ke tab bawaan: membulatkannya berarti membaca tabel
	// yang berbeda dari yang diminta tanpa satu pun tanda di layar.
	ErrUnknownSource = errors.New("inboxautoclaim: tab tidak dikenal")

	// ErrEmptyUpload: berkas yang diunggah tidak memuat satu baris data pun.
	ErrEmptyUpload = errors.New("inboxautoclaim: berkas unggahan tidak memuat baris data")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar.
	//
	// Untuk unggahan ia menyebut baris berkasnya juga — "baris 12 · nopolis" — karena
	// yang diperbaiki pengguna adalah berkasnya, bukan sebuah kolom di layar.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (P-5). InputRegister_act sistem lama memeriksa
// belasan aturan lalu menampilkan semuanya bersamaan. Pada unggahan berisi ratusan baris,
// mengembalikan satu galat per percobaan berarti pengguna mengunggah ulang ratusan kali.
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "inboxautoclaim: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// FormatBatchNumber menyusun nomor batch dari nomor urut.
//
// Bentuknya angka polos tanpa pemadatan nol. Dasarnya RDB List/InsertUpdateAutoClaim-SQL.xml
// yang mendeklarasikan `tBATCH NUMBER;` sebelum meneruskannya ke POOLDATA.INSERT_AUTOCLAIM
// — kolomnya diperlakukan sebagai angka oleh satu-satunya prosedur yang menulisinya.
func FormatBatchNumber(sequence int) string {
	return strconv.Itoa(sequence)
}

// NextBatchNumber memilih nomor batch berikutnya dari nomor yang sudah dipakai.
//
// # Kenapa dihitung di Go, bukan dengan MAX di SQL
//
// Alasannya sama dengan masterstatusprogres.nextID, dan penyebabnya sama: DDL tabelnya
// tidak ada di export (R-08), sehingga tipe kolom BATCH tidak dapat dipastikan. Bila ia
// VARCHAR2, MAX(BATCH) adalah maksimum LEKSIKOGRAFIS — begitu tabel memuat "10",
// maksimumnya tetap "9". Nomor berikutnya kembali 10, dan dua batch memakai nomor sama.
//
// Menafsirkan setiap nomor sebagai angka menghindari pertanyaan itu seluruhnya, dan untuk
// kolom yang memang numerik hasilnya sama persis.
//
// Nomor yang tidak dapat ditafsirkan sebagai angka tetap DIHITUNG TERPAKAI walau tidak
// ikut menentukan yang terbesar: baris lama dapat memuat apa saja, dan menabraknya jauh
// lebih buruk daripada melewatinya.
func NextBatchNumber(used []string) string {
	taken := make(map[string]bool, len(used))
	highest := 0
	for _, number := range used {
		clean := strings.TrimSpace(number)
		taken[clean] = true
		if value, err := strconv.Atoi(clean); err == nil && value > highest {
			highest = value
		}
	}

	for candidate := highest + 1; ; candidate++ {
		next := FormatBatchNumber(candidate)
		if !taken[next] {
			return next
		}
	}
}

// Repo adalah seam ke penyimpanan Inbox Auto Claim SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (ADR-0030 Opsi 1). Yang memilih instans mana yang dipakai satu permintaan
// adalah RepoSelector.
type Repo interface {
	// ListBatch mengembalikan satu halaman daftar batch beserta jumlah seluruhnya.
	ListBatch(ctx context.Context, filter BatchFilter) (BatchPage, error)

	// ListCompany mengembalikan pilihan penyaring Nama Perusahaan.
	//
	// Sumbernya MASTER perusahaan, mengikuti
	// InboxAutoClaim/BrowseCompanyClaimCredit-SQL.xml. Perusahaan yang belum pernah
	// mengirim apa pun karena itu ikut tampil, dan memilihnya menghasilkan daftar kosong
	// — itu perilaku aslinya, dan justru berguna: petugas jadi dapat memastikan batch-nya
	// memang belum ada, bukan sekadar melihat namanya hilang dari daftar.
	ListCompany(ctx context.Context) ([]Company, error)

	// SummarizeCompany menghitung JUMLAH BATCH per perusahaan untuk panel ringkasan.
	//
	// Ia kueri tersendiri, bukan hasil menjumlahkan ListBatch: ListBatch dipotong
	// paginasi, sehingga menjumlahkannya hanya akan meringkas halaman yang sedang tampil
	// — dan angkanya berubah-ubah setiap kali pengguna berpindah halaman.
	//
	// Sumbernya TABEL BATCH, bukan master, dan itu berbeda dengan ListCompany dengan
	// sengaja: ringkasan menjawab "apa yang ADA", penyaring menjawab "apa yang DAPAT
	// DIPILIH". Perusahaan tanpa batch layak muncul di penyaring, tetapi tidak layak
	// muncul di grafik sebagai irisan bernilai nol.
	SummarizeCompany(ctx context.Context, source Source) (Summary, error)

	// ListLine mengembalikan satu halaman rincian satu batch.
	ListLine(ctx context.Context, query LineQuery) (LinePage, error)

	// ExportLine mengembalikan SELURUH baris yang cocok, tanpa paginasi.
	//
	// Terpisah dari ListLine karena keduanya menjawab kebutuhan berbeda: layar membaca
	// sepotong, berkas unduhan membaca semuanya. Menyatukannya memaksa salah satunya
	// berkompromi.
	ExportLine(ctx context.Context, query LineQuery) ([]Line, error)

	// BatchExists menyatakan pasangan (perusahaan, batch) benar-benar ada.
	//
	// Dipakai sebelum mengunduh berkas: unduhan kosong untuk batch yang salah ketik
	// tidak dapat dibedakan dari unduhan kosong untuk batch yang memang belum diproses.
	BatchExists(ctx context.Context, source Source, companyCode, batchNumber string) (bool, error)

	// ResolveReceiver menurunkan perusahaan rekanan dari NOMOR POLIS.
	//
	// Ini langkah pertama rantai unggah, dan sumber satu-satunya kode perusahaan: berkas
	// unggahan TIDAK memuatnya. Asalnya `t_general.sourceofbusiness` milik polis,
	// dicocokkan ke master yang `CLAIM_ALLOWED = 1` dan `APPROVAL = '1'`
	// (RDB List/GetReceiverClaimAsuransiKredit-SQL.xml).
	//
	// Nilai kedua false bila tidak ketemu — bukan galat, melainkan hasil yang sah yang
	// berarti MessageReceiverNotFound.
	ResolveReceiver(ctx context.Context, policyNo string) (Company, bool, error)

	// FindPolicyProductSeq mencari PRODKE termutakhir sebuah polis.
	//
	// Langkah kedua rantai unggah (RDB List/BrowsePolisAso-SQL.xml, langkah "get prodke").
	// Nilai kedua false bila polisnya tidak ada di JSON_POLIS.
	FindPolicyProductSeq(ctx context.Context, policyNo string) (string, bool, error)

	// InsertUpload menyimpan seluruh baris dalam SATU transaksi dan mengembalikan batch
	// yang terbentuk.
	//
	// Baris yang Message-nya terisi TETAP DISISIPKAN — itu perilaku
	// `Activity/InsertKlaimToTable_Other-Act.xml`, yang menulis alasan kegagalan ke
	// TMP_MESSAGE alih-alih menolak berkas. Versi pertama modul ini menolak seluruh
	// berkas bila ada satu baris cacat; itu model yang berbeda, dan diperbaiki setelah
	// activity aslinya tiba.
	//
	// Nomor batch diturunkan di dalam repo, bukan diterima dari pemanggil: ia berasal dari
	// isi tabel itu sendiri, dan memisahkan "ambil nomor" dari "sisipkan" melebarkan jarak
	// yang justru membuat dua unggahan bersamaan bertabrakan.
	InsertUpload(ctx context.Context, source Source, line []UploadLine, uploadedBy string) (UploadResult, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
type RepoSelector func(portalAlias string) (Repo, error)
