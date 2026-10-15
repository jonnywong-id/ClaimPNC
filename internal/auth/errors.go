package auth

import (
	"errors"
	"sort"
	"strings"
)

// Tiga jenis kegagalan autentikasi yang menuntut perlakuan berbeda. Pemanggil
// membedakannya dengan errors.Is, bukan dengan memeriksa teks pesan.
//
//   - ErrWrongCredential    → pengguna mengetik ulang. Pesan ke pengguna TIDAK BOLEH
//     membedakan "pengguna tidak ada" dari "kata sandi salah";
//     membedakannya membocorkan keberadaan akun.
//   - ErrUserInactive → mengetik ulang tidak menolong; pengguna harus menghubungi
//     administrator.
//   - ErrIdentitySystemUnreachable → bukan salah pengguna. Mencoba berulang kali justru
//     membanjiri sistem yang sedang bermasalah.
var (
	ErrWrongCredential           = errors.New("auth: kredensial tidak sah")
	ErrUserInactive              = errors.New("auth: pengguna tidak aktif")
	ErrIdentitySystemUnreachable = errors.New("auth: sistem identitas tidak dapat dihubungi")
)

// ErrUserNotFound dikembalikan UserRepo bila NIK yang diminta belum
// tercatat.
var ErrUserNotFound = errors.New("auth: pengguna tidak ditemukan")

// Tiga kegagalan sesi yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
// Pembedaan ErrSessionNotFound dari ErrSessionExpired bukan kemewahan: yang pertama
// berarti token tidak sah — layar masuk biasa; yang kedua berarti pekerjaan pengguna
// terputus di tengah jalan dan frontend perlu menyelamatkan isian yang belum tersimpan.
var (
	ErrSessionNotFound = errors.New("auth: sesi tidak ditemukan")
	ErrSessionExpired  = errors.New("auth: masa berlaku sesi habis")
	ErrSessionRevoked  = errors.New("auth: sesi sudah dicabut")
)

// IncompleteProfileError dikembalikan bila sistem identitas menjawab dengan profil
// yang salah satu fieldnya kosong.
type IncompleteProfileError struct {
	EmptyFields []string
}

func (g *IncompleteProfileError) Error() string {
	field := append([]string(nil), g.EmptyFields...)
	sort.Strings(field)
	return "auth: profil tidak lengkap, field kosong: " + strings.Join(field, ", ")
}
