// Package memory adalah pengisi seam komite.Repo yang hidup di dalam memori.
//
// Ia ada supaya mesin penjenjangan dan layar yang memakainya dapat diuji tanpa basis
// data — adapter kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Ambang Komite
// dan simulasi penjenjangan dapat dicoba lengkap dengan isi master yang benar sebelum
// akun aplikasi diberi hak baca ke POOLDATA.EMAILKOMITE.
package memory

import (
	"context"
	"sync"

	"claim-pnc/internal/komite"
)

// Repo menyimpan master ambang komite di memori.
//
// Dilindungi mutex karena satu instans dipakai bersama seluruh permintaan HTTP yang
// berjalan bersamaan.
type Repo struct {
	mu   sync.RWMutex
	rows []komite.Threshold
	err  error
}

// NewRepo membentuk repo berisi baris yang diberikan.
//
// Dipanggil tanpa argumen, ia KOSONG — bukan otomatis berisi contoh. Pengisian dengan
// SampleThresholds dilakukan pemanggil secara eksplisit, supaya pengujian yang ingin
// menguji master kosong tidak perlu melawan nilai bawaan yang tidak diminta.
func NewRepo(rows ...komite.Threshold) *Repo {
	copied := make([]komite.Threshold, 0, len(rows))
	for _, t := range rows {
		copied = append(copied, t.Normalized())
	}
	return &Repo{rows: copied}
}

// NewSampleRepo membentuk repo berisi master yang berlaku hari ini.
func NewSampleRepo() *Repo { return NewRepo(SampleThresholds()...) }

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// ListThresholds mengembalikan seluruh baris master apa adanya.
//
// Salinan dikembalikan, bukan senarai aslinya: pemanggil yang mengurutkan hasilnya tidak
// boleh diam-diam mengubah isi repo yang dipakai bersama permintaan lain.
func (r *Repo) ListThresholds(_ context.Context) ([]komite.Threshold, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.err != nil {
		return nil, r.err
	}
	return append([]komite.Threshold(nil), r.rows...), nil
}
