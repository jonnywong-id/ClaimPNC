package memory

import (
	"context"
	"sync"

	"claim-pnc/internal/registrasi"
)

// FaceSheet adalah data pendamping Claim Face Sheet di memori, untuk pengujian dan
// pengembangan lokal. Revisi yang disimpan dapat dibaca kembali lewat Revision.
type FaceSheet struct {
	mu sync.Mutex

	CauseOfLoss map[string]string
	Operator    map[string]string
	CoMember    map[string][]registrasi.CoinsuranceRow
	Reinsurer   map[string][]registrasi.FacReinsurer
	Revision    []registrasi.FaceSheetRevision
}

// NewFaceSheet membentuk penyimpanan Claim Face Sheet kosong.
func NewFaceSheet() *FaceSheet {
	return &FaceSheet{
		CauseOfLoss: map[string]string{},
		Operator:    map[string]string{},
		CoMember:    map[string][]registrasi.CoinsuranceRow{},
		Reinsurer:   map[string][]registrasi.FacReinsurer{},
	}
}

func (f *FaceSheet) CauseOfLossName(_ context.Context, id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.CauseOfLoss[id], nil
}

func (f *FaceSheet) OperatorName(_ context.Context, id string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Operator[id], nil
}

func (f *FaceSheet) Coinsurance(_ context.Context, policy string) ([]registrasi.CoinsuranceRow, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.CoMember[policy], nil
}

func (f *FaceSheet) FacReinsurers(_ context.Context, policy string) ([]registrasi.FacReinsurer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Reinsurer[policy], nil
}

func (f *FaceSheet) LastRevision(_ context.Context, claimID, objectID string, coverageSeq int) (int, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	last, found := 0, false
	for _, r := range f.Revision {
		if r.ClaimID == claimID && r.ObjectID == objectID && r.CoverageSeq == coverageSeq && (!found || r.Revision > last) {
			last, found = r.Revision, true
		}
	}
	return last, found, nil
}

func (f *FaceSheet) SaveRevision(_ context.Context, r registrasi.FaceSheetRevision) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.Revision {
		if x.ClaimID == r.ClaimID && x.ObjectID == r.ObjectID && x.CoverageSeq == r.CoverageSeq && x.Revision == r.Revision {
			return registrasi.ErrInvalidAction
		}
	}
	f.Revision = append(f.Revision, r)
	return nil
}

var _ registrasi.FaceSheetSource = (*FaceSheet)(nil)
