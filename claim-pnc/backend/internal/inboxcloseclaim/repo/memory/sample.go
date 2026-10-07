package memory

import (
	_ "embed"

	"claim-pnc/internal/inboxcloseclaim"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleClaims adalah data contoh untuk menjalankan aplikasi tanpa Oracle.
//
// # SELURUHNYA KARANGAN
//
// Tidak ada satu pun nomor polis, nama tertanggung, atau nomor klaim nyata di sini.
// `D-69` menetapkan data nasabah TIDAK PERNAH ditulis ke berkas yang di-commit — dan
// berkas contoh adalah tempat yang paling mudah melanggarnya tanpa disadari, karena data
// nyata membuat layarnya terlihat lebih meyakinkan.
//
// # Yang sengaja diwakili
//
// Barisnya dipilih supaya setiap cabang penyaring punya sekurangnya satu contoh, dan
// supaya keadaan yang mudah keliru terlihat langsung di layar:
//
//	PNCN.26.0001  Non-MBU · sudah transfer · LUNAS
//	PNCN.26.0002  Non-MBU · belum transfer · belum lunas
//	PNCN.26.0003  Bonding · kelompok bisnis Bonding, untuk menguji arah `IN`
//	PNCN.26.0004  PA      · ditolak, bukan selesai — kolom status menampilkan "Reject"
//	PNCN.26.0005  Travel  · TANPA kode status, untuk memperlihatkan baris yang HILANG
//	              saat penyaring "BELUM LUNAS" dipakai
//	PNC-9001      klaim warisan tanpa tanggal tutup — Lama Waktu Klaim jatuh ke cadangan
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Kode statusnya sengaja KOSONG. Baris ini HILANG saat penyaring "BELUM LUNAS"
// dipakai — perilaku `<>` terhadap NULL di Oracle, yang ditiru apa adanya.
// Lihat unpaidMatches.
// Klaim warisan: bernomor `PNC-xxxx` (`D-22`) dan TANPA tanggal tutup.
// Kolom Lama Waktu Klaim jatuh ke PYRESOLVEDTIMESTAMP — cadangan pertama.
func SampleClaims() []inboxcloseclaim.ClosedClaim {
	return sampledata.Must[[]inboxcloseclaim.ClosedClaim](sampleJSON, "SampleClaims")
}
