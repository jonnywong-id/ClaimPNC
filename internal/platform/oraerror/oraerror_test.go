package oraerror

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDescribeWriteWithNamedTable(t *testing.T) {
	err := fmt.Errorf("registrasi/sqlstore: menyisipkan klaim ke POOLDATA.T_CLAIM_PNC: %w",
		errors.New("ORA-01427: single-row subquery returns more than one row\n error occur at position: 12"))
	msg, ok := Describe(err)
	require.True(t, ok)
	require.Equal(t, "Gagal penyimpanan ke tabel POOLDATA.T_CLAIM_PNC: ORA-01427: single-row subquery returns more than one row", msg)
}

func TestDescribeTableFromOracleText(t *testing.T) {
	err := fmt.Errorf("x: menyimpan objek 1: %w", errors.New(
		`ORA-12899: value too large for column "POOLDATA"."T_CLAIM_PNC"."COINSNAME" (actual: 53, maximum: 50)`))
	msg, ok := Describe(err)
	require.True(t, ok)
	require.Equal(t, `Gagal penyimpanan ke tabel POOLDATA.T_CLAIM_PNC: ORA-12899: value too large for column "POOLDATA"."T_CLAIM_PNC"."COINSNAME" (actual: 53, maximum: 50)`, msg)
}

func TestDescribeReadAndUnknownTable(t *testing.T) {
	msg, ok := Describe(fmt.Errorf("x: membaca berkas: %w", errors.New("ORA-00942: table or view does not exist")))
	require.True(t, ok)
	require.Equal(t, "Gagal mengakses basis data: ORA-00942: table or view does not exist", msg)

	msg, ok = Describe(fmt.Errorf("x: menyimpan: %w", errors.New("ORA-00001: unique constraint (POOLDATA.PK_X) violated")))
	require.True(t, ok)
	require.Equal(t, "Gagal penyimpanan ke basis data: ORA-00001: unique constraint (POOLDATA.PK_X) violated", msg)
}

func TestDescribeIgnoresNonOracle(t *testing.T) {
	_, ok := Describe(errors.New("jaringan putus"))
	require.False(t, ok)
	_, ok = Describe(nil)
	require.False(t, ok)
}
