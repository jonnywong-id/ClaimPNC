package inboxlaporanklaim

import (
	"fmt"
	"strings"
)

// FormatReportNumber menyusun nomor register laporan yang diterbitkan aplikasi ini.
//
// Bentuknya `RCVN.YY.xxxx` — tiga segmen dipisahkan titik, mengikuti `D-71` yang
// menetapkan bentuk itu untuk nomor klaim (`PNCN.YY.xxxx`). Alasan menirunya bukan
// keseragaman kosmetik: selama masa paralel, asal sebuah baris harus terbaca langsung
// dari nomornya tanpa tabel pemetaan, dan itu berlaku sama untuk berkas laporan.
//
// # Dua hal yang SENGAJA berbeda dari D-71
//
//  1. Nomor urutnya dipadatkan nol sampai empat digit. `D-71` butir 2 mencatat bahwa
//     `TO_CHAR(seq.NEXTVAL)` tanpa format mask membuat lebar segmen terakhir
//     berubah-ubah, sehingga pengurutan sebagai teks tidak sesuai urutan penerbitan —
//     ".10" mendahului ".9". Cacat itu tidak dibawa ke nomor baru: ia diketahui sebelum
//     baris pertama terbit, dan memperbaikinya sekarang tidak memutus data historis
//     apa pun karena belum ada satu pun nomor berbentuk ini.
//
//  2. Tahunnya diambil dari jam aplikasi lewat seam Clock, bukan dari `SYSDATE` basis
//     data. `D-71` mencatat `SYSDATE` sebagai catatan ketiganya — berkas yang terbit di
//     sekitar pergantian tahun mengambil tahun dari jam server basis data, yang bertaut
//     dengan `R-12`. Di sini sumber waktunya satu, dan dapat diuji.
//
// Nomor urut yang melewati empat digit TIDAK dipotong: ia tumbuh menjadi lima digit dan
// seterusnya. Memotongnya akan menerbitkan nomor ganda, dan nomor ganda jauh lebih mahal
// daripada kolom yang melebar.
func FormatReportNumber(year int, sequence int64) string {
	return fmt.Sprintf("%s.%02d.%04d", ReportNumberPrefix, year%100, sequence)
}

// IssuedHere menyatakan sebuah nomor diterbitkan aplikasi ini, bukan Pega.
//
// Pemeriksaannya dilakukan atas NOMOR, bukan atas kolom penanda, dan itu yang membuat
// asal sebuah baris tetap terbaca meski barisnya disalin ke tempat lain — misalnya ke
// berkas ekspor atau ke lampiran uji kesetaraan.
func IssuedHere(number string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(number)), ReportNumberPrefix+".")
}
