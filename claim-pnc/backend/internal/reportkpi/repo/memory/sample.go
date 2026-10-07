package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
	"claim-pnc/internal/reportkpi"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// NewSampleStore membentuk penyimpanan berisi contoh.
//
// # Seluruh isinya KARANGAN
//
// Nama adjuster, nomor case, dan nilainya tidak diambil dari produksi. `D-69` melarang
// data nasabah masuk ke berkas yang di-commit, dan nama adjuster adalah nama orang atau
// perusahaan yang nyata di produksi.
//
// # Yang dicakup contoh ini, dan kenapa masing-masing ada
//
// Ia bukan sekadar "beberapa baris supaya layar tidak kosong". Setiap baris mewakili satu
// keadaan yang layar harus tangani benar, termasuk empat yang paling mudah terlewat:
//
//	dua tipe pada SATU adjuster    membuktikan tipe ALL menghasilkan dua baris ringkasan,
//	                               bukan satu baris gabungan
//	komponen yang KOSONG           membuktikan sel kosong tidak menjadi 0
//	komponen BUKAN ANGKA           membuktikan satu nilai rusak tidak menyeret rata-rata
//	                               komponen itu, dan tidak membuang barisnya
//	tanggal di TEPI rentang        membuktikan batas atas ikut terhitung — cacat
//	                               setengah-terbuka yang paling sering lolos uji
func NewSampleStore() *Store {
	return NewStoreWith(SampleRows()).
		WithAdminRows(SampleAdminRows()).
		WithPICTeknik(SamplePICTeknik())
}

// SampleRows adalah baris contoh, terbuka supaya uji dapat memakainya apa adanya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// ── Adjuster pertama: dua tipe sekaligus ────────────────────────────────
//
// Ia yang membuktikan perilaku tipe ALL. Pada OUTSTANDING nilainya lebih
// rendah daripada pada FINAL, sehingga kedua barisnya dapat dibedakan sekilas
// dan baris yang tertukar akan terlihat.
// ── Adjuster kedua: komponen yang belum dinilai ─────────────────────────
//
// Tiga komponen sengaja TIDAK ADA di peta, yang berarti kolomnya NULL. Layar
// harus menggambarnya sebagai tanda hubung, bukan sebagai 0 — dan
// rata-ratanya harus KOSONG pula, bukan nol.
// INTERIM REPORT, UPDATE PROGRESS, dan TANGGAPAN KOMUNIKASI
// sengaja tidak ada.
// ── Adjuster ketiga: satu nilai BUKAN ANGKA ─────────────────────────────
//
// Kolomnya bertipe teks di Oracle, sehingga isi seperti ini NYATA-nyata
// mungkin. Di sini ia membuktikan dua hal sekaligus: komponen itu tidak ikut
// rata-rata, dan kedelapan komponen lain pada baris yang sama TETAP ikut.
//
// Catatan yang perlu diingat saat membaca hasil uji: terhadap Oracle,
// `TO_NUMBER` pada baris seperti ini MENGGAGALKAN laporan dengan `ORA-01722`
// — lihat alasannya di reportkpi.sql. Pengisi memori sengaja lebih pemaaf
// supaya layar dapat diuji; selisih itu ada di penanganan data rusak, bukan
// pada data yang bersih.
// ── Tepi rentang ────────────────────────────────────────────────────────
//
// Dua baris pada 1 Maret dan 31 Maret. Rentang 2026-03-01..2026-03-31 WAJIB
// memuat keduanya; penyaring yang keliru menulis `< sampai` alih-alih
// `< sampai + 1 hari` akan membuang yang kedua tanpa satu pun galat.
// ── DI LUAR rentang contoh ──────────────────────────────────────────────
//
// Sehari sebelum dan sehari sesudah Maret. Keduanya ada supaya uji dapat
// membuktikan penyaring periode BENAR-BENAR menyaring — penyaring yang tidak
// bekerja sama sekali tidak dapat dibedakan dari penyaring yang bekerja, bila
// seluruh data kebetulan berada di dalam rentangnya.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }

// fullScores menyusun peta kesembilan komponen dalam urutan kolom layar.
//
// Parameternya sengaja tidak bernama satu per satu: urutannya SAMA dengan
// reportkpi.ComponentCodes(), dan menuliskan sembilan nama parameter di sini hanya
// menambah tempat kedua yang harus dijaga kesamaannya.
func fullScores(values ...string) map[string]string {
	codes := reportkpi.ComponentCodes()

	result := make(map[string]string, len(codes))
	for i, code := range codes {
		if i >= len(values) {
			break
		}
		result[code] = values[i]
	}
	return result
}
