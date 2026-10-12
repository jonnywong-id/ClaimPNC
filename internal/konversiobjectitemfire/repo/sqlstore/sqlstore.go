// Package sqlstore adalah pengisi Source (LIVE) dan Target (TEST) modul Konversi Object
// Item Fire di atas Oracle.
//
// SQL-nya dirangkai di sini, tidak di berkas .sql terpisah seperti modul lain, karena daftar
// kolom T_PROPERTYITEMLIST baru diketahui saat berjalan — dibaca dari kamus data basis data
// TEST. Yang dirangkai hanyalah NAMA KOLOM dari ALL_TAB_COLUMNS dan nama tabel tetap di kode;
// setiap NILAI tetap diikat lewat parameter binding.
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

	ki "claim-pnc/internal/konversiobjectitemfire"
	"claim-pnc/internal/konversiobjectitemfire/usecase"
	"claim-pnc/internal/platform/db"
)

const owner = "POOLDATA"

const columnsSQL = `SELECT COLUMN_NAME, DATA_TYPE, DATA_LENGTH, CHAR_LENGTH, CHAR_USED
  FROM ALL_TAB_COLUMNS
 WHERE OWNER = :1 AND TABLE_NAME = :2
 ORDER BY COLUMN_ID`

// dictionary membaca dan menyimpan susunan kolom tabel satu basis data.
type dictionary struct {
	conn *sql.DB

	mu    sync.Mutex
	cache map[string]map[string]ki.Column
}

func (d *dictionary) columns(ctx context.Context, table string) (map[string]ki.Column, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cached, ok := d.cache[table]; ok {
		return cached, nil
	}
	rows, err := d.conn.QueryContext(ctx, columnsSQL, owner, table)
	if err != nil {
		return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, table, err)
	}
	defer rows.Close()
	result := map[string]ki.Column{}
	for rows.Next() {
		var (
			name, dataType string
			dataLength     sql.NullInt64
			charLength     sql.NullInt64
			charUsed       sql.NullString
		)
		if err := rows.Scan(&name, &dataType, &dataLength, &charLength, &charUsed); err != nil {
			return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, table, err)
		}
		column := ki.Column{Name: name, DataType: strings.ToUpper(dataType)}
		if isTextType(column.DataType) {
			column.MaxBytes = int(dataLength.Int64)
			if charUsed.String == "C" {
				column.MaxChars = int(charLength.Int64)
			}
		}
		result[name] = column
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, table, err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("tabel %s.%s tidak ada (atau tidak dapat dibaca pengguna koneksi ini)", owner, table)
	}
	if d.cache == nil {
		d.cache = map[string]map[string]ki.Column{}
	}
	d.cache[table] = result
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

// objectsSQL membaca baris objek satu polis, seluruh versinya.
var objectsSQL = fmt.Sprintf(`SELECT IDPEGA, NOPOLIS, TO_CHAR(PRODKE), TO_CHAR(INDEXOBJECT), %s
  FROM %s.%s
 WHERE NOPOLIS = :1
 ORDER BY PRODKE, INDEXOBJECT`, ki.BlobColumn, owner, ki.ObjectTable)

// Objects membaca baris T_PROPERTYLIST satu polis, seluruh versinya.
//
// Pembacaan dibungkus `SET TRANSACTION READ ONLY` — bila suatu hari ada pernyataan tulis yang
// keliru diarahkan ke sini, Oracle sendiri yang menolaknya (ORA-01456).
func (l *Live) Objects(ctx context.Context, policyNo string) ([]ki.ObjectRow, error) {
	tx, err := l.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // hanya membaca; tidak ada yang perlu dipertahankan
	if _, err := tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, fmt.Errorf("membuka transaksi baca-saja: %w", err)
	}
	rows, err := tx.QueryContext(ctx, objectsSQL, policyNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ki.ObjectRow
	for rows.Next() {
		var (
			idPega, nopolis, prodke, index sql.NullString
			document                       []byte
		)
		if err := rows.Scan(&idPega, &nopolis, &prodke, &index, &document); err != nil {
			return nil, err
		}
		result = append(result, ki.ObjectRow{
			IDPega: idPega.String, PolicyNo: nopolis.String, ProdKe: prodke.String,
			IndexObject: index.String, Document: document,
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

// Apply menulis satu polis di dalam satu transaksi.
//
// Urutannya: salin BLOB ke baris objek → hapus item lama polis itu (per versi PRODKE yang
// dikonversi) → sisipkan hasil konversi.
//
// Penghapusan fisik di sini SENGAJA, dan hanya karena tujuannya basis data TEST: ia membuat
// konversi dapat diulang tanpa menumpuk baris ganda. Cakupannya dibatasi NOPOLIS + PRODKE yang
// sedang dikonversi — polis lain tidak tersentuh.
func (t *Test) Apply(ctx context.Context, batch usecase.Batch, commit bool) (usecase.Applied, error) {
	var applied usecase.Applied
	warnings := map[string]bool{}
	warn := func(text string) {
		if text != "" {
			warnings[text] = true
		}
	}
	finish := func() usecase.Applied {
		for w := range warnings {
			applied.Warnings = append(applied.Warnings, w)
		}
		sort.Strings(applied.Warnings)
		return applied
	}

	// Kolom tabel tujuan dibaca SEBELUM transaksi dibuka, supaya tabel yang belum ada di
	// TEST terlapor dengan jelas alih-alih ORA-00942 di tengah jalan.
	columns, err := t.dict.columns(ctx, ki.ItemTable)
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

	versions := map[string]bool{}
	for _, object := range batch.Objects {
		versions[object.ProdKe] = true
		n, err := updateObject(ctx, tx, object)
		if err != nil {
			return finish(), fmt.Errorf("memperbarui %s INDEXOBJECT %s: %w", ki.ObjectTable, object.IndexObject, err)
		}
		if n == 0 {
			applied.ObjectsMissing = append(applied.ObjectsMissing,
				fmt.Sprintf("INDEXOBJECT %s PRODKE %s", object.IndexObject, object.ProdKe))
		}
		applied.ObjectsUpdated += int(n)
	}

	for _, version := range sortedSet(versions) {
		n, err := execCount(ctx, tx, fmt.Sprintf(
			"DELETE FROM %s.%s WHERE NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2", owner, ki.ItemTable),
			batch.PolicyNo, version)
		if err != nil {
			return finish(), fmt.Errorf("menghapus %s lama: %w", ki.ItemTable, err)
		}
		applied.ItemsDeleted += int(n)
	}
	n, err := insertRows(ctx, tx, columns, batch.Items, warn)
	if err != nil {
		return finish(), err
	}
	applied.ItemsInserted = n

	if !commit {
		return finish(), nil // uji coba: dibatalkan oleh defer
	}
	if err := tx.Commit(); err != nil {
		return finish(), err
	}
	committed = true
	return finish(), nil
}

// updateObject menyalin BLOB LIVE apa adanya ke baris objek yang sama di TEST.
//
// Baris dicocokkan dengan NOPOLIS + PRODKE + INDEXOBJECT + IDPEGA. Bila tidak ada yang cocok —
// IDPEGA di TEST dapat berbeda bila datanya disalin dari sumber lain — pencocokan diulang
// tanpa IDPEGA.
func updateObject(ctx context.Context, tx *sql.Tx, object ki.ObjectRow) (int64, error) {
	base := fmt.Sprintf("UPDATE %s.%s SET %s = :1 WHERE NOPOLIS = :2 AND TO_CHAR(PRODKE) = :3 AND TO_CHAR(INDEXOBJECT) = :4",
		owner, ki.ObjectTable, ki.BlobColumn)
	blob := db.BlobValue(object.Document)
	if object.IDPega != "" {
		n, err := execCount(ctx, tx, base+" AND IDPEGA = :5", blob, object.PolicyNo, object.ProdKe, object.IndexObject, object.IDPega)
		if err != nil || n > 0 {
			return n, err
		}
	}
	return execCount(ctx, tx, base, blob, object.PolicyNo, object.ProdKe, object.IndexObject)
}

// insertRows menyisipkan baris ke T_PROPERTYITEMLIST. Kunci JSON tanpa kolom dilaporkan sekali.
func insertRows(ctx context.Context, tx *sql.Tx, columns map[string]ki.Column,
	rows []ki.Row, warn func(string)) (int, error) {
	unknown := map[string]bool{}
	statements := map[string]*sql.Stmt{}
	defer func() {
		for _, s := range statements {
			_ = s.Close()
		}
	}()

	inserted := 0
	for i, row := range rows {
		var names []string
		for key := range row {
			if _, ok := columns[key]; ok {
				names = append(names, key)
			} else {
				unknown[key] = true
			}
		}
		if len(names) == 0 {
			continue
		}
		sort.Strings(names)

		args := make([]any, len(names))
		for j, name := range names {
			value, w := ki.Coerce(row[name], columns[name])
			warn(w)
			switch v := value.(type) {
			case []byte:
				value = db.BlobValue(v)
			case string:
				if dt := columns[name].DataType; dt == "CLOB" || dt == "NCLOB" {
					value = db.ClobValue(v)
				}
			}
			args[j] = value
		}

		key := strings.Join(names, ",")
		stmt, ok := statements[key]
		if !ok {
			placeholders := make([]string, len(names))
			for j := range names {
				placeholders[j] = ":" + strconv.Itoa(j+1)
			}
			query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", owner, ki.ItemTable, key, strings.Join(placeholders, ", "))
			prepared, err := tx.PrepareContext(ctx, query)
			if err != nil {
				return inserted, fmt.Errorf("menyiapkan sisipan %s: %w", ki.ItemTable, err)
			}
			statements[key] = prepared
			stmt = prepared
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return inserted, fmt.Errorf("menyisipkan %s baris ke-%d (INDEXOBJECT %v, ITEMTYPE %v): %w",
				ki.ItemTable, i+1, row["INDEXOBJECT"], row["ITEMTYPE"], err)
		}
		inserted++
	}
	if len(unknown) > 0 {
		warn(fmt.Sprintf("%s: kunci JSON tanpa kolom, tidak ditulis — %s", ki.ItemTable, strings.Join(sortedSet(unknown), ", ")))
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

func sortedSet(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
