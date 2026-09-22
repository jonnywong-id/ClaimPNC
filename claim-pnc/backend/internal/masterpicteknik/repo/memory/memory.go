// Package memori memenuhi seam penyimpanan dan direktori Master PIC Teknik di dalam
// memori.
//
// Ia ada supaya aturan modul dapat diuji tanpa basis data dan tanpa jaringan sama
// sekali — adapter keduanyalah yang membuat kedua seam menjadi seam nyata, bukan seam
// hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: master yang hilang setiap kali proses
// dijalankan ulang tidak ada gunanya.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterpicteknik"
)

// Repo menyimpan master PIC teknik di memori, dikunci ID operator huruf besar.
type Repo struct {
	mu  sync.RWMutex
	content map[string]masterpicteknik.PICTeknik
}

// RepoBaru membentuk store berisi baris awal yang diberikan.
func NewRepo(initial ...masterpicteknik.PICTeknik) *Repo {
	s := &Repo{content: map[string]masterpicteknik.PICTeknik{}}
	for _, p := range initial {
		clean := p.Clean()
		s.content[masterpicteknik.IDKey(clean.OperatorID)] = clean
	}
	return s
}

// Daftar membaca seluruh PIC teknik, terurut menurut ID operator.
func (s *Repo) List(context.Context) ([]masterpicteknik.PICTeknik, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]masterpicteknik.PICTeknik, 0, len(s.content))
	for _, p := range s.content {
		result = append(result, p)
	}
	// Urutan tetap, sama dengan ORDER BY di sqlstore. Tanpa itu daftar berubah urutan
	// pada setiap permintaan, karena iterasi map Go sengaja acak.
	sort.Slice(result, func(i, j int) bool {
		return masterpicteknik.IDKey(result[i].OperatorID) < masterpicteknik.IDKey(result[j].OperatorID)
	})
	return result, nil
}

// Ambil membaca satu PIC teknik.
func (s *Repo) Get(_ context.Context, operatorID string) (masterpicteknik.PICTeknik, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, exists := s.content[masterpicteknik.IDKey(operatorID)]
	if !exists {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrNotFound
	}
	return p, nil
}

// Sisip menyimpan PIC teknik baru.
func (s *Repo) Insert(_ context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := masterpicteknik.IDKey(p.OperatorID)
	if _, exists := s.content[key]; exists {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrAlreadyExists
	}
	clean := p.Clean()
	s.content[key] = clean
	return clean, nil
}

// Perbarui mengubah PIC teknik yang sudah ada.
func (s *Repo) Update(_ context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := masterpicteknik.IDKey(p.OperatorID)
	previous, exists := s.content[key]
	if !exists {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrNotFound
	}

	clean := p.Clean()
	// GrupPanel tidak pernah ditulis penyimpanan sungguhan; ia dipertahankan di sini
	// juga supaya perilaku kedua adapter tidak berbeda.
	clean.GrupPanel = previous.GrupPanel
	s.content[key] = clean
	return clean, nil
}

// Direktori memenuhi seam DirektoriOperator di dalam memori.
type Directory struct {
	mu   sync.RWMutex
	name map[string]string
}

// DirektoriBaru membentuk direktori dari pasangan ID operator dan namanya.
func NewDirectory(pairs map[string]string) *Directory {
	content := make(map[string]string, len(pairs))
	for id, name := range pairs {
		content[masterpicteknik.IDKey(id)] = strings.TrimSpace(name)
	}
	return &Directory{name: content}
}

// NamaOperator mencari nama petugas.
func (d *Directory) OperatorName(_ context.Context, operatorID string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	name, exists := d.name[masterpicteknik.IDKey(operatorID)]
	if !exists || name == "" {
		return "", masterpicteknik.ErrUnknownOperator
	}
	return name, nil
}

// Daftarkan menambahkan satu operator ke direktori. Dipakai pengujian.
func (d *Directory) Add(operatorID, name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.name[masterpicteknik.IDKey(operatorID)] = strings.TrimSpace(name)
}

var (
	_ masterpicteknik.Repo              = (*Repo)(nil)
	_ masterpicteknik.OperatorDirectory = (*Directory)(nil)
)
