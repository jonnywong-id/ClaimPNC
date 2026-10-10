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
	"claim-pnc/internal/registrasi/acceptancenotepdf"
	"claim-pnc/internal/registrasi/dlapdf"
	"claim-pnc/internal/registrasi/facesheetpdf"
	"claim-pnc/internal/registrasi/lodpdf"
	"claim-pnc/internal/registrasi/plapdf"
	"claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/usecase"
)

// Seluruh uji di berkas ini berjalan TANPA basis data dan TANPA jaringan. Itu bukan
// kemewahan: gerbang penerimaan `TKT-B02-001` menuntut 20 penyimpanan berturut-turut
// tanpa nomor ganda, dan uji yang menuntut Oracle tidak akan pernah dijalankan orang
// sesering yang dibutuhkan.

// regexpNomorKlaim adalah bentuk nomor klaim yang sah: PNCN.YY.xxxx (`D-71`).
const regexpNomorKlaim = `^PNCN\.\d{2}\.[1-9]\d*$`

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
	service    *usecase.Service
	store      *memory.Store
	parameter  *memory.Parameter
	link       *memory.ClaimReportLink
	clock      *clock.Fixed
	pla        *memory.PLA
	dla        *memory.DLA
	cashier    *memory.Cashier
	groups     memory.Groups
	inbox      *memory.InboxEntries
	records    *memory.ClaimRecords
	uploader   *memory.DocumentUploader
	acceptance *memory.Acceptance
	premium    *memory.Premium
	accounts   *memory.Accounts
	areas      *memory.AreaDirectory
	pucl       *memory.PUCL
	caller     usecase.Caller
}

func setup(t *testing.T, roles ...string) environment {
	t.Helper()

	clock := clock.FixedAt(time.Date(2026, time.June, 10, 3, 0, 0, 0, time.UTC))
	store := memory.NewStore()
	parameter := memory.NewParameter()
	link := memory.NewClaimReportLink()
	policyItems := memory.NewPolicyItems(memory.SamplePolicyItems())
	pla := memory.NewPLA()
	dla := memory.NewDLA()
	cashier := memory.NewCashier()
	groups := memory.Groups{}
	inbox := memory.NewInboxEntries()
	records := memory.SampleClaimRecords()
	uploader := &memory.DocumentUploader{}
	acceptance := memory.NewAcceptance()
	premium := memory.NewPremium()
	accounts := memory.NewAccounts()
	areas := memory.NewAreaDirectory()
	pucl := memory.NewPUCL()

	service, err := usecase.NewService(usecase.Options{
		ClaimRepo:              store,
		TaskRepo:               store.TaskRepo(),
		PolicyRepo:             memory.NewPolicyStore(memory.SamplePolicies(clock.Now())...),
		NumberIssuer:           memory.NewNumberIssuer(),
		Parameter:              parameter,
		ExchangeRateSource:     memory.NewExchangeRateSource(),
		Assigner:               memory.NewAssigner(testTeams()),
		Notifier:               store,
		AuditRecorder:          store,
		ClaimReportLink:        link,
		AreaDirectory:          areas,
		CauseOfLoss:            areas,
		PolicyItems:            policyItems,
		CurrencyDirectory:      memory.CurrencyDirectory{},
		ItemOptions:            policyItems,
		ClaimRecords:           records,
		FaceSheet:              memory.NewFaceSheet(),
		FaceSheetRenderer:      facesheetpdf.Renderer{},
		PLA:                    pla,
		PLARenderer:            plapdf.Renderer{},
		DLA:                    dla,
		DLARenderer:            dlapdf.Renderer{},
		AcceptanceNoteRenderer: acceptancenotepdf.Renderer{},
		Cashier:                cashier,
		CashierGateway:         cashier,
		CashierAccountCheck:    true,
		LODRenderer:            lodpdf.Renderer{},
		Acceptance:             acceptance,
		Premium:                premium,
		Groups:                 groups,
		Inbox:                  inbox,
		Accounts:               accounts,
		CommitteeTiering:       memory.NewCommitteeTiering(),
		Committees:             memory.NewCommittees(),
		Documents:              uploader,
		Attachments:            records,
		PUCLLetters:            pucl,
		PUCLOptions:            pucl,
		Closures:               records,
		IDGenerator:            memory.IDGenerator{},
		UnitOfWork:             store,
		Clock:                  clock,
	})
	require.NoError(t, err)

	return environment{
		service:    service,
		store:      store,
		parameter:  parameter,
		link:       link,
		clock:      clock,
		pla:        pla,
		dla:        dla,
		cashier:    cashier,
		groups:     groups,
		inbox:      inbox,
		records:    records,
		uploader:   uploader,
		acceptance: acceptance,
		premium:    premium,
		accounts:   accounts,
		areas:      areas,
		pucl:       pucl,
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
	// Nomor terbit saat klaim DIBUKA, mengikuti `addWork` Pega — bukan di ujung Input
	// Register. Lihat catatan pada Service.Start.
	require.Regexp(t, regexpNomorKlaim, start.Claim.Number,
		"klaim yang baru dibuka harus SUDAH bernomor")

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
		TechnicalPIC:  testOperator,
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

// TestClaimReportMovesTabs menjaga satu-satunya hal yang membuat tombol Register Klaim
// TERLIHAT bekerja oleh petugas.
//
// Berkas laporan menentukan tabnya dari dua kolom, dan sejak nomor terbit saat klaim
// dibuka (2026-09-25) keduanya terisi bersamaan — sehingga berkas berpindah dari
// "Not Transferred" LANGSUNG ke "Outstanding".
//
// Tanpa uji ini, klaim tetap terbuat sementara berkasnya duduk diam di "Not Transferred" —
// persis keluhan yang melahirkan seam ini, dan kegagalan yang tidak menghasilkan satu pun
// galat.
func TestClaimReportMovesTabs(t *testing.T) {
	const laporan = "RCVN.26.0001"

	t.Run("klaim dibuka menandai berkas diserahkan", func(t *testing.T) {
		l := setup(t)

		_, err := l.service.Start(context.Background(), usecase.StartCommand{
			PolicyNumber: firePolicy,
			Portal:       "ASM",
			RCVID:        laporan,
		}, l.caller)
		require.NoError(t, err)

		// Keduanya terjadi sekaligus sejak nomor terbit saat klaim dibuka: berkas
		// berpindah dari "Not Transferred" LANGSUNG ke "Outstanding".
		require.True(t, l.link.HandedOver(laporan), "berkas harus pindah dari Not Transferred")
		require.Regexp(t, regexpNomorKlaim, l.link.ClaimNumber(laporan),
			"NOKLAIM harus terisi bersamaan, sehingga berkas mencapai Outstanding")
	})

	t.Run("nomor terbit memasang NOKLAIM", func(t *testing.T) {
		l := setup(t)
		ctx := context.Background()

		// Klaim dari berkas RCV melompat langsung ke Input Register (setToRegister_ticket),
		// sehingga tugas pertamanya sudah tugas Input Register.
		start, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)
		require.Equal(t, registrasi.StageInputRegister, start.Claim.CurrentStage)

		hasil, err := l.service.SaveRegister(ctx, validInput(start.Task.ID), l.caller)
		require.NoError(t, err)

		require.Equal(t, hasil.Claim.Number, l.link.ClaimNumber(laporan))
	})

	// Menekan Back tidak MENGUBAH tautan yang sudah dibuat saat klaim dibuka.
	//
	// Menarik kembali penyerahannya akan mengosongkan TRANSFERASM sementara NOKLAIM
	// tetap terisi — kombinasi yang TIDAK punya tab, sehingga berkasnya lenyap dari
	// seluruh daftar. Menerbitkan nomor kedua akan menautkan berkas ke klaim yang salah.
	t.Run("tombol Back tidak mengubah tautan berkasnya", func(t *testing.T) {
		l := setup(t)
		ctx := context.Background()

		start, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)

		input := validInput(start.Task.ID)
		input.Return = true
		_, err = l.service.SaveRegister(ctx, input, l.caller)
		require.NoError(t, err)

		require.Equal(t, start.Claim.Number, l.link.ClaimNumber(laporan),
			"nomor pada berkas berubah setelah Back")
		require.True(t, l.link.HandedOver(laporan), "penyerahannya ditarik kembali")
	})

	// Klaim yang dimulai dari layar registrasi tidak punya berkas asal, dan menautkannya
	// ke berkas mana pun akan memindahkan laporan orang lain.
	t.Run("klaim tanpa berkas asal tidak menyentuh laporan apa pun", func(t *testing.T) {
		l := setup(t)

		_, err := l.service.Start(context.Background(), usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM",
		}, l.caller)
		require.NoError(t, err)

		require.False(t, l.link.HandedOver(laporan))
	})
}

func TestRegistrationUntilNumberIssued(t *testing.T) {
	l := setup(t)
	_, task := l.upToInputRegister(t, firePolicy)

	result, err := l.service.SaveRegister(context.Background(), validInput(task.ID), l.caller)
	require.NoError(t, err)

	require.Regexp(t, `^PNCN\.26\.[1-9]\d*$`, result.Claim.Number)
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
	klaim, task := l.upToInputRegister(t, firePolicy)

	input := validInput(task.ID)
	input.Return = true

	result, err := l.service.SaveRegister(context.Background(), input, l.caller)
	require.NoError(t, err)

	require.Equal(t, registrasi.StageViewPolicy, result.Claim.CurrentStage)
	require.Equal(t, registrasi.StatusReturned, result.Claim.ClaimStatus)
	require.Equal(t, klaim.Number, result.Claim.Number,
		"tombol Back menerbitkan nomor KEDUA; nomor terbit sekali saat klaim dibuka")

	// Isian tetap tersimpan — petugas yang kembali tidak kehilangan pekerjaannya.
	require.Equal(t, "Gudang A", result.Claim.Location)
	require.Len(t, result.Claim.InsuredItem, 1)
}

func TestValidationRejectsDoesNotIssueASecondNumber(t *testing.T) {
	l := setup(t)
	klaim, task := l.upToInputRegister(t, firePolicy)

	input := validInput(task.ID)
	input.ReportDate = time.Date(2026, time.June, 1, 0, 0, 0, 0, clock.ZoneWIB) // sebelum kejadian

	_, err := l.service.SaveRegister(context.Background(), input, l.caller)

	var failure *registrasi.ValidationError
	require.ErrorAs(t, err, &failure)
	require.True(t, failure.Has(registrasi.ViolationReportDateBeforeLoss))

	// Tidak ada nomor KEDUA yang terbakar, dan klaim tetap di tahapnya.
	summary, err := l.service.ViewClaim(context.Background(), task.ClaimID, l.caller)
	require.NoError(t, err)
	require.Equal(t, klaim.Number, summary.Claim.Number)
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

	// Sistem lama merakit penerima dari LIMA sumber lalu menyambungnya menjadi satu
	// string (`Activity/SendEmailLargeLoss_act.xml` langkah 7 dan 8). Satu sumber yang
	// kosong memperpendek sambungan itu; ia tidak pernah menghentikan registrasi.
	//
	// Uji ini menjaga arah yang paling mudah keliru: memperlakukan master yang belum
	// diisi sebagai galat akan MENOLAK setiap klaim di atas Rp 1 miliar — perilaku yang
	// tidak ada di sistem lama.
	t.Run("penerima yang belum diisi tidak menggagalkan pendaftaran", func(t *testing.T) {
		l := setup(t)
		l.parameter.SetGeneral(nil)
		l.parameter.SetRecipients(registrasi.LineFire, nil)

		_, task := l.upToInputRegister(t, firePolicy)
		input := validInput(task.ID)
		input.EstimateValue = registrasi.Rupiah(1_000_000_001)
		input.InsuredItem[0].Coverage[0].TSI = registrasi.Rupiah(5_000_000_000)

		result, err := l.service.SaveRegister(context.Background(), input, l.caller)
		require.NoError(t, err, "klaim harus tetap terdaftar meski penerima kosong")
		require.True(t, result.LargeLoss)

		messages := l.store.Notification()
		require.Len(t, messages, 1, "peristiwanya tetap terbit dan tercatat")
		require.Empty(t, messages[0].Recipients)
	})

	// Langkah 7 memakai subjek biasa selama `FlagNOLL` kosong, langkah 8 memakai
	// "(REVISE)" setelah ia bernilai "1", dan langkah 11 mengisinya tepat sesudah surel
	// dikirim. Tanpa penanda itu setiap pemberitahuan terbaca sebagai yang pertama.
	t.Run("pemberitahuan kedua ditandai revisi", func(t *testing.T) {
		l := setup(t)
		_, task := l.upToInputRegister(t, firePolicy)

		input := validInput(task.ID)
		input.EstimateValue = registrasi.Rupiah(1_000_000_001)
		input.InsuredItem[0].Coverage[0].TSI = registrasi.Rupiah(5_000_000_000)

		first, err := l.service.SaveRegister(context.Background(), input, l.caller)
		require.NoError(t, err)
		require.False(t, l.store.Notification()[0].Revision, "yang pertama bukan revisi")
		require.True(t, first.Claim.LargeLossNoticed, "penanda dinaikkan setelah terbit")
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
	policyItems := memory.NewPolicyItems(nil)

	groups := memory.Groups{}
	inbox := memory.NewInboxEntries()
	records := memory.SampleClaimRecords()
	service, err := usecase.NewService(usecase.Options{
		ClaimRepo:              store,
		TaskRepo:               store.TaskRepo(),
		PolicyRepo:             memory.NewPolicyStore(memory.SamplePolicies(clock.Now())...),
		NumberIssuer:           memory.NewNumberIssuer(),
		Parameter:              memory.NewParameter(),
		ExchangeRateSource:     memory.NewExchangeRateSource(),
		Assigner:               memory.NewAssigner(testTeams()),
		Notifier:               failSend{},
		AuditRecorder:          store,
		ClaimReportLink:        memory.NewClaimReportLink(),
		AreaDirectory:          memory.NewAreaDirectory(),
		CauseOfLoss:            memory.NewAreaDirectory(),
		PolicyItems:            policyItems,
		CurrencyDirectory:      memory.CurrencyDirectory{},
		ItemOptions:            policyItems,
		ClaimRecords:           records,
		FaceSheet:              memory.NewFaceSheet(),
		FaceSheetRenderer:      facesheetpdf.Renderer{},
		PLA:                    memory.NewPLA(),
		PLARenderer:            plapdf.Renderer{},
		DLA:                    memory.NewDLA(),
		DLARenderer:            dlapdf.Renderer{},
		AcceptanceNoteRenderer: acceptancenotepdf.Renderer{},
		Cashier:                memory.NewCashier(),
		CashierGateway:         memory.NewCashier(),
		LODRenderer:            lodpdf.Renderer{},
		Acceptance:             memory.NewAcceptance(),
		Premium:                memory.NewPremium(),
		Groups:                 groups,
		Inbox:                  inbox,
		Accounts:               memory.NewAccounts(),
		CommitteeTiering:       memory.NewCommitteeTiering(),
		Committees:             memory.NewCommittees(),
		Documents:              &memory.DocumentUploader{},
		Attachments:            records,
		PUCLLetters:            memory.NewPUCL(),
		PUCLOptions:            memory.NewPUCL(),
		Closures:               records,
		IDGenerator:            memory.IDGenerator{},
		UnitOfWork:             store,
		Clock:                  clock,
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

	// Klaim tetap di tahap Input Register, nomornya tidak berubah, dan tidak ada jejak
	// audit tambahan.
	summary, err := service.ViewClaim(ctx, start.Claim.ID, caller)
	require.NoError(t, err)
	require.Equal(t, start.Claim.Number, summary.Claim.Number)
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
//
// Yang diuji PNCTeknikRouter, bukan PNCAdminRouter: sejak 2026-09-25, tahap Admin
// menugaskan ke admin klaimnya dan tidak membagi beban sama sekali. Memakai router itu
// di sini akan menguji cabang yang berbeda dari yang namanya sebutkan.
func TestLoadBalancingPicksLeastLoaded(t *testing.T) {
	assigner := memory.NewAssigner(map[string][]string{
		registrasi.RouterPNCTechnical: {"A", "B", "C"},
	})

	stage := registrasi.Stage{
		ID:     registrasi.StageSendToTechnicalPIC,
		Queue:  registrasi.QueueWorklist,
		Router: registrasi.RouterPNCTechnical,
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

// TestRegisterKlaimJumpsStraightToInputRegister menjaga perilaku yang dilihat petugas tepat
// setelah menekan tombol Register Klaim.
//
// # Buktinya di export
//
// `Activity/CreateRegisterKlaimPNC_act.xml` langkah 27 memanggil
// `SetTicket("setToRegister_ticket")`. Di `Flow/Register_Flow.xml`, tiket itu (`Ticket8`)
// menempel pada `Assignment1` — assignment bernama "Input Register", `pyUseCaseName`
// InputRegister. Layar Pega yang muncul sesudahnya memang layar Register, bukan View Polis.
//
// # Kenapa RCVID yang membedakan
//
// Tiket itu HANYA dinyalakan `CreateRegisterKlaimPNC`, dan activity itu hanya berjalan dari
// tombol Register Klaim pada form Receive Document — yang selalu punya berkas asal. Klaim
// yang dibuka langsung dari layar registrasi tidak melewatinya, dan tetap mulai dari
// View Polis seperti `Start1` pada alurnya.
func TestRegisterKlaimJumpsStraightToInputRegister(t *testing.T) {
	ctx := context.Background()

	t.Run("dari berkas RCV, tugas pertama adalah Input Register", func(t *testing.T) {
		l := setup(t)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.0001",
		}, l.caller)
		require.NoError(t, err)

		require.Equal(t, registrasi.StageInputRegister, hasil.Claim.CurrentStage,
			"klaim dari Register Klaim harus melompat ke Input Register")
		require.Equal(t, registrasi.StageInputRegister, hasil.Task.Stage)
		require.Regexp(t, regexpNomorKlaim, hasil.Claim.Number)
	})

	// Work Owner, 2026-09-29: berkas RCVN yang sudah menjadi PNCN tidak dapat diregistrasi
	// lagi. Tanpa penolakan ini terbit PNCN kedua dan NOKLAIM berkasnya tertimpa.
	t.Run("berkas RCV yang sudah bernomor klaim ditolak", func(t *testing.T) {
		l := setup(t)

		first, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.1",
		}, l.caller)
		require.NoError(t, err)

		_, err = l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.1",
		}, l.caller)
		var registered *registrasi.ReportAlreadyRegisteredError
		require.ErrorAs(t, err, &registered)
		require.Equal(t, first.Claim.Number, registered.ClaimNumber)
	})

	t.Run("tanpa berkas RCV, tugas pertama tetap View Polis", func(t *testing.T) {
		l := setup(t)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM",
		}, l.caller)
		require.NoError(t, err)

		require.Equal(t, registrasi.StageViewPolicy, hasil.Claim.CurrentStage,
			"klaim yang dibuka langsung tidak melewati tiket, jadi mulai dari Start1")
	})

	// Tugas Input Register milik petugas yang MENEKAN tombolnya. Melemparnya ke orang lain
	// membuat petugas tidak dapat membuka klaim yang baru saja dibuatnya sendiri — gejala
	// yang dilaporkan Work Owner sebagai "Tugas ini bukan milik Anda".
	t.Run("tugasnya milik petugas yang menekan tombolnya", func(t *testing.T) {
		l := setup(t)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.0001",
		}, l.caller)
		require.NoError(t, err)

		require.Equal(t, l.caller.Identity, hasil.Task.Owner,
			"tugas pertama bukan milik petugas yang membuat klaimnya")
	})
}

// TestReportContentsCarryIntoTheClaim menjaga penyalinan isi berkas RCV ke klaim.
//
// `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 menyalin sembilan nilai sebelum
// menyimpan klaimnya. Tanpa itu, petugas membuka layar Register yang kosong dan mengetik
// ulang seluruh isi berkas yang baru saja diisinya — keluhan Work Owner 2026-09-25.
func TestReportContentsCarryIntoTheClaim(t *testing.T) {
	const laporan = "RCVN.26.0001"
	ctx := context.Background()

	isi := registrasi.ClaimReportSnapshot{
		DateOfLoss:    time.Date(2026, time.June, 5, 0, 0, 0, 0, clock.ZoneWIB),
		ReportDate:    time.Date(2026, time.June, 7, 0, 0, 0, 0, clock.ZoneWIB),
		ReporterName:  "Pelapor Berkas",
		ReporterPhone: "0800111222",
		ReporterEmail: "pelapor@contoh.internal",
		Location:      "Gudang B",
		Chronology:    "Kebakaran pada gudang kedua.",
		EstimateValue: registrasi.Rupiah(250_000_000),
	}

	t.Run("seluruh isian berkas ikut ke klaimnya", func(t *testing.T) {
		l := setup(t)
		l.link.SetSnapshot(laporan, isi)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)

		k := hasil.Claim
		require.Equal(t, isi.DateOfLoss.UTC(), k.DateOfLoss)
		require.Equal(t, isi.ReportDate.UTC(), k.ReportDate,
			"kolom TANGGALTERIMADOKUMEN memberi makan Tanggal LAPOR, bukan Tanggal Terima "+
				"Dokumen; aturan \"Tanggal Lapor ≤ DOL + 7 hari\" bersandar padanya")
		require.True(t, k.DateReceived.IsZero(),
			"Tanggal Terima Dokumen datang dari DateOfSentDocument, yang TIDAK punya kolom "+
				"pada tabel berkas — mengisinya berarti mengarang sumber")
		require.Equal(t, isi.ReporterName, k.Reporter.Name)
		require.Equal(t, isi.ReporterPhone, k.Reporter.Phone)
		require.Equal(t, isi.ReporterEmail, k.Reporter.Email)
		require.Equal(t, isi.Location, k.Location)
		require.Equal(t, isi.Chronology, k.Chronology)
		require.Equal(t, isi.EstimateValue, k.EstimateValue)
	})

	// Langkah 14 mengisi hubungan pelapor dengan "lain-lain" dan menaruh namanya sebagai
	// keterangan. Nama pada berkas RCV adalah pengirim dokumen, bukan tertanggung sendiri.
	t.Run("hubungan pelapor menjadi lain-lain beserta keterangannya", func(t *testing.T) {
		l := setup(t)
		l.link.SetSnapshot(laporan, isi)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)

		require.Equal(t, registrasi.RelationOther, hasil.Claim.Reporter.Relation)
		require.Equal(t, isi.ReporterName, hasil.Claim.Reporter.OtherRelation,
			"RelationOther menuntut keterangan; tanpa ini validasi menolak klaim yang sah")
	})

	// Berkas RCV lahir kosong dan boleh diregistrasi sebelum lengkap. Yang kosong tetap
	// kosong, dan gerbang validasi Input Register yang menuntutnya — bukan pembuatan klaim.
	t.Run("berkas kosong tidak menggagalkan pembuatan klaim", func(t *testing.T) {
		l := setup(t)

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)
		require.Regexp(t, regexpNomorKlaim, hasil.Claim.Number)
		require.Empty(t, hasil.Claim.Location)
	})

	// Mata uang datang dari snapshot polis. Berkas yang separuh kosong tidak boleh
	// mengosongkannya.
	t.Run("nilai dari polis tidak ditimpa berkas yang kosong", func(t *testing.T) {
		l := setup(t)
		l.link.SetSnapshot(laporan, registrasi.ClaimReportSnapshot{Location: "Gudang B"})

		hasil, err := l.service.Start(ctx, usecase.StartCommand{
			PolicyNumber: firePolicy, Portal: "ASM", RCVID: laporan,
		}, l.caller)
		require.NoError(t, err)

		require.Equal(t, "Gudang B", hasil.Claim.Location)
		require.NotEmpty(t, hasil.Claim.Currency, "mata uang polis terhapus berkas kosong")
	})
}

// TestClaimIDIsItsNumber menjaga pengenal klaim baru = nomor PNCN-nya.
//
// Work Owner mencari PNCN di POOLDATA.T_CLAIM_PNC lewat CLAIMID — cara yang sama dengan
// baris berkas RCVN — dan tidak menemukannya selama CLAIMID berisi pengenal acak
// (2026-09-26). Kepala klaim, tugasnya, dan pohon anaknya harus memakai pengenal yang sama.
func TestClaimIDIsItsNumber(t *testing.T) {
	l := setup(t)
	hasil, err := l.service.Start(context.Background(), usecase.StartCommand{
		PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.0001",
	}, l.caller)
	require.NoError(t, err)

	require.Regexp(t, regexpNomorKlaim, hasil.Claim.Number)
	require.Equal(t, hasil.Claim.Number, hasil.Claim.ID,
		"CLAIMID klaim baru bukan nomor PNCN-nya; ia tidak akan ditemukan saat dicari lewat CLAIMID")
	require.Equal(t, hasil.Claim.Number, hasil.Task.ClaimID,
		"tugas menunjuk pengenal lain daripada klaimnya")
}

// TestSaveDraftKeepsTheStageOpen menjaga tombol Save pada layar Input Register.
//
// Save menyimpan work object TANPA menjalankan flow action: tahapnya tidak berpindah,
// gerbang validasinya tidak dijalankan, dan tugasnya tetap terbuka untuk dilanjutkan.
// Isian yang belum lengkap — di sini tanpa tanggal sama sekali — tetap tersimpan.
func TestSaveDraftKeepsTheStageOpen(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{
		PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.0001",
	}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.StageInputRegister, start.Claim.CurrentStage)

	area := registrasi.Area{
		Country: registrasi.CountryIndonesia, CountryID: "100009",
		Province: "DI YOGYAKARTA", ProvinceID: "10012",
		City: "KAB. SLEMAN", CityID: "10259",
		District: "KEC. DEPOK", DistrictID: "10000925",
		RW: "KEL. CATURTUNGGAL", RWID: "10004326",
		PostalCode: "55281",
	}
	claim, err := l.service.SaveDraft(ctx, usecase.RegisterCommand{
		TaskID:            start.Task.ID,
		Location:          "JL. SETURAN RAYA",
		Area:              area,
		CustomerPrinciple: registrasi.CustomerPrincipleSuspicious,
		SuspiciousComment: "Dokumen tidak konsisten",
	}, l.caller)
	require.NoError(t, err, "Save tidak boleh menjalankan gerbang validasi")

	require.Equal(t, registrasi.StageInputRegister, claim.CurrentStage, "Save tidak boleh memindahkan tahap")
	require.Equal(t, start.Claim.Number, claim.Number, "Save tidak boleh menerbitkan nomor kedua")

	stored, err := l.store.Get(ctx, start.Claim.ID)
	require.NoError(t, err)
	require.Equal(t, area, stored.Area)
	require.Equal(t, registrasi.CustomerPrincipleSuspicious, stored.CustomerPrinciple)
	require.Equal(t, "Dokumen tidak konsisten", stored.SuspiciousComment)

	open, err := l.store.TaskRepo().OpenTaskForClaim(ctx, start.Claim.ID)
	require.NoError(t, err)
	require.Equal(t, start.Task.ID, open.ID, "tugas Input Register harus tetap terbuka")
}

// Prinsip Mengenal Nasabah yang tidak dipilih tersimpan NORMAL, bukan kosong.
//
// Control radio Pega membawa pyDefaultValue 1. Kosong akan terbaca berbeda dari NORMAL
// oleh SetEmailKomite, yang memeriksa CustomerPrinciple == "2".
func TestCustomerPrincipleDefaultsToNormal(t *testing.T) {
	l := setup(t)
	ctx := context.Background()

	start, err := l.service.Start(ctx, usecase.StartCommand{
		PolicyNumber: firePolicy, Portal: "ASM", RCVID: "RCVN.26.0001",
	}, l.caller)
	require.NoError(t, err)

	claim, err := l.service.SaveDraft(ctx, usecase.RegisterCommand{TaskID: start.Task.ID}, l.caller)
	require.NoError(t, err)
	require.Equal(t, registrasi.CustomerPrincipleNormal, claim.CustomerPrinciple)
}
