package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
)

// Pemeriksa untuk `claimpnc -periksa`.
//
// # Kenapa modul ini butuh pemeriksanya sendiri
//
// Ketiga tabel yang dibacanya tidak disentuh modul lain mana pun. Tanpa pemeriksa,
// ketiadaan atau salah nama kolomnya baru ketahuan ketika seorang pelapor membuka layar
// dan mendapat galat 500 — dan di modul ini yang gagal adalah laporan ke regulator.
//
// # Kenapa sentinel-nya bertipe TEKS
//
// Karena ketiga kolom yang disaring bertipe teks. Mengirim sentinel teks ke kolom NUMBER
// menghasilkan ORA-01722, sehingga pemeriksanya SELALU gagal dan tidak pernah memeriksa
// apa pun — cacat yang sudah pernah terjadi dua kali di repo ini
// (`docs/catatan-pengembangan.md` §54.2). Yang diperiksa di sini adalah tabel dan
// kolomnya dapat diurai Oracle, bukan isinya.
const probeSentinel = "__PERIKSA__"

// ProbeResult adalah hasil satu pemeriksaan.
type ProbeResult struct {
	Name string
	Err  error
}

// OK menyatakan pemeriksaannya lolos.
func (p ProbeResult) OK() bool { return p.Err == nil }

// Probe menjalankan seluruh pemeriksaan modul ini.
//
// Ketiganya dijalankan SELURUHNYA meski yang pertama gagal: pelapor perlu tahu ketiga
// tabelnya sekaligus, bukan satu per satu setiap kali aplikasi dijalankan ulang.
func Probe(ctx context.Context, db *sql.DB) []ProbeResult {
	checks := []struct {
		name  string
		query string
	}{
		// Ketiga yang pertama menghidupkan PEMANTAUAN.
		{"Tabel SLIK OJK (segmen D01) terbaca", "probe_slik_table"},
		{"Tabel objek klaim (segmen F06) terbaca", "probe_objectlist_table"},
		{"Tabel polis untuk kolom Operasi Data terbaca", "probe_general_table"},

		// Ketiga berikutnya menghidupkan PENYUSUNAN laporan ("Proses Data Klaim").
		// Ketiadaannya tidak menghalangi pemantauan — dan perbedaan itu perlu terbaca
		// dari laporannya, bukan disimpulkan.
		{"Tabel klaim PNC untuk penyusunan laporan terbaca", "probe_claim_pnc_table"},
		{"Tabel adjustment untuk nilai tunggakan terbaca", "probe_adjustment_table"},
		{"Tabel coverage untuk jumlah kewajiban terbaca", "probe_coverage_table"},

		// Yang terakhir menghidupkan PENGIRIMAN ("SLIK OJK").
		{"Tabel pengiriman SLIK terbaca", "probe_submission_table"},
	}

	results := make([]ProbeResult, 0, len(checks))
	for _, check := range checks {
		results = append(results, ProbeResult{
			Name: check.name,
			Err:  probeOne(ctx, db, check.query),
		})
	}
	return results
}

// probeOne menjalankan satu kueri pemeriksa.
//
// Hasil hitungannya diabaikan dengan sengaja — yang diperiksa adalah kuerinya BERJALAN.
// Nol baris adalah jawaban yang benar untuk sentinel yang memang tidak ada.
func probeOne(ctx context.Context, db *sql.DB, name string) error {
	var total sql.NullInt64
	if err := db.QueryRowContext(ctx, query(name), probeSentinel).Scan(&total); err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: %s: %w", name, err)
	}
	return nil
}
