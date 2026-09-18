// Package authhttp adalah lapisan transport modul auth: bentuk permintaan dan respons,
// pemetaan galat, middleware sesi, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya. Foldernya `http` supaya letaknya
// seragam dengan modul lain; nama paketnya `authhttp` supaya tidak menutupi `net/http`
// yang dipakai hampir di setiap berkas di dalamnya.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package authhttp

import "time"

// LoginRequest adalah isian layar masuk.
type LoginRequest struct {
	Username string `json:"nama_pengguna"`
	Password string `json:"kata_sandi"`
}

// UserDTO adalah bentuk profil yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari auth.User. Memakai tipe modul langsung sebagai
// bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4). Ia juga tidak memuat kata sandi
// dalam bentuk apa pun, dan tidak memuat data nasabah.
//
// Email dan Perusahaan dapat kosong: sumber identitas non-karyawan
// (POOLDATA.M_LOGIN_PNC) memang tidak mengirimkan keduanya.
type UserDTO struct {
	Identity string `json:"identitas"`
	Name     string `json:"nama"`
	Kind     string `json:"jenis"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Company  string `json:"perusahaan"`
}

// LoginResponse adalah jawaban atas masuk yang berhasil.
//
// Token dikirim sekali di sini dan tidak pernah muncul lagi di respons mana pun.
type LoginResponse struct {
	Token     string    `json:"token"`
	TipeToken string    `json:"tipe_token"`
	ExpiresAt time.Time `json:"berlaku_sampai"`
	User      UserDTO   `json:"pengguna"`
}

// MeResponse adalah identitas pemanggil beserta keadaan sesinya.
type MeResponse struct {
	User      UserDTO   `json:"pengguna"`
	ExpiresAt time.Time `json:"berlaku_sampai"`
}

// RenewResponse adalah jawaban atas perpanjangan sesi.
type RenewResponse struct {
	ExpiresAt time.Time `json:"berlaku_sampai"`
}

// ErrorResponse adalah bentuk galat seragam.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuk ini SEMENTARA: kontrak galat API yang mengikat seluruh aplikasi adalah
// TKT-F1-004, yang masih terhalang keputusan Work Owner soal kegagalan senyap.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}
