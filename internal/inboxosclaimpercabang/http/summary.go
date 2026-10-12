package inboxosclaimpercabanghttp

import (
	"net/http"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
)

// Summary melayani panel ringkasan di atas grid.
//
// Ia rute TERSENDIRI, bukan bagian jawaban daftar, karena keduanya berubah pada irama yang
// berbeda: daftar berganti halaman, panel tidak. Menyatukannya berarti menghitung ulang
// seluruh ringkasan cabang setiap kali pengguna pindah halaman.
//
// Kegagalannya juga terpisah: panel yang gagal dimuat tidak boleh mengosongkan grid di
// bawahnya, dan sebaliknya.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	summarized, err := h.service.Summary(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toSummaryResponse(summarized, active.Alias))
}

// SummaryResponse adalah jawaban panel ringkasan.
type SummaryResponse struct {
	// AsOf adalah kapan angkanya dibaca, untuk baris "Posisi <tanggal>".
	AsOf string `json:"posisi"`

	TotalClaims int `json:"total_berkas"`

	// EstimationTotal dan ReserveOR dikirim sebagai TEKS desimal, bukan angka JSON.
	//
	// Angka JSON dibaca JavaScript sebagai float64, dan nilai uang tidak pernah float
	// (`I-12`).
	EstimationTotal string `json:"total_estimasi"`
	ReserveOR       string `json:"total_reserve_or"`

	// ReserveORAvailable membedakan "nol" dari "tidak terbaca".
	//
	// Keduanya WAJIB dibedakan di layar: Rp 0 adalah jawaban yang sah dan memang lazim —
	// `treaty_loss@asmd` hanya memuat 19 baris berawalan `PNC-` dari 10.152 — sedangkan
	// tidak terbaca berarti DB Link sedang bermasalah dan menuntut tindakan lain.
	ReserveORAvailable bool `json:"total_reserve_or_terbaca"`

	OverTwoYears int `json:"umur_di_atas_2_tahun"`

	Buckets  []AgeBucketDTO    `json:"sebaran_umur"`
	ByCOB    []SummaryGroupDTO `json:"per_cob"`
	BySource []SummaryGroupDTO `json:"per_sumber_bisnis"`

	Branch BranchDTO `json:"cabang"`
	Portal string    `json:"portal"`
}

// AgeBucketDTO adalah satu pita pada batang sebaran umur.
type AgeBucketDTO struct {
	Label  string `json:"label"`
	Claims int    `json:"berkas"`
	Value  string `json:"nilai"`
}

// SummaryGroupDTO adalah satu baris rincian per COB atau per sumber bisnis.
type SummaryGroupDTO struct {
	Name         string `json:"nama"`
	Claims       int    `json:"berkas"`
	Value        string `json:"nilai"`
	OverTwoYears int    `json:"umur_di_atas_2_tahun"`
}

// toSummaryResponse menyusun jawaban panel.
func toSummaryResponse(s usecase.Summarized, portalAlias string) SummaryResponse {
	buckets := make([]AgeBucketDTO, 0, len(s.Summary.Buckets))
	for _, b := range s.Summary.Buckets {
		buckets = append(buckets, AgeBucketDTO{
			Label:  b.Label,
			Claims: b.Claims,
			Value:  b.Value.String(),
		})
	}

	return SummaryResponse{
		AsOf:               isoDate(&s.Summary.AsOf),
		TotalClaims:        s.Summary.TotalClaims,
		EstimationTotal:    s.Summary.EstimationTotal.String(),
		ReserveOR:          s.Summary.ReserveOR.String(),
		ReserveORAvailable: s.Summary.ReserveORAvailable,
		OverTwoYears:       s.Summary.OverTwoYears,
		Buckets:            buckets,
		ByCOB:              toGroupDTOs(s.Summary.ByCOB),
		BySource:           toGroupDTOs(s.Summary.BySource),
		Branch:             BranchDTO{Code: s.Query.Branch.Code, Name: s.Query.Branch.Name},
		Portal:             portalAlias,
	}
}

// toGroupDTOs menyalin daftar rincian.
//
// Nama kosong diganti penanda DI SINI, bukan di domain: domain tidak memutuskan bagaimana
// sesuatu digambar. Sumber bisnis memang kerap kosong — 22 dari 469 klaim outstanding tidak
// punya `sobname` — dan baris tanpa nama tidak dapat dibedakan dari baris yang namanya belum
// termuat.
func toGroupDTOs(groups []inboxosclaimpercabang.SummaryGroup) []SummaryGroupDTO {
	result := make([]SummaryGroupDTO, 0, len(groups))
	for _, g := range groups {
		name := g.Name
		if name == "" {
			name = "(tanpa keterangan)"
		}
		result = append(result, SummaryGroupDTO{
			Name:         name,
			Claims:       g.Claims,
			Value:        g.Value.String(),
			OverTwoYears: g.OverTwoYears,
		})
	}
	return result
}

