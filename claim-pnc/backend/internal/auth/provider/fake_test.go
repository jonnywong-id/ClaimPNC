package provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// Provider tiruan menolak start bila lingkungan bertanda produksi, dan penolakannya ada
// di kode — bukan hanya di nilai konfigurasi (TKT-F3-001).
func TestFakeRefusesProduction(t *testing.T) {
	fake, err := provider.NewFake(true, nil)
	require.ErrorIs(t, err, provider.ErrFakeInProduction)
	require.Nil(t, fake)
}

func TestFakeRunsOutsideProduction(t *testing.T) {
	fake, err := provider.NewFake(false, nil)
	require.NoError(t, err)
	require.NotNil(t, fake)
}

// Ketiga jenis galat dapat dibedakan pemanggil TANPA memeriksa teks pesan
// (TKT-F3-001). Pembedaannya lewat errors.Is.
func TestFakeDistinguishesThreeErrorKinds(t *testing.T) {
	ctx := context.Background()
	fake, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	t.Run("kredensial salah", func(t *testing.T) {
		_, err := fake.Verify(ctx, auth.Credential{Username: "adminpnc", Password: "salah"})
		require.ErrorIs(t, err, auth.ErrWrongCredential)
		require.NotErrorIs(t, err, auth.ErrUserInactive)
		require.NotErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
	})

	t.Run("pengguna tidak aktif", func(t *testing.T) {
		_, err := fake.Verify(ctx, auth.Credential{Username: "penggunanonaktif", Password: "rahasia123"})
		require.ErrorIs(t, err, auth.ErrUserInactive)
		require.NotErrorIs(t, err, auth.ErrWrongCredential)
	})

	t.Run("sistem identitas tidak dapat dihubungi", func(t *testing.T) {
		down, err := provider.NewFake(false, nil)
		require.NoError(t, err)
		down.SetSimulateOutage(true)

		_, err = down.Verify(ctx, auth.Credential{Username: "adminpnc", Password: "rahasia123"})
		require.ErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
		require.NotErrorIs(t, err, auth.ErrWrongCredential)
	})
}

// Galat untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah harus
// PERSIS SAMA — membedakannya membocorkan siapa yang punya akun (TKT-U1-002).
func TestFakeDoesNotLeakAccountExistence(t *testing.T) {
	ctx := context.Background()
	fake, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	_, errTidakAda := fake.Verify(ctx, auth.Credential{Username: "tidakpernahada", Password: "apa saja"})
	_, errWrongPassword := fake.Verify(ctx, auth.Credential{Username: "adminpnc", Password: "bukan sandinya"})

	require.ErrorIs(t, errTidakAda, auth.ErrWrongCredential)
	require.ErrorIs(t, errWrongPassword, auth.ErrWrongCredential)
	require.Equal(t, errTidakAda.Error(), errWrongPassword.Error())
}

func TestFakeAcceptsValidCredential(t *testing.T) {
	fake, err := provider.NewFake(false, nil)
	require.NoError(t, err)

	profile, err := fake.Verify(context.Background(),
		auth.Credential{Username: "AdminPNC", Password: "rahasia123"})
	require.NoError(t, err)
	require.Equal(t, "90000001", profile.Identity)
	require.NoError(t, profile.Check())
}
