package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/reportkpi"
)

// picListMarker adalah tempat daftar PIC disisipkan.
//
// Kueri Pega menulis `IN {ASIS:TempDateReport.UserTeknis}` — perangkaian NILAI ke dalam
// teks SQL, dan itu persis celah yang `D-20` tutup. Yang disisipkan di sini adalah
// PENANDA BIND sebanyak PIC-nya (`(:3, :4, …)`), bukan nilainya.
const picListMarker = "/*PIC_LIST*/"

// tanggalTampil adalah bentuk tanggal pada berkas ekspor.
//
// Sama persis dengan `TO_CHAR(..., 'dd/mm/yyyy')` yang dipakai kueri Pega. Kueri di sini
// mengembalikan TANGGAL, dan pemformatannya dilakukan di Go (`D-20` §4.3).
const tanggalTampil = "02/01/2006"

// kolomTahun adalah kolom yang ditulis sebagai TAHUN saja, bukan tanggal penuh.
//
// Satu-satunya. Kueri Pega menulisnya `TO_CHAR(a.DATEOFLOSS, 'YYYY')` sementara kolom
// tetangganya — `EDMDATE` — menulis TANGGAL YANG SAMA secara penuh. Tanpa pengecualian
// ini, keduanya akan tergambar identik dan satu kolom berkas menjadi salah tanpa ada yang
// menyadarinya.
var kolomTahun = map[string]bool{"ENDDATE": true}

// PICExport menjalankan satu kueri ekspor tab KPI PIC Teknik.
//
// # Kenapa judul kolomnya diambil dari hasil kueri
//
// `rows.Columns()` mengembalikan alias persis seperti tertulis di berkas .sql. Menyalinnya
// ke daftar di dalam kode berarti dua sumber kebenaran untuk hal yang sama — dan yang
// kedua akan menyimpang diam-diam begitu satu kolom ditambahkan. Karena judul inilah yang
// menjadi kepala berkas CSV yang dibandingkan dengan Pega, penyimpangannya baru terlihat
// ketika seseorang membandingkan dua berkas.
func (r *Repo) PICExport(
	ctx context.Context,
	kind reportkpi.PICExportKind,
	q reportkpi.PICQuery,
	pics []string,
) (reportkpi.PICExportTable, error) {
	// Tanpa satu pun PIC, kueri Pega menghasilkan `IN ()` — galat sintaks. Di sini ia
	// dijawab sebagai berkas berjudul kolom tanpa baris: tidak ada yang dapat dinilai,
	// dan itu bukan kegagalan.
	if len(pics) == 0 {
		header, err := r.picExportHeader(ctx, kind)
		return reportkpi.PICExportTable{Header: header}, err
	}

	name := kind.QueryName()
	statement, args := picExportStatement(query(name), q, pics)

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return reportkpi.PICExportTable{}, fmt.Errorf("%w: kueri %q: %w", ErrQueryFailed, name, err)
	}
	defer func() { _ = rows.Close() }()

	return bacaTabelEkspor(rows)
}

// picExportHeader mengambil judul kolom saja, tanpa membaca satu baris pun.
func (r *Repo) picExportHeader(
	ctx context.Context,
	kind reportkpi.PICExportKind,
) ([]string, error) {
	name := kind.QueryName()
	// Satu penanda palsu yang tidak akan cocok dengan PIC mana pun: yang dibutuhkan hanya
	// bentuk kolomnya, bukan isinya.
	statement, args := picExportStatement(query(name), reportkpi.PICQuery{}, []string{""})

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: kueri %q: %w", ErrQueryFailed, name, err)
	}
	defer func() { _ = rows.Close() }()

	names, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("%w: membaca kolom %q: %w", ErrQueryFailed, name, err)
	}
	return names, nil
}

// picExportStatement menyisipkan penanda bind daftar PIC dan menyusun parameternya.
//
// Urutannya: `:1` dari, `:2` sampai, lalu satu penanda per PIC. Nomor penanda dihitung,
// bukan ditebak, supaya penambahan parameter di kemudian hari tidak diam-diam menggeser
// daftar PIC-nya.
func picExportStatement(
	text string,
	q reportkpi.PICQuery,
	pics []string,
) (string, []any) {
	placeholder := make([]string, 0, len(pics))
	args := []any{q.From, q.To}
	for i, pic := range pics {
		placeholder = append(placeholder, ":"+strconv.Itoa(i+3))
		args = append(args, pic)
	}
	statement := strings.Replace(text, picListMarker, "("+strings.Join(placeholder, ", ")+")", 1)
	return statement, args
}

// bacaTabelEkspor mengubah hasil kueri menjadi judul kolom beserta barisnya.
func bacaTabelEkspor(rows *sql.Rows) (reportkpi.PICExportTable, error) {
	names, err := rows.Columns()
	if err != nil {
		return reportkpi.PICExportTable{}, fmt.Errorf("%w: membaca kolom ekspor: %w", ErrQueryFailed, err)
	}

	table := reportkpi.PICExportTable{Header: names}
	cell := make([]any, len(names))
	into := make([]any, len(names))
	for i := range cell {
		into[i] = &cell[i]
	}

	for rows.Next() {
		if err := rows.Scan(into...); err != nil {
			return reportkpi.PICExportTable{}, fmt.Errorf("%w: memindai baris ekspor: %w", ErrQueryFailed, err)
		}
		record := make([]string, len(names))
		for i, name := range names {
			record[i] = selEkspor(cell[i], kolomTahun[name])
		}
		table.Rows = append(table.Rows, record)
	}
	if err := rows.Err(); err != nil {
		return reportkpi.PICExportTable{}, fmt.Errorf("%w: membaca baris ekspor: %w", ErrQueryFailed, err)
	}
	return table, nil
}

// selEkspor menuliskan satu sel.
//
// Tanggal diformat di sini, bukan di SQL — itu yang memungkinkan kuerinya portabel. Nilai
// kosong ditulis sebagai teks kosong, bukan "NULL" maupun tanda hubung: berkas ini dibaca
// mesin dan dibandingkan dengan berkas Pega, bukan dibaca manusia di layar.
func selEkspor(value any, tahunSaja bool) string {
	switch v := value.(type) {
	case nil:
		return ""
	case time.Time:
		if tahunSaja {
			return v.Format("2006")
		}
		return v.Format(tanggalTampil)
	case []byte:
		return string(v)
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		if v {
			return "1"
		}
		return "0"
	default:
		return fmt.Sprint(v)
	}
}
