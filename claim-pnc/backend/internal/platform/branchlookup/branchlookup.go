// Package branchlookup menerjemahkan login pengguna menjadi kode cabangnya lewat basis data —
// kueri yang menyambung BRANCH, LST_USER_ASURANSI, dan V_HRD_MST lewat DB Link ke HRD.
//
// Kuerinya tetap milik modul pemakai (berkas .sql-nya sendiri); yang tinggal di sini hanyalah
// langkah di sekelilingnya, yang sebelumnya tersalin sama persis di dua modul inbox.
package branchlookup

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Timeout membatasi lama penerjemahan cabang. DB Link yang mati membuat kueri menggantung,
// dan tanpa batas ini layar ikut menggantung sampai koneksinya diputus.
const Timeout = 5 * time.Second

// Resolver menerjemahkan login menjadi kode cabang.
type Resolver struct {
	db     *sql.DB
	query  string
	prefix string
}

// New membentuk Resolver. query menerima login sebagai :1 dan mengembalikan satu kolom kode
// cabang; prefix mengawali setiap pesan galat, misalnya "inboxlaporanklaim/sqlstore".
func New(db *sql.DB, query, prefix string) *Resolver {
	return &Resolver{db: db, query: query, prefix: prefix}
}

// Resolve mengembalikan kode cabang login. Nilai kedua false bila login kosong, tidak dikenal,
// atau cabangnya kosong — keadaan yang bukan galat.
func (r *Resolver) Resolve(ctx context.Context, login string) (string, bool, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	var code sql.NullString
	err := r.db.QueryRowContext(ctx, r.query, clean).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "", false, fmt.Errorf(
			"%s: penerjemahan cabang %q tidak dijawab dalam %s; "+
				"DB Link ke HRD kemungkinan tidak hidup", r.prefix, clean, Timeout)
	}
	if err != nil {
		return "", false, fmt.Errorf("%s: menerjemahkan cabang %q: %w", r.prefix, clean, err)
	}
	value := strings.TrimSpace(code.String)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

// CheckTable membuktikan ketiga tabel di balik kuerinya dapat dibaca. Dipakai `claimpnc
// -periksa`; login yang tidak ada bukan galat.
func (r *Resolver) CheckTable(ctx context.Context) error {
	var code sql.NullString
	err := r.db.QueryRowContext(ctx, r.query, "__periksa__").Scan(&code)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: BRANCH, LST_USER_ASURANSI, atau V_HRD_MST tidak dapat dibaca: %w", r.prefix, err)
	}
	return nil
}
