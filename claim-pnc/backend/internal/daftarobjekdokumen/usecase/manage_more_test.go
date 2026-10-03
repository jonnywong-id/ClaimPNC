package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftarobjekdokumen"
	"claim-pnc/internal/daftarobjekdokumen/repo/memory"
	"claim-pnc/internal/daftarobjekdokumen/usecase"
	"claim-pnc/internal/portal"
)

// Portal yang tidak dapat dilayani menolak Get, Create, dan Update dengan galat pemilihnya.
func TestUnknownPortalRejectedOnEveryOperation(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	_, err := service.Get(ctx, "XXX", "10001")
	require.ErrorIs(t, err, portal.ErrNotReady)
	_, err = service.Create(ctx, "XXX", daftarobjekdokumen.Input{})
	require.ErrorIs(t, err, portal.ErrNotReady)
	_, err = service.Update(ctx, "XXX", "10001", daftarobjekdokumen.Input{})
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Penyuntingan dengan isian tidak sah ditolak sebelum disimpan, dan isi lama tetap utuh.
func TestUpdateWithInvalidInputRejected(t *testing.T) {
	service, asm, _ := newService(t)
	_, err := service.Update(context.Background(), "ASM", "10001", daftarobjekdokumen.Input{
		Description: strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1),
	})
	var validation *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validation)

	got, err := asm.Get(context.Background(), "10001")
	require.NoError(t, err)
	require.Equal(t, "KTP Tertanggung", got.Description)
}

// Bila pemilih master bisnis gagal, penyimpanan tetap berhasil; nama disimpan tanpa ID.
func TestBusinessSelectorFailureKeepsNamesWithoutID(t *testing.T) {
	repo := memory.NewRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (daftarobjekdokumen.Repo, error) { return repo, nil },
		BusinessSelector: func(string) (daftarobjekdokumen.BusinessRepo, error) {
			return nil, portal.ErrNotReady
		},
	})
	require.NoError(t, err)

	saved, err := service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description:   "Polis",
		BusinessNames: []string{"ANEKA"},
	})
	require.NoError(t, err)
	require.Equal(t, []daftarobjekdokumen.Business{{Name: "ANEKA"}}, saved.Businesses)
}
