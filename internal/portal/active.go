package portal

import "errors"

// ErrNotReady dikembalikan bila portalnya ada di daftar tetapi koneksi basis datanya
// belum hidup — kredensialnya belum diisi, atau basis datanya tidak dapat dihubungi
// saat aplikasi start.
//
// Ia DIBEDAKAN dari ErrNotFound dengan sengaja: "entitas ini belum kami layani" dan
// "entitas ini tidak ada" menuntut tindak lanjut yang sama sekali berbeda dari pengguna
// maupun dari tim infrastruktur.
var ErrNotReady = errors.New("portal: koneksi basis data portal belum tersedia")

// SelectActive menentukan portal yang melayani satu permintaan.
//
// # Kenapa fungsi ini ada
//
// ADR-0030 menetapkan satu basis data per entitas dan perpindahan portal tanpa login
// ulang, sementara D-78 menetapkan satu identitas berlaku di keempat portal. Gabungan
// keduanya berarti satu sesi dapat menjangkau empat basis data milik empat badan hukum,
// dan yang tersisa sebagai pembatas hanyalah pemeriksaan pada setiap permintaan (R-20).
//
// Dua jalur kegagalan yang sudah dikenali R-20 ditutup di sini, dan keduanya ditutup
// dengan cara yang sama — MENOLAK:
//
//   - Alias kosong tidak jatuh ke portal utama. Permintaan tanpa portal ditolak
//     (TKT-F6-002). Jatuh ke koneksi default berarti menulis data satu badan hukum ke
//     basis data badan hukum lain tanpa satu pun pesan galat, dan layarnya akan tampak
//     normal — angkanya masuk akal, yang salah hanya milik siapa data itu.
//   - Alias yang tidak dikenal juga ditolak, bukan dinormalkan menjadi yang terdekat.
//
// Hasilnya melekat pada permintaan, bukan disimpan sebagai keadaan global di server:
// dua permintaan bersamaan dari pengguna yang sama akan saling menimpa bila portal
// aktif disimpan di satu tempat bersama, dan data kedua entitas tertukar.
//
// # Yang BELUM diperiksa di sini, dan harus disebut terang
//
// Fungsi ini menjawab "portal ini ada dan koneksinya hidup". Ia TIDAK menjawab
// "pengguna ini berwenang atas portal ini". Kewenangan portal per pengguna adalah
// TKT-F6-003, dan ia masih terhalang: D-78 menyisakan pertanyaan terbuka di mana data
// kewenangan itu disimpan — ia tidak dapat ikut tinggal di basis data masing-masing
// entitas, karena memeriksa hak atas portal B menuntut membaca basis data B sebelum
// pengguna terbukti berhak membacanya.
//
// Selama itu belum terjawab, setiap pengguna yang sudah masuk dapat memilih portal mana
// pun yang koneksinya hidup. Itu adalah R-20 yang masih terbuka, bukan yang sudah
// ditutup.
func SelectActive(list []Portal, readyAliases []string, alias string) (Portal, error) {
	if normalize(alias) == "" {
		return Portal{}, ErrNotStated
	}

	selected, err := Find(list, alias)
	if err != nil {
		return Portal{}, err
	}

	for _, ready := range readyAliases {
		if normalize(ready) == normalize(selected.Alias) {
			return selected, nil
		}
	}
	return Portal{}, ErrNotReady
}

// ErrNotStated dikembalikan bila permintaan tidak menyebut portal sama sekali.
var ErrNotStated = errors.New("portal: permintaan tidak menyebut portal")
