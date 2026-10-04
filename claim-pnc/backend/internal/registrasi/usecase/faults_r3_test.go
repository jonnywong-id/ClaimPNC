package usecase_test

import (
	"claim-pnc/internal/registrasi/acceptancenotepdf"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
	"claim-pnc/internal/registrasi/dlapdf"
	"claim-pnc/internal/registrasi/facesheetpdf"
	"claim-pnc/internal/registrasi/lodpdf"
	"claim-pnc/internal/registrasi/plapdf"
	"claim-pnc/internal/registrasi/repo/memory"
	"claim-pnc/internal/registrasi/usecase"
)

// Berkas ini membungkus setiap seam layanan dengan penyuntik galat. Uji propagasi galat
// memakainya untuk memastikan setiap kegagalan seam diteruskan ke pemanggil — bukan
// ditelan — tanpa menulis satu uji tangan per cabang.

var errFault = errors.New("galat suntikan uji")

// faults menyimpan galat yang dipasang per seam: galat menyala sekali pada panggilan ke
// skip+1 setelah dipasang.
type faults struct {
	mu    sync.Mutex
	armed map[string]int
	fired bool
}

func (f *faults) arm(key string, skip int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed = map[string]int{key: skip}
	f.fired = false
}

func (f *faults) disarm() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armed = nil
}

func (f *faults) hit(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	skip, ok := f.armed[key]
	if !ok {
		return nil
	}
	if skip > 0 {
		f.armed[key] = skip - 1
		return nil
	}
	delete(f.armed, key)
	f.fired = true
	return errFault
}

func (f *faults) didFire() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fired
}

// ── Pembungkus seam ──────────────────────────────────────────────────────────────

type fClaimRepo struct {
	registrasi.ClaimRepo
	f *faults
}

func (w fClaimRepo) Save(ctx context.Context, k registrasi.Claim) error {
	if err := w.f.hit("Claim.Save"); err != nil {
		return err
	}
	return w.ClaimRepo.Save(ctx, k)
}

func (w fClaimRepo) Get(ctx context.Context, id string) (registrasi.Claim, error) {
	if err := w.f.hit("Claim.Get"); err != nil {
		return registrasi.Claim{}, err
	}
	return w.ClaimRepo.Get(ctx, id)
}

func (w fClaimRepo) GetByNumber(ctx context.Context, n string) (registrasi.Claim, error) {
	if err := w.f.hit("Claim.GetByNumber"); err != nil {
		return registrasi.Claim{}, err
	}
	return w.ClaimRepo.GetByNumber(ctx, n)
}

func (w fClaimRepo) FindDuplicates(ctx context.Context, key []registrasi.DuplicateKey, except string) ([]registrasi.DuplicateClaim, error) {
	if err := w.f.hit("Claim.FindDuplicates"); err != nil {
		return nil, err
	}
	return w.ClaimRepo.FindDuplicates(ctx, key, except)
}

type fTaskRepo struct {
	registrasi.TaskRepo
	f *faults
}

func (w fTaskRepo) Save(ctx context.Context, t registrasi.Task) error {
	if err := w.f.hit("Task.Save"); err != nil {
		return err
	}
	return w.TaskRepo.Save(ctx, t)
}

func (w fTaskRepo) Get(ctx context.Context, id string) (registrasi.Task, error) {
	if err := w.f.hit("Task.Get"); err != nil {
		return registrasi.Task{}, err
	}
	return w.TaskRepo.Get(ctx, id)
}

func (w fTaskRepo) OpenTaskForClaim(ctx context.Context, id string) (registrasi.Task, error) {
	if err := w.f.hit("Task.OpenTaskForClaim"); err != nil {
		return registrasi.Task{}, err
	}
	return w.TaskRepo.OpenTaskForClaim(ctx, id)
}

func (w fTaskRepo) Inbox(ctx context.Context, op string, wb, st []string) ([]registrasi.Task, error) {
	if err := w.f.hit("Task.Inbox"); err != nil {
		return nil, err
	}
	return w.TaskRepo.Inbox(ctx, op, wb, st)
}

type fInbox struct {
	registrasi.InboxMirror
	f *faults
}

func (w fInbox) Mirror(ctx context.Context, e registrasi.InboxEntry) error {
	if err := w.f.hit("Inbox.Mirror"); err != nil {
		return err
	}
	return w.InboxMirror.Mirror(ctx, e)
}

type fAudit struct {
	registrasi.AuditRecorder
	f *faults
}

func (w fAudit) Record(ctx context.Context, j registrasi.AuditTrail) error {
	if err := w.f.hit("Audit.Record"); err != nil {
		return err
	}
	return w.AuditRecorder.Record(ctx, j)
}

type fNotifier struct {
	registrasi.Notifier
	f *faults
}

func (w fNotifier) Send(ctx context.Context, m registrasi.Notification) error {
	if err := w.f.hit("Notifier.Send"); err != nil {
		return err
	}
	return w.Notifier.Send(ctx, m)
}

type fUnit struct {
	registrasi.UnitOfWork
	f *faults
}

func (w fUnit) Run(ctx context.Context, work func(context.Context) error) error {
	if err := w.f.hit("Unit.Run"); err != nil {
		return err
	}
	return w.UnitOfWork.Run(ctx, work)
}

type fDLA struct {
	registrasi.DLASource
	f *faults
}

func (w fDLA) Issued(ctx context.Context, c, o string, cs, as int) ([]registrasi.DLA, error) {
	if err := w.f.hit("DLA.Issued"); err != nil {
		return nil, err
	}
	return w.DLASource.Issued(ctx, c, o, cs, as)
}

func (w fDLA) PreDLA(ctx context.Context, c, o string, cs, as int) ([]registrasi.DLA, error) {
	if err := w.f.hit("DLA.PreDLA"); err != nil {
		return nil, err
	}
	return w.DLASource.PreDLA(ctx, c, o, cs, as)
}

func (w fDLA) Previous(ctx context.Context, c, code string) (registrasi.DLAPrevious, bool, error) {
	if err := w.f.hit("DLA.Previous"); err != nil {
		return registrasi.DLAPrevious{}, false, err
	}
	return w.DLASource.Previous(ctx, c, code)
}

func (w fDLA) Policy(ctx context.Context, n, prodKe string) (registrasi.DLAPolicy, error) {
	if err := w.f.hit("DLA.Policy"); err != nil {
		return registrasi.DLAPolicy{}, err
	}
	return w.DLASource.Policy(ctx, n, prodKe)
}

func (w fDLA) ReinsuranceCase(ctx context.Context) (map[string]string, error) {
	if err := w.f.hit("DLA.ReinsuranceCase"); err != nil {
		return nil, err
	}
	return w.DLASource.ReinsuranceCase(ctx)
}

func (w fDLA) Treaty(ctx context.Context, b string, y int, tt string) (registrasi.TreatyArrangement, error) {
	if err := w.f.hit("DLA.Treaty"); err != nil {
		return registrasi.TreatyArrangement{}, err
	}
	return w.DLASource.Treaty(ctx, b, y, tt)
}

func (w fDLA) NextNumber(ctx context.Context, code string, year int) (string, error) {
	if err := w.f.hit("DLA.NextNumber"); err != nil {
		return "", err
	}
	return w.DLASource.NextNumber(ctx, code, year)
}

func (w fDLA) Save(ctx context.Context, d registrasi.DLA) error {
	if err := w.f.hit("DLA.Save"); err != nil {
		return err
	}
	return w.DLASource.Save(ctx, d)
}

func (w fDLA) MarkPrinted(ctx context.Context, c, n, note string) error {
	if err := w.f.hit("DLA.MarkPrinted"); err != nil {
		return err
	}
	return w.DLASource.MarkPrinted(ctx, c, n, note)
}

type fPLA struct {
	registrasi.PLASource
	f *faults
}

func (w fPLA) CoinsMembers(ctx context.Context, p, prodKe string) ([]registrasi.PLACoinsMember, error) {
	if err := w.f.hit("PLA.CoinsMembers"); err != nil {
		return nil, err
	}
	return w.PLASource.CoinsMembers(ctx, p, prodKe)
}

func (w fPLA) Recipient(ctx context.Context, code, name string) (registrasi.PLARecipientInfo, error) {
	if err := w.f.hit("PLA.Recipient"); err != nil {
		return registrasi.PLARecipientInfo{}, err
	}
	return w.PLASource.Recipient(ctx, code, name)
}

func (w fPLA) Previous(ctx context.Context, c, code string) (registrasi.PLAPrevious, bool, error) {
	if err := w.f.hit("PLA.Previous"); err != nil {
		return registrasi.PLAPrevious{}, false, err
	}
	return w.PLASource.Previous(ctx, c, code)
}

func (w fPLA) Issued(ctx context.Context, c, o string, cs, rev int) ([]registrasi.PLA, error) {
	if err := w.f.hit("PLA.Issued"); err != nil {
		return nil, err
	}
	return w.PLASource.Issued(ctx, c, o, cs, rev)
}

func (w fPLA) NextNumber(ctx context.Context, code string, year int) (string, error) {
	if err := w.f.hit("PLA.NextNumber"); err != nil {
		return "", err
	}
	return w.PLASource.NextNumber(ctx, code, year)
}

func (w fPLA) Save(ctx context.Context, p registrasi.PLA) error {
	if err := w.f.hit("PLA.Save"); err != nil {
		return err
	}
	return w.PLASource.Save(ctx, p)
}

func (w fPLA) UpdateNote(ctx context.Context, c, n string, rev int, note string) error {
	if err := w.f.hit("PLA.UpdateNote"); err != nil {
		return err
	}
	return w.PLASource.UpdateNote(ctx, c, n, rev, note)
}

func (w fPLA) Signature(ctx context.Context, e string) (string, []byte, error) {
	if err := w.f.hit("PLA.Signature"); err != nil {
		return "", nil, err
	}
	return w.PLASource.Signature(ctx, e)
}

func (w fPLA) LODEmails(ctx context.Context, n, pic string) (string, string, error) {
	if err := w.f.hit("PLA.LODEmails"); err != nil {
		return "", "", err
	}
	return w.PLASource.LODEmails(ctx, n, pic)
}

type fAcceptance struct {
	registrasi.AcceptanceSource
	f *faults
}

func (w fAcceptance) NextNumber(ctx context.Context, year int) (string, error) {
	if err := w.f.hit("Acc.NextNumber"); err != nil {
		return "", err
	}
	return w.AcceptanceSource.NextNumber(ctx, year)
}

func (w fAcceptance) Save(ctx context.Context, c, o string, cs, as int, line registrasi.SettlementLine) error {
	if err := w.f.hit("Acc.Save"); err != nil {
		return err
	}
	return w.AcceptanceSource.Save(ctx, c, o, cs, as, line)
}

func (w fAcceptance) RecordLODPrint(ctx context.Context, c, o string, cs, as int, at time.Time, typ string) error {
	if err := w.f.hit("Acc.RecordLODPrint"); err != nil {
		return err
	}
	return w.AcceptanceSource.RecordLODPrint(ctx, c, o, cs, as, at, typ)
}

func (w fAcceptance) Fields(ctx context.Context, c, o string, cs, as int, line *registrasi.SettlementLine) error {
	if err := w.f.hit("Acc.Fields"); err != nil {
		return err
	}
	return w.AcceptanceSource.Fields(ctx, c, o, cs, as, line)
}

func (w fAcceptance) OtherDLA(ctx context.Context, c, o string, cs, as int) ([]registrasi.AcceptanceDLAState, error) {
	if err := w.f.hit("Acc.OtherDLA"); err != nil {
		return nil, err
	}
	return w.AcceptanceSource.OtherDLA(ctx, c, o, cs, as)
}

func (w fAcceptance) AddHistory(ctx context.Context, c, note, user string, at time.Time) error {
	if err := w.f.hit("Acc.AddHistory"); err != nil {
		return err
	}
	return w.AcceptanceSource.AddHistory(ctx, c, note, user, at)
}

func (w fAcceptance) OpenPosition(ctx context.Context, n, pos string) (string, error) {
	if err := w.f.hit("Acc.OpenPosition"); err != nil {
		return "", err
	}
	return w.AcceptanceSource.OpenPosition(ctx, n, pos)
}

func (w fAcceptance) StartProgress(ctx context.Context, p registrasi.ProgressStart) (string, error) {
	if err := w.f.hit("Acc.StartProgress"); err != nil {
		return "", err
	}
	return w.AcceptanceSource.StartProgress(ctx, p)
}

func (w fAcceptance) AddProgress(ctx context.Context, p registrasi.ProgressUpdate) error {
	if err := w.f.hit("Acc.AddProgress"); err != nil {
		return err
	}
	return w.AcceptanceSource.AddProgress(ctx, p)
}

func (w fAcceptance) PolicyCaseID(ctx context.Context, p, prodKe string) (string, error) {
	if err := w.f.hit("Acc.PolicyCaseID"); err != nil {
		return "", err
	}
	return w.AcceptanceSource.PolicyCaseID(ctx, p, prodKe)
}

func (w fAcceptance) OpenProtectionApproved(ctx context.Context, p, c, n string) (bool, error) {
	if err := w.f.hit("Acc.OpenProtectionApproved"); err != nil {
		return false, err
	}
	return w.AcceptanceSource.OpenProtectionApproved(ctx, p, c, n)
}

func (w fAcceptance) TravelClientName(ctx context.Context, p string) (string, error) {
	if err := w.f.hit("Acc.TravelClientName"); err != nil {
		return "", err
	}
	return w.AcceptanceSource.TravelClientName(ctx, p)
}

type fCommittees struct {
	registrasi.CommitteeStore
	f *faults
}

func (w fCommittees) NextCaseID(ctx context.Context, at time.Time) (string, error) {
	if err := w.f.hit("Committee.NextCaseID"); err != nil {
		return "", err
	}
	return w.CommitteeStore.NextCaseID(ctx, at)
}

func (w fCommittees) Save(ctx context.Context, c registrasi.CommitteeCase) error {
	if err := w.f.hit("Committee.Save"); err != nil {
		return err
	}
	return w.CommitteeStore.Save(ctx, c)
}

func (w fCommittees) Get(ctx context.Context, id string) (registrasi.CommitteeCase, error) {
	if err := w.f.hit("Committee.Get"); err != nil {
		return registrasi.CommitteeCase{}, err
	}
	return w.CommitteeStore.Get(ctx, id)
}

func (w fCommittees) Pending(ctx context.Context, op string) ([]registrasi.CommitteeMember, error) {
	if err := w.f.hit("Committee.Pending"); err != nil {
		return nil, err
	}
	return w.CommitteeStore.Pending(ctx, op)
}

type fTiering struct {
	registrasi.CommitteeTiering
	f *faults
}

func (w fTiering) Route(ctx context.Context, line string, v registrasi.Money, app string) (registrasi.CommitteeRoute, error) {
	if err := w.f.hit("Tiering.Route"); err != nil {
		return registrasi.CommitteeRoute{}, err
	}
	return w.CommitteeTiering.Route(ctx, line, v, app)
}

type fRecords struct {
	*memory.ClaimRecords
	f *faults
}

func (w fRecords) Surveys(ctx context.Context, k registrasi.RecordKeys) ([]registrasi.Survey, error) {
	if err := w.f.hit("Records.Surveys"); err != nil {
		return nil, err
	}
	return w.ClaimRecords.Surveys(ctx, k)
}

func (w fRecords) DocumentTypes(ctx context.Context, code string) ([]registrasi.DocumentType, error) {
	if err := w.f.hit("Records.DocumentTypes"); err != nil {
		return nil, err
	}
	return w.ClaimRecords.DocumentTypes(ctx, code)
}

func (w fRecords) Attachments(ctx context.Context, k registrasi.RecordKeys) ([]registrasi.Attachment, error) {
	if err := w.f.hit("Records.Attachments"); err != nil {
		return nil, err
	}
	return w.ClaimRecords.Attachments(ctx, k)
}

func (w fRecords) Progress(ctx context.Context, k registrasi.RecordKeys) ([]registrasi.ProgressEntry, error) {
	if err := w.f.hit("Records.Progress"); err != nil {
		return nil, err
	}
	return w.ClaimRecords.Progress(ctx, k)
}

func (w fRecords) Communications(ctx context.Context, k registrasi.RecordKeys) ([]registrasi.Communication, error) {
	if err := w.f.hit("Records.Communications"); err != nil {
		return nil, err
	}
	return w.ClaimRecords.Communications(ctx, k)
}

func (w fRecords) AddAttachment(ctx context.Context, a registrasi.NewAttachment) error {
	if err := w.f.hit("Records.AddAttachment"); err != nil {
		return err
	}
	return w.ClaimRecords.AddAttachment(ctx, a)
}

type fFaceSheet struct {
	registrasi.FaceSheetSource
	f *faults
}

func (w fFaceSheet) CauseOfLossName(ctx context.Context, id string) (string, error) {
	if err := w.f.hit("FS.CauseOfLossName"); err != nil {
		return "", err
	}
	return w.FaceSheetSource.CauseOfLossName(ctx, id)
}

func (w fFaceSheet) OperatorName(ctx context.Context, id string) (string, error) {
	if err := w.f.hit("FS.OperatorName"); err != nil {
		return "", err
	}
	return w.FaceSheetSource.OperatorName(ctx, id)
}

func (w fFaceSheet) Coinsurance(ctx context.Context, p, prodKe string) ([]registrasi.CoinsuranceRow, error) {
	if err := w.f.hit("FS.Coinsurance"); err != nil {
		return nil, err
	}
	return w.FaceSheetSource.Coinsurance(ctx, p, prodKe)
}

func (w fFaceSheet) FacReinsurers(ctx context.Context, p, prodKe string) ([]registrasi.FacReinsurer, error) {
	if err := w.f.hit("FS.FacReinsurers"); err != nil {
		return nil, err
	}
	return w.FaceSheetSource.FacReinsurers(ctx, p, prodKe)
}

func (w fFaceSheet) LastRevision(ctx context.Context, c, o string, cs int) (int, bool, error) {
	if err := w.f.hit("FS.LastRevision"); err != nil {
		return 0, false, err
	}
	return w.FaceSheetSource.LastRevision(ctx, c, o, cs)
}

func (w fFaceSheet) SaveRevision(ctx context.Context, r registrasi.FaceSheetRevision) error {
	if err := w.f.hit("FS.SaveRevision"); err != nil {
		return err
	}
	return w.FaceSheetSource.SaveRevision(ctx, r)
}

type fFSRender struct {
	registrasi.FaceSheetRenderer
	f *faults
}

func (w fFSRender) Render(d registrasi.FaceSheet) ([]byte, error) {
	if err := w.f.hit("Render.FaceSheet"); err != nil {
		return nil, err
	}
	return w.FaceSheetRenderer.Render(d)
}

type fPLARender struct {
	registrasi.PLARenderer
	f *faults
}

func (w fPLARender) Render(d registrasi.PLADocument) ([]byte, error) {
	if err := w.f.hit("Render.PLA"); err != nil {
		return nil, err
	}
	return w.PLARenderer.Render(d)
}

type fDLARender struct {
	registrasi.DLARenderer
	f *faults
}

func (w fDLARender) Render(d registrasi.DLADocument) ([]byte, error) {
	if err := w.f.hit("Render.DLA"); err != nil {
		return nil, err
	}
	return w.DLARenderer.Render(d)
}

type fLODRender struct {
	registrasi.LODRenderer
	f *faults
}

func (w fLODRender) Render(d registrasi.LODDocument) ([]byte, error) {
	if err := w.f.hit("Render.LOD"); err != nil {
		return nil, err
	}
	return w.LODRenderer.Render(d)
}

type fCashier struct {
	*memory.Cashier
	f *faults
}

func (w fCashier) BankGroupID(ctx context.Context, name, id string) (string, bool, error) {
	if err := w.f.hit("Cashier.BankGroupID"); err != nil {
		return "", false, err
	}
	return w.Cashier.BankGroupID(ctx, name, id)
}

func (w fCashier) Log(ctx context.Context, e registrasi.CashierLog) error {
	if err := w.f.hit("Cashier.Log"); err != nil {
		return err
	}
	return w.Cashier.Log(ctx, e)
}

func (w fCashier) MarkTransferred(ctx context.Context, c, o string, cs, as int, at time.Time, id string) error {
	if err := w.f.hit("Cashier.MarkTransferred"); err != nil {
		return err
	}
	return w.Cashier.MarkTransferred(ctx, c, o, cs, as, at, id)
}

type fCurrency struct {
	registrasi.CurrencyDirectory
	f *faults
}

func (w fCurrency) Currencies(ctx context.Context) ([]registrasi.CurrencyOption, error) {
	if err := w.f.hit("Currency.Currencies"); err != nil {
		return nil, err
	}
	return w.CurrencyDirectory.Currencies(ctx)
}

type fPolicyItems struct {
	*memory.PolicyItems
	f *faults
}

func (w fPolicyItems) Items(ctx context.Context, p registrasi.Policy) ([]registrasi.SourceItem, error) {
	if err := w.f.hit("Items.Items"); err != nil {
		return nil, err
	}
	return w.PolicyItems.Items(ctx, p)
}

func (w fPolicyItems) ItemOptions(ctx context.Context, p registrasi.Policy, o string) ([]registrasi.ItemOption, error) {
	if err := w.f.hit("Items.ItemOptions"); err != nil {
		return nil, err
	}
	return w.PolicyItems.ItemOptions(ctx, p, o)
}

type fArea struct {
	registrasi.AreaDirectory
	f *faults
}

func (w fArea) Options(ctx context.Context, l registrasi.AreaLevel, parent string) ([]registrasi.AreaOption, error) {
	if err := w.f.hit("Area.Options"); err != nil {
		return nil, err
	}
	return w.AreaDirectory.Options(ctx, l, parent)
}

type fUploader struct {
	*memory.DocumentUploader
	f *faults
}

func (w fUploader) Upload(ctx context.Context, d registrasi.DocumentFile) (string, error) {
	if err := w.f.hit("Upload.Upload"); err != nil {
		return "", err
	}
	return w.DocumentUploader.Upload(ctx, d)
}

type fRate struct {
	registrasi.ExchangeRateSource
	f *faults
}

func (w fRate) Find(ctx context.Context, c string, d time.Time) (registrasi.ExchangeRate, error) {
	if err := w.f.hit("Rate.Find"); err != nil {
		return 0, err
	}
	return w.ExchangeRateSource.Find(ctx, c, d)
}

type fPolicy struct {
	registrasi.PolicyRepo
	f *faults
}

func (w fPolicy) Get(ctx context.Context, n string) (registrasi.Policy, error) {
	if err := w.f.hit("Policy.Get"); err != nil {
		return registrasi.Policy{}, err
	}
	return w.PolicyRepo.Get(ctx, n)
}

type fNumber struct {
	registrasi.NumberIssuer
	f *faults
}

func (w fNumber) Issue(ctx context.Context, at time.Time) (string, error) {
	if err := w.f.hit("Number.Issue"); err != nil {
		return "", err
	}
	return w.NumberIssuer.Issue(ctx, at)
}

type fAssigner struct {
	registrasi.Assigner
	f *faults
}

func (w fAssigner) Assign(ctx context.Context, s registrasi.Stage, c registrasi.Claim, by string) (registrasi.Assignee, error) {
	if err := w.f.hit("Assigner.Assign"); err != nil {
		return registrasi.Assignee{}, err
	}
	return w.Assigner.Assign(ctx, s, c, by)
}

type fParameter struct {
	*memory.Parameter
	f *faults
}

func (w fParameter) LargeLossThreshold(ctx context.Context) (registrasi.Money, error) {
	if err := w.f.hit("Param.LargeLossThreshold"); err != nil {
		return 0, err
	}
	return w.Parameter.LargeLossThreshold(ctx)
}

func (w fParameter) LargeLossRecipients(ctx context.Context, l registrasi.LineOfBusiness) ([]string, error) {
	if err := w.f.hit("Param.LargeLossRecipients"); err != nil {
		return nil, err
	}
	return w.Parameter.LargeLossRecipients(ctx, l)
}

type fAccounts struct {
	registrasi.AccountDirectory
	f *faults
}

func (w fAccounts) FindAccount(ctx context.Context, n string) (registrasi.BankAccount, error) {
	if err := w.f.hit("Accounts.FindAccount"); err != nil {
		return registrasi.BankAccount{}, err
	}
	return w.AccountDirectory.FindAccount(ctx, n)
}

type fGroups struct {
	registrasi.GroupSource
	f *faults
}

func (w fGroups) GroupsOf(ctx context.Context, login string) ([]string, error) {
	if err := w.f.hit("Groups.GroupsOf"); err != nil {
		return nil, err
	}
	return w.GroupSource.GroupsOf(ctx, login)
}

type fLink struct {
	*memory.ClaimReportLink
	f *faults
}

func (w fLink) MarkHandedOver(ctx context.Context, id string, at time.Time) error {
	if err := w.f.hit("Link.MarkHandedOver"); err != nil {
		return err
	}
	return w.ClaimReportLink.MarkHandedOver(ctx, id, at)
}

func (w fLink) AttachClaimNumber(ctx context.Context, id, n string) error {
	if err := w.f.hit("Link.AttachClaimNumber"); err != nil {
		return err
	}
	return w.ClaimReportLink.AttachClaimNumber(ctx, id, n)
}

func (w fLink) Snapshot(ctx context.Context, id string) (registrasi.ClaimReportSnapshot, error) {
	if err := w.f.hit("Link.Snapshot"); err != nil {
		return registrasi.ClaimReportSnapshot{}, err
	}
	return w.ClaimReportLink.Snapshot(ctx, id)
}

type fPremium struct {
	*memory.Premium
	f *faults
}

func (w fPremium) Statement(ctx context.Context, app string, q registrasi.PremiumQuery) (registrasi.PremiumStatement, error) {
	if err := w.f.hit("Premium.Statement"); err != nil {
		return registrasi.PremiumStatement{}, err
	}
	return w.Premium.Statement(ctx, app, q)
}

// faultKeys adalah seluruh kunci yang dapat dipasang.
var faultKeys = []string{
	"Claim.Save", "Claim.Get", "Claim.GetByNumber", "Claim.FindDuplicates",
	"Task.Save", "Task.Get", "Task.OpenTaskForClaim", "Task.Inbox",
	"Inbox.Mirror", "Audit.Record", "Notifier.Send", "Unit.Run",
	"DLA.Issued", "DLA.PreDLA", "DLA.Previous", "DLA.Policy", "DLA.ReinsuranceCase", "DLA.Treaty",
	"DLA.NextNumber", "DLA.Save", "DLA.MarkPrinted",
	"PLA.CoinsMembers", "PLA.Recipient", "PLA.Previous", "PLA.Issued", "PLA.NextNumber", "PLA.Save",
	"PLA.UpdateNote", "PLA.Signature", "PLA.LODEmails",
	"Acc.NextNumber", "Acc.Save", "Acc.RecordLODPrint", "Acc.Fields", "Acc.OtherDLA", "Acc.AddHistory",
	"Acc.OpenPosition", "Acc.StartProgress", "Acc.AddProgress", "Acc.PolicyCaseID",
	"Acc.OpenProtectionApproved", "Acc.TravelClientName",
	"Committee.NextCaseID", "Committee.Save", "Committee.Get", "Committee.Pending", "Tiering.Route",
	"Records.Surveys", "Records.DocumentTypes", "Records.Attachments", "Records.Progress",
	"Records.Communications", "Records.AddAttachment",
	"FS.CauseOfLossName", "FS.OperatorName", "FS.Coinsurance", "FS.FacReinsurers", "FS.LastRevision",
	"FS.SaveRevision",
	"Render.FaceSheet", "Render.PLA", "Render.DLA", "Render.LOD",
	"Cashier.BankGroupID", "Cashier.Log", "Cashier.MarkTransferred",
	"Currency.Currencies", "Items.Items", "Items.ItemOptions", "Area.Options", "Upload.Upload",
	"Rate.Find", "Policy.Get", "Number.Issue", "Assigner.Assign",
	"Param.LargeLossThreshold", "Param.LargeLossRecipients", "Accounts.FindAccount", "Groups.GroupsOf",
	"Link.MarkHandedOver", "Link.AttachClaimNumber", "Link.Snapshot", "Premium.Statement",
}

// faultySetup membentuk lingkungan uji yang sama dengan setup, tetapi setiap seam
// dibungkus penyuntik galat. Pembantu alur (approvedClaim, acceptedForDLA, …) tetap
// berlaku karena mereka bekerja lewat environment.
func faultySetup(t *testing.T) (environment, *faults) {
	t.Helper()
	f := &faults{}

	fixed := clock.FixedAt(time.Date(2026, time.June, 10, 3, 0, 0, 0, time.UTC))
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

	rec := fRecords{records, f}
	items := fPolicyItems{policyItems, f}
	cash := fCashier{cashier, f}
	service, err := usecase.NewService(usecase.Options{
		CauseOfLoss:            memory.NewAreaDirectory(),
		AcceptanceNoteRenderer: acceptancenotepdf.Renderer{},
		ClaimRepo:              fClaimRepo{store, f},
		TaskRepo:               fTaskRepo{store.TaskRepo(), f},
		PolicyRepo:             fPolicy{memory.NewPolicyStore(memory.SamplePolicies(fixed.Now())...), f},
		NumberIssuer:           fNumber{memory.NewNumberIssuer(), f},
		Parameter:              fParameter{parameter, f},
		ExchangeRateSource:     fRate{memory.NewExchangeRateSource(), f},
		Assigner:               fAssigner{memory.NewAssigner(testTeams()), f},
		Notifier:               fNotifier{store, f},
		AuditRecorder:          fAudit{store, f},
		ClaimReportLink:        fLink{link, f},
		AreaDirectory:          fArea{memory.NewAreaDirectory(), f},
		PolicyItems:            items,
		CurrencyDirectory:      fCurrency{memory.CurrencyDirectory{}, f},
		ItemOptions:            items,
		ClaimRecords:           rec,
		FaceSheet:              fFaceSheet{memory.NewFaceSheet(), f},
		FaceSheetRenderer:      fFSRender{facesheetpdf.Renderer{}, f},
		PLA:                    fPLA{pla, f},
		PLARenderer:            fPLARender{plapdf.Renderer{}, f},
		DLA:                    fDLA{dla, f},
		DLARenderer:            fDLARender{dlapdf.Renderer{}, f},
		Cashier:                cash,
		CashierGateway:         cashier,
		LODRenderer:            fLODRender{lodpdf.Renderer{}, f},
		Acceptance:             fAcceptance{acceptance, f},
		Premium:                fPremium{premium, f},
		Groups:                 fGroups{groups, f},
		Inbox:                  fInbox{inbox, f},
		Accounts:               fAccounts{memory.NewAccounts(), f},
		CommitteeTiering:       fTiering{memory.NewCommitteeTiering(), f},
		Committees:             fCommittees{memory.NewCommittees(), f},
		Documents:              fUploader{uploader, f},
		Attachments:            rec,
		Closures:               rec,
		IDGenerator:            memory.IDGenerator{},
		UnitOfWork:             fUnit{store, f},
		Clock:                  fixed,
	})
	require.NoError(t, err)

	return environment{
		service: service, store: store, parameter: parameter, link: link, clock: fixed,
		pla: pla, dla: dla, cashier: cashier, groups: groups, inbox: inbox, records: records,
		uploader: uploader, acceptance: acceptance, premium: premium,
		caller: usecase.Caller{
			Identity: testOperator, Name: "Petugas Uji",
			Workbasket: []string{registrasi.WorkbasketRCLPUCL, registrasi.WorkbasketInvestigator, registrasi.WorkbasketCompliance},
		},
	}, f
}

// propagation adalah satu skenario uji propagasi galat: prepare membangun keadaan (tanpa
// galat), op menjalankan satu operasi layanan.
type propagation struct {
	name    string
	prepare func(t *testing.T, l environment) any
	op      func(l environment, state any) error

	// swallowed adalah kunci yang galatnya SENGAJA tidak diteruskan operasi ini.
	swallowed map[string]bool

	// translated adalah kunci yang galatnya sengaja diterjemahkan menjadi galat lain.
	translated map[string]error
}

// runPropagation memasang galat pada setiap seam, satu panggilan demi satu panggilan, dan
// memastikan setiap galat yang menyala diteruskan ke pemanggil.
func runPropagation(t *testing.T, p propagation) {
	t.Helper()
	for _, key := range faultKeys {
		for skip := 0; skip < 25; skip++ {
			l, f := faultySetup(t)
			state := p.prepare(t, l)
			f.arm(key, skip)
			err := p.op(l, state)
			fired := f.didFire()
			f.disarm()
			if !fired {
				break
			}
			if p.swallowed[key] {
				continue
			}
			if want, ok := p.translated[key]; ok {
				require.ErrorIs(t, err, want, "%s: galat %s (panggilan ke-%d)", p.name, key, skip+1)
				continue
			}
			require.ErrorIs(t, err, errFault, "%s: galat %s (panggilan ke-%d) tidak diteruskan", p.name, key, skip+1)
		}
	}
}
