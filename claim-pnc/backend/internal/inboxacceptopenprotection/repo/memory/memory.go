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

	// claims meniru `POOLDATA.T_CLAIM_PNC` sejauh yang disentuh modul ini:
	// `CLAIMID` -> `DATEOFLOSS`.
	//
	// Dikunci CLAIMID (ID_CLAIM pada proteksi), bukan nomor klaim — sama dengan kuerinya.
	// Klaim yang tidak terdaftar berperilaku seperti klaim yang tidak punya baris di sana.
	claims map[string]time.Time
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

	// Klaim contoh, meniru baris `T_CLAIM_PNC` — dikunci CLAIMID, bukan nomor klaim.
	//
	// Klaim sistem baru: CLAIMID sama dengan nomor klaimnya.
	// Baris warisan `OPC-216` menunjuk klaim Pega, yang CLAIMID-nya berawalan
	// `ASM-FW-GCNMFW-WORK ` — sengaja TIDAK didaftarkan, supaya jalur `ErrClaimNotSynced`
	// benar-benar terlihat saat pengembangan lokal dan bukan hanya ada di uji.
	wib := time.FixedZone("WIB", 7*60*60)
	for _, claimID := range []string{
		"PNCN.26.0007", "PNCN.26.0008", "PNCN.26.0009", "PNCN.26.0010", "PNCN.26.0012",
	} {
		r.AddClaim(claimID, time.Date(2026, time.August, 3, 0, 0, 0, 0, wib))
	}

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

	// Penerapan ke klaim ditiru supaya uji atas usecase bermakna.
	//
	// Adapter Oracle menjalankan keduanya dalam SATU transaksi; di sini urutannya ditiru
	// dengan menyimpan perubahan proteksi HANYA setelah penerapannya berhasil. Tanpa itu,
	// uji akan lolos pada perilaku yang tidak pernah terjadi di produksi.
	if tanggal, perlu := inboxacceptopenprotection.LossDateToApply(existing, d); perlu {
		if r.claims == nil {
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrClaimNotSynced
		}
		if _, ada := r.claims[normalize(existing.ClaimReference)]; !ada {
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrClaimNotSynced
		}
		r.claims[normalize(existing.ClaimReference)] = tanggal
	}

	r.protections[index] = existing
	return existing, nil
}

// AddClaim mendaftarkan klaim beserta Tanggal Kejadiannya, meniru satu baris
// `POOLDATA.T_CLAIM_PNC` — dikunci CLAIMID.
//
// Klaim yang TIDAK didaftarkan berperilaku seperti klaim tanpa baris di sana:
// keputusan atasnya ditolak `ErrClaimNotSynced`. Keadaan itu perlu dapat diuji — tabelnya
// baru memuat 1.014 dari 7.703 klaim.
func (r *Repo) AddClaim(claimID string, lossDate time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.claims == nil {
		r.claims = map[string]time.Time{}
	}
	r.claims[normalize(claimID)] = lossDate
}

// LossDateOf mengembalikan Tanggal Kejadian klaim yang tersimpan.
//
// Dipakai uji untuk membuktikan penerapannya benar-benar terjadi — bukan sekadar tidak
// bergalat.
func (r *Repo) LossDateOf(claimID string) (time.Time, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ada := r.claims[normalize(claimID)]
	return t, ada
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
