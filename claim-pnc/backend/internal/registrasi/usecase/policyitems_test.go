package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi/usecase"
)

// Klaim yang baru dibuka membawa objek polisnya, tetapi TANPA coverage — petugas
// menambahkannya sendiri lewat Tambah coverage, untuk seluruh lini (Work Owner 2026-10-09).
// Pilihan dropdown-nya tetap coverage polis objek itu, lengkap dengan spreading dan nama treaty.
func TestStartFillsItemsFromPolicy(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)

	require.Len(t, start.Claim.InsuredItem, 1)
	item := start.Claim.InsuredItem[0]
	require.Empty(t, item.Coverage)
	require.Equal(t, 0, start.Claim.SpreadingCount())

	saved, err := l.service.ViewClaim(ctx, start.Claim.ID, l.caller)
	require.NoError(t, err)
	require.Len(t, saved.Claim.InsuredItem, 1)
	require.Equal(t, item.ID, saved.Claim.InsuredItem[0].ID)
	require.Empty(t, saved.Claim.InsuredItem[0].Coverage)

	options, err := l.service.CoverageOptions(ctx, start.Claim.ID, item.ID)
	require.NoError(t, err)
	require.Len(t, options, 1)
	require.Equal(t, "FLEXAS", options[0].Name)
	require.Len(t, options[0].Spreading, 1)
	// Nama Treaty terisi dari master nama treaty, bukan dibiarkan kosong: dokumen polis hanya
	// menyimpan jenis treaty-nya.
	require.Equal(t, "OR", options[0].Spreading[0].Name)
}

// PA: objek (peserta) tetap dibuat dari polis, tetapi coverage tidak diisi otomatis — petugas
// menambahkannya sendiri (Work Owner 2026-10-08).
func TestStartLeavesPACoverageEmpty(t *testing.T) {
	l := setup(t)
	start, err := l.service.Start(context.Background(), usecase.StartCommand{PolicyNumber: "POL-PA-0002", Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	require.Len(t, start.Claim.InsuredItem, 1)
	require.Equal(t, "PESERTA CONTOH", start.Claim.InsuredItem[0].Name)
	require.Empty(t, start.Claim.InsuredItem[0].Coverage)
}

// PA: Deskripsi Laporan klaim yang baru dibuka diisi kalimat baku CallActivityInputRegister
// langkah 44.
func TestStartFillsPAReportDescription(t *testing.T) {
	l := setup(t)
	start, err := l.service.Start(context.Background(), usecase.StartCommand{PolicyNumber: "POL-PA-0002", Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(start.Claim.Chronology, "Berdasarkan surat keterangan kematian dari ..."))

	fire, err := l.service.Start(context.Background(), usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	require.NotContains(t, fire.Claim.Chronology, "surat keterangan kematian")
}
