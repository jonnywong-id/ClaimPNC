package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/platform/money"
	"claim-pnc/internal/platform/sqlvalue"
)

// Repo membaca daftar klaim outstanding dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama, dan
// selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca daftar di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// BranchOf menerjemahkan kode cabang rinci milik pemanggil menjadi cabang yang barisnya ia
// lihat, atau false bila kode itu tidak dikenal master cabang.
//
// Kode yang tidak dikenal BUKAN galat. Ia keadaan yang mungkin dan punya arti tersendiri —
// pemanggil tidak punya cabang, atau kodenya berada di ruang kode yang lain — dan pemanggil
// yang memutuskan apa yang dilakukan atasnya (lihat usecase.Service.List).
//
// Kode kosong dijawab false tanpa menyentuh basis data. Menanyakannya tetap akan menghasilkan
// nol baris, dan perjalanan yang jawabannya sudah pasti tidak perlu dibayar.
func (r *Repo) BranchOf(
	ctx context.Context,
	detailBranchCode string,
) (inboxosclaimpercabang.Branch, bool, error) {
	if strings.TrimSpace(detailBranchCode) == "" {
		return inboxosclaimpercabang.Branch{}, false, nil
	}

	var code, name sql.NullString

	err := r.db.QueryRowContext(ctx, query("branch_of"), detailBranchCode).Scan(&code, &name)
	switch {
	case err == sql.ErrNoRows:
		return inboxosclaimpercabang.Branch{}, false, nil
	case err != nil:
		return inboxosclaimpercabang.Branch{}, false,
			fmt.Errorf("membaca POOLDATA.BRANCH: %w", err)
	}

	// Kode kosong diperlakukan sebagai tidak dikenal. Meneruskannya akan membuat kueri daftar
	// menyaring `branchcode = ''` — nol baris, tanpa satu pun tanda bahwa cabangnya yang tidak
	// terbaca.
	if strings.TrimSpace(code.String) == "" {
		return inboxosclaimpercabang.Branch{}, false, nil
	}

	return inboxosclaimpercabang.Branch{
		Code: strings.TrimSpace(code.String),
		Name: strings.TrimSpace(name.String),
	}, true, nil
}

// List mengambil satu halaman baris grid beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi. Jumlah seluruhnya datang dari kolom
// TOTAL_ROWS pada baris mana pun; ia sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
//
// Kolom Aging TIDAK diisi di sini — ia dihitung pemanggil terhadap tanggal WIB. Alasannya di
// inboxosclaimpercabang.AgingDaysSince.
func (r *Repo) List(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
	page inboxosclaimpercabang.Pagination,
) (inboxosclaimpercabang.Page, error) {
	clean := page.Normalize()
	result := inboxosclaimpercabang.Page{
		Items:      []inboxosclaimpercabang.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query("list"),
		q.Branch.Code, clean.Offset(), clean.Size)
	if err != nil {
		return inboxosclaimpercabang.Page{}, fmt.Errorf("menjalankan kueri list: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxosclaimpercabang.Page{},
				fmt.Errorf("membaca baris kueri list: %w", err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxosclaimpercabang.Page{},
			fmt.Errorf("menelusuri hasil kueri list: %w", err)
	}

	return result, nil
}

// ListForExport mengambil satu halaman baris berkas ekspor.
//
// Ia memakai kueri yang BERBEDA dari List, bukan kueri yang sama dengan kolom tambahan —
// termasuk cara menghitung nilai uangnya. Lihat catatan di kepala berkas .sql.
func (r *Repo) ListForExport(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
	page inboxosclaimpercabang.Pagination,
) (inboxosclaimpercabang.ExportPage, error) {
	clean := page.Normalize()
	result := inboxosclaimpercabang.ExportPage{
		Items:      []inboxosclaimpercabang.ExportRow{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query("list_export"),
		q.Branch.Code, clean.Offset(), clean.Size)
	if err != nil {
		return inboxosclaimpercabang.ExportPage{},
			fmt.Errorf("menjalankan kueri list_export: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		row, total, err := scanExportRow(rows)
		if err != nil {
			return inboxosclaimpercabang.ExportPage{},
				fmt.Errorf("membaca baris kueri list_export: %w", err)
		}
		result.Items = append(result.Items, row)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxosclaimpercabang.ExportPage{},
			fmt.Errorf("menelusuri hasil kueri list_export: %w", err)
	}

	return result, nil
}

// DominantFactors mengembalikan faktor dominan setiap klaim outstanding satu cabang.
//
// Barisnya sudah berurut menurut `idx_dominanfactor`, sehingga urutan di dalam setiap daftar
// mengikuti urutan kueri apa adanya — tidak diurutkan ulang di sini dengan aturan yang dapat
// berselisih dengan yang di berkas .sql.
func (r *Repo) DominantFactors(
	ctx context.Context,
	q inboxosclaimpercabang.Query,
) (map[string][]string, error) {
	rows, err := r.db.QueryContext(ctx, query("dominant_factors"), q.Branch.Code)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri dominant_factors: %w", err)
	}
	defer rows.Close()

	result := map[string][]string{}
	for rows.Next() {
		var claimKey, name sql.NullString
		if err := rows.Scan(&claimKey, &name); err != nil {
			return nil, fmt.Errorf("membaca baris kueri dominant_factors: %w", err)
		}
		if claimKey.String == "" {
			continue
		}
		result[claimKey.String] = append(result[claimKey.String], name.String)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri dominant_factors: %w", err)
	}

	return result, nil
}

// CheckTable memastikan tabel yang disentuh modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah hak
// baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_PNC, POOLDATA.T_CLAIMLIST_ADMIN, "+
				"POOLDATA.GCNM_PROGRESS_CLAIM, POOLDATA.GCNM_MST_PROGRESS, "+
				"POOLDATA.T_CLAIM_ESTIMASI, POOLDATA.T_SURVEYORLIST, "+
				"POOLDATA.T_CLAIM_OBJECTCOVERAGE, atau POOLDATA.BRANCH: %w", err)
	}
	return nil
}

// CheckExportTable memastikan tabel yang HANYA dipakai ekspor terbaca, termasuk lewat DB Link.
//
// Ia terpisah dari CheckTable supaya DB Link `@asmd` yang sedang padam terbaca sebagai
// kegagalan EKSPOR — bukan sebagai kegagalan seluruh layar. Keduanya punya tindak lanjut yang
// berbeda: yang pertama menunggu tim jaringan, yang kedua menghentikan pekerjaan cabang.
func (r *Repo) CheckExportTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_export_tables")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_GENERAL, POOLDATA.T_CLAIM_DOMINANFACTOR, "+
				"POOLDATA.M_DOMINAN_FACTOR, atau treaty_loss@asmd: %w", err)
	}
	return nil
}

// AnyBranchWithClaims mengembalikan satu cabang yang benar-benar punya klaim outstanding,
// beserta kode RINCI-nya.
//
// Kode rinci ikut dikembalikan supaya `-periksa` dapat menempuh jalur yang sama dengan layar:
// kode rinci lalu BranchOf lalu List. Memberi List kode klaim secara langsung akan melewati
// terjemahannya — tepat langkah yang paling mungkin salah.
//
// Dipakai perintah `-periksa` saja, dan karena itu ia TIDAK ada di seam Repo: layar tidak
// pernah memilih cabang sendiri — batas datanya selalu cabang pemanggil. Menaruhnya di seam
// akan membuat kemampuan "pilih cabang mana saja" tersedia bagi kode yang ditulis kemudian.
//
// Nilai kedua false berarti tidak ada satu pun klaim outstanding di basis data itu. Itu bukan
// galat; ia keadaan yang wajar pada basis data yang baru disalin.
func (r *Repo) AnyBranchWithClaims(ctx context.Context) (branchCode, detailBranchCode string, found bool, err error) {
	var code, detail sql.NullString

	err = r.db.QueryRowContext(ctx, query("any_branch_with_claims")).Scan(&code, &detail)
	switch {
	case err == sql.ErrNoRows:
		return "", "", false, nil
	case err != nil:
		return "", "", false, fmt.Errorf("membaca kode cabang contoh: %w", err)
	case !code.Valid || strings.TrimSpace(code.String) == "":
		return "", "", false, nil
	}

	return strings.TrimSpace(code.String), strings.TrimSpace(detail.String), true, nil
}

// scanner adalah bentuk minimal yang dibutuhkan pemindai, sehingga keduanya dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris menjadi WorkItem beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan listColumns dan dengan urutan kolom di berkas .sql. Ketiganya
// dijaga query_test.go.
//
// Seluruh kolom teks dipindai lewat tipe yang mengizinkan NULL. Itu bukan kehati-hatian
// berlebih: enam gabungannya LEFT JOIN, sehingga klaim tanpa catatan progres, tanpa estimasi,
// tanpa adjuster, atau tanpa coverage mengembalikan NULL pada seluruh kolom tabel itu
// sekaligus.
//
// Kedua kolom tanggal dipindai sebagai penunjuk. Kosong berbeda artinya dari tanggal mana pun:
// klaim tanpa catatan progres tidak punya "tanggal update terakhir", dan menggambarnya sebagai
// awal zaman akan menampilkan tanggal yang tidak pernah ada.
func scanWorkItem(row scanner) (inboxosclaimpercabang.WorkItem, int, error) {
	var (
		branchName, branchCode, businessSource, businessName sql.NullString
		policyNumber, insuredName, claimNumber               sql.NullString
		registerDate, lossDate, lastProgressAt               sql.NullTime
		remarkRecommendation                                 sql.NullString
		estimationValue                                      any
		progressStatus1, progressStatus2, technicalPIC       sql.NullString
		progressNote, adjusterName, causeOfLoss, chronology  sql.NullString
		progressStalled                                      sql.NullInt64
		total                                                sql.NullInt64
	)

	err := row.Scan(
		&branchName, &branchCode, &businessSource, &businessName,
		&policyNumber, &insuredName, &claimNumber, &registerDate, &lossDate,
		&remarkRecommendation, &estimationValue, &lastProgressAt,
		&progressStatus1, &progressStatus2, &technicalPIC, &progressNote,
		&adjusterName, &causeOfLoss, &chronology, &progressStalled,
		&total,
	)
	if err != nil {
		return inboxosclaimpercabang.WorkItem{}, 0, err
	}

	value, err := minorUnits(estimationValue)
	if err != nil {
		return inboxosclaimpercabang.WorkItem{}, 0, err
	}

	return inboxosclaimpercabang.WorkItem{
		BranchName:           branchName.String,
		BranchCode:           branchCode.String,
		BusinessSource:       businessSource.String,
		BusinessName:         businessName.String,
		PolicyNumber:         policyNumber.String,
		ClaimNumber:          claimNumber.String,
		RegisterDate:         nullableTime(registerDate),
		LossDate:             nullableTime(lossDate),
		RemarkRecommendation: remarkRecommendation.String,
		EstimationValue:      value,
		LastProgressAt:       nullableTime(lastProgressAt),
		ProgressStatus1:      progressStatus1.String,
		ProgressStatus2:      progressStatus2.String,
		TechnicalPIC:         technicalPIC.String,
		ProgressNote:         progressNote.String,
		AdjusterName:         adjusterName.String,
		CauseOfLoss:          causeOfLoss.String,
		Chronology:           chronology.String,
		ProgressStalled:      progressStalled.Int64 == 1,
	}, int(total.Int64), nil
}

// scanExportRow memindai satu baris berkas ekspor.
//
// Ke-20 kolom pertamanya SAMA PERSIS dengan scanWorkItem, karena `list_export` memuat alias
// `list` sebagai awalan — kecuali TOTAL_ROWS yang pindah ke ujung. Pemindainya karena itu
// ditulis lengkap di sini alih-alih memanggil scanWorkItem: kedua kueri boleh berbeda
// panjangnya, dan pemindai yang dibagi akan menyembunyikan perbedaan itu di tempat yang tidak
// terbaca.
func scanExportRow(row scanner) (inboxosclaimpercabang.ExportRow, int, error) {
	var (
		branchName, branchCode, businessSource, businessName sql.NullString
		policyNumber, insuredName, claimNumber               sql.NullString
		registerDate, lossDate, lastProgressAt               sql.NullTime
		remarkRecommendation                                 sql.NullString
		estimationValue                                      any
		progressStatus1, progressStatus2, technicalPIC       sql.NullString
		progressNote, adjusterName, causeOfLoss, chronology  sql.NullString
		progressStalled                                      sql.NullInt64

		claimKey, policyBusinessName         sql.NullString
		reserveFull, reserveASM, coinsurance any
		shares                               [24]any

		total sql.NullInt64
	)

	dest := []any{
		&branchName, &branchCode, &businessSource, &businessName,
		&policyNumber, &insuredName, &claimNumber, &registerDate, &lossDate,
		&remarkRecommendation, &estimationValue, &lastProgressAt,
		&progressStatus1, &progressStatus2, &technicalPIC, &progressNote,
		&adjusterName, &causeOfLoss, &chronology, &progressStalled,
		&claimKey, &policyBusinessName,
		&reserveFull, &reserveASM, &coinsurance,
	}
	for i := range shares {
		dest = append(dest, &shares[i])
	}
	dest = append(dest, &total)

	if err := row.Scan(dest...); err != nil {
		return inboxosclaimpercabang.ExportRow{}, 0, err
	}

	// Keempat nilai uang dan ke-24 pembagian treaty diubah SESUDAH pemindaian, bukan lewat
	// tipe pemindai. Kolomnya NUMBER dan driver dapat menyerahkannya dalam beberapa bentuk;
	// minorUnits yang menanganinya di satu tempat.
	values := map[string]money.Money{}
	for label, raw := range map[string]any{
		"ESTIMATION_VALUE":   estimationValue,
		"RESERVE_CLAIM_FULL": reserveFull,
		"RESERVE_CLAIM_ASM":  reserveASM,
		"COINSURANCE":        coinsurance,
	} {
		converted, err := minorUnits(raw)
		if err != nil {
			return inboxosclaimpercabang.ExportRow{}, 0,
				fmt.Errorf("kolom %s: %w", label, err)
		}
		values[label] = converted
	}

	treaty, err := treatySharesFrom(shares)
	if err != nil {
		return inboxosclaimpercabang.ExportRow{}, 0, err
	}

	return inboxosclaimpercabang.ExportRow{
		WorkItem: inboxosclaimpercabang.WorkItem{
			BranchName:           branchName.String,
			BranchCode:           branchCode.String,
			BusinessSource:       businessSource.String,
			BusinessName:         businessName.String,
			PolicyNumber:         policyNumber.String,
			ClaimNumber:          claimNumber.String,
			RegisterDate:         nullableTime(registerDate),
			LossDate:             nullableTime(lossDate),
			RemarkRecommendation: remarkRecommendation.String,
			EstimationValue:      values["ESTIMATION_VALUE"],
			LastProgressAt:       nullableTime(lastProgressAt),
			ProgressStatus1:      progressStatus1.String,
			ProgressStatus2:      progressStatus2.String,
			TechnicalPIC:         technicalPIC.String,
			ProgressNote:         progressNote.String,
			AdjusterName:         adjusterName.String,
			CauseOfLoss:          causeOfLoss.String,
			Chronology:           chronology.String,
			ProgressStalled:      progressStalled.Int64 == 1,
		},
		ClaimKey:           claimKey.String,
		PolicyBusinessName: policyBusinessName.String,
		ReserveClaimFull:   values["RESERVE_CLAIM_FULL"],
		ReserveClaimASM:    values["RESERVE_CLAIM_ASM"],
		Coinsurance:        values["COINSURANCE"],
		TreatyShares:       treaty,
	}, int(total.Int64), nil
}

// treatySharesFrom menyusun ke-24 pembagian treaty dari hasil pindai.
//
// Urutannya WAJIB sama dengan exportOnlyColumns dan dengan urutan kolom `list_export`. Ia
// ditulis berurut dan lengkap, bukan lewat refleksi: satu pergeseran indeks pada berkas berisi
// angka uang tidak menghasilkan satu pun galat.
func treatySharesFrom(v [24]any) (inboxosclaimpercabang.TreatyShares, error) {
	var converted [24]money.Money
	for index, raw := range v {
		value, err := minorUnits(raw)
		if err != nil {
			return inboxosclaimpercabang.TreatyShares{}, fmt.Errorf(
				"kolom treaty %s: %w",
				inboxosclaimpercabang.ExportTreatyColumns[index], err)
		}
		converted[index] = value
	}

	var result inboxosclaimpercabang.TreatyShares
	result.OR = converted[0]
	result.FacOut = converted[1]
	result.FacOB = converted[2]
	result.QS = converted[3]
	result.FSPL = converted[4]
	result.SSPL = converted[5]
	result.ER1 = converted[6]
	result.ER2 = converted[7]
	result.BPPDAN = converted[8]
	result.PSRQS = converted[9]
	result.PSRSPL = converted[10]
	result.ORS = converted[11]
	result.XL = converted[12]
	result.PSROR = converted[13]
	result.QSOR = converted[14]
	result.PSS = converted[15]
	result.PRGBI = converted[16]
	result.PFRA = converted[17]
	result.FSPLNSRI = converted[18]
	result.PSPLNSRI = converted[19]
	result.FSPLNSOR = converted[20]
	result.PSPLNSOR = converted[21]
	result.FacOBSRB = converted[22]
	result.FacOBIndt = converted[23]
	return result, nil
}

// minorUnits mengubah nilai uang dari basis data menjadi money.Money.
//
// # Kenapa ia tidak memakai money.FromSQLValue
//
// Karena yang itu menerima nilai dalam RUPIAH dan mengalikannya seratus. Kueri di modul ini
// sudah mengembalikan satuan terkecil — `ROUND(<ekspresi> * 100)` — dan menyerahkannya ke
// FromSQLValue akan menghasilkan nilai seratus kali lipat, diam-diam dan tanpa satu pun galat.
//
// Pembagian tugasnya karena itu tegas: yang mengalikan seratus adalah SQL, satu kali, di
// tempat yang terbaca.
//
// # Kenapa bentuknya harus ditangani satu per satu
//
// Kolom Oracle bertipe NUMBER dapat sampai ke Go sebagai int64, float64, []byte, atau string,
// bergantung pada presisi kolom dan driver. `ROUND` menghasilkan bilangan bulat, tetapi
// bentuknya di sisi Go tetap bergantung driver — pada koneksi yang diuji ia datang sebagai
// int64.
//
// float64 diterima HANYA bila nilainya bilangan bulat dan masih di bawah 2^53. Di luar itu ia
// ditolak, bukan dibulatkan diam-diam: nilai uang yang sudah kehilangan ketepatan tidak boleh
// diteruskan seolah-olah masih tepat.
func minorUnits(value any) (money.Money, error) {
	switch v := value.(type) {
	case nil:
		// NULL berarti tidak ada nilai. Nol menyatakan itu dengan tepat, dan seluruh
		// gabungan yang menghasilkannya sudah memakai COALESCE.
		return money.Zero, nil
	case int64:
		return money.FromMinorUnits(v), nil
	case int:
		return money.FromMinorUnits(int64(v)), nil
	case float64:
		switch {
		case math.IsNaN(v) || math.IsInf(v, 0):
			return money.Zero, fmt.Errorf("nilai uang bukan bilangan: %v", v)
		case v != math.Trunc(v):
			return money.Zero, fmt.Errorf(
				"nilai uang %v berdesimal padahal kueri sudah membulatkannya ke satuan "+
					"terkecil; ketepatannya tidak dapat dijamin", v)
		case v >= 1<<53 || v <= -(1<<53):
			return money.Zero, fmt.Errorf(
				"nilai uang %v melampaui rentang yang float64 masih tepat", v)
		}
		return money.FromMinorUnits(int64(v)), nil
	case []byte:
		return minorUnitsFromText(string(v))
	case string:
		return minorUnitsFromText(v)
	default:
		// Tipe angka milik driver yang berperilaku seperti teks tercakup di sini. Cabang ini
		// menjaga pergantian driver kelak tidak langsung membuat modul ini gagal.
		if text, able := value.(fmt.Stringer); able {
			return minorUnitsFromText(text.String())
		}
		return money.Zero, fmt.Errorf("nilai uang bertipe %T tidak dikenali", value)
	}
}

// minorUnitsFromText membaca satuan terkecil yang datang sebagai teks.
//
// Ia menuntut BILANGAN BULAT. Kueri sudah membulatkannya dengan `ROUND`, sehingga teks
// berdesimal berarti kuerinya berubah tanpa pemindainya ikut disesuaikan — dan itu harus
// terlihat sebagai galat, bukan dibulatkan lagi di sini.
func minorUnitsFromText(text string) (money.Money, error) {
	units, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return money.Zero, fmt.Errorf(
			"nilai uang %q bukan bilangan bulat satuan terkecil: %w", text, err)
	}
	return money.FromMinorUnits(units), nil
}

// nullableTime mengubah tanggal yang boleh kosong menjadi penunjuk.
//
// Kosong dikembalikan sebagai nil, bukan sebagai waktu nol. Keduanya berbeda artinya di layar:
// yang pertama kolom kosong, yang kedua tanggal 1 Januari tahun 1.
func nullableTime(value sql.NullTime) *time.Time { return sqlvalue.TimeOrNil(value) }
