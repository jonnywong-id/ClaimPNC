package komitehttp

import "github.com/go-chi/chi/v5"

// Mount mendaftarkan seluruh rute modul Komite ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di cmd/claimpnc
// ia dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga middleware sesi.
//
// # Kenapa rutenya terbelah di dua tempat
//
// Master ambang berada di bawah `master/`, bersama Master Status Klaim, karena ia data
// acuan dan berbagi satu menu serta satu kewenangan dengan master lain (`D-59`, satuan
// izin adalah menu). Perhitungan penjenjangan berada di bawah `komite/` karena ia
// bukan data acuan melainkan aturan bisnis modul `B-7`.
//
// Pembelahan itu mengikuti kepemilikan modul: tangga ambangnya milik `F-4`, cara
// membacanya milik `B-7`. Menaruh keduanya di satu awalan akan menyamarkan batas itu,
// dan penegakan izin per menu kelak harus dipasang pada dua kewenangan yang berbeda.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya. Penegakan "apakah peran
// pemanggil memiliki menu ini" adalah `TKT-F3-005`, yang bergantung pada tabel peran
// `TKT-F3-004` — dan tabel itu dapat dibangun tetapi belum dapat diisi, karena penugasan
// operator ke peran tidak ada di basis data maupun di export
// (`docs/Steering/11-SECURITY.md` §3.1).
//
// Keadaan ini sama dengan seluruh rute lain yang sudah ada hari ini, dan dicatat terbuka
// di `docs/keputusan-implementasi.md`. Yang perlu disadari khusus modul ini: peran
// pengujinya kelak adalah **PNCKomite** dan **PNCKomiteTeknik**, dan isi layar ini
// memperlihatkan siapa yang berwenang menyetujui uang.
func Mount(r chi.Router, h *Handler) {
	r.Route("/master/ambang-komite", func(master chi.Router) {
		master.Get("/", h.ListThresholds)

		// Sub-sumber daya, bukan parameter kueri pada daftar: hasilnya berbentuk lain
		// dan dibaca pada saat yang lain. Menggabungkannya akan membuat satu respons
		// membawa dua hal yang tidak pernah dibutuhkan bersamaan.
		master.Get("/integritas", h.Integrity)
	})

	r.Route("/komite", func(k chi.Router) {
		k.Get("/penjenjangan", h.Tiering)
	})
}
