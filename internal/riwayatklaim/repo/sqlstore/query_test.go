package sqlstore

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/riwayatklaim"
)

// searchQueryNames adalah kesebelas kueri pencarian yang benar-benar dapat dijalankan.
func searchQueryNames() []string {
	names := make([]string, 0, len(queryByType))
	for _, name := range queryByType {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// gateQueryNames adalah kueri gerbang proteksi data.
var gateQueryNames = []string{
	"protection_find",
	"protection_count_usage",
	"protection_record_usage",
	"protection_check_table",
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Tanpa uji ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpoint-nya
// — dan bentuknya panik, bukan galat yang rapi.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range append(searchQueryNames(), gateQueryNames...) {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = query(name) })
			require.NotEmpty(t, strings.TrimSpace(query(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = query("kueri_yang_tidak_pernah_ada") })
}

// Setiap tipe pencarian yang TERSEDIA wajib punya kueri, dan sebaliknya.
//
// Kedua arah diperiksa. Tipe yang tersedia tanpa kueri gagal panik saat dipakai; kueri
// tanpa tipe adalah SQL mati yang tetap ikut di-review dan dipelihara tanpa ada yang
// memanggilnya.
func TestSetiapTipeTersediaPunyaKueri(t *testing.T) {
	for _, searchType := range riwayatklaim.SearchTypes() {
		name, mapped := queryByType[searchType.Code]

		if searchType.Available {
			require.True(t, mapped,
				"tipe %q (%s) tersedia tetapi tidak punya kueri",
				searchType.Code, searchType.Label)
			require.NotPanics(t, func() { _ = query(name) })
			continue
		}

		require.False(t, mapped,
			"tipe %q (%s) belum tersedia tetapi punya kueri",
			searchType.Code, searchType.Label)
	}

	require.Len(t, queryByType, 11, "sebelas dari dua belas tipe dapat dijalankan")
}

// Keenam belas alias WAJIB sama, dalam urutan yang sama, di setiap kueri pencarian.
//
// Tiga hal bergantung padanya: satu pemindai Go melayani kesebelasnya, subquery paginasi
// merujuk kolomnya dengan nama, dan kolom yang bergeser satu posisi akan mengisi field
// yang salah TANPA galat — nama tertanggung muncul di kolom cabang, dan tak satu pun uji
// lain akan menangkapnya.
func TestSeluruhKueriPencarianMemakaiAliasYangSama(t *testing.T) {
	want := []string{
		"REFERENCE", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "LOSS_DATE",
		"BUSINESS_NAME", "BRANCH_NAME", "WORK_STATUS", "CLAIM_POSITION",
		"CLOSE_DATE", "CLOSE_NOTE", "TECHNICAL_PIC",
		"ACCEPTANCE_NUMBER", "AUCTION_HOUSE_ID", "INSURED_ITEM_NAME", "BIRTH_DATE",
	}

	aliasPattern := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)\b`)

	for _, name := range searchQueryNames() {
		t.Run(name, func(t *testing.T) {
			found := aliasPattern.FindAllStringSubmatch(query(name), -1)

			aliases := make([]string, 0, len(found))
			for _, match := range found {
				alias := strings.ToUpper(match[1])
				// `CAST(NULL AS VARCHAR2(400))` dan `AS DATE` bukan alias kolom.
				if alias == "VARCHAR2" || alias == "DATE" || alias == "NUMBER" {
					continue
				}
				aliases = append(aliases, alias)
			}

			require.Equal(t, want, aliases)
		})
	}

	// Konstanta yang dipakai membungkus kueri harus menyebut alias yang sama pula.
	for _, alias := range want {
		require.Contains(t, resultColumns, alias)
	}
}

// Setiap kueri pencarian memakai TEPAT SATU parameter, dan namanya `:1`.
//
// paged() menambahkan `:2` untuk offset dan `:3` untuk ukuran halaman. Kueri yang memakai
// dua parameter akan membuat keduanya bergeser, dan halaman yang diminta bukan halaman
// yang diterima.
func TestSetiapKueriPencarianMemakaiSatuParameter(t *testing.T) {
	bind := regexp.MustCompile(`:(\d+)`)

	for _, name := range searchQueryNames() {
		t.Run(name, func(t *testing.T) {
			used := map[string]bool{}
			for _, match := range bind.FindAllStringSubmatch(query(name), -1) {
				used[match[1]] = true
			}
			require.Equal(t, map[string]bool{"1": true}, used)
		})
	}
}

// Kueri yang sudah dibungkus tetap memakai ketiga parameter pada posisi yang benar.
func TestKueriBerhalamanMemakaiTigaParameter(t *testing.T) {
	text := paged("search_claim_number")

	require.Contains(t, text, ":1")
	require.Contains(t, text, "OFFSET :2 ROWS")
	require.Contains(t, text, "FETCH NEXT :3 ROWS ONLY")
	require.Contains(t, text, "ORDER BY REFERENCE",
		"tanpa urutan yang ditetapkan, paginasi membuat baris muncul di dua halaman")

	require.NotContains(t, counted("search_claim_number"), "OFFSET",
		"penghitung tidak boleh ikut dipaginasi")
}

// Tidak ada satu pun nilai yang dirangkai ke dalam teks SQL.
//
// Ini menutup cacat nyata, bukan cacat teoretis: kedua belas kueri lama menyisipkan nilai
// langsung ke teks SQL lewat `{InputData.CARI4}` dan `{ASIS:InputData.CARI4}`, dan yang
// kedua menyisipkan POTONGAN SQL — bukan nilai.
func TestTidakAdaPerangkaianNilaiKeSQL(t *testing.T) {
	for _, name := range append(searchQueryNames(), gateQueryNames...) {
		t.Run(name, func(t *testing.T) {
			text := query(name)
			require.NotContains(t, text, "{ASIS", "pola penyisipan SQL warisan")
			require.NotContains(t, text, "{InputData", "pola penyisipan nilai warisan")
			require.NotContains(t, text, "||'", "perangkaian nilai ke teks SQL")
		})
	}
}

// Disiplin SQL portabel `09-DATABASE-STRATEGY.md` §4 ditegakkan uji, bukan kesepakatan.
//
// Aturan yang hanya ada di dokumen akan dilanggar pada bulan ketiga, ketika tekanan
// jadwal membuat orang menempuh jalan pintas. Yang dipagari uji tidak.
func TestKueriMengikutiDisiplinSQLPortabel(t *testing.T) {
	forbidden := []struct {
		pattern *regexp.Regexp
		reason  string
	}{
		{regexp.MustCompile(`(?i)\bNVL\s*\(`), "pakai COALESCE"},
		{regexp.MustCompile(`(?i)\bSYSDATE\b`), "pakai CURRENT_TIMESTAMP"},
		{regexp.MustCompile(`(?i)\bDECODE\s*\(`), "pakai CASE WHEN"},
		{regexp.MustCompile(`(?i)\bROWNUM\b`), "pakai OFFSET ... FETCH NEXT"},
		{regexp.MustCompile(`(?i)\bINSTR\s*\(`), "pakai POSITION"},
		{regexp.MustCompile(`(?i)\bLISTAGG\s*\(`), "pakai STRING_AGG"},
		{regexp.MustCompile(`\(\+\)`), "pakai LEFT JOIN"},
		{regexp.MustCompile(`(?i)\bSELECT\s+\*`), "sebutkan nama kolom"},
	}

	for _, name := range append(searchQueryNames(), gateQueryNames...) {
		t.Run(name, func(t *testing.T) {
			text := query(name)
			for _, rule := range forbidden {
				require.False(t, rule.pattern.MatchString(text),
					"%s melanggar: %s", name, rule.reason)
			}
		})
	}
}

// TO_CHAR hanya boleh dipakai untuk MEMBANDINGKAN, bukan untuk memformat tampilan.
//
// Pemformatan tanggal dan angka dipindahkan ke Go (`09-DATABASE-STRATEGY.md` §3.2). Satu
// tempat yang tersisa adalah `TO_CHAR(p.INDEXCOUNT)` pada pencarian Tanggal Lahir, yang
// menyamakan tipe kolom saat menggabungkan tabel — persis seperti kueri lama.
func TestToCharHanyaUntukPenyamaanTipe(t *testing.T) {
	for _, name := range searchQueryNames() {
		text := query(name)
		if !strings.Contains(strings.ToUpper(text), "TO_CHAR") {
			continue
		}
		require.Equal(t, "search_birth_date", name,
			"hanya kueri Tanggal Lahir yang boleh memakai TO_CHAR")
		require.Contains(t, strings.ToUpper(text), "TO_CHAR(P.INDEXCOUNT)")
	}
}

// Tidak satu pun kueri modul ini mengubah data milik sistem lama.
//
// Layar View History Claim hanya membaca. Satu-satunya penulisan modul ini adalah
// pencatatan jejak ke tabel MILIK APLIKASI — dan uji berikutnya menjaganya tetap begitu.
func TestKueriPencarianTidakMengubahApaPun(t *testing.T) {
	mutating := regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|MERGE|TRUNCATE)\b`)

	for _, name := range searchQueryNames() {
		t.Run(name, func(t *testing.T) {
			require.False(t, mutating.MatchString(query(name)))
		})
	}
}

// Master proteksi milik sistem lama HANYA DIBACA — `P-1`.
//
// Sistem lama mengurangi jatah dengan `update POOLDATA.MST_PROTEKSI_DATA_PNC`. Menirunya
// berarti dua sistem menulis satu tabel, dan akibatnya bukan galat melainkan jatah yang
// saling menimpa tanpa jejak. Uji ini yang menjaga larangan itu tetap berlaku.
func TestMasterProteksiTidakPernahDitulis(t *testing.T) {
	mutating := regexp.MustCompile(`(?i)\b(INSERT\s+INTO|UPDATE|DELETE\s+FROM|MERGE\s+INTO)\b`)

	for _, name := range append(searchQueryNames(), gateQueryNames...) {
		text := strings.ToUpper(query(name))
		if !strings.Contains(text, "MST_PROTEKSI_DATA_PNC") {
			continue
		}
		require.False(t, mutating.MatchString(text),
			"%s menulis ke master proteksi milik sistem lama", name)
	}
}

// Penulisan modul ini HANYA menyentuh tabel milik aplikasi.
func TestPenulisanHanyaKeTabelAplikasi(t *testing.T) {
	text := strings.ToUpper(query("protection_record_usage"))
	require.Contains(t, text, "INSERT INTO POOLDATA.CPNC_PEMAKAIAN_PROTEKSI")
	require.NotContains(t, text, "MST_PROTEKSI_DATA_PNC")
	require.NotContains(t, text, "LOG_DATA_PROTEKSI_KLAIM")
}

// Jejak audit bersifat append-only — tidak ada UPDATE maupun DELETE terhadapnya (`D-28`).
//
// Hak aksesnya pun dibatasi SELECT dan INSERT di migrasi 0004. Uji ini menjaga sisi
// kodenya, supaya pelanggaran tertangkap sebelum sampai ke basis data yang menolaknya.
func TestJejakAuditTidakPernahDiubahAtauDihapus(t *testing.T) {
	mutating := regexp.MustCompile(`(?i)\b(UPDATE|DELETE\s+FROM)\b`)

	for _, name := range gateQueryNames {
		text := strings.ToUpper(query(name))
		if !strings.Contains(text, "CPNC_PEMAKAIAN_PROTEKSI") {
			continue
		}
		require.False(t, mutating.MatchString(text),
			"%s mengubah atau menghapus jejak audit", name)
	}
}

// Nama tabel di kode dan di migrasi tidak boleh berbeda diam-diam.
func TestNamaTabelSamaDenganMigrasi(t *testing.T) {
	require.Equal(t, "CPNC_PEMAKAIAN_PROTEKSI", TableName)
	require.Contains(t, query("protection_check_table"), TableName)
	require.Contains(t, query("protection_record_usage"), TableName)
}

// Setiap LIKE memakai ESCAPE.
//
// Tanpa itu, satu tanda `%` yang diketik pengguna berubah menjadi pola dan menarik
// seluruh tabel — pada T_CLAIM_PNC yang berpuluh juta baris, itu bukan gangguan kecil.
func TestSetiapLikeMemakaiEscape(t *testing.T) {
	for _, name := range searchQueryNames() {
		text := query(name)
		if !strings.Contains(strings.ToUpper(text), " LIKE ") {
			continue
		}
		require.Contains(t, text, `ESCAPE '\'`, "%s memakai LIKE tanpa ESCAPE", name)
	}
}
