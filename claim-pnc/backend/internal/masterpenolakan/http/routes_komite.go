package masterpenolakanhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/masterpenolakan"
	"claim-pnc/internal/portal"

	portalhttp "claim-pnc/internal/portal/http"
)

// ListKomite menangani GET /master/penolakan-komite.
func (h *Handler) ListKomite(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	list, err := h.komite.List(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, KomiteListResponse{
		CommitteeRejection: toKomiteListDTO(list),
		Portal:             active.Alias,
	})
}

// CreateKomite menangani POST /master/penolakan-komite.
func (h *Handler) CreateKomite(w http.ResponseWriter, r *http.Request) {
	active, request, ready := h.prepareKomite(w, r)
	if !ready {
		return
	}

	saved, err := h.komite.Create(r.Context(), active.Alias, masterpenolakan.InputKomite{Note: request.Note})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, KomiteSingleResponse{
		CommitteeRejection: toKomiteDTO(saved),
		Portal:             active.Alias,
	})
}

// UpdateKomite menangani PUT /master/penolakan-komite/{id}.
//
// Ini yang di layar lama berupa tombol "ubah" pada setiap baris grid
// (`RDB List/GetMasterRejectedKomites-SQL.xml` mengirim labelnya sebagai kolom ketiga).
// Hanya catatannya yang berubah; IDMASTER adalah kunci dan tidak pernah di-SET.
func (h *Handler) UpdateKomite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeModuleError(w, r, masterpenolakan.ErrKomiteNotFound)
		return
	}

	active, request, ready := h.prepareKomite(w, r)
	if !ready {
		return
	}

	saved, err := h.komite.Update(r.Context(), active.Alias, id, masterpenolakan.InputKomite{Note: request.Note})
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusOK, KomiteSingleResponse{
		CommitteeRejection: toKomiteDTO(saved),
		Portal:             active.Alias,
	})
}

// prepareKomite memeriksa portal aktif lalu membaca badan permintaan.
//
// Ia TIDAK memeriksa identitas pemanggil seperti prepare pada tab sebelah, dan itu bukan
// kelalaian: POOLDATA.MST_REJECTED_KOMITE tidak punya kolom pelaku sama sekali — hanya
// IDMASTER dan NOTEMASTER. Menuntut identitas lalu membuangnya akan menyiratkan ada jejak
// yang tersimpan, padahal tidak ada.
//
// Akibatnya perubahan pada master ini tidak meninggalkan jejak siapa dan kapan.
// Keterbatasan itu milik skemanya, bukan milik lapisan ini, dan menambah kolom menuntut
// persetujuan Work Owner serta pelaksanaan DBA (`D-63`).
//
// Nilai terakhir false bila responsnya sudah ditulis.
func (h *Handler) prepareKomite(w http.ResponseWriter, r *http.Request) (portal.Portal, KomiteSaveRequest, bool) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return portal.Portal{}, KomiteSaveRequest{}, false
	}

	var request KomiteSaveRequest
	if !h.decode(w, r, &request) {
		return portal.Portal{}, KomiteSaveRequest{}, false
	}
	return active, request, true
}
