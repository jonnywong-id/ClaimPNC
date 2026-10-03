package usecase_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator/repo/memory"
)

// Portal yang dapat dilayani lolos pemeriksaan tanpa membaca satu baris pun.
func TestEnsurePortalReadyAcceptsAServedPortal(t *testing.T) {
	service := serviceWith(t, memory.NewSampleRepo())
	require.NoError(t, service.EnsurePortalReady("asm"))
}

// Portal yang tidak dapat dilayani ditolak dengan galat yang menyebut aliasnya.
func TestEnsurePortalReadyRejectsAnUnservedPortal(t *testing.T) {
	service := serviceWith(t, memory.NewSampleRepo())

	err := service.EnsurePortalReady("smi")
	require.EqualError(t, err,
		`inboxinvestigator/usecase: portal "smi" tidak dapat dilayani: portal tidak dikenal: smi`)
}
