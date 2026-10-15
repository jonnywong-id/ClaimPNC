package sqlstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

var paymentDetailColumns = []string{
	"CLAIM_ID", "ACCEPTED_NO", "PIC", "REMARK_PIC", "CLAIM_NO", "INSURED_NAME", "PIC_EMAIL",
	"ADJ_CLAIM_ID", "RECEIVER", "PAYMENT_TYPE", "GROSS_VALUE", "CLAIM_VALUE", "CURRENCY",
	"CURRENCY_ID", "SHARE_ASM",
}

func TestPaymentDetail(t *testing.T) {
	repo, mock := newMock(t)
	decided := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	input := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(exact("payment_detail")).WithArgs("AKS-1").WillReturnRows(
		sqlmock.NewRows(paymentDetailColumns).AddRow(
			"CLAIM-1 ", "AKS-1", "ANDIKA", "catatan pic", "C-1", "Tertanggung", "pic@contoh.test",
			"CLAIM-1 ", "Penerima", "2", "1250000.5", "1000000", nil, "10026", "80"))
	mock.ExpectQuery(exact("payment_history")).WithArgs("CLAIM-1 ").WillReturnRows(
		sqlmock.NewRows([]string{"CLAIM_NO", "ACCEPTED_NO", "DECIDED_AT", "NOTE_MANAGER", "NOTE_PIC", "STATUS"}).
			AddRow("C-1", "AKS-0", decided, "catatan atasan", "catatan pic", "2"))
	mock.ExpectQuery(exact("payment_documents")).WithArgs("AKS-1").WillReturnRows(
		sqlmock.NewRows([]string{"DOCUMENT_NAME", "MIME_TYPE", "NOTE", "INPUT_DATE", "IMAGE_ID", "HAS_FILE"}).
			AddRow("baru.pdf", "application/pdf", "", input, "IMG-1", "0").
			AddRow("lama.pdf", "application/pdf", "lama", nil, nil, "1"))

	detail, err := repo.PaymentDetail(context.Background(), " AKS-1 ")
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	require.Equal(t, "CLAIM-1 ", detail.ClaimID, "CLAIMID dikirim balik tanpa dipangkas")
	require.Equal(t, "1250000.5", detail.GrossValue)
	require.Equal(t, "10026", detail.Currency, "ID mata uang bila kodenya tidak ada di master")
	require.True(t, detail.AdjustmentFound)
	require.Equal(t, "pic@contoh.test", detail.PICEmail)
	require.Empty(t, detail.PaymentType, "nama tipe diisi usecase lewat aturan Registrasi")

	require.Equal(t, []inboxmanager.PaymentHistoryRow{{
		ClaimNo: "C-1", AcceptedNo: "AKS-0", Date: &decided, NoteManager: "catatan atasan",
		NotePIC: "catatan pic", StatusCode: "2", Status: "Tidak Setuju",
	}}, detail.History)

	require.Len(t, detail.Documents, 2)
	require.Equal(t, "1", detail.Documents[0].Key)
	require.True(t, detail.Documents[0].Linked())
	require.Equal(t, &input, detail.Documents[0].InputDate)
	require.Equal(t, "2", detail.Documents[1].Key)
	require.True(t, detail.Documents[1].HasFile)
	require.Nil(t, detail.Documents[1].InputDate)
}

func TestPaymentDetailTanpaAdjustment(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_detail")).WithArgs("AKS-1").WillReturnRows(
		sqlmock.NewRows(paymentDetailColumns).AddRow(
			"CLAIM-1", "AKS-1", "ANDIKA", nil, "C-1", nil, nil,
			nil, nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectQuery(exact("payment_history")).WithArgs("CLAIM-1").
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM_NO", "ACCEPTED_NO", "DECIDED_AT", "NOTE_MANAGER", "NOTE_PIC", "STATUS"}))
	mock.ExpectQuery(exact("payment_documents")).WithArgs("AKS-1").
		WillReturnRows(sqlmock.NewRows([]string{"DOCUMENT_NAME", "MIME_TYPE", "NOTE", "INPUT_DATE", "IMAGE_ID", "HAS_FILE"}))

	detail, err := repo.PaymentDetail(context.Background(), "AKS-1")
	require.NoError(t, err)
	require.False(t, detail.AdjustmentFound)
	require.Empty(t, detail.Currency)
	require.Empty(t, detail.History)
	require.Empty(t, detail.Documents)
}

func TestPaymentDetailGalat(t *testing.T) {
	ctx := context.Background()

	repo, _ := newMock(t)
	_, err := repo.PaymentDetail(ctx, " ")
	require.ErrorIs(t, err, inboxmanager.ErrPaymentNotFound)

	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_detail")).WillReturnRows(sqlmock.NewRows(paymentDetailColumns))
	_, err = repo.PaymentDetail(ctx, "AKS-1")
	require.ErrorIs(t, err, inboxmanager.ErrPaymentNotFound)

	repo, mock = newMock(t)
	mock.ExpectQuery(exact("payment_detail")).WillReturnError(errors.New("ORA-04063: view has errors"))
	_, err = repo.PaymentDetail(ctx, "AKS-1")
	require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)
}

func TestPaymentDocumentContent(t *testing.T) {
	ctx := context.Background()
	columns := []string{"DOCUMENT_NAME", "MIME_TYPE", "IMAGE_ID", "CONTENT"}

	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_document_content")).WithArgs("AKS-1", 1).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("lama.pdf", "application/pdf", nil, []byte("%PDF")))
	content, err := repo.PaymentDocumentContent(ctx, "AKS-1", "2")
	require.NoError(t, err)
	require.Equal(t, inboxmanager.PaymentDocumentContent{
		Name: "lama.pdf", MimeType: "application/pdf", Content: []byte("%PDF"),
	}, content)

	// Kunci yang bukan nomor urut tidak menyentuh basis data.
	repo, _ = newMock(t)
	_, err = repo.PaymentDocumentContent(ctx, "AKS-1", "../1")
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)

	// Lampiran baru (tanpa berkas di ATTACHFILE) dan kunci di luar daftar.
	repo, mock = newMock(t)
	mock.ExpectQuery(exact("payment_document_content")).WithArgs("AKS-1", 0).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("baru.pdf", "application/pdf", "IMG-1", nil))
	mock.ExpectQuery(exact("payment_document_content")).WithArgs("AKS-1", 8).
		WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.PaymentDocumentContent(ctx, "AKS-1", "1")
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)
	_, err = repo.PaymentDocumentContent(ctx, "AKS-1", "9")
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)
}

func paymentSave(status string) inboxmanager.PaymentSave {
	return inboxmanager.PaymentSave{
		AcceptedNo: "AKS-1", Status: status, Remark: "nilai belum sesuai",
		Caller: inboxmanager.Caller{Login: "MGR"},
	}
}

func TestSavePaymentMenulisCheckerLaluPenanda(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_marks_probe")).
		WillReturnRows(sqlmock.NewRows([]string{"FLAGATASAN", "STSPENERIMAKLAIM"}))
	mock.ExpectBegin()
	mock.ExpectQuery(exact("payment_claim_ids")).WithArgs("AKS-1").
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("CLAIM-1 "))
	mock.ExpectExec(exact("decide_payment_akseptasi")).
		WithArgs("2", "nilai belum sesuai", "AKS-1").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(exact("payment_mark_adjustment")).
		WithArgs("1", "2", "CLAIM-1 ", "AKS-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := repo.SavePayment(context.Background(), paymentSave("2"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, inboxmanager.PaymentSaveResult{Changed: 2, AdjustmentMarked: true}, result)
}

// Sebelum DBA menjalankan migrasi 0017, SIMPAN tetap berjalan tanpa penanda.
func TestSavePaymentTanpaKolomPenanda(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_marks_probe")).
		WillReturnError(errors.New(`ORA-00904: "STSPENERIMAKLAIM": invalid identifier`))
	mock.ExpectBegin()
	mock.ExpectQuery(exact("payment_claim_ids")).
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("CLAIM-1"))
	mock.ExpectExec(exact("decide_payment_akseptasi")).
		WithArgs("0", nil, "AKS-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	save := paymentSave("0")
	save.Remark = ""
	result, err := repo.SavePayment(context.Background(), save)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, inboxmanager.PaymentSaveResult{Changed: 1}, result)
}

// Nomor yang sudah diputuskan orang lain tidak ditandai ulang.
func TestSavePaymentBasiTidakMenandai(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_marks_probe")).
		WillReturnRows(sqlmock.NewRows([]string{"FLAGATASAN", "STSPENERIMAKLAIM"}))
	mock.ExpectBegin()
	mock.ExpectQuery(exact("payment_claim_ids")).WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}))
	mock.ExpectExec(exact("decide_payment_akseptasi")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	result, err := repo.SavePayment(context.Background(), paymentSave("2"))
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
	require.Equal(t, inboxmanager.PaymentSaveResult{}, result)
}

func TestSavePaymentGalatMembatalkan(t *testing.T) {
	ctx := context.Background()
	probe := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(exact("payment_marks_probe")).
			WillReturnRows(sqlmock.NewRows([]string{"FLAGATASAN", "STSPENERIMAKLAIM"}))
	}

	repo, mock := newMock(t)
	mock.ExpectQuery(exact("payment_marks_probe")).WillReturnError(errors.New("ORA-03113"))
	_, err := repo.SavePayment(ctx, paymentSave("2"))
	require.ErrorContains(t, err, "ORA-03113")

	repo, mock = newMock(t)
	probe(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(exact("payment_claim_ids")).
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM_ID"}).AddRow("CLAIM-1"))
	mock.ExpectExec(exact("decide_payment_akseptasi")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("payment_mark_adjustment")).WillReturnError(errors.New("ORA-00054"))
	mock.ExpectRollback()
	_, err = repo.SavePayment(ctx, paymentSave("2"))
	require.ErrorContains(t, err, "ORA-00054")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMock(t)
	probe(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(exact("payment_claim_ids")).WillReturnError(errors.New("ORA-01013"))
	mock.ExpectRollback()
	_, err = repo.SavePayment(ctx, paymentSave("2"))
	require.ErrorContains(t, err, "ORA-01013")
	require.NoError(t, mock.ExpectationsWereMet())
}
