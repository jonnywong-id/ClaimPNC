package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/logging"
)

func konfPengembangan() config.Konfigurasi {
	return config.Konfigurasi{
		Lingkungan:       config.Pengembangan,
		AdapterIdentitas: config.AdapterIdentitasTiruan,
		Penyimpanan:      config.PenyimpananMemori,
		PortalUtama:      "ASM",
		Sesi:             config.Sesi{MasaBerlaku: 30 * time.Minute},
	}
}

// Dua adapter pengembangan hanya boleh hidup di luar produksi, dan penolakannya ada di
// kode — bukan pada nilai konfigurasi yang dapat dibalik seseorang.
func TestAdapterPengembanganMenolakProduksi(t *testing.T) {
	konf := konfPengembangan()
	konf.Lingkungan = config.Produksi

	t.Run("provider identitas tiruan", func(t *testing.T) {
		_, err := rakitIdentitas(konf, true, nil)
		require.ErrorIs(t, err, provider.ErrTiruanDiProduksi)
	})

	t.Run("penyimpanan memori", func(t *testing.T) {
		// Sesi di memori satu instans tidak akan dikenali instans kedua di belakang
		// load balancer — pelanggaran langsung terhadap tuntutan stateless (D-27).
		_, err := rakitPenyimpanan(konf, true, logging.Baru(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "menolak berjalan di lingkungan produksi")
	})

	t.Run("perakitan seluruh modul ikut gagal", func(t *testing.T) {
		_, err := rakit(konf, logging.Baru(0))
		require.Error(t, err)
	})
}

func TestPerakitanBerjalanDiLuarProduksi(t *testing.T) {
	hasil, err := rakit(konfPengembangan(), logging.Baru(0))
	require.NoError(t, err)
	require.NotNil(t, hasil.auth)
	require.NotNil(t, hasil.portal)
	require.NotNil(t, hasil.aliasSiap)
	hasil.tutup()
}

// Provider HCQ menuntut KONEKSI basis data — tetapi bukan tabel CPNC_. Alamat layanannya dibaca dari
// POOLDATA.GCNM_CONNECT_REST dan daftar login non-karyawan dari POOLDATA.M_LOGIN_PNC.
// Menyalakannya tanpa Oracle harus gagal dengan pesan yang menyebut sebabnya.
func TestIdentitasNyataMenuntutOracle(t *testing.T) {
	konf := konfPengembangan()
	konf.AdapterIdentitas = config.AdapterIdentitasNyata

	_, err := rakitIdentitas(konf, false, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "koneksi basis data")
}

// Portal yang variabel koneksinya belum lengkap dilewati, bukan membuat aplikasi gagal
// start: pengisian kredensial tiap entitas berjalan bertahap.
func TestPortalBelumLengkapDilewati(t *testing.T) {
	konf := konfPengembangan()
	konf.Portal = map[string]config.Basisdata{
		"ASM": {Alias: "ASM", Host: "h", Service: "s", Pengguna: "u", KataSandi: "p", Port: 1521},
		"ASI": {Alias: "ASI", Host: "", Service: "", Pengguna: "", KataSandi: ""},
	}

	parameter := parameterPortal(konf)
	require.Len(t, parameter, 1)
	require.Equal(t, "ASM", parameter[0].Alias)
}

// Koneksi Oracle dan penyimpanan sesi adalah dua hal berbeda, dan memisahkannya penting:
// integrasi HCC/HCQ dapat dicoba lewat layar SEBELUM migrasi 0001 dijalankan DBA.
func TestKoneksiOracleTerpisahDariPenyimpananSesi(t *testing.T) {
	kasus := []struct {
		nama        string
		penyimpanan string
		adapter     string
		butuh       bool
	}{
		{"keduanya tiruan", config.PenyimpananMemori, config.AdapterIdentitasTiruan, false},
		{"identitas nyata, sesi di memori", config.PenyimpananMemori, config.AdapterIdentitasNyata, true},
		{"sesi di Oracle, identitas tiruan", config.PenyimpananOracle, config.AdapterIdentitasTiruan, true},
		{"keduanya nyata", config.PenyimpananOracle, config.AdapterIdentitasNyata, true},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			konf := konfPengembangan()
			konf.Penyimpanan = k.penyimpanan
			konf.AdapterIdentitas = k.adapter
			require.Equal(t, k.butuh, butuhOracle(konf))
		})
	}
}

// Modul Master Status Progres ikut terpasang saat perakitan, dan penyimpanan tanpa
// basis data pun tetap MENOLAK portal selain portal utama.
//
// Penolakan itu yang membuat perilaku pengembangan sama dengan produksi: memilih
// entitas yang koneksinya belum hidup menghasilkan galat di keduanya, bukan diam-diam
// dilayani basis data entitas lain (R-20).
func TestModulStatusProgresTerpasangDanPortalLainDitolak(t *testing.T) {
	hasil, err := rakit(konfPengembangan(), logging.Baru(0))
	require.NoError(t, err)
	t.Cleanup(hasil.tutup)
	require.NotNil(t, hasil.statusProgres)

	// Portal utama dilayani.
	require.NoError(t, hasil.statusProgres.PastikanPortalSiap("ASM"))

	// Entitas lain — dan portal yang tidak disebut sama sekali — ditolak.
	for _, alias := range []string{"ASI", "SMAS", "TIDAKADA", ""} {
		require.Error(t, hasil.statusProgres.PastikanPortalSiap(alias),
			"portal %q tidak boleh dilayani tanpa koneksi basis datanya sendiri", alias)
	}
}

// Penyimpanan di memori dipakai kembali antarpermintaan, bukan dibuat ulang.
//
// Bila dibuat ulang, penambahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab yang terlihat.
func TestPenyimpananMemoriDipakaiKembali(t *testing.T) {
	pemilih := pemilihStatusProgresMemori("ASM")

	pertama, err := pemilih("ASM")
	require.NoError(t, err)
	kedua, err := pemilih("asm") // huruf kecil harus menunjuk penyimpanan yang sama
	require.NoError(t, err)
	require.Same(t, pertama, kedua)
}
