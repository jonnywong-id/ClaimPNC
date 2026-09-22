package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// queryNames adalah seluruh nama kueri yang dipanggil kode modul ini.
var queryNames = []string{
	"report_list",
	"report_count",
	"report_summary",
	"report_get",
	"report_next_sequence",
	"report_insert",
	"report_update",
	"report_check_table",
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Tanpa uji ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya
// — dan bentuknya panik, bukan galat yang rapi.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range queryNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = query(name) })
			require.NotEmpty(t, strings.TrimSpace(query(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = query("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel `09-DATABASE-STRATEGY.md` §4 ditegakkan uji, bukan hanya
// kesepakatan.
//
// Aturan yang hanya ada di dokumen akan dilanggar pada bulan ketiga, ketika tekanan
// jadwal membuat orang menempuh jalan pintas. Yang dipagari uji tidak.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := []struct {
		pattern string
		reason  string
	}{
		{"SELECT *", "kolom baru di basis data tidak boleh diam-diam mengubah perilaku aplikasi"},
		{"NVL(", "pakai COALESCE"},
		{"SYSDATE", "waktu datang dari seam Jam, bukan dari jam basis data"},
		{"ROWNUM", "pakai OFFSET … FETCH NEXT"},
		{"DECODE(", "pakai CASE WHEN"},
		{"INSTR(", "pakai POSITION"},
		{"LISTAGG(", "pakai STRING_AGG"},
		{"TO_CHAR(", "pemformatan tanggal dan angka dikerjakan di Go"},
		{"(+)", "pakai LEFT JOIN"},
	}

	for _, name := range queryNames {
		text := strings.ToUpper(query(name))
		for _, f := range forbidden {
			require.NotContains(t, text, strings.ToUpper(f.pattern),
				"kueri %s memuat %q — %s", name, f.pattern, f.reason)
		}
	}
}

// FROM DUAL hanya boleh ada di kueri urutan.
//
// Ia satu-satunya bentuk khas Oracle yang tidak terhindarkan: NEXTVAL menuntutnya.
// Pengecualiannya dipagari di sini supaya ia tidak menyebar — uji ini GAGAL bila ada
// kueri kedua memakainya, persis seperti pagar yang sama pada modul Master Status Klaim.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	for _, name := range queryNames {
		text := strings.ToUpper(query(name))
		if name == "report_next_sequence" {
			require.Contains(t, text, "FROM DUAL", "kueri urutan memang membutuhkannya")
			continue
		}
		require.NotContains(t, text, "DUAL",
			"kueri %s memakai FROM DUAL; pengecualian dialek hanya untuk kueri urutan", name)
	}
}

// Seluruh nilai lewat parameter binding.
//
// Ini menutup cacat nyata: ketiga kueri inbox sistem lama masing-masing memuat TIGA
// penyisipan `{ASIS:tempQuery.*}` yang merangkai potongan SQL dari nilai properti
// klipboard (`RDB List/ViewTableBrowseRCVInProcess-SQL.xml` dan dua saudaranya).
//
// Yang diuji: setiap kueri yang menerima masukan wajib memuat penanda posisi, dan tidak
// satu pun kueri memuat pola perangkaian warisan.
func TestQueriesUseParameterBinding(t *testing.T) {
	takingInput := []string{
		"report_list", "report_count", "report_summary",
		"report_get", "report_insert", "report_update",
	}
	for _, name := range takingInput {
		require.Contains(t, query(name), ":1",
			"kueri %s menerima masukan tetapi tidak punya penanda posisi", name)
	}

	for _, name := range queryNames {
		require.NotContains(t, query(name), "{ASIS:",
			"kueri %s memuat pola perangkaian warisan", name)
	}
}

// Tidak ada kueri yang menghapus.
//
// `ADR-0012` menetapkan penghapusan lunak menyeluruh, dan laporan yang sudah tertaut
// klaim dirujuk klaimnya lewat `ClaimData.RCV_ID`. Ketiadaan DELETE dipagari di sini
// supaya penambahannya kelak menjadi keputusan sadar, bukan kelalaian yang lolos review.
func TestNoQueryDeletes(t *testing.T) {
	for _, name := range queryNames {
		text := strings.ToUpper(query(name))
		require.NotContains(t, text, "DELETE", "kueri %s menghapus baris", name)
		require.NotContains(t, text, "TRUNCATE", "kueri %s mengosongkan tabel", name)
	}
}

// Seluruh kueri hanya menyentuh tabel milik aplikasi.
//
// `P-1` menetapkan satu tabel ditulis satu sistem. Modul ini tidak menyentuh satu pun
// tabel warisan — tidak `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, dan tidak
// `POOLDATA.T_CLAIM_RECIVEDCLAIM`. Uji ini yang membuktikannya, bukan pembacaan ulang.
func TestQueriesOnlyTouchApplicationTables(t *testing.T) {
	for _, name := range queryNames {
		text := strings.ToUpper(query(name))
		require.NotContains(t, text, "DATAPEGA.",
			"kueri %s menyentuh skema milik engine Pega", name)
		require.NotContains(t, text, "T_CLAIM_RECIVEDCLAIM",
			"kueri %s menyentuh tabel warisan yang sudah ditinggalkan", name)
	}
}

// Ekspresi tahap SAMA PERSIS di ketiga kueri yang memakainya.
//
// Ia satu-satunya definisi tahap di sisi SQL, dan ia harus sejalan dengan
// ClaimReport.Stage di Go. Bila salah satu kueri berbeda, tab akan menampilkan isi yang
// tidak cocok dengan lencananya — dan bedanya tidak menimbulkan galat apa pun.
func TestStageExpressionIdenticalAcrossQueries(t *testing.T) {
	markers := []string{
		"WHEN HASIL_KLAIM = 'DIAKSEPTASI' THEN 'SUDAH_AKSEPTASI'",
		"WHEN HASIL_KLAIM = 'DITOLAK'     THEN 'DITOLAK'",
		"WHEN NOMOR_KLAIM IS NOT NULL     THEN 'SUDAH_REGISTRASI'",
		"WHEN DITRANSFER = 1              THEN 'BELUM_REGISTRASI'",
		"ELSE 'BELUM_TRANSFER'",
	}

	for _, name := range []string{"report_list", "report_count", "report_summary"} {
		text := query(name)
		for _, marker := range markers {
			require.Contains(t, text, marker,
				"kueri %s memakai ekspresi tahap yang berbeda", name)
		}
	}
}

// Penyaring pada kueri daftar dan kueri jumlah harus sama.
//
// Bila keduanya berbeda, layar akan menampilkan jumlah halaman yang tidak pernah ada
// isinya — dan pengguna menekan "berikutnya" ke halaman kosong.
func TestListAndCountShareTheSameFilters(t *testing.T) {
	whereOf := func(name string) string {
		text := query(name)
		start := strings.Index(text, "WHERE")
		require.Positive(t, start, "kueri %s tidak punya klausa WHERE", name)

		rest := text[start:]
		if end := strings.Index(rest, "ORDER BY"); end > 0 {
			rest = rest[:end]
		}
		return strings.Join(strings.Fields(rest), " ")
	}

	require.Equal(t, whereOf("report_count"), whereOf("report_list"),
		"penyaring kueri daftar dan kueri jumlah harus sama persis")
}

// Nama constraint yang dipakai menerjemahkan galat harus sama dengan yang dibuat migrasi.
//
// Mengganti namanya di migrasi tanpa mengganti konstanta ini akan membuat bentrok nomor
// muncul sebagai galat 500 alih-alih pesan yang dapat dibaca pengguna.
func TestPrimaryKeyNameMatchesMigration(t *testing.T) {
	require.Equal(t, "CPNC_LAPORAN_KLAIM_PK", PrimaryKeyName,
		"lihat migrations/0003_pelaporan_klaim.up.sql langkah 1")
}

// Wildcard LIKE yang diketik pengguna dilarikan.
//
// Tanpa ini, mencari "100%" akan cocok dengan SEMUA baris. Pada layar berisi data
// nasabah, penyaring yang diam-diam melebar berarti petugas melihat baris yang tidak ia
// cari.
func TestSearchPatternEscapesWildcards(t *testing.T) {
	require.Equal(t, `100\%`, searchPattern("100%"))
	require.Equal(t, `A\_B`, searchPattern("A_B"))

	// Pelariannya sendiri dilarikan LEBIH DULU, supaya ia tidak dilarikan dua kali.
	require.Equal(t, `a\\b`, searchPattern(`a\b`))

	require.Empty(t, searchPattern("   "), "pencarian berisi spasi saja tidak menyaring apa pun")
}

// Klausa ESCAPE ada pada setiap LIKE.
//
// Melarikan wildcard di Go tanpa menyebut ESCAPE di SQL tidak menolong: backslash-nya
// justru ikut dicari sebagai karakter biasa.
func TestEveryLikeUsesEscape(t *testing.T) {
	for _, name := range []string{"report_list", "report_count", "report_summary"} {
		text := query(name)
		likeCount := strings.Count(strings.ToUpper(text), " LIKE ")
		escapeCount := strings.Count(strings.ToUpper(text), "ESCAPE")
		require.Equal(t, likeCount, escapeCount,
			"kueri %s punya %d LIKE tetapi %d ESCAPE", name, likeCount, escapeCount)
	}
}
