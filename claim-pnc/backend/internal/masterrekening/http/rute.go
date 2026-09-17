package masterrekeninghttp

import (
	"github.com/go-chi/chi/v5"
)

// Pasang mendaftarkan rute modul master rekening ke router yang diberikan.
//
// Modul mendaftarkan rutenya sendiri; server di platform/httpserver tidak tahu apa pun
// tentang isi modul ini. Menambah modul berarti menambah satu baris di cmd/claimpnc.
//
// SELURUH RUTE DI SINI TERLINDUNGI. Pemanggil wajib memasangnya di dalam grup yang
// sudah memakai middleware sesi — master rekening memuat nama, NIK, nomor rekening, dan
// alamat surel pihak ketiga, dan tidak satu pun boleh terbaca tanpa sesi.
//
// CATATAN LINGKUP. Yang ditegakkan pemanggil baru "punya sesi yang sah". Pemeriksaan
// "berwenang atas menu master rekening" adalah TKT-F3-005, yang bergantung pada tabel
// peran TKT-F3-004 dan belum dikerjakan. Sampai itu ada, setiap pengguna yang dapat
// masuk dapat membuka layar ini — dicatat sebagai penghalang di
// docs/keputusan-implementasi.md, bukan diam-diam dianggap selesai.
func Pasang(r chi.Router, h *Handler) {
	r.Route("/master-rekening", func(rute chi.Router) {
		rute.Get("/", h.Daftar)
		rute.Post("/", h.Ajukan)

		// Daftar bank didaftarkan SEBELUM pola berparameter di bawahnya. Bila
		// urutannya dibalik, "bank" akan terbaca sebagai kode bank.
		rute.Get("/bank", h.DaftarBank)

		// Kunci alaminya pasangan kode bank + nomor rekening; keduanya ada di jalur
		// dengan urutan itu supaya /master-rekening/{kodeBank} kelak dapat berarti
		// "seluruh rekening di bank ini" tanpa mengubah jalur yang sudah ada.
		rute.Get("/{kodeBank}/{nomorRekening}", h.Ambil)
		rute.Put("/{kodeBank}/{nomorRekening}", h.Ubah)
		rute.Post("/{kodeBank}/{nomorRekening}/keputusan", h.Putuskan)
	})
}
