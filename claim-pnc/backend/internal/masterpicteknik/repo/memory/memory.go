// Package memory adalah pengisi seam masterpicteknik.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master PIC
// Teknik dapat dicoba lengkap — termasuk jalur "petugas nonaktif tidak muncul di daftar"
// yang justru paling mudah terlewat bila hanya diuji dengan data aktif.
package memory

import (
	"context"
	"sort"
	"sync"

	"claim-pnc/internal/masterpicteknik"
)

// Repo menyimpan master PIC teknik di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	rows   map[string]masterpicteknik.Technician
	issues error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...masterpicteknik.Technician) *Repo {
	r := &Repo{rows: make(map[string]masterpicteknik.Technician, len(list))}
	for _, t := range list {
		clean := t.Clean()
		r.rows[masterpicteknik.IDKey(clean.OperatorID)] = clean
	}
	return r
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issues = err
}

// List mengembalikan petugas AKTIF saja, terurut menurut ID operator.
//
// Penyaringan ada di sini — bukan di usecase — supaya kedua pengisi seam berperilaku
// sama. Di sqlstore penyaringnya bagian dari kueri; di sini ia harus ditiru, kalau tidak
// pengujian terhadap memori akan meloloskan cacat yang muncul terhadap Oracle.
func (r *Repo) List(_ context.Context) ([]masterpicteknik.Technician, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := make([]masterpicteknik.Technician, 0, len(r.rows))
	for _, t := range r.rows {
		if !t.Active {
			continue
		}
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].OperatorID < result[j].OperatorID
	})
	return result, nil
}

// Get mengembalikan satu petugas, aktif maupun tidak.
func (r *Repo) Get(_ context.Context, operatorID string) (masterpicteknik.Technician, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterpicteknik.Technician{}, r.issues
	}

	t, existing := r.rows[masterpicteknik.IDKey(operatorID)]
	if !existing {
		return masterpicteknik.Technician{}, masterpicteknik.ErrNotFound
	}
	return t, nil
}

// Insert menyimpan petugas baru.
func (r *Repo) Insert(_ context.Context, t masterpicteknik.Technician) (masterpicteknik.Technician, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterpicteknik.Technician{}, r.issues
	}

	t = t.Clean()
	key := masterpicteknik.IDKey(t.OperatorID)
	if _, clash := r.rows[key]; clash {
		return masterpicteknik.Technician{}, masterpicteknik.ErrAlreadyExists
	}

	r.rows[key] = t
	return t, nil
}

// Update mengubah petugas yang sudah ada.
func (r *Repo) Update(_ context.Context, t masterpicteknik.Technician) (masterpicteknik.Technician, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterpicteknik.Technician{}, r.issues
	}

	t = t.Clean()
	key := masterpicteknik.IDKey(t.OperatorID)
	previous, existing := r.rows[key]
	if !existing {
		return masterpicteknik.Technician{}, masterpicteknik.ErrNotFound
	}

	// ID operator dipertahankan apa adanya dari baris yang tersimpan, bukan diambil dari
	// masukan: besar-kecil hurufnya milik baris itu, dan menimpanya akan membuat ID
	// berubah bentuk setiap kali seseorang mengetiknya berbeda.
	t.OperatorID = previous.OperatorID
	// Dua nilai yang tidak dikelola layar ini tidak pernah berubah lewat Update.
	t.PanelGroup = previous.PanelGroup
	t.Workload = previous.Workload

	r.rows[key] = t
	return t, nil
}

var _ masterpicteknik.Repo = (*Repo)(nil)
