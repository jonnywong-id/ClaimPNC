package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpicteknik"
	"claim-pnc/internal/masterpicteknik/directory"
	"claim-pnc/internal/masterpicteknik/repo/memory"
	"claim-pnc/internal/masterpicteknik/usecase"
	"claim-pnc/internal/portal"
)

const asm = "ASM"

func newService(t *testing.T) (*usecase.Service, *memory.Repo, *directory.Fake) {
	t.Helper()

	repo := memory.NewRepo(memory.SampleList()...)
	fake := directory.NewFake(directory.SampleEmployees()...)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterpicteknik.Repo, error) {
			if alias != asm {
				return nil, portal.ErrNotReady
			}
			return repo, nil
		},
		Directory: fake,
	})
	require.NoError(t, err)
	return service, repo, fake
}

func TestNewServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Directory: directory.NewFake()})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterpicteknik.Repo, error) { return nil, nil },
	})
	require.Error(t, err)
}

// Daftar hanya memuat petugas AKTIF — penyaring `STS_AKTIF = '1'` yang dipatok di Report
// Definition lama. Contoh PICTEKNIK04 sengaja nonaktif supaya aturan ini benar-benar
// teruji, bukan hanya dipercaya.
func TestListReturnsOnlyActiveTechnicians(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), asm)
	require.NoError(t, err)

	for _, technician := range list {
		require.True(t, technician.Active, "petugas nonaktif bocor ke daftar: %s", technician.OperatorID)
	}
	require.NotEmpty(t, list)
}

// Yang tidak muncul di daftar HARUS tetap dapat dibuka lewat ID — itulah yang membuat
// petugas nonaktif masih dapat diaktifkan kembali. `GetMasterPICTeknis` pun tidak
// menyaring status aktif.
func TestGetReachesInactiveTechnician(t *testing.T) {
	service, _, _ := newService(t)

	technician, err := service.Get(context.Background(), asm, "PICTEKNIK04")
	require.NoError(t, err)
	require.False(t, technician.Active)
}

func TestGetIsCaseInsensitive(t *testing.T) {
	service, _, _ := newService(t)

	technician, err := service.Get(context.Background(), asm, "picteknik01")
	require.NoError(t, err)
	require.Equal(t, "PICTEKNIK01", technician.OperatorID)
}

func TestGetUnknownReturnsNotFound(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Get(context.Background(), asm, "TIDAKADA")
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)
}

// Portal yang belum siap DITOLAK, tidak diam-diam dialihkan ke portal utama (`R-20`).
func TestEveryActionRejectsUnknownPortal(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "ASI")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Get(ctx, "ASI", "PICTEKNIK01")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Lookup(ctx, "ASI", "PICTEKNIK05")
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(ctx, "ASI", masterpicteknik.Technician{OperatorID: "X", Email: "a@b.co"})
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Update(ctx, "ASI", "PICTEKNIK01", masterpicteknik.Technician{Email: "a@b.co"})
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Nama TIDAK pernah diambil dari isian; ia selalu ditimpa hasil pencarian direktori.
// Tanpa aturan ini, master dapat memuat nama yang tidak cocok dengan identitasnya.
func TestCreateTakesNameFromDirectoryNotFromInput(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK05",
		Name:       "Nama Karangan Yang Dikirim Klien",
		Email:      "baru@example.invalid",
		Quota:      5,
		Active:     true,
	})
	require.NoError(t, err)
	require.Equal(t, "Contoh Petugas Baru", saved.Name)
}

// Atasan diusulkan dari blok EmpLeader respons direktori bila pengguna mengosongkannya.
func TestCreateFillsSupervisorFromDirectoryWhenEmpty(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK06",
		Email:      "surveyor@example.invalid",
		Active:     true,
	})
	require.NoError(t, err)
	require.Equal(t, "PICTEKNIK02", saved.Supervisor)
}

// Usulan direktori TIDAK menimpa pilihan pengguna: atasan menurut struktur organisasi
// belum tentu atasan penanganan klaim, dan layar Pega pun membiarkannya dapat disunting.
func TestCreateKeepsSupervisorChosenByUser(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK06",
		Email:      "surveyor@example.invalid",
		Supervisor: "PICTEKNIK03",
		Active:     true,
	})
	require.NoError(t, err)
	require.Equal(t, "PICTEKNIK03", saved.Supervisor)
}

// Inilah langkah "set error kalau tidak ditemukan di service": petugas yang tidak dikenal
// direktori DITOLAK, dan penolakannya menunjuk kolom id_operator supaya layar dapat
// menandainya.
func TestCreateRejectsEmployeeUnknownToDirectory(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "TIDAKTERDAFTAR",
		Email:      "x@example.invalid",
		Active:     true,
	})

	var validationError *masterpicteknik.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, masterpicteknik.FieldOperatorID, validationError.Violation[0].Field)
}

// Direktori yang tidak dapat dihubungi BUKAN kesalahan pengguna — ia tidak boleh
// disamarkan sebagai isian yang salah, karena tidak ada yang dapat diperbaiki pengguna.
func TestCreatePassesDirectoryOutageThrough(t *testing.T) {
	service, _, fake := newService(t)
	fake.SetError(masterpicteknik.ErrDirectoryUnreachable)

	_, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK05",
		Email:      "baru@example.invalid",
	})
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)

	var validationError *masterpicteknik.ValidationError
	require.False(t, errors.As(err, &validationError), "gangguan direktori tidak boleh menjadi galat validasi")
}

// Katalog yang belum diisi dibedakan dari jaringan yang putus: yang satu tidak akan pulih
// sendiri, yang lain akan.
func TestCreateDistinguishesUnconfiguredDirectory(t *testing.T) {
	service, _, fake := newService(t)
	fake.SetError(masterpicteknik.ErrDirectoryNotConfigured)

	_, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK05",
		Email:      "baru@example.invalid",
	})
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryNotConfigured)
	require.NotErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
}

func TestCreateRejectsDuplicateOperatorID(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK01",
		Email:      "lagi@example.invalid",
		Active:     true,
	})
	require.ErrorIs(t, err, masterpicteknik.ErrAlreadyExists)
}

// Isian yang cacat ditolak SEBELUM direktori ditembak — pencarian yang pasti sia-sia tidak
// perlu membebani layanan luar.
func TestCreateValidatesBeforeCallingDirectory(t *testing.T) {
	service, _, fake := newService(t)
	fake.SetError(errors.New("direktori tidak boleh dipanggil"))

	_, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "",
		Email:      "",
	})

	var validationError *masterpicteknik.ValidationError
	require.ErrorAs(t, err, &validationError)
}

// Tiga nilai yang tidak dikelola layar ini tidak boleh datang dari klien.
func TestCreateIgnoresFieldsItDoesNotOwn(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), asm, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK05",
		Email:      "baru@example.invalid",
		PanelGroup: "DIKARANG",
		Workload:   999,
		Active:     true,
	})
	require.NoError(t, err)
	require.Empty(t, saved.PanelGroup)
	require.Zero(t, saved.Workload)
}

func TestUpdateChangesEditableFields(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Update(context.Background(), asm, "PICTEKNIK02", masterpicteknik.Technician{
		Email:         "adjuster.baru@example.invalid",
		BusinessLine:  "NONMBU",
		Group:         "TEKNIK BANDUNG",
		Supervisor:    "PICTEKNIK01",
		Quota:         25,
		ExternalQuota: 4,
		Active:        true,
	})
	require.NoError(t, err)
	require.Equal(t, "adjuster.baru@example.invalid", saved.Email)
	require.Equal(t, "TEKNIK BANDUNG", saved.Group)
	require.Equal(t, 25, saved.Quota)
	require.Equal(t, 4, saved.ExternalQuota)
}

// ID diambil dari jalur URL, bukan dari badan permintaan: dua sumber untuk satu nilai
// berarti keduanya dapat berbeda, dan yang menang menjadi soal urutan baca.
func TestUpdateIgnoresOperatorIDInsideBody(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Update(context.Background(), asm, "PICTEKNIK02", masterpicteknik.Technician{
		OperatorID: "PICTEKNIK03",
		Email:      "adjuster@example.invalid",
		Active:     true,
	})
	require.NoError(t, err)
	require.Equal(t, "PICTEKNIK02", saved.OperatorID)
}

// Menonaktifkan petugas membuatnya hilang dari daftar, tetapi TIDAK hilang dari sistem —
// dan karena itu dapat diaktifkan kembali. Inilah yang menahan keputusan "daftar hanya
// yang aktif" berubah menjadi jalan buntu.
func TestUpdateCanDeactivateThenReactivate(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	_, err := service.Update(ctx, asm, "PICTEKNIK01", masterpicteknik.Technician{
		Email:  "contoh.kepalateknik@example.invalid",
		Active: false,
	})
	require.NoError(t, err)

	list, err := service.List(ctx, asm)
	require.NoError(t, err)
	for _, technician := range list {
		require.NotEqual(t, "PICTEKNIK01", technician.OperatorID)
	}

	// Masih terjangkau lewat ID, lalu diaktifkan kembali.
	_, err = service.Update(ctx, asm, "PICTEKNIK01", masterpicteknik.Technician{
		Email:  "contoh.kepalateknik@example.invalid",
		Active: true,
	})
	require.NoError(t, err)

	list, err = service.List(ctx, asm)
	require.NoError(t, err)
	require.Contains(t, operatorIDs(list), "PICTEKNIK01")
}

func TestUpdateUnknownReturnsNotFound(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Update(context.Background(), asm, "TIDAKADA", masterpicteknik.Technician{
		Email:  "x@example.invalid",
		Active: true,
	})
	require.ErrorIs(t, err, masterpicteknik.ErrNotFound)
}

// Nilai yang tidak dikelola layar ini dipertahankan dari baris tersimpan, tidak ditimpa
// apa pun yang dikirim klien.
func TestUpdateKeepsFieldsItDoesNotOwn(t *testing.T) {
	service, repo, _ := newService(t)
	ctx := context.Background()

	_, err := repo.Update(ctx, masterpicteknik.Technician{
		OperatorID: "PICTEKNIK02",
		Email:      "adjuster@example.invalid",
		Active:     true,
	})
	require.NoError(t, err)

	saved, err := service.Update(ctx, asm, "PICTEKNIK02", masterpicteknik.Technician{
		Email:      "adjuster@example.invalid",
		PanelGroup: "DIKARANG",
		Workload:   999,
		Active:     true,
	})
	require.NoError(t, err)
	require.NotEqual(t, "DIKARANG", saved.PanelGroup)
	require.NotEqual(t, 999, saved.Workload)
}

func TestLookupReturnsNameAndSupervisor(t *testing.T) {
	service, _, _ := newService(t)

	employee, err := service.Lookup(context.Background(), asm, "picteknik02")
	require.NoError(t, err)
	require.Equal(t, "Contoh Adjuster Madya", employee.Name)
	require.Equal(t, "PICTEKNIK01", employee.SupervisorID)
	require.Equal(t, "Contoh Kepala Teknik", employee.SupervisorName)
}

// Pencarian yang tidak menemukan siapa pun dijawab sebagai galat VALIDASI pada kolom
// id_operator, bukan sebagai galat sistem.
//
// Itu disengaja: yang salah memang isian penggunanya, dan jawaban berbentuk pelanggaran
// per kolom membuat layar dapat menandai kolom itu — bukan menampilkan pesan umum yang
// tidak menunjuk ke mana pun. Jalur simpan memakai pemetaan yang sama persis, sehingga
// pesan yang dilihat pengguna tidak berubah-ubah tergantung ia menekan Cari atau Simpan.
func TestLookupUnknownIsReportedOnOperatorIDField(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.Lookup(context.Background(), asm, "TIDAKTERDAFTAR")

	var validationError *masterpicteknik.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, masterpicteknik.FieldOperatorID, validationError.Violation[0].Field)
}

// Gangguan direktori pada jalur pencarian TIDAK menjadi galat validasi — tidak ada yang
// dapat diperbaiki pengguna di sana.
func TestLookupPassesDirectoryOutageThrough(t *testing.T) {
	service, _, fake := newService(t)
	fake.SetError(masterpicteknik.ErrDirectoryUnreachable)

	_, err := service.Lookup(context.Background(), asm, "PICTEKNIK01")
	require.ErrorIs(t, err, masterpicteknik.ErrDirectoryUnreachable)
}

func operatorIDs(list []masterpicteknik.Technician) []string {
	result := make([]string, 0, len(list))
	for _, technician := range list {
		result = append(result, technician.OperatorID)
	}
	return result
}
