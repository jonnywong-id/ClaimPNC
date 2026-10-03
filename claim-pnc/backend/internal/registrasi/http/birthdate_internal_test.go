package registrasihttp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ASMDATEOFBIRTH T_PERSONLIST: yyyymmdd (mayoritas) atau dd/mm/yyyy; bentuk lain dikosongkan.
func TestBirthDate(t *testing.T) {
	require.Equal(t, "1989-05-24", birthDate("19890524"))
	require.Equal(t, "1989-05-24", birthDate(" 24/05/1989 "))
	require.Equal(t, "", birthDate(""))
	require.Equal(t, "", birthDate("24-05-89"))
}
