package inboxosclaimpercabanghttp

import (
	"net/http"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/httpquery"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// List menangani GET /api/inbox-os-claim-per-cabang.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	query := r.URL.Query()

	listed, err := h.Service.List(
		r.Context(),
		active.Alias,
		caller,
		inboxosclaimpercabang.Pagination{
			Page: positiveNumber(query.Get("halaman")),
			Size: positiveNumber(query.Get("ukuran")),
		},
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toListResponse(listed, active.Alias))
}

// prepare memeriksa portal dan identitas pemanggil sekaligus.
//
// Ia dikumpulkan karena List dan Export menuntut keduanya dengan urutan yang sama, dan
// urutan itu penting: portal diperiksa LEBIH DULU, supaya permintaan tanpa portal dijawab
// sebagai permintaan tanpa portal — bukan sebagai sesi yang tidak lengkap (`R-20`).
func (h *Handler) prepare(w http.ResponseWriter, r *http.Request) (
	portal.Portal, inboxosclaimpercabang.Caller, bool,
) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return portal.Portal{}, inboxosclaimpercabang.Caller{}, false
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inboxosclaimpercabang.ErrCallerUnknown)
		return portal.Portal{}, inboxosclaimpercabang.Caller{}, false
	}

	return active, caller, true
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
//
// Cabang yang kosong TIDAK membuat pemanggil dianggap tidak terbaca. Keduanya keadaan yang
// berbeda dan punya jawaban yang berbeda: yang pertama menyuruh pengguna masuk ulang, yang
// kedua menyuruhnya menghubungi Tim IT.
func (h *Handler) readCaller(r *http.Request) (inboxosclaimpercabang.Caller, bool) {
	if h.Caller == nil {
		return inboxosclaimpercabang.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inboxosclaimpercabang.Caller{}, false
	}
	return inboxosclaimpercabang.Caller{
		Login:            caller.Login,
		DetailBranchCode: caller.DetailBranchCode,
	}, true
}

// positiveNumber membaca angka dari parameter query.
//
// Nilai yang tidak dapat dibaca menghasilkan 0, dan Pagination.Normalize membetulkannya
// menjadi nilai bawaan. Menolak seluruh permintaan karena `halaman=abc` akan membuat layar
// gagal tanpa alasan yang terbaca pengguna — sementara menampilkan halaman pertama adalah
// jawaban yang selalu masuk akal.
func positiveNumber(raw string) int { return httpquery.NonNegative(raw) }
