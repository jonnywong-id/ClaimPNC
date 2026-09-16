package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth"
)

func profilLengkap() auth.Profil {
	return auth.Profil{
		Identitas: "90000001",
		Nama:      "Contoh Pengguna",
		Jenis:     auth.Karyawan,
		Login:     "contoh",
		Email:     "contoh@example.invalid",
	}
}

func TestProfilLengkapDiterima(t *testing.T) {
	require.NoError(t, profilLengkap().Periksa())
}

// Field yang kosong ditolak sebagai galat, bukan diteruskan diam-diam
// (TKT-F3-001, acceptance criteria terakhir).
func TestProfilDenganFieldKosongDitolak(t *testing.T) {
	kasus := map[string]func(*auth.Profil){
		"identitas kosong":    func(p *auth.Profil) { p.Identitas = "" },
		"nama kosong":         func(p *auth.Profil) { p.Nama = "" },
		"jenis tidak dikenal": func(p *auth.Profil) { p.Jenis = "" },
	}
	for nama, kosongkan := range kasus {
		t.Run(nama, func(t *testing.T) {
			p := profilLengkap()
			kosongkan(&p)
			err := p.Periksa()
			require.Error(t, err)

			var galat *auth.GalatProfilTidakLengkap
			require.ErrorAs(t, err, &galat)
			require.Len(t, galat.FieldKosong, 1)
		})
	}
}

func TestKredensialKosongTidakLengkap(t *testing.T) {
	require.False(t, auth.Kredensial{}.Lengkap())
	require.False(t, auth.Kredensial{NamaPengguna: " ", KataSandi: "x"}.Lengkap())
	require.False(t, auth.Kredensial{NamaPengguna: "budi"}.Lengkap())
	require.True(t, auth.Kredensial{NamaPengguna: "budi", KataSandi: "x"}.Lengkap())
}

// Sidik token bersifat satu arah dan panjangnya tetap; token mentahnya tidak muncul
// di dalamnya.
func TestSidikTokenSatuArah(t *testing.T) {
	token := auth.Token("token-contoh-yang-panjang")
	sidik := token.Sidik()

	require.Len(t, sidik, 64)
	require.NotContains(t, sidik, string(token))
	require.Equal(t, sidik, token.Sidik(), "sidik harus tetap untuk token yang sama")
	require.NotEqual(t, sidik, auth.Token("token-contoh-yang-panjan").Sidik())
}

func TestTerbitkanTokenMenghasilkanNilaiBerbeda(t *testing.T) {
	pertama, err := auth.TerbitkanToken(nil)
	require.NoError(t, err)
	kedua, err := auth.TerbitkanToken(nil)
	require.NoError(t, err)
	require.NotEqual(t, pertama, kedua)
}

// Sesi yang dicabut terbaca sebagai dicabut walau kebetulan juga sudah kedaluwarsa —
// administrator yang mencabut harus melihat alasan yang benar.
func TestPencabutanDidahulukanDaripadaKedaluwarsa(t *testing.T) {
	sekarang := time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
	dicabut := sekarang.Add(-time.Hour)

	s := auth.Sesi{
		BerlakuSampai: sekarang.Add(-time.Minute),
		DicabutPada:   &dicabut,
	}
	require.ErrorIs(t, s.Periksa(sekarang), auth.ErrSesiDicabut)
	require.False(t, s.Aktif(sekarang))
	require.Zero(t, s.SisaBerlaku(sekarang))
}

func TestSesiAktifSelamaBelumLewatBatas(t *testing.T) {
	sekarang := time.Date(2026, 9, 15, 3, 0, 0, 0, time.UTC)
	s := auth.Sesi{BerlakuSampai: sekarang.Add(10 * time.Minute)}

	require.NoError(t, s.Periksa(sekarang))
	require.True(t, s.Aktif(sekarang))
	require.Equal(t, 10*time.Minute, s.SisaBerlaku(sekarang))

	require.ErrorIs(t, s.Periksa(s.BerlakuSampai), auth.ErrSesiKedaluwarsa,
		"tepat pada batas berlaku, sesi sudah tidak sah")
}
