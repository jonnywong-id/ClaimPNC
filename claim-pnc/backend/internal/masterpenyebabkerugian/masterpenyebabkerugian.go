// Package masterpenyebabkerugian adalah inti modul Master Penyebab Kerugian (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Penyebab Kerugian adalah daftar acuan **sebab terjadinya kerugian** sebuah klaim. Ia
// bertingkat dua di sistem lama, dan modul ini hanya memegang tingkat PERTAMA:
//
//	POOLDATA.M_CAUSE_OF_LOSS   golongan  — layar `CauseOfLossInbox`, MENU_ID 20  ← modul ini
//	POOLDATA.D_CAUSE_OF_LOSS   rincian   — layar `DetailCauseOfLoss`, MENU_ID 38
//
// Keduanya butir menu terpisah di `POOLDATA.M_MENU_APLIKASI_PNC`, dengan harness
// terpisah, dan karena itu modul terpisah. Yang dirujuk Work Owner pada sesi ini adalah
// `Harness/CauseOfLossInbox-Harness.xml` — tingkat golongan.
//
// # Kenapa layar ini tidak sesepele tampilannya
//
// Layarnya hanya dua kolom, tetapi isinya dibaca jauh di luar layar ini:
// **19 rule Pega membaca `V_M_CAUSE_OF_LOSS`**, termasuk dasbor klaim per penyebab
// kerugian (`RDB List/BrowseCaseClaimPerCauseOfLoss-SQL.xml`) dan laporan XOL per bisnis
// (`GetDataXOLPerBusiness-SQL.xml`) yang mengelompokkan hasilnya dengan `GROUP BY
// COL_DESC`. Mengubah keterangan di sini **langsung mengubah pengelompokan laporan itu** —
// keterangannya bukan label internal.
//
// # Bentuk ID: kode situs ditambah tiga digit
//
// `Database/PEGA_M_CAUSE_OF_LOSS.prc:20` membentuknya
// `id_site || lpad(to_char(M_CAUSE_SEQ.nextval), 3, '0')`, dengan `id_site` dibaca dari
// `M_SITE_DATABASE WHERE CURRENT_SITE = '1'`. Polanya sama persis dengan Master Status
// Klaim, dan pembentukannya ditiru apa adanya supaya ID yang diterbitkan aplikasi ini
// melanjutkan deret yang sudah ada — bukan memulai deret baru yang bisa bertabrakan.
//
// # Tidak ada Hapus, dan itu bukan kelalaian
//
// Procedure lama hanya mengenal INSERT dan UPDATE — cabang ketiganya tidak ada — dan
// layar Pega pun tidak punya tombolnya. Menghapus satu golongan akan membuat setiap baris
// `D_CAUSE_OF_LOSS` yang menyimpan `M_COL_ID` itu kehilangan induknya, dan setiap klaim
// lama yang menyebutnya kehilangan artinya di 19 rule pembaca. `ADR-0012` melarangnya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpenyebabkerugian

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxDescriptionLength membatasi panjang keterangan penyebab kerugian.
//
// # Ini penjaga basis data, BUKAN aturan bisnis
//
// Work Owner menetapkan 2026-09-20 bahwa validasi layar ini mengikuti Pega apa adanya —
// dan Pega tidak memvalidasi satu pun isian di sini. Keterangan kosong dan keterangan
// ganda karena itu DITERIMA (lihat CheckDescription).
//
// Batas panjang tetap ada karena ia menjawab pertanyaan yang berbeda: bukan "apakah isian
// ini masuk akal secara bisnis", melainkan "apakah isian ini muat di kolomnya". Tanpa
// batas, isian yang melebihi lebar kolom sampai ke pengguna sebagai galat 500 beserta
// nomor galat Oracle — bukan pesan yang dapat dibacanya.
//
// Angkanya diambil dari keputusan yang sama pada tiga master sebelumnya —
// masterstatus.MaxLabelLength, mastertipesurveyors, dan masterdominanfactor. Lebar
// `COL_DESC` yang sebenarnya **BELUM DIKETAHUI**: DDL tabelnya tidak ada di export
// (`R-08`), dan tidak seperti `LSC_NOTE` yang sempat diverifikasi ke katalog pada
// 2026-09-17, kolom ini belum pernah dibaca siapa pun di tim ini.
//
// Bila DBA kelak melaporkan kolomnya lebih sempit, yang berubah cukup konstanta ini dan
// pasangannya di frontend.
const MaxDescriptionLength = 100

// CauseOfLoss adalah satu golongan penyebab kerugian.
//
// Ketiga field memetakan kolom `M_COL_ID`, `OLD_M_COL_ID`, dan `COL_DESC` pada
// `V_M_CAUSE_OF_LOSS`. Namanya mengikuti `CONTEXT.md` — bukan nama kolomnya; pemetaannya
// ada di repo/sqlstore, satu tempat saja.
type CauseOfLoss struct {
	// ID dibuat sistem saat golongan ditambahkan dan TIDAK PERNAH berubah sesudahnya.
	// Layar Pega pun menandainya read-only, dan `D_CAUSE_OF_LOSS.M_COL_ID` menyimpan
	// nilainya pada setiap rincian yang bernaung di bawahnya.
	ID string

	// LegacyID adalah kolom OLD_M_COL_ID — penomoran sebelum sistem ini dibangun.
	//
	// Ia JEJAK SEJARAH: tidak pernah diisi aplikasi dan tidak pernah diubah. Ia
	// ditampilkan karena ia satu-satunya yang menjelaskan data historis yang masih
	// memakai penomoran lama. Kosong pada golongan yang tidak pernah punya nomor lama.
	LegacyID string

	// Description adalah kolom COL_DESC, teks yang dibaca pengguna.
	//
	// Di layar Pega ia berlabel **"Deskripsi Kerugian"** — `pyCaption` yang terpasang pada
	// kolom grid maupun isian formnya (`Section/BrowseCauseOfLoss-Section.xml`). Di laporan
	// ia menjadi kolom pengelompokan.
	Description string
}

// Clean mengembalikan salinan dengan spasi tepi dibuang.
//
// Perapian ini nyata gunanya, bukan kerapian kosmetik. Dua alasannya:
//
//  1. `M_COL_ID` dideklarasikan `varchar2(4)` di procedure lama, tetapi tipe kolomnya
//     yang sebenarnya belum diverifikasi (`R-08`). Bila ia ternyata CHAR — seperti
//     `LSC_ID` pada Master Status Klaim yang terbukti CHAR(4) — nilainya kembali membawa
//     padding, dan `"1001 "` tidak akan dikenali sama dengan `"1001"`.
//  2. Keterangan yang membawa spasi tepi akan menghasilkan KELOMPOK TERSENDIRI pada
//     `GROUP BY COL_DESC` di laporan XOL per bisnis — dua baris yang terbaca identik
//     tampil sebagai dua kelompok berbeda.
func (c CauseOfLoss) Clean() CauseOfLoss {
	return CauseOfLoss{
		ID:          strings.TrimSpace(c.ID),
		LegacyID:    strings.TrimSpace(c.LegacyID),
		Description: strings.TrimSpace(c.Description),
	}
}

// Repo adalah seam ke penyimpanan master penyebab kerugian SATU portal.
//
// # Satu portal, bukan satu aplikasi
//
// `M_CAUSE_OF_LOSS` ada di basis data SETIAP entitas, dan penyebab kerugian milik satu
// badan hukum tidak boleh terbaca dari badan hukum lain (`D-75` butir 4, `ADR-0030`).
//
// Pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat penyaringan baris
// (`ADR-0030` Opsi 1). Tidak ada satu pun kueri di pengisinya yang menyaring berdasarkan
// entitas, dan memang tidak boleh ada.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Antarmukanya berbicara dalam istilah domain — List, Get, Insert,
// Update — bukan istilah SQL.
type Repo interface {
	// List mengembalikan seluruh golongan, terurut menurut ID.
	List(ctx context.Context) ([]CauseOfLoss, error)

	// Get mengembalikan satu golongan. Mengembalikan ErrNotFound bila ID-nya tidak ada.
	Get(ctx context.Context, id string) (CauseOfLoss, error)

	// Insert menyimpan golongan baru dan mengembalikannya LENGKAP DENGAN ID yang dibuat
	// penyimpanan. ID tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, description string) (CauseOfLoss, error)

	// Update mengubah keterangan golongan yang sudah ada. ID dan ID lama tidak ikut
	// berubah. Mengembalikan ErrNotFound bila ID-nya tidak ada.
	Update(ctx context.Context, id, description string) (CauseOfLoss, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti membaca atau menulis data
// satu badan hukum di basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// CheckDescription mengumpulkan SELURUH pelanggaran aturan keterangan, bukan berhenti
// pada yang pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama menampilkan
// seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan membuat
// pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// # Dua aturan yang SENGAJA TIDAK ADA di sini
//
// Keputusan Work Owner 2026-09-20: validasi layar ini **mengikuti Pega saja**. Yang
// diikuti adalah kenyataan bahwa `Section/BrowseCauseOfLoss-Section.xml` tidak memasang
// satu pun aturan — tidak ada `pyRequired`, tidak ada `pyMaxLength`, dan
// `Database/PEGA_M_CAUSE_OF_LOSS.prc` tidak memeriksa apa pun sebelum menyisipkan.
//
//   - **Deskripsi kosong DITERIMA.** Akibat yang diterima sadar: golongan tanpa deskripsi
//     muncul sebagai kelompok kosong pada `GROUP BY COL_DESC` di laporan.
//   - **Deskripsi ganda DITERIMA.** Tidak ada satu pun constraint keunikan di sistem
//     lama, dan modul ini TIDAK menambahkannya. Akibat yang diterima sadar: laporan yang
//     sama dapat menampilkan dua kelompok yang terbaca identik.
//
// Inilah yang membedakannya dari Master Status Klaim, yang justru menolak keduanya
// (keputusan Work Owner 2026-09-17). Kedua keputusan itu diambil untuk layar yang berbeda
// dan keduanya berlaku; perbedaannya dicatat supaya tidak dibaca sebagai ketidakkonsistenan.
//
// Karena tidak ada keunikan yang ditegakkan, migrasi modul ini **tidak membuat indeks
// unik apa pun** — berbeda dari migrasi 0002 yang membuatnya.
func CheckDescription(description string) []Violation {
	clean := strings.TrimSpace(description)
	var violations []Violation

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if length := utf8.RuneCountInString(clean); length > MaxDescriptionLength {
		violations = append(violations, Violation{
			Field:   FieldDescription,
			Message: "Deskripsi Kerugian paling panjang " + itoa(MaxDescriptionLength) + " karakter.",
		})
	}

	return violations
}

// SortByID mengurutkan daftar menurut ID secara teks, dari kecil ke besar.
//
// # Kenapa teks, padahal Master Dominan Factor justru menolak pengurutan teks
//
// Keduanya berbeda karena BENTUK ID-nya berbeda, dan itu terbaca dari procedure
// masing-masing:
//
//	M_DOMINAN_FACTOR   ID = max(ID)+1          → "9", "10"   panjangnya berubah-ubah
//	M_CAUSE_OF_LOSS    ID = situs || 3 digit   → "1009", "1010"  panjangnya TETAP
//
// Karena tiga digit terakhirnya selalu ber-nol di depan, pengurutan teks dan pengurutan
// numerik menghasilkan urutan yang SAMA — dan pengurutan teks tetap benar meski suatu
// saat ada ID yang bukan angka sama sekali.
//
// Satu perkecualian yang sengaja tidak ditangani: bila urutan menembus 999, ID menjadi
// lima karakter dan pengurutan teks menempatkan `"11000"` sebelum `"1999"`. Itu tidak
// ditambal di sini karena pada titik yang sama penyisipannya sendiri sudah DITOLAK basis
// data — lihat ThreeDigits di repo/sqlstore. Menambal urutannya akan menyembunyikan
// masalah yang justru harus terlihat.
//
// Pengurutan dikerjakan di Go, bukan di SQL, karena `ORDER BY` atas kolom CHAR berpadding
// berperilaku berbeda antar-basis-data — dan `D-20` menuntut satu set SQL yang berjalan di
// Oracle maupun PostgreSQL.
func SortByID(list []CauseOfLoss) {
	sort.SliceStable(list, func(a, b int) bool {
		return list[a].ID < list[b].ID
	})
}

// itoa mengubah bilangan kecil menjadi teks tanpa menarik strconv ke lapisan domain
// hanya untuk satu pesan.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
