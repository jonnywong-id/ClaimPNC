package memory

import (
	_ "embed"

	"claim-pnc/internal/inboxrcl"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleLogin adalah login pengembangan lokal (`auth/provider/fake.go`) yang PUNYA identitas
// lama pada grup yang diizinkan, sehingga antrean contoh terlihat saat masuk.
const SampleLogin = "adminpnc"

// SampleLegacyID adalah identitas lama SampleLogin — padanan `TempOperator.City`.
//
// Karangan. Operator ID sungguhan yang tertanam di sistem lama tidak dipakai sebagai data
// yang dijalankan (`D-15`); `D-69` mengizinkannya hanya di dokumen.
const SampleLegacyID = "DOKTERRCL01"

// SampleLoginNoLegacy adalah login pengembangan lokal yang identitas lamanya HANYA tercatat
// pada grup yang tidak diizinkan — ia harus melihat keadaan "identitas lama tidak ditemukan".
const SampleLoginNoLegacy = "pictekniks"

// SampleOtherLegacyID adalah dokter RCL LAIN, dipakai membuktikan batas kewenangan bekerja.
const SampleOtherLegacyID = "DOKTERRCL02"

// SampleAccess adalah baris contoh `T_ACCESS_GROUP_PNC`.
//
// Satu orang punya satu baris per grup akses, sama seperti tabel aslinya. Setiap baris yang
// harus TERTOLAK diberi keterangan di tempatnya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// TERTOLAK — grupnya bukan salah satu dari tiga yang diizinkan.
// TERTOLAK — barisnya tidak aktif.
var SampleAccess = sampledata.Must[[]AccessRow](sampleJSON, "SampleAccess")

// SampleTasks adalah antrean contoh. Nomor polis dan nama tertanggungnya karangan (`D-69`).
//
// Apa yang sengaja dibuat pada contohnya:
//
//   - Satu baris milik dokter LAIN — bila penyaring A/D hilang, ia bocor ke layar.
//   - Satu baris yang penugasannya milik pemanggil tetapi dokter RCL-nya orang lain — hanya
//     penyaring D yang menolaknya.
//   - Satu baris tanpa Tanggal Masuk Inbox — hanya penyaring C yang menolaknya.
//   - Satu baris ber-`Resolved-Completed` (TERTOLAK) dan satu ber-`Resolved-Rejected` (TETAP
//     MUNCUL).
//   - Dua baris berwaktu daftar SAMA PERSIS, untuk pemutus seri `pyID` menurun.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Berwaktu daftar SAMA PERSIS dengan baris di bawahnya — …0405 harus mendahului …0404.
// HARUS TETAP MUNCUL — hanya `Resolved-Completed` yang dikecualikan.
// TIDAK BOLEH MUNCUL — tugasnya sudah tuntas (penyaring B).
// TIDAK BOLEH MUNCUL — belum dikirim analis ke dokter RCL (penyaring C).
// TIDAK BOLEH MUNCUL — penugasan milik pemanggil, tetapi dokter RCL-nya orang lain
// (penyaring D).
// TIDAK BOLEH MUNCUL bagi SampleLegacyID — milik dokter lain (penyaring A dan D).
var SampleTasks = sampledata.Must[[]inboxrcl.RCLTask](sampleJSON, "SampleTasks")

// NewSampleStore membentuk pembaca berisi data contoh — salinan, supaya dua portal tidak
// berbagi senarai yang sama (`R-20`).
func NewSampleStore() *Store {
	access := make([]AccessRow, len(SampleAccess))
	copy(access, SampleAccess)
	tasks := make([]inboxrcl.RCLTask, len(SampleTasks))
	copy(tasks, SampleTasks)
	return NewStore(access, tasks)
}
