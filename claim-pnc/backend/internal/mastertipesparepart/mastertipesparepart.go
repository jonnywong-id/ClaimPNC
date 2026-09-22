// Package mastertipesparepart adalah inti modul Master Tipe Sparepart.
//
// # Apa yang dimodelkan di sini
//
// Daftar **tipe suku cadang alat berat**, dan setiap tipe BERINDUK pada sebuah kategori.
// Ia tabel acuan yang menyuapi satu modul lain:
//
//	Master Sparepart   TIPE_SPART menunjuk PART_SECTION_ID di sini
//
// Modul ini menjadi **penulis** tabel itu; `mastersparepart` tetap sekadar pembaca lewat
// lookup.go-nya. Pembagian itu memenuhi `P-1` — satu tabel, satu penulis.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/GCNMMasterSparepartType-Harness.xml             layar, MENU_ID 34
//	Section/MasterTipeSparepartHE-Section.xml               judul, tombol, 3 tab
//	Section/MasterTipeSparepartHEApprove-Section.xml        tab disetujui, pyPageSize=50
//	Section/MasterTipeSparepartHEReject-Section.xml         tab ditolak
//	Section/MasterTipeSparepartHEApproval-Section.xml       tab antrean persetujuan
//	Section/ApprovalMasterTipeSparepartHE-Section.xml       persetujuan borongan Inbox Manager
//	RDB List/BrowseSparepartTipeClaimHE2-SQL.xml            daftar, APPROVAL = '0'
//	RDB List/BrowseTipeSparepart-SQL.xml                    daftar, APPROVAL = '1'
//	RDB List/BrowseMasterSparepartTypeClaimHE_sql-SQL.xml   satu baris + JOIN nama kategori
//	RDB List/BrowseSparepartTypeClaimHE_sql-SQL.xml         satu baris, tanpa penyaring status
//	RDB List/InsertMasterSparepartType_sql-SQL.xml          sisip, ID = max+1
//	RDB List/UpdateMasterSparepartType_sql2-SQL.xml         simpan nama, kategori, APPROVAL
//	RDB List/ValidationSparepartType-SQL.xml                tolak nama ganda
//	RDB List/CountMasterTipeSparepartManager-SQL.xml        pencacah antrean persetujuan
//	Activity/UpdateTypeSparepart_act2-Act.xml               urutan langkah simpan
//	Activity/ValidateMasterTipeSparepart-Act.xml            pesan galat nama ganda
//	Activity/SetMasterTipeSparepart_act-Act.xml             pemuatan baris ke form
//	Activity/UpdateTipeSparepart_act-Act.xml                keputusan persetujuan
//	Database/m_menu_aplikasi_pnc.csv                        MENU_ID 34 "Master Tipe Sparepart"
//
// # Tiga hal yang membedakannya dari Master Kategori Sparepart
//
//  1. **Ia punya induk.** `PART_CATEGORY_ID` menunjuk
//     `POOLDATA.GCNM_M_SPAREPART_CATEGORY`, dan itu menjadikan modul ini master pertama di
//     rumpun sparepart yang menyimpan kunci asing. Konsekuensinya ada di lookup.go: layar
//     butuh daftar kategori, dan daftar itu dibaca dari tabel milik modul LAIN.
//  2. **Empat kolom, bukan tiga.** `POOLDATA.GCNM_M_SPAREPART_TYPE` punya
//     `PART_SECTION_ID`, `PART_SECTION_NAME`, `PART_CATEGORY_ID`, dan `APPROVAL`. Keempatnya
//     terbaca lengkap dari kesembilan rule yang menyentuh tabel itu.
//  3. **Nama kolomnya tidak sejalan dengan nama bisnisnya.** Yang disebut "Tipe Sparepart"
//     di seluruh layar tersimpan di kolom berawalan `PART_SECTION_*` — "section", bukan
//     "type". Nama tabelnya sendiri `GCNM_M_SPAREPART_TYPE`. Selisih itu dipertahankan di
//     sisi basis data (`D-80`: nama kolom tetap seperti adanya) dan TIDAK dibawa ke nama
//     tipe Go; lihat PartType.
//
// # Yang SAMA persis dengan Master Kategori Sparepart
//
// Tidak ada pencatat pelaku dan tidak ada stempel waktu — tabelnya tidak punya kolomnya,
// sehingga siapa yang menyimpan dan kapan TIDAK tersimpan di mana pun. Dan seperti
// kategori, tipe tidak ikut `Activity/SetApprovalAllMaster`, yang hanya melayani bengkel,
// panel, dan sparepart.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastertipesparepart

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada
// `POOLDATA.GCNM_M_SPAREPART_TYPE`: tabelnya masih dibaca sistem lama selama masa paralel
// (ADR-0004), dan ia juga dibaca modul `mastersparepart` di aplikasi ini sendiri lewat
// `lookup.go` yang menyaring `APPROVAL = '1'`. Mengubah sandi nilainya akan membuat
// dropdown Tipe pada layar Master Sparepart kosong tanpa satu pun pesan galat.
//
// Ketiga sandinya terbaca dari tiga sumber yang saling menguatkan:
//
//	Section/MasterTipeSparepartHE            tiga tab dengan penyaring berbeda
//	RDB List/CountMasterTipeSparepartManager mencacah APPROVAL = '0'
//	RDB List/BrowseTipeSparepart             memakai APPROVAL = '1' sebagai "disetujui"
//
// Sandinya kebetulan sama dengan Master Bengkel, Panel, Sparepart, Kategori Sparepart,
// Auto Claim, dan Rekening. Ketujuhnya sengaja TIDAK dipakai bersama: tabelnya berbeda,
// dan tipe bersama membuat perubahan di satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan. Nilai lahir setiap baris baru DAN setiap
	// baris yang disunting: `Activity/UpdateTypeSparepart_act2` menetapkan
	// `InputKategori.NO_ACCOUNT := "0"` tanpa syarat, dan properti itulah yang dipetakan ke
	// kolom APPROVAL oleh kedua rule simpannya.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. HANYA baris berstatus ini yang muncul sebagai pilihan
	// Tipe di layar Master Sparepart.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Ketiganya diambil dari caption tab pada `Section/MasterTipeSparepartHE-Section.xml` apa
// adanya — layar itu memang menuliskan "Approve", "Reject", dan "Waiting Approval". Sama
// persis dengan Master Kategori Sparepart, Sparepart, Panel, dan Bengkel.
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

// PartType adalah satu tipe sparepart — satu baris `POOLDATA.GCNM_M_SPAREPART_TYPE`.
//
// # Kenapa namanya PartType, dan bukan Type maupun PartSection
//
// `Type` adalah kata kunci Go dan tidak dapat dipakai sebagai nama tipe. `PartSection` —
// yang akan mengikuti awalan kolomnya sendiri, seperti yang dilakukan
// `masterkategorisparepart.PartCategory` terhadap `PART_CATEGORY_*` — justru menyesatkan di
// sini: tidak ada satu pun layar, menu, maupun caption yang menyebut "section". Yang
// dilihat dan diucapkan pengguna adalah **Tipe Sparepart**.
//
// Nama `PartType` karena itu mengikuti NAMA TABELNYA (`GCNM_M_SPAREPART_TYPE`) dan nama
// bisnisnya, bukan awalan kolomnya. Ia juga sama dengan `mastersparepart.PartType`, yang
// sudah lebih dulu menamai hal yang sama pada lookup-nya — dua paket berbeda menamai satu
// konsep dengan satu nama.
//
// # Empat kolom, dan itu sudah dipastikan
//
// Kesembilan rule yang menyentuh tabel ini — empat browse, satu insert, satu update, satu
// validasi, satu pencacah, satu keputusan — tidak satu pun menyebut kolom di luar keempat
// ini. Tidak ada kolom pencatat pelaku, tidak ada stempel waktu, dan tidak ada kolom alasan
// penolakan.
//
// Akibat yang harus disadari: **siapa yang menambah atau menolak sebuah tipe tidak
// tersimpan di mana pun.** Itu keterbatasan tabelnya, bukan kelalaian modul ini; lihat
// catatan pada usecase.Service.Decide.
type PartType struct {
	// ID adalah kolom PART_SECTION_ID — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `InsertMasterSparepartType_sql` menerbitkannya dari isi
	// tabelnya sendiri; lihat IDSource.
	//
	// Disimpan sebagai TEKS meski isinya angka, dengan alasan yang sama seperti pada
	// `masterkategorisparepart.PartCategory.ID`: DDL tabelnya tidak ada di export (`R-08`),
	// dan nilai ini disalin apa adanya ke `SPAREPART_HE.TIPE_SPART` yang bertipe teks.
	ID string

	// Name adalah kolom PART_SECTION_NAME — "Nama Tipe Sparepart".
	//
	// Satu-satunya isian teks yang diketik pengguna, dan satu-satunya kunci alami; lihat
	// ErrNameTaken.
	Name string

	// CategoryID adalah kolom PART_CATEGORY_ID — induk tipe ini.
	//
	// Ia kunci asing ke `POOLDATA.GCNM_M_SPAREPART_CATEGORY`, tetapi TIDAK dijaga constraint
	// apa pun sejauh yang dapat dipastikan: DDL-nya tidak ada di export (`R-08`). Yang
	// menjaganya adalah pemeriksaan di lapisan aplikasi — lihat usecase.Service.Create.
	CategoryID string

	// CategoryName adalah kolom PART_CATEGORY_NAME milik tabel kategori, ikut dibaca lewat
	// JOIN.
	//
	// Ia BUKAN kolom tabel ini dan TIDAK pernah ditulis. Ia ikut dibawa karena grid layar
	// lama menampilkan keduanya berdampingan —
	// `BrowseMasterSparepartTypeClaimHE_sql-SQL.xml` menggabungkan kedua tabel justru untuk
	// itu:
	//
	//	from POOLDATA.gcnm_m_sparepart_type a, POOLDATA.GCNM_M_SPAREPART_CATEGORY b
	//	where A.PART_CATEGORY_ID = B.PART_CATEGORY_ID
	//
	// Ia dapat KOSONG, dan itu bukan galat: JOIN lama berbentuk inner join, sehingga tipe
	// yang menunjuk kategori yang sudah tidak ada akan HILANG dari daftar di sistem lama.
	// Modul ini memakai LEFT JOIN supaya barisnya tetap terlihat dengan nama kategori
	// kosong — lihat catatan pada berkas .sql. Itu satu-satunya selisih perilaku yang
	// disengaja pada jalur baca.
	CategoryName string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// DUA isian, mengikuti caption pada
// `Section/MasterTipeSparepartHEApproval-Section.xml`: "Nama Tipe Sparepart" dan pilihan
// Kategori. Yang ADA di tabel tetapi TIDAK di sini, karena keduanya diturunkan sistem:
//
//	PART_SECTION_ID  kunci baris; diterbitkan saat penambahan, lihat IDSource
//	APPROVAL         selalu StatusPending pada penyimpanan lewat layar ini
type Input struct {
	Name string

	// CategoryID adalah kategori yang dipilih dari dropdown "---PILIH KATEGORI---".
	//
	// Yang dikirim layar adalah ID-nya, bukan namanya. Itu mengikuti layar lama:
	// dropdown-nya menyimpan `.District` — alias untuk `PART_CATEGORY_ID` — dan menampilkan
	// `.DistrictID`, alias untuk `PART_CATEGORY_NAME`. Lihat lookup.go untuk kedua alias
	// yang menyesatkan itu.
	CategoryID string
}

// MaxNameLength adalah panjang maksimum nama tipe.
//
// ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun dibaca dari DDL:
// `POOLDATA.GCNM_M_SPAREPART_TYPE` tidak ada DDL-nya di export (`R-08`), dan layar lamanya
// tidak memasang satu pun `pyMaxLength`.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Seratus dipilih agar sama dengan `masterkategorisparepart.MaxNameLength` dan
// `mastersparepart.MaxNameLength`: nilai kolom ini dipakai sebagai label pada layar Master
// Sparepart, dan tiga batas yang berbeda pada tiga layar bertetangga hanya akan
// membingungkan.
//
// Angka yang sama diulang di `PartTypeForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji di
// mastertipesparepart_test.go.
const MaxNameLength = 100

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("mastertipesparepart: tipe sparepart tidak ditemukan")

	// ErrNameTaken: PART_SECTION_NAME yang akan disimpan sudah dipakai baris lain.
	//
	// Padanan `Activity/ValidateMasterTipeSparepart`, yang membandingkan nama yang diketik
	// terhadap `upper(PART_SECTION_NAME)` lewat `RDB List/ValidationSparepartType-SQL.xml`
	// lalu menyusun pesan galat.
	//
	// PERHATIKAN CAKUPANNYA, DAN INILAH YANG PALING MUDAH MENGEJUTKAN DI MODUL INI. Rule itu
	// TIDAK menyaring APPROVAL **dan TIDAK menyaring PART_CATEGORY_ID**:
	//
	//	select PART_SECTION_NAME from POOLDATA.gcnm_m_sparepart_type
	//	 where upper(PART_SECTION_NAME) = {TempValidateTypeSparepart.CaseID}
	//
	// Akibatnya nama tipe harus unik **di seluruh tabel**, bukan di dalam satu kategori:
	// "KACA DEPAN" tidak dapat ada sekaligus di kategori BODY dan KABIN. Nama yang pernah
	// DITOLAK pun tetap memblokir selamanya.
	//
	// Keduanya ditiru apa adanya atas keputusan Work Owner 2026-09-21 (`P-5`: perilaku
	// dipertahankan lebih dulu, diperbaiki kemudian). Perbaikannya — keunikan per kategori —
	// dicatat sebagai Future Enhancement, bukan dikerjakan sambil jalan.
	ErrNameTaken = errors.New("mastertipesparepart: nama tipe sparepart sudah dipakai")

	// ErrCategoryNotFound: kategori yang dipilih tidak ada, atau tidak berstatus disetujui.
	//
	// TIDAK ada padanannya di sistem lama — layar lama hanya menawarkan kategori dari
	// daftarnya sendiri dan tidak pernah memeriksa ulang saat menyimpan. Pemeriksaan ini
	// DITAMBAHKAN karena API dapat ditembak tanpa melewati layar, dan tipe yang menunjuk
	// kategori yang tidak ada akan tampil tanpa nama kategori di setiap grid yang
	// menampilkannya. Lihat usecase.Service.Create.
	ErrCategoryNotFound = errors.New("mastertipesparepart: kategori sparepart tidak ditemukan")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("mastertipesparepart: status persetujuan tidak dikenal")
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
// `P-5` menuntut seluruh pelanggaran dikirim bersamaan, meniru `InputRegister_act` yang
// menampilkan semua pesan galat dalam satu kali. Pada modul berisi dua isian wajib,
// bentuknya benar-benar terpakai: pengguna yang mengosongkan keduanya melihat keduanya
// disorot sekaligus.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data — keunikan
// nama dan keberadaan kategori — supaya galatnya sampai ke layar dalam bentuk yang SAMA
// dengan pelanggaran isian lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "mastertipesparepart: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Nama TIDAK di-UPPERCASE. Rule validasinya memang membandingkan `upper(...)`, tetapi yang
// di-uppercase di sana adalah PEMBANDINGNYA — bukan nilai yang disimpan. Memaksa huruf
// besar akan mengubah tampilan setiap baris yang disunting, dan itu selisih yang tidak
// diminta siapa pun. Perlakuan yang sama dipakai Master Kategori Sparepart.
func (i Input) Clean() Input {
	return Input{
		Name:       strings.TrimSpace(i.Name),
		CategoryID: strings.TrimSpace(i.CategoryID),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// Layar lamanya TIDAK menandai satu pun isian `pyRequired`, dan
// `Activity/UpdateTypeSparepart_act2` tetap menyimpan apa pun yang diketik — termasuk
// kosong. Kedua kewajiban di bawah karena itu **DITAMBAHKAN** terhadap sistem lama, dan
// alasannya bukan kerapian:
//
//   - **Nama kosong** akan muncul sebagai baris kosong pada dropdown Tipe di layar Master
//     Sparepart — tidak dapat dibedakan dari "belum memilih", dan tidak dapat dipilih ulang
//     setelah salah pilih.
//   - **Kategori kosong** membuat tipe menggantung tanpa induk. Grid layar ini
//     menampilkan kolom Kategori untuk setiap baris, dan baris tanpa induk akan selamanya
//     menampilkan kolom kosong yang tidak dapat dijelaskan dari layar mana pun.
//
// Baris lama yang sudah kosong tetap DIBACA apa adanya; penolakan hanya terjadi saat
// barisnya disimpan ulang.
//
// # Yang TIDAK diperiksa di sini
//
// Keberadaan kategori yang dipilih. Ia menuntut pembacaan tabel lain, dan domain tidak
// boleh menyentuh basis data — pemeriksaannya ada di usecase.Service.
func (i Input) Check() error {
	var violation []Violation

	switch {
	case i.Name == "":
		violation = append(violation, Violation{
			Field:   "nama_tipe_sparepart",
			Message: "Nama tipe sparepart wajib diisi.",
		})
	case len(i.Name) > MaxNameLength:
		violation = append(violation, Violation{
			Field: "nama_tipe_sparepart",
			Message: fmt.Sprintf("Nama tipe sparepart paling panjang %d karakter.",
				MaxNameLength),
		})
	}

	if i.CategoryID == "" {
		violation = append(violation, Violation{
			Field:   "id_kategori_sparepart",
			Message: "Kategori sparepart wajib dipilih.",
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/MasterTipeSparepartHE-Section.xml`, yang ketiganya
// membaca tabel yang sama dan hanya berbeda pada nilai APPROVAL-nya.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nama tipe DAN nama kategorinya.
	//
	// DITAMBAHKAN terhadap sistem lama, yang memuat seluruh baris ke klipboard lalu
	// menyaringnya di peramban (`pyPageSize=50`, tanpa satu pun kotak pencarian di
	// section-nya). Kosong berarti tanpa penyaring.
	//
	// Nama KATEGORI ikut dicari, dan itu berbeda dari Master Kategori Sparepart yang hanya
	// mencari satu kolom. Alasannya: kategori adalah cara pengguna mengelompokkan tipe di
	// kepalanya — "apa saja tipe di HYDRAULIC" adalah pertanyaan yang wajar, dan tanpa ini
	// ia hanya dapat dijawab dengan memindai seluruh daftar.
	Keyword string
}

// IDSource menerbitkan ID baru.
//
// Ia seam tersendiri, bukan method pada Repo, dengan alasan yang sama seperti pada modul
// master lain: yang dikerjakannya bukan urusan tipe melainkan urusan **penomoran**.
//
// # Bentuknya, dibaca dari RDB List/InsertMasterSparepartType_sql-SQL.xml
//
//	insert into POOLDATA.gcnm_m_sparepart_type
//	       (PART_SECTION_ID, PART_SECTION_NAME, PART_CATEGORY_ID, APPROVAL)
//	Values ((select nvl(max(PART_SECTION_ID),0)+1 from POOLDATA.gcnm_m_sparepart_type),
//	        {InputKategori.CITY_ID}, {InputKategori.DISC_JASA}, {InputKategori.NO_ACCOUNT})
//
// **Angka berurut biasa** — tanpa kode situs, tanpa sequence, dan tanpa nol di depan. Sama
// persis dengan Master Kategori Sparepart, dan berbeda dari Master Sparepart serta Master
// Panel yang keduanya memakai `M_SITE_DATABASE` ditambah sequence. Bentuknya ditiru apa
// adanya: menerbitkan kunci berbentuk lain akan membuat baris baru tidak sebentuk dengan
// baris yang sudah ada, dan nilai itu disalin ke `SPAREPART_HE.TIPE_SPART` yang sudah
// berisi angka-angka lama.
//
// # Balapan yang ADA di sistem lama, dan bagaimana ia ditutup
//
// `max(...)+1` di dalam satu INSERT tidak menghalangi dua penyimpanan bersamaan membaca
// nilai maksimum yang sama lalu menyisipkan ID kembar. Sistem lama tidak menjaganya sama
// sekali.
//
// Ditutup dengan penguncian tabel, mengikuti keputusan yang sama pada Master Kategori
// Sparepart (Work Owner, 2026-09-21): bentuk ID-nya tetap sama persis dengan Pega (`P-5`),
// dan modulnya tidak tertahan menunggu prosedur perubahan skema (`D-63`). Pelaksanaannya
// ada di repo — lihat `type_next_id` pada berkas .sql.
//
// KETERBATASAN YANG DISADARI sama seperti di sana: penguncian menutup balapan antar-penulis
// yang melewati basis data ini, tetapi tidak menutup baris kembar yang sudah terlanjur ada.
// Penutupnya constraint unik, yang menunggu DDL (`R-08`).
type IDSource interface {
	// NextID mengembalikan ID berikutnya.
	//
	// Nilainya HANYA sah di dalam transaksi yang menerbitkannya — lihat Repo.Insert, yang
	// menerbitkan dan menyisipkan dalam satu transaksi. Memanggilnya di luar itu
	// mengembalikan angka yang dapat basi sebelum dipakai.
	NextID(ctx context.Context) (string, error)
}

// Repo adalah seam ke penyimpanan master tipe sparepart SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring, beserta nama kategori induknya.
	List(ctx context.Context, filter Filter) ([]PartType, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/BrowseSparepartTypeClaimHE_sql-SQL.xml`, yang dipakai
	// `Activity/SetMasterTipeSparepart_act` untuk memuat baris ke form.
	Get(ctx context.Context, id string) (PartType, error)

	// FindByName mencari baris menurut PART_SECTION_NAME-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationSparepartType-SQL.xml`, yang mencocokkan
	// `upper(PART_SECTION_NAME)` — perlakuan yang ditiru apa adanya, termasuk TIDAK
	// menyaring APPROVAL dan TIDAK menyaring kategori. Lihat ErrNameTaken.
	FindByName(ctx context.Context, name string) (PartType, error)

	// Insert menyisipkan baris baru dan mengembalikan baris yang benar-benar tersimpan.
	//
	// ID pada argumen DIABAIKAN: ia diterbitkan di dalam operasi ini, di dalam transaksi
	// yang sama dengan penyisipannya. Memisahkannya menjadi "terbitkan" lalu "sisip" di
	// lapisan aplikasi akan membuka kembali balapan yang justru sedang ditutup — lihat
	// IDSource.
	//
	// Pemeriksaan keunikan nama juga berada DI DALAM operasi ini, dengan alasan yang sama.
	// Sistem lama memecahnya — `ValidateMasterTipeSparepart` dipanggil lebih dulu,
	// penyisipannya menyusul — dan jarak di antara keduanya tidak dijaga apa pun.
	//
	// CategoryName pada argumen maupun pada nilai baliknya TIDAK ditulis: ia milik tabel
	// kategori. Yang dikembalikan mengisinya dari hasil pembacaan ulang bila ada; lihat
	// pengisi masing-masing.
	Insert(ctx context.Context, t PartType) (PartType, error)

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	//
	// Padanan `RDB List/UpdateMasterSparepartType_sql2-SQL.xml`, yang menulis
	// PART_SECTION_NAME, PART_CATEGORY_ID, dan APPROVAL sekaligus.
	Update(ctx context.Context, t PartType) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus.
	//
	// # Rule yang menjalankannya TIDAK ADA di export
	//
	// `Activity/UpdateTipeSparepart_act` menerima kunci dan status lalu memanggil sebuah
	// rule UPDATE — dan rule itu **tidak ada di antara 2.634 berkas export** (`R-16`). Tipe
	// juga tidak ikut `Activity/SetApprovalAllMaster`, yang hanya melayani M_BENGKEL_HE,
	// M_PANEL_HE, dan M_SPAREPART_HE.
	//
	// Bentuk pernyataannya karena itu **DIREKONSTRUKSI**, bukan dibaca: ia menulis kolom
	// APPROVAL pada baris yang ditunjuk PART_SECTION_ID, mengikuti
	// `UpdateMasterSparepartType_sql2` yang menulis ketiga kolom pada tabel yang sama.
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
// Ketiganya tetap DIDEKLARASIKAN terpisah — Repo untuk tabelnya, LookupRepo untuk tabel
// acuan milik modul lain, IDSource untuk penomoran — karena ketiganya menjawab pertanyaan
// yang berbeda dan dapat berubah sendiri-sendiri. Yang disatukan hanyalah CARA MEMILIHNYA:
// ketiganya selalu berasal dari koneksi entitas yang sama.
//
// IDSource tetap ada di sini meski Insert menerbitkan ID-nya sendiri: ia dipakai
// `claimpnc -periksa` untuk membuktikan penerbitan kunci bekerja terhadap basis data nyata
// TANPA menyisipkan satu baris pun.
type Store interface {
	Repo
	LookupRepo
	IDSource
}
