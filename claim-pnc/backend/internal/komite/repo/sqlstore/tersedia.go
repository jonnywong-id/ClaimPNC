package sqlstore

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

// Pendeteksi ketersediaan tabel jejak keputusan komite.
//
// ============================================================================
// KENAPA BERKAS INI ADA
// ============================================================================
//
// `POOLDATA.CPNC_KOMITE_KEPUTUSAN` dibuat migrasi `0004_komite_keputusan.up.sql`, dan
// migrasi menempuh `D-63`: permintaan tertulis tim pengembang, persetujuan Work Owner,
// pelaksanaan DBA. Akun aplikasi tidak memiliki hak DDL, sehingga aplikasi TIDAK DAPAT
// membuat tabel itu sendiri — ia hanya dapat menunggu.
//
// Sampai migrasi itu dijalankan, keempat kueri Inbox Komite gagal dengan `ORA-00942`,
// karena tiga di antaranya menggabungkan tabel tersebut. Akibatnya SELURUH layar mati:
// 1.542 kasus komite yang nyata-nyata ada di tabel warisan tidak dapat dibaca sama
// sekali, hanya karena satu kolom pelengkap tidak tersedia.
//
// Itu tidak sepadan. Yang hilang bila tabelnya belum ada hanyalah **keputusan yang
// dicatat aplikasi ini**; daftar pekerjaannya sendiri, nilai klaimnya, penilaian AI-nya,
// dan riwayat keputusan Pega seluruhnya berada di tabel warisan dan tetap terbaca.
//
// Karena itu repo memeriksa ketersediaannya lebih dulu, lalu memilih kueri yang sesuai.
// Layar tetap menyala dalam MODE TERBATAS, dan keterbatasannya dinyatakan terang kepada
// pemakainya lewat penanda pada respons — bukan disembunyikan.
//
// ============================================================================
// KENAPA PEMERIKSAANNYA MEMAKAI SELECT BIASA, BUKAN KUERI KATALOG
// ============================================================================
//
// `ALL_TABLES` milik Oracle dan `information_schema` milik PostgreSQL. Memakai salah
// satunya berarti menambah pengecualian portabilitas keempat, sementara `D-20` menetapkan
// satu set SQL yang berjalan di keduanya dan `09-DATABASE-STRATEGY.md` §3 menyebut
// tiganya satu per satu.
//
// `SELECT … WHERE 1 = 0` berjalan apa adanya di kedua basis data, tidak mengambil satu
// baris pun, dan sekaligus menjawab pertanyaan yang SEBENARNYA ingin kita jawab: "dapatkah
// akun ini membaca tabel itu". Tabel yang ada tetapi tak dapat dibaca akun aplikasi
// berakibat sama persis dengan tabel yang tidak ada, dan keduanya memang harus ditangani
// sama.
//
// Yang dipakai adalah `decision_check_table` — kueri yang sudah ada, dipakai pula mode
// `-periksa`. Satu pernyataan untuk dua keperluan, sehingga tidak ada dua definisi
// "tabelnya dapat dipakai" yang bisa berbeda diam-diam.

// probeCooldown adalah jeda terpendek antar-pemeriksaan ketika tabelnya belum ada.
//
// Tanpa jeda, setiap permintaan inbox akan menambah satu perjalanan ke basis data selama
// berhari-hari sampai DBA menjalankan migrasinya. Dengan jeda ini, biayanya paling banyak
// dua kueri per menit per instans.
//
// Jedanya sengaja pendek: begitu DBA menjalankan migrasi 0004, layar pulih sendiri dalam
// setengah menit tanpa perlu me-restart aplikasi. Aplikasi yang menuntut restart setelah
// perubahan skema akan membuat orang menunda menjalankan migrasinya.
const probeCooldown = 30 * time.Second

// tableProbe mengingat apakah sebuah tabel dapat dipakai, tanpa menanyakannya berulang.
//
// # Jawaban "ada" disimpan selamanya, jawaban "belum ada" tidak
//
// Tabel yang sudah dibuat tidak lenyap dengan sendirinya, sehingga jawaban positif tidak
// perlu ditanyakan dua kali. Jawaban negatif justru DIHARAPKAN berubah — itulah yang
// terjadi ketika DBA menjalankan migrasinya — sehingga ia hanya ditahan selama
// probeCooldown.
type tableProbe struct {
	db    *sql.DB
	query string

	mu        sync.Mutex
	present   bool
	checkedAt time.Time
}

func newTableProbe(db *sql.DB, queryName string) *tableProbe {
	return &tableProbe{db: db, query: queryName}
}

// available menjawab apakah tabelnya dapat dipakai saat ini.
//
// Kegagalan APA PUN dijawab "belum tersedia", termasuk basis data yang sedang tidak dapat
// dihubungi. Itu disengaja dan tidak menyembunyikan apa pun: bila basis datanya benar-benar
// sedang putus, kueri utama yang menyusul akan gagal juga dan layar menampilkan galatnya.
// Yang dihindari di sini hanyalah menebak-nebak sebab kegagalan di tempat yang tidak punya
// cukup keterangan untuk membedakannya.
func (p *tableProbe) available(ctx context.Context) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.present {
		return true
	}
	if !p.checkedAt.IsZero() && time.Since(p.checkedAt) < probeCooldown {
		return false
	}
	p.checkedAt = time.Now()

	rows, err := p.db.QueryContext(ctx, query(p.query))
	if err != nil {
		return false
	}
	failed := rows.Err()
	_ = rows.Close()

	p.present = failed == nil
	return p.present
}
