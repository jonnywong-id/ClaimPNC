package sqlstore

import (
	"context"
	"database/sql"
)

// PICSourceState melaporkan keadaan satu objek yang dibaca tab KPI PIC Teknik.
type PICSourceState struct {
	// Object adalah nama objeknya seperti tertulis di kueri.
	Object string

	// Purpose menyebut untuk apa ia dibaca — supaya baris yang gagal dapat dipahami
	// tanpa membuka berkas .sql.
	Purpose string

	// Secondary menandai objek yang dibaca di koneksi KEDUA, bukan di basis data portal.
	Secondary bool

	// Err berisi penolakan basis data, atau nil bila objeknya terbaca.
	Err error
}

// CheckSources memeriksa SETIAP objek yang dibaca tab KPI PIC Teknik, satu per satu.
//
// # Kenapa satu per satu, bukan sekali jalan
//
// Tab ini menjalankan delapan kueri atas tujuh objek. Ketika tombol Cari gagal, yang
// sampai ke layar hanya satu kalimat — dan kalimat itu tidak menyebutkan objek mana dari
// tujuh yang menolak. Pada 2026-10-08 itu benar-benar terjadi, dan menemukan sebabnya
// menuntut membaca berkas SQL satu per satu.
//
// Pemeriksaan sekali jalan tidak menyelesaikannya: ia berhenti pada kegagalan pertama,
// sehingga objek kedua yang juga bermasalah baru ketahuan setelah yang pertama dibereskan.
// Daftar lengkap dalam satu kali jalan itulah yang membuat permintaan ke DBA dapat
// diajukan sekaligus, bukan berulang kali.
//
// # Kenapa probenya menyebut kolom
//
// Kolom yang hilang (`ORA-00904`) sama sering terjadi dengan tabel yang hilang
// (`ORA-00942`), dan keduanya menghentikan tab ini. Probe `SELECT 1 FROM x` akan lulus
// pada tabel yang kolomnya tidak sesuai, lalu membiarkan kegagalannya muncul di tangan
// pengguna. DDL-nya sendiri belum pernah kami lihat (`R-08`), jadi kolom adalah satu-
// satunya hal yang dapat kami pastikan dari sini.
func (r *Repo) CheckSources(ctx context.Context) []PICSourceState {
	source := []struct {
		query     string
		object    string
		purpose   string
		secondary bool
	}{
		{"probe_pic", "POOLDATA.MST_USER_TEKNIK",
			"daftar petugas per lini bisnis", false},
		{"probe_tangga_nilai", "POOLDATA.M_KPI_PNC",
			"tangga nilai seluruh komponen", false},
		{"probe_progres", "POOLDATA.GCNM_PROGRESS_CLAIM",
			"komponen Update Progress Klaim", false},
		{"probe_dashboard", "POOLDATA.PEGA_DASHBOARDPNC",
			"penyaring status dan PIC pada komponen Progress", false},
		{"probe_klaim", "POOLDATA.T_CLAIM_PNC",
			"komponen Analisa, Akseptasi, dan SLA", false},
		{"probe_adjustment", "POOLDATA.T_CLAIM_ADJUSTMENT",
			"tanggal komite dan akseptasi", false},
		{"probe_followup_aritmetika", "POOLDATA.GCNM_PROGRESS_CLAIM.NEXT_FOLLOWUP",
			"tipe kolomnya harus dapat dihitung sebagai tanggal", false},
		{"probe_hari_libur", "GENERAL.HRD_LBR",
			"kalender hari libur untuk hitungan hari kerja", true},
	}

	state := make([]PICSourceState, 0, len(source))
	for _, item := range source {
		conn, nama, lewatTautan := r.db, item.query, false
		if item.secondary {
			conn = r.aneka
			// Diperiksa SEPERTI yang dilakukan saat melayani permintaan, bukan seperti
			// yang kita harapkan: tanpa koneksi kedua, Holidays jatuh ke DB Link, dan
			// probe yang tetap menguji koneksi kedua akan melaporkan GAGAL untuk tab
			// yang sebenarnya berjalan. Laporan yang salah arah lebih buruk daripada
			// tidak ada laporan.
			if conn == nil {
				conn, nama, lewatTautan = r.db, item.query+"_dblink", true
			}
		}
		keterangan := item.purpose
		if lewatTautan {
			keterangan += " (lewat DB Link — koneksi kedua belum terpasang)"
		}
		state = append(state, PICSourceState{
			Object:    item.object,
			Purpose:   keterangan,
			Secondary: item.secondary,
			Err:       probe(ctx, conn, query(nama)),
		})
	}
	return state
}

// probe menjalankan satu pernyataan dan membuang hasilnya.
//
// Koneksi nil dilaporkan sebagai ErrHolidayCalendarUnavailable, bukan sebagai panic
// maupun sebagai "terbaca". Ia keadaan yang memang mungkin: koneksi kedua boleh tidak
// terpasang, dan `-periksa` justru ada untuk menyatakannya.
func probe(ctx context.Context, conn *sql.DB, statement string) error {
	if conn == nil {
		return ErrHolidayCalendarUnavailable
	}
	rows, err := conn.QueryContext(ctx, statement)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}
