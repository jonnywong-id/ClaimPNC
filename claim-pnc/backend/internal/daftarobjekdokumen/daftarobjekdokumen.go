// Package daftarobjekdokumen adalah inti modul Daftar Objek Dokumen (`F-4`, MENU_ID 43).
//
// # Apa yang dimodelkan di sini
//
// Objek Dokumen adalah data acuan yang menyatakan DOKUMEN ITU MELEKAT PADA APA — objek
// pertanggungan mana, atau pihak mana. Ia dirujuk master lain, bukan berdiri sendiri:
//
//	POOLDATA.LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID  <- merujuk baris yang dikelola paket ini
//
// Rujukan itu terbaca apa adanya di `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:26`, dan
// itulah sebab ID sebuah baris TIDAK PERNAH boleh berubah dan barisnya tidak pernah
// dihapus: keduanya akan memutus baris "Detail Tipe Dokumen per Bisnis" yang bernaung di
// bawahnya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ListDocumentObject-Harness.xml        layar yang diganti modul ini
//	Section/ListDocumentObject-Section.xml         bingkai layar, tombol Tambah & Refresh
//	Section/BrowseDocumentObject-Section.xml       grid dua kolom, form "Memperbaharui
//	                                               Data", aksi baris "Ubah", tombol
//	                                               "Simpan", dan grid "ID Bisnis"
//	Report Definition/BrowseVLstDocObj_RD-RD.xml   sumber data: V_LST_DOC_OBJ dengan
//	                                               kolom ID, KET_DOC_OBJ, OLD_ID
//	Data Transform/CNMRefreshListDocumentObject_dt tombol Refresh memuat ulang halaman
//	RDB List/GetLBUID_SQL-SQL.xml                  bentuk pemetaan ke Bisnis, dan
//	                                               BUSINESS(ID, NOTE) sebagai sumbernya
//	Database/PEGA_LST_DOC_TYPE.prc                 bentuk penerbitan ID pada rumpun LST_*
//
// # Kenapa "Bisnis" berupa DAFTAR, bukan satu nilai
//
// Bukan tafsiran. `Section/BrowseDocumentObject-Section.xml:13720` mendeklarasikan
// `<pyPageListProperty>TempDocObj.LIST_LBU_ID</pyPageListProperty>` — sebuah repeat grid
// dengan tombol `addRow`, berjudul kolom "ID Bisnis", yang selnya terikat `.Note` berkelas
// `ASM-FW-GISFW-Int-BUSINESS`. Bentuk yang sama persis sudah dikerjakan modul Master COL
// Simas Online, dan sumber masternya pun sama.
//
// # Bisnis dimiliki tim lain
//
// Kelas `ASM-FW-GISFW-Int-BUSINESS` berada di ruleset **GISFW**, yang `D-03` tetapkan
// dikembangkan tim lain. Modul ini karena itu hanya MEMBACA daftar bisnis dan tidak pernah
// menulisnya — seam BusinessRepo sengaja tidak punya operasi tulis sama sekali, sehingga
// batas kepemilikannya ditegakkan bentuk antarmuka, bukan disiplin orang berikutnya.
//
// # Dua activity yang HILANG dari export, dan akibatnya
//
// Jalur simpan layar lama menunjuk `CNMInsertLstDocObj_act` (tombol Simpan) dan
// `SetsLstDocObjValue_act` (klik baris). **Keduanya tidak ada di export** (`R-16`), dan
// tidak ada pula `PEGA_LST_DOC_OBJ.prc` di `Database/`.
//
// Yang dapat dibaca tetap banyak — APA yang disimpan terbaca lengkap dari form dan dari
// Report Definition-nya. Yang tidak dapat dibaca hanyalah KE MANA. Perlakuannya sama
// dengan modul Daftar Detail Dokumen Travel yang menghadapi keadaan serupa: seluruh nama
// objek tulis DIISOLASI di `repo/sqlstore/daftarobjekdokumen.sql`, yang merupakan dugaan
// ditandai sebagai dugaan, dan migrasi ditulis sebagai daftar pertanyaan ke DBA.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package daftarobjekdokumen

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
	FieldDescription = "objek_dokumen"
	FieldBusiness    = "bisnis"
)

// Batas panjang kedua isian teks.
//
// Keduanya PENJAGA TEKNIS, bukan aturan bisnis. Lebar kolom KET_DOC_OBJ dan lebar kolom
// nama bisnis pada tabel pemetaan BELUM DIKETAHUI — DDL-nya belum diterima (`R-08`), dan
// tabel pemetaannya bahkan belum ada. Angka di bawah mengikuti modul Master COL Simas
// Online atas kolom yang sejenis, dan sengaja ditahan rendah: menaikkannya kelak tidak
// merusak apa pun, menurunkannya merusak.
//
// Tanpa penjaga ini, nilai yang melampaui lebar kolom akan ditolak Oracle dengan ORA-12899
// — galat yang sampai ke layar sebagai 500 dan tidak dapat dibaca pengguna.
//
// Keduanya harus berubah SERENTAK dengan `DocumentObjectForm.tsx` di frontend. Uji di
// `daftarobjekdokumen_test.go` yang menjaga duplikasi itu tetap terlihat.
const (
	MaxDescriptionLength  = 100
	MaxBusinessNameLength = 100
)

// DocumentObject adalah satu baris objek dokumen beserta pemetaan bisnisnya.
type DocumentObject struct {
	// ID adalah kolom ID. Diterbitkan penyimpanan saat baris ditambahkan dan TIDAK
	// PERNAH berubah sesudahnya — layar Pega pun menandai isiannya `pyReadOnly=true`
	// (`Section/BrowseDocumentObject-Section.xml:11465`).
	//
	// Ia dirujuk LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan.
	ID string

	// Description adalah kolom KET_DOC_OBJ.
	//
	// Berlabel "Daftar Objek Dokumen" di layar — sama dengan judul layarnya, dan itu
	// memang begitu di Pega (`:11801` untuk isian form, `:6715` untuk judul kolom grid).
	Description string

	// OldID adalah kolom OLD_ID: penomoran sebelum sistem ini dibangun.
	//
	// Tidak pernah diisi maupun diubah aplikasi, dan kosong pada baris yang tidak pernah
	// punya nomor lama.
	//
	// # Dikirim, tetapi TIDAK ditampilkan layar
	//
	// Dua artefak Pega yang berbeda diikuti pada tempatnya masing-masing, mengikuti
	// perlakuan yang sama pada modul Master Penyebab Kerugian:
	//
	//	lapisan data   BrowseVLstDocObj_RD          memuat OLD_ID
	//	lapisan layar  BrowseDocumentObject-Section grid hanya DUA kolom, tanpa OLD_ID
	//
	// Ia tetap dikirim karena lapisan datanya memang memuatnya, dan modul yang merujuk
	// objek dokumen kelak tidak perlu mengubah kontrak ini untuk menautkan data
	// historisnya.
	OldID string

	// Businesses adalah daftar bisnis yang memakai objek dokumen ini.
	//
	// Selalu terisi saat dibaca lewat Get; pada List ia sengaja dibiarkan kosong — lihat
	// Repo.List.
	Businesses []Business
}

// Business adalah satu lini bisnis, dibaca dari POOLDATA.BUSINESS.
//
// Hanya dua kolom yang dipakai, dan keduanya terbukti dari `RDB List/GetLBUID_SQL-SQL.xml`:
// `select ID, NOTE as "Note" from ... BUSINESS b where a.BISNISID = b.ID`.
type Business struct {
	// ID adalah BUSINESS.ID, tersimpan sebagai BISNISID.
	//
	// BOLEH KOSONG, dan itu bukan cacat. Sel grid Bisnis di Pega terikat ke `.Note` lewat
	// kontrol `pxAutoComplete` (`Section/BrowseDocumentObject-Section.xml:14465`), dan
	// `.ID` hanya kolom tersembunyi yang ikut terisi saat sebuah pilihan diambil dari
	// daftar. Nama yang diketik sendiri karena itu tersimpan tanpa ID.
	//
	// Akibatnya: kolom ini TIDAK dapat dipercaya sebagai rujukan yang selalu ketemu saat
	// di-join ke POOLDATA.BUSINESS. Setiap pembacanya wajib menyiapkan hasil kosong.
	ID string

	// Name adalah nama bisnis — BUSINESS.NOTE bila dipilih dari master, atau apa yang
	// diketik petugas bila tidak.
	//
	// Inilah yang benar-benar terikat di layar, sehingga Name yang menjadi identitas baris
	// pemetaan, bukan ID.
	Name string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Terpisah dari DocumentObject dengan sengaja: ID tidak pernah berasal dari pengguna. Pada
// penambahan ia diterbitkan penyimpanan; pada penyuntingan ia diambil dari jalur URL.
// OldID pun tidak ada di sini — ia jejak sejarah, bukan isian.
type Input struct {
	Description string

	// BusinessNames adalah NAMA bisnis, bukan ID-nya.
	//
	// Nama yang dikirim karena itulah yang diketik dan dilihat petugas di layar, dan
	// karena nama yang diketik bebas memang tidak punya ID. ID-nya diselesaikan di sisi
	// server dengan mencocokkan nama ke master — cara yang sama dengan autocomplete Pega
	// yang mengisi `.ID` saat sebuah pilihan diambil dari daftar.
	//
	// Nama yang tidak ada di master TETAP diterima dan disimpan tanpa ID.
	BusinessNames []string
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

	// Businesses sudah berurutan sesuai susunan pengguna. Name selalu terisi; ID boleh
	// kosong untuk nama yang diketik bebas dan tidak ada di master.
	Businesses []Business
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("daftarobjekdokumen: objek dokumen tidak ditemukan")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field memakai nama isian yang dikenali layar, bukan nama kolom basis data.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`). `InputRegister_act` sistem lama memeriksa
// belasan aturan lalu menampilkan semuanya bersamaan; mengembalikan satu galat per
// percobaan akan membuat pengguna menekan Simpan berkali-kali hanya untuk menemukan
// kesalahan berikutnya (`12-CROSSCUTTING.md` §1.2 butir 1).
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violation))
	for _, p := range e.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "daftarobjekdokumen: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian dan membuang nama bisnis kosong.
//
// Dipisahkan dari Check supaya yang tersimpan adalah nilai yang sudah dipangkas — bukan
// nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Urutan nama bisnis DIPERTAHANKAN apa adanya, tidak diurutkan ulang. Pengguna menyusun
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
		BusinessNames: businessNames,
	}
}

// Check menjalankan seluruh aturan isian yang dapat diperiksa TANPA menyentuh
// penyimpanan, dan mengembalikan semua pelanggarannya.
//
// Nil berarti isian sah sejauh yang dapat diperiksa di sini. Input sudah harus melewati
// Clean lebih dulu.
//
// # Kenapa nyaris tidak ada yang diperiksa
//
// Karena layar lama pun tidak memeriksa apa pun. Pemeriksaan langsung ke export:
//
//	Section/BrowseDocumentObject-Section.xml   pyRequired=false pada SELURUH isiannya
//	(export)                                   nol Rule-Obj-Validate untuk kelas
//	                                           ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ
//
// `P-5` menetapkan perilaku dipertahankan lebih dulu, dan Work Owner sudah menetapkan hal
// yang sama untuk dua modul sejenis (Master COL Simas Online 2026-09-21, Daftar Detail
// Dokumen Travel 2026-09-22). Akibat yang sengaja diterima: objek dokumen tanpa keterangan
// dapat tersimpan, dan baris itu akan muncul sebagai pilihan KOSONG di layar yang
// merujuknya.
//
// Keberadaan setiap nama bisnis TIDAK diperiksa di sini: ia menuntut membaca master
// bisnis, sedangkan fungsi ini murni dan dapat diuji tanpa apa pun. Penyelesaiannya ada di
// usecase — dan yang dikerjakan di sana bukan penolakan melainkan pelengkapan ID.
func (i Input) Check() error {
	var violation []Violation

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if utf8.RuneCountInString(i.Description) > MaxDescriptionLength {
		violation = append(violation, Violation{
			Field:   FieldDescription,
			Message: "Daftar Objek Dokumen paling panjang " + itoa(MaxDescriptionLength) + " karakter.",
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

	// BISNIS KEMBAR TIDAK DITOLAK, mengikuti grid Pega yang tidak punya satu pun penanda
	// keunikan — tidak ada `pyUnique`, tidak ada validasi. Perlakuan yang sama sudah
	// ditetapkan Work Owner untuk grid yang sebentuk pada Master COL Simas Online.

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
// Ia TIDAK dipakai untuk menolak kembar. Yang dikerjakannya hanyalah memastikan "aneka"
// yang diketik pengguna tetap menemukan ID bisnis "ANEKA" di master.
//
// Diekspor supaya pemakainya di lapisan lain memakai bentuk yang sama persis; dua tempat
// yang menormalkan dengan cara berbeda akan menghasilkan ID yang kadang ketemu kadang
// tidak.
func NormalizeBusinessName(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain hanya
// untuk dua pesan. Pola yang sama dipakai masterstatus dan mastercolsimasonline.
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

// Repo adalah seam ke penyimpanan objek dokumen SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (`ADR-0030` Opsi 1). Yang memilih instans mana yang dipakai satu permintaan adalah
// RepoSelector.
//
// Tidak ada operasi hapus, dan itu bukan kelalaian: `D-66` menetapkan tidak ada penghapusan
// fisik pada data bernilai bisnis, layar Pega pun tidak punya tombol hapus
// (`pyGridDeleteActivityExists=false` pada seluruh gridnya), dan barisnya dirujuk
// LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan.
type Repo interface {
	// List mengembalikan seluruh baris, terurut seperti kueri lama.
	//
	// Businesses pada setiap baris sengaja DIBIARKAN KOSONG. Grid di layar hanya
	// menampilkan ID dan keterangannya (`Section/BrowseDocumentObject-Section.xml`), dan
	// memuat pemetaan bisnis seluruh baris berarti satu kueri tambahan yang hasilnya tidak
	// pernah dilihat siapa pun.
	List(ctx context.Context) ([]DocumentObject, error)

	// Get mengembalikan satu baris LENGKAP dengan pemetaan bisnisnya;
	// ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (DocumentObject, error)

	// Insert menerbitkan ID baru, menyimpan barisnya beserta pemetaan bisnisnya, lalu
	// mengembalikan baris yang benar-benar tersimpan.
	//
	// Penerbitan ID berada di dalam satu operasi repo, bukan dipecah menjadi "ambil nomor"
	// lalu "sisip" di lapisan aplikasi: keduanya harus berada dalam satu transaksi, dan
	// batas transaksi tidak dapat digambar dari luar seam ini.
	Insert(ctx context.Context, data SaveData) (DocumentObject, error)

	// Update menyimpan perubahan pada baris yang sudah ada, termasuk menggantikan seluruh
	// pemetaan bisnisnya; ErrNotFound bila barisnya hilang di antara pemuatan layar dan
	// penyimpanan.
	//
	// ID tidak pernah ikut berubah.
	Update(ctx context.Context, id string, data SaveData) (DocumentObject, error)
}

// BusinessRepo adalah seam BACA-SAJA ke master bisnis milik GISFW.
//
// Ketiadaan operasi tulis di sini disengaja dan merupakan penegakan `D-03`: data bisnis
// dimiliki tim lain, dan modul ini tidak boleh menulisnya. Batas itu ditegakkan oleh bentuk
// antarmuka, bukan oleh ingatan orang yang menulis kode berikutnya.
type BusinessRepo interface {
	// List mengembalikan seluruh bisnis, terurut menurut namanya.
	List(ctx context.Context) ([]Business, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// BusinessRepoSelector memilih BusinessRepo milik satu portal entitas.
//
// Terpisah dari RepoSelector meski keduanya selalu dipilih bersamaan, karena keduanya
// mengisi seam yang berbeda: yang satu tabel milik modul ini, yang lain tabel milik GISFW
// yang hanya dibaca.
type BusinessRepoSelector func(portalAlias string) (BusinessRepo, error)
