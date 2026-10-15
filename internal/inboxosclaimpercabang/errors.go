package inboxosclaimpercabang

import "errors"

// Galat domain modul Inbox OS Claim per Cabang.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	ErrCallerUnknown = errors.New(
		"inboxosclaimpercabang: identitas pemanggil tidak terbaca")

	// ErrBranchUnknown berarti cabang pemanggil tidak dapat ditentukan.
	//
	// # Ia galat, bukan daftar kosong — dan itu meniru sistem lama, bukan menyimpang darinya
	//
	// `Activity/OutstandingperCabang_PreAct-Act.xml` langkah 2 menyetel pita peringatan
	// ketika `OperatorID.pyTelephone == ""`, yakni ketika kode cabang pemanggil kosong.
	// Layar lama karena itu MEMBERI TAHU, bukan menampilkan grid kosong yang tidak
	// menjelaskan dirinya.
	//
	// Perilaku itu justru makin penting di sini: batas data layar ini SELURUHNYA cabang.
	// Daftar kosong dan "Anda belum punya cabang" terlihat sama di layar, padahal yang
	// pertama berarti tidak ada pekerjaan dan yang kedua berarti layar tidak dapat bekerja.
	ErrBranchUnknown = errors.New(
		"inboxosclaimpercabang: cabang pemanggil tidak dapat ditentukan")

	// ErrClaimNotFound berarti klaim yang diminta tidak ada DI CABANG PEMANGGIL.
	//
	// Kedua sebabnya sengaja tidak dibedakan — klaim yang memang tidak ada, dan klaim yang
	// ada tetapi milik cabang lain. Membedakannya membuat endpoint ini dapat dipakai
	// memastikan sebuah nomor klaim ada di cabang lain, cukup dengan membaca pesannya
	// (`R-20`).
	ErrClaimNotFound = errors.New(
		"inboxosclaimpercabang: klaim tidak ditemukan pada cabang pemanggil")
)

// ClaimNotFoundNotice adalah pesan yang dibaca pengguna ketika klaimnya tidak ditemukan.
//
// Ia TIDAK ada padanannya di Pega — popup lama hanya dapat dibuka dari baris yang sudah
// tampil, sehingga keadaan ini tidak pernah terjadi di sana. Karena tidak ada teks lama yang
// dapat ditiru (`D-13`), teksnya ditulis baru dalam bahasa Indonesia dan menyebutkan tindak
// lanjutnya, bukan hanya menyatakan kegagalan.
const ClaimNotFoundNotice = "Klaim tidak ditemukan pada cabang Anda. " +
	"Muat ulang daftar, lalu buka kembali dari barisnya."

// BranchUnknownNotice adalah pesan yang dibaca pengguna ketika cabangnya tidak diketahui.
//
// Teksnya diambil APA ADANYA dari `Activity/OutstandingperCabang_PreAct-Act.xml` langkah 2,
// termasuk huruf kapitalnya. `D-13` menetapkan teks yang dilihat pengguna mengikuti layar
// Pega, dan pesan inilah yang selama ini mereka baca beserta tindak lanjutnya — menghubungi
// Tim IT.
const BranchUnknownNotice = "Belum ada Data Cabang pada Akun ini. Mohon Menghubungi Tim IT"
