package documentlink

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// Tanpa layanan dokumen penunjang, tautan dijawab galat yang dikenali layar — bukan panic.
func TestLinkTanpaLayanan(t *testing.T) {
	_, err := New(nil).Link(context.Background(), "ASM", "IMG-1", "MGR")
	require.ErrorIs(t, err, inboxmanager.ErrDocumentLinkUnavailable)

	var nilLinker *Linker
	_, err = nilLinker.Link(context.Background(), "ASM", "IMG-1", "MGR")
	require.ErrorIs(t, err, inboxmanager.ErrDocumentLinkUnavailable)
}
