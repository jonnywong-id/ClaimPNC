// Package sqlstore memenuhi seam penyimpanan Master Rekening dengan SQL.
//
// Dua aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke
//     dalam teks SQL — itu tepat kegagalan yang diwarisi rule lama, yang menyusun
//     klausa WHERE-nya dari properti klipboard lewat {ASIS:...}.
package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var berkasKueri embed.FS

// kueri memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var kueri = muatSeluruhKueri()

// ambilKueri mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func ambilKueri(nama string) string {
	teks, ada := kueri[nama]
	if !ada {
		panic(fmt.Sprintf("masterrekening/sqlstore: kueri %q tidak ditemukan di berkas .sql", nama))
	}
	return teks
}

func muatSeluruhKueri() map[string]string {
	hasil := map[string]string{}
	daftar, err := berkasKueri.ReadDir(".")
	if err != nil {
		panic("masterrekening/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, berkas := range daftar {
		isi, err := berkasKueri.ReadFile(berkas.Name())
		if err != nil {
			panic("masterrekening/sqlstore: tidak dapat membaca " + berkas.Name() + ": " + err.Error())
		}
		for nama, teks := range pecahPerNama(string(isi)) {
			if _, bentrok := hasil[nama]; bentrok {
				panic("masterrekening/sqlstore: nama kueri ganda: " + nama)
			}
			hasil[nama] = teks
		}
	}
	return hasil
}

// pecahPerNama memecah isi berkas pada penanda "-- name: <nama>". Satu berkas karena
// itu dapat memuat beberapa pernyataan dan tetap terbaca sebagai satu kesatuan saat
// di-review.
//
// Komentar yang berdiri sendiri SETELAH sebuah pernyataan — penjelasan kenapa kueri
// itu ditulis begitu — dibuang dari teks yang dikirim ke basis data. Penjelasannya
// tetap terbaca di berkas .sql tempatnya berguna, tanpa ikut terbawa ke jaringan pada
// setiap pemanggilan.
func pecahPerNama(isi string) map[string]string {
	const penanda = "-- name:"
	hasil := map[string]string{}
	nama := ""
	var badan []string

	simpan := func() {
		if nama == "" {
			return
		}
		if teks := strings.TrimSpace(strings.Join(buangKomentarEkor(badan), "\n")); teks != "" {
			hasil[nama] = teks
		}
	}
	for _, baris := range strings.Split(isi, "\n") {
		if potong := strings.TrimSpace(baris); strings.HasPrefix(potong, penanda) {
			simpan()
			nama = strings.TrimSpace(strings.TrimPrefix(potong, penanda))
			badan = nil
			continue
		}
		badan = append(badan, baris)
	}
	simpan()
	return hasil
}

// buangKomentarEkor membuang baris kosong dan baris komentar di ujung sebuah kueri.
//
// Komentar di TENGAH pernyataan dibiarkan — ia menjelaskan baris di sekitarnya dan
// membuangnya akan merusak pernyataan yang memuat komentar sebaris.
func buangKomentarEkor(baris []string) []string {
	akhir := len(baris)
	for akhir > 0 {
		potong := strings.TrimSpace(baris[akhir-1])
		if potong == "" || strings.HasPrefix(potong, "--") {
			akhir--
			continue
		}
		break
	}
	return baris[:akhir]
}
