package sqlstore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// Tombol Tutup Klaim: kolom penutupan, log, dashboard, beban PIC, identitas lama, dan penanda.
func TestClosureStore(t *testing.T) {
	db, mock := be4DB(t)
	s := NewClosureStore(db)
	ctx := context.Background()
	closed := time.Date(2026, 10, 4, 5, 0, 0, 0, time.UTC) // 12:00 WIB
	wall := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

	mock.ExpectExec(be4Q("klaim_tutup_simpan")).
		WithArgs("n", "u", nil, nil, nil, wall, "K1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.SaveClosure(ctx, "K1", registrasi.Closure{Note: "n", Proposal: "u"}, closed))

	mock.ExpectExec(be4Q("klaim_tutup_simpan")).
		WithArgs("n", nil, nil, nil, "true", nil, "K1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.SaveClosure(ctx, "K1", registrasi.Closure{Note: "n", Temporary: true}, time.Time{}))

	mock.ExpectExec(be4Q("klaim_tutup_log")).
		WithArgs("ASM-FW-GCNMFW-WORK PNCN.26.1", "PNCN.26.1", wall, "JONNY", "Close", "n").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.LogClosure(ctx, registrasi.ClosureLog{
		CaseKey: "ASM-FW-GCNMFW-WORK PNCN.26.1", ClaimNumber: "PNCN.26.1", User: "JONNY", Action: "Close", Note: "n", At: closed,
	}))

	mock.ExpectExec(be4Q("klaim_tutup_dashboard")).WithArgs("PNCN.26.1").WillReturnResult(sqlmock.NewResult(0, 0))
	require.NoError(t, s.MarkDashboardClosed(ctx, "PNCN.26.1"))

	mock.ExpectExec(be4Q("klaim_tutup_beban_pic")).WithArgs("JONNY").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.ReleaseTechnicalPIC(ctx, "JONNY"))

	mock.ExpectQuery(be4Q("operator_lama")).WithArgs("JONNY").WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow(" OLD "))
	old, err := s.LegacyOperator(ctx, "JONNY")
	require.NoError(t, err)
	require.Equal(t, "OLD", old)
	mock.ExpectQuery(be4Q("operator_lama")).WithArgs("X").WillReturnError(sql.ErrNoRows)
	old, err = s.LegacyOperator(ctx, "X")
	require.NoError(t, err)
	require.Empty(t, old)

	mock.ExpectQuery(be4Q("klaim_tutup_sementara")).WithArgs("K1").WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("true"))
	pending, err := s.PendingClose(ctx, "K1")
	require.NoError(t, err)
	require.True(t, pending)

	mock.ExpectExec(be4Q("klaim_tutup_dashboard")).WillReturnError(be4Boom)
	require.ErrorIs(t, s.MarkDashboardClosed(ctx, "PNCN.26.1"), be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())
}
