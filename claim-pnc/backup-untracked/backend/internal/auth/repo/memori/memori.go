// Package memori adalah pengisi seam penyimpanan modul auth yang hidup di dalam memori.
//
// Ia ada supaya aturan sesi dan alur masuk dapat diuji tanpa basis data dan tanpa
// jaringan sama sekali — inilah adapter kedua yang membuat seam penyimpanan menjadi
// seam nyata, bukan seam hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: sesi yang hidup di memori satu instans
// melanggar tuntutan stateless (D-27).
package memori

import (
	"context"
	"sync"
	"time"

	"claim-pnc/internal/auth"
)

// PenggunaRepo menyimpan catatan pengguna di memori, dikunci Identitas.
type PenggunaRepo struct {
	mu  sync.RWMutex
	isi map[string]auth.Pengguna
}

// PenggunaRepoBaru membentuk store kosong.
func PenggunaRepoBaru() *PenggunaRepo {
	return &PenggunaRepo{isi: map[string]auth.Pengguna{}}
}

// AmbilByIdentitas membaca satu catatan pengguna.
func (s *PenggunaRepo) AmbilByIdentitas(_ context.Context, identitas string) (auth.Pengguna, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ada := s.isi[identitas]
	if !ada {
		return auth.Pengguna{}, auth.ErrPenggunaTidakDitemukan
	}
	return p, nil
}

// SimpanAtauPerbarui menulis catatan pengguna.
func (s *PenggunaRepo) SimpanAtauPerbarui(_ context.Context, p auth.Pengguna) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lama, ada := s.isi[p.Identitas]; ada {
		// Status aktif dimiliki administrator, bukan sistem identitas luar; ia tidak
		// ikut tertimpa saat profil disegarkan.
		p.Aktif = lama.Aktif
		p.OperatorID = lama.OperatorID
		p.DibuatPada = lama.DibuatPada
	}
	s.isi[p.Identitas] = p
	return nil
}

// SetAktif mengubah status aktif satu pengguna. Dipakai pengujian dan belum punya
// padanan layar administrasi — itu lingkup TKT-F3-004 yang masih terhalang artefak.
func (s *PenggunaRepo) SetAktif(identitas string, aktif bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ada := s.isi[identitas]; ada {
		p.Aktif = aktif
		s.isi[identitas] = p
	}
}

// SesiRepo menyimpan sesi di memori, dikunci sidik token.
type SesiRepo struct {
	mu  sync.RWMutex
	isi map[string]auth.Sesi
}

// SesiRepoBaru membentuk store kosong.
func SesiRepoBaru() *SesiRepo { return &SesiRepo{isi: map[string]auth.Sesi{}} }

// Simpan menuliskan sesi baru.
func (s *SesiRepo) Simpan(_ context.Context, baru auth.Sesi) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isi[baru.SidikToken] = baru
	return nil
}

// AmbilBySidikToken mencari sesi dari sidik tokennya.
func (s *SesiRepo) AmbilBySidikToken(_ context.Context, sidik string) (auth.Sesi, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ditemukan, ada := s.isi[sidik]
	if !ada {
		return auth.Sesi{}, auth.ErrSesiTidakDitemukan
	}
	return ditemukan, nil
}

// Cabut menandai sesi sebagai dicabut tanpa menghapus barisnya.
func (s *SesiRepo) Cabut(_ context.Context, id string, pada time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sidik, item := range s.isi {
		if item.ID == id && item.DicabutPada == nil {
			waktu := pada.UTC()
			item.DicabutPada = &waktu
			s.isi[sidik] = item
		}
	}
	return nil
}

// Perpanjang menggeser batas berlaku sesi yang masih aktif.
func (s *SesiRepo) Perpanjang(_ context.Context, id string, berlakuSampai time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sidik, item := range s.isi {
		if item.ID == id && item.DicabutPada == nil {
			item.BerlakuSampai = berlakuSampai.UTC()
			s.isi[sidik] = item
		}
	}
	return nil
}

// Jumlah mengembalikan banyaknya sesi yang pernah tersimpan, termasuk yang sudah
// dicabut — karena pencabutan tidak menghapus baris.
func (s *SesiRepo) Jumlah() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.isi)
}
