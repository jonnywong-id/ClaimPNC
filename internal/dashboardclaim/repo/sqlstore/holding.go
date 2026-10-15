package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/dashboardclaim"
)

// holdingFilterArgs menyusun argumen tab Inbox Tampungan PIC.
//
// HANYA DUA, bukan delapan seperti tab sebelah: kueri lamanya tidak punya penanda lini
// bisnis sama sekali, dan layar lamanya tidak menggambar dropdown-nya pada tab ini.
//
// Dipisahkan menjadi fungsinya sendiri supaya perbedaan itu terlihat. Memakai ulang
// filterArgs akan mengirim enam argumen yang tidak punya penanda, dan Oracle menolaknya
// dengan ORA-01008 pada permintaan pertama yang datang.
func holdingFilterArgs(f dashboardclaim.Filter) []any {
	search := nilIfEmpty(f.Search)
	searchPattern := nilIfEmpty(likePattern(f.Search))

	return []any{
		search,        // :1 kotak cari aktif?
		searchPattern, // :2 PYID
	}
}

// ListHolding membaca satu halaman tab Inbox Tampungan PIC.
func (r *Repo) ListHolding(ctx context.Context, f dashboardclaim.Filter) (dashboardclaim.HoldingPage, error) {
	f = f.Normalize()
	filters := holdingFilterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("holding_count"), filters...).Scan(&total); err != nil {
		return dashboardclaim.HoldingPage{}, fmt.Errorf("dashboardclaim/sqlstore: menghitung klaim tampungan: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("holding_list"), withPaging(filters, f)...)
	if err != nil {
		return dashboardclaim.HoldingPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca daftar klaim tampungan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]dashboardclaim.HoldingRow, 0, f.Limit)
	for rows.Next() {
		row, err := scanHoldingRow(rows)
		if err != nil {
			return dashboardclaim.HoldingPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca baris klaim tampungan: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return dashboardclaim.HoldingPage{}, fmt.Errorf("dashboardclaim/sqlstore: menutup daftar klaim tampungan: %w", err)
	}

	return dashboardclaim.HoldingPage{Rows: result, Total: total}, nil
}

// scanHoldingRow membaca satu baris penampungan.
//
// Urutan pemindaiannya WAJIB sama dengan urutan kolom pada `holding_list`.
func scanHoldingRow(s scanner) (dashboardclaim.HoldingRow, error) {
	var (
		claimID        sql.NullString
		claimNumber    sql.NullString
		policyNumber   sql.NullString
		insuredName    sql.NullString
		businessName   sql.NullString
		businessSource sql.NullString
		branchName     sql.NullString
		adminPNC       sql.NullString
		registeredAt   sql.NullTime
	)

	if err := s.Scan(
		&claimID,
		&claimNumber,
		&policyNumber,
		&insuredName,
		&businessName,
		&businessSource,
		&branchName,
		&adminPNC,
		&registeredAt,
	); err != nil {
		return dashboardclaim.HoldingRow{}, err
	}

	row := dashboardclaim.HoldingRow{
		ClaimID:        text(claimID),
		ClaimNumber:    text(claimNumber),
		PolicyNumber:   text(policyNumber),
		InsuredName:    text(insuredName),
		BusinessName:   text(businessName),
		BusinessSource: text(businessSource),
		BranchName:     text(branchName),
		AdminPNC:       text(adminPNC),
	}
	if registeredAt.Valid {
		row.RegisteredAt = registeredAt.Time
	}
	return row, nil
}
