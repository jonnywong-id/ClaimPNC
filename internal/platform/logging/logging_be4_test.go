package logging_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/logging"
)

// TestRequestIDRoundTrip membuktikan ID permintaan tersimpan dan terbaca dari context.
func TestRequestIDRoundTrip(t *testing.T) {
	require.Empty(t, logging.RequestID(context.Background()))
	ctx := logging.WithRequestID(context.Background(), "abc123")
	require.Equal(t, "abc123", logging.RequestID(ctx))
}

// TestFromAttachesRequestID membuktikan baris log membawa id_permintaan bila ada.
func TestFromAttachesRequestID(t *testing.T) {
	var buffer bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buffer, nil))

	require.Same(t, base, logging.From(context.Background(), base))

	ctx := logging.WithRequestID(context.Background(), "req-7")
	logging.From(ctx, base).Info("halo")

	var line map[string]any
	require.NoError(t, json.Unmarshal(buffer.Bytes(), &line))
	require.Equal(t, "req-7", line["id_permintaan"])
	require.Equal(t, "halo", line["msg"])
}

// TestNewLoggerRespectsLevel membuktikan logger baru menyaring menurut tingkat.
func TestNewLoggerRespectsLevel(t *testing.T) {
	logger := logging.New(slog.LevelWarn)
	require.False(t, logger.Enabled(context.Background(), slog.LevelInfo))
	require.True(t, logger.Enabled(context.Background(), slog.LevelError))
}

// TestMaskNeverReturnsTheValue membuktikan Mask hanya menyebut panjang dalam rune.
func TestMaskNeverReturnsTheValue(t *testing.T) {
	require.Equal(t, "(kosong)", logging.Mask(""))
	require.Equal(t, "(disamarkan, 6 karakter)", logging.Mask("rahsia"))
	require.Equal(t, "(disamarkan, 2 karakter)", logging.Mask("é1"))
}
