package masterstatushttp

import "github.com/go-chi/chi/v5"

// Mount mendaftarkan seluruh rute modul Master Status Klaim ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// Jalur yang didaftarkan relatif terhadap tempat pemanggil memasangnya — di
// cmd/claimpnc ia dipasang di bawah /api, DI DALAM kelompok yang sudah dijaga
// middleware sesi.
//
// # Kenapa "master/status-klaim" dan bukan "status-klaim"
//
// Awalan master menyatakan golongan data, bukan sekadar merapikan URL: sekurang-
// kurangnya 29 kelompok master akan menyusul (docs/Steering/05-DOMAIN-MODEL.md §5), dan
// seluruhnya berbagi satu menu serta satu kewenangan. Mengelompokkannya sejak sekarang
// membuat penegakan izin per menu (D-59) dapat dipasang pada satu tempat, bukan
// ditambahkan satu per satu ke 29 rute yang sudah telanjur tersebar.
//
// # Kewenangan
//
// Rutenya TERLINDUNGI sesi, tetapi BELUM diperiksa perannya. Penegakan "apakah peran
// pemanggil memiliki menu Master Data" adalah TKT-F3-005, yang bergantung pada tabel
// peran TKT-F3-004 — dan tabel itu dapat dibangun tetapi belum dapat diisi, karena
// penugasan operator ke peran tidak ada di basis data maupun di export
// (docs/Steering/11-SECURITY.md §3.1). Keadaan ini sama dengan seluruh rute lain yang
// sudah ada hari ini, dan dicatat terbuka di docs/keputusan-implementasi.md.
func Mount(r chi.Router, h *Handler) {
	r.Route("/master/status-klaim", func(master chi.Router) {
		master.Get("/", h.List)
		master.Post("/", h.Create)
		master.Get("/{kode}", h.Get)

		// PUT, bukan PATCH: seluruh isi yang boleh diubah — satu field, label — dikirim
		// setiap kali, sehingga permintaannya menggantikan dan idempoten. Mengirim
		// permintaan yang sama dua kali menghasilkan keadaan akhir yang sama.
		master.Put("/{kode}", h.Update)
	})
}
