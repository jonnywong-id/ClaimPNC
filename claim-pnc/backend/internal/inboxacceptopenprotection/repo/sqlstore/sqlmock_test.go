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

	"claim-pnc/internal/inboxacceptopenprotection"
)

// Uji di berkas ini menembak adapter lewat go-sqlmock, sehingga yang diperiksa adalah kueri
// yang benar-benar dikirim beserta argumennya, dan pemetaan baris ke struct.

var (
	reCount  = regexp.QuoteMeta("SELECT COUNT(*)") + ".*T_CLAIM_OPENPROTECTION"
	reList   = regexp.QuoteMeta("ORDER BY p.CREATE_DATE DESC") + ".*OFFSET"
	reGet    = regexp.QuoteMeta("POOLDATA.T_GENERAL") + ".*" + regexp.QuoteMeta("UPPER(TRIM(p.OPEN_PROTECTION_ID)) = :1")
	reDecide = regexp.QuoteMeta("UPDATE POOLDATA.T_CLAIM_OPENPROTECTION")
	reApply  = regexp.QuoteMeta("UPDATE POOLDATA.T_CLAIM_PNC")

	// Dibedakan dari reApply supaya sebuah uji tidak lolos karena kebetulan mencocoki UPDATE
	// yang salah: keduanya sama-sama UPDATE di dalam satu transaksi.
	reApplyCOL = regexp.QuoteMeta("UPDATE POOLDATA.T_CLAIM_OBJECTCOVERAGE")
	reDescribe = regexp.QuoteMeta("FROM POOLDATA.D_CAUSE_OF_LOSS")
	reGroups   = "M_LOGIN_GROUP_PNC"
	errBasis   = errors.New("basis data mati")
	listCols   = []string{"OPEN_PROTECTION_ID", "POLICY_NO", "CLAIM_NO", "ID_CLAIM", "PROTECTION_TYPE_ID", "PROTECTION_TYPE_NAME", "CREATE_DATE", "NOTES", "CREATED_BY", "OLD_DATA", "NEW_DATA", "OBJECT_NAME", "BRANCH_NAME", "APPROVAL_STATUS", "RESOLVED_DATETIME", "RESOLVED_BY", "OBJECT_ID", "OBJECT_COVERAGE_ID"}
	detailCol  = append(append([]string(nil), listCols...), "THEINSURED", "STARTDATE", "ENDDATE")
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// detailRow membentuk satu baris kueri detail, TANPA sasaran coverage.
//
// Keduanya NULL — persis baris warisan Pega, yang tidak punya kolom asal bagi `OBJECT_ID`
// dan `OBJECT_COVERAGE_ID`. Pengujian perubahan Cause of Loss memakai detailRowCOL.
func detailRow(id, typ, oldData, newData, status, claimRef string) *sqlmock.Rows {
	return detailRowCOL(id, typ, oldData, newData, status, claimRef, nil, nil)
}

// detailRowCOL membentuk satu baris kueri detail BESERTA sasaran coverage-nya.
func detailRowCOL(
	id, typ, oldData, newData, status, claimRef string,
	objectID, coverageID any,
) *sqlmock.Rows {
	dibuat := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	mulai := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	var st any
	if status != "" {
		st = status
	}
	var ref any
	if claimRef != "" {
		ref = claimRef
	}
	return sqlmock.NewRows(detailCol).AddRow(
		" "+id+" ", "POL-1", "PNCN.26.0001", ref, typ, "Nama Tipe", dibuat,
		"catatan", "PEMBUAT", oldData, newData, "Objek", "Cabang",
		st, nil, nil, objectID, coverageID, " Tertanggung ", mulai, akhir)
}

func TestListMapsRowsAndSendsQueueAndSearchArgs(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	dibuat := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	diputuskan := time.Date(2026, 9, 21, 3, 0, 0, 0, time.UTC)

	// Antrean PREMI: penanda sama = 1, kode '2'; penanda aktif membawa kata apa adanya, sedangkan pola LIKE tanpa wildcard
	// yang diketik pengguna.
	mock.ExpectQuery(reCount).
		WithArgs(1, "2", 1, "2", "A%B_", "%AB%", "%AB%", "%AB%").
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(2))
	mock.ExpectQuery(reList).
		WithArgs(1, "2", 1, "2", "A%B_", "%AB%", "%AB%", "%AB%", 5, 10).
		WillReturnRows(sqlmock.NewRows(listCols).
			AddRow("OPCN.1", " POL ", "KLM", "REF", "2", nil, dibuat, nil, "U",
				nil, nil, nil, nil, nil, nil, nil, nil, nil).
			AddRow("OPCN.2", "POL", "KLM", "REF", "7", "Ubah DOL", nil, "n", "U",
				"2026-08-03", "rusak", "Obj", "Cab", "1", diputuskan, "PETUGAS", nil, nil))

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{
		Queue: inboxacceptopenprotection.QueuePremium, Search: " a%b_ ", Limit: 10, Offset: 5,
	})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Protections, 2)

	first := page.Protections[0]
	require.Equal(t, "OPCN.1", first.Number)
	require.Equal(t, "POL", first.PolicyNumber)
	require.Equal(t, "", first.TypeName)
	require.True(t, first.InputDate.Equal(dibuat))
	require.Nil(t, first.AcceptedAt)
	require.True(t, first.Change.Empty())

	second := page.Protections[1]
	require.True(t, second.InputDate.IsZero())
	require.Equal(t, "1", second.AcceptStatus)
	require.Equal(t, "PETUGAS", second.AcceptedBy)
	require.True(t, second.AcceptedAt.Equal(diputuskan))
	// Tanggal lama terbaca; tanggal baru yang tidak dapat dibaca menjadi nil, bukan galat.
	require.NotNil(t, second.Change.LossDateBefore)
	require.Equal(t, time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), *second.Change.LossDateBefore)
	require.Nil(t, second.Change.LossDateAfter)
	require.Equal(t, "Obj", second.Change.ObjectName)
	// Kueri daftar tidak membawa kolom polis.
	require.Nil(t, second.PolicyStart)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListNonPremiumWithoutSearchSendsNulls(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(reCount).
		WithArgs(0, "2", 0, "2", nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(0))
	mock.ExpectQuery(reList).
		WithArgs(0, "2", 0, "2", nil, nil, nil, nil, 0, inboxacceptopenprotection.DefaultLimit).
		WillReturnRows(sqlmock.NewRows(listCols))

	page, err := repo.List(context.Background(), inboxacceptopenprotection.Filter{})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Protections)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reCount).WillReturnError(errBasis)

		_, err := NewRepo(db).List(context.Background(), inboxacceptopenprotection.Filter{})
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "menghitung antrean akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reCount).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		mock.ExpectQuery(reList).WillReturnError(errBasis)

		_, err := NewRepo(db).List(context.Background(), inboxacceptopenprotection.Filter{})
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "membaca antrean akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reCount).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		// Kolom yang kurang membuat Scan gagal.
		mock.ExpectQuery(reList).WillReturnRows(sqlmock.NewRows([]string{"a"}).AddRow("x"))

		_, err := NewRepo(db).List(context.Background(), inboxacceptopenprotection.Filter{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "memindai antrean akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reCount).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
		mock.ExpectQuery(reList).WillReturnRows(sqlmock.NewRows(listCols).
			AddRow("A", "P", "K", "R", "1", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil).
			RowError(0, errBasis))

		_, err := NewRepo(db).List(context.Background(), inboxacceptopenprotection.Filter{})
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "membaca baris antrean akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetMapsDetailIncludingPolicySnapshot(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(reGet).WithArgs("OPCN.26.0005").
		WillReturnRows(detailRow("OPCN.26.0005", "8", "12001", "12002", "", "REF"))

	p, err := NewRepo(db).Get(context.Background(), " opcn.26.0005 ")
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0005", p.Number)
	require.Equal(t, "REF", p.ClaimReference)
	require.Equal(t, "Nama Tipe", p.TypeName)
	require.Equal(t, "Tertanggung", p.InsuredName)
	require.True(t, p.Pending())
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), *p.PolicyStart)
	require.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), *p.PolicyEnd)
	// Tipe '8': isinya kode penyebab kerugian.
	require.Equal(t, "12001", p.Change.CauseOfLossBefore)
	require.Equal(t, "12002", p.Change.CauseOfLossAfter)
	require.Nil(t, p.Change.LossDateBefore)
	require.Equal(t, "Cabang", p.Change.BranchName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOtherTypeIgnoresChangeColumns(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(reGet).WithArgs("X").
		WillReturnRows(detailRow("X", "1", "2026-08-03", "2026-08-05", "", ""))

	p, err := NewRepo(db).Get(context.Background(), "x")
	require.NoError(t, err)
	require.Equal(t, inboxacceptopenprotection.ChangeDetail{}, p.Change)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotFoundAndScanError(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(reGet).WithArgs("A").WillReturnRows(sqlmock.NewRows(detailCol))
	mock.ExpectQuery(reGet).WithArgs("B").WillReturnError(errBasis)

	repo := NewRepo(db)
	_, err := repo.Get(context.Background(), "A")
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrNotFound)

	_, err = repo.Get(context.Background(), "B")
	require.ErrorIs(t, err, errBasis)
	require.Contains(t, err.Error(), "memindai antrean akseptasi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideRejectsUnknownDecisionWithoutDatabase(t *testing.T) {
	db, mock := newMock(t)
	_, err := NewRepo(db).Decide(context.Background(), "A", "x", "B", time.Now())
	require.ErrorIs(t, err, inboxacceptopenprotection.ErrUnknownDecision)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideApproveAppliesLossDateInOneTransaction(t *testing.T) {
	db, mock := newMock(t)
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(reDecide).WithArgs("1", "PETUGAS", at, "OPCN.26.0004").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(reGet).WithArgs("OPCN.26.0004").
		WillReturnRows(detailRow("OPCN.26.0004", "7", "2026-08-03", "2026-08-05", "1", " ref-1 "))
	mock.ExpectExec(reApply).WithArgs(time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC), "REF-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	saved, err := NewRepo(db).Decide(context.Background(), "opcn.26.0004",
		inboxacceptopenprotection.DecisionApprove, " PETUGAS ", at)
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0004", saved.Number)
	require.True(t, saved.Approved())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideRejectSkipsClaimUpdate(t *testing.T) {
	db, mock := newMock(t)
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(reDecide).WithArgs("2", "P", at, "A").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(reGet).WithArgs("A").
		WillReturnRows(detailRow("A", "7", "2026-08-03", "2026-08-05", "2", "REF"))
	mock.ExpectCommit()

	saved, err := NewRepo(db).Decide(context.Background(), "A", inboxacceptopenprotection.DecisionReject, "P", at)
	require.NoError(t, err)
	require.Equal(t, inboxacceptopenprotection.AcceptRejected, saved.AcceptStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideNothingTouchedIsExplained(t *testing.T) {
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		rows *sqlmock.Rows
		want error
	}{
		{"tidak ada", sqlmock.NewRows(detailCol), inboxacceptopenprotection.ErrNotFound},
		{"sudah diputuskan", detailRow("A", "1", "", "", "1", "R"), inboxacceptopenprotection.ErrAlreadyDecided},
		{"belum lengkap", detailRow("A", "1", "", "", "", "R"), inboxacceptopenprotection.ErrIncomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectBegin()
			mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(reGet).WithArgs("A").WillReturnRows(c.rows)
			mock.ExpectRollback()

			_, err := NewRepo(db).Decide(context.Background(), "A", inboxacceptopenprotection.DecisionApprove, "P", at)
			require.ErrorIs(t, err, c.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDecideFailures(t *testing.T) {
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	approve := inboxacceptopenprotection.DecisionApprove

	t.Run("begin", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBasis)
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "membuka transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnError(errBasis)
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "menyimpan keputusan akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewErrorResult(errBasis))
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "jumlah baris tersentuh")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("read in tx", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnError(errBasis)
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no claim reference", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(detailRow("A", "7", "", "2026-08-05", "1", ""))
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("apply exec", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(detailRow("A", "7", "", "2026-08-05", "1", "R"))
		mock.ExpectExec(reApply).WillReturnError(errBasis)
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "menerapkan tanggal kejadian")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("apply rows affected", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(detailRow("A", "7", "", "2026-08-05", "1", "R"))
		mock.ExpectExec(reApply).WillReturnResult(sqlmock.NewErrorResult(errBasis))
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "baris klaim tersentuh")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("claim row missing", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(detailRow("A", "7", "", "2026-08-05", "1", "R"))
		mock.ExpectExec(reApply).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(detailRow("A", "1", "", "", "1", "R"))
		mock.ExpectCommit().WillReturnError(errBasis)
		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "menyimpan keputusan akseptasi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGroupsOf(t *testing.T) {
	db, mock := newMock(t)
	repo := NewGroupRepo(db)

	// Login kosong tidak menembak basis data.
	groups, err := repo.GroupsOf(context.Background(), "  ")
	require.NoError(t, err)
	require.Nil(t, groups)

	mock.ExpectQuery(reGroups).WithArgs("LOGIN").
		WillReturnRows(sqlmock.NewRows([]string{"GROUP_ID"}).
			AddRow(" CaseManager ").AddRow(nil).AddRow("  ").AddRow("PncCollection"))

	groups, err = repo.GroupsOf(context.Background(), " LOGIN ")
	require.NoError(t, err)
	require.Equal(t, []string{"CaseManager", "PncCollection"}, groups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGroupsOfErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reGroups).WillReturnError(errBasis)
		_, err := NewGroupRepo(db).GroupsOf(context.Background(), "L")
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "membaca access group login")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reGroups).WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow("x", "y"))
		_, err := NewGroupRepo(db).GroupsOf(context.Background(), "L")
		require.Error(t, err)
		require.Contains(t, err.Error(), "memindai access group")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows err", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(reGroups).WillReturnRows(sqlmock.NewRows([]string{"g"}).AddRow("x").RowError(0, errBasis))
		_, err := NewGroupRepo(db).GroupsOf(context.Background(), "L")
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "membaca baris access group")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxacceptopenprotection/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { _ = query("tidak_ada") })
}

func TestSplitByNameDropsCommentsAndEmptyBodies(t *testing.T) {
	got := splitByName("-- kepala\nSELECT 0\n-- name: a\n-- penjelas\nSELECT 1\n\n-- name: kosong\n-- name: b\nSELECT 2\n")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}

func TestSearchArgsAndEscape(t *testing.T) {
	require.Equal(t, []any{nil, nil, nil, nil}, searchArgs("   "))
	require.Equal(t, []any{"50%_X", "%50X%", "%50X%", "%50X%"}, searchArgs(" 50%_x "))
}

// TestDecideMenerapkanPenyebabKerugian menjaga jalur tipe '8' SAMPAI KE BASIS DATA.
//
// Yang diperiksa bukan hanya "ada UPDATE", melainkan NILAI yang dikirim ke kelima penanda —
// karena kesalahan yang paling mungkin di sini adalah urutan argumen yang tertukar, dan
// akibatnya adalah perubahan yang diterapkan ke baris coverage yang salah.
func TestDecideMenerapkanPenyebabKerugian(t *testing.T) {
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)

	db, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(reGet).
		WillReturnRows(detailRowCOL("A", "8", "12001", "12002", "1", "R", " 1 ", " 3 "))

	// Deskripsinya dicari LEBIH DULU, dari master — bukan dibawa dari permintaan.
	mock.ExpectQuery(reDescribe).WithArgs("12002").
		WillReturnRows(sqlmock.NewRows([]string{"DESCRIPTION"}).AddRow(" KEBAKARAN "))

	// Kelima argumen, berurutan: kode baru, deskripsi, CLAIMID, OBJECTID, OBJECTCOVERAGEID.
	mock.ExpectExec(reApplyCOL).
		WithArgs("12002", "KEBAKARAN", "R", "1", "3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := NewRepo(db).Decide(
		context.Background(), "A", inboxacceptopenprotection.DecisionApprove, "P", at)
	require.NoError(t, err)
	require.Equal(t, "12002", got.Change.CauseOfLossAfter)
	require.Equal(t, "1", got.Change.ObjectID)
	require.Equal(t, "3", got.Change.ObjectCoverageID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDecideTidakMenerapkanPenyebabKerugian menjaga keadaan yang TIDAK boleh menyentuh
// coverage.
//
// Ketiganya berakhir COMMIT, bukan rollback: persetujuannya tetap sah. Inilah yang membuat
// seluruh antrean warisan tipe '8' tetap dapat diputuskan.
func TestDecideTidakMenerapkanPenyebabKerugian(t *testing.T) {
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)

	kasus := map[string]struct {
		rows *sqlmock.Rows
		d    inboxacceptopenprotection.Decision
	}{
		"ditolak": {
			detailRowCOL("A", "8", "12001", "12002", "1", "R", "1", "3"),
			inboxacceptopenprotection.DecisionReject,
		},
		"warisan tanpa sasaran": {
			detailRowCOL("A", "8", "12001", "12002", "1", "R", nil, nil),
			inboxacceptopenprotection.DecisionApprove,
		},
		"warisan tanpa kode baru": {
			detailRowCOL("A", "8", "12001", "", "1", "R", "1", "3"),
			inboxacceptopenprotection.DecisionApprove,
		},
	}

	for nama, c := range kasus {
		t.Run(nama, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectBegin()
			mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectQuery(reGet).WillReturnRows(c.rows)
			// Tanpa ExpectQuery/ExpectExec apa pun di antaranya: pemanggilan yang tidak
			// diharapkan akan menggagalkan uji ini, dan itulah yang dijaga.
			mock.ExpectCommit()

			_, err := NewRepo(db).Decide(context.Background(), "A", c.d, "P", at)
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestDecideGagalMenerapkanPenyebabKerugian menjaga setiap jalur gagal MEMBATALKAN keputusan.
func TestDecideGagalMenerapkanPenyebabKerugian(t *testing.T) {
	at := time.Date(2026, 9, 25, 2, 0, 0, 0, time.UTC)
	approve := inboxacceptopenprotection.DecisionApprove
	baris := func() *sqlmock.Rows {
		return detailRowCOL("A", "8", "12001", "12002", "1", "R", "1", "3")
	}

	t.Run("kode tidak ada di master", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(baris())
		mock.ExpectQuery(reDescribe).WillReturnRows(sqlmock.NewRows([]string{"DESCRIPTION"}))
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrUnknownCauseOfLoss)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// Barisnya ADA tetapi deskripsinya kosong. Menuliskannya akan menghapus nama penyebab
	// kerugian pada coverage tanpa menghapus kodenya.
	t.Run("deskripsi kosong", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(baris())
		mock.ExpectQuery(reDescribe).
			WillReturnRows(sqlmock.NewRows([]string{"DESCRIPTION"}).AddRow("   "))
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrUnknownCauseOfLoss)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("master gagal dibaca", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(baris())
		mock.ExpectQuery(reDescribe).WillReturnError(errBasis)
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "deskripsi penyebab kerugian")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(baris())
		mock.ExpectQuery(reDescribe).
			WillReturnRows(sqlmock.NewRows([]string{"DESCRIPTION"}).AddRow("KEBAKARAN"))
		mock.ExpectExec(reApplyCOL).WillReturnError(errBasis)
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, errBasis)
		require.Contains(t, err.Error(), "menerapkan penyebab kerugian")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// Coverage-nya sudah dibuang dari klaim setelah permintaan diajukan. Sama dengan DOL:
	// keputusannya dibatalkan, bukan disimpan tanpa akibat.
	t.Run("baris coverage sudah tidak ada", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).WillReturnRows(baris())
		mock.ExpectQuery(reDescribe).
			WillReturnRows(sqlmock.NewRows([]string{"DESCRIPTION"}).AddRow("KEBAKARAN"))
		mock.ExpectExec(reApplyCOL).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("proteksi tanpa ID_CLAIM", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(reDecide).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(reGet).
			WillReturnRows(detailRowCOL("A", "8", "12001", "12002", "1", "", "1", "3"))
		mock.ExpectRollback()

		_, err := NewRepo(db).Decide(context.Background(), "A", approve, "P", at)
		require.ErrorIs(t, err, inboxacceptopenprotection.ErrClaimNotSynced)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
