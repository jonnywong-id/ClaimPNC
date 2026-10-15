package memory

import (
	"context"
	"strings"
	"sync"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// BranchResolver memenuhi seam inboxkomunikasicabang.BranchResolver di dalam memori.
//
// Ia meniru satu perilaku yang paling mudah terlewat pada adapter SQL: login yang TIDAK
// terdaftar menghasilkan "tidak ditemukan", bukan galat, dan bukan pula kode kosong yang
// tampak sah. Ketiganya berakibat berbeda pada batas data, dan seam yang kedua pengisinya
// tidak sepakat soal itu adalah seam yang menyembunyikan cacat.
//
// Di layar ini pembedaannya menentukan APA YANG DILIHAT petugas, bukan sekadar apakah ia
// dilayani: "tidak ditemukan" menjatuhkannya ke percakapan kantor pusat (`P-5`), sementara
// galat menutup layarnya.
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

// NewSampleBranchResolver membentuk penerjemah berisi pemetaan contoh.
func NewSampleBranchResolver() *BranchResolver {
	return NewBranchResolver(SampleBranchOfLogin())
}

// SetError membuat penerjemah menjawab dengan galat, untuk menguji jalur gagal.
//
// Ia dipakai membuktikan bahwa sumber cabang yang MATI menutup layar — berbeda dari petugas
// yang sekadar tidak terdaftar, yang tetap dilayani sebagai kantor pusat. Tanpa uji itu,
// kedua keadaan dapat tertukar tanpa satu pun tanda.
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
// Ketiga login pertama ada di provider identitas tiruan. `penggunanonaktif` dan
// `profilbolong` SENGAJA tidak dimasukkan: keduanya mewakili petugas yang cabangnya tidak
// dapat ditentukan, dan di layar INI mereka tetap dilayani — sebagai kantor pusat (`P-5`).
//
// Itu berbeda dari modul Inbox Laporan Klaim, yang menolak keduanya
// (`keputusan-implementasi.md` §20). Perbedaannya disengaja dan merupakan keputusan Work
// Owner 2026-09-24; kedua daftar contoh sengaja dibiarkan menyatakan hal yang berbeda
// supaya perbedaan keputusannya ikut teruji.
//
// `adminpnc` dipetakan ke kode kantor pusat supaya jalur kantor pusat punya saksi yang
// cabangnya BENAR-BENAR terbaca — tanpa itu, jalur itu hanya dapat dicapai lewat kegagalan,
// dan keduanya tidak akan dapat dibedakan di uji.
func SampleBranchOfLogin() map[string]string {
	return map[string]string{
		"adminpnc":   inboxkomunikasicabang.HeadOfficeBranch,
		"pictekniks": "1001",
		"picteknikb": "1002",
	}
}

var _ inboxkomunikasicabang.BranchResolver = (*BranchResolver)(nil)
