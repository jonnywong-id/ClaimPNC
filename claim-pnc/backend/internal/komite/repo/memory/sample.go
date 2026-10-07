package memory

import (
	_ "embed"

	"claim-pnc/internal/komite"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleThresholds mengembalikan isi master ambang komite sebagaimana yang berlaku hari
// ini.
//
// # Dari mana angkanya
//
// Diturunkan baris per baris dari `Database/emailkomite.csv` — berkas yang diserahkan
// Work Owner. Ia BUKAN data karangan: setiap batas, jenjang, dan penanda status di bawah
// ada di berkas itu, dan itulah yang membuat ketujuh kasus jumlah penyetuju pada
// `docs/ticketing/B-7-Komite-Persetujuan-Klaim/spec.md` dapat diuji tanpa basis data
// sama sekali.
//
// # Isinya 29 baris, bukan 30 — dan kenapa itu sempat salah dibaca
//
// Berkasnya 31 baris fisik: satu judul dan 30 baris isi. Tetapi salah satu record
// membentang di DUA baris fisik, karena OPERATOR_ID pada baris ID 4 diakhiri BARIS BARU
// di dalam tanda kutip. Dibaca per baris, record itu terbelah dan menghasilkan dua baris
// palsu — sempat memunculkan "TYPE_BUSINESS" bernilai `NJOMANSUDARTHA` dan `1`, yang
// keduanya sebenarnya pecahan kolom lain.
//
// Dibaca dengan pengurai CSV yang benar, isinya **29 record**, dan **15 di antaranya
// jenjang persetujuan aktif** — tepat sebanyak yang ada di bawah.
//
// # Yang dibawa ke sini: seluruh jenjang, sebagian yang bukan
//
// Kelima belas baris jenjang persetujuan dibawa SELURUHNYA — itu yang menentukan hasil
// perhitungan. Dari 14 baris sisanya, hanya EMPAT yang dibawa sebagai contoh: baris tidak
// aktif dan baris pemberitahuan registrasi, masing-masing dua, supaya pengujian dapat
// membuktikan keduanya memang tidak pernah ikut menyetujui. Sisanya tidak menambah satu
// pun kasus uji yang belum tercakup.
//
// # Yang sengaja TIDAK dibawa: alamat surel
//
// Master aslinya memuat kolom EMAIL dan CC. Keduanya tidak dibaca modul ini dan tidak
// disalin ke sini, karena dua alasan yang saling menguatkan:
//
//   - Modul ini menghitung SIAPA yang menyetujui, bukan ke mana pemberitahuan dikirim;
//     yang terakhir adalah `S-3`.
//   - `D-69` mewajibkan alamat surel disamarkan di seluruh artefak yang di-commit, dan
//     `D-67` menetapkan alamat pribadi pada master lama — sekurang-kurangnya enam akun
//     Gmail di jalur produksi — TIDAK dibawa ke sistem baru sama sekali.
//
// Menyalin kolomnya ke berkas yang akan di-commit hanya akan memindahkan masalah yang
// sudah diputuskan untuk dihapus.
//
// Nama orang dan Operator ID ditulis lengkap, dan itu memang dibolehkan `D-69`: tanpa
// keduanya, hasil penjenjangan tidak dapat ditelusuri kembali ke baris masternya.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// ── Non-MBU, pita 1 — klaim sampai Rp 100.000.000 ────────────────────────────
//
// Perhatikan kedua baris ini ber-DEGREE SAMA (1). Itu bukan salah salin: di
// master pun demikian, dan itulah sebab penanda AmbiguousOrder ada.
// ── Non-MBU, pita 2 — klaim di atas Rp 100.000.000 ───────────────────────────
// OPERATOR_ID baris ini di berkas asli diakhiri BARIS BARU —
// "MARTENPETRUSLALAMENTIK_1\n". Di sini sudah bersih; perapiannya dikerjakan
// Threshold.Normalized supaya data dari Oracle pun ikut terlindungi.
// ── Baris Non-MBU yang BUKAN jenjang persetujuan ─────────────────────────────
//
// Inilah jawaban atas pertanyaan terbuka "baris DEGREE=0 maksudnya apa?" pada
// TKT-B07-001, dan jawabannya terbaca dari datanya sendiri: ia aktif, tetapi
// STS_ADJ-nya KOSONG dan STS_REG-nya menyala. Ia penerima pemberitahuan saat
// registrasi, bukan jenjang — dan penyaring STS_ADJ sudah mengeluarkannya tanpa
// perlu aturan khusus tentang DEGREE.
//
// Ia sengaja dibawa ke berkas contoh supaya pengujian membuktikan baris seperti
// ini benar-benar tidak pernah terhitung sebagai penyetuju.
// ── Non-MBU AB dan C — masing-masing satu jenjang ────────────────────────────
// ── Baris tidak aktif — dibawa supaya pengujian membuktikan ia diabaikan ─────
// ── Travel — tiga jenjang, TANPA pita ────────────────────────────────────────
//
// Seluruh jenjang aktifnya ber-TYPE_KOMITE "1", termasuk yang di atas
// Rp 100.000.000. Bila pita diberlakukan di sini, klaim Travel Rp 150.000.000
// akan kehilangan SELURUH penyetujunya.
// ── Personal Accident — empat jenjang, TANPA pita ────────────────────────────
//
// TYPE_KOMITE di sini berselang-seling 2 · 1 · 1 · 2 menaiki tangga. Itulah
// bukti paling jelas bahwa kolom tersebut BUKAN pita nilai pada lini ini,
// melainkan pembeda PA reguler dari PA TKI (`D-70`).
// ── Bonding — satu jenjang, batasnya 0/0 ─────────────────────────────────────
//
// Kueri Bonding di sistem lama (`EmailKomiteBerjenjangBonding_sql`) memang TIDAK
// menyaring LIMIT sama sekali, dan isi masternya sejalan dengan itu.
func SampleThresholds() []komite.Threshold {
	return sampledata.Must[[]komite.Threshold](sampleJSON, "SampleThresholds")
}
