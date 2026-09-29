package sqlstore

import (
	"context"
	"fmt"
)

// Pemeriksaan yang dijalankan perintah `-periksa`.
//
// Ketiganya TERPISAH, dan pemisahannya bukan kerapian. Ketiganya gagal karena sebab yang
// sangat berbeda:
//
//	CheckTables   hak baca, atau tabel yang belum pernah dipakai modul mana pun
//	CheckColumns  nama kolom yang belum dipastikan DBA
//	CheckKPI      tabel yang diisi procedure terpisah, dan boleh saja belum ada
//
// Galat yang menyebut sebab yang salah akan mengirim orang yang memperbaikinya ke arah yang
// keliru — dan pada modul ini ketiganya diperbaiki oleh orang yang berbeda.

// CheckTables memastikan ketiga tabel modul ini terbaca dari koneksi yang dipakai.
//
// `POOLDATA.T_SURVEYORLIST` yang paling patut diperhatikan: ia tabel yang BELUM pernah
// dibaca modul mana pun di aplikasi ini, sehingga hak bacanya belum pernah terbukti.
func (r *Repo) CheckTables(ctx context.Context) error {
	var probe int
	if err := r.db.QueryRowContext(ctx, query("check_tables")).Scan(&probe); err != nil {
		return fmt.Errorf("membaca tabel antrean survei: %w", err)
	}
	return nil
}

// CheckColumns memastikan kolom yang menentukan isi layar memang ada.
//
// Dua di antaranya BELUM terkonfirmasi DBA — `ADJUSTERPIC_1` yang dipetakan ke "Appointment
// No", dan `LOSSTYPE` yang dipetakan ke "Cause Of Loss". Kolom tab ikut diperiksa karena
// tab yang kolomnya hilang gagal SELURUHNYA, bukan menampilkan satu sel kosong.
func (r *Repo) CheckColumns(ctx context.Context) error {
	probes := make([]any, 11)
	values := make([]int, 11)
	for i := range values {
		probes[i] = &values[i]
	}

	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(probes...); err != nil {
		return fmt.Errorf(
			"kolom antrean survei pada POOLDATA.T_SURVEYORLIST / POOLDATA.T_CLAIMLIST_ADMIN "+
				"tidak dapat dibaca. Dua di antaranya belum dikonfirmasi DBA — "+
				"ADJUSTERPIC_1 untuk \"Appointment No\" dan LOSSTYPE untuk \"Cause Of "+
				"Loss\" — lihat kepala inboxsurvey.sql §C: %w", err)
	}
	return nil
}

// CheckKPI memastikan tabel KPI beserta kesembilan kolom angkanya ada.
//
// Kegagalannya TIDAK menghalangi tab INBOX. `POOLDATA.DETAIL_KPI_ADJUSTER` diisi
// `Database/INSERT_KPIADJUSTER.prc`, dan ketiadaannya berarti tab KPI kosong — bukan layar
// yang rusak.
func (r *Repo) CheckKPI(ctx context.Context) error {
	probes := make([]any, 12)
	values := make([]int, 12)
	for i := range values {
		probes[i] = &values[i]
	}

	if err := r.db.QueryRowContext(ctx, query("check_kpi")).Scan(probes...); err != nil {
		return fmt.Errorf(
			"POOLDATA.DETAIL_KPI_ADJUSTER tidak dapat dibaca. Tabel itu diisi procedure "+
				"INSERT_KPIADJUSTER, dan ketiadaannya hanya mengosongkan tab KPI: %w", err)
	}
	return nil
}
