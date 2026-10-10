// Package inboxpladla adalah inti modul Inbox PLA DLA — layar milik REASURADUR.
//
// # Nama modul ini
//
// Diambil dari masternya sendiri: `Database/m_menu_aplikasi_pnc.csv` baris `MENU_ID 45`
// berbunyi **"Inbox PLA DLA"** dengan `MENU_PROGRAM` `InboxPLADLA`. Nama menu dan nama
// harness kebetulan sama, sehingga tidak ada yang perlu dikarang (`D-81`).
//
// # INI BUKAN LAYAR YANG SAMA DENGAN `MENU_ID 44`
//
// Keduanya bersebelahan di menu, judulnya hampir sama, dan keduanya menyebut PLA dan DLA.
// Yang membedakannya adalah SIAPA yang membacanya, dan perbedaan itu menentukan segalanya:
//
//	MENU_ID 44  "Inbox PLA, DLA, Pre DLA"  petugas INTERNAL
//	            dokumen yang BELUM dikirim — antrean pekerjaan mengirim
//
//	MENU_ID 45  "Inbox PLA DLA"            REASURADUR
//	            dokumen yang SUDAH dikirim kepadanya — antrean pekerjaan menanggapi
//
// Penyaringnya berlawanan arah: yang satu `ISKIRIM IS NULL`, yang lain `ISKIRIM = '1'`.
// Menyatukan keduanya akan menggabungkan dua antrean yang isinya justru saling
// meniadakan. Lihat paket `inboxpladlapredla` untuk yang pertama.
//
// # Artefak Pega yang dibaca
//
//	Harness/InboxPLADLA-Harness.xml             rangka layar
//	Section/InboxDLAReas_sect-Section.xml       isi layar — 4 grid + 1 bagan
//	Activity/SetDataPLADLA-Act.xml              pemuat keenam daftar
//	Activity/GetDataDLAReas_Act-Act.xml         rincian DLA satu klaim
//	Activity/ExportDataPLADLAReas-Act.xml       tombol Export To Excel
//	RDB List/GetPNCList_PLA1-SQL.xml            daftar PLA
//	RDB List/GetPNCList_PLADLA-SQL.xml          daftar DLA
//	RDB List/GetPNCList_PLADLAClose-SQL.xml     daftar Close
//	RDB List/GetDLAListReas-SQL.xml             rincian DLA
//
// # SIAPA PEMANGGILNYA MENENTUKAN SELURUH ISI LAYAR
//
// Ketiga kueri daftar menyaring lewat rantai yang sama:
//
//	reinscode ... (SELECT reinsurerid FROM POOLDATA.T_REINSURER WHERE login = <pemanggil>)
//
// Artinya login yang dipakai masuk dicocokkan ke kolom `LOGIN` pada master reasuransi.
// Pengguna INTERNAL — yang loginnya tidak ada di sana — melihat layar KOSONG, dan itu
// bukan kerusakan melainkan perilaku yang benar. Modul `masterreas` sudah mencatat hal
// yang sama: "LOGIN … lima kueri inbox menyaring klaim dengannya".
//
// Ini satu-satunya layar yang sudah dibangun tempat identitas pemanggil MENYARING, bukan
// sekadar dicatat.
//
// # TAMPILAN "DATA PLA DLA XOL KLAIM" TIDAK DIBAWA — IA KODE MATI DI PEGA
//
// Sectionnya memang memuat wadah berjudul itu, dan `SetDataPLADLA` memang punya cabang
// `param.tipe=="7"` yang mengisi `DataKomiteXOL`. Tetapi wadahnya bersyarat:
//
//	Section/InboxDLAReas_sect-Section.xml   pyContainerVisibleWhen = TempView.CityID==7
//
// dan `TempView.CityID` HANYA pernah diisi dari `param.tipe`, yang pada gilirannya hanya
// pernah berisi `.CityID` sebuah baris tabel "Status / Jumlah" atau `TempView.CityID`
// sendiri. Literal `7` nol kemunculan sebagai nilai `tipe` di seluruh export, dan tabel
// ringkasnya berisi ENAM baris.
//
// Artinya tampilan itu tidak pernah dapat tergambar di Pega. Ia sempat dibangun di sini
// sebagai tab ketujuh lalu sebagai panel permanen, dan keduanya memberi akses kepada
// sesuatu yang layar lama tidak pernah tampilkan; dicabut 2026-10-09 atas keputusan Work
// Owner.
//
// Satu alasan tambahan yang membuat pencabutannya bukan kerugian: kueri XOL-nya di Pega
// TIDAK memakai login pemanggil melainkan nilai yang ditulis tetap di dalam activity —
//
//	Activity/SetDataPLADLA-Act.xml   Local.loginreas = <satu login mitra, ditulis tetap>
//
// diberikan tepat sekali dan tidak pernah ditimpa. Seandainya tampilannya dapat dibuka,
// setiap mitra akan melihat ringkasan XOL milik mitra itu, bukan miliknya sendiri.
//
// # DUA KUERI MEMILIH SATU KODE REASURADUR, SATU KUERI MEMILIH SEMUANYA
//
// Perbedaan yang mudah terlewat, dan ia dibawa apa adanya (`P-5`):
//
//	GetPNCList_PLA1       reinscode =  (SELECT … ORDER BY reinsurerid DESC FETCH 1)
//	GetPNCList_PLADLA     reinscode =  (SELECT … ORDER BY reinsurerid DESC FETCH 1)
//	GetPNCList_PLADLAClose reinscode IN (SELECT …)          <- SELURUHNYA
//
// Bila satu login memetakan ke lebih dari satu `REINSURERID`, tab Close menampilkan klaim
// dari SEMUA kodenya sedangkan dua tab lain hanya dari kode tertinggi. Apakah itu
// disengaja tidak tertulis di mana pun; menyeragamkannya akan mengubah isi daftar, dan itu
// selisih yang belum diminta siapa pun.
//
// # ALIAS KOLOM DI LAYAR INI MENYESATKAN LEBIH JAUH DARIPADA BIASANYA
//
//	kolom sebenarnya    alias Pega        arti sesungguhnya
//	------------------- ----------------- --------------------------------
//	b.CLAIMID           "TSI"         (!) kunci kerja Pega
//	b.CLAIMNO           "BRANCH_CODE" (!) nomor klaim
//	b.REGISTERDATE      "BUSINESS_NAME"(!) tanggal registrasi
//	b.DATEOFLOSS        "CURRENCY"    (!) tanggal kejadian
//	b.QQNAME            "pyNote"      (!) nama tertanggung
//	b.NOPOLIS           "POLICY_NO"       nomor polis
//	b.BUSINESSNAME      "MARKETING"   (!) nama lini bisnis
//	statusclaim_1       "pyLabel"         kode status klaim
//	b.PICTEKNIK         "BRANCH_NAME" (!) PIC Teknik
//	b.CLOSECLAIMNOTE    "CaseID"      (!) catatan penutupan klaim
//	nopla               "BUSINESS_CODE"(!) nomor PLA
//
// Sepuluh dari sebelas tidak menyatakan isinya, dan `"TSI"` untuk kunci klaim adalah yang
// paling berbahaya: di seluruh modul lain `TSI` berarti nilai pertanggungan. `D-19`
// melarang membawanya; nama di berkas ini menyebut isinya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxpladla

import (
	"context"
	"errors"
	"strings"
)

// DateLayout adalah bentuk tanggal yang dibawa modul ini ke lapisan transport.
//
// Tanggal dibawa sebagai TEKS karena tidak satu pun dihitung — semuanya hanya digambar.
// Layar yang memformatnya ke bentuk Indonesia, dan konversi zona waktunya terjadi di sana
// (`08-TECHNICAL-STRATEGY.md` §4.4).
const DateLayout = "2006-01-02"

// Row adalah satu baris daftar klaim, sama bentuknya di ketiga tab.
type Row struct {
	// ClaimKey adalah `T_CLAIM_PNC.CLAIMID`, beralias `"TSI"` di kueri lama.
	//
	// Ia kunci objek kerja Pega, berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`, dan BUKAN nilai
	// pertanggungan seperti yang disiratkan aliasnya. Ia tidak digambar sebagai kolom;
	// yang memakainya adalah tombol rincian.
	ClaimKey string

	// ClaimNo — kolom **"No Klaim"** <- `CLAIMNO`, beralias `"BRANCH_CODE"`.
	ClaimNo string

	// PolicyNo — kolom **"No Polis"** <- `NOPOLIS`, beralias `"POLICY_NO"`.
	PolicyNo string

	// Insured — kolom **"Nama Tertanggung"** <- `QQNAME`, beralias `"pyNote"`.
	Insured string

	// BusinessName — kolom **"Bisnis"** <- `BUSINESSNAME`, beralias `"MARKETING"`.
	BusinessName string

	// RegisterDate — kolom **"Tanggal Register"** <- `REGISTERDATE`, beralias
	// `"BUSINESS_NAME"`.
	RegisterDate string

	// LossDate — kolom **"Tanggal Kejadian"** <- `DATEOFLOSS`, beralias `"CURRENCY"`.
	LossDate string

	// PICTeknik — kolom **"PIC Teknik"** <- `PICTEKNIK`, beralias `"BRANCH_NAME"`.
	PICTeknik string

	// StatusCode — kolom **"Status"** <- `T_CLAIM_PNC.STATUSCLAIM` (di Pega
	// `PC_ASM_FW_GCNMFW_WORK.STATUSCLAIM_1`, beralias `"pyLabel"`; tabel kerja itu sudah tidak
	// dipakai sejak 2026-10-08).
	//
	// Pada tab DLA ia DIGANTI `1139` ketika `ISPENDINGCLOSE = 'true'` — klaim yang
	// sebenarnya sudah `Resolved-Completed` tetapi masih menunggu penutupan. Penggantian
	// itu hanya ada di kueri tab DLA; kedua tab lain memakai kode aslinya.
	StatusCode string

	// StatusLabel adalah arti kode itu menurut `POOLDATA.M_STS_CLAIM.LSC_NOTE`.
	//
	// Ia DITAMBAHKAN. Kueri lama hanya mengambil kodenya, sehingga reasuradur membaca
	// angka `1139` alih-alih artinya. `R-06` sudah tertutup — 33 kode beserta artinya
	// diterima — dan menahannya tidak memberi manfaat apa pun.
	//
	// Kode yang tidak ada di master menghasilkan label KOSONG, bukan galat: layar
	// menggambar kodenya, dan itu lebih berguna daripada baris yang hilang.
	StatusLabel string

	// AdviceNo — kolom **"No PLA"** <- sub-kueri ke `T_PLALIST.NOPLA`, beralias
	// `"BUSINESS_CODE"`.
	//
	// Yang diambil adalah PLA dengan `REVISI` tertinggi milik reasuradur pemanggil. Kueri
	// lama memakai `ORDER BY revisi DESC FETCH NEXT 1 ROW ONLY` **tanpa pemutus seri**,
	// sehingga dua PLA berrevisi sama menghasilkan nomor yang tidak ditentukan. Perilaku
	// itu dibawa apa adanya.
	AdviceNo string

	// CloseNote — kolom **"Catatan Tutup Klaim"** <- `CLOSECLAIMNOTE`, beralias
	// `"CaseID"`.
	CloseNote string
}

// StatusCount adalah satu baris tabel ringkas **"Status / Jumlah"** di atas daftar.
//
// # Dari mana ia datang, dan kenapa ia kueri TERSENDIRI di sini
//
// Di Pega ia BUKAN kueri. Ketiga kueri daftarnya tidak berpaginasi sama sekali — tidak
// satu pun memuat `ROWNUM`, `OFFSET`, maupun `FETCH` pada tingkat luar — sehingga seluruh
// baris yang cocok dimuat ke klipboard, dan tabel ringkas beserta bagannya dihitung dari
// senarai yang sudah ada di memori itu.
//
// Di sini daftarnya DIPAGINASI (`15-NFR-PERFORMANCE-SCALABILITY.md` §3.2), sehingga
// menghitungnya dari halaman yang sedang tampil akan menghasilkan angka yang berubah-ubah
// setiap kali pengguna berpindah halaman. Ia karena itu dihitung di basis data, atas
// penyaring yang SAMA dengan daftarnya.
type StatusCount struct {
	// Code adalah kode status, sama isinya dengan Row.StatusCode.
	Code string

	// Label adalah artinya menurut master status. Kosong bila kodenya tidak ada di sana.
	Label string

	// Total adalah jumlah klaim berstatus itu pada daftar yang sedang dibuka.
	Total int
}

// ListCount adalah satu baris tabel **"Status / Jumlah"** di samping daftar.
//
// # Ia mencacah DAFTAR, bukan status klaim
//
// Inilah tabel yang benar-benar digambar Pega, dan isinya bukan status klaim melainkan
// keenam DAFTAR layar ini beserta jumlah klaim di masing-masingnya:
//
//	PLA · PLA & DLA · CLOSE CLAIM · NOT ANSWERED · NOT REPLIED FROM ASM · REPLIED FROM ASM
//
// Terbaca langsung dari `Activity/GetLostAdjuster_act-Act.xml`, yang menyusun barisnya
// satu per satu: `.CauseOfLoss` diisi NAMA DAFTAR (`"NOT ANSWERED"`,
// `"NOT REPLIED FROM ASM"`, `"REPLIED FROM ASM"`, …), `.CityID` kode daftarnya, dan
// `.City` jumlahnya. Aliasnya menyesatkan seluruhnya — tidak satu pun berhubungan dengan
// penyebab kerugian maupun kota.
//
// # Ia BUKAN StatusCount, dan keduanya menjawab pertanyaan yang berbeda
//
// StatusCount mencacah status klaim DI DALAM satu daftar yang sedang terbuka. Pega tidak
// menggambar hal itu di layar ini sama sekali. Keduanya sengaja dibiarkan berdampingan:
// yang satu menjawab "berapa klaim di tiap daftar" dan dipakai sebagai navigasi, yang
// lain menjawab "status apa saja isi daftar ini".
//
// # Kenapa angkanya DAPAT DIKLIK
//
// Karena di Pega pun begitu, dan itulah satu-satunya navigasi layar ini: sel "Jumlah"
// ber-`pyFormat=pxLink` dengan aksi `refresh` yang membawa `.CityID`
// (`Section/InboxDLAReas_sect-Section.xml`). Layar lama TIDAK punya bilah tab.
type ListCount struct {
	// Code adalah kode daftarnya, sama isinya dengan Tab.Code.
	Code string

	// Name adalah nama daftar seperti tertulis di Pega.
	Name string

	// Total adalah jumlah klaim pada daftar itu, mengikuti pencarian yang sedang aktif.
	Total int
}

// Caller adalah identitas pemanggil.
//
// # Ia MENYARING di layar ini, berbeda dari modul inbox lain
//
// Login-nya dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`, dan hasil pencocokan itulah yang
// menentukan klaim mana yang terlihat. Pemanggil yang tidak terdaftar sebagai reasuradur
// melihat daftar kosong — perilaku yang benar, bukan kerusakan.
//
// Karena itu identitas yang tidak terbaca di sini BUKAN sekadar kehilangan jejak: tanpa
// login, penyaringnya tidak dapat disusun sama sekali.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// `GetPNCList_PLA1` mencocokkan `OperatorID.pyUserIdentifier` — pengenal operator,
	// bukan nama lengkap. Memakai NIK di sini akan membuat layar kosong bagi setiap
	// reasuradur.
	Login string

	// Name adalah nama yang dibaca manusia (`OperatorID.pyUserName`).
	//
	// Ia TIDAK menyaring apa pun. Satu-satunya pemakainya adalah balasan komunikasi, yang
	// menulisnya ke `M_KOMUNIKASI_PNC.REPLYFROMNAME` — kolom keterangan, bukan kolom
	// identitas. Karena itu ia boleh kosong, dan operasi baca mana pun tidak menuntutnya.
	//
	// Membedakan keduanya bukan kerapian: `ReplyKomunikasi-SQL.xml` memang mengambil
	// `pyUserIdentifier` untuk satu kolom dan `pyUserName` untuk kolom lain, dan
	// menyamakannya akan membuat kolom nama berisi kode login.
	Name string
}

// Clean memangkas spasi di ujung identitas pemanggil.
func (c Caller) Clean() Caller {
	return Caller{
		Login: strings.TrimSpace(c.Login),
		Name:  strings.TrimSpace(c.Name),
	}
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
// pada `Section/InboxDLAReas_sect-Section.xml`.
//
// Perhatikan bedanya dengan Pega: di sana angka itu memaginasi senarai yang SUDAH seluruhnya
// dimuat ke klipboard, sedangkan di sini ia memotong di basis data. Lihat StatusCount.
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
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
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data.
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

// ErrCallerNotAReinsurer berarti login pemanggil tidak ada di master reasuransi.
//
// # Kenapa ia BUKAN daftar kosong
//
// Karena keduanya berarti hal yang berbeda, dan hanya satu yang dapat ditindaklanjuti
// pengguna:
//
//	terdaftar, tidak ada klaim   tidak ada pekerjaan hari ini — tunggu
//	tidak terdaftar              layar ini memang bukan untuk Anda — hubungi admin
//
// Di Pega keduanya terlihat sama: layar kosong tanpa satu pun keterangan. Petugas internal
// yang tersesat ke menu ini akan menyimpulkan sistemnya rusak, dan reasuradur yang
// loginnya belum didaftarkan akan menunggu pekerjaan yang tidak akan pernah muncul.
var ErrCallerNotAReinsurer = errors.New(
	"inboxpladla: login pemanggil tidak terdaftar sebagai reasuradur")

// ErrRowNotFound berarti klaim yang dimintakan rinciannya tidak ada pada entitas ini.
var ErrRowNotFound = errors.New("inboxpladla: klaim tidak ditemukan")

// PlannedDifferences adalah selisih terhadap layar Pega yang sudah diputuskan.
//
// Ia dikirim ke layar dan digambar di kakinya, bukan hanya tercatat di kode: selisih yang
// tidak dinyatakan akan dilaporkan sebagai kerusakan oleh orang yang membandingkan kedua
// layar berdampingan.
var PlannedDifferences = []string{
	"Tampilan \"DATA PLA DLA XOL KLAIM\" TIDAK dibawa sama sekali. Sectionnya memang " +
		"memuat wadah berjudul itu, tetapi wadahnya bersyarat " +
		"`pyContainerVisibleWhen = TempView.CityID==7` sementara `CityID` hanya pernah " +
		"diisi dari baris tabel \"Status / Jumlah\" — dan tabel itu berisi enam baris, " +
		"tidak satu pun bernilai 7. Di Pega tampilan itu tidak pernah dapat tergambar.",

	"Kolom \"Status\" kini menyertakan ARTI kodenya, bukan angka saja. Kueri lama hanya " +
		"mengambil kodenya; `R-06` sudah tertutup dan ke-33 artinya tersedia di master.",

	"Daftar DIPAGINASI di basis data. Ketiga kueri Pega memuat SELURUH baris yang cocok " +
		"ke memori sekaligus, tanpa satu pun batas. Tabel ringkas \"Status / Jumlah\" " +
		"karena itu dihitung di basis data di sini, bukan dari baris yang sedang tampil.",

	"Pencarian TIDAK peka huruf besar-kecil. Kueri lama membandingkan apa adanya.",

	"Ketiga daftar komunikasi TIDAK mengecualikan lini Personal Accident (`002`) maupun " +
		"Travel (`005`), sementara ketiga daftar pemberitahuan mengecualikan keduanya. " +
		"Itu perbedaan yang memang ada di kueri Pega — `BrowseCommunicationReas` tidak " +
		"memuat satu pun syarat `grouppanel` — dan ia dibawa apa adanya.",

	"Grid PLA dan DLA pada layar rincian hanya menampilkan pemberitahuan MILIK ANDA yang " +
		"sudah terkirim. Di Pega gridnya dimuat dari objek kerja klaim, sehingga ia " +
		"memuat pemberitahuan SELURUH mitra pada klaim itu — beserta nilai masing-" +
		"masing. Itu kebocoran antar mitra, dan ia tidak dibawa.",

	"Dokumen disaring pula menurut LOGIN Anda (`T_DOC_REAS.LOGIN`). `GetDokumenReas` " +
		"tidak memakai kolom itu — ia menyaring nomor pemberitahuan saja. Penyaring " +
		"tambahan ini diputuskan Work Owner pada 2026-09-28.",

	"Balasan komunikasi menolak percakapan yang SUDAH dijawab. `ReplyKomunikasi` tidak " +
		"memagarinya, dan tanpa pagar itu balasan kedua menimpa balasan pertama pada " +
		"kolom yang sama tanpa dapat dipulihkan.",

	"Tombol \"Detail\" dan \"DLA\" TIDAK dibawa, dan keduanya bukan penundaan. \"DLA\" " +
		"berada di dalam wadah bersyarat `1==2` — ia tidak pernah tergambar di Pega. " +
		"\"Detail\" hanya tampil bagi satu Operator ID yang ditulis tetap di dalam rule, " +
		"dan penulisan seperti itu dilarang `D-15`.",

	"Perbedaan yang DIBAWA apa adanya: daftar \"Close\" mencocokkan SELURUH kode " +
		"reasuradur milik login Anda, sementara daftar \"PLA\" dan \"DLA\" hanya " +
		"mencocokkan kode tertinggi. Itu perbedaan di kueri Pega, dan alasannya tidak " +
		"tertulis di mana pun.",

	"Perbedaan yang DIBAWA apa adanya: daftar \"Close\" disaring oleh PLA yang terkirim, " +
		"bukan oleh DLA — meski ia menampilkan klaim yang sudah selesai.",
}

// Repo adalah seam ke penyimpanan, dideklarasikan di sini karena di sinilah ia DIPAKAI
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 2).
type Repo interface {
	// ReinsurerCodes mengembalikan kode reasuradur milik satu login, terurut MENURUN.
	//
	// Kosong berarti login itu bukan reasuradur — pemanggil yang menerjemahkannya menjadi
	// ErrCallerNotAReinsurer.
	//
	// # Kenapa SELURUH kodenya, bukan satu
	//
	// Karena ketiga kueri Pega tidak sepakat: dua memakai kode TERTINGGI saja, satu
	// memakai semuanya. Seam ini menyerahkan keduanya dan Tab yang memutuskan mana yang
	// dipakai — sehingga perbedaan itu terbaca di satu tempat, bukan tersebar di tiga
	// kueri.
	ReinsurerCodes(ctx context.Context, login string) ([]string, error)

	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// Counts mengembalikan tabel ringkas "Status / Jumlah" atas penyaring yang SAMA
	// dengan daftarnya.
	//
	// Ia terpisah dari List karena memang hitungan yang berbeda — dan karena isinya tidak
	// berubah saat pengguna berpindah halaman, sehingga layar dapat menyimpannya lebih
	// lama.
	Counts(ctx context.Context, query Query) ([]StatusCount, error)

	// ============================================================================
	// LAYAR RINCIAN — tombol "Detail Claim"
	// ============================================================================
	//
	// Keenam operasi di bawah menerima SCOPE, bukan sekadar kunci klaim. Itu bukan
	// kerapian: setiap satunya dapat dipanggil lewat alamat langsung, dan tanpa batas
	// reasuradur di dalam pernyataannya, sebuah kunci klaim yang ditebak akan membuka
	// pemberitahuan, dokumen, dan percakapan milik mitra lain.
	//
	// Batasnya karena itu dibawa sebagai parameter yang TIDAK DAPAT DILUPAKAN, bukan
	// diperiksa di lapisan di atasnya.

	// ClaimHeader mengembalikan keterangan klaim di kepala layar rincian.
	//
	// ErrRowNotFound berarti klaimnya tidak ada ATAU tidak satu pun pemberitahuannya
	// pernah dikirimkan kepada pemanggil — dan keduanya dijawab sama, dengan alasan yang
	// sama seperti ErrDocumentNotFound.
	ClaimHeader(ctx context.Context, scope DetailScope) (ClaimHeader, error)

	// Advices mengembalikan isi grid PLA atau grid DLA satu klaim.
	//
	// Hanya yang SUDAH terkirim kepada pemanggil, dan hanya miliknya. Lihat catatan
	// selisih pada PlannedDifferences.
	Advices(ctx context.Context, scope DetailScope, kind AdviceKind) ([]AdviceRow, error)

	// Documents mengembalikan dokumen satu nomor pemberitahuan.
	//
	//	POOLDATA.T_DOC_REAS  ->  POOLDATA.DATA_ATTACHFILE
	//
	// Kuncinya `CLAIMID + NO_PLADLA + TIPE_PLADLA`, persis `GetDokumenReas`, DITAMBAH
	// `LOGIN` atas keputusan Work Owner.
	Documents(
		ctx context.Context, scope DetailScope, adviceNo string, kind AdviceKind,
	) ([]DocumentRow, error)

	// DocumentContent mengembalikan ISI satu dokumen.
	//
	// Ia memeriksa ulang seluruh rantai kepemilikannya — bukan hanya id dokumennya.
	// Sebuah `DATAID` adalah angka, dan angka dapat ditebak.
	DocumentContent(
		ctx context.Context, scope DetailScope, documentID string,
	) (DocumentContent, error)

	// Conversations mengembalikan riwayat komunikasi satu klaim yang menyangkut pemanggil.
	Conversations(ctx context.Context, scope DetailScope) ([]Conversation, error)

	// Reply menyimpan balasan atas satu percakapan.
	//
	// Ia menolak dengan ErrConversationNotFound bila percakapannya bukan milik pemanggil,
	// dan dengan ErrConversationAlreadyAnswered bila sudah pernah dijawab — keduanya
	// diputuskan di dalam pernyataannya sendiri, bukan lewat baca-lalu-tulis. Dua
	// permintaan yang datang bersamaan akan membuat pemeriksaan terpisah meloloskan
	// keduanya, dan balasan kedua menimpa yang pertama tanpa jejak.
	Reply(ctx context.Context, scope DetailScope, command ReplyCommand) error
}

// DetailScope adalah batas yang berlaku pada seluruh operasi layar rincian.
//
// # Kenapa ia tipe tersendiri
//
// Karena keenam operasi rincian menuntut batas yang SAMA PERSIS, dan menuliskannya sebagai
// tiga parameter lepas pada enam tanda tangan berarti enam kesempatan melupakan salah
// satunya. Yang dilupakan tidak menghasilkan galat — hanya data mitra lain yang terbuka.
type DetailScope struct {
	// ClaimKey adalah `T_CLAIM_PNC.CLAIMID`.
	ClaimKey string

	// Login adalah login pemanggil, dipakai menerjemahkan kode reasuradurnya di dalam SQL
	// dan menyaring `T_DOC_REAS.LOGIN` serta kedua sisi percakapan.
	Login string

	// ReinsurerCodes adalah kode reasuradur milik pemanggil, terurut MENURUN.
	//
	// Ia dibawa meski kuerinya menerjemahkan login sendiri, karena penyimpanan MEMORI
	// tidak punya SQL untuk menerjemahkannya — dan kedua penyimpanan harus menegakkan
	// batas yang sama.
	ReinsurerCodes []string
}

// Clean memangkas spasi pada batas rincian.
func (s DetailScope) Clean() DetailScope {
	return DetailScope{
		ClaimKey:       strings.TrimSpace(s.ClaimKey),
		Login:          strings.TrimSpace(s.Login),
		ReinsurerCodes: s.ReinsurerCodes,
	}
}

// RepoSelector memilih penyimpanan milik satu portal entitas.
//
// Ia ada karena `D-75` menetapkan SATU BASIS DATA PER ENTITAS. Alias portal yang tidak
// dikenal menghasilkan galat, bukan jatuh ke koneksi bawaan (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
