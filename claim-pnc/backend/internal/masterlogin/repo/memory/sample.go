package memory

import (
	_ "embed"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

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
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Leader tim. Ketiga baris berikutnya menunjuknya lewat LOGINLEADER, sehingga
// penambahan yang dilakukan salah satu dari mereka menurunkan leader yang sama.
// Tanda hubung pada Nama dibuang saat LOGIN diturunkan.
// ALAMAT kosong — isian yang memang tidak wajib.
// LOGINLEADER kosong — surveyor yang tidak bertaut ke tim mana pun.
func SampleList() []masterlogin.SurveyorLogin {
	return sampledata.Must[[]masterlogin.SurveyorLogin](sampleJSON, "SampleList")
}
