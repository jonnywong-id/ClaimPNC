package laporanhasilaihttp

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/laporanhasilai"
	"claim-pnc/internal/laporanhasilai/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Handler melayani permintaan Laporan Hasil AI.
//
// # Tidak ada Caller di sini, dan itu bukan kelalaian
//
// Modul yang menulis menerima identitas pemanggil untuk mengisi kolom pencatat siapa.
// Modul ini **tidak menulis apa pun** — layar lamanya baca-saja, dan kedua tombolnya
// memanggil activity yang sama tanpa satu pun langkah tulis.
type Handler struct {
	service    *usecase.Service
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// Service melayani seluruh perkara modul ini. Wajib.
	Service *usecase.Service

	Logger *slog.Logger

	// WriteJSON disuntikkan dari cmd, bukan diimpor dari modul auth. Modul tidak saling
	// mengimpor lapisan transport-nya — itulah yang membuat modul dapat dipindahkan tanpa
	// menariknya serta.
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan
	// galat portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Laporan Hasil AI.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("laporanhasilai/http: Service wajib diisi")
	}
	if o.WriteJSON == nil {
		return nil, errors.New("laporanhasilai/http: WriteJSON wajib diisi")
	}
	return &Handler{
		service:    o.Service,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}, nil
}

// Search menangani GET /api/laporan-hasil-ai.
//
// # Empat parameter kueri, dan dua di antaranya punya padanan di layar lama
//
//	dari      isian "Tgl Input Dari"     -> LaporanDataAIKlaim.AnalystTransferDate
//	sampai    isian "Tgl Input Sampai"   -> LaporanDataAIKlaim.DateOfLoss
//	halaman   nomor halaman              -> TIDAK ADA padanannya; lihat bawah
//	ukuran    baris per halaman          -> TIDAK ADA padanannya; lihat bawah
//
// Kedua yang terakhir tidak punya padanan karena layar lama tidak memaginasi di server
// sama sekali — ia memuat seluruh hasil ke klipboard lalu memaginasinya di peramban.
// Lihat doc `laporanhasilai.DefaultPageSize`.
//
// # Kedua tanggal WAJIB
//
// Bukan tambahan: layar lama pun tidak dapat berjalan tanpanya. Lihat doc
// `laporanhasilai.Filter.Validate`.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	filter := readFilter(query)
	page := readPagination(query)

	result, err := h.service.Search(r.Context(), active.Alias, filter, page)
	if err != nil {
		h.reportError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SearchResponse{
		Summary:    toSummaryDTO(result.Summary),
		Rows:       toRowListDTO(result.Page.Rows),
		Pagination: toPaginationDTO(page, result.Page.Total),
		Filter:     toFilterDTO(filter),
		Portal:     active.Alias,
	})
}

// readFilter membaca kedua isian tanggal dari parameter kueri.
//
// Tanggal yang tidak dapat dibaca diperlakukan sebagai KOSONG, bukan sebagai galat
// tersendiri. Akibatnya ia jatuh ke pesan "belum diisi" yang sudah ada, alih-alih
// memunculkan pesan kedua tentang bentuk tanggal yang tidak pernah diketik pengguna —
// isian di layar adalah pemilih tanggal, dan bentuk yang salah hanya mungkin datang dari
// alamat yang disunting tangan.
func readFilter(query url.Values) laporanhasilai.Filter {
	return laporanhasilai.Filter{
		From: parseDate(query.Get("dari")),
		To:   parseDate(query.Get("sampai")),
	}
}

// parseDate membaca YYYY-MM-DD; yang tidak terbaca menjadi waktu nol.
func parseDate(raw string) time.Time {
	text := strings.TrimSpace(raw)
	if text == "" {
		return time.Time{}
	}
	value, err := time.Parse(dateLayout, text)
	if err != nil {
		return time.Time{}
	}
	return value
}

// readPagination membaca jendela halaman dari parameter kueri.
//
// Nilai yang tidak terbaca menjadi nol, dan `Normalize` membetulkannya ke bawaan. Ia tidak
// ditolak: nomor halaman datang dari tautan paginasi, dan tautan yang cacat bukan
// kesalahan yang dapat diperbaiki pengguna dengan mengetik.
func readPagination(query url.Values) laporanhasilai.Pagination {
	return laporanhasilai.Pagination{
		Page: positiveNumber(query.Get("halaman")),
		Size: positiveNumber(query.Get("ukuran")),
	}
}

func positiveNumber(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

// activePortal membaca portal aktif; nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) activePortal(w http.ResponseWriter, r *http.Request) (portal.Portal, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.reportError(w, r, portal.ErrNotStated)
		return portal.Portal{}, false
	}
	return active, true
}

// reportError mencatat lalu meneruskan galat ke penulis galat modul ini.
func (h *Handler) reportError(w http.ResponseWriter, r *http.Request, err error) {
	logging.From(r.Context(), h.logger).Warn("permintaan Laporan Hasil AI gagal",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
	h.writeError(w, r, err)
}
