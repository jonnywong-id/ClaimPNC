package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// be4MockDB membuat satu koneksi tiruan yang mengharapkan ditutup.
func be4MockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	require.NoError(t, err)
	mock.ExpectClose()
	return conn, mock
}

// be4Unreachable adalah parameter koneksi ke port yang tidak melayani apa pun.
func be4Unreachable(alias string) Parameter {
	return Parameter{
		Alias: alias, Host: "127.0.0.1", Port: 1, Service: "XE",
		User: "pengguna", Password: "s@ndi/rahasia", MaxConnections: 2, MaxIdle: 1,
	}
}

// TestPoolLookupsAndClose memeriksa Has, Primary, For, Available, dan Close.
func TestPoolLookupsAndClose(t *testing.T) {
	asm, asmMock := be4MockDB(t)
	asi, asiMock := be4MockDB(t)
	pool := &Pool{connections: map[string]*sql.DB{"ASM": asm, "ASI": asi}, primary: "ASM"}

	require.True(t, pool.Has(" asm "))
	require.False(t, pool.Has("SMAS"))
	require.Same(t, asm, pool.Primary())
	require.Equal(t, "ASM", pool.PrimaryAlias())
	require.Equal(t, []string{"ASI", "ASM"}, pool.Available())

	got, err := pool.For("asi")
	require.NoError(t, err)
	require.Same(t, asi, got)

	_, err = pool.For("TL")
	require.EqualError(t, err, `db: portal "TL" tidak tersedia; yang tersedia: ASI, ASM`)

	pool.Close()
	require.NoError(t, asmMock.ExpectationsWereMet())
	require.NoError(t, asiMock.ExpectationsWereMet())
}

// TestDSNEscapesCredentials membuktikan karakter khusus pada kredensial disandikan URL.
func TestDSNEscapesCredentials(t *testing.T) {
	require.Equal(t, "oracle://pengguna:s%40ndi%2Frahasia@127.0.0.1:1/XE", dsn(be4Unreachable("ASM")))
}

// TestOpenFailsOnUnreachableHostWithoutLeakingPassword membuktikan galat ping menyebut alamat, bukan sandi.
func TestOpenFailsOnUnreachableHostWithoutLeakingPassword(t *testing.T) {
	_, err := Open(context.Background(), be4Unreachable("ASM"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "tidak dapat menghubungi Oracle di 127.0.0.1:1/XE sebagai pengguna")
	require.NotContains(t, err.Error(), "rahasia")
}

// TestNewPoolPrimaryFailureIsFatal membuktikan portal utama yang gagal menggagalkan start.
func TestNewPoolPrimaryFailureIsFatal(t *testing.T) {
	pool, err := NewPool(context.Background(), "ASM", []Parameter{be4Unreachable("ASM")}, nil)
	require.Nil(t, pool)
	require.ErrorContains(t, err, `portal utama "ASM"`)
}

// TestNewPoolSecondaryFailureIsRecordedAndPrimaryMissingIsFatal memeriksa portal lain yang gagal.
func TestNewPoolSecondaryFailureIsRecordedAndPrimaryMissingIsFatal(t *testing.T) {
	var recorded []string
	pool, err := NewPool(context.Background(), "ASM", []Parameter{be4Unreachable("ASI")},
		func(alias string, err error) {
			require.Error(t, err)
			recorded = append(recorded, alias)
		})
	require.Nil(t, pool)
	require.EqualError(t, err, `db: portal utama "ASM" tidak ada di daftar koneksi`)
	require.Equal(t, []string{"ASI"}, recorded)

	// Tanpa pencatat pun kegagalan portal lain tidak membuat panik.
	_, err = NewPool(context.Background(), "ASM", []Parameter{be4Unreachable("ASI")}, nil)
	require.Error(t, err)
}

// TestNewOptionalPoolNeverFails membuktikan kumpulan opsional menelan kegagalan ke pencatat.
func TestNewOptionalPoolNeverFails(t *testing.T) {
	var recorded []string
	pool := NewOptionalPool(context.Background(),
		[]Parameter{be4Unreachable("ASM"), be4Unreachable("ASI")},
		func(alias string, err error) { recorded = append(recorded, alias) })
	require.NotNil(t, pool)
	require.Empty(t, pool.Available())
	require.False(t, pool.Has("ASM"))
	require.Equal(t, []string{"ASM", "ASI"}, recorded)

	pool = NewOptionalPool(context.Background(), []Parameter{be4Unreachable("TL")}, nil)
	require.Empty(t, pool.Available())
}

// TestIsDuplicateKey membuktikan hanya ORA-00001 yang dianggap tabrakan kunci.
func TestIsDuplicateKey(t *testing.T) {
	require.False(t, IsDuplicateKey(nil))
	require.True(t, IsDuplicateKey(errors.New("ORA-00001: unique constraint violated")))
	require.False(t, IsDuplicateKey(errors.New("ORA-00942: table or view does not exist")))
}
