package mastersupplier_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
)

// Pesan galat validasi menyebut setiap pelanggaran beserta isiannya, dipisah titik koma.
func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	err := &mastersupplier.ValidationError{Violation: []mastersupplier.Violation{
		{Field: "nama", Message: "wajib diisi"},
		{Field: "kota", Message: "wajib diisi"},
	}}
	require.Equal(t, "mastersupplier: isian tidak sah (nama: wajib diisi; kota: wajib diisi)", err.Error())

	one := mastersupplier.OneViolation("nama", "sudah dipakai")
	require.Equal(t, "mastersupplier: isian tidak sah (nama: sudah dipakai)", one.Error())
}
