// Package memory memenuhi seam dashboardclaim.Repo dan ClosedClaimReader di dalam memori.
//
// Ia adalah adapter KEDUA di balik seam yang sama, dan karena itulah seam-nya nyata dan
// bukan hipotetis: satu adapter berarti abstraksi yang belum terbukti dibutuhkan.
//
// Dua kegunaannya:
//
//   - pengujian aturan layar tanpa basis data sama sekali, sehingga uji berjalan cepat dan
//     tidak bergantung pada Oracle yang hidup;
//   - mode pengembangan tanpa Oracle, supaya layar dapat dibuka dan digarap frontend
//     sebelum koneksi entitas tersedia.
//
// Penyaringnya sengaja menempuh aturan yang SAMA dengan sisi SQL — lini bisnis, kotak cari,
// kelima penyaring panel, paginasi — supaya perilaku yang diuji di sini memang perilaku yang
// berjalan di produksi.
// Yang tidak ditiru hanyalah bentuk gabung tabelnya, karena di sini tidak ada tabel.
package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/dashboardclaim"
)

// ClaimRecord adalah satu klaim beserta penanda yang dipakai menyaring lini bisnis.
//
// GroupPanel dan BusinessGroupID TIDAK ada di dashboardclaim.ClaimRow karena layar tidak
// menampilkannya. Keduanya tetap dibutuhkan di sini untuk menyaring, sehingga disimpan
// berdampingan — bukan ditambahkan ke tipe domain demi kemudahan adapter ini.
type ClaimRecord struct {
	Row             dashboardclaim.ClaimRow
	GroupPanel      string
	BusinessGroupID string

	// TransferredToCashier menirukan keberadaan baris `POOLDATA.T_CLAIM_ADJUSTMENT`
	// ber-`TRANSFER_CASHIER_DATE` tidak kosong.
	//
	// Ia penanda, bukan tanggal: yang disaring layar ini hanya "ada atau tidak", dan
	// menyimpan tanggalnya akan mengundang penyaring lain yang tidak ada di Pega.
	TransferredToCashier bool
}

// SurveyRecord adalah satu survei beserta penanda penyaringnya.
type SurveyRecord struct {
	Row             dashboardclaim.SurveyRow
	Kind            dashboardclaim.SurveyorType
	GroupPanel      string
	BusinessGroupID string
}

// Repo menyimpan klaim dan survei satu entitas di dalam memori.
type Repo struct {
	mu      sync.RWMutex
	claims  []ClaimRecord
	surveys []SurveyRecord

	// holding adalah isi tab Inbox Tampungan PIC.
	//
	// Disimpan TERPISAH dari claims, bukan disaring darinya: di produksi keduanya memang
	// dibaca kueri yang berbeda dengan syarat yang berbeda.
	holding []dashboardclaim.HoldingRow
}

// NewRepo membentuk penyimpanan memori.
//
// Menyalin irisannya, tidak menyimpan rujukan: pemanggil yang mengubah irisan asalnya
// setelah ini tidak boleh diam-diam mengubah isi penyimpanan.
func NewRepo(claims []ClaimRecord, surveys []SurveyRecord) *Repo {
	return &Repo{
		claims:  append([]ClaimRecord(nil), claims...),
		surveys: append([]SurveyRecord(nil), surveys...),
		holding: SampleHolding(),
	}
}

// CountOutstanding menghitung klaim berjalan yang cocok.
func (r *Repo) CountOutstanding(_ context.Context, f dashboardclaim.Filter) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.matchingClaims(f.Normalize())), nil
}

// ListOutstanding membaca satu halaman klaim berjalan.
func (r *Repo) ListOutstanding(_ context.Context, f dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f = f.Normalize()
	matched := r.matchingClaims(f)

	rows := make([]dashboardclaim.ClaimRow, 0, f.Limit)
	for _, record := range paginate(len(matched), f) {
		rows = append(rows, matched[record].Row)
	}
	return dashboardclaim.ClaimPage{Rows: rows, Total: len(matched)}, nil
}

// CountSurvey menghitung survei menurut jenis surveyornya.
//
// # Kedua jenis TIDAK menghitung satuan yang sama, dan itu ditiru di sini
//
// Sisi SQL mengikuti kueri Pega apa adanya (keputusan Work Owner 2026-09-26):
//
//	loss adjuster      jumlah KLAIM yang punya sekurang-kurangnya satu survei
//	internal surveyor  jumlah SURVEI-nya
//
// Ganda-uji ini menirunya alih-alih menyederhanakan keduanya menjadi "hitung baris". Ganda
// yang berperilaku lebih rapi daripada yang ditirunya akan membuat uji lulus pada semantik
// yang tidak pernah berjalan di produksi — dan justru perbedaan inilah yang perlu terlihat.
//
// Yang TIDAK ditiru: penggandaan akibat gabung ke PC_ASSIGN_WORKLIST, karena di sini tidak
// ada tabel penugasan untuk digabung. Itu satu-satunya sifat kueri lama yang tidak terwakili.
func (r *Repo) CountSurvey(_ context.Context, kind dashboardclaim.SurveyorType, f dashboardclaim.Filter) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matched := r.matchingSurveys(kind, f.Normalize())

	if kind == dashboardclaim.SurveyorAdjuster {
		claims := map[string]bool{}
		for _, record := range matched {
			claims[record.Row.ClaimNumber] = true
		}
		return len(claims), nil
	}
	return len(matched), nil
}

// ListSurvey membaca satu halaman survei menurut jenis surveyornya.
func (r *Repo) ListSurvey(_ context.Context, kind dashboardclaim.SurveyorType, f dashboardclaim.Filter) (dashboardclaim.SurveyPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f = f.Normalize()
	matched := r.matchingSurveys(kind, f)

	rows := make([]dashboardclaim.SurveyRow, 0, f.Limit)
	for _, record := range paginate(len(matched), f) {
		rows = append(rows, matched[record].Row)
	}
	return dashboardclaim.SurveyPage{Rows: rows, Total: len(matched)}, nil
}

// matchingClaims menyaring klaim. Pemanggil wajib sudah memegang kunci baca.
func (r *Repo) matchingClaims(f dashboardclaim.Filter) []ClaimRecord {
	result := make([]ClaimRecord, 0, len(r.claims))
	for _, record := range r.claims {
		if !matchesBusiness(f.Business, record.GroupPanel, record.BusinessGroupID) {
			continue
		}
		if !matchesSearch(f.Search, record.Row.PolicyNumber, record.Row.ClaimNumber) {
			continue
		}
		if !matchesPanel(f, record) {
			continue
		}
		result = append(result, record)
	}
	return result
}

// matchesPanel menerapkan kelima penyaring panel `FilterDashboardClaim`.
//
// Ia ADA, dan itu bukan kelengkapan yang berlebihan. Penyaring yang hadir di layar tetapi
// tidak berpengaruh pada mode tanpa basis data akan terbaca sebagai "tidak ada yang cocok"
// — bukan sebagai "penyaring ini belum bekerja". Itu persis kelas kekeliruan yang membuat
// panel ini dibangun lengkap atau tidak sama sekali.
func matchesPanel(f dashboardclaim.Filter, record ClaimRecord) bool {
	if !containsFold(record.Row.PolicyNumber, f.PolicyNumber) {
		return false
	}
	if !containsFold(record.Row.ClaimNumber, f.ClaimNumber) {
		return false
	}
	if !containsFold(record.Row.TechnicalPIC, f.TechnicalPIC) {
		return false
	}

	switch f.Cashier {
	case dashboardclaim.CashierDone:
		if !record.TransferredToCashier {
			return false
		}
	case dashboardclaim.CashierTodo:
		if record.TransferredToCashier {
			return false
		}
	}

	// `paidStatusCode` adalah kode Paid pada master status klaim. Perbandingannya memakai
	// kode, bukan labelnya: label dapat berubah di master tanpa kodenya berubah.
	switch f.Payment {
	case dashboardclaim.PaymentPaid:
		if record.Row.ClaimStatusCode != paidStatusCode {
			return false
		}
	case dashboardclaim.PaymentUnpaid:
		// Sengaja `!=` tanpa memperlakukan kosong secara khusus — SQL-nya pun begitu, dan
		// baris tanpa status klaim ikut terbuang di sana. Lihat catatan di outstanding.sql.
		if record.Row.ClaimStatusCode == paidStatusCode || record.Row.ClaimStatusCode == "" {
			return false
		}
	}

	return true
}

// paidStatusCode adalah `1163` — Paid pada master status klaim (`R-06`).
const paidStatusCode = "1163"

// containsFold menjawab "apakah isian penyaring cocok", dengan isian kosong selalu cocok.
func containsFold(value, needle string) bool {
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(value), strings.ToUpper(needle))
}

// matchingSurveys menyaring survei. Pemanggil wajib sudah memegang kunci baca.
func (r *Repo) matchingSurveys(kind dashboardclaim.SurveyorType, f dashboardclaim.Filter) []SurveyRecord {
	result := make([]SurveyRecord, 0, len(r.surveys))
	for _, record := range r.surveys {
		if record.Kind != kind {
			continue
		}
		if !matchesBusiness(f.Business, record.GroupPanel, record.BusinessGroupID) {
			continue
		}
		if !matchesSearch(f.Search, record.Row.PolicyNumber, record.Row.SurveyNumber) {
			continue
		}
		result = append(result, record)
	}
	return result
}

// bondingGroups adalah keempat kode kelompok bisnis yang menandai lini Bonding.
//
// Nilainya sama persis dengan yang tertanam di kueri lama. Di sistem baru ia SEHARUSNYA
// master data (`D-15`); dibiarkan sebagai konstanta di sini karena adapter memori hanya
// dipakai untuk uji dan mode tanpa basis data — bukan jalur produksi.
var bondingGroups = map[string]bool{
	"10008": true, "10010": true, "10015": true, "10023": true,
}

// matchesBusiness menerapkan penyaring lini bisnis.
//
// Aturannya sama dengan `Activity/SetDashboardClaim-Act.xml`, termasuk bahwa Group Panel
// `009` TIDAK termasuk Non-MBU. Lihat catatan pada dashboardclaim.BusinessLine.
func matchesBusiness(line dashboardclaim.BusinessLine, groupPanel, businessGroupID string) bool {
	switch line {
	case dashboardclaim.BusinessAll, "":
		return true
	case dashboardclaim.BusinessNonMBU:
		switch groupPanel {
		case "003", "004", "006":
			return !bondingGroups[businessGroupID]
		default:
			return false
		}
	case dashboardclaim.BusinessBonding:
		return bondingGroups[businessGroupID]
	case dashboardclaim.BusinessPA:
		return groupPanel == "002"
	case dashboardclaim.BusinessTravel:
		return groupPanel == "005"
	default:
		return false
	}
}

// matchesSearch menerapkan kotak cari gabungan atas dua kolom.
//
// Perbandingannya tidak peka huruf besar-kecil, sama dengan `UPPER(...) LIKE` di sisi SQL.
// Menyeragamkan hanya satu sisi akan membuat pencarian huruf kecil tidak pernah cocok, dan
// gagalnya diam.
func matchesSearch(search string, fields ...string) bool {
	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToUpper(field), needle) {
			return true
		}
	}
	return false
}

// paginate mengembalikan indeks baris yang masuk halaman yang diminta.
//
// Mengembalikan indeks, bukan irisan, supaya pemanggil dapat memetakannya ke bentuk baris
// yang berbeda tanpa menyalin dua kali.
func paginate(total int, f dashboardclaim.Filter) []int {
	if f.Offset >= total {
		return nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}
	indexes := make([]int, 0, end-f.Offset)
	for i := f.Offset; i < end; i++ {
		indexes = append(indexes, i)
	}
	return indexes
}

// ListHolding membaca satu halaman tab Inbox Tampungan PIC.
//
// Penampungannya disimpan terpisah dari klaim berjalan, bukan disaring darinya: di produksi
// keduanya memang dibaca kueri yang berbeda dengan syarat yang berbeda — tugas yang diparkir
// di akun `ServicePNC` dan belum punya PIC Teknik.
//
// Penyaring lini bisnis TIDAK diterapkan di sini, sama seperti di sisi SQL.
func (r *Repo) ListHolding(_ context.Context, f dashboardclaim.Filter) (dashboardclaim.HoldingPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f = f.Normalize()

	matched := make([]dashboardclaim.HoldingRow, 0, len(r.holding))
	for _, row := range r.holding {
		if !matchesSearch(f.Search, row.ClaimNumber) {
			continue
		}
		matched = append(matched, row)
	}

	rows := make([]dashboardclaim.HoldingRow, 0, f.Limit)
	for _, index := range paginate(len(matched), f) {
		rows = append(rows, matched[index])
	}
	return dashboardclaim.HoldingPage{Rows: rows, Total: len(matched)}, nil
}
