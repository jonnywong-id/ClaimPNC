package sqlstore

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/detailpenyebab"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql.
//
// Nama kueri adalah konstanta di dalam kode, dan ketiadaannya baru terlihat saat kueri itu
// pertama dijalankan — yang bisa jadi berbulan-bulan kemudian pada jalur yang jarang
// dilewati. Uji ini memindahkan kegagalannya ke waktu build.
func TestEveryUsedQueryExists(t *testing.T) {
	for _, name := range []string{
		"detail_list",
		"detail_get",
		"detail_business_list",
		"detail_document",
		"detail_insert",
		"detail_update",
		"detail_exists",
		"detail_site",
		"detail_next_sequence",
		"detail_check_table",
		"detail_check_writable",
		"detail_check_business_view",
		"master_search",
		"master_get",
		"master_check_table",
		"business_search",
		"business_check_table",
	} {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

// MENULIS ke tabel, MEMBACA dari view — dan keduanya tidak boleh tertukar.
//
// Ini invarian terpenting modul ini. `POOLDATA.V_D_CAUSE_OF_LOSS` adalah view berkolom
// yang membentangkan `D_CAUSE_OF_LOSS.JSONDATA`, dan hampir pasti tidak dapat ditulis
// langsung. Satu INSERT atau UPDATE yang keliru menyebut view akan gagal di produksi —
// bukan saat build, dan bukan saat uji yang tidak menyentuh Oracle.
func TestWriteStatementsNeverTargetAView(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, "INSERT") && !strings.Contains(upperCase, "UPDATE ") {
			continue
		}
		require.NotContainsf(t, upperCase, "V_D_CAUSE_OF_LOSS",
			"kueri tulis %q menyebut VIEW; ia harus menyebut tabel POOLDATA.D_CAUSE_OF_LOSS", name)
		require.Containsf(t, upperCase, "POOLDATA.D_CAUSE_OF_LOSS",
			"kueri tulis %q harus menyentuh tabel POOLDATA.D_CAUSE_OF_LOSS", name)
	}
}

// Hanya TABEL detail yang boleh ditulis; seluruh objek lain hanya dibaca.
//
// Kewenangannya berbeda: POOLDATA.D_CAUSE_OF_LOSS ditulis modul ini, sedangkan
// V_M_CAUSE_OF_LOSS, V_D_CAUSE_OF_LOSS_BUSINESS, BUSINESS, dan M_SITE_DATABASE milik
// sistem lain (ADR-0004, penulis tunggal per tabel). Satu kueri yang keliru menulis ke
// sana melanggar kewenangan itu tanpa satu pun galat basis data.
func TestOnlyTheDetailTableIsEverWritten(t *testing.T) {
	readOnly := []string{
		"POOLDATA.V_M_CAUSE_OF_LOSS",
		"POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS",
		"POOLDATA.BUSINESS",
		"POOLDATA.M_SITE_DATABASE",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(strings.TrimSpace(text))
		if !strings.HasPrefix(upperCase, "INSERT") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}
		for _, object := range readOnly {
			require.NotContainsf(t, upperCase, object,
				"kueri tulis %q menyentuh %s, yang hanya boleh DIBACA", name, object)
		}
	}
}

// TIDAK ADA DELETE di seluruh modul ini.
//
// Layar lamanya tidak punya tombolnya, `pyDeleteSQL` pada rule simpannya kosong, dan
// `D-66` melarang penghapusan fisik data bernilai bisnis. Baris yang tidak lagi dipakai
// dinyatakan lewat STS_AKTIF.
func TestNoDeleteStatementExists(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; modul ini tidak menghapus baris (D-66)", name)
	}
}

// Pola SQL yang dilarang Coding Standards tidak boleh ada.
//
// Keempatnya menghalangi D-20 — satu set SQL yang berjalan sama di Oracle 19c dan
// PostgreSQL 17+. Pemeriksaannya di sini, bukan hanya di CI, supaya kegagalannya menyebut
// kueri MANA yang bermasalah.
func TestNoForbiddenDialectPattern(t *testing.T) {
	forbidden := []string{"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "TO_CHAR(", "LISTAGG(", "INSTR("}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memuat %q yang dilarang D-20", name, pattern)
		}
	}
}

// SELECT * dilarang: kolom baru di basis data tidak boleh diam-diam mengubah perilaku.
func TestNoSelectStar(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "SELECT *",
			"kueri %q memakai SELECT *; sebutkan nama kolomnya", name)
	}
}

// FROM DUAL hanya boleh ada di SATU kueri — pengambil nomor urut.
//
// Ia satu-satunya bentuk khas Oracle di modul ini dan tidak terhindarkan: NEXTVAL
// menuntutnya. Mengisolasinya di satu kueri membuat perpindahan ke PostgreSQL menyentuh
// satu tempat, persis seperti generator nomor klaim pada ADR-0005.
func TestFromDualIsIsolatedToTheSequenceQuery(t *testing.T) {
	for name, text := range query {
		if strings.Contains(strings.ToUpper(text), "FROM DUAL") {
			require.Equalf(t, "detail_next_sequence", name,
				"FROM DUAL hanya boleh di detail_next_sequence, tetapi ada di %q", name)
		}
	}
}

// Batas hasil pencarian di SQL harus sama dengan batas di domain.
//
// Keduanya mudah berpisah diam-diam: yang satu angka di dalam teks SQL, yang satu konstanta
// Go. Bila SQL memotong lebih dulu, batas domain menjadi tidak berarti; bila domain
// memotong lebih dulu, basis data menarik baris yang pasti dibuang.
func TestLookupRowLimitMatchesTheDomain(t *testing.T) {
	want := "FETCH NEXT " + strconv.Itoa(detailpenyebab.MaxLookupRows) + " ROWS ONLY"

	for _, name := range []string{"master_search", "business_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), want,
			"batas baris pada kueri %q harus sama dengan detailpenyebab.MaxLookupRows", name)
	}
}

// Setiap pencarian LIKE wajib menyebut ESCAPE.
//
// Oracle TIDAK punya karakter pelolos bawaan pada LIKE; PostgreSQL memakai backslash
// sebagai bawaan. Menyebutkannya membuat keduanya berperilaku sama (D-20) — dan tanpa itu,
// kata kunci yang memuat `%` atau `_` akan mencocokkan hal yang tidak diminta siapa pun.
func TestEveryLikeDeclaresItsEscapeCharacter(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, "LIKE ") {
			continue
		}
		require.Containsf(t, upperCase, "ESCAPE",
			"kueri %q memakai LIKE tanpa ESCAPE", name)
	}
}

// Dokumen JSON yang disusun harus memakai kunci yang dikenali view.
//
// Kunci-kunci itu adalah nama properti page `TempDcol`, dan view yang membentangkannya
// membacanya berdasarkan nama. Satu huruf yang berbeda membuat kolom view kosong TANPA
// satu pun galat — kegagalan yang paling mahal ditemukan, karena ia tampak seperti data
// yang memang belum diisi.
func TestNewDocumentUsesTheKeysTheViewExpects(t *testing.T) {
	payload, err := json.Marshal(newDocument(detailpenyebab.CauseOfLossDetail{
		ID:          "100001",
		LegacyID:    "COL-1",
		MasterID:    "9001",
		Description: "Kebakaran",
		LossCode:    "FIRE-01",
		Active:      detailpenyebab.ActiveYes,
		Business:    []detailpenyebab.Business{{ID: "006", Name: "Fire / Property"}},
	}))
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(payload, &fields))

	for _, key := range []string{
		"D_COL_ID", "OLD_D_COL_ID", "M_COL_ID", "DESCRIPTION", "LOSS_CODE", "STS_AKTIF",
		"BISNISID",
	} {
		require.Containsf(t, fields, key, "dokumen JSON kehilangan kunci %q", key)
	}
}

// Daftar lini bisnis memakai kunci `ID` dan `Note`, bukan `NOTE` maupun `note`.
//
// Asalnya `Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1784-1785`, yang mengisi
// `TempDcol.BISNISID(<LAST>).Note`. Jalur JSON bersifat case-sensitive.
func TestBusinessLinesUseTheExactKeyCasing(t *testing.T) {
	payload, err := json.Marshal(newDocument(detailpenyebab.CauseOfLossDetail{
		Business: []detailpenyebab.Business{{ID: "006", Name: "Fire / Property"}},
	}))
	require.NoError(t, err)
	require.Contains(t, string(payload), `"ID":"006"`)
	require.Contains(t, string(payload), `"Note":"Fire / Property"`)
}

// Daftar lini bisnis kosong ditulis sebagai array kosong, bukan null.
func TestEmptyBusinessListIsWrittenAsAnEmptyArray(t *testing.T) {
	payload, err := json.Marshal(newDocument(detailpenyebab.CauseOfLossDetail{}))
	require.NoError(t, err)
	require.Contains(t, string(payload), `"BISNISID":[]`)
}

// Menyimpan perubahan TIDAK boleh membuang kunci yang tidak dikenal modul ini.
//
// Dokumen dapat memuat kunci yang tidak dibentangkan view mana pun — `TempDcol` terbukti
// juga menampung `pyNote` dan `pyLabel`. Menyusun dokumen baru hanya dari kolom yang
// dikenal akan membuangnya diam-diam pada setiap penyimpanan.
func TestMergeKeepsUnknownKeysFromTheStoredDocument(t *testing.T) {
	stored := `{"D_COL_ID":"100001","DESCRIPTION":"lama","pyLabel":"Close","KunciAsing":"jangan hilang"}`

	merged, err := mergeDocument(stored, detailpenyebab.CauseOfLossDetail{
		ID:          "100001",
		Description: "baru",
	})
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal([]byte(merged), &fields))

	require.Equal(t, "baru", fields["DESCRIPTION"], "kunci yang disunting harus tertimpa")
	require.Equal(t, "Close", fields["pyLabel"], "kunci Pega harus tetap ada")
	require.Equal(t, "jangan hilang", fields["KunciAsing"], "kunci asing harus tetap ada")
}

// Dokumen tersimpan yang rusak tidak boleh mengunci barisnya.
//
// Baris yang JSONDATA-nya tidak dapat diurai tetap harus dapat diperbaiki lewat layar;
// menolaknya justru menutup satu-satunya jalan memperbaikinya.
func TestBrokenStoredDocumentIsRebuiltInsteadOfRejected(t *testing.T) {
	merged, err := mergeDocument("{bukan json", detailpenyebab.CauseOfLossDetail{
		ID:          "100001",
		Description: "baru",
	})
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal([]byte(merged), &fields))
	require.Equal(t, "100001", fields["D_COL_ID"])
	require.Equal(t, "baru", fields["DESCRIPTION"])
}

// Nilai JSON bertipe angka harus terbaca sebagai teks.
//
// Baris lama dapat memuat `"STS_AKTIF": 1` — angka, bukan teks — dan tanpa penerima yang
// memaafkan, satu baris semacam itu membuat SELURUH daftar gagal dibaca.
func TestNumericJSONValuesAreReadAsText(t *testing.T) {
	var doc document
	require.NoError(t, json.Unmarshal(
		[]byte(`{"STS_AKTIF":1,"M_COL_ID":9001,"DESCRIPTION":null}`), &doc))

	require.Equal(t, jsonText("1"), doc.Active)
	require.Equal(t, jsonText("9001"), doc.MasterID)
	require.Equal(t, jsonText(""), doc.Description, "null harus menjadi teks kosong")
}

// Karakter khas LIKE di dalam kata kunci harus dilolosi.
//
// Tanpa itu, petugas yang mengetik `_` akan menerima hasil yang cocok dengan karakter apa
// pun — dan tidak ada apa pun di layar yang menjelaskan kenapa.
func TestLikeWildcardsInsideTheKeywordAreEscaped(t *testing.T) {
	require.Equal(t, `%100\_%`, likePattern("100_"))
	require.Equal(t, `%50\%%`, likePattern("50%"))
	require.Equal(t, "%", likePattern("   "), "kata kunci kosong menjadi pola apa saja")
}

// Penyaring kosong dikirim sebagai NULL, bukan teks kosong.
//
// Kueri memeriksanya dengan `:n IS NULL`, dan teks kosong BUKAN NULL — mengirimnya apa
// adanya akan membuat penyaring kosong menyaring sungguhan dan mengosongkan daftar.
func TestEmptyFilterIsSentAsNull(t *testing.T) {
	require.Nil(t, nullable(""))
	require.Nil(t, nullable("   "))
	require.Equal(t, "9001", nullable("  9001  "))
}
