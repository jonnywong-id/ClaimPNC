package memory

import (
	"context"
	"sync"

	"claim-pnc/internal/dashboardclaim"
)

// ClosedReader memenuhi seam dashboardclaim.ClosedClaimReader di dalam memori.
//
// Ia TERPISAH dari Repo, dan pemisahan itu disengaja: di produksi kedua seam memang dipenuhi
// dua hal yang berbeda — Repo oleh sqlstore modul ini, ClosedClaimReader oleh adapter atas
// modul inboxcloseclaim. Menyatukan keduanya di sini akan membuat uji lulus pada susunan
// yang tidak pernah benar-benar dirakit.
//
// Ia mengabaikan portalAlias, dan itu BUKAN pengabaian aturan portal: satu ClosedReader
// mewakili SATU entitas, persis seperti satu Repo. Pemilihan menurut portal dikerjakan
// selector di cmd — yang juga diuji, di routes_test.go, dengan dua entitas berisi berbeda.
type ClosedReader struct {
	mu     sync.RWMutex
	claims []ClaimRecord
}

// NewClosedReader membentuk pembaca klaim tutup dalam memori.
func NewClosedReader(claims []ClaimRecord) *ClosedReader {
	return &ClosedReader{claims: append([]ClaimRecord(nil), claims...)}
}

// Count menghitung klaim tutup yang cocok dengan penyaring.
func (r *ClosedReader) Count(_ context.Context, _ string, f dashboardclaim.Filter) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.matching(f.Normalize())), nil
}

// List membaca satu halaman klaim tutup.
func (r *ClosedReader) List(_ context.Context, _ string, f dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f = f.Normalize()
	matched := r.matching(f)

	rows := make([]dashboardclaim.ClaimRow, 0, f.Limit)
	for _, index := range paginate(len(matched), f) {
		rows = append(rows, matched[index].Row)
	}
	return dashboardclaim.ClaimPage{Rows: rows, Total: len(matched)}, nil
}

// matching menyaring klaim tutup dengan aturan yang sama seperti klaim berjalan.
func (r *ClosedReader) matching(f dashboardclaim.Filter) []ClaimRecord {
	result := make([]ClaimRecord, 0, len(r.claims))
	for _, record := range r.claims {
		if !matchesBusiness(f.Business, record.GroupPanel, record.BusinessGroupID) {
			continue
		}
		if !matchesSearch(f.Search, record.Row.PolicyNumber, record.Row.ClaimNumber) {
			continue
		}
		result = append(result, record)
	}
	return result
}
