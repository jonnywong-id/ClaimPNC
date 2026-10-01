package sqlstore

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/detailpenyebab"
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
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var detailColumns = []string{"D_COL_ID", "OLD", "M_COL_ID", "DESCRIPTION", "LOSS_CODE", "STS_AKTIF", "LABEL"}

func detailRows() *sqlmock.Rows { return sqlmock.NewRows(detailColumns) }

func TestListSendsFilterAndMapsRows(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("detail_list")).
		WithArgs("ban_jir", `%BAN\_JIR%`, `%BAN\_JIR%`, `%BAN\_JIR%`, "M1", "M1", nil, "").
		WillReturnRows(detailRows().AddRow(" D1 ", " O1 ", " M1 ", " Banjir ", " L1 ", " 1 ", " Master 1 "))
	got, err := repo.List(context.Background(), detailpenyebab.Filter{Keyword: " ban_jir ", MasterID: " M1 "})
	require.NoError(t, err)
	require.Equal(t, []detailpenyebab.CauseOfLossDetail{{ID: "D1", LegacyID: "O1", MasterID: "M1",
		MasterLabel: "Master 1", Description: "Banjir", LossCode: "L1", Active: "1"}}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("detail_list")).WillReturnError(errDB)
	_, err := repo.List(ctx, detailpenyebab.Filter{})
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("detail_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(ctx, detailpenyebab.Filter{})
	require.ErrorContains(t, err, "membaca baris")

	mock.ExpectQuery(q("detail_list")).WillReturnRows(
		detailRows().AddRow("a", "b", "c", "d", "e", "f", "g").RowError(0, errDB))
	_, err = repo.List(ctx, detailpenyebab.Filter{})
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAttachesBusiness(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("detail_get")).WithArgs("D1").
		WillReturnRows(detailRows().AddRow("D1", "", "M1", "Banjir", "", "1", "Master"))
	mock.ExpectQuery(q("detail_business_list")).WithArgs("D1").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" 01 ", " Fire ").AddRow(nil, nil))
	got, err := repo.Get(context.Background(), " D1 ")
	require.NoError(t, err)
	require.Equal(t, "D1", got.ID)
	require.Equal(t, []detailpenyebab.Business{{ID: "01", Name: "Fire"}, {}}, got.Business)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMock(t)
	_, err := repo.Get(ctx, " ")
	require.ErrorIs(t, err, detailpenyebab.ErrNotFound)

	mock.ExpectQuery(q("detail_get")).WillReturnError(errDB)
	_, err = repo.Get(ctx, "D1")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("detail_get")).WillReturnRows(detailRows())
	_, err = repo.Get(ctx, "D1")
	require.ErrorIs(t, err, detailpenyebab.ErrNotFound)

	mock.ExpectQuery(q("detail_get")).WillReturnRows(
		detailRows().AddRow("a", "b", "c", "d", "e", "f", "g").RowError(0, errDB))
	_, err = repo.Get(ctx, "D1")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("detail_get")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Get(ctx, "D1")
	require.ErrorContains(t, err, "membaca baris")

	found := func() {
		mock.ExpectQuery(q("detail_get")).WillReturnRows(detailRows().AddRow("D1", "", "", "", "", "", ""))
	}
	found()
	mock.ExpectQuery(q("detail_business_list")).WillReturnError(errDB)
	_, err = repo.Get(ctx, "D1")
	require.ErrorIs(t, err, errDB)

	found()
	mock.ExpectQuery(q("detail_business_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Get(ctx, "D1")
	require.ErrorContains(t, err, "membaca lini bisnis")

	found()
	mock.ExpectQuery(q("detail_business_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.Get(ctx, "D1")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func sampleInput() detailpenyebab.Input {
	return detailpenyebab.Input{LegacyID: "O1", MasterID: "M1", Description: "Banjir", LossCode: "L",
		Active: "1", Business: []detailpenyebab.Business{{ID: "01", Name: "Fire"}}}
}

func TestInsertBuildsIDFromSiteAndSequence(t *testing.T) {
	repo, mock := newMock(t)
	in := sampleInput()
	fresh := detailpenyebab.CauseOfLossDetail{ID: "7" + "0042", LegacyID: in.LegacyID, MasterID: in.MasterID,
		Description: in.Description, LossCode: in.LossCode, Active: in.Active, Business: in.Business}
	payload, err := json.Marshal(newDocument(fresh))
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"SITE"}).AddRow(" 7 "))
	mock.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))
	mock.ExpectQuery(q("detail_exists")).WithArgs("70042").WillReturnRows(sqlmock.NewRows([]string{"X"}))
	mock.ExpectExec(q("detail_insert")).WithArgs("70042", string(payload)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("master_get")).WithArgs("M1").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "LABEL"}).AddRow("M1", " M1 - Alam "))

	got, err := repo.Insert(context.Background(), in)
	require.NoError(t, err)
	require.Equal(t, "70042", got.ID)
	require.Equal(t, "M1 - Alam", got.MasterLabel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	ctx := context.Background()
	begin := func(m sqlmock.Sqlmock) { m.ExpectBegin() }
	site := func(m sqlmock.Sqlmock) {
		begin(m)
		m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow("7"))
	}
	sequence := func(m sqlmock.Sqlmock) {
		site(m)
		m.ExpectQuery(q("detail_next_sequence")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	}
	free := func(m sqlmock.Sqlmock) {
		sequence(m)
		m.ExpectQuery(q("detail_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	}
	inserted := func(m sqlmock.Sqlmock) {
		free(m)
		m.ExpectExec(q("detail_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	}

	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  error
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "membuka transaksi"},
		{"site missing", func(m sqlmock.Sqlmock) {
			begin(m)
			m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"S"}))
		}, nil, "tidak memuat baris"},
		{"site error", func(m sqlmock.Sqlmock) {
			begin(m)
			m.ExpectQuery(q("detail_site")).WillReturnError(errDB)
		}, errDB, "membaca kode situs"},
		{"site blank", func(m sqlmock.Sqlmock) {
			begin(m)
			m.ExpectQuery(q("detail_site")).WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow(" "))
		}, nil, "kode situs pada POOLDATA.M_SITE_DATABASE kosong"},
		{"sequence", func(m sqlmock.Sqlmock) {
			site(m)
			m.ExpectQuery(q("detail_next_sequence")).WillReturnError(errDB)
		}, errDB, "nomor urut"},
		{"exists error", func(m sqlmock.Sqlmock) {
			sequence(m)
			m.ExpectQuery(q("detail_exists")).WillReturnError(errDB)
		}, errDB, "memeriksa ID"},
		{"exists rows error", func(m sqlmock.Sqlmock) {
			sequence(m)
			m.ExpectQuery(q("detail_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}).RowError(0, errDB).AddRow("1"))
		}, errDB, "memeriksa ID"},
		{"taken", func(m sqlmock.Sqlmock) {
			sequence(m)
			m.ExpectQuery(q("detail_exists")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow("70001"))
		}, detailpenyebab.ErrIDTaken, ""},
		{"insert", func(m sqlmock.Sqlmock) {
			free(m)
			m.ExpectExec(q("detail_insert")).WillReturnError(errDB)
		}, errDB, "menyisipkan baris"},
		{"commit", func(m sqlmock.Sqlmock) {
			inserted(m)
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menyimpan transaksi"},
		{"label", func(m sqlmock.Sqlmock) {
			inserted(m)
			m.ExpectCommit()
			m.ExpectQuery(q("master_get")).WillReturnError(errDB)
		}, errDB, "sebutan master"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.Insert(ctx, sampleInput())
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			if tc.text != "" {
				require.ErrorContains(t, err, tc.text)
			}
		})
	}
}

func TestUpdateMergesStoredDocument(t *testing.T) {
	repo, mock := newMock(t)
	in := sampleInput()
	in.MasterID = ""
	stored := `{"EXTRA":"tetap","DESCRIPTION":"lama"}`
	expected, err := mergeDocument(stored, detailpenyebab.CauseOfLossDetail{ID: "D1", LegacyID: in.LegacyID,
		Description: in.Description, LossCode: in.LossCode, Active: in.Active, Business: in.Business})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_document")).WithArgs("D1").WillReturnRows(sqlmock.NewRows([]string{"DOC"}).AddRow(stored))
	mock.ExpectExec(q("detail_update")).WithArgs(expected, "D1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	got, err := repo.Update(context.Background(), " D1 ", in)
	require.NoError(t, err)
	require.Equal(t, "D1", got.ID)
	// Master kosong tidak membaca sebutan master sama sekali.
	require.Empty(t, got.MasterLabel)
	require.Contains(t, expected, `"EXTRA":"tetap"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	ctx := context.Background()
	document := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(q("detail_document")).WillReturnRows(sqlmock.NewRows([]string{"DOC"}).AddRow(nil))
	}

	repo, _ := newMock(t)
	_, err := repo.Update(ctx, " ", sampleInput())
	require.ErrorIs(t, err, detailpenyebab.ErrNotFound)

	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  error
		text  string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "membuka transaksi"},
		{"document missing", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_document")).WillReturnRows(sqlmock.NewRows([]string{"DOC"}))
		}, detailpenyebab.ErrNotFound, ""},
		{"document error", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("detail_document")).WillReturnError(errDB)
		}, errDB, "membaca dokumen"},
		{"update error", func(m sqlmock.Sqlmock) {
			document(m)
			m.ExpectExec(q("detail_update")).WillReturnError(errDB)
		}, errDB, "memperbarui baris"},
		{"no row changed", func(m sqlmock.Sqlmock) {
			document(m)
			m.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 0))
		}, detailpenyebab.ErrNotFound, ""},
		{"commit", func(m sqlmock.Sqlmock) {
			document(m)
			m.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menyimpan transaksi"},
		{"label", func(m sqlmock.Sqlmock) {
			document(m)
			m.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit()
			m.ExpectQuery(q("master_get")).WillReturnError(errDB)
		}, errDB, "sebutan master"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.Update(ctx, "D1", sampleInput())
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			if tc.text != "" {
				require.ErrorContains(t, err, tc.text)
			}
		})
	}
}

// RowsAffected yang gagal dibaca TIDAK dianggap "tidak ditemukan": penyimpanan berlanjut.
func TestUpdateIgnoresUnreadableRowsAffected(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("detail_document")).WillReturnRows(sqlmock.NewRows([]string{"DOC"}).AddRow("{}"))
	mock.ExpectExec(q("detail_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	mock.ExpectCommit()
	mock.ExpectQuery(q("master_get")).WithArgs("M1").WillReturnRows(sqlmock.NewRows([]string{"ID", "LABEL"}))
	got, err := repo.Update(context.Background(), "D1", sampleInput())
	require.NoError(t, err)
	// Master tidak ditemukan → sebutan kosong, bukan galat.
	require.Empty(t, got.MasterLabel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchMasterAndBusiness(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectQuery(q("master_search")).WithArgs("%BAN%", "ban").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "LABEL"}).AddRow(" M1 ", " Banjir "))
	masters, err := repo.SearchMaster(ctx, " ban ")
	require.NoError(t, err)
	require.Equal(t, []detailpenyebab.MasterOption{{ID: "M1", Label: "Banjir"}}, masters)

	mock.ExpectQuery(q("business_search")).WithArgs("%", "").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" 01 ", " Fire "))
	business, err := repo.SearchBusiness(ctx, "")
	require.NoError(t, err)
	require.Equal(t, []detailpenyebab.Business{{ID: "01", Name: "Fire"}}, business)

	for _, name := range []string{"master_search", "business_search"} {
		call := func() error {
			if name == "master_search" {
				_, err := repo.SearchMaster(ctx, "x")
				return err
			}
			_, err := repo.SearchBusiness(ctx, "x")
			return err
		}
		mock.ExpectQuery(q(name)).WillReturnError(errDB)
		require.ErrorIs(t, call(), errDB)
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
		require.Error(t, call())
		mock.ExpectQuery(q(name)).WillReturnRows(sqlmock.NewRows([]string{"ID", "N"}).AddRow("1", "a").RowError(0, errDB))
		require.ErrorIs(t, call(), errDB)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTablesReportsEachObject(t *testing.T) {
	repo, mock := newMock(t)
	mock.MatchExpectationsInOrder(false)
	mock.ExpectQuery(q("detail_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	mock.ExpectQuery(q("detail_check_writable")).WillReturnError(errDB)
	mock.ExpectQuery(q("detail_check_business_view")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	mock.ExpectQuery(q("master_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))
	mock.ExpectQuery(q("business_check_table")).WillReturnRows(sqlmock.NewRows([]string{"A"}))

	result := repo.CheckTables(context.Background())
	require.Len(t, result, 5)
	require.ErrorIs(t, result["POOLDATA.D_CAUSE_OF_LOSS"], errDB)
	require.NoError(t, result["POOLDATA.V_D_CAUSE_OF_LOSS"])
	require.NoError(t, result["POOLDATA.BUSINESS"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJSONTextRejectsObjects(t *testing.T) {
	var value jsonText
	require.NoError(t, value.UnmarshalJSON([]byte(" null ")))
	require.Equal(t, jsonText(""), value)
	require.NoError(t, value.UnmarshalJSON([]byte("true")))
	require.Equal(t, jsonText("true"), value)
	require.Error(t, value.UnmarshalJSON([]byte(`"tidak tertutup`)))
	require.ErrorContains(t, value.UnmarshalJSON([]byte(`{"a":1}`)), "bukan teks, angka, maupun boolean")
}
