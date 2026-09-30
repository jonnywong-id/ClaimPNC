package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	"claim-pnc/internal/masterbengkel/usecase"
)

// fixedClock memaku waktunya supaya DATAID dan waktu unggah dapat disebutkan di uji.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

func documentService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return repo, nil },
		Clock:        fixedClock{at: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)},
	})
	require.NoError(t, err)
	return service, repo
}

// firstWorkshopID mengambil satu ID yang benar-benar ada di contoh, bukan yang dikarang.
func firstWorkshopID(t *testing.T, repo *memory.Repo) string {
	t.Helper()

	rows, err := repo.List(context.Background(), masterbengkel.Filter{
		Status: masterbengkel.StatusApproved,
	})
	require.NoError(t, err)
	require.NotEmpty(t, rows, "contoh harus punya sekurangnya satu bengkel disetujui")
	return rows[0].ID
}

// DATAID berbentuk sama dengan yang diterbitkan Oracle: dua digit tahun + sepuluh digit
// nomor urut yang diratakan nol (`Database/SET_ATTACHMENT_64BIT.prc:18-26`).
func TestUploadDocumentIssuesPegaShapedID(t *testing.T) {
	service, repo := documentService(t)
	id := firstWorkshopID(t, repo)

	document, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: "adminpnc"},
		masterbengkel.UploadInput{
			WorkshopID: id,
			FileName:   "kerjasama.pdf",
			Content:    []byte("%PDF-1.4 contoh"),
		})
	require.NoError(t, err)

	require.Len(t, document.ID, 12, "dua digit tahun + sepuluh digit nomor urut")
	require.Equal(t, memory.SampleDocumentYear, document.ID[:2])
	require.Equal(t, "0000000001", document.ID[2:])
	require.Equal(t, "application/pdf", document.MimeType,
		"tipe media diturunkan dari akhiran berkas, bukan dari yang dikirim peramban")
	require.Equal(t, "adminpnc", document.UploadedBy)
}

// Unggahan menautkan dirinya ke barisnya, dan dapat dibaca kembali lewat bengkelnya —
// meniru GetIDDokumenBengkel lalu GetAttachmentFromDB_Sql.
func TestUploadedDocumentIsReadableThroughItsWorkshop(t *testing.T) {
	service, repo := documentService(t)
	id := firstWorkshopID(t, repo)

	uploaded, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: "adminpnc"},
		masterbengkel.UploadInput{
			WorkshopID: id,
			FileName:   "npwp.png",
			Content:    []byte("isi berkas"),
		})
	require.NoError(t, err)

	found, err := service.Document(context.Background(), "ASM", id)
	require.NoError(t, err)
	require.Equal(t, uploaded.ID, found.ID)
	require.Equal(t, "npwp.png", found.Name)
	require.True(t, found.HasContent())
}

// BENGKEL_HE hanya punya SATU kolom DOKUMENID, sehingga unggahan berikutnya menggantikan
// tautannya — persis seperti Pega. Baris lampiran lamanya tidak dihapus (`D-66`).
func TestSecondUploadReplacesTheLink(t *testing.T) {
	service, repo := documentService(t)
	id := firstWorkshopID(t, repo)

	first, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: "adminpnc"},
		masterbengkel.UploadInput{WorkshopID: id, FileName: "lama.pdf", Content: []byte("a")})
	require.NoError(t, err)

	second, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: "adminpnc"},
		masterbengkel.UploadInput{WorkshopID: id, FileName: "baru.pdf", Content: []byte("b")})
	require.NoError(t, err)

	found, err := service.Document(context.Background(), "ASM", id)
	require.NoError(t, err)
	require.Equal(t, second.ID, found.ID, "yang tertaut adalah unggahan terakhir")

	// Yang lama tidak dibuang; ia hanya tidak lagi tertaut.
	old, err := repo.FindDocument(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, "lama.pdf", old.Name)
}

// Bengkel yang belum pernah dilampiri dibedakan dari bengkel yang tidak ada.
func TestDocumentDistinguishesMissingAttachmentFromMissingWorkshop(t *testing.T) {
	service, repo := documentService(t)

	_, err := service.Document(context.Background(), "ASM", firstWorkshopID(t, repo))
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)

	_, err = service.Document(context.Background(), "ASM", "010000009999")
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
}

// Berkas ditolak SEBELUM menyentuh penyimpanan, dan seluruh pelanggaran dikembalikan
// sekaligus — kesetaraan perilaku dengan layar lama yang menampilkan semua pesan bersamaan.
func TestUploadRejectsUnacceptableFiles(t *testing.T) {
	service, repo := documentService(t)
	id := firstWorkshopID(t, repo)

	for name, input := range map[string]masterbengkel.UploadInput{
		"jenis tidak diizinkan": {FileName: "skrip.exe", Content: []byte("x")},
		"berkas kosong":         {FileName: "kosong.pdf", Content: nil},
		"melebihi batas":        {FileName: "besar.pdf", Content: make([]byte, masterbengkel.MaxDocumentBytes+1)},
	} {
		t.Run(name, func(t *testing.T) {
			input.WorkshopID = id
			_, err := service.UploadDocument(context.Background(), "ASM",
				usecase.Actor{Login: "adminpnc"}, input)

			var validationError *masterbengkel.ValidationError
			require.ErrorAs(t, err, &validationError)
			require.NotEmpty(t, validationError.Violation)
		})
	}

	// Tidak satu pun yang tersimpan.
	_, err := service.Document(context.Background(), "ASM", id)
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)
}

// Nama berkas yang membawa jalur dipangkas menjadi nama dasarnya oleh transport; di sini
// yang dijaga adalah pemetaan tipe medianya, yang TIDAK memercayai kiriman peramban.
func TestDocumentMimeTypeComesFromExtension(t *testing.T) {
	require.Equal(t, "image/jpeg", masterbengkel.DocumentMimeType("FOTO.JPG"))
	require.Equal(t, "application/pdf", masterbengkel.DocumentMimeType("berkas.pdf"))
	require.Equal(t, "", masterbengkel.DocumentMimeType("skrip.exe"))
}

// Pesan jenis berkas menyebut daftar yang diterima, dan urutannya tetap — daftar yang
// berubah-ubah antarpermintaan membuat pesannya tidak dapat diuji maupun diterjemahkan.
func TestAllowedExtensionMessageIsStable(t *testing.T) {
	input := masterbengkel.UploadInput{FileName: "skrip.exe", Content: []byte("x")}

	var first string
	for i := 0; i < 5; i++ {
		err := input.Check()
		var validationError *masterbengkel.ValidationError
		require.ErrorAs(t, err, &validationError)

		message := validationError.Violation[0].Message
		require.True(t, strings.Contains(message, "pdf"), "pesan menyebut jenis yang diterima")
		if i == 0 {
			first = message
			continue
		}
		require.Equal(t, first, message, "urutan daftarnya harus tetap")
	}
}
