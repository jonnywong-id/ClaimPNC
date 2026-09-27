package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxoutstanding"
)

// Repo membaca POOLDATA.T_CLAIMLIST_ADMIN.
//
// Tabel itu MILIK SISTEM LAMA dan diisi olehnya. Repo ini hanya membaca, dan tidak punya
// satu pun method tulis — `P-1` menetapkan satu tabel ditulis satu sistem.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca satu halaman pekerjaan milik pemanggil beserta jumlah seluruh yang cocok.
//
// Pemanggil WAJIB terisi. Tanpa itu kueri tidak dijalankan sama sekali — bukan dijalankan
// tanpa penyaring. Menjalankannya tanpa penyaring akan menampilkan pekerjaan SELURUH
// operator di layar yang bernama "My Inbox", dan kegagalan itu tidak menghasilkan galat
// apa pun: layarnya terisi, hanya isinya bukan milik yang melihat.
func (r *Repo) List(ctx context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	f = f.Normalize()

	if f.AssignedTo == "" {
		return inboxoutstanding.Page{}, inboxoutstanding.ErrAssigneeRequired
	}

	// Penyaring status dokumen ikut pada DAFTAR dan HITUNGANNYA, supaya total paginasi
	// cocok dengan isi grid setelah pengguna mengeklik satu irisan donut.
	filters := append(filterArgs(f), statusArgs(f)...)

	var total int
	if err := r.db.QueryRowContext(ctx, query("my_inbox_count"), filters...).Scan(&total); err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: menghitung klaim: %w", err)
	}

	// Paginasi memakai :18 dan :19 — dua parameter terakhir, disisipkan sesudah penyaring.
	listArgs := append(append([]any(nil), filters...), f.Offset, f.Limit)

	rows, err := r.db.QueryContext(ctx, query("my_inbox_list"), listArgs...)
	if err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: membaca daftar klaim: %w", err)
	}
	defer func() { _ = rows.Close() }()

	claims := make([]inboxoutstanding.OutstandingClaim, 0, f.Limit)
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: membaca baris klaim: %w", err)
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: menelusuri daftar klaim: %w", err)
	}

	return inboxoutstanding.Page{Claims: claims, Total: total}, nil
}

// Export membaca sekumpulan baris untuk unduhan CSV.
//
// # Ia TIDAK menyaring pemilik pekerjaan, dan itu disengaja
//
// `RDB List/ExportDataDetailKlaim-SQL.xml` tidak punya satu pun penyaring operator.
// Export sebelumnya memanggil ulang List, sehingga ikut terkena `PXASSIGNEDOPERATORID`
// dan mengembalikan berkas kosong bagi petugas yang inbox-nya kosong — padahal berkas itu
// mestinya tetap berisi seluruh klaim dalam cakupan lini bisnisnya.
//
// Yang membatasi di sini adalah cakupan lini bisnis, bukan identitas.
func (r *Repo) Export(ctx context.Context, f inboxoutstanding.ExportFilter) (inboxoutstanding.Page, error) {
	f = f.Normalize()
	args := exportArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("my_inbox_export_count"), args...).Scan(&total); err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: menghitung baris export: %w", err)
	}

	// Paginasi memakai :10 dan :11 — dua parameter terakhir, disisipkan sesudah penyaring.
	pageArgs := append(append([]any(nil), args...), f.Offset, f.Limit)

	rows, err := r.db.QueryContext(ctx, query("my_inbox_export"), pageArgs...)
	if err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: membaca baris export: %w", err)
	}
	defer func() { _ = rows.Close() }()

	claims := make([]inboxoutstanding.OutstandingClaim, 0, f.Limit)
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: membaca satu baris export: %w", err)
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: menelusuri baris export: %w", err)
	}

	return inboxoutstanding.Page{Claims: claims, Total: total}, nil
}

// LineBusinessFor membaca lini bisnis petugas dari M_LOGIN_PNC.
//
// Petugas tanpa baris mengembalikan LineUnknown TANPA galat. Kolomnya baru terisi pada
// sebagian petugas, dan menjadikan ketiadaannya galat akan membuat export gagal bagi
// mereka — padahal sistem lama justru melayaninya tanpa cakupan.
func (r *Repo) LineBusinessFor(ctx context.Context, loginID string) (inboxoutstanding.LineBusiness, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return inboxoutstanding.LineUnknown, nil
	}

	var line sql.NullString
	err := r.db.QueryRowContext(ctx, query("line_business_for"), id).Scan(&line)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return inboxoutstanding.LineUnknown, nil
	case err != nil:
		return inboxoutstanding.LineUnknown, fmt.Errorf("inboxoutstanding/sqlstore: membaca lini bisnis petugas: %w", err)
	}
	return inboxoutstanding.NormalizeLineBusiness(line.String), nil
}

// exportArgs menyusun sembilan argumen pertama, sama untuk kedua kueri export.
//
// Lini bisnis dikirim LIMA KALI dan tiap tanggal DUA KALI, dengan alasan yang sama seperti
// pada filterArgs: driver mengikat menurut urutan kemunculan penanda, bukan nomornya.
func exportArgs(f inboxoutstanding.ExportFilter) []any {
	line := string(f.LineBusiness)

	var from, to any
	if f.From != nil {
		from = f.From.UTC()
	}
	if f.To != nil {
		to = f.To.UTC()
	}

	return []any{
		line, // :1 PA
		line, // :2 TRAVEL
		line, // :3 BONDING
		line, // :4 NONMBU
		line, // :5 cabang "tanpa cakupan"
		from, // :6 IS NULL
		from, // :7 perbandingan
		to,   // :8 IS NULL
		to,   // :9 perbandingan
	}
}

// filterArgs menyusun tiga belas argumen pertama, sama untuk kedua kueri.
//
// Pola "NULL berarti tidak menyaring" dipakai supaya satu teks SQL melayani seluruh
// kombinasi penyaring. Menyusun WHERE-nya di Go akan mengembalikan SQL ke dalam kode —
// persis yang aturan §4.3 larang.
func filterArgs(f inboxoutstanding.Filter) []any {
	search := nilIfEmpty(f.Search)
	pattern := nilIfEmpty(searchPattern(f.Search))

	// Empat penyaring opsional dikirim DUA KALI, dan itu bukan kelalaian.
	//
	// Masing-masing muncul dua kali di dalam SQL — sekali pada `IS NULL`, sekali pada
	// perbandingannya — dan driver mengikat argumen menurut urutan KEMUNCULAN penanda,
	// bukan menurut nomornya. Mengirim sekali menghasilkan ORA-01008. Lihat kepala
	// outstanding.sql.
	panel := nilIfEmpty(strings.ToUpper(f.GroupPanel))
	rcv := nilIfEmpty(strings.ToUpper(f.RCVID))
	stage := nilIfEmpty(strings.ToUpper(f.Stage))
	branch := nilIfEmpty(strings.ToUpper(f.BranchCode))

	// Identitas lama diulangi dengan identitas sekarang bila kosong.
	//
	// `IN (:1, :2)` dengan :2 bernilai NULL tidak salah di Oracle, tetapi mengirim nilai
	// yang sama dua kali membuat maksudnya terbaca tanpa perlu menalar perilaku NULL pada
	// klausa IN — dan hasilnya identik.
	primary := strings.ToUpper(f.AssignedTo)
	legacy := strings.ToUpper(f.AssignedToLegacy)
	if legacy == "" {
		legacy = primary
	}

	return []any{
		primary, // :1 WAJIB — identitas login
		legacy,  // :2 identitas lama orang yang sama
		search,  // :3 penentu apakah pencarian aktif
		pattern, // :4 PYID
		pattern, // :5 POLICYNO
		pattern, // :6 USERTEKNIS_1
		panel,   // :7 IS NULL
		panel,   // :8 perbandingan
		rcv,     // :9 IS NULL
		rcv,     // :10 perbandingan
		stage,   // :11 IS NULL
		stage,   // :12 perbandingan
		branch,  // :13 IS NULL
		branch,  // :14 perbandingan
	}
}

// statusArgs menambahkan tiga argumen penyaring status dokumen.
//
// Terpisah dari filterArgs karena ringkasan memakai KESEMBILAN BELAS yang pertama saja
// sampai :14 — ia sengaja tidak menyaring status, supaya seluruh irisan tetap terlihat.
func statusArgs(f inboxoutstanding.Filter) []any {
	status := nilIfEmpty(string(f.DocumentStatus))
	return []any{
		status, // :15 IS NULL
		status, // :16 uji LENGKAP
		status, // :17 uji BELUM
	}
}

// SummarizeDocumentStatus menghitung isi inbox pemanggil per status kelengkapan dokumen.
//
// Pemilik WAJIB terisi, dengan alasan yang sama seperti pada List: ringkasan tanpa pemilik
// adalah ringkasan pekerjaan SELURUH operator, dan angkanya akan tampak wajar.
func (r *Repo) SummarizeDocumentStatus(ctx context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Summary, error) {
	f = f.Normalize()
	if f.AssignedTo == "" {
		return inboxoutstanding.Summary{}, inboxoutstanding.ErrAssigneeRequired
	}

	var lengkap, belum, total sql.NullInt64
	err := r.db.QueryRowContext(ctx, query("my_inbox_document_status"), filterArgs(f)...).
		Scan(&lengkap, &belum, &total)
	if err != nil {
		return inboxoutstanding.Summary{}, fmt.Errorf("inboxoutstanding/sqlstore: meringkas status dokumen: %w", err)
	}

	// SUM atas himpunan kosong mengembalikan NULL, bukan nol — sehingga inbox yang kosong
	// harus menghasilkan angka nol di sini, bukan galat konversi.
	return inboxoutstanding.BuildSummary(map[inboxoutstanding.DocumentStatus]int{
		inboxoutstanding.StatusComplete:   int(lengkap.Int64),
		inboxoutstanding.StatusIncomplete: int(belum.Int64),
		inboxoutstanding.StatusAll:        int(total.Int64),
	}, int(total.Int64)), nil
}

// LegacyOperatorFor membaca identitas lama petugas dari T_ACCESS_GROUP_PNC.
//
// Ketiadaan baris BUKAN galat: ia berarti identitas orang itu tidak pernah berganti.
func (r *Repo) LegacyOperatorFor(ctx context.Context, loginID string) (string, error) {
	id := strings.ToUpper(strings.TrimSpace(loginID))
	if id == "" {
		return "", nil
	}

	var legacy sql.NullString
	err := r.db.QueryRowContext(ctx, query("legacy_operator_for"), id).Scan(&legacy)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("inboxoutstanding/sqlstore: membaca identitas lama petugas: %w", err)
	}

	// MAX atas himpunan kosong mengembalikan satu baris bernilai NULL, bukan nol baris —
	// sehingga ErrNoRows di atas jarang terjadi dan cabang inilah yang biasa dilalui.
	old := strings.ToUpper(strings.TrimSpace(legacy.String))
	if old == id {
		// Identitasnya tidak berganti; tidak ada yang perlu ditambahkan.
		return "", nil
	}
	return old, nil
}

// searchPattern membentuk pola LIKE.
//
// Diseragamkan menjadi huruf besar karena sisi SQL memakai UPPER(...). Perbandingan yang
// hanya satu sisinya diseragamkan tidak pernah cocok, dan gagalnya diam — pengguna
// mengetik dengan huruf kecil lalu diberi tahu bahwa klaimnya tidak ada.
//
// Karakter khusus LIKE di-escape supaya pencarian "100%" tidak berubah menjadi pola yang
// mencocokkan apa saja. ESCAPE-nya dinyatakan di sisi SQL.
func searchPattern(search string) string {
	trimmed := strings.TrimSpace(search)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// nilIfEmpty mengubah string kosong menjadi NULL, supaya penyaring "NULL berarti semua"
// pada SQL bekerja.
func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// scanClaim membaca satu baris.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: kolomnya nullable di skema, dan
// membacanya langsung ke string akan gagal dengan galat konversi pada baris pertama yang
// kosong — misalnya klaim yang belum bernomor, yang justru keadaan biasa.
func scanClaim(rows *sql.Rows) (inboxoutstanding.OutstandingClaim, error) {
	var (
		claimID        string
		number         sql.NullString
		policyNumber   sql.NullString
		insuredName    sql.NullString
		businessName   sql.NullString
		businessSource sql.NullString
		branchName     sql.NullString
		groupPanel     sql.NullString
		rcvID          sql.NullString
		registerDate   sql.NullTime
		createdAt      sql.NullTime
		lossDate       sql.NullTime
		reportDate     sql.NullTime
		aging          sql.NullInt64
		processStatus  sql.NullString
		claimStatus    sql.NullString
		progressStatus sql.NullString
		technicalPIC   sql.NullString
		recordedBy     sql.NullString
		currentStage   sql.NullString
		currentHolder  sql.NullString
		documentDone   sql.NullString
	)

	// Urutannya WAJIB sama persis dengan daftar kolom pada my_inbox_list. Menambah
	// kolom di SQL tanpa menambahnya di sini menghasilkan galat jumlah kolom; menukar
	// urutannya menghasilkan data yang tertukar TANPA galat.
	if err := rows.Scan(
		&claimID, &number, &policyNumber, &insuredName, &businessName, &businessSource,
		&branchName, &groupPanel, &rcvID,
		&registerDate, &createdAt, &lossDate, &reportDate, &aging,
		&processStatus, &claimStatus, &progressStatus,
		&technicalPIC, &recordedBy, &currentStage, &currentHolder, &documentDone,
	); err != nil {
		return inboxoutstanding.OutstandingClaim{}, err
	}

	claim := inboxoutstanding.OutstandingClaim{
		ClaimID:        claimID,
		ClaimNumber:    strings.TrimSpace(number.String),
		PolicyNumber:   strings.TrimSpace(policyNumber.String),
		InsuredName:    insuredName.String,
		BusinessName:   businessName.String,
		BusinessSource: businessSource.String,
		BranchName:     branchName.String,
		GroupPanel:     strings.TrimSpace(groupPanel.String),
		RCVID:          strings.TrimSpace(rcvID.String),
		ProcessStatus:  strings.TrimSpace(processStatus.String),
		ClaimStatus:    strings.TrimSpace(claimStatus.String),
		ProgressStatus: strings.TrimSpace(progressStatus.String),
		TechnicalPIC:   strings.TrimSpace(technicalPIC.String),
		RecordedBy:     strings.TrimSpace(recordedBy.String),
		CurrentStage:   strings.TrimSpace(currentStage.String),
		CurrentHolder:  strings.TrimSpace(currentHolder.String),

		// HANYA '1' yang berarti lengkap. '0' dan NULL sama-sama "belum" — aturan Pega,
		// bukan penyederhanaan (`SetClaimPNC-Act.xml:1630`, `:1799`).
		DocumentComplete: strings.TrimSpace(documentDone.String) == "1",
	}

	// REGISTERDATE_1 lebih dulu, PXCREATEDATETIME sebagai cadangan.
	//
	// Keduanya ada di tabel dan mudah tertukar: yang pertama tanggal registrasi klaim,
	// yang kedua waktu barisnya dibuat di Pega. Yang dipakai kolom "Register Date" adalah
	// yang pertama; cadangannya ada supaya baris lama yang kolomnya belum terisi tidak
	// tampil tanpa tanggal sama sekali.
	switch {
	case registerDate.Valid:
		claim.RegisteredAt = registerDate.Time.UTC()
	case createdAt.Valid:
		claim.RegisteredAt = createdAt.Time.UTC()
	}

	// Salinan lokal pada tiap nilai bertipe pointer: mengambil alamat field struct
	// sql.NullTime akan membuat seluruh baris berbagi pointer yang sama saat di-loop.
	if lossDate.Valid {
		date := lossDate.Time.UTC()
		claim.LossDate = &date
	}
	if reportDate.Valid {
		date := reportDate.Time.UTC()
		claim.ReportDate = &date
	}
	if aging.Valid {
		days := int(aging.Int64)
		claim.AgingDays = &days
	}
	return claim, nil
}
