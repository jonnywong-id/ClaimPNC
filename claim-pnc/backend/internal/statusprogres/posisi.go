package statusprogres

import "strings"

// Posisi adalah satu tahap perjalanan klaim tempat status progres dicatat.
type Posisi struct {
	// Kode adalah nilai yang tersimpan di kolom STATUS.
	Kode string

	// Nama adalah label yang dibaca pengguna di dropdown.
	Nama string
}

// daftarPosisi adalah keempat posisi klaim, apa adanya seperti di sistem lama.
//
// # Dari mana angka-angka ini
//
// Di Pega, dropdown "Posisi" pada layar Master Status Progress 1
// (`Section/BrowseStatusProgress-Section.xml`, sel ber-`pyLabelFieldValue` = "Posisi")
// mengambil pilihannya dari page klipboard `TempPosition.pxResults` — dan page itu
// TIDAK dibaca dari tabel mana pun. Ia dirakit di dalam kode, pada step
// `Activity/ViewStatusProgress_act-Act.xml` yang berketerangan
// "INPUT POSISI KLAIM untuk dropdown", sebagai empat pasang nilai literal:
//
//	"REGISTER"  "002"
//	"SURVEY"    "004"
//	"KOMITE"    "006"
//	"AKSEPTASI" "007"
//
// Urutan di bawah mengikuti urutan perjalanan klaim, bukan urutan kodenya — itulah
// urutan yang berguna bagi pengguna, dan kodenya memang tidak berurutan rapat (003 dan
// 005 tidak dipakai di jalur ini).
//
// # Kenapa masih di kode, padahal D-15 melarang nilai bisnis di-hardcode
//
// Keputusan Work Owner 2026-09-17: keempatnya hidup sebagai daftar bernama di lapisan
// domain dan disajikan lewat API, BUKAN disalin ke frontend dan BUKAN tabel baru yang
// dikarang. Alasannya dua, dan keduanya disadari:
//
//   - Tidak ada tabel master posisi di sistem lama yang dapat dibaca. Membuatnya
//     menuntut permintaan perubahan skema tertulis, persetujuan Work Owner, dan
//     pelaksanaan DBA (`D-63`) — dan modul ini akan terhalang sampai itu selesai.
//   - Mengarang tabel beserta isinya berarti menebak, dan menebak dilarang.
//
// UTANG TEKNIS YANG DICATAT. Daftar ini seharusnya menjadi master data milik `F-4`
// sesuai `D-15`, dapat diubah tanpa merilis ulang aplikasi. Selama masih di sini,
// menambah posisi berarti mengubah kode. Satu-satunya hal yang sudah dibereskan
// sekarang adalah tempatnya: ia ada di SATU tempat, bukan tersebar — di sistem lama
// keempat pasang nilai itu ditulis ulang di setiap activity yang membutuhkannya.
var daftarPosisi = []Posisi{
	{Kode: "002", Nama: "REGISTER"},
	{Kode: "004", Nama: "SURVEY"},
	{Kode: "006", Nama: "KOMITE"},
	{Kode: "007", Nama: "AKSEPTASI"},
}

// DaftarPosisi mengembalikan seluruh posisi klaim, terurut sesuai perjalanan klaim.
//
// Salinan yang dikembalikan berdiri sendiri: pemanggil yang mengubahnya tidak dapat
// merusak daftar bagi pemanggil berikutnya.
func DaftarPosisi() []Posisi {
	salinan := make([]Posisi, len(daftarPosisi))
	copy(salinan, daftarPosisi)
	return salinan
}

// CariPosisi menemukan posisi berdasarkan kodenya. Nilai kedua false bila tidak dikenal.
//
// Pencocokannya mengabaikan spasi di ujung supaya kode yang datang dari basis data
// dengan tipe CHAR berlebar tetap (misalnya "002 ") tetap dikenali. Kolom STATUS belum
// diketahui tipenya — DDL-nya tidak ada di export (R-08) — dan CHAR memadatkan nilainya
// dengan spasi tanpa memberi tanda apa pun.
func CariPosisi(kode string) (Posisi, bool) {
	dicari := strings.ToUpper(strings.TrimSpace(kode))
	for _, p := range daftarPosisi {
		if strings.ToUpper(p.Kode) == dicari {
			return p, true
		}
	}
	return Posisi{}, false
}

// NamaPosisi mengembalikan nama posisi untuk sebuah kode.
//
// Kode yang tidak dikenal dikembalikan APA ADANYA, tidak diganti teks kosong maupun
// tanda tanya. Baris lama dapat memuat kode di luar keempat yang dikenali — kolom
// STATUS di sistem lama tidak punya constraint yang membatasinya — dan menyembunyikan
// nilai itu di layar akan membuat barisnya tampak kosong tanpa sebab. Pengguna perlu
// melihat apa yang benar-benar tersimpan.
func NamaPosisi(kode string) string {
	if p, dikenal := CariPosisi(kode); dikenal {
		return p.Nama
	}
	return strings.TrimSpace(kode)
}
