package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputacceptation"
	"claim-pnc/internal/inputacceptation/repo/memory"
	"claim-pnc/internal/inputacceptation/usecase"
)

const portalUtama = "ASM"

var caller = inputacceptation.Caller{Login: "ADMINNONPROP1"}

// errPortalAsing ditolak selector saat portalnya bukan portal utama.
var errPortalAsing = errors.New("portal tidak dikenal")

func newService(t *testing.T) *usecase.Service {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inputacceptation.Repo, error) {
			if alias != portalUtama {
				return nil, errPortalAsing
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service
}

func TestRepoSelectorWajibDiisi(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Bentuk layar tidak menyentuh basis data sama sekali.
//
// Ia hasil pembacaan export, bukan data entitas — sehingga sama di keempat portal.
func TestMetadataTidakMenyentuhPenyimpanan(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inputacceptation.Repo, error) {
			t.Fatal("Metadata tidak boleh memilih penyimpanan")
			return nil, nil
		},
	})
	require.NoError(t, err)

	meta := service.Metadata()
	require.NotEmpty(t, meta.Groups)
	require.Len(t, meta.Grids, 13)
	require.NotEmpty(t, meta.PlannedDifferences)
}

func TestRincianTerbaca(t *testing.T) {
	detail, err := newService(t).Find(
		context.Background(), portalUtama, caller, "CLMNP-1001")
	require.NoError(t, err)

	require.Equal(t, "CLMNP-1001", detail.ClaimID)
	require.Equal(t, "Pending-Acceptation", detail.StatusWork)
	require.Equal(t, "XOL Property 2026 Layer 1", detail.Get("treaty_name"))
	require.NotEmpty(t, detail.Rows(inputacceptation.GridAdjustment))
}

// Klaim yang ADA tetapi dokumennya kosong TETAP dapat dibuka.
//
// Ia tidak boleh dijawab "tidak ditemukan": klaimnya ada, hanya isinya yang belum tersalin —
// dan itulah yang ditetapkan gabungan LEFT JOIN pada kuerinya.
func TestKlaimTanpaDokumenTetapDapatDibuka(t *testing.T) {
	detail, err := newService(t).Find(
		context.Background(), portalUtama, caller, "CLMNP-1002")
	require.NoError(t, err)

	require.Equal(t, "CLMNP-1002", detail.ClaimID)
	require.Empty(t, detail.Get("treaty_name"))
}

func TestKlaimTidakAdaMenghasilkanErrNotFound(t *testing.T) {
	_, err := newService(t).Find(
		context.Background(), portalUtama, caller, "CLMNP-9999")
	require.ErrorIs(t, err, inputacceptation.ErrNotFound)
}

// Portal yang tidak dikenal DITOLAK, tidak pernah jatuh ke portal utama.
//
// Jatuh ke koneksi bawaan berarti menampilkan nilai akseptasi satu badan hukum kepada petugas
// badan hukum lain — dan pada layar yang juga menulis, menulisinya (`R-20`).
func TestPortalAsingDitolak(t *testing.T) {
	_, err := newService(t).Find(context.Background(), "SIMASNET", caller, "CLMNP-1001")
	require.ErrorIs(t, err, errPortalAsing)

	err = newService(t).Submit(
		context.Background(), "SIMASNET", caller, "CLMNP-1001",
		map[string]string{"dla_no_ceding": "X"}, nil)
	require.ErrorIs(t, err, errPortalAsing)
}

// Submit MEMBACA klaimnya lebih dulu, sehingga klaim yang tidak ada ditolak sebagai tidak
// ditemukan — bukan sebagai penolakan kepemilikan tabel.
//
// Urutan itu penting: kunci teknis objek kerja diambil dari penyimpanan, bukan dari klien.
func TestSubmitKlaimTidakAdaDitolakSebagaiTidakDitemukan(t *testing.T) {
	err := newService(t).Submit(
		context.Background(), portalUtama, caller, "CLMNP-9999",
		map[string]string{"dla_no_ceding": "X"}, nil)
	require.ErrorIs(t, err, inputacceptation.ErrNotFound)
}

// Muatan yang keliru ditolak SEBELUM penolakan kepemilikan tabel.
//
// Pengguna yang mengirim isian keliru tetap diberi tahu apa yang keliru, alih-alih hanya
// diberi tahu bahwa penyimpanannya belum tersedia.
func TestSubmitMemvalidasiMuatanSebelumMenolakKepemilikan(t *testing.T) {
	err := newService(t).Submit(
		context.Background(), portalUtama, caller, "CLMNP-1001",
		map[string]string{"treaty_id": "TNP-DIUBAH"}, nil)

	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)
	require.NotErrorIs(t, err, inputacceptation.ErrWriteNotOwned)
}

// Muatan yang SAH tetap ditolak, dengan alasan kepemilikan tabel.
//
// Selama masa paralel, tabel objek kerja dan JSON_KLAIM masih ditulis Pega (`P-1`). Penolakan
// itu keadaan yang diketahui dan punya jalan keluar (`D-63`), bukan kerusakan — karena itu ia
// galat tersendiri, bukan 500.
func TestSubmitYangSahDitolakKarenaKepemilikanTabel(t *testing.T) {
	err := newService(t).Submit(
		context.Background(), portalUtama, caller, "CLMNP-1001",
		map[string]string{"dla_no_ceding": "DLA/2026/0002"}, nil)

	require.ErrorIs(t, err, inputacceptation.ErrWriteNotOwned)
}

// Nomor klaim di luar lini non-prop ditolak di kedua operasi.
func TestNomorKlaimLiniLainDitolakDiKeduaOperasi(t *testing.T) {
	service := newService(t)

	_, err := service.Find(context.Background(), portalUtama, caller, "CLMP-70")
	var validation *inputacceptation.ValidationError
	require.ErrorAs(t, err, &validation)

	err = service.Submit(context.Background(), portalUtama, caller, "CLMP-70", nil, nil)
	require.ErrorAs(t, err, &validation)
}
