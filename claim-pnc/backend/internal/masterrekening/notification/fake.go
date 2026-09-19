package notification

import (
	"context"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Fake memenuhi seam Notifier tanpa jaringan sama sekali.
//
// Ia dipakai di lingkungan development dan di seluruh pengujian. Adapter keduanyalah
// yang membuat seam Notifier menjadi seam nyata, bukan seam hipotetis
// (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.6).
type Fake struct {
	mu sync.Mutex

	// Sent mencatat setiap peringatan, supaya pengujian dapat memeriksa isinya.
	Sent []masterrekening.Alert

	// Error, bila terisi, dikembalikan sebagai kegagalan pengiriman.
	Error error
}

// WarnCashierFailure mencatat satu peringatan.
func (t *Fake) WarnCashierFailure(_ context.Context, p masterrekening.Alert) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Sent = append(t.Sent, p)
	return t.Error
}

// Count mengembalikan banyaknya peringatan yang tercatat.
func (t *Fake) Count() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Sent)
}

var _ masterrekening.Notifier = (*Fake)(nil)
