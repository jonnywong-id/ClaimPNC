package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK, DATAPEGA.PC_ASSIGN_WORKBASKET, "+
				"dan POOLDATA.T_CLAIM_PNC: %w", err)
	}

	if err := r.db.QueryRowContext(
		ctx, query("check_table_post_audit"),
	).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca POOLDATA.T_CLAIM_COMPLIANCE_H: %w", err)
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
		complianceSentDate                           sql.NullTime
	)

	err := row.Scan(
		&caseID, &reference, &policyNumber, &insuredName,
		&businessName, &branchName, &adminName, &complianceSentDate,
	)
	if err != nil {
		return inboxcompliance.WorkItem{}, err
	}

	return inboxcompliance.WorkItem{
		CaseID:             caseID.String,
		Reference:          reference.String,
		PolicyNumber:       policyNumber.String,
		InsuredName:        insuredName.String,
		BusinessName:       businessName.String,
		BranchName:         branchName.String,
		AdminName:          adminName.String,
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

	var sequence int64
	if err := tx.QueryRowContext(
		ctx, query("post_audit_next_sequence"),
	).Scan(&sequence); err != nil {
		return inboxcompliance.PostAuditEntry{}, fmt.Errorf(
			"mengambil nomor urut Post Audit: %w", err)
	}

	saved := entry
	saved.CaseID = BuildCaseID(sequence)

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
