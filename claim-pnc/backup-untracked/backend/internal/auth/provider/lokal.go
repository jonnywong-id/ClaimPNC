package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"claim-pnc/internal/auth"
)

// LoginLokal adalah satu baris POOLDATA.M_LOGIN_PNC yang cocok.
type LoginLokal struct {
	LoginID   string
	LoginNama string
}

// DaftarLoginRepo adalah seam ke tabel POOLDATA.M_LOGIN_PNC.
//
// Tabel itu memuat pengguna NON-KARYAWAN — broker dan surveyor independen — yang tidak
// punya akun di HCC/HCQ.
type DaftarLoginRepo interface {
	// CariAktif mencari baris yang login_id, sidik kata sandinya cocok, dan
	// active_status = '1'. Ia mengembalikan ErrLoginTidakCocok bila tidak ada.
	//
	// Ketiga syarat sengaja digabung dalam satu kueri: memisahkannya menjadi "cari
	// pengguna" lalu "bandingkan sandi" membuat lamanya jawaban berbeda antara akun
	// yang ada dan yang tidak — dan selisih waktu itu membocorkan keberadaan akun.
	CariAktif(ctx context.Context, loginID, sidikKataSandi string) (LoginLokal, error)
}

// ErrLoginTidakCocok dikembalikan DaftarLoginRepo bila tidak ada baris yang cocok.
var ErrLoginTidakCocok = errors.New("provider: login lokal tidak cocok")

// Lokal memverifikasi kredensial non-karyawan ke POOLDATA.M_LOGIN_PNC.
type Lokal struct {
	repo DaftarLoginRepo
}

// LokalBaru membentuk provider lokal.
func LokalBaru(repo DaftarLoginRepo) (*Lokal, error) {
	if repo == nil {
		return nil, errors.New("provider: daftar login repo wajib diisi")
	}
	return &Lokal{repo: repo}, nil
}

// SidikKataSandi menghitung sidik kata sandi sesuai skema yang dipakai kolom
// M_LOGIN_PNC.HASH_PASSWORD: SHA-256 polos, dituliskan heksadesimal HURUF BESAR.
//
// # Catatan keamanan yang disadari
//
// Skema ini **lemah menurut ukuran hari ini**: tanpa garam dan tanpa peregangan, satu
// kata sandi lemah dapat dibalik dari tabel pelangi dalam hitungan detik — dan data
// contoh membuktikannya, sidik pada baris contoh adalah SHA-256 dari "123".
//
// Ia tetap dipakai apa adanya karena nilainya **sudah tersimpan** di kolom itu dan
// dipakai sistem yang sedang berjalan; menggantinya dengan bcrypt atau Argon2 menuntut
// seluruh pengguna menyetel ulang kata sandinya, dan itu keputusan Work Owner, bukan
// keputusan yang boleh diambil diam-diam saat memindahkan aplikasi. Dicatat sebagai
// pertanyaan terbuka di docs/keputusan-implementasi.md.
func SidikKataSandi(kataSandi string) string {
	jumlah := sha256.Sum256([]byte(kataSandi))
	return strings.ToUpper(hex.EncodeToString(jumlah[:]))
}

// Verifikasi mencocokkan kredensial ke daftar login lokal.
func (l *Lokal) Verifikasi(ctx context.Context, k auth.Kredensial) (auth.Profil, error) {
	loginID := strings.TrimSpace(k.NamaPengguna)
	if loginID == "" || k.KataSandi == "" {
		return auth.Profil{}, auth.ErrKredensialSalah
	}

	baris, err := l.repo.CariAktif(ctx, loginID, SidikKataSandi(k.KataSandi))
	switch {
	case errors.Is(err, ErrLoginTidakCocok):
		// Satu galat untuk tiga sebab: login tidak ada, sandi salah, atau akun tidak
		// aktif. Membedakan yang ketiga di sini akan membocorkan keberadaan akun;
		// pembedaan "tidak aktif" hanya dilakukan untuk pengguna yang identitasnya
		// sudah terbukti, yaitu lewat catatan pengguna lokal aplikasi ini.
		return auth.Profil{}, auth.ErrKredensialSalah
	case err != nil:
		return auth.Profil{}, err
	}

	profil := auth.Profil{
		Identitas: strings.TrimSpace(baris.LoginID),
		Nama:      strings.TrimSpace(baris.LoginNama),
		Jenis:     auth.NonKaryawan,
		Login:     strings.TrimSpace(baris.LoginID),
	}
	if err := profil.Periksa(); err != nil {
		return auth.Profil{}, err
	}
	return profil, nil
}
