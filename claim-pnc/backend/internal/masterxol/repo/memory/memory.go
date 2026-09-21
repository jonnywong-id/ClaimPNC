// Package memory memenuhi seam masterxol.Repo di dalam memori proses.
//
// Ia dipakai pengembangan dan pengujian: aturan modul dapat diuji tanpa Oracle sama
// sekali, dan itulah yang membuat seam Repo benar-benar seam dan bukan abstraksi
// hipotetis (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3.1).
//
// Isinya TIDAK pernah dipakai di produksi. Pemilihannya ada di cmd/claimpnc, lewat
// PENYIMPANAN=memori.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterxol"
)

// Repo menyimpan Master XOL di memori proses.
//
// Seluruh operasi dijaga satu mutex. Sederhana dan memadai: penyimpanan ini melayani
// pengembangan dan pengujian, bukan beban produksi.
type Repo struct {
	mutex sync.RWMutex

	// master dikunci dengan ID induk.
	master map[string]masterxol.Master

	// year dan businessGroup adalah data acuan yang tidak berubah selama proses hidup.
	year          []string
	businessGroup []businessGroupRow
}

// businessGroupRow adalah satu baris pilihan grup bisnis beserta nama grup treaty yang
// menentukan ia muncul pada Type XOL yang mana.
type businessGroupRow struct {
	business  masterxol.Business
	treatyTag string
}

// NewRepo membentuk penyimpanan kosong tanpa satu pun baris acuan.
func NewRepo() *Repo {
	return &Repo{master: map[string]masterxol.Master{}}
}

// NewSampleRepo membentuk penyimpanan berisi contoh yang meniru bentuk data produksi.
func NewSampleRepo() *Repo {
	r := NewRepo()
	for _, m := range SampleMaster() {
		r.master[m.ID] = m
	}
	r.year = SampleYear()
	r.businessGroup = sampleBusinessGroup()
	return r
}

// List mengembalikan seluruh induk tanpa anaknya, terurut numerik menurut ID.
func (r *Repo) List(_ context.Context) ([]masterxol.Master, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]masterxol.Master, 0, len(r.master))
	for _, m := range r.master {
		m = m.Clean()
		// Anaknya dibuang, sama seperti yang dilakukan kueri daftar di sqlstore. Bila di
		// sini ikut terbawa, uji tidak akan menangkap pemanggil yang diam-diam
		// bergantung pada anaknya ada di daftar.
		m.Business = nil
		m.Layer = nil
		result = append(result, m)
	}
	sortByID(result)
	return result, nil
}

// Get mengembalikan satu induk lengkap dengan anaknya.
func (r *Repo) Get(_ context.Context, id string) (masterxol.Master, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	m, exists := r.master[strings.TrimSpace(id)]
	if !exists {
		return masterxol.Master{}, masterxol.ErrNotFound
	}

	// Dirapikan saat DIBACA, sama seperti yang dilakukan sqlstore.
	//
	// Bukan sekadar kerapian: Clean yang menghitung ConvertedLimit dari Limit dan kurs
	// induk. Tanpa ini, baris contoh yang dimuat apa adanya akan mengembalikan
	// ConvertedLimit nol — dan penyimpanan memori berperilaku berbeda dari penyimpanan
	// sungguhan, sehingga uji terhadapnya berhenti membuktikan apa pun.
	return cloneMaster(m).Clean(), nil
}

// Save menyimpan satu induk beserta anaknya; ID kosong berarti menambah.
func (r *Repo) Save(_ context.Context, master masterxol.Master) (masterxol.Master, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	master = master.Clean()

	if master.ID == "" {
		master.ID = masterxol.NextID(r.idList())
		if _, taken := r.master[master.ID]; taken {
			return masterxol.Master{}, masterxol.ErrIDTaken
		}
	} else {
		existing, exists := r.master[master.ID]
		if !exists {
			return masterxol.Master{}, masterxol.ErrNotFound
		}
		// Kolom komite tidak pernah ikut berubah lewat Simpan. Yang mengubahnya adalah
		// SubmitToCommittee dan layar persetujuan komite — bukan form ini.
		master.PIC = existing.PIC
		master.CommitteeStatus = existing.CommitteeStatus
		master.Committee = existing.Committee
		master.RemarkCommittee = existing.RemarkCommittee
	}

	master.Layer = r.withLayerID(master.ID, master.Layer)
	r.master[master.ID] = cloneMaster(master)
	return cloneMaster(master), nil
}

// withLayerID memberi nomor pada lapisan yang belum punya.
//
// Nomornya diterbitkan dari seluruh lapisan di SELURUH induk, bukan hanya induk ini —
// IDLAYER adalah kunci utama tabelnya sendiri, dan produksi membuktikannya berjalan
// global: nomor 10001 sampai 10018 tersebar di delapan induk.
func (r *Repo) withLayerID(masterID string, list []masterxol.Layer) []masterxol.Layer {
	used := r.layerIDList()
	result := make([]masterxol.Layer, 0, len(list))
	for _, l := range list {
		if l.ID == "" {
			l.ID = masterxol.NextID(used)
			used = append(used, l.ID)
		}
		result = append(result, l)
	}
	_ = masterID
	return result
}

// DeleteMaster menghapus satu induk beserta seluruh anaknya.
func (r *Repo) DeleteMaster(_ context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id = strings.TrimSpace(id)
	if _, exists := r.master[id]; !exists {
		return masterxol.ErrNotFound
	}
	// Anaknya ikut terbuang karena ia tersimpan di dalam induknya. Di sqlstore kaskadenya
	// harus dinyatakan sendiri.
	delete(r.master, id)
	return nil
}

// DeleteBusiness menghapus satu baris grup bisnis.
func (r *Repo) DeleteBusiness(_ context.Context, masterID, businessID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m, exists := r.master[strings.TrimSpace(masterID)]
	if !exists {
		return masterxol.ErrNotFound
	}

	businessID = strings.TrimSpace(businessID)
	kept := make([]masterxol.Business, 0, len(m.Business))
	for _, b := range m.Business {
		if b.ID == businessID {
			continue
		}
		kept = append(kept, b)
	}
	m.Business = kept
	r.master[m.ID] = m
	return nil
}

// DeleteLayer menghapus satu lapisan beserta seluruh reas-nya.
func (r *Repo) DeleteLayer(_ context.Context, masterID, layerID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m, exists := r.master[strings.TrimSpace(masterID)]
	if !exists {
		return masterxol.ErrNotFound
	}

	layerID = strings.TrimSpace(layerID)
	kept := make([]masterxol.Layer, 0, len(m.Layer))
	found := false
	for _, l := range m.Layer {
		if l.ID == layerID {
			found = true
			continue
		}
		kept = append(kept, l)
	}
	if !found {
		return masterxol.ErrLayerNotFound
	}
	m.Layer = kept
	r.master[m.ID] = m
	return nil
}

// DeleteReinsurer menghapus satu baris reas dari sebuah lapisan.
func (r *Repo) DeleteReinsurer(_ context.Context, layerID, reinsurerID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	layerID = strings.TrimSpace(layerID)
	reinsurerID = strings.TrimSpace(reinsurerID)

	for id, m := range r.master {
		for index, l := range m.Layer {
			if l.ID != layerID {
				continue
			}
			kept := make([]masterxol.Reinsurer, 0, len(l.Reinsurer))
			for _, reas := range l.Reinsurer {
				if reas.ID == reinsurerID {
					continue
				}
				kept = append(kept, reas)
			}
			m.Layer[index].Reinsurer = kept
			r.master[id] = m
			return nil
		}
	}
	return masterxol.ErrLayerNotFound
}

// ListYear mengembalikan pilihan tahun.
func (r *Repo) ListYear(_ context.Context) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return append([]string(nil), r.year...), nil
}

// ListBusinessGroup mengembalikan pilihan grup bisnis untuk sebuah Type XOL.
//
// Penyaringnya memakai pola yang sama dengan sqlstore — lihat
// masterxol.BusinessGroupPattern — sehingga uji terhadap penyimpanan ini menguji aturan
// yang benar-benar berlaku, bukan aturan yang disederhanakan.
func (r *Repo) ListBusinessGroup(_ context.Context, t masterxol.Type) ([]masterxol.Business, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	pattern := masterxol.BusinessGroupPattern(t)
	var result []masterxol.Business
	for _, row := range r.businessGroup {
		if matchAny(row.treatyTag, pattern) {
			result = append(result, row.business)
		}
	}

	// Baris TREATY INWARD selalu ikut, tanpa ID — meniru
	// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml` yang menambahkannya pada setiap
	// Type.
	result = append(result, masterxol.Business{Name: masterxol.TreatyInwardName})
	return result, nil
}

// matchAny menirukan `LIKE '%X%'` untuk pola yang dipakai modul ini.
//
// Cukup memeriksa isi di antara kedua tanda persen: seluruh pola di
// masterxol.BusinessGroupPattern berbentuk `%…%`, dan pola `%` berarti apa saja.
func matchAny(value string, pattern []string) bool {
	upper := strings.ToUpper(value)
	for _, p := range pattern {
		needle := strings.ToUpper(strings.Trim(p, "%"))
		if needle == "" || strings.Contains(upper, needle) {
			return true
		}
	}
	return false
}

// SubmitToCommittee mencatat pengajuan sebuah induk ke komite.
func (r *Repo) SubmitToCommittee(_ context.Context, id, pic, remark string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	m, exists := r.master[strings.TrimSpace(id)]
	if !exists {
		return masterxol.ErrNotFound
	}
	m.PIC = strings.TrimSpace(pic)
	m.CommitteeStatus = masterxol.CommitteePending
	m.RemarkPIC = strings.TrimSpace(remark)
	r.master[m.ID] = m
	return nil
}

func (r *Repo) idList() []string {
	result := make([]string, 0, len(r.master))
	for id := range r.master {
		result = append(result, id)
	}
	return result
}

func (r *Repo) layerIDList() []string {
	var result []string
	for _, m := range r.master {
		for _, l := range m.Layer {
			result = append(result, l.ID)
		}
	}
	return result
}

// cloneMaster menyalin dalam, supaya pemanggil tidak dapat mengubah isi penyimpanan
// dengan menyunting senarai yang ia terima.
func cloneMaster(m masterxol.Master) masterxol.Master {
	m.Business = append([]masterxol.Business(nil), m.Business...)

	layer := make([]masterxol.Layer, 0, len(m.Layer))
	for _, l := range m.Layer {
		l.Reinsurer = append([]masterxol.Reinsurer(nil), l.Reinsurer...)
		layer = append(layer, l)
	}
	m.Layer = layer
	return m
}

// sortByID mengurutkan numerik, bukan leksikografis.
//
// Perlu karena ID tidak bernol di depan: sebagai teks, "10010" mendahului "1009".
func sortByID(list []masterxol.Master) {
	sort.SliceStable(list, func(i, j int) bool {
		left, leftErr := strconv.ParseInt(list[i].ID, 10, 64)
		right, rightErr := strconv.ParseInt(list[j].ID, 10, 64)
		if leftErr != nil || rightErr != nil {
			return list[i].ID < list[j].ID
		}
		return left < right
	})
}

var _ masterxol.Repo = (*Repo)(nil)
