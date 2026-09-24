// Package memory memenuhi seam mastersurveyors.Repo tanpa basis data.
//
// Dipakai dua hal: pengujian aturan bisnis, dan menjalankan aplikasi dengan
// PENYIMPANAN=memori supaya layar dapat dibuka sebelum migrasi 0004 tersedia di basis
// data mana pun.
//
// Ia BUKAN tiruan yang longgar. Aturan yang ditegakkannya sama persis dengan yang
// ditegakkan Oracle — keunikan nama menurut NameKey dan keunikan login — supaya cacat yang
// muncul di produksi tidak lolos di pengujian hanya karena penyimpanannya berbeda.
//
// Lapisan Adapter — memenuhi interface yang dideklarasikan Domain.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/mastersurveyors"
)

// Repo menyimpan surveyor di memori.
//
// Dijaga mutex karena satu instans dipakai bersama seluruh permintaan HTTP.
type Repo struct {
	mu sync.RWMutex

	// rows dikunci D_SURVEY_ID.
	rows map[string]mastersurveyors.Surveyor

	// sequence adalah nomor urut berikutnya, menirukan POOLDATA.D_SURVEYORS_SEQ.
	sequence int64

	// site adalah kode situs, menirukan POOLDATA.M_SITE_DATABASE.
	site string
}

// NewRepo membuat penyimpanan memori yang sudah berisi contoh data.
func NewRepo() *Repo {
	r := &Repo{
		rows: map[string]mastersurveyors.Surveyor{},
		site: "1",
	}
	for _, s := range sampleRows() {
		r.rows[s.ID] = s
	}
	// Nomor urut dimulai SESUDAH contoh data, bukan dari satu. Memulainya dari satu akan
	// membuat penambahan pertama membentuk ID yang sudah dipakai contoh data, lalu
	// menimpanya diam-diam — persis kelas cacat yang kunci utama di Oracle akan tangkap
	// tetapi penyimpanan memori tidak.
	r.sequence = int64(len(r.rows)) + 1
	return r
}

// List membaca surveyor yang cocok dengan filter.
//
// Saringannya ditulis ulang di Go, dan urutan penerapannya sengaja dibuat sama dengan
// kueri SQL-nya — termasuk bahwa antrean komite juga menuntut status masih menunggu.
func (r *Repo) List(ctx context.Context, f mastersurveyors.Filter) ([]mastersurveyors.Surveyor, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []mastersurveyors.Surveyor
	for _, s := range r.rows {
		if !matches(s, f) {
			continue
		}
		matched = append(matched, s)
	}

	// Urutan sama dengan SQL: nama lebih dulu, ID sebagai pemutus seri.
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Name != matched[j].Name {
			return matched[i].Name < matched[j].Name
		}
		return matched[i].ID < matched[j].ID
	})

	total := len(matched)

	skip := f.Offset
	if skip < 0 {
		skip = 0
	}
	if skip >= total {
		return nil, total, nil
	}
	matched = matched[skip:]

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > len(matched) {
		limit = len(matched)
	}
	return matched[:limit], total, nil
}

// Get membaca satu surveyor.
func (r *Repo) Get(ctx context.Context, id string) (mastersurveyors.Surveyor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	s, found := r.rows[strings.TrimSpace(id)]
	if !found {
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrNotFound
	}
	return s, nil
}

// FindByNameKey membaca seluruh surveyor yang namanya sama menurut NameKey.
func (r *Repo) FindByNameKey(ctx context.Context, key string) ([]mastersurveyors.Surveyor, error) {
	wanted := mastersurveyors.NameKey(key)
	if wanted == "" {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []mastersurveyors.Surveyor
	for _, s := range r.rows {
		if mastersurveyors.NameKey(s.Name) == wanted {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// FindByAppLogin membaca seluruh surveyor dengan nama login tertentu.
func (r *Repo) FindByAppLogin(ctx context.Context, login string) ([]mastersurveyors.Surveyor, error) {
	wanted := strings.ToUpper(strings.TrimSpace(login))
	if wanted == "" {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []mastersurveyors.Surveyor
	for _, s := range r.rows {
		if strings.ToUpper(strings.TrimSpace(s.AppLogin)) == wanted {
			result = append(result, s)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Insert menyimpan surveyor baru dan membentuk ID-nya.
//
// Bentuk ID-nya sama dengan yang dibentuk sqlstore — kode situs ditambah enam digit —
// supaya contoh data dan data sungguhan tidak terlihat berbeda di layar.
func (r *Repo) Insert(ctx context.Context, s mastersurveyors.Surveyor) (mastersurveyors.Surveyor, error) {
	clean := s.Clean()

	r.mu.Lock()
	defer r.mu.Unlock()

	if taken := r.nameTakenLocked(clean.Name, ""); taken {
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrNameTaken
	}
	if taken := r.loginTakenLocked(clean.AppLogin, ""); taken {
		return mastersurveyors.Surveyor{}, mastersurveyors.ErrLoginTaken
	}

	// Melewati ID yang sudah terpakai. Oracle menangkap keadaan ini lewat kunci utama;
	// penyimpanan memori tidak punya penjaga semacam itu, dan tanpa lompatan ini sebuah
	// penambahan dapat MENIMPA baris yang sudah ada tanpa satu pun pesan.
	for {
		candidate := r.site + pad6(r.sequence)
		r.sequence++
		if _, taken := r.rows[candidate]; taken {
			continue
		}
		clean.ID = candidate
		break
	}
	r.rows[clean.ID] = clean
	return clean, nil
}

// Update menulis ulang surveyor yang sudah ada.
func (r *Repo) Update(ctx context.Context, s mastersurveyors.Surveyor) error {
	clean := s.Clean()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, found := r.rows[clean.ID]; !found {
		return mastersurveyors.ErrNotFound
	}
	if taken := r.nameTakenLocked(clean.Name, clean.ID); taken {
		return mastersurveyors.ErrNameTaken
	}
	if taken := r.loginTakenLocked(clean.AppLogin, clean.ID); taken {
		return mastersurveyors.ErrLoginTaken
	}

	r.rows[clean.ID] = clean
	return nil
}

// nameTakenLocked menguji keunikan nama. Pemanggil WAJIB sudah memegang kunci tulis.
func (r *Repo) nameTakenLocked(name, exceptID string) bool {
	wanted := mastersurveyors.NameKey(name)
	if wanted == "" {
		return false
	}
	for id, s := range r.rows {
		if id == exceptID {
			continue
		}
		if mastersurveyors.NameKey(s.Name) == wanted {
			return true
		}
	}
	return false
}

// loginTakenLocked menguji keunikan login. Pemanggil WAJIB sudah memegang kunci tulis.
func (r *Repo) loginTakenLocked(login, exceptID string) bool {
	wanted := strings.ToUpper(strings.TrimSpace(login))
	if wanted == "" {
		return false
	}
	for id, s := range r.rows {
		if id == exceptID {
			continue
		}
		if strings.ToUpper(strings.TrimSpace(s.AppLogin)) == wanted {
			return true
		}
	}
	return false
}

// matches menguji satu baris terhadap filter.
func matches(s mastersurveyors.Surveyor, f mastersurveyors.Filter) bool {
	if f.Status != "" && s.Status != f.Status {
		return false
	}
	if f.Name != "" && !containsFold(s.Name, f.Name) {
		return false
	}
	if f.AppLogin != "" && !containsFold(s.AppLogin, f.AppLogin) {
		return false
	}
	if f.TypeCode != "" && strings.TrimSpace(s.TypeCode) != strings.TrimSpace(f.TypeCode) {
		return false
	}
	if f.MyCommitteeOnly {
		wanted := strings.ToUpper(strings.TrimSpace(f.CommitteeIdentity))
		// Identitas kosong tidak menyaring apa pun — sama dengan perilaku sqlstore.
		if wanted != "" {
			if strings.ToUpper(strings.TrimSpace(s.Committee)) != wanted {
				return false
			}
			if s.Status != mastersurveyors.StatusPending {
				return false
			}
		}
	}
	return true
}

// containsFold menguji pencarian sebagian tanpa peduli besar-kecil huruf.
func containsFold(haystack, needle string) bool {
	return strings.Contains(strings.ToUpper(haystack), strings.ToUpper(strings.TrimSpace(needle)))
}

// pad6 menuliskan nomor urut dalam enam digit dengan nol di depan.
func pad6(n int64) string {
	text := ""
	for v := n; v > 0; v /= 10 {
		text = string(rune('0'+v%10)) + text
	}
	if text == "" {
		text = "0"
	}
	for len(text) < 6 {
		text = "0" + text
	}
	return text
}
