package usecase_test

import (
	"claim-pnc/internal/platform/clock"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/mastersparepart/repo/memory"
	usecase "claim-pnc/internal/mastersparepart/usecase"
)

const headerCSV = "NAMA_SPART,NO_SPART,KODE_SPART,HARGA_JUAL\n"

// Baris ber-NO_SPART baru menjadi baris BARU, dan ID-nya diterbitkan server.
func TestImportCSVMenambahBarisBaru(t *testing.T) {
	service, repo := newSampleService(t)

	report, err := service.ImportCSV(context.Background(), portalAlias,
		[]byte(headerCSV+"SEAL KIT BARU,NS-BARU-1,KD-BARU-1,125000\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.NoError(t, err)

	require.Equal(t, 1, report.Total)
	require.Equal(t, 1, report.Created)
	require.Equal(t, 0, report.Updated)
	require.Equal(t, 0, report.Failed)
	require.Equal(t, usecase.ImportCreated, report.Rows[0].Outcome)
	require.NotEmpty(t, report.Rows[0].ID, "ID baris baru harus diterbitkan server")

	saved, err := repo.FindByNumber(context.Background(), "NS-BARU-1")
	require.NoError(t, err)
	require.Equal(t, "SEAL KIT BARU", saved.Name)
}

// Baris ber-NO_SPART yang SUDAH ADA memperbarui baris itu, bukan menambah baris kedua.
//
// Ini inti upsert-nya, dan kuncinya NO_SPART — bukan nama, bukan kode.
func TestImportCSVMemperbaruiLewatNomorSparepart(t *testing.T) {
	service, repo := newSampleService(t)

	lama, err := repo.List(context.Background(), mastersparepart.Filter{
		Status: mastersparepart.StatusApproved,
	})
	require.NoError(t, err)
	require.NotEmpty(t, lama)
	target := lama[0]

	report, err := service.ImportCSV(context.Background(), portalAlias,
		[]byte(headerCSV+"NAMA BERUBAH,"+target.Number+",KODE-BARU-9,777000\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.NoError(t, err)

	require.Equal(t, 1, report.Updated)
	require.Equal(t, 0, report.Created)
	require.Equal(t, target.ID, report.Rows[0].ID, "ID lama harus dipakai ulang")

	sesudah, err := repo.Get(context.Background(), target.ID)
	require.NoError(t, err)
	require.Equal(t, "NAMA BERUBAH", sesudah.Name)
}

// Setiap baris hasil unggah masuk antrean persetujuan.
//
// `PNCUploadMasterSparepart_Act` menyetel `APPROVAL := "0"` tanpa syarat. Bila ini gagal,
// unggahan massal memintas seluruh kontrol persetujuan — akibat paling berat dari modul ini.
func TestImportCSVSelaluMasukWaitingApproval(t *testing.T) {
	service, repo := newSampleService(t)

	lama, err := repo.List(context.Background(), mastersparepart.Filter{
		Status: mastersparepart.StatusApproved,
	})
	require.NoError(t, err)
	target := lama[0]

	_, err = service.ImportCSV(context.Background(), portalAlias,
		[]byte(headerCSV+"APA SAJA,"+target.Number+",KD-X,1000\n"+
			"BARIS BARU,NS-BARU-2,KD-BARU-2,2000\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.NoError(t, err)

	diperbarui, err := repo.Get(context.Background(), target.ID)
	require.NoError(t, err)
	require.Equal(t, mastersparepart.StatusPending, diperbarui.Status,
		"baris yang diperbarui lewat CSV harus kembali menunggu persetujuan")

	baru, err := repo.FindByNumber(context.Background(), "NS-BARU-2")
	require.NoError(t, err)
	require.Equal(t, mastersparepart.StatusPending, baru.Status)
}

// Satu baris yang gagal TIDAK membatalkan baris lain.
//
// Mengikuti Pega, yang memutar baris satu per satu tanpa transaksi yang membungkus
// keseluruhan. Yang membuatnya dapat diterima adalah laporannya — lihat uji berikutnya.
func TestImportCSVBarisGagalTidakMembatalkanYangLain(t *testing.T) {
	service, repo := newSampleService(t)

	report, err := service.ImportCSV(context.Background(), portalAlias,
		[]byte(headerCSV+
			"BENAR SATU,NS-OK-1,KD-OK-1,1000\n"+
			",NS-GAGAL,KD-GAGAL,2000\n"+ // nama kosong -> ditolak
			"BENAR DUA,NS-OK-2,KD-OK-2,3000\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.NoError(t, err)

	require.Equal(t, 3, report.Total)
	require.Equal(t, 2, report.Created)
	require.Equal(t, 1, report.Failed)

	for _, nomor := range []string{"NS-OK-1", "NS-OK-2"} {
		_, err := repo.FindByNumber(context.Background(), nomor)
		require.NoErrorf(t, err, "baris %s seharusnya tersimpan", nomor)
	}
	_, err = repo.FindByNumber(context.Background(), "NS-GAGAL")
	require.ErrorIs(t, err, mastersparepart.ErrNotFound)
}

// Laporan memuat SELURUH baris beserta nomor barisnya, bukan yang gagal saja.
//
// Tanpa nomor baris, "1 baris gagal" memaksa pengguna menebak yang mana.
func TestImportCSVMelaporkanSetiapBarisBesertaNomornya(t *testing.T) {
	service, _ := newSampleService(t)

	report, err := service.ImportCSV(context.Background(), portalAlias,
		[]byte(headerCSV+
			"SATU,NS-L-1,KD-L-1,1000\n"+
			",NS-L-2,KD-L-2,2000\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.NoError(t, err)

	require.Len(t, report.Rows, 2, "seluruh baris dilaporkan, bukan yang gagal saja")
	require.Equal(t, 2, report.Rows[0].Line, "header terhitung sebagai baris 1")
	require.Equal(t, 3, report.Rows[1].Line)
	require.Equal(t, usecase.ImportFailed, report.Rows[1].Outcome)
	require.NotEmpty(t, report.Rows[1].Message, "baris gagal harus menyebut sebabnya")
}

// Berkas yang tidak dapat dibaca ditolak SELURUHNYA, bukan dilaporkan "0 berhasil".
func TestImportCSVMenolakBerkasTanpaHeaderLengkap(t *testing.T) {
	service, _ := newSampleService(t)

	_, err := service.ImportCSV(context.Background(), portalAlias,
		[]byte("NAMA_SPART,NO_SPART\nA,NS-1\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.ErrorIs(t, err, mastersparepart.ErrCSVHeaderMissing)
}

// Portal yang tidak dikenal ditolak sebelum satu baris pun disentuh (`R-20`).
func TestImportCSVMenolakPortalTakDikenal(t *testing.T) {
	service := newService(t, memory.NewSampleRepo())

	_, err := service.ImportCSV(context.Background(), "ENTITAS-LAIN",
		[]byte(headerCSV+"A,NS-1,KD-1,1\n"),
		usecase.Actor{Login: "JONNY"}, nil)
	require.Error(t, err)
}

// UploadAvailable menentukan HIDUP-MATINYA tombol "Upload Document" di layar.
//
// Uji ini ada karena cacatnya tidak terlihat sebagai galat: bila benderanya salah, tombolnya
// hanya diam saat ditekan, dan tidak ada satu pun pesan yang menyebutkan sebabnya.
func TestUploadAvailableMengikutiAdaTidaknyaUploader(t *testing.T) {
	tanpa := newService(t, memory.NewSampleRepo())
	require.False(t, tanpa.UploadAvailable(),
		"tanpa Uploader, layar harus menyatakan unggah belum tersedia")

	dengan, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersparepart.Store, error) {
			return memory.NewSampleRepo(), nil
		},
		Clock:    clock.FixedAt(fixedNow),
		Uploader: uploaderPalsu{},
	})
	require.NoError(t, err)
	require.True(t, dengan.UploadAvailable(),
		"dengan Uploader terpasang, tombol unggah harus hidup")
}

// uploaderPalsu adalah adapter kedua seam DocumentUploader — yang membuat seam ini nyata,
// bukan hipotetis.
type uploaderPalsu struct{}

func (uploaderPalsu) Upload(context.Context, mastersparepart.DocumentFile) (string, error) {
	return "IMG-1", nil
}
