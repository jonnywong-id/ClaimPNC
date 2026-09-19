package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"claim-pnc/internal/auth"
)

// SessionRepo memenuhi auth.SessionRepo terhadap basis data relasional.
//
// Sesi sengaja hidup di basis data, bukan di memori proses: load balancer mengarahkan
// permintaan ke instans mana pun (D-27), dan pencabutan harus berlaku seketika.
//
// Ia dipasang pada koneksi **portal utama**, bukan pada portal yang sedang dipilih
// pengguna — ADR-0030 menetapkan berpindah portal tidak menuntut login ulang, dan itu
// hanya mungkin bila sesinya tidak ikut berpindah basis data.
type SessionRepo struct {
	db *sql.DB
}

// NewSessionRepo membentuk repo; db wajib sudah terhubung.
func NewSessionRepo(db *sql.DB) *SessionRepo { return &SessionRepo{db: db} }

// Save menuliskan sesi yang baru diterbitkan.
func (r *SessionRepo) Save(ctx context.Context, freshSession auth.Session) error {
	_, err := r.db.ExecContext(ctx, getQuery("session_insert"),
		freshSession.ID, freshSession.TokenDigest, freshSession.Identity,
		freshSession.IssuedAt.UTC(), freshSession.ExpiresAt.UTC())
	if err != nil {
		// Galat sengaja tidak memuat sidik token maupun tokennya.
		return fmt.Errorf("sqlstore: menyimpan sesi: %w", err)
	}
	return nil
}

// GetByTokenDigest mencari sesi dari sidik tokennya.
func (r *SessionRepo) GetByTokenDigest(ctx context.Context, fingerprint string) (auth.Session, error) {
	rows := r.db.QueryRowContext(ctx, getQuery("session_get_by_fingerprint"), fingerprint)

	var (
		result  auth.Session
		dicabut sql.NullTime
	)
	err := rows.Scan(&result.ID, &result.TokenDigest, &result.Identity,
		&result.IssuedAt, &result.ExpiresAt, &dicabut)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return auth.Session{}, auth.ErrSessionNotFound
	case err != nil:
		return auth.Session{}, fmt.Errorf("sqlstore: membaca sesi: %w", err)
	}
	if dicabut.Valid {
		pada := dicabut.Time.UTC()
		result.RevokedAt = &pada
	}
	result.IssuedAt = result.IssuedAt.UTC()
	result.ExpiresAt = result.ExpiresAt.UTC()
	return result, nil
}

// Revoke menandai sesi sebagai dicabut. Barisnya tidak dihapus (ADR-0012).
func (r *SessionRepo) Revoke(ctx context.Context, id string, pada time.Time) error {
	if _, err := r.db.ExecContext(ctx, getQuery("session_revoke"), pada.UTC(), id); err != nil {
		return fmt.Errorf("sqlstore: mencabut sesi: %w", err)
	}
	return nil
}

// Renew menggeser batas berlaku sesi yang masih aktif.
func (r *SessionRepo) Renew(ctx context.Context, id string, berlakuSampai time.Time) error {
	if _, err := r.db.ExecContext(ctx, getQuery("session_extend"), berlakuSampai.UTC(), id); err != nil {
		return fmt.Errorf("sqlstore: memperpanjang sesi: %w", err)
	}
	return nil
}
