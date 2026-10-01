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

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

var errDB = errors.New("basis data mati")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah kueri bernama menjadi pola regexp literal.
func q(name string) string { return "^" + regexp.QuoteMeta(getQuery(name)) + "$" }

var (
	issuedAt  = time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	expiresAt = time.Date(2026, 9, 20, 3, 30, 0, 0, time.UTC)
	wib       = time.FixedZone("WIB", 7*3600)
)

var userColumns = []string{
	"IDENTITAS", "JENIS", "NAMA", "LOGIN", "EMAIL", "PERUSAHAAN", "CABANG", "KODE_CABANG",
	"KODE_DETAIL_CABANG", "JABATAN", "OPERATOR_ID", "AKTIF", "DIBUAT", "DIUBAH",
}

func TestLegacyFindActive(t *testing.T) {
	db, mock := newMock(t)
	legacy := NewLegacy(db)
	require.Same(t, db, legacy.DB())

	mock.ExpectQuery(q("local_login_find_active")).WithArgs("admin", "abc").
		WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}).AddRow("admin", "Admin PNC"))
	got, err := legacy.FindActive(context.Background(), "admin", "abc")
	require.NoError(t, err)
	require.Equal(t, provider.LocalLogin{LoginID: "admin", LoginName: "Admin PNC"}, got)

	mock.ExpectQuery(q("local_login_find_active")).WillReturnRows(sqlmock.NewRows([]string{"ID", "NAMA"}))
	_, err = legacy.FindActive(context.Background(), "x", "y")
	require.ErrorIs(t, err, provider.ErrLoginMismatch)

	mock.ExpectQuery(q("local_login_find_active")).WillReturnError(errDB)
	_, err = legacy.FindActive(context.Background(), "x", "y")
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "membaca daftar login")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyServiceAddress(t *testing.T) {
	db, mock := newMock(t)
	legacy := NewLegacy(db)

	mock.ExpectQuery(q("service_address")).WithArgs("PNC", "HCQ").
		WillReturnRows(sqlmock.NewRows([]string{"ALAMAT"}).AddRow("  https://hcq.invalid/api  "))
	got, err := legacy.ServiceAddress(context.Background(), "PNC", "HCQ")
	require.NoError(t, err)
	require.Equal(t, "https://hcq.invalid/api", got)

	mock.ExpectQuery(q("service_address")).WillReturnRows(sqlmock.NewRows([]string{"ALAMAT"}).AddRow("   "))
	_, err = legacy.ServiceAddress(context.Background(), "PNC", "HCQ")
	require.ErrorIs(t, err, provider.ErrServiceNotRegistered)

	mock.ExpectQuery(q("service_address")).WillReturnRows(sqlmock.NewRows([]string{"ALAMAT"}))
	_, err = legacy.ServiceAddress(context.Background(), "PNC", "HCQ")
	require.ErrorIs(t, err, provider.ErrServiceNotRegistered)

	mock.ExpectQuery(q("service_address")).WillReturnError(errDB)
	_, err = legacy.ServiceAddress(context.Background(), "PNC", "HCQ")
	require.ErrorContains(t, err, "membaca alamat layanan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLegacyCheckTable(t *testing.T) {
	db, mock := newMock(t)
	legacy := NewLegacy(db)

	mock.ExpectQuery(q("local_login_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, legacy.CheckTable(context.Background(), "local_login_check_table"))

	mock.ExpectQuery(q("session_check_table")).WillReturnError(errDB)
	require.ErrorIs(t, legacy.CheckTable(context.Background(), "session_check_table"), errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepoSaveConvertsToUTC(t *testing.T) {
	db, mock := newMock(t)
	repo := NewSessionRepo(db)

	mock.ExpectExec(q("session_insert")).WithArgs("s1", "d1", "900", issuedAt, expiresAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Save(context.Background(), auth.Session{
		ID: "s1", TokenDigest: "d1", Identity: "900", IssuedAt: issuedAt.In(wib), ExpiresAt: expiresAt.In(wib),
	}))

	mock.ExpectExec(q("session_insert")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Save(context.Background(), auth.Session{}), "menyimpan sesi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepoGetByTokenDigest(t *testing.T) {
	db, mock := newMock(t)
	repo := NewSessionRepo(db)
	columns := []string{"ID", "DIGEST", "IDENTITAS", "TERBIT", "BERAKHIR", "DICABUT"}

	mock.ExpectQuery(q("session_get_by_fingerprint")).WithArgs("d1").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("s1", "d1", "900", issuedAt.In(wib), expiresAt.In(wib), nil))
	got, err := repo.GetByTokenDigest(context.Background(), "d1")
	require.NoError(t, err)
	require.Equal(t, auth.Session{ID: "s1", TokenDigest: "d1", Identity: "900", IssuedAt: issuedAt, ExpiresAt: expiresAt}, got)
	require.Equal(t, time.UTC, got.IssuedAt.Location())

	mock.ExpectQuery(q("session_get_by_fingerprint")).WithArgs("d2").
		WillReturnRows(sqlmock.NewRows(columns).AddRow("s2", "d2", "900", issuedAt, expiresAt, expiresAt.In(wib)))
	got, err = repo.GetByTokenDigest(context.Background(), "d2")
	require.NoError(t, err)
	require.NotNil(t, got.RevokedAt)
	require.Equal(t, expiresAt, *got.RevokedAt)
	require.Equal(t, time.UTC, got.RevokedAt.Location())

	mock.ExpectQuery(q("session_get_by_fingerprint")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.GetByTokenDigest(context.Background(), "x")
	require.ErrorIs(t, err, auth.ErrSessionNotFound)

	mock.ExpectQuery(q("session_get_by_fingerprint")).WillReturnError(errDB)
	_, err = repo.GetByTokenDigest(context.Background(), "x")
	require.ErrorContains(t, err, "membaca sesi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSessionRepoRevokeAndRenew(t *testing.T) {
	db, mock := newMock(t)
	repo := NewSessionRepo(db)

	mock.ExpectExec(q("session_revoke")).WithArgs(issuedAt, "s1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Revoke(context.Background(), "s1", issuedAt.In(wib)))

	mock.ExpectExec(q("session_revoke")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Revoke(context.Background(), "s1", issuedAt), "mencabut sesi")

	mock.ExpectExec(q("session_extend")).WithArgs(expiresAt, "s1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Renew(context.Background(), "s1", expiresAt.In(wib)))

	mock.ExpectExec(q("session_extend")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Renew(context.Background(), "s1", expiresAt), "memperpanjang sesi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoGetByIdentityMapsNullsAndActiveFlag(t *testing.T) {
	db, mock := newMock(t)
	repo := NewUserRepo(db)

	mock.ExpectQuery(q("user_get_by_identity")).WithArgs("900").WillReturnRows(sqlmock.NewRows(userColumns).AddRow(
		"900", "KARYAWAN", "Contoh", "contoh", "c@example.invalid", "ASM", "Jakarta", "01", "0101", "Staf", "OP1", "Y", issuedAt, expiresAt))
	got, err := repo.GetByIdentity(context.Background(), "900")
	require.NoError(t, err)
	require.Equal(t, auth.User{
		Identity: "900", Kind: auth.Employee, Name: "Contoh", Login: "contoh", Email: "c@example.invalid",
		Company: "ASM", Branch: "Jakarta", BranchCode: "01", DetailBranchCode: "0101", Position: "Staf",
		OperatorID: "OP1", Active: true, CreatedAt: issuedAt, UpdatedAt: expiresAt,
	}, got)

	mock.ExpectQuery(q("user_get_by_identity")).WithArgs("901").WillReturnRows(sqlmock.NewRows(userColumns).AddRow(
		"901", "NON_KARYAWAN", "Mitra", nil, nil, nil, nil, nil, nil, nil, nil, "N", issuedAt, issuedAt))
	got, err = repo.GetByIdentity(context.Background(), "901")
	require.NoError(t, err)
	require.False(t, got.Active)
	require.Equal(t, auth.NonEmployee, got.Kind)
	require.Equal(t, "", got.Login)

	mock.ExpectQuery(q("user_get_by_identity")).WillReturnRows(sqlmock.NewRows(userColumns))
	_, err = repo.GetByIdentity(context.Background(), "x")
	require.ErrorIs(t, err, auth.ErrUserNotFound)

	mock.ExpectQuery(q("user_get_by_identity")).WillReturnError(errDB)
	_, err = repo.GetByIdentity(context.Background(), "x")
	require.ErrorContains(t, err, "membaca pengguna")
	require.NoError(t, mock.ExpectationsWereMet())
}

func sampleUser() auth.User {
	return auth.User{
		Identity: "900", Kind: auth.Employee, Name: "Contoh", Login: "contoh",
		Branch: "Jakarta", Active: true, CreatedAt: issuedAt.In(wib), UpdatedAt: expiresAt.In(wib),
	}
}

// updateArgs adalah argumen user_update untuk sampleUser; isian kosong dikirim NULL.
func updateArgs() []driver.Value {
	return []driver.Value{"KARYAWAN", "Contoh", "contoh", nil, nil, "Jakarta", nil, nil, nil, expiresAt, "900"}
}

func TestUserRepoSaveUpdatesExistingRow(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(q("user_update")).WithArgs(updateArgs()...).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewUserRepo(db).Save(context.Background(), sampleUser()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoSaveInsertsWhenNothingUpdated(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("user_insert")).
		WithArgs("900", "KARYAWAN", "Contoh", "contoh", nil, nil, "Jakarta", nil, nil, nil, nil, "Y", issuedAt, expiresAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewUserRepo(db).Save(context.Background(), sampleUser()))
	require.NoError(t, mock.ExpectationsWereMet())

	// Pengguna nonaktif disisipkan dengan penanda "N".
	db, mock = newMock(t)
	inactive := sampleUser()
	inactive.Active = false
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("user_insert")).
		WithArgs("900", "KARYAWAN", "Contoh", "contoh", nil, nil, "Jakarta", nil, nil, nil, nil, "N", issuedAt, expiresAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewUserRepo(db).Save(context.Background(), inactive))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoSaveRetriesUpdateWhenInsertRaces(t *testing.T) {
	// Sisipan gagal karena baris baru saja dibuat proses lain: pembaruan ulang menyelamatkannya.
	db, mock := newMock(t)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("user_insert")).WillReturnError(errDB)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewUserRepo(db).Save(context.Background(), sampleUser()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepoSaveFailures(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(q("user_update")).WillReturnError(errDB)
	err := NewUserRepo(db).Save(context.Background(), sampleUser())
	require.ErrorContains(t, err, "memperbarui pengguna")
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	err = NewUserRepo(db).Save(context.Background(), sampleUser())
	require.ErrorContains(t, err, "membaca jumlah baris terkena")
	require.NoError(t, mock.ExpectationsWereMet())

	// Sisipan gagal dan pembaruan ulang juga tidak mengenai baris.
	db, mock = newMock(t)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(q("user_insert")).WillReturnError(errDB)
	mock.ExpectExec(q("user_update")).WillReturnResult(sqlmock.NewResult(0, 0))
	err = NewUserRepo(db).Save(context.Background(), sampleUser())
	require.ErrorIs(t, err, errDB)
	require.ErrorContains(t, err, "menyisipkan pengguna")
	require.NoError(t, mock.ExpectationsWereMet())
}
