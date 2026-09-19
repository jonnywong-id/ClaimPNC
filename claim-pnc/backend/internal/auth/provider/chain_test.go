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
	profile auth.Profile
	failure error
	called  int
}

func (s *fakeSource) Verify(context.Context, auth.Credential) (auth.Profile, error) {
	s.called++
	if s.failure != nil {
		return auth.Profile{}, s.failure
	}
	return s.profile, nil
}

func chain(t *testing.T, source ...auth.Identity) *provider.Chain {
	t.Helper()
	code := make([]provider.ChainLink, 0, len(source))
	for i, s := range source {
		code = append(code, provider.ChainLink{Name: string(rune('a' + i)), Source: s})
	}
	b, err := provider.NewChain(code...)
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

	profile, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.Employee, profile.Kind)
	require.Equal(t, 1, hcq.called)
	require.Zero(t, local.called, "jalur non-karyawan tidak boleh dijalankan bila HCQ menerima")
}

// Langkah 2: HCQ gagal → jalur POOLDATA.M_LOGIN_PNC dicoba.
func TestChainFallsThroughWhenFirstSourceRejects(t *testing.T) {
	hcq := &fakeSource{failure: auth.ErrWrongCredential}
	local := &fakeSource{profile: nonEmployeeProfile()}

	profile, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.NonEmployee, profile.Kind)
	require.Equal(t, 1, local.called)
}

// Broker harus tetap dapat masuk ketika HCC/HCQ sedang mati — sistem identitas yang
// putus tidak boleh ikut mengunci pengguna yang tidak memakainya.
func TestChainKeepsServingWhenFirstSourceIsDown(t *testing.T) {
	hcq := &fakeSource{failure: auth.ErrIdentitySystemDown}
	local := &fakeSource{profile: nonEmployeeProfile()}

	profile, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.NoError(t, err)
	require.Equal(t, auth.NonEmployee, profile.Kind)
}

// Bila ADA sumber yang putus dan tidak satu pun menerima, yang dilaporkan adalah
// "sistem tidak dapat dihubungi" — bukan "kata sandi salah". Kita memang tidak tahu
// apakah kredensialnya benar, dan menyuruh pengguna mengetik ulang sesuatu yang sudah
// benar hanya membuang waktunya.
func TestChainReportsOutageNotWrongCredential(t *testing.T) {
	hcq := &fakeSource{failure: auth.ErrIdentitySystemDown}
	local := &fakeSource{failure: auth.ErrWrongCredential}

	_, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrIdentitySystemDown)
	require.NotErrorIs(t, err, auth.ErrWrongCredential)
}

// Bila seluruh sumber hidup dan semuanya menolak, barulah kredensialnya memang salah.
func TestChainReportsWrongCredentialWhenAllUp(t *testing.T) {
	hcq := &fakeSource{failure: auth.ErrWrongCredential}
	local := &fakeSource{failure: auth.ErrWrongCredential}

	_, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrWrongCredential)
	require.NotErrorIs(t, err, auth.ErrIdentitySystemDown)
}

// Akun yang sengaja dinonaktifkan menghentikan rantai: mencoba sumber berikutnya
// untuknya tidak masuk akal.
func TestChainStopsOnInactiveUser(t *testing.T) {
	hcq := &fakeSource{failure: auth.ErrUserInactive}
	local := &fakeSource{profile: nonEmployeeProfile()}

	_, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorIs(t, err, auth.ErrUserInactive)
	require.Zero(t, local.called)
}

// Galat yang tidak dikenali — misalnya profil tidak lengkap dari sumber yang menjawab
// berhasil — tidak boleh tersamarkan menjadi "kredensial salah".
func TestChainDoesNotSwallowUnknownFailure(t *testing.T) {
	gap := &auth.IncompleteProfileError{EmptyFields: []string{"nama"}}
	hcq := &fakeSource{failure: gap}
	local := &fakeSource{profile: nonEmployeeProfile()}

	_, err := chain(t, hcq, local).Verify(context.Background(), auth.Credential{})
	require.ErrorAs(t, err, &gap)
	require.Zero(t, local.called)
}

func TestChainRejectsEmptyChain(t *testing.T) {
	_, err := provider.NewChain()
	require.Error(t, err)

	_, err = provider.NewChain(provider.ChainLink{Name: "kosong"})
	require.ErrorContains(t, err, "tanpa sumber identitas")
}
