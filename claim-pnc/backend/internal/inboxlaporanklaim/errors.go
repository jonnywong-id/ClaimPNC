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

	// # Kenapa TIDAK ada ErrBranchUnknown
	//
	// Versi pertama modul ini menolak pembuatan berkas ketika cabang pemanggil tidak
	// terbaca, dengan alasan berkas tanpa cabang akan hilang dari daftar yang disaring
	// cabang. Penolakan itu DICABUT setelah dibandingkan ke sumbernya.
	//
	// `Activity/CreateNewCaseRCV-Act.xml` langkah 19 mengisi cabang dari hasil
	// `GetIDCabang`, dan TIDAK memeriksa hasilnya sama sekali: bila kueri itu tidak
	// mengembalikan baris, nilainya kosong dan berkas tetap dibuat. Menolaknya adalah
	// aturan BARU, bukan aturan yang dipindahkan — dan `P-5` menetapkan perilaku
	// dipertahankan lebih dulu, kecuali untuk 13 butir yang `D-49` sebut satu per satu.
	// Penolakan ini tidak ada di antaranya.
	//
	// Akibat yang disadari: berkas yang lahir tanpa cabang tetap terlihat pembuatnya —
	// penyaring cabang tidak berlaku bagi pemanggil yang cabangnya kosong — tetapi TIDAK
	// terlihat petugas cabang mana pun. Itu perilaku sistem lama, dan memperbaikinya
	// adalah keputusan Work Owner, bukan keputusan modul ini.

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
