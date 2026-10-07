// Package sqlstore adalah pengisi seam masterkategorisparepart.Store terhadap basis data.
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

	"claim-pnc/internal/masterkategorisparepart"
	"claim-pnc/internal/platform/sqlfile"
	"claim-pnc/internal/platform/sqlkit"
	"claim-pnc/internal/platform/sqlvalue"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "masterkategorisparepart/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string {
	return sqlfile.MustGet(query, "masterkategorisparepart/sqlstore", name)
}

// splitByName memecah isi satu berkas .sql dengan aturan yang sama seperti pemuat di atas.
func splitByName(content string) map[string]string { return sqlfile.Split(content) }

// Repo membaca dan menulis POOLDATA.GCNM_M_SPAREPART_CATEGORY.
//
// Satu struct memenuhi kedua seam — masterkategorisparepart.Repo dan IDSource — karena
// keduanya selalu berasal dari koneksi entitas yang sama. Yang terpisah adalah
// interface-nya, bukan pengisinya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel —
// lihat berkas .sql untuk alasannya.
func (r *Repo) List(
	ctx context.Context,
	filter masterkategorisparepart.Filter,
) ([]masterkategorisparepart.PartCategory, error) {
	keyword := strings.ToUpper(strings.TrimSpace(filter.Keyword))

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("category_list"), string(filter.Status))
	} else {
		// Tanda persen dipasang di sini, bukan di dalam teks SQL: nilai yang dikirim ke basis
		// data tetap lewat parameter binding, dan pola pencariannya tetap terbaca di satu
		// tempat.
		rows, err = r.db.QueryContext(ctx, getQuery("category_list_search"),
			string(filter.Status), "%"+escapeLike(keyword)+"%")
	}
	return sqlkit.Collect(rows, err, scanRow, "masterkategorisparepart/sqlstore: membaca daftar", "", "masterkategorisparepart/sqlstore: menelusuri daftar")
}

// Get mengembalikan satu baris berdasarkan kuncinya.
func (r *Repo) Get(
	ctx context.Context,
	id string,
) (masterkategorisparepart.PartCategory, error) {
	row := r.db.QueryRowContext(ctx, getQuery("category_get"), strings.TrimSpace(id))

	return sqlkit.One(row, scanRow, masterkategorisparepart.ErrNotFound)
}

// FindByName mencari baris menurut namanya.
//
// Pembandingnya di-uppercase DI SINI dan bukan di dalam kueri untuk sisi parameternya:
// kueri sudah memakai `UPPER(TRIM(PART_CATEGORY_NAME))` di sisi kolom, dan menaruh UPPER
// pada kedua sisi di dalam teks SQL akan memaksa basis data mengubah nilai yang sudah dapat
// diubah sekali di sini.
func (r *Repo) FindByName(
	ctx context.Context,
	name string,
) (masterkategorisparepart.PartCategory, error) {
	row := r.db.QueryRowContext(ctx, getQuery("category_find_by_name"),
		strings.ToUpper(strings.TrimSpace(name)))

	return sqlkit.One(row, scanRow, masterkategorisparepart.ErrNotFound)
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
// ID pada argumen DIABAIKAN dengan sengaja; lihat masterkategorisparepart.Repo.Insert.
func (r *Repo) Insert(
	ctx context.Context,
	c masterkategorisparepart.PartCategory,
) (masterkategorisparepart.PartCategory, error) {
	var fresh masterkategorisparepart.PartCategory
	err := sqlkit.InTx(ctx, r.db, "masterkategorisparepart/sqlstore: memulai transaksi", "masterkategorisparepart/sqlstore: menutup transaksi sisip", func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, getQuery("category_lock_table")); err != nil {
			return fmt.Errorf("masterkategorisparepart/sqlstore: mengunci tabel kategori: %w", err)
		}
		if err := rejectTakenName(ctx, tx, c.Name); err != nil {
			return err
		}
		id, err := nextID(ctx, tx)
		if err != nil {
			return err
		}
		fresh = masterkategorisparepart.PartCategory{
			ID:     id,
			Name:   c.Name,
			Status: c.Status,
		}
		if _, err := tx.ExecContext(ctx, getQuery("category_insert"),
			fresh.ID, fresh.Name, string(fresh.Status)); err != nil {
			return fmt.Errorf("masterkategorisparepart/sqlstore: menyisipkan %q: %w", fresh.Name, err)
		}
		return nil
	})
	if err != nil {
		return masterkategorisparepart.PartCategory{}, err
	}
	return fresh, nil
}

// rejectTakenName menolak nama yang sudah dipakai baris mana pun.
//
// Dijalankan DI DALAM transaksi penambahan, bukan lewat FindByName di luar: pemeriksaan
// yang berjalan pada koneksi lain tidak melihat kunci tabel yang sedang dipegang, sehingga
// hasilnya dapat basi sebelum penyisipannya berjalan.
//
// TIDAK menyaring APPROVAL, meniru `ValidationSparepartCat` apa adanya — nama yang pernah
// ditolak tetap memblokir. Lihat masterkategorisparepart.ErrNameTaken.
func rejectTakenName(ctx context.Context, tx *sql.Tx, name string) error {
	row := tx.QueryRowContext(ctx, getQuery("category_find_by_name"), strings.ToUpper(strings.TrimSpace(name)))
	return sqlkit.Taken(row, scanRow, masterkategorisparepart.ErrNameTaken)
}

// nextID membaca nomor urut berikutnya di dalam transaksi yang sudah memegang kunci tabel.
//
// Hasilnya dibaca sebagai int64 lalu diubah menjadi teks desimal, bukan dibaca langsung
// sebagai string. Alasannya: driver Oracle mengembalikan NUMBER sebagai bentuk yang
// bergantung pada konfigurasinya — "8", "8.0", atau bahkan notasi ilmiah pada nilai besar —
// dan ID yang tersimpan sebagai "8.0" tidak akan pernah cocok dengan
// `SPAREPART_HE.KATEGORI_SPART` yang berisi "8".
func nextID(ctx context.Context, tx *sql.Tx) (string, error) {
	return sqlkit.NextSerial(ctx, tx, getQuery("category_next_id"), "masterkategorisparepart/sqlstore", "ID kategori")
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
	next, err := sqlkit.Serial(ctx, r.db, getQuery("category_next_id"), "masterkategorisparepart/sqlstore", "ID kategori")
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(next, 10), nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// Pemeriksaan bentrok nama TIDAK dilakukan di sini, dan itu mengikuti sistem lama:
// `ValidateMasterKategoriSparepart` dipanggil dari layar, bukan dari jalur penyimpanan.
// Lapisan aplikasi yang memeriksanya, dan ia mengecualikan baris itu sendiri.
func (r *Repo) Update(ctx context.Context, c masterkategorisparepart.PartCategory) error {
	result, err := r.db.ExecContext(ctx, getQuery("category_update"),
		c.Name, string(c.Status), strings.TrimSpace(c.ID))
	if err != nil {
		return fmt.Errorf("masterkategorisparepart/sqlstore: memperbarui %q: %w", c.ID, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh satu
	// baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan mengatakan
	// "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	return sqlkit.RequireAffected(result, masterkategorisparepart.ErrNotFound)
}

// SetStatus menetapkan APPROVAL sejumlah baris di dalam SATU transaksi.
//
// Bentuk pernyataannya DIREKONSTRUKSI; lihat catatan pada `category_set_status` di berkas
// .sql dan pada masterkategorisparepart.Repo.SetStatus.
func (r *Repo) SetStatus(ctx context.Context, id []string, status masterkategorisparepart.ApprovalStatus) (int, error) {
	return sqlkit.SetEach(ctx, r.db, getQuery("category_set_status"), string(status), id, "masterkategorisparepart/sqlstore")
}

// CountByStatus mencacah baris satu status. Dipakai `claimpnc -periksa`.
func (r *Repo) CountByStatus(
	ctx context.Context,
	status masterkategorisparepart.ApprovalStatus,
) (int, error) {
	return r.count(ctx, "category_count_by_status", string(status))
}

// CountAll mencacah seluruh baris tanpa memandang status. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "category_count_all")
}

// CountUnknownStatus mencacah baris yang APPROVAL-nya di luar ketiga sandi yang dikenal.
//
// Baris seperti itu tidak muncul di satu pun tab; lihat berkas .sql.
func (r *Repo) CountUnknownStatus(ctx context.Context) (int, error) {
	return r.count(ctx, "category_count_unknown_status")
}

// CountDuplicateName mencacah NAMA yang dipakai lebih dari satu baris.
func (r *Repo) CountDuplicateName(ctx context.Context) (int, error) {
	return r.count(ctx, "category_count_duplicate_name")
}

// CountOrphanSparepart mencacah sparepart yang menunjuk kategori yang tidak ada.
func (r *Repo) CountOrphanSparepart(ctx context.Context) (int, error) {
	return r.count(ctx, "category_count_orphan_sparepart")
}

// CheckTable membuktikan tabel beserta ketiga kolomnya ada dan dapat dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	return sqlkit.CheckReadable(ctx, r.db, getQuery("category_check_table"), "masterkategorisparepart/sqlstore: memeriksa tabel kategori")
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	return sqlkit.Count(ctx, r.db, getQuery(name), "masterkategorisparepart/sqlstore: "+name, argument...)
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun
// di pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat
// yang dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner = sqlkit.Scanner

// scanRow membaca satu baris menjadi PartCategory.
//
// # Ketiga kolomnya dibaca sebagai teks yang boleh NULL
//
// Termasuk PART_CATEGORY_ID, yang di basis data bertipe angka. Membacanya sebagai int64
// akan menolak baris yang ID-nya NULL — keadaan yang tidak seharusnya ada tetapi tidak
// dijaga constraint apa pun (R-08) — dan menggagalkan SELURUH daftar karena satu baris
// rusak.
//
// Bentuk teksnya kemudian dirapikan: driver Oracle dapat mengembalikan NUMBER sebagai "8.0"
// atau notasi ilmiah, dan ID yang berbeda bentuk tidak akan cocok dengan
// `SPAREPART_HE.KATEGORI_SPART` yang menyimpan "8". Lihat tidyNumber.
func scanRow(row rowScanner) (masterkategorisparepart.PartCategory, error) {
	var (
		id     sql.NullString
		name   sql.NullString
		status sql.NullString
	)
	if err := row.Scan(&id, &name, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterkategorisparepart.PartCategory{}, err
		}
		return masterkategorisparepart.PartCategory{}, fmt.Errorf(
			"masterkategorisparepart/sqlstore: membaca baris: %w", err)
	}

	return masterkategorisparepart.PartCategory{
		ID:   tidyNumber(strings.TrimSpace(id.String)),
		Name: strings.TrimSpace(name.String),
		Status: masterkategorisparepart.ApprovalStatus(
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
func tidyNumber(text string) string { return sqlvalue.TidyNumber(text) }

// escapeLike menetralkan karakter pola pada kata kunci pencarian.
//
// Tanpa ini, pengguna yang mengetik "%" akan mencocokkan SELURUH baris, dan yang mengetik
// "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun tanda
// bahwa yang dicari bukan yang diketik.
//
// Karakter pelolosnya tidak dinyatakan dengan klausa ESCAPE karena baik Oracle maupun
// PostgreSQL memakai backslash sebagai pelolos bawaan pada LIKE. Backslash sendiri
// dilipatgandakan lebih dulu supaya ia tetap dapat dicari.
func escapeLike(keyword string) string { return sqlvalue.EscapeLike(keyword) }

var _ masterkategorisparepart.Store = (*Repo)(nil)
