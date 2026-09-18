package cashier

import (
	"context"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Fake memenuhi seam Cashier tanpa jaringan sama sekali.
//
// Ia dipakai di lingkungan development dan di seluruh pengujian. Keberadaannya bukan
// kenyamanan: adapter keduanyalah yang membuat seam Cashier menjadi seam nyata, bukan
// seam hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.4).
//
// Ia MENOLAK dipakai di produksi — pemeriksaannya ada di cmd/claimpnc, tempat seluruh
// penolakan lingkungan lain berada.
type Fake struct {
	mu sync.Mutex

	// Response menentukan apa yang dikembalikan panggilan berikutnya. Nol-nilainya
	// adalah pendaftaran yang berhasil.
	Response masterrekening.CashierResult

	// Error, bila terisi, dikembalikan sebagai kegagalan menghubungi Kasir.
	Error error

	// Registered dan Diperbarui mencatat rekening yang dikirim, supaya pengujian
	// dapat memeriksa jalur mana yang dipakai.
	Registered []masterrekening.Account
	Updated    []masterrekening.Account
}

// NewFake membentuk tiruan yang selalu menjawab berhasil.
func NewFake() *Fake {
	return &Fake{
		Response: masterrekening.CashierResult{
			Succeeded: true,
			Code:      "0",
			Message:   "[TIRUAN] Rekening diterima sistem Kasir.",
			AccountID: "TIRUAN-0001",
		},
	}
}

// Register mencatat rekening dan mengembalikan jawaban yang sudah disiapkan.
func (t *Fake) Register(_ context.Context, r masterrekening.Account) (masterrekening.CashierResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Registered = append(t.Registered, r)
	if t.Error != nil {
		return masterrekening.CashierResult{}, t.Error
	}
	return t.Response, nil
}

// Update mencatat rekening dan mengembalikan jawaban yang sudah disiapkan.
func (t *Fake) Update(_ context.Context, r masterrekening.Account) (masterrekening.CashierResult, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Updated = append(t.Updated, r)
	if t.Error != nil {
		return masterrekening.CashierResult{}, t.Error
	}
	return t.Response, nil
}

var _ masterrekening.Cashier = (*Fake)(nil)
