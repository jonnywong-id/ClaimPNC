package inboxmanagerhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/usecase"
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
	// Setiap keputusan yang ditulis modul ini dicatat atas namanya, dan pada dua antrean ia
	// ikut tersimpan di kolom basis data. Tanpa login, tidak satu pun keputusan boleh
	// ditulis.
	Login string

	// OrgUnit adalah unit organisasi pengguna — padanan `OperatorID.pyOrgUnit`.
	//
	// Satu nilai punya arti khusus: `Development` membuka tab yang dibatasi lini bisnis.
	OrgUnit string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Manager.
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

// NewHandler membentuk handler modul Inbox Manager.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-manager/tab.
//
// Jawabannya BERBEDA menurut pemanggil: yang dikirim bukan ketiga belas tab melainkan tab
// yang boleh ia lihat, ditambah lini bisnis yang benar-benar terbaca untuknya.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	meta, err := h.service.Metadata(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(meta))
}

// Counters menangani GET /api/inbox-manager/ringkasan.
//
// Ia endpoint TERSENDIRI karena isinya tidak berubah saat tab berpindah — lihat catatan pada
// usecase.Counters.
func (h *Handler) Counters(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	counters, err := h.service.Counters(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toCountersResponse(counters))
}

// List menangani GET /api/inbox-manager.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	view, err := h.service.List(r.Context(), active.Alias, readQuery(r), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, toListResponse(view))
}

// Decide menangani POST /api/inbox-manager/keputusan.
//
// # Kenapa POST, dan kenapa satu alamat untuk sembilan antrean
//
// POST karena ia MENGUBAH data — satu-satunya rute yang demikian di seluruh modul inbox.
//
// Satu alamat karena kunci antrean tidak dapat membedakan antreannya sendiri: pada enam dari
// sembilan antrean kuncinya hanyalah sebuah ID, dan ID yang sama ada di tabel yang berbeda.
// Tab karena itu dikirim di BADAN permintaan, bukan ditebak dari kuncinya.
func (h *Handler) Decide(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body DecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, r, inboxmanager.NewValidationError([]inboxmanager.Violation{{
			Field:   inboxmanager.FieldKeys,
			Message: "Badan permintaan tidak dapat dibaca.",
		}}))
		return
	}

	result, err := h.service.Decide(r.Context(), active.Alias, usecase.DecideInput{
		Tab:     body.Tab,
		Verdict: inboxmanager.Verdict(strings.TrimSpace(body.Verdict)),
		Keys:    body.Keys,
		Reason:  body.Reason,
	}, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeJSON(w, r, http.StatusOK, DecisionResponse{
		Requested: result.Requested,
		Changed:   result.Changed,
		Stale:     result.Stale(),
		Message:   decisionMessage(result),
	})
}

// decisionMessage menyusun kalimat hasil yang menyatakan selisihnya APA ADANYA.
//
// # Kenapa selisihnya dikatakan, bukan disembunyikan
//
// Karena baris yang tidak berubah berarti orang lain sudah memutuskannya sementara daftar di
// layar masih yang lama. Melaporkan "3 baris diputuskan" untuk permintaan yang hanya mengubah
// satu adalah laporan yang salah tentang hal yang paling perlu dipercaya — dan di Pega
// keadaan itu bahkan tidak dapat diketahui, karena keputusan terakhir menimpa yang sebelumnya
// tanpa jejak.
func decisionMessage(result inboxmanager.DecisionResult) string {
	stale := result.Stale()
	if stale == 0 {
		return fmt.Sprintf("%d baris diputuskan.", result.Changed)
	}
	return fmt.Sprintf(
		"%d dari %d baris berubah. Sisanya sudah diputuskan orang lain sementara daftar "+
			"ini masih yang lama — muat ulang untuk melihat keadaan terkini.",
		result.Changed, result.Requested)
}

// prepare membaca portal aktif dan identitas pemanggil, atau menjawab galat.
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxmanager.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxmanager.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxmanager.ErrCallerUnknown)
		return portal.Portal{}, inboxmanager.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Login yang kosong membuat pemanggil dianggap tidak terbaca; unit organisasi yang kosong
// TIDAK.
//
// # Lini bisnis SENGAJA tidak diambil dari sesi
//
// Ia dibaca usecase dari `M_LOGIN_PNC` pada portal yang aktif. Mengambilnya dari sesi berarti
// mengambilnya dari jabatan kepegawaian HCQ — nilai yang tidak pernah berisi kode lini bisnis
// — dan itu cacat yang sudah pernah mengosongkan layar modul lain selama berhari-hari tanpa
// satu pun galat (koreksi 2026-09-27).
func (h *Handler) readCaller(r *http.Request) (inboxmanager.Caller, bool) {
	if h.caller == nil {
		return inboxmanager.Caller{}, false
	}

	caller, exists := h.caller(r.Context())
	if !exists {
		return inboxmanager.Caller{}, false
	}

	clean := inboxmanager.Caller{
		Login:   strings.TrimSpace(caller.Login),
		OrgUnit: strings.TrimSpace(caller.OrgUnit),
	}
	if clean.Login == "" {
		return inboxmanager.Caller{}, false
	}

	return clean, true
}

// readQuery membaca isian penyaring dari parameter query.
//
// Ia dipakai daftar DAN ekspor, supaya keduanya membaca parameter yang SAMA PERSIS. Ekspor
// yang membaca penyaring dengan cara berbeda akan menghasilkan berkas yang isinya tidak dapat
// dicocokkan dengan apa pun di layar.
func readQuery(r *http.Request) inboxmanager.QueryInput {
	query := r.URL.Query()

	input := inboxmanager.QueryInput{
		Tab: strings.TrimSpace(query.Get("tab")),
		Period: inboxmanager.PeriodInput{
			Mode:  inboxmanager.PeriodMode(strings.TrimSpace(query.Get("bentuk_periode"))),
			Month: strings.TrimSpace(query.Get("bulan")),
			From:  strings.TrimSpace(query.Get("dari")),
			Until: strings.TrimSpace(query.Get("sampai")),
		},
		Page: inboxmanager.Pagination{
			Page: atoiOrZero(query.Get("halaman")),
			Size: atoiOrZero(query.Get("ukuran")),
		},

		// Kedua penyaring grid ketiga tab Outstanding. Kosong berarti "All", sama seperti
		// pilihan bawaan di layar lama.
		Reinsurer:  strings.TrimSpace(query.Get("reinsurer")),
		CategoryOS: strings.TrimSpace(query.Get("kategori_os")),
	}

	return input
}

// atoiOrZero membaca angka dan mengembalikan nol bila tidak terbaca.
//
// Nol aman karena Pagination.Normalize membetulkannya menjadi nilai baku. Menolak permintaan
// karena `halaman=abc` akan membuat layar gagal tanpa alasan yang terbaca pengguna.
func atoiOrZero(value string) int {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return number
}
