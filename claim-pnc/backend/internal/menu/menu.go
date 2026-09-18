// Package menu menyusun peta menu aplikasi beserta kewenangan pemakainya.
//
// # Apa yang dimodelkan di sini
//
// Satu aplikasi (POOLDATA.M_APLIKASI) memiliki sederet butir menu
// (POOLDATA.M_MENU_APLIKASI_PNC) yang tersusun berjenjang lewat MENU_ID_LEADER dan
// berurutan lewat MENU_SEQUENCE. Siapa boleh membuka butir yang mana ditetapkan
// POOLDATA.M_OTORISASI_PNC, yang memberi izin kepada **login** maupun kepada **group**;
// keanggotaan group sendiri ada di POOLDATA.M_LOGIN_GROUP_PNC.
//
// # Ini kemampuan BARU, bukan pemindahan perilaku Pega
//
// Ketiga tabel di atas **tidak dipakai satu pun rule di export Pega** — sudah dicari ke
// seluruh 2.634 berkas XML. DDL-nya datang sebagai `Database/CREATE_MENU.sql`, yaitu
// skrip pembuatan tabel untuk aplikasi baru ini, bersama isinya dalam bentuk CSV.
//
// Akibatnya dua hal, dan keduanya harus disebut terang:
//
//   - Tidak ada baseline Pega untuk diuji kesetaraannya. Gerbang 1 modul ini karena itu
//     diganti uji fungsional terhadap kontrak — pola yang sama dengan `F-3` dan `S-5`
//     pada `D-56`.
//   - `D-59` menetapkan satuan izin adalah MENU, dan `TKT-F3-004` selama ini terhalang
//     karena "penugasan operator ke peran tidak ada di basis data". Tabel-tabel ini
//     menjawab tepat kekosongan itu, tetapi dengan model yang BERBEDA dari 22 access
//     group Pega (`D-58`): di sini subjeknya login dan group, bukan peran.
//
// # Yang TIDAK dijawab paket ini
//
// Paket ini menjawab "butir menu mana yang boleh dilihat pemanggil". Ia TIDAK menjawab
// "layarnya sudah dibangun atau belum" — itu diketahui frontend, yang memegang peta
// rutenya. Menyembunyikan menu juga BUKAN kendali akses (`D-59`): yang menggerbang
// tetap pemeriksaan di server pada setiap endpoint modulnya masing-masing.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package menu

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// AppName adalah nilai M_APLIKASI.APP_DESC milik aplikasi ini.
//
// Ia dipakai sebagai penyaring, bukan APP_ID, mengikuti kueri yang ditetapkan Work
// Owner. Alasannya masuk akal: APP_ID adalah nomor urut yang dapat berbeda antar
// lingkungan, sedangkan nama aplikasinya tidak.
const AppName = "CLAIM PNC"

// ErrAppNotFound: tidak ada baris M_APLIKASI dengan APP_DESC yang diminta.
var ErrAppNotFound = errors.New("menu: aplikasi tidak ditemukan di M_APLIKASI")

// Item adalah satu baris M_MENU_APLIKASI_PNC.
type Item struct {
	// ID adalah kolom MENU_ID, unik di dalam satu aplikasi.
	ID int

	// Description adalah kolom MENU_DESC — teks yang dibaca pengguna di menu.
	Description string

	// Program adalah kolom MENU_PROGRAM: nama harness Pega yang dituju butir ini.
	//
	// Kosong berarti butir ini tidak menuju layar mana pun. Ada DUA sebab yang berbeda,
	// dan keduanya nyata di data yang diterima:
	//
	//	MENU_ID 1..4   induk kelompok (MASTER, INBOX, VIEW, REPORT) — memang judul
	//	MENU_ID 83     "Report Adjuster" — daun berinduk REPORT, tetapi tanpa program
	//
	// Yang kedua bukan kelalaian pembacaan: barisnya memang kosong di sumbernya.
	Program string

	// ParentID adalah kolom MENU_ID_LEADER. Nil berarti butir ini kelompok tingkat atas.
	ParentID *int

	// Sequence adalah kolom MENU_SEQUENCE, penentu urutan tampil.
	Sequence int
}

// IsGroup menyatakan butir ini kelompok tingkat atas, bukan butir yang dapat dibuka.
func (i Item) IsGroup() bool { return i.ParentID == nil }

// Node adalah satu butir menu beserta anak-anaknya, siap dikirim ke layar.
type Node struct {
	Item
	Children []Node
}

// Repo adalah seam ke penyimpanan peta menu dan kewenangannya.
//
// Ketiga methodnya sengaja dipisah alih-alih disatukan menjadi satu kueri besar
// ber-join: urutan langkahnya ditetapkan Work Owner (cari group dulu, baru cari izin
// untuk group DAN untuk login), dan memisahkannya membuat urutan itu terbaca di kode
// sebagaimana ia dijelaskan. Ketiganya ringan — tabelnya berisi puluhan baris, bukan
// jutaan.
type Repo interface {
	// List mengembalikan SELURUH butir menu aplikasi, terurut MENU_SEQUENCE.
	//
	// Ia tidak menyaring kewenangan: penyaringannya ada di domain, supaya aturan
	// "induk tampil bila ada anaknya yang tampil" hidup di satu tempat dan dapat diuji
	// tanpa basis data.
	List(ctx context.Context, appName string) ([]Item, error)

	// GroupsOf mengembalikan GROUP_ID yang diikuti sebuah login.
	//
	// Senarai kosong berarti login itu tidak tergabung di group mana pun — bukan galat.
	GroupsOf(ctx context.Context, loginID string) ([]string, error)

	// AuthorizedIDs mengembalikan MENU_ID yang diizinkan untuk subjek-subjek yang
	// disebut. Satu subjek adalah sebuah LOGIN_ID_GROUP: boleh nama login, boleh nama
	// group — kolomnya satu dan menampung keduanya.
	AuthorizedIDs(ctx context.Context, appName string, subjects []string) ([]int, error)
}

// Subjects menyusun daftar subjek otorisasi untuk satu login.
//
// Urutannya mengikuti langkah yang ditetapkan Work Owner 2026-09-18: dari login yang
// diketik, cari GROUP_ID-nya, lalu cari izin untuk group-group itu DAN untuk login itu
// sendiri. Keduanya digabung — izin yang diberikan langsung kepada seseorang berlaku di
// samping izin yang ia warisi dari groupnya, bukan menggantikannya.
//
// Nilai dipangkas dan diseragamkan menjadi huruf besar karena kolomnya VARCHAR2 tanpa
// penyeragaman apa pun, dan satu spasi di ujung akan membuat baris yang sah tidak
// pernah cocok.
func Subjects(loginID string, groups []string) []string {
	seen := map[string]bool{}
	var result []string

	add := func(value string) {
		clean := normalize(value)
		if clean == "" || seen[clean] {
			return
		}
		seen[clean] = true
		result = append(result, clean)
	}

	for _, group := range groups {
		add(group)
	}
	add(loginID)

	return result
}

// BuildTree menyusun butir menu menjadi pohon, hanya yang boleh dilihat pemanggil.
//
// # Aturan tampil
//
//   - Butir yang dapat dibuka (punya induk) tampil bila MENU_ID-nya ada di authorized.
//   - Kelompok tingkat atas tampil bila ada SEKURANG-KURANGNYA SATU anaknya yang tampil.
//
// Aturan kedua bukan pilihan gaya melainkan bacaan atas data yang diterima: group `IT`
// diberi izin atas MENU_ID 11..86 dan TIDAK satu pun atas MENU_ID 1..4. Menuntut
// kelompoknya punya baris izin sendiri akan membuat seluruh menunya hilang.
//
// Sebaliknya, kelompok yang punya baris izin tetapi seluruh anaknya tidak — dan itu pun
// ada di data, login `JONNY` diberi izin atas MENU_ID 4 — tetap disembunyikan bila
// anaknya kosong. Judul kelompok yang tidak membuka apa pun hanya menambah barang di
// layar.
//
// # Urutan
//
// Seluruh tingkat diurutkan MENU_SEQUENCE menaik, sesuai kueri yang ditetapkan Work
// Owner. Pengurutannya stabil, sehingga dua butir bersequence sama tetap tampil dalam
// urutan yang sama pada setiap permintaan.
func BuildTree(items []Item, authorized []int) []Node {
	allowed := make(map[int]bool, len(authorized))
	for _, id := range authorized {
		allowed[id] = true
	}

	sorted := make([]Item, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(a, b int) bool { return sorted[a].Sequence < sorted[b].Sequence })

	childrenOf := map[int][]Item{}
	var roots []Item
	for _, item := range sorted {
		if item.ParentID == nil {
			roots = append(roots, item)
			continue
		}
		childrenOf[*item.ParentID] = append(childrenOf[*item.ParentID], item)
	}

	var build func(parent Item) (Node, bool)
	build = func(parent Item) (Node, bool) {
		node := Node{Item: parent}
		for _, child := range childrenOf[parent.ID] {
			if grandchild, visible := build(child); visible {
				node.Children = append(node.Children, grandchild)
			}
		}
		if len(node.Children) > 0 {
			return node, true
		}
		// Daun: yang menentukan hanyalah izinnya sendiri. Kelompok yang seluruh anaknya
		// tersaring karena itu ikut hilang, walau barisnya sendiri diizinkan.
		//
		// Daun yang TIDAK punya program pun tetap tampil bila diizinkan — MENU_ID 83
		// "Report Adjuster" adalah contohnya. Ia akan terlihat sebagai butir yang tidak
		// dapat diklik, dan itu memang keadaannya: barisnya ada di master, tujuannya
		// tidak. Menyembunyikannya justru mengubur kekosongan data yang perlu dilihat.
		return node, !parent.IsGroup() && allowed[parent.ID]
	}

	var result []Node
	for _, root := range roots {
		if node, visible := build(root); visible {
			result = append(result, node)
		}
	}
	return result
}

func normalize(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
