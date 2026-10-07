package memory

import (
	_ "embed"

	"claim-pnc/internal/detailpenyebab"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// NewSampleRepo membentuk penyimpanan berisi baris contoh.
//
// # Isinya DIKARANG, dan itu dinyatakan terang-terangan
//
// Tidak ada satu pun baris `POOLDATA.D_CAUSE_OF_LOSS` di repository ini — yang diterima
// dari DBA sejauh ini hanyalah `v_sts_claim.csv`, `emailkomite.csv`, dan berkas master
// menu. Baris di bawah karena itu **bukan salinan data produksi**; ia disusun agar
// layarnya dapat dijalankan dan diuji.
//
// Dua hal yang dijaga supaya contoh ini tetap berguna dan tidak menyesatkan:
//
//  1. **Bentuknya benar.** ID mengikuti aturan `PEGA_D_CAUSE_OF_LOSS.prc:19` — kode situs
//     disambung empat digit berpadding nol — sehingga cacat penyusunan ID akan terlihat di
//     sini juga.
//  2. **Isinya tidak menyerupai data nyata.** Kode situsnya `99`, yang tidak dipakai
//     entitas mana pun, dan tidak ada nomor polis, nama tertanggung, maupun nilai uang di
//     dalamnya (`D-69`).
//
// # Keragaman yang sengaja dimasukkan
//
// Baris contohnya tidak seragam, karena keseragaman menyembunyikan cacat:
//
//   - satu baris **tidak aktif** — menguji kolom Status Aktif benar-benar terbaca
//   - satu baris ber-Status Aktif **kosong** — meniru baris yang lahir sebelum isiannya
//     ada; lihat detailpenyebab.ActiveLabel
//   - satu baris **tanpa induk** — baris yatim, yang mungkin ada karena tidak ada foreign
//     key yang diketahui (`R-08`)
//   - satu baris **tanpa lini bisnis**, dan satu baris dengan **tiga** lini bisnis
//   - satu baris **tanpa Kode Kehilangan**
//
// Kelima keadaan itu sah di sistem lama, karena jalur simpannya tidak memeriksa apa pun —
// lihat detailpenyebab.Input.Check.
func NewSampleRepo() *Repo {
	repo := NewRepo()
	repo.seed(sampleRows(), sampleMaster(), sampleBusiness())
	return repo
}

// sampleMaster adalah Master Penyebab Kerugian contoh — induk dari baris di bawah.
//
// Tabel aslinya `POOLDATA.V_M_CAUSE_OF_LOSS`, dan modul ini hanya MEMBACA-nya. Modul yang
// mengelolanya — Master Penyebab Kerugian, MENU_ID 20 `CauseOfLossInbox` — belum dibangun.
func sampleMaster() []detailpenyebab.MasterOption {
	return sampledata.Must[[]detailpenyebab.MasterOption](sampleJSON, "sampleMaster")
}

// sampleBusiness adalah lini bisnis contoh.
//
// Nama-namanya mengikuti Group Panel yang tercatat di `02-BUSINESS-UNDERSTANDING.md` §1,
// supaya istilah yang muncul di layar adalah istilah yang memang dipakai petugas.
func sampleBusiness() []detailpenyebab.Business {
	return sampledata.Must[[]detailpenyebab.Business](sampleJSON, "sampleBusiness")
}

// sampleRows adalah baris Detail Penyebab Kerugian contoh.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Baris TANPA lini bisnis — sah, karena isiannya tidak wajib.
// Baris dengan TIGA lini bisnis.
// Baris TIDAK AKTIF.
// Baris ber-Status Aktif KOSONG — meniru baris yang lahir sebelum isiannya ada.
// Baris TANPA Kode Kehilangan.
// Baris YATIM — MasterID menunjuk induk yang tidak ada di sampleMaster.
// Sebutan induknya akan tampil kosong di layar, persis seperti di Pega.
func sampleRows() []detailpenyebab.CauseOfLossDetail {
	return sampledata.Must[[]detailpenyebab.CauseOfLossDetail](sampleJSON, "sampleRows")
}
