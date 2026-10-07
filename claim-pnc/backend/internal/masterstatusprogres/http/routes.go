package masterstatusprogreshttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/httpjson"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua isian pendek; 64 KiB sudah jauh lebih dari cukup.
// Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori,
// bukan setelah.
const maxRequestBody = 64 << 10

// Position menangani GET /master/posisi-klaim.
//
// Rutenya TIDAK dipasangi PortalAktif: daftar posisi klaim adalah milik aplikasi, bukan
// isi basis data entitas mana pun (lihat masterstatusprogres/position.go).
// Menuntut portal di sini akan membuat dropdown gagal justru saat pengguna belum memilih
// portal — padahal tidak ada satu baris data entitas pun yang dibacanya.
func (h *Handler) Position(w http.ResponseWriter, r *http.Request) {
	list := h.service.Position()

	content := make([]PositionDTO, 0, len(list))
	for _, p := range list {
		content = append(content, PositionDTO{Code: p.Code, Name: p.Name})
	}
	h.writeResponse(w, r, http.StatusOK, PositionListResponse{Position: content})
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	ok := httpjson.Decode(w, r, maxRequestBody, &request, h.writeResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
	return request, ok
}

// Mount mendaftarkan rute modul master status progres.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// Middleware PortalAktif dipasang di sini, hanya pada rute yang menyentuh basis data
// entitas. Rute daftar posisi sengaja berada di luarnya (lihat Handler.Position).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Get("/master/posisi-klaim", h.Position)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/status-progres-1", h.List)
		perPortal.Post("/master/status-progres-1", h.Create)
		perPortal.Put("/master/status-progres-1/{id}", h.Update)
	})
}
