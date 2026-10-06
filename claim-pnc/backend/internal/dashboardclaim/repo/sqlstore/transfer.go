package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// TransferRepo menulis dan membaca permintaan transfer pada satu basis data entitas.
//
// TERPISAH dari Repo, dan pemisahannya mengikuti kepemilikan tabel: Repo membaca tabel milik
// Pega, yang ini menulis tabel milik aplikasi sendiri (`P-1`).
type TransferRepo struct {
	db *sql.DB
}

// NewTransferRepo membentuk penyimpanan permintaan transfer.
func NewTransferRepo(db *sql.DB) *TransferRepo { return &TransferRepo{db: db} }

// Record mencatat satu permintaan transfer.
func (r *TransferRepo) Record(ctx context.Context, request dashboardclaim.TransferRequest) error {
	_, err := r.db.ExecContext(ctx, query("transfer_insert"),
		request.ID,
		string(request.Scope),
		nilIfEmpty(request.ClaimID),
		nilIfEmpty(request.ClaimNumber),
		nilIfEmpty(request.FromOperator),
		request.ToOperator,
		nilIfEmpty(request.UserType),
		nilIfEmpty(request.Reason),
		request.RequestedBy,
		nilIfEmpty(request.RequestedByName),
	)
	if err != nil {
		return fmt.Errorf("dashboardclaim/sqlstore: mencatat permintaan transfer: %w", err)
	}
	return nil
}

// PendingFor membaca permintaan yang masih menunggu atas sekumpulan klaim.
//
// Daftar kosong dijawab peta kosong TANPA menyentuh basis data: `IN ()` bukan SQL yang sah,
// dan menyusunnya tetap akan gagal pada halaman yang kebetulan tidak punya baris.
func (r *TransferRepo) PendingFor(
	ctx context.Context,
	claimIDs []string,
) (map[string][]dashboardclaim.TransferRequest, error) {
	result := map[string][]dashboardclaim.TransferRequest{}
	if len(claimIDs) == 0 {
		return result, nil
	}

	statement, args := expandClaims(query("transfer_pending"), claimIDs)

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: membaca permintaan transfer tertunda: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		request, err := scanTransferRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("dashboardclaim/sqlstore: membaca baris permintaan transfer: %w", err)
		}
		result[request.ClaimID] = append(result[request.ClaimID], request)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("dashboardclaim/sqlstore: menutup permintaan transfer tertunda: %w", err)
	}
	return result, nil
}

// expandClaims mengganti penanda /*CLAIMS*/ dengan daftar parameter sepanjang klaim yang
// diminta.
//
// Yang disisipkan adalah PENANDA (`:1, :2, …`), bukan nilainya. Jumlah penandanya memang
// berubah tiap permintaan — itu satu-satunya yang dirangkai; nilainya tidak pernah menyentuh
// teks SQL.
func expandClaims(statement string, claimIDs []string) (string, []any) {
	markers := make([]string, 0, len(claimIDs))
	args := make([]any, 0, len(claimIDs))

	for index, id := range claimIDs {
		markers = append(markers, ":"+strconv.Itoa(index+1))
		args = append(args, id)
	}

	return strings.Replace(statement, "/*CLAIMS*/", strings.Join(markers, ", "), 1), args
}

// scanTransferRequest membaca satu baris permintaan.
func scanTransferRequest(s scanner) (dashboardclaim.TransferRequest, error) {
	var (
		id           sql.NullString
		scope        sql.NullString
		claimID      sql.NullString
		claimNumber  sql.NullString
		fromOperator sql.NullString
		toOperator   sql.NullString
		userType     sql.NullString
		reason       sql.NullString
		status       sql.NullString
		requestedBy  sql.NullString
		requesterNm  sql.NullString
		requestedAt  sql.NullTime
	)

	if err := s.Scan(
		&id, &scope, &claimID, &claimNumber,
		&fromOperator, &toOperator, &userType, &reason,
		&status, &requestedBy, &requesterNm, &requestedAt,
	); err != nil {
		return dashboardclaim.TransferRequest{}, err
	}

	request := dashboardclaim.TransferRequest{
		ID:              text(id),
		Scope:           dashboardclaim.TransferScope(text(scope)),
		ClaimID:         text(claimID),
		ClaimNumber:     text(claimNumber),
		FromOperator:    text(fromOperator),
		ToOperator:      text(toOperator),
		UserType:        text(userType),
		Reason:          text(reason),
		Status:          dashboardclaim.TransferStatus(text(status)),
		RequestedBy:     text(requestedBy),
		RequestedByName: text(requesterNm),
	}
	if requestedAt.Valid {
		request.RequestedAt = requestedAt.Time
	}
	return request, nil
}
