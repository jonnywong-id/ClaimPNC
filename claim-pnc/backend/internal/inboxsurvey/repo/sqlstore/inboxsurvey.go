package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxsurvey"
)

// Repo membaca antrean survei dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama,
// dan selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengambil satu halaman satu tab.
//
// # Tab yang belum dapat dihitung dicegat DI SINI, sebelum menyentuh basis data
//
// Empat dari tujuh tab bergantung pada kolom yang tidak ada di tabel mana pun yang dibaca
// modul ini (lihat inboxsurvey.UnavailableReason). Untuk keempatnya, kueri dilewati dan
// halaman kosong dikembalikan.
//
// Ini bukan penyamaran: layar TIDAK membaca kosongnya sebagai "tidak ada pekerjaan", karena
// jawaban metadata sudah lebih dulu menyatakan tab itu belum tersedia beserta sebabnya.
// Melewati kuerinya membuat perjalanan yang pasti sia-sia tidak pernah dilakukan.
//
// # Satu perjalanan, bukan dua
//
// Jumlah seluruh baris ikut dibawa kueri yang sama lewat `COUNT(*) OVER ()`. Kueri kedua
// yang hanya menghitung akan membaca ulang gabungan yang sama — dan gabungan itulah bagian
// yang mahal, bukan pemotongan halamannya.
//
// Akibatnya: pada halaman KOSONG, jumlah total tidak ikut terbawa karena tidak ada baris
// yang membawanya. Itu bukan sekadar detail — tanpa penanganannya, bilah halaman menghilang
// begitu pengguna membuka halaman terakhir yang kebetulan kosong. Yang menanganinya adalah
// nilai awal `Total` yang tetap nol, dan layar yang membaca `Total` dari jawaban sebelumnya.
func (r *Repo) List(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
	f inboxsurvey.Filter,
	ready inboxsurvey.Readiness,
) (inboxsurvey.Page, error) {
	clean := f.Normalize()

	result := inboxsurvey.Page{Tasks: []inboxsurvey.SurveyTask{}}

	if !ready.TabAvailable(clean.Tab) {
		return result, nil
	}

	// Parameter BERNAMA, bukan `:1`/`:2`. Lihat banner inboxsurvey.sql — ringkasnya: kueri ini
	// menyebut penanda yang sama berkali-kali (`:tab` 7 kali pada varian penuh), dan go-ora
	// menghitung SETIAP KEMUNCULAN sebagai bind tersendiri bila argumennya tanpa nama. Dengan
	// bind posisional, 12 argumen untuk 35 kemunculan menghasilkan ORA-01008.
	//
	// Dengan `sql.Named`, go-ora menempuh `useNamedParameters()` yang mencocokkan per NAMA dan
	// menandai kemunculan berulang sendiri (`command.go:1822-1847`). Urutan argumen di bawah
	// karena itu tidak lagi menentukan apa pun — dan itu memang yang diinginkan.
	name, args := "list_tasks", []any{
		sql.Named("scope", scopeValue(identity.Scope)),
		sql.Named("tab", string(clean.Tab)),
		sql.Named("login", identity.Login),
		sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
		sql.Named("search", keyword(clean.Search)),
		sql.Named("skip", clean.Offset),
		sql.Named("take", clean.Limit),
	}
	if ready.Complete() {
		name, args = "list_tasks_full", []any{
			sql.Named("scope", scopeValue(identity.Scope)),
			sql.Named("tab", string(clean.Tab)),
			sql.Named("work_done", inboxsurvey.StatusWorkCompleted),
			sql.Named("work_rejected", inboxsurvey.StatusWorkRejected),
			sql.Named("adjuster_confirmed", inboxsurvey.AdjusterConfirmed),
			sql.Named("invoice_fee", inboxsurvey.StatusInvoiceFee),
			sql.Named("login", identity.Login),
			sql.Named("msg_open", inboxsurvey.CommunicationOpen),
			sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
			sql.Named("search", keyword(clean.Search)),
			sql.Named("skip", clean.Offset),
			sql.Named("take", clean.Limit),
		}
	}

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxsurvey.Page{}, fmt.Errorf("menjalankan kueri %s: %w", name, err)
	}
	defer rows.Close()

	for rows.Next() {
		task, total, err := scanTask(rows)
		if err != nil {
			return inboxsurvey.Page{}, fmt.Errorf("membaca baris kueri list_tasks: %w", err)
		}
		result.Tasks = append(result.Tasks, task)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxsurvey.Page{}, fmt.Errorf("menelusuri hasil kueri list_tasks: %w", err)
	}

	return result, nil
}

// Counts menghitung isi tab yang DAPAT dihitung, sekaligus.
//
// # Kenapa tidak ketujuhnya
//
// Empat tab bergantung pada kolom yang tidak ada. Mengembalikan nol untuk keempatnya akan
// menyatakan "tab ini kosong", padahal yang benar adalah "tab ini belum dapat dihitung" —
// dan nol yang keliru tidak pernah dilaporkan siapa pun sebagai kerusakan.
//
// Yang dikembalikan karena itu HANYA tab tersedia, dan pemanggil di lapisan aplikasi yang
// menyatukannya dengan daftar tab lengkap. Urutannya tetap mengikuti inboxsurvey.Tabs():
// layar menggambar bilah tab dari urutan itu, bukan dari urutannya sendiri.
func (r *Repo) Counts(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
	ready inboxsurvey.Readiness,
) ([]inboxsurvey.TabCount, error) {

	// Parameter bernama; alasannya sama dengan List. Di sini kebutuhannya bahkan lebih jelas:
	// `:work_done` muncul 8 kali dan `:scope` sekali, dan bind posisional menuntut 23 nilai
	// untuk 8 hal.
	name, columns, args := "count_tabs", countColumns, []any{
		sql.Named("login", identity.Login),
		sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
		sql.Named("scope", scopeValue(identity.Scope)),
	}
	if ready.Complete() {
		name, columns, args = "count_tabs_full", countColumnsFull, []any{
			sql.Named("work_done", inboxsurvey.StatusWorkCompleted),
			sql.Named("work_rejected", inboxsurvey.StatusWorkRejected),
			sql.Named("adjuster_confirmed", inboxsurvey.AdjusterConfirmed),
			sql.Named("invoice_fee", inboxsurvey.StatusInvoiceFee),
			sql.Named("login", identity.Login),
			sql.Named("msg_open", inboxsurvey.CommunicationOpen),
			sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
			sql.Named("scope", scopeValue(identity.Scope)),
		}
	}

	row := r.db.QueryRowContext(ctx, query(name), args...)

	// Ketiganya dibaca sebagai NullInt64, bukan int.
	//
	// `SUM(...)` atas himpunan KOSONG mengembalikan NULL di Oracle, bukan nol — dan itu
	// persis keadaan seorang surveyor yang belum punya pekerjaan sama sekali. Membaca
	// langsung ke int akan menjatuhkan seluruh bilah tab pada pengguna baru.
	counts := make([]sql.NullInt64, len(columns))
	targets := make([]any, len(columns))
	for i := range counts {
		targets[i] = &counts[i]
	}

	if err := row.Scan(targets...); err != nil {
		return nil, fmt.Errorf("membaca hasil kueri %s: %w", name, err)
	}

	result := make([]inboxsurvey.TabCount, 0, len(columns))
	next := 0
	for _, tab := range inboxsurvey.Tabs() {
		if !ready.TabAvailable(tab) {
			continue
		}
		if next >= len(counts) {
			// Jumlah kolom kueri lebih sedikit daripada jumlah tab tersedia. Itu berarti
			// kueri dan domain sudah tidak sejalan, dan diam-diam memotong daftarnya akan
			// menampilkan angka milik tab lain pada tab ini.
			return nil, fmt.Errorf(
				"kueri %s mengembalikan %d kolom, sementara ada lebih banyak tab tersedia",
				name, len(counts),
			)
		}
		result = append(result, inboxsurvey.TabCount{
			Tab:   tab,
			Total: int(counts[next].Int64),
		})
		next++
	}
	return result, nil
}

// KPI mengambil ringkasan KPI adjuster.
//
// # Isian layar menentukan kueri, bukan sebaliknya
//
// Panel KPI Pega punya empat kendali — Status Survey, Tipe Report, Kuartal, Tahun Kuartal —
// dan `Activity/GetReportKPIAdjuster-Act.xml` menurunkan laporan mana yang dijalankan dari
// kombinasinya. Dua yang menentukan di sini:
//
//	Status Survey  -> nilai penyaring kolom `tipe`; "ALL" berarti TANPA penyaring
//	Kuartal/Tahun  -> begitu salah satunya diisi, pengelompokan berpindah ke per TAHUN
//
// Tipe Report sudah dicegat lebih dulu di KPIFilter.Check — "DATA DETAIL" ditolak karena rule
// penyusunnya tidak ada di export (`R-16`), dan menjawabnya dengan tabel kosong akan terbaca
// sebagai "tidak ada datanya".
func (r *Repo) KPI(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
	f inboxsurvey.KPIFilter,
) ([]inboxsurvey.KPIRow, error) {
	clean := f.Normalize()

	// Kuerinya dipilih BENTUK hasil, dan bentuk itu diturunkan domain dari kombinasi isian
	// (lihat inboxsurvey.KPIFilter.Shape). Repo tidak memutuskannya sendiri: keputusan itu
	// meniru percabangan `GetReportKPIAdjuster`, dan percabangan bisnis hidup di domain.
	//
	// Setiap bentuk membawa parameternya sendiri. Mengirim parameter yang tidak disebut
	// kuerinya akan ditolak go-ora dengan "parameter X is not defined in parameter list".
	scope := sql.Named("scope", scopeValue(identity.Scope))

	var (
		name string
		args []any
	)
	switch clean.Shape() {
	case inboxsurvey.ShapePerAdjusterStatus:
		name, args = "kpi_by_adjuster_all", []any{scope}

	case inboxsurvey.ShapePerYear:
		name, args = "kpi_by_year", []any{
			scope,
			sql.Named("year", keyword(clean.Year)),
			sql.Named("quarter", keyword(clean.Quarter)),
		}

	case inboxsurvey.ShapePerQuarterYear:
		name, args = "kpi_by_quarter_year", []any{
			scope,
			sql.Named("year", keyword(clean.Year)),
		}

	case inboxsurvey.ShapeDetail:
		name, args = "kpi_detail", []any{
			scope,
			sql.Named("year", keyword(clean.Year)),
			sql.Named("quarter", keyword(clean.Quarter)),
		}

	default:
		name, args = "kpi_by_adjuster", []any{
			scope,
			sql.Named("kpi_type", clean.CategoryValue()),
		}
	}

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri %s: %w", name, err)
	}
	defer rows.Close()

	result := []inboxsurvey.KPIRow{}
	for rows.Next() {
		entry, err := scanKPI(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris kueri %s: %w", name, err)
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri %s: %w", name, err)
	}

	return result, nil
}

// KPIYears mengambil tahun yang benar-benar ada pada baris milik cakupan pemanggil.
//
// Tahun KOSONG dilewati, bukan dikembalikan sebagai pilihan kosong: dropdown yang memuat
// baris tanpa label tidak dapat dipilih dengan sengaja oleh siapa pun.
func (r *Repo) KPIYears(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query("kpi_years"),
		sql.Named("scope", scopeValue(identity.Scope)))
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri kpi_years: %w", err)
	}
	defer rows.Close()

	result := []string{}
	for rows.Next() {
		var year sql.NullString
		if err := rows.Scan(&year); err != nil {
			return nil, fmt.Errorf("membaca baris kueri kpi_years: %w", err)
		}
		if trimmed := strings.TrimSpace(year.String); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri kpi_years: %w", err)
	}

	return result, nil
}

// scopeValue menyusun cakupan nama surveyor menjadi satu teks berpembatas.
//
// # Kenapa satu teks, bukan klausa IN
//
// Cakupan seorang leader berpanjang berubah, dan `IN (:1, :2, …)` menuntut jumlah bind yang
// tetap. Merangkai namanya ke dalam teks SQL adalah persis celah `{ASIS:...}` yang sedang
// dihapus (§4.5), sehingga yang dipakai adalah satu nilai ber-bind yang diperiksa `INSTR`.
//
// Pembatas `|` dipasang di KEDUA sisi tiap nama — termasuk di ujung — supaya "BUDI" tidak
// cocok dengan "BUDIONO". Tanpa itu seorang surveyor akan melihat pekerjaan surveyor lain
// yang namanya kebetulan memuat namanya.
//
// Nama dinaikkan menjadi huruf besar di sini dan di dalam kueri sekaligus, supaya perbedaan
// kapitalisasi antara `MST_LOGIN_SURVEYOR.NAMA` dan `T_SURVEYORLIST.SURVEYOR_NAME` tidak
// membuat antrean tampak kosong (lihat CATATAN 2 pada berkas .sql).
//
// Cakupan KOSONG menghasilkan `"|"`, dan itu tidak cocok dengan apa pun — jawaban yang benar
// untuk pemanggil tanpa cakupan. Ia TIDAK boleh menghasilkan teks kosong: mencari di dalam
// teks kosong juga tidak cocok dengan apa pun, tetapi mengandalkan itu membuat perilakunya
// bergantung pada kebetulan alih-alih pada bentuk nilainya.
func scopeValue(names []string) string {
	var builder strings.Builder
	builder.WriteByte('|')

	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		builder.WriteString(strings.ToUpper(trimmed))
		builder.WriteByte('|')
	}

	return builder.String()
}

// keyword menyiapkan nilai opsional sebagai nilai yang boleh NULL.
//
// Nilai kosong dikirim sebagai NULL, dan kueri menjawabnya dengan `:n IS NULL` yang mematikan
// saringannya. Mengirimnya sebagai teks kosong akan membuat `LIKE '%%'` — yang kebetulan juga
// cocok dengan semuanya, tetapi memaksa basis data memindai setiap baris alih-alih melewati
// predikatnya.
func keyword(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// scanTask membaca satu baris hasil beserta jumlah seluruh baris yang menyertainya.
//
// Urutan pembacaan di sini WAJIB sama dengan urutan kolom pada kueri dan dengan taskColumns
// di query.go. Ketiganya dijaga query_test.go.
func scanTask(rows *sql.Rows) (inboxsurvey.SurveyTask, int, error) {
	var (
		task inboxsurvey.SurveyTask

		surveyID        sql.NullString
		claimID         sql.NullString
		surveyIndex     sql.NullString
		referenceNumber sql.NullString
		claimNumber     sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		classOfBusiness sql.NullString
		causeOfLoss     sql.NullString
		location        sql.NullString
		technicalPIC    sql.NullString
		adjusterPIC     sql.NullString
		dateOfLoss      sql.NullTime
		createdAt       sql.NullTime
		asmStatus       sql.NullString
		surveyorType    sql.NullString
		total           int
	)

	// SELURUH kolom teks dibaca sebagai NullString, termasuk yang "pasti terisi".
	//
	// Itu bukan kehati-hatian berlebihan. Kolom `T_CLAIM_PNC` tidak punya constraint NOT NULL
	// yang terbaca dari mana pun — DDL-nya belum ada (`R-08`). `T_SURVEYORLIST` lebih longgar
	// lagi: `INSERT_SURVEYORLIST.prc` menyisipkan tujuh belas kolom tanpa satu pun pemeriksaan.
	// Membaca langsung ke string akan menghasilkan galat pemindaian pada satu baris warisan,
	// dan galat itu menjatuhkan SELURUH halaman — bukan satu sel.
	//
	// AppointmentNumber TIDAK dipindai di sini, dan itu BUKAN karena ia belum tersedia: ia
	// **diturunkan** dari `SURVEY_ID` oleh `inboxsurvey.SurveyTask.AppointmentNo`, persis
	// seperti Pega memotongnya dari kunci objek kerja. Memindainya sebagai kolom tersendiri
	// akan membuat dua sumber untuk satu nilai, dan keduanya bisa berbeda.
	//
	// ReferenceNumber DIPINDAI dari kedua varian kueri, dan itulah sebabnya varian terbatas
	// tetap mengembalikan kolomnya sebagai `CAST(NULL AS VARCHAR2(101))`. Bentuk kedua varian
	// WAJIB sama persis supaya keduanya dapat dibaca pemindai yang satu ini — kalau tidak,
	// satu kolom bergeser dan nomor polis masuk ke kolom nama tertanggung tanpa galat apa pun.
	if err := rows.Scan(
		&surveyID,
		&claimID,
		&surveyIndex,
		&referenceNumber,
		&claimNumber,
		&policyNumber,
		&insuredName,
		&classOfBusiness,
		&causeOfLoss,
		&location,
		&technicalPIC,
		&adjusterPIC,
		&dateOfLoss,
		&createdAt,
		&asmStatus,
		&surveyorType,
		&total,
	); err != nil {
		return inboxsurvey.SurveyTask{}, 0, err
	}

	task.SurveyID = surveyID.String
	task.ClaimID = claimID.String
	task.SurveyIndex = surveyIndex.String
	task.ReferenceNumber = referenceNumber.String
	task.ClaimNumber = claimNumber.String
	task.PolicyNumber = policyNumber.String
	task.InsuredName = insuredName.String
	task.ClassOfBusiness = classOfBusiness.String
	task.CauseOfLoss = causeOfLoss.String
	task.Location = location.String
	task.TechnicalPIC = technicalPIC.String
	task.AdjusterPIC = adjusterPIC.String
	task.ASMStatus = asmStatus.String
	task.SurveyorType = surveyorType.String

	if dateOfLoss.Valid {
		task.DateOfLoss = dateOfLoss.Time
	}

	// CreatedAt NULL dibiarkan sebagai waktu kosong, dan itu BUKAN kelalaian.
	//
	// Ia dasar perhitungan Aging, dan SurveyTask.AgingDays menjawab waktu kosong dengan
	// penunjuk kosong — bukan dengan nol hari. Nol hari dan "tidak dapat dihitung" adalah dua
	// keadaan berbeda; menyamakannya menampilkan angka yang terlihat sah dan salah.
	if createdAt.Valid {
		task.CreatedAt = createdAt.Time
	}

	return task, total, nil
}

// scanKPI membaca satu baris ringkasan KPI.
//
// Kesembilan angkanya dibaca sebagai NullFloat64: `AVG(...)` atas himpunan kosong
// mengembalikan NULL, dan kolom yang tidak pernah diisi menghasilkan hal yang sama. Nilai
// NULL menjadi nol di sini — pada angka RATA-RATA, nol adalah bacaan yang benar untuk "tidak
// ada nilai", berbeda dari kolom Aging yang menyatakan jumlah hari.
func scanKPI(rows *sql.Rows) (inboxsurvey.KPIRow, error) {
	// LIMA kolom kunci di depan, lalu sembilan angka. Kelima kueri KPI berbentuk sama
	// persis — kolom yang tidak berlaku diisi NULL di SQL — supaya seluruhnya dibaca
	// fungsi yang SATU ini. Satu kolom yang bergeser di salah satu kueri akan memindahkan
	// angka ke kolom tetangganya, dan dua angka penilaian yang tertukar sama-sama masuk akal.
	var (
		groupKey   sql.NullString
		statusKey  sql.NullString
		quarterKey sql.NullString
		monthKey   sql.NullString
		caseKey    sql.NullString
		numbers    = make([]sql.NullFloat64, len(kpiColumns)-len(kpiKeyColumns))
	)

	targets := make([]any, 0, len(kpiColumns))
	targets = append(targets, &groupKey, &statusKey, &quarterKey, &monthKey, &caseKey)
	for i := range numbers {
		targets = append(targets, &numbers[i])
	}

	if err := rows.Scan(targets...); err != nil {
		return inboxsurvey.KPIRow{}, err
	}

	return inboxsurvey.KPIRow{
		Group:                 groupKey.String,
		Status:                statusKey.String,
		Quarter:               quarterKey.String,
		Month:                 monthKey.String,
		CaseID:                caseKey.String,
		SurveyScheduling:      numbers[0].Float64,
		ImmediateAdvice:       numbers[1].Float64,
		PreliminaryAdvice:     numbers[2].Float64,
		InterimReport:         numbers[3].Float64,
		ProgressUpdate:        numbers[4].Float64,
		CommunicationResponse: numbers[5].Float64,
		ProposeAdjustment:     numbers[6].Float64,
		FinalReport:           numbers[7].Float64,
		Value:                 numbers[8].Float64,
	}, nil
}
