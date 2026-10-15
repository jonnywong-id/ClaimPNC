// Command cekkueri menjalankan kueri tab KPI PIC Teknik APA ADANYA terhadap Oracle.
//
// # Kenapa ia ada, padahal sudah ada `-periksa`
//
// Probe `-periksa` menjalankan kueri DASAR dengan `WHERE 1 = 0`. Ia membuktikan objek dan
// kolomnya ada, tetapi TIDAK menyentuh dua hal yang justru paling sering salah:
//
//	penyaring lini bisnis  disisipkan saat melayani permintaan, bukan ada di kueri dasar
//	parameter tanggal      tidak pernah terikat pada probe
//
// Pada 2026-10-08 keduanya menjadi sebab nyata: penyaring lini menyebut kolom yang tidak
// ada di `T_CLAIM_PNC`, dan `-periksa` tetap melaporkan seluruh sumber `[ok]`.
//
// Perkakas ini menutup celah itu â€” ia menjalankan kueri LENGKAP, dengan penyaring dan
// parameter terpasang, lalu membuang hasilnya. Ia MEMBACA saja.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"

	"claim-pnc/internal/reportkpi"
	reportkpisql "claim-pnc/internal/reportkpi/repo/sqlstore"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	env, err := bacaEnv(".env")
	if err != nil {
		return err
	}

	dsn := fmt.Sprintf("oracle://%s:%s@%s:%s/%s",
		url.QueryEscape(env["POOLDATA_ASM_PENGGUNA"]), url.QueryEscape(env["POOLDATA_ASM_SANDI"]),
		env["POOLDATA_ASM_HOST"], env["POOLDATA_ASM_PORT"], env["POOLDATA_ASM_SERVICE"])

	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// Koneksi kedua sengaja nil: itulah keadaan pemasangan hari ini, dan repo akan jatuh
	// ke jalur DB Link persis seperti saat melayani permintaan.
	repo := reportkpisql.NewRepo(db, nil)

	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	ctx, batal := context.WithTimeout(context.Background(), 120*time.Second)
	defer batal()

	for _, line := range reportkpi.BusinessLines() {
		fmt.Printf("%s%s\n", "=== lini ", line.Code)
		q := reportkpi.PICQuery{Line: line.Code, From: from, To: to}

		lapor("pic_list", func() (int, error) {
			rows, err := repo.PICs(ctx, line.Code)
			return len(rows), err
		})
		lapor("progress_counts", func() (int, error) {
			rows, err := repo.ProgressCounts(ctx, q)
			return len(rows), err
		})
		lapor("analysis_spans", func() (int, error) {
			rows, err := repo.AnalysisSpans(ctx, q)
			return len(rows), err
		})
		lapor("acceptance_spans", func() (int, error) {
			rows, err := repo.AcceptanceSpans(ctx, q)
			return len(rows), err
		})
		lapor("closure_spans", func() (int, error) {
			rows, err := repo.ClosureSpans(ctx, q)
			return len(rows), err
		})
	}

	lapor("holidays", func() (int, error) {
		days, err := repo.Holidays(ctx, from, to)
		return len(days), err
	})

	cekTanggaProgress(ctx, db, from, to)
	cekCiriProgress(ctx, db, from, to)
	cekDaftarAdjuster(ctx, db)
	cekRingkasanAdjuster(ctx, repo, db)
	cekAdmin(ctx, repo)
	cekPengulanganBind(ctx, db)
	cekPemilikView(ctx, db)
	cekObjekASMD(ctx, db)
	cekPenggantiDatapega(ctx, db)

	return cekEkspor(ctx, repo, from, to)
}

// cekEkspor menjalankan keempat kueri ekspor "Pilih Data KPI".
//
// # Kenapa NONMBU saja
//
// Tab ini memang selalu NONMBU â€” lihat `D-70` dan catatan Â§31.
//
// Jumlah kolom ikut dicetak supaya dapat dibandingkan dengan Pega: 43 Â· 38 Â· 51 Â· 51.
//
// # Nol baris BUKAN berarti lulus
//
// Kueri yang sah tetapi penyaringnya terlalu ketat juga mengembalikan nol baris, dan dari
// luar keduanya tampak sama. Karena itu jendelanya dibuat selebar pemeriksaan lain, dan
// bila hasilnya tetap nol, sebabnya dicari ke tabel dasarnya â€” bukan dianggap wajar.
func cekEkspor(ctx context.Context, repo *reportkpisql.Repo, from, to time.Time) error {
	fmt.Printf("=== ekspor Pilih Data KPI (NONMBU, %s s.d. %s)\n",
		from.Format("2006-01-02"), to.Format("2006-01-02"))

	span := reportkpi.PICQuery{Line: reportkpi.LineNonMBU, From: from, To: to}

	profiles, err := repo.PICs(ctx, span.Line)
	if err != nil {
		return fmt.Errorf("daftar PIC: %w", err)
	}
	pics := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		pics = append(pics, profile.OperatorID)
	}
	fmt.Printf("  daftar PIC: %d orang\n", len(pics))

	for _, option := range reportkpi.PICExportOptions() {
		nama := option.Label
		table, err := repo.PICExport(ctx, option.Code, span, pics)
		if err != nil {
			fmt.Printf("  [GAGAL] %-22s %v\n", nama, err)
			continue
		}
		fmt.Printf("  [ok]    %-22s %d kolom, %d baris\n",
			nama, len(table.Header), len(table.Rows))
	}
	return nil
}

// lapor menjalankan satu pembacaan dan menuliskan hasilnya dalam satu baris.
func lapor(nama string, jalan func() (int, error)) {
	jumlah, err := jalan()
	if err != nil {
		fmt.Printf("  [GAGAL] %-18s %v\n", nama, err)
		return
	}
	fmt.Printf("  [ok]    %-18s %d baris\n", nama, jumlah)
}

// bacaEnv membaca berkas .env sederhana: KUNCI=nilai, baris komentar diabaikan.
func bacaEnv(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out, nil
}

// cekTanggaProgress mencari predikat mana yang menghabiskan seluruh baris Progress.
//
// Keempat kueri ekspor berjalan, tetapi Progress mengembalikan nol baris sementara tiga
// lainnya tidak â€” dan `progress_counts` juga nol di seluruh lini. Dua gejala pada jalur
// yang sama bukan kebetulan.
//
// Caranya: predikat ditambahkan SATU PER SATU. Angka yang jatuh ke nol menunjuk
// penyebabnya tanpa perlu menebak.
func cekTanggaProgress(ctx context.Context, db *sql.DB, from, to time.Time) {
	fmt.Println("=== tangga predikat Progress")

	tangga := []struct {
		nama  string
		where string
	}{
		{"c saja + tanggal", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY`},
		{"+ join b.PIC = c.USER_INPUT", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT`},
		{"+ b.PNCCASEID = c.PNCCASEID", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID`},
		{"+ STSKLAIM NOT IN", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID
			  AND b.STSKLAIM NOT IN ('1', '2', '3')`},
		{"+ KETERANGAN bukan AUTO/SYSTEM", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID
			  AND b.STSKLAIM NOT IN ('1', '2', '3')
			  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
			  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'`},
		{"+ PIC <> ASNET", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID
			  AND b.STSKLAIM NOT IN ('1', '2', '3')
			  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
			  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'
			  AND b.PIC <> 'ASNET'`},
		{"+ GROUP_PANEL IN", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID
			  AND b.STSKLAIM NOT IN ('1', '2', '3')
			  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
			  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'
			  AND b.PIC <> 'ASNET'
			  AND b.GROUP_PANEL IN ('003', '004', '006')`},
		{"+ GROUPBISNISID NOT IN", `
			FROM pooldata.GCNM_PROGRESS_CLAIM c, pooldata.PEGA_DASHBOARDPNC b
			WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
			  AND b.PIC = c.USER_INPUT AND b.NOKLAIM = c.PNCCASEID
			  AND b.STSKLAIM NOT IN ('1', '2', '3')
			  AND UPPER(c.KETERANGAN) NOT LIKE '%AUTO%'
			  AND UPPER(c.KETERANGAN) NOT LIKE '%SYSTEM%'
			  AND b.PIC <> 'ASNET'
			  AND b.GROUP_PANEL IN ('003', '004', '006')
			  AND b.GROUPBISNISID NOT IN ('09', '11', '16', '25')`},
	}

	for _, langkah := range tangga {
		var jumlah int64
		err := db.QueryRowContext(ctx, "SELECT COUNT(*) "+langkah.where, from, to).Scan(&jumlah)
		if err != nil {
			fmt.Printf("  [GAGAL] %-32s %v\n", langkah.nama, err)
			continue
		}
		fmt.Printf("  %-32s %d\n", langkah.nama, jumlah)
	}
}

// cekCiriProgress mencirikan 13 baris yang bertahan sampai sebelum penyaring STSKLAIM.
//
// Tangga predikat menunjuk `STSKLAIM NOT IN ('1','2','3')` sebagai yang menghabiskan
// seluruh baris. Yang belum terjawab: apakah itu karena datanya memang semua berstatus
// 1/2/3, atau karena yang sampai ke titik itu sudah terlalu sedikit sejak awal.
//
// Dua angka menjawabnya â€” sebaran STSKLAIM pada ketiga belas baris, dan ukuran
// PEGA_DASHBOARDPNC itu sendiri.
func cekCiriProgress(ctx context.Context, db *sql.DB, from, to time.Time) {
	fmt.Println("=== ciri data Progress")

	rows, err := db.QueryContext(ctx, `
		SELECT b.STSKLAIM, COUNT(*)
		  FROM pooldata.GCNM_PROGRESS_CLAIM c
		  JOIN pooldata.PEGA_DASHBOARDPNC b ON b.NOKLAIM = c.PNCCASEID
		 WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY
		   AND b.PIC = c.USER_INPUT
		 GROUP BY b.STSKLAIM
		 ORDER BY b.STSKLAIM`, from, to)
	if err != nil {
		fmt.Printf("  [GAGAL] sebaran STSKLAIM: %v\n", err)
	} else {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var status sql.NullString
			var jumlah int64
			if err := rows.Scan(&status, &jumlah); err != nil {
				fmt.Printf("  [GAGAL] baca sebaran: %v\n", err)
				break
			}
			fmt.Printf("  STSKLAIM %-6s %d baris\n", status.String, jumlah)
		}
	}

	// Join TANPA b.PIC = c.USER_INPUT. Selisihnya terhadap 13 memisahkan dua dugaan yang
	// sangat berbeda akibatnya: nomor klaimnya tidak cocok, atau nomor klaimnya cocok
	// tetapi PIC klaim jarang sama dengan orang yang menulis progresnya.
	var tanpaPIC int64
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		  FROM pooldata.GCNM_PROGRESS_CLAIM c
		  JOIN pooldata.PEGA_DASHBOARDPNC b ON b.NOKLAIM = c.PNCCASEID
		 WHERE c.TGL_INPUT >= :1 AND c.TGL_INPUT < :2 + INTERVAL '1' DAY`, from, to).
		Scan(&tanpaPIC)
	if err != nil {
		fmt.Printf("  [GAGAL] join tanpa PIC: %v\n", err)
	} else {
		fmt.Printf("  join nomor klaim saja (tanpa b.PIC = c.USER_INPUT): %d baris\n", tanpaPIC)
	}

	// Pega menulis `keterangan` dan `TGL_INPUT` TANPA alias tabel. Itu hanya sah bila
	// kolomnya ada di salah satu tabel saja; bila ada di keduanya, Oracle menolak dengan
	// ORA-00918 dan kueri Pega bukan "mengembalikan kosong" melainkan GAGAL â€” cerita yang
	// sama sekali berbeda. Pemberian alias `c.` di sisi kami hanya benar bila kolomnya
	// memang milik `c`.
	for _, kolom := range []string{"KETERANGAN", "TGL_INPUT"} {
		var punyaC, punyaB int64
		err := db.QueryRowContext(ctx, `
			SELECT
			  (SELECT COUNT(*) FROM ALL_TAB_COLUMNS
			    WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'GCNM_PROGRESS_CLAIM'
			      AND COLUMN_NAME = :1),
			  (SELECT COUNT(*) FROM ALL_TAB_COLUMNS
			    WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'PEGA_DASHBOARDPNC'
			      AND COLUMN_NAME = :2)
			FROM DUAL`, kolom, kolom).Scan(&punyaC, &punyaB)
		if err != nil {
			fmt.Printf("  [GAGAL] kepemilikan %s: %v\n", kolom, err)
			continue
		}
		fmt.Printf("  kolom %-11s GCNM_PROGRESS_CLAIM=%d  PEGA_DASHBOARDPNC=%d%s\n",
			kolom, punyaC, punyaB,
			map[bool]string{true: "  <-- AMBIGU di Pega", false: ""}[punyaC > 0 && punyaB > 0])
	}

	var total, klaim int64
	var awal, akhir sql.NullTime
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT NOKLAIM) FROM pooldata.PEGA_DASHBOARDPNC`).
		Scan(&total, &klaim)
	if err != nil {
		fmt.Printf("  [GAGAL] ukuran PEGA_DASHBOARDPNC: %v\n", err)
	} else {
		fmt.Printf("  PEGA_DASHBOARDPNC: %d baris, %d nomor klaim\n", total, klaim)
	}

	err = db.QueryRowContext(ctx, `
		SELECT MIN(TGL_INPUT), MAX(TGL_INPUT) FROM pooldata.GCNM_PROGRESS_CLAIM`).
		Scan(&awal, &akhir)
	if err != nil {
		fmt.Printf("  [GAGAL] rentang GCNM_PROGRESS_CLAIM: %v\n", err)
	} else {
		fmt.Printf("  GCNM_PROGRESS_CLAIM TGL_INPUT: %s s.d. %s\n",
			tgl(awal), tgl(akhir))
	}
}

func tgl(v sql.NullTime) string {
	if !v.Valid {
		return "(kosong)"
	}
	return v.Time.Format("2006-01-02")
}

// cekDaftarAdjuster memeriksa sumber dropdown "Pilih Adjuster" pada tab KPI Adjuster.
//
// Kuerinya disalin dari `RDB List/BrowseAdjsuterExternal-SQL.xml` â€” rule yang ADA di
// export, hanya belum pernah dipakai di sisi kami. Yang diperiksa: view-nya terbaca dari
// koneksi portal, dan berapa nama yang akan muncul di dropdown.
func cekDaftarAdjuster(ctx context.Context, db *sql.DB) {
	fmt.Println("=== sumber dropdown Pilih Adjuster")

	var jumlah int64
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM V_D_SURVEYORS WHERE M_SURVEY_ID IN ('1004', '1002')`).
		Scan(&jumlah)
	if err != nil {
		fmt.Printf("  [GAGAL] V_D_SURVEYORS: %v\n", err)
		return
	}
	fmt.Printf("  [ok]    V_D_SURVEYORS m_survey_id IN ('1004','1002'): %d nama\n", jumlah)

	// Sebaran per kode supaya terlihat apakah KEDUA kode benar-benar terpakai. Bila salah
	// satunya nol, penyaring itu sebenarnya hanya menyaring satu kelompok â€” dan itu perlu
	// diketahui sebelum dipakai sebagai acuan.
	rows, err := db.QueryContext(ctx, `
		SELECT M_SURVEY_ID, COUNT(*) FROM V_D_SURVEYORS
		 WHERE M_SURVEY_ID IN ('1004', '1002')
		 GROUP BY M_SURVEY_ID ORDER BY M_SURVEY_ID`)
	if err != nil {
		fmt.Printf("  [GAGAL] sebaran M_SURVEY_ID: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kode sql.NullString
		var n int64
		if err := rows.Scan(&kode, &n); err != nil {
			fmt.Printf("  [GAGAL] baca sebaran: %v\n", err)
			return
		}
		fmt.Printf("          M_SURVEY_ID %-6s %d nama\n", kode.String, n)
	}

	// Nama kosong akan menjadi baris dropdown yang tidak dapat dipilih maknanya.
	var kosong int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM V_D_SURVEYORS
		 WHERE M_SURVEY_ID IN ('1004', '1002') AND (NAME IS NULL OR TRIM(NAME) = '')`).
		Scan(&kosong); err == nil && kosong > 0 {
		fmt.Printf("          PERHATIAN: %d baris NAME kosong\n", kosong)
	}
}

// cekPemilikView menyebut SIAPA pemilik V_D_SURVEYORS.
//
// Kueri Pega menulisnya tanpa nama skema, sehingga ia bergantung pada skema sesi atau
// sinonim. Menambahkan awalan `POOLDATA.` hanya benar bila view-nya memang milik POOLDATA;
// bila ia sinonim ke skema lain, awalan itu justru memutus kueri yang tadinya jalan.
func cekPemilikView(ctx context.Context, db *sql.DB) {
	fmt.Println("=== pemilik V_D_SURVEYORS")

	rows, err := db.QueryContext(ctx, `
		SELECT OWNER, OBJECT_TYPE FROM ALL_OBJECTS
		 WHERE OBJECT_NAME = 'V_D_SURVEYORS' ORDER BY OWNER`)
	if err != nil {
		fmt.Printf("  [GAGAL] ALL_OBJECTS: %v\n", err)
		return
	}
	defer func() { _ = rows.Close() }()

	ada := false
	for rows.Next() {
		var owner, jenis string
		if err := rows.Scan(&owner, &jenis); err != nil {
			fmt.Printf("  [GAGAL] baca baris: %v\n", err)
			return
		}
		fmt.Printf("  %-14s %s\n", owner, jenis)
		ada = true
	}
	if !ada {
		fmt.Println("  tidak terlihat di ALL_OBJECTS â€” kemungkinan lewat sinonim publik")
	}

	// Uji langsung: apakah penulisan BERSKEMA benar-benar jalan?
	var n int64
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM POOLDATA.V_D_SURVEYORS WHERE M_SURVEY_ID IN ('1004','1002')`).
		Scan(&n)
	if err != nil {
		fmt.Printf("  POOLDATA.V_D_SURVEYORS: TIDAK jalan (%v)\n", err)
		return
	}
	fmt.Printf("  POOLDATA.V_D_SURVEYORS: jalan, %d nama\n", n)
}

// cekRingkasanAdjuster menjalankan kueri tab KPI Adjuster APA ADANYA.
//
// Layar melaporkan "Ringkasan tidak dapat diambil / Terjadi kesalahan pada sistem" — galat
// 500 yang sengaja TIDAK menyebut sebabnya kepada pengguna. Sebabnya hanya terlihat di
// sini, dengan kueri dan parameter yang sama persis dengan yang ditekan di layar.
func cekRingkasanAdjuster(ctx context.Context, repo *reportkpisql.Repo, db *sql.DB) {
	fmt.Println("=== tab KPI Adjuster (ALL, 2020-01-01 s.d. 2026-12-31)")

	q, err := reportkpi.NewQuery(reportkpi.QueryInput{
		ReportType: "ALL",
		From:       "2020-01-01",
		To:         "2026-12-31",
	}, reportkpi.Caller{Login: "cekkueri"})
	if err != nil {
		fmt.Printf("  [GAGAL] menyusun permintaan: %v\n", err)
		return
	}

	lapor("summary", func() (int, error) {
		rows, err := repo.Summary(ctx, q)
		return len(rows), err
	})
	lapor("detail", func() (int, error) {
		page, err := repo.Detail(ctx, q, reportkpi.Pagination{Page: 1, Size: 50})
		return len(page.Rows), err
	})
	lapor("adjusters", func() (int, error) {
		rows, err := repo.Adjusters(ctx)
		return len(rows), err
	})

	// Pita kolom KATEGORI. Ia dibaca dari tabel yang sama dengan tangga nilai PIC tetapi
	// dengan `TIPE = 'ADJUSTER'`, dan penyaring itulah satu-satunya yang membedakan —
	// sehingga salah menuliskannya menghasilkan nol baris, bukan galat.
	kategori, err := repo.AdjusterCategories(ctx)
	if err != nil {
		fmt.Printf("  [GAGAL] pita kategori: %v\n", err)
	} else {
		fmt.Printf("  [ok]    kategori           %d pita\n", len(kategori))
		for _, p := range kategori {
			fmt.Printf("          %g – %g  %s\n", p.Bottom, p.Top, p.Note)
		}
	}

	// Nol baris pada kueri yang SAH tidak membedakan "tabelnya kosong" dari "penyaringnya
	// terlalu sempit". Isi tabelnya dihitung supaya keduanya terbedakan.
	state, err := repo.CheckSource(ctx)
	if err != nil {
		fmt.Printf("  [GAGAL] isi tabel sumber: %v\n", err)
		return
	}
	fmt.Printf("  isi %s: %d baris · tipe: %v\n",
		reportkpi.SourceTable, state.Rows, state.Types)

	// Rentang tanggal yang BENAR-BENAR ada. Tanpa angka ini, "nol baris" tidak dapat
	// dibedakan dari "periode yang dicari memang kosong".
	var awal, akhir sql.NullTime
	if err := db.QueryRowContext(ctx,
		`SELECT MIN(TANGGAL), MAX(TANGGAL) FROM POOLDATA.DETAIL_KPI_ADJUSTER`).
		Scan(&awal, &akhir); err != nil {
		fmt.Printf("  [GAGAL] rentang TANGGAL: %v\n", err)
		return
	}
	fmt.Printf("  rentang TANGGAL: %s s.d. %s\n", tgl(awal), tgl(akhir))
}

// cekPengulanganBind menguji bagaimana driver menghitung penanda bind YANG BERULANG.
//
// Dugaan atas ORA-01008 pada tab KPI Adjuster: `:1` yang ditulis DUA KALI dihitung sebagai
// dua bind, bukan satu. Dugaan itu diuji di sini alih-alih dipercaya — kekeliruan yang
// sama pernah terjadi, dan yang mematahkannya dulu adalah uji, bukan pembacaan ulang.
func cekPengulanganBind(ctx context.Context, db *sql.DB) {
	fmt.Println("=== perilaku penanda bind berulang")

	coba := func(nama, sql string, args ...any) {
		var n int64
		err := db.QueryRowContext(ctx, sql, args...).Scan(&n)
		if err != nil {
			fmt.Printf("  [GAGAL] %-34s %v\n", nama, ringkas(err))
			return
		}
		fmt.Printf("  [ok]    %-34s hasil=%d\n", nama, n)
	}

	// Satu penanda, satu nilai — pembanding supaya kegagalan di bawah berarti sesuatu.
	coba("satu :1, satu nilai", "SELECT COUNT(*) FROM DUAL WHERE :1 IS NOT NULL", "x")

	// `:1` DUA kali, satu nilai. Bila ini gagal, dugaannya benar.
	coba("dua kali :1, satu nilai",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :1 = 'x')", "x")

	// `:1` dua kali, nilai dikirim dua kali.
	coba("dua kali :1, dua nilai",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :1 = 'x')", "x", "x")

	// Penanda BERBEDA untuk nilai yang sama — bentuk yang diusulkan sebagai perbaikan.
	coba("penanda berbeda, dua nilai",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :2 = 'x')", "x", "x")

	// NIL telanjang. Inilah yang dikirim kueri Adjuster ketika penyaringnya kosong:
	// `nullableText("")` dan `reportTypeBind(ALL)` sama-sama mengembalikan `nil`.
	coba("nil telanjang", "SELECT COUNT(*) FROM DUAL WHERE :1 IS NULL", nil)

	coba("nil telanjang, dipakai dua kali",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :1 = 'x')", nil)

	// sql.NullString kosong — bentuk yang menyatakan NULL secara BERTIPE.
	coba("sql.NullString kosong",
		"SELECT COUNT(*) FROM DUAL WHERE :1 IS NULL", sql.NullString{})

	coba("sql.NullString kosong, dua kali",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :1 = 'x')", sql.NullString{})

	// Bentuk yang SAMA PERSIS dengan kueri `summary`: dua penanda berulang berurutan,
	// lalu dua penanda tunggal. Inilah yang gagal, dan susunan inilah yang membedakannya
	// dari percobaan di atas.
	coba("dua pasang berulang + dua tunggal",
		"SELECT COUNT(*) FROM DUAL WHERE (:1 IS NULL OR :1 = 'a') "+
			"AND (:2 IS NULL OR :2 = 'b') AND :3 IS NOT NULL AND :4 IS NOT NULL",
		nil, nil, "2025-01-01", "2026-01-10")

	// Pembanding: susunan yang sama, tetapi tiap penanda hanya SEKALI.
	coba("empat penanda, tanpa pengulangan",
		"SELECT COUNT(*) FROM DUAL WHERE :1 IS NULL AND :2 IS NULL "+
			"AND :3 IS NOT NULL AND :4 IS NOT NULL",
		nil, nil, "2025-01-01", "2026-01-10")

	fmt.Println("=== objek lintas DB Link")

	// Kalender libur — jalur yang sudah terbukti jalan (§29), sebagai pembanding.
	coba("GENERAL.HRD_LBR@ASMD",
		"SELECT COUNT(*) FROM GENERAL.HRD_LBR@asmd.sinarmas.co.id WHERE ROWNUM = 1")

	// Fungsi jam kerja — dipakai SELURUH kueri tab KPI Admin.
	coba("DATAMINING.GET_WORKING_HOURS@ASMD",
		"SELECT datamining.get_working_hours@asmd.sinarmas.co.id("+
			"TO_DATE('2026-01-01','YYYY-MM-DD'), TO_DATE('2026-01-05','YYYY-MM-DD')) FROM DUAL")

	fmt.Println("=== pemilik kolom penyaring KPI Admin")

	// `GROUP_PANEL` dan `GROUPBISNISID` ditulis TANPA alias di kueri Pega. Kueri kami
	// menempelkannya ke `b` (t_claim_pnc) dan ditolak ORA-00904. Katalog yang menentukan
	// alias mana yang benar — bukan tebakan.
	for _, kolom := range []string{"GROUP_PANEL", "GROUPBISNISID"} {
		var diClaim, diDashboard int64
		err := db.QueryRowContext(ctx, `
			SELECT
			  (SELECT COUNT(*) FROM ALL_TAB_COLUMNS WHERE OWNER = 'POOLDATA'
			     AND TABLE_NAME = 'T_CLAIM_PNC' AND COLUMN_NAME = :1),
			  (SELECT COUNT(*) FROM ALL_TAB_COLUMNS WHERE OWNER = 'POOLDATA'
			     AND TABLE_NAME = 'PEGA_DASHBOARDPNC' AND COLUMN_NAME = :2)
			FROM DUAL`, kolom, kolom).Scan(&diClaim, &diDashboard)
		if err != nil {
			fmt.Printf("  [GAGAL] %s: %v\n", kolom, err)
			continue
		}
		fmt.Printf("  %-14s T_CLAIM_PNC=%d  PEGA_DASHBOARDPNC=%d\n",
			kolom, diClaim, diDashboard)
	}
}

// ringkas memotong pesan galat Oracle pada baris pertamanya.
func ringkas(err error) string {
	pesan := err.Error()
	if i := strings.IndexAny(pesan, "\r\n"); i > 0 {
		return pesan[:i]
	}
	return pesan
}

// cekAdmin menjalankan kueri tab KPI Admin APA ADANYA.
func cekAdmin(ctx context.Context, repo *reportkpisql.Repo) {
	fmt.Println("=== tab KPI Admin (2020-01-01 s.d. 2026-12-31)")

	for _, kelompok := range reportkpi.AdminGroups() {
		q, err := reportkpi.NewAdminQuery(reportkpi.AdminQueryInput{
			Group: string(kelompok.Code),
			From:  "2020-01-01",
			To:    "2026-12-31",
		}, reportkpi.Caller{Login: "cekkueri"})
		if err != nil {
			fmt.Printf("  [GAGAL] menyusun permintaan %s: %v\n", kelompok.Code, err)
			continue
		}

		lapor(string(kelompok.Code)+" kartu-skor", func() (int, error) {
			_, err := repo.AdminTotals(ctx, q)
			return 1, err
		})
		lapor(string(kelompok.Code)+" rincian", func() (int, error) {
			page, err := repo.AdminDetail(ctx, q, reportkpi.Pagination{Page: 1, Size: 50})
			return len(page.Rows), err
		})
	}
}

// cekObjekASMD menjelaskan DUA nama yang sering disebut: DATAMINING dan HRD_LBR.
//
// Keduanya berada di basis data lain yang dicapai lewat DB Link `@asmd.sinarmas.co.id`.
// Yang diperiksa di sini: skema `DATAMINING` terlihat atau tidak, dan isi `HRD_LBR`
// sebenarnya apa.
func cekObjekASMD(ctx context.Context, db *sql.DB) {
	fmt.Println("=== isi HRD_LBR (kalender libur)")

	rows, err := db.QueryContext(ctx, `
		SELECT COLUMN_NAME, DATA_TYPE
		  FROM ALL_TAB_COLUMNS@asmd.sinarmas.co.id
		 WHERE OWNER = 'GENERAL' AND TABLE_NAME = 'HRD_LBR'
		 ORDER BY COLUMN_ID`)
	if err != nil {
		fmt.Printf("  [GAGAL] kolom HRD_LBR: %v\n", ringkas(err))
	} else {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var nama, jenis string
			if err := rows.Scan(&nama, &jenis); err == nil {
				fmt.Printf("  kolom %-18s %s\n", nama, jenis)
			}
		}
	}

	var jumlah int64
	var awal, akhir sql.NullTime
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(*), MIN(TANGGAL), MAX(TANGGAL)
		  FROM GENERAL.HRD_LBR@asmd.sinarmas.co.id`).Scan(&jumlah, &awal, &akhir); err != nil {
		fmt.Printf("  [GAGAL] isi HRD_LBR: %v\n", ringkas(err))
	} else {
		fmt.Printf("  %d baris · %s s.d. %s\n", jumlah, tgl(awal), tgl(akhir))
	}

	fmt.Println("=== skema DATAMINING di basis data seberang")

	rows2, err := db.QueryContext(ctx, `
		SELECT OBJECT_TYPE, COUNT(*)
		  FROM ALL_OBJECTS@asmd.sinarmas.co.id
		 WHERE OWNER = 'DATAMINING'
		 GROUP BY OBJECT_TYPE ORDER BY OBJECT_TYPE`)
	if err != nil {
		fmt.Printf("  [GAGAL] baca ALL_OBJECTS: %v\n", ringkas(err))
		return
	}
	defer func() { _ = rows2.Close() }()

	ada := false
	for rows2.Next() {
		var jenis string
		var n int64
		if err := rows2.Scan(&jenis, &n); err == nil {
			fmt.Printf("  %-18s %d objek\n", jenis, n)
			ada = true
		}
	}
	if !ada {
		fmt.Println("  TIDAK ADA satu objek pun yang terlihat oleh pengguna DB Link")
		fmt.Println("  (skemanya mungkin ada, tetapi tanpa hak akses ia tidak tampak)")
	}

	fmt.Println("=== mencari GET_WORKING_HOURS")

	// Dicari di SELURUH pemilik, bukan hanya DATAMINING. Bila ia ternyata milik skema
	// lain, yang dibutuhkan bukan hak akses melainkan nama yang benar.
	rows3, err := db.QueryContext(ctx, `
		SELECT OWNER, OBJECT_NAME, OBJECT_TYPE, STATUS
		  FROM ALL_OBJECTS@asmd.sinarmas.co.id
		 WHERE OBJECT_NAME LIKE '%WORKING_HOUR%'
		 ORDER BY OWNER, OBJECT_NAME`)
	if err != nil {
		fmt.Printf("  [GAGAL] mencari: %v\n", ringkas(err))
		return
	}
	defer func() { _ = rows3.Close() }()

	ketemu := false
	for rows3.Next() {
		var owner, nama, jenis, status string
		if err := rows3.Scan(&owner, &nama, &jenis, &status); err == nil {
			fmt.Printf("  %-14s %-24s %-10s %s\n", owner, nama, jenis, status)
			ketemu = true
		}
	}
	if !ketemu {
		fmt.Println("  TIDAK DITEMUKAN — tidak ada objek bernama *WORKING_HOUR* yang")
		fmt.Println("  terlihat oleh pengguna DB Link di basis data seberang")
	}

	// Yang MIRIP, supaya permintaan ke Infra dapat menyebut nama yang benar bila
	// fungsinya ternyata berganti nama. Paket ikut didaftar: fungsi DI DALAM paket tidak
	// muncul sebagai objek tersendiri, dan pemanggilannya pun berbeda bentuk.
	fmt.Println("=== kandidat nama lain di DATAMINING")
	rows4, err := db.QueryContext(ctx, `
		SELECT OBJECT_NAME, OBJECT_TYPE
		  FROM ALL_OBJECTS@asmd.sinarmas.co.id
		 WHERE OWNER = 'DATAMINING'
		   AND (OBJECT_TYPE = 'PACKAGE'
		        OR OBJECT_NAME LIKE '%HOUR%' OR OBJECT_NAME LIKE '%JAM%'
		        OR OBJECT_NAME LIKE '%KERJA%' OR OBJECT_NAME LIKE '%WORK%'
		        OR OBJECT_NAME LIKE '%SELISIH%' OR OBJECT_NAME LIKE '%TAT%')
		 ORDER BY OBJECT_TYPE, OBJECT_NAME`)
	if err != nil {
		fmt.Printf("  [GAGAL] mencari kandidat: %v\n", ringkas(err))
		return
	}
	defer func() { _ = rows4.Close() }()
	for rows4.Next() {
		var nama, jenis string
		if err := rows4.Scan(&nama, &jenis); err == nil {
			fmt.Printf("  %-12s %s\n", jenis, nama)
		}
	}
}

// cekPenggantiDatapega mencari kolom POOLDATA yang dapat menggantikan `datapega`.
//
// Kueri tab KPI Admin menumpang `datapega.pc_asm_fw_gcnmfw_work d` untuk DUA hal saja:
//
//	d.pxcreateoperator  siapa admin yang membuat klaim  -> penyaring utama
//	d.pystatuswork      status pekerjaan                -> dipakai kolom STATUS
//
// Keduanya tabel ENGINE Pega (`D-21`). Yang dicari di sini: kolom setara di tabel milik
// POOLDATA, supaya kueri tidak lagi bergantung pada tabel Pega.
func cekPenggantiDatapega(ctx context.Context, db *sql.DB) {
	fmt.Println("=== kandidat pengganti kolom datapega")

	for _, tabel := range []string{"T_CLAIM_PNC", "PEGA_DASHBOARDPNC"} {
		rows, err := db.QueryContext(ctx, `
			SELECT COLUMN_NAME, DATA_TYPE
			  FROM ALL_TAB_COLUMNS
			 WHERE OWNER = 'POOLDATA' AND TABLE_NAME = :1
			   AND (COLUMN_NAME LIKE '%OPERATOR%' OR COLUMN_NAME LIKE '%USER%'
			        OR COLUMN_NAME LIKE '%CREATE%' OR COLUMN_NAME LIKE '%ADMIN%'
			        OR COLUMN_NAME LIKE '%STATUSWORK%' OR COLUMN_NAME LIKE '%STSWORK%'
			        OR COLUMN_NAME LIKE '%PIC%' OR COLUMN_NAME LIKE '%INPUT%')
			 ORDER BY COLUMN_NAME`, tabel)
		if err != nil {
			fmt.Printf("  [GAGAL] %s: %v\n", tabel, ringkas(err))
			continue
		}
		fmt.Printf("  --- POOLDATA.%s ---\n", tabel)
		for rows.Next() {
			var nama, jenis string
			if err := rows.Scan(&nama, &jenis); err == nil {
				fmt.Printf("      %-24s %s\n", nama, jenis)
			}
		}
		_ = rows.Close()
	}

	// Apakah isinya benar-benar cocok? Dibandingkan langsung terhadap datapega untuk
	// klaim yang sama — kalau cocok, penggantinya aman; kalau tidak, ia bukan pengganti.
	fmt.Println("  --- kecocokan isi terhadap datapega ---")

	pasangan := []struct{ nama, kiri, kanan string }{
		{"ADMINKLAIM vs pxcreateoperator", "b.ADMINKLAIM", "d.pxcreateoperator"},
		{"STATUSWORK vs pystatuswork", "b.STATUSWORK", "d.pystatuswork"},
	}
	for _, p := range pasangan {
		var cocok, beda, kosongKiri int64
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(CASE WHEN `+p.kiri+` = `+p.kanan+` THEN 1 END),
			       COUNT(CASE WHEN `+p.kiri+` <> `+p.kanan+` THEN 1 END),
			       COUNT(CASE WHEN `+p.kiri+` IS NULL AND `+p.kanan+` IS NOT NULL THEN 1 END)
			  FROM (SELECT * FROM POOLDATA.T_CLAIM_PNC WHERE ROWNUM <= 20000) b
			  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK d ON b.CLAIMID = d.pzinskey`).
			Scan(&cocok, &beda, &kosongKiri)
		if err != nil {
			fmt.Printf("      %-34s tidak dapat diuji (%s)\n", p.nama, ringkas(err))
			continue
		}
		fmt.Printf("      %-34s cocok %d · beda %d · kiri kosong %d\n",
			p.nama, cocok, beda, kosongKiri)
	}

	// Pencarian lebih luas: kolom POOLDATA mana pun yang ISINYA cocok dengan
	// `pxcreateoperator`. Nama tidak dipercaya — yang dinilai kecocokan nilainya.
	fmt.Println("  --- pencarian luas: kolom teks T_CLAIM_PNC yang isinya cocok ---")
	kolom, err := db.QueryContext(ctx, `
		SELECT COLUMN_NAME FROM ALL_TAB_COLUMNS
		 WHERE OWNER = 'POOLDATA' AND TABLE_NAME = 'T_CLAIM_PNC'
		   AND DATA_TYPE = 'VARCHAR2' AND DATA_LENGTH BETWEEN 5 AND 120
		 ORDER BY COLUMN_NAME`)
	if err != nil {
		fmt.Printf("      [GAGAL] daftar kolom: %v\n", ringkas(err))
		return
	}
	var nama []string
	for kolom.Next() {
		var n string
		if err := kolom.Scan(&n); err == nil {
			nama = append(nama, n)
		}
	}
	_ = kolom.Close()

	for _, n := range nama {
		var cocok, isi int64
		err := db.QueryRowContext(ctx, `
			SELECT COUNT(CASE WHEN b."`+n+`" = d.pxcreateoperator THEN 1 END),
			       COUNT(CASE WHEN b."`+n+`" IS NOT NULL THEN 1 END)
			  FROM (SELECT * FROM POOLDATA.T_CLAIM_PNC WHERE ROWNUM <= 20000) b
			  JOIN DATAPEGA.PC_ASM_FW_GCNMFW_WORK d ON b.CLAIMID = d.pzinskey`).
			Scan(&cocok, &isi)
		if err != nil || cocok == 0 {
			continue
		}
		fmt.Printf("      %-26s cocok %d dari %d terisi\n", n, cocok, isi)
	}

	// Apakah DBA sudah membuat tabel PENGGANTI `datapega` (`D-21`)? Dicari dengan dua
	// cara: awalan penamaan sistem baru (`CPNC_`), dan nama yang menyiratkan tabel kerja
	// klaim. Tanggal pembuatan ikut dicetak supaya yang baru terlihat.
	fmt.Println("  --- tabel pengganti datapega di POOLDATA ---")
	rows, err := db.QueryContext(ctx, `
		SELECT OBJECT_NAME, OBJECT_TYPE, TO_CHAR(CREATED, 'YYYY-MM-DD')
		  FROM ALL_OBJECTS
		 WHERE OWNER = 'POOLDATA'
		   AND OBJECT_TYPE IN ('TABLE', 'VIEW')
		   AND (OBJECT_NAME LIKE 'CPNC%'
		        OR OBJECT_NAME LIKE '%GCNMFW%'
		        OR OBJECT_NAME LIKE '%ASSIGN%'
		        OR OBJECT_NAME LIKE '%WORKLIST%')
		 ORDER BY OBJECT_NAME`)
	if err != nil {
		fmt.Printf("      [GAGAL] %v\n", ringkas(err))
		return
	}
	defer func() { _ = rows.Close() }()

	ada := false
	for rows.Next() {
		var nama, jenis, dibuat string
		if err := rows.Scan(&nama, &jenis, &dibuat); err == nil {
			fmt.Printf("      %-32s %-6s dibuat %s\n", nama, jenis, dibuat)
			ada = true
		}
	}
	if !ada {
		fmt.Println("      NIHIL — tidak ada tabel pengganti di POOLDATA")
	}
}
