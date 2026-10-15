// Package memory memenuhi seam outstandingclaim.Repo tanpa basis data.
//
// Dipakai pengembangan lokal dan pengujian. Ia bukan cache dan bukan tiruan sebagian: ia
// pengisi seam yang setara, sehingga aturan modul dapat diuji tanpa Oracle sama sekali —
// itulah gunanya seam punya lebih dari satu pengisi (`04-FUTURE-ARCHITECTURE.md` §3.1).
package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/outstandingclaim"
)

// Repo menyimpan rincian klaim di memori.
type Repo struct {
	// details dikunci NOMOR KLAIM dalam huruf besar.
	//
	// Huruf besar supaya pencocokannya tidak peka huruf: nomor klaim datang dari alamat
	// yang diketik atau disalin orang, dan `clmp-70` adalah klaim yang sama dengan
	// `CLMP-70`. Pengisi SQL membandingkannya apa adanya karena di sana nilainya datang
	// dari kolom, bukan dari alamat — perbedaan itu disengaja dan dicatat di sini supaya
	// tidak terbaca sebagai ketidakcocokan.
	details map[string]outstandingclaim.Detail
}

// NewRepo membentuk penyimpanan memori berisi baris yang diberikan.
func NewRepo(details []outstandingclaim.Detail) *Repo {
	indexed := make(map[string]outstandingclaim.Detail, len(details))
	for _, detail := range details {
		indexed[strings.ToUpper(detail.ClaimID)] = detail
	}
	return &Repo{details: indexed}
}

// NewSampleStore membentuk penyimpanan berisi rincian contoh.
//
// Ia dipakai cabang pengembangan tanpa Oracle. Namanya disamakan dengan penyimpanan memori
// modul lain supaya perakitnya di cmd/claimpnc terbaca seragam.
func NewSampleStore() *Repo {
	return NewRepo(SampleDetails())
}

// Find mengambil satu rincian klaim.
func (r *Repo) Find(
	_ context.Context,
	q outstandingclaim.Query,
) (outstandingclaim.Detail, error) {
	detail, found := r.details[strings.ToUpper(strings.TrimSpace(q.ClaimID))]
	if !found {
		return outstandingclaim.Detail{}, outstandingclaim.ErrNotFound
	}
	return detail, nil
}
