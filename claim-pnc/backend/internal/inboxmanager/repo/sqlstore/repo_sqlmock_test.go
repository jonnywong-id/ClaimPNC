package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// newMock membentuk Repo di atas sqlmock yang mencocokkan teks kueri persis (di-escape).
func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string {
	return "^" + regexp.QuoteMeta(query(name)) + "$"
}

func values(args []any) []driver.Value {
	result := make([]driver.Value, 0, len(args))
	for _, a := range args {
		result = append(result, a)
	}
	return result
}

func mustTab(t *testing.T, code string) inboxmanager.Tab {
	t.Helper()
	tab, known := inboxmanager.FindTab(code)
	require.Truef(t, known, "tab %s tidak terdaftar", code)
	return tab
}

var refreshed = time.Date(2026, 9, 28, 2, 0, 0, 0, time.UTC)

func TestLineFlags(t *testing.T) {
	require.Equal(t, []any{0, 1, 0, 0, 0}, lineFlags(" nonmbu "))
	require.Equal(t, []any{0, 0, 1, 0, 0}, lineFlags("BONDING"))
	require.Equal(t, []any{0, 0, 0, 1, 0}, lineFlags("pa"))
	require.Equal(t, []any{0, 0, 0, 0, 1}, lineFlags("Travel"))
	require.Equal(t, []any{1, 0, 0, 0, 0}, lineFlags("IT SPECIALIST"))
	require.Equal(t, []any{1, 0, 0, 0, 0}, lineFlags(""))
}

// Satu pencacah yang rusak tidak menghilangkan yang lain.
func TestCountersKeepGoingWhenOneSourceFails(t *testing.T) {
	repo, mock := newMock(t)
	count := func(n int) *sqlmock.Rows { return sqlmock.NewRows([]string{"TOTAL"}).AddRow(n) }

	mock.ExpectQuery(exact("count_outstanding")).WithArgs(0, 1, 0, 0, 0).WillReturnRows(count(7))
	mock.ExpectQuery(exact("count_bengkel")).WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}))
	mock.ExpectQuery(exact("count_panel")).WillReturnError(errors.New("ORA-00942: table missing"))
	mock.ExpectQuery(exact("count_nomor_rangka")).WillReturnRows(count(3))
	mock.ExpectQuery(exact("count_sparepart")).
		WillReturnError(errors.New("ORA-04063: view POOLDATA.SPAREPART_HE has errors"))
	mock.ExpectQuery(exact("count_kategori_sparepart")).WillReturnRows(count(1))
	mock.ExpectQuery(exact("count_tipe_sparepart")).WillReturnRows(count(2))
	mock.ExpectQuery(exact("count_grouping_sparepart")).WillReturnRows(count(0))
	mock.ExpectQuery(exact("count_payment_akseptasi")).WillReturnRows(count(4))
	mock.ExpectQuery(exact("count_penolakan_klaim")).WillReturnRows(count(5))

	counters, err := repo.Counters(context.Background(), inboxmanager.Caller{LineBusiness: "NONMBU"})
	require.NoError(t, err)
	require.Len(t, counters, 10)

	byTab := map[string]inboxmanager.Counter{}
	for _, c := range counters {
		byTab[c.TabCode] = c
	}

	outstanding := byTab[inboxmanager.TabOutstanding]
	require.Equal(t, 7, outstanding.Count)
	require.Empty(t, outstanding.Parent, "Outstanding adalah pencacah tingkat atas")
	require.Equal(t, mustTab(t, inboxmanager.TabOutstanding).Name, outstanding.Label)

	require.Equal(t, 0, byTab[inboxmanager.TabMasterBengkel].Count)
	require.Empty(t, byTab[inboxmanager.TabMasterBengkel].Unavailable)
	require.Equal(t, inboxmanager.TabApprovalMaster, byTab[inboxmanager.TabMasterBengkel].Parent)

	require.Equal(t, "Sumber antrean ini sedang tidak dapat dibaca basis data.",
		byTab[inboxmanager.TabMasterPanel].Unavailable)
	require.Contains(t, byTab[inboxmanager.TabMasterSparepart].Unavailable,
		"view POOLDATA.SPAREPART_HE")
	require.Equal(t, 5, byTab[inboxmanager.TabPenolakanKlaim].Count)

	// Urutannya mengikuti urutan kontainer: Outstanding lebih dulu.
	require.Equal(t, inboxmanager.TabOutstanding, counters[0].TabCode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUnavailableReasonOnlyNamesSparepartViewForInvalidObject(t *testing.T) {
	generic := "Sumber antrean ini sedang tidak dapat dibaca basis data."
	require.Equal(t, generic, unavailableReason(inboxmanager.TabMasterSparepart, errors.New("ORA-00942")))
	require.Equal(t, generic, unavailableReason(inboxmanager.TabMasterPanel, errors.New("ORA-04063")))
}

func TestWrapQueryError(t *testing.T) {
	invalid := wrapQueryError("membaca x", errors.New("ORA-04063: invalid"))
	require.ErrorIs(t, invalid, inboxmanager.ErrSourceUnavailable)
	require.Contains(t, invalid.Error(), "membaca x")
	require.Contains(t, invalid.Error(), "ORA-04063")

	missing := wrapQueryError("membaca x", errors.New("ORA-00904: invalid identifier"))
	require.ErrorIs(t, missing, inboxmanager.ErrSourceUnavailable)

	unbound := errors.New("ORA-01008: not all variables bound")
	wrapped := wrapQueryError("membaca x", unbound)
	require.ErrorIs(t, wrapped, unbound)
	require.NotErrorIs(t, wrapped, inboxmanager.ErrSourceUnavailable)
	require.Contains(t, wrapped.Error(), "jumlah argumen bind")

	other := errors.New("putus")
	plain := wrapQueryError("membaca x", other)
	require.ErrorIs(t, plain, other)
	require.Equal(t, "membaca x: putus", plain.Error())

	require.False(t, isInvalidObject(nil))
	require.False(t, isMissingColumn(nil))
	require.False(t, isUnboundVariable(nil))
}

func outstandingQuery(t *testing.T, line string) inboxmanager.Query {
	t.Helper()
	return inboxmanager.Query{Tab: mustTab(t, inboxmanager.TabOutstanding), LineBusiness: line}
}

func TestDashboardOutstanding(t *testing.T) {
	repo, mock := newMock(t)
	flags := lineFlags("PA")
	picArgs := append(append([]any{}, flags...), flags...)

	mock.ExpectQuery(exact("dashboard_os_pic")).WithArgs(values(picArgs)...).
		WillReturnRows(sqlmock.NewRows([]string{"PIC", "TOTAL"}).AddRow("ANDI", 3).AddRow(nil, 1))
	mock.ExpectQuery(exact("dashboard_os_business_group")).WithArgs(values(flags)...).
		WillReturnRows(sqlmock.NewRows([]string{"GRUP", "TOTAL"}).AddRow("PROPERTY", 4))

	view, err := repo.Dashboard(context.Background(), outstandingQuery(t, "PA"))
	require.NoError(t, err)
	require.Nil(t, view.RefreshedAt, "Outstanding tidak punya penyegaran")
	require.Len(t, view.Panels, 2)

	require.Equal(t, []inboxmanager.DashboardRow{
		{Cells: map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldPIC: {Text: "ANDI"}, inboxmanager.FieldJumlahKlaim: {Count: 3},
		}},
		{Cells: map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldPIC: {Text: ""}, inboxmanager.FieldJumlahKlaim: {Count: 1},
		}},
	}, view.Panels[0].Rows)
	require.Equal(t, []inboxmanager.DashboardRow{
		{Cells: map[string]inboxmanager.DashboardCell{
			inboxmanager.FieldGrupBisnis: {Text: "PROPERTY"}, inboxmanager.FieldJumlahKlaim: {Count: 4},
		}},
	}, view.Panels[1].Rows)

	// Definisi tab global tidak ikut terisi.
	require.Empty(t, mustTab(t, inboxmanager.TabOutstanding).Panels[0].Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardOutstandingErrors(t *testing.T) {
	boom := errors.New("putus")

	t.Run("panel PIC gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_os_pic")).WillReturnError(errors.New("ORA-00904: X"))

		_, err := repo.Dashboard(context.Background(), outstandingQuery(t, ""))
		require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)
		require.Contains(t, err.Error(), "dashboard_os_pic")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("panel grup gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_os_pic")).
			WillReturnRows(sqlmock.NewRows([]string{"PIC", "TOTAL"}))
		mock.ExpectQuery(exact("dashboard_os_business_group")).WillReturnError(boom)

		_, err := repo.Dashboard(context.Background(), outstandingQuery(t, ""))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_os_pic")).
			WillReturnRows(sqlmock.NewRows([]string{"PIC", "TOTAL"}).AddRow("A", "bukan angka"))

		_, err := repo.Dashboard(context.Background(), outstandingQuery(t, ""))
		require.ErrorContains(t, err, "membaca baris dashboard_os_pic")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_os_pic")).
			WillReturnRows(sqlmock.NewRows([]string{"PIC", "TOTAL"}).AddRow("A", 1).RowError(0, boom))

		_, err := repo.Dashboard(context.Background(), outstandingQuery(t, ""))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menelusuri hasil dashboard_os_pic")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDashboardRejectsNonDashboardTab(t *testing.T) {
	repo, mock := newMock(t)
	_, err := repo.Dashboard(context.Background(),
		inboxmanager.Query{Tab: mustTab(t, inboxmanager.TabMasterBengkel)})
	require.EqualError(t, err, "tab 5 bukan dashboard")
	require.NoError(t, mock.ExpectationsWereMet())
}

func produktivitasQuery(t *testing.T) inboxmanager.Query {
	t.Helper()
	q, err := inboxmanager.NewQuery(inboxmanager.QueryInput{
		Tab:    inboxmanager.TabProduktivitas,
		Period: inboxmanager.PeriodInput{Month: "2026-09"},
	}, inboxmanager.Caller{Login: "MGR", LineBusiness: "TRAVEL"})
	require.NoError(t, err)
	return q
}

var comparisonColumns = []string{"DIM", "A", "B", "C", "D", "E", "F", "G", "H"}

func TestDashboardProduktivitas(t *testing.T) {
	repo, mock := newMock(t)
	q := produktivitasQuery(t)

	periods := produktivitasPeriodArgs(q.Period)
	require.Len(t, periods, 16)
	require.Equal(t, q.Period.From, periods[0])
	require.Equal(t, q.Period.Until, periods[1])
	require.Equal(t, q.Period.PriorFrom, periods[2])
	require.Equal(t, q.Period.PriorUntil, periods[3])

	businessArgs := append(append([]any{}, periods...), lineFlags("TRAVEL")...)
	picArgs := append(append([]any{}, businessArgs...), businessArgs...)

	mock.ExpectQuery(exact("dashboard_produktivitas_business")).WithArgs(values(businessArgs)...).
		WillReturnRows(sqlmock.NewRows(comparisonColumns).AddRow("TRAVEL", 1, 2, 3, 4, 5, 6, 7, 8))
	mock.ExpectQuery(exact("dashboard_produktivitas_pic")).WithArgs(values(picArgs)...).
		WillReturnRows(sqlmock.NewRows(comparisonColumns).AddRow(nil, 0, 0, 0, 0, 0, 0, 0, 9))
	mock.ExpectQuery(exact("dashboard_refreshed_at")).
		WillReturnRows(sqlmock.NewRows([]string{"REFRESHDATE"}).AddRow(refreshed))

	view, err := repo.Dashboard(context.Background(), q)
	require.NoError(t, err)
	require.NotNil(t, view.RefreshedAt)
	require.Equal(t, refreshed, *view.RefreshedAt)

	require.Equal(t, map[string]inboxmanager.DashboardCell{
		inboxmanager.FieldDimensi:         {Text: "TRAVEL"},
		inboxmanager.FieldTotalPeriodeIni: {Count: 1},
		inboxmanager.FieldTotalPeriodeLTY: {Count: 2},
		inboxmanager.FieldAksepPeriodeIni: {Count: 3},
		inboxmanager.FieldAksepPeriodeLTY: {Count: 4},
		inboxmanager.FieldTolakPeriodeIni: {Count: 5},
		inboxmanager.FieldTolakPeriodeLTY: {Count: 6},
		inboxmanager.FieldOSPeriodeIni:    {Count: 7},
		inboxmanager.FieldOSPeriodeLTY:    {Count: 8},
	}, view.Panels[0].Rows[0].Cells)
	require.Equal(t, 9, view.Panels[1].Rows[0].Cells[inboxmanager.FieldOSPeriodeLTY].Count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardProduktivitasErrors(t *testing.T) {
	boom := errors.New("putus")

	t.Run("panel bisnis gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_produktivitas_business")).WillReturnError(boom)
		_, err := repo.Dashboard(context.Background(), produktivitasQuery(t))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("panel PIC gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_produktivitas_business")).
			WillReturnRows(sqlmock.NewRows(comparisonColumns))
		mock.ExpectQuery(exact("dashboard_produktivitas_pic")).WillReturnError(boom)
		_, err := repo.Dashboard(context.Background(), produktivitasQuery(t))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_produktivitas_business")).
			WillReturnRows(sqlmock.NewRows([]string{"DIM"}).AddRow("X"))
		_, err := repo.Dashboard(context.Background(), produktivitasQuery(t))
		require.ErrorContains(t, err, "membaca baris dashboard_produktivitas_business")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_produktivitas_business")).
			WillReturnRows(sqlmock.NewRows(comparisonColumns).
				AddRow("X", 1, 2, 3, 4, 5, 6, 7, 8).RowError(0, boom))
		_, err := repo.Dashboard(context.Background(), produktivitasQuery(t))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menelusuri hasil dashboard_produktivitas_business")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("waktu penyegaran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_produktivitas_business")).
			WillReturnRows(sqlmock.NewRows(comparisonColumns))
		mock.ExpectQuery(exact("dashboard_produktivitas_pic")).
			WillReturnRows(sqlmock.NewRows(comparisonColumns))
		mock.ExpectQuery(exact("dashboard_refreshed_at")).WillReturnError(boom)
		_, err := repo.Dashboard(context.Background(), produktivitasQuery(t))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "membaca waktu penyegaran dashboard")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func klaimQuery(t *testing.T, period inboxmanager.PeriodInput) inboxmanager.Query {
	t.Helper()
	q, err := inboxmanager.NewQuery(inboxmanager.QueryInput{Tab: inboxmanager.TabKlaim, Period: period},
		inboxmanager.Caller{Login: "MGR"})
	require.NoError(t, err)
	return q
}

var (
	klaimBusinessColumns = []string{"BISNIS", "TOTAL", "AKSEP", "TOLAK", "OS", "NAKSEP", "NTOLAK", "NOS"}
	klaimCauseColumns    = []string{"BISNIS", "PENYEBAB", "TOTAL", "AKSEP", "TOLAK", "OS", "NAKSEP", "NTOLAK", "NOS"}
)

func TestDashboardKlaimWithoutPeriodSendsAllPeriodsFlag(t *testing.T) {
	repo, mock := newMock(t)
	q := klaimQuery(t, inboxmanager.PeriodInput{})
	require.True(t, q.Period.Empty())

	args := append([]any{1, time.Time{}, time.Time{}}, lineFlags("")...)
	mock.ExpectQuery(exact("dashboard_klaim_business")).WithArgs(values(args)...).
		WillReturnRows(sqlmock.NewRows(klaimBusinessColumns).
			AddRow("PROPERTY", 10, 4, 3, 2, "1500000.50", nil, "750000"))
	mock.ExpectQuery(exact("dashboard_klaim_cause")).WithArgs(values(args)...).
		WillReturnRows(sqlmock.NewRows(klaimCauseColumns).
			AddRow("PROPERTY", "KEBAKARAN", 1, 1, 0, 0, "100", "0", "0"))
	mock.ExpectQuery(exact("dashboard_refreshed_at")).
		WillReturnRows(sqlmock.NewRows([]string{"REFRESHDATE"}))

	view, err := repo.Dashboard(context.Background(), q)
	require.NoError(t, err)
	require.Nil(t, view.RefreshedAt, "belum pernah disegarkan bukan galat")

	require.Equal(t, map[string]inboxmanager.DashboardCell{
		inboxmanager.FieldNamaBisnisDK: {Text: "PROPERTY"},
		inboxmanager.FieldTotalKlaim:   {Count: 10},
		inboxmanager.FieldJumlahAksep:  {Count: 4},
		inboxmanager.FieldJumlahTolak:  {Count: 3},
		inboxmanager.FieldJumlahOS:     {Count: 2},
		inboxmanager.FieldNilaiAksep:   {Amount: "1500000.50"},
		inboxmanager.FieldNilaiTolak:   {Amount: ""},
		inboxmanager.FieldNilaiOS:      {Amount: "750000"},
	}, view.Panels[0].Rows[0].Cells)
	require.Equal(t, inboxmanager.DashboardCell{Text: "KEBAKARAN"},
		view.Panels[1].Rows[0].Cells[inboxmanager.FieldPenyebab])
	_, hasCause := view.Panels[0].Rows[0].Cells[inboxmanager.FieldPenyebab]
	require.False(t, hasCause, "panel bisnis tidak punya kolom penyebab")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardKlaimWithPeriodSendsBounds(t *testing.T) {
	repo, mock := newMock(t)
	q := klaimQuery(t, inboxmanager.PeriodInput{From: "2026-09-01", Until: "2026-09-30"})

	args := append([]any{0, q.Period.From, q.Period.Until}, lineFlags("")...)
	mock.ExpectQuery(exact("dashboard_klaim_business")).WithArgs(values(args)...).
		WillReturnRows(sqlmock.NewRows(klaimBusinessColumns))
	mock.ExpectQuery(exact("dashboard_klaim_cause")).WithArgs(values(args)...).
		WillReturnRows(sqlmock.NewRows(klaimCauseColumns))
	mock.ExpectQuery(exact("dashboard_refreshed_at")).
		WillReturnRows(sqlmock.NewRows([]string{"REFRESHDATE"}).AddRow(nil))

	view, err := repo.Dashboard(context.Background(), q)
	require.NoError(t, err)
	require.Nil(t, view.RefreshedAt, "REFRESHDATE NULL berarti belum diketahui")
	require.Empty(t, view.Panels[0].Rows)
	require.NotNil(t, view.Panels[0].Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardKlaimErrors(t *testing.T) {
	boom := errors.New("putus")

	t.Run("panel bisnis gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_klaim_business")).WillReturnError(boom)
		_, err := repo.Dashboard(context.Background(), klaimQuery(t, inboxmanager.PeriodInput{}))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("panel penyebab gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_klaim_business")).
			WillReturnRows(sqlmock.NewRows(klaimBusinessColumns))
		mock.ExpectQuery(exact("dashboard_klaim_cause")).WillReturnError(boom)
		_, err := repo.Dashboard(context.Background(), klaimQuery(t, inboxmanager.PeriodInput{}))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_klaim_business")).
			WillReturnRows(sqlmock.NewRows([]string{"BISNIS"}).AddRow("X"))
		_, err := repo.Dashboard(context.Background(), klaimQuery(t, inboxmanager.PeriodInput{}))
		require.ErrorContains(t, err, "membaca baris dashboard_klaim_business")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_klaim_business")).
			WillReturnRows(sqlmock.NewRows(klaimBusinessColumns).
				AddRow("X", 1, 1, 0, 0, "1", "0", "0").RowError(0, boom))
		_, err := repo.Dashboard(context.Background(), klaimQuery(t, inboxmanager.PeriodInput{}))
		require.ErrorIs(t, err, boom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("waktu penyegaran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("dashboard_klaim_business")).
			WillReturnRows(sqlmock.NewRows(klaimBusinessColumns))
		mock.ExpectQuery(exact("dashboard_klaim_cause")).
			WillReturnRows(sqlmock.NewRows(klaimCauseColumns))
		mock.ExpectQuery(exact("dashboard_refreshed_at")).WillReturnError(errors.New("ORA-04063: x"))
		_, err := repo.Dashboard(context.Background(), klaimQuery(t, inboxmanager.PeriodInput{}))
		require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// queueRowValues membentuk satu baris antrean: kunci lalu satu nilai per kolom tab.
func queueRowValues(key string, cells ...any) []driver.Value {
	result := []driver.Value{key}
	for _, c := range cells {
		result = append(result, c)
	}
	return result
}

func queueColumns(tab inboxmanager.Tab) []string {
	columns := []string{"ROW_KEY"}
	for _, c := range tab.Columns {
		columns = append(columns, c.Key)
	}
	return columns
}

func TestQueueReadsKeyAndCellsAsText(t *testing.T) {
	repo, mock := newMock(t)
	tab := mustTab(t, inboxmanager.TabPaymentAkseptasi)
	require.Len(t, tab.Columns, 4)

	input := time.Date(2026, 9, 5, 13, 0, 0, 0, time.UTC)
	mock.ExpectQuery(exact("queue_payment_akseptasi")).WithoutArgs().
		WillReturnRows(sqlmock.NewRows(queueColumns(tab)).
			AddRow(queueRowValues(" K-1 ", " PNC-1 ", []byte(" AKS-9 "), nil, input)...).
			AddRow(queueRowValues("K-2", int64(42), "x", "PIC", "y")...))

	rows, err := repo.Queue(context.Background(), inboxmanager.Query{Tab: tab})
	require.NoError(t, err)
	require.Equal(t, []inboxmanager.QueueRow{
		{Key: "K-1", Cells: map[string]string{
			inboxmanager.FieldNoKlaim:      "PNC-1",
			inboxmanager.FieldNoAkseptasi:  "AKS-9",
			inboxmanager.FieldPIC:          "",
			inboxmanager.FieldTanggalInput: "05/09/2026",
		}},
		{Key: "K-2", Cells: map[string]string{
			inboxmanager.FieldNoKlaim:      "42",
			inboxmanager.FieldNoAkseptasi:  "x",
			inboxmanager.FieldPIC:          "PIC",
			inboxmanager.FieldTanggalInput: "y",
		}},
	}, rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueueErrors(t *testing.T) {
	boom := errors.New("putus")
	tab := mustTab(t, inboxmanager.TabMasterBengkel)

	t.Run("bukan antrean", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.Queue(context.Background(),
			inboxmanager.Query{Tab: mustTab(t, inboxmanager.TabOutstanding)})
		require.EqualError(t, err, "tab 1 bukan antrean")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kueri gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("queue_bengkel")).WillReturnError(boom)
		_, err := repo.Queue(context.Background(), inboxmanager.Query{Tab: tab})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menjalankan kueri queue_bengkel")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("queue_bengkel")).
			WillReturnRows(sqlmock.NewRows([]string{"ROW_KEY"}).AddRow("K"))
		_, err := repo.Queue(context.Background(), inboxmanager.Query{Tab: tab})
		require.ErrorContains(t, err, "membaca baris queue_bengkel")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		cells := make([]any, len(tab.Columns))
		mock.ExpectQuery(exact("queue_bengkel")).
			WillReturnRows(sqlmock.NewRows(queueColumns(tab)).
				AddRow(queueRowValues("K", cells...)...).RowError(0, boom))
		_, err := repo.Queue(context.Background(), inboxmanager.Query{Tab: tab})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menelusuri hasil queue_bengkel")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDecideArgsUnknownTab(t *testing.T) {
	_, _, err := decideArgs(inboxmanager.Decision{Tab: inboxmanager.Tab{Code: "99"}}, "K")
	require.EqualError(t, err, "tab 99 tidak dapat diputuskan")
}

func bengkelDecision(t *testing.T, verdict inboxmanager.Verdict, keys ...string) inboxmanager.Decision {
	t.Helper()
	return inboxmanager.Decision{
		Tab: mustTab(t, inboxmanager.TabMasterBengkel), Verdict: verdict, Keys: keys,
		Reason: " tidak lengkap ", Caller: inboxmanager.Caller{Login: "MGR"},
	}
}

// Seluruh kunci ditulis di satu transaksi; jumlah yang berubah dijumlahkan.
func TestDecideCommitsAllKeysInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(exact("decide_bengkel")).WithArgs("2", "tidak lengkap", "B-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("decide_bengkel")).WithArgs("2", "tidak lengkap", "B-2").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	changed, err := repo.Decide(context.Background(),
		bengkelDecision(t, inboxmanager.VerdictReject, "B-1", "B-2"))
	require.NoError(t, err)
	require.Equal(t, 1, changed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDecideErrorsRollBack(t *testing.T) {
	boom := errors.New("putus")

	t.Run("memulai transaksi gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(boom)
		_, err := repo.Decide(context.Background(), bengkelDecision(t, inboxmanager.VerdictApprove, "B-1"))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "memulai transaksi keputusan 5")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kunci nomor rangka salah bentuk", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		_, err := repo.Decide(context.Background(), inboxmanager.Decision{
			Tab: mustTab(t, inboxmanager.TabNomorRangka), Verdict: inboxmanager.VerdictApprove,
			Keys: []string{"A|B"},
		})
		require.ErrorContains(t, err, "gabungan empat kolom")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pernyataan gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact("decide_bengkel")).WillReturnError(errors.New("ORA-01008: not bound"))
		mock.ExpectRollback()
		_, err := repo.Decide(context.Background(), bengkelDecision(t, inboxmanager.VerdictApprove, "B-1"))
		require.ErrorContains(t, err, "menjalankan decide_bengkel")
		require.ErrorContains(t, err, "jumlah argumen bind")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("jumlah baris tidak terbaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact("decide_bengkel")).WillReturnResult(sqlmock.NewErrorResult(boom))
		mock.ExpectRollback()
		_, err := repo.Decide(context.Background(), bengkelDecision(t, inboxmanager.VerdictApprove, "B-1"))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "membaca jumlah baris terubah decide_bengkel")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact("decide_bengkel")).WithArgs("1", "tidak lengkap", "B-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(boom)
		_, err := repo.Decide(context.Background(), bengkelDecision(t, inboxmanager.VerdictApprove, "B-1"))
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menyimpan keputusan 5")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestLineBusinessFor(t *testing.T) {
	t.Run("login kosong tidak mengirim kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		line, err := repo.LineBusinessFor(context.Background(), "  ")
		require.NoError(t, err)
		require.Empty(t, line)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("login diseragamkan dan hasil dipangkas", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("line_business_for")).WithArgs("MANAGER").
			WillReturnRows(sqlmock.NewRows([]string{"LINE_BUSINESS"}).AddRow(" PA "))
		line, err := repo.LineBusinessFor(context.Background(), " manager ")
		require.NoError(t, err)
		require.Equal(t, "PA", line)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tanpa baris bukan galat", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("line_business_for")).
			WillReturnRows(sqlmock.NewRows([]string{"LINE_BUSINESS"}))
		line, err := repo.LineBusinessFor(context.Background(), "x")
		require.NoError(t, err)
		require.Empty(t, line)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat dibungkus", func(t *testing.T) {
		repo, mock := newMock(t)
		boom := errors.New("putus")
		mock.ExpectQuery(exact("line_business_for")).WillReturnError(boom)
		_, err := repo.LineBusinessFor(context.Background(), "x")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "membaca lini bisnis petugas X")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	empty := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"X"}) }
	one := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"X"}).AddRow(1) }
	boom := errors.New("ORA-00942")

	t.Run("ketiganya terbaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("check_table")).WillReturnRows(empty())
		mock.ExpectQuery(exact("check_dashboard_snapshot")).WillReturnRows(one())
		mock.ExpectQuery(exact("check_line_business")).WillReturnRows(empty())
		require.NoError(t, repo.CheckTable(context.Background()))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tabel kerja gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("check_table")).WillReturnError(boom)
		err := repo.CheckTable(context.Background())
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "POOLDATA.T_CLAIMLIST_ADMIN")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cuplikan dashboard gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("check_table")).WillReturnRows(one())
		mock.ExpectQuery(exact("check_dashboard_snapshot")).WillReturnError(boom)
		err := repo.CheckTable(context.Background())
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "POOLDATA.PEGA_DASHBOARDPNC")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kolom lini bisnis gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("check_table")).WillReturnRows(one())
		mock.ExpectQuery(exact("check_dashboard_snapshot")).WillReturnRows(one())
		mock.ExpectQuery(exact("check_line_business")).WillReturnError(boom)
		err := repo.CheckTable(context.Background())
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "LINE_BUSINESS")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAsTextFormatsByType(t *testing.T) {
	require.Equal(t, "", asText(nil))
	require.Equal(t, "a", asText(" a "))
	require.Equal(t, "b", asText([]byte(" b ")))
	require.Equal(t, "31/12/2026", asText(time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)))
	require.Equal(t, "3.5", asText(3.5))
}

func TestNullIfEmpty(t *testing.T) {
	require.Nil(t, nullIfEmpty("   "))
	require.Equal(t, "x", nullIfEmpty(" x "))
}
