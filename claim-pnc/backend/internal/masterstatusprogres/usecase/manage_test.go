package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/masterstatusprogres/repo/memory"
	"claim-pnc/internal/masterstatusprogres/usecase"
	"claim-pnc/internal/portal"
)

// twoPortals menyiapkan layanan dengan DUA penyimpanan terpisah, satu per entitas.
//
// Ini bentuk pengujian yang sengaja dipilih: cacat yang paling ingin dicegah modul ini
// adalah data satu badan hukum masuk ke basis data badan hukum lain (R-20), dan cacat
// itu hanya terlihat bila ada lebih dari satu penyimpanan.
func twoPortals(t *testing.T) (*usecase.Service, *memory.Repo, *memory.Repo) {
	t.Helper()

	asm := memory.NewRepo(memory.SampleList()...)
	asi := memory.NewRepo()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterstatusprogres.Repo, error) {
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

func TestServiceRejectsIncompleteDeps(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err, "rakitan setengah jadi harus gagal saat start, bukan saat pengguna bekerja")
}

func TestListReadFromRequestedPortal(t *testing.T) {
	service, _, _ := twoPortals(t)

	fromASM, err := service.List(context.Background(), "ASM")
	require.NoError(t, err)
	require.Len(t, fromASM, 6)

	fromASI, err := service.List(context.Background(), "ASI")
	require.NoError(t, err)
	require.Empty(t, fromASI, "entitas lain punya basis datanya sendiri (ADR-0030)")
}

// Penambahan pada satu entitas TIDAK boleh terlihat di entitas lain. Inilah pemisahan
// yang ditetapkan ADR-0030 dan yang dilindungi R-20.
func TestCreateDoesNotLeakBetweenPortals(t *testing.T) {
	service, _, _ := twoPortals(t)
	ctx := context.Background()

	saved, err := service.Create(ctx, "ASI", masterstatusprogres.Input{
		Name:         "MENUNGGU BERKAS",
		PositionCode: "REGISTER",
	})
	require.NoError(t, err)
	require.Equal(t, "01", saved.ID, "tabel ASI masih kosong, nomor mulai dari 1")

	inASI, err := service.List(ctx, "ASI")
	require.NoError(t, err)
	require.Len(t, inASI, 1)

	inASM, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, inASM, 6, "isi ASM tidak berubah sama sekali")
	for _, sp := range inASM {
		require.NotEqual(t, "MENUNGGU BERKAS", sp.Name)
	}
}

// Portal yang tidak dapat dilayani menghasilkan galat — TIDAK dialihkan ke portal utama.
func TestUnknownPortalRejectedNotRedirected(t *testing.T) {
	service, _, _ := twoPortals(t)
	ctx := context.Background()

	_, err := service.List(ctx, "SMAS")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(ctx, "SMAS", masterstatusprogres.Input{Name: "APA SAJA", PositionCode: "REGISTER"})
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Update(ctx, "SMAS", "01", masterstatusprogres.Input{Name: "APA SAJA", PositionCode: "REGISTER"})
	require.ErrorIs(t, err, portal.ErrNotReady)

	// Yang terpenting: tidak ada satu baris pun yang masuk ke entitas mana pun.
	inASM, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, inASM, 6)
	inASI, err := service.List(ctx, "ASI")
	require.NoError(t, err)
	require.Empty(t, inASI)
}

func TestNewIDContinuesFromHighest(t *testing.T) {
	service, _, _ := twoPortals(t)

	saved, err := service.Create(context.Background(), "ASM", masterstatusprogres.Input{
		Name:         "MENUNGGU PEMBAYARAN",
		PositionCode: "AKSEPTASI",
	})
	require.NoError(t, err)
	require.Equal(t, "07", saved.ID, "contoh berisi 01..06, berikutnya 07")
	require.Equal(t, "MENUNGGU PEMBAYARAN", saved.Name)
	require.Equal(t, "AKSEPTASI", saved.PositionCode)
}

// Input tidak sah ditolak SEBELUM menyentuh penyimpanan.
func TestInvalidInputNeverTouchesStorage(t *testing.T) {
	service, _, _ := twoPortals(t)
	ctx := context.Background()

	_, err := service.Create(ctx, "ASM", masterstatusprogres.Input{Name: "", PositionCode: "999"})
	var validationErr *masterstatusprogres.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.Len(t, validationErr.Violation, 2)

	list, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, list, 6, "tidak ada baris yang tersisip")
}

func TestUpdateSavesNameAndPositionWithoutChangingID(t *testing.T) {
	service, _, _ := twoPortals(t)
	ctx := context.Background()

	result, err := service.Update(ctx, "ASM", "03", masterstatusprogres.Input{
		Name:         "SURVEI DIJADWALKAN",
		PositionCode: "KOMITE",
	})
	require.NoError(t, err)
	require.Equal(t, "03", result.ID, "ID adalah kunci baris, bukan isian")
	require.Equal(t, "SURVEI DIJADWALKAN", result.Name)
	require.Equal(t, "KOMITE", result.PositionCode)

	loaded, err := service.Get(ctx, "ASM", "03")
	require.NoError(t, err)
	require.Equal(t, "SURVEI DIJADWALKAN", loaded.Name)

	// Jumlah baris tidak bertambah: penyuntingan bukan penambahan.
	list, err := service.List(ctx, "ASM")
	require.NoError(t, err)
	require.Len(t, list, 6)
}

func TestUpdateMissingRow(t *testing.T) {
	service, _, _ := twoPortals(t)

	_, err := service.Update(context.Background(), "ASM", "99", masterstatusprogres.Input{
		Name:         "APA SAJA",
		PositionCode: "REGISTER",
	})
	require.ErrorIs(t, err, masterstatusprogres.ErrNotFound)
}

// Galat penyimpanan diteruskan apa adanya, tidak ditelan menjadi daftar kosong.
func TestStorageErrorPropagated(t *testing.T) {
	service, asm, _ := twoPortals(t)
	storageErr := errors.New("basis data tidak dapat dihubungi")
	asm.SetError(storageErr)

	_, err := service.List(context.Background(), "ASM")
	require.ErrorIs(t, err, storageErr)
}

func TestPositionsServedFromOnePlace(t *testing.T) {
	service, _, _ := twoPortals(t)
	require.Equal(t, masterstatusprogres.ListPositions(), service.Position())
}
