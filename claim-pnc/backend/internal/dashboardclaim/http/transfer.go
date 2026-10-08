package dashboardclaimhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/usecase"
	"claim-pnc/internal/portal"
	portalhttp "claim-pnc/internal/portal/http"
)

// maxTransferBodyBytes membatasi badan permintaan Transfer.
//
// Formnya lima isian pendek; batas ini jauh di atasnya dan masih mencegah satu permintaan
// menahan memori. Badan yang lebih besar ditolak sebelum diurai, bukan sesudahnya.
const maxTransferBodyBytes = 4 << 10

// Caller adalah identitas pemanggil yang dibutuhkan modul ini.
//
// Dideklarasikan DI SINI, sesempit yang dibutuhkan, supaya modul ini tidak mengimpor modul
// auth. Jembatannya dipasang di cmd — keduanya karena itu tetap tidak saling mengenal.
type Caller struct {
	Login string
	Name  string
}

// transferRequestDTO adalah badan permintaan POST /dashboard-claim/transfer.
type transferRequestDTO struct {
	Lingkup      string `json:"lingkup"`
	KlaimID      string `json:"klaim_id"`
	NomorKlaim   string `json:"nomor_klaim"`
	UserIDLama   string `json:"user_id_lama"`
	UserIDBaru   string `json:"user_id_baru"`
	TipePengguna string `json:"tipe_pengguna"`
	Alasan       string `json:"alasan"`
}

// transferResponseDTO adalah jawabannya.
type transferResponseDTO struct {
	Permintaan transferDTO `json:"permintaan"`
	Portal     string      `json:"portal"`

	// PelaksanaBelumAda menyatakan bahwa permintaan TERCATAT tetapi belum akan dijalankan.
	//
	// Ia dikirim ke layar, bukan disembunyikan: pengguna yang menekan Transfer perlu tahu
	// bahwa penugasannya BELUM berpindah — barisnya memang masih ada di daftar, dan tanpa
	// keterangan ini ia akan menekannya lagi.
	PelaksanaBelumAda bool `json:"pelaksana_belum_ada"`
}

// transferDTO adalah satu permintaan transfer.
type transferDTO struct {
	ID           string `json:"id"`
	Lingkup      string `json:"lingkup"`
	KlaimID      string `json:"klaim_id,omitempty"`
	NomorKlaim   string `json:"nomor_klaim,omitempty"`
	UserIDLama   string `json:"user_id_lama,omitempty"`
	UserIDBaru   string `json:"user_id_baru"`
	TipePengguna string `json:"tipe_pengguna,omitempty"`
	Alasan       string `json:"alasan,omitempty"`
	Status       string `json:"status"`
	Pemohon      string `json:"pemohon"`
	PemohonNama  string `json:"pemohon_nama,omitempty"`
	Pada         string `json:"pada"`
}

// Transfer menjawab POST /dashboard-claim/transfer.
//
// Menggantikan tombol "Transfer" pada setiap baris dan "Transfer All Case By UserID" pada
// layar lama — keduanya di satu rute, dibedakan field `lingkup`.
//
// # Ia MENCATAT PERMINTAAN, bukan memindahkan penugasan
//
// `P-1` menetapkan `DATAPEGA.PC_ASSIGN_WORKLIST` ditulis Pega selama masa paralel. Yang
// tercatat di sini adalah permintaan beserta pemohonnya; pelaksanaannya tetap di Pega.
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	active, exists := portalhttp.ActivePortalFrom(r.Context())
	if !exists {
		h.writeError(w, r, portal.ErrNotStated)
		return
	}

	caller, known := h.caller(r.Context())
	if !known || caller.Login == "" {
		// Tidak boleh terjadi: rute dilindungi sesi. Bila terjadi, mencatat permintaan tanpa
		// pemohon menghapus satu-satunya kontrol pengimbang yang tersisa (`D-59`).
		h.writeResponse(w, r, http.StatusUnauthorized, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Identitas pemohon tidak terbaca dari sesi.",
		})
		return
	}

	var body transferRequestDTO
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxTransferBodyBytes))
	if err := decoder.Decode(&body); err != nil {
		h.writeResponse(w, r, http.StatusBadRequest, ErrorResponse{
			Code:    CodeBadRequest,
			Message: "Badan permintaan tidak dapat dibaca.",
		})
		return
	}

	scope, valid := dashboardclaim.ParseTransferScope(body.Lingkup)
	if !valid {
		// Dijawab sebagai pelanggaran validasi, bukan 400: bentuk permintaannya benar, yang
		// salah isinya — dan frontend menempelkan pesannya ke isian yang bersangkutan.
		h.writeError(w, r, dashboardclaim.NewValidationError([]dashboardclaim.Violation{{
			Field:   dashboardclaim.FieldScope,
			Message: "Lingkup transfer tidak dikenal.",
		}}))
		return
	}

	request, err := h.service.Transfer(r.Context(), usecase.TransferCommand{
		PortalAlias: active.Alias,
		Caller:      usecase.TransferCaller{Login: caller.Login, Name: caller.Name},
		Request: dashboardclaim.TransferCommand{
			Scope:        scope,
			ClaimID:      body.KlaimID,
			ClaimNumber:  body.NomorKlaim,
			FromOperator: body.UserIDLama,
			ToOperator:   body.UserIDBaru,
			UserType:     body.TipePengguna,
			Reason:       body.Alasan,
		},
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	h.writeResponse(w, r, http.StatusCreated, transferResponseDTO{
		Permintaan:        adaptTransfer(request, h.location),
		Portal:            active.Alias,
		PelaksanaBelumAda: true,
	})
}

// adaptTransfer memetakan satu permintaan menjadi DTO.
func adaptTransfer(request dashboardclaim.TransferRequest, loc *time.Location) transferDTO {
	return transferDTO{
		ID:           request.ID,
		Lingkup:      string(request.Scope),
		KlaimID:      request.ClaimID,
		NomorKlaim:   request.ClaimNumber,
		UserIDLama:   request.FromOperator,
		UserIDBaru:   request.ToOperator,
		TipePengguna: request.UserType,
		Alasan:       request.Reason,
		Status:       string(request.Status),
		Pemohon:      request.RequestedBy,
		PemohonNama:  request.RequestedByName,
		Pada:         formatDateTime(request.RequestedAt, loc),
	}
}

// formatDateTime memformat stempel waktu jejak.
//
// Berbeda dari formatDate: jejak permintaan menyebutkan JAM, karena dua permintaan pada hari
// yang sama harus dapat diurutkan oleh orang yang membacanya.
func formatDateTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}

// callerBridge adalah bentuk jembatan identitas yang disuntik cmd.
type callerBridge func(ctx context.Context) (Caller, bool)
