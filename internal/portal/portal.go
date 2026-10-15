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
	Name  string
	Alias string
}

// ErrNotFound dikembalikan bila alias yang diminta tidak ada di daftar.
var ErrNotFound = errors.New("portal: tidak ada di daftar")

// Repo adalah seam ke sumber daftar portal.
//
// Pengisinya ada di portal/repo/sqlstore dan portal/repo/memory.
type Repo interface {
	// List mengembalikan seluruh portal, terurut tetap.
	List(ctx context.Context) ([]Portal, error)
}

// Availability menandai portal yang koneksi basis datanya hidup.
//
// Sebuah portal dapat ada di tabel tetapi belum dapat dipakai, karena kredensial basis
// datanya belum diisi. Membedakan keduanya penting: pengguna melihat seluruh entitas
// yang direncanakan, dan yang belum siap ditandai — bukan disembunyikan seolah tidak
// pernah ada.
type Availability struct {
	Portal
	Ready bool
}

// MarkAvailable menggabungkan daftar portal dengan alias yang koneksinya hidup.
func MarkAvailable(list []Portal, readyAliases []string) []Availability {
	ready := make(map[string]bool, len(readyAliases))
	for _, a := range readyAliases {
		ready[normalize(a)] = true
	}

	result := make([]Availability, 0, len(list))
	for _, p := range list {
		result = append(result, Availability{Portal: p, Ready: ready[normalize(p.Alias)]})
	}
	return result
}

// Find menemukan satu portal berdasarkan aliasnya, tanpa peduli besar-kecil huruf.
//
// Penormalan huruf disengaja: `docs/Steering/11-SECURITY.md` §3.1 mencatat tiga nama
// access group Pega muncul dalam dua kapitalisasi berbeda, dan perbandingan di rule
// lama tidak konsisten soal itu. Kesalahan yang sama tidak diulang di sini.
func Find(list []Portal, alias string) (Portal, error) {
	wanted := normalize(alias)
	for _, p := range list {
		if normalize(p.Alias) == wanted {
			return p, nil
		}
	}
	return Portal{}, ErrNotFound
}

func normalize(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
