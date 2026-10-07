package inboxpladlahttp

import (
	"log/slog"
	"net/http"
	"strings"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// Metadata menangani GET /api/inbox-pla-dla/daftar.
//
// Ia TIDAK menuntut pemanggilnya reasuradur: bentuk layar — daftar tab dan kolomnya —
// sama bagi siapa pun, dan menolaknya di sini akan membuat layar tidak dapat menggambar
// pesan penjelasnya sendiri.
func (h *Handler) Metadata(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	h.WriteJSON(w, r, http.StatusOK,
		toMetadataResponse(h.Service.Metadata(), active.Alias))
}

// List menangani GET /api/inbox-pla-dla.
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
		readFilter(r),
		inboxpladla.Pagination{
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

	counts, err := h.Service.Counts(r.Context(), active.Alias, caller, readFilter(r))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toCountsResponse(counts, active.Alias))
}

// XOL menangani GET /api/inbox-pla-dla/xol.
//
// Rute TERSENDIRI, bukan bagian jawaban daftar, karena gridnya tidak disaring tab maupun
// kata kunci — isinya sama berapa pun tab yang sedang dibuka. Menggabungkannya akan
// menjalankan gabungan dua tabel XOL setiap kali pengguna berpindah tab atau mengetik satu
// huruf di kotak pencarian.
func (h *Handler) XOL(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	rows, err := h.Service.XOL(r.Context(), active.Alias, caller)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toXOLResponse(rows, active.Alias))
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
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	action := strings.TrimSpace(r.URL.Query().Get("tindakan"))

	if h.Logger != nil {
		h.Logger.Info(
			"tindakan diminta pada tombol yang belum dibangun",
			slog.String("modul", "inbox-pla-dla"),
			slog.String("jalur", r.URL.Path),
			slog.String("tindakan", action),
		)
	}

	h.WriteError(w, r, h.Service.RejectAction(action))
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Urutannya penting: portal diperiksa LEBIH DULU (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxpladla.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxpladla.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxpladla.ErrCallerUnknown)
		return portal.Portal{}, inboxpladla.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil lewat jembatan yang disuntikkan cmd.
func (h *Handler) readCaller(r *http.Request) (inboxpladla.Caller, bool) {
	if h.Caller == nil {
		return inboxpladla.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
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
func positiveNumber(raw string) int { return httpquery.NonNegativeTrimmed(raw) }
