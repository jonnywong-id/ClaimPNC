package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
	"claim-pnc/internal/masterpanel/repo/memory"
	"claim-pnc/internal/masterpanel/usecase"
)

const portalAlias = "ASM"

// newService merakit layanan di atas penyimpanan memori satu portal.
//
// Penyimpanan memori dipakai, bukan tiruan yang dibuat khusus untuk uji ini: ia adapter
// KEDUA yang membuat seam masterpanel.Repo nyata, dan menirunya sekali lagi di sini
// berarti menguji terhadap perilaku yang tidak dipakai siapa pun.
func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	store := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpanel.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return service, store
}

func validInput() masterpanel.Input {
	return masterpanel.Input{
		Name:                "Kap Mesin",
		RepairStatus:        "1",
		EditQuantityStatus:  "1",
		PremiumRepairStatus: "0",
		ShatterStatus:       "0",
		StickerStatus:       "1",
		SideStatus:          "-",
		SevereDamageStatus:  "0",
		ActiveStatus:        "1",
		ExclusionC:          "0",
		Location: []masterpanel.PanelLocation{
			{Name: "DEPAN", Side: masterpanel.SideNone},
		},
	}
}

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Portal yang tidak dikenal WAJIB ditolak, bukan dialihkan ke portal utama. Mengembalikan
// repo portal lain berarti menulis data satu badan hukum ke basis data badan hukum lain
// tanpa satu pun pesan galat (R-20).
func TestUnknownPortalRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), "SMI", masterpanel.StatusApproved, "")
	require.Error(t, err)

	require.Error(t, service.EnsurePortalReady("SMI"))
	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), portalAlias, masterpanel.ApprovalStatus("9"), "")
	require.ErrorIs(t, err, masterpanel.ErrUnknownStatus)
}

// Ketiga tab terisi dari contoh, termasuk tab Reject yang paling mudah terlupa diuji.
func TestListReturnsEachTab(t *testing.T) {
	service, _ := newService(t)

	for _, status := range []masterpanel.ApprovalStatus{
		masterpanel.StatusApproved,
		masterpanel.StatusPending,
		masterpanel.StatusRejected,
	} {
		list, err := service.List(context.Background(), portalAlias, status, "")
		require.NoError(t, err)
		require.NotEmpty(t, list, "tab %q harus terisi", status)
		for _, one := range list {
			require.Equal(t, status, one.Status)
		}
	}
}

// Daftar membawa lokasinya, bukan hanya baris induknya.
func TestListCarriesLocation(t *testing.T) {
	service, _ := newService(t)

	list, err := service.List(context.Background(), portalAlias, masterpanel.StatusApproved, "")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Len(t, list[0].Location, 2)
	require.Equal(t, "KIRI", list[0].Location[0].Name)
}

// Daftar diurutkan ID MENURUN, sama dengan adapter SQL dan sama dengan grid Pega.
//
// Ditiru di adapter memori juga: tanpa itu, urutan yang terlihat saat pengembangan berbeda
// dari urutan yang terlihat di produksi, dan uji yang memeriksa baris pertama akan lulus di
// satu tempat lalu gagal di tempat lain.
func TestListOrderIsIDDescending(t *testing.T) {
	service, store := newService(t)

	extra := masterpanel.Panel{
		ID: "01000010", Name: "Atap", RepairStatus: "1", EditQuantityStatus: "1",
		PremiumRepairStatus: "0", ShatterStatus: "0", StickerStatus: "1", SideStatus: "-",
		SevereDamageStatus: "0", ActiveStatus: "1", ExclusionC: "0",
		Status: masterpanel.StatusApproved,
	}
	require.NoError(t, store.Insert(context.Background(), extra))

	list, err := service.List(context.Background(), portalAlias, masterpanel.StatusApproved, "")
	require.NoError(t, err)
	require.Len(t, list, 2)

	// 01000010 lebih besar daripada 01000001, sehingga ia di puncak — bukan "Atap" yang
	// lebih dulu menurut abjad.
	require.Equal(t, "01000010", list[0].ID)
	require.Equal(t, "01000001", list[1].ID)
}

func TestListKeywordNarrowsByName(t *testing.T) {
	service, _ := newService(t)

	found, err := service.List(context.Background(), portalAlias, masterpanel.StatusApproved, "pintu")
	require.NoError(t, err)
	require.Len(t, found, 1)

	none, err := service.List(context.Background(), portalAlias, masterpanel.StatusApproved, "tidak ada")
	require.NoError(t, err)
	require.Empty(t, none)
}

// Penambahan menerbitkan ID berbentuk kode situs + ENAM digit, dan lahir berstatus
// menunggu.
func TestCreateIssuesKeyAndStartsPending(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(), usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.Equal(t, "01000004", saved.ID, "kode situs 01 ditambah nomor urut keempat")
	require.Equal(t, masterpanel.StatusPending, saved.Status)
	require.Len(t, saved.Location, 1)
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.RepairStatus = ""

	_, err := service.Create(context.Background(), portalAlias, input, usecase.Actor{}, nil)

	var validationError *masterpanel.ValidationError
	require.ErrorAs(t, err, &validationError)
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.Name = "Pintu Depan" // sudah ada di contoh

	_, err := service.Create(context.Background(), portalAlias, input, usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterpanel.ErrNameTaken)
}

// Menyimpan SELALU mengembalikan baris ke antrean persetujuan.
//
// Itu bukan efek samping melainkan langkah tersendiri di sistem lama:
// `Activity/CNMUpdatePanelHE_act` menetapkan APPROVAL := "0" tanpa syarat apa pun.
func TestSaveAlwaysReturnsRowToTheQueue(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.Name = "Pintu Depan Diperbarui"

	saved, err := service.Save(context.Background(), portalAlias, "01000001", input, usecase.Actor{}, nil)
	require.NoError(t, err)
	require.Equal(t, masterpanel.StatusPending, saved.Status,
		"baris yang sudah disetujui lalu disunting harus kembali menunggu")
}

// Menyimpan mengganti SELURUH daftar lokasi, bukan menggabungkannya.
func TestSaveReplacesTheWholeLocationList(t *testing.T) {
	service, _ := newService(t)

	before, err := service.Get(context.Background(), portalAlias, "01000001")
	require.NoError(t, err)
	require.Len(t, before.Location, 2)

	input := validInput()
	input.Name = before.Name
	input.Location = []masterpanel.PanelLocation{{Name: "BELAKANG", Side: masterpanel.SideNone}}

	saved, err := service.Save(context.Background(), portalAlias, "01000001", input, usecase.Actor{}, nil)
	require.NoError(t, err)
	require.Len(t, saved.Location, 1)
	require.Equal(t, "BELAKANG", saved.Location[0].Name)
}

// Menyimpan tanpa mengubah nama TIDAK boleh ditolak karena namanya sendiri sudah dipakai
// oleh dirinya sendiri.
func TestSaveKeepingItsOwnNameIsAccepted(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.Name = "Pintu Depan"

	_, err := service.Save(context.Background(), portalAlias, "01000001", input, usecase.Actor{}, nil)
	require.NoError(t, err)
}

// Menyimpan dengan nama yang dipakai BARIS LAIN ditolak.
func TestSaveWithAnotherRowsNameRejected(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.Name = "Kaca Depan" // milik 01000002

	_, err := service.Save(context.Background(), portalAlias, "01000001", input, usecase.Actor{}, nil)

	var validationError *masterpanel.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Equal(t, "nama_panel", validationError.Violation[0].Field)
}

// Empat kolom TIDAK ikut berubah saat disimpan; ketiganya diturunkan sistem atau diisi
// jalur lain.
func TestSaveKeepsDerivedColumns(t *testing.T) {
	service, store := newService(t)

	seeded := masterpanel.Panel{
		ID:                  "01000009",
		Name:                "Spakbor",
		RepairStatus:        "1",
		EditQuantityStatus:  "1",
		PremiumRepairStatus: "1",
		ShatterStatus:       "1",
		StickerStatus:       "1",
		SideStatus:          "1",
		SevereDamageStatus:  "1",
		ActiveStatus:        "1",
		ExclusionC:          "1",
		ApprovalMark:        "TANDA",
		RejectReason:        "alasan lama",
		DocumentID:          "DOK-1",
		Status:              masterpanel.StatusApproved,
	}
	require.NoError(t, store.Insert(context.Background(), seeded))

	input := validInput()
	input.Name = "Spakbor"

	saved, err := service.Save(context.Background(), portalAlias, "01000009", input, usecase.Actor{}, nil)
	require.NoError(t, err)

	require.Equal(t, "01000009", saved.ID, "kunci tidak berpindah")
	require.Equal(t, "TANDA", saved.ApprovalMark, "STS_APPROVAL tidak punya isian di layar mana pun")
	require.Equal(t, "alasan lama", saved.RejectReason, "ALASAN_TOLAK diisi jalur keputusan")
	require.Equal(t, "DOK-1", saved.DocumentID, "lampiran yang sudah ada tidak boleh lenyap")
}

func TestSaveRejectsMissingKey(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Save(context.Background(), portalAlias, "   ", validInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
}

func TestGetUnknownKeyIsNotFound(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Get(context.Background(), portalAlias, "tidak-ada")
	require.ErrorIs(t, err, masterpanel.ErrNotFound)
}

// Keputusan borongan menetapkan status seluruh baris yang dipilih sekaligus.
func TestDecideMovesEveryChosenRow(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000002"}, masterpanel.StatusApproved, "", usecase.Actor{}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	moved, err := service.Get(context.Background(), portalAlias, "01000002")
	require.NoError(t, err)
	require.Equal(t, masterpanel.StatusApproved, moved.Status)
}

// Alasan hanya tersimpan pada keputusan TOLAK.
//
// Menuliskannya pada persetujuan akan mengisi kolom bernama "alasan tolak" pada baris yang
// justru disetujui.
func TestReasonSavedOnlyOnRejection(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000002"}, masterpanel.StatusRejected, "Nama tidak baku.", usecase.Actor{}, nil)
	require.NoError(t, err)

	rejected, err := service.Get(context.Background(), portalAlias, "01000002")
	require.NoError(t, err)
	require.Equal(t, "Nama tidak baku.", rejected.RejectReason)

	_, err = service.Decide(context.Background(), portalAlias,
		[]string{"01000002"}, masterpanel.StatusApproved, "catatan yang tidak seharusnya tersimpan",
		usecase.Actor{}, nil)
	require.NoError(t, err)

	approved, err := service.Get(context.Background(), portalAlias, "01000002")
	require.NoError(t, err)
	require.Empty(t, approved.RejectReason)
}

// Kunci ganda dibuang supaya jumlah yang dilaporkan mencerminkan BARIS, bukan centang.
func TestDecideIgnoresDuplicateKeys(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000002", "01000002", "  "}, masterpanel.StatusApproved, "", usecase.Actor{}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestDecideWithoutSelectionRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias,
		[]string{"  ", ""}, masterpanel.StatusApproved, "", usecase.Actor{}, nil)

	var validationError *masterpanel.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Equal(t, "id_panel", validationError.Violation[0].Field)
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000002"}, masterpanel.ApprovalStatus("9"), "", usecase.Actor{}, nil)
	require.ErrorIs(t, err, masterpanel.ErrUnknownStatus)
}

func TestDecideRejectsOverlongReason(t *testing.T) {
	service, _ := newService(t)

	long := make([]byte, masterpanel.MaxReasonLength+1)
	for i := range long {
		long[i] = 'A'
	}

	_, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000002"}, masterpanel.StatusRejected, string(long), usecase.Actor{}, nil)

	var validationError *masterpanel.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Equal(t, "catatan", validationError.Violation[0].Field)
}

// Baris yang sudah berstatus itu tidak ikut terhitung sebagai berubah.
func TestDecideDoesNotCountRowsAlreadyInThatStatus(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"01000001"}, masterpanel.StatusApproved, "", usecase.Actor{}, nil)
	require.NoError(t, err)
	require.Zero(t, changed, "01000001 memang sudah disetujui")
}

// Kegagalan penyimpanan diteruskan apa adanya, bukan ditelan.
func TestStoreFailureIsPropagated(t *testing.T) {
	service, store := newService(t)
	failure := errors.New("basis data tidak dapat dihubungi")
	store.SetError(failure)

	_, err := service.List(context.Background(), portalAlias, masterpanel.StatusApproved, "")
	require.ErrorIs(t, err, failure)

	_, err = service.Create(context.Background(), portalAlias, validInput(), usecase.Actor{}, nil)
	require.ErrorIs(t, err, failure)
}
