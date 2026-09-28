package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/registrasi"
)

// Committees adalah T_CLAIM_KOMITE_LIST di memori.
type Committees struct {
	mu    sync.Mutex
	cases map[string]registrasi.CommitteeCase
	next  int
}

// NewCommittees membentuk penyimpan kasus komite kosong.
func NewCommittees() *Committees { return &Committees{cases: map[string]registrasi.CommitteeCase{}} }

// NextCaseID menerbitkan KMTN-00001, KMTN-00002, …
func (c *Committees) NextCaseID(context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	return fmt.Sprintf("%s%05d", registrasi.CommitteeCasePrefix, c.next), nil
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

// Approvers mengembalikan penyetuju tetap, tanpa pengaju.
func (t CommitteeTiering) Approvers(_ context.Context, _ string, _ registrasi.Money, applicant string) ([]registrasi.CommitteeApprover, error) {
	var result []registrasi.CommitteeApprover
	for _, a := range t.List {
		if !strings.EqualFold(a.OperatorID, strings.TrimSpace(applicant)) {
			result = append(result, a)
		}
	}
	return result, nil
}
