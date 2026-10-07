package decision

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueIDs(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, UniqueIDs([]string{" a ", "", "b", "a", "  "}))
	require.Empty(t, UniqueIDs(nil))
}

func TestWarnPartial(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	WarnPartial(logger, "sebagian", "ASM", 2, 2, "budi")
	WarnPartial(nil, "sebagian", "ASM", 2, 1, "budi")
	require.Empty(t, buf.String())
	WarnPartial(logger, "sebagian", "ASM", 2, 1, "budi")
	require.Contains(t, buf.String(), "dipilih=2 berubah=1 oleh=budi")
}
