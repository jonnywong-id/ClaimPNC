package inputacceptationhttp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxSubmitBody membatasi besar muatan Submit.
//
// Layar ini mengirim ~11 isian dan lima grid; 1 MiB jauh di atas kebutuhannya. Batasnya ada
// supaya satu permintaan tidak dapat memaksa peladen membaca muatan tanpa batas ke memori —
// dan karena `http.MaxBytesReader` menjawabnya sebagai galat yang terbaca, bukan sebagai
// proses yang menggantung.
const maxSubmitBody = 1 << 20

// Detail menangani GET /api/input-acceptation/{no_klaim}.
//
// Bentuk layar dan isinya dikirim BERSAMA. Layar ini selalu dibuka untuk satu klaim tertentu;
// memisahkannya menjadi dua endpoint berarti dua perjalanan untuk satu layar, dan kemungkinan
// keduanya menjawab keadaan yang berbeda.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inputacceptation.ErrCallerUnknown)
		return
	}

	detail, err := h.Service.Find(
		r.Context(), active.Alias, caller, chi.URLParam(r, "no_klaim"))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK,
		toDetailResponse(h.Service.Metadata(), detail, active.Alias))
}

// Submit menangani POST /api/input-acceptation/{no_klaim}.
//
// Ia padanan tombol Submit pada Flow Action `InputAcceptation`. Muatannya divalidasi penuh
// terhadap katalog — isian read-only ditolak, kolom read-only ditolak, kunci yang tidak
// dikenal ditolak — lalu diserahkan ke penyimpanan.
//
// Selama masa paralel penyimpanan MENOLAK, dengan alasan yang terbaca pengguna: tabelnya masih
// ditulis Pega (`P-1`). Penolakan itu terjadi SESUDAH validasi, bukan sebelumnya, dan itu
// disengaja — pengguna yang mengirim muatan keliru tetap diberi tahu apa yang keliru, alih-alih
// hanya diberi tahu bahwa penyimpanannya belum tersedia.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.WriteError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.readCaller(r)
	if !known {
		h.WriteError(w, r, inputacceptation.ErrCallerUnknown)
		return
	}

	var body SubmitRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxSubmitBody)
	decoder := json.NewDecoder(r.Body)

	// Isian yang tidak dikenal DITOLAK di sini pula, bukan hanya di domain. Keduanya
	// memeriksa hal yang berbeda: yang ini menolak field JSON yang bukan bagian kontrak,
	// yang di domain menolak KUNCI isian yang bukan bagian layar.
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&body); err != nil {
		h.WriteError(w, r, inputacceptation.NewValidationError(
			[]inputacceptation.Violation{{
				Field:   "badan",
				Message: "Muatan permintaan tidak dapat dibaca sebagai JSON yang sah.",
			}}))
		return
	}

	claimID := chi.URLParam(r, "no_klaim")
	err := h.Service.Submit(
		r.Context(), active.Alias, caller, claimID, body.Values, toGridRows(body.Grids))
	if err != nil {
		h.WriteError(w, r, err)
		return
	}

	h.WriteJSON(w, r, http.StatusOK, SubmitResponse{
		ClaimID: claimID,
		Message: "Akseptasi tersimpan.",
	})
}

// readCaller membaca identitas pemanggil, atau menyatakan ia tidak terbaca.
func (h *Handler) readCaller(r *http.Request) (inputacceptation.Caller, bool) {
	if h.Caller == nil {
		return inputacceptation.Caller{}, false
	}
	caller, exists := h.Caller(r.Context())
	if !exists || strings.TrimSpace(caller.Login) == "" {
		return inputacceptation.Caller{}, false
	}
	return inputacceptation.Caller{Login: caller.Login}, true
}
