package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
)

// TestListIsSortedAndCleaned membuktikan daftar terurut kode dan spasi tepi dibuang.
func TestListIsSortedAndCleaned(t *testing.T) {
	repo := NewRepo(
		mastertipesurveyors.SurveyorType{Code: "1002 ", Description: " LOSS ADJUSTER "},
		mastertipesurveyors.SurveyorType{Code: "1001", Description: "INTERNAL SURVEYOR"},
	)
	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, []mastertipesurveyors.SurveyorType{
		{Code: "1001", Description: "INTERNAL SURVEYOR"},
		{Code: "1002", Description: "LOSS ADJUSTER"},
	}, list)
}

// TestGetFindsTrimmedCodeOrReportsNotFound memeriksa Get atas kode ada dan tidak ada.
func TestGetFindsTrimmedCodeOrReportsNotFound(t *testing.T) {
	repo := NewRepo(SampleList()...)
	got, err := repo.Get(context.Background(), " 1003 ")
	require.NoError(t, err)
	require.Equal(t, "EXPERT", got.Description)

	_, err = repo.Get(context.Background(), "9999")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)
}

// TestInsertContinuesSequence membuktikan kode baru melanjutkan kode tertinggi.
func TestInsertContinuesSequence(t *testing.T) {
	repo := NewRepo(SampleList()...)
	added, err := repo.Insert(context.Background(), "  BARU ")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1005", Description: "BARU"}, added)

	got, err := repo.Get(context.Background(), "1005")
	require.NoError(t, err)
	require.Equal(t, added, got)
}

// TestInsertRejectsDuplicateDescription memeriksa keunikan tanpa peduli huruf.
func TestInsertRejectsDuplicateDescription(t *testing.T) {
	repo := NewRepo(SampleList()...)
	_, err := repo.Insert(context.Background(), "expert")
	require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
}

// TestInsertDetectsCodeClash membuktikan bentrok kode dilaporkan, bukan menimpa.
func TestInsertDetectsCodeClash(t *testing.T) {
	// Urutan sengaja dimundurkan ke 2 sehingga penambahan berikutnya menerbitkan "1003" yang
	// sudah dipakai — meniru urutan basis data yang tertinggal dari isi tabel.
	repo := NewRepo(
		mastertipesurveyors.SurveyorType{Code: "1002", Description: "A"},
		mastertipesurveyors.SurveyorType{Code: "1003", Description: "B"},
	)
	repo.order = 2
	_, err := repo.Insert(context.Background(), "C")
	require.ErrorIs(t, err, mastertipesurveyors.ErrCodeTaken)
}

// TestFirstInsertOnEmptyRepo membuktikan repo kosong memulai urutan dari 1.
func TestFirstInsertOnEmptyRepo(t *testing.T) {
	repo := NewRepo()
	added, err := repo.Insert(context.Background(), "PERTAMA")
	require.NoError(t, err)
	require.Equal(t, "1001", added.Code)
}

// TestUpdateChangesOnlyDescription memeriksa jalur ubah, tidak ditemukan, dan bentrok.
func TestUpdateChangesOnlyDescription(t *testing.T) {
	repo := NewRepo(
		mastertipesurveyors.SurveyorType{Code: "1001", Description: "A", LegacyCode: "01"},
		mastertipesurveyors.SurveyorType{Code: "1002", Description: "B"},
	)
	ctx := context.Background()

	updated, err := repo.Update(ctx, " 1001 ", " a ")
	require.NoError(t, err)
	require.Equal(t, mastertipesurveyors.SurveyorType{Code: "1001", Description: "a", LegacyCode: "01"}, updated)

	_, err = repo.Update(ctx, "1009", "X")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)

	_, err = repo.Update(ctx, "1001", "b")
	require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
}

// TestSetErrorMakesEveryOperationFail membuktikan SetError memaksa jalur gagal.
func TestSetErrorMakesEveryOperationFail(t *testing.T) {
	repo := NewRepo(SampleList()...)
	boom := errors.New("rusak")
	repo.SetError(boom)
	ctx := context.Background()

	_, err := repo.List(ctx)
	require.ErrorIs(t, err, boom)
	_, err = repo.Get(ctx, "1001")
	require.ErrorIs(t, err, boom)
	_, err = repo.Insert(ctx, "X")
	require.ErrorIs(t, err, boom)
	_, err = repo.Update(ctx, "1001", "X")
	require.ErrorIs(t, err, boom)
}

// TestThreeDigitsPadsLikeLpad meniru lpad Oracle termasuk bilangan di atas 999.
func TestThreeDigitsPadsLikeLpad(t *testing.T) {
	require.Equal(t, "000", threeDigits(0))
	require.Equal(t, "007", threeDigits(7))
	require.Equal(t, "042", threeDigits(42))
	require.Equal(t, "999", threeDigits(999))
	require.Equal(t, "1000", threeDigits(1000))
}

// TestSequenceFromCodeReadsOnlySiteCodes memeriksa semua cabang pembacaan urutan.
func TestSequenceFromCodeReadsOnlySiteCodes(t *testing.T) {
	require.Equal(t, 4, sequenceFromCode("1004", "1"))
	require.Equal(t, 0, sequenceFromCode("2004", "1"))
	require.Equal(t, 0, sequenceFromCode("1", "1"))
	require.Equal(t, 0, sequenceFromCode("10A4", "1"))
}
