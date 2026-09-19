package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/auth/provider"
)

// Legacy membaca tabel milik sistem lama yang dipakai saat masuk.
//
// Dua tabel, keduanya **hanya dibaca**: POOLDATA.M_LOGIN_PNC untuk login non-karyawan,
// dan POOLDATA.GCNM_CONNECT_REST untuk alamat layanan HCC/HCQ. Penulisnya tetap sistem
// yang sekarang memilikinya (ADR-0004, penulis tunggal per tabel) — aplikasi ini tidak
// pernah menulis ke sana.
type Legacy struct {
	db *sql.DB
}

// NewLegacy membentuk repo; db wajib sudah terhubung ke basis data portal utama.
func NewLegacy(db *sql.DB) *Legacy { return &Legacy{db: db} }

// FindActive memenuhi provider.LoginListRepo.
func (w *Legacy) FindActive(ctx context.Context, loginID, passwordFingerprint string) (provider.LocalLogin, error) {
	rows := w.db.QueryRowContext(ctx, getQuery("local_login_find_active"), loginID, passwordFingerprint)

	var result provider.LocalLogin
	err := rows.Scan(&result.LoginID, &result.LoginName)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return provider.LocalLogin{}, provider.ErrLoginMismatch
	case err != nil:
		// Galat sengaja tidak memuat sidik kata sandi maupun login yang dicoba.
		return provider.LocalLogin{}, fmt.Errorf("sqlstore: membaca daftar login: %w", err)
	}
	return result, nil
}

// ServiceAddress memenuhi provider.ServiceCatalog.
func (w *Legacy) ServiceAddress(ctx context.Context, app, serviceKind string) (string, error) {
	rows := w.db.QueryRowContext(ctx, getQuery("service_address"), app, serviceKind)

	var address string
	err := rows.Scan(&address)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", provider.ErrServiceNotRegistered
	case err != nil:
		return "", fmt.Errorf("sqlstore: membaca alamat layanan: %w", err)
	}
	if strings.TrimSpace(address) == "" {
		return "", provider.ErrServiceNotRegistered
	}
	return strings.TrimSpace(address), nil
}

// Pastikan Legacy benar-benar memenuhi kedua seam yang dipakai provider. Pemeriksaan
// ini terjadi saat kompilasi, bukan saat pengguna pertama mencoba masuk.
var (
	_ provider.LoginListRepo  = (*Legacy)(nil)
	_ provider.ServiceCatalog = (*Legacy)(nil)
)

// CheckTable menguji apakah sebuah tabel ada dan dapat dibaca akun aplikasi, tanpa
// mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: tabel belum dibuat (migrasi belum dijalankan DBA) versus
// akun aplikasi tidak punya hak baca.
func (w *Legacy) CheckTable(ctx context.Context, queryName string) error {
	rows, err := w.db.QueryContext(ctx, getQuery(queryName))
	if err != nil {
		return err
	}
	return rows.Close()
}

// DB membuka koneksi yang dipegang repo ini supaya repo lain dapat dipasang di atas
// koneksi yang sama, tanpa perakit di cmd perlu memegang dua rujukan ke benda yang
// sama.
func (w *Legacy) DB() *sql.DB { return w.db }
