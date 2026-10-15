package masterstatusprogres

import "strings"

// Position adalah satu tahap perjalanan klaim tempat status progres dicatat.
type Position struct {
	// Code adalah nilai yang tersimpan di kolom STATUS.
	Code string

	// Name adalah label yang dibaca pengguna di dropdown.
	//
	// Pada layar ini Name SELALU sama persis dengan Code — bukan kebetulan, melainkan
	// bentuk dropdown-nya di sistem lama; lihat keterangan `positions` di bawah.
	// Keduanya tetap dipisah supaya kontrak API tidak perlu berubah bila kelak daftarnya
	// pindah menjadi master data `F-4` dan labelnya dibedakan dari nilai simpanannya.
	Name string
}

// positions adalah kesembilan baris dropdown "Posisi", apa adanya seperti di layar lama
// — termasuk "All" yang memang muncul dua kali. Nilai yang berbeda ada delapan.
//
// # KOREKSI 2026-09-20 — daftar sebelumnya SALAH, pada dua hal sekaligus
//
// Versi pertama berkas ini memuat EMPAT pasang kode→label:
//
//	"002" REGISTER · "004" SURVEY · "006" KOMITE · "007" AKSEPTASI
//
// Keduanya keliru, dan Work Owner yang menemukannya dengan membandingkan layar baru
// terhadap layar Pega yang sedang berjalan:
//
//  1. JUMLAHNYA. Dropdown yang sebenarnya memuat DELAPAN pilihan, bukan empat.
//  2. NILAI YANG TERSIMPAN. Yang masuk ke kolom STATUS adalah TEKS-nya ("REGISTER"),
//     bukan kode angka ("002").
//
// # Bukti untuk butir 2
//
// Sel "Posisi" pada `Section/BrowseStatusProgress-Section.xml` mengikat dropdown-nya
// begini:
//
//	pySourceName = TempPosition.pxResults   (kelas Code-Pega-List)
//	pyValue      = .CaseID
//	pyPrompt     = .CaseID
//
// `pyValue` (yang disimpan) dan `pyPrompt` (yang ditampilkan) menunjuk properti yang
// SAMA. Jadi apa pun isi `.CaseID`, nilai simpanan dan labelnya memang satu dan sama —
// itulah sebabnya Code dan Name di atas selalu bernilai sama.
//
// Dan `.CaseID` berisi teks, bukan kode. `Activity/ViewStatusProgress_act-Act.xml`
// mengisinya dengan literal:
//
//	TempPositionClaim.pxResults(<APPEND>).CaseID := "AKSEPTASI"
//
// # Dari mana empat kode angka itu tadinya saya ambil
//
// Dari activity yang sama, tetapi dari KOLOM YANG BERBEDA — pasangan angka di sebelahnya
// mengisi properti lain, untuk dropdown layar lain. Saya membacanya sebagai pasangan
// kode→label untuk layar INI tanpa memeriksa `pyValue` sel-nya lebih dulu. Itu kesalahan
// pembacaan, bukan perbedaan data.
//
// # Kenapa daftarnya berasal dari tangkapan layar, bukan dari export
//
// Rule yang mengisi `TempPosition` untuk harness ini TIDAK ADA di export — sudah dicari
// ke seluruh Activity, Section, RDB List, dan Data Transform, dan kedelapan nilainya
// tidak muncul bersama di satu tempat mana pun. Ini persis kelas kekurangan yang dicatat
// `R-16` (±242 rule dirujuk tetapi tidak ikut diekspor).
//
// Karena artefaknya tidak ada, sumber yang sahih adalah layar Pega yang sedang berjalan,
// dan itulah yang diberikan Work Owner. Urutan di bawah mengikuti urutan pada layar itu
// apa adanya — bukan urutan perjalanan klaim, dan bukan urutan abjad.
//
// # DUA HAL YANG SEMPAT DIRAGUKAN, DAN JAWABANNYA
//
// Keduanya diajukan ke Work Owner, dan dijawab 2026-09-20: **"Sesuaikan saja dengan
// Pega."** Keduanya lalu diperiksa ulang ke export sebelum diterapkan.
//
//  1. "All" MUNCUL DUA KALI di layar lama — paling atas (terpilih) dan paling bawah, dan
//     di sini keduanya ditulis apa adanya. Sempat diduga yang pertama adalah baris
//     kosong bawaan Pega yang berkapsi "All", bukan data. Dugaan itu GUGUR:
//     `pyHasNoSelection = false` pada sel ini, yang berarti Pega TIDAK menyisipkan baris
//     kosong apa pun. Jadi dropdown-nya memang berisi sembilan baris, dengan "All"
//     benar-benar terulang di data.
//
//     Pengulangannya tidak mengubah apa yang tersimpan — kedua baris bernilai sama — dan
//     ia direplikasi supaya layar baru berisi baris yang persis sama dengan layar lama
//     (`D-13`). Menghapus baris kedua adalah perubahan satu baris di sini bila Work Owner
//     kelak menghendakinya.
//
//  2. "PROCUREMENT" BENAR, tanpa "/SUPPLIER". Kekhawatirannya berasal dari
//     `Section/ApprovalProgressKlaim-Section.xml` yang membandingkan sebuah nilai dengan
//     `'PROCUREMENT/SUPPLIER'`. Pemeriksaan ke seluruh export menemukan teks itu hanya
//     muncul pada empat berkas, dan SELURUHNYA membandingkan properti lain —
//     `.AlasanTerlambat` dan `.SurveyorAddrress` — bukan kolom STATUS milik master ini.
//     Tidak ada satu pun tempat yang membandingkan posisi klaim dengan "/SUPPLIER".
//
// # Kenapa masih di kode, padahal D-15 melarang nilai bisnis di-hardcode
//
// Keputusan Work Owner 2026-09-17, dan alasannya tidak berubah oleh koreksi ini: tidak
// ada tabel master posisi di sistem lama yang dapat dibaca, dan membuatnya menuntut
// permintaan perubahan skema tertulis, persetujuan Work Owner, serta pelaksanaan DBA
// (`D-63`) — modul ini akan terhalang sampai itu selesai. Mengarang tabel beserta isinya
// berarti menebak, dan menebak dilarang.
//
// UTANG TEKNIS YANG DICATAT. Daftar ini seharusnya menjadi master data milik `F-4`,
// dapat diubah tanpa merilis ulang aplikasi. Selama masih di sini, menambah posisi
// berarti mengubah kode. Yang sudah dibereskan hanyalah tempatnya: ia ada di SATU
// tempat, bukan tersebar seperti di sistem lama.
var positions = []Position{
	{Code: "All", Name: "All"},
	{Code: "REGISTER", Name: "REGISTER"},
	{Code: "KOMITE", Name: "KOMITE"},
	{Code: "SURVEY", Name: "SURVEY"},
	{Code: "AKSEPTASI", Name: "AKSEPTASI"},
	{Code: "OUTSTANDING", Name: "OUTSTANDING"},
	{Code: "BENGKEL", Name: "BENGKEL"},
	{Code: "PROCUREMENT", Name: "PROCUREMENT"},
	// Baris kesembilan, dan ia memang pengulangan baris pertama. Lihat butir 1 di atas:
	// `pyHasNoSelection = false` membuktikan Pega tidak menyisipkan baris kosong, jadi
	// pengulangan ini ada di datanya sendiri.
	{Code: "All", Name: "All"},
}

// ListPositions mengembalikan seluruh posisi klaim, terurut seperti di layar lama.
//
// Salinan yang dikembalikan berdiri sendiri: pemanggil yang mengubahnya tidak dapat
// merusak daftar bagi pemanggil berikutnya.
func ListPositions() []Position {
	copied := make([]Position, len(positions))
	copy(copied, positions)
	return copied
}

// FindPosition menemukan posisi berdasarkan nilai simpanannya. Nilai kedua false bila
// tidak dikenal.
//
// Pencocokannya mengabaikan beda huruf besar-kecil dan spasi di ujung. Spasi diabaikan
// karena kolom STATUS belum diketahui tipenya — DDL-nya tidak ada di export (`R-08`) —
// dan CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun.
// Beda huruf diabaikan karena nilai yang tersimpan adalah teks bebas: baris lama dapat
// memuat "register" sementara daftar ini menulis "REGISTER".
//
// Yang dikembalikan adalah baris dari daftar di atas, yaitu EJAAN BAKUNYA — bukan ejaan
// yang dikirim pemanggil. Itu yang dipakai Input.Clean untuk membakukan isian sebelum
// disimpan, sehingga tabel tidak terisi "All", "ALL", dan "all" sekaligus.
func FindPosition(code string) (Position, bool) {
	wanted := strings.ToUpper(strings.TrimSpace(code))
	for _, p := range positions {
		if strings.ToUpper(p.Code) == wanted {
			return p, true
		}
	}
	return Position{}, false
}

// PositionName mengembalikan label untuk sebuah nilai simpanan.
//
// Nilai yang tidak dikenal dikembalikan APA ADANYA, tidak diganti teks kosong maupun
// tanda tanya. Baris lama dapat memuat nilai di luar kedelapan yang dikenali — kolom
// STATUS di sistem lama tidak punya constraint yang membatasinya, dan daftar di atas
// sendiri berasal dari tangkapan layar, bukan dari sumber yang lengkap. Menyembunyikan
// nilai itu di layar akan membuat barisnya tampak kosong tanpa sebab; pengguna perlu
// melihat apa yang benar-benar tersimpan.
func PositionName(code string) string {
	if p, known := FindPosition(code); known {
		return p.Name
	}
	return strings.TrimSpace(code)
}
