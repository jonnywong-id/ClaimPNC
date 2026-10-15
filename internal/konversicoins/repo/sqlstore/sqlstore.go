// Package sqlstore adalah pengisi Source (LIVE) dan Target (TEST) modul Konversi Coins di
// atas Oracle.
//
// SQL-nya dirangkai di sini, tidak di berkas .sql terpisah seperti modul lain, karena kolom
// yang ditulis bergantung pada kamus data basis data TEST (kolom yang tidak ada di TEST
// dilewati). Yang dirangkai hanyalah NAMA KOLOM dari kc.Columns dan nama tabel tetap di
// kode; setiap NILAI tetap diikat lewat parameter binding.
package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	kc "claim-pnc/internal/konversicoins"
	"claim-pnc/internal/konversicoins/usecase"
)

const owner = "POOLDATA"

const columnsSQL = `SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, CHAR_LENGTH, CHAR_USED
  FROM ALL_TAB_COLUMNS
 WHERE OWNER = :1 AND TABLE_NAME = :2
 ORDER BY COLUMN_ID`

// dictionary membaca dan menyimpan susunan kolom T_COINSLIST basis data TEST.
type dictionary struct {
	conn *sql.DB

	mu     sync.Mutex
	cached map[string]kc.Column
}

func (d *dictionary) columns(ctx context.Context) (map[string]kc.Column, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached != nil {
		return d.cached, nil
	}
	rows, err := d.conn.QueryContext(ctx, columnsSQL, owner, kc.CoinsTable)
	if err != nil {
		return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, kc.CoinsTable, err)
	}
	defer rows.Close()
	result := map[string]kc.Column{}
	for rows.Next() {
		var (
			name, dataType string
			dataLength     sql.NullInt64
			charLength     sql.NullInt64
			charUsed       sql.NullString
		)
		if err := rows.Scan(&name, &dataType, &dataLength, &charLength, &charUsed); err != nil {
			return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, kc.CoinsTable, err)
		}
		column := kc.Column{Name: name, DataType: strings.ToUpper(dataType)}
		if isTextType(column.DataType) {
			column.MaxBytes = int(dataLength.Int64)
			if charUsed.String == "C" {
				column.MaxChars = int(charLength.Int64)
			}
		}
		result[name] = column
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, kc.CoinsTable, err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("tabel %s.%s tidak ada (atau tidak dapat dibaca pengguna koneksi ini)", owner, kc.CoinsTable)
	}
	d.cached = result
	return result, nil
}

func isTextType(dataType string) bool {
	return dataType == "VARCHAR2" || dataType == "CHAR" || dataType == "NVARCHAR2" || dataType == "NCHAR"
}

// ============================================================================
// LIVE
// ============================================================================

// Live membaca basis data LIVE. Ia TIDAK punya satu pun pernyataan tulis.
type Live struct {
	conn *sql.DB
}

// NewLive menyusun pembaca LIVE.
func NewLive(conn *sql.DB) *Live { return &Live{conn: conn} }

var _ usecase.Source = (*Live)(nil)

// documentsSQL memilih satu dokumen per PRODKE: baris ber-DATA_JSONBLOB dengan TGL_INPUT
// terbaru. Baris itulah yang IDPEGA-nya dipakai Pega di T_COINSLIST LIVE (40/40 polis,
// terukur 2026-10-08).
var documentsSQL = fmt.Sprintf(`SELECT IDPEGA, NOPOLIS, PRODKE, %[1]s
  FROM (SELECT p.IDPEGA, p.NOPOLIS, p.PRODKE, p.%[1]s,
               ROW_NUMBER() OVER (PARTITION BY p.PRODKE ORDER BY p.TGL_INPUT DESC NULLS LAST) AS RN
          FROM %[2]s.%[3]s p
         WHERE p.NOPOLIS = :1
           AND p.%[1]s IS NOT NULL)
 WHERE RN = 1
 ORDER BY LENGTH(PRODKE), PRODKE`, kc.BlobColumn, owner, kc.SourceTable)

// Documents membaca dokumen polis satu nomor polis, satu per versi.
//
// Pembacaan dibungkus `SET TRANSACTION READ ONLY` — bila suatu hari ada pernyataan tulis yang
// keliru diarahkan ke sini, Oracle sendiri yang menolaknya (ORA-01456).
func (l *Live) Documents(ctx context.Context, policyNo string) ([]kc.PolicyDoc, error) {
	tx, err := l.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // hanya membaca; tidak ada yang perlu dipertahankan
	if _, err := tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, fmt.Errorf("membuka transaksi baca-saja: %w", err)
	}
	rows, err := tx.QueryContext(ctx, documentsSQL, policyNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []kc.PolicyDoc
	for rows.Next() {
		var (
			idPega, nopolis, prodke sql.NullString
			document                []byte
		)
		if err := rows.Scan(&idPega, &nopolis, &prodke, &document); err != nil {
			return nil, err
		}
		result = append(result, kc.PolicyDoc{
			IDPega: idPega.String, PolicyNo: nopolis.String, ProdKe: prodke.String, Document: document,
		})
	}
	return result, rows.Err()
}

// ============================================================================
// TEST
// ============================================================================

// Test menulis basis data TEST.
type Test struct {
	conn *sql.DB
	dict *dictionary
}

// NewTest menyusun penulis TEST.
func NewTest(conn *sql.DB) *Test {
	return &Test{conn: conn, dict: &dictionary{conn: conn}}
}

var _ usecase.Target = (*Test)(nil)

// Apply menulis satu polis di dalam satu transaksi: hapus baris T_COINSLIST lama polis itu
// per versi PRODKE yang dikonversi, lalu sisipkan hasil konversi.
//
// Penghapusan fisik di sini SENGAJA, dan hanya karena tujuannya basis data TEST: ia membuat
// konversi dapat diulang tanpa menumpuk baris ganda, dan versi yang di LIVE tanpa koasuransi
// ikut menjadi kosong di TEST. Cakupannya dibatasi NOPOLIS + PRODKE yang sedang dikonversi —
// polis lain tidak tersentuh.
func (t *Test) Apply(ctx context.Context, batch usecase.Batch, commit bool) (usecase.Applied, error) {
	var applied usecase.Applied
	warnings := map[string]bool{}
	finish := func() usecase.Applied {
		for w := range warnings {
			applied.Warnings = append(applied.Warnings, w)
		}
		sort.Strings(applied.Warnings)
		return applied
	}

	// Kolom dibaca SEBELUM transaksi dibuka, supaya tabel yang belum ada di TEST terlapor
	// dengan jelas alih-alih ORA-00942 di tengah jalan.
	columns, err := t.dict.columns(ctx)
	if err != nil {
		return finish(), err
	}

	tx, err := t.conn.BeginTx(ctx, nil)
	if err != nil {
		return finish(), err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, version := range batch.Versions {
		n, err := execCount(ctx, tx, fmt.Sprintf(
			"DELETE FROM %s.%s WHERE NOPOLIS = :1 AND PRODKE = :2", owner, kc.CoinsTable),
			batch.PolicyNo, version)
		if err != nil {
			return finish(), fmt.Errorf("menghapus %s lama PRODKE %s: %w", kc.CoinsTable, version, err)
		}
		applied.Deleted += int(n)
	}

	n, err := insertRows(ctx, tx, columns, batch.Rows, func(w string) {
		if w != "" {
			warnings[w] = true
		}
	})
	applied.Inserted = n
	if err != nil {
		return finish(), err
	}

	if !commit {
		return finish(), nil // uji coba: dibatalkan oleh defer
	}
	if err := tx.Commit(); err != nil {
		return finish(), err
	}
	committed = true
	return finish(), nil
}

// targetColumns adalah kolom yang mungkin ditulis, urutan tetap: kunci lalu pemetaan item.
func targetColumns() []string {
	names := []string{kc.ColIDPega, kc.ColPolicyNo, kc.ColProdKe}
	for _, m := range kc.Columns {
		names = append(names, m.Column)
	}
	return names
}

// insertRows menyisipkan baris ke T_COINSLIST. Kolom pemetaan yang tidak ada di TEST
// dilaporkan sekali dan dilewati.
func insertRows(ctx context.Context, tx *sql.Tx, columns map[string]kc.Column,
	rows []kc.Row, warn func(string)) (int, error) {
	var names, missing []string
	for _, name := range targetColumns() {
		if _, ok := columns[name]; ok {
			names = append(names, name)
		} else {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		warn(fmt.Sprintf("%s TEST tidak punya kolom %s — tidak ditulis", kc.CoinsTable, strings.Join(missing, ", ")))
	}
	if len(rows) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(names))
	for i := range names {
		placeholders[i] = ":" + strconv.Itoa(i+1)
	}
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)",
		owner, kc.CoinsTable, strings.Join(names, ", "), strings.Join(placeholders, ", ")))
	if err != nil {
		return 0, fmt.Errorf("menyiapkan sisipan %s: %w", kc.CoinsTable, err)
	}
	defer stmt.Close()

	inserted := 0
	for i, row := range rows {
		args := make([]any, len(names))
		for j, name := range names {
			value, w := kc.Coerce(row[name], columns[name])
			warn(w)
			args[j] = value
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return inserted, fmt.Errorf("menyisipkan %s baris ke-%d (PRODKE %v, COINSID %v): %w",
				kc.CoinsTable, i+1, row[kc.ColProdKe], row["COINSID"], err)
		}
		inserted++
	}
	return inserted, nil
}

func execCount(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, errors.Join(errors.New("jumlah baris terdampak tidak terbaca"), err)
	}
	return n, nil
}
