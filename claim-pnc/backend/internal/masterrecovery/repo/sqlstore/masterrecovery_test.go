package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrecovery"
)

// q mengubah teks kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	r := NewRepo(db)
	r.now = func() time.Time { return time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC) }
	return r, mock
}

var errDB = errors.New("db rusak")

func TestNextBatch(t *testing.T) {
	r, mock := newMock(t)
	mock.ExpectQuery(q("recovery_next_batch")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(7))
	batch, err := r.NextBatch(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(7), batch)

	mock.ExpectQuery(q("recovery_next_batch")).WillReturnError(errDB)
	_, err = r.NextBatch(context.Background())
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func sampleRecovery() masterrecovery.Recovery {
	return masterrecovery.Recovery{
		PrincipalName: "PT A", Year: "2026", ClaimAmount: 100, PreviousPayment: 0, Payment: 10,
		Remainder: 90, Remark: "ket", CasePosition: "", DocumentID: "D1", VirtualAccountNumber: "VA1",
		InputBy: "U1", ClientID: "C1", ServiceLogID: "", PolicyNo: "P1", BusinessID: "B",
		BranchID: "BR", AgentID: "AG", MarketingID: "MO",
		ClaimLine: []masterrecovery.ClaimLine{{PolicyNo: "P1", ClaimAmount: 60}, {PolicyNo: "P2", ClaimAmount: 40}},
	}
}

func TestInsertCommitsBatchAndClaimLines(t *testing.T) {
	r, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("recovery_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(q("recovery_next_batch")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(12))
	mock.ExpectExec(q("recovery_insert")).WithArgs(
		int64(12), "PT A", "2026", int64(100), int64(0), int64(10), int64(90), "ket", nil, "D1", "VA1",
		"U1", "C1", nil, "P1", "B", "BR", "AG", "MO",
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(q("recovery_claim_line_insert")).WithArgs(int64(12), 1, "P1", int64(60)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(q("recovery_claim_line_insert")).WithArgs(int64(12), 2, "P2", int64(40)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	saved, err := r.Insert(context.Background(), sampleRecovery())
	require.NoError(t, err)
	require.Equal(t, int64(12), saved.Batch)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("begin", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errDB)
		_, err := r.Insert(ctx, sampleRecovery())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("lock", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("recovery_lock_table")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.Insert(ctx, sampleRecovery())
		require.ErrorContains(t, err, "mengunci tabel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("next batch", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("recovery_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(q("recovery_next_batch")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.Insert(ctx, sampleRecovery())
		require.ErrorContains(t, err, "menerbitkan nomor batch")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("primary key clash", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("recovery_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(q("recovery_next_batch")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(1))
		mock.ExpectExec(q("recovery_insert")).WillReturnError(errors.New("ORA-00001: unique constraint (POOLDATA.mst_recovery_asm_penjaminan_pk) violated"))
		mock.ExpectRollback()
		_, err := r.Insert(ctx, sampleRecovery())
		require.ErrorIs(t, err, masterrecovery.ErrBatchTaken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("claim line", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("recovery_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(q("recovery_next_batch")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(1))
		mock.ExpectExec(q("recovery_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(q("recovery_claim_line_insert")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.Insert(ctx, sampleRecovery())
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "menyisipkan baris klaim recovery")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		r, mock := newMock(t)
		rec := sampleRecovery()
		rec.ClaimLine = nil
		mock.ExpectBegin()
		mock.ExpectExec(q("recovery_lock_table")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(q("recovery_next_batch")).WillReturnRows(sqlmock.NewRows([]string{"b"}).AddRow(1))
		mock.ExpectExec(q("recovery_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errDB)
		_, err := r.Insert(ctx, rec)
		require.ErrorContains(t, err, "menyimpan batch recovery")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

var listColumns = []string{
	"batch", "principal", "year", "claim", "prev", "pay", "rem", "remark", "pos", "doc", "va",
	"client", "policy", "log", "inputdate", "aname", "aop", "adate",
}

func TestListMapsRowsAndAttachment(t *testing.T) {
	r, mock := newMock(t)
	at := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)

	mock.ExpectQuery(q("recovery_count")).WithArgs("PT", "PT", nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
	mock.ExpectQuery(q("recovery_list")).WithArgs("PT", "PT", nil, nil, 10, 5).
		WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow(1, " PT A ", "2026", 100, 0, 10, 90, " r ", "pos", " D1 ", "VA", "C", "P", "L", at, " b.pdf ", " U ", at).
			// DOKUMENID terisi tetapi baris lampiran tidak ketemu → tanpa lampiran.
			AddRow(2, "PT B", nil, nil, nil, nil, nil, nil, nil, "D2", nil, nil, nil, nil, nil, nil, nil, nil).
			// Tanpa DOKUMENID → tanpa lampiran, meski kolom lampiran kebetulan terisi.
			AddRow(3, "PT B", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "x", nil, nil))

	rows, total, err := r.List(context.Background(), masterrecovery.ListFilter{PrincipalName: "PT", Offset: 10, Limit: 5})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, rows, 3)
	require.Equal(t, masterrecovery.Recovery{
		Batch: 1, PrincipalName: "PT A", Year: "2026", ClaimAmount: 100, Payment: 10, Remainder: 90,
		Remark: "r", CasePosition: "pos", DocumentID: "D1", VirtualAccountNumber: "VA", ClientID: "C",
		PolicyNo: "P", ServiceLogID: "L", InputDate: at,
		Attachment: &masterrecovery.AttachmentInfo{ID: "D1", Name: "b.pdf", UploadedBy: "U", UploadedAt: at},
	}, rows[0])
	require.Nil(t, rows[1].Attachment)
	require.Equal(t, "D2", rows[1].DocumentID)
	require.True(t, rows[1].InputDate.IsZero())
	require.Nil(t, rows[2].Attachment)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("count", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectQuery(q("recovery_count")).WillReturnError(errDB)
		_, _, err := r.List(ctx, masterrecovery.ListFilter{})
		require.ErrorContains(t, err, "menghitung batch")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectQuery(q("recovery_count")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		mock.ExpectQuery(q("recovery_list")).WillReturnError(errDB)
		_, _, err := r.List(ctx, masterrecovery.ListFilter{})
		require.ErrorContains(t, err, "membaca daftar batch")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectQuery(q("recovery_count")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		mock.ExpectQuery(q("recovery_list")).WillReturnRows(sqlmock.NewRows([]string{"batch"}).AddRow(1))
		_, _, err := r.List(ctx, masterrecovery.ListFilter{})
		require.ErrorContains(t, err, "membaca baris batch")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectQuery(q("recovery_count")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		mock.ExpectQuery(q("recovery_list")).WillReturnRows(sqlmock.NewRows(listColumns).
			AddRow(1, "A", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
			RowError(0, errDB))
		_, _, err := r.List(ctx, masterrecovery.ListFilter{})
		require.ErrorContains(t, err, "menelusuri daftar batch")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindDocument(t *testing.T) {
	ctx := context.Background()
	columns := []string{"content", "name", "mime", "note", "image"}

	r, mock := newMock(t)
	mock.ExpectQuery(q("attachment_read")).WithArgs("D1").
		WillReturnRows(sqlmock.NewRows(columns).AddRow([]byte("isi"), " a.pdf ", " application/pdf ", " n ", nil))
	doc, err := r.FindDocument(ctx, "D1")
	require.NoError(t, err)
	require.Equal(t, masterrecovery.Document{Name: "a.pdf", MimeType: "application/pdf", Note: "n", Content: []byte("isi")}, doc)

	mock.ExpectQuery(q("attachment_read")).WithArgs("D2").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(nil, "a", nil, nil, "IMG1"))
	_, err = r.FindDocument(ctx, "D2")
	require.ErrorIs(t, err, masterrecovery.ErrDocumentElsewhere)

	mock.ExpectQuery(q("attachment_read")).WithArgs("D3").WillReturnError(sql.ErrNoRows)
	_, err = r.FindDocument(ctx, "D3")
	require.ErrorIs(t, err, masterrecovery.ErrDocumentNotFound)

	mock.ExpectQuery(q("attachment_read")).WithArgs("D4").WillReturnError(errDB)
	_, err = r.FindDocument(ctx, "D4")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

var principalColumns = []string{"client", "name", "va", "email", "status", "message"}

func TestListPrincipal(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	mock.ExpectQuery(q("principal_list")).WillReturnRows(sqlmock.NewRows(principalColumns).
		AddRow(" C1 ", " PT A ", " VA1 ", " e@x ", nil, nil))
	list, err := r.ListPrincipal(ctx)
	require.NoError(t, err)
	require.Equal(t, []masterrecovery.Principal{{ClientID: "C1", Name: "PT A", VirtualAccountNumber: "VA1", Email: "e@x"}}, list)

	mock.ExpectQuery(q("principal_list")).WillReturnError(errDB)
	_, err = r.ListPrincipal(ctx)
	require.ErrorContains(t, err, "membaca daftar principal")

	mock.ExpectQuery(q("principal_list")).WillReturnRows(sqlmock.NewRows([]string{"client"}).AddRow("C"))
	_, err = r.ListPrincipal(ctx)
	require.ErrorContains(t, err, "membaca baris principal")

	mock.ExpectQuery(q("principal_list")).WillReturnRows(sqlmock.NewRows(principalColumns).
		AddRow("C", "N", nil, nil, nil, nil).RowError(0, errDB))
	_, err = r.ListPrincipal(ctx)
	require.ErrorContains(t, err, "menelusuri daftar principal")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindPrincipal(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	mock.ExpectQuery(q("principal_find")).WithArgs("C1", "PT A").
		WillReturnRows(sqlmock.NewRows(principalColumns).AddRow("C1", "PT A", "VA1", nil, "OK", "m"))
	p, err := r.FindPrincipal(ctx, "C1", "PT A")
	require.NoError(t, err)
	require.Equal(t, masterrecovery.Principal{ClientID: "C1", Name: "PT A", VirtualAccountNumber: "VA1", Status: "OK", Message: "m"}, p)

	mock.ExpectQuery(q("principal_find")).WillReturnError(sql.ErrNoRows)
	_, err = r.FindPrincipal(ctx, "x", "y")
	require.ErrorIs(t, err, masterrecovery.ErrPrincipalNotFound)

	mock.ExpectQuery(q("principal_find")).WillReturnError(errDB)
	_, err = r.FindPrincipal(ctx, "x", "y")
	require.ErrorContains(t, err, "mencari principal")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSavePrincipal(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	mock.ExpectExec(q("principal_insert")).WithArgs("C1", "PT A", nil, nil, "VA1", "e@x").
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, r.SavePrincipal(ctx, masterrecovery.Principal{ClientID: "C1", Name: "PT A", VirtualAccountNumber: "VA1", Email: "e@x"}))

	mock.ExpectExec(q("principal_insert")).WillReturnError(errDB)
	require.ErrorContains(t, r.SavePrincipal(ctx, masterrecovery.Principal{}), "mencatat principal")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLookupPolicy(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	mock.ExpectQuery(q("policy_reference")).WithArgs("P1").
		WillReturnRows(sqlmock.NewRows([]string{"b", "br", "ag", "mo"}).AddRow(" 01 ", "001", nil, "MO"))
	ref, err := r.LookupPolicy(ctx, "P1")
	require.NoError(t, err)
	require.Equal(t, masterrecovery.PolicyReference{BusinessID: "01", BranchID: "001", MarketingID: "MO"}, ref)

	mock.ExpectQuery(q("policy_reference")).WillReturnError(sql.ErrNoRows)
	_, err = r.LookupPolicy(ctx, "X")
	require.ErrorIs(t, err, masterrecovery.ErrPolicyNotFound)

	mock.ExpectQuery(q("policy_reference")).WillReturnError(errDB)
	_, err = r.LookupPolicy(ctx, "X")
	require.ErrorContains(t, err, "membaca identitas polis")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveDocument(t *testing.T) {
	ctx := context.Background()
	doc := masterrecovery.Document{Name: "a.pdf", MimeType: "application/pdf", Content: []byte("isi"), UploadedBy: "U1"}

	r, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("attachment_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow(1900000))
	mock.ExpectExec(q("attachment_counter_insert")).WithArgs(sqlmock.AnyArg(), "26", int64(1900000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(q("attachment_insert")).WithArgs(
		"260001900000", []byte("isi"), "U1", "a.pdf", nil, "application/pdf", DocumentCategory, DocumentSubCategory, nil,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	id, err := r.SaveDocument(ctx, doc)
	require.NoError(t, err)
	require.Equal(t, "260001900000", id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveDocumentFailures(t *testing.T) {
	ctx := context.Background()
	doc := masterrecovery.Document{Name: "a.pdf", Content: []byte("isi")}
	seq := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		mock.ExpectQuery(q("attachment_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow(5))
	}

	t.Run("begin", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errDB)
		_, err := r.SaveDocument(ctx, doc)
		require.ErrorContains(t, err, "memulai transaksi lampiran")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("sequence", func(t *testing.T) {
		r, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("attachment_next_sequence")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.SaveDocument(ctx, doc)
		require.ErrorContains(t, err, "nomor urut lampiran")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("counter", func(t *testing.T) {
		r, mock := newMock(t)
		seq(mock)
		mock.ExpectExec(q("attachment_counter_insert")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.SaveDocument(ctx, doc)
		require.ErrorContains(t, err, "mencatat penerbitan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("insert", func(t *testing.T) {
		r, mock := newMock(t)
		seq(mock)
		mock.ExpectExec(q("attachment_counter_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(q("attachment_insert")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := r.SaveDocument(ctx, doc)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "menyimpan bukti bayar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit", func(t *testing.T) {
		r, mock := newMock(t)
		seq(mock)
		mock.ExpectExec(q("attachment_counter_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(q("attachment_insert")).WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errDB)
		_, err := r.SaveDocument(ctx, doc)
		require.ErrorContains(t, err, "menyimpan bukti bayar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFormattingHelpers(t *testing.T) {
	require.Equal(t, "05", TwoDigitYear(time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, "26", TwoDigitYear(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, "0000000007", TenDigits(7))
	require.Equal(t, "12345678901", TenDigits(12345678901))

	key, err := newCounterKey()
	require.NoError(t, err)
	require.Len(t, key, 32)

	require.Nil(t, nullable("  "))
	require.Equal(t, "x", nullable("x"))

	err = translateWriteError(errDB, "aktivitas")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "aktivitas")
}

func TestCheckTable(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	for _, name := range []string{"recovery_check_table", "principal_check_table", "attachment_check_table"} {
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"x"}))
	}
	require.NoError(t, r.CheckTable(ctx))

	mock.ExpectQuery(q("recovery_check_table")).WillReturnRows(sqlmock.NewRows([]string{"x"}))
	mock.ExpectQuery(q("principal_check_table")).WillReturnError(errDB)
	err := r.CheckTable(ctx)
	require.ErrorContains(t, err, "POOLDATA.MST_VIRTUAL_ACCOUNT_PNC tidak dapat dibaca")

	mock.ExpectQuery(q("recovery_check_table")).WillReturnRows(sqlmock.NewRows([]string{"x"}).CloseError(errDB))
	err = r.CheckTable(ctx)
	require.ErrorContains(t, err, "menutup pemeriksaan POOLDATA.MST_RECOVERY_ASM_PENJAMINAN")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckClaimLineTableAndPolicyLink(t *testing.T) {
	ctx := context.Background()
	r, mock := newMock(t)
	mock.ExpectQuery(q("recovery_claim_line_check_table")).WillReturnRows(sqlmock.NewRows([]string{"x"}))
	require.NoError(t, r.CheckClaimLineTable(ctx))
	mock.ExpectQuery(q("recovery_claim_line_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, r.CheckClaimLineTable(ctx), "CPNC_RECOVERY_BARIS_KLAIM")

	mock.ExpectQuery(q("policy_reference")).WithArgs("").WillReturnRows(sqlmock.NewRows([]string{"x"}))
	require.NoError(t, r.CheckPolicyLink(ctx))
	mock.ExpectQuery(q("policy_reference")).WillReturnError(errDB)
	require.ErrorContains(t, r.CheckPolicyLink(ctx), "MST_DET_SALES@ASMD")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryLoading(t *testing.T) {
	// Seluruh kueri yang dipakai kode termuat dari berkas .sql.
	for _, name := range []string{
		"recovery_next_batch", "recovery_lock_table", "recovery_insert", "recovery_list",
		"recovery_count", "attachment_read", "principal_list", "principal_find",
		"principal_insert", "policy_reference", "attachment_next_sequence",
		"attachment_counter_insert", "attachment_insert",
	} {
		require.NotEmpty(t, getQuery(name), name)
		require.False(t, strings.Contains(getQuery(name), "-- name:"), name)
	}
	require.PanicsWithValue(t, `masterrecovery/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`, func() { getQuery("tidak_ada") })

	got := splitByName("-- kepala\n-- name: a\n-- penjelas\nselect 1\n\n-- name: kosong\n\n-- name: b\nselect 2\n")
	require.Equal(t, map[string]string{"a": "select 1", "b": "select 2"}, got)
}
