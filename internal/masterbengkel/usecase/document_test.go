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

// fakeUploader berdiri di tempat layanan penyimpanan internal.
//
// Ia mencatat apa yang diterimanya supaya uji dapat menegaskan bahwa yang dikirim ke
// penyimpanan memang berkas yang dipilih pengguna — bukan hanya bahwa unggahannya berhasil.
type fakeUploader struct {
	imageID string
	err     error
	seen    []masterbengkel.DocumentFile
}

func (u *fakeUploader) Upload(_ context.Context, f masterbengkel.DocumentFile) (string, error) {
	u.seen = append(u.seen, f)
	if u.err != nil {
		return "", u.err
	}
	if u.imageID == "" {
		return "img-contoh", nil
	}
	return u.imageID, nil
}

func documentService(t *testing.T) (*usecase.Service, *memory.Repo) {
	service, repo, _ := documentServiceWith(t, &fakeUploader{})
	return service, repo
}

func documentServiceWith(
	t *testing.T,
	uploader *fakeUploader,
) (*usecase.Service, *memory.Repo, *fakeUploader) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return repo, nil },
		Clock:        fixedClock{at: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)},
		Uploader:     uploader,
	})
	require.NoError(t, err)
	return service, repo, uploader
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
	require.True(t, found.HasFile())
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

// Isi berkas benar-benar DIKIRIM ke layanan penyimpanan, dan IMAGEID balasannya yang
// tersimpan di barisnya.
//
// Ini uji yang menutup cacat sistem lama secara langsung: di Pega berkasnya dibaca,
// dibawa melintasi tiga rule, lalu jatuh di pemanggilan procedure-nya — dan `IMAGEID`
// tidak pernah terisi, sehingga barisnya menunjuk ke ketiadaan.
func TestUploadSendsTheFileToStorageAndKeepsItsKey(t *testing.T) {
	service, repo, uploader := documentServiceWith(t, &fakeUploader{imageID: "img-9"})
	id := firstWorkshopID(t, repo)

	document, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: " penguji "},
		masterbengkel.UploadInput{WorkshopID: id, FileName: "bukti.pdf",
			Content: []byte("%PDF isi")})
	require.NoError(t, err)

	require.Equal(t, "img-9", document.ImageID)
	require.True(t, document.HasFile())

	require.Len(t, uploader.seen, 1)
	require.Equal(t, "ASM", uploader.seen[0].Portal)
	require.Equal(t, "bukti.pdf", uploader.seen[0].FileName)
	require.Equal(t, []byte("%PDF isi"), uploader.seen[0].Content)
	require.Equal(t, "penguji", uploader.seen[0].By)
}

// Unggahan yang gagal TIDAK meninggalkan baris lampiran.
//
// Urutannya yang menjamin itu: isi berkas pergi lebih dulu, dan DATAID baru diterbitkan
// sesudah layanan penyimpanan menjawab. Baris tanpa berkas adalah keadaan yang modul ini
// ada untuk mencegahnya.
func TestFailedUploadLeavesNoRow(t *testing.T) {
	gagal := &masterbengkel.DocumentUploadError{
		Kind:    masterbengkel.UploadUnavailable,
		Message: "Layanan penyimpanan dokumen sedang tidak dapat dihubungi.",
	}
	service, repo, _ := documentServiceWith(t, &fakeUploader{err: gagal})
	id := firstWorkshopID(t, repo)

	_, err := service.UploadDocument(context.Background(), "ASM", usecase.Actor{Login: "penguji"},
		masterbengkel.UploadInput{WorkshopID: id, FileName: "bukti.pdf", Content: []byte("x")})

	var upload *masterbengkel.DocumentUploadError
	require.ErrorAs(t, err, &upload)
	require.Equal(t, masterbengkel.UploadUnavailable, upload.Kind)

	_, err = service.Document(context.Background(), "ASM", id)
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)
}

// Tanpa pengunggah, unggahan ditolak dengan sebab yang menyebut pemasangannya — dan layar
// dapat mengetahuinya SEBELUM pengguna memilih berkas.
func TestUploadWithoutAnUploaderIsRejectedUpFront(t *testing.T) {
	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return repo, nil },
	})
	require.NoError(t, err)
	require.False(t, service.UploadAvailable())

	_, err = service.UploadDocument(context.Background(), "ASM", usecase.Actor{Login: "penguji"},
		masterbengkel.UploadInput{WorkshopID: firstWorkshopID(t, repo),
			FileName: "bukti.pdf", Content: []byte("x")})

	var upload *masterbengkel.DocumentUploadError
	require.ErrorAs(t, err, &upload)
	require.Equal(t, masterbengkel.UploadMisconfigured, upload.Kind)
}

// Catatan yang gagal disimpan SESUDAH berkasnya terkirim dinyatakan UploadHalfDone —
// bukan galat biasa yang mengundang pengulangan.
//
// Mengulang di keadaan itu menumpuk berkas ganda di layanan penyimpanan, sebab unggahan
// yang pertama tidak dapat ditarik kembali.
func TestMetadataFailureAfterUploadIsHalfDone(t *testing.T) {
	service := serviceOver(t, storePenyimpanGagal{Repo: memory.NewSampleRepo()})

	_, err := service.UploadDocument(context.Background(), portalAlias, usecase.Actor{},
		masterbengkel.UploadInput{WorkshopID: "010000000001", FileName: "a.pdf",
			Content: []byte("isi")})

	var upload *masterbengkel.DocumentUploadError
	require.ErrorAs(t, err, &upload)
	require.Equal(t, masterbengkel.UploadHalfDone, upload.Kind)
	require.Contains(t, upload.Message, "JANGAN unggah ulang")

	// Galat aslinya tetap terbaca lewat rantai, supaya log tidak kehilangan sebabnya.
	require.ErrorIs(t, err, errOracle)
}
