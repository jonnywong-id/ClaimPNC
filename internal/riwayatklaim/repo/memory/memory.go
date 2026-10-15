// Package memory memenuhi seam riwayatklaim.Repo dan riwayatklaim.ProtectionRepo di
// dalam proses.
//
// Ia dipakai dua hal: pengujian, dan menjalankan aplikasi tanpa Oracle. Yang kedua bukan
// kenyamanan belaka di modul ini — layar View History Claim tergerbang proteksi data, dan
// tanpa penyimpanan memori beserta contoh proteksinya, layar itu tidak dapat dicoba sama
// sekali sebelum DBA menjalankan migrasi 0004 DAN mendaftarkan penguji di Master Proteksi
// Data.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/riwayatklaim"
)

// Claim adalah satu baris contoh beserta isian yang dapat dicari.
//
// Ia lebih lebar daripada riwayatklaim.ClaimHistory karena pencarian di sistem lama
// menyentuh kolom yang TIDAK ditampilkan di grid — nomor PLA, nomor DLA, nama objek,
// nomor survei. Tanpa isian itu, kesebelas tipe pencarian tidak dapat dicoba.
type Claim struct {
	// History adalah baris yang ditampilkan.
	History riwayatklaim.ClaimHistory

	// Isian yang dicari tetapi tidak ditampilkan.
	ObjectName   string
	PLANumber    string
	DLANumber    string
	AuctionID    string
	SurveyNumber string

	// PersonName dan PersonBirthDate adalah peserta pada lini Personal Accident.
	// Keduanya yang muncul sebagai Nama Objek dan Tanggal Lahir pada pencarian tipe 9.
	PersonName      string
	PersonBirthDate *time.Time

	// AcceptanceNumber dicari tipe 8 dan ditampilkan tipe 13.
	AcceptanceNumber string
}

// Repo menyimpan riwayat klaim di memori.
type Repo struct {
	mu     sync.RWMutex
	claims []Claim
}

// NewRepo membentuk penyimpanan memori berisi baris yang diberikan.
func NewRepo(claims ...Claim) *Repo {
	stored := make([]Claim, len(claims))
	copy(stored, claims)
	return &Repo{claims: stored}
}

// Search menyaring baris contoh mengikuti semantik kueri Oracle-nya.
//
// Kecocokannya sengaja dibuat setara dengan SQL-nya, termasuk mana yang sebagian dan mana
// yang persis: pencarian yang berperilaku berbeda antara memori dan Oracle membuat uji
// yang lulus di memori tidak membuktikan apa pun tentang produksi.
func (r *Repo) Search(
	_ context.Context,
	criteria riwayatklaim.Criteria,
	page riwayatklaim.Pagination,
) (riwayatklaim.Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	page = page.Normalize()

	matched := make([]riwayatklaim.ClaimHistory, 0, len(r.claims))
	for _, claim := range r.claims {
		if !matches(claim, criteria) {
			continue
		}
		matched = append(matched, decorate(claim, criteria))
	}

	// Diurutkan menurut Reference, sama dengan ORDER BY pada adapter SQL. Tanpa urutan
	// yang sama, paginasi di memori dan di Oracle menghasilkan halaman yang berbeda.
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].Reference < matched[j].Reference
	})

	total := len(matched)
	start := page.Offset()
	if start >= total {
		return riwayatklaim.Page{
			Claims:     []riwayatklaim.ClaimHistory{},
			Total:      total,
			Pagination: page,
		}, nil
	}

	end := start + page.Size
	if end > total {
		end = total
	}

	window := make([]riwayatklaim.ClaimHistory, end-start)
	copy(window, matched[start:end])

	return riwayatklaim.Page{Claims: window, Total: total, Pagination: page}, nil
}

// matches menilai satu baris terhadap kriteria.
func matches(claim Claim, criteria riwayatklaim.Criteria) bool {
	text, date, isDate := criteria.QueryValue()

	if isDate {
		// Tanggal kosong tidak pernah cocok — perbandingan dengan NULL di Oracle
		// menghasilkan UNKNOWN, dan hasilnya kosong. Inilah yang membuat pencarian
		// Tanggal Lahir tidak pernah mengembalikan baris, di sini maupun di Oracle.
		if date == nil {
			return false
		}
		switch criteria.Type.Code {
		case riwayatklaim.TypeLossDate:
			return sameDay(claim.History.LossDate, date)
		case riwayatklaim.TypeBirthDate:
			return sameDay(claim.PersonBirthDate, date)
		default:
			return false
		}
	}

	switch criteria.Type.Code {
	case riwayatklaim.TypePolicyNumber:
		return equalFold(claim.History.PolicyNumber, text)
	case riwayatklaim.TypeInsuredName:
		return contains(claim.History.InsuredName, text)
	case riwayatklaim.TypeInsuredItemName:
		return contains(claim.ObjectName, text)
	case riwayatklaim.TypePLANumber:
		return contains(claim.PLANumber, text)
	case riwayatklaim.TypeDLANumber:
		return contains(claim.DLANumber, text)
	case riwayatklaim.TypeClaimNumber:
		return equalFold(claim.History.Number, text)
	case riwayatklaim.TypeAcceptanceNumber:
		return equalFold(claim.AcceptanceNumber, text)
	case riwayatklaim.TypeSurveyNumber:
		return equalFold(claim.SurveyNumber, text)
	case riwayatklaim.TypeAuctionHouseID:
		return equalFold(claim.AuctionID, text)
	default:
		return false
	}
}

// decorate mengisi kolom yang hanya dibawa sebagian kueri.
//
// Di adapter SQL perbedaan ini datang dari kueri yang berbeda; di sini ia harus
// diberlakukan sendiri, supaya kolom yang seharusnya kosong pada sebuah tipe pencarian
// benar-benar kosong. Kalau tidak, uji yang memeriksa kolom Posisi Klaim kosong pada
// pencarian Nama Objek akan lulus di Oracle dan gagal di memori — atau sebaliknya.
func decorate(claim Claim, criteria riwayatklaim.Criteria) riwayatklaim.ClaimHistory {
	history := claim.History

	switch criteria.Type.Code {
	case riwayatklaim.TypeBirthDate:
		history.InsuredItemName = claim.PersonName
		history.BirthDate = claim.PersonBirthDate
	case riwayatklaim.TypeAuctionHouseID:
		history.AcceptanceNumber = claim.AcceptanceNumber
		history.AuctionHouseID = claim.AuctionID
	case riwayatklaim.TypeInsuredItemName:
		// CACAT YANG DIREPLIKASI: kueri tipe 3 adalah satu-satunya yang tidak membawa
		// subquery V_STS_CLAIM, sehingga Posisi Klaim kosong. Lihat riwayatklaim.sql.
		history.ClaimPosition = ""
	}

	return history
}

func equalFold(value, keyword string) bool {
	return strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(keyword))
}

func contains(value, keyword string) bool {
	return strings.Contains(
		strings.ToUpper(strings.TrimSpace(value)),
		strings.ToUpper(strings.TrimSpace(keyword)),
	)
}

func sameDay(value, want *time.Time) bool {
	if value == nil || want == nil {
		return false
	}
	valueYear, valueMonth, valueDay := value.UTC().Date()
	wantYear, wantMonth, wantDay := want.UTC().Date()
	return valueYear == wantYear && valueMonth == wantMonth && valueDay == wantDay
}

// ProtectionRepo menyimpan gerbang proteksi data di memori.
//
// Bagian BACA-nya mewakili POOLDATA.MST_PROTEKSI_DATA_PNC milik sistem lama; bagian
// TULIS-nya mewakili POOLDATA.CPNC_PEMAKAIAN_PROTEKSI milik aplikasi ini. Keduanya
// dipisahkan di sini pula supaya perilaku yang diuji sama dengan perilaku di Oracle.
type ProtectionRepo struct {
	mu          sync.Mutex
	protections map[string]riwayatklaim.Protection
	usage       []riwayatklaim.Usage
}

// NewProtectionRepo membentuk gerbang di memori berisi baris proteksi yang diberikan.
func NewProtectionRepo(protections ...riwayatklaim.Protection) *ProtectionRepo {
	stored := map[string]riwayatklaim.Protection{}
	for _, protection := range protections {
		stored[key(protection.Login)] = protection
	}
	return &ProtectionRepo{protections: stored}
}

// Find membaca baris proteksi milik satu pengguna.
//
// Modul diabaikan di sini, dan itu disengaja: penyimpanan memori hanya melayani satu
// layar, sehingga membedakan modul akan menambah keadaan yang tidak pernah berbeda.
func (r *ProtectionRepo) Find(
	_ context.Context,
	login, _ string,
) (riwayatklaim.Protection, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	protection, exists := r.protections[key(login)]
	return protection, exists, nil
}

// CountUsage menghitung pemakaian yang mengurangi jatah.
func (r *ProtectionRepo) CountUsage(_ context.Context, login, module string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := 0
	for _, usage := range r.usage {
		if !usage.ConsumesQuota {
			continue
		}
		if key(usage.Login) == key(login) && key(usage.Module) == key(module) {
			total++
		}
	}
	return total, nil
}

// RecordUsage mencatat satu pemakaian.
func (r *ProtectionRepo) RecordUsage(_ context.Context, usage riwayatklaim.Usage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.usage = append(r.usage, usage)
	return nil
}

// Usage mengembalikan seluruh jejak yang tercatat.
//
// Ia ada untuk pengujian: jejak audit adalah satu-satunya kontrol pengimbang yang tersisa
// (`D-59`), dan kontrol yang tidak diperiksa uji apa pun adalah kontrol yang dapat hilang
// tanpa ada yang menyadarinya.
func (r *ProtectionRepo) Usage() []riwayatklaim.Usage {
	r.mu.Lock()
	defer r.mu.Unlock()

	list := make([]riwayatklaim.Usage, len(r.usage))
	copy(list, r.usage)
	return list
}

func key(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
