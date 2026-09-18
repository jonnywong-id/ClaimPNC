package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/cashier"
	"claim-pnc/internal/masterrekening/notification"
	"claim-pnc/internal/masterrekening/repo/memory"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/platform/clock"
)

// Seluruh uji di berkas ini berjalan tanpa basis data dan tanpa jaringan: penyimpanan
// di memori dan Kasir fake keduanya hidup di dalam proses.

func TestNewSubmissionAlwaysStartsPendingCommitteeDecision(t *testing.T) {
	build := assembly(t, "ASM")

	acct, err := build.service.Submit(context.Background(), completeSubmission(), submitter())
	require.NoError(t, err)

	assert.Equal(t, masterrekening.StatusPending, acct.Status,
		"tidak boleh ada jalan bagi submitter untuk menerbitkan rekening yang langsung disetujui")
	assert.Equal(t, "3171999", acct.CreatedBy)
	assert.Equal(t, build.clock.Now(), acct.CreatedAt)
}

func TestIncompleteSubmissionRejectedBeforeTouchingStorage(t *testing.T) {
	build := assembly(t, "ASM")

	p := completeSubmission()
	p.NIK = ""

	_, err := build.service.Submit(context.Background(), p, submitter())

	var validasi *masterrekening.ValidationError
	require.ErrorAs(t, err, &validasi)
	assert.Contains(t, validasi.Field, "nik")

	_, total, err := build.service.List(context.Background(), masterrekening.Filter{})
	require.NoError(t, err)
	assert.Zero(t, total, "pengajuan yang gagal validasi tidak boleh tersimpan")
}

func TestPendingAccountNumberCannotBeSubmittedAgain(t *testing.T) {
	build := assembly(t, "ASM")
	_, err := build.service.Submit(context.Background(), completeSubmission(), submitter())
	require.NoError(t, err)

	_, err = build.service.Submit(context.Background(), completeSubmission(), submitter())
	assert.ErrorIs(t, err, masterrekening.ErrAlreadyExists)
}

func TestAccountNumberRejectedByCommitteeCanBeResubmitted(t *testing.T) {
	// Perilaku sistem lama dipertahankan apa adanya (P-5): baris bekas penolakan
	// dibuang, lalu pengajuan baru disisipkan.
	build := assembly(t, "ASM")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	_, err = build.service.Decide(ctx, acct.KeyOf(), usecase.Decision{
		Status: masterrekening.StatusRejected,
		Note:   "Buku rekening tidak sesuai.",
	}, committee(), nil)
	require.NoError(t, err)

	ulang, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusPending, ulang.Status)

	_, total, err := build.service.List(ctx, masterrekening.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "baris bekas penolakan tidak boleh tertinggal sebagai baris kedua")
}

func TestCommitteeCannotApproveWithoutPassbook(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	p := completeSubmission()
	p.DocumentID = ""
	acct, err := build.service.Submit(ctx, p, submitter())
	require.NoError(t, err)

	_, err = build.service.Decide(ctx, acct.KeyOf(), usecase.Decision{
		Status: masterrekening.StatusApproved,
		Note:   "Disetujui atasan.",
	}, committee(), nil)

	var validasi *masterrekening.ValidationError
	require.ErrorAs(t, err, &validasi)
	assert.Contains(t, validasi.Field, "id_dokumen")

	stored, err := build.service.Get(ctx, acct.KeyOf())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusPending, stored.Status,
		"persetujuan yang ditolak validasi tidak boleh mengubah status")
}

func TestRejectionDoesNotRequirePassbook(t *testing.T) {
	// Account ditolak justru sering karena buktinya tidak ada; menuntut buku rekening
	// untuk menolak akan mengunci komite pada rekening yang justru ingin ditolaknya.
	build := assembly(t, "ASM")
	ctx := context.Background()

	p := completeSubmission()
	p.DocumentID = ""
	acct, err := build.service.Submit(ctx, p, submitter())
	require.NoError(t, err)

	rejected, err := build.service.Decide(ctx, acct.KeyOf(), usecase.Decision{
		Status: masterrekening.StatusRejected,
		Note:   "Buku rekening tidak dilampirkan.",
	}, committee(), nil)
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusRejected, rejected.Status)
}

func TestCommitteeDecisionCannotBeTakenTwice(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	_, err = build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	_, err = build.service.Decide(ctx, acct.KeyOf(), usecase.Decision{
		Status: masterrekening.StatusRejected,
		Note:   "Berubah pikiran.",
	}, committee(), nil)
	assert.ErrorIs(t, err, masterrekening.ErrAlreadyDecided)
}

func TestApprovedAccountIsRegisteredToCashier(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	result, err := build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	require.Len(t, build.cashier.Registered, 1)
	assert.Empty(t, build.cashier.Updated)
	assert.Equal(t, "BERHASIL", result.ServiceStatus)
	assert.Equal(t, "TIRUAN-0001", result.CashierAccountID)
	assert.Equal(t, "Rekening diterima sistem Kasir.", result.CashierResponse,
		"pesan Cashier dipangkas sampai setelah tanda ] seperti layar lama")
}

func TestReplacementAccountSentViaCashierUpdatePath(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	p := completeSubmission()
	p.PreviousNumber = "0987654321"
	p.PreviousBankCode = "008"
	acct, err := build.service.Submit(ctx, p, submitter())
	require.NoError(t, err)

	_, err = build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	assert.Empty(t, build.cashier.Registered)
	require.Len(t, build.cashier.Updated, 1)
}

func TestPortalsOutsideASMAndSIMASNETDoNotTouchCashier(t *testing.T) {
	// Prasyarat aslinya: TempGetApp.LSC_ID=="ASM" || TempGetApp.LSC_ID=="SIMASNET".
	build := assembly(t, "SAS")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	result, err := build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	assert.Empty(t, build.cashier.Registered)
	assert.Empty(t, result.ServiceStatus)
	assert.Equal(t, masterrekening.StatusApproved, result.Status,
		"portal tanpa Cashier tetap boleh menyetujui rekening")
}

func TestDeadCashierDoesNotVoidCommitteeDecision(t *testing.T) {
	// Keputusan komite adalah fakta bisnis yang sudah terjadi. Kegagalan jaringan
	// tidak boleh membuangnya dan memaksa komite memutuskan ulang.
	build := assembly(t, "ASM")
	build.cashier.Error = errors.New("koneksi terputus")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	result, err := build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusApproved, result.Status)
	assert.Equal(t, "GAGAL", result.ServiceStatus)

	stored, err := build.service.Get(ctx, acct.KeyOf())
	require.NoError(t, err)
	assert.Equal(t, masterrekening.StatusApproved, stored.Status)

	// TIDAK ada surel. Sistem lama hanya mengirim surel pada ResponseCode "9", dan
	// alurnya dipertahankan apa adanya — tidak ada pemicu baru yang ditambahkan.
	// Kegagalannya tetap terlihat: tercatat di log dan pada kolom Kasir di layar.
	assert.Empty(t, build.notifier.Sent,
		"kegagalan menghubungi Cashier tidak memicu surel di sistem lama")
}

func TestResponseCodeNineRaisesAlertToITTeam(t *testing.T) {
	// Meniru SendEmailAlertRekening: CekStatus.pxResults(1).ResponseCode == "9".
	build := assembly(t, "ASM")
	build.cashier.Response = masterrekening.CashierResult{
		Succeeded: false,
		Code:      "9",
		Message:   "[ERR-09] Rekening sudah terdaftar di Kasir.",
	}
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	result, err := build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	assert.Equal(t, "GAGAL", result.ServiceStatus)
	assert.Equal(t, "Rekening sudah terdaftar di Kasir.", result.CashierResponse)

	require.Len(t, build.notifier.Sent, 1)
	peringatan := build.notifier.Sent[0]
	assert.Equal(t, "9", peringatan.Code)
	assert.Equal(t, "1234567890", peringatan.Account.Number)
	// Committee yang memutuskan ikut disebut sebagai KETERANGAN, supaya Tim IT tahu
	// kepada siapa harus bertanya. Ia bukan penerima surelnya.
	assert.Equal(t, "Committee Contoh", peringatan.DecidedBy.Name)
}

func TestResponseCodeOneFailsWithoutRaisingAlert(t *testing.T) {
	// Kode "1" gagal tetapi tidak mengirim surel; hanya "9" yang memicunya.
	build := assembly(t, "ASM")
	build.cashier.Response = masterrekening.CashierResult{Succeeded: false, Code: "1", Message: "[ERR-01] Ditolak."}
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)

	result, err := build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	assert.Equal(t, "GAGAL", result.ServiceStatus)
	assert.Empty(t, build.notifier.Sent)
}

func TestDecidedAccountCannotBeChanged(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	acct, err := build.service.Submit(ctx, completeSubmission(), submitter())
	require.NoError(t, err)
	_, err = build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	p := completeSubmission()
	p.OwnerName = "NAMA LAIN"
	_, err = build.service.Update(ctx, acct.KeyOf(), p, submitter())
	assert.ErrorIs(t, err, masterrekening.ErrAlreadyDecided)
}

func TestStatusFilterSeparatesTheFourScreenTabs(t *testing.T) {
	build := assembly(t, "ASM")
	ctx := context.Background()

	menunggu := completeSubmission()
	_, err := build.service.Submit(ctx, menunggu, submitter())
	require.NoError(t, err)

	approved := completeSubmission()
	approved.Number = "2222222222"
	acct, err := build.service.Submit(ctx, approved, submitter())
	require.NoError(t, err)
	_, err = build.service.Decide(ctx, acct.KeyOf(), approve(), committee(), nil)
	require.NoError(t, err)

	_, pendingCount, err := build.service.List(ctx, masterrekening.Filter{Status: masterrekening.StatusPending})
	require.NoError(t, err)
	assert.Equal(t, 1, pendingCount)

	_, approvedCount, err := build.service.List(ctx, masterrekening.Filter{Status: masterrekening.StatusApproved})
	require.NoError(t, err)
	assert.Equal(t, 1, approvedCount)

	_, rejectedCount, err := build.service.List(ctx, masterrekening.Filter{Status: masterrekening.StatusRejected})
	require.NoError(t, err)
	assert.Zero(t, rejectedCount)
}

// --- bahan uji ---

type deps struct {
	service  *usecase.Service
	cashier  *cashier.Fake
	notifier *notification.Fake
	clock    *clock.Fixed
}

func assembly(t *testing.T, portal string) deps {
	t.Helper()

	jam := clock.FixedAt(time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC))
	fake := cashier.NewFake()
	notifier := &notification.Fake{}

	return deps{
		service: usecase.NewService(usecase.Options{
			Repo:             memory.NewRepo(),
			Bank:             memory.NewBankRepo(memory.SampleBanks()...),
			Cashier:          fake,
			Notifier:         notifier,
			Clock:            jam,
			PortalAlias:      portal,
			DefaultCommittee: "KOMITE-01",
		}),
		cashier:  fake,
		notifier: notifier,
		clock:    jam,
	}
}

func completeSubmission() usecase.Submission {
	return usecase.Submission{
		Number:      "1234567890",
		OwnerName:   "BENGKEL CONTOH SEJAHTERA",
		BankName:    "BANK CONTOH",
		BankBranch:  "JAKARTA PUSAT",
		BankAddress: "JL. CONTOH NO. 1",
		BankCode:    "014",
		AccountType: "BIASA",
		Email:       "keuangan@contoh.co.id",
		Phone:       "0211234567",
		NIK:         "3171000000000000",
		DocumentID:  "DOK-001",
		Active:      true,
	}
}

func submitter() usecase.Submitter {
	return usecase.Submitter{Identity: "3171999", Name: "Petugas Contoh", Email: "petugas@sinarmas.id"}
}

func committee() usecase.Committee {
	return usecase.Committee{
		Identity: "KOMITE-01",
		Name:     "Committee Contoh",
		Email:    "komite@sinarmas.id",
	}
}

func approve() usecase.Decision {
	return usecase.Decision{
		Status: masterrekening.StatusApproved,
		Note:   "Disetujui atasan, buku rekening sesuai.",
	}
}
