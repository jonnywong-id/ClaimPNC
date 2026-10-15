package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterxol"
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

var masterColumns = []string{"ID", "NAMA", "TAHUN", "KURS", "TYPEXOL", "PIC", "STSKOMITE",
	"KOMITE", "REMARKPIC", "REMARKKOMITE"}

func masterRows() *sqlmock.Rows { return sqlmock.NewRows(masterColumns) }

// expectGet menyiapkan keempat kueri pembaca satu induk: induk, bisnis, layer, reas.
func expectGet(mock sqlmock.Sqlmock, id string) {
	mock.ExpectQuery(q("xol_get")).WithArgs(id).WillReturnRows(masterRows().
		AddRow(id, " Treaty ", "2025", 15000, "1", "PIC1", "0", "K", "rp", "rk"))
	mock.ExpectQuery(q("xol_business_list")).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("01", "Fire").AddRow(nil, "TREATY INWARD"))
	mock.ExpectQuery(q("xol_layer_list")).WithArgs(id).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA", "LIMIT", "EXCESS", "CONV"}).
			AddRow("10010", "L2", 3, 1, 0).
			AddRow(" 1009 ", "L1", 2, 1, 99))
	mock.ExpectQuery(q("xol_reas_list")).WithArgs("1009").WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA", "SHARE"}).AddRow("R1", "Reas 1", 60))
	mock.ExpectQuery(q("xol_reas_list")).WithArgs("10010").WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA", "SHARE"}))
}

func TestListSortsNumerically(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_list")).WillReturnRows(masterRows().
		AddRow("10010", "B", nil, nil, nil, nil, nil, nil, nil, nil).
		AddRow("1009", " A ", "2024", 10, "2", "p", "1", "k", "", "").
		AddRow("ABC", "C", nil, nil, nil, nil, nil, nil, nil, nil))
	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"1009", "10010", "ABC"}, []string{got[0].ID, got[1].ID, got[2].ID})
	require.Equal(t, "A", got[0].Name)
	require.Equal(t, masterxol.Amount(10), got[0].ExchangeRate)
	require.Equal(t, masterxol.TypeAccident, got[0].Type)
	require.Equal(t, masterxol.CommitteeApproved, got[0].CommitteeStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_list")).WillReturnError(errDB)
	_, err := repo.List(ctx)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("xol_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.List(ctx)
	require.ErrorContains(t, err, "membaca baris master XOL")

	mock.ExpectQuery(q("xol_list")).WillReturnRows(masterRows().
		AddRow("1", "", "", 0, "", "", "", "", "", "").RowError(0, errDB))
	_, err = repo.List(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAssemblesChildren(t *testing.T) {
	repo, mock := newMock(t)
	expectGet(mock, "10001")
	got, err := repo.Get(context.Background(), " 10001 ")
	require.NoError(t, err)
	require.Equal(t, "Treaty", got.Name)
	require.Equal(t, []masterxol.Business{{ID: "01", Name: "Fire"}, {ID: "", Name: "TREATY INWARD"}}, got.Business)
	require.Len(t, got.Layer, 2)
	require.Equal(t, "1009", got.Layer[0].ID)
	// ConvertedLimit dihitung ulang dari Limit × kurs, bukan nilai tersimpan.
	require.Equal(t, masterxol.Amount(30000), got.Layer[0].ConvertedLimit)
	require.Equal(t, []masterxol.Reinsurer{{ID: "R1", Name: "Reas 1", Share: 60}}, got.Layer[0].Reinsurer)
	require.Empty(t, got.Layer[1].Reinsurer)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("not found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("xol_get")).WillReturnRows(masterRows())
		_, err := repo.Get(ctx, "1")
		require.ErrorIs(t, err, masterxol.ErrNotFound)
	})
	t.Run("master error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("xol_get")).WillReturnError(errDB)
		_, err := repo.Get(ctx, "1")
		require.ErrorIs(t, err, errDB)
	})

	master := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(q("xol_get")).WillReturnRows(masterRows().
			AddRow("1", "", "", 1, "", "", "", "", "", ""))
	}
	business := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(q("xol_business_list")).WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}))
	}
	layerColumns := []string{"ID", "NAMA", "LIMIT", "EXCESS", "CONV"}
	cases := map[string]func(sqlmock.Sqlmock){
		"business query": func(m sqlmock.Sqlmock) {
			master(m)
			m.ExpectQuery(q("xol_business_list")).WillReturnError(errDB)
		},
		"business scan": func(m sqlmock.Sqlmock) {
			master(m)
			m.ExpectQuery(q("xol_business_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("x"))
		},
		"business rows": func(m sqlmock.Sqlmock) {
			master(m)
			m.ExpectQuery(q("xol_business_list")).WillReturnRows(
				sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "a").RowError(0, errDB))
		},
		"layer query": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnError(errDB)
		},
		"layer scan": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("x"))
		},
		"layer rows": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnRows(
				sqlmock.NewRows(layerColumns).AddRow("1", "a", 1, 1, 1).RowError(0, errDB))
		},
		"reas query": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnRows(sqlmock.NewRows(layerColumns).AddRow("1", "a", 1, 1, 1))
			m.ExpectQuery(q("xol_reas_list")).WillReturnError(errDB)
		},
		"reas scan": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnRows(sqlmock.NewRows(layerColumns).AddRow("1", "a", 1, 1, 1))
			m.ExpectQuery(q("xol_reas_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("x"))
		},
		"reas rows": func(m sqlmock.Sqlmock) {
			master(m)
			business(m)
			m.ExpectQuery(q("xol_layer_list")).WillReturnRows(sqlmock.NewRows(layerColumns).AddRow("1", "a", 1, 1, 1))
			m.ExpectQuery(q("xol_reas_list")).WillReturnRows(
				sqlmock.NewRows([]string{"ID", "NAMA", "SHARE"}).AddRow("r", "n", 1).RowError(0, errDB))
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			repo, mock := newMock(t)
			setup(mock)
			_, err := repo.Get(ctx, "1")
			require.Error(t, err)
			require.ErrorContains(t, err, "masterxol/sqlstore")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSaveInsertsNewMasterInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("xol_lock_master_ids")).WillReturnRows(
		sqlmock.NewRows([]string{"ID"}).AddRow(" 10001 ").AddRow("10003"))
	mock.ExpectExec(q("xol_insert_master")).WithArgs("10004", "Treaty", "2025", int64(10), nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Bisnis "01" sudah ada → dilewati; bisnis kosong belum ada → disisipkan dengan NULL.
	mock.ExpectQuery(q("xol_business_count")).WithArgs("10004", "01").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectQuery(q("xol_business_count")).WithArgs("10004", nil).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
	mock.ExpectExec(q("xol_insert_business")).WithArgs("10004", "TREATY INWARD", nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Satu layer baru dan satu layer lama.
	mock.ExpectQuery(q("xol_lock_layer_ids")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("20005"))
	mock.ExpectExec(q("xol_insert_layer")).WithArgs("10004", "20006", "L-new", int64(2), int64(1), int64(20)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("xol_reas_count")).WithArgs("20006", "R1").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
	mock.ExpectExec(q("xol_insert_reas")).WithArgs("20006", "Reas", "R1", int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("xol_update_layer")).WithArgs(nil, int64(3), int64(0), int64(30), "20001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("xol_reas_count")).WithArgs("20001", "R2").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	mock.ExpectExec(q("xol_update_reas")).WithArgs("Reas 2", int64(50), "20001", "R2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectGet(mock, "10004")

	got, err := repo.Save(context.Background(), masterxol.Master{
		Name: " Treaty ", Year: "2025", ExchangeRate: 10,
		Business: []masterxol.Business{{ID: "01", Name: "Fire"}, {Name: "TREATY INWARD"}},
		Layer: []masterxol.Layer{
			{Name: "L-new", Limit: 2, Excess: 1, Reinsurer: []masterxol.Reinsurer{{ID: "R1", Name: "Reas", Share: 100}}},
			{ID: "20001", Limit: 3, Reinsurer: []masterxol.Reinsurer{{ID: "R2", Name: "Reas 2", Share: 50}}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "10004", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveUpdatesExistingMaster(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("xol_update_master")).WithArgs("N", nil, int64(5), "3", "10002").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	expectGet(mock, "10002")
	got, err := repo.Save(context.Background(), masterxol.Master{ID: " 10002 ", Name: "N", ExchangeRate: 5, Type: masterxol.TypeMarine})
	require.NoError(t, err)
	require.Equal(t, "10002", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveFailuresRollBack(t *testing.T) {
	ctx := context.Background()
	newMaster := masterxol.Master{Name: "N"}
	oldMaster := masterxol.Master{ID: "10002", Name: "N"}
	withBusiness := masterxol.Master{ID: "10002", Business: []masterxol.Business{{ID: "01"}}}
	withNewLayer := masterxol.Master{ID: "10002", Layer: []masterxol.Layer{{Name: "L"}}}
	withOldLayer := masterxol.Master{ID: "10002", Layer: []masterxol.Layer{{ID: "20001"}}}
	withReas := masterxol.Master{ID: "10002", Layer: []masterxol.Layer{{ID: "20001",
		Reinsurer: []masterxol.Reinsurer{{ID: "R1"}}}}}
	updated := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectExec(q("xol_update_master")).WillReturnResult(sqlmock.NewResult(0, 1))
	}

	cases := []struct {
		name   string
		master masterxol.Master
		setup  func(sqlmock.Sqlmock)
		want   error
		text   string
	}{
		{"begin", newMaster, func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errDB) }, errDB, "memulai transaksi"},
		{"lock master ids", newMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("xol_lock_master_ids")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "mengunci daftar nomor"},
		{"lock scan", newMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("xol_lock_master_ids")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
			m.ExpectRollback()
		}, nil, "membaca nomor terkunci"},
		{"lock rows", newMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("xol_lock_master_ids")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1").RowError(0, errDB))
			m.ExpectRollback()
		}, errDB, "menelusuri nomor terkunci"},
		{"insert master", newMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("xol_lock_master_ids")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectExec(q("xol_insert_master")).WithArgs("10001", "N", nil, int64(0), nil).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan master XOL"},
		{"update master", oldMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("xol_update_master")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "mengubah master XOL"},
		{"update rows affected", oldMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("xol_update_master")).WillReturnResult(sqlmock.NewErrorResult(errDB))
			m.ExpectRollback()
		}, errDB, "membaca jumlah baris terubah"},
		{"update not found", oldMaster, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("xol_update_master")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, masterxol.ErrNotFound, ""},
		{"business count", withBusiness, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectQuery(q("xol_business_count")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "memeriksa bisnis"},
		{"business insert", withBusiness, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectQuery(q("xol_business_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
			m.ExpectExec(q("xol_insert_business")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan bisnis"},
		{"lock layer ids", withNewLayer, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectQuery(q("xol_lock_layer_ids")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "xol_lock_layer_ids"},
		{"insert layer", withNewLayer, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectQuery(q("xol_lock_layer_ids")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectExec(q("xol_insert_layer")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan layer"},
		{"update layer", withOldLayer, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectExec(q("xol_update_layer")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "mengubah layer"},
		{"reas count", withReas, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectExec(q("xol_update_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("xol_reas_count")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "memeriksa reas"},
		{"reas update", withReas, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectExec(q("xol_update_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("xol_reas_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
			m.ExpectExec(q("xol_update_reas")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "mengubah reas"},
		{"reas insert", withReas, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectExec(q("xol_update_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectQuery(q("xol_reas_count")).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
			m.ExpectExec(q("xol_insert_reas")).WillReturnError(errDB)
			m.ExpectRollback()
		}, errDB, "menyisipkan reas"},
		{"commit", oldMaster, func(m sqlmock.Sqlmock) {
			updated(m)
			m.ExpectCommit().WillReturnError(errDB)
		}, errDB, "menyimpan master XOL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			_, err := repo.Save(ctx, tc.master)
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			if tc.text != "" {
				require.ErrorContains(t, err, tc.text)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDeleteMaster(t *testing.T) {
	ctx := context.Background()
	children := []string{"xol_delete_reas_of_master", "xol_delete_layer_of_master", "xol_delete_business_of_master"}

	t.Run("cascade order", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		for _, name := range children {
			mock.ExpectExec(q(name)).WithArgs("10001").WillReturnResult(sqlmock.NewResult(0, 2))
		}
		mock.ExpectExec(q("xol_delete_master")).WithArgs("10001").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.DeleteMaster(ctx, " 10001 "))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("begin", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errDB)
		require.ErrorIs(t, repo.DeleteMaster(ctx, "1"), errDB)
	})
	t.Run("child", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q(children[0])).WillReturnError(errDB)
		mock.ExpectRollback()
		err := repo.DeleteMaster(ctx, "1")
		require.ErrorIs(t, err, errDB)
		require.ErrorContains(t, err, children[0])
		require.NoError(t, mock.ExpectationsWereMet())
	})
	expectChildren := func(mock sqlmock.Sqlmock) {
		mock.ExpectBegin()
		for _, name := range children {
			mock.ExpectExec(q(name)).WillReturnResult(sqlmock.NewResult(0, 0))
		}
	}
	t.Run("master", func(t *testing.T) {
		repo, mock := newMock(t)
		expectChildren(mock)
		mock.ExpectExec(q("xol_delete_master")).WillReturnError(errDB)
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteMaster(ctx, "1"), errDB)
	})
	t.Run("rows affected", func(t *testing.T) {
		repo, mock := newMock(t)
		expectChildren(mock)
		mock.ExpectExec(q("xol_delete_master")).WillReturnResult(sqlmock.NewErrorResult(errDB))
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteMaster(ctx, "1"), errDB)
	})
	t.Run("not found", func(t *testing.T) {
		repo, mock := newMock(t)
		expectChildren(mock)
		mock.ExpectExec(q("xol_delete_master")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteMaster(ctx, "1"), masterxol.ErrNotFound)
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock := newMock(t)
		expectChildren(mock)
		mock.ExpectExec(q("xol_delete_master")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errDB)
		require.ErrorIs(t, repo.DeleteMaster(ctx, "1"), errDB)
	})
}

func TestDeleteBusiness(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	// ID kosong tidak pernah dapat dihapus — diwarisi dari kueri lama.
	require.ErrorIs(t, repo.DeleteBusiness(ctx, "1", "  "), masterxol.ErrNotFound)

	mock.ExpectExec(q("xol_delete_business")).WithArgs("10001", "01").WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, repo.DeleteBusiness(ctx, " 10001 ", " 01 "))

	mock.ExpectExec(q("xol_delete_business")).WillReturnError(errDB)
	require.ErrorIs(t, repo.DeleteBusiness(ctx, "1", "2"), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteLayer(t *testing.T) {
	ctx := context.Background()
	owner := func(m sqlmock.Sqlmock) {
		m.ExpectBegin()
		m.ExpectQuery(q("xol_layer_owner")).WithArgs("10001", "20001").
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("20001"))
	}

	t.Run("ok", func(t *testing.T) {
		repo, mock := newMock(t)
		owner(mock)
		mock.ExpectExec(q("xol_delete_reas_of_layer")).WithArgs("20001").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(q("xol_delete_layer")).WithArgs("10001", "20001").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
		require.NoError(t, repo.DeleteLayer(ctx, " 10001", "20001 "))
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("begin", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errDB)
		require.ErrorIs(t, repo.DeleteLayer(ctx, "1", "2"), errDB)
	})
	t.Run("not owned", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("xol_layer_owner")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteLayer(ctx, "1", "2"), masterxol.ErrLayerNotFound)
	})
	t.Run("owner error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("xol_layer_owner")).WillReturnError(errDB)
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteLayer(ctx, "1", "2"), errDB)
	})
	t.Run("reas", func(t *testing.T) {
		repo, mock := newMock(t)
		owner(mock)
		mock.ExpectExec(q("xol_delete_reas_of_layer")).WillReturnError(errDB)
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteLayer(ctx, "10001", "20001"), errDB)
	})
	t.Run("layer", func(t *testing.T) {
		repo, mock := newMock(t)
		owner(mock)
		mock.ExpectExec(q("xol_delete_reas_of_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(q("xol_delete_layer")).WillReturnError(errDB)
		mock.ExpectRollback()
		require.ErrorIs(t, repo.DeleteLayer(ctx, "10001", "20001"), errDB)
	})
	t.Run("commit", func(t *testing.T) {
		repo, mock := newMock(t)
		owner(mock)
		mock.ExpectExec(q("xol_delete_reas_of_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(q("xol_delete_layer")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errDB)
		require.ErrorIs(t, repo.DeleteLayer(ctx, "10001", "20001"), errDB)
	})
}

func TestDeleteReinsurer(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectExec(q("xol_delete_reas")).WithArgs("20001", "R1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.DeleteReinsurer(context.Background(), " 20001 ", " R1 "))
	mock.ExpectExec(q("xol_delete_reas")).WillReturnError(errDB)
	require.ErrorIs(t, repo.DeleteReinsurer(context.Background(), "1", "2"), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListYearSortsDescendingAndDropsBlanks(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_year_list")).WillReturnRows(sqlmock.NewRows([]string{"TAHUN"}).
		AddRow("2023").AddRow(" ").AddRow(nil).AddRow("2025").AddRow("2024"))
	got, err := repo.ListYear(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"2025", "2024", "2023"}, got)

	mock.ExpectQuery(q("xol_year_list")).WillReturnError(errDB)
	_, err = repo.ListYear(ctx)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("xol_year_list")).WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
	_, err = repo.ListYear(ctx)
	require.ErrorContains(t, err, "membaca baris tahun")

	mock.ExpectQuery(q("xol_year_list")).WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("1").RowError(0, errDB))
	_, err = repo.ListYear(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBusinessGroupAppendsTreatyInward(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_business_group_list")).WithArgs("%PROPERTY%", "%MOTOR%", "%ENGINEERING%").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" 2 ", " Motor ").AddRow("1", "Engineering"))
	got, err := repo.ListBusinessGroup(ctx, masterxol.TypeProperty)
	require.NoError(t, err)
	require.Equal(t, []masterxol.Business{{ID: "1", Name: "Engineering"}, {ID: "2", Name: "Motor"},
		{Name: masterxol.TreatyInwardName}}, got)

	mock.ExpectQuery(q("xol_business_group_list")).WithArgs("%", "%", "%").WillReturnError(errDB)
	_, err = repo.ListBusinessGroup(ctx, masterxol.TypeUnknown)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("xol_business_group_list")).WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
	_, err = repo.ListBusinessGroup(ctx, masterxol.TypeMarine)
	require.ErrorContains(t, err, "membaca baris grup bisnis")

	mock.ExpectQuery(q("xol_business_group_list")).WillReturnRows(
		sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("1", "a").RowError(0, errDB))
	_, err = repo.ListBusinessGroup(ctx, masterxol.TypeAccident)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSubmitToCommittee(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectExec(q("xol_submit_committee")).WithArgs("PIC1", "0", nil, "10001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.SubmitToCommittee(ctx, " 10001 ", " PIC1 ", "  "))

	mock.ExpectExec(q("xol_submit_committee")).WillReturnResult(sqlmock.NewResult(0, 0))
	require.ErrorIs(t, repo.SubmitToCommittee(ctx, "1", "p", "r"), masterxol.ErrNotFound)

	mock.ExpectExec(q("xol_submit_committee")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.ErrorIs(t, repo.SubmitToCommittee(ctx, "1", "p", "r"), errDB)

	mock.ExpectExec(q("xol_submit_committee")).WillReturnError(errDB)
	require.ErrorIs(t, repo.SubmitToCommittee(ctx, "1", "p", "r"), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrphanCount(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_orphan_count")).WillReturnRows(
		sqlmock.NewRows([]string{"JENIS", "JUMLAH"}).AddRow("layer", 2).AddRow("bisnis", 1))
	got, err := repo.OrphanCount(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string]int{"layer": 2, "bisnis": 1}, got)

	mock.ExpectQuery(q("xol_orphan_count")).WillReturnError(errDB)
	_, err = repo.OrphanCount(ctx)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(q("xol_orphan_count")).WillReturnRows(sqlmock.NewRows([]string{"JENIS"}).AddRow("x"))
	_, err = repo.OrphanCount(ctx)
	require.ErrorContains(t, err, "membaca jumlah baris yatim")

	mock.ExpectQuery(q("xol_orphan_count")).WillReturnRows(
		sqlmock.NewRows([]string{"JENIS", "JUMLAH"}).AddRow("x", 1).RowError(0, errDB))
	_, err = repo.OrphanCount(ctx)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("xol_check_table")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	require.NoError(t, repo.CheckTable(context.Background()))
	mock.ExpectQuery(q("xol_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, repo.CheckTable(context.Background()), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}
