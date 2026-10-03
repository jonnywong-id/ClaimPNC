package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ReceiverCommand adalah isian satu penerima klaim — panel `Section/InputReceiver_sect.xml`
// yang terbuka dari baris grid Penerima Klaim, atau baris baru dari tombol Tambah.
type ReceiverCommand struct {
	ClaimID string
	TaskID  string

	// ReceiverID adalah IDRECEIVER penerima yang diubah; kosong berarti penerima baru.
	ReceiverID string

	AccountNo string

	// Email dan Telephone ikut diperiksa tetapi BELUM tersimpan: T_CLAIM_RECEIVER tidak punya
	// kolomnya, dan Work Owner memutuskan skema tidak diubah dulu (2026-09-28). Pega
	// menyimpannya hanya di JSON klaim (ReceiverClaim.EmailReceiver/Telephone).
	Email     string
	Telephone string
}

// FindAccount membaca satu rekening Master Rekening — isian No Rekening, yang di Pega
// memanggil `GetDataBankMaster(norekening)` setiap nilainya berubah.
func (l *Service) FindAccount(ctx context.Context, number string) (registrasi.BankAccount, error) {
	number = strings.TrimSpace(number)
	if number == "" {
		return registrasi.BankAccount{}, registrasi.ErrAccountNotFound
	}
	return l.accounts.FindAccount(ctx, number)
}

// SaveReceiver menyimpan satu penerima klaim (tombol Simpan InputReceiver).
//
// Penjaganya sama dengan grid Adjustment di layar yang sama: tugas terbuka pada tahap
// InputSurveyor, dan pemanggil berwenang mengerjakannya.
func (l *Service) SaveReceiver(ctx context.Context, p ReceiverCommand, by Caller) (registrasi.Claim, error) {
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: p.TaskID, action: registrasi.ActionInputSurveyor})
	if err != nil {
		return registrasi.Claim{}, err
	}
	if claim.ID != p.ClaimID {
		return registrasi.Claim{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, p.TaskID, p.ClaimID)
	}
	if !settlementStages[task.Stage] {
		return registrasi.Claim{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, task.Stage)
	}
	if !l.canWork(task, by) {
		return registrasi.Claim{}, registrasi.ErrNotTaskOwner
	}

	index := -1
	if id := strings.TrimSpace(p.ReceiverID); id != "" {
		for i, r := range claim.Receiver {
			if strings.TrimSpace(r.ID) == id {
				index = i
			}
		}
		if index < 0 {
			return registrasi.Claim{}, fmt.Errorf("%w: penerima %s tidak ada pada klaim %s", registrasi.ErrInvalidAction, id, claim.Number)
		}
	}

	account, err := l.checkReceiver(ctx, p)
	if err != nil {
		return registrasi.Claim{}, err
	}

	receiver := registrasi.Receiver{ID: registrasi.NextReceiverID(claim.Receiver)}
	if index >= 0 {
		receiver = claim.Receiver[index]
	}
	receiver.ApplyAccount(account)
	if index >= 0 {
		claim.Receiver[index] = receiver
	} else {
		claim.Receiver = append(claim.Receiver, receiver)
	}

	now := l.clock.Now().UTC()
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "PENERIMA_DISIMPAN",
			Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Penerima %s: rekening %s bank %s", receiver.ID, receiver.AccountNo, receiver.BankName),
		})
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	return claim, nil
}

// checkReceiver memeriksa isian dan mengumpulkan seluruh pelanggarannya sekaligus.
//
//   - No Rekening harus terdaftar di Master Rekening: Nama, Nama Bank, dan Alamat baca-saja
//     di InputReceiver, jadi hanya rekening master yang dapat mengisinya.
//   - Email wajib bila `ObjectCoverageList(1).UserBusinessPA != '1'`. Penanda itu diisi alur
//     Komite (`KomitePost_Adjustment`), yang belum ada di aplikasi ini — pada klaim PNCN ia
//     selalu kosong, sehingga Email selalu wajib.
func (l *Service) checkReceiver(ctx context.Context, p ReceiverCommand) (registrasi.BankAccount, error) {
	var violations []registrasi.Violation
	var account registrasi.BankAccount

	number := strings.TrimSpace(p.AccountNo)
	if number == "" {
		violations = append(violations, registrasi.Violation{
			Code: registrasi.ViolationReceiverAccountEmpty, Field: "nomor_rekening", Message: "Value cannot be blank",
		})
	} else {
		found, err := l.accounts.FindAccount(ctx, number)
		switch {
		case err == nil && !found.AccountApproved():
			// GetDataPenerimaKlaim hanya memakai rekening APPROVAL = '1'. Pesan "sedang proses
			// approval" disalin dari langkah 11 (jalur PA); Pega memakainya hanya untuk PA, di
			// sini untuk semua lini supaya rekening yang ditolak diam-diam mendapat alasannya.
			message := "Account number is not approved in Master Rekening"
			if strings.TrimSpace(found.Approval) == "0" {
				message = "No Rekening Sedang Proses Approval"
			}
			violations = append(violations, registrasi.Violation{
				Code: registrasi.ViolationReceiverNotApproved, Field: "nomor_rekening", Message: message,
			})
		case err == nil:
			account = found
			if l.cashierAccountCheck {
				registered, err := l.cashier.AccountRegistered(ctx, number, found.BankID)
				if err != nil {
					return registrasi.BankAccount{}, err
				}
				if !registered {
					// Pesan Pega apa adanya (`GetDataBankMaster` langkah 7).
					violations = append(violations, registrasi.Violation{
						Code: registrasi.ViolationReceiverNotInCashier, Field: "nomor_rekening",
						Message: "Norekening Belum Terdaftar Di Sistem Kasir",
					})
				}
			}
		case errors.Is(err, registrasi.ErrAccountNotFound):
			violations = append(violations, registrasi.Violation{
				Code: registrasi.ViolationReceiverAccountUnknown, Field: "nomor_rekening",
				Message: "Account number is not registered in Master Rekening",
			})
		default:
			return registrasi.BankAccount{}, fmt.Errorf("registrasi/usecase: membaca master rekening: %w", err)
		}
	}
	if strings.TrimSpace(p.Email) == "" {
		violations = append(violations, registrasi.Violation{
			Code: registrasi.ViolationReceiverEmailEmpty, Field: "email", Message: "Value cannot be blank",
		})
	}
	if len(violations) > 0 {
		return registrasi.BankAccount{}, &registrasi.ValidationError{Violation: violations}
	}
	return account, nil
}
