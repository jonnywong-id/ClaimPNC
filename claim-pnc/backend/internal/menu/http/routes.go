package menuhttp

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/platform/apierror"
	"claim-pnc/internal/platform/logging"
)

// KodeMenuTidakTerkonfigurasi: baris M_APLIKASI untuk aplikasi ini tidak ada.
//
// Ia dibedakan dari galat internal biasa supaya jawabannya dapat ditindaklanjuti:
// yang salah bukan permintaan pengguna melainkan isi master, dan yang memperbaikinya
// DBA — bukan pengguna yang mencoba lagi.
const KodeMenuTidakTerkonfigurasi = "menu_tidak_terkonfigurasi"

// JSONWriter menuliskan badan respons yang berhasil.
type JSONWriter = apierror.JSONWriter

// ErrorWriter menuliskan galat dalam bentuk respons HTTP.
type ErrorWriter = apierror.ErrorWriter

// List menangani GET /api/menu.
//
// Yang dikembalikan hanyalah butir yang boleh DILIHAT pemanggil. Itu tetap BUKAN
// kendali akses (`D-59`): yang menggerbang adalah pemeriksaan di server pada setiap
// endpoint modulnya masing-masing. Menyaring di sini hanya membuat layar tidak
// menawarkan pintu yang pasti tertutup.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	caller, existing := h.Caller(r.Context())
	if !existing {
		// Rutenya berada di balik middleware sesi, sehingga keadaan ini berarti
		// perakitannya keliru — bukan permintaan yang cacat. Diserahkan ke penulis galat
		// bersama, yang menjawab 500 dengan pesan umum dan menaruh rinciannya di log.
		h.WriteError(w, r, errors.New("menu/http: konteks pemanggil tidak ada di balik middleware sesi"))
		return
	}

	tree, err := h.Service.ForLogin(r.Context(), caller.Login)
	if err != nil {
		if errors.Is(err, menu.ErrAppNotFound) {
			logging.From(r.Context(), h.Logger).Error("menu tidak dapat disusun",
				slog.String("jalur", r.URL.Path),
				slog.String("sebab", "baris M_APLIKASI dengan APP_DESC "+menu.AppName+" tidak ada"),
			)
			h.WriteResponse(w, r, http.StatusInternalServerError, ErrorResponse{
				Kode:  KodeMenuTidakTerkonfigurasi,
				Pesan: "Menu aplikasi belum terdaftar. Hubungi administrator Claim PNC.",
			})
			return
		}
		h.WriteError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, ListResponse{Menu: toListDTO(tree)})
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
