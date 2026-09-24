package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
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

// Nomor parameter pertama untuk penanda /*SCOPE*/ pada masing-masing kueri.
//
// Keduanya berbeda karena outstanding_list memakai dua parameter tambahan untuk paginasi.
// Lihat kepala outstanding.sql; keduanya dijaga uji di query_test.go.
const (
	scopeFirstBindList  = 11
	scopeFirstBindCount = 9
)

// List membaca satu halaman klaim yang masih berjalan beserta jumlah seluruh yang cocok.
func (r *Repo) List(ctx context.Context, f inboxoutstanding.Filter) (inboxoutstanding.Page, error) {
	f = f.Normalize()

	filters := filterArgs(f)

	countSQL, countArgs := expandScope(query("outstanding_count"), f.Scope, scopeFirstBindCount)
	countArgs = append(append([]any(nil), filters...), countArgs...)

	var total int
	if err := r.db.QueryRowContext(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return inboxoutstanding.Page{}, fmt.Errorf("inboxoutstanding/sqlstore: menghitung klaim: %w", err)
	}

	listSQL, scopeArgs := expandScope(query("outstanding_list"), f.Scope, scopeFirstBindList)

	// Urutan argumen mengikuti NOMOR parameter, bukan urutan kemunculannya di dalam teks:
	// paginasi memakai :9 dan :10, sehingga ia disisipkan sebelum argumen scope.
	listArgs := append(append([]any(nil), filters...), f.Offset, f.Limit)
	listArgs = append(listArgs, scopeArgs...)

	rows, err := r.db.QueryContext(ctx, listSQL, listArgs...)
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

// filterArgs menyusun delapan argumen pertama, sama untuk kedua kueri.
//
// Pola "NULL berarti tidak menyaring" dipakai supaya satu teks SQL melayani seluruh
// kombinasi penyaring. Menyusun WHERE-nya di Go akan mengembalikan SQL ke dalam kode —
// persis yang aturan §4.3 larang.
func filterArgs(f inboxoutstanding.Filter) []any {
	search := nilIfEmpty(f.Search)
	pattern := nilIfEmpty(searchPattern(f.Search))

	// Tahap dan cabang dikirim DUA KALI, dan itu bukan kelalaian.
	//
	// Keduanya muncul dua kali di dalam SQL — sekali pada `IS NULL`, sekali pada
	// perbandingannya — dan driver mengikat argumen menurut urutan KEMUNCULAN penanda,
	// bukan menurut nomornya. Mengirim sekali menghasilkan ORA-01008. Lihat kepala
	// outstanding.sql.
	stage := nilIfEmpty(strings.ToUpper(f.Stage))
	branch := nilIfEmpty(strings.ToUpper(f.BranchCode))

	return []any{
		search,  // :1 penentu apakah pencarian aktif
		pattern, // :2 PYID
		pattern, // :3 POLICYNO
		pattern, // :4 USERTEKNIS_1
		stage,   // :5 IS NULL
		stage,   // :6 perbandingan
		branch,  // :7 IS NULL
		branch,  // :8 perbandingan
	}
}

// expandScope mengganti penanda /*SCOPE*/ dengan klausa batas data.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang disisipkan hanyalah `:9, :11, …` — PENANDA parameter, bukan nilainya. Seluruh nilai
// tetap dikirim terpisah lewat args, sehingga celah `{ASIS:…}` warisan tetap tertutup.
// Satu-satunya teks bukan-penanda yang disisipkan adalah `AND 1 = 0`, dan ia konstanta di
// dalam kode — tidak berasal dari data mana pun.
//
// # Empat keadaan, dan yang terakhir yang paling penting
//
//	Unrestricted            -> tidak ada klausa sama sekali; seluruh lini terlihat
//	ada GroupPanels         -> AND TRIM(k.GROUPPANEL_1) IN (…)
//	ada ExcludedBusinessG.  -> AND (k.BUSINESSGROUPID IS NULL OR … NOT IN (…))
//	tidak satu pun terisi   -> AND 1 = 0, yaitu TIDAK MELOLOSKAN APA PUN
//
// Keduanya dapat berlaku bersamaan: NONMBU menyaring Group Panel DAN mengecualikan
// kelompok bisnis, sedangkan BONDING hanya mengecualikan kelompok bisnis.
//
// Keadaan terakhir adalah gagal TERTUTUP. Sebuah scope yang tidak menyebut apa pun dan
// tidak pula menyatakan dirinya tanpa batas hampir pasti cacat pemrograman; meloloskan
// semuanya akan mengubah cacat itu menjadi kebocoran data antar lini yang senyap.
//
// # Kenapa pengecualian memeriksa NULL lebih dulu
//
// `NOT IN` pada kolom bernilai NULL mengembalikan UNKNOWN, dan baris ber-UNKNOWN dibuang
// WHERE. Tanpa `IS NULL OR`, setiap klaim yang kelompok bisnisnya belum terisi akan hilang
// dari layar — bukan karena dikecualikan, melainkan karena aritmetika tiga-nilai SQL.
func expandScope(statement string, scope inboxoutstanding.LineScope, firstBind int) (string, []any) {
	const marker = "/*SCOPE*/"

	if scope.Unrestricted {
		return strings.Replace(statement, marker, "", 1), nil
	}
	if len(scope.GroupPanels) == 0 && len(scope.ExcludedBusinessGroups) == 0 {
		return strings.Replace(statement, marker, "AND 1 = 0", 1), nil
	}

	var (
		clauses []string
		args    []any
		bind    = firstBind
	)

	if len(scope.GroupPanels) > 0 {
		marks := make([]string, 0, len(scope.GroupPanels))
		for _, panel := range scope.GroupPanels {
			marks = append(marks, ":"+strconv.Itoa(bind))
			args = append(args, strings.TrimSpace(panel))
			bind++
		}
		clauses = append(clauses,
			"AND TRIM(k.GROUPPANEL_1) IN ("+strings.Join(marks, ", ")+")")
	}

	if len(scope.ExcludedBusinessGroups) > 0 {
		marks := make([]string, 0, len(scope.ExcludedBusinessGroups))
		for _, group := range scope.ExcludedBusinessGroups {
			marks = append(marks, ":"+strconv.Itoa(bind))
			args = append(args, strings.TrimSpace(group))
			bind++
		}
		clauses = append(clauses,
			"AND (k.BUSINESSGROUPID IS NULL OR TRIM(k.BUSINESSGROUPID) NOT IN ("+
				strings.Join(marks, ", ")+"))")
	}

	return strings.Replace(statement, marker, strings.Join(clauses, "\n   "), 1), args
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
		claimID         string
		number          sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		businessName    sql.NullString
		businessSource  sql.NullString
		branchName      sql.NullString
		groupPanel      sql.NullString
		businessGroupID sql.NullString
		registerDate    sql.NullTime
		createdAt       sql.NullTime
		lossDate        sql.NullTime
		reportDate      sql.NullTime
		aging           sql.NullInt64
		processStatus   sql.NullString
		claimStatus     sql.NullString
		progressStatus  sql.NullString
		technicalPIC    sql.NullString
		recordedBy      sql.NullString
		currentStage    sql.NullString
		currentHolder   sql.NullString
	)

	// Urutannya WAJIB sama persis dengan daftar kolom pada outstanding_list. Menambah
	// kolom di SQL tanpa menambahnya di sini menghasilkan galat jumlah kolom; menukar
	// urutannya menghasilkan data yang tertukar TANPA galat.
	if err := rows.Scan(
		&claimID, &number, &policyNumber, &insuredName, &businessName, &businessSource,
		&branchName, &groupPanel, &businessGroupID,
		&registerDate, &createdAt, &lossDate, &reportDate, &aging,
		&processStatus, &claimStatus, &progressStatus,
		&technicalPIC, &recordedBy, &currentStage, &currentHolder,
	); err != nil {
		return inboxoutstanding.OutstandingClaim{}, err
	}

	claim := inboxoutstanding.OutstandingClaim{
		ClaimID:         claimID,
		ClaimNumber:     strings.TrimSpace(number.String),
		PolicyNumber:    strings.TrimSpace(policyNumber.String),
		InsuredName:     insuredName.String,
		BusinessName:    businessName.String,
		BusinessSource:  businessSource.String,
		BranchName:      branchName.String,
		GroupPanel:      strings.TrimSpace(groupPanel.String),
		BusinessGroupID: strings.TrimSpace(businessGroupID.String),
		ProcessStatus:   strings.TrimSpace(processStatus.String),
		ClaimStatus:     strings.TrimSpace(claimStatus.String),
		ProgressStatus:  strings.TrimSpace(progressStatus.String),
		TechnicalPIC:    strings.TrimSpace(technicalPIC.String),
		RecordedBy:      strings.TrimSpace(recordedBy.String),
		CurrentStage:    strings.TrimSpace(currentStage.String),
		CurrentHolder:   strings.TrimSpace(currentHolder.String),
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
