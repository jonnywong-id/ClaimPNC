package mastercolsimasonline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Pesan log galat validasi menyebut setiap isian beserta pesannya.
func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	err := &ValidationError{Violation: []Violation{
		{Field: FieldDescription, Message: "terlalu panjang"},
		{Field: FieldBusiness, Message: "juga terlalu panjang"},
	}}
	require.Equal(t,
		"mastercolsimasonline: isian tidak sah (nama: terlalu panjang; bisnis: juga terlalu panjang)",
		err.Error())
}

// itoa menuliskan nol sebagai "0", bukan teks kosong.
func TestItoaWritesZeroAndMultiDigitNumbers(t *testing.T) {
	require.Equal(t, "0", itoa(0))
	require.Equal(t, "100", itoa(100))
	require.Equal(t, "7", itoa(7))
}
