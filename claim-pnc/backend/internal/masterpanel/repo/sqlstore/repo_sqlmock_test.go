package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

var (
	errOracle = errors.New("oracle menolak")
	panelCols = []string{"ID", "NAME", "R", "E", "P", "SH", "ST", "SI", "SE", "A", "C", "M", "REASON", "DOC", "APPROVAL"}
	ownedCols = []string{"ID_PANEL", "LOKASI", "SISI"}
	locCols   = []string{"LOKASI", "SISI"}
	singleCol = []string{"X"}
	mockPanel = masterpanel.Panel{
		ID: "01000001", Name: "Pintu", RepairStatus: "1", EditQuantityStatus: "1",
		PremiumRepairStatus: "0", ShatterStatus: "0", StickerStatus: "1", SideStatus: "1",
		SevereDamageStatus: "0", ActiveStatus: "1", ExclusionC: "C", ApprovalMark: "M",
		RejectReason: "", DocumentID: "D", Status: masterpanel.StatusPending,
		Location: []masterpanel.PanelLocation{
			{Name: " KIRI ", Side: masterpanel.SideLeft}, {Name: "KANAN", Side: masterpanel.SideRight}},
	}
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

func panelRow(id, name string) *sqlmock.Rows {
	return sqlmock.NewRows(panelCols).AddRow(id+" ", name, "1", "1", "0", "0", "1", "1", "0",
		"1", nil, nil, nil, nil, "0 ")
}

// Daftar tanpa kata kunci memakai kueri dasar, lalu menempelkan lokasi per ID induk.
func TestListAttachesLocationsByOwner(t *testing.T) {
	repo, mock := newMock(t)

	rows := panelRow("01000001", "Pintu")
	rows.AddRow("01000002", "Kaca", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "0")
	mock.ExpectQuery(q("panel_list")).WithArgs("0").WillReturnRows(rows)
	mock.ExpectQuery(q("panel_location_by_status")).WithArgs("0").
		WillReturnRows(sqlmock.NewRows(ownedCols).
			AddRow("01000001 ", " KIRI ", "1").
			AddRow("01000001", "KANAN", "2"))

	list, err := repo.List(context.Background(), masterpanel.Filter{Status: masterpanel.StatusPending})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, masterpanel.Panel{
		ID: "01000001", Name: "Pintu", RepairStatus: "1", EditQuantityStatus: "1",
		PremiumRepairStatus: "0", ShatterStatus: "0", StickerStatus: "1", SideStatus: "1",
		SevereDamageStatus: "0", ActiveStatus: "1", Status: masterpanel.StatusPending,
		Location: []masterpanel.PanelLocation{
			{Name: "KIRI", Side: masterpanel.SideLeft}, {Name: "KANAN", Side: masterpanel.SideRight}},
	}, list[0])
	require.Nil(t, list[1].Location)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci memakai kedua kueri pencarian dengan pola LIKE yang diloloskan.
func TestListWithKeywordUsesTheSearchQueries(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(q("panel_list_search")).WithArgs("1", `%A\%B%`).
		WillReturnRows(panelRow("01000001", "Pintu"))
	mock.ExpectQuery(q("panel_location_by_status_search")).WithArgs("1", `%A\%B%`).
		WillReturnRows(sqlmock.NewRows(ownedCols))

	list, err := repo.List(context.Background(),
		masterpanel.Filter{Status: masterpanel.StatusApproved, Keyword: " a%b "})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Daftar kosong tidak menjalankan kueri lokasi.
func TestEmptyListSkipsTheLocationQuery(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(q("panel_list")).WillReturnRows(sqlmock.NewRows(panelCols))

	list, err := repo.List(context.Background(), masterpanel.Filter{Status: masterpanel.StatusRejected})
	require.NoError(t, err)
	require.Empty(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()
	filter := masterpanel.Filter{Status: masterpanel.StatusPending}

	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		message string
	}{
		{"daftar", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnError(errOracle)
		}, "membaca daftar"},
		{"pindai", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow("x"))
		}, "membaca baris daftar"},
		{"telusur", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnRows(panelRow("1", "a").RowError(0, errOracle))
		}, "menelusuri daftar"},
		{"lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_by_status")).WillReturnError(errOracle)
		}, "membaca lokasi panel"},
		{"pindai lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_by_status")).
				WillReturnRows(sqlmock.NewRows(singleCol).AddRow("x"))
		}, "membaca baris lokasi"},
		{"telusur lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_list")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_by_status")).
				WillReturnRows(sqlmock.NewRows(ownedCols).AddRow("1", "a", "1").RowError(0, errOracle))
		}, "menelusuri lokasi panel"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			_, err := repo.List(ctx, filter)
			require.ErrorContains(t, err, c.message)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Get membaca induk lalu lokasinya.
func TestGetReadsTheRowAndItsLocations(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(q("panel_get")).WithArgs("01000001").WillReturnRows(panelRow("01000001", "Pintu"))
	mock.ExpectQuery(q("panel_location_get")).WithArgs("01000001").
		WillReturnRows(sqlmock.NewRows(locCols).AddRow(" KIRI ", "1"))

	panel, err := repo.Get(context.Background(), " 01000001 ")
	require.NoError(t, err)
	require.Equal(t, []masterpanel.PanelLocation{{Name: "KIRI", Side: masterpanel.SideLeft}},
		panel.Location)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"tidak ada", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_get")).WillReturnError(sql.ErrNoRows)
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterpanel.ErrNotFound) }},
		{"gagal", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_get")).WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `membaca "1"`) }},
		{"lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_get")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_get")).WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `membaca lokasi "1"`) }},
		{"pindai lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_get")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_get")).
				WillReturnRows(sqlmock.NewRows(singleCol).AddRow("x"))
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "membaca baris lokasi") }},
		{"telusur lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(q("panel_get")).WillReturnRows(panelRow("1", "a"))
			m.ExpectQuery(q("panel_location_get")).
				WillReturnRows(sqlmock.NewRows(locCols).AddRow("a", "1").RowError(0, errOracle))
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menelusuri lokasi") }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			_, err := repo.Get(ctx, "1")
			c.check(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindByName(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	_, err := repo.FindByName(ctx, "  ")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)

	mock.ExpectQuery(q("panel_find_by_name")).WithArgs("PINTU").WillReturnRows(panelRow("1", "Pintu"))
	panel, err := repo.FindByName(ctx, " pintu ")
	require.NoError(t, err)
	require.Equal(t, "1", panel.ID)

	mock.ExpectQuery(q("panel_find_by_name")).WillReturnError(sql.ErrNoRows)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)

	mock.ExpectQuery(q("panel_find_by_name")).WillReturnError(errOracle)
	_, err = repo.FindByName(ctx, "x")
	require.ErrorContains(t, err, "mencari nama")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Penyisipan memeriksa nama di dalam transaksi, lalu menyisipkan induk dan setiap lokasi.
func TestInsertWritesParentAndChildrenInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_lock_by_name")).WithArgs("PINTU").
		WillReturnRows(sqlmock.NewRows(singleCol))
	mock.ExpectExec(q("panel_insert")).WithArgs("01000001", "Pintu", "1", "1", "0", "0", "1",
		"1", "0", "1", "C", "M", "", "D", "0").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_location_insert")).WithArgs("01000001", "KIRI", "1", "KIRI").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_location_insert")).WithArgs("01000001", "KANAN", "2", "KANAN").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Insert(context.Background(), mockPanel))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertErrors(t *testing.T) {
	cases := []struct {
		name    string
		panel   masterpanel.Panel
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"mulai", mockPanel, func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi") }},
		{"nama dipakai", mockPanel, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("panel_lock_by_name")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow(1))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterpanel.ErrNameTaken) }},
		{"periksa nama", mockPanel, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("panel_lock_by_name")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memeriksa nama") }},
		{"telusur nama", mockPanel, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("panel_lock_by_name")).
				WillReturnRows(sqlmock.NewRows(singleCol).AddRow(1).RowError(0, errOracle))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menelusuri pemeriksaan") }},
		{"sisip induk", masterpanel.Panel{ID: "1"}, func(m sqlmock.Sqlmock) {
			// Nama kosong tidak diperiksa ke basis data.
			m.ExpectBegin()
			m.ExpectExec(q("panel_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `menyisipkan "1"`) }},
		{"sisip lokasi", mockPanel, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectQuery(q("panel_lock_by_name")).WillReturnRows(sqlmock.NewRows(singleCol))
			m.ExpectExec(q("panel_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("panel_location_insert")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menyisipkan lokasi") }},
		{"commit", masterpanel.Panel{ID: "1"}, func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("panel_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi sisip") }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			c.check(t, repo.Insert(context.Background(), c.panel))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Penyuntingan menimpa induk, membuang lokasi lama, lalu menyisipkan ulang.
func TestUpdateReplacesTheLocations(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(q("panel_update")).WithArgs("Pintu", "1", "1", "0", "0", "1", "1", "0",
		"1", "C", "M", "", "D", "0", "01000001").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_location_clear")).WithArgs("01000001").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(q("panel_location_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_location_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.Update(context.Background(), mockPanel))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateErrors(t *testing.T) {
	plain := masterpanel.Panel{ID: "1"}
	cases := []struct {
		name    string
		prepare func(sqlmock.Sqlmock)
		check   func(*testing.T, error)
	}{
		{"mulai", func(m sqlmock.Sqlmock) {
			m.ExpectBegin().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "memulai transaksi ubah") }},
		{"perbarui", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("panel_update")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, `memperbarui "1"`) }},
		{"tidak ada", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("panel_update")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorIs(t, err, masterpanel.ErrNotFound) }},
		{"buang lokasi", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			// Driver tanpa RowsAffected tidak dianggap gagal.
			m.ExpectExec(q("panel_update")).WillReturnResult(sqlmock.NewErrorResult(errOracle))
			m.ExpectExec(q("panel_location_clear")).WillReturnError(errOracle)
			m.ExpectRollback()
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "membuang lokasi") }},
		{"commit", func(m sqlmock.Sqlmock) {
			m.ExpectBegin()
			m.ExpectExec(q("panel_update")).WillReturnResult(sqlmock.NewResult(0, 1))
			m.ExpectExec(q("panel_location_clear")).WillReturnResult(sqlmock.NewResult(0, 0))
			m.ExpectCommit().WillReturnError(errOracle)
		}, func(t *testing.T, err error) { require.ErrorContains(t, err, "menutup transaksi ubah") }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, mock := newMock(t)
			c.prepare(mock)
			c.check(t, repo.Update(context.Background(), plain))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	// Lokasi yang gagal disisipkan pada penyuntingan ikut membatalkan.
	repo, mock := newMock(t)
	mock.ExpectBegin()
	mock.ExpectExec(q("panel_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_location_clear")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("panel_location_insert")).WillReturnError(errOracle)
	mock.ExpectRollback()
	require.ErrorContains(t, repo.Update(context.Background(), mockPanel), "menyisipkan lokasi")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Keputusan borongan berjalan dalam satu transaksi dan menghitung baris yang berubah.
func TestSetStatusCountsChangedRows(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	changed, err := repo.SetStatus(ctx, nil, masterpanel.StatusApproved, "")
	require.NoError(t, err)
	require.Zero(t, changed)

	mock.ExpectBegin()
	mock.ExpectExec(q("panel_set_status")).WithArgs("2", "alasan", "A").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(q("panel_set_status")).WithArgs("2", "alasan", "B").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("panel_set_status")).WithArgs("2", "alasan", "C").
		WillReturnResult(sqlmock.NewErrorResult(errOracle))
	mock.ExpectCommit()

	changed, err = repo.SetStatus(ctx, []string{" A ", "B", "C"}, masterpanel.StatusRejected, "alasan")
	require.NoError(t, err)
	require.Equal(t, 2, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetStatusErrors(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectBegin().WillReturnError(errOracle)
	_, err := repo.SetStatus(ctx, []string{"A"}, masterpanel.StatusApproved, "")
	require.ErrorContains(t, err, "memulai transaksi keputusan")

	mock.ExpectBegin()
	mock.ExpectExec(q("panel_set_status")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.SetStatus(ctx, []string{"A"}, masterpanel.StatusApproved, "")
	require.ErrorContains(t, err, `menetapkan status "A"`)

	mock.ExpectBegin()
	mock.ExpectExec(q("panel_set_status")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errOracle)
	_, err = repo.SetStatus(ctx, []string{"A"}, masterpanel.StatusApproved, "")
	require.ErrorContains(t, err, "menutup transaksi keputusan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// NextID menyusun kode situs + enam digit dalam satu transaksi.
func TestNextID(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_site")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow(" 01 "))
	mock.ExpectQuery(q("panel_next_sequence")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow(7))
	mock.ExpectCommit()
	id, err := repo.NextID(ctx)
	require.NoError(t, err)
	require.Equal(t, masterpanel.ComposeID("01", 7, sequenceWidth), id)

	mock.ExpectBegin().WillReturnError(errOracle)
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "memulai transaksi penomoran")

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_site")).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "tidak punya baris CURRENT_SITE='1'")

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_site")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "membaca kode situs")

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_site")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow("01"))
	mock.ExpectQuery(q("panel_next_sequence")).WillReturnError(errOracle)
	mock.ExpectRollback()
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "mengambil nomor urut")

	mock.ExpectBegin()
	mock.ExpectQuery(q("panel_site")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow("01"))
	mock.ExpectQuery(q("panel_next_sequence")).WillReturnRows(sqlmock.NewRows(singleCol).AddRow(1))
	mock.ExpectCommit().WillReturnError(errOracle)
	_, err = repo.NextID(ctx)
	require.ErrorContains(t, err, "menutup transaksi penomoran")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pemeriksaan tabel dan pencacah mode periksa.
func TestChecksAndCounters(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMock(t)

	checks := []struct {
		name  string
		call  func(context.Context) error
		table string
	}{
		{"panel_check_table", repo.CheckTable, "POOLDATA.PANEL_HE"},
		{"panel_check_location_table", repo.CheckLocationTable, "POOLDATA.LOKASI_PANEL_HE"},
		{"panel_check_json_mirror", repo.CheckJSONMirror, "POOLDATA.M_PANEL_HE"},
	}
	for _, c := range checks {
		mock.ExpectQuery(q(c.name)).WillReturnRows(sqlmock.NewRows(singleCol))
		require.NoError(t, c.call(ctx))
		mock.ExpectQuery(q(c.name)).WillReturnError(errOracle)
		require.ErrorContains(t, c.call(ctx), c.table+" tidak dapat dibaca")
	}

	counters := []struct {
		name string
		call func(context.Context) (int, error)
	}{
		{"panel_count_all", repo.CountAll},
		{"panel_count_json_mirror", repo.CountJSONMirror},
		{"panel_count_location", repo.CountLocation},
		{"panel_count_location_orphan", repo.CountLocationOrphan},
		{"panel_count_location_name_mismatch", repo.CountLocationNameMismatch},
	}
	for i, c := range counters {
		mock.ExpectQuery(q(c.name)).WillReturnRows(sqlmock.NewRows(singleCol).AddRow(i + 1))
		total, err := c.call(ctx)
		require.NoError(t, err)
		require.Equal(t, i+1, total)
	}

	mock.ExpectQuery(q("panel_count_pending")).WithArgs("0").
		WillReturnRows(sqlmock.NewRows(singleCol).AddRow(4))
	pending, err := repo.CountByStatus(ctx, masterpanel.StatusPending)
	require.NoError(t, err)
	require.Equal(t, 4, pending)

	mock.ExpectQuery(q("panel_count_all")).WillReturnError(errOracle)
	_, err = repo.CountAll(ctx)
	require.ErrorContains(t, err, "menjalankan panel_count_all")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnknownQueryNamePanics(t *testing.T) {
	require.Panics(t, func() { getQuery("tidak_ada") })
}
