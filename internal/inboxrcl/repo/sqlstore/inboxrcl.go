package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxrcl"
)

// Repo membaca antrean RCL Dokter dari SATU basis data entitas — `POOLDATA.TC_PNC_PUCL`
// dan `POOLDATA.M_LOGIN_PNC`. Satu-satunya operasi yang menulis adalah Decide (decision.go,
// decision.sql).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// OperatorFor membaca LOGIN_ID pemanggil dari POOLDATA.M_LOGIN_PNC — pengganti
// `TempOperator.City` (T_ACCESS_GROUP_PNC tidak dipakai lagi; Work Owner 2026-10-05).
//
// Login yang tidak ada atau tidak aktif BUKAN galat: ia mengembalikan string kosong.
func (r *Repo) OperatorFor(ctx context.Context, loginID string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return "", nil
	}

	var login sql.NullString
	err := r.db.QueryRowContext(ctx, query("operator_for"), id, inboxrcl.LoginAktif).Scan(&login)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("inboxrcl/sqlstore: membaca login dari M_LOGIN_PNC: %w", err)
	}

	// MAX atas himpunan kosong mengembalikan satu baris bernilai NULL, bukan nol baris.
	return strings.ToUpper(strings.TrimSpace(login.String)), nil
}

// List mengambil satu halaman antrean milik seorang operator dari TC_PNC_PUCL.
func (r *Repo) List(
	ctx context.Context,
	operator string,
	f inboxrcl.Filter,
) (inboxrcl.Page, error) {
	clean := f.Normalize()

	result := inboxrcl.Page{Tasks: []inboxrcl.RCLTask{}}

	// Satu argumen untuk SETIAP kemunculan penanda — godror mengikat menurut urutan
	// kemunculan (lihat kepala inboxrcl.sql).
	pattern := searchPattern(clean.Search)
	rows, err := r.db.QueryContext(ctx, query("list_tasks"),
		operator,
		inboxrcl.StatusKerjaSelesai,
		string(inboxrcl.ModeRCL),
		string(inboxrcl.ModeMSIG),
		inboxrcl.StageRCLDoctor,
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

// Detail mengambil isi layar kerja `RCLDokter` satu klaim — hanya bila klaim itu ada di
// antrean operator. Selebihnya ErrClaimNotFound.
func (r *Repo) Detail(
	ctx context.Context,
	operator, claimNumber string,
) (inboxrcl.RCLDetail, error) {
	number := strings.TrimSpace(claimNumber)
	if number == "" || strings.TrimSpace(operator) == "" {
		return inboxrcl.RCLDetail{}, inboxrcl.ErrClaimNotFound
	}

	row := r.db.QueryRowContext(ctx, query("claim_detail"),
		number,
		operator,
		inboxrcl.StatusKerjaSelesai,
		string(inboxrcl.ModeRCL),
		string(inboxrcl.ModeMSIG),
		inboxrcl.StageRCLDoctor,
	)

	detail, err := scanDetail(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return inboxrcl.RCLDetail{}, inboxrcl.ErrClaimNotFound
	case err != nil:
		return inboxrcl.RCLDetail{}, fmt.Errorf("menjalankan kueri claim_detail: %w", err)
	}
	return detail, nil
}

// searchPattern membentuk pola LIKE `%KATA%` berhuruf besar; karakter khusus LIKE di-escape
// supaya pencarian "100%" tidak menjadi pola yang cocok dengan semuanya. Preseden
// `inboxoutstanding`.
func searchPattern(search string) string {
	trimmed := strings.TrimSpace(search)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// nilIfEmpty mengirim pola kosong sebagai NULL, supaya `:5 IS NULL` mematikan seluruh
// saringan pencarian.
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// scanTask membaca satu baris list_tasks. Urutannya WAJIB sama dengan taskColumns.
//
// Seluruh kolom teks dibaca sebagai NullString: satu baris warisan yang kosong tidak boleh
// menjatuhkan seluruh halaman.
func scanTask(rows *sql.Rows) (inboxrcl.RCLTask, int, error) {
	var (
		caseID, policyNumber, insuredName, analystNote, mode, processStatus, assignedOperator sql.NullString
		sentToRCLAt, registeredAt                                                             sql.NullTime
		total                                                                                 int
	)

	if err := rows.Scan(
		&caseID,
		&policyNumber,
		&insuredName,
		&sentToRCLAt,
		&analystNote,
		&mode,
		&registeredAt,
		&processStatus,
		&assignedOperator,
		&total,
	); err != nil {
		return inboxrcl.RCLTask{}, 0, err
	}

	task := inboxrcl.RCLTask{
		ClaimNumber:      strings.TrimSpace(caseID.String),
		PolicyNumber:     policyNumber.String,
		InsuredName:      insuredName.String,
		AnalystNote:      analystNote.String,
		Mode:             inboxrcl.Mode(strings.TrimSpace(mode.String)),
		ProcessStatus:    processStatus.String,
		AssignedOperator: assignedOperator.String,
	}
	// Waktu dibawa UTC; konversi ke WIB hanya di lapisan transport.
	if sentToRCLAt.Valid {
		task.SentToRCLAt = sentToRCLAt.Time.UTC()
	}
	if registeredAt.Valid {
		task.RegisteredAt = registeredAt.Time.UTC()
	}
	return task, total, nil
}

// scanDetail membaca satu baris claim_detail. Urutannya WAJIB sama dengan detailColumns.
func scanDetail(row *sql.Row) (inboxrcl.RCLDetail, error) {
	var (
		caseID, policyNumber, insuredName, mode, analystNote, reason, doctorReason sql.NullString
		statusClaim, processStatus, assignedOperator                               sql.NullString
		sentToRCLAt                                                                sql.NullTime
	)

	if err := row.Scan(
		&caseID,
		&policyNumber,
		&insuredName,
		&mode,
		&analystNote,
		&reason,
		&doctorReason,
		&statusClaim,
		&processStatus,
		&assignedOperator,
		&sentToRCLAt,
	); err != nil {
		return inboxrcl.RCLDetail{}, err
	}

	detail := inboxrcl.RCLDetail{
		ClaimNumber:      strings.TrimSpace(caseID.String),
		PolicyNumber:     policyNumber.String,
		InsuredName:      insuredName.String,
		Mode:             inboxrcl.Mode(strings.TrimSpace(mode.String)),
		AnalystNote:      analystNote.String,
		Reason:           reason.String,
		DoctorReason:     doctorReason.String,
		StatusClaim:      strings.TrimSpace(statusClaim.String),
		ProcessStatus:    processStatus.String,
		AssignedOperator: assignedOperator.String,
	}
	if sentToRCLAt.Valid {
		detail.SentToRCLAt = sentToRCLAt.Time.UTC()
	}
	return detail, nil
}

// CheckTables memastikan TC_PNC_PUCL dan M_LOGIN_PNC terbaca dari koneksi ini. Dipakai
// `-periksa`.
func (r *Repo) CheckTables(ctx context.Context) error {
	var probe int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&probe); err != nil {
		return fmt.Errorf("membaca tabel antrean RCL Dokter: %w", err)
	}
	return nil
}

// CheckColumns memastikan kedua kolom tambahan ada di TC_PNC_PUCL —
// `ALASAN_DOKTER_REJECT_RCL` dan `NAMA_DOKTER_RCL`.
//
// Yang kedua dipakai modul `registrasi` saat Kirim ke RCL/PUCL, bukan oleh modul ini.
// Diperiksa dari sini karena di sinilah `-periksa` atas tabel itu berada — dan kegagalannya
// jauh lebih luas daripada layar ini: kolom yang belum dibuat membuat SELURUH tombol Kirim
// ke RCL/PUCL gagal dengan ORA-00904 pada klaim pertama di produksi.
func (r *Repo) CheckColumns(ctx context.Context) error {
	var alasan, nama int
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(&alasan, &nama); err != nil {
		return fmt.Errorf(
			"kolom ALASAN_DOKTER_REJECT_RCL atau NAMA_DOKTER_RCL pada POOLDATA.TC_PNC_PUCL "+
				"tidak dapat dibaca: %w", err)
	}
	return nil
}
