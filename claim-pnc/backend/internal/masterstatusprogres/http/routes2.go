package masterstatusprogreshttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// Handler2 melayani permintaan master status progres tingkat 2.
//
// Ia terpisah dari Handler tingkat 1, bukan menambah method padanya, karena keduanya
// memegang layanan yang berbeda. Menyatukannya berarti mengubah Options milik Handler yang
// sudah dirakit di cmd — perubahan yang membongkar, bukan menambah.
type Handler2 struct {
	service       *usecase.Service2
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options2 adalah bahan pembentuk Handler2.
type Options2 struct {
	Service *usecase.Service2
	Logger  *slog.Logger

	// TulisRespon dan TulisGalat disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler2 membentuk handler modul master status progres tingkat 2.
func NewHandler2(o Options2) (*Handler2, error) {
	if o.Service == nil {
		return nil, errors.New("masterstatusprogres/http: Layanan tingkat 2 wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterstatusprogres/http: TulisRespon dan TulisGalat wajib diisi")
	}
	return &Handler2{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/status-progres-2.
func (h *Handler2) List(w http.ResponseWriter, r *http.Request) {
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

	h.writeResponse(w, r, http.StatusOK, ListResponse2{
		ProgressStatus2: toListDTO2(list),
		Portal:          active.Alias,
	})
}

// Parent menangani GET /master/status-progres-2/induk.
//
// Rutenya DIPASANGI PortalAktif, berbeda dari daftar posisi klaim pada tingkat 1: isinya
// dibaca dari tabel tingkat 1 milik entitas yang bersangkutan, bukan daftar tetap milik
// aplikasi. Dua entitas punya Status Progres 1 yang berbeda, dan menyajikan daftar satu
// entitas kepada entitas lain adalah kebocoran yang justru dicegah R-20.
func (h *Handler2) Parent(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.ListParents(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ParentListResponse{
		Parent: toParentListDTO(list),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/status-progres-2.
func (h *Handler2) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	request, parsed := h.readRequest(w, r)
	if !parsed {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, masterstatusprogres.Input2{
		Name:     request.Name,
		ParentID: request.ParentID,
	})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID dan
	// ParentName yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse2{
		ProgressStatus2: toDTO2(saved),
		Portal:          active.Alias,
	})
}

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler2) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest2, bool) {
	var request SaveRequest2

	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan
		// badan permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest2{}, false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return SaveRequest2{}, false
	}

	return request, true
}

// writeModuleError memakai pemetaan yang sama dengan tingkat 1.
//
// Ia disalin ke sini alih-alih dipakai bersama lewat satu method, karena keduanya
// bergantung pada penerima yang berbeda (Handler dan Handler2) sementara isinya hanya
// beberapa baris. Pemetaannya sendiri — petakanGalat — TIDAK disalin: ia satu fungsi yang
// dipakai keduanya, sehingga tidak ada dua tafsiran atas galat yang sama.
func (h *Handler2) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		h.writeError(w, r, err)
		return
	}

	if status >= http.StatusInternalServerError {
		logging.From(r.Context(), h.logger).Error("permintaan gagal",
			slog.String("jalur", r.URL.Path),
			slog.String("galat", err.Error()),
		)
	}
	h.writeResponse(w, r, status, body)
}

// Mount2 mendaftarkan rute modul master status progres tingkat 2.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// SELURUH rute dipasangi PortalAktif — berbeda dari tingkat 1, yang menyisakan daftar
// posisi klaim di luarnya. Di sini tidak ada satu pun rute yang isinya milik aplikasi:
// daftar induk pun dibaca dari basis data entitas.
//
// # Yang TIDAK didaftarkan, dan kenapa
//
// Tidak ada PUT maupun DELETE. Sistem lama tidak memiliki satu pun pernyataan yang
// mengubah atau menghapus isi POOLDATA.GCNM_MST_PROGRESS; buktinya ada pada doc comment
// masterstatusprogres.Repo2. Mendaftarkan rute yang tidak dapat berbuat apa-apa hanya
// memindahkan kejutannya dari layar ke API.
//
// # Kenapa jalurnya tanpa /v1
//
// Alasannya sama dengan Mount tingkat 1: kontrak API yang ada belum memakai awalan
// versi, dan memperkenalkannya di satu modul saja akan membuat dua gaya jalur hidup
// berdampingan. Penyeragamannya dicatat sebagai utang teknis.
func Mount2(r chi.Router, h *Handler2, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		// Jalur induk didaftarkan LEBIH DULU daripada rute ber-parameter apa pun pada
		// prefiks yang sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya
		// sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat pembaca
		// tidak perlu mengetahui hal itu untuk yakin.
		perPortal.Get("/master/status-progres-2/induk", h.Parent)

		perPortal.Get("/master/status-progres-2", h.List)
		perPortal.Post("/master/status-progres-2", h.Create)
	})
}
