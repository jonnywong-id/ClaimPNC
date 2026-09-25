package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// Repo membaca antrean pekerjaan Investigator dari tabel engine Pega dan tabel bisnisnya.
//
// **Hanya membaca.** Tidak ada satu pun method yang menulis, dan itu bukan kelalaian:
// mengambil pekerjaan dari antrean serta mencatat hasil investigasi terjadi di layar kerja
// yang belum dibangun. Lihat banner paket inboxinvestigator.
//
// Keempat tabel yang disentuh, gabungannya, dan asal setiap nama kolom ada di kepala
// inboxinvestigator.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca pekerjaan yang menunggu di workbasket Investigator.
//
// # Cara pemotongannya diketahui, dan kenapa begitu
//
// Kueri meminta `MaxRows + 1` baris, lalu baris terakhir DIBUANG bila jumlahnya melebihi
// MaxRows. Keberadaan baris ke-(N+1) itulah satu-satunya bukti bahwa masih ada yang
// tertinggal.
//
// Cara lain — menjalankan COUNT(*) terpisah — akan menembak basis data dua kali untuk
// satu layar, dan kedua angkanya dapat berasal dari saat yang berbeda sehingga daftar dan
// keterangannya saling bertentangan. Satu baris tambahan jauh lebih murah dan tidak dapat
// berbeda saatnya dengan barisnya sendiri.
//
// Inilah yang membedakan modul ini dari `pyMaxRecords = 500` sistem lama: keduanya
// memotong, tetapi yang ini MENYATAKANNYA.
func (r *Repo) List(
	ctx context.Context,
	filter inboxinvestigator.Filter,
) (inboxinvestigator.Page, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	// Satu lebih banyak daripada yang akan dikirim; lihat catatan di atas.
	limit := inboxinvestigator.MaxRows + 1
	basket := strings.ToUpper(inboxinvestigator.Workbasket)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx,
			getQuery("investigator_inbox_list"), basket, limit)
	} else {
		// Nilai yang sama dikirim tujuh kali; lihat catatan pada
		// investigator_inbox_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("investigator_inbox_search"),
			basket, pattern, pattern, pattern, pattern, pattern, pattern, pattern, limit)
	}
	if err != nil {
		return inboxinvestigator.Page{}, fmt.Errorf(
			"inboxinvestigator/sqlstore: membaca antrean: %w", err)
	}
	defer func() { _ = rows.Close() }()

	tasks := make([]inboxinvestigator.Task, 0, inboxinvestigator.MaxRows)
	for rows.Next() {
		one, err := scanTask(rows)
		if err != nil {
			return inboxinvestigator.Page{}, err
		}
		tasks = append(tasks, one)
	}
	if err := rows.Err(); err != nil {
		return inboxinvestigator.Page{}, fmt.Errorf(
			"inboxinvestigator/sqlstore: menelusuri antrean: %w", err)
	}

	truncated := len(tasks) > inboxinvestigator.MaxRows
	if truncated {
		tasks = tasks[:inboxinvestigator.MaxRows]
	}
	return inboxinvestigator.Page{Tasks: tasks, Truncated: truncated}, nil
}

// CountWaiting mencacah seluruh pekerjaan yang menunggu, tanpa dipotong. Dipakai
// `claimpnc -periksa`.
func (r *Repo) CountWaiting(ctx context.Context) (int, error) {
	return r.count(ctx, "investigator_inbox_count_waiting")
}

// CountWithoutSurvey mencacah pekerjaan menunggu yang tidak punya satu pun baris survei —
// yaitu yang kolom kesembilannya akan kosong di layar. Lihat catatan pada kuerinya.
func (r *Repo) CountWithoutSurvey(ctx context.Context) (int, error) {
	return r.count(ctx, "investigator_inbox_count_without_survey")
}

// CheckTable membuktikan keempat tabel beserta kolom yang dibaca modul ini ada dan dapat
// dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("investigator_inbox_check_table"))
	if err != nil {
		return fmt.Errorf(
			"inboxinvestigator/sqlstore: memeriksa tabel inbox investigator: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah atas workbasket Investigator.
func (r *Repo) count(ctx context.Context, name string) (int, error) {
	var total int
	basket := strings.ToUpper(inboxinvestigator.Workbasket)
	if err := r.db.QueryRowContext(ctx, getQuery(name), basket).Scan(&total); err != nil {
		return 0, fmt.Errorf("inboxinvestigator/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyiapkan pola pencarian LIKE dari kata kunci pengguna.
//
// Tanpa pelolosan, pengguna yang mengetik "%" akan mencocokkan SELURUH antrean, dan yang
// mengetik "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun
// tanda bahwa yang dicari bukan yang diketik.
//
// Urutannya menentukan: garis miring terbalik diloloskan LEBIH DULU, sebelum persen dan
// garis bawah. Membaliknya akan meloloskan garis miring yang baru saja ditambahkan, dan pola
// yang dihasilkan tidak lagi cocok dengan apa pun.
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
// di pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat
// yang dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner interface {
	Scan(target ...any) error
}

// scanTask membaca satu baris menjadi Task.
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada investigator_inbox_list,
// investigator_inbox_search, dan investigator_inbox_check_table. Ketiganya menyebut kolom
// yang sama pada urutan yang sama; itu yang membuat satu fungsi cukup untuk ketiganya, dan
// query_test.go yang menjaganya tetap begitu.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tidak ada DDL yang membuktikan sebaliknya (`R-08`).
// Kedua subquery-nya bahkan PASTI dapat NULL: klaim tanpa objek atau tanpa baris survei
// tidak mengembalikan apa-apa.
func scanTask(row rowScanner) (inboxinvestigator.Task, error) {
	var (
		reference       sql.NullString
		caseNumber      sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		participantName sql.NullString
		businessName    sql.NullString
		branchName      sql.NullString
		adminName       sql.NullString
		registeredAt    sql.NullTime
		surveyDate      sql.NullTime
	)

	if err := row.Scan(
		&reference, &caseNumber, &policyNumber, &insuredName, &participantName,
		&businessName, &branchName, &adminName, &registeredAt, &surveyDate,
	); err != nil {
		return inboxinvestigator.Task{}, fmt.Errorf(
			"inboxinvestigator/sqlstore: membaca baris antrean: %w", err)
	}

	return inboxinvestigator.Task{
		Reference:       strings.TrimSpace(reference.String),
		CaseNumber:      strings.TrimSpace(caseNumber.String),
		PolicyNumber:    strings.TrimSpace(policyNumber.String),
		InsuredName:     strings.TrimSpace(insuredName.String),
		ParticipantName: strings.TrimSpace(participantName.String),
		BusinessName:    strings.TrimSpace(businessName.String),
		BranchName:      strings.TrimSpace(branchName.String),
		AdminName:       strings.TrimSpace(adminName.String),
		RegisteredAt:    nullableTime(registeredAt),
		SurveyDate:      nullableTime(surveyDate),
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

var _ inboxinvestigator.Repo = (*Repo)(nil)
