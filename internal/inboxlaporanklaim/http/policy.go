package inboxlaporanklaimhttp

import (
	"net/http"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// PolicyNoticeDTO adalah satu pesan hasil pencarian polis.
type PolicyNoticeDTO struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`

	// Memblokir berarti tombol Simpan dan Register Klaim dimatikan.
	Memblokir bool `json:"memblokir"`
}

// PolicyResponse adalah jawaban GET /inbox/laporan-klaim/polis?nomor=….
type PolicyResponse struct {
	NomorPolis  string `json:"nomor_polis"`
	Ditemukan   bool   `json:"ditemukan"`
	Tertanggung string `json:"tertanggung"`
	KodeBisnis  string `json:"kode_bisnis"`
	NamaBisnis  string `json:"nama_bisnis"`
	NomorRujuk  string `json:"nomor_rujukan"`
	GroupPanel  string `json:"group_panel"`
	PolisLeader string `json:"polis_leader"`
	Syariah     bool   `json:"syariah"`

	Pesan     []PolicyNoticeDTO `json:"pesan"`
	Memblokir bool              `json:"memblokir"`
}

// Policy menangani GET /inbox/laporan-klaim/polis?nomor=… — pengisian otomatis form
// Input Receive Document saat Nomor Polis diisi.
//
// Polis yang tidak ditemukan dijawab 200 dengan `ditemukan: false`, bukan 404: alamatnya
// ada, yang kosong adalah hasilnya, dan layar tetap harus mengosongkan isian yang
// bergantung pada polis.
func (h *Handler) Policy(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	result, err := h.service.LookupPolicy(r.Context(), active.Alias, r.URL.Query().Get("nomor"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	notice := make([]PolicyNoticeDTO, 0, len(result.Notice))
	blocked := false
	for _, n := range result.Notice {
		notice = append(notice, PolicyNoticeDTO{Kode: n.Code, Pesan: n.Message, Memblokir: n.Blocking})
		blocked = blocked || n.Blocking
	}

	h.writeResponse(w, r, http.StatusOK, PolicyResponse{
		NomorPolis:  result.Number,
		Ditemukan:   result.Found,
		Tertanggung: result.Policy.InsuredName,
		KodeBisnis:  result.Policy.BusinessCode,
		NamaBisnis:  result.Policy.BusinessName,
		NomorRujuk:  result.Policy.ReferenceNumber,
		GroupPanel:  result.Policy.GroupPanel,
		PolisLeader: result.Policy.Leader,
		Syariah:     result.Policy.Syariah,
		Pesan:       notice,
		Memblokir:   blocked,
	})
}
