// Package provider berisi implementasi seam auth.Identitas.
//
// Tiga di antaranya nyata dan satu untuk pengembangan:
//
//	HCQ      — karyawan, ke API internal HCC/HCQ (TKT-F3-002)
//	Lokal    — non-karyawan, ke POOLDATA.M_LOGIN_PNC
//	Berantai — menggabungkan keduanya sesuai urutan yang ditetapkan bisnis
//	Tiruan   — daftar pengguna contoh di memori; MENOLAK berjalan di produksi
package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/auth"
)

// PenggunaTiruan adalah satu baris daftar pengguna contoh.
//
// Nama-nama di daftar bawaan sengaja bukan nama pegawai nyata.
type PenggunaTiruan struct {
	NamaPengguna string
	KataSandi    string
	Aktif        bool
	Profil       auth.Profil
}

// Tiruan adalah provider identitas berbasis daftar pengguna di memori.
//
// Ia tidak menyentuh jaringan maupun basis data sama sekali, sehingga seluruh modul
// yang membutuhkan identitas dapat diuji dengan jaringan dimatikan.
type Tiruan struct {
	pengguna map[string]PenggunaTiruan
	// simulasiPutus membuat provider menjawab seolah sistem identitas tidak dapat
	// dihubungi. Ia ada supaya jalur galat ketiga dapat dicoba di layar masuk tanpa
	// harus benar-benar mematikan apa pun.
	simulasiPutus bool
}

// ErrTiruanDiProduksi adalah penolakan yang membuat aplikasi gagal start.
var ErrTiruanDiProduksi = errors.New("provider identitas tiruan menolak berjalan di lingkungan produksi")

// TiruanBaru membentuk provider tiruan.
//
// Penolakan terhadap produksi berada DI SINI, di dalam kode — bukan sekadar nilai
// konfigurasi yang bisa dibalik seseorang. Siapa pun yang menyalakan provider ini di
// produksi mendapatkan aplikasi yang tidak mau start, bukan aplikasi yang menerima
// kata sandi palsu.
func TiruanBaru(lingkunganProduksi bool, daftar []PenggunaTiruan) (*Tiruan, error) {
	if lingkunganProduksi {
		return nil, ErrTiruanDiProduksi
	}
	if len(daftar) == 0 {
		daftar = DaftarContoh()
	}
	indeks := make(map[string]PenggunaTiruan, len(daftar))
	for _, p := range daftar {
		kunci := strings.ToLower(strings.TrimSpace(p.NamaPengguna))
		if kunci == "" {
			return nil, fmt.Errorf("provider identitas tiruan: ada baris tanpa nama pengguna")
		}
		indeks[kunci] = p
	}
	return &Tiruan{pengguna: indeks}, nil
}

// SetSimulasiPutus menyalakan atau mematikan simulasi sistem identitas yang tidak
// dapat dihubungi.
func (t *Tiruan) SetSimulasiPutus(putus bool) { t.simulasiPutus = putus }

// Verifikasi memeriksa kredensial terhadap daftar pengguna contoh.
//
// Kata sandi dibandingkan apa adanya karena provider ini tidak pernah memegang kata
// sandi sungguhan; yang memegangnya adalah HCC/HCQ dan M_LOGIN_PNC.
func (t *Tiruan) Verifikasi(ctx context.Context, k auth.Kredensial) (auth.Profil, error) {
	if err := ctx.Err(); err != nil {
		return auth.Profil{}, fmt.Errorf("%w: %v", auth.ErrSistemTidakTerhubung, err)
	}
	if t.simulasiPutus {
		return auth.Profil{}, auth.ErrSistemTidakTerhubung
	}

	p, ada := t.pengguna[strings.ToLower(strings.TrimSpace(k.NamaPengguna))]
	if !ada {
		// Pengguna yang tidak ada dan kata sandi yang salah menghasilkan galat yang
		// sama persis. Membedakannya membocorkan siapa saja yang punya akun.
		return auth.Profil{}, auth.ErrKredensialSalah
	}
	if p.KataSandi != k.KataSandi {
		return auth.Profil{}, auth.ErrKredensialSalah
	}
	if !p.Aktif {
		return auth.Profil{}, auth.ErrPenggunaTidakAktif
	}
	return p.Profil, nil
}

// DaftarContoh adalah pengguna bawaan untuk pengembangan.
//
// Seluruh nama, NIK, dan email di sini karangan — bukan pegawai nyata, bukan alamat
// surel nyata. Daftarnya memuat kedua jenis pengguna supaya perbedaan karyawan dan
// non-karyawan dapat dicoba tanpa basis data.
func DaftarContoh() []PenggunaTiruan {
	return []PenggunaTiruan{
		{
			// Akun pintas untuk mencoba layar berulang kali saat pengembangan: paling
			// pendek diketik, dan sengaja ditaruh paling atas supaya mudah ditemukan
			// saat hendak diganti. Kata sandinya lemah dengan sadar — ia tidak pernah
			// sampai ke produksi karena TiruanBaru menolak start di sana.
			NamaPengguna: "admin",
			KataSandi:    "admin",
			Aktif:        true,
			Profil: auth.Profil{
				Identitas:  "90000000",
				Nama:       "Contoh Admin Pintas",
				Jenis:      auth.Karyawan,
				Login:      "admin",
				Email:      "contoh.pintas@example.invalid",
				Perusahaan: "ASM",
			},
		},
		{
			NamaPengguna: "adminpnc",
			KataSandi:    "rahasia123",
			Aktif:        true,
			Profil: auth.Profil{
				Identitas:  "90000001",
				Nama:       "Contoh Administrator",
				Jenis:      auth.Karyawan,
				Login:      "adminpnc",
				Email:      "contoh.admin@example.invalid",
				Perusahaan: "ASM",
			},
		},
		{
			NamaPengguna: "pictekniks",
			KataSandi:    "rahasia123",
			Aktif:        true,
			Profil: auth.Profil{
				Identitas:  "90000002",
				Nama:       "Contoh PIC Teknik",
				Jenis:      auth.Karyawan,
				Login:      "pictekniks",
				Email:      "contoh.picteknik@example.invalid",
				Perusahaan: "ASM",
			},
		},
		{
			// Non-karyawan: hanya punya LOGIN_ID dan nama, tanpa NIK dan tanpa email —
			// persis sebatas yang dimuat POOLDATA.M_LOGIN_PNC.
			NamaPengguna: "brokercontoh",
			KataSandi:    "rahasia123",
			Aktif:        true,
			Profil: auth.Profil{
				Identitas: "BROKERCONTOH",
				Nama:      "Contoh Broker Rekanan",
				Jenis:     auth.NonKaryawan,
				Login:     "BROKERCONTOH",
			},
		},
		{
			NamaPengguna: "penggunanonaktif",
			KataSandi:    "rahasia123",
			Aktif:        false,
			Profil: auth.Profil{
				Identitas: "90000003",
				Nama:      "Contoh Pengguna Nonaktif",
				Jenis:     auth.Karyawan,
				Login:     "penggunanonaktif",
				Email:     "contoh.nonaktif@example.invalid",
			},
		},
		{
			// Dipakai untuk membuktikan bahwa profil tidak lengkap ditolak, bukan
			// diteruskan diam-diam: sumber menjawab "berhasil" tanpa nama.
			NamaPengguna: "profilbolong",
			KataSandi:    "rahasia123",
			Aktif:        true,
			Profil: auth.Profil{
				Identitas: "90000004",
				Nama:      "",
				Jenis:     auth.Karyawan,
				Login:     "profilbolong",
			},
		},
	}
}
