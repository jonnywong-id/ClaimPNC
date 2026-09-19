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
func (w *Legacy) FindActive(ctx context.Context, loginID, passwordDigest string) (provider.LocalLogin, error) {
	row := w.db.QueryRowContext(ctx, loadQuery("login_lokal_cari_aktif"), loginID, passwordDigest)

	var result provider.LocalLogin
	err := row.Scan(&result.LoginID, &result.LoginName)
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
	row := w.db.QueryRowContext(ctx, loadQuery("layanan_alamat"), app, serviceKind)

	var address string
	err := row.Scan(&address)
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

// Pastikan Warisan benar-benar memenuhi kedua seam yang dipakai provider. Pemeriksaan
// ini terjadi saat kompilasi, bukan saat pengguna pertama mencoba masuk.
var (
	_ provider.LoginListRepo  = (*Legacy)(nil)
	_ provider.ServiceCatalog = (*Legacy)(nil)
)

// CheckTables menguji apakah sebuah tabel ada dan dapat dibaca akun aplikasi, tanpa
// mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: tabel belum dibuat (migrasi belum dijalankan DBA) versus
// akun aplikasi tidak punya hak baca.
func (w *Legacy) CheckTables(ctx context.Context, queryName string) error {
	row, err := w.db.QueryContext(ctx, loadQuery(queryName))
	if err != nil {
		return err
	}
	return row.Close()
}

// DB membuka koneksi yang dipegang repo ini supaya repo lain dapat dipasang di atas
// koneksi yang sama, tanpa perakit di cmd perlu memegang dua rujukan ke benda yang
// sama.
func (w *Legacy) DB() *sql.DB { return w.db }
