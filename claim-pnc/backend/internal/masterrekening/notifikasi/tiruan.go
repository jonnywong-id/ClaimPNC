package notifikasi

import (
	"context"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Tiruan memenuhi seam Notifier tanpa jaringan sama sekali.
//
// Ia dipakai di lingkungan development dan di seluruh pengujian. Adapter keduanyalah
// yang membuat seam Notifier menjadi seam nyata, bukan seam hipotetis
// (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.6).
type Tiruan struct {
	mu sync.Mutex

	// Terkirim mencatat setiap peringatan, supaya pengujian dapat memeriksa isinya.
	Terkirim []masterrekening.Peringatan

	// Galat, bila terisi, dikembalikan sebagai kegagalan pengiriman.
	Galat error
}

// PeringatkanKegagalanKasir mencatat satu peringatan.
func (t *Tiruan) PeringatkanKegagalanKasir(_ context.Context, p masterrekening.Peringatan) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Terkirim = append(t.Terkirim, p)
	return t.Galat
}

// Jumlah mengembalikan banyaknya peringatan yang tercatat.
func (t *Tiruan) Jumlah() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Terkirim)
}

var _ masterrekening.Notifier = (*Tiruan)(nil)
