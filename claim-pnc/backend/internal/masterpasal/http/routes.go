package masterpasalhttp

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpasal"
	"claim-pnc/internal/masterpasal/usecase"
	"claim-pnc/internal/platform/logging"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Ia lebih besar daripada modul master lain — 256 KiB, bukan 64 KiB — dan itu disengaja.
// Badan permintaan di sini memuat ISI PASAL, yaitu teks ketentuan polis yang dapat
// panjang, ditambah daftar lini bisnis yang panjangnya tidak dibatasi apa pun di layar
// lama.
//
// Batasnya tetap ada: tanpa itu, satu permintaan bertubuh raksasa memakan memori sebelum
// satu pun pemeriksaan sempat berjalan. Ia juga satu-satunya pembatas panjang yang
// dimiliki modul ini, karena lebar kolom IDPASAL dan kapasitas CLOB-nya belum diketahui
// (`R-08`) — lihat masterpasal.Input.Check.
const maxRequestBody = 256 << 10

// Handler melayani permintaan Master Pasal Kerugian.
//
// # Tidak ada Caller di sini, dan itu bukan kelalaian
//
// Modul master lain menerima identitas pemanggil untuk mengisi kolom pencatat siapa.
// POOLDATA.V_M_DATA_PASAL tidak punya kolom semacam itu — hanya IDDATA, IDPASAL, dan
// JSONPASAL — sehingga tidak ada tempat untuk menuliskannya.
//
// Akibatnya dicatat sebagai keterbatasan, bukan ditambal dengan kolom yang dikarang:
// perubahan dan penghapusan di layar ini TIDAK MENINGGALKAN JEJAK di basis data.
// Menambah kolom menempuh `D-63`.
type Handler struct {
	service       *usecase.Service
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	// Service melayani seluruh perkara modul ini. Wajib.
	Service *usecase.Service

	Logger *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	// Modul tidak saling mengimpor lapisan transport-nya — itulah yang membuat modul
	// dapat dipindahkan tanpa menariknya serta.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul Master Pasal Kerugian.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterpasal/http: Service wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("masterpasal/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/pasal-kerugian.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	list, err := h.service.List(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Clause: toListDTO(list),
		Portal: active.Alias,
	})
}

// Get menangani GET /master/pasal-kerugian/{id}.
//
// Ia rute tersendiri, bukan sekadar mengambil satu baris dari daftar yang sudah dimuat
// layar, karena yang dikembalikannya LEBIH LENGKAP: daftar lini bisnisnya ikut dimuat dan
// namanya disegarkan dari master yang berlaku sekarang. Itu persis yang dilakukan tombol
// "Ubah" pada layar lama — `PNCGetListPasalDataCOL_Act(idstatusp=<IDDATA>)`.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpasal.ErrNotFound)
		return
	}

	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	clause, err := h.service.Get(r.Context(), active.Alias, id)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Clause: toDTO(clause),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/pasal-kerugian.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
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

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk ID dan sebutan
	// kategori yang keduanya diterbitkan server, sehingga layar tidak punya cara lain
	// mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Clause: toDTO(saved),
		Portal: active.Alias,
	})
}

// Update menangani PUT /master/pasal-kerugian/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpasal.ErrNotFound)
		return
	}

	active, ready := h.activePortal(w, r)
	if !ready {
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
		Clause: toDTO(saved),
		Portal: active.Alias,
	})
}

// Delete menangani DELETE /master/pasal-kerugian/{id}.
//
// # Ia PERMANEN, dan tidak ada apa pun di belakangnya yang dapat memulihkannya
//
// Peringatan lengkapnya ada pada doc comment masterpasal.Repo. Ringkasnya: `D-66`
// menetapkan soft delete menyeluruh, tabel ini tidak punya kolom penanda terhapus, dan
// Work Owner memilih "jalankan as is" pada 2026-09-19 — yaitu `DELETE` fisik seperti
// Pega.
//
// Konfirmasi sebelum menghapus ada di LAYAR, bukan di sini. Menaruhnya di API berarti
// menambah satu putaran permintaan untuk sesuatu yang hanya dapat dijawab manusia di
// depan layarnya.
//
// Jawabannya 200 dengan badan yang menyebutkan ID yang terhapus, bukan 204 tanpa badan:
// penghapusan ini tidak meninggalkan jejak apa pun di basis data, sehingga jawabannya
// adalah satu-satunya catatan yang sampai ke pemanggil — dan satu-satunya yang muncul di
// log akses.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpasal.ErrNotFound)
		return
	}

	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	if err := h.service.Delete(r.Context(), active.Alias, id); err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	logging.From(r.Context(), h.logger).Info("pasal kerugian dihapus permanen",
		slog.String("portal", active.Alias),
		slog.String("id", id),
	)

	h.writeResponse(w, r, http.StatusOK, DeleteResponse{ID: id, Portal: active.Alias})
}

// Business menangani GET /master/pasal-kerugian/bisnis?cari=...
//
// Daftar pilihan lini bisnis. Ia DIPASANGI pemeriksaan portal: POOLDATA.BUSINESS dibaca
// dari basis data entitas yang sedang dipilih, dan dua entitas dapat punya lini bisnis
// yang berbeda. Menyajikan daftar satu entitas kepada entitas lain adalah kebocoran yang
// justru dicegah R-20.
func (h *Handler) Business(w http.ResponseWriter, r *http.Request) {
	active, ready := h.activePortal(w, r)
	if !ready {
		return
	}

	list, err := h.service.SearchBusiness(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, BusinessListResponse{
		Business: toBusinessListDTO(list),
		Portal:   active.Alias,
	})
}

// Category menangani GET /master/pasal-kerugian/kategori.
//
// Ia TIDAK dipasangi pemeriksaan portal, dan itu satu-satunya rute modul ini yang tidak.
// Isinya milik aplikasi — ketiga pilihan diturunkan dari ekspresi di
// `Activity/CNMInsertPasalDataMaster-Act.xml`, bukan dibaca dari basis data mana pun —
// sehingga menuntut portal di sini akan membuat form gagal dimuat justru saat pengguna
// belum memilih entitas.
func (h *Handler) Category(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(w, r, http.StatusOK, CategoryListResponse{
		Category: toCategoryListDTO(masterpasal.Categories()),
	})
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

// readRequest membaca badan JSON. Nilai kedua false bila responsnya sudah ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request) (SaveRequest, bool) {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam: salah ketik nama field
	// akan terbaca sebagai "isian tidak dikirim" dan menyimpan nilai kosong tanpa satu pun
	// tanda bahwa ada yang salah.
	decoder.DisallowUnknownFields()

	var request SaveRequest
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

// writeModuleError memetakan galat yang dikenali modul ini, dan meneruskan sisanya.
func (h *Handler) writeModuleError(w http.ResponseWriter, r *http.Request, err error) {
	status, body, known := mapError(err)
	if !known {
		// Galat yang tidak dikenali modul ini — kegagalan basis data, kegagalan jaringan,
		// cacat pemrograman — diserahkan ke penulis bersama, yang menjawab 500 dengan
		// pesan umum dan menaruh rinciannya di log saja. Rincian galat internal tidak
		// pernah dikirim ke peramban.
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

// Mount mendaftarkan seluruh rute modul Master Pasal Kerugian.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// # Satu rute di luar pemeriksaan portal
//
// `/kategori` berada di luar grup PortalAktif karena isinya milik aplikasi, bukan dibaca
// dari basis data entitas. Seluruh rute lain — termasuk daftar lini bisnis — dipasangi
// pemeriksaan portal.
//
// # Urutan pendaftaran
//
// Kedua jalur statis di bawah `/master/pasal-kerugian/` didaftarkan LEBIH DULU daripada
// rute ber-parameter pada prefiks yang sama. chi mencocokkan segmen statis lebih dulu,
// sehingga urutannya sebenarnya tidak menentukan — tetapi menuliskannya berurutan membuat
// pembaca tidak perlu mengetahui hal itu untuk yakin bahwa `/kategori` tidak akan terbaca
// sebagai sebuah `{id}`.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Get("/master/pasal-kerugian/kategori", h.Category)

	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/pasal-kerugian/bisnis", h.Business)

		perPortal.Get("/master/pasal-kerugian", h.List)
		perPortal.Post("/master/pasal-kerugian", h.Create)
		perPortal.Get("/master/pasal-kerugian/{id}", h.Get)
		perPortal.Put("/master/pasal-kerugian/{id}", h.Update)
		perPortal.Delete("/master/pasal-kerugian/{id}", h.Delete)
	})
}
