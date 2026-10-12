package reportkpi

import (
	"fmt"
	"time"
)

// Berkas ini memuat isian **"Periode KPI"** — dropdown penyaring blok PA pada tab KPI
// Admin.
//
// # Kenapa blok PA berbeda dari blok NON MBU
//
// Keduanya memang berbeda di layar lama, dan itu bukan kelalaian ekspor. Pada
// `Section/ReportKPI_Section-Section.xml`:
//
//	blok NON MBU   `Dari` + `Sampai`, dua isian tanggal
//	blok PA        `Periode KPI`, SATU dropdown — terikat `TempLaporan.Province`
//
// Masing-masing punya tombol `Cari` dan `Export Detail Data` sendiri. Jadi tab Admin
// sebenarnya dua penyaring yang berdiri sendiri, bukan satu penyaring bersama.
//
// # Dari mana isi dropdownnya
//
// `Activity/AddPeriodeKPIADMIN -Act.xml` — dua perulangan, masing-masing 12 kali:
//
//	perulangan 1   tahun LALU, bulan 01..12
//	perulangan 2   tahun INI, bulan 01..12
//
// Nilainya `YYYYMM` dan ditampilkan apa adanya (`pyValue` dan `pyPrompt` sama-sama
// `.City`). Pilihan pertama menjadi bawaan.
//
// Bulan terpilih kemudian diubah menjadi rentang satu bulan penuh — lihat
// AdminPeriodRange.

// AdminPeriodOption adalah satu baris dropdown "Periode KPI".
//
// Code dan Label sengaja sama: layar lama menampilkan `YYYYMM` apa adanya, tanpa nama
// bulan dan tanpa pemisah. Menerjemahkannya menjadi "Januari 2026" akan membuat angka
// yang disebut pengguna lewat telepon tidak lagi cocok dengan yang terlihat di Pega
// selama masa paralel (`D-13`).
type AdminPeriodOption struct {
	Code  string
	Label string
}

// adminPeriodMonths adalah jumlah bulan per tahun pada dropdown.
//
// Dua belas, apa adanya seperti `pyStepsRepeatDefLimit` pada kedua perulangan — termasuk
// bulan tahun ini yang BELUM terjadi. Memotongnya di bulan berjalan akan menghilangkan
// pilihan yang di Pega ada, dan itu selisih yang tidak diminta siapa pun.
const adminPeriodMonths = 12

// BuildAdminPeriods menyusun isi dropdown "Periode KPI" terhadap satu titik waktu.
//
// Ia menerima `now` alih-alih memanggil jam sendiri supaya dapat diuji di sekitar
// pergantian tahun — satu-satunya saat isinya berubah, dan satu-satunya saat ia mungkin
// salah.
func BuildAdminPeriods(now time.Time) []AdminPeriodOption {
	year := now.Year()

	options := make([]AdminPeriodOption, 0, adminPeriodMonths*2)
	for _, y := range []int{year - 1, year} {
		for month := 1; month <= adminPeriodMonths; month++ {
			code := fmt.Sprintf("%04d%02d", y, month)
			options = append(options, AdminPeriodOption{Code: code, Label: code})
		}
	}
	return options
}

// AdminPeriodRange mengubah `YYYYMM` menjadi rentang satu bulan penuh.
//
// Keduanya INKLUSIF, meniru `PNCReportKPIAdmin_Act_khususPA`:
//
//	Local.awal    "01/MM/YYYY"
//	Local.akhir   "<hari terakhir>/MM/YYYY"
//
// lalu dipasang sebagai `trunc(b.registerdate) >= awal and trunc(b.registerdate) <= akhir`.
//
// Hari terakhir dihitung, bukan didaftar: maju satu bulan lalu mundur satu hari. Dengan
// begitu Februari tahun kabisat ikut benar tanpa perkecualian yang perlu diingat siapa
// pun.
func AdminPeriodRange(code string) (from, to time.Time, err error) {
	if len(code) != 6 {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"reportkpi: periode KPI %q tidak berbentuk YYYYMM", code)
	}

	first, err := time.ParseInLocation("200601", code, time.UTC)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf(
			"reportkpi: periode KPI %q tidak berbentuk YYYYMM: %w", code, err)
	}

	last := first.AddDate(0, 1, -1)
	return first, last, nil
}
