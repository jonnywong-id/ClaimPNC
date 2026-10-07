package mastercolsimasonlinehttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat dua isian pendek ditambah daftar ID bisnis; 64 KiB sudah
// jauh lebih dari cukup. Batasnya ada supaya permintaan bertubuh raksasa ditolak sebelum
// memakan memori, bukan setelah.
const maxRequestBody = 64 << 10

// Business menangani GET /master/bisnis.
func (h *Handler) Business(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.Service.ListBusiness(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.WriteResponse(w, r, http.StatusOK, BusinessListResponse{
		Business: toBusinessListDTO(list),
		Portal:   active.Alias,
	})
}

// requestToInput mengubah badan permintaan menjadi masukan domain.
func requestToInput(request SaveRequest) mastercolsimasonline.Input {
	return mastercolsimasonline.Input{
		Description:   request.Name,
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

// Mount mendaftarkan rute modul master COL Simas Online.
//
// # Yang dituntut pemanggil
//
// Seluruh rute di sini WAJIB sudah berada di balik middleware Autentikasi. Paket ini
// tidak memasangnya sendiri supaya modul tidak mengimpor lapisan transport modul auth;
// yang merakit urutannya adalah cmd/claimpnc.
//
// Middleware PortalAktif dipasang di sini pada SELURUH rute, termasuk daftar bisnis:
// POOLDATA.BUSINESS hidup di basis data setiap entitas, sehingga "bisnis milik siapa"
// ditentukan portal yang aktif. Ini berbeda dari daftar posisi klaim pada modul Master
// Status Progres, yang memang milik aplikasi dan bukan isi basis data entitas mana pun.
//
// # Kenapa /master/bisnis berada di modul ini
//
// Daftar bisnis adalah data acuan yang kelak dipakai lebih dari satu layar, dan jalurnya
// karena itu tidak diberi awalan nama modul — mengikuti `/master/posisi-klaim` yang juga
// dimiliki satu modul tetapi dinamai menurut isinya.
//
// Konsekuensi yang harus disadari: bila kelak ada modul Master Bisnis tersendiri, rute
// ini PINDAH ke sana dan modul ini menjadi pemakainya. Yang tidak boleh terjadi adalah
// dua modul mendaftarkan jalur yang sama — chi akan panik saat start, dan itu justru
// yang membuat kekeliruan ini mustahil lolos diam-diam.
//
// # Kenapa jalurnya tanpa /v1
//
// Kontrak API yang ada belum memakai awalan versi (`/api/masuk`, `/api/portal`).
// `10-API-STRATEGY.md` §2 menetapkan `/api/v1/...`, dan memperkenalkannya di modul ini
// saja akan membuat dua gaya jalur hidup berdampingan. Penyeragamannya dicatat sebagai
// utang teknis, bukan diselesaikan sepihak di satu modul.
func Mount(r chi.Router, h *Handler, portalDeps portalhttp.ActivePortalDeps) {
	r.Group(func(perPortal chi.Router) {
		perPortal.Use(portalhttp.ActivePortal(portalDeps))

		perPortal.Get("/master/bisnis", h.Business)

		perPortal.Route("/master/col-simas-online", func(master chi.Router) {
			master.Get("/", h.List)
			master.Post("/", h.Create)
			master.Get("/{id}", h.Get)

			// PUT, bukan PATCH: seluruh isi yang boleh diubah dikirim setiap kali,
			// sehingga permintaannya menggantikan dan idempoten. Mengirim permintaan
			// yang sama dua kali menghasilkan keadaan akhir yang sama — termasuk untuk
			// daftar bisnisnya, yang memang diganti seluruhnya.
			master.Put("/{id}", h.Update)
		})
	})
}
