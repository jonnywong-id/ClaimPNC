package reportkpi

import "strings"

// Berkas ini memuat tab **KPI Admin** — kinerja tim ADMIN REGISTRASI klaim.
//
// Ia berbeda dari tab KPI Adjuster pada dua hal yang menentukan bentuknya:
//
//   - Yang diukur bukan sembilan komponen yang sama untuk setiap orang, melainkan satu
//     KARTU SKOR per kelompok, dengan metrik yang berbeda antar kelompok.
//   - Seluruh hitungannya dikerjakan BASIS DATA, bukan activity Pega. Karena itu ia dapat
//     dipindahkan setia — yang dipindahkan adalah kuerinya, bukan ~900 KB langkah Java.
//
// # Nilai yang di-hardcode DIBIARKAN, atas ketetapan Work Owner
//
// Enam Operator ID, nama koordinator, NIK, unit kerja, bobot, dan ambang nilai seluruhnya
// ditulis di dalam rule Pega. `D-15` menetapkan tidak satu pun boleh berada di dalam kode,
// tetapi Work Owner menetapkan 2026-09-24: **"seperti aplikasi PEGA saja"**.
//
// Karena itu nilainya ditiru apa adanya dan dikumpulkan DI SATU TEMPAT — di sini dan di
// `reportkpi_admin.sql` — bukan disebar. Ketika masternya kelak dibuat, yang berubah hanya
// kedua berkas itu.

// AdminGroup adalah kelompok yang diukur tab KPI Admin.
//
// Ia isi dropdown "Pilih Data KPI" pada layar lama. Kedua kelompok punya kartu skor dan
// grid rincian yang BERBEDA bentuknya — bukan penyaring atas bentuk yang sama.
type AdminGroup string

// Kedua kelompok yang dikenal.
const (
	// AdminGroupNonMBU mengukur tim admin lini NON-MBU, dibedakan leader dan member.
	AdminGroupNonMBU AdminGroup = "NONMBU"

	// AdminGroupPA mengukur tim admin lini Personal Accident, dibedakan tahap registrasi
	// dan tahap pembayaran.
	AdminGroupPA AdminGroup = "PA"
)

// AdminGroupOption adalah satu pilihan pada dropdown "Pilih Data KPI".
type AdminGroupOption struct {
	Code  AdminGroup
	Label string
	Note  string
}

// adminGroups adalah kedua pilihan beserta keterangannya.
//
// Labelnya mengikuti teks yang ditulis kuerinya sendiri sebagai kolom kategori —
// `'KLAIM NON MBU'` dan `'KLAIM PA'` (`D-13`).
var adminGroups = []AdminGroupOption{
	{
		Code: AdminGroupNonMBU, Label: "KLAIM NON MBU",
		Note: "Dibedakan LEADER dan MEMBER. Yang diukur satu tahap: TAT registrasi.",
	},
	{
		Code: AdminGroupPA, Label: "KLAIM PA",
		Note: "Dibedakan tahap REGISTRASI dan PEMBAYARAN. Tidak ada leader/member di sini.",
	},
}

// AdminGroups mengembalikan salinan kedua pilihan.
func AdminGroups() []AdminGroupOption {
	result := make([]AdminGroupOption, len(adminGroups))
	copy(result, adminGroups)
	return result
}

// FindAdminGroup mencari kelompok menurut kodenya, mengabaikan besar-kecil huruf.
func FindAdminGroup(code string) (AdminGroup, bool) {
	wanted := strings.ToUpper(strings.TrimSpace(code))
	for _, g := range adminGroups {
		if string(g.Code) == wanted {
			return g.Code, true
		}
	}
	return "", false
}

// AdminIdentity adalah kepala kartu skor — siapa yang dinilai.
//
// Keempat isiannya ditulis sebagai LITERAL di dalam rule Pega, bukan dibaca dari tabel
// mana pun. Nilainya dikumpulkan di adminIdentities di bawah.
type AdminIdentity struct {
	// Category <- kolom kategori pada kueri: "KLAIM NON MBU" / "KLAIM PA".
	Category string

	// Coordinator adalah NAMA KOORDINATOR.
	Coordinator string

	// NIK koordinator.
	NIK string

	// WorkUnit adalah UNIT KERJA.
	WorkUnit string
}

// adminIdentities memetakan kelompok ke identitas koordinatornya.
//
// # SATU KEJANGGALAN YANG DIREPLIKASI, DAN PERLU DIKETAHUI SEBELUM DIBANDINGKAN
//
// Pada kelompok NON-MBU, kueri dan activity menyebut nama yang BERBEDA:
//
//	RDB List/GetDataKPIAdmin-SQL.xml     'YUSMIARSIH DYAHPUSPITA S' AS "Remark"
//	Activity/PNCReportKPIAdmin_Act       .Remark := "MORASOTARDODOTARIGAN"
//
// Activity berjalan SESUDAH kueri, sehingga yang benar-benar dilihat pengguna adalah nama
// kedua. Itulah yang dipakai di sini (`P-5` — hasil yang sama dengan Pega).
//
// Nama pertama tidak dibuang; ia dicatat pada CoordinatorInQuery supaya selisihnya terlihat
// saat seseorang membandingkan layar ini dengan teks kuerinya. Mana yang BENAR menurut
// bisnis belum dipastikan — diajukan ke Work Owner.
var adminIdentities = map[AdminGroup]AdminIdentity{
	AdminGroupNonMBU: {
		Category:    "KLAIM NON MBU",
		Coordinator: "MORASOTARDODOTARIGAN",
		NIK:         "96030583",
		WorkUnit:    "ALL (NON HEALTH DAN NON MBU)",
	},
	AdminGroupPA: {
		Category:    "KLAIM PA",
		Coordinator: "YUNIARPAMORSUARI",
		NIK:         "11908020",
		WorkUnit:    "KOORDINASI PA",
	},
}

// CoordinatorInQuery adalah nama koordinator sebagaimana ditulis di dalam TEKS KUERI lama.
//
// Hanya NON-MBU yang punya selisih; pada PA kueri dan activity sepakat. Dipakai keterangan
// layar, bukan sebagai nilai yang ditampilkan.
const CoordinatorInQuery = "YUSMIARSIH DYAHPUSPITA S"

// AdminIdentityFor mengembalikan identitas koordinator sebuah kelompok.
func AdminIdentityFor(group AdminGroup) AdminIdentity {
	return adminIdentities[group]
}

// MetricFormat menyatakan cara sebuah angka digambar.
//
// Ia dibawa sebagai DATA, bukan disimpulkan layar dari nama metriknya: satu kartu skor
// memuat cacah, persentase, bobot, nilai 1–5, dan rasio sekaligus — dan menebaknya dari
// nama akan salah pada metrik berikutnya yang ditambahkan.
type MetricFormat string

// Keempat bentuk angka pada kartu skor.
const (
	// FormatCount adalah cacah klaim — bilangan bulat.
	FormatCount MetricFormat = "cacah"

	// FormatPercent adalah persentase, digambar dengan tanda persen.
	FormatPercent MetricFormat = "persen"

	// FormatScore adalah nilai 1–5 hasil tangga ambang.
	FormatScore MetricFormat = "nilai"

	// FormatDecimal adalah pecahan biasa — bobot dan rasio pencapaian.
	FormatDecimal MetricFormat = "desimal"
)

// Metric adalah satu baris terukur pada kartu skor.
type Metric struct {
	Code  string
	Label string

	// Value kosong bila metriknya tidak dapat dihitung — misalnya ketika pembaginya nol
	// karena tidak ada satu pun klaim pada periode yang dipilih.
	Value Score

	Format MetricFormat
}

// AdminScorecard adalah kartu skor satu kelompok — satu "baris" pada grid "Data KPI".
//
// Ia TIDAK berbentuk tabel berbaris-baris melainkan satu kartu berisi metrik berurutan,
// dan itu mengikuti layar lama: kuerinya mengembalikan TEPAT SATU baris dengan belasan
// kolom, bukan banyak baris.
type AdminScorecard struct {
	Group    AdminGroup
	Identity AdminIdentity

	// EffectiveOn adalah TANGGAL EFEKTIF — periode yang dinilai, sebagai teks.
	//
	// Di Pega ia dirangkai di dalam SELECT dari dua properti tanggal. Di sini ia disusun
	// Go dari rentang permintaan: merangkainya di SQL berarti pemformatan tanggal kembali
	// ke basis data, yang `D-20` justru keluarkan dari sana.
	EffectiveOn string

	// Metrics adalah metrik kelompok ini, DALAM URUTAN kartu skor layar lama.
	Metrics []Metric

	// Achievement adalah kesimpulan: tercapai atau tidak.
	//
	// Kosong pada kelompok PA — kartu skornya memang tidak punya baris kesimpulan, dan
	// mengarangnya akan menampilkan penilaian yang tidak pernah dibuat Pega.
	Achievement string
}

// Kode metrik kartu skor NON-MBU.
const (
	MetricLeaderOverSLA    = "leader_lewat_sla"
	MetricLeaderTotal      = "total_klaim_leader"
	MetricLeaderPercent    = "persentase_leader"
	MetricLeaderScore      = "nilai_leader_sla"
	MetricLeaderWeight     = "bobot_leader"
	MetricLeaderSubtotal   = "subtotal_leader"
	MetricMemberOverSLA    = "member_lewat_sla"
	MetricMemberTotal      = "total_klaim_member"
	MetricMemberPercent    = "persentase_member"
	MetricMemberScore      = "nilai_member_sla"
	MetricMemberWeight     = "bobot_member"
	MetricMemberSubtotal   = "subtotal_member"
	MetricQuantitativeSum  = "total_kuantitatif"
	MetricAchievementRatio = "pencapaian_kuantitatif"
)

// Kode metrik kartu skor PA.
const (
	MetricRegisterOverSLA = "regist_klaim_lewat_sla"
	MetricRegisterTotal   = "total_klaim_regist"
	MetricRegisterScore   = "nilai_regist"
	MetricPaymentOverSLA  = "pembayaran_klaim_lewat_sla"
	MetricPaymentTotal    = "total_klaim_bayar"
	MetricPaymentScore    = "nilai_pembayaran"
)

// Bobot kartu skor NON-MBU, ditulis apa adanya dari `GetDataKPIAdmin-SQL.xml`.
//
// Keduanya TIDAK dihitung basis data melainkan dipilih sebagai literal, lalu dipakai
// mengalikan nilai. Ia ada di sini pula supaya kartu skornya dapat menggambarnya sebagai
// baris tersendiri — begitulah layar lama menampilkannya.
const (
	AdminLeaderWeight = 0.45
	AdminMemberWeight = 0.40
)

// AchievementReached dan AchievementMissed adalah teks kesimpulan kartu skor NON-MBU.
//
// Ambangnya rasio pencapaian `< 1`, persis `CASE` pada kueri lama. Teksnya pun apa adanya,
// termasuk bahwa keduanya berbahasa Indonesia tanpa tanda baca (`D-13`).
const (
	AchievementReached = "TERCAPAI TARGET"
	AchievementMissed  = "TIDAK TERCAPAI TARGET"
)

// AdminDetailRow adalah satu baris grid rincian tab KPI Admin.
//
// # Kenapa SATU tipe untuk DUA kelompok yang kolomnya berbeda
//
// Karena keduanya menggambarkan hal yang sama — satu klaim yang ditangani tim admin — dan
// yang berbeda hanyalah tahap mana yang diukur. Kolom yang tidak berlaku pada sebuah
// kelompok dibiarkan kosong, dan yang menentukan mana yang DIGAMBAR adalah daftar kolom
// grid pada screen.go, bukan ada-tidaknya isi.
//
// Memisahkannya menjadi dua tipe akan menggandakan pemindai, penyusun DTO, dan penyusun
// berkas ekspor — seluruhnya untuk membedakan lima kolom.
type AdminDetailRow struct {
	ClaimNumber  string // "No Klaim"
	PolicyNumber string // "No Polis"

	// BusinessName hanya NON-MBU — kolom "BUSINESS".
	BusinessName string

	// RegisterDate — "Tgl Regist Klaim". Berbentuk `YYYY-MM-DD`.
	RegisterDate string

	// TransferDate hanya NON-MBU — "Tgl Terima Dokumen", yaitu saat klaim diserahkan ke
	// PIC Teknik. Namanya di basis data `TRANSFERPIC_DATE`, dan judul kolomnya di layar
	// lama TIDAK menyebut PIC sama sekali (`D-13`).
	TransferDate string

	// TeamFlag hanya NON-MBU — kolom "Flag", berisi penanda leader atau member.
	TeamFlag string

	// RegisterAging adalah "Aging Regist Klaim" — TAT registrasi dalam hari kerja.
	RegisterAging Score

	// Lima isian berikut hanya PA.

	ReceiveDate    string // "Tgl Terima Dokumen"
	LODReceiveDate string // "Tgl Terima LOD"
	AcceptanceDate string // "Tgl Pembayaran"
	PaymentAging   Score  // "Aging Pembayaran klaim"
	RegisterSLA    string // "Status SLA Regist Klaim" — teks `SLA` / `TIDAK SLA`
	PaymentSLA     string // "Status SLA Pembayaran Klaim"

	// AdminName dan ClaimStatus hanya PA, dan keduanya TIDAK digambar sebagai kolom.
	//
	// Keduanya diambil kueri lama tetapi tidak muncul di grid mana pun. Dibawa supaya
	// berkas ekspor dapat menyertakannya kelak tanpa menyentuh kueri, dan supaya
	// penelusuran ke sistem lama tetap mungkin.
	AdminName   string
	ClaimStatus string
}

// AdminDetailPage adalah satu halaman grid rincian beserta jumlah seluruhnya.
type AdminDetailPage struct {
	Rows  []AdminDetailRow
	Total int
}
