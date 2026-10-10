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
func TestTiruanMenolakProduksi(t *testing.T) {
	tiruan, err := provider.TiruanBaru(true, nil)
	require.ErrorIs(t, err, provider.ErrTiruanDiProduksi)
	require.Nil(t, tiruan)
}

func TestTiruanBerjalanDiLuarProduksi(t *testing.T) {
	tiruan, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)
	require.NotNil(t, tiruan)
}

// Ketiga jenis galat dapat dibedakan pemanggil TANPA memeriksa teks pesan
// (TKT-F3-001). Pembedaannya lewat errors.Is.
func TestTiruanMembedakanTigaJenisGalat(t *testing.T) {
	ctx := context.Background()
	tiruan, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	t.Run("kredensial salah", func(t *testing.T) {
		_, err := tiruan.Verifikasi(ctx, auth.Kredensial{NamaPengguna: "adminpnc", KataSandi: "salah"})
		require.ErrorIs(t, err, auth.ErrKredensialSalah)
		require.NotErrorIs(t, err, auth.ErrPenggunaTidakAktif)
		require.NotErrorIs(t, err, auth.ErrSistemTidakTerhubung)
	})

	t.Run("pengguna tidak aktif", func(t *testing.T) {
		_, err := tiruan.Verifikasi(ctx, auth.Kredensial{NamaPengguna: "penggunanonaktif", KataSandi: "rahasia123"})
		require.ErrorIs(t, err, auth.ErrPenggunaTidakAktif)
		require.NotErrorIs(t, err, auth.ErrKredensialSalah)
	})

	t.Run("sistem identitas tidak dapat dihubungi", func(t *testing.T) {
		putus, err := provider.TiruanBaru(false, nil)
		require.NoError(t, err)
		putus.SetSimulasiPutus(true)

		_, err = putus.Verifikasi(ctx, auth.Kredensial{NamaPengguna: "adminpnc", KataSandi: "rahasia123"})
		require.ErrorIs(t, err, auth.ErrSistemTidakTerhubung)
		require.NotErrorIs(t, err, auth.ErrKredensialSalah)
	})
}

// Galat untuk pengguna yang tidak ada dan pengguna yang ada berkata sandi salah harus
// PERSIS SAMA — membedakannya membocorkan siapa yang punya akun (TKT-U1-002).
func TestTiruanTidakMembocorkanKeberadaanAkun(t *testing.T) {
	ctx := context.Background()
	tiruan, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	_, errTidakAda := tiruan.Verifikasi(ctx, auth.Kredensial{NamaPengguna: "tidakpernahada", KataSandi: "apa saja"})
	_, errSandiSalah := tiruan.Verifikasi(ctx, auth.Kredensial{NamaPengguna: "adminpnc", KataSandi: "bukan sandinya"})

	require.ErrorIs(t, errTidakAda, auth.ErrKredensialSalah)
	require.ErrorIs(t, errSandiSalah, auth.ErrKredensialSalah)
	require.Equal(t, errTidakAda.Error(), errSandiSalah.Error())
}

func TestTiruanMenerimaKredensialSah(t *testing.T) {
	tiruan, err := provider.TiruanBaru(false, nil)
	require.NoError(t, err)

	profil, err := tiruan.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "AdminPNC", KataSandi: "rahasia123"})
	require.NoError(t, err)
	require.Equal(t, "90000001", profil.Identitas)
	require.NoError(t, profil.Periksa())
}
