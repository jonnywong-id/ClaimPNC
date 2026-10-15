package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Committees adalah T_CLAIM_KOMITE_LIST di memori.
type Committees struct {
	mu    sync.Mutex
	cases map[string]registrasi.CommitteeCase
	next  map[int]int64
}

// NewCommittees membentuk penyimpan kasus komite kosong.
func NewCommittees() *Committees {
	return &Committees{cases: map[string]registrasi.CommitteeCase{}, next: map[int]int64{}}
}

// NextCaseID menerbitkan KMTN.26.1, KMTN.26.2, … — deret per tahun WIB.
func (c *Committees) NextCaseID(_ context.Context, at time.Time) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	year := clock.DateWIB(at).Year()
	c.next[year]++
	return registrasi.FormatCommitteeCaseID(year, c.next[year])
}

// Save menyimpan salinan kasus.
func (c *Committees) Save(_ context.Context, k registrasi.CommitteeCase) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	k.Members = append([]registrasi.CommitteeMember(nil), k.Members...)
	c.cases[k.ID] = k
	return nil
}

// Get membaca salinan kasus.
func (c *Committees) Get(_ context.Context, id string) (registrasi.CommitteeCase, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k, ok := c.cases[strings.TrimSpace(id)]
	if !ok {
		return registrasi.CommitteeCase{}, registrasi.ErrCommitteeNotFound
	}
	k.Members = append([]registrasi.CommitteeMember(nil), k.Members...)
	return k, nil
}

// Pending mengembalikan anggota yang sedang ditunggu dan milik operator itu.
func (c *Committees) Pending(_ context.Context, operator string) ([]registrasi.CommitteeMember, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []registrasi.CommitteeMember
	for _, k := range c.cases {
		if m, ok := k.Current(); ok && strings.EqualFold(m.Operator, strings.TrimSpace(operator)) {
			result = append(result, m)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CaseID < result[j].CaseID })
	return result, nil
}

// CommitteeTiering adalah penjenjangan tetap: setiap lini mendapat penyetuju yang sama.
// Mode memori tidak membaca EMAILKOMITE.
type CommitteeTiering struct {
	List []registrasi.CommitteeApprover
}

// NewCommitteeTiering membentuk penjenjangan tetap dengan dua jenjang contoh.
func NewCommitteeTiering(approvers ...registrasi.CommitteeApprover) CommitteeTiering {
	if len(approvers) == 0 {
		approvers = []registrasi.CommitteeApprover{
			{OperatorID: "KOMITE01", Name: "KOMITE CONTOH 1"},
			{OperatorID: "KOMITE02", Name: "KOMITE CONTOH 2"},
		}
	}
	return CommitteeTiering{List: approvers}
}

// Route mengembalikan penyetuju tetap, tanpa pengaju; pita tidak dipakai.
func (t CommitteeTiering) Route(_ context.Context, _ string, _ registrasi.Money, applicant string) (registrasi.CommitteeRoute, error) {
	var result registrasi.CommitteeRoute
	for _, a := range t.List {
		if !strings.EqualFold(a.OperatorID, strings.TrimSpace(applicant)) {
			result.Approvers = append(result.Approvers, a)
		}
	}
	return result, nil
}
