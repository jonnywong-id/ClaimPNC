package memory

import (
	_ "embed"

	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleList adalah data contoh mode memori.
//
// # Apa yang ditiru, dan apa yang TIDAK
//
// Yang ditiru adalah BENTUKNYA, bukan isinya:
//
//   - modul `PNCSearchKlaim` — satu-satunya nilai MODUL yang ada di portal ASM;
//   - sub modul berupa daftar dipisah koma DENGAN koma di ujung, persis seperti tersimpan;
//   - kuota yang berbeda-beda, termasuk satu yang sangat besar, karena sebaran aslinya
//     memang 1 sampai 100.000 — bentuk itu penting supaya layar diuji terhadap angka yang
//     benar-benar terjadi, bukan hanya angka kecil yang nyaman;
//   - campuran baris aktif dan nonaktif, karena 10 dari 25 baris asli nonaktif dan layar
//     harus terbaca benar untuk keduanya;
//   - satu baris tanpa sub modul, karena sub modul memang boleh kosong.
//
// Yang TIDAK ditiru adalah nama penggunanya. Login di sini dikarang dan sengaja dibuat
// terbaca sebagai contoh. Data produksi tidak disalin ke berkas yang di-commit (`D-69`),
// dan modul ini memuat KEWENANGAN MELIHAT DATA PRIBADI — menyalin daftar nama sungguhan ke
// dalam kode berarti menerbitkan daftar siapa yang boleh membuka nomor KTP nasabah.
//
// Nama cabang di bawah adalah nama kantor perusahaan, bukan data nasabah, sehingga ia
// dipakai apa adanya supaya pencarian berdasarkan nama cabang dapat dicoba dengan wajar.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Waktu dipatok, bukan time.Now(), supaya urutan daftar contohnya selalu sama setiap
// aplikasi dijalankan — daftar yang berpindah urutan tanpa sebab membuat pengujian
// layar tampak gagal secara acak.
// Boleh melihat KTP dan telepon, tetapi TIDAK surel — kombinasi seperti ini
// adalah yang paling sering ada di data asli, dan layar harus menampilkannya
// sebagai tiga keputusan terpisah, bukan satu saklar.
// Kuota besar memang ada di data asli. Ia dibawa supaya kolom angka di layar
// diuji terhadap nilai yang lebar, bukan hanya satu digit.
// Sub modul kosong: sah, dan ada padanannya di data asli.
// Baris NONAKTIF — kewenangannya sudah dicabut, tetapi catatannya tetap ada.
// Inilah wujud `D-66`: "hapus" tidak membuang baris.
func SampleList() []mastermasking.Masking {
	return sampledata.Must[[]mastermasking.Masking](sampleJSON, "SampleList")
}

// SampleBranches adalah pilihan cabang tambahan untuk mode memori.
//
// Kelimanya belum dipakai baris contoh mana pun, sehingga menambah data untuk cabang baru
// dapat dicoba tanpa Oracle. Seluruhnya nama kantor perusahaan yang benar-benar ada, bukan
// karangan, supaya pencarian cabang berperilaku seperti di produksi.
func SampleBranches() []mastermasking.Branch {
	return sampledata.Must[[]mastermasking.Branch](sampleJSON, "SampleBranches")
}
