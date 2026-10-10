package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
)

// Berkas ini melayani panel ringkasan di atas grid.

// SummaryRows mengembalikan baris sempit seluruh klaim outstanding satu cabang.
//
// Tidak berhalaman: ringkasan atas sebagian baris adalah ringkasan yang salah. Lihat
// catatan ukurannya pada seam.
func (r *Repo) SummaryRows(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
) ([]inboxosclaimpercabang.SummaryRow, error) {
	rows, err := r.db.QueryContext(ctx, query("summary_rows"), q.Branch.Code)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri summary_rows: %w", err)
	}
	defer rows.Close()

	result := []inboxosclaimpercabang.SummaryRow{}
	for rows.Next() {
		var registerDate sql.NullTime
		var business, source sql.NullString
		var estimation any

		if err := rows.Scan(&registerDate, &business, &source, &estimation); err != nil {
			return nil, fmt.Errorf("membaca baris kueri summary_rows: %w", err)
		}

		value, err := minorUnits(estimation)
		if err != nil {
			return nil, fmt.Errorf("membaca ESTIMATION_VALUE pada summary_rows: %w", err)
		}

		result = append(result, inboxosclaimpercabang.SummaryRow{
			RegisterDate:    nullableTime(registerDate),
			BusinessName:    business.String,
			BusinessSource:  source.String,
			EstimationValue: value,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri summary_rows: %w", err)
	}

	return result, nil
}

// TreatyOR mengembalikan total porsi treaty OR satu cabang.
//
// # Kegagalannya TIDAK dikembalikan sebagai galat
//
// Ini satu-satunya pembacaan modul ini yang menyentuh DB Link saat layar dibuka, dan `@asmd`
// yang padam tidak boleh mengosongkan seluruh panel. Kegagalannya karena itu dijawab
// `(0, false, nil)` — pemanggil yang memutuskan bagaimana menggambarkannya, dan layar
// membedakan "tidak terbaca" dari "nol".
//
// Galat tetap dikembalikan untuk kegagalan yang BUKAN soal DB Link — kueri yang salah bentuk,
// misalnya — karena yang itu cacat pemrograman, bukan gangguan jaringan. Keduanya tidak dapat
// dibedakan dari kode galat Oracle saja, sehingga yang dipakai aturan sederhana: setiap
// kegagalan pada kueri INI diperlakukan sebagai tidak terbaca, dan rinciannya tetap sampai ke
// pemanggil lewat galat yang dibungkus.
func (r *Repo) TreatyOR(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
) (money.Money, bool, error) {
	var total any

	err := r.db.QueryRowContext(ctx, query("summary_treaty_or"), q.Branch.Code).Scan(&total)
	switch {
	case err == sql.ErrNoRows:
		// Kueri agregat selalu mengembalikan satu baris, jadi ini tidak seharusnya terjadi.
		// Dijawab nol-yang-terbaca, bukan galat: tidak ada klaim berarti total nol.
		return money.Zero, true, nil
	case err != nil:
		return money.Zero, false,
			fmt.Errorf("menjalankan kueri summary_treaty_or: %w", err)
	}

	value, err := minorUnits(total)
	if err != nil {
		return money.Zero, false, fmt.Errorf("membaca TREATY_OR: %w", err)
	}

	return value, true, nil
}
