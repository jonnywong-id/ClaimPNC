package inboxlaporanklaim

import (
	"errors"
	"strings"
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrUnknownCategory: tab yang diminta tidak ada.
	ErrUnknownCategory = errors.New("inboxlaporanklaim: kategori tidak dikenal")

	// ErrUnknownBusinessLine: pilihan lini bisnis tidak ada di daftar.
	//
	// Menolak nilai yang tidak dikenal adalah kendali yang di layar diberikan oleh
	// dropdown. Diterima apa adanya, kendali itu hilang begitu permintaan datang dari
	// luar layar — dan API memang dapat ditembak langsung.
	ErrUnknownBusinessLine = errors.New("inboxlaporanklaim: lini bisnis tidak dikenal")

	// ErrCallerUnknown: identitas pemanggil tidak diketahui.
	//
	// Ia BUKAN galat sesi. Sesi sudah diperiksa middleware jauh sebelum sampai ke sini;
	// yang ini berarti rakitan di cmd tidak memasang jembatan ke konteks pemanggil, dan
	// itu cacat pemrograman yang harus terlihat.
	ErrCallerUnknown = errors.New("inboxlaporanklaim: identitas pemanggil tidak diketahui")

	// ErrBranchUnknown: cabang klaim petugas tidak dapat ditentukan dari login-nya.
	//
	// # Riwayatnya, karena ia sempat dicabut lalu dikembalikan dengan arti yang berbeda
	//
	// Versi pertama modul ini memakai galat bernama sama untuk menolak PEMBUATAN berkas,
	// dan itu dicabut dengan benar: `Activity/CreateNewCaseRCV-Act.xml` langkah 19
	// mengisi cabang dari hasil `GetIDCabang` dan TIDAK memeriksa hasilnya sama sekali.
	// Menolak di sana adalah aturan baru yang tidak ada di 13 butir `D-49`.
	//
	// Yang sekarang berbeda sebabnya: **Work Owner menetapkan 2026-09-22 bahwa petugas
	// yang cabangnya tidak terbaca TIDAK boleh melihat seluruh cabang.** Itu keputusan
	// batas data, bukan tafsiran atas perilaku Pega — dan sebuah keputusan Work Owner
	// adalah dasar yang sah untuk menyimpang, persis seperti butir-butir `D-49`.
	//
	// Di sistem lama keadaan ini menghasilkan `branch where ID=''` sehingga daftarnya
	// kosong. Kekosongan itu tidak ditiru: daftar kosong tidak terbedakan dari "tidak ada
	// pekerjaan hari ini", dan ketidakterbedaan itulah cacat yang membuat modul ini harus
	// diperbaiki pada mulanya. Yang dikembalikan adalah penolakan yang MENYEBUTKAN
	// sebabnya.
	ErrBranchUnknown = errors.New("inboxlaporanklaim: cabang klaim petugas tidak dikenali")

	// ErrBranchUnreadable: sumber data cabang tidak dapat dibaca.
	//
	// Dipisahkan dari ErrBranchUnknown meski akibatnya di layar sama — keduanya menutup
	// layar. Sebab dan perbaikannya berbeda jauh:
	//
	//	ErrBranchUnknown     login petugas belum terdaftar di HRD → urusan data pegawai
	//	ErrBranchUnreadable  POOLDATA.BRANCH atau DB link @asmd tidak terbaca → urusan
	//	                     infrastruktur, dan ia menimpa SELURUH petugas sekaligus
	//
	// Menyatukan keduanya akan membuat gangguan infrastruktur terbaca sebagai kesalahan
	// data satu orang, lalu dicari di tempat yang salah.
	ErrBranchUnreadable = errors.New("inboxlaporanklaim: sumber cabang klaim tidak dapat dibaca")

	// ErrNotFound: berkas laporan yang diminta tidak ada.
	ErrNotFound = errors.New("inboxlaporanklaim: laporan klaim tidak ditemukan")

	// ErrReadOnlyOrigin: baris milik Pega tidak boleh ditulis aplikasi ini.
	//
	// Selama masa paralel, tepat satu sistem yang menulis sebuah baris (`ADR-0004`).
	// Baris ber-Origin OriginLegacy dimiliki Pega; menulisnya dari sini berarti dua
	// sistem menulis satu baris dengan aturan validasi yang berbeda — kelas cacat yang
	// `P-1` ada untuk mencegahnya.
	ErrReadOnlyOrigin = errors.New("inboxlaporanklaim: laporan milik Pega hanya dapat dibaca")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Kesetaraan perilaku, bukan selera (`P-5`): layar lama menampilkan seluruh pesan
// bersamaan, dan mengembalikan satu galat per percobaan akan membuat pengguna menebak
// isian mana lagi yang salah (`11-CROSSCUTTING.md` §1.2 aturan 1).
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violation))
	for _, p := range e.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "inboxlaporanklaim: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}
