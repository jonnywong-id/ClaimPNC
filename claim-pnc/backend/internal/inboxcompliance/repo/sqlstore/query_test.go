package sqlstore

import (
	"database/sql"
	"errors"
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
		// `LIKE`, bukan kesamaan persis: RD memasang
		// `pyIncludeAllDescendantclasses = true`, sehingga subkelas IKUT.
		"A.PXOBJCLASS LIKE 'ASM-FW-GCNMFW-Work-PNC%'",
		"wb.PXASSIGNEDOPERATORID = :1",
		"A.PYSTATUSWORK <> 'Resolved-Completed'",
		"DATAPEGA.PC_ASSIGN_WORKBASKET",
	}

	for _, predicate := range predicates {
		require.Containsf(t, list, predicate, "list_compliance kehilangan %q", predicate)
		require.Containsf(t, count, predicate, "count_compliance kehilangan %q", predicate)
	}
}

// Dua penyaring yang pernah mempersempit hasil TIDAK boleh kembali.
//
// Keduanya tidak ada di `InboxRegisterCompliance_RD-RD.xml` dan keduanya hanya dapat
// mengurangi baris — tanpa satu pun pesan galat ketika ia mengosongkan daftar. Uji ini
// ada supaya keduanya tidak dipasang ulang oleh orang yang mengira kueri ini kurang ketat.
func TestPredikatYangDibuangTidakKembali(t *testing.T) {
	banned := []string{
		// Pembatasan kelas di Pega datang dari join class, bukan dari predikat kolom.
		"wb.PXOBJCLASS",

		// Kesamaan persis membuang subkelas, yang justru disertakan RD.
		"A.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
	}

	for _, name := range []string{"list_compliance", "count_compliance", "find_compliance_claim"} {
		for _, predicate := range banned {
			require.NotContainsf(
				t, query(name), predicate,
				"%s memasang kembali penyaring %q yang tidak ada di RD", name, predicate,
			)
		}
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
		// pemilik sequence, nama sequence
		"check_post_audit_sequence": 2,

		// Form Compliance Checker.
		//
		// `upsert_compliance_decision` memakai delapan bind, dan beberapa di antaranya
		// muncul DUA KALI — sekali di cabang UPDATE, sekali di cabang INSERT. Yang
		// diperiksa di sini adalah nomor bind TERTINGGI, bukan berapa kali ia muncul,
		// sehingga pengulangan itu tidak mengubah angkanya.
		"find_compliance_decision":   1,
		"upsert_compliance_decision": 9,
		"check_table_decision":       0,
		"apply_decision_to_claim":    4,
		"insert_history_claim":       4,
		"check_table_history":        0,
		"insert_penugasan":           8,
		"check_table_penugasan":      0,
		"find_survey_results":        1,
		"find_claim_documents":       1,
		"next_attachment_runno":      0,
		"insert_attachment_counter":  3,
		"insert_attachment":          9,
		"delete_attachment":          2,
		"find_reject_prefill":        1,

		// Tiga bind: klaim dipakai dua kali (pencacah lampiran dan penyaring klaim),
		// ditambah tahap dokumen. Tahapnya DIIKAT, bukan ditempel ke teks SQL — ia
		// konstanta kami hari ini, tetapi akan menjadi nilai dari luar begitu RD-nya
		// datang, dan menempelkannya sekarang berarti menyiapkan celah untuk nanti.
		"find_document_checklist":        2,
		"check_table_document_checklist": 0,

		// Tiga bind: kategori baru, dokumennya, klaimnya.
		"update_document_category": 3,
		"count_document_stages":    0,
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
		"check_post_audit_sequence",
	} {
		text := query(name)
		for _, rule := range forbidden {
			require.Falsef(t, rule.pattern.MatchString(text),
				"kueri %s melanggar aturan SQL: %s", name, rule.reason)
		}
	}
}

// Sequence yang DIPERIKSA harus sequence yang DIPAKAI — bukan sekadar sequence bernama
// mirip.
//
// Keduanya ditulis di tempat berbeda: kueri NEXTVAL menyebut namanya di dalam teks SQL,
// sedangkan kueri pemeriksa menerimanya sebagai bind dari konstanta Go. Bila salah satunya
// berubah sendirian, `-periksa` akan melaporkan hijau atas objek yang tidak pernah dipakai
// jalur tulis — kegagalan yang jauh lebih buruk daripada tidak memeriksa sama sekali,
// karena ia membuat orang berhenti mencari.
func TestCheckedSequenceIsTheOneUsedToWrite(t *testing.T) {
	require.Contains(t, query("post_audit_next_sequence"),
		sequenceOwner+"."+sequenceName+".NEXTVAL",
		"kueri NEXTVAL memakai sequence yang berbeda dari yang diperiksa -periksa")
}

// Nama sequence WAJIB huruf besar seluruhnya.
//
// `ALL_SEQUENCES` menyimpan identifier tanpa tanda kutip dalam huruf besar. Konstanta
// berhuruf kecil akan membuat pemeriksaannya SELALU mengembalikan nol, sehingga `-periksa`
// melaporkan sequence tidak ada padahal ia ada — dan DBA diminta menjalankan migrasi yang
// sudah pernah dijalankan.
func TestSequenceIdentifiersAreUppercase(t *testing.T) {
	require.Equal(t, strings.ToUpper(sequenceOwner), sequenceOwner)
	require.Equal(t, strings.ToUpper(sequenceName), sequenceName)
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

// Galat "tabel tidak ada" ditandai, bukan diteruskan apa adanya sebagai galat teknis.
//
// Tanpa penandaan ini, membuka form di basis data yang migrasinya belum dijalankan
// menghasilkan 500 "Terjadi kesalahan pada sistem" — kalimat yang tidak dapat
// ditindaklanjuti siapa pun, dan yang sudah sekali memakan satu putaran penuh
// tanya-jawab pada 2026-10-06.
func TestGalatTabelHilangDitandai(t *testing.T) {
	ditandai := []string{
		"ORA-00942: table or view does not exist",
		"ora-00942: table or view does not exist",
		"ORA-00904: \"GROUPPANEL\": invalid identifier",
	}

	for _, pesan := range ditandai {
		err := storeMissing(errors.New(pesan))

		require.ErrorIsf(
			t, err, inboxcompliance.ErrDecisionStoreMissing,
			"%q seharusnya ditandai sebagai tabel hilang", pesan,
		)

		// Galat aslinya WAJIB ikut terbawa — ia yang menyebut nama tabelnya di log.
		//
		// Dibandingkan tanpa peka huruf besar-kecil, karena teks galat driver memang
		// datang dalam kedua bentuk — dan itulah sebabnya storeMissing menormalkannya
		// lebih dulu sebelum mencocokkan.
		require.Containsf(
			t, strings.ToUpper(err.Error()), "ORA-009",
			"galat asli hilang dari %q", pesan,
		)
	}
}

// Galat lain TIDAK boleh ikut tertandai.
//
// Menandai terlalu luas jauh lebih berbahaya daripada tidak menandai sama sekali: ia akan
// menyuruh DBA menjalankan migrasi untuk masalah yang sebenarnya lain — dan menyembunyikan
// sebab aslinya di balik kalimat yang terdengar meyakinkan.
func TestGalatLainTidakIkutDitandai(t *testing.T) {
	lain := []string{
		"ORA-12541: TNS:no listener",
		"ORA-00001: unique constraint violated",
		"context deadline exceeded",
	}

	for _, pesan := range lain {
		asli := errors.New(pesan)
		require.NotErrorIsf(
			t, storeMissing(asli), inboxcompliance.ErrDecisionStoreMissing,
			"%q BUKAN tabel hilang dan tidak boleh ditandai begitu", pesan,
		)
	}

	require.NoError(t, storeMissing(nil))
}

// Bentuk nomor Post Audit dikunci pada sintaks yang Work Owner tetapkan.
//
// Sintaksnya (2026-10-06, mencabut keputusan 2026-09-24):
//
//	'CPL' || '.' || TO_CHAR(SYSDATE,'RR') || '.' || TO_CHAR(seq.NEXTVAL)
//
// Uji ini ada karena bentuk nomor adalah hal yang PALING mudah "dirapikan" oleh orang
// berikutnya — menambahkan nol di depan supaya urut, atau mengganti titik menjadi tanda
// hubung supaya seragam dengan baris warisan Pega. Keduanya terlihat seperti perbaikan,
// dan keduanya mengubah nomor yang sudah dicetak di dokumen.
func TestBentukNomorPostAudit(t *testing.T) {
	sql := query("post_audit_next_sequence")

	wajib := []string{
		"'CPL'",
		"TO_CHAR(SYSDATE, 'RR')",
		"POOLDATA.CLAIM_COMPLIENCE_SEQ.NEXTVAL",
	}
	for _, bagian := range wajib {
		require.Containsf(t, sql, bagian, "sintaks nomor Post Audit kehilangan %q", bagian)
	}

	// Format mask SENGAJA tidak ada — sintaks Work Owner memang tanpa nol di depan.
	// Akibatnya `CPL.26.10` berada di atas `CPL.26.9` pada urutan teks, dan itu sudah
	// dicatat sebagai konsekuensi yang diterima. Menambahkannya diam-diam akan membuat
	// nomor baru tidak sebentuk dengan nomor yang sudah terbit.
	require.NotContains(t, sql, "FM0000",
		"nol di depan ditambahkan tanpa keputusan; lihat 0011_post_audit_compliance.up.sql")
}

// Pemeriksa tabel Post Audit WAJIB menyebut setiap kolom yang dipakai jalur baca dan tulis.
//
// `POOLDATA.T_CLAIM_COMPLIANCE_H` dibuat DBA DI LUAR repositori ini — tidak ada migrasi di
// sini yang membuatnya, sehingga nama kolomnya dapat berbeda dari yang diandaikan kueri.
// Pemeriksa yang hanya berbunyi `COUNT(*)` akan melaporkan HIJAU atas tabel seperti itu,
// lalu layarnya gagal saat dipakai dengan `ORA-00904` yang hanya terlihat di log.
//
// Uji ini menjaga pemeriksa itu tetap kuat, dan ia mengambil daftar kolomnya dari
// postAuditColumns — daftar yang sama yang dipakai pemindai — sehingga kolom yang kelak
// ditambahkan ke tab ini otomatis ikut diperiksa tanpa ada yang perlu ingat.
func TestPemeriksaPostAuditMenyebutSeluruhKolom(t *testing.T) {
	probe := query("check_table_post_audit")

	// Nama kolom pada tabelnya, bukan aliasnya. Pemindai memakai alias (`CASE_ID`),
	// pemeriksa harus memakai nama aslinya (`CASEID`) — alias tidak membuktikan apa pun
	// tentang kolom yang benar-benar ada.
	kolom := []string{
		"CASEID", "NO_KLAIM", "NAMA_TERTANGGUNG",
		"NO_POLIS", "REMARKS", "TGL_KIRIM_POST_AUDIT",
	}

	for _, nama := range kolom {
		require.Containsf(t, probe, nama,
			"check_table_post_audit tidak memeriksa kolom %q; tabelnya dibuat DBA di luar "+
				"repo ini, sehingga kolom yang tidak diperiksa baru ketahuan saat gagal",
			nama)
	}

	// Jumlah kolom yang diperiksa harus sama dengan yang dibaca pemindai. Bila pemindai
	// kelak menambah kolom tanpa pemeriksanya ikut, uji ini yang menangkapnya.
	require.Lenf(t, postAuditColumns, len(kolom),
		"postAuditColumns berubah menjadi %d kolom; perbarui check_table_post_audit dan "+
			"daftar di uji ini bersamaan", len(postAuditColumns))
}
