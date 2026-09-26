package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastergroupingsparepart"
	"claim-pnc/internal/mastergroupingsparepart/repo/memory"
	"claim-pnc/internal/mastergroupingsparepart/usecase"
)

const portalAlias = "asm"

// newService merakit layanan di atas penyimpanan contoh.
//
// Repo-nya dikembalikan supaya uji dapat memeriksa apa yang benar-benar tersimpan — bukan
// hanya apa yang dikembalikan layanan.
func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastergroupingsparepart.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

// validInput adalah isian yang lolos seluruh pemeriksaan DAN menunjuk sparepart yang ada.
//
// Nomor rangkanya sengaja BARU — supaya penambahan tidak bertabrakan dengan baris contoh, dan
// supaya grup yang terbit benar-benar grup baru.
func validInput() mastergroupingsparepart.Input {
	return mastergroupingsparepart.Input{
		PartNumber:    "SP-1002",
		PanelID:       "PNL02",
		PanelName:     "KABIN",
		PanelSide:     "2",
		ChassisNumber: "MHFNEW0001K0000001",
		VehicleType:   "EXCAVATOR",
		Note:          "Baris baru.",
	}
}

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Penambahan mengisi kelima isian turunan DARI MASTER SPAREPART, bukan dari badan permintaan.
//
// Itu yang membuat nomor sparepart menjadi satu-satunya sumber kelimanya; lihat
// usecase.resolvePart.
func TestCreateFillsDerivedFieldsFromPartMaster(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.Equal(t, "SEAL KIT BOOM", saved.PartName)
	require.Equal(t, "KAT02", saved.CategoryID)
	require.Equal(t, "TIP03", saved.TypeID)
	require.Equal(t, "KD-1002", saved.PartCode, "kode sparepart ikut disalin")
	require.Equal(t, "03/01/2025", saved.ProductionDate)
}

// Penambahan menerbitkan ID DAN nomor grup, dan barisnya lahir berstatus menunggu.
func TestCreateIssuesKeyAndGroupAndStartsPending(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	// Contoh berisi empat baris ber-ID "1".."4"; penambahan melanjutkan deret.
	require.Equal(t, "5", saved.ID)
	// Nomor grup tertinggi pada contoh adalah 3, sehingga yang baru adalah 4.
	require.Equal(t, mastergroupingsparepart.ComposeGroupNumber(4), saved.GroupNumber)
	require.Equal(t, mastergroupingsparepart.StatusPending, saved.Status)
}

// Nomor sparepart yang tidak ada di Master Sparepart ditolak, dan pesannya menempel pada
// isiannya — bukan sebagai 404 yang menutup form.
func TestCreateRejectsUnknownPartNumber(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.PartNumber = "SP-TIDAK-ADA"

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "nomor_sparepart")
	require.Contains(t, err.Error(), "Data Sparepart tidak ditemukan")
}

// Kunci alaminya EMPAT KOLOM BERSAMA-SAMA, bukan empat kolom yang masing-masing unik.
//
// Satu nomor sparepart boleh muncul berkali-kali — memang itu gunanya layar ini.
func TestCreateAllowsRepeatedPartNumberOnDifferentPanel(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	// Nomor sparepart yang sama dengan baris contoh "1", tetapi panel dan rangka berbeda.
	input.PartNumber = "SP-1001"

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
}

// Keempat kunci alami yang sama persis ditolak.
func TestCreateRejectsDuplicateNaturalKey(t *testing.T) {
	service, _ := newService(t)

	// Baris contoh "1": SP-1001 · KABIN · MHFXW1234K5678901 · sisi "1".
	input := mastergroupingsparepart.Input{
		PartNumber:    "SP-1001",
		PanelID:       "PNL02",
		PanelName:     "KABIN",
		PanelSide:     "1",
		ChassisNumber: "MHFXW1234K5678901",
	}

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, mastergroupingsparepart.ErrDuplicate)
}

// Pemeriksaan duplikat mengabaikan besar-kecil huruf dan spasi tepi.
//
// SELISIH YANG DIRENCANAKAN: kueri lama membandingkannya apa adanya, sehingga nomor sparepart
// bertuliskan huruf kecil lolos sebagai baris yang berbeda.
func TestCreateDuplicateCheckIgnoresCaseAndSpace(t *testing.T) {
	service, _ := newService(t)

	input := mastergroupingsparepart.Input{
		PartNumber:    " sp-1001 ",
		PanelID:       "PNL02",
		PanelName:     " kabin ",
		PanelSide:     " 1 ",
		ChassisNumber: " mhfxw1234k5678901 ",
	}

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, mastergroupingsparepart.ErrDuplicate)
}

// Mengisi "Grouping Dengan No Rangka" membuat baris IKUT grup yang sudah ada — dengan nomor
// grup yang SAMA PERSIS, bukan bentuk teks yang berbeda.
//
// Di sistem lama `GetNoGroup` mengembalikannya lewat TO_NUMBER, sehingga baris yang bergabung
// menyimpan "1" sementara baris yang membuka grup menyimpan "0001"; lihat
// usecase.Service.resolveGroupNumber.
func TestCreateJoiningGroupKeepsTheSameGroupNumber(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.GroupWithChassis = "MHFXW1234K5678901" // nomor rangka baris contoh "1" dan "2"

	saved, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.Equal(t, mastergroupingsparepart.ComposeGroupNumber(1), saved.GroupNumber)
}

// Nomor rangka grouping yang tidak pernah didaftarkan ditolak, dan pesannya menempel pada
// isiannya.
func TestCreateRejectsUnknownGroupChassis(t *testing.T) {
	service, _ := newService(t)

	input := validInput()
	input.GroupWithChassis = "MHFTIDAKADA00000"

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "grouping_dengan_no_rangka")
	require.Contains(t, err.Error(), "Nomor rangka dalam grouping tidak ditemukan")
}

// Menyimpan SELALU mengembalikan baris ke antrean persetujuan.
func TestSaveAlwaysReturnsRowToPending(t *testing.T) {
	service, repo := newService(t)

	before, err := repo.Get(context.Background(), "1")
	require.NoError(t, err)
	require.Equal(t, mastergroupingsparepart.StatusApproved, before.Status)

	input := mastergroupingsparepart.Input{
		PartNumber:    before.PartNumber,
		PanelID:       before.PanelID,
		PanelName:     before.PanelName,
		PanelSide:     before.PanelSide,
		ChassisNumber: before.ChassisNumber,
		VehicleType:   before.VehicleType,
		Note:          "Diubah.",
	}

	saved, err := service.Save(context.Background(), portalAlias, "1", input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
	require.Equal(t, mastergroupingsparepart.StatusPending, saved.Status)
	require.Equal(t, "Diubah.", saved.Note)
}

// Menyimpan tanpa mengubah kuncinya sendiri TIDAK ditolak sebagai duplikat.
func TestSaveAllowsUnchangedOwnKey(t *testing.T) {
	service, repo := newService(t)

	before, err := repo.Get(context.Background(), "1")
	require.NoError(t, err)

	_, err = service.Save(context.Background(), portalAlias, "1",
		mastergroupingsparepart.Input{
			PartNumber:    before.PartNumber,
			PanelID:       before.PanelID,
			PanelName:     before.PanelName,
			PanelSide:     before.PanelSide,
			ChassisNumber: before.ChassisNumber,
		}, usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
}

// Menyunting sebuah baris menjadi kunci milik baris LAIN ditolak.
//
// SELISIH YANG DIRENCANAKAN: sistem lama hanya memeriksa duplikat pada PENAMBAHAN
// (`TempSparepart.ID=="UnknownID"`), sehingga dua baris berkunci sama dapat lahir cukup dengan
// menyunting salah satunya.
func TestSaveRejectsKeyTakenByAnotherRow(t *testing.T) {
	service, repo := newService(t)

	other, err := repo.Get(context.Background(), "1")
	require.NoError(t, err)

	_, err = service.Save(context.Background(), portalAlias, "3",
		mastergroupingsparepart.Input{
			PartNumber:    other.PartNumber,
			PanelID:       other.PanelID,
			PanelName:     other.PanelName,
			PanelSide:     other.PanelSide,
			ChassisNumber: other.ChassisNumber,
		}, usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "nomor_sparepart")
	require.Contains(t, err.Error(), "Data sudah ada")
}

// Nomor grup baris yang sudah punya TIDAK diterbitkan ulang saat disimpan.
//
// Syarat langkah 6 pada `UpdateGroupingSparepartHE_act` menyebut KEDUANYA — isian grouping
// kosong DAN nomor grup kosong — sehingga baris lama tidak mendapat nomor baru hanya karena
// disimpan ulang.
func TestSaveKeepsExistingGroupNumber(t *testing.T) {
	service, repo := newService(t)

	before, err := repo.Get(context.Background(), "1")
	require.NoError(t, err)

	saved, err := service.Save(context.Background(), portalAlias, "1",
		mastergroupingsparepart.Input{
			PartNumber:    before.PartNumber,
			PanelID:       before.PanelID,
			PanelName:     before.PanelName,
			PanelSide:     before.PanelSide,
			ChassisNumber: before.ChassisNumber,
			// Sengaja kosong: baris ini membuka grupnya sendiri.
			GroupWithChassis: "",
		}, usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
	require.Equal(t, before.GroupNumber, saved.GroupNumber)
}

func TestSaveRejectsMissingRow(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Save(context.Background(), portalAlias, "999", validInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, mastergroupingsparepart.ErrNotFound)
}

func TestListFiltersByStatus(t *testing.T) {
	service, _ := newService(t)

	approved, err := service.List(context.Background(), portalAlias,
		mastergroupingsparepart.StatusApproved, "")
	require.NoError(t, err)
	require.Len(t, approved, 2)

	pending, err := service.List(context.Background(), portalAlias,
		mastergroupingsparepart.StatusPending, "")
	require.NoError(t, err)
	require.Len(t, pending, 1)
}

// Pencarian menelusuri KEEMPAT kolom, bukan hanya namanya.
func TestListSearchesAllFourColumns(t *testing.T) {
	service, _ := newService(t)

	for _, keyword := range []string{"SP-1001", "FILTER OLI", "KABIN", "MHFXW1234"} {
		found, err := service.List(context.Background(), portalAlias,
			mastergroupingsparepart.StatusApproved, keyword)
		require.NoErrorf(t, err, "kata kunci %q", keyword)
		require.NotEmptyf(t, found, "kata kunci %q harus menemukan baris", keyword)
	}
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), portalAlias, "9", "")
	require.ErrorIs(t, err, mastergroupingsparepart.ErrUnknownStatus)
}

func TestOptionsReturnsBothLookups(t *testing.T) {
	service, _ := newService(t)

	set, err := service.Options(context.Background(), portalAlias)
	require.NoError(t, err)
	require.Len(t, set.Panel, 3)
	require.Len(t, set.VehicleType, 3)
}

// Daftar Sisi berbeda antarpanel — itulah yang dibuktikan penyaringnya bekerja.
func TestSidesDifferPerPanel(t *testing.T) {
	service, _ := newService(t)

	cabin, err := service.Sides(context.Background(), portalAlias,
		mastergroupingsparepart.SideKey{PanelID: "PNL02", PanelName: "KABIN"})
	require.NoError(t, err)
	require.Len(t, cabin, 3)

	bucket, err := service.Sides(context.Background(), portalAlias,
		mastergroupingsparepart.SideKey{PanelID: "PNL03", PanelName: "BUCKET"})
	require.NoError(t, err)
	require.Len(t, bucket, 1)
}

// Panel yang tidak dikenal mengembalikan daftar KOSONG, bukan galat: itu keadaan data, bukan
// kegagalan permintaan.
func TestSidesOfUnknownPanelIsEmptyNotError(t *testing.T) {
	service, _ := newService(t)

	side, err := service.Sides(context.Background(), portalAlias,
		mastergroupingsparepart.SideKey{PanelID: "PNL99", PanelName: "TIDAK ADA"})
	require.NoError(t, err)
	require.Empty(t, side)
}

// Permintaan tanpa panel sama sekali DITOLAK — kueri aslinya akan mencocoki seluruh baris
// lokasi di basis data.
func TestSidesWithoutPanelIsRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Sides(context.Background(), portalAlias, mastergroupingsparepart.SideKey{})
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "id_panel")
}

func TestLookupPartReturnsDerivedFields(t *testing.T) {
	service, _ := newService(t)

	part, err := service.LookupPart(context.Background(), portalAlias, "sp-1001")
	require.NoError(t, err)
	require.Equal(t, "FILTER OLI", part.Name)
	require.Equal(t, "KAT01", part.CategoryID)
}

func TestLookupPartRejectsUnknownNumber(t *testing.T) {
	service, _ := newService(t)

	_, err := service.LookupPart(context.Background(), portalAlias, "SP-TIDAK-ADA")
	require.ErrorIs(t, err, mastergroupingsparepart.ErrPartNotFound)
}

func TestDecideChangesSelectedRows(t *testing.T) {
	service, repo := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias, []string{"3"},
		mastergroupingsparepart.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)

	after, err := repo.Get(context.Background(), "3")
	require.NoError(t, err)
	require.Equal(t, mastergroupingsparepart.StatusApproved, after.Status)
}

// Kunci ganda dibuang: jumlah yang dilaporkan harus mencerminkan BARIS, bukan centang.
func TestDecideIgnoresDuplicateKeys(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"3", "3", " 3 ", ""},
		mastergroupingsparepart.StatusRejected, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

func TestDecideRejectsEmptySelection(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias, nil,
		mastergroupingsparepart.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.Error(t, err)
	require.Contains(t, violationFields(t, err), "id_grouping")
}

func TestDecideRejectsUnknownStatus(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias, []string{"3"}, "9",
		usecase.Actor{Login: "manajer"}, nil)
	require.ErrorIs(t, err, mastergroupingsparepart.ErrUnknownStatus)
}

// Keputusan TIDAK menyentuh satu pun kolom lain.
//
// Sistem lama menyimpan ulang seluruh barisnya saat menyetujui; lihat usecase.Service.Decide.
func TestDecideTouchesOnlyStatus(t *testing.T) {
	service, repo := newService(t)

	before, err := repo.Get(context.Background(), "3")
	require.NoError(t, err)

	_, err = service.Decide(context.Background(), portalAlias, []string{"3"},
		mastergroupingsparepart.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)

	after, err := repo.Get(context.Background(), "3")
	require.NoError(t, err)

	before.Status = mastergroupingsparepart.StatusApproved
	require.Equal(t, before, after)
}

// Portal yang tidak dikenal DITOLAK pada setiap jalur, bukan dijawab dengan data portal utama.
//
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan hukum
// ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestUnknownPortalIsRejectedOnEveryPath(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "lain", mastergroupingsparepart.StatusApproved, "")
	require.Error(t, err)

	_, err = service.Get(ctx, "lain", "1")
	require.Error(t, err)

	_, err = service.Options(ctx, "lain")
	require.Error(t, err)

	_, err = service.Sides(ctx, "lain", mastergroupingsparepart.SideKey{PanelID: "PNL01"})
	require.Error(t, err)

	_, err = service.LookupPart(ctx, "lain", "SP-1001")
	require.Error(t, err)

	_, err = service.Create(ctx, "lain", validInput(), usecase.Actor{Login: "x"}, nil)
	require.Error(t, err)

	_, err = service.Save(ctx, "lain", "1", validInput(), usecase.Actor{Login: "x"}, nil)
	require.Error(t, err)

	_, err = service.Decide(ctx, "lain", []string{"1"},
		mastergroupingsparepart.StatusApproved, usecase.Actor{Login: "x"}, nil)
	require.Error(t, err)

	require.Error(t, service.EnsurePortalReady("lain"))
	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

// violationFields mengambil nama isian yang dilaporkan sebuah galat validasi.
func violationFields(t *testing.T, err error) []string {
	t.Helper()

	var violation *mastergroupingsparepart.ValidationError
	require.ErrorAs(t, err, &violation)

	result := make([]string, 0, len(violation.Violation))
	for _, one := range violation.Violation {
		result = append(result, one.Field)
	}
	return result
}
