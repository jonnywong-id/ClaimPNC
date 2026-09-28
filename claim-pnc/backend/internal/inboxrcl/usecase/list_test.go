package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/inboxrcl/repo/memory"
	"claim-pnc/internal/inboxrcl/usecase"
	"claim-pnc/internal/portal"
)

func service(t *testing.T) *usecase.Service {
	t.Helper()
	store := memory.NewSampleStore()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxrcl.Repo, error) {
			if alias != "ASM" {
				return nil, portal.ErrNotReady
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return svc
}

func TestAntreanDisaringDenganIdentitasLamaBukanLogin(t *testing.T) {
	listed, err := service(t).List(context.Background(), "ASM",
		inboxrcl.Caller{Login: memory.SampleLogin}, inboxrcl.Filter{})
	require.NoError(t, err)

	require.True(t, listed.LegacyIdentityFound)
	require.Equal(t, 4, listed.Page.Total)
	for _, task := range listed.Page.Tasks {
		require.Equal(t, memory.SampleLegacyID, task.AssignedOperator)
	}
}

// TestTanpaIdentitasLamaAntreanKosongDanDinyatakan membedakan "tidak ada pekerjaan" dari
// "belum diketahui pekerjaan siapa" — keduanya grid kosong di Pega.
func TestTanpaIdentitasLamaAntreanKosongDanDinyatakan(t *testing.T) {
	listed, err := service(t).List(context.Background(), "ASM",
		inboxrcl.Caller{Login: memory.SampleLoginNoLegacy}, inboxrcl.Filter{})
	require.NoError(t, err)

	require.False(t, listed.LegacyIdentityFound)
	require.Empty(t, listed.Page.Tasks)
	require.Zero(t, listed.Page.Total)
}

func TestTanpaLoginDitolak(t *testing.T) {
	_, err := service(t).List(context.Background(), "ASM", inboxrcl.Caller{}, inboxrcl.Filter{})
	require.ErrorIs(t, err, inboxrcl.ErrCallerUnknown)
}

// TestPortalTidakDikenalGalatBukanCadangan — `R-20`.
func TestPortalTidakDikenalGalatBukanCadangan(t *testing.T) {
	_, err := service(t).List(context.Background(), "LAIN",
		inboxrcl.Caller{Login: memory.SampleLogin}, inboxrcl.Filter{})
	require.True(t, errors.Is(err, portal.ErrNotReady))
}

func TestBatasDipangkasDanDikembalikan(t *testing.T) {
	listed, err := service(t).List(context.Background(), "ASM",
		inboxrcl.Caller{Login: memory.SampleLogin}, inboxrcl.Filter{Limit: 5000, Offset: -3})
	require.NoError(t, err)

	require.Equal(t, inboxrcl.MaxLimit, listed.Filter.Limit)
	require.Zero(t, listed.Filter.Offset)
}

// TestKolomMengikutiHarness mengunci kelima judul `pyCaption` dan urutan sel section.
func TestKolomMengikutiHarness(t *testing.T) {
	titles := []string{}
	for _, c := range usecase.Columns() {
		titles = append(titles, c.Title)
	}
	require.Equal(t, []string{
		"Nomor Case", "No Polis", "Nama Tertanggung", "Tanggal Masuk Inbox", "Deskripsi Analyst",
	}, titles)
}

func TestTanpaSelectorDitolak(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}
