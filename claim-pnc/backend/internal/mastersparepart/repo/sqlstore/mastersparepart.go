// Package sqlstore memenuhi seam mastersparepart.Store dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/mastersparepart"
	"claim-pnc/internal/platform/idformat"
	"claim-pnc/internal/platform/sqlfile"
	"claim-pnc/internal/platform/sqlkit"
	"claim-pnc/internal/platform/sqlvalue"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "mastersparepart/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string { return sqlfile.MustGet(query, "mastersparepart/sqlstore", name) }

// sequenceWidth adalah lebar nomor urut pada ID.
//
// Meniru `Database/PEGA_M_SPAREPART_HE.prc:21` persis:
//
//	id := id_site || lpad(to_Char(SPAREPART_HE_SEQ.nextval),10,'0');
//
// SEPULUH digit — bukan enam seperti Master Panel. Ketiga procedure keluarga alat berat
// ditulis dengan pola yang sama dan lebarnya tetap berbeda; menyeragamkannya akan
// menerbitkan ID yang tidak sebentuk dengan ID yang sudah ada.
//
// Pembentukannya dilakukan di Go dan bukan di SQL supaya kuerinya tetap portabel — LPAD ada
// di kedua basis data, tetapi menyusun kunci di dalam kueri berarti bentuk kuncinya
// tersebar ke berkas .sql dan ke sini sekaligus.
const sequenceWidth = 10

// approvedLookup adalah nilai APPROVAL yang dipakai kedua kueri acuan.
//
// '1', bukan '0'. Itu yang dilakukan `Activity/BrowseTipeKategoriPart-Act.xml`, satu-satunya
// activity yang dirujuk ketiga section tab Master Sparepart; lihat catatan pada
// mastersparepart/lookup.go.
const approvedLookup = string(mastersparepart.StatusApproved)

// Repo membaca dan menulis POOLDATA.SPAREPART_HE, dan HANYA MEMBACA kedua tabel acuannya.
//
// Satu struct memenuhi ketiga seam — mastersparepart.Repo, LookupRepo, dan IDSource —
// karena ketiganya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel
// — lihat berkas .sql untuk alasannya.
func (r *Repo) List(
	ctx context.Context,
	filter mastersparepart.Filter,
) ([]mastersparepart.Sparepart, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword != "" {
		// Pola yang sama dikirim tiga kali: satu untuk nama, satu untuk nomor, satu untuk
		// kode. Lihat catatan pada sparepart_list_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("sparepart_list_search"),
			string(filter.Status), pattern, pattern, pattern)
	} else {
		rows, err = r.db.QueryContext(ctx, getQuery("sparepart_list"), string(filter.Status))
	}
	if err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastersparepart.Sparepart
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("mastersparepart/sqlstore: membaca baris daftar: %w", err)
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get membaca satu baris berdasarkan ID-nya.
func (r *Repo) Get(ctx context.Context, id string) (mastersparepart.Sparepart, error) {
	row := r.db.QueryRowContext(ctx, getQuery("sparepart_get"), strings.TrimSpace(id))

	s, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
	}
	if err != nil {
		return mastersparepart.Sparepart{}, fmt.Errorf(
			"mastersparepart/sqlstore: membaca %q: %w", id, err)
	}
	return s, nil
}

// FindByName mencari baris menurut NAMA_SPART-nya.
func (r *Repo) FindByName(
	ctx context.Context,
	name string,
) (mastersparepart.Sparepart, error) {
	return r.findBy(ctx, "sparepart_find_by_name", "nama", name)
}

// FindByNumber mencari baris menurut NO_SPART-nya.
func (r *Repo) FindByNumber(
	ctx context.Context,
	number string,
) (mastersparepart.Sparepart, error) {
	return r.findBy(ctx, "sparepart_find_by_number", "nomor", number)
}

// FindByCode mencari baris menurut KODE_SPART-nya.
func (r *Repo) FindByCode(
	ctx context.Context,
	code string,
) (mastersparepart.Sparepart, error) {
	return r.findBy(ctx, "sparepart_find_by_code", "kode", code)
}

// findBy menjalankan salah satu dari ketiga pencarian kunci alami.
//
// Ketiganya satu fungsi karena ketiganya berperilaku sama persis — hanya kuerinya yang
// berbeda. Menyalinnya tiga kali berarti tiga tempat yang dapat berbeda soal pemangkasan
// dan huruf besar.
func (r *Repo) findBy(
	ctx context.Context,
	queryName, label, value string,
) (mastersparepart.Sparepart, error) {
	clean := strings.ToUpper(strings.TrimSpace(value))
	if clean == "" {
		return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
	}

	row := r.db.QueryRowContext(ctx, getQuery(queryName), clean)

	s, err := scanRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return mastersparepart.Sparepart{}, mastersparepart.ErrNotFound
	}
	if err != nil {
		return mastersparepart.Sparepart{}, fmt.Errorf(
			"mastersparepart/sqlstore: mencari %s %q: %w", label, value, err)
	}
	return s, nil
}

// ListCategories membaca kategori yang sudah disetujui.
func (r *Repo) ListCategories(ctx context.Context) ([]mastersparepart.Category, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("sparepart_category_list"), approvedLookup)
	if err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: membaca kategori: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastersparepart.Category, 0, 64)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastersparepart/sqlstore: membaca baris kategori: %w", err)
		}
		result = append(result, mastersparepart.Category{
			ID:   strings.TrimSpace(id.String),
			Name: strings.TrimSpace(name.String),
		})
		if len(result) >= mastersparepart.MaxLookupRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: menelusuri kategori: %w", err)
	}
	return result, nil
}

// ListTypes membaca tipe yang sudah disetujui, beserta kategori induknya.
func (r *Repo) ListTypes(ctx context.Context) ([]mastersparepart.PartType, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("sparepart_type_list"), approvedLookup)
	if err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: membaca tipe: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastersparepart.PartType, 0, 128)
	for rows.Next() {
		var id, name, category sql.NullString
		if err := rows.Scan(&id, &name, &category); err != nil {
			return nil, fmt.Errorf("mastersparepart/sqlstore: membaca baris tipe: %w", err)
		}
		result = append(result, mastersparepart.PartType{
			ID:         strings.TrimSpace(id.String),
			Name:       strings.TrimSpace(name.String),
			CategoryID: strings.TrimSpace(category.String),
		})
		if len(result) >= mastersparepart.MaxLookupRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastersparepart/sqlstore: menelusuri tipe: %w", err)
	}
	return result, nil
}

// Insert menolak ketiga kunci alami yang sudah dipakai, lalu menyisipkan barisnya.
//
// Keduanya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi,
// dan alasannya satu: pemeriksaan dan penyisipan bukan dua perkara melainkan satu.
// Memisahkannya membuka kembali lubang balapan yang justru sedang dipersempit.
func (r *Repo) Insert(ctx context.Context, s mastersparepart.Sparepart) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa ini,
	// satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan kunci
	// baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if err := lockedKeys(ctx, tx, s); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, getQuery("sparepart_insert"), insertArguments(s)...); err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: menyisipkan %q: %w", s.ID, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: menutup transaksi sisip: %w", err)
	}
	return nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Pemeriksaan bentrok kunci TIDAK dilakukan di sini, dan itu mengikuti sistem lama:
// `ValidateMasterSparepart` dipanggil dari layar, bukan dari jalur penyimpanan. Lapisan
// aplikasi yang memeriksanya, dan ia mengecualikan baris itu sendiri.
func (r *Repo) Update(ctx context.Context, s mastersparepart.Sparepart) error {
	result, err := r.db.ExecContext(ctx, getQuery("sparepart_update"), updateArguments(s)...)
	if err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: memperbarui %q: %w", s.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh satu
	// baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan mengatakan
	// "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return mastersparepart.ErrNotFound
	}
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Satu pernyataan per baris, bukan satu pernyataan dengan daftar kunci yang panjangnya
// berubah-ubah — lihat `sparepart_set_status` pada berkas .sql untuk alasannya.
//
// Transaksinya melingkupi seluruh baris supaya persetujuan borongan tidak pernah
// setengah jalan: bila baris kelima gagal, keempat yang sebelumnya ikut dibatalkan.
// Sistem lama tidak menjamin itu — `SetApprovalAllMaster` menjalankan satu RDB-List per
// baris tanpa transaksi yang melingkupinya.
func (r *Repo) SetStatus(ctx context.Context, id []string, status mastersparepart.ApprovalStatus) (int, error) {
	return sqlkit.SetEach(ctx, r.db, getQuery("sparepart_set_status"), string(status), id, "mastersparepart/sqlstore")
}

// NextID menerbitkan ID berikutnya.
//
// Bentuknya meniru `Database/PEGA_M_SPAREPART_HE.prc:12,21` persis: kode situs ditambah
// nomor urut SEPULUH digit bertambal nol.
//
// Kedua kueri dijalankan di dalam SATU transaksi. Bukan demi keatomikan — sequence tidak
// dapat dibatalkan — melainkan supaya keduanya pasti dilayani koneksi yang sama; kode
// situs dan sequence yang berasal dari dua koneksi berbeda pada pool yang sama tetap
// benar, tetapi jaminannya tidak berasal dari mana pun selain kebetulan.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	return idformat.Next(ctx, r.db, getQuery("sparepart_site"), getQuery("sparepart_next_sequence"), "mastersparepart/sqlstore", sequenceWidth)
}

// CheckTable memastikan POOLDATA.SPAREPART_HE ada dan kedua puluh empat kolomnya dapat
// dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	return r.checkReadable(ctx, "sparepart_check_table", "POOLDATA.SPAREPART_HE")
}

// CheckCategoryTable memastikan tabel acuan Kategori dapat dibaca.
func (r *Repo) CheckCategoryTable(ctx context.Context) error {
	return r.checkReadable(ctx, "sparepart_check_category_table",
		"POOLDATA.GCNM_M_SPAREPART_CATEGORY")
}

// CheckTypeTable memastikan tabel acuan Tipe dapat dibaca.
func (r *Repo) CheckTypeTable(ctx context.Context) error {
	return r.checkReadable(ctx, "sparepart_check_type_table", "POOLDATA.GCNM_M_SPAREPART_TYPE")
}

// CheckJSONMirror memastikan POOLDATA.M_SPAREPART_HE_BU — tabel JSON milik Pega — dapat
// dibaca.
//
// Dipakai mode periksa saja. Lihat banner berkas .sql: perbandingan jumlah barisnya dengan
// SPAREPART_HE adalah cara termurah mengetahui apakah keduanya satu sumber.
func (r *Repo) CheckJSONMirror(ctx context.Context) error {
	return r.checkReadable(ctx, "sparepart_check_json_mirror", "POOLDATA.M_SPAREPART_HE_BU")
}

func (r *Repo) checkReadable(ctx context.Context, name, table string) error {
	rows, err := r.db.QueryContext(ctx, getQuery(name))
	if err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: %s tidak dapat dibaca: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// CountByStatus menghitung baris pada satu status persetujuan.
func (r *Repo) CountByStatus(
	ctx context.Context,
	status mastersparepart.ApprovalStatus,
) (int, error) {
	return r.count(ctx, "sparepart_count_by_status", string(status))
}

// CountAll menghitung seluruh baris SPAREPART_HE.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "sparepart_count_all")
}

// CountJSONMirror menghitung baris POOLDATA.M_SPAREPART_HE_BU.
func (r *Repo) CountJSONMirror(ctx context.Context) (int, error) {
	return r.count(ctx, "sparepart_count_json_mirror")
}

// CountOrphanCategory menghitung baris yang kategorinya tidak ada di tabel acuan.
func (r *Repo) CountOrphanCategory(ctx context.Context) (int, error) {
	return r.count(ctx, "sparepart_count_orphan_category")
}

// CountOrphanType menghitung baris yang tipenya tidak ada di tabel acuan.
func (r *Repo) CountOrphanType(ctx context.Context) (int, error) {
	return r.count(ctx, "sparepart_count_orphan_type")
}

func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastersparepart/sqlstore: menjalankan %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyusun pola LIKE dari sebuah kata kunci.
//
// Tanda persen, garis bawah, dan backslash pada kata kunci DILOLOSKAN lebih dulu. Tanpa
// itu, pengguna yang mengetik "%" menarik seluruh tabel dan yang mengetik "_" mencocoki
// karakter apa pun — bukan celah keamanan karena nilainya tetap terikat sebagai
// parameter, tetapi hasil yang tidak dapat dijelaskan kepada yang mengetiknya.
//
// `ESCAPE '\'` disebut eksplisit di kuerinya karena Oracle tidak punya karakter pelolos
// bawaan pada LIKE.
func likePattern(keyword string) string { return sqlvalue.Like(keyword) }

type scanner = sqlkit.Scanner

// scanRow membaca satu baris master sparepart.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang
// diketahui (R-08).
//
// TGL_UPDATE_HARGA dibaca lewat sql.NullTime — ia satu-satunya kolom yang diperlakukan
// sebagai waktu; lihat banner berkas .sql untuk asumsi tipenya. PROD_DATE TIDAK, dan itu
// disengaja: layar lama merendernya sebagai teks biasa.
//
// Urutan kolomnya mengikuti berkas .sql, dan kedua puluh empat kolom dibaca pada urutan yang
// sama oleh sparepart_list, sparepart_list_search, sparepart_get, dan ketiga kueri pencarian
// kunci. Itu yang membuat satu fungsi cukup untuk keenamnya — dan yang membuat uji urutan
// kolom pada query_test.go layak ada.
func scanRow(p scanner) (mastersparepart.Sparepart, error) {
	var (
		id, name, number, code, price             sql.NullString
		category, partType, weight, length, width sql.NullString
		height, minStock, maxStock, orderQuantity sql.NullString
		productionDate, substitute, kind, unit    sql.NullString
		activeStatus, partStatus, updatedBy       sql.NullString
		documentID, status                        sql.NullString
		priceUpdatedAt                            sql.NullTime
	)
	if err := p.Scan(
		&id, &name, &number, &code, &price,
		&category, &partType, &weight, &length, &width,
		&height, &minStock, &maxStock, &orderQuantity, &productionDate,
		&substitute, &kind, &unit, &activeStatus, &partStatus,
		&updatedBy, &priceUpdatedAt, &documentID, &status,
	); err != nil {
		return mastersparepart.Sparepart{}, err
	}

	trim := strings.TrimSpace
	result := mastersparepart.Sparepart{
		ID:             trim(id.String),
		Name:           trim(name.String),
		Number:         trim(number.String),
		Code:           trim(code.String),
		SellingPrice:   trim(price.String),
		CategoryID:     trim(category.String),
		TypeID:         trim(partType.String),
		Weight:         trim(weight.String),
		Length:         trim(length.String),
		Width:          trim(width.String),
		Height:         trim(height.String),
		MinStock:       trim(minStock.String),
		MaxStock:       trim(maxStock.String),
		OrderQuantity:  trim(orderQuantity.String),
		ProductionDate: trim(productionDate.String),
		Substitute:     trim(substitute.String),
		Kind:           trim(kind.String),
		Unit:           trim(unit.String),
		ActiveStatus:   trim(activeStatus.String),
		PartStatus:     trim(partStatus.String),
		UpdatedBy:      trim(updatedBy.String),
		DocumentID:     trim(documentID.String),
		Status:         mastersparepart.ApprovalStatus(trim(status.String)),
	}

	if priceUpdatedAt.Valid {
		// Disalin ke variabel lokal lebih dulu: mengambil alamat field sql.NullTime akan
		// menautkan hasilnya ke variabel yang dipakai ulang pada iterasi berikutnya, sehingga
		// seluruh baris berakhir menunjuk waktu yang sama.
		at := priceUpdatedAt.Time.UTC()
		result.PriceUpdatedAt = &at
	}

	return result, nil
}

// insertArguments menyusun kedua puluh empat nilai sparepart_insert pada urutan kolomnya.
//
// Urutannya WAJIB sama dengan daftar kolom pada kueri. Ia dipisahkan menjadi fungsi
// tersendiri supaya urutan itu dapat diuji terhadap berkas .sql, bukan hanya dipercaya.
func insertArguments(s mastersparepart.Sparepart) []any {
	return []any{
		s.ID, s.Name, s.Number, s.Code, s.SellingPrice,
		s.CategoryID, s.TypeID, s.Weight, s.Length, s.Width,
		s.Height, s.MinStock, s.MaxStock, s.OrderQuantity, s.ProductionDate,
		s.Substitute, s.Kind, s.Unit, s.ActiveStatus, s.PartStatus,
		s.UpdatedBy, priceTime(s.PriceUpdatedAt), s.DocumentID, string(s.Status),
	}
}

// updateArguments menyusun nilai sparepart_update; ID berada di posisi TERAKHIR karena ia
// penyaring WHERE, bukan kolom yang ditulis.
func updateArguments(s mastersparepart.Sparepart) []any {
	return []any{
		s.Name, s.Number, s.Code, s.SellingPrice, s.CategoryID,
		s.TypeID, s.Weight, s.Length, s.Width, s.Height,
		s.MinStock, s.MaxStock, s.OrderQuantity, s.ProductionDate, s.Substitute,
		s.Kind, s.Unit, s.ActiveStatus, s.PartStatus, s.UpdatedBy,
		priceTime(s.PriceUpdatedAt), s.DocumentID, string(s.Status),
		strings.TrimSpace(s.ID),
	}
}

// priceTime mengubah penunjuk waktu menjadi nilai yang dapat diikat driver.
//
// nil menjadi NULL, bukan waktu nol. Tanpa ini, baris yang harganya belum pernah diisi akan
// tersimpan dengan tanggal tahun 1 — nilai yang tampak sah di layar dan tidak dapat
// dibedakan dari tanggal yang sungguh-sungguh diisi.
func priceTime(at *time.Time) any {
	if at == nil {
		return nil
	}
	return at.UTC()
}

// lockedKeys menjalankan sparepart_lock_by_keys dan melaporkan kunci mana yang bentrok.
//
// Ia mengembalikan ValidationError berisi SELURUH kunci yang bentrok, bukan yang pertama
// saja — petugas yang salah pada nomor DAN kode tahu keduanya dalam satu kali simpan.
//
// Ketiga galat sentinel ErrNameTaken, ErrNumberTaken, dan ErrCodeTaken TIDAK dipakai di
// sini, dan keberadaannya bukan sia-sia: adapter memori memakainya, dan transport
// memetakannya ke 409. Yang dipakai di jalur ini adalah bentuk yang dapat memuat lebih dari
// satu pelanggaran sekaligus.
func lockedKeys(ctx context.Context, tx *sql.Tx, s mastersparepart.Sparepart) error {
	upper := func(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }

	name, number, code := upper(s.Name), upper(s.Number), upper(s.Code)
	if name == "" && number == "" && code == "" {
		return nil
	}

	rows, err := tx.QueryContext(ctx, getQuery("sparepart_lock_by_keys"), name, number, code)
	if err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: memeriksa kunci %q: %w", s.ID, err)
	}
	defer func() { _ = rows.Close() }()

	var violation []mastersparepart.Violation
	seen := map[string]bool{}

	for rows.Next() {
		var otherID, otherName, otherNumber, otherCode sql.NullString
		if err := rows.Scan(&otherID, &otherName, &otherNumber, &otherCode); err != nil {
			return fmt.Errorf("mastersparepart/sqlstore: membaca hasil pemeriksaan: %w", err)
		}

		for _, clash := range []struct{ field, label, mine, theirs string }{
			{"nomor_sparepart", "Nomor", number, upper(otherNumber.String)},
			{"nama_sparepart", "Nama", name, upper(otherName.String)},
			{"kode_sparepart", "Kode", code, upper(otherCode.String)},
		} {
			if clash.mine == "" || clash.mine != clash.theirs || seen[clash.field] {
				continue
			}
			seen[clash.field] = true
			violation = append(violation, mastersparepart.Violation{
				Field: clash.field,
				Message: clash.label + " tersebut telah digunakan sparepart " +
					strings.TrimSpace(otherID.String) + ". Silakan ganti dengan yang lain.",
			})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("mastersparepart/sqlstore: menelusuri pemeriksaan kunci: %w", err)
	}

	if len(violation) > 0 {
		return &mastersparepart.ValidationError{Violation: violation}
	}
	return nil
}

var _ mastersparepart.Store = (*Repo)(nil)
