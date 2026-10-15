package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
	"claim-pnc/internal/mastersupplier/repo/memory"
	"claim-pnc/internal/mastersupplier/usecase"
	"claim-pnc/internal/platform/clock"
)

const portalAlias = "asm"

// fixedNow adalah waktu yang dipegang jam uji.
//
// 03:00 UTC = 10:00 WIB, jauh dari tengah malam di kedua zona — sehingga uji yang
// memeriksa TGL_INSERT tidak lulus atau gagal karena kebetulan tanggalnya bergeser.
var fixedNow = time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)

// build merakit layanan di atas repo memori berisi contoh.
func build(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (mastersupplier.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak dikenal")
			}
			return repo, nil
		},
		Clock: clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)
	return service, repo
}

func validInput() mastersupplier.Input {
	return mastersupplier.Input{
		Name:            "Supplier Baru",
		Address:         "Jalan Uji Nomor 9",
		City:            "Jakarta Pusat",
		BranchName:      "Cabang Contoh Pusat",
		Country:         "Indonesia",
		Phone:           "021-0000009",
		ContactPerson:   "Narahubung Uji",
		PartnerStatus:   "1",
		SupplyType:      mastersupplier.SupplyTypeHeavyEquipment,
		TermOfPayment:   "30",
		TermOfDelivery:  "7",
		Bank:            "Bank Contoh Satu",
		AccountNumber:   "9000000009",
		SupplierType:    "1",
		ActiveRequested: mastersupplier.ActiveYes,
	}
}

func actor() usecase.Actor { return usecase.Actor{Login: "PETUGAS.UJI"} }

func TestNewServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(fixedNow)})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersupplier.Store, error) { return nil, nil },
	})
	require.Error(t, err, "Clock wajib: modul ini menulis dua jejak waktu")
}

// # Supplier baru SELALU lahir tidak aktif
//
// `Activity/CreateNewMasterSupplier_post` step 6 menetapkan `STS_AKTIF := "0"` tanpa
// syarat apa pun — berapa pun yang dipilih di layar. Itu yang membuat supplier baru tidak
// dapat dipakai sebelum permintaannya diputuskan.
func TestCreateAlwaysStartsInactive(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.ActiveRequested = mastersupplier.ActiveYes

	saved, err := service.Create(context.Background(), portalAlias, input, actor(), nil)
	require.NoError(t, err)

	require.Equal(t, mastersupplier.ActiveNo, saved.Active,
		"yang BERLAKU harus tidak aktif")
	require.Equal(t, mastersupplier.ActiveYes, saved.ActiveRequested,
		"yang DIMINTA tetap seperti yang dipilih petugas")
}

// ID diterbitkan server, bukan diketik — dan bentuknya mengikuti procedure lama.
func TestCreateIssuesIdentifier(t *testing.T) {
	service, _ := build(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.NoError(t, err)

	require.Equal(t, "0100000000004", saved.ID,
		"kode situs contoh ditambah nomor urut sebelas digit, melanjutkan SampleSequence")
}

// SUPPLIER_HE diturunkan dari JENIS_STATUS pada setiap penyimpanan, bukan dikirim layar.
func TestCreateDerivesHeavyEquipment(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.SupplyType = mastersupplier.SupplyTypeHeavyEquipment

	saved, err := service.Create(context.Background(), portalAlias, input, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, mastersupplier.SupplyTypeHeavyEquipment, saved.HeavyEquipment)
}

// USERKLAIMID dan TGL_INSERT ditulis, dan tanggalnya WIB.
func TestCreateWritesAuditTrail(t *testing.T) {
	service, _ := build(t)

	saved, err := service.Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.NoError(t, err)

	require.Equal(t, "PETUGAS.UJI", saved.UpdatedBy)
	require.Equal(t, "20/09/2026", saved.UpdatedAt)
}

// Penambahan SELALU meminta persetujuan, tanpa syarat.
//
// Prasyarat Pega di `CreateNewMasterSupplier_post` step 12 hanyalah "penyimpanannya
// berhasil" — bukan syarat atas status aktif seperti pada jalur penyimpanan.
func TestCreateAlwaysRequestsApproval(t *testing.T) {
	service, repo := build(t)

	input := validInput()
	input.ActiveRequested = mastersupplier.ActiveNo

	saved, err := service.Create(context.Background(), portalAlias, input, actor(), nil)
	require.NoError(t, err)

	queue := repo.Approval()
	require.Len(t, queue, 1)
	require.Equal(t, saved.ID, queue[0].SupplierID)
	require.Equal(t, mastersupplier.PositionRequested, queue[0].Position)
	require.Equal(t, "PETUGAS.UJI", queue[0].RequestedBy)
	require.Equal(t, fixedNow, queue[0].RequestedAt)
}

// Baris permintaan membawa SALINAN isi supplier, supaya yang memutuskan dapat melihat apa
// yang disetujuinya tanpa membaca tabel master.
func TestApprovalCarriesSnapshot(t *testing.T) {
	service, repo := build(t)

	input := validInput()
	input.Note = "Keterangan uji"

	saved, err := service.Create(context.Background(), portalAlias, input, actor(), nil)
	require.NoError(t, err)

	queue := repo.Approval()
	require.Len(t, queue, 1)
	require.Equal(t, saved.Name, queue[0].Snapshot.Name)
	require.Equal(t, "Keterangan uji", queue[0].Reason,
		"ALASAN_REQ diisi KETERANGAN supplier")
}

func TestCreateRejectsInvalidInput(t *testing.T) {
	service, repo := build(t)

	_, err := service.Create(context.Background(), portalAlias, mastersupplier.Input{}, actor(), nil)

	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Empty(t, repo.Approval(), "isian yang ditolak tidak boleh meninggalkan permintaan")
}

func TestCreateRejectsDuplicateName(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.Name = "supplier contoh utama" // sama dengan baris contoh, beda huruf besar-kecil

	_, err := service.Create(context.Background(), portalAlias, input, actor(), nil)
	require.ErrorIs(t, err, mastersupplier.ErrNameTaken)
}

// # Nama TIDAK dapat diubah setelah tersimpan
//
// `Section/CreateMasterSupplier_Sec-Section.xml` memasang syarat read-only pada isian
// NAMA:
//
//	pyReadOnlyCondition: MasterSupplier.ID != ''
//
// Layar Pega menegakkannya dengan mengunci isiannya; server ikut memeriksanya karena
// permintaan yang tidak datang dari layar tidak tersentuh penguncian itu.
func TestSaveRejectsRenamedSupplier(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.Name = "Nama Yang Diganti"

	_, err := service.Save(context.Background(), portalAlias, "0100000000001", input, actor(), nil)
	require.ErrorIs(t, err, mastersupplier.ErrNameLocked)
}

// Nama yang dikirim kembali dengan huruf atau spasi berbeda TETAP diterima: yang dilarang
// adalah menggantinya, bukan mengirimkannya ulang dalam bentuk yang sedikit berbeda.
func TestSaveAcceptsSameNameDifferentCasing(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.Name = "  supplier contoh utama  "

	saved, err := service.Save(context.Background(), portalAlias, "0100000000001", input, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, "Supplier Contoh Utama", saved.Name,
		"yang tersimpan tetap ejaan aslinya, bukan yang dikirim")
}

// Pada penyimpanan, STS_AKTIF menyusul pilihan di layar — berbeda dari penambahan yang
// selalu "0" (`EditMasterSupplier_post` step 7).
func TestSaveAppliesRequestedActiveStatus(t *testing.T) {
	service, _ := build(t)

	input := validInput()
	input.Name = "Supplier Contoh Nonaktif"
	input.ActiveRequested = mastersupplier.ActiveYes

	saved, err := service.Save(context.Background(), portalAlias, "0100000000003", input, actor(), nil)
	require.NoError(t, err)
	require.Equal(t, mastersupplier.ActiveYes, saved.Active)
}

// # Menonaktifkan supplier TIDAK meminta persetujuan siapa pun
//
// `EditMasterSupplier_post` step 12 memasang prasyarat
// `STS_AKTIF_PROMLIST == "1" || STS_AKTIF_PROMLIST == ""`. Menutup kerja sama karena itu
// berlaku seketika, sedangkan membukanya harus menunggu.
//
// Ia satu-satunya jalur di modul ini yang mengubah keadaan tanpa melewati antrean mana
// pun, dan uji ini yang membuatnya tetap terlihat.
func TestSaveSkipsApprovalWhenDeactivating(t *testing.T) {
	service, repo := build(t)

	input := validInput()
	input.Name = "Supplier Contoh Utama"
	input.ActiveRequested = mastersupplier.ActiveNo

	_, err := service.Save(context.Background(), portalAlias, "0100000000001", input, actor(), nil)
	require.NoError(t, err)
	require.Empty(t, repo.Approval(),
		"menonaktifkan supplier berlaku seketika, tanpa persetujuan")
}

func TestSaveRequestsApprovalWhenActivating(t *testing.T) {
	service, repo := build(t)

	input := validInput()
	input.Name = "Supplier Contoh Nonaktif"
	input.ActiveRequested = mastersupplier.ActiveYes

	_, err := service.Save(context.Background(), portalAlias, "0100000000003", input, actor(), nil)
	require.NoError(t, err)
	require.Len(t, repo.Approval(), 1)
}

// Nilai kosong ikut diterima sebagai "perlu persetujuan", persis seperti prasyarat Pega —
// meski form mewajibkan isian itu, permintaan yang tidak datang dari form dapat
// mengirimkannya kosong.
func TestSaveTreatsEmptyActiveAsNeedingApproval(t *testing.T) {
	service, repo := build(t)

	input := validInput()
	input.Name = "Supplier Contoh Utama"
	input.ActiveRequested = ""

	// Isian wajib yang kosong tetap ditolak lebih dulu — itu urutan yang benar.
	_, err := service.Save(context.Background(), portalAlias, "0100000000001", input, actor(), nil)
	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Empty(t, repo.Approval())
}

func TestSaveRejectsUnknownIdentifier(t *testing.T) {
	service, _ := build(t)

	_, err := service.Save(context.Background(), portalAlias, "tidak-ada", validInput(), actor(), nil)
	require.ErrorIs(t, err, mastersupplier.ErrNotFound)
}

func TestGetDerivesSupplyTypeFromStoredFlag(t *testing.T) {
	service, _ := build(t)

	found, err := service.Get(context.Background(), portalAlias, "0100000000001")
	require.NoError(t, err)
	require.Equal(t, mastersupplier.SupplyTypeHeavyEquipment, found.SupplyType)
	require.Equal(t, mastersupplier.SupplyTypeHeavyEquipment, found.HeavyEquipment)
}

func TestListFiltersByKeyword(t *testing.T) {
	service, _ := build(t)

	list, err := service.List(context.Background(), portalAlias, "bandung")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "Supplier Contoh Aneka", list[0].Name)
}

func TestListSortsByName(t *testing.T) {
	service, _ := build(t)

	list, err := service.List(context.Background(), portalAlias, "")
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, "Supplier Contoh Aneka", list[0].Name)
}

// Portal yang tidak dikenal WAJIB menghasilkan galat, bukan jatuh ke portal utama.
//
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
func TestUnknownPortalIsRejected(t *testing.T) {
	service, _ := build(t)

	_, err := service.List(context.Background(), "entah", "")
	require.Error(t, err)

	_, err = service.Create(context.Background(), "entah", validInput(), actor(), nil)
	require.Error(t, err)

	require.Error(t, service.EnsurePortalReady("entah"))
	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

// Daftar sandi MENGGABUNGKAN nilai yang artinya terbukti dengan nilai yang ada di data.
//
// Tanpa penggabungan itu, basis data yang masih kosong akan menyajikan dropdown tanpa satu
// pun pilihan — dan supplier pertama tidak akan pernah dapat ditambahkan.
func TestListCodesMergesKnownAndStored(t *testing.T) {
	service, _ := build(t)

	set, err := service.ListCodes(context.Background(), portalAlias)
	require.NoError(t, err)

	require.Equal(t, []mastersupplier.CodeOption{
		{Value: "1", Label: "Heavy Equipment"},
		{Value: "0", Label: "Selain Heavy Equipment"},
	}, set.SupplyType, "nilai yang artinya terbukti muncul lebih dulu, dan tidak digandakan")

	// STS_REKANAN tidak punya arti yang terbukti; seluruh pilihannya datang dari data.
	require.Len(t, set.PartnerStatus, 2)

	// JENIS_SUPPLIER contoh memuat "1" dan "2".
	require.Len(t, set.SupplierType, 2)
}

func TestListCodesOnEmptyStoreStillOffersProvenValues(t *testing.T) {
	repo := memory.NewRepo(memory.Options{})
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (mastersupplier.Store, error) { return repo, nil },
		Clock:        clock.FixedAt(fixedNow),
	})
	require.NoError(t, err)

	set, err := service.ListCodes(context.Background(), portalAlias)
	require.NoError(t, err)

	require.Len(t, set.SupplyType, 2)
	require.Len(t, set.Active, 2)
	require.Empty(t, set.PartnerStatus,
		"tanpa data dan tanpa bukti, dropdown ini memang tidak punya pilihan — dan itu jujur")
}

// Kata kunci lookup Kota yang terlalu pendek dijawab daftar kosong, BUKAN galat.
func TestSearchCitiesIgnoresShortKeyword(t *testing.T) {
	service, _ := build(t)

	list, err := service.SearchCities(context.Background(), portalAlias, "j")
	require.NoError(t, err)
	require.Empty(t, list)

	list, err = service.SearchCities(context.Background(), portalAlias, "ja")
	require.NoError(t, err)
	require.NotEmpty(t, list)
}

func TestLookupsAreServed(t *testing.T) {
	service, _ := build(t)
	ctx := context.Background()

	branch, err := service.ListBranches(ctx, portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, branch)

	country, err := service.ListCountries(ctx, portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, country)

	bank, err := service.ListBanks(ctx, portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, bank)
}

// Kegagalan penyimpanan tidak boleh menyisakan permintaan persetujuan.
func TestStorageFailureLeavesNoApproval(t *testing.T) {
	service, repo := build(t)
	repo.SetError(errors.New("basis data sedang tidak dapat dihubungi"))

	_, err := service.Create(context.Background(), portalAlias, validInput(), actor(), nil)
	require.Error(t, err)
	require.Empty(t, repo.Approval())
}
