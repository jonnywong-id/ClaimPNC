package provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
	"claim-pnc/internal/auth/provider"
)

// sidikNyata adalah isi kolom HASH_PASSWORD pada baris contoh
// POOLDATA.M_LOGIN_PNC yang diekspor Work Owner (Database/m_login_pnc.csv).
const sidikNyata = "A665A45920422F9D417E4867EFDC4FB8A04A1F3FFF1FA07E998E86F7F7A27AE3"

// daftarLoginTiruan meniru POOLDATA.M_LOGIN_PNC: ia hanya menjawab bila ketiga syarat
// terpenuhi sekaligus — login_id cocok, sidik cocok, dan baris aktif.
type daftarLoginTiruan struct {
	loginID   string
	loginNama string
	sidik     string
	galat     error

	sidikDiminta string
}

func (d *daftarLoginTiruan) CariAktif(_ context.Context, loginID, sidik string) (provider.LoginLokal, error) {
	d.sidikDiminta = sidik
	if d.galat != nil {
		return provider.LoginLokal{}, d.galat
	}
	if loginID != d.loginID || sidik != d.sidik {
		return provider.LoginLokal{}, provider.ErrLoginTidakCocok
	}
	return provider.LoginLokal{LoginID: d.loginID, LoginNama: d.loginNama}, nil
}

// Sidik yang dihitung aplikasi harus sama persis dengan yang sudah tersimpan di kolom
// HASH_PASSWORD. Uji ini memakai nilai nyata dari export, bukan nilai karangan — kalau
// skemanya salah, seluruh pengguna non-karyawan tidak akan pernah bisa masuk.
func TestSidikKataSandiCocokDenganDataNyata(t *testing.T) {
	require.Equal(t, sidikNyata, provider.SidikKataSandi("123"))
}

func TestSidikKataSandiHurufBesarDanPanjangTetap(t *testing.T) {
	sidik := provider.SidikKataSandi("apa saja")
	require.Len(t, sidik, 64, "SHA-256 heksadesimal")
	require.Equal(t, sidik, provider.SidikKataSandi("apa saja"), "sidik harus tetap")
	require.NotEqual(t, sidik, provider.SidikKataSandi("apa saj"))
	for _, huruf := range sidik {
		require.NotContains(t, "abcdef", string(huruf),
			"kolom HASH_PASSWORD menyimpan heksadesimal huruf besar")
	}
}

func TestLokalMenerimaLoginYangCocok(t *testing.T) {
	daftar := &daftarLoginTiruan{loginID: "JONNY", loginNama: "JONNY WONG", sidik: sidikNyata}
	lokal, err := provider.LokalBaru(daftar)
	require.NoError(t, err)

	profil, err := lokal.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNY", KataSandi: "123"})
	require.NoError(t, err)

	require.Equal(t, "JONNY", profil.Identitas)
	require.Equal(t, "JONNY WONG", profil.Nama)
	require.Equal(t, auth.NonKaryawan, profil.Jenis, "broker dan surveyor independen bukan karyawan")
	require.Equal(t, "JONNY", profil.Login)
	require.Empty(t, profil.Email, "M_LOGIN_PNC tidak memuat email")
	require.Empty(t, profil.Cabang, "M_LOGIN_PNC tidak memuat cabang")

	// Yang dikirim ke basis data adalah sidiknya, bukan kata sandinya.
	require.Equal(t, sidikNyata, daftar.sidikDiminta)
	require.NotContains(t, daftar.sidikDiminta, "123")
}

func TestLokalMenolakSandiSalah(t *testing.T) {
	daftar := &daftarLoginTiruan{loginID: "JONNY", loginNama: "JONNY WONG", sidik: sidikNyata}
	lokal, err := provider.LokalBaru(daftar)
	require.NoError(t, err)

	_, err = lokal.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNY", KataSandi: "bukan 123"})
	require.ErrorIs(t, err, auth.ErrKredensialSalah)
}

// Login tidak ada, sandi salah, dan akun tidak aktif ketiganya dijawab sama: kueri
// menggabungkan ketiga syarat sehingga aplikasi tidak dapat — dan tidak boleh —
// membedakannya.
func TestLokalTidakMembocorkanKeberadaanAkun(t *testing.T) {
	daftar := &daftarLoginTiruan{loginID: "JONNY", loginNama: "JONNY WONG", sidik: sidikNyata}
	lokal, err := provider.LokalBaru(daftar)
	require.NoError(t, err)

	_, errTidakAda := lokal.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "TIDAKADA", KataSandi: "123"})
	_, errSandiSalah := lokal.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNY", KataSandi: "salah"})

	require.ErrorIs(t, errTidakAda, auth.ErrKredensialSalah)
	require.Equal(t, errTidakAda.Error(), errSandiSalah.Error())
}

func TestLokalMeneruskanGalatBasisData(t *testing.T) {
	galatDB := errors.New("koneksi terputus")
	daftar := &daftarLoginTiruan{galat: galatDB}
	lokal, err := provider.LokalBaru(daftar)
	require.NoError(t, err)

	_, err = lokal.Verifikasi(context.Background(),
		auth.Kredensial{NamaPengguna: "JONNY", KataSandi: "123"})
	require.ErrorIs(t, err, galatDB,
		"galat basis data tidak boleh tersamarkan menjadi kredensial salah")
}
