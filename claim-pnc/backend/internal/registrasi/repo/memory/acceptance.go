package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/registrasi"
)

// Acceptance adalah penyimpanan akseptasi di memori, untuk pengujian dan pengembangan lokal.
type Acceptance struct {
	mu sync.Mutex

	counter  int64
	Saved    map[string]registrasi.SettlementLine       // per kunci baris
	DLA      map[string][]registrasi.AcceptanceDLAState // per id klaim
	History  []AcceptanceHistory
	Progress []registrasi.ProgressUpdate
	LODPrint map[string]LODPrint // per kunci baris

	// Positions adalah ID posisi progres terbuka per PositionKey(nomor klaim, nama posisi).
	Positions map[string]string
	// Started adalah pemanggilan StartProgress (kategori INSERT).
	Started []registrasi.ProgressStart

	CaseIDs        map[string]string // per nomor polis
	OpenProtection map[string]bool   // per nomor polis
	TravelClients  map[string]string // per nomor polis
}

// LODPrint adalah catatan Print LOD satu baris: tanggal cetak pertama dan jenis terakhir.
type LODPrint struct {
	PrintedAt time.Time
	Type      string
}

// AcceptanceHistory adalah satu baris riwayat yang ditulis.
type AcceptanceHistory struct {
	CaseID, Note, User string
	At                 time.Time
}

// NewAcceptance membentuk penyimpanan akseptasi kosong.
func NewAcceptance() *Acceptance {
	return &Acceptance{Saved: map[string]registrasi.SettlementLine{}, DLA: map[string][]registrasi.AcceptanceDLAState{}}
}

func lineKey(claimID, objectID string, coverageSeq, adjustmentSeq int) string {
	return fmt.Sprintf("%s/%s/%d/%d", claimID, objectID, coverageSeq, adjustmentSeq)
}

func (s *Acceptance) NextNumber(_ context.Context, year int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return registrasi.PLANumber(registrasi.AcceptanceCode, year, "10", s.counter), nil
}

func (s *Acceptance) Save(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line registrasi.SettlementLine) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Saved[lineKey(claimID, objectID, coverageSeq, adjustmentSeq)] = line
	return nil
}

func (s *Acceptance) RecordLODPrint(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, printedAt time.Time, lodType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.LODPrint == nil {
		s.LODPrint = map[string]LODPrint{}
	}
	key := lineKey(claimID, objectID, coverageSeq, adjustmentSeq)
	p := s.LODPrint[key]
	if p.PrintedAt.IsZero() {
		p.PrintedAt = printedAt
	}
	p.Type = lodType
	s.LODPrint[key] = p
	return nil
}

func (s *Acceptance) SetLODType(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, lodType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.LODPrint == nil {
		s.LODPrint = map[string]LODPrint{}
	}
	key := lineKey(claimID, objectID, coverageSeq, adjustmentSeq)
	p := s.LODPrint[key]
	p.Type = lodType
	s.LODPrint[key] = p
	return nil
}

func (s *Acceptance) Fields(_ context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line *registrasi.SettlementLine) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if saved, ok := s.Saved[lineKey(claimID, objectID, coverageSeq, adjustmentSeq)]; ok {
		line.Acceptance.Form = saved.Acceptance.Form
	}
	return nil
}

func (s *Acceptance) OtherDLA(_ context.Context, claimID, _ string, _, _ int) ([]registrasi.AcceptanceDLAState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.DLA[claimID], nil
}

func (s *Acceptance) AddHistory(_ context.Context, caseID, note, user string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.History = append(s.History, AcceptanceHistory{CaseID: caseID, Note: note, User: user, At: at})
	return nil
}

// PositionKey adalah kunci Positions.
func PositionKey(claimNumber, position string) string { return claimNumber + "|" + position }

func (s *Acceptance) OpenPosition(_ context.Context, claimNumber, position string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Positions[PositionKey(claimNumber, position)], nil
}

func (s *Acceptance) StartProgress(_ context.Context, p registrasi.ProgressStart) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Positions == nil {
		s.Positions = map[string]string{}
	}
	s.Started = append(s.Started, p)
	id := fmt.Sprintf("%d", len(s.Started))
	s.Positions[PositionKey(p.ClaimNumber, p.Position)] = id
	return id, nil
}

func (s *Acceptance) AddProgress(_ context.Context, p registrasi.ProgressUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Progress = append(s.Progress, p)
	// Posisi yang ditutup tidak lagi terbuka.
	for k, id := range s.Positions {
		if id == p.PositionID && strings.HasPrefix(k, p.ClaimNumber+"|") && p.Position == registrasi.AcceptanceProgressDone {
			delete(s.Positions, k)
		}
	}
	return nil
}

func (s *Acceptance) PolicyCaseID(_ context.Context, policy, _ string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.CaseIDs[policy], nil
}

func (s *Acceptance) OpenProtectionApproved(_ context.Context, policy, _, _ string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.OpenProtection[policy], nil
}

func (s *Acceptance) TravelClientName(_ context.Context, policy string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TravelClients[policy], nil
}

var _ registrasi.AcceptanceSource = (*Acceptance)(nil)
