package daftarobjekdokumenhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/platform/crudhttp"
	"claim-pnc/internal/platform/httpjson"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat satu isian pendek ditambah daftar nama bisnis; 64 KiB sudah
// jauh lebih dari cukup. Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum
// memakan memori, bukan setelah.
const maxRequestBody = 64 << 10

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
	ok := httpjson.Decode(w, r, maxRequestBody, &request, h.WriteResponse, ErrorResponse{
		Code:    CodeMalformedRequest,
		Message: "Permintaan tidak dapat dibaca.",
	})
	return request, ok
}

// Mount mendaftarkan rute modul Daftar Objek Dokumen.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
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
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	crudhttp.Mount(r, portalDeps, "/master/objek-dokumen", "id", h)
}
