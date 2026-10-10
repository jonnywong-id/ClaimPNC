package sqlstore

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

// Tanpa koneksi kedua, kedua pembacaan `general.*` jatuh ke kueri cadangan DB Link yang
// dijalankan pada koneksi PORTAL — bukan gagal, dan bukan pula dikosongkan.
//
// # Kenapa uji ini ada
//
// Pemilihannya tidak terlihat dari pemanggilnya: `holidayCalendar` dan `mitraKeep`
// keduanya hanya menyebut nama kueri tanpa akhiran. Yang memilih `Repo.koneksiKedua`,
// dan satu-satunya cara membuktikan pilihannya benar adalah memastikan kueri MANA yang
// benar-benar dikirim — termasuk ke koneksi yang mana.
//
// Keliru di sini tidak menimbulkan galat: kueri tanpa akhiran DB Link tetap sah, hanya
// tabelnya tidak ada di basis data portal. Galatnya ORA-00942, dan ia baru muncul saat
// pengguna menekan tombol.
func TestTanpaKoneksiKeduaMemakaiKueriCadanganDBLink(t *testing.T) {
	t.Run("daftar mitra", func(t *testing.T) {
		db, mock := newDB(t)
		repo := NewRepo(db, nil) // koneksi kedua TIDAK ada

		mock.ExpectQuery(q("report_mitra_logins_dblink")).WillReturnRows(
			sqlmock.NewRows([]string{"LOGIN"}).AddRow("MITRA01").AddRow("mitra02"))

		keep, err := mitraKeep(context.Background(), repo, reportklaim.Filter{})
		require.NoError(t, err, "cadangan DB Link seharusnya menggantikan koneksi kedua")
		require.NoError(t, mock.ExpectationsWereMet())

		// Penyaringnya tetap bekerja, dan tetap tidak peka huruf besar-kecil.
		require.True(t, keep(reportklaim.Row{"QQNAME": "mitra01"}))
		require.True(t, keep(reportklaim.Row{"QQNAME": " MITRA02 "}))
		require.False(t, keep(reportklaim.Row{"QQNAME": "BUKANMITRA"}))
	})

	t.Run("kalender libur", func(t *testing.T) {
		db, mock := newDB(t)
		repo := NewRepo(db, nil)

		mock.ExpectQuery(q("report_holiday_calendar_dblink")).WithArgs(from, to).WillReturnRows(
			sqlmock.NewRows([]string{"TGL"}).AddRow(from))

		calendar := repo.holidayCalendar(context.Background(), from, to)
		require.True(t, calendar.Available(), "kalender seharusnya terbaca lewat cadangan")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Koneksi kedua TETAP didahulukan begitu ia ada — cadangannya tidak boleh ikut terpakai.
//
// Tanpa uji ini, cadangan yang semestinya sementara dapat diam-diam menjadi jalur tetap:
// `ANEKA_*` terisi, tetapi laporan tetap menempuh DB Link, dan tidak ada yang menyadari
// bahwa utang portabilitasnya tidak pernah lunas.
func TestKoneksiKeduaDidahulukanDaripadaCadangan(t *testing.T) {
	db, mock := newDB(t)
	aneka, anekaMock := newDB(t)
	repo := NewRepo(db, aneka)

	// Yang diharapkan ada di koneksi KEDUA, dan kueri TANPA akhiran DB Link.
	anekaMock.ExpectQuery(q("report_mitra_logins")).WillReturnRows(
		sqlmock.NewRows([]string{"LOGIN"}).AddRow("MITRA01"))

	_, err := mitraKeep(context.Background(), repo, reportklaim.Filter{})
	require.NoError(t, err)

	require.NoError(t, anekaMock.ExpectationsWereMet())
	// Dan koneksi portal tidak disentuh sama sekali.
	require.NoError(t, mock.ExpectationsWereMet())
}
