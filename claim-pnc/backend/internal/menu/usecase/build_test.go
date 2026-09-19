package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/menu/repo/memory"
	"claim-pnc/internal/menu/usecase"
)

func service(t *testing.T, repo menu.Repo) *usecase.Service {
	t.Helper()
	s, err := usecase.NewService(usecase.Options{Repo: repo})
	require.NoError(t, err)
	return s
}

// Rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna bekerja.
func TestServiceRejectsIncompleteDeps(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Login `JONNY` anggota group `IT`. Izin keduanya BERBEDA, dan keduanya harus berlaku:
// IT memberi MASTER/INBOX/VIEW, JONNY memberi REPORT.
func TestGrantsOfGroupAndLoginAreCombined(t *testing.T) {
	tree, err := service(t, memory.NewSampleRepo()).ForLogin(context.Background(), "JONNY")
	require.NoError(t, err)

	var groups []string
	for _, node := range tree {
		groups = append(groups, node.Description)
	}
	require.Equal(t, []string{"MASTER", "INBOX", "VIEW", "REPORT"}, groups,
		"REPORT hanya datang dari izin login, sisanya dari izin group")
}

// Tanpa keanggotaan group, yang berlaku hanyalah izin atas namanya sendiri.
func TestLoginWithoutGroupKeepsItsOwnGrants(t *testing.T) {
	repo := memory.NewRepo(memory.SampleItems(), nil, memory.SampleGrants())

	tree, err := service(t, repo).ForLogin(context.Background(), "JONNY")
	require.NoError(t, err)

	require.Len(t, tree, 1)
	require.Equal(t, "REPORT", tree[0].Description)
	require.Len(t, tree[0].Children, 5, "MENU_ID 82..86")
}

// Group `IT` tidak diberi izin atas satu pun kelompok tingkat atas. Kelompoknya tetap
// harus muncul, diturunkan dari anak-anaknya.
func TestGroupOnlyGrantsStillProduceTheirHeadings(t *testing.T) {
	repo := memory.NewRepo(
		memory.SampleItems(),
		map[string][]string{"SURVEYOR": {"IT"}},
		memory.SampleGrants(),
	)

	tree, err := service(t, repo).ForLogin(context.Background(), "SURVEYOR")
	require.NoError(t, err)

	var groups []string
	for _, node := range tree {
		groups = append(groups, node.Description)
	}
	require.Equal(t, []string{"MASTER", "INBOX", "VIEW"}, groups)
	require.NotContains(t, groups, "REPORT", "izin REPORT milik login JONNY, bukan group IT")
}

// Pengguna yang tidak punya satu pun izin melihat menu kosong — bukan menu penuh, dan
// bukan galat.
func TestLoginWithoutAnyGrantSeesNothing(t *testing.T) {
	tree, err := service(t, memory.NewSampleRepo()).ForLogin(context.Background(), "ORANGLAIN")
	require.NoError(t, err)
	require.Empty(t, tree)
}

// Kolomnya VARCHAR2 tanpa penyeragaman. Satu spasi di ujung tidak boleh menghapus
// seluruh menu seseorang.
func TestLoginIsMatchedWithoutRegardToCaseOrSpace(t *testing.T) {
	tree, err := service(t, memory.NewSampleRepo()).ForLogin(context.Background(), "  jonny  ")
	require.NoError(t, err)
	require.NotEmpty(t, tree)
}

func TestEmptyLoginSeesNothing(t *testing.T) {
	tree, err := service(t, memory.NewSampleRepo()).ForLogin(context.Background(), "   ")
	require.NoError(t, err)
	require.Empty(t, tree)
}

// Kegagalan penyimpanan diteruskan apa adanya. Menelannya akan menampilkan menu kosong
// yang tidak dapat dibedakan dari "memang tidak punya izin".
func TestStorageErrorPropagated(t *testing.T) {
	repo := memory.NewSampleRepo()
	storageErr := errors.New("basis data tidak dapat dihubungi")
	repo.SetError(storageErr)

	_, err := service(t, repo).ForLogin(context.Background(), "JONNY")
	require.ErrorIs(t, err, storageErr)
}

// Butir menu yang tampil membawa MENU_PROGRAM-nya. Frontend yang memutuskan layarnya
// sudah ada atau belum, dan ia hanya dapat melakukannya bila namanya ikut dikirim.
func TestVisibleItemCarriesItsProgram(t *testing.T) {
	tree, err := service(t, memory.NewSampleRepo()).ForLogin(context.Background(), "JONNY")
	require.NoError(t, err)

	program := map[string]string{}
	for _, group := range tree {
		for _, child := range group.Children {
			program[child.Description] = child.Program
		}
	}

	require.Equal(t, "StatusClaimInbox", program["Master Status Klaim"])
	require.Equal(t, "MasterRekening", program["Master Rekening"])
	require.Equal(t, "StatusProgress", program["Master Status Progress 1"])
	// MENU_ID 83 ada di master tanpa program; ia tetap tampil apa adanya.
	require.Equal(t, "", program["Report Adjuster"])
}
