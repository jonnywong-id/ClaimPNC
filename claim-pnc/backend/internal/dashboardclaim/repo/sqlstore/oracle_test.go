package sqlstore

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dashboardclaim"
)

// TestNeedsDBARecognisesWhatIsActuallyMissing menjaga pembedaan yang menentukan pesan layar.
//
// Tiga keadaan sempat tampil sama — "Terjadi kesalahan pada sistem" — padahal tindak
// lanjutnya berbeda: yang satu menunggu migrasi, yang satu menunggu GRANT, yang satu
// benar-benar cacat. Pesan yang sama untuk ketiganya membuat pengguna menyangka sistemnya
// rusak.
func TestNeedsDBARecognisesWhatIsActuallyMissing(t *testing.T) {
	require.True(t, needsDBA(errors.New("ORA-00942: table or view does not exist")))
	require.True(t, needsDBA(errors.New("ORA-01031: insufficient privileges")))

	// Huruf besar-kecil tidak menentukan: driver dan tiruan menuliskannya berbeda.
	require.True(t, needsDBA(errors.New("ora-00942")))

	// Yang BUKAN urusan DBA tidak boleh ikut tertangkap — kueri yang salah kolom akan
	// menunggu DBA selamanya bila ia ikut.
	require.False(t, needsDBA(errors.New("ORA-00904: invalid identifier")))
	require.False(t, needsDBA(errors.New("connection refused")))
	require.False(t, needsDBA(nil))

	require.True(t, badColumn(errors.New("ORA-00904: \"TOTAL_JOB\": invalid identifier")))
	require.False(t, badColumn(errors.New("ORA-00942")))
	require.False(t, badColumn(nil))
}

// Hak UPDATE yang belum diberikan dijawab ErrAssignmentUnavailable.
//
// Ia tertangkap pada pembacaan PIC lama, sebelum satu kolom pun berubah: `FOR UPDATE`
// menuntut hak menulis, bukan hanya membaca.
func TestMovePICReportsMissingGrantAsUnavailable(t *testing.T) {
	writer, mock := newMockWriter(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("pic_sekarang")).WithArgs("KUNCI-1").
		WillReturnError(errors.New("ORA-01031: insufficient privileges"))
	mock.ExpectRollback()

	_, err := writer.MovePIC(context.Background(), dashboardclaim.PICMove{
		ClaimID: "KUNCI-1", ToOperator: "PICBARU",
	})

	require.ErrorIs(t, err, dashboardclaim.ErrAssignmentUnavailable)
	require.ErrorContains(t, err, "ORA-01031")
}
