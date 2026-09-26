package laporanhasilaihttp

import (
	"github.com/go-chi/chi/v5"

	portalhttp "claim-pnc/internal/portal/http"
)

// Mount mendaftarkan seluruh rute modul Laporan Hasil AI.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah `cmd/claimpnc`.
//
// # Dua rute, meski layar lamanya hanya punya satu activity
//
// Kedua tombol layar lama — "Cari Data" dan "Export To Excel" — memanggil
// `SearchDataLaporanAI(flagss=2)` yang sama; yang membedakan hanya bahwa tombol kedua
// membukanya di jendela baru sehingga jawabannya terunduh sebagai berkas.
//
// Di sini keduanya dipisah karena keluarannya memang dua bentuk yang berbeda: satu JSON
// yang dibaca layar, satu CSV yang diunduh. Menggabungkannya menjadi satu rute berarti
// bentuk jawabannya ditentukan sebuah parameter, dan permintaan yang salah parameter akan
// membuat peramban mengunduh JSON atau menggambar CSV.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/laporan-hasil-ai", h.Search)
		perPortal.Get("/laporan-hasil-ai/ekspor", h.Export)
	})
}
