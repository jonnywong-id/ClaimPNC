package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// ClosureStore memenuhi registrasi.ClosureStore.
type ClosureStore struct {
	db *sql.DB
}

// NewClosureStore membentuk ClosureStore.
func NewClosureStore(db *sql.DB) *ClosureStore { return &ClosureStore{db: db} }

// SaveClosure memenuhi registrasi.ClosureStore.
func (s *ClosureStore) SaveClosure(ctx context.Context, claimID string, c registrasi.Closure, closedAt time.Time) error {
	pending := any(nil)
	if c.Temporary {
		pending = "true"
	}
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("klaim_tutup_simpan"),
		emptyTextAsNil(c.Note), emptyTextAsNil(c.Proposal), emptyTextAsNil(c.Effort), emptyTextAsNil(c.Obstacle),
		pending, wallOrNil(closedAt), claimID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan penutupan klaim %s: %w", claimID, err)
	}
	return nil
}

// LogClosure memenuhi registrasi.ClosureStore.
func (s *ClosureStore) LogClosure(ctx context.Context, l registrasi.ClosureLog) error {
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("klaim_tutup_log"),
		l.CaseKey, l.ClaimNumber, wallOrNil(l.At), emptyTextAsNil(l.User), l.Action, emptyTextAsNil(l.Note))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis log penutupan %s: %w", l.ClaimNumber, err)
	}
	return nil
}

// MarkDashboardClosed memenuhi registrasi.ClosureStore.
func (s *ClosureStore) MarkDashboardClosed(ctx context.Context, claimNumber string) error {
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("klaim_tutup_dashboard"), claimNumber); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai dashboard klaim %s tutup: %w", claimNumber, err)
	}
	return nil
}

// ReleaseTechnicalPIC memenuhi registrasi.ClosureStore.
func (s *ClosureStore) ReleaseTechnicalPIC(ctx context.Context, operatorID string) error {
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("klaim_tutup_beban_pic"), operatorID); err != nil {
		return fmt.Errorf("registrasi/sqlstore: mengurangi beban PIC %s: %w", operatorID, err)
	}
	return nil
}

// LegacyOperator memenuhi registrasi.ClosureStore.
func (s *ClosureStore) LegacyOperator(ctx context.Context, operatorID string) (string, error) {
	var old sql.NullString
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("operator_lama"), operatorID).Scan(&old)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca identitas lama pengguna: %w", err)
	}
	return strings.TrimSpace(old.String), nil
}

// PendingClose memenuhi registrasi.ClosureStore.
func (s *ClosureStore) PendingClose(ctx context.Context, claimID string) (bool, error) {
	var v sql.NullString
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("klaim_tutup_sementara"), claimID).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("registrasi/sqlstore: membaca penanda tutup sementara: %w", err)
	}
	return strings.EqualFold(strings.TrimSpace(v.String), "true"), nil
}

var _ registrasi.ClosureStore = (*ClosureStore)(nil)
