package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"claim-pnc/internal/auth"
)

// LocalLogin adalah satu baris POOLDATA.M_LOGIN_PNC yang cocok.
type LocalLogin struct {
	LoginID   string
	LoginName string
}

// LoginListRepo adalah seam ke tabel POOLDATA.M_LOGIN_PNC.
//
// Tabel itu memuat pengguna NON-KARYAWAN — broker dan surveyor independen — yang tidak
// punya akun di HCC/HCQ.
type LoginListRepo interface {
	// FindActive mencari baris yang login_id, sidik kata sandinya cocok, dan
	// active_status = '1'. Ia mengembalikan ErrLoginMismatch bila tidak ada.
	//
	// Ketiga syarat sengaja digabung dalam satu kueri: memisahkannya menjadi "cari
	// pengguna" lalu "bandingkan sandi" membuat lamanya jawaban berbeda antara akun
	// yang ada dan yang tidak — dan selisih waktu itu membocorkan keberadaan akun.
	FindActive(ctx context.Context, loginID, passwordDigest string) (LocalLogin, error)
}

// ErrLoginMismatch dikembalikan LoginListRepo bila tidak ada baris yang cocok.
var ErrLoginMismatch = errors.New("provider: login lokal tidak cocok")

// Local memverifikasi kredensial non-karyawan ke POOLDATA.M_LOGIN_PNC.
type Local struct {
	repo LoginListRepo
}

// NewLocal membentuk provider lokal.
func NewLocal(repo LoginListRepo) (*Local, error) {
	if repo == nil {
		return nil, errors.New("provider: daftar login repo wajib diisi")
	}
	return &Local{repo: repo}, nil
}

// PasswordDigest menghitung sidik kata sandi sesuai skema yang dipakai kolom
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
func PasswordDigest(password string) string {
	count := sha256.Sum256([]byte(password))
	return strings.ToUpper(hex.EncodeToString(count[:]))
}

// Verify mencocokkan kredensial ke daftar login lokal.
func (l *Local) Verify(ctx context.Context, k auth.Credential) (auth.Profile, error) {
	loginID := strings.TrimSpace(k.Username)
	if loginID == "" || k.Password == "" {
		return auth.Profile{}, auth.ErrWrongCredential
	}

	row, err := l.repo.FindActive(ctx, loginID, PasswordDigest(k.Password))
	switch {
	case errors.Is(err, ErrLoginMismatch):
		// Satu galat untuk tiga sebab: login tidak ada, sandi salah, atau akun tidak
		// aktif. Membedakan yang ketiga di sini akan membocorkan keberadaan akun;
		// pembedaan "tidak aktif" hanya dilakukan untuk pengguna yang identitasnya
		// sudah terbukti, yaitu lewat catatan pengguna lokal aplikasi ini.
		return auth.Profile{}, auth.ErrWrongCredential
	case err != nil:
		return auth.Profile{}, err
	}

	profile := auth.Profile{
		Identity: strings.TrimSpace(row.LoginID),
		Name:     strings.TrimSpace(row.LoginName),
		Kind:     auth.NonEmployee,
		Login:    strings.TrimSpace(row.LoginID),
	}
	if err := profile.Check(); err != nil {
		return auth.Profile{}, err
	}
	return profile, nil
}
