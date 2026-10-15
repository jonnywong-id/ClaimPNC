package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterautoclaim"
	"claim-pnc/internal/masterautoclaim/repo/memory"
	"claim-pnc/internal/masterautoclaim/usecase"
	"claim-pnc/internal/portal"
)

const (
	portalASM = "ASM"
	portalASI = "ASI"
)

// newService merakit layanan di atas DUA portal.
//
// Portal kedua sengaja ada dan sengaja KOSONG: ia yang membuktikan pemisahan
// antarentitas benar-benar berlaku, bukan hanya dimaksudkan (ADR-0030, R-20). Portal
// ketiga tidak dikenal sama sekali, dan penolakannya ikut diuji.
func newService(t *testing.T) (*usecase.Service, *memory.Repo, *memory.Repo) {
	t.Helper()

	asm := memory.NewSampleRepo()
	asi := memory.NewRepo(memory.Options{
		// Tanpa satu baris master pun, tetapi acuannya ada — supaya kegagalan yang
		// muncul benar-benar "tidak ada datanya", bukan "lookup-nya kosong".
		Sources:   memory.SampleBusinessSources(),
		Clients:   memory.SampleClients(),
		Banks:     memory.SampleBanks(),
		Committee: memory.SampleCommittee,
	})

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (masterautoclaim.Store, error) {
			switch alias {
			case portalASM:
				return asm, nil
			case portalASI:
				return asi, nil
			default:
				return nil, portal.ErrNotReady
			}
		},
	})
	require.NoError(t, err)
	return service, asm, asi
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func committeeActor() usecase.Actor { return usecase.Actor{Login: memory.SampleCommittee} }

func validInput() masterautoclaim.Input {
	return masterautoclaim.Input{
		Initial:         "AGN007",
		ReceiverName:    "PT. LEASING CONTOH UTAMA",
		BankName:        "BANK CONTOH NIAGA",
		AccountNumber:   "1000000007",
		MaxPercent:      "90",
		ReporterPIC:     "PIC Contoh Tujuh",
		ReporterEmail:   "pic.tujuh@contoh.example",
		ReceiverAddress: "Jalan Contoh Nomor 7",
		ClientID:        "CLI007",
		ClientName:      "PT. TERTANGGUNG CONTOH KETUJUH",
	}
}

func TestNewServiceRejectsMissingSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// Keempat tab layar lama adalah empat kombinasi penyaring. Uji ini yang memastikan
// keempatnya memilih baris yang berbeda — bukan satu penyaring yang kebetulan sama.
func TestListMatchesTheFourTabs(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	t.Run("Master Auto Klaim — status 1", func(t *testing.T) {
		list, err := service.List(ctx, portalASM, masterautoclaim.StatusApproved, false, committeeActor())
		require.NoError(t, err)
		require.Len(t, list, 2)
		for _, ac := range list {
			require.Equal(t, masterautoclaim.StatusApproved, ac.Status)
		}
	})

	t.Run("Waiting Approval — status 0", func(t *testing.T) {
		list, err := service.List(ctx, portalASM, masterautoclaim.StatusPending, false, committeeActor())
		require.NoError(t, err)
		require.Len(t, list, 3)
	})

	t.Run("Komite Approval — status 0 milik saya", func(t *testing.T) {
		list, err := service.List(ctx, portalASM, masterautoclaim.StatusPending, true, committeeActor())
		require.NoError(t, err)

		// TIGA baris menunggu, tetapi hanya DUA yang penyetujunya adalah pemanggil.
		// Yang ketiga (AGN005) tidak punya penyetuju sama sekali — dan justru itu yang
		// membuktikan penyaring komite benar-benar bekerja.
		require.Len(t, list, 2)
		for _, ac := range list {
			require.Equal(t, memory.SampleCommittee, ac.Committee)
		}
	})

	t.Run("Reject — status 2", func(t *testing.T) {
		list, err := service.List(ctx, portalASM, masterautoclaim.StatusRejected, false, committeeActor())
		require.NoError(t, err)
		require.Len(t, list, 1)
	})
}

// Antrean komite orang lain TIDAK terlihat. Penyaringnya memakai identitas pemanggil,
// bukan nilai yang dikirim layar.
func TestListCommitteeQueueIsPerCaller(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.List(context.Background(), portalASM,
		masterautoclaim.StatusPending, true, usecase.Actor{Login: "ORANGLAIN"})
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestListRejectsUnknownStatus(t *testing.T) {
	service, _, _ := newService(t)

	_, err := service.List(context.Background(), portalASM, "9", false, committeeActor())
	require.ErrorIs(t, err, masterautoclaim.ErrUnknownStatus)
}

// Portal yang tidak dikenal DITOLAK, tidak pernah dialihkan ke portal utama sebagai
// cadangan — itu inti R-20.
func TestUnknownPortalRejected(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	_, err := service.List(ctx, "TIDAKADA", masterautoclaim.StatusApproved, false, committeeActor())
	require.ErrorIs(t, err, portal.ErrNotReady)

	_, err = service.Create(ctx, "TIDAKADA", validInput(), committeeActor(), quietLogger())
	require.ErrorIs(t, err, portal.ErrNotReady)
}

// Data satu entitas TIDAK terlihat dari entitas lain.
func TestEntitiesAreSeparate(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	fromASM, err := service.List(ctx, portalASM, masterautoclaim.StatusApproved, false, committeeActor())
	require.NoError(t, err)
	require.NotEmpty(t, fromASM)

	fromASI, err := service.List(ctx, portalASI, masterautoclaim.StatusApproved, false, committeeActor())
	require.NoError(t, err)
	require.Empty(t, fromASI, "master entitas lain tidak boleh terlihat dari sini")
}

// Penambahan menurunkan TIGA nilai sendiri, dan tidak satu pun berasal dari layar.
func TestCreateDerivesStatusCommitteeAndClaimAllowed(t *testing.T) {
	service, _, _ := newService(t)

	saved, err := service.Create(context.Background(), portalASM, validInput(), committeeActor(), quietLogger())
	require.NoError(t, err)

	require.Equal(t, masterautoclaim.StatusPending, saved.Status, "baris baru selalu lahir menunggu")
	require.Equal(t, masterautoclaim.ClaimAllowedYes, saved.ClaimAllowed)
	require.Equal(t, memory.SampleCommittee, saved.Committee, "penyetuju diambil dari EMAILKOMITE, bukan dari layar")
	require.Equal(t, memory.SampleCommittee, saved.SubmittedBy, "USERINPUT adalah operator yang sedang masuk")
}

// Sumber bisnis yang sudah punya baris DITOLAK — padanan "Data sudah pernah diinput."
func TestCreateRejectsDuplicateInitial(t *testing.T) {
	service, _, _ := newService(t)

	input := validInput()
	input.Initial = "AGN001" // sudah ada di SampleList

	_, err := service.Create(context.Background(), portalASM, input, committeeActor(), quietLogger())
	require.ErrorIs(t, err, masterautoclaim.ErrInitialTaken)
}

// Bank yang tidak ada di GENERAL.LST_BANK_GROUP ditolak sebagai PELANGGARAN ISIAN,
// bukan galat teknis — padanan "Nama bank jangan diketik manual".
func TestCreateRejectsUnknownBankAsFieldViolation(t *testing.T) {
	service, _, _ := newService(t)

	input := validInput()
	input.BankName = "BANK YANG DIKETIK SENDIRI"

	_, err := service.Create(context.Background(), portalASM, input, committeeActor(), quietLogger())

	var validationError *masterautoclaim.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, "nama_bank", validationError.Violation[0].Field)
}

// EMAILKOMITE kosong BUKAN galat — sistem lama pun menyimpan KOMITE kosong. Yang wajib
// terjadi adalah barisnya tersimpan, dan akibatnya tercatat di log.
func TestCreateWithoutCommitteeStillSaves(t *testing.T) {
	repo := memory.NewRepo(memory.Options{
		Sources: memory.SampleBusinessSources(),
		Clients: memory.SampleClients(),
		Banks:   memory.SampleBanks(),
		// Committee sengaja kosong.
	})
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterautoclaim.Store, error) { return repo, nil },
	})
	require.NoError(t, err)

	saved, err := service.Create(context.Background(), portalASM, validInput(), committeeActor(), quietLogger())
	require.NoError(t, err)
	require.Empty(t, saved.Committee)

	// Dan akibatnya nyata: barisnya tidak akan pernah muncul di tab Komite Approval
	// siapa pun. Itulah yang membuat peringatan di log perlu ada.
	list, err := service.List(context.Background(), portalASM,
		masterautoclaim.StatusPending, true, committeeActor())
	require.NoError(t, err)
	require.Empty(t, list)
}

// Menyunting baris yang sudah disetujui MENGEMBALIKANNYA ke antrean persetujuan.
//
// Itu bentuk sistem lama: tombol Update pada tab Master mengirim stsapprove="0".
// Persetujuan lama tidak berlaku atas isi yang sudah berubah.
func TestSaveWithPendingStatusReturnsRowToQueue(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	before, err := service.Get(ctx, portalASM, "AGN001")
	require.NoError(t, err)
	require.Equal(t, masterautoclaim.StatusApproved, before.Status)

	after, err := service.Save(ctx, portalASM, "AGN001", saveInput(before, masterautoclaim.StatusPending), committeeActor())
	require.NoError(t, err)
	require.Equal(t, masterautoclaim.StatusPending, after.Status)
}

// Approve dan Reject adalah operasi yang SAMA dengan simpan, dibedakan statusnya —
// persis UpdateMstAutoClaim_act(stsapprove).
func TestSaveCarriesCommitteeDecision(t *testing.T) {
	for _, c := range []struct {
		name   string
		status masterautoclaim.ApprovalStatus
	}{
		{"approve", masterautoclaim.StatusApproved},
		{"reject", masterautoclaim.StatusRejected},
	} {
		t.Run(c.name, func(t *testing.T) {
			service, _, _ := newService(t)
			ctx := context.Background()

			before, err := service.Get(ctx, portalASM, "AGN003")
			require.NoError(t, err)
			require.Equal(t, masterautoclaim.StatusPending, before.Status)

			after, err := service.Save(ctx, portalASM, "AGN003", saveInput(before, c.status), committeeActor())
			require.NoError(t, err)
			require.Equal(t, c.status, after.Status)
			require.Equal(t, memory.SampleCommittee, after.SubmittedBy)
		})
	}
}

// NAMA_PENERIMA TIDAK BERUBAH lewat penyimpanan — keputusan Work Owner 2026-09-19, dan
// kueri lama pun tidak menyebut kolomnya.
func TestSaveNeverChangesReceiverName(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	before, err := service.Get(ctx, portalASM, "AGN001")
	require.NoError(t, err)

	input := saveInput(before, masterautoclaim.StatusApproved)
	input.ReceiverName = "NAMA YANG DICOBA DIGANTI"

	after, err := service.Save(ctx, portalASM, "AGN001", input, committeeActor())
	require.NoError(t, err)
	require.Equal(t, before.ReceiverName, after.ReceiverName)

	// Dan tersimpan begitu, bukan hanya pada jawaban permintaannya.
	reloaded, err := service.Get(ctx, portalASM, "AGN001")
	require.NoError(t, err)
	require.Equal(t, before.ReceiverName, reloaded.ReceiverName)
}

// KOMITE dipertahankan dari baris tersimpan, tidak pernah datang dari permintaan.
//
// Di Pega ia ditulis dari isi form, dan form yang belum dimuat mengosongkannya —
// sehingga menyetujui sebuah baris dapat MENGHAPUS penyetujunya sendiri. Jalur itu
// tidak dibawa, dan uji ini yang menjaganya tidak kembali.
func TestSaveKeepsCommittee(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	before, err := service.Get(ctx, portalASM, "AGN003")
	require.NoError(t, err)
	require.NotEmpty(t, before.Committee)

	after, err := service.Save(ctx, portalASM, "AGN003", saveInput(before, masterautoclaim.StatusApproved), committeeActor())
	require.NoError(t, err)
	require.Equal(t, before.Committee, after.Committee)
}

// CLAIM_ALLOWED tetap "1" pada jalur simpan. Ini SELISIH YANG DIRENCANAKAN terhadap
// Pega, yang menulis apa pun yang diketik pengguna — dan itulah yang membuat sebuah
// baris dapat "mati" tanpa satu pun pesan.
func TestSaveKeepsClaimAllowedAtOne(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	before, err := service.Get(ctx, portalASM, "AGN003")
	require.NoError(t, err)

	after, err := service.Save(ctx, portalASM, "AGN003", saveInput(before, masterautoclaim.StatusApproved), committeeActor())
	require.NoError(t, err)
	require.Equal(t, masterautoclaim.ClaimAllowedYes, after.ClaimAllowed)
	require.True(t, after.Usable(), "baris yang disetujui harus benar-benar dapat dipakai klaim otomatis")
}

func TestSaveRejectsUnknownStatus(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	before, err := service.Get(ctx, portalASM, "AGN003")
	require.NoError(t, err)

	_, err = service.Save(ctx, portalASM, "AGN003", saveInput(before, "9"), committeeActor())
	require.ErrorIs(t, err, masterautoclaim.ErrUnknownStatus)
}

func TestSaveRejectsMissingRow(t *testing.T) {
	service, _, _ := newService(t)

	input := masterautoclaim.Input{
		BankName:        "BANK CONTOH NIAGA",
		AccountNumber:   "1",
		MaxPercent:      "10",
		ReporterPIC:     "PIC",
		ReporterEmail:   "a@contoh.example",
		ReceiverAddress: "Jalan Contoh",
		Status:          masterautoclaim.StatusApproved,
	}
	_, err := service.Save(context.Background(), portalASM, "TIDAKADA", input, committeeActor())
	require.ErrorIs(t, err, masterautoclaim.ErrNotFound)
}

// Kata kunci yang terlalu pendek dijawab daftar kosong, BUKAN galat: pengguna yang baru
// mengetik satu huruf belum melakukan kesalahan apa pun.
func TestLookupShortKeywordReturnsNothing(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	source, err := service.SearchBusinessSources(ctx, portalASM, "A")
	require.NoError(t, err)
	require.Empty(t, source)

	client, err := service.SearchClients(ctx, portalASM, "")
	require.NoError(t, err)
	require.Empty(t, client)
}

// Pembuangan titik dipertahankan dari Pega: "PT. LEASING" dan "PT LEASING" sama-sama
// ketemu.
func TestLookupIgnoresDots(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	withDot, err := service.SearchBusinessSources(ctx, portalASM, "PT. LEASING")
	require.NoError(t, err)
	require.NotEmpty(t, withDot)

	withoutDot, err := service.SearchBusinessSources(ctx, portalASM, "PT LEASING")
	require.NoError(t, err)
	require.Equal(t, withDot, withoutDot)
}

func TestLookupFindsByID(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.SearchBusinessSources(context.Background(), portalASM, "AGN009")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "AGN009", list[0].ID)
}

func TestListBanks(t *testing.T) {
	service, _, _ := newService(t)

	list, err := service.ListBanks(context.Background(), portalASM)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	for _, b := range list {
		require.NotEmpty(t, b.Code)
		require.NotEmpty(t, b.Name)
	}
}

// Kegagalan penyimpanan diteruskan apa adanya, tidak ditelan menjadi "berhasil".
func TestStoreFailureSurfaces(t *testing.T) {
	repo := memory.NewSampleRepo()
	failure := errors.New("basis data tidak dapat dihubungi")
	repo.SetError(failure)

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterautoclaim.Store, error) { return repo, nil },
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), portalASM, masterautoclaim.StatusApproved, false, committeeActor())
	require.ErrorIs(t, err, failure)
}

func TestEnsurePortalReady(t *testing.T) {
	service, _, _ := newService(t)

	require.NoError(t, service.EnsurePortalReady(portalASM))
	require.Error(t, service.EnsurePortalReady("TIDAKADA"))
}

// saveInput menyusun badan penyimpanan dari baris yang sudah ada.
//
// Ia meniru apa yang dilakukan layar: mengirim ULANG seluruh isian beserta status yang
// dikehendaki — bentuk yang dipertahankan atas keputusan Work Owner 2026-09-19.
func saveInput(from masterautoclaim.AutoClaim, status masterautoclaim.ApprovalStatus) masterautoclaim.Input {
	return masterautoclaim.Input{
		BankName:        from.BankName,
		AccountNumber:   from.AccountNumber,
		MaxPercent:      from.MaxPercent,
		ReporterPIC:     from.ReporterPIC,
		ReporterEmail:   from.ReporterEmail,
		ReceiverAddress: from.ReceiverAddress,
		ClientID:        from.ClientID,
		ClientName:      from.ClientName,
		Status:          status,
	}
}
