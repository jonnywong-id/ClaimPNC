package inboxsalvage

// CountSource menyatakan dari mana sebuah baris pencacah menghitung.
//
// Ia perlu dinyatakan, bukan disimpulkan dari tabnya, karena DUA baris menghitung populasi
// yang berbeda dari daftar yang dibukanya — dan itu direplikasi (`P-5`). Lihat CountRow.
type CountSource string

const (
	// CountFromClaim menghitung `POOLDATA.T_CLAIM_PNC` menurut `STSSALVAGE`.
	CountFromClaim CountSource = "klaim"

	// CountFromClaimBuyback menghitung klaim yang punya baris adjustment ber-nilai
	// salvage terisi.
	CountFromClaimBuyback CountSource = "klaim-buyback"

	// CountFromSalvage menghitung `POOLDATA.PNC_SALVAGE` menurut `STSTRANSFER`.
	CountFromSalvage CountSource = "salvage"

	// CountFromSalvageDetail menghitung `POOLDATA.DETAIL_PNC_SALVAGE` menurut
	// `STATUSTERJUAL`.
	CountFromSalvageDetail CountSource = "detail-salvage"
)

// CountRow adalah satu baris tabel ringkas "Status Salvage / Jumlah", beserta cara
// menghitungnya.
//
// # Kenapa ia tidak sekadar memakai penyaring tabnya
//
// Karena di Pega pun tidak. Dua baris menghitung populasi yang BERBEDA dari daftar yang
// dibukanya:
//
//	"Outstanding"  pencacah: STSSALVAGE 3 atau 5     daftar: STSSALVAGE kosong
//	"Checker"      pencacah: STSTRANSFER 3 atau 5    daftar: STSTRANSFER 3
//
// Keputusan Work Owner 2026-09-25: keduanya direplikasi apa adanya, karena angkanya
// berjalan dan dibaca orang setiap hari. Menurunkan pencacah dari penyaring tab akan
// "memperbaiki" keduanya diam-diam.
type CountRow struct {
	// Label adalah teks kolom "Status Salvage", disalin harfiah dari
	// `Activity/GCNMCountSalvage_act-Act.xml`.
	Label string

	// Tab adalah kode tab yang dibuka bila barisnya diklik. Kosong berarti barisnya tidak
	// menuju daftar mana pun.
	Tab string

	// Source menyatakan tabel mana yang dihitung.
	Source CountSource

	// SalvageStatuses adalah nilai `STSSALVAGE` yang dihitung, untuk CountFromClaim.
	SalvageStatuses []string

	// SalvageStatusIsNull menyatakan yang dihitung adalah `STSSALVAGE IS NULL`.
	SalvageStatusIsNull bool

	// TransferStatuses adalah nilai `STSTRANSFER` yang dihitung, untuk CountFromSalvage.
	TransferStatuses []string

	// OwnedByCaller menyatakan hitungan ini dibatasi baris milik pemanggil.
	OwnedByCaller bool

	// HiddenReason menyatakan baris ini TIDAK digambar, beserta alasannya.
	//
	// Kosong berarti digambar. Nilainya memakai konstanta yang sama dengan Tab, sebab
	// alasannya memang sama — dan keduanya harus berubah bersamaan: baris pencacah tanpa
	// daftarnya adalah angka yang tidak dapat ditindaklanjuti, dan daftar tanpa baris
	// pencacahnya tidak dapat ditemukan.
	HiddenReason string
}

// CountRows adalah keempat belas baris pencacah, berurutan seperti tampilnya.
//
// Urutannya mengikuti urutan langkah `Activity/GCNMCountSalvage_act-Act.xml` yang
// menuliskannya: langkah 10, 16, 18, 21, 23, 24, 26, 28, 32, 36, 40, 44, 46, dan 50.
//
// # Baris yang di Pega TIDAK selalu tergambar, dan keputusannya
//
// PERTAMA — dua baris tidak pernah tergambar sama sekali. "Salvage Diterima" dan "Salvage
// Ditolak" ditulis ke daftar BERSARANG (`TempALLSalvage.pxResults(1).pxResults(<APPEND>)`)
// alih-alih ke daftar utamanya. Keputusan Work Owner 2026-09-25: diperbaiki, karena kode
// itu tidak pernah berjalan sebagaimana dimaksud penulisnya.
//
// KEDUA — baris mana yang tergambar bergantung pada SIAPA yang membuka layar, dan
// syaratnya adalah DUA NAMA ORANG yang tertanam di dalam activity:
//
//	langkah 18  "Checker"           dilewati BAGI kedua nama itu
//	langkah 21  "Salvage Diterima"  hanya BAGI kedua nama itu
//	langkah 23  "Salvage Ditolak"   hanya BAGI kedua nama itu
//
// Artinya dua orang melihat tabel ringkas yang berbeda dari semua orang lain. `D-15`
// menetapkan tidak ada nilai bisnis yang boleh di-hardcode, dan nama orang sebagai penentu
// perilaku adalah tepat yang dilarangnya — ia salah satu dari 24 Operator ID yang `F-4`
// hapus. Ketiga baris karena itu tergambar untuk SEMUA pengguna di sini, dan selisihnya
// dinyatakan di PlannedDifferences.
//
// Nama kedua operator itu TIDAK direproduksi di berkas ini. `D-69` membolehkannya, tetapi
// tidak ada gunanya di sini: yang perlu diketahui pembaca adalah bahwa syaratnya nama
// orang, bukan nama siapa.
func CountRows() []CountRow {
	result := []CountRow{}
	for _, row := range allCountRows() {
		if row.HiddenReason == "" {
			result = append(result, row)
		}
	}
	return result
}

// AllCountRows adalah SELURUH baris pencacah, termasuk yang tidak digambar.
//
// Dipakai uji dan penelusuran, bukan oleh layar.
func AllCountRows() []CountRow { return allCountRows() }

func allCountRows() []CountRow {
	return []CountRow{
		{
			// `CountSalvage_sql11OS` — dan perhatikan ia menghitung STSSALVAGE 3 atau 5,
			// bukan yang kosong seperti daftarnya.
			Label:           "Outstanding",
			Tab:             TabOutstanding,
			Source:          CountFromClaim,
			SalvageStatuses: []string{"3", "5"},
		},
		{
			Label:           "Ekonomis",
			Tab:             TabEkonomis,
			Source:          CountFromClaim,
			SalvageStatuses: []string{"3"},
		},
		{
			// `CountSalvage_sql11` alias "CityID" — STSTRANSFER 3 ATAU 5, sementara
			// daftar Checker menyaring 3 saja.
			Label:            "Checker",
			HiddenReason:     HiddenManagerOnly,
			Tab:              TabChecker,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"3", "5"},
		},
		{
			// `CountSalvageDiterimaSalvage`.
			Label:            "Salvage Diterima",
			HiddenReason:     HiddenManagerOnly,
			Tab:              TabSalvageDiterima,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"3"},
		},
		{
			// `CountSalvageDitolakSalvage`.
			Label:            "Salvage Ditolak",
			HiddenReason:     HiddenManagerOnly,
			Tab:              TabSalvageDitolak,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"5"},
		},
		{
			// `CountSalvage_sql11` alias "Email".
			Label:            "Rejected Checker",
			Tab:              TabRejectedChecker,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"4"},
		},
		{
			// `CountSalvage_sql11` alias "ClaimNo".
			Label:            "Balai Lelang",
			Tab:              TabBalaiLelang,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"1"},
		},
		{
			// `CountSalvage_sql11` alias "EmailBroker" — satu-satunya hitungan yang
			// dibatasi pemanggil.
			Label:            "Request Balai Lelang",
			HiddenReason:     HiddenNotOnScreen,
			Tab:              TabRequestBalai,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"7"},
			OwnedByCaller:    true,
		},
		{
			Label:           "Tidak Ekonomis",
			Tab:             TabTidakEkonomis,
			Source:          CountFromClaim,
			SalvageStatuses: []string{"4"},
		},
		{
			Label:           "TBA",
			Tab:             TabTBA,
			Source:          CountFromClaim,
			SalvageStatuses: []string{"5"},
		},
		{
			Label:           "Tidak Ada Salvage",
			Tab:             TabTidakAdaSalvage,
			Source:          CountFromClaim,
			SalvageStatuses: []string{"1"},
		},
		{
			Label:  "Salvage Buyback",
			Tab:    TabBuyback,
			Source: CountFromClaimBuyback,
		},
		{
			// `CountSalvage_sql11` alias "City" — STSTRANSFER 1 atau 6, sementara daftar
			// Histori tidak menyaring sama sekali. Ini selisih KETIGA antara pencacah dan
			// daftarnya, dan ia direplikasi dengan alasan yang sama.
			Label:            "Histori Salvage",
			Tab:              TabHistori,
			Source:           CountFromSalvage,
			TransferStatuses: []string{"1", "6"},
		},
		{
			// Baris ini TIDAK menuju daftar mana pun, dan itu keadaan di Pega: tidak ada
			// satu pun tab yang menerima kodenya. Ia tetap digambar karena activity
			// pencacah menggambarnya tanpa syarat apa pun.
			Label:        "Tidak Terjual",
			HiddenReason: HiddenNotOnScreen,
			Tab:          "",
			Source:       CountFromSalvageDetail,
		},
	}
}
