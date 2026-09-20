package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastermasking"
	"claim-pnc/internal/mastermasking/repo/memory"
	"claim-pnc/internal/mastermasking/usecase"
	"claim-pnc/internal/portal"
)

const alias = "ASM"

// fixedNow membuat waktu penyimpanan dapat dinyatakan, bukan ditebak.
var fixedNow = time.Date(2026, 9, 20, 4, 0, 0, 0, time.UTC)

func newService(t *testing.T, repo mastermasking.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(a string) (mastermasking.Repo, error) {
			if a == alias {
				return repo, nil
			}
			return nil, portal.ErrNotReady
		},
		Now: func() time.Time { return fixedNow },
	})
	require.NoError(t, err)
	return service
}

func newRepo() *memory.Repo {
	return memory.NewRepo(memory.SampleList()...).WithBranches(memory.SampleBranches()...)
}

func sample() mastermasking.Masking {
	return mastermasking.Masking{
		BranchID:    "100001", // AGENCY MANADO — cabang yang belum punya baris
		Login:       "BARU.SEKALI",
		Module:      "PNCSearchKlaim",
		SubModule:   "Registrasi,",
		SearchQuota: 5,
		ViewQuota:   5,
		ViewIDCard:  true,
	}
}

// Bahan yang tidak lengkap ditolak saat perakitan, bukan saat permintaan pertama datang.
func TestNewServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Portal yang belum siap ditolak, TIDAK pernah dialihkan ke portal utama sebagai cadangan.
//
// Inilah `R-20`: mengalihkannya berarti memberi seseorang kewenangan melihat data pribadi
// di badan hukum yang bukan haknya, tanpa satu pun pesan galat.
func TestUnknownPortalRejected(t *testing.T) {
	service := newService(t, newRepo())

	_, err := service.List(context.Background(), "SMAS", mastermasking.Filter{})
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(context.Background(), "SMAS", sample(), "ADMIN")
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Daftar menampilkan baris aktif MAUPUN nonaktif secara baku.
//
// Itu perilaku layar lama: `SearchData.Type == "1"` tidak menyusun penyaring apa pun.
func TestListShowsInactiveByDefault(t *testing.T) {
	service := newService(t, newRepo())

	all, err := service.List(context.Background(), alias, mastermasking.Filter{})
	require.NoError(t, err)
	require.Len(t, all, 5)
}

// Pencarian menurut status menyaring PERSIS, bukan sebagian.
//
// Ia tipe pencarian keempat di layar lama (`SearchData.Type == "4"`), yang menyusun
// `STS_AKTF = '<nilai>'`. Kedua arahnya harus dapat dipilih — bukan hanya "yang aktif".
func TestSearchByStatus(t *testing.T) {
	service := newService(t, newRepo())

	active, err := service.List(context.Background(), alias, mastermasking.Filter{
		By: mastermasking.SearchByStatus, Keyword: mastermasking.StatusActive,
	})
	require.NoError(t, err)
	require.Len(t, active, 4)

	inactive, err := service.List(context.Background(), alias, mastermasking.Filter{
		By: mastermasking.SearchByStatus, Keyword: mastermasking.StatusInactive,
	})
	require.NoError(t, err)
	require.Len(t, inactive, 1, "satu baris contoh memang nonaktif")
}

// Pencarian status tanpa menyebut statusnya DITOLAK.
//
// Layar lama menyiapkan pesan "Pilih Status Aktif" untuk keadaan yang sama. Mengabaikannya
// akan menampilkan seluruh baris kepada pengguna yang mengira ia sedang menyaring.
func TestSearchByStatusWithoutValueRejected(t *testing.T) {
	service := newService(t, newRepo())

	_, err := service.List(context.Background(), alias, mastermasking.Filter{
		By: mastermasking.SearchByStatus,
	})
	require.ErrorIs(t, err, mastermasking.ErrStatusNotChosen)

	_, err = service.List(context.Background(), alias, mastermasking.Filter{
		By: mastermasking.SearchByStatus, Keyword: "ENTAH",
	})
	require.ErrorIs(t, err, mastermasking.ErrStatusNotChosen)
}

// Kata kunci tanpa tipe pencarian diarahkan ke login, bukan diabaikan.
//
// Mengabaikannya akan menampilkan seluruh baris kepada pengguna yang sudah mengetik sesuatu
// — layar tampak tidak menyaring apa pun tanpa ada yang salah terlihat.
func TestKeywordWithoutTypeSearchesLogin(t *testing.T) {
	service := newService(t, newRepo())

	found, err := service.List(context.Background(), alias, mastermasking.Filter{Keyword: "SURVEYOR"})
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "CONTOH.SURVEYOR", found[0].Login)
}

// Pencarian cabang menelusuri NAMA cabang, bukan kodenya — sama seperti layar lama.
func TestSearchByBranchUsesBranchName(t *testing.T) {
	service := newService(t, newRepo())

	found, err := service.List(context.Background(), alias, mastermasking.Filter{
		By:      mastermasking.SearchByBranch,
		Keyword: "malang",
	})
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "MALANG", found[0].BranchName)
}

// Tipe pencarian yang tidak dikenal ditolak.
func TestUnknownSearchTypeRejected(t *testing.T) {
	service := newService(t, newRepo())

	_, err := service.List(context.Background(), alias, mastermasking.Filter{By: "modul", Keyword: "x"})
	require.Error(t, err)
}

// Penambahan mengisi pelaku dan waktu dari SESI dan JAM, bukan dari masukan.
//
// Ini bukan kerapian: `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang, dan
// jejak yang isinya ditentukan pemanggil tidak membuktikan apa pun.
func TestCreateStampsCallerAndClock(t *testing.T) {
	service := newService(t, newRepo())

	// Pemanggil mencoba menentukan pelakunya sendiri; nilainya harus ditimpa.
	attempt := sample()
	attempt.InputBy = "ORANG.LAIN"

	saved, err := service.Create(context.Background(), alias, attempt, "PETUGAS.ASLI")
	require.NoError(t, err)
	require.Equal(t, "PETUGAS.ASLI", saved.InputBy)
	require.Equal(t, fixedNow, saved.InputAt)
	require.NotEmpty(t, saved.ID, "ID dibuat penyimpanan")
}

// Baris kedua untuk pasangan cabang+login yang sama ditolak.
//
// Aturannya diambil apa adanya dari procedure lama, yang menolak insert dengan pesan
// "LOGIN … SUDAH ADA". Dua baris untuk orang yang sama di satu cabang berarti dua
// kewenangan yang keduanya berlaku, dan yang satu dapat luput saat dicabut.
func TestDuplicatePairRejected(t *testing.T) {
	service := newService(t, newRepo())

	duplicate := sample()
	duplicate.BranchID = "100081"
	duplicate.Login = "CONTOH.ADMIN"

	_, err := service.Create(context.Background(), alias, duplicate, "ADMIN")
	require.ErrorIs(t, err, mastermasking.ErrPairTaken)
}

// Perbandingan pasangan mengabaikan besar-kecil huruf.
func TestDuplicatePairIgnoresCase(t *testing.T) {
	service := newService(t, newRepo())

	duplicate := sample()
	duplicate.BranchID = "100081"
	duplicate.Login = "contoh.admin"

	_, err := service.Create(context.Background(), alias, duplicate, "ADMIN")
	require.ErrorIs(t, err, mastermasking.ErrPairTaken)
}

// Cabang yang tidak ada ditolak.
//
// Tanpa pemeriksaan ini, kewenangan dapat diberikan pada cabang yang tidak pernah ada —
// yang berarti ia tidak pernah berlaku sekaligus tidak pernah terlihat salah.
func TestUnknownBranchRejected(t *testing.T) {
	service := newService(t, newRepo())

	wrong := sample()
	wrong.BranchID = "999999"

	_, err := service.Create(context.Background(), alias, wrong, "ADMIN")
	require.ErrorIs(t, err, mastermasking.ErrBranchUnknown)
}

// Isian cacat ditolak sebelum penyimpanan disentuh.
func TestCreateValidates(t *testing.T) {
	service := newService(t, newRepo())

	empty := sample()
	empty.Login = ""

	_, err := service.Create(context.Background(), alias, empty, "ADMIN")
	var validationError *mastermasking.ValidationError
	require.ErrorAs(t, err, &validationError)
}

// Menyimpan ulang baris tanpa mengubah cabang maupun loginnya TIDAK ditolak.
//
// Tanpa pengecualian terhadap dirinya sendiri, setiap penyuntingan akan gagal sebagai
// "pasangan sudah dipakai" — bentrok dengan barisnya sendiri.
func TestUpdateSamePairAllowed(t *testing.T) {
	repo := newRepo()
	service := newService(t, repo)

	current, err := service.Get(context.Background(), alias, "1")
	require.NoError(t, err)

	current.ViewQuota = 77
	saved, err := service.Update(context.Background(), alias, "1", current, "PETUGAS")
	require.NoError(t, err)
	require.Equal(t, 77, saved.ViewQuota)
}

// Menyimpan form IKUT mengubah status, meniru layar lama.
//
// `MasterProteksi_Sec:22449` memuat isian berlabel "STATUS" yang terikat
// `InputData.BranchID`, dan procedure memetakannya ke `T_STSAKTF`. Yang menjaga status
// tidak berubah tanpa sengaja adalah LAYAR — tombol Ubah hanya muncul pada baris aktif —
// bukan penolakan di lapisan ini.
func TestUpdateCarriesStatusFromForm(t *testing.T) {
	service := newService(t, newRepo())

	current, err := service.Get(context.Background(), alias, "1")
	require.NoError(t, err)
	require.True(t, current.Active)

	current.Active = false
	saved, err := service.Update(context.Background(), alias, "1", current, "PETUGAS")
	require.NoError(t, err)
	require.False(t, saved.Active, "status pada form ikut tersimpan")

	// Dan barisnya tetap ada — menonaktifkan tidak pernah membuang apa pun (`D-66`).
	still, err := service.Get(context.Background(), alias, "1")
	require.NoError(t, err)
	require.False(t, still.Active)
}

// Mengubah baris yang tidak ada dijawab "tidak ditemukan", bukan galat lain.
func TestUpdateMissingRow(t *testing.T) {
	service := newService(t, newRepo())

	_, err := service.Update(context.Background(), alias, "9999", sample(), "PETUGAS")
	require.ErrorIs(t, err, mastermasking.ErrNotFound)
}

// Menonaktifkan tidak membuang baris — ia tetap dapat dibaca dan dapat diaktifkan kembali.
//
// Inilah wujud `D-66`: tombol bernama DELETE di layar lama pun hanya mengubah STS_AKTF.
func TestSetActiveKeepsRow(t *testing.T) {
	service := newService(t, newRepo())

	off, err := service.SetActive(context.Background(), alias, "1", false, "PENCABUT")
	require.NoError(t, err)
	require.False(t, off.Active)
	require.Equal(t, "PENCABUT", off.InputBy, "pencabutan ikut tercatat pelakunya")

	still, err := service.Get(context.Background(), alias, "1")
	require.NoError(t, err)
	require.False(t, still.Active)

	on, err := service.SetActive(context.Background(), alias, "1", true, "PEMULIH")
	require.NoError(t, err)
	require.True(t, on.Active)
}

// Daftar cabang dapat disaring, dan hasilnya dibatasi.
func TestBranchesFiltered(t *testing.T) {
	service := newService(t, newRepo())

	all, err := service.Branches(context.Background(), alias, "")
	require.NoError(t, err)
	require.NotEmpty(t, all)

	found, err := service.Branches(context.Background(), alias, "manado")
	require.NoError(t, err)
	require.Len(t, found, 1)
	require.Equal(t, "AGENCY MANADO", found[0].Name)
}
