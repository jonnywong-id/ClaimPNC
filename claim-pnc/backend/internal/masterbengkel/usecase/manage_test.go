package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	"claim-pnc/internal/masterbengkel/usecase"
)

const portalAlias = "ASM"

func newService(t *testing.T) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterbengkel.Store, error) {
			if alias != portalAlias {
				return nil, errors.New("portal tidak tersedia")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service, repo
}

func newInput() masterbengkel.Input {
	return masterbengkel.Input{
		Name:           "Bengkel Contoh Baru",
		Address:        "Jalan Contoh Nomor 9",
		PartnerStatus:  "1",
		WorkshopStatus: "1",
		Login:          "bengkelbaru",
		BranchID:       "001",
		BranchName:     "Cabang Contoh Pusat",
		CityID:         "3171",
		CityName:       "Jakarta Pusat",
		BankID:         "002",
		BankName:       "Bank Contoh Satu",
		AccountNumber:  "9000000009",
		ValueAddedTax:  "11",
	}
}

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Baris baru SELALU lahir berstatus menunggu, dan kuncinya diterbitkan server.
//
// Keduanya tidak pernah datang dari layar: `UpdateBengkelHE_act` step 7 menetapkan
// APPROVAL, dan `PEGA_M_BENGKEL_HE.prc:19` menerbitkan ID.
func TestCreateIssuesKeyAndStartsPending(t *testing.T) {
	service, _ := newService(t)

	saved, err := service.Create(context.Background(), portalAlias, newInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.Equal(t, masterbengkel.StatusPending, saved.Status)
	require.NotEmpty(t, saved.ID)
	require.Equal(t, "Bengkel Contoh Baru", saved.Name)

	// Kuncinya berbentuk kode situs + sepuluh digit, sama seperti yang diterbitkan Pega.
	require.Len(t, saved.ID, len(memory.SampleSite)+10)
}

// Dua penambahan berturut-turut tidak pernah memakai kunci yang sama.
func TestCreateIssuesDistinctKeys(t *testing.T) {
	service, _ := newService(t)

	first, err := service.Create(context.Background(), portalAlias, newInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	second := newInput()
	second.Name = "Bengkel Contoh Kedua"
	second.Login = "bengkelkedua"
	saved, err := service.Create(context.Background(), portalAlias, second,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.NotEqual(t, first.ID, saved.ID)
}

// Nama bengkel yang sudah dipakai ditolak.
//
// Padanan `Activity/ValidateMasterBengkel`. Pencocokannya mengabaikan besar-kecil huruf
// dan spasi tepi, meniru `upper(trim(nama_bengkel))` pada kuerinya.
func TestCreateRejectsDuplicateName(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Name = "  bengkel contoh utama  " // sudah ada pada SampleList
	input.Login = "loginlain"

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrNameTaken)
}

// Login aplikasi yang sudah dipakai ditolak.
//
// Padanan `Activity/ValidationLoginBengkel_act` step 7, dengan satu perbedaan yang tidak
// terhindarkan: yang dicari adalah kolom LOGIN_APLIKASI pada tabel ini, bukan daftar
// operator Pega yang tidak ada padanannya di sistem baru.
func TestCreateRejectsDuplicateLogin(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Login = "BENGKELCONTOH1" // sudah ada pada SampleList, beda besar-kecil huruf

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrLoginTaken)
}

// Dua bengkel NON-REKANAN tanpa login tidak saling menolak.
//
// Tanpa penyaring "login kosong tidak pernah cocok", bengkel non-rekanan kedua akan
// ditolak karena "login kosong sudah dipakai" — dan pesannya tidak akan masuk akal bagi
// siapa pun yang membacanya.
func TestTwoNonPartnersWithoutLoginCoexist(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Name = "Bengkel Non Rekanan Kedua"
	input.PartnerStatus = masterbengkel.PartnerStatusNonPartner
	input.Login = ""

	_, err := service.Create(context.Background(), portalAlias, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
}

// Isian yang tidak sah ditolak SEBELUM kunci diterbitkan.
//
// Kalau tidak, setiap percobaan yang gagal akan membakar satu nomor urut — dan deret
// ID_BENGKEL berlubang tanpa ada baris yang menjelaskannya.
func TestCreateChecksInputBeforeIssuingKey(t *testing.T) {
	service, repo := newService(t)

	before, err := repo.NextID(context.Background())
	require.NoError(t, err)

	_, err = service.Create(context.Background(), portalAlias, masterbengkel.Input{},
		usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)

	after, err := repo.NextID(context.Background())
	require.NoError(t, err)

	// Dua pemanggilan NextID di uji ini sendiri menaikkan urutan dua kali; yang gagal di
	// antaranya TIDAK boleh menaikkannya lagi.
	require.NotEqual(t, before, after)
	require.Equal(t, masterbengkel.ComposeID(memory.SampleSite, sequenceOf(after), 10), after)
}

// Menyimpan perubahan SELALU mengembalikan baris ke antrean persetujuan.
//
// Ia bukan tafsiran melainkan langkah tersendiri di sistem lama:
// `Activity/UpdateBengkelHE_act` step 7 menetapkan APPROVAL := "0" tanpa syarat apa pun.
func TestSaveReturnsRowToPending(t *testing.T) {
	service, _ := newService(t)

	const approvedID = "010000000001" // berstatus Approve pada SampleList

	input := newInput()
	input.Name = "Bengkel Contoh Utama" // namanya sendiri, tidak berubah
	input.Login = "bengkelcontoh1"      // login-nya sendiri
	input.Address = "Alamat baru"

	saved, err := service.Save(context.Background(), portalAlias, approvedID, input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)

	require.Equal(t, masterbengkel.StatusPending, saved.Status)
	require.Equal(t, "Alamat baru", saved.Address)
	require.Equal(t, approvedID, saved.ID, "kunci baris tidak boleh berpindah")
}

// Menyimpan tanpa mengubah nama TIDAK ditolak karena namanya sendiri.
func TestSaveAllowsKeepingOwnNameAndLogin(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Name = "Bengkel Contoh Utama"
	input.Login = "bengkelcontoh1"

	_, err := service.Save(context.Background(), portalAlias, "010000000001", input,
		usecase.Actor{Login: "petugas"}, nil)
	require.NoError(t, err)
}

// Menyimpan dengan nama milik BARIS LAIN ditolak.
//
// Tanpa pemeriksaan ini, dua bengkel bernama sama dapat lahir — cukup dengan menyunting
// salah satunya.
func TestSaveRejectsNameOwnedByAnotherRow(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Name = "Bengkel Contoh Menunggu" // milik baris 010000000002
	input.Login = "bengkelcontoh1"

	_, err := service.Save(context.Background(), portalAlias, "010000000001", input,
		usecase.Actor{Login: "petugas"}, nil)

	var validationError *masterbengkel.ValidationError
	require.True(t, errors.As(err, &validationError))
	require.Equal(t, "nama_bengkel", validationError.Violation[0].Field)
}

// Menyimpan dengan login milik BARIS LAIN ditolak.
func TestSaveRejectsLoginOwnedByAnotherRow(t *testing.T) {
	service, _ := newService(t)

	input := newInput()
	input.Name = "Bengkel Contoh Utama"
	input.Login = "bengkelcontoh2" // milik baris 010000000002

	_, err := service.Save(context.Background(), portalAlias, "010000000001", input,
		usecase.Actor{Login: "petugas"}, nil)

	var validationError *masterbengkel.ValidationError
	require.True(t, errors.As(err, &validationError))
	require.Equal(t, "login_aplikasi", validationError.Violation[0].Field)
}

// Baris yang sudah tidak ada menghasilkan ErrNotFound, bukan diam-diam tersimpan.
func TestSaveMissingRow(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Save(context.Background(), portalAlias, "tidak-ada", newInput(),
		usecase.Actor{Login: "petugas"}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrNotFound)
}

// Keputusan borongan menetapkan status seluruh baris yang dipilih.
//
// Padanan `Activity/SetApprovalAllMaster`, yang menelusuri baris bercentang lalu
// menetapkan status yang sama pada masing-masing.
func TestDecideSetsStatusOnEveryChosenRow(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"010000000002", "010000000003"},
		masterbengkel.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)
	require.Equal(t, 2, changed)

	list, err := service.List(context.Background(), portalAlias, masterbengkel.StatusApproved, "")
	require.NoError(t, err)
	require.Len(t, list, 3) // satu yang sudah disetujui ditambah dua yang baru
}

// Baris yang SUDAH berstatus itu tidak dihitung sebagai berubah.
//
// Yang dilaporkan adalah baris yang benar-benar berubah, supaya layar dapat mengatakan
// "1 dari 2" alih-alih melaporkan keberhasilan atas baris yang tidak tersentuh.
func TestDecideCountsOnlyRowsThatChanged(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"010000000001", "010000000002"}, // yang pertama sudah Approve
		masterbengkel.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

// Kunci ganda dibuang sebelum diproses.
//
// Layar dapat mengirim baris yang sama dua kali bila daftarnya dimuat ulang saat centang
// masih terpasang, dan jumlah baris berubah yang dilaporkan harus mencerminkan BARIS,
// bukan centang.
func TestDecideIgnoresDuplicateKeys(t *testing.T) {
	service, _ := newService(t)

	changed, err := service.Decide(context.Background(), portalAlias,
		[]string{"010000000002", "010000000002", " 010000000002 "},
		masterbengkel.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.NoError(t, err)
	require.Equal(t, 1, changed)
}

// Keputusan tanpa satu pun baris ditolak sebagai pelanggaran isian, bukan 500.
func TestDecideWithoutRowsRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.Decide(context.Background(), portalAlias, nil,
		masterbengkel.StatusApproved, usecase.Actor{Login: "manajer"}, nil)

	var validationError *masterbengkel.ValidationError
	require.True(t, errors.As(err, &validationError))
	require.Equal(t, "id_bengkel", validationError.Violation[0].Field)
}

// Status yang tidak dikenal ditolak pada SETIAP jalur yang menerimanya.
func TestUnknownStatusRejected(t *testing.T) {
	service, _ := newService(t)

	_, err := service.List(context.Background(), portalAlias, "3", "")
	require.ErrorIs(t, err, masterbengkel.ErrUnknownStatus)

	_, err = service.Decide(context.Background(), portalAlias, []string{"010000000001"},
		"3", usecase.Actor{Login: "manajer"}, nil)
	require.ErrorIs(t, err, masterbengkel.ErrUnknownStatus)
}

// Penyaring kata kunci mempersempit pada nama bengkel, kota, dan cabang.
func TestListKeywordFilter(t *testing.T) {
	service, _ := newService(t)

	for _, keyword := range []string{"utama", "JAKARTA", "cabang contoh pusat"} {
		list, err := service.List(context.Background(), portalAlias, masterbengkel.StatusApproved, keyword)
		require.NoError(t, err)
		require.Len(t, list, 1, "kata kunci %q", keyword)
		require.Equal(t, "Bengkel Contoh Utama", list[0].Name)
	}

	list, err := service.List(context.Background(), portalAlias, masterbengkel.StatusApproved, "tidak ada")
	require.NoError(t, err)
	require.Empty(t, list)
}

// Portal yang tidak dikenal DITOLAK, tidak pernah dialihkan ke portal utama.
//
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
func TestUnknownPortalRejectedOnEveryPath(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "LAIN", masterbengkel.StatusApproved, "")
	require.Error(t, err)

	_, err = service.Get(ctx, "LAIN", "010000000001")
	require.Error(t, err)

	_, err = service.Create(ctx, "LAIN", newInput(), usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)

	_, err = service.Save(ctx, "LAIN", "010000000001", newInput(), usecase.Actor{Login: "petugas"}, nil)
	require.Error(t, err)

	_, err = service.Decide(ctx, "LAIN", []string{"010000000001"},
		masterbengkel.StatusApproved, usecase.Actor{Login: "manajer"}, nil)
	require.Error(t, err)

	_, err = service.ListBranches(ctx, "LAIN")
	require.Error(t, err)

	_, err = service.SearchCities(ctx, "LAIN", "jakarta")
	require.Error(t, err)

	_, err = service.ListBanks(ctx, "LAIN")
	require.Error(t, err)

	require.Error(t, service.EnsurePortalReady("LAIN"))
	require.NoError(t, service.EnsurePortalReady(portalAlias))
}

// Kata kunci lookup yang terlalu pendek dijawab daftar kosong, BUKAN galat.
//
// Pengguna yang baru mengetik satu huruf belum melakukan kesalahan apa pun, dan pesan
// galat di bawah kotak pencarian yang sedang diketik hanya akan mengganggu.
func TestShortLookupKeywordReturnsEmpty(t *testing.T) {
	service, _ := newService(t)

	city, err := service.SearchCities(context.Background(), portalAlias, "j")
	require.NoError(t, err)
	require.Empty(t, city)

	city, err = service.SearchCities(context.Background(), portalAlias, "ja")
	require.NoError(t, err)
	require.NotEmpty(t, city)
}

// Ketiga lookup terisi pada repo contoh, sehingga form dapat dicoba tanpa Oracle.
func TestLookupsAvailable(t *testing.T) {
	service, _ := newService(t)
	ctx := context.Background()

	branch, err := service.ListBranches(ctx, portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, branch)

	bank, err := service.ListBanks(ctx, portalAlias)
	require.NoError(t, err)
	require.NotEmpty(t, bank)
}

// sequenceOf membaca nomor urut dari sebuah ID_BENGKEL contoh.
func sequenceOf(id string) int64 {
	var number int64
	for _, digit := range id[len(memory.SampleSite):] {
		number = number*10 + int64(digit-'0')
	}
	return number
}
