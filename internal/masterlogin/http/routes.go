package masterloginhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/usecase"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat empat isian pendek. 32 KiB sudah jauh lebih dari cukup —
// angkanya disamakan dengan modul master lain alih-alih diperkecil, supaya tidak ada satu
// modul yang diam-diam menolak permintaan yang diterima modul tetangganya.
const maxRequestBody = 32 << 10

// Caller adalah pemanggil yang sudah terverifikasi sesinya.
//
// Satu field, dan ia dipakai DUA hal — perbedaan yang menentukan:
//
//  1. **Menurunkan LOGINLEADER** pada penambahan. Itu DATA yang tersimpan, bukan
//     pencatatan. Lihat usecase.Service.Create.
//  2. **Mengisi log.** Tabelnya tidak punya kolom pencatat pelaku sama sekali.
//
// Karena yang pertama, identitas pemanggil di modul ini BUKAN sekadar pelengkap: tanpa
// nilainya, setiap baris baru lahir tanpa tautan tim.
type Caller struct {
	Login string
}

// CallerReader mengambil identitas pemanggil dari konteks permintaan.
//
// Ia jembatan SATU ARAH dari modul auth, disuntikkan dari cmd. Modul ini tidak mengimpor
// lapisan transport modul auth — itulah yang membuat keduanya dapat berpindah tanpa
// menyeret satu sama lain.
type CallerReader func(ctx context.Context) (Caller, bool)

// Handler melayani permintaan master login surveyor.
type Handler struct {
	service       *usecase.Service
	caller        CallerReader
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service
	Caller  CallerReader
	Logger  *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd, bukan diimpor dari modul auth.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul master login.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("masterlogin/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("masterlogin/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New(
			"masterlogin/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /master/login.
//
// # Penyaringnya
//
//	?cari=...  mempersempit pada Nama, Login, dan Email
//
// TANPA penyaring status: tabelnya tidak punya kolom APPROVAL, dan layar lamanya tidak
// bertab. Lihat banner masterlogin.SurveyorLogin.
//
// Cakupan daftarnya SELURUH baris tabel; rule yang mengisi grid Pega tidak ada di export
// (`R-16`), dan alasannya beserta bacaan lain yang mungkin ada pada masterlogin.Filter.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.service.List(r.Context(), active.Alias, r.URL.Query().Get("cari"))
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{
		Login:  toListDTO(list),
		Portal: active.Alias,
	})
}

// Get menangani GET /master/login/{login}.
//
// Jalurnya memakai LOGIN, bukan sebuah ID terpisah: tabelnya tidak punya kunci lain, dan
// setiap pernyataan simpannya menyaring `where login = ...`.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	login := chi.URLParam(r, "login")
	if login == "" {
		h.writeModuleError(w, r, masterlogin.ErrNotFound)
		return
	}

	found, err := h.service.Get(r.Context(), active.Alias, login)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Login:  toDTO(found),
		Portal: active.Alias,
	})
}

// Create menangani POST /master/login.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras — dan pada modul ini akibatnya lebih dari sekadar log tanpa
		// pelaku: identitas pemanggil yang hilang membuat LOGINLEADER baris baru kosong
		// tanpa satu pun tanda.
		h.writeError(w, r, errors.New(
			"masterlogin/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk LOGIN,
	// STSLOGIN, dan LOGINLEADER yang ketiganya diturunkan server, sehingga layar tidak punya
	// cara lain mengetahuinya.
	h.writeResponse(w, r, http.StatusCreated, SingleResponse{
		Login:  toDTO(saved),
		Portal: active.Alias,
	})
}

// Save menangani PUT /master/login/{login}.
//
// Nama yang dikirim WAJIB sama dengan yang tersimpan; lihat masterlogin.ErrNameLocked. Ia
// tetap diminta — bukan dihilangkan dari badan permintaan — supaya layar mengirim form yang
// utuh dan server dapat menolak permintaan yang mengiranya dapat diubah, alih-alih
// membuangnya diam-diam.
func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	by, known := h.caller(r.Context())
	if !known {
		h.writeError(w, r, errors.New(
			"masterlogin/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	login := chi.URLParam(r, "login")
	if login == "" {
		h.writeModuleError(w, r, masterlogin.ErrNotFound)
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.service.Save(r.Context(), active.Alias, login, request.toInput(),
		usecase.Actor{Login: by.Login}, h.logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, SingleResponse{
		Login:  toDTO(saved),
		Portal: active.Alias,
	})
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	reader := http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(reader)
	// Field yang tidak dikenal ditolak, tidak diabaikan diam-diam. Pada modul ini itu lebih
	// dari sekadar menangkap salah ketik: `login`, `status_login`, dan `login_leader`
	// DIKIRIM pada setiap jawaban tetapi tidak dapat dikirim balik, dan klien yang
	// mengembalikan seluruh objek apa adanya harus mengetahuinya saat pertama dicoba —
	// bukan menemukan bahwa ketiganya diam-diam tidak berpengaruh.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		// Rincian galat penguraian tidak dikirim ke peramban: isinya memuat cuplikan badan
		// permintaan.
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	// Badan yang memuat lebih dari satu dokumen JSON ditolak.
	if err := decoder.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeMalformedRequest,
			Message: "Permintaan tidak dapat dibaca.",
		})
		return false
	}

	return true
}

// Mount mendaftarkan rute modul master login.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # SELURUH rute dipasangi pemeriksaan portal
//
// `POOLDATA.MST_LOGIN_SURVEYOR` ada di basis data SETIAP entitas dan isinya berbeda — ia
// menentukan siapa yang boleh bekerja sebagai surveyor pada badan hukum itu. Tidak ada satu
// pun rute di modul ini yang isinya milik aplikasi, sehingga tidak ada yang dikecualikan.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini saja
// akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai utang
// teknis, bukan diselesaikan sepihak di satu modul.
//
// # Yang TIDAK didaftarkan
//
// **Tidak ada DELETE.** Tidak satu pun rule di export menghapus baris tabel ini, dan `D-66`
// melarang penghapusan fisik data bernilai bisnis.
//
// Yang harus disadari: tabelnya juga TIDAK punya penanda aktif, sehingga tidak ada cara
// menyatakan sebuah login sudah tidak berlaku — bukan lewat penghapusan, dan bukan lewat
// penonaktifan. Itu keterbatasan tabelnya, dan ia dinyatakan di layar alih-alih ditutupi
// dengan tombol yang mengarang kolom baru.
//
// **Tidak ada jalur keputusan.** Tabelnya tidak punya kolom APPROVAL; lihat banner
// masterlogin.SurveyorLogin.
//
// **Tidak ada endpoint `/pilihan`.** Modul ini tidak punya tabel acuan — kelima isiannya
// diketik bebas, dan dua kolom sisanya diturunkan server.
//
// # Urutan pendaftaran
//
// Rute ber-parameter `{login}` didaftarkan setelah rute tanpa parameter pada prefiks yang
// sama. chi mencocokkan segmen statis lebih dulu, sehingga urutannya sebenarnya tidak
// menentukan — tetapi menuliskannya berurutan membuat pembaca tidak perlu mengetahui hal itu.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/login", h.List)
		perPortal.Post("/master/login", h.Create)
		perPortal.Get("/master/login/{login}", h.Get)
		perPortal.Put("/master/login/{login}", h.Save)
	})
}
