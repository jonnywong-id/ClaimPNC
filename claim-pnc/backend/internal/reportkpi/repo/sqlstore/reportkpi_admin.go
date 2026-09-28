package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/reportkpi"
)

// Berkas ini memenuhi bagian tab **KPI Admin** dari seam reportkpi.Repo.
//
// Ia terpisah dari reportkpi.go karena bentuk datanya memang berbeda: tab Adjuster membaca
// SATU tabel datar berisi nilai yang sudah jadi, sedangkan tab ini menghitung kartu skor
// dari tiga tabel sekaligus dan membedakan dua kelompok yang kolomnya tidak sama.

// AdminTotals mengambil angka mentah kartu skor.
//
// Percabangan kelompoknya ada DI SINI, bukan di dalam satu kueri bersyarat: kedua kueri
// membaca tabel yang berbeda, menyaring dengan operator yang berbeda, dan menghitung
// metrik yang berbeda. Menyatukannya menjadi satu kueri ber-`CASE` hanya akan membuat
// keduanya sulit dibandingkan dengan rule Pega-nya masing-masing.
func (r *Repo) AdminTotals(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	if q.Group == reportkpi.AdminGroupPA {
		return r.adminTotalsPA(ctx, q)
	}
	return r.adminTotalsNonMBU(ctx, q)
}

// adminTotalsNonMBU mengambil kartu skor NON-MBU.
func (r *Repo) adminTotalsNonMBU(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	var (
		leaderOver, leaderTotal, leaderPercent sql.NullFloat64
		memberOver, memberTotal, memberPercent sql.NullFloat64
		leaderScore, memberScore               sql.NullFloat64
		leaderSubtotal, memberSubtotal         sql.NullFloat64
		quantitativeTotal, achievementRatio    sql.NullFloat64
	)

	err := r.db.QueryRowContext(ctx, query("admin_scorecard_nonmbu"),
		q.Range.From, q.Range.To,
	).Scan(
		&leaderOver, &leaderTotal, &leaderPercent,
		&memberOver, &memberTotal, &memberPercent,
		&leaderScore, &memberScore,
		&leaderSubtotal, &memberSubtotal,
		&quantitativeTotal, &achievementRatio,
	)
	if err != nil {
		return reportkpi.AdminTotals{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca kartu skor NON-MBU: %w", err)
	}

	return reportkpi.AdminTotals{
		LeaderOverSLA:     score(leaderOver),
		LeaderTotal:       score(leaderTotal),
		LeaderPercent:     score(leaderPercent),
		LeaderScore:       score(leaderScore),
		LeaderSubtotal:    score(leaderSubtotal),
		MemberOverSLA:     score(memberOver),
		MemberTotal:       score(memberTotal),
		MemberPercent:     score(memberPercent),
		MemberScore:       score(memberScore),
		MemberSubtotal:    score(memberSubtotal),
		QuantitativeTotal: score(quantitativeTotal),
		AchievementRatio:  score(achievementRatio),
	}, nil
}

// adminTotalsPA mengambil kartu skor PA.
//
// Rentang periodenya dikirim TIGA KALI — kueri lama mengulang penyaring yang sama pada
// ketiga subquery-nya, dan pengulangan itu dipertahankan supaya masing-masing subquery
// tetap dapat dibaca berdampingan dengan rule aslinya.
//
// Subquery KEEMPAT sengaja TIDAK menerima periode: rentangnya terkunci pada 2023 di dalam
// teks kueri. Lihat reportkpi_admin.sql.
func (r *Repo) adminTotalsPA(
	ctx context.Context,
	q reportkpi.AdminQuery,
) (reportkpi.AdminTotals, error) {
	var (
		registerOver, paymentOver   sql.NullFloat64
		claimTotal, paymentTotal    sql.NullFloat64
		registerScore, paymentScore sql.NullFloat64
	)

	err := r.db.QueryRowContext(ctx, query("admin_scorecard_pa"),
		q.Range.From, q.Range.To,
		q.Range.From, q.Range.To,
		q.Range.From, q.Range.To,
	).Scan(
		&registerOver, &paymentOver, &claimTotal, &paymentTotal,
		&registerScore, &paymentScore,
	)
	if err != nil {
		return reportkpi.AdminTotals{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca kartu skor PA: %w", err)
	}

	return reportkpi.AdminTotals{
		RegisterOverSLA: score(registerOver),
		RegisterTotal:   score(claimTotal),
		RegisterScore:   score(registerScore),
		PaymentOverSLA:  score(paymentOver),
		PaymentTotal:    score(paymentTotal),
		PaymentScore:    score(paymentScore),
	}, nil
}

// AdminDetail mengambil satu halaman grid rincian tab KPI Admin.
func (r *Repo) AdminDetail(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	if q.Group == reportkpi.AdminGroupPA {
		return r.adminDetailPA(ctx, q, page)
	}
	return r.adminDetailNonMBU(ctx, q, page)
}

// adminDetailNonMBU mengambil rincian NON-MBU — tujuh kolom.
func (r *Repo) adminDetailNonMBU(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	clean := page.Normalize()

	rows, err := r.db.QueryContext(ctx, query("admin_detail_nonmbu"),
		q.Range.From, q.Range.To, clean.Offset(), clean.Size,
	)
	if err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian NON-MBU: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := reportkpi.AdminDetailPage{Rows: []reportkpi.AdminDetailRow{}}
	for rows.Next() {
		var (
			claimNumber, policyNumber, businessName, teamFlag sql.NullString
			registerDate, transferDate                        sql.NullTime
			registerAging                                     sql.NullFloat64
			total                                             sql.NullInt64
		)
		if err := rows.Scan(
			&claimNumber, &policyNumber, &businessName,
			&registerDate, &transferDate, &teamFlag, &registerAging, &total,
		); err != nil {
			return reportkpi.AdminDetailPage{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai rincian NON-MBU: %w", err)
		}

		result.Rows = append(result.Rows, reportkpi.AdminDetailRow{
			ClaimNumber:   claimNumber.String,
			PolicyNumber:  policyNumber.String,
			BusinessName:  businessName.String,
			RegisterDate:  isoDate(registerDate),
			TransferDate:  isoDate(transferDate),
			TeamFlag:      teamFlag.String,
			RegisterAging: score(registerAging),
		})
		result.Total = int(total.Int64)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian NON-MBU: %w", err)
	}
	return result, nil
}

// adminDetailPA mengambil rincian PA — dua umur dan dua penanda SLA.
func (r *Repo) adminDetailPA(
	ctx context.Context,
	q reportkpi.AdminQuery,
	page reportkpi.Pagination,
) (reportkpi.AdminDetailPage, error) {
	clean := page.Normalize()

	rows, err := r.db.QueryContext(ctx, query("admin_detail_pa"),
		q.Range.From, q.Range.To, clean.Offset(), clean.Size,
	)
	if err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian PA: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := reportkpi.AdminDetailPage{Rows: []reportkpi.AdminDetailRow{}}
	for rows.Next() {
		var (
			claimNumber, policyNumber, adminName sql.NullString
			registerSLA, paymentSLA, claimStatus sql.NullString
			policyStart, policyEnd               sql.NullTime
			receiveDate, registerDate            sql.NullTime
			lodReceiveDate, acceptanceDate       sql.NullTime
			registerAging, paymentAging          sql.NullFloat64
			total                                sql.NullInt64
		)
		if err := rows.Scan(
			&claimNumber, &policyNumber, &policyStart, &policyEnd, &adminName,
			&receiveDate, &registerDate, &lodReceiveDate, &acceptanceDate,
			&registerAging, &paymentAging, &registerSLA, &paymentSLA, &claimStatus,
			&total,
		); err != nil {
			return reportkpi.AdminDetailPage{}, fmt.Errorf(
				"reportkpi/sqlstore: memindai rincian PA: %w", err)
		}

		// Tanggal berlaku polis DIBACA tetapi tidak digambar sebagai kolom — kueri lama
		// mengambilnya, grid-nya tidak menampilkannya. Dibiarkan terbaca supaya
		// penelusuran ke sistem lama tetap mungkin dan supaya jumlah kolom pindaian
		// cocok dengan jumlah kolom kueri.
		_, _ = policyStart, policyEnd

		result.Rows = append(result.Rows, reportkpi.AdminDetailRow{
			ClaimNumber:    claimNumber.String,
			PolicyNumber:   policyNumber.String,
			AdminName:      adminName.String,
			ReceiveDate:    isoDate(receiveDate),
			RegisterDate:   isoDate(registerDate),
			LODReceiveDate: isoDate(lodReceiveDate),
			AcceptanceDate: isoDate(acceptanceDate),
			RegisterAging:  score(registerAging),
			PaymentAging:   score(paymentAging),
			RegisterSLA:    registerSLA.String,
			PaymentSLA:     paymentSLA.String,
			ClaimStatus:    claimStatus.String,
		})
		result.Total = int(total.Int64)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.AdminDetailPage{}, fmt.Errorf(
			"reportkpi/sqlstore: membaca rincian PA: %w", err)
	}
	return result, nil
}

// score mengubah hasil pindaian menjadi nilai domain.
//
// NULL menjadi "tidak ada", BUKAN nol. Pada kartu skor perbedaan itu nyata: nol berarti
// tidak satu pun klaim melewati SLA — hasil TERBAIK — sedangkan tidak ada berarti
// pembaginya nol karena tidak ada klaim sama sekali pada periode itu.
func score(value sql.NullFloat64) reportkpi.Score {
	if !value.Valid {
		return reportkpi.EmptyScore()
	}
	return reportkpi.NewScore(value.Float64)
}
