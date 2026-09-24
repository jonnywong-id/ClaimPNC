// Package memory adalah pengisi seam masterpenyebabkerugian.Repo yang hidup di dalam
// memori.
//
// Ia ada supaya modul dan layar yang memakainya dapat diuji tanpa basis data — adapter
// kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Master Penyebab
// Kerugian dapat dicoba lengkap sebelum koneksi ke basis data entitas tersedia DAN sebelum
// DBA menjalankan migrasi 0005.
package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/masterpenyebabkerugian"
)

// Repo menyimpan master penyebab kerugian di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu     sync.RWMutex
	rows   map[string]masterpenyebabkerugian.CauseOfLoss
	order  int
	site   string
	issues error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...masterpenyebabkerugian.CauseOfLoss) *Repo {
	r := &Repo{
		rows: make(map[string]masterpenyebabkerugian.CauseOfLoss, len(list)),
		// Meniru pembentukan ID di sistem lama: kode situs "1" ditambah urutan tiga digit
		// (`Database/PEGA_M_CAUSE_OF_LOSS.prc:20`). Nilai "1" bukan karangan — ia terbaca
		// dari ID penyebab kerugian yang dikutip aturan duplikasi klaim PA, `12002`, yang
		// berbentuk situs `1` ditambah empat digit pada tingkat rincian.
		site: "1",
	}
	for _, c := range list {
		clean := c.Clean()
		r.rows[clean.ID] = clean
		// Urutan dimulai dari nomor tertinggi yang sudah ada supaya penambahan pertama
		// menghasilkan nomor berikutnya yang wajar, bukan bentrok dengan baris contoh.
		if n := sequenceFromID(clean.ID, r.site); n > r.order {
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

// List mengembalikan seluruh golongan, terurut menurut ID.
func (r *Repo) List(_ context.Context) ([]masterpenyebabkerugian.CauseOfLoss, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return nil, r.issues
	}

	result := make([]masterpenyebabkerugian.CauseOfLoss, 0, len(r.rows))
	for _, c := range r.rows {
		result = append(result, c)
	}
	// Pengurutan memakai fungsi domain yang sama dengan adapter SQL, supaya keduanya tidak
	// dapat berbeda pendapat tentang urutan yang benar.
	masterpenyebabkerugian.SortByID(result)
	return result, nil
}

// Get mengembalikan satu golongan.
func (r *Repo) Get(_ context.Context, id string) (masterpenyebabkerugian.CauseOfLoss, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.issues != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, r.issues
	}

	c, existing := r.rows[strings.TrimSpace(id)]
	if !existing {
		return masterpenyebabkerugian.CauseOfLoss{}, masterpenyebabkerugian.ErrNotFound
	}
	return c, nil
}

// Insert menyimpan golongan baru dengan ID yang dibentuk seperti sistem lama.
//
// Penguncian di sini setara dengan urutan basis data pada adapter SQL: pembacaan nomor
// berikutnya dan penyisipannya tidak dapat disela penyimpanan lain.
func (r *Repo) Insert(_ context.Context, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, r.issues
	}

	r.order++
	id := r.site + ThreeDigits(int64(r.order))
	if _, taken := r.rows[id]; taken {
		return masterpenyebabkerugian.CauseOfLoss{}, masterpenyebabkerugian.ErrIDTaken
	}

	// Deskripsi kosong dan deskripsi ganda TIDAK ditolak di sini, dan itu bukan kelalaian
	// adapter: modul ini memang menerima keduanya (keputusan Work Owner 2026-09-20).
	// Menolaknya di sini akan membuat adapter memori berperilaku berbeda dari Oracle, dan
	// pengujian yang lolos di sini akan gagal di sana.
	//
	// LegacyID sengaja dibiarkan kosong: penomoran lama melekat pada baris warisan dan
	// tidak pernah diberikan pada golongan baru.
	cause := masterpenyebabkerugian.CauseOfLoss{ID: id, Description: strings.TrimSpace(description)}
	r.rows[id] = cause
	return cause, nil
}

// Update mengganti deskripsi golongan yang sudah ada.
func (r *Repo) Update(_ context.Context, id, description string) (masterpenyebabkerugian.CauseOfLoss, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.issues != nil {
		return masterpenyebabkerugian.CauseOfLoss{}, r.issues
	}

	id = strings.TrimSpace(id)
	previous, existing := r.rows[id]
	if !existing {
		return masterpenyebabkerugian.CauseOfLoss{}, masterpenyebabkerugian.ErrNotFound
	}

	// Hanya deskripsi yang berubah; ID dan ID lama dipertahankan apa adanya.
	previous.Description = strings.TrimSpace(description)
	r.rows[id] = previous
	return previous, nil
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0') pada procedure lama.
//
// Perilakunya sengaja sama persis dengan pasangannya di repo/sqlstore, termasuk saat
// bilangannya menembus 999 — lihat penjelasan lengkapnya di sana. Kedua adapter harus
// membentuk ID dengan cara yang sama; kalau tidak, uji yang lolos tanpa Oracle akan gagal
// dengannya.
func ThreeDigits(n int64) string {
	digits := ""
	if n == 0 {
		digits = "0"
	}
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	for len(digits) < 3 {
		digits = "0" + digits
	}
	return digits
}

// sequenceFromID membaca nomor urut dari sebuah ID yang berawalan kode situs.
//
// Mengembalikan 0 bila ID-nya tidak berbentuk seperti yang diharapkan — baris warisan
// dengan penomoran yang berbeda karena itu tidak mengacaukan nomor berikutnya, ia hanya
// tidak ikut menaikkannya.
func sequenceFromID(id, site string) int {
	rest, found := strings.CutPrefix(id, site)
	if !found || rest == "" {
		return 0
	}
	value := 0
	for _, digit := range rest {
		if digit < '0' || digit > '9' {
			return 0
		}
		value = value*10 + int(digit-'0')
	}
	return value
}

var _ masterpenyebabkerugian.Repo = (*Repo)(nil)
