package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"time"
)

// tokenByteLength 32 byte acak = 256 bit, cukup jauh di atas ambang tebakan praktis.
const tokenByteLength = 32

// Token adalah nilai mentah yang dipegang peramban.
//
// Nilai ini TIDAK PERNAH disimpan di basis data dan TIDAK PERNAH masuk log. Yang
// disimpan hanyalah sidiknya, sehingga bocornya isi tabel sesi tidak dengan sendirinya
// memberi orang lain sesi yang dapat dipakai.
type Token string

// Digest mengembalikan sidik SHA-256 token dalam heksadesimal. Fungsinya satu arah:
// dari sidik tidak dapat disusun kembali tokennya.
func (t Token) Digest() string {
	count := sha256.Sum256([]byte(t))
	return hex.EncodeToString(count[:])
}

// IssueToken membuat token acak baru. Sumber keacakan diterima sebagai parameter
// supaya pengujian dapat menjalankannya secara deterministik; di produksi ia selalu
// crypto/rand.Reader.
func IssueToken(random io.Reader) (Token, error) {
	if random == nil {
		random = rand.Reader
	}
	body := make([]byte, tokenByteLength)
	if _, err := io.ReadFull(random, body); err != nil {
		return "", err
	}
	return Token(base64.RawURLEncoding.EncodeToString(body)), nil
}

// NewSessionID membuat pengenal sesi yang acak dan berdiri sendiri.
//
// Pengenal ini sengaja TIDAK diturunkan dari token maupun sidiknya: pengenal sesi
// muncul di jejak audit dan di layar administrasi, dan tidak satu pun dari keduanya
// boleh menjadi petunjuk menuju token yang masih hidup.
func NewSessionID(random io.Reader) (string, error) {
	if random == nil {
		random = rand.Reader
	}
	body := make([]byte, 16)
	if _, err := io.ReadFull(random, body); err != nil {
		return "", err
	}
	return hex.EncodeToString(body), nil
}

// Session adalah satu sesi yang pernah diterbitkan.
//
// Ia tidak memuat kredensial, tidak memuat data nasabah, dan tidak memuat izin. Izin
// sengaja tidak ikut: izin dapat berubah kapan saja lewat layar master data, dan izin
// yang tertanam di sesi baru berlaku setelah sesi berakhir — bisa satu jam kemudian
// (docs/Steering/11-SECURITY.md §2.2).
type Session struct {
	ID          string
	TokenDigest string
	Identity    string
	IssuedAt    time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

// Check menyatakan apakah sesi masih boleh dipakai pada waktu tertentu.
//
// Pencabutan diperiksa lebih dulu daripada kedaluwarsa: sesi yang dicabut administrator
// harus terbaca sebagai dicabut walau kebetulan juga sudah lewat masa berlakunya.
func (s Session) Check(now time.Time) error {
	if s.RevokedAt != nil {
		return ErrSessionRevoked
	}
	if !now.Before(s.ExpiresAt) {
		return ErrSessionExpired
	}
	return nil
}

// Active adalah bentuk ringkas Periksa untuk tempat yang hanya butuh ya atau tidak.
func (s Session) Active(now time.Time) bool { return s.Check(now) == nil }

// TimeLeft mengembalikan berapa lama lagi sesi ini berlaku; nol bila sudah habis.
func (s Session) TimeLeft(now time.Time) time.Duration {
	if left := s.ExpiresAt.Sub(now); left > 0 {
		return left
	}
	return 0
}

// SessionRepo adalah seam ke tempat sesi aktif disimpan.
//
// Sesi disimpan di basis data, bukan di memori satu instans: aplikasi wajib stateless
// karena load balancer mengarahkan permintaan ke instans mana pun (D-27), dan
// pencabutan harus berlaku seketika — bukan menunggu masa berlaku habis.
//
// Pengisinya ada di auth/repo/sqlstore dan auth/repo/memory.
type SessionRepo interface {
	// Save menuliskan sesi yang baru diterbitkan.
	Save(ctx context.Context, s Session) error

	// GetByTokenDigest mencari sesi dari sidik tokennya. Ia mengembalikan
	// ErrSessionNotFound bila tidak ada yang cocok — dan galat itu sengaja
	// berbeda dari ErrSessionExpired.
	GetByTokenDigest(ctx context.Context, digest string) (Session, error)

	// Revoke menandai sesi sebagai dicabut pada waktu tertentu. Barisnya tidak dihapus
	// secara fisik (ADR-0012): sesi yang pernah ada tetap dapat ditelusuri jejak audit.
	Revoke(ctx context.Context, id string, at time.Time) error

	// Extend menggeser batas berlaku sesi yang masih aktif.
	Extend(ctx context.Context, id string, expiresAt time.Time) error
}
