package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxxol"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

// qText mengubah teks kueri yang sudah diekspansi menjadi pola regexp literal.
func qText(text string) string { return "^" + regexp.QuoteMeta(text) + "$" }

var masterColumns = []string{"ID", "NAMA", "TAHUN", "KURS", "TIPE", "STSKOMITE", "KET_KOMITE",
	"PIC", "KET_PIC", "EMAIL_PIC", "MIN_LIMIT"}

var adviceColumns = []string{"NO", "REV", "REAS_ID", "REAS", "LAYER_ID", "LAYER", "TAHUN", "COL",
	"KURS", "SHARE", "LIMIT", "STATUS", "MASTER", "INPUT", "KET_REAS", "KET_APPROVE", "KET_PIC",
	"EMAIL", "NEGARA", "EMAIL_INPUT", "TGL"}

func TestListMasterXOLAttachesBusinessGroups(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows(masterColumns).
		AddRow(" X1 ", " Treaty A ", "2025", 15000.5, "XOL", "1", "ok", "pic", "cat", "p@x", 2000.0).
		AddRow("X2", "Treaty B", "2026", nil, nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectQuery(q("master_business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"MASTER", "GROUP", "NAMA"}).
			AddRow(" X1 ", " 01 ", " Fire ").
			AddRow("X1", "02", "Marine"))

	got, err := repo.ListMasterXOL(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, inboxxol.MasterXOL{
		ID: "X1", Name: "Treaty A", Year: "2025", ExchangeRate: 15000.5, MinLimit: 2000, Type: "XOL",
		CommitteeStatus: "1", CommitteeNote: "ok", PIC: "pic", PICNote: "cat", PICEmail: "p@x",
		BusinessGroups: []inboxxol.BusinessGroup{{ID: "01", Name: "Fire"}, {ID: "02", Name: "Marine"}},
	}, got[0])
	require.Equal(t, "X2", got[1].ID)
	require.Nil(t, got[1].BusinessGroups)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPendingMasterApprovalUsesCommitteeQuery(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("master_pending_committee")).WillReturnRows(sqlmock.NewRows(masterColumns).
		AddRow("X9", "N", "2024", 1.0, "T", "0", "", "", "", "", nil))
	mock.ExpectQuery(q("master_business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"MASTER", "GROUP", "NAMA"}))

	got, err := repo.ListPendingMasterApproval(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "0", got[0].CommitteeStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMastersErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnError(errDB)
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "master_list")
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("X"))
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorContains(t, err, "master_list")
	})
	t.Run("rows err", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows(masterColumns).
			AddRow("X", "", "", 0.0, "", "", "", "", "", "", nil).RowError(0, errDB))
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorIs(t, err, errDB)
	})
	t.Run("business query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows(masterColumns))
		mock.ExpectQuery(q("master_business_list")).WillReturnError(errDB)
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "master_business_list")
	})
	t.Run("business scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows(masterColumns))
		mock.ExpectQuery(q("master_business_list")).WillReturnRows(
			sqlmock.NewRows([]string{"MASTER"}).AddRow("X"))
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorContains(t, err, "master_business_list")
	})
	t.Run("business rows err", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("master_list")).WillReturnRows(sqlmock.NewRows(masterColumns))
		mock.ExpectQuery(q("master_business_list")).WillReturnRows(
			sqlmock.NewRows([]string{"MASTER", "GROUP", "NAMA"}).AddRow("X", "1", "N").RowError(0, errDB))
		_, err := repo.ListMasterXOL(ctx)
		require.ErrorIs(t, err, errDB)
	})
}

func TestSummarizeClaims(t *testing.T) {
	ctx := context.Background()
	filter := inboxxol.ClaimFilter{Year: " 2025 ", BusinessGroupIDs: []string{"01", "02"}}
	text, _, err := expandIDs("claim_summary", []any{"2025"}, filter.BusinessGroupIDs)
	require.NoError(t, err)

	t.Run("empty filter skips query", func(t *testing.T) {
		repo, mock := newMock(t)
		got, err := repo.SummarizeClaims(ctx, inboxxol.ClaimFilter{Year: "2025"})
		require.NoError(t, err)
		require.Nil(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WithArgs("2025", "01", "02").WillReturnRows(
			sqlmock.NewRows([]string{"DOL", "COL", "OS", "AKSEP"}).
				AddRow(" 01/01/2025 ", " Banjir ", 100.0, 50.0).
				AddRow("02/01/2025", "Api", nil, nil))
		got, err := repo.SummarizeClaims(ctx, filter)
		require.NoError(t, err)
		require.Equal(t, []inboxxol.ClaimSummary{
			{LossDate: "01/01/2025", CauseOfLoss: "Banjir", OutstandingValue: 100, AcceptedValue: 50},
			{LossDate: "02/01/2025", CauseOfLoss: "Api"},
		}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnError(errDB)
		_, err := repo.SummarizeClaims(ctx, filter)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "claim_summary")
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnRows(sqlmock.NewRows([]string{"DOL"}).AddRow("x"))
		_, err := repo.SummarizeClaims(ctx, filter)
		require.ErrorContains(t, err, "claim_summary")
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnRows(
			sqlmock.NewRows([]string{"DOL", "COL", "OS", "AKSEP"}).AddRow("a", "b", 1.0, 1.0).RowError(0, errDB))
		_, err := repo.SummarizeClaims(ctx, filter)
		require.ErrorIs(t, err, errDB)
	})
}

func TestBreakdownByBusiness(t *testing.T) {
	ctx := context.Background()
	filter := inboxxol.BreakdownFilter{LossDate: " 01/01/2025 ", CauseOfLoss: " Banjir ",
		BusinessGroupIDs: []string{"01"}}
	text, _, err := expandIDs("breakdown_business", []any{"01/01/2025", "Banjir"}, []string{"01"})
	require.NoError(t, err)
	columns := []string{"NAMA", "ID", "JML", "OS", "AKSEP"}

	t.Run("empty", func(t *testing.T) {
		repo, mock := newMock(t)
		got, err := repo.BreakdownByBusiness(ctx, inboxxol.BreakdownFilter{LossDate: "x", CauseOfLoss: "y"})
		require.NoError(t, err)
		require.Nil(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WithArgs("01/01/2025", "Banjir", "01").WillReturnRows(
			sqlmock.NewRows(columns).AddRow(" Fire ", " 01 ", 3, 10.0, 5.0).AddRow(nil, "02", 1, 0.0, 0.0))
		got, err := repo.BreakdownByBusiness(ctx, filter)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.Equal(t, inboxxol.BusinessBreakdown{BusinessGroup: "Fire", BusinessGroupID: "01",
			ClaimCount: 3, OutstandingValue: 10, AcceptedValue: 5, Source: inboxxol.SourceOwnBusiness}, got[0])
		// Nama kosong diganti penanda tampilan yang sama dengan DisplayName.
		require.Equal(t, inboxxol.BusinessGroup{}.DisplayName(), got[1].BusinessGroup)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnError(errDB)
		_, err := repo.BreakdownByBusiness(ctx, filter)
		require.ErrorIs(t, err, errDB)
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnRows(sqlmock.NewRows([]string{"NAMA"}).AddRow("x"))
		_, err := repo.BreakdownByBusiness(ctx, filter)
		require.ErrorContains(t, err, "breakdown_business")
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(qText(text)).WillReturnRows(
			sqlmock.NewRows(columns).AddRow("a", "b", 1, 1.0, 1.0).RowError(0, errDB))
		_, err := repo.BreakdownByBusiness(ctx, filter)
		require.ErrorIs(t, err, errDB)
	})
}

func TestBreakdownTreatyInward(t *testing.T) {
	ctx := context.Background()
	filter := inboxxol.BreakdownFilter{LossDate: " 01/01/2025 ", CauseOfLoss: " Banjir "}
	columns := []string{"NAMA", "JML", "OS", "AKSEP", "KURS_HILANG"}

	t.Run("empty", func(t *testing.T) {
		repo, mock := newMock(t)
		got, err := repo.BreakdownTreatyInward(ctx, inboxxol.BreakdownFilter{LossDate: "x"})
		require.NoError(t, err)
		require.Nil(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("row uses base currency", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		require.NoError(t, err)
		defer db.Close()
		repo := NewRepoWithBaseCurrency(db, "20002")
		mock.ExpectQuery(q("breakdown_treaty_inward")).WithArgs("01/01/2025", "Banjir", "20002").
			WillReturnRows(sqlmock.NewRows(columns).AddRow(" Treaty In ", 2, 7.5, 3.5, 0))
		got, err := repo.BreakdownTreatyInward(ctx, filter)
		require.NoError(t, err)
		require.Equal(t, []inboxxol.BusinessBreakdown{{BusinessGroup: "Treaty In", ClaimCount: 2,
			OutstandingValue: 7.5, AcceptedValue: 3.5, Source: inboxxol.SourceTreatyInward}}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("zero row without missing rate is dropped", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("breakdown_treaty_inward")).
			WillReturnRows(sqlmock.NewRows(columns).AddRow("T", 0, 0.0, 0.0, 0))
		got, err := repo.BreakdownTreatyInward(ctx, filter)
		require.NoError(t, err)
		require.Nil(t, got)
	})
	t.Run("zero row with missing rate is kept", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("breakdown_treaty_inward")).
			WillReturnRows(sqlmock.NewRows(columns).AddRow("T", 0, 0.0, 0.0, 1))
		got, err := repo.BreakdownTreatyInward(ctx, filter)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.True(t, got[0].RateMissing)
	})
	t.Run("no rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("breakdown_treaty_inward")).WillReturnRows(sqlmock.NewRows(columns))
		got, err := repo.BreakdownTreatyInward(ctx, filter)
		require.NoError(t, err)
		require.Nil(t, got)
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("breakdown_treaty_inward")).WillReturnError(errDB)
		_, err := repo.BreakdownTreatyInward(ctx, filter)
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, "breakdown_treaty_inward")
	})
}

func adviceRow(rows *sqlmock.Rows, number string) *sqlmock.Rows {
	return rows.AddRow(number, " 1 ", "R1", " Reas ", "L1", "Layer", "2025", "Banjir", 15000.0, 25.5,
		"10jt", "0", "X1", "admin", "ket", "apr", "pic", "e@x", "ID", "i@x", "01/02/2025")
}

func TestSearchAdvice(t *testing.T) {
	ctx := context.Background()

	t.Run("unknown type", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.SearchAdvice(ctx, inboxxol.AdviceFilter{Type: "XXX"})
		require.ErrorContains(t, err, "XXX")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	for _, tc := range []struct {
		kind inboxxol.AdviceType
		name string
	}{{inboxxol.AdvicePLA, "advice_list_pla"}, {inboxxol.AdviceDLA, "advice_list_dla"}} {
		t.Run(string(tc.kind), func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(q(tc.name)).WithArgs("2025", "Banjir").
				WillReturnRows(adviceRow(sqlmock.NewRows(adviceColumns), " PLA-1 "))
			got, err := repo.SearchAdvice(ctx, inboxxol.AdviceFilter{Year: " 2025 ", CauseOfLoss: " Banjir ", Type: tc.kind})
			require.NoError(t, err)
			require.Equal(t, []inboxxol.Advice{{
				Number: "PLA-1", Revision: "1", ReinsurerID: "R1", ReinsurerName: "Reas", LayerID: "L1",
				LayerName: "Layer", Year: "2025", CauseOfLoss: "Banjir", ExchangeRate: 15000, SharePercent: 25.5,
				Limit: "10jt", ApprovalStatus: "0", MasterID: "X1", InputBy: "admin", Remark: "ket",
				ApprovalNote: "apr", PICNote: "pic", Email: "e@x", Country: "ID", InputByEmail: "i@x",
				IssuedOn: "01/02/2025", Type: tc.kind,
			}}, got)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
	filter := inboxxol.AdviceFilter{Year: "2025", CauseOfLoss: "B", Type: inboxxol.AdvicePLA}
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("advice_list_pla")).WillReturnError(errDB)
		_, err := repo.SearchAdvice(ctx, filter)
		require.ErrorIs(t, err, errDB)
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("advice_list_pla")).WillReturnRows(sqlmock.NewRows([]string{"NO"}).AddRow("x"))
		_, err := repo.SearchAdvice(ctx, filter)
		require.ErrorContains(t, err, "advice_list_pla")
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("advice_list_pla")).
			WillReturnRows(adviceRow(sqlmock.NewRows(adviceColumns), "x").RowError(0, errDB))
		_, err := repo.SearchAdvice(ctx, filter)
		require.ErrorIs(t, err, errDB)
	})
}

func TestListPendingAdviceApproval(t *testing.T) {
	ctx := context.Background()
	columns := []string{"TAHUN", "COL", "TIPE", "TGL"}

	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("approval_advice_queue")).WillReturnRows(sqlmock.NewRows(columns).
			AddRow(" 2025 ", " Banjir ", "PLA", " 01/01 ").AddRow("2024", "Api", "ZZZ", ""))
		got, err := repo.ListPendingAdviceApproval(ctx)
		require.NoError(t, err)
		require.Equal(t, inboxxol.ApprovalItem{Year: "2025", CauseOfLoss: "Banjir",
			Type: inboxxol.AdvicePLA, LastInsertedAt: "01/01"}, got[0])
		parsed, _ := inboxxol.ParseAdviceType("ZZZ")
		require.Equal(t, parsed, got[1].Type)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("approval_advice_queue")).WillReturnError(errDB)
		_, err := repo.ListPendingAdviceApproval(ctx)
		require.ErrorIs(t, err, errDB)
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("approval_advice_queue")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
		_, err := repo.ListPendingAdviceApproval(ctx)
		require.ErrorContains(t, err, "approval_advice_queue")
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("approval_advice_queue")).WillReturnRows(
			sqlmock.NewRows(columns).AddRow("a", "b", "PLA", "d").RowError(0, errDB))
		_, err := repo.ListPendingAdviceApproval(ctx)
		require.ErrorIs(t, err, errDB)
	})
}

func TestListCauseOfLoss(t *testing.T) {
	ctx := context.Background()
	columns := []string{"ID", "DESKRIPSI"}

	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("cause_of_loss_list")).WillReturnRows(
			sqlmock.NewRows(columns).AddRow(" 12002 ", " Banjir "))
		got, err := repo.ListCauseOfLoss(ctx)
		require.NoError(t, err)
		require.Equal(t, []inboxxol.CauseOfLoss{{ID: "12002", Description: "Banjir"}}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("cause_of_loss_list")).WillReturnError(errDB)
		_, err := repo.ListCauseOfLoss(ctx)
		require.ErrorIs(t, err, errDB)
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("cause_of_loss_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("x"))
		_, err := repo.ListCauseOfLoss(ctx)
		require.ErrorContains(t, err, "cause_of_loss_list")
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("cause_of_loss_list")).WillReturnRows(
			sqlmock.NewRows(columns).AddRow("a", "b").RowError(0, errDB))
		_, err := repo.ListCauseOfLoss(ctx)
		require.ErrorIs(t, err, errDB)
	})
}
