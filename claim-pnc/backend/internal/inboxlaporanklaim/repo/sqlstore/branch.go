package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxlaporanklaim"
)

// BranchResolver memenuhi seam inboxlaporanklaim.BranchResolver dengan SQL.
//
// Ia terpisah dari Repo dengan sengaja: Repo membaca berkas laporan di basis data satu
// entitas, sedangkan penerjemah ini membaca HRD dan master pengguna asuransi lewat DB
// Link. Keduanya dapat gagal sendiri-sendiri, dan menyatukannya membuat kegagalan yang
// satu tampak seperti kegagalan yang lain.
type BranchResolver struct {
	db *sql.DB
}

// NewBranchResolver membentuk penerjemah; db wajib sudah terhubung.
func NewBranchResolver(db *sql.DB) *BranchResolver {
	return &BranchResolver{db: db}
}

// Resolve menerjemahkan login petugas menjadi kode cabang klaimnya.
//
// Tiga keluaran yang dibedakan, dan pemanggil memperlakukannya berbeda:
//
//	("1001", true,  nil)  cabangnya ditemukan
//	("",     false, nil)  petugasnya tidak terdaftar di HRD — bukan galat
//	("",     false, err)  sumbernya tidak dapat dibaca — DB Link mati, hak akses kurang
//
// Baris kedua terjadi untuk petugas non-karyawan: broker dan surveyor independen masuk
// lewat POOLDATA.M_LOGIN_PNC dan memang tidak pernah ada di HRD.
func (r *BranchResolver) Resolve(ctx context.Context, login string) (string, bool, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return "", false, nil
	}

	var code sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("branch_of_login"), clean).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("inboxlaporanklaim/sqlstore: menerjemahkan cabang %q: %w", clean, err)
	}

	// Kolom CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa
	// pun. Kode yang tidak dipangkas tidak akan pernah cocok dengan penyaringnya sendiri
	// — dan itu kelas cacat yang sama dengan yang sedang diperbaiki di sini.
	value := strings.TrimSpace(code.String)
	if value == "" {
		return "", false, nil
	}
	return value, true, nil
}

// CheckTable memastikan ketiga objek yang dibutuhkan dapat dibaca akun aplikasi.
//
// Ia dijalankan dengan login karangan yang pasti tidak ada, sehingga tidak mengembalikan
// satu baris pun — yang diuji adalah KETERBACAAN objeknya, bukan isinya. Dua di antaranya
// berada di basis data lain lewat DB Link, dan kegagalannya adalah kelas kegagalan
// tersendiri: bukan "migrasi belum jalan", melainkan "sambungan ke HRD tidak hidup".
func (r *BranchResolver) CheckTable(ctx context.Context) error {
	var code sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("branch_of_login"), "__periksa__").Scan(&code)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("inboxlaporanklaim/sqlstore: BRANCH, LST_USER_ASURANSI, atau V_HRD_MST tidak dapat dibaca: %w", err)
	}
	return nil
}

var _ inboxlaporanklaim.BranchResolver = (*BranchResolver)(nil)
