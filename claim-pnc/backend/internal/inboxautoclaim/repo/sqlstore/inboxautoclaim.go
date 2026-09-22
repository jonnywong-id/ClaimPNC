// Package sqlstore memenuhi seam inboxautoclaim.Repo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxautoclaim"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// dateLayout adalah bentuk tanggal yang dipakai seluruh layar dan berkas modul ini.
//
// Pemformatannya dilakukan di Go, bukan dengan TO_CHAR di SQL (D-20) — kueri asli Pega
// memakai `TO_CHAR(a.TGLPROSES,'DD/MM/YYYY')`, dan itu satu dari sembilan pola yang
// 09-DATABASE-STRATEGY.md §3.2 pindahkan ke aplikasi.
const dateLayout = "02/01/2006"

// Repo membaca POOLDATA.TMP_BATCH_AUTO_CLAIM dan POOLDATA.M_AUTO_CLAIM_PNC, serta menulis
// baris unggahan ke tabel pertama.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// ListBatch membaca satu halaman daftar batch beserta jumlah seluruhnya.
//
// Dua kueri, bukan satu: jumlah keseluruhan tidak dapat diketahui dari halaman yang sudah
// dipotong OFFSET.
//
// Kode perusahaan TIDAK di-TRIM di kueri. DDL yang diterima 2026-09-19 menyatakan
// INISIALID bertipe VARCHAR2(20), bukan CHAR — TRIM tidak diperlukan, dan memakainya
// justru mematikan index TMP_BATCH_AUTO_CLAIM_IDX yang kolom keduanya persis INISIALID.
//
// # Kode perusahaan juga TIDAK di-UPPERCASE, dan itu perbaikan atas cacat nyata
//
// Versi sebelumnya memanggil strings.ToUpper di sini. Nilai yang dibandingkan datang dari
// ListCompany, yang membacanya APA ADANYA dari M_AUTO_CLAIM_PNC dan tidak meng-uppercase
// apa pun. Jadi nilai keluar dan nilai masuk diperlakukan berbeda: begitu INISIALID di
// basis data tidak seluruhnya huruf besar, WHERE-nya tidak akan pernah cocok dan
// penyaring perusahaan selalu menghasilkan tabel kosong.
//
// Kueri Pega aslinya menjodohkan `a.INISIALID = b.INISIALID` tanpa mengubah keduanya.
// Aturannya di sini sama: **nilai yang dibaca dari basis data dikembalikan apa adanya**,
// dan tidak ada transformasi sepihak di salah satu arah saja.
func (r *Repo) ListBatch(ctx context.Context, filter inboxautoclaim.BatchFilter) (inboxautoclaim.BatchPage, error) {
	page := filter.Page.Clean()
	company := strings.TrimSpace(filter.CompanyCode)

	var (
		rows  *sql.Rows
		total int
		err   error
	)

	if company == "" {
		total, err = r.countOne(ctx, filter.Source, "auto_claim_batch_count")
		if err != nil {
			return inboxautoclaim.BatchPage{}, err
		}
		// MessageSuccess dikirim DUA KALI karena kuerinya memakai dua bind terpisah
		// untuk nilai yang sama — lihat catatan pada berkas .sql.
		rows, err = r.db.QueryContext(ctx, getQueryFor(filter.Source, "auto_claim_batch_list"),
			inboxautoclaim.MessageSuccess, inboxautoclaim.MessageSuccess,
			filter.Page.Offset(), page.Size)
	} else {
		total, err = r.countOne(ctx, filter.Source, "auto_claim_batch_count_by_company", company)
		if err != nil {
			return inboxautoclaim.BatchPage{}, err
		}
		rows, err = r.db.QueryContext(ctx, getQueryFor(filter.Source, "auto_claim_batch_list_by_company"),
			inboxautoclaim.MessageSuccess, inboxautoclaim.MessageSuccess,
			company, filter.Page.Offset(), page.Size)
	}
	if err != nil {
		return inboxautoclaim.BatchPage{}, fmt.Errorf("inboxautoclaim/sqlstore: membaca daftar batch: %w", err)
	}
	defer func() { _ = rows.Close() }()

	item := make([]inboxautoclaim.Batch, 0, page.Size)
	for rows.Next() {
		batch, scanErr := scanBatch(rows)
		if scanErr != nil {
			return inboxautoclaim.BatchPage{}, scanErr
		}
		item = append(item, batch)
	}
	if err := rows.Err(); err != nil {
		return inboxautoclaim.BatchPage{}, fmt.Errorf("inboxautoclaim/sqlstore: menelusuri daftar batch: %w", err)
	}
	return inboxautoclaim.BatchPage{Item: item, Total: total}, nil
}

// ListCompany membaca pilihan penyaring dari MASTER perusahaan.
//
// Sumbernya master, bukan tabel batch — diselaraskan ke
// InboxAutoClaim/BrowseCompanyClaimCredit-SQL.xml. Perusahaan yang belum pernah mengirim
// apa pun karena itu ikut tampil, dan memilihnya menghasilkan daftar kosong. Itu perilaku
// aslinya; versi pertama modul ini menyaringnya dan karena itu berbeda.
func (r *Repo) ListCompany(ctx context.Context) ([]inboxautoclaim.Company, error) {
	rows, err := r.db.QueryContext(ctx, getQueryFor(inboxautoclaim.DefaultSource, "auto_claim_company_list"))
	if err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca daftar perusahaan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []inboxautoclaim.Company
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca perusahaan: %w", err)
		}
		result = append(result, inboxautoclaim.Company{
			Code: strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: menelusuri perusahaan: %w", err)
	}
	return result, nil
}

// SummarizeCompany menghitung jumlah batch per perusahaan.
//
// Totalnya dijumlahkan DI SINI dari hasil yang sama, bukan lewat kueri hitung terpisah.
// Sebabnya konsistensi: dua kueri yang berjalan pada saat berbeda dapat melihat data yang
// berbeda, dan hasilnya baris "All" yang tidak sama dengan jumlah irisannya — cacat yang
// hanya muncul saat ada yang mengunggah tepat di antara kedua kueri, dan karena itu nyaris
// mustahil ditelusuri.
func (r *Repo) SummarizeCompany(ctx context.Context, source inboxautoclaim.Source) (inboxautoclaim.Summary, error) {
	rows, err := r.db.QueryContext(ctx, getQueryFor(source, "auto_claim_company_summary"))
	if err != nil {
		return inboxautoclaim.Summary{}, fmt.Errorf(
			"inboxautoclaim/sqlstore: meringkas batch per perusahaan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	summary := inboxautoclaim.Summary{Company: []inboxautoclaim.CompanySummary{}}
	for rows.Next() {
		var (
			code, name sql.NullString
			count      int
		)
		if err := rows.Scan(&code, &name, &count); err != nil {
			return inboxautoclaim.Summary{}, fmt.Errorf(
				"inboxautoclaim/sqlstore: membaca ringkasan: %w", err)
		}
		summary.Company = append(summary.Company, inboxautoclaim.CompanySummary{
			Code:       strings.TrimSpace(code.String),
			Name:       strings.TrimSpace(name.String),
			BatchCount: count,
		})
		summary.Total += count
	}
	if err := rows.Err(); err != nil {
		return inboxautoclaim.Summary{}, fmt.Errorf(
			"inboxautoclaim/sqlstore: menelusuri ringkasan: %w", err)
	}
	return summary, nil
}

// ListLine membaca satu halaman rincian batch.
func (r *Repo) ListLine(ctx context.Context, q inboxautoclaim.LineQuery) (inboxautoclaim.LinePage, error) {
	page := q.Page.Clean()
	company := strings.TrimSpace(q.CompanyCode)
	batch := strings.TrimSpace(q.BatchNumber)

	listName, countName := lineQueryName(q.Result)

	countArgument := []any{company, batch}
	listArgument := []any{company, batch}
	if q.Result != inboxautoclaim.ResultAll {
		countArgument = append(countArgument, inboxautoclaim.MessageSuccess)
		listArgument = append(listArgument, inboxautoclaim.MessageSuccess)
	}
	listArgument = append(listArgument, q.Page.Offset(), page.Size)

	total, err := r.countOne(ctx, q.Source, countName, countArgument...)
	if err != nil {
		return inboxautoclaim.LinePage{}, err
	}

	rows, err := r.db.QueryContext(ctx, getQueryFor(q.Source, listName), listArgument...)
	if err != nil {
		return inboxautoclaim.LinePage{}, fmt.Errorf("inboxautoclaim/sqlstore: membaca rincian batch: %w", err)
	}
	defer func() { _ = rows.Close() }()

	item := make([]inboxautoclaim.Line, 0, page.Size)
	for rows.Next() {
		line, scanErr := scanLine(rows)
		if scanErr != nil {
			return inboxautoclaim.LinePage{}, scanErr
		}
		item = append(item, line)
	}
	if err := rows.Err(); err != nil {
		return inboxautoclaim.LinePage{}, fmt.Errorf("inboxautoclaim/sqlstore: menelusuri rincian batch: %w", err)
	}
	return inboxautoclaim.LinePage{Item: item, Total: total}, nil
}

// ExportLine membaca SELURUH baris yang cocok untuk berkas unduhan.
//
// Ia memakai kueri TERSENDIRI, bukan kueri layar: berkas ekspor punya susunan kolom sendiri
// yang disalin dari InboxAutoClaim/BrowseReportClaimSPK_AutoClaim-SQL.xml, dan kolom
// layar yang tidak ada di berkas tidak perlu ikut dibaca.
func (r *Repo) ExportLine(ctx context.Context, q inboxautoclaim.LineQuery) ([]inboxautoclaim.Line, error) {
	company := strings.TrimSpace(q.CompanyCode)
	batch := strings.TrimSpace(q.BatchNumber)

	name := "auto_claim_export"
	argument := []any{company, batch}
	switch q.Result {
	case inboxautoclaim.ResultSucceeded:
		name = "auto_claim_export_succeeded"
		argument = append(argument, inboxautoclaim.MessageSuccess)
	case inboxautoclaim.ResultFailed:
		name = "auto_claim_export_failed"
		argument = append(argument, inboxautoclaim.MessageSuccess)
	}

	rows, err := r.db.QueryContext(ctx, getQueryFor(q.Source, name), argument...)
	if err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca baris ekspor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []inboxautoclaim.Line
	for rows.Next() {
		var (
			code, policyNo, claimID, acceptanceNo sql.NullString
			claimAmount, causeOfLoss              sql.NullString
			currencyCode, message                 sql.NullString
		)
		if err := rows.Scan(&code, &policyNo, &claimID, &acceptanceNo,
			&claimAmount, &causeOfLoss, &currencyCode, &message); err != nil {
			return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca baris ekspor: %w", err)
		}
		result = append(result, inboxautoclaim.Line{
			CompanyCode:  strings.TrimSpace(code.String),
			BatchNumber:  batch,
			PolicyNo:     strings.TrimSpace(policyNo.String),
			ClaimID:      strings.TrimSpace(claimID.String),
			AcceptanceNo: strings.TrimSpace(acceptanceNo.String),
			ClaimAmount:  strings.TrimSpace(claimAmount.String),
			CauseOfLoss:  strings.TrimSpace(causeOfLoss.String),
			CurrencyCode: strings.TrimSpace(currencyCode.String),
			Message:      strings.TrimSpace(message.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: menelusuri baris ekspor: %w", err)
	}
	return result, nil
}

// BatchExists menyatakan pasangan (perusahaan, batch) benar-benar ada.
func (r *Repo) BatchExists(ctx context.Context, source inboxautoclaim.Source, companyCode, batchNumber string) (bool, error) {
	row := r.db.QueryRowContext(ctx, getQueryFor(source, "auto_claim_batch_exists"),
		strings.TrimSpace(companyCode), strings.TrimSpace(batchNumber))

	var marker sql.NullInt64
	switch err := row.Scan(&marker); {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("inboxautoclaim/sqlstore: memeriksa batch: %w", err)
	default:
		return true, nil
	}
}

// ResolveReceiver menurunkan perusahaan rekanan dari nomor polis.
func (r *Repo) ResolveReceiver(ctx context.Context, policyNo string) (inboxautoclaim.Company, bool, error) {
	row := r.db.QueryRowContext(ctx, getQueryFor(inboxautoclaim.DefaultSource, "auto_claim_receiver"), strings.TrimSpace(policyNo))

	var code, name sql.NullString
	switch err := row.Scan(&code, &name); {
	case err == sql.ErrNoRows:
		return inboxautoclaim.Company{}, false, nil
	case err != nil:
		return inboxautoclaim.Company{}, false,
			fmt.Errorf("inboxautoclaim/sqlstore: mencari penerima klaim: %w", err)
	}

	clean := strings.TrimSpace(code.String)
	if clean == "" {
		// Baris ada tetapi kodenya kosong. Diperlakukan sama dengan tidak ketemu:
		// kode kosong tidak dapat dipakai mengelompokkan batch mana pun.
		return inboxautoclaim.Company{}, false, nil
	}
	return inboxautoclaim.Company{Code: clean, Name: strings.TrimSpace(name.String)}, true, nil
}

// FindPolicyProductSeq mencari PRODKE termutakhir sebuah polis.
func (r *Repo) FindPolicyProductSeq(ctx context.Context, policyNo string) (string, bool, error) {
	row := r.db.QueryRowContext(ctx, getQueryFor(inboxautoclaim.DefaultSource, "auto_claim_policy"), strings.TrimSpace(policyNo))

	var productSeq sql.NullString
	switch err := row.Scan(&productSeq); {
	case err == sql.ErrNoRows:
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("inboxautoclaim/sqlstore: mencari polis: %w", err)
	}

	clean := strings.TrimSpace(productSeq.String)
	return clean, clean != "", nil
}

// InsertUpload menyimpan seluruh baris unggahan dalam SATU transaksi.
//
// # Kenapa satu transaksi untuk seluruh berkas
//
// Unggahan yang tersimpan setengah adalah keadaan terburuk yang mungkin: petugas melihat
// batch yang jumlah barisnya tidak sesuai berkasnya, dan tidak punya cara mengetahui baris
// mana yang hilang. Sistem lama tidak menjamin ini — penyisipannya per baris — tetapi D-68
// sudah memindahkan kepemilikan transaksi ke Go untuk kelas masalah yang sama pada B-4
// dan B-9.
//
// Baris yang GAGAL pemeriksaan tetap ikut disisipkan, dengan pesannya pada IDPEGA,
// NOAKSEPTASI, dan TMP_MESSAGE sekaligus — itu yang dilakukan
// Activity/InsertKlaimToTable_Other-Act.xml.
func (r *Repo) InsertUpload(
	ctx context.Context,
	source inboxautoclaim.Source,
	line []inboxautoclaim.UploadLine,
	uploadedBy string,
) (inboxautoclaim.UploadResult, error) {
	if len(line) == 0 {
		return inboxautoclaim.UploadResult{}, inboxautoclaim.ErrEmptyUpload
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxautoclaim.UploadResult{}, fmt.Errorf("inboxautoclaim/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
	defer func() { _ = tx.Rollback() }()

	order, group := inboxautoclaim.GroupUploadLine(line)

	result := inboxautoclaim.UploadResult{}
	for _, companyCode := range order {
		used, err := usedBatchNumber(ctx, tx, source, companyCode)
		if err != nil {
			return inboxautoclaim.UploadResult{}, err
		}
		batchNumber := inboxautoclaim.NextBatchNumber(used)

		reference := inboxautoclaim.BatchRef{CompanyCode: companyCode, BatchNumber: batchNumber}
		for _, l := range group[companyCode] {
			// Ketiga kolom ini menerima nilai yang SAMA: pesan galat bila baris gagal,
			// NULL bila lolos. Yang NULL itulah yang membuat barisnya terambil pemrosesan.
			mark := nullable(l.Message)

			if _, err := tx.ExecContext(ctx, getQueryFor(source, "auto_claim_line_insert"),
				batchNumber,
				companyCode,
				l.Row.PolicyNo,
				nullable(l.ProductSeq),
				uploadedBy,
				mark, // IDPEGA
				l.Row.DateOfLoss,
				l.Row.ReportDate,
				nil, // CURRENCY — dari snapshot polis, menunggu B-1
				nullable(l.Row.CauseOfLoss),
				l.Row.ClaimAmount,
				nullable(l.Row.Reason),
				nullable(l.Row.Keyword),
				mark, // TMP_MESSAGE
				mark, // NOAKSEPTASI
				nullable(l.Row.ObjectName),
				nullable(l.Row.FlagNoPayout),
			); err != nil {
				return inboxautoclaim.UploadResult{}, fmt.Errorf(
					"inboxautoclaim/sqlstore: menyisipkan baris %d: %w", l.Row.LineNumber, err)
			}

			reference.Rows++
			if l.Accepted() {
				reference.Succeeded++
			} else {
				reference.Failed++
			}
		}

		result.Batch = append(result.Batch, reference)
		result.Rows += reference.Rows
	}

	if err := tx.Commit(); err != nil {
		return inboxautoclaim.UploadResult{}, fmt.Errorf("inboxautoclaim/sqlstore: menutup transaksi unggah: %w", err)
	}
	return result, nil
}

// CheckTable memastikan ketiga tabel yang dipakai modul ini dapat dibaca akun aplikasi.
//
// POOLDATA.CURRENCY ikut diperiksa walau hanya dipakai lookup: tanpa hak baca padanya,
// kolom mata uang kosong di layar DETAIL dan di KEDUA berkas ekspor — dan kosongnya tidak
// menghasilkan galat apa pun.
func (r *Repo) CheckTable(ctx context.Context) error {
	type pemeriksaan struct {
		source inboxautoclaim.Source
		name   string
		table  string
	}

	// KETIGA tabel batch diperiksa, satu per tab. Memeriksa satu saja akan melaporkan
	// "siap" sementara dua tab lain menolak setiap permintaan — dan pengguna baru
	// menemukannya saat membuka tab itu.
	check := []pemeriksaan{}
	for _, source := range inboxautoclaim.AllSource() {
		info, exists := source.Info()
		if !exists {
			continue
		}
		check = append(check, pemeriksaan{source, "auto_claim_check_table", info.Table})
	}
	check = append(check,
		pemeriksaan{inboxautoclaim.DefaultSource, "auto_claim_check_master", "POOLDATA.M_AUTO_CLAIM_PNC"},
		pemeriksaan{inboxautoclaim.DefaultSource, "auto_claim_check_currency", "POOLDATA.CURRENCY"},
	)

	for _, c := range check {
		rows, err := r.db.QueryContext(ctx, getQueryFor(c.source, c.name))
		if err != nil {
			return fmt.Errorf("inboxautoclaim/sqlstore: %s tidak dapat dibaca: %w", c.table, err)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inboxautoclaim/sqlstore: %s tidak dapat dibaca: %w", c.table, err)
		}
		_ = rows.Close()
	}
	return nil
}

// lineQueryName memilih pasangan kueri daftar dan hitungan sesuai penyaring hasil.
func lineQueryName(result inboxautoclaim.Result) (string, string) {
	switch result {
	case inboxautoclaim.ResultSucceeded:
		return "auto_claim_line_list_succeeded", "auto_claim_line_count_succeeded"
	case inboxautoclaim.ResultFailed:
		return "auto_claim_line_list_failed", "auto_claim_line_count_failed"
	default:
		return "auto_claim_line_list", "auto_claim_line_count"
	}
}

// usedBatchNumber membaca nomor batch yang sudah dipakai satu perusahaan.
func usedBatchNumber(ctx context.Context, tx *sql.Tx, source inboxautoclaim.Source, companyCode string) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQueryFor(source, "auto_claim_batch_number_used"), companyCode)
	if err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca nomor batch terpakai: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var used []string
	for rows.Next() {
		var number sql.NullString
		if err := rows.Scan(&number); err != nil {
			return nil, fmt.Errorf("inboxautoclaim/sqlstore: membaca nomor batch: %w", err)
		}
		used = append(used, strings.TrimSpace(number.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxautoclaim/sqlstore: menelusuri nomor batch: %w", err)
	}
	return used, nil
}

// countOne menjalankan kueri hitungan bernama tertentu.
func (r *Repo) countOne(ctx context.Context, source inboxautoclaim.Source, name string, argument ...any) (int, error) {
	var total sql.NullInt64
	if err := r.db.QueryRowContext(ctx, getQueryFor(source, name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("inboxautoclaim/sqlstore: menghitung (%s): %w", name, err)
	}
	return int(total.Int64), nil
}

type scanner interface {
	Scan(target ...any) error
}

// scanBatch membaca satu baris hasil kueri daftar batch.
//
// Kolom teks dibaca lewat sql.NullString lalu dipangkas: NAMA_PENERIMA dapat NULL karena
// join-nya LEFT, dan USERINPUT dapat NULL pada baris lama.
func scanBatch(p scanner) (inboxautoclaim.Batch, error) {
	var (
		code, name, number, uploadedBy       sql.NullString
		processedDate                        sql.NullTime
		uploaded, processed, succeeded, fail sql.NullInt64
	)
	if err := p.Scan(&code, &name, &number, &processedDate,
		&uploaded, &processed, &succeeded, &fail, &uploadedBy); err != nil {
		return inboxautoclaim.Batch{}, fmt.Errorf("inboxautoclaim/sqlstore: membaca baris batch: %w", err)
	}
	return inboxautoclaim.Batch{
		CompanyCode:   strings.TrimSpace(code.String),
		CompanyName:   strings.TrimSpace(name.String),
		BatchNumber:   strings.TrimSpace(number.String),
		ProcessedDate: formatDate(processedDate),
		Uploaded:      int(uploaded.Int64),
		Processed:     int(processed.Int64),
		Succeeded:     int(succeeded.Int64),
		Failed:        int(fail.Int64),
		UploadedBy:    strings.TrimSpace(uploadedBy.String),
	}, nil
}

// scanLine membaca satu baris hasil kueri rincian.
func scanLine(p scanner) (inboxautoclaim.Line, error) {
	var (
		code, batch, policyNo, productSeq   sql.NullString
		claimID, acceptanceNo               sql.NullString
		currency, currencyCode, claimAmount sql.NullString
		causeOfLoss, dateOfLoss, reportDate sql.NullString
		note, keyword, objectName, noPayout sql.NullString
		message, uploadedBy                 sql.NullString
		processedAt                         sql.NullTime
	)
	if err := p.Scan(&code, &batch, &policyNo, &productSeq, &claimID, &acceptanceNo,
		&currency, &currencyCode, &claimAmount, &causeOfLoss,
		&dateOfLoss, &reportDate, &processedAt,
		&note, &keyword, &objectName, &noPayout, &message, &uploadedBy); err != nil {
		return inboxautoclaim.Line{}, fmt.Errorf("inboxautoclaim/sqlstore: membaca baris rincian: %w", err)
	}
	return inboxautoclaim.Line{
		CompanyCode:   strings.TrimSpace(code.String),
		BatchNumber:   strings.TrimSpace(batch.String),
		PolicyNo:      strings.TrimSpace(policyNo.String),
		ProductSeq:    strings.TrimSpace(productSeq.String),
		ClaimID:       strings.TrimSpace(claimID.String),
		AcceptanceNo:  strings.TrimSpace(acceptanceNo.String),
		Currency:      strings.TrimSpace(currency.String),
		CurrencyCode:  strings.TrimSpace(currencyCode.String),
		ClaimAmount:   strings.TrimSpace(claimAmount.String),
		CauseOfLoss:   strings.TrimSpace(causeOfLoss.String),
		DateOfLoss:    strings.TrimSpace(dateOfLoss.String),
		ReportDate:    strings.TrimSpace(reportDate.String),
		ProcessedDate: formatDate(processedAt),
		Note:          strings.TrimSpace(note.String),
		Keyword:       strings.TrimSpace(keyword.String),
		ObjectName:    strings.TrimSpace(objectName.String),
		FlagNoPayout:  strings.TrimSpace(noPayout.String),
		Message:       strings.TrimSpace(message.String),
		UploadedBy:    strings.TrimSpace(uploadedBy.String),
	}, nil
}

// formatDate mengubah TIMESTAMP menjadi teks dd/mm/yyyy, atau kosong bila NULL.
//
// Zona waktunya TIDAK dikonversi. TGLPROSES ditulis basis data dengan jam basis data, dan
// yang ditampilkan Pega pun nilai itu apa adanya (`TO_CHAR(a.TGLPROSES,'DD/MM/YYYY')`).
// Mengonversinya ke WIB di sini akan menggeser tanggalnya pada baris yang dibuat menjelang
// tengah malam — dan menghasilkan selisih terhadap Pega yang tidak berasal dari perbaikan
// apa pun.
func formatDate(value sql.NullTime) string {
	if !value.Valid || value.Time.IsZero() {
		return ""
	}
	return value.Time.Format(dateLayout)
}

// nullable mengubah teks kosong menjadi NULL.
//
// Kolom yang diisi teks kosong dan kolom yang NULL terlihat sama di layar, tetapi TIDAK
// sama bagi kueri yang memeriksanya dengan IS NULL — dan pemrosesan batch memeriksa persis
// begitu (`GroupingAutoClaim2`: `idpega is null`). Menyimpan teks kosong pada kolom yang
// ditunggu NULL akan membuat barisnya tidak pernah terambil.
func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan.
// resolved memuat kueri yang PLACEHOLDER-nya sudah diisi, satu salinan per tab.
//
// Dibangun EAGER saat paket dimuat, bukan saat permintaan datang: placeholder yang salah
// ketik atau tab yang belum punya isian akan membuat aplikasi gagal START, bukan gagal di
// tangan pengguna. Itu aturan yang sama dengan konfigurasi wajib (11-CROSSCUTTING §3.1).
var resolved = resolveAllQueries()

// resolveAllQueries mengisi {{TABEL}} dan {{KOLOM}} untuk ketiga tab.
//
// # Kenapa teks SQL-nya disusun, padahal merangkai SQL dilarang
//
// Yang dilarang 08-TECHNICAL-STRATEGY.md §4.3 adalah merangkai SQL dari NILAI — pola
// {ASIS:...} warisan yang menyisipkan isi properti klipboard mentah-mentah. Yang diisi di
// sini bukan nilai melainkan NAMA TABEL dan NAMA KOLOM, dan keduanya datang dari enum
// tertutup berisi tiga anggota di dalam kode. Tidak ada satu pun jalur dari masukan
// pengguna ke sini: tab yang tidak dikenal DITOLAK domain sebelum sampai ke lapisan ini.
//
// # Kenapa tidak disalin tiga kali saja
//
// Pega melakukannya: BrowseClaimSPKAutoClaim, BrowseClaimSPKClaimKredit, dan
// BrowseClaimSPKTravel adalah tiga rule berisi kueri yang sama persis kecuali nama tabel
// dan satu nama kolom. 03-CURRENT-ARCHITECTURE.md §4.6 mencatat pola itu sebagai utang
// teknis: "satu perubahan aturan harus diterapkan di empat tempat — dan sering hanya
// diterapkan di sebagian". Menyalinnya tiga kali di sini berarti memindahkan cacatnya,
// bukan sistemnya.
func resolveAllQueries() map[string]string {
	result := map[string]string{}
	for _, source := range inboxautoclaim.AllSource() {
		info, exists := source.Info()
		if !exists {
			panic("inboxautoclaim/sqlstore: tab tanpa tabel: " + string(source))
		}
		pengganti := strings.NewReplacer("{{TABEL}}", info.Table, "{{KOLOM}}", info.CompanyColumn)
		for name, text := range query {
			isi := pengganti.Replace(text)
			if strings.Contains(isi, "{{") {
				panic(fmt.Sprintf(
					"inboxautoclaim/sqlstore: kueri %q tab %q masih memuat placeholder yang tidak dikenal",
					name, source))
			}
			result[string(source)+"::"+name] = isi
		}
	}
	return result
}

// getQueryFor mengembalikan kueri yang sudah diisi untuk satu tab.
func getQueryFor(source inboxautoclaim.Source, name string) string {
	text, exists := resolved[string(source)+"::"+name]
	if !exists {
		panic(fmt.Sprintf("inboxautoclaim/sqlstore: kueri %q tidak ada untuk tab %q", name, source))
	}
	return text
}

func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf("inboxautoclaim/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("inboxautoclaim/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("inboxautoclaim/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("inboxautoclaim/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, line := range body {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				continue
			}
			statement = append(statement, line)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, line := range strings.Split(content, "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, line)
	}
	save()
	return result
}

// Jaminan waktu kompilasi bahwa seluruh seam terpenuhi.
var _ inboxautoclaim.Repo = (*Repo)(nil)
