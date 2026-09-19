// Package memory adalah pengisi seam portal.Repo yang hidup di dalam memori.
//
// Ia ada supaya modul portal dan layar yang memakainya dapat diuji tanpa basis data —
// adapter kedua yang membuat seam ini nyata, bukan hipotetis.
package memory

import (
	"context"

	"claim-pnc/internal/portal"
)

// Repo menyimpan daftar portal di memori.
type Repo struct {
	list    []portal.Portal
	failure error
}

// NewRepo membentuk repo berisi daftar yang diberikan.
func NewRepo(list ...portal.Portal) *Repo { return &Repo{list: list} }

// SetError membuat repo menjawab dengan galat, untuk menguji jalur gagal.
func (r *Repo) SetError(err error) { r.failure = err }

// List mengembalikan portal yang tersimpan.
func (r *Repo) List(_ context.Context) ([]portal.Portal, error) {
	if r.failure != nil {
		return nil, r.failure
	}
	return r.list, nil
}

// SampleList adalah keenam portal nyata sesuai isi POOLDATA.M_PORTAL_PNC, dipakai
// pengujian dan pengembangan tanpa basis data.
func SampleList() []portal.Portal {
	return []portal.Portal{
		{ID: "202600101", Name: "ASURANSI SINAR MAS", Alias: "ASM"},
		{ID: "202600102", Name: "ASURANSI SIMAS INSURTECH", Alias: "ASI"},
		{ID: "202600103", Name: "SINARMAS ASURANSI SYARIAH", Alias: "SMAS"},
		{ID: "202600104", Name: "SINARMAS INSURANCE", Alias: "SMI"},
		{ID: "202600105", Name: "SINARMAS PENJAMINAN KREDIT", Alias: "SPK"},
		{ID: "202600106", Name: "SINARMAS PENJAMINAN KREDIT SYARIAH", Alias: "SPKS"},
	}
}

var _ portal.Repo = (*Repo)(nil)
