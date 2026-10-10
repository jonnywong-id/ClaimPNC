package auth

import (
	"context"
	"time"
)

// Pengguna adalah catatan lokal satu orang. Identitas adalah kunci alaminya.
//
// Sistem identitas luar memiliki identitasnya; aplikasi ini menyimpan salinannya agar
// pengguna dapat dirujuk oleh data klaim, peran, dan jejak audit bahkan ketika sistem
// identitas sedang tidak dapat dihubungi (docs/Steering/11-SECURITY.md §2.1 langkah 4).
//
// Catatan terbuka (ADR-0024, Pertanyaan terbuka nomor 4): cara mencocokkan identitas
// HCC/HCQ dengan OPERATOR_ID yang dipakai di seluruh data klaim belum ditetapkan.
// Sampai itu diputuskan, OperatorID dibiarkan kosong — bukan ditebak.
type Pengguna struct {
	Identitas  string
	Jenis      JenisPengguna
	Nama       string
	Login      string
	Email      string
	Perusahaan string
	Cabang     string
	KodeCabang string
	Jabatan    string
	OperatorID string

	Aktif          bool
	DibuatPada     time.Time
	DiperbaruiPada time.Time
}

// DariProfil membentuk catatan pengguna baru dari profil yang baru diverifikasi.
func DariProfil(p Profil, sekarang time.Time) Pengguna {
	return Pengguna{
		Identitas:      p.Identitas,
		Jenis:          p.Jenis,
		Nama:           p.Nama,
		Login:          p.Login,
		Email:          p.Email,
		Perusahaan:     p.Perusahaan,
		Cabang:         p.Cabang,
		KodeCabang:     p.KodeCabang,
		Jabatan:        p.Jabatan,
		Aktif:          true,
		DibuatPada:     sekarang,
		DiperbaruiPada: sekarang,
	}
}

// SegarkanDari menyalin field yang dimiliki sistem identitas ke catatan yang sudah ada.
//
// Yang TIDAK ikut tersalin: status aktif, OperatorID, dan waktu pembuatan. Ketiganya
// dimiliki administrator aplikasi ini, bukan sistem identitas luar — menimpanya setiap
// kali pengguna masuk akan menghidupkan kembali akun yang sengaja dinonaktifkan.
func (u *Pengguna) SegarkanDari(p Profil, sekarang time.Time) {
	u.Jenis = p.Jenis
	u.Nama = p.Nama
	u.Login = p.Login
	u.Email = p.Email
	u.Perusahaan = p.Perusahaan
	u.Cabang = p.Cabang
	u.KodeCabang = p.KodeCabang
	u.Jabatan = p.Jabatan
	u.DiperbaruiPada = sekarang
	if u.DibuatPada.IsZero() {
		u.DibuatPada = sekarang
	}
}

// PenggunaRepo adalah seam ke tempat catatan pengguna disimpan.
//
// Pengisinya ada di auth/repo/sqlstore dan auth/repo/memori.
type PenggunaRepo interface {
	// AmbilByIdentitas mengembalikan ErrPenggunaTidakDitemukan bila identitas itu
	// belum pernah tercatat.
	AmbilByIdentitas(ctx context.Context, identitas string) (Pengguna, error)

	// SimpanAtauPerbarui menulis catatan pengguna berdasarkan Identitas: menyisipkan
	// bila belum ada, memperbarui bila sudah. Ia tidak pernah menghapus.
	SimpanAtauPerbarui(ctx context.Context, p Pengguna) error
}
