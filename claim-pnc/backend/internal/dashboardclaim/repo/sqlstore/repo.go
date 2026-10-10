package sqlstore

import (
	"database/sql"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// Repo membaca ringkasan dan telusur Dashboard Claim dari satu basis data entitas.
//
// Satu Repo melayani SATU portal. Pemilihannya dikerjakan RepoSelector di cmd, bukan di
// sini: modul ini tidak pernah tahu ada portal lain, sehingga tidak ada jalur yang dapat
// keliru membaca entitas yang salah (`R-20`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk Repo atas satu koneksi.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// filterBindCount adalah jumlah argumen penyaring yang dipakai SELURUH kueri di modul ini.
//
// Keempat kueri sengaja disusun memakai penyaring yang sama persis dan berjumlah sama,
// sehingga satu fungsi filterArgs melayani semuanya. Nilainya dijaga uji: kueri yang
// menambah penanda tanpa menambah argumennya akan gagal saat dijalankan, dan gagalnya
// terjadi di produksi — bukan saat ditulis.
const filterBindCount = 8

// outstandingBindCount adalah jumlah argumen penyaring kueri OUTSTANDING.
//
// Ia LEBIH BANYAK daripada kueri survei sejak panel penyaring dibangun: tile Outstanding
// punya lima penyaring tambahan — Nopolis, No Klaim, PIC, Status Transfer, Status
// Pembayaran — yang datang dari `Section/FilterDashboardClaim_sec-Section.xml`, dan tile
// survei tidak punya panel itu.
//
// Kedua konstanta sengaja dipisah, bukan disatukan ke angka terbesar: menyamakannya akan
// menuntut kueri survei menyiapkan argumen yang tidak pernah dipakainya.
const outstandingBindCount = 20

// filterArgs menyusun kedelapan argumen penyaring.
//
// Setiap penanda `:n` memperoleh argumennya sendiri, termasuk ketika nilainya sama — itulah
// konvensi yang dipakai seluruh sqlstore di sini, dan yang membuat jumlah penanda dan jumlah
// argumen dapat dibandingkan langsung oleh uji.
//
// Urutannya WAJIB sama dengan urutan kemunculan `:n` di dalam teks SQL.
func filterArgs(f dashboardclaim.Filter) []any {
	business := string(f.Business)

	search := nilIfEmpty(f.Search)
	searchPattern := nilIfEmpty(likePattern(f.Search))

	return []any{
		business, // :1 ALL
		business, // :2 NONMBU
		business, // :3 BONDING
		business, // :4 PA
		business, // :5 TRAVEL

		search,        // :6 kotak cari aktif?
		searchPattern, // :7 POLICYNO
		searchPattern, // :8 PYID
	}
}

// outstandingFilterArgs menyusun argumen untuk kedua kueri tile OUTSTANDING.
//
// Urutannya BERBEDA dari filterArgs: pada kueri itu penyaring pencarian ditulis lebih dulu,
// karena penyaring lini bisnisnya berada di klausa WHERE terluar dan bukan di dalam EXISTS.
// Dipisahkan menjadi fungsinya sendiri supaya perbedaan urutan itu terlihat, bukan
// tersembunyi sebagai parameter pada satu fungsi yang melayani dua bentuk.
func outstandingFilterArgs(f dashboardclaim.Filter) []any {
	business := string(f.Business)

	search := nilIfEmpty(f.Search)
	searchPattern := nilIfEmpty(likePattern(f.Search))

	transfer := nilIfEmpty(string(f.Cashier))
	payment := nilIfEmpty(string(f.Payment))

	return []any{
		search,        // :1 kotak cari aktif?
		searchPattern, // :2 POLICYNO
		searchPattern, // :3 PYID

		business, // :4 ALL
		business, // :5 NONMBU
		business, // :6 BONDING
		business, // :7 PA
		business, // :8 TRAVEL

		// Ketiga penyaring panel. Masing-masing dua penanda — satu menjawab "apakah isian
		// ini diisi", satu membawa polanya — mengikuti pola penyaring pencarian di atas.
		nilIfEmpty(f.PolicyNumber),              // :9  Nopolis diisi?
		nilIfEmpty(likePattern(f.PolicyNumber)), // :10 polanya
		nilIfEmpty(f.ClaimNumber),               // :11 No Klaim diisi?
		nilIfEmpty(likePattern(f.ClaimNumber)),  // :12 polanya
		nilIfEmpty(f.TechnicalPIC),              // :13 PIC diisi?
		nilIfEmpty(likePattern(f.TechnicalPIC)), // :14 polanya

		// Kedua dropdown. Tiga penanda masing-masing: satu menjawab "apakah dropdown ini
		// dipilih", dua sisanya memilih salah satu dari dua cabang klausanya.
		//
		// Nilainya dikirim APA ADANYA, bukan sebagai pola LIKE — keduanya dibandingkan
		// dengan konstanta, bukan dicocokkan.
		transfer, // :15 Status Transfer dipilih?
		transfer, // :16 cabang SUDAH
		transfer, // :17 cabang BELUM
		payment,  // :18 Status Pembayaran dipilih?
		payment,  // :19 cabang LUNAS
		payment,  // :20 cabang BELUM
	}
}

// withPaging menambahkan OFFSET dan LIMIT di belakang argumen penyaring.
//
// Ia menyalin irisannya lebih dulu, tidak menambah ke irisan asal: dua kueri memakai
// argumen penyaring yang sama dalam satu permintaan — yang menghitung dan yang mendaftar —
// dan menambahkan ke irisan yang sama akan membuat kueri hitung ikut membawa paginasi pada
// pemanggilan berikutnya.
func withPaging(filters []any, f dashboardclaim.Filter) []any {
	args := make([]any, 0, len(filters)+2)
	args = append(args, filters...)
	return append(args, f.Offset, f.Limit)
}

// likePattern membentuk pola LIKE.
//
// Diseragamkan menjadi huruf besar karena sisi SQL memakai UPPER(...). Perbandingan yang
// hanya satu sisinya diseragamkan tidak pernah cocok, dan gagalnya DIAM — pengguna mengetik
// dengan huruf kecil lalu diberi tahu bahwa klaimnya tidak ada.
//
// Karakter khusus LIKE di-escape supaya pencarian "100%" tidak berubah menjadi pola yang
// mencocokkan apa saja. ESCAPE-nya dinyatakan di sisi SQL.
func likePattern(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// nilIfEmpty mengubah string kosong menjadi NULL.
//
// Dipakai supaya pola "NULL berarti tidak menyaring" pada sisi SQL bekerja. Tanpa itu,
// kotak cari yang kosong akan diperlakukan sebagai pencarian atas teks kosong.
func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// text membaca kolom teks yang boleh kosong.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: kolomnya nullable di skema warisan, dan
// membacanya langsung ke string akan gagal dengan galat konversi pada baris pertama yang
// kosong — kegagalan yang muncul di produksi, bukan saat ditulis.
func text(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
