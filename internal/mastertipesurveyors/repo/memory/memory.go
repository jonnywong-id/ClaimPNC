// Package memory adalah pengisi seam mastertipesurveyors.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master Tipe
// Surveyors dapat dicoba lengkap dengan keempat baris yang benar.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/mastertipesurveyors"
)

// Repo menyimpan master tipe surveyor di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	rows   map[string]mastertipesurveyors.SurveyorType
	order  int
	site   string
	issues error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...mastertipesurveyors.SurveyorType) *Repo {
	r := &Repo{
		rows: make(map[string]mastertipesurveyors.SurveyorType, len(list)),
		// Nilai berikut meniru pembentukan kode di sistem lama: id_site = "1" dan urutan
		// tiga digit. Urutan dimulai dari kode tertinggi yang sudah ada supaya penambahan
		// pertama menghasilkan kode berikutnya yang wajar, bukan bentrok.
		site: "1",
	}
	for _, t := range list {
		clean := t.Clean()
		r.rows[clean.Code] = clean
		if n := sequenceFromCode(clean.Code, r.site); n > r.order {
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

// List mengembalikan seluruh tipe surveyor, terurut menurut kode.
func (r *Repo) List(_ context.Context) ([]mastertipesurveyors.SurveyorType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := make([]mastertipesurveyors.SurveyorType, 0, len(r.rows))
	for _, t := range r.rows {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

// Get mengembalikan satu tipe surveyor.
func (r *Repo) Get(_ context.Context, code string) (mastertipesurveyors.SurveyorType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return mastertipesurveyors.SurveyorType{}, r.issues
	}

	t, existing := r.rows[strings.TrimSpace(code)]
	if !existing {
		return mastertipesurveyors.SurveyorType{}, mastertipesurveyors.ErrNotFound
	}
	return t, nil
}

// Insert menyimpan tipe baru dengan kode yang dibentuk seperti sistem lama.
func (r *Repo) Insert(_ context.Context, description string) (mastertipesurveyors.SurveyorType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return mastertipesurveyors.SurveyorType{}, r.issues
	}
	if err := r.ensureDescriptionUnique(description, ""); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	r.order++
	code := r.site + threeDigits(r.order)
	if _, clash := r.rows[code]; clash {
		return mastertipesurveyors.SurveyorType{}, mastertipesurveyors.ErrCodeTaken
	}

	// LegacyCode sengaja dibiarkan kosong: ia jejak penomoran sistem sebelumnya dan tidak
	// pernah diberikan pada tipe baru.
	surveyorType := mastertipesurveyors.SurveyorType{
		Code:        code,
		Description: strings.TrimSpace(description),
	}
	r.rows[code] = surveyorType
	return surveyorType, nil
}

// Update mengganti deskripsi tipe yang sudah ada.
func (r *Repo) Update(_ context.Context, code, description string) (mastertipesurveyors.SurveyorType, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return mastertipesurveyors.SurveyorType{}, r.issues
	}

	code = strings.TrimSpace(code)
	previous, existing := r.rows[code]
	if !existing {
		return mastertipesurveyors.SurveyorType{}, mastertipesurveyors.ErrNotFound
	}
	if err := r.ensureDescriptionUnique(description, code); err != nil {
		return mastertipesurveyors.SurveyorType{}, err
	}

	// Hanya deskripsi yang berubah; kode dan kode lama dipertahankan apa adanya.
	previous.Description = strings.TrimSpace(description)
	r.rows[code] = previous
	return previous, nil
}

// ensureDescriptionUnique meniru indeks unik basis data, supaya jalur gagal yang sama
// dapat diuji tanpa Oracle. Pemanggil sudah memegang kunci.
func (r *Repo) ensureDescriptionUnique(description, exceptCode string) error {
	wanted := mastertipesurveyors.DescriptionKey(description)
	for code, t := range r.rows {
		if code == exceptCode {
			continue
		}
		if mastertipesurveyors.DescriptionKey(t.Description) == wanted {
			return mastertipesurveyors.ErrDescriptionTaken
		}
	}
	return nil
}

// threeDigits meniru `lpad(to_char(seq), 3, '0')` di `Database/PEGA_M_SURVEYORS.prc:19`.
//
// Bilangan di atas 999 dikembalikan APA ADANYA, tanpa dipotong — sama seperti LPAD
// Oracle. Akibatnya kode ke-1000 menjadi lima karakter, dan itu memang cacat skema warisan
// yang sengaja tidak ditutupi di sini: menutupinya akan menghasilkan kode ganda.
func threeDigits(n int) string {
	s := ""
	for remainder := n; remainder > 0; remainder /= 10 {
		s = string(rune('0'+remainder%10)) + s
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
	remainder := strings.TrimPrefix(code, site)
	if remainder == code || remainder == "" {
		return 0
	}
	n := 0
	for _, r := range remainder {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

var _ mastertipesurveyors.Repo = (*Repo)(nil)
