package memory

import (
	"time"

	"claim-pnc/internal/mastermasking"
)

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
func SampleList() []mastermasking.Masking {
	// Waktu dipatok, bukan time.Now(), supaya urutan daftar contohnya selalu sama setiap
	// aplikasi dijalankan — daftar yang berpindah urutan tanpa sebab membuat pengujian
	// layar tampak gagal secara acak.
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

	return []mastermasking.Masking{
		{
			ID: "1", BranchID: "100081", BranchName: "KANTOR PUSAT",
			Login:  "CONTOH.ADMIN",
			Module: "PNCSearchKlaim", SubModule: "Registrasi,Dokumen,",
			SearchQuota: 100, ViewQuota: 100,
			ViewIDCard: true, ViewEmail: true, ViewPhone: true,
			Active: true, InputBy: "CONTOH.SUPERVISOR", InputAt: base.Add(120 * time.Hour),
		},
		{
			ID: "2", BranchID: "100069", BranchName: "BOGOR",
			Login:  "CONTOH.TEKNIK",
			Module: "PNCSearchKlaim", SubModule: "Penerimaan Pembayaran Klaim,Registrasi,Dokumen,",
			SearchQuota: 5, ViewQuota: 5,
			// Boleh melihat KTP dan telepon, tetapi TIDAK surel — kombinasi seperti ini
			// adalah yang paling sering ada di data asli, dan layar harus menampilkannya
			// sebagai tiga keputusan terpisah, bukan satu saklar.
			ViewIDCard: true, ViewEmail: false, ViewPhone: true,
			Active: true, InputBy: "CONTOH.SUPERVISOR", InputAt: base.Add(96 * time.Hour),
		},
		{
			ID: "3", BranchID: "100084", BranchName: "MALANG",
			Login:  "CONTOH.SURVEYOR",
			Module: "PNCSearchKlaim", SubModule: "Dokumen,",
			SearchQuota: 4, ViewQuota: 17,
			ViewIDCard: false, ViewEmail: true, ViewPhone: true,
			Active: true, InputBy: "CONTOH.ADMIN", InputAt: base.Add(72 * time.Hour),
		},
		{
			ID: "4", BranchID: "100089", BranchName: "YOGYAKARTA",
			Login:  "CONTOH.KASIR",
			Module: "PNCSearchKlaim", SubModule: "Penerimaan Pembayaran Klaim,",
			// Kuota besar memang ada di data asli. Ia dibawa supaya kolom angka di layar
			// diuji terhadap nilai yang lebar, bukan hanya satu digit.
			SearchQuota: 100000, ViewQuota: 100000,
			ViewIDCard: false, ViewEmail: false, ViewPhone: true,
			Active: true, InputBy: "CONTOH.ADMIN", InputAt: base.Add(48 * time.Hour),
		},
		{
			ID: "5", BranchID: "100004", BranchName: "KETAPANG",
			Login: "CONTOH.MUTASI",
			// Sub modul kosong: sah, dan ada padanannya di data asli.
			Module: "PNCSearchKlaim", SubModule: "",
			SearchQuota: 0, ViewQuota: 0,
			ViewIDCard: false, ViewEmail: false, ViewPhone: false,
			// Baris NONAKTIF — kewenangannya sudah dicabut, tetapi catatannya tetap ada.
			// Inilah wujud `D-66`: "hapus" tidak membuang baris.
			Active: false, InputBy: "CONTOH.SUPERVISOR", InputAt: base.Add(24 * time.Hour),
		},
	}
}

// SampleBranches adalah pilihan cabang tambahan untuk mode memori.
//
// Kelimanya belum dipakai baris contoh mana pun, sehingga menambah data untuk cabang baru
// dapat dicoba tanpa Oracle. Seluruhnya nama kantor perusahaan yang benar-benar ada, bukan
// karangan, supaya pencarian cabang berperilaku seperti di produksi.
func SampleBranches() []mastermasking.Branch {
	return []mastermasking.Branch{
		{ID: "100001", Name: "AGENCY MANADO"},
		{ID: "100052", Name: "AGENCY BINTARO"},
		{ID: "100054", Name: "GARUT"},
		{ID: "100104", Name: "MADIUN"},
		{ID: "100128", Name: "GRESIK"},
	}
}
