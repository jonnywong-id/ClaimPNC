package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersparepart"
)

// sampleSparepart adalah baris yang dipakai menghitung jumlah argumen.
//
// Isinya tidak diperiksa satu pun uji di berkas ini — yang diuji adalah JUMLAH argumen yang
// dihasilkan insertArguments dan updateArguments, bukan nilainya. Ia sengaja hampir kosong
// supaya tidak ada yang tergoda menambahkan pemeriksaan nilai ke uji yang menjaga bentuk.
func sampleSparepart() mastersparepart.Sparepart {
	return mastersparepart.Sparepart{ID: "SP0000000001", Name: "Sparepart Contoh"}
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"sparepart_list",
		"sparepart_list_search",
		"sparepart_get",
		"sparepart_find_by_name",
		"sparepart_find_by_number",
		"sparepart_find_by_code",
		"sparepart_lock_by_keys",
		"sparepart_insert",
		"sparepart_update",
		"sparepart_set_status",
		"sparepart_category_list",
		"sparepart_type_list",
		"sparepart_count_by_status",
		"sparepart_count_all",
		"sparepart_check_table",
		"sparepart_check_category_table",
		"sparepart_check_type_table",
		"sparepart_check_json_mirror",
		"sparepart_count_json_mirror",
		"sparepart_count_orphan_category",
		"sparepart_count_orphan_type",
		"sparepart_site",
		"sparepart_next_sequence",
	}

	for _, name := range usedNames {
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
	require.Lenf(t, query, len(usedNames),
		"ada kueri di berkas .sql yang tidak dipakai kode, atau sebaliknya")
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { getQuery("sparepart_tidak_ada") })
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *": "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":     "pakai COALESCE",
		"SYSDATE":  "pakai CURRENT_TIMESTAMP",
		"DECODE(":  "pakai CASE WHEN",
		"ROWNUM":   "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":   "pakai POSITION",
		"LISTAGG(": "pakai STRING_AGG",
		"TO_CHAR(": "pemformatan tanggal dan angka dilakukan di Go",
		"VARCHAR2": "VARCHAR sudah diterima Oracle maupun PostgreSQL",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai sequence yang
// sama dengan procedure lama adalah syarat agar ID yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan ID yang pernah diterbitkan Pega.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "sparepart_next_sequence"

	for name, text := range query {
		if name == exempted {
			require.Contains(t, strings.ToUpper(text), "FROM DUAL",
				"kueri urutan memang harus memakainya; bila tidak lagi, hapus pengecualiannya")
			continue
		}
		require.NotContainsf(t, strings.ToUpper(text), "FROM DUAL",
			"kueri %q memakai FROM DUAL; hanya %q yang dibenarkan", name, exempted)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL adalah
// celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan.
func TestQueriesUseParameterBinding(t *testing.T) {
	for _, name := range []string{
		"sparepart_list",
		"sparepart_list_search",
		"sparepart_get",
		"sparepart_find_by_name",
		"sparepart_find_by_number",
		"sparepart_find_by_code",
		"sparepart_lock_by_keys",
		"sparepart_insert",
		"sparepart_update",
		"sparepart_set_status",
		"sparepart_category_list",
		"sparepart_type_list",
		"sparepart_count_by_status",
	} {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	// Pencarian mengikat kata kuncinya TIGA KALI — satu untuk tiap kunci alami. Lihat
	// catatan pada sparepart_list_search.
	text := getQuery("sparepart_list_search")
	for _, marker := range []string{":2", ":3", ":4"} {
		require.Containsf(t, text, marker,
			"kata kunci pencarian harus terikat pada %s, bukan dirangkai ke teks SQL", marker)
	}
}

// TIDAK ADA DELETE sama sekali di modul ini.
//
// `D-66` menetapkan soft delete menyeluruh, dan berbeda dari Master Panel — yang tabel
// anaknya menuntut hapus-lalu-sisip-ulang — modul ini tidak punya tabel anak. Tidak ada
// alasan apa pun untuk sebuah DELETE di sini, dan uji ini memastikannya tetap begitu.
func TestNoDeleteAnywhere(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; D-66 melarang penghapusan fisik data bernilai bisnis", name)
	}
}

// Hanya SATU tabel yang ditulis: POOLDATA.SPAREPART_HE.
//
// Kedua tabel acuan dan tabel JSON milik Pega HANYA DIBACA (ADR-0004, penulis tunggal per
// tabel). Satu INSERT atau UPDATE yang menyentuh salah satunya berarti aplikasi ini menulis
// ke tabel yang pemiliknya sistem lain.
func TestOnlyTheOwnedTableIsWritten(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, "INSERT INTO") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}
		require.Containsf(t, upperCase, "POOLDATA.SPAREPART_HE",
			"kueri tulis %q harus menyentuh POOLDATA.SPAREPART_HE", name)
		for _, readOnly := range []string{
			"GCNM_M_SPAREPART_CATEGORY",
			"GCNM_M_SPAREPART_TYPE",
			"M_SPAREPART_HE_BU",
			"M_SITE_DATABASE",
		} {
			require.NotContainsf(t, upperCase, readOnly,
				"kueri tulis %q menyentuh %s yang HANYA BOLEH DIBACA", name, readOnly)
		}
	}
}

// Keenam kueri pembaca menyebut kolom pada urutan yang SAMA.
//
// Satu fungsi scanRow membaca keenamnya berdasarkan POSISI, dan satu kolom yang bergeser
// akan menaruh harga jual ke kolom berat tanpa satu pun galat.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	reference := selectedColumns(t, getQuery("sparepart_list"))
	require.Len(t, reference, 24, "kedua puluh empat kolom harus dibaca")

	for _, name := range []string{
		"sparepart_list_search",
		"sparepart_get",
		"sparepart_find_by_name",
		"sparepart_find_by_number",
		"sparepart_find_by_code",
		"sparepart_check_table",
	} {
		require.Equalf(t, reference, selectedColumns(t, getQuery(name)),
			"kueri %q membaca kolom pada urutan yang berbeda dari sparepart_list", name)
	}
}

// Kedua kolom tanggal ikut dibaca kueri pemeriksaan.
//
// Justru keduanya yang asumsi tipenya perlu dibuktikan sebelum jalur tulis dipakai; lihat
// banner mastersparepart.sql.
func TestCheckQueryIncludesBothDateColumns(t *testing.T) {
	text := strings.ToUpper(getQuery("sparepart_check_table"))
	require.Contains(t, text, "PROD_DATE")
	require.Contains(t, text, "TGL_UPDATE_HARGA")
	require.Contains(t, text, "1 = 0", "kueri pemeriksaan tidak boleh mengambil satu baris pun")
}

func TestInsertColumnCountMatchesArguments(t *testing.T) {
	text := getQuery("sparepart_insert")

	open := strings.Index(text, "(")
	closing := strings.Index(text, ")")
	require.Greater(t, closing, open)

	columns := strings.Split(text[open+1:closing], ",")
	require.Len(t, insertArguments(sampleSparepart()), len(columns),
		"jumlah argumen tidak sama dengan jumlah kolom pada sparepart_insert")
	require.Len(t, columns, 24)
}

func TestUpdateArgumentCountMatchesQuery(t *testing.T) {
	text := getQuery("sparepart_update")

	// Dua puluh empat argumen: dua puluh tiga kolom yang ditulis, ditambah ID sebagai
	// penyaring WHERE.
	require.Len(t, updateArguments(sampleSparepart()), 24)
	require.Contains(t, text, ":24", "ID harus menjadi parameter terakhir")
}

// Kunci baris TIDAK PERNAH berpindah: sparepart_update tidak boleh menyebut ID sebagai
// kolom yang ditulis.
func TestUpdateNeverMovesTheKey(t *testing.T) {
	text := strings.ToUpper(getQuery("sparepart_update"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	// Dicari sebagai "ID =" dengan spasi, supaya DOKUMENID dan KATEGORI_SPART tidak ikut
	// tertangkap hanya karena namanya memuat huruf yang sama.
	require.NotContains(t, setClause, " ID =", "sparepart_update tidak boleh menulis ID")
}

// Keputusan borongan hanya menyentuh APPROVAL.
//
// Tanpa uji ini, satu kolom yang ikut tertulis pada jalur keputusan akan menimpa isian yang
// diketik petugas lain — tanpa satu pun tanda di layar. USER_UPDATE khususnya: ia harus
// tetap berisi siapa yang MENGAJUKAN, meniru `Activity/SetApprovalAllMaster`.
func TestDecisionQueryTouchesOnlyApproval(t *testing.T) {
	text := strings.ToUpper(getQuery("sparepart_set_status"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.Contains(t, setClause, "APPROVAL")
	for _, column := range []string{
		"USER_UPDATE", "TGL_UPDATE_HARGA", "NAMA_SPART", "NO_SPART", "HARGA_JUAL",
	} {
		require.NotContainsf(t, setClause, column,
			"jalur keputusan tidak boleh menulis %s", column)
	}

	require.Contains(t, text, "<>",
		"baris yang sudah berstatus itu harus dikecualikan supaya jumlah berubah bermakna")
}

// Kedua kueri acuan menyaring status persetujuan lewat PARAMETER, bukan literal.
//
// Nilainya '1' — meniru `Activity/BrowseTipeKategoriPart` — tetapi ia datang dari konstanta
// Go, bukan dirangkai ke teks SQL. Itu yang membuatnya dapat diubah di satu tempat bila
// kelak Work Owner memutuskan lain.
func TestLookupQueriesFilterApprovalByParameter(t *testing.T) {
	for _, name := range []string{"sparepart_category_list", "sparepart_type_list"} {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "APPROVAL", "kueri %q harus menyaring APPROVAL", name)
		require.Containsf(t, text, ":1", "penyaring pada %q harus terikat sebagai parameter", name)
		require.NotContainsf(t, text, "= '1'",
			"kueri %q tidak boleh merangkai nilai penyaringnya ke teks SQL", name)
	}

	// Kueri tipe menyebut kategori induknya; layar memakainya untuk mempersempit daftar.
	require.Contains(t, strings.ToUpper(getQuery("sparepart_type_list")), "PART_CATEGORY_ID")
}

// Penyaring APPROVAL kedua lookup memakai nilai "sudah disetujui".
//
// Ia konstanta Go, bukan literal SQL; uji ini yang menjaga nilainya tidak bergeser tanpa
// disadari. Lihat catatan pada mastersparepart/lookup.go untuk buktinya.
func TestLookupFilterIsApproved(t *testing.T) {
	require.Equal(t, string(mastersparepart.StatusApproved), approvedLookup)
	require.Equal(t, "1", approvedLookup)
}

func TestLikeQueriesDeclareEscape(t *testing.T) {
	text := strings.ToUpper(getQuery("sparepart_list_search"))
	require.Contains(t, text, "LIKE")
	require.Equal(t, 3, strings.Count(text, "ESCAPE '\\'"),
		"ketiga LIKE harus menyebut karakter pelolosnya; Oracle tidak punya bawaan")
}

// Urutan daftar DITAMBAHKAN terhadap sistem lama, yang tidak menyebut satu pun kolom
// pengurut. Lihat catatan pada sparepart_list.
func TestListOrderIsDeterministic(t *testing.T) {
	for _, name := range []string{"sparepart_list", "sparepart_list_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "ORDER BY ID",
			"kueri %q harus punya urutan yang pasti", name)
	}
}

func TestLikePattern(t *testing.T) {
	require.Equal(t, "%FILTER%", likePattern("filter"))
	require.Equal(t, "%FILTER%", likePattern("  Filter  "))

	// Tanda khusus LIKE diloloskan, supaya yang mengetik "%" tidak menarik seluruh tabel.
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, `%A\\B%`, likePattern(`a\b`))
}

// priceTime mengubah nil menjadi NULL, bukan waktu nol.
//
// Tanpa itu, baris yang harganya belum pernah diisi akan tersimpan dengan tanggal tahun 1 —
// nilai yang tampak sah di layar dan tidak dapat dibedakan dari tanggal yang sungguh diisi.
func TestPriceTimeKeepsNilAsNull(t *testing.T) {
	require.Nil(t, priceTime(nil))

	sample := sampleSparepart()
	require.Nil(t, insertArguments(sample)[21], "TGL_UPDATE_HARGA nil harus menjadi NULL")
}

// Lebar nomor urut meniru `Database/PEGA_M_SPAREPART_HE.prc:21`: SEPULUH digit, bukan enam
// seperti Master Panel.
func TestSequenceWidthFollowsTheProcedure(t *testing.T) {
	require.Equal(t, 10, sequenceWidth)
}

// selectedColumns membaca nama kolom yang disebut sebuah SELECT, pada urutannya.
func selectedColumns(t *testing.T, text string) []string {
	t.Helper()

	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT ")
	require.GreaterOrEqual(t, start, 0)
	end := strings.Index(upperCase, "FROM ")
	require.Greater(t, end, start)

	var result []string
	for _, one := range strings.Split(upperCase[start+len("SELECT "):end], ",") {
		clean := strings.TrimSpace(one)
		// Alias tabel dibuang: yang dibandingkan adalah urutan kolomnya, bukan cara
		// menyebutnya.
		if dot := strings.LastIndex(clean, "."); dot >= 0 {
			clean = clean[dot+1:]
		}
		result = append(result, clean)
	}
	return result
}
