package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/registrasi"
)

// Accounts adalah Master Rekening di memori — contoh fiktif, bukan data nasabah.
type Accounts struct {
	mu   sync.Mutex
	rows map[string]registrasi.BankAccount
}

// NewAccounts membentuk Master Rekening berisi dua rekening contoh.
func NewAccounts() *Accounts {
	a := &Accounts{rows: map[string]registrasi.BankAccount{}}
	a.Add(registrasi.BankAccount{
		Number: "1234567890", Name: "PT CONTOH PENERIMA", BankName: "BANK CONTOH", Branch: "JAKARTA",
		Address: "JL. CONTOH NO. 1", BankID: "001", Email: "penerima@contoh.internal",
		CommitteeApprovedAt: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), Approval: "1",
	})
	a.Add(registrasi.BankAccount{
		Number: "9876543210", Name: "CV CONTOH KEDUA", BankName: "BANK CONTOH", Branch: "SURABAYA",
		Address: "JL. CONTOH NO. 2", BankID: "001", Approval: "1",
	})
	return a
}

// Add menambahkan atau mengganti satu rekening.
func (a *Accounts) Add(r registrasi.BankAccount) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rows[strings.TrimSpace(r.Number)] = r
}

// FindAccount mengembalikan rekening bernomor itu, atau registrasi.ErrAccountNotFound.
func (a *Accounts) FindAccount(_ context.Context, number string) (registrasi.BankAccount, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	r, ok := a.rows[strings.TrimSpace(number)]
	if !ok {
		return registrasi.BankAccount{}, registrasi.ErrAccountNotFound
	}
	return r, nil
}
