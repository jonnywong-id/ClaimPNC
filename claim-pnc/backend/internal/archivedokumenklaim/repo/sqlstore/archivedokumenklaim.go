package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/archivedokumenklaim"
)

// Repo membaca dan menulis POOLDATA.T_CLAIM_ARCHIVE_FILE.
//
// # Siapa pemilik tabel ini
//
// Layar Archive Dokumen Klaim adalah SATU-SATUNYA penulisnya di sistem lama: hanya
// `RDB List/InsertToClaimArchive-SQL.xml` yang memanggil prosedur penyisipannya, dan
// hanya `UpdateDataArchiveKlaimSetelahService` serta pySaveSQL
// `SearchDataArchiveFillingCase` yang mengubahnya — ketiganya dipanggil dari layar ini.
// Memindahkan layarnya karena itu memindahkan kepemilikan tabelnya secara utuh, dan `P-1`
// terpenuhi tanpa negosiasi (`D-21`).
//
// Tiga tabel lain HANYA DIBACA: T_CLAIM_PNC, V_LST_DOC_TYPE, dan V_LST_DET_TYPE_DOC —
// ketiganya milik modul lain, dan tidak satu pun kueri di sini menulisnya.
//
// Pemetaan nama menyesatkan, kelima perubahan terhadap kueri lama, dan alasan
// masing-masing ada di kepala archivedokumenklaim.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// Search membaca satu halaman grid ARCHIVE FILE KLAIM.
func (r *Repo) Search(
	ctx context.Context,
	criteria archivedokumenklaim.Criteria,
	page archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	name, args := searchQuery(criteria)
	return r.listPage(ctx, name, args, page)
}

// PendingBranch membaca berkas yang belum dikirim ke layanan Arsip.
func (r *Repo) PendingBranch(
	ctx context.Context,
	scope archivedokumenklaim.BranchScope,
	page archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	name, args, err := pendingQuery(scope)
	if err != nil {
		return archivedokumenklaim.ArchivePage{}, err
	}
	return r.listPage(ctx, name, args, page)
}

// searchQuery memilih kueri dan menyusun parameternya menurut mode pencarian.
//
// Kata kunci yang sama diisikan ke TIGA penanda berbeda — bukan karena nilainya berbeda,
// melainkan karena satu penanda yang dipakai berulang berperilaku berbeda antar driver.
func searchQuery(criteria archivedokumenklaim.Criteria) (string, []any) {
	if criteria.Mode == archivedokumenklaim.ModeInputDate {
		from := *criteria.From

		// Batas atas digeser satu hari dan dibuat eksklusif, sehingga berkas yang diinput
		// pukul berapa pun pada tanggal akhir tetap ikut — persis `trunc(TGLINPUT) <=`
		// pada kueri lama, tetapi tanpa mematikan index. Lihat archivedokumenklaim.sql.
		until := criteria.To.AddDate(0, 0, 1)

		return "search_input_date", []any{from, until}
	}

	keyword := criteria.Keyword
	return "search_keyword", []any{keyword, keyword, keyword}
}

// pendingQuery memilih kueri daftar kirim ke cabang menurut cakupan lini bisnis.
//
// Ketiga bentuknya ditulis sebagai kueri terpisah, bukan satu kueri yang klausanya
// dirangkai menurut panjang daftar. Merangkainya akan menghidupkan kembali pola yang
// justru dihapus dari modul ini — dan tiga bentuk adalah seluruh kemungkinan yang ada,
// karena `Activity/GetDataArchiveCabangKlaim-Act.xml` hanya mengenal tiga jabatan.
func pendingQuery(scope archivedokumenklaim.BranchScope) (string, []any, error) {
	switch len(scope.ExcludedGroupPanels) {
	case 0:
		return "pending_all", nil, nil
	case 1:
		return "pending_exclude_one", []any{scope.ExcludedGroupPanels[0]}, nil
	case 2:
		return "pending_exclude_two", []any{
			scope.ExcludedGroupPanels[0],
			scope.ExcludedGroupPanels[1],
		}, nil
	default:
		return "", nil, fmt.Errorf(
			"archivedokumenklaim/sqlstore: %d lini bisnis dikecualikan, kueri hanya mengenal 0..2",
			len(scope.ExcludedGroupPanels))
	}
}

// listPage menjalankan satu kueri daftar beserta penghitungnya.
func (r *Repo) listPage(
	ctx context.Context,
	name string,
	args []any,
	page archivedokumenklaim.Pagination,
) (archivedokumenklaim.ArchivePage, error) {
	page = page.Normalize()

	var total int
	if err := r.db.QueryRowContext(ctx, counted(name), args...).Scan(&total); err != nil {
		return archivedokumenklaim.ArchivePage{}, fmt.Errorf("menghitung berkas arsip: %w", err)
	}

	// Halaman yang seluruhnya berada di luar hasil tidak perlu menembak basis data lagi.
	// Ia bukan galat — pengguna dapat sampai ke sana dengan mengubah alamat, atau dengan
	// berada di halaman 5 saat data berkurang.
	if total == 0 || page.Offset() >= total {
		return archivedokumenklaim.ArchivePage{
			Files:      []archivedokumenklaim.ArchiveFile{},
			Total:      total,
			Pagination: page,
		}, nil
	}

	pagedArgs := append(append([]any{}, args...), page.Offset(), page.Size)

	rows, err := r.db.QueryContext(ctx, paged(name), pagedArgs...)
	if err != nil {
		return archivedokumenklaim.ArchivePage{}, fmt.Errorf("membaca berkas arsip: %w", err)
	}
	defer rows.Close()

	files := make([]archivedokumenklaim.ArchiveFile, 0, page.Size)
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return archivedokumenklaim.ArchivePage{}, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return archivedokumenklaim.ArchivePage{}, fmt.Errorf("membaca berkas arsip: %w", err)
	}

	return archivedokumenklaim.ArchivePage{
		Files:      files,
		Total:      total,
		Pagination: page,
	}, nil
}

// FindByID membaca satu berkas arsip.
func (r *Repo) FindByID(
	ctx context.Context,
	id int64,
) (archivedokumenklaim.ArchiveFile, bool, error) {
	rows, err := r.db.QueryContext(ctx, query("find_by_id"), id)
	if err != nil {
		return archivedokumenklaim.ArchiveFile{}, false, fmt.Errorf("membaca berkas arsip: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return archivedokumenklaim.ArchiveFile{}, false,
				fmt.Errorf("membaca berkas arsip: %w", err)
		}
		return archivedokumenklaim.ArchiveFile{}, false, nil
	}

	file, err := scanFile(rows)
	if err != nil {
		return archivedokumenklaim.ArchiveFile{}, false, err
	}
	return file, true, nil
}

// SearchClaims mencari calon klaim yang berkasnya hendak diarsipkan.
func (r *Repo) SearchClaims(
	ctx context.Context,
	criteria archivedokumenklaim.ClaimCriteria,
) ([]archivedokumenklaim.ClaimCandidate, error) {
	name := "search_claim_any"
	args := []any{}

	value := strings.ToUpper(criteria.Value)

	if criteria.Type == archivedokumenklaim.ClaimByPolicy {
		name = "search_claim_by_policy"
		args = append(args, value)
	} else {
		args = append(args, value, value, value)
	}

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, fmt.Errorf("mencari klaim: %w", err)
	}
	defer rows.Close()

	candidates := make([]archivedokumenklaim.ClaimCandidate, 0, 16)
	for rows.Next() {
		candidate, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mencari klaim: %w", err)
	}

	return candidates, nil
}

// Save menyimpan satu berkas arsip.
//
// # Kenapa dibungkus transaksi meski hanya satu pernyataan yang menulis
//
// Penyisipan menempuh DUA pernyataan: membaca nomor berikutnya, lalu menyisipkan
// barisnya. Keduanya wajib berada di satu transaksi, karena di antara keduanya terdapat
// celah yang menjadi cacat penomoran sistem lama.
//
// Transaksinya TIDAK menutup celah itu — nilai `MAX(ID_ARCHIVE)+1` tetap dapat terbaca
// sama oleh dua penyimpanan yang berjalan bersamaan pada tingkat isolasi bawaan Oracle.
// Ia direplikasi apa adanya atas keputusan Work Owner 2026-09-24; lihat kueri
// `next_archive_id`.
func (r *Repo) Save(ctx context.Context, draft archivedokumenklaim.Draft) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("membuka transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
	// ini, jalur galat mana pun meninggalkan transaksi yang menggantung.
	defer func() { _ = tx.Rollback() }()

	id := draft.ID

	if draft.IsNew() {
		if err := tx.QueryRowContext(ctx, query("next_archive_id")).Scan(&id); err != nil {
			return 0, fmt.Errorf("menentukan nomor arsip berikutnya: %w", err)
		}

		_, err = tx.ExecContext(ctx, query("insert_archive"),
			id,
			draft.ClaimNumber,
			nullable(draft.PolicyNumber),
			nullable(draft.InsuredName),
			nullableTime(draft.LossDate),
			nullable(draft.TechnicalPIC),
			nullableTime(draft.DocumentReceivedDate),
			draft.SheetCount,
			draft.DocumentTypeCode,
			draft.DocumentKindCode,
			draft.BoxName,
			draft.FillingCode,
			draft.InputUser,
			nullable(draft.BranchCode),
			nullable(draft.GroupPanel),
		)
		if err != nil {
			return 0, fmt.Errorf("menyisipkan berkas arsip: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, query("update_archive"),
			draft.ClaimNumber,
			nullable(draft.PolicyNumber),
			nullable(draft.InsuredName),
			nullableTime(draft.LossDate),
			nullable(draft.TechnicalPIC),
			nullableTime(draft.DocumentReceivedDate),
			draft.SheetCount,
			draft.DocumentTypeCode,
			draft.DocumentKindCode,
			draft.BoxName,
			draft.FillingCode,
			nullable(draft.GroupPanel),
			draft.ID,
		)
		if err != nil {
			return 0, fmt.Errorf("mengubah berkas arsip: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("menyimpan berkas arsip: %w", err)
	}
	return id, nil
}

// StoreReceipt menyimpan jawaban layanan Arsip TANPA menandai barisnya terkirim.
//
// Alasan ia terpisah dari MarkSent ada di seam archivedokumenklaim.Repo dan di kueri
// `store_receipt`. Singkatnya: jalur Simpan tidak menandai, jalur Dokument Cabang
// menandai — dan perbedaan itu yang membuat berkasnya terkirim dua kali di sistem lama.
func (r *Repo) StoreReceipt(ctx context.Context, receipt archivedokumenklaim.Receipt) error {
	return r.applyReceipt(ctx, "store_receipt", receipt)
}

// MarkSent menyimpan jawaban layanan Arsip dan menandai barisnya sudah dikirim.
func (r *Repo) MarkSent(ctx context.Context, receipt archivedokumenklaim.Receipt) error {
	return r.applyReceipt(ctx, "mark_sent", receipt)
}

// applyReceipt menjalankan salah satu dari kedua pernyataan penyimpanan jawaban.
//
// Keduanya menerima parameter yang sama persis dan berbeda hanya pada satu kolom, sehingga
// menuliskannya dua kali berarti dua tempat yang harus berubah berpasangan.
func (r *Repo) applyReceipt(
	ctx context.Context,
	name string,
	receipt archivedokumenklaim.Receipt,
) error {
	result, err := r.db.ExecContext(ctx, query(name),
		nullable(receipt.Code),
		nullable(receipt.Note),
		nullable(receipt.Request),
		receipt.SentAt,
		receipt.ID,
	)
	if err != nil {
		return fmt.Errorf("menyimpan jawaban layanan Arsip: %w", err)
	}

	// UPDATE yang tidak mengenai satu baris pun TIDAK gagal di Oracle. Tanpa pemeriksaan
	// ini, berkas yang barisnya sudah hilang tetap dilaporkan berhasil dikirim — padahal
	// jawaban layanan Arsip tidak tersimpan di mana pun.
	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkannya bukan alasan menggagalkan penyimpanan
		// yang sudah berlangsung.
		return nil
	}
	if affected == 0 {
		return archivedokumenklaim.ErrNotFound
	}
	return nil
}

// DocumentTypes membaca pilihan Tipe Dokumen.
func (r *Repo) DocumentTypes(
	ctx context.Context,
) ([]archivedokumenklaim.DocumentTypeOption, error) {
	rows, err := r.db.QueryContext(ctx, query("document_types"))
	if err != nil {
		return nil, fmt.Errorf("membaca tipe dokumen: %w", err)
	}
	defer rows.Close()

	options := make([]archivedokumenklaim.DocumentTypeOption, 0, 32)
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("membaca tipe dokumen: %w", err)
		}
		options = append(options, archivedokumenklaim.DocumentTypeOption{
			Code: strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca tipe dokumen: %w", err)
	}
	return options, nil
}

// DocumentKinds membaca pilihan Jenis Dokumen beserta tipe pemiliknya.
func (r *Repo) DocumentKinds(
	ctx context.Context,
) ([]archivedokumenklaim.DocumentKindOption, error) {
	rows, err := r.db.QueryContext(ctx, query("document_kinds"))
	if err != nil {
		return nil, fmt.Errorf("membaca jenis dokumen: %w", err)
	}
	defer rows.Close()

	options := make([]archivedokumenklaim.DocumentKindOption, 0, 64)
	for rows.Next() {
		var code, name, typeCode sql.NullString
		if err := rows.Scan(&code, &name, &typeCode); err != nil {
			return nil, fmt.Errorf("membaca jenis dokumen: %w", err)
		}
		options = append(options, archivedokumenklaim.DocumentKindOption{
			Code:     strings.TrimSpace(code.String),
			Name:     strings.TrimSpace(name.String),
			TypeCode: strings.TrimSpace(typeCode.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca jenis dokumen: %w", err)
	}
	return options, nil
}

// FillingCodes membaca isi pemilih "Pilih Kode".
func (r *Repo) FillingCodes(
	ctx context.Context,
	keyword string,
) ([]archivedokumenklaim.FillingCodeOption, error) {
	pattern := likePattern(keyword)

	rows, err := r.db.QueryContext(ctx, query("filling_codes"), pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("membaca kode filling: %w", err)
	}
	defer rows.Close()

	options := make([]archivedokumenklaim.FillingCodeOption, 0, 32)
	for rows.Next() {
		var (
			code    sql.NullString
			boxName sql.NullString
			usage   sql.NullInt64
		)
		if err := rows.Scan(&code, &boxName, &usage); err != nil {
			return nil, fmt.Errorf("membaca kode filling: %w", err)
		}
		options = append(options, archivedokumenklaim.FillingCodeOption{
			Code:       strings.TrimSpace(code.String),
			BoxName:    strings.TrimSpace(boxName.String),
			UsageCount: int(usage.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca kode filling: %w", err)
	}
	return options, nil
}

// scanFile membaca satu baris kueri daftar arsip.
//
// Seluruh kolom teks dibaca sebagai sql.NullString dan seluruh tanggal sebagai
// sql.NullTime: kolomnya nullable di skema warisan — DDL-nya tidak ada di export
// (`R-08`), sehingga tidak satu pun dapat diandalkan berisi nilai. Membacanya langsung ke
// string akan gagal dengan galat konversi pada baris pertama yang kosong.
func scanFile(rows *sql.Rows) (archivedokumenklaim.ArchiveFile, error) {
	var (
		id               sql.NullInt64
		claimNumber      sql.NullString
		policyNumber     sql.NullString
		insuredName      sql.NullString
		lossDate         sql.NullTime
		technicalPIC     sql.NullString
		receivedDate     sql.NullTime
		inputDate        sql.NullTime
		sheetCount       sql.NullInt64
		documentTypeCode sql.NullString
		documentTypeName sql.NullString
		documentKindCode sql.NullString
		documentKindName sql.NullString
		boxName          sql.NullString
		fillingCode      sql.NullString
		inputUser        sql.NullString
		sentDate         sql.NullTime
		groupPanel       sql.NullString
		branchStatus     sql.NullString
		serviceCode      sql.NullString
		serviceNote      sql.NullString
	)

	if err := rows.Scan(
		&id, &claimNumber, &policyNumber, &insuredName, &lossDate,
		&technicalPIC, &receivedDate, &inputDate, &sheetCount,
		&documentTypeCode, &documentTypeName, &documentKindCode, &documentKindName,
		&boxName, &fillingCode, &inputUser, &sentDate,
		&groupPanel, &branchStatus, &serviceCode, &serviceNote,
	); err != nil {
		return archivedokumenklaim.ArchiveFile{}, fmt.Errorf("membaca baris berkas arsip: %w", err)
	}

	return archivedokumenklaim.ArchiveFile{
		ID:                   id.Int64,
		ClaimNumber:          strings.TrimSpace(claimNumber.String),
		PolicyNumber:         strings.TrimSpace(policyNumber.String),
		InsuredName:          strings.TrimSpace(insuredName.String),
		LossDate:             timeOrNil(lossDate),
		TechnicalPIC:         strings.TrimSpace(technicalPIC.String),
		DocumentReceivedDate: timeOrNil(receivedDate),
		InputDate:            timeOrNil(inputDate),
		SheetCount:           int(sheetCount.Int64),
		DocumentTypeCode:     strings.TrimSpace(documentTypeCode.String),
		DocumentTypeName:     strings.TrimSpace(documentTypeName.String),
		DocumentKindCode:     strings.TrimSpace(documentKindCode.String),
		DocumentKindName:     strings.TrimSpace(documentKindName.String),
		BoxName:              strings.TrimSpace(boxName.String),
		FillingCode:          strings.TrimSpace(fillingCode.String),
		InputUser:            strings.TrimSpace(inputUser.String),
		SentDate:             timeOrNil(sentDate),
		GroupPanel:           strings.TrimSpace(groupPanel.String),
		BranchStatus:         strings.TrimSpace(branchStatus.String),
		ServiceCode:          strings.TrimSpace(serviceCode.String),
		ServiceNote:          strings.TrimSpace(serviceNote.String),
	}, nil
}

// scanCandidate membaca satu baris hasil pencarian klaim.
func scanCandidate(rows *sql.Rows) (archivedokumenklaim.ClaimCandidate, error) {
	var (
		number        sql.NullString
		policyNumber  sql.NullString
		insuredName   sql.NullString
		lossDate      sql.NullTime
		businessName  sql.NullString
		branchName    sql.NullString
		workStatus    sql.NullString
		claimPosition sql.NullString
		closeDate     sql.NullTime
		closeNote     sql.NullString
		technicalPIC  sql.NullString
		groupPanel    sql.NullString
	)

	if err := rows.Scan(
		&number, &policyNumber, &insuredName, &lossDate,
		&businessName, &branchName, &workStatus, &claimPosition,
		&closeDate, &closeNote, &technicalPIC, &groupPanel,
	); err != nil {
		return archivedokumenklaim.ClaimCandidate{}, fmt.Errorf("membaca baris klaim: %w", err)
	}

	return archivedokumenklaim.ClaimCandidate{
		Number:        strings.TrimSpace(number.String),
		PolicyNumber:  strings.TrimSpace(policyNumber.String),
		InsuredName:   strings.TrimSpace(insuredName.String),
		LossDate:      timeOrNil(lossDate),
		BusinessName:  strings.TrimSpace(businessName.String),
		BranchName:    strings.TrimSpace(branchName.String),
		WorkStatus:    strings.TrimSpace(workStatus.String),
		ClaimPosition: strings.TrimSpace(claimPosition.String),
		CloseDate:     timeOrNil(closeDate),
		CloseNote:     strings.TrimSpace(closeNote.String),
		TechnicalPIC:  strings.TrimSpace(technicalPIC.String),
		GroupPanel:    strings.TrimSpace(groupPanel.String),
	}, nil
}

// timeOrNil mengubah sql.NullTime menjadi pointer.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	moment := value.Time.UTC()
	return &moment
}

// nullable mengubah teks kosong menjadi NULL basis data.
//
// Kolom warisan membedakan keduanya, dan menulis teks kosong ke kolom yang selama ini
// NULL membuat baris terbitan sistem baru terlihat berbeda dari baris lama pada setiap
// kueri yang memakai `IS NULL`.
func nullable(value string) any {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return nil
	}
	return clean
}

// nullableTime mengubah tanggal yang tidak diisi menjadi NULL basis data.
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

// likePattern menyusun pola LIKE yang huruf besar-kecilnya sudah diseragamkan, dengan
// karakter jokernya dilucuti.
//
// Tanpa pelucutan itu, kode filling yang memuat `%` atau `_` berubah menjadi pola
// pencarian — dan `_` cukup lazim di kode arsip. ESCAPE-nya dinyatakan di kuerinya.
func likePattern(keyword string) string {
	clean := strings.TrimSpace(keyword)
	if clean == "" {
		return "%"
	}
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + strings.ToUpper(replacer.Replace(clean)) + "%"
}

// Repo wajib memenuhi seam modul. Pernyataan ini membuat ketidakcocokan terbaca saat
// kompilasi, bukan saat perakitan di cmd.
var _ archivedokumenklaim.Repo = (*Repo)(nil)
