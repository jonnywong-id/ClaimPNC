package auth

import (
	"context"
	"time"
)

// User adalah catatan lokal satu orang. Identity adalah kunci alaminya.
//
// Sistem identitas luar memiliki identitasnya; aplikasi ini menyimpan salinannya agar
// pengguna dapat dirujuk oleh data klaim, peran, dan jejak audit bahkan ketika sistem
// identitas sedang tidak dapat dihubungi (docs/Steering/11-SECURITY.md §2.1 langkah 4).
//
// Catatan terbuka (ADR-0024, Pertanyaan terbuka nomor 4): cara mencocokkan identitas
// HCC/HCQ dengan OPERATOR_ID yang dipakai di seluruh data klaim belum ditetapkan.
// Sampai itu diputuskan, OperatorID dibiarkan kosong — bukan ditebak.
type User struct {
	Identity   string
	Kind       UserKind
	Name       string
	Login      string
	Email      string
	Company    string
	Branch     string
	BranchCode string
	Position   string
	OperatorID string

	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FromProfile membentuk catatan pengguna baru dari profil yang baru diverifikasi.
func FromProfile(p Profile, sekarang time.Time) User {
	return User{
		Identity:   p.Identity,
		Kind:       p.Kind,
		Name:       p.Name,
		Login:      p.Login,
		Email:      p.Email,
		Company:    p.Company,
		Branch:     p.Branch,
		BranchCode: p.BranchCode,
		Position:   p.Position,
		Active:     true,
		CreatedAt:  sekarang,
		UpdatedAt:  sekarang,
	}
}

// RefreshFrom menyalin field yang dimiliki sistem identitas ke catatan yang sudah ada.
//
// Yang TIDAK ikut tersalin: status aktif, OperatorID, dan waktu pembuatan. Ketiganya
// dimiliki administrator aplikasi ini, bukan sistem identitas luar — menimpanya setiap
// kali pengguna masuk akan menghidupkan kembali akun yang sengaja dinonaktifkan.
func (u *User) RefreshFrom(p Profile, sekarang time.Time) {
	u.Kind = p.Kind
	u.Name = p.Name
	u.Login = p.Login
	u.Email = p.Email
	u.Company = p.Company
	u.Branch = p.Branch
	u.BranchCode = p.BranchCode
	u.Position = p.Position
	u.UpdatedAt = sekarang
	if u.CreatedAt.IsZero() {
		u.CreatedAt = sekarang
	}
}

// UserRepo adalah seam ke tempat catatan pengguna disimpan.
//
// Pengisinya ada di auth/repo/sqlstore dan auth/repo/memory.
type UserRepo interface {
	// GetByIdentity mengembalikan ErrUserNotFound bila identitas itu
	// belum pernah tercatat.
	GetByIdentity(ctx context.Context, identity string) (User, error)

	// Save menulis catatan pengguna berdasarkan Identity: menyisipkan
	// bila belum ada, memperbarui bila sudah. Ia tidak pernah menghapus.
	Save(ctx context.Context, p User) error
}
