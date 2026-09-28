package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// saveReceivers menuliskan penerima klaim ke T_CLAIM_RECEIVER.
func saveReceivers(ctx context.Context, exec executor, claimID string, receivers []registrasi.Receiver) error {
	for _, r := range receivers {
		values := []any{
			emptyTextAsNil(r.Name), emptyTextAsNil(r.Address), emptyTextAsNil(r.BankName),
			emptyTextAsNil(r.AccountNo), claimID, r.ID,
		}
		if err := upsert(ctx, exec, "penerima_perbarui", values, "penerima_sisip", values); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan penerima %s: %w", r.ID, err)
		}
	}
	return nil
}

// loadReceivers membaca penerima klaim.
func loadReceivers(ctx context.Context, exec executor, claimID string) ([]registrasi.Receiver, error) {
	rows, err := exec.QueryContext(ctx, loadQuery("penerima_daftar"), claimID)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca penerima klaim: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.Receiver
	for rows.Next() {
		var id, name, address, bank, account sql.NullString
		if err := rows.Scan(&id, &name, &address, &bank, &account); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris penerima klaim: %w", err)
		}
		result = append(result, registrasi.Receiver{
			ID: strings.TrimSpace(id.String), Name: name.String, Address: address.String,
			BankName: bank.String, AccountNo: account.String,
		})
	}
	return result, rows.Err()
}
