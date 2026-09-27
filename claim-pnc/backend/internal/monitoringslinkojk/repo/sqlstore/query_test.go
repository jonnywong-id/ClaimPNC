package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
)

// readOnlyQueries adalah kueri yang TIDAK boleh menulis apa pun.
//
// Daftar putih, bukan daftar hitam. Sejak modul ini punya aksi tulis (2026-09-26),
// memeriksa "tidak ada satu pun kueri yang menulis" menjadi salah — yang benar adalah
// **kueri mana yang boleh**. Kueri baru yang tidak disebut di sini ikut terperiksa lewat
// TestEveryQueryIsClassified, sehingga ia tidak dapat lolos hanya karena tidak didaftarkan.
var readOnlyQueries = []string{
	"d01_rows", "d01_count", "d01_export",
	"f06_rows", "f06_count", "f06_export",
	"source_rows",
	"report_count",
	"submission_next",
	"probe_slik_table", "probe_objectlist_table", "probe_general_table",
	"probe_claim_pnc_table", "probe_adjustment_table", "probe_coverage_table",
	"probe_submission_table",
}

// writeQueries adalah kueri yang MEMANG menulis.
//
// Ketiganya menulis tabel yang selama masa paralel juga diisi Pega (`P-1`). Work Owner
// memutuskan pada 2026-09-26 bahwa ketiga tombolnya tetap dibangun — lihat kepala
// write.sql. Daftar ini yang membuat keputusan itu terbaca sebagai daftar tertutup: kueri
// tulis KEEMPAT tidak dapat ditambahkan tanpa menyunting berkas uji ini.
var writeQueries = []string{
	"report_insert",
	"submission_insert",
	"submission_done",
}

// Kueri baca TIDAK boleh menulis.
//
// Aturan yang hanya ada di komentar akan dilanggar pada bulan ketiga, ketika tekanan
// jadwal membuat orang menempuh jalan pintas.
func TestReadOnlyQueriesNeverWrite(t *testing.T) {
	forbidden := []string{"INSERT", "UPDATE", "DELETE", "MERGE", "TRUNCATE", "DROP", "ALTER"}

	for _, name := range readOnlyQueries {
		upper := strings.ToUpper(query(name))
		for _, word := range forbidden {
			require.NotContainsf(t, upper, word+" ",
				"kueri %q memuat %s padahal ia kueri baca", name, word)
		}
	}
}

// SETIAP kueri wajib terdaftar sebagai baca atau tulis.
//
// Tanpa uji ini, daftar putih di atas dapat dilewati begitu saja: kueri tulis baru yang
// tidak didaftarkan tidak akan diperiksa siapa pun.
func TestEveryQueryIsClassified(t *testing.T) {
	known := map[string]bool{}
	for _, name := range readOnlyQueries {
		known[name] = true
	}
	for _, name := range writeQueries {
		require.Falsef(t, known[name], "kueri %q terdaftar sebagai baca DAN tulis", name)
		known[name] = true
	}

	for name := range queries {
		require.Truef(t, known[name],
			"kueri %q belum digolongkan sebagai baca atau tulis di query_test.go", name)
	}
}

// Kueri tulis hanya menyentuh KEDUA tabel yang disepakati.
//
// Ia yang mencegah aksi tulis merembet: satu `UPDATE` ke tabel klaim yang diselipkan
// kelak tidak akan lolos uji ini.
func TestWriteQueriesTouchOnlyAgreedTables(t *testing.T) {
	allowed := []string{
		"POOLDATA.T_CLAIM_SLIK_OJK",
		"POOLDATA.T_CLAIM_SLINK_INDIVIDU",
	}

	for _, name := range writeQueries {
		upper := strings.ToUpper(query(name))
		var matched bool
		for _, table := range allowed {
			if strings.Contains(upper, table) {
				matched = true
			}
		}
		require.Truef(t, matched,
			"kueri tulis %q menyentuh tabel di luar yang disepakati", name)
	}
}

// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL.
//
// Kedua kueri lama justru begitu: penyaringnya dirangkai dari isian layar lalu disisipkan
// mentah dengan `{ASIS:…}`. Uji ini memastikan polanya tidak ikut terbawa.
func TestQueriesUseBindParametersOnly(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %q memuat pola perangkaian warisan", name)
		require.NotContainsf(t, text, "||'", "kueri %q merangkai teks SQL", name)
	}
}

// Kueri yang diharapkan ada, benar-benar ada.
//
// Salah ketik pada nama kueri tidak menghasilkan galat kompilasi — ia panik saat pertama
// dipanggil, yang di produksi berarti saat pengguna pertama membuka layarnya.
func TestQueriesPresent(t *testing.T) {
	for _, name := range []string{
		"d01_rows", "d01_count", "d01_export",
		"f06_rows", "f06_count", "f06_export",
		"probe_slik_table", "probe_objectlist_table", "probe_general_table",
	} {
		require.NotPanicsf(t, func() { query(name) }, "kueri %q tidak ditemukan", name)
	}
}

// ============================================================================
// JUMLAH PARAMETER
// ============================================================================

var bindPattern = regexp.MustCompile(`:(\d+)`)

// highestBind mengembalikan nomor parameter tertinggi di dalam satu kueri.
func highestBind(text string) int {
	highest := 0
	for _, match := range bindPattern.FindAllStringSubmatch(text, -1) {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if value > highest {
			highest = value
		}
	}
	return highest
}

// Jumlah argumen yang dikirim WAJIB sama dengan jumlah parameter di berkas .sql.
//
// # Kenapa uji ini yang paling penting di berkas ini
//
// Pola `(:n IS NULL OR …)` menyebut nilainya lebih dari sekali, dan Oracle
// memperlakukan setiap `:n` sebagai parameter posisional tersendiri. Satu argumen yang
// tertinggal MENGGESER seluruh sisanya — dan pergeseran itu **tidak menghasilkan galat**:
// kuerinya tetap berjalan, hanya menyaring dengan nilai yang tertukar. Laporan ke
// regulator yang menyaring dengan nilai tertukar tidak terlihat salah oleh siapa pun.
func TestArgumentCountMatchesPlaceholders(t *testing.T) {
	from := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	filter := monitoringslinkojk.Filter{
		BusinessScope:         monitoringslinkojk.ScopeCreditInsurance,
		DateOfLoss:            &from,
		DateOfRequestDocument: &to,
	}.Normalize()

	repo := &Repo{branchCode: DefaultBranchCode}

	for _, segment := range []monitoringslinkojk.Segment{
		monitoringslinkojk.SegmentD01,
		monitoringslinkojk.SegmentF06,
	} {
		plan, err := repo.planFor(segment)
		require.NoError(t, err)

		require.Equalf(t, highestBind(query(plan.rowsQuery)),
			len(plan.rowArgs(repo.branchCode, filter)),
			"jumlah argumen %q tidak cocok dengan parameter di berkas .sql", plan.rowsQuery)

		require.Equalf(t, highestBind(query(plan.countQuery)),
			len(plan.countArgs(filter)),
			"jumlah argumen %q tidak cocok dengan parameter di berkas .sql", plan.countQuery)

		require.Equalf(t, highestBind(query(plan.exportQuery)),
			len(plan.exportArgs(repo.branchCode, filter)),
			"jumlah argumen %q tidak cocok dengan parameter di berkas .sql", plan.exportQuery)
	}
}

// Kueri penghitung menyaring dengan syarat yang SAMA dengan kueri barisnya.
//
// Bila keduanya berbeda, bilah halaman menjanjikan halaman yang isinya kosong — dan tidak
// ada apa pun yang menandainya.
func TestCountSharesFilterWithRows(t *testing.T) {
	pairs := [][2]string{
		{"d01_rows", "d01_count"},
		{"f06_rows", "f06_count"},
	}
	for _, pair := range pairs {
		rowsWhere := whereClause(query(pair[0]))
		countWhere := whereClause(query(pair[1]))

		require.NotEmpty(t, rowsWhere)
		require.NotEmpty(t, countWhere)

		require.Equalf(t,
			strings.Count(rowsWhere, "BUSINESSTYPE"), strings.Count(countWhere, "BUSINESSTYPE"),
			"%s dan %s tidak menyaring lini bisnis dengan cara yang sama", pair[0], pair[1])
		require.Equalf(t,
			strings.Count(rowsWhere, "REGISTERDATE_1"), strings.Count(countWhere, "REGISTERDATE_1"),
			"%s dan %s tidak menyaring tanggal dengan cara yang sama", pair[0], pair[1])
		require.Equalf(t,
			strings.Count(rowsWhere, "LIKE"), strings.Count(countWhere, "LIKE"),
			"%s dan %s tidak mencari dengan cara yang sama", pair[0], pair[1])
	}
}

// whereClause memotong bagian WHERE TERLUAR satu kueri, berhenti sebelum ORDER BY atau
// OFFSET.
//
// # Kenapa ia mencari WHERE TERAKHIR, bukan yang pertama
//
// Karena kueri segmen F06 memuat subkueri `EXISTS (… WHERE g.IDPEGA LIKE '%EDM%')` di
// dalam daftar SELECT — sebelum WHERE utamanya. Mengambil WHERE pertama akan ikut
// menghitung syarat milik subkueri itu, sehingga uji melaporkan perbedaan yang tidak ada.
//
// Versi pertama uji ini memang begitu dan GAGAL; kegagalannya berasal dari alat ukurnya,
// bukan dari kuerinya. Dicatat karena polanya berulang: alat ukur diperiksa dulu sebelum
// temuannya dipercaya (`docs/penggunaan-skill.md`).
//
// Yang dicari adalah baris yang DIMULAI dengan `WHERE` setelah dirapikan — subkueri di
// berkas ini selalu menuliskannya menjorok di tengah baris, bukan di awal baris.
func whereClause(text string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(line)), "WHERE ") {
			start = i
		}
	}
	if start < 0 {
		return ""
	}

	clause := strings.Join(lines[start:], "\n")
	for _, stop := range []string{"ORDER BY", "OFFSET"} {
		if idx := strings.Index(strings.ToUpper(clause), stop); idx >= 0 {
			clause = clause[:idx]
		}
	}
	return clause
}

// Kolom tanggal dibandingkan LANGSUNG, tanpa `TO_DATE` di sisi kolom.
//
// `to_date(B.REGISTERDATE_1,'yyyy/mm/dd')` pada kueri lama bergantung pada NLS sesi dan
// mematikan index. Uji ini menjaga keduanya tidak kembali.
func TestNoToDateOnColumn(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "TO_DATE(",
			"kueri %q memakai TO_DATE pada kolom; lihat catatan di kepala berkas .sql", name)
		require.NotContainsf(t, strings.ToUpper(text), "TO_CHAR(",
			"kueri %q memformat di SQL; pemformatan tampilan dilakukan di Go", name)
	}
}

// ============================================================================
// PENYUSUN ARGUMEN
// ============================================================================

// Batas atas rentang tanggal dimajukan satu hari dan dibandingkan `<`.
//
// Dengan `<=` terhadap tengah malam, seluruh klaim yang diregistrasi pada hari terakhir
// rentang hilang dari laporan tanpa satu pun tanda.
func TestDateBoundsUpperIsExclusiveNextDay(t *testing.T) {
	from := time.Date(2026, time.March, 1, 14, 30, 0, 0, time.UTC)
	to := time.Date(2026, time.March, 31, 9, 0, 0, 0, time.UTC)

	low, high := dateBounds(monitoringslinkojk.Filter{
		DateOfLoss:            &from,
		DateOfRequestDocument: &to,
	})

	require.Equal(t,
		time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC), low,
		"batas bawah dipangkas ke awal hari")
	require.Equal(t,
		time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), high,
		"batas atas dimajukan ke awal hari BERIKUTNYA")
}

func TestDateBoundsEmptyMeansNoFilter(t *testing.T) {
	low, high := dateBounds(monitoringslinkojk.Filter{})
	require.Nil(t, low)
	require.Nil(t, high)
}

// "SURETY BOND" MENIADAKAN Asuransi Kredit alih-alih memilih Surety Bond.
//
// Itu perilaku warisan yang direplikasi dengan sengaja (`P-5`); uji ini menyatakannya
// supaya tidak ada yang "memperbaikinya" tanpa keputusan tertulis.
func TestScopeArgsSuretyBondExcludes(t *testing.T) {
	mode, value := scopeArgs(monitoringslinkojk.ScopeSuretyBond)
	require.Equal(t, scopeExclude, mode)
	require.Equal(t, monitoringslinkojk.CreditInsuranceBusinessType, value)

	mode, value = scopeArgs(monitoringslinkojk.ScopeCreditInsurance)
	require.Equal(t, scopeInclude, mode)
	require.Equal(t, monitoringslinkojk.CreditInsuranceBusinessType, value)

	mode, value = scopeArgs(monitoringslinkojk.ScopeAll)
	require.Nil(t, mode)
	require.Nil(t, value)
}

// ============================================================================
// PEMFORMATAN NILAI
// ============================================================================

// Nilai dibaca sebagai `any` karena tipe kolomnya belum diketahui (`R-08`). Uji ini
// menjaga keempat bentuk yang mungkin datang dari godror tetap terbaca sama.
func TestTextFormatsEveryShape(t *testing.T) {
	require.Equal(t, "", text(nil))
	require.Equal(t, "PNCN.26.0001", text("  PNCN.26.0001  "))
	require.Equal(t, "PNCN.26.0002", text([]byte("PNCN.26.0002")))
	require.Equal(t, "04/03/2026",
		text(time.Date(2026, time.March, 4, 9, 15, 0, 0, time.UTC)))
	require.Equal(t, "", text(time.Time{}))
	require.Equal(t, "45000000", text(int64(45_000_000)))

	// Tanpa `'f', -1`, nilai klaim besar tampil sebagai notasi ilmiah di layar maupun
	// di berkas laporan.
	require.Equal(t, "22500000", text(float64(22_500_000)))
	require.Equal(t, "12.5", text(12.5))
}

// Kunci baris dirangkai dari beberapa bagian dengan pemisah yang tidak muncul di nomor
// klaim maupun nomor kontrak.
func TestRowKeyJoinsParts(t *testing.T) {
	require.Equal(t, "PNCN.26.0101|KTR-2026-0101",
		rowKey(" PNCN.26.0101 ", "KTR-2026-0101"))
}

// Satu klaim dengan DUA fasilitas kredit menghasilkan kunci baris yang BERBEDA.
//
// Bila kunci dirangkai dari nomor klaim saja, tabel di layar menampilkan salah satunya
// dua kali dan yang satunya hilang.
func TestD01RowKeysDifferPerContract(t *testing.T) {
	first := d01Row(map[string]string{"NO_KLAIM": "PNCN.26.0101", "CONTRACT_NO": "KTR-1"})
	second := d01Row(map[string]string{"NO_KLAIM": "PNCN.26.0101", "CONTRACT_NO": "KTR-2"})

	require.NotEqual(t,
		first.Get(monitoringslinkojk.RowKeyColumn),
		second.Get(monitoringslinkojk.RowKeyColumn))
}

// Baris F06 memuat kedua puluh kolom grid — daftar yang SAMA dengan D01.
//
// Uji ini sempat menuntut kebalikannya: hanya delapan kolom, sisanya sengaja kosong. Itu
// berlaku ketika grid F06 keliru dibangun dari judul berkas ekspor.
//
// Yang tetap dijaga adalah kolom khusus ekspor: `nomor_cif_debitur` ikut terbawa meski
// tidak tampil di grid, karena berkas ekspor F06 memasangkannya ke judul CIF-nya.
func TestF06RowCarriesTheSameColumnsAsD01(t *testing.T) {
	row := f06Row(map[string]string{
		"NO_KLAIM": "PNCN.26.0101", "CONTRACT_NO": "KTR-1",
		"NO_CIF_DEBITUR": "CIF1", "JENIS_KELAMIN": "L",
		"KETERANGAN": "contoh",
	})

	for _, column := range monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06) {
		_, exists := row[column.Key]
		require.Truef(t, exists, "kolom grid %q tidak ada di baris F06", column.Key)
	}

	require.Equal(t, "CIF1", row.Get("no_cif_debitur"))
	require.Equal(t, "CIF1", row.Get("nomor_cif_debitur"),
		"kunci khusus berkas ekspor F06 tetap terbawa")
	require.Equal(t, "contoh", row.Get("keterangan"))
}

// Kode kantor cabang jatuh ke nilai bawaan, bukan ke teks kosong.
func TestBranchCodeFallsBackToDefault(t *testing.T) {
	require.Equal(t, DefaultBranchCode, NewRepoWithBranchCode(nil, "   ").branchCode)
	require.Equal(t, "007", NewRepoWithBranchCode(nil, " 007 ").branchCode)
}
