// Package sqlstore memenuhi seam reportklaim.Repo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (`ADR-0030` Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
//
// # Kenapa satu pemindai untuk 26 kueri
//
// Karena setiap kueri sudah MENAMAI kolomnya persis seperti nama properti pada
// `CSVProperties` milik langkah `pxConvertResultsToCSV`. Dengan begitu baris hasil dapat
// dipetakan ke reportklaim.Row tanpa satu baris pun kode pemetaan per laporan — dan yang
// menjamin kolomnya benar adalah aliasnya sendiri, bukan urutan pemindaian yang harus
// dijaga sama di dua tempat.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
)

// ErrQueryNotPorted: kueri laporannya belum selesai dipindahkan dari export Pega.
//
// Ia DIBEDAKAN dari galat basis data biasa supaya lapisan transport dapat menjawabnya
// dengan keterangan yang benar — ini bukan gangguan yang akan pulih sendiri bila dicoba
// lagi, melainkan bagian modul yang memang belum selesai.
//
// Menyembunyikannya sebagai hasil kosong akan membuat pengguna menyimpulkan tidak ada
// data pada periode itu; menyembunyikannya sebagai galat teknis akan membuat orang
// mencarinya di basis data.
var ErrQueryNotPorted = errors.New("reportklaim/sqlstore: kueri laporan belum dipindahkan")

// Repo menjalankan kueri laporan atas basis data satu portal.
type Repo struct {
	db *sql.DB

	// aneka adalah koneksi KEDUA milik portal yang sama — pengganti DB Link `@ASMD`
	// (keputusan Work Owner 2026-09-24, keputusan-implementasi.md §49).
	//
	// BOLEH nil, dan itu keadaan yang sah: portal yang belum punya blok
	// `ANEKA_<PORTAL_ALIAS>_*` tetap dilayani, dan yang hilang hanyalah kolom laporan
	// yang membutuhkannya. Setiap pemakaian di bawah memeriksanya lebih dulu.
	aneka *sql.DB
}

// NewRepo membentuk repo.
//
// db wajib sudah terhubung ke basis data portal yang dimaksud. aneka adalah koneksi
// kedua portal yang SAMA — boleh nil.
//
// Keduanya diminta bersamaan, bukan dipasang belakangan lewat setter, supaya tidak ada
// keadaan antara di mana repo sudah dapat dipakai tetapi koneksi keduanya belum
// terpasang. Pada keadaan seperti itu laporan akan mengosongkan kolom yang sebenarnya
// dapat diisi, dan tidak ada yang terlihat keliru.
func NewRepo(db, aneka *sql.DB) *Repo { return &Repo{db: db, aneka: aneka} }

// plan menghubungkan satu laporan ke kueri dan parameternya.
type plan struct {
	// query memilih nama kueri. Ia fungsi, bukan nilai tetap, karena tiga laporan
	// memakai kueri yang BERBEDA untuk Non-MBU — dan nama kuerinya sendiri yang
	// menyatakannya (`...NONMBU`).
	query func(reportklaim.Filter) string

	// args menyusun nilai bind sesuai urutan yang didokumentasikan di berkas .sql.
	args func(reportklaim.Filter) []any

	// derive menyusun kolom yang TIDAK dapat dihasilkan SQL portabel.
	//
	// # Kenapa ada sama sekali
	//
	// Tiga hal yang di sistem lama dikerjakan SQL tidak dapat ditulis secara portabel:
	//
	//	periode "2026-09"        TO_CHAR(tanggal,'YYYY-MM')   — TO_CHAR dilarang §4
	//	singkatan bulan "SEP"    SUBSTR(tanggal,4,3)          — bergantung NLS server
	//	jumlah hari kerja        kalender libur lewat DB Link — `D-50`: ditulis di Go
	//
	// Ketiganya karena itu dihitung di Go, sama seperti pemformatan tanggal di seluruh
	// modul ini. Yang di SQL hanyalah BAHANNYA.
	//
	// Nil berarti baris dipakai apa adanya.
	derive deriveBuilder
}

// satu menyusun pemilih kueri yang selalu mengembalikan nama yang sama.
func satu(name string) func(reportklaim.Filter) string {
	return func(reportklaim.Filter) string { return name }
}

// fixed menyusun plan berkueri tunggal tanpa kolom turunan.
func fixed(name string, args func(reportklaim.Filter) []any) plan {
	return plan{query: satu(name), args: args}
}

// branch adalah satu cabang laporan berkueri ganda: kueri beserta parameternya.
//
// # Kenapa parameternya melekat pada cabang, bukan pada laporan
//
// Karena kedua cabang tidak menerima parameter yang sama. Susunan ringkas menyaring lini
// bisnis lewat bind; susunan rinci Non-MBU tidak — lininya sudah tertanam di dalam
// kuerinya sebagai daftar `group_panel`, dan yang dibutuhkannya justru parameter yang
// tidak dikenal susunan ringkas.
//
// Sebelum ini keduanya berbagi satu penyusun parameter, dan itu memaksa kueri Non-MBU
// menerima bind yang tidak dipakainya. Bind yang tidak dipakai bukan sekadar tidak rapi:
// jumlah bind yang tidak cocok dengan jumlah penanda adalah galat saat dijalankan, dan
// galatnya baru muncul ketika seseorang benar-benar menekan tombolnya.
type branch struct {
	query string
	args  func(reportklaim.Filter) []any
}

// byLine menyusun plan yang kueri DAN parameternya berbeda untuk Non-MBU.
//
// derive boleh nil bila laporannya tidak punya kolom turunan. Bila ada, penyusunnya
// menerima Filter yang sama sehingga dapat ikut bercabang — dan memang harus: susunan
// rinci punya kolom turunan yang tidak ada sama sekali di susunan ringkas.
func byLine(nonMBU, other branch, derive deriveBuilder) plan {
	pick := func(f reportklaim.Filter) branch {
		if f.BusinessLine == reportklaim.BusinessLineNonMBU {
			return nonMBU
		}
		return other
	}
	return plan{
		query:  func(f reportklaim.Filter) string { return pick(f).query },
		args:   func(f reportklaim.Filter) []any { return pick(f).args(f) },
		derive: derive,
	}
}

// Urutan bind yang dipakai seluruh kueri berpenyaring tanggal dan lini bisnis.
//
// Ia diseragamkan dengan sengaja. Dua puluh enam kueri dengan urutan bind yang
// berbeda-beda berarti dua puluh enam kesempatan menukar "dari" dengan "sampai" — dan
// tertukarnya tidak menghasilkan galat, melainkan berkas kosong.
func dateAndLine(f reportklaim.Filter) []any {
	return []any{f.From, f.To, string(f.BusinessLine)}
}

func dateOnly(f reportklaim.Filter) []any {
	return []any{f.From, f.To}
}

func lineOnly(f reportklaim.Filter) []any {
	return []any{string(f.BusinessLine)}
}

func noArgs(reportklaim.Filter) []any { return nil }

// plans memetakan setiap laporan yang dapat dijalankan ke kueri dan parameternya.
//
// Laporan yang TERHALANG sengaja tidak ada di sini — reportklaim.Lookup menolaknya jauh
// sebelum Repo dipanggil, dan mendaftarkannya berarti menyediakan jalan yang seharusnya
// tertutup.
var plans = map[reportklaim.Code]plan{
	// ---- kelompok Klaim ----
	reportklaim.CodeTAT:         {query: satu("report_tat"), args: dateAndLine, derive: tatDerive},
	reportklaim.CodeKlaimHarian: fixed("report_klaim_harian", dateAndLine),
	reportklaim.CodeRejectKlaim: {query: satu("report_reject_klaim"), args: dateAndLine, derive: staticDerive(rejectDerive)},
	reportklaim.CodeCloseKlaim: byLine(
		branch{"report_close_klaim_nonmbu", closeNonMBUArgs("false")},
		branch{"report_close_klaim", dateAndLine},
		closeDerive),
	reportklaim.CodeTemporaryCloseKlaim: byLine(
		branch{"report_close_klaim_nonmbu", closeNonMBUArgs("true")},
		branch{"report_temporary_close_klaim", dateAndLine},
		closeDerive),
	reportklaim.CodeAIKlaim: fixed("report_ai_klaim", lineOnly),

	// ---- kelompok PLA / DLA ----
	reportklaim.CodePLA:           fixed("report_pla", dateAndLine),
	reportklaim.CodeDLA:           fixed("report_dla", dateAndLine),
	reportklaim.CodePengirimanPLA: fixed("report_pengiriman_pla", dateOnly),
	reportklaim.CodePengirimanDLA: fixed("report_pengiriman_dla", dateOnly),

	// ---- kelompok Akseptasi & Penyelesaian ----
	reportklaim.CodeKasirSudahBayar: fixed("report_kasir_sudah_bayar", dateAndLine),
	reportklaim.CodeKasirBelumBayar: fixed("report_kasir_belum_bayar", dateAndLine),
	reportklaim.CodePendingLOD:      fixed("report_pending_lod", dateAndLine),
	reportklaim.CodeAkseptasi:       fixed("report_akseptasi", dateOnly),
	reportklaim.CodeOSKomite: {query: satu("report_os_komite"), args: dateOnly,
		derive: staticDerive(osKomiteDerive)},
	reportklaim.CodeOSBelumKomite: fixed("report_os_belum_komite", dateOnly),
	reportklaim.CodeKomite: byLine(
		branch{"report_komite_nonmbu", komiteNonMBUArgs},
		branch{"report_komite", dateAndLine},
		komiteDerive),

	// ---- kelompok Lini Bisnis & Mitra Kerja Sama ----
	reportklaim.CodeKlaimHE:             fixed("report_klaim_he", dateOnly),
	reportklaim.CodeRegistSimasOnline:   fixed("report_regist_simas_online", noArgs),
	reportklaim.CodeKlaimAsuransiKredit: fixed("report_klaim_asuransi_kredit", dateOnly),
	reportklaim.CodeKlaimPerBisnis:      fixed("report_klaim_per_bisnis", businessCodeArgs),
	reportklaim.CodeKlaimTraveloka:      fixed("report_klaim_traveloka", dateOnly),
	reportklaim.CodeKlaimPegiPegi:       fixed("report_klaim_pegipegi", dateOnly),

	// ---- kelompok Operasional ----
	reportklaim.CodeProduksiKlaimPA: {query: satu("report_produksi_klaim_pa"), args: dateOnly, derive: staticDerive(derivePeriodeProduksiPA)},
	reportklaim.CodeKomunikasiKlaim: fixed("report_komunikasi_klaim", noArgs),
}

// komiteNonMBUArgs menyusun parameter susunan rinci Data Komite.
//
// `statusapprove` BENAR-BENAR menyaring di sini — tombol Approve dan Rejected
// menghasilkan berkas yang berbeda. Pada susunan ringkas nilainya tidak dipakai sama
// sekali, dan itu keanehan sistem lama yang ditiru apa adanya (lihat report_komite).
func komiteNonMBUArgs(f reportklaim.Filter) []any {
	return []any{f.From, f.To, f.FixedParam["statusapprove"]}
}

// closeNonMBUArgs menyusun parameter susunan rinci Close Klaim.
//
// Parameter ketiganya adalah pembeda dua panel yang berbagi satu kueri: penutupan
// SEMENTARA versus penutupan tetap.
func closeNonMBUArgs(pending string) func(reportklaim.Filter) []any {
	return func(f reportklaim.Filter) []any {
		return []any{f.From, f.To, pending}
	}
}

func businessCodeArgs(f reportklaim.Filter) []any {
	return []any{f.BusinessCode}
}

// holidayCalendar membaca kalender libur untuk satu rentang laporan.
//
// # Ia dibaca lewat KONEKSI KEDUA, bukan lewat DB Link
//
// Kueri aslinya membaca `general.hrd_lbr@ASMD.SINARMAS.CO.ID`
// (`RDB List/CheckHoliday_SQL-SQL.xml`). Keputusan Work Owner 2026-09-24 menetapkan yang
// BUKAN sub-query memakai koneksi langsung, dan ini kueri tersendiri — bukan sub-query.
// Karena itu akhiran DB Link-nya hilang, dan yang berubah hanya koneksi tempat ia
// dijalankan.
//
// # Kegagalan TIDAK dinaikkan sebagai galat
//
// Tiga keadaan — portal tanpa koneksi kedua, koneksi yang tidak dapat dihubungi, dan
// kueri yang gagal — semuanya menghasilkan reportklaim.UnavailableCalendar. Akibatnya
// kolom hari kerja menjadi SEL KOSONG, dan laporannya tetap terbit dengan kolom lain yang
// utuh.
//
// Menggagalkan seluruh laporan karena empat kolom tidak dapat dihitung akan menghilangkan
// 35 kolom yang sudah benar. Mengisinya dengan angka yang dihitung tanpa hari libur jauh
// lebih buruk lagi: angkanya akan lebih besar dari yang sebenarnya, dan tidak ada apa pun
// di berkas yang menandakannya.
func (r *Repo) holidayCalendar(ctx context.Context, from, to time.Time) *reportklaim.HolidayCalendar {
	if r.aneka == nil || from.IsZero() || to.IsZero() {
		return reportklaim.UnavailableCalendar()
	}

	rows, err := r.aneka.QueryContext(ctx, getQuery("report_holiday_calendar"), from, to)
	if err != nil {
		return reportklaim.UnavailableCalendar()
	}
	defer rows.Close()

	var days []time.Time
	for rows.Next() {
		var day sql.NullTime
		if err := rows.Scan(&day); err != nil {
			return reportklaim.UnavailableCalendar()
		}
		if day.Valid {
			days = append(days, day.Time)
		}
	}
	if rows.Err() != nil {
		return reportklaim.UnavailableCalendar()
	}
	return reportklaim.NewHolidayCalendar(days)
}

// Stream menjalankan kueri laporan dan memanggil emit sekali per baris.
func (r *Repo) Stream(
	ctx context.Context,
	report reportklaim.Report,
	filter reportklaim.Filter,
	emit func(reportklaim.Row) error,
) error {
	p, known := plans[report.Code]
	if !known {
		// Ia cacat pemrograman, bukan galat pengguna: katalog menyatakan laporan ini
		// dapat dijalankan, tetapi tidak ada kueri yang melayaninya. Dibiarkan lewat
		// sebagai "hasil kosong", ia akan terbaca sebagai "tidak ada data".
		return fmt.Errorf("reportklaim/sqlstore: laporan %q tidak punya kueri", report.Code)
	}

	name := p.query(filter)
	if !hasQuery(name) {
		// Kueri yang BELUM dipindahkan dari export — bukan cacat rakitan. Penolakannya
		// menyebutkan laporan DAN lini bisnisnya, karena laporan yang sama berjalan
		// normal pada lini lain, dan tanpa keduanya pesan itu menyesatkan.
		return fmt.Errorf("%w: laporan %q lini %q (kueri %q)",
			ErrQueryNotPorted, report.Code, filter.BusinessLine, name)
	}

	// Bahan kolom turunan disiapkan SEBELUM kueri utama dijalankan. Menyiapkannya di
	// tengah pengaliran berarti memegang kursor terbuka selama pembacaan kedua — dan
	// kegagalannya terjadi setelah sebagian baris sudah terkirim.
	var derive func(reportklaim.Row)
	if p.derive != nil {
		built, err := p.derive(ctx, r, filter)
		if err != nil {
			return fmt.Errorf("reportklaim/sqlstore: menyiapkan kolom turunan %q: %w", report.Code, err)
		}
		derive = built
	}

	rows, err := r.db.QueryContext(ctx, getQuery(name), p.args(filter)...)
	if err != nil {
		return fmt.Errorf("reportklaim/sqlstore: menjalankan laporan %q: %w", report.Code, err)
	}
	defer rows.Close()

	names, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("reportklaim/sqlstore: membaca kolom laporan %q: %w", report.Code, err)
	}

	cell := make([]any, len(names))
	into := make([]any, len(names))
	for i := range cell {
		into[i] = &cell[i]
	}

	for rows.Next() {
		if err := rows.Scan(into...); err != nil {
			return fmt.Errorf("reportklaim/sqlstore: memindai baris laporan %q: %w", report.Code, err)
		}
		row := make(reportklaim.Row, len(names))
		for i, name := range names {
			row[name] = text(cell[i])
		}
		if derive != nil {
			derive(row)
		}
		if err := emit(row); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ListBusinessOptions membaca isi autocomplete "Bisnis".
func (r *Repo) ListBusinessOptions(ctx context.Context) ([]reportklaim.BusinessOption, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("report_business_options"))
	if err != nil {
		return nil, fmt.Errorf("reportklaim/sqlstore: membaca pilihan bisnis: %w", err)
	}
	defer rows.Close()

	var out []reportklaim.BusinessOption
	for rows.Next() {
		var code, name sql.NullString
		if err := rows.Scan(&code, &name); err != nil {
			return nil, fmt.Errorf("reportklaim/sqlstore: memindai pilihan bisnis: %w", err)
		}
		out = append(out, reportklaim.BusinessOption{
			Code: strings.TrimSpace(code.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	return out, rows.Err()
}

// text mengubah satu sel hasil kueri menjadi teks yang siap ditulis ke CSV.
//
// # Di sinilah pemformatan tanggal terjadi, dan HANYA di sini
//
// Kueri aslinya memformat tanggal di dalam SQL dengan `to_char(...,'dd/mm/yyyy')` —
// 411 kali di seluruh sistem lama. Itu dilarang (`09-DATABASE-STRATEGY.md` §4) karena
// mengikat SQL pada satu dialek, DAN karena mengubah tanggal menjadi teks membuat
// pengurutannya menjadi pengurutan teks: `01/12/2024` terbaca lebih kecil dari
// `02/01/2020`.
//
// Kolom tanggal karena itu dikembalikan sebagai DATE oleh kueri, lalu diformat di sini —
// satu tempat, dengan zona WIB, menghasilkan teks yang sama persis dengan berkas lama.
func text(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	case []byte:
		return string(value)
	case time.Time:
		// Nilai waktu tidak diubah; yang dipilih hanyalah zona untuk membacanya.
		return value.In(clock.ZoneWIB).Format("02/01/2006")
	case bool:
		if value {
			return "1"
		}
		return "0"
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		// 'f' dengan presisi -1 menulis angka sependek mungkin tanpa kehilangan nilai,
		// dan TIDAK pernah beralih ke notasi ilmiah seperti %v — nilai uang bernilai
		// besar yang keluar sebagai "1.2345e+09" tidak dapat dibaca berkas kerja
		// penggunanya.
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

// feeScale membaca tangga fee adjuster untuk satu kali jalan laporan.
//
// # Ia dibaca dari koneksi PORTAL, bukan koneksi kedua
//
// `POOLDATA.GCNM_FEE_SCALE` adalah master milik basis data portal itu sendiri — berbeda
// dari kalender libur yang hidup di ASMD. Tidak ada DB Link di sini, dan tidak pernah ada.
//
// # Kegagalan TIDAK dinaikkan sebagai galat
//
// Ia menghasilkan reportklaim.UnavailableFeeScale, dan akibatnya satu kolom — "Adjuster
// Fee" — menjadi sel kosong. Menjatuhkan seluruh laporan karena satu master yang tidak
// terbaca berarti 80 kolom lain ikut hilang; sel kosong pada satu kolom jauh lebih jujur
// daripada berkas yang tidak ada.
func (r *Repo) feeScale(ctx context.Context) *reportklaim.FeeScale {
	if r.db == nil {
		return reportklaim.UnavailableFeeScale()
	}
	rows, err := r.db.QueryContext(ctx, getQuery("report_fee_scale"))
	if err != nil {
		return reportklaim.UnavailableFeeScale()
	}
	defer rows.Close()

	var bands []reportklaim.FeeBand
	for rows.Next() {
		var (
			index  sql.NullInt64
			amount sql.NullFloat64
			fee    sql.NullFloat64
		)
		if err := rows.Scan(&index, &amount, &fee); err != nil {
			return reportklaim.UnavailableFeeScale()
		}
		if !index.Valid {
			continue
		}
		bands = append(bands, reportklaim.FeeBand{
			Index:      int(index.Int64),
			LossAmount: amount.Float64,
			Fee:        fee.Float64,
		})
	}
	if rows.Err() != nil {
		return reportklaim.UnavailableFeeScale()
	}
	return reportklaim.NewFeeScale(bands)
}

// progressNames membaca nama tahapan progres untuk satu kali jalan laporan.
//
// Sama seperti feeScale: master milik basis data portal, dan kegagalannya mengosongkan
// satu kolom alih-alih menjatuhkan laporan.
func (r *Repo) progressNames(ctx context.Context) *reportklaim.ProgressNames {
	if r.db == nil {
		return reportklaim.UnavailableProgressNames()
	}
	rows, err := r.db.QueryContext(ctx, getQuery("report_progress_names"))
	if err != nil {
		return reportklaim.UnavailableProgressNames()
	}
	defer rows.Close()

	name := map[string]string{}
	for rows.Next() {
		var id, label sql.NullString
		if err := rows.Scan(&id, &label); err != nil {
			return reportklaim.UnavailableProgressNames()
		}
		if !id.Valid {
			continue
		}
		name[id.String] = label.String
	}
	if rows.Err() != nil {
		return reportklaim.UnavailableProgressNames()
	}
	return reportklaim.NewProgressNames(name)
}

// dominantFactors membaca faktor dominan seluruh klaim dalam rentang laporan.
//
// Kegagalannya mengosongkan satu kolom, sama seperti tangga fee dan nama tahapan
// progres — bukan menjatuhkan laporan.
func (r *Repo) dominantFactors(ctx context.Context, from, to time.Time) *reportklaim.DominantFactors {
	if r.db == nil || from.IsZero() || to.IsZero() {
		return reportklaim.UnavailableDominantFactors()
	}

	rows, err := r.db.QueryContext(ctx, getQuery("report_dominant_factors"), from, to)
	if err != nil {
		return reportklaim.UnavailableDominantFactors()
	}
	defer rows.Close()

	var pairs [][2]string
	for rows.Next() {
		var claim, name sql.NullString
		if err := rows.Scan(&claim, &name); err != nil {
			return reportklaim.UnavailableDominantFactors()
		}
		pairs = append(pairs, [2]string{claim.String, name.String})
	}
	if rows.Err() != nil {
		return reportklaim.UnavailableDominantFactors()
	}
	return reportklaim.NewDominantFactors(pairs)
}
