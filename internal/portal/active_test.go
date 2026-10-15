package portal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/portal"
	"claim-pnc/internal/portal/repo/memory"
)

func TestSelectActiveRejectsEmptyAlias(t *testing.T) {
	// Permintaan tanpa portal tidak jatuh ke portal utama (TKT-F6-002).
	for _, alias := range []string{"", "   "} {
		_, err := portal.SelectActive(memory.SampleList(), []string{"ASM"}, alias)
		require.ErrorIs(t, err, portal.ErrNotStated)
	}
}

func TestSelectActiveRejectsUnknownAlias(t *testing.T) {
	_, err := portal.SelectActive(memory.SampleList(), []string{"ASM"}, "TIDAKADA")
	require.ErrorIs(t, err, portal.ErrNotFound)
}

func TestSelectActiveRejectsPortalWhoseConnectionIsNotReady(t *testing.T) {
	// SMAS ada di daftar, tetapi koneksinya belum hidup.
	_, err := portal.SelectActive(memory.SampleList(), []string{"ASM", "ASI"}, "SMAS")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = portal.SelectActive(memory.SampleList(), nil, "ASM")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

func TestSelectActiveIgnoresCaseOnBothSides(t *testing.T) {
	selected, err := portal.SelectActive(memory.SampleList(), []string{"asi", " asm "}, " Asm ")
	require.NoError(t, err)
	require.Equal(t, portal.Portal{ID: "202600101", Name: "ASURANSI SINAR MAS", Alias: "ASM"}, selected)
}
