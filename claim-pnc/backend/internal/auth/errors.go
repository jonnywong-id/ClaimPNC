package auth

import (
	"errors"
	"sort"
	"strings"
)

// Tiga jenis kegagalan autentikasi yang menuntut perlakuan berbeda. Pemanggil
// membedakannya dengan errors.Is, bukan dengan memeriksa teks pesan.
//
//   - ErrKredensialSalah    → pengguna mengetik ulang. Pesan ke pengguna TIDAK BOLEH
//     membedakan "pengguna tidak ada" dari "kata sandi salah";
//     membedakannya membocorkan keberadaan akun.
//   - ErrPenggunaTidakAktif → mengetik ulang tidak menolong; pengguna harus menghubungi
//     administrator.
//   - ErrSistemTidakTerhubung → bukan salah pengguna. Mencoba berulang kali justru
//     membanjiri sistem yang sedang bermasalah.
var (
	ErrKredensialSalah      = errors.New("auth: kredensial tidak sah")
	ErrPenggunaTidakAktif   = errors.New("auth: pengguna tidak aktif")
	ErrSistemTidakTerhubung = errors.New("auth: sistem identitas tidak dapat dihubungi")
)

// ErrPenggunaTidakDitemukan dikembalikan PenggunaRepo bila NIK yang diminta belum
// tercatat.
var ErrPenggunaTidakDitemukan = errors.New("auth: pengguna tidak ditemukan")

// Tiga kegagalan sesi yang wajib dapat dibedakan pemanggil tanpa membaca teks pesan.
//
// Pembedaan ErrSesiTidakDitemukan dari ErrSesiKedaluwarsa bukan kemewahan: yang pertama
// berarti token tidak sah — layar masuk biasa; yang kedua berarti pekerjaan pengguna
// terputus di tengah jalan dan frontend perlu menyelamatkan isian yang belum tersimpan.
var (
	ErrSesiTidakDitemukan = errors.New("auth: sesi tidak ditemukan")
	ErrSesiKedaluwarsa    = errors.New("auth: masa berlaku sesi habis")
	ErrSesiDicabut        = errors.New("auth: sesi sudah dicabut")
)

// GalatProfilTidakLengkap dikembalikan bila sistem identitas menjawab dengan profil
// yang salah satu fieldnya kosong.
type GalatProfilTidakLengkap struct {
	FieldKosong []string
}

func (g *GalatProfilTidakLengkap) Error() string {
	field := append([]string(nil), g.FieldKosong...)
	sort.Strings(field)
	return "auth: profil tidak lengkap, field kosong: " + strings.Join(field, ", ")
}
