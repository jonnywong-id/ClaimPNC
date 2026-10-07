// Package sqlstore memenuhi seam masterstatusprogres.Repo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/masterstatusprogres"
	"claim-pnc/internal/platform/sqlfile"
	"claim-pnc/internal/platform/sqlkit"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = sqlfile.MustLoad(queryFiles, "masterstatusprogres/sqlstore")

// getQuery mengembalikan teks SQL bernama tertentu; ia panik bila namanya tidak ada
// (lihat sqlfile.MustGet).
func getQuery(name string) string {
	return sqlfile.MustGet(query, "masterstatusprogres/sqlstore", name)
}

// Repo membaca dan menulis POOLDATA.GCNM_MST_PROGRESS_KLAIM.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca seluruh status progres.
func (r *Repo) List(ctx context.Context) ([]masterstatusprogres.ProgressStatus, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("progress_status_list"))
	return sqlkit.Collect(rows, err, scanRow, "masterstatusprogres/sqlstore: membaca daftar", "", "masterstatusprogres/sqlstore: menelusuri daftar")
}

// Get membaca satu status progres berdasarkan ID-nya.
func (r *Repo) Get(ctx context.Context, id string) (masterstatusprogres.ProgressStatus, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("progress_status_get"), id)

	sp, err := scanSingleRow(rows)
	if errors.Is(err, sql.ErrNoRows) {
		return masterstatusprogres.ProgressStatus{}, masterstatusprogres.ErrNotFound
	}
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("masterstatusprogres/sqlstore: membaca %q: %w", id, err)
	}
	return sp, nil
}

// InsertNew menurunkan ID dari isi tabel lalu menyisipkan barisnya.
//
// Keduanya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi:
// nomor baru diturunkan dari isi tabel itu sendiri, sehingga membaca dan menulisnya
// tidak dapat dipisahkan tanpa membuka kembali lubang balapan yang justru sedang
// ditutup. Alasannya dicatat di docs/keputusan-implementasi.md.
func (r *Repo) InsertNew(ctx context.Context, input masterstatusprogres.Input) (masterstatusprogres.ProgressStatus, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("masterstatusprogres/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	used, err := usedIDs(ctx, tx)
	if err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}

	id := nextID(used)
	if _, err := tx.ExecContext(ctx, getQuery("progress_status_insert"), id, input.Name, input.PositionCode); err != nil {
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("masterstatusprogres/sqlstore: menyisipkan %q: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return masterstatusprogres.ProgressStatus{}, fmt.Errorf("masterstatusprogres/sqlstore: menutup transaksi sisip: %w", err)
	}

	return masterstatusprogres.ProgressStatus{ID: id, Name: input.Name, PositionCode: input.PositionCode}, nil
}

// Update menyimpan perubahan pada baris yang sudah ada.
func (r *Repo) Update(ctx context.Context, sp masterstatusprogres.ProgressStatus) error {
	result, err := r.db.ExecContext(ctx, getQuery("progress_status_update"), sp.Name, sp.PositionCode, sp.ID)
	if err != nil {
		return fmt.Errorf("masterstatusprogres/sqlstore: memperbarui %q: %w", sp.ID, err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
		// kegagalan: pernyataannya sendiri sudah berhasil.
		return nil
	}
	if affected == 0 {
		return masterstatusprogres.ErrNotFound
	}
	return nil
}

// CheckTable memastikan tabelnya ada dan dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("progress_status_check_table"))
	if err != nil {
		return fmt.Errorf("masterstatusprogres/sqlstore: POOLDATA.GCNM_MST_PROGRESS_KLAIM tidak dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// usedIDs mengunci baris yang ada lalu mengembalikan seluruh ID-nya.
func usedIDs(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx, getQuery("progress_status_list_id_locked"))
	if err != nil {
		return nil, fmt.Errorf("masterstatusprogres/sqlstore: mengunci daftar ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var used []string
	for rows.Next() {
		var id sql.NullString
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("masterstatusprogres/sqlstore: membaca ID: %w", err)
		}
		used = append(used, strings.TrimSpace(id.String))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("masterstatusprogres/sqlstore: menelusuri daftar ID: %w", err)
	}
	return used, nil
}

// nextID menyusun ID baru yang belum dipakai.
//
// # Kenapa nomornya dihitung di Go, bukan dengan MAX di SQL
//
// Kueri lama memakai `NVL(MAX(A.ID_PROGRESS),0)+1`. Bila ID_PROGRESS bertipe VARCHAR2,
// MAX-nya adalah maksimum LEKSIKOGRAFIS — dan begitu tabel memuat "010", maksimumnya
// tetap "09" karena '9' > '1' pada karakter kedua. Nomor berikutnya kembali menjadi 10,
// dan ID "010" diterbitkan dua kali. Tipe kolomnya sendiri belum diketahui karena DDL
// tidak ada di export (R-08), sehingga cacat itu mungkin sudah aktif hari ini atau
// mungkin tidak — bergantung pada tipe yang dipilih DBA dahulu.
//
// Menghitungnya di Go menghindari pertanyaan itu seluruhnya: setiap ID ditafsirkan
// sebagai angka, diambil yang terbesar, lalu ditambah satu. Untuk rentang yang kedua
// cara sepakat — satu sampai sembilan baris — hasilnya sama persis dengan Pega,
// sehingga uji kesetaraan tidak melihat selisih. Sekaligus memenuhi aturan Steering
// bahwa pemformatan angka dilakukan di Go, bukan di SQL.
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
func nextID(used []string) string {
	taken := make(map[string]bool, len(used))
	highest := 0
	for _, id := range used {
		taken[id] = true
		if number, err := strconv.Atoi(strings.TrimSpace(id)); err == nil && number > highest {
			highest = number
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk kasus
	// tabel yang sudah memuat ID berbentuk lain — misalnya "010" yang diterbitkan cacat
	// MAX leksikografis kueri lama — supaya baris baru tidak menabraknya.
	for number := highest + 1; ; number++ {
		candidate := masterstatusprogres.FormatID(number)
		if !taken[candidate] {
			return candidate
		}
	}
}

type scanner = sqlkit.Scanner

// scanRow membaca satu baris hasil kueri menjadi ProgressStatus.
//
// Ketiga kolom dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom yang bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena tabel ini tidak punya constraint NOT NULL yang
// diketahui (R-08).
func scanRow(p scanner) (masterstatusprogres.ProgressStatus, error) {
	var id, name, positionCode sql.NullString
	if err := p.Scan(&id, &name, &positionCode); err != nil {
		return masterstatusprogres.ProgressStatus{}, err
	}
	return masterstatusprogres.ProgressStatus{
		ID:           strings.TrimSpace(id.String),
		Name:         strings.TrimSpace(name.String),
		PositionCode: strings.TrimSpace(positionCode.String),
	}, nil
}

func scanSingleRow(rows *sql.Row) (masterstatusprogres.ProgressStatus, error) {
	return scanRow(rows)
}

var _ masterstatusprogres.Repo = (*Repo)(nil)
