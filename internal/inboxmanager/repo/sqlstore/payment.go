package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxmanager"
)

// Panel detail Payment Klaim Akseptasi. Kuerinya di payment.sql; pemetaan isiannya ke layar
// Pega di inboxmanager/payment.go.

// PaymentDetail membaca satu nomor akseptasi yang sedang menunggu beserta riwayat klaimnya
// dan daftar lampirannya.
func (r *Repo) PaymentDetail(ctx context.Context, acceptedNo string) (inboxmanager.PaymentDetail, error) {
	no := strings.TrimSpace(acceptedNo)
	if no == "" {
		return inboxmanager.PaymentDetail{}, inboxmanager.ErrPaymentNotFound
	}

	var (
		claimID, accepted, pic, remark, claimNo, insured, email   any
		adjClaimID, receiver, paymentType, gross, value, currency any
		currencyID, share                                         any
	)
	err := r.db.QueryRowContext(ctx, query("payment_detail"), no).Scan(
		&claimID, &accepted, &pic, &remark, &claimNo, &insured, &email,
		&adjClaimID, &receiver, &paymentType, &gross, &value, &currency, &currencyID, &share,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxmanager.PaymentDetail{}, inboxmanager.ErrPaymentNotFound
	}
	if err != nil {
		return inboxmanager.PaymentDetail{}, wrapQueryError("membaca detail akseptasi", err)
	}

	detail := inboxmanager.PaymentDetail{
		AcceptedNo:      asText(accepted),
		ClaimID:         rawText(claimID),
		ClaimNo:         asText(claimNo),
		PIC:             asText(pic),
		InsuredName:     asText(insured),
		Receiver:        asText(receiver),
		PaymentTypeCode: asText(paymentType),
		GrossValue:      asText(gross),
		ClaimValue:      asText(value),
		ShareASM:        asText(share),
		RemarkPIC:       asText(remark),
		AdjustmentFound: asText(adjClaimID) != "",
		PICEmail:        asText(email),
	}

	// Mata uang ditampilkan dengan kodenya (`BrowseCurrency_RD`); bila ID-nya tidak ada di
	// master, ID itu sendiri yang ditampilkan — lebih jujur daripada kosong.
	detail.Currency = asText(currency)
	if detail.Currency == "" {
		detail.Currency = asText(currencyID)
	}

	if detail.History, err = r.paymentHistory(ctx, detail.ClaimID); err != nil {
		return inboxmanager.PaymentDetail{}, err
	}
	if detail.Documents, err = r.paymentDocuments(ctx, no); err != nil {
		return inboxmanager.PaymentDetail{}, err
	}
	return detail, nil
}

func (r *Repo) paymentHistory(ctx context.Context, claimID string) ([]inboxmanager.PaymentHistoryRow, error) {
	if strings.TrimSpace(claimID) == "" {
		return []inboxmanager.PaymentHistoryRow{}, nil
	}

	rows, err := r.db.QueryContext(ctx, query("payment_history"), claimID)
	if err != nil {
		return nil, wrapQueryError("membaca riwayat akseptasi", err)
	}
	defer rows.Close()

	history := []inboxmanager.PaymentHistoryRow{}
	for rows.Next() {
		var claimNo, accepted, decidedAt, noteManager, notePIC, status any
		if err := rows.Scan(&claimNo, &accepted, &decidedAt, &noteManager, &notePIC, &status); err != nil {
			return nil, fmt.Errorf("membaca baris riwayat akseptasi: %w", err)
		}
		code := asText(status)
		history = append(history, inboxmanager.PaymentHistoryRow{
			ClaimNo:     asText(claimNo),
			AcceptedNo:  asText(accepted),
			Date:        asTime(decidedAt),
			NoteManager: asText(noteManager),
			NotePIC:     asText(notePIC),
			StatusCode:  code,
			Status:      inboxmanager.PaymentStatusLabel(code),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat akseptasi: %w", err)
	}
	return history, nil
}

func (r *Repo) paymentDocuments(ctx context.Context, acceptedNo string) ([]inboxmanager.PaymentDocument, error) {
	rows, err := r.db.QueryContext(ctx, query("payment_documents"), acceptedNo)
	if err != nil {
		return nil, wrapQueryError("membaca lampiran akseptasi", err)
	}
	defer rows.Close()

	documents := []inboxmanager.PaymentDocument{}
	for rows.Next() {
		var name, mime, note, inputDate, imageID, hasFile any
		if err := rows.Scan(&name, &mime, &note, &inputDate, &imageID, &hasFile); err != nil {
			return nil, fmt.Errorf("membaca baris lampiran akseptasi: %w", err)
		}
		documents = append(documents, inboxmanager.PaymentDocument{
			Key:       inboxmanager.PaymentDocumentKey(len(documents) + 1),
			Name:      asText(name),
			MimeType:  asText(mime),
			Note:      asText(note),
			InputDate: asTime(inputDate),
			ImageID:   asText(imageID),
			HasFile:   asText(hasFile) == "1",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca lampiran akseptasi: %w", err)
	}
	return documents, nil
}

// asTime membaca sel tanggal; selain time.Time menjadi nil.
func asTime(value any) *time.Time {
	typed, ok := value.(time.Time)
	if !ok || typed.IsZero() {
		return nil
	}
	return &typed
}

// PaymentDocumentContent membaca isi satu lampiran lama.
func (r *Repo) PaymentDocumentContent(
	ctx context.Context, acceptedNo, key string,
) (inboxmanager.PaymentDocumentContent, error) {
	no := strings.TrimSpace(acceptedNo)
	index, ok := inboxmanager.PaymentDocumentIndex(key)
	if no == "" || !ok {
		return inboxmanager.PaymentDocumentContent{}, inboxmanager.ErrPaymentDocumentNotFound
	}

	var name, mime, imageID any
	var content []byte
	err := r.db.QueryRowContext(ctx, query("payment_document_content"), no, index-1).
		Scan(&name, &mime, &imageID, &content)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxmanager.PaymentDocumentContent{}, inboxmanager.ErrPaymentDocumentNotFound
	}
	if err != nil {
		return inboxmanager.PaymentDocumentContent{}, wrapQueryError("membaca isi lampiran akseptasi", err)
	}
	if len(content) == 0 {
		return inboxmanager.PaymentDocumentContent{}, inboxmanager.ErrPaymentDocumentNotFound
	}

	return inboxmanager.PaymentDocumentContent{
		Name:     asText(name),
		MimeType: asText(mime),
		Content:  content,
	}, nil
}

// SavePayment menuliskan SIMPAN (`SaveApprovalAkseptasiPaymentLeader_Act`).
//
// # Yang ditulis
//
//  1. Baris checker yang menunggu — `SaveApproveAkseptasiPaymentLeader_Sql`, lewat pernyataan
//     yang sama dengan keputusan tab ini (decide_payment_akseptasi). `PICNOTES` tidak ditulis:
//     Pega menulis ulang Remark Akseptasi yang di layarnya read-only, jadi nilainya tetap.
//  2. Penanda adjustment (migrasi 0017), pengganti penulisan ke case Pega.
//
// Keduanya satu transaksi. Bila nol baris checker berubah — nomornya sudah diputuskan orang
// lain — penanda tidak ditulis.
func (r *Repo) SavePayment(ctx context.Context, s inboxmanager.PaymentSave) (inboxmanager.PaymentSaveResult, error) {
	marksReady, err := r.paymentMarksReady(ctx)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, fmt.Errorf("memulai transaksi SIMPAN akseptasi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	claimIDs, err := paymentClaimIDs(ctx, tx, s.AcceptedNo)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, err
	}

	updated, err := tx.ExecContext(ctx, query("decide_payment_akseptasi"),
		s.Status, nullIfEmpty(s.Remark), s.AcceptedNo)
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, wrapQueryError("menjalankan decide_payment_akseptasi", err)
	}
	changed, err := updated.RowsAffected()
	if err != nil {
		return inboxmanager.PaymentSaveResult{}, fmt.Errorf("membaca jumlah baris terubah SIMPAN: %w", err)
	}

	result := inboxmanager.PaymentSaveResult{Changed: int(changed)}
	if changed > 0 && marksReady {
		for _, claimID := range claimIDs {
			marked, err := tx.ExecContext(ctx, query("payment_mark_adjustment"),
				nullIfEmpty(s.AdjustmentFlag()), s.Status, claimID, s.AcceptedNo)
			if err != nil {
				return inboxmanager.PaymentSaveResult{}, wrapQueryError("menjalankan payment_mark_adjustment", err)
			}
			if n, err := marked.RowsAffected(); err == nil && n > 0 {
				result.AdjustmentMarked = true
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return inboxmanager.PaymentSaveResult{}, fmt.Errorf("menyimpan SIMPAN akseptasi: %w", err)
	}
	return result, nil
}

// paymentMarksReady menyatakan kolom penanda migrasi 0017 sudah ada. Kolom yang belum ada
// BUKAN galat — SIMPAN tetap berjalan tanpa penanda.
func (r *Repo) paymentMarksReady(ctx context.Context) (bool, error) {
	rows, err := r.db.QueryContext(ctx, query("payment_marks_probe"))
	if isMissingColumn(err) {
		return false, nil
	}
	if err != nil {
		return false, wrapQueryError("memeriksa kolom penanda adjustment", err)
	}
	_ = rows.Close()
	return true, nil
}

func paymentClaimIDs(ctx context.Context, tx *sql.Tx, acceptedNo string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, query("payment_claim_ids"), acceptedNo)
	if err != nil {
		return nil, wrapQueryError("membaca klaim nomor akseptasi", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id any
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("membaca klaim nomor akseptasi: %w", err)
		}
		if text := rawText(id); strings.TrimSpace(text) != "" {
			ids = append(ids, text)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca klaim nomor akseptasi: %w", err)
	}
	return ids, nil
}

// rawText membaca sel teks TANPA memangkas spasi — untuk nilai yang dikirim balik sebagai
// argumen pembanding kolom yang tidak di-TRIM (CLAIMID), supaya index-nya tetap terpakai.
func rawText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}
