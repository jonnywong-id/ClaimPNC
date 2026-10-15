package inboxbandinghargasalvagehttp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxbandinghargasalvage"
	"claim-pnc/internal/inboxbandinghargasalvage/usecase"
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
	// Itulah yang dicocokkan ke kolom `NAMAKOMITE` pada `T_CLAIM_CHEKER_SALVAGE`. Memakai
	// NIK di sini akan membuat kedua tab tampak kosong bagi setiap pengguna.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Banding Harga Salvage.
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

	// FallbackErrorWriter menangani galat yang bukan milik modul ini — galat sesi dan galat
	// portal.
	FallbackErrorWriter ErrorWriter
}

// NewHandler membentuk handler modul Inbox Banding Harga Salvage.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-banding-harga-salvage/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab dan kolomnya adalah bentuk layar, bukan data
// entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(h.service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-banding-harga-salvage.
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
		inboxbandinghargasalvage.QueryInput{
			Tab:     query.Get("tab"),
			Keyword: query.Get("cari"),
		},
		inboxbandinghargasalvage.Pagination{
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

// Summary menangani GET /api/inbox-banding-harga-salvage/ringkas.
//
// Rute TERSENDIRI, bukan bagian jawaban daftar, karena isinya tidak berubah saat pengguna
// berpindah tab — dan menggabungkannya akan menjalankan kedua hitungannya setiap kali tab
// dibuka.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	summary, err := h.service.Summary(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toSummaryResponse(summary, inboxbandinghargasalvage.ReviewerFor(caller), active.Alias))
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxbandinghargasalvage.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxbandinghargasalvage.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxbandinghargasalvage.ErrCallerUnknown)
		return portal.Portal{}, inboxbandinghargasalvage.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil lewat jembatan yang disuntikkan cmd.
func (h *Handler) readCaller(r *http.Request) (inboxbandinghargasalvage.Caller, bool) {
	if h.caller == nil {
		return inboxbandinghargasalvage.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || caller.Login == "" {
		return inboxbandinghargasalvage.Caller{}, false
	}
	return inboxbandinghargasalvage.Caller{Login: caller.Login}, true
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

// Decisions menangani GET /api/inbox-banding-harga-salvage/riwayat/{noKlaim}.
//
// Panel rincian pada grid "History Cheker": seluruh keputusan banding harga di bawah satu
// klaim. Ia GET dan tidak mengubah apa pun.
//
// Klaim yang tidak punya keputusan dijawab `200` dengan daftar kosong, BUKAN `404`. Keduanya
// berbeda artinya, dan hanya yang pertama yang benar: panel ini dibuka dari baris yang sudah
// tergambar di grid, sehingga klaimnya pasti ada — yang mungkin kosong adalah keputusannya.
func (h *Handler) Decisions(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	decided, err := h.service.Decisions(
		r.Context(), active.Alias, caller, chi.URLParam(r, "noKlaim"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDecisionsResponse(decided, active.Alias))
}

// Decide menangani POST /api/inbox-banding-harga-salvage/keputusan.
//
// Kedua tombol — Approve dan Reject — menempuh rute yang SAMA, dibedakan isian `setujui` di
// dalam badan permintaan. Alasannya bukan kerapian: di Pega pun keduanya memanggil activity
// yang sama, dan yang membedakannya hanya satu parameter. Memisahkannya menjadi dua rute
// berarti dua jalur yang harus dijaga tetap setara.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body DecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		// 400, bukan 422: badan yang tidak dapat diurai adalah cacat pemanggil, bukan
		// isian pengguna yang melanggar aturan (`10-API-STRATEGY.md` §5).
		h.writeJSON(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return
	}

	result, err := h.service.Decide(
		r.Context(), active.Alias, caller,
		inboxbandinghargasalvage.DecisionInput{
			DetailObject: body.DetailObject,
			SalvageID:    body.SalvageID,
			RequestPrice: body.RequestPrice,
			Note:         body.Note,
			Approve:      body.Approve,
		},
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK,
		toDecisionResultResponse(result, body.Approve, active.Alias))
}

// Documents menangani GET /api/inbox-banding-harga-salvage/dokumen.
//
// Ia isi dialog "Lihat File": dokumen banding satu barang, terbaru di atas.
//
// Kedua id dibaca dari parameter kueri, bukan dari ruas jalur, karena `IDDETAILSALVAGE`
// memuat garis miring di dalamnya — sebagai ruas jalur ia akan terpecah menjadi dua segmen.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	documents, err := h.service.Documents(
		r.Context(), active.Alias, caller,
		r.URL.Query().Get("detail_object"),
		r.URL.Query().Get("id_salvage"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toDocumentsResponse(documents, active.Alias))
}

// DocumentContent menangani GET /api/inbox-banding-harga-salvage/dokumen/{dokumen}.
//
// Ia menyerahkan ISI berkas, bukan JSON — karena itu ia tidak memakai h.writeJSON, dan
// header-nya disusun di sini.
func (h *Handler) DocumentContent(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	document, err := h.service.DocumentContent(
		r.Context(), active.Alias, caller,
		r.URL.Query().Get("detail_object"),
		r.URL.Query().Get("id_salvage"),
		chi.URLParam(r, "dokumen"),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", contentTypeOf(document))
	w.Header().Set("Content-Disposition", attachmentHeader(document.Name))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	// Galat penulisan sengaja diabaikan: header sudah terkirim, sehingga tidak ada lagi
	// jawaban galat yang dapat disusun. Penyebab lazimnya pengguna membatalkan unduhan.
	_, _ = w.Write(document.Content)
}

// contentTypeOf menyerahkan tipe isi yang aman dikirim.
//
// Tipe yang kosong diganti `application/octet-stream`, yakni "berkas yang diunduh". Ia
// SENGAJA tidak ditebak dari nama berkasnya: menebaknya berarti sebuah berkas yang
// sebenarnya HTML dapat dikirim sebagai `text/html` dan dijalankan peramban di asal yang
// sama dengan aplikasi ini.
func contentTypeOf(document inboxbandinghargasalvage.DocumentContent) string {
	if strings.TrimSpace(document.MIMEType) == "" {
		return "application/octet-stream"
	}
	return document.MIMEType
}

// attachmentHeader menyusun header Content-Disposition.
//
// Nama berkas dibersihkan dari tanda kutip dan pemisah jalur. Keduanya bukan kerapian: tanda
// kutip memutus header itu sendiri, dan pemisah jalur dapat mengarahkan penyimpanan ke luar
// folder unduhan pada sebagian peramban lama.
func attachmentHeader(name string) string {
	bersih := strings.Map(func(r rune) rune {
		switch r {
		case '"', '\\', '/', '\r', '\n':
			return -1
		}
		return r
	}, strings.TrimSpace(name))

	if bersih == "" {
		bersih = "dokumen-banding-salvage"
	}
	return `attachment; filename="` + bersih + `"`
}
