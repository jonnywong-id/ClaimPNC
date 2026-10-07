// Package sampledata memuat data contoh repositori memori dari berkas JSON yang disematkan.
//
// Data contoh adalah DATA, bukan logika. Menuliskannya sebagai literal Go membuat ratusan
// baris berbentuk sama tersebar di banyak paket; menyimpannya sebagai JSON membuat isinya
// terbaca sebagai tabel dan paket memori tinggal memuatnya. Data contoh hanya dipakai saat
// aplikasi berjalan tanpa basis data dan di uji, bukan di produksi.
package sampledata

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Must mengurai bagian name dari berkas JSON raw — sebuah objek yang kuncinya nama data
// contoh — menjadi T. Ia panik bila berkasnya rusak, bagiannya tidak ada, atau isinya memuat
// field yang tidak dikenal T: berkas tersemat yang rusak adalah kesalahan pemrograman, dan
// harus gagal seketika saat paket dimuat, bukan menghasilkan data contoh yang diam-diam salah.
func Must[T any](raw []byte, name string) T {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		panic(fmt.Sprintf("sampledata: berkas data contoh rusak: %v", err))
	}
	part, ok := all[name]
	if !ok {
		panic(fmt.Sprintf("sampledata: data contoh %q tidak ada", name))
	}
	var value T
	decoder := json.NewDecoder(bytes.NewReader(part))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		panic(fmt.Sprintf("sampledata: data contoh %q tidak sesuai bentuknya: %v", name, err))
	}
	return value
}
