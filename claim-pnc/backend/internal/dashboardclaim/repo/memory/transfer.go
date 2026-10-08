package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// TransferStore menyimpan permintaan transfer di dalam memori.
//
// Ia adapter KEDUA di balik seam dashboardclaim.TransferRepo, dan karena itulah seam-nya
// nyata. Dipakai pengujian dan mode pengembangan tanpa Oracle — yang pada modul INI punya
// nilai tambahan: tabelnya dibuat migrasi `0014` yang belum dijalankan DBA di lingkungan mana
// pun, sehingga tanpa adapter ini tombol Transfer tidak dapat dicoba sama sekali.
type TransferStore struct {
	mu       sync.RWMutex
	requests []dashboardclaim.TransferRequest
}

// NewTransferStore membentuk penyimpanan permintaan transfer.
func NewTransferStore() *TransferStore { return &TransferStore{} }

// Record mencatat satu permintaan.
func (s *TransferStore) Record(_ context.Context, request dashboardclaim.TransferRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests = append(s.requests, request)
	return nil
}

// PendingFor membaca permintaan yang masih menunggu atas sekumpulan klaim.
func (s *TransferStore) PendingFor(
	_ context.Context,
	claimIDs []string,
) (map[string][]dashboardclaim.TransferRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	diminta := make(map[string]bool, len(claimIDs))
	for _, id := range claimIDs {
		diminta[id] = true
	}

	result := map[string][]dashboardclaim.TransferRequest{}
	for _, request := range s.requests {
		if request.Status != dashboardclaim.TransferPending {
			continue
		}
		if !diminta[request.ClaimID] {
			continue
		}
		result[request.ClaimID] = append(result[request.ClaimID], request)
	}
	return result, nil
}

// IDGenerator membangkitkan pengenal permintaan.
//
// Acak 128 bit, bukan berurut: pengenal permintaan tidak boleh membocorkan berapa banyak
// permintaan yang sudah tercatat.
type IDGenerator struct{}

// New membangkitkan satu pengenal.
//
// Kegagalan crypto/rand dikembalikan sebagai galat, TIDAK diganti nilai cadangan: pengenal
// yang dapat ditebak pada tabel jejak lebih buruk daripada permintaan yang gagal tercatat.
func (IDGenerator) New() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

// FixedClock adalah jam tetap untuk pengujian.
type FixedClock struct{ At time.Time }

// Now mengembalikan waktu tetap.
func (c FixedClock) Now() time.Time { return c.At }
