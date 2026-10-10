// Package memori adalah pengisi seam komite.Repo yang hidup di dalam memori.
//
// Ia ada supaya mesin penjenjangan dan layar yang memakainya dapat diuji tanpa basis
// data — adapter kedua yang membuat seam ini nyata, bukan hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
//
// Ia juga yang dipakai saat aplikasi berjalan tanpa Oracle, sehingga layar Ambang Komite
// dan simulasi penjenjangan dapat dicoba lengkap dengan isi master yang benar sebelum
// akun aplikasi diberi hak baca ke POOLDATA.EMAILKOMITE.
package memori

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
	mu    sync.RWMutex
	baris []komite.Ambang
	galat error
}

// RepoBaru membentuk repo berisi baris yang diberikan.
//
// Dipanggil tanpa argumen, ia KOSONG — bukan otomatis berisi contoh. Pengisian dengan
// AmbangContoh dilakukan pemanggil secara eksplisit, supaya pengujian yang ingin menguji
// master kosong tidak perlu melawan nilai bawaan yang tidak diminta.
func RepoBaru(baris ...komite.Ambang) *Repo {
	salinan := make([]komite.Ambang, 0, len(baris))
	for _, a := range baris {
		salinan = append(salinan, a.Bersih())
	}
	return &Repo{baris: salinan}
}

// RepoContoh membentuk repo berisi master yang berlaku hari ini.
func RepoContoh() *Repo { return RepoBaru(AmbangContoh()...) }

// SetGalat membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetGalat(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.galat = err
}

// DaftarAmbang mengembalikan seluruh baris master apa adanya.
//
// Salinan dikembalikan, bukan senarai aslinya: pemanggil yang mengurutkan hasilnya tidak
// boleh diam-diam mengubah isi repo yang dipakai bersama permintaan lain.
func (r *Repo) DaftarAmbang(_ context.Context) ([]komite.Ambang, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.galat != nil {
		return nil, r.galat
	}
	return append([]komite.Ambang(nil), r.baris...), nil
}
