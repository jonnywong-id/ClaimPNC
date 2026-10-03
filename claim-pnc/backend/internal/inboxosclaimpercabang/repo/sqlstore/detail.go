package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Berkas ini melayani popup Detail. Ia terpisah dari inboxosclaimpercabang.go karena isinya
// menjawab pertanyaan yang berbeda — satu klaim, bukan satu halaman daftar — dan menempuh
// lima kueri alih-alih satu.

// FindDetail mengambil seluruh isi popup untuk satu klaim milik cabang pada query.
//
// # Kepala dibaca LEBIH DULU, dan itu bukan sekadar urutan
//
// Kueri kepala yang menegakkan batas cabang. Keempat kueri anak menerima `CLAIMID`, dan
// `CLAIMID` itu hanya diperoleh dari kepala yang sudah lolos penyaring cabang. Menjalankan
// keempatnya lebih dulu — atau paralel — berarti membaca isi klaim sebelum diketahui ia milik
// siapa (`R-20`).
//
// Nilai kedua false berarti klaim tidak ada, atau ada tetapi bukan milik cabang itu. Keduanya
// sengaja tidak dibedakan.
func (r *Repo) FindDetail(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
	claimNumber string,
) (inboxosclaimpercabang.Detail, bool, error) {
	clean := strings.TrimSpace(claimNumber)
	if clean == "" {
		return inboxosclaimpercabang.Detail{}, false, nil
	}

	detail, found, err := r.detailHeader(ctx, q.Branch.Code, clean)
	if err != nil || !found {
		return inboxosclaimpercabang.Detail{}, false, err
	}

	if detail.Objects, err = r.detailObjects(ctx, detail.ClaimKey); err != nil {
		return inboxosclaimpercabang.Detail{}, false, err
	}

	// Riwayat progres dikunci NOMOR klaim, bukan CLAIMID: `GCNM_PROGRESS_CLAIM.PNCCASEID`
	// berisi nomor, dan kueri lama pun menyetelnya dari `.ClaimNo`.
	if detail.ProgressHistory, err = r.detailProgress(ctx, detail.ClaimNumber); err != nil {
		return inboxosclaimpercabang.Detail{}, false, err
	}

	if detail.AdjusterMessages, err = r.detailMessages(ctx, detail.ClaimKey); err != nil {
		return inboxosclaimpercabang.Detail{}, false, err
	}

	if detail.DominantFactors, err = r.detailFactors(ctx, detail.ClaimKey); err != nil {
		return inboxosclaimpercabang.Detail{}, false, err
	}

	return detail, true, nil
}

// detailHeader membaca delapan nilai ringkasan pada kepala popup.
func (r *Repo) detailHeader(
	ctx context.Context,
	branchCode, claimNumber string,
) (inboxosclaimpercabang.Detail, bool, error) {
	var (
		detail                            inboxosclaimpercabang.Detail
		number, key, business, occupation sql.NullString
		chronology, recommendation, note  sql.NullString
		sumInsured, estimation            any
		registerDate                      sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query("detail_header"), branchCode, claimNumber).Scan(
		&number, &key, &business, &occupation, &sumInsured,
		&chronology, &estimation, &registerDate, &recommendation, &note)
	switch {
	case err == sql.ErrNoRows:
		return inboxosclaimpercabang.Detail{}, false, nil
	case err != nil:
		return inboxosclaimpercabang.Detail{}, false,
			fmt.Errorf("menjalankan kueri detail_header: %w", err)
	}

	tsi, err := minorUnits(sumInsured)
	if err != nil {
		return inboxosclaimpercabang.Detail{}, false,
			fmt.Errorf("membaca TOTAL_SUM_INSURED: %w", err)
	}
	reserve, err := minorUnits(estimation)
	if err != nil {
		return inboxosclaimpercabang.Detail{}, false,
			fmt.Errorf("membaca ESTIMATION_VALUE: %w", err)
	}

	detail.ClaimNumber = number.String
	detail.ClaimKey = key.String
	detail.BusinessName = business.String
	detail.Occupation = occupation.String
	detail.TotalSumInsured = tsi
	detail.Chronology = chronology.String
	detail.EstimationValue = reserve
	detail.RegisterDate = nullableTime(registerDate)
	detail.RemarkRecommendation = recommendation.String
	detail.ProgressNote = note.String

	return detail, true, nil
}

// detailObjects membaca grid objek pertanggungan.
func (r *Repo) detailObjects(
	ctx context.Context,
	claimKey string,
) ([]inboxosclaimpercabang.DetailObject, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_objects"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_objects: %w", err)
	}
	defer rows.Close()

	// Irisan kosong, bukan nil: ia diserialkan menjadi `[]` alih-alih `null`, dan layar tidak
	// perlu membedakan "tidak ada objek" dari "gagal dibaca".
	result := []inboxosclaimpercabang.DetailObject{}
	for rows.Next() {
		var name, location, job, idCard, status sql.NullString
		var birth sql.NullTime

		if err := rows.Scan(&name, &location, &job, &birth, &idCard, &status); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_objects: %w", err)
		}

		result = append(result, inboxosclaimpercabang.DetailObject{
			Name:              name.String,
			Location:          location.String,
			Job:               job.String,
			DateOfBirth:       nullableTime(birth),
			IDCard:            idCard.String,
			ParticipantStatus: status.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_objects: %w", err)
	}

	return result, nil
}

// detailProgress membaca grid riwayat progres.
func (r *Repo) detailProgress(
	ctx context.Context,
	claimNumber string,
) ([]inboxosclaimpercabang.DetailProgress, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_progress"), claimNumber)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_progress: %w", err)
	}
	defer rows.Close()

	result := []inboxosclaimpercabang.DetailProgress{}
	for rows.Next() {
		var number, status1, status2, enteredBy, status, note sql.NullString
		var recordedAt, followUpAt sql.NullTime

		if err := rows.Scan(&recordedAt, &number, &status1, &status2,
			&enteredBy, &followUpAt, &status, &note); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_progress: %w", err)
		}

		result = append(result, inboxosclaimpercabang.DetailProgress{
			RecordedAt:     nullableTime(recordedAt),
			ClaimNumber:    number.String,
			Status1:        status1.String,
			Status2:        status2.String,
			EnteredBy:      enteredBy.String,
			NextFollowUpAt: nullableTime(followUpAt),
			Status:         status.String,
			Note:           note.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_progress: %w", err)
	}

	return result, nil
}

// detailMessages membaca grid komunikasi dengan loss adjuster.
func (r *Repo) detailMessages(
	ctx context.Context,
	claimKey string,
) ([]inboxosclaimpercabang.DetailMessage, error) {
	// Kunci yang sama dipakai dua kali — untuk klaimnya sendiri dan untuk daftar survei
	// miliknya. Keduanya bind terpisah supaya kuerinya tetap portabel; Oracle dan PostgreSQL
	// memperlakukan pengulangan penanda bind secara berbeda.
	rows, err := r.db.QueryContext(ctx, query("detail_messages"), claimKey, claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_messages: %w", err)
	}
	defer rows.Close()

	result := []inboxosclaimpercabang.DetailMessage{}
	for rows.Next() {
		var sender, message, reply sql.NullString
		var sentAt, repliedAt sql.NullTime
		var internal sql.NullInt64

		if err := rows.Scan(&sender, &sentAt, &message,
			&repliedAt, &reply, &internal); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_messages: %w", err)
		}

		result = append(result, inboxosclaimpercabang.DetailMessage{
			SenderName: sender.String,
			SentAt:     nullableTime(sentAt),
			Message:    message.String,
			RepliedAt:  nullableTime(repliedAt),
			Reply:      reply.String,
			Internal:   internal.Int64 == 1,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_messages: %w", err)
	}

	return result, nil
}

// detailFactors membaca faktor dominan satu klaim dan merangkainya.
//
// Bentuk rangkaiannya mengikuti kueri lama apa adanya — dipisah `", "`, dan `"-"` bila kosong.
// Tanda `"-"` itu dipertahankan meski sel kosong akan sama jelasnya: ia yang selama ini
// terbaca di layar.
func (r *Repo) detailFactors(ctx context.Context, claimKey string) (string, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_dominant_factors"), claimKey)
	if err != nil {
		return "", fmt.Errorf("menjalankan kueri detail_dominant_factors: %w", err)
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return "", fmt.Errorf("membaca baris kueri detail_dominant_factors: %w", err)
		}
		if strings.TrimSpace(name.String) != "" {
			names = append(names, name.String)
		}
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("menelusuri hasil kueri detail_dominant_factors: %w", err)
	}

	if len(names) == 0 {
		return "-", nil
	}
	return strings.Join(names, ", "), nil
}

// CheckDetailTable memastikan tabel yang HANYA dipakai popup Detail terbaca.
//
// Terpisah dari CheckTable supaya kegagalannya terbaca sebagai kegagalan POPUP, bukan sebagai
// kegagalan daftarnya. Keduanya punya tindak lanjut yang berbeda: yang pertama membuat satu
// tombol tidak bekerja, yang kedua menghentikan seluruh layar.
func (r *Repo) CheckDetailTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_detail_tables")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_OBJECTLIST, POOLDATA.T_CLAIM_OBJECTCOVERAGE, "+
				"POOLDATA.GCNM_MST_PROGRESS_KLAIM, POOLDATA.M_KOMUNIKASI_PNC, "+
				"POOLDATA.MST_USER_TEKNIK, atau OCCUPATION: %w", err)
	}
	return nil
}
