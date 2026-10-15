package usecase_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastertipesurveyors"
	"claim-pnc/internal/mastertipesurveyors/repo/memory"
	"claim-pnc/internal/mastertipesurveyors/usecase"
)

const portalASM = "ASM"

// errPortalNotReady meniru galat yang dihasilkan pemilih repo sungguhan ketika portalnya
// tidak dikenal atau koneksinya belum hidup.
var errPortalNotReady = errors.New("portal belum siap")

// newService merakit layanan di atas penyimpanan memori berisi keempat tipe nyata.
//
// Pemilih repo-nya HANYA melayani ASM. Portal lain menghasilkan galat — persis seperti di
// produksi, sehingga perilaku penolakannya ikut teruji di sini dan bukan hanya nanti.
func newService(t *testing.T, list ...mastertipesurveyors.SurveyorType) (*usecase.Service, *memory.Repo) {
	t.Helper()

	if len(list) == 0 {
		list = memory.SampleList()
	}
	repo := memory.NewRepo(list...)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastertipesurveyors.Repo, error) {
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
	require.Error(t, err, "rakitan tanpa pemilih repo harus gagal saat start, bukan saat dipakai")
}

// ── Membaca ─────────────────────────────────────────────────────────────────────

func TestListReturnsFourTypesSorted(t *testing.T) {
	service, _ := newService(t)

	list, err := service.List(t.Context(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 4)
	require.Equal(t, "1001", list[0].Code)
	require.Equal(t, "1004", list[3].Code)
}

func TestGetExistingType(t *testing.T) {
	service, _ := newService(t)

	tipe, err := service.Get(t.Context(), portalASM, "1002")
	require.NoError(t, err)
	require.Equal(t, "LOSS ADJUSTER", tipe.Description)
}

func TestGetTypeThatDoesNotExist(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Get(t.Context(), portalASM, "9999")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound,
		"galatnya harus dapat dikenali pemanggil tanpa membaca teks pesan")
}

// ── Portal ──────────────────────────────────────────────────────────────────────

// Portal yang tidak dapat dilayani menghasilkan galat pada SETIAP aksi — tidak ada satu
// pun yang diam-diam dialihkan ke portal lain (R-20).
func TestEveryActionRejectsUnservableportal(t *testing.T) {
	service, repo := newService(t)

	t.Run("daftar", func(t *testing.T) {
		_, err := service.List(t.Context(), "ASI")
		require.ErrorIs(t, err, errPortalNotReady)
	})
	t.Run("ambil", func(t *testing.T) {
		_, err := service.Get(t.Context(), "ASI", "1001")
		require.ErrorIs(t, err, errPortalNotReady)
	})
	t.Run("tambah", func(t *testing.T) {
		_, err := service.Create(t.Context(), "ASI", "TIPE BARU")
		require.ErrorIs(t, err, errPortalNotReady)
	})
	t.Run("ubah", func(t *testing.T) {
		_, err := service.Update(t.Context(), "ASI", "1001", "TIPE BARU")
		require.ErrorIs(t, err, errPortalNotReady)
	})

	// Dan tidak satu baris pun berubah di portal yang sah.
	list, err := repo.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 4)
}

// Penolakan portal terjadi SEBELUM isian diperiksa. Urutannya penting: isian yang tidak
// sah pada portal yang tidak dapat dilayani harus dijawab soal portalnya — karena itulah
// yang harus diperbaiki lebih dulu.
func TestPortalCheckedBeforeInput(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Create(t.Context(), "ASI", "")
	require.ErrorIs(t, err, errPortalNotReady)

	var validationError *mastertipesurveyors.ValidationError
	require.False(t, errors.As(err, &validationError),
		"yang dilaporkan harus portalnya, bukan isiannya")
}

func TestEnsurePortalReady(t *testing.T) {
	service, _ := newService(t)

	require.NoError(t, service.EnsurePortalReady(portalASM))
	require.Error(t, service.EnsurePortalReady("ASI"))
}

// ── Menambah ────────────────────────────────────────────────────────────────────

// Kode diterbitkan penyimpanan, melanjutkan urutan yang sudah ada — bukan diterima dari
// pemanggil, dan bukan pula mengulang nomor yang sudah dipakai.
func TestCreateIssuesNextCode(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(t.Context(), portalASM, "ADJUSTER INDEPENDEN")
	require.NoError(t, err)
	require.Equal(t, "1005", saved.Code, "melanjutkan 1004, bukan mengulang")
	require.Equal(t, "ADJUSTER INDEPENDEN", saved.Description)
	require.Empty(t, saved.LegacyCode, "tipe baru tidak pernah diberi penomoran lama")

	list, err := service.List(t.Context(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 5)
}

func TestCreateWithEmptyDescriptionRejected(t *testing.T) {
	service, repo := newService(t)

	_, err := service.Create(t.Context(), portalASM, "   ")

	var validationError *mastertipesurveyors.ValidationError
	require.ErrorAs(t, err, &validationError)

	list, err := repo.List(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 4, "tidak ada baris yang tersimpan saat isian ditolak")
}

func TestCreateWithTakenDescriptionRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Create(t.Context(), portalASM, "EXPERT")
	require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
}

func TestDescriptionUniquenessIgnoresCaseAndSpaces(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Create(t.Context(), portalASM, "  expert  ")
	require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken,
		`"expert" dan "EXPERT" adalah tipe yang sama bagi pengguna`)
}

// Spasi tepi dibuang sebelum menyentuh penyimpanan, bukan dibiarkan ikut tersimpan.
func TestEdgeSpacesTrimmedBeforeSaving(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(t.Context(), portalASM, "  SURVEYOR MANDIRI  ")
	require.NoError(t, err)
	require.Equal(t, "SURVEYOR MANDIRI", saved.Description)
}

// ── Mengubah ────────────────────────────────────────────────────────────────────

func TestUpdateReplacesDescription(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Update(t.Context(), portalASM, "1003", "TENAGA AHLI")
	require.NoError(t, err)
	require.Equal(t, "1003", saved.Code, "kode tidak ikut berubah")
	require.Equal(t, "TENAGA AHLI", saved.Description)

	reloaded, err := service.Get(t.Context(), portalASM, "1003")
	require.NoError(t, err)
	require.Equal(t, "TENAGA AHLI", reloaded.Description)
}

// Menyimpan ulang tanpa mengubah namanya TIDAK boleh ditolak sebagai bentrok — baris itu
// bentrok dengan dirinya sendiri, dan menolaknya akan membuat tombol Simpan tampak rusak.
func TestUpdateWithSameDescriptionNotTreatedAsConflict(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(t.Context(), portalASM, "1003", "EXPERT")
	require.NoError(t, err)
}

func TestUpdateToAnotherTypeDescriptionRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(t.Context(), portalASM, "1003", "LOSS ADJUSTER")
	require.ErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
}

func TestUpdateWithEmptyDescriptionRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(t.Context(), portalASM, "1003", "")

	var validationError *mastertipesurveyors.ValidationError
	require.ErrorAs(t, err, &validationError)

	unchanged, err := service.Get(t.Context(), portalASM, "1003")
	require.NoError(t, err)
	require.Equal(t, "EXPERT", unchanged.Description)
}

func TestUpdateTypeThatDoesNotExist(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(t.Context(), portalASM, "9999", "APA SAJA")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)
}

// Mengubah tipe yang tidak ada DENGAN nama yang bentrok harus dijawab "tidak ditemukan",
// bukan "nama sudah dipakai". Yang kedua menyesatkan: pengguna akan mengganti namanya
// berulang kali padahal barisnya memang tidak ada.
func TestUpdateMissingTypeWithConflictingDescription(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Update(t.Context(), portalASM, "9999", "EXPERT")
	require.ErrorIs(t, err, mastertipesurveyors.ErrNotFound)
	require.NotErrorIs(t, err, mastertipesurveyors.ErrDescriptionTaken)
}

// Kode lama dipertahankan saat deskripsi diubah — ia jejak sejarah, bukan field yang
// dikelola. Diuji dengan baris karangan karena keempat baris nyata tidak memilikinya.
func TestUpdateKeepsLegacyCode(t *testing.T) {
	service, _ := newService(t, mastertipesurveyors.SurveyorType{
		Code: "1001", Description: "INTERNAL SURVEYOR", LegacyCode: "01",
	})

	saved, err := service.Update(t.Context(), portalASM, "1001", "SURVEYOR INTERNAL")
	require.NoError(t, err)
	require.Equal(t, "01", saved.LegacyCode)
}

// ── Kegagalan penyimpanan ───────────────────────────────────────────────────────

// Kegagalan penyimpanan dibungkus konteks, TIDAK ditelan. Tanpa itu, kegagalan basis data
// akan sampai ke layar sebagai daftar kosong yang tampak normal.
func TestStorageErrorWrappedNotSwallowed(t *testing.T) {
	service, repo := newService(t)
	kegagalan := errors.New("koneksi terputus")
	repo.SetError(kegagalan)

	_, err := service.List(t.Context(), portalASM)
	require.ErrorIs(t, err, kegagalan)
	require.Contains(t, err.Error(), "mastertipesurveyors/usecase")
}

// Seam-nya TIDAK menyediakan operasi hapus, dan itu penegakan di tingkat tipe — bukan
// sekadar kesepakatan. Uji ini gagal dikompilasi bila Hapus kelak ditambahkan diam-diam.
func TestSeamProvidesNoDeleteOperation(t *testing.T) {
	var seam mastertipesurveyors.Repo = memory.NewRepo()

	_, sanggupHapus := seam.(interface {
		Delete(code string) error
	})
	require.False(t, sanggupHapus,
		"layar Pega tidak punya tombol hapus dan ADR-0012 melarang master dihapus permanen")
}
