package dokumenpenunjang_test

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/dokumenpenunjang"
	"claim-pnc/internal/platform/clock"
)

func TestImageIDSebentukGenerateImageID(t *testing.T) {
	at := time.Date(2026, 9, 30, 14, 5, 9, 7_000_000, clock.ZoneWIB)
	sum := md5.Sum([]byte("ASMPP30/09/2026 14:05:09.007"))
	id := dokumenpenunjang.NewImageID(at)
	require.Equal(t, strings.ToUpper(hex.EncodeToString(sum[:])), id)
	require.Len(t, id, 32)
	require.Equal(t, strings.ToUpper(id), id)
}
