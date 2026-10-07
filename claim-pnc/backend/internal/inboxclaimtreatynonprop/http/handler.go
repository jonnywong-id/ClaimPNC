package inboxclaimtreatynonprophttp

import (
	"log/slog"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxclaimtreatynonprop"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-claim-treaty-non-prop/tab.
//
// Ia GET dan tidak mengubah apa pun: daftar tab, kolom, dan selisih terencana adalah bentuk
// layar, bukan data entitas.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-claim-treaty-non-prop.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
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
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
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
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	// Dicatat, bukan hanya ditolak. Selama masa paralel, inilah satu-satunya tanda
	// seberapa sering pengguna benar-benar membutuhkan aksi ini — dan itu yang menjadi
	// dasar memutuskan kapan kepemilikan tabelnya dipindahkan (`P-1`).
	if h.Logger != nil {
		h.Logger.Info(
			"aksi tulis diminta pada modul yang belum menulis",
			slog.String("modul", "inbox-claim-treaty-non-prop"),
			slog.String("jalur", r.URL.Path),
		)
	}

	h.WriteError(w, r, inboxclaimtreatynonprop.ErrWriteNotAvailable)
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
		h.WriteError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxclaimtreatynonprop.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxclaimtreatynonprop.ErrCallerUnknown)
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
	if h.Caller == nil {
		return inboxclaimtreatynonprop.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
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
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }

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
