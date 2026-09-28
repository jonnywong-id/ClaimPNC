package reportkpihttp

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/reportkpi"
	"claim-pnc/internal/reportkpi/usecase"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia tidak dipakai menyaring satu pun kueri di modul ini — laporannya pandangan
	// penyelia atas seluruh adjuster. Yang memakainya adalah jejak log, dan itulah
	// satu-satunya kontrol yang tersisa selama pemeriksaan peran belum ada
	// (`TKT-F3-004`, `D-59`).
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Report KPI PNC.
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

// NewHandler membentuk handler modul Report KPI PNC.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/report-kpi/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, grid, komponen, dan selisih terencana
// adalah bentuk layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata()))
}

// Summary menangani GET /api/report-kpi/adjuster/ringkasan.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	result, err := h.service.Summary(r.Context(), active.Alias, caller, readFilter(r.URL.Query()))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, SummaryResponse{
		Rows:   toSummaryRows(result.Rows),
		Filter: toFilterDTO(result.Query),
		Portal: active.Alias,
	})
}

// Detail menangani GET /api/report-kpi/adjuster.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()
	page := reportkpi.Pagination{
		Page: positiveNumber(query.Get("halaman")),
		Size: positiveNumber(query.Get("ukuran")),
	}

	result, err := h.service.Detail(r.Context(), active.Alias, caller, readFilter(query), page)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, DetailResponse{
		Rows:       toDetailRows(result.Page.Rows),
		Pagination: toPaginationDTO(page, result.Page.Total),
		Filter:     toFilterDTO(result.Query),
		Portal:     active.Alias,
	})
}

// Adjusters menangani GET /api/report-kpi/adjuster/pilihan.
//
// Ia isi dropdown "Pilih Adjuster", dan ia menuntut tipe report serta periode seperti
// permintaan lainnya — daftarnya menyempit mengikuti keduanya. Lihat usecase.Adjusters.
func (h *Handler) Adjusters(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	input := readFilter(r.URL.Query())

	names, err := h.service.Adjusters(r.Context(), active.Alias, caller, input)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	// Penyaring yang dikirim balik sengaja TIDAK memuat adjuster: daftar pilihan tidak
	// disaring oleh pilihan yang sedang aktif, dan menyatakannya sebaliknya akan membuat
	// layar mengira daftarnya sudah menyempit.
	input.Adjuster = ""
	query, _ := reportkpi.NewQuery(input, reportkpi.Caller{Login: caller.Login})

	h.writeJSON(w, r, http.StatusOK, AdjusterListResponse{
		Adjusters: names,
		Filter:    toFilterDTO(query),
		Portal:    active.Alias,
	})
}

// RejectWrite menangani POST /api/report-kpi/tindakan.
//
// Rutenya ADA supaya tombol yang di layar lama MENGHITUNG ULANG penilaian dijawab dengan
// alasan, bukan dengan "halaman tidak ditemukan". Keduanya terlihat sangat berbeda bagi
// pengguna, dan hanya yang pertama memberi tahu apa yang harus dilakukannya.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang
// menyebut portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar
// mengira ia sudah berada di portal yang benar.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda seberapa
	// sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi dasar
	// memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	if h.logger != nil {
		h.logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "report-kpi"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", strings.TrimSpace(r.URL.Query().Get("tindakan"))),
		)
	}

	h.writeError(w, r, reportkpi.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, reportkpi.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, reportkpi.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, reportkpi.ErrCallerUnknown)
		return portal.Portal{}, reportkpi.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring dari parameter query.
//
// Ia dikumpulkan sebagai satu fungsi supaya ringkasan, rincian, daftar pilihan, dan ekspor
// membaca parameter yang SAMA PERSIS. Ekspor yang membaca penyaring dengan cara berbeda
// akan menghasilkan berkas yang isinya tidak dapat dicocokkan dengan apa pun di layar.
//
// Nama parameternya mengikuti isian layar lama: "Pilih Tipe Report", "Pilih Adjuster",
// dan "Periode" Dari–Sampai (`D-13`). Ketiganya kontrak API, sehingga tetap berbahasa
// Indonesia (`D-80`).
func readFilter(query url.Values) reportkpi.QueryInput {
	return reportkpi.QueryInput{
		ReportType: query.Get("tipe_report"),
		Adjuster:   query.Get("adjuster"),
		From:       query.Get("dari"),
		To:         query.Get("sampai"),
	}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (reportkpi.Caller, bool) {
	if h.caller == nil {
		return reportkpi.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return reportkpi.Caller{}, false
	}
	return reportkpi.Caller{Login: caller.Login}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
