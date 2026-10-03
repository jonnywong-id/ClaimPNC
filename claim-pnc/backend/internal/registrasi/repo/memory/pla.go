package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"claim-pnc/internal/registrasi"
)

// PLA adalah penyimpanan PLA di memori, untuk pengujian dan pengembangan lokal.
type PLA struct {
	mu sync.Mutex

	Coins      map[string][]registrasi.PLACoinsMember // per nomor polis
	Recipients map[string]registrasi.PLARecipientInfo
	Saved      []registrasi.PLA
	counter    int64

	InsuredEmails map[string]string // per nomor klaim
	PICEmails     map[string]string // per operator PIC teknik
}

// NewPLA membentuk penyimpanan PLA kosong.
func NewPLA() *PLA {
	return &PLA{Coins: map[string][]registrasi.PLACoinsMember{}, Recipients: map[string]registrasi.PLARecipientInfo{}}
}

func (s *PLA) CoinsMembers(_ context.Context, policy, _ string) ([]registrasi.PLACoinsMember, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Coins[policy], nil
}

func (s *PLA) Recipient(_ context.Context, code, _ string) (registrasi.PLARecipientInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Recipients[code], nil
}

func (s *PLA) Previous(_ context.Context, claimID, code string) (registrasi.PLAPrevious, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var last registrasi.PLAPrevious
	found := false
	for _, p := range s.Saved {
		if p.ClaimID == claimID && p.RecipientCode == code && (!found || !p.Date.Before(last.Date)) {
			last, found = registrasi.PLAPrevious{Number: p.Number, Date: p.Date}, true
		}
	}
	return last, found, nil
}

func (s *PLA) Issued(_ context.Context, claimID, objectID string, coverageSeq, revision int) ([]registrasi.PLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []registrasi.PLA
	for _, p := range s.Saved {
		if p.ClaimID == claimID && p.ObjectID == objectID && p.CoverageSeq == coverageSeq && p.Revision == revision {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *PLA) NextNumber(_ context.Context, code string, year int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return registrasi.PLANumber(code, year, "1", s.counter), nil
}

func (s *PLA) Save(_ context.Context, p registrasi.PLA) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.Saved {
		if x.Number == p.Number {
			return fmt.Errorf("memory: nomor PLA %s ganda", p.Number)
		}
	}
	if p.Date.IsZero() {
		p.Date = time.Now()
	}
	s.Saved = append(s.Saved, p)
	return nil
}

func (s *PLA) UpdateNote(_ context.Context, claimID, number string, revision int, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Saved {
		if s.Saved[i].ClaimID == claimID && s.Saved[i].Number == number && s.Saved[i].Revision == revision {
			s.Saved[i].Note = note
			return nil
		}
	}
	return fmt.Errorf("memory: PLA %s tidak ada", number)
}

func (s *PLA) Signature(_ context.Context, _ string) (string, []byte, error) { return "", nil, nil }

func (s *PLA) LODEmails(_ context.Context, claimNumber, pic string) (string, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.InsuredEmails[claimNumber], s.PICEmails[pic], nil
}

var _ registrasi.PLASource = (*PLA)(nil)
