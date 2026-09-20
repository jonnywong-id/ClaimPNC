package masterpenolakan_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenolakan"
)

// violationsOf mengumpulkan pelanggaran per nama isian, supaya uji dapat menyebut isian
// mana yang salah alih-alih mencocokkan teks pesannya.
func violationsOf(t *testing.T, err error) map[string]string {
	t.Helper()

	var validationError *masterpenolakan.ValidationError
	require.ErrorAs(t, err, &validationError)

	result := map[string]string{}
	for _, violation := range validationError.Violation {
		result[violation.Field] = violation.Message
	}
	return result
}

func TestEmptyFieldsAreAllReportedAtOnce(t *testing.T) {
	// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja. Ini kesetaraan
	// perilaku (P-5): layar lama memeriksa belasan aturan lalu menampilkan semuanya
	// bersamaan.
	err := masterpenolakan.Input{}.Clean().Check()

	violation := violationsOf(t, err)
	require.Len(t, violation, 2)
	require.Contains(t, violation, "nama")
	require.Contains(t, violation, "nama_status_1")
}

func TestWhitespaceOnlyFieldIsTreatedAsEmpty(t *testing.T) {
	// Clean berjalan SEBELUM Check. Kalau dibalik, isian berisi spasi saja lolos aturan
	// wajib-isi lalu tersimpan sebagai teks kosong.
	input := masterpenolakan.Input{Name: "   ", ParentName: "\t\n"}.Clean()

	violation := violationsOf(t, input.Check())
	require.Contains(t, violation, "nama")
	require.Contains(t, violation, "nama_status_1")
}

func TestNameLongerThanLimitIsRejected(t *testing.T) {
	input := masterpenolakan.Input{
		Name:       strings.Repeat("A", masterpenolakan.MaxNameLength+1),
		ParentName: "POLIS TIDAK BERLAKU",
	}.Clean()

	violation := violationsOf(t, input.Check())
	require.Contains(t, violation, "nama")
	require.NotContains(t, violation, "nama_status_1")
}

func TestNameExactlyAtLimitIsAccepted(t *testing.T) {
	// Kasus TEPAT DI BATAS tidak boleh dilewatkan: di situlah aturan berbatas paling
	// sering salah (`14-TESTING-STRATEGY.md` §3.1).
	input := masterpenolakan.Input{
		Name:       strings.Repeat("A", masterpenolakan.MaxNameLength),
		ParentName: "POLIS TIDAK BERLAKU",
	}.Clean()

	require.NoError(t, input.Check())
}

func TestChoosingExistingParentDoesNotRequireNewParentName(t *testing.T) {
	// Induk diperiksa menurut CARA pengguna memilihnya. Memeriksa keduanya sekaligus akan
	// menuntut isian yang memang sengaja dikosongkan.
	input := masterpenolakan.Input{Name: "PREMI BELUM DIBAYAR", ParentID: "1"}.Clean()

	require.NoError(t, input.Check())
	require.False(t, input.WantsNewParent())
}

func TestEmptyParentIDMeansNewParentIsRequested(t *testing.T) {
	input := masterpenolakan.Input{Name: "PREMI BELUM DIBAYAR", ParentName: "POLIS TIDAK BERLAKU"}.Clean()

	require.True(t, input.WantsNewParent())
	require.NoError(t, input.Check())
}

func TestFillingBothParentFieldsIsRejected(t *testing.T) {
	// Dua sumber untuk satu nilai berarti keduanya dapat berbeda, dan yang mana yang
	// menang menjadi pertanyaan yang tidak perlu ada. Ditolak terang-terangan alih-alih
	// dipilih diam-diam.
	input := masterpenolakan.Input{
		Name:       "PREMI BELUM DIBAYAR",
		ParentID:   "1",
		ParentName: "POLIS TIDAK BERLAKU",
	}.Clean()

	violation := violationsOf(t, input.Check())
	require.Contains(t, violation, "nama_status_1")
}

func TestApprovalStatusLabelMatchesLegacyQuery(t *testing.T) {
	// Ketiga teks disalin apa adanya dari derivasi
	// `RDB List/BrowseStatusPenolakanKlaim2-SQL.xml`, termasuk huruf besarnya (`D-13`).
	require.Equal(t, "MENUNGGU", masterpenolakan.StatusPending.Label())
	require.Equal(t, "APPROVED", masterpenolakan.StatusApproved.Label())
	require.Equal(t, "REJECTED", masterpenolakan.StatusRejected.Label())
}

func TestUnknownStatusFallsBackToWaitingLikeLegacyQuery(t *testing.T) {
	// Kueri lama memakai ELSE, bukan WHEN '0' — baris ber-STATUS kosong, NULL, atau
	// bernilai lain apa pun ikut terbaca MENUNGGU. Ditiru supaya baris warisan yang
	// kolomnya tidak terisi tidak menghasilkan sel kosong yang tampak seperti cacat layar.
	require.Equal(t, "MENUNGGU", masterpenolakan.ApprovalStatus("").Label())
	require.Equal(t, "MENUNGGU", masterpenolakan.ApprovalStatus("9").Label())

	// Label memaafkan; Known tidak. Yang boleh DITULIS hanya ketiga nilai yang sah.
	require.False(t, masterpenolakan.ApprovalStatus("9").Known())
	require.True(t, masterpenolakan.StatusPending.Known())
}

func TestNextSequenceContinuesFromHighestNumber(t *testing.T) {
	require.Equal(t, "1", masterpenolakan.NextSequence(nil, masterpenolakan.FormatID))
	require.Equal(t, "4", masterpenolakan.NextSequence([]string{"1", "2", "3"}, masterpenolakan.FormatID))
}

func TestNextSequenceReadsIDsAsNumbersNotText(t *testing.T) {
	// Procedure lama memakai `max(to_number(ID_ST))+1`, sehingga "10" lebih besar dari
	// "9". Menghitungnya sebagai teks akan mengembalikan "10" lagi dan menabrak baris yang
	// sudah ada.
	require.Equal(t, "11", masterpenolakan.NextSequence([]string{"9", "10"}, masterpenolakan.FormatID))
}

func TestNextSequenceSkipsUnreadableIDsWithoutColliding(t *testing.T) {
	// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
	// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja.
	//
	// Di procedure lama, satu baris semacam ini menghentikan SELURUH penambahan dengan
	// ORA-01722 karena `to_number` gagal. Di sini ia hanya dilewati.
	require.Equal(t, "3", masterpenolakan.NextSequence([]string{"2", "LAMA"}, masterpenolakan.FormatID))
}

func TestNextSequenceAvoidsIDAlreadyTaken(t *testing.T) {
	// Perulangan pencari calon ada untuk tabel yang sudah memuat ID berbentuk lain,
	// supaya baris baru tidak menabraknya.
	taken := []string{"1", "3"}
	require.Equal(t, "4", masterpenolakan.NextSequence(taken, masterpenolakan.FormatID))
}

func TestFormatIDHasNoPaddingLikeLegacyProcedure(t *testing.T) {
	// Angka polos tanpa awalan, dari `MASTERPENOLAKANKLAIM1.prc:6` dan `..2.prc:6` yang
	// keduanya memperlakukan kolom teks itu sebagai angka lewat to_number. Bentuk "01"
	// justru akan menyimpang darinya.
	require.Equal(t, "7", masterpenolakan.FormatID(7))
	require.Equal(t, "7", masterpenolakan.FormatID2(7))
}
