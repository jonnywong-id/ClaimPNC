package memory

import (
	"context"
	"strings"

	"claim-pnc/internal/dashboardclaim"
)

// FindClaimDetail membaca rincian satu klaim dari simpanan memori.
//
// Adapter KEDUA di balik seam `ClaimDetailReader`. Nilainya nyata di seam ini: sisi SQL-nya
// membaca `POOLDATA.JSON_KLAIM`, tabel yang tidak ada di lingkungan pengembangan, sehingga
// tanpa adapter ini popup rincian tidak dapat dibuka sama sekali tanpa Oracle.
//
// # Dokumennya SENGAJA kosong
//
// Simpanan memori memuat baris daftar, bukan dokumen JSON klaim — dan bentuk dokumen itu
// belum pernah diperiksa (`R-08`). Mengarang isinya akan membuat layar tampak benar saat
// dicoba tanpa Oracle lalu berperilaku lain di Oracle: persis kelas kekeliruan yang sudah
// dua kali ditemukan di modul ini.
//
// Yang dikembalikan karena itu ketujuh nilai yang MEMANG dimiliki baris daftar, ditambah
// dokumen kosong — keadaan yang sah dan sudah ditangani layar.
func (r *Repo) FindClaimDetail(
	ctx context.Context,
	claimID string,
) (dashboardclaim.ClaimDetail, error) {
	_ = ctx

	key := strings.TrimSpace(claimID)
	if key == "" {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.claims {
		row := r.claims[i].Row
		if row.ClaimID != key && row.ClaimNumber != key {
			continue
		}

		return dashboardclaim.ClaimDetail{
			ClaimID:       row.ClaimID,
			ClaimNumber:   row.ClaimNumber,
			ProcessStatus: row.ProcessStatus,
			ClaimStatus:   row.ClaimStatusCode,
			TechnicalPIC:  row.TechnicalPIC,
			AdminPNC:      row.AdminPNC,
			RegisteredAt:  row.RegisteredAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			Document:      map[string]any{},
		}, nil
	}

	return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
}
