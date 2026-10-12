// Package memory adalah pengisi seam penyimpanan modul auth yang hidup di dalam memori.
//
// Ia ada supaya aturan sesi dan alur masuk dapat diuji tanpa basis data dan tanpa
// jaringan sama sekali — inilah adapter kedua yang membuat seam penyimpanan menjadi
// seam nyata, bukan seam hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: sesi yang hidup di memori satu instans
// melanggar tuntutan stateless (D-27).
package memory

import (
	"context"
	"sync"
	"time"

	"claim-pnc/internal/auth"
)

// UserRepo menyimpan catatan pengguna di memori, dikunci Identity.
type UserRepo struct {
	mu      sync.RWMutex
	content map[string]auth.User
}

// NewUserRepo membentuk store kosong.
func NewUserRepo() *UserRepo {
	return &UserRepo{content: map[string]auth.User{}}
}

// GetByIdentity membaca satu catatan pengguna.
func (s *UserRepo) GetByIdentity(_ context.Context, identity string) (auth.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, existing := s.content[identity]
	if !existing {
		return auth.User{}, auth.ErrUserNotFound
	}
	return p, nil
}

// Save menulis catatan pengguna.
func (s *UserRepo) Save(_ context.Context, p auth.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lama, existing := s.content[p.Identity]; existing {
		// Status aktif dimiliki administrator, bukan sistem identitas luar; ia tidak
		// ikut tertimpa saat profil disegarkan.
		p.Active = lama.Active
		p.OperatorID = lama.OperatorID
		p.CreatedAt = lama.CreatedAt
	}
	s.content[p.Identity] = p
	return nil
}

// SetActive mengubah status aktif satu pengguna. Dipakai pengujian dan belum punya
// padanan layar administrasi — itu lingkup TKT-F3-004 yang masih terhalang artefak.
func (s *UserRepo) SetActive(identity string, active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, existing := s.content[identity]; existing {
		p.Active = active
		s.content[identity] = p
	}
}

// SessionRepo menyimpan sesi di memori, dikunci sidik token.
type SessionRepo struct {
	mu      sync.RWMutex
	content map[string]auth.Session
}

// NewSessionRepo membentuk store kosong.
func NewSessionRepo() *SessionRepo { return &SessionRepo{content: map[string]auth.Session{}} }

// Save menuliskan sesi baru.
func (s *SessionRepo) Save(_ context.Context, fresh auth.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content[fresh.TokenDigest] = fresh
	return nil
}

// GetByTokenDigest mencari sesi dari sidik tokennya.
func (s *SessionRepo) GetByTokenDigest(_ context.Context, fingerprint string) (auth.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ditemukan, existing := s.content[fingerprint]
	if !existing {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	return ditemukan, nil
}

// Revoke menandai sesi sebagai dicabut tanpa menghapus barisnya.
func (s *SessionRepo) Revoke(_ context.Context, id string, pada time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for fingerprint, item := range s.content {
		if item.ID == id && item.RevokedAt == nil {
			clock := pada.UTC()
			item.RevokedAt = &clock
			s.content[fingerprint] = item
		}
	}
	return nil
}

// Renew menggeser batas berlaku sesi yang masih aktif.
func (s *SessionRepo) Renew(_ context.Context, id string, berlakuSampai time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for fingerprint, item := range s.content {
		if item.ID == id && item.RevokedAt == nil {
			item.ExpiresAt = berlakuSampai.UTC()
			s.content[fingerprint] = item
		}
	}
	return nil
}

// Count mengembalikan banyaknya sesi yang pernah tersimpan, termasuk yang sudah
// dicabut — karena pencabutan tidak menghapus baris.
func (s *SessionRepo) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.content)
}
