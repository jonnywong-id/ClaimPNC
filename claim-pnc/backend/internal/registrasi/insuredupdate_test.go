package registrasi_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// No KTP dan Pengkinian Data dibatasi panjang kolomnya: 100, 100, dan 200 karakter.
func TestValidateInsuredUpdate(t *testing.T) {
	ok := registrasi.InsuredUpdate{IDCard: strings.Repeat("1", 100), Phone: strings.Repeat("0", 100), Email: strings.Repeat("e", 200)}
	require.NoError(t, registrasi.ValidateInsuredUpdate(ok))

	err := registrasi.ValidateInsuredUpdate(registrasi.InsuredUpdate{
		IDCard: strings.Repeat("1", 101), Phone: strings.Repeat("0", 101), Email: strings.Repeat("e", 201),
	})
	var v *registrasi.ValidationError
	require.ErrorAs(t, err, &v)
	require.Len(t, v.Violation, 3)

	require.Equal(t, registrasi.InsuredUpdate{IDCard: "123", Phone: "0812", Email: "a@contoh.example"},
		registrasi.InsuredUpdate{IDCard: " 123 ", Phone: "0812 ", Email: " a@contoh.example"}.Trimmed())
}

// Jenis Laporan dibatasi panjang kolom REPORTTYPE (10 karakter).
func TestValidateReportType(t *testing.T) {
	require.NoError(t, registrasi.ValidateReportType("1"))
	require.NoError(t, registrasi.ValidateReportType(strings.Repeat("9", 10)))
	var v *registrasi.ValidationError
	require.ErrorAs(t, registrasi.ValidateReportType(strings.Repeat("9", 11)), &v)
	require.Equal(t, registrasi.ViolationReportTypeTooLong, v.Violation[0].Code)
}
