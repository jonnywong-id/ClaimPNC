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

// panjangTokenByte 32 byte acak = 256 bit, cukup jauh di atas ambang tebakan praktis.
const panjangTokenByte = 32

// Token adalah nilai mentah yang dipegang peramban.
//
// Nilai ini TIDAK PERNAH disimpan di basis data dan TIDAK PERNAH masuk log. Yang
// disimpan hanyalah sidiknya, sehingga bocornya isi tabel sesi tidak dengan sendirinya
// memberi orang lain sesi yang dapat dipakai.
type Token string

// Sidik mengembalikan sidik SHA-256 token dalam heksadesimal. Fungsinya satu arah:
// dari sidik tidak dapat disusun kembali tokennya.
func (t Token) Sidik() string {
	jumlah := sha256.Sum256([]byte(t))
	return hex.EncodeToString(jumlah[:])
}

// TerbitkanToken membuat token acak baru. Sumber keacakan diterima sebagai parameter
// supaya pengujian dapat menjalankannya secara deterministik; di produksi ia selalu
// crypto/rand.Reader.
func TerbitkanToken(acak io.Reader) (Token, error) {
	if acak == nil {
		acak = rand.Reader
	}
	isi := make([]byte, panjangTokenByte)
	if _, err := io.ReadFull(acak, isi); err != nil {
		return "", err
	}
	return Token(base64.RawURLEncoding.EncodeToString(isi)), nil
}

// IDSesiBaru membuat pengenal sesi yang acak dan berdiri sendiri.
//
// Pengenal ini sengaja TIDAK diturunkan dari token maupun sidiknya: pengenal sesi
// muncul di jejak audit dan di layar administrasi, dan tidak satu pun dari keduanya
// boleh menjadi petunjuk menuju token yang masih hidup.
func IDSesiBaru(acak io.Reader) (string, error) {
	if acak == nil {
		acak = rand.Reader
	}
	isi := make([]byte, 16)
	if _, err := io.ReadFull(acak, isi); err != nil {
		return "", err
	}
	return hex.EncodeToString(isi), nil
}

// Sesi adalah satu sesi yang pernah diterbitkan.
//
// Ia tidak memuat kredensial, tidak memuat data nasabah, dan tidak memuat izin. Izin
// sengaja tidak ikut: izin dapat berubah kapan saja lewat layar master data, dan izin
// yang tertanam di sesi baru berlaku setelah sesi berakhir — bisa satu jam kemudian
// (docs/Steering/11-SECURITY.md §2.2).
type Sesi struct {
	ID              string
	SidikToken      string
	Identitas       string
	DiterbitkanPada time.Time
	BerlakuSampai   time.Time
	DicabutPada     *time.Time
}

// Periksa menyatakan apakah sesi masih boleh dipakai pada waktu tertentu.
//
// Pencabutan diperiksa lebih dulu daripada kedaluwarsa: sesi yang dicabut administrator
// harus terbaca sebagai dicabut walau kebetulan juga sudah lewat masa berlakunya.
func (s Sesi) Periksa(sekarang time.Time) error {
	if s.DicabutPada != nil {
		return ErrSesiDicabut
	}
	if !sekarang.Before(s.BerlakuSampai) {
		return ErrSesiKedaluwarsa
	}
	return nil
}

// Aktif adalah bentuk ringkas Periksa untuk tempat yang hanya butuh ya atau tidak.
func (s Sesi) Aktif(sekarang time.Time) bool { return s.Periksa(sekarang) == nil }

// SisaBerlaku mengembalikan berapa lama lagi sesi ini berlaku; nol bila sudah habis.
func (s Sesi) SisaBerlaku(sekarang time.Time) time.Duration {
	if sisa := s.BerlakuSampai.Sub(sekarang); sisa > 0 {
		return sisa
	}
	return 0
}

// SesiRepo adalah seam ke tempat sesi aktif disimpan.
//
// Sesi disimpan di basis data, bukan di memori satu instans: aplikasi wajib stateless
// karena load balancer mengarahkan permintaan ke instans mana pun (D-27), dan
// pencabutan harus berlaku seketika — bukan menunggu masa berlaku habis.
//
// Pengisinya ada di auth/repo/sqlstore dan auth/repo/memori.
type SesiRepo interface {
	// Simpan menuliskan sesi yang baru diterbitkan.
	Simpan(ctx context.Context, s Sesi) error

	// AmbilBySidikToken mencari sesi dari sidik tokennya. Ia mengembalikan
	// ErrSesiTidakDitemukan bila tidak ada yang cocok — dan galat itu sengaja
	// berbeda dari ErrSesiKedaluwarsa.
	AmbilBySidikToken(ctx context.Context, sidik string) (Sesi, error)

	// Cabut menandai sesi sebagai dicabut pada waktu tertentu. Barisnya tidak dihapus
	// secara fisik (ADR-0012): sesi yang pernah ada tetap dapat ditelusuri jejak audit.
	Cabut(ctx context.Context, id string, pada time.Time) error

	// Perpanjang menggeser batas berlaku sesi yang masih aktif.
	Perpanjang(ctx context.Context, id string, berlakuSampai time.Time) error
}
