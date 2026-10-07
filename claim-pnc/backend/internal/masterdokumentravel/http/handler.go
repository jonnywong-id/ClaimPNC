package masterdokumentravelhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat satu isian pendek; 64 KiB sudah jauh lebih dari cukup.
// Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori, bukan
// setelah.
//
// Ini BUKAN validasi isian — Work Owner menetapkan layar ini tanpa validasi. Yang dijaga
// di sini adalah sumber daya server, bukan aturan bisnis, dan keduanya berbeda: yang satu
// menolak permintaan yang tidak wajar, yang lain menolak isian yang tidak sah.
const maxRequestBody = 64 << 10

// Get menangani GET /api/master/dokumen-travel/{id}.
//
// Menggantikan `SetMstDocTravelValue_act(docid)`, yang menjalankan Report Definition
// lalu menyalin baris yang cocok ke halaman `TempMstDocTravel` untuk diisikan ke form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	doc, err := h.Service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
		TravelDocument: toDTO(doc),
		Portal:         active.Alias,
	})
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	ok := httpjson.Decode(w, r, maxRequestBody, &request, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
	return request, ok
}
