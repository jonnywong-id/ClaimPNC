package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/config"
)

// cleanEnv memastikan satu uji tidak mewarisi variabel dari uji lain.
func cleanEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"APP_ENV", "APP_ALAMAT", "IDENTITAS_ADAPTER", "PENYIMPANAN", "PORTAL_UTAMA",
		"SESI_MASA_BERLAKU", "HCQ_LOGIN_USER", "HCQ_LOGIN_PASSWORD",
	} {
		t.Setenv(name, "")
		require.NoError(t, os.Unsetenv(name))
	}
	for _, rows := range os.Environ() {
		if len(rows) > 9 && rows[:9] == "POOLDATA_" {
			name := rows[:index(rows, '=')]
			require.NoError(t, os.Unsetenv(name))
		}
		// Koneksi kedua ikut dibersihkan. Tanpa ini, satu uji yang memasangnya akan
		// mewariskannya ke uji berikutnya — dan uji yang menuntut "tanpa koneksi kedua"
		// akan lulus atau gagal menurut urutan jalannya.
		if len(rows) > 6 && rows[:6] == "ANEKA_" {
			name := rows[:index(rows, '=')]
			require.NoError(t, os.Unsetenv(name))
		}
	}
}

func index(s string, c byte) int {
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
func TestPortalAliasFoundFromEnvironment(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	setPortal(t, "ASI")
	setPortal(t, "SPKS")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"ASM", "ASI", "SPKS"}, cfg.AvailableAliases())
	require.Equal(t, "host-ASI", cfg.Portal["ASI"].Host)
	require.Equal(t, 1521, cfg.Portal["ASM"].Port, "port punya nilai baku")
}

// Portal yang variabelnya belum lengkap terbaca tetapi ditandai belum siap — pengisian
// kredensial tiap entitas berjalan bertahap.
func TestIncompletePortalReadButMarkedUnavailable(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	t.Setenv("POOLDATA_SMI_HOST", "host-SMI") // sengaja hanya HOST

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Equal(t, []string{"ASM"}, cfg.AvailableAliases())

	smi := cfg.Portal["SMI"]
	require.False(t, smi.Complete())
	require.ElementsMatch(t,
		[]string{"POOLDATA_SMI_PENGGUNA", "POOLDATA_SMI_SANDI", "POOLDATA_SMI_SERVICE"},
		smi.Missing(), "pesannya menyebut variabel mana yang missing")
}

// Portal utama adalah satu-satunya yang wajib lengkap: basis datanya melayani daftar
// portal, alamat HCQ, login non-karyawan, dan tabel sesi.
func TestPrimaryPortalMustBeComplete(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	t.Setenv("POOLDATA_ASM_HOST", "host")

	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "portal utama")
	require.Contains(t, err.Error(), "POOLDATA_ASM_SANDI")
}

func TestPrimaryPortalMissingEntirely(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "SMAS")
	setPortal(t, "ASM")

	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), `PORTAL_UTAMA "SMAS"`)
}

func TestHCQCredentialRequiredWhenRealAdapter(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "memori")
	t.Setenv("IDENTITAS_ADAPTER", "hcq")

	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "HCQ_LOGIN_USER")
	require.Contains(t, err.Error(), "HCQ_LOGIN_PASSWORD")
}

// Rahasia tidak pernah ikut di ringkasan yang ditulis ke log.
func TestSummaryHoldsNoSecrets(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	t.Setenv("IDENTITAS_ADAPTER", "hcq")
	t.Setenv("HCQ_LOGIN_USER", "pengguna-hcq")
	t.Setenv("HCQ_LOGIN_PASSWORD", "sandi-hcq-rahasia")
	setPortal(t, "ASM")

	cfg, err := config.Load()
	require.NoError(t, err)

	ringkas := cfg.Summary()
	for key, value := range ringkas {
		require.NotEqual(t, "sandi-ASM", value, "kata sandi basis data bocor lewat %q", key)
		require.NotEqual(t, "sandi-hcq-rahasia", value, "kata sandi HCQ bocor lewat %q", key)
	}
	require.Equal(t, true, ringkas["hcq_sandi_diisi"], "yang dilaporkan hanya terisi atau tidak")
}

func TestLoadEnvFile(t *testing.T) {
	cleanEnv(t)
	filePath := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"# komentar diabaikan\n" +
		"\n" +
		"APP_ALAMAT=:9999\n" +
		`POOLDATA_ASM_SANDI="sandi dengan spasi"` + "\n" +
		"PORTAL_UTAMA=ASI # komentar di belakang\n"
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o600))

	require.NoError(t, config.LoadEnvFile(filePath))
	require.Equal(t, ":9999", os.Getenv("APP_ALAMAT"))
	require.Equal(t, "sandi dengan spasi", os.Getenv("POOLDATA_ASM_SANDI"), "kutip dilepas")
	require.Equal(t, "ASI", os.Getenv("PORTAL_UTAMA"), "komentar di belakang nilai dibuang")
}

// Variabel yang sudah ada di lingkungan menang atas berkas: satu perintah harus dapat
// menimpa satu nilai tanpa menyunting .env.
func TestEnvFileDoesNotOverrideEnvironment(t *testing.T) {
	cleanEnv(t)
	t.Setenv("APP_ALAMAT", ":7777")

	filePath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(filePath, []byte("APP_ALAMAT=:9999\n"), 0o600))
	require.NoError(t, config.LoadEnvFile(filePath))

	require.Equal(t, ":7777", os.Getenv("APP_ALAMAT"))
}

// Berkas .env yang tidak ada bukan galat: di server nilai datang dari lingkungan
// proses, bukan dari berkas (ADR-0025).
func TestMissingEnvFileIsNotAnError(t *testing.T) {
	require.NoError(t, config.LoadEnvFile(filepath.Join(t.TempDir(), "tidak-ada.env")))
}

func TestMalformedEnvFileRejected(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(filePath, []byte("BARIS TANPA SAMA DENGAN\n"), 0o600))

	err := config.LoadEnvFile(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "baris 1")
}

// Clone yang baru harus dapat dijalankan dengan `go run` tanpa satu pun kredensial.
// Ini yang gagal pada 2026-09-16 dan menjadi alasan nilai baku PENYIMPANAN diubah.
func TestStartsWithNoEnvAtAll(t *testing.T) {
	cleanEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err, "go run pada clone bersih tidak boleh gagal karena konfigurasi")
	require.Equal(t, config.StorageMemory, cfg.Storage)
	require.Equal(t, config.Development, cfg.Environment)
	require.Equal(t, "ASM", cfg.PrimaryPortal)
}

// Staging dan produksi tetap menuntut Oracle: nilai baku yang memudahkan pengembangan
// tidak boleh ikut memudahkan produksi berjalan tanpa basis data.
func TestStagingAndProductionStillRequireOracle(t *testing.T) {
	for _, environment := range []string{"staging", "production"} {
		t.Run(environment, func(t *testing.T) {
			cleanEnv(t)
			t.Setenv("APP_ENV", environment)

			_, err := config.Load()
			require.Error(t, err, "tanpa kredensial portal, %s harus gagal start", environment)
			require.Contains(t, err.Error(), "POOLDATA_ASM_")
		})
	}
}

// Galat konfigurasi harus menyebut CARA memperbaikinya, bukan hanya apa yang missing.
// Galat saat start dibaca orang yang sedang terhenti.
func TestConfigErrorNamesHowToFixIt(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")

	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), ".env.example", "menyebut berkas contoh yang harus disalin")
	require.Contains(t, err.Error(), "PENYIMPANAN=memori", "menyebut jalan keluar tanpa basis data")
	require.Contains(t, err.Error(), "direktori kerja",
		"menyebut jebakan .env dibaca relatif terhadap direktori kerja")
}

// setAneka memasang koneksi KEDUA milik sebuah portal — pengganti DB Link (`R-03`),
// keputusan Work Owner 2026-09-24.
func setAneka(t *testing.T, alias string) {
	t.Helper()
	t.Setenv("ANEKA_"+alias+"_HOST", "aneka-host-"+alias)
	t.Setenv("ANEKA_"+alias+"_SERVICE", "aneka-svc-"+alias)
	t.Setenv("ANEKA_"+alias+"_PENGGUNA", "aneka-user-"+alias)
	t.Setenv("ANEKA_"+alias+"_SANDI", "aneka-sandi-"+alias)
}

// Koneksi kedua ditemukan dengan cara yang sama dengan portal: memindai lingkungan.
// Menambah entitas tetap cukup dengan menambah baris .env, tanpa menyentuh kode.
func TestKoneksiKeduaDitemukanDariLingkungan(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	setPortal(t, "ASI")
	setAneka(t, "ASM")

	cfg, err := config.Load()
	require.NoError(t, err)

	require.Equal(t, "aneka-host-ASM", cfg.Aneka["ASM"].Host)
	require.Equal(t, 1521, cfg.Aneka["ASM"].Port, "port punya nilai baku yang sama dengan portal")
	require.True(t, cfg.Aneka["ASM"].Complete())

	// Portal yang belum punya blok ANEKA bukan galat — ia hanya belum punya koneksi
	// kedua, dan laporan yang membutuhkannya mengosongkan kolomnya.
	_, ada := cfg.Aneka["ASI"]
	require.False(t, ada)
}

// Seluruh portal boleh tanpa koneksi kedua. Itu keadaan yang sah, bukan konfigurasi
// yang belum selesai — dan aplikasi tetap harus start.
func TestTanpaKoneksiKeduaSamaSekaliTetapSah(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")

	cfg, err := config.Load()
	require.NoError(t, err)
	require.Empty(t, cfg.Aneka)
}

// Koneksi kedua selalu MILIK sebuah portal. Alias yang tidak punya blok POOLDATA hampir
// pasti salah ketik, dan tanpa pemeriksaan ini ia lolos diam-diam: bloknya terbaca,
// tidak pernah dipakai, dan laporannya tetap kosong seolah belum diisi.
func TestKoneksiKeduaTanpaPortalDitolak(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	setAneka(t, "ASMM") // salah ketik

	_, err := config.Load()
	require.Error(t, err)
	require.Contains(t, err.Error(), "ANEKA_ASMM_*")
	require.Contains(t, err.Error(), "POOLDATA_ASMM_*")
}

// Pesan "yang missing" harus menyebut NAMA VARIABEL YANG SEBENARNYA. Bila ia menyebut
// POOLDATA_ padahal yang kurang ANEKA_, operator akan memperbaiki baris yang sudah benar.
func TestKoneksiKeduaYangBelumLengkapMenyebutVariabelnyaSendiri(t *testing.T) {
	cleanEnv(t)
	t.Setenv("PENYIMPANAN", "oracle")
	t.Setenv("PORTAL_UTAMA", "ASM")
	setPortal(t, "ASM")
	t.Setenv("ANEKA_ASM_HOST", "aneka-host")

	cfg, err := config.Load()
	require.NoError(t, err, "koneksi kedua yang belum lengkap BUKAN galat start")

	second := cfg.Aneka["ASM"]
	require.False(t, second.Complete())
	require.Contains(t, second.Missing(), "ANEKA_ASM_SERVICE")
	require.Contains(t, second.Missing(), "ANEKA_ASM_PENGGUNA")
	require.Contains(t, second.Missing(), "ANEKA_ASM_SANDI")
	require.NotContains(t, second.Missing(), "POOLDATA_ASM_SERVICE")
}
