package memory

import (
	"context"
	"fmt"
	"sync"

	"claim-pnc/internal/registrasi"
)

// DLA adalah penyimpanan DLA di memori, untuk pengujian dan pengembangan lokal.
type DLA struct {
	mu sync.Mutex

	Policies map[string]registrasi.DLAPolicy // per nomor polis
	Cases    map[string]string               // REINSURANCETYPE.TYPE per kode treaty
	Treaties map[string]registrasi.TreatyArrangement
	Pre      []registrasi.DLA
	Saved    []registrasi.DLA
	counter  int64
}

// NewDLA membentuk penyimpanan DLA kosong.
func NewDLA() *DLA {
	return &DLA{
		Policies: map[string]registrasi.DLAPolicy{}, Cases: map[string]string{},
		Treaties: map[string]registrasi.TreatyArrangement{},
	}
}

var _ registrasi.DLASource = (*DLA)(nil)

func matchDLA(d registrasi.DLA, claimID, objectID string, coverageSeq, adjustmentSeq int) bool {
	return d.ClaimID == claimID && d.ObjectID == objectID && d.CoverageSeq == coverageSeq && d.AdjustmentSeq == adjustmentSeq
}

func (s *DLA) Issued(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.DLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []registrasi.DLA
	for _, d := range s.Saved {
		if matchDLA(d, claimID, objectID, coverageSeq, adjustmentSeq) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *DLA) PreDLA(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.DLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []registrasi.DLA
	for _, d := range s.Pre {
		if matchDLA(d, claimID, objectID, coverageSeq, adjustmentSeq) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *DLA) Previous(_ context.Context, claimID, code string) (registrasi.DLAPrevious, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var last registrasi.DLAPrevious
	found := false
	for _, d := range s.Saved {
		if d.ClaimID == claimID && d.RecipientCode == code && (!found || !d.Date.Before(last.Date)) {
			last, found = registrasi.DLAPrevious{Number: d.Number, Date: d.Date}, true
		}
	}
	return last, found, nil
}

func (s *DLA) Policy(_ context.Context, number string) (registrasi.DLAPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Policies[number], nil
}

func (s *DLA) ReinsuranceCase(context.Context) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]string{}
	for k, v := range s.Cases {
		out[k] = v
	}
	return out, nil
}

func (s *DLA) Treaty(_ context.Context, _ string, _ int, treatyType string) (registrasi.TreatyArrangement, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Treaties[treatyType], nil
}

func (s *DLA) NextNumber(_ context.Context, code string, year int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return registrasi.PLANumber(code, year, "1", s.counter), nil
}

func (s *DLA) Save(_ context.Context, d registrasi.DLA) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, x := range s.Saved {
		if x.ClaimID == d.ClaimID && x.Number == d.Number {
			return fmt.Errorf("memory: nomor DLA %s ganda", d.Number)
		}
	}
	s.Saved = append(s.Saved, d)
	return nil
}

func (s *DLA) MarkPrinted(_ context.Context, claimID, number, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Saved {
		if s.Saved[i].ClaimID == claimID && s.Saved[i].Number == number {
			s.Saved[i].Note = note
			s.Saved[i].Printed = true
		}
	}
	return nil
}
