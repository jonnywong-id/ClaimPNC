// Package sqlstore adalah pengisi seam masterpasalai.Repo terhadap basis data.
//
// Satu instans selalu terikat pada SATU koneksi entitas; pemisahan antarentitas ada di
// tingkat koneksi, bukan di tingkat kueri (`ADR-0030` Opsi 1).
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterpasalai"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca POOLDATA.MST_PASAL_AI.
//
// Ia TIDAK punya method tulis, dan itu bukan kelalaian — lihat doc comment
// masterpasalai.Repo.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List mencacah lalu membaca satu halaman — urutan yang sama dengan activity lamanya.
//
// # Kenapa cacah lebih dulu, bukan sesudah
//
// Sama seperti `GetListPasalAI`: ia menjalankan `CountDataPasalAI` (langkah 4) SEBELUM
// `GetListDataPasalAI` (langkah 6). Urutannya dipertahankan supaya cacah tidak pernah
// dihitung atas keadaan yang lebih baru daripada barisnya.
//
// # Kenapa TIDAK di dalam satu transaksi
//
// Keduanya pembacaan, dan sistem lama pun menjalankannya sebagai dua pernyataan lepas.
// Membungkusnya dalam transaksi akan menahan kunci baca pada tabel yang dibaca setiap kali
// layar dibuka, demi ketepatan cacah yang sudah tidak berarti begitu halamannya tergambar.
//
// Akibatnya disadari: bila ada baris masuk di antara kedua kueri, cacahnya dapat meleset
// satu. Itu tidak dapat diperbaiki dari sini — layar yang menampilkan angka apa pun sudah
// menampilkan keadaan beberapa saat lalu.
func (r *Repo) List(
	ctx context.Context,
	filter masterpasalai.Filter,
) (masterpasalai.Page, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	page := masterpasalai.Page{Number: filter.Page}

	var (
		total int
		err   error
	)
	if keyword == "" {
		total, err = r.count(ctx, "clause_count")
	} else {
		pattern := likePattern(keyword)
		total, err = r.count(ctx, "clause_count_search", pattern, pattern, pattern)
	}
	if err != nil {
		return masterpasalai.Page{}, err
	}
	page.Total = total

	// Halaman di luar jangkauan tidak perlu menembak basis data sama sekali. Cacahnya sudah
	// di tangan, dan `OFFSET` di luar jangkauan menghasilkan daftar kosong yang sama —
	// hanya setelah basis data memindainya lebih dulu.
	offset := filter.Offset()
	if offset >= total {
		return page, nil
	}

	var rows *sql.Rows
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("clause_list"),
			offset, masterpasalai.PageSize)
	} else {
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("clause_list_search"),
			pattern, pattern, pattern, offset, masterpasalai.PageSize)
	}
	if err != nil {
		return masterpasalai.Page{}, fmt.Errorf("masterpasalai/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return masterpasalai.Page{}, err
		}
		page.Clause = append(page.Clause, one)
	}
	if err := rows.Err(); err != nil {
		return masterpasalai.Page{}, fmt.Errorf(
			"masterpasalai/sqlstore: menelusuri daftar: %w", err)
	}
	return page, nil
}

// CountAll mencacah seluruh baris. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "clause_count")
}

// CheckTable membuktikan tabel beserta keempat kolomnya ada dan dapat dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("clause_check_table"))
	if err != nil {
		return fmt.Errorf("masterpasalai/sqlstore: memeriksa tabel Pasal AI: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterpasalai/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyiapkan pola pencarian LIKE dari kata kunci pengguna.
//
// Tanpa pelolosan, pengguna yang mengetik "%" akan mencocokkan SELURUH baris, dan yang
// mengetik "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun
// tanda bahwa yang dicari bukan yang diketik.
//
// Kata kunci di-uppercase karena kuerinya membandingkan `UPPER(kolom)`; keduanya harus
// searah, dan menaikkannya di sini membuat basis data tidak perlu melakukannya per baris
// untuk sisi kanan.
func likePattern(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun di
// pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat yang
// dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris menjadi Clause.
//
// Keempat kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris lama
// dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui (`R-08`).
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada clause_list, clause_list_search, dan
// clause_check_table. Ketiganya menyebut kolom yang sama pada urutan yang sama; itu yang
// membuat satu fungsi cukup untuk ketiganya. Dijaga query_test.go.
func scanRow(row rowScanner) (masterpasalai.Clause, error) {
	var (
		id        sql.NullString
		number    sql.NullString
		paragraph sql.NullString
		event     sql.NullString
	)
	if err := row.Scan(&id, &number, &paragraph, &event); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterpasalai.Clause{}, err
		}
		return masterpasalai.Clause{}, fmt.Errorf(
			"masterpasalai/sqlstore: membaca baris: %w", err)
	}

	return masterpasalai.Clause{
		ID:        strings.TrimSpace(id.String),
		Number:    strings.TrimSpace(number.String),
		Paragraph: strings.TrimSpace(paragraph.String),
		Event:     strings.TrimSpace(event.String),
	}, nil
}

// getQuery mengambil pernyataan SQL menurut namanya.
//
// Ia PANIC bila namanya tidak ada, dan itu disengaja: nama kueri adalah konstanta yang ditulis
// programmer, bukan masukan pengguna. Salah ketik harus gagal saat uji pertama dijalankan,
// bukan menjadi galat runtime di hadapan petugas.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf(
			"masterpasalai/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterpasalai/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterpasalai/sqlstore: tidak dapat membaca " +
				file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterpasalai/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris komentar
// dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
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

var _ masterpasalai.Repo = (*Repo)(nil)
