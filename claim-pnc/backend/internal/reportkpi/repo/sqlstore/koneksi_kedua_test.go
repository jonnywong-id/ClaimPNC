package sqlstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// Kalender libur dibaca di KONEKSI KEDUA, bukan di basis data portal.
//
// # Kenapa uji ini ada
//
// Ia mengunci cacat nyata yang ditemukan 2026-10-08: tombol Cari tab KPI PIC Teknik
// menjawab "Terjadi kesalahan pada sistem" untuk setiap pencarian.
//
// Sebabnya, kueri `holidays` membaca `GENERAL.HRD_LBR` di koneksi POOLDATA. Objek itu
// tidak ada di sana — di Pega ia dicapai lewat DB Link `@ASMD.SINARMAS.CO.ID`
// (`RDB List/CheckHoliday_SQL-SQL.xml`, satu-satunya kemunculannya di seluruh export),
// dan `D-25`/`R-03` menggantinya dengan koneksi kedua.
//
// Modul Report Klaim sudah benar sejak awal. Yang keliru hanya modul ini, dan
// ketidakkonsistenan itu tidak terlihat karena keduanya ditulis terpisah — tidak ada satu
// pun uji yang memeriksa DI KONEKSI MANA sebuah kueri berjalan.
//
// Uji ini memakai DUA mock yang berbeda, karena itu satu-satunya cara membedakannya:
// dengan satu mock, kueri yang salah koneksi tetap lulus.
func TestHolidaysDibacaDiKoneksiKedua(t *testing.T) {
	utama, mockUtama, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = utama.Close() }()

	kedua, mockKedua, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = kedua.Close() }()

	hari := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockKedua.ExpectQuery(sqlOf("holidays")).
		WithArgs(hari, hari).
		WillReturnRows(sqlmock.NewRows([]string{"TANGGAL"}).AddRow(hari))

	repo := NewRepo(utama, kedua)
	days, err := repo.Holidays(context.Background(), hari, hari)
	require.NoError(t, err)
	require.Equal(t, []time.Time{hari}, days)

	// Koneksi KEDUA menerima kuerinya …
	require.NoError(t, mockKedua.ExpectationsWereMet())
	// … dan koneksi utama TIDAK menerima apa pun. Inilah inti ujinya.
	require.NoError(t, mockUtama.ExpectationsWereMet())
}

// Tanpa koneksi kedua, permintaan DITOLAK — bukan dihitung tanpa kalender libur.
//
// # Kenapa penolakan, bukan kalender kosong
//
// Tanpa kalender, perhitungan hari kerja hanya memotong akhir pekan. Setiap rentang yang
// memuat hari libur karena itu dihitung LEBIH PANJANG, dan setiap PIC tampak LEBIH LAMBAT
// dari kenyataannya — pada angka yang dipakai menilai orang.
//
// Yang membuatnya berbahaya bukan besarnya selisih, melainkan DIAMNYA: hasilnya tampil
// sebagai nilai yang wajar, tanpa satu pun tanda bahwa ia salah. Itu kelas cacat yang
// sama dengan `GETCURRENCYSTANDARD` yang mengembalikan `1` dan `GETSELISIHJAM` yang
// mengembalikan `0` — dua-duanya diperbaiki `D-49` justru karena kegagalannya diam.
func TestTanpaKoneksiKeduaKalenderLiburDitolak(t *testing.T) {
	utama, mockUtama, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = utama.Close() }()

	hari := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Jalur cadangan DB Link ikut gagal — meniru basis data yang DB Link-nya sudah dicabut.
	mockUtama.ExpectQuery(sqlOf("holidays_dblink")).WillReturnError(errBoom)

	repo := NewRepo(utama, nil)
	_, err = repo.Holidays(context.Background(), hari, hari)

	require.ErrorIs(t, err, ErrHolidayCalendarUnavailable)
	require.Contains(t, err.Error(), "ANEKA_<PORTAL_ALIAS>_*",
		"pesannya harus menyebutkan APA yang dipasang, bukan hanya bahwa sesuatu kurang")
	require.NotErrorIs(t, err, ErrQueryFailed,
		"yang dilaporkan adalah ketiadaan koneksi kedua — itu yang dapat ditindaklanjuti; "+
			"galat DB Link hanya keterangan, karena DB Link memang akan dipensiunkan")

	require.NoError(t, mockUtama.ExpectationsWereMet())
}

// Tanpa koneksi kedua, kalender diambil lewat DB LINK — persis seperti Pega.
//
// # Kenapa jalur cadangan ini ada
//
// Pada 2026-10-08 terbukti `GENERAL.HRD_LBR@ASMD.SINARMAS.CO.ID` MASIH terbaca dari akun
// POOLDATA, sementara kredensial koneksi kedua belum tersedia dan menunggunya memblokir
// seluruh tab. Jalur ini memakai objek dan DB Link yang sama dengan
// `RDB List/CheckHoliday_SQL-SQL.xml`, jadi ia meniru Pega apa adanya (`P-5`).
//
// Uji ini memakai DUA mock supaya arah jatuhnya terbukti: kueri cadangan harus mendarat
// di koneksi UTAMA, bukan di tempat lain.
func TestTanpaKoneksiKeduaJatuhKeDBLink(t *testing.T) {
	utama, mockUtama, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer func() { _ = utama.Close() }()

	hari := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockUtama.ExpectQuery(sqlOf("holidays_dblink")).
		WithArgs(hari, hari).
		WillReturnRows(sqlmock.NewRows([]string{"TANGGAL"}).AddRow(hari))

	repo := NewRepo(utama, nil)
	days, err := repo.Holidays(context.Background(), hari, hari)

	require.NoError(t, err)
	require.Equal(t, []time.Time{hari}, days)
	require.NoError(t, mockUtama.ExpectationsWereMet())
}

// Kueri cadangan BENAR-BENAR menyebut DB Link; yang utama TIDAK.
//
// Tanpa uji ini, keduanya dapat diam-diam menjadi kueri yang sama — dan jalur "utama"
// berhenti menjadi pengganti DB Link, yang justru seluruh alasan `D-25` ada.
func TestKueriCadanganMemakaiDBLinkKueriUtamaTidak(t *testing.T) {
	require.Contains(t, query("holidays_dblink"), "@ASMD.SINARMAS.CO.ID")
	require.NotContains(t, query("holidays"), "@",
		"kueri koneksi kedua TIDAK boleh berakhiran DB Link — koneksinya sendiri yang "+
			"sudah menunjuk basis data itu")
}

// Kueri yang ditolak basis data membawa NAMA kuerinya.
//
// Tab ini menjalankan delapan kueri. Tanpa namanya, "Terjadi kesalahan pada sistem"
// tidak menyebutkan satu pun dari delapan — dan itulah yang membuat cacat di atas
// memakan waktu jauh lebih lama daripada seharusnya.
func TestGalatKueriMembawaNamanya(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("closure_spans")).WillReturnError(errBoom)

	_, err := repo.ClosureSpans(context.Background(), picSpan())

	require.ErrorIs(t, err, ErrQueryFailed)
	require.ErrorIs(t, err, errBoom, "galat aslinya harus tetap terbaca di rantainya")
	require.Contains(t, err.Error(), "closure_spans")
	require.True(t, errors.Is(err, ErrQueryFailed))
}
