package memory

import (
	"context"
	"strings"
)

// Batas data pada penyimpanan memori.
//
// Bawaannya TIDAK membatasi siapa pun: tidak ada petugas yang punya cabang maupun group,
// sehingga layar pengembangan berperilaku seperti petugas kantor pusat. Uji yang menguji
// batas data mengisinya lewat SetBranch dan SetGroups.

// SampleRegions adalah isi contoh dropdown "Pilih Kanwil".
var SampleRegions = []string{"1", "2"}

// SetBranch menetapkan cabang seorang petugas.
func (s *Store) SetBranch(login, code string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.branches == nil {
		s.branches = map[string]string{}
	}
	s.branches[strings.ToUpper(login)] = code
}

// SetGroups menetapkan access group seorang petugas.
func (s *Store) SetGroups(login string, groups ...string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.groups == nil {
		s.groups = map[string][]string{}
	}
	s.groups[strings.ToUpper(login)] = groups
}

// BranchOfLogin memenuhi inboxadmin.ScopeRepo.
func (s *Store) BranchOfLogin(_ context.Context, login string) (string, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	code, ok := s.branches[strings.ToUpper(strings.TrimSpace(login))]
	return code, ok, nil
}

// GroupsOf memenuhi inboxadmin.ScopeRepo.
func (s *Store) GroupsOf(_ context.Context, login string) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return append([]string{}, s.groups[strings.ToUpper(strings.TrimSpace(login))]...), nil
}

// Regions memenuhi inboxadmin.ScopeRepo.
func (s *Store) Regions(context.Context) ([]string, error) {
	return append([]string{}, SampleRegions...), nil
}
