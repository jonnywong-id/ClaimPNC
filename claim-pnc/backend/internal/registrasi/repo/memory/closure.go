package memory

import (
	"context"
	"time"

	"claim-pnc/internal/registrasi"
)

// ClosureState merekam akibat tombol Tutup Klaim untuk pengujian.
type ClosureState struct {
	Saved           map[string]registrasi.Closure
	ClosedAt        map[string]time.Time
	Logs            []registrasi.ClosureLog
	DashboardClosed []string
	ReleasedPIC     []string
	LegacyOperators map[string]string
}

// SaveClosure memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) SaveClosure(_ context.Context, claimID string, c registrasi.Closure, closedAt time.Time) error {
	if r.Closure.Saved == nil {
		r.Closure.Saved = map[string]registrasi.Closure{}
		r.Closure.ClosedAt = map[string]time.Time{}
	}
	r.Closure.Saved[claimID] = c
	if !closedAt.IsZero() {
		r.Closure.ClosedAt[claimID] = closedAt
	}
	return nil
}

// LogClosure memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) LogClosure(_ context.Context, l registrasi.ClosureLog) error {
	r.Closure.Logs = append(r.Closure.Logs, l)
	return nil
}

// MarkDashboardClosed memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) MarkDashboardClosed(_ context.Context, claimNumber string) error {
	r.Closure.DashboardClosed = append(r.Closure.DashboardClosed, claimNumber)
	return nil
}

// ReleaseTechnicalPIC memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) ReleaseTechnicalPIC(_ context.Context, operatorID string) error {
	r.Closure.ReleasedPIC = append(r.Closure.ReleasedPIC, operatorID)
	return nil
}

// LegacyOperator memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) LegacyOperator(_ context.Context, operatorID string) (string, error) {
	return r.Closure.LegacyOperators[operatorID], nil
}

// PendingClose memenuhi registrasi.ClosureStore.
func (r *ClaimRecords) PendingClose(_ context.Context, claimID string) (bool, error) {
	return r.Closure.Saved[claimID].Temporary, nil
}

var _ registrasi.ClosureStore = (*ClaimRecords)(nil)
