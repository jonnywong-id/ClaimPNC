package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// CauseOfLossOptions membaca pilihan Penyebab Kerugian satu kode bisnis polis. Kode bisnis
// kosong menghasilkan daftar kosong, bukan seluruh isi master.
func (d *AreaDirectory) CauseOfLossOptions(ctx context.Context, businessCode string) ([]registrasi.CauseOfLossOption, error) {
	businessCode = strings.TrimSpace(businessCode)
	out := []registrasi.CauseOfLossOption{}
	if businessCode == "" {
		return out, nil
	}
	rows, err := executorFrom(ctx, d.db).QueryContext(ctx, loadQuery("penyebab_kerugian_bisnis"), businessCode)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca penyebab kerugian bisnis %s: %w", businessCode, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris penyebab kerugian: %w", err)
		}
		if strings.TrimSpace(id.String) == "" {
			continue
		}
		out = append(out, registrasi.CauseOfLossOption{ID: strings.TrimSpace(id.String), Name: strings.TrimSpace(name.String)})
	}
	return out, rows.Err()
}

var _ registrasi.CauseOfLossDirectory = (*AreaDirectory)(nil)
