// Package memory adalah pengisi seam penyimpanan Master Rekening yang hidup di dalam
// memori.
//
// Ia ada supaya alur pengajuan dan keputusan komite dapat diuji tanpa basis data dan
// tanpa jaringan sama sekali — adapter kedua inilah yang membuat seam penyimpanan
// menjadi seam nyata, bukan seam hipotetis
// (docs/Steering/04-FUTURE-ARCHITECTURE.md §3.1).
//
// Adapter ini TIDAK dipakai di produksi: master rekening yang hilang saat proses
// dijalankan ulang tidak ada gunanya.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterrekening"
)

// Repo menyimpan master rekening di memori, dikunci Number+BankCode.
type Repo struct {
	mu      sync.RWMutex
	content map[masterrekening.Key]masterrekening.Account
}

// NewRepo membentuk store berisi baris awal yang diberikan.
func NewRepo(start ...masterrekening.Account) *Repo {
	s := &Repo{content: map[masterrekening.Key]masterrekening.Account{}}
	for _, r := range start {
		s.content[r.KeyOf()] = r
	}
	return s
}

// List membaca rekening yang cocok dengan filter.
func (s *Repo) List(_ context.Context, f masterrekening.Filter) ([]masterrekening.Account, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cocok := make([]masterrekening.Account, 0, len(s.content))
	for _, r := range s.content {
		if !passesFilter(r, f) {
			continue
		}
		cocok = append(cocok, r)
	}

	// Urutan tetap: baris terbaru lebih dulu, lalu nomor rekening sebagai pemutus
	// seri. Tanpa pemutus seri, dua baris berwaktu sama dapat bertukar tempat di
	// antara dua permintaan dan membuat paginasi melewatkan baris.
	sort.Slice(cocok, func(i, j int) bool {
		if !cocok[i].CreatedAt.Equal(cocok[j].CreatedAt) {
			return cocok[i].CreatedAt.After(cocok[j].CreatedAt)
		}
		return cocok[i].Number < cocok[j].Number
	})

	total := len(cocok)
	return trimmed(cocok, f), total, nil
}

func trimmed(rows []masterrekening.Account, f masterrekening.Filter) []masterrekening.Account {
	mulai := f.Offset
	if mulai < 0 {
		mulai = 0
	}
	if mulai >= len(rows) {
		return []masterrekening.Account{}
	}
	sisa := rows[mulai:]
	if f.Limit > 0 && f.Limit < len(sisa) {
		sisa = sisa[:f.Limit]
	}
	return append([]masterrekening.Account(nil), sisa...)
}

func passesFilter(r masterrekening.Account, f masterrekening.Filter) bool {
	if f.Status != "" && r.Status != f.Status {
		return false
	}
	if !contains(r.Number, f.Number) {
		return false
	}
	if !contains(r.OwnerName, f.OwnerName) {
		return false
	}
	if !contains(r.BankName, f.BankName) {
		return false
	}
	if f.MyCommitteeOnly && f.CommitteeIdentity != "" {
		if !equalIgnoringCase(r.CommitteeApproval, f.CommitteeIdentity) {
			return false
		}
	}
	return true
}

func contains(value, search string) bool {
	search = strings.TrimSpace(search)
	if search == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(value), strings.ToUpper(search))
}

func equalIgnoringCase(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// Get membaca satu rekening.
func (s *Repo) Get(_ context.Context, k masterrekening.Key) (masterrekening.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, existing := s.content[k]
	if !existing {
		return masterrekening.Account{}, masterrekening.ErrNotFound
	}
	return r, nil
}

// FindByNumber membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli banknya.
func (s *Repo) FindByNumber(_ context.Context, nomor string) ([]masterrekening.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nomor = strings.TrimSpace(nomor)
	var result []masterrekening.Account
	for _, r := range s.content {
		if r.Number == nomor {
			result = append(result, r)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].BankCode < result[j].BankCode })
	return result, nil
}

// Save menyisipkan rekening baru.
func (s *Repo) Save(_ context.Context, r masterrekening.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existing := s.content[r.KeyOf()]; existing {
		return masterrekening.ErrAlreadyExists
	}
	s.content[r.KeyOf()] = r
	return nil
}

// Update menulis ulang rekening yang sudah ada.
func (s *Repo) Update(_ context.Context, r masterrekening.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existing := s.content[r.KeyOf()]; !existing {
		return masterrekening.ErrNotFound
	}
	s.content[r.KeyOf()] = r
	return nil
}

// ClearRejected membuang baris bekas penolakan komite.
//
// Ia menolak menghapus baris yang statusnya bukan StatusRejected. Syarat itu ditegakkan
// di sini, bukan dipercayakan kepada pemanggil: sebuah method bernama "hapus" yang mau
// menghapus apa saja cepat atau lambat akan dipanggil untuk apa saja.
func (s *Repo) ClearRejected(_ context.Context, k masterrekening.Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, existing := s.content[k]
	if !existing {
		return masterrekening.ErrNotFound
	}
	if r.Status != masterrekening.StatusRejected {
		return masterrekening.ErrAlreadyDecided
	}
	delete(s.content, k)
	return nil
}

// BankRepo menyimpan daftar bank di memori.
type BankRepo struct {
	list []masterrekening.Bank
}

// NewBankRepo membentuk daftar bank.
func NewBankRepo(list ...masterrekening.Bank) *BankRepo {
	return &BankRepo{list: list}
}

// List mengembalikan seluruh bank.
func (s *BankRepo) List(context.Context) ([]masterrekening.Bank, error) {
	return append([]masterrekening.Bank(nil), s.list...), nil
}

// SampleBanks adalah beberapa bank untuk menjalankan aplikasi tanpa basis data.
//
// Kodenya adalah LBG_ID yang sesungguhnya dipakai GENERAL.LST_BANK_GROUP; namanya
// adalah nama bank umum di Indonesia — bukan data nasabah, bukan karangan yang
// menyesatkan.
func SampleBanks() []masterrekening.Bank {
	return []masterrekening.Bank{
		{Code: "002", Name: "BANK BRI"},
		{Code: "008", Name: "BANK MANDIRI"},
		{Code: "009", Name: "BANK BNI"},
		{Code: "014", Name: "BANK BCA"},
		{Code: "011", Name: "BANK DANAMON"},
		{Code: "013", Name: "BANK PERMATA"},
		{Code: "022", Name: "BANK CIMB NIAGA"},
		{Code: "016", Name: "BANK MAYBANK INDONESIA"},
		{Code: "153", Name: "BANK SINARMAS"},
		{Code: "451", Name: "BANK SYARIAH INDONESIA"},
	}
}

var (
	_ masterrekening.Repo     = (*Repo)(nil)
	_ masterrekening.BankRepo = (*BankRepo)(nil)
)
