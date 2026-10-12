package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"claim-pnc/internal/masterdominanfactor"
)

// Repo membaca dan menulis POOLDATA.M_DOMINAN_FACTOR.
//
// # Kenapa modul ini boleh MENULIS ke tabel milik sistem lama
//
// `P-1` menetapkan satu tabel hanya boleh ditulis satu sistem selama masa paralel — bukan
// bahwa tabel lama tidak boleh ditulis sama sekali. Layar Master Dominan Factor adalah
// SATU-SATUNYA penulis M_DOMINAN_FACTOR di sistem lama: `RDB List/InsertDominanfactor-SQL.xml`
// adalah satu-satunya pemanggil `POOLDATA.PEGA_M_DOMINAN_FACTOR` di seluruh export.
// Memindahkan layar itu ke sini memindahkan kepemilikan tabelnya secara utuh, dan Pega
// berubah menjadi pembaca saja.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh faktor dominan, terurut numerik menurut ID.
func (r *Repo) List(ctx context.Context) ([]masterdominanfactor.DominantFactor, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("dominant_factor_list"))
	if err != nil {
		return nil, fmt.Errorf("masterdominanfactor/sqlstore: membaca daftar faktor dominan: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []masterdominanfactor.DominantFactor
	for rows.Next() {
		factor, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("masterdominanfactor/sqlstore: membaca baris faktor dominan: %w", err)
		}
		result = append(result, factor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterdominanfactor/sqlstore: menelusuri daftar faktor dominan: %w", err)
	}

	// Diurutkan di sini, bukan di SQL: ID tidak bernol di depan, sehingga pengurutan
	// teks menempatkan `10` sebelum `9`. Alasan lengkapnya di masterdominanfactor.SortByID.
	masterdominanfactor.SortByID(result)
	return result, nil
}

// Get membaca satu faktor dominan.
func (r *Repo) Get(ctx context.Context, id string) (masterdominanfactor.DominantFactor, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("dominant_factor_get"), id)

	factor, err := scanRow(rows)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrNotFound
	case err != nil:
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: membaca faktor dominan: %w", err)
	}
	return factor, nil
}

// Insert menyimpan faktor baru dengan ID yang dibentuk seperti procedure lama.
//
// Ketiga langkahnya berada dalam SATU transaksi, dan itu memperbaiki dua cacat nyata
// sistem lama sekaligus:
//
//  1. `PEGA_M_DOMINAN_FACTOR.prc` menjalankan COMMIT sendiri di dalam cabang INSERT
//     (`:15`), sementara satu-satunya ROLLBACK-nya berada di handler terluar yang
//     berjalan SESUDAH commit itu — sehingga tidak memulihkan apa pun. `D-68` menetapkan
//     kepemilikan transaksi berpindah ke Go persis karena pola seperti ini.
//  2. `max(ID)+1` tanpa kunci apa pun. Dua penyimpanan yang tiba bersamaan sama-sama
//     membaca nomor tertinggi yang sama, lalu sama-sama menyisipkannya. Di sini
//     pembacaannya memakai FOR UPDATE, sehingga yang kedua menunggu yang pertama selesai.
func (r *Repo) Insert(ctx context.Context, name string) (masterdominanfactor.DominantFactor, error) {
	transaction, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback pada jalur gagal. Setelah Commit berhasil, panggilan ini tidak berakibat
	// apa-apa — itulah sebabnya ia aman ditaruh sebagai defer tanpa percabangan.
	defer func() { _ = transaction.Rollback() }()

	existing, err := lockExistingIDs(ctx, transaction)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}

	id := masterdominanfactor.NextID(existing)

	// Pemeriksaan terakhir sebelum menyisipkan.
	//
	// Nama constraint kunci utama tabel ini TIDAK DIKETAHUI — DDL-nya tidak ada di export
	// (`R-08`), sehingga galat bentrok dari basis data tidak dapat diterjemahkan dengan
	// mencocokkan namanya seperti yang dilakukan modul Master Status Klaim. Pemeriksaan
	// di sini menggantikannya, dan ia dapat diandalkan karena seluruh ID sudah dibaca
	// DAN dikunci di dalam transaksi yang sama.
	for _, taken := range existing {
		if taken == id {
			return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrIDTaken
		}
	}

	if _, err := transaction.ExecContext(ctx, getQuery("dominant_factor_insert"), id, name); err != nil {
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: menyisipkan faktor dominan: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: menyimpan faktor dominan baru: %w", err)
	}

	return masterdominanfactor.DominantFactor{ID: id, Name: name}, nil
}

// Update mengganti nama faktor yang sudah ada.
func (r *Repo) Update(ctx context.Context, id, name string) (masterdominanfactor.DominantFactor, error) {
	result, err := r.db.ExecContext(ctx, getQuery("dominant_factor_update"), name, id)
	if err != nil {
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: mengubah faktor dominan: %w", err)
	}

	// Baris yang tersentuh diperiksa, bukan diabaikan: UPDATE terhadap ID yang tidak ada
	// berhasil tanpa galat di SQL, dan membiarkannya akan melaporkan "tersimpan" atas
	// perubahan yang tidak pernah terjadi. Procedure lama melakukan persis itu — ia
	// mengembalikan `'Data Sudah Diupdate dengan ID : ' || id` (`:23`) tanpa memeriksa
	// satu baris pun tersentuh.
	touched, err := result.RowsAffected()
	if err != nil {
		return masterdominanfactor.DominantFactor{}, fmt.Errorf("masterdominanfactor/sqlstore: membaca jumlah baris terubah: %w", err)
	}
	if touched == 0 {
		return masterdominanfactor.DominantFactor{}, masterdominanfactor.ErrNotFound
	}

	// Dibaca ulang supaya yang dikembalikan adalah isi baris yang sebenarnya, termasuk
	// perapian yang dilakukan Clean atas nilai yang tersimpan.
	return r.Get(ctx, id)
}

// lockExistingIDs membaca seluruh ID sambil menguncinya di dalam transaksi berjalan.
//
// Nilai yang dikembalikan dipakai dua kali: menghitung ID berikutnya, dan memastikan
// nomor itu belum dipakai.
func lockExistingIDs(ctx context.Context, transaction *sql.Tx) ([]string, error) {
	rows, err := transaction.QueryContext(ctx, getQuery("dominant_factor_lock_ids"))
	if err != nil {
		return nil, fmt.Errorf("masterdominanfactor/sqlstore: mengunci daftar ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterdominanfactor/sqlstore: membaca ID terkunci: %w", err)
		}
		// Dirapikan di sini juga: bila kolomnya ternyata CHAR, nilainya kembali membawa
		// padding dan `"1  "` tidak akan dikenali sama dengan `"1"`.
		result = append(result, masterdominanfactor.DominantFactor{ID: id.String}.Clean().ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterdominanfactor/sqlstore: menelusuri ID terkunci: %w", err)
	}
	return result, nil
}

// rowScanner menyatukan *sql.Row dan *sql.Rows, yang keduanya punya Scan dengan bentuk
// sama tetapi tidak berbagi satu antarmuka di pustaka standar.
type rowScanner interface{ Scan(to ...any) error }

func scanRow(rows rowScanner) (masterdominanfactor.DominantFactor, error) {
	var (
		id   string
		name sql.NullString
	)
	if err := rows.Scan(&id, &name); err != nil {
		return masterdominanfactor.DominantFactor{}, err
	}
	// NAME dibaca sebagai NullString karena kolomnya tidak diketahui NOT NULL atau bukan
	// (`R-08`), dan modul ini memang MENERIMA nama kosong (keputusan Work Owner
	// 2026-09-20). NULL dan string kosong diperlakukan sama.
	return masterdominanfactor.DominantFactor{
		ID:   id,
		Name: name.String,
	}.Clean(), nil
}

var _ masterdominanfactor.Repo = (*Repo)(nil)

// CheckTable menguji apakah tabel ada dan kedua kolomnya dapat dibaca akun aplikasi,
// tanpa mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: tabelnya tidak ada di portal itu, versus akun aplikasi
// tidak punya hak baca atas tabel warisan.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("dominant_factor_check_table"))
	if err != nil {
		return err
	}
	return rows.Close()
}
