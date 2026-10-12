package inboxrclpucl_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/platform/clock"
)

// TestADriverTimestampIsDrawnAsADateNotAsISOText menjaga hal yang sampai 2026-09-30
// benar-benar terlihat pengguna.
//
// Keempat kolom tanggal modul ini bertipe `TIMESTAMP(6)`, dan driver mengembalikannya
// sebagai teks ISO ber-offset. Teks itu sampai ke layar apa adanya — bentuk yang tidak
// pernah muncul di layar Pega, tempat ketiga selnya digambar kontrol `pxDateTime`.
func TestADriverTimestampIsDrawnAsADateNotAsISOText(t *testing.T) {
	// Bentuk persis yang dikembalikan go-ora, disalin dari pembacaan langsung ke Oracle.
	require.Equal(t,
		"2025-06-13 14:41:01",
		inboxrclpucl.DisplayTimeText("2025-06-13T14:41:01.532+07:00"))
}

// TestAWallClockTextIsNotShiftedSevenHours menjaga kekeliruan yang TIDAK menghasilkan
// galat.
//
// Nilai tanpa zona adalah jam dinding WIB. Membacanya sebagai UTC lalu mengubahnya ke WIB
// memajukannya tujuh jam — dan pada nilai menjelang tengah malam itu memindahkannya ke hari
// berikutnya. `keputusan-implementasi.md` §49 mencatat kelas kekeliruan yang sama pada arah
// sebaliknya, dan di sana ia terbukti mengenai SETIAP tanggal, bukan kasus tepi.
func TestAWallClockTextIsNotShiftedSevenHours(t *testing.T) {
	require.Equal(t,
		"2026-09-14 23:30:00",
		inboxrclpucl.DisplayTimeText("2026-09-14 23:30:00"),
		"nilai tanpa zona adalah jam dinding WIB dan tidak boleh digeser")

	require.Equal(t,
		"2026-09-14 00:00:00",
		inboxrclpucl.DisplayTimeText("2026-09-14"),
		"tanggal tanpa jam pun tidak boleh berpindah hari")
}

// TestAnUnknownShapeIsPassedThroughUntouched menjaga modul ini tetap dapat dipakai di
// portal yang belum diperiksa.
//
// Tipe kolomnya terverifikasi pada SATU basis data dari enam (`R-08`). Di portal yang
// menyimpannya sebagai teks, mengosongkan nilai yang tidak dikenali akan menghapus isi
// kolom pada seluruh baris — kegagalan yang tidak menghasilkan satu pun galat.
func TestAnUnknownShapeIsPassedThroughUntouched(t *testing.T) {
	require.Equal(t, "12", inboxrclpucl.DisplayTimeText("12"))
	require.Equal(t, "belum ada", inboxrclpucl.DisplayTimeText("  belum ada  "))
	require.Equal(t, "", inboxrclpucl.DisplayTimeText("   "))
}

// TestAZeroTimeIsEmptyNotYearOne memisahkan ketiadaan tanggal dari tanggal tahun 1.
func TestAZeroTimeIsEmptyNotYearOne(t *testing.T) {
	require.Equal(t, "", inboxrclpucl.DisplayTime(time.Time{}))

	at := time.Date(2026, time.September, 30, 3, 15, 0, 0, time.UTC)
	require.Equal(t, "2026-09-30 10:15:00", inboxrclpucl.DisplayTime(at),
		"waktu ber-zona dikonversi ke WIB, satu-satunya tempat pergeseran zona terjadi")
	require.Equal(t, "2026-09-30 10:15:00", at.In(clock.ZoneWIB).Format(inboxrclpucl.DisplayTimeLayout))
}

// TestDrawingAnAlreadyDrawnValueChangesNothing menjaga penggambarnya idempoten.
//
// Ia dipakai penyimpanan SQL DAN penyimpanan memori, dan pada yang kedua sebagian nilainya
// sudah berbentuk tampilan. Penggambar yang menggeser nilai yang sudah digambar akan
// membuat kedua pengisi seam menghasilkan teks yang berbeda — persis yang harus dicegah.
func TestDrawingAnAlreadyDrawnValueChangesNothing(t *testing.T) {
	once := inboxrclpucl.DisplayTimeText("2025-06-13T14:41:01.532+07:00")
	require.Equal(t, once, inboxrclpucl.DisplayTimeText(once))
}
