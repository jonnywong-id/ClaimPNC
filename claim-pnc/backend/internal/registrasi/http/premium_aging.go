package registrasihttp

import "net/http"

// PremiumAgingResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/aging.
type PremiumAgingResponse struct {
	// Amount adalah AgingAmount dengan dua desimal; null bila layanan tidak mengisinya.
	Amount *string `json:"aging_amount"`
	// Available false berarti layanan premi tidak dapat dihubungi.
	Available bool `json:"tersedia"`
}

// PremiumAging menangani GET /api/registrasi/klaim/{klaimID}/aging — isian Aging Amount
// layar Input Estimasi (`.PaymentData.AgingAmount`).
func (h *Handler) PremiumAging(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	aging, err := h.service.PremiumAgingOf(r.Context(), claimID)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := PremiumAgingResponse{Available: aging.Available}
	if aging.Amount != nil {
		text := aging.Amount.FloatString(2)
		body.Amount = &text
	}
	h.writeResponse(w, r, http.StatusOK, body)
}
