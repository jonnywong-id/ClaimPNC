package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
)

// simpanDokumen mencatat satu dokumen dengan alamat dan masa berlaku tertentu.
func simpanDokumen(t *testing.T, repo interface {
	Simpan(context.Context, dokumenpenunjang.Document) error
}, url string, exp *time.Time) {
	t.Helper()
	require.NoError(t, repo.Simpan(context.Background(), dokumenpenunjang.Document{
		ImageID:     "IMG-9",
		FileName:    "Revisi023pdf",
		URL:         url,
		ExpiresAt:   exp,
		Folder:      "gs://ember-uji-contoh/Doc/2026/10/Revisi023pdf",
		ClaimNumber: "PNCN.26.27",
	}))
}

// Alamat yang masih berlaku dipakai apa adanya — GetLinkViewDoc_Act langkah 7.
func TestTautanBerlakuTidakDiperpanjang(t *testing.T) {
	sekarang := time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC)
	service, repo, storage := bangun(t, sekarang)
	berlaku := sekarang.Add(time.Hour)
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", &berlaku)

	dokumen, err := service.Tautan(context.Background(), portal, "IMG-9", "JONNY")
	require.NoError(t, err)
	require.Equal(t, "https://penyimpanan.contoh/lama", dokumen.URL)
	require.Empty(t, storage.Tautan())
}

// Alamat kedaluwarsa diperpanjang lewat NewLinkDokumenPNC dengan halaman DocAPI yang sama
// seperti Pega, lalu disimpan kembali (UpdateNewDocumentPNC).
func TestTautanKedaluwarsaDiperpanjangDanDisimpan(t *testing.T) {
	sekarang := time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC)
	service, repo, storage := bangun(t, sekarang)
	lewat := sekarang.Add(-time.Minute)
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", &lewat)

	dokumen, err := service.Tautan(context.Background(), portal, "IMG-9", "JONNY")
	require.NoError(t, err)
	require.Equal(t, "https://penyimpanan.contoh/IMG-9?ke=1", dokumen.URL)

	kirim := storage.Tautan()
	require.Len(t, kirim, 1)
	require.Equal(t, "IMG-9", kirim[0].ImageID)
	require.Equal(t, "klaimpnc", kirim[0].NamaAplikasi)
	require.Equal(t, "JONNY", kirim[0].Pengunggah)
	require.Equal(t, "KODE-1", kirim[0].KodeAkses)
	require.Equal(t, "Doc/2026/10/", kirim[0].Folder)
	require.Equal(t, "Revisi023pdf", kirim[0].NamaBerkas)
	require.Equal(t, 3600, kirim[0].Durasi)

	tersimpan, err := repo.Ambil(context.Background(), "IMG-9")
	require.NoError(t, err)
	require.Equal(t, dokumen.URL, tersimpan.URL)
}

// Masa berlaku yang tidak tercatat diperpanjang, seperti @CompareDates dengan tanggal kosong.
func TestTautanTanpaMasaBerlakuDiperpanjang(t *testing.T) {
	service, repo, storage := bangun(t, time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC))
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", nil)

	_, err := service.Tautan(context.Background(), portal, "IMG-9", "JONNY")
	require.NoError(t, err)
	require.Len(t, storage.Tautan(), 1)
}

func TestTautanGagalDilaporkanSebagaiErrTautanGagal(t *testing.T) {
	sekarang := time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC)
	service, repo, storage := bangun(t, sekarang)
	lewat := sekarang.Add(-time.Minute)
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", &lewat)
	storage.Gagalkan(errors.New("layanan mati"))

	_, err := service.Tautan(context.Background(), portal, "IMG-9", "JONNY")
	require.ErrorIs(t, err, dokumenpenunjang.ErrTautanGagal)
}

func TestFolderDariAppFolder(t *testing.T) {
	require.Equal(t, "Doc/2026/10/", dokumenpenunjang.FolderDariAppFolder("gs://ember-uji-contoh/Doc/2026/10/berkas.pdf"))
	// Bucket 12 karakter: sama dengan potongan tetap Pega @substring(appfolder,17,29).
	appfolder := "gs://ember-uji-12/Doc/2026/10/berkas.pdf"
	require.Equal(t, appfolder[17:29], "/"+dokumenpenunjang.FolderDariAppFolder(appfolder)[:11])
	require.Empty(t, dokumenpenunjang.FolderDariAppFolder("Doc/2026/10/berkas.pdf"))
	require.Empty(t, dokumenpenunjang.FolderDariAppFolder("gs://ember/berkas.pdf"))
}

// Delete: jalur berkas dari APPFOLDER, muatan seperti DeleteAttachDoc langkah 6–12.
func TestHapusMengirimJalurBerkas(t *testing.T) {
	service, repo, storage := bangun(t, time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC))
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", nil)

	require.NoError(t, service.Hapus(context.Background(), portal, "IMG-9", "JONNY"))
	kirim := storage.Dihapus()
	require.Len(t, kirim, 1)
	require.Equal(t, "Doc/2026/10/Revisi023pdf", kirim[0].Jalur)
	require.Equal(t, "klaimpnc", kirim[0].NamaAplikasi)
	require.Equal(t, "KODE-1", kirim[0].KodeAkses)

	// Metadata penyimpanan tidak dihapus — DeleteDataStorage_SQL Pega menghapus IMAGEID tiruan.
	_, err := repo.Ambil(context.Background(), "IMG-9")
	require.NoError(t, err)
}

func TestHapusGagalDilaporkanSebagaiErrHapusGagal(t *testing.T) {
	service, repo, storage := bangun(t, time.Date(2026, 10, 4, 3, 0, 0, 0, time.UTC))
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", nil)
	storage.Gagalkan(errors.New("layanan mati"))

	err := service.Hapus(context.Background(), portal, "IMG-9", "JONNY")
	require.ErrorIs(t, err, dokumenpenunjang.ErrHapusGagal)
}

func TestJalurDariAppFolder(t *testing.T) {
	require.Equal(t, "Doc/2026/10/a.pdf", dokumenpenunjang.JalurDariAppFolder("gs://ember-uji-contoh/Doc/2026/10/a.pdf"))
	require.Empty(t, dokumenpenunjang.JalurDariAppFolder("Doc/2026/10/a.pdf"))
	require.Empty(t, dokumenpenunjang.JalurDariAppFolder("gs://ember/"))
}
