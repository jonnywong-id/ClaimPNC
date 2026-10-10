package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxcompliance"
)

// Repo membaca antrean Compliance dari SATU basis data entitas, dan menulis SATU tabel.
//
// Tiga tabel warisan Pega dibaca saja; satu-satunya yang ditulis adalah
// `POOLDATA.T_CLAIM_COMPLIANCE_H`, tabel baru yang tidak dikenal Pega. Pembedaan itu yang
// membuatnya tidak melanggar `P-1` — lihat kepala berkas inboxcompliance.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// plan menyebut kueri mana yang melayani sebuah tab, bagaimana argumennya disusun, dan
// pemindai mana yang membaca hasilnya.
//
// Ketiganya dikumpulkan dalam satu nilai karena ketiganya HARUS berubah bersamaan: kueri
// yang ditukar tanpa menukar pemindainya menghasilkan nilai yang masuk ke isian yang salah,
// dan itu bukan galat melainkan kolom yang tertukar di layar.
type plan struct {
	list  string
	count string

	// listArgs menyusun argumen bind kueri daftar, sesuai urutan `:1`, `:2`, `:3`.
	listArgs func(inboxcompliance.Query, inboxcompliance.Pagination) []any

	// countArgs menyusun argumen bind kueri hitung. Boleh mengembalikan nil.
	countArgs func(inboxcompliance.Query) []any

	scan func(scanner) (inboxcompliance.WorkItem, error)
}

// plans memetakan kode tab ke kuerinya.
//
// Peta ini adalah satu-satunya tempat kode tab bertemu nama kueri. Tab yang tidak ada di
// sini menghasilkan galat yang menyebut kodenya — bukan kueri kosong yang mengembalikan nol
// baris dan terbaca seperti antrean yang memang kosong.
var plans = map[string]plan{
	inboxcompliance.TabCompliance: {
		list:  "list_compliance",
		count: "count_compliance",
		listArgs: func(q inboxcompliance.Query, p inboxcompliance.Pagination) []any {
			return []any{q.Workbasket, p.Offset(), p.Size}
		},
		countArgs: func(q inboxcompliance.Query) []any {
			return []any{q.Workbasket}
		},
		scan: scanComplianceItem,
	},
	inboxcompliance.TabPostAudit: {
		list:  "list_post_audit",
		count: "count_post_audit",
		listArgs: func(_ inboxcompliance.Query, p inboxcompliance.Pagination) []any {
			return []any{p.Offset(), p.Size}
		},
		// Tanpa argumen: tab ini tidak punya penyaring apa pun. Lihat catatan di
		// inboxcompliance.sql.
		countArgs: func(inboxcompliance.Query) []any { return nil },
		scan:      scanPostAuditItem,
	},
}

// List mengambil satu halaman antrean beserta jumlah seluruh baris yang cocok.
//
// # Kenapa dua kueri, bukan satu
//
// Karena jumlah total dan isi halaman adalah dua pertanyaan berbeda, dan menyatukannya
// lewat fungsi jendela membuat basis data menghitung total untuk SETIAP baris halaman.
// Dua kueri dengan predikat yang sama lebih murah dan jauh lebih mudah dibaca DBA.
//
// Keduanya dijalankan di luar transaksi. Antrean dapat berubah di antara keduanya, sehingga
// total dan isi halaman secara teori dapat sedikit berselisih — dan itu diterima: membuka
// transaksi hanya untuk menyamakan dua bacaan pada layar yang menyegarkan dirinya sendiri
// berarti menahan kunci baris pada tabel yang sedang dipakai Pega melayani produksi.
func (r *Repo) List(
	ctx context.Context,
	q inboxcompliance.Query,
	page inboxcompliance.Pagination,
) (inboxcompliance.Page, error) {
	selected, known := plans[q.Tab.Code]
	if !known {
		return inboxcompliance.Page{}, fmt.Errorf(
			"inboxcompliance/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}

	clean := page.Normalize()

	total, err := r.count(ctx, selected, q)
	if err != nil {
		return inboxcompliance.Page{}, err
	}

	items, err := r.rows(ctx, selected, q, clean)
	if err != nil {
		return inboxcompliance.Page{}, err
	}

	return inboxcompliance.Page{Items: items, Total: total, Pagination: clean}, nil
}

// count menghitung seluruh baris yang cocok.
func (r *Repo) count(
	ctx context.Context, selected plan, q inboxcompliance.Query,
) (int, error) {
	var total int
	if err := r.db.QueryRowContext(
		ctx, query(selected.count), selected.countArgs(q)...,
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("menjalankan kueri %s: %w", selected.count, err)
	}
	return total, nil
}

// rows mengambil satu halaman baris.
func (r *Repo) rows(
	ctx context.Context,
	selected plan,
	q inboxcompliance.Query,
	page inboxcompliance.Pagination,
) ([]inboxcompliance.WorkItem, error) {
	rows, err := r.db.QueryContext(ctx, query(selected.list), selected.listArgs(q, page)...)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri %s: %w", selected.list, err)
	}
	defer rows.Close()

	items := []inboxcompliance.WorkItem{}
	for rows.Next() {
		item, err := selected.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris kueri %s: %w", selected.list, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri %s: %w", selected.list, err)
	}

	return items, nil
}

// CheckTable memastikan seluruh tabel modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
//
// Dua pemeriksaan terpisah, dan pesannya pun terpisah. Ketiga tabel yang pertama adalah
// tabel WARISAN Pega — kegagalan di sana hampir selalu berarti grant-nya belum diberikan,
// bukan objeknya belum dibuat. Tabel yang kedua BARU, sehingga kegagalannya justru
// kemungkinan besar berarti tabelnya memang belum ada di entitas itu. Menyatukan keduanya
// akan membuat penelusurannya menempuh satu putaran lebih panjang.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASSIGN_WORKBASKET, POOLDATA.T_CLAIM_PNC, "+
				"dan POOLDATA.T_CLAIMLIST_ADMIN: %w", err)
	}

	// ExecContext, BUKAN QueryRow().Scan().
	//
	// Kueri pemeriksanya menyebut keenam kolom dan ber-`WHERE 1 = 0`, sehingga ia
	// mengembalikan NOL BARIS. `QueryRow().Scan()` atas nol baris memulangkan
	// `sql.ErrNoRows` — dan itu akan dilaporkan sebagai kegagalan padahal pemeriksaannya
	// justru BERHASIL: pernyataannya ter-parse, yang berarti tabel dan keenam kolomnya ada.
	//
	// Yang diuji di sini adalah PARSING, bukan isi. Oracle memvalidasi seluruh nama kolom
	// saat mem-parse, sehingga kolom yang salah nama gagal di sini — jauh sebelum petugas
	// membuka layarnya.
	if _, err := r.db.ExecContext(ctx, query("check_table_post_audit")); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_COMPLIANCE_H beserta keenam kolom yang dipakai "+
				"(CASEID, NO_KLAIM, NAMA_TERTANGGUNG, NO_POLIS, REMARKS, "+
				"TGL_KIRIM_POST_AUDIT): %w", storeMissing(err))
	}

	return nil
}

// CheckPostAuditWritable memeriksa prasyarat jalur TULIS, terpisah dari CheckTable.
//
// # Kenapa terpisah
//
// Karena akibat kegagalannya berbeda. Tanpa hak baca, seluruh layar kosong dan modulnya
// memang mati. Tanpa sequence, layar tetap utuh, kedua tab tetap terbaca, dan yang gagal
// HANYA tombol Kirim ke Post Audit. Melaporkan keduanya sebagai satu kegagalan membuat
// modul yang sebagian besar berfungsi terbaca seperti modul yang tidak berfungsi.
//
// # Apa yang dibuktikan, dan apa yang tidak
//
// Dibuktikan: sequence-nya ada dan dapat diakses akun aplikasi.
//
// TIDAK dibuktikan: hak INSERT pada `POOLDATA.T_CLAIM_COMPLIANCE_H`. Membuktikannya
// menuntut penyisipan baris sungguhan, dan `-periksa` berjanji tidak menulis apa pun.
// Batas itu disebutkan dalam pesan galatnya supaya tidak dikira sudah tercakup.
func (r *Repo) CheckPostAuditWritable(ctx context.Context) error {
	var found int

	if err := r.db.QueryRowContext(
		ctx, query("check_post_audit_sequence"), sequenceOwner, sequenceName,
	).Scan(&found); err != nil {
		return fmt.Errorf("membaca katalog ALL_SEQUENCES: %w", err)
	}

	if found == 0 {
		return fmt.Errorf(
			"sequence %s.%s tidak ada atau tidak dapat diakses akun aplikasi; "+
				"jalankan migrations/0011_post_audit_compliance.up.sql beserta kedua "+
				"GRANT di dalamnya (DBA, `D-63`), di basis data SETIAP entitas",
			sequenceOwner, sequenceName)
	}

	return nil
}

// scanner adalah bentuk minimal yang dibutuhkan scanWorkItem, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanComplianceItem memindai satu baris tab Compliance menjadi WorkItem.
//
// Urutannya WAJIB sama dengan complianceColumns dan dengan urutan kolom di
// inboxcompliance.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom dipindai lewat tipe yang mengizinkan NULL. Itu bukan kehati-hatian
// berlebihan: `COMPLIANCE_SENT_DATE` datang dari LEFT JOIN sehingga memang dapat NULL, dan
// kolom teks pada tabel warisan tidak punya jaminan NOT NULL apa pun selama DDL-nya belum
// ada (`R-08`).
func scanComplianceItem(row scanner) (inboxcompliance.WorkItem, error) {
	var (
		caseID, reference, policyNumber, insuredName sql.NullString
		businessName, branchName, adminName          sql.NullString
		groupPanel, technician                       sql.NullString
		complianceSentDate                           sql.NullTime
	)

	err := row.Scan(
		&caseID, &reference, &policyNumber, &insuredName,
		&businessName, &branchName, &adminName, &groupPanel, &technician,
		&complianceSentDate,
	)
	if err != nil {
		return inboxcompliance.WorkItem{}, err
	}

	return inboxcompliance.WorkItem{
		CaseID:       caseID.String,
		Reference:    reference.String,
		PolicyNumber: policyNumber.String,
		InsuredName:  insuredName.String,
		BusinessName: businessName.String,
		BranchName:   branchName.String,
		AdminName:    adminName.String,
		// NULL menjadi teks kosong, dan itu AMAN: ActionsFor memperlakukan lini bisnis
		// kosong sebagai bukan-Travel dan bukan-PA — sama dengan hasil
		// `compareTwoValues("", "=", "005")` di Pega.
		GroupPanel:         groupPanel.String,
		TechnicianID:       technician.String,
		ComplianceSentDate: timeOrNil(complianceSentDate),
	}, nil
}

// scanPostAuditItem memindai satu baris tab Post Audit menjadi WorkItem.
//
// Urutannya WAJIB sama dengan postAuditColumns dan dengan urutan kolom di
// inboxcompliance.sql. Ketiganya dijaga query_test.go.
//
// SELURUH kolom `POOLDATA.T_CLAIM_COMPLIANCE_H` boleh NULL — DDL-nya tidak memuat satu pun
// `NOT NULL`, tidak ada primary key, dan tidak ada constraint unik. Itu bukan dugaan
// melainkan bacaan langsung dari DDL yang diterima 2026-09-24.
func scanPostAuditItem(row scanner) (inboxcompliance.WorkItem, error) {
	var (
		caseID, claimNumber, insuredName, policyNumber sql.NullString
		complianceRemarks                              sql.NullString
		postAuditSentDate                              sql.NullTime
	)

	err := row.Scan(
		&caseID, &claimNumber, &insuredName, &policyNumber,
		&complianceRemarks, &postAuditSentDate,
	)
	if err != nil {
		return inboxcompliance.WorkItem{}, err
	}

	return inboxcompliance.WorkItem{
		CaseID:      caseID.String,
		ClaimNumber: claimNumber.String,

		// Reference mengambil nilai yang SAMA dengan ClaimNumber, dan itu disengaja:
		// `NO_KLAIM` berisi kunci teknis Pega ("ASM-FW-GCNMFW-WORK PNC-2114"), sehingga ia
		// sekaligus yang dibutuhkan tombol buka detail klaim.
		//
		// Ia tetap disalin ke dua isian, bukan dibaca ulang dari ClaimNumber saat
		// dibutuhkan, supaya layar memakai isian yang sama untuk tombol detail di kedua
		// tab — tanpa perlu tahu tab mana yang sedang terbuka.
		Reference: claimNumber.String,

		InsuredName:       insuredName.String,
		PolicyNumber:      policyNumber.String,
		ComplianceRemarks: complianceRemarks.String,
		PostAuditSentDate: timeOrNil(postAuditSentDate),
	}, nil
}

// timeOrNil mengubah kolom tanggal yang boleh NULL menjadi pointer.
//
// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari "belum
// diisi" saat ditampilkan, dan kolom Aging yang dihitung darinya akan menghasilkan angka
// dalam jutaan jam.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	at := value.Time
	return &at
}

// FindInQueue mencari satu klaim yang sedang menunggu di antrean Compliance.
//
// Nilai kedua bernilai salah bila klaimnya tidak ada di antrean — bukan galat. Klaimnya
// boleh jadi ada dan sehat, hanya sudah dikirim petugas lain atau sudah selesai.
func (r *Repo) FindInQueue(
	ctx context.Context, q inboxcompliance.Query, reference string,
) (inboxcompliance.WorkItem, bool, error) {
	row := r.db.QueryRowContext(ctx, query("find_compliance_claim"), q.Workbasket, reference)

	item, err := scanComplianceItem(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxcompliance.WorkItem{}, false, nil
	}
	if err != nil {
		return inboxcompliance.WorkItem{}, false, fmt.Errorf(
			"menjalankan kueri find_compliance_claim: %w", err)
	}

	return item, true, nil
}

// CreatePostAudit menulis satu baris Post Audit dan mengembalikannya beserta nomor terbit.
//
// # Kenapa satu transaksi
//
// Karena dua pernyataan berjalan berurutan: mengambil nomor urut, lalu menyisipkan
// barisnya. Bila yang kedua gagal sementara yang pertama sudah berjalan, nomornya hilang —
// dan lubang penomoran pada nomor yang dibaca orang selalu menimbulkan pertanyaan yang
// mahal dijawab.
//
// Transaksi TIDAK mengembalikan nomor yang sudah diambil: sequence Oracle memang tidak
// dapat dibatalkan oleh rollback, dan itu sifatnya di basis data mana pun. Yang dijamin
// transaksi ini hanyalah bahwa baris yang tersimpan selalu punya nomor, tidak pernah
// sebaliknya.
//
// # Kenapa bukan di lapisan aplikasi
//
// `08-TECHNICAL-STRATEGY.md` §4.5 menetapkan transaksi dimulai di lapisan aplikasi. Di sini
// keduanya adalah SATU operasi penyimpanan yang tidak punya arti terpisah — nomor tanpa
// baris bukan apa-apa — sehingga memecahnya ke usecase hanya akan memindahkan detail
// penyimpanan ke lapisan yang tidak boleh mengetahuinya.
func (r *Repo) CreatePostAudit(
	ctx context.Context, entry inboxcompliance.PostAuditEntry,
) (inboxcompliance.PostAuditEntry, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxcompliance.PostAuditEntry{}, fmt.Errorf("memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Nomornya diterbitkan BASIS DATA secara utuh — `CPL.26.1`, bukan angkanya saja.
	// Sintaksnya ditetapkan Work Owner; lihat number.go dan kueri yang dipanggil di sini.
	var caseID string
	if err := tx.QueryRowContext(
		ctx, query("post_audit_next_sequence"),
	).Scan(&caseID); err != nil {
		// Penyebab paling mungkin bukan cacat kode melainkan migrasi 0011 yang belum
		// dijalankan DBA, dan Oracle melaporkan keduanya dengan pesan yang sama
		// ("sequence does not exist") baik objeknya memang tidak ada maupun haknya belum
		// diberikan. Menyebut berkas migrasinya di sini membuat galat itu dapat
		// ditindaklanjuti tanpa menebak — lihat juga CheckPostAuditWritable, yang
		// menemukannya saat start sehingga seharusnya tidak pernah sampai ke sini.
		return inboxcompliance.PostAuditEntry{}, fmt.Errorf(
			"menerbitkan nomor Post Audit dari %s.%s "+
				"(bila sequence-nya belum ada, jalankan "+
				"migrations/0011_post_audit_compliance.up.sql): %w",
			sequenceOwner, sequenceName, err)
	}

	saved := entry
	saved.CaseID = caseID

	if _, err := tx.ExecContext(
		ctx, query("insert_post_audit"),
		saved.CaseID,
		saved.ClaimNumber,
		saved.InsuredName,
		saved.PolicyNumber,
		nullableText(saved.Remarks),
		saved.SentAt,
	); err != nil {
		return inboxcompliance.PostAuditEntry{}, fmt.Errorf(
			"menyimpan baris Post Audit: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return inboxcompliance.PostAuditEntry{}, fmt.Errorf("menutup transaksi: %w", err)
	}

	return saved, nil
}

// nullableText mengirim teks kosong sebagai NULL.
//
// Catatan yang tidak diisi disimpan sebagai NULL, bukan sebagai teks kosong. Keduanya
// tampil sama di layar, tetapi hanya yang pertama yang dapat dibedakan dari "diisi lalu
// dikosongkan" oleh siapa pun yang kelak membaca tabelnya langsung.
//
// Oracle sebenarnya menyimpan teks kosong SEBAGAI NULL dengan sendirinya, sehingga ini
// tidak mengubah apa pun di Oracle. Ia ditulis tetap supaya perilakunya sama ketika basis
// datanya berpindah ke PostgreSQL (`D-24`), yang membedakan keduanya.
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// FindDecision mengambil keputusan Compliance yang sudah tersimpan atas satu klaim.
//
// Nilai kedua bernilai salah ketika klaimnya belum pernah diputuskan — bukan galat. Setiap
// klaim yang baru masuk antrean berada dalam keadaan itu.
func (r *Repo) FindDecision(
	ctx context.Context, reference string,
) (inboxcompliance.Decision, bool, error) {
	row := r.db.QueryRowContext(ctx, query("find_compliance_decision"), reference)

	var (
		choice    sql.NullString
		note      sql.NullString
		remarks   sql.NullString
		decidedBy sql.NullString
		decidedAt sql.NullTime
		validated sql.NullTime
		sent      sql.NullTime
		comments  sql.NullString
	)

	err := row.Scan(
		&choice, &note, &remarks, &decidedBy, &decidedAt, &validated, &sent, &comments)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxcompliance.Decision{}, false, nil
	}
	if err != nil {
		return inboxcompliance.Decision{}, false, fmt.Errorf(
			"menjalankan kueri find_compliance_decision: %w", storeMissing(err))
	}

	decision := inboxcompliance.Decision{
		Reference: reference,
		Choice:    choice.String,
		Note:      note.String,
		Remarks:   remarks.String,
		DecidedBy: decidedBy.String,
	}
	if decidedAt.Valid {
		decision.DecidedAt = decidedAt.Time
	}

	// Kedua stempel waktu di bawah dibiarkan nil ketika kolomnya NULL, dan pembedaan itu
	// yang dibaca layar: `TGL_VALID` terisi HANYA pada pilihan Bayar/Valid, dan
	// `TGL_KIRIM_POST_AUDIT` HANYA pada Bayar/PostAudit. Memulangkan waktu nol tahun 1
	// akan membuat keduanya tampak selalu terisi.
	if validated.Valid {
		at := validated.Time
		decision.ValidatedAt = &at
	}
	if sent.Valid {
		at := sent.Time
		decision.SentToPostAuditAt = &at
	}

	// Komentarnya datang dari kolom yang SAMA, bukan dari tabel kedua — grid itu bagian
	// dari keputusan yang sama, dan form menyimpan keduanya dalam satu tombol.
	parsed, err := decodeComments(comments.String)
	if err != nil {
		return inboxcompliance.Decision{}, false, fmt.Errorf(
			"membaca grid komentar klaim %s: %w", reference, err)
	}
	decision.Comments = parsed

	return decision, true, nil
}

// SaveDecision menyimpan keputusan Compliance, menimpa keputusan sebelumnya atas klaim yang
// sama.
//
// # Ke tabel mana, dan apa yang TIDAK ikut tersentuh
//
// Ke `POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE`, tabel baru milik aplikasi ini. Klaimnya sendiri
// — `T_CLAIM_PNC` dan `PC_ASM_FW_GCNMFW_WORK` — TIDAK disentuh, karena keduanya masih
// dimiliki Pega selama masa paralel (`P-1`). Konsekuensinya dijelaskan di
// usecase.SubmitDecision, bagian "Akibat yang BELUM sampai ke klaim".
//
// Satu pernyataan, sehingga tidak perlu transaksi: `MERGE` sudah atomik dengan sendirinya.
func (r *Repo) SaveDecision(
	ctx context.Context, decision inboxcompliance.Decision,
) error {
	// Grid komentar dirakit menjadi satu kolom JSON, sehingga keputusan dan komentarnya
	// tersimpan dalam SATU pernyataan. Lihat comments_json.go.
	comments, err := encodeComments(decision.Comments)
	if err != nil {
		return err
	}

	if _, err := r.db.ExecContext(
		ctx, query("upsert_compliance_decision"),
		decision.Reference,
		decision.Choice,
		nullableText(decision.Note),
		nullableText(decision.Remarks),
		nullableText(decision.DecidedBy),
		decision.DecidedAt,
		nullableTime(decision.ValidatedAt),
		nullableTime(decision.SentToPostAuditAt),
		comments,
	); err != nil {
		// Sama seperti pada CreatePostAudit, penyebab paling mungkin adalah migrasinya
		// yang belum dijalankan DBA — bukan cacat kode. Menyebut berkasnya membuat galat
		// itu dapat ditindaklanjuti tanpa menebak.
		return fmt.Errorf(
			"menyimpan keputusan Compliance ke POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE "+
				"(bila tabelnya belum ada, jalankan "+
				"migrations/0012_keputusan_compliance.up.sql): %w", err)
	}

	// Komentarnya disimpan SESUDAH keputusannya, dan keduanya TIDAK dalam satu transaksi.
	//
	// Urutannya disengaja, sama alasannya dengan urutan keputusan→Post Audit pada
	// usecase.SubmitDecision: bila yang kedua gagal, yang tersimpan adalah keputusan tanpa
	// komentar — keadaan yang terlihat petugas dan dapat diulang karena form masih
	// terbuka. Kebalikannya lebih buruk: komentar tanpa keputusan yang menaunginya.
	//
	// Membungkus keduanya menuntut kepemilikan transaksi di lapisan aplikasi
	// (`08-TECHNICAL-STRATEGY.md` §4.5), yang seam Repo modul ini belum punya.
	return nil
}

// CheckDecisionWritable membuktikan tabel keputusan Compliance ada dan dapat diakses akun
// aplikasi.
//
// Ia dipanggil `-periksa` saat start. Seperti CheckPostAuditWritable, ia membuktikan
// keberadaan DAN hak baca — tetapi TIDAK membuktikan hak tulis, karena membuktikannya
// menuntut benar-benar menulis lalu membatalkannya.
func (r *Repo) CheckDecisionWritable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("check_table_decision")); err != nil {
		return fmt.Errorf(
			"tabel POOLDATA.CPNC_KEPUTUSAN_COMPLIANCE tidak ada atau tidak dapat diakses "+
				"akun aplikasi; jalankan migrations/0012_keputusan_compliance.up.sql "+
				"beserta GRANT di dalamnya (DBA, `D-63`), di basis data SETIAP entitas: %w",
			err)
	}
	return nil
}

// nullableTime mengirim waktu yang tidak terisi sebagai NULL.
//
// Pointer, bukan time.Time kosong, karena tahun 1 tidak dapat dibedakan dari "tidak
// terisi" — dan pada kedua kolom ini perbedaannya menentukan pilihan mana yang diambil.
func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

// storeMissing menandai "tabel tidak ada" sebagai ErrDecisionStoreMissing, tanpa membuang
// galat aslinya — yang tetap masuk log beserta nama tabelnya.
//
// # Kenapa pencocokan teks, bukan tipe galat
//
// Karena `database/sql` tidak membawa kode galat basis data, dan menariknya dari tipe
// khusus godror akan mengikat berkas ini pada satu driver — padahal `D-20` menetapkan satu
// set SQL yang berjalan di Oracle maupun PostgreSQL. Pola yang sama sudah dipakai
// `dokumenpenunjang/repo/sqlstore.linkError` untuk galat Oracle Net.
//
// Kedua kode di bawah berarti hal yang sama bagi pengguna, tetapi berbeda sebabnya:
//
//	ORA-00942  tabel/view tidak ada — ATAU ada tetapi akun aplikasi tidak punya hak
//	           SELECT atasnya. Oracle sengaja tidak membedakan keduanya, supaya
//	           keberadaan sebuah tabel tidak bocor ke akun yang tidak berhak.
//	ORA-00904  kolomnya tidak dikenal — tabelnya ada tetapi lebih tua daripada kueri ini
//
// Keduanya diselesaikan tindakan yang sama: jalankan migrasinya beserta GRANT di dalamnya.
func storeMissing(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToUpper(err.Error())
	for _, code := range []string{"ORA-00942", "ORA-00904"} {
		if strings.Contains(message, code) {
			return fmt.Errorf("%w: %w", inboxcompliance.ErrDecisionStoreMissing, err)
		}
	}
	return err
}
