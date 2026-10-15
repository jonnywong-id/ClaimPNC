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

	"claim-pnc/internal/inboxprogressclaim"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// literal mengubah teks kueri menjadi pola regexp yang cocok persis.
func literal(text string) string { return "^" + regexp.QuoteMeta(text) + "$" }

// positionsQuery adalah kueri posisi setelah penanda IN diisi untuk n nomor klaim.
func positionsQuery(n int) string {
	return literal(strings.Replace(query("positions"), "%s", inList(1, n), 1))
}

var (
	registered = time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	followUp   = time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)
	todayDate  = time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
)

func claimRows() *sqlmock.Rows {
	return sqlmock.NewRows(claimColumns).
		AddRow("PNCN.26.0101", "16.001", "PT Contoh", registered, registered, "LGB", "ADMIN", followUp, registered, "1").
		AddRow("PNCN.26.0101", "16.001", "PT Contoh", nil, nil, nil, nil, nil, nil, nil).
		AddRow("", "16.002", "Tanpa Nomor", nil, nil, nil, nil, nil, nil, nil)
}

func TestListClaimsOutstandingReadsCountPageAndPositions(t *testing.T) {
	db, mock := newMock(t)
	q := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding, Keyword: "PNCN", Today: todayDate}

	mock.ExpectQuery(literal(query("claims_outstanding_count"))).WithArgs("PNCN").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(3))
	mock.ExpectQuery(literal(query("claims_outstanding_list"))).WithArgs("PNCN", 0, 15).
		WillReturnRows(claimRows())
	// Nomor kembar dan nomor kosong tidak ikut dikirim: hanya satu penanda.
	mock.ExpectQuery(positionsQuery(1)).WithArgs("PNCN.26.0101").WillReturnRows(
		sqlmock.NewRows(positionColumns).
			AddRow("PNCN.26.0101", "SURVEY", "Berjalan", "Menunggu", followUp).
			AddRow("PNCN.26.0101", "KOMITE", nil, nil, nil).
			AddRow("LAIN", "X", "", "", nil))

	page, err := NewRepo(db).ListClaims(context.Background(), q, inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Equal(t, inboxprogressclaim.Pagination{Page: 1, Size: 15}, page.Pagination)
	require.Len(t, page.Items, 3)

	first := page.Items[0]
	require.Equal(t, "PNCN.26.0101", first.ClaimNumber)
	require.Equal(t, registered, *first.RegisterDate)
	require.Equal(t, followUp, *first.EarliestFollowUp)
	require.Equal(t, "1", first.ProdKe)
	require.Equal(t, []inboxprogressclaim.Position{
		{Name: "SURVEY", Status1: "Berjalan", Status2: "Menunggu", NextFollowUp: &followUp},
		{Name: "KOMITE"},
	}, first.Positions)

	// Baris kembar memakai indeks pertama; yang kedua tetap tanpa posisi.
	require.Nil(t, page.Items[1].RegisterDate)
	require.Empty(t, page.Items[1].Positions)
	require.Empty(t, page.Items[2].Positions)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimsNextFollowUpSendsNullKeywordAndToday(t *testing.T) {
	db, mock := newMock(t)
	q := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewNextFollowUp, Today: todayDate}

	mock.ExpectQuery(literal(query("claims_next_fu_count"))).WithArgs(nil, todayDate).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(40))
	mock.ExpectQuery(literal(query("claims_next_fu_list"))).WithArgs(nil, todayDate, 20, 10).
		WillReturnRows(sqlmock.NewRows(claimColumns))

	page, err := NewRepo(db).ListClaims(context.Background(), q, inboxprogressclaim.Pagination{Page: 3, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 40, page.Total)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimsStopsAfterCountWhenNothingOnPage(t *testing.T) {
	db, mock := newMock(t)
	q := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding}

	// Nol baris.
	mock.ExpectQuery(literal(query("claims_outstanding_count"))).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
	page, err := NewRepo(db).ListClaims(context.Background(), q, inboxprogressclaim.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 0, page.Total)

	// Halaman melampaui jumlah baris.
	mock.ExpectQuery(literal(query("claims_outstanding_count"))).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(5))
	page, err = NewRepo(db).ListClaims(context.Background(), q, inboxprogressclaim.Pagination{Page: 2, Size: 5})
	require.NoError(t, err)
	require.Equal(t, 5, page.Total)
	require.Empty(t, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimsUnknownViewHasNoQuery(t *testing.T) {
	db, mock := newMock(t)
	_, err := NewRepo(db).ListClaims(context.Background(), inboxprogressclaim.ClaimQuery{View: "per-pic"}, inboxprogressclaim.Pagination{})
	require.ErrorContains(t, err, `bagian "per-pic" belum punya kueri`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimsFailures(t *testing.T) {
	q := inboxprogressclaim.ClaimQuery{View: inboxprogressclaim.ViewOutstanding}
	count := func(m sqlmock.Sqlmock) {
		m.ExpectQuery(literal(query("claims_outstanding_count"))).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	}
	oneRow := func() *sqlmock.Rows {
		return sqlmock.NewRows(claimColumns).AddRow("A", "", "", nil, nil, nil, nil, nil, nil, nil)
	}
	cases := []struct {
		name  string
		setup func(sqlmock.Sqlmock)
		want  string
	}{
		{"count", func(m sqlmock.Sqlmock) {
			m.ExpectQuery(literal(query("claims_outstanding_count"))).WillReturnError(errDB)
		}, "menghitung baris kueri claims_outstanding_count"},
		{"list", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnError(errDB)
		}, "menjalankan kueri claims_outstanding_list"},
		{"scan", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow("A"))
		}, "membaca baris kueri claims_outstanding_list"},
		{"rows", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnRows(oneRow().RowError(0, errDB))
		}, "menelusuri hasil kueri claims_outstanding_list"},
		{"positions query", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnRows(oneRow())
			m.ExpectQuery(positionsQuery(1)).WillReturnError(errDB)
		}, "menjalankan kueri positions"},
		{"positions scan", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnRows(oneRow())
			m.ExpectQuery(positionsQuery(1)).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow("A"))
		}, "membaca baris kueri positions"},
		{"positions rows", func(m sqlmock.Sqlmock) {
			count(m)
			m.ExpectQuery(literal(query("claims_outstanding_list"))).WillReturnRows(oneRow())
			m.ExpectQuery(positionsQuery(1)).WillReturnRows(
				sqlmock.NewRows(positionColumns).AddRow("A", "", "", "", nil).RowError(0, errDB))
		}, "menelusuri hasil kueri positions"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			c.setup(mock)
			_, err := NewRepo(db).ListClaims(context.Background(), q, inboxprogressclaim.Pagination{})
			require.Error(t, err)
			require.ErrorContains(t, err, c.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAttachPositionsSkipsQueryWhenNoClaimNumber(t *testing.T) {
	db, mock := newMock(t)
	items := []inboxprogressclaim.ClaimRow{{ClaimNumber: ""}}
	require.NoError(t, NewRepo(db).attachPositions(context.Background(), items))
	require.Equal(t, []inboxprogressclaim.Position{}, items[0].Positions)
	require.NoError(t, NewRepo(db).attachPositions(context.Background(), nil))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPICSummaryMapsCounters(t *testing.T) {
	db, mock := newMock(t)
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	q := inboxprogressclaim.PICQuery{
		Business: inboxprogressclaim.BusinessNonMBU,
		From:     &from,
		Caller:   inboxprogressclaim.Caller{Login: "ADMINKLAIM"},
		Today:    todayDate,
	}
	mock.ExpectQuery(literal(query("pic_summary"))).WithArgs("NONMBU", "ADMINKLAIM", todayDate, from, nil).
		WillReturnRows(sqlmock.NewRows(picColumns).
			AddRow("ADMINKLAIM", 12, 37, 2, 30, 5).
			AddRow(nil, nil, nil, nil, nil, nil))

	got, err := NewRepo(db).ListPICSummary(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, []inboxprogressclaim.PICSummary{
		{PIC: "ADMINKLAIM", ClaimCount: 12, UpdateCount: 37, DueTodayCount: 2, OnTimeCount: 30, LateCount: 5},
		{},
	}, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListPICSummaryFailures(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)
	q := inboxprogressclaim.PICQuery{Business: inboxprogressclaim.BusinessPA}

	mock.ExpectQuery(literal(query("pic_summary"))).WillReturnError(errDB)
	_, err := repo.ListPICSummary(context.Background(), q)
	require.ErrorContains(t, err, "menjalankan kueri pic_summary")

	mock.ExpectQuery(literal(query("pic_summary"))).WillReturnRows(sqlmock.NewRows([]string{"PIC"}).AddRow("A"))
	_, err = repo.ListPICSummary(context.Background(), q)
	require.ErrorContains(t, err, "membaca baris kueri pic_summary")

	mock.ExpectQuery(literal(query("pic_summary"))).WillReturnRows(
		sqlmock.NewRows(picColumns).AddRow("A", 1, 1, 1, 1, 1).RowError(0, errDB))
	_, err = repo.ListPICSummary(context.Background(), q)
	require.ErrorContains(t, err, "menelusuri hasil kueri pic_summary")

	// Tanpa baris: senarai kosong, bukan nil.
	mock.ExpectQuery(literal(query("pic_summary"))).WillReturnRows(sqlmock.NewRows(picColumns))
	got, err := repo.ListPICSummary(context.Background(), q)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTable(t *testing.T) {
	db, mock := newMock(t)
	repo := NewRepo(db)

	mock.ExpectQuery(literal(query("check_table"))).WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(literal(query("check_table"))).WillReturnError(errDB)
	err := repo.CheckTable(context.Background())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca POOLDATA.PEGA_DASHBOARDPNC")
	require.NoError(t, mock.ExpectationsWereMet())
}
