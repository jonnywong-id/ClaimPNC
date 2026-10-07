package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// Login contoh.
//
// # Kenapa BUKAN nama sungguhan
//
// `D-69` mengizinkan nama Operator ID ditulis di DOKUMEN supaya tiket dapat menunjuk hardcode
// mana yang dibuang. Izin itu untuk dokumen, bukan untuk data yang dijalankan — menuliskan
// login sungguhan sebagai data contoh akan memindahkan hardcode ke sistem baru lewat pintu
// belakang, tepat hal yang sedang dihapus (`D-15`).
const (
	// SampleLeaderLogin adalah adjuster yang MEMBAWAHI orang lain.
	SampleLeaderLogin = "ADJLEADER"

	// SampleMemberLogin adalah anggota di bawah leader di atas.
	SampleMemberLogin = "ADJMEMBER"

	// SampleInternalLogin adalah surveyor INTERNAL — `SURVEYTYPE = "1"`.
	//
	// Ia ada supaya keputusan Work Owner 2026-09-28 dapat dibuktikan: layar ini melayani
	// DUA populasi, dan yang membedakan keduanya adalah identitas yang masuk — bukan
	// penyaring yang dipilih pengguna.
	SampleInternalLogin = "SURVINTERN"

	// SampleOutsiderLogin adalah pengguna yang TIDAK terdaftar sebagai surveyor sama sekali.
	//
	// Tanpa baris seperti ini, ErrNotSurveyor tidak dapat dibuktikan berbeda dari antrean
	// kosong — dan itu persis perbedaan yang paling mudah hilang.
	SampleOutsiderLogin = "BUKANSURVEYOR"

	// SampleOtherSender adalah pengirim pesan dari PIHAK LAIN — petugas teknis ASM.
	//
	// Ia yang membedakan tab "belum dijawab" dari "belum dibalas ASM": keduanya berstatus
	// pesan sama, dan HANYA arah pengirimnya yang berbeda.
	SampleOtherSender = "PICTEKNIK1"
)

// Nama surveyor contoh.
//
// `SampleNearMissName` sengaja dibuat MEMUAT `SampleLeaderName` sebagai awalan. Ia yang
// membuktikan pembatas `|` pada cakupan benar-benar bekerja: tanpa pembatas, pemegang nama
// pendek akan melihat pekerjaan pemegang nama panjang.
const (
	SampleLeaderName   = "BUDI"
	SampleMemberName   = "SITI RAHAYU"
	SampleInternalName = "AGUS INTERNAL"
	SampleNearMissName = "BUDIONO SETIAWAN"
)

// at membentuk waktu UTC, supaya contoh terbaca dan urutannya mudah diperiksa dengan mata.
func at(year int, month time.Month, date, hour int) time.Time {
	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC)
}

// SampleSurveyors adalah isi `POOLDATA.MST_LOGIN_SURVEYOR` contoh.
//
// Perhatikan `SampleNearMissName`: ia terdaftar sebagai surveyor, tetapi BUKAN anggota
// leader mana pun. Itu yang membuat kebocoran cakupan — bila pembatas `|` hilang — benar-benar
// terlihat sebagai baris yang muncul di layar orang lain.
var SampleSurveyors = sampledata.Must[[]SurveyorRecord](sampleJSON, "SampleSurveyors")

// SampleRecords adalah antrean contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa nomor polis dan nama tertanggungnya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama tertanggung, dan nomor klaim di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya.
//
// # Kenapa hampir seluruh baris punya pesan
//
// Karena HANYA ketiga tab komunikasi yang dapat dihitung hari ini. Baris tanpa pesan tidak
// muncul di tab mana pun, sehingga contoh tanpa pesan tidak dapat membuktikan apa pun tentang
// cakupan, urutan, maupun pencarian.
//
// Satu baris SENGAJA dibiarkan tanpa pesan (`SRV-0006`) — ia yang membuktikan penyaring tab
// benar-benar menyaring, bukan meloloskan semuanya.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": setiap penyaring memperoleh baris yang membuatnya dapat
// dibuktikan bekerja.
//
//   - Satu baris milik `SampleNearMissName`, yang namanya MEMUAT nama leader sebagai awalan.
//     Bila pembatas cakupan hilang, baris itu bocor ke antrean leader.
//   - Satu baris milik anggota. Bila hierarki leader hilang, leader tidak melihatnya.
//   - Satu baris milik surveyor INTERNAL, `SURVEYTYPE = "1"`.
//   - Tiga baris berkomunikasi: dari pihak lain, dari diri sendiri belum dibalas, dan sudah
//     dibalas — ketiganya menempati tab yang berbeda.
//   - Satu baris ber-`CreatedAt` KOSONG, supaya "umur belum dapat dihitung" dapat dibedakan
//     dari nol hari. Ia ditaruh pada baris yang BENAR-BENAR tampil di sebuah tab; baris yang
//     tidak tampil di tab mana pun tidak pernah sampai ke layar dan tidak pernah teruji.
//   - Dua baris berwaktu input SAMA PERSIS, sehingga pemutus seri dapat diperiksa.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// `CreatedAt` sengaja dibiarkan kosong — umurnya tidak dapat dihitung, dan itu BERBEDA
// dari nol hari. Ia tetap punya pesan supaya benar-benar tampil di sebuah tab.
// Milik ANGGOTA. Leader harus melihatnya; anggota lain tidak.
// Milik orang yang namanya MEMUAT nama leader sebagai awalan.
//
// Bila pembatas cakupan hilang, baris ini muncul di antrean leader — dan tampak
// wajar, karena seluruh kolomnya terisi.
// Pesan dari DIRI SENDIRI, belum dibalas — tab "Not replied from ASM".
// Pesan dari diri sendiri yang SUDAH dibalas — tab "Replied from ASM".
// TANPA pesan sama sekali, dan itu bukan kelalaian.
//
// Ketiga tab yang dapat dihitung seluruhnya berbasis komunikasi, sehingga baris ini
// tidak muncul di tab mana pun. Ia yang membuktikan penyaring tab benar-benar
// menyaring — tanpa baris seperti ini, penyaring yang meloloskan segalanya akan lulus
// setiap uji di sini.
// Berwaktu input SAMA PERSIS dengan baris berikutnya, supaya pemutus seri teruji.
// Waktu input sama dengan baris sebelumnya; urutannya ditentukan pemutus seri.
// Milik surveyor INTERNAL — `SURVEYTYPE = "1"`.
var SampleRecords = sampledata.Must[[]Record](sampleJSON, "SampleRecords")

// SampleKPI adalah isi `POOLDATA.DETAIL_KPI_ADJUSTER` contoh.
//
// Dua baris per adjuster pada tahun yang sama, supaya RATA-RATA benar-benar dihitung — satu
// baris saja tidak dapat membedakan rata-rata dari penjumlahan.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Kategori BUKAN "FINAL". Ia harus TERTOLAK oleh ringkasan Final dan Kuartal, dan
// hanya muncul pada ringkasan Outstanding tanpa penyaring kategori.
// Milik adjuster DI LUAR cakupan. Bila penyaring cakupan hilang pada KPI, baris ini
// ikut terhitung — dan papan penilaian menjadi bocor.
var SampleKPI = sampledata.Must[[]KPIRecord](sampleJSON, "SampleKPI")

// NewSampleStore membentuk pembaca berisi seluruh data contoh.
func NewSampleStore() *Store {
	return NewStore(SampleRecords, SampleSurveyors, SampleKPI)
}
