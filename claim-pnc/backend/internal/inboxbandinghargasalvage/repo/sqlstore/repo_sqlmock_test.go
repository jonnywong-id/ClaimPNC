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

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Uji di berkas ini menjalankan ketiga pengisi SQL terhadap basis data tiruan.

var errOracle = errors.New("oracle menolak")

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func sqlNamed(name string) string { return regexp.QuoteMeta(query(name)) }

func tabNamed(t *testing.T, code string) inboxbandinghargasalvage.Tab {
	t.Helper()
	tab, ok := inboxbandinghargasalvage.FindTab(code)
	require.True(t, ok)
	return tab
}

func requestQuery(t *testing.T, keyword, waitFor string) inboxbandinghargasalvage.Query {
	t.Helper()
	return inboxbandinghargasalvage.Query{
		Tab:      tabNamed(t, inboxbandinghargasalvage.TabRequest),
		Keyword:  keyword,
		Reviewer: inboxbandinghargasalvage.Reviewer{Name: "komite1", WaitFor: waitFor},
	}
}

// Tab Request: hitung lebih dulu dengan lima argumen, lalu ambil halamannya.
func TestListRequestCountsThenReadsThePage(t *testing.T) {
	db, mock := newMockDB(t)
	requested := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(sqlNamed("count_request")).
		WithArgs("KOMITE1", "PNC-1", "PNC-1", "KOMITE0", "KOMITE0").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(12))
	mock.ExpectQuery(sqlNamed("list_request")).
		WithArgs("KOMITE1", "PNC-1", "PNC-1", "KOMITE0", "KOMITE0", 10, 10).
		WillReturnRows(sqlmock.NewRows(requestColumns).
			AddRow(" PNC-1 ", requested, "PNC-1/1", "Mobil", "100", "120", "catatan", 4.6,
				nil, "451", "KOMITE1").
			AddRow("PNC-2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := NewRepo(db).List(context.Background(), requestQuery(t, "PNC-1", " komite0 "),
		inboxbandinghargasalvage.Pagination{Page: 2, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 12, page.Total)
	require.Equal(t, []inboxbandinghargasalvage.AppealRow{
		{
			ClaimNo: "PNC-1", RequestDate: &requested, DetailObject: "PNC-1/1",
			ItemName: "Mobil", ItemPrice: "100", RequestPrice: "120", RequestNote: "catatan",
			AgingDays: 5, SalvageID: "451", CommitteeName: "KOMITE1",
		},
		{ClaimNo: "PNC-2"},
	}, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tab History memakai tiga argumen dan pemindai empat kolom; tanpa kata kunci ia NULL.
func TestListHistoryUsesThreeArguments(t *testing.T) {
	db, mock := newMockDB(t)
	q := inboxbandinghargasalvage.Query{
		Tab:      tabNamed(t, inboxbandinghargasalvage.TabHistory),
		Reviewer: inboxbandinghargasalvage.Reviewer{Name: "komite1"},
	}

	mock.ExpectQuery(sqlNamed("count_history")).WithArgs("KOMITE1", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(sqlNamed("list_history")).WithArgs("KOMITE1", nil, nil, 0, 20).
		WillReturnRows(sqlmock.NewRows(historyColumns).
			AddRow(" PNC-1 ", "Mobil", " Gudang ", "PIC"))

	page, err := NewRepo(db).List(context.Background(), q,
		inboxbandinghargasalvage.Pagination{Size: 20})
	require.NoError(t, err)
	require.Equal(t, []inboxbandinghargasalvage.AppealRow{
		{ClaimNo: "PNC-1", SalvageType: "Mobil", SalvageLocation: "Gudang", PIC: "PIC"},
	}, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tanpa baris yang cocok, kueri daftar tidak dijalankan sama sekali.
func TestListWithNothingToShowSkipsTheListQuery(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery(sqlNamed("count_request")).WithArgs("KOMITE1", nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))

	page, err := NewRepo(db).List(context.Background(), requestQuery(t, "", ""),
		inboxbandinghargasalvage.Pagination{})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	q := requestQuery(t, "", "")
	counted := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(sqlNamed("count_request")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(3))
	}

	db, mock := newMockDB(t)
	mock.ExpectQuery(sqlNamed("count_request")).WillReturnError(errOracle)
	_, err := NewRepo(db).List(ctx, q, inboxbandinghargasalvage.Pagination{})
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "count_request")

	db, mock = newMockDB(t)
	counted(mock)
	mock.ExpectQuery(sqlNamed("list_request")).WillReturnError(errOracle)
	_, err = NewRepo(db).List(ctx, q, inboxbandinghargasalvage.Pagination{})
	require.ErrorContains(t, err, "menjalankan kueri list_request")

	db, mock = newMockDB(t)
	counted(mock)
	mock.ExpectQuery(sqlNamed("list_request")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = NewRepo(db).List(ctx, q, inboxbandinghargasalvage.Pagination{})
	require.ErrorContains(t, err, "membaca baris kueri list_request")

	db, mock = newMockDB(t)
	counted(mock)
	mock.ExpectQuery(sqlNamed("list_request")).
		WillReturnRows(sqlmock.NewRows(requestColumns).
			AddRow("a", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).RowError(0, errOracle))
	_, err = NewRepo(db).List(ctx, q, inboxbandinghargasalvage.Pagination{})
	require.ErrorContains(t, err, "menelusuri hasil kueri list_request")

	// Pemindai History juga melaporkan galat Scan.
	db, mock = newMockDB(t)
	mock.ExpectQuery(sqlNamed("count_history")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(sqlNamed("list_history")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = NewRepo(db).List(ctx, inboxbandinghargasalvage.Query{
		Tab: tabNamed(t, inboxbandinghargasalvage.TabHistory)}, inboxbandinghargasalvage.Pagination{})
	require.ErrorContains(t, err, "membaca baris kueri list_history")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan tabel dan kolom tujuan penulisan.
func TestCheckTableAndWriteTargets(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewRepo(db)

	mock.ExpectQuery(sqlNamed("check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	require.NoError(t, repo.CheckTable(ctx))

	mock.ExpectQuery(sqlNamed("check_table")).WillReturnError(errOracle)
	require.ErrorContains(t, repo.CheckTable(ctx), "POOLDATA.T_CLAIM_CHEKER_SALVAGE")

	mock.ExpectQuery(sqlNamed("check_write_targets")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(WriteTargetColumns))
	found, err := repo.CheckWriteTargets(ctx)
	require.NoError(t, err)
	require.Equal(t, WriteTargetColumns, found)

	mock.ExpectQuery(sqlNamed("check_write_targets")).WillReturnError(errOracle)
	_, err = repo.CheckWriteTargets(ctx)
	require.ErrorContains(t, err, "membaca katalog kolom")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Panel rincian membaca keputusan satu klaim dengan nomor dan nama komite huruf besar.
func TestListDecisionsMapsEveryColumn(t *testing.T) {
	db, mock := newMockDB(t)
	approved := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(sqlNamed("list_decisions")).WithArgs("PNCN.26.0451", "KOMITE1").
		WillReturnRows(sqlmock.NewRows(decisionColumns).
			AddRow(approved, " PNC-0451/1 ", "Mobil", "100", "120", " 1 ", "KOMITE1").
			AddRow(nil, nil, nil, nil, nil, nil, nil))

	items, err := NewRepo(db).ListDecisions(context.Background(),
		inboxbandinghargasalvage.DecisionQuery{
			ClaimNo:  "pncn.26.0451",
			Reviewer: inboxbandinghargasalvage.Reviewer{Name: "komite1"},
		})
	require.NoError(t, err)
	require.Equal(t, []inboxbandinghargasalvage.Decision{
		{ApprovedAt: &approved, DetailObject: "PNC-0451/1", ItemName: "Mobil",
			ItemPrice: "100", RequestPrice: "120", Status: "1", CommitteeName: "KOMITE1"},
		{},
	}, items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListDecisionsErrors(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewRepo(db)
	q := inboxbandinghargasalvage.DecisionQuery{ClaimNo: "X"}

	mock.ExpectQuery(sqlNamed("list_decisions")).WillReturnError(errOracle)
	_, err := repo.ListDecisions(ctx, q)
	require.ErrorContains(t, err, "menjalankan kueri list_decisions")

	mock.ExpectQuery(sqlNamed("list_decisions")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = repo.ListDecisions(ctx, q)
	require.ErrorContains(t, err, "membaca baris kueri list_decisions")

	mock.ExpectQuery(sqlNamed("list_decisions")).
		WillReturnRows(sqlmock.NewRows(decisionColumns).
			AddRow(nil, nil, nil, nil, nil, nil, nil).RowError(0, errOracle))
	_, err = repo.ListDecisions(ctx, q)
	require.ErrorContains(t, err, "menelusuri hasil kueri list_decisions")
	require.NoError(t, mock.ExpectationsWereMet())
}

func fullCommand() inboxbandinghargasalvage.DecisionCommand {
	return inboxbandinghargasalvage.DecisionCommand{
		DetailObject: "PNC-0451/1", SalvageID: "451", RequestPrice: "9750000.00",
		Note: "ok", Status: inboxbandinghargasalvage.DecisionApproved,
		Reviewer:  inboxbandinghargasalvage.Reviewer{Name: " komite1 "},
		CascadeTo: " komite2 ", ApplyPrice: true, MarkDocument: true,
	}
}

// Keputusan lengkap menjalankan keempat langkah dalam satu transaksi.
func TestDecideRunsEveryStepInOneTransaction(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(sqlNamed("decide_record")).
		WithArgs("1", "ok", "KOMITE1", "PNC-0451/1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(sqlNamed("decide_cascade")).
		WithArgs("1", "KOMITE2", "PNC-0451/1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(sqlNamed("decide_mark_document")).
		WithArgs("PNC-0451/1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(sqlNamed("decide_apply_price")).
		WithArgs("9750000.00", "451", "PNC-0451/1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewWriter(db).Decide(context.Background(), fullCommand())
	require.NoError(t, err)
	require.Equal(t, inboxbandinghargasalvage.DecisionResult{
		Recorded: true, PriceApplied: true, DocumentMarked: true}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Keputusan sederhana hanya mencatat putusan.
func TestDecideWithoutExtraStepsOnlyRecords(t *testing.T) {
	db, mock := newMockDB(t)

	mock.ExpectBegin()
	mock.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := NewWriter(db).Decide(context.Background(),
		inboxbandinghargasalvage.DecisionCommand{
			DetailObject: "D", Status: inboxbandinghargasalvage.DecisionRejected})
	require.NoError(t, err)
	require.Equal(t, inboxbandinghargasalvage.DecisionResult{Recorded: true}, result)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideErrors(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"mulai transaksi", func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"sudah diputus", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, func(t *testing.T, err error) {
			require.ErrorIs(t, err, inboxbandinghargasalvage.ErrAlreadyDecided)
		}},
		{"catat gagal", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "mencatat keputusan") }},
		{"jumlah baris", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).
				WillReturnResult(sqlmock.NewErrorResult(errOracle))
			m.ExpectRollback()
		}, func(t *testing.T, err error) {
			require.ErrorContains(t, err, "membaca jumlah baris keputusan")
		}},
		{"jenjang berikutnya", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_cascade")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) {
			require.ErrorContains(t, err, "menutup baris komite berikutnya")
		}},
		{"tandai dokumen", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_cascade")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_mark_document")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menandai dokumen") }},
		{"terapkan harga", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_cascade")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_mark_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_apply_price")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menerapkan harga") }},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(sqlNamed("decide_record")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_cascade")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_mark_document")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(sqlNamed("decide_apply_price")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) {
			require.ErrorContains(t, err, "menyimpan keputusan banding")
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMockDB(t)
			c.prepare(mock)
			_, err := NewWriter(db).Decide(context.Background(), fullCommand())
			c.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func documentQuery() inboxbandinghargasalvage.DocumentQuery {
	return inboxbandinghargasalvage.DocumentQuery{
		DetailObject: "PNC-0451/1", SalvageID: "451",
		Reviewer: inboxbandinghargasalvage.Reviewer{Name: " komite1 "},
	}
}

// Daftar dokumen memetakan id, nama, dan tanggal unggah.
func TestListDocumentsMapsRows(t *testing.T) {
	db, mock := newMockDB(t)
	uploaded := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(sqlNamed("list_documents")).WithArgs("PNC-0451/1", "451", "KOMITE1").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAME", "TGLINS"}).
			AddRow("9001", " a.pdf ", uploaded).
			AddRow("9002", nil, nil))

	rows, err := NewDocumentReader(db).ListDocuments(context.Background(), documentQuery())
	require.NoError(t, err)
	require.Equal(t, []inboxbandinghargasalvage.DocumentRow{
		{ID: "9001", Name: "a.pdf", UploadedAt: &uploaded},
		{ID: "9002"},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListDocumentsErrors(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	reader := NewDocumentReader(db)

	mock.ExpectQuery(sqlNamed("list_documents")).WillReturnError(errOracle)
	_, err := reader.ListDocuments(ctx, documentQuery())
	require.ErrorIs(t, err, errOracle)

	mock.ExpectQuery(sqlNamed("list_documents")).
		WillReturnRows(sqlmock.NewRows([]string{"satu"}).AddRow("x"))
	_, err = reader.ListDocuments(ctx, documentQuery())
	require.ErrorContains(t, err, "memindai dokumen banding")

	mock.ExpectQuery(sqlNamed("list_documents")).
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAME", "TGLINS"}).
			AddRow("1", nil, nil).RowError(0, errOracle))
	_, err = reader.ListDocuments(ctx, documentQuery())
	require.ErrorIs(t, err, errOracle)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Isi dokumen dibaca bersama penyaring kepemilikan; nol baris berarti tidak ditemukan.
func TestDocumentContent(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	reader := NewDocumentReader(db)

	mock.ExpectQuery(sqlNamed("document_content")).
		WithArgs("9001", "PNC-0451/1", "451", "KOMITE1").
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "MIME", "CONTENT"}).
			AddRow(" a.pdf ", " application/pdf ", []byte("%PDF")))
	content, err := reader.DocumentContent(ctx, "9001", documentQuery())
	require.NoError(t, err)
	require.Equal(t, inboxbandinghargasalvage.DocumentContent{
		Name: "a.pdf", MIMEType: "application/pdf", Content: []byte("%PDF")}, content)

	mock.ExpectQuery(sqlNamed("document_content")).WillReturnError(sql.ErrNoRows)
	_, err = reader.DocumentContent(ctx, "9004", documentQuery())
	require.ErrorIs(t, err, inboxbandinghargasalvage.ErrDocumentNotFound)

	mock.ExpectQuery(sqlNamed("document_content")).WillReturnError(errOracle)
	_, err = reader.DocumentContent(ctx, "9001", documentQuery())
	require.ErrorIs(t, err, errOracle)
	require.ErrorContains(t, err, "membaca isi dokumen banding")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnknownQueryNamePanics(t *testing.T) {
	require.Panics(t, func() { query("tidak_ada") })
}
