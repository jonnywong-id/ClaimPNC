package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/masterrekening"
)

// BankRepo membaca GENERAL.LST_BANK_GROUP.
//
// Tabel milik sistem lain; aplikasi ini hanya membaca (ADR-0004).
type BankRepo struct {
	db *sql.DB
}

// NewBankRepo membentuk repo; db wajib sudah terhubung.
func NewBankRepo(db *sql.DB) *BankRepo { return &BankRepo{db: db} }

// List membaca seluruh bank.
func (r *BankRepo) List(ctx context.Context) ([]masterrekening.Bank, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("bank_list"))
	if err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterrekening.Bank
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("masterrekening/sqlstore: membaca baris bank: %w", err)
		}
		b := masterrekening.Bank{Code: text(code), Name: text(name)}
		// Baris tanpa kode tidak dapat dipilih pengguna dan hanya akan menghasilkan
		// rekening tanpa bank. Ia dilewati di sini, bukan dibiarkan muncul di layar.
		if b.Code == "" {
			continue
		}
		if b.Name == "" {
			b.Name = b.Code
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: menelusuri daftar bank: %w", err)
	}
	return result, nil
}

var _ masterrekening.BankRepo = (*BankRepo)(nil)
