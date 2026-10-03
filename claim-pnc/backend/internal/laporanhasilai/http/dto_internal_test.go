package laporanhasilaihttp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFormatDateWritesZeroAsEmptyText(t *testing.T) {
	require.Equal(t, "", formatDate(time.Time{}))
	require.Equal(t, "2026-09-30",
		formatDate(time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)))
}

func TestTruncationNoticeWithoutColumnsIsEmpty(t *testing.T) {
	// Lebar nol tidak boleh membuat indeks ke-0 diisi (yang akan panik).
	require.Empty(t, truncationNotice(0, 99))
}
