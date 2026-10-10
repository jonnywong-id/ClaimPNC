// Command cekddl memeriksa bahwa setiap kueri Report Klaim benar-benar DAPAT DIURAI oleh
// basis data — nama tabel ada, nama kolom ada, dan sintaksnya sah.
//
// Caranya `EXPLAIN PLAN FOR <kueri>`: peladen mengurai dan menyelesaikan seluruh nama,
// lalu berhenti sebelum membaca satu baris pun. Jadi ia memeriksa tanpa menyentuh data.
//
// # Kenapa bukan PrepareContext
//
// Pilihan pertama adalah `db.PrepareContext`, dan ia melaporkan "36 lulus, 0 gagal" —
// termasuk untuk kueri yang sengaja dirusak menjadi `SELECT FROM WHERE`. Penyebabnya:
// go-ora tidak menempuh perjalanan ke peladen saat menyiapkan pernyataan, sehingga
// "lulus" di sana tidak berarti apa pun.
//
// Karena itu perkakas ini SELALU menguji dirinya sendiri lebih dulu dengan dua pernyataan
// kendali — satu yang pasti sah dan satu yang pasti rusak. Bila yang rusak ikut lulus,
// perkakasnya berhenti alih-alih melaporkan kabar baik yang palsu.
//
// Perkakas sementara. Ia dipakai sewaktu memindahkan kueri Pega, bukan bagian dari
// aplikasi — `claimpnc -periksa` yang menjadi pemeriksa tetapnya.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "github.com/sijms/go-ora/v2"

	"claim-pnc/internal/platform/config"
	"claim-pnc/internal/reportklaim/repo/sqlstore"
)

// ora memotong pesan go-ora yang panjang menjadi baris ORA-nya saja.
var ora = regexp.MustCompile(`ORA-\d+:[^\n]*`)

// anekaOnly adalah kueri yang berjalan pada koneksi KEDUA portal (`ANEKA_<ALIAS>_*`),
// pengganti DB Link `@ASMD` (`D-25`, `R-03`). Memeriksanya terhadap POOLDATA akan selalu
// gagal ORA-00942, dan kegagalan itu palsu.
var anekaOnly = map[string]bool{
	"report_holiday_calendar": true,
	"report_mitra_logins":     true,
}

// Sejak 2026-10-09 keduanya punya CADANGAN ber-DB Link yang berjalan pada koneksi portal,
// sehingga keduanya dapat diperiksa di sini — dan justru harus, karena itulah jalur yang
// benar-benar dipakai selama ANEKA_* kosong.

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadEnvFile(".env"); err != nil {
		return fmt.Errorf("membaca .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("konfigurasi: %w", err)
	}

	p, ok := cfg.Portal[cfg.PrimaryPortal]
	if !ok || !p.Complete() {
		return fmt.Errorf("portal utama %q tidak punya parameter koneksi lengkap", cfg.PrimaryPortal)
	}

	db, err := sql.Open("oracle", dsn(p))
	if err != nil {
		return fmt.Errorf("membuka koneksi: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("menghubungi %s: %w", cfg.PrimaryPortal, err)
	}
	fmt.Printf("Portal %s · %s:%d/%s · pengguna %s\n\n",
		cfg.PrimaryPortal, p.Host, p.Port, p.Service, p.User)

	if err := ujiPerkakas(ctx, db); err != nil {
		return err
	}

	ujiBind(ctx, db)
	cariObjekKoneksiKedua(ctx, db)
	ujiLewatDBLink(ctx, db)
	ujiPenyaringMitra(ctx, db)
	diagnosaNolBaris(ctx, db)
	sebaranStatusApprove(ctx, db)

	// Tahap 1 — urai saja, tanpa membaca baris.
	uraiErr := periksaKueri(ctx, db)

	// Tahap 2 — jalankan sungguhan, dengan izin Work Owner 2026-10-09. Dijalankan meski
	// tahap 1 gagal: kueri yang lulus urai bisa saja tetap gagal dijalankan, dan kedua
	// daftarnya lebih berguna dibaca berdampingan daripada satu per satu.
	fmt.Println()
	probeErr := jalankanProbe(ctx, db)

	if uraiErr != nil {
		return uraiErr
	}
	return probeErr
}

func dsn(d config.Database) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		url.QueryEscape(d.User), url.QueryEscape(d.Password), d.Host, d.Port, d.Service)
}

// ujiPerkakas membuktikan alat ukurnya memang mengukur, SEBELUM angkanya dipercaya.
func ujiPerkakas(ctx context.Context, db *sql.DB) error {
	if err := explain(ctx, db, "SELECT 1 FROM DUAL"); err != nil {
		return fmt.Errorf("kendali sah justru ditolak — perkakasnya rusak: %w", err)
	}
	if err := explain(ctx, db, "SELECT FROM WHERE"); err == nil {
		return fmt.Errorf("kendali rusak justru diterima — perkakasnya tidak memeriksa apa pun")
	}
	fmt.Println("Uji perkakas: kendali sah lulus, kendali rusak ditolak. Angkanya boleh dipercaya.")
	fmt.Println()
	return nil
}

func explain(ctx context.Context, db *sql.DB, q string) error {
	_, err := db.ExecContext(ctx, "EXPLAIN PLAN SET STATEMENT_ID='cekddl' FOR "+strings.TrimSpace(q))
	return err
}

func periksaKueri(ctx context.Context, db *sql.DB) error {
	semua := sqlstore.AllQueriesForCheck()

	nama := make([]string, 0, len(semua))
	for n := range semua {
		nama = append(nama, n)
	}
	sort.Strings(nama)

	var lulus, gagal, dilewati int
	for _, n := range nama {
		if anekaOnly[n] {
			dilewati++
			fmt.Printf("  %-34s DILEWATI  koneksi kedua (ANEKA_*) belum terpasang\n", n)
			continue
		}
		if err := explain(ctx, db, semua[n]); err != nil {
			gagal++
			fmt.Printf("  %-34s GAGAL     %s\n", n, ora.FindString(err.Error()))
			continue
		}
		lulus++
		fmt.Printf("  %-34s lulus\n", n)
	}

	fmt.Printf("\n%d lulus · %d GAGAL · %d dilewati · %d kueri\n",
		lulus, gagal, dilewati, len(nama))
	if gagal > 0 {
		return fmt.Errorf("%d kueri tidak dapat diurai basis data", gagal)
	}
	return nil
}
