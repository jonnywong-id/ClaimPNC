package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/daftartipedokumenbisnis"
)

// Repo membaca dan menulis aturan dokumen pada satu basis data entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// ListBusinesses membaca lini bisnis yang sudah punya aturan dokumen.
func (r *Repo) ListBusinesses(ctx context.Context) ([]daftartipedokumenbisnis.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("business_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca daftar bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result, err := scanBusinesses(rows)
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris bisnis: %w", err)
	}
	return result, nil
}

// ListByBusiness membaca aturan dokumen milik satu lini bisnis.
func (r *Repo) ListByBusiness(ctx context.Context, businessID string) ([]daftartipedokumenbisnis.DocumentRule, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("rule_list_by_business"), strings.TrimSpace(businessID))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca aturan bisnis %q: %w", businessID, err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftartipedokumenbisnis.DocumentRule
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris aturan: %w", err)
		}
		result = append(result, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: menelusuri aturan bisnis %q: %w", businessID, err)
	}
	return result, nil
}

// Get membaca satu aturan lengkap dengan daftar jaminannya.
//
// Dua kueri, bukan satu JOIN. Sebabnya bentuk hasilnya: JOIN akan mengembalikan baris
// aturan berulang sebanyak jaminannya, dan menyusunnya kembali menjadi satu objek berarti
// menulis pengelompokan sendiri di Go. Untuk satu baris yang dibuka pengguna, dua kueri
// jauh lebih terbaca dan bedanya tidak terukur.
func (r *Repo) Get(ctx context.Context, id string) (daftartipedokumenbisnis.DocumentRule, error) {
	key := strings.TrimSpace(id)

	row := r.db.QueryRowContext(ctx, getQuery("rule_get"), key)
	rule, err := scanRule(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftartipedokumenbisnis.DocumentRule{}, daftartipedokumenbisnis.ErrNotFound
	case err != nil:
		return daftartipedokumenbisnis.DocumentRule{}, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca aturan %q: %w", id, err)
	}

	coverages, err := r.coveragesOf(ctx, key)
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}
	rule.Coverages = coverages
	return rule, nil
}

// InsertBatch menyisipkan perkalian bisnis kali baris aturan.
//
// Seluruhnya dalam SATU transaksi, berbeda dari sistem lama yang memanggil procedure-nya
// sekali per baris dengan COMMIT masing-masing (`UpdateDetTypeDocBusiness_SQL`). Di sana
// kegagalan di tengah meninggalkan sebagian bisnis terisi dan sebagian tidak, tanpa satu
// pun tanda di layar — dan petugas yang mengulanginya memperoleh baris kembar pada bisnis
// yang sudah telanjur berhasil. `D-68` memindahkan kepemilikan transaksi ke Go persis
// untuk kelas kegagalan ini.
func (r *Repo) InsertBatch(
	ctx context.Context,
	input daftartipedokumenbisnis.BatchInput,
	by daftartipedokumenbisnis.Editor,
) ([]daftartipedokumenbisnis.DocumentRule, error) {
	var saved []daftartipedokumenbisnis.DocumentRule

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		site, err := siteCode(ctx, tx)
		if err != nil {
			return err
		}

		for _, businessID := range input.BusinessIDs {
			for _, rule := range input.Rules {
				id, err := nextID(ctx, tx, site)
				if err != nil {
					return err
				}

				_, err = tx.ExecContext(ctx, getQuery("rule_insert"),
					id,
					businessID,
					rule.DocumentTypeID,
					nullIfEmpty(rule.ObjectDocID),
					rule.DetailTypeDocID,
					rule.DetailDocument,
					mandatoryCode(rule.Mandatory),
					rule.MinDocument,
					by.At,
					by.Identity,
				)
				if err != nil {
					return fmt.Errorf("daftartipedokumenbisnis/sqlstore: menyisipkan aturan %q: %w", id, err)
				}

				saved = append(saved, daftartipedokumenbisnis.DocumentRule{
					ID:              id,
					BusinessID:      businessID,
					DocumentTypeID:  rule.DocumentTypeID,
					ObjectDocID:     rule.ObjectDocID,
					DetailTypeDocID: rule.DetailTypeDocID,
					DetailDocument:  rule.DetailDocument,
					Mandatory:       rule.Mandatory,
					MinDocument:     rule.MinDocument,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

// Update mengubah satu baris aturan.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	input daftartipedokumenbisnis.Input,
	by daftartipedokumenbisnis.Editor,
) (daftartipedokumenbisnis.DocumentRule, error) {
	key := strings.TrimSpace(id)

	result, err := r.db.ExecContext(ctx, getQuery("rule_update"),
		input.DocumentTypeID,
		nullIfEmpty(input.ObjectDocID),
		input.DetailTypeDocID,
		input.DetailDocument,
		mandatoryCode(input.Mandatory),
		input.MinDocument,
		by.At,
		by.Identity,
		key,
	)
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, fmt.Errorf("daftartipedokumenbisnis/sqlstore: memperbarui aturan %q: %w", id, err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi.
	//
	// Kegagalan RowsAffected sendiri TIDAK diperlakukan sebagai baris hilang — sebagian
	// driver memang tidak dapat melaporkannya, dan menolak penyimpanan karena keterbatasan
	// driver akan menampilkan galat atas perubahan yang justru berhasil.
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return daftartipedokumenbisnis.DocumentRule{}, daftartipedokumenbisnis.ErrNotFound
	}

	// Baris dibaca ulang, bukan disusun dari masukan. Dua nama — bisnis dan tahap dokumen
	// — berasal dari join dan tidak pernah dikirim pengguna; menyusunnya sendiri berarti
	// mengembalikan baris yang kedua namanya kosong, lalu layar menampilkannya sebagai
	// data yang baru saja hilang.
	return r.Get(ctx, key)
}

// AddCoverage menambahkan satu jaminan pada sebuah aturan.
//
// Menambah, bukan mengganti — lihat komentar seam-nya di lapisan domain. Jaminan yang
// sudah ada dibiarkan dan tidak dilaporkan sebagai galat, meniru
// `PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:55`.
func (r *Repo) AddCoverage(ctx context.Context, id string, coverageID string) (daftartipedokumenbisnis.DocumentRule, error) {
	key := strings.TrimSpace(id)
	coverage := strings.TrimSpace(coverageID)

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		// Barisnya dipastikan ada lebih dulu, dan bisnisnya diambil dari sana — bukan dari
		// pemanggil. Dua sebab:
		//
		// Pertama, tabel jaminan tidak memiliki foreign key yang menolak baris yatim
		// (`R-08`: DDL-nya tidak ada di export), sehingga tanpa pemeriksaan ini jaminan
		// dapat dipasang pada aturan yang tidak pernah ada — dan baris seperti itu tidak
		// terlihat di layar mana pun.
		//
		// Kedua, BUSINESSID pada baris jaminan WAJIB sama dengan milik baris aturannya:
		// keenam kueri klaim mencocokkan keduanya sekaligus
		// (`where a.id=b.id and a.businessid=b.businessid`). Membiarkan pemanggil
		// mengirimkannya berarti membuka kemungkinan keduanya berbeda, dan jaminan yang
		// tidak cocok itu **diam** — ia tidak menimbulkan galat, hanya membuat dokumennya
		// tidak pernah menjadi wajib.
		var businessID sql.NullString
		err := tx.QueryRowContext(ctx, getQuery("rule_business_of"), key).Scan(&businessID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return daftartipedokumenbisnis.ErrNotFound
		case err != nil:
			return fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca aturan %q: %w", id, err)
		}

		var exists int
		if err := tx.QueryRowContext(ctx, getQuery("coverage_exists"), key, coverage).Scan(&exists); err != nil {
			return fmt.Errorf("daftartipedokumenbisnis/sqlstore: memeriksa jaminan %q: %w", coverage, err)
		}
		if exists > 0 {
			return nil
		}

		if _, err := tx.ExecContext(ctx, getQuery("coverage_insert"),
			key, strings.TrimSpace(businessID.String), coverage); err != nil {
			return fmt.Errorf("daftartipedokumenbisnis/sqlstore: menambahkan jaminan %q: %w", coverage, err)
		}
		return nil
	})
	if err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}
	return r.Get(ctx, key)
}

// CheckTable memastikan kedua tabel modul ini ada dan dapat dibaca akun aplikasi.
func (r *Repo) CheckTable(ctx context.Context) error {
	for _, name := range []string{"rule_check_table", "coverage_check_table"} {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			return fmt.Errorf("daftartipedokumenbisnis/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, err)
		}
		closeErr := rows.Err()
		_ = rows.Close()
		if closeErr != nil {
			return fmt.Errorf("daftartipedokumenbisnis/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, closeErr)
		}
	}
	return nil
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
func (r *Repo) inTransaction(ctx context.Context, work func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("daftartipedokumenbisnis/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun.
	// Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan
	// menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if err := work(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("daftartipedokumenbisnis/sqlstore: menutup transaksi: %w", err)
	}
	return nil
}

// coveragesOf membaca jaminan milik satu aturan.
func (r *Repo) coveragesOf(ctx context.Context, id string) ([]daftartipedokumenbisnis.Coverage, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("coverage_list"), id)
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca jaminan aturan %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftartipedokumenbisnis.Coverage
	for rows.Next() {
		var coverageID sql.NullString
		if err := rows.Scan(&coverageID); err != nil {
			return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris jaminan: %w", err)
		}
		result = append(result, daftartipedokumenbisnis.Coverage{
			ID: strings.TrimSpace(coverageID.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: menelusuri jaminan aturan %q: %w", id, err)
	}
	return result, nil
}

// siteCode membaca kode situs, awalan ID baris baru.
func siteCode(ctx context.Context, tx *sql.Tx) (string, error) {
	var site sql.NullString
	err := tx.QueryRowContext(ctx, getQuery("rule_site")).Scan(&site)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Pesannya menyebut tabelnya, bukan sekadar "tidak ada baris": tanpa kode situs
		// tidak satu pun ID dapat diterbitkan, dan sebabnya selalu sama — M_SITE_DATABASE
		// belum punya baris ber-CURRENT_SITE='1' di basis data entitas itu. Menyebutnya
		// menghemat satu putaran penelusuran.
		return "", errors.New("daftartipedokumenbisnis/sqlstore: POOLDATA.M_SITE_DATABASE tidak memuat baris ber-CURRENT_SITE '1'")
	case err != nil:
		return "", fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca kode situs: %w", err)
	}
	return strings.TrimSpace(site.String), nil
}

// nextID mengambil nomor urut berikutnya dan memformatnya menjadi ID.
func nextID(ctx context.Context, tx *sql.Tx, site string) (string, error) {
	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("rule_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftartipedokumenbisnis/sqlstore: mengambil nomor urut: %w", err)
	}
	return daftartipedokumenbisnis.FormatID(site, sequence), nil
}

// mandatoryCode mengubah penanda wajib menjadi nilai yang tersimpan di STS_WAJIB.
//
// TEKS '1' dan '0', bukan angka. Buktinya keenam kueri pembacanya, yang seluruhnya
// membandingkannya dengan literal berkutip — antara lain
// `RDB List/BrowseRegisterCvg-SQL.xml`: `WHEN sts_wajib = '0' THEN 'Tidak'`.
func mandatoryCode(mandatory bool) string {
	if mandatory {
		return "1"
	}
	return "0"
}

// nullIfEmpty menulis NULL alih-alih teks kosong.
//
// Dipakai OBJECT_DOC_ID saja, dan itu meniru sistem lama:
// `Activity/InsertDetailTypeDocumentBusiness_act-Act.xml:944` mengirim teks kosong ketika
// keterangan objeknya kosong, dan `RDB List/BrowseCommitee_upload-SQL.xml` memeriksanya
// dengan `object_doc_id is null`. Menyimpan teks kosong akan membuat baris itu lolos dari
// pemeriksaan NULL dan gagal pula pada pemeriksaan IN — dokumen yang seharusnya tidak
// wajib menjadi tidak terbaca sama sekali.
func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan berbentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(target ...any) error }

// scanRule membaca satu baris hasil kueri menjadi DocumentRule.
//
// Seluruh kolom teks dibaca lewat tipe Null* lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena constraint tabelnya tidak diketahui (`R-08` — DDL-nya
// tidak ada di export). Ditambah satu sebab khas kueri ini: ketiga nama datang dari LEFT
// JOIN, sehingga NULL adalah keadaan yang memang diharapkan terjadi.
//
// STS_WAJIB dibaca sebagai TEKS. Nilai selain "1" dijawab "tidak wajib", termasuk NULL —
// bukan ditolak sebagai galat. Baris lama dapat memuat apa saja, dan menggagalkan seluruh
// daftar karena satu baris berisi nilai tak terduga jauh lebih buruk daripada
// menampilkannya sebagai "Tidak".
func scanRule(rows rowScanner) (daftartipedokumenbisnis.DocumentRule, error) {
	var id, businessID, businessName, documentTypeID, documentTypeName sql.NullString
	var objectDocID, objectDocName, detailTypeDocID, detailDocument, mandatory sql.NullString
	var minDocument sql.NullInt64

	if err := rows.Scan(
		&id, &businessID, &businessName, &documentTypeID, &documentTypeName,
		&objectDocID, &objectDocName, &detailTypeDocID, &detailDocument, &mandatory,
		&minDocument,
	); err != nil {
		return daftartipedokumenbisnis.DocumentRule{}, err
	}

	return daftartipedokumenbisnis.DocumentRule{
		ID:               strings.TrimSpace(id.String),
		BusinessID:       strings.TrimSpace(businessID.String),
		BusinessName:     strings.TrimSpace(businessName.String),
		DocumentTypeID:   strings.TrimSpace(documentTypeID.String),
		DocumentTypeName: strings.TrimSpace(documentTypeName.String),
		ObjectDocID:      strings.TrimSpace(objectDocID.String),
		ObjectDocName:    strings.TrimSpace(objectDocName.String),
		DetailTypeDocID:  strings.TrimSpace(detailTypeDocID.String),
		DetailDocument:   strings.TrimSpace(detailDocument.String),
		Mandatory:        strings.TrimSpace(mandatory.String) == "1",
		MinDocument:      int(minDocument.Int64),
	}, nil
}

// scanBusinesses membaca hasil kueri daftar bisnis.
func scanBusinesses(rows *sql.Rows) ([]daftartipedokumenbisnis.Business, error) {
	var result []daftartipedokumenbisnis.Business
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result = append(result, daftartipedokumenbisnis.Business{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	return result, rows.Err()
}

// scanReferences membaca hasil kueri daftar pilihan.
func scanReferences(rows *sql.Rows) ([]daftartipedokumenbisnis.Reference, error) {
	var result []daftartipedokumenbisnis.Reference
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result = append(result, daftartipedokumenbisnis.Reference{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
	}
	return result, rows.Err()
}

// BusinessRepo membaca master lini bisnis sebagai daftar pilihan.
//
// Tipe tersendiri, bukan metode tambahan pada Repo, supaya batas kepemilikannya terbaca
// dari bentuknya: ia hanya punya List, dan tidak ada tempat untuk menambahkan operasi
// tulis tanpa sengaja (`P-1`, `D-03`).
type BusinessRepo struct{ db *sql.DB }

// NewBusinessRepo membentuk pembaca master lini bisnis.
func NewBusinessRepo(db *sql.DB) *BusinessRepo { return &BusinessRepo{db: db} }

// List membaca SELURUH lini bisnis dari POOLDATA.BUSINESS.
func (r *BusinessRepo) List(ctx context.Context) ([]daftartipedokumenbisnis.Business, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("business_choice_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca master bisnis: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result, err := scanBusinesses(rows)
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris master bisnis: %w", err)
	}
	return result, nil
}

// DocumentTypeRepo membaca master tahap dokumen (V_LST_DOC_TYPE).
type DocumentTypeRepo struct{ db *sql.DB }

// NewDocumentTypeRepo membentuk pembaca master tahap dokumen.
func NewDocumentTypeRepo(db *sql.DB) *DocumentTypeRepo { return &DocumentTypeRepo{db: db} }

// List membaca seluruh tahap dokumen.
func (r *DocumentTypeRepo) List(ctx context.Context) ([]daftartipedokumenbisnis.Reference, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("document_type_choice_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca master tipe dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result, err := scanReferences(rows)
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris master tipe dokumen: %w", err)
	}
	return result, nil
}

// DetailTypeDocRepo membaca master rincian dokumen (V_LST_DET_TYPE_DOC).
type DetailTypeDocRepo struct{ db *sql.DB }

// NewDetailTypeDocRepo membentuk pembaca master rincian dokumen.
func NewDetailTypeDocRepo(db *sql.DB) *DetailTypeDocRepo { return &DetailTypeDocRepo{db: db} }

// List membaca seluruh rincian dokumen beserta tahap pemiliknya.
//
// Tiga kolom, bukan dua seperti kedua daftar pilihan lain, sehingga ia tidak dapat memakai
// scanReferences — lihat komentar Reference.ParentID di lapisan domain untuk alasan kolom
// ketiganya ada.
func (r *DetailTypeDocRepo) List(ctx context.Context) ([]daftartipedokumenbisnis.Reference, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_type_doc_choice_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca master detail dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftartipedokumenbisnis.Reference
	for rows.Next() {
		var id, name, parentID sql.NullString
		if err := rows.Scan(&id, &name, &parentID); err != nil {
			return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris master detail dokumen: %w", err)
		}
		result = append(result, daftartipedokumenbisnis.Reference{
			ID:       strings.TrimSpace(id.String),
			Name:     strings.TrimSpace(name.String),
			ParentID: strings.TrimSpace(parentID.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: menelusuri master detail dokumen: %w", err)
	}
	return result, nil
}

// ObjectDocRepo membaca master objek dokumen (V_LST_DOC_OBJ).
type ObjectDocRepo struct{ db *sql.DB }

// NewObjectDocRepo membentuk pembaca master objek dokumen.
func NewObjectDocRepo(db *sql.DB) *ObjectDocRepo { return &ObjectDocRepo{db: db} }

// List membaca seluruh objek dokumen.
func (r *ObjectDocRepo) List(ctx context.Context) ([]daftartipedokumenbisnis.Reference, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("object_doc_choice_list"))
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca master objek dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result, err := scanReferences(rows)
	if err != nil {
		return nil, fmt.Errorf("daftartipedokumenbisnis/sqlstore: membaca baris master objek dokumen: %w", err)
	}
	return result, nil
}

var (
	_ daftartipedokumenbisnis.Repo              = (*Repo)(nil)
	_ daftartipedokumenbisnis.BusinessRepo      = (*BusinessRepo)(nil)
	_ daftartipedokumenbisnis.DocumentTypeRepo  = (*DocumentTypeRepo)(nil)
	_ daftartipedokumenbisnis.DetailTypeDocRepo = (*DetailTypeDocRepo)(nil)
	_ daftartipedokumenbisnis.ObjectDocRepo     = (*ObjectDocRepo)(nil)
)
