// Package archivedokumenklaim adalah inti modul Archive Dokumen Klaim.
//
// # Layar apa ini
//
// Menu `MENU_ID 77` "Archive Dokumen Klaim" pada POOLDATA.M_MENU_APLIKASI_PNC, di bawah
// kelompok VIEW (`MENU_ID 3`, urutan 1167), yang menunjuk harness `PNCArchiveDokumen`.
//
// Isinya pencatatan **berkas fisik klaim yang diarsipkan** — berapa lembar, jenis
// dokumennya apa, disimpan di boks mana, dengan kode filling berapa — lalu pengirimannya
// ke sistem Arsip milik tim lain. Ia BUKAN Inbox menurut `D-79`: barisnya bukan pekerjaan
// yang menunggu, tidak hilang setelah ditindaklanjuti, dan tidak punya tenggat.
//
// # Tiga bagian, dan ketiganya dibangun
//
// Work Owner menetapkan 2026-09-24 modul ini mengikuti layar Pega apa adanya, termasuk
// ketiga bagiannya:
//
//	Cari Archive      grid ARCHIVE FILE KLAIM atas POOLDATA.T_CLAIM_ARCHIVE_FILE
//	Input Archive     cari klaim di T_CLAIM_PNC, pilih satu, isi berkasnya, simpan
//	Kirim ke Cabang   kirim baris ber-CABANGSTATUS='0' ke layanan Arsip, lalu tandai
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/PNCArchiveDokumen-Harness.xml             pembungkus layar; judul dan isian
//	Section/SecArchiveDokumen-Section.xml             susunan grid dan formulir
//	Section/KodeArchiveDoc-Section.xml                pemilih Kode Filling (SecCariKodeArchiveDoc)
//	Activity/SearchDataArchiveFilling-Act.xml         dua mode pencarian + pengayaan nama dokumen
//	Activity/ShowInsertArchiveKlaim_Act-Act.xml       tiga tipe pencarian klaim
//	Activity/SetDataArchiveDokumentCase-Act.xml       pemindahan baris klaim terpilih ke formulir
//	Activity/FlagForArchiveData-Act.xml               pengosongan formulir antar mode
//	Activity/GetDataArchiveCabangKlaim-Act.xml        daftar kirim ke cabang + saringan jabatan
//	Activity/SENDDATACABANGKEARCHIVE-Act.xml          urutan kirim lalu tandai
//	Activity/SendDataArchiveDOcumentByService-Act.xml badan permintaan dan pemetaan jawaban
//	Connect REST/InjectDataArchiveDokumentKlaim-ConnectREST.xml  alamat dan bentuk layanan
//	RDB List/SearchDataArchiveFillingCase-SQL.xml     kueri grid arsip
//	RDB List/SearchArchiveInsert-SQL.xml              kueri calon klaim (no klaim/nama)
//	RDB List/SearchArchiveInsertPolis-SQL.xml         kueri calon klaim (no polis)
//	RDB List/InsertToClaimArchive-SQL.xml             pemanggilan prosedur simpan
//	RDB List/UpdateDataArchiveKlaimSetelahService-SQL.xml  penyimpanan jawaban layanan
//	Database/INSERTDATASFILLINGARCHIVE.prc            logika simpan yang ditulis ulang di Go
//
// # Kenapa nama field di sini tidak mirip nama properti Pega
//
// Karena layar lama memakai properti klipboard yang sudah ada alih-alih membuat properti
// baru, sehingga namanya tidak lagi menyatakan isinya — utang teknis
// `03-CURRENT-ARCHITECTURE.md` §4.2 dalam bentuk paling pekat setelah View History Claim.
// Jumlah lembar dibawa `.AgingAmount`, tipe dokumen dibawa `.RWID`, jenis dokumen dibawa
// `.TelpTertanggung`, nama boks dibawa `.CABANG`, dan kode filling dibawa `.KodeCabang`.
// Pemetaan lengkapnya berdampingan ada di repo/sqlstore/archivedokumenklaim.sql.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam yang ia deklarasikan sendiri.
//
// # Susunan subpaket
//
//	archivedokumenklaim/          aturan modul + seam          ← paket ini
//	archivedokumenklaim/usecase/  orkestrasi: cari, simpan, kirim
//	archivedokumenklaim/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	archivedokumenklaim/gateway/  pengisi seam layanan Arsip   — http, fake
//	archivedokumenklaim/http/     lapisan transport modul ini  — handler, dto, rute
package archivedokumenklaim

import (
	"context"
	"strings"
	"time"
)

// ArchiveFile adalah satu baris POOLDATA.T_CLAIM_ARCHIVE_FILE — satu berkas klaim yang
// sudah diarsipkan.
//
// # Kenapa kode DAN namanya dibawa bersama
//
// Tipe dan jenis dokumen tersimpan sebagai kode, sedangkan grid menampilkan namanya.
// Sistem lama mengambil namanya dengan satu kueri PER BARIS di dalam perulangan
// (`Activity/SearchDataArchiveFilling-Act.xml` langkah 12–16). Di sini keduanya dibawa
// sekaligus lewat satu kueri bergabung, sehingga layar tidak perlu menembak lagi dan
// aturan "tidak ada kueri di dalam perulangan" (`15-NFR` §3.2) tidak dilanggar.
type ArchiveFile struct {
	// ID adalah kolom ID_ARCHIVE. Ia dibutuhkan untuk mengubah baris dan untuk
	// mengirimkannya ke layanan Arsip.
	ID int64

	// ClaimNumber — NOKLAIM, di klipboard lama dibawa `.CaseID`.
	ClaimNumber string

	// PolicyNumber — NOPOLIS, dibawa `.PolicyNo`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — TERTANGGUNG, dibawa `.NIK`.
	//
	// Nama properti lamanya menyebut NIK; isinya nama. Itu bukan salah baca melainkan
	// properti yang dipinjam.
	InsuredName string

	// LossDate adalah Tanggal Kejadian — DOL, dibawa `.DateOfLoss`.
	//
	// Nil bila kolomnya kosong. Pointer, bukan time.Time kosong: tanggal nol tahun 1
	// tidak dapat dibedakan dari "belum diisi" saat ditampilkan.
	LossDate *time.Time

	// TechnicalPIC adalah PIC Teknis — PICTEKNIK, dibawa `.UserTeknis`.
	TechnicalPIC string

	// DocumentReceivedDate adalah Tgl Terima Dokumen — TGLTERIMADOK, dibawa
	// `.TanggalCetakDLA`.
	DocumentReceivedDate *time.Time

	// InputDate adalah TGL INPUT — TGLINPUT, dibawa `.TanggalAnalystSendRCL`.
	//
	// Ia diisi waktu basis data saat baris dibuat, bukan tanggal yang dipilih pengguna.
	// Lihat Draft.
	InputDate *time.Time

	// SheetCount adalah Jumlah Lembar — JUMLAHLEMBAR, dibawa `.AgingAmount`.
	SheetCount int

	// DocumentTypeCode adalah Tipe Dokumen — TIPEDOK, dibawa `.RWID`.
	// Kodenya menunjuk POOLDATA.V_LST_DOC_TYPE.ID.
	DocumentTypeCode string

	// DocumentTypeName adalah nama tipe dokumen — V_LST_DOC_TYPE.STS_PROSES.
	// Kosong bila kodenya tidak ada di master; grid lama pun menampilkannya kosong.
	DocumentTypeName string

	// DocumentKindCode adalah Jenis Dokumen — JENISDOK, dibawa `.TelpTertanggung`.
	// Kodenya menunjuk POOLDATA.V_LST_DET_TYPE_DOC.ID.
	DocumentKindCode string

	// DocumentKindName adalah nama jenis dokumen — V_LST_DET_TYPE_DOC.DETAIL_DOCUMENT.
	DocumentKindName string

	// BoxName adalah Nama BOX — NAMABOX, dibawa `.CABANG`.
	BoxName string

	// FillingCode adalah Kode Filling — KODEFILLING, dibawa `.KodeCabang`.
	FillingCode string

	// InputUser adalah User Input — USERINPUT, dibawa `.UserName`.
	InputUser string

	// SentDate adalah Tanggal Kirim Dok — TGLKIRIMDOK, dibawa `.TanggalAI`.
	//
	// Ia TIDAK pernah diisi jalur simpan; hanya jalur kirim ke cabang yang mengisinya.
	SentDate *time.Time

	// GroupPanel adalah lini bisnis klaim — GROUPPANEL. Ia tidak ditampilkan di grid
	// arsip, tetapi menentukan baris mana yang tampak di daftar kirim ke cabang.
	GroupPanel string

	// BranchStatus adalah CABANGSTATUS: "0" belum dikirim ke layanan Arsip, "1" sudah.
	BranchStatus string

	// ServiceCode dan ServiceNote adalah jawaban layanan Arsip — KODESERVICE dan
	// NOTESERVICE. Keduanya kosong sebelum pengiriman pertama.
	ServiceCode string
	ServiceNote string
}

// BranchStatus yang dikenali.
//
// Nilainya teks, bukan boolean, karena begitulah kolomnya tersimpan; memetakannya ke
// boolean akan membuat nilai di luar kedua ini hilang tanpa jejak.
const (
	BranchStatusPending = "0"
	BranchStatusSent    = "1"
)

// ClaimCandidate adalah satu baris hasil pencarian klaim pada bagian Input Data Archive.
//
// Ia dibaca dari POOLDATA.T_CLAIM_PNC dan BUKAN dari tabel arsip: yang dicari di sini
// adalah klaim yang berkasnya hendak diarsipkan, bukan arsip yang sudah ada.
type ClaimCandidate struct {
	// Number adalah No Klaim — CLAIMNO, di grid lama dibawa `.Notes`.
	Number string

	// PolicyNumber — NOPOLIS, dibawa `.BranchOfBank`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — QQNAME, dibawa `.NameOfBank`.
	InsuredName string

	// LossDate adalah Tgl Kejadian — DATEOFLOSS, dibawa `.DateTransferPajak`.
	LossDate *time.Time

	// BusinessName adalah Bisnis — BUSINESSNAME, dibawa `.Receiver`.
	BusinessName string

	// BranchName adalah Cabang — BRANCHNAME, dibawa `.BatasLapor`.
	BranchName string

	// WorkStatus adalah Status — STATUSWORK, dibawa `.StatusClaim`.
	WorkStatus string

	// ClaimPosition adalah Posisi Klaim — V_STS_CLAIM.LSC_NOTE, dibawa `.BranchOfBank2`.
	ClaimPosition string

	// CloseDate adalah Tanggal Close — CLOSECLAIMDATE, dibawa `.AcceptedDate`.
	CloseDate *time.Time

	// CloseNote adalah Catatan Close — CLOSECLAIMNOTE, dibawa `.NoteAkseptasi`.
	CloseNote string

	// TechnicalPIC adalah PIC Teknis — PICTEKNIK, dibawa `.CAUSE_OF_LOSS`.
	TechnicalPIC string

	// GroupPanel adalah lini bisnis — GROUPPANEL, dibawa `.Initial`.
	//
	// Ia tidak ditampilkan di grid, tetapi IKUT TERSIMPAN ke baris arsip: kolom
	// GROUPPANEL itulah yang kemudian menentukan siapa melihat barisnya pada daftar
	// kirim ke cabang.
	GroupPanel string
}

// DocumentTypeOption adalah satu pilihan Tipe Dokumen — POOLDATA.V_LST_DOC_TYPE.
type DocumentTypeOption struct {
	Code string
	Name string
}

// DocumentKindOption adalah satu pilihan Jenis Dokumen —
// POOLDATA.V_LST_DET_TYPE_DOC, yang selalu bergantung pada satu Tipe Dokumen.
type DocumentKindOption struct {
	Code string
	Name string

	// TypeCode adalah DOC_TYPE_ID pemiliknya. Ia dibawa supaya layar dapat menyaring
	// pilihan jenis mengikuti tipe yang sedang dipilih tanpa menembak server lagi.
	TypeCode string
}

// FillingCodeOption adalah satu baris pada pemilih Kode Filling.
//
// # Dari mana daftarnya, dan apa yang belum diketahui
//
// Pemilih "Pilih Kode" di sistem lama membuka `SecCariKodeArchiveDoc`
// (`Section/KodeArchiveDoc-Section.xml`) dengan tiga kolom — Kode Archive, Desc Archive,
// dan Nama Box — yang diisi activity `SetKodeandSearchArchiveDoc`.
//
// Activity itu TIDAK ADA di export, dan tidak ada satu pun tabel master kode arsip di
// seluruh 2.634 berkas (`R-16`). Yang dapat dibaca hanyalah bentuk layarnya.
//
// Daftar di sini karena itu REKONSTRUKSI: ia dikumpulkan dari kode filling yang sudah
// benar-benar dipakai baris T_CLAIM_ARCHIVE_FILE. Dua hal membuat rekonstruksi ini
// masuk akal alih-alih tebakan buta — layar lama menyediakan tombol "Input Kode" dan
// "Generated Kode" di samping "Cari Kode", yang berarti kode memang dapat DIBUAT dari
// layar ini, bukan hanya dipilih dari master tetap.
//
// Konsekuensinya disebut terang: kode yang belum pernah dipakai tidak akan muncul di
// daftar, dan karena itu isian Kode Filling tetap dapat diketik langsung. Permintaan
// artefaknya tercatat di docs/permintaan-artefak-pega.md.
type FillingCodeOption struct {
	// Code adalah Kode Archive — kolom KODEFILLING.
	Code string

	// BoxName adalah Nama Box — kolom NAMABOX yang paling sering dipakai bersama kode
	// ini.
	BoxName string

	// UsageCount adalah berapa berkas sudah memakai kode ini.
	//
	// Ia menggantikan kolom "Desc Archive" yang di sistem lama diisi master yang tidak
	// kita miliki. Menampilkan kolom kosong akan membuat pengguna menduga datanya
	// hilang; menampilkan jumlah pemakaian memberi keterangan yang benar-benar ada.
	UsageCount int
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa modul ini memaginasi sementara sistem lama tidak
//
// Kueri `SearchDataArchiveFillingCase` tidak menyetel MaxRecords, sehingga seluruh baris
// yang cocok ditarik sekaligus ke halaman klipboard. Pencarian rentang tanggal yang lebar
// karena itu menarik seluruh isi tabel arsip.
//
// Paginasi di sini adalah **perubahan perilaku yang disadari**, bukan pemeliharaan —
// `09-DATABASE-STRATEGY.md` §6.3 menyatakannya harus diuji per layar.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi. MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih
// besar DITOLAK menjadi batas, bukan dipenuhi diam-diam.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari
// parameter query yang mudah salah ketik, dan menolak seluruh permintaan karena
// `halaman=0` membuat layar gagal tanpa alasan yang terbaca pengguna.
func (p Pagination) Normalize() Pagination {
	clean := p
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

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// ArchivePage adalah satu halaman grid ARCHIVE FILE KLAIM beserta jumlah seluruh
// barisnya.
//
// Keduanya dikembalikan bersama, bukan lewat dua panggilan terpisah, supaya angka
// "menampilkan 21–40 dari 57" tidak dapat berasal dari dua saat yang berbeda.
type ArchivePage struct {
	Files      []ArchiveFile
	Total      int
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p ArchivePage) TotalPages() int {
	size := p.Pagination.Normalize().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang diketik saat masuk. Ia yang tersimpan ke kolom
	// USERINPUT, sama seperti `OperatorID.pyUserIdentifier` di sistem lama.
	Login string

	// Position adalah jabatan petugas — `OperatorID.pyPosition` di sistem lama.
	//
	// Ia MENENTUKAN baris mana yang tampak di daftar kirim ke cabang. Lihat
	// BranchScopeFor.
	Position string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{
		Login:    strings.TrimSpace(c.Login),
		Position: strings.TrimSpace(c.Position),
	}
}

// Repo adalah seam ke penyimpanan arsip dokumen SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut
// entitas, dan memang tidak boleh ada.
type Repo interface {
	// Search membaca satu halaman grid ARCHIVE FILE KLAIM.
	Search(ctx context.Context, criteria Criteria, page Pagination) (ArchivePage, error)

	// SearchClaims mencari calon klaim yang berkasnya hendak diarsipkan.
	SearchClaims(ctx context.Context, criteria ClaimCriteria) ([]ClaimCandidate, error)

	// Save menyimpan satu berkas arsip — menyisipkan bila baru, mengubah bila sudah ada.
	//
	// Ia mengembalikan ID baris yang tersimpan. Pada penyisipan, ID itulah yang baru
	// diterbitkan; pada pengubahan, ID yang sama dengan yang diminta.
	Save(ctx context.Context, draft Draft) (int64, error)

	// DocumentTypes membaca pilihan Tipe Dokumen.
	DocumentTypes(ctx context.Context) ([]DocumentTypeOption, error)

	// DocumentKinds membaca pilihan Jenis Dokumen beserta tipe pemiliknya.
	DocumentKinds(ctx context.Context) ([]DocumentKindOption, error)

	// FillingCodes membaca daftar Kode Filling yang sudah dipakai, disaring teks
	// pencarian. Lihat FillingCodeOption untuk asal daftarnya.
	FillingCodes(ctx context.Context, keyword string) ([]FillingCodeOption, error)

	// PendingBranch membaca berkas yang belum dikirim ke layanan Arsip, disaring
	// menurut lini bisnis yang boleh dilihat pemanggil.
	PendingBranch(ctx context.Context, scope BranchScope, page Pagination) (ArchivePage, error)

	// FindByID membaca satu berkas arsip. Nilai kedua bernilai false bila barisnya tidak
	// ada.
	FindByID(ctx context.Context, id int64) (ArchiveFile, bool, error)

	// MarkSent menyimpan jawaban layanan Arsip dan menandai barisnya sudah dikirim.
	MarkSent(ctx context.Context, receipt Receipt) error

	// StoreReceipt menyimpan jawaban layanan Arsip TANPA menandai barisnya terkirim.
	//
	// # Kenapa ada dua, dan kenapa keduanya harus tetap berbeda
	//
	// Sistem lama mengirim berkas ke layanan Arsip dari DUA tempat, dan hanya salah
	// satunya menandai `CABANGSTATUS`:
	//
	//	SaveAttachArchiveToDatabase langkah 5   kirim, simpan jawaban, TIDAK menandai
	//	SENDDATACABANGKEARCHIVE                 kirim, simpan jawaban, MENANDAI '1'
	//
	// Akibatnya berkas yang sama dikirim dua kali: sekali saat disimpan, sekali lagi
	// dari layar Dokument Cabang yang masih memuatnya karena statusnya belum berubah.
	//
	// Work Owner memutuskan 2026-09-25 perilaku itu **direplikasi apa adanya**. Menyatukan
	// keduanya menjadi satu operasi akan menghapus pengiriman kedua — dan itu perubahan
	// perilaku, bukan pembersihan.
	StoreReceipt(ctx context.Context, receipt Receipt) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti mengarsipkan berkas satu
// badan hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
