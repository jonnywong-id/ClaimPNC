package inboxlaporanklaimhttp

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/inboxlaporanklaim/usecase"
	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Form Input Receive Document memuat dua isian bernarasi panjang — kronologis kejadian
// dan rincian kerusakan, masing-masing 4.000 karakter. 64 KiB sudah jauh lebih dari
// cukup, dan batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan
// memori, bukan sesudah.
const maxRequestBody = 64 << 10

// CallerLookup membaca identitas pemanggil dari konteks permintaan.
//
// Ia disuntikkan dari cmd, bukan diimpor dari modul auth: modul tidak saling mengimpor
// lapisan transport-nya — itulah yang membuat modul dapat dipindahkan tanpa menariknya
// serta. Pola yang sama dipakai modul Master Rekening dan modul menu.
type CallerLookup func(ctx context.Context) (inboxlaporanklaim.Caller, bool)

// List menangani GET /inbox/laporan-klaim.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	result, err := h.Service.List(r.Context(), active.Alias, caller, readQuery(r))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	now := h.Service.Now()
	h.WriteResponse(w, r, http.StatusOK, ListResponse{
		Laporan:  toListDTO(result.Page.Report, now),
		Kategori: toCategoryDTO(result.Summary),
		Halaman: PaginationDTO{
			Halaman:      result.Page.Pagination.Page,
			Ukuran:       result.Page.Pagination.Size,
			Total:        result.Page.Total,
			TotalHalaman: result.Page.TotalPages(),
		},
		BatasCabang: result.BranchScope,
		Portal:      active.Alias,
	})
}

// Options menangani GET /inbox/laporan-klaim/pilihan.
//
// Lencana pada daftar tab di sini sengaja KOSONG: menghitungnya menuntut penyaring yang
// baru dipilih pengguna setelah layar tergambar. Angkanya datang bersama daftar.
func (h *Handler) Options(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	region, err := h.Service.Regions(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	category := make([]CategoryDTO, 0)
	for _, info := range inboxlaporanklaim.ListCategories() {
		category = append(category, CategoryDTO{
			Kode:       string(info.Category),
			Judul:      info.Title,
			Komunikasi: info.Message,
		})
	}

	business := make([]OptionDTO, 0)
	for _, line := range inboxlaporanklaim.ListBusinessLines() {
		business = append(business, OptionDTO{Kode: string(line.Line), Nama: line.Name})
	}

	area := make([]OptionDTO, 0, len(region))
	for _, item := range region {
		area = append(area, OptionDTO{Kode: item.Code, Nama: item.Name})
	}

	h.WriteResponse(w, r, http.StatusOK, OptionResponse{
		Kategori: category,
		Bisnis:   business,
		Kanwil:   area,
		Portal:   active.Alias,
	})
}

// Get menangani GET /inbox/laporan-klaim/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrNotFound)
		return
	}

	report, err := h.Service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, singleResponse(report, active.Alias, h.Service.Now()))
}

// Save menangani PUT /inbox/laporan-klaim/{id} — tombol Simpan pada form Input Receive
// Document.
//
// PUT, bukan PATCH: form mengirim SELURUH isian setiap kali disimpan, sehingga
// permintaannya menggantikan dan idempoten. Menekan Simpan dua kali menghasilkan keadaan
// yang sama persis.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrNotFound)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.Service.Save(r.Context(), active.Alias, id, caller, toDetail(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, singleResponse(saved, active.Alias, h.Service.Now()))
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest
	ok := httpjson.Decode(w, r, maxRequestBody, &request, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
	return request, ok
}

// singleResponse menyusun jawaban yang memuat satu berkas beserta isian formnya.
func singleResponse(
	report inboxlaporanklaim.ClaimReport,
	portalAlias string,
	now time.Time,
) SingleResponse {
	detail := toDetailDTO(inboxlaporanklaim.DetailOf(report))
	return SingleResponse{
		Laporan: toDTO(report, now),
		Isian:   &detail,
		// Berkas milik Pega hanya dapat dibaca dari sini selama masa paralel
		// (`ADR-0004`, `P-1`). Layar memakai penanda ini untuk menggambar formnya dalam
		// modus baca saja — bukan menyimpulkannya sendiri dari kolom `asal`. Berkas yang
		// sudah menjadi klaim juga terkunci (Work Owner, 2026-09-29).
		DapatDisunting:    report.Editable(),
		SudahDiregistrasi: report.Registered(),
		Portal:            portalAlias,
	}
}

// Create menangani POST /inbox/laporan-klaim — tombol "Buat Baru".
//
// Badan permintaan TIDAK dibaca, dan itu bukan kelalaian: tombolnya di sistem lama tidak
// meminta satu pun isian. Lihat usecase.Service.Create.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	report, err := h.Service.Create(r.Context(), active.Alias, caller)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta nomornya. Nomor
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya — dan layar
	// LANGSUNG membuka form isiannya, persis seperti alur Pega yang meneruskan ke
	// assignment "Receive Document" begitu berkasnya dibuat.
	h.WriteResponse(w, r, http.StatusCreated, singleResponse(report, active.Alias, h.Service.Now()))
}

// prepare mengambil portal aktif dan identitas pemanggil sekaligus.
//
// Nilai ketiga false berarti responsnya sudah ditulis dan pemanggil harus berhenti.
func (h *Handler) prepare(
	w http.ResponseWriter,
	r *http.Request,
) (portal.Portal, inboxlaporanklaim.Caller, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxlaporanklaim.Caller{}, false
	}

	caller, known := h.Caller(r.Context())
	if !known {
		h.writeModuleError(w, r, inboxlaporanklaim.ErrCallerUnknown)
		return portal.Portal{}, inboxlaporanklaim.Caller{}, false
	}
	return active, caller, true
}

// readQuery membaca penyaring dari parameter kueri.
//
// Nilai yang tidak dapat dibaca sebagai angka DIABAIKAN, bukan ditolak: halaman dan
// ukuran halaman adalah kendali tampilan, dan menolak seluruh permintaan karena satu
// parameter cacat hanya memindahkan kerepotan ke layar. Penyaring yang MENENTUKAN ISI —
// kategori dan lini bisnis — tetap ditolak bila tidak dikenal, dan penolakannya terjadi
// di lapisan usecase.
//
// Kategori yang tidak disebut sama sekali jatuh ke tab "All data".
//
// # Kenapa BUKAN tab pertama Pega
//
// Layar lama membuka "Outstanding Data", dan bawaan di sini semula menirunya (`D-13`).
// Akibatnya berkas yang baru dibuat TIDAK TAMPAK saat menu dibuka: berkas baru selalu
// lahir berposisi "Not Transferred", dan tab Outstanding hanya memuat berkas yang sudah
// bernomor klaim.
//
// Work Owner melaporkannya tiga kali (2026-09-24, 2026-09-25) dengan dugaan yang wajar —
// datanya dikira tidak tersimpan — dan meminta perilakunya diubah.
//
// # Kenapa "All data", bukan "Data hasn't been transferred"
//
// Tab "belum diserahkan" juga membuat berkas baru tampak, tetapi hanya sampai berkas itu
// diregistrasi — sesudahnya ia berpindah tab dan menghilang lagi dari tampilan pertama.
// Itu bentuk lain dari keluhan yang sama.
//
// "All data" memuat SELURUH berkas pada tahap mana pun, diurutkan terbaru lebih dulu,
// sehingga tidak ada tahap yang membuat sebuah berkas lenyap dari tampilan pertama.
func readQuery(r *http.Request) usecase.Query {
	value := r.URL.Query()

	category := strings.TrimSpace(value.Get("kategori"))
	if category == "" {
		category = string(inboxlaporanklaim.DefaultCategory)
	}

	return usecase.Query{
		Category:     inboxlaporanklaim.Category(category),
		RegionCode:   value.Get("kanwil"),
		BusinessLine: inboxlaporanklaim.BusinessLine(strings.TrimSpace(value.Get("bisnis"))),
		Keyword:      value.Get("cari"),
		Pagination: inboxlaporanklaim.Pagination{
			Page: number(value.Get("halaman")),
			Size: number(value.Get("ukuran")),
		},
	}
}

func number(text string) int {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0
	}
	return n
}
