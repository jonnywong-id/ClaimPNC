package inboxrclpuclhttp

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: modul tidak saling mengimpor lapisan
// transport-nya, dan jembatan di antara keduanya dipasang cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia tidak dipakai menyaring satu pun kueri di modul ini — antreannya bersama. Yang
	// memakainya adalah jejak log, dan itulah satu-satunya kontrol yang tersisa selama
	// pemeriksaan peran belum ada (`TKT-F3-004`).
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox RCL/PUCL.
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

// NewHandler membentuk handler modul Inbox RCL/PUCL.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-rcl-pucl/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-rcl-pucl.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.service.List(
		r.Context(),
		active.Alias,
		caller,
		readFilter(query),
		inboxrclpucl.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// RejectWrite menjawab aksi tulis yang belum tersedia.
//
// Ia sengaja BUKAN 404. Layar lama punya dua tindakan yang menulis — mencetak surat
// PUCL/RCL, yang mengisi `TANGGALCETAKDOKUMENPUCL_1` sehingga klaimnya BERPINDAH dari tab
// "Cetak Surat" ke tab "Kelengkapan Dokumen", dan mengirim Reminder PUCL. Tindakan yang
// dijawab "halaman tidak ditemukan" terbaca sebagai kerusakan, sementara yang dibutuhkan
// pengguna adalah tahu ke mana ia harus pergi.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang menyebut
// portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar mengira ia
// sudah berada di portal yang benar.
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
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", strings.TrimSpace(r.URL.Query().Get("tindakan"))),
		)
	}

	h.writeError(w, r, inboxrclpucl.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List, Export, dan Report menuntut keduanya dengan urutan yang sama,
// dan urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal
// dijawab sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxrclpucl.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxrclpucl.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxrclpucl.ErrCallerUnknown)
		return portal.Portal{}, inboxrclpucl.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring grid dari parameter query.
//
// Hanya satu: tab mana yang diminta. Kedua isian tanggal TIDAK dibaca di sini, dan itu
// bukan kelalaian — keduanya tidak menyaring grid sama sekali. Yang memakainya adalah
// laporan harian, lewat readReport di bawah.
//
// Ia tetap dikumpulkan sebagai fungsi tersendiri supaya daftar dan ekspor membaca parameter
// yang SAMA PERSIS. Ekspor yang membaca tab dengan cara berbeda akan menghasilkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func readFilter(query url.Values) inboxrclpucl.QueryInput {
	return inboxrclpucl.QueryInput{Tab: query.Get("tab")}
}

// readReport membaca permintaan laporan harian dari parameter query.
//
// Nama parameternya mengikuti nama isian di layar lama — "dari" dan "sampai" adalah padanan
// Indonesia dari "FROM RCL/PUCL" dan "TO RCL/PUCL" (`D-13`). Keduanya kontrak API, sehingga
// tetap berbahasa Indonesia (`D-80`).
func readReport(query url.Values) inboxrclpucl.ReportInput {
	return inboxrclpucl.ReportInput{
		Tab:  query.Get("tab"),
		From: query.Get("dari"),
		To:   query.Get("sampai"),
	}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxrclpucl.Caller, bool) {
	if h.caller == nil {
		return inboxrclpucl.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxrclpucl.Caller{}, false
	}
	return inboxrclpucl.Caller{Login: caller.Login}, true
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
