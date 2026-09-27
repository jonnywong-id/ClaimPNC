package sqlstore

import (
	"database/sql"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

// Alias kueri daftar WAJIB sama persis dengan daftar kolomnya, berikut urutannya.
//
// Begitu satu alias berubah nama atau bergeser urutannya, pemindainya memasukkan nilai ke
// isian yang salah — dan akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
// Nomor polis yang tampil di kolom Nama Tertanggung tidak akan menghentikan apa pun; ia
// hanya salah.
func TestListQueryAliasesMatchResultColumns(t *testing.T) {
	// Hanya alias di UJUNG baris kolom yang dihitung, supaya konstruksi seperti
	// `CAST(NULL AS DATE)` tidak ikut terbaca sebagai alias bernama "DATE".
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	for name, want := range map[string][]string{
		"list_compliance": complianceColumns,
		"list_post_audit": postAuditColumns,
	} {
		found := []string{}
		for _, match := range alias.FindAllStringSubmatch(query(name), -1) {
			found = append(found, strings.ToUpper(match[1]))
		}

		require.Equalf(t, want, found, "alias kueri %s berbeda dari daftar kolomnya", name)
	}
}

// Jumlah isian yang dipindai harus sama dengan jumlah kolom yang dikembalikan kuerinya.
//
// Uji ini menangkap selisihnya tanpa basis data: pemindai palsu menghitung berapa banyak
// tujuan yang diminta, lalu membandingkannya dengan daftar kolomnya.
func TestScannersReadEveryColumn(t *testing.T) {
	tests := []struct {
		name    string
		scan    func(scanner) (inboxcompliance.WorkItem, error)
		columns []string
	}{
		{"scanComplianceItem", scanComplianceItem, complianceColumns},
		{"scanPostAuditItem", scanPostAuditItem, postAuditColumns},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			counter := &countingScanner{}

			_, err := tc.scan(counter)
			require.NoError(t, err)

			require.Len(t, tc.columns, counter.count,
				"jumlah kolom yang dipindai berbeda dari jumlah alias kueri")
		})
	}
}

// Setiap tab WAJIB punya rencana kuerinya, dan rencana itu tidak boleh menganggur.
//
// Tab tanpa rencana menghasilkan galat saat permintaan pertama datang; rencana tanpa tab
// adalah kode mati yang kelak dikira siap dipakai.
func TestEveryTabHasPlan(t *testing.T) {
	codes := map[string]bool{}
	for _, tab := range inboxcompliance.Tabs() {
		codes[tab.Code] = true

		selected, known := plans[tab.Code]
		require.Truef(t, known, "tab %s (%s) tidak ada di plans", tab.Code, tab.Name)

		// Kueri yang disebut rencana harus benar-benar ada; query() panik bila tidak.
		require.NotEmpty(t, query(selected.list))
		require.NotEmpty(t, query(selected.count))
		require.NotNil(t, selected.scan)
	}

	for code := range plans {
		require.Truef(t, codes[code], "rencana %q tidak dimiliki tab mana pun", code)
	}
}

// Kedua kueri yang melayani tab yang sama WAJIB memakai predikat yang sama.
//
// Bila tidak, jumlah total dan isi halaman bercerita berbeda: layar menggambar halaman yang
// tidak pernah berisi apa pun, dan selisihnya tidak terlihat sampai seseorang membuka
// halaman terakhir.
func TestCountAndListSharePredicates(t *testing.T) {
	list, count := query("list_compliance"), query("count_compliance")

	predicates := []string{
		"A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
		"wb.PXASSIGNEDOPERATORID = :1",
		"A.PYSTATUSWORK <> 'Resolved-Completed'",
		"DATAPEGA.PC_ASSIGN_WORKBASKET",
		"wb.PXOBJCLASS = 'Assign-WorkBasket'",
	}

	for _, predicate := range predicates {
		require.Containsf(t, list, predicate, "list_compliance kehilangan %q", predicate)
		require.Containsf(t, count, predicate, "count_compliance kehilangan %q", predicate)
	}
}

// Nama workbasket TIDAK boleh ditulis tetap di dalam teks SQL.
//
// Ia datang sebagai bind, sehingga satu-satunya tempat nama antrean ditulis adalah konstanta
// Go yang menyebut baris flow asalnya. Ini yang membedakan modul ini dari kueri Inbox Admin,
// yang menuliskan `'RCLPUCL'` langsung di dalam SQL-nya.
func TestWorkbasketIsBoundNotInlined(t *testing.T) {
	for _, name := range []string{"list_compliance", "count_compliance"} {
		require.NotContainsf(t, query(name), "CompliancePNC",
			"%s menuliskan nama workbasket di dalam SQL; ia harus datang sebagai bind", name)
	}
}

// Jumlah bind yang dipakai kueri harus sama dengan jumlah argumen yang dikirim Repo.
//
// Argumen yang kurang menghasilkan ORA-01008 saat permintaan pertama datang; argumen yang
// berlebih menghasilkan ORA-01036. Keduanya hanya terlihat di produksi bila tidak diuji di
// sini, karena kuerinya tidak pernah dijalankan saat kompilasi.
func TestBindCountMatchesSuppliedArguments(t *testing.T) {
	bind := regexp.MustCompile(`:(\d+)`)

	expected := map[string]int{
		// workbasket, offset, jumlah baris
		"list_compliance": 3,
		// workbasket
		"count_compliance": 1,
		// offset, jumlah baris — tab Post Audit tidak punya penyaring
		"list_post_audit": 2,
		// tidak ada
		"count_post_audit":       0,
		"check_table":            0,
		"check_table_post_audit": 0,
	}

	for name, want := range expected {
		highest := 0
		for _, match := range bind.FindAllStringSubmatch(query(name), -1) {
			index, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			if index > highest {
				highest = index
			}
		}
		require.Equalf(t, want, highest, "jumlah bind kueri %s tidak sesuai", name)
	}
}

// Tidak ada pola SQL terlarang di berkas kueri modul ini.
//
// Daftarnya mengikuti `08-TECHNICAL-STRATEGY.md` §4.3 dan §6. Ia diuji di sini, bukan hanya
// dijaga saat review, karena aturan yang hanya ada di dokumen akan dilanggar begitu tekanan
// jadwal membuat orang menempuh jalan pintas.
func TestNoForbiddenSQLPatterns(t *testing.T) {
	forbidden := []struct {
		pattern *regexp.Regexp
		reason  string
	}{
		{regexp.MustCompile(`(?i)\bSELECT\s+\*`), "SELECT * dilarang; sebutkan nama kolom"},
		{regexp.MustCompile(`(?i)\bNVL\s*\(`), "pakai COALESCE, bukan NVL"},
		{regexp.MustCompile(`(?i)\bROWNUM\b`), "pakai OFFSET … FETCH NEXT, bukan ROWNUM"},
		{regexp.MustCompile(`(?i)\bSYSDATE\b`), "pakai CURRENT_TIMESTAMP, atau hitung di Go"},
		{regexp.MustCompile(`(?i)\bDECODE\s*\(`), "pakai CASE WHEN, bukan DECODE"},
		{regexp.MustCompile(`(?i)\bTO_CHAR\s*\(`), "pemformatan tampilan dilakukan di Go"},
		{regexp.MustCompile(`(?i)\bFROM\s+DUAL\b`), "hilangkan FROM DUAL"},
		{regexp.MustCompile(`\{ASIS:`), "perangkaian nilai ke dalam SQL dilarang"},
	}

	for _, name := range []string{
		"list_compliance", "count_compliance",
		"list_post_audit", "count_post_audit",
		"check_table", "check_table_post_audit",
	} {
		text := query(name)
		for _, rule := range forbidden {
			require.Falsef(t, rule.pattern.MatchString(text),
				"kueri %s melanggar aturan SQL: %s", name, rule.reason)
		}
	}
}

// countingScanner menghitung berapa banyak tujuan yang diminta pemindai, lalu mengisi
// masing-masing dengan nilai kosong yang sah.
type countingScanner struct{ count int }

func (c *countingScanner) Scan(dest ...any) error {
	c.count = len(dest)

	for _, target := range dest {
		switch typed := target.(type) {
		case *sql.NullString:
			*typed = sql.NullString{}
		case *sql.NullTime:
			*typed = sql.NullTime{Valid: true, Time: time.Now()}
		}
	}

	return nil
}
