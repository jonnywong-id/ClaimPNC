package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/komite/repo/memory"
	"claim-pnc/internal/komite/usecase"
	"claim-pnc/internal/platform/money"
)

func serviceUji(t *testing.T) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{Repo: memory.NewSampleRepo()})
	require.NoError(t, err)
	return service
}

// Bahan yang tidak lengkap ditolak saat start, bukan saat pengguna pertama membuka layar.
func TestLayananBaruMenolakBahanTidakLengkap(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestDaftarAmbangMengembalikanSeluruhBaris(t *testing.T) {
	daftar, err := serviceUji(t).ListThresholds(context.Background())
	require.NoError(t, err)
	require.Len(t, daftar, len(memory.SampleThresholds()))
}

func TestDaftarLiniDiambilDariData(t *testing.T) {
	lini, err := serviceUji(t).ListBusinessLines(context.Background())
	require.NoError(t, err)
	require.Equal(t,
		[]komite.BusinessLine{"BONDING", "NONMBU", "NONMBUAB", "NONMBUC", "PA", "TRAVEL"},
		lini)
}

// Galat domain diteruskan APA ADANYA, tidak dibungkus.
//
// Ini bukan kerapian: lapisan transport memetakannya ke status HTTP lewat errors.Is dan
// errors.As, dan membungkusnya dengan fmt.Errorf tanpa %w akan membuat pemetaan itu
// gagal diam-diam — lini yang salah ketik akan muncul sebagai galat 500 alih-alih 404.
func TestGalatDomainDapatDikenaliPemanggil(t *testing.T) {
	service := serviceUji(t)

	t.Run("lini tidak dikenal", func(t *testing.T) {
		_, err := service.Tiering(context.Background(), money.FromRupiah(1_000_000), "MARINE")
		require.ErrorIs(t, err, komite.ErrUnknownBusinessLine)
	})

	t.Run("validasi masukan", func(t *testing.T) {
		_, err := service.Tiering(context.Background(), money.FromRupiah(1_000_000), "  ")

		var validasi *komite.ValidationError
		require.ErrorAs(t, err, &validasi)
	})
}

// Galat penyimpanan DIBUNGKUS dengan konteks, supaya jejaknya terbaca dari pesannya —
// bukan sekadar "koneksi putus" tanpa keterangan sedang membaca apa.
func TestGalatPenyimpananDibungkusDenganKonteks(t *testing.T) {
	repo := memory.NewSampleRepo()
	putus := errors.New("koneksi putus")
	repo.SetError(putus)

	service, err := usecase.NewService(usecase.Options{Repo: repo})
	require.NoError(t, err)

	for nama, jalankan := range map[string]func() error{
		"daftar ambang": func() error {
			_, e := service.ListThresholds(context.Background())
			return e
		},
		"penjenjangan": func() error {
			_, e := service.Tiering(context.Background(), money.FromRupiah(1), "PA")
			return e
		},
		"integritas": func() error {
			_, e := service.Integrity(context.Background())
			return e
		},
	} {
		t.Run(nama, func(t *testing.T) {
			galat := jalankan()
			require.ErrorIs(t, galat, putus, "galat asalnya harus tetap dapat dikenali")
			require.Contains(t, galat.Error(), "komite/usecase",
				"pesannya harus menyebut di mana kegagalannya terjadi")
		})
	}
}

// Kebijakan pita dapat dipasok dari luar.
//
// Sifat inilah yang membuat batas pita — satu-satunya nilai bisnis yang masih berupa
// angka di dalam kode — dapat berpindah menjadi master `F-4` tanpa menyentuh satu baris
// pun aturan penjenjangan.
func TestKebijakanPitaDapatDiganti(t *testing.T) {
	// Batas diturunkan menjadi Rp 60 juta. Klaim Rp 80 juta yang tadinya masuk pita
	// bawah kini masuk pita atas, dan jumlah penyetujunya ikut berubah.
	kebijakan := komite.Policy{
		Bands: map[komite.BusinessLine]komite.BandPolicy{
			komite.BusinessLineNonMBU: {
				Boundary: money.FromRupiah(60_000_000),
				Lower:    komite.BandLower,
				Upper:    komite.BandUpper,
			},
		},
	}

	service, err := usecase.NewService(usecase.Options{
		Repo:   memory.NewSampleRepo(),
		Policy: &kebijakan,
	})
	require.NoError(t, err)

	hasil, err := service.Tiering(context.Background(), money.FromRupiah(80_000_000), "NONMBU")
	require.NoError(t, err)

	require.Equal(t, komite.BandUpper, hasil.Band)
	require.Zero(t, hasil.TierCount(),
		"pita atas tidak punya anak tangga di bawah Rp 100.000.001")

	// Dan kebijakannya dapat dibaca kembali, supaya layar dapat menampilkannya.
	require.Equal(t, money.FromRupiah(60_000_000), service.Policy().Bands[komite.BusinessLineNonMBU].Boundary)
}

func TestIntegritasMelaporkanTemuanMasterYangBerlaku(t *testing.T) {
	temuan, err := serviceUji(t).Integrity(context.Background())
	require.NoError(t, err)

	cacat := 0
	for _, s := range temuan {
		if s.Severity == komite.SeverityDefect {
			cacat++
		}
	}
	require.Zero(t, cacat, "master yang berlaku bersih dari cacat")
	require.NotEmpty(t, temuan, "peringatannya tetap ada dan itulah nilainya")
}
