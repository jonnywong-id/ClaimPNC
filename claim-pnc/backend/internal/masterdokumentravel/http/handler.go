package masterdokumentravelhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterdokumentravel"
	"claim-pnc/internal/masterdokumentravel/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat satu isian pendek; 64 KiB sudah jauh lebih dari cukup.
// Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum memakan memori, bukan
// setelah.
//
// Ini BUKAN validasi isian — Work Owner menetapkan layar ini tanpa validasi. Yang
// dijaga di sini adalah sumber daya server, bukan aturan bisnis, dan keduanya berbeda:
// yang satu menolak permintaan yang tidak wajar, yang lain menolak isian yang tidak
// sah.
const maxRequestBody = 64 << 10

// Handler melayani permintaan Master Dokumen Travel.
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

// NewHandler membentuk handler modul Master Dokumen Travel.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterdokumentravel/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterdokumentravel/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/master/dokumen-travel.
//
// Menggantikan Report Definition `BrowseMstDocTravel_RD` yang mengisi grid layar
// `BrowseMasterDocumentTravel_Harness`.
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
		TravelDocument: content,
		Total:          len(content),
		Portal:         active.Alias,
	})
}

// Get menangani GET /api/master/dokumen-travel/{id}.
//
// Menggantikan `SetMstDocTravelValue_act(docid)`, yang menjalankan Report Definition
// lalu menyalin baris yang cocok ke halaman `TempMstDocTravel` untuk diisikan ke form.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	doc, err := h.service.Get(r.Context(), active.Alias, chi.URLParam(r, "id"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		TravelDocument: toDTO(doc),
		Portal:         active.Alias,
	})
}

// Create menangani POST /api/master/dokumen-travel.
//
// Menggantikan tombol Tambah lalu Simpan pada harness, yang mengirim sentinel
// `"UnknownID"` supaya procedure memilih cabang INSERT. Di sini ID tidak pernah ikut di
// badan permintaan sama sekali — sentinel itu tidak dibawa.
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

	saved, err := h.service.Create(r.Context(), active.Alias, masterdokumentravel.Input{Name: request.Name})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta ID-nya. ID
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		TravelDocument: toDTO(saved),
		Portal:         active.Alias,
	})
}

// Update menangani PUT /api/master/dokumen-travel/{id}.
//
// PUT, bukan PATCH: seluruh isi yang boleh diubah — satu field, judul — dikirim setiap
// kali, sehingga permintaannya menggantikan dan idempoten. Mengirim permintaan yang sama
// dua kali menghasilkan keadaan akhir yang sama.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterdokumentravel.ErrNotFound)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, masterdokumentravel.Input{Name: request.Name})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		TravelDocument: toDTO(saved),
		Portal:         active.Alias,
	})
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest

	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama
	// field akan terbaca sebagai "isian tidak dikirim" dan menyimpan judul kosong tanpa
	// satu pun tanda bahwa ada yang salah — dan pada modul tanpa validasi, judul kosong
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
