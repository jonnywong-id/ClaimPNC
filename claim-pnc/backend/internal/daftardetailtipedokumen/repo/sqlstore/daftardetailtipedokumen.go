package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/daftardetailtipedokumen"
)

// Repo membaca dan menulis detail tipe dokumen pada satu basis data entitas.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh rincian TANPA daftar bisnisnya.
func (r *Repo) List(ctx context.Context) ([]daftardetailtipedokumen.DetailType, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_list"))
	if err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca daftar detail: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetailtipedokumen.DetailType
	for rows.Next() {
		row, err := scanDetail(rows)
		if err != nil {
			return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca baris detail: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: menelusuri daftar detail: %w", err)
	}
	return result, nil
}

// Get membaca satu rincian lengkap dengan daftar bisnisnya.
//
// Dua kueri, bukan satu JOIN. Sebabnya bentuk hasilnya: JOIN akan mengembalikan baris
// induk berulang sebanyak bisnisnya, dan menyusunnya kembali menjadi satu objek berarti
// menulis pengelompokan sendiri di Go. Untuk satu baris yang dibuka pengguna, dua kueri
// jauh lebih terbaca dan bedanya tidak terukur.
func (r *Repo) Get(ctx context.Context, id string) (daftardetailtipedokumen.DetailType, error) {
	key := strings.TrimSpace(id)

	row := r.db.QueryRowContext(ctx, getQuery("detail_get"), key)
	detail, err := scanDetail(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return daftardetailtipedokumen.DetailType{}, daftardetailtipedokumen.ErrNotFound
	case err != nil:
		return daftardetailtipedokumen.DetailType{}, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca detail %q: %w", id, err)
	}

	businesses, err := r.businessesOf(ctx, key)
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}
	detail.Businesses = businesses
	return detail, nil
}

// InsertNew menerbitkan ID lalu menyisipkan barisnya beserta seluruh baris bisnisnya.
//
// Seluruhnya dalam SATU transaksi. Ini bukan kehati-hatian berlebihan: sebuah rincian
// dokumen yang tersimpan tanpa daftar bisnisnya berarti dokumen itu tidak diminta pada
// lini bisnis mana pun — kebalikan dari yang dimaksud petugas, dan tidak terlihat sebagai
// galat di layar mana pun. `D-68` menetapkan kepemilikan transaksi berpindah ke Go
// persis karena kelas kegagalan seperti ini, dan procedure lama memperagakan akibatnya:
// ia `COMMIT` sendiri lalu `ROLLBACK` sesudahnya, sehingga rollback-nya tidak memulihkan
// apa pun.
func (r *Repo) InsertNew(
	ctx context.Context,
	input daftardetailtipedokumen.Input,
	by daftardetailtipedokumen.Editor,
) (daftardetailtipedokumen.DetailType, error) {
	var saved daftardetailtipedokumen.DetailType

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		id, err := nextID(ctx, tx)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, getQuery("detail_insert"),
			id,
			input.DocumentTypeID,
			input.Detail,
			input.InsuredStatus,
			input.CauseOfLossID,
			input.CauseOfLossDescription,
			input.ObjectDocumentID,
			input.ObjectDocumentDescription,
			input.Risk,
			pegaTimestamp(by.At),
			by.Identity,
		)
		if err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: menyisipkan detail %q: %w", id, err)
		}

		if err := insertBusinesses(ctx, tx, id, input); err != nil {
			return err
		}

		saved = detailFrom(id, input)
		return nil
	})
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}

	// Baris dibaca ULANG lewat view, bukan disusun dari isian yang baru dikirim.
	//
	// Dua hal yang tidak diketahui pemanggil membuatnya perlu: `TYPE_DOCUMENT` hasil join
	// ke master tipe dokumen, dan nama bisnis pada setiap baris anaknya hasil join ke
	// POOLDATA.BUSINESS. Keduanya tidak ada di isian, dan mengarangnya berarti menampilkan
	// nama untuk kode yang mungkin tidak ada di master mana pun.
	return r.Get(ctx, saved.ID)
}

// Update mengganti isi satu rincian beserta SELURUH daftar bisnisnya.
//
// Penggantian daftar bisnis dilakukan dengan menghapus lalu menyisip ulang, di dalam
// transaksi yang sama dengan perubahan barisnya. Kegagalan di tengah karena itu tidak
// dapat meninggalkan rincian dokumen yang kehilangan seluruh lini bisnisnya.
func (r *Repo) Update(
	ctx context.Context,
	id string,
	input daftardetailtipedokumen.Input,
	by daftardetailtipedokumen.Editor,
) (daftardetailtipedokumen.DetailType, error) {
	key := strings.TrimSpace(id)

	err := r.inTransaction(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, getQuery("detail_update"),
			input.DocumentTypeID,
			input.Detail,
			input.InsuredStatus,
			input.CauseOfLossID,
			input.CauseOfLossDescription,
			input.ObjectDocumentID,
			input.ObjectDocumentDescription,
			input.Risk,
			pegaTimestamp(by.At),
			by.Identity,
			key,
		)
		if err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: memperbarui detail %q: %w", id, err)
		}

		// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak
		// ada berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan"
		// atas perubahan yang tidak pernah terjadi — lalu MENGHAPUS baris bisnis milik
		// baris yang tidak ada.
		affected, err := result.RowsAffected()
		if err == nil && affected == 0 {
			return daftardetailtipedokumen.ErrNotFound
		}

		if _, err := tx.ExecContext(ctx, getQuery("detail_business_delete_all"), key); err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: membuang bisnis detail %q: %w", id, err)
		}

		return insertBusinesses(ctx, tx, key, input)
	})
	if err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}

	// Alasannya sama dengan InsertNew: keterangannya milik master.
	return r.Get(ctx, key)
}

// CheckTable memastikan seluruh objek yang disentuh modul ini ada dan dapat dibaca akun
// aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
//
// Pemeriksaan BACA dan pemeriksaan TULIS terpisah, dan pemisahan itu yang membuat laporan
// mode periksa berguna: nama kolom tabel dasar adalah DUGAAN (lihat kepala berkas .sql),
// dan yang membuktikannya benar atau salah hanyalah pemeriksaan tulis.
//
// Yang TIDAK diperiksa di sini: urutan penerbit ID. Memeriksanya berarti MENGHABISKAN
// satu nomor — efek samping yang tidak pantas dimiliki mode periksa.
func (r *Repo) CheckTable(ctx context.Context) error {
	return r.checkAll(ctx, "detail_check_table")
}

// CheckWriteTable memastikan kedua TABEL DASAR modul ini punya bentuk yang dipakai jalur
// simpan — dan, untuk tabel anaknya, jalur baca satu baris.
//
// # Kenapa tabel anak ada di SINI, bukan di CheckTable
//
// Karena kueri daftar tidak menyentuhnya. `detail_list` hanya membaca view induk; tabel
// anaknya baru dipakai saat satu baris DIBUKA (`detail_business_list`) dan saat disimpan.
//
// Pemisahan ini yang membuat laporan mode periksa jujur: bila hanya pemeriksaan ini yang
// gagal, **daftarnya tetap dapat dimuat** dan yang terblokir hanya membuka baris serta
// menyimpan. Menggabungkannya akan melaporkan seluruh layar mati padahal tidak.
func (r *Repo) CheckWriteTable(ctx context.Context) error {
	return r.checkAll(ctx, "detail_write_check_table", "detail_business_check_table")
}

func (r *Repo) checkAll(ctx context.Context, names ...string) error {
	for _, name := range names {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, err)
		}
		closeErr := rows.Err()
		_ = rows.Close()
		if closeErr != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: objek modul tidak dapat dibaca (%s): %w", name, closeErr)
		}
	}
	return nil
}

// inTransaction menjalankan satu satuan kerja di dalam transaksi.
func (r *Repo) inTransaction(ctx context.Context, work func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("daftardetailtipedokumen/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if err := work(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("daftardetailtipedokumen/sqlstore: menutup transaksi: %w", err)
	}
	return nil
}

// businessesOf membaca aturan per lini bisnis milik satu rincian.
func (r *Repo) businessesOf(ctx context.Context, id string) ([]daftardetailtipedokumen.BusinessRule, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("detail_business_list"), id)
	if err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca bisnis detail %q: %w", id, err)
	}
	defer func() { _ = rows.Close() }()

	var result []daftardetailtipedokumen.BusinessRule
	for rows.Next() {
		// MIN_DOC dibaca sebagai TEKS, bukan angka, karena itulah tipenya: pada view
		// anak ia `VARCHAR2(10)` hasil `JSON_TABLE ... PATH '$.MIN_DOC'`. Membacanya
		// sebagai angka akan menggagalkan seluruh daftar karena satu baris warisan yang
		// isinya kosong atau bukan angka.
		var businessID, businessName, mandatory, minDocument sql.NullString
		if err := rows.Scan(&businessID, &businessName, &mandatory, &minDocument); err != nil {
			return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca baris bisnis: %w", err)
		}
		result = append(result, daftardetailtipedokumen.BusinessRule{
			BusinessID:   strings.TrimSpace(businessID.String),
			BusinessName: strings.TrimSpace(businessName.String),
			Mandatory:    daftardetailtipedokumen.MandatoryFrom(mandatory.String),
			MinDocument:  minimumFrom(minDocument.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: menelusuri bisnis detail %q: %w", id, err)
	}
	return result, nil
}

// insertBusinesses menyisipkan seluruh baris bisnis milik satu rincian.
func insertBusinesses(
	ctx context.Context,
	tx *sql.Tx,
	parentID string,
	input daftardetailtipedokumen.Input,
) error {
	for _, row := range input.Businesses {
		_, err := tx.ExecContext(ctx, getQuery("detail_business_insert"),
			parentID,
			row.BusinessID,
			daftardetailtipedokumen.MandatoryText(row.Mandatory),
			row.MinDocument,
		)
		if err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: menyisipkan bisnis detail %q: %w", parentID, err)
		}
	}
	return nil
}

// nextID mengambil kode situs dan nomor urut berikutnya, lalu menyusunnya menjadi ID.
//
// Keduanya dibaca DI DALAM transaksi yang sama dengan penyisipannya, meniru
// `Database/PEGA_LST_DET_TYPE_DOC.prc` yang melakukan keduanya dalam satu blok.
func nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	var site sql.NullString
	if err := tx.QueryRowContext(ctx, getQuery("detail_site")).Scan(&site); err != nil {
		return "", fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca kode situs: %w", err)
	}

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("detail_next_sequence")).Scan(&sequence); err != nil {
		return "", fmt.Errorf("daftardetailtipedokumen/sqlstore: mengambil nomor urut: %w", err)
	}
	return daftardetailtipedokumen.FormatID(site.String, sequence), nil
}

// detailFrom menyusun baris tersimpan dari isian yang dikirim.
//
// Hasilnya sengaja tidak dikembalikan langsung ke pemanggil — lihat InsertNew, yang
// membacanya ulang. Fungsi ini hanya membawa ID-nya keluar dari transaksi.
func detailFrom(id string, input daftardetailtipedokumen.Input) daftardetailtipedokumen.DetailType {
	return daftardetailtipedokumen.DetailType{
		ID:                        id,
		DocumentTypeID:            input.DocumentTypeID,
		Detail:                    input.Detail,
		InsuredStatus:             input.InsuredStatus,
		CauseOfLossID:             input.CauseOfLossID,
		CauseOfLossDescription:    input.CauseOfLossDescription,
		ObjectDocumentID:          input.ObjectDocumentID,
		ObjectDocumentDescription: input.ObjectDocumentDescription,
		Risk:                      input.Risk,
	}
}

// minimumFrom membaca MIN_DOC yang bertipe teks menjadi angka.
//
// Isi yang bukan angka — termasuk kosong dan NULL — dijawab NOL, bukan galat. Baris lama
// dapat memuat apa saja, dan menggagalkan seluruh daftar karena satu baris berisi nilai
// tak terduga jauh lebih buruk daripada menampilkannya sebagai nol.
//
// Perlakuannya sengaja sama dengan MandatoryFrom pada lapisan domain: longgar saat
// membaca, tegas saat menulis.
func minimumFrom(stored string) int {
	value, err := strconv.Atoi(strings.TrimSpace(stored))
	if err != nil {
		return 0
	}
	return value
}

// pegaTimestamp mengubah waktu menjadi bentuk TEKS yang dipakai kolom TGL_EDIT.
//
// # Kenapa teks, dan kenapa bentuk ini
//
// Kolomnya VARCHAR2, bukan DATE — terbaca dari katalog Oracle 2026-09-23. Isi baris yang
// sudah ada berbentuk:
//
//	20231030T075651.051 GMT
//
// yakni keluaran `@getCurrentTimeStamp()` Pega, yang dipakai
// `Activity/CNMInsertDetailTypeDocument_act-Act.xml` saat menyimpan. Menulis `time.Time`
// ke kolom itu akan menghasilkan bentuk yang BERBEDA dari seluruh baris lain — dan
// perbedaan itu tidak menimbulkan galat, hanya kolom yang tidak lagi dapat diurutkan
// maupun dibandingkan secara seragam.
//
// Selalu UTC, karena akhiran "GMT" pada baris lama menyatakan demikian. Menulis waktu
// lokal dengan akhiran GMT akan menggeser seluruh jejak tujuh jam (`R-12`).
//
// Akhiran " GMT" disambung sebagai teks biasa, tidak dimasukkan ke dalam layout, supaya
// tidak ada satu pun huruf di dalamnya yang tertukar dengan token format Go.
func pegaTimestamp(at time.Time) string {
	return at.UTC().Format("20060102T150405.000") + " GMT"
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan berbentuk sama
// tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(target ...any) error }

// scanDetail membaca satu baris hasil kueri menjadi DetailType.
//
// Seluruh kolom dibaca lewat tipe Null* lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena constraint tabelnya tidak diketahui (`R-08` — DDL-nya
// tidak ada di export).
//
// RISK dibaca sebagai TEKS, bukan angka, karena itulah bentuk kolomnya. Bahwa isinya
// dibaca sebagai bilangan oleh `Activity/SetTypePDFAdjustment-Act.xml` tidak mengubah
// cara ia disimpan — dan membacanya sebagai angka di sini akan menggagalkan seluruh
// daftar karena satu baris warisan yang isinya bukan angka.
func scanDetail(rows rowScanner) (daftardetailtipedokumen.DetailType, error) {
	var id, documentTypeID, documentTypeName, detail, insuredStatus sql.NullString
	var causeID, causeInfo, objectID, objectDesc, risk sql.NullString

	if err := rows.Scan(
		&id, &documentTypeID, &documentTypeName, &detail, &insuredStatus,
		&causeID, &causeInfo, &objectID, &objectDesc, &risk,
	); err != nil {
		return daftardetailtipedokumen.DetailType{}, err
	}
	return daftardetailtipedokumen.DetailType{
		ID:                        strings.TrimSpace(id.String),
		DocumentTypeID:            strings.TrimSpace(documentTypeID.String),
		DocumentTypeName:          strings.TrimSpace(documentTypeName.String),
		Detail:                    strings.TrimSpace(detail.String),
		InsuredStatus:             strings.TrimSpace(insuredStatus.String),
		CauseOfLossID:             strings.TrimSpace(causeID.String),
		CauseOfLossDescription:    strings.TrimSpace(causeInfo.String),
		ObjectDocumentID:          strings.TrimSpace(objectID.String),
		ObjectDocumentDescription: strings.TrimSpace(objectDesc.String),
		Risk:                      strings.TrimSpace(risk.String),
	}, nil
}

// ReferenceRepo membaca keempat master yang dirujuk layar ini sebagai daftar pilihan.
//
// Tipe tersendiri, bukan metode tambahan pada Repo, supaya batas kepemilikannya terbaca
// dari bentuknya: ia hanya punya operasi baca, dan tidak ada tempat untuk menambahkan
// operasi tulis tanpa sengaja (`P-1`).
type ReferenceRepo struct {
	db *sql.DB
}

// NewReferenceRepo membentuk pembaca keempat master rujukan.
func NewReferenceRepo(db *sql.DB) *ReferenceRepo { return &ReferenceRepo{db: db} }

// ListDocumentTypes membaca pilihan isian ID Tipe Dokumen.
func (r *ReferenceRepo) ListDocumentTypes(ctx context.Context) ([]daftardetailtipedokumen.DocumentTypeOption, error) {
	pairs, err := r.listPairs(ctx, "document_type_choice_list", "POOLDATA.V_LST_DOC_TYPE")
	if err != nil {
		return nil, err
	}
	result := make([]daftardetailtipedokumen.DocumentTypeOption, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, daftardetailtipedokumen.DocumentTypeOption{ID: pair.first, Name: pair.second})
	}
	return result, nil
}

// ListCausesOfLoss membaca pilihan isian Dokumen kolom ID.
func (r *ReferenceRepo) ListCausesOfLoss(ctx context.Context) ([]daftardetailtipedokumen.CauseOfLossOption, error) {
	pairs, err := r.listPairs(ctx, "cause_of_loss_choice_list", "POOLDATA.M_CAUSE_OF_LOSS")
	if err != nil {
		return nil, err
	}
	result := make([]daftardetailtipedokumen.CauseOfLossOption, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, daftardetailtipedokumen.CauseOfLossOption{ID: pair.first, Description: pair.second})
	}
	return result, nil
}

// ListObjectDocuments membaca pilihan isian Objek Dokumen.
func (r *ReferenceRepo) ListObjectDocuments(ctx context.Context) ([]daftardetailtipedokumen.ObjectDocumentOption, error) {
	pairs, err := r.listPairs(ctx, "object_document_choice_list", "POOLDATA.V_LST_DOC_OBJ")
	if err != nil {
		return nil, err
	}
	result := make([]daftardetailtipedokumen.ObjectDocumentOption, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, daftardetailtipedokumen.ObjectDocumentOption{ID: pair.first, Description: pair.second})
	}
	return result, nil
}

// ListBusinesses membaca pilihan isian ID Bisnis pada grid.
func (r *ReferenceRepo) ListBusinesses(ctx context.Context) ([]daftardetailtipedokumen.Business, error) {
	pairs, err := r.listPairs(ctx, "business_choice_list", "POOLDATA.BUSINESS")
	if err != nil {
		return nil, err
	}
	result := make([]daftardetailtipedokumen.Business, 0, len(pairs))
	for _, pair := range pairs {
		result = append(result, daftardetailtipedokumen.Business{ID: pair.first, Name: pair.second})
	}
	return result, nil
}

// CheckTable memastikan keempat master rujukan dapat dibaca akun aplikasi.
func (r *ReferenceRepo) CheckTable(ctx context.Context) error {
	for name, object := range map[string]string{
		"document_type_check_table":   "POOLDATA.V_LST_DOC_TYPE",
		"cause_of_loss_check_table":   "POOLDATA.M_CAUSE_OF_LOSS",
		"object_document_check_table": "POOLDATA.V_LST_DOC_OBJ",
		"business_check_table":        "POOLDATA.BUSINESS",
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: %s tidak dapat dibaca: %w", object, err)
		}
		closeErr := rows.Err()
		_ = rows.Close()
		if closeErr != nil {
			return fmt.Errorf("daftardetailtipedokumen/sqlstore: %s tidak dapat dibaca: %w", object, closeErr)
		}
	}
	return nil
}

// pair adalah satu baris berisi kode dan keterangannya.
//
// Keempat daftar pilihan berbentuk sama persis, sehingga satu pembaca melayani
// keempatnya. Menulis empat fungsi yang isinya identik akan membuat satu perbaikan harus
// diingat di empat tempat.
type pair struct {
	first  string
	second string
}

func (r *ReferenceRepo) listPairs(ctx context.Context, queryName, object string) ([]pair, error) {
	rows, err := r.db.QueryContext(ctx, getQuery(queryName))
	if err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca %s: %w", object, err)
	}
	defer func() { _ = rows.Close() }()

	var result []pair
	for rows.Next() {
		var first, second sql.NullString
		if err := rows.Scan(&first, &second); err != nil {
			return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: membaca baris %s: %w", object, err)
		}
		result = append(result, pair{
			first:  strings.TrimSpace(first.String),
			second: strings.TrimSpace(second.String),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("daftardetailtipedokumen/sqlstore: menelusuri %s: %w", object, err)
	}
	return result, nil
}

var (
	_ daftardetailtipedokumen.Repo          = (*Repo)(nil)
	_ daftardetailtipedokumen.ReferenceRepo = (*ReferenceRepo)(nil)
)
