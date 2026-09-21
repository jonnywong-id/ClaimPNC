package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"claim-pnc/internal/riwayatklaim"
)

// Repo membaca riwayat klaim dari POOLDATA.T_CLAIM_PNC dan kerabatnya.
//
// Seluruhnya MILIK SISTEM LAMA dan HANYA DIBACA di sini. Layar View History Claim tidak
// mengubah satu baris pun — `P-1` karena itu terpenuhi tanpa negosiasi kepemilikan: Pega
// tetap satu-satunya yang menulis tabelnya sendiri.
//
// Pemetaan alias menyesatkan, keempat perubahan terhadap kueri lama, dan alasan
// masing-masing ada di kepala riwayatklaim.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// queryByType memetakan kode tipe pencarian ke nama kueri di riwayatklaim.sql.
//
// Ia map, bukan rangkaian if: menambah tipe pencarian berarti menambah satu baris di sini
// dan satu blok di berkas .sql, dan tipe yang lupa dipetakan gagal dengan pesan yang
// menyebut kodenya — bukan diam-diam jatuh ke kueri tipe lain.
//
// Tipe 12 "No Rekening" TIDAK ADA di sini, dan ketiadaannya disengaja: kueri lamanya
// menembus DB Link `gl.t_all_payment@asmd` yang belum punya API pengganti (`R-03`).
// riwayatklaim.SearchType menandainya belum tersedia, sehingga permintaannya sudah
// ditolak di lapisan domain jauh sebelum sampai ke sini.
var queryByType = map[string]string{
	riwayatklaim.TypePolicyNumber:     "search_policy_number",
	riwayatklaim.TypeInsuredName:      "search_insured_name",
	riwayatklaim.TypeInsuredItemName:  "search_insured_item_name",
	riwayatklaim.TypePLANumber:        "search_pla_number",
	riwayatklaim.TypeDLANumber:        "search_dla_number",
	riwayatklaim.TypeLossDate:         "search_loss_date",
	riwayatklaim.TypeClaimNumber:      "search_claim_number",
	riwayatklaim.TypeAcceptanceNumber: "search_acceptance_number",
	riwayatklaim.TypeBirthDate:        "search_birth_date",
	riwayatklaim.TypeSurveyNumber:     "search_survey_number",
	riwayatklaim.TypeAuctionHouseID:   "search_auction_house_id",
}

// Search menjalankan kueri milik satu tipe pencarian.
func (r *Repo) Search(
	ctx context.Context,
	criteria riwayatklaim.Criteria,
	page riwayatklaim.Pagination,
) (riwayatklaim.Page, error) {
	name, known := queryByType[criteria.Type.Code]
	if !known {
		return riwayatklaim.Page{}, fmt.Errorf(
			"riwayatklaim/sqlstore: tipe pencarian %q belum punya kueri", criteria.Type.Code)
	}

	page = page.Normalize()
	value := bindValue(criteria)

	var total int
	if err := r.db.QueryRowContext(ctx, counted(name), value).Scan(&total); err != nil {
		return riwayatklaim.Page{}, fmt.Errorf("menghitung hasil pencarian: %w", err)
	}

	// Halaman yang seluruhnya berada di luar hasil tidak perlu menembak basis data lagi.
	// Ia bukan galat — pengguna dapat sampai ke sana dengan mengubah alamat, atau dengan
	// berada di halaman 5 saat data berkurang.
	if total == 0 || page.Offset() >= total {
		return riwayatklaim.Page{
			Claims:     []riwayatklaim.ClaimHistory{},
			Total:      total,
			Pagination: page,
		}, nil
	}

	rows, err := r.db.QueryContext(ctx, paged(name), value, page.Offset(), page.Size)
	if err != nil {
		return riwayatklaim.Page{}, fmt.Errorf("membaca hasil pencarian: %w", err)
	}
	defer rows.Close()

	list := make([]riwayatklaim.ClaimHistory, 0, page.Size)
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return riwayatklaim.Page{}, err
		}
		list = append(list, claim)
	}
	if err := rows.Err(); err != nil {
		return riwayatklaim.Page{}, fmt.Errorf("membaca hasil pencarian: %w", err)
	}

	return riwayatklaim.Page{Claims: list, Total: total, Pagination: page}, nil
}

// bindValue menyiapkan satu nilai yang dikirim sebagai parameter `:1`.
//
// Kesebelas kueri pencarian masing-masing memakai TEPAT SATU parameter, dan keseragaman
// itu dijaga uji di query_test.go — kueri yang memakai dua akan membuat offset dan ukuran
// halaman bergeser ke posisi yang salah.
//
// Nilainya dapat berupa teks atau tanggal, dan mana yang dipakai ditentukan
// riwayatklaim.Criteria.QueryValue — termasuk cacat tipe Tanggal Lahir yang direplikasi
// sadar. Tanggal yang kosong dikirim sebagai NULL, dan perbandingan dengan NULL
// menghasilkan UNKNOWN sehingga hasilnya kosong tanpa galat — persis sistem lama.
func bindValue(criteria riwayatklaim.Criteria) any {
	text, date, isDate := criteria.QueryValue()
	if !isDate {
		return text
	}
	if date == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *date, Valid: true}
}

// scanner menyatukan *sql.Row dan *sql.Rows sehingga pemindainya cukup satu.
type scanner interface {
	Scan(dest ...any) error
}

// scanClaim memindai keenam belas kolom menjadi satu baris riwayat klaim.
//
// Urutannya WAJIB sama dengan urutan kolom pada setiap kueri di riwayatklaim.sql. Karena
// seluruh kueri mengembalikan keenam belas kolom yang sama — sebagian NULL — satu
// pemindai melayani kesebelasnya.
func scanClaim(row scanner) (riwayatklaim.ClaimHistory, error) {
	var (
		reference       sql.NullString
		number          sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		lossDate        sql.NullTime
		businessName    sql.NullString
		branchName      sql.NullString
		workStatus      sql.NullString
		claimPosition   sql.NullString
		closeDate       sql.NullTime
		closeNote       sql.NullString
		technicalPIC    sql.NullString
		acceptanceNo    sql.NullString
		auctionHouseID  sql.NullString
		insuredItemName sql.NullString
		birthDate       sql.NullTime
	)

	if err := row.Scan(
		&reference, &number, &policyNumber, &insuredName, &lossDate,
		&businessName, &branchName, &workStatus, &claimPosition,
		&closeDate, &closeNote, &technicalPIC,
		&acceptanceNo, &auctionHouseID, &insuredItemName, &birthDate,
	); err != nil {
		return riwayatklaim.ClaimHistory{}, fmt.Errorf("memindai baris riwayat klaim: %w", err)
	}

	return riwayatklaim.ClaimHistory{
		Reference:        reference.String,
		Number:           number.String,
		PolicyNumber:     policyNumber.String,
		InsuredName:      insuredName.String,
		LossDate:         nullableTime(lossDate),
		BusinessName:     businessName.String,
		BranchName:       branchName.String,
		WorkStatus:       workStatus.String,
		ClaimPosition:    claimPosition.String,
		CloseDate:        nullableTime(closeDate),
		CloseNote:        closeNote.String,
		TechnicalPIC:     technicalPIC.String,
		AcceptanceNumber: acceptanceNo.String,
		AuctionHouseID:   auctionHouseID.String,
		InsuredItemName:  insuredItemName.String,
		BirthDate:        nullableTime(birthDate),
	}, nil
}

// nullableTime mengubah kolom tanggal yang dapat kosong menjadi pointer.
//
// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari "belum
// diisi" saat ditampilkan, dan layar akan menuliskan "01/01/0001" alih-alih tanda hubung.
func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	moment := value.Time
	return &moment
}
