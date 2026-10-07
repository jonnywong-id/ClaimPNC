package inboxmanageradminhttp

import (
	"net/http"
	"strings"

	"claim-pnc/internal/inboxmanageradmin"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

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

	meta, err := h.Service.Metadata(r.Context(), active.Alias, caller)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toMetadataResponse(meta, active.Alias))
}

// List menangani GET /api/inbox-manager-admin.
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
		inboxmanageradmin.Pagination{
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
		h.WriteError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxmanageradmin.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxmanageradmin.ErrCallerUnknown)
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
	if h.Caller == nil {
		return inboxmanageradmin.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
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
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
