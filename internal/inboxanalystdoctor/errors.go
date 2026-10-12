package inboxanalystdoctor

import "errors"

// Galat domain modul Inbox Analyst Doctor.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP, dan
// domain tidak boleh tahu apa pun tentang HTTP (`11-CROSSCUTTING.md` §1.2 — kesalahan domain
// adalah tipe, bukan string).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Pada modul ini ia BUKAN gangguan kecil. Penyaring B Report Definition membandingkan
	// `pxAssignedOperatorID` dengan pemanggil, sehingga tanpa identitas tidak ada antrean
	// yang dapat ditampilkan sama sekali.
	//
	// Yang harus dihindari adalah jawaban "antrean Anda kosong" pada keadaan ini: kosong
	// terbaca sebagai tidak ada pekerjaan, dan tidak ada seorang pun yang melaporkannya.
	ErrCallerUnknown = errors.New("inboxanalystdoctor: identitas pemanggil tidak terbaca")
)
