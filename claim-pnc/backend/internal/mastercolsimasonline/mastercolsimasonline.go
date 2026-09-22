// Package mastercolsimasonline adalah inti modul Master COL Simas Online (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// COL adalah singkatan **Cause of Loss** — penyebab kerugian (`CONTEXT.md`). Sistem lama
// menyimpannya berjenjang dua tingkat, dan modul ini mengerjakan tingkat SATU:
//
//	Master COL   POOLDATA.M_CAUSE_OF_LOSS   <- yang dikerjakan paket ini
//	  └─ Detail COL  POOLDATA.D_CAUSE_OF_LOSS  <- anak, merujuk M_COL_ID
//
// Yang membuatnya "Simas Online" bukan tabelnya — tabelnya sama — melainkan **layarnya**.
// Layar Simas Online menampilkan dua hal yang tidak ada di layar Master COL biasa:
// **ID Master Kerugian** (`MST_COL_ID`, kunci padanan di sistem Simas Online) dan
// **daftar Bisnis** yang memakai penyebab kerugian itu.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/CauseOfLossInboxSimasOnline-Harness.xml  layar yang diganti modul ini
//	Section/Online_GridCauseOfLoss-Section.xml        judul, grid ID + Description,
//	                                                  tombol Tambah & Refresh
//	Section/Online_BrowseCauseOfLoss-Section.xml      form 4 isian; BISNISID terbukti
//	                                                  pyPageListProperty (banyak baris)
//	Activity/PageNewForSetSimasOnlineCOL-Act.xml      tombol Tambah -> form kosong
//	Activity/SetDataCauseofflossOnline-Act.xml        klik baris -> muat ke form
//	Activity/Online_nsertCauseOfLoss_act-Act.xml      Simpan; M_COL_ID kosong => baru
//	Database/PEGA_M_CAUSE_OF_LOSS.prc                 bentuk kode dan tabel sebenarnya
//	RDB List/GetLBUID_SQL-SQL.xml                     pemetaan COL -> Bisnis, dan
//	                                                  BUSINESS(ID, NOTE) sebagai sumber
//
// # Kenapa "Bisnis" berupa DAFTAR, bukan satu nilai
//
// Bukan tafsiran. `Section/Online_BrowseCauseOfLoss-Section.xml:5752` mendeklarasikan
// `<pyPageListProperty>TempCauseOfLoss.BISNISID</pyPageListProperty>` dengan
// `pyPageListPropertyClass = ASM-FW-GISFW-Int-BUSINESS` — sebuah repeat grid, bukan
// isian tunggal. Bentuk yang sama terbaca di tingkat detail:
// `RDB List/GetLBUID_SQL-SQL.xml` membaca `V_D_CAUSE_OF_LOSS_BUSINESS (D_COL_ID,
// BISNISID)` lalu men-join-nya ke `BUSINESS (ID, NOTE)`.
//
// # Bisnis dimiliki tim lain
//
// Kelas `ASM-FW-GISFW-Int-BUSINESS` berada di ruleset **GISFW**, yang `D-03` tetapkan
// dikembangkan tim lain. Modul ini karena itu hanya MEMBACA daftar bisnis dan tidak
// pernah menulisnya — seam BusinessRepo sengaja tidak punya operasi tulis sama sekali,
// sehingga batas kepemilikannya ditegakkan tipe, bukan disiplin.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastercolsimasonline

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

// Nama isian yang dipakai layar untuk menyorot pelanggaran.
//
// Nilainya sama dengan nama field JSON pada permintaan, sehingga layar tidak perlu
// memetakan apa pun: apa yang dikirim salah adalah apa yang disorot.
const (
	FieldDescription = "nama"
	FieldMasterCode  = "id_master_kerugian"
	FieldBusiness    = "bisnis"
)

// Batas panjang ketiga isian teks.
//
// # PERINGATAN — ketiga angka ini PENJAGA, bukan lebar kolom yang diketahui
//
// DDL `POOLDATA.M_CAUSE_OF_LOSS` tidak ada di export (`R-08`), dan kolom COL_DESC serta
// MST_COL_ID belum pernah dilihat katalognya. Layar Pega sendiri tidak membatasi apa pun
// — `Section/Online_BrowseCauseOfLoss-Section.xml` tidak memuat `pyMaxLength` pada isian
// mana pun.
//
// Angka di bawah karena itu dipilih mengikuti modul Master Status Klaim yang lebar
// kolomnya sudah diketahui (VARCHAR2(100)), sebagai penjaga yang menolak lebih awal
// daripada dibiarkan basis data menolaknya dengan ORA-12899 yang tidak dapat dibaca
// pengguna.
//
// KETIGANYA WAJIB DISESUAIKAN begitu DBA mengirimkan DDL-nya, dan berubahnya harus
// serentak dengan `CauseOfLossForm.tsx` di frontend. Uji di `mastercolsimasonline_test.go`
// yang menjaga duplikasi itu tetap terlihat.
const (
	MaxDescriptionLength = 100

	// MaxMasterCodeLength membatasi MST_COL_ID. Sejak 2026-09-21 isinya adalah M_COL_ID
	// baris lain (lihat CauseOfLoss.MasterCode), sehingga panjangnya terikat lebar kode —
	// bukan lebar teks bebas. Penjaga ini sengaja longgar sampai lebar M_COL_ID diketahui.
	MaxMasterCodeLength = 20

	// MaxBusinessNameLength membatasi nama bisnis yang diketik bebas.
	//
	// Ia ada KARENA nama bebas diterima (`pyAllowFreeFormInput=true`): nama yang dipilih
	// dari master pasti muat di kolomnya, sedangkan nama yang diketik sendiri tidak ada
	// yang menjamin.
	MaxBusinessNameLength = 100
)

// CauseOfLoss adalah satu baris master penyebab kerugian beserta pemetaan bisnisnya.
type CauseOfLoss struct {
	// Code adalah M_COL_ID. Diterbitkan penyimpanan saat baris ditambahkan dan TIDAK
	// PERNAH berubah sesudahnya — layar Pega pun menandainya `pyReadOnly=true`
	// (`Section/Online_BrowseCauseOfLoss-Section.xml:4335` dan sekitarnya).
	Code string

	// Description adalah COL_DESC, berlabel "Nama Cause of loss" di layar.
	Description string

	// MasterCode adalah MST_COL_ID, berlabel "ID Master Kerugian" di layar.
	//
	// # Ia RUJUKAN-DIRI, bukan kode dari sistem sebelah
	//
	// Dugaan pertama — bahwa isinya kode padanan di sistem Simas Online — TERBANTAH oleh
	// export. `Section/Online_BrowseCauseOfLoss-Section.xml:4594-4604` menunjukkan
	// isiannya berupa daftar yang sumbernya `BrowseVMCauseOfLoss_RD` ber-`pyAppliesTo`
	// `ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`, dengan `pyValue = .M_COL_ID` dan
	// `pyPrompt = .COL_DESC`.
	//
	// Artinya isinya adalah **Code milik baris LAIN di master yang sama** — penyebab
	// kerugian ini bernaung di bawah penyebab kerugian itu. Ditetapkan Work Owner
	// 2026-09-21.
	//
	// Boleh kosong: layar Pega tidak mewajibkannya, dan baris tingkat atas memang tidak
	// punya induk.
	MasterCode string

	// Businesses adalah daftar bisnis yang memakai penyebab kerugian ini.
	//
	// Selalu terisi saat dibaca lewat Get; pada List ia sengaja dibiarkan kosong —
	// lihat Repo.List.
	Businesses []Business
}

// Business adalah satu lini bisnis, dibaca dari POOLDATA.BUSINESS.
//
// Hanya dua kolom yang dipakai, dan keduanya terbukti dari `GetLBUID_SQL-SQL.xml`:
// `select ID, NOTE as "Note" from V_D_CAUSE_OF_LOSS_BUSINESS a, BUSINESS b
// where a.BISNISID = b.ID`.
type Business struct {
	// ID adalah BUSINESS.ID, tersimpan sebagai BISNISID.
	//
	// BOLEH KOSONG, dan itu bukan cacat. Isian Bisnis di Pega ber-
	// `pyAllowFreeFormInput=true` (`Online_BrowseCauseOfLoss-Section.xml:6089`),
	// sehingga petugas dapat mengetik nama yang tidak ada di master — dan baris itu
	// tersimpan tanpa ID. Work Owner menetapkan 2026-09-21 perilaku itu dipertahankan.
	//
	// Akibatnya: kolom ini TIDAK dapat dipercaya sebagai rujukan yang selalu ketemu saat
	// di-join ke POOLDATA.BUSINESS. Setiap pembacanya wajib menyiapkan hasil kosong.
	ID string

	// Name adalah nama bisnis — BUSINESS.NOTE bila dipilih dari master, atau apa yang
	// diketik petugas bila tidak.
	//
	// Inilah yang benar-benar terikat di layar Pega: sel gridnya `pyValue = .Note`,
	// sedangkan `.ID` hanya kolom tersembunyi yang ikut terisi saat dipilih dari daftar
	// (`pySetValueOnSelect=true`). Karena itu Name yang menjadi identitas baris pemetaan,
	// bukan ID.
	Name string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Ia sengaja terpisah dari CauseOfLoss: Code tidak pernah berasal dari pengguna. Pada
// penambahan ia diterbitkan penyimpanan; pada penyuntingan ia diambil dari jalur URL.
type Input struct {
	Description string

	// MasterCode adalah Code baris lain yang menjadi induk. Kosong berarti tanpa induk.
	MasterCode string

	// BusinessNames adalah NAMA bisnis, bukan ID-nya.
	//
	// Nama yang dikirim, bukan ID, karena itulah yang benar-benar diketik dan dilihat
	// petugas di layar Pega (`pyValue = .Note`), dan karena nama yang diketik bebas
	// memang tidak punya ID. ID-nya diselesaikan di sisi server dengan mencocokkan nama
	// ke master — cara yang sama dengan autocomplete Pega yang mengisi `.ID` saat sebuah
	// pilihan diambil dari daftar.
	//
	// Nama yang tidak ada di master TETAP diterima dan disimpan tanpa ID.
	BusinessNames []string
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("mastercolsimasonline: penyebab kerugian tidak ditemukan")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field memakai nama isian yang dikenali layar, bukan nama kolom basis data.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`). `InputRegister_act` sistem lama
// memeriksa belasan aturan lalu menampilkan semuanya bersamaan; mengembalikan satu galat
// per percobaan akan membuat pengguna menekan Simpan berkali-kali hanya untuk menemukan
// kesalahan berikutnya (`12-CROSSCUTTING.md` §1.2 butir 1).
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violation))
	for _, p := range e.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "mastercolsimasonline: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian dan membuang ID bisnis kosong.
//
// Dipisahkan dari Check supaya yang tersimpan adalah nilai yang sudah dipangkas — bukan
// nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Urutan ID bisnis DIPERTAHANKAN apa adanya, tidak diurutkan ulang. Pengguna menyusun
// barisnya sendiri di grid, dan mengurutkannya diam-diam akan membuat layar menampilkan
// urutan yang berbeda dari yang baru saja ia simpan.
func (i Input) Clean() Input {
	businessNames := make([]string, 0, len(i.BusinessNames))
	for _, name := range i.BusinessNames {
		if trimmed := strings.TrimSpace(name); trimmed != "" {
			businessNames = append(businessNames, trimmed)
		}
	}

	return Input{
		Description:   strings.TrimSpace(i.Description),
		MasterCode:    strings.TrimSpace(i.MasterCode),
		BusinessNames: businessNames,
	}
}

// Check menjalankan seluruh aturan isian yang dapat diperiksa TANPA menyentuh
// penyimpanan, dan mengembalikan semua pelanggarannya.
//
// Nil berarti isian sah sejauh yang dapat diperiksa di sini. Input sudah harus melewati
// Clean lebih dulu.
//
// Keberadaan setiap ID bisnis TIDAK diperiksa di sini: ia menuntut membaca master
// bisnis, sedangkan fungsi ini murni dan dapat diuji tanpa apa pun. Pemeriksaannya ada
// di usecase, mengikuti pemisahan yang sama pada masterstatus.CheckLabel.
func (i Input) Check() error {
	var violation []Violation

	// NAMA TIDAK DIWAJIBKAN, dan itu hasil pemeriksaan ulang ke export — bukan kelalaian.
	//
	// Versi pertama modul ini mewajibkannya sebagai perbaikan terencana. Work Owner
	// menetapkan 2026-09-21 layar disamakan dengan Pega, dan pemeriksaan ulang
	// membuktikan Pega memang tidak memvalidasi apa pun di sini:
	//
	//	Section/Online_BrowseCauseOfLoss-Section.xml   pyRequired=false pada SELURUH 19 isian
	//	Activity/Online_nsertCauseOfLoss_act-Act.xml   nol Page-Validate, nol Property-Validate
	//	(export)                                       nol Rule-Obj-Validate untuk kelas ini
	//
	// Akibat yang harus disadari, dan sengaja diterima: penyebab kerugian tanpa nama
	// dapat tersimpan, dan baris itu akan muncul sebagai pilihan KOSONG di setiap
	// dropdown penyebab kerugian. Itu perilaku sistem lama, dan `P-5` menetapkan
	// perilaku dipertahankan lebih dulu.
	//
	// Yang tetap diperiksa hanyalah PANJANGNYA — dan itu bukan aturan bisnis melainkan
	// penjaga teknis: nilai yang melampaui lebar kolom akan ditolak basis data dengan
	// ORA-12899, galat yang muncul di layar sebagai 500 dan tidak dapat dibaca pengguna.
	//
	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if utf8.RuneCountInString(i.Description) > MaxDescriptionLength {
		violation = append(violation, Violation{
			Field:   FieldDescription,
			Message: "Nama Cause of loss paling panjang " + itoa(MaxDescriptionLength) + " karakter.",
		})
	}

	// MST_COL_ID sengaja TIDAK diwajibkan — layar Pega pun tidak, dan baris tingkat atas
	// memang tidak punya induk. Yang diperiksa di sini hanya panjangnya; keberadaan
	// induknya menuntut membaca penyimpanan dan karena itu ada di usecase.
	if utf8.RuneCountInString(i.MasterCode) > MaxMasterCodeLength {
		violation = append(violation, Violation{
			Field:   FieldMasterCode,
			Message: "ID Master Kerugian paling panjang " + itoa(MaxMasterCodeLength) + " karakter.",
		})
	}

	for _, name := range i.BusinessNames {
		if utf8.RuneCountInString(name) > MaxBusinessNameLength {
			violation = append(violation, Violation{
				Field:   FieldBusiness,
				Message: "Nama bisnis paling panjang " + itoa(MaxBusinessNameLength) + " karakter: " + name,
			})
			break
		}
	}

	// BISNIS KEMBAR TIDAK DITOLAK.
	//
	// Versi pertama modul ini menolaknya. Pemeriksaan ulang 2026-09-21 membuktikan grid
	// Pega tidak punya satu pun penanda keunikan — tidak ada `pyUnique`, tidak ada
	// validasi, tidak ada apa pun — sehingga satu bisnis memang dapat dipilih dua kali.
	// Work Owner menetapkan layar disamakan dengan Pega.
	//
	// Akibatnya mengikat bentuk penyimpanan: nama bisnis TIDAK dapat menjadi kunci baris
	// pemetaan, dan yang menjadi kunci adalah POSISINYA di dalam grid. Lihat
	// repo/sqlstore dan migrasi 0004.

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// NormalizeBusinessName menyusun bentuk nama bisnis yang dipakai saat mencocokkannya ke
// master.
//
// Pencocokan mengabaikan besar-kecil huruf dan spasi tepi. Alasannya ada presedennya di
// repo ini: `11-SECURITY.md` §3.1 mencatat nama access group Pega muncul dalam dua
// kapitalisasi karena perbandingan rule lama tidak konsisten soal itu — dan di sini
// namanya bahkan boleh diketik bebas, sehingga "Aneka" dan "ANEKA" pasti terjadi.
//
// Ia TIDAK dipakai untuk menolak kembar: bisnis yang sama boleh dipilih dua kali, sama
// seperti di Pega. Yang dikerjakannya hanyalah memastikan "aneka" yang diketik pengguna
// tetap menemukan ID bisnis "ANEKA" di master.
//
// Diekspor supaya pemakainya di lapisan lain memakai bentuk yang sama persis; dua tempat
// yang menormalkan dengan cara berbeda akan menghasilkan ID yang kadang ketemu kadang
// tidak.
func NormalizeBusinessName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain
// hanya untuk dua pesan. Pola yang sama dipakai masterstatus.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Repo adalah seam ke penyimpanan master COL SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di
// tingkat kueri (`ADR-0030` Opsi 1). Yang memilih instans mana yang dipakai satu
// permintaan adalah RepoSelector.
//
// Tidak ada operasi hapus, dan itu bukan kelalaian: `D-66` menetapkan tidak ada
// penghapusan fisik pada data bernilai bisnis, dan sistem lama pun tidak punya satu pun
// pernyataan DELETE terhadap M_CAUSE_OF_LOSS. Baris ini dirujuk D_CAUSE_OF_LOSS.M_COL_ID
// pada data yang sudah berjalan.
type Repo interface {
	// List mengembalikan seluruh baris, terurut seperti kueri lama.
	//
	// Businesses pada setiap baris sengaja DIBIARKAN KOSONG. Grid di layar hanya
	// menampilkan ID dan Description (`Section/Online_GridCauseOfLoss-Section.xml`),
	// dan memuat pemetaan bisnis seluruh baris berarti satu kueri tambahan yang
	// hasilnya tidak pernah dilihat siapa pun.
	List(ctx context.Context) ([]CauseOfLoss, error)

	// Get mengembalikan satu baris LENGKAP dengan pemetaan bisnisnya;
	// ErrNotFound bila tidak ada.
	Get(ctx context.Context, code string) (CauseOfLoss, error)

	// Insert menerbitkan kode baru, menyimpan barisnya beserta pemetaan bisnisnya, lalu
	// mengembalikan baris yang benar-benar tersimpan.
	//
	// Penerbitan kode berada di dalam satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi: keduanya harus berada dalam satu
	// transaksi, dan batas transaksi tidak dapat digambar dari luar seam ini.
	Insert(ctx context.Context, data SaveData) (CauseOfLoss, error)

	// Update menyimpan perubahan pada baris yang sudah ada, termasuk menggantikan
	// seluruh pemetaan bisnisnya; ErrNotFound bila barisnya hilang di antara pemuatan
	// layar dan penyimpanan.
	//
	// Code tidak pernah ikut berubah.
	Update(ctx context.Context, code string, data SaveData) (CauseOfLoss, error)
}

// SaveData adalah isi yang benar-benar disimpan — sudah bersih, sudah lolos pemeriksaan,
// dan nama bisnisnya sudah diselesaikan menjadi pasangan nama dan ID.
//
// Ia terpisah dari Input dengan sengaja. Input adalah apa yang DIKIRIM pengguna; SaveData
// adalah apa yang DISIMPAN. Di antara keduanya ada langkah yang menuntut membaca master
// bisnis, dan memakai satu tipe untuk keduanya akan menyembunyikan langkah itu — sehingga
// repo tampak boleh dipanggil dengan nama mentah, padahal tidak.
type SaveData struct {
	Description string
	MasterCode  string

	// Businesses sudah berurutan sesuai susunan pengguna. Name selalu terisi; ID boleh
	// kosong untuk nama yang diketik bebas dan tidak ada di master.
	Businesses []Business
}

// BusinessRepo adalah seam BACA-SAJA ke master bisnis milik GISFW.
//
// Ketiadaan operasi tulis di sini disengaja dan merupakan penegakan `D-03`: data bisnis
// dimiliki tim lain, dan modul ini tidak boleh menulisnya. Batas itu ditegakkan oleh
// bentuk antarmuka, bukan oleh ingatan orang yang menulis kode berikutnya.
type BusinessRepo interface {
	// List mengembalikan seluruh bisnis, terurut menurut namanya.
	List(ctx context.Context) ([]Business, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada
// saat permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// BusinessRepoSelector memilih BusinessRepo milik satu portal entitas.
//
// Terpisah dari RepoSelector meski keduanya selalu dipilih bersamaan, karena keduanya
// mengisi seam yang berbeda: yang satu tabel milik modul ini, yang lain tabel milik
// GISFW yang hanya dibaca.
type BusinessRepoSelector func(portalAlias string) (BusinessRepo, error)
