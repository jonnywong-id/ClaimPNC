// Package memory adalah pengisi seam daftartipedokumen.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga yang
// memungkinkan aplikasi dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul
// auth, portal, dan seluruh modul master yang sudah ada.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya dan cara ID
// diterbitkan — kalau tidak, uji yang lulus di sini tidak membuktikan apa pun tentang
// adapter SQL.
package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"claim-pnc/internal/daftartipedokumen"
)

// defaultSite meniru isi POOLDATA.M_SITE_DATABASE pada basis data pengembangan.
//
// Nilainya sama dengan yang dipakai adapter memori modul master lainnya, dan sama dengan
// kode situs yang menghasilkan kode-kode warisan yang benar-benar ada hari ini.
const defaultSite = "1"

// Repo menyimpan daftar tipe dokumen satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun tetap
// memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penerbitan ID harus berjalan satu per satu — persis seperti transaksi pada adapter
	// SQL.
	mutex    sync.Mutex
	rows     map[string]daftartipedokumen.DocumentType
	site     string
	sequence int64
	failure  error
}

// NewRepo membentuk repo berisi baris yang diberikan.
//
// Nomor urut dimulai dari nomor tertinggi yang sudah terpakai, supaya penambahan pertama
// menghasilkan ID berikutnya yang wajar dan bukan yang bentrok.
func NewRepo(rows ...daftartipedokumen.DocumentType) *Repo {
	r := &Repo{
		rows: make(map[string]daftartipedokumen.DocumentType, len(rows)),
		site: defaultSite,
	}
	for _, doc := range rows {
		clean := daftartipedokumen.DocumentType{
			ID:            strings.TrimSpace(doc.ID),
			Type:          strings.TrimSpace(doc.Type),
			ProcessStatus: strings.TrimSpace(doc.ProcessStatus),
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

// List mengembalikan seluruh tipe dokumen, terurut menurut ID seperti kueri lama.
func (r *Repo) List(_ context.Context) ([]daftartipedokumen.DocumentType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftartipedokumen.DocumentType, 0, len(r.rows))
	for _, doc := range r.rows {
		result = append(result, doc)
	}
	// Perbandingan teks, bukan angka — sama seperti ORDER BY ID pada kueri lama, yang juga
	// mengurutkan kolom bertipe teks secara leksikografis.
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// Get mengembalikan satu tipe dokumen.
func (r *Repo) Get(_ context.Context, id string) (daftartipedokumen.DocumentType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftartipedokumen.DocumentType{}, r.failure
	}

	doc, exists := r.rows[strings.TrimSpace(id)]
	if !exists {
		return daftartipedokumen.DocumentType{}, daftartipedokumen.ErrNotFound
	}
	return doc, nil
}

// InsertNew menerbitkan ID lalu menyimpan barisnya.
//
// Tidak ada pemeriksaan nama ganda, dan itu bukan kelalaian: Work Owner menetapkan layar
// ini meniru Pega apa adanya, dan `PEGA_LST_DOC_TYPE.prc` pun menyisipkan tanpa memeriksa
// apa pun. Adapter memori yang lebih ketat daripada adapter SQL akan membuat uji lulus di
// sini lalu gagal di sana.
//
// Editor tidak disimpan: USER_EDIT dan TGL_EDIT tidak pernah dibaca kembali modul ini —
// baik grid maupun form di layar Pega tidak menampilkan keduanya. Menyimpannya di sini
// berarti menyediakan nilai yang tidak pernah dibandingkan dengan apa pun, dan uji yang
// memakainya akan menguji adapter memori, bukan perilaku yang dilihat pengguna.
func (r *Repo) InsertNew(
	_ context.Context,
	input daftartipedokumen.Input,
	_ daftartipedokumen.Editor,
) (daftartipedokumen.DocumentType, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftartipedokumen.DocumentType{}, r.failure
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	var id string
	for {
		r.sequence++
		id = daftartipedokumen.FormatID(r.site, r.sequence)
		if _, taken := r.rows[id]; !taken {
			break
		}
	}

	doc := daftartipedokumen.DocumentType{
		ID:            id,
		Type:          input.Type,
		ProcessStatus: input.ProcessStatus,
	}
	r.rows[id] = doc
	return doc, nil
}

// Update mengganti isi baris yang sudah ada.
func (r *Repo) Update(
	_ context.Context,
	doc daftartipedokumen.DocumentType,
	_ daftartipedokumen.Editor,
) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return r.failure
	}

	id := strings.TrimSpace(doc.ID)
	if _, exists := r.rows[id]; !exists {
		return daftartipedokumen.ErrNotFound
	}
	// Hanya isian yang berubah; ID dipertahankan apa adanya.
	r.rows[id] = daftartipedokumen.DocumentType{
		ID:            id,
		Type:          doc.Type,
		ProcessStatus: doc.ProcessStatus,
	}
	return nil
}

// sequenceFromID membaca kembali nomor urut dari sebuah ID, supaya penambahan berikutnya
// melanjutkan dan tidak mengulang nomor yang sudah dipakai.
//
// ID yang tidak berbentuk "<situs><angka>" dijawab 0 dan karena itu tidak mempengaruhi
// nomor berikutnya — baris lama dapat memuat apa saja, dan melewatinya lebih baik daripada
// salah menafsirkannya.
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

var _ daftartipedokumen.Repo = (*Repo)(nil)
