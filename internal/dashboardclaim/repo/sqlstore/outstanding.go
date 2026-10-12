package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// CountOutstanding menghitung klaim berjalan seluruh organisasi pada entitas ini.
//
// Ia TIDAK menyaring menurut operator pemegang tugas, dan itu perbedaan yang menentukan
// terhadap `inboxoutstanding`: layar ini pandangan manajerial, bukan "My Inbox". Lihat
// catatan paket domain.
func (r *Repo) CountOutstanding(ctx context.Context, f dashboardclaim.Filter) (int, error) {
	f = f.Normalize().CountFilter()

	var total int
	err := r.db.QueryRowContext(ctx, query("outstanding_count"), outstandingFilterArgs(f)...).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("dashboardclaim/sqlstore: menghitung klaim berjalan: %w", err)
	}
	return total, nil
}

// ListOutstanding membaca satu halaman klaim berjalan beserta jumlah seluruh baris yang
// cocok.
//
// Kedua kueri dijalankan dari argumen penyaring YANG SAMA, bukan disusun dua kali: itulah
// yang membuat angka pada kartu dan jumlah baris pada telusurnya tidak dapat menyimpang.
func (r *Repo) ListOutstanding(ctx context.Context, f dashboardclaim.Filter) (dashboardclaim.ClaimPage, error) {
	f = f.Normalize()
	filters := outstandingFilterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("outstanding_count"), filters...).Scan(&total); err != nil {
		return dashboardclaim.ClaimPage{}, fmt.Errorf("dashboardclaim/sqlstore: menghitung klaim berjalan: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("outstanding_list"), withPaging(filters, f)...)
	if err != nil {
		return dashboardclaim.ClaimPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca daftar klaim berjalan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]dashboardclaim.ClaimRow, 0, f.Limit)
	for rows.Next() {
		row, err := scanClaimRow(rows)
		if err != nil {
			return dashboardclaim.ClaimPage{}, fmt.Errorf("dashboardclaim/sqlstore: membaca baris klaim: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		// Galat yang muncul SETELAH baris terakhir dibaca tetap kegagalan. Mengabaikannya
		// berarti mengembalikan halaman yang terpotong seolah-olah ia lengkap.
		return dashboardclaim.ClaimPage{}, fmt.Errorf("dashboardclaim/sqlstore: menutup daftar klaim berjalan: %w", err)
	}

	return dashboardclaim.ClaimPage{Rows: result, Total: total}, nil
}

// scanClaimRow membaca satu baris klaim.
//
// Urutan pemindaiannya WAJIB sama dengan urutan kolom pada `outstanding_list`. Keduanya
// dijaga berdampingan supaya penambahan kolom yang lupa diikuti pemindainya gagal saat uji,
// bukan menggeser seluruh nilai satu kolom tanpa galat apa pun.
func scanClaimRow(s scanner) (dashboardclaim.ClaimRow, error) {
	var (
		claimID          sql.NullString
		claimNumber      sql.NullString
		policyNumber     sql.NullString
		insuredName      sql.NullString
		businessName     sql.NullString
		businessSource   sql.NullString
		branchName       sql.NullString
		technicalPIC     sql.NullString
		adminPNC         sql.NullString
		claimStatusCode  sql.NullString
		claimStatusLabel sql.NullString
		processStatus    sql.NullString
		lossDate         sql.NullTime
		registeredAt     sql.NullTime

		// Tanggal lapor kini dibaca dari `T_CLAIM_PNC.RECEIVEDATE`, kolom bertipe DATE.
		// Dulu sumbernya `RECEIVEDDATE_1` objek kerja Pega — TEKS bergaya
		// `20200106T142602.000 GMT` — yang tidak ada di `T_CLAIMLIST_ADMIN`.
		reportDate sql.NullTime
	)

	if err := s.Scan(
		&claimID,
		&claimNumber,
		&policyNumber,
		&insuredName,
		&businessName,
		&businessSource,
		&branchName,
		&technicalPIC,
		&adminPNC,
		&claimStatusCode,
		&claimStatusLabel,
		&processStatus,
		&lossDate,
		&reportDate,
		&registeredAt,
	); err != nil {
		return dashboardclaim.ClaimRow{}, err
	}

	row := dashboardclaim.ClaimRow{
		ClaimID:          text(claimID),
		ClaimNumber:      text(claimNumber),
		PolicyNumber:     text(policyNumber),
		InsuredName:      text(insuredName),
		BusinessName:     text(businessName),
		BusinessSource:   text(businessSource),
		BranchName:       text(branchName),
		TechnicalPIC:     text(technicalPIC),
		AdminPNC:         text(adminPNC),
		ClaimStatusCode:  text(claimStatusCode),
		ClaimStatusLabel: text(claimStatusLabel),
		ProcessStatus:    text(processStatus),
		LossDate:         timeOrNil(lossDate),
		ReportDate:       timeOrNil(reportDate),
	}
	if registeredAt.Valid {
		row.RegisteredAt = registeredAt.Time
	}
	return row, nil
}

// timeOrNil mengubah kolom waktu yang boleh kosong menjadi penunjuk.
//
// Kosong dinyatakan sebagai nil, BUKAN sebagai waktu nol. Waktu nol akan tampil di layar
// sebagai tanggal 1 Januari tahun 1 — nilai yang tampak seperti data dan bukan seperti
// ketiadaan data.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	moment := value.Time
	return &moment
}
