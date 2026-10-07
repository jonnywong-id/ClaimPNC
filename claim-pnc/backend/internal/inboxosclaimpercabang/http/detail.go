package inboxosclaimpercabanghttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Detail melayani isi popup untuk satu klaim.
//
// Nomor klaim diambil dari JALUR, bukan dari query string, karena ia yang menentukan sumber
// daya mana yang diminta — bukan cara menyaringnya (`10-API-STRATEGY.md` §2).
//
// # Tidak ada satu pun nilai popup yang diterima dari klien
//
// Tombol Detail di Pega mengirim lima parameter dari baris yang diklik — nomor klaim, nilai
// cadangan, umur, lini bisnis, dan catatan PIC. Di sini hanya NOMOR yang diterima; empat
// sisanya dibaca ulang dari penyimpanan.
//
// Perbedaannya bukan kerapian. Menerima nilai cadangan dari klien berarti angka uang yang
// tampil di popup ditentukan pengirim permintaan, dan itu dapat diubah begitu saja lewat alat
// peramban biasa.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	detailed, err := h.Service.Detail(
		r.Context(),
		active.Alias,
		caller,
		chi.URLParam(r, "nomor"),
	)
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, toDetailResponse(detailed, active.Alias))
}
