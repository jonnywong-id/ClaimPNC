package auth

import (
	"context"
	"strings"
)

// JenisPengguna membedakan dua populasi pengguna yang diverifikasi lewat dua jalur
// berbeda — dan perbedaan itu nyata secara bisnis, bukan sekadar teknis.
const (
	// Karyawan diverifikasi ke API HCC/HCQ. Identitasnya adalah NIK.
	Karyawan JenisPengguna = "KARYAWAN"

	// NonKaryawan — broker dan surveyor independen — diverifikasi ke tabel
	// POOLDATA.M_LOGIN_PNC. Identitasnya adalah LOGIN_ID; mereka tidak punya NIK.
	NonKaryawan JenisPengguna = "NON_KARYAWAN"
)

// JenisPengguna adalah asal identitas seorang pengguna.
type JenisPengguna string

// Dikenal menyatakan apakah jenis ini salah satu dari dua yang sah.
func (j JenisPengguna) Dikenal() bool {
	return j == Karyawan || j == NonKaryawan
}

// Kredensial adalah yang dimasukkan pengguna di layar masuk.
//
// Nilainya tidak pernah ditulis ke log dan tidak pernah disimpan di basis data —
// aplikasi ini tidak mengelola kata sandi karyawan sama sekali, dan untuk non-karyawan
// yang tersimpan hanyalah sidiknya, bukan kata sandinya.
type Kredensial struct {
	NamaPengguna string
	KataSandi    string
}

// Profil adalah jawaban sistem identitas atas kredensial yang sah.
//
// # Kenapa hanya tiga field yang wajib
//
// `D-07` dan `docs/Steering/11-SECURITY.md` §2.1 menyatakan HCC/HCQ mengembalikan
// "profil lengkap: NIK, nama, cabang, jabatan, email". **Kontrak yang akhirnya diterima
// pada 2026-09-16 membantah itu.** Responsnya hanya memuat `NIK`, `Name`, `Login`,
// `pyEmail1`, dan `pyCompany` — tanpa cabang dan tanpa jabatan. Sumber kedua,
// `POOLDATA.M_LOGIN_PNC`, bahkan hanya memuat `login_id` dan `login_name`.
//
// Karena itu yang wajib adalah tiga hal yang benar-benar selalu ada: siapa dia
// (Identitas), namanya, dan dari jalur mana ia diverifikasi. Sisanya opsional dan
// dibiarkan kosong bila sumbernya memang tidak mengirimkannya — bukan diisi tebakan.
type Profil struct {
	// Identitas adalah kunci alami pengguna: NIK untuk karyawan, LOGIN_ID untuk
	// non-karyawan.
	Identitas string
	Nama      string
	Jenis     JenisPengguna

	// Login adalah yang diketik pengguna di layar masuk — alamat surel untuk karyawan,
	// LOGIN_ID untuk non-karyawan. Disimpan supaya pengguna dapat ditelusuri dengan
	// yang ia kenal, bukan hanya dengan NIK.
	Login string

	// Field berikut OPSIONAL. Kosong berarti sumber identitasnya memang tidak
	// mengirimkannya — POOLDATA.M_LOGIN_PNC hanya memuat login_id dan nama.
	Email      string // HCQ: EmpResponse.Person.pyEmail1
	Perusahaan string // HCQ: EmpResponse.Person.pyCompany
	Cabang     string // HCQ: EmpResponse.Placement.BranchName
	KodeCabang string // HCQ: EmpResponse.Placement.BranchCode
	Jabatan    string // HCQ: EmpResponse.Placement.PositionName

	// AktifDiSumber adalah EmpResponse.Placement.IsActive apa adanya.
	//
	// Ia DIREKAM tetapi TIDAK dipakai menolak masuk: aturan yang ditetapkan Work Owner
	// hanya menyebut pyErrorCode = "200" sebagai syarat masuk. Menambah syarat sendiri
	// berarti mengarang aturan kewenangan. Yang menggerbang tetap kolom AKTIF pada
	// catatan pengguna lokal, yang dikelola administrator aplikasi ini.
	// Lihat pertanyaan terbuka di docs/keputusan-implementasi.md.
	AktifDiSumber *bool
}

// Identitas adalah seam ke sistem autentikasi.
//
// Implementasinya hanya boleh menjawab satu pertanyaan: apakah kredensial ini sah, dan
// bila sah, siapa pemiliknya. Ia tidak tahu apa pun soal sesi maupun izin.
//
// Pengisinya ada di auth/provider: HCQ untuk karyawan, Lokal untuk non-karyawan, dan
// Berantai yang menggabungkan keduanya sesuai urutan yang ditetapkan bisnis.
type Identitas interface {
	Verifikasi(ctx context.Context, kredensial Kredensial) (Profil, error)
}

// Periksa memastikan profil layak dipakai.
//
// Identitas yang kosong ditolak karena pengguna tanpa kunci alami tidak dapat
// dicocokkan dengan data klaimnya sendiri — kegagalan yang jauh lebih mahal bila baru
// ketahuan setelah ia mengisi satu form registrasi penuh.
func (p Profil) Periksa() error {
	kosong := []string{}
	if strings.TrimSpace(p.Identitas) == "" {
		kosong = append(kosong, "identitas")
	}
	if strings.TrimSpace(p.Nama) == "" {
		kosong = append(kosong, "nama")
	}
	if !p.Jenis.Dikenal() {
		kosong = append(kosong, "jenis")
	}
	if len(kosong) == 0 {
		return nil
	}
	return &GalatProfilTidakLengkap{FieldKosong: kosong}
}

// Lengkap menyatakan apakah kredensial terisi kedua-duanya.
func (k Kredensial) Lengkap() bool {
	return strings.TrimSpace(k.NamaPengguna) != "" && k.KataSandi != ""
}
