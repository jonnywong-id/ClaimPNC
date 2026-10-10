package sqlstore

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
)

// newMockWriter menyiapkan AssignmentWriter di atas koneksi tiruan.
func newMockWriter(t *testing.T) (*AssignmentWriter, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewAssignmentWriter(db), mock
}

// TestMovePICWritesEverythingInOneTransaction menjaga yang paling mudah rusak diam-diam.
//
// PIC yang berpindah tanpa pencacah yang ikut bergeser membuat pemilihan petugas otomatis
// (`R-04`) memakai angka yang salah — dan kesalahan itu tidak tampak sebagai galat, hanya
// sebagai pembagian beban yang timpang.
func TestMovePICWritesEverythingInOneTransaction(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{"USERTEKNIS_1"}).AddRow("PICLAMA"))
	mock.ExpectExec(exact("pindah_pic")).WithArgs("PICBARU", "KUNCI-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("pencacah_naik")).WithArgs("PICBARU").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Beban PIC LAMA TIDAK diturunkan: kueri Pega-nya menurunkan TOTAL_JOB saja, dan kolom
	// itu tidak ada pada tabel yang dapat kita tulis. Lihat pindahpic.sql.
	mock.ExpectExec(exact("dashboard_pic")).WithArgs("PICBARU", "PNC-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	hasil, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "KUNCI-1", ClaimNumber: "PNC-1", ToOperator: "PICBARU",
	})
	require.NoError(t, err)

	// PIC lama dikembalikan supaya jejaknya mencatat keadaan SEBELUM pemindahan.
	require.Equal(t, "PICLAMA", hasil.FromOperator)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Klaim di tab Tampungan PIC belum punya PIC sama sekali.
//
// Dulu uji ini menjaga bahwa pencacah TURUN dilewati pada keadaan itu. Pencacah turun kini
// tidak ada sama sekali, sehingga yang dijaganya berubah: pemindahan klaim TANPA PIC lama
// tetap berjalan utuh dan mengembalikan PIC lama kosong — bukan galat.
func TestMovePICHandlesClaimWithoutPreviousPIC(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("KUNCI-2").
		WillReturnRows(sqlmock.NewRows([]string{"USERTEKNIS_1"}).AddRow(nil))
	mock.ExpectExec(exact("pindah_pic")).WithArgs("PICBARU", "KUNCI-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("pencacah_naik")).WithArgs("PICBARU").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("dashboard_pic")).WithArgs("PICBARU", "PNC-2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	hasil, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "KUNCI-2", ClaimNumber: "PNC-2", ToOperator: "PICBARU",
	})
	require.NoError(t, err)
	require.Equal(t, "", hasil.FromOperator)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Memindahkan ke petugas yang sama tidak menggeser pencacah: beban seseorang tidak bertambah
// hanya karena tombolnya ditekan dua kali.
func TestMovePICToSameOperatorChangesNothing(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("KUNCI-3").
		WillReturnRows(sqlmock.NewRows([]string{"USERTEKNIS_1"}).AddRow("PICSAMA"))
	mock.ExpectCommit()

	hasil, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "KUNCI-3", ClaimNumber: "PNC-3", ToOperator: "PICSAMA",
	})
	require.NoError(t, err)
	require.Equal(t, "PICSAMA", hasil.FromOperator)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Klaim yang hilang dijawab ErrClaimNotFound, bukan galat teknis — layar menjawabnya 404.
func TestMovePICReportsMissingClaim(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("TIDAK-ADA").
		WillReturnRows(sqlmock.NewRows([]string{"USERTEKNIS_1"}))
	mock.ExpectRollback()

	_, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "TIDAK-ADA", ToOperator: "PICBARU",
	})
	require.ErrorIs(t, err, dashboardclaim.ErrClaimNotFound)
}

// Kegagalan di tengah tidak boleh meninggalkan PIC yang sudah berpindah tanpa pencacahnya.
func TestMovePICRollsBackWhenACounterFails(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("KUNCI-4").
		WillReturnRows(sqlmock.NewRows([]string{"USERTEKNIS_1"}).AddRow("PICLAMA"))
	mock.ExpectExec(exact("pindah_pic")).WithArgs("PICBARU", "KUNCI-4").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("pencacah_naik")).WithArgs("PICBARU").
		WillReturnError(errors.New("hak UPDATE belum diberikan"))
	mock.ExpectRollback()

	_, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "KUNCI-4", ClaimNumber: "PNC-4", ToOperator: "PICBARU",
	})
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Isian kosong ditolak SEBELUM transaksi dibuka — tidak ada kunci baris yang diambil
// percuma.
func TestMovePICRejectsEmptyInput(t *testing.T) {
	writer, _ := newMockWriter(t)

	_, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{ToOperator: "PICBARU"})
	require.Error(t, err)

	_, err = writer.MovePIC(context.Background(), dashboardclaim.PICMove{ClaimID: "KUNCI"})
	require.Error(t, err)
}

// Kunci baris WAJIB ada pada pembacaan PIC lama.
//
// Tanpa FOR UPDATE, dua pemindahan bersamaan sama-sama membaca PIC lama yang sama lalu
// menurunkan pencacah orang yang sama dua kali — dan selisihnya tidak pernah terlihat
// sebagai galat.
func TestCurrentPICQueryLocksTheRow(t *testing.T) {
	require.Regexp(t, regexp.MustCompile(`(?i)FOR\s+UPDATE`), query("pic_sekarang"))
}

// TestMoveAllMatchingBindsTheSameFilterAsTheList menjaga penjodohan yang paling mahal.
//
// "Select All" memindahkan klaim yang cocok penyaring. Bila argumennya bergeser satu posisi
// dari urutan yang dipakai kueri daftar, tombolnya memindahkan himpunan yang BERBEDA dari
// yang dilihat pengguna — dan tidak ada galat yang muncul.
//
// `:1` ditempati operator tujuan, sehingga penyaringnya menyusul mulai `:2`.
func TestMoveAllMatchingBindsTheSameFilterAsTheList(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectExec(exact("pindah_pic_saring")).
		WithArgs("PICBARU",
			"pol_1", `%POL\_1%`, `%POL\_1%`, "PA", "PA", "PA", "PA", "PA",
			nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil).
		WillReturnResult(sqlmock.NewResult(0, 7))

	moved, err := writer.MoveAllMatching(context.Background(), dashboardclaim.Filter{
		Search: "pol_1", Business: dashboardclaim.BusinessPA,
	}, "PICBARU")
	require.NoError(t, err)
	require.Equal(t, 7, moved)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Petugas tujuan kosong ditolak sebelum satu baris pun tersentuh.
func TestMoveAllMatchingRejectsEmptyTarget(t *testing.T) {
	writer, _ := newMockWriter(t)

	_, err := writer.MoveAllMatching(context.Background(), dashboardclaim.Filter{}, "   ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "petugas tujuan")
}
