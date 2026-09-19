package menuhttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/menu/usecase"
	"claim-pnc/internal/platform/logging"
)

// KodeMenuTidakTerkonfigurasi: baris M_APLIKASI untuk aplikasi ini tidak ada.
//
// Ia dibedakan dari galat internal biasa supaya jawabannya dapat ditindaklanjuti:
// yang salah bukan permintaan pengguna melainkan isi master, dan yang memperbaikinya
// DBA — bukan pengguna yang mencoba lagi.
const KodeMenuTidakTerkonfigurasi = "menu_tidak_terkonfigurasi"

// Caller adalah bagian identitas pemanggil yang dibutuhkan modul ini.
//
// Hanya SATU field, dan itu disengaja: menu hanya perlu tahu login siapa yang bertanya.
// Menerima seluruh catatan pengguna akan membuat modul ini bergantung pada bentuk data
// modul auth.
type Caller struct {
	// Login adalah yang DIKETIK pengguna di layar masuk — itulah yang dicocokkan ke
	// M_LOGIN_GROUP_PNC.LOGIN_ID dan M_OTORISASI_PNC.LOGIN_ID_GROUP, sesuai aturan yang
	// ditetapkan Work Owner 2026-09-18.
	Login string
}

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter func(w http.ResponseWriter, r *http.Request, status int, body any)

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter func(w http.ResponseWriter, r *http.Request, err error)

// Handler melayani permintaan menu.
type Handler struct {
	service       *usecase.Service
	caller        func(ctx context.Context) (Caller, bool)
	logger        *slog.Logger
	writeResponse JSONWriter
	writeError    ErrorWriter
}

// Options adalah bahan pembentuk Handler.
type Options struct {
	Service *usecase.Service

	// Caller adalah jembatan SATU ARAH dari modul auth ke modul ini. Ia disuntikkan
	// dari cmd, bukan diimpor, supaya kedua modul tetap tidak saling mengimpor — yang
	// tahu keduanya hanyalah berkas perakitan.
	Caller func(ctx context.Context) (Caller, bool)

	Logger *slog.Logger

	// WriteResponse dan WriteError disuntikkan dari cmd supaya seluruh modul menuliskan
	// respons dan galat sesi dengan cara yang sama.
	WriteResponse JSONWriter
	WriteError    ErrorWriter
}

// NewHandler membentuk handler modul menu.
func NewHandler(o Options) (*Handler, error) {
	if o.Service == nil {
		return nil, errors.New("menu/http: Service wajib diisi")
	}
	if o.Caller == nil {
		return nil, errors.New("menu/http: Caller wajib diisi")
	}
	if o.WriteResponse == nil || o.WriteError == nil {
		return nil, errors.New("menu/http: WriteResponse dan WriteError wajib diisi")
	}
	return &Handler{
		service:       o.Service,
		caller:        o.Caller,
		logger:        o.Logger,
		writeResponse: o.WriteResponse,
		writeError:    o.WriteError,
	}, nil
}

// List menangani GET /api/menu.
//
// Yang dikembalikan hanyalah butir yang boleh DILIHAT pemanggil. Itu tetap BUKAN
// kendali akses (`D-59`): yang menggerbang adalah pemeriksaan di server pada setiap
// endpoint modulnya masing-masing. Menyaring di sini hanya membuat layar tidak
// menawarkan pintu yang pasti tertutup.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caller, existing := h.caller(r.Context())
	if !existing {
		// Rutenya berada di balik middleware sesi, sehingga keadaan ini berarti
		// perakitannya keliru — bukan permintaan yang cacat. Diserahkan ke penulis galat
		// bersama, yang menjawab 500 dengan pesan umum dan menaruh rinciannya di log.
		h.writeError(w, r, errors.New("menu/http: konteks pemanggil tidak ada di balik middleware sesi"))
		return
	}

	tree, err := h.service.ForLogin(r.Context(), caller.Login)
	if err != nil {
		if errors.Is(err, menu.ErrAppNotFound) {
			logging.From(r.Context(), h.logger).Error("menu tidak dapat disusun",
				slog.String("jalur", r.URL.Path),
				slog.String("sebab", "baris M_APLIKASI dengan APP_DESC "+menu.AppName+" tidak ada"),
			)
			h.writeResponse(w, r, http.StatusInternalServerError, ErrorResponse{
				Kode:  KodeMenuTidakTerkonfigurasi,
				Pesan: "Menu aplikasi belum terdaftar. Hubungi administrator Claim PNC.",
			})
			return
		}
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, ListResponse{Menu: toListDTO(tree)})
}

// Mount mendaftarkan rute modul menu.
//
// # Yang dituntut pemanggil
//
// Rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini tidak
// memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth; yang
// merakit urutannya adalah cmd/claimpnc.
//
// # Kenapa TIDAK dipasangi middleware portal
//
// Peta menu dan kewenangannya hidup di basis data portal utama dan tidak punya kolom
// entitas — menu seseorang sama di keempat portal. Menuntut portal di sini akan membuat
// menunya gagal justru saat pengguna belum memilih entitas, padahal tidak ada satu baris
// data entitas pun yang dibacanya.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler) {
	r.Get("/menu", h.List)
}
