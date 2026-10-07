package sqlkit

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

// Beginner adalah *sql.DB, atau apa pun yang dapat memulai transaksi.
type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// RowQuerier adalah *sql.DB maupun *sql.Tx untuk kueri satu baris.
type RowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// RowsQuerier adalah *sql.DB maupun *sql.Tx untuk kueri banyak baris.
type RowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// InTx menjalankan body di dalam SATU transaksi.
//
// Rollback dipasang tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa
// itu, satu jalur galat yang terlewat meninggalkan transaksi menggantung dan menahan kunci
// sampai koneksinya didaur ulang. beginFail dan commitFail adalah awalan pesan galat saat
// transaksi gagal dimulai dan gagal ditutup; galat dari body dikembalikan apa adanya.
func InTx(ctx context.Context, db Beginner, beginFail, commitFail string, body func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", beginFail, err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := body(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: %w", commitFail, err)
	}
	return nil
}

// NextSerial membaca nomor urut berikutnya dari kueri `COALESCE(MAX(id),0)+1` lalu
// mengembalikannya sebagai teks desimal.
//
// Hasilnya dibaca sebagai int64, bukan langsung sebagai teks: driver Oracle mengembalikan
// NUMBER dalam bentuk yang bergantung pada konfigurasinya — "8", "8.0", atau notasi ilmiah —
// dan ID yang tersimpan sebagai "8.0" tidak akan pernah cocok dengan kolom acuan yang berisi
// "8". Nilai di bawah 1 ditolak: kueri itu tidak dapat menghasilkannya pada tabel yang waras,
// dan bila ia terjadi ada ID negatif di tabelnya yang deretnya akan tertimpa.
//
// Pesan galatnya berawalan prefix dan menyebut what, misalnya "ID kategori".
func NextSerial(ctx context.Context, q RowQuerier, query, prefix, what string) (string, error) {
	next, err := Serial(ctx, q, query, prefix, what)
	if err != nil {
		return "", err
	}
	if next <= 0 {
		return "", fmt.Errorf("%s: %s berikutnya tidak masuk akal: %d", prefix, what, next)
	}
	return strconv.FormatInt(next, 10), nil
}

// Serial membaca nomor urut berikutnya apa adanya, tanpa pemeriksaan batas.
func Serial(ctx context.Context, q RowQuerier, query, prefix, what string) (int64, error) {
	var next int64
	if err := q.QueryRowContext(ctx, query).Scan(&next); err != nil {
		return 0, fmt.Errorf("%s: menerbitkan %s: %w", prefix, what, err)
	}
	return next, nil
}

// Count menjalankan satu kueri pencacah. Pesan galatnya berawalan label.
func Count(ctx context.Context, q RowQuerier, query, label string, args ...any) (int, error) {
	var total int
	if err := q.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("%s: %w", label, err)
	}
	return total, nil
}

// CheckReadable membuktikan kueri dapat dijalankan dan barisnya dapat dibaca — dipakai
// `claimpnc -periksa` untuk memastikan tabel beserta kolomnya ada. Pesan galatnya berawalan
// fail.
func CheckReadable(ctx context.Context, q RowsQuerier, query, fail string) error {
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("%s: %w", fail, err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// RequireAffected mengembalikan notFound bila pernyataan tidak menyentuh satu baris pun.
//
// UPDATE yang tidak menyentuh baris berhasil menurut basis data, dan tanpa pemeriksaan ini
// layar akan mengatakan "tersimpan" atas baris yang sudah tidak ada. Driver yang tidak
// mendukung RowsAffected mengembalikan galat; dalam keadaan itu perubahannya TIDAK dianggap
// gagal — pernyataannya sendiri sudah berhasil, dan menolaknya akan menampilkan kegagalan
// palsu.
func RequireAffected(result sql.Result, notFound error) error {
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return notFound
	}
	return nil
}
