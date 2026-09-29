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
) (inboxsurvey.Page, error) {
	clean := f.Normalize()

	result := inboxsurvey.Page{Tasks: []inboxsurvey.SurveyTask{}}

	rows, err := r.db.QueryContext(ctx, query("list_tasks"),
		scopeValue(identity.Scope),
		string(clean.Tab),
		identity.Login,
		inboxsurvey.AdjusterConfirmed,
		inboxsurvey.StatusInvoiceFee,
		inboxsurvey.StatusCloseCase,
		inboxsurvey.CommunicationOpen,
		inboxsurvey.CommunicationAnswered,
		keyword(clean.Search),
		clean.Offset,
		clean.Limit,
	)
	if err != nil {
		return inboxsurvey.Page{}, fmt.Errorf("menjalankan kueri list_tasks: %w", err)
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

// Counts menghitung isi ketujuh tab sekaligus.
//
// Urutan yang dikembalikan mengikuti inboxsurvey.Tabs(), dan itu bukan kebetulan: layar
// menggambar bilah tab dari urutan ini, bukan dari urutannya sendiri.
func (r *Repo) Counts(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
) ([]inboxsurvey.TabCount, error) {
	row := r.db.QueryRowContext(ctx, query("count_tabs"),
		scopeValue(identity.Scope),
		identity.Login,
		inboxsurvey.AdjusterConfirmed,
		inboxsurvey.StatusInvoiceFee,
		inboxsurvey.StatusCloseCase,
		inboxsurvey.CommunicationOpen,
		inboxsurvey.CommunicationAnswered,
	)

	// SELURUH tujuh dibaca sebagai NullInt64, bukan int.
	//
	// `SUM(...)` atas himpunan KOSONG mengembalikan NULL di Oracle, bukan nol — dan itu
	// persis keadaan seorang surveyor yang belum punya pekerjaan sama sekali. Membaca
	// langsung ke int akan menjatuhkan seluruh bilah tab pada pengguna baru.
	counts := make([]sql.NullInt64, len(countColumns))
	targets := make([]any, len(countColumns))
	for i := range counts {
		targets[i] = &counts[i]
	}

	if err := row.Scan(targets...); err != nil {
		return nil, fmt.Errorf("membaca hasil kueri count_tabs: %w", err)
	}

	tabs := inboxsurvey.Tabs()
	result := make([]inboxsurvey.TabCount, 0, len(tabs))
	for i, tab := range tabs {
		result = append(result, inboxsurvey.TabCount{
			Tab:   tab,
			Total: int(counts[i].Int64),
		})
	}
	return result, nil
}

// KPI mengambil ringkasan KPI adjuster.
//
// Jenis ringkasan menentukan kueri mana yang dijalankan DAN nilai `tipe` yang dikirim:
// KPIFinal dan KPIQuarterly keduanya mematok `'FINAL'` di sistem lama, sedangkan
// KPIOutstanding menerimanya dari pemanggil.
func (r *Repo) KPI(
	ctx context.Context,
	identity inboxsurvey.SurveyorIdentity,
	f inboxsurvey.KPIFilter,
) ([]inboxsurvey.KPIRow, error) {
	clean := f.Normalize()

	name := "kpi_by_adjuster"
	if clean.Kind == inboxsurvey.KPIQuarterly {
		name = "kpi_by_year"
	}

	rows, err := r.db.QueryContext(ctx, query(name),
		scopeValue(identity.Scope),
		keyword(kpiCategory(clean)),
		keyword(clean.Year),
	)
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

// kpiCategory memutuskan nilai kolom `tipe` yang dikirim ke kueri.
//
// KPIFinal dan KPIQuarterly keduanya mematok `'FINAL'` — itu tertulis di dalam rule-nya
// sendiri (`where tipe='FINAL'`), bukan diserahkan pemanggil. Menerima kategori dari layar
// untuk kedua jenis itu akan membuat layar dapat menampilkan angka yang di Pega tidak pernah
// dapat ditampilkan.
func kpiCategory(f inboxsurvey.KPIFilter) string {
	if f.Kind == inboxsurvey.KPIOutstanding {
		return f.Category
	}
	return inboxsurvey.KPITypeFinal
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

		surveyID          sql.NullString
		claimID           sql.NullString
		surveyIndex       sql.NullString
		appointmentNumber sql.NullString
		referenceNumber   sql.NullString
		claimNumber       sql.NullString
		policyNumber      sql.NullString
		insuredName       sql.NullString
		classOfBusiness   sql.NullString
		causeOfLoss       sql.NullString
		location          sql.NullString
		technicalPIC      sql.NullString
		adjusterPIC       sql.NullString
		dateOfLoss        sql.NullTime
		agingDays         sql.NullInt64
		asmStatus         sql.NullString
		surveyorType      sql.NullString
		surveyStatus      sql.NullString
		total             int
	)

	// SELURUH kolom teks dibaca sebagai NullString, termasuk yang "pasti terisi".
	//
	// Itu bukan kehati-hatian berlebihan. `docs/kolom-t-claimlist-admin.md` mencatat 63 dari
	// 186 kolom tabel itu TIDAK PERNAH diisi, dan kolom yang terisi pun tidak punya
	// constraint NOT NULL. `T_SURVEYORLIST` lebih longgar lagi: `INSERT_SURVEYORLIST.prc`
	// menyisipkan tujuh belas kolom tanpa satu pun pemeriksaan. Membaca langsung ke string
	// akan menghasilkan galat pemindaian pada satu baris warisan, dan galat itu menjatuhkan
	// SELURUH halaman — bukan satu sel.
	if err := rows.Scan(
		&surveyID,
		&claimID,
		&surveyIndex,
		&appointmentNumber,
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
		&agingDays,
		&asmStatus,
		&surveyorType,
		&surveyStatus,
		&total,
	); err != nil {
		return inboxsurvey.SurveyTask{}, 0, err
	}

	task.SurveyID = surveyID.String
	task.ClaimID = claimID.String
	task.SurveyIndex = surveyIndex.String
	task.AppointmentNumber = appointmentNumber.String
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
	task.SurveyStatus = surveyStatus.String

	if dateOfLoss.Valid {
		task.DateOfLoss = dateOfLoss.Time
	}

	// Aging NULL dibiarkan sebagai penunjuk kosong, TIDAK dijadikan nol.
	//
	// Nol hari dan "belum dihitung" adalah dua keadaan berbeda, dan menyamakannya akan
	// menampilkan angka "0" pada baris yang sebenarnya tidak punya angka.
	if agingDays.Valid {
		days := int(agingDays.Int64)
		task.AgingDays = &days
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
	var (
		groupKey sql.NullString
		numbers  = make([]sql.NullFloat64, len(kpiColumns)-1)
	)

	targets := make([]any, 0, len(kpiColumns))
	targets = append(targets, &groupKey)
	for i := range numbers {
		targets = append(targets, &numbers[i])
	}

	if err := rows.Scan(targets...); err != nil {
		return inboxsurvey.KPIRow{}, err
	}

	return inboxsurvey.KPIRow{
		Group:                 groupKey.String,
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
