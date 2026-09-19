package provider_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// fakeSource adalah satu mata rantai yang jawabannya ditentukan uji.
type fakeSource struct {
	profile   auth.Profile
	issues    error
	dipanggil int
}

func (s *fakeSource) Verify(context.Context, auth.Credential) (auth.Profile, error) {
	s.dipanggil++
	if s.issues != nil {
		return auth.Profile{}, s.issues
	}
	return s.profile, nil
}

func rantai(t *testing.T, sumber ...auth.Identity) *provider.Chain {
	t.Helper()
	mata := make([]provider.Link, 0, len(sumber))
	for i, s := range sumber {
		mata = append(mata, provider.Link{Name: string(rune('a' + i)), Sumber: s})
	}
	b, err := provider.NewChain(mata...)
	require.NoError(t, err)
	return b
}

func employeeProfile() auth.Profile {
	return auth.Profile{Identity: "99091100", Name: "JONNY", Kind: auth.Employee}
}

func nonEmployeeProfile() auth.Profile {
	return auth.Profile{Identity: "JONNY", Name: "JONNY WONG", Kind: auth.NonEmployee}
}

// Langkah 1: HCQ berhasil → masuk, dan jalur kedua TIDAK dijalankan. Menjalankannya
// tetap berarti kata sandi karyawan ikut disidik dan dicocokkan ke tabel non-karyawan.
func TestChainStopsWhenFirstSourceSucceeds(t *testing.T) {
	hcq := &fakeSource{profile: employeeProfile()}
	local := &fakeSource{profile: nonEmployeeProfile()}

	profile, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.Employee, profile.Kind)
	require.Equal(t, 1, hcq.dipanggil)
	require.Zero(t, local.dipanggil, "jalur non-karyawan tidak boleh dijalankan bila HCQ menerima")
}

// Langkah 2: HCQ gagal → jalur POOLDATA.M_LOGIN_PNC dicoba.
func TestChainFallsToSecondSourceWhenFirstRejects(t *testing.T) {
	hcq := &fakeSource{issues: auth.ErrWrongCredential}
	local := &fakeSource{profile: nonEmployeeProfile()}

	profile, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.NonEmployee, profile.Kind)
	require.Equal(t, 1, local.dipanggil)
}

// Broker harus tetap dapat masuk ketika HCC/HCQ sedang mati — sistem identitas yang
// putus tidak boleh ikut mengunci pengguna yang tidak memakainya.
func TestChainKeepsServingWhenFirstSourceIsDown(t *testing.T) {
	hcq := &fakeSource{issues: auth.ErrIdentitySystemUnreachable}
	local := &fakeSource{profile: nonEmployeeProfile()}

	profile, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.NonEmployee, profile.Kind)
}

// Bila ADA sumber yang putus dan tidak satu pun menerima, yang dilaporkan adalah
// "sistem tidak dapat dihubungi" — bukan "kata sandi salah". Kita memang tidak tahu
// apakah kredensialnya benar, dan menyuruh pengguna mengetik ulang sesuatu yang sudah
// benar hanya membuang waktunya.
func TestChainReportsOutageNotWrongCredential(t *testing.T) {
	hcq := &fakeSource{issues: auth.ErrIdentitySystemUnreachable}
	local := &fakeSource{issues: auth.ErrWrongCredential}

	_, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
	require.NotErrorIs(t, err, auth.ErrWrongCredential)
}

// Bila seluruh sumber hidup dan semuanya menolak, barulah kredensialnya memang salah.
func TestChainReportsWrongCredentialWhenAllSourcesAlive(t *testing.T) {
	hcq := &fakeSource{issues: auth.ErrWrongCredential}
	local := &fakeSource{issues: auth.ErrWrongCredential}

	_, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrWrongCredential)
	require.NotErrorIs(t, err, auth.ErrIdentitySystemUnreachable)
}

// Akun yang sengaja dinonaktifkan menghentikan rantai: mencoba sumber berikutnya
// untuknya tidak masuk akal.
func TestChainStopsAtInactiveUser(t *testing.T) {
	hcq := &fakeSource{issues: auth.ErrUserInactive}
	local := &fakeSource{profile: nonEmployeeProfile()}

	_, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrUserInactive)
	require.Zero(t, local.dipanggil)
}

// Galat yang tidak dikenali — misalnya profil tidak lengkap dari sumber yang menjawab
// berhasil — tidak boleh tersamarkan menjadi "kredensial salah".
func TestChainDoesNotSwallowUnknownError(t *testing.T) {
	bolong := &auth.IncompleteProfileError{EmptyFields: []string{"nama"}}
	hcq := &fakeSource{issues: bolong}
	local := &fakeSource{profile: nonEmployeeProfile()}

	_, err := rantai(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorAs(t, err, &bolong)
	require.Zero(t, local.dipanggil)
}

func TestChainRejectsEmptyChain(t *testing.T) {
	_, err := provider.NewChain()
	require.Error(t, err)

	_, err = provider.NewChain(provider.Link{Name: "kosong"})
	require.ErrorContains(t, err, "tanpa sumber identitas")
}
