// Package monitoringslinkojk adalah inti modul **Monitoring SLINK OJK** (`MENU_ID 78`).
//
// # Yang dimigrasikan
//
//	Harness/MonitoringSLINKOJK-Harness.xml          layar rujukan, judul "Inbox Monitoring Slik OJK"
//	Section/Sec_MonitoringSLINKOJK-Section.xml      pembungkus + dropdown "Pilih Segmen"
//	Section/Sec_SegmentD01_1-Section.xml            segmen D01 — 20 kolom, 6 tombol
//	Section/Sec_SegmentF06-Section.xml              segmen F06 — 38 kolom, 4 tombol
//	Activity/GetTempDataD01-Act.xml                 tombol "Cari Data" segmen D01
//	Activity/GetAllDataSumbisSlink-Act.xml          tombol "Cari Data" segmen F06
//	Activity/ExportDataSlinkD01-Act.xml             tombol "Export Data" segmen D01
//	Activity/ExportDataSlinkFOG-Act.xml             tombol "Export Data" segmen F06
//	Activity/DownloadFileCSVFormatSlikOJK-Act.xml   tombol "Format File"
//	RDB List/GetDataSlinkAllFOGF06-SQL.xml          kueri segmen D01
//	RDB List/GetDataSlinkAllFOG-SQL.xml             kueri segmen F06
//
// # Apa yang dipantau layar ini
//
// SLIK adalah **Sistem Layanan Informasi Keuangan** OJK. Untuk lini **Asuransi Kredit**
// (SPK) dan **Surety Bond**, setiap klaim yang dibayarkan wajib dilaporkan ke OJK dalam
// bentuk berkas bersegmen. Layar ini memantau **apa yang akan dan sudah dilaporkan**.
//
// Dua segmen, dan keduanya melaporkan hal yang BERBEDA:
//
//	D01  FASILITAS kredit — nomor rekening fasilitas, kolektibilitas, tunggakan, kondisi
//	F06  DEBITUR individu — identitas, alamat, pekerjaan, pasangan, penghasilan
//
// Keduanya bukan dua tampilan dari data yang sama. D01 membaca tabel yang SUDAH terisi
// (`POOLDATA.T_CLAIM_SLIK_OJK`); F06 membaca data klaim SUMBERNYA
// (`POOLDATA.T_CLAIM_OBJECTLIST`) — yakni calon laporan yang belum tersusun. Itulah sebab
// keduanya punya penyaring yang sama tetapi jumlah kolom yang jauh berbeda.
//
// # Kenapa kolomnya berupa KATALOG, bukan field struct
//
// Segmen F06 punya **38 kolom**, dan segmen D01 punya 20. Menuliskannya sebagai field
// struct berarti 58 field yang harus diulang di DTO, di CSV, dan di tabel frontend —
// empat tempat yang harus berubah bersamaan setiap kali satu kolom bergeser.
//
// Presedennya sudah ada di modul `reportklaim`, yang menyalin ~700 kolom dari 28 laporan
// dengan cara yang sama: `Column{Key, Header, Source}` beserta `Row` bertipe peta.
// Katalognya dikirim ke layar, sehingga tabel 38 kolom digambar dari satu sumber
// kebenaran alih-alih ditulis ulang dalam TypeScript.
//
// # Penamaan: alias Pega SENGAJA dipertahankan di modul ini
//
// Ini **pengecualian yang diminta Work Owner (2026-09-26)**, bukan kelalaian. `D-19` dan
// `D-80` menetapkan alias Pega yang salah arti tidak dibawa; di sini dua penyaringnya
// tetap bernama seperti di Pega:
//
//	DateOfLoss             -> sebenarnya berisi "Dari"   (batas bawah tanggal registrasi)
//	DateOfRequestDocument  -> sebenarnya berisi "Sampai" (batas atas tanggal registrasi)
//
// Tidak satu pun berhubungan dengan tanggal kejadian maupun tanggal terima dokumen.
// Namanya dipertahankan supaya penelusuran ke `GetTempDataD01` dan `GetAllDataSumbisSlink`
// tetap langsung; ARTINYA dijelaskan di setiap tempat ia muncul, termasuk di DTO dan di
// berkas .sql. Itulah harga yang disepakati: nama yang menyesatkan ditebus dengan
// keterangan yang tidak boleh hilang.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package monitoringslinkojk

import (
	"context"
	"strings"
	"time"
)

// Segment adalah pilihan pada dropdown "Pilih Segmen".
//
// Nilainya kode segmen SLIK OJK apa adanya — `D01` dan `F06` — bukan nama section Pega
// (`Sec_SegmentD01_1`, `Sec_SegmentF06`). Kode segmen itulah yang dikenal pelapor dan
// yang tertulis di berkas laporan ke OJK; nama section hanya artefak Pega.
type Segment string

const (
	// SegmentD01 — segmen FASILITAS kredit.
	SegmentD01 Segment = "D01"

	// SegmentF06 — segmen DEBITUR individu.
	SegmentF06 Segment = "F06"
)

// Valid menyatakan segmennya dikenal.
func (s Segment) Valid() bool {
	return s == SegmentD01 || s == SegmentF06
}

// Label adalah judul segmen yang dilihat pengguna.
//
// Diambil dari caption tautan silang di kedua section — "Segment Slik D01" dan
// "Segment Slik F06" — karena itulah satu-satunya tempat di export yang menamai keduanya.
func (s Segment) Label() string {
	switch s {
	case SegmentD01:
		return "Segment Slik D01"
	case SegmentF06:
		return "Segment Slik F06"
	}
	return string(s)
}

// ParseSegment membaca kode segmen dari permintaan.
//
// Kosong jatuh ke D01, karena itulah segmen yang terbuka lebih dulu di layar lama:
// `Sec_MonitoringSLINKOJK` menyertakan `Sec_SegmentD01_1` sebagai bawaan, dan F06
// dicapai lewat tautan "Segment Slik F06".
func ParseSegment(raw string) (Segment, bool) {
	clean := Segment(strings.ToUpper(strings.TrimSpace(raw)))
	if clean == "" {
		return SegmentD01, true
	}
	if !clean.Valid() {
		return "", false
	}
	return clean, true
}

// BusinessScope adalah pilihan dropdown "Business Name".
//
// # Ia BUKAN daftar lini bisnis, melainkan SATU syarat yang dibalik
//
// `GetTempDataD01-Act.xml` hanya punya dua cabang, dan keduanya menyaring kolom yang
// sama dengan operator yang berlawanan:
//
//	BusinessName == "AS. KREDIT"   ->  "and b.businesstype  = 'AsuransiKredit'"
//	BusinessName == "SURETY BOND"  ->  "and b.businesstype != 'AsuransiKredit'"
//
// Jadi "SURETY BOND" TIDAK berarti lini Surety Bond; ia berarti **seluruh lini selain
// Asuransi Kredit**. Bila kelak ada lini ketiga yang wajib lapor SLIK, ia akan ikut
// terbawa ke dalam pilihan itu tanpa satu pun tanda di layar.
//
// Itu dicatat sebagai perilaku warisan, bukan diperbaiki diam-diam: `P-5` menetapkan
// hasil yang benar adalah hasil yang sama dengan Pega, kecuali yang diputuskan eksplisit
// pada 13 butir `D-49` — dan ini bukan salah satunya.
type BusinessScope string

const (
	// ScopeAll — dropdown belum dipilih; seluruh lini ikut terbawa.
	ScopeAll BusinessScope = ""

	// ScopeCreditInsurance — "AS. KREDIT" → businesstype = 'AsuransiKredit'.
	ScopeCreditInsurance BusinessScope = "AS. KREDIT"

	// ScopeSuretyBond — "SURETY BOND" → businesstype != 'AsuransiKredit'. Lihat catatan
	// pada BusinessScope: ia MENIADAKAN, bukan memilih.
	ScopeSuretyBond BusinessScope = "SURETY BOND"
)

// CreditInsuranceBusinessType adalah nilai `BUSINESSTYPE` yang menandai Asuransi Kredit.
//
// # Ia BELUM SEHARUSNYA berupa konstanta, dan itu diakui di sini
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode; tempatnya adalah
// master Lini Bisnis pada `F-4`. Sampai master itu ada, nilainya tinggal di SATU tempat
// alih-alih tersebar di dua kueri — sehingga ketika masternya tiba, yang berubah hanya
// baris ini.
const CreditInsuranceBusinessType = "AsuransiKredit"

// Valid menyatakan pilihannya dikenal.
func (s BusinessScope) Valid() bool {
	return s == ScopeAll || s == ScopeCreditInsurance || s == ScopeSuretyBond
}

// ParseBusinessScope membaca pilihan "Business Name" dari permintaan.
func ParseBusinessScope(raw string) (BusinessScope, bool) {
	clean := BusinessScope(strings.ToUpper(strings.TrimSpace(raw)))
	if !clean.Valid() {
		return "", false
	}
	return clean, true
}

// BusinessScopes adalah isi dropdown "Business Name", sesuai urutan di layar lama.
//
// Dikembalikan ke layar alih-alih ditulis ulang di TypeScript: keduanya harus sepakat,
// dan satu-satunya cara menjamin itu adalah satu sumber.
func BusinessScopes() []BusinessScope {
	return []BusinessScope{ScopeCreditInsurance, ScopeSuretyBond}
}

// Filter adalah penyaring layar — sama untuk kedua segmen.
//
// Keempatnya opsional. Layar lama pun membiarkan seluruhnya kosong: `GetTempDataD01`
// memasang syarat tanggal HANYA bila "Dari" terisi
// (`pyStepsPreCondParamsWhen: DetailTempSlink.DateOfLoss==""`), dan tanpa itu kuerinya
// membaca seluruh baris.
type Filter struct {
	// BusinessScope adalah pilihan dropdown "Business Name". Lihat BusinessScope.
	BusinessScope BusinessScope

	// Tidak ada isian "Tipe Generate" di sini, dan itu disengaja.
	//
	// Properti `.GenerateType` memang ADA di `Sec_SegmentD01_1-Section.xml` sebagai
	// dropdown berlabel "Tipe Generate", dan atas dasar itu ia sempat dibangun. Tangkapan
	// layar Pega yang berjalan (2026-10-08) membuktikan ia **tidak tampil**: segmen D01 di
	// sana hanya punya Business Name, Dari, dan Sampai.
	//
	// Section di export kita karena itu lebih tua daripada yang terpasang — hal yang sama
	// sudah terbukti dua arah: tombol yang ada di XML tetapi tidak dirender, dan kolom
	// "Nama Debitur" yang dirender tetapi tidak ada di XML.
	//
	// Dicabut. Tidak ada kueri maupun aktivitas yang membacanya, sehingga tidak ada
	// perilaku yang hilang.

	// DateOfLoss adalah isian berlabel **"Dari"** — batas BAWAH tanggal registrasi klaim.
	//
	// Namanya menyesatkan dan sengaja dipertahankan; lihat kepala paket. Ia TIDAK ada
	// hubungannya dengan Tanggal Kejadian.
	DateOfLoss *time.Time

	// DateOfRequestDocument adalah isian berlabel **"Sampai"** — batas ATAS tanggal
	// registrasi klaim.
	//
	// Namanya menyesatkan dan sengaja dipertahankan; lihat kepala paket. Ia TIDAK ada
	// hubungannya dengan Tanggal Terima Dokumen.
	DateOfRequestDocument *time.Time

	// Tidak ada kotak pencarian di sini, dan itu disengaja.
	//
	// Sempat ada — mencari pada No Klaim dan Contract No — sebagai penambahan yang
	// dicatat. Layar lama tidak punya: sectionnya hanya mengikat "Business Name",
	// "Dari", dan "Sampai", dan kedua kueri pencariannya hanya memuat dua placeholder
	// (`NoteKasir` dan `BusinessID`). Work Owner meminta layar ini disamakan dengan
	// Pega (2026-09-27), sehingga kotaknya dicabut.
	//
	// Akibatnya: pada hasil yang berhalaman, satu klaim tertentu hanya dapat ditemukan
	// dengan mempersempit rentang tanggalnya.

	// Page dan Size adalah paginasi, dihitung dari 1.
	Page int
	Size int
}

// Batas paginasi.
//
// `DefaultPageSize` mengikuti `pyMaxRecords=500` yang terpasang pada 54 dari 56 laporan
// Pega — bukan karena angka itu benar, melainkan karena ia satu-satunya angka yang pernah
// berlaku. `MaxPageSize` memagarinya supaya satu permintaan tidak menarik puluhan juta
// baris sebelum `ADR-0011` menjawab berapa baris yang wajib dilayani.
const (
	DefaultPageSize = 50
	MaxPageSize     = 500
)

// Normalize merapikan penyaring dan mengisi nilai bawaan paginasi.
//
// Dipanggil usecase, bukan transport: aturan "halaman minimal 1" adalah aturan modul,
// dan menaruhnya di transport berarti pemanggil lain — uji, perkakas, ekspor — harus
// mengulanginya.
func (f Filter) Normalize() Filter {
	clean := f

	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah banyaknya baris yang dilewati untuk halaman yang diminta.
func (f Filter) Offset() int {
	return (f.Page - 1) * f.Size
}

// DateRangeReversed menyatakan "Dari" berada SESUDAH "Sampai".
//
// Sistem lama tidak memeriksanya: `GetTempDataD01` merangkai kedua batas apa adanya,
// sehingga rentang terbalik menghasilkan **nol baris tanpa satu pun pesan**. Bagi layar
// pemantauan laporan regulator, nol baris yang tampak wajar adalah keadaan paling
// berbahaya — ia terbaca sebagai "tidak ada yang perlu dilaporkan".
//
// Ia karena itu ditolak dengan pesan, bukan dilayani dengan senarai kosong.
func (f Filter) DateRangeReversed() bool {
	if f.DateOfLoss == nil || f.DateOfRequestDocument == nil {
		return false
	}
	return f.DateOfLoss.After(*f.DateOfRequestDocument)
}

// Row adalah satu baris grid, dikunci dengan Column.Key.
//
// Peta, bukan struct, karena kolomnya berupa katalog — lihat kepala paket. Nilainya
// SUDAH berupa teks siap tampil: pemformatan tanggal dan angka dilakukan di lapisan
// penyimpanan (`08-TECHNICAL-STRATEGY.md` §4.3 melarang `TO_CHAR` untuk tampilan di SQL),
// sehingga layar dan CSV membaca nilai yang sama persis.
type Row map[string]string

// Get mengembalikan isi satu kolom, atau teks kosong bila kolomnya tidak terisi.
//
// Teks kosong, bukan penanda "tidak ada". Kolom F06 yang tidak punya sumber di export
// memang tidak akan pernah terisi, dan membedakan "kosong" dari "tidak ada sumber" adalah
// tugas katalog (lihat Column.Source) — bukan tugas setiap sel.
func (r Row) Get(key string) string {
	if r == nil {
		return ""
	}
	return r[key]
}

// Page adalah satu halaman hasil beserta jumlah seluruh barisnya.
type Page struct {
	Rows []Row

	// Total adalah jumlah SELURUH baris yang cocok di server, bukan yang tampil di
	// halaman ini. Ia yang membuat bilah halaman berarti.
	Total int
}

// Repo membaca DAN menulis data Monitoring SLINK OJK.
//
// # Empat method pertama membaca, empat berikutnya menulis
//
// Aksi tulisnya ditambahkan pada 2026-09-26 atas keputusan Work Owner, sesudah
// konsekuensi `P-1` disampaikan — lihat kepala write.go. Sebelumnya modul ini membaca
// saja, dan pembatasan itu dicabut secara sadar, bukan luput.
type Repo interface {
	// Search mengembalikan satu halaman baris pada segmen yang diminta.
	Search(ctx context.Context, segment Segment, filter Filter) (Page, error)

	// Stream membaca SELURUH baris yang cocok, satu per satu, untuk keperluan ekspor.
	//
	// Ia terpisah dari Search — bukan Search dengan ukuran halaman besar — karena
	// ekspor tidak boleh menampung seluruh hasilnya di memori lebih dulu
	// (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6). Penghentian dini dinyatakan
	// dengan mengembalikan galat dari `emit`.
	Stream(ctx context.Context, segment Segment, filter Filter, emit func(Row) error) error

	// StreamSource membaca data klaim SUMBER yang siap disusun menjadi laporan.
	//
	// Padanan `RDB List/GetDataSlinkAllFOG-SQL.xml`, dan ia BERBEDA dari Stream segmen
	// F06 meski kuerinya bersaudara: yang ini mengambil seluruh kolom yang dibutuhkan
	// INSERT, sedangkan yang itu hanya kolom yang tampil di grid.
	StreamSource(ctx context.Context, filter Filter, emit func(ReportEntry) error) error

	// CountReport menghitung baris laporan yang sudah ada untuk satu klaim.
	//
	// Padanan `GetCountTClaimSlikOJK`. Hasilnya menentukan `operasidata` — lihat
	// DataOperationFor.
	CountReport(ctx context.Context, claimID string) (int, error)

	// InsertReport menyusun satu baris laporan ke POOLDATA.T_CLAIM_SLIK_OJK.
	//
	// Padanan `InsertDataSlikOJKF06`. Ia MENAMBAH, tidak menimpa — sama seperti sistem
	// lama, yang menandai baris berulang lewat `operasidata = 'U'` alih-alih
	// memperbaruinya.
	InsertReport(ctx context.Context, entry ReportEntry) error

	// LoadDebtor membaca data debitur yang menyusun badan permintaan pendaftaran klien.
	//
	// # Kenapa ia method tersendiri, bukan diambil dari baris grid
	//
	// Karena yang dikirim BUKAN isi laporan SLIK melainkan identitas debiturnya — nama,
	// jenis kelamin, tanggal lahir, alamat, telepon. Sistem lama membacanya dari
	// `pyWorkPagee.ClaimData.PolicyData.CIFData`, yaitu CIF pada snapshot polis.
	//
	// Di sini sumbernya `POOLDATA.T_CLAIM_OBJECTLIST`, yang memuat kolom padanannya dan
	// sudah dibaca kueri segmen F06. Itu **pilihan yang dicatat**, bukan kesetaraan yang
	// terbukti: bila CIF polis memuat field yang tidak ada di tabel objek — NIK, NPWP,
	// nama ibu kandung — muatannya lebih miskin daripada yang dikirim Pega.
	//
	// Field yang tidak tersedia dibiarkan KOSONG, tidak diisi tebakan. Penerimanya hanya
	// mewajibkan satu field per cabang (lihat Debtor.RequiredFieldMissing), sehingga
	// muatan yang lebih miskin tetap diterima — hanya kurang lengkap.
	//
	// Tercatat sebagai pertanyaan terbuka di `docs/permintaan-artefak-pega.md`.
	LoadDebtor(ctx context.Context, claimID, contractNo string) (Debtor, error)

	// NextSubmissionID mengembalikan nomor urut pengiriman berikutnya.
	//
	// Padanan `select nvl(max(id),0)+1 as "CASEDB" from pooldata.t_claim_slink_individu`.
	NextSubmissionID(ctx context.Context) (int64, error)

	// RecordSubmission mencatat satu pengiriman SEBELUM dikirim.
	//
	// Urutannya mengikuti sistem lama: dicatat lebih dulu, baru dikirim. Dengan begitu
	// pengiriman yang gagal di tengah tetap meninggalkan jejak — baris tanpa
	// `id_transaction` adalah pengiriman yang tidak pernah sampai.
	RecordSubmission(ctx context.Context, submission Submission) error

	// CompleteSubmission menyimpan jawaban sistem SLIK ke baris pengirimannya.
	//
	// Padanan `UpdateTransactionClaimSlinkIndividu`.
	CompleteSubmission(ctx context.Context, submission Submission, result SubmissionResult) error
}

// RepoSelector memilih penyimpanan milik satu portal.
//
// Fungsi, bukan repo tunggal: laporan SLIK adalah kewajiban regulator milik SATU badan
// hukum, dan satu repo bersama akan menampilkan kewajiban lapor entitas lain tanpa satu
// pun pesan galat (`R-20`, `ADR-0030`).
type RepoSelector func(portalAlias string) (Repo, error)

// Caller adalah identitas pemanggil.
type Caller struct {
	// Login adalah nama pengguna yang diketik saat masuk — padanan
	// `OperatorID.pyUserIdentifier` di sistem lama.
	Login string
}

// Clean merapikan identitas pemanggil.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}
