package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ErrCashierUnavailable menandai sistem Kasir yang tidak dapat dihubungi atau belum
// terdaftar alamatnya. Tidak ada yang ditandai terkirim.
var ErrCashierUnavailable = errors.New("cashier system could not be reached")

// ErrCashierNoReply menandai Kasir yang menerima permintaan tetapi TIDAK MENJAWAB dalam batas
// waktu. Berbeda dari ErrCashierUnavailable: Kasir mungkin sudah memproses pembayarannya,
// sehingga layar harus meminta petugas memeriksa Kasir sebelum mencoba lagi — mengulang
// begitu saja dapat menghasilkan transfer ganda.
var ErrCashierNoReply = errors.New("cashier did not reply in time")

// CashierRejectedError adalah jawaban Kasir yang tidak memuat CaseIDCashier / NoTransClaim.
type CashierRejectedError struct{ Message string }

func (e *CashierRejectedError) Error() string { return "kasir menolak: " + e.Message }

// CashierCommand adalah permintaan tombol Transfer Kasir satu baris adjustment (berbasis 1).
type CashierCommand struct {
	ClaimID    string
	TaskID     string
	Object     int
	Coverage   int
	Adjustment int

	// TransferType adalah "Tipe Transfer Kasir" (`.JoinPlacement`); kosong berarti Pembayaran Biasa.
	TransferType string
	// UnpaidFacOut adalah No DLA FAC OUT yang dicentang "Pilih Fac-out Tidak Dibayar".
	UnpaidFacOut []string
}

// CashierPreview adalah isi dialog konfirmasi Transfer Kasir.
type CashierPreview struct {
	AcceptedNo string
	Receiver   registrasi.CashierReceiver
	Nett       registrasi.Money
	Currency   string
	// Problem adalah galat validasi pertama; kosong bila siap ditransfer.
	Problem string
	// Confirmation adalah kalimat Pre_AlertTransferkasir.
	Confirmation string
	// FacOut adalah isi tabel "Pilih Fac-out Tidak Dibayar" (GetdataFacoutJoinPlacement).
	FacOut []registrasi.CashierFacOut
}

type cashierScope struct {
	claim    registrasi.Claim
	object   registrasi.InsuredItem
	line     registrasi.SettlementLine
	receiver registrasi.CashierReceiver
	policy   registrasi.DLAPolicy
	nett     registrasi.Money
	check    registrasi.CashierCheck
	bankID   string
	facOut   []registrasi.CashierFacOut
}

func (l *Service) cashierScopeOf(ctx context.Context, p CashierCommand, by Caller) (cashierScope, error) {
	claim, _, err := l.lodTask(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return cashierScope{}, err
	}
	line, err := settlementAt(&claim, p.Object, p.Coverage, p.Adjustment)
	if err != nil {
		return cashierScope{}, err
	}
	if err := registrasi.CanTransferCashier(*line); err != nil {
		return cashierScope{}, err
	}
	object := claim.InsuredItem[p.Object-1]

	// Penerima: ReceiverClaim yang dipilih pada akseptasi; email, telepon, dan IDBank dari
	// rekening masternya (GetDataBankMaster).
	var receiver registrasi.CashierReceiver
	for _, r := range claim.Receiver {
		if strings.TrimSpace(r.ID) == strings.TrimSpace(line.Acceptance.Form.ReceiverID) {
			receiver = registrasi.CashierReceiver{Name: r.Name, AccountNo: r.AccountNo, BankName: r.BankName}
		}
	}
	if receiver.AccountNo != "" {
		account, err := l.accounts.FindAccount(ctx, receiver.AccountNo)
		if err != nil && !errors.Is(err, registrasi.ErrAccountNotFound) {
			return cashierScope{}, err
		}
		receiver.Email, receiver.Telephone, receiver.BankID = account.Email, account.Telephone, account.BankID
		if receiver.BankName == "" {
			receiver.BankName = account.BankName
		}
	}

	policy, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return cashierScope{}, err
	}
	nett := registrasi.CashierNett(claim.Portal, *line, claim.Policy.TypeOfCoins, policy.Coins)

	bankID, found := "", false
	if receiver.BankName != "" {
		if bankID, found, err = l.cashier.BankGroupID(ctx, receiver.BankName, receiver.BankID); err != nil {
			return cashierScope{}, err
		}
	}
	dla, err := l.dla.Issued(ctx, claim.ID, object.ID, p.Coverage, p.Adjustment)
	if err != nil {
		return cashierScope{}, err
	}
	printed := true
	for _, d := range dla {
		printed = printed && d.Printed
	}
	return cashierScope{
		claim: claim, object: object, line: *line, receiver: receiver, policy: policy, nett: nett, bankID: bankID,
		facOut: registrasi.CashierFacOuts(dla),
		check:  registrasi.CashierCheck{Line: *line, Receiver: receiver, Nett: nett, BankFound: found, DLAPrinted: printed},
	}, nil
}

// PreviewCashierTransfer mengisi dialog konfirmasi Transfer Kasir.
func (l *Service) PreviewCashierTransfer(ctx context.Context, p CashierCommand, by Caller) (CashierPreview, error) {
	sc, err := l.cashierScopeOf(ctx, p, by)
	if err != nil {
		return CashierPreview{}, err
	}
	names, _, err := l.currencyNames(ctx)
	if err != nil {
		return CashierPreview{}, err
	}
	out := CashierPreview{
		AcceptedNo: sc.line.AcceptedNo, Receiver: sc.receiver, Nett: sc.nett,
		Currency: firstText(names[sc.line.Currency], sc.line.Currency),
		Confirmation: registrasi.CashierConfirmation(sc.line.AcceptedNo,
			registrasi.CashierService(sc.claim.Portal, sc.policy.Syariah, sc.nett, sc.line.Rate)),
		FacOut: sc.facOut,
	}
	var validation *registrasi.ValidationError
	if err := registrasi.ValidateCashier(sc.check); errors.As(err, &validation) {
		if v, ok := validation.First(); ok {
			out.Problem = v.Message
		}
	}
	return out, nil
}

// TransferCashier mengirim pembayaran satu adjustment ke sistem Kasir — TransferToKasir_act
// jalur langsung. Berhasil: TRANSFER_CASHIER_DATE dan IDCHASIER terisi, Status Klaim 1162,
// dan tombolnya mati.
func (l *Service) TransferCashier(ctx context.Context, p CashierCommand, by Caller) (registrasi.Claim, error) {
	sc, err := l.cashierScopeOf(ctx, p, by)
	if err != nil {
		return registrasi.Claim{}, err
	}
	if err := registrasi.ValidateCashier(sc.check); err != nil {
		return registrasi.Claim{}, err
	}
	unpaid, err := registrasi.ChooseUnpaidFacOut(p.TransferType, p.UnpaidFacOut, sc.facOut)
	if err != nil {
		return registrasi.Claim{}, err
	}
	_, pic, err := l.pla.LODEmails(ctx, sc.claim.Number, sc.claim.TechnicalPIC)
	if err != nil {
		return registrasi.Claim{}, err
	}
	now := l.clock.Now().UTC()
	coverage := sc.object.Coverage[p.Coverage-1]
	payload := registrasi.BuildCashierPayload(sc.line, sc.receiver, sc.nett, registrasi.CashierFacts{
		Portal: sc.claim.Portal, ClaimNumber: sc.claim.Number, PolicyNumber: sc.claim.Policy.Number,
		BusinessCode: firstText(sc.policy.BusinessCode, sc.claim.Policy.BusinessCode), BranchCode: sc.claim.Policy.BranchCode,
		DateOfLoss: sc.claim.DateOfLoss, CauseOfLoss: coverage.CauseOfLoss, Syariah: sc.policy.Syariah,
		ExGratia: sc.claim.ExGratia, User: by.Identity, PICEmail: pic, BankGroupID: sc.bankID,
	}, now)
	registrasi.AddUnpaidFacOut(&payload, unpaid)
	service := registrasi.CashierService(sc.claim.Portal, sc.policy.Syariah, sc.nett, sc.line.Rate)

	reply, err := l.cashierGateway.Transfer(ctx, sc.claim.Portal, service, payload)

	// Log layanan ditulis untuk SETIAP pemanggilan, berhasil atau tidak — seperti Pega yang
	// mencatat JSONIN sebelum memanggil. Gagal menulis log TIDAK membatalkan transfer: Kasir
	// sudah menerima pembayaran, dan membatalkan di sini membuka jalan transfer ganda.
	response := reply.Body
	if err != nil {
		response = err.Error()
	}
	logNote := ""
	if logErr := l.cashier.LogService(ctx, registrasi.CashierServiceLog{
		ClaimNumber: sc.claim.Number, AcceptedNo: sc.line.AcceptedNo, Request: payload, Response: response,
	}); logErr != nil {
		logNote = "; log layanan gagal ditulis: " + logErr.Error()
	}
	if err != nil {
		var timeout interface{ Timeout() bool }
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
			return registrasi.Claim{}, fmt.Errorf("%w: %v", ErrCashierNoReply, err)
		}
		return registrasi.Claim{}, fmt.Errorf("%w: %v", ErrCashierUnavailable, err)
	}
	// TRF_KASIR_LOG ditulis begitu Kasir menjawab SUCCESS — sebelum CaseID diperiksa, seperti Pega.
	if reply.Logged() {
		if err := l.cashier.Log(ctx, registrasi.CashierLog{
			AcceptedNo: sc.line.AcceptedNo, ClaimNumber: sc.claim.Number, PIC: sc.claim.TechnicalPIC,
			Status: registrasi.CashierLogStatusTransfer, Reason: registrasi.CashierLogReasonTransfer,
		}); err != nil {
			return registrasi.Claim{}, err
		}
	}
	if !reply.Accepted() {
		message := firstText(reply.ResponseMessage, reply.Raw, "Kasir tidak mengembalikan CaseIDCashier.")
		return registrasi.Claim{}, &CashierRejectedError{Message: message}
	}

	at := now
	if !sc.line.CashierTransferredAt.IsZero() {
		at = sc.line.CashierTransferredAt
	}
	claim := sc.claim
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.cashier.MarkTransferred(ctx, claim.ID, sc.object.ID, p.Coverage, p.Adjustment, at, reply.CaseID()); err != nil {
			return err
		}
		claim.ClaimStatus = registrasi.StatusClaimTransferCashier
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "KASIR_TRANSFER", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Transfer Kasir %s — objek %d jaminan %d adjustment %d, layanan %s, CaseIDCashier %s, Fac-out tidak dibayar %d%s",
				sc.line.AcceptedNo, p.Object, p.Coverage, p.Adjustment, service, reply.CaseID(), len(unpaid), logNote),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return l.claim.Get(ctx, claim.ID)
}
