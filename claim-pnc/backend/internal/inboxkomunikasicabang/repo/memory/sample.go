package memory

import (
	_ "embed"

	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// SampleRows adalah baris contoh untuk pengembangan lokal dan pengujian.
//
// # Janji yang dipegang daftar ini
//
// SETIAP penyaring punya baris yang cocok MAUPUN yang tidak. Baris yang hanya cocok
// membuktikan penyaringnya meloloskan yang benar; baris yang tidak cocok membuktikan ia
// menolak yang salah — dan penyaring yang hilang hanya ketahuan lewat yang kedua.
//
// Penyaring yang punya saksi penolak, satu per satu:
//
//	CASEID = 'CABANG'         -> KOM-0007 ber-CABANG SELESAI
//	SENDER IS NOT NULL        -> KOM-0008 tanpa pengirim
//	MESSAGE IS NOT NULL       -> KOM-0009 tanpa pesan
//	batas cabang              -> KOM-0010 milik cabang 1003
//	REPLYMESSAGE (tab)        -> tersebar di seluruh daftar
//	REPLYFROM (pencacah)      -> KOM-0006, dibalas TANPA penjawab tercatat
//
// # Tidak ada data nasabah di sini
//
// Nama tertanggung, nomor polis, dan nomor klaim TIDAK ada di tabel ini sama sekali —
// isinya percakapan, bukan klaim. Nama operator yang dipakai adalah login contoh yang sama
// dengan provider identitas tiruan, bukan nama orang sungguhan (`D-69`).
//
// # Kode cabang yang dipakai
//
//	1        kantor pusat  (inboxkomunikasicabang.HeadOfficeCode)
//	1001     cabang petugas `adminpnc`
//	1002     cabang petugas `pictekniks`
//	1003     cabang yang TIDAK dimiliki satu pun login contoh — saksi penolak
//
// Ketiganya sejalan dengan SampleBranchOfLogin di branch.go, dan itu bukan kebetulan:
// tanpa kesejajaran itu, tidak satu pun login contoh dapat melihat satu pun baris.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// ── Terlihat oleh KANTOR PUSAT (kode "1") ──────────────────────────────────
// Nama pengirim TERISI, dan justru itu yang diuji: ia tidak boleh muncul di
// layar. Kolom "Pengirim(Dari)" menampilkan kode asal, bukan nama ini.
// Lampiran yang BELUM diunggah — tanggalnya kosong.
//
// Ia saksi bahwa Attachment.Uploaded benar-benar membedakan keduanya.
// Tanpa baris ini, penanda yang selalu bernilai sama tetap lulus uji.
// Percakapan yang tanggal pesannya PALING LAMA di antara yang belum dijawab.
//
// Ia yang harus berada di baris TERATAS tab "Belum Dijawab" — dan itulah yang
// membuktikan urutan menaiknya benar-benar berlaku.
// ── Terlihat oleh CABANG 1001 ──────────────────────────────────────────────
// Balasan TERBARU di antara yang sudah dijawab pada cabang 1001.
//
// Ia yang harus berada di baris TERATAS tab "Sudah Dijawab" — pasangan dari
// KOM-0003, dan bersamanya ia membuktikan kedua arah urutan memang berlawanan.
// DIBALAS, tetapi penjawabnya TIDAK tercatat.
//
// Ia saksi selisih satu kolom antara grid dan pencacah: baris ini MUNCUL di tab
// "Sudah Dijawab" (penyaringnya hanya `REPLYMESSAGE`) tetapi TIDAK terhitung di
// pencacah mana pun — bukan di "Answered" karena `REPLYFROM` kosong, bukan pula
// di "Not Answered" karena `REPLYMESSAGE` terisi.
//
// Tanpa baris ini, selisih itu tidak dapat dibuktikan ada.
// ── Saksi PENOLAK — tidak boleh muncul di layar mana pun ───────────────────
// Percakapan yang SUDAH DITUTUP lewat tombol "Selesai Komunikasi".
//
// Nilainya berawalan `CABANG`, dan justru itulah gunanya: penyaring yang
// ditulis sebagai awalan alih-alih perbandingan persis akan meloloskannya.
// Tanpa PENGIRIM — tertolak `SENDER IS NOT NULL`.
// Tanpa PESAN — tertolak `MESSAGE IS NOT NULL`.
// Milik cabang 1003, yang TIDAK dimiliki satu pun login contoh.
//
// Ia saksi batas cabang: petugas cabang 1001 maupun 1002 tidak boleh melihatnya,
// dan kantor pusat pun tidak — karena asal maupun tujuannya bukan `1`.
func SampleRows() []Row { return sampledata.Must[[]Row](sampleJSON, "SampleRows") }

// SampleHistory adalah baris contoh `POOLDATA.M_KOMUNIKASI_CABANG` — UTAS percakapan.
//
// # Kenapa ia ada sejak 2026-09-24
//
// Karena layar detail membaca tabel ini, bukan tabel percakapan. Penyimpanan tanpa riwayat
// akan menampilkan layar detail yang KOSONG untuk setiap percakapan contoh — dan uji yang
// memakainya gagal karena alasan yang tidak ada hubungannya dengan yang diujinya.
//
// # Kesejajaran yang dijaga
//
// Setiap baris di sini menunjuk percakapan yang BENAR-BENAR ada di SampleRows, dan isinya
// sejalan: percakapan yang sudah dijawab punya DUA ucapan — pesan lalu balasan — sementara
// yang belum dijawab punya satu.
//
// Tanggalnya pula dijaga sejalan: ucapan pertama bertanggal sama dengan `CreatedAt` barisnya,
// ucapan kedua dengan `RepliedAt`. Riwayat yang tanggalnya menyimpang dari kepalanya akan
// membuat utas terbaca dengan urutan yang tidak masuk akal.
//
// # SATU percakapan sengaja TIDAK punya riwayat
//
// KOM-0003 punya kepala tetapi tidak satu pun baris di sini. Ia saksi keadaan yang nyata di
// produksi: percakapan yang dibuat lewat jalur lain, atau data warisan sebelum tabel riwayat
// dipakai, punya kepala tanpa utas.
//
// Tanpa saksi itu, "utas kosong" dan "percakapan tidak ada" tidak dapat dibuktikan berbeda —
// dan menyamakan keduanya akan menjawab "tidak ditemukan" untuk percakapan yang nyata.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// KOM-0002 — DUA ucapan: pesannya, lalu balasannya.
// KOM-0004 — dua ucapan pula.
func SampleHistory() []ReplyHistory {
	return sampledata.Must[[]ReplyHistory](sampleJSON, "SampleHistory")
}
