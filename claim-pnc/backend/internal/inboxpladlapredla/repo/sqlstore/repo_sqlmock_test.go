package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
)

// Uji di berkas ini menjalankan Repo terhadap basis data tiruan: kueri yang ditembak,
// argumennya, pemetaan baris ke struct, dan setiap cabang galat.

const mockKey = "ASM-FW-GCNMFW-WORK PNC-1001"

func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func sqlOf(name string) string { return regexp.QuoteMeta(query(name)) }

func mustTab(t *testing.T, code string) inboxpladlapredla.Tab {
	t.Helper()
	tab, ok := inboxpladlapredla.FindTab(code)
	require.True(t, ok)
	return tab
}

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

var errDB = errors.New("oracle menolak")

// Daftar memetakan setiap kolom, mengikat delapan argumen, dan mengambil totalnya.
func TestListMapsRowsAndBindsEveryFilter(t *testing.T) {
	repo, mock := newMockRepo(t)

	from := day(2026, 1, 1)
	to := day(2026, 2, 1)
	q := inboxpladlapredla.Query{
		Tab: mustTab(t, "dla"), Search: "a%b_c", From: &from, To: &to,
	}

	mock.ExpectQuery(sqlOf("list_dla")).
		WithArgs(
			sql.NullString{String: "1", Valid: true},
			sql.NullString{String: `%A\%B\_C%`, Valid: true},
			sql.NullString{String: "1", Valid: true}, sql.NullTime{Time: from, Valid: true},
			sql.NullString{String: "1", Valid: true}, sql.NullTime{Time: to, Valid: true},
			10, 10,
		).
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow(" "+mockKey+" ", "PNC-1001 ", "POL-1", "PT Satu",
				day(2026, 1, 5), day(2026, 1, 2), " BUDI ", day(2026, 1, 10), 11).
			AddRow("k2", "PNC-1002", nil, nil, nil, nil, nil, nil, 11))

	page, err := repo.List(context.Background(), q,
		inboxpladlapredla.Pagination{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 11, page.Total)
	require.Equal(t, inboxpladlapredla.Pagination{Page: 2, Size: 10}, page.Pagination)
	require.Equal(t, []inboxpladlapredla.Row{
		{
			ClaimKey: mockKey, ClaimNo: "PNC-1001", PolicyNo: "POL-1", Insured: "PT Satu",
			RegisterDate: "2026-01-05", LossDate: "2026-01-02", PICTeknik: "BUDI",
			AdviceDate: "2026-01-10",
		},
		{ClaimKey: "k2", ClaimNo: "PNC-1002"},
	}, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tanpa penyaring, penanda dan nilainya terikat sebagai NULL.
func TestListWithoutFiltersBindsNulls(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("list_pre_dla")).
		WithArgs(sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullTime{},
			sql.NullString{}, sql.NullTime{}, 0, inboxpladlapredla.DefaultPageSize).
		WillReturnRows(sqlmock.NewRows(listColumns))

	page, err := repo.List(context.Background(),
		inboxpladlapredla.Query{Tab: mustTab(t, "pre-dla")}, inboxpladlapredla.Pagination{})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.Equal(t, 0, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	q := inboxpladlapredla.Query{Tab: mustTab(t, "pla")}

	// Jenis dokumen yang tidak dikenal ditolak sebelum basis data disentuh.
	repo, mock := newMockRepo(t)
	_, err := repo.List(ctx, inboxpladlapredla.Query{}, inboxpladlapredla.Pagination{})
	require.ErrorContains(t, err, "jenis dokumen")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("list_pla")).WillReturnError(errDB)
	_, err = repo.List(ctx, q, inboxpladlapredla.Pagination{})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "list_pla")

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("list_pla")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.List(ctx, q, inboxpladlapredla.Pagination{})
	require.ErrorContains(t, err, "memindai baris")

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("list_pla")).
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow("k", "n", "p", "i", nil, nil, "x", nil, 1).
			RowError(0, errDB))
	_, err = repo.List(ctx, q, inboxpladlapredla.Pagination{})
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Rincian memeriksa keberadaan klaim lebih dulu, lalu memetakan seluruh kolom.
func TestDocumentsChecksTheClaimThenMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("claim_exists")).WithArgs(mockKey).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(sqlOf("documents_pla")).WithArgs(mockKey).
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow(" PLA/1 ", "Reas A", "OR", "0", day(2026, 1, 10), "1",
				day(2026, 1, 11), day(2026, 1, 12), " catatan ", "a@contoh.example", nil).
			AddRow("PLA/2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	docs, err := repo.Documents(context.Background(), mustTab(t, "pla"), " "+mockKey)
	require.NoError(t, err)
	require.Equal(t, []inboxpladlapredla.Document{
		{
			AdviceNo: "PLA/1", Reinsurer: "Reas A", AdviceType: "OR", Revision: "0",
			AdviceDate: "2026-01-10", Sent: "1", SentDate: "2026-01-11",
			ReceivedDate: "2026-01-12", Notes: "catatan", Email: "a@contoh.example",
		},
		{AdviceNo: "PLA/2"},
	}, docs)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentsErrors(t *testing.T) {
	ctx := context.Background()
	dla := mustTab(t, "dla")

	repo, mock := newMockRepo(t)
	_, err := repo.Documents(ctx, dla, "  ")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	_, err = repo.Documents(ctx, mustTab(t, "pre-dla"), mockKey)
	require.ErrorContains(t, err, "tidak punya grid rincian")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("claim_exists")).WillReturnError(sql.ErrNoRows)
	_, err = repo.Documents(ctx, dla, mockKey)
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("claim_exists")).WillReturnError(errDB)
	_, err = repo.Documents(ctx, dla, mockKey)
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "claim_exists")

	existing := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(sqlOf("claim_exists")).
			WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	}

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("documents_dla")).WillReturnError(errDB)
	_, err = repo.Documents(ctx, dla, mockKey)
	require.ErrorContains(t, err, "documents_dla")

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("documents_dla")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.Documents(ctx, dla, mockKey)
	require.ErrorContains(t, err, "memindai baris")

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("documents_dla")).
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow("D", nil, nil, nil, nil, nil, nil, nil, nil, nil, "AKS").
			RowError(0, errDB))
	_, err = repo.Documents(ctx, dla, mockKey)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Panel Print Pre DLA menuliskan tanggal kirim dalam waktu Jakarta.
func TestPrintPreDLAMapsRowsInJakartaTime(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("claim_exists")).WithArgs(mockKey).
		WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(sqlOf("print_pre_dla")).WithArgs(mockKey).
		WillReturnRows(sqlmock.NewRows(printColumns).
			AddRow(" PRE/1 ", "Reas", "OR", time.Date(2026, 1, 10, 20, 0, 0, 0, time.UTC),
				"0", " ATT-1 ").
			AddRow("PRE/2", nil, nil, nil, nil, nil))

	items, err := repo.PrintPreDLA(context.Background(), mockKey)
	require.NoError(t, err)
	require.Equal(t, []inboxpladlapredla.PreDLADocument{
		{AdviceNo: "PRE/1", Reinsurer: "Reas", AdviceType: "OR", SentDate: "2026-01-11",
			Sent: "0", AttachmentKey: "ATT-1"},
		{AdviceNo: "PRE/2"},
	}, items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrintPreDLAErrors(t *testing.T) {
	ctx := context.Background()

	repo, _ := newMockRepo(t)
	_, err := repo.PrintPreDLA(ctx, "")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("claim_exists")).WillReturnError(sql.ErrNoRows)
	_, err = repo.PrintPreDLA(ctx, mockKey)
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(sqlOf("claim_exists")).WillReturnError(errDB)
	_, err = repo.PrintPreDLA(ctx, mockKey)
	require.ErrorIs(t, err, errDB)

	existing := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(sqlOf("claim_exists")).
			WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	}

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("print_pre_dla")).WillReturnError(errDB)
	_, err = repo.PrintPreDLA(ctx, mockKey)
	require.ErrorContains(t, err, "print_pre_dla")

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("print_pre_dla")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.PrintPreDLA(ctx, mockKey)
	require.ErrorContains(t, err, "memindai baris")

	repo, mock = newMockRepo(t)
	existing(mock)
	mock.ExpectQuery(sqlOf("print_pre_dla")).
		WillReturnRows(sqlmock.NewRows(printColumns).
			AddRow("P", nil, nil, nil, nil, nil).RowError(0, errDB))
	_, err = repo.PrintPreDLA(ctx, mockKey)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Diagnosis menghitung setiap tahap penyaring panel.
func TestDiagnosePrintPreDLAReadsEveryStage(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("print_pre_dla_diagnosa")).WithArgs(mockKey).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D"}).AddRow(4, 3, 2, 1))

	out, err := repo.DiagnosePrintPreDLA(context.Background(), " "+mockKey+" ")
	require.NoError(t, err)
	require.Equal(t, PrintPreDLADiagnosis{
		PreDLARows: 4, AdviceNoIs11: 3, HasAttachment: 2, CategoryIsDLA: 1}, out)

	mock.ExpectQuery(sqlOf("print_pre_dla_diagnosa")).WillReturnError(errDB)
	_, err = repo.DiagnosePrintPreDLA(context.Background(), mockKey)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penandaan Pre-DLA membedakan "baru ditandai" dari "sudah ditandai" lewat jumlah baris.
func TestMarkPreDLASent(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	_, err := repo.MarkPreDLASent(ctx, mockKey, " ")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	mock.ExpectExec(sqlOf("mark_pre_dla_sent")).WithArgs(mockKey, "PRE/1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	changed, err := repo.MarkPreDLASent(ctx, " "+mockKey, " PRE/1 ")
	require.NoError(t, err)
	require.True(t, changed)

	mock.ExpectExec(sqlOf("mark_pre_dla_sent")).WillReturnResult(sqlmock.NewResult(0, 0))
	changed, err = repo.MarkPreDLASent(ctx, mockKey, "PRE/1")
	require.NoError(t, err)
	require.False(t, changed)

	mock.ExpectExec(sqlOf("mark_pre_dla_sent")).WillReturnError(errDB)
	_, err = repo.MarkPreDLASent(ctx, mockKey, "PRE/1")
	require.ErrorIs(t, err, errDB)

	mock.ExpectExec(sqlOf("mark_pre_dla_sent")).
		WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = repo.MarkPreDLASent(ctx, mockKey, "PRE/1")
	require.ErrorContains(t, err, "membaca jumlah baris")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pembaca dokumen kirim memetakan dokumen dan klaimnya dari satu baris.
func TestAdviceForSendingMapsAdviceAndClaim(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("advice_for_sending_dla")).WithArgs(mockKey, "DLA/1").
		WillReturnRows(sqlmock.NewRows(sendableColumns).
			AddRow(" DLA/1 ", "OR", "Reas", "R1", " a@contoh.example ", "0", "LOG",
				"INDONESIA", "PNC-1001", "POL-1", "PT Satu", "Marine", day(2026, 1, 2)))

	advice, claim, err := repo.AdviceForSending(
		context.Background(), mustTab(t, "dla"), mockKey, " DLA/1 ")
	require.NoError(t, err)
	require.Equal(t, inboxpladlapredla.SendableAdvice{
		AdviceNo: "DLA/1", AdviceType: "OR", Reinsurer: "Reas", ReinsurerID: "R1",
		Email: "a@contoh.example", Sent: "0", Login: "LOG", Country: "INDONESIA",
	}, advice)
	require.Equal(t, inboxpladlapredla.ClaimSummary{
		ClaimNo: "PNC-1001", PolicyNo: "POL-1", Insured: "PT Satu", Business: "Marine",
		LossDate: "2026-01-02",
	}, claim)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdviceForSendingErrors(t *testing.T) {
	ctx := context.Background()
	pla := mustTab(t, "pla")

	repo, mock := newMockRepo(t)
	_, _, err := repo.AdviceForSending(ctx, pla, "", "PLA/1")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	_, _, err = repo.AdviceForSending(ctx, mustTab(t, "pre-dla"), mockKey, "PRE/1")
	require.ErrorIs(t, err, inboxpladlapredla.ErrDocumentsNotOnTab)

	mock.ExpectQuery(sqlOf("advice_for_sending_pla")).WillReturnError(sql.ErrNoRows)
	_, _, err = repo.AdviceForSending(ctx, pla, mockKey, "PLA/1")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	mock.ExpectQuery(sqlOf("advice_for_sending_pla")).WillReturnError(errDB)
	_, _, err = repo.AdviceForSending(ctx, pla, mockKey, "PLA/1")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "advice_for_sending_pla")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Lampiran tanpa isi dilewati; sisanya dipetakan apa adanya.
func TestAttachmentsForClaimSkipsEmptyContent(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("attachments_for_claim")).WithArgs(mockKey, "PLA").
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "MIME", "CONTENT"}).
			AddRow(" a.pdf ", " application/pdf ", []byte("isi")).
			AddRow("kosong.pdf", nil, []byte{}))

	items, err := repo.AttachmentsForClaim(context.Background(), " "+mockKey, " PLA ")
	require.NoError(t, err)
	require.Equal(t, []inboxpladlapredla.Attachment{
		{Name: "a.pdf", MIMEType: "application/pdf", Content: []byte("isi")},
	}, items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAttachmentsForClaimErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	_, err := repo.AttachmentsForClaim(ctx, " ", "PLA")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	mock.ExpectQuery(sqlOf("attachments_for_claim")).WillReturnError(errDB)
	_, err = repo.AttachmentsForClaim(ctx, mockKey, "PLA")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(sqlOf("attachments_for_claim")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.AttachmentsForClaim(ctx, mockKey, "PLA")
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(sqlOf("attachments_for_claim")).
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "MIME", "CONTENT"}).
			AddRow("a", "b", []byte("c")).RowError(0, errDB))
	_, err = repo.AttachmentsForClaim(ctx, mockKey, "PLA")
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penandaan PLA/DLA memperbarui master reasuransi dalam transaksi yang sama.
func TestMarkAdviceSentUpdatesTheMasterInOneTransaction(t *testing.T) {
	repo, mock := newMockRepo(t)
	advice := inboxpladlapredla.SendableAdvice{
		AdviceNo: "DLA/1", Reinsurer: "Reas", ReinsurerID: "R1", Email: "a@contoh.example",
	}

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_dla")).
		WithArgs(mockKey, "DLA/1", "a@contoh.example").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(sqlOf("update_reinsurer_email")).
		WithArgs("a@contoh.example", "R1", "Reas", "D").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	changed, err := repo.MarkAdviceSent(
		context.Background(), mustTab(t, "dla"), mockKey, " DLA/1 ", advice)
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tanpa kode reasuradur, master tidak disentuh.
func TestMarkAdviceSentWithoutAReinsurerCodeSkipsTheMaster(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	changed, err := repo.MarkAdviceSent(context.Background(), mustTab(t, "pla"), mockKey,
		"PLA/1", inboxpladlapredla.SendableAdvice{AdviceNo: "PLA/1", Email: "a@b.c"})
	require.NoError(t, err)
	require.True(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Dokumen yang sudah terkirim membatalkan transaksinya tanpa memperbarui master.
func TestMarkAdviceSentAlreadySentRollsBack(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	changed, err := repo.MarkAdviceSent(context.Background(), mustTab(t, "pla"), mockKey,
		"PLA/1", inboxpladlapredla.SendableAdvice{AdviceNo: "PLA/1", ReinsurerID: "R",
			Reinsurer: "Reas"})
	require.NoError(t, err)
	require.False(t, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkAdviceSentErrors(t *testing.T) {
	ctx := context.Background()
	pla := mustTab(t, "pla")
	full := inboxpladlapredla.SendableAdvice{
		AdviceNo: "PLA/1", Reinsurer: "Reas", ReinsurerID: "R1", Email: "a@b.c"}

	repo, mock := newMockRepo(t)
	_, err := repo.MarkAdviceSent(ctx, pla, "", "PLA/1", full)
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	_, err = repo.MarkAdviceSent(ctx, mustTab(t, "pre-dla"), mockKey, "PRE/1", full)
	require.ErrorIs(t, err, inboxpladlapredla.ErrDocumentsNotOnTab)

	mock.ExpectBegin().WillReturnError(errDB)
	_, err = repo.MarkAdviceSent(ctx, pla, mockKey, "PLA/1", full)
	require.ErrorContains(t, err, "memulai transaksi")

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.MarkAdviceSent(ctx, pla, mockKey, "PLA/1", full)
	require.ErrorContains(t, err, "mark_advice_sent_pla")

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).
		WillReturnResult(sqlmock.NewErrorResult(errDB))
	mock.ExpectRollback()
	_, err = repo.MarkAdviceSent(ctx, pla, mockKey, "PLA/1", full)
	require.ErrorContains(t, err, "membaca jumlah baris")

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(sqlOf("update_reinsurer_email")).WillReturnError(errDB)
	mock.ExpectRollback()
	_, err = repo.MarkAdviceSent(ctx, pla, mockKey, "PLA/1", full)
	require.ErrorContains(t, err, "update_reinsurer_email")

	mock.ExpectBegin()
	mock.ExpectExec(sqlOf("mark_advice_sent_pla")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(sqlOf("update_reinsurer_email")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errDB)
	_, err = repo.MarkAdviceSent(ctx, pla, mockKey, "PLA/1", full)
	require.ErrorContains(t, err, "menyimpan transaksi")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pelepasan wildcard menyentuh backslash, persen, dan garis bawah.
func TestEscapeLikeEscapesEveryWildcard(t *testing.T) {
	require.Equal(t, `a\\b\%c\_d`, escapeLike(`a\b%c_d`))
}

// Kueri yang tidak terdaftar adalah cacat pemrograman dan panik.
func TestAnUnknownQueryNamePanics(t *testing.T) {
	require.Panics(t, func() { query("tidak_ada") })
}
