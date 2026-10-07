package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// Data contoh modul Inbox Salvage.
//
// # Aturan yang mengikat berkas ini
//
// TIDAK ADA data nasabah nyata di sini. Nomor klaim, nama barang, dan nama PIC seluruhnya
// karangan, dan tidak satu pun disalin dari produksi maupun dari export (`D-69`).
//
// # Apa yang ia harus buktikan
//
// Bukan "layarnya terisi" melainkan setiap aturan yang dapat salah:
//
//   - Ketiga belas daftar punya isi, sehingga tab yang kosong berarti penyaringnya salah,
//     bukan datanya habis.
//   - Daftar Salvage Outstanding dan pencacah "Outstanding" menghitung populasi yang
//     BERBEDA — selisih yang direplikasi (`P-5`) dan harus terlihat di layar.
//   - Daftar Request Balai Lelang berisi baris milik dua PIC yang berbeda, sehingga
//     penyaring "hanya milik saya" dapat gagal dengan terlihat.
//   - Satu baris ber-`NILAIAKSEP` NOL, supaya "Belum Terjual" tidak dapat lolos hanya
//     karena kolomnya kosong.
//   - Satu baris ber-catatan request terisi dan satu yang kosong, supaya kedua nilai
//     "Tipe Pengajuan" tergambar.

// SampleCallerPIC adalah PIC yang memiliki baris pada daftar Request Balai Lelang.
//
// Ia dipakai uji untuk memastikan penyaring "hanya milik saya" benar-benar menyaring:
// daftar itu berisi baris milik PIC ini DAN milik PIC lain, sehingga penyaring yang lupa
// dipasang akan terlihat sebagai baris tambahan, bukan sebagai daftar kosong.
// SampleClaimWithoutSalvage adalah klaim contoh yang BELUM punya pengajuan salvage.
//
// Diberi nama supaya uji dapat menunjuknya tanpa mengandalkan kebetulan — bila kelak ada
// yang menambahkan pengajuan untuk klaim ini, uji yang memakainya akan gagal dan menyebut
// alasannya, alih-alih diam-diam menguji hal lain.
const SampleClaimWithoutSalvage = "PNC-2049"

const SampleCallerPIC = "SITIRAHAYU"

// sampleOtherPIC memiliki baris pada daftar yang sama, dan tidak boleh terlihat oleh
// SampleCallerPIC.
const sampleOtherPIC = "BUDISANTOSO"

// SampleToday adalah tanggal yang dianggap "hari ini" oleh data contoh.
//
// Ia TETAP, bukan jam mesin, supaya kolom "Aging" menghasilkan angka yang sama pada setiap
// pembukaan — kalau tidak, satu-satunya uji yang dapat ditulis adalah uji yang tidak
// memeriksa angkanya.
const SampleToday = "2026-09-25"

// NewSampleStore membentuk penyimpanan berisi data contoh.
func NewSampleStore() *Store {
	store := NewStore()
	store.SetNow(func() string { return SampleToday })
	store.Seed(sampleClaims(), sampleSalvages())
	return store
}

// sampleClaims memasok keluarga A dan B.
//
// Sebaran `STSSALVAGE`-nya disengaja:
//
//	kosong  2 baris  -> daftar Salvage Outstanding
//	"3"     2 baris  -> daftar Ekonomis, DAN pencacah "Outstanding"
//	"5"     1 baris  -> daftar TBA, DAN pencacah "Outstanding"
//	"4"     1 baris  -> daftar Tidak Ekonomis
//	"1"     1 baris  -> daftar Tidak Ada Salvage
//
// Dengan sebaran itu pencacah "Outstanding" menyebut 3 sementara daftarnya menampilkan 2 —
// selisih yang memang ada di Pega, dan yang sekarang dapat dilihat alih-alih dipercaya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Klaim yang BELUM punya satu pun pengajuan salvage.
//
// Ia ada supaya panel rincian yang dibuka dari daftar berbasis klaim punya
// kasus nyata untuk keadaan yang paling lazim di daftar Salvage Outstanding:
// klaim yang salvage-nya belum diajukan sama sekali.
//
// Tanpa baris ini, seluruh uji panel akan berjalan atas klaim yang kebetulan
// selalu punya pengajuan — dan keadaan yang justru paling sering dilihat
// pengguna tidak pernah teruji.
// Sudah selesai — TIDAK boleh muncul di daftar Salvage Outstanding, karena
// hanya daftar itu yang menyaring status kerja.
// Klaim yang sama juga punya nilai salvage di adjustment, sehingga ia muncul
// di daftar Ekonomis DAN Salvage Buyback. Itu memang mungkin: kedua
// penyaringnya tidak saling meniadakan.
func sampleClaims() []Claim { return sampledata.Must[[]Claim](sampleJSON, "sampleClaims") }

// sampleSalvages memasok keluarga C.
//
// Sebaran `STSTRANSFER`-nya menyentuh setiap daftar keluarga C:
//
//	"1"  Balai Lelang           2 baris, satu terjual dan satu belum
//	"3"  Checker / Diterima     2 baris, satu ber-request dan satu tidak
//	"4"  Rejected Checker       1 baris
//	"5"  Salvage Ditolak        1 baris  <- di Pega daftarnya tidak pernah dijalankan
//	"7"  Request Balai Lelang   2 baris, PIC berbeda
//	"6"                         1 baris  <- hanya terhitung pencacah "Histori Salvage"
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Sudah laku — kolom "Status Lelang" harus berbunyi "Terjual".
// NOL, bukan kosong. "Belum Terjual" di sini membuktikan aturannya memeriksa
// nilainya, bukan sekadar keberadaannya.
// Tanpa catatan request -> "Tipe Pengajuan" berbunyi "Pengajuan Baru".
// Dengan catatan request -> "Request Balai Lelang".
// PIC BERBEDA — baris ini tidak boleh terlihat oleh SampleCallerPIC pada
// daftar Request Balai Lelang maupun pada pencacahnya.
// `STSTRANSFER` 6 tidak punya daftar sendiri. Ia ADA supaya pencacah "Histori
// Salvage" — yang menghitung 1 atau 6 — menyebut angka yang berbeda dari
// jumlah baris daftarnya, persis seperti di Pega.
func sampleSalvages() []Salvage { return sampledata.Must[[]Salvage](sampleJSON, "sampleSalvages") }
