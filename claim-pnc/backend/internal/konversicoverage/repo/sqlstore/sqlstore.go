// Package sqlstore adalah pengisi Source (LIVE) dan Target (TEST) modul Konversi
// Coverage di atas Oracle.
//
// SQL-nya dirangkai di sini, tidak di berkas .sql terpisah seperti modul lain, karena
// daftar kolomnya baru diketahui saat berjalan — dibaca dari kamus data basis data
// TEST. Yang dirangkai hanyalah NAMA TABEL dan NAMA KOLOM, keduanya berasal dari daftar
// tetap di kode (kc.Lines) atau dari ALL_TAB_COLUMNS; setiap NILAI tetap diikat lewat
// parameter binding.
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

	kc "claim-pnc/internal/konversicoverage"
	"claim-pnc/internal/konversicoverage/usecase"
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
	cache map[string]map[string]kc.Column
}

func (d *dictionary) columns(ctx context.Context, table string) (map[string]kc.Column, error) {
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
	result := map[string]kc.Column{}
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
		return nil, fmt.Errorf("membaca kolom %s.%s: %w", owner, table, err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("tabel %s.%s tidak ada (atau tidak dapat dibaca pengguna koneksi ini)", owner, table)
	}
	if d.cache == nil {
		d.cache = map[string]map[string]kc.Column{}
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
	dict *dictionary
}

// NewLive menyusun pembaca LIVE.
func NewLive(conn *sql.DB) *Live {
	return &Live{conn: conn, dict: &dictionary{conn: conn}}
}

// Objects membaca baris objek satu polis, seluruh versinya.
//
// Pembacaan dibungkus `SET TRANSACTION READ ONLY` — bila suatu hari ada pernyataan tulis
// yang keliru diarahkan ke sini, Oracle sendiri yang menolaknya (ORA-01456).
func (l *Live) Objects(ctx context.Context, line kc.Line, policyNo string) ([]kc.ObjectRow, error) {
	columns, err := l.dict.columns(ctx, line.ObjectTable)
	if err != nil {
		return nil, err
	}
	if _, ok := columns[line.BlobColumn]; !ok {
		return nil, fmt.Errorf("kolom %s tidak ada di %s.%s LIVE", line.BlobColumn, owner, line.ObjectTable)
	}
	applicationID := "NULL"
	if _, ok := columns["APPLICATIONID"]; ok {
		applicationID = "APPLICATIONID"
	}
	// INDEXTANEKALIST hanya ada di T_ANEKALIST; ia penghubung coverage Aneka ke objeknya.
	listIndex := "NULL"
	if _, ok := columns["INDEXTANEKALIST"]; ok {
		listIndex = "TO_CHAR(INDEXTANEKALIST)"
	}
	query := fmt.Sprintf(`SELECT IDPEGA, %s, NOPOLIS, TO_CHAR(PRODKE), TO_CHAR(INDEXOBJECT), %s, %s
  FROM %s.%s
 WHERE NOPOLIS = :1
 ORDER BY PRODKE, INDEXOBJECT`, applicationID, listIndex, line.BlobColumn, owner, line.ObjectTable)

	tx, err := l.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck // hanya membaca; tidak ada yang perlu dipertahankan
	if _, err := tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, fmt.Errorf("membuka transaksi baca-saja: %w", err)
	}
	rows, err := tx.QueryContext(ctx, query, policyNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []kc.ObjectRow
	for rows.Next() {
		var (
			idPega, appID, nopolis, prodke, index, listIndex sql.NullString
			document                                         []byte
		)
		if err := rows.Scan(&idPega, &appID, &nopolis, &prodke, &index, &listIndex, &document); err != nil {
			return nil, err
		}
		result = append(result, kc.ObjectRow{
			IDPega:        idPega.String,
			ApplicationID: appID.String,
			PolicyNo:      nopolis.String,
			ProdKe:        prodke.String,
			IndexObject:   index.String,
			ListIndex:     listIndex.String,
			Document:      document,
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

// Apply menulis satu polis di dalam satu transaksi.
//
// Urutannya: salin BLOB ke baris objek → hapus coverage dan spreading lama polis itu
// (per versi PRODKE yang dikonversi) → sisipkan hasil konversi.
//
// Penghapusan fisik di sini SENGAJA, dan hanya karena tujuannya basis data TEST: ia
// membuat konversi dapat diulang tanpa menumpuk baris ganda. Cakupannya dibatasi
// NOPOLIS + PRODKE yang sedang dikonversi — polis lain tidak tersentuh.
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

	// Kolom ketiga tabel tujuan dibaca SEBELUM transaksi dibuka, supaya tabel yang belum
	// ada di TEST terlapor dengan jelas alih-alih ORA-00942 di tengah jalan.
	spreadColumns, err := t.dict.columns(ctx, kc.SpreadingTable)
	if err != nil && len(batch.Spreading) > 0 {
		return finish(), err
	}
	coverageColumns := map[string]map[string]kc.Column{}
	for _, lb := range batch.Lines {
		if len(lb.Coverage) == 0 {
			continue
		}
		cols, err := t.dict.columns(ctx, lb.Line.CoverageTable)
		if err != nil {
			return finish(), err
		}
		coverageColumns[lb.Line.CoverageTable] = cols
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

	spreadVersions := map[string]bool{}
	for _, lb := range batch.Lines {
		versions := map[string]bool{}
		for _, object := range lb.Objects {
			versions[object.ProdKe] = true
			spreadVersions[object.ProdKe] = true
			n, err := updateObject(ctx, tx, lb.Line, object)
			if err != nil {
				return finish(), fmt.Errorf("memperbarui %s INDEXOBJECT %s: %w", lb.Line.ObjectTable, object.IndexObject, err)
			}
			if n == 0 {
				applied.ObjectsMissing = append(applied.ObjectsMissing, fmt.Sprintf("%s INDEXOBJECT %s PRODKE %s",
					lb.Line.ObjectTable, object.IndexObject, object.ProdKe))
			}
			applied.ObjectsUpdated += int(n)
		}

		if cols, ok := coverageColumns[lb.Line.CoverageTable]; ok {
			for _, version := range sortedSet(versions) {
				n, err := execCount(ctx, tx, fmt.Sprintf(
					"DELETE FROM %s.%s WHERE NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2", owner, lb.Line.CoverageTable),
					batch.PolicyNo, version)
				if err != nil {
					return finish(), fmt.Errorf("menghapus %s lama: %w", lb.Line.CoverageTable, err)
				}
				applied.CoverageDeleted += int(n)
			}
			n, err := insertRows(ctx, tx, lb.Line.CoverageTable, cols, lb.Coverage, warn)
			if err != nil {
				return finish(), err
			}
			applied.CoverageInserted += n
		}
	}

	if len(batch.Spreading) > 0 {
		for _, version := range sortedSet(spreadVersions) {
			n, err := execCount(ctx, tx, fmt.Sprintf(
				"DELETE FROM %s.%s WHERE NOPOLIS = :1 AND TO_CHAR(PRODKE) = :2", owner, kc.SpreadingTable),
				batch.PolicyNo, version)
			if err != nil {
				return finish(), fmt.Errorf("menghapus %s lama: %w", kc.SpreadingTable, err)
			}
			applied.SpreadDeleted += int(n)
		}
		n, err := insertRows(ctx, tx, kc.SpreadingTable, spreadColumns, batch.Spreading, warn)
		if err != nil {
			return finish(), err
		}
		applied.SpreadInserted = n
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

// updateObject menyalin BLOB LIVE apa adanya ke baris objek yang sama di TEST.
//
// Baris dicocokkan dengan NOPOLIS + PRODKE + INDEXOBJECT + IDPEGA. Bila tidak ada yang
// cocok — IDPEGA di TEST dapat berbeda bila datanya disalin dari sumber lain — pencocokan
// diulang tanpa IDPEGA.
func updateObject(ctx context.Context, tx *sql.Tx, line kc.Line, object kc.ObjectRow) (int64, error) {
	base := fmt.Sprintf("UPDATE %s.%s SET %s = :1 WHERE NOPOLIS = :2 AND TO_CHAR(PRODKE) = :3 AND TO_CHAR(INDEXOBJECT) = :4",
		owner, line.ObjectTable, line.BlobColumn)
	blob := db.BlobValue(object.Document)
	if object.IDPega != "" {
		n, err := execCount(ctx, tx, base+" AND IDPEGA = :5", blob, object.PolicyNo, object.ProdKe, object.IndexObject, object.IDPega)
		if err != nil || n > 0 {
			return n, err
		}
	}
	return execCount(ctx, tx, base, blob, object.PolicyNo, object.ProdKe, object.IndexObject)
}

// insertRows menyisipkan baris ke satu tabel. Kunci JSON tanpa kolom dilaporkan sekali
// per tabel.
func insertRows(ctx context.Context, tx *sql.Tx, table string, columns map[string]kc.Column,
	rows []kc.Row, warn func(string)) (int, error) {
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
			value, w := kc.Coerce(row[name], columns[name])
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
			query := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", owner, table, key, strings.Join(placeholders, ", "))
			prepared, err := tx.PrepareContext(ctx, query)
			if err != nil {
				return inserted, fmt.Errorf("menyiapkan sisipan %s: %w", table, err)
			}
			statements[key] = prepared
			stmt = prepared
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return inserted, fmt.Errorf("menyisipkan %s baris ke-%d (INDEXOBJECT %v, INDEXCOVERAGE %v): %w",
				table, i+1, row["INDEXOBJECT"], row["INDEXCOVERAGE"], err)
		}
		inserted++
	}
	if len(unknown) > 0 {
		warn(fmt.Sprintf("%s: kunci JSON tanpa kolom, tidak ditulis — %s", table, strings.Join(sortedSet(unknown), ", ")))
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
