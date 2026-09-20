// Package memory adalah pengisi seam mastermasking.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master Masking
// dapat dicoba lengkap dengan baris yang bentuknya sama dengan produksi.
//
// # Yang ditiru dari Oracle, dan kenapa
//
// Repo ini meniru tiga perilaku yang bila berbeda akan membuat pengujian membuktikan hal
// yang salah:
//
//   - ID dibentuk `MAX+1`, sama dengan `masking_next_id`.
//   - Pencarian cabang menelusuri NAMA cabang, bukan kodenya.
//   - Baris nonaktif tetap tersimpan; "hapus" hanya mengubah statusnya.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/mastermasking"
)

// Repo menyimpan master masking di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu       sync.RWMutex
	rows     map[string]mastermasking.Masking
	branches []mastermasking.Branch
	nextID   int
}

// NewRepo membentuk repo berisi daftar yang diberikan.
//
// Daftar cabangnya ikut diambil dari baris yang diberikan supaya pemeriksaan keberadaan
// cabang tidak menolak data contohnya sendiri. Cabang tambahan disuntikkan lewat
// WithBranches.
func NewRepo(list ...mastermasking.Masking) *Repo {
	r := &Repo{rows: make(map[string]mastermasking.Masking, len(list))}

	seen := map[string]bool{}
	for _, m := range list {
		clean := m.Clean()
		r.rows[clean.ID] = clean
		if n, err := strconv.Atoi(clean.ID); err == nil && n >= r.nextID {
			r.nextID = n + 1
		}
		if clean.BranchID != "" && !seen[clean.BranchID] {
			seen[clean.BranchID] = true
			r.branches = append(r.branches, mastermasking.Branch{
				ID:   clean.BranchID,
				Name: clean.BranchName,
			})
		}
	}
	if r.nextID == 0 {
		r.nextID = 1
	}
	sortBranches(r.branches)
	return r
}

// WithBranches menambahkan pilihan cabang yang belum terpakai baris mana pun.
//
// Tanpa ini, satu-satunya cabang yang dapat dipilih adalah yang sudah punya baris — dan
// menambah data untuk cabang baru menjadi mustahil di mode memori.
func (r *Repo) WithBranches(list ...mastermasking.Branch) *Repo {
	r.mu.Lock()
	defer r.mu.Unlock()

	seen := map[string]bool{}
	for _, b := range r.branches {
		seen[b.ID] = true
	}
	for _, b := range list {
		id := strings.TrimSpace(b.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		r.branches = append(r.branches, mastermasking.Branch{ID: id, Name: strings.TrimSpace(b.Name)})
	}
	sortBranches(r.branches)
	return r
}

// List mengembalikan baris yang cocok dengan penyaring, terbaru lebih dulu.
func (r *Repo) List(ctx context.Context, filter mastermasking.Filter) ([]mastermasking.Masking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filter = filter.Clean()
	keyword := strings.ToUpper(filter.Keyword)
	wantedStatus, statusValid := filter.StatusKeyword()

	var result []mastermasking.Masking
	for _, m := range r.rows {
		switch filter.By {
		case mastermasking.SearchByLogin:
			if !strings.Contains(strings.ToUpper(m.Login), keyword) {
				continue
			}
		case mastermasking.SearchByBranch:
			if !strings.Contains(strings.ToUpper(m.BranchName), keyword) {
				continue
			}
		case mastermasking.SearchByStatus:
			// Cocok PERSIS, bukan sebagian — sama dengan `STS_AKTF = '…'` di layar lama.
			// Status yang tidak sah tidak mengembalikan apa pun; usecase menolaknya lebih
			// dulu, dan ini jaring pengamannya.
			if !statusValid || statusOf(m) != wantedStatus {
				continue
			}
		}
		result = append(result, m)
	}

	// Urutan mengikuti Oracle: terbaru lebih dulu, ID menurun sebagai pemecah seri.
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].InputAt.Equal(result[j].InputAt) {
			return result[i].InputAt.After(result[j].InputAt)
		}
		return numeric(result[i].ID) > numeric(result[j].ID)
	})
	return result, nil
}

// Get mengembalikan satu baris menurut ID.
func (r *Repo) Get(ctx context.Context, id string) (mastermasking.Masking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, found := r.rows[strings.TrimSpace(id)]
	if !found {
		return mastermasking.Masking{}, mastermasking.ErrNotFound
	}
	return m, nil
}

// FindByPair mencari baris menurut pasangan cabang+login.
func (r *Repo) FindByPair(ctx context.Context, branchID, login string) (mastermasking.Masking, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted := mastermasking.PairKey(branchID, login)
	for _, m := range r.rows {
		if mastermasking.PairKey(m.BranchID, m.Login) == wanted {
			return m, nil
		}
	}
	return mastermasking.Masking{}, mastermasking.ErrNotFound
}

// Insert menyimpan baris baru beserta ID yang dibentuk repo.
func (r *Repo) Insert(ctx context.Context, m mastermasking.Masking) (mastermasking.Masking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clean := m.Clean()
	clean.ID = strconv.Itoa(r.nextID)
	clean.BranchName = r.branchNameLocked(clean.BranchID)
	r.nextID++
	r.rows[clean.ID] = clean
	return clean, nil
}

// Update mengubah baris yang sudah ada.
//
// Status aktif IKUT diubah, sama dengan `masking_update` — layar lama memuat isian
// "STATUS" pada form, dan procedure-nya menulis kolom itu.
func (r *Repo) Update(ctx context.Context, m mastermasking.Masking) (mastermasking.Masking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	clean := m.Clean()
	if _, found := r.rows[clean.ID]; !found {
		return mastermasking.Masking{}, mastermasking.ErrNotFound
	}
	clean.BranchName = r.branchNameLocked(clean.BranchID)
	r.rows[clean.ID] = clean
	return clean, nil
}

// SetActive mengaktifkan atau menonaktifkan satu baris.
func (r *Repo) SetActive(ctx context.Context, id string, active bool, by string, at time.Time) (mastermasking.Masking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.TrimSpace(id)
	current, found := r.rows[key]
	if !found {
		return mastermasking.Masking{}, mastermasking.ErrNotFound
	}
	current.Active = active
	current.InputBy = strings.TrimSpace(by)
	current.InputAt = at
	r.rows[key] = current
	return current, nil
}

// ListBranches mengembalikan pilihan cabang yang cocok dengan kata kunci.
func (r *Repo) ListBranches(ctx context.Context, keyword string, limit int) ([]mastermasking.Branch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 1
	}
	wanted := strings.ToUpper(strings.TrimSpace(keyword))

	var result []mastermasking.Branch
	for _, b := range r.branches {
		if wanted != "" &&
			!strings.Contains(strings.ToUpper(b.Name), wanted) &&
			!strings.Contains(strings.ToUpper(b.ID), wanted) {
			continue
		}
		result = append(result, b)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

// BranchExists menyatakan apakah kode cabang dikenal.
func (r *Repo) BranchExists(ctx context.Context, branchID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wanted := strings.TrimSpace(branchID)
	for _, b := range r.branches {
		if b.ID == wanted {
			return true, nil
		}
	}
	return false, nil
}

// branchNameLocked mencari nama cabang. Pemanggil wajib sudah memegang kunci.
//
// Nama cabang tidak pernah disimpan di baris masking — di Oracle ia hasil LEFT JOIN yang
// dihitung ulang setiap pembacaan. Repo ini melakukan hal yang sama supaya keduanya
// berperilaku sama.
func (r *Repo) branchNameLocked(branchID string) string {
	for _, b := range r.branches {
		if b.ID == branchID {
			return b.Name
		}
	}
	return ""
}

// statusOf mengembalikan teks status sebuah baris, sama dengan yang tersimpan di kolom
// STS_AKTF pada Oracle. Ia ada supaya penyaring status di sini dan di sana membandingkan
// hal yang sama persis.
func statusOf(m mastermasking.Masking) string {
	if m.Active {
		return mastermasking.StatusActive
	}
	return mastermasking.StatusInactive
}

// numeric membaca ID sebagai angka untuk keperluan pengurutan; nol bila bukan angka.
func numeric(id string) int {
	n, err := strconv.Atoi(strings.TrimSpace(id))
	if err != nil {
		return 0
	}
	return n
}

// sortBranches mengurutkan pilihan cabang menurut nama, sama dengan `branch_list`.
func sortBranches(list []mastermasking.Branch) {
	sort.SliceStable(list, func(i, j int) bool { return list[i].Name < list[j].Name })
}
