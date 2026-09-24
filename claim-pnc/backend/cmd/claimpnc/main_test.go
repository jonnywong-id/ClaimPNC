package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/auth/provider"
	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/platform/logging"
)

func devConfig() config.Config {
	return config.Config{
		Environment:     config.Development,
		IdentityAdapter: config.IdentityAdapterFake,
		Storage:         config.StorageMemory,
		PrimaryPortal:   "ASM",
		Session:         config.Session{Lifetime: 30 * time.Minute},
	}
}

// Dua adapter pengembangan hanya boleh hidup di luar produksi, dan penolakannya ada di
// kode — bukan pada nilai konfigurasi yang dapat dibalik seseorang.
func TestDevelopmentAdapterRefusesProduction(t *testing.T) {
	cfg := devConfig()
	cfg.Environment = config.Production

	t.Run("provider identitas tiruan", func(t *testing.T) {
		_, err := buildIdentity(cfg, true, nil)
		require.ErrorIs(t, err, provider.ErrFakeInProduction)
	})

	t.Run("penyimpanan memori", func(t *testing.T) {
		// Session di memori satu instans tidak akan dikenali instans kedua di belakang
		// load balancer — pelanggaran langsung terhadap tuntutan stateless (D-27).
		_, err := buildStorage(cfg, true, logging.New(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "menolak berjalan di lingkungan produksi")
	})

	t.Run("perakitan seluruh modul ikut gagal", func(t *testing.T) {
		_, err := build(cfg, logging.New(0))
		require.Error(t, err)
	})
}

func TestAssemblyRunsOutsideProduction(t *testing.T) {
	result, err := build(devConfig(), logging.New(0))
	require.NoError(t, err)
	require.NotNil(t, result.auth)
	require.NotNil(t, result.portal)
	require.NotNil(t, result.readyAliases)
	result.close()
}

// Provider HCQ menuntut KONEKSI basis data — tetapi bukan tabel CPNC_. Alamat layanannya dibaca dari
// POOLDATA.GCNM_CONNECT_REST dan daftar login non-karyawan dari POOLDATA.M_LOGIN_PNC.
// Menyalakannya tanpa Oracle harus gagal dengan pesan yang menyebut sebabnya.
func TestRealIdentityAdapterRequiresOracle(t *testing.T) {
	cfg := devConfig()
	cfg.IdentityAdapter = config.IdentityAdapterHCQ

	_, err := buildIdentity(cfg, false, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "koneksi basis data")
}

// Portal yang variabel koneksinya belum lengkap dilewati, bukan membuat aplikasi gagal
// start: pengisian kredensial tiap entitas berjalan bertahap.
func TestIncompletePortalSkipped(t *testing.T) {
	cfg := devConfig()
	cfg.Portal = map[string]config.Database{
		"ASM": {Alias: "ASM", Host: "h", Service: "s", User: "u", Password: "p", Port: 1521},
		"ASI": {Alias: "ASI", Host: "", Service: "", User: "", Password: ""},
	}

	parameter := portalParameters(cfg)
	require.Len(t, parameter, 1)
	require.Equal(t, "ASM", parameter[0].Alias)
}

// Koneksi Oracle dan penyimpanan sesi adalah dua hal berbeda, dan memisahkannya penting:
// integrasi HCC/HCQ dapat dicoba lewat layar SEBELUM migrasi 0001 dijalankan DBA.
func TestOracleConnectionSeparateFromSessionStorage(t *testing.T) {
	kasus := []struct {
		name    string
		storage string
		adapter string
		wants   bool
	}{
		{"keduanya tiruan", config.StorageMemory, config.IdentityAdapterFake, false},
		{"identitas nyata, sesi di memori", config.StorageMemory, config.IdentityAdapterHCQ, true},
		{"sesi di Oracle, identitas tiruan", config.StorageOracle, config.IdentityAdapterFake, true},
		{"keduanya nyata", config.StorageOracle, config.IdentityAdapterHCQ, true},
	}
	for _, k := range kasus {
		t.Run(k.name, func(t *testing.T) {
			cfg := devConfig()
			cfg.Storage = k.storage
			cfg.IdentityAdapter = k.adapter
			require.Equal(t, k.wants, needsOracle(cfg))
		})
	}
}

// Modul Master Status Progres ikut terpasang saat perakitan, dan penyimpanan tanpa
// basis data pun tetap MENOLAK portal selain portal utama.
//
// Penolakan itu yang membuat perilaku pengembangan sama dengan produksi: memilih
// entitas yang koneksinya belum hidup menghasilkan galat di keduanya, bukan diam-diam
// dilayani basis data entitas lain (R-20).
func TestProgressStatusModuleMountedAndOtherPortalsRejected(t *testing.T) {
	result, err := build(devConfig(), logging.New(0))
	require.NoError(t, err)
	t.Cleanup(result.close)
	require.NotNil(t, result.masterStatusProgres)

	// Portal utama dilayani.
	require.NoError(t, result.masterStatusProgres.EnsurePortalReady("ASM"))

	// Entitas lain — dan portal yang tidak disebut sama sekali — ditolak.
	for _, alias := range []string{"ASI", "SMAS", "TIDAKADA", ""} {
		require.Error(t, result.masterStatusProgres.EnsurePortalReady(alias),
			"portal %q tidak boleh dilayani tanpa koneksi basis datanya sendiri", alias)
	}
}

// Penyimpanan di memori dipakai kembali antarpermintaan, bukan dibuat ulang.
//
// Bila dibuat ulang, penambahan yang baru disimpan akan hilang pada permintaan
// berikutnya dan layarnya tampak rusak tanpa sebab yang terlihat.
func TestMemoryStoreIsReused(t *testing.T) {
	selector := progressStatusSelectorMemory("ASM")

	first, err := selector("ASM")
	require.NoError(t, err)
	second, err := selector("asm") // huruf kecil harus menunjuk penyimpanan yang sama
	require.NoError(t, err)
	require.Same(t, first, second)
}

// fakeConnector membuka *sql.DB tanpa pernah menyentuh basis data.
//
// database/sql tidak menghubungi driver sampai kueri pertama, sehingga ini cukup untuk
// menguji PERAKITAN — yang diuji adalah daftar seam, bukan SQL-nya.
type fakeConnector struct{}

func (fakeConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("uji perakitan: koneksi tidak pernah dipakai")
}
func (fakeConnector) Driver() driver.Driver { return nil }

// TestRegistrationAssemblesOnBothBranches menjaga daftar seam modul Registrasi tetap utuh
// di KEDUA cabang perakitannya.
//
// # Kenapa uji ini ada
//
// Pada 2026-09-24 seam `ClaimReportLink` ditambahkan ke `assembleRegistration`, sementara
// `build` merakit modulnya SENDIRI dengan daftar seam terpisah. Yang disunting adalah
// daftar yang tidak pernah dipanggil, dan akibatnya baru terlihat saat aplikasi
// dijalankan:
//
//	gagal menjalankan aplikasi: registrasi/usecase: seam belum terpasang: [TautanLaporan]
//
// Build tetap bersih — `go vet` tidak menandai fungsi yang tidak terpakai — dan seluruh
// uji lulus, karena uji usecase merakit layanannya sendiri. Duplikasinya sudah dihapus;
// uji ini menjaga agar seam baru tidak lolos lagi tanpa ketahuan.
func TestRegistrationAssemblesOnBothBranches(t *testing.T) {
	logger := logging.New(slog.LevelError)

	t.Run("tanpa Oracle", func(t *testing.T) {
		service, err := assembleRegistration(nil, logger)
		require.NoError(t, err)
		require.NotNil(t, service)
	})

	t.Run("dengan Oracle", func(t *testing.T) {
		db := sql.OpenDB(fakeConnector{})
		defer db.Close()

		service, err := assembleRegistration(db, logger)
		require.NoError(t, err)
		require.NotNil(t, service)
	})
}
