// Package memory memenuhi seam inboxxol.Repo tanpa basis data.
//
// Dua kegunaan, dan keduanya nyata:
//
//   - Menjalankan aplikasi tanpa Oracle saat pengembangan, sehingga seluruh tab layar
//     Inbox XOL dapat dicoba sebelum DBA membuka akses ke tabel XOL.
//   - Menguji aturan modul tanpa basis data, tanpa jaringan, dan tanpa berkas — itulah
//     yang membuat uji aturan bisnis cepat (`14-TESTING-STRATEGY.md` §3).
//
// # Seluruh isinya KARANGAN
//
// Tidak satu pun angka di sample.go berasal dari data produksi. Nomor PLA/DLA, nama
// reasuradur, dan nilai klaimnya dibuat supaya seluruh jalur layar dapat ditempuh —
// termasuk jalur yang paling mudah terlewat: perjanjian tanpa group business, dan baris
// treaty inward yang kursnya tidak ditemukan.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/inboxxol"
)

// Repo menyimpan data Inbox XOL di memori.
//
// Aman dipakai bersamaan: seluruh pembacaan menempuh satu kunci baca. Meski tidak ada
// satu pun operasi yang menulis hari ini, kuncinya tetap ada — penyimpanan yang dibaca
// banyak permintaan HTTP sekaligus tanpa kunci adalah cacat yang hanya muncul di bawah
// beban, dan justru tidak terlihat saat diuji.
type Repo struct {
	mu sync.RWMutex

	masters     []inboxxol.MasterXOL
	summaries   map[string][]inboxxol.ClaimSummary
	breakdowns  map[string][]inboxxol.BusinessBreakdown
	treaty      map[string][]inboxxol.BusinessBreakdown
	advices     []inboxxol.Advice
	causeOfLoss []inboxxol.CauseOfLoss
}

// Option mengubah isi penyimpanan saat dibentuk.
type Option func(*Repo)

// NewRepo membentuk penyimpanan kosong yang diisi Option.
func NewRepo(options ...Option) *Repo {
	repo := &Repo{
		summaries:  map[string][]inboxxol.ClaimSummary{},
		breakdowns: map[string][]inboxxol.BusinessBreakdown{},
		treaty:     map[string][]inboxxol.BusinessBreakdown{},
	}
	for _, option := range options {
		option(repo)
	}
	return repo
}

// WithMasters mengisi daftar perjanjian XOL.
func WithMasters(masters ...inboxxol.MasterXOL) Option {
	return func(r *Repo) { r.masters = masters }
}

// WithSummaries mengisi akumulasi klaim satu tahun perjanjian.
//
// Kuncinya TAHUN, bukan kode perjanjian: kueri sistem lama pun menyaring menurut tahun,
// dan dua perjanjian bertahun sama akan membaca akumulasi yang sama.
func WithSummaries(year string, rows ...inboxxol.ClaimSummary) Option {
	return func(r *Repo) { r.summaries[strings.TrimSpace(year)] = rows }
}

// WithBreakdown mengisi rincian klaim milik sendiri untuk satu tanggal dan sebab.
func WithBreakdown(lossDate, causeOfLoss string, rows ...inboxxol.BusinessBreakdown) Option {
	return func(r *Repo) { r.breakdowns[breakdownKey(lossDate, causeOfLoss)] = rows }
}

// WithTreatyInward mengisi rincian treaty inward untuk satu tanggal dan sebab.
func WithTreatyInward(lossDate, causeOfLoss string, rows ...inboxxol.BusinessBreakdown) Option {
	return func(r *Repo) { r.treaty[breakdownKey(lossDate, causeOfLoss)] = rows }
}

// WithAdvices mengisi pemberitahuan PLA/DLA yang sudah diterbitkan.
func WithAdvices(advices ...inboxxol.Advice) Option {
	return func(r *Repo) { r.advices = advices }
}

// WithCauseOfLoss mengisi daftar Penyebab Kerugian.
func WithCauseOfLoss(causes ...inboxxol.CauseOfLoss) Option {
	return func(r *Repo) { r.causeOfLoss = causes }
}

// ListMasterXOL mengembalikan seluruh perjanjian XOL.
func (r *Repo) ListMasterXOL(_ context.Context) ([]inboxxol.MasterXOL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.MasterXOL(nil), r.masters...), nil
}

// ListPendingMasterApproval mengembalikan perjanjian yang menunggu persetujuan komite.
//
// Penyaringnya memakai MasterXOL.AwaitingCommittee, bukan perbandingan teks sendiri:
// aturan "STSKOMITE='0' berarti menunggu" hidup di satu tempat, sehingga penyimpanan
// memori dan SQL tidak dapat berselisih menafsirkannya.
func (r *Repo) ListPendingMasterApproval(_ context.Context) ([]inboxxol.MasterXOL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]inboxxol.MasterXOL, 0, len(r.masters))
	for _, master := range r.masters {
		if master.AwaitingCommittee() {
			result = append(result, master)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Year < result[j].Year })
	return result, nil
}

// SummarizeClaims mengembalikan akumulasi klaim satu perjanjian.
//
// Penyaring group business diterapkan meski penyimpanan ini tidak punya kolomnya: yang
// ditiru adalah PENOLAKANNYA — perjanjian tanpa group business tidak mengembalikan baris
// apa pun, sama seperti di SQL. Tanpa itu, jalur "perjanjian belum diisi" hanya dapat
// dicoba dengan Oracle.
func (r *Repo) SummarizeClaims(_ context.Context, filter inboxxol.ClaimFilter) ([]inboxxol.ClaimSummary, error) {
	if filter.Empty() {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.ClaimSummary(nil), r.summaries[strings.TrimSpace(filter.Year)]...), nil
}

// BreakdownByBusiness mengembalikan rincian klaim milik sendiri.
func (r *Repo) BreakdownByBusiness(_ context.Context, filter inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	if filter.Empty() {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.BusinessBreakdown(nil),
		r.breakdowns[breakdownKey(filter.LossDate, filter.CauseOfLoss)]...), nil
}

// BreakdownTreatyInward mengembalikan rincian treaty inward.
func (r *Repo) BreakdownTreatyInward(_ context.Context, filter inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	if filter.TreatyEmpty() {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.BusinessBreakdown(nil),
		r.treaty[breakdownKey(filter.LossDate, filter.CauseOfLoss)]...), nil
}

// SearchAdvice mengembalikan pemberitahuan yang cocok dengan penyaring.
func (r *Repo) SearchAdvice(_ context.Context, filter inboxxol.AdviceFilter) ([]inboxxol.Advice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]inboxxol.Advice, 0, len(r.advices))
	for _, advice := range r.advices {
		if advice.Type != filter.Type {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(advice.Year), strings.TrimSpace(filter.Year)) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(advice.CauseOfLoss), strings.TrimSpace(filter.CauseOfLoss)) {
			continue
		}
		result = append(result, advice)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Revision != result[j].Revision {
			return result[i].Revision < result[j].Revision
		}
		return result[i].LayerID < result[j].LayerID
	})
	return result, nil
}

// ListPendingAdviceApproval merakit antrean persetujuan dari pemberitahuan yang ada.
//
// Pengelompokannya ditiru dari kueri lama: satu baris per tahun × sebab × tipe, hanya
// untuk yang STATUSAPPROVE-nya masih '0'. Termasuk cara `LAST_INSERTED` diambil, yaitu
// nilai TEKS terbesar — cacat yang direplikasi, lihat inboxxol.ApprovalItem.
func (r *Repo) ListPendingAdviceApproval(_ context.Context) ([]inboxxol.ApprovalItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	type key struct {
		year, cause string
		adviceType  inboxxol.AdviceType
	}
	latest := map[key]string{}
	for _, advice := range r.advices {
		if strings.TrimSpace(advice.ApprovalStatus) != "0" {
			continue
		}
		k := key{
			year:       strings.TrimSpace(advice.Year),
			cause:      strings.TrimSpace(advice.CauseOfLoss),
			adviceType: advice.Type,
		}
		if advice.IssuedOn > latest[k] {
			latest[k] = advice.IssuedOn
		}
	}

	result := make([]inboxxol.ApprovalItem, 0, len(latest))
	for k, inserted := range latest {
		result = append(result, inboxxol.ApprovalItem{
			Year:           k.year,
			CauseOfLoss:    k.cause,
			Type:           k.adviceType,
			LastInsertedAt: inserted,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].Year > result[j].Year
	})
	return result, nil
}

// ListCauseOfLoss mengembalikan daftar Penyebab Kerugian.
func (r *Repo) ListCauseOfLoss(_ context.Context) ([]inboxxol.CauseOfLoss, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.CauseOfLoss(nil), r.causeOfLoss...), nil
}

// breakdownKey menyatukan tanggal dan sebab menjadi satu kunci.
//
// Pemisahnya karakter yang tidak mungkin ada di dalam tanggal maupun deskripsi sebab
// kerugian, supaya dua pasangan berbeda tidak pernah menghasilkan kunci yang sama.
func breakdownKey(lossDate, causeOfLoss string) string {
	return strings.TrimSpace(lossDate) + "\x00" + strings.ToUpper(strings.TrimSpace(causeOfLoss))
}
