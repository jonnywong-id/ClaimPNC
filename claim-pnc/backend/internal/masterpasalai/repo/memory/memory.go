// Package memory memenuhi seam masterpasalai.Repo tanpa basis data.
//
// Ia adapter KEDUA di balik seam yang sama, dan itu yang membuat seam-nya nyata alih-alih
// hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Dua kegunaannya:
//
//	pengujian      aturan modul dapat diuji tanpa Oracle dan tanpa jaringan
//	pengembangan   layar dapat dicoba utuh sebelum hak akses tabelnya diberikan DBA
//
// Setiap keputusan di sini menyebutkan **padanan SQL-nya**, supaya keduanya tidak berpisah
// diam-diam: dua adapter yang berbeda perilakunya membuat uji yang lulus di memori gagal di
// Oracle, dan itu kelas cacat yang paling mahal ditemukan belakangan.
//
// Ia TIDAK dipakai di produksi. `cmd/claimpnc` menolak penyimpanan memori di lingkungan
// produksi jauh sebelum modul ini dirakit.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/masterpasalai"
)

// Repo menyimpan wording polis di memori, satu portal.
//
// Mutex-nya bukan formalitas: satu instans dipakai bersama seluruh permintaan sebuah portal,
// dan peramban yang membuka dua tab sudah cukup untuk membuat dua permintaan berjalan
// bersamaan.
type Repo struct {
	mutex sync.RWMutex

	// clause disimpan sebagai daftar, bukan map, supaya urutannya dapat dibuat sama persis
	// dengan `ORDER BY WP_ID` pada kueri SQL-nya.
	clause []masterpasalai.Clause
}

// NewRepo membentuk penyimpanan berisi baris awal yang diberikan.
//
// Baris yang ketiga isiannya kosong dibuang: ia tidak menunjuk apa pun, dan membiarkannya
// hanya membuat cacah total berbeda dari yang terlihat di layar.
func NewRepo(clause []masterpasalai.Clause) *Repo {
	kept := make([]masterpasalai.Clause, 0, len(clause))
	for _, one := range clause {
		if one.IsEmpty() {
			continue
		}
		kept = append(kept, one)
	}
	return &Repo{clause: kept}
}

// List menyaring, mencacah, lalu memotong satu halaman — urutan yang sama dengan Pega.
//
// # Penyaringnya meniru `Activity/GetListPasalAI_act.xml:548` apa adanya
//
//	WHERE (WP_PASAL LIKE '%kw%' OR WP_AYAT LIKE '%kw%' OR WP_KEJADIAN LIKE '%kw%')
//
// Satu kata kunci, ketiga kolom, digabung `OR`. Kata kunci kosong berarti tanpa `WHERE`
// sama sekali (`:379`), bukan `LIKE '%%'`.
//
// Perbandingannya TIDAK bergantung besar-kecil huruf. Pega membandingkan apa adanya di sisi
// basis data, dan Oracle peka huruf pada `LIKE`; adapter SQL nanti memasang `UPPER` di kedua
// sisi. Yang di sini memakai `strings.EqualFold` lewat `ToUpper` supaya keduanya menjawab
// sama — dua adapter yang berbeda perilakunya membuat uji yang lulus di memori gagal di
// Oracle, dan itu kelas cacat yang paling mahal ditemukan belakangan.
//
// # Urutannya `ORDER BY WP_ID`, sama dengan kuerinya
//
// Diurutkan menurut ID, meniru `ORDER BY WP_ID` pada
// `RDB List/GetListDataPasalAI_SQL.xml`. Ia pengurutan **teks** pada kolom bertipe teks,
// sehingga "10" mendahului "9" — diterima apa adanya, karena tipe kolomnya belum diketahui
// (`R-08`) dan memperbaikinya di sini berarti menebak bahwa isinya selalu angka.
//
// Tanpa urutan yang sama persis, dua adapter akan mengembalikan halaman yang berbeda untuk
// permintaan yang sama, dan selisihnya baru terlihat setelah datanya banyak.
func (r *Repo) List(
	ctx context.Context,
	filter masterpasalai.Filter,
) (masterpasalai.Page, error) {
	_ = ctx

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	matched := make([]masterpasalai.Clause, 0, len(r.clause))
	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))
	for _, one := range r.clause {
		if keyword == "" || matches(one, keyword) {
			matched = append(matched, one)
		}
	}

	// ORDER BY WP_ID — pengurutan teks, sama seperti kuerinya. Memakai perbandingan angka
	// di sini akan membuat uji urutan lulus di memori dan gagal di Oracle.
	sort.SliceStable(matched, func(a, b int) bool {
		return matched[a].ID < matched[b].ID
	})

	page := masterpasalai.Page{
		Total:  len(matched),
		Number: filter.Page,
	}

	// Pemotongan halaman dilakukan SETELAH mencacah, bukan sebelumnya. Cacah yang dihitung
	// atas halaman akan membuat paginator melaporkan satu halaman berapa pun isinya.
	offset := filter.Offset()
	if offset >= len(matched) {
		// Halaman di luar jangkauan dijawab halaman KOSONG, bukan galat. Nomor halaman
		// datang dari tautan paginasi, dan tautan yang basi bukan kesalahan pengguna.
		// Total tetap dikembalikan supaya layar dapat menunjukkan halaman mana yang ada.
		return page, nil
	}

	end := offset + masterpasalai.PageSize
	if end > len(matched) {
		end = len(matched)
	}

	// Salinan dibuat, bukan potongan langsung: potongan tetap menunjuk senarai milik repo,
	// dan pemanggil yang menyuntingnya akan mengubah isi penyimpanan.
	page.Clause = append([]masterpasalai.Clause(nil), matched[offset:end]...)
	return page, nil
}

// matches mencocokkan satu baris dengan kata kunci yang SUDAH di-uppercase.
//
// Ketiga kolom diperiksa, dan pemeriksaannya berhenti pada yang pertama cocok — persis
// seperti `OR` pada kueri aslinya.
func matches(one masterpasalai.Clause, upperKeyword string) bool {
	for _, field := range []string{one.Number, one.Paragraph, one.Event} {
		if strings.Contains(strings.ToUpper(field), upperKeyword) {
			return true
		}
	}
	return false
}

// CountAll mencacah seluruh baris. Dipakai pemeriksaan kesiapan.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	_ = ctx

	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return len(r.clause), nil
}

var _ masterpasalai.Repo = (*Repo)(nil)
