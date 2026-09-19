package usecase_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/usecase"
)

// Seluruh uji di berkas ini berjalan TANPA basis data dan TANPA jaringan. Itu bukan
// kemewahan: gerbang penerimaan `TKT-B02-001` menuntut 20 penyimpanan berturut-turut
// tanpa nomor ganda, dan uji yang menuntut Oracle tidak akan pernah dijalankan orang
// sesering yang dibutuhkan.

const (
	firePolicy = "POL-FIRE-0001"
	paPolicy   = "POL-PA-0002"

	// testOperator adalah satu-satunya petugas pada seluruh aturan routing di berkas ini.
	//
	// Pembagian beban yang sesungguhnya berputar di antara beberapa orang, dan itu
	// diuji tersendiri pada TestLoadBalancingPicksLeastLoaded. Di sini tim
	// sengaja dibuat beranggota satu supaya uji alur menguji ALURNYA — bukan menebak
	// giliran siapa yang kebagian tugas berikutnya.
	testOperator = "ADMINPNC01"
)

// testTeams menugaskan seluruh aturan routing kepada satu orang.
func testTeams() map[string][]string {
	return map[string][]string{
		registrasi.RouterPNCAdmin:     {testOperator},
		registrasi.RouterPNCTechnical: {testOperator},
		registrasi.RouterRCLDoctor:    {testOperator},
	}
}

type environment struct {
	service   *usecase.Service
	store     *memory.Store
	parameter *memory.Parameter
	clock     *clock.Fixed
	caller    usecase.Caller
}

func setup(t *testing.T, roles ...string) environment {
	t.Helper()

	clock := clock.FixedAt(time.Date(2026, time.June, 10, 3, 0, 0, 0, time.UTC))
	store := memory.NewStore()
	parameter := memory.NewParameter()

	service, err := usecase.NewService(usecase.Options{
		ClaimRepo:          store,
		TaskRepo:           store.TaskRepo(),
		PolicyRepo:         memory.NewPolicyStore(memory.SamplePolicies(clock.Now())...),
		NumberIssuer:       memory.NewNumberIssuer(),
		Parameter:          parameter,
		ExchangeRateSource: memory.NewExchangeRateSource(),
		Assigner:           memory.NewAssigner(testTeams()),
		Notifier:           store,
		AuditRecorder:      store,
		IDGenerator:        memory.IDGenerator{},
		UnitOfWork:         store,
		Clock:              clock,
	})
	require.NoError(t, err)

	return environment{
		service:   service,
		store:     store,
		parameter: parameter,
		clock:     clock,
		caller: usecase.Caller{
			Identity:   testOperator,
			Name:       "Petugas Uji",
			Roles:      roles,
			Workbasket: []string{registrasi.WorkbasketRCLPUCL, registrasi.WorkbasketInvestigator, registrasi.WorkbasketCompliance},
		},
	}
}

// upToInputRegister membuka klaim dan menutup tahap View Polis, sehingga klaim berada
// tepat di tahap Input Register.
func (l environment) upToInputRegister(t *testing.T, policyNumber string) (registrasi.Claim, registrasi.Task) {
	t.Helper()
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{PolicyNumber: policyNumber, Portal: "ASM"}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageViewPolicy, start.Claim.CurrentStage)
	require.Empty(t, start.Claim.Number, "nomor klaim belum boleh terbit di tahap View Polis")

	result, err := l.service.CompleteStage(ctx, usecase.CompleteCommand{
		TaskID: start.Task.ID,
		Action: registrasi.ActionViewPolicy,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, result.Claim.CurrentStage)
	require.NotNil(t, result.NextTask)

	return result.Claim, *result.NextTask
}

func validInput(taskID string) usecase.RegisterCommand {
	return usecase.RegisterCommand{
		TaskID:       taskID,
		DateOfLoss:   time.Date(2026, time.June, 5, 0, 0, 0, 0, clock.ZoneWIB),
		ReportDate:   time.Date(2026, time.June, 6, 0, 0, 0, 0, clock.ZoneWIB),
		DateReceived: time.Date(2026, time.June, 7, 0, 0, 0, 0, clock.ZoneWIB),
		Location:     "Gudang A",
		Chronology:   "Kebakaran pada gudang penyimpanan.",
		Reporter: registrasi.Reporter{
			Name:     "Pelapor Uji",
			Phone:    "0800000000",
			Relation: 1,
		},
		EstimateValue: registrasi.Rupiah(10_000_000),
		Currency:      "IDR",
		TechnicalPIC:  "TEKNIK01",
		InsuredItem: []usecase.InsuredItemInput{{
			ID:       "OBJ-1",
			Name:     "Gudang",
			Location: "Gudang A",
			Coverage: []usecase.CoverageInput{{
				ID:          "CVG-1",
				CauseOfLoss: "11817",
				TSI:         registrasi.Rupiah(500_000_000),
				Spreading: []usecase.SpreadingInput{
					{TreatyKind: "10007", Name: "OR", Share: 600_000},
					{TreatyKind: "10008", Name: "Treaty", Share: 400_000},
				},
			}},
		}},
	}
}

func TestRegistrationUntilNumberIssued(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	result, err := l.service.SaveRegister(context.Background(), validInput(task.ID), l.caller)
	require.NoError(t, err)

	require.Regexp(t, `^PNCN\.26\.\d{4}$`, result.Claim.Number)
	require.Equal(t, registrasi.StatusRegistered, result.Claim.ClaimStatus)
	require.Equal(t, registrasi.ProcessRunning, result.Claim.ProcessStatus)

	// Lini Fire bukan PA dan bukan Travel: cabang ELSE membawanya ke Input Estimasi
	// Admin.
	require.Equal(t, registrasi.StageEstimateAdmin, result.Claim.CurrentStage)
	require.NotNil(t, result.NextTask)
	require.Equal(t, registrasi.QueueWorklist, result.NextTask.Queue)

	// Jejak keputusan menyebut cabang yang dipilih, supaya perpindahan tahap dapat
	// dijelaskan tanpa membaca kode.
	require.Contains(t, result.DecisionTrace, "IsBack → Continue")
	require.Contains(t, result.DecisionTrace, "IsPA → NotPA")
	require.Contains(t, result.DecisionTrace, "IsTravel → NonMBU")
}

// TestClaimNumberNeverDuplicated adalah kriteria penerimaan `TKT-B02-001`: 20
// penyimpanan berturut-turut, nol nomor ganda.
func TestClaimNumberNeverDuplicated(t *testing.T) {
	l := setup(t)
	seen := map[string]bool{}

	for i := 0; i < 20; i++ {
		_, task := l.upToInputRegister(t, firePolicy)

		input := validInput(task.ID)
		// InsuredItem dibuat berbeda tiap klaim supaya pemeriksaan duplikasi tidak menolak
		// klaim kedua — yang diuji di sini penomorannya, bukan aturan duplikasinya.
		input.InsuredItem[0].ID = "OBJ-" + strconv.Itoa(i)

		result, err := l.service.SaveRegister(context.Background(), input, l.caller)
		require.NoError(t, err)
		require.False(t, seen[result.Claim.Number], "nomor %s terbit dua kali", result.Claim.Number)
		seen[result.Claim.Number] = true
	}
	require.Len(t, seen, 20)
}

func TestBackButtonReturnsToViewPolicy(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	input := validInput(task.ID)
	input.Return = true

	result, err := l.service.SaveRegister(context.Background(), input, l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.StageViewPolicy, result.Claim.CurrentStage)
	require.Equal(t, registrasi.StatusReturned, result.Claim.ClaimStatus)
	require.Empty(t, result.Claim.Number, "tombol Back tidak boleh menerbitkan nomor klaim")

	// Isian tetap tersimpan — petugas yang kembali tidak kehilangan pekerjaannya.
	require.Equal(t, "Gudang A", result.Claim.Location)
	require.Len(t, result.Claim.InsuredItem, 1)
}

func TestValidationRejectsBeforeNumberIssued(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	input := validInput(task.ID)
	input.ReportDate = time.Date(2026, time.June, 1, 0, 0, 0, 0, clock.ZoneWIB) // sebelum kejadian

	_, err := l.service.SaveRegister(context.Background(), input, l.caller)

	var failure *registrasi.ValidationError
	require.ErrorAs(t, err, &failure)
	require.True(t, failure.Has(registrasi.ViolationReportDateBeforeLoss))

	// Tidak ada nomor yang terbakar, dan klaim tetap di tahapnya.
	summary, err := l.service.ViewClaim(context.Background(), task.ClaimID, l.caller)
	require.NoError(t, err)
	require.Empty(t, summary.Claim.Number)
	require.Equal(t, registrasi.StageInputRegister, summary.Claim.CurrentStage)
}

func TestDuplicateClaimRejectedAndNamesNumber(t *testing.T) {
	l := setup(t)

	_, firstTask := l.upToInputRegister(t, firePolicy)
	first, err := l.service.SaveRegister(context.Background(), validInput(firstTask.ID), l.caller)
	require.NoError(t, err)

	_, secondTask := l.upToInputRegister(t, firePolicy)
	_, err = l.service.SaveRegister(context.Background(), validInput(secondTask.ID), l.caller)

	var failure *registrasi.ValidationError
	require.ErrorAs(t, err, &failure)
	require.True(t, failure.Has(registrasi.ViolationDuplicateClaim))

	p, ok := failure.First()
	require.True(t, ok)
	require.Contains(t, p.Message, first.Claim.Number)
}

// TestNoticeOfLargeLosses menguji kedua batas ambang.
func TestNoticeOfLargeLosses(t *testing.T) {
	t.Run("tepat pada ambang tidak memicu", func(t *testing.T) {
		l := setup(t)
		_, task := l.upToInputRegister(t, firePolicy)

		input := validInput(task.ID)
		input.EstimateValue = registrasi.Rupiah(1_000_000_000)
		input.InsuredItem[0].Coverage[0].TSI = registrasi.Rupiah(5_000_000_000)

		result, err := l.service.SaveRegister(context.Background(), input, l.caller)
		require.NoError(t, err)
		require.False(t, result.LargeLoss)
		require.Empty(t, l.store.Notification())
	})

	t.Run("melebihi ambang memicu tepat satu peristiwa", func(t *testing.T) {
		l := setup(t)
		_, task := l.upToInputRegister(t, firePolicy)

		input := validInput(task.ID)
		input.EstimateValue = registrasi.Rupiah(1_000_000_001)
		input.InsuredItem[0].Coverage[0].TSI = registrasi.Rupiah(5_000_000_000)

		result, err := l.service.SaveRegister(context.Background(), input, l.caller)
		require.NoError(t, err)
		require.True(t, result.LargeLoss)

		messages := l.store.Notification()
		require.Len(t, messages, 1)
		require.Equal(t, registrasi.NotificationLargeLoss, messages[0].Kind)
		require.Equal(t, result.Claim.Number, messages[0].ClaimNumber)
		require.NotEmpty(t, messages[0].Recipients)
	})

	t.Run("ambang diubah tanpa deployment mengubah titik pemicu", func(t *testing.T) {
		l := setup(t)
		l.parameter.SetThreshold(registrasi.Rupiah(5_000_000))

		_, task := l.upToInputRegister(t, firePolicy)
		result, err := l.service.SaveRegister(context.Background(), validInput(task.ID), l.caller)
		require.NoError(t, err)
		require.True(t, result.LargeLoss, "estimasi Rp 10 juta melampaui ambang Rp 5 juta yang baru")
	})
}

// failSend adalah Notifier yang selalu gagal, dipakai membuktikan batas transaksi.
type failSend struct{}

var errSendFailed = errors.New("notifier uji: sengaja gagal")

func (failSend) Send(context.Context, registrasi.Notification) error { return errSendFailed }

// TestFailedSaveLeavesNoRow adalah kriteria penerimaan `TKT-B02-001`.
//
// Kegagalan sengaja dipicu pada langkah TERAKHIR di dalam transaksi, setelah klaim,
// tugas, dan jejak audit sudah ditulis. Bila batas transaksinya benar, ketiganya ikut
// dibatalkan.
func TestFailedSaveLeavesNoRow(t *testing.T) {
	clock := clock.FixedAt(time.Date(2026, time.June, 10, 3, 0, 0, 0, time.UTC))
	store := memory.NewStore()

	service, err := usecase.NewService(usecase.Options{
		ClaimRepo:          store,
		TaskRepo:           store.TaskRepo(),
		PolicyRepo:         memory.NewPolicyStore(memory.SamplePolicies(clock.Now())...),
		NumberIssuer:       memory.NewNumberIssuer(),
		Parameter:          memory.NewParameter(),
		ExchangeRateSource: memory.NewExchangeRateSource(),
		Assigner:           memory.NewAssigner(testTeams()),
		Notifier:           failSend{},
		AuditRecorder:      store,
		IDGenerator:        memory.IDGenerator{},
		UnitOfWork:         store,
		Clock:              clock,
	})
	require.NoError(t, err)

	caller := usecase.Caller{Identity: testOperator}
	ctx := context.Background()

	start, err := service.Start(ctx, usecase.StartCommand{PolicyNumber: firePolicy, Portal: "ASM"}, caller)
	require.NoError(t, err)
	proceed, err := service.CompleteStage(ctx, usecase.CompleteCommand{
		TaskID: start.Task.ID,
		Action: registrasi.ActionViewPolicy,
	}, caller)
	require.NoError(t, err)

	traceBefore := len(store.AuditTrail())

	input := validInput(proceed.NextTask.ID)
	input.EstimateValue = registrasi.Rupiah(2_000_000_000) // melampaui ambang, memicu notifier
	input.InsuredItem[0].Coverage[0].TSI = registrasi.Rupiah(5_000_000_000)

	_, err = service.SaveRegister(ctx, input, caller)
	require.ErrorIs(t, err, errSendFailed)

	// Claim tetap di tahap Input Register, tanpa nomor, dan tanpa jejak audit tambahan.
	summary, err := service.ViewClaim(ctx, start.Claim.ID, caller)
	require.NoError(t, err)
	require.Empty(t, summary.Claim.Number)
	require.Equal(t, registrasi.StageInputRegister, summary.Claim.CurrentStage)
	require.Len(t, store.AuditTrail(), traceBefore)
}

func TestAuditTrailRecordsOneRowOnIssue(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	before := len(l.store.AuditTrail())
	result, err := l.service.SaveRegister(context.Background(), validInput(task.ID), l.caller)
	require.NoError(t, err)

	trace := l.store.AuditTrail()
	require.Len(t, trace, before+1)

	last := trace[len(trace)-1]
	require.Equal(t, "KLAIM_TERDAFTAR", last.Event)
	require.Equal(t, l.caller.Identity, last.Actor)
	require.Equal(t, result.Claim.Number, last.ClaimNumber)
	require.False(t, last.At.IsZero())
}

// TestPAPathGoesThroughInvestigatorWorkbasket menelusuri jalur PA dari registrasi sampai
// tugas Workbasket, lalu membuktikan tugas antrean harus DIAMBIL lebih dulu.
func TestPAPathGoesThroughInvestigatorWorkbasket(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	_, task := l.upToInputRegister(t, paPolicy)

	input := validInput(task.ID)
	input.InsuredItem[0].Coverage[0].CauseOfLoss = registrasi.CauseOfLossPA

	result, err := l.service.SaveRegister(ctx, input, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageEstimatePA, result.Claim.CurrentStage)

	proceed, err := l.service.CompleteStage(ctx, usecase.CompleteCommand{
		TaskID: result.NextTask.ID,
		Action: registrasi.ActionInputSurveyor,
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInvestigator, proceed.Claim.CurrentStage)

	queuedTask := proceed.NextTask
	require.NotNil(t, queuedTask)
	require.Equal(t, registrasi.QueueWorkbasket, queuedTask.Queue)
	require.Empty(t, queuedTask.Owner, "tugas Workbasket lahir tanpa pemilik")

	// Menyelesaikan tugas antrean tanpa mengambilnya ditolak: tanpa langkah mengambil,
	// tidak ada catatan siapa yang mengerjakannya.
	_, err = l.service.CompleteStage(ctx, usecase.CompleteCommand{
		TaskID: queuedTask.ID,
		Action: registrasi.ActionInputInvestigation,
	}, l.caller)
	require.ErrorIs(t, err, registrasi.ErrNotTaskOwner)

	claimed, err := l.service.ClaimTask(ctx, queuedTask.ID, l.caller)
	require.NoError(t, err)
	require.Equal(t, l.caller.Identity, claimed.Owner)

	// Pengguna kedua yang mencoba mengambil tugas yang sama ditolak dengan galat yang
	// dapat dibedakan — bukan diam-diam kehilangan pekerjaannya.
	other := l.caller
	other.Identity = "ADMINPNC02"
	_, err = l.service.ClaimTask(ctx, queuedTask.ID, other)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyClaimed)
}

// TestInboxHoldsTwoGroups membuktikan isi Inbox sesuai definisi `D-79`.
func TestInboxHoldsTwoGroups(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	_, task := l.upToInputRegister(t, firePolicy)

	inbox, err := l.service.Inbox(ctx, l.caller)
	require.NoError(t, err)

	var found bool
	for _, i := range inbox {
		if i.ID == task.ID {
			found = true
		}
		require.True(t, i.Open(), "inbox tidak boleh memuat tugas yang sudah selesai")
	}
	require.True(t, found, "tugas milik pemanggil harus muncul di inbox")
}

// TestAlreadyAdvancedStageRejected menjaga hasil kerja tidak tertulis ke tahap yang
// sudah lewat.
func TestAlreadyAdvancedStageRejected(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	_, task := l.upToInputRegister(t, firePolicy)
	_, err := l.service.SaveRegister(ctx, validInput(task.ID), l.caller)
	require.NoError(t, err)

	// Menyimpan ulang dengan tugas yang sama: tugasnya sudah selesai.
	_, err = l.service.SaveRegister(ctx, validInput(task.ID), l.caller)
	require.ErrorIs(t, err, registrasi.ErrTaskAlreadyDone)
}

// TestMissingExchangeRateRejectsClaim adalah perilaku yang `ADR-0015` tetapkan.
func TestMissingExchangeRateRejectsClaim(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	input := validInput(task.ID)
	input.Currency = "USD" // belum ada kursnya

	_, err := l.service.SaveRegister(context.Background(), input, l.caller)
	require.ErrorIs(t, err, registrasi.ErrExchangeRateNotFound)
}

// TestLoadBalancingPicksLeastLoaded membuktikan algoritma `counter_quota` yang
// terbaca dari `RDB List/BrowsePICRandomTeam-SQL.xml`.
func TestLoadBalancingPicksLeastLoaded(t *testing.T) {
	assigner := memory.NewAssigner(map[string][]string{
		registrasi.RouterPNCAdmin: {"A", "B", "C"},
	})

	stage := registrasi.Stage{
		ID:     registrasi.StageInputRegister,
		Queue:  registrasi.QueueWorklist,
		Router: registrasi.RouterPNCAdmin,
	}

	for i := 0; i < 6; i++ {
		_, err := assigner.Assign(context.Background(), stage, registrasi.Claim{}, "PEMANGGIL")
		require.NoError(t, err)
	}

	load := assigner.Load()
	require.Equal(t, 2, load["A"])
	require.Equal(t, 2, load["B"])
	require.Equal(t, 2, load["C"])
}
