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
	"strconv"
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

	// salvageUploads merekam baris yang disisipkan "Upload MBU Salvage".
	salvageUploads []inboxxol.SalvageInsert

	// dolColInserts merekam baris yang disisipkan "INSERT DOL DAN COL".
	dolColInserts []inboxxol.DolColInsert
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

// SummarizeBusiness menyusun grid "Summary Data XOL" dari rincian yang sudah ada.
//
// # Kenapa diturunkan, bukan disimpan terpisah
//
// Karena isinya memang turunan: kuerinya hanya menyebutkan group business MANA yang
// menanggung klaim pada tanggal dan penyebab kerugian itu — persis himpunan yang sudah
// dibawa kedua daftar rincian. Menyimpannya sebagai daftar ketiga membuka kemungkinan
// ketiganya saling bertentangan di data contoh, dan itu cacat yang hanya ada di tiruan.
func (r *Repo) SummarizeBusiness(
	_ context.Context,
	filter inboxxol.SummaryFilter,
) ([]inboxxol.SummaryBusiness, error) {
	if filter.Empty() {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := breakdownKey(filter.LossDate, filter.CauseOfLoss)
	result := make([]inboxxol.SummaryBusiness, 0, 8)
	seen := make(map[string]bool)

	for _, row := range append(append([]inboxxol.BusinessBreakdown(nil),
		r.breakdowns[key]...), r.treaty[key]...) {
		id := strings.TrimSpace(row.BusinessGroupID)
		name := strings.TrimSpace(row.BusinessGroup)
		if seen[id+"|"+name] {
			continue
		}
		seen[id+"|"+name] = true
		result = append(result, inboxxol.SummaryBusiness{
			BusinessGroupID:   id,
			BusinessGroupName: name,
		})
	}
	return result, nil
}

// ListClaims menyusun grid "No Klaim" dari rincian yang sudah ada.
//
// Diturunkan, bukan disimpan terpisah — alasannya sama dengan SummarizeBusiness: data
// contoh yang menyimpan hal yang sama dua kali dapat saling bertentangan, dan cacat itu
// hanya ada di tiruannya.
//
// Satu baris per group business, dengan nomor klaim karangan yang diturunkan dari nama
// group business-nya. Yang diuji lewat repo ini adalah PERILAKU layar — berapa baris,
// bagaimana baris berkurs hilang diperlakukan — bukan isi nomor klaimnya.
func (r *Repo) ListClaims(
	_ context.Context,
	filter inboxxol.ClaimListFilter,
) ([]inboxxol.ClaimListItem, error) {
	if filter.Empty() {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := breakdownKey(filter.LossDate, filter.CauseOfLoss)
	result := make([]inboxxol.ClaimListItem, 0, 8)

	for _, row := range r.breakdowns[key] {
		result = append(result, inboxxol.ClaimListItem{
			ClaimNo:          "PNC-" + strings.ToUpper(strings.TrimSpace(row.BusinessGroupID)),
			CurrencyName:     "IDR",
			Source:           inboxxol.SourceOwnBusiness,
			OutstandingValue: row.OutstandingValue,
			AcceptedValue:    row.AcceptedValue,
		})
	}
	for _, row := range r.treaty[key] {
		result = append(result, inboxxol.ClaimListItem{
			ClaimNo:          strings.TrimSpace(row.BusinessGroup),
			CurrencyName:     "USD",
			Source:           inboxxol.SourceTreatyInward,
			OutstandingValue: row.OutstandingValue,
			AcceptedValue:    row.AcceptedValue,
			RateMissing:      row.RateMissing,
		})
	}
	return result, nil
}

// ExportClaimDetail menyusun isi berkas unduhan dari rincian yang sudah ada.
//
// Judul kolomnya TIDAK ditiru — yang ditiru adalah bentuknya: satu baris judul, lalu satu
// baris per klaim. Yang diuji lewat repo ini adalah perilaku layar dan handler, bukan isi
// berkas; kesetiaan kolom diuji terhadap basis data sungguhan (`S-8`).
func (r *Repo) ExportClaimDetail(
	_ context.Context,
	filter inboxxol.ExportFilter,
) (inboxxol.ExportTable, error) {
	if filter.Empty() {
		return inboxxol.ExportTable{}, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key := breakdownKey(filter.LossDate, filter.CauseOfLoss)
	source := r.breakdowns[key]
	if strings.TrimSpace(filter.BusinessGroupID) == inboxxol.ExportBusinessGroupTreaty {
		source = r.treaty[key]
	}

	table := inboxxol.ExportTable{
		Headers: []string{"CLAIM NO", "BUSINESS", "RESERVE AMOUNT", "ACCEPTED"},
		Rows:    make([][]string, 0, len(source)),
	}
	for _, row := range source {
		table.Rows = append(table.Rows, []string{
			"PNC-" + strings.ToUpper(strings.TrimSpace(row.BusinessGroupID)),
			strings.TrimSpace(row.BusinessGroup),
			strconv.FormatFloat(row.OutstandingValue, 'f', 2, 64),
			strconv.FormatFloat(row.AcceptedValue, 'f', 2, 64),
		})
	}
	return table, nil
}

// CurrencyIDByName mencari ID mata uang dari namanya.
//
// Daftar contohnya sengaja sempit — hanya yang benar-benar dipakai uji. Nama di luar itu
// dijawab teks kosong, dan itulah yang membuat jalur "mata uang tidak dikenal" dapat diuji
// tanpa basis data.
func (r *Repo) CurrencyIDByName(_ context.Context, name string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "IDR":
		return "1", nil
	case "USD":
		return "2", nil
	}
	return "", nil
}

// UploadSalvageMBU menyimpan baris unggahan ke dalam ingatan.
func (r *Repo) UploadSalvageMBU(_ context.Context, rows []inboxxol.SalvageInsert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.salvageUploads = append(r.salvageUploads, rows...)
	return nil
}

// SalvageUploads mengembalikan baris yang sudah tersimpan — dipakai uji untuk memastikan
// yang tersimpan memang yang dipetakan, bukan sekadar jumlahnya cocok.
func (r *Repo) SalvageUploads() []inboxxol.SalvageInsert {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.SalvageInsert(nil), r.salvageUploads...)
}

// InsertDolCol menyimpan baris "INSERT DOL DAN COL" ke dalam ingatan.
func (r *Repo) InsertDolCol(_ context.Context, rows []inboxxol.DolColInsert) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dolColInserts = append(r.dolColInserts, rows...)
	return nil
}

// DolColInserts mengembalikan baris yang sudah tersimpan — dipakai uji untuk memastikan
// satu simpan menghasilkan satu baris PER GROUP BUSINESS, bukan satu baris saja.
func (r *Repo) DolColInserts() []inboxxol.DolColInsert {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]inboxxol.DolColInsert(nil), r.dolColInserts...)
}
