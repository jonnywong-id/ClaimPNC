package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLengthAndRequired(t *testing.T) {
	require.Nil(t, Length("nama", "Nama", "abc", 3))
	require.Equal(t, []Violation{{Field: "nama", Message: "Nama paling panjang 3 karakter."}}, Length("nama", "Nama", "abcd", 3))
	require.Equal(t, []Violation{{Field: "nama", Message: "Nama wajib diisi."}}, Required("nama", "Nama", "", 3))
	require.Nil(t, Required("nama", "Nama", "ab", 3))
}

func TestEmailLooksValid(t *testing.T) {
	require.True(t, EmailLooksValid(" a@b.co "))
	for _, bad := range []string{"", "@b.co", "a@", "a@b@c.co", "a@bco", "a@.co", "a@b."} {
		require.False(t, EmailLooksValid(bad), bad)
	}
}
