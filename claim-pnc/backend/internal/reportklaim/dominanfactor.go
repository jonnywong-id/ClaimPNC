package reportklaim

import "strings"

// DominantFactors adalah daftar faktor dominan per klaim.
//
// # Kenapa dirangkai di Go, bukan di SQL
//
// Karena tidak ada satu pun fungsi perangkai yang berjalan di KEDUA basis data. Oracle
// punya `LISTAGG`, PostgreSQL punya `STRING_AGG`, dan `09-DATABASE-STRATEGY.md` §4
// melarang yang pertama justru karena tidak portabel — sementara yang kedua tidak ada di
// Oracle, tempat aplikasi ini berjalan hari ini.
//
// Merangkainya di Go menghapus persoalan itu seluruhnya, dan sekaligus menempatkan aturan
// perangkaiannya — pemisah satu spasi, urutan menurut `idx_dominanfactor` — di tempat yang
// dapat diuji tanpa basis data.
//
// # Kenapa dibaca sekali per laporan
//
// Sama dengan tangga fee dan nama tahapan progres: satu pembacaan untuk seluruh rentang
// laporan, bukan satu anak-kueri per baris. Sistem lama justru lebih mahal lagi — ia
// MEMBUKA OBJEK KERJA setiap klaim untuk menelusuri page list-nya.
type DominantFactors struct {
	byClaim map[string][]string
	loaded  bool
}

// NewDominantFactors membentuk daftar dari pasangan (claimid, nama) yang SUDAH terurut
// menurut `idx_dominanfactor`.
//
// Urutannya dijaga kueri, bukan di sini: pengurutan ulang di Go menuntut membawa indeksnya
// serta, dan itu menduakan satu-satunya sumber urutan.
func NewDominantFactors(rows [][2]string) *DominantFactors {
	d := &DominantFactors{byClaim: map[string][]string{}, loaded: true}
	for _, r := range rows {
		claim, name := r[0], r[1]
		if claim == "" || name == "" {
			continue
		}
		d.byClaim[claim] = append(d.byClaim[claim], name)
	}
	return d
}

// UnavailableDominantFactors adalah daftar yang sumbernya tidak dapat dibaca.
func UnavailableDominantFactors() *DominantFactors { return &DominantFactors{} }

// Available menyatakan apakah daftar ini terisi dari sumbernya.
func (d *DominantFactors) Available() bool { return d != nil && d.loaded }

// Count mengembalikan banyaknya klaim yang punya faktor dominan.
func (d *DominantFactors) Count() int {
	if d == nil {
		return 0
	}
	return len(d.byClaim)
}

// Names merangkai nama faktor dominan sebuah klaim, dipisah satu spasi.
//
// # Bentuknya disalin dari akumulator sistem lama
//
// `TempDominan.City = TempDominan.City + " " + .DominanName`, dijalankan sekali per baris
// page list `ClaimData.DominanFactorList`.
//
// Dua hal yang TIDAK ditiru, dan keduanya disebut supaya tidak terbaca sebagai kelalaian:
//
//  1. Akumulator itu menambahkan spasi SEBELUM setiap nama, sehingga hasilnya berawalan
//     satu spasi. Di sini spasi hanya berada DI ANTARA nama.
//
//  2. Export tidak memperlihatkan apakah akumulatornya dikosongkan pada setiap baris
//     laporan. Bila tidak, nilainya menumpuk lintas baris — dan tumpukan seperti itu
//     bukan sesuatu yang layak ditiru meski ternyata memang terjadi.
//
// Klaim tanpa faktor dominan menghasilkan teks kosong.
func (d *DominantFactors) Names(claimID string) string {
	if !d.Available() {
		return ""
	}
	return strings.Join(d.byClaim[claimID], " ")
}
