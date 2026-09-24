// Package sqlstore adalah pengisi seam masterlogin.Repo terhadap basis data.
//
// Satu instans selalu terikat pada SATU koneksi entitas; pemisahan antarentitas ada di
// tingkat koneksi, bukan di tingkat kueri (ADR-0030 Opsi 1).
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterlogin"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca dan menulis POOLDATA.MST_LOGIN_SURVEYOR.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca baris yang cocok dengan penyaring.
//
// Penyaring kata kunci memakai kueri TERSENDIRI, bukan satu kueri yang klausanya ditempel —
// lihat berkas .sql untuk alasannya, yang pada modul ini lebih dari sekadar kerapian.
func (r *Repo) List(
	ctx context.Context,
	filter masterlogin.Filter,
) ([]masterlogin.SurveyorLogin, error) {
	keyword := strings.TrimSpace(filter.Keyword)

	var (
		rows *sql.Rows
		err  error
	)
	if keyword == "" {
		rows, err = r.db.QueryContext(ctx, getQuery("login_list"))
	} else {
		// Nilai yang sama dikirim tiga kali; lihat catatan pada login_list_search.
		pattern := likePattern(keyword)
		rows, err = r.db.QueryContext(ctx, getQuery("login_list_search"),
			pattern, pattern, pattern)
	}
	if err != nil {
		return nil, fmt.Errorf("masterlogin/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterlogin.SurveyorLogin
	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterlogin/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Get mengembalikan satu baris berdasarkan LOGIN-nya.
//
// Pembandingnya TIDAK di-uppercase; lihat banner berkas .sql.
func (r *Repo) Get(
	ctx context.Context,
	login string,
) (masterlogin.SurveyorLogin, error) {
	row := r.db.QueryRowContext(ctx, getQuery("login_get"), strings.TrimSpace(login))

	found, err := scanRow(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterlogin.SurveyorLogin{}, masterlogin.ErrNotFound
	case err != nil:
		return masterlogin.SurveyorLogin{}, err
	}
	return found, nil
}

// FindLeaderOf mengembalikan isi LOGINLEADER milik satu login.
//
// Baris yang tidak ada menghasilkan ErrNotFound, bukan teks kosong: keduanya berbeda arti —
// "tidak terdaftar" versus "terdaftar tetapi tidak bertim" — dan hanya pemanggil yang tahu
// mana yang boleh dianggap wajar. Lapisan aplikasi yang memutuskannya; lihat
// usecase.leaderOf.
func (r *Repo) FindLeaderOf(ctx context.Context, login string) (string, error) {
	var leader sql.NullString

	err := r.db.QueryRowContext(ctx, getQuery("login_find_leader"),
		strings.TrimSpace(login)).Scan(&leader)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", masterlogin.ErrNotFound
	case err != nil:
		return "", fmt.Errorf("masterlogin/sqlstore: membaca login leader %q: %w", login, err)
	}
	return strings.TrimSpace(leader.String), nil
}

// Insert memeriksa keunikan LOGIN lalu menyisipkan — keduanya dalam SATU transaksi yang
// menahan kunci tabel.
//
// # Urutan langkahnya, dan kenapa persis begitu
//
//  1. LOCK TABLE ... IN EXCLUSIVE MODE   tidak ada penulis lain sampai transaksi selesai
//  2. SELECT ... UPPER(TRIM(LOGIN))      login ganda ditolak
//  3. INSERT                             sisipkan
//
// Pemeriksaan berada DI DALAM transaksi, bukan sebagai langkah terpisah sebelumnya.
// Sistem lama memecahnya jauh lebih lebar daripada sekadar dua pernyataan berurutan:
// pemeriksaannya berjalan di LAYAR saat Nama diketik (`SetLoginSurveyor_act` dipicu even
// `change`), dan penyisipannya menyusul saat tombol Simpan ditekan — jarak yang diisi
// waktu pengguna mengisi tiga isian berikutnya.
func (r *Repo) Insert(
	ctx context.Context,
	one masterlogin.SurveyorLogin,
) (masterlogin.SurveyorLogin, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterlogin.SurveyorLogin{}, fmt.Errorf(
			"masterlogin/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback tanpa syarat; setelah Commit berhasil ia tidak berakibat apa pun. Tanpa ini,
	// satu jalur galat yang terlewat meninggalkan transaksi menggantung — dan karena
	// transaksi ini memegang kunci TABEL, yang tertahan bukan satu baris melainkan seluruh
	// penambahan login sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, getQuery("login_lock_table")); err != nil {
		return masterlogin.SurveyorLogin{}, fmt.Errorf(
			"masterlogin/sqlstore: mengunci tabel login surveyor: %w", err)
	}

	if err := rejectTakenLogin(ctx, tx, one.Login); err != nil {
		return masterlogin.SurveyorLogin{}, err
	}

	if _, err := tx.ExecContext(ctx, getQuery("login_insert"),
		one.Name, one.Login, one.Email, one.Phone, one.Address,
		one.LoginStatus, one.LeaderLogin); err != nil {
		return masterlogin.SurveyorLogin{}, fmt.Errorf(
			"masterlogin/sqlstore: menyisipkan %q: %w", one.Login, err)
	}

	if err := tx.Commit(); err != nil {
		return masterlogin.SurveyorLogin{}, fmt.Errorf(
			"masterlogin/sqlstore: menutup transaksi sisip: %w", err)
	}
	return one, nil
}

// rejectTakenLogin menolak LOGIN yang sudah dipakai baris mana pun.
//
// Dijalankan DI DALAM transaksi penambahan, bukan lewat kueri di luar: pemeriksaan yang
// berjalan pada koneksi lain tidak melihat kunci tabel yang sedang dipegang, sehingga
// hasilnya dapat basi sebelum penyisipannya berjalan.
func rejectTakenLogin(ctx context.Context, tx *sql.Tx, login string) error {
	row := tx.QueryRowContext(ctx, getQuery("login_find_by_key"),
		strings.ToUpper(strings.TrimSpace(login)))

	_, err := scanRow(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return err
	default:
		return masterlogin.ErrLoginTaken
	}
}

// Update menyimpan perubahan pada baris yang sudah ada.
//
// KETUJUH kolom ditulis, meniru `UpdateMasterLoginSurvey` apa adanya — termasuk LOGIN, yang
// juga menjadi penyaringnya. Nilainya selalu sama dengan penyaringnya karena lapisan
// aplikasi mengambilnya dari baris yang tersimpan; lihat catatan pada login_update.
func (r *Repo) Update(ctx context.Context, one masterlogin.SurveyorLogin) error {
	result, err := r.db.ExecContext(ctx, getQuery("login_update"),
		one.Name, one.Login, one.Email, one.Phone, one.Address,
		one.LoginStatus, one.LeaderLogin, strings.TrimSpace(one.Login))
	if err != nil {
		return fmt.Errorf("masterlogin/sqlstore: memperbarui %q: %w", one.Login, err)
	}

	// Jumlah baris terpengaruh diperiksa, bukan diabaikan: UPDATE yang tidak menyentuh satu
	// baris pun berhasil menurut basis data, dan tanpa pemeriksaan ini layar akan mengatakan
	// "tersimpan" atas baris yang sudah tidak ada.
	//
	// Driver yang tidak mendukung RowsAffected mengembalikan galat; dalam keadaan itu
	// perubahannya TIDAK dianggap gagal — pernyataannya sendiri sudah berhasil, dan
	// menolaknya akan menampilkan kegagalan palsu.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return masterlogin.ErrNotFound
	}
	return nil
}

// CountAll mencacah seluruh baris. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context) (int, error) {
	return r.count(ctx, "login_count_all")
}

// CountDuplicateKey mencacah LOGIN yang dipakai lebih dari satu baris.
func (r *Repo) CountDuplicateKey(ctx context.Context) (int, error) {
	return r.count(ctx, "login_count_duplicate_key")
}

// CountEmptyKey mencacah baris yang LOGIN-nya kosong atau NULL.
func (r *Repo) CountEmptyKey(ctx context.Context) (int, error) {
	return r.count(ctx, "login_count_empty_key")
}

// CountOrphanLeader mencacah baris yang LOGINLEADER-nya menunjuk login yang tidak ada.
func (r *Repo) CountOrphanLeader(ctx context.Context) (int, error) {
	return r.count(ctx, "login_count_orphan_leader")
}

// CountMissingContact mencacah baris yang EMAIL atau TELP-nya kosong.
func (r *Repo) CountMissingContact(ctx context.Context) (int, error) {
	return r.count(ctx, "login_count_missing_contact")
}

// CheckTable membuktikan tabel beserta ketujuh kolomnya ada dan dapat dibaca.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("login_check_table"))
	if err != nil {
		return fmt.Errorf("masterlogin/sqlstore: memeriksa tabel login surveyor: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("masterlogin/sqlstore: %s: %w", name, err)
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

// scanRow membaca satu baris menjadi SurveyorLogin.
//
// Ketujuh kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang diketahui
// (R-08) — layar lamanya pun hanya mewajibkan isian di antarmuka.
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada login_list, login_list_search,
// login_get, login_find_by_key, dan login_check_table. Kelimanya menyebut kolom yang sama
// pada urutan yang sama; itu yang membuat satu fungsi cukup untuk kelimanya.
func scanRow(row rowScanner) (masterlogin.SurveyorLogin, error) {
	var (
		name        sql.NullString
		login       sql.NullString
		email       sql.NullString
		phone       sql.NullString
		address     sql.NullString
		loginStatus sql.NullString
		leaderLogin sql.NullString
	)
	if err := row.Scan(&name, &login, &email, &phone, &address,
		&loginStatus, &leaderLogin); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return masterlogin.SurveyorLogin{}, err
		}
		return masterlogin.SurveyorLogin{}, fmt.Errorf(
			"masterlogin/sqlstore: membaca baris: %w", err)
	}

	return masterlogin.SurveyorLogin{
		Name:        strings.TrimSpace(name.String),
		Login:       strings.TrimSpace(login.String),
		Email:       strings.TrimSpace(email.String),
		Phone:       strings.TrimSpace(phone.String),
		Address:     strings.TrimSpace(address.String),
		LoginStatus: strings.TrimSpace(loginStatus.String),
		LeaderLogin: strings.TrimSpace(leaderLogin.String),
	}, nil
}

// getQuery mengambil pernyataan SQL menurut namanya.
//
// Ia PANIC bila namanya tidak ada, dan itu disengaja: nama kueri adalah konstanta yang
// ditulis programmer, bukan masukan pengguna. Salah ketik harus gagal saat uji pertama
// dijalankan, bukan menjadi galat runtime di hadapan petugas.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf(
			"masterlogin/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("masterlogin/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("masterlogin/sqlstore: tidak dapat membaca " +
				file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("masterlogin/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
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

var _ masterlogin.Repo = (*Repo)(nil)
