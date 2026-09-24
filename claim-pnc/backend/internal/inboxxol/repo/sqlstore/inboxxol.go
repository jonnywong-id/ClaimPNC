package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxxol"
)

// DefaultBaseCurrencyID adalah kode mata uang pembagi pada perhitungan treaty inward.
//
// Nilainya dibaca dari `RDB List/GetDataTrytyInwardFromUploadData-SQL.xml`, yang memanggil
// `POOLDATA.GETCURRENCYSTANDARD('10001', …)` sebagai penyebut — sehingga hasilnya berada
// dalam mata uang berkode itu. Judul kolom di layar lama menyebutnya USD.
//
// # Ia BELUM SEHARUSNYA berupa konstanta, dan itu diakui di sini
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode; tempatnya adalah master
// Mata Uang & Kurs pada `F-4`, yang belum ada (`TKT-F4-004` masih terhalang, isi
// `m_currencystandard` belum diterima DBA). Sampai master itu ada, nilainya dapat ditimpa
// lewat NewRepoWithBaseCurrency tanpa menyunting kode — dan ketika master tiba, yang
// berubah hanya satu tempat.
const DefaultBaseCurrencyID = "10001"

// Repo membaca data Inbox XOL dari POOLDATA.
//
// Seluruh tabelnya MILIK SISTEM LAMA dan HANYA DIBACA di sini. Tidak ada satu pun
// operasi yang menulis — lihat kepala paket dan kepala inboxxol.sql.
type Repo struct {
	db *sql.DB

	// baseCurrencyID adalah penyebut konversi treaty inward. Lihat DefaultBaseCurrencyID.
	baseCurrencyID string
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo {
	return NewRepoWithBaseCurrency(db, DefaultBaseCurrencyID)
}

// NewRepoWithBaseCurrency membentuk repo dengan mata uang dasar yang ditentukan pemanggil.
//
// Dipakai saat master Mata Uang & Kurs (`F-4`) sudah ada, atau saat sebuah entitas memakai
// mata uang perjanjian yang berbeda. Nilai kosong jatuh ke DefaultBaseCurrencyID —
// bukan ke teks kosong, yang akan membuat setiap baris treaty inward kehilangan kursnya
// tanpa sebab yang terbaca.
func NewRepoWithBaseCurrency(db *sql.DB, baseCurrencyID string) *Repo {
	clean := strings.TrimSpace(baseCurrencyID)
	if clean == "" {
		clean = DefaultBaseCurrencyID
	}
	return &Repo{db: db, baseCurrencyID: clean}
}

// ListMasterXOL mengembalikan seluruh perjanjian XOL beserta group business-nya.
//
// Dua kueri, bukan satu dengan join: group business adalah satu-ke-banyak, dan
// menggabungkannya dalam satu kueri berarti mengulang seluruh kolom master sebanyak
// group business-nya lalu menyatukannya kembali di Go. Dua kueri yang masing-masing
// dibaca sekali lebih murah dan jauh lebih mudah diperiksa.
func (r *Repo) ListMasterXOL(ctx context.Context) ([]inboxxol.MasterXOL, error) {
	return r.listMasters(ctx, "master_list")
}

// ListPendingMasterApproval mengembalikan perjanjian yang menunggu persetujuan komite.
func (r *Repo) ListPendingMasterApproval(ctx context.Context) ([]inboxxol.MasterXOL, error) {
	return r.listMasters(ctx, "master_pending_committee")
}

// listMasters menjalankan salah satu kueri master lalu melekatkan group business-nya.
func (r *Repo) listMasters(ctx context.Context, name string) ([]inboxxol.MasterXOL, error) {
	rows, err := r.db.QueryContext(ctx, query(name))
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	masters := make([]inboxxol.MasterXOL, 0, 32)
	for rows.Next() {
		master, err := scanMaster(rows)
		if err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
		}
		masters = append(masters, master)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
	}

	groups, err := r.businessGroups(ctx)
	if err != nil {
		return nil, err
	}
	for i := range masters {
		masters[i].BusinessGroups = groups[strings.TrimSpace(masters[i].ID)]
	}
	return masters, nil
}

// businessGroups membaca seluruh group business, dikelompokkan menurut perjanjian.
//
// Dibaca sekali untuk seluruh perjanjian, bukan sekali per perjanjian: jumlah barisnya
// sekitar sebanyak perjanjian × group business, yaitu ratusan — bukan puluhan juta seperti
// tabel klaim. Satu kueri per perjanjian akan menghasilkan pola N+1 yang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 larang.
func (r *Repo) businessGroups(ctx context.Context) (map[string][]inboxxol.BusinessGroup, error) {
	rows, err := r.db.QueryContext(ctx, query("master_business_list"))
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: master_business_list: %w", err)
	}
	defer rows.Close()

	result := map[string][]inboxxol.BusinessGroup{}
	for rows.Next() {
		var masterID, groupID, groupName sql.NullString
		if err := rows.Scan(&masterID, &groupID, &groupName); err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: master_business_list: %w", err)
		}
		key := strings.TrimSpace(masterID.String)
		result[key] = append(result[key], inboxxol.BusinessGroup{
			ID:   strings.TrimSpace(groupID.String),
			Name: strings.TrimSpace(groupName.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: master_business_list: %w", err)
	}
	return result, nil
}

// SummarizeClaims mengembalikan akumulasi klaim per Tanggal Kejadian dan Penyebab
// Kerugian. Nilainya masih RUPIAH.
func (r *Repo) SummarizeClaims(ctx context.Context, filter inboxxol.ClaimFilter) ([]inboxxol.ClaimSummary, error) {
	if filter.Empty() {
		// Dijawab daftar kosong, bukan galat: penyaring kosong berarti perjanjiannya
		// memang tidak menanggung group business mana pun. Menjalankan kuerinya akan
		// menghasilkan `IN ()` yang tidak sah.
		return nil, nil
	}

	text, arguments, err := expandIDs("claim_summary",
		[]any{strings.TrimSpace(filter.Year)}, filter.BusinessGroupIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, text, arguments...)
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: claim_summary: %w", err)
	}
	defer rows.Close()

	result := make([]inboxxol.ClaimSummary, 0, 64)
	for rows.Next() {
		var (
			lossDate    sql.NullString
			causeOfLoss sql.NullString
			outstanding sql.NullFloat64
			accepted    sql.NullFloat64
		)
		if err := rows.Scan(&lossDate, &causeOfLoss, &outstanding, &accepted); err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: claim_summary: %w", err)
		}
		result = append(result, inboxxol.ClaimSummary{
			LossDate:         strings.TrimSpace(lossDate.String),
			CauseOfLoss:      strings.TrimSpace(causeOfLoss.String),
			OutstandingValue: outstanding.Float64,
			AcceptedValue:    accepted.Float64,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: claim_summary: %w", err)
	}
	return result, nil
}

// BreakdownByBusiness mengembalikan rincian klaim milik sendiri per group business.
// Nilainya masih rupiah.
func (r *Repo) BreakdownByBusiness(ctx context.Context, filter inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	if filter.Empty() {
		return nil, nil
	}

	text, arguments, err := expandIDs("breakdown_business",
		[]any{strings.TrimSpace(filter.LossDate), strings.TrimSpace(filter.CauseOfLoss)},
		filter.BusinessGroupIDs)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, text, arguments...)
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: breakdown_business: %w", err)
	}
	defer rows.Close()

	result := make([]inboxxol.BusinessBreakdown, 0, 16)
	for rows.Next() {
		var (
			groupName   sql.NullString
			groupID     sql.NullString
			claimCount  sql.NullInt64
			outstanding sql.NullFloat64
			accepted    sql.NullFloat64
		)
		if err := rows.Scan(&groupName, &groupID, &claimCount, &outstanding, &accepted); err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: breakdown_business: %w", err)
		}
		result = append(result, inboxxol.BusinessBreakdown{
			// Nama yang kosong digantikan penanda yang sama dengan yang dipakai
			// GET_GROUPBUSINESS_XOL — di Go, bukan di basis data.
			BusinessGroup:    inboxxol.BusinessGroup{Name: strings.TrimSpace(groupName.String)}.DisplayName(),
			BusinessGroupID:  strings.TrimSpace(groupID.String),
			ClaimCount:       int(claimCount.Int64),
			OutstandingValue: outstanding.Float64,
			AcceptedValue:    accepted.Float64,
			Source:           inboxxol.SourceOwnBusiness,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: breakdown_business: %w", err)
	}
	return result, nil
}

// BreakdownTreatyInward mengembalikan rincian klaim treaty inward.
//
// Nilainya SUDAH dikonversi ke mata uang perjanjian di dalam kueri — lihat alasannya di
// inboxxol.sql. Baris yang seluruh isinya nol DAN tidak menandai kurs hilang tidak
// dikembalikan: kueri agregat tanpa GROUP BY selalu menghasilkan satu baris, bahkan
// ketika tidak ada satu pun klaim treaty inward pada tanggal itu, dan baris nol yang
// tampil di grid terbaca sebagai "ada, nilainya nol" alih-alih "tidak ada".
func (r *Repo) BreakdownTreatyInward(ctx context.Context, filter inboxxol.BreakdownFilter) ([]inboxxol.BusinessBreakdown, error) {
	if filter.TreatyEmpty() {
		return nil, nil
	}

	var (
		groupName   sql.NullString
		claimCount  sql.NullInt64
		outstanding sql.NullFloat64
		accepted    sql.NullFloat64
		rateMissing sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, query("breakdown_treaty_inward"),
		strings.TrimSpace(filter.LossDate),
		strings.TrimSpace(filter.CauseOfLoss),
		r.baseCurrencyID,
	).Scan(&groupName, &claimCount, &outstanding, &accepted, &rateMissing)

	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("inboxxol/sqlstore: breakdown_treaty_inward: %w", err)
	}

	missing := rateMissing.Int64 == 1
	if claimCount.Int64 == 0 && !missing {
		return nil, nil
	}

	return []inboxxol.BusinessBreakdown{{
		BusinessGroup:    strings.TrimSpace(groupName.String),
		ClaimCount:       int(claimCount.Int64),
		OutstandingValue: outstanding.Float64,
		AcceptedValue:    accepted.Float64,
		Source:           inboxxol.SourceTreatyInward,
		RateMissing:      missing,
	}}, nil
}

// adviceQueryByType memetakan tipe pemberitahuan ke nama kueri di inboxxol.sql.
//
// Ia map, bukan perangkaian nama tabel. Tipe yang tidak dikenal gagal dengan pesan yang
// menyebutnya — bukan diam-diam membaca tabel yang salah.
var adviceQueryByType = map[inboxxol.AdviceType]string{
	inboxxol.AdvicePLA: "advice_list_pla",
	inboxxol.AdviceDLA: "advice_list_dla",
}

// SearchAdvice mengembalikan pemberitahuan PLA atau DLA yang sudah diterbitkan.
func (r *Repo) SearchAdvice(ctx context.Context, filter inboxxol.AdviceFilter) ([]inboxxol.Advice, error) {
	name, known := adviceQueryByType[filter.Type]
	if !known {
		return nil, fmt.Errorf("inboxxol/sqlstore: tipe pemberitahuan %q tidak dikenal", filter.Type)
	}

	rows, err := r.db.QueryContext(ctx, query(name),
		strings.TrimSpace(filter.Year), strings.TrimSpace(filter.CauseOfLoss))
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	result := make([]inboxxol.Advice, 0, 32)
	for rows.Next() {
		advice, err := scanAdvice(rows)
		if err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
		}
		advice.Type = filter.Type
		result = append(result, advice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: %s: %w", name, err)
	}
	return result, nil
}

// ListPendingAdviceApproval mengembalikan antrean persetujuan pemberitahuan.
func (r *Repo) ListPendingAdviceApproval(ctx context.Context) ([]inboxxol.ApprovalItem, error) {
	rows, err := r.db.QueryContext(ctx, query("approval_advice_queue"))
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: approval_advice_queue: %w", err)
	}
	defer rows.Close()

	result := make([]inboxxol.ApprovalItem, 0, 32)
	for rows.Next() {
		var year, causeOfLoss, adviceType, lastInserted sql.NullString
		if err := rows.Scan(&year, &causeOfLoss, &adviceType, &lastInserted); err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: approval_advice_queue: %w", err)
		}
		// Tipe berasal dari kolom literal 'PLA'/'DLA' di dalam kueri, bukan dari
		// masukan siapa pun; nilai yang tidak dikenal dibiarkan apa adanya supaya
		// datanya terlihat, bukan disembunyikan.
		parsed, _ := inboxxol.ParseAdviceType(adviceType.String)
		result = append(result, inboxxol.ApprovalItem{
			Year:           strings.TrimSpace(year.String),
			CauseOfLoss:    strings.TrimSpace(causeOfLoss.String),
			Type:           parsed,
			LastInsertedAt: strings.TrimSpace(lastInserted.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: approval_advice_queue: %w", err)
	}
	return result, nil
}

// ListCauseOfLoss mengembalikan daftar Penyebab Kerugian.
func (r *Repo) ListCauseOfLoss(ctx context.Context) ([]inboxxol.CauseOfLoss, error) {
	rows, err := r.db.QueryContext(ctx, query("cause_of_loss_list"))
	if err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: cause_of_loss_list: %w", err)
	}
	defer rows.Close()

	result := make([]inboxxol.CauseOfLoss, 0, 64)
	for rows.Next() {
		var id, description sql.NullString
		if err := rows.Scan(&id, &description); err != nil {
			return nil, fmt.Errorf("inboxxol/sqlstore: cause_of_loss_list: %w", err)
		}
		result = append(result, inboxxol.CauseOfLoss{
			ID:          strings.TrimSpace(id.String),
			Description: strings.TrimSpace(description.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxxol/sqlstore: cause_of_loss_list: %w", err)
	}
	return result, nil
}

// scanner adalah apa pun yang dapat memindai satu baris — *sql.Row maupun *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

// scanMaster memindai satu baris perjanjian XOL.
//
// Urutannya WAJIB sama dengan urutan kolom pada master_list DAN master_pending_committee.
// Kesamaan kedua kueri itu diuji di query_test.go — salah satu yang berubah sendirian
// tidak menghasilkan galat kompilasi, hanya nilai yang tertukar diam-diam.
func scanMaster(row scanner) (inboxxol.MasterXOL, error) {
	var (
		id              sql.NullString
		name            sql.NullString
		year            sql.NullString
		exchangeRate    sql.NullFloat64
		masterType      sql.NullString
		committeeStatus sql.NullString
		committeeNote   sql.NullString
		pic             sql.NullString
		picNote         sql.NullString
		picEmail        sql.NullString
	)
	if err := row.Scan(&id, &name, &year, &exchangeRate, &masterType,
		&committeeStatus, &committeeNote, &pic, &picNote, &picEmail); err != nil {
		return inboxxol.MasterXOL{}, err
	}
	return inboxxol.MasterXOL{
		ID:              strings.TrimSpace(id.String),
		Name:            strings.TrimSpace(name.String),
		Year:            strings.TrimSpace(year.String),
		ExchangeRate:    exchangeRate.Float64,
		Type:            strings.TrimSpace(masterType.String),
		CommitteeStatus: strings.TrimSpace(committeeStatus.String),
		CommitteeNote:   strings.TrimSpace(committeeNote.String),
		PIC:             strings.TrimSpace(pic.String),
		PICNote:         strings.TrimSpace(picNote.String),
		PICEmail:        strings.TrimSpace(picEmail.String),
	}, nil
}

// scanAdvice memindai satu baris pemberitahuan PLA/DLA.
//
// Urutannya WAJIB sama dengan urutan kolom pada advice_list_pla DAN advice_list_dla.
func scanAdvice(row scanner) (inboxxol.Advice, error) {
	var (
		number         sql.NullString
		revision       sql.NullString
		reinsurerID    sql.NullString
		reinsurerName  sql.NullString
		layerID        sql.NullString
		layerName      sql.NullString
		year           sql.NullString
		causeOfLoss    sql.NullString
		exchangeRate   sql.NullFloat64
		sharePercent   sql.NullFloat64
		layerLimit     sql.NullString
		approvalStatus sql.NullString
		masterID       sql.NullString
		inputBy        sql.NullString
		reinsurerNote  sql.NullString
		approvalNote   sql.NullString
		picNote        sql.NullString
		email          sql.NullString
		country        sql.NullString
		inputByEmail   sql.NullString
		issuedOn       sql.NullString
	)
	if err := row.Scan(&number, &revision, &reinsurerID, &reinsurerName, &layerID, &layerName,
		&year, &causeOfLoss, &exchangeRate, &sharePercent, &layerLimit, &approvalStatus,
		&masterID, &inputBy, &reinsurerNote, &approvalNote, &picNote,
		&email, &country, &inputByEmail, &issuedOn); err != nil {
		return inboxxol.Advice{}, err
	}
	return inboxxol.Advice{
		Number:         strings.TrimSpace(number.String),
		Revision:       strings.TrimSpace(revision.String),
		ReinsurerID:    strings.TrimSpace(reinsurerID.String),
		ReinsurerName:  strings.TrimSpace(reinsurerName.String),
		LayerID:        strings.TrimSpace(layerID.String),
		LayerName:      strings.TrimSpace(layerName.String),
		Year:           strings.TrimSpace(year.String),
		CauseOfLoss:    strings.TrimSpace(causeOfLoss.String),
		ExchangeRate:   exchangeRate.Float64,
		SharePercent:   sharePercent.Float64,
		Limit:          strings.TrimSpace(layerLimit.String),
		ApprovalStatus: strings.TrimSpace(approvalStatus.String),
		MasterID:       strings.TrimSpace(masterID.String),
		InputBy:        strings.TrimSpace(inputBy.String),
		Remark:         strings.TrimSpace(reinsurerNote.String),
		ApprovalNote:   strings.TrimSpace(approvalNote.String),
		PICNote:        strings.TrimSpace(picNote.String),
		Email:          strings.TrimSpace(email.String),
		Country:        strings.TrimSpace(country.String),
		InputByEmail:   strings.TrimSpace(inputByEmail.String),
		IssuedOn:       strings.TrimSpace(issuedOn.String),
	}, nil
}
