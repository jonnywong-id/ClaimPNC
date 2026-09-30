package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/money"
)

// TransferRepo membaca rincian "Lihat Detail Transfer" dari tabel WARISAN.
//
// Ia tidak pernah menulis. Kedua tabel yang dibacanya masih ditulis Pega (`P-1`).
type TransferRepo struct {
	db *sql.DB
}

// NewTransferRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewTransferRepo(db *sql.DB) *TransferRepo { return &TransferRepo{db: db} }

// FindTransfer mengembalikan rincian transfer satu case komite.
//
// Case tanpa baris adjustment BUKAN galat. Dari 189 case yang dapat muncul di inbox, hanya
// 41 punya barisnya — sisanya belum sampai ke tahap itu, atau berjenis komite yang memang
// tidak memilikinya. Menjadikannya galat akan membuat mayoritas case gagal dibuka.
func (r *TransferRepo) FindTransfer(
	ctx context.Context,
	caseID string,
) (komite.TransferDetail, error) {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		return komite.TransferDetail{}, nil
	}

	lines, err := r.lines(ctx, caseID)
	if err != nil {
		return komite.TransferDetail{}, err
	}
	members, err := r.entries(ctx, "transfer_members", caseID)
	if err != nil {
		return komite.TransferDetail{}, err
	}

	record, found, err := r.committee(ctx, caseID)
	if err != nil {
		return komite.TransferDetail{}, err
	}

	panel, bisnis, kunciKlaim, err := r.caseContext(ctx, caseID)
	if err != nil {
		return komite.TransferDetail{}, err
	}

	// Case yang dibentuk aplikasi ini tidak punya baris kerja Pega; kepalanya di
	// TC_PNC_KOMITE. Hanya dicoba untuk nomor KMTN supaya portal yang belum punya tabel itu
	// tetap dapat membuka case Pega.
	if strings.TrimSpace(kunciKlaim) == "" && isNewCase(caseID) {
		panel, bisnis, kunciKlaim, err = r.newCaseContext(ctx, caseID)
		if err != nil {
			return komite.TransferDetail{}, err
		}
	}

	detail := komite.TransferDetail{
		Lines:              lines,
		Committee:          record,
		HasCommitteeRecord: found,
		GroupPanel:         panel,
		BusinessType:       bisnis,
		Members:            members,
	}

	// Tanpa kunci klaim, blok klaim dan coverage tidak dapat dibaca — tetapi rincian
	// uangnya tetap sah. Ketiadaannya karena itu bukan galat.
	if strings.TrimSpace(kunciKlaim) == "" {
		return detail, nil
	}

	detail.Claim, detail.HasClaim, err = r.claim(ctx, kunciKlaim)
	if err != nil {
		return komite.TransferDetail{}, err
	}
	detail.Coverages, err = r.coverages(ctx, kunciKlaim)
	if err != nil {
		return komite.TransferDetail{}, err
	}
	detail.Spreading, err = r.spreading(ctx, kunciKlaim)
	if err != nil {
		return komite.TransferDetail{}, err
	}
	detail.DominantFactors, err = r.dominantFactors(ctx, kunciKlaim)
	if err != nil {
		return komite.TransferDetail{}, err
	}
	detail.Attachments, err = r.attachments(ctx, kunciKlaim)
	if err != nil {
		return komite.TransferDetail{}, err
	}

	// Case KMTN dinaungi TC_PNC_KOMITE, case Pega dinaungi baris kerja Pega; satu klaim
	// hanya punya salah satunya, dan tabel TC_PNC_KOMITE tidak disentuh untuk case Pega.
	historyQuery := "transfer_history_legacy"
	if isNewCase(caseID) {
		historyQuery = "transfer_history_new"
	}
	detail.History, err = r.entries(ctx, historyQuery, kunciKlaim, caseID)
	if err != nil {
		return komite.TransferDetail{}, err
	}

	if polis := strings.TrimSpace(detail.Claim.PolicyNumber); polis != "" {
		detail.Policy, detail.HasPolicy, err = r.policy(ctx, polis)
		if err != nil {
			return komite.TransferDetail{}, err
		}
	}
	return detail, nil
}

// isNewCase menyatakan nomor case dibentuk aplikasi ini — "KMTN.YY.n" atau bentuk lama
// "KMTN-00001". Nomor Pega berawalan "KMT-".
func isNewCase(caseID string) bool {
	id := strings.ToUpper(strings.TrimSpace(caseID))
	return strings.HasPrefix(id, "KMTN.") || strings.HasPrefix(id, "KMTN-")
}

// newCaseContext membaca padanan caseContext dari TC_PNC_KOMITE.
func (r *TransferRepo) newCaseContext(
	ctx context.Context,
	caseID string,
) (string, string, string, error) {
	var panel, bisnis, kunciKlaim any
	err := r.db.QueryRowContext(ctx, query("transfer_case_new"), caseID).
		Scan(&panel, &bisnis, &kunciKlaim)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"komite/sqlstore: membaca kepala case komite %s: %w", caseID, err)
	}
	return toText(panel), toText(bisnis), toText(kunciKlaim), nil
}

// policy membaca periode, CoinsList, dan FacOfferList dokumen polis terbaru.
func (r *TransferRepo) policy(
	ctx context.Context,
	policyNumber string,
) (komite.PolicyFacts, bool, error) {
	var facts komite.PolicyFacts

	var start, end sql.NullString
	err := r.db.QueryRowContext(ctx, query("transfer_policy"), policyNumber).Scan(&start, &end)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: membaca polis: %w", err)
	}
	facts.Start = parsePegaTime(start.String)
	facts.End = parsePegaTime(end.String)

	rows, err := r.db.QueryContext(ctx, query("transfer_coinsurance"), policyNumber)
	if err != nil {
		return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: membaca koasuransi: %w", err)
	}
	for rows.Next() {
		var leader, name, pct sql.NullString
		if err := rows.Scan(&leader, &name, &pct); err != nil {
			_ = rows.Close()
			return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: baris koasuransi: %w", err)
		}
		facts.Coinsurance = append(facts.Coinsurance, komite.CoinsuranceShare{
			Name:    strings.TrimSpace(name.String),
			Leader:  strings.EqualFold(strings.TrimSpace(leader.String), "true"),
			Percent: strings.TrimSpace(pct.String),
		})
	}
	if err := closeRows(rows); err != nil {
		return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: menelusuri koasuransi: %w", err)
	}

	rows, err = r.db.QueryContext(ctx, query("transfer_fac_offer"), policyNumber)
	if err != nil {
		return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: membaca fac offer: %w", err)
	}
	for rows.Next() {
		var name, pct sql.NullString
		if err := rows.Scan(&name, &pct); err != nil {
			_ = rows.Close()
			return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: baris fac offer: %w", err)
		}
		facts.FacOffers = append(facts.FacOffers, komite.FacOffer{
			ReinsurerName: strings.TrimSpace(name.String),
			Percent:       strings.TrimSpace(pct.String),
		})
	}
	if err := closeRows(rows); err != nil {
		return komite.PolicyFacts{}, false, fmt.Errorf("komite/sqlstore: menelusuri fac offer: %w", err)
	}
	return facts, found, nil
}

func (r *TransferRepo) spreading(ctx context.Context, kunciKlaim string) ([]komite.SpreadingShare, error) {
	rows, err := r.db.QueryContext(ctx, query("transfer_spreading"), kunciKlaim)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca spreading %s: %w", kunciKlaim, err)
	}
	var hasil []komite.SpreadingShare
	for rows.Next() {
		var objectID, coverageID, treaty, name, pct any
		if err := rows.Scan(&objectID, &coverageID, &treaty, &name, &pct); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("komite/sqlstore: baris spreading: %w", err)
		}
		hasil = append(hasil, komite.SpreadingShare{
			ObjectID:   toText(objectID),
			CoverageID: toText(coverageID),
			TreatyType: toText(treaty),
			TreatyName: toText(name),
			Percent:    toText(pct),
		})
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri spreading: %w", err)
	}
	return hasil, nil
}

func (r *TransferRepo) dominantFactors(ctx context.Context, kunciKlaim string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query("transfer_dominant_factors"), kunciKlaim)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca faktor dominan %s: %w", kunciKlaim, err)
	}
	var hasil []string
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("komite/sqlstore: baris faktor dominan: %w", err)
		}
		if n := strings.TrimSpace(name.String); n != "" {
			hasil = append(hasil, n)
		}
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri faktor dominan: %w", err)
	}
	return hasil, nil
}

// legacyPrefix adalah prefix kelas Pega pada kunci klaim warisan (`D-22`).
const legacyPrefix = "ASM-FW-GCNMFW-WORK "

func (r *TransferRepo) attachments(ctx context.Context, kunciKlaim string) ([]komite.Attachment, error) {
	nomor := strings.TrimSpace(strings.TrimPrefix(kunciKlaim, legacyPrefix))
	rows, err := r.db.QueryContext(ctx, query("transfer_attachments"),
		kunciKlaim, nomor, legacyPrefix+nomor)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca lampiran %s: %w", kunciKlaim, err)
	}
	var hasil []komite.Attachment
	for rows.Next() {
		var id, name, note, category, by sql.NullString
		var at sql.NullTime
		if err := rows.Scan(&id, &name, &note, &category, &by, &at); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("komite/sqlstore: baris lampiran: %w", err)
		}
		hasil = append(hasil, komite.Attachment{
			ID:       strings.TrimSpace(id.String),
			Name:     strings.TrimSpace(name.String),
			Note:     strings.TrimSpace(note.String),
			Category: strings.TrimSpace(category.String),
			InputBy:  strings.TrimSpace(by.String),
			InputAt:  at.Time,
		})
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri lampiran: %w", err)
	}
	return hasil, nil
}

// entries membaca baris anggota komite (Daftar Komite atau History) — keduanya berbentuk
// sama: nama, jenjang, status, catatan, tanggal, Komite ID.
func (r *TransferRepo) entries(ctx context.Context, name string, args ...any) ([]komite.CommitteeEntry, error) {
	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca %s: %w", name, err)
	}
	var hasil []komite.CommitteeEntry
	for rows.Next() {
		var member, tier, status, note, caseID any
		var at sql.NullTime
		if err := rows.Scan(&member, &tier, &status, &note, &at, &caseID); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("komite/sqlstore: baris %s: %w", name, err)
		}
		level, _ := toInt(tier)
		hasil = append(hasil, komite.CommitteeEntry{
			CaseID:     toText(caseID),
			MemberName: toText(member),
			Tier:       level,
			Status:     toText(status),
			Note:       toText(note),
			DecidedAt:  at.Time,
		})
	}
	if err := closeRows(rows); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri %s: %w", name, err)
	}
	return hasil, nil
}

// closeRows menutup rows dan mengembalikan galat penelusurannya bila ada.
func closeRows(rows *sql.Rows) error {
	iterErr := rows.Err()
	closeErr := rows.Close()
	if iterErr != nil {
		return iterErr
	}
	return closeErr
}

// parsePegaTime membaca cap waktu berbentuk Pega: `20260801T050000.000 GMT`.
//
// Sama dengan pengurai di registrasi/sqlstore. Bentuk yang tidak dikenali menjadi waktu
// kosong — layar menampilkannya "—", bukan tanggal tahun 1.
func parsePegaTime(text string) time.Time {
	clean := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "GMT"))
	if clean == "" {
		return time.Time{}
	}
	for _, layout := range []string{"20060102T150405.000", "20060102T150405", "20060102"} {
		if t, err := time.Parse(layout, clean); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// caseContext membaca medan baris kerja yang tidak ada di kedua tabel nilai.
//
// Ia dipisahkan dari committee() karena Oracle menolak menggabungkannya — lihat komentar
// pada `transfer_case`.
//
// Nilai ketiga yang dikembalikannya, kunci klaim, sengaja TIDAK dipangkas prefiksnya di
// sini: ia dipakai sebagai kunci join ke tabel POOLDATA yang menyimpannya ber-prefix.
func (r *TransferRepo) caseContext(
	ctx context.Context,
	caseID string,
) (string, string, string, error) {
	var panel, bisnis, kunciKlaim any
	err := r.db.QueryRowContext(ctx, query("transfer_case"), caseID).
		Scan(&panel, &bisnis, &kunciKlaim)
	if err != nil {
		return "", "", "", fmt.Errorf(
			"komite/sqlstore: membaca konteks case komite %s: %w", caseID, err)
	}
	return toText(panel), toText(bisnis), toText(kunciKlaim), nil
}

func (r *TransferRepo) claim(
	ctx context.Context,
	kunciKlaim string,
) (komite.ClaimSummary, bool, error) {
	var (
		dol, register              sql.NullTime
		location, chronology       any
		status, recommendation     any
		share, coins, mata, gratia any
		count                      int

		claimNo, polis, insured, bisnis   any
		cabang, sob, peran, panel, kodeMU any
	)

	err := r.db.QueryRowContext(ctx, query("transfer_claim"), kunciKlaim).Scan(
		&dol, &register, &location, &chronology, &status,
		&recommendation, &share, &coins, &mata, &gratia, &count,
		&claimNo, &polis, &insured, &bisnis, &cabang, &sob, &peran, &panel, &kodeMU,
	)
	if err != nil {
		return komite.ClaimSummary{}, false, fmt.Errorf(
			"komite/sqlstore: membaca klaim %s: %w", kunciKlaim, err)
	}
	if count == 0 {
		return komite.ClaimSummary{}, false, nil
	}

	return komite.ClaimSummary{
		DateOfLoss:     dol.Time,
		RegisterDate:   register.Time,
		Location:       toText(location),
		Chronology:     toText(chronology),
		ClaimStatus:    toText(status),
		Recommendation: toText(recommendation),
		ASMShare:       toText(share),
		CoinsName:      toText(coins),
		Currency:       toText(mata),
		ExGratia:       toText(gratia),

		ClaimNumber:      toText(claimNo),
		PolicyNumber:     toText(polis),
		InsuredName:      toText(insured),
		BusinessName:     toText(bisnis),
		BranchName:       toText(cabang),
		SourceOfBusiness: toText(sob),
		CoinsRole:        toText(peran),
		GroupPanel:       toText(panel),
		CurrencyCode:     toText(kodeMU),
	}, true, nil
}

func (r *TransferRepo) coverages(
	ctx context.Context,
	kunciKlaim string,
) ([]komite.CoverageAnalysis, error) {
	rows, err := r.db.QueryContext(ctx, query("transfer_coverages"), kunciKlaim)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca coverage %s: %w", kunciKlaim, err)
	}
	defer func() { _ = rows.Close() }()

	var hasil []komite.CoverageAnalysis
	for rows.Next() {
		var (
			objectID, coverageID       any
			objectName, coverageName   any
			cause                      any
			tsi                        any
			mata                       any
			circum, extent, liability  any
			remarks, diagnose, initial any
			tanggal                    sql.NullTime
		)
		if err := rows.Scan(
			&objectID, &coverageID, &objectName, &coverageName, &cause,
			&tsi, &mata, &circum, &extent, &liability,
			&remarks, &diagnose, &initial, &tanggal,
		); err != nil {
			return nil, fmt.Errorf("komite/sqlstore: membaca baris coverage: %w", err)
		}

		// TSI adalah nilai uang, dan ia dibandingkan terhadap nilai akseptasi oleh orang
		// yang sedang memutuskan. Ia gagal KERAS, sama seperti nilai uang lainnya.
		sum, err := moneyOrZero(tsi)
		if err != nil {
			return nil, fmt.Errorf("komite/sqlstore: TSI coverage %s: %w", kunciKlaim, err)
		}

		hasil = append(hasil, komite.CoverageAnalysis{
			ObjectID:       toText(objectID),
			CoverageID:     toText(coverageID),
			ObjectName:     toText(objectName),
			CoverageName:   toText(coverageName),
			CauseOfLoss:    toText(cause),
			SumInsured:     sum,
			Currency:       toText(mata),
			Circumstances:  toText(circum),
			ExtentOfLoss:   toText(extent),
			LegalLiability: toText(liability),
			Remarks:        toText(remarks),
			Diagnose:       toText(diagnose),
			InitialName:    toText(initial),
			CommitteeDate:  tanggal.Time,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri coverage: %w", err)
	}
	return hasil, nil
}

func (r *TransferRepo) lines(ctx context.Context, caseID string) ([]komite.AdjustmentLine, error) {
	rows, err := r.db.QueryContext(ctx, query("transfer_lines"), caseID)
	if err != nil {
		return nil, fmt.Errorf("komite/sqlstore: membaca rincian transfer: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var lines []komite.AdjustmentLine
	for rows.Next() {
		line, err := scanAdjustment(rows)
		if err != nil {
			return nil, fmt.Errorf("komite/sqlstore: membaca baris transfer %s: %w", caseID, err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("komite/sqlstore: menelusuri rincian transfer: %w", err)
	}
	return lines, nil
}

func (r *TransferRepo) committee(
	ctx context.Context,
	caseID string,
) (komite.CommitteeRecord, bool, error) {
	var (
		name, tier, kindCode, paymentType, note any
		claimValue, shareASM                    any
		decidedAt, createdAt                    sql.NullTime
		approve                                 any
		count                                   int
	)

	err := r.db.QueryRowContext(ctx, query("transfer_committee"), caseID).Scan(
		&name, &tier, &kindCode, &paymentType, &note,
		&claimValue, &shareASM, &decidedAt, &approve, &count, &createdAt,
	)
	if err != nil {
		return komite.CommitteeRecord{}, false, fmt.Errorf(
			"komite/sqlstore: membaca keputusan komite %s: %w", caseID, err)
	}
	if count == 0 {
		return komite.CommitteeRecord{}, false, nil
	}

	// Nilai klaim TIDAK boleh gagal diam-diam: ia yang dibandingkan terhadap ambang
	// penjenjangan, dan nol yang seharusnya empat puluh lima juta tidak terlihat keliru.
	value, err := moneyOrZero(claimValue)
	if err != nil {
		return komite.CommitteeRecord{}, false, fmt.Errorf(
			"komite/sqlstore: nilai klaim komite %s: %w", caseID, err)
	}

	// Jenjang adalah keterangan pelengkap. Yang tidak terbaca menjadi nol, bukan
	// menggagalkan seluruh rincian.
	level, levelErr := toInt(tier)
	if levelErr != nil {
		level = 0
	}

	return komite.CommitteeRecord{
		MemberName: toText(name),
		Tier:       level,
		Kind:       komite.CommitteeKindOf(toText(kindCode), toText(paymentType)),

		// Kode mentahnya ikut dibawa: judul layar menurunkan hal LAIN dari kode yang
		// sama, dan menurunkannya balik dari Kind berarti menebak.
		TransferTypeCode: toText(kindCode),
		PaymentTypeCode:  toText(paymentType),

		Note:       toText(note),
		ClaimValue: value,
		ASMShare:   toText(shareASM),
		DecidedAt:  decidedAt.Time,
		Outcome:    legacyOutcome(toText(approve)),
		CreatedAt:  createdAt.Time,
	}, true, nil
}

// CheckTables memastikan kedua tabel dapat dibaca akun aplikasi.
func (r *TransferRepo) CheckTables(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("transfer_check_table"))
	if err != nil {
		return fmt.Errorf("komite/sqlstore: memeriksa tabel rincian transfer: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// scanAdjustment memetakan satu baris `T_CLAIM_ADJUSTMENT` menjadi baris domain.
//
// # Di sinilah nama kolom warisan berhenti
//
// `NILAIAKSEPTASI`, `NILAI_SALVAGE_A`, `PROPOSE_VALUE`, `CIRCUMCAUSEOFLOSS` — seluruhnya
// berhenti di fungsi ini. Lapisan di atasnya membaca nama yang sesuai isinya (`D-19`).
//
// # Nilai uang gagal KERAS, keterangan tidak
//
// Keterangan yang tidak terbaca cukup menjadi teks kosong. Nilai uang tidak: ia yang dibaca
// anggota komite saat memutuskan, dan nol yang seharusnya angka besar tidak terlihat
// keliru. Keduanya sengaja diperlakukan berbeda.
func scanAdjustment(row scanner) (komite.AdjustmentLine, error) {
	var (
		claimID, objectID, coverageID any
		acceptanceNo                  any
		acceptedAt                    sql.NullTime
		currency, paymentType         any
		gross, propose, accepted      any
		salvage, sharePct, shareValue any
		individualRisk                any
		exGratia, notes, cause        any
		currencyCode, adjusterFee     any
		adjustmentID, totalClaim, loc any
		riskType, riskPct             any
		estimation, interim           any
	)

	if err := row.Scan(
		&claimID, &objectID, &coverageID, &acceptanceNo, &acceptedAt,
		&currency, &paymentType,
		&gross, &propose, &accepted, &salvage, &sharePct, &shareValue,
		&individualRisk, &exGratia, &notes, &cause,
		&currencyCode, &adjusterFee,
		&adjustmentID, &totalClaim, &loc, &riskType, &riskPct, &estimation, &interim,
	); err != nil {
		return komite.AdjustmentLine{}, err
	}

	uang := func(nama string, raw any) (money.Money, error) {
		value, err := moneyOrZero(raw)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", nama, err)
		}
		return value, nil
	}

	grossValue, err := uang("gross", gross)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	proposeValue, err := uang("propose", propose)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	acceptedValue, err := uang("nilai akseptasi", accepted)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	salvageValue, err := uang("salvage", salvage)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	shareMoney, err := uang("nilai ASM share", shareValue)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	riskValue, err := uang("risiko sendiri", individualRisk)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	feeValue, err := uang("fee adjuster", adjusterFee)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	totalValue, err := uang("total klaim", totalClaim)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	estimationValue, err := uang("estimasi", estimation)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}
	interimValue, err := uang("interim", interim)
	if err != nil {
		return komite.AdjustmentLine{}, err
	}

	line := komite.AdjustmentLine{
		// Prefix kelas Pega dibuang di sini juga, dengan alasan yang sama seperti pada
		// daftar: ia tidak pernah boleh sampai ke domain maupun ke layar (`D-22`).
		ClaimNumber: strings.TrimSpace(
			strings.ReplaceAll(toText(claimID), "ASM-FW-GCNMFW-WORK ", "")),
		ObjectID:     toText(objectID),
		CoverageID:   toText(coverageID),
		AcceptanceNo: toText(acceptanceNo),
		AcceptedAt:   acceptedAt.Time,

		Currency:     toText(currency),
		PaymentType:  toText(paymentType),
		CurrencyCode: toText(currencyCode),
		AdjusterFee:  feeValue,

		AdjustmentID:  toText(adjustmentID),
		TotalClaim:    totalValue,
		LOCPercent:    toText(loc),
		RiskType:      toText(riskType),
		RiskPercent:   toText(riskPct),
		Estimation:    estimationValue,
		HasEstimation: estimation != nil,
		InterimPaid:   interimValue,

		GrossValue:     grossValue,
		ProposeValue:   proposeValue,
		AcceptedValue:  acceptedValue,
		SalvageValue:   salvageValue,
		ASMShareValue:  shareMoney,
		IndividualRisk: riskValue,

		ASMSharePercent: toText(sharePct),
		ExGratia:        toFlag(exGratia),
		Notes:           toText(notes),
		CauseOfLoss:     toText(cause),
	}
	return line, nil
}
