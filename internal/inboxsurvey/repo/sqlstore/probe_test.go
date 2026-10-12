package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	go_ora "github.com/sijms/go-ora/v2"

	"claim-pnc/internal/inboxsurvey"
)

// Probe diagnosis: menjalankan kedua varian kueri terhadap basis data SUNGGUHAN, lalu
// mencetak galat Oracle-nya apa adanya.
//
// # Kenapa ada
//
// Dua kali berturut-turut sebab kegagalan layar ditebak dari membaca kode — kolom hilang,
// lalu penomoran bind — dan dua kali tebakannya tidak menyelesaikan masalahnya. Menebak
// ketiga kalinya lebih mahal daripada menanyakan langsung kepada basis datanya.
//
// # Dilewati secara baku
//
// Hanya jalan bila `PROBE_INBOXSURVEY` di-set. Ia menyentuh basis data sungguhan, sehingga
// tidak boleh ikut pada `go test ./...` biasa maupun di CI.
//
//	PROBE_INBOXSURVEY=1 go test ./internal/inboxsurvey/repo/sqlstore -run TestProbeKueri -v
//
// # Yang TIDAK dicetaknya
//
// Tidak satu pun nilai baris, nama tertanggung, nomor polis, hostname, maupun kredensial
// (`D-69`). Yang dicetak hanya: nama kueri, jumlah baris, dan teks galat Oracle.
func TestProbeKueri(t *testing.T) {
	if os.Getenv("PROBE_INBOXSURVEY") == "" {
		t.Skip("probe basis data; set PROBE_INBOXSURVEY=1 untuk menjalankannya")
	}

	prefix := strings.TrimSpace(os.Getenv("PROBE_PORTAL"))
	if prefix == "" {
		prefix = strings.TrimSpace(os.Getenv("PORTAL_UTAMA"))
	}
	if prefix == "" {
		prefix = "ASM"
	}

	env := func(suffix string) string {
		return strings.TrimSpace(os.Getenv("POOLDATA_" + prefix + "_" + suffix))
	}

	port := 1521
	if p := env("PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	url := go_ora.BuildUrl(env("HOST"), port, env("SERVICE"), env("PENGGUNA"), env("SANDI"), nil)
	db, err := sql.Open("oracle", url)
	if err != nil {
		t.Fatalf("membuka koneksi portal %s: %v", prefix, err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("menyapa basis data portal %s: %v", prefix, err)
	}
	fmt.Printf("\n=== portal %s terhubung ===\n", prefix)

	// Cakupan sengaja dibuat TIDAK COCOK dengan nama siapa pun. Yang diuji adalah apakah
	// kuerinya dapat di-parse dan dieksekusi, bukan isinya — dan cakupan yang tidak cocok
	// membuat probe ini tidak pernah menarik satu baris data nasabah pun.
	const cakupan = "|PROBE TIDAK ADA|"
	const login = "PROBE"

	jalankan := func(nama string, args ...any) {
		rows, err := db.QueryContext(ctx, query(nama), args...)
		if err != nil {
			fmt.Printf("%-18s GAGAL: %v\n", nama, err)
			return
		}
		defer rows.Close()

		n := 0
		for rows.Next() {
			n++
		}
		if err := rows.Err(); err != nil {
			fmt.Printf("%-18s GAGAL saat menelusuri: %v\n", nama, err)
			return
		}
		fmt.Printf("%-18s OK (%d baris)\n", nama, n)
	}

	fmt.Println("--- varian terbatas ---")
	jalankan("count_tabs",
		sql.Named("login", login), sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered), sql.Named("scope", cakupan))
	jalankan("list_tasks",
		sql.Named("scope", cakupan), sql.Named("tab", string(inboxsurvey.TabNotAnswered)),
		sql.Named("login", login), sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
		sql.Named("search", nil), sql.Named("skip", 0), sql.Named("take", 25))

	fmt.Println("--- varian penuh ---")
	jalankan("count_tabs_full",
		sql.Named("work_done", inboxsurvey.StatusWorkCompleted),
		sql.Named("work_rejected", inboxsurvey.StatusWorkRejected),
		sql.Named("adjuster_confirmed", inboxsurvey.AdjusterConfirmed),
		sql.Named("invoice_fee", inboxsurvey.StatusInvoiceFee),
		sql.Named("login", login), sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered), sql.Named("scope", cakupan))
	jalankan("list_tasks_full",
		sql.Named("scope", cakupan), sql.Named("tab", string(inboxsurvey.TabOutstanding)),
		sql.Named("work_done", inboxsurvey.StatusWorkCompleted),
		sql.Named("work_rejected", inboxsurvey.StatusWorkRejected),
		sql.Named("adjuster_confirmed", inboxsurvey.AdjusterConfirmed),
		sql.Named("invoice_fee", inboxsurvey.StatusInvoiceFee),
		sql.Named("login", login), sql.Named("msg_open", inboxsurvey.CommunicationOpen),
		sql.Named("msg_answered", inboxsurvey.CommunicationAnswered),
		sql.Named("search", nil), sql.Named("skip", 0), sql.Named("take", 25))

	fmt.Println("--- KPI (kelima bentuk) ---")
	jalankan("kpi_by_adjuster", sql.Named("scope", cakupan), sql.Named("kpi_type", "FINAL"))
	jalankan("kpi_by_adjuster_all", sql.Named("scope", cakupan))
	jalankan("kpi_by_year", sql.Named("scope", cakupan), sql.Named("year", nil), sql.Named("quarter", "3"))
	jalankan("kpi_by_quarter_year", sql.Named("scope", cakupan), sql.Named("year", nil))
	jalankan("kpi_detail", sql.Named("scope", cakupan), sql.Named("year", nil), sql.Named("quarter", nil))
	jalankan("kpi_years", sql.Named("scope", cakupan))

	fmt.Println("--- pemeriksa kesiapan ---")
	jalankan("check_new_columns")
	jalankan("check_filled_columns")
	fmt.Println()
}
