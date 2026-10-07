package mastersurveyorshttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastersurveyors"
	"claim-pnc/internal/mastersurveyors/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxSaveBodyBytes membatasi ukuran badan permintaan simpan.
//
// Formulir surveyor punya tiga belas isian pendek; apa pun yang lebih besar dari ini bukan
// permintaan yang wajar, dan menolaknya lebih awal menjaga memori tidak dihabiskan badan
// permintaan yang dikarang.
const maxSaveBodyBytes = 16 << 10

// Service adalah bagian usecase yang dipakai handler ini.
//
// Dinyatakan sebagai antarmuka sempit di paket yang MEMAKAInya, bukan diimpor dari
// usecase, supaya handler dapat diuji tanpa membentuk seluruh layanan beserta
// penyimpanan dan kedua seam-nya.
type Service interface {
	List(ctx context.Context, portalAlias string, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error)
	Get(ctx context.Context, portalAlias, id string) (mastersurveyors.Surveyor, error)
	Submit(ctx context.Context, portalAlias string, in usecase.Submission, by usecase.Submitter) (mastersurveyors.Surveyor, error)
	Update(ctx context.Context, portalAlias, id string, in usecase.Submission, by usecase.Submitter) (mastersurveyors.Surveyor, error)
	Decide(ctx context.Context, portalAlias, id string, d usecase.Decision, by usecase.Committee) (mastersurveyors.Surveyor, error)
}

// List menangani GET /api/master/surveyor.
//
// Menggantikan Report Definition `BrowseVDSurveyors_RD` yang mengisi keempat grid layar
// `DetailSurveyorsInbox`. Keempatnya dilayani SATU endpoint yang dibedakan parameter
// kueri, bukan empat endpoint — yang berbeda hanyalah saringannya.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	filter, err := h.filterFrom(r)
	if err != nil {
		h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: err.Error(),
		})
		return
	}

	rows, total, err := h.Service.List(r.Context(), active.Alias, filter)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := make([]SurveyorDTO, 0, len(rows))
	for _, s := range rows {
		content = append(content, toDTO(s))
	}
	h.WriteResponse(w, r, http.StatusOK, ListResponse{
		Surveyor: content,
		Total:    total,
		Portal:   active.Alias,
	})
}

// Get menangani GET /api/master/surveyor/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	surveyor, err := h.Service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{Surveyor: toDTO(surveyor), Portal: active.Alias})
}

// Create menangani POST /api/master/surveyor.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, caller, body, ok := h.prepareSave(w, r)
	if !ok {
		return
	}

	surveyor, err := h.Service.Submit(r.Context(), active.Alias, submissionFrom(body), usecase.Submitter{
		Identity: caller.Identity,
		Name:     caller.Name,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusCreated, SingleResponse{Surveyor: toDTO(surveyor), Portal: active.Alias})
}

// Update menangani PUT /api/master/surveyor/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, caller, body, ok := h.prepareSave(w, r)
	if !ok {
		return
	}

	surveyor, err := h.Service.Update(r.Context(), active.Alias, chi.URLParam(r, "id"), submissionFrom(body), usecase.Submitter{
		Identity: caller.Identity,
		Name:     caller.Name,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{Surveyor: toDTO(surveyor), Portal: active.Alias})
}

// Decide menangani POST /api/master/surveyor/{id}/keputusan.
//
// Jalur terpisah, bukan bagian dari PUT, dan itu disengaja: keputusan komite adalah
// PERISTIWA yang punya invarian sendiri dan wajib tercatat — bukan pembaruan field biasa
// (`docs/Steering/10-API-STRATEGY.md` §2).
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	var body DecisionRequest
	if !h.readBody(w, r, &body) {
		return
	}
	caller, known := h.Caller(r.Context())
	if !known {
		h.WriteError(w, r, errors.New("mastersurveyors/http: konteks pemanggil tidak ada"))
		return
	}

	surveyor, err := h.Service.Decide(r.Context(), active.Alias, chi.URLParam(r, "id"),
		usecase.Decision{
			Status: mastersurveyors.ApprovalStatus(strings.TrimSpace(body.Status)),
			Note:   body.Note,
		},
		usecase.Committee{Identity: caller.Identity, Name: caller.Name},
	)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, SingleResponse{Surveyor: toDTO(surveyor), Portal: active.Alias})
}

// prepareSave menjalankan tiga langkah yang sama pada Create dan Update: memastikan
// portal aktif, membaca badan permintaan, dan membaca identitas pemanggil.
//
// Digabung karena ketiganya harus terjadi dalam urutan itu dan gagal dengan cara yang
// sama; menuliskannya dua kali membuka peluang keduanya lambat laun berbeda.
func (h *Handler) prepareSave(w http.ResponseWriter, r *http.Request) (portal.Portal, Caller, SaveRequest, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, Caller{}, SaveRequest{}, false
	}

	var body SaveRequest
	if !h.readBody(w, r, &body) {
		return portal.Portal{}, Caller{}, SaveRequest{}, false
	}

	caller, known := h.Caller(r.Context())
	if !known {
		h.WriteError(w, r, errors.New("mastersurveyors/http: konteks pemanggil tidak ada"))
		return portal.Portal{}, Caller{}, SaveRequest{}, false
	}
	return active, caller, body, true
}

// readBody membaca badan permintaan JSON dengan batas ukuran.
//
// DisallowUnknownFields dinyalakan: field yang tidak dikenal DITOLAK, bukan diabaikan
// diam-diam. Itu yang menangkap salah ketik nama field sebelum pengguna menyangka
// isiannya tersimpan — dan yang mencegah klien mengira dapat mengirim `status` untuk
// menyetujui surveyornya sendiri.
func (h *Handler) readBody(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxSaveBodyBytes)
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		message := "Badan permintaan bukan JSON yang dikenali."
		if errors.Is(err, io.EOF) {
			message = "Badan permintaan kosong."
		}
		h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: message,
		})
		return false
	}
	return true
}

// filterFrom menyusun saringan daftar dari parameter kueri.
//
// # Kenapa saringannya eksplisit per field, bukan bahasa filter umum
//
// Parameter yang dikenali disebutkan satu per satu dan sisanya diabaikan. Bahasa filter
// umum — yang menerima nama kolom dari klien — adalah persis celah yang pola `{ASIS:...}`
// warisan buka (`docs/Steering/10-API-STRATEGY.md` §4).
func (h *Handler) filterFrom(r *http.Request) (mastersurveyors.Filter, error) {
	q := r.URL.Query()

	f := mastersurveyors.Filter{
		Name:     strings.TrimSpace(q.Get("nama")),
		AppLogin: strings.TrimSpace(q.Get("login")),
		TypeCode: strings.TrimSpace(q.Get("kode_tipe")),
	}

	if raw := strings.TrimSpace(q.Get("status")); raw != "" {
		status := mastersurveyors.ApprovalStatus(raw)
		if !status.Known() {
			return mastersurveyors.Filter{}, errors.New("Nilai status tidak dikenal. Yang sah: 0, 1, atau 2.")
		}
		f.Status = status
	}

	// Antrean komite: identitasnya diambil dari SESI, tidak pernah dari parameter kueri.
	// Menerimanya dari kueri akan membuat siapa pun dapat melihat antrean komite lain.
	if strings.TrimSpace(q.Get("antrean_saya")) == "1" {
		caller, known := h.Caller(r.Context())
		if !known {
			return mastersurveyors.Filter{}, errors.New("Antrean komite hanya dapat dibaca setelah masuk.")
		}
		f.MyCommitteeOnly = true
		f.CommitteeIdentity = caller.Identity
	}

	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return mastersurveyors.Filter{}, errors.New("Nilai limit harus bilangan bulat tidak negatif.")
		}
		f.Limit = value
	}
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return mastersurveyors.Filter{}, errors.New("Nilai offset harus bilangan bulat tidak negatif.")
		}
		f.Offset = value
	}
	return f, nil
}

// submissionFrom menyalin badan permintaan menjadi isian usecase.
func submissionFrom(body SaveRequest) usecase.Submission {
	return usecase.Submission{
		TypeCode:     body.TypeCode,
		Name:         body.Name,
		Address:      body.Address,
		PostalCode:   body.PostalCode,
		State:        body.State,
		Phone:        body.Phone,
		Fax:          body.Fax,
		Email:        body.Email,
		OtherContact: body.OtherContact,
		BranchCode:   body.BranchCode,
		BranchName:   body.BranchName,
		AppLogin:     body.AppLogin,
		DocumentID:   body.DocumentID,
	}
}
