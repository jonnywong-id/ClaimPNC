package reportklaim

import (
	"strings"
	"time"
)

// FilterSet menyatakan penyaring bersama mana yang dipakai sebuah laporan.
//
// Ia sengaja berupa himpunan bit, bukan daftar string: pertanyaannya selalu "apakah
// laporan ini memakai X", dan jawabannya harus dapat diuji tanpa mengalokasi apa pun.
type FilterSet uint8

const (
	// FilterDateRange: isian "Dari" dan "Sampai".
	//
	// Keduanya SATU penyaring, bukan dua. Tidak ada satu pun kueri di export yang
	// menerima salah satunya saja — keduanya selalu muncul berpasangan sebagai
	// `trunc(kolom) >= to_date({awal}) and trunc(kolom) <= to_date({akhir})`.
	FilterDateRange FilterSet = 1 << iota

	// FilterBusinessLine: dropdown yang di layar berlabel "Treaty".
	FilterBusinessLine

	// FilterComplianceStatus: radio "Status Compliance".
	FilterComplianceStatus

	// FilterBusinessCode: autocomplete "Bisnis" pada panel Klaim Per Bisnis.
	//
	// Ia BERBEDA dari FilterBusinessLine meski namanya mirip. Yang ini memilih SATU kode
	// bisnis dari master (`TempLaporan.CountryID`, sumbernya `BrowseBusiness_RD`);
	// yang itu memilih KELOMPOK lini bisnis. Menyatukan keduanya akan membuat panel
	// Klaim Per Bisnis menyaring kelompok, bukan satu bisnis — dan seluruh isinya salah
	// tanpa satu pun pesan galat.
	FilterBusinessCode

	// FilterDetail: kotak centang yang di layar berlabel "Checkbox".
	//
	// Isinya `TempLaporan.Remark`, dan yang ia tentukan BUKAN penyaring baris melainkan
	// **susunan kolom**: tidak dicentang menghasilkan berkas ringkas, dicentang
	// menghasilkan berkas rinci. Dipakai panel Akseptasi, OS Komite, dan OS Belum Komite.
	FilterDetail
)

// Has menyatakan apakah penyaring tertentu termasuk dalam himpunan ini.
func (s FilterSet) Has(f FilterSet) bool { return s&f != 0 }

// BusinessLine adalah pilihan dropdown yang di layar berlabel **"Treaty"**.
//
// # Labelnya salah, dan ia tetap ditiru
//
// Isinya bukan treaty melainkan **lini bisnis**. Label "Treaty" tetap dipakai di layar
// atas keputusan Work Owner 2026-09-24, mengikuti `D-13` — pengguna sudah mengenalnya
// dengan nama itu selama bertahun-tahun, dan memperbaikinya di layar berarti melatih
// ulang tanpa ada yang meminta. Di dalam kode ia bernama menurut isinya.
//
// # Nilainya dibaca dari rule, LABELNYA tidak ada di export
//
// Kelima nilai di bawah terverifikasi dari perbandingan `TempLaporan.StatusReceiver` di
// seluruh export: `""` 4 kali, `"002"` 62 kali, `"005"` 59 kali, `"346"` 33 kali,
// `"003"` 14 kali.
//
// **Label pilihannya tidak dapat dibaca dari export.** Dropdown-nya ber-`pyListSource`
// `associated`, artinya daftarnya hidup di rule Property — dan export ini tidak memuat
// satu pun rule Property (tidak ada direktori Property sama sekali). Nama di bawah karena
// itu diturunkan dari **potongan SQL yang dipasang tiap pilihan**, bukan dari rule label
// yang hilang, dan memakai istilah `CONTEXT.md`. Lihat businessLineName.
type BusinessLine string

const (
	// BusinessLineAll: tanpa penyaring lini bisnis. Nilai Pega: "".
	BusinessLineAll BusinessLine = ""

	// BusinessLinePA: Personal Accident. Nilai Pega: "002", Group Panel 002.
	BusinessLinePA BusinessLine = "002"

	// BusinessLineTravel: Travel. Nilai Pega: "005", Group Panel 005.
	BusinessLineTravel BusinessLine = "005"

	// BusinessLineNonMBU: Non-MBU umum. Nilai Pega: "346".
	//
	// Angkanya tidak berhubungan dengan Group Panel mana pun — Group Panel yang
	// disaringnya adalah 003, 004, 006, dan 009 dikurangi sejumlah businesscode. Ia
	// sekadar nilai pilihan, dan itulah sebabnya ia tidak diterjemahkan menjadi kode
	// panel di sini.
	BusinessLineNonMBU BusinessLine = "346"

	// BusinessLineBonding: Bonding. Nilai Pega: "003".
	//
	// JANGAN tertukar dengan Group Panel "003" yang berarti Aneka. Kebetulan angkanya
	// sama, artinya tidak: pilihan ini menyaring Group Panel 003 DITAMBAH daftar
	// businesscode tertentu, dan daftar itu BERBEDA-BEDA antar laporan.
	BusinessLineBonding BusinessLine = "003"
)

// businessLineName adalah label pilihan yang dibaca pengguna.
//
// Diturunkan dari potongan SQL tiap pilihan, bukan dari rule label — lihat BusinessLine.
func businessLineName(l BusinessLine) string {
	switch l {
	case BusinessLineAll:
		return "----- Pilih -----"
	case BusinessLinePA:
		return "Personal Accident"
	case BusinessLineTravel:
		return "Travel"
	case BusinessLineNonMBU:
		return "Non-MBU"
	case BusinessLineBonding:
		return "Bonding"
	}
	return string(l)
}

// BusinessLineOption adalah satu baris pilihan pada dropdown "Treaty".
type BusinessLineOption struct {
	Value BusinessLine
	Name  string
}

// BusinessLineOptions mengembalikan kelima pilihan sesuai urutan yang masuk akal dibaca.
//
// Pilihan kosong ada di depan dan bertuliskan "----- Pilih -----" apa adanya — itu
// `pyNoSelectionText` dropdown-nya di harness, disalin termasuk garisnya.
func BusinessLineOptions() []BusinessLineOption {
	lines := []BusinessLine{
		BusinessLineAll,
		BusinessLinePA,
		BusinessLineTravel,
		BusinessLineNonMBU,
		BusinessLineBonding,
	}
	out := make([]BusinessLineOption, 0, len(lines))
	for _, l := range lines {
		out = append(out, BusinessLineOption{Value: l, Name: businessLineName(l)})
	}
	return out
}

// ParseBusinessLine menerima nilai dari luar dan menolak yang tidak dikenal.
//
// Menolak nilai asing adalah kendali yang di layar diberikan oleh dropdown. Diterima apa
// adanya, kendali itu hilang begitu permintaan datang dari luar layar — dan API memang
// dapat ditembak langsung.
func ParseBusinessLine(raw string) (BusinessLine, bool) {
	switch BusinessLine(strings.TrimSpace(raw)) {
	case BusinessLineAll:
		return BusinessLineAll, true
	case BusinessLinePA:
		return BusinessLinePA, true
	case BusinessLineTravel:
		return BusinessLineTravel, true
	case BusinessLineNonMBU:
		return BusinessLineNonMBU, true
	case BusinessLineBonding:
		return BusinessLineBonding, true
	}
	return BusinessLineAll, false
}

// IsPersonal menyatakan apakah pilihan ini termasuk jalur PA atau Travel.
//
// Keduanya dipasangkan berkali-kali di export sebagai satu syarat —
// `StatusReceiver=="002"||StatusReceiver=="005"` — dan itulah yang memilih susunan kolom
// pada laporan TAT, Data AI Klaim, dan Data Komite. Menuliskan syaratnya berulang kali
// berarti ketiganya dapat berbeda saat salah satu diubah.
func (l BusinessLine) IsPersonal() bool {
	return l == BusinessLinePA || l == BusinessLineTravel
}

// Filter adalah seluruh penyaring yang dikirim layar saat satu laporan dijalankan.
type Filter struct {
	// From dan To adalah isian "Dari" dan "Sampai", sebagai TANGGAL WIB.
	//
	// Bukan timestamp. Seluruh kueri membandingkannya dengan `trunc(kolom)`, yaitu
	// perbandingan hari kalender — dan aturan berbasis hari kalender dihitung terhadap
	// tanggal WIB, bukan terhadap UTC (`08-TECHNICAL-STRATEGY.md` §4.4).
	From time.Time
	To   time.Time

	// BusinessLine adalah pilihan dropdown "Treaty".
	BusinessLine BusinessLine

	// ComplianceStatus adalah pilihan radio "Status Compliance".
	//
	// # Kenapa ia string bebas, dan itu bukan kelalaian
	//
	// Daftar pilihannya TIDAK ADA di export: radionya ber-`pyListSource` `associated`,
	// dan rule Property tempat daftarnya hidup tidak ikut dikirim — tidak ada satu pun
	// direktori Property di export ini (`R-16`). Pencarian ke seluruh Activity, RDB List,
	// dan When rule juga tidak menemukan satu pun perbandingan terhadap nilainya.
	//
	// Mengarang daftarnya berarti menampilkan pilihan yang mungkin tidak ada di data, dan
	// laporan yang selalu kosong tanpa satu pun tanda. Ia karena itu diteruskan apa
	// adanya, dan pertanyaannya ditujukan ke Tim Pega.
	ComplianceStatus string

	// BusinessCode adalah kode bisnis terpilih pada panel Klaim Per Bisnis.
	BusinessCode string

	// Detail adalah kotak centang "Checkbox" — ia memilih susunan kolom, bukan baris.
	Detail bool

	// FixedParam adalah parameter tetap milik TOMBOL yang ditekan, bukan isian pengguna.
	//
	// Ia disalin lapisan transport dari Action.FixedParam, bukan dibaca dari permintaan.
	// Perbedaan itu penting: `statusapprove` menentukan apakah yang diunduh adalah komite
	// yang MENYETUJUI atau yang MENOLAK, dan membiarkannya datang dari luar berarti
	// membiarkan pemanggil memilih sesuatu yang di layar ditentukan oleh tombol.
	FixedParam map[string]string

	// Entity adalah entitas yang sedang aktif, yaitu portal pemanggil.
	//
	// # Ia MENGGANTIKAN perbandingan nama server
	//
	// `Activity/PNCTATReport1_Act-Act.xml` bercabang pada
	// `pxRequestor.pxReqServer=="pega.simasinsurtech.com"` untuk menambah pengecualian
	// businesscode. Membandingkan nama server dilarang (`12-CROSSCUTTING.md` §3.4).
	//
	// Nilainya diisi lapisan transport dari portal aktif, bukan dibaca dari lingkungan:
	// entitas dinyatakan oleh pemanggil, tidak disimpulkan dari mesin yang kebetulan
	// melayaninya (`D-75`, `ADR-0030`).
	Entity string
}

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, supaya layar dapat
	// menyorot isian yang dimaksud.
	Field   string
	Message string
}

// Validate memeriksa penyaring terhadap laporan yang diminta.
//
// Ia mengembalikan SELURUH pelanggaran sekaligus, bukan yang pertama saja — kesetaraan
// perilaku, bukan selera (`P-5`): layar lama menampilkan seluruh pesan bersamaan.
func (r Report) Validate(f Filter) []Violation {
	var v []Violation

	if r.Uses.Has(FilterDateRange) {
		switch {
		case f.From.IsZero():
			v = append(v, Violation{Field: "dari", Message: "Tanggal Dari wajib diisi."})
		case f.To.IsZero():
			v = append(v, Violation{Field: "sampai", Message: "Tanggal Sampai wajib diisi."})
		case f.To.Before(f.From):
			// Dibandingkan sebagai tanggal, bukan sebagai saat: keduanya sudah berupa
			// tanggal WIB tanpa jam (lihat Filter.From).
			v = append(v, Violation{
				Field:   "sampai",
				Message: "Tanggal Sampai tidak boleh mendahului Tanggal Dari.",
			})
		}
	}

	if r.Uses.Has(FilterBusinessCode) && strings.TrimSpace(f.BusinessCode) == "" {
		v = append(v, Violation{Field: "bisnis", Message: "Bisnis wajib dipilih."})
	}

	return v
}
