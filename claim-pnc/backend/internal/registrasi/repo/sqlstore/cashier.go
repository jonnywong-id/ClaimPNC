package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
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

// AccountRegistered memenuhi registrasi.CashierStore.
func (s *CashierStore) AccountRegistered(ctx context.Context, accountNo, bankID string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, loadQuery("kasir_rekening_terdaftar"),
		strings.TrimSpace(accountNo), strings.TrimSpace(bankID)).Scan(&n); err != nil {
		return false, fmt.Errorf("registrasi/sqlstore: memeriksa rekening di Kasir: %w", err)
	}
	return n > 0, nil
}

// LogService menulis satu baris CLAIM_SERVICE_LOG. JSONIN adalah badan yang dikirim ke Kasir —
// json.Marshal atas muatan yang sama dengan cashierlink, sehingga isinya sama persis.
func (s *CashierStore) LogService(ctx context.Context, e registrasi.CashierServiceLog) error {
	body, err := json.Marshal(e.Request)
	if err != nil {
		return err
	}
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("kasir_log_layanan"),
		e.ClaimNumber, string(body), emptyTextAsNil(e.Response), registrasi.CashierServiceLogCategory,
		emptyTextAsNil(e.AcceptedNo)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis log layanan kasir: %w", err)
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
