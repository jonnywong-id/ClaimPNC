package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/inboxlaporanklaim"
)

// BranchResolver memenuhi seam inboxlaporanklaim.BranchResolver di dalam memori.
//
// Ia meniru satu perilaku yang paling mudah terlewat pada adapter SQL: login yang TIDAK
// terdaftar menghasilkan "tidak ditemukan", bukan galat, dan bukan pula kode kosong yang
// tampak sah. Ketiganya berakibat berbeda pada batas data, dan seam yang kedua
// pengisinya tidak sepakat soal itu adalah seam yang menyembunyikan cacat.
type BranchResolver struct {
	mutex   sync.Mutex
	byLogin map[string]string
	failure error
}

// NewBranchResolver membentuk penerjemah berisi pemetaan yang diberikan.
func NewBranchResolver(byLogin map[string]string) *BranchResolver {
	copied := map[string]string{}
	for login, code := range byLogin {
		copied[strings.ToUpper(strings.TrimSpace(login))] = code
	}
	return &BranchResolver{byLogin: copied}
}

// SetError membuat penerjemah menjawab dengan galat, untuk menguji jalur gagal.
//
// Ia dipakai membuktikan bahwa kegagalan menerjemahkan cabang TIDAK mengosongkan daftar —
// perilaku yang justru menjadi sebab cacat yang diperbaiki sesi ini.
func (r *BranchResolver) SetError(err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.failure = err
}

// Resolve menerjemahkan login petugas menjadi kode cabang klaimnya.
func (r *BranchResolver) Resolve(_ context.Context, login string) (string, bool, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.failure != nil {
		return "", false, r.failure
	}

	code, found := r.byLogin[strings.ToUpper(strings.TrimSpace(login))]
	if !found || strings.TrimSpace(code) == "" {
		return "", false, nil
	}
	return code, true, nil
}

// SampleBranchOfLogin memetakan login contoh ke cabangnya.
//
// Kedua login ini ada di provider identitas tiruan. `penggunanonaktif` dan `profilbolong`
// SENGAJA tidak dimasukkan: keduanya mewakili petugas yang cabangnya tidak dapat
// ditentukan, dan layar harus tetap menampilkan daftar untuk mereka — bukan mengosongkannya.
func SampleBranchOfLogin() map[string]string {
	return map[string]string{
		"adminpnc":   "1001",
		"pictekniks": "1002",
	}
}

var _ inboxlaporanklaim.BranchResolver = (*BranchResolver)(nil)
