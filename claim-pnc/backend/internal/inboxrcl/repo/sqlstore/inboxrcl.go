package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/platform/sqlvalue"
)

// Repo membaca antrean RCL Dokter dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// LegacyOperatorFor membaca identitas lama petugas — padanan `TempOperator.City`.
//
// Ketiadaan baris BUKAN galat: di Pega ia menghasilkan antrean kosong, dan pemanggil yang
// menyatakan keadaannya ke layar.
//
// # Kenapa identitas lama yang SAMA dengan login tetap dikembalikan
//
// Berbeda dari `inboxoutstanding.LegacyOperatorFor`, yang mengosongkannya karena di sana ia
// TAMBAHAN di samping login. Di sini ia SATU-SATUNYA identitas yang menyaring antrean —
// mengosongkannya akan menghapus antrean pengguna yang identitasnya tidak pernah berganti.
func (r *Repo) LegacyOperatorFor(ctx context.Context, loginID string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return "", nil
	}

	groups := inboxrcl.LegacyAccessGroups

	var legacy sql.NullString
	err := r.db.QueryRowContext(ctx, query("legacy_operator_for"),
		id,
		groups[0], groups[1], groups[2],
		inboxrcl.ExcludedAccessGroup,
	).Scan(&legacy)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("inboxrcl/sqlstore: membaca identitas lama petugas: %w", err)
	}

	// MAX atas himpunan kosong mengembalikan satu baris bernilai NULL, bukan nol baris.
	return strings.ToUpper(strings.TrimSpace(legacy.String)), nil
}

// List mengambil satu halaman antrean milik sebuah identitas lama.
//
// Jumlah seluruh baris ikut dibawa kueri yang sama lewat `COUNT(*) OVER ()`, sehingga pada
// halaman KOSONG jumlah totalnya nol — halaman di luar jangkauan ditangani layar dengan
// kembali ke halaman pertama.
func (r *Repo) List(
	ctx context.Context,
	operator string,
	f inboxrcl.Filter,
) (inboxrcl.Page, error) {
	clean := f.Normalize()

	result := inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}}

	// Satu argumen untuk SETIAP kemunculan penanda — godror mengikat menurut urutan
	// kemunculan (lihat kepala list_tasks). Pola yang sama dikirim dua kali, untuk PYID dan
	// untuk POLICYNO.
	pattern := searchPattern(clean.Search)
	rows, err := r.db.QueryContext(ctx, query("list_tasks"),
		operator,
		inboxrcl.StatusKerjaSelesai,
		operator,
		nilIfEmpty(pattern),
		pattern,
		pattern,
		clean.Offset,
		clean.Limit,
	)
	if err != nil {
		return inboxrcl.Page{}, fmt.Errorf("menjalankan kueri list_tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		task, total, err := scanTask(rows)
		if err != nil {
			return inboxrcl.Page{}, fmt.Errorf("membaca baris kueri list_tasks: %w", err)
		}
		result.Tasks = append(result.Tasks, task)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxrcl.Page{}, fmt.Errorf("menelusuri hasil kueri list_tasks: %w", err)
	}

	return result, nil
}

// searchPattern membentuk pola LIKE `%KATA%` berhuruf besar — sisi SQL memakai UPPER(...),
// dan perbandingan yang hanya satu sisinya diseragamkan tidak pernah cocok.
//
// Karakter khusus LIKE di-escape supaya pencarian "100%" tidak berubah menjadi pola yang
// mencocokkan apa saja; `ESCAPE '\'`-nya dinyatakan di sisi SQL. Preseden `inboxoutstanding`.
func searchPattern(search string) string {
	trimmed := strings.TrimSpace(search)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// nilIfEmpty mengirim pola kosong sebagai NULL, supaya `:4 IS NULL` mematikan seluruh
// saringan pencarian alih-alih memaksa `LIKE` memindai setiap baris.
func nilIfEmpty(value string) any { return sqlvalue.NilIfEmpty(value) }

// scanTask membaca satu baris hasil beserta jumlah seluruh baris yang menyertainya.
//
// Urutan pembacaan WAJIB sama dengan urutan kolom pada kueri dan dengan taskColumns.
// Seluruh kolom teks dibaca sebagai NullString: kolom tabel ini tidak punya constraint
// NOT NULL, dan satu baris warisan yang kosong tidak boleh menjatuhkan seluruh halaman.
func scanTask(rows *sql.Rows) (inboxrcl.RCLTask, int, error) {
	var (
		reference        sql.NullString
		caseID           sql.NullString
		policyNumber     sql.NullString
		insuredName      sql.NullString
		sentToRCLAt      sql.NullTime
		analystNote      sql.NullString
		rclDoctor        sql.NullString
		registeredAt     sql.NullTime
		processStatus    sql.NullString
		assignedOperator sql.NullString
		total            int
	)

	if err := rows.Scan(
		&reference,
		&caseID,
		&policyNumber,
		&insuredName,
		&sentToRCLAt,
		&analystNote,
		&rclDoctor,
		&registeredAt,
		&processStatus,
		&assignedOperator,
		&total,
	); err != nil {
		return inboxrcl.RCLTask{}, 0, err
	}

	task := inboxrcl.RCLTask{
		ClaimID:          reference.String,
		ClaimNumber:      caseID.String,
		PolicyNumber:     policyNumber.String,
		InsuredName:      insuredName.String,
		AnalystNote:      analystNote.String,
		RCLDoctor:        rclDoctor.String,
		ProcessStatus:    processStatus.String,
		AssignedOperator: assignedOperator.String,
	}

	// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4); konversi ke WIB hanya di
	// lapisan transport.
	if sentToRCLAt.Valid {
		task.SentToRCLAt = sentToRCLAt.Time.UTC()
	}
	if registeredAt.Valid {
		task.RegisteredAt = registeredAt.Time.UTC()
	}

	return task, total, nil
}

// CheckTables memastikan T_CLAIMLIST_ADMIN dan T_ACCESS_GROUP_PNC terbaca dari koneksi ini.
// Dipakai `-periksa`.
func (r *Repo) CheckTables(ctx context.Context) error {
	var probe int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&probe); err != nil {
		return fmt.Errorf("membaca tabel antrean RCL Dokter: %w", err)
	}
	return nil
}

// CheckColumns memastikan ketiga kolom migrasi 0012 sudah ada di T_CLAIMLIST_ADMIN.
//
// Terpisah dari CheckTables karena sebab gagalnya berbeda — lihat check_columns.
func (r *Repo) CheckColumns(ctx context.Context) error {
	var sent, doctor, note int
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(&sent, &doctor, &note); err != nil {
		return fmt.Errorf(
			"kolom TANGGALANALYSTSENDRCL_1 / NAMADOKTERRCL_1 / KOMENTARANALISATOR_1 pada "+
				"POOLDATA.T_CLAIMLIST_ADMIN tidak dapat dibaca — migrasi "+
				"0012_claimlist_admin_rcl belum dijalankan DBA: %w", err)
	}
	return nil
}
