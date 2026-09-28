package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/repo/memory"
)

// Baris yang diserahkan pemanggil adalah SALINAN, bukan isi penyimpanan.
//
// # Kenapa uji ini ada
//
// Tanpa salinan, pemanggil yang mengubah satu sel akan mengubah isi penyimpanan bagi
// SELURUH permintaan berikutnya. Cacat itu tidak pernah terjadi pada penyimpanan SQL —
// yang selalu membaca ulang dari basis data — sehingga uji terhadap memori justru akan
// MENYEMBUNYIKANNYA alih-alih menemukannya, dan layarnya baru terlihat salah setelah
// dipakai beberapa kali.
func TestSearchReturnsCopies(t *testing.T) {
	repo := memory.NewSampleRepo()
	filter := monitoringslinkojk.Filter{}.Normalize()

	first, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01, filter)
	require.NoError(t, err)
	require.NotEmpty(t, first.Rows)

	asli := first.Rows[0].Get("no_klaim")
	require.NotEmpty(t, asli)

	first.Rows[0]["no_klaim"] = "DIUBAH-PEMANGGIL"

	kedua, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01, filter)
	require.NoError(t, err)
	require.Equal(t, asli, kedua.Rows[0].Get("no_klaim"),
		"perubahan pemanggil tidak boleh menyentuh isi penyimpanan")
}

// Stream pun menyerahkan salinan, dengan alasan yang sama.
func TestStreamReturnsCopies(t *testing.T) {
	repo := memory.NewSampleRepo()
	filter := monitoringslinkojk.Filter{}.Normalize()

	var asli string
	err := repo.Stream(context.Background(), monitoringslinkojk.SegmentD01, filter,
		func(row monitoringslinkojk.Row) error {
			if asli == "" {
				asli = row.Get("no_klaim")
				row["no_klaim"] = "DIUBAH-PEMANGGIL"
			}
			return nil
		})
	require.NoError(t, err)

	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01, filter)
	require.NoError(t, err)
	require.Equal(t, asli, page.Rows[0].Get("no_klaim"))
}

// Urutannya SAMA dengan kueri SQL — tanggal registrasi menurun, lalu kunci baris.
//
// Kesetaraan itu bukan kerapian: uji yang lulus terhadap penyimpanan memori tidak boleh
// menyembunyikan perbedaan urutan yang baru muncul di produksi.
func TestSearchOrdersNewestFirst(t *testing.T) {
	repo := memory.NewSampleRepo()
	filter := monitoringslinkojk.Filter{}.Normalize()

	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01, filter)
	require.NoError(t, err)
	require.Len(t, page.Rows, 4)

	// Baris termuda pada data contoh adalah klaim Surety Bond, 31 Maret pukul 23.40.
	require.Equal(t, "PNCN.26.0207", page.Rows[0].Get("no_klaim"))
	// Yang tertua adalah klaim Januari.
	require.Equal(t, "PNCN.26.0033", page.Rows[3].Get("no_klaim"))
}

// Segmen yang tidak punya data contoh dijawab senarai kosong, bukan panik.
func TestSearchUnknownSegmentIsEmpty(t *testing.T) {
	repo := memory.NewRepo()
	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentF06,
		monitoringslinkojk.Filter{}.Normalize())
	require.NoError(t, err)
	require.Empty(t, page.Rows)
	require.Zero(t, page.Total)
}

// Konteks yang sudah dibatalkan dihormati, bukan diabaikan.
func TestSearchRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := memory.NewSampleRepo().Search(ctx, monitoringslinkojk.SegmentD01,
		monitoringslinkojk.Filter{}.Normalize())
	require.ErrorIs(t, err, context.Canceled)
}

// Batas rentang tanggal di penyimpanan memori berperilaku PERSIS seperti di SQL:
// batas bawah inklusif, batas atas eksklusif terhadap hari berikutnya.
func TestDateBoundsMatchSQLBehaviour(t *testing.T) {
	repo := memory.NewSampleRepo()

	awal := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	page, err := repo.Search(context.Background(), monitoringslinkojk.SegmentD01,
		monitoringslinkojk.Filter{
			DateOfLoss:            &awal,
			DateOfRequestDocument: &akhir,
		}.Normalize())
	require.NoError(t, err)

	// Rentang SATU HARI yang berisi klaim pukul 23.40 pada hari itu wajib
	// mengembalikannya. Dengan `<=` terhadap tengah malam, hasilnya nol.
	require.Len(t, page.Rows, 1)
	require.Equal(t, "PNCN.26.0207", page.Rows[0].Get("no_klaim"))
}
