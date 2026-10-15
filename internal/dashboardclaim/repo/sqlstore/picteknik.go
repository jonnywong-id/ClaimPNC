package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/dashboardclaim"
)

// ListTechnicalPIC membaca daftar PIC Teknik yang dapat menerima pemindahan klaim.
//
// Menggantikan `BrowseVMstUserTeknis_RD`; penyaring dan sumber tabelnya ada di
// `picteknik.sql`.
func (r *Repo) ListTechnicalPIC(
	ctx context.Context,
	filter dashboardclaim.TechnicalPICFilter,
) (dashboardclaim.TechnicalPICPage, error) {
	filter = filter.Normalize()

	// BusinessType tidak lagi dipakai — lihat catatan pada picteknik.sql. Ia dibiarkan ada
	// pada filter supaya pemanggil tidak perlu berubah saat penyaringnya kelak dikembalikan
	// bersama master pemetaan pengguna ke lini bisnis (`F-4`).
	args := picArgs(filter)

	var total int
	if err := r.db.QueryRowContext(ctx, query("pic_teknik_count"), args...).Scan(&total); err != nil {
		return dashboardclaim.TechnicalPICPage{}, fmt.Errorf("sqlstore: menghitung PIC Teknik: %w", err)
	}

	// `withPaging` menerima `Filter` tile, bukan penyaring master ini. Yang dipakainya hanya
	// Offset dan Limit, jadi keduanya disalin ke sana alih-alih menambah parameter pada
	// fungsi yang sudah dipakai empat pembaca lain.
	paging := dashboardclaim.Filter{Offset: filter.Offset, Limit: filter.Limit}

	rows, err := r.db.QueryContext(ctx, query("pic_teknik_list"), withPaging(args, paging)...)
	if err != nil {
		return dashboardclaim.TechnicalPICPage{}, fmt.Errorf("sqlstore: membaca PIC Teknik: %w", err)
	}
	defer rows.Close()

	page := dashboardclaim.TechnicalPICPage{Total: total}
	for rows.Next() {
		row, err := scanTechnicalPIC(rows)
		if err != nil {
			return dashboardclaim.TechnicalPICPage{}, err
		}
		page.Rows = append(page.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return dashboardclaim.TechnicalPICPage{}, fmt.Errorf("sqlstore: membaca PIC Teknik: %w", err)
	}

	return page, nil
}

// picArgs menyusun argumen `:1`…`:3` — seluruhnya tentang kotak cari.
//
// Pola yang sama dikirim tiga kali karena SQL-nya memakai tiga penanda: `:1` menjawab
// "apakah pengguna mencari sesuatu", lalu `:2` dan `:3` mencocokkan OPERATOR_ID dan
// MCL_NAME. Satu penanda satu kemunculan — aturan modul ini, dijaga uji
// TestBindMarkersAreUniqueAndAscending.
func picArgs(filter dashboardclaim.TechnicalPICFilter) []any {
	pattern := nilIfEmpty(likePattern(filter.Search))
	return []any{pattern, pattern, pattern}
}

// scanTechnicalPIC membaca satu baris.
//
// COUNTER_QUOTA dipindai sebagai NULL-able: master ini diisi tangan, dan baris yang
// pencacahnya belum pernah dinaikkan menyimpan NULL, bukan nol.
//
// TOTAL_JOB TIDAK dibaca — kolomnya tidak ada pada master ini. Lihat picteknik.sql.
func scanTechnicalPIC(rows scanner) (dashboardclaim.TechnicalPICRow, error) {
	var (
		operatorID, name, email, team sql.NullString
		workload                      sql.NullInt64
	)

	if err := rows.Scan(&operatorID, &name, &email, &team, &workload); err != nil {
		return dashboardclaim.TechnicalPICRow{}, fmt.Errorf("sqlstore: memindai PIC Teknik: %w", err)
	}

	return dashboardclaim.TechnicalPICRow{
		OperatorID: text(operatorID),
		Name:       text(name),
		Email:      text(email),
		TeamGroup:  text(team),
		Workload:   int(workload.Int64),
	}, nil
}

// CheckPICTable membuktikan master PIC Teknik terbaca beserta kolom yang dipakai daftarnya.
//
// Dipanggil `claimpnc -periksa`. Ia menutup kelas kegagalan yang sudah terjadi dua kali di
// modul ini: kueri menyebut kolom yang tidak ada, lalu layar gagal dimuat dengan galat Oracle
// yang tidak menyebut kolom mana — dan yang menemukannya adalah pengguna, bukan terminal.
//
// Yang diperiksa adalah DAFTAR KOLOM yang sama dengan `pic_teknik_list`, bukan sekadar
// keberadaan tabel. Probe `SELECT 1` akan lulus pada tabel yang kolomnya tidak cocok.
func (r *Repo) CheckPICTable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("pic_teknik_check")); err != nil {
		return fmt.Errorf(
			"master POOLDATA.MST_USER_TEKNIK tidak terbaca akun aplikasi, atau salah satu "+
				"kolom yang dipakai daftar PIC Teknik tidak ada "+
				"(OPERATOR_ID, MCL_NAME, EMAIL, TEAM_GROUP, COUNTER_QUOTA, STS_AKTIF): %w",
			err)
	}
	return nil
}
