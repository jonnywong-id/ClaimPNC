package masterpenolakan_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
)

// Pesan galat validasi menyebut setiap isian beserta pesannya, dipisah titik koma.
func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	err := masterpenolakan.Input{}.Check()
	require.EqualError(t, err,
		"masterpenolakan: isian tidak sah (nama: Status Penolakan 2 wajib diisi.; "+
			"nama_status_1: Status Penolakan 1 wajib diisi.)")
}
