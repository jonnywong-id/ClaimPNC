package inboxrclhttp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxDecisionBody adalah batas badan permintaan keputusan — alasan dokter paling banyak 4.000
// byte, ditambah ruang untuk bentuk JSON-nya.
const maxDecisionBody = 64 << 10

// DecisionRequest adalah badan POST /api/inbox-rcl/klaim/{nomor}/keputusan.
type DecisionRequest struct {
	// Keputusan adalah nilai `StatusRCL` tombol, apa adanya: "SETUJU", "MSIG",
	// "TidakSetuju", atau "BackMSIG".
	Keputusan string `json:"keputusan"`

	// AlasanDokter adalah isian "Alasan Dokter" — hanya dibaca Tidak Setuju dan Back.
	AlasanDokter string `json:"alasan_dokter"`
}

// DecisionResponse adalah hasil keputusan.
type DecisionResponse struct {
	Portal          string `json:"portal"`
	NomorCase       string `json:"nomor_case"`
	StatusKlaim     string `json:"status_klaim"`
	TahapBerikutnya string `json:"tahap_berikutnya"`
}

// Decide menangani POST /api/inbox-rcl/klaim/{nomor}/keputusan.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxrcl.ErrCallerUnknown)
		return
	}

	var body DecisionRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxDecisionBody)
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeUnreadableBody,
			Message: "Isian yang dikirim tidak terbaca.",
		})
		return
	}

	number := chi.URLParam(r, "nomor")
	out, err := h.service.Decide(
		r.Context(), active.Alias, caller, number, body.Keputusan, body.AlasanDokter, h.now(),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, DecisionResponse{
		Portal:          active.Alias,
		NomorCase:       number,
		StatusKlaim:     out.StatusClaim,
		TahapBerikutnya: out.NextStageName,
	})
}
