package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/repo/memory"
	"claim-pnc/internal/dokumenpenunjang/usecase"
)

// Penimpa nama folder penyimpanan (`PENYIMPANAN_DOKUMEN_FOLDER`).
//
// # Kenapa ini diuji seketat ini
//
// Folder menentukan DI MANA berkas mendarat, dan salah folder tidak menghasilkan satu pun
// galat — berkasnya tersimpan, hanya di tempat yang bukan haknya. Diperiksa 2026-10-14:
// `T_FOLDER_STORAGE` menjawab folder UJI sementara dokumen produksi ada di folder lain,
// dan layanan menolak unggahan ke folder uji dengan "invalid kodestring data".
//
// Penimpa ini jalan keluarnya, dan justru karena ia menyimpang dari Pega, batas
// berlakunya harus dikunci: hanya saat diisi, dan tidak mengubah apa pun saat kosong.

// folderKosong adalah repo yang MENOLAK setiap pencarian folder.
//
// Ia yang membuktikan penimpa benar-benar memintas tabel: bila jalur tabelnya masih
// tersentuh, uji di bawah akan gagal dengan ErrFolderAplikasiTidakAda.
type folderKosong struct{ *memory.Repo }

func (folderKosong) NamaFolderAplikasi(context.Context, string) (string, error) {
	return "", dokumenpenunjang.ErrFolderAplikasiTidakAda
}

func bangunDenganFolder(
	t *testing.T,
	folder string,
	repo dokumenpenunjang.Repo,
) (*usecase.Service, *memory.Storage) {
	t.Helper()

	storage := memory.NewStorage()
	service, err := usecase.NewService(usecase.Options{
		Repos:      func(string) (dokumenpenunjang.Repo, error) { return repo, nil },
		Storage:    storage,
		Converter:  memory.NewConverter(),
		Clock:      jamTetap{pada: time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)},
		FolderName: folder,
	})
	require.NoError(t, err)
	return service, storage
}

func TestFolderPenimpaDipakaiDanMemintasTabel(t *testing.T) {
	// Repo yang pencarian foldernya SELALU gagal. Kalau penimpanya benar-benar dipakai,
	// kegagalan itu tidak pernah tersentuh.
	service, storage := bangunDenganFolder(t, "klaimpnc", folderKosong{memory.NewRepo()})

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: "ASM",
		Request:     permintaan(),
	})
	require.NoError(t, err, "penimpa folder harus memintas pencarian ke T_FOLDER_STORAGE")

	require.Equal(t, "klaimpnc", storage.Terakhir().NamaAplikasi,
		"folder yang dikirim ke layanan harus folder penimpa, bukan hasil pencarian tabel")
}

// Tanpa penimpa, perilakunya TIDAK berubah sedikit pun: namanya tetap dibaca dari tabel,
// persis seperti Pega. Ini yang menjaga penimpa tetap menjadi jalan keluar sementara, bukan
// perubahan perilaku yang diam-diam menetap.
func TestTanpaPenimpaNamaFolderTetapDariTabel(t *testing.T) {
	service, _, storage := bangun(t, time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC))

	_, err := service.Upload(context.Background(), usecase.UploadCommand{
		PortalAlias: "ASM",
		Request:     permintaan(),
	})
	require.NoError(t, err)

	dikirim := storage.Terakhir().NamaAplikasi
	require.NotEmpty(t, dikirim)
	require.NotEqual(t, dokumenpenunjang.NamaAplikasi, dikirim,
		"yang dikirim adalah NAMA FOLDER hasil pencarian, bukan kunci pencariannya")
}

// Penimpa berlaku pada SELURUH jalur yang membutuhkan folder, bukan pada unggah saja.
//
// Perpanjangan tautan memakai folder yang sama; bila ia masih membaca tabel sementara
// unggahnya tidak, berkas akan terunggah ke satu folder lalu dicari di folder lain.
func TestFolderPenimpaBerlakuSaatMemperpanjangTautan(t *testing.T) {
	sekarang := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)
	repo := memory.NewRepo()
	service, storage := bangunDenganFolder(t, "klaimpnc", folderKosong{repo})

	// Alamatnya sudah kedaluwarsa, sehingga perpanjangan benar-benar ditempuh — bukan
	// dilewati karena alamat lamanya masih berlaku.
	lewat := sekarang.Add(-time.Minute)
	simpanDokumen(t, repo, "https://penyimpanan.contoh/lama", &lewat)

	_, err := service.Tautan(context.Background(), "ASM", "IMG-9", "PETUGAS01")
	require.NoError(t, err, "perpanjangan tautan harus memakai penimpa folder yang sama")

	tautan := storage.Tautan()
	require.Len(t, tautan, 1, "perpanjangan tautan harus benar-benar ditempuh")
	require.Equal(t, "klaimpnc", tautan[0].NamaAplikasi)
}
