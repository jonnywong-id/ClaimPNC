package reportklaimhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/portal"
	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// CallerLookup membaca identitas pemanggil dari konteks permintaan.
//
// Ia disuntikkan dari cmd, bukan diimpor dari modul auth: modul tidak saling mengimpor
// lapisan transport-nya — itulah yang membuat modul dapat dipindahkan tanpa menariknya
// serta.
type CallerLookup func(ctx context.Context) (reportklaim.Caller, bool)

// Handler melayani permintaan Report Klaim.
type Handler struct {
	service       *usecase.Service
	caller        CallerLookup
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerLookup
	Logger  *slog.Logger

	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Report Klaim.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("reportklaim/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("reportklaim/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("reportklaim/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// Catalog menangani GET /report-klaim — isi layar sebelum satu tombol pun ditekan.
func (h *Handler) Catalog(w http.ResponseWriter, r *http.Request) {
	if _, _, ready := h.prepare(w, r); !ready {
		return
	}
	h.writeResponse(w, r, http.StatusOK, toCatalogDTO(h.service.Catalog()))
}

// BusinessOptions menangani GET /report-klaim/pilihan-bisnis.
func (h *Handler) BusinessOptions(w http.ResponseWriter, r *http.Request) {
	active, _, ready := h.prepare(w, r)
	if !ready {
		return
	}

	option, err := h.service.BusinessOptions(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	body := BusinessOptionResponse{Bisnis: make([]BusinessOptionDTO, 0, len(option))}
	for _, o := range option {
		body.Bisnis = append(body.Bisnis, BusinessOptionDTO{Kode: o.Code, Nama: o.Name})
	}
	h.writeResponse(w, r, http.StatusOK, body)
}

// prepare memeriksa portal aktif dan identitas pemanggil.
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (portal.Portal, reportklaim.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, reportklaim.Caller{}, false
	}

	caller, known := h.caller(r.Context())
	if !known {
		h.writeModuleError(w, r, reportklaim.ErrCallerUnknown)
		return portal.Portal{}, reportklaim.Caller{}, false
	}
	return active, caller, true
}

// readFilter membaca penyaring dari parameter kueri.
//
// # Nilai yang tidak dikenal DITOLAK, bukan diabaikan
//
// Pilihan lini bisnis menentukan ISI berkas — bukan sekadar tampilan. Menerima nilai
// asing apa adanya berarti kendali yang di layar diberikan dropdown hilang begitu
// permintaan datang dari luar layar, dan API memang dapat ditembak langsung.
func readFilter(r *http.Request, entity string) (reportklaim.Filter, error) {
	value := r.URL.Query()

	line, known := reportklaim.ParseBusinessLine(value.Get("lini"))
	if !known {
		return reportklaim.Filter{}, reportklaim.ErrUnknownBusinessLine
	}

	from, err := readDate(value.Get("dari"))
	if err != nil {
		return reportklaim.Filter{}, err
	}
	to, err := readDate(value.Get("sampai"))
	if err != nil {
		return reportklaim.Filter{}, err
	}

	return reportklaim.Filter{
		From:             from,
		To:               to,
		BusinessLine:     line,
		ComplianceStatus: strings.TrimSpace(value.Get("status_compliance")),
		BusinessCode:     strings.TrimSpace(value.Get("bisnis")),
		Detail:           value.Get("rincian") == "1",
		Entity:           entity,
	}, nil
}

// tanggalISO adalah bentuk tanggal yang dikirim layar.
//
// Dipilih ISO, bukan `dd/mm/yyyy` yang dipakai di dalam berkas: bentuk ISO tidak dapat
// dibaca dua arti (`01/02/2026` adalah 1 Februari bagi sebagian orang dan 2 Januari bagi
// sebagian lain), dan itulah yang dihasilkan `<input type="date">` di peramban.
const tanggalISO = "2006-01-02"

// readDate membaca satu isian tanggal sebagai TANGGAL WIB.
//
// Kosong menghasilkan nilai nol, bukan galat — laporan yang tidak memakai rentang tanggal
// memang tidak mengirimnya, dan yang MEMAKAINYA ditolak di lapisan domain dengan pesan
// yang menyebut isiannya.
func readDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation(tanggalISO, raw, clock.ZoneWIB)
	if err != nil {
		return time.Time{}, &reportklaim.ValidationError{
			Violation: []reportklaim.Violation{{
				Field:   "tanggal",
				Message: "Tanggal harus berbentuk YYYY-MM-DD.",
			}},
		}
	}
	return t, nil
}

// reportCode membaca kode laporan dari jalur.
func reportCode(r *http.Request) reportklaim.Code {
	return reportklaim.Code(strings.TrimSpace(chi.URLParam(r, "kode")))
}
