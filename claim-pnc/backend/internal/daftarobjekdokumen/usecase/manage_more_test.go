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

	_, err := service.Get(ctx, "XXX", "100001")
	require.ErrorIs(t, err, portal.ErrNotReady)
	_, err = service.Create(ctx, "XXX", daftarobjekdokumen.Input{})
	require.ErrorIs(t, err, portal.ErrNotReady)
	_, err = service.Update(ctx, "XXX", "100001", daftarobjekdokumen.Input{})
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Penyuntingan dengan isian tidak sah ditolak sebelum disimpan, dan isi lama tetap utuh.
func TestUpdateWithInvalidInputRejected(t *testing.T) {
	service, asm, _ := newService(t)
	_, err := service.Update(context.Background(), "ASM", "100001", daftarobjekdokumen.Input{
		Description: strings.Repeat("A", daftarobjekdokumen.MaxDescriptionLength+1),
	})
	var validation *daftarobjekdokumen.ValidationError
	require.ErrorAs(t, err, &validation)

	got, err := asm.Get(context.Background(), "100001")
	require.NoError(t, err)
	require.Equal(t, "KTP Tertanggung", got.Description)
}

// Bila pemilih master bisnis gagal, penyimpanan DITOLAK — bukan disimpan tanpa ID.
//
// # Uji ini mengunci perilaku yang BERKEBALIKAN dari versi sebelumnya
//
// Versi sebelumnya menuntut penyimpanan tetap berhasil dengan nama tersimpan tanpa ID.
// Itu benar untuk modul Master COL Simas Online, yang tabel pemetaannya punya kolom NOTE —
// dan keliru di sini.
//
// Pembacaan katalog 2026-10-03 membuktikan pemetaan modul ini hidup di dalam dokumen JSON
// baris induknya, dan bentuknya hanya memuat ID:
//
//	{"ID":"100766","LIST_LBU_ID":[{"ID":"10027"}],"KET_DOC_OBJ":"STOCK"}
//
// Tidak ada tempat bagi nama tanpa ID. "Berhasil disimpan tanpa ID" karena itu bukan
// kompromi yang baik hati melainkan kehilangan data yang senyap: pemetaannya tidak pernah
// benar-benar tersimpan.
func TestBusinessSelectorFailureRejectsSave(t *testing.T) {
	repo := memory.NewRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (daftarobjekdokumen.Repo, error) { return repo, nil },
		BusinessSelector: func(string) (daftarobjekdokumen.BusinessRepo, error) {
			return nil, portal.ErrNotReady
		},
	})
	require.NoError(t, err)

	_, err = service.Create(context.Background(), "ASM", daftarobjekdokumen.Input{
		Description:   "Polis",
		BusinessNames: []string{"ANEKA"},
	})
	require.ErrorIs(t, err, portal.ErrNotReady)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list, "tidak ada baris yang tersimpan")
}
