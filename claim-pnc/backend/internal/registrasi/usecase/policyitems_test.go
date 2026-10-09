package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/usecase"
)

// Klaim yang baru dibuka sudah membawa objek, coverage, dan spreading polisnya
// (CallActivityInputRegister), dan ketiganya tersimpan bersama klaim.
func TestStartFillsItemsFromPolicy(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)

	require.Len(t, start.Claim.InsuredItem, 1)
	item := start.Claim.InsuredItem[0]
	require.Len(t, item.Coverage, 1)
	require.Equal(t, "FLEXAS", item.Coverage[0].Name)
	require.Len(t, item.Coverage[0].Spreading, 1)
	require.Equal(t, 1, start.Claim.SpreadingCount())

	saved, err := l.service.ViewClaim(ctx, start.Claim.ID, l.caller)
	require.NoError(t, err)
	require.Equal(t, start.Claim.InsuredItem, saved.Claim.InsuredItem)
}

// Nama Treaty klaim baru terisi dari master nama treaty, bukan dibiarkan kosong: dokumen
// polis hanya menyimpan jenis treaty-nya.
func TestStartFillsTreatyName(t *testing.T) {
	l := setup(t)
	start, err := l.service.Start(context.Background(), usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	require.Equal(t, "OR", start.Claim.InsuredItem[0].Coverage[0].Spreading[0].Name)
}

// Penyebab Kerugian terisi sendiri HANYA bila kode bisnis polis punya tepat satu pilihan.
// Lebih dari satu pilihan berarti petugas yang memilih — tidak ada tebakan.
func TestStartFillsCauseOfLossOnlyWhenThereIsOneChoice(t *testing.T) {
	ctx := context.Background()

	many := setup(t)
	start, err := many.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, many.caller)
	require.NoError(t, err)
	require.Equal(t, "", start.Claim.InsuredItem[0].Coverage[0].CauseOfLoss)

	one := setup(t)
	one.areas.Causes["10013"] = []registrasi.CauseOfLossOption{{ID: "11997", Name: "FIRE - OPEN FLAME"}}
	start, err = one.service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, one.caller)
	require.NoError(t, err)
	require.Equal(t, "11997", start.Claim.InsuredItem[0].Coverage[0].CauseOfLoss)

	options, err := one.service.CauseOfLossOptions(ctx, "10013")
	require.NoError(t, err)
	require.Len(t, options, 1)
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
