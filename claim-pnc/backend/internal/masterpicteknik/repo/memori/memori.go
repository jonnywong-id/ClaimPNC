// Package memori memenuhi seam penyimpanan dan direktori Master PIC Teknik di dalam
// memori.
//
// Ia ada supaya aturan modul dapat diuji tanpa basis data dan tanpa jaringan sama
// sekali — adapter keduanyalah yang membuat kedua seam menjadi seam nyata, bukan seam
// hipotetis (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: master yang hilang setiap kali proses
// dijalankan ulang tidak ada gunanya.
package memori

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
	isi map[string]masterpicteknik.PICTeknik
}

// RepoBaru membentuk store berisi baris awal yang diberikan.
func RepoBaru(awal ...masterpicteknik.PICTeknik) *Repo {
	s := &Repo{isi: map[string]masterpicteknik.PICTeknik{}}
	for _, p := range awal {
		bersih := p.Bersih()
		s.isi[masterpicteknik.KunciID(bersih.IDOperator)] = bersih
	}
	return s
}

// Daftar membaca seluruh PIC teknik, terurut menurut ID operator.
func (s *Repo) Daftar(context.Context) ([]masterpicteknik.PICTeknik, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	hasil := make([]masterpicteknik.PICTeknik, 0, len(s.isi))
	for _, p := range s.isi {
		hasil = append(hasil, p)
	}
	// Urutan tetap, sama dengan ORDER BY di sqlstore. Tanpa itu daftar berubah urutan
	// pada setiap permintaan, karena iterasi map Go sengaja acak.
	sort.Slice(hasil, func(i, j int) bool {
		return masterpicteknik.KunciID(hasil[i].IDOperator) < masterpicteknik.KunciID(hasil[j].IDOperator)
	})
	return hasil, nil
}

// Ambil membaca satu PIC teknik.
func (s *Repo) Ambil(_ context.Context, idOperator string) (masterpicteknik.PICTeknik, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ada := s.isi[masterpicteknik.KunciID(idOperator)]
	if !ada {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrTidakDitemukan
	}
	return p, nil
}

// Sisip menyimpan PIC teknik baru.
func (s *Repo) Sisip(_ context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kunci := masterpicteknik.KunciID(p.IDOperator)
	if _, ada := s.isi[kunci]; ada {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrSudahAda
	}
	bersih := p.Bersih()
	s.isi[kunci] = bersih
	return bersih, nil
}

// Perbarui mengubah PIC teknik yang sudah ada.
func (s *Repo) Perbarui(_ context.Context, p masterpicteknik.PICTeknik) (masterpicteknik.PICTeknik, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	kunci := masterpicteknik.KunciID(p.IDOperator)
	lama, ada := s.isi[kunci]
	if !ada {
		return masterpicteknik.PICTeknik{}, masterpicteknik.ErrTidakDitemukan
	}

	bersih := p.Bersih()
	// GrupPanel tidak pernah ditulis penyimpanan sungguhan; ia dipertahankan di sini
	// juga supaya perilaku kedua adapter tidak berbeda.
	bersih.GrupPanel = lama.GrupPanel
	s.isi[kunci] = bersih
	return bersih, nil
}

// Direktori memenuhi seam DirektoriOperator di dalam memori.
type Direktori struct {
	mu   sync.RWMutex
	nama map[string]string
}

// DirektoriBaru membentuk direktori dari pasangan ID operator dan namanya.
func DirektoriBaru(pasangan map[string]string) *Direktori {
	isi := make(map[string]string, len(pasangan))
	for id, nama := range pasangan {
		isi[masterpicteknik.KunciID(id)] = strings.TrimSpace(nama)
	}
	return &Direktori{nama: isi}
}

// NamaOperator mencari nama petugas.
func (d *Direktori) NamaOperator(_ context.Context, idOperator string) (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	nama, ada := d.nama[masterpicteknik.KunciID(idOperator)]
	if !ada || nama == "" {
		return "", masterpicteknik.ErrOperatorTidakDikenal
	}
	return nama, nil
}

// Daftarkan menambahkan satu operator ke direktori. Dipakai pengujian.
func (d *Direktori) Daftarkan(idOperator, nama string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nama[masterpicteknik.KunciID(idOperator)] = strings.TrimSpace(nama)
}

var (
	_ masterpicteknik.Repo              = (*Repo)(nil)
	_ masterpicteknik.DirektoriOperator = (*Direktori)(nil)
)
