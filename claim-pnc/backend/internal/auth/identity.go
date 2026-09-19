package auth

import (
	"context"
	"strings"
)

// UserKind membedakan dua populasi pengguna yang diverifikasi lewat dua jalur
// berbeda — dan perbedaan itu nyata secara bisnis, bukan sekadar teknis.
const (
	// Employee diverifikasi ke API HCC/HCQ. Identitasnya adalah NIK.
	Employee UserKind = "KARYAWAN"

	// NonEmployee — broker dan surveyor independen — diverifikasi ke tabel
	// POOLDATA.M_LOGIN_PNC. Identitasnya adalah LOGIN_ID; mereka tidak punya NIK.
	NonEmployee UserKind = "NON_KARYAWAN"
)

// UserKind adalah asal identitas seorang pengguna.
type UserKind string

// Known menyatakan apakah jenis ini salah satu dari dua yang sah.
func (j UserKind) Known() bool {
	return j == Employee || j == NonEmployee
}

// Credential adalah yang dimasukkan pengguna di layar masuk.
//
// Nilainya tidak pernah ditulis ke log dan tidak pernah disimpan di basis data —
// aplikasi ini tidak mengelola kata sandi karyawan sama sekali, dan untuk non-karyawan
// yang tersimpan hanyalah sidiknya, bukan kata sandinya.
type Credential struct {
	Username string
	Password string
}

// Profile adalah jawaban sistem identitas atas kredensial yang sah.
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
type Profile struct {
	// Identity adalah kunci alami pengguna: NIK untuk karyawan, LOGIN_ID untuk
	// non-karyawan.
	Identity string
	Name     string
	Kind     UserKind

	// Login adalah yang diketik pengguna di layar masuk — alamat surel untuk karyawan,
	// LOGIN_ID untuk non-karyawan. Disimpan supaya pengguna dapat ditelusuri dengan
	// yang ia kenal, bukan hanya dengan NIK.
	Login string

	// Field berikut OPSIONAL. Kosong berarti sumber identitasnya memang tidak
	// mengirimkannya — POOLDATA.M_LOGIN_PNC hanya memuat login_id dan nama.
	Email      string // HCQ: EmpResponse.Person.pyEmail1
	Company    string // HCQ: EmpResponse.Person.pyCompany
	Branch     string // HCQ: EmpResponse.Placement.BranchName
	BranchCode string // HCQ: EmpResponse.Placement.BranchCode
	Position   string // HCQ: EmpResponse.Placement.PositionName

	// ActiveAtSource adalah EmpResponse.Placement.IsActive apa adanya.
	//
	// Ia DIREKAM tetapi TIDAK dipakai menolak masuk: aturan yang ditetapkan Work Owner
	// hanya menyebut pyErrorCode = "200" sebagai syarat masuk. Menambah syarat sendiri
	// berarti mengarang aturan kewenangan. Yang menggerbang tetap kolom AKTIF pada
	// catatan pengguna lokal, yang dikelola administrator aplikasi ini.
	// Lihat pertanyaan terbuka di docs/keputusan-implementasi.md.
	ActiveAtSource *bool
}

// Identity adalah seam ke sistem autentikasi.
//
// Implementasinya hanya boleh menjawab satu pertanyaan: apakah kredensial ini sah, dan
// bila sah, siapa pemiliknya. Ia tidak tahu apa pun soal sesi maupun izin.
//
// Pengisinya ada di auth/provider: HCQ untuk karyawan, Lokal untuk non-karyawan, dan
// Berantai yang menggabungkan keduanya sesuai urutan yang ditetapkan bisnis.
type Identity interface {
	Verify(ctx context.Context, credential Credential) (Profile, error)
}

// Check memastikan profil layak dipakai.
//
// Identitas yang kosong ditolak karena pengguna tanpa kunci alami tidak dapat
// dicocokkan dengan data klaimnya sendiri — kegagalan yang jauh lebih mahal bila baru
// ketahuan setelah ia mengisi satu form registrasi penuh.
func (p Profile) Check() error {
	empty := []string{}
	if strings.TrimSpace(p.Identity) == "" {
		empty = append(empty, "identitas")
	}
	if strings.TrimSpace(p.Name) == "" {
		empty = append(empty, "nama")
	}
	if !p.Kind.Known() {
		empty = append(empty, "jenis")
	}
	if len(empty) == 0 {
		return nil
	}
	return &IncompleteProfileError{EmptyFields: empty}
}

// Complete menyatakan apakah kredensial terisi kedua-duanya.
func (k Credential) Complete() bool {
	return strings.TrimSpace(k.Username) != "" && k.Password != ""
}
