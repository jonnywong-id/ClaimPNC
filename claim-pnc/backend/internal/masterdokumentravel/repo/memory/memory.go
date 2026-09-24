// Package memory adalah pengisi seam masterdokumentravel.Repo yang hidup di dalam
// memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga
// yang memungkinkan aplikasi dijalankan tanpa Oracle saat pengembangan, mengikuti pola
// modul auth, portal, dan kedua modul master yang sudah ada.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara DOCID
// diterbitkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/masterdokumentravel"
)

// defaultSite meniru isi POOLDATA.M_SITE_DATABASE pada basis data pengembangan.
//
// Nilainya sama dengan yang dipakai adapter memori modul Master Status Klaim, dan sama
// dengan kode situs yang menghasilkan kode-kode warisan yang benar-benar ada hari ini.
const defaultSite = "1"

// Repo menyimpan master dokumen travel satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus,
	// dan penerbitan DOCID harus berjalan satu per satu — persis seperti transaksi pada
	// adapter SQL.
	mutex    sync.Mutex
	rows     map[string]masterdokumentravel.TravelDocument
	site     string
	sequence int64
	failure  error
}

// NewRepo membentuk repo berisi baris yang diberikan.
//
// Nomor urut dimulai dari nomor tertinggi yang sudah terpakai, supaya penambahan
// pertama menghasilkan DOCID berikutnya yang wajar dan bukan yang bentrok.
func NewRepo(rows ...masterdokumentravel.TravelDocument) *Repo {
	r := &Repo{
		rows: make(map[string]masterdokumentravel.TravelDocument, len(rows)),
		site: defaultSite,
	}
	for _, doc := range rows {
		clean := masterdokumentravel.TravelDocument{
			ID:   strings.TrimSpace(doc.ID),
			Name: strings.TrimSpace(doc.Name),
		}
		r.rows[clean.ID] = clean
		if n := sequenceFromID(clean.ID, r.site); n > r.sequence {
			r.sequence = n
		}
	}
	return r
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh dokumen, terurut menurut DOCID seperti kueri lama.
func (r *Repo) List(_ context.Context) ([]masterdokumentravel.TravelDocument, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]masterdokumentravel.TravelDocument, 0, len(r.rows))
	for _, doc := range r.rows {
		result = append(result, doc)
	}
	// Perbandingan teks, bukan angka — sama seperti ORDER BY DOCID pada kueri lama,
	// yang juga mengurutkan kolom bertipe teks secara leksikografis.
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu dokumen.
func (r *Repo) Get(_ context.Context, id string) (masterdokumentravel.TravelDocument, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterdokumentravel.TravelDocument{}, r.failure
	}

	doc, exists := r.rows[strings.TrimSpace(id)]
	if !exists {
		return masterdokumentravel.TravelDocument{}, masterdokumentravel.ErrNotFound
	}
	return doc, nil
}

// InsertNew menerbitkan DOCID lalu menyimpan barisnya.
//
// Tidak ada pemeriksaan judul ganda, dan itu bukan kelalaian: Work Owner menetapkan
// layar ini meniru Pega apa adanya, dan `DOCTRAVEL_CVG.prc` pun menyisipkan tanpa
// memeriksa apa pun. Adapter memori yang lebih ketat daripada adapter SQL akan membuat
// uji lulus di sini lalu gagal di sana.
func (r *Repo) InsertNew(_ context.Context, input masterdokumentravel.Input) (masterdokumentravel.TravelDocument, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return masterdokumentravel.TravelDocument{}, r.failure
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat DOCID berbentuk lain, supaya baris baru tidak menabraknya.
	var id string
	for {
		r.sequence++
		id = masterdokumentravel.FormatID(r.site, r.sequence)
		if _, taken := r.rows[id]; !taken {
			break
		}
	}

	doc := masterdokumentravel.TravelDocument{ID: id, Name: input.Name}
	r.rows[id] = doc
	return doc, nil
}

// Update mengganti judul dokumen yang sudah ada.
func (r *Repo) Update(_ context.Context, doc masterdokumentravel.TravelDocument) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	id := strings.TrimSpace(doc.ID)
	if _, exists := r.rows[id]; !exists {
		return masterdokumentravel.ErrNotFound
	}
	// Hanya judul yang berubah; DOCID dipertahankan apa adanya.
	r.rows[id] = masterdokumentravel.TravelDocument{ID: id, Name: doc.Name}
	return nil
}

// sequenceFromID membaca kembali nomor urut dari sebuah DOCID, supaya penambahan
// berikutnya melanjutkan dan tidak mengulang nomor yang sudah dipakai.
//
// DOCID yang tidak berbentuk "<situs><angka>" dijawab 0 dan karena itu tidak
// mempengaruhi nomor berikutnya — baris lama dapat memuat apa saja, dan melewatinya
// lebih baik daripada salah menafsirkannya.
func sequenceFromID(id, site string) int64 {
	rest := strings.TrimPrefix(id, site)
	if rest == id || rest == "" {
		return 0
	}
	n, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

var _ masterdokumentravel.Repo = (*Repo)(nil)
