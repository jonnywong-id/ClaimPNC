package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
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

// registrationQuery adalah tab Registrasi SC tanpa pencarian.
func registrationQuery(t *testing.T, keyword string) inboxservicecenter.Query {
	t.Helper()
	q, err := inboxservicecenter.NewQuery(
		inboxservicecenter.QueryInput{Tab: inboxservicecenter.TabRegistration, Keyword: keyword},
		inboxservicecenter.Caller{Login: "picsc"},
	)
	require.NoError(t, err)
	return q
}

// filterValues menyerahkan argumen penyaring sebagai driver.Value untuk WithArgs.
func filterValues(q inboxservicecenter.Query) []driver.Value {
	args := filterArgs(q)
	values := make([]driver.Value, 0, len(args))
	for _, a := range args {
		values = append(values, a)
	}
	return values
}

var inputDate = time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)

func claimRow(id string, date any) []driver.Value {
	return []driver.Value{
		" " + id + " ", "1000101", "PNCN.26.0101", "90-001", "Nasabah", "Gadget", "PICSC",
		date, "IMEI-1", nil, "1", "PICSC", "KOMITE",
	}
}

func TestListPaginatedSendsOffsetAndLimit(t *testing.T) {
	repo, mock := newMock(t)
	q := registrationQuery(t, "")
	filters := filterValues(q)

	require.Equal(t, []driver.Value{"Y", "N", nil, nil, "PICSC", nil, "", "", "", "", nil, "", "", "", ""}, filters)

	mock.ExpectQuery(exact("count_claims")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(3))
	mock.ExpectQuery(exact("list_claims")).WithArgs(append(filters, 2, 2)...).
		WillReturnRows(sqlmock.NewRows(resultColumns).
			AddRow(claimRow("SC-3", inputDate)...).
			AddRow(claimRow("SC-4", nil)...))

	page, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.True(t, page.Paginated)
	require.Equal(t, inboxservicecenter.Pagination{Page: 2, Size: 2}, page.Pagination)
	require.Len(t, page.Items, 2)

	first := page.Items[0]
	require.Equal(t, inboxservicecenter.ServiceClaim{
		ID: "SC-3", RepairID: "1000101", ClaimNumber: "PNCN.26.0101", PolicyNumber: "90-001",
		CustomerName: "Nasabah", Type: "Gadget", TechnicalPIC: "PICSC", InputDate: &inputDate,
		IMEI: "IMEI-1", ApprovalStatus: "", RepairStatus: "1", Owner: "PICSC",
		CommitteeApprover: "KOMITE",
	}, first)
	require.Nil(t, page.Items[1].InputDate)

	require.NoError(t, mock.ExpectationsWereMet())
}

// Saat mencari, offset 0 dan batasnya sebesar jumlah baris yang cocok.
func TestListSearchFetchesAllMatches(t *testing.T) {
	repo, mock := newMock(t)
	q := registrationQuery(t, "abc")
	filters := filterValues(q)

	mock.ExpectQuery(exact("count_claims")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
	mock.ExpectQuery(exact("list_claims")).WithArgs(append(filters, 0, 1)...).
		WillReturnRows(sqlmock.NewRows(resultColumns).AddRow(claimRow("SC-1", inputDate)...))

	page, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{Page: 4, Size: 10})
	require.NoError(t, err)
	require.False(t, page.Paginated)
	require.Len(t, page.Items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tanpa baris yang cocok, kueri daftar tidak dikirim sama sekali.
func TestListEmptySkipsListQuery(t *testing.T) {
	for _, keyword := range []string{"", "abc"} {
		repo, mock := newMock(t)
		q := registrationQuery(t, keyword)

		mock.ExpectQuery(exact("count_claims")).WithArgs(filterValues(q)...).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))

		page, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{})
		require.NoError(t, err)
		require.Equal(t, 0, page.Total)
		require.NotNil(t, page.Items)
		require.Empty(t, page.Items)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestListErrors(t *testing.T) {
	boom := errors.New("putus")

	t.Run("hitung gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		q := registrationQuery(t, "")
		mock.ExpectQuery(exact("count_claims")).WillReturnError(boom)

		_, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "count_claims")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("daftar gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		q := registrationQuery(t, "")
		mock.ExpectQuery(exact("count_claims")).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact("list_claims")).WillReturnError(boom)

		_, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menjalankan kueri list_claims")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		q := registrationQuery(t, "")
		mock.ExpectQuery(exact("count_claims")).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact("list_claims")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("SC-1"))

		_, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "membaca baris kueri list_claims")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		q := registrationQuery(t, "")
		mock.ExpectQuery(exact("count_claims")).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(1))
		mock.ExpectQuery(exact("list_claims")).
			WillReturnRows(sqlmock.NewRows(resultColumns).
				AddRow(claimRow("SC-1", inputDate)...).RowError(0, boom))

		_, err := repo.List(context.Background(), q, inboxservicecenter.Pagination{})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menelusuri hasil kueri list_claims")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCheckTable(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))
	require.NoError(t, repo.CheckTable(context.Background()))

	mock.ExpectQuery(exact("check_table")).WillReturnError(errors.New("ORA-00942"))
	err := repo.CheckTable(context.Background())
	require.ErrorContains(t, err, "POOLDATA.T_KLAIM_PORTAL_REKANAN")
	require.ErrorContains(t, err, "ORA-00942")
	require.NoError(t, mock.ExpectationsWereMet())
}

// isTimeColumn menandai kolom tanggal pada find_detail.
func isTimeColumn(column string) bool {
	switch column {
	case "INPUTDATE", "STARTDATE", "ENDDATE", "ACKNOWLEDGE_DATE", "ASSIGNED_DATE",
		"COMPLETED_DATE", "RELEASE_DATE", "INVOICE_DATE", "ESTIMATED_PICKUP_DATE",
		"PICKUP_COURIER_DATE":
		return true
	}
	return false
}

// detailRow mengisi setiap kolom teks dengan " <nama kolom> " supaya pemetaannya terbaca.
func detailRow(dates any) []driver.Value {
	values := make([]driver.Value, 0, len(detailColumns))
	for _, column := range detailColumns {
		if isTimeColumn(column) {
			values = append(values, dates)
			continue
		}
		values = append(values, " "+column+" ")
	}
	return values
}

func TestFindDetailMapsEveryColumn(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("find_detail")).WithArgs("SC-1", "PICSC").
		WillReturnRows(sqlmock.NewRows(detailColumns).AddRow(detailRow(inputDate)...))

	got, err := repo.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: " sc-1 ", Caller: inboxservicecenter.Caller{Login: " picsc "},
	})
	require.NoError(t, err)

	// Sampel dari setiap kelompok, termasuk kolom yang namanya berbeda dari isiannya.
	require.Equal(t, "ID", got.ID)
	require.Equal(t, "LOGIN", got.Owner)
	require.Equal(t, "QQNAME", got.InsuredName)
	require.Equal(t, "QUOTATION_AMOUNT", got.QuotationValue)
	require.Equal(t, "OBJECT", got.ObjectName)
	require.Equal(t, "CUST_ARRIVAL", got.CustomerReply)
	require.Equal(t, "STS_APPROVAL", got.ApprovalStatus)
	require.Equal(t, "ACCESSORIESLAINYA", got.AccessoriesOther)
	require.Equal(t, "DEDUCAPPROVE", got.DeductibleAppr)
	require.Equal(t, "TOTAL_FEEAPPROVE", got.TotalFeeAppr)
	require.Equal(t, "CGARGERCABLE", got.ChargerCable)
	require.Equal(t, "BOXUNIT", got.BoxUnit)
	require.Equal(t, "CASE", got.UnitCase)
	for _, at := range []*time.Time{
		got.InputDate, got.WarrantyStart, got.WarrantyEnd, got.AcknowledgeDate, got.AssignedDate,
		got.CompletedDate, got.ReleaseDate, got.InvoiceDate, got.EstimatedPickupAt, got.CourierPickupAt,
	} {
		require.NotNil(t, at)
		require.Equal(t, inputDate, *at)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDetailNullDatesStayNil(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("find_detail")).
		WillReturnRows(sqlmock.NewRows(detailColumns).AddRow(detailRow(nil)...))

	got, err := repo.FindDetail(context.Background(), inboxservicecenter.DetailQuery{
		ID: "SC-1", Caller: inboxservicecenter.Caller{Login: "PICSC"},
	})
	require.NoError(t, err)
	require.Nil(t, got.InputDate)
	require.Nil(t, got.CourierPickupAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDetailErrors(t *testing.T) {
	t.Run("tidak ada baris menjadi ErrNotFound", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("find_detail")).WillReturnError(sql.ErrNoRows)

		_, err := repo.FindDetail(context.Background(), inboxservicecenter.DetailQuery{ID: "X"})
		require.Equal(t, inboxservicecenter.ErrNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat lain dibungkus", func(t *testing.T) {
		repo, mock := newMock(t)
		boom := errors.New("putus")
		mock.ExpectQuery(exact("find_detail")).WillReturnError(boom)

		_, err := repo.FindDetail(context.Background(), inboxservicecenter.DetailQuery{ID: "X"})
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menjalankan kueri find_detail")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListProgress(t *testing.T) {
	columns := []string{"REPAIRID", "INSERTDATE", "NOTEPROGRESS", "USERUPDATE"}

	t.Run("REPAIRID kosong tidak mengirim kueri", func(t *testing.T) {
		repo, mock := newMock(t)
		notes, err := repo.ListProgress(context.Background(), "  ")
		require.NoError(t, err)
		require.NotNil(t, notes)
		require.Empty(t, notes)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("baris dipetakan dan dipangkas", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_progress")).WithArgs("1000101").
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow(" 1000101 ", inputDate, " catatan ", " PICSC ").
				AddRow("1000101", nil, nil, nil))

		notes, err := repo.ListProgress(context.Background(), " 1000101 ")
		require.NoError(t, err)
		require.Equal(t, []inboxservicecenter.ProgressNote{
			{ClaimID: "1000101", RecordedAt: &inputDate, Note: "catatan", RecordedBy: "PICSC"},
			{ClaimID: "1000101"},
		}, notes)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kueri gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		boom := errors.New("putus")
		mock.ExpectQuery(exact("list_progress")).WillReturnError(boom)

		_, err := repo.ListProgress(context.Background(), "1")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menjalankan kueri list_progress")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pindai gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_progress")).
			WillReturnRows(sqlmock.NewRows([]string{"REPAIRID"}).AddRow("1"))

		_, err := repo.ListProgress(context.Background(), "1")
		require.ErrorContains(t, err, "membaca baris kueri list_progress")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penelusuran gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		boom := errors.New("putus")
		mock.ExpectQuery(exact("list_progress")).
			WillReturnRows(sqlmock.NewRows(columns).
				AddRow("1", inputDate, "a", "b").RowError(0, boom))

		_, err := repo.ListProgress(context.Background(), "1")
		require.ErrorIs(t, err, boom)
		require.Contains(t, err.Error(), "menelusuri hasil kueri list_progress")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUnknownQueryPanics(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxservicecenter/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { _ = query("tidak_ada") })
}

func TestCodePairShapes(t *testing.T) {
	a, b := codePair(nil)
	require.Nil(t, a)
	require.Nil(t, b)
	a, b = codePair([]string{"1"})
	require.Equal(t, "1", a)
	require.Equal(t, "1", b)
	a, b = codePair([]string{"2", "3", "9"})
	require.Equal(t, "2", a)
	require.Equal(t, "3", b)
}
