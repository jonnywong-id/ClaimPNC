package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatus"
	"claim-pnc/internal/masterstatus/repo/memory"
	"claim-pnc/internal/masterstatus/usecase"
)

const portalASM = "ASM"

// errPortalNotReady meniru galat pemilih repo sungguhan saat portalnya tidak dilayani.
var errPortalNotReady = errors.New("portal belum siap")

func testService(t *testing.T, content ...masterstatus.ClaimStatus) (*usecase.Service, *memory.Repo) {
	t.Helper()
	if len(content) == 0 {
		content = memory.SampleList()
	}
	repo := memory.NewRepo(content...)
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterstatus.Repo, error) {
			if alias != portalASM {
				return nil, errPortalNotReady
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

func TestServiceRejectsIncompleteDeps(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err, "kegagalan terjadi saat start, bukan saat pengguna membuka layar")
}

func TestListReturns33StatusesSorted(t *testing.T) {
	service, _ := testService(t)

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 33)

	for i := 1; i < len(list); i++ {
		require.Less(t, list[i-1].Code, list[i].Code, "daftar harus terurut menurut kode")
	}
}

func TestGetExistingStatus(t *testing.T) {
	service, _ := testService(t)

	status, err := service.Get(context.Background(), portalASM, "1163")
	require.NoError(t, err)
	require.Equal(t, "Paid", status.Label)
}

func TestGetStatusThatDoesNotExist(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Get(context.Background(), portalASM, "9999")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
}

// Kode dibuat penyimpanan, melanjutkan deret yang sudah ada — persis seperti procedure
// lama yang menerima sentinel "UnknownID" lalu menentukan kodenya sendiri.
func TestCreateIssuesNextCode(t *testing.T) {
	service, _ := testService(t)

	status, err := service.Create(context.Background(), portalASM, "Status Percobaan")
	require.NoError(t, err)
	require.Equal(t, "1167", status.Code, "melanjutkan 1166, tidak mengulang dari awal")
	require.Equal(t, "Status Percobaan", status.Label)
	require.Empty(t, status.LegacyCode, "status baru tidak pernah diberi penomoran lama")

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 34)
}

func TestCreateWithEmptyLabelRejected(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Create(context.Background(), portalASM, "   ")

	var validasi *masterstatus.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violation, 1)
	require.Equal(t, masterstatus.FieldLabel, validasi.Violation[0].Field)

	list, _ := service.List(context.Background(), portalASM)
	require.Len(t, list, 33, "tidak ada yang tersimpan saat validasi gagal")
}

func TestCreateWithTakenLabelRejected(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Create(context.Background(), portalASM, "Paid")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)
}

// Keunikan tidak boleh dapat ditembus hanya dengan mengubah besar-kecil huruf atau
// menambah spasi.
func TestLabelUniquenessIgnoresCaseAndSpaces(t *testing.T) {
	for _, input := range []string{"PAID", "paid", "  Paid  ", "pAiD"} {
		service, _ := testService(t)
		_, err := service.Create(context.Background(), portalASM, input)
		require.ErrorIs(t, err, masterstatus.ErrLabelTaken,
			"%q seharusnya dikenali sama dengan status Paid yang sudah ada", input)
	}
}

func TestUpdateReplacesLabel(t *testing.T) {
	service, _ := testService(t)

	status, err := service.Update(context.Background(), portalASM, "1163", "Sudah Dibayar")
	require.NoError(t, err)
	require.Equal(t, "1163", status.Code, "kode tidak ikut berubah")
	require.Equal(t, "Sudah Dibayar", status.Label)

	read, err := service.Get(context.Background(), portalASM, "1163")
	require.NoError(t, err)
	require.Equal(t, "Sudah Dibayar", read.Label, "perubahan benar-benar tersimpan")
}

// Kode lama adalah jejak sejarah; mengubah label tidak boleh menghapusnya.
func TestUpdateKeepsLegacyCode(t *testing.T) {
	service, _ := testService(t)

	before, err := service.Get(context.Background(), portalASM, "1134")
	require.NoError(t, err)
	require.Equal(t, "01", before.LegacyCode, "prasyarat uji")

	after, err := service.Update(context.Background(), portalASM, "1134", "Laporan Ringkas")
	require.NoError(t, err)
	require.Equal(t, "01", after.LegacyCode)
}

// Menyimpan ulang tanpa mengubah label tidak boleh ditolak karena bentrok dengan
// dirinya sendiri — kesalahan klasik pada pemeriksaan keunikan.
func TestUpdateWithSameLabelNotTreatedAsConflict(t *testing.T) {
	service, _ := testService(t)

	status, err := service.Update(context.Background(), portalASM, "1163", "Paid")
	require.NoError(t, err)
	require.Equal(t, "Paid", status.Label)
}

func TestUpdateToAnotherStatusLabelRejected(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Update(context.Background(), portalASM, "1163", "Register")
	require.ErrorIs(t, err, masterstatus.ErrLabelTaken)
}

func TestUpdateWithEmptyLabelRejected(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Update(context.Background(), portalASM, "1163", "")

	var validasi *masterstatus.ValidationError
	require.ErrorAs(t, err, &validasi)

	tetap, _ := service.Get(context.Background(), portalASM, "1163")
	require.Equal(t, "Paid", tetap.Label, "label lama tidak tertimpa saat validasi gagal")
}

// Mengubah status yang tidak ada dijawab "tidak ditemukan", bukan galat lain yang
// menyesatkan — kode diperiksa sebelum keunikan label.
func TestUpdateStatusThatDoesNotExist(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Update(context.Background(), portalASM, "9999", "Apa Saja")
	require.ErrorIs(t, err, masterstatus.ErrNotFound)
}

func TestUpdateMissingStatusWithConflictingLabel(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Update(context.Background(), portalASM, "9999", "Paid")
	require.ErrorIs(t, err, masterstatus.ErrNotFound,
		"kode yang tidak ada lebih dulu dilaporkan daripada label yang bentrok")
}

// Galat penyimpanan dibungkus konteks, tidak ditelan dan tidak disamarkan menjadi
// "tidak ditemukan".
func TestStorageErrorWrappedNotSwallowed(t *testing.T) {
	service, repo := testService(t)
	rusak := errors.New("koneksi terputus")
	repo.SetError(rusak)

	_, err := service.List(context.Background(), portalASM)
	require.ErrorIs(t, err, rusak)
	require.Contains(t, err.Error(), "masterstatus/usecase", "jejaknya terbaca dari pesan")

	_, err = service.Get(context.Background(), portalASM, "1163")
	require.ErrorIs(t, err, rusak)
	require.NotErrorIs(t, err, masterstatus.ErrNotFound)
}

// Spasi tepi pada masukan pengguna dibuang sebelum disimpan, supaya label tidak
// tersimpan dengan spasi yang tak terlihat siapa pun.
func TestEdgeSpacesTrimmedBeforeSaving(t *testing.T) {
	service, _ := testService(t)

	status, err := service.Create(context.Background(), portalASM, "   Status Baru   ")
	require.NoError(t, err)
	require.Equal(t, "Status Baru", status.Label)
}

// Tidak ada Hapus — dan itu disengaja. Uji ini mengunci ketiadaan itu supaya penambahan
// operasi hapus menjadi keputusan sadar, bukan kelalaian yang lolos review.
func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	var seam masterstatus.Repo = memory.NewRepo()

	// Bila seseorang menambah Delete pada masterstatus.Repo, penegasan tipe di bawah
	// akan mulai berhasil dan uji ini gagal — memaksa keputusannya dibicarakan.
	_, hasDelete := seam.(interface {
		Delete(ctx context.Context, code string) error
	})
	require.False(t, hasDelete,
		"layar Pega tidak punya tombol hapus dan ADR-0012 melarang master dihapus permanen")
}
