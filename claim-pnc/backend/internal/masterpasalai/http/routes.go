package masterpasalaihttp

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpasalai"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxKeywordLength membatasi panjang kata kunci yang diterima.
//
// Tanpa batas, satu permintaan dapat mengirim kata kunci sepanjang megabyte yang kemudian
// masuk ke dalam pola `LIKE` — basis data memindai seluruh tabel untuk sesuatu yang mustahil
// cocok.
//
// 200 dipilih karena kolom terpanjang yang dicari, `WP_KEJADIAN`, digambar `pxTextArea`
// selebar 283 px di layar lama; kata kunci yang lebih panjang dari satu kalimat penuh tidak
// pernah menjadi pencarian yang dimaksudkan.
//
// Kata kunci yang melebihi batas DITOLAK sebagai permintaan cacat, bukan dipotong diam-diam:
// memotongnya akan mengembalikan hasil yang tidak diminta siapa pun.
const maxKeywordLength = 200

// List menangani GET /master/pasal-ai.
//
// # Dua parameter kueri, dan keduanya punya padanan di layar lama
//
//	cari      isi kotak "Cari"   -> TempSearch.Country
//	halaman   nomor halaman      -> Pagination.CurrentIndex
//
// Keduanya opsional. Tanpa keduanya, jawabannya halaman pertama tanpa penyaring — persis
// pemuatan pertama layar lama, yang menjalankan `GetListPasalAI(Flags=1)` dengan kotak cari
// masih kosong.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	keyword := r.URL.Query().Get("cari")
	if len(keyword) > maxKeywordLength {
		h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Kata kunci pencarian terlalu panjang.",
		})
		return
	}

	page, err := h.Service.List(r.Context(), active.Alias, masterpasalai.Filter{
		Keyword: keyword,
		Page:    pageNumber(r.URL.Query().Get("halaman")),
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, ListResponse{
		Clause: toListDTO(page.Clause),
		Page:   toPageDTO(page),
		Portal: active.Alias,
	})
}

// pageNumber membaca nomor halaman dari kueri.
//
// Nilai yang tidak dapat dibaca sebagai angka menjadi halaman pertama, bukan galat: ia datang
// dari tautan paginasi, dan tautan yang cacat bukan kesalahan yang dapat diperbaiki pengguna
// dengan mengetik. Domain menormalkannya lagi lewat Filter.Clean.
func pageNumber(raw string) int {
	if raw == "" {
		return masterpasalai.FirstPage
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return masterpasalai.FirstPage
	}
	return value
}

// activePortal membaca portal aktif; nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) activePortal(w http.ResponseWriter, r *http.Request) (portal.Portal, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, false
	}
	return active, true
}

// writeModuleError meneruskan galat ke penulis bersama.
//
// Modul ini tidak punya galat domain sendiri — ia baca-saja, sehingga tidak ada isian yang
// dapat cacat maupun baris yang dapat bentrok. Yang lewat sini hanyalah galat portal (yang
// dipetakan pembungkusnya) dan kegagalan teknis (yang dijawab 500 dengan pesan umum).
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	logging.From(r.Context(), h.Logger).Warn("permintaan Master Pasal AI gagal",
		slog.String("jalur", r.URL.Path),
		slog.String("galat", err.Error()),
	)
	h.WriteError(w, r, err)
}

// Mount mendaftarkan seluruh rute modul Master Pasal AI.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang merakit
// urutannya adalah `cmd/claimpnc`.
//
// # Satu rute saja
//
// Modul ini baca-saja, dan layar lamanya tidak punya jalur membuka satu baris — ketiga
// kolomnya sudah muat di dalam grid. Menambahkan `GET /{id}` berarti menyediakan jalur tanpa
// pemakai, dengan kunci yang bahkan tidak diketahui ada atau tidak.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	portalhttp.MountGets(r, portalDeps,
		portalhttp.Route{Path: "/master/pasal-ai", Handler: h.List},
	)
}
