// Package memory adalah pengisi ketiga seam modul Daftar Detail Dokumen Travel yang
// hidup di dalam memori.
//
// Ia ada supaya modul dan layarnya dapat diuji tanpa basis data — adapter kedua yang
// membuat seam ini nyata, bukan hipotetis (`04-FUTURE-ARCHITECTURE.md` §3). Ia juga yang
// memungkinkan aplikasi dijalankan tanpa Oracle saat pengembangan, mengikuti pola modul
// auth, portal, dan keempat modul master yang sudah ada.
//
// Yang ditiru bukan hanya bentuk datanya, tetapi juga urutan barisnya, cara ID
// diterbitkan, dan cara daftar coverage DIGANTI SELURUHNYA saat disimpan — kalau tidak,
// uji yang lulus di sini tidak membuktikan apa pun tentang adapter SQL.
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

		// ID baris coverage ikut diperhitungkan, bukan hanya ID induknya: keduanya
		// berbagi satu urutan, sama seperti di adapter SQL. Melewatkannya akan membuat
		// penambahan pertama menerbitkan nomor yang sudah dipakai sebuah baris coverage.
		if n := sequenceFromID(clean.ID); n > r.sequence {
			r.sequence = n
		}
		for _, coverage := range clean.Coverages {
			if n := sequenceFromID(coverage.ID); n > r.sequence {
				r.sequence = n
			}
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

// List mengembalikan seluruh aturan TANPA coverage-nya, terurut seperti grid lama.
func (r *Repo) List(_ context.Context) ([]daftardetaildokumentravel.Detail, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := make([]daftardetaildokumentravel.Detail, 0, len(r.rows))
	for _, row := range r.rows {
		// Coverage sengaja dibuang di sini, meniru kueri daftar yang memang tidak
		// membacanya. Membiarkannya ikut akan membuat uji layar lulus di sini lalu gagal
		// terhadap Oracle, karena di sana daftarnya benar-benar kosong.
		row.Coverages = nil
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

// Get mengembalikan satu aturan lengkap dengan coverage-nya.
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
	return copyDetail(row), nil
}

// InsertNew menerbitkan ID lalu menyimpan barisnya beserta coverage-nya.
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

	row := detailFrom(id, input, r.nextID)
	r.rows[id] = row
	return copyDetail(row), nil
}

// Update mengganti isi satu aturan beserta SELURUH daftar coverage-nya.
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

	// Baris lama DIBUANG seluruhnya, termasuk coverage-nya, lalu disusun ulang dari
	// isian yang dikirim. Inilah yang ditiru adapter SQL dengan menghapus lalu menyisip
	// ulang baris coverage di dalam satu transaksi.
	row := detailFrom(key, input, r.nextID)
	r.rows[key] = row
	return copyDetail(row), nil
}

// nextID menerbitkan nomor berikutnya dari urutan yang sama dengan ID baris detail.
//
// Satu urutan untuk baris detail dan baris coverage, meniru adapter SQL yang memakai
// POOLDATA.LST_DOC_TRAVEL_SEQ untuk keduanya. Akibatnya deret ID baris detail berlubang
// — dan itu memang yang akan terlihat di Oracle, sehingga menirunya di sini membuat data
// pengembangan tidak menyesatkan.
//
// Pemanggilnya sudah memegang mutex; metode ini TIDAK mengambilnya sendiri.
func (r *Repo) nextID() string {
	r.sequence++
	return formatID(r.sequence)
}

// detailFrom menyusun baris tersimpan dari isian yang dikirim layar.
func detailFrom(
	id string,
	input daftardetaildokumentravel.Input,
	nextID func() string,
) daftardetaildokumentravel.Detail {
	row := daftardetaildokumentravel.Detail{
		ID:           id,
		DocumentID:   input.DocumentID,
		DocumentName: input.DocumentName,
		Mandatory:    input.Mandatory,
		MinUpload:    input.MinUpload,
	}
	for _, coverage := range input.Coverages {
		row.Coverages = append(row.Coverages, daftardetaildokumentravel.Coverage{
			ID:           nextID(),
			PlanID:       coverage.PlanID,
			PlanName:     coverage.PlanName,
			CoverageID:   coverage.CoverageID,
			CoverageName: coverage.CoverageName,
		})
	}
	return row
}

// cleanDetail memangkas isian baris yang diberikan sebagai isi awal.
func cleanDetail(row daftardetaildokumentravel.Detail) daftardetaildokumentravel.Detail {
	clean := daftardetaildokumentravel.Detail{
		ID:           strings.TrimSpace(row.ID),
		DocumentID:   strings.TrimSpace(row.DocumentID),
		DocumentName: strings.TrimSpace(row.DocumentName),
		Mandatory:    row.Mandatory,
		MinUpload:    row.MinUpload,
	}
	for _, coverage := range row.Coverages {
		clean.Coverages = append(clean.Coverages, daftardetaildokumentravel.Coverage{
			ID:           strings.TrimSpace(coverage.ID),
			PlanID:       strings.TrimSpace(coverage.PlanID),
			PlanName:     strings.TrimSpace(coverage.PlanName),
			CoverageID:   strings.TrimSpace(coverage.CoverageID),
			CoverageName: strings.TrimSpace(coverage.CoverageName),
		})
	}
	return clean
}

// copyDetail menyalin baris beserta senarai coverage-nya.
//
// Salinan senarainya WAJIB, bukan kehati-hatian berlebihan: tanpa itu pemanggil memegang
// senarai yang sama dengan yang tersimpan, dan mengubah satu elemennya akan mengubah isi
// repo tanpa melewati Update sama sekali. Adapter SQL tidak punya kelemahan itu karena
// ia selalu menyusun senarai baru dari hasil kueri.
func copyDetail(row daftardetaildokumentravel.Detail) daftardetaildokumentravel.Detail {
	clone := row
	if row.Coverages != nil {
		clone.Coverages = append([]daftardetaildokumentravel.Coverage(nil), row.Coverages...)
	}
	return clone
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

// PlanRepo menyimpan pilihan plan dan jaminan di memori.
type PlanRepo struct {
	mutex     sync.Mutex
	plans     []daftardetaildokumentravel.Plan
	coverages []daftardetaildokumentravel.CoverageOption
	failure   error
}

// NewPlanRepo membentuk pembaca master plan dan jaminan.
func NewPlanRepo(
	plans []daftardetaildokumentravel.Plan,
	coverages []daftardetaildokumentravel.CoverageOption,
) *PlanRepo {
	return &PlanRepo{
		plans:     append([]daftardetaildokumentravel.Plan(nil), plans...),
		coverages: append([]daftardetaildokumentravel.CoverageOption(nil), coverages...),
	}
}

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
//
// Jalur itu penting dan bukan sekadar kelengkapan: kegagalan membaca master plan TIDAK
// BOLEH menghalangi penyimpanan. Nama plan dan jaminan boleh diketik sendiri, sehingga
// yang hilang saat daftarnya gagal dimuat hanyalah kenyamanan memilih — perlakuan yang
// sama dengan daftar bisnis pada modul Master COL Simas Online.
func (r *PlanRepo) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// ListPlans mengembalikan plan yang dapat dipilih, terurut menurut namanya.
func (r *PlanRepo) ListPlans(_ context.Context) ([]daftardetaildokumentravel.Plan, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := append([]daftardetaildokumentravel.Plan(nil), r.plans...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListCoverages mengembalikan jaminan yang dapat dipilih beserta plan pemiliknya.
func (r *PlanRepo) ListCoverages(_ context.Context) ([]daftardetaildokumentravel.CoverageOption, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return nil, r.failure
	}

	result := append([]daftardetaildokumentravel.CoverageOption(nil), r.coverages...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].PlanID != result[j].PlanID {
			return result[i].PlanID < result[j].PlanID
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

var (
	_ daftardetaildokumentravel.Repo         = (*Repo)(nil)
	_ daftardetaildokumentravel.DocumentRepo = (*DocumentRepo)(nil)
	_ daftardetaildokumentravel.PlanRepo     = (*PlanRepo)(nil)
)
