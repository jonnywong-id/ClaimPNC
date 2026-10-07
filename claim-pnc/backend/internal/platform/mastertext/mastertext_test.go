package mastertext

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/validation"
)

func TestCleanAndNormalize(t *testing.T) {
	d, names := Clean("  a  ", []string{" x ", "", "  ", "y"})
	require.Equal(t, "a", d)
	require.Equal(t, []string{"x", "y"}, names)
	require.Equal(t, "ASM", NormalizeBusinessName(" asm "))
}

func TestCheckLengths(t *testing.T) {
	l := Limits{DescriptionField: "nama", DescriptionLabel: "Nama X", MaxDescription: 3, BusinessField: "bisnis", MaxBusinessName: 2}
	require.Empty(t, CheckLengths("abc", []string{"ab"}, l))
	require.Equal(t, []validation.Violation{
		{Field: "nama", Message: "Nama X paling panjang 3 karakter."},
		{Field: "bisnis", Message: "Nama bisnis paling panjang 2 karakter: abc"},
	}, CheckLengths(strings.Repeat("é", 4), []string{"abc", "defg"}, l))
}
