package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
)

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

var documentColumns = []string{"ID", "NAME", "URL", "FOLDER", "EXP", "TYPE", "CLAIM", "UPLOADED"}

var at = time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)

func TestFolderLookup(t *testing.T) {
	repo, mock := newMock(t)
	ctx := context.Background()

	mock.ExpectQuery(exact("folder_aplikasi")).WithArgs("KLAIMPNC").
		WillReturnRows(sqlmock.NewRows([]string{"NAMA"}).AddRow(" klaimpnc "))
	folder, err := repo.NamaFolderAplikasi(ctx, " klaimpnc ")
	require.NoError(t, err)
	require.Equal(t, "klaimpnc", folder)

	mock.ExpectQuery(exact("folder_aplikasi")).WillReturnRows(sqlmock.NewRows([]string{"NAMA"}))
	_, err = repo.NamaFolderAplikasi(ctx, "KLAIMPNC")
	require.ErrorIs(t, err, dokumenpenunjang.ErrFolderAplikasiTidakAda)

	mock.ExpectQuery(exact("folder_aplikasi")).
		WillReturnRows(sqlmock.NewRows([]string{"NAMA"}).AddRow(nil))
	_, err = repo.NamaFolderAplikasi(ctx, "KLAIMPNC")
	require.ErrorIs(t, err, dokumenpenunjang.ErrFolderAplikasiTidakAda)
	require.ErrorContains(t, err, "NAMA_FOLDER kosong")

	mock.ExpectQuery(exact("folder_aplikasi")).WillReturnError(errors.New("ORA-12541: no listener"))
	_, err = repo.NamaFolderAplikasi(ctx, "KLAIMPNC")
	require.ErrorIs(t, err, dokumenpenunjang.ErrLinkTakTerjangkau)

	mock.ExpectQuery(exact("folder_aplikasi")).WillReturnError(errDB)
	_, err = repo.NamaFolderAplikasi(ctx, "KLAIMPNC")
	require.ErrorIs(t, err, errDB)
	require.NotErrorIs(t, err, dokumenpenunjang.ErrLinkTakTerjangkau)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUploadAccessIsRecordedWithAnUppercaseToken(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectExec(exact("catat_akses_unggah")).
		WithArgs("KLAIMPNC", sqlmock.AnyArg(), "BUDI").
		WillReturnResult(sqlmock.NewResult(0, 1))
	token, err := repo.CatatAksesUnggah(context.Background(), " KLAIMPNC ", " BUDI ")
	require.NoError(t, err)
	require.Regexp(t, `^[0-9A-F]{32}$`, token)

	mock.ExpectExec(exact("catat_akses_unggah")).WillReturnError(errDB)
	_, err = repo.CatatAksesUnggah(context.Background(), "KLAIMPNC", "BUDI")
	require.ErrorContains(t, err, "mencatat izin unggah")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTokenIsDeterministicForAnInstant(t *testing.T) {
	require.Equal(t, tokenAkses(at), tokenAkses(at))
	require.NotEqual(t, tokenAkses(at), tokenAkses(at.Add(time.Nanosecond)))
}

func TestSaveBindsNullsForEmptyValues(t *testing.T) {
	repo, mock := newMock(t)
	expires := at.Add(24 * time.Hour)

	mock.ExpectExec(exact("simpan_metadata")).WithArgs(
		"IMG-1", "http://u", "Doc/2026/09/", expires, "a.pdf", "klaimpnc",
		dokumenpenunjang.JenisPenyimpanan, "KTP", "PNC-1", at,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Simpan(context.Background(), dokumenpenunjang.Document{
		ImageID: " IMG-1 ", URL: "http://u", Folder: "Doc/2026/09/", ExpiresAt: &expires,
		FileName: "a.pdf", DocumentType: "KTP", ClaimNumber: "PNC-1", UploadedAt: &at,
	}))

	mock.ExpectExec(exact("simpan_metadata")).WithArgs(
		"IMG-2", nil, nil, nil, nil, "klaimpnc", dokumenpenunjang.JenisPenyimpanan, nil,
		dokumenpenunjang.TanpaKlaim, nil,
	).WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Simpan(context.Background(), dokumenpenunjang.Document{ImageID: "IMG-2"}))

	mock.ExpectExec(exact("simpan_metadata")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Simpan(context.Background(), dokumenpenunjang.Document{}),
		"mencatat metadata dokumen")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPerClaimMapsRows(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("dokumen_per_klaim")).WithArgs("PNC-1", "KLAIMPNC").
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow(" IMG-1 ", " a.pdf ", " http://u ", " Doc/ ", at, " KTP ", " PNC-1 ", at).
			AddRow("IMG-2", nil, nil, nil, nil, nil, nil, nil))
	rows, err := repo.PerKlaim(context.Background(), " pnc-1 ")
	require.NoError(t, err)
	require.Equal(t, []dokumenpenunjang.Document{
		{ImageID: "IMG-1", FileName: "a.pdf", URL: "http://u", Folder: "Doc/",
			ExpiresAt: &at, DocumentType: "KTP", ClaimNumber: "PNC-1", UploadedAt: &at},
		{ImageID: "IMG-2"},
	}, rows)

	mock.ExpectQuery(exact("dokumen_per_klaim")).WillReturnError(errors.New("ORA-03113: end-of-file"))
	_, err = repo.PerKlaim(context.Background(), "PNC-1")
	require.ErrorIs(t, err, dokumenpenunjang.ErrLinkTakTerjangkau)

	mock.ExpectQuery(exact("dokumen_per_klaim")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.PerKlaim(context.Background(), "PNC-1")
	require.ErrorContains(t, err, "memindai dokumen")

	mock.ExpectQuery(exact("dokumen_per_klaim")).
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow("I", nil, nil, nil, nil, nil, nil, nil).RowError(0, errDB))
	_, err = repo.PerKlaim(context.Background(), "PNC-1")
	require.ErrorContains(t, err, "membaca baris dokumen klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOneDocument(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("dokumen_menurut_imageid")).WithArgs("IMG-1").
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow("IMG-1", "a.pdf", nil, nil, nil, nil, "PNC-1", nil))
	got, err := repo.Ambil(context.Background(), " IMG-1 ")
	require.NoError(t, err)
	require.Equal(t, dokumenpenunjang.Document{ImageID: "IMG-1", FileName: "a.pdf", ClaimNumber: "PNC-1"}, got)

	mock.ExpectQuery(exact("dokumen_menurut_imageid")).WillReturnRows(sqlmock.NewRows(documentColumns))
	_, err = repo.Ambil(context.Background(), "IMG-1")
	require.ErrorIs(t, err, dokumenpenunjang.ErrTidakDitemukan)

	mock.ExpectQuery(exact("dokumen_menurut_imageid")).WillReturnError(errDB)
	_, err = repo.Ambil(context.Background(), "IMG-1")
	require.ErrorContains(t, err, "membaca dokumen")

	mock.ExpectQuery(exact("dokumen_menurut_imageid")).
		WillReturnRows(sqlmock.NewRows(documentColumns).
			AddRow("I", nil, nil, nil, nil, nil, nil, nil).RowError(0, errDB))
	_, err = repo.Ambil(context.Background(), "IMG-1")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryPanicsOnAnUnknownName(t *testing.T) {
	require.Panics(t, func() { query("tidak_ada") })
}
