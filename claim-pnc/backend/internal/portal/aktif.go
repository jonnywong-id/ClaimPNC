package portal

import "errors"

// ErrBelumSiap dikembalikan bila portalnya ada di daftar tetapi koneksi basis datanya
// belum hidup — kredensialnya belum diisi, atau basis datanya tidak dapat dihubungi
// saat aplikasi start.
//
// Ia DIBEDAKAN dari ErrTidakAda dengan sengaja: "entitas ini belum kami layani" dan
// "entitas ini tidak ada" menuntut tindak lanjut yang sama sekali berbeda dari pengguna
// maupun dari tim infrastruktur.
var ErrBelumSiap = errors.New("portal: koneksi basis data portal belum tersedia")

// PilihAktif menentukan portal yang melayani satu permintaan.
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
func PilihAktif(daftar []Portal, aliasSiap []string, alias string) (Portal, error) {
	if normalkan(alias) == "" {
		return Portal{}, ErrTidakDisebut
	}

	dipilih, err := Cari(daftar, alias)
	if err != nil {
		return Portal{}, err
	}

	for _, siap := range aliasSiap {
		if normalkan(siap) == normalkan(dipilih.Alias) {
			return dipilih, nil
		}
	}
	return Portal{}, ErrBelumSiap
}

// ErrTidakDisebut dikembalikan bila permintaan tidak menyebut portal sama sekali.
var ErrTidakDisebut = errors.New("portal: permintaan tidak menyebut portal")
