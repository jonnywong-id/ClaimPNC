// Package random menyediakan pemilih acak sungguhan untuk mengisi seam komite.Randomizer.
//
// # Kenapa ia sebuah paket tersendiri
//
// Sama seperti platform/waktu yang memisahkan jam sistem dari aturan yang memakainya:
// pengacakan adalah sumber ketidakpastian, dan lapisan domain tidak boleh memuatnya.
// Domain hanya mendeklarasikan apa yang dibutuhkannya — "pilih satu dari sekian" — dan
// paket inilah yang mengisinya di produksi, sementara pengujian mengisinya dengan pemilih
// tetap.
//
// # Di mana ia dipakai
//
// Hanya pada kebijakan komite bermode satu-penyetuju, yaitu entitas Simasnet. Di sana
// beberapa orang berwenang pada tingkat yang sama, dan sistem lama menyebar beban di
// antara mereka dengan `ORDER BY degree, dbms_random.value` lalu mengambil satu baris.
//
// Lapisan platform: tidak mengimpor apa pun dari domain, adapter, maupun transport.
package random

import "math/rand/v2"

// System memilih secara acak memakai pembangkit bilangan acak pustaka standar.
//
// Ia TIDAK dipakai untuk apa pun yang berhubungan dengan keamanan — pemilihan penyetuju
// adalah penyebaran beban kerja, bukan rahasia — sehingga pembangkit biasa sudah memadai
// dan crypto/rand tidak diperlukan.
type System struct{}

// Pick mengembalikan indeks dalam rentang [0, count).
//
// Nilai count yang tidak masuk akal dijawab nol alih-alih panik: pemanggilnya sudah menjamin
// count > 0, dan menjatuhkan proses yang sedang melayani pengguna lain karena penjagaan
// ganda bukan pertukaran yang sepadan.
func (System) Pick(count int) int {
	if count <= 1 {
		return 0
	}
	return rand.IntN(count)
}
