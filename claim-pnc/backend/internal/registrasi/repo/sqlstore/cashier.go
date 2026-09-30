package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// CashierStore memenuhi registrasi.CashierStore.
type CashierStore struct {
	db *sql.DB
}

// NewCashierStore membentuk penyimpanan Transfer Kasir.
func NewCashierStore(db *sql.DB) *CashierStore { return &CashierStore{db: db} }

var _ registrasi.CashierStore = (*CashierStore)(nil)

// BankGroupID mencari LBG_ID nama bank yang sama dengan IDBank penerima.
func (s *CashierStore) BankGroupID(ctx context.Context, bankName, bankID string) (string, bool, error) {
	rows, err := s.db.QueryContext(ctx, loadQuery("kasir_kode_bank"), strings.TrimSpace(bankName))
	if err != nil {
		return "", false, fmt.Errorf("registrasi/sqlstore: membaca kode bank: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return "", false, fmt.Errorf("registrasi/sqlstore: membaca baris kode bank: %w", err)
		}
		if strings.TrimSpace(id.String) == strings.TrimSpace(bankID) && strings.TrimSpace(id.String) != "" {
			return strings.TrimSpace(id.String), true, nil
		}
	}
	return "", false, rows.Err()
}

// Log menulis satu baris TRF_KASIR_LOG.
func (s *CashierStore) Log(ctx context.Context, e registrasi.CashierLog) error {
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("kasir_log"),
		e.AcceptedNo, emptyTextAsNil(e.ClaimNumber), emptyTextAsNil(e.PIC), e.Status, emptyTextAsNil(e.Reason)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis log kasir: %w", err)
	}
	return nil
}

// MarkTransferred mengisi TRANSFER_CASHIER_DATE (bila kosong) dan IDCHASIER.
func (s *CashierStore) MarkTransferred(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, at time.Time, caseID string) error {
	res, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("kasir_tandai"),
		wallDate(at), caseID, claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai transfer kasir: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return fmt.Errorf("registrasi/sqlstore: baris adjustment %s/%d/%d tidak ditemukan", objectID, coverageSeq, adjustmentSeq)
	}
	return nil
}
