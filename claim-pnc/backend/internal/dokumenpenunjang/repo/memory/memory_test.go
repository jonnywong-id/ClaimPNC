package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/dokumenpenunjang/repo/memory"
)

var errBoom = errors.New("rusak")

func TestRepoFolderAccessAndForget(t *testing.T) {
	repo := memory.NewRepo()
	ctx := context.Background()

	folder, err := repo.NamaFolderAplikasi(ctx, dokumenpenunjang.NamaAplikasi)
	require.NoError(t, err)
	require.Equal(t, "klaimpnc", folder)

	repo.LupakanFolder(dokumenpenunjang.NamaAplikasi)
	_, err = repo.NamaFolderAplikasi(ctx, dokumenpenunjang.NamaAplikasi)
	require.ErrorIs(t, err, dokumenpenunjang.ErrFolderAplikasiTidakAda)

	code, err := repo.CatatAksesUnggah(ctx, "KLAIMPNC", "BUDI")
	require.NoError(t, err)
	require.Equal(t, "KODE-1", code)
	code, _ = repo.CatatAksesUnggah(ctx, "KLAIMPNC", "SITI")
	require.Equal(t, "KODE-2", code)
	require.Equal(t, []memory.CatatanAkses{
		{Aplikasi: "KLAIMPNC", Pengunggah: "BUDI"},
		{Aplikasi: "KLAIMPNC", Pengunggah: "SITI"},
	}, repo.Akses())
}

func TestRepoSaveListAndGet(t *testing.T) {
	repo := memory.NewRepo()
	ctx := context.Background()

	require.NoError(t, repo.Simpan(ctx, dokumenpenunjang.Document{ImageID: "A", ClaimNumber: "PNC-1"}))
	require.NoError(t, repo.Simpan(ctx, dokumenpenunjang.Document{ImageID: "B", ClaimNumber: "pnc-1 "}))
	require.NoError(t, repo.Simpan(ctx, dokumenpenunjang.Document{ImageID: "C", ClaimNumber: "PNC-2"}))
	// Menyimpan ulang id yang sama menimpa isinya tanpa menggandakan urutannya.
	require.NoError(t, repo.Simpan(ctx, dokumenpenunjang.Document{
		ImageID: "A", ClaimNumber: "PNC-1", FileName: "baru",
	}))

	rows, err := repo.PerKlaim(ctx, " PNC-1 ")
	require.NoError(t, err)
	require.Equal(t, []string{"B", "A"}, []string{rows[0].ImageID, rows[1].ImageID},
		"terbaru lebih dulu")
	require.Equal(t, "baru", rows[1].FileName)

	got, err := repo.Ambil(ctx, " C ")
	require.NoError(t, err)
	require.Equal(t, "PNC-2", got.ClaimNumber)
	_, err = repo.Ambil(ctx, "Z")
	require.ErrorIs(t, err, dokumenpenunjang.ErrTidakDitemukan)

	repo.GagalkanSimpan(errBoom)
	require.ErrorIs(t, repo.Simpan(ctx, dokumenpenunjang.Document{ImageID: "D"}), errBoom)
}

func TestStorageRecordsUploadsAndCanFail(t *testing.T) {
	storage := memory.NewStorage()
	ctx := context.Background()

	require.Panics(t, func() { storage.Terakhir() }, "belum ada unggahan")

	result, err := storage.Upload(ctx, dokumenpenunjang.PerintahUnggah{Folder: "Doc/2026/09/", Isi: []byte("a")})
	require.NoError(t, err)
	require.Equal(t, "Doc/2026/09/", result.Folder)
	require.Regexp(t, `^[0-9a-f]{12}-1$`, result.ImageID)

	second, _ := storage.Upload(ctx, dokumenpenunjang.PerintahUnggah{Isi: []byte("a"), NamaBerkas: "x"})
	require.NotEqual(t, result.ImageID, second.ImageID, "urutan membedakan isi yang sama")
	require.Len(t, storage.Diterima(), 2)
	require.Equal(t, "x", storage.Terakhir().NamaBerkas)

	storage.Gagalkan(errBoom)
	_, err = storage.Upload(ctx, dokumenpenunjang.PerintahUnggah{})
	require.ErrorIs(t, err, errBoom)
	require.Len(t, storage.Diterima(), 2, "unggahan yang gagal tidak dicatat")
}

func TestConverterPrefixesEmptiesAndFails(t *testing.T) {
	converter := memory.NewConverter()
	ctx := context.Background()

	out, err := converter.Convert(ctx, []byte("isi"))
	require.NoError(t, err)
	require.Equal(t, []byte("AVIF:isi"), out)

	converter.KembalikanKosong()
	out, err = converter.Convert(ctx, []byte("dua"))
	require.NoError(t, err)
	require.Nil(t, out)
	require.Equal(t, [][]byte{[]byte("isi"), []byte("dua")}, converter.Diterima())

	converter.Gagalkan(errBoom)
	_, err = converter.Convert(ctx, []byte("tiga"))
	require.ErrorIs(t, err, errBoom)
}
