package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterrekening"
	"claim-pnc/internal/masterrekening/cashier"
	"claim-pnc/internal/masterrekening/notification"
	"claim-pnc/internal/masterrekening/repo/memory"
	"claim-pnc/internal/masterrekening/usecase"
	"claim-pnc/internal/platform/clock"
)

var errStore = errors.New("penyimpanan rusak")

// failingRepo meneruskan ke penyimpanan memori, kecuali operasi yang ditandai gagal.
// updateFailFrom > 0 berarti pemanggilan Update ke-n dan seterusnya gagal.
type failingRepo struct {
	*memory.Repo
	failList, failFind, failClear, failSave bool
	updateFailFrom                          int
	updates                                 int
}

func (f *failingRepo) List(ctx context.Context, x masterrekening.Filter) ([]masterrekening.Account, int, error) {
	if f.failList {
		return nil, 0, errStore
	}
	return f.Repo.List(ctx, x)
}

func (f *failingRepo) FindByNumber(ctx context.Context, n string) ([]masterrekening.Account, error) {
	if f.failFind {
		return nil, errStore
	}
	return f.Repo.FindByNumber(ctx, n)
}

func (f *failingRepo) ClearRejected(ctx context.Context, k masterrekening.Key) error {
	if f.failClear {
		return errStore
	}
	return f.Repo.ClearRejected(ctx, k)
}

func (f *failingRepo) Save(ctx context.Context, r masterrekening.Account) error {
	if f.failSave {
		return errStore
	}
	return f.Repo.Save(ctx, r)
}

func (f *failingRepo) Update(ctx context.Context, r masterrekening.Account) error {
	f.updates++
	if f.updateFailFrom > 0 && f.updates >= f.updateFailFrom {
		return errStore
	}
	return f.Repo.Update(ctx, r)
}

// failingBank selalu gagal membaca daftar bank.
type failingBank struct{}

func (failingBank) List(context.Context) ([]masterrekening.Bank, error) { return nil, errStore }

func serviceWith(repo masterrekening.Repo, bank masterrekening.BankRepo,
	cash masterrekening.Cashier, notifier masterrekening.Notifier,
) *usecase.Service {
	return usecase.NewService(usecase.Options{
		Repo: repo, Bank: bank, Cashier: cash, Notifier: notifier,
		Clock:            clock.FixedAt(time.Date(2026, 9, 17, 3, 0, 0, 0, time.UTC)),
		PortalAlias:      " asm ",
		DefaultCommittee: "KOMITE-01",
	})
}

func pending() masterrekening.Account {
	return masterrekening.Account{
		Number: "1234567890", OwnerName: "BENGKEL", BankName: "BANK", BankBranch: "CAB",
		BankAddress: "JL", BankCode: "014", AccountType: "BIASA", Email: "a@contoh.co.id",
		NIK: "317", Phone: "0211234567", SubmitterEmail: "pengaju@contoh.co.id",
		DocumentID: "DOK", Note: "lama", Status: masterrekening.StatusPending,
	}
}

func TestSubmitWrapsStorageFailures(t *testing.T) {
	ctx := context.Background()

	repo := &failingRepo{Repo: memory.NewRepo(), failFind: true}
	_, err := serviceWith(repo, nil, nil, nil).Submit(ctx, completeSubmission(), submitter())
	require.ErrorIs(t, err, errStore)
	require.ErrorContains(t, err, "memeriksa duplikasi")

	rejected := pending()
	rejected.Status = masterrekening.StatusRejected
	repo = &failingRepo{Repo: memory.NewRepo(rejected), failClear: true}
	_, err = serviceWith(repo, nil, nil, nil).Submit(ctx, completeSubmission(), submitter())
	require.ErrorContains(t, err, "membuang pengajuan yang ditolak")

	repo = &failingRepo{Repo: memory.NewRepo(), failSave: true}
	_, err = serviceWith(repo, nil, nil, nil).Submit(ctx, completeSubmission(), submitter())
	require.ErrorContains(t, err, "menyimpan rekening")
}

func TestSubmitKeepsAnApprovedSiblingUntouched(t *testing.T) {
	// Satu nomor yang ditolak di bank lain dibuang; yang lain dibiarkan. Hanya baris
	// pertama (urut kode bank) yang menentukan boleh tidaknya diajukan ulang.
	rejected := pending()
	rejected.BankCode = "002"
	rejected.Status = masterrekening.StatusRejected
	approved := pending()
	approved.BankCode = "009"
	approved.Status = masterrekening.StatusApproved

	repo := memory.NewRepo(rejected, approved)
	service := serviceWith(repo, nil, nil, nil)

	_, err := service.Submit(context.Background(), completeSubmission(), submitter())
	require.NoError(t, err)

	_, err = repo.Get(context.Background(), masterrekening.Key{Number: "1234567890", BankCode: "002"})
	require.ErrorIs(t, err, masterrekening.ErrNotFound, "pengajuan yang ditolak dibuang")
	kept, err := repo.Get(context.Background(), masterrekening.Key{Number: "1234567890", BankCode: "009"})
	require.NoError(t, err)
	require.Equal(t, masterrekening.StatusApproved, kept.Status)
}

func TestUpdateRewritesAPendingAccount(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewRepo(pending())
	service := serviceWith(repo, nil, nil, nil)
	key := pending().KeyOf()

	change := completeSubmission()
	change.OwnerName = " BENGKEL BARU "
	change.DocumentID = ""
	change.Note = " "
	updated, err := service.Update(ctx, key, change, submitter())
	require.NoError(t, err)
	require.Equal(t, "BENGKEL BARU", updated.OwnerName)
	require.Equal(t, "DOK", updated.DocumentID, "dokumen kosong tidak menimpa yang lama")
	require.Equal(t, "lama", updated.Note, "catatan kosong tidak menimpa yang lama")
	require.Equal(t, "3171999", updated.UpdatedBy)

	change.DocumentID = "DOK-2"
	change.Note = "baru"
	updated, err = service.Update(ctx, key, change, submitter())
	require.NoError(t, err)
	require.Equal(t, "DOK-2", updated.DocumentID)
	require.Equal(t, "baru", updated.Note)

	stored, _ := repo.Get(ctx, key)
	require.Equal(t, "DOK-2", stored.DocumentID)

	change.NIK = ""
	_, err = service.Update(ctx, key, change, submitter())
	var validation *masterrekening.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Field, "nik")
}

func TestUpdateRefusals(t *testing.T) {
	ctx := context.Background()

	_, err := serviceWith(memory.NewRepo(), nil, nil, nil).
		Update(ctx, masterrekening.Key{Number: "x"}, completeSubmission(), submitter())
	require.ErrorIs(t, err, masterrekening.ErrNotFound)

	repo := &failingRepo{Repo: memory.NewRepo(pending()), updateFailFrom: 1}
	_, err = serviceWith(repo, nil, nil, nil).
		Update(ctx, pending().KeyOf(), completeSubmission(), submitter())
	require.ErrorContains(t, err, "memperbarui rekening")
}

func TestListAndBanksWrapFailures(t *testing.T) {
	ctx := context.Background()

	_, _, err := serviceWith(&failingRepo{Repo: memory.NewRepo(), failList: true}, nil, nil, nil).
		List(ctx, masterrekening.Filter{})
	require.ErrorContains(t, err, "membaca daftar rekening")

	_, err = serviceWith(memory.NewRepo(), nil, nil, nil).ListBanks(ctx)
	require.ErrorContains(t, err, "sumber daftar bank belum dipasang")

	_, err = serviceWith(memory.NewRepo(), failingBank{}, nil, nil).ListBanks(ctx)
	require.ErrorIs(t, err, errStore)

	banks, err := serviceWith(memory.NewRepo(), memory.NewBankRepo(memory.SampleBanks()...),
		nil, nil).ListBanks(ctx)
	require.NoError(t, err)
	require.Equal(t, memory.SampleBanks(), banks)
}

func TestDecideRefusalsAndStorageFailure(t *testing.T) {
	ctx := context.Background()
	approve := usecase.Decision{Status: masterrekening.StatusApproved, Note: "ok"}

	_, err := serviceWith(memory.NewRepo(), nil, nil, nil).
		Decide(ctx, masterrekening.Key{Number: "x"}, approve, committee(), nil)
	require.ErrorIs(t, err, masterrekening.ErrNotFound)

	repo := &failingRepo{Repo: memory.NewRepo(pending()), updateFailFrom: 1}
	_, err = serviceWith(repo, nil, nil, nil).Decide(ctx, pending().KeyOf(), approve, committee(), nil)
	require.ErrorContains(t, err, "menyimpan keputusan komite")
}

func TestDecideWithoutACashierKeepsTheApproval(t *testing.T) {
	result, err := serviceWith(memory.NewRepo(pending()), nil, nil, nil).Decide(
		context.Background(), pending().KeyOf(),
		usecase.Decision{Status: masterrekening.StatusApproved, DocumentID: " DOK-9 "},
		committee(), nil)
	require.NoError(t, err)
	require.Equal(t, masterrekening.StatusApproved, result.Status)
	require.Equal(t, "DOK-9", result.DocumentID)
	require.Equal(t, "lama", result.Note, "catatan kosong tidak menimpa")
	require.Empty(t, result.ServiceStatus, "tanpa Kasir tidak ada jejak layanan")
}

func TestCashierFailuresAreLogged(t *testing.T) {
	ctx := context.Background()
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	approve := usecase.Decision{Status: masterrekening.StatusApproved, Note: "ok"}

	// Kasir tidak dapat dihubungi: dicatat, lalu jejaknya gagal disimpan.
	fake := cashier.NewFake()
	fake.Error = errors.New("koneksi terputus")
	repo := &failingRepo{Repo: memory.NewRepo(pending()), updateFailFrom: 2}
	result, err := serviceWith(repo, nil, fake, nil).Decide(ctx, pending().KeyOf(), approve,
		committee(), logger)
	require.NoError(t, err)
	require.Equal(t, "GAGAL", result.ServiceStatus)
	require.Equal(t, "Sistem Cashier tidak dapat dihubungi.", result.CashierResponse)
	require.Contains(t, logs.String(), "pendaftaran rekening ke Kasir gagal")
	require.Contains(t, logs.String(), "gagal menyimpan jejak pendaftaran Cashier")

	// Kode 9 dengan peringatan yang gagal terkirim.
	logs.Reset()
	fake = cashier.NewFake()
	fake.Response = masterrekening.CashierResult{Code: "9", Message: "[E] ganda", AccountID: "ID-9"}
	notifier := &notification.Fake{Error: errors.New("smtp mati")}
	result, err = serviceWith(memory.NewRepo(pending()), nil, fake, notifier).
		Decide(ctx, pending().KeyOf(), approve, committee(), logger)
	require.NoError(t, err)
	require.Equal(t, "ID-9", result.CashierAccountID)
	require.Equal(t, 1, notifier.Count())
	require.Contains(t, logs.String(), "gagal mengirim peringatan kegagalan Cashier")

	// Kode 9 tanpa pemberi tahu: hanya dicatat.
	logs.Reset()
	result, err = serviceWith(memory.NewRepo(pending()), nil, fake, nil).
		Decide(ctx, pending().KeyOf(), approve, committee(), logger)
	require.NoError(t, err)
	require.Equal(t, "GAGAL", result.ServiceStatus)
	require.Contains(t, logs.String(), "pendaftaran rekening ke Kasir gagal")
}
