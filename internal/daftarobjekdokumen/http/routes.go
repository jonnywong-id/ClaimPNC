package daftarobjekdokumenhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat satu isian pendek ditambah daftar nama bisnis; 64 KiB sudah
// jauh lebih dari cukup. Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum
// memakan memori, bukan setelah.
const maxRequestBody = 64 << 10

// Handler melayani permintaan Daftar Objek Dokumen.
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
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul dapat
	// dipindahkan tanpa menariknya serta.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Daftar Objek Dokumen.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("daftarobjekdokumen/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("daftarobjekdokumen/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/objek-dokumen.
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
		DocumentObject: content,
		Total:          len(content),
		Portal:         active.Alias,
	})
}

// Get menangani GET /master/objek-dokumen/{id}.
//
// Rute tersendiri, bukan sekadar mencari di hasil List, karena hanya di sini pemetaan
// bisnisnya ikut dimuat — dan layar membutuhkannya tepat saat baris dibuka untuk disunting,
// bukan saat daftarnya ditampilkan.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, daftarobjekdokumen.ErrNotFound)
		return
	}

	row, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		DocumentObject: toDTO(row),
		Portal:         active.Alias,
	})
}

// Create menangani POST /master/objek-dokumen.
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

	saved, err := h.service.Create(r.Context(), active.Alias, requestToInput(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan beserta ID-nya. ID
	// diterbitkan server, sehingga layar tidak punya cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		DocumentObject: toDTO(saved),
		Portal:         active.Alias,
	})
}

// Update menangani PUT /master/objek-dokumen/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, daftarobjekdokumen.ErrNotFound)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Update(r.Context(), active.Alias, id, requestToInput(request))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		DocumentObject: toDTO(saved),
		Portal:         active.Alias,
	})
}

// requestToInput mengubah badan permintaan menjadi masukan domain.
func requestToInput(request SaveRequest) daftarobjekdokumen.Input {
	return daftarobjekdokumen.Input{
		Description:   request.Description,
		BusinessNames: request.Businesses,
	}
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	var request SaveRequest

	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
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

// Mount mendaftarkan rute modul Daftar Objek Dokumen.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// Middleware PortalAktif dipasang di sini pada seluruh rute: setiap tabel yang disentuh
// modul ini hidup di basis data setiap entitas.
//
// # Kenapa /master/bisnis TIDAK didaftarkan di sini
//
// Karena sudah dimiliki modul Master COL Simas Online, dan chi akan PANIK saat start bila
// dua modul mendaftarkan jalur yang sama. Layar modul ini memakai rute yang sama itu —
// daftar bisnis adalah data acuan bersama, bukan milik satu layar, dan menduplikasinya
// dengan jalur berbeda akan membuat dua layar menampilkan master yang seolah berbeda.
//
// Konsekuensi yang harus disadari: bila kelak ada modul Master Bisnis tersendiri, rute itu
// pindah ke sana dan kedua modul ini menjadi pemakainya. Yang tidak boleh terjadi adalah
// dua modul mendaftarkannya bersamaan — dan panik chi itu justru yang membuat kekeliruan
// ini mustahil lolos diam-diam.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Route("/master/objek-dokumen", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{id}", h.Get)

			// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali,
			// sehingga permintaannya menggantikan dan idempoten. Mengirim permintaan yang
			// sama dua kali menghasilkan keadaan akhir yang sama — termasuk untuk daftar
			// bisnisnya, yang memang diganti seluruhnya.
			master.Put("/{id}", h.Update)
		})
	})
}
