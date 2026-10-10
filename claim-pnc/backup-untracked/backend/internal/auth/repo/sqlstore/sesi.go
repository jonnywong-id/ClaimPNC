package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/auth"
)

// SesiRepo memenuhi auth.SesiRepo terhadap basis data relasional.
//
// Sesi sengaja hidup di basis data, bukan di memori proses: load balancer mengarahkan
// permintaan ke instans mana pun (D-27), dan pencabutan harus berlaku seketika.
//
// Ia dipasang pada koneksi **portal utama**, bukan pada portal yang sedang dipilih
// pengguna — ADR-0030 menetapkan berpindah portal tidak menuntut login ulang, dan itu
// hanya mungkin bila sesinya tidak ikut berpindah basis data.
type SesiRepo struct {
	db *sql.DB
}

// SesiRepoBaru membentuk repo; db wajib sudah terhubung.
func SesiRepoBaru(db *sql.DB) *SesiRepo { return &SesiRepo{db: db} }

// Simpan menuliskan sesi yang baru diterbitkan.
func (r *SesiRepo) Simpan(ctx context.Context, sesiBaru auth.Sesi) error {
	_, err := r.db.ExecContext(ctx, ambilKueri("sesi_sisip"),
		sesiBaru.ID, sesiBaru.SidikToken, sesiBaru.Identitas,
		sesiBaru.DiterbitkanPada.UTC(), sesiBaru.BerlakuSampai.UTC())
	if err != nil {
		// Galat sengaja tidak memuat sidik token maupun tokennya.
		return fmt.Errorf("sqlstore: menyimpan sesi: %w", err)
	}
	return nil
}

// AmbilBySidikToken mencari sesi dari sidik tokennya.
func (r *SesiRepo) AmbilBySidikToken(ctx context.Context, sidik string) (auth.Sesi, error) {
	baris := r.db.QueryRowContext(ctx, ambilKueri("sesi_ambil_by_sidik"), sidik)

	var (
		hasil   auth.Sesi
		dicabut sql.NullTime
	)
	err := baris.Scan(&hasil.ID, &hasil.SidikToken, &hasil.Identitas,
		&hasil.DiterbitkanPada, &hasil.BerlakuSampai, &dicabut)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return auth.Sesi{}, auth.ErrSesiTidakDitemukan
	case err != nil:
		return auth.Sesi{}, fmt.Errorf("sqlstore: membaca sesi: %w", err)
	}
	if dicabut.Valid {
		pada := dicabut.Time.UTC()
		hasil.DicabutPada = &pada
	}
	hasil.DiterbitkanPada = hasil.DiterbitkanPada.UTC()
	hasil.BerlakuSampai = hasil.BerlakuSampai.UTC()
	return hasil, nil
}

// Cabut menandai sesi sebagai dicabut. Barisnya tidak dihapus (ADR-0012).
func (r *SesiRepo) Cabut(ctx context.Context, id string, pada time.Time) error {
	if _, err := r.db.ExecContext(ctx, ambilKueri("sesi_cabut"), pada.UTC(), id); err != nil {
		return fmt.Errorf("sqlstore: mencabut sesi: %w", err)
	}
	return nil
}

// Perpanjang menggeser batas berlaku sesi yang masih aktif.
func (r *SesiRepo) Perpanjang(ctx context.Context, id string, berlakuSampai time.Time) error {
	if _, err := r.db.ExecContext(ctx, ambilKueri("sesi_perpanjang"), berlakuSampai.UTC(), id); err != nil {
		return fmt.Errorf("sqlstore: memperpanjang sesi: %w", err)
	}
	return nil
}
