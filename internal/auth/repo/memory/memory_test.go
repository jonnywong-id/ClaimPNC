package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/repo/memory"
)

var (
	firstAt  = time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	secondAt = time.Date(2026, 9, 2, 3, 0, 0, 0, time.UTC)
)

func TestUserRepoSaveAndGet(t *testing.T) {
	repo := memory.NewUserRepo()
	ctx := context.Background()

	_, err := repo.GetByIdentity(ctx, "1")
	require.ErrorIs(t, err, auth.ErrUserNotFound)

	require.NoError(t, repo.Save(ctx, auth.User{Identity: "1", Name: "A", Active: true, OperatorID: "OP", CreatedAt: firstAt}))
	got, err := repo.GetByIdentity(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, "A", got.Name)
}

func TestUserRepoSaveKeepsLocalFieldsOfExistingUser(t *testing.T) {
	repo := memory.NewUserRepo()
	ctx := context.Background()
	require.NoError(t, repo.Save(ctx, auth.User{Identity: "1", Name: "A", Active: true, OperatorID: "OP", CreatedAt: firstAt}))

	// Simpan ulang membawa nilai lain untuk kolom milik aplikasi: yang lama dipertahankan.
	require.NoError(t, repo.Save(ctx, auth.User{Identity: "1", Name: "B", Active: false, OperatorID: "X", CreatedAt: secondAt}))
	got, err := repo.GetByIdentity(ctx, "1")
	require.NoError(t, err)
	require.Equal(t, "B", got.Name)
	require.True(t, got.Active)
	require.Equal(t, "OP", got.OperatorID)
	require.Equal(t, firstAt, got.CreatedAt)
}

func TestUserRepoSetActive(t *testing.T) {
	repo := memory.NewUserRepo()
	ctx := context.Background()
	require.NoError(t, repo.Save(ctx, auth.User{Identity: "1", Active: true}))

	repo.SetActive("1", false)
	got, err := repo.GetByIdentity(ctx, "1")
	require.NoError(t, err)
	require.False(t, got.Active)

	// Identitas yang tidak ada tidak dibuat.
	repo.SetActive("2", true)
	_, err = repo.GetByIdentity(ctx, "2")
	require.ErrorIs(t, err, auth.ErrUserNotFound)
}

func TestSessionRepoLifecycle(t *testing.T) {
	repo := memory.NewSessionRepo()
	ctx := context.Background()

	_, err := repo.GetByTokenDigest(ctx, "x")
	require.ErrorIs(t, err, auth.ErrSessionNotFound)

	require.NoError(t, repo.Save(ctx, auth.Session{ID: "s1", TokenDigest: "d1", ExpiresAt: firstAt}))
	require.NoError(t, repo.Save(ctx, auth.Session{ID: "s2", TokenDigest: "d2", ExpiresAt: firstAt}))
	require.Equal(t, 2, repo.Count())

	wib := time.FixedZone("WIB", 7*3600)
	require.NoError(t, repo.Renew(ctx, "s1", secondAt.In(wib)))
	got, err := repo.GetByTokenDigest(ctx, "d1")
	require.NoError(t, err)
	require.Equal(t, secondAt, got.ExpiresAt)
	require.Equal(t, time.UTC, got.ExpiresAt.Location())

	require.NoError(t, repo.Revoke(ctx, "s1", secondAt.In(wib)))
	got, err = repo.GetByTokenDigest(ctx, "d1")
	require.NoError(t, err)
	require.NotNil(t, got.RevokedAt)
	require.Equal(t, secondAt, *got.RevokedAt)

	// Sesi yang sudah dicabut tidak diperpanjang dan waktu cabutnya tidak berubah.
	require.NoError(t, repo.Renew(ctx, "s1", secondAt.Add(time.Hour)))
	require.NoError(t, repo.Revoke(ctx, "s1", secondAt.Add(time.Hour)))
	got, err = repo.GetByTokenDigest(ctx, "d1")
	require.NoError(t, err)
	require.Equal(t, secondAt, got.ExpiresAt)
	require.Equal(t, secondAt, *got.RevokedAt)

	// Sesi lain tidak tersentuh.
	other, err := repo.GetByTokenDigest(ctx, "d2")
	require.NoError(t, err)
	require.Nil(t, other.RevokedAt)
	require.Equal(t, firstAt, other.ExpiresAt)
}
