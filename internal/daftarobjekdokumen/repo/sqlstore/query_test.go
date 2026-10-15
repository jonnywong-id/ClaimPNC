package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// allQueryNames adalah seluruh kueri yang dipanggil kode modul ini.
//
// Daftarnya MENGECIL dibanding versi pertama: tiga kueri terhadap
// POOLDATA.LST_DOC_OBJ_BUSINESS dibuang seluruhnya setelah katalog membuktikan tabel itu
// kosong, hanya punya dua kolom, dan tidak dibaca siapa pun — pemetaan yang sesungguhnya
// ada di dalam JSON_DATA.
var allQueryNames = []string{
	"document_object_list",
	"document_object_get",
	"document_object_site",
	"document_object_next_sequence",
	"document_object_insert",
	"document_object_update",
	"business_list",
	"document_object_check_table",
	"business_check_table",
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range allQueryNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Berkas .sql tidak boleh memuat kueri yang sudah tidak dipanggil siapa pun.
//
// Kueri yatim bukan sekadar sampah: ia menyiratkan jalur yang sebenarnya tidak ada lagi.
// Tiga kueri terhadap tabel pemetaan yang dibuang pada perbaikan 2026-10-03 persis seperti
// itu — namanya masih terbaca, padahal tabelnya tidak dipakai sama sekali.
func TestNoOrphanQuery(t *testing.T) {
	used := make(map[string]bool, len(allQueryNames))
	for _, name := range allQueryNames {
		used[name] = true
	}
	for name := range query {
		require.True(t, used[name],
			"kueri %q ada di berkas .sql tetapi tidak dipanggil kode mana pun", name)
	}
}

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji inilah perkakasnya untuk modul ini.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := []struct {
		pattern string
		reason  string
	}{
		{"NVL(", "pakai COALESCE"},
		{"SYSDATE", "pakai CURRENT_TIMESTAMP, atau baca jam lewat seam Clock"},
		{"DECODE(", "pakai CASE WHEN"},
		{"ROWNUM", "pakai OFFSET … FETCH NEXT"},
		{"TO_CHAR(", "pemformatan dikerjakan di Go"},
		{"LPAD(", "pemadatan nol dikerjakan di Go"},
		{"INSTR(", "pakai POSITION"},
		{"LISTAGG(", "pakai STRING_AGG"},
		{"SELECT *", "kolom selalu disebut namanya"},
	}

	for _, name := range allQueryNames {
		text := strings.ToUpper(getQuery(name))
		for _, f := range forbidden {
			require.NotContains(t, text, f.pattern,
				"kueri %q memakai %s — %s", name, f.pattern, f.reason)
		}
	}
}

// Fungsi JSON hanya boleh ada di jalur BACA.
//
// Dokumen JSON yang disimpan disusun di Go, bukan dengan JSON_OBJECT atau JSON_ARRAY:
// keduanya ada di Oracle dan PostgreSQL 17+ dengan sintaks yang berbeda cukup jauh,
// sedangkan `JSON_VALUE` memang portabel (`D-24`).
func TestJSONFunctionsOnlyOnReadPath(t *testing.T) {
	for _, name := range []string{"document_object_insert", "document_object_update"} {
		text := strings.ToUpper(getQuery(name))
		for _, pattern := range []string{"JSON_VALUE", "JSON_OBJECT", "JSON_ARRAY", "JSON_TABLE", "JSON_QUERY"} {
			require.NotContains(t, text, pattern,
				"kueri tulis %q memakai %s; dokumen JSON disusun di Go", name, pattern)
		}
	}

	// Dan jalur bacanya memang memakainya — kalau tidak, nama yang hari ini hanya ada di
	// JSON tidak akan muncul di layar sama sekali.
	require.Contains(t, strings.ToUpper(getQuery("document_object_list")), "JSON_VALUE")
	require.Contains(t, strings.ToUpper(getQuery("document_object_get")), "JSON_VALUE")
}

// Nama dibaca dari KOLOM dan dari JSON sekaligus, dengan kolom didahulukan.
//
// Pada data hari ini KET_DOC_OBJ kosong di seluruh 12 baris dan isinya ada di JSON; baris
// yang ditulis modul ini mengisi keduanya. Membaca salah satu saja membuat salah satu
// kelompok baris tampil tanpa nama.
func TestNameReadFromColumnAndJSON(t *testing.T) {
	for _, name := range []string{"document_object_list", "document_object_get"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "COALESCE(KET_DOC_OBJ, JSON_VALUE(JSON_DATA, '$.KET_DOC_OBJ'))",
			"kueri %q harus membaca kolom DAN dokumen JSON", name)
	}
}

// Penyimpanan menulis KEDUANYA — kolom dan dokumen JSON.
//
// JSON karena di situlah isi yang sesungguhnya dan karena view
// POOLDATA.V_LST_DOC_OBJ_BISNIS membacanya; kolom supaya POOLDATA.V_LST_DOC_OBJ berhenti
// mengembalikan NULL kepada setiap pembacanya.
func TestWritesBothColumnAndJSON(t *testing.T) {
	insert := strings.ToUpper(getQuery("document_object_insert"))
	require.Contains(t, insert, "KET_DOC_OBJ")
	require.Contains(t, insert, "JSON_DATA")

	update := strings.ToUpper(getQuery("document_object_update"))
	require.Contains(t, update, "KET_DOC_OBJ = :1")
	require.Contains(t, update, "JSON_DATA   = :2")
}

// FROM DUAL hanya boleh berada di kueri urutan — NEXTVAL menuntutnya.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	for _, name := range allQueryNames {
		text := strings.ToUpper(getQuery(name))
		if name == "document_object_next_sequence" {
			require.Contains(t, text, "FROM DUAL")
			continue
		}
		require.NotContains(t, text, "FROM DUAL", "kueri %q tidak boleh memakai FROM DUAL", name)
	}
}

// Urutannya LST_DOC_OBJ_SEQ, bukan SET_LST_DOC_OBJ.
//
// Keduanya ada di basis data dan namanya mirip, tetapi yang kedua adalah PROCEDURE penyusun
// senarai bisnis — bukan urutan. Tertukar sekali, dan penyimpanan gagal seluruhnya; itulah
// salah satu sebab versi pertama modul ini tidak dapat menyimpan apa pun.
func TestSequenceIsTheRealOne(t *testing.T) {
	text := strings.ToUpper(getQuery("document_object_next_sequence"))
	require.Contains(t, text, "POOLDATA.LST_DOC_OBJ_SEQ.NEXTVAL")
	require.NotContains(t, text, "SET_LST_DOC_OBJ")
}

// Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
func TestQueriesUseParameterBinding(t *testing.T) {
	needParameter := map[string]int{
		"document_object_get":    1,
		"document_object_insert": 3,
		"document_object_update": 3,
	}

	for name, count := range needParameter {
		text := getQuery(name)
		for i := 1; i <= count; i++ {
			require.Contains(t, text, ":"+string(rune('0'+i)),
				"kueri %q harus memakai parameter :%d", name, i)
		}
		require.NotContains(t, text, "'||", "kueri %q merangkai teks SQL", name)
		require.NotContains(t, text, "||'", "kueri %q merangkai teks SQL", name)
	}
}

// ID tidak pernah ikut di-SET saat pembaruan; ia hanya menyaring.
//
// Ia dirujuk POOLDATA.LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	update := strings.ToUpper(getQuery("document_object_update"))

	setClause := update[strings.Index(update, "SET"):strings.Index(update, "WHERE")]
	require.NotContains(t, setClause, "ID =", "ID tidak boleh ikut di-SET")
	require.Contains(t, update, "WHERE ID = :3")
}

// OLD_ID tidak pernah ditulis: ia jejak sejarah yang diisi POOLDATA.PROCESS_LST_DOC_OBJ saat
// memindahkan data dari GENERAL.LST_DOC_OBJ@ASMD.
func TestOldIDNeverWritten(t *testing.T) {
	for _, name := range []string{"document_object_insert", "document_object_update"} {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "OLD_ID",
			"kueri %q tidak boleh menulis OLD_ID", name)
	}
}

// TIDAK ADA penghapusan fisik di mana pun (`D-66`).
func TestNoPhysicalDeleteAnywhere(t *testing.T) {
	for _, name := range allQueryNames {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "DELETE",
			"kueri %q memakai DELETE; D-66 melarang penghapusan fisik", name)
	}
}

// POOLDATA.LST_DOC_OBJ_BUSINESS TIDAK disentuh sama sekali.
//
// Tabelnya ada dan kosong, kolomnya hanya dua, dan tidak ada satu pun pembaca. Menulisnya
// berarti membuat sumber kebenaran KEDUA yang hasilnya tidak pernah dilihat siapa pun —
// sedangkan pemetaan yang benar-benar dibaca ada di dalam JSON_DATA.
func TestMappingTableNeverTouched(t *testing.T) {
	for _, name := range allQueryNames {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "LST_DOC_OBJ_BUSINESS",
			"kueri %q menyentuh tabel pemetaan yang tidak dipakai", name)
	}
}

// POOLDATA.BUSINESS TIDAK PERNAH ditulis: ia milik GISFW (`D-03`).
func TestBusinessTableIsNeverWritten(t *testing.T) {
	for _, name := range allQueryNames {
		text := strings.ToUpper(getQuery(name))
		if !strings.Contains(text, "POOLDATA.BUSINESS") {
			continue
		}
		for _, write := range []string{"INSERT", "UPDATE", "MERGE"} {
			require.NotContains(t, text, write,
				"kueri %q menulis POOLDATA.BUSINESS milik GISFW", name)
		}
	}
}

// Baca dan tulis sama-sama ke TABEL DASAR, bukan ke view.
//
// Ini mengoreksi versi pertama, yang membaca lewat POOLDATA.V_LST_DOC_OBJ. View itu tidak
// memuat JSON_DATA, sedangkan di situlah isi yang sesungguhnya — membacanya berarti
// menampilkan 12 baris tanpa nama dan tanpa satu pun pemetaan bisnis.
func TestReadsAndWritesTheBaseTable(t *testing.T) {
	for _, name := range allQueryNames {
		text := strings.ToUpper(getQuery(name))
		require.NotContains(t, text, "V_LST_DOC_OBJ",
			"kueri %q membaca view; JSON_DATA hanya ada di tabel dasarnya", name)
	}
	require.Contains(t, strings.ToUpper(getQuery("document_object_list")), "POOLDATA.LST_DOC_OBJ")
}

// Kueri pemeriksaan tidak mengambil satu baris pun, sehingga aman dijalankan terhadap
// produksi.
func TestCheckTableFetchesNoRows(t *testing.T) {
	for _, name := range []string{"document_object_check_table", "business_check_table"} {
		require.Contains(t, getQuery(name), "WHERE 1 = 0", "kueri %q harus tidak mengambil baris", name)
	}
}

// Pemeriksaan induk menyentuh JSON_DATA juga — di situlah isinya tinggal, dan hak baca
// atasnya harus ikut terbukti sebelum layar dibuka pengguna pertama.
func TestCheckTableIncludesJSONColumn(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("document_object_check_table")), "JSON_DATA")
}

// FiveDigits meniru lpad(to_char(seq), 5, '0') pada POOLDATA.PEGA_LST_DOC_OBJ, TERMASUK
// perilakunya di atas 99999: dikembalikan apa adanya, tidak dipotong.
//
// Memotongnya akan menghasilkan ID GANDA, yang jauh lebih buruk daripada penyisipan yang
// gagal dengan pesan jelas.
func TestFiveDigitsMirrorsOracleLPAD(t *testing.T) {
	require.Equal(t, "00001", FiveDigits(1))
	require.Equal(t, "00778", FiveDigits(778), "nilai urutan yang sebenarnya hari ini")
	require.Equal(t, "99999", FiveDigits(99999))
	require.Equal(t, "100000", FiveDigits(100000), "di atas 99999 tidak dipotong, sama seperti LPAD")
}

// Bentuk ID yang dihasilkan harus sama persis dengan yang sudah ada di produksi: kode situs
// "1" ditambah lima digit, enam karakter.
func TestIDShapeMatchesProduction(t *testing.T) {
	require.Equal(t, "100766", "1"+FiveDigits(766))
	require.Equal(t, "100778", "1"+FiveDigits(778))
	require.Len(t, "1"+FiveDigits(778), 6, "kolom ID berupa CHAR(6)")
}
