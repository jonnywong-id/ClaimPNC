package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// namedQueries adalah ketiga kueri yang wajib ada di berkas .sql.
var namedQueries = []string{"list_tasks", "check_tables", "check_columns"}

func TestSetiapKueriBernamaAda(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong atau tidak ada", name)
	}
}

func TestTidakAdaKueriTakTerpakaiDiBerkasSQL(t *testing.T) {
	// Kueri yang menganggur adalah kode mati yang kelak dikira siap dipakai.
	require.Len(t, queries, len(namedQueries))
}

// TestAliasKueriSamaDenganDaftarDanUrutannya menjaga tiga tempat tetap sepadan.
//
// Urutan kolom pada kueri, isi `taskColumns`, dan urutan pembacaan `scanTask` WAJIB sama.
// Satu kolom yang bergeser akan memindahkan nomor polis ke kolom nama tertanggung — dan
// keduanya bertipe teks, sehingga tidak ada satu pun galat yang muncul.
func TestAliasKueriSamaDenganDaftarDanUrutannya(t *testing.T) {
	text := query("list_tasks")

	aliases := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(text, -1)
	require.Len(t, aliases, len(taskColumns))

	for i, match := range aliases {
		require.Equalf(t, taskColumns[i], strings.ToUpper(match[1]),
			"alias ke-%d tidak sepadan dengan taskColumns", i+1)
	}
}

// TestKueriDaftarMenyaringKelasObjekKerja mengunci CATATAN 1 pada berkas .sql.
//
// `PC_ASM_FW_GCNMFW_WORK` menampung DUA kelas objek kerja. Report Definition tidak
// menyaringnya karena engine Pega yang melakukannya; SQL langsung harus menuliskannya.
// Melupakannya mencampur berkas penerimaan dokumen ke dalam antrean medis, tanpa galat.
func TestKueriDaftarMenyaringKelasObjekKerja(t *testing.T) {
	require.Contains(t, query("list_tasks"), "'ASM-FW-GCNMFW-Work-PNC'")
}

// TestKueriDaftarMemakaiGabunganDalamKeWorklist mengunci bentuk gabungannya.
//
// `pyJoinType = INNER` pada Report Definition. Menggantinya dengan `EXISTS` akan mengubah
// jumlah baris yang terlihat pengguna — klaim dengan dua penugasan terbuka muncul dua kali di
// Pega, dan itu perilaku yang dibawa (`P-5`).
func TestKueriDaftarMemakaiGabunganDalamKeWorklist(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST")
	require.Contains(t, text, "A.PXREFOBJECTKEY = W.PZINSKEY")
	require.NotContains(t, text, "EXISTS")
}

// TestKueriDaftarMemakaiWorklistBukanWorkbasket menjaga model penugasannya.
//
// Report Definition menggabung ke `Assign-Worklist` — antrean PER ORANG. Tab RCL/PUCL pada
// modul lain memakai `Assign-Workbasket`, antrean bersama, dan keduanya mudah tertukar saat
// menyalin kueri antarmodul.
func TestKueriDaftarMemakaiWorklistBukanWorkbasket(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "PC_ASSIGN_WORKLIST")
	require.NotContains(t, text, "PC_ASSIGN_WORKBASKET")
}

// TestPenyaringStatusMemakaiTidakSamaDengan adalah uji yang paling mudah terbalik.
//
// Filter C berbunyi `!=`. Satu tanda yang salah membalik seluruh isi layar: yang tampil
// menjadi tugas yang sudah tuntas, dan tidak ada apa pun yang menandakannya.
func TestPenyaringStatusMemakaiTidakSamaDengan(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), "W.PYSTATUSWORK <> :3")
}

// TestResolvedRejectedTidakIkutDikecualikan menjaga perbedaan terhadap Inbox Outstanding.
func TestResolvedRejectedTidakIkutDikecualikan(t *testing.T) {
	require.NotContains(t, query("list_tasks"), "Resolved-Rejected")
}

// TestUrutanMenurunDenganPemutusSeri mengunci kedua `pySortType = DESC`.
func TestUrutanMenurunDenganPemutusSeri(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"ORDER BY W.PXCREATEDATETIME DESC, W.PZINSKEY DESC")
}

// TestPaginasiDikerjakanBasisData menjaga halaman dipotong sebelum baris meninggalkannya.
func TestPaginasiDikerjakanBasisData(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"OFFSET :5 ROWS FETCH NEXT :6 ROWS ONLY")
}

// TestJumlahBarisDihitungFungsiJendela menjaga satu perjalanan, bukan dua.
func TestJumlahBarisDihitungFungsiJendela(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), "COUNT(*) OVER ()")
}

// TestSeluruhNilaiLewatParameterBinding adalah uji keamanan, bukan uji gaya.
//
// Kueri lama sistem Pega menyisipkan nilai pengguna langsung ke teks SQL lewat penanda
// `{ASIS:…}` — 538 kemunculan di seluruh export. Larangan perangkaian
// (`08-TECHNICAL-STRATEGY.md` §4.3) tidak dikecualikan oleh keputusan mana pun.
//
// Yang diperiksa: setiap penanda bind `:n` yang dipakai berurutan dari 1, dan tidak ada
// penanda gaya lain yang menandakan perangkaian.
func TestSeluruhNilaiLewatParameterBinding(t *testing.T) {
	text := query("list_tasks")

	markers := regexp.MustCompile(`:\d+`).FindAllString(text, -1)
	require.NotEmpty(t, markers)

	seen := map[string]bool{}
	for _, marker := range markers {
		seen[marker] = true
	}
	for _, expected := range []string{":1", ":2", ":3", ":4", ":5", ":6"} {
		require.Truef(t, seen[expected], "penanda bind %s tidak dipakai", expected)
	}

	require.NotContains(t, text, "{ASIS", "perangkaian gaya Pega tidak boleh terbawa")

	// SATU-SATUNYA perangkaian yang diizinkan adalah pola wildcard `LIKE`, dan yang
	// dirangkai di sana hanyalah tanda persen — nilainya tetap lewat bind.
	//
	// Pemeriksaannya dilakukan dengan MEMBUANG pola yang sah lebih dulu, lalu menuntut tidak
	// ada `||` yang tersisa. Sekadar melarang `' ||` akan menolak pola yang benar sekaligus
	// meloloskan `|| TempFilter.Nama ||` yang justru berbahaya.
	const wildcard = "'%' || UPPER(:4) || '%'"
	require.Contains(t, text, wildcard)

	rest := strings.ReplaceAll(text, wildcard, "")
	require.NotContains(t, rest, "||",
		"selain pola wildcard LIKE, tidak boleh ada perangkaian apa pun ke teks SQL")
}

// TestPencarianDimatikanSaatKataKunciNULL menjaga kotak cari kosong tidak memaksa pemindaian.
//
// `LIKE '%%'` kebetulan cocok dengan semuanya, tetapi memaksa basis data memeriksa setiap
// baris alih-alih melewati predikatnya.
func TestPencarianDimatikanSaatKataKunciNULL(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), ":4 IS NULL")
}

// TestPerbandinganOperatorTidakPekaHurufBesarKecil mengunci CATATAN 3.
func TestPerbandinganOperatorTidakPekaHurufBesarKecil(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"UPPER(A.PXASSIGNEDOPERATORID) = UPPER(:2)")
}

// TestTidakAdaPernyataanYangMenulis menjaga `P-1`.
//
// Seluruh tabel yang dibaca modul ini milik Pega selama masa paralel. Satu `UPDATE` yang
// lolos ke sini berarti dua sistem menulis tabel yang sama — dan kegagalannya tidak
// menghasilkan galat, hanya data yang berubah sendiri.
func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, text, forbidden,
				"kueri %s memuat pernyataan yang menulis: %s", name, forbidden)
		}
	}
}

// TestKueriPeriksaTidakMembacaSatuBarisPun menjaga `-periksa` tetap murah.
func TestKueriPeriksaTidakMembacaSatuBarisPun(t *testing.T) {
	for _, name := range []string{"check_tables", "check_columns"} {
		require.Containsf(t, query(name), "1 = 0",
			"kueri %s harus menolak seluruh baris", name)
	}
}

// TestKueriPeriksaKolomMenyebutKeduaKolomYangBelumTerkonfirmasi.
//
// Inilah satu-satunya tempat yang memberi tahu operator bahwa kedua kolom itu belum ada,
// SEBELUM ada pengguna yang membuka layarnya.
func TestKueriPeriksaKolomMenyebutKeduaKolomYangBelumTerkonfirmasi(t *testing.T) {
	text := strings.ToUpper(query("check_columns"))

	require.Contains(t, text, "ISCOMPLIANCETRANSFER_1")
	require.Contains(t, text, "ANALYSTDOCTORREMAKS_1")
}

// TestKomentarTidakIkutDikirimKeBasisData menjaga pemecah berkas .sql bekerja.
//
// Berkas ini memuat komentar kepala yang panjang. Bila pemecahnya tidak membuangnya, seluruh
// penjelasan itu ikut terkirim pada setiap permintaan.
func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range namedQueries {
		require.NotContainsf(t, query(name), "--",
			"kueri %s masih memuat baris komentar", name)
	}
}
