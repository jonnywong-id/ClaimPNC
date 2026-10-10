package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
)

// storeDenganDokumen membentuk antrean Travel beserta tiga lampiran di dua kategori.
func storeTigaDokumen(t *testing.T) *memory.Store {
	t.Helper()

	store := travelStore()
	store.SeedDocuments(claimKey,
		inboxcompliance.Document{ID: "1", Name: "a.pdf", Category: "10064", StorageID: "IMG-1"},
		inboxcompliance.Document{ID: "2", Name: "b.pdf", Category: "10064", StorageID: "IMG-2"},
		inboxcompliance.Document{ID: "3", Name: "c.pdf", Category: "10065", StorageID: "IMG-3"},
	)
	return store
}

// "Lihat dokumen" hanya menyerahkan lampiran pada kategori yang ditekan.
func TestDaftarDokumenDisaringMenurutKategori(t *testing.T) {
	t.Parallel()

	service := newService(t, storeTigaDokumen(t))

	dokumen, err := service.ListDocumentsInCategory(
		context.Background(), portal, claimKey, "10064")
	require.NoError(t, err)
	require.Len(t, dokumen, 2)
	require.Equal(t, "a.pdf", dokumen[0].Name)
	require.Equal(t, "b.pdf", dokumen[1].Name)
}

// Kategori KOSONG berarti seluruh lampiran klaim — dipakai tombol Ubah Kategori.
//
// Bukan kelonggaran yang tidak disengaja: dokumen yang kategorinya belum terdaftar di
// master tetap harus dapat dipindahkan, dan ia tidak akan muncul pada penyaringan apa pun.
func TestKategoriKosongMenyerahkanSeluruhDokumen(t *testing.T) {
	t.Parallel()

	service := newService(t, storeTigaDokumen(t))

	dokumen, err := service.ListDocumentsInCategory(
		context.Background(), portal, claimKey, "")
	require.NoError(t, err)
	require.Len(t, dokumen, 3)
}

// Kategori yang tidak punya lampiran menjawab senarai kosong, bukan galat.
func TestKategoriTanpaLampiranBukanGalat(t *testing.T) {
	t.Parallel()

	service := newService(t, storeTigaDokumen(t))

	dokumen, err := service.ListDocumentsInCategory(
		context.Background(), portal, claimKey, "99999")
	require.NoError(t, err)
	require.Empty(t, dokumen)
}

// Pemindahan kategori mengubah barisnya, dan menulis jejaknya.
func TestUbahKategoriMemindahkanDanMenulisJejak(t *testing.T) {
	t.Parallel()

	store := storeTigaDokumen(t)
	service := newService(t, store)
	ctx := context.Background()

	require.NoError(t, service.ChangeDocumentCategory(
		ctx, portal, usecaseCaller(), claimKey, "1", "10065"))

	// Barisnya benar-benar pindah: kategori asal berkurang, tujuan bertambah.
	asal, err := service.ListDocumentsInCategory(ctx, portal, claimKey, "10064")
	require.NoError(t, err)
	require.Len(t, asal, 1)

	tujuan, err := service.ListDocumentsInCategory(ctx, portal, claimKey, "10065")
	require.NoError(t, err)
	require.Len(t, tujuan, 2)

	// Dan jejaknya tertulis, menyebut kedua kategorinya.
	riwayat := store.History(claimKey)
	require.Len(t, riwayat, 1)
	require.Contains(t, riwayat[0].Note, "a.pdf")
	require.Contains(t, riwayat[0].Note, "10064")
	require.Contains(t, riwayat[0].Note, "10065")
	require.NotEmpty(t, riwayat[0].By)
}

/*
Memindahkan ke kategori yang SUDAH sama bukan galat, dan bukan pekerjaan.

Dua hal diuji sekaligus, dan keduanya penting:

  - tidak ditolak — petugas yang menekan tombol dua kali tidak boleh dapat pesan salah
  - tidak menulis jejak — jejak untuk perpindahan yang tidak terjadi mengotori
    satu-satunya kontrol pengimbang yang kita punya (`D-59`)
*/
func TestUbahKeKategoriYangSamaTidakMenulisJejak(t *testing.T) {
	t.Parallel()

	store := storeTigaDokumen(t)
	service := newService(t, store)
	ctx := context.Background()

	require.NoError(t, service.ChangeDocumentCategory(
		ctx, portal, usecaseCaller(), claimKey, "1", "10064"))

	riwayat := store.History(claimKey)
	require.Empty(t, riwayat, "perpindahan yang tidak terjadi tidak boleh berjejak")
}

// Dokumen yang tidak ada pada klaim ini ditolak — dan tidak berjejak.
func TestUbahKategoriDokumenAsingDitolak(t *testing.T) {
	t.Parallel()

	store := storeTigaDokumen(t)
	service := newService(t, store)
	ctx := context.Background()

	err := service.ChangeDocumentCategory(
		ctx, portal, usecaseCaller(), claimKey, "9999", "10065")
	require.ErrorIs(t, err, inboxcompliance.ErrDocumentNotFound)

	riwayat := store.History(claimKey)
	require.Empty(t, riwayat)
}

// Kategori tujuan kosong ditolak sebagai validasi, bukan diteruskan ke basis data.
func TestUbahKategoriTanpaTujuanDitolak(t *testing.T) {
	t.Parallel()

	service := newService(t, storeTigaDokumen(t))

	err := service.ChangeDocumentCategory(
		context.Background(), portal, usecaseCaller(), claimKey, "1", "")
	require.Error(t, err)

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
}
