package inboxmanagerhttp

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/usecase"
)

// Panel detail Payment Klaim Akseptasi — DETAILS, Dokumen, dan SIMPAN. Asal dan pemetaannya
// di inboxmanager/payment.go.

// PaymentDetailResponse adalah jawaban GET /api/inbox-manager/payment-akseptasi/{noakseptasi}.
type PaymentDetailResponse struct {
	AcceptedNo string `json:"no_akseptasi"`
	ClaimNo    string `json:"no_klaim"`
	PIC        string `json:"pic"`

	// Detail Penerima Akseptasi.
	InsuredName     string `json:"nama_tertanggung"`
	Receiver        string `json:"penerima_klaim"`
	PaymentTypeCode string `json:"kode_tipe_pembayaran"`
	PaymentType     string `json:"tipe_pembayaran"`
	GrossValue      string `json:"gross_value"`
	ClaimValue      string `json:"nilai_klaim"`
	Currency        string `json:"mata_uang"`
	ShareASM        string `json:"share_asm"`
	RemarkPIC       string `json:"remark_akseptasi"`
	AdjustmentFound bool   `json:"adjustment_ditemukan"`

	// StatusOptions adalah pilihan "Status Approval", berurutan seperti di Pega.
	StatusOptions []FilterOptionDTO `json:"pilihan_status"`

	History   []PaymentHistoryDTO  `json:"riwayat"`
	Documents []PaymentDocumentDTO `json:"dokumen"`
}

// PaymentHistoryDTO adalah satu baris grid "History Payment Akseptasi Manager".
type PaymentHistoryDTO struct {
	ClaimNo     string     `json:"no_klaim"`
	AcceptedNo  string     `json:"no_akseptasi"`
	Date        *time.Time `json:"tanggal"`
	NoteManager string     `json:"note_manager"`
	NotePIC     string     `json:"note_pic"`
	StatusCode  string     `json:"kode_status"`
	Status      string     `json:"status"`
}

// PaymentDocumentDTO adalah satu lampiran tombol Dokumen.
type PaymentDocumentDTO struct {
	Key       string     `json:"kunci"`
	Name      string     `json:"nama"`
	MimeType  string     `json:"jenis"`
	Note      string     `json:"catatan"`
	InputDate *time.Time `json:"tanggal_input"`

	// Linked: dibuka lewat tautan layanan penyimpanan (rute .../tautan); selain itu, bila
	// Openable, berkasnya diunduh langsung.
	Linked   bool `json:"tertaut"`
	Openable bool `json:"dapat_dibuka"`
}

// PaymentSaveRequest adalah badan POST .../simpan.
type PaymentSaveRequest struct {
	Status string `json:"status"`
	Remark string `json:"catatan"`
}

// PaymentSaveResponse adalah jawaban SIMPAN. Angka dan benderanya dikirim apa adanya; layar
// yang menyusun kalimatnya.
type PaymentSaveResponse struct {
	Changed          int  `json:"berubah"`
	AdjustmentMarked bool `json:"penanda_adjustment"`
	Notified         bool `json:"surel_terkirim"`
}

// PaymentLinkResponse adalah jawaban GET .../dokumen/{kunci}/tautan.
type PaymentLinkResponse struct {
	URL string `json:"url"`
}

func paymentDetailResponse(d inboxmanager.PaymentDetail) PaymentDetailResponse {
	options := make([]FilterOptionDTO, 0, len(inboxmanager.PaymentStatusOptions))
	for _, option := range inboxmanager.PaymentStatusOptions {
		options = append(options, FilterOptionDTO{Value: option.Value, Label: option.Label})
	}

	history := make([]PaymentHistoryDTO, 0, len(d.History))
	for _, h := range d.History {
		history = append(history, PaymentHistoryDTO{
			ClaimNo:     h.ClaimNo,
			AcceptedNo:  h.AcceptedNo,
			Date:        h.Date,
			NoteManager: h.NoteManager,
			NotePIC:     h.NotePIC,
			StatusCode:  h.StatusCode,
			Status:      h.Status,
		})
	}

	documents := make([]PaymentDocumentDTO, 0, len(d.Documents))
	for _, doc := range d.Documents {
		documents = append(documents, PaymentDocumentDTO{
			Key:       doc.Key,
			Name:      doc.Name,
			MimeType:  doc.MimeType,
			Note:      doc.Note,
			InputDate: doc.InputDate,
			Linked:    doc.Linked(),
			Openable:  doc.Openable(),
		})
	}

	return PaymentDetailResponse{
		AcceptedNo:      d.AcceptedNo,
		ClaimNo:         d.ClaimNo,
		PIC:             d.PIC,
		InsuredName:     d.InsuredName,
		Receiver:        d.Receiver,
		PaymentTypeCode: d.PaymentTypeCode,
		PaymentType:     d.PaymentType,
		GrossValue:      d.GrossValue,
		ClaimValue:      d.ClaimValue,
		Currency:        d.Currency,
		ShareASM:        d.ShareASM,
		RemarkPIC:       d.RemarkPIC,
		AdjustmentFound: d.AdjustmentFound,
		StatusOptions:   options,
		History:         history,
		Documents:       documents,
	}
}

// PaymentDetail menangani GET /api/inbox-manager/payment-akseptasi/{noakseptasi} — DETAILS.
func (h *Handler) PaymentDetail(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}
	detail, err := h.service.PaymentDetail(r.Context(), active.Alias,
		chi.URLParam(r, "noakseptasi"), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, paymentDetailResponse(detail))
}

// PaymentDocumentLink menangani GET .../dokumen/{kunci}/tautan — lampiran di layanan
// penyimpanan, alamatnya diperpanjang bila kedaluwarsa.
func (h *Handler) PaymentDocumentLink(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}
	url, err := h.service.PaymentDocumentLink(r.Context(), active.Alias,
		chi.URLParam(r, "noakseptasi"), chi.URLParam(r, "kunci"), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, PaymentLinkResponse{URL: url})
}

// inlineTypes adalah jenis berkas yang aman digambar peramban. Selebihnya diunduh: lampiran
// disajikan dari asal yang sama dengan aplikasi, dan berkas HTML/SVG yang digambar dapat
// menjalankan skrip atas nama petugas yang sedang masuk (preseden inboxrclpucl).
var inlineTypes = map[string]bool{
	"application/pdf": true,
	"image/png":       true,
	"image/jpeg":      true,
	"image/gif":       true,
}

// PaymentDocument menangani GET .../dokumen/{kunci} — isi lampiran lama.
func (h *Handler) PaymentDocument(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}
	document, err := h.service.PaymentDocument(r.Context(), active.Alias,
		chi.URLParam(r, "noakseptasi"), chi.URLParam(r, "kunci"), caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	kind := strings.ToLower(strings.TrimSpace(document.MimeType))
	disposition := "attachment"
	if inlineTypes[kind] {
		disposition = "inline"
	} else {
		kind = "application/octet-stream"
	}
	w.Header().Set("Content-Type", kind)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", disposition+`; filename="`+safeFilename(document.Name)+`"`)
	// Dokumen milik nasabah tidak boleh disimpan perantara mana pun.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(document.Content)
}

// safeFilename membuang tanda kutip dan pergantian baris dari nama berkas — nama itu berasal
// dari data, dan keduanya dapat menyudahi header lebih awal.
func safeFilename(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = "dokumen"
	}
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(clean)
}

// SavePayment menangani POST .../simpan — SIMPAN.
func (h *Handler) SavePayment(w http.ResponseWriter, r *http.Request) {
	active, caller, ready := h.prepare(w, r)
	if !ready {
		return
	}

	var body PaymentSaveRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeError(w, r, inboxmanager.NewValidationError([]inboxmanager.Violation{{
			Field:   inboxmanager.FieldVerdict,
			Message: "Badan permintaan tidak dapat dibaca.",
		}}))
		return
	}

	result, err := h.service.SavePayment(r.Context(), active.Alias, usecase.SavePaymentInput{
		AcceptedNo: chi.URLParam(r, "noakseptasi"),
		Status:     body.Status,
		Remark:     body.Remark,
	}, caller)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, r, http.StatusOK, PaymentSaveResponse{
		Changed:          result.Changed,
		AdjustmentMarked: result.AdjustmentMarked,
		Notified:         result.Notified,
	})
}
