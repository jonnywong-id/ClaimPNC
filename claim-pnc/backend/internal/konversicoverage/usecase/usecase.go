// Package usecase menjalankan konversi coverage satu daftar polis: membaca LIVE,
// mengurai dokumen, lalu menulis TEST — satu transaksi per polis.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	kc "claim-pnc/internal/konversicoverage"
)

// MaxPolicies membatasi jumlah polis satu kali jalan. Konversi berjalan di dalam satu
// permintaan HTTP; daftar yang jauh lebih panjang lebih aman dipecah.
const MaxPolicies = 500

// Source membaca basis data LIVE. Pengisinya WAJIB hanya membaca.
type Source interface {
	// Objects mengembalikan baris objek satu polis di satu lini, seluruh versinya (PRODKE).
	Objects(ctx context.Context, line kc.Line, policyNo string) ([]kc.ObjectRow, error)
}

// Batch adalah seluruh tulisan untuk satu polis.
type Batch struct {
	PolicyNo string
	Lines    []LineBatch
	// Spreading seluruh lini satu polis — satu tabel tujuan untuk keempatnya.
	Spreading []kc.Row
}

// LineBatch adalah tulisan satu lini.
type LineBatch struct {
	Line     kc.Line
	Objects  []kc.ObjectRow // kolom BLOB-nya disalin ke TEST
	Coverage []kc.Row
}

// Applied adalah hasil penulisan satu polis.
type Applied struct {
	ObjectsUpdated   int
	ObjectsMissing   []string // "T_ANEKALIST INDEXOBJECT 3 PRODKE 0" yang tidak ada di TEST
	CoverageDeleted  int
	CoverageInserted int
	SpreadDeleted    int
	SpreadInserted   int
	Warnings         []string
}

// Target menulis basis data TEST.
type Target interface {
	// Apply menulis satu polis di dalam SATU transaksi. commit=false berarti seluruhnya
	// dijalankan lalu dibatalkan (uji coba) — galat bentuk data tetap tertangkap.
	Apply(ctx context.Context, batch Batch, commit bool) (Applied, error)
}

// PolicyResult adalah laporan satu polis.
type PolicyResult struct {
	PolicyNo string   `json:"nomor_polis"`
	Status   string   `json:"status"` // berhasil | tidak_ditemukan | gagal
	Lini     []string `json:"lini"`
	Versi    []string `json:"versi_prodke"`

	Objek           int      `json:"jumlah_objek"`
	ObjekDiperbarui int      `json:"objek_diperbarui"`
	ObjekTidakAda   []string `json:"objek_tidak_ada_di_test,omitempty"`

	CoverageDihapus  int `json:"coverage_dihapus"`
	CoverageDitulis  int `json:"coverage_ditulis"`
	SpreadingDihapus int `json:"spreading_dihapus"`
	SpreadingDitulis int `json:"spreading_ditulis"`
	DokumenKosong    int `json:"dokumen_kosong"`

	Peringatan []string `json:"peringatan,omitempty"`
	Galat      string   `json:"galat,omitempty"`
}

// Report adalah laporan satu kali jalan.
type Report struct {
	UjiCoba     bool           `json:"uji_coba"`
	Mulai       time.Time      `json:"mulai"`
	Selesai     time.Time      `json:"selesai"`
	Polis       []PolicyResult `json:"polis"`
	Berhasil    int            `json:"berhasil"`
	TidakAda    int            `json:"tidak_ditemukan"`
	Gagal       int            `json:"gagal"`
	TotalCover  int            `json:"total_coverage"`
	TotalSpread int            `json:"total_spreading"`
}

// ErrBusy berarti konversi lain sedang berjalan.
var ErrBusy = errors.New("konversi lain sedang berjalan; tunggu sampai selesai")

// ErrEmptyList berarti tidak ada nomor polis yang diberikan.
var ErrEmptyList = errors.New("daftar nomor polis kosong")

// ErrTooMany berarti daftar melebihi MaxPolicies.
var ErrTooMany = fmt.Errorf("daftar nomor polis melebihi %d; pecah menjadi beberapa kali jalan", MaxPolicies)

// Service menjalankan konversi.
type Service struct {
	source Source
	target Target
	now    func() time.Time

	running sync.Mutex
}

// New menyusun Service.
func New(source Source, target Target, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{source: source, target: target, now: now}
}

// Run mengonversi daftar polis. commit=false menjalankan uji coba: seluruh tulisan
// dijalankan di TEST lalu dibatalkan, sehingga angka laporannya sama dengan yang akan
// terjadi.
//
// Satu polis yang gagal tidak menghentikan polis lain; transaksinya sendiri yang
// dibatalkan.
func (s *Service) Run(ctx context.Context, policies []string, commit bool) (Report, error) {
	if len(policies) == 0 {
		return Report{}, ErrEmptyList
	}
	if len(policies) > MaxPolicies {
		return Report{}, ErrTooMany
	}
	if !s.running.TryLock() {
		return Report{}, ErrBusy
	}
	defer s.running.Unlock()

	report := Report{UjiCoba: !commit, Mulai: s.now()}
	for _, policyNo := range policies {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		result := s.one(ctx, policyNo, commit)
		switch result.Status {
		case "berhasil":
			report.Berhasil++
			report.TotalCover += result.CoverageDitulis
			report.TotalSpread += result.SpreadingDitulis
		case "tidak_ditemukan":
			report.TidakAda++
		default:
			report.Gagal++
		}
		report.Polis = append(report.Polis, result)
	}
	report.Selesai = s.now()
	return report, nil
}

func (s *Service) one(ctx context.Context, policyNo string, commit bool) PolicyResult {
	result := PolicyResult{PolicyNo: policyNo}
	batch := Batch{PolicyNo: policyNo}
	versions := map[string]bool{}

	for _, line := range kc.Lines {
		objects, err := s.source.Objects(ctx, line, policyNo)
		if err != nil {
			result.Status = "gagal"
			result.Galat = fmt.Sprintf("membaca LIVE %s: %v", line.ObjectTable, err)
			return result
		}
		if len(objects) == 0 {
			continue
		}
		lb := LineBatch{Line: line, Objects: objects}
		for _, object := range objects {
			versions[object.ProdKe] = true
			if len(object.Document) == 0 {
				result.DokumenKosong++
				continue
			}
			converted, err := kc.Convert(object, line)
			if err != nil {
				result.Status = "gagal"
				result.Galat = fmt.Sprintf("%s INDEXOBJECT %s PRODKE %s: %v",
					line.ObjectTable, object.IndexObject, object.ProdKe, err)
				return result
			}
			lb.Coverage = append(lb.Coverage, converted.Coverage...)
			batch.Spreading = append(batch.Spreading, converted.Spreading...)
		}
		result.Lini = append(result.Lini, line.ObjectTable)
		result.Objek += len(objects)
		batch.Lines = append(batch.Lines, lb)
	}
	for v := range versions {
		result.Versi = append(result.Versi, v)
	}
	sort.Strings(result.Versi)

	if len(batch.Lines) == 0 {
		result.Status = "tidak_ditemukan"
		result.Galat = "nomor polis tidak ada di keempat tabel objek LIVE"
		return result
	}

	applied, err := s.target.Apply(ctx, batch, commit)
	result.Peringatan = applied.Warnings
	if err != nil {
		result.Status = "gagal"
		result.Galat = "menulis TEST: " + err.Error()
		return result
	}
	result.Status = "berhasil"
	result.ObjekDiperbarui = applied.ObjectsUpdated
	result.ObjekTidakAda = applied.ObjectsMissing
	result.CoverageDihapus = applied.CoverageDeleted
	result.CoverageDitulis = applied.CoverageInserted
	result.SpreadingDihapus = applied.SpreadDeleted
	result.SpreadingDitulis = applied.SpreadInserted
	return result
}
