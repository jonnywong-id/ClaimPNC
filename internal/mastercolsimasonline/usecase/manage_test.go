package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastercolsimasonline"
	"claim-pnc/internal/mastercolsimasonline/repo/memory"
	"claim-pnc/internal/mastercolsimasonline/usecase"
)

const portalAlias = "ASM"

// harness merakit layanan di atas repo memori, beserta kedua repo-nya supaya uji dapat
// memeriksa keadaan yang benar-benar tersimpan — bukan hanya nilai yang dikembalikan.
type harness struct {
	service  *usecase.Service
	repo     *memory.Repo
	business *memory.BusinessRepo
}

func newHarness(t *testing.T, rows ...mastercolsimasonline.CauseOfLoss) harness {
	t.Helper()

	business := memory.NewBusinessRepo(memory.SampleBusinessList()...)
	repo := memory.NewRepo(rows...)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastercolsimasonline.Repo, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		BusinessSelector: func(alias string) (mastercolsimasonline.BusinessRepo, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return business, nil
		},
	})
	require.NoError(t, err)

	return harness{service: service, repo: repo, business: business}
}

func validInput() mastercolsimasonline.Input {
	return mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"FIRE / PROPERTY", "ANEKA"},
	}
}

// Rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func TestServiceRefusesIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastercolsimasonline.Repo, error) { return nil, nil },
	})
	require.Error(t, err, "BusinessSelector wajib: tanpa itu nama bisnis tidak dapat diselesaikan")
}

func TestCreateIssuesTheCodeItselfAndStoresTheRow(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, validInput())
	require.NoError(t, err)

	require.NotEmpty(t, saved.Code, "kode diterbitkan penyimpanan, bukan dikirim pemanggil")
	require.Equal(t, "BANJIR", saved.Description)
	require.Len(t, saved.Businesses, 2)
}

// Nama bisnis diselesaikan menjadi ID dengan mencocokkan ke master — cara yang sama
// dengan autocomplete Pega yang mengisi `.ID` saat sebuah pilihan diambil dari daftar.
func TestBusinessNameIsResolvedToItsIDFromTheMaster(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"FIRE / PROPERTY"},
	})
	require.NoError(t, err)
	require.Equal(t, "006", saved.Businesses[0].ID)
	require.Equal(t, "FIRE / PROPERTY", saved.Businesses[0].Name)
}

// Pencocokan mengabaikan besar-kecil huruf, DAN nama yang tersimpan adalah ejaan resmi
// masternya — bukan ejaan yang diketik pengguna. Tanpa itu, satu bisnis yang sama dapat
// tampil dalam dua ejaan di layar.
func TestResolvedNameUsesTheMasterSpellingNotTheTypedOne(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"  fire / property  "},
	})
	require.NoError(t, err)
	require.Equal(t, "006", saved.Businesses[0].ID)
	require.Equal(t, "FIRE / PROPERTY", saved.Businesses[0].Name)
}

// PERILAKU PEGA YANG DIPERTAHANKAN (`pyAllowFreeFormInput=true`, Work Owner 2026-09-21):
// nama di luar master TETAP tersimpan, dengan ID kosong. Ia bukan galat.
func TestBusinessNameOutsideTheMasterIsStoredWithoutAnID(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"FIRE / PROPERTY", "BENGKEL BARU"},
	})
	require.NoError(t, err)
	require.Len(t, saved.Businesses, 2)

	require.Equal(t, "006", saved.Businesses[0].ID)
	require.Empty(t, saved.Businesses[1].ID, "nama bebas memang tidak punya ID")
	require.Equal(t, "BENGKEL BARU", saved.Businesses[1].Name)
}

// Urutan yang disusun pengguna dipertahankan sampai ke penyimpanan.
func TestBusinessOrderSurvivesSaving(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"TRAVEL", "ANEKA", "MARINE CARGO"},
	})
	require.NoError(t, err)

	names := []string{}
	for _, b := range saved.Businesses {
		names = append(names, b.Name)
	}
	require.Equal(t, []string{"TRAVEL", "ANEKA", "MARINE CARGO"}, names)
}

func TestCreateTrimsBeforeStoring(t *testing.T) {
	h := newHarness(t)

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description: "  BANJIR  ",
	})
	require.NoError(t, err)
	require.Equal(t, "BANJIR", saved.Description)
}

func TestUpdateChangesValuesButNeverTheCode(t *testing.T) {
	h := newHarness(t, memory.SampleList()...)

	updated, err := h.service.Update(context.Background(), portalAlias, "1002", mastercolsimasonline.Input{
		Description:   "KEBAKARAN AKIBAT PETIR DAN KORSLETING",
		BusinessNames: []string{"ANEKA"},
	})
	require.NoError(t, err)

	require.Equal(t, "1002", updated.Code, "kode tidak pernah ikut berubah")
	require.Equal(t, "KEBAKARAN AKIBAT PETIR DAN KORSLETING", updated.Description)
}

// Pemetaan bisnis DIGANTI seluruhnya, bukan digabung: layar mengirim keadaan akhir grid
// apa adanya, sehingga baris yang dihapus pengguna memang harus hilang.
func TestUpdateReplacesTheBusinessMappingInsteadOfMergingIt(t *testing.T) {
	h := newHarness(t, memory.SampleList()...)

	before, err := h.service.Get(context.Background(), portalAlias, "1001")
	require.NoError(t, err)
	require.Len(t, before.Businesses, 2)

	after, err := h.service.Update(context.Background(), portalAlias, "1001", mastercolsimasonline.Input{
		Description:   "KEBAKARAN",
		BusinessNames: []string{"ANEKA"},
	})
	require.NoError(t, err)
	require.Len(t, after.Businesses, 1)
	require.Equal(t, "003", after.Businesses[0].ID)
}

func TestUpdateOnMissingRowIsNotFound(t *testing.T) {
	h := newHarness(t)

	_, err := h.service.Update(context.Background(), portalAlias, "9999", validInput())
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)
}

func TestGetOnMissingRowIsNotFound(t *testing.T) {
	h := newHarness(t)

	_, err := h.service.Get(context.Background(), portalAlias, "9999")
	require.ErrorIs(t, err, mastercolsimasonline.ErrNotFound)
}

// Daftar tidak membawa pemetaan bisnis: grid layar hanya menampilkan ID dan nama, dan
// menariknya untuk seluruh baris adalah kueri yang hasilnya tidak pernah dilihat.
func TestListDoesNotCarryTheBusinessMapping(t *testing.T) {
	h := newHarness(t, memory.SampleList()...)

	list, err := h.service.List(context.Background(), portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, row := range list {
		require.Empty(t, row.Businesses, "pemetaan bisnis hanya dimuat lewat Get")
	}
}

// Portal yang tidak dikenal WAJIB menghasilkan galat, tidak pernah dialihkan ke portal
// utama sebagai cadangan — itu kebocoran lintas badan hukum yang dicegah `R-20`.
func TestUnknownPortalIsRefusedOnEveryOperation(t *testing.T) {
	h := newHarness(t, memory.SampleList()...)
	ctx := context.Background()

	_, err := h.service.List(ctx, "ENTITAS-LAIN")
	require.Error(t, err)

	_, err = h.service.Get(ctx, "ENTITAS-LAIN", "1001")
	require.Error(t, err)

	_, err = h.service.Create(ctx, "ENTITAS-LAIN", validInput())
	require.Error(t, err)

	_, err = h.service.Update(ctx, "ENTITAS-LAIN", "1001", validInput())
	require.Error(t, err)

	_, err = h.service.ListBusiness(ctx, "ENTITAS-LAIN")
	require.Error(t, err)

	require.Error(t, h.service.EnsurePortalReady("ENTITAS-LAIN"))
	require.NoError(t, h.service.EnsurePortalReady(portalAlias))
}

// Kegagalan penyimpanan diteruskan apa adanya, bukan ditelan dan dijawab "berhasil".
func TestStorageFailureIsPropagated(t *testing.T) {
	h := newHarness(t)
	failure := errors.New("basis data tidak dapat dihubungi")
	h.repo.SetError(failure)

	_, err := h.service.List(context.Background(), portalAlias)
	require.ErrorIs(t, err, failure)
}

// Kegagalan membaca master bisnis TIDAK menggagalkan penyimpanan.
//
// Karena nama bebas memang diterima, master hanya dipakai untuk MELENGKAPI ID — dan
// melengkapi yang gagal lebih baik daripada menolak penyimpanan yang sebenarnya sah.
// Yang hilang hanya ID-nya, dan itu keadaan yang memang sudah harus ditangani setiap
// pembaca.
func TestBusinessMasterFailureDoesNotBlockSaving(t *testing.T) {
	h := newHarness(t)
	h.business.SetError(errors.New("basis data tidak dapat dihubungi"))

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description:   "BANJIR",
		BusinessNames: []string{"FIRE / PROPERTY"},
	})
	require.NoError(t, err)
	require.Len(t, saved.Businesses, 1)
	require.Equal(t, "FIRE / PROPERTY", saved.Businesses[0].Name)
	require.Empty(t, saved.Businesses[0].ID, "ID tidak dapat dilengkapi saat master tidak terbaca")
}

// Bila tidak ada bisnis yang dipilih, master bisnis tidak perlu dibaca sama sekali.
func TestBusinessMasterIsNotReadWhenNoBusinessIsChosen(t *testing.T) {
	h := newHarness(t)
	h.business.SetError(errors.New("basis data tidak dapat dihubungi"))

	saved, err := h.service.Create(context.Background(), portalAlias, mastercolsimasonline.Input{
		Description: "BANJIR",
	})
	require.NoError(t, err)
	require.Empty(t, saved.Businesses)
}
