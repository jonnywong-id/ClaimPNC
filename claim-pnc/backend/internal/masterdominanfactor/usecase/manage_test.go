package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterdominanfactor"
	"claim-pnc/internal/masterdominanfactor/repo/memory"
	"claim-pnc/internal/masterdominanfactor/usecase"
	"claim-pnc/internal/portal"
)

const portalASM = "ASM"

// serviceWith membentuk layanan di atas dua entitas: ASM berisi, ASI kosong.
//
// ASI sengaja kosong — ia yang membuktikan pemisahan antarentitas benar-benar terjadi,
// bukan sekadar dinyatakan.
func serviceWith(t *testing.T, list ...masterdominanfactor.DominantFactor) (*usecase.Service, *memory.Repo, *memory.Repo) {
	t.Helper()

	asm := memory.NewRepo(list...)
	asi := memory.NewRepo()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterdominanfactor.Repo, error) {
			switch alias {
			case "ASM":
				return asm, nil
			case "ASI":
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)
	return service, asm, asi
}

func TestNewServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err, "kegagalan terjadi saat start, bukan saat pengguna pertama membuka layar")
}

func TestListReturnsSortedNumerically(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 10)
	require.Equal(t, "1", list[0].ID)
	require.Equal(t, "10", list[9].ID,
		"urutan numerik: `10` di belakang `9`, bukan di belakang `1`")
}

// Portal yang koneksinya belum hidup WAJIB menghasilkan galat, bukan dilayani portal
// utama sebagai cadangan. Jalan pintas itu berarti membaca data satu badan hukum dari
// basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestUnknownPortalRejected(t *testing.T) {
	service, _, _ := serviceWith(t)

	_, err := service.List(context.Background(), "SMAS")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

func TestEnsurePortalReady(t *testing.T) {
	service, _, _ := serviceWith(t)

	require.NoError(t, service.EnsurePortalReady(portalASM))
	require.Error(t, service.EnsurePortalReady("SMAS"))
}

// Entitas yang satu tidak boleh melihat isi entitas lain. Ini pemeriksaan pemisahan yang
// sesungguhnya, bukan sekadar membaca komentar di seam-nya.
func TestEntitiesDoNotSeeEachOther(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)
	ctx := context.Background()

	atASM, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.NotEmpty(t, atASM)

	atASI, err := service.List(ctx, "ASI")
	require.NoError(t, err)
	require.Empty(t, atASI, "ASI kosong; isinya tidak boleh datang dari ASM")

	// Menambah di ASI tidak boleh terlihat di ASM.
	_, err = service.Create(ctx, "ASI", "Hanya milik ASI")
	require.NoError(t, err)

	atASM, err = service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, atASM, 10, "isi ASM tidak berubah oleh penambahan di ASI")
}

func TestCreateIssuesNextID(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)

	factor, err := service.Create(context.Background(), portalASM, "Faktor baru")
	require.NoError(t, err)
	require.Equal(t, "11", factor.ID, "melanjutkan dari 10, tanpa nol di depan")
	require.Equal(t, "Faktor baru", factor.Name)
}

func TestCreateOnEmptyMasterStartsAtOne(t *testing.T) {
	service, _, _ := serviceWith(t)

	factor, err := service.Create(context.Background(), portalASM, "Pertama")
	require.NoError(t, err)
	require.Equal(t, "1", factor.ID,
		"nvl(max(to_number(ID)),0)+1 pada tabel kosong menghasilkan 1")
}

// Nama kosong dan nama ganda DITERIMA (keputusan Work Owner 2026-09-20). Keduanya diuji
// bersama karena keduanya adalah keputusan yang sama: meniru layar Pega apa adanya.
func TestCreateAcceptsEmptyAndDuplicateNames(t *testing.T) {
	service, _, _ := serviceWith(t)
	ctx := context.Background()

	empty, err := service.Create(ctx, portalASM, "   ")
	require.NoError(t, err, "layar Pega menerimanya; P-5 menetapkan perilaku dipertahankan")
	require.Empty(t, empty.Name, "spasi tepi tetap dibuang sebelum disimpan")

	first, err := service.Create(ctx, portalASM, "Faktor Alam")
	require.NoError(t, err)
	second, err := service.Create(ctx, portalASM, "Faktor Alam")
	require.NoError(t, err, "tidak ada constraint keunikan di sistem lama, dan tidak ditambahkan")

	require.NotEqual(t, first.ID, second.ID, "keduanya baris berbeda dengan nomor berbeda")
}

func TestCreateRejectsNameBeyondLimit(t *testing.T) {
	service, _, _ := serviceWith(t)

	_, err := service.Create(context.Background(), portalASM,
		strings.Repeat("a", masterdominanfactor.MaxNameLength+1))

	var validation *masterdominanfactor.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violation, 1)
}

func TestUpdateChangesNameOnly(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)

	updated, err := service.Update(context.Background(), portalASM, "3", "Nama baru")
	require.NoError(t, err)
	require.Equal(t, "3", updated.ID, "ID tidak pernah ikut berubah")
	require.Equal(t, "Nama baru", updated.Name)
}

// UPDATE terhadap ID yang tidak ada berhasil tanpa galat di SQL. Procedure lama
// melakukan persis itu — ia mengembalikan "Data Sudah Diupdate dengan ID : ..." tanpa
// memeriksa satu baris pun tersentuh.
func TestUpdateUnknownIDReportsNotFound(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)

	_, err := service.Update(context.Background(), portalASM, "999", "apa pun")
	require.ErrorIs(t, err, masterdominanfactor.ErrNotFound)
}

func TestGetUnknownIDReportsNotFound(t *testing.T) {
	service, _, _ := serviceWith(t, memory.SampleList()...)

	_, err := service.Get(context.Background(), portalASM, "999")
	require.ErrorIs(t, err, masterdominanfactor.ErrNotFound)
}

// Galat penyimpanan yang bukan galat domain dibungkus dengan konteks, bukan diteruskan
// telanjang — supaya jejaknya terbaca dari pesannya.
func TestStorageFailureWrapped(t *testing.T) {
	service, asm, _ := serviceWith(t)
	failure := errors.New("koneksi terputus")
	asm.SetError(failure)

	_, err := service.List(context.Background(), portalASM)
	require.ErrorIs(t, err, failure)
	require.Contains(t, err.Error(), "masterdominanfactor/usecase")
}
