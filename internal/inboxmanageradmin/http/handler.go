package inboxmanageradminhttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/inboxmanageradmin/usecase"
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
	// Ia tidak dipakai menyaring satu pun kueri di modul ini — layar ini pandangan penyelia
	// atas pekerjaan satu unit organisasi. Yang memakainya adalah jejak log, dan itulah
	// satu-satunya kontrol yang tersisa selama pemeriksaan peran belum ada
	// (`TKT-F3-004`).
	Login string

	// OrgUnit adalah unit organisasi pengguna — padanan `OperatorID.pyOrgUnit`.
	//
	// Satu nilai punya arti khusus: `Development` membuka ketiga tab sekaligus.
	OrgUnit string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Manager Admin.
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

// NewHandler membentuk handler modul Inbox Manager Admin.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-manager-admin/tab.
//
// # Kenapa ia butuh identitas, berbeda dari modul inbox lain
//
// Karena jawabannya BERBEDA menurut pemanggil: yang dikirim bukan ketiga tab melainkan tab
// yang boleh ia lihat. Modul lain dapat menjawab keterangan layar tanpa mengenali siapa yang
// bertanya; modul ini tidak.
//
// Ia tetap GET dan tidak mengubah apa pun.
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

	h.writeJSON(w, r, http.StatusOK, toMetadataResponse(meta, active.Alias))
}

// List menangani GET /api/inbox-manager-admin.
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
		inboxmanageradmin.Pagination{
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

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena Metadata, List, dan Export menuntut keduanya dengan urutan yang
// sama, dan urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal
// dijawab sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxmanageradmin.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxmanageradmin.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxmanageradmin.ErrCallerUnknown)
		return portal.Portal{}, inboxmanageradmin.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca isian penyaring dari parameter query.
//
// Hanya satu: tab mana yang diminta. Layar lama tidak punya satu pun kotak cari maupun
// dropdown — Report Definition-nya tidak menyaring menurut kata kunci sama sekali.
//
// Ia tetap dikumpulkan sebagai fungsi tersendiri supaya daftar dan ekspor membaca parameter
// yang SAMA PERSIS. Ekspor yang membaca tab dengan cara berbeda akan menghasilkan berkas
// yang isinya tidak dapat dicocokkan dengan apa pun di layar.
func readFilter(query map[string][]string) inboxmanageradmin.QueryInput {
	values := query["tab"]
	if len(values) == 0 {
		return inboxmanageradmin.QueryInput{}
	}
	return inboxmanageradmin.QueryInput{Tab: values[0]}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Login yang kosong membuat pemanggil dianggap tidak terbaca; unit organisasi yang kosong
// TIDAK. Pembedaan itu disengaja: tanpa login, pembukaan layar tidak dapat dicatat atas nama
// siapa pun — dan tanpa login pula lini bisnisnya tidak dapat dicari.
//
// # Lini bisnis SENGAJA tidak diambil dari sesi
//
// Sampai 2026-09-27 fungsi ini menyalin `caller.Position` dari sesi, yang bersumber dari
// jabatan kepegawaian HCQ. Nilai itu tidak pernah cocok dengan `NONMBU`/`PA`/`TRAVEL`,
// sehingga tidak seorang pun melihat satu tab pun.
//
// Sekarang lini bisnis dibaca usecase dari `M_LOGIN_PNC` milik portal yang aktif. Lapisan
// ini tidak boleh mengisinya: ia tidak tahu portal mana yang aktif pada saat identitas
// dibaca, dan menebaknya berarti menilai kewenangan dengan data entitas yang salah.
func (h *Handler) readCaller(r *http.Request) (inboxmanageradmin.Caller, bool) {
	if h.caller == nil {
		return inboxmanageradmin.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxmanageradmin.Caller{}, false
	}
	return inboxmanageradmin.Caller{
		Login:   caller.Login,
		OrgUnit: caller.OrgUnit,
	}, true
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
