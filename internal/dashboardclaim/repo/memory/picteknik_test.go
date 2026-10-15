package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
	"claim-pnc/internal/dashboardclaim/repo/memory"
)

// TestBusinessTypeTidakMenyaring menjaga adapter memori TIDAK lebih ketat daripada SQL-nya.
//
// `picteknik.sql` membuang penyaring `TYPE_BUSINESS` karena nilainya tidak ada di sistem baru
// (`pyPosition` Pega bukan `Placement.PositionName` HCC/HCQ). Bila adapter memori tetap
// menyaring, daftar PIC tampak benar saat dicoba tanpa Oracle lalu berperilaku lain di
// Oracle — dan selisih seperti itu baru ketahuan di tangan pengguna.
//
// Contohnya sengaja memuat dua lini: satu baris ber-`PA`, tiga ber-`NONMBU`. Bila penyaringnya
// hidup kembali, permintaan ber-lini `PA` akan mengembalikan satu baris, bukan empat.
func TestBusinessTypeTidakMenyaring(t *testing.T) {
	t.Parallel()

	reader := memory.NewPICReader()
	seluruh := len(reader.Rows)
	require.Greater(t, seluruh, 1, "contoh harus memuat lebih dari satu baris agar uji ini bermakna")

	halaman, err := reader.ListTechnicalPIC(context.Background(), dashboardclaim.TechnicalPICFilter{
		BusinessType: "PA",
		Limit:        50,
	})
	require.NoError(t, err)
	require.Equal(t, seluruh, halaman.Total,
		"lini bisnis tidak boleh menyaring — picteknik.sql pun tidak menyaringnya")
}

// TestPencarianTetapMenyaring memastikan pembuangan di atas tidak ikut membuang kotak cari.
//
// Keduanya diperiksa bersama dengan sengaja: menghapus satu penyaring lalu diam-diam ikut
// menghapus yang lain adalah kekeliruan yang mudah terjadi dan tidak kelihatan.
func TestPencarianTetapMenyaring(t *testing.T) {
	t.Parallel()

	reader := memory.NewPICReader()
	target := reader.Rows[0]

	halaman, err := reader.ListTechnicalPIC(context.Background(), dashboardclaim.TechnicalPICFilter{
		Search: target.OperatorID,
		Limit:  50,
	})
	require.NoError(t, err)
	require.Equal(t, 1, halaman.Total)
	require.Equal(t, target.OperatorID, halaman.Rows[0].OperatorID)
}
