package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterpanel"
)

// samplePanel adalah baris yang dipakai menghitung jumlah argumen.
//
// Isinya tidak diperiksa satu pun uji di berkas ini — yang diuji adalah JUMLAH argumen
// yang dihasilkan insertArguments dan updateArguments, bukan nilainya. Ia sengaja hampir
// kosong supaya tidak ada yang tergoda menambahkan pemeriksaan nilai ke uji yang menjaga
// bentuk.
func samplePanel() masterpanel.Panel {
	return masterpanel.Panel{ID: "01000001", Name: "Panel Contoh"}
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"panel_list",
		"panel_list_search",
		"panel_get",
		"panel_find_by_name",
		"panel_lock_by_name",
		"panel_insert",
		"panel_update",
		"panel_set_status",
		"panel_location_by_status",
		"panel_location_by_status_search",
		"panel_location_get",
		"panel_location_clear",
		"panel_location_insert",
		"panel_count_pending",
		"panel_count_all",
		"panel_check_table",
		"panel_check_location_table",
		"panel_count_location",
		"panel_count_location_orphan",
		"panel_count_location_name_mismatch",
		"panel_check_json_mirror",
		"panel_count_json_mirror",
		"panel_site",
		"panel_next_sequence",
	}

	for _, name := range usedNames {
		require.NotEmptyf(t, getQuery(name), "kueri %q kosong", name)
	}
	require.Lenf(t, query, len(usedNames),
		"ada kueri di berkas .sql yang tidak dipakai kode, atau sebaliknya")
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { getQuery("panel_tidak_ada") })
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
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "panel_next_sequence"

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

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"panel_list",
		"panel_list_search",
		"panel_get",
		"panel_find_by_name",
		"panel_lock_by_name",
		"panel_insert",
		"panel_update",
		"panel_set_status",
		"panel_location_by_status",
		"panel_location_by_status_search",
		"panel_location_get",
		"panel_location_clear",
		"panel_location_insert",
		"panel_count_pending",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	for _, name := range []string{"panel_list_search", "panel_location_by_status_search"} {
		require.Containsf(t, getQuery(name), ":2",
			"kata kunci pencarian pada %q harus terikat, bukan dirangkai ke teks SQL", name)
	}
}

// DELETE hanya boleh ada pada SATU kueri: pembuangan baris lokasi sebelum disisipkan
// ulang.
//
// Ia pertentangan yang disadari dengan D-66, yang menetapkan soft delete menyeluruh.
// Alasannya ada di banner masterpanel.sql: baris anak tidak punya kunci sendiri, dan Work
// Owner memilih "jalankan as is" pada 2026-09-19.
//
// Uji ini memagari pengecualian itu. Yang paling penting dijaga: TIDAK ADA DELETE terhadap
// tabel INDUK. Panel yang tidak lagi dipakai ditolak atau ditandai lewat STS_AKTIF, bukan
// dibuang.
func TestDeleteOnlyOnTheChildTable(t *testing.T) {
	const exempted = "panel_location_clear"

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if name == exempted {
			require.Contains(t, upperCase, "DELETE FROM POOLDATA.LOKASI_PANEL_HE")
			// Sasarannya yang diperiksa, bukan sekadar ada-tidaknya nama tabel:
			// "LOKASI_PANEL_HE" berakhiran "PANEL_HE", sehingga pencocokan substring
			// akan selalu cocok dan uji ini tidak membuktikan apa pun.
			require.NotContains(t, upperCase, "DELETE FROM POOLDATA.PANEL_HE",
				"pembuangan hanya boleh mengenai tabel anak, bukan tabel induk")
			continue
		}
		require.NotContainsf(t, upperCase, "DELETE",
			"kueri %q memuat DELETE; hanya %q yang dibenarkan (D-66)", name, exempted)
	}
}

// Tabel JSON milik Pega HANYA DIBACA (ADR-0004, penulis tunggal per tabel).
//
// Dua tabel yang boleh ditulis modul ini adalah POOLDATA.PANEL_HE dan
// POOLDATA.LOKASI_PANEL_HE. Yang paling penting dijaga di sini: M_PANEL_HE tidak boleh
// ikut ditulis. Dua penulis atas satu master adalah persis keadaan yang P-1 larang.
func TestOnlyTheTwoOwnedTablesAreWritten(t *testing.T) {
	readOnlyObjects := []string{
		"POOLDATA.M_PANEL_HE",
		"POOLDATA.M_SITE_DATABASE",
		"POOLDATA.PANEL_HE_SEQ",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		isWrite := strings.Contains(upperCase, "INSERT INTO") ||
			strings.Contains(upperCase, "UPDATE ") ||
			strings.Contains(upperCase, "DELETE FROM")
		if !isWrite {
			continue
		}
		for _, object := range readOnlyObjects {
			require.NotContainsf(t, upperCase, object,
				"kueri %q menulis ke %s yang seharusnya hanya dibaca", name, object)
		}
	}
}

// Keempat kueri pembaca induk menyebut kolom pada URUTAN yang sama.
//
// scanRow membaca keempatnya berdasarkan POSISI. Satu kolom yang bergeser di salah satunya
// akan menaruh alasan penolakan ke kolom status — tanpa satu pun galat, dan tanpa satu pun
// cara mengetahuinya selain melihat layar yang salah.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	reference := selectedColumns(t, getQuery("panel_list"))
	require.Len(t, reference, 15, "kelima belas kolom induk harus dibaca")

	for _, name := range []string{"panel_list_search", "panel_get", "panel_find_by_name", "panel_check_table"} {
		require.Equalf(t, reference, selectedColumns(t, getQuery(name)),
			"kueri %q membaca kolom pada urutan yang berbeda dari panel_list", name)
	}
}

// Kedua kueri pembaca lokasi per status menyebut kolom pada urutan yang sama.
func TestLocationReaderQueriesShareColumnOrder(t *testing.T) {
	reference := selectedColumns(t, getQuery("panel_location_by_status"))
	require.Len(t, reference, 3)
	require.Equal(t, reference, selectedColumns(t, getQuery("panel_location_by_status_search")))
}

func TestInsertColumnCountMatchesArguments(t *testing.T) {
	text := getQuery("panel_insert")

	open := strings.Index(text, "(")
	close := strings.Index(text, ")")
	require.Greater(t, close, open)

	columns := strings.Split(text[open+1:close], ",")
	require.Len(t, insertArguments(samplePanel()), len(columns),
		"jumlah argumen tidak sama dengan jumlah kolom pada panel_insert")
}

func TestLocationInsertHasFourArguments(t *testing.T) {
	text := strings.ToUpper(getQuery("panel_location_insert"))

	// Keempat kolomnya disebut, dan NAMA adalah yang keempat. Lihat banner
	// masterpanel.sql: nilainya sengaja sama dengan LOKASI_PANEL.
	for _, column := range []string{"ID_PANEL", "LOKASI_PANEL", "SISI_PANEL", "NAMA"} {
		require.Containsf(t, text, column, "panel_location_insert harus menyebut %s", column)
	}
	require.Contains(t, text, ":4", "keempat nilainya harus terikat sebagai parameter")
}

func TestUpdateArgumentCountMatchesQuery(t *testing.T) {
	text := getQuery("panel_update")

	// Kelima belas argumen: empat belas kolom yang ditulis, ditambah ID_PANEL sebagai
	// penyaring WHERE.
	require.Len(t, updateArguments(samplePanel()), 15)
	require.Contains(t, text, ":15", "ID_PANEL harus menjadi parameter terakhir")
}

// Kunci baris TIDAK PERNAH berpindah: panel_update tidak boleh menyebut ID_PANEL sebagai
// kolom yang ditulis.
func TestUpdateNeverMovesTheKey(t *testing.T) {
	text := strings.ToUpper(getQuery("panel_update"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.NotContains(t, setClause, "ID_PANEL",
		"panel_update tidak boleh menulis ID_PANEL")
}

// Keputusan borongan hanya menyentuh APPROVAL dan ALASAN_TOLAK.
//
// Tanpa uji ini, satu kolom yang ikut tertulis pada jalur keputusan akan menimpa isian
// yang diketik petugas lain — tanpa satu pun tanda di layar.
func TestDecisionQueryTouchesOnlyApprovalAndReason(t *testing.T) {
	text := strings.ToUpper(getQuery("panel_set_status"))

	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.Contains(t, setClause, "APPROVAL")
	require.Contains(t, setClause, "ALASAN_TOLAK")

	for _, column := range []string{
		"NAME", "STS_REPAIR", "STS_EDIT_QTY", "STS_PREMIUM_REPAIR", "STS_PECAH",
		"STS_STICKER", "STS_SISI", "STS_RUSAK_PARAH", "STS_AKTIF", "EXCLUSION_C",
		"STS_APPROVAL", "DOKUMENID",
	} {
		require.NotContainsf(t, setClause, column,
			"keputusan tidak boleh menyentuh %s", column)
	}
}

// Kueri LIKE wajib menyebut ESCAPE secara eksplisit.
//
// Oracle tidak punya karakter pelolos bawaan; tanpa ESCAPE, tanda persen yang diketik
// pengguna tetap berlaku sebagai wildcard meski sudah diloloskan di Go.
func TestLikeQueriesDeclareEscape(t *testing.T) {
	for name, text := range query {
		if !strings.Contains(strings.ToUpper(text), "LIKE") {
			continue
		}
		require.Containsf(t, text, `ESCAPE '\'`,
			"kueri %q memakai LIKE tanpa ESCAPE eksplisit", name)
	}
}

// Kedua penyaring kata kunci — induk dan anak — harus SAMA PERSIS.
//
// Bila berbeda, daftar induk dan daftar anaknya akan berisi panel yang berbeda, dan
// sebagian baris akan tampil tanpa lokasinya tanpa satu pun tanda bahwa ada yang salah.
func TestParentAndChildSearchFiltersMatch(t *testing.T) {
	parent := strings.ToUpper(getQuery("panel_list_search"))
	child := strings.ToUpper(getQuery("panel_location_by_status_search"))

	require.Contains(t, parent, "UPPER(NAME) LIKE :2")
	require.Contains(t, child, "UPPER(P.NAME) LIKE :2")
}

// Daftar diurutkan ID MENURUN, meniru `pySortType=DESC` pada report definition lama.
//
// Ia sempat diganti `ORDER BY NAME` dengan alasan "lebih mudah dicari mata", dan itu
// dibatalkan setelah layar Pega yang sebenarnya dibandingkan. Uji ini menjaganya supaya
// tidak berubah lagi tanpa alasan yang lebih kuat daripada selera.
func TestListOrderFollowsPega(t *testing.T) {
	for _, name := range []string{"panel_list", "panel_list_search"} {
		text := strings.ToUpper(getQuery(name))
		require.Containsf(t, text, "ORDER BY ID_PANEL DESC",
			"kueri %q harus mengurut ID menurun seperti grid Pega", name)
		require.NotContainsf(t, text, "ORDER BY NAME",
			"kueri %q mengurut nama; itu bukan urutan yang dilihat petugas di Pega", name)
	}
}

func TestLikePattern(t *testing.T) {
	require.Equal(t, "%PINTU%", likePattern("pintu"))
	require.Equal(t, "%PINTU%", likePattern("  Pintu  "))

	// Karakter khusus LIKE diloloskan supaya pengguna yang mengetiknya tidak menarik
	// seluruh tabel.
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, `%\\%`, likePattern(`\`))
}

// selectedColumns membaca daftar kolom dari klausa SELECT sebuah kueri.
//
// Hanya bagian sebelum FROM yang dibaca, dan setiap kolom dipangkas serta dinaikkan
// menjadi huruf besar supaya perbandingannya tidak bergantung pada penulisan.
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
		// Alias tabel dibuang: kueri lokasi menyebut kolomnya sebagai `L.ID_PANEL`
		// sementara kueri induk menyebutnya `ID_PANEL`, dan yang dibandingkan adalah
		// urutan kolomnya — bukan cara menyebutnya.
		if dot := strings.LastIndex(clean, "."); dot >= 0 {
			clean = clean[dot+1:]
		}
		result = append(result, clean)
	}
	return result
}
