// Package sqlstore adalah pengisi seam mastertipesparepart.Store terhadap basis data.
//
// Satu instans selalu terikat pada SATU koneksi entitas; pemisahan antarentitas ada di
// tingkat koneksi, bukan di tingkat kueri (ADR-0030 Opsi 1).
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/mastertipesparepart"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// approvedLookup adalah nilai APPROVAL yang dipakai kueri acuan kategori.
//
// Ia REKONSTRUKSI, bukan pembacaan — rule yang mengisi dropdown layar ini tidak ada di
// export. Rantai buktinya ada pada mastertipesparepart.LookupRepo.ListCategories.
//
// Diturunkan dari StatusApproved alih-alih ditulis "1" begitu saja, supaya keduanya tidak
// dapat berbeda tanpa ada yang menyadarinya.
const approvedLookup = string(mastertipesparepart.StatusApproved)

// Repo membaca dan menulis POOLDATA.GCNM_M_SPAREPART_TYPE, dan MEMBACA SAJA
// POOLDATA.GCNM_M_SPAREPART_CATEGORY.
//
// Satu struct memenuhi ketiga seam — mastertipesparepart.Repo, LookupRepo, dan IDSource —
// karena ketiganya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring, beserta nama kategori induknya.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel —
// lihat berkas .sql untuk alasannya.
func (r *Repo) List(
	ctx context.Context,
	filter mastertipesparepart.Filter,
) ([]mastertipesparepart.PartType, error) {
	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("type_list"), string(filter.Status))
	} else {
		// DUA argumen untuk kueri yang menyebut `:2` dua kali — sekali pada nama tipe, sekali
		// pada nama kategori. Penyebutan ulang satu parameter posisional diterima kedua
		// driver, dan pola yang sama sudah dipakai `bengkel_list_search`.
		//
		// Tanda persen dipasang di sini, bukan di dalam teks SQL: nilai yang dikirim ke basis
		// data tetap lewat parameter binding, dan pola pencariannya tetap terbaca di satu
		// tempat.
		rows, err = r.db.QueryContext(ctx, getQuery("type_list_search"),
			string(filter.Status), "%"+escapeLike(keyword)+"%")
	}
	if err != nil {
		return nil, fmt.Errorf("mastertipesparepart/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []mastertipesparepart.PartType
	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastertipesparepart/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get mengembalikan satu baris berdasarkan kuncinya.
func (r *Repo) Get(
	ctx context.Context,
	id string,
) (mastertipesparepart.PartType, error) {
	row := r.db.QueryRowContext(ctx, getQuery("type_get"), strings.TrimSpace(id))

	found, err := scanRow(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return mastertipesparepart.PartType{}, mastertipesparepart.ErrNotFound
	case err != nil:
		return mastertipesparepart.PartType{}, err
	}
	return found, nil
}

// FindByName mencari baris menurut namanya.
//
// Pembandingnya di-uppercase DI SINI dan bukan di dalam kueri untuk sisi parameternya:
// kueri sudah memakai `UPPER(TRIM(PART_SECTION_NAME))` di sisi kolom, dan menaruh UPPER
// pada kedua sisi di dalam teks SQL akan memaksa basis data mengubah nilai yang sudah dapat
// diubah sekali di sini.
//
// Kueri yang dipakainya TIDAK ber-JOIN, sehingga CategoryName pada hasilnya selalu kosong.
// Itu tidak berakibat apa pun: satu-satunya pemakai hasil ini adalah pemeriksaan keunikan,
// yang hanya melihat kuncinya.
func (r *Repo) FindByName(
	ctx context.Context,
	name string,
) (mastertipesparepart.PartType, error) {
	row := r.db.QueryRowContext(ctx, getQuery("type_find_by_name"),
		strings.ToUpper(strings.TrimSpace(name)))

	found, err := scanNameRow(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return mastertipesparepart.PartType{}, mastertipesparepart.ErrNotFound
	case err != nil:
		return mastertipesparepart.PartType{}, err
	}
	return found, nil
}

// ListCategories membaca kategori yang sudah disetujui, untuk dropdown Kategori.
//
// Tabel yang dibacanya milik modul masterkategorisparepart; modul ini tidak pernah
// menulisnya (P-1).
func (r *Repo) ListCategories(
	ctx context.Context,
) ([]mastertipesparepart.Category, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("type_category_list"), approvedLookup)
	if err != nil {
		return nil, fmt.Errorf("mastertipesparepart/sqlstore: membaca kategori: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]mastertipesparepart.Category, 0, 64)
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("mastertipesparepart/sqlstore: membaca baris kategori: %w", err)
		}
		result = append(result, mastertipesparepart.Category{
			ID:   tidyNumber(strings.TrimSpace(id.String)),
			Name: strings.TrimSpace(name.String),
		})
		if len(result) >= mastertipesparepart.MaxLookupRows {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mastertipesparepart/sqlstore: menelusuri kategori: %w", err)
	}
	return result, nil
}

// Insert menerbitkan ID, memeriksa keunikan nama, lalu menyisipkan — seluruhnya dalam SATU
// transaksi yang menahan kunci tabel.
//
// # Urutan langkahnya, dan kenapa persis begitu
//
//  1. LOCK TABLE ... IN EXCLUSIVE MODE   tidak ada penulis lain sampai transaksi selesai
//  2. SELECT ... UPPER(TRIM(name))       nama ganda ditolak sebelum ID terpakai
//  3. SELECT COALESCE(MAX(id),0)+1       terbitkan kunci
//  4. INSERT                             sisipkan
//
// Pemeriksaan nama berada SETELAH penguncian dan SEBELUM penerbitan ID. Setelah, supaya
// tidak ada penambahan lain yang menyelinap di antara pemeriksaan dan penyisipan; sebelum,
// supaya nomor urut tidak terpakai oleh percobaan yang memang akan ditolak — ID yang
// terlewat tidak merusak apa pun, tetapi deret yang berlubang membuat orang mencari baris
// yang tidak pernah ada.
//
// Keberadaan KATEGORI tidak diperiksa di sini melainkan di lapisan aplikasi, dan itu
// disengaja: pemeriksaannya membaca tabel milik modul lain, dan menariknya ke dalam
// transaksi ini akan memperluas cakupan kunci tanpa menutup keadaan apa pun. Alasannya
// lengkap ada pada `type_lock_table` di berkas .sql.
//
// ID pada argumen DIABAIKAN dengan sengaja; lihat mastertipesparepart.Repo.Insert.
func (r *Repo) Insert(
	ctx context.Context,
	t mastertipesparepart.PartType,
) (mastertipesparepart.PartType, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa ini,
	// satu jalur galat yang terlewat meninggalkan transaksi menggantung — dan karena
	// transaksi ini memegang kunci TABEL, yang tertahan bukan satu baris melainkan seluruh
	// penambahan tipe sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, getQuery("type_lock_table")); err != nil {
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: mengunci tabel tipe: %w", err)
	}

	if err := rejectTakenName(ctx, tx, t.Name); err != nil {
		return mastertipesparepart.PartType{}, err
	}

	id, err := nextID(ctx, tx)
	if err != nil {
		return mastertipesparepart.PartType{}, err
	}

	fresh := mastertipesparepart.PartType{
		ID:         id,
		Name:       t.Name,
		CategoryID: t.CategoryID,
		Status:     t.Status,
	}
	if _, err := tx.ExecContext(ctx, getQuery("type_insert"),
		fresh.ID, fresh.Name, fresh.CategoryID, string(fresh.Status)); err != nil {
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: menyisipkan %q: %w", fresh.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: menutup transaksi sisip: %w", err)
	}

	// CategoryName TIDAK diisi di sini: ia milik tabel kategori, dan membacanya menuntut
	// perjalanan kedua yang hanya untuk memperindah satu jawaban. Lapisan aplikasi yang
	// mengisinya dari daftar kategori yang memang sudah dibacanya untuk memeriksa
	// keberadaannya — lihat usecase.Service.Create.
	return fresh, nil
}

// rejectTakenName menolak nama yang sudah dipakai baris mana pun.
//
// Dijalankan DI DALAM transaksi penambahan, bukan lewat FindByName di luar: pemeriksaan
// yang berjalan pada koneksi lain tidak melihat kunci tabel yang sedang dipegang, sehingga
// hasilnya dapat basi sebelum penyisipannya berjalan.
//
// TIDAK menyaring APPROVAL dan TIDAK menyaring kategori, meniru `ValidationSparepartType`
// apa adanya. Lihat mastertipesparepart.ErrNameTaken.
func rejectTakenName(ctx context.Context, tx *sql.Tx, name string) error {
	row := tx.QueryRowContext(ctx, getQuery("type_find_by_name"),
		strings.ToUpper(strings.TrimSpace(name)))

	_, err := scanNameRow(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	default:
		return mastertipesparepart.ErrNameTaken
	}
}

// nextID membaca nomor urut berikutnya di dalam transaksi yang sudah memegang kunci tabel.
//
// Hasilnya dibaca sebagai int64 lalu diubah menjadi teks desimal, bukan dibaca langsung
// sebagai string. Alasannya: driver Oracle mengembalikan NUMBER sebagai bentuk yang
// bergantung pada konfigurasinya — "8", "8.0", atau bahkan notasi ilmiah pada nilai besar —
// dan ID yang tersimpan sebagai "8.0" tidak akan pernah cocok dengan
// `SPAREPART_HE.TIPE_SPART` yang berisi "8".
func nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	var next int64
	if err := tx.QueryRowContext(ctx, getQuery("type_next_id")).Scan(&next); err != nil {
		return "", fmt.Errorf(
			"mastertipesparepart/sqlstore: menerbitkan ID tipe: %w", err)
	}
	if next <= 0 {
		// COALESCE(MAX(...),0)+1 tidak dapat menghasilkan nilai ini pada tabel yang waras.
		// Bila ia terjadi, ada ID negatif di tabelnya — dan menyisipkan baris di atasnya akan
		// menimpa deret yang sudah ada.
		return "", fmt.Errorf(
			"mastertipesparepart/sqlstore: ID tipe berikutnya tidak masuk akal: %d", next)
	}
	return strconv.FormatInt(next, 10), nil
}

// NextID menerbitkan ID berikutnya di luar transaksi penambahan.
//
// Ia TIDAK dipakai jalur simpan — Insert menerbitkannya sendiri di dalam transaksinya, dan
// itulah satu-satunya cara nilainya dijamin masih benar saat dipakai.
//
// Yang memakainya adalah `claimpnc -periksa`: ia membuktikan penerbitan kunci bekerja
// terhadap basis data nyata TANPA menyisipkan satu baris pun. Tanpa itu, kegagalan
// penerbitan ID baru terlihat saat petugas menekan Simpan untuk pertama kalinya.
//
// Nilai yang dikembalikannya karena itu bersifat SEMENTARA dan tidak boleh dipakai
// menyisipkan apa pun.
func (r *Repo) NextID(ctx context.Context) (string, error) {
	var next int64
	if err := r.db.QueryRowContext(ctx, getQuery("type_next_id")).Scan(&next); err != nil {
		return "", fmt.Errorf(
			"mastertipesparepart/sqlstore: menerbitkan ID tipe: %w", err)
	}
	return strconv.FormatInt(next, 10), nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Pemeriksaan bentrok nama TIDAK dilakukan di sini, dan itu mengikuti sistem lama:
// `ValidateMasterTipeSparepart` dipanggil dari layar, bukan dari jalur penyimpanan.
// Lapisan aplikasi yang memeriksanya, dan ia mengecualikan baris itu sendiri.
//
// CategoryName TIDAK ditulis: ia milik tabel kategori.
func (r *Repo) Update(ctx context.Context, t mastertipesparepart.PartType) error {
	result, err := r.db.ExecContext(ctx, getQuery("type_update"),
		t.Name, t.CategoryID, string(t.Status), strings.TrimSpace(t.ID))
	if err != nil {
		return fmt.Errorf("mastertipesparepart/sqlstore: memperbarui %q: %w", t.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh satu
	// baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan mengatakan
	// "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return mastertipesparepart.ErrNotFound
	}
	return nil
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Bentuk pernyataannya DIREKONSTRUKSI; lihat catatan pada `type_set_status` di berkas .sql
// dan pada mastertipesparepart.Repo.SetStatus.
func (r *Repo) SetStatus(
	ctx context.Context,
	id []string,
	status mastertipesparepart.ApprovalStatus,
) (int, error) {
	if len(id) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf(
			"mastertipesparepart/sqlstore: memulai transaksi keputusan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, one := range id {
		result, err := tx.ExecContext(ctx, getQuery("type_set_status"),
			string(status), strings.TrimSpace(one))
		if err != nil {
			return 0, fmt.Errorf(
				"mastertipesparepart/sqlstore: menetapkan status %q: %w", one, err)
		}
		// Driver yang tidak mendukung RowsAffected membuat pencacahnya tidak dapat
		// diandalkan. Barisnya tetap dianggap berubah: pernyataannya sudah berhasil, dan
		// melaporkan nol akan menampilkan kegagalan palsu.
		affected, err := result.RowsAffected()
		if err != nil || affected > 0 {
			changed++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf(
			"mastertipesparepart/sqlstore: menutup transaksi keputusan: %w", err)
	}
	return changed, nil
}

// CountByStatus mencacah baris satu status. Dipakai `claimpnc -periksa`.
func (r *Repo) CountByStatus(
	ctx context.Context,
	status mastertipesparepart.ApprovalStatus,
) (int, error) {
	return r.count(ctx, "type_count_by_status", string(status))
}

// CountAll mencacah seluruh baris tanpa memandang status. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "type_count_all")
}

// CountUnknownStatus mencacah baris yang APPROVAL-nya di luar ketiga sandi yang dikenal.
//
// Baris seperti itu tidak muncul di satu pun tab; lihat berkas .sql.
func (r *Repo) CountUnknownStatus(ctx context.Context) (int, error) {
	return r.count(ctx, "type_count_unknown_status")
}

// CountDuplicateName mencacah NAMA yang dipakai lebih dari satu baris.
func (r *Repo) CountDuplicateName(ctx context.Context) (int, error) {
	return r.count(ctx, "type_count_duplicate_name")
}

// CountOrphanCategory mencacah tipe yang menunjuk kategori yang tidak ada.
//
// Inilah baris yang di sistem lama HILANG dari layar karena inner join-nya. Lihat berkas
// .sql.
func (r *Repo) CountOrphanCategory(ctx context.Context) (int, error) {
	return r.count(ctx, "type_count_orphan_category")
}

// CountOrphanSparepart mencacah sparepart yang menunjuk tipe yang tidak ada.
func (r *Repo) CountOrphanSparepart(ctx context.Context) (int, error) {
	return r.count(ctx, "type_count_orphan_sparepart")
}

// CheckTable membuktikan tabel beserta keempat kolomnya ada dan dapat dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("type_check_table"))
	if err != nil {
		return fmt.Errorf("mastertipesparepart/sqlstore: memeriksa tabel tipe: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("mastertipesparepart/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun
// di pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat
// yang dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris LIMA kolom menjadi PartType.
//
// Urutan kolomnya mengikuti berkas .sql, dan ketiga kueri ber-JOIN — type_list,
// type_list_search, dan type_get — membacanya pada urutan yang sama. Itu yang membuat satu
// fungsi cukup untuk ketiganya, dan yang membuat uji urutan kolom pada query_test.go layak
// ada.
//
// # Seluruh kolomnya dibaca sebagai teks yang boleh NULL
//
// Termasuk kedua kunci, yang di basis data bertipe angka. Membacanya sebagai int64 akan
// menolak baris yang kuncinya NULL — keadaan yang tidak seharusnya ada tetapi tidak dijaga
// constraint apa pun (R-08) — dan menggagalkan SELURUH daftar karena satu baris rusak.
//
// PART_CATEGORY_NAME memang BOLEH NULL, dan itu bukan kerusakan: kueri memakai LEFT JOIN,
// sehingga tipe yang menunjuk kategori yang tidak ada tetap terbaca dengan nama kategori
// kosong. Lihat banner pada berkas .sql.
//
// Bentuk teks kedua kunci kemudian dirapikan: driver Oracle dapat mengembalikan NUMBER
// sebagai "8.0" atau notasi ilmiah, dan ID yang berbeda bentuk tidak akan cocok dengan
// `SPAREPART_HE.TIPE_SPART` yang menyimpan "8". Lihat tidyNumber.
func scanRow(row rowScanner) (mastertipesparepart.PartType, error) {
	var (
		id           sql.NullString
		name         sql.NullString
		categoryID   sql.NullString
		categoryName sql.NullString
		status       sql.NullString
	)
	if err := row.Scan(&id, &name, &categoryID, &categoryName, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mastertipesparepart.PartType{}, err
		}
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: membaca baris: %w", err)
	}

	return mastertipesparepart.PartType{
		ID:           tidyNumber(strings.TrimSpace(id.String)),
		Name:         strings.TrimSpace(name.String),
		CategoryID:   tidyNumber(strings.TrimSpace(categoryID.String)),
		CategoryName: strings.TrimSpace(categoryName.String),
		Status: mastertipesparepart.ApprovalStatus(
			strings.TrimSpace(status.String)),
	}, nil
}

// scanNameRow membaca satu baris EMPAT kolom — hasil type_find_by_name, yang tidak ber-JOIN.
//
// Terpisah dari scanRow karena jumlah kolomnya berbeda, dan menyatukannya akan menuntut
// kueri pemeriksaan keunikan ikut ber-JOIN hanya supaya bentuk barisnya sama. Itu menambah
// pekerjaan pada jalur yang dilewati SETIAP penyimpanan, demi kolom yang tidak dipakai
// pemanggilnya.
//
// CategoryName pada hasilnya karena itu selalu kosong.
func scanNameRow(row rowScanner) (mastertipesparepart.PartType, error) {
	var (
		id         sql.NullString
		name       sql.NullString
		categoryID sql.NullString
		status     sql.NullString
	)
	if err := row.Scan(&id, &name, &categoryID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return mastertipesparepart.PartType{}, err
		}
		return mastertipesparepart.PartType{}, fmt.Errorf(
			"mastertipesparepart/sqlstore: membaca baris menurut nama: %w", err)
	}

	return mastertipesparepart.PartType{
		ID:         tidyNumber(strings.TrimSpace(id.String)),
		Name:       strings.TrimSpace(name.String),
		CategoryID: tidyNumber(strings.TrimSpace(categoryID.String)),
		Status: mastertipesparepart.ApprovalStatus(
			strings.TrimSpace(status.String)),
	}, nil
}

// tidyNumber merapikan bentuk teks sebuah angka bulat.
//
// Yang dirapikan hanyalah bentuk yang PASTI mewakili bilangan bulat yang sama: "8.0"
// menjadi "8", "08" menjadi "8". Teks yang tidak dapat dibaca sebagai bilangan bulat
// dikembalikan APA ADANYA — termasuk notasi ilmiah dan kunci yang ternyata bukan angka.
//
// Mengubah yang tidak dikenali akan lebih berbahaya daripada membiarkannya: kunci yang
// dikarang tidak akan menemukan barisnya, sedangkan kunci ganjil yang dibiarkan setidaknya
// masih menunjuk baris yang benar. Bila ia muncul, ia terlihat di layar sebagaimana adanya
// — dan itu yang diinginkan.
func tidyNumber(text string) string {
	if text == "" {
		return ""
	}
	number, err := strconv.ParseInt(strings.TrimSuffix(text, ".0"), 10, 64)
	if err != nil {
		return text
	}
	return strconv.FormatInt(number, 10)
}

// escapeLike menetralkan karakter pola pada kata kunci pencarian.
//
// Tanpa ini, pengguna yang mengetik "%" akan mencocokkan SELURUH baris, dan yang mengetik
// "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun tanda
// bahwa yang dicari bukan yang diketik.
//
// Karakter pelolosnya tidak dinyatakan dengan klausa ESCAPE karena baik Oracle maupun
// PostgreSQL memakai backslash sebagai pelolos bawaan pada LIKE. Backslash sendiri
// dilipatgandakan lebih dulu supaya ia tetap dapat dicari.
func escapeLike(keyword string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(keyword)
}

// getQuery mengambil pernyataan SQL menurut namanya.
//
// Ia PANIC bila namanya tidak ada, dan itu disengaja: nama kueri adalah konstanta yang
// ditulis programmer, bukan masukan pengguna. Salah ketik harus gagal saat uji pertama
// dijalankan, bukan menjadi galat runtime di hadapan petugas.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf(
			"mastertipesparepart/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("mastertipesparepart/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("mastertipesparepart/sqlstore: tidak dapat membaca " +
				file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("mastertipesparepart/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, rows := range body {
			if strings.HasPrefix(strings.TrimSpace(rows), "--") {
				continue
			}
			statement = append(statement, rows)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, rows := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(rows)
		if strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}

var _ mastertipesparepart.Store = (*Repo)(nil)
