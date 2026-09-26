package memory

import "claim-pnc/internal/masterlogin"

// SampleList adalah isi contoh untuk pengembangan tanpa Oracle.
//
// # Seluruh nama, surel, telepon, dan alamat di sini KARANGAN
//
// Bukan surveyor nyata, bukan alamat nyata, dan bukan nomor nyata. Surelnya memakai domain
// `contoh.invalid` — ranah tingkat atas yang RFC 2606 cadangkan supaya tidak pernah dapat
// diselesaikan DNS, sehingga contoh yang tidak sengaja terkirim tidak akan sampai ke siapa
// pun.
//
// Aturannya sendiri berasal dari `D-69`: alamat surel **selalu disamarkan** di artefak yang
// di-commit. Di sini bahkan tidak ada yang disamarkan — tidak satu pun alamat nyata pernah
// masuk ke berkas ini.
//
// # Yang diwakilinya, dan kenapa persis ini
//
// Enam baris, dipilih supaya setiap keadaan yang menentukan perilaku layar dapat dicoba
// tanpa menyiapkan data sendiri:
//
//   - **Seorang leader beserta tiga anggotanya.** `LOGINLEADER` ketiganya menunjuk
//     `BudiHartono`, dan baris leader itu sendiri ada di daftar. Itu yang membuat penurunan
//     LOGINLEADER pada penambahan benar-benar teruji — lihat usecase.Service.Create.
//   - **Satu baris LOGINLEADER kosong.** Keadaan yang sah, dan yang tidak dapat dibedakan
//     dari "gagal menemukan timnya"; lihat usecase.noteLeaderMissing.
//   - **Satu baris ALAMAT kosong.** Alamat memang tidak wajib (`pyRequired = false`), dan
//     tanpa contoh seperti ini kolom kosong di grid tidak pernah terlihat saat mencoba.
//
// # LOGIN-nya konsisten dengan penurunannya
//
// Setiap `Login` di bawah sama persis dengan `masterlogin.DeriveLogin(Name)` — spasi,
// titik, koma, dan tanda hubung dibuang, tanpa perubahan huruf besar-kecil. Itu dijaga uji
// di masterlogin_test.go.
//
// Dua di antaranya sengaja punya nama yang menuntut pembuangan: "Rina Ayu Lestari" (dua
// spasi) dan "Siti Nur-Halimah" (tanda hubung). Tanpa keduanya, penurunan LOGIN tampak
// seperti sekadar menyalin Nama.
//
// # STSLOGIN semuanya "Member"
//
// Tidak satu pun rule di export menuliskan nilai lain (`R-16`), dan modul ini pun hanya
// menuliskan itu. Baris leader-nya juga "Member" — perannya sebagai leader dinyatakan oleh
// baris LAIN yang menunjuknya lewat LOGINLEADER, bukan oleh kolom ini. Itu bentuk data yang
// membingungkan, dan ia ditiru apa adanya karena itulah yang ada.
func SampleList() []masterlogin.SurveyorLogin {
	return []masterlogin.SurveyorLogin{
		{
			// Leader tim. Ketiga baris berikutnya menunjuknya lewat LOGINLEADER, sehingga
			// penambahan yang dilakukan salah satu dari mereka menurunkan leader yang sama.
			Name:        "Budi Hartono",
			Login:       "BudiHartono",
			Email:       "budi.hartono@contoh.invalid",
			Phone:       "021-5550101",
			Address:     "Jl. Melati Raya No. 12, Jakarta Selatan",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "",
		},
		{
			Name:        "Rina Ayu Lestari",
			Login:       "RinaAyuLestari",
			Email:       "rina.lestari@contoh.invalid",
			Phone:       "021-5550102",
			Address:     "Jl. Kenanga No. 7, Jakarta Pusat",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "BudiHartono",
		},
		{
			// Tanda hubung pada Nama dibuang saat LOGIN diturunkan.
			Name:        "Siti Nur-Halimah",
			Login:       "SitiNurHalimah",
			Email:       "siti.halimah@contoh.invalid",
			Phone:       "0811-5550103",
			Address:     "Jl. Anggrek No. 3, Bandung",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "BudiHartono",
		},
		{
			// ALAMAT kosong — isian yang memang tidak wajib.
			Name:        "Agus Pratama",
			Login:       "AgusPratama",
			Email:       "agus.pratama@contoh.invalid",
			Phone:       "0812-5550104",
			Address:     "",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "BudiHartono",
		},
		{
			// LOGINLEADER kosong — surveyor yang tidak bertaut ke tim mana pun.
			Name:        "Dewi Kartika",
			Login:       "DewiKartika",
			Email:       "dewi.kartika@contoh.invalid",
			Phone:       "0813-5550105",
			Address:     "Jl. Cendana No. 21, Surabaya",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "",
		},
		{
			Name:        "Eko Nugroho",
			Login:       "EkoNugroho",
			Email:       "eko.nugroho@contoh.invalid",
			Phone:       "0814-5550106",
			Address:     "Jl. Diponegoro No. 45, Semarang",
			LoginStatus: masterlogin.LoginStatusMember,
			LeaderLogin: "DewiKartika",
		},
	}
}
