package memory

import (
	"context"
	"sync"

	"claim-pnc/internal/registrasi"
)

// InboxEntries adalah padanan memori POOLDATA.T_CLAIMLIST_ADMIN: satu baris per nomor klaim.
type InboxEntries struct {
	mu   sync.Mutex
	rows map[string]registrasi.InboxEntry
}

// NewInboxEntries membentuk daftar kerja memori yang masih kosong.
func NewInboxEntries() *InboxEntries {
	return &InboxEntries{rows: map[string]registrasi.InboxEntry{}}
}

// Mirror menimpa baris klaim — perilaku update-lalu-insert penyimpanan SQL.
func (s *InboxEntries) Mirror(_ context.Context, e registrasi.InboxEntry) error {
	if e.Key() == "" {
		return nil
	}
	if e.Task != nil {
		t := *e.Task
		e.Task = &t
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[e.Key()] = e
	return nil
}

// Get mengembalikan baris sebuah nomor klaim; ok false bila belum pernah ditulis.
func (s *InboxEntries) Get(number string) (registrasi.InboxEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.rows[number]
	return e, ok
}

var _ registrasi.InboxMirror = (*InboxEntries)(nil)
