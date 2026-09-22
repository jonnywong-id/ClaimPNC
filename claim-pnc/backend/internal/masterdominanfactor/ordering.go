package masterdominanfactor

import (
	"sort"
	"strconv"
	"strings"
)

// NumericID membaca sebuah ID sebagai bilangan.
//
// Nilai kedua false berarti ID-nya BUKAN bilangan. Keadaan itu tidak mustahil: kolomnya
// bertipe teks, dan lebar maupun constraint-nya tidak diketahui karena DDL tabel tidak
// ada di export (`R-08`). Procedure lama menjawabnya dengan cara terburuk — `to_number(ID)`
// pada `Database/PEGA_M_DOMINAN_FACTOR.prc:11` akan gagal dengan ORA-01722 dan MEMBATALKAN
// seluruh penambahan, tanpa pesan yang menyebut baris mana penyebabnya.
//
// Di sini baris seperti itu diperlakukan sebagai bukan-bilangan dan dilewati, bukan
// dibiarkan menjatuhkan layar. Itu perbedaan yang disengaja, dan arahnya menguntungkan:
// satu baris warisan yang cacat tidak lagi membuat penambahan faktor baru mustahil.
func NumericID(id string) (int64, bool) {
	clean := strings.TrimSpace(id)
	if clean == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(clean, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// SortByID mengurutkan daftar menurut ID secara NUMERIK, di tempat.
//
// # Kenapa tidak diurutkan di SQL
//
// ID dibentuk `max+1` tanpa nol di depan, sehingga isinya `1`, `2`, … `9`, `10`, `11`.
// Diurutkan sebagai TEKS, `10` mendahului `9` — dan daftar acuan yang nomornya melompat-
// lompat membuat pengguna ragu apakah ada baris yang hilang.
//
// Mengurutkannya di SQL menuntut `TO_NUMBER(ID)`, yang terikat dialek Oracle dan dilarang
// `docs/Steering/09-DATABASE-STRATEGY.md` §4. Jadi urutannya dikerjakan di sini, satu
// tempat, dipakai adapter SQL maupun adapter memori — supaya keduanya tidak dapat berbeda
// pendapat tentang urutan yang benar.
//
// # Ini PERBEDAAN yang disengaja dari sistem lama
//
// Kueri Pega tidak memuat ORDER BY sama sekali, sehingga urutan barisnya ditentukan basis
// data dan dapat berubah sewaktu-waktu. Urutan yang tetap adalah perbaikan, dan ia tidak
// mengubah satu pun nilai — hanya susunannya di layar.
//
// Baris yang ID-nya bukan bilangan ditempatkan di BELAKANG, terurut menurut teksnya.
// Menaruhnya di depan akan menyembunyikan baris normal di bawah keanehan warisan.
func SortByID(list []DominantFactor) {
	sort.SliceStable(list, func(i, j int) bool {
		left, leftNumber := NumericID(list[i].ID)
		right, rightNumber := NumericID(list[j].ID)

		switch {
		case leftNumber && rightNumber:
			return left < right
		case leftNumber != rightNumber:
			// Yang berupa bilangan selalu lebih dulu.
			return leftNumber
		default:
			return list[i].ID < list[j].ID
		}
	})
}

// NextID menghitung ID berikutnya dari daftar ID yang sudah ada.
//
// Ia meniru `select nvl(max(to_number(ID)),0)+1` pada
// `Database/PEGA_M_DOMINAN_FACTOR.prc:11` — termasuk hasilnya saat tabel kosong, yaitu
// `"1"`.
//
// Tanpa nol di depan, sama seperti aslinya. Menambahkan padding di sini akan membuat ID
// yang diterbitkan aplikasi ini berbentuk berbeda dari yang sudah tersimpan, dan
// `T_CLAIM_DOMINANFACTOR` menyimpan nilainya apa adanya — `"01"` dan `"1"` tidak akan
// saling cocok.
//
// Diekspor supaya perilaku ini dapat diuji, dan supaya adapter SQL dan adapter memori
// memakai perhitungan yang sama persis.
func NextID(existing []string) string {
	var highest int64
	for _, id := range existing {
		if n, isNumber := NumericID(id); isNumber && n > highest {
			highest = n
		}
	}
	return strconv.FormatInt(highest+1, 10)
}
