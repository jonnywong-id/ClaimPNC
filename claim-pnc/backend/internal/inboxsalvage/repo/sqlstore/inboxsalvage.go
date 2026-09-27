package sqlstore

import (
	"context"
	"database/sql"
	"errors"

	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/inboxsalvage"
)

// Repo membaca dan menulis data salvage pada SATU basis data entitas.
//
// # Ia MENULIS, berbeda dari sqlstore modul inbox lain
//
// Dua tabel — `POOLDATA.PNC_SALVAGE` dan `POOLDATA.DETAIL_PNC_SALVAGE` — dimiliki modul
// ini selama masa paralel, sehingga `P-1` terpenuhi. Seluruh tabel lain hanya dibaca.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk penyimpanan di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// PegaWorkKeyPrefix adalah awalan kunci objek kerja Pega pada kolom `T_CLAIM_PNC.CLAIMID`.
//
// Nama kelas internal Pega tertanam di dalam kunci data bisnis — utang teknis §4.1 yang
// `D-22` dan `D-71` hapus untuk klaim baru. Ia diikat sebagai nilai, bukan ditulis di dalam
// SQL, supaya terbaca sebagai sesuatu yang kelak berubah.
//
// Perhatikan SPASI di ujungnya: ia bagian dari nilainya, dan menghilangkannya membuat
// gabungan tidak pernah cocok — daftar kosong tanpa satu pun galat.
const PegaWorkKeyPrefix = "ASM-FW-GCNMFW-WORK "

// List mengembalikan satu halaman baris yang cocok beserta jumlah seluruhnya.
func (r *Repo) List(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	if q.Tab.Family == inboxsalvage.FamilySalvage {
		return r.listSalvage(ctx, q, page)
	}
	return r.listClaim(ctx, q, page)
}

// listClaim melayani keluarga A dan B.
func (r *Repo) listClaim(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	clean := page.Normalize()
	searching := q.Search != ""

	name, args := claimPlan(q, clean, searching)

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxsalvage.Page{}, fmt.Errorf("inboxsalvage/sqlstore: %s: %w", name, err)
	}
	defer rows.Close()

	result := inboxsalvage.Page{Pagination: clean, Items: []inboxsalvage.Row{}}
	for rows.Next() {
		var (
			claimNo, pic, business, fourth sql.NullString
			total                          int
		)
		if err := rows.Scan(&claimNo, &pic, &business, &fourth, &total); err != nil {
			return inboxsalvage.Page{}, fmt.Errorf(
				"inboxsalvage/sqlstore: %s: memindai baris: %w", name, err)
		}

		row := inboxsalvage.Row{
			Reference:    claimNo.String,
			ClaimNo:      claimNo.String,
			PIC:          pic.String,
			BusinessName: business.String,
		}

		// Kolom keempat berarti hal yang BERBEDA menurut keluarganya, dan itu satu-satunya
		// tempat kedua kueri berbeda bentuknya.
		if q.Tab.Family == inboxsalvage.FamilyClaim {
			row.LossDate = fourth.String
		} else {
			row.ObjectName = fourth.String
		}

		result.Items = append(result.Items, row)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: %s: membaca hasil: %w", name, err)
	}

	return result, nil
}

// claimPlan memilih kueri keluarga A/B beserta susunan bind-nya.
//
// # Kenapa nama kueri dan susunan bind dipilih BERSAMAAN
//
// Karena keduanya harus berpindah bersama. Memisahkannya — peta kode tab ke nama kueri di
// satu tempat, susunan bind di tempat lain — membuat penambahan satu bind pada satu kueri
// dapat lupa diikuti di tempat yang lain, dan akibatnya adalah galat bind yang menyebut
// nomor, bukan menyebut tab mana yang rusak.
func claimPlan(
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
	searching bool,
) (string, []any) {
	pattern := "%" + escapeLike(q.Search) + "%"

	switch {
	case q.Tab.BuybackFilter && searching:
		return "list_claim_buyback_search", []any{pattern, page.Offset(), page.Size}

	case q.Tab.BuybackFilter:
		return "list_claim_buyback", []any{page.Offset(), page.Size}

	case q.Tab.SalvageStatusIsNull && searching:
		return "list_claim_search", []any{
			inboxsalvage.ClosedWorkStatuses[0],
			inboxsalvage.ClosedWorkStatuses[1],
			pattern, page.Offset(), page.Size,
		}

	case q.Tab.SalvageStatusIsNull:
		return "list_claim", []any{
			inboxsalvage.ClosedWorkStatuses[0],
			inboxsalvage.ClosedWorkStatuses[1],
			page.Offset(), page.Size,
		}

	case searching:
		return "list_claim_object_search", []any{
			q.Tab.SalvageStatus, pattern, page.Offset(), page.Size,
		}

	default:
		return "list_claim_object", []any{
			q.Tab.SalvageStatus, page.Offset(), page.Size,
		}
	}
}

// listSalvage melayani keluarga C — ketujuh daftar, satu kueri.
func (r *Repo) listSalvage(
	ctx context.Context,
	q inboxsalvage.Query,
	page inboxsalvage.Pagination,
) (inboxsalvage.Page, error) {
	clean := page.Normalize()

	// Penyaring status transfer. Tab Histori Salvage tidak menyaringnya sama sekali.
	transfer := patternAll
	if q.Tab.TransferStatus != "" {
		transfer = escapeLike(q.Tab.TransferStatus)
	}

	// Penyaring pemilik. Hanya tab Request Balai Lelang memakainya.
	owner := patternAll
	if q.Tab.OwnedByCaller {
		owner = escapeLike(q.Caller.Login)
	}

	claimPattern, picPattern := searchPatterns(q)

	args := []any{
		PegaWorkKeyPrefix,
		transfer,
		owner,
		claimPattern,
		picPattern,
		clean.Offset(),
		clean.Size,
	}

	rows, err := r.db.QueryContext(ctx, query("list_salvage"), args...)
	if err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: list_salvage: %w", err)
	}
	defer rows.Close()

	today, err := r.today(ctx)
	if err != nil {
		return inboxsalvage.Page{}, err
	}

	result := inboxsalvage.Page{Pagination: clean, Items: []inboxsalvage.Row{}}
	for rows.Next() {
		var (
			salvageID, claimNo, inputDate, pic, salvageType sql.NullString
			location, quantity, estimate, email, remark     sql.NullString
			acceptanceNo, transferStatus, acceptedValue     sql.NullString
			requestValue, requestNote                       sql.NullString
			total                                           int
		)
		if err := rows.Scan(
			&salvageID, &claimNo, &inputDate, &pic, &salvageType,
			&location, &quantity, &estimate, &email, &remark,
			&acceptanceNo, &transferStatus, &acceptedValue,
			&requestValue, &requestNote,
			&total,
		); err != nil {
			return inboxsalvage.Page{}, fmt.Errorf(
				"inboxsalvage/sqlstore: list_salvage: memindai baris: %w", err)
		}

		input := dateOnly(inputDate.String)

		result.Items = append(result.Items, inboxsalvage.Row{
			Reference:       salvageID.String,
			ClaimNo:         claimNo.String,
			SalvageID:       salvageID.String,
			InputDate:       input,
			PIC:             pic.String,
			SalvageType:     salvageType.String,
			SalvageLocation: location.String,
			AuctionStatus:   inboxsalvage.AuctionStatusOf(acceptedValue.String),
			Quantity:        quantity.String,
			EstimateValue:   estimate.String,
			Email:           email.String,
			Remark:          remark.String,
			AcceptanceNo:    acceptanceNo.String,
			TransferStatus:  transferStatus.String,
			RequestValue:    requestValue.String,
			RequestNote:     requestNote.String,
			SubmissionType:  inboxsalvage.SubmissionTypeOf(requestNote.String),
			Aging:           inboxsalvage.AgingOf(input, today),

			// Kolom "Catatan" SELALU kosong — lihat inboxsalvage.Row.Note.
			Note: "",
		})
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxsalvage.Page{}, fmt.Errorf(
			"inboxsalvage/sqlstore: list_salvage: membaca hasil: %w", err)
	}

	return result, nil
}

// searchPatterns menyusun kedua pola pencarian keluarga C.
//
// Keduanya digabung dengan ATAU di dalam kueri, sehingga tab yang hanya mencari nomor klaim
// harus MEMATIKAN cabang PIC — bukan membiarkannya `%`, yang akan mencocokkan segalanya dan
// membuat pencarian tidak menyaring apa pun.
func searchPatterns(q inboxsalvage.Query) (string, string) {
	if q.Search == "" {
		return patternAll, patternNever
	}

	escaped := escapeLike(q.Search)

	if q.Tab.SearchExact {
		// Tanpa `%` di kedua ujung, `LIKE` berperilaku sama dengan `=` — dan itulah yang
		// dipakai kueri lama pada ketiga tab ini.
		if q.Tab.SearchByPIC {
			return escaped, escaped
		}
		return escaped, patternNever
	}

	pattern := "%" + escaped + "%"
	if q.Tab.SearchByPIC {
		return pattern, pattern
	}
	return pattern, patternNever
}

// Detail mengembalikan isi panel "Detail Salvage" untuk satu pengajuan.
//
// DUA kueri, bukan satu dengan gabungan: kepala dan daftar barangnya punya kardinalitas
// yang berbeda, dan menggabungkannya akan mengulang seluruh isian kepala pada setiap baris
// barang. Keduanya tetap satu perjalanan pulang-pergi masing-masing — bukan satu per baris
// seperti pada jalur checker di sistem lama.
func (r *Repo) Detail(ctx context.Context, salvageID string) (inboxsalvage.Detail, error) {
	clean := strings.TrimSpace(salvageID)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	detail, err := r.detailHeader(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	items, err := r.detailItems(ctx, detail.ClaimNo, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	detail.Items = items

	history, err := r.historyOfClaim(ctx, detail.ClaimNo)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	detail.History = history

	return detail, nil
}

// historyOfClaim membaca grid "Detail History Salvage" satu klaim.
func (r *Repo) historyOfClaim(
	ctx context.Context,
	claimNo string,
) ([]inboxsalvage.HistoryRow, error) {
	rows, err := r.db.QueryContext(ctx, query("salvage_history_of_claim"), claimNo)
	if err != nil {
		return nil, fmt.Errorf("membaca riwayat salvage klaim: %w", err)
	}
	defer rows.Close()

	history := []inboxsalvage.HistoryRow{}
	for rows.Next() {
		var (
			inputDate, no, pic, minimum sql.NullString
			transferStatus, hasAccNo    sql.NullString
			salvageID                   sql.NullString
		)

		if err := rows.Scan(&inputDate, &no, &pic, &minimum,
			&transferStatus, &hasAccNo, &salvageID); err != nil {
			return nil, fmt.Errorf("memindai riwayat salvage: %w", err)
		}

		history = append(history, inboxsalvage.HistoryRow{
			SalvageID:    strings.TrimSpace(salvageID.String),
			InputDate:    dateOnly(inputDate.String),
			ClaimNo:      strings.TrimSpace(no.String),
			PIC:          strings.TrimSpace(pic.String),
			MinimumValue: strings.TrimSpace(minimum.String),
			Position: inboxsalvage.HistoryPositionOf(
				transferStatus.String, flagIsSet(hasAccNo.String)),
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca riwayat salvage: %w", err)
	}

	return history, nil
}

// DetailByClaim membaca rincian lewat nomor klaim.
//
// Tiga langkah, dan urutannya menentukan pesan yang dilihat pengguna:
//
//  1. Klaimnya dibaca. Tidak ada -> ErrRowNotFound; nomor klaim yang salah memang
//     kekeliruan, dan pada aplikasi ini penyebab tersering bukan salah ketik melainkan
//     klaim milik entitas lain (`R-20`).
//  2. Pengajuan TERAKHIR miliknya dicari. Tidak ada -> panel tetap dikembalikan, dengan
//     isian klaim terisi dan HasSubmission salah. Itu keadaan yang sah, bukan kegagalan:
//     daftar Salvage Outstanding justru berisi klaim yang salvage-nya belum diajukan.
//  3. Kepala panel dan barangnya dibaca dengan ID itu.
func (r *Repo) DetailByClaim(
	ctx context.Context,
	claimNo string,
) (inboxsalvage.Detail, error) {
	clean := strings.TrimSpace(claimNo)
	if clean == "" {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}

	base, err := r.claimHeader(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	// Riwayat dibaca SEBELUM pengajuan terakhirnya, karena ia dibutuhkan pada KEDUA
	// cabang di bawah — termasuk cabang klaim yang belum punya pengajuan sama sekali,
	// tempat form "Menambahkan Data Salvage" menggambarnya.
	history, err := r.historyOfClaim(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	base.History = history

	salvageID, err := r.latestSalvageOfClaim(ctx, clean)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}
	if salvageID == "" {
		return base, nil
	}

	detail, err := r.detailHeader(ctx, salvageID)
	if err != nil {
		// Pengajuan yang tercatat di DETAIL_PNC_SALVAGE tetapi TIDAK ada di PNC_SALVAGE
		// adalah data yang tidak sinkron, bukan kekeliruan pengguna. Panel tetap
		// menampilkan klaimnya, dengan keterangan bahwa pengajuannya tidak terbaca —
		// alih-alih menolak membuka seluruh panel.
		if errors.Is(err, inboxsalvage.ErrRowNotFound) {
			return base, nil
		}
		return inboxsalvage.Detail{}, err
	}

	items, err := r.detailItems(ctx, detail.ClaimNo, salvageID)
	if err != nil {
		return inboxsalvage.Detail{}, err
	}

	// Isian klaim dipertahankan: kepala panel pengajuan tidak memuat PIC maupun tanggal
	// kejadian, dan keduanya digambar pada panel yang dibuka dari daftar berbasis klaim.
	detail.PIC = base.PIC
	detail.LossDate = base.LossDate
	if detail.BusinessName == "" {
		detail.BusinessName = base.BusinessName
	}
	detail.Items = items
	detail.History = base.History

	return detail, nil
}

// claimHeader membaca isian yang berasal dari klaim, bukan dari pengajuan.
func (r *Repo) claimHeader(
	ctx context.Context,
	claimNo string,
) (inboxsalvage.Detail, error) {
	var no, pic, businessName, lossDate sql.NullString

	row := r.db.QueryRowContext(ctx, query("claim_header"), claimNo)
	if err := row.Scan(&no, &pic, &businessName, &lossDate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
		}
		return inboxsalvage.Detail{}, fmt.Errorf("membaca klaim %q: %w", claimNo, err)
	}

	return inboxsalvage.Detail{
		ClaimNo:      strings.TrimSpace(no.String),
		PIC:          strings.TrimSpace(pic.String),
		BusinessName: strings.TrimSpace(businessName.String),
		LossDate:     dateOnly(lossDate.String),
		Items:        []inboxsalvage.DetailBarang{},
	}, nil
}

// latestSalvageOfClaim mencari ID pengajuan terakhir milik sebuah klaim.
//
// Kosong berarti klaim itu belum punya pengajuan sama sekali — bukan galat.
func (r *Repo) latestSalvageOfClaim(
	ctx context.Context,
	claimNo string,
) (string, error) {
	var id sql.NullString

	row := r.db.QueryRowContext(ctx, query("latest_salvage_of_claim"), claimNo)
	if err := row.Scan(&id); err != nil {
		// Agregat `MAX` selalu mengembalikan satu baris, bahkan ketika tidak ada baris
		// yang cocok — barisnya berisi NULL. ErrNoRows karena itu tidak diharapkan;
		// ditangani supaya perubahan kuerinya kelak tidak menjadi galat yang membingungkan.
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("mencari pengajuan terakhir klaim %q: %w", claimNo, err)
	}

	return strings.TrimSpace(id.String), nil
}

// detailHeader membaca kepala panel.
func (r *Repo) detailHeader(
	ctx context.Context,
	salvageID string,
) (inboxsalvage.Detail, error) {
	var (
		id, claimNo, inputDate, salvageType, quantity  sql.NullString
		estimate, location, transferGA, transferStatus sql.NullString
		acceptanceDate, acceptanceNo, remark, currency sql.NullString
		objectName, objectID, coverageName, coverageID sql.NullString
		acceptedValue, email, offerValue, winnerName   sql.NullString
		auctionDate, surveyorName, surveyorPhone       sql.NullString
		surveyorEmail, inJabodetabek, legacyFlag       sql.NullString
		businessName                                   sql.NullString
	)

	err := r.db.QueryRowContext(
		ctx, query("detail_header"), PegaWorkKeyPrefix, salvageID,
	).Scan(
		&id, &claimNo, &inputDate, &salvageType, &quantity,
		&estimate, &location, &transferGA, &transferStatus,
		&acceptanceDate, &acceptanceNo, &remark, &currency,
		&objectName, &objectID, &coverageName, &coverageID,
		&acceptedValue, &email, &offerValue, &winnerName,
		&auctionDate, &surveyorName, &surveyorPhone,
		&surveyorEmail, &inJabodetabek, &legacyFlag,
		&businessName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxsalvage.Detail{}, inboxsalvage.ErrRowNotFound
	}
	if err != nil {
		return inboxsalvage.Detail{}, fmt.Errorf(
			"inboxsalvage/sqlstore: detail_header: %w", err)
	}

	return inboxsalvage.Detail{
		// Baris yang terbaca dari `PNC_SALVAGE` berarti pengajuannya memang ada.
		HasSubmission: true,

		SalvageID:            id.String,
		ClaimNo:              claimNo.String,
		BusinessName:         businessName.String,
		InputDate:            dateOnly(inputDate.String),
		SalvageType:          salvageType.String,
		Quantity:             quantity.String,
		EstimateValue:        estimate.String,
		Location:             location.String,
		TransferGADate:       dateOnly(transferGA.String),
		TransferStatus:       transferStatus.String,
		Position:             inboxsalvage.PositionLabelOf(transferStatus.String),
		AcceptanceDate:       dateOnly(acceptanceDate.String),
		AcceptanceNo:         acceptanceNo.String,
		Remark:               remark.String,
		Currency:             currency.String,
		ObjectID:             objectID.String,
		ObjectName:           objectName.String,
		CoverageID:           coverageID.String,
		CoverageName:         coverageName.String,
		AcceptedValue:        acceptedValue.String,
		Email:                email.String,
		OfferValue:           offerValue.String,
		WinnerName:           winnerName.String,
		AuctionDate:          dateOnly(auctionDate.String),
		SurveyorName:         surveyorName.String,
		SurveyorPhone:        surveyorPhone.String,
		SurveyorEmail:        surveyorEmail.String,
		InJabodetabek:        flagIsSet(inJabodetabek.String),
		LegacyBeforeJuly2023: flagIsSet(legacyFlag.String),
	}, nil
}

// detailItems membaca daftar barang pada satu pengajuan.
func (r *Repo) detailItems(
	ctx context.Context,
	claimNo, salvageID string,
) ([]inboxsalvage.DetailBarang, error) {
	rows, err := r.db.QueryContext(ctx, query("detail_items"), claimNo, salvageID)
	if err != nil {
		return nil, fmt.Errorf("inboxsalvage/sqlstore: detail_items: %w", err)
	}
	defer rows.Close()

	items := []inboxsalvage.DetailBarang{}
	for rows.Next() {
		var (
			name, unit, total, soldStatus          sql.NullString
			winner, acceptanceNo, accepted, remark sql.NullString
			count                                  int
		)
		if err := rows.Scan(
			&name, &count, &unit, &total, &soldStatus,
			&winner, &acceptanceNo, &accepted, &remark,
		); err != nil {
			return nil, fmt.Errorf(
				"inboxsalvage/sqlstore: detail_items: memindai baris: %w", err)
		}

		items = append(items, inboxsalvage.DetailBarang{
			Name:          name.String,
			Count:         count,
			Unit:          unit.String,
			TotalValue:    total.String,
			SoldStatus:    inboxsalvage.SoldStatusOf(soldStatus.String),
			WinnerName:    winner.String,
			AcceptanceNo:  acceptanceNo.String,
			AcceptedValue: accepted.String,
			Remark:        remark.String,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxsalvage/sqlstore: detail_items: membaca hasil: %w", err)
	}

	return items, nil
}

// flagIsSet membaca penanda biner yang disimpan sebagai teks.
//
// Hanya `"1"` yang berarti benar. Kolomnya dapat berisi `NULL`, `"0"`, atau `"1"` — dan
// memperlakukan apa pun yang tidak kosong sebagai benar akan membalik arti `"0"`.
func flagIsSet(value string) bool {
	return strings.TrimSpace(value) == "1"
}

// Counts menyusun tabel ringkas "Status Salvage / Jumlah".
//
// # Kenapa satu kueri per baris, bukan satu kueri berisi banyak SUM
//
// Karena baris-barisnya menghitung TABEL YANG BERBEDA — dua di antaranya `T_CLAIM_PNC`,
// sepuluh `PNC_SALVAGE`, satu `DETAIL_PNC_SALVAGE`. Sistem lama menempuhnya dengan lima
// rule terpisah, dan tiga di antaranya menggabung-silangkan dua tabel tanpa syarat gabungan
// di `WHERE` hanya supaya seluruh hitungan muat dalam satu pernyataan. Itu bukan pola yang
// layak ditiru.
func (r *Repo) Counts(
	ctx context.Context,
	caller inboxsalvage.Caller,
) ([]inboxsalvage.StatusCount, error) {
	definitions := inboxsalvage.CountRows()
	counts := make([]inboxsalvage.StatusCount, 0, len(definitions))

	for _, definition := range definitions {
		total, err := r.countOne(ctx, definition, caller)
		if err != nil {
			return nil, err
		}
		counts = append(counts, inboxsalvage.StatusCount{
			Label: definition.Label,
			Tab:   definition.Tab,
			Total: total,
		})
	}

	return counts, nil
}

// countOne menjalankan satu hitungan pencacah.
func (r *Repo) countOne(
	ctx context.Context,
	definition inboxsalvage.CountRow,
	caller inboxsalvage.Caller,
) (int, error) {
	var (
		name string
		args []any
	)

	switch definition.Source {
	case inboxsalvage.CountFromSalvage:
		owner := patternAll
		if definition.OwnedByCaller {
			owner = escapeLike(caller.Login)
		}
		name = "count_salvage_status"
		args = []any{
			definition.TransferStatuses[0],
			lastOrFirst(definition.TransferStatuses),
			owner,
		}

	case inboxsalvage.CountFromClaimBuyback:
		name = "count_claim_buyback"

	case inboxsalvage.CountFromSalvageDetail:
		name = "count_salvage_detail_unsold"
		args = []any{UnsoldDetailStatus}

	default:
		name = "count_claim_status"
		args = []any{
			definition.SalvageStatuses[0],
			lastOrFirst(definition.SalvageStatuses),
		}
	}

	var total int
	if err := r.db.QueryRowContext(ctx, query(name), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("inboxsalvage/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// lastOrFirst mengembalikan nilai kedua sebuah pasangan, atau mengulang yang pertama bila
// hanya ada satu.
//
// Kueri pencacah memakai `IN (:1, :2)` supaya satu pernyataan melayani baris yang
// menghitung satu nilai maupun dua. Mengulang nilai yang sama tidak mengubah hasilnya.
func lastOrFirst(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

// Create menyimpan satu pengajuan salvage beserta detail itemnya.
//
// # SATU transaksi, bukan sembilan commit
//
// Sistem lama menempuh jalur ini lewat procedure yang meng-`COMMIT` sendiri
// (`Database/INSERT_SALVAGE.prc:26` dan `:47`), lalu procedure KEDUA yang meng-`COMMIT`
// lagi untuk setiap baris detail. Kegagalan di tengah meninggalkan pengajuan tanpa detail,
// atau detail sebagian.
//
// `D-68` memindahkan kepemilikan transaksi ke Go. Di sini seluruhnya — pengajuan dan
// seluruh baris detailnya — berada dalam SATU transaksi: gagal berarti tidak ada apa pun
// yang tertinggal.
//
// Konsekuensinya untuk uji kesetaraan disadari: keadaan akhir SAAT GAGAL berbeda dari
// sistem lama, dan itu selisih yang disengaja (`14-TESTING-STRATEGY.md` §6.4).
func (r *Repo) Create(ctx context.Context, form inboxsalvage.Form) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	salvageID := form.SalvageID
	if form.Mode != inboxsalvage.FormModeUpdate {
		salvageID, err = nextSalvageID(ctx, tx)
		if err != nil {
			return "", err
		}
	}

	args := insertArgs(form, salvageID)

	statement := "insert_salvage"
	if form.Mode == inboxsalvage.FormModeUpdate {
		statement = "update_salvage"
	}

	result, err := tx.ExecContext(ctx, query(statement), args...)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: %s: %w", statement, err)
	}

	// Pembaruan yang tidak menyentuh satu baris pun berarti pengajuannya tidak ada.
	// Membiarkannya lolos akan melaporkan "tersimpan" untuk penyimpanan yang tidak terjadi.
	if form.Mode == inboxsalvage.FormModeUpdate {
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return "", inboxsalvage.ErrRowNotFound
		}
	}

	if err := insertDetails(ctx, tx, form, salvageID); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: menyimpan transaksi: %w", err)
	}
	return salvageID, nil
}

// nextSalvageID membaca nomor ID salvage berikutnya.
//
// # Cacat yang DIWARISI, dan sejauh mana ia ditutup
//
// `MAX(IDSALVAGE) + 1` dapat menghasilkan nomor yang sama bagi dua penyimpanan yang
// berjalan bersamaan. Procedure lama punya cacat yang sama persis.
//
// Yang ditutup dari sisi ini: pembacaannya berada DI DALAM transaksi yang sama dengan
// penyisipannya, sehingga benturan berakhir sebagai galat kunci ganda — bukan sebagai dua
// baris ber-ID sama yang keduanya tersimpan. Penyimpanan yang kalah menerima galat dan
// pengguna dapat mengulang.
//
// Yang TIDAK ditutup: benturannya sendiri. Menghapusnya menuntut sequence, dan memindahkan
// penomoran yang sudah berjalan di produksi ke sequence adalah keputusan tersendiri yang
// belum diambil.
func nextSalvageID(ctx context.Context, tx *sql.Tx) (string, error) {
	var next int64
	if err := tx.QueryRowContext(ctx, query("next_salvage_id")).Scan(&next); err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: next_salvage_id: %w", err)
	}
	return strconv.FormatInt(next, 10), nil
}

// insertArgs menyusun ke-22 bind pernyataan simpan.
//
// Urutannya mengikuti urutan bind di inboxsalvage.sql, dan kesesuaiannya dijaga
// query_test.go. Pemetaannya ke parameter procedure lama ada di kepala berkas .sql itu.
func insertArgs(form inboxsalvage.Form, salvageID string) []any {
	return []any{
		form.ClaimNo,                        // :1  NOKLAIM
		nullIfEmpty(form.InputDate),         // :2  TGLINPUT
		form.SalvageType,                    // :3  JENISSALVAGE
		numberOrNull(form.Quantity),         // :4  QUANTITYSALVAGE
		numberOrNull(form.MinimumValue),     // :5  ESTIMASINILAI
		nullIfEmpty(form.Location),          // :6  LOKASISALVAGE
		form.TransferStatus,                 // :7  STSTRANSFER
		nullIfEmpty(form.Remark),            // :8  REMARK
		form.Caller.Login,                   // :9  PIC
		salvageID,                           // :10 IDSALVAGE
		nullIfEmpty(form.ObjectID),          // :11 IDOBJECT
		form.ObjectName,                     // :12 OBJECTNAME
		nullIfEmpty(form.CoverageID),        // :13 IDCOVERAGE
		form.CoverageName,                   // :14 COVERAGENAME
		nullIfEmpty(form.Currency),          // :15 CURRENCY
		numberOrNull(form.InsuredShare),     // :16 NILAIAKSEP
		nullIfEmpty(form.Email),             // :17 EMAIL
		numberOrNull(form.OfferValue),       // :18 NILAIPENAWARAN
		nullIfEmpty(form.SurveyorName),      // :19 PICSURVEY
		nullIfEmpty(form.SurveyorEmail),     // :20 EMAILSURVEY
		nullIfEmpty(form.SurveyorPhone),     // :21 NOTELP
		jabodetabekFlag(form.InJabodetabek), // :22 ISJABODATABEK
	}
}

// Nilai kolom `DETAIL_PNC_SALVAGE` yang di procedure lama ditulis TETAP.
//
// Keduanya diikat, bukan ditanam di dalam SQL, karena keduanya nilai bisnis (`D-15`).
// Nilainya sendiri disalin apa adanya dari `Database/INSERT_SALVAGE_DETAILS.prc:26`.
const (
	// NewDetailStatus adalah `STATUSTERJUAL` baris detail yang baru disisipkan — `'3'`.
	//
	// Perhatikan `'3'` pula yang dipakai kueri agregat sebagai penyaring
	// (`statusterjual is null or statusterjual = '3'`), sehingga baris yang baru disimpan
	// langsung ikut terhitung pada kolom "Nilai Request Balai Lelang".
	NewDetailStatus = "3"

	// EmptyAcceptanceNo adalah `NOAKSEPTASI` baris detail yang belum berakseptasi — `'-'`.
	//
	// Tanda hubung, bukan NULL. Itu keadaan di procedure lama, dan mengubahnya menjadi
	// NULL akan membuat baris baru dan baris lama terbaca berbeda pada laporan yang
	// membandingkan kolom ini.
	EmptyAcceptanceNo = "-"

	// UnsoldDetailStatus adalah `STATUSTERJUAL` yang dihitung baris pencacah "Tidak
	// Terjual" — `'0'`, dari `Activity/GCNMCountSalvage_act-Act.xml` langkah 48.
	UnsoldDetailStatus = "0"
)

// insertDetails menyisipkan seluruh baris Detail Item Salvage.
//
// Urutan `IDDETAILSALVAGE` melanjutkan baris yang SUDAH ADA untuk pengajuan itu, sama
// seperti procedure lama yang menghitungnya lebih dulu. Hitungannya dilakukan SEKALI di
// awal, bukan sekali per baris: keduanya berada di dalam satu transaksi, sehingga tidak ada
// baris lain yang dapat menyisip di antaranya.
func insertDetails(
	ctx context.Context,
	tx *sql.Tx,
	form inboxsalvage.Form,
	salvageID string,
) error {
	if len(form.Items) == 0 {
		return nil
	}

	var existing int
	err := tx.QueryRowContext(
		ctx, query("count_salvage_detail_for"), salvageID, form.ClaimNo,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("inboxsalvage/sqlstore: count_salvage_detail_for: %w", err)
	}

	statement := query("insert_salvage_detail")
	for index, item := range form.Items {
		sequence := existing + index + 1
		detailID := form.ClaimNo + "/" + salvageID + "/" + strconv.Itoa(sequence)

		_, err := tx.ExecContext(ctx, statement,
			salvageID,                   // :1  IDSALVAGE
			form.ClaimNo,                // :2  NOKLAIM
			detailID,                    // :3  IDDETAILSALVAGE
			item.Name,                   // :4  NAMABARANG
			nullIfEmpty(item.Unit),      // :5  SATUAN
			nullIfEmpty(item.Remarks),   // :6  NOTE_ITEM
			numberOrNull(item.Quantity), // :7  HARGAITEM
			strconv.Itoa(sequence),      // :8  IDOBJECT
			nullIfEmpty(item.Remarks),   // :9  REMARK
			NewDetailStatus,             // :10 STATUSTERJUAL
			EmptyAcceptanceNo,           // :11 NOAKSEPTASI
		)
		if err != nil {
			return fmt.Errorf(
				"inboxsalvage/sqlstore: insert_salvage_detail baris %d: %w",
				index+1, err)
		}
	}

	return nil
}

// jabodetabekFlag menerjemahkan penanda "Lokasi Salvage Di Jabodatabek".
//
// Kolomnya bertipe teks di basis data, dan nilainya `"1"` atau `"0"` — bentuk yang sama
// dipakai seluruh penanda biner di skema ini.
func jabodetabekFlag(in bool) string {
	if in {
		return "1"
	}
	return "0"
}

// nullIfEmpty mengirim NULL alih-alih teks kosong.
//
// Perbedaannya nyata di layar: kolom yang NULL tergambar kosong, sementara teks kosong
// tergambar kosong pula — tetapi keduanya berbeda pada kueri yang memakai `IS NULL`, dan
// kueri agregat modul ini memakainya.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// numberOrNull mengirim NULL alih-alih teks kosong untuk kolom bertipe angka.
//
// Isian kosong TIDAK boleh dikirim sebagai `""`: `TO_NUMBER(”)` gagal di Oracle dengan
// `ORA-01722`, dan galat itu sampai ke pengguna sebagai kegagalan mentah.
//
// Nilainya sendiri tetap dikirim sebagai TEKS, bukan sebagai `float64`. `D-51` menetapkan
// nilai uang disimpan presisi penuh; mengubahnya menjadi bilangan pecahan biner di sini
// berarti pembulatan terjadi sebelum nilainya sampai ke basis data.
func numberOrNull(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return strings.ReplaceAll(clean, ",", ".")
}

// dateOnly memotong bagian waktu dari nilai tanggal yang dibaca basis data.
//
// Driver mengembalikan `DATE` Oracle sebagai teks berbentuk `YYYY-MM-DD HH24:MI:SS` pada
// sebagian setelan. Yang digambar layar hanyalah tanggalnya, dan yang dihitung "Aging" pun
// hanya tanggalnya.
func dateOnly(value string) string {
	clean := strings.TrimSpace(value)
	if len(clean) > len(inboxsalvage.DateLayout) {
		return clean[:len(inboxsalvage.DateLayout)]
	}
	return clean
}

// today membaca tanggal hari ini menurut BASIS DATA, bukan menurut mesin aplikasi.
//
// # Kenapa basis data
//
// Karena tanggal input salvage pun berasal dari sana. Menghitung selisih antara tanggal
// basis data dan jam mesin aplikasi berarti membandingkan dua jam yang dapat berbeda —
// dan pada layar yang menghitung umur pekerjaan, selisih satu hari terlihat sebagai
// pekerjaan yang terlambat padahal tidak.
//
// `CURRENT_DATE` dipakai, bukan `SYSDATE`: yang pertama mengikuti zona waktu sesi, yang
// kedua zona waktu mesin basis data. Perbedaannya adalah persis kelas cacat yang `R-12`
// catat.
func (r *Repo) today(ctx context.Context) (string, error) {
	var value sql.NullString
	err := r.db.QueryRowContext(
		ctx, "SELECT TO_CHAR(CURRENT_DATE, 'YYYY-MM-DD') FROM DUAL",
	).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("inboxsalvage/sqlstore: membaca tanggal basis data: %w", err)
	}
	return value.String, nil
}

// Ready memeriksa kolom yang dibaca modul ini memang ada.
//
// Dipakai perintah `check`, bukan jalur permintaan. Ia menjawab pertanyaan yang paling
// sering muncul saat layar kosong: apakah kolomnya tidak ada, atau datanya yang tidak ada.
func (r *Repo) Ready(ctx context.Context) (int, error) {
	var found int
	err := r.db.QueryRowContext(ctx, query("check_salvage_columns")).Scan(&found)
	if err != nil {
		return 0, fmt.Errorf("inboxsalvage/sqlstore: check_salvage_columns: %w", err)
	}
	return found, nil
}
