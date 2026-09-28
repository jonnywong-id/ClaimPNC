// Package memory memenuhi seam monitoringslinkojk.Repo tanpa basis data.
//
// Dua kegunaan, dan keduanya nyata:
//
//   - Menjalankan aplikasi tanpa Oracle saat pengembangan, sehingga kedua segmen layar
//     Monitoring SLINK OJK dapat dicoba sebelum DBA membuka akses ke tabel SLIK.
//   - Menguji aturan modul tanpa basis data, tanpa jaringan, dan tanpa berkas — itulah
//     yang membuat uji aturan bisnis cepat (`14-TESTING-STRATEGY.md` §3).
//
// # Seluruh isinya KARANGAN
//
// Tidak satu pun nilai di sample.go berasal dari data produksi. Nomor klaim, nomor
// kontrak, nomor rekening fasilitas, dan nomor CIF dibuat supaya seluruh jalur layar
// dapat ditempuh — termasuk yang paling mudah terlewat: baris berlini SURETY BOND yang
// hanya muncul ketika penyaringnya DIBALIK, dan satu klaim dengan DUA fasilitas kredit
// yang membuktikan kunci barisnya tidak tertukar.
//
// `D-69` melarang data nasabah ditulis ke berkas yang di-commit; berkas ini mematuhinya
// dengan tidak memakainya sama sekali.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
)

// Repo menyimpan data Monitoring SLINK OJK di memori.
//
// Aman dipakai bersamaan: seluruh pembacaan menempuh satu kunci baca. Meski tidak ada
// satu pun operasi yang menulis hari ini, kuncinya tetap ada — penyimpanan yang dibaca
// banyak permintaan HTTP sekaligus tanpa kunci adalah cacat yang hanya muncul di bawah
// beban, dan justru tidak terlihat saat diuji.
type Repo struct {
	mu sync.RWMutex

	// rows dikunci per segmen. Kedua segmen membaca tabel yang berbeda di produksi,
	// sehingga memisahkannya di sini menjaga penyimpanan memori berperilaku setara.
	rows map[monitoringslinkojk.Segment][]Record

	// reports adalah padanan POOLDATA.T_CLAIM_SLIK_OJK — baris laporan yang sudah
	// tersusun. Ia BERTAMBAH, tidak pernah ditimpa; lihat InsertReport.
	reports []monitoringslinkojk.ReportEntry

	// submissions adalah padanan POOLDATA.T_CLAIM_SLINK_INDIVIDU.
	submissions []recordedSubmission
}

// Record adalah satu baris contoh beserta kolom yang dipakai menyaringnya.
//
// Penyaring dipisahkan dari isi barisnya karena keduanya memang berbeda asalnya: di
// produksi, `REGISTERDATE_1` dan `BUSINESSTYPE` datang dari tabel KLAIM sedangkan isi
// barisnya datang dari tabel SLIK. Menyatukannya akan membuat penyimpanan memori
// menyaring hal yang tidak dapat disaring penyimpanan SQL.
type Record struct {
	// RegisteredAt adalah `REGISTERDATE_1` — tanggal registrasi klaim, yang disaring
	// kedua isian berlabel "Dari" dan "Sampai".
	RegisteredAt time.Time

	// BusinessType adalah `BUSINESSTYPE` pada tabel klaim.
	BusinessType string

	// Row adalah isi barisnya, sudah berkunci Column.Key.
	Row monitoringslinkojk.Row
}

// Option mengubah isi penyimpanan saat dibentuk.
type Option func(*Repo)

// NewRepo membentuk penyimpanan kosong yang diisi Option.
func NewRepo(options ...Option) *Repo {
	repo := &Repo{rows: map[monitoringslinkojk.Segment][]Record{}}
	for _, option := range options {
		option(repo)
	}
	return repo
}

// WithRows mengisi baris satu segmen.
func WithRows(segment monitoringslinkojk.Segment, records []Record) Option {
	return func(r *Repo) {
		r.rows[segment] = append([]Record(nil), records...)
	}
}

// Search mengembalikan satu halaman baris pada segmen yang diminta.
func (r *Repo) Search(
	ctx context.Context,
	segment monitoringslinkojk.Segment,
	filter monitoringslinkojk.Filter,
) (monitoringslinkojk.Page, error) {
	if err := ctx.Err(); err != nil {
		return monitoringslinkojk.Page{}, err
	}

	matched := r.matching(segment, filter)
	total := len(matched)

	offset := filter.Offset()
	if offset >= total {
		return monitoringslinkojk.Page{Rows: []monitoringslinkojk.Row{}, Total: total}, nil
	}
	end := offset + filter.Size
	if end > total {
		end = total
	}

	page := make([]monitoringslinkojk.Row, 0, end-offset)
	for _, record := range matched[offset:end] {
		page = append(page, cloneRow(record.Row))
	}
	return monitoringslinkojk.Page{Rows: page, Total: total}, nil
}

// Stream membaca seluruh baris yang cocok, satu per satu.
func (r *Repo) Stream(
	ctx context.Context,
	segment monitoringslinkojk.Segment,
	filter monitoringslinkojk.Filter,
	emit func(monitoringslinkojk.Row) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, record := range r.matching(segment, filter) {
		if err := emit(cloneRow(record.Row)); err != nil {
			return err
		}
	}
	return nil
}

// matching menyaring dan mengurutkan baris satu segmen.
//
// Urutannya DISAMAKAN dengan kueri SQL — tanggal registrasi menurun, lalu kunci baris —
// supaya uji yang lulus terhadap penyimpanan memori tidak menyembunyikan perbedaan
// urutan yang baru muncul di produksi.
func (r *Repo) matching(
	segment monitoringslinkojk.Segment,
	filter monitoringslinkojk.Filter,
) []Record {
	r.mu.RLock()
	defer r.mu.RUnlock()

	source := r.rows[segment]
	matched := make([]Record, 0, len(source))
	for _, record := range source {
		if !matchesDate(record, filter) {
			continue
		}
		if !matchesScope(record, filter.BusinessScope) {
			continue
		}
		matched = append(matched, record)
	}

	sort.SliceStable(matched, func(i, j int) bool {
		if !matched[i].RegisteredAt.Equal(matched[j].RegisteredAt) {
			return matched[i].RegisteredAt.After(matched[j].RegisteredAt)
		}
		return matched[i].Row.Get(monitoringslinkojk.RowKeyColumn) <
			matched[j].Row.Get(monitoringslinkojk.RowKeyColumn)
	})
	return matched
}

// matchesDate meniru batas tanggal kueri SQL, termasuk batas atas yang EKSKLUSIF.
//
// Kesetaraan itu bukan kerapian: bila di sini `<=` dipakai sementara SQL memakai `<`,
// uji akan lulus atas perilaku yang berbeda dari produksi tepat pada hari terakhir
// rentang — dan itu persis hari yang paling sering diperiksa pelapor.
func matchesDate(record Record, filter monitoringslinkojk.Filter) bool {
	if filter.DateOfLoss != nil {
		start := startOfDay(*filter.DateOfLoss)
		if record.RegisteredAt.Before(start) {
			return false
		}
	}
	if filter.DateOfRequestDocument != nil {
		limit := startOfDay(*filter.DateOfRequestDocument).AddDate(0, 0, 1)
		if !record.RegisteredAt.Before(limit) {
			return false
		}
	}
	return true
}

func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

// matchesScope meniru penyaring "Business Name", termasuk sifat MENIADAKAN pada
// "SURETY BOND". Lihat monitoringslinkojk.BusinessScope.
func matchesScope(record Record, scope monitoringslinkojk.BusinessScope) bool {
	credit := strings.TrimSpace(record.BusinessType) ==
		monitoringslinkojk.CreditInsuranceBusinessType

	switch scope {
	case monitoringslinkojk.ScopeCreditInsurance:
		return credit
	case monitoringslinkojk.ScopeSuretyBond:
		return !credit
	default:
		return true
	}
}

// cloneRow menyalin satu baris sebelum diserahkan pemanggil.
//
// Tanpa salinan, pemanggil yang mengubah satu sel akan mengubah isi penyimpanan bagi
// seluruh permintaan berikutnya — cacat yang tidak pernah terjadi pada penyimpanan SQL,
// sehingga uji terhadap memori akan menyembunyikannya alih-alih menemukannya.
func cloneRow(row monitoringslinkojk.Row) monitoringslinkojk.Row {
	clone := make(monitoringslinkojk.Row, len(row))
	for key, value := range row {
		clone[key] = value
	}
	return clone
}
