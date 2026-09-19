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

// FakeUser adalah satu baris daftar pengguna contoh.
//
// Nama-nama di daftar bawaan sengaja bukan nama pegawai nyata.
type FakeUser struct {
	Username string
	Password string
	Active   bool
	Profile  auth.Profile
}

// Fake adalah provider identitas berbasis daftar pengguna di memori.
//
// Ia tidak menyentuh jaringan maupun basis data sama sekali, sehingga seluruh modul
// yang membutuhkan identitas dapat diuji dengan jaringan dimatikan.
type Fake struct {
	user map[string]FakeUser
	// simulateOutage membuat provider menjawab seolah sistem identitas tidak dapat
	// dihubungi. Ia ada supaya jalur galat ketiga dapat dicoba di layar masuk tanpa
	// harus benar-benar mematikan apa pun.
	simulateOutage bool
}

// ErrFakeInProduction adalah penolakan yang membuat aplikasi gagal start.
var ErrFakeInProduction = errors.New("provider identitas tiruan menolak berjalan di lingkungan produksi")

// NewFake membentuk provider tiruan.
//
// Penolakan terhadap produksi berada DI SINI, di dalam kode — bukan sekadar nilai
// konfigurasi yang bisa dibalik seseorang. Siapa pun yang menyalakan provider ini di
// produksi mendapatkan aplikasi yang tidak mau start, bukan aplikasi yang menerima
// kata sandi palsu.
func NewFake(productionEnv bool, list []FakeUser) (*Fake, error) {
	if productionEnv {
		return nil, ErrFakeInProduction
	}
	if len(list) == 0 {
		list = SampleList()
	}
	index := make(map[string]FakeUser, len(list))
	for _, p := range list {
		key := strings.ToLower(strings.TrimSpace(p.Username))
		if key == "" {
			return nil, fmt.Errorf("provider identitas tiruan: ada baris tanpa nama pengguna")
		}
		index[key] = p
	}
	return &Fake{user: index}, nil
}

// SetSimulateOutage menyalakan atau mematikan simulasi sistem identitas yang tidak
// dapat dihubungi.
func (t *Fake) SetSimulateOutage(decision bool) { t.simulateOutage = decision }

// Verify memeriksa kredensial terhadap daftar pengguna contoh.
//
// Kata sandi dibandingkan apa adanya karena provider ini tidak pernah memegang kata
// sandi sungguhan; yang memegangnya adalah HCC/HCQ dan M_LOGIN_PNC.
func (t *Fake) Verify(ctx context.Context, k auth.Credential) (auth.Profile, error) {
	if err := ctx.Err(); err != nil {
		return auth.Profile{}, fmt.Errorf("%w: %v", auth.ErrIdentitySystemDown, err)
	}
	if t.simulateOutage {
		return auth.Profile{}, auth.ErrIdentitySystemDown
	}

	p, ok := t.user[strings.ToLower(strings.TrimSpace(k.Username))]
	if !ok {
		// Pengguna yang tidak ada dan kata sandi yang salah menghasilkan galat yang
		// sama persis. Membedakannya membocorkan siapa saja yang punya akun.
		return auth.Profile{}, auth.ErrWrongCredential
	}
	if p.Password != k.Password {
		return auth.Profile{}, auth.ErrWrongCredential
	}
	if !p.Active {
		return auth.Profile{}, auth.ErrUserInactive
	}
	return p.Profile, nil
}

// SampleList adalah pengguna bawaan untuk pengembangan.
//
// Seluruh nama, NIK, dan email di sini karangan — bukan pegawai nyata, bukan alamat
// surel nyata. Daftarnya memuat kedua jenis pengguna supaya perbedaan karyawan dan
// non-karyawan dapat dicoba tanpa basis data.
func SampleList() []FakeUser {
	return []FakeUser{
		{
			Username: "adminpnc",
			Password: "rahasia123",
			Active:   true,
			Profile: auth.Profile{
				Identity: "90000001",
				Name:     "Contoh Administrator",
				Kind:     auth.Employee,
				Login:    "adminpnc",
				Email:    "contoh.admin@example.invalid",
				Company:  "ASM",
			},
		},
		{
			Username: "pictekniks",
			Password: "rahasia123",
			Active:   true,
			Profile: auth.Profile{
				Identity: "90000002",
				Name:     "Contoh PIC Teknik",
				Kind:     auth.Employee,
				Login:    "pictekniks",
				Email:    "contoh.picteknik@example.invalid",
				Company:  "ASM",
			},
		},
		{
			// Non-karyawan: hanya punya LOGIN_ID dan nama, tanpa NIK dan tanpa email —
			// persis sebatas yang dimuat POOLDATA.M_LOGIN_PNC.
			Username: "brokercontoh",
			Password: "rahasia123",
			Active:   true,
			Profile: auth.Profile{
				Identity: "BROKERCONTOH",
				Name:     "Contoh Broker Rekanan",
				Kind:     auth.NonEmployee,
				Login:    "BROKERCONTOH",
			},
		},
		{
			Username: "penggunanonaktif",
			Password: "rahasia123",
			Active:   false,
			Profile: auth.Profile{
				Identity: "90000003",
				Name:     "Contoh Pengguna Nonaktif",
				Kind:     auth.Employee,
				Login:    "penggunanonaktif",
				Email:    "contoh.nonaktif@example.invalid",
			},
		},
		{
			// Dipakai untuk membuktikan bahwa profil tidak lengkap ditolak, bukan
			// diteruskan diam-diam: sumber menjawab "berhasil" tanpa nama.
			Username: "profilbolong",
			Password: "rahasia123",
			Active:   true,
			Profile: auth.Profile{
				Identity: "90000004",
				Name:     "",
				Kind:     auth.Employee,
				Login:    "profilbolong",
			},
		},
	}
}
