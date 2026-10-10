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

// linkerPalsu merekam permintaan terakhir dan mengembalikan tautan yang ditentukan uji.
type linkerPalsu struct {
	terakhir inboxcompliance.LinkRequest
	url      string
	galat    error
}

func (l *linkerPalsu) NewLink(
	_ context.Context, request inboxcompliance.LinkRequest,
) (inboxcompliance.DocumentLink, error) {
	l.terakhir = request
	if l.galat != nil {
		return inboxcompliance.DocumentLink{}, l.galat
	}
	return inboxcompliance.DocumentLink{URL: l.url}, nil
}

func serviceDenganLinker(
	t *testing.T, store *memory.Store, linker inboxcompliance.DocumentLinker,
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
		DocumentLinkerSelector: func(alias string) (inboxcompliance.DocumentLinker, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return linker, nil
		},
	})
	require.NoError(t, err)
	return service
}

func storeDenganDokumen() *memory.Store {
	store := paStore()
	store.SeedDocuments(claimKey, inboxcompliance.Document{
		ID:        "4411",
		Name:      "Surat Keterangan.pdf",
		StorageID: "IMG-7781",
	})
	return store
}

// Tautan diterbitkan, lalu DIBUNGKUS penampil Office Online.
//
// Pembungkusnya bukan hiasan: `GetLinkViewDoc_Act` langkah 22 memang membukanya lewat
// `view.officeapps.live.com`, dan itu yang ditiru.
func TestMembukaDokumenMembungkusTautanDenganPenampil(t *testing.T) {
	t.Parallel()

	linker := &linkerPalsu{url: "https://penyimpanan.contoh/berkas?ttd=abc"}
	service := serviceDenganLinker(t, storeDenganDokumen(), linker)

	hasil, err := service.OpenDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "4411")
	require.NoError(t, err)

	require.Equal(t,
		"https://view.officeapps.live.com/op/view.aspx?src="+
			"https%3A%2F%2Fpenyimpanan.contoh%2Fberkas%3Fttd%3Dabc",
		hasil.ViewerURL)
	require.Equal(t, "Surat Keterangan.pdf", hasil.Document.Name)
}

// Permintaan membawa kunci penyimpanan, nama berkas, pelaku, dan Durasi bawaan.
//
// Durasi 3600 bukan angka karangan — ia nilai jatuh-balik Pega ketika master kosong.
func TestPermintaanTautanMembawaIsianYangBenar(t *testing.T) {
	t.Parallel()

	linker := &linkerPalsu{url: "https://penyimpanan.contoh/berkas"}
	service := serviceDenganLinker(t, storeDenganDokumen(), linker)

	_, err := service.OpenDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "4411")
	require.NoError(t, err)

	require.Equal(t, "IMG-7781", linker.terakhir.StorageID)
	require.Equal(t, "Surat Keterangan.pdf", linker.terakhir.FileName)
	require.NotEmpty(t, linker.terakhir.UserInput, "pelaku harus ikut terkirim")
	require.Equal(t, 3600, linker.terakhir.DurationSeconds)
}

// Dokumen milik klaim LAIN ditolak — dan inilah uji terpenting di berkas ini.
//
// Layanan penyimpanan di seberang `pyUseAuthentication = false`: ia menerbitkan tautan
// untuk kunci apa pun yang dikirimkan, tanpa memeriksa kepemilikan. Jadi satu-satunya
// yang menahan pengambilan berkas milik klaim lain adalah pemeriksaan DI SINI.
func TestDokumenKlaimLainDitolak(t *testing.T) {
	t.Parallel()

	linker := &linkerPalsu{url: "https://penyimpanan.contoh/berkas"}
	service := serviceDenganLinker(t, storeDenganDokumen(), linker)

	_, err := service.OpenDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "9999")
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentNotFound)

	require.Empty(t, linker.terakhir.StorageID,
		"layanan dokumen TIDAK boleh dihubungi untuk dokumen yang bukan milik klaim ini")
}

// Tanpa layanan dokumen terkonfigurasi, galatnya menyebut sebabnya — bukan galat internal.
func TestTanpaLayananDokumenGalatnyaJelas(t *testing.T) {
	t.Parallel()

	service := newService(t, storeDenganDokumen()) // tanpa DocumentLinkerSelector

	_, err := service.OpenDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "4411")
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentServiceMissing)
}

// Kegagalan layanan dibawa naik, tidak disamarkan sebagai dokumen tidak ditemukan.
func TestKegagalanLayananDokumenDibawaNaik(t *testing.T) {
	t.Parallel()

	linker := &linkerPalsu{galat: errors.New("layanan dokumen menjawab 503")}
	service := serviceDenganLinker(t, storeDenganDokumen(), linker)

	_, err := service.OpenDocument(
		context.Background(), portal, usecaseCaller(), claimKey, "4411")
	require.Error(t, err)
	require.NotErrorIs(t, err, inboxcompliance.ErrDocumentNotFound)
}
