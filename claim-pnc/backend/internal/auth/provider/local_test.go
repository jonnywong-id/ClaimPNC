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
	return provider.LocalLogin{
		LoginID:   d.loginID,
		LoginName: d.loginName,
	}, nil
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
	require.Empty(t, profile.Branch, "M_LOGIN_PNC tidak memuat NAMA cabang")
	require.Empty(t, profile.DetailBranchCode,
		"tanpa baris di master HRD, cabang tetap kosong")

	// Yang dikirim ke basis data adalah sidiknya, bukan kata sandinya.
	require.Equal(t, realFingerprint, list.wantedFingerprint)
	require.NotContains(t, list.wantedFingerprint, "123")
}

// Jalur login lokal TIDAK PERNAH membawa cabang, dan itu keputusan Work Owner.
//
// Uji ini dulu berlawanan arah: ia memastikan cabang dari master HRD lokal
// (`POOLDATA.V_HRD_M_MST.LCA_ID`) ikut terbawa ke profil. Sumber itu DICABUT pada
// 2026-10-11 — ia pilihan sendiri, nol kemunculan di export Pega, dan hanya 68% sepakat
// dengan sumber Pega yang sebenarnya (`LST_USER_ASURANSI.CAB_ID`).
//
// Work Owner menetapkan: cabang hanya datang dari layanan HCC, dan pengguna tanpa data
// cabang memang tidak berhak membuka layar berbatas cabang. Uji ini menjaga ketetapan itu —
// bila kelak ada yang menambahkan sumber cabang lain di jalur ini, ia gagal.
func TestLocalNeverCarriesABranch(t *testing.T) {
	list := &fakeLoginList{
		loginID:     "JONNY",
		loginName:   "JONNY WONG",
		fingerprint: realFingerprint,
	}
	local, err := provider.NewLocal(list)
	require.NoError(t, err)

	profile, err := local.Verify(context.Background(),
		auth.Credential{Username: "JONNY", Password: "123"})
	require.NoError(t, err)

	// Ketiga field cabang kosong. `DetailBranchCode` adalah yang dibaca layar OS Claim per
	// Cabang; `BranchCode` punya pembaca lain yang MENULISKANNYA ke basis data (modul
	// Archive Dokumen Klaim, kolom KODECABANG), jadi ia tidak boleh terisi diam-diam.
	require.Empty(t, profile.DetailBranchCode,
		"cabang hanya dari HCC — jalur lokal tidak boleh mengisinya dari tabel mana pun")
	require.Empty(t, profile.BranchCode)
	require.Empty(t, profile.Branch)
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
