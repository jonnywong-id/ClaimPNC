package registrasihttp

import (
	"net/http"
	"strconv"
)

// CommitteeStatusDTO adalah satu baris grid "Status Penerimaan Komite".
type CommitteeStatusDTO struct {
	Level    int    `json:"jenjang"`
	Name     string `json:"nama_komite"`
	Decision string `json:"status"` // 0 menunggu, 1 setuju, 2 tolak
	Date     string `json:"tanggal"`
	Note     string `json:"komentar"`
}

// CashierHistoryDTO adalah satu baris grid "Histori Transfer Kasir".
type CashierHistoryDTO struct {
	PIC    string `json:"pic_teknik"`
	Date   string `json:"tanggal"`
	Status string `json:"status_kasir"`
	Note   string `json:"komentar"`
}

// SettlementHistoryResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/adjustment/riwayat.
type SettlementHistoryResponse struct {
	Committee []CommitteeStatusDTO `json:"komite"`
	Cashier   []CashierHistoryDTO  `json:"kasir"`
}

// SettlementHistory menangani GET /api/registrasi/klaim/{klaimID}/adjustment/riwayat
// ?objek=&coverage=&adjustment= (ketiganya urutan berbasis 1).
func (h *Handler) SettlementHistory(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	q := r.URL.Query()
	object, _ := strconv.Atoi(q.Get("objek"))
	coverage, _ := strconv.Atoi(q.Get("coverage"))
	adjustment, _ := strconv.Atoi(q.Get("adjustment"))
	history, err := h.service.SettlementHistory(r.Context(), claimID, object, coverage, adjustment)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := SettlementHistoryResponse{
		Committee: make([]CommitteeStatusDTO, 0, len(history.Committee)),
		Cashier:   make([]CashierHistoryDTO, 0, len(history.Cashier)),
	}
	for _, m := range history.Committee {
		body.Committee = append(body.Committee, CommitteeStatusDTO{
			Level: m.Level, Name: m.Operator, Decision: m.Decision, Date: formatMoment(m.DecidedAt), Note: m.Note,
		})
	}
	for _, c := range history.Cashier {
		body.Cashier = append(body.Cashier, CashierHistoryDTO{
			PIC: c.PIC, Date: formatMoment(c.At), Status: c.Status, Note: c.Reason,
		})
	}
	h.writeResponse(w, r, http.StatusOK, body)
}
