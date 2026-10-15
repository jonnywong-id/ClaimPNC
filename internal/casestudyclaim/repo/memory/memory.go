// Package memory memenuhi seam casestudyclaim.Repo di dalam memori.
//
// Dipakai dua hal: pengujian, dan menjalankan aplikasi tanpa Oracle saat pengembangan.
// Keduanya menuntut hal yang sama — penyaring di sini harus berperilaku PERSIS seperti
// penyaring di SQL, kalau tidak layar yang benar saat pengembangan akan salah di produksi.
//
// Karena itu penyaringnya ditulis mengikuti kueri baris per baris, termasuk hal-hal yang
// tampak aneh: ambang yang membandingkan SATU baris settlement (bukan jumlahnya), dan
// status "bukan ditolak" yang meloloskan nilai asing apa pun.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
)

// Record adalah satu klaim beserta nilai-nilai yang dipakai MENYARING.
//
// Baris yang ditampilkan dan bahan penyaringnya dipisahkan dengan sengaja: di SQL,
// `GROUPPANEL`, `BUSINESSCODE`, dan `STSKLAIM` ikut menentukan siapa yang muncul tetapi
// TIDAK pernah ditampilkan sebagai kolom. Menyatukannya ke dalam baris tampilan akan
// membuat penyimpanan ini menyaring atas data yang tidak dimiliki penyimpanan sebenarnya.
type Record struct {
	Row casestudyclaim.CaseStudyRow

	// GroupPanel dan BusinessCode adalah `b.GROUPPANEL` dan `b.BUSINESSCODE`.
	GroupPanel   string
	BusinessCode string

	// StatusCode adalah `a.STSKLAIM` — kode mentahnya, bukan labelnya.
	StatusCode string

	// LargestAdjustment adalah nilai settlement TERBESAR milik klaim ini, yaitu
	// `MAX(TOTAL_CLAIM * CURRENCYVALUE)`.
	//
	// Ia dipakai menguji ambang Rp 5 miliar, dan itu SATU BARIS — bukan jumlah seluruh
	// baris. Kueri lama memakai `EXISTS (… WHERE total_claim*currencyvalue > 5000000000)`,
	// yang benar bila ADA SATU baris yang melampauinya. Klaim bernilai total Rp 8 miliar
	// yang terpecah menjadi dua baris Rp 4 miliar karena itu TIDAK masuk layar ini.
	LargestAdjustment money.Money
}

// Store menyimpan klaim telaah di memori.
type Store struct {
	mutex   sync.RWMutex
	records []Record
}

// NewStore membentuk penyimpanan kosong.
func NewStore() *Store { return &Store{} }

// NewRepo membentuk penyimpanan berisi baris yang diberikan.
func NewRepo(records ...Record) *Store {
	store := NewStore()
	store.records = append(store.records, records...)
	return store
}

// List menyaring, mengurutkan, dan memotong halaman — meniru case_study_list.
func (s *Store) List(ctx context.Context, f casestudyclaim.Filter) (casestudyclaim.Page, error) {
	if err := ctx.Err(); err != nil {
		return casestudyclaim.Page{}, err
	}
	f = f.Normalize()

	if err := casestudyclaim.ValidatePeriod(f.FromYear, f.ToYear); err != nil {
		return casestudyclaim.Page{}, err
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	matched := make([]casestudyclaim.CaseStudyRow, 0, len(s.records))
	for _, record := range s.records {
		if !record.matches(f) {
			continue
		}
		matched = append(matched, record.Row)
	}

	// Urutan yang sama dengan SQL: nomor klaim, lalu apa pun yang memutus seri. Tanpa
	// urutan pasti, paginasi mengulang dan melewatkan baris.
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].ClaimNumber < matched[j].ClaimNumber
	})

	total := len(matched)
	if f.Offset >= total {
		return casestudyclaim.Page{Rows: []casestudyclaim.CaseStudyRow{}, Total: total}, nil
	}
	end := f.Offset + f.Limit
	if end > total {
		end = total
	}

	page := make([]casestudyclaim.CaseStudyRow, end-f.Offset)
	copy(page, matched[f.Offset:end])
	return casestudyclaim.Page{Rows: page, Total: total}, nil
}

// matches menjawab apakah satu klaim lolos seluruh penyaring.
func (r Record) matches(f casestudyclaim.Filter) bool {
	// Ambang lebih dulu, karena itulah yang mendefinisikan layar ini.
	if r.LargestAdjustment <= casestudyclaim.LargeClaimThreshold {
		return false
	}

	// Tahun dibandingkan sebagai TEKS, sama seperti SQL membandingkan `A.THNREGIS`
	// terhadap keluaran `TO_CHAR(…,'yyyy')`.
	year := strings.TrimSpace(r.Row.PolicyPeriod)
	if year < f.FromYear || year > f.ToYear {
		return false
	}

	if !r.inBusinessScope(f.Business) {
		return false
	}
	return r.inStatus(f.Status)
}

// nonMBUExcluded adalah kode bisnis yang DIKELUARKAN dari cakupan NONMBU.
//
// Disalin apa adanya dari `and b.businesscode NOT IN (…)` pada kueri lama. Di-hardcode di
// sana, dan di sini pun — keduanya menunggu master `F-4` (`D-15`).
var nonMBUExcluded = map[string]bool{
	"10145": true, "10168": true, "10165": true, "10164": true, "10053": true,
}

// bondingIncluded adalah kesepuluh kode bisnis yang MEMBENTUK cakupan BONDING.
//
// Inilah yang membedakan BONDING dari bagian Aneka pada NONMBU: keduanya berada di Group
// Panel `003`, dan hanya daftar inilah yang memisahkannya.
var bondingIncluded = map[string]bool{
	"10076": true, "10077": true, "10007": true, "10011": true, "10083": true,
	"10141": true, "10131": true, "10126": true, "10055": true, "10075": true,
}

func (r Record) inBusinessScope(scope casestudyclaim.BusinessScope) bool {
	panel := strings.TrimSpace(r.GroupPanel)
	code := strings.TrimSpace(r.BusinessCode)

	switch scope {
	case casestudyclaim.ScopeAll:
		return true
	case casestudyclaim.ScopePA:
		return panel == "002"
	case casestudyclaim.ScopeTravel:
		return panel == "005"
	case casestudyclaim.ScopeNonMBU:
		switch panel {
		case "003", "004", "006", "009":
			return !nonMBUExcluded[code]
		default:
			return false
		}
	case casestudyclaim.ScopeBonding:
		return panel == "003" && bondingIncluded[code]
	default:
		// Cakupan tak dikenal tidak pernah sampai ke sini — transport menolaknya. Bila
		// ternyata sampai, ia MENYEMBUNYIKAN seluruh baris, bukan menampilkan semuanya.
		return false
	}
}

func (r Record) inStatus(status casestudyclaim.ClaimStatusFilter) bool {
	code := strings.TrimSpace(r.StatusCode)

	switch status {
	case casestudyclaim.StatusAny:
		return true
	case casestudyclaim.StatusInProgressOrAccepted:
		// `!= '3'`, bukan `IN ('0','1')`. Nilai asing apa pun ikut lolos, persis seperti
		// di SQL — dan itu direplikasi, bukan dirapikan.
		return code != "3"
	case casestudyclaim.StatusRejected:
		return code == "3"
	default:
		return false
	}
}

// SaveRemark menuliskan catatan telaah satu klaim.
//
// Nilai pertama `false` berarti nomor klaimnya tidak ada — keadaan yang sah, bukan galat.
func (s *Store) SaveRemark(ctx context.Context, claimNumber, remark string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	number := strings.TrimSpace(claimNumber)
	if number == "" {
		return false, casestudyclaim.ErrClaimRequired
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for index := range s.records {
		if strings.TrimSpace(s.records[index].Row.ClaimNumber) != number {
			continue
		}
		s.records[index].Row.Remark = strings.TrimSpace(remark)
		return true, nil
	}
	return false, nil
}
