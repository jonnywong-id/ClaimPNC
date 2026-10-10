package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// FindSurveyResults membaca hasil investigasi satu klaim dari
// `POOLDATA.T_SURVEYORLIST`.
//
// # Tabel ini TIDAK diperiksa `-periksa`, dan itu disengaja
//
// Berbeda dengan tabel keputusan, riwayat, dan penugasan, tabel ini **hanya dibaca** dan
// sudah dibaca modul `inboxinvestigator` lebih dulu. Ia bukan tabel baru yang menunggu
// DBA, melainkan tabel produksi yang sudah terisi — sehingga pemeriksa start tidak
// menambah kepastian apa pun.
//
// Yang penting justru sebaliknya: kegagalan membacanya **tidak boleh** menutup form.
// Lihat usecase OpenChecker.
func (r *Repo) FindSurveyResults(
	ctx context.Context, reference string,
) ([]inboxcompliance.SurveyResult, error) {
	rows, err := r.db.QueryContext(ctx, query("find_survey_results"), reference)
	if err != nil {
		return nil, fmt.Errorf(
			"membaca hasil investigasi dari POOLDATA.T_SURVEYORLIST: %w", err)
	}
	defer rows.Close()

	// Senarai kosong, bukan nil: "belum pernah disurvei" adalah keadaan normal, dan
	// nil memaksa setiap pemanggil mengingat perbedaan yang tidak berarti apa-apa.
	results := []inboxcompliance.SurveyResult{}

	for rows.Next() {
		var (
			surveyedAt     sql.NullTime
			objectName     sql.NullString
			objectLocation sql.NullString
			status         sql.NullString
		)
		if err := rows.Scan(
			&surveyedAt, &objectName, &objectLocation, &status,
		); err != nil {
			return nil, fmt.Errorf("membaca baris hasil investigasi: %w", err)
		}

		result := inboxcompliance.SurveyResult{
			ObjectName:     objectName.String,
			ObjectLocation: objectLocation.String,
			Status:         status.String,
		}
		if surveyedAt.Valid {
			at := surveyedAt.Time
			result.SurveyedAt = &at
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil investigasi: %w", err)
	}
	return results, nil
}
