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

// PermintaanMasuk adalah isian layar masuk.
type PermintaanMasuk struct {
	NamaPengguna string `json:"nama_pengguna"`
	KataSandi    string `json:"kata_sandi"`
}

// PenggunaDTO adalah bentuk profil yang dikirim ke peramban.
//
// Tipe ini sengaja TERPISAH dari auth.Pengguna. Memakai tipe modul langsung sebagai
// bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (docs/Steering/08-TECHNICAL-STRATEGY.md §2 aturan 4). Ia juga tidak memuat kata sandi
// dalam bentuk apa pun, dan tidak memuat data nasabah.
//
// Email dan Perusahaan dapat kosong: sumber identitas non-karyawan
// (POOLDATA.M_LOGIN_PNC) memang tidak mengirimkan keduanya.
type PenggunaDTO struct {
	Identitas  string `json:"identitas"`
	Nama       string `json:"nama"`
	Jenis      string `json:"jenis"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	Perusahaan string `json:"perusahaan"`
}

// ResponsMasuk adalah jawaban atas masuk yang berhasil.
//
// Token dikirim sekali di sini dan tidak pernah muncul lagi di respons mana pun.
type ResponsMasuk struct {
	Token         string      `json:"token"`
	TipeToken     string      `json:"tipe_token"`
	BerlakuSampai time.Time   `json:"berlaku_sampai"`
	Pengguna      PenggunaDTO `json:"pengguna"`
}

// ResponsSaya adalah identitas pemanggil beserta keadaan sesinya.
type ResponsSaya struct {
	Pengguna      PenggunaDTO `json:"pengguna"`
	BerlakuSampai time.Time   `json:"berlaku_sampai"`
}

// ResponsPerpanjang adalah jawaban atas perpanjangan sesi.
type ResponsPerpanjang struct {
	BerlakuSampai time.Time `json:"berlaku_sampai"`
}

// ResponsGalat adalah bentuk galat seragam.
//
// Kode dimaksudkan untuk dibaca program, Pesan untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Pesan.
//
// Bentuk ini SEMENTARA: kontrak galat API yang mengikat seluruh aplikasi adalah
// TKT-F1-004, yang masih terhalang keputusan Work Owner soal kegagalan senyap.
type ResponsGalat struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`
}
