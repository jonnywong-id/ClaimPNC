package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// realFingerprint adalah isi kolom HASH_PASSWORD pada baris contoh
// POOLDATA.M_LOGIN_PNC yang diekspor Work Owner (Database/m_login_pnc.csv).
const realFingerprint = "A665A45920422F9D417E4867EFDC4FB8A04A1F3FFF1FA07E998E86F7F7A27AE3"

// fakeLoginList meniru POOLDATA.M_LOGIN_PNC: ia hanya menjawab bila ketiga syarat
// terpenuhi sekaligus — login_id cocok, sidik cocok, dan baris aktif.
type fakeLoginList struct {
	loginID     string
	loginName   string
	fingerprint string
	issues      error

	wantedFingerprint string
}

func (d *fakeLoginList) FindActive(_ context.Context, loginID, fingerprint string) (provider.LocalLogin, error) {
	d.wantedFingerprint = fingerprint
	if d.issues != nil {
		return provider.LocalLogin{}, d.issues
	}
	if loginID != d.loginID || fingerprint != d.fingerprint {
		return provider.LocalLogin{}, provider.ErrLoginMismatch
	}
	return provider.LocalLogin{LoginID: d.loginID, LoginName: d.loginName}, nil
}

// Sidik yang dihitung aplikasi harus sama persis dengan yang sudah tersimpan di kolom
// HASH_PASSWORD. Uji ini memakai nilai nyata dari export, bukan nilai karangan — kalau
// skemanya salah, seluruh pengguna non-karyawan tidak akan pernah bisa masuk.
func TestPasswordDigestMatchesRealData(t *testing.T) {
	require.Equal(t, realFingerprint, provider.PasswordDigest("123"))
}

func TestPasswordDigestIsUppercaseAndFixedLength(t *testing.T) {
	fingerprint := provider.PasswordDigest("apa saja")
	require.Len(t, fingerprint, 64, "SHA-256 heksadesimal")
	require.Equal(t, fingerprint, provider.PasswordDigest("apa saja"), "sidik harus tetap")
	require.NotEqual(t, fingerprint, provider.PasswordDigest("apa saj"))
	for _, huruf := range fingerprint {
		require.NotContains(t, "abcdef", string(huruf),
			"kolom HASH_PASSWORD menyimpan heksadesimal huruf besar")
	}
}

func TestLocalAcceptsMatchingLogin(t *testing.T) {
	list := &fakeLoginList{loginID: "JONNY", loginName: "JONNY WONG", fingerprint: realFingerprint}
	local, err := provider.NewLocal(list)
	require.NoError(t, err)

	profile, err := local.Verify(context.Background(),
		auth.Credential{Username: "JONNY", Password: "123"})
	require.NoError(t, err)

	require.Equal(t, "JONNY", profile.Identity)
	require.Equal(t, "JONNY WONG", profile.Name)
	require.Equal(t, auth.NonEmployee, profile.Kind, "broker dan surveyor independen bukan karyawan")
	require.Equal(t, "JONNY", profile.Login)
	require.Empty(t, profile.Email, "M_LOGIN_PNC tidak memuat email")
	require.Empty(t, profile.Branch, "M_LOGIN_PNC tidak memuat cabang")

	// Yang dikirim ke basis data adalah sidiknya, bukan kata sandinya.
	require.Equal(t, realFingerprint, list.wantedFingerprint)
	require.NotContains(t, list.wantedFingerprint, "123")
}

func TestLocalRejectsWrongPassword(t *testing.T) {
	list := &fakeLoginList{loginID: "JONNY", loginName: "JONNY WONG", fingerprint: realFingerprint}
	local, err := provider.NewLocal(list)
	require.NoError(t, err)

	_, err = local.Verify(context.Background(),
		auth.Credential{Username: "JONNY", Password: "bukan 123"})
	require.ErrorIs(t, err, auth.ErrWrongCredential)
}

// Login tidak ada, sandi salah, dan akun tidak aktif ketiganya dijawab sama: kueri
// menggabungkan ketiga syarat sehingga aplikasi tidak dapat — dan tidak boleh —
// membedakannya.
func TestLocalDoesNotLeakAccountExistence(t *testing.T) {
	list := &fakeLoginList{loginID: "JONNY", loginName: "JONNY WONG", fingerprint: realFingerprint}
	local, err := provider.NewLocal(list)
	require.NoError(t, err)

	_, errTidakAda := local.Verify(context.Background(),
		auth.Credential{Username: "TIDAKADA", Password: "123"})
	_, errWrongPassword := local.Verify(context.Background(),
		auth.Credential{Username: "JONNY", Password: "salah"})

	require.ErrorIs(t, errTidakAda, auth.ErrWrongCredential)
	require.Equal(t, errTidakAda.Error(), errWrongPassword.Error())
}

func TestLocalForwardsDatabaseError(t *testing.T) {
	dbErrs := errors.New("koneksi terputus")
	list := &fakeLoginList{issues: dbErrs}
	local, err := provider.NewLocal(list)
	require.NoError(t, err)

	_, err = local.Verify(context.Background(),
		auth.Credential{Username: "JONNY", Password: "123"})
	require.ErrorIs(t, err, dbErrs,
		"galat basis data tidak boleh tersamarkan menjadi kredensial salah")
}
