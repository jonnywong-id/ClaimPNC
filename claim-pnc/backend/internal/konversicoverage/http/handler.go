// Package konversicoveragehttp adalah lapisan transport modul Konversi Coverage
// (`MENU_ID 87`).
package konversicoveragehttp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	kc "claim-pnc/internal/konversicoverage"
	"claim-pnc/internal/konversicoverage/repo/sqlstore"
	"claim-pnc/internal/konversicoverage/usecase"
	"claim-pnc/internal/platform/db"
)

// Caller adalah pengguna yang menjalankan konversi — dicatat di log.
type Caller struct{ Login string }

// Options adalah ketergantungan Handler.
type Options struct {
	Config    kc.Config
	GetCaller func(ctx context.Context) (Caller, bool)
	Logger    *slog.Logger
}

// Handler melayani rute modul.
//
// Koneksi LIVE dan TEST dibuka SAAT PERTAMA DIJALANKAN, bukan saat aplikasi start:
// modul ini alat bantu data uji, dan basis data LIVE yang tidak terjangkau dari mesin
// pengembang tidak boleh menghentikan seluruh aplikasi.
type Handler struct {
	opts Options

	mu      sync.Mutex
	service *usecase.Service
	conns   []*sql.DB
}

// NewHandler menyusun Handler.
func NewHandler(opts Options) *Handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Handler{opts: opts}
}

// Close menutup koneksi yang sudah dibuka.
func (h *Handler) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, c := range h.conns {
		_ = c.Close()
	}
	h.conns, h.service = nil, nil
}

// Mount mendaftarkan rute. Rutenya WAJIB sudah berada di balik middleware Autentikasi.
//
// Tidak di balik pemeriksaan portal: kedua koneksinya ditentukan konfigurasi modul ini
// sendiri (KONVERSI_LIVE_* dan KONVERSI_TEST_*), bukan portal yang dipilih pengguna.
func Mount(r chi.Router, h *Handler) {
	r.Get("/konversi-coverage/keterangan", h.info)
	r.Post("/konversi-coverage/jalankan", h.run)
}

type connectionInfo struct {
	Alamat  string   `json:"alamat"`
	Lengkap bool     `json:"lengkap"`
	Kurang  []string `json:"kurang"`
}

type infoResponse struct {
	Live      connectionInfo `json:"live"`
	Test      connectionInfo `json:"test"`
	Polis     []string       `json:"polis"`
	MaksPolis int            `json:"maks_polis"`
	Galat     string         `json:"galat,omitempty"`
}

func describe(c kc.Connection) connectionInfo {
	missing := c.Missing()
	if missing == nil {
		missing = []string{}
	}
	return connectionInfo{Alamat: c.Label(), Lengkap: c.Complete(), Kurang: missing}
}

func (h *Handler) info(w http.ResponseWriter, r *http.Request) {
	cfg := h.opts.Config
	policies := cfg.Policies
	if policies == nil {
		policies = []string{}
	}
	response := infoResponse{
		Live: describe(cfg.Live), Test: describe(cfg.Test),
		Polis: policies, MaksPolis: usecase.MaxPolicies,
	}
	if err := cfg.Validate(); errors.Is(err, kc.ErrSameDatabase) {
		response.Galat = err.Error()
	}
	writeJSON(w, http.StatusOK, response)
}

type runRequest struct {
	Polis   string `json:"polis"`
	UjiCoba bool   `json:"uji_coba"`
}

func (h *Handler) run(w http.ResponseWriter, r *http.Request) {
	var request runRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "permintaan_tidak_sah", "Badan permintaan tidak dapat dibaca.")
		return
	}
	policies := kc.ParsePolicyList(request.Polis)

	service, err := h.ensure(r.Context())
	if err != nil {
		h.opts.Logger.Warn("konversi coverage: koneksi gagal", slog.Any("galat", err))
		writeError(w, http.StatusServiceUnavailable, "koneksi_gagal", err.Error())
		return
	}

	login := ""
	if caller, ok := h.opts.GetCaller(r.Context()); ok {
		login = caller.Login
	}
	h.opts.Logger.Info("konversi coverage dimulai",
		slog.String("login", login), slog.Int("jumlah_polis", len(policies)), slog.Bool("uji_coba", request.UjiCoba))

	report, err := service.Run(r.Context(), policies, !request.UjiCoba)
	switch {
	case errors.Is(err, usecase.ErrEmptyList), errors.Is(err, usecase.ErrTooMany):
		writeError(w, http.StatusUnprocessableEntity, "validasi_gagal", err.Error())
		return
	case errors.Is(err, usecase.ErrBusy):
		writeError(w, http.StatusConflict, "sedang_berjalan", err.Error())
		return
	case err != nil:
		h.opts.Logger.Error("konversi coverage terhenti", slog.Any("galat", err))
		writeError(w, http.StatusInternalServerError, "galat_internal", "Konversi terhenti: "+err.Error())
		return
	}
	h.opts.Logger.Info("konversi coverage selesai", slog.String("login", login),
		slog.Bool("uji_coba", report.UjiCoba), slog.Int("berhasil", report.Berhasil),
		slog.Int("gagal", report.Gagal), slog.Int("tidak_ditemukan", report.TidakAda))
	writeJSON(w, http.StatusOK, report)
}

// ensure membuka kedua koneksi sekali, lalu memakainya ulang.
func (h *Handler) ensure(ctx context.Context) (*usecase.Service, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.service != nil {
		return h.service, nil
	}
	cfg := h.opts.Config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	live, err := open(ctx, cfg.Live)
	if err != nil {
		return nil, errors.New("koneksi LIVE: " + err.Error())
	}
	test, err := open(ctx, cfg.Test)
	if err != nil {
		_ = live.Close()
		return nil, errors.New("koneksi TEST: " + err.Error())
	}
	h.conns = []*sql.DB{live, test}
	h.service = usecase.New(sqlstore.NewLive(live), sqlstore.NewTest(test), time.Now)
	return h.service, nil
}

func open(ctx context.Context, c kc.Connection) (*sql.DB, error) {
	return db.Open(ctx, db.Parameter{
		Alias: "KONVERSI_" + c.Role, Host: c.Host, Port: c.Port, Service: c.Service,
		User: c.User, Password: c.Password,
		// Kecil dengan sengaja: konversi berjalan satu per satu, dan kedua basis data
		// dipakai Pega.
		MaxConnections: 2, MaxIdle: 1, ConnectionLifetime: 10 * time.Minute,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"kode": code, "pesan": message})
}
