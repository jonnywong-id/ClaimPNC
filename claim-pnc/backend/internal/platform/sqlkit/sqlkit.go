// Package sqlkit memuat langkah baca-tulis basis data yang dipakai bersama paket sqlstore.
//
// SQL-nya tetap milik setiap modul (berkas .sql), begitu pula cara satu baris dipindai menjadi
// tipe domain. Yang dipusatkan di sini hanyalah langkah di sekelilingnya: mengumpulkan baris,
// menerjemahkan sql.ErrNoRows menjadi galat domain, dan mengubah status beberapa baris dalam
// satu transaksi. Pesan galat tetap disusun pemanggil, supaya teksnya tidak berubah.
package sqlkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Scanner adalah *sql.Row maupun *sql.Rows.
type Scanner interface {
	Scan(target ...any) error
}

// One memindai satu baris; sql.ErrNoRows menjadi notFound, galat lain diteruskan apa adanya.
func One[T any](row Scanner, scan func(Scanner) (T, error), notFound error) (T, error) {
	found, err := scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		var none T
		return none, notFound
	case err != nil:
		var none T
		return none, err
	}
	return found, nil
}

// OneOr seperti One, tetapi galat selain sql.ErrNoRows dibungkus lewat wrap.
func OneOr[T any](row Scanner, scan func(Scanner) (T, error), notFound error, wrap func(error) error) (T, error) {
	found, err := scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		var none T
		return none, notFound
	case err != nil:
		var none T
		return none, wrap(err)
	}
	return found, nil
}

// Taken memindai satu baris pemeriksa keunikan: tidak ada baris berarti nilainya bebas (nil);
// ada baris berarti taken.
func Taken[T any](row Scanner, scan func(Scanner) (T, error), taken error) error {
	_, err := scan(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	default:
		return taken
	}
}

// Collect mengumpulkan seluruh baris hasil kueri. Galat kueri dibungkus queryFail, galat saat
// menelusuri baris dibungkus iterFail, dan galat pemindaian dibungkus scanFail (semuanya
// "…: %w"); scanFail kosong berarti galat pemindaian diteruskan apa adanya. Tanpa baris,
// hasilnya nil.
//
// scan boleh menerima *sql.Rows maupun Scanner.
func Collect[T any, R Scanner](rows *sql.Rows, err error, scan func(R) (T, error), queryFail, scanFail, iterFail string) ([]T, error) {
	if err != nil {
		return nil, fmt.Errorf("%s: %w", queryFail, err)
	}
	defer func() { _ = rows.Close() }()
	var result []T
	for rows.Next() {
		one, err := scan(any(rows).(R))
		if err != nil {
			if scanFail == "" {
				return nil, err
			}
			return nil, fmt.Errorf("%s: %w", scanFail, err)
		}
		result = append(result, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", iterFail, err)
	}
	return result, nil
}

// SetEach menjalankan statement (argumen: status, kunci) untuk setiap kunci dalam SATU
// transaksi, dan mengembalikan banyaknya baris yang dianggap berubah. Kunci dipangkas spasinya.
// Pesan galatnya berawalan prefix, misalnya "masterbengkel/sqlstore".
func SetEach(ctx context.Context, db *sql.DB, statement, status string, ids []string, prefix string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("%s: memulai transaksi keputusan: %w", prefix, err)
	}
	defer func() { _ = tx.Rollback() }()

	changed := 0
	for _, one := range ids {
		result, err := tx.ExecContext(ctx, statement, status, strings.TrimSpace(one))
		if err != nil {
			return 0, fmt.Errorf("%s: menetapkan status %q: %w", prefix, one, err)
		}
		affected, err := result.RowsAffected()
		if err != nil || affected > 0 {
			changed++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: menutup transaksi keputusan: %w", prefix, err)
	}
	return changed, nil
}
