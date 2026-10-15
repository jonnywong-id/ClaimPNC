package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// AccountDirectory membaca Master Rekening (POOLDATA.LST_ACCOUNT).
type AccountDirectory struct{ db *sql.DB }

// NewAccountDirectory membentuk pembaca Master Rekening di atas sebuah koneksi.
func NewAccountDirectory(db *sql.DB) *AccountDirectory { return &AccountDirectory{db: db} }

// FindAccount mengembalikan rekening bernomor itu, atau registrasi.ErrAccountNotFound.
func (d *AccountDirectory) FindAccount(ctx context.Context, number string) (registrasi.BankAccount, error) {
	var (
		no, name, bank, branch, address, bankID, email, phone, approval sql.NullString
		cashier, committee                                              sql.NullTime
	)
	err := d.db.QueryRowContext(ctx, loadQuery("rekening_ambil"), strings.TrimSpace(number)).
		Scan(&no, &name, &bank, &branch, &address, &bankID, &email, &phone, &cashier, &committee, &approval)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.BankAccount{}, registrasi.ErrAccountNotFound
	}
	if err != nil {
		return registrasi.BankAccount{}, fmt.Errorf("registrasi/sqlstore: membaca master rekening: %w", err)
	}
	return registrasi.BankAccount{
		Number: strings.TrimSpace(no.String), Name: strings.TrimSpace(name.String),
		BankName: strings.TrimSpace(bank.String), Branch: strings.TrimSpace(branch.String),
		Address: strings.TrimSpace(address.String), BankID: strings.TrimSpace(bankID.String),
		Email: strings.TrimSpace(email.String), Telephone: strings.TrimSpace(phone.String),
		CashierApprovedAt: cashier.Time, CommitteeApprovedAt: committee.Time,
		Approval: strings.TrimSpace(approval.String),
	}, nil
}
