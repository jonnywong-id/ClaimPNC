package daftardetaildokumentravelhttp

import (
	"net/http"

	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat beberapa isian pendek ditambah satu senarai pembatasan
// plan. 256 KiB sudah jauh lebih dari cukup — satu aturan dokumen dengan seratus baris
// pembatasan pun tidak mendekati seperempatnya.
//
// Batasnya lebih besar daripada modul master lain justru KARENA senarai itu: modul yang
// hanya punya isian tunggal tidak punya cara menghasilkan badan besar, modul ini punya.
//
// Ini BUKAN validasi isian — Work Owner menetapkan layar ini tanpa validasi. Yang dijaga
// di sini adalah sumber daya server, bukan aturan bisnis, dan keduanya berbeda: yang satu
// menolak permintaan yang tidak wajar, yang lain menolak isian yang tidak sah.
const maxRequestBody = 256 << 10

// Documents menangani GET /api/master/dokumen-travel-pilihan.
//
// Menggantikan autocomplete `BrowseMstDocTravel_RD` pada isian ID Dokumen.
//
// Rutenya terpisah dari `/api/master/dokumen-travel` milik modul Master Dokumen Travel
// meski keduanya membaca tabel yang sama, dan itu disengaja: yang satu daftar yang dapat
// disunting, yang lain daftar pilihan. Menyatukannya akan membuat perubahan bentuk
// respons salah satunya merambat ke layar yang tidak ada hubungannya.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.Documents(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, DocumentListResponse{
		Document: toDocumentListDTO(list),
		Portal:   active.Alias,
	})
}

// Plans menangani GET /api/master/plan-travel.
//
// Menggantikan autocomplete `BrowsePlanTravelMaster_RD` dan `SearchCoverageTravel_RD`
// sekaligus — keduanya membaca POOLDATA.M_PLANTRAVEL yang sama, dan layar selalu
// membutuhkan keduanya bersamaan.
func (h *Handler) Plans(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	plans, err := h.Service.Plans(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	coverages, err := h.Service.Coverages(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, PlanListResponse{
		Plan:     toPlanListDTO(plans),
		Coverage: toCoverageOptionListDTO(coverages),
		Portal:   active.Alias,
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
