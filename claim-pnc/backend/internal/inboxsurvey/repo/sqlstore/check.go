package sqlstore

import (
	"context"
	"fmt"
)

// Pemeriksaan yang dijalankan perintah `-periksa`.
//
// Kelimanya TERPISAH, dan pemisahannya bukan kerapian. Masing-masing gagal karena sebab yang
// sangat berbeda:
//
//	CheckTables          hak baca, atau tabel yang belum pernah dipakai modul mana pun
//	CheckColumns         nama kolom yang belum dipastikan DBA
//	CheckMissingColumns  kolom yang MEMANG belum ada — gagalnya diharapkan
//	CheckFilledColumns   kolomnya ada, tetapi isinya belum ditulis procedure
//	CheckKPI             tabel yang diisi procedure terpisah, dan boleh saja belum ada
//
// Galat yang menyebut sebab yang salah akan mengirim orang yang memperbaikinya ke arah yang
// keliru — dan pada modul ini masing-masing diperbaiki oleh orang yang berbeda.

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
// Kolom `T_CLAIM_PNC` ikut diperiksa meski tabelnya sudah dipakai modul lain: penamaannya
// BERBEDA JAUH dari tabel datar yang dipakai sebelumnya — `CLAIMNO` bukan `PYID`, `NOPOLIS`
// bukan `POLICYNO`, `PICTEKNIK` bukan `USERTEKNIS_1` — dan salah satu saja menjatuhkan seluruh
// layar dengan ORA-00904.
//
// Satu kolom BELUM terkonfirmasi DBA: `LOSSTYPE` yang dipetakan ke "Cause Of Loss". Kueri yang
// mengisinya di Pega hilang dari export (`R-16`).
func (r *Repo) CheckColumns(ctx context.Context) error {
	probes := make([]any, 13)
	values := make([]int, 13)
	for i := range values {
		probes[i] = &values[i]
	}

	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(probes...); err != nil {
		return fmt.Errorf(
			"kolom antrean survei pada POOLDATA.T_SURVEYORLIST / POOLDATA.T_CLAIM_PNC tidak "+
				"dapat dibaca. Perhatikan penamaan kolom klaim yang BERBEDA dari tabel datar "+
				"(CLAIMNO bukan PYID, NOPOLIS bukan POLICYNO, PICTEKNIK bukan USERTEKNIS_1), "+
				"dan LOSSTYPE untuk \"Cause Of Loss\" yang belum dikonfirmasi DBA — lihat "+
				"kepala inboxsurvey.sql: %w", err)
	}
	return nil
}

// NewColumns menyatakan kolom mana dari kelima yang ditunggu sudah ada.
//
// # Kenapa jumlahnya berubah-ubah, dan itu bukan kebingungan
//
// Daftar ini EMPAT pada 2026-09-29, TIGA setelah `ADJUSTER_PIC` dicoret sebagai asal
// "Appointment No", lalu LIMA setelah keempat kueri tab tiba dan dua kolom lain terbukti
// benar-benar berbeda. Setiap perubahan bersandar pada satu pengukuran, bukan pada pembacaan
// ulang — dan daftar yang mengecil justru hasil terbaiknya.
type NewColumns struct {
	Accept         bool
	Reference      bool
	WorkStatus     bool
	AdjusterPIC    bool
	SurveyLocation bool
}

// All menyatakan kelimanya sudah ada.
func (c NewColumns) All() bool {
	return c.Accept && c.Reference && c.WorkStatus && c.AdjusterPIC && c.SurveyLocation
}

// CheckNewColumns melaporkan kolom mana dari keempatnya yang sudah ada.
//
// # Kenapa lewat katalog, bukan probe parsing
//
// Probe berbasis `SELECT COUNT(kolom) … WHERE 1=0` bersifat SEMUA-ATAU-TIDAK: satu kolom yang
// belum ada menghasilkan ORA-00904, dan kolom lain yang sudah ada ikut terbaca sebagai belum
// ada. Itu persis keadaan 2026-09-30 — dua dari empat kolom tiba lebih dulu, dan probe lama
// tidak dapat menunjukkan mana yang sudah.
//
// Keberadaan kolom **bukan** berarti terisi. Keterisian diukur CheckFilledColumns.
func (r *Repo) CheckNewColumns(ctx context.Context) (NewColumns, error) {
	var accept, reference, workStatus, adjusterPIC, surveyLocation int

	err := r.db.QueryRowContext(ctx, query("check_new_columns")).
		Scan(&accept, &reference, &workStatus, &adjusterPIC, &surveyLocation)
	if err != nil {
		return NewColumns{}, fmt.Errorf(
			"membaca katalog kolom POOLDATA.T_SURVEYORLIST: %w", err)
	}

	return NewColumns{
		Accept:         accept > 0,
		Reference:      reference > 0,
		WorkStatus:     workStatus > 0,
		AdjusterPIC:    adjusterPIC > 0,
		SurveyLocation: surveyLocation > 0,
	}, nil
}

// FilledColumns adalah hasil pengukuran keterisian kolom yang SUDAH ADA.
//
// Kelimanya sudah ADA di basis data per 2026-10-03 — yang diukur di sini adalah ISINYA.
type FilledColumns struct {
	TotalRows      int
	Accept         int
	Reference      int
	WorkStatus     int
	AdjusterPIC    int
	SurveyLocation int
}

// CheckFilledColumns menghitung berapa baris kolom yang sudah ada BENAR-BENAR terisi.
//
// # Kenapa TERPISAH dari CheckNewColumns
//
// Karena keduanya diperbaiki langkah yang berbeda: kolom ditambahkan DBA lewat `ALTER`,
// sedangkan isinya ditulis procedure. Menyatukannya membuat "kolomnya sudah ada tetapi masih
// kosong" terbaca sebagai siap.
//
// # Kenapa nol pada Accept adalah keadaan yang PALING berbahaya
//
// Tab Outstanding menyaring `ADJUSTERACCEPT IS NULL`. Kolom yang ADA tetapi seluruhnya kosong
// akan menampilkan **seluruh antrean** sebagai "belum dikonfirmasi adjuster" — layar terisi
// wajar, angkanya masuk akal, dan isinya salah. Itu cacat diam, dan pengukuran ini satu-satunya
// yang menangkapnya sebelum pengguna melihatnya.
func (r *Repo) CheckFilledColumns(ctx context.Context) (FilledColumns, error) {
	var result FilledColumns

	err := r.db.QueryRowContext(ctx, query("check_filled_columns")).Scan(
		&result.TotalRows,
		&result.Accept,
		&result.Reference,
		&result.WorkStatus,
		&result.AdjusterPIC,
		&result.SurveyLocation,
	)
	if err != nil {
		return FilledColumns{}, fmt.Errorf(
			"menghitung keterisian kolom POOLDATA.T_SURVEYORLIST: %w", err)
	}
	return result, nil
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
