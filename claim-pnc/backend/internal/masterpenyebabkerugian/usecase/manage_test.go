package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpenyebabkerugian"
	"claim-pnc/internal/masterpenyebabkerugian/repo/memory"
	"claim-pnc/internal/masterpenyebabkerugian/usecase"
	"claim-pnc/internal/portal"
)

const portalASM = "ASM"

// newService membentuk layanan dengan dua entitas: ASM berisi contoh, ASI KOSONG.
// ASI yang kosong itulah yang membuktikan pemisahan antarentitas benar-benar terjadi.
func newService(t *testing.T) (*usecase.Service, *memory.Repo, *memory.Repo) {
	t.Helper()

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpenyebabkerugian.Repo, error) {
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

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err, "kegagalan terjadi saat start, bukan saat pengguna membuka layar")
}

func TestEnsurePortalReady(t *testing.T) {
	service, _, _ := newService(t)

	require.NoError(t, service.EnsurePortalReady(portalASM))
	require.Error(t, service.EnsurePortalReady("SMAS"),
		"portal yang koneksinya belum hidup tidak boleh dinyatakan siap")
}

func TestListReturnsEntityContentInOrder(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), portalASM)
	require.NoError(t, err)
	require.Len(t, list, 10)
	require.Equal(t, "1001", list[0].ID)
	require.Equal(t, "1010", list[9].ID)
}

// Entitas yang satu tidak melihat isi entitas lain. Pemisahannya ada di tingkat KONEKSI,
// bukan penyaringan baris (`ADR-0030` Opsi 1, `R-20`).
func TestListSeparatesEntities(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, list, "ASI kosong; isinya tidak boleh datang dari ASM")
}

// Portal yang tidak dikenal WAJIB menghasilkan galat — tidak pernah jatuh ke portal utama
// sebagai cadangan. Jalan pintas itu adalah `R-20` yang sesungguhnya.
func TestUnknownPortalRejected(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.List(context.Background(), "TIDAK-ADA")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

func TestGetReturnsOneRow(t *testing.T) {
	service, _, _ := newService(t)

	cause, err := service.Get(context.Background(), portalASM, "1003")
	require.NoError(t, err)
	require.Equal(t, "1003", cause.ID)
}

// Spasi tepi pada ID dibuang sebelum menyentuh penyimpanan — kolomnya mungkin CHAR
// berpadding, dan pengguna dapat menyalin ID beserta spasinya.
func TestGetTrimsID(t *testing.T) {
	service, _, _ := newService(t)

	cause, err := service.Get(context.Background(), portalASM, "  1003  ")
	require.NoError(t, err)
	require.Equal(t, "1003", cause.ID)
}

func TestGetUnknownIDReturnsNotFound(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Get(context.Background(), portalASM, "9999")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)
}

// ID dibuat penyimpanan, tidak pernah datang dari pemanggil, dan ia melanjutkan deret
// yang sudah ada.
func TestCreateIssuesNextID(t *testing.T) {
	service, _, _ := newService(t)

	cause, err := service.Create(context.Background(), portalASM, "Golongan baru")
	require.NoError(t, err)
	require.Equal(t, "1011", cause.ID, "melanjutkan dari 1010")
	require.Equal(t, "Golongan baru", cause.Description)
	require.Empty(t, cause.LegacyID, "penomoran lama tidak pernah diberikan pada baris baru")
}

// Deskripsi kosong dan deskripsi ganda DITERIMA — keputusan Work Owner 2026-09-20,
// meniru layar Pega apa adanya (`P-5`).
func TestCreateAcceptsEmptyAndDuplicateDescriptions(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	empty, err := service.Create(ctx, portalASM, "")
	require.NoError(t, err)
	require.Empty(t, empty.Description)

	first, err := service.Create(ctx, portalASM, "Kembar")
	require.NoError(t, err)

	second, err := service.Create(ctx, portalASM, "Kembar")
	require.NoError(t, err, "tidak ada constraint keunikan di sistem lama, dan tidak ditambahkan")
	require.NotEqual(t, first.ID, second.ID, "keduanya baris terpisah dengan ID berbeda")
}

func TestCreateRejectsTooLongDescription(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Create(context.Background(), portalASM,
		strings.Repeat("a", masterpenyebabkerugian.MaxDescriptionLength+1))

	var validation *masterpenyebabkerugian.ValidationError
	require.True(t, errors.As(err, &validation))
	require.Len(t, validation.Violation, 1)
}

// Validasi dijalankan SEBELUM penyimpanan disentuh: isian yang ditolak tidak boleh
// menghabiskan satu nomor urut pun.
func TestCreateValidatesBeforeTouchingStorage(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	_, err := service.Create(ctx, portalASM,
		strings.Repeat("a", masterpenyebabkerugian.MaxDescriptionLength+1))
	require.Error(t, err)

	after, err := service.Create(ctx, portalASM, "Berhasil")
	require.NoError(t, err)
	require.Equal(t, "1011", after.ID, "nomor urut tidak terbuang oleh isian yang ditolak")
}

func TestUpdateChangesDescriptionOnly(t *testing.T) {
	service, _, _ := newService(t)

	before, err := service.Get(context.Background(), portalASM, "1001")
	require.NoError(t, err)
	require.NotEmpty(t, before.LegacyID, "baris ini memang punya penomoran lama")

	after, err := service.Update(context.Background(), portalASM, "1001", "Deskripsi diperbarui")
	require.NoError(t, err)
	require.Equal(t, "1001", after.ID, "ID tidak pernah ikut berubah")
	require.Equal(t, before.LegacyID, after.LegacyID, "ID lama adalah jejak sejarah, bukan isian")
	require.Equal(t, "Deskripsi diperbarui", after.Description)
}

func TestUpdateUnknownIDReturnsNotFound(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Update(context.Background(), portalASM, "9999", "apa pun")
	require.ErrorIs(t, err, masterpenyebabkerugian.ErrNotFound)
}

// Galat penyimpanan yang bukan milik domain dibungkus konteks, bukan diteruskan telanjang
// — jejaknya harus terbaca dari pesannya.
func TestStorageFailureIsWrapped(t *testing.T) {
	service, asm, _ := newService(t)
	asm.SetError(errors.New("koneksi terputus"))

	_, err := service.List(context.Background(), portalASM)
	require.Error(t, err)
	require.Contains(t, err.Error(), "masterpenyebabkerugian/usecase")
	require.Contains(t, err.Error(), "koneksi terputus")
}
