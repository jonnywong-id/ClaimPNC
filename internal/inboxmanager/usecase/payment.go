package usecase

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"claim-pnc/internal/inboxmanager"
)

// Panel detail Payment Klaim Akseptasi — tombol DETAILS, Dokumen, dan SIMPAN. Asal dan
// pemetaannya di inboxmanager/payment.go.

// paymentCaller memastikan pemanggil boleh membuka tab Payment Klaim Akseptasi.
func (s *Service) paymentCaller(
	ctx context.Context,
	portalAlias string,
	caller inboxmanager.Caller,
) (inboxmanager.Caller, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return inboxmanager.Caller{}, err
	}
	if resolved.Login == "" {
		return inboxmanager.Caller{}, inboxmanager.ErrCallerUnknown
	}
	tab, _ := inboxmanager.FindTab(inboxmanager.TabPaymentAkseptasi)
	if !inboxmanager.CanSee(resolved, tab) {
		return inboxmanager.Caller{}, inboxmanager.ErrTabNotAllowed
	}
	return resolved, nil
}

// PaymentDetail membaca panel detail satu nomor akseptasi yang sedang menunggu — tombol
// DETAILS (`GetDataAkseptasiKlaimKasir`).
func (s *Service) PaymentDetail(
	ctx context.Context,
	portalAlias, acceptedNo string,
	caller inboxmanager.Caller,
) (inboxmanager.PaymentDetail, error) {
	if _, err := s.paymentCaller(ctx, portalAlias, caller); err != nil {
		return inboxmanager.PaymentDetail{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxmanager.PaymentDetail{}, err
	}

	detail, err := repo.PaymentDetail(ctx, acceptedNo)
	if err != nil {
		return inboxmanager.PaymentDetail{}, err
	}
	detail.PaymentType = s.paymentTypeName(detail.PaymentTypeCode)
	return detail, nil
}

// paymentTypeName menamai kode Tipe Pembayaran lewat aturan modul Registrasi. Tanpa aturan,
// atau untuk kode yang tidak dikenalnya, kodenya sendiri yang ditampilkan.
func (s *Service) paymentTypeName(code string) string {
	clean := strings.TrimSpace(code)
	if clean == "" || s.paymentRules.PaymentTypeName == nil {
		return clean
	}
	if name := strings.TrimSpace(s.paymentRules.PaymentTypeName(clean)); name != "" {
		return name
	}
	return clean
}

// PaymentDocumentLink mengembalikan alamat lampiran yang tersimpan di layanan penyimpanan,
// diperpanjang bila sudah kedaluwarsa.
//
// Lampirannya dicari di daftar nomor akseptasi itu — kunci dari nomor lain tidak dapat
// membuka apa pun.
func (s *Service) PaymentDocumentLink(
	ctx context.Context,
	portalAlias, acceptedNo, key string,
	caller inboxmanager.Caller,
) (string, error) {
	resolved, err := s.paymentCaller(ctx, portalAlias, caller)
	if err != nil {
		return "", err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return "", err
	}
	detail, err := repo.PaymentDetail(ctx, acceptedNo)
	if err != nil {
		return "", err
	}

	for _, document := range detail.Documents {
		if document.Key != strings.TrimSpace(key) {
			continue
		}
		if !document.Linked() {
			return "", inboxmanager.ErrPaymentDocumentNotFound
		}
		if s.linker == nil {
			return "", inboxmanager.ErrDocumentLinkUnavailable
		}
		url, err := s.linker.Link(ctx, portalAlias, document.ImageID, resolved.Login)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(url) == "" {
			return "", inboxmanager.ErrPaymentDocumentNotFound
		}
		return url, nil
	}
	return "", inboxmanager.ErrPaymentDocumentNotFound
}

// PaymentDocument membaca isi satu lampiran lama (kolom ATTACHFILE).
func (s *Service) PaymentDocument(
	ctx context.Context,
	portalAlias, acceptedNo, key string,
	caller inboxmanager.Caller,
) (inboxmanager.PaymentDocumentContent, error) {
	if _, err := s.paymentCaller(ctx, portalAlias, caller); err != nil {
		return inboxmanager.PaymentDocumentContent{}, err
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxmanager.PaymentDocumentContent{}, err
	}
	return repo.PaymentDocumentContent(ctx, acceptedNo, key)
}

// SavePaymentInput adalah permintaan SIMPAN mentah dari layar.
type SavePaymentInput struct {
	AcceptedNo string
	Status     string
	Remark     string
}

// SavePayment menuliskan SIMPAN (`SaveApprovalAkseptasiPaymentLeader_Act`), lalu memberi
// tahu PIC lewat surel.
//
// # Surel yang gagal tidak menggagalkan SIMPAN
//
// Keputusannya sudah tertulis dan tidak dibatalkan karena server surel tidak menjawab. Di
// Pega pun pengiriman surel adalah langkah terakhir, setelah commit. Kegagalannya dicatat di
// log dan dinyatakan di hasil (Notified = false), sehingga layar dapat mengatakannya.
func (s *Service) SavePayment(
	ctx context.Context,
	portalAlias string,
	input SavePaymentInput,
	caller inboxmanager.Caller,
) (inboxmanager.PaymentSaveResult, error) {
	resolved, err := s.resolveCaller(ctx, portalAlias, caller)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, err
	}
	save, err := inboxmanager.NewPaymentSave(input.AcceptedNo, input.Status, input.Remark, resolved)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, err
	}

	// Bahan surel dibaca SEBELUM menulis: setelah SIMPAN berstatus selain "Sedang Proses
	// Komite", nomornya tidak lagi menunggu dan detailnya tidak terbaca lagi.
	detail, detailErr := repo.PaymentDetail(ctx, save.AcceptedNo)

	result, err := repo.SavePayment(ctx, save)
	if err != nil {
		s.logPaymentSave(portalAlias, save, inboxmanager.PaymentSaveResult{}, err, nil)
		return inboxmanager.PaymentSaveResult{}, err
	}
	if result.Changed == 0 {
		s.logPaymentSave(portalAlias, save, result, nil, nil)
		return result, nil
	}

	var notifyErr error
	switch {
	case detailErr != nil:
		notifyErr = detailErr
	case s.notifier == nil:
		notifyErr = errNotifierMissing
	default:
		notifyErr = s.notifier.NotifyPaymentChecked(ctx, inboxmanager.PaymentNotice{
			To:          detail.PICEmail,
			ClaimNo:     detail.ClaimNo,
			InsuredName: detail.InsuredName,
			AcceptedNo:  save.AcceptedNo,
			Status:      save.Status,
			Remark:      save.Remark,
			DecidedBy:   save.Caller.Login,
		})
	}
	result.Notified = notifyErr == nil

	s.logPaymentSave(portalAlias, save, result, nil, notifyErr)
	return result, nil
}

// errNotifierMissing hanya masuk log: pengirim surel tidak dirakit (SMTP belum diisi).
var errNotifierMissing = errors.New("pengirim surel belum dikonfigurasi")

// logPaymentSave mencatat setiap SIMPAN — jejak audit yang sama wajibnya dengan keputusan
// antrean (lihat logDecision). Catatan atasan tidak ikut dicatat; ia sudah tersimpan di
// kolom NOTEAPPROVE.
func (s *Service) logPaymentSave(
	portalAlias string,
	save inboxmanager.PaymentSave,
	result inboxmanager.PaymentSaveResult,
	failure, notifyErr error,
) {
	if s.logger == nil {
		return
	}
	attrs := []any{
		slog.String("modul", "inbox-manager"),
		slog.String("portal", portalAlias),
		slog.String("login", save.Caller.Login),
		slog.String("tab", inboxmanager.TabPaymentAkseptasi),
		slog.String("no_akseptasi", save.AcceptedNo),
		slog.String("status", save.Status),
		slog.Int("berubah", result.Changed),
		slog.Bool("penanda_adjustment", result.AdjustmentMarked),
		slog.Bool("surel_terkirim", result.Notified),
	}

	switch {
	case failure != nil:
		s.logger.ErrorContext(context.Background(), "SIMPAN payment akseptasi GAGAL",
			append(attrs, slog.String("galat", failure.Error()))...)
	case result.Changed == 0:
		s.logger.WarnContext(context.Background(),
			"SIMPAN payment akseptasi tidak mengubah baris: nomor sudah diputuskan lebih dulu", attrs...)
	case notifyErr != nil:
		s.logger.WarnContext(context.Background(),
			"SIMPAN payment akseptasi ditulis, tetapi surel ke PIC tidak terkirim",
			append(attrs, slog.String("galat_surel", notifyErr.Error()))...)
	default:
		s.logger.InfoContext(context.Background(), "SIMPAN payment akseptasi ditulis", attrs...)
	}
}
