package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// FindRejectPrefill membaca isian pra-isi form Surat Penolakan.
//
// Klaim yang belum punya baris di tabel datar `T_CLAIM_PNC` mengembalikan isian KOSONG
// tanpa galat. Itu keadaan nyata dan bukan kelainan: tabel itu diisi prosedur konversi,
// dan kueri daftar modul ini pun meng-`LEFT JOIN` ke sana justru karena barisnya dapat
// belum ada. Form tetap dapat dibuka; isiannya saja yang diketik sendiri.
func (r *Repo) FindRejectPrefill(
	ctx context.Context, reference string,
) (inboxcompliance.RejectPrefill, error) {
	var (
		dateOfLoss sql.NullTime
		location   sql.NullString
		patient    sql.NullString
	)

	err := r.db.QueryRowContext(
		ctx, query("find_reject_prefill"), reference,
	).Scan(&dateOfLoss, &location, &patient)

	if errors.Is(err, sql.ErrNoRows) {
		return inboxcompliance.RejectPrefill{}, nil
	}
	if err != nil {
		return inboxcompliance.RejectPrefill{},
			fmt.Errorf("membaca pra-isi Surat Penolakan: %w", err)
	}

	prefill := inboxcompliance.RejectPrefill{
		IncidentPlace: location.String,
		PatientName:   patient.String,
	}
	if dateOfLoss.Valid {
		at := dateOfLoss.Time
		prefill.IncidentDate = &at
	}
	return prefill, nil
}
