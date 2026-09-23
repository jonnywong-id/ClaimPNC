// Package memory adalah pengisi seluruh seam penyimpanan modul registrasi yang hidup di
// dalam memori.
//
// Ia ada supaya modul ini dapat dijalankan dan diuji tanpa basis data — adapter kedua
// yang membuat seam-nya nyata, bukan hipotetis. Tabel klaim dan tugas baru ada setelah
// DBA menjalankan migrasi `0002`; sampai itu terjadi, inilah satu-satunya jalan
// menjalankan alur Register dari ujung ke ujung.
//
// # Yang ditiru dengan sungguh-sungguh, dan kenapa
//
// Batas transaksi. `Jalankan` mengambil salinan seluruh penyimpanan sebelum kerja
// dimulai dan MENGEMBALIKANNYA bila kerja gagal. Tanpa itu, uji "penyimpanan yang gagal
// tidak meninggalkan satu baris pun" (`TKT-B02-001`) akan lulus di memori dan gagal di
// Oracle — dan yang gagal di produksi bukan ujinya, melainkan klaimnya.
package memory

import (
	"context"
	"sort"
	"sync"

	"claim-pnc/internal/registrasi"
)

type contextKey string

// txKey menandai context yang sudah berada di dalam unit kerja. Repo yang
// melihat tanda ini tidak mengunci ulang — mutex Go tidak reentrant, dan mengunci dua
// kali dari goroutine yang sama membekukan permintaan selamanya.
const txKey contextKey = "registrasi_memori_transaksi"

// Store memegang klaim, tugas, dan jejak audit di memori.
type Store struct {
	mu sync.Mutex

	claim    map[string]registrasi.Claim
	task     map[string]registrasi.Task
	audit    []registrasi.AuditTrail
	messages []registrasi.Notification
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store {
	return &Store{
		claim: map[string]registrasi.Claim{},
		task:  map[string]registrasi.Task{},
	}
}

func (p *Store) key(ctx context.Context) func() {
	if ctx.Value(txKey) != nil {
		return func() {}
	}
	p.mu.Lock()
	return p.mu.Unlock
}

// Run menjalankan kerja di dalam satu batas transaksi.
func (p *Store) Run(ctx context.Context, work func(context.Context) error) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	claimCopy := make(map[string]registrasi.Claim, len(p.claim))
	for k, v := range p.claim {
		claimCopy[k] = v
	}
	taskCopy := make(map[string]registrasi.Task, len(p.task))
	for k, v := range p.task {
		taskCopy[k] = v
	}
	auditCopy := append([]registrasi.AuditTrail(nil), p.audit...)
	messageCopy := append([]registrasi.Notification(nil), p.messages...)

	if err := work(context.WithValue(ctx, txKey, true)); err != nil {
		p.claim = claimCopy
		p.task = taskCopy
		p.audit = auditCopy
		p.messages = messageCopy
		return err
	}
	return nil
}

// ── ClaimRepo ────────────────────────────────────────────────────────────────────

// Save menuliskan klaim beserta seluruh pohon di bawahnya.
func (p *Store) Save(ctx context.Context, k registrasi.Claim) error {
	defer p.key(ctx)()
	p.claim[k.ID] = copyClaim(k)
	return nil
}

// Get mengembalikan klaim berdasarkan pengenal internalnya.
func (p *Store) Get(ctx context.Context, id string) (registrasi.Claim, error) {
	defer p.key(ctx)()
	k, ok := p.claim[id]
	if !ok {
		return registrasi.Claim{}, registrasi.ErrClaimNotFound
	}
	return copyClaim(k), nil
}

// GetByNumber mengembalikan klaim berdasarkan nomor klaimnya.
func (p *Store) GetByNumber(ctx context.Context, number string) (registrasi.Claim, error) {
	defer p.key(ctx)()
	for _, k := range p.claim {
		if k.Number == number {
			return copyClaim(k), nil
		}
	}
	return registrasi.Claim{}, registrasi.ErrClaimNotFound
}

// FindDuplicates mencari klaim lain yang memenuhi salah satu kunci duplikasi.
//
// Klaim yang ditandai terhapus tidak ikut terhitung (`ADR-0012`), dan klaim yang belum
// bernomor juga tidak: klaim tanpa nomor belum benar-benar terdaftar, dan menyebutnya
// dalam pesan galat akan menyuruh petugas memeriksa sesuatu yang tidak dapat ia cari.
func (p *Store) FindDuplicates(ctx context.Context, key []registrasi.DuplicateKey, exceptID string) ([]registrasi.DuplicateClaim, error) {
	defer p.key(ctx)()

	var result []registrasi.DuplicateClaim
	seen := map[string]bool{}

	// Urutan iterasi map Go acak; hasilnya diurutkan supaya pesan galat yang sama
	// selalu muncul dalam urutan yang sama.
	id := make([]string, 0, len(p.claim))
	for k := range p.claim {
		id = append(id, k)
	}
	sort.Strings(id)

	for _, i := range id {
		other := p.claim[i]
		if other.ID == exceptID || other.Deleted() || other.Number == "" {
			continue
		}
		for _, dupKey := range key {
			insuredItem, match := matchKey(other, dupKey)
			if !match {
				continue
			}
			marker := other.Number + "|" + insuredItem
			if seen[marker] {
				continue
			}
			seen[marker] = true
			result = append(result, registrasi.DuplicateClaim{Number: other.Number, InsuredItem: insuredItem})
		}
	}
	return result, nil
}

func matchKey(k registrasi.Claim, dupKey registrasi.DuplicateKey) (string, bool) {
	if k.Policy.Number != dupKey.PolicyNumber {
		return "", false
	}
	if dupKey.Location != "" && !sameText(k.Location, dupKey.Location) {
		return "", false
	}
	for _, o := range k.InsuredItem {
		if o.ID != dupKey.InsuredItemID {
			continue
		}
		if dupKey.CauseOfLoss == "" {
			return o.ID, true
		}
		for _, c := range o.Coverage {
			if c.CauseOfLoss == dupKey.CauseOfLoss {
				return o.ID, true
			}
		}
	}
	return "", false
}

// ── TaskRepo ────────────────────────────────────────────────────────────────────

// SaveTask menuliskan satu tugas.
//
// Namanya berbeda dari Simpan milik ClaimRepo karena satu tipe Go tidak dapat punya dua
// metode bernama sama. Pembungkusnya ada di TaskRepo() di bawah.
func (p *Store) SaveTask(ctx context.Context, t registrasi.Task) error {
	defer p.key(ctx)()
	p.task[t.ID] = t
	return nil
}

// ClaimTask mengembalikan satu tugas.
func (p *Store) ClaimTask(ctx context.Context, id string) (registrasi.Task, error) {
	defer p.key(ctx)()
	t, ok := p.task[id]
	if !ok {
		return registrasi.Task{}, registrasi.ErrTaskNotFound
	}
	return t, nil
}

// OpenTaskForClaim mengembalikan tugas yang masih menunggu untuk sebuah klaim.
func (p *Store) OpenTaskForClaim(ctx context.Context, claimID string) (registrasi.Task, error) {
	defer p.key(ctx)()
	for _, t := range sortTasks(p.task) {
		if t.ClaimID == claimID && t.Open() {
			return t, nil
		}
	}
	return registrasi.Task{}, registrasi.ErrTaskNotFound
}

// Inbox mengembalikan pekerjaan yang menunggu seorang pengguna.
func (p *Store) Inbox(ctx context.Context, operator string, workbasket []string) ([]registrasi.Task, error) {
	defer p.key(ctx)()

	allowed := map[string]bool{}
	for _, w := range workbasket {
		allowed[w] = true
	}

	var result []registrasi.Task
	for _, t := range sortTasks(p.task) {
		if !t.Open() {
			continue
		}
		switch {
		case t.Queue == registrasi.QueueWorklist && t.Owner == operator:
			result = append(result, t)
		case t.Queue == registrasi.QueueWorkbasket && !t.Owned() && allowed[t.Workbasket]:
			result = append(result, t)
		case t.Queue == registrasi.QueueWorkbasket && t.Owner == operator:
			result = append(result, t)
		}
	}
	return result, nil
}

// ── AuditRecorder dan Notifier ────────────────────────────────────────────────────

// Record menambahkan satu baris jejak audit. Baris tidak pernah diubah maupun dihapus
// (`ADR-0026`).
func (p *Store) Record(ctx context.Context, j registrasi.AuditTrail) error {
	defer p.key(ctx)()
	p.audit = append(p.audit, j)
	return nil
}

// Send MENCATAT peristiwa pemberitahuan; ia tidak mengirim apa pun. Lihat kontrak seam
// Notifier.
func (p *Store) Send(ctx context.Context, m registrasi.Notification) error {
	defer p.key(ctx)()
	p.messages = append(p.messages, m)
	return nil
}

// AuditTrail mengembalikan seluruh jejak yang terekam. Dipakai pengujian.
func (p *Store) AuditTrail() []registrasi.AuditTrail {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]registrasi.AuditTrail(nil), p.audit...)
}

// Notification mengembalikan seluruh peristiwa yang tercatat. Dipakai pengujian.
func (p *Store) Notification() []registrasi.Notification {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]registrasi.Notification(nil), p.messages...)
}

// ── Pembungkus supaya satu penyimpanan memenuhi dua seam ─────────────────────────

// TaskRepo membungkus penyimpanan menjadi registrasi.TaskRepo.
func (p *Store) TaskRepo() registrasi.TaskRepo { return taskRepo{p} }

type taskRepo struct{ p *Store }

func (t taskRepo) Save(ctx context.Context, task registrasi.Task) error {
	return t.p.SaveTask(ctx, task)
}

func (t taskRepo) Get(ctx context.Context, id string) (registrasi.Task, error) {
	return t.p.ClaimTask(ctx, id)
}

func (t taskRepo) OpenTaskForClaim(ctx context.Context, claimID string) (registrasi.Task, error) {
	return t.p.OpenTaskForClaim(ctx, claimID)
}

func (t taskRepo) Inbox(ctx context.Context, operator string, workbasket []string) ([]registrasi.Task, error) {
	return t.p.Inbox(ctx, operator, workbasket)
}

func sortTasks(m map[string]registrasi.Task) []registrasi.Task {
	result := make([]registrasi.Task, 0, len(m))
	for _, t := range m {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].CreatedAt.Equal(result[j].CreatedAt) {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].ID < result[j].ID
	})
	return result
}

// copyClaim menyalin pohon objek–coverage–spreading.
//
// Tanpa salinan dalam, pemanggil yang mengubah slice objek akan ikut mengubah isi
// penyimpanan tanpa melewati Simpan — kebocoran yang tidak mungkin terjadi pada adapter
// SQL, dan karena itu tidak boleh mungkin terjadi di sini.
func copyClaim(k registrasi.Claim) registrasi.Claim {
	copy := k
	copy.InsuredItem = make([]registrasi.InsuredItem, 0, len(k.InsuredItem))
	for _, o := range k.InsuredItem {
		insuredItem := o
		insuredItem.Coverage = make([]registrasi.Coverage, 0, len(o.Coverage))
		for _, c := range o.Coverage {
			coverage := c
			coverage.Spreading = append([]registrasi.Spreading(nil), c.Spreading...)
			insuredItem.Coverage = append(insuredItem.Coverage, coverage)
		}
		copy.InsuredItem = append(copy.InsuredItem, insuredItem)
	}
	if k.DeletedAt != nil {
		t := *k.DeletedAt
		copy.DeletedAt = &t
	}
	return copy
}

var (
	_ registrasi.ClaimRepo     = (*Store)(nil)
	_ registrasi.UnitOfWork    = (*Store)(nil)
	_ registrasi.AuditRecorder = (*Store)(nil)
	_ registrasi.Notifier      = (*Store)(nil)
	_ registrasi.TaskRepo      = taskRepo{}
)
