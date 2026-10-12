package inboxclaimtreatynonprophttp

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/inboxclaimtreatynonprop/usecase"
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
	// Itulah yang disetel `GetDataTreatyinNonProp_Act` langkah 1 ke `Inputdata.CARI10`
	// lalu dicocokkan ke `PXASSIGNEDOPERATORID` pada tabel penugasan Pega. Memakai NIK di
	// sini akan membuat tab Admin tampak kosong bagi setiap pengguna.
	Login string
}

// CallerReader membaca identitas pemanggil dari konteks permintaan.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan modul Inbox Claim Treaty Non Prop.
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

// NewHandler membentuk handler modul Inbox Claim Treaty Non Prop.
func NewHandler(o Options) *Handler {
	return &Handler{
		service:    o.Service,
		caller:     o.GetCaller,
		logger:     o.Logger,
		writeJSON:  o.WriteJSON,
		writeError: WriteError(o.Logger, o.WriteJSON, o.FallbackErrorWriter),
	}
}

// Metadata menangani GET /api/inbox-claim-treaty-non-prop/tab.
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

// List menangani GET /api/inbox-claim-treaty-non-prop.
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
		inboxclaimtreatynonprop.Pagination{
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
// Ia sengaja BUKAN 404. Tombol "Create Claim Treaty Non Prop" digambar di layar mengikuti
// perlakuan yang sama dengan modul Prop, dan tombol yang dijawab "halaman tidak ditemukan"
// terbaca sebagai kerusakan — sementara yang dibutuhkan pengguna adalah tahu ke mana ia
// harus pergi.
//
// Portal tetap diperiksa lebih dulu meski permintaannya pasti ditolak: jawaban yang
// menyebut portal aktif untuk permintaan yang tidak menyebut portal akan membuat layar
// mengira ia sudah berada di portal yang benar.
func (h *Handler) RejectWrite(w http.ResponseWriter, r *http.Request) {
	if _, exists := portalhttp.ActivePortalFrom(r.Context()); !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda
	// seberapa sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi
	// dasar memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	if h.logger != nil {
		h.logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "inbox-claim-treaty-non-prop"),
			slog.String("jalur", r.URL.Path),
		)
	}

	h.writeError(w, r, inboxclaimtreatynonprop.ErrWriteNotAvailable)
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List dan Export menuntut keduanya dengan urutan yang sama, dan
// urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxclaimtreatynonprop.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxclaimtreatynonprop.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.writeError(w, r, inboxclaimtreatynonprop.ErrCallerUnknown)
		return portal.Portal{}, inboxclaimtreatynonprop.Caller{}, false
	}

	return active, caller, true
}

// readFilter membaca ketiga isian penyaring dari parameter query.
//
// Ia dikumpulkan supaya daftar dan ekspor membaca parameter yang SAMA PERSIS. Ekspor yang
// membaca penyaring dengan cara berbeda akan menghasilkan berkas yang isinya tidak dapat
// dicocokkan dengan apa pun di layar.
func readFilter(query map[string][]string) inboxclaimtreatynonprop.QueryInput {
	get := func(name string) string {
		values := query[name]
		if len(values) == 0 {
			return ""
		}
		return values[0]
	}

	return inboxclaimtreatynonprop.QueryInput{
		Tab:     get("tab"),
		SeeAll:  truthy(get("lihat_semua")),
		TBAOnly: truthy(get("lihat_tba")),
	}
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inboxclaimtreatynonprop.Caller, bool) {
	if h.caller == nil {
		return inboxclaimtreatynonprop.Caller{}, false
	}
	caller, exists := h.caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxclaimtreatynonprop.Caller{}, false
	}
	return inboxclaimtreatynonprop.Caller{Login: caller.Login}, true
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

// truthy membaca penanda boolean dari parameter query.
//
// Hanya bentuk yang jelas berarti "ya" yang diterima. Nilai yang tidak dikenali berarti
// TIDAK, bukan galat: `lihat_semua` melepas penyaring kepemilikan, dan nilai yang salah
// ketik harus jatuh ke pilihan yang lebih sempit — bukan yang lebih luas.
func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "ya":
		return true
	default:
		return false
	}
}
