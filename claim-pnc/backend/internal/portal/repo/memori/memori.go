// Package memori adalah pengisi seam portal.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul portal dan layar yang memakainya dapat diuji tanpa basis data —
// adapter kedua yang membuat seam ini nyata, bukan hipotetis.
package memori

import (
	"context"

	"claim-pnc/internal/portal"
)

// Repo menyimpan daftar portal di memori.
type Repo struct {
	daftar []portal.Portal
	galat  error
}

// RepoBaru membentuk repo berisi daftar yang diberikan.
func RepoBaru(daftar ...portal.Portal) *Repo { return &Repo{daftar: daftar} }

// SetGalat membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetGalat(err error) { r.galat = err }

// Daftar mengembalikan portal yang tersimpan.
func (r *Repo) Daftar(_ context.Context) ([]portal.Portal, error) {
	if r.galat != nil {
		return nil, r.galat
	}
	return r.daftar, nil
}

// DaftarContoh adalah keenam portal nyata sesuai isi POOLDATA.M_PORTAL_PNC, dipakai
// pengujian dan pengembangan tanpa basis data.
func DaftarContoh() []portal.Portal {
	return []portal.Portal{
		{ID: "202600101", Nama: "ASURANSI SINAR MAS", Alias: "ASM"},
		{ID: "202600102", Nama: "ASURANSI SIMAS INSURTECH", Alias: "ASI"},
		{ID: "202600103", Nama: "SINARMAS ASURANSI SYARIAH", Alias: "SMAS"},
		{ID: "202600104", Nama: "SINARMAS INSURANCE", Alias: "SMI"},
		{ID: "202600105", Nama: "SINARMAS PENJAMINAN KREDIT", Alias: "SPK"},
		{ID: "202600106", Nama: "SINARMAS PENJAMINAN KREDIT SYARIAH", Alias: "SPKS"},
	}
}

var _ portal.Repo = (*Repo)(nil)
