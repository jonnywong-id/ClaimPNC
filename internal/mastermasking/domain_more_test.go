package mastermasking_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastermasking"
)

// Modul dan sub modul yang melewati batas panjang ditolak pada isiannya masing-masing.
func TestModuleAndSubModuleTooLong(t *testing.T) {
	m := valid()
	m.Module = strings.Repeat("m", mastermasking.MaxModuleLength+1)
	m.SubModule = strings.Repeat("s", mastermasking.MaxSubModuleLength+1)

	violations := mastermasking.CheckMasking(m)
	require.Equal(t, []mastermasking.Violation{
		{Field: mastermasking.FieldModule, Message: "Modul paling panjang 100 karakter."},
		{Field: mastermasking.FieldSubModule, Message: "Sub modul paling panjang 400 karakter."},
	}, violations)
}

// Penyaring dipangkas spasinya, baik jenis pencarian maupun kata kuncinya.
func TestFilterClean(t *testing.T) {
	got := mastermasking.Filter{By: " status ", Keyword: "  aktif "}.Clean()
	require.Equal(t, mastermasking.Filter{By: mastermasking.SearchByStatus, Keyword: "aktif"}, got)
}

// Pesan galat validasi menyebut setiap pelanggaran; daftar kosong tidak menghasilkan galat.
func TestValidationErrorMessage(t *testing.T) {
	require.NoError(t, mastermasking.NewValidationError(nil))

	err := mastermasking.NewValidationError([]mastermasking.Violation{
		{Field: mastermasking.FieldLogin, Message: "wajib"},
		{Field: mastermasking.FieldModule, Message: "terlalu panjang"},
	})
	require.EqualError(t, err, "mastermasking: validasi gagal — login: wajib; modul: terlalu panjang")

	var validation *mastermasking.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violation, 2)
}
