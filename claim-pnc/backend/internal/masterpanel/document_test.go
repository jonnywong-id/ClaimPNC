package masterpanel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

// berkas menyusun satu permintaan unggah yang sah, supaya setiap uji hanya menyebut
// bagian yang sedang diujinya.
func berkas() masterpanel.DocumentUpload {
	return masterpanel.DocumentUpload{
		PanelID: "01000001",
		Note:    "Foto panel depan",
		File: masterpanel.DocumentFile{
			Portal:   "asm",
			FileName: "panel.pdf",
			Content:  []byte("isi"),
			By:       "PETUGAS",
		},
	}
}

func TestPermintaanUnggahYangSahLolos(t *testing.T) {
	require.NoError(t, berkas().CleanDocument().CheckDocument())
}

func TestBerkasKosongDitolak(t *testing.T) {
	u := berkas()
	u.File.Content = nil

	err := u.CleanDocument().CheckDocument()

	var invalid *masterpanel.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Len(t, invalid.Violation, 1)
	require.Equal(t, "berkas", invalid.Violation[0].Field)
}

func TestTanpaNamaBerkasDitolak(t *testing.T) {
	u := berkas()
	u.File.FileName = "   "

	err := u.CleanDocument().CheckDocument()

	require.Error(t, err)
	require.Contains(t, err.Error(), "")
}

// Seluruh pelanggaran dikumpulkan, bukan berhenti pada yang pertama — aturan yang sama
// dengan validasi panel, dan alasannya sama: Pega menampilkan semuanya sekaligus.
func TestSeluruhPelanggaranDikumpulkan(t *testing.T) {
	u := masterpanel.DocumentUpload{
		Note: strings.Repeat("x", masterpanel.MaxDocumentNoteLength+1),
	}

	err := u.CleanDocument().CheckDocument()

	var invalid *masterpanel.ValidationError
	require.ErrorAs(t, err, &invalid)

	field := map[string]bool{}
	for _, v := range invalid.Violation {
		field[v.Field] = true
	}
	require.True(t, field["id"], "panel yang tidak dikenali harus dilaporkan")
	require.True(t, field["berkas"], "berkas yang hilang harus dilaporkan")
	require.True(t, field["catatan"], "catatan yang kepanjangan harus dilaporkan")
}

func TestCatatanTepatDiBatasDiterima(t *testing.T) {
	u := berkas()
	u.Note = strings.Repeat("x", masterpanel.MaxDocumentNoteLength)

	require.NoError(t, u.CleanDocument().CheckDocument())
}

// CleanDocument memangkas spasi tetapi TIDAK membersihkan nama berkas. Pembersihan nama
// adalah aturan modul dokumen penunjang; menirunya di sini akan membuat dua tempat
// memutuskan hal yang sama dengan hasil yang dapat berbeda.
func TestNamaBerkasHanyaDipangkas(t *testing.T) {
	u := berkas()
	u.File.FileName = "  Foto Kerugian.pdf  "

	cleaned := u.CleanDocument()

	require.Equal(t, "Foto Kerugian.pdf", cleaned.File.FileName,
		"spasi dipangkas, tetapi nama dan ekstensinya dibiarkan utuh")
}

func TestGalatUnggahMembawaGolonganDanPesan(t *testing.T) {
	err := &masterpanel.DocumentUploadError{
		Kind:    masterpanel.UploadHalfDone,
		Message: "Berkas sudah terkirim tetapi catatannya gagal disimpan.",
	}

	require.Equal(t, masterpanel.UploadHalfDone, err.Kind)
	require.Equal(t, "Berkas sudah terkirim tetapi catatannya gagal disimpan.", err.Error())
}
