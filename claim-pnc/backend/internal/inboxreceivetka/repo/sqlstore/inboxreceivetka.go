package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxreceivetka"
)

// registerDateLayout adalah bentuk teks kolom `REGISTERDATE_1`.
//
// Kolomnya `VARCHAR2(32)`, bukan tanggal, dan isinya terverifikasi berformat `yyyymmdd`
// pada data produksi (`20230510`, `20240319`). Penguraiannya dilakukan DI SINI, bukan di
// dalam SQL: `TO_DATE` adalah fungsi yang `D-20` larang dari kueri portabel.
const registerDateLayout = "20060102"

// Repo membaca daftar Inbox Receive TKA dan menuliskan tanggal kelengkapan dokumennya.
//
// Ketiga tabel yang disentuh, kedua gabungannya, dan asal setiap nama kolom ada di kepala
// inboxreceivetka.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca klaim TKA yang tanggal kelengkapan dokumennya belum diisi.
//
// # Cara pemotongannya diketahui, dan kenapa begitu
//
// Kueri meminta `MaxRows + 1` baris, lalu baris terakhir DIBUANG bila jumlahnya melebihi
// MaxRows. Keberadaan baris ke-(N+1) itulah satu-satunya bukti bahwa masih ada yang
// tertinggal.
//
// Cara lain — menjalankan COUNT(*) terpisah — akan menembak basis data dua kali untuk satu
// layar, dan kedua angkanya dapat berasal dari saat yang berbeda sehingga daftar dan
// keterangannya saling bertentangan.
func (r *Repo) List(
	ctx context.Context,
	filter inboxreceivetka.Filter,
) (inboxreceivetka.Page, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	// Satu lebih banyak daripada yang akan dikirim; lihat catatan di atas.
	limit := inboxreceivetka.MaxRows + 1

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("tka_inbox_list"),
			inboxreceivetka.ResolvedWorkStatus, limit)
	} else {
		// Nilai yang sama dikirim empat kali; lihat catatan pada tka_inbox_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("tka_inbox_search"),
			inboxreceivetka.ResolvedWorkStatus,
			pattern, pattern, pattern, pattern, limit)
	}
	if err != nil {
		return inboxreceivetka.Page{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]inboxreceivetka.Task, 0, inboxreceivetka.MaxRows)
	for rows.Next() {
		one, err := scanTask(rows)
		if err != nil {
			return inboxreceivetka.Page{}, err
		}
		tasks = append(tasks, one)
	}
	if err := rows.Err(); err != nil {
		return inboxreceivetka.Page{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: menelusuri daftar: %w", err)
	}

	truncated := len(tasks) > inboxreceivetka.MaxRows
	if truncated {
		tasks = tasks[:inboxreceivetka.MaxRows]
	}
	return inboxreceivetka.Page{Tasks: tasks, Truncated: truncated}, nil
}

// Complete mengisi tanggal kelengkapan dokumen pada klaim yang sebenarnya.
//
// # Urutan langkahnya, dan kenapa persis begitu
//
//  1. BEGIN
//  2. baca baris pekerjaannya          — sekaligus menyediakan isi surel
//  3. tolak bila klaimnya tidak ada    — ErrClaimMissing, sebelum menulis apa pun
//  4. UPDATE T_CLAIM_PNC               — dengan penjaga `TGLDOKLENGKAP IS NULL`
//  5. COMMIT
//
// # SATU tabel yang ditulis, dan itu keputusan yang disengaja
//
// Hanya `POOLDATA.T_CLAIM_PNC.TGLDOKLENGKAP`. Tabel engine Pega TIDAK disentuh sama sekali.
//
// Pega menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya ke kolom
// `PC_ASM_FW_GCNMFW_WORK.TANGGALDOKLENGKAP`. Menulis kolom itu langsung akan tertimpa tanpa
// satu pun tanda begitu Pega menyimpan kasusnya lagi — dan pekerjaan yang tampil di layar
// ini semuanya masih berjalan, sehingga Pega pasti akan menyentuhnya lagi.
//
// Yang membuat barisnya tetap hilang dari layar adalah penyaring daftarnya, yang memeriksa
// KEDUA kolom tanggal. Lihat kepala inboxreceivetka.sql.
//
// # TANPA `SELECT ... FOR UPDATE`
//
// Penguncian akan menahan baris pada tabel yang sedang dilayani Pega. Ia juga tidak
// dibutuhkan: penjaga `TGLDOKLENGKAP IS NULL` pada UPDATE bersifat atomik, dan pemeriksaan
// jumlah baris terpengaruh yang mengubah perlombaan menjadi penolakan yang bersuara.
//
// Dua permintaan bersamaan karena itu hanya membuat satu di antaranya menyentuh satu baris;
// yang kedua menyentuh nol dan ditolak — sehingga surel ganda tidak dapat terjadi tanpa
// perlu kunci idempotensi terpisah.
func (r *Repo) Complete(
	ctx context.Context,
	one inboxreceivetka.Completion,
) (inboxreceivetka.Task, error) {
	cleaned := one.Clean()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxreceivetka.Task{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa ini,
	// satu jalur galat yang terlewat meninggalkan transaksi menggantung.
	defer func() { _ = tx.Rollback() }()

	found, err := findOne(ctx, tx, cleaned.ClaimNumber)
	if err != nil {
		return inboxreceivetka.Task{}, err
	}
	switch len(found) {
	case 0:
		// Tidak ada, sudah diisi orang lain, sudah selesai, atau bukan klaim TKA. Keempatnya
		// berarti hal yang sama bagi pengguna; lihat catatan pada ErrTaskNotFound.
		return inboxreceivetka.Task{}, inboxreceivetka.ErrTaskNotFound
	case 1:
	default:
		return inboxreceivetka.Task{}, inboxreceivetka.ErrClaimAmbiguous
	}

	task := found[0]
	if task.ClaimKey == "" {
		// Pekerjaannya ada, klaimnya tidak. Ditolak SEBELUM menulis apa pun, dan dengan
		// galat tersendiri: menyegarkan daftar tidak akan menolong, sehingga layar harus
		// mengatakan sesuatu yang lain daripada "coba lagi".
		return inboxreceivetka.Task{}, inboxreceivetka.ErrClaimMissing
	}

	result, err := tx.ExecContext(ctx, getQuery("tka_claim_set_document_date"),
		cleaned.CompletedAt, task.ClaimKey)
	if err != nil {
		return inboxreceivetka.Task{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: memperbarui klaim %q: %w", task.ClaimNumber, err)
	}
	if err := expectSingleRow(result, task.ClaimNumber); err != nil {
		return inboxreceivetka.Task{}, err
	}

	if err := tx.Commit(); err != nil {
		return inboxreceivetka.Task{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: menutup transaksi pengisian: %w", err)
	}
	return task, nil
}

// findOne membaca baris pekerjaan yang akan diisi.
//
// Mengembalikan SELURUH baris yang cocok, bukan yang pertama: jumlahnya adalah keterangan
// yang dibutuhkan pemanggil untuk membedakan "tidak ada", "sebagaimana mestinya", dan
// "nomor kasus kembar".
func findOne(
	ctx context.Context,
	tx *sql.Tx,
	claimNumber string,
) ([]inboxreceivetka.Task, error) {
	rows, err := tx.QueryContext(ctx, getQuery("tka_inbox_find_one"),
		inboxreceivetka.ResolvedWorkStatus, claimNumber)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxreceivetka/sqlstore: membaca pekerjaan %q: %w", claimNumber, err)
	}
	defer func() { _ = rows.Close() }()

	var result []inboxreceivetka.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxreceivetka/sqlstore: menelusuri pekerjaan %q: %w", claimNumber, err)
	}
	return result, nil
}

// expectSingleRow menolak UPDATE yang tidak menyentuh tepat satu baris.
//
// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu perubahan
// TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan menolaknya akan
// membatalkan pekerjaan yang sebenarnya tersimpan.
func expectSingleRow(result sql.Result, claimNumber string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return nil
	}
	switch {
	case affected == 1:
		return nil
	case affected == 0:
		// Perlombaan: permintaan lain mengisi tanggalnya di antara pembacaan dan penulisan.
		// Penjaga `TGLDOKLENGKAP IS NULL` pada UPDATE yang menangkapnya, dan itulah yang
		// membuat surel ganda tidak dapat terjadi.
		return fmt.Errorf("inboxreceivetka/sqlstore: klaim %q tidak tersentuh pembaruan: %w",
			claimNumber, inboxreceivetka.ErrTaskNotFound)
	default:
		return fmt.Errorf("inboxreceivetka/sqlstore: klaim %q tersentuh %d baris: %w",
			claimNumber, affected, inboxreceivetka.ErrClaimAmbiguous)
	}
}

// CheckTable membuktikan ketiga tabel beserta kolom yang DIBACA modul ini ada dan dapat
// dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	return r.checkStatement(ctx, "tka_inbox_check_table", "daftar Inbox Receive TKA")
}

// CheckClaimColumn membuktikan kolom sasaran tulis ada dan dapat dibaca.
func (r *Repo) CheckClaimColumn(ctx context.Context) error {
	return r.checkStatement(ctx, "tka_inbox_check_claim_column",
		"kolom TGLDOKLENGKAP pada T_CLAIM_PNC")
}

func (r *Repo) checkStatement(ctx context.Context, name, what string) error {
	rows, err := r.db.QueryContext(ctx, getQuery(name))
	if err != nil {
		return fmt.Errorf("inboxreceivetka/sqlstore: memeriksa %s: %w", what, err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountWaiting mencacah seluruh pekerjaan yang menunggu, tanpa dipotong.
//
// Angkanya dapat dibandingkan LANGSUNG dengan jumlah baris pada layar Pega: keduanya kini
// memakai tabel dan penyaring yang sama.
func (r *Repo) CountWaiting(ctx context.Context) (int, error) {
	return r.count(ctx, "tka_inbox_count_waiting")
}

// CountPegaOnly mencacah pekerjaan yang sudah dikerjakan di sini tetapi MASIH tampil di
// layar Pega. Lihat catatan pada kuerinya.
func (r *Repo) CountPegaOnly(ctx context.Context) (int, error) {
	return r.count(ctx, "tka_inbox_count_pega_only")
}

// CountOrphanClaim mencacah pekerjaan TKA yang klaimnya tidak ada di T_CLAIM_PNC.
func (r *Repo) CountOrphanClaim(ctx context.Context) (int, error) {
	return r.count(ctx, "tka_inbox_count_orphan_claim")
}

// CountMissingParticipant mencacah pekerjaan menunggu yang nama pesertanya tidak ditemukan
// di tabel polis.
func (r *Repo) CountMissingParticipant(ctx context.Context) (int, error) {
	return r.count(ctx, "tka_inbox_count_missing_participant")
}

// SampleRegisteredOn mengembalikan sampai dua puluh nilai `REGISTERDATE_1` yang berbeda.
//
// Dipakai `claimpnc -periksa` untuk memastikan format `yyyymmdd` berlaku pada seluruh
// daftar, bukan hanya pada dua baris yang sempat diperiksa tangan.
func (r *Repo) SampleRegisteredOn(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("tka_inbox_sample_registered_on"))
	if err != nil {
		return nil, fmt.Errorf(
			"inboxreceivetka/sqlstore: mengambil contoh tanggal registrasi: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf(
				"inboxreceivetka/sqlstore: membaca contoh tanggal registrasi: %w", err)
		}
		result = append(result, strings.TrimSpace(value.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxreceivetka/sqlstore: menelusuri contoh tanggal registrasi: %w", err)
	}
	return result, nil
}

// count menjalankan satu kueri pencacah atas pekerjaan yang belum selesai.
func (r *Repo) count(ctx context.Context, name string) (int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, getQuery(name),
		inboxreceivetka.ResolvedWorkStatus).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("inboxreceivetka/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyiapkan pola pencarian LIKE dari kata kunci pengguna.
//
// Tanpa pelolosan, pengguna yang mengetik "%" akan mencocokkan SELURUH daftar, dan yang
// mengetik "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun
// tanda bahwa yang dicari bukan yang diketik.
//
// Urutannya menentukan: garis miring terbalik diloloskan LEBIH DULU, sebelum persen dan
// garis bawah. Membaliknya akan meloloskan garis miring yang baru saja ditambahkan.
func likePattern(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun
// di pustaka standar.
type rowScanner interface {
	Scan(target ...any) error
}

// scanTask membaca satu baris menjadi Task.
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada tka_inbox_list, tka_inbox_search,
// tka_inbox_find_one, dan tka_inbox_check_table. Keempatnya menyebut kolom yang sama pada
// urutan yang sama; itu yang membuat satu pemindai cukup untuk keempatnya, dan query_test.go
// yang menjaganya tetap begitu.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom Pega
// bertipe VARCHAR2 tanpa penyeragaman sehingga spasi di ujung mungkin ada, dan dua kolom
// PASTI dapat NULL — `CLAIM_KEY` dan `PARTICIPANT_NAME` keduanya berasal dari gabungan LEFT.
func scanTask(row rowScanner) (inboxreceivetka.Task, error) {
	var (
		reference       sql.NullString
		claimKey        sql.NullString
		claimNumber     sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		participantName sql.NullString
		dateOfLoss      sql.NullTime
		registeredOn    sql.NullString
	)

	if err := row.Scan(
		&reference, &claimKey, &claimNumber, &policyNumber, &insuredName,
		&participantName, &dateOfLoss, &registeredOn,
	); err != nil {
		return inboxreceivetka.Task{}, fmt.Errorf(
			"inboxreceivetka/sqlstore: membaca baris daftar: %w", err)
	}

	return inboxreceivetka.Task{
		Reference:       strings.TrimSpace(reference.String),
		ClaimKey:        strings.TrimSpace(claimKey.String),
		ClaimNumber:     strings.TrimSpace(claimNumber.String),
		PolicyNumber:    strings.TrimSpace(policyNumber.String),
		InsuredName:     strings.TrimSpace(insuredName.String),
		ParticipantName: strings.TrimSpace(participantName.String),
		DateOfLoss:      nullableTime(dateOfLoss),
		RegisteredOn:    parseRegisterDate(registeredOn),
	}, nil
}

// parseRegisterDate mengurai kolom `REGISTERDATE_1` menjadi tanggal.
//
// Kolomnya TEKS, bukan tanggal — `VARCHAR2(32)` berisi `yyyymmdd`. Bentuk yang tidak
// dikenali menghasilkan nil, BUKAN galat dan BUKAN tanggal karangan: satu baris yang
// formatnya menyimpang tidak boleh menggagalkan seluruh daftar, dan menebaknya akan
// menampilkan lama menunggu yang salah tanpa satu pun tanda.
//
// `claimpnc -periksa` yang memperlihatkan berapa sering itu terjadi.
func parseRegisterDate(value sql.NullString) *time.Time {
	trimmed := strings.TrimSpace(value.String)
	if !value.Valid || trimmed == "" {
		return nil
	}
	// Sebagian nilai Pega membawa bagian jam di belakangnya (`yyyymmddThhmmss...`);
	// delapan karakter pertama sudah memuat tanggalnya.
	if len(trimmed) > 8 {
		trimmed = trimmed[:8]
	}
	moment, err := time.Parse(registerDateLayout, trimmed)
	if err != nil {
		return nil
	}
	return &moment
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

var _ inboxreceivetka.Repo = (*Repo)(nil)
