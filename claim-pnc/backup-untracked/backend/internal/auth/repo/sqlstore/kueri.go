// Package sqlstore memenuhi seam penyimpanan dengan SQL terhadap basis data relasional.
//
// Dua aturan mengikat seluruh berkas di sini:
//   - Teks SQL berada di berkas .sql terpisah, bukan string di tengah kode Go, supaya
//     dapat dibaca, di-review, dan dijalankan langsung terhadap basis data.
//   - Nilai selalu lewat parameter binding. Tidak pernah ada perangkaian nilai ke
//     dalam teks SQL.
package sqlstore

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.sql
var berkasKueri embed.FS

// kueri memuat seluruh pernyataan SQL, dikunci dengan namanya.
var kueri = muatSeluruhKueri()

// ambilKueri mengembalikan teks SQL bernama tertentu dan panik bila namanya tidak ada.
//
// Panik di sini disengaja dan aman: nama kueri adalah konstanta di dalam kode, bukan
// masukan pengguna, sehingga ketiadaannya adalah cacat pemrograman yang harus terlihat
// saat pertama dijalankan — bukan galat runtime yang menunggu pengguna menemukannya.
func ambilKueri(nama string) string {
	teks, ada := kueri[nama]
	if !ada {
		panic(fmt.Sprintf("sqlstore: kueri %q tidak ditemukan di berkas .sql", nama))
	}
	return teks
}

// muatSeluruhKueri membaca setiap berkas .sql dan memecahnya pada penanda
// "-- name: <nama>". Satu berkas karena itu dapat memuat beberapa pernyataan dan tetap
// terbaca sebagai satu kesatuan saat di-review.
func muatSeluruhKueri() map[string]string {
	hasil := map[string]string{}
	daftar, err := berkasKueri.ReadDir(".")
	if err != nil {
		panic("sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, berkas := range daftar {
		isi, err := berkasKueri.ReadFile(berkas.Name())
		if err != nil {
			panic("sqlstore: tidak dapat membaca " + berkas.Name() + ": " + err.Error())
		}
		for nama, teks := range pecahPerNama(string(isi)) {
			if _, bentrok := hasil[nama]; bentrok {
				panic("sqlstore: nama kueri ganda: " + nama)
			}
			hasil[nama] = teks
		}
	}
	return hasil
}

func pecahPerNama(isi string) map[string]string {
	const penanda = "-- name:"
	hasil := map[string]string{}
	nama := ""
	var badan []string

	simpan := func() {
		if nama != "" {
			if teks := strings.TrimSpace(strings.Join(badan, "\n")); teks != "" {
				hasil[nama] = teks
			}
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
