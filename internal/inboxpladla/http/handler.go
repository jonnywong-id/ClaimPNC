package inboxpladlahttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Caller adalah identitas pemanggil sebagaimana dilihat lapisan transport modul ini.
//
// Ia tipe milik modul ini, bukan tipe modul auth: jembatan di antara keduanya dipasang
// cmd/claimpnc.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Ia dicocokkan ke `POOLDATA.T_REINSURER.LOGIN`. Memakai NIK di sini akan membuat
	// layar kosong bagi SETIAP reasuradur — dan kosongnya tidak dapat dibedakan dari
	// "belum ada pekerjaan".
	Login string

	// Name adalah nama yang dibaca manusia.
	//
	// Ia tidak menyaring apa pun; satu-satunya pemakainya adalah kolom nama pembalas pada
	// balasan komunikasi. Boleh kosong.
	Name string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox PLA DLA.
type Handler struct {
	service    *usecase.Service
	caller     CallerReader
	logger     *slog.Logger
	writeJSON  JSONWriter
	writeError ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service   *usecase.Service
	GetCaller CallerReader
	Logger    *slog.Logger
	WriteJSON JSONWriter

	// FallbackErrorWriter menangani galat yang bukan milik modul ini.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox PLA DLA.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-pla-dla/daftar.
//
// Ia TIDAK menuntut pemanggilnya reasuradur: bentuk layar — daftar tab dan kolomnya —
// sama bagi siapa pun, dan menolaknya di sini akan membuat layar tidak dapat menggambar
// pesan penjelasnya sendiri.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-pla-dla.
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
		readFilter(r),
		inboxpladla.Pagination{
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

// Counts menangani GET /api/inbox-pla-dla/ringkas.
//
// Ia terpisah dari List karena isinya tidak berubah saat pengguna berpindah HALAMAN —
// sehingga layar dapat menyimpannya lebih lama. Ia MEMANG berubah saat pengguna berpindah
// tab atau mengubah pencarian, dan karena itu penyaringnya ikut dikirim.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	counts, err := h.service.Counts(r.Context(), active.Alias, caller, readFilter(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toCountsResponse(counts, active.Alias))
}

// ListCounts menangani GET /api/inbox-pla-dla/ringkas-daftar.
//
// Inilah tabel "Status / Jumlah" yang benar-benar digambar Pega, dan di layar lama ia
// satu-satunya navigasi: angkanya tautan yang mengganti isi grid di sebelahnya.
//
// Kata kunci pencarian ikut dibaca, kode tabnya TIDAK — tabelnya menyebut seluruh daftar,
// bukan daftar yang sedang terbuka.
func (h *Handler) ListCounts(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	counts, err := h.service.ListCounts(r.Context(), active.Alias, caller, readFilter(r))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListCountsResponse(counts, active.Alias))
}

// RejectWrite menjawab tombol yang belum tersedia.
//
// Ia sengaja BUKAN 404. Kedua tombolnya — "Download ALL PLA" dan "Download ALL DLA" —
// tergambar di layar rincian, dan tombol yang dijawab "halaman tidak ditemukan" terbaca
// sebagai kerusakan.
//
// Tindakannya DIBACA dari alamat, supaya alasan yang dijawab menyebut tombol yang
// benar-benar ditekan. Satu kalimat untuk keduanya akan membuat pengguna yang menekan
// "Download ALL DLA" membaca penjelasan tentang PLA.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	action := strings.TrimSpace(r.URL.Query().Get("tindakan"))

	if h.logger != nil {
		h.logger.Info(
			"tindakan diminta pada tombol yang belum dibangun",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", action),
		)
	}

	h.writeError(w, r, h.service.RejectAction(action))
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxpladla.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxpladla.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxpladla.ErrCallerUnknown)
		return portal.Portal{}, inboxpladla.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil lewat jembatan yang disuntikkan cmd.
func (h *Handler) readCaller(r *http.Request) (inboxpladla.Caller, bool) {
	if h.caller == nil {
		return inboxpladla.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists {
		return inboxpladla.Caller{}, false
	}
	return inboxpladla.Caller{Login: caller.Login, Name: caller.Name}, true
}

// readFilter membaca isian penyaring dari parameter query.
func readFilter(r *http.Request) inboxpladla.QueryInput {
	query := r.URL.Query()
	return inboxpladla.QueryInput{
		Tab:    strings.TrimSpace(query.Get("daftar")),
		Search: strings.TrimSpace(query.Get("cari")),
	}
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak terbaca menghasilkan 0, yang kemudian DIBETULKAN Pagination.Normalize
// menjadi nilai bawaan — bukan ditolak.
func positiveNumber(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 0 {
		return 0
	}
	return value
}
