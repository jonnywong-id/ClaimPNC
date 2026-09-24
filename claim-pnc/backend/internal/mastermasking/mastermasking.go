// Package mastermasking adalah inti modul Master Masking (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Data nasabah tertentu — **nomor KTP, alamat surel, dan nomor telepon** — ditampilkan
// TERSAMAR bagi kebanyakan petugas. Master ini menyatakan SIAPA yang boleh melihatnya
// tidak tersamar, PADA modul apa, dan SEBERAPA BANYAK data yang boleh ia cari serta
// lihat. Satu baris = satu pengguna pada satu cabang.
//
//	POOLDATA.MST_PROTEKSI_DATA_PNC   <- yang dikerjakan paket ini
//	POOLDATA.LOG_DATA_PROTEKSI_KLAIM <- jejak pemakaiannya; BUKAN lingkup paket ini
//
// Penegakan masking-nya sendiri — yang menyamarkan nilai di layar klaim dan memotong
// kuota — hidup di modul Proses Produksi dan belum dibangun. Paket ini hanya mengelola
// daftarnya, persis seperti harness `MasterProteksiVisibilityData` di sistem lama.
//
// # Kenapa ini bukan master biasa
//
// Isinya adalah KEWENANGAN MELIHAT DATA PRIBADI. Satu baris yang salah membuat seorang
// petugas dapat membaca nomor KTP dan nomor telepon nasabah yang seharusnya tersamar.
// `docs/Steering/11-SECURITY.md` §4.2 menetapkan masking sistem lama dipertahankan, dan
// `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena tidak ada
// pemisahan tugas formal. Itulah sebabnya setiap perubahan di sini dicatat pelakunya.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega dan dari basis data yang berjalan, bukan
// dikarang:
//
//	Harness/MasterProteksiVisibilityData-Harness.xml  layar "Master Masking Data"
//	Section/MasterProteksi_Sec-Section.xml             kerangka layar, grid, tombolnya
//	Section/ActionMaskingData_Sec-Section.xml          aksi EDIT dan DELETE per baris
//	RDB List/SearchMasking_SQL-SQL.xml                 grid; dua tipe pencarian
//	RDB List/GetTipeProteksi-SQL.xml                   baris aktif saja
//	RDB List/DeleteMstProteksi_SQL-SQL.xml             "hapus" = UPDATE STS_AKTF
//	Activity/SearchDataMasking-Act.xml                 penyusun penyaring pencarian
//	Activity/DeleteMasking-Act.xml                     nilai "TIDAK AKTIF"
//	Activity/InsermaskingDataKlaimPnc_-Act.xml         pemetaan isian → parameter
//	Database/UPDATE_LOG_PROTEKSI.prc                   aturan insert/update yang sebenarnya
//
// Beberapa hal TIDAK ada di export dan dibaca langsung dari POOLDATA portal ASM pada
// 2026-09-20 (baca-saja, 25 baris): nilai `'Ya'`/`'Tidak'`, sebaran kuota, dan bentuk
// `SUBMODUL`. Rule `MODULKLAIMMASKING` yang memasok daftar pilihannya hilang dari export
// (`R-16`), sehingga daftar itu tidak dapat direproduksi — lihat CheckMasking.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastermasking

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Batas panjang isian teks.
//
// Seluruh kolom teks tabel ini bertipe VARCHAR2(1000) — dibaca dari ALL_TAB_COLUMNS pada
// 2026-09-20. Angka itu tidak membatasi apa pun yang berguna: login sepanjang 1000
// karakter akan merusak setiap grid yang menampilkannya. Batas di bawah karena itu
// dipilih terhadap isi yang benar-benar ada, dengan ruang tumbuh yang lapang.
//
// Angka yang sama diulang di frontend supaya pengguna tahu sebelum mengirim; server tetap
// yang berwenang. Bila berubah, KEDUA tempat wajib ikut berubah.
const (
	// MaxLoginLength membatasi nama pengguna. Terpanjang yang ada hari ini jauh di bawah
	// ini; 100 dipilih sama dengan master lain yang sudah dibangun supaya batas yang
	// dilihat pengguna seragam antarlayar.
	MaxLoginLength = 100

	// MaxModuleLength membatasi nama modul. Satu-satunya nilai yang ada hari ini
	// ("PNCSearchKlaim") sepanjang 14 karakter.
	MaxModuleLength = 100

	// MaxSubModuleLength membatasi daftar sub modul.
	//
	// Ia jauh lebih longgar daripada yang lain KARENA isinya memang daftar, bukan satu
	// nilai: baris terpanjang di portal ASM memuat tiga sub modul sekaligus dan mencapai
	// 46 karakter. Empat ratus memberi ruang bagi belasan sub modul tanpa mendekati batas
	// kolomnya.
	MaxSubModuleLength = 400

	// MaxBranchIDLength mengikuti kolom POOLDATA.BRANCH.ID yang bertipe VARCHAR2(6) —
	// satu-satunya kolom pada jalur ini yang batas sebenarnya memang sempit.
	MaxBranchIDLength = 6
)

// MaxQuota membatasi nilai kuota cari dan kuota lihat.
//
// Nilai terbesar yang benar-benar ada di portal ASM adalah 100.000, dan ada pula baris
// bernilai 10.000 serta 2.000 — jadi angka besar memang dipakai dengan sengaja, bukan
// salah ketik. Sejuta dipilih sebagai langit-langit yang tidak akan pernah menghalangi
// pemakaian wajar, tetapi tetap menolak angka yang jelas keliru seperti hasil menempel
// nomor polis ke kolom kuota.
//
// Batas bawahnya nol, bukan satu: nol berarti "tidak boleh mencari sama sekali", dan itu
// pernyataan kewenangan yang sah.
const MaxQuota = 1_000_000

// Masking adalah satu baris master masking.
//
// Nama field-nya mengikuti `D-19` dan `D-80` — bukan nama kolomnya, yang di sistem lama
// menyesatkan. Pemetaannya ada di repo/sqlstore, satu tempat saja.
type Masking struct {
	// ID adalah ID_MST. Dibuat penyimpanan saat baris ditambahkan dan tidak pernah
	// berubah sesudahnya. Bertipe NUMBER di basis data, dibawa sebagai teks di sini
	// supaya ia diperlakukan sebagai penanda, bukan sebagai angka yang dapat dihitung.
	ID string

	// BranchID adalah CABANG — kunci ke POOLDATA.BRANCH.ID.
	//
	// Seluruh 25 baris portal ASM menunjuk cabang yang benar-benar ada; tidak satu pun
	// menggantung. Itulah sebabnya ia diperlakukan sebagai kunci asing sungguhan dan
	// dipilih dari daftar, bukan diketik bebas.
	BranchID string

	// BranchName adalah BRANCHNAME hasil penggabungan ke POOLDATA.BRANCH.
	//
	// BACA-SAJA: ia tidak pernah disimpan ke tabel masking. Sistem lama pun mengambilnya
	// lewat subkueri setiap kali grid dimuat. Ia ada di sini supaya layar tidak perlu
	// memanggil dua kali untuk menampilkan satu baris.
	BranchName string

	// Login adalah LOGIN — pengguna yang diberi kewenangan.
	Login string

	// Module adalah MODUL, layar tempat kewenangan ini berlaku.
	Module string

	// SubModule adalah SUBMODUL: DAFTAR bagian layar, dipisah koma.
	//
	// Bentuknya dibawa APA ADANYA sebagai teks, termasuk koma di ujungnya — keputusan
	// Work Owner 2026-09-20. Sistem lama menyimpannya begitu ("Registrasi,Dokumen,") dan
	// Pega masih membaca tabel yang sama selama masa paralel (`D-21`), sehingga mengubah
	// bentuknya akan memutus pembacaan di sana.
	SubModule string

	// SearchQuota adalah LOGSEARCH, berlabel "MAX CARI DATA" di layar lama.
	//
	// Ia KUOTA, bukan penanda boleh-tidak. Sebarannya di portal ASM 1 sampai 100.000 —
	// angka yang tidak masuk akal bagi sebuah flag, dan itulah bukti yang menutup
	// pertanyaannya.
	SearchQuota int

	// ViewQuota adalah LOGSEEN, berlabel "MAX LIHAT DATA".
	ViewQuota int

	// Tiga kewenangan melihat data pribadi tidak tersamar.
	//
	// Di basis data nilainya teks `'Ya'` / `'Tidak'` — BUKAN `'1'`/`'0'` seperti yang
	// sempat diduga dari kode yang dikomentari di UPDATE_LOG_PROTEKSI.prc. Dibaca langsung
	// dari data pada 2026-09-20: hanya kedua nilai itu yang muncul, pada seluruh 25 baris.
	// Penerjemahannya ada di repo/sqlstore.
	ViewIDCard bool // STS_KTP    — nomor KTP
	ViewEmail  bool // STS_EMAIL  — alamat surel
	ViewPhone  bool // STS_NOTELP — nomor telepon

	// Active adalah STS_AKTF, bernilai `'AKTIF'` atau `'TIDAK AKTIF'`.
	//
	// Inilah yang menjadi penghapusan: tombol DELETE di layar lama TIDAK membuang baris,
	// ia hanya mengubah kolom ini (`RDB List/DeleteMstProteksi_SQL-SQL.xml`). Sejalan
	// penuh dengan `D-66` yang melarang penghapusan fisik data bernilai bisnis.
	Active bool

	// InputBy adalah USERINPUT — siapa yang terakhir menyimpan baris ini.
	//
	// Diisi dari SESI, tidak pernah dari badan permintaan. Karena `D-59` menjadikan jejak
	// audit satu-satunya kontrol pengimbang, membiarkan klien menyebut pelakunya sendiri
	// akan membuat jejak itu tidak membuktikan apa pun.
	InputBy string

	// InputAt adalah TANGGALINPUT, waktu penyimpanan terakhir. Diisi penyimpanan.
	InputAt time.Time
}

// Branch adalah satu pilihan cabang untuk isian CABANG.
//
// Ia sengaja BUKAN master milik modul ini: POOLDATA.BRANCH dimiliki sistem lain dan hanya
// DIBACA di sini. Isinya 803 baris pada portal ASM — terlalu banyak untuk daftar biasa,
// sehingga layar menyaringnya dengan kata kunci, sama seperti autocomplete di Pega.
type Branch struct {
	ID   string
	Name string
}

// Dua nilai sah kolom STS_AKTF.
//
// Keduanya diekspor karena ia bukan sekadar bentuk penyimpanan: layar lama memakainya
// langsung sebagai PILIHAN PENCARIAN (`SearchData.Type == "4"` menyusun
// `STS_AKTF = '<nilai>'`), sehingga nilainya ikut menjadi bagian kontrak.
//
// Nilainya dibaca dari basis data pada 2026-09-20 — hanya kedua teks inilah yang muncul
// pada seluruh 25 baris portal ASM.
const (
	StatusActive   = "AKTIF"
	StatusInactive = "TIDAK AKTIF"
)

// SearchBy menyatakan cara pencarian menyaring daftar.
//
// Keempatnya persis yang ada di layar lama. `Activity/SearchDataMasking-Act.xml` menyusun
// penyaring menurut `SearchData.Type`, dan nilainya empat — bukan dua seperti yang sempat
// saya simpulkan pada pembacaan pertama:
//
//	Type "1"  tidak menyaring apa pun
//	Type "2"  CABANG IN (SELECT ID FROM POOLDATA.BRANCH WHERE BRANCHNAME LIKE '%…%')
//	Type "3"  LOGIN LIKE '%…%'
//	Type "4"  STS_AKTF = '<nilai>'
type SearchBy string

const (
	// SearchByAll tidak menyaring apa pun — Type "1" di layar lama. Ia keadaan awal layar,
	// saat pengguna belum memilih apa pun.
	SearchByAll SearchBy = ""

	// SearchByBranch menelusuri NAMA cabang, bukan kodenya — Type "2".
	SearchByBranch SearchBy = "cabang"

	// SearchByLogin menelusuri nama pengguna — Type "3".
	SearchByLogin SearchBy = "login"

	// SearchByStatus menyaring menurut status aktif — Type "4".
	//
	// Berbeda dari kedua di atas, ia cocok PERSIS, bukan sebagian: layar lama menyusun
	// `STS_AKTF = '…'`, bukan `LIKE`. Nilainya salah satu dari StatusActive atau
	// StatusInactive.
	SearchByStatus SearchBy = "status"
)

// Known menyatakan apakah tipe pencarian ini dikenal.
func (s SearchBy) Known() bool {
	switch s {
	case SearchByAll, SearchByBranch, SearchByLogin, SearchByStatus:
		return true
	}
	return false
}

// Filter adalah penyaring daftar.
//
// Bentuknya struct, bukan deretan parameter, supaya menambah penyaring kelak tidak
// mengubah tanda tangan setiap lapisan yang dilewatinya.
//
// Ia sengaja memakai SATU nilai pencarian untuk keempat tipe, sama seperti layar lama yang
// menyusun satu penyaring `InputSearch.CARI1` apa pun tipenya. Menambah field terpisah per
// tipe akan membuat bentuknya menyimpang dari yang digantikannya tanpa manfaat apa pun.
type Filter struct {
	// By memilih cara menyaring. Kosong berarti seluruh baris.
	By SearchBy

	// Keyword adalah nilai pencarian.
	//
	// Untuk SearchByBranch dan SearchByLogin ia dicocokkan SEBAGIAN, tidak peduli
	// besar-kecil huruf — meniru `LIKE '%…%'` layar lama. Untuk SearchByStatus ia
	// dicocokkan PERSIS, dan hanya StatusActive atau StatusInactive yang sah.
	Keyword string
}

// StatusKeyword mengembalikan nilai status yang diminta filter, dan apakah ia sah.
//
// Hanya dipakai saat By bernilai SearchByStatus. Nilai di luar kedua status yang dikenal
// DITOLAK, tidak diam-diam diperlakukan sebagai "tanpa penyaring" — layar lama pun menolak
// dengan pesan "Pilih Status Aktif" bila status belum dipilih.
func (f Filter) StatusKeyword() (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(f.Keyword)) {
	case StatusActive:
		return StatusActive, true
	case StatusInactive:
		return StatusInactive, true
	}
	return "", false
}

// Clean mengembalikan salinan dengan spasi tepi dibuang.
//
// Bukan kerapian kosmetik: `Activity/SearchDataMasking-Act.xml` merangkai kata kunci
// langsung ke teks SQL, sehingga spasi yang tidak disengaja ikut menjadi bagian pencarian
// dan membuat pengguna melihat "tidak ada hasil" untuk kata yang sebenarnya ada.
func (f Filter) Clean() Filter {
	return Filter{
		By:      SearchBy(strings.TrimSpace(string(f.By))),
		Keyword: strings.TrimSpace(f.Keyword),
	}
}

// Clean mengembalikan salinan dengan seluruh isian teks dirapikan.
func (m Masking) Clean() Masking {
	m.ID = strings.TrimSpace(m.ID)
	m.BranchID = strings.TrimSpace(m.BranchID)
	m.BranchName = strings.TrimSpace(m.BranchName)
	m.Login = strings.TrimSpace(m.Login)
	m.Module = strings.TrimSpace(m.Module)
	m.SubModule = strings.TrimSpace(m.SubModule)
	m.InputBy = strings.TrimSpace(m.InputBy)
	return m
}

// PairKey adalah bentuk pasangan cabang+login yang dipakai menguji keunikan.
//
// Perbandingan mengabaikan besar-kecil huruf. Alasannya bukan selera:
// `docs/Steering/11-SECURITY.md` §3.1 mencatat nama access group Pega muncul dalam dua
// kapitalisasi berbeda karena perbandingan rule lama tidak konsisten soal itu. Dua baris
// untuk orang yang sama, satu ditulis `BUDI` dan satu `budi`, adalah dua kewenangan
// terpisah yang keduanya berlaku — dan yang satu dapat luput saat dicabut.
func PairKey(branchID, login string) string {
	return strings.ToUpper(strings.TrimSpace(branchID)) + "|" + strings.ToUpper(strings.TrimSpace(login))
}

// Repo adalah seam ke penyimpanan master masking SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
//
// Antarmukanya berbicara dalam istilah domain, bukan istilah SQL.
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring, terbaru lebih dulu —
	// mengikuti `ORDER BY TANGGALINPUT DESC` pada SearchMasking_SQL.
	List(ctx context.Context, filter Filter) ([]Masking, error)

	// Get mengembalikan satu baris menurut ID_MST. ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Masking, error)

	// FindByPair mencari baris menurut pasangan cabang+login.
	//
	// Ia ada karena pasangan itulah KUNCI ALAMI tabel ini: procedure lama menolak insert
	// bila pasangannya sudah ada, dan update pun mencocokkan pasangan itu, bukan ID_MST
	// (`Database/UPDATE_LOG_PROTEKSI.prc:29-31`, `:74-76`).
	FindByPair(ctx context.Context, branchID, login string) (Masking, error)

	// Insert menyimpan baris baru dan mengembalikannya LENGKAP DENGAN ID yang dibuat
	// penyimpanan. ID tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, m Masking) (Masking, error)

	// Update mengubah baris yang sudah ada. ID tidak ikut berubah.
	Update(ctx context.Context, m Masking) (Masking, error)

	// SetActive mengaktifkan atau menonaktifkan satu baris — inilah "hapus" di layar lama.
	SetActive(ctx context.Context, id string, active bool, by string, at time.Time) (Masking, error)

	// ListBranches mengembalikan pilihan cabang yang cocok dengan kata kunci.
	ListBranches(ctx context.Context, keyword string, limit int) ([]Branch, error)

	// BranchExists menyatakan apakah kode cabang benar-benar ada di POOLDATA.BRANCH.
	BranchExists(ctx context.Context, branchID string) (bool, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti memberi seseorang
// kewenangan melihat data pribadi di badan hukum yang bukan haknya, tanpa satu pun pesan
// galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// CheckMasking mengumpulkan SELURUH pelanggaran aturan isian, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama menampilkan
// seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan membuat
// pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// Keunikan dan keberadaan cabang TIDAK diperiksa di sini: keduanya menuntut membaca
// penyimpanan, sedangkan fungsi ini murni dan dapat diuji tanpa apa pun. Pemeriksaannya
// ada di usecase.
//
// # Kenapa MODUL dan SUB MODUL tidak dibatasi ke daftar pilihan
//
// Keduanya diketik bebas — keputusan Work Owner 2026-09-20, "seperti aplikasi Pega saja".
// Daftar pilihannya di Pega dipasok rule `MODULKLAIMMASKING`, dan rule itu **hilang dari
// export** (`R-16`), sehingga daftarnya tidak dapat direproduksi tanpa mengarangnya.
// Membatasi isian ke satu-satunya nilai yang kebetulan ada di data hari ini
// ("PNCSearchKlaim") akan menolak nilai sah yang belum pernah dipakai — menolak data yang
// benar jauh lebih merugikan daripada menerima salah ketik yang dapat disunting kembali.
func CheckMasking(m Masking) []Violation {
	m = m.Clean()
	var violation []Violation

	add := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if m.BranchID == "" {
		add(FieldBranchID, "Cabang wajib dipilih.")
	} else if utf8.RuneCountInString(m.BranchID) > MaxBranchIDLength {
		// Batas ini datang dari kolom POOLDATA.BRANCH.ID yang VARCHAR2(6). Kode yang lebih
		// panjang mustahil cocok ke cabang mana pun, jadi menolaknya di sini memberi pesan
		// yang jelas alih-alih "cabang tidak ditemukan" yang membingungkan.
		add(FieldBranchID, "Kode cabang paling panjang "+strconv.Itoa(MaxBranchIDLength)+" karakter.")
	}

	if m.Login == "" {
		add(FieldLogin, "Nama pengguna wajib diisi.")
	} else if utf8.RuneCountInString(m.Login) > MaxLoginLength {
		add(FieldLogin, "Nama pengguna paling panjang "+strconv.Itoa(MaxLoginLength)+" karakter.")
	}

	if m.Module == "" {
		// Sistem lama tidak memaksa, tetapi SELURUH 25 baris terisi — dan baris tanpa modul
		// tidak menyatakan kewenangan apa pun yang dapat ditegakkan. Ia akan menjadi baris
		// yang tampak memberi hak padahal tidak berlaku di mana-mana.
		add(FieldModule, "Modul wajib diisi.")
	} else if utf8.RuneCountInString(m.Module) > MaxModuleLength {
		add(FieldModule, "Modul paling panjang "+strconv.Itoa(MaxModuleLength)+" karakter.")
	}

	// Sub modul TIDAK diwajibkan. Empat baris portal ASM hanya memuat satu sub modul dan
	// bentuknya memang daftar yang boleh pendek; memaksanya terisi akan menolak keadaan
	// yang sah dan sudah berjalan.
	if utf8.RuneCountInString(m.SubModule) > MaxSubModuleLength {
		add(FieldSubModule, "Sub modul paling panjang "+strconv.Itoa(MaxSubModuleLength)+" karakter.")
	}

	checkQuota(&violation, FieldSearchQuota, "Maks. cari data", m.SearchQuota)
	checkQuota(&violation, FieldViewQuota, "Maks. lihat data", m.ViewQuota)

	return violation
}

// checkQuota memeriksa satu kuota. Keduanya punya aturan yang sama persis, sehingga
// menuliskannya dua kali hanya membuka peluang keduanya menyimpang kelak.
func checkQuota(violation *[]Violation, field, label string, value int) {
	switch {
	case value < 0:
		*violation = append(*violation, Violation{
			Field:   field,
			Message: label + " tidak boleh kurang dari 0.",
		})
	case value > MaxQuota:
		*violation = append(*violation, Violation{
			Field:   field,
			Message: label + " paling besar " + strconv.Itoa(MaxQuota) + ".",
		})
	}
}

// ErrNoSequence dikembalikan bila ID baru tidak dapat dibentuk karena tabelnya kosong dan
// nilai awalnya tidak dapat disimpulkan. Ia dideklarasikan di domain supaya transport dapat
// membedakannya dari kegagalan basis data biasa.
var ErrNoSequence = errors.New("mastermasking: ID_MST baru tidak dapat dibentuk")
