package sqlstore

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/memory"
)

func TestSearchDiagnosis(t *testing.T) {
	db, mock := be4DB(t)
	d := NewDiagnosisDirectory(db)

	found, err := d.SearchDiagnosis(context.Background(), "  ")
	require.NoError(t, err)
	require.Nil(t, found)

	// Wildcard pengguna di-escape; kode dibandingkan persis, deskripsi sebagian.
	mock.ExpectQuery(be4Q("diagnosa_cari")).WithArgs("fra_c%", `%FRA\_C\%%`).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b"}).AddRow(" S92.0 ", "FRACTURE OF CALCANEUS"))
	found, err = d.SearchDiagnosis(context.Background(), " fra_c% ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.DiagnosisOption{{Code: "S92.0", Description: "FRACTURE OF CALCANEUS"}}, found)

	mock.ExpectQuery(be4Q("diagnosa_cari")).WillReturnError(be4Boom)
	_, err = d.SearchDiagnosis(context.Background(), "x")
	require.ErrorIs(t, err, be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())

	mem, err := memory.DiagnosisDirectory{}.SearchDiagnosis(context.Background(), "fracture")
	require.NoError(t, err)
	require.Len(t, mem, 2)
}
