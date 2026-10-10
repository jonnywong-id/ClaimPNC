package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
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
		var id, name, location, job, idCard, status sql.NullString
		var birth sql.NullTime

		if err := rows.Scan(&id, &name, &location, &job, &birth, &idCard, &status); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_objects: %w", err)
		}

		result = append(result, inboxosclaimpercabang.DetailObject{
			ID:                strings.TrimSpace(id.String),
			Name:              name.String,
			Location:          location.String,
			Job:               job.String,
			DateOfBirth:       nullableTime(birth),
			IDCard:            idCard.String,
			ParticipantStatus: status.String,
			// Irisan kosong, bukan nil — alasannya sama dengan `result` di atas.
			Coverages: []inboxosclaimpercabang.DetailCoverage{},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_objects: %w", err)
	}

	coverages, err := r.detailObjectCoverages(ctx, claimKey)
	if err != nil {
		return nil, err
	}

	items, err := r.detailObjectItems(ctx, claimKey)
	if err != nil {
		return nil, err
	}

	estimations, err := r.detailEstimations(ctx, claimKey)
	if err != nil {
		return nil, err
	}

	// Perangkaian dari dalam ke luar: estimasi ke item, item ke coverage, coverage ke objek.
	//
	// Keempatnya dibaca dengan EMPAT kueri, bukan satu kueri bersarang per baris. Pohon ini
	// dapat memuat puluhan baris estimasi pada satu klaim, dan membacanya per baris yang
	// dibuka akan menjadi N+1 berlapis — tidak terlihat sampai sebuah klaim benar-benar besar.
	for coverageKey, daftarItem := range items {
		for i := range daftarItem {
			kunci := itemKey{
				object:   daftarItem[i].ObjectID,
				coverage: daftarItem[i].CoverageID,
				item:     daftarItem[i].ID,
			}
			if milik, ada := estimations[kunci]; ada {
				daftarItem[i].Estimations = milik
			}
		}
		items[coverageKey] = daftarItem
	}

	spreadings, err := r.detailSpreading(ctx, claimKey)
	if err != nil {
		return nil, err
	}

	coMembers, err := r.detailCoMembers(ctx, claimKey)
	if err != nil {
		return nil, err
	}

	for objectID := range coverages {
		daftarCoverage := coverages[objectID]
		for i := range daftarCoverage {
			kunci := coverageKey{
				object:   daftarCoverage[i].ObjectID,
				coverage: daftarCoverage[i].ID,
			}
			if milik, ada := items[kunci]; ada {
				daftarCoverage[i].Items = milik
			}

			// Nilai uang kedua grid di bawah ini DIHITUNG dari jumlah estimasi coverage
			// ini, karena kolom tersimpannya NULL pada seluruh baris. Jumlahnya dihitung
			// sekali di sini, bukan di dalam kedua perulangan.
			jumlahEstimasi := money.Zero
			for _, item := range daftarCoverage[i].Items {
				for _, e := range item.Estimations {
					jumlahEstimasi += e.Value
				}
			}

			daftarCoverage[i].Spreadings = []inboxosclaimpercabang.DetailSpreading{}
			for _, s := range spreadings[kunci] {
				s.Currency = daftarCoverage[i].Currency
				s.EstimationValue = jumlahEstimasi
				s.ResultValue = inboxosclaimpercabang.ShareOf(
					jumlahEstimasi, s.SharePercentScaled)
				daftarCoverage[i].Spreadings = append(daftarCoverage[i].Spreadings, s)
			}

			// Daftar koasuransi berkunci POLIS, sehingga daftarnya SAMA untuk setiap
			// coverage. Yang berbeda hanya nilai uangnya, karena basisnya estimasi
			// coverage masing-masing.
			daftarCoverage[i].CoMembers = []inboxosclaimpercabang.DetailCoMember{}
			for _, m := range coMembers {
				m.ObjectID = daftarCoverage[i].ObjectID
				m.CoverageID = daftarCoverage[i].ID
				m.Currency = daftarCoverage[i].Currency
				m.EstimationValue = jumlahEstimasi
				m.ResultValue = inboxosclaimpercabang.ShareOf(
					jumlahEstimasi, m.SharePercentScaled)
				daftarCoverage[i].CoMembers = append(daftarCoverage[i].CoMembers, m)
			}
		}
		coverages[objectID] = daftarCoverage
	}

	for i := range result {
		if milik, ada := coverages[result[i].ID]; ada {
			result[i].Coverages = milik
		}
	}

	return result, nil
}

// coverageKey dan itemKey adalah kunci gabungan untuk mengelompokkan baris anak.
//
// Kunci gabungan, bukan satu kolom: `OBJECTCOVERAGEID` dan `OBJECTITEMID` hanya unik DI DALAM
// induknya — `OBJECTITEMID` bahkan bernilai `1` pada 51.370 dari 51.532 baris. Memakai salah
// satunya sendirian akan menempelkan estimasi milik coverage lain ke item yang kebetulan
// bernomor sama.
type coverageKey struct{ object, coverage string }

type itemKey struct{ object, coverage, item string }

// detailObjectCoverages membaca SELURUH coverage satu klaim sekaligus, dikelompokkan menurut
// objeknya.
//
// Satu kueri untuk seluruh klaim, bukan satu kueri per baris yang dibuka: layar ini memang
// dirancang untuk dibuka-tutup berkali-kali, dan pembacaan per baris akan menjadi N+1
// permintaan yang tidak terlihat sampai sebuah klaim punya banyak objek.
func (r *Repo) detailObjectCoverages(
	ctx context.Context,
	claimKey string,
) (map[string][]inboxosclaimpercabang.DetailCoverage, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_object_coverages"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_object_coverages: %w", err)
	}
	defer rows.Close()

	result := map[string][]inboxosclaimpercabang.DetailCoverage{}
	for rows.Next() {
		var objectID, coverageID, name, currency sql.NullString
		var sumTSI any

		if err := rows.Scan(&objectID, &coverageID, &name, &currency, &sumTSI); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_object_coverages: %w", err)
		}

		// minorUnits, BUKAN money.FromSQLValue: kuerinya sudah mengembalikan satuan
		// terkecil lewat `ROUND(... * 100)`, dan FromSQLValue akan mengalikannya seratus
		// sekali lagi — diam-diam, tanpa satu pun galat. Lihat catatan pada minorUnits.
		nilai, err := minorUnits(sumTSI)
		if err != nil {
			return nil, fmt.Errorf("membaca COVERAGE_SUM_TSI: %w", err)
		}

		kunci := strings.TrimSpace(objectID.String)
		result[kunci] = append(result[kunci], inboxosclaimpercabang.DetailCoverage{
			ObjectID: kunci,
			ID:       strings.TrimSpace(coverageID.String),
			Name:     strings.TrimSpace(name.String),
			Currency: strings.TrimSpace(currency.String),
			SumTSI:   nilai,
			// Irisan kosong, bukan nil — alasannya sama dengan di tingkat atasnya.
			Items: []inboxosclaimpercabang.DetailItem{},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_object_coverages: %w", err)
	}

	return result, nil
}

// detailObjectItems membaca grid "Object Item", dikelompokkan menurut coverage-nya.
func (r *Repo) detailObjectItems(
	ctx context.Context,
	claimKey string,
) (map[coverageKey][]inboxosclaimpercabang.DetailItem, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_object_items"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_object_items: %w", err)
	}
	defer rows.Close()

	result := map[coverageKey][]inboxosclaimpercabang.DetailItem{}
	for rows.Next() {
		var objectID, coverageID, itemID, name, description sql.NullString

		if err := rows.Scan(&objectID, &coverageID, &itemID, &name, &description); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_object_items: %w", err)
		}

		item := inboxosclaimpercabang.DetailItem{
			ObjectID:    strings.TrimSpace(objectID.String),
			CoverageID:  strings.TrimSpace(coverageID.String),
			ID:          strings.TrimSpace(itemID.String),
			Name:        strings.TrimSpace(name.String),
			Description: strings.TrimSpace(description.String),
			Estimations: []inboxosclaimpercabang.DetailEstimation{},
		}
		kunci := coverageKey{object: item.ObjectID, coverage: item.CoverageID}
		result[kunci] = append(result[kunci], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_object_items: %w", err)
	}

	return result, nil
}

// detailSpreading membaca grid "List Spreading", dikelompokkan menurut coverage-nya.
//
// Nilai uangnya TIDAK diisi di sini — ia dihitung pemanggil dari jumlah estimasi coverage,
// karena kolom tersimpannya NULL pada seluruh baris tabel.
func (r *Repo) detailSpreading(
	ctx context.Context,
	claimKey string,
) (map[coverageKey][]inboxosclaimpercabang.DetailSpreading, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_coverage_spreading"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_coverage_spreading: %w", err)
	}
	defer rows.Close()

	result := map[coverageKey][]inboxosclaimpercabang.DetailSpreading{}
	for rows.Next() {
		var objectID, coverageID, treaty sql.NullString
		var share sql.NullInt64

		if err := rows.Scan(&objectID, &coverageID, &treaty, &share); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_coverage_spreading: %w", err)
		}

		row := inboxosclaimpercabang.DetailSpreading{
			ObjectID:           strings.TrimSpace(objectID.String),
			CoverageID:         strings.TrimSpace(coverageID.String),
			TreatyName:         strings.TrimSpace(treaty.String),
			SharePercentScaled: share.Int64,
		}
		kunci := coverageKey{object: row.ObjectID, coverage: row.CoverageID}
		result[kunci] = append(result[kunci], row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_coverage_spreading: %w", err)
	}

	return result, nil
}

// detailCoMembers membaca grid "CO MEMBER".
//
// Hasilnya satu daftar, bukan peta: tabelnya berkunci POLIS, sehingga daftar yang sama berlaku
// untuk seluruh coverage pada klaim itu.
func (r *Repo) detailCoMembers(
	ctx context.Context,
	claimKey string,
) ([]inboxosclaimpercabang.DetailCoMember, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_coverage_comember"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_coverage_comember: %w", err)
	}
	defer rows.Close()

	result := []inboxosclaimpercabang.DetailCoMember{}
	for rows.Next() {
		var name sql.NullString
		var share sql.NullInt64

		if err := rows.Scan(&name, &share); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_coverage_comember: %w", err)
		}

		result = append(result, inboxosclaimpercabang.DetailCoMember{
			InsurerName:        strings.TrimSpace(name.String),
			SharePercentScaled: share.Int64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_coverage_comember: %w", err)
	}

	return result, nil
}

// detailEstimations membaca grid "Estimasi", dikelompokkan menurut itemnya.
func (r *Repo) detailEstimations(
	ctx context.Context,
	claimKey string,
) (map[itemKey][]inboxosclaimpercabang.DetailEstimation, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_estimations"), claimKey)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri detail_estimations: %w", err)
	}
	defer rows.Close()

	result := map[itemKey][]inboxosclaimpercabang.DetailEstimation{}
	for rows.Next() {
		var objectID, coverageID, itemID, sequence, jenis, currency sql.NullString
		var recordedAt sql.NullTime
		var rate, value any

		if err := rows.Scan(&objectID, &coverageID, &itemID, &sequence,
			&recordedAt, &jenis, &currency, &rate, &value); err != nil {
			return nil, fmt.Errorf("membaca baris kueri detail_estimations: %w", err)
		}

		// minorUnits, BUKAN money.FromSQLValue — lihat catatan pada minorUnits.
		nilaiKurs, err := minorUnits(rate)
		if err != nil {
			return nil, fmt.Errorf("membaca ESTIMATION_RATE: %w", err)
		}
		nilai, err := minorUnits(value)
		if err != nil {
			return nil, fmt.Errorf("membaca ESTIMATION_VALUE: %w", err)
		}

		row := inboxosclaimpercabang.DetailEstimation{
			ObjectID:   strings.TrimSpace(objectID.String),
			CoverageID: strings.TrimSpace(coverageID.String),
			ItemID:     strings.TrimSpace(itemID.String),
			Sequence:   strings.TrimSpace(sequence.String),
			RecordedAt: nullableTime(recordedAt),
			Type:       strings.TrimSpace(jenis.String),
			Currency:   strings.TrimSpace(currency.String),
			Rate:       nilaiKurs,
			Value:      nilai,
		}
		kunci := itemKey{object: row.ObjectID, coverage: row.CoverageID, item: row.ItemID}
		result[kunci] = append(result[kunci], row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri detail_estimations: %w", err)
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
