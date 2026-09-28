package inboxlaporanklaim

import (
	"fmt"
	"strings"
)

// FormatReportNumber menyusun nomor register laporan yang diterbitkan aplikasi ini.
//
// Bentuknya `RCVN.YY.xxxx` — tiga segmen dipisahkan titik, mengikuti `D-71` yang
// menetapkan bentuk itu untuk nomor klaim (`PNCN.YY.xxxx`). Alasan menirunya bukan
// keseragaman kosmetik: selama masa paralel, asal sebuah baris harus terbaca langsung dari
// nomornya tanpa tabel pemetaan, dan itu berlaku sama untuk berkas laporan.
//
// # Bentuk ini sempat diganti, lalu dikembalikan
//
// Pada 2026-09-24 Work Owner menetapkan bentuk `RCVN-xxxx` tanpa segmen tahun, dan
// keesokan harinya membatalkannya — bentuk bertitik dipakai kembali. Satu berkas telanjur
// terbit dengan bentuk itu (`RCVN-0015`), dan ia TETAP milik aplikasi ini; lihat
// IssuedHere.
//
// # Deret dihitung PER TAHUN
//
// Segmen tahun ada, sehingga deretnya boleh berulang tiap tahun tanpa menerbitkan nomor
// ganda: `RCVN.26.0001` dan `RCVN.27.0001` adalah dua nomor yang berbeda.
//
// # Nomor urut dipadatkan nol sampai empat digit, dan TUMBUH bila terlampaui
//
// Lebar tetap membuat urutan teks sama dengan urutan angka DI DALAM SATU TAHUN, sehingga
// `ORDER BY` atas kolomnya benar tanpa mengurai apa pun. `D-71` butir 2 mencatat cacat
// sebaliknya pada nomor klaim: tanpa pemadatan, ".10" mendahului ".9".
//
// Memotongnya pada empat digit akan menerbitkan nomor ganda pada berkas ke-10.001, dan
// nomor ganda jauh lebih mahal daripada kolom yang melebar.
func FormatReportNumber(year int, sequence int64) string {
	return fmt.Sprintf("%s.%02d.%04d", ReportNumberPrefix, year%100, sequence)
}

// IssuedHere menyatakan sebuah nomor diterbitkan aplikasi ini, bukan Pega.
//
// Pemeriksaannya dilakukan atas NOMOR, bukan atas kolom penanda, dan itu yang membuat
// asal sebuah baris tetap terbaca meski barisnya disalin ke tempat lain — misalnya ke
// berkas ekspor atau ke lampiran uji kesetaraan.
//
// # DUA bentuk diterima, dan pemisahnya tetap diperiksa
//
// Keduanya milik aplikasi ini: `RCVN.YY.xxxx` yang berlaku dan diterbitkan sekarang, dan
// `RCVN-xxxx` pada satu berkas yang telanjur terbit saat bentuk itu sempat dipakai
// (2026-09-24). Menolak yang kedua berarti berkas itu mendadak terbaca sebagai milik Pega
// — dan berkas milik Pega hanya dapat dibaca (`P-1`), sehingga ia berhenti dapat disunting
// tanpa satu pun galat yang menjelaskan sebabnya.
//
// Penerbitan dan pengenalan karena itu punya aturan yang BERBEDA, dan itu disengaja:
// yang diterbitkan hanya satu bentuk, yang dikenali dua.
//
// Pemisahnya tetap wajib ada. `RCVN0001` tanpa pemisah BUKAN nomor modul ini, dan
// memeriksa awalan saja akan meloloskan nomor apa pun yang kebetulan berawalan sama.
func IssuedHere(number string) bool {
	clean := strings.ToUpper(strings.TrimSpace(number))
	return strings.HasPrefix(clean, ReportNumberPrefix+"-") ||
		strings.HasPrefix(clean, ReportNumberPrefix+".")
}
