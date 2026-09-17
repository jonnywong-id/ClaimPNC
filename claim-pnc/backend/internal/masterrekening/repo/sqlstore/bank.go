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

// BankRepoBaru membentuk repo; db wajib sudah terhubung.
func BankRepoBaru(db *sql.DB) *BankRepo { return &BankRepo{db: db} }

// Daftar membaca seluruh bank.
func (r *BankRepo) Daftar(ctx context.Context) ([]masterrekening.Bank, error) {
	baris, err := r.db.QueryContext(ctx, ambilKueri("bank_daftar"))
	if err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: membaca daftar bank: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []masterrekening.Bank
	for baris.Next() {
		var kode, nama sql.NullString
		if err := baris.Scan(&kode, &nama); err != nil {
			return nil, fmt.Errorf("masterrekening/sqlstore: membaca baris bank: %w", err)
		}
		b := masterrekening.Bank{Kode: teks(kode), Nama: teks(nama)}
		// Baris tanpa kode tidak dapat dipilih pengguna dan hanya akan menghasilkan
		// rekening tanpa bank. Ia dilewati di sini, bukan dibiarkan muncul di layar.
		if b.Kode == "" {
			continue
		}
		if b.Nama == "" {
			b.Nama = b.Kode
		}
		hasil = append(hasil, b)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("masterrekening/sqlstore: menelusuri daftar bank: %w", err)
	}
	return hasil, nil
}

var _ masterrekening.BankRepo = (*BankRepo)(nil)
