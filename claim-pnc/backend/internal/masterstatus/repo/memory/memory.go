// Package memory adalah pengisi seam masterstatus.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master
// Status Klaim dapat dicoba lengkap dengan 33 baris yang benar sebelum DBA menjalankan
// migrasi 0002.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterstatus"
)

// Repo menyimpan master status di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	rows   map[string]masterstatus.ClaimStatus
	order  int
	site   string
	issues error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...masterstatus.ClaimStatus) *Repo {
	r := &Repo{
		rows: make(map[string]masterstatus.ClaimStatus, len(list)),
		// Nilai berikut meniru pembentukan kode di sistem lama: id_site = "1" dan
		// urutan tiga digit. Urutan dimulai dari kode tertinggi yang sudah ada supaya
		// penambahan pertama menghasilkan kode berikutnya yang wajar, bukan bentrok.
		site: "1",
	}
	for _, s := range list {
		bersih := s.Clean()
		r.rows[bersih.Code] = bersih
		if n := sequenceFromCode(bersih.Code, r.site); n > r.order {
			r.order = n
		}
	}
	return r
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.issues = err
}

// Daftar mengembalikan seluruh status, terurut menurut kode.
func (r *Repo) List(_ context.Context) ([]masterstatus.ClaimStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := make([]masterstatus.ClaimStatus, 0, len(r.rows))
	for _, s := range r.rows {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

// Ambil mengembalikan satu status.
func (r *Repo) Get(_ context.Context, code string) (masterstatus.ClaimStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterstatus.ClaimStatus{}, r.issues
	}

	s, existing := r.rows[strings.TrimSpace(code)]
	if !existing {
		return masterstatus.ClaimStatus{}, masterstatus.ErrNotFound
	}
	return s, nil
}

// Sisip menyimpan status baru dengan kode yang dibentuk seperti sistem lama.
func (r *Repo) Insert(_ context.Context, label string) (masterstatus.ClaimStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterstatus.ClaimStatus{}, r.issues
	}
	if err := r.ensureLabelUnique(label, ""); err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	r.order++
	code := r.site + threeDigits(r.order)
	if _, bentrok := r.rows[code]; bentrok {
		return masterstatus.ClaimStatus{}, masterstatus.ErrCodeTaken
	}

	// LegacyCode sengaja dibiarkan kosong: penomoran `01`–`11` hanya melekat pada sebelas
	// kode pertama dan tidak pernah diberikan pada status baru.
	status := masterstatus.ClaimStatus{Code: code, Label: strings.TrimSpace(label)}
	r.rows[code] = status
	return status, nil
}

// Perbarui mengganti label status yang sudah ada.
func (r *Repo) Update(_ context.Context, code, label string) (masterstatus.ClaimStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterstatus.ClaimStatus{}, r.issues
	}

	code = strings.TrimSpace(code)
	lama, existing := r.rows[code]
	if !existing {
		return masterstatus.ClaimStatus{}, masterstatus.ErrNotFound
	}
	if err := r.ensureLabelUnique(label, code); err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	// Hanya label yang berubah; kode dan kode lama dipertahankan apa adanya.
	lama.Label = strings.TrimSpace(label)
	r.rows[code] = lama
	return lama, nil
}

// ensureLabelUnique meniru indeks unik basis data, supaya jalur gagal yang sama dapat
// diuji tanpa Oracle. Pemanggil sudah memegang kunci.
func (r *Repo) ensureLabelUnique(label, exceptCode string) error {
	wanted := masterstatus.LabelKey(label)
	for code, s := range r.rows {
		if code == exceptCode {
			continue
		}
		if masterstatus.LabelKey(s.Label) == wanted {
			return masterstatus.ErrLabelTaken
		}
	}
	return nil
}

// threeDigits meniru `lpad(to_char(seq), 3, '0')` di PEGA_M_STS_CLAIM.prc:19.
//
// Bilangan di atas 999 dikembalikan APA ADANYA, tanpa dipotong — sama seperti LPAD
// Oracle. Akibatnya kode ke-1000 menjadi lima karakter, dan itu memang cacat skema
// warisan yang sengaja tidak ditutupi di sini: menutupinya akan menghasilkan kode ganda.
func threeDigits(n int) string {
	s := ""
	for sisa := n; sisa > 0; sisa /= 10 {
		s = string(rune('0'+sisa%10)) + s
	}
	if s == "" {
		s = "0"
	}
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

// sequenceFromCode membaca kembali nomor urut dari sebuah kode, supaya penambahan
// berikutnya melanjutkan dan tidak mengulang nomor yang sudah dipakai.
func sequenceFromCode(code, site string) int {
	sisa := strings.TrimPrefix(code, site)
	if sisa == code || sisa == "" {
		return 0
	}
	n := 0
	for _, r := range sisa {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

var _ masterstatus.Repo = (*Repo)(nil)
