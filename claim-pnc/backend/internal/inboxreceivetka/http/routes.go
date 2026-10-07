package inboxreceivetkahttp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// MaxKeywordLength membatasi panjang penyaring `cari`.
//
// Ia BUKAN aturan bisnis. Batasnya dipasang karena kata kunci masuk ke dalam pola LIKE, dan
// kata kunci sepanjang megabita hanya menghasilkan pemindaian penuh yang pasti tidak
// menemukan apa pun.
//
// Seratus disamakan dengan modul lain: tidak ada nomor klaim, nomor polis, nama tertanggung,
// maupun nama peserta yang lebih panjang dari itu, sehingga batas ini tidak pernah menolak
// pencarian yang sungguh-sungguh.
const MaxKeywordLength = 100

// maxRequestBody membatasi ukuran badan permintaan.
//
// Badan permintaan modul ini hanya memuat dua field pendek. Batas ini bukan penyaring
// bisnis melainkan pagar terhadap badan raksasa yang hanya menghabiskan memori sebelum
// ditolak.
const maxRequestBody = 8 << 10

// List menangani GET /inbox/receive-tka.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada nomor klaim, nomor polis, nama tertanggung, nama peserta
//
// TANPA penyaring penanda TKA: keanggotaan tabel sumbernya yang menentukan, dan menjadikannya
// parameter berarti menyediakan cara membaca klaim non-TKA lewat endpoint ini.
//
// TANPA penyaring tanggal kelengkapan dokumen: daftar ini menurut definisinya hanya memuat
// yang belum terisi.
//
// TANPA paginasi. Work Owner memilih penyaringan dan paginasi dikerjakan peramban, sehingga
// satu permintaan mengembalikan seluruh daftar sampai batas `inboxreceivetka.MaxRows` — dan
// menyatakan bila terpotong.
//
// # Ia GET, dan memang aman diulang
//
// Tidak ada keadaan yang berubah karenanya.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, keyword, ok := portalhttp.ActiveKeyword(w, r, MaxKeywordLength, h.writeModuleError, h.keywordTooLong)
	if !ok {
		return
	}

	page, err := h.Service.List(r.Context(), active.Alias, keyword)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, toListResponse(page, active.Alias))
}

// Complete menangani POST /inbox/receive-tka/kelengkapan-dokumen.
//
// Ia padanan tombol **Submit** pada layar lama, yang menjalankan
// `Activity/SubmitTanggalLengkapTKA-Act.xml`.
//
// # Kenapa POST ke sub-sumber daya, bukan PATCH pada barisnya
//
// `10-API-STRATEGY.md` §2 menetapkan aksi bisnis dimodelkan sebagai PERISTIWA, bukan
// pembaruan field. Mengisi tanggal kelengkapan dokumen bukan sekadar menyetel satu kolom: ia
// memindahkan pekerjaan keluar dari inbox, menulis DUA tabel dalam satu transaksi, dan
// melepaskan pemberitahuan. `PATCH` dengan `{"tanggal_dokumen_lengkap": ...}` akan
// menyembunyikan ketiganya di balik bentuk yang tampak seperti penyuntingan biasa.
//
// # Ia TIDAK idempoten, dan itu perilaku yang benar
//
// Permintaan kedua atas baris yang sama ditolak dengan 409 — barisnya sudah tidak memenuhi
// penyaring inbox. `10-API-STRATEGY.md` §7 mewajibkan kunci idempotensi pada aksi yang
// menimbulkan akibat di luar sistem, dan surel memang akibat seperti itu; yang menggantikan
// kunci itu di sini adalah penyaring `TGL_DOC_LENGKAP IS NULL` yang ikut dikunci di dalam
// transaksi. Dua permintaan yang tiba bersamaan hanya membuat satu di antaranya berhasil,
// sehingga surel ganda tidak dapat terjadi.
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	var body CompleteRequest
	if !h.decodeBody(w, r, &body) {
		return
	}

	completion, err := parseCompletion(body)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	result, err := h.Service.Complete(r.Context(), active.Alias, completion)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, toCompleteResponse(
		result.Task.ClaimNumber,
		completion.CompletedAt,
		result.NotificationAttempted,
		result.NotificationSent,
		active.Alias,
	))
}

// decodeBody membaca badan JSON, dan menjawab 400 bila tidak dapat dibaca.
//
// Nilai baliknya menyatakan apakah pemanggil boleh melanjutkan.
func (h *Handler) decodeBody(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)

	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam. Jawaban daftar memuat
	// tujuh field per baris sementara permintaan ini hanya menerima dua; klien yang
	// mengirim balik seluruh objek baris harus mengetahuinya saat pertama dicoba — bukan
	// menemukan bahwa lima di antaranya diam-diam tidak berpengaruh.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
		h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	return true
}

// Mount mendaftarkan rute modul Inbox Receive TKA.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// Daftar pekerjaan ada di basis data SETIAP entitas, dan isinya memuat nama tertanggung
// serta nama peserta — data nasabah satu badan hukum. Rute tulisnya bahkan lebih berat lagi:
// tanpa pemeriksaan portal, sebuah permintaan dapat mengubah tanggal pada klaim milik badan
// hukum lain.
//
// Permintaan tanpa header portal DITOLAK, tidak pernah dilayani portal utama sebagai
// cadangan (`R-20`, `TKT-F6-002`).
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
//
// # Kenapa di bawah `/inbox/...`
//
// INBOX adalah kelompok menu tersendiri di sistem lama — `MENU_ID 2`, induk dari 30 butir,
// dan `MENU_ID 49` ini salah satunya. Inbox Investigator sudah menempati awalan yang sama.
//
// # Yang TIDAK didaftarkan
//
// **Tidak ada PUT maupun DELETE.** Tanggal yang sudah diisi tidak dapat diubah maupun
// dibatalkan lewat layar ini, dan itu bukan kekurangan yang belum digarap: layar lamanya pun
// tidak bisa. Begitu `TGL_DOC_LENGKAP` terisi, barisnya lenyap dari inbox dan tidak ada satu
// pun kendali di harness itu yang dapat memanggilnya kembali. Membuat pembatalan berarti
// membuat kemampuan BARU yang belum pernah diminta dan belum diputuskan siapa yang berhak.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/inbox/receive-tka", h.List)
		perPortal.Post("/inbox/receive-tka/kelengkapan-dokumen", h.Complete)
	})
}

// keywordTooLong menjawab kata pencarian yang melampaui MaxKeywordLength.
func (h *Handler) keywordTooLong(w http.ResponseWriter, r *http.Request) {
	h.WriteResponse(w, r, http.StatusBadRequest, ErrorResponse{Code: CodeMalformedRequest, Message: "Kata pencarian terlalu panjang."})
}
