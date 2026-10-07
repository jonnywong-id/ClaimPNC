package idformat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Next menerbitkan kunci berikutnya: kode situs (`POOLDATA.M_SITE_DATABASE`, baris
// CURRENT_SITE='1') dan nomor urut dibaca di dalam SATU transaksi, lalu disusun lewat Compose.
// siteQuery mengembalikan satu kolom kode situs; sequenceQuery satu kolom nomor urut. prefix
// mengawali setiap pesan galat, misalnya "masterbengkel/sqlstore".
func Next(ctx context.Context, db *sql.DB, siteQuery, sequenceQuery, prefix string, width int) (string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("%s: memulai transaksi penomoran: %w", prefix, err)
	}
	defer func() { _ = tx.Rollback() }()

	var site sql.NullString
	if err := tx.QueryRowContext(ctx, siteQuery).Scan(&site); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New(prefix + ": POOLDATA.M_SITE_DATABASE tidak punya baris CURRENT_SITE='1'")
		}
		return "", fmt.Errorf("%s: membaca kode situs: %w", prefix, err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, sequenceQuery).Scan(&sequence); err != nil {
		return "", fmt.Errorf("%s: mengambil nomor urut: %w", prefix, err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("%s: menutup transaksi penomoran: %w", prefix, err)
	}
	return Compose(strings.TrimSpace(site.String), sequence, width), nil
}
