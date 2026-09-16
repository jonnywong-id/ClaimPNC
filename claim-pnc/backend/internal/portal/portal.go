// Package portal adalah inti modul Portal Multi-Entitas (`F-6`).
//
// # Apa yang dimodelkan di sini
//
// Satu aplikasi melayani beberapa entitas Sinarmas, masing-masing dengan basis datanya
// sendiri (ADR-0030). Portal adalah entitas itu: Asuransi Sinar Mas, Asuransi Simas
// Insurtech, Sinarmas Asuransi Syariah, dan seterusnya.
//
// # Kenapa daftarnya data, bukan konstanta
//
// Sistem lama membedakan entitas dengan **membandingkan nama server** — 48 perbandingan
// terhadap `pxRequestor.pxReqServer` pada tiga hostname, tidak terlihat pengguna dan
// tidak pernah tercatat sebagai rancangan. Salah satunya bahkan mengubah ambang komite
// dari Rp 50.000.000 menjadi 3.500, yang ternyata mata uang berbeda.
//
// ADR-0030 menetapkan perilaku itu dibuat eksplisit dan daftarnya menjadi **data**:
// ia dibaca dari POOLDATA.M_PORTAL_PNC, bukan ditulis di kode. Menambah entitas berarti
// menambah baris tabel dan lima variabel koneksi — bukan merilis ulang aplikasi.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package portal

import (
	"context"
	"errors"
	"strings"
)

// Portal adalah satu entitas yang dilayani aplikasi ini.
type Portal struct {
	ID    string
	Nama  string
	Alias string
}

// ErrTidakAda dikembalikan bila alias yang diminta tidak ada di daftar.
var ErrTidakAda = errors.New("portal: tidak ada di daftar")

// Repo adalah seam ke sumber daftar portal.
//
// Pengisinya ada di portal/repo/sqlstore dan portal/repo/memori.
type Repo interface {
	// Daftar mengembalikan seluruh portal, terurut tetap.
	Daftar(ctx context.Context) ([]Portal, error)
}

// Tersedia menandai portal yang koneksi basis datanya hidup.
//
// Sebuah portal dapat ada di tabel tetapi belum dapat dipakai, karena kredensial basis
// datanya belum diisi. Membedakan keduanya penting: pengguna melihat seluruh entitas
// yang direncanakan, dan yang belum siap ditandai — bukan disembunyikan seolah tidak
// pernah ada.
type Tersedia struct {
	Portal
	Siap bool
}

// TandaiTersedia menggabungkan daftar portal dengan alias yang koneksinya hidup.
func TandaiTersedia(daftar []Portal, aliasSiap []string) []Tersedia {
	siap := make(map[string]bool, len(aliasSiap))
	for _, a := range aliasSiap {
		siap[normalkan(a)] = true
	}

	hasil := make([]Tersedia, 0, len(daftar))
	for _, p := range daftar {
		hasil = append(hasil, Tersedia{Portal: p, Siap: siap[normalkan(p.Alias)]})
	}
	return hasil
}

// Cari menemukan satu portal berdasarkan aliasnya, tanpa peduli besar-kecil huruf.
//
// Penormalan huruf disengaja: `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama
// access group Pega muncul dalam dua kapitalisasi berbeda, dan perbandingan di rule
// lama tidak konsisten soal itu. Kesalahan yang sama tidak diulang di sini.
func Cari(daftar []Portal, alias string) (Portal, error) {
	dicari := normalkan(alias)
	for _, p := range daftar {
		if normalkan(p.Alias) == dicari {
			return p, nil
		}
	}
	return Portal{}, ErrTidakAda
}

func normalkan(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
