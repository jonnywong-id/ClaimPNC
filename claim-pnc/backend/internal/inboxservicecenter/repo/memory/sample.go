package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/inboxservicecenter"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleOwner adalah login PIC pemilik baris contoh.
//
// Keempat tab menyaring `PIC` menurut pengguna yang login, sehingga tanpa nilai ini tidak
// ada satu baris pun yang tampil saat pengembangan lokal. Pada basis data sungguhan, PIC-nya
// adalah petugas yang benar-benar ditunjuk menangani barisnya.
const SampleOwner = "PICSERVICECENTER"

// sampleOther adalah PIC lain, dipakai membuktikan penyaring kepemilikan bekerja.
const sampleOther = "PICLAIN"

// day membentuk tanggal UTC tanpa jam, supaya contohnya terbaca dan urutannya mudah
// diperiksa dengan mata.
func day(year int, month time.Month, date int) *time.Time {
	at := time.Date(year, month, date, 0, 0, 0, 0, time.UTC)
	return &at
}

// SampleClaims adalah klaim portal rekanan contoh untuk pengembangan lokal dan pengujian.
//
// # Kenapa seluruh isinya karangan
//
// Karena data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit (`D-69`). Nomor polis,
// nama nasabah, nomor klaim, dan IMEI di sini seluruhnya karangan yang bentuknya saja
// menyerupai aslinya. IMEI contoh sengaja TIDAK 15 digit yang sah, supaya ia tidak dapat
// tertukar dengan nomor perangkat sungguhan.
//
// # Apa yang sengaja dibuat pada contohnya
//
// Ia tidak sekadar "beberapa baris": tiap penyaring memperoleh baris yang membuktikannya
// bekerja.
//
//   - Keempat tab terisi, termasuk DUA baris pada tab Rejected — satu berkode `3` (REJECT)
//     dan satu berkode `2` (TLO) — supaya penyaing `IN ('2','3')` terbukti membawa keduanya.
//   - Satu baris ber-PIC ORANG LAIN, sehingga penyaring kepemilikan yang lupa dipasang akan
//     langsung terlihat sebagai baris yang bocor.
//   - Satu baris TANPA tanggal input, supaya urutan `INPUTDATE DESC` yang salah menaruh
//     NULL langsung terlihat.
//   - Satu baris yang hanya dapat ditemukan lewat NOPOLIS/QQNAME/IMEI dan satu yang ID-nya
//     khas, supaya cacat pencarian irisan (lihat Limitations) terbukti direplikasi — bukan
//     tidak sengaja diperbaiki.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// ------------------------------------------------ tab Registrasi SC (NULL)
// Baris milik PIC LAIN. Ia tidak boleh pernah tampil bagi SampleOwner.
// ------------------------------------------------ tab Waiting Approval ("0")
// ------------------------------------------------ tab Approved ("1")
// ------------------------------------------------ tab Rejected ("2" dan "3")
// TLO — kode "2". Ia masuk tab Rejected bersama kode "3".
// REJECT — kode "3".
func SampleClaims() []inboxservicecenter.ServiceClaim {
	return sampledata.Must[[]inboxservicecenter.ServiceClaim](sampleJSON, "SampleClaims")
}
