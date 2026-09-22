package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// LineBusinessRepo membaca POOLDATA.M_LOGIN_PNC.LINEBUSINESS.
//
// Tabel itu MILIK SISTEM LAMA dan ditulis Pega. Repo ini hanya membaca, sehingga `P-1`
// terpenuhi. Kolom LINEBUSINESS sendiri ditambahkan migrasi 0004 dan belum ada — lihat
// kepala linebusiness.sql.
type LineBusinessRepo struct {
	db *sql.DB
}

// NewLineBusinessRepo membentuk repo.
//
// # Kenapa basis data PORTAL UTAMA, bukan portal aktif
//
// `M_LOGIN_PNC` adalah master pengguna, dan pengguna tidak berpindah identitas saat
// berpindah portal — `D-78` menetapkan satu login berlaku di keempat entitas. Membacanya
// dari basis data portal aktif akan membuat lini bisnis seseorang berubah-ubah mengikuti
// portal yang sedang dibuka.
//
// Pilihan ini sama dengan yang sudah dipakai modul menu dan modul auth untuk tabel yang
// sama; pemanggil yang menentukan db mana yang diberikan.
func NewLineBusinessRepo(db *sql.DB) *LineBusinessRepo { return &LineBusinessRepo{db: db} }

// LineBusinessFor memenuhi inboxoutstanding.LineBusinessRepo.
//
// # Tiga keadaan, dan hanya satu yang dianggap kegagalan
//
//	baris ada, kolom terisi     -> nilainya
//	baris tidak ada             -> "" tanpa galat
//	baris ada, kolom NULL       -> "" tanpa galat
//	kegagalan pembacaan         -> galat
//
// Dua keadaan tengah BUKAN kegagalan: pengguna yang belum dilengkapi admin adalah keadaan
// yang diharapkan selama kolomnya baru saja ada, dan karyawan yang belum didaftarkan di
// M_LOGIN_PNC adalah keadaan yang berlaku bagi hampir semua orang hari ini.
//
// Keduanya menghasilkan batas data "tanpa batas" — perilaku yang Work Owner tetapkan
// ditiru dari Pega apa adanya.
func (r *LineBusinessRepo) LineBusinessFor(ctx context.Context, loginID string) (string, error) {
	trimmed := strings.TrimSpace(loginID)
	if trimmed == "" {
		return "", nil
	}

	var line sql.NullString
	err := r.db.QueryRowContext(ctx, query("line_business_for"), trimmed).Scan(&line)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", nil
	case err != nil:
		return "", fmt.Errorf("inboxoutstanding/sqlstore: membaca lini bisnis: %w", err)
	}

	return strings.TrimSpace(line.String), nil
}

// TableReady memeriksa apakah kolom LINEBUSINESS sudah ada.
//
// Dipakai mode `-periksa` supaya pertanyaan "apakah migrasi 0004 sudah dijalankan?" dapat
// dijawab tanpa menjalankan server dan tanpa menebak. Tanpa ini, satu-satunya cara
// mengetahuinya adalah membuka layar lalu memperhatikan bahwa batas datanya tidak berlaku
// — yaitu gejala yang justru paling mudah terlewat.
func (r *LineBusinessRepo) TableReady(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("line_business_check_table"))
	if err != nil {
		return fmt.Errorf("inboxoutstanding/sqlstore: kolom LINEBUSINESS belum dapat dibaca: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}
