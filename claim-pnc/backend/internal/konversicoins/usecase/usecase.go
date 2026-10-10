// Package usecase menjalankan Konversi Coins satu daftar polis: membaca dokumen polis LIVE,
// mengurai CoinsList, lalu menulis T_COINSLIST TEST — satu transaksi per polis.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	kc "claim-pnc/internal/konversicoins"
)

// MaxPolicies membatasi jumlah polis satu kali jalan. Konversi berjalan di dalam satu
// permintaan HTTP; daftar yang jauh lebih panjang lebih aman dipecah.
const MaxPolicies = 500

// Source membaca basis data LIVE. Pengisinya WAJIB hanya membaca.
type Source interface {
	// Documents mengembalikan satu dokumen polis per versi (PRODKE): baris JSON_POLIS
	// ber-DATA_JSONBLOB dengan TGL_INPUT terbaru.
	Documents(ctx context.Context, policyNo string) ([]kc.PolicyDoc, error)
}

// Batch adalah seluruh tulisan untuk satu polis.
type Batch struct {
	PolicyNo string
	Versions []string // PRODKE yang barisnya diganti di TEST
	Rows     []kc.Row
}

// Applied adalah hasil penulisan satu polis.
type Applied struct {
	Deleted  int
	Inserted int
	Warnings []string
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

	// TanpaCoins adalah jumlah versi yang dokumennya tidak membawa CoinsList.
	TanpaCoins   int `json:"versi_tanpa_coins"`
	BarisDihapus int `json:"baris_dihapus"`
	BarisDitulis int `json:"baris_ditulis"`

	Peringatan []string `json:"peringatan,omitempty"`
	Galat      string   `json:"galat,omitempty"`
}

// Report adalah laporan satu kali jalan.
type Report struct {
	UjiCoba    bool           `json:"uji_coba"`
	Mulai      time.Time      `json:"mulai"`
	Selesai    time.Time      `json:"selesai"`
	Polis      []PolicyResult `json:"polis"`
	Berhasil   int            `json:"berhasil"`
	TidakAda   int            `json:"tidak_ditemukan"`
	Gagal      int            `json:"gagal"`
	TotalBaris int            `json:"total_baris"`
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
			report.TotalBaris += result.BarisDitulis
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

	docs, err := s.source.Documents(ctx, policyNo)
	if err != nil {
		result.Status = "gagal"
		result.Galat = fmt.Sprintf("membaca LIVE %s: %v", kc.SourceTable, err)
		return result
	}
	if len(docs) == 0 {
		result.Status = "tidak_ditemukan"
		result.Galat = fmt.Sprintf("nomor polis tidak ada di %s LIVE, atau %s-nya kosong", kc.SourceTable, kc.BlobColumn)
		return result
	}

	batch := Batch{PolicyNo: policyNo}
	for _, d := range docs {
		rows, warnings, err := kc.Convert(d)
		if err != nil {
			result.Status = "gagal"
			result.Galat = fmt.Sprintf("PRODKE %s: %v", d.ProdKe, err)
			return result
		}
		if len(rows) == 0 {
			result.TanpaCoins++
		}
		batch.Versions = append(batch.Versions, d.ProdKe)
		batch.Rows = append(batch.Rows, rows...)
		result.Peringatan = append(result.Peringatan, warnings...)
	}
	result.Versi = batch.Versions

	applied, err := s.target.Apply(ctx, batch, commit)
	result.Peringatan = append(result.Peringatan, applied.Warnings...)
	if err != nil {
		result.Status = "gagal"
		result.Galat = "menulis TEST: " + err.Error()
		return result
	}
	result.Status = "berhasil"
	result.BarisDihapus = applied.Deleted
	result.BarisDitulis = applied.Inserted
	return result
}
