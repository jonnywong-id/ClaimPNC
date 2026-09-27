// Package masterkategorisparepart adalah inti modul Master Kategori Sparepart.
//
// # Apa yang dimodelkan di sini
//
// Daftar **penggolongan suku cadang alat berat** — satu baris per kategori, dan setiap
// kategori hanya punya nama. Ia tabel acuan yang menyuapi dua modul lain:
//
//	Master Sparepart      KATEGORI_SPART menunjuk PART_CATEGORY_ID di sini
//	Master Tipe Sparepart PART_CATEGORY_ID menjadi induk setiap tipe
//
// Modul ini menjadi **penulis** tabel itu; `mastersparepart` tetap sekadar pembaca lewat
// lookup.go-nya. Pembagian itu memenuhi `P-1` — satu tabel, satu penulis.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/GCNMCatSparepart-Harness.xml                    layar, MENU_ID 33
//	Section/MasterKategoriSparepartHE-Section.xml           judul, tombol, 3 tab
//	Section/MasterKategoriSparepartHEApprove-Section.xml    tab disetujui, pyPageSize=50
//	Section/MasterKategoriSparepartHEReject-Section.xml     tab ditolak
//	Section/MasterKategoriSparepartHEApproval-Section.xml   tab antrean persetujuan
//	Section/ApprovalMasterKategoriSparepartHE-Section.xml   persetujuan borongan Inbox Manager
//	RDB List/BrowseSparepartCategoryClaimHE-SQL.xml         daftar, APPROVAL = '0'
//	RDB List/BrowseMasterSparepartCategoryClaimHE-SQL.xml   daftar, APPROVAL berparameter
//	RDB List/BrowseSparepartCategoryClaimHE_sql-SQL.xml     satu baris menurut ID
//	RDB List/InsertMasterSparepartCategory_sql-SQL.xml      sisip, ID = max+1
//	RDB List/UpdateMasterSparepartCategory_sql2-SQL.xml     simpan nama dan APPROVAL
//	RDB List/ValidationSparepartCat-SQL.xml                 tolak nama ganda
//	RDB List/CountMasterKatSparepartManager-SQL.xml         pencacah antrean persetujuan
//	Activity/UpdateKategoriSparepart_act2-Act.xml           urutan langkah simpan
//	Activity/ValidateMasterKategoriSparepart-Act.xml        pesan galat nama ganda
//	Activity/SetMasterKategoriSparepart_act-Act.xml         pemuatan baris ke form
//	Activity/UpdateKategoriSparepart_act-Act.xml            keputusan persetujuan
//	Database/m_menu_aplikasi_pnc.csv                        MENU_ID 33 "Master Kategori Sparepart"
//
// # Empat hal yang membedakannya dari Master Sparepart
//
//  1. **Tiga kolom, bukan dua puluh empat.** `POOLDATA.GCNM_M_SPAREPART_CATEGORY` hanya
//     punya `PART_CATEGORY_ID`, `PART_CATEGORY_NAME`, dan `APPROVAL`. Ketiganya terbaca
//     lengkap dari kesembilan rule yang menyentuh tabel itu — bukan dari satu rule saja.
//  2. **ID-nya angka berurut, bukan kode situs + sequence.**
//     `InsertMasterSparepartCategory_sql` menerbitkannya dengan
//     `nvl(max(PART_CATEGORY_ID),0)+1`; tidak ada sequence maupun `M_SITE_DATABASE` di
//     jalur ini. Lihat IDSource.
//  3. **Tidak ada pencatat pelaku maupun stempel waktu.** Tabelnya tidak punya
//     `USER_UPDATE` maupun kolom tanggal apa pun, sehingga siapa yang menyimpan dan kapan
//     TIDAK tersimpan di mana pun. Sama seperti Master Panel, dan berbeda dari Master
//     Sparepart.
//  4. **Tidak ikut `SetApprovalAllMaster`.** Activity itu melayani bengkel, panel, dan
//     sparepart saja. Kategori punya jalurnya sendiri; lihat Repo.SetStatus.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterkategorisparepart

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY`: tabelnya masih dibaca sistem lama selama masa
// paralel (ADR-0004), dan ia juga dibaca modul `mastersparepart` di aplikasi ini sendiri
// lewat `lookup.go` yang menyaring `APPROVAL = '1'`. Mengubah sandi nilainya akan membuat
// dropdown Kategori pada layar Master Sparepart kosong tanpa satu pun pesan galat.
//
// Ketiga sandinya terbaca dari tiga sumber yang saling menguatkan:
//
//	Section/MasterKategoriSparepartHE      tiga tab dengan penyaring berbeda
//	RDB List/CountMasterKatSparepartManager  mencacah APPROVAL = '0'
//	mastersparepart/lookup.go                memakai APPROVAL = '1' sebagai "disetujui"
//
// Sandinya kebetulan sama dengan Master Bengkel, Panel, Sparepart, Auto Claim, dan
// Rekening. Keenamnya sengaja TIDAK dipakai bersama: tabelnya berbeda, dan tipe bersama
// membuat perubahan di satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan. Nilai lahir setiap baris baru DAN setiap
	// baris yang disunting: `Activity/UpdateKategoriSparepart_act2` menetapkan
	// `InputKategori.LOGIN_APLIKASI := "0"` tanpa syarat, dan properti itulah yang
	// dipetakan ke kolom APPROVAL oleh kedua rule simpannya.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. HANYA baris berstatus ini yang muncul sebagai pilihan
	// Kategori di layar Master Sparepart dan Master Tipe Sparepart.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Ketiganya diambil dari caption tab pada
// `Section/MasterKategoriSparepartHE-Section.xml` apa adanya — layar itu memang
// menuliskan "Approve", "Reject", dan "Waiting Approval". Sama persis dengan Master
// Sparepart, Panel, dan Bengkel.
func (s ApprovalStatus) Label() string {
	switch s {
	case StatusPending:
		return "Waiting Approval"
	case StatusApproved:
		return "Approve"
	case StatusRejected:
		return "Reject"
	default:
		return ""
	}
}

// Known menyatakan status ini termasuk salah satu dari tiga yang sah.
func (s ApprovalStatus) Known() bool {
	return s == StatusPending || s == StatusApproved || s == StatusRejected
}

// PartCategory adalah satu kategori sparepart — satu baris
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY`.
//
// Namanya **PartCategory**, mengikuti awalan kolomnya sendiri (`PART_CATEGORY_*`), bukan
// `Category` begitu saja: `Category` terlalu umum untuk tipe yang menempati paket
// bernama modul, dan `mastersparepart` sudah memakai nama itu untuk DTO lookup-nya.
//
// # Hanya tiga kolom, dan itu sudah dipastikan
//
// Kesembilan rule yang menyentuh tabel ini — lima browse, satu insert, satu update, satu
// validasi, satu pencacah — tidak satu pun menyebut kolom di luar ketiga ini. Tidak ada
// kolom pencatat pelaku, tidak ada stempel waktu, dan tidak ada kolom alasan penolakan.
//
// Akibat yang harus disadari: **siapa yang menambah atau menolak sebuah kategori tidak
// tersimpan di mana pun.** Itu keterbatasan tabelnya, bukan kelalaian modul ini; lihat
// catatan pada usecase.Service.Decide.
type PartCategory struct {
	// ID adalah kolom PART_CATEGORY_ID — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `InsertMasterSparepartCategory_sql` menerbitkannya dari
	// isi tabelnya sendiri; lihat IDSource.
	//
	// Disimpan sebagai TEKS meski isinya angka. Alasannya sama dengan modul master lain:
	// DDL tabelnya tidak ada di export (`R-08`), dan membacanya sebagai int64 lalu
	// menuliskannya kembali akan mengubah bentuk baris yang tidak pernah disunting siapa
	// pun — misalnya "007" menjadi "7". Nilai ini juga disalin apa adanya ke
	// `SPAREPART_HE.KATEGORI_SPART` yang bertipe teks.
	ID string

	// Name adalah kolom PART_CATEGORY_NAME — "Nama Kategori Sparepart".
	//
	// Satu-satunya isian yang diketik pengguna, dan satu-satunya kunci alami; lihat
	// ErrNameTaken.
	Name string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// SATU isian saja — caption `pyCaption Nama Kategori Sparepart` pada
// `Section/MasterKategoriSparepartHEApproval-Section.xml`. Yang ADA di tabel tetapi TIDAK
// di sini, karena keduanya diturunkan sistem:
//
//	PART_CATEGORY_ID  kunci baris; diterbitkan saat penambahan, lihat IDSource
//	APPROVAL          selalu StatusPending pada penyimpanan lewat layar ini
type Input struct {
	Name string
}

// MaxNameLength adalah panjang maksimum nama kategori.
//
// ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun dibaca dari DDL:
// `POOLDATA.GCNM_M_SPAREPART_CATEGORY` tidak ada DDL-nya di export (`R-08`), dan layar
// lamanya tidak memasang satu pun `pyMaxLength`.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Seratus dipilih agar sama dengan `mastersparepart.MaxNameLength`: nilai kolom ini
// dipakai sebagai label pada layar Master Sparepart, dan dua batas yang berbeda pada dua
// layar bertetangga hanya akan membingungkan.
//
// Angka yang sama diulang di `PartCategoryForm.tsx`. Bila berubah, KEDUA tempat harus
// ikut berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh
// uji di masterkategorisparepart_test.go.
const MaxNameLength = 100

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterkategorisparepart: kategori sparepart tidak ditemukan")

	// ErrNameTaken: PART_CATEGORY_NAME yang akan disimpan sudah dipakai baris lain.
	//
	// Padanan `Activity/ValidateMasterKategoriSparepart`, yang membandingkan
	// `@toUpperCase(Param.Name)` terhadap `upper(PART_CATEGORY_NAME)` lewat
	// `RDB List/ValidationSparepartCat-SQL.xml` lalu menyusun pesan:
	//
	//	"Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain."
	//
	// PERHATIKAN CAKUPANNYA. Rule itu TIDAK menyaring APPROVAL sama sekali, sehingga nama
	// kategori yang pernah DITOLAK tetap memblokir pemakaian nama itu selamanya. Perilaku
	// itu ditiru apa adanya atas keputusan Work Owner 2026-09-21 (`P-5`: perilaku
	// dipertahankan lebih dulu, diperbaiki kemudian); perbaikannya dicatat sebagai Future
	// Enhancement, bukan dikerjakan sambil jalan.
	ErrNameTaken = errors.New("masterkategorisparepart: nama kategori sparepart sudah dipakai")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("masterkategorisparepart: status persetujuan tidak dikenal")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Modul ini hanya punya satu isian, sehingga bentuk jamaknya tampak berlebihan. Ia tetap
// dipakai karena bentuk galat yang sama dipakai seluruh modul master, dan layar
// menanganinya dengan satu kode yang sama — `P-5` menuntut seluruh pelanggaran dikirim
// bersamaan, dan bentuk tunggal di satu modul akan menjadi perkecualian yang harus
// diingat setiap kali layar baru ditulis.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data — keunikan
// nama — supaya galatnya sampai ke layar dalam bentuk yang SAMA dengan pelanggaran isian
// lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterkategorisparepart: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Nama TIDAK di-UPPERCASE. Rule validasinya memang membandingkan `upper(...)`, tetapi yang
// di-uppercase di sana adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf
// besar akan mengubah tampilan setiap baris yang disunting, dan itu selisih yang tidak
// diminta siapa pun. Perlakuan yang sama dipakai Master Sparepart.
func (i Input) Clean() Input {
	return Input{Name: strings.TrimSpace(i.Name)}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// Nama wajib diisi. Layar lamanya TIDAK menandainya `pyRequired` — tidak satu pun isian
// pada section ini bertanda itu — tetapi `Activity/UpdateKategoriSparepart_act2` tetap
// menyimpan apa pun yang diketik, termasuk kosong.
//
// Kewajiban di sini karena itu **DITAMBAHKAN** terhadap sistem lama, dan alasannya bukan
// kerapian: kategori tanpa nama akan muncul sebagai baris kosong pada dropdown Kategori di
// layar Master Sparepart — tidak dapat dibedakan dari "belum memilih", dan tidak dapat
// dipilih ulang setelah salah pilih. Baris lama yang sudah kosong tetap DIBACA apa adanya;
// penolakan hanya terjadi saat barisnya disimpan ulang.
func (i Input) Check() error {
	var violation []Violation

	switch {
	case i.Name == "":
		violation = append(violation, Violation{
			Field:   "nama_kategori_sparepart",
			Message: "Nama kategori sparepart wajib diisi.",
		})
	case len(i.Name) > MaxNameLength:
		violation = append(violation, Violation{
			Field: "nama_kategori_sparepart",
			Message: fmt.Sprintf("Nama kategori sparepart paling panjang %d karakter.",
				MaxNameLength),
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/MasterKategoriSparepartHE-Section.xml`, yang ketiganya
// membaca tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nama kategori.
	//
	// DITAMBAHKAN terhadap sistem lama, yang memuat seluruh baris ke klipboard lalu
	// menyaringnya di peramban (`pyPageSize=50`, tanpa satu pun kotak pencarian di
	// section-nya). Kosong berarti tanpa penyaring.
	//
	// Hanya nama, karena hanya itu yang bermakna dicari: ID-nya nomor urut yang tidak
	// dihafal siapa pun, dan status sudah menjadi tab tersendiri.
	Keyword string
}

// IDSource menerbitkan ID baru.
//
// Ia seam tersendiri, bukan method pada Repo, dengan alasan yang sama seperti pada modul
// master lain: yang dikerjakannya bukan urusan kategori melainkan urusan **penomoran**.
//
// # Bentuknya, dibaca dari RDB List/InsertMasterSparepartCategory_sql-SQL.xml
//
//	insert into POOLDATA.gcnm_m_sparepart_category
//	       (PART_CATEGORY_ID, PART_CATEGORY_NAME, APPROVAL)
//	Values ((select nvl(max(PART_CATEGORY_ID),0)+1 from POOLDATA.gcnm_m_sparepart_category),
//	        {InputKategori.CITY_ID}, {InputKategori.LOGIN_APLIKASI})
//
// **Angka berurut biasa** — tanpa kode situs, tanpa sequence, dan tanpa nol di depan. Itu
// berbeda dari Master Sparepart dan Master Panel, yang keduanya memakai
// `M_SITE_DATABASE` ditambah sequence. Bentuknya ditiru apa adanya: menerbitkan kunci
// berbentuk lain akan membuat baris baru tidak sebentuk dengan baris yang sudah ada, dan
// nilai itu disalin ke `SPAREPART_HE.KATEGORI_SPART` yang sudah berisi angka-angka lama.
//
// # Balapan yang ADA di sistem lama, dan bagaimana ia ditutup
//
// `max(...)+1` di dalam satu INSERT tidak menghalangi dua penyimpanan bersamaan membaca
// nilai maksimum yang sama lalu menyisipkan ID kembar. Sistem lama tidak menjaganya sama
// sekali.
//
// Work Owner memutuskan (2026-09-21) balapan itu **ditutup dengan penguncian**, bukan
// dengan meminta sequence baru ke DBA: bentuk ID-nya tetap sama persis dengan Pega
// (`P-5`), dan modulnya tidak tertahan menunggu prosedur perubahan skema (`D-63`).
// Pelaksanaannya ada di repo — lihat `category_next_id` pada berkas .sql.
//
// KETERBATASAN YANG DISADARI: penguncian hanya menutup balapan ANTAR-INSTANS APLIKASI INI.
// Selama Pega masih menulis tabel yang sama, keduanya tetap dapat menerbitkan ID kembar.
// Penutupnya adalah kewenangan tulis yang berpindah penuh saat modul ini lulus gerbang 2
// (`P-1`), ditambah constraint unik yang menunggu DDL (`R-08`).
type IDSource interface {
	// NextID mengembalikan ID berikutnya.
	//
	// Nilainya HANYA sah di dalam transaksi yang menerbitkannya — lihat Repo.Insert, yang
	// menerbitkan dan menyisipkan dalam satu transaksi. Memanggilnya di luar itu
	// mengembalikan angka yang dapat basi sebelum dipakai.
	NextID(ctx context.Context) (string, error)
}

// Repo adalah seam ke penyimpanan master kategori sparepart SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]PartCategory, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/BrowseSparepartCategoryClaimHE_sql-SQL.xml`, yang dipakai
	// `Activity/SetMasterKategoriSparepart_act` untuk memuat baris ke form.
	Get(ctx context.Context, id string) (PartCategory, error)

	// FindByName mencari baris menurut PART_CATEGORY_NAME-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationSparepartCat-SQL.xml`, yang mencocokkan
	// `upper(PART_CATEGORY_NAME)` — perlakuan yang ditiru apa adanya, termasuk TIDAK
	// menyaring APPROVAL. Lihat ErrNameTaken.
	FindByName(ctx context.Context, name string) (PartCategory, error)

	// Insert menyisipkan baris baru dan mengembalikan baris yang benar-benar tersimpan.
	//
	// ID pada argumen DIABAIKAN: ia diterbitkan di dalam operasi ini, di dalam transaksi
	// yang sama dengan penyisipannya. Memisahkannya menjadi "terbitkan" lalu "sisip" di
	// lapisan aplikasi akan membuka kembali balapan yang justru sedang ditutup — lihat
	// IDSource.
	//
	// Pemeriksaan keunikan nama juga berada DI DALAM operasi ini, dengan alasan yang sama.
	// Sistem lama memecahnya — `ValidateMasterKategoriSparepart` dipanggil lebih dulu,
	// penyisipannya menyusul — dan jarak di antara keduanya tidak dijaga apa pun.
	Insert(ctx context.Context, c PartCategory) (PartCategory, error)

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	//
	// Padanan `RDB List/UpdateMasterSparepartCategory_sql2-SQL.xml`, yang menulis
	// PART_CATEGORY_NAME dan APPROVAL sekaligus.
	Update(ctx context.Context, c PartCategory) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
	//
	// # Rule yang menjalankannya TIDAK ADA di export
	//
	// `Activity/UpdateKategoriSparepart_act` menerima `Param.ID` dan `Param.Approval` lalu
	// memanggil `UpdateSparepartCategoryClaimHE_sql` — dan rule itu **tidak ada di antara
	// 2.634 berkas export** (`R-16`). Kategori juga tidak ikut
	// `Activity/SetApprovalAllMaster`, yang hanya melayani bengkel, panel, dan sparepart.
	//
	// Bentuk pernyataannya karena itu **DIREKONSTRUKSI**, bukan dibaca: ia menulis kolom
	// APPROVAL pada baris yang ditunjuk PART_CATEGORY_ID, mengikuti
	// `UpdateMasterSparepartCategory_sql2` yang menulis kedua kolom pada tabel yang sama.
	// Rekonstruksi itu dinyatakan di sini supaya ia dapat diuji ulang begitu rule aslinya
	// tiba, bukan tersamar sebagai fakta.
	//
	// TANPA alasan penolakan: tabelnya tidak punya kolom penampungnya. Lihat
	// usecase.Service.Decide.
	//
	// Yang dikembalikan adalah jumlah baris yang benar-benar berubah, supaya pemanggil
	// dapat membedakan "tidak ada yang dipilih" dari "yang dipilih sudah tidak ada".
	SetStatus(ctx context.Context, id []string, status ApprovalStatus) (int, error)
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan seam yang dipakai layanan modul ini.
//
// Keduanya tetap DIDEKLARASIKAN terpisah — Repo untuk tabelnya, IDSource untuk penomoran —
// karena keduanya menjawab pertanyaan yang berbeda dan dapat berubah sendiri-sendiri.
// Yang disatukan hanyalah CARA MEMILIHNYA: keduanya selalu berasal dari koneksi entitas
// yang sama.
//
// IDSource tetap ada di sini meski Insert menerbitkan ID-nya sendiri: ia dipakai
// `claimpnc -periksa` untuk membuktikan penerbitan kunci bekerja terhadap basis data nyata
// TANPA menyisipkan satu baris pun.
type Store interface {
	Repo
	IDSource
}
