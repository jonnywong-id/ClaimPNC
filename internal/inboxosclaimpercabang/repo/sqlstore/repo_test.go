package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
)

// newMockRepo membentuk Repo di atas sqlmock dengan pencocokan kueri berbasis regex.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exactQuery mengubah teks kueri bernama menjadi pola regex yang cocok persis.
func exactQuery(name string) string {
	return "^" + regexp.QuoteMeta(query(name)) + "$"
}

var errBoom = errors.New("boom")

// branchQuery adalah kueri dengan cabang yang sudah diterjemahkan.
var branchQuery = inboxosclaimpercabang.Query{
	Branch: inboxosclaimpercabang.Branch{Code: "100100", Name: "JAKARTA"},
}

// listRowValues mengembalikan ke-20 nilai kolom awalan beserta tanggal acuannya.
func listRowValues(at time.Time) []driver.Value {
	return []driver.Value{
		"JAKARTA", "100100", "DIRECT", "FIRE",
		"POL-1", "PT SATU", "PNC-1", at, at,
		"REKOM", int64(150000), at,
		"S1", "S2", "PIC", "NOTE",
		"ADJ", "COL", "KRONO", int64(1),
	}
}

func TestBranchOfEmptyCodeSkipsTheDatabase(t *testing.T) {
	repo, mock := newMockRepo(t)

	branch, found, err := repo.BranchOf(context.Background(), "   ")
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, inboxosclaimpercabang.Branch{}, branch)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBranchOfTrimsAndReturnsTheBranch(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("branch_of")).WithArgs("123").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAME"}).AddRow(" 100100 ", " JAKARTA "))

	branch, found, err := repo.BranchOf(context.Background(), "123")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, inboxosclaimpercabang.Branch{Code: "100100", Name: "JAKARTA"}, branch)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBranchOfNoRowsIsUnknownNotError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("branch_of")).WithArgs("999").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAME"}))

	_, found, err := repo.BranchOf(context.Background(), "999")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBranchOfBlankCodeIsUnknown(t *testing.T) {
	// Kode cabang kosong dari basis data diperlakukan sebagai tidak dikenal.
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("branch_of")).WithArgs("123").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAME"}).AddRow(nil, "X"))

	_, found, err := repo.BranchOf(context.Background(), "123")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBranchOfWrapsQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("branch_of")).WithArgs("123").WillReturnError(errBoom)

	_, found, err := repo.BranchOf(context.Background(), "123")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "POOLDATA.BRANCH")
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListMapsRowsAndPaginatesInTheDatabase(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

	values := append(listRowValues(at), int64(42))
	rows := sqlmock.NewRows(listColumns).AddRow(values...).
		AddRow("B", "100100", nil, nil, nil, nil, "PNC-2", nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil, nil, int64(42))
	// Halaman 3 ukuran 10 berarti melewati 20 baris.
	// Tiga bind di tengah adalah kotak cari yang kosong: NULL berarti "tampilkan semua",
	// dan kedua pola dibiarkan kosong karena cabang `IS NULL` sudah memutus perbandingannya.
	mock.ExpectQuery(exactQuery("list")).
		WithArgs("100100", nil, "", "", 20, 10).
		WillReturnRows(rows)

	page, err := repo.List(context.Background(), branchQuery,
		inboxosclaimpercabang.Pagination{Page: 3, Size: 10})
	require.NoError(t, err)
	require.Equal(t, 42, page.Total)
	require.Equal(t, inboxosclaimpercabang.Pagination{Page: 3, Size: 10}, page.Pagination)
	require.Len(t, page.Items, 2)

	first := page.Items[0]
	require.Equal(t, "JAKARTA", first.BranchName)
	require.Equal(t, "100100", first.BranchCode)
	require.Equal(t, "DIRECT", first.BusinessSource)
	require.Equal(t, "FIRE", first.BusinessName)
	require.Equal(t, "POL-1", first.PolicyNumber)
	require.Equal(t, "PNC-1", first.ClaimNumber)
	require.Equal(t, &at, first.RegisterDate)
	require.Equal(t, &at, first.LossDate)
	require.Equal(t, &at, first.LastProgressAt)
	require.Equal(t, "REKOM", first.RemarkRecommendation)
	require.Equal(t, money.FromMinorUnits(150000), first.EstimationValue)
	require.Equal(t, "S1", first.ProgressStatus1)
	require.Equal(t, "S2", first.ProgressStatus2)
	require.Equal(t, "PIC", first.TechnicalPIC)
	require.Equal(t, "NOTE", first.ProgressNote)
	require.Equal(t, "ADJ", first.AdjusterName)
	require.Equal(t, "COL", first.CauseOfLoss)
	require.Equal(t, "KRONO", first.Chronology)
	require.True(t, first.ProgressStalled)

	// Baris kedua penuh NULL: tanggal nil, nilai uang nol, tidak mandek.
	second := page.Items[1]
	require.Nil(t, second.RegisterDate)
	require.Nil(t, second.LastProgressAt)
	require.Equal(t, money.Zero, second.EstimationValue)
	require.False(t, second.ProgressStalled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListEmptyReturnsEmptySlice(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("list")).WithArgs("100100", nil, "", "", 0, 25).
		WillReturnRows(sqlmock.NewRows(listColumns))

	page, err := repo.List(context.Background(), branchQuery, inboxosclaimpercabang.Pagination{})
	require.NoError(t, err)
	require.NotNil(t, page.Items)
	require.Empty(t, page.Items)
	require.Equal(t, 0, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsQueryError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("list")).WillReturnError(errBoom)

	_, err := repo.List(context.Background(), branchQuery, inboxosclaimpercabang.Pagination{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menjalankan kueri list")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRejectsAnInvalidMoneyValue(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	values := listRowValues(at)
	values[10] = "12.5"
	mock.ExpectQuery(exactQuery("list")).
		WillReturnRows(sqlmock.NewRows(listColumns).AddRow(append(values, int64(1))...))

	_, err := repo.List(context.Background(), branchQuery, inboxosclaimpercabang.Pagination{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris kueri list")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsScanError(t *testing.T) {
	repo, mock := newMockRepo(t)
	// Jumlah kolom salah membuat Scan gagal.
	mock.ExpectQuery(exactQuery("list")).
		WillReturnRows(sqlmock.NewRows([]string{"ONLY"}).AddRow("x"))

	_, err := repo.List(context.Background(), branchQuery, inboxosclaimpercabang.Pagination{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris kueri list")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsRowsError(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(listColumns).AddRow(append(listRowValues(at), int64(1))...).
		RowError(0, errBoom)
	mock.ExpectQuery(exactQuery("list")).WillReturnRows(rows)

	_, err := repo.List(context.Background(), branchQuery, inboxosclaimpercabang.Pagination{})
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menelusuri hasil kueri list")
	require.NoError(t, mock.ExpectationsWereMet())
}

// exportColumns adalah susunan kolom lengkap kueri list_export.
func exportColumns() []string {
	cols := append([]string{}, listColumns[:len(listColumns)-1]...)
	cols = append(cols, exportOnlyColumns...)
	return append(cols, "TOTAL_ROWS")
}

// exportRowValues membentuk satu baris ekspor; pembagian treaty ke-i bernilai i+1.
func exportRowValues(at time.Time) []driver.Value {
	values := listRowValues(at)
	values = append(values, "CLAIMKEY-1", "BISNIS POLIS", int64(300), []byte("200"), "100")
	for i := 0; i < 24; i++ {
		values = append(values, int64(i+1))
	}
	return append(values, int64(7))
}

func TestListForExportMapsMoneyAndTreatyShares(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(exactQuery("list_export")).WithArgs("100100", 100, 100).
		WillReturnRows(sqlmock.NewRows(exportColumns()).AddRow(exportRowValues(at)...))

	page, err := repo.ListForExport(context.Background(), branchQuery,
		inboxosclaimpercabang.Pagination{Page: 2, Size: 500})
	require.NoError(t, err)
	require.Equal(t, 7, page.Total)
	require.Equal(t, inboxosclaimpercabang.Pagination{Page: 2, Size: 100}, page.Pagination)
	require.Len(t, page.Items, 1)

	row := page.Items[0]
	require.Equal(t, "PNC-1", row.ClaimNumber)
	require.Equal(t, "JAKARTA", row.BranchName)
	require.Equal(t, &at, row.RegisterDate)
	require.Equal(t, money.FromMinorUnits(150000), row.EstimationValue)
	require.True(t, row.ProgressStalled)
	require.Equal(t, "CLAIMKEY-1", row.ClaimKey)
	require.Equal(t, "BISNIS POLIS", row.PolicyBusinessName)
	require.Equal(t, money.FromMinorUnits(300), row.ReserveClaimFull)
	require.Equal(t, money.FromMinorUnits(200), row.ReserveClaimASM)
	require.Equal(t, money.FromMinorUnits(100), row.Coinsurance)

	// Urutan ke-24 pembagian treaty harus mengikuti urutan alias SHARE_*.
	for i, value := range row.TreatyShares.TreatyValues() {
		require.Equalf(t, money.FromMinorUnits(int64(i+1)), value,
			"pembagian treaty %s tertukar", inboxosclaimpercabang.ExportTreatyColumns[i])
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListForExportRejectsAnInvalidReserve(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	values := exportRowValues(at)
	values[22] = 1.5 // RESERVE_CLAIM_FULL berdesimal
	mock.ExpectQuery(exactQuery("list_export")).
		WillReturnRows(sqlmock.NewRows(exportColumns()).AddRow(values...))

	_, err := repo.ListForExport(context.Background(), branchQuery,
		inboxosclaimpercabang.Pagination{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "kolom RESERVE_CLAIM_FULL")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListForExportRejectsAnInvalidTreatyShare(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	values := exportRowValues(at)
	values[25] = "bukan angka" // pembagian treaty pertama
	mock.ExpectQuery(exactQuery("list_export")).
		WillReturnRows(sqlmock.NewRows(exportColumns()).AddRow(values...))

	_, err := repo.ListForExport(context.Background(), branchQuery,
		inboxosclaimpercabang.Pagination{})
	require.Error(t, err)
	require.Contains(t, err.Error(),
		"kolom treaty "+inboxosclaimpercabang.ExportTreatyColumns[0])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListForExportWrapsQueryScanAndRowsErrors(t *testing.T) {
	ctx := context.Background()
	empty := inboxosclaimpercabang.Pagination{}

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("list_export")).WillReturnError(errBoom)
	_, err := repo.ListForExport(ctx, branchQuery, empty)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menjalankan kueri list_export")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("list_export")).
		WillReturnRows(sqlmock.NewRows([]string{"ONLY"}).AddRow("x"))
	_, err = repo.ListForExport(ctx, branchQuery, empty)
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris kueri list_export")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	at := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(exactQuery("list_export")).WillReturnRows(
		sqlmock.NewRows(exportColumns()).AddRow(exportRowValues(at)...).RowError(0, errBoom))
	_, err = repo.ListForExport(ctx, branchQuery, empty)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menelusuri hasil kueri list_export")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDominantFactorsGroupsByClaimAndSkipsBlankKeys(t *testing.T) {
	repo, mock := newMockRepo(t)
	rows := sqlmock.NewRows([]string{"CLAIMID", "NAME"}).
		AddRow("K1", "Banjir").
		AddRow("K1", "Api").
		AddRow(nil, "Abaikan").
		AddRow("K2", "Pencurian")
	mock.ExpectQuery(exactQuery("dominant_factors")).WithArgs("100100").WillReturnRows(rows)

	factors, err := repo.DominantFactors(context.Background(), branchQuery)
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		"K1": {"Banjir", "Api"},
		"K2": {"Pencurian"},
	}, factors)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDominantFactorsWrapsErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("dominant_factors")).WillReturnError(errBoom)
	_, err := repo.DominantFactors(ctx, branchQuery)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menjalankan kueri dominant_factors")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("dominant_factors")).
		WillReturnRows(sqlmock.NewRows([]string{"ONLY"}).AddRow("x"))
	_, err = repo.DominantFactors(ctx, branchQuery)
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris kueri dominant_factors")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("dominant_factors")).WillReturnRows(
		sqlmock.NewRows([]string{"CLAIMID", "NAME"}).AddRow("K1", "A").RowError(0, errBoom))
	_, err = repo.DominantFactors(ctx, branchQuery)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menelusuri hasil kueri dominant_factors")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTablesSucceedAndWrapErrors(t *testing.T) {
	ctx := context.Background()
	checks := []struct {
		name    string
		run     func(*Repo, context.Context) error
		message string
	}{
		{"check_tables", (*Repo).CheckTable, "POOLDATA.T_CLAIM_PNC"},
		{"check_export_tables", (*Repo).CheckExportTable, "treaty_loss@asmd"},
		{"check_detail_tables", (*Repo).CheckDetailTable, "POOLDATA.M_KOMUNIKASI_PNC"},
	}

	for _, check := range checks {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exactQuery(check.name)).
			WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
		require.NoError(t, check.run(repo, ctx), check.name)
		require.NoError(t, mock.ExpectationsWereMet())

		repo, mock = newMockRepo(t)
		mock.ExpectQuery(exactQuery(check.name)).WillReturnError(errBoom)
		err := check.run(repo, ctx)
		require.ErrorIs(t, err, errBoom, check.name)
		require.Contains(t, err.Error(), check.message)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestAnyBranchWithClaims(t *testing.T) {
	ctx := context.Background()
	cols := []string{"BRANCHCODE", "OLDID"}

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("any_branch_with_claims")).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(" 100100 ", " 123 "))
	code, detail, found, err := repo.AnyBranchWithClaims(ctx)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "100100", code)
	require.Equal(t, "123", detail)
	require.NoError(t, mock.ExpectationsWereMet())

	// Tidak ada baris: bukan galat.
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("any_branch_with_claims")).WillReturnRows(sqlmock.NewRows(cols))
	_, _, found, err = repo.AnyBranchWithClaims(ctx)
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())

	// Kode NULL diperlakukan sama dengan tidak ada.
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("any_branch_with_claims")).
		WillReturnRows(sqlmock.NewRows(cols).AddRow(nil, "123"))
	_, _, found, err = repo.AnyBranchWithClaims(ctx)
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("any_branch_with_claims")).WillReturnError(errBoom)
	_, _, found, err = repo.AnyBranchWithClaims(ctx)
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "membaca kode cabang contoh")
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

// stringer adalah tipe angka tiruan milik driver yang berperilaku seperti teks.
type stringer struct{ text string }

func (s stringer) String() string { return s.text }

func TestMinorUnitsAcceptsEveryDriverShape(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  money.Money
	}{
		{"nil", nil, money.Zero},
		{"int64", int64(1250), money.FromMinorUnits(1250)},
		{"int", 75, money.FromMinorUnits(75)},
		{"float64 bulat", float64(-400), money.FromMinorUnits(-400)},
		{"bytes", []byte(" 99 "), money.FromMinorUnits(99)},
		{"string", "10", money.FromMinorUnits(10)},
		{"stringer", stringer{"33"}, money.FromMinorUnits(33)},
	}
	for _, tc := range cases {
		got, err := minorUnits(tc.value)
		require.NoError(t, err, tc.name)
		require.Equal(t, tc.want, got, tc.name)
	}
}

func TestMinorUnitsRejectsImpreciseOrUnknownValues(t *testing.T) {
	cases := []struct {
		name    string
		value   any
		message string
	}{
		{"NaN", math.NaN(), "bukan bilangan"},
		{"Inf", math.Inf(1), "bukan bilangan"},
		{"desimal", 1.25, "berdesimal"},
		{"terlalu besar", float64(1 << 53), "melampaui rentang"},
		{"terlalu kecil", -float64(1 << 53), "melampaui rentang"},
		{"teks desimal", "1.5", "bukan bilangan bulat satuan terkecil"},
		{"tipe asing", true, "tidak dikenali"},
	}
	for _, tc := range cases {
		got, err := minorUnits(tc.value)
		require.Error(t, err, tc.name)
		require.Contains(t, err.Error(), tc.message, tc.name)
		require.Equal(t, money.Zero, got, tc.name)
	}
}

func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxosclaimpercabang/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { query("tidak_ada") })
}

func TestSplitByNameDropsCommentsAndEmptyBodies(t *testing.T) {
	content := "-- kepala berkas\n" +
		"-- name: kosong\n" +
		"-- hanya komentar\n" +
		"-- name: satu\n" +
		"SELECT 1\n" +
		"-- penjelas\n" +
		"FROM x\n"

	require.Equal(t, map[string]string{"satu": "SELECT 1\nFROM x"}, splitByName(content))
}

// headerColumns adalah kolom kueri detail_header.
var headerColumns = []string{
	"CLAIM_NUMBER", "CLAIM_KEY", "BUSINESS_NAME", "OCCUPATION", "TOTAL_SUM_INSURED",
	"CHRONOLOGY", "ESTIMATION_VALUE", "REGISTER_DATE", "REMARK", "NOTE",
}

// expectHeader menyiapkan kueri kepala yang berhasil untuk klaim PNC-1.
func expectHeader(mock sqlmock.Sqlmock, at time.Time) {
	mock.ExpectQuery(exactQuery("detail_header")).WithArgs("100100", "PNC-1").
		WillReturnRows(sqlmock.NewRows(headerColumns).AddRow(
			"PNC-1", "K1", "FIRE", "KANTOR", int64(1000000),
			"KRONO", int64(5000), at, "REKOM", "CATATAN"))
}

func TestFindDetailReadsHeaderThenEveryChild(t *testing.T) {
	repo, mock := newMockRepo(t)
	at := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)

	expectHeader(mock, at)
	mock.ExpectQuery(exactQuery("detail_objects")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "A", "B", "C", "D", "E", "F"}).
			AddRow(" OBJ-1 ", "Objek", "Lokasi", "Kerja", at, "KTP", "AKTIF"))
	// Coverage SELURUH klaim dibaca sekali, lalu dikelompokkan menurut objeknya. Baris
	// bertuan objek lain sengaja ikut dikirim: pengelompokannya harus membuangnya, bukan
	// menempelkannya ke objek pertama yang ada.
	mock.ExpectQuery(exactQuery("detail_object_coverages")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"OBJ", "ID", "NAME", "CUR", "TSI"}).
			AddRow("OBJ-1", " CV-1 ", " FLEXAS ", " IDR ", int64(250000)).
			AddRow("OBJ-9", "CV-9", "GEMPA", "USD", int64(100)))
	// Object Item dan Estimasi dibaca dengan kueri tersendiri, lalu dirangkai dari dalam ke
	// luar. Baris bertuan coverage lain sengaja ikut dikirim pada keduanya: pengelompokannya
	// memakai kunci GABUNGAN, dan kunci tunggal akan menempelkannya ke induk yang salah —
	// `OBJECTITEMID` bernilai `1` pada hampir seluruh baris nyata.
	mock.ExpectQuery(exactQuery("detail_object_items")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"OBJ", "CVG", "ITEM", "NAMA", "DESK"}).
			AddRow("OBJ-1", "CV-1", " 1 ", " BUILDINGS ", " Gudang utama ").
			AddRow("OBJ-9", "CV-9", "1", "LAIN", "Milik coverage lain"))
	mock.ExpectQuery(exactQuery("detail_estimations")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{
			"OBJ", "CVG", "ITEM", "SEQ", "TGL", "TIPE", "CUR", "KURS", "NILAI"}).
			AddRow("OBJ-1", "CV-1", "1", " 1 ", at, " Claim ", " IDR ", int64(100), int64(10000)).
			AddRow("OBJ-1", "CV-1", "1", "2", at, "Claim", "IDR", int64(100), int64(-4000)).
			AddRow("OBJ-9", "CV-9", "1", "1", at, "Claim", "USD", int64(100), int64(500)))
	mock.ExpectQuery(exactQuery("detail_coverage_spreading")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"OBJ", "CVG", "TREATY", "SHARE"}).
			AddRow("OBJ-1", "CV-1", " FAC-OUT ", int64(1000000)))
	mock.ExpectQuery(exactQuery("detail_coverage_comember")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"NAMA", "SHARE"}).
			AddRow(" ASURANSI CONTOH ", int64(510000)))
	// Riwayat progres dikunci nomor klaim, bukan CLAIMID.
	mock.ExpectQuery(exactQuery("detail_progress")).WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D", "E", "F", "G", "H"}).
			AddRow(at, "PNC-1", "S1", "S2", "USER", nil, "OPEN", "N"))
	mock.ExpectQuery(exactQuery("detail_messages")).WithArgs("K1", "K1").
		WillReturnRows(sqlmock.NewRows([]string{"A", "B", "C", "D", "E", "F"}).
			AddRow("Pengirim", at, "Pesan", nil, "Balas", int64(1)))
	mock.ExpectQuery(exactQuery("detail_dominant_factors")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"NAME"}).
			AddRow("Banjir").AddRow("  ").AddRow("Api"))

	detail, found, err := repo.FindDetail(context.Background(), branchQuery, " PNC-1 ")
	require.NoError(t, err)
	require.True(t, found)

	require.Equal(t, "PNC-1", detail.ClaimNumber)
	require.Equal(t, "K1", detail.ClaimKey)
	require.Equal(t, "FIRE", detail.BusinessName)
	require.Equal(t, "KANTOR", detail.Occupation)
	require.Equal(t, money.FromMinorUnits(1000000), detail.TotalSumInsured)
	require.Equal(t, "KRONO", detail.Chronology)
	require.Equal(t, money.FromMinorUnits(5000), detail.EstimationValue)
	require.Equal(t, &at, detail.RegisterDate)
	require.Equal(t, "REKOM", detail.RemarkRecommendation)
	require.Equal(t, "CATATAN", detail.ProgressNote)

	require.Equal(t, []inboxosclaimpercabang.DetailObject{{
		ID:   "OBJ-1",
		Name: "Objek", Location: "Lokasi", Job: "Kerja", DateOfBirth: &at,
		IDCard: "KTP", ParticipantStatus: "AKTIF",
		Coverages: []inboxosclaimpercabang.DetailCoverage{{
			ObjectID: "OBJ-1", ID: "CV-1", Name: "FLEXAS", Currency: "IDR",
			SumTSI: money.FromMinorUnits(250000),
			Items: []inboxosclaimpercabang.DetailItem{{
				ObjectID: "OBJ-1", CoverageID: "CV-1", ID: "1",
				Name: "BUILDINGS", Description: "Gudang utama",
				Estimations: []inboxosclaimpercabang.DetailEstimation{
					{
						ObjectID: "OBJ-1", CoverageID: "CV-1", ItemID: "1",
						Sequence: "1", RecordedAt: &at, Type: "Claim", Currency: "IDR",
						Rate:  money.FromMinorUnits(100),
						Value: money.FromMinorUnits(10000),
					},
					{
						// Nilai NEGATIF dibawa apa adanya — koreksi yang saling
						// meniadakan memang terjadi di data nyata.
						ObjectID: "OBJ-1", CoverageID: "CV-1", ItemID: "1",
						Sequence: "2", RecordedAt: &at, Type: "Claim", Currency: "IDR",
						Rate:  money.FromMinorUnits(100),
						Value: money.FromMinorUnits(-4000),
					},
				},
			}},
			// Jumlah estimasi coverage ini 10000 + (-4000) = 6000 satuan terkecil.
			// Share 100% menghasilkan nilai yang sama; share 51% menghasilkan 3060.
			Spreadings: []inboxosclaimpercabang.DetailSpreading{{
				ObjectID: "OBJ-1", CoverageID: "CV-1",
				TreatyName: "FAC-OUT", Currency: "IDR",
				EstimationValue:    money.FromMinorUnits(6000),
				SharePercentScaled: 1000000,
				ResultValue:        money.FromMinorUnits(6000),
			}},
			CoMembers: []inboxosclaimpercabang.DetailCoMember{{
				ObjectID: "OBJ-1", CoverageID: "CV-1",
				InsurerName: "ASURANSI CONTOH", Currency: "IDR",
				EstimationValue:    money.FromMinorUnits(6000),
				SharePercentScaled: 510000,
				ResultValue:        money.FromMinorUnits(3060),
			}},
		}},
	}}, detail.Objects,
		"baris anak milik coverage lain tidak boleh ikut menempel")
	require.Equal(t, []inboxosclaimpercabang.DetailProgress{{
		RecordedAt: &at, ClaimNumber: "PNC-1", Status1: "S1", Status2: "S2",
		EnteredBy: "USER", Status: "OPEN", Note: "N",
	}}, detail.ProgressHistory)
	require.Equal(t, []inboxosclaimpercabang.DetailMessage{{
		SenderName: "Pengirim", SentAt: &at, Message: "Pesan", Reply: "Balas", Internal: true,
	}}, detail.AdjusterMessages)
	require.Equal(t, "Banjir, Api", detail.DominantFactors)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDetailEmptyNumberSkipsTheDatabase(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, found, err := repo.FindDetail(context.Background(), branchQuery, "  ")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDetailUnknownClaimRunsNoChildQuery(t *testing.T) {
	// Klaim milik cabang lain berhenti di kepala; tidak ada kueri anak yang dijalankan.
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("detail_header")).WithArgs("100100", "PNC-9").
		WillReturnRows(sqlmock.NewRows(headerColumns))

	_, found, err := repo.FindDetail(context.Background(), branchQuery, "PNC-9")
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindDetailHeaderErrors(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("detail_header")).WillReturnError(errBoom)
	_, found, err := repo.FindDetail(ctx, branchQuery, "PNC-1")
	require.ErrorIs(t, err, errBoom)
	require.Contains(t, err.Error(), "menjalankan kueri detail_header")
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("detail_header")).WillReturnRows(
		sqlmock.NewRows(headerColumns).AddRow(
			"PNC-1", "K1", "", "", 1.5, "", int64(1), at, "", ""))
	_, _, err = repo.FindDetail(ctx, branchQuery, "PNC-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca TOTAL_SUM_INSURED")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exactQuery("detail_header")).WillReturnRows(
		sqlmock.NewRows(headerColumns).AddRow(
			"PNC-1", "K1", "", "", int64(1), "", "x", at, "", ""))
	_, _, err = repo.FindDetail(ctx, branchQuery, "PNC-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca ESTIMATION_VALUE")
	require.NoError(t, mock.ExpectationsWereMet())
}

// childStep menyiapkan kueri anak ke-n dengan hasil yang ditentukan.
type childStep struct {
	name string
	args []driver.Value
	cols []string
	row  []driver.Value
}

// detailChildren adalah kelima kueri anak berurutan, masing-masing dengan baris sah.
func detailChildren(at time.Time) []childStep {
	return []childStep{
		{"detail_objects", []driver.Value{"K1"}, []string{"ID", "A", "B", "C", "D", "E", "F"},
			[]driver.Value{"OBJ-1", "O", "L", "J", at, "K", "S"}},
		{"detail_object_coverages", []driver.Value{"K1"},
			[]string{"OBJ", "ID", "NAME", "CUR", "TSI"},
			[]driver.Value{"OBJ-1", "CV-1", "FLEXAS", "IDR", int64(100)}},
		{"detail_object_items", []driver.Value{"K1"},
			[]string{"OBJ", "CVG", "ITEM", "NAMA", "DESK"},
			[]driver.Value{"OBJ-1", "CV-1", "1", "BUILDINGS", "Gudang"}},
		{"detail_estimations", []driver.Value{"K1"},
			[]string{"OBJ", "CVG", "ITEM", "SEQ", "TGL", "TIPE", "CUR", "KURS", "NILAI"},
			[]driver.Value{"OBJ-1", "CV-1", "1", "1", at, "Claim", "IDR", int64(100), int64(100)}},
		{"detail_coverage_spreading", []driver.Value{"K1"},
			[]string{"OBJ", "CVG", "TREATY", "SHARE"},
			[]driver.Value{"OBJ-1", "CV-1", "ORS", int64(1000000)}},
		{"detail_coverage_comember", []driver.Value{"K1"},
			[]string{"NAMA", "SHARE"},
			[]driver.Value{"ASURANSI CONTOH", int64(1000000)}},
		{"detail_progress", []driver.Value{"PNC-1"}, []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			[]driver.Value{at, "PNC-1", "", "", "", at, "", ""}},
		{"detail_messages", []driver.Value{"K1", "K1"}, []string{"A", "B", "C", "D", "E", "F"},
			[]driver.Value{"S", at, "M", at, "R", int64(0)}},
		{"detail_dominant_factors", []driver.Value{"K1"}, []string{"NAME"}, []driver.Value{"A"}},
	}
}

func TestFindDetailWrapsEveryChildError(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	children := detailChildren(at)

	// Setiap kueri anak diuji gagal di tiga tempat: Query, Scan, dan Rows.Err.
	for failing := range children {
		for _, mode := range []string{"query", "scan", "rows"} {
			repo, mock := newMockRepo(t)
			expectHeader(mock, at)
			for i := 0; i < failing; i++ {
				c := children[i]
				mock.ExpectQuery(exactQuery(c.name)).WithArgs(c.args...).
					WillReturnRows(sqlmock.NewRows(c.cols).AddRow(c.row...))
			}

			c := children[failing]
			expect := mock.ExpectQuery(exactQuery(c.name)).WithArgs(c.args...)
			var want string
			switch mode {
			case "query":
				expect.WillReturnError(errBoom)
				want = "menjalankan kueri " + c.name
			case "scan":
				expect.WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("x", "y"))
				want = "membaca baris kueri " + c.name
			case "rows":
				expect.WillReturnRows(
					sqlmock.NewRows(c.cols).AddRow(c.row...).RowError(0, errBoom))
				want = "menelusuri hasil kueri " + c.name
			}

			_, found, err := repo.FindDetail(ctx, branchQuery, "PNC-1")
			require.Errorf(t, err, "%s/%s", c.name, mode)
			require.Containsf(t, err.Error(), want, "%s/%s", c.name, mode)
			require.False(t, found)
			require.NoError(t, mock.ExpectationsWereMet())
		}
	}
}

func TestDetailFactorsWithoutNamesIsDash(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exactQuery("detail_dominant_factors")).WithArgs("K1").
		WillReturnRows(sqlmock.NewRows([]string{"NAME"}).AddRow(nil))

	got, err := repo.detailFactors(context.Background(), "K1")
	require.NoError(t, err)
	require.Equal(t, "-", got)
	require.NoError(t, mock.ExpectationsWereMet())
}
