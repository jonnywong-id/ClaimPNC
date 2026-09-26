package daftardetaildokumentravelhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/daftardetaildokumentravel"
	"claim-pnc/internal/daftardetaildokumentravel/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat beberapa isian pendek ditambah satu senarai pembatasan
// plan. 256 KiB sudah jauh lebih dari cukup — satu aturan dokumen dengan seratus baris
// pembatasan pun tidak mendekati seperempatnya.
//
// Batasnya lebih besar daripada modul master lain justru KARENA senarai itu: modul yang
// hanya punya isian tunggal tidak punya cara menghasilkan badan besar, modul ini punya.
//
// Ini BUKAN validasi isian — Work Owner menetapkan layar ini tanpa validasi. Yang dijaga
// di sini adalah sumber daya server, bukan aturan bisnis, dan keduanya berbeda: yang satu
// menolak permintaan yang tidak wajar, yang lain menolak isian yang tidak sah.
const maxRequestBody = 256 << 10

// Handler melayani permintaan Daftar Detail Dokumen Travel.
type Handler struct {
	service       *usecase.Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
	// dapat dipindahkan tanpa menariknya serta.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Daftar Detail Dokumen Travel.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("daftardetaildokumentravel/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("daftardetaildokumentravel/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/master/daftar-detail-dokumen-travel.
//
// Menggantikan Report Definition `BrowseLstDocTravel_RD` yang mengisi grid layar
// `ListDocumentTravel_Harness`.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.List(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	content := toListDTO(list)
	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Detail: content,
		Total:  len(content),
		Portal: active.Alias,
	})
}

// Get menangani GET /api/master/daftar-detail-dokumen-travel/{id}.
//
// Menggantikan `CNMSetDetailTravelDocument_act`, yang mengisi form dari baris terpilih.
// Berbeda dari List, jawaban ini MEMBAWA daftar pembatasan plan dan jaminannya — dan
// form memang membutuhkannya tepat saat baris dibuka, bukan saat daftarnya ditampilkan.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, daftardetaildokumentravel.ErrNotFound)
		return
	}

	row, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Detail: toDTO(row),
		Portal: active.Alias,
	})
}

// Create menangani POST /api/master/daftar-detail-dokumen-travel.
//
// Menggantikan tombol Tambah lalu Simpan pada harness, yang memanggil
// `CNMInsertDocumentTravel_act` — activity yang TIDAK ADA di export (`R-16`), sehingga
// yang ditiru adalah apa yang disimpannya, bukan urutan langkahnya.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, toInput(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta ID-nya. ID
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Detail: toDTO(saved),
		Portal: active.Alias,
	})
}

// Update menangani PUT /api/master/daftar-detail-dokumen-travel/{id}.
//
// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali, sehingga
// permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama dua kali
// menghasilkan keadaan akhir yang sama — termasuk untuk daftar pembatasan plan-nya, yang
// memang diganti seluruhnya.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, daftardetaildokumentravel.ErrNotFound)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, toInput(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Detail: toDTO(saved),
		Portal: active.Alias,
	})
}

// Documents menangani GET /api/master/dokumen-travel-pilihan.
//
// Menggantikan autocomplete `BrowseMstDocTravel_RD` pada isian ID Dokumen.
//
// Rutenya terpisah dari `/api/master/dokumen-travel` milik modul Master Dokumen Travel
// meski keduanya membaca tabel yang sama, dan itu disengaja: yang satu daftar yang dapat
// disunting, yang lain daftar pilihan. Menyatukannya akan membuat perubahan bentuk
// respons salah satunya merambat ke layar yang tidak ada hubungannya.
func (h *Handler) Documents(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.Documents(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, DocumentListResponse{
		Document: toDocumentListDTO(list),
		Portal:   active.Alias,
	})
}

// Plans menangani GET /api/master/plan-travel.
//
// Menggantikan autocomplete `BrowsePlanTravelMaster_RD` dan `SearchCoverageTravel_RD`
// sekaligus — keduanya membaca POOLDATA.M_PLANTRAVEL yang sama, dan layar selalu
// membutuhkan keduanya bersamaan.
func (h *Handler) Plans(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	plans, err := h.service.Plans(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	coverages, err := h.service.Coverages(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, PlanListResponse{
		Plan:     toPlanListDTO(plans),
		Coverage: toCoverageOptionListDTO(coverages),
		Portal:   active.Alias,
	})
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest

	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama
	// field akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa
	// satu pun tanda bahwa ada yang salah — dan pada modul tanpa validasi, nilai kosong
	// memang akan diterima.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest{}, false
	}

	return request, true
}
