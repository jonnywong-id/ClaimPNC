package daftardetailtipedokumenhttp

import (
	"net/http"

	"claim-pnc/internal/platform/httpjson"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxRequestBody membatasi besar badan permintaan yang dibaca.
//
// Badan JSON modul ini memuat enam isian pendek ditambah satu senarai aturan bisnis.
// 256 KiB sudah jauh lebih dari cukup — satu rincian dokumen dengan seratus lini bisnis
// pun tidak mendekati seperempatnya.
//
// Batasnya lebih besar daripada modul master berisian tunggal justru KARENA senarai itu:
// modul yang hanya punya isian tunggal tidak punya cara menghasilkan badan besar, modul
// ini punya. Angkanya disamakan dengan modul Daftar Detail Dokumen Travel, yang bentuk
// permintaannya paling mirip.
//
// Ini BUKAN validasi isian. Yang dijaga di sini adalah sumber daya server, bukan aturan
// bisnis, dan keduanya berbeda: yang satu menolak permintaan yang tidak wajar, yang lain
// menolak isian yang tidak sah.
const maxRequestBody = 256 << 10

// References menangani GET /api/master/detail-tipe-dokumen/pilihan.
//
// Menggantikan keempat autocomplete pada form Pega sekaligus: tipe dokumen
// (`ASM-FW-GCNMFW-Int-V_LST_DOC_TYPE`), penyebab kerugian
// (`ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`), objek dokumen
// (`ASM-FW-GCNMFW-Int-V_LST_DOC_OBJ`), dan bisnis (`ASM-FW-GISFW-Int-BUSINESS`).
//
// Jalurnya berada DI BAWAH sub-rute modul ini, berbeda dari modul Daftar Detail Dokumen
// Travel yang menaruh daftar pilihannya sejajar. Sebabnya: yang dikembalikan di sini
// bukan salah satu master melainkan GABUNGAN keempatnya dalam bentuk yang hanya berarti
// bagi form ini. Ia tidak menyiratkan kepemilikan tabel mana pun — dan tabel aslinya
// tetap dilayani modul pemiliknya masing-masing.
func (h *Handler) References(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeModuleError(w, r, portal.ErrNotStated)
		return
	}

	references, err := h.Service.References(r.Context(), active.Alias)
	if err != nil {
		h.writeModuleError(w, r, err)
		return
	}
	h.WriteResponse(w, r, http.StatusOK, toReferenceResponse(references, active.Alias))
}

// identity membaca identitas pemanggil; kosong bila tidak terbaca.
//
// Kosong TIDAK menggagalkan permintaan — lihat alasannya pada `usecase.editor`.
func (h *Handler) identity(r *http.Request) string {
	if h.Caller == nil {
		return ""
	}
	who, exists := h.Caller(r.Context())
	if !exists {
		return ""
	}
	return who.Identity
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
