// Package memory memenuhi seam masterpasal.Store tanpa basis data.
//
// Ia adapter KEDUA di balik seam yang sama, dan itu yang membuat seam-nya nyata alih-alih
// hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Dua kegunaannya:
//
//	pengujian      aturan modul dapat diuji tanpa Oracle dan tanpa jaringan
//	pengembangan   layar dapat dicoba utuh sebelum hak akses tabelnya diberikan DBA
//
// Ia TIDAK dipakai di produksi. `cmd/claimpnc` menolak penyimpanan memori di lingkungan
// produksi jauh sebelum modul ini dirakit.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterpasal"
)

// Repo menyimpan pasal kerugian di memori, satu portal.
//
// Mutex-nya bukan formalitas: satu instans dipakai bersama seluruh permintaan sebuah
// portal, dan peramban yang membuka dua tab sudah cukup untuk membuat dua permintaan
// berjalan bersamaan.
type Repo struct {
	mutex sync.RWMutex

	// clause disimpan sebagai daftar, bukan map, supaya urutan penyisipan terjaga dan
	// pengurutannya dapat dibuat sama persis dengan kueri SQL-nya.
	clause []masterpasal.Clause

	// business adalah master lini bisnis tiruan — padanan POOLDATA.BUSINESS.
	business []masterpasal.Business
}

// NewRepo membentuk penyimpanan berisi baris awal yang diberikan.
func NewRepo(clause []masterpasal.Clause, business []masterpasal.Business) *Repo {
	return &Repo{
		clause:   append([]masterpasal.Clause(nil), clause...),
		business: append([]masterpasal.Business(nil), business...),
	}
}

// List mengembalikan seluruh pasal, terurut seperti kueri `clause_list`.
//
// Daftar lini bisnisnya dibuang, sama seperti adapter SQL. Dua adapter yang mengembalikan
// bentuk berbeda untuk seam yang sama akan membuat uji yang lulus di memori gagal di
// Oracle — justru kelas cacat yang paling mahal ditemukan belakangan.
func (r *Repo) List(ctx context.Context) ([]masterpasal.Clause, error) {
	_ = ctx

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]masterpasal.Clause, 0, len(r.clause))
	for _, clause := range r.clause {
		clause.Business = nil
		result = append(result, clause)
	}

	// ORDER BY IDPASAL ASC, IDDATA ASC — pengurutan TEKS, sama seperti kuerinya. Memakai
	// perbandingan angka di sini akan membuat uji urutan lulus di memori dan gagal di
	// Oracle.
	sort.SliceStable(result, func(a, b int) bool {
		if result[a].Number != result[b].Number {
			return result[a].Number < result[b].Number
		}
		return result[a].ID < result[b].ID
	})
	return result, nil
}

// Get mengembalikan satu pasal beserta daftar lini bisnisnya.
//
// Nama lini bisnisnya DISEGARKAN dari master tiruan, persis seperti adapter SQL — dan
// butir yang kodenya tidak ketemu mempertahankan nama tersimpannya, juga persis sama.
func (r *Repo) Get(ctx context.Context, id string) (masterpasal.Clause, error) {
	_ = ctx

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	position := r.find(id)
	if position < 0 {
		return masterpasal.Clause{}, masterpasal.ErrNotFound
	}

	clause := r.clause[position]
	clause.Business = r.refresh(clause.Business)
	return clause, nil
}

// Insert menyimpan satu pasal baru.
func (r *Repo) Insert(ctx context.Context, in masterpasal.Input) (masterpasal.Clause, error) {
	_ = ctx

	r.mutex.Lock()
	defer r.mutex.Unlock()

	used := make([]string, 0, len(r.clause))
	for _, clause := range r.clause {
		used = append(used, clause.ID)
	}

	fresh := masterpasal.Clause{
		ID:            masterpasal.NextSequence(used),
		Number:        in.Number,
		Text:          in.Text,
		Description:   in.Description,
		Category:      in.Category,
		CategoryLabel: masterpasal.CategoryLabel(in.Category),
		Business:      append([]masterpasal.Business(nil), in.Business...),
	}
	r.clause = append(r.clause, fresh)
	return fresh, nil
}

// Update menyimpan perubahan pada pasal yang sudah ada.
func (r *Repo) Update(ctx context.Context, id string, in masterpasal.Input) (masterpasal.Clause, error) {
	_ = ctx

	r.mutex.Lock()
	defer r.mutex.Unlock()

	position := r.find(id)
	if position < 0 {
		return masterpasal.Clause{}, masterpasal.ErrNotFound
	}

	saved := masterpasal.Clause{
		// ID diambil dari baris yang tersimpan, bukan dari argumen: argumennya sudah
		// dipangkas pemanggil, dan menuliskannya kembali akan diam-diam mengubah kunci
		// baris yang kebetulan tersimpan dengan spasi tepi.
		ID:            r.clause[position].ID,
		Number:        in.Number,
		Text:          in.Text,
		Description:   in.Description,
		Category:      in.Category,
		CategoryLabel: masterpasal.CategoryLabel(in.Category),
		Business:      append([]masterpasal.Business(nil), in.Business...),
	}
	r.clause[position] = saved
	return saved, nil
}

// Delete membuang satu baris.
func (r *Repo) Delete(ctx context.Context, id string) error {
	_ = ctx

	r.mutex.Lock()
	defer r.mutex.Unlock()

	position := r.find(id)
	if position < 0 {
		return masterpasal.ErrNotFound
	}
	r.clause = append(r.clause[:position], r.clause[position+1:]...)
	return nil
}

// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
//
// Pencocokannya meniru kueri SQL-nya: tidak peka besar-kecil huruf pada nama, dan cocok
// persis pada kode. Batas MaxLookupRows ikut diberlakukan supaya bentuk hasilnya sama.
func (r *Repo) SearchBusiness(ctx context.Context, keyword string) ([]masterpasal.Business, error) {
	_ = ctx

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clean := strings.ToUpper(strings.TrimSpace(keyword))
	if clean == "" {
		return nil, nil
	}

	var result []masterpasal.Business
	for _, business := range r.business {
		if business.ID == "" {
			continue
		}
		matched := strings.Contains(strings.ToUpper(business.Name), clean) ||
			strings.EqualFold(business.ID, clean)
		if !matched {
			continue
		}
		result = append(result, business)
		if len(result) == masterpasal.MaxLookupRows {
			break
		}
	}

	sort.SliceStable(result, func(a, b int) bool { return result[a].Name < result[b].Name })
	return result, nil
}

// find mengembalikan posisi baris ber-IDDATA tertentu, atau -1.
//
// Pembandingnya dipangkas di kedua sisi dengan alasan yang sama seperti `TRIM` pada
// kueri SQL-nya: baris lama dapat tersimpan dengan spasi tepi.
func (r *Repo) find(id string) int {
	key := strings.TrimSpace(id)
	for position, clause := range r.clause {
		if strings.TrimSpace(clause.ID) == key {
			return position
		}
	}
	return -1
}

// refresh menyegarkan nama setiap lini bisnis dari master tiruan.
func (r *Repo) refresh(stored []masterpasal.Business) []masterpasal.Business {
	if len(stored) == 0 {
		return nil
	}

	result := make([]masterpasal.Business, 0, len(stored))
	for _, business := range stored {
		code := strings.TrimSpace(business.ID)
		if code != "" {
			for _, master := range r.business {
				if strings.EqualFold(strings.TrimSpace(master.ID), code) && master.Name != "" {
					business.Name = master.Name
					break
				}
			}
		}
		result = append(result, business)
	}
	return result
}
