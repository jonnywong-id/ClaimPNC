package provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// sumberTiruan adalah satu mata rantai yang jawabannya ditentukan uji.
type sumberTiruan struct {
	profil    auth.Profil
	galat     error
	dipanggil int
}

func (s *sumberTiruan) Verifikasi(context.Context, auth.Kredensial) (auth.Profil, error) {
	s.dipanggil++
	if s.galat != nil {
		return auth.Profil{}, s.galat
	}
	return s.profil, nil
}

func rantai(t *testing.T, sumber ...auth.Identitas) *provider.Berantai {
	t.Helper()
	mata := make([]provider.MataRantai, 0, len(sumber))
	for i, s := range sumber {
		mata = append(mata, provider.MataRantai{Nama: string(rune('a' + i)), Sumber: s})
	}
	b, err := provider.BerantaiBaru(mata...)
	require.NoError(t, err)
	return b
}

func profilKaryawan() auth.Profil {
	return auth.Profil{Identitas: "99091100", Nama: "JONNY", Jenis: auth.Karyawan}
}

func profilNonKaryawan() auth.Profil {
	return auth.Profil{Identitas: "JONNY", Nama: "JONNY WONG", Jenis: auth.NonKaryawan}
}

// Langkah 1: HCQ berhasil → masuk, dan jalur kedua TIDAK dijalankan. Menjalankannya
// tetap berarti kata sandi karyawan ikut disidik dan dicocokkan ke tabel non-karyawan.
func TestBerantaiBerhentiSaatSumberPertamaBerhasil(t *testing.T) {
	hcq := &sumberTiruan{profil: profilKaryawan()}
	lokal := &sumberTiruan{profil: profilNonKaryawan()}

	profil, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.NoError(t, err)
	require.Equal(t, auth.Karyawan, profil.Jenis)
	require.Equal(t, 1, hcq.dipanggil)
	require.Zero(t, lokal.dipanggil, "jalur non-karyawan tidak boleh dijalankan bila HCQ menerima")
}

// Langkah 2: HCQ gagal → jalur POOLDATA.M_LOGIN_PNC dicoba.
func TestBerantaiLanjutKeSumberKeduaSaatPertamaMenolak(t *testing.T) {
	hcq := &sumberTiruan{galat: auth.ErrKredensialSalah}
	lokal := &sumberTiruan{profil: profilNonKaryawan()}

	profil, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.NoError(t, err)
	require.Equal(t, auth.NonKaryawan, profil.Jenis)
	require.Equal(t, 1, lokal.dipanggil)
}

// Broker harus tetap dapat masuk ketika HCC/HCQ sedang mati — sistem identitas yang
// putus tidak boleh ikut mengunci pengguna yang tidak memakainya.
func TestBerantaiTetapMelayaniSaatSumberPertamaPutus(t *testing.T) {
	hcq := &sumberTiruan{galat: auth.ErrSistemTidakTerhubung}
	lokal := &sumberTiruan{profil: profilNonKaryawan()}

	profil, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.NoError(t, err)
	require.Equal(t, auth.NonKaryawan, profil.Jenis)
}

// Bila ADA sumber yang putus dan tidak satu pun menerima, yang dilaporkan adalah
// "sistem tidak dapat dihubungi" — bukan "kata sandi salah". Kita memang tidak tahu
// apakah kredensialnya benar, dan menyuruh pengguna mengetik ulang sesuatu yang sudah
// benar hanya membuang waktunya.
func TestBerantaiMelaporkanPutusBukanKredensialSalah(t *testing.T) {
	hcq := &sumberTiruan{galat: auth.ErrSistemTidakTerhubung}
	lokal := &sumberTiruan{galat: auth.ErrKredensialSalah}

	_, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.ErrorIs(t, err, auth.ErrSistemTidakTerhubung)
	require.NotErrorIs(t, err, auth.ErrKredensialSalah)
}

// Bila seluruh sumber hidup dan semuanya menolak, barulah kredensialnya memang salah.
func TestBerantaiMelaporkanKredensialSalahBilaSemuaHidup(t *testing.T) {
	hcq := &sumberTiruan{galat: auth.ErrKredensialSalah}
	lokal := &sumberTiruan{galat: auth.ErrKredensialSalah}

	_, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.ErrorIs(t, err, auth.ErrKredensialSalah)
	require.NotErrorIs(t, err, auth.ErrSistemTidakTerhubung)
}

// Akun yang sengaja dinonaktifkan menghentikan rantai: mencoba sumber berikutnya
// untuknya tidak masuk akal.
func TestBerantaiBerhentiPadaPenggunaTidakAktif(t *testing.T) {
	hcq := &sumberTiruan{galat: auth.ErrPenggunaTidakAktif}
	lokal := &sumberTiruan{profil: profilNonKaryawan()}

	_, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.ErrorIs(t, err, auth.ErrPenggunaTidakAktif)
	require.Zero(t, lokal.dipanggil)
}

// Galat yang tidak dikenali — misalnya profil tidak lengkap dari sumber yang menjawab
// berhasil — tidak boleh tersamarkan menjadi "kredensial salah".
func TestBerantaiTidakMenelanGalatTakDikenal(t *testing.T) {
	bolong := &auth.GalatProfilTidakLengkap{FieldKosong: []string{"nama"}}
	hcq := &sumberTiruan{galat: bolong}
	lokal := &sumberTiruan{profil: profilNonKaryawan()}

	_, err := rantai(t, hcq, lokal).Verifikasi(context.Background(), auth.Kredensial{})
	require.ErrorAs(t, err, &bolong)
	require.Zero(t, lokal.dipanggil)
}

func TestBerantaiMenolakRantaiKosong(t *testing.T) {
	_, err := provider.BerantaiBaru()
	require.Error(t, err)

	_, err = provider.BerantaiBaru(provider.MataRantai{Nama: "kosong"})
	require.ErrorContains(t, err, "tanpa sumber identitas")
}
