package masterloginhttp

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterlogin"
	"claim-pnc/internal/masterlogin/usecase"
	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat empat isian pendek. 32 KiB sudah jauh lebih dari cukup —
// angkanya disamakan dengan modul master lain alih-alih diperkecil, supaya tidak ada satu
// modul yang diam-diam menolak permintaan yang diterima modul tetangganya.
const maxRequestBody = 32 << 10

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

	found, err := h.Service.Get(r.Context(), active.Alias, login)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
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

	by, known := h.Caller(r.Context())
	if !known {
		// Tidak mungkin terjadi di balik middleware Autentikasi. Dinyatakan supaya cacat
		// perakitan gagal keras — dan pada modul ini akibatnya lebih dari sekadar log tanpa
		// pelaku: identitas pemanggil yang hilang membuat LOGINLEADER baris baru kosong
		// tanpa satu pun tanda.
		h.WriteError(w, r, errors.New(
			"masterlogin/http: identitas pemanggil tidak ada di konteks"))
		return
	}

	var request SaveRequest
	if !h.readRequest(w, r, &request) {
		return
	}

	saved, err := h.Service.Create(r.Context(), active.Alias, request.toInput(),
		usecase.Actor{Login: by.Login}, h.Logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	// 201, dan badannya memuat baris yang benar-benar tersimpan — termasuk LOGIN,
	// STSLOGIN, dan LOGINLEADER yang ketiganya diturunkan server, sehingga layar tidak punya
	// cara lain mengetahuinya.
	h.WriteResponse(w, r, http.StatusCreated, SingleResponse{
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

	by, known := h.Caller(r.Context())
	if !known {
		h.WriteError(w, r, errors.New(
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

	saved, err := h.Service.Save(r.Context(), active.Alias, login, request.toInput(),
		usecase.Actor{Login: by.Login}, h.Logger)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, SingleResponse{
		Login:  toDTO(saved),
		Portal: active.Alias,
	})
}

// readRequest membaca badan JSON ke dalam target. Nilai balik false bila responsnya sudah
// ditulis.
func (h *Handler) readRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	return httpjson.Decode(w, r, maxRequestBody, target, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
}

// Mount mendaftarkan rute modul master login.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
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
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
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
