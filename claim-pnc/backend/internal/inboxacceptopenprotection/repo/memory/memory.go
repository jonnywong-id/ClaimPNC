// Package memory memenuhi seam inboxacceptopenprotection.Repo di dalam proses, tanpa basis
// data.
//
// Ia dipakai pengembangan lokal (`PENYIMPANAN=memori`) dan seluruh pengujian. Untuk modul
// ini ia juga satu-satunya penyimpanan yang dapat menjalankan layar hari ini, karena tabel
// `POOLDATA.T_CLAIM_OPENPROTECTION` belum dibuat.
//
// Aturan penyaringan di sini menirukan bentuk kueri yang direncanakan sedekat mungkin —
// bukan disederhanakan — supaya yang teruji adalah penyaring yang sesungguhnya berjalan,
// bukan penyaring di layar.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// Repo menyimpan proteksi di memori.
type Repo struct {
	mu          sync.Mutex
	protections []inboxacceptopenprotection.Protection
}

// NewRepo membentuk repo kosong.
func NewRepo() *Repo {
	return &Repo{}
}

// NewRepoWithSamples membentuk repo berisi proteksi contoh.
//
// Data contohnya KARANGAN (`D-69`).
func NewRepoWithSamples() *Repo {
	r := NewRepo()
	r.protections = sampleProtections()
	return r
}

// Add menambahkan proteksi. Dipakai pengujian.
func (r *Repo) Add(protections ...inboxacceptopenprotection.Protection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.protections = append(r.protections, protections...)
}

// List membaca satu halaman antrean akseptasi.
func (r *Repo) List(
	ctx context.Context,
	f inboxacceptopenprotection.Filter,
) (inboxacceptopenprotection.Page, error) {
	if err := ctx.Err(); err != nil {
		return inboxacceptopenprotection.Page{}, err
	}

	f = f.Normalize()
	wantType, equal := f.Queue.TypeFilter()

	r.mu.Lock()
	defer r.mu.Unlock()

	matched := make([]inboxacceptopenprotection.Protection, 0, len(r.protections))
	for _, p := range r.protections {
		// Ketiga syarat di bawah adalah DEFINISI layar ini, bukan pilihan pengguna:
		// `InboxOpenProtection2_RD` menyaring
		// `CaseID IS NOT NULL AND PolicyNo IS NOT NULL AND AcceptStatus IS NULL`.
		if !p.Pending() {
			continue
		}
		if strings.TrimSpace(p.ClaimNumber) == "" || strings.TrimSpace(p.PolicyNumber) == "" {
			continue
		}

		isPremium := strings.TrimSpace(p.Type) == wantType
		if isPremium != equal {
			continue
		}
		if !matches(p, f.Search) {
			continue
		}
		matched = append(matched, p)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].InputDate.Equal(matched[j].InputDate) {
			return matched[i].InputDate.After(matched[j].InputDate)
		}
		return matched[i].Number > matched[j].Number
	})

	total := len(matched)
	if f.Offset >= total {
		return inboxacceptopenprotection.Page{
			Protections: []inboxacceptopenprotection.Protection{},
			Total:       total,
		}, nil
	}

	end := f.Offset + f.Limit
	if end > total {
		end = total
	}

	page := make([]inboxacceptopenprotection.Protection, end-f.Offset)
	copy(page, matched[f.Offset:end])

	return inboxacceptopenprotection.Page{Protections: page, Total: total}, nil
}

// Get membaca satu proteksi menurut nomornya, TANPA menyaring status akseptasi.
func (r *Repo) Get(ctx context.Context, number string) (inboxacceptopenprotection.Protection, error) {
	if err := ctx.Err(); err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.indexOf(number)
	if index < 0 {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrNotFound
	}
	return r.protections[index], nil
}

// Decide menuliskan keputusan akseptasi.
//
// Pemeriksaan "belum diputuskan" DIULANG di dalam kunci. Itulah yang benar-benar menahan
// petugas kedua pada antrean bersama — pemeriksaan di usecase dilewati keduanya.
func (r *Repo) Decide(
	ctx context.Context,
	number string,
	d inboxacceptopenprotection.Decision,
	by string,
	at time.Time,
) (inboxacceptopenprotection.Protection, error) {
	if err := ctx.Err(); err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}
	if !d.Valid() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrUnknownDecision
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	index := r.indexOf(number)
	if index < 0 {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrNotFound
	}

	existing := r.protections[index]
	if !existing.Pending() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrAlreadyDecided
	}

	// TIGA kolom, dan hanya tiga. Kolom pembuatan dimiliki modul inputreqprotection dan
	// tidak boleh disentuh di sini (`P-1`).
	decidedAt := at
	existing.AcceptStatus = d.Status()
	existing.AcceptedAt = &decidedAt
	existing.AcceptedBy = by

	r.protections[index] = existing
	return existing, nil
}

// indexOf mencari proteksi menurut nomornya. Pemanggil WAJIB memegang kunci.
func (r *Repo) indexOf(number string) int {
	target := normalize(number)
	if target == "" {
		return -1
	}
	for i, p := range r.protections {
		if normalize(p.Number) == target {
			return i
		}
	}
	return -1
}

func matches(p inboxacceptopenprotection.Protection, search string) bool {
	needle := normalize(search)
	if needle == "" {
		return true
	}
	return strings.Contains(normalize(p.Number), needle) ||
		strings.Contains(normalize(p.PolicyNumber), needle) ||
		strings.Contains(normalize(p.ClaimNumber), needle)
}

func normalize(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}
