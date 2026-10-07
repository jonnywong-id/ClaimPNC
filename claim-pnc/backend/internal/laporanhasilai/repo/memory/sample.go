package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleRows adalah baris contoh Laporan Hasil AI untuk pengembangan lokal dan pengujian.
//
// # Nilainya KARANGAN, dan itu disengaja
//
// Tidak satu pun nomor klaim, nama tertanggung, maupun nomor polis nyata boleh masuk ke
// berkas yang di-commit (`D-69`). Yang ditiru di sini adalah BENTUK dan KEANEHAN datanya,
// bukan isinya.
//
// # Apa yang sengaja diwakili
//
// Setiap baris ada untuk membuktikan satu hal, dan tanpa salah satunya ada penyaring atau
// aturan tampilan yang tidak akan pernah ketahuan salah:
//
//	AI setuju dengan komite          jalur paling umum
//	AI berbeda dengan komite         justru inilah yang dicari laporan ini
//	komite MENUNGGU                  membuktikan kolom "Menunggu" pada ringkasan
//	AI belum menilai (kosong)        membuktikan Total != jumlah baris
//	jenjang kedua pada klaim sama    membuktikan No Klaim dikosongkan
//	dua objek pada satu jenjang      membuktikan satu baris = satu penilaian AI
//	TANGGALKOMITE kosong             WAJIB HILANG dari hasil
//	case belum Resolved-Completed    WAJIB HILANG dari hasil
//	di luar rentang tanggal          WAJIB HILANG pada penyaring yang wajar
//
// Dua yang terakhir sebelum baris penutup adalah yang paling penting: keduanya harus
// TIDAK MUNCUL. Data contoh yang seluruhnya lolos penyaring tidak menguji penyaring apa
// pun.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Tanggal disusun sebagai tanggal kalender polos di UTC, sama seperti yang dihasilkan
// Filter.Clean. Tidak ada penambahan tujuh jam di mana pun.
// Satu klaim, satu jenjang, dua objek pertanggungan. Keduanya muncul sebagai dua
// baris; hanya baris pertama yang bernomor klaim karena keduanya jenjang "1" —
// dan itu memang yang terjadi di Pega: aturannya melihat JENJANG, bukan urutan
// baris. Akibatnya nomor klaim yang sama tergambar dua kali.
// Jenjang KEDUA pada klaim yang sama: nomor klaimnya dikosongkan.
// AI menerima, komite MENOLAK — selisih pendapat, yang justru dicari laporan ini.
// Komite belum memutuskan. Ia masuk kolom "Menunggu" dan TIDAK masuk Total.
// AI belum menilai sama sekali. Barisnya TETAP muncul — komite memutuskan tanpa
// penilaian AI, dan itu keadaan yang justru perlu terlihat.
// HILANG — TANGGALKOMITE kosong.
// HILANG — belum ada baris komite ber-STATUSCASE Resolved-Completed.
// Jauh di luar rentang yang wajar dipilih pengguna.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }
