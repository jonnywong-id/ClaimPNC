package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterreas"
)

// Repo membaca POOLDATA.T_REINSURER.
//
// **Hanya membaca.** Tidak ada satu pun method yang menulis, dan itu keputusan berdasar
// bukti — lihat banner paket masterreas dan banner berkas masterreas.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel —
// lihat banner berkas .sql untuk alasannya.
func (r *Repo) List(
	ctx context.Context,
	filter masterreas.Filter,
) ([]masterreas.Member, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("reas_list"))
	} else {
		// Nilai yang sama dikirim empat kali; lihat catatan pada reas_list_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("reas_list_search"),
			pattern, pattern, pattern, pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("masterreas/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterreas.Member
	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterreas/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// CountAll mencacah seluruh baris. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_all")
}

// CountDuplicateKey mencacah kunci alami — REINSURERID + REINSURERNAME + TYPE — yang dipakai
// lebih dari satu baris.
func (r *Repo) CountDuplicateKey(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_duplicate_key")
}

// CountSharedLogin mencacah LOGIN yang dipakai lebih dari satu REINSURERID.
//
// Pemeriksaan bertaruh paling tinggi di modul ini: LOGIN menentukan klaim mana yang dilihat
// seorang mitra reasuransi. Lihat catatan pada reas_count_shared_login.
func (r *Repo) CountSharedLogin(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_shared_login")
}

// CountEmptyLogin mencacah baris yang LOGIN-nya kosong atau NULL.
func (r *Repo) CountEmptyLogin(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_empty_login")
}

// CountMissingEmail mencacah baris yang EMAIL-nya kosong atau NULL.
func (r *Repo) CountMissingEmail(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_missing_email")
}

// CountWithoutFallback mencacah perusahaan reasuransi yang tidak punya baris ber-TYPE '1'.
func (r *Repo) CountWithoutFallback(ctx context.Context) (int, error) {
	return r.count(ctx, "reas_count_without_fallback")
}

// CheckTable membuktikan tabel beserta keenam kolom yang dibaca modul ini ada dan dapat
// dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("reas_check_table"))
	if err != nil {
		return fmt.Errorf("masterreas/sqlstore: memeriksa tabel member reas: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name)).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterreas/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// likePattern menyiapkan pola pencarian LIKE dari kata kunci pengguna.
//
// Tanpa pelolosan, pengguna yang mengetik "%" akan mencocokkan SELURUH baris, dan yang
// mengetik "_" akan mencocokkan sembarang satu karakter — keduanya diam-diam, tanpa satu pun
// tanda bahwa yang dicari bukan yang diketik.
//
// `ESCAPE '\'` disebut eksplisit di kuerinya karena Oracle tidak punya karakter pelolos
// bawaan pada LIKE.
//
// Urutannya menentukan: garis miring terbalik diloloskan LEBIH DULU, sebelum persen dan
// garis bawah. Membaliknya akan meloloskan garis miring yang baru saja ditambahkan, dan
// pola yang dihasilkan tidak lagi cocok dengan apa pun.
func likePattern(keyword string) string {
	escaped := strings.ToUpper(strings.TrimSpace(keyword))
	for _, special := range []string{`\`, `%`, `_`} {
		escaped = strings.ReplaceAll(escaped, special, `\`+special)
	}
	return "%" + escaped + "%"
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun
// di pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat
// yang dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris menjadi Member.
//
// Keenam kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui
// (`R-08`) — `GetListDataLoginReas` bahkan menyisipkan baris TANPA kolom COUNTRY dan TYPE
// sama sekali, sehingga keduanya pasti NULL pada baris yang lahir dari jalur itu.
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada reas_list, reas_list_search, dan
// reas_check_table. Ketiganya menyebut kolom yang sama pada urutan yang sama; itu yang
// membuat satu fungsi cukup untuk ketiganya.
func scanRow(row rowScanner) (masterreas.Member, error) {
	var (
		reinsurerID   sql.NullString
		reinsurerName sql.NullString
		login         sql.NullString
		email         sql.NullString
		country       sql.NullString
		memberType    sql.NullString
	)
	if err := row.Scan(&reinsurerID, &reinsurerName, &login, &email,
		&country, &memberType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterreas.Member{}, err
		}
		return masterreas.Member{}, fmt.Errorf(
			"masterreas/sqlstore: membaca baris: %w", err)
	}

	return masterreas.Member{
		ReinsurerID:   strings.TrimSpace(reinsurerID.String),
		ReinsurerName: strings.TrimSpace(reinsurerName.String),
		Login:         strings.TrimSpace(login.String),
		Email:         strings.TrimSpace(email.String),
		Country:       strings.TrimSpace(country.String),
		Type:          strings.TrimSpace(memberType.String),
	}, nil
}

var _ masterreas.Repo = (*Repo)(nil)
