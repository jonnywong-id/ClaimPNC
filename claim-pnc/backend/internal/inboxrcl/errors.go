package inboxrcl

import "errors"

// Galat domain modul Inbox RCL — tipe tersendiri, bukan teks (`11-CROSSCUTTING.md` §1.2).
var (
	// ErrCallerUnknown berarti identitas pemanggil tidak terbaca dari sesi.
	ErrCallerUnknown = errors.New("inboxrcl: identitas pemanggil tidak terbaca")

	// ErrClaimNotFound berarti klaim tidak ada di antrean RCL pemanggil — tidak ada, bukan
	// miliknya, atau sudah keluar dari antrean. Ketiganya sengaja tidak dibedakan: membedakannya
	// memberi tahu pengguna bahwa klaim milik orang lain itu ada.
	ErrClaimNotFound = errors.New("inboxrcl: klaim tidak ada di antrean RCL Anda")

	// ErrUnknownDecision berarti nilai keputusan bukan salah satu dari keempat tombol.
	ErrUnknownDecision = errors.New("inboxrcl: keputusan tidak dikenal")

	// ErrDecisionNotAllowed berarti tombol itu tidak ada pada mode layar klaim ini — Setuju dan
	// Tidak Setuju hanya pada mode RCL, Submit dan Back hanya pada mode MSIG.
	ErrDecisionNotAllowed = errors.New("inboxrcl: keputusan tidak berlaku untuk mode klaim ini")

	// ErrTechnicalPICUnknown berarti klaim yang dikembalikan ke analis tidak punya PIC Teknik.
	// Tugas Send To Analis adalah worklist yang wajib bertuan sejak lahir (`D-26`); tanpa
	// pemilik, klaim hilang dari setiap inbox tanpa galat.
	ErrTechnicalPICUnknown = errors.New("inboxrcl: PIC Teknik klaim tidak diketahui")
)
