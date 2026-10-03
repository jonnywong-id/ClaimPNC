package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastermasking"
)

var maskingColumns = []string{
	"ID_MST", "BRANCHID", "BRANCHNAME", "LOGIN", "MODUL", "SUB_MODUL",
	"MAX_SEARCH", "MAX_VIEW", "STS_KTP", "STS_EMAIL", "STS_NOTELP",
	"STS_AKTF", "INPUT_BY", "INPUT_DATE",
}

var stamp = time.Date(2026, time.September, 20, 3, 0, 0, 0, time.UTC)

func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func q(name string) string { return regexp.QuoteMeta(getQuery(name)) }

// fullRow adalah satu baris berisi lengkap dengan nilai bertepi spasi.
func fullRow(id string) *sqlmock.Rows {
	return sqlmock.NewRows(maskingColumns).AddRow(
		" "+id+" ", " 01 ", " JAKARTA ", " ANDI ", " KLAIM ", " PA ",
		10, 20, "ya", "Tidak", "Ya", "aktif", " ADMIN ", stamp,
	)
}

func expectedFull(id string) mastermasking.Masking {
	return mastermasking.Masking{
		ID: id, BranchID: "01", BranchName: "JAKARTA", Login: "ANDI", Module: "KLAIM",
		SubModule: "PA", SearchQuota: 10, ViewQuota: 20, ViewIDCard: true, ViewEmail: false,
		ViewPhone: true, Active: true, InputBy: "ADMIN", InputAt: stamp,
	}
}

func sampleMasking() mastermasking.Masking {
	return mastermasking.Masking{
		BranchID: " 01 ", Login: " ANDI ", Module: "KLAIM", SubModule: "PA",
		SearchQuota: 10, ViewQuota: 20, ViewIDCard: true, ViewEmail: false, ViewPhone: true,
		Active: true, InputBy: "ADMIN", InputAt: stamp,
	}
}

// Daftar mengikat jenis pencarian, pola LIKE dua kali, dan status yang dicari.
func TestListBindsPatternAndStatus(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(q("masking_list")).
		WithArgs("status", "%AKTIF%", "%AKTIF%", "AKTIF").
		WillReturnRows(fullRow("7").AddRow("8", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	list, err := repo.List(context.Background(), mastermasking.Filter{By: " status ", Keyword: " aktif "})
	require.NoError(t, err)
	require.Equal(t, []mastermasking.Masking{expectedFull("7"), {ID: "8"}}, list,
		"kolom NULL tidak menggagalkan daftar dan tidak pernah membuka kewenangan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Status yang tidak sah dan jenis selain status mengikat teks kosong pada parameter status.
func TestListNonStatusBindsEmptyStatus(t *testing.T) {
	for _, filter := range []mastermasking.Filter{
		{By: mastermasking.SearchByLogin, Keyword: "andi"},
		{By: mastermasking.SearchByStatus, Keyword: "entah"},
	} {
		repo, mock := newMockRepo(t)
		pattern := "%" + map[string]string{"andi": "ANDI", "entah": "ENTAH"}[filter.Keyword] + "%"
		mock.ExpectQuery(q("masking_list")).
			WithArgs(string(filter.By), pattern, pattern, "").
			WillReturnRows(sqlmock.NewRows(maskingColumns))

		list, err := repo.List(context.Background(), filter)
		require.NoError(t, err)
		require.Empty(t, list)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestListErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("masking_list")).WillReturnError(sql.ErrConnDone)
	_, err := repo.List(ctx, mastermasking.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "membaca daftar masking")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_list")).WillReturnRows(sqlmock.NewRows(maskingColumns).
		AddRow("1", "01", "", "A", "M", "", "bukan-angka", 0, "", "", "", "", "", nil))
	_, err = repo.List(ctx, mastermasking.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris masking")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_list")).WillReturnRows(fullRow("1").RowError(0, boom))
	_, err = repo.List(ctx, mastermasking.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menelusuri daftar masking")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGet(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("masking_get")).WithArgs("7").WillReturnRows(fullRow("7"))
	got, err := repo.Get(ctx, "7")
	require.NoError(t, err)
	require.Equal(t, expectedFull("7"), got)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_get")).WithArgs("9").WillReturnRows(sqlmock.NewRows(maskingColumns))
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_get")).WithArgs("9").WillReturnError(sql.ErrConnDone)
	_, err = repo.Get(ctx, "9")
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), `membaca masking "9"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pasangan cabang+login dicari dalam huruf besar tanpa spasi.
func TestFindByPair(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("masking_find_pair")).WithArgs("01", "ANDI").WillReturnRows(fullRow("7"))
	got, err := repo.FindByPair(ctx, " 01 ", " andi ")
	require.NoError(t, err)
	require.Equal(t, "7", got.ID)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_find_pair")).WillReturnRows(sqlmock.NewRows(maskingColumns))
	_, err = repo.FindByPair(ctx, "01", "ANDI")
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("masking_find_pair")).WillReturnError(sql.ErrConnDone)
	_, err = repo.FindByPair(ctx, "01", "ANDI")
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), `mencari masking cabang "01" pengguna "ANDI"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Sisip membaca nomor berikutnya, menulis seluruh kolom (termasuk PASSWORD warisan), lalu
// membaca ulang barisnya.
func TestInsertWritesEveryColumn(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(26))
	mock.ExpectExec(q("masking_insert")).
		WithArgs("26", "01", "ANDI", "KLAIM", "PA", 10, 20, "Ya", "Tidak", "Ya",
			"AKTIF", legacyPassword, "ADMIN", stamp).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("masking_get")).WithArgs("26").WillReturnRows(fullRow("26"))

	saved, err := repo.Insert(context.Background(), sampleMasking())
	require.NoError(t, err)
	require.Equal(t, expectedFull("26"), saved)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Bentrok nomor ID dicoba ulang dengan nomor yang dibaca ulang.
func TestInsertRetriesOnIDTaken(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(5))
	mock.ExpectExec(q("masking_insert")).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		sqlmock.AnyArg(), sqlmock.AnyArg(), "TIDAK AKTIF", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("ORA-00001: unique constraint violated"))
	mock.ExpectRollback()

	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(6))
	mock.ExpectExec(q("masking_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(q("masking_get")).WithArgs("6").WillReturnRows(fullRow("6"))

	m := sampleMasking()
	m.Active = false
	saved, err := repo.Insert(context.Background(), m)
	require.NoError(t, err)
	require.Equal(t, "6", saved.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Setelah tiga bentrok berturut-turut, galatnya menyebut jumlah percobaan.
func TestInsertGivesUpAfterAttempts(t *testing.T) {
	repo, mock := newMockRepo(t)
	for i := 0; i < insertAttempts; i++ {
		mock.ExpectBegin()
		mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(5))
		mock.ExpectExec(q("masking_insert")).WillReturnError(errors.New("ORA-00001"))
		mock.ExpectRollback()
	}

	_, err := repo.Insert(context.Background(), sampleMasking())
	require.ErrorIs(t, err, mastermasking.ErrIDTaken)
	require.Contains(t, err.Error(), "setelah 3 percobaan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertFailures(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectBegin().WillReturnError(sql.ErrConnDone)
	_, err := repo.Insert(ctx, sampleMasking())
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "membuka transaksi")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	_, err = repo.Insert(ctx, sampleMasking())
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Contains(t, err.Error(), "mengambil nomor berikutnya")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(0))
	mock.ExpectRollback()
	_, err = repo.Insert(ctx, sampleMasking())
	require.ErrorIs(t, err, mastermasking.ErrNoSequence)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(5))
	mock.ExpectExec(q("masking_insert")).WillReturnError(errors.New("ORA-12899: value too large"))
	mock.ExpectRollback()
	_, err = repo.Insert(ctx, sampleMasking())
	require.EqualError(t, err, "mastermasking/sqlstore: menyisipkan masking: ORA-12899: value too large")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectBegin()
	mock.ExpectQuery(q("masking_next_id")).WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(5))
	mock.ExpectExec(q("masking_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(sql.ErrTxDone)
	_, err = repo.Insert(ctx, sampleMasking())
	require.ErrorIs(t, err, sql.ErrTxDone)
	require.Contains(t, err.Error(), "menyimpan masking")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Ubah menulis seluruh kolom dengan ID di urutan terakhir, lalu membaca ulang barisnya.
func TestUpdate(t *testing.T) {
	ctx := context.Background()
	m := sampleMasking()
	m.ID = " 7 "

	repo, mock := newMockRepo(t)
	mock.ExpectExec(q("masking_update")).
		WithArgs("01", "ANDI", "KLAIM", "PA", 10, 20, "Ya", "Tidak", "Ya",
			"AKTIF", legacyPassword, "ADMIN", stamp, "7").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("masking_get")).WithArgs("7").WillReturnRows(fullRow("7"))
	saved, err := repo.Update(ctx, m)
	require.NoError(t, err)
	require.Equal(t, expectedFull("7"), saved)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectExec(q("masking_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.Update(ctx, m)
	require.ErrorIs(t, err, mastermasking.ErrNotFound, "tidak ada baris tersentuh berarti tidak ada")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectExec(q("masking_update")).WillReturnError(errors.New("ORA-00001"))
	_, err = repo.Update(ctx, m)
	require.ErrorIs(t, err, mastermasking.ErrIDTaken)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectExec(q("masking_update")).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("driver tidak tahu")))
	_, err = repo.Update(ctx, m)
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca jumlah baris terubah")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSetActive(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectExec(q("masking_set_active")).WithArgs("TIDAK AKTIF", "ADMIN", stamp, "7").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("masking_get")).WithArgs("7").WillReturnRows(fullRow("7"))
	_, err := repo.SetActive(ctx, "7", false, "ADMIN", stamp)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectExec(q("masking_set_active")).WithArgs("AKTIF", "ADMIN", stamp, "7").
		WillReturnResult(sqlmock.NewResult(0, 0))
	_, err = repo.SetActive(ctx, "7", true, "ADMIN", stamp)
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectExec(q("masking_set_active")).WillReturnError(sql.ErrConnDone)
	_, err = repo.SetActive(ctx, "7", true, "ADMIN", stamp)
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "mengubah status masking")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kata kunci kosong dikirim sebagai NULL; batas nol dibetulkan menjadi satu.
func TestListBranches(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("branch_list")).WithArgs(nil, nil, nil, 1).
		WillReturnRows(sqlmock.NewRows([]string{"BRANCHID", "BRANCHNAME"}).
			AddRow(" 01 ", " JAKARTA ").AddRow(nil, nil))
	list, err := repo.ListBranches(ctx, "  ", 0)
	require.NoError(t, err)
	require.Equal(t, []mastermasking.Branch{{ID: "01", Name: "JAKARTA"}, {}}, list)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("branch_list")).WithArgs("%JAK%", "%JAK%", "%JAK%", 20).
		WillReturnRows(sqlmock.NewRows([]string{"BRANCHID", "BRANCHNAME"}))
	list, err = repo.ListBranches(ctx, " jak ", 20)
	require.NoError(t, err)
	require.Empty(t, list)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBranchesErrors(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("branch_list")).WillReturnError(sql.ErrConnDone)
	_, err := repo.ListBranches(ctx, "", 5)
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "membaca daftar cabang")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("branch_list")).
		WillReturnRows(sqlmock.NewRows([]string{"BRANCHID"}).AddRow("01"))
	_, err = repo.ListBranches(ctx, "", 5)
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris cabang")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("branch_list")).
		WillReturnRows(sqlmock.NewRows([]string{"BRANCHID", "BRANCHNAME"}).AddRow("01", "A").RowError(0, boom))
	_, err = repo.ListBranches(ctx, "", 5)
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menelusuri daftar cabang")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBranchExists(t *testing.T) {
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(q("branch_exists")).WithArgs("01").
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	exists, err := repo.BranchExists(ctx, " 01 ")
	require.NoError(t, err)
	require.True(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("branch_exists")).WithArgs("99").
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	exists, err = repo.BranchExists(ctx, "99")
	require.NoError(t, err)
	require.False(t, exists)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(q("branch_exists")).WillReturnError(sql.ErrConnDone)
	_, err = repo.BranchExists(ctx, "99")
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), `memeriksa cabang "99"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Galat nil tetap nil; penerjemah hanya mengubah pelanggaran keunikan menjadi galat domain.
func TestTranslateWriteErrorNil(t *testing.T) {
	require.NoError(t, translateWriteError(nil, "apa saja"))
}
