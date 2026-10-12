// Package masterdominanfactor adalah inti modul Master Dominan Factor (`F-4`).
//
// # Apa yang dimodelkan di sini
//
// Dominan Factor adalah daftar acuan **faktor dominan penyebab kerugian** sebuah klaim.
// Ia dipakai di dua tempat, dan keduanya terbaca langsung dari export:
//
//   - `RDB List/GetDataDominanFactorList-SQL.xml` dan `-ListOS-SQL.xml` — faktor yang
//     dipilih untuk satu klaim, tersimpan di `POOLDATA.T_CLAIM_DOMINANFACTOR` dan
//     ditautkan ke master ini lewat `ID_DOMINANFACTOR`.
//   - `RDB List/GetDataOutstandingperCabangExport-SQL.xml` — laporan Outstanding per
//     Cabang, yang merangkai nama seluruh faktor satu klaim dengan `LISTAGG(m.name, ', ')`.
//
// Butir kedua adalah akibat yang tidak boleh dilupakan: **mengubah nama sebuah faktor di
// layar ini langsung mengubah isi laporan Outstanding per Cabang.** Namanya bukan label
// internal — ia dibaca manajemen.
//
// # Modelnya memang sesederhana ini
//
// `POOLDATA.M_DOMINAN_FACTOR` hanya punya DUA kolom, `ID` dan `NAME`, dan itu dibaca
// langsung dari `Database/PEGA_M_DOMINAN_FACTOR.prc:13` (INSERT) dan `:23` (UPDATE) —
// bukan disimpulkan dari layarnya. Tidak ada kolom status aktif, tidak ada kolom jejak,
// dan tidak ada penomoran lama seperti yang dimiliki Master Status Klaim.
//
// # Tidak ada Hapus, dan itu bukan kelalaian
//
// Procedure lama hanya mengenal `Insert` dan `Update` — cabang ketiganya tidak ada. Dan
// menghapus satu baris di sini akan membuat setiap klaim lama yang menyimpan `ID` itu di
// `T_CLAIM_DOMINANFACTOR` kehilangan artinya, sehingga laporan Outstanding menampilkan
// faktor yang hilang tanpa penjelasan. `ADR-0012` melarangnya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterdominanfactor

import (
	"context"
	"strings"
	"unicode/utf8"
)

// MaxNameLength membatasi panjang nama faktor dominan.
//
// Batas ini adalah KEBUTUHAN BARU, bukan peniruan: layar Pega tidak membatasi panjang
// sama sekali (`Section/DetailDominanFactor_Sec-Section.xml` — `pyRequired=false`, tanpa
// `pyMaxLength`), dan DDL tabelnya tidak ada di export sehingga lebar kolom `NAME` yang
// sebenarnya BELUM DIKETAHUI (`R-08`).
//
// Angkanya karena itu diambil dari keputusan yang sama pada dua master sebelumnya —
// masterstatus.MaxLabelLength dan mastertipesurveyors — dan disetujui Work Owner
// 2026-09-20. Bila DBA kelak melaporkan kolomnya lebih sempit, yang berubah cukup
// konstanta ini.
//
// Kenapa membatasi sama sekali padahal isian kosong dan ganda justru DITERIMA di modul
// ini: batas panjang bukan aturan bisnis melainkan penjaga terhadap penolakan basis data.
// Tanpa batas, isian yang melebihi lebar kolom akan sampai ke pengguna sebagai galat 500
// beserta nomor galat Oracle — bukan pesan yang dapat dibacanya.
const MaxNameLength = 100

// DominantFactor adalah satu baris master faktor dominan.
//
// Kedua field memetakan langsung ke kolom `ID` dan `NAME`, dan namanya mengikuti
// `CONTEXT.md` — bukan nama kolomnya. Pemetaannya ada di repo/sqlstore, satu tempat saja.
type DominantFactor struct {
	// ID dibuat sistem saat faktor ditambahkan dan TIDAK PERNAH berubah sesudahnya.
	// Layar Pega pun menandainya read-only, dan `T_CLAIM_DOMINANFACTOR` menyimpan
	// nilainya pada setiap klaim yang pernah memakai faktor ini.
	ID string

	// Name adalah kolom NAME, teks yang dibaca pengguna. Di layar Pega ia berlabel
	// "Keterangan".
	Name string
}

// Clean mengembalikan salinan dengan spasi tepi dibuang.
//
// Perapian ini nyata gunanya, bukan kerapian kosmetik. Dua alasannya:
//
//  1. Lebar kolom `ID` tidak diketahui (`R-08`), dan bila ia bertipe CHAR — seperti
//     `LSC_ID` pada Master Status Klaim yang terbukti CHAR(4) — nilainya kembali
//     membawa padding. `"1  "` dan `"1"` harus dikenali sebagai baris yang sama.
//  2. Nama yang membawa spasi tepi akan muncul apa adanya di `LISTAGG` laporan
//     Outstanding per Cabang, di antara koma pemisahnya.
func (f DominantFactor) Clean() DominantFactor {
	return DominantFactor{
		ID:   strings.TrimSpace(f.ID),
		Name: strings.TrimSpace(f.Name),
	}
}

// Repo adalah seam ke penyimpanan master faktor dominan SATU portal.
//
// # Satu portal, bukan satu aplikasi
//
// `M_DOMINAN_FACTOR` ada di basis data SETIAP entitas, dan faktor dominan milik satu
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
	// List mengembalikan seluruh faktor, terurut menurut ID secara numerik.
	List(ctx context.Context) ([]DominantFactor, error)

	// Get mengembalikan satu faktor. Mengembalikan ErrNotFound bila ID-nya tidak ada.
	Get(ctx context.Context, id string) (DominantFactor, error)

	// Insert menyimpan faktor baru dan mengembalikannya LENGKAP DENGAN ID yang dibuat
	// penyimpanan. ID tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, name string) (DominantFactor, error)

	// Update mengubah nama faktor yang sudah ada. ID tidak ikut berubah.
	// Mengembalikan ErrNotFound bila ID-nya tidak ada.
	Update(ctx context.Context, id, name string) (DominantFactor, error)
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

// CheckName mengumpulkan SELURUH pelanggaran aturan nama, bukan berhenti pada yang
// pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama
// menampilkan seluruh pesan validasi sekaligus, dan mengembalikannya satu per satu akan
// membuat pengguna menekan Simpan berkali-kali untuk menemukan kesalahan berikutnya
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// # Dua aturan yang SENGAJA TIDAK ADA di sini
//
// Modul ini berbeda dari Master Status Klaim dan Master Tipe Surveyors, dan perbedaannya
// diputuskan Work Owner 2026-09-20 setelah keduanya dijelaskan berikut akibatnya:
//
//   - **Nama kosong DITERIMA.** Layar Pega menerimanya (`pyRequired=false`), dan `P-5`
//     menetapkan perilaku dipertahankan lebih dulu. Akibat yang diterima sadar: faktor
//     bernama kosong akan muncul sebagai entri kosong di antara koma pada `LISTAGG`
//     laporan Outstanding per Cabang.
//   - **Nama ganda DITERIMA.** Tidak ada satu pun constraint keunikan di sistem lama, dan
//     tidak ada indeks unik yang ditambahkan modul ini. Akibat yang diterima sadar:
//     laporan yang sama dapat menampilkan dua entri yang terbaca identik.
//
// Karena itu modul ini TIDAK menuntut satu pun berkas migrasi — ia bekerja penuh di atas
// tabel yang sudah ada.
func CheckName(name string) []Violation {
	clean := strings.TrimSpace(name)
	var violations []Violation

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if length := utf8.RuneCountInString(clean); length > MaxNameLength {
		violations = append(violations, Violation{
			Field:   FieldName,
			Message: "Keterangan paling panjang " + itoa(MaxNameLength) + " karakter.",
		})
	}

	return violations
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
