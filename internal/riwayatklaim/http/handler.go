package riwayatklaimhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/riwayatklaim"
	"claim-pnc/internal/riwayatklaim/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Itulah yang dicocokkan ke kolom LOGIN pada POOLDATA.MST_PROTEKSI_DATA_PNC. Memakai
	// NIK di sini akan membuat setiap pengguna tampak belum terdaftar.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul View History Claim.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// GetCaller adalah jembatan SATU ARAH dari modul auth. Ia disuntikkan cmd, bukan
	// diimpor dari modul auth — itulah yang membuat modul ini dapat dipindahkan tanpa
	// menariknya serta.
	GetCaller CallerReader

	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul View History Claim.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Open menangani POST /riwayat-klaim/buka.
//
// # Kenapa POST, bukan GET
//
// Karena ia MENGUBAH keadaan: satu jatah pencarian terpakai setiap kali layar dibuka,
// persis sistem lama. Menjadikannya GET berarti berjanji permintaannya aman diulang —
// janji yang tidak dipenuhi, dan yang akan dilanggar oleh hal-hal yang tidak terlihat
// seperti prefetch peramban atau percobaan ulang otomatis.
func (h *Handler) Open(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, riwayatklaim.ErrCallerUnknown)
		return
	}

	opened, err := h.service.Open(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, OpenResponse{
		SearchTypes: toSearchTypeListDTO(opened.SearchTypes),
		Access:      toAccessDTO(opened.Access),
		Portal:      active.Alias,
	})
}

// Search menangani GET /riwayat-klaim.
//
// Ia GET meski mencatat jejak audit: yang dicatat adalah LOG, bukan keadaan yang dilihat
// pengguna, dan jatah tidak berkurang karenanya. Mengulang permintaan yang sama
// menghasilkan hasil yang sama.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, riwayatklaim.ErrCallerUnknown)
		return
	}

	query := r.URL.Query()

	searchDate, readable := parseDate(query.Get("tanggal_pencarian"))
	if !readable {
		h.writeBadDate(w, r, riwayatklaim.FieldSearchDate)
		return
	}

	birthDate, readable := parseDate(query.Get("tanggal_lahir"))
	if !readable {
		h.writeBadDate(w, r, riwayatklaim.FieldBirthDate)
		return
	}

	found, err := h.service.Search(
		r.Context(),
		active.Alias,
		caller,
		riwayatklaim.CriteriaInput{
			Type:       query.Get("tipe"),
			Text:       query.Get("nilai"),
			SearchDate: searchDate,
			BirthDate:  birthDate,
		},
		riwayatklaim.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toSearchResponse(found, active.Alias))
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (riwayatklaim.Caller, bool) {
	if h.caller == nil {
		return riwayatklaim.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return riwayatklaim.Caller{}, false
	}
	return riwayatklaim.Caller{Login: caller.Login}, true
}

// writeBadDate menjawab tanggal yang tidak dapat dibaca.
//
// 422 dengan penunjuk isian, bukan 400: bentuk permintaannya benar — parameternya ada dan
// bertipe teks — hanya isinya yang tidak dapat dibaca sebagai tanggal. Layar dapat
// menandai isian yang salah alih-alih menampilkan galat umum.
func (h *Handler) writeBadDate(w http.ResponseWriter, r *http.Request, field string) {
	h.writeError(w, r, riwayatklaim.NewValidationError([]riwayatklaim.Violation{{
		Field:   field,
		Message: "Tanggal tidak dapat dibaca. Bentuknya YYYY-MM-DD.",
	}}))
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan riwayatklaim.Pagination.Normalize
// membetulkannya menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc`
// akan membuat layar gagal tanpa alasan yang terbaca pengguna — sementara menampilkan
// halaman pertama adalah jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
