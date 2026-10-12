// Package memory adalah pengisi seam masterdominanfactor.Repo yang hidup di dalam
// memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master Dominan
// Factor dapat dicoba lengkap sebelum koneksi ke basis data entitas tersedia.
package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/masterdominanfactor"
)

// Repo menyimpan master faktor dominan di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	rows   map[string]masterdominanfactor.DominantFactor
	issues error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...masterdominanfactor.DominantFactor) *Repo {
	r := &Repo{rows: make(map[string]masterdominanfactor.DominantFactor, len(list))}
	for _, f := range list {
		clean := f.Clean()
		r.rows[clean.ID] = clean
	}
	return r
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issues = err
}

// List mengembalikan seluruh faktor, terurut numerik menurut ID.
func (r *Repo) List(_ context.Context) ([]masterdominanfactor.DominantFactor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := make([]masterdominanfactor.DominantFactor, 0, len(r.rows))
	for _, f := range r.rows {
		result = append(result, f)
	}
	// Pengurutan memakai fungsi domain yang sama dengan adapter SQL, supaya keduanya
	// tidak dapat berbeda pendapat tentang urutan yang benar.
	masterdominanfactor.SortByID(result)
	return result, nil
}

// Get mengembalikan satu faktor.
func (r *Repo) Get(_ context.Context, id string) (masterdominanfactor.DominantFactor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterdominanfactor.DominantFactor{}, r.issues
	}

	f, existing := r.rows[strings.TrimSpace(id)]
	if !existing {
		return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrNotFound
	}
	return f, nil
}

// Insert menyimpan faktor baru dengan ID yang dibentuk seperti sistem lama.
//
// Penguncian di sini setara dengan FOR UPDATE pada adapter SQL: pembacaan nomor
// tertinggi dan penyisipannya tidak dapat disela penyimpanan lain.
func (r *Repo) Insert(_ context.Context, name string) (masterdominanfactor.DominantFactor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterdominanfactor.DominantFactor{}, r.issues
	}

	existing := make([]string, 0, len(r.rows))
	for id := range r.rows {
		existing = append(existing, id)
	}

	id := masterdominanfactor.NextID(existing)
	if _, taken := r.rows[id]; taken {
		return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrIDTaken
	}

	// Nama kosong TIDAK ditolak di sini, dan itu bukan kelalaian adapter: modul ini
	// memang menerimanya (keputusan Work Owner 2026-09-20). Menolaknya di sini akan
	// membuat adapter memori berperilaku berbeda dari Oracle, dan pengujian yang lolos
	// di sini akan gagal di sana.
	factor := masterdominanfactor.DominantFactor{ID: id, Name: strings.TrimSpace(name)}
	r.rows[id] = factor
	return factor, nil
}

// Update mengganti nama faktor yang sudah ada.
func (r *Repo) Update(_ context.Context, id, name string) (masterdominanfactor.DominantFactor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterdominanfactor.DominantFactor{}, r.issues
	}

	id = strings.TrimSpace(id)
	previous, existing := r.rows[id]
	if !existing {
		return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrNotFound
	}

	// Hanya nama yang berubah; ID dipertahankan apa adanya.
	previous.Name = strings.TrimSpace(name)
	r.rows[id] = previous
	return previous, nil
}

var _ masterdominanfactor.Repo = (*Repo)(nil)
