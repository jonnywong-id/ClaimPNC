package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/registrasi"
)

// ClaimStore menyimpan klaim beserta pohon objek–coverage–spreading di bawahnya.
type ClaimStore struct {
	db *sql.DB
}

// NewClaimStore membentuk repo; db wajib sudah terhubung.
func NewClaimStore(db *sql.DB) *ClaimStore { return &ClaimStore{db: db} }

// Save menuliskan klaim beserta seluruh pohon di bawahnya.
//
// # Kenapa UPDATE lalu INSERT, bukan MERGE
//
// Mengikuti keputusan 2.9 pada `docs/keputusan-implementasi.md`: MERGE bukan sintaks yang
// sama antara Oracle dan PostgreSQL, dan `ADR-0005` menetapkan perpindahan ke PostgreSQL
// akan datang. UPDATE-lalu-INSERT berjalan apa adanya di keduanya.
//
// # Kenapa baris anak ditandai, bukan dihapus
//
// Objek yang dibuang petugas TIDAK dihapus dari tabel; ia ditandai lewat DIHAPUS_PADA.
// `ADR-0012` melarang penghapusan fisik, dan pola hapus-lalu-sisip-ulang yang
// menggantikannya masih menunggu `ADR-0013`.
//
// Pemanggil bertanggung jawab atas batas transaksi. Bila context sudah membawa
// transaksi, seluruh pernyataan di sini ikut transaksi itu — itulah yang membuat janji
// "gagal di langkah mana pun tidak meninggalkan satu baris pun" dapat ditepati.
func (r *ClaimStore) Save(ctx context.Context, k registrasi.Claim) error {
	exec := executorFrom(ctx, r.db)

	if err := r.saveHeader(ctx, exec, k); err != nil {
		return err
	}
	return r.saveTree(ctx, exec, k)
}

func (r *ClaimStore) saveHeader(ctx context.Context, exec executor, k registrasi.Claim) error {
	args := []any{
		emptyTextAsNil(k.Number),
		k.Portal,
		k.Policy.Number,
		string(k.Policy.Line),
		k.Policy.BusinessType,
		timeOrNil(k.Policy.CoverageStart),
		timeOrNil(k.Policy.CoverageEnd),
		yesNo(k.Policy.Declaration),
		k.Policy.Currency,
		yesNo(k.Policy.CreditGuarantee),
		k.Policy.InsuredName,
		k.Policy.BranchCode,
		timeOrNil(k.DateOfLoss),
		timeOrNil(k.ReportDate),
		timeOrNil(k.DateReceived),
		k.Location,
		k.Chronology,
		k.Reporter.Name,
		k.Reporter.Phone,
		k.Reporter.Email,
		k.Reporter.Address,
		k.Reporter.Relation,
		k.Reporter.OtherRelation,
		int64(k.EstimateValue),
		k.Currency,
		k.SLIKNumber,
		yesNo(k.ExGratia),
		k.TechnicalPIC,
		k.RCVID,
		k.PUCLStatus,
		yesNo(k.ComplianceTransfer),
		yesNo(k.RequestReturn),
		string(k.ProcessStatus),
		string(k.ClaimStatus),
		string(k.ClaimFlag),
		string(k.ProgressPositionStatus),
		k.CurrentStage,
		k.UpdatedBy,
		k.UpdatedAt.UTC(),
		timePtrOrNil(k.DeletedAt),
		flagNOLL(k.LargeLossNoticed),
		k.ID,
	}

	result, err := exec.ExecContext(ctx, loadQuery("klaim_perbarui"), args...)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: memperbarui klaim: %w", err)
	}
	row, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca jumlah baris klaim: %w", err)
	}
	if row > 0 {
		return nil
	}

	args = append(args, k.CreatedBy, k.CreatedAt.UTC())
	if _, err := exec.ExecContext(ctx, loadQuery("klaim_sisip"), args...); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyisipkan klaim: %w", err)
	}
	return nil
}

func (r *ClaimStore) saveTree(ctx context.Context, exec executor, k registrasi.Claim) error {
	now := k.UpdatedAt.UTC()

	for i, o := range k.InsuredItem {
		itemSeq := i + 1
		if err := upsert(ctx, exec,
			"objek_perbarui", []any{o.ID, o.Name, o.Location, k.ID, itemSeq},
			"objek_sisip", []any{o.ID, o.Name, o.Location, k.ID, itemSeq},
		); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan objek %d: %w", itemSeq, err)
		}

		for j, c := range o.Coverage {
			coverageSeq := j + 1
			// o.ID ikut dikirim karena OBJECTID `NOT NULL` di POOLDATA.T_CLAIM_OBJECTCOVERAGE.
			// Coverage memang milik sebuah objek; tabel warisan menuntutnya dinyatakan, dan
			// domain sudah memilikinya di tangan.
			if err := upsert(ctx, exec,
				"coverage_perbarui", []any{
					c.ID, c.CauseOfLoss, int64(c.TSI), o.ID, coverageSeq,
					k.ID, itemSeq, coverageSeq},
				"coverage_sisip", []any{
					c.ID, c.CauseOfLoss, int64(c.TSI), o.ID, coverageSeq, now,
					k.ID, itemSeq, coverageSeq},
			); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menyimpan coverage %d.%d: %w", itemSeq, coverageSeq, err)
			}

			for n, s := range c.Spreading {
				spreadingSeq := n + 1
				value := []any{s.TreatyKind, s.Name, int64(s.Share), yesNo(s.Removed), s.FacOfferItem}
				if err := upsert(ctx, exec,
					"spreading_perbarui", append(append([]any{}, value...), k.ID, itemSeq, coverageSeq, spreadingSeq),
					"spreading_sisip", append(append([]any{}, value...), k.ID, itemSeq, coverageSeq, spreadingSeq),
				); err != nil {
					return fmt.Errorf("registrasi/sqlstore: menyimpan spreading %d.%d.%d: %w",
						itemSeq, coverageSeq, spreadingSeq, err)
				}
			}

			if _, err := exec.ExecContext(ctx, loadQuery("spreading_tandai_sisa"),
				now, k.ID, itemSeq, coverageSeq, len(c.Spreading)); err != nil {
				return fmt.Errorf("registrasi/sqlstore: menandai sisa spreading: %w", err)
			}
		}

		if _, err := exec.ExecContext(ctx, loadQuery("coverage_tandai_sisa"),
			now, k.ID, itemSeq, len(o.Coverage)); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menandai sisa coverage: %w", err)
		}
	}

	if _, err := exec.ExecContext(ctx, loadQuery("objek_tandai_sisa"),
		now, k.ID, len(k.InsuredItem)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menandai sisa objek: %w", err)
	}
	return nil
}

// upsert menjalankan UPDATE lebih dulu dan menyisipkan hanya bila tidak ada baris yang
// terpengaruh.
func upsert(ctx context.Context, exec executor, updateName string, updateArgs []any, insertName string, insertArgs []any) error {
	result, err := exec.ExecContext(ctx, loadQuery(updateName), updateArgs...)
	if err != nil {
		return err
	}
	row, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if row > 0 {
		return nil
	}
	_, err = exec.ExecContext(ctx, loadQuery(insertName), insertArgs...)
	return err
}

// Get mengembalikan klaim berdasarkan pengenal internalnya.
func (r *ClaimStore) Get(ctx context.Context, id string) (registrasi.Claim, error) {
	return r.getBy(ctx, "klaim_ambil", id)
}

// GetByNumber mengembalikan klaim berdasarkan nomor klaimnya.
func (r *ClaimStore) GetByNumber(ctx context.Context, number string) (registrasi.Claim, error) {
	return r.getBy(ctx, "klaim_ambil_per_nomor", number)
}

func (r *ClaimStore) getBy(ctx context.Context, queryName, value string) (registrasi.Claim, error) {
	exec := executorFrom(ctx, r.db)

	var (
		k                                       registrasi.Claim
		number, portal, line, businessType      sql.NullString
		policyStart, policyEnd                  sql.NullTime
		declarationPolicy, policyCurrency       sql.NullString
		creditPolicy, insured, branchCode       sql.NullString
		lossDate, reportDate, receivedDate      sql.NullTime
		location, chronology                    sql.NullString
		rName, rPhone, rEmail, rAddress         sql.NullString
		rRelation                               sql.NullInt64
		rRelationOther                          sql.NullString
		estimate                                sql.NullInt64
		currency, slikNumber, exGratia          sql.NullString
		technicalPIC, rcvID                     sql.NullString
		puclStatus                              sql.NullInt64
		complianceTransfer, requestReturn       sql.NullString
		processStatus, claimStatus              sql.NullString
		claimFlag, positionStatus, currentStage sql.NullString
		createdBy, updatedBy                    sql.NullString
		createdAt, updatedAt                    sql.NullTime
		deletedAt                               sql.NullTime
		flagNoll                                sql.NullString
	)

	row := exec.QueryRowContext(ctx, loadQuery(queryName), value)
	err := row.Scan(
		&k.ID, &number, &portal,
		&k.Policy.Number, &line, &businessType, &policyStart, &policyEnd,
		&declarationPolicy, &policyCurrency, &creditPolicy, &insured, &branchCode,
		&lossDate, &reportDate, &receivedDate,
		&location, &chronology,
		&rName, &rPhone, &rEmail, &rAddress, &rRelation, &rRelationOther,
		&estimate, &currency, &slikNumber, &exGratia, &technicalPIC, &rcvID,
		&puclStatus, &complianceTransfer, &requestReturn,
		&processStatus, &claimStatus, &claimFlag, &positionStatus,
		&currentStage,
		&createdBy, &createdAt, &updatedBy, &updatedAt, &deletedAt,
		&flagNoll,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.Claim{}, registrasi.ErrClaimNotFound
	}
	if err != nil {
		return registrasi.Claim{}, fmt.Errorf("registrasi/sqlstore: membaca klaim: %w", err)
	}

	k.Number = number.String
	k.Portal = portal.String
	k.Policy.Line = registrasi.LineOfBusiness(line.String)
	k.Policy.BusinessType = businessType.String
	k.Policy.CoverageStart = policyStart.Time
	k.Policy.CoverageEnd = policyEnd.Time
	k.Policy.Declaration = fromYesNo(declarationPolicy.String)
	k.Policy.Currency = policyCurrency.String
	k.Policy.CreditGuarantee = fromYesNo(creditPolicy.String)
	k.Policy.InsuredName = insured.String
	k.Policy.BranchCode = branchCode.String

	k.DateOfLoss = lossDate.Time
	k.ReportDate = reportDate.Time
	k.DateReceived = receivedDate.Time
	k.Location = location.String
	k.Chronology = chronology.String

	k.Reporter = registrasi.Reporter{
		Name:          rName.String,
		Phone:         rPhone.String,
		Email:         rEmail.String,
		Address:       rAddress.String,
		Relation:      int(rRelation.Int64),
		OtherRelation: rRelationOther.String,
	}

	k.EstimateValue = registrasi.Money(estimate.Int64)
	k.Currency = currency.String
	k.SLIKNumber = slikNumber.String
	k.ExGratia = fromYesNo(exGratia.String)
	k.TechnicalPIC = technicalPIC.String
	k.RCVID = rcvID.String

	k.PUCLStatus = int(puclStatus.Int64)
	k.ComplianceTransfer = fromYesNo(complianceTransfer.String)
	k.RequestReturn = fromYesNo(requestReturn.String)

	k.ProcessStatus = registrasi.ProcessStatus(processStatus.String)
	k.ClaimStatus = registrasi.ClaimStatus(claimStatus.String)
	k.ClaimFlag = registrasi.ClaimFlag(claimFlag.String)
	k.ProgressPositionStatus = registrasi.ProgressPositionStatus(positionStatus.String)
	k.CurrentStage = currentStage.String

	k.CreatedBy = createdBy.String
	k.CreatedAt = createdAt.Time
	k.UpdatedBy = updatedBy.String
	k.UpdatedAt = updatedAt.Time
	if deletedAt.Valid {
		t := deletedAt.Time
		k.DeletedAt = &t
	}
	k.LargeLossNoticed = fromFlagNOLL(flagNoll.String)

	if err := r.loadTree(ctx, exec, &k); err != nil {
		return registrasi.Claim{}, err
	}
	return k, nil
}

// loadTree mengisi objek, coverage, dan spreading dalam TIGA kueri, bukan satu kueri
// per objek.
//
// Satu klaim kebakaran besar dapat memuat puluhan objek dengan coverage masing-masing;
// memuatnya satu per satu berarti puluhan perjalanan bolak-balik ke basis data untuk
// membuka satu layar.
func (r *ClaimStore) loadTree(ctx context.Context, exec executor, k *registrasi.Claim) error {
	itemIndex := map[int]int{}

	row, err := exec.QueryContext(ctx, loadQuery("objek_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca objek: %w", err)
	}
	defer func() { _ = row.Close() }()
	for row.Next() {
		var (
			seq            int
			itemID         string
			name, location sql.NullString
		)
		if err := row.Scan(&seq, &itemID, &name, &location); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris objek: %w", err)
		}
		itemIndex[seq] = len(k.InsuredItem)
		k.InsuredItem = append(k.InsuredItem, registrasi.InsuredItem{ID: itemID, Name: name.String, Location: location.String})
	}
	if err := row.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri objek: %w", err)
	}
	_ = row.Close()

	coverageIndex := map[[2]int]int{}

	coverageRow, err := exec.QueryContext(ctx, loadQuery("coverage_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca coverage: %w", err)
	}
	defer func() { _ = coverageRow.Close() }()
	for coverageRow.Next() {
		var (
			itemSeq, seq      int
			coverageID, cause sql.NullString
			tsi               sql.NullInt64
		)
		if err := coverageRow.Scan(&itemSeq, &seq, &coverageID, &cause, &tsi); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris coverage: %w", err)
		}
		i, ok := itemIndex[itemSeq]
		if !ok {
			// Coverage yang objek induknya sudah ditandai terhapus. Ia dilewati, bukan
			// dianggap galat: penandaan induk memang membuat anaknya tidak lagi
			// terlihat.
			continue
		}
		coverageIndex[[2]int{itemSeq, seq}] = len(k.InsuredItem[i].Coverage)
		k.InsuredItem[i].Coverage = append(k.InsuredItem[i].Coverage, registrasi.Coverage{
			ID:          coverageID.String,
			CauseOfLoss: cause.String,
			TSI:         registrasi.Money(tsi.Int64),
		})
	}
	if err := coverageRow.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri coverage: %w", err)
	}
	_ = coverageRow.Close()

	spreadingRow, err := exec.QueryContext(ctx, loadQuery("spreading_daftar"), k.ID)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca spreading: %w", err)
	}
	defer func() { _ = spreadingRow.Close() }()
	for spreadingRow.Next() {
		var (
			itemSeq, coverageSeq, seq     int
			kind, name, facOffer, removed sql.NullString
			share                         sql.NullInt64
		)
		if err := spreadingRow.Scan(&itemSeq, &coverageSeq, &seq, &kind, &name, &share, &removed, &facOffer); err != nil {
			return fmt.Errorf("registrasi/sqlstore: membaca baris spreading: %w", err)
		}
		i, ok := itemIndex[itemSeq]
		if !ok {
			continue
		}
		j, ok := coverageIndex[[2]int{itemSeq, coverageSeq}]
		if !ok {
			continue
		}
		k.InsuredItem[i].Coverage[j].Spreading = append(k.InsuredItem[i].Coverage[j].Spreading, registrasi.Spreading{
			TreatyKind:   kind.String,
			Name:         name.String,
			Share:        registrasi.Percent(share.Int64),
			Removed:      fromYesNo(removed.String),
			FacOfferItem: facOffer.String,
		})
	}
	if err := spreadingRow.Err(); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menelusuri spreading: %w", err)
	}
	return nil
}

// FindDuplicates mencari klaim lain yang memenuhi salah satu kunci duplikasi.
func (r *ClaimStore) FindDuplicates(ctx context.Context, key []registrasi.DuplicateKey, exceptID string) ([]registrasi.DuplicateClaim, error) {
	exec := executorFrom(ctx, r.db)

	var result []registrasi.DuplicateClaim
	seen := map[string]bool{}

	for _, dupKey := range key {
		useLocation := 0
		if dupKey.Location != "" {
			useLocation = 1
		}
		useCause := 0
		if dupKey.CauseOfLoss != "" {
			useCause = 1
		}

		row, err := exec.QueryContext(ctx, loadQuery("klaim_cari_ganda"),
			dupKey.PolicyNumber, exceptID, dupKey.InsuredItemID,
			useLocation, dupKey.Location,
			useCause, dupKey.CauseOfLoss,
		)
		if err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: memeriksa klaim ganda: %w", err)
		}

		for row.Next() {
			var number, insuredItem string
			if err := row.Scan(&number, &insuredItem); err != nil {
				_ = row.Close()
				return nil, fmt.Errorf("registrasi/sqlstore: membaca baris klaim ganda: %w", err)
			}
			marker := number + "|" + insuredItem
			if seen[marker] {
				continue
			}
			seen[marker] = true
			result = append(result, registrasi.DuplicateClaim{Number: number, InsuredItem: insuredItem})
		}
		if err := row.Err(); err != nil {
			_ = row.Close()
			return nil, fmt.Errorf("registrasi/sqlstore: menelusuri klaim ganda: %w", err)
		}
		_ = row.Close()
	}
	return result, nil
}

func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

func timePtrOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC()
}

// emptyTextAsNil menjaga kolom NOMOR tetap NULL selama klaim belum bernomor.
//
// Ini bukan kerapian: kolomnya berada di bawah UNIQUE, dan Oracle memperlakukan NULL
// sebagai "tidak diketahui" sehingga banyak baris boleh sama-sama NULL. Bila string
// kosong yang tersimpan, klaim kedua yang belum bernomor akan ditolak constraint.
func emptyTextAsNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

var _ registrasi.ClaimRepo = (*ClaimStore)(nil)
