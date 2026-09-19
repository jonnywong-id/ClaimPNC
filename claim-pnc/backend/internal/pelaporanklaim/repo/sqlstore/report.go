package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/pelaporanklaim"
)

// Repo membaca dan menulis POOLDATA.CPNC_LAPORAN_KLAIM.
//
// Tabel itu MILIK APLIKASI INI sepenuhnya — dibuat migrasi `0003`, tidak dibaca dan tidak
// ditulis Pega. `P-1` karena itu terpenuhi tanpa perlu negosiasi kepemilikan: tidak ada
// sistem lain yang menyentuhnya.
//
// Alasan tabelnya baru, dan apa yang ditinggalkan bersama tabel lama, ada di kepala
// report.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// PrimaryKeyName dipakai menerjemahkan galat bentrok menjadi galat domain.
//
// Ia konstanta supaya kode dan migrasi tidak dapat berbeda pendapat diam-diam. Mengganti
// namanya di migrasi tanpa mengganti konstanta ini akan membuat bentrok nomor muncul
// sebagai galat `500` alih-alih pesan yang dapat dibaca.
const PrimaryKeyName = "CPNC_LAPORAN_KLAIM_PK"

// List membaca satu halaman laporan beserta jumlah seluruh yang cocok.
func (r *Repo) List(ctx context.Context, f pelaporanklaim.Filter) (pelaporanklaim.Page, error) {
	f = f.Normalize()
	filters := filterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("report_count"), filters...).Scan(&total); err != nil {
		return pelaporanklaim.Page{}, fmt.Errorf("pelaporanklaim/sqlstore: menghitung laporan: %w", err)
	}

	args := append(append([]any(nil), filters...), f.Offset, f.Limit)
	rows, err := r.db.QueryContext(ctx, query("report_list"), args...)
	if err != nil {
		return pelaporanklaim.Page{}, fmt.Errorf("pelaporanklaim/sqlstore: membaca daftar laporan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	reports := make([]pelaporanklaim.ClaimReport, 0, f.Limit)
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return pelaporanklaim.Page{}, fmt.Errorf("pelaporanklaim/sqlstore: membaca baris laporan: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return pelaporanklaim.Page{}, fmt.Errorf("pelaporanklaim/sqlstore: menelusuri daftar laporan: %w", err)
	}

	return pelaporanklaim.Page{Reports: reports, Total: total}, nil
}

// Summary menghitung jumlah laporan per tahap dalam satu perjalanan ke basis data.
func (r *Repo) Summary(ctx context.Context, f pelaporanklaim.Filter) (pelaporanklaim.StageSummary, error) {
	f = f.Normalize()

	branch := nilIfEmpty(f.BranchCode)
	search := nilIfEmpty(searchPattern(f.Search))
	args := []any{
		branch, branch,
		search, search, search, search, search, search,
	}

	rows, err := r.db.QueryContext(ctx, query("report_summary"), args...)
	if err != nil {
		return nil, fmt.Errorf("pelaporanklaim/sqlstore: menghitung ringkasan tahap: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Kelima tahap dimulai dari nol, bukan dari map kosong. Tahap yang tidak punya satu
	// pun baris TIDAK muncul di hasil GROUP BY, dan tanpa nilai awal lencananya akan
	// hilang dari layar alih-alih menunjukkan angka nol.
	summary := pelaporanklaim.StageSummary{
		pelaporanklaim.StageNotTransferred: 0,
		pelaporanklaim.StageNotRegistered:  0,
		pelaporanklaim.StageRegistered:     0,
		pelaporanklaim.StageAccepted:       0,
		pelaporanklaim.StageRejected:       0,
	}
	for rows.Next() {
		var (
			stage string
			count int
		)
		if err := rows.Scan(&stage, &count); err != nil {
			return nil, fmt.Errorf("pelaporanklaim/sqlstore: membaca baris ringkasan: %w", err)
		}
		summary[pelaporanklaim.Stage(strings.TrimSpace(stage))] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pelaporanklaim/sqlstore: menelusuri ringkasan tahap: %w", err)
	}
	return summary, nil
}

// Get membaca satu laporan.
func (r *Repo) Get(ctx context.Context, number string) (pelaporanklaim.ClaimReport, error) {
	row := r.db.QueryRowContext(ctx, query("report_get"), strings.TrimSpace(number))

	report, err := scanReport(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrNotFound
	case err != nil:
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/sqlstore: membaca laporan: %w", err)
	}
	return report, nil
}

// Insert menyimpan laporan baru dengan nomor yang diterbitkan di sini.
//
// Kedua langkahnya — mengambil nomor urut dan menyisipkan baris — berada dalam SATU
// transaksi. Ini memperbaiki cacat nyata sistem lama: `PROCINSERTDATARECIVEDKLAIM`
// menjalankan ROLLBACK-nya sendiri di dalam setiap handler galat, sementara rule yang
// memanggilnya menjalankan COMMIT sesudahnya
// (`RDB List/Rcv_ProcInsertRecivedDocument-SQL.xml:127`). `D-68` menetapkan kepemilikan
// transaksi berpindah ke Go persis karena pola seperti itu.
func (r *Repo) Insert(ctx context.Context, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = tx.Rollback() }()

	var sequence int64
	if err := tx.QueryRowContext(ctx, query("report_next_sequence")).Scan(&sequence); err != nil {
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/sqlstore: mengambil nomor urut: %w", err)
	}

	report = report.Clean()
	report.Number = BuildNumber(report.CreatedAt, sequence)

	if _, err := tx.ExecContext(ctx, query("report_insert"), insertArgs(report)...); err != nil {
		return pelaporanklaim.ClaimReport{}, translateWriteError(err, "menyisipkan laporan")
	}
	if err := tx.Commit(); err != nil {
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/sqlstore: menyimpan laporan baru: %w", err)
	}
	return report, nil
}

// Update mengubah laporan yang sudah ada.
func (r *Repo) Update(ctx context.Context, report pelaporanklaim.ClaimReport) (pelaporanklaim.ClaimReport, error) {
	report = report.Clean()

	result, err := r.db.ExecContext(ctx, query("report_update"), updateArgs(report)...)
	if err != nil {
		return pelaporanklaim.ClaimReport{}, translateWriteError(err, "mengubah laporan")
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap nomor yang tidak
	// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi.
	affected, err := result.RowsAffected()
	if err != nil {
		return pelaporanklaim.ClaimReport{}, fmt.Errorf("pelaporanklaim/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if affected == 0 {
		return pelaporanklaim.ClaimReport{}, pelaporanklaim.ErrNotFound
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya — termasuk
	// kolom yang tidak ikut diubah dan nilai yang dirapikan basis data.
	return r.Get(ctx, report.Number)
}

// CheckTable menguji apakah tabel migrasi 0003 sudah ada dan dapat dibaca akun aplikasi,
// tanpa mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: migrasi belum dijalankan DBA, versus akun aplikasi tidak
// punya hak baca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("report_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}

// filterArgs menyiapkan argumen penyaring, masing-masing diulang sebanyak kemunculannya
// di kueri.
//
// Pengulangan itu ada karena penanda posisi Oracle menuntut satu argumen per penanda,
// dan penyaring yang sama muncul dua kali pada pola `:n IS NULL OR ...`. Urutannya harus
// sama persis dengan urutan penanda pada report_list dan report_count.
func filterArgs(f pelaporanklaim.Filter) []any {
	stage := nilIfEmpty(string(f.Stage))
	branch := nilIfEmpty(f.BranchCode)
	search := nilIfEmpty(searchPattern(f.Search))

	return []any{
		stage, stage,
		branch, branch,
		search, search, search, search, search, search,
	}
}

// nilIfEmpty mengubah teks kosong menjadi NULL.
//
// Itulah yang membuat satu kueri melayani seluruh gabungan penyaring tanpa merangkai teks
// SQL: penyaring yang tidak diisi menjadi NULL, dan `:n IS NULL OR …` membuatnya tidak
// mempersempit apa pun.
func nilIfEmpty(s string) any {
	if trimmed := strings.TrimSpace(s); trimmed != "" {
		return trimmed
	}
	return nil
}

// searchPattern melindungi wildcard LIKE yang diketik pengguna.
//
// Tanpa ini, mencari "100%" akan cocok dengan SEMUA baris, dan mencari "A_B" akan cocok
// dengan "AXB". Keduanya bukan sekadar hasil yang aneh — pada layar yang menampilkan data
// nasabah, penyaring yang diam-diam melebar berarti petugas melihat baris yang tidak ia
// cari.
//
// Karakter pelarian yang dipakai backslash, sama dengan klausa ESCAPE pada kuerinya.
// Backslash itu sendiri ikut dilarikan LEBIH DULU, supaya pelariannya tidak dilarikan
// dua kali.
func searchPattern(search string) string {
	search = strings.TrimSpace(search)
	if search == "" {
		return ""
	}
	search = strings.ReplaceAll(search, `\`, `\\`)
	search = strings.ReplaceAll(search, "%", `\%`)
	search = strings.ReplaceAll(search, "_", `\_`)
	return search
}

// insertArgs menyusun argumen INSERT sesuai urutan penanda pada report_insert.
func insertArgs(r pelaporanklaim.ClaimReport) []any {
	return []any{
		r.Number,
		r.ReporterName,
		nilIfEmpty(r.SenderEmail),
		nilIfEmpty(r.SenderPhone),
		nilIfEmpty(r.CourierName),
		nilIfEmpty(r.EmailSubject),
		nilIfEmpty(r.PolicyNumber),
		nilIfEmpty(r.InsuredName),
		nilIfEmpty(r.InsuredEmail),
		nilIfEmpty(r.BusinessCode),
		nilIfEmpty(r.GroupPanel),
		nilIfEmpty(r.ReferenceNumber),
		timeOrNil(r.LossDate),
		nilIfEmpty(r.LossLocation),
		nilIfEmpty(r.Chronology),
		nilIfEmpty(r.DamageDetails),
		nilIfEmpty(r.DriverLicense),
		nilIfEmpty(r.EstimatedValue),
		nilIfEmpty(r.ClaimType),
		r.DocumentCount,
		timeOrNil(r.DocumentReceivedDate),
		nilIfEmpty(r.ClaimNumber),
		transferredCode(r.Transferred),
		timeOrNil(r.TransferredAt),
		timeOrNil(r.RegisteredAt),
		nilIfEmpty(r.NotTransferredReason),
		nilIfEmpty(r.NotRegisteredNote),
		nilIfEmpty(string(r.Outcome)),
		nilIfEmpty(r.BranchCode),
		nilIfEmpty(r.CreatedBy),
		r.CreatedAt.UTC(),
		r.UpdatedAt.UTC(),
	}
}

// updateArgs menyusun argumen UPDATE sesuai urutan penanda pada report_update.
//
// Perhatikan NOMOR berada di AKHIR, karena ia klausa WHERE — bukan di awal seperti pada
// INSERT. Menukarnya akan mengubah laporan yang salah tanpa galat apa pun.
func updateArgs(r pelaporanklaim.ClaimReport) []any {
	return []any{
		r.ReporterName,
		nilIfEmpty(r.SenderEmail),
		nilIfEmpty(r.SenderPhone),
		nilIfEmpty(r.CourierName),
		nilIfEmpty(r.EmailSubject),
		nilIfEmpty(r.PolicyNumber),
		nilIfEmpty(r.InsuredName),
		nilIfEmpty(r.InsuredEmail),
		nilIfEmpty(r.BusinessCode),
		nilIfEmpty(r.GroupPanel),
		nilIfEmpty(r.ReferenceNumber),
		timeOrNil(r.LossDate),
		nilIfEmpty(r.LossLocation),
		nilIfEmpty(r.Chronology),
		nilIfEmpty(r.DamageDetails),
		nilIfEmpty(r.DriverLicense),
		nilIfEmpty(r.EstimatedValue),
		nilIfEmpty(r.ClaimType),
		r.DocumentCount,
		timeOrNil(r.DocumentReceivedDate),
		nilIfEmpty(r.ClaimNumber),
		transferredCode(r.Transferred),
		timeOrNil(r.TransferredAt),
		timeOrNil(r.RegisteredAt),
		nilIfEmpty(r.NotTransferredReason),
		nilIfEmpty(r.NotRegisteredNote),
		nilIfEmpty(string(r.Outcome)),
		nilIfEmpty(r.BranchCode),
		r.UpdatedAt.UTC(),
		r.Number,
	}
}

// transferredCode mengubah boolean domain menjadi sandi kolom.
//
// `09-DATABASE-STRATEGY.md` §5 menetapkan boolean disimpan `NUMBER(1)` di Oracle dan
// dipetakan di adapter — bukan disimpan sebagai teks "Ya"/"Tidak" seperti tabel warisan
// `LST_ACCOUNT`. Tabel modul ini baru, sehingga tidak ada sandi lama yang harus dihormati.
func transferredCode(transferred bool) int {
	if transferred {
		return 1
	}
	return 0
}

// timeOrNil mengubah penunjuk waktu kosong menjadi NULL.
//
// Tanggal yang belum diketahui disimpan sebagai NULL, bukan sebagai tanggal nol. Tanggal
// nol Go adalah tahun 1, dan menyimpannya akan membuat laporan yang tanggal kejadiannya
// belum diisi muncul di urutan paling awal pada setiap laporan berbasis periode.
func timeOrNil(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC()
}

// scanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type scanner interface{ Scan(dest ...any) error }

// scanReport membaca satu baris hasil menjadi entitas domain.
//
// Urutan Scan HARUS sama persis dengan urutan kolom pada report_list dan report_get.
// Keduanya ditulis berdampingan di report.sql supaya dapat dibandingkan tanpa berpindah
// berkas.
func scanReport(row scanner) (pelaporanklaim.ClaimReport, error) {
	var (
		number               string
		reporterName         sql.NullString
		senderEmail          sql.NullString
		senderPhone          sql.NullString
		courierName          sql.NullString
		emailSubject         sql.NullString
		policyNumber         sql.NullString
		insuredName          sql.NullString
		insuredEmail         sql.NullString
		businessCode         sql.NullString
		groupPanel           sql.NullString
		referenceNumber      sql.NullString
		lossDate             sql.NullTime
		lossLocation         sql.NullString
		chronology           sql.NullString
		damageDetails        sql.NullString
		driverLicense        sql.NullString
		estimatedValue       sql.NullString
		claimType            sql.NullString
		documentCount        sql.NullInt64
		documentReceivedDate sql.NullTime
		claimNumber          sql.NullString
		transferred          sql.NullInt64
		transferredAt        sql.NullTime
		registeredAt         sql.NullTime
		notTransferredReason sql.NullString
		notRegisteredNote    sql.NullString
		outcome              sql.NullString
		branchCode           sql.NullString
		createdBy            sql.NullString
		createdAt            sql.NullTime
		updatedAt            sql.NullTime
	)

	err := row.Scan(
		&number, &reporterName, &senderEmail, &senderPhone, &courierName, &emailSubject,
		&policyNumber, &insuredName, &insuredEmail, &businessCode, &groupPanel,
		&referenceNumber, &lossDate, &lossLocation, &chronology, &damageDetails,
		&driverLicense, &estimatedValue, &claimType, &documentCount,
		&documentReceivedDate, &claimNumber, &transferred, &transferredAt,
		&registeredAt, &notTransferredReason, &notRegisteredNote, &outcome,
		&branchCode, &createdBy, &createdAt, &updatedAt,
	)
	if err != nil {
		return pelaporanklaim.ClaimReport{}, err
	}

	return pelaporanklaim.ClaimReport{
		Number:               number,
		ReporterName:         reporterName.String,
		SenderEmail:          senderEmail.String,
		SenderPhone:          senderPhone.String,
		CourierName:          courierName.String,
		EmailSubject:         emailSubject.String,
		PolicyNumber:         policyNumber.String,
		InsuredName:          insuredName.String,
		InsuredEmail:         insuredEmail.String,
		BusinessCode:         businessCode.String,
		GroupPanel:           groupPanel.String,
		ReferenceNumber:      referenceNumber.String,
		LossDate:             timeOrNothing(lossDate),
		LossLocation:         lossLocation.String,
		Chronology:           chronology.String,
		DamageDetails:        damageDetails.String,
		DriverLicense:        driverLicense.String,
		EstimatedValue:       estimatedValue.String,
		ClaimType:            claimType.String,
		DocumentCount:        int(documentCount.Int64),
		DocumentReceivedDate: timeOrNothing(documentReceivedDate),
		ClaimNumber:          claimNumber.String,
		Transferred:          transferred.Int64 == 1,
		TransferredAt:        timeOrNothing(transferredAt),
		RegisteredAt:         timeOrNothing(registeredAt),
		NotTransferredReason: notTransferredReason.String,
		NotRegisteredNote:    notRegisteredNote.String,
		Outcome:              pelaporanklaim.ClaimOutcome(strings.TrimSpace(outcome.String)),
		BranchCode:           branchCode.String,
		CreatedBy:            createdBy.String,
		CreatedAt:            createdAt.Time.UTC(),
		UpdatedAt:            updatedAt.Time.UTC(),
		// Clean dipanggil di sini, bukan diserahkan ke pemanggil: kolom CHAR di Oracle
		// dikembalikan berisi padding spasi, dan membiarkannya membuat perbandingan nomor
		// gagal tanpa sebab yang terlihat.
	}.Clean(), nil
}

// timeOrNothing mengubah kolom tanggal NULL menjadi penunjuk kosong.
func timeOrNothing(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	t := n.Time.UTC()
	return &t
}

// translateWriteError mengubah pelanggaran kunci utama menjadi galat domain.
//
// Yang dicocokkan adalah NAMA CONSTRAINT yang kita buat sendiri di migrasi 0003, bukan
// nomor galat driver — nama itu milik kita dan tidak berubah saat driver atau basis
// datanya berganti.
func translateWriteError(err error, activity string) error {
	if strings.Contains(strings.ToUpper(err.Error()), PrimaryKeyName) {
		// Kunci utama bentrok berarti urutan mengeluarkan nomor yang sudah dipakai.
		// Seharusnya mustahil; bila terjadi ia harus terlihat, bukan menimpa laporan lain.
		return pelaporanklaim.ErrNumberTaken
	}
	return fmt.Errorf("pelaporanklaim/sqlstore: %s: %w", activity, err)
}

var _ pelaporanklaim.Repo = (*Repo)(nil)
