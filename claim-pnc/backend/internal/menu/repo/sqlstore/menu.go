// Package sqlstore memenuhi seam menu.Repo dengan SQL.
//
// Ia terikat pada koneksi portal UTAMA, bukan pada koneksi entitas yang sedang dipilih.
// Alasannya sama dengan POOLDATA.M_LOGIN_PNC dan M_PORTAL_PNC yang sudah dibaca dari
// sana: peta menu dan kewenangan pemakainya adalah data lingkup IDENTITAS, bukan data
// bisnis milik satu badan hukum. Keempat tabelnya pun tidak punya kolom entitas.
//
// Akibat yang disengaja: menu seseorang SAMA di keempat portal. Berpindah portal
// mengubah data yang dibaca layar, bukan daftar layar yang boleh ia buka.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/menu"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// subjectsMarker adalah penanda di dalam menu_authorized_ids yang diganti daftar
// penanda parameter. Lihat expandSubjects.
const subjectsMarker = "/*SUBJECTS*/"

// Repo membaca peta menu dan kewenangannya.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal utama.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh butir menu sebuah aplikasi, terurut MENU_SEQUENCE.
func (r *Repo) List(ctx context.Context, appName string) ([]menu.Item, error) {
	// Keberadaan aplikasinya diperiksa lebih dulu supaya "APP_DESC salah ketik" tidak
	// terbaca sebagai "aplikasinya memang belum punya menu". Keduanya sama-sama
	// menghasilkan nol baris, dan hanya yang pertama yang perlu diperbaiki orang.
	if err := r.ensureApp(ctx, appName); err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, getQuery("menu_list"), appName)
	if err != nil {
		return nil, fmt.Errorf("menu/sqlstore: membaca daftar menu: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []menu.Item
	for rows.Next() {
		var (
			id       sql.NullInt64
			desc     sql.NullString
			program  sql.NullString
			leader   sql.NullInt64
			sequence sql.NullInt64
		)
		if err := rows.Scan(&id, &desc, &program, &leader, &sequence); err != nil {
			return nil, fmt.Errorf("menu/sqlstore: membaca baris menu: %w", err)
		}

		item := menu.Item{
			ID:          int(id.Int64),
			Description: strings.TrimSpace(desc.String),
			Program:     strings.TrimSpace(program.String),
			Sequence:    int(sequence.Int64),
		}
		// NULL pada MENU_ID_LEADER berarti kelompok tingkat atas. Ia dibedakan dari
		// nol dengan sengaja: nol adalah MENU_ID yang sah secara tipe, dan memperlakukan
		// keduanya sama akan menggantung butir yang berinduk MENU_ID 0.
		if leader.Valid {
			parent := int(leader.Int64)
			item.ParentID = &parent
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menu/sqlstore: menelusuri daftar menu: %w", err)
	}
	return result, nil
}

// GroupsOf membaca GROUP_ID yang diikuti sebuah login.
func (r *Repo) GroupsOf(ctx context.Context, loginID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("menu_groups_of_login"), loginID)
	if err != nil {
		return nil, fmt.Errorf("menu/sqlstore: membaca group login: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var group sql.NullString
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("menu/sqlstore: membaca group: %w", err)
		}
		if clean := strings.TrimSpace(group.String); clean != "" {
			result = append(result, clean)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menu/sqlstore: menelusuri group login: %w", err)
	}
	return result, nil
}

// AuthorizedIDs membaca MENU_ID yang diizinkan untuk subjek-subjek yang disebut.
func (r *Repo) AuthorizedIDs(ctx context.Context, appName string, subjects []string) ([]int, error) {
	if len(subjects) == 0 {
		// Tanpa satu pun subjek, tidak ada yang perlu ditanyakan ke basis data — dan
		// `IN ()` bukan SQL yang sah di dialek mana pun.
		return nil, nil
	}

	statement, args := expandSubjects(getQuery("menu_authorized_ids"), appName, subjects)

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("menu/sqlstore: membaca otorisasi menu: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []int
	for rows.Next() {
		var id sql.NullInt64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("menu/sqlstore: membaca MENU_ID: %w", err)
		}
		if id.Valid {
			result = append(result, int(id.Int64))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menu/sqlstore: menelusuri otorisasi menu: %w", err)
	}
	return result, nil
}

// CheckTable memastikan keempat tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("menu_check_table"))
	if err != nil {
		return fmt.Errorf("menu/sqlstore: tabel menu dan otorisasi tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

func (r *Repo) ensureApp(ctx context.Context, appName string) error {
	rows, err := r.db.QueryContext(ctx, getQuery("menu_app_exists"), appName)
	if err != nil {
		return fmt.Errorf("menu/sqlstore: memeriksa aplikasi %q: %w", appName, err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("menu/sqlstore: memeriksa aplikasi %q: %w", appName, err)
		}
		return fmt.Errorf("%w: APP_DESC %q", menu.ErrAppNotFound, appName)
	}
	return rows.Err()
}

// expandSubjects mengganti penanda /*SUBJECTS*/ dengan daftar penanda parameter.
//
// # Kenapa daftarnya disusun di Go, bukan ditulis tetap di berkas .sql
//
// Banyaknya subjek berbeda tiap pengguna: satu login dapat tergabung di nol group atau
// beberapa. Menuliskannya tetap berarti memilih jumlah maksimum lalu mengisi sisanya
// dengan nilai palsu.
//
// # Kenapa ini BUKAN perangkaian SQL yang dilarang
//
// Yang disisipkan hanyalah `:2, :3, …` — PENANDA parameter, bukan nilainya. Seluruh
// nilai tetap dikirim terpisah lewat args, sehingga celah `{ASIS:...}` warisan
// (`03-CURRENT-ARCHITECTURE.md` §4.5) tetap tertutup. Uji di query_test.go menjaga
// pernyataan itu tetap benar.
//
// Subjek diseragamkan menjadi huruf besar di sisi Go supaya cocok dengan
// UPPER(TRIM(...)) di sisi SQL — perbandingan yang satu sisinya saja diseragamkan tidak
// pernah cocok, dan gagalnya diam.
func expandSubjects(statement, appName string, subjects []string) (string, []any) {
	marks := make([]string, 0, len(subjects))
	args := make([]any, 0, len(subjects)+1)
	args = append(args, appName)

	for i, subject := range subjects {
		// Parameter pertama sudah dipakai appName, sehingga subjek mulai dari :2.
		marks = append(marks, ":"+strconv.Itoa(i+2))
		args = append(args, strings.ToUpper(strings.TrimSpace(subject)))
	}

	return strings.Replace(statement, subjectsMarker, strings.Join(marks, ", "), 1), args
}

// getQuery mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func getQuery(name string) string {
	text, existing := query[name]
	if !existing {
		panic(fmt.Sprintf("menu/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("menu/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("menu/sqlstore: tidak dapat membaca " + file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("menu/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memisahkan isi berkas menjadi pernyataan bernama, membuang baris komentar
// supaya yang dikirim ke basis data hanyalah SQL-nya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name != "" {
			if text := strings.TrimSpace(strings.Join(body, "\n")); text != "" {
				result[name] = text
			}
		}
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		if name == "" || strings.HasPrefix(trimmed, "--") {
			// Komentar kepala berkas dan komentar penjelas tiap kueri tidak ikut
			// dikirim: yang dibaca DBA adalah berkasnya, bukan jejak di basis data.
			continue
		}
		body = append(body, line)
	}
	save()
	return result
}

var _ menu.Repo = (*Repo)(nil)
