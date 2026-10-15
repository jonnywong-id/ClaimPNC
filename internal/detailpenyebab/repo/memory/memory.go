// Package memory memenuhi seam detailpenyebab.Store di dalam memori proses.
//
// # Untuk apa ia ada
//
// Dua hal, dan keduanya nyata:
//
//  1. **Menjalankan aplikasi tanpa Oracle.** Modul ini belum dapat dipakai terhadap basis
//     data sungguhan sampai hak akses POOLDATA.D_CAUSE_OF_LOSS diberikan DBA; sampai itu
//     tiba, layarnya tetap dapat dibuka, diuji, dan diperlihatkan kepada Work Owner.
//  2. **Menguji aturan tanpa infrastruktur.** Uji aturan bisnis berjalan tanpa basis data,
//     tanpa jaringan, dan tanpa berkas (`14-TESTING-STRATEGY.md` §3).
//
// # Isinya HILANG saat aplikasi berhenti, dan itu disengaja
//
// Ia bukan basis data cadangan. Baris yang disimpan lewat layar akan lenyap saat proses
// dimatikan, dan itu harus dinyatakan di layar supaya tidak ada petugas yang mengira
// pekerjaannya tersimpan.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/detailpenyebab"
)

// siteCode adalah kode situs tiruan, bagian pertama setiap D_COL_ID.
//
// Nilainya dikarang — `POOLDATA.M_SITE_DATABASE` tidak ada di lingkungan ini — dan sengaja
// dibuat TIDAK menyerupai kode situs sungguhan mana pun, supaya baris contoh tidak pernah
// tertukar dengan baris produksi bila keduanya kebetulan dibandingkan.
const siteCode = "99"

// Repo menyimpan Detail Penyebab Kerugian satu portal di dalam memori.
//
// Seluruh operasinya dijaga satu mutex. Beban modul master sangat rendah — satu layar,
// beberapa petugas — sehingga kunci tunggal jauh lebih mudah dibaca daripada penguncian
// per baris, dan tidak ada bedanya bagi pengguna.
type Repo struct {
	mutex    sync.RWMutex
	rows     []detailpenyebab.CauseOfLossDetail
	sequence int64

	master   []detailpenyebab.MasterOption
	business []detailpenyebab.Business
}

// NewRepo membentuk penyimpanan kosong, tanpa satu pun baris contoh.
func NewRepo() *Repo { return &Repo{} }

// List mengembalikan baris yang cocok dengan penyaring, urut menurut Deskripsi Kerugian.
//
// Urutannya meniru `order by DESCRIPTION` pada
// `RDB List/QueryGetAllDataCauseOfLoss-SQL.xml:100`.
func (r *Repo) List(
	ctx context.Context,
	filter detailpenyebab.Filter,
) ([]detailpenyebab.CauseOfLossDetail, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))
	master := strings.TrimSpace(filter.MasterID)
	business := strings.TrimSpace(filter.BusinessID)

	list := make([]detailpenyebab.CauseOfLossDetail, 0, len(r.rows))
	for _, one := range r.rows {
		if keyword != "" && !matchesKeyword(one, keyword) {
			continue
		}
		if master != "" && one.MasterID != master {
			continue
		}
		if business != "" && !hasBusiness(one, business) {
			continue
		}

		// Daftar lini bisnis TIDAK ikut dikembalikan, sama seperti repo SQL — grid tidak
		// menampilkannya. Menyertakannya di sini akan membuat uji lulus terhadap memori
		// lalu gagal terhadap Oracle.
		summary := one
		summary.Business = nil
		summary.MasterLabel = r.labelOf(one.MasterID)
		list = append(list, summary)
	}

	sort.SliceStable(list, func(a, b int) bool {
		return strings.ToUpper(list[a].Description) < strings.ToUpper(list[b].Description)
	})
	return list, nil
}

// Get mengembalikan satu baris LENGKAP dengan daftar lini bisnisnya.
func (r *Repo) Get(
	ctx context.Context,
	id string,
) (detailpenyebab.CauseOfLossDetail, error) {
	if err := ctx.Err(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	key := strings.TrimSpace(id)
	for _, one := range r.rows {
		if one.ID != key {
			continue
		}
		found := one
		found.MasterLabel = r.labelOf(one.MasterID)
		found.Business = append([]detailpenyebab.Business(nil), one.Business...)
		return found, nil
	}
	return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
}

// Insert menyimpan satu baris baru beserta ID yang diterbitkan.
func (r *Repo) Insert(
	ctx context.Context,
	in detailpenyebab.Input,
) (detailpenyebab.CauseOfLossDetail, error) {
	if err := ctx.Err(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.sequence++
	fresh := detailpenyebab.CauseOfLossDetail{
		ID:          detailpenyebab.FormatID(siteCode, r.sequence),
		LegacyID:    in.LegacyID,
		MasterID:    in.MasterID,
		Description: in.Description,
		LossCode:    in.LossCode,
		Active:      in.Active,
		Business:    append([]detailpenyebab.Business(nil), in.Business...),
	}

	for _, one := range r.rows {
		if one.ID == fresh.ID {
			return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrIDTaken
		}
	}

	r.rows = append(r.rows, fresh)

	saved := fresh
	saved.MasterLabel = r.labelOf(fresh.MasterID)
	return saved, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	in detailpenyebab.Input,
) (detailpenyebab.CauseOfLossDetail, error) {
	if err := ctx.Err(); err != nil {
		return detailpenyebab.CauseOfLossDetail{}, err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	key := strings.TrimSpace(id)
	for index, one := range r.rows {
		if one.ID != key {
			continue
		}

		updated := detailpenyebab.CauseOfLossDetail{
			ID:          one.ID,
			LegacyID:    in.LegacyID,
			MasterID:    in.MasterID,
			Description: in.Description,
			LossCode:    in.LossCode,
			Active:      in.Active,
			Business:    append([]detailpenyebab.Business(nil), in.Business...),
		}
		r.rows[index] = updated

		saved := updated
		saved.MasterLabel = r.labelOf(updated.MasterID)
		return saved, nil
	}
	return detailpenyebab.CauseOfLossDetail{}, detailpenyebab.ErrNotFound
}

// SearchMaster mencari Master Penyebab Kerugian menurut sebutan atau kodenya.
func (r *Repo) SearchMaster(
	ctx context.Context,
	keyword string,
) ([]detailpenyebab.MasterOption, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clean := strings.ToUpper(strings.TrimSpace(keyword))
	var list []detailpenyebab.MasterOption
	for _, one := range r.master {
		if clean == "" ||
			strings.Contains(strings.ToUpper(one.Label), clean) ||
			strings.EqualFold(one.ID, strings.TrimSpace(keyword)) {
			list = append(list, one)
		}
		if len(list) >= detailpenyebab.MaxLookupRows {
			break
		}
	}
	return list, nil
}

// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
func (r *Repo) SearchBusiness(
	ctx context.Context,
	keyword string,
) ([]detailpenyebab.Business, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clean := strings.ToUpper(strings.TrimSpace(keyword))
	var list []detailpenyebab.Business
	for _, one := range r.business {
		if clean == "" ||
			strings.Contains(strings.ToUpper(one.Name), clean) ||
			strings.EqualFold(one.ID, strings.TrimSpace(keyword)) {
			list = append(list, one)
		}
		if len(list) >= detailpenyebab.MaxLookupRows {
			break
		}
	}
	return list, nil
}

// labelOf menurunkan sebutan induk dari kodenya.
//
// Pemanggil WAJIB sudah memegang kunci. Ia tidak mengunci sendiri supaya dapat dipanggil
// dari dalam operasi yang sudah mengunci — mengunci dua kali pada mutex yang sama akan
// menggantung selamanya.
func (r *Repo) labelOf(masterID string) string {
	key := strings.TrimSpace(masterID)
	if key == "" {
		return ""
	}
	for _, one := range r.master {
		if one.ID == key {
			return one.Label
		}
	}
	// Induk yang tidak ditemukan menghasilkan sebutan kosong, bukan galat — sama seperti
	// repo SQL. Lihat Repo.masterLabel di sana.
	return ""
}

// matchesKeyword menyatakan apakah satu baris cocok dengan kata kunci.
//
// Ketiga kolom yang dicari sama dengan yang dicari repo SQL: Deskripsi Kerugian, Kode
// Kehilangan, dan ID. Keduanya harus dijaga sejalan — kalau tidak, pencarian akan
// berperilaku berbeda terhadap memori dan terhadap Oracle.
func matchesKeyword(one detailpenyebab.CauseOfLossDetail, keyword string) bool {
	return strings.Contains(strings.ToUpper(one.Description), keyword) ||
		strings.Contains(strings.ToUpper(one.LossCode), keyword) ||
		strings.Contains(strings.ToUpper(one.ID), keyword)
}

// hasBusiness menyatakan apakah satu baris menempel pada sebuah lini bisnis.
func hasBusiness(one detailpenyebab.CauseOfLossDetail, businessID string) bool {
	for _, line := range one.Business {
		if strings.TrimSpace(line.ID) == businessID {
			return true
		}
	}
	return false
}

// seed mengisi penyimpanan dengan baris contoh.
//
// Ia tidak diekspor: satu-satunya jalan masuknya adalah NewSampleRepo, sehingga tidak ada
// jalur yang dapat menambahkan baris contoh ke penyimpanan yang sudah dipakai.
func (r *Repo) seed(
	rows []detailpenyebab.CauseOfLossDetail,
	master []detailpenyebab.MasterOption,
	business []detailpenyebab.Business,
) {
	r.rows = rows
	r.master = master
	r.business = business

	// Nomor urut dilanjutkan dari baris contoh yang paling besar, supaya baris yang
	// ditambahkan petugas tidak menabrak ID contoh.
	for _, one := range rows {
		digits := strings.TrimPrefix(one.ID, siteCode)
		if number, err := strconv.ParseInt(digits, 10, 64); err == nil && number > r.sequence {
			r.sequence = number
		}
	}
}
