package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastergroupingsparepart"
)

// errBoom adalah galat tiruan dari basis data.
var errBoom = errors.New("boom")

// groupingColumns adalah keenam belas kolom yang dibaca scanRow, pada urutannya.
var groupingColumns = []string{
	"ID", "NO_PART", "NAMA_PART", "KODE_PART", "KATEGORI", "TIPE", "TGL_PRODUKSI",
	"ID_PANEL", "NAMA_PANEL", "SISI_PANEL", "NO_RANGKA", "TIPE_KENDARAAN",
	"GROUP_DENGAN_RANGKA", "NO_GROUP_RANGKA", "CATATAN", "APPROVAL",
}

// newMock membentuk Repo di atas sqlmock dengan pencocok regexp.
func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// q mengubah teks kueri bernama menjadi pola regexp yang mencocokinya persis.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

// groupingRow menyusun satu baris lengkap dengan spasi tepi, supaya pemangkasan teruji.
func groupingRow(id string) []driver.Value {
	return []driver.Value{
		" " + id + " ", "SP-1 ", "FILTER ", "KD ", "KAT ", "TIP ", "01/01/2024 ",
		"PNL ", "KABIN ", "1 ", "RANGKA ", "EXCAVATOR ", "", "0001 ", "catatan ", "1 ",
	}
}

func expectedGrouping(id string) mastergroupingsparepart.Grouping {
	return mastergroupingsparepart.Grouping{
		ID: id, PartNumber: "SP-1", PartName: "FILTER", PartCode: "KD", CategoryID: "KAT",
		TypeID: "TIP", ProductionDate: "01/01/2024", PanelID: "PNL", PanelName: "KABIN",
		PanelSide: "1", ChassisNumber: "RANGKA", VehicleType: "EXCAVATOR",
		GroupNumber: "0001", Note: "catatan", Status: mastergroupingsparepart.StatusApproved,
	}
}

func sampleKeyGrouping() mastergroupingsparepart.Grouping {
	return mastergroupingsparepart.Grouping{
		ID: " 7 ", PartNumber: "sp-1", PanelName: "kabin", ChassisNumber: "rk", PanelSide: "1",
		VehicleType: "EXCAVATOR", Status: mastergroupingsparepart.StatusPending,
	}
}

func TestListWithoutKeywordMapsRows(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("grouping_list")).WithArgs("1").
		WillReturnRows(sqlmock.NewRows(groupingColumns).AddRow(groupingRow("1")...))

	got, err := repo.List(context.Background(),
		mastergroupingsparepart.Filter{Status: mastergroupingsparepart.StatusApproved})
	require.NoError(t, err)
	require.Equal(t, []mastergroupingsparepart.Grouping{expectedGrouping("1")}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWithKeywordBindsPatternFourTimes(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("grouping_list_search")).
		WithArgs("0", "%A\\_B%", "%A\\_B%", "%A\\_B%", "%A\\_B%").
		WillReturnRows(sqlmock.NewRows(groupingColumns))

	got, err := repo.List(context.Background(),
		mastergroupingsparepart.Filter{Status: mastergroupingsparepart.StatusPending, Keyword: " a_b "})
	require.NoError(t, err)
	require.Nil(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_list")).WillReturnError(errBoom)
		_, err := repo.List(context.Background(), mastergroupingsparepart.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "membaca daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_list")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("1"))
		_, err := repo.List(context.Background(), mastergroupingsparepart.Filter{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows err", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_list")).WillReturnRows(
			sqlmock.NewRows(groupingColumns).AddRow(groupingRow("1")...).RowError(0, errBoom))
		_, err := repo.List(context.Background(), mastergroupingsparepart.Filter{})
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menelusuri daftar")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGet(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_get")).WithArgs("5").
			WillReturnRows(sqlmock.NewRows(groupingColumns).AddRow(groupingRow("5")...))
		got, err := repo.Get(context.Background(), " 5 ")
		require.NoError(t, err)
		require.Equal(t, expectedGrouping("5"), got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("not found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_get")).WithArgs("5").
			WillReturnRows(sqlmock.NewRows(groupingColumns))
		_, err := repo.Get(context.Background(), "5")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_get")).WillReturnError(errBoom)
		_, err := repo.Get(context.Background(), "5")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindByKey(t *testing.T) {
	key := mastergroupingsparepart.NaturalKey{
		PartNumber: " sp-1 ", PanelName: "kabin", ChassisNumber: "rk ", PanelSide: " 1",
	}
	t.Run("empty key skips database", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.FindByKey(context.Background(), mastergroupingsparepart.NaturalKey{})
		require.ErrorIs(t, err, mastergroupingsparepart.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_find_by_key")).WithArgs("SP-1", "KABIN", "RK", "1").
			WillReturnRows(sqlmock.NewRows(groupingColumns).AddRow(groupingRow("9")...))
		got, err := repo.FindByKey(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, "9", got.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("not found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_find_by_key")).
			WillReturnRows(sqlmock.NewRows(groupingColumns))
		_, err := repo.FindByKey(context.Background(), key)
		require.ErrorIs(t, err, mastergroupingsparepart.ErrNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_find_by_key")).WillReturnError(errBoom)
		_, err := repo.FindByKey(context.Background(), key)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "mencari kunci grouping")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindGroupByChassis(t *testing.T) {
	t.Run("empty chassis", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.FindGroupByChassis(context.Background(), "  ")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrGroupChassisNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("found is returned as is", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_by_chassis")).WithArgs("RK").
			WillReturnRows(sqlmock.NewRows([]string{"G"}).AddRow(" 0001 "))
		got, err := repo.FindGroupByChassis(context.Background(), " rk ")
		require.NoError(t, err)
		require.Equal(t, "0001", got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("blank group", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_by_chassis")).
			WillReturnRows(sqlmock.NewRows([]string{"G"}).AddRow("   "))
		_, err := repo.FindGroupByChassis(context.Background(), "rk")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrGroupChassisNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_by_chassis")).
			WillReturnRows(sqlmock.NewRows([]string{"G"}))
		_, err := repo.FindGroupByChassis(context.Background(), "rk")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrGroupChassisNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_by_chassis")).WillReturnError(errBoom)
		_, err := repo.FindGroupByChassis(context.Background(), "rk")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListPanels(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_panel_list")).WithArgs("1").
			WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" P1 ", " KABIN "))
		got, err := repo.ListPanels(context.Background())
		require.NoError(t, err)
		require.Equal(t, []mastergroupingsparepart.Panel{{ID: "P1", Name: "KABIN"}}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("capped at max rows", func(t *testing.T) {
		repo, mock := newMock(t)
		rows := sqlmock.NewRows([]string{"ID", "NAMA"})
		for i := 0; i < mastergroupingsparepart.MaxLookupRows+3; i++ {
			rows.AddRow("P", "N")
		}
		mock.ExpectQuery(q("grouping_panel_list")).WillReturnRows(rows)
		got, err := repo.ListPanels(context.Background())
		require.NoError(t, err)
		require.Len(t, got, mastergroupingsparepart.MaxLookupRows)
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_panel_list")).WillReturnError(errBoom)
		_, err := repo.ListPanels(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_panel_list")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("P"))
		_, err := repo.ListPanels(context.Background())
		require.ErrorContains(t, err, "membaca baris panel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_panel_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("P", "N").RowError(0, errBoom))
		_, err := repo.ListPanels(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListSides(t *testing.T) {
	key := mastergroupingsparepart.SideKey{PanelID: " P1 ", PanelName: " kabin "}
	t.Run("blank sides are skipped", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_side_list")).WithArgs("P1", "KABIN").
			WillReturnRows(sqlmock.NewRows([]string{"S"}).AddRow(" 1 ").AddRow("  ").AddRow(nil))
		got, err := repo.ListSides(context.Background(), key)
		require.NoError(t, err)
		require.Equal(t, []mastergroupingsparepart.Side{"1"}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_side_list")).WillReturnError(errBoom)
		_, err := repo.ListSides(context.Background(), key)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_side_list")).
			WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
		_, err := repo.ListSides(context.Background(), key)
		require.ErrorContains(t, err, "membaca baris sisi panel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_side_list")).WillReturnRows(
			sqlmock.NewRows([]string{"S"}).AddRow("1").RowError(0, errBoom))
		_, err := repo.ListSides(context.Background(), key)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListVehicleTypes(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_vehicle_type_list")).WithArgs("ANEKA").
			WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow(" T1", "EXCAVATOR "))
		got, err := repo.ListVehicleTypes(context.Background())
		require.NoError(t, err)
		require.Equal(t, []mastergroupingsparepart.VehicleType{{ID: "T1", Name: "EXCAVATOR"}}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("capped at max rows", func(t *testing.T) {
		repo, mock := newMock(t)
		rows := sqlmock.NewRows([]string{"ID", "NAMA"})
		for i := 0; i < mastergroupingsparepart.MaxLookupRows+1; i++ {
			rows.AddRow("T", "N")
		}
		mock.ExpectQuery(q("grouping_vehicle_type_list")).WillReturnRows(rows)
		got, err := repo.ListVehicleTypes(context.Background())
		require.NoError(t, err)
		require.Len(t, got, mastergroupingsparepart.MaxLookupRows)
	})
	t.Run("query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_vehicle_type_list")).WillReturnError(errBoom)
		_, err := repo.ListVehicleTypes(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_vehicle_type_list")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("T"))
		_, err := repo.ListVehicleTypes(context.Background())
		require.ErrorContains(t, err, "membaca baris tipe kendaraan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_vehicle_type_list")).WillReturnRows(
			sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("T", "N").RowError(0, errBoom))
		_, err := repo.ListVehicleTypes(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestFindPart(t *testing.T) {
	columns := []string{"NO", "NAMA", "KAT", "TIPE", "KODE", "TGL"}
	t.Run("empty number", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.FindPart(context.Background(), " ")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrPartNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("found", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_part_find")).WithArgs("SP-1").
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow(" SP-1 ", " FILTER ", " K ", " T ", " KD ", nil))
		got, err := repo.FindPart(context.Background(), " sp-1 ")
		require.NoError(t, err)
		require.Equal(t, mastergroupingsparepart.PartRef{
			Number: "SP-1", Name: "FILTER", CategoryID: "K", TypeID: "T", Code: "KD",
		}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("no rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_part_find")).WillReturnRows(sqlmock.NewRows(columns))
		_, err := repo.FindPart(context.Background(), "SP-1")
		require.ErrorIs(t, err, mastergroupingsparepart.ErrPartNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_part_find")).WillReturnError(errBoom)
		_, err := repo.FindPart(context.Background(), "SP-1")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestInsertCommitsBothTables(t *testing.T) {
	repo, mock := newMock(t)
	g := sampleKeyGrouping()
	mock.ExpectBegin()
	mock.ExpectQuery(q("grouping_lock_by_key")).WithArgs("SP-1", "KABIN", "RK", "1").
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))
	args := make([]driver.Value, 0, 15)
	for _, a := range insertArguments(g) {
		args = append(args, a)
	}
	mock.ExpectExec(q("grouping_insert")).WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("grouping_group_insert")).WithArgs("7", "rk", "EXCAVATOR").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Insert(context.Background(), g))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertWithEmptyKeySkipsLock(t *testing.T) {
	repo, mock := newMock(t)
	g := mastergroupingsparepart.Grouping{ID: "1"}
	mock.ExpectBegin()
	mock.ExpectExec(q("grouping_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("grouping_group_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Insert(context.Background(), g))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	g := sampleKeyGrouping()
	cases := []struct {
		name   string
		setup  func(mock sqlmock.Sqlmock)
		target error
		text   string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errBoom) },
			errBoom, "memulai transaksi"},
		{"duplicate", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).
				WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("3"))
			m.ExpectRollback()
		}, mastergroupingsparepart.ErrDuplicate, ""},
		{"lock query", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "memeriksa kunci"},
		{"lock scan", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).
				WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
			m.ExpectRollback()
		}, nil, "membaca hasil pemeriksaan"},
		{"lock rows err", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).WillReturnRows(
				sqlmock.NewRows([]string{"ID"}).AddRow("1").RowError(0, errBoom))
			m.ExpectRollback()
		}, errBoom, "menelusuri pemeriksaan kunci"},
		{"insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectExec(q("grouping_insert")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "menyisipkan"},
		{"companion", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectExec(q("grouping_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("grouping_group_insert")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "menyisipkan pendamping"},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("grouping_lock_by_key")).WillReturnRows(sqlmock.NewRows([]string{"ID"}))
			m.ExpectExec(q("grouping_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("grouping_group_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errBoom)
		}, errBoom, "menutup transaksi sisip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			err := repo.Insert(context.Background(), g)
			require.Error(t, err)
			if tc.target != nil {
				require.ErrorIs(t, err, tc.target)
			}
			if tc.text != "" {
				require.Contains(t, err.Error(), tc.text)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateUpdatesCompanionInPlace(t *testing.T) {
	repo, mock := newMock(t)
	g := sampleKeyGrouping()
	args := make([]driver.Value, 0, 15)
	for _, a := range updateArguments(g) {
		args = append(args, a)
	}
	mock.ExpectBegin()
	mock.ExpectExec(q("grouping_update")).WithArgs(args...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("grouping_group_update")).WithArgs("rk", "EXCAVATOR", "7").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), g))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateInsertsMissingCompanion(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("grouping_group_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("grouping_group_insert")).WithArgs("7", "rk", "EXCAVATOR").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), sampleKeyGrouping()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Driver yang tidak dapat melaporkan RowsAffected tidak dianggap gagal, dan baris pendamping
// TIDAK disisipkan "untuk berjaga-jaga".
func TestUpdateToleratesRowsAffectedErrors(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
	mock.ExpectExec(q("grouping_group_update")).WillReturnResult(sqlmock.NewErrorResult(errBoom))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), sampleKeyGrouping()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateFailures(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(mock sqlmock.Sqlmock)
		target error
		text   string
	}{
		{"begin", func(m sqlmock.Sqlmock) { m.ExpectBegin().WillReturnError(errBoom) },
			errBoom, "memulai transaksi ubah"},
		{"update", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("grouping_update")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "memperbarui"},
		{"not found", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, mastergroupingsparepart.ErrNotFound, ""},
		{"companion update", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("grouping_group_update")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "memperbarui pendamping"},
		{"companion insert", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("grouping_group_update")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectExec(q("grouping_group_insert")).WillReturnError(errBoom)
			m.ExpectRollback()
		}, errBoom, "menyisipkan pendamping"},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("grouping_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("grouping_group_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errBoom)
		}, errBoom, "menutup transaksi ubah"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, mock := newMock(t)
			tc.setup(mock)
			err := repo.Update(context.Background(), sampleKeyGrouping())
			require.ErrorIs(t, err, tc.target)
			if tc.text != "" {
				require.Contains(t, err.Error(), tc.text)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSetStatus(t *testing.T) {
	t.Run("empty list touches nothing", func(t *testing.T) {
		repo, mock := newMock(t)
		n, err := repo.SetStatus(context.Background(), nil, mastergroupingsparepart.StatusApproved)
		require.NoError(t, err)
		require.Zero(t, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("counts changed rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("grouping_set_status")).WithArgs("1", "1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(q("grouping_set_status")).WithArgs("1", "2").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(q("grouping_set_status")).WithArgs("1", "3").
			WillReturnResult(sqlmock.NewErrorResult(errBoom))
		mock.ExpectCommit()
		n, err := repo.SetStatus(context.Background(), []string{" 1 ", "2", "3"},
			mastergroupingsparepart.StatusApproved)
		require.NoError(t, err)
		// Baris ke-3 dianggap berubah karena RowsAffected tidak tersedia.
		require.Equal(t, 2, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("begin error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := repo.SetStatus(context.Background(), []string{"1"},
			mastergroupingsparepart.StatusApproved)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("exec error rolls back", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("grouping_set_status")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.SetStatus(context.Background(), []string{"1"},
			mastergroupingsparepart.StatusRejected)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "menetapkan status")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(q("grouping_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := repo.SetStatus(context.Background(), []string{"1"},
			mastergroupingsparepart.StatusRejected)
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNextID(t *testing.T) {
	maxRow := func(v driver.Value) *sqlmock.Rows {
		return sqlmock.NewRows([]string{"M"}).AddRow(v)
	}
	t.Run("mirror wins when larger", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnRows(maxRow(" 7 "))
		mock.ExpectQuery(q("grouping_max_id_mirror")).WillReturnRows(maxRow("12"))
		mock.ExpectCommit()
		id, err := repo.NextID(context.Background())
		require.NoError(t, err)
		require.Equal(t, "13", id)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("live wins and empty table starts at one", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnRows(maxRow(nil))
		mock.ExpectQuery(q("grouping_max_id_mirror")).WillReturnRows(maxRow(""))
		mock.ExpectCommit()
		id, err := repo.NextID(context.Background())
		require.NoError(t, err)
		require.Equal(t, "1", id)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("unreadable mirror is ignored", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnRows(maxRow("4"))
		mock.ExpectQuery(q("grouping_max_id_mirror")).WillReturnError(errBoom)
		mock.ExpectCommit()
		id, err := repo.NextID(context.Background())
		require.NoError(t, err)
		require.Equal(t, "5", id)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("begin error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)
		_, err := repo.NextID(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("live query error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnError(errBoom)
		mock.ExpectRollback()
		_, err := repo.NextID(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "grouping_max_id")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("live not numeric", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnRows(maxRow("ABC"))
		mock.ExpectRollback()
		_, err := repo.NextID(context.Background())
		require.ErrorContains(t, err, `"ABC" yang bukan angka`)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("commit error", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("grouping_max_id")).WillReturnRows(maxRow("1"))
		mock.ExpectQuery(q("grouping_max_id_mirror")).WillReturnRows(maxRow("1"))
		mock.ExpectCommit().WillReturnError(errBoom)
		_, err := repo.NextID(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestNextGroupNumberSkipsUnreadableValues(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("grouping_group_numbers")).WillReturnRows(
		sqlmock.NewRows([]string{"G"}).AddRow("0009").AddRow("12").AddRow("X1").AddRow(nil))
	got, err := repo.NextGroupNumber(context.Background())
	require.NoError(t, err)
	require.Equal(t, mastergroupingsparepart.ComposeGroupNumber(13), got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNextGroupNumberErrors(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).WillReturnError(errBoom)
		_, err := repo.NextGroupNumber(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).
			WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
		_, err := repo.NextGroupNumber(context.Background())
		require.ErrorContains(t, err, "membaca baris nomor grup")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).WillReturnRows(
			sqlmock.NewRows([]string{"G"}).AddRow("1").RowError(0, errBoom))
		_, err := repo.NextGroupNumber(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCountUnreadableGroupNumber(t *testing.T) {
	t.Run("counts non numeric values", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).WillReturnRows(
			sqlmock.NewRows([]string{"G"}).AddRow("0001").AddRow("A").AddRow("+1").AddRow(nil))
		n, err := repo.CountUnreadableGroupNumber(context.Background())
		require.NoError(t, err)
		require.Equal(t, 2, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).WillReturnError(errBoom)
		_, err := repo.CountUnreadableGroupNumber(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).
			WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "2"))
		_, err := repo.CountUnreadableGroupNumber(context.Background())
		require.ErrorContains(t, err, "membaca baris nomor grup")
		require.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(q("grouping_group_numbers")).WillReturnRows(
			sqlmock.NewRows([]string{"G"}).AddRow("1").RowError(0, errBoom))
		_, err := repo.CountUnreadableGroupNumber(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckFunctions(t *testing.T) {
	checks := []struct {
		query string
		table string
		call  func(r *Repo) error
	}{
		{"grouping_check_table", "POOLDATA.SPAREPART_HE_VIN_KEY",
			func(r *Repo) error { return r.CheckTable(context.Background()) }},
		{"grouping_check_group_table", "POOLDATA.SPAREPART_HE_VIN_GROUP",
			func(r *Repo) error { return r.CheckGroupTable(context.Background()) }},
		{"grouping_check_vehicle_type_table", "branddetail",
			func(r *Repo) error { return r.CheckVehicleTypeTable(context.Background()) }},
		{"grouping_check_child_name_column", "POOLDATA.LOKASI_PANEL_HE",
			func(r *Repo) error { return r.CheckChildNameColumn(context.Background()) }},
		{"grouping_check_json_mirror", "POOLDATA.M_SPAREPART_HE_VIN_KEY",
			func(r *Repo) error { return r.CheckJSONMirror(context.Background()) }},
	}
	for _, c := range checks {
		t.Run(c.query, func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(q(c.query)).WillReturnRows(sqlmock.NewRows([]string{"X"}))
			require.NoError(t, c.call(repo))

			mock.ExpectQuery(q(c.query)).WillReturnError(errBoom)
			err := c.call(repo)
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), c.table+" tidak dapat dibaca")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCountFunctions(t *testing.T) {
	counts := []struct {
		query string
		args  []driver.Value
		call  func(r *Repo) (int, error)
	}{
		{"grouping_count_by_status", []driver.Value{"0"}, func(r *Repo) (int, error) {
			return r.CountByStatus(context.Background(), mastergroupingsparepart.StatusPending)
		}},
		{"grouping_count_all", nil,
			func(r *Repo) (int, error) { return r.CountAll(context.Background()) }},
		{"grouping_count_without_group", nil,
			func(r *Repo) (int, error) { return r.CountWithoutGroup(context.Background()) }},
		{"grouping_count_chassis_mismatch", nil,
			func(r *Repo) (int, error) { return r.CountChassisMismatch(context.Background()) }},
		{"grouping_count_child_name_as_panel", nil,
			func(r *Repo) (int, error) { return r.CountChildNameAsPanel(context.Background()) }},
		{"grouping_count_child_name_as_location", nil,
			func(r *Repo) (int, error) { return r.CountChildNameAsLocation(context.Background()) }},
		{"grouping_count_child_rows", nil,
			func(r *Repo) (int, error) { return r.CountChildRows(context.Background()) }},
		{"grouping_count_orphan_part", nil,
			func(r *Repo) (int, error) { return r.CountOrphanPart(context.Background()) }},
		{"grouping_count_orphan_panel", nil,
			func(r *Repo) (int, error) { return r.CountOrphanPanel(context.Background()) }},
		{"grouping_count_json_mirror", nil,
			func(r *Repo) (int, error) { return r.CountJSONMirror(context.Background()) }},
	}
	for _, c := range counts {
		t.Run(c.query, func(t *testing.T) {
			repo, mock := newMock(t)
			expect := mock.ExpectQuery(q(c.query))
			if c.args != nil {
				expect.WithArgs(c.args...)
			}
			expect.WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))
			n, err := c.call(repo)
			require.NoError(t, err)
			require.Equal(t, 42, n)

			mock.ExpectQuery(q(c.query)).WillReturnError(errBoom)
			_, err = c.call(repo)
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), c.query)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// splitByName membuang baris komentar dan mengabaikan teks sebelum penanda pertama.
func TestSplitByNameDropsCommentsAndPreamble(t *testing.T) {
	got := splitByName(strings.Join([]string{
		"-- pembuka tanpa nama",
		"SELECT 0",
		"-- name: one",
		"-- komentar",
		"SELECT 1",
		"-- name: empty",
		"-- hanya komentar",
		"-- name: two",
		"SELECT 2",
	}, "\n"))
	require.Equal(t, map[string]string{"one": "SELECT 1", "two": "SELECT 2"}, got)
}
