// Package inboxpladlapredla adalah inti modul Inbox PLA, DLA, Pre DLA.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan tertulis di masternya sendiri:
// `Database/m_menu_aplikasi_pnc.csv` baris `MENU_ID 44` berbunyi **"Inbox PLA, DLA,
// Pre DLA"** dengan `MENU_PROGRAM` `InboxPLA_harness`. `D-81` menetapkan nama modul
// mengikuti nama yang dipakai Work Owner, bukan nama yang dikarang.
//
// Ia BERBEDA dari `MENU_ID 45` "Inbox PLA DLA" (`MENU_PROGRAM` `InboxPLADLA`), yang
// harness dan pembacanya lain sama sekali — lihat paket `inboxpladla`. Keduanya sengaja
// tidak disatukan: yang ini antrean petugas internal, yang itu layar milik reasuradur.
//
// # Artefak Pega yang dibaca
//
//	Harness/InboxPLA_harness-Harness.xml        rangka layar, tiga tab
//	Section/InboxPLADLA_sect-Section.xml        wadah tab: PLA · DLA · Pre DLA
//	Section/InboxPLA_sect-Section.xml           isi tab PLA
//	Section/InboxDLA_sect-Section.xml           isi tab DLA
//	Section/PNCInboxPreDLA_sect-Section.xml     isi tab Pre DLA
//	Activity/GetSearchPNCList_PLA-Act.xml       pemuat daftar PLA + penyaringnya
//	Activity/GetSearchPNCList_DLA-Act.xml       pemuat daftar DLA + penyaringnya
//	Activity/GetPNCList_PreDLA-Act.xml          pemuat daftar Pre DLA + penyaringnya
//	Activity/UpdateDetailPLA2-Act.xml           tombol Send pada tab PLA
//	Activity/UpdateDetailDLA2-Act.xml           tombol Send pada tab DLA
//	RDB List/GetPNCList_PLA-SQL.xml             daftar PLA
//	RDB List/GetPNCList_DLA-SQL.xml             daftar DLA
//	RDB List/GetPNCList_PreDLA-SQL.xml          daftar Pre DLA
//	RDB List/CountPNCList_{PLA,DLA,PreDLA}      pencacah masing-masing
//	RDB List/GetPLAList-SQL.xml                 grid "Detail PLA List"
//	RDB List/GetDLAList-SQL.xml                 grid "Detail DLA List"
//	RDB List/UpdatePLAList-SQL.xml              penanda terkirim PLA
//	RDB List/UpdateDLAList-SQL.xml              penanda terkirim DLA
//
// # Apa yang dikerjakan layar ini
//
// `CONTEXT.md` menyebut ketiga dokumennya berurutan, dan urutan itulah yang menjelaskan
// mengapa tabnya tiga:
//
//	PLA      Preliminary Loss Advice — memberitahukan nilai ESTIMASI klaim
//	Pre-DLA  memberitahukan nilai yang AKAN diakseptasi, sebelum akseptasi
//	DLA      Definite Loss Advice — memberitahukan nilai AKSEPTASI
//
// Ketiganya ditujukan kepada koasuransi/reasuransi. Layar ini adalah antrean **dokumen
// yang sudah terbit tetapi BELUM dikirim**, dan itulah penyaring yang sama di ketiga
// tabnya — bukan penyaring status klaim.
//
// # INI BENAR-BENAR INBOX MENURUT KEEMPAT CIRI `D-79`
//
// Barisnya pekerjaan (dokumen menunggu dikirim) · baris HILANG setelah dikerjakan
// (`ISKIRIM` menjadi `'1'`) · barisnya punya tenggat (klaim menunggu pemberitahuan ke
// reasuradur) · dan daftarnya adalah antrean, bukan data acuan. Ia bukan layar master.
//
// # PENYARING "BELUM TERKIRIM" BERBEDA DI TAB PRE DLA, DAN PERBEDAANNYA BUKAN KELALAIAN
//
//	tab PLA      T_PLALIST     (ISKIRIM IS NULL OR ISKIRIM = '0')  DAN REINSCODE IS NOT NULL
//	tab DLA      T_DLALIST     (ISKIRIM IS NULL OR ISKIRIM = '0')
//	tab Pre DLA  T_PREDLALIST  NOAKSEP IS NULL
//
// Tab Pre DLA TIDAK memeriksa `ISKIRIM` sama sekali. Yang membuat sebuah Pre-DLA keluar
// dari antrean adalah terbitnya **Nomor Akseptasi** — dan itu memang benar secara bisnis:
// Pre-DLA memberitahukan nilai yang akan diaksep, sehingga ia kehilangan gunanya begitu
// akseptasinya terbit, bukan begitu suratnya terkirim.
//
// Menyeragamkan ketiganya akan mengubah isi antrean tanpa satu pun galat. Ia dibawa apa
// adanya (`P-5`).
//
// # LINGKUP BARIS: KETIGANYA MENGECUALIKAN PA DAN TRAVEL, DLA MENGECUALIKAN DUA HAL LAGI
//
//	ketiga tab   GROUPPANEL != '002' (Personal Accident) dan != '005' (Travel)
//	tab DLA      ditambah BRANCHNAME != 'ASNET'
//	tab DLA      ditambah tidak ada di BUSINESSGROUPID '10008'
//
// Kedua penyaring tambahan pada DLA tidak punya pasangan di tab lain. Keduanya dibawa apa
// adanya; alasan bisnisnya tidak tertulis di mana pun di export.
//
// # ALIAS KOLOM DI LAYAR INI MENYESATKAN — DAN TUJUH DARI DELAPAN TIDAK MENYATAKAN ISINYA
//
// Ketiga kueri daftar mengalihnamakan kolomnya menjadi nama properti kelas
// `ASM-FW-GCNMFW-Int-V_POLIS` yang sudah ada — utang teknis §4.2 pada
// `03-CURRENT-ARCHITECTURE.md`, yang `D-19` larang dibawa:
//
//	kolom sebenarnya    alias Pega        arti sesungguhnya
//	------------------- ----------------- ----------------------------------
//	b.CLAIMID           "BRANCH_NAME"     kunci kerja Pega   (!)
//	b.CLAIMNO           "BRANCH_CODE"     nomor klaim        (!)
//	b.NOPOLIS           "POLICY_NO"       nomor polis        — satu-satunya yang benar
//	b.QQNAME            "CUSTOMER"        nama tertanggung
//	b.REGISTERDATE      "START_DATE"      tanggal registrasi
//	b.DATEOFLOSS        "END_DATE"        tanggal kejadian   (!)
//	b.PICTEKNIK         "BUSINESS_NAME"   PIC Teknik         (!)
//	MAX(TGLPLA/TGLDLA)  "pyCreateDate"    tanggal advice     (!)
//
// Empat tanda (!) itu adalah tempat paling mudah salah: `"BRANCH_NAME"` dan
// `"BRANCH_CODE"` berdampingan padahal tidak satu pun menyangkut cabang, dan
// `"END_DATE"` berarti tanggal kejadian — bukan tanggal berakhirnya apa pun.
//
// Nama di berkas ini menyebut ISINYA. Penelusuran balik ke export ditempuh lewat nama
// tabel dan kolom sebenarnya, disebut pada setiap isian di bawah.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxpladlapredla

import (
	"context"
	"errors"
	"strings"
)

// DateLayout adalah bentuk tanggal yang dibawa modul ini ke lapisan transport.
//
// # Kenapa tanggal dibawa sebagai TEKS, bukan sebagai time.Time
//
// Karena tidak satu pun tanggal di layar ini dihitung — semuanya hanya digambar. Kolom
// tanggal advice bahkan tidak selalu ada isinya (lihat Row.AdviceDate), dan `time.Time`
// kosong terlihat sebagai `0001-01-01` di setiap tempat yang lupa memeriksanya.
//
// Yang MEMBANDINGKAN tanggal adalah penyaring rentang, dan di sana ia memang `time.Time`
// — lihat Query.From. Perbedaan keduanya disengaja: yang dibandingkan bertipe waktu, yang
// digambar bertipe teks.
//
// Bentuknya `YYYY-MM-DD`, bukan `dd/mm/yyyy` seperti Pega. Layar yang memformatnya ke
// bentuk Indonesia, dan konversi zona waktunya terjadi di sana — satu-satunya tempat yang
// boleh melakukannya (`08-TECHNICAL-STRATEGY.md` §4.4).
const DateLayout = "2006-01-02"

// Row adalah satu baris grid antrean, sama bentuknya di ketiga tab.
//
// # Kenapa SATU tipe untuk tiga tab
//
// Karena ketiga kueri daftar mengembalikan kolom yang SAMA PERSIS — hanya tabel yang
// di-EXISTS-kan dan penyaringnya yang berbeda. Membuat tiga tipe akan memaksa tiga jalur
// di seam Repo dan tiga penyusun DTO untuk data yang bentuknya identik.
//
// Itu berbeda dari grid rincian di bawahnya, yang memang berbeda kolom per tab — lihat
// Document.
type Row struct {
	// ClaimKey adalah `T_CLAIM_PNC.CLAIMID`, beralias `"BRANCH_NAME"` di kueri lama.
	//
	// Ia BUKAN nama cabang dan bukan nomor klaim: ia kunci objek kerja Pega, berbentuk
	// `ASM-FW-GCNMFW-WORK PNC-xxxx` — utang teknis §4.1 yang `D-22` dan `D-71` hapus untuk
	// klaim baru.
	//
	// Ia dibawa karena grid rincian dicari DENGANNYA (`GetPLAList … where claimid = …`),
	// bukan dengan nomor klaim. Ia tidak digambar sebagai kolom.
	ClaimKey string

	// ClaimNo — kolom **"No Klaim"** <- `T_CLAIM_PNC.CLAIMNO`, beralias `"BRANCH_CODE"`.
	ClaimNo string

	// PolicyNo — kolom **"No Polis"** <- `T_CLAIM_PNC.NOPOLIS`, beralias `"POLICY_NO"`.
	//
	// Satu-satunya alias di layar ini yang menyatakan isinya.
	PolicyNo string

	// Insured — kolom **"Nama Tertanggung"** <- `T_CLAIM_PNC.QQNAME`, beralias
	// `"CUSTOMER"`.
	Insured string

	// RegisterDate — kolom **"Tanggal Register"** <- `T_CLAIM_PNC.REGISTERDATE`,
	// beralias `"START_DATE"`. Ia pula kolom pengurut ketiga daftar.
	RegisterDate string

	// LossDate — kolom **"Tanggal Kejadian"** <- `T_CLAIM_PNC.DATEOFLOSS`, beralias
	// `"END_DATE"`.
	LossDate string

	// PICTeknik — kolom **"PIC Teknik"** <- `T_CLAIM_PNC.PICTEKNIK`, beralias
	// `"BUSINESS_NAME"`.
	//
	// Ia DIAMBIL ketiga kueri tetapi TIDAK digambar di grid Pega mana pun — ketiga grid
	// hanya menggambar tujuh kolom, dan ini yang kedelapan. Ia tetap dibawa dan
	// DIGAMBAR di sini, karena antrean yang tidak menyebutkan penanggung jawabnya
	// memaksa petugas membuka klaimnya satu per satu untuk tahu itu pekerjaan siapa.
	//
	// Penambahan kolom adalah selisih terhadap layar lama; ia dinyatakan di
	// PlannedDifferences, bukan disamarkan.
	PICTeknik string

	// AdviceDate — kolom **"Tanggal PLA"** / **"Tanggal DLA"** / **"Tanggal Pre DLA"**,
	// beralias `"pyCreateDate"`.
	//
	// Isinya tanggal dokumen TERBARU yang belum terkirim milik klaim itu:
	//
	//	tab PLA      MAX(T_PLALIST.TGLPLA)     yang ISKIRIM IS NULL
	//	tab DLA      MAX(T_DLALIST.TGLDLA)     yang ISKIRIM IS NULL
	//	tab Pre DLA  MAX(T_PREDLALIST.TGLDLA)  yang ISKIRIM IS NULL
	//
	// Kueri lama menempuhnya dengan `ORDER BY … DESC FETCH NEXT 1 ROW ONLY`; di sini ia
	// `MAX(...)`. Keduanya menghasilkan nilai yang sama dan `MAX` tidak menuntut
	// pengurutan sub-kueri per baris.
	//
	// Perhatikan SATU KEJANGGALAN yang dibawa apa adanya: sub-kueri ini menyaring
	// `ISKIRIM IS NULL` saja, sementara penyaring keanggotaan barisnya menerima
	// `ISKIRIM IS NULL OR ISKIRIM = '0'`. Klaim yang SELURUH dokumennya ber-`ISKIRIM='0'`
	// karena itu tetap muncul di daftar, tetapi kolom tanggalnya KOSONG. Itu bukan
	// kerusakan sistem baru — itu perilaku Pega (`P-5`).
	//
	// Pada tab Pre DLA kejanggalannya lebih jauh lagi: keanggotaan barisnya ditentukan
	// `NOAKSEP IS NULL` — yang tidak menyinggung `ISKIRIM` sama sekali — sedangkan
	// kolom tanggalnya tetap menyaring `ISKIRIM IS NULL`. Keduanya dibawa apa adanya.
	AdviceDate string
}

// Document adalah satu baris grid rincian di bawah antrean.
//
// Judulnya **"Detail PLA List"** pada tab PLA dan **"Detail DLA List"** pada tab DLA. Tab
// Pre DLA TIDAK punya grid ini di Pega; lihat Tab.HasDocuments.
//
// # Kenapa SATU tipe untuk dua grid yang kolomnya berbeda
//
// Karena keduanya berbeda hanya pada DUA isian — `REVISI` yang hanya ada di PLA dan
// `NOAKSEP` yang hanya ada di DLA — dan yang menentukan kolom mana digambar adalah
// Tab.DocumentColumns, bukan tipe barisnya. Preseden yang sama dipakai modul Inbox
// Salvage untuk tiga keluarga kueri sekaligus.
type Document struct {
	// AdviceNo — kolom **"No PLA"** / **"No DLA"**.
	//
	//	PLA  `T_PLALIST.NOPLA`  beralias `"BUSINESS_CODE"`
	//	DLA  `T_DLALIST.NODLA`  beralias `"BUSINESS_CODE"`
	AdviceNo string

	// Reinsurer — kolom **"Reasuradur"**.
	//
	//	PLA  `T_PLALIST.PLAREINSURER`  beralias `"BUSINESS_NAME"`
	//	DLA  `T_DLALIST.DLAREINSURER`  beralias `"BUSINESS_NAME"`
	Reinsurer string

	// AdviceType — kolom **"Tipe"** <- `TIPEPLA` / `TIPEDLA`, beralias `"CURRENCY"`.
	//
	// Aliasnya menyesatkan: ia BUKAN mata uang. Menyalinnya akan menampilkan tipe
	// dokumen di bawah judul "Currency" tanpa satu pun galat.
	AdviceType string

	// Revision — kolom **"Revisi"** <- `T_PLALIST.REVISI`, beralias `"TSI"`.
	//
	// **PLA saja.** `GetDLAList` tidak mengambilnya sama sekali. Aliasnya `"TSI"` —
	// nama yang di seluruh modul lain berarti nilai pertanggungan.
	Revision string

	// AdviceDate — kolom **"Tanggal PLA"** / **"Tanggal DLA"** <- `TGLPLA` / `TGLDLA`,
	// beralias `"START_DATE"`.
	AdviceDate string

	// Sent — kolom **"Terkirim"** <- `ISKIRIM`, beralias `"MARKETING"`.
	//
	// Nilainya `'1'` bila sudah dikirim; `NULL` atau `'0'` bila belum. Ia dibawa apa
	// adanya sebagai teks, bukan diubah menjadi boolean: `NULL` dan `'0'` berarti hal
	// yang sama bagi penyaring daftar, tetapi keduanya benar-benar ada di data dan
	// membedakannya membantu menelusuri baris yang tanggalnya kosong (lihat
	// Row.AdviceDate).
	Sent string

	// SentDate — kolom **"Tanggal Kirim"** <- `TGLKIRIM`, beralias `"END_DATE"`.
	SentDate string

	// ReceivedDate — kolom **"Tanggal Terima"** <- `TGLTERIMAPLA` / `TGLTERIMADLA`,
	// beralias `"POLICY_NO"`.
	//
	// Alias yang paling berbahaya di grid ini: `"POLICY_NO"` berarti nomor polis di
	// SELURUH modul lain, dan di sini ia tanggal.
	ReceivedDate string

	// Notes — kolom **"Catatan"** <- `NOTES`, beralias `"CUSTOMER"`.
	Notes string

	// Email — kolom **"Email"** <- `EMAILPLA` / `EMAILDLA`, beralias `"BRANCH_NAME"`.
	//
	// Ia diisi tombol Send bersamaan dengan `ISKIRIM`, sehingga baris yang belum
	// terkirim selalu kosong di kolom ini.
	Email string

	// AcceptanceNo — kolom **"No Akseptasi"** <- `T_DLALIST.NOAKSEP`, beralias `"pyID"`.
	//
	// **DLA saja.** Di tab Pre DLA kolom yang sama justru menjadi PENYARING keanggotaan
	// daftar (`NOAKSEP IS NULL`), bukan kolom yang digambar.
	AcceptanceNo string
}

// Caller adalah identitas pemanggil.
//
// # Ia TIDAK menyaring di layar ini
//
// Ketiga daftar bersama: tidak satu pun kuerinya menyebut pemanggil. Itu perilaku Pega —
// `GetSearchPNCList_PLA`, `_DLA`, dan `GetPNCList_PreDLA` hanya menyaring lini bisnis,
// keberadaan dokumen, tanggal, dan nomor klaim.
//
// Ia tetap dituntut ada karena SETIAP pembukaan dicatat. Barisnya memuat nama
// tertanggung dan nomor polis — data nasabah — dan selama pemeriksaan peran belum ada
// (`TKT-F3-004`), jejak inilah satu-satunya yang menyatakan siapa yang membukanya
// (`D-59`).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	Login string
}

// Clean memangkas spasi di ujung identitas pemanggil.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Pagination adalah permintaan satu halaman.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// DefaultPageSize **10** adalah angka layar ini sendiri: `<pyRDLPageSize>10</pyRDLPageSize>`
// pada ketiga section tabnya, dan `.PageSize = 10` pada `GetSearchPNCList_PLA`. Modul inbox
// lain memakai 20 dan 50; perbedaannya tidak diseragamkan (`D-13`).
//
// MaxPageSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam.
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari
// parameter query yang mudah salah ketik, dan menolak seluruh permintaan karena
// `halaman=0` akan membuat layar gagal tanpa alasan yang terbaca pengguna.
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

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Items []Row

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
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

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat kepala berkas .sql-nya.
func Slice(all []Row, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []Row{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Items = all[offset:end]
	return result
}

// ErrRowNotFound berarti klaim yang dimintakan rinciannya tidak ada pada entitas ini.
//
// Ia dipisahkan dari "klaim ada tetapi belum punya dokumen": yang kedua BUKAN galat dan
// menghasilkan daftar kosong, sedangkan yang pertama menandakan kunci yang salah — atau
// kunci yang benar dibuka pada portal yang keliru (`R-20`).
var ErrRowNotFound = errors.New("inboxpladlapredla: klaim tidak ditemukan")

// PlannedDifferences adalah selisih terhadap layar Pega yang sudah diputuskan.
//
// Ia dikirim ke layar dan digambar di kakinya. Alasannya bukan kejujuran demi kejujuran:
// petugas yang membandingkan layar ini dengan Pega akan menemukan selisihnya, dan selisih
// yang tidak dinyatakan akan dilaporkan sebagai kerusakan — lalu ditelusuri ulang oleh
// orang yang tidak tahu bahwa ia disengaja.
var PlannedDifferences = []string{
	"Tombol \"Send\" belum tersedia. Di Pega ia mengirim surat PLA/DLA beserta " +
		"lampirannya ke reasuradur lewat email, memperbarui master reasuransi, " +
		"menyisipkan dokumennya, lalu menandai dokumen sebagai terkirim. Tiga yang " +
		"pertama menembak sistem di luar basis data ini dan belum dibangun. " +
		"Menandainya terkirim tanpa mengirimnya akan membuat baris HILANG dari antrean " +
		"padahal tidak satu pun surat sampai — karena itu tidak dikerjakan sebagian.",

	"Tombol \"Upload File Penunjang\" belum tersedia. Ia menyimpan lampiran ke " +
		"penyimpanan dokumen internal (`D-16`), yang belum tersambung.",

	"Tombol \"Print Pre DLA\" belum tersedia. Ia membangkitkan berkas PDF, dan mesin " +
		"dokumen (`D-11`) belum dibangun.",

	"Ekspor tab Pre DLA MENGIKUTI penyaring yang sedang aktif. Di Pega " +
		"`GetExportDataPreDLA` tidak menyaring apa pun — ia menarik seluruh isi " +
		"T_PREDLALIST, mengabaikan Dari/Sampai/No Klaim yang sedang terisi. Perilaku itu " +
		"tidak dibawa: `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 melarang ekspor tanpa " +
		"batas atas tabel berisi puluhan juta baris, dan berkas yang isinya berbeda dari " +
		"layar di atasnya lebih menyesatkan daripada berguna.",

	"Kolom \"PIC Teknik\" DITAMBAHKAN. Ketiga kueri Pega mengambilnya " +
		"(`b.picteknik as BUSINESS_NAME`) tetapi tidak satu pun grid menggambarnya. " +
		"Tanpa kolom itu, antrean tidak menyebutkan pekerjaan ini milik siapa.",

	"Pencarian TIDAK peka huruf besar-kecil. Kueri lama membandingkan apa adanya, " +
		"sehingga nomor klaim yang diketik huruf kecil di Pega tidak menemukan apa-apa.",

	"Rentang tanggal yang DIKOSONGKAN berarti tidak menyaring. Di Pega isian kosong " +
		"tetap dirangkai menjadi `to_date('','dd/mm/yyyy')`, yang ditolak Oracle.",

	"Paginasi dikerjakan basis data (`OFFSET … FETCH NEXT`), bukan dengan menyisipkan " +
		"batas baris sebagai TEKS ke dalam kueri seperti pola `{ASIS:Pagination.FirstRow}` " +
		"di sistem lama.",
}

// Repo adalah seam ke penyimpanan, dideklarasikan di sini karena di sinilah ia DIPAKAI
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 2).
//
// Dua pengisi: SQL terhadap Oracle, dan penyimpanan memori untuk pengembangan lokal serta
// uji aturan bisnis tanpa infrastruktur. Dua pengisi nyata itulah yang membuat seam ini
// seam, bukan abstraksi hipotetis.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// Documents mengembalikan isi grid "Detail PLA List" / "Detail DLA List" satu klaim.
	//
	// Kuncinya `T_CLAIM_PNC.CLAIMID` — parameter yang sama dengan
	// `TempPLA.ClaimData.CaseID` di sistem lama, dan BUKAN nomor klaim.
	//
	// Klaim yang tidak ada menghasilkan ErrRowNotFound. Klaim yang ada tetapi belum punya
	// dokumen menghasilkan daftar kosong tanpa galat — keduanya terlihat sama di layar,
	// dan hanya yang pertama yang merupakan kekeliruan.
	Documents(ctx context.Context, tab Tab, claimKey string) ([]Document, error)
}

// RepoSelector memilih penyimpanan milik satu portal entitas.
//
// Ia ada karena `D-75` menetapkan SATU BASIS DATA PER ENTITAS. Alias portal yang tidak
// dikenal menghasilkan galat, bukan jatuh ke koneksi bawaan — jatuh ke bawaan berarti
// menampilkan klaim satu badan hukum kepada petugas badan hukum lain tanpa satu pun
// pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
