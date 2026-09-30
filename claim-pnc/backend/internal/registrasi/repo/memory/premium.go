package memory

import (
	"context"
	"math/big"
	"sync"

	"claim-pnc/internal/registrasi"
)

// Premium adalah layanan status premi di memori. Tanpa isian, setiap polis dijawab lunas
// (AgingAmount 0) — pengembangan lokal tidak menjangkau layanan premi sungguhan.
type Premium struct {
	mu         sync.Mutex
	Statements map[string]registrasi.PremiumStatement // per nomor polis
	Err        error
}

// NewPremium membentuk layanan premi kosong.
func NewPremium() *Premium {
	return &Premium{Statements: map[string]registrasi.PremiumStatement{}}
}

func (s *Premium) Statement(_ context.Context, _ string, q registrasi.PremiumQuery) (registrasi.PremiumStatement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Err != nil {
		return registrasi.PremiumStatement{}, s.Err
	}
	if st, ok := s.Statements[q.PolicyNumber]; ok {
		return st, nil
	}
	return registrasi.PremiumStatement{AgingAmount: new(big.Rat)}, nil
}

var _ registrasi.PremiumService = (*Premium)(nil)
