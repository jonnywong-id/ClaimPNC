package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// allQueryNames adalah seluruh kueri yang dipanggil kode modul ini.
var allQueryNames = []string{
	"document_object_list",
	"document_object_get",
	"document_object_business_list",
	"document_object_site",
	"document_object_next_sequence",
	"document_object_insert",
	"document_object_update",
	"document_object_business_deactivate",
	"document_object_business_activate",
	"document_object_business_insert",
	"business_list",
	"document_object_check_table",
	"document_object_write_check_table",
	"document_object_business_check_table",
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

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji inilah perkakasnya untuk modul ini.
//
// Kata terlarangnya diambil dari `09-DATABASE-STRATEGY.md` §4. Yang dicari adalah pemakaian
// sebagai FUNGSI atau kata kunci, bukan sebagai bagian nama kolom — karena itu setiap pola
// diperiksa bersama tanda kurung atau spasi di belakangnya.
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

// FROM DUAL adalah satu-satunya bentuk khas Oracle yang dibiarkan, dan ia HANYA boleh berada
// di kueri urutan — NEXTVAL menuntutnya.
//
// Ia sengaja diisolasi di satu kueri, perlakuannya sama dengan generator nomor klaim pada
// `ADR-0005`. Uji ini yang menjaga isolasi itu tidak bocor ke kueri lain.
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

// Nilai selalu lewat parameter binding, tidak pernah dirangkai ke teks SQL.
//
// Inilah yang menutup celah pola `{ASIS:…}` warisan — 538 kemunculannya di sistem lama adalah
// risiko SQL injection sekaligus penghalang portabilitas (`11-SECURITY.md` §5).
func TestQueriesUseParameterBinding(t *testing.T) {
	needParameter := map[string]int{
		"document_object_get":                 1,
		"document_object_business_list":       1,
		"document_object_insert":              2,
		"document_object_update":              2,
		"document_object_business_deactivate": 1,
		"document_object_business_activate":   4,
		"document_object_business_insert":     4,
	}

	for name, count := range needParameter {
		text := getQuery(name)
		for i := 1; i <= count; i++ {
			require.Contains(t, text, ":"+itoa(i), "kueri %q harus memakai parameter :%d", name, i)
		}
		// Tanda kutip tunggal hanya boleh muncul pada penanda tetap seperti '1' dan '0',
		// tidak pernah membungkus nilai yang berasal dari pengguna.
		require.NotContains(t, text, "'||", "kueri %q merangkai teks SQL", name)
		require.NotContains(t, text, "||'", "kueri %q merangkai teks SQL", name)
	}
}

// ID tidak pernah ikut di-SET saat pembaruan; ia hanya menyaring.
//
// Ia dirujuk LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID pada data yang sudah berjalan, sehingga
// mengubahnya akan memutus setiap baris yang bernaung di bawahnya.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	update := strings.ToUpper(getQuery("document_object_update"))

	setClause := update[strings.Index(update, "SET"):strings.Index(update, "WHERE")]
	require.NotContains(t, setClause, "ID =", "ID tidak boleh ikut di-SET")
	require.Contains(t, update, "WHERE ID = :2")
}

// OLD_ID tidak pernah ditulis: ia jejak sejarah yang ditulis sebelum sistem ini ada.
func TestOldIDNeverWritten(t *testing.T) {
	for _, name := range []string{"document_object_insert", "document_object_update"} {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "OLD_ID",
			"kueri %q tidak boleh menulis OLD_ID", name)
	}
}

// TIDAK ADA penghapusan fisik di mana pun (`D-66`).
//
// Pencabutan pemetaan bisnis dikerjakan dengan PENANDAAN — STS_AKTIF menjadi '0' — bukan
// dengan DELETE. Uji ini yang menjaga keputusan itu tidak diam-diam dibalik saat seseorang
// merasa DELETE "lebih sederhana".
func TestNoPhysicalDeleteAnywhere(t *testing.T) {
	for _, name := range allQueryNames {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "DELETE",
			"kueri %q memakai DELETE; D-66 melarang penghapusan fisik", name)
	}

	deactivate := strings.ToUpper(getQuery("document_object_business_deactivate"))
	require.Contains(t, deactivate, "SET STS_AKTIF = '0'")
}

// Pembaca hanya mengambil pemetaan yang AKTIF. Tanpa saringan itu, bisnis yang sudah dicabut
// pengguna akan muncul kembali di form — dan pencabutannya tampak gagal.
func TestBusinessReaderFiltersActiveOnly(t *testing.T) {
	list := strings.ToUpper(getQuery("document_object_business_list"))
	require.Contains(t, list, "STS_AKTIF  = '1'")
}

// Urutan pemetaan bisnis mengikuti URUTAN, bukan nama.
//
// Susunan baris di grid adalah susunan yang disimpan pengguna. Mengurutkannya menurut nama
// akan membuat layar menampilkan urutan yang berbeda dari yang baru saja ia simpan — persis
// kekurangan yang masih dimiliki modul Master COL Simas Online karena tabelnya tidak punya
// kolom urutan.
func TestBusinessOrderFollowsPosition(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("document_object_business_list")), "ORDER BY URUTAN")
}

// Upsert pemetaan bisnis berkunci POSISI, bukan nama.
//
// Dengan begitu dua baris bernama sama tetap dua baris — dan itu sah, karena grid Pega tidak
// punya satu pun penanda keunikan.
func TestBusinessUpsertMatchesByPosition(t *testing.T) {
	activate := strings.ToUpper(getQuery("document_object_business_activate"))
	require.Contains(t, activate, "WHERE ID_DOC_OBJ = :3")
	require.Contains(t, activate, "AND URUTAN     = :4")
}

// Pembaca pemetaan TIDAK menjoin POOLDATA.BUSINESS.
//
// Menjoin akan MEMBUANG tepat baris yang namanya diketik bebas — baris yang BISNISID-nya
// memang tidak ada di master, dan yang justru wajib tetap terbaca.
func TestBusinessReaderDoesNotJoinTheGISFWTable(t *testing.T) {
	require.NotContains(t, strings.ToUpper(getQuery("document_object_business_list")), "BUSINESS ")
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

// Pembacaan lewat VIEW, penulisan ke TABEL DASAR.
//
// Yang dibaca adalah objek yang namanya pasti (terbukti dari Report Definition); yang ditulis
// adalah tabel dasarnya, yang namanya dugaan. Uji ini membuat pembagian itu terlihat, sehingga
// menukarnya menjadi keputusan dan bukan kekeliruan yang menyelinap.
func TestReadsViewWritesBaseTable(t *testing.T) {
	for _, name := range []string{"document_object_list", "document_object_get", "document_object_check_table"} {
		require.Contains(t, strings.ToUpper(getQuery(name)), "POOLDATA.V_LST_DOC_OBJ",
			"kueri baca %q harus lewat view", name)
	}
	for _, name := range []string{"document_object_insert", "document_object_update"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "POOLDATA.LST_DOC_OBJ", "kueri tulis %q harus ke tabel dasar", name)
		require.NotContains(t, text, "POOLDATA.V_LST_DOC_OBJ", "kueri tulis %q tidak boleh lewat view", name)
	}
}

// Kueri pemeriksaan tidak mengambil satu baris pun, sehingga aman dijalankan terhadap
// produksi.
func TestCheckTableFetchesNoRows(t *testing.T) {
	for _, name := range []string{
		"document_object_check_table",
		"document_object_write_check_table",
		"document_object_business_check_table",
		"business_check_table",
	} {
		require.Contains(t, getQuery(name), "WHERE 1 = 0", "kueri %q harus tidak mengambil baris", name)
	}
}

// FourDigits meniru lpad(to_char(seq), 4, '0') pada procedure lama, TERMASUK perilakunya di
// atas 9999: dikembalikan apa adanya, tidak dipotong.
//
// Memotongnya akan menghasilkan ID GANDA, yang jauh lebih buruk daripada penyisipan yang
// gagal dengan pesan jelas.
func TestFourDigitsMirrorsOracleLPAD(t *testing.T) {
	require.Equal(t, "0001", FourDigits(1))
	require.Equal(t, "0042", FourDigits(42))
	require.Equal(t, "9999", FourDigits(9999))
	require.Equal(t, "10000", FourDigits(10000), "di atas 9999 tidak dipotong, sama seperti LPAD")
}

// Isian opsional yang kosong disimpan sebagai NULL, bukan teks kosong.
//
// Keduanya berbeda di basis data, dan membiarkan keduanya masuk berarti dua bentuk "tidak
// diisi" yang harus sama-sama diingat setiap kueri sesudahnya.
func TestEmptyOptionalValueBecomesNull(t *testing.T) {
	require.Nil(t, nullIfEmpty(""))
	require.Nil(t, nullIfEmpty("   "))
	require.Equal(t, "003", nullIfEmpty("003"))
}

// itoa kecil untuk uji parameter binding, supaya berkas uji tidak menarik strconv hanya untuk
// satu pemakaian.
func itoa(n int) string {
	return string(rune('0' + n))
}
