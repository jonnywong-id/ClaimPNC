package masterbengkel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
)

func uploadViolations(t *testing.T, input masterbengkel.UploadInput) map[string]string {
	t.Helper()
	err := input.Clean().Check()
	if err == nil {
		return map[string]string{}
	}
	var validation *masterbengkel.ValidationError
	require.ErrorAs(t, err, &validation)
	result := map[string]string{}
	for _, v := range validation.Violation {
		result[v.Field] = v.Message
	}
	return result
}

// Unggahan yang sah tidak punya pelanggaran.
func TestAValidUploadPasses(t *testing.T) {
	require.Empty(t, uploadViolations(t, masterbengkel.UploadInput{
		WorkshopID: " 010000000001 ", FileName: " bukti.PDF ", Content: []byte("isi"),
	}))
}

// Nama kosong dan isi kosong dilaporkan bersamaan.
func TestAnEmptyUploadReportsBothViolations(t *testing.T) {
	violations := uploadViolations(t, masterbengkel.UploadInput{FileName: "  "})
	require.Equal(t, "Nama berkas wajib ada.", violations["nama_berkas"])
	require.Equal(t, "Berkas kosong.", violations["berkas"])
}

// Nama terlalu panjang, jenis tak diizinkan, dan berkas terlalu besar ditolak.
func TestUploadLimitsAreEnforced(t *testing.T) {
	long := uploadViolations(t, masterbengkel.UploadInput{
		FileName: strings.Repeat("n", masterbengkel.MaxDocumentNameLength) + ".pdf",
		Content:  []byte("x"),
	})
	require.Contains(t, long["nama_berkas"], "paling panjang")

	kind := uploadViolations(t, masterbengkel.UploadInput{
		FileName: "skrip.exe", Content: []byte("x")})
	require.Equal(t,
		"Jenis berkas tidak diizinkan. Yang diterima: csv, jpeg, jpg, pdf, png, xls, xlsx.",
		kind["nama_berkas"])

	large := uploadViolations(t, masterbengkel.UploadInput{
		FileName: "besar.pdf", Content: make([]byte, masterbengkel.MaxDocumentBytes+1)})
	require.Equal(t, "Ukuran berkas melebihi batas 10 MB.", large["berkas"])
}

// Tipe media diambil dari akhiran berkas, bukan dari klien.
func TestDocumentMimeTypeFollowsTheExtension(t *testing.T) {
	require.Equal(t, ".pdf", masterbengkel.DocumentExtension(" Bukti.PDF "))
	require.Equal(t, "application/pdf", masterbengkel.DocumentMimeType("bukti.pdf"))
	require.Equal(t, "image/jpeg", masterbengkel.DocumentMimeType("foto.JPEG"))
	require.Equal(t, "", masterbengkel.DocumentMimeType("skrip.exe"))
}

// DATAID = dua digit tahun + nomor urut sepuluh digit, tanpa pemotongan.
func TestComposeDocumentID(t *testing.T) {
	require.Equal(t, "260000000042", masterbengkel.ComposeDocumentID(" 26 ", 42))
	require.Equal(t, "2612345678901", masterbengkel.ComposeDocumentID("26", 12345678901))
}

// Dokumen warisan tanpa isi dibedakan dari dokumen berisi.
func TestHasContent(t *testing.T) {
	require.False(t, masterbengkel.Document{ID: "1"}.HasContent())
	require.True(t, masterbengkel.Document{Content: []byte("x")}.HasContent())
}

// Pesan log galat validasi menyebut setiap isian.
func TestValidationErrorMessage(t *testing.T) {
	err := &masterbengkel.ValidationError{Violation: []masterbengkel.Violation{
		{Field: "a", Message: "satu"}, {Field: "b", Message: "dua"},
	}}
	require.Equal(t, "masterbengkel: isian tidak sah (a: satu; b: dua)", err.Error())
}

// Isian opsional yang melebihi batas tetap ditolak.
func TestAnOverlongOptionalFieldIsRejected(t *testing.T) {
	input := validInput()
	input.Address = strings.Repeat("A", 5000)

	err := input.Clean().Check()
	var validation *masterbengkel.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violation, 1)
	require.Contains(t, validation.Violation[0].Message, "paling panjang")
}
