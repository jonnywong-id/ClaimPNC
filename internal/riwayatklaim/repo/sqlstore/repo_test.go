package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
)

func newDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func criteriaOf(code, text string, date *time.Time) riwayatklaim.Criteria {
	searchType, _ := riwayatklaim.FindSearchType(code)
	return riwayatklaim.Criteria{Type: searchType, Text: text, SearchDate: date}
}

var claimColumns = []string{
	"REFERENCE", "NUMBER", "POLICY", "INSURED", "LOSS_DATE", "BUSINESS", "BRANCH", "STATUS",
	"POSITION", "CLOSE_DATE", "CLOSE_NOTE", "PIC", "ACCEPTANCE", "AUCTION", "ITEM", "BIRTH",
}

func TestSearchCountsThenReadsPage(t *testing.T) {
	db, mock := newDB(t)
	loss := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(counted("search_policy_number")).WithArgs("POL").
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(3))
	mock.ExpectQuery(paged("search_policy_number")).WithArgs("POL", 2, 2).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("R", "PNC-1", "POL", "Nama", loss, "Fire", "Pusat", "Open", "Register",
				loss, "catatan", "PIC", "AKS", "BL", "Objek", loss).
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := NewRepo(db).Search(context.Background(),
		criteriaOf(riwayatklaim.TypePolicyNumber, "POL", nil),
		riwayatklaim.Pagination{Page: 2, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Claims, 2)
	require.Equal(t, riwayatklaim.ClaimHistory{
		Reference: "R", Number: "PNC-1", PolicyNumber: "POL", InsuredName: "Nama",
		LossDate: &loss, BusinessName: "Fire", BranchName: "Pusat", WorkStatus: "Open",
		ClaimPosition: "Register", CloseDate: &loss, CloseNote: "catatan", TechnicalPIC: "PIC",
		AcceptanceNumber: "AKS", AuctionHouseID: "BL", InsuredItemName: "Objek", BirthDate: &loss,
	}, page.Claims[0])
	require.Nil(t, page.Claims[1].LossDate)
	require.Nil(t, page.Claims[1].BirthDate)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchSkipsPageQueryWhenEmptyOrBeyondTotal(t *testing.T) {
	db, mock := newDB(t)
	day := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(counted("search_loss_date")).WithArgs(day).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	page, err := NewRepo(db).Search(context.Background(),
		criteriaOf(riwayatklaim.TypeLossDate, "", &day), riwayatklaim.Pagination{})
	require.NoError(t, err)
	require.Empty(t, page.Claims)
	require.NotNil(t, page.Claims)

	// Tipe tanggal lahir mengirim tanggal pencarian yang kosong — NULL.
	mock.ExpectQuery(counted("search_birth_date")).WithArgs(nil).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(5))
	page, err = NewRepo(db).Search(context.Background(),
		criteriaOf(riwayatklaim.TypeBirthDate, "", nil), riwayatklaim.Pagination{Page: 9, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 5, page.Total)
	require.Empty(t, page.Claims)
	require.Equal(t, 9, page.Pagination.Page)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchRejectsTypeWithoutQuery(t *testing.T) {
	db, mock := newDB(t)
	_, err := NewRepo(db).Search(context.Background(),
		criteriaOf(riwayatklaim.TypeAccountNumber, "", nil), riwayatklaim.Pagination{})
	require.EqualError(t, err, `riwayatklaim/sqlstore: tipe pencarian "12" belum punya kueri`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSearchWrapsErrors(t *testing.T) {
	c := criteriaOf(riwayatklaim.TypeClaimNumber, "PNC", nil)
	count := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(counted("search_claim_number")).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	}

	db, mock := newDB(t)
	mock.ExpectQuery(counted("search_claim_number")).WillReturnError(errors.New("ora"))
	_, err := NewRepo(db).Search(context.Background(), c, riwayatklaim.Pagination{})
	require.ErrorContains(t, err, "menghitung hasil pencarian: ora")

	db, mock = newDB(t)
	count(mock)
	mock.ExpectQuery(paged("search_claim_number")).WillReturnError(errors.New("ora"))
	_, err = NewRepo(db).Search(context.Background(), c, riwayatklaim.Pagination{})
	require.ErrorContains(t, err, "membaca hasil pencarian: ora")

	db, mock = newDB(t)
	count(mock)
	mock.ExpectQuery(paged("search_claim_number")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = NewRepo(db).Search(context.Background(), c, riwayatklaim.Pagination{})
	require.ErrorContains(t, err, "memindai baris riwayat klaim")

	db, mock = newDB(t)
	count(mock)
	mock.ExpectQuery(paged("search_claim_number")).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow(make([]driver.Value, len(claimColumns))...).
			RowError(0, errors.New("putus")))
	_, err = NewRepo(db).Search(context.Background(), c, riwayatklaim.Pagination{})
	require.ErrorContains(t, err, "membaca hasil pencarian: putus")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProtectionFind(t *testing.T) {
	db, mock := newDB(t)
	repo := NewProtectionRepo(db)

	mock.ExpectQuery(query("protection_find")).WithArgs("adminpnc", "MODUL").
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D", "E", "F"}).
			AddRow(50, 7, " ya ", "Tidak", "YA", "PNCSearchKlaim, ,Lain"))
	protection, registered, err := repo.Find(context.Background(), " adminpnc ", " MODUL ")
	require.NoError(t, err)
	require.True(t, registered)
	require.Equal(t, riwayatklaim.Protection{
		Login: "adminpnc", SearchQuota: 50, ViewQuota: 7, MaskPhone: true, MaskIDCard: true,
		SubModules: []string{"PNCSearchKlaim", "Lain"},
	}, protection)

	mock.ExpectQuery(query("protection_find")).WillReturnError(sql.ErrNoRows)
	_, registered, err = repo.Find(context.Background(), "x", "m")
	require.NoError(t, err)
	require.False(t, registered)

	mock.ExpectQuery(query("protection_find")).WillReturnError(errors.New("ora"))
	_, _, err = repo.Find(context.Background(), "x", "m")
	require.ErrorContains(t, err, "membaca master proteksi data: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProtectionCountUsageAndTableReady(t *testing.T) {
	db, mock := newDB(t)
	repo := NewProtectionRepo(db)

	mock.ExpectQuery(query("protection_count_usage")).WithArgs("a", "m").
		WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(4))
	used, err := repo.CountUsage(context.Background(), " a ", " m ")
	require.NoError(t, err)
	require.Equal(t, 4, used)

	mock.ExpectQuery(query("protection_count_usage")).WillReturnError(errors.New("ora"))
	_, err = repo.CountUsage(context.Background(), "a", "m")
	require.ErrorContains(t, err, "menghitung pemakaian jatah proteksi: ora")

	mock.ExpectQuery(query("protection_check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(1))
	ready, err := repo.TableReady(context.Background())
	require.NoError(t, err)
	require.True(t, ready)

	mock.ExpectQuery(query("protection_check_table")).
		WillReturnRows(sqlmock.NewRows([]string{"T"}).AddRow(0))
	ready, err = repo.TableReady(context.Background())
	require.NoError(t, err)
	require.False(t, ready)

	mock.ExpectQuery(query("protection_check_table")).WillReturnError(errors.New("ora"))
	_, err = repo.TableReady(context.Background())
	require.ErrorContains(t, err, "memeriksa tabel "+TableName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProtectionRecordUsageBindsNormalizedValues(t *testing.T) {
	db, mock := newDB(t)
	repo := NewProtectionRepo(db)
	at := time.Date(2026, 9, 20, 10, 0, 0, 0, time.FixedZone("WIB", 7*3600))
	long := strings.Repeat("a", maxSearchValue+20)

	mock.ExpectExec(query("protection_record_usage")).
		WithArgs("adminpnc", "MODUL", "7", strings.Repeat("a", maxSearchValue), 1, at.UTC()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.RecordUsage(context.Background(), riwayatklaim.Usage{
		Login: " adminpnc ", Module: "MODUL", SearchTypeCode: "7", SearchValue: long,
		ConsumesQuota: true, At: at,
	}))

	mock.ExpectExec(query("protection_record_usage")).
		WithArgs("adminpnc", "MODUL", nil, nil, 0, at.UTC()).
		WillReturnError(errors.New("ora"))
	err := repo.RecordUsage(context.Background(), riwayatklaim.Usage{
		Login: "adminpnc", Module: "MODUL", SearchValue: "  ", At: at,
	})
	require.ErrorContains(t, err, "mencatat pemakaian jatah proteksi: ora")
	require.NoError(t, mock.ExpectationsWereMet())
}
