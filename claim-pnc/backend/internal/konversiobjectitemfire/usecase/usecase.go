// Package usecase menjalankan Konversi Object Item Fire satu daftar polis: membaca LIVE,
// mengurai PropertyItemList, lalu menulis TEST — satu transaksi per polis.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	ki "claim-pnc/internal/konversiobjectitemfire"
)

// MaxPolicies membatasi jumlah polis satu kali jalan. Konversi berjalan di dalam satu
// permintaan HTTP; daftar yang jauh lebih panjang lebih aman dipecah.
const MaxPolicies = 500

// Source membaca basis data LIVE. Pengisinya WAJIB hanya membaca.
type Source interface {
	// Objects mengembalikan baris T_PROPERTYLIST satu polis, seluruh versinya (PRODKE).
	Objects(ctx context.Context, policyNo string) ([]ki.ObjectRow, error)
}

// Batch adalah seluruh tulisan untuk satu polis.
type Batch struct {
	PolicyNo string
	Objects  []ki.ObjectRow // kolom BLOB-nya disalin ke TEST
	Items    []ki.Row
}

// Applied adalah hasil penulisan satu polis.
type Applied struct {
	ObjectsUpdated int
	ObjectsMissing []string // "INDEXOBJECT 3 PRODKE 0" yang tidak ada di TEST
	ItemsDeleted   int
	ItemsInserted  int
	Warnings       []string
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
	Versi    []string `json:"versi_prodke"`

	Objek           int      `json:"jumlah_objek"`
	ObjekDiperbarui int      `json:"objek_diperbarui"`
	ObjekTidakAda   []string `json:"objek_tidak_ada_di_test,omitempty"`

	ItemDihapus   int `json:"item_dihapus"`
	ItemDitulis   int `json:"item_ditulis"`
	DokumenKosong int `json:"dokumen_kosong"`

	Peringatan []string `json:"peringatan,omitempty"`
	Galat      string   `json:"galat,omitempty"`
}

// Report adalah laporan satu kali jalan.
type Report struct {
	UjiCoba   bool           `json:"uji_coba"`
	Mulai     time.Time      `json:"mulai"`
	Selesai   time.Time      `json:"selesai"`
	Polis     []PolicyResult `json:"polis"`
	Berhasil  int            `json:"berhasil"`
	TidakAda  int            `json:"tidak_ditemukan"`
	Gagal     int            `json:"gagal"`
	TotalItem int            `json:"total_item"`
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
// Satu polis yang gagal tidak menghentikan polis lain; transaksinya sendiri yang dibatalkan.
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
			report.TotalItem += result.ItemDitulis
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

	objects, err := s.source.Objects(ctx, policyNo)
	if err != nil {
		result.Status = "gagal"
		result.Galat = fmt.Sprintf("membaca LIVE %s: %v", ki.ObjectTable, err)
		return result
	}
	if len(objects) == 0 {
		result.Status = "tidak_ditemukan"
		result.Galat = "nomor polis tidak ada di " + ki.ObjectTable + " LIVE"
		return result
	}

	batch := Batch{PolicyNo: policyNo, Objects: objects}
	versions := map[string]bool{}
	for _, object := range objects {
		versions[object.ProdKe] = true
		if len(object.Document) == 0 {
			result.DokumenKosong++
			continue
		}
		rows, err := ki.Convert(object)
		if err != nil {
			result.Status = "gagal"
			result.Galat = fmt.Sprintf("INDEXOBJECT %s PRODKE %s: %v", object.IndexObject, object.ProdKe, err)
			return result
		}
		batch.Items = append(batch.Items, rows...)
	}
	for v := range versions {
		result.Versi = append(result.Versi, v)
	}
	sort.Strings(result.Versi)
	result.Objek = len(objects)

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
	result.ItemDihapus = applied.ItemsDeleted
	result.ItemDitulis = applied.ItemsInserted
	return result
}
