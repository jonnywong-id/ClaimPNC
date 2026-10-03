package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// FaceSheetStore membaca data pendamping Claim Face Sheet dan menyimpan revisinya.
type FaceSheetStore struct {
	db     *sql.DB
	policy *PolicyRepo
}

// NewFaceSheetStore membentuk penyimpanan Claim Face Sheet.
func NewFaceSheetStore(db *sql.DB) *FaceSheetStore {
	return &FaceSheetStore{db: db, policy: NewPolicyRepo(db)}
}

// CauseOfLossName membaca uraian penyebab kerugian; kosong bila kodenya tidak dikenal.
func (s *FaceSheetStore) CauseOfLossName(ctx context.Context, id string) (string, error) {
	var name sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("cfs_penyebab"), strings.TrimSpace(id)).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca penyebab kerugian %q: %w", id, err)
	}
	return strings.TrimSpace(name.String), nil
}

// OperatorName membaca nama operator; kosong bila tidak ditemukan.
func (s *FaceSheetStore) OperatorName(ctx context.Context, operatorID string) (string, error) {
	id := strings.TrimSpace(operatorID)
	if id == "" {
		return "", nil
	}
	rows, err := s.db.QueryContext(ctx, loadQuery("cfs_operator"), id, id)
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca nama operator %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return "", fmt.Errorf("registrasi/sqlstore: membaca baris nama operator: %w", err)
		}
		if n := strings.TrimSpace(name.String); n != "" {
			return n, nil
		}
	}
	return "", rows.Err()
}

// Coinsurance membaca T_COINSLIST polis pada PRODKE-nya.
func (s *FaceSheetStore) Coinsurance(ctx context.Context, policyNumber, prodKe string) ([]registrasi.CoinsuranceRow, error) {
	return s.policy.coinsurance(ctx, s.db, strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe))
}

// FacReinsurers membaca T_FACOFFER polis pada PRODKE-nya.
func (s *FaceSheetStore) FacReinsurers(ctx context.Context, policyNumber, prodKe string) ([]registrasi.FacReinsurer, error) {
	rows, err := s.db.QueryContext(ctx, loadQuery("cfs_fac_offer"), strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca fac offer polis %q: %w", policyNumber, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.FacReinsurer
	for rows.Next() {
		var name, share sql.NullString
		if err := rows.Scan(&name, &share); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris fac offer: %w", err)
		}
		p, ok := parsePercent(share.String)
		out = append(out, registrasi.FacReinsurer{Name: strings.TrimSpace(name.String), Share: p, HasShare: ok})
	}
	return out, rows.Err()
}

// LastRevision mengembalikan nomor revisi terakhir sebuah jaminan.
func (s *FaceSheetStore) LastRevision(ctx context.Context, claimID, objectID string, coverageSeq int) (int, bool, error) {
	var last sql.NullInt64
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("cfs_revisi_terakhir"),
		claimID, objectID, strconv.Itoa(coverageSeq)).Scan(&last)
	if err != nil {
		return 0, false, fmt.Errorf("registrasi/sqlstore: membaca revisi Claim Face Sheet: %w", err)
	}
	return int(last.Int64), last.Valid, nil
}

// SaveRevision menulis satu revisi beserta reserve-nya.
func (s *FaceSheetStore) SaveRevision(ctx context.Context, r registrasi.FaceSheetRevision) error {
	exec := executorFrom(ctx, s.db)
	coverageID := strconv.Itoa(r.CoverageSeq)
	if _, err := exec.ExecContext(ctx, loadQuery("cfs_sisip"),
		r.ClaimID, r.ObjectID, coverageID, r.Revision, r.Date.UTC(), r.FileName); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan revisi Claim Face Sheet: %w", err)
	}
	for i, reserve := range r.Reserve {
		if _, err := exec.ExecContext(ctx, loadQuery("cfs_estimasi_sisip"),
			r.ClaimID, r.ObjectID, coverageID, r.Revision, i+1, reserve.Currency, r.Date.UTC(),
			int64(reserve.Value)); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan estimasi Claim Face Sheet: %w", err)
		}
	}
	return nil
}

var _ registrasi.FaceSheetSource = (*FaceSheetStore)(nil)
