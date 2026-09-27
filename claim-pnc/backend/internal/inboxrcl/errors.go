package inboxrcl

import "errors"

// Galat domain modul Inbox RCL.
//
// Ia tipe tersendiri, bukan teks: lapisan transport yang memetakannya ke kode HTTP
// (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	//
	// Tanpa login, identitas lamanya tidak dapat dicari, dan tanpa identitas lama tidak ada
	// antrean yang dapat ditampilkan. Menjawab "antrean Anda kosong" di sini akan terbaca
	// sebagai tidak ada pekerjaan — dan tidak ada seorang pun yang melaporkannya.
	ErrCallerUnknown = errors.New("inboxrcl: identitas pemanggil tidak terbaca")
)
