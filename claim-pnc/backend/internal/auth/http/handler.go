package authhttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/usecase"
)

// maxLoginBodyBytes membatasi ukuran badan permintaan masuk. Layar masuk hanya mengirim
// dua field pendek; apa pun yang lebih besar dari ini bukan permintaan yang wajar.
const maxLoginBodyBytes = 4 << 10

// Service adalah bagian usecase yang dipakai handler ini.
type Service interface {
	Login(ctx context.Context, k auth.Credential) (usecase.Result, error)
	Renew(ctx context.Context, token auth.Token) (auth.Session, error)
	Logout(ctx context.Context, token auth.Token) error
}

// Handler memuat handler masuk, keluar, identitas pemanggil, dan perpanjangan sesi.
type Handler struct {
	service    Service
	logger     *slog.Logger
	writeError ErrorWriter
}

// NewHandler membentuk handler modul auth.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger, writeError: WriteError(logger)}
}

// Login menangani POST /api/masuk.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		// Isi badan permintaan tidak ikut dicatat maupun dikembalikan: di situlah kata
		// sandi berada.
		WriteJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		}, h.logger)
		return
	}

	result, err := h.service.Login(r.Context(), auth.Credential{
		Username: request.Username,
		Password: request.Password,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, LoginResponse{
		Token:     string(result.Token),
		TipeToken: "Bearer",
		ExpiresAt: result.Session.ExpiresAt,
		User:      toUserDTO(result.User),
	}, h.logger)
}

// Me menangani GET /api/saya — dipakai peramban untuk memulihkan keadaan setelah
// muat ulang halaman tanpa harus menyimpan profil pengguna sendiri.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	baseCtx, existing := CallerFromContext(r.Context())
	if !existing {
		h.writeError(w, r, auth.ErrSessionNotFound)
		return
	}
	WriteJSON(w, r, http.StatusOK, MeResponse{
		User:      toUserDTO(baseCtx.User),
		ExpiresAt: baseCtx.Session.ExpiresAt,
	}, h.logger)
}

// Renew menangani POST /api/sesi/perpanjang.
func (h *Handler) Renew(w http.ResponseWriter, r *http.Request) {
	token, existing := TokenFromRequest(r)
	if !existing {
		h.writeError(w, r, auth.ErrSessionNotFound)
		return
	}
	extended, err := h.service.Renew(r.Context(), token)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusOK, RenewResponse{
		ExpiresAt: extended.ExpiresAt,
	}, h.logger)
}

// Logout menangani POST /api/keluar.
//
// Ia mencabut sesi di server, bukan sekadar meminta peramban melupakan tokennya —
// token lama harus ditolak sejak permintaan berikutnya.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token, existing := TokenFromRequest(r)
	if !existing {
		// Logout tanpa token bukan kegagalan yang perlu diperlihatkan: hasil akhirnya
		// sama-sama tidak ada sesi.
		WriteJSON(w, r, http.StatusNoContent, nil, h.logger)
		return
	}
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.writeError(w, r, err)
		return
	}
	WriteJSON(w, r, http.StatusNoContent, nil, h.logger)
}

func toUserDTO(p auth.User) UserDTO {
	return UserDTO{
		Identity: p.Identity,
		Name:     p.Name,
		Kind:     string(p.Kind),
		Login:    p.Login,
		Email:    p.Email,
		Company:  p.Company,
	}
}
