package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/auth/provider"
)

// Warisan membaca tabel milik sistem lama yang dipakai saat masuk.
//
// Dua tabel, keduanya **hanya dibaca**: POOLDATA.M_LOGIN_PNC untuk login non-karyawan,
// dan POOLDATA.GCNM_CONNECT_REST untuk alamat layanan HCC/HCQ. Penulisnya tetap sistem
// yang sekarang memilikinya (ADR-0004, penulis tunggal per tabel) — aplikasi ini tidak
// pernah menulis ke sana.
type Warisan struct {
	db *sql.DB
}

// WarisanBaru membentuk repo; db wajib sudah terhubung ke basis data portal utama.
func WarisanBaru(db *sql.DB) *Warisan { return &Warisan{db: db} }

// CariAktif memenuhi provider.DaftarLoginRepo.
func (w *Warisan) CariAktif(ctx context.Context, loginID, sidikKataSandi string) (provider.LoginLokal, error) {
	baris := w.db.QueryRowContext(ctx, ambilKueri("login_lokal_cari_aktif"), loginID, sidikKataSandi)

	var hasil provider.LoginLokal
	err := baris.Scan(&hasil.LoginID, &hasil.LoginNama)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return provider.LoginLokal{}, provider.ErrLoginTidakCocok
	case err != nil:
		// Galat sengaja tidak memuat sidik kata sandi maupun login yang dicoba.
		return provider.LoginLokal{}, fmt.Errorf("sqlstore: membaca daftar login: %w", err)
	}
	return hasil, nil
}

// AlamatLayanan memenuhi provider.KatalogLayanan.
func (w *Warisan) AlamatLayanan(ctx context.Context, app, jenisLayanan string) (string, error) {
	baris := w.db.QueryRowContext(ctx, ambilKueri("layanan_alamat"), app, jenisLayanan)

	var alamat string
	err := baris.Scan(&alamat)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", provider.ErrLayananTidakTerdaftar
	case err != nil:
		return "", fmt.Errorf("sqlstore: membaca alamat layanan: %w", err)
	}
	if strings.TrimSpace(alamat) == "" {
		return "", provider.ErrLayananTidakTerdaftar
	}
	return strings.TrimSpace(alamat), nil
}

// Pastikan Warisan benar-benar memenuhi kedua seam yang dipakai provider. Pemeriksaan
// ini terjadi saat kompilasi, bukan saat pengguna pertama mencoba masuk.
var (
	_ provider.DaftarLoginRepo = (*Warisan)(nil)
	_ provider.KatalogLayanan  = (*Warisan)(nil)
)

// PeriksaTabel menguji apakah sebuah tabel ada dan dapat dibaca akun aplikasi, tanpa
// mengambil satu baris pun.
//
// Dipakai mode periksa untuk membedakan dua sebab kegagalan yang tampak mirip tetapi
// perbaikannya berbeda jauh: tabel belum dibuat (migrasi belum dijalankan DBA) versus
// akun aplikasi tidak punya hak baca.
func (w *Warisan) PeriksaTabel(ctx context.Context, namaKueri string) error {
	baris, err := w.db.QueryContext(ctx, ambilKueri(namaKueri))
	if err != nil {
		return err
	}
	return baris.Close()
}

// DB membuka koneksi yang dipegang repo ini supaya repo lain dapat dipasang di atas
// koneksi yang sama, tanpa perakit di cmd perlu memegang dua rujukan ke benda yang
// sama.
func (w *Warisan) DB() *sql.DB { return w.db }
