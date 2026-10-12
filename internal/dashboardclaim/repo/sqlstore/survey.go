package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/dashboardclaim"
)

// queryNamesFor memilih pasangan kueri menurut jenis surveyor.
//
// Keduanya TIDAK dapat disatukan menjadi satu kueri berparameter, dan itu bukan pilihan
// gaya: kueri lamanya benar-benar berbeda syarat — penyaring lini bisnisnya menyaring tabel
// yang berbeda, dan status klaim induk hanya diperiksa di salah satunya. Lihat catatan di
// survey.sql.
//
// Jenis yang tidak dikenal menghasilkan galat, BUKAN kueri bawaan: menjalankan kueri
// surveyor internal untuk loss adjuster akan mengembalikan baris yang tampak sah dan
// seluruhnya salah.
func queryNamesFor(kind dashboardclaim.SurveyorType) (count string, list string, err error) {
	switch kind {
	case dashboardclaim.SurveyorAdjuster:
		return "loss_adjuster_count", "loss_adjuster_list", nil
	case dashboardclaim.SurveyorInternal:
		return "internal_surveyor_count", "internal_surveyor_list", nil
	default:
		return "", "", fmt.Errorf("dashboardclaim/sqlstore: jenis surveyor %q tidak dikenal", kind)
	}
}

// CountSurvey menghitung baris survei menurut jenis surveyornya.
//
// Kedua jenis menghitung BARIS SURVEI, sehingga angkanya sejalan dengan isi telusurnya.
// Sistem lama tidak demikian pada loss adjuster — ia menghitung klaim. Selisih itu
// terencana dan menunggu persetujuan Work Owner (`D-54`); lihat survey.sql.
func (r *Repo) CountSurvey(ctx context.Context, kind dashboardclaim.SurveyorType, f dashboardclaim.Filter) (int, error) {
	countQuery, _, err := queryNamesFor(kind)
	if err != nil {
		return 0, err
	}
	f = f.Normalize().CountFilter()

	var total int
	if err := r.db.QueryRowContext(ctx, query(countQuery), filterArgs(f)...).Scan(&total); err != nil {
		return 0, fmt.Errorf("dashboardclaim/sqlstore: menghitung survei %q: %w", kind, err)
	}
	return total, nil
}

// ListSurvey membaca satu halaman survei beserta jumlah seluruh baris yang cocok.
func (r *Repo) ListSurvey(ctx context.Context, kind dashboardclaim.SurveyorType, f dashboardclaim.Filter) (dashboardclaim.SurveyPage, error) {
	countQuery, listQuery, err := queryNamesFor(kind)
	if err != nil {
		return dashboardclaim.SurveyPage{}, err
	}

	f = f.Normalize()
	filters := filterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query(countQuery), filters...).Scan(&total); err != nil {
		return dashboardclaim.SurveyPage{}, fmt.Errorf("dashboardclaim/sqlstore: menghitung survei %q: %w", kind, err)
	}

	rows, err := r.db.QueryContext(ctx, query(listQuery), withPaging(filters, f)...)
	if err != nil {
		return dashboardclaim.SurveyPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca daftar survei %q: %w", kind, err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]dashboardclaim.SurveyRow, 0, f.Limit)
	for rows.Next() {
		row, err := scanSurveyRow(rows)
		if err != nil {
			return dashboardclaim.SurveyPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca baris survei: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return dashboardclaim.SurveyPage{}, fmt.Errorf("dashboardclaim/sqlstore: menutup daftar survei %q: %w", kind, err)
	}

	return dashboardclaim.SurveyPage{Rows: result, Total: total}, nil
}

// scanSurveyRow membaca satu baris survei.
//
// Urutan pemindaiannya WAJIB sama dengan urutan kolom pada kedua kueri daftar survei, dan
// kedua kueri itu sengaja menyebut kolom yang sama dalam urutan yang sama supaya satu
// pemindai melayani keduanya. Perbedaan isinya dinyatakan di sisi SQL — loss adjuster
// mengembalikan NULL untuk TANGGAL_SURVEI karena kueri lamanya memang tidak mengambilnya.
func scanSurveyRow(s scanner) (dashboardclaim.SurveyRow, error) {
	var (
		surveyID        sql.NullString
		surveyNumber    sql.NullString
		claimNumber     sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		referenceNumber sql.NullString
		surveyorName    sql.NullString
		technicalPIC    sql.NullString
		surveyLocation  sql.NullString
		surveyStatus    sql.NullString
		processStatus   sql.NullString
		adjusterPIC     sql.NullString
		scheduledAt     sql.NullTime
		assignedAt      sql.NullTime
	)

	if err := s.Scan(
		&surveyID,
		&surveyNumber,
		&claimNumber,
		&policyNumber,
		&insuredName,
		&referenceNumber,
		&surveyorName,
		&technicalPIC,
		&surveyLocation,
		&surveyStatus,
		&processStatus,
		&adjusterPIC,
		&scheduledAt,
		&assignedAt,
	); err != nil {
		return dashboardclaim.SurveyRow{}, err
	}

	row := dashboardclaim.SurveyRow{
		SurveyID:        text(surveyID),
		SurveyNumber:    text(surveyNumber),
		ClaimNumber:     text(claimNumber),
		PolicyNumber:    text(policyNumber),
		InsuredName:     text(insuredName),
		ReferenceNumber: text(referenceNumber),
		SurveyorName:    text(surveyorName),
		TechnicalPIC:    text(technicalPIC),
		SurveyLocation:  text(surveyLocation),
		SurveyStatus:    text(surveyStatus),
		ProcessStatus:   text(processStatus),
		AdjusterPIC:     text(adjusterPIC),
		ScheduledAt:     timeOrNil(scheduledAt),
	}
	if assignedAt.Valid {
		row.AssignedAt = assignedAt.Time
	}
	return row, nil
}
