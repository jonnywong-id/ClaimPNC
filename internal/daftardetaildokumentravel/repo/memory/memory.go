// Package memory adalah pengisi kedua seam modul Daftar Detail Dokumen Travel yang hidup
// di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga yang
// memungkinkan aplikasi dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul
// auth, portal, dan keempat modul master yang sudah ada.
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

	"claim-pnc/internal/daftardetaildokumentravel"
)

// sequenceDigits adalah lebar nomor urut pada ID baris detail.
//
// Bentuk ID yang sebenarnya BELUM DIKETAHUI: activity penyimpannya hilang dari export
// (`R-16`) dan DDL tabelnya belum diterima (`R-08`). Lima digit berpadding nol dipilih
// karena itulah satu-satunya skema penomoran yang benar-benar terbaca pada rumpun tabel
// ini — `Database/DOCTRAVEL_CVG.prc:20` membentuk DOCID dengan cara yang sama.
//
// Paddingnya bukan hiasan: `BrowseLstDocTravel_RD-RD.xml` mengurutkan menurut ID, dan
// kolomnya teks. Tanpa padding, "10" mendahului "9" dan urutan yang tampil di layar
// berbeda dari urutan penerbitannya.
//
// Angka ini harus sama dengan sequenceDigits pada repo/sqlstore.
const sequenceDigits = 5

// Repo menyimpan detail dokumen travel satu portal di memori.
//
// Setiap portal mendapat instans sendiri, sehingga pengembangan tanpa basis data pun
// tetap memperlihatkan perilaku yang benar: berpindah portal berarti berpindah data.
// Menyatukannya justru akan menyembunyikan kelas cacat yang paling ingin dicegah
// `ADR-0030` dan `R-20`.
type Repo struct {
	// mutex melindungi baris. Permintaan HTTP dilayani beberapa goroutine sekaligus, dan
	// penerbitan ID harus berjalan satu per satu — persis seperti transaksi pada adapter
	// SQL.
	mutex    sync.Mutex
	rows     map[string]daftardetaildokumentravel.Detail
	sequence int64
	failure  error
}

// NewRepo membentuk repo berisi baris yang diberikan.
//
// Nomor urut dimulai dari nomor tertinggi yang sudah terpakai, supaya penambahan pertama
// menghasilkan ID berikutnya yang wajar dan bukan yang bentrok.
func NewRepo(rows ...daftardetaildokumentravel.Detail) *Repo {
	r := &Repo{rows: make(map[string]daftardetaildokumentravel.Detail, len(rows))}
	for _, row := range rows {
		clean := cleanDetail(row)
		r.rows[clean.ID] = clean
		if n := sequenceFromID(clean.ID); n > r.sequence {
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

// List mengembalikan seluruh aturan, terurut seperti grid lama.
func (r *Repo) List(_ context.Context) ([]daftardetaildokumentravel.Detail, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftardetaildokumentravel.Detail, 0, len(r.rows))
	for _, row := range r.rows {
		result = append(result, row)
	}

	// Urutannya mengikuti `BrowseLstDocTravel_RD-RD.xml`: ID menaik (pySortOrder 1),
	// lalu DOCID menaik (pySortOrder 2). Perbandingan teks, bukan angka — sama seperti
	// ORDER BY pada kolom bertipe teks.
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].DocumentID < result[j].DocumentID
	})
	return result, nil
}

// Get mengembalikan satu aturan.
func (r *Repo) Get(_ context.Context, id string) (daftardetaildokumentravel.Detail, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetaildokumentravel.Detail{}, r.failure
	}

	row, exists := r.rows[strings.TrimSpace(id)]
	if !exists {
		return daftardetaildokumentravel.Detail{}, daftardetaildokumentravel.ErrNotFound
	}
	return row, nil
}

// InsertNew menerbitkan ID lalu menyimpan barisnya.
//
// Tidak ada pemeriksaan DOCID ganda maupun DOCID yang tidak ada di master, dan itu bukan
// kelalaian: Work Owner menetapkan layar ini meniru Pega apa adanya. Adapter memori yang
// lebih ketat daripada adapter SQL akan membuat uji lulus di sini lalu gagal di sana.
func (r *Repo) InsertNew(
	_ context.Context,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetaildokumentravel.Detail{}, r.failure
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	var id string
	for {
		r.sequence++
		id = formatID(r.sequence)
		if _, taken := r.rows[id]; !taken {
			break
		}
	}

	row := detailFrom(id, input)
	r.rows[id] = row
	return row, nil
}

// Update mengganti isi satu aturan.
func (r *Repo) Update(
	_ context.Context,
	id string,
	input daftardetaildokumentravel.Input,
) (daftardetaildokumentravel.Detail, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return daftardetaildokumentravel.Detail{}, r.failure
	}

	key := strings.TrimSpace(id)
	if _, exists := r.rows[key]; !exists {
		return daftardetaildokumentravel.Detail{}, daftardetaildokumentravel.ErrNotFound
	}

	row := detailFrom(key, input)
	r.rows[key] = row
	return row, nil
}

// detailFrom menyusun baris tersimpan dari isian yang dikirim layar.
func detailFrom(id string, input daftardetaildokumentravel.Input) daftardetaildokumentravel.Detail {
	return daftardetaildokumentravel.Detail{
		ID:           id,
		DocumentID:   input.DocumentID,
		DocumentName: input.DocumentName,
		Mandatory:    input.Mandatory,
		MinUpload:    input.MinUpload,
	}
}

// cleanDetail memangkas isian baris yang diberikan sebagai isi awal.
func cleanDetail(row daftardetaildokumentravel.Detail) daftardetaildokumentravel.Detail {
	return daftardetaildokumentravel.Detail{
		ID:           strings.TrimSpace(row.ID),
		DocumentID:   strings.TrimSpace(row.DocumentID),
		DocumentName: strings.TrimSpace(row.DocumentName),
		Mandatory:    row.Mandatory,
		MinUpload:    row.MinUpload,
	}
}

// formatID menyusun ID baris detail dari nomor urutnya.
func formatID(sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < sequenceDigits {
		digits = "0" + digits
	}
	return digits
}

// sequenceFromID membaca kembali nomor urut dari sebuah ID, supaya penambahan berikutnya
// melanjutkan dan tidak mengulang nomor yang sudah dipakai.
//
// ID yang bukan angka dijawab 0 dan karena itu tidak mempengaruhi nomor berikutnya —
// baris lama dapat memuat apa saja, dan melewatinya lebih baik daripada salah
// menafsirkannya.
func sequenceFromID(id string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// DocumentRepo menyimpan daftar pilihan ID Dokumen di memori.
//
// Terpisah dari Repo karena ia mengisi seam yang berbeda, dan seam itu BACA-SAJA.
type DocumentRepo struct {
	mutex   sync.Mutex
	rows    []daftardetaildokumentravel.Document
	failure error
}

// NewDocumentRepo membentuk pembaca master dokumen berisi baris yang diberikan.
func NewDocumentRepo(rows ...daftardetaildokumentravel.Document) *DocumentRepo {
	return &DocumentRepo{rows: append([]daftardetaildokumentravel.Document(nil), rows...)}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
//
// Jalur itu penting dan bukan sekadar kelengkapan: kegagalan membaca master dokumen
// TIDAK BOLEH menghalangi penyimpanan. Kode dokumen boleh diketik sendiri, sehingga yang
// hilang saat daftarnya gagal dimuat hanyalah kenyamanan memilih.
func (r *DocumentRepo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// List mengembalikan seluruh dokumen, terurut menurut ID seperti kueri lama.
func (r *DocumentRepo) List(_ context.Context) ([]daftardetaildokumentravel.Document, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := append([]daftardetaildokumentravel.Document(nil), r.rows...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

var (
	_ daftardetaildokumentravel.Repo         = (*Repo)(nil)
	_ daftardetaildokumentravel.DocumentRepo = (*DocumentRepo)(nil)
)
