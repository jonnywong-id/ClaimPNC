package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/config"
)

// bersihkanEnv memastikan satu uji tidak mewarisi variabel dari uji lain.
func bersihkanEnv(t *testing.T) {
	t.Helper()
	for _, nama := range []string{
		"APP_ENV", "APP_ALAMAT", "IDENTITAS_ADAPTER", "PENYIMPANAN", "PORTAL_UTAMA",
		"SESI_MASA_BERLAKU", "HCQ_LOGIN_USER", "HCQ_LOGIN_PASSWORD",
	} {
		t.Setenv(nama, "")
		require.NoError(t, os.Unsetenv(nama))
	}
	for _, baris := range os.Environ() {
		if len(baris) > 9 && baris[:9] == "POOLDATA_" {
			nama := baris[:indeks(baris, '=')]
			require.NoError(t, os.Unsetenv(nama))
		}
	}
}

func indeks(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return len(s)
}

func setPortal(t *testing.T, alias string) {
	t.Helper()
	t.Setenv("POOLDATA_"+alias+"_HOST", "host-"+alias)
	t.Setenv("POOLDATA_"+alias+"_SERVICE", "svc-"+alias)
	t.Setenv("POOLDATA_"+alias+"_PENGGUNA", "user-"+alias)
	t.Setenv("POOLDATA_"+alias+"_SANDI", "sandi-"+alias)
}

// Daftar portal ditemukan dengan memindai lingkungan, bukan dari daftar tetap di kode
// (ADR-0030: daftar portal adalah data). Menambah entitas cukup menambah lima baris
// .env.
func TestAliasPortalDitemukanDariLingkungan(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	setPortal(t, "ASI")
	setPortal(t, "SPKS")

	konf, err := config.Muat()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"ASM", "ASI", "SPKS"}, konf.AliasTersedia())
	require.Equal(t, "host-ASI", konf.Portal["ASI"].Host)
	require.Equal(t, 1521, konf.Portal["ASM"].Port, "port punya nilai baku")
}

// Portal yang variabelnya belum lengkap terbaca tetapi ditandai belum siap — pengisian
// kredensial tiap entitas berjalan bertahap.
func TestPortalBelumLengkapTerbacaTetapiTidakTersedia(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	t.Setenv("POOLDATA_SMI_HOST", "host-SMI") // sengaja hanya HOST

	konf, err := config.Muat()
	require.NoError(t, err)
	require.Equal(t, []string{"ASM"}, konf.AliasTersedia())

	smi := konf.Portal["SMI"]
	require.False(t, smi.Lengkap())
	require.ElementsMatch(t,
		[]string{"POOLDATA_SMI_PENGGUNA", "POOLDATA_SMI_SANDI", "POOLDATA_SMI_SERVICE"},
		smi.YangKurang(), "pesannya menyebut variabel mana yang kurang")
}

// Portal utama adalah satu-satunya yang wajib lengkap: basis datanya melayani daftar
// portal, alamat HCQ, login non-karyawan, dan tabel sesi.
func TestPortalUtamaWajibLengkap(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	t.Setenv("POOLDATA_ASM_HOST", "host")

	_, err := config.Muat()
	require.Error(t, err)
	require.Contains(t, err.Error(), "portal utama")
	require.Contains(t, err.Error(), "POOLDATA_ASM_SANDI")
}

func TestPortalUtamaTidakAdaSamaSekali(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "SMAS")
	setPortal(t, "ASM")

	_, err := config.Muat()
	require.Error(t, err)
	require.Contains(t, err.Error(), `PORTAL_UTAMA "SMAS"`)
}

func TestKredensialHCQWajibBilaAdapterNyata(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "memori")
	t.Setenv("IDENTITAS_ADAPTER", "hcq")

	_, err := config.Muat()
	require.Error(t, err)
	require.Contains(t, err.Error(), "HCQ_LOGIN_USER")
	require.Contains(t, err.Error(), "HCQ_LOGIN_PASSWORD")
}

// Rahasia tidak pernah ikut di ringkasan yang ditulis ke log.
func TestRingkasTidakMemuatRahasia(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	t.Setenv("IDENTITAS_ADAPTER", "hcq")
	t.Setenv("HCQ_LOGIN_USER", "pengguna-hcq")
	t.Setenv("HCQ_LOGIN_PASSWORD", "sandi-hcq-rahasia")
	setPortal(t, "ASM")

	konf, err := config.Muat()
	require.NoError(t, err)

	ringkas := konf.Ringkas()
	for kunci, nilai := range ringkas {
		require.NotEqual(t, "sandi-ASM", nilai, "kata sandi basis data bocor lewat %q", kunci)
		require.NotEqual(t, "sandi-hcq-rahasia", nilai, "kata sandi HCQ bocor lewat %q", kunci)
	}
	require.Equal(t, true, ringkas["hcq_sandi_diisi"], "yang dilaporkan hanya terisi atau tidak")
}

func TestMuatBerkasEnv(t *testing.T) {
	bersihkanEnv(t)
	jalur := filepath.Join(t.TempDir(), ".env")
	isi := "" +
		"# komentar diabaikan\n" +
		"\n" +
		"APP_ALAMAT=:9999\n" +
		`POOLDATA_ASM_SANDI="sandi dengan spasi"` + "\n" +
		"PORTAL_UTAMA=ASI # komentar di belakang\n"
	require.NoError(t, os.WriteFile(jalur, []byte(isi), 0o600))

	require.NoError(t, config.MuatBerkasEnv(jalur))
	require.Equal(t, ":9999", os.Getenv("APP_ALAMAT"))
	require.Equal(t, "sandi dengan spasi", os.Getenv("POOLDATA_ASM_SANDI"), "kutip dilepas")
	require.Equal(t, "ASI", os.Getenv("PORTAL_UTAMA"), "komentar di belakang nilai dibuang")
}

// Variabel yang sudah ada di lingkungan menang atas berkas: satu perintah harus dapat
// menimpa satu nilai tanpa menyunting .env.
func TestBerkasEnvTidakMenimpaLingkungan(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("APP_ALAMAT", ":7777")

	jalur := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(jalur, []byte("APP_ALAMAT=:9999\n"), 0o600))
	require.NoError(t, config.MuatBerkasEnv(jalur))

	require.Equal(t, ":7777", os.Getenv("APP_ALAMAT"))
}

// Berkas .env yang tidak ada bukan galat: di server nilai datang dari lingkungan
// proses, bukan dari berkas (ADR-0025).
func TestBerkasEnvTidakAdaBukanGalat(t *testing.T) {
	require.NoError(t, config.MuatBerkasEnv(filepath.Join(t.TempDir(), "tidak-ada.env")))
}

func TestBerkasEnvCacatDitolak(t *testing.T) {
	jalur := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(jalur, []byte("BARIS TANPA SAMA DENGAN\n"), 0o600))

	err := config.MuatBerkasEnv(jalur)
	require.Error(t, err)
	require.Contains(t, err.Error(), "baris 1")
}

// Clone yang baru harus dapat dijalankan dengan `go run` tanpa satu pun kredensial.
// Ini yang gagal pada 2026-09-16 dan menjadi alasan nilai baku PENYIMPANAN diubah.
func TestTanpaEnvSamaSekaliTetapDapatStart(t *testing.T) {
	bersihkanEnv(t)

	konf, err := config.Muat()
	require.NoError(t, err, "go run pada clone bersih tidak boleh gagal karena konfigurasi")
	require.Equal(t, config.PenyimpananMemori, konf.Penyimpanan)
	require.Equal(t, config.Pengembangan, konf.Lingkungan)
	require.Equal(t, "ASM", konf.PortalUtama)
}

// Staging dan produksi tetap menuntut Oracle: nilai baku yang memudahkan pengembangan
// tidak boleh ikut memudahkan produksi berjalan tanpa basis data.
func TestStagingDanProduksiTetapMenuntutOracle(t *testing.T) {
	for _, lingkungan := range []string{"staging", "production"} {
		t.Run(lingkungan, func(t *testing.T) {
			bersihkanEnv(t)
			t.Setenv("APP_ENV", lingkungan)

			_, err := config.Muat()
			require.Error(t, err, "tanpa kredensial portal, %s harus gagal start", lingkungan)
			require.Contains(t, err.Error(), "POOLDATA_ASM_")
		})
	}
}

// Galat konfigurasi harus menyebut CARA memperbaikinya, bukan hanya apa yang kurang.
// Galat saat start dibaca orang yang sedang terhenti.
func TestGalatKonfigurasiMenyebutCaraMemperbaiki(t *testing.T) {
	bersihkanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")

	_, err := config.Muat()
	require.Error(t, err)
	require.Contains(t, err.Error(), ".env.example", "menyebut berkas contoh yang harus disalin")
	require.Contains(t, err.Error(), "PENYIMPANAN=memori", "menyebut jalan keluar tanpa basis data")
	require.Contains(t, err.Error(), "direktori kerja",
		"menyebut jebakan .env dibaca relatif terhadap direktori kerja")
}
