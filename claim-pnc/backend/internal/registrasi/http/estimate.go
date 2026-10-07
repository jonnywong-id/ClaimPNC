package registrasihttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// EstimationDTO adalah satu baris estimasi.
//
// kurs_e4, nilai_idr_sen, dan sudah_cfs hanya dikirim server; yang dikirim layar diabaikan.
// sudah_cfs berarti estimasi sudah dibuatkan Claim Face Sheet dan terkunci.
type EstimationDTO struct {
	Type        string `json:"tipe"`
	Currency    string `json:"mata_uang"`
	Date        string `json:"tanggal"`
	ValueCents  int64  `json:"nilai_sen"`
	RateE4      int64  `json:"kurs_e4"`
	RupiahCents int64  `json:"nilai_idr_sen"`
	FaceSheet   bool   `json:"sudah_cfs"`
}

// ObjectItemDTO adalah satu item objek beserta estimasinya.
type ObjectItemDTO struct {
	Name        string          `json:"nama"`
	Description string          `json:"deskripsi"`
	Group       string          `json:"kelompok"`
	Estimation  []EstimationDTO `json:"estimasi"`
}

// EstimateCoverageDTO adalah item-item satu coverage pada permintaan estimasi.
type EstimateCoverageDTO struct {
	Item []ObjectItemDTO `json:"item"`
}

// EstimateObjectDTO adalah coverage-coverage satu objek pada permintaan estimasi.
type EstimateObjectDTO struct {
	Coverage []EstimateCoverageDTO `json:"coverage"`
}

// EstimateRequest adalah badan POST /api/registrasi/estimasi dan …/estimasi/simpan.
type EstimateRequest struct {
	TaskID string              `json:"tugas_id"`
	Return bool                `json:"kembali"`
	Object []EstimateObjectDTO `json:"objek"`
}

// CurrencyDTO adalah satu pilihan Mata Uang.
type CurrencyDTO struct {
	ID   string `json:"id"`
	Name string `json:"nama"`
}

// CurrenciesResponse adalah jawaban GET /api/registrasi/mata-uang.
type CurrenciesResponse struct {
	Option []CurrencyDTO `json:"pilihan"`
}

func itemDTO(items []registrasi.ObjectItem) []ObjectItemDTO {
	result := make([]ObjectItemDTO, 0, len(items))
	for _, it := range items {
		dto := ObjectItemDTO{Name: it.Name, Description: it.Description, Group: it.Group, Estimation: make([]EstimationDTO, 0, len(it.Estimation))}
		for _, e := range it.Estimation {
			dto.Estimation = append(dto.Estimation, EstimationDTO{
				Type: e.Type, Currency: e.Currency, Date: formatDate(e.Date),
				ValueCents: int64(e.Value), RateE4: int64(e.Rate), RupiahCents: int64(e.Converted),
				FaceSheet: e.FaceSheet,
			})
		}
		result = append(result, dto)
	}
	return result
}

func estimateCommand(b EstimateRequest) (usecase.EstimateCommand, error) {
	command := usecase.EstimateCommand{TaskID: strings.TrimSpace(b.TaskID), Return: b.Return}
	for _, o := range b.Object {
		var coverages [][]usecase.ObjectItemInput
		for _, c := range o.Coverage {
			var items []usecase.ObjectItemInput
			for _, it := range c.Item {
				item := usecase.ObjectItemInput{Name: it.Name, Description: it.Description, Group: it.Group}
				for _, e := range it.Estimation {
					var date time.Time
					if strings.TrimSpace(e.Date) != "" {
						d, err := parseDate(e.Date, "Tanggal Estimasi")
						if err != nil {
							return usecase.EstimateCommand{}, err
						}
						date = d
					}
					if e.Type != "" && e.Type != registrasi.EstimateClaim && e.Type != registrasi.EstimateAdjuster {
						return usecase.EstimateCommand{}, fmt.Errorf("Tipe Estimasi %q tidak dikenal", e.Type)
					}
					item.Estimation = append(item.Estimation, usecase.EstimationInput{
						Type: e.Type, Currency: e.Currency, Date: date, Value: registrasi.Money(e.ValueCents),
					})
				}
				items = append(items, item)
			}
			coverages = append(coverages, items)
		}
		command.Item = append(command.Item, coverages)
	}
	return command, nil
}

// SaveEstimate menangani POST /api/registrasi/estimasi/simpan — tombol Save.
func (h *Handler) SaveEstimate(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body EstimateRequest
	if !h.readBody(w, r, &body) {
		return
	}
	command, err := estimateCommand(body)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: err.Error()})
		return
	}
	claim, err := h.service.SaveEstimate(r.Context(), command, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, ClaimResponse{Claim: claimDTO(claim)})
}

// CompleteEstimate menangani POST /api/registrasi/estimasi — tombol Next, atau Back bila
// `kembali`.
func (h *Handler) CompleteEstimate(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.callerOf(w, r)
	if !ok {
		return
	}
	var body EstimateRequest
	if !h.readBody(w, r, &body) {
		return
	}
	command, err := estimateCommand(body)
	if err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: err.Error()})
		return
	}
	result, err := h.service.CompleteEstimate(r.Context(), command, caller)
	if err != nil {
		h.failure(w, r, err)
		return
	}
	response := ClaimResponse{Claim: claimDTO(result.Claim), DecisionTrace: result.DecisionTrace}
	if result.NextTask != nil {
		t := taskDTO(*result.NextTask, h.service.Flow())
		response.Task = &t
	}
	h.writeResponse(w, r, http.StatusOK, response)
}

// ItemOptionDTO adalah satu pilihan Objek item estimasi.
type ItemOptionDTO struct {
	Name     string `json:"nama"`
	Group    string `json:"kelompok"`
	TSICents int64  `json:"tsi_sen"`
}

// ItemOptionsResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/pilihan-item.
type ItemOptionsResponse struct {
	Option []ItemOptionDTO `json:"pilihan"`
}

// ItemOptions menangani GET /api/registrasi/klaim/{klaimID}/pilihan-item?objek=….
func (h *Handler) ItemOptions(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	option, err := h.service.ItemOptions(r.Context(), claimID, r.URL.Query().Get("objek"))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := make([]ItemOptionDTO, 0, len(option))
	for _, o := range option {
		body = append(body, ItemOptionDTO{Name: o.Name, Group: o.Group, TSICents: int64(o.TSI)})
	}
	h.writeResponse(w, r, http.StatusOK, ItemOptionsResponse{Option: body})
}

// CoverageOptionsResponse adalah jawaban GET /api/registrasi/klaim/{klaimID}/pilihan-coverage.
type CoverageOptionsResponse struct {
	Option []CoverageDTO `json:"pilihan"`
}

// CoverageOptions menangani GET /api/registrasi/klaim/{klaimID}/pilihan-coverage?objek=… —
// isi dropdown "Tambah coverage": coverage polis milik objek itu beserta TSI dan
// spreading-nya, sehingga memilih satu coverage mengisi seluruh barisnya.
func (h *Handler) CoverageOptions(w http.ResponseWriter, r *http.Request, claimID string) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	option, err := h.service.CoverageOptions(r.Context(), claimID, r.URL.Query().Get("objek"))
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := make([]CoverageDTO, 0, len(option))
	for _, c := range option {
		cov := CoverageDTO{
			ID:          c.ID,
			Name:        c.Name,
			CauseOfLoss: c.CauseOfLoss,
			TSICents:    int64(c.TSI),
			Spreading:   make([]SpreadingDTO, 0, len(c.Spreading)),
		}
		for _, s := range c.Spreading {
			cov.Spreading = append(cov.Spreading, SpreadingDTO{
				TreatyKind: s.TreatyKind, Name: s.Name, Share: Percent(s.Share),
				Removed: s.Removed, FacOfferItem: s.FacOfferItem,
			})
		}
		body = append(body, cov)
	}
	h.writeResponse(w, r, http.StatusOK, CoverageOptionsResponse{Option: body})
}

// Currencies menangani GET /api/registrasi/mata-uang.
func (h *Handler) Currencies(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.callerOf(w, r); !ok {
		return
	}
	option, err := h.service.Currencies(r.Context())
	if err != nil {
		h.failure(w, r, err)
		return
	}
	body := make([]CurrencyDTO, 0, len(option))
	for _, o := range option {
		body = append(body, CurrencyDTO{ID: o.ID, Name: o.Name})
	}
	h.writeResponse(w, r, http.StatusOK, CurrenciesResponse{Option: body})
}
