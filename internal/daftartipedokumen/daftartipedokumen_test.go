package daftartipedokumen_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftartipedokumen"
)

// Lebar nomor urut EMPAT, bukan tiga seperti Master Status Klaim dan bukan lima seperti
// Master Dokumen Travel.
//
// Uji ini ada karena ketiga modul memakai pola yang sama dengan lebar berbeda, dan menyalin
// lebar dari modul tetangga adalah kekeliruan yang paling mungkin terjadi — sekaligus yang
// paling sulit terlihat, karena ID yang dihasilkan tetap "masuk akal" di layar padahal
// tidak dikenali data historis.
func TestIDFollowsTheFourDigitShapeOfTheOldProcedure(t *testing.T) {
	require.Equal(t, 4, daftartipedokumen.SequenceDigits)

	require.Equal(t, "10001", daftartipedokumen.FormatID("1", 1))
	require.Equal(t, "10042", daftartipedokumen.FormatID("1", 42))
	require.Equal(t, "19999", daftartipedokumen.FormatID("1", 9999))
}

// Nomor di atas 9999 dikembalikan APA ADANYA, tidak dipotong — sama seperti LPAD Oracle.
//
// Memotongnya akan menghasilkan ID GANDA, dan ID ganda di sini berarti dua tipe dokumen
// berbeda berbagi satu kunci yang dirujuk dua master turunan. Penyisipan yang gagal dengan
// pesan jelas jauh lebih baik daripada itu.
func TestSequenceBeyondFourDigitsIsNotTruncated(t *testing.T) {
	require.Equal(t, "110000", daftartipedokumen.FormatID("1", 10000))
	require.Equal(t, "1123456", daftartipedokumen.FormatID("1", 123456))
}

// Kode situs ikut dipangkas: ia dibaca dari kolom yang dapat berisi padding.
func TestSiteCodeIsTrimmedBeforeBeingJoined(t *testing.T) {
	require.Equal(t, "10001", daftartipedokumen.FormatID(" 1 ", 1))
}

// Clean memangkas spasi tepi kedua isian, dan TIDAK melakukan apa pun selain itu.
func TestCleanTrimsBothFieldsAndNothingElse(t *testing.T) {
	cleaned := daftartipedokumen.Input{
		Type:          "  Dokumen Survey  ",
		ProcessStatus: "\tSurvey\n",
	}.Clean()

	require.Equal(t, "Dokumen Survey", cleaned.Type)
	require.Equal(t, "Survey", cleaned.ProcessStatus)
}

// Isian kosong TETAP kosong setelah Clean — ia tidak diganti nilai bawaan apa pun.
//
// Uji ini mengunci keputusan Work Owner 2026-09-21 pada tingkat domain: layar ini meniru
// Pega apa adanya, tanpa validasi. `Section/BrowseListDocumentType-Section.xml` tidak memuat
// satu pun `pyRequired=true`, dan `Database/PEGA_LST_DOC_TYPE.prc` menyisipkan tanpa
// memeriksa apa pun.
//
// Bila kelak layar ini diperketat, uji inilah yang akan gagal lebih dulu — dan itu memang
// gunanya: perubahan kebijakan harus terlihat, bukan menyelinap.
func TestEmptyInputIsLeftEmptyBecauseThisScreenHasNoValidation(t *testing.T) {
	cleaned := daftartipedokumen.Input{Type: "   ", ProcessStatus: ""}.Clean()

	require.Empty(t, cleaned.Type)
	require.Empty(t, cleaned.ProcessStatus)
}

// Modul ini hanya punya SATU galat domain, dan itu konsekuensi langsung dari "tanpa
// validasi": tidak ada aturan isian yang dapat dilanggar, dan tidak ada keunikan yang dapat
// bentrok.
func TestTheOnlyDomainErrorIsNotFound(t *testing.T) {
	require.EqualError(t, daftartipedokumen.ErrNotFound,
		"daftartipedokumen: tipe dokumen tidak ditemukan")
}
