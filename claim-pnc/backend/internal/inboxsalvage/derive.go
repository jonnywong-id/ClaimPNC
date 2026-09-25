package inboxsalvage

import (
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// Teks kolom "Status Lelang".
//
// Keduanya DISALIN HARFIAH dari `RDB List/GcnmSalvageData_CloseOs_SQL-SQL.xml`. Mengubah
// satu huruf pun — misalnya menjadi "Belum terjual" dengan t kecil, seperti yang ditulis
// kueri ekspor — membuat baris lama dan baris baru tidak dapat dibandingkan pada uji
// kesetaraan.
const (
	AuctionSold   = "Terjual"
	AuctionUnsold = "Belum Terjual"
)

// AuctionStatusOf menurunkan kolom "Status Lelang" dari nilai akseptasi.
//
// Aturannya dari kueri itu sendiri:
//
//	NILAIAKSEP IS NOT NULL AND NILAIAKSEP != 0  -> "Terjual"
//	selain itu                                  -> "Belum Terjual"
//
// # Kenapa `!= 0` diperiksa, bukan hanya kekosongan
//
// Karena keduanya berbeda di sini dan perbedaannya bernilai uang: baris ber-`NILAIAKSEP`
// nol berarti lelangnya sudah berjalan tetapi tidak menghasilkan apa-apa, dan sistem lama
// menggambarnya sebagai BELUM terjual. Memperlakukan nol sebagai terjual akan melaporkan
// pemasukan yang tidak pernah ada.
//
// Perhatikan pula kolom ini TIDAK membaca `DETAIL_PNC_SALVAGE.STATUSTERJUAL`, yang punya
// lima keadaan berbeda. Kedua sumber dapat berselisih, dan yang digambar di grid adalah
// yang ini (`P-5`).
func AuctionStatusOf(acceptedValue string) string {
	clean := strings.TrimSpace(acceptedValue)
	if clean == "" {
		return AuctionUnsold
	}

	value, err := strconv.ParseFloat(strings.ReplaceAll(clean, ",", "."), 64)
	if err != nil {
		// Nilai yang tidak terbaca sebagai angka diperlakukan TERISI, sama seperti di
		// Oracle: di sana kolomnya bertipe angka, sehingga apa pun yang tersimpan di
		// dalamnya pasti angka. Nilai tak terbaca hanya dapat lahir dari data contoh.
		return AuctionSold
	}
	if value == 0 {
		return AuctionUnsold
	}
	return AuctionSold
}

// Teks kolom "Tipe Pengajuan".
//
// Keduanya disalin harfiah dari `Activity/PNCSalvageHistorySemuaKlaim_checker-Act.xml`
// langkah 2.4.
const (
	SubmissionNew     = "Pengajuan Baru"
	SubmissionRequest = "Request Balai Lelang"
)

// SubmissionTypeOf menurunkan kolom "Tipe Pengajuan" dari catatan request.
//
// Terisi atau tidaknya `DETAIL_PNC_SALVAGE.NOTE_REQUEST` yang menentukan, bukan isinya:
//
//	@if(Local.remarks="", "Pengajuan Baru", "Request Balai Lelang")
//
// Artinya baris yang balai lelangnya sudah pernah meminta nilai berbeda akan terbaca
// "Request Balai Lelang" selamanya — catatan itu tidak pernah dikosongkan kembali.
func SubmissionTypeOf(requestNote string) string {
	if strings.TrimSpace(requestNote) == "" {
		return SubmissionNew
	}
	return SubmissionRequest
}

// AgingOf menurunkan kolom "Aging" dari tanggal input salvage.
//
// # Bentuknya
//
// `"N day"` — angka, spasi, kata `day` dalam bahasa Inggris tunggal, berapa pun angkanya.
// Bentuk itu disalin harfiah dari `Activity/SetDataSalavage_act-Act.xml` langkah 43.3
// (`Param.CountBusiness + " day"`), termasuk ketiadaan bentuk jamaknya.
//
// # Yang BERBEDA dari sistem lama
//
// Angkanya. Sistem lama menghitung HARI KERJA lewat `GET_WORKING_HOURS` yang hidup di basis
// data lain (`@ASMD`); di sini yang dihitung adalah HARI KALENDER. Akibatnya angka di sini
// LEBIH BESAR untuk setiap baris yang melewati akhir pekan atau hari libur.
//
// `D-50` menetapkan perhitungan jam kerja dan kalender libur DITULIS ULANG di Go karena ia
// aturan bisnis, bukan pengambilan data — dan `D-25` mengganti DB Link dengan pemanggilan
// API. Keduanya menunggu kalender libur menjadi master data (`F-4`). Sampai itu tiba,
// selisihnya dinyatakan di PlannedDifferences, bukan disamarkan.
//
// Tanggal yang tidak terbaca menghasilkan teks KOSONG, bukan `"0 day"`. Keduanya berbeda
// dan perbedaannya penting: nol hari berarti salvage diajukan hari ini, sementara kosong
// berarti tanggalnya tidak diketahui. Menyamakannya mengulang cacat `GETSELISIHJAM` yang
// mengembalikan `0` saat gagal — butir 13 daftar perbaikan `P-5` (`D-49` #10).
func AgingOf(inputDate, today string) string {
	from, ok := parseISODate(inputDate)
	if !ok {
		return ""
	}
	to, ok := parseISODate(today)
	if !ok {
		return ""
	}

	days := clock.DaysBetween(from, to)
	if days < 0 {
		days = 0
	}
	return strconv.Itoa(days) + " day"
}

// DateLayout adalah bentuk tanggal yang dipakai kontrak API modul ini.
//
// Bentuk ISO, bukan `dd/mm/yyyy` seperti di layar lama. Alasannya sama dengan modul lain:
// `dd/mm/yyyy` tidak dapat dibedakan dari `mm/dd/yyyy` oleh pembacanya, dan satu kekeliruan
// di sana menghasilkan tanggal yang sah tetapi salah tanpa satu pun galat. Penerjemahannya
// ke bentuk yang dimengerti basis data dikerjakan penyimpanan.
const DateLayout = "2006-01-02"

// parseISODate membaca tanggal `YYYY-MM-DD` sebagai tengah malam WIB.
//
// WIB, bukan UTC. Bila keduanya dicampur, selisih hari di sekitar tengah malam bergeser
// satu — persis kelas cacat yang `F-5` hapus dan yang `R-12` catat.
func parseISODate(value string) (time.Time, bool) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return time.Time{}, false
	}

	// Tanggal yang datang dari basis data dapat membawa bagian waktu. Yang dipakai hanya
	// sepuluh huruf pertamanya — tanggalnya.
	if len(clean) > len(DateLayout) {
		clean = clean[:len(DateLayout)]
	}

	parsed, err := time.ParseInLocation(DateLayout, clean, clock.ZoneWIB)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// TodayWIB mengembalikan tanggal hari ini menurut WIB, berbentuk `YYYY-MM-DD`.
//
// Ia dipakai penyimpanan memori untuk menghitung Aging. Penyimpanan SQL memakai tanggal
// basis data, dan perbedaan itu disengaja: yang diuji di memori adalah bentuk dan aturan
// perhitungannya, bukan jam mesin mana yang dipercaya.
func TodayWIB() string {
	return clock.DateWIB(time.Now().UTC()).Format(DateLayout)
}
