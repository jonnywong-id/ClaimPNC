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
)

// BranchUnknownNotice adalah pesan yang dibaca pengguna ketika cabangnya tidak diketahui.
//
// Teksnya diambil APA ADANYA dari `Activity/OutstandingperCabang_PreAct-Act.xml` langkah 2,
// termasuk huruf kapitalnya. `D-13` menetapkan teks yang dilihat pengguna mengikuti layar
// Pega, dan pesan inilah yang selama ini mereka baca beserta tindak lanjutnya — menghubungi
// Tim IT.
const BranchUnknownNotice = "Belum ada Data Cabang pada Akun ini. Mohon Menghubungi Tim IT"
