package inboxsalvage_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// validDocument adalah satu lampiran terkecil yang sah, dipakai sebagai titik awal setiap
// uji yang mengubah satu hal saja.
func validDocument() inboxsalvage.DocumentUpload {
	return inboxsalvage.DocumentUpload{
		ClaimNo:   "PNC-700071",
		SalvageID: "4411",
		FileName:  "foto.pdf",
		MimeType:  "pdf",
		Content64: "aXNpIGJlcmthcw==",
		Operator:  "90000071",
	}
}

func TestDocumentNeedsAClaimANameAndItsContents(t *testing.T) {
	require.NoError(t, validDocument().Validate())

	kosongKlaim := validDocument()
	kosongKlaim.ClaimNo = ""
	require.Error(t, kosongKlaim.Validate())

	kosongNama := validDocument()
	kosongNama.FileName = ""
	require.Error(t, kosongNama.Validate())

	kosongIsi := validDocument()
	kosongIsi.Content64 = ""
	require.Error(t, kosongIsi.Validate())
}

// Pengajuan yang belum punya nomor TETAP sah.
//
// Pada form pengajuan baru, berkas diunggah sebelum Submit ditekan, sehingga penaut ke
// pengajuannya memang belum dapat ditulis. Menolaknya berarti memaksa pengguna menyimpan
// pengajuan kosong lebih dulu hanya untuk dapat melampirkan berkas.
func TestDocumentWithoutASalvageIDIsStillValid(t *testing.T) {
	tanpaPengajuan := validDocument()
	tanpaPengajuan.SalvageID = ""
	require.NoError(t, tanpaPengajuan.Validate())
}

// Nomor klaim dibesarkan hurufnya, karena kunci warisan membandingkannya apa adanya.
//
// `CLAIMID` di tabel warisan berbentuk `'ASM-FW-GCNMFW-WORK ' || nomor`, dan layar lama
// menyusunnya lewat `@toUpperCase`. Nomor berhuruf kecil yang dibiarkan apa adanya tidak
// akan cocok dengan satu baris pun — dan tidak akan memunculkan galat apa pun.
func TestDocumentCleanUppercasesTheClaimNumber(t *testing.T) {
	kotor := validDocument()
	kotor.ClaimNo = "  pnc-700071  "

	require.Equal(t, "PNC-700071", kotor.Clean().ClaimNo)
}

// Titik di depan ekstensi DIBUANG, karena penyusun nama menambahkannya sendiri.
//
// Peramban mengirimkan `.pdf` pada sebagian berkas dan `pdf` pada sebagian lain.
// Membiarkannya apa adanya menghasilkan nama berakhiran `..pdf` pada yang pertama.
func TestDocumentCleanDropsTheLeadingDotOnTheExtension(t *testing.T) {
	bertitik := validDocument()
	bertitik.MimeType = ".PDF"

	require.Equal(t, "pdf", bertitik.Clean().MimeType)
}

// Kedua batas yang tertulis merah di modal adalah batas yang BERLAKU, bukan hiasan.
func TestTheTwoUploadLimitsAreTheOnesWrittenOnTheScreen(t *testing.T) {
	require.Equal(t, 5, inboxsalvage.MaxDocumentPerUpload)
	require.Equal(t, 1024*1024, inboxsalvage.MaxDocumentSizeBytes)
}

// NewImageID menghasilkan bentuk yang sama dengan yang disusun Oracle.
//
// Yang diperiksa BENTUKNYA — 32 heksadesimal huruf besar — bukan nilainya, karena
// nilainya bergantung cap waktu. Bentuk inilah yang menentukan apakah kunci baru dapat
// hidup berdampingan dengan kunci yang sudah ada di tabel.
func TestImageIDKeepsTheShapeOracleProduced(t *testing.T) {
	id := inboxsalvage.NewImageID(time.Date(2026, 10, 3, 9, 15, 0, 0, time.UTC))

	require.Len(t, id, 32)
	require.Equal(t, strings.ToUpper(id), id)
	require.Regexp(t, `^[0-9A-F]{32}$`, id)
}

// Dua berkas yang diunggah pada detik yang sama TIDAK boleh berbagi IMAGEID.
//
// `IMAGEID` yang menautkan keterangan dengan isinya; dua berkas yang berbagi satu kunci
// berarti satu di antaranya menunjuk isi berkas yang bukan miliknya — dan tidak ada galat
// yang muncul, hanya dokumen yang salah saat dibuka.
func TestImageIDDiffersForDifferentInstants(t *testing.T) {
	awal := time.Date(2026, 10, 3, 9, 15, 0, 0, time.UTC)

	require.NotEqual(
		t,
		inboxsalvage.NewImageID(awal),
		inboxsalvage.NewImageID(awal.Add(time.Millisecond)),
	)
}

// SafeFileName membuang apa pun yang dapat berpindah makna di dalam nama berkas.
//
// Nama berkas datang dari pengguna dan berakhir di dalam nama yang tersimpan. Yang
// dijaganya bukan kerapian melainkan PEMISAH JALUR: tanpa itu, sebuah nama dapat membawa
// berkasnya keluar dari tempat yang dimaksud.
func TestSafeFileNameKeepsOnlyHarmlessCharacters(t *testing.T) {
	require.Equal(t, "laporan-survei.pdf", inboxsalvage.SafeFileName("laporan-survei.pdf"))
	require.Equal(t, "fotorusak.jpg", inboxsalvage.SafeFileName("foto rusak.jpg"))

	// Titiknya DIPERTAHANKAN — ia bagian sah dari ekstensi — tetapi kedua pemisah
	// jalurnya hilang, sehingga yang tersisa tidak lagi menunjuk ke mana pun.
	dibersihkan := inboxsalvage.SafeFileName("../../etc/passwd")
	require.NotContains(t, dibersihkan, "/")
	require.NotContains(t, dibersihkan, `\`)
	require.Equal(t, "....etcpasswd", dibersihkan)
}
