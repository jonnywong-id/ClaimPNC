package money_test

import (
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/money"
)

func TestUraiMenerimaBentukYangSah(t *testing.T) {
	kasus := []struct {
		teks   string
		satuan int64
	}{
		{"0", 0},
		{"1", 100},
		// Ambang bawah yang benar-benar ada di master ambang komite.
		{"50000000", 5_000_000_000},
		{"50000001", 5_000_000_100},
		{"100000001", 10_000_000_100},
		{"1000000001", 100_000_000_100},
		// Nilai terbesar pada master hari ini — Database/emailkomite.csv baris ID 4.
		{"100000000000", 10_000_000_000_000},
		{"12.5", 1_250},
		{"12.50", 1_250},
		{"12.05", 1_205},
		{"-7.25", -725},
		{"+7.25", 725},
		{"  42  ", 4_200},
	}

	for _, k := range kasus {
		t.Run(k.teks, func(t *testing.T) {
			hasil, err := money.Parse(k.teks)
			require.NoError(t, err)
			require.Equal(t, k.satuan, hasil.MinorUnits())
		})
	}
}

func TestUraiMenolakBentukYangTidakSah(t *testing.T) {
	// Pemisah ribuan ditolak dengan sengaja: titik berarti ribuan di Indonesia dan
	// desimal di Inggris, sehingga menerimanya berarti menebak maksud pengirim.
	// Notasi ilmiah ditolak karena ia jalan masuk ketidaktepatan floating point.
	kasus := []string{
		"", "   ", "-", "+", "abc", "1,5", "50.000.000", "1e8", "1.234", ".5", "5.", "1 000",
	}

	for _, teks := range kasus {
		t.Run(teks, func(t *testing.T) {
			_, err := money.Parse(teks)
			require.ErrorIs(t, err, money.ErrFormat)
		})
	}
}

func TestStringSelaluDuaAngkaDiBelakangTitik(t *testing.T) {
	kasus := []struct {
		nilai money.Money
		mau   string
	}{
		{money.FromRupiah(0), "0.00"},
		{money.FromRupiah(50_000_000), "50000000.00"},
		{money.FromMinorUnits(1_205), "12.05"},
		{money.FromMinorUnits(1_250), "12.50"},
		{money.FromMinorUnits(-725), "-7.25"},
	}

	for _, k := range kasus {
		t.Run(k.mau, func(t *testing.T) {
			require.Equal(t, k.mau, k.nilai.String())
		})
	}
}

// Bentuk kanonik harus dapat diurai kembali menjadi nilai yang sama persis. Tanpa sifat
// ini, nilai yang dikirim ke peramban lalu dikembalikan akan bergeser diam-diam.
func TestStringDanUraiSalingMembalik(t *testing.T) {
	nilai := []money.Money{
		money.Zero,
		money.FromRupiah(1),
		money.FromRupiah(50_000_001),
		money.FromRupiah(100_000_000_000),
		money.FromMinorUnits(1_205),
		money.FromMinorUnits(-725),
	}

	for _, n := range nilai {
		t.Run(n.String(), func(t *testing.T) {
			kembali, err := money.Parse(n.String())
			require.NoError(t, err)
			require.Equal(t, n, kembali)
		})
	}
}

// Perbandingan langsung dengan operator harus berperilaku seperti yang terbaca. Inilah
// sifat yang dipakai mesin penjenjangan saat menguji LIMIT_BOTTOM <= nilai klaim, dan
// selisih satu rupiah di sana menentukan satu jenjang persetujuan ikut atau tidak.
func TestPerbandinganTepatPadaSelisihSatuRupiah(t *testing.T) {
	ambang := money.FromRupiah(50_000_001)

	require.False(t, ambang <= money.FromRupiah(50_000_000), "Rp 50.000.000 belum melampaui ambang")
	require.True(t, ambang <= money.FromRupiah(50_000_001), "Rp 50.000.001 tepat melampaui ambang")
	require.True(t, ambang <= money.FromRupiah(50_000_002))
}

func TestDariNilaiSQLMenanganiSetiapBentukDriver(t *testing.T) {
	t.Run("nil menjadi nol", func(t *testing.T) {
		hasil, err := money.FromSQLValue(nil)
		require.NoError(t, err)
		require.Equal(t, money.Zero, hasil)
	})

	t.Run("int64", func(t *testing.T) {
		hasil, err := money.FromSQLValue(int64(50_000_001))
		require.NoError(t, err)
		require.Equal(t, money.FromRupiah(50_000_001), hasil)
	})

	t.Run("string", func(t *testing.T) {
		hasil, err := money.FromSQLValue("50000001")
		require.NoError(t, err)
		require.Equal(t, money.FromRupiah(50_000_001), hasil)
	})

	t.Run("byte", func(t *testing.T) {
		hasil, err := money.FromSQLValue([]byte("100000000000"))
		require.NoError(t, err)
		require.Equal(t, money.FromRupiah(100_000_000_000), hasil)
	})

	// Rp 100.000.000 sebagai float64 adalah bilangan bulat dan masih jauh di bawah
	// 2^53, sehingga nilainya tepat dan boleh diterima.
	t.Run("float64 bulat dalam rentang tepat", func(t *testing.T) {
		hasil, err := money.FromSQLValue(float64(100_000_000))
		require.NoError(t, err)
		require.Equal(t, money.FromRupiah(100_000_000), hasil)
	})

	// Yang berdesimal ditolak, bukan dibulatkan: nilai uang yang sudah kehilangan
	// ketepatan tidak boleh diteruskan seolah-olah masih tepat.
	t.Run("float64 berdesimal ditolak", func(t *testing.T) {
		_, err := money.FromSQLValue(float64(12.5))
		require.ErrorIs(t, err, money.ErrFormat)
	})

	t.Run("float64 di luar rentang tepat ditolak", func(t *testing.T) {
		_, err := money.FromSQLValue(float64(1 << 54))
		require.ErrorIs(t, err, money.ErrFormat)
	})

	t.Run("tipe asing ditolak", func(t *testing.T) {
		_, err := money.FromSQLValue(struct{}{})
		require.ErrorIs(t, err, money.ErrFormat)
	})
}

// Notasi ilmiah adalah bentuk yang akan dihasilkan database/sql bila float64 dipaksa
// menjadi string. Menolaknya di sini membuktikan kenapa FromSQLValue menangani float64
// sendiri alih-alih memindai ke *string.
func TestUraiMenolakNotasiIlmiahYangDihasilkanFormatFloat(t *testing.T) {
	_, err := money.Parse("1e+08")
	require.ErrorIs(t, err, money.ErrFormat)
}

func TestDariNilaiSQLMenolakBilanganTerlaluBesarTanpaPanik(t *testing.T) {
	// Data dari basis data adalah masukan dari luar; yang aneh harus menjadi galat,
	// bukan menjatuhkan proses yang sedang melayani pengguna lain.
	require.NotPanics(t, func() {
		_, err := money.FromSQLValue(int64(math.MaxInt64))
		require.Error(t, err)
		require.True(t, errors.Is(err, money.ErrFormat))
	})
}
