package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
)

// storePalsu merekam urutan panggilan, bukan hanya nilainya.
type storePalsu struct {
	unggah      []inboxcompliance.UploadRequest
	hapus       []inboxcompliance.DeleteRequest
	imageID     string
	galatUnggah error
	galatHapus  error
}

func (s *storePalsu) Upload(
	_ context.Context, r inboxcompliance.UploadRequest,
) (inboxcompliance.UploadResult, error) {
	s.unggah = append(s.unggah, r)
	if s.galatUnggah != nil {
		return inboxcompliance.UploadResult{}, s.galatUnggah
	}
	return inboxcompliance.UploadResult{StorageID: s.imageID}, nil
}

func (s *storePalsu) Delete(_ context.Context, r inboxcompliance.DeleteRequest) error {
	s.hapus = append(s.hapus, r)
	return s.galatHapus
}

func serviceDenganStore(
	t *testing.T, store *memory.Store, berkas inboxcompliance.DocumentStore,
) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxcompliance.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
		Clock: fixedClock{at: now},
		DocumentStoreSelector: func(alias string) (inboxcompliance.DocumentStore, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return berkas, nil
		},
	})
	require.NoError(t, err)
	return service
}

func unggahContoh() usecase.UploadInput {
	return usecase.UploadInput{
		Reference: claimKey,
		FileName:  "Surat Keterangan.pdf",
		Extension: "pdf",
		Category:  "Dokumen Pendukung",
		Content:   []byte("isi berkas"),
	}
}

// Unggah menempatkan berkas DULU, baru mencatat barisnya.
//
// Barisnya membawa `ImageID` yang diterbitkan layanan — tanpa itu dokumen tampil di grid
// tetapi tidak pernah dapat dibuka.
func TestUnggahMencatatBarisDenganKunciPenyimpanan(t *testing.T) {
	t.Parallel()

	store := paStore()
	berkas := &storePalsu{imageID: "IMG-555"}
	service := serviceDenganStore(t, store, berkas)

	doc, err := service.UploadDocument(
		context.Background(), portal, usecaseCaller(), unggahContoh())
	require.NoError(t, err)

	require.Len(t, berkas.unggah, 1)
	require.Equal(t, "Surat Keterangan.pdf", berkas.unggah[0].FileName)
	require.Equal(t, "pdf", berkas.unggah[0].MimeTypeHint)
	require.NotEmpty(t, berkas.unggah[0].UserInput)

	require.Equal(t, "IMG-555", doc.StorageID)
	require.NotEmpty(t, doc.ID, "DATAID harus terbit dari penyimpanan")

	// Dan barisnya benar-benar tercatat pada klaim.
	//
	// Diperiksa lewat repo, bukan lewat OpenChecker: sejak grid dokumennya dicabut
	// (2026-10-08), form tidak lagi membaca dokumen saat dibuka. Yang diuji di sini
	// adalah PENCATATANNYA — dan itu tidak berubah.
	dokumen, err := store.FindDocuments(context.Background(), claimKey)
	require.NoError(t, err)
	require.Len(t, dokumen, 1)
	require.Equal(t, "Surat Keterangan.pdf", dokumen[0].Name)
}

// Layanan gagal berarti TIDAK ada baris yang tercatat.
//
// Baris yang menunjuk berkas yang tidak pernah ada akan tampil di grid dengan tombol
// Lihat yang selalu gagal — itu yang dicegah urutan "layanan dulu".
func TestUnggahGagalTidakMencatatBaris(t *testing.T) {
	t.Parallel()

	store := paStore()
	berkas := &storePalsu{galatUnggah: errors.New("layanan menjawab 503")}
	service := serviceDenganStore(t, store, berkas)

	_, err := service.UploadDocument(
		context.Background(), portal, usecaseCaller(), unggahContoh())
	require.Error(t, err)

	dokumen, err := store.FindDocuments(context.Background(), claimKey)
	require.NoError(t, err)
	require.Empty(t, dokumen, "tidak boleh ada baris tanpa berkasnya")
}

// Klaim yang TIDAK di antrean Compliance tidak dapat ditempeli berkas.
func TestUnggahKeKlaimDiLuarAntreanDitolak(t *testing.T) {
	t.Parallel()

	berkas := &storePalsu{imageID: "IMG-1"}
	service := serviceDenganStore(t, paStore(), berkas)

	input := unggahContoh()
	input.Reference = "ASM-FW-GCNMFW-WORK PNC-9999"

	_, err := service.UploadDocument(
		context.Background(), portal, usecaseCaller(), input)
	require.ErrorIs(t, err, inboxcompliance.ErrClaimNotInQueue)
	require.Empty(t, berkas.unggah, "layanan tidak boleh dihubungi sama sekali")
}

// Hapus menulis RIWAYAT lebih dulu, lalu baris, lalu berkasnya.
//
// Urutannya yang diuji, bukan sekadar hasilnya: riwayat yang ditulis belakangan dan gagal
// berarti jejak penghapusan yang sudah terjadi hilang, dan itu tidak dapat dipulihkan.
func TestHapusMenulisJejakSebelumMenghapus(t *testing.T) {
	t.Parallel()

	store := paStore()
	berkas := &storePalsu{imageID: "IMG-777"}
	service := serviceDenganStore(t, store, berkas)
	ctx := context.Background()

	doc, err := service.UploadDocument(ctx, portal, usecaseCaller(), unggahContoh())
	require.NoError(t, err)

	require.NoError(t, service.DeleteDocument(
		ctx, portal, usecaseCaller(), claimKey, doc.ID))

	// Jejaknya ada, menyebut nama berkasnya dan pelakunya.
	riwayat := store.History(claimKey)
	require.Len(t, riwayat, 1)
	require.Contains(t, riwayat[0].Note, "Surat Keterangan.pdf")
	require.NotEmpty(t, riwayat[0].By)

	// Barisnya hilang — penghapusan di tabel ini FISIK.
	dokumen, err := store.FindDocuments(ctx, claimKey)
	require.NoError(t, err)
	require.Empty(t, dokumen)

	// Dan berkasnya ikut dihapus dari penyimpanan, dengan kunci yang benar.
	require.Len(t, berkas.hapus, 1)
	require.Equal(t, "IMG-777", berkas.hapus[0].ObjectPath)
}

// Dokumen milik klaim LAIN tidak dapat dihapus, dan tidak meninggalkan jejak palsu.
func TestHapusDokumenKlaimLainDitolak(t *testing.T) {
	t.Parallel()

	store := paStore()
	berkas := &storePalsu{imageID: "IMG-1"}
	service := serviceDenganStore(t, store, berkas)

	err := service.DeleteDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "9999")
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentNotFound)

	require.Empty(t, store.History(claimKey), "jejak tidak boleh ditulis untuk penghapusan yang ditolak")
	require.Empty(t, berkas.hapus, "penyimpanan tidak boleh disentuh")
}

// Tanpa layanan dokumen, keduanya menjawab galat yang menyebut sebabnya.
func TestUnggahDanHapusTanpaLayananDokumen(t *testing.T) {
	t.Parallel()

	service := newService(t, paStore()) // tanpa DocumentStoreSelector
	ctx := context.Background()

	_, err := service.UploadDocument(ctx, portal, usecaseCaller(), unggahContoh())
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentServiceMissing)

	err = service.DeleteDocument(ctx, portal, usecaseCaller(), claimKey, "1")
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentServiceMissing)
}
