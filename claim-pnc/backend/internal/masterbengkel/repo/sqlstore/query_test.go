package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
)

// sampleWorkshop adalah baris yang dipakai menghitung jumlah argumen.
//
// Isinya tidak diperiksa satu pun uji di berkas ini — yang diuji adalah JUMLAH argumen
// yang dihasilkan insertArguments dan updateArguments, bukan nilainya. Ia sengaja kosong
// supaya tidak ada yang tergoda menambahkan pemeriksaan nilai ke uji yang menjaga bentuk.
func sampleWorkshop() masterbengkel.Workshop {
	return masterbengkel.Workshop{ID: "010000000001", Name: "Bengkel Contoh"}
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"bengkel_list",
		"bengkel_list_search",
		"bengkel_get",
		"bengkel_find_by_name",
		"bengkel_find_by_login",
		"bengkel_lock_by_name",
		"bengkel_lock_by_login",
		"bengkel_insert",
		"bengkel_update",
		"bengkel_set_status",
		"bengkel_count_pending",
		"bengkel_count_all",
		"bengkel_check_table",
		"bengkel_check_json_mirror",
		"bengkel_count_json_mirror",
		"bengkel_site",
		"bengkel_next_sequence",
		"bengkel_branch_list",
		"bengkel_city_search",
		"bengkel_bank_list",
	}
	for _, name := range usedNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
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
// pernah bertabrakan dengan ID yang pernah diterbitkan Pega. Perlakuannya sama dengan
// generator nomor klaim pada ADR-0005 dan dengan Master Status Klaim.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "bengkel_next_sequence"

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
		"bengkel_list",
		"bengkel_list_search",
		"bengkel_get",
		"bengkel_find_by_name",
		"bengkel_find_by_login",
		"bengkel_lock_by_name",
		"bengkel_lock_by_login",
		"bengkel_insert",
		"bengkel_update",
		"bengkel_set_status",
		"bengkel_count_pending",
		"bengkel_city_search",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	require.Containsf(t, getQuery("bengkel_list_search"), ":2",
		"kata kunci pencarian harus terikat, bukan dirangkai ke teks SQL")
}

// Tidak ada DELETE terhadap tabel mana pun. Sistem lama tidak punya satu pun, dan D-66
// melarang penghapusan fisik data bernilai bisnis.
func TestNoDeleteStatement(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; bengkel yang tidak dipakai ditolak, bukan dibuang (D-66)", name)
	}
}

// Kelima objek acuan HANYA DIBACA (ADR-0004, penulis tunggal per tabel).
//
// Satu-satunya tabel yang boleh ditulis modul ini adalah POOLDATA.BENGKEL_HE. Yang
// paling penting dijaga di sini: M_BENGKEL_HE — tabel JSON milik Pega — tidak boleh ikut
// ditulis. Dua penulis atas satu master adalah persis keadaan yang P-1 larang.
func TestOnlyTheMasterTableIsWritten(t *testing.T) {
	readOnlyObjects := []string{
		"POOLDATA.M_BENGKEL_HE",
		"GENERAL.LST_USER_ASURANSI",
		"LST_DET_CABANG",
		"GENERAL.LST_BANK_GROUP",
		"POOLDATA.M_SITE_DATABASE",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.HasPrefix(upperCase, "INSERT") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}
		for _, object := range readOnlyObjects {
			require.NotContainsf(t, upperCase, object,
				"kueri %q menulis ke %s, padahal objek itu hanya boleh dibaca (ADR-0004)", name, object)
		}
		require.Containsf(t, upperCase, "POOLDATA.BENGKEL_HE",
			"kueri tulis %q menyentuh tabel yang bukan milik modul ini", name)
	}
}

// Kelima kueri pembaca master harus menyebut kolom pada URUTAN YANG SAMA.
//
// scanRow membaca kelimanya dengan satu fungsi, berdasarkan POSISI kolom. Satu kolom
// yang bergeser di salah satu kueri akan menaruh nomor rekening ke kolom alamat NPWP
// tanpa satu pun galat — dan pada modul ini kolomnya empat puluh satu, sehingga
// pergeseran itu tidak mungkin terlihat dengan membaca.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	expected := []string{
		"ID_BENGKEL", "NAMA_BENGKEL", "ALM_BENGKEL", "TELP_BENGKEL", "NOHP_BENGKEL",
		"MAIL", "MAIL_WO", "CABANG_ID", "NAMA_CABANG", "CITY_ID",
		"NAMA_KABUPATEN", "STATUS_REKANAN", "STS_BENGKEL", "ALASAN_STS_BGKL", "TGL_STATUS",
		"LOGIN_APLIKASI", "BANK_ID", "NAMA_BANK", "NO_ACCOUNT", "NAMA_ACCOUNT",
		"ACCOUNT_ID", "NAMA_NPWP", "NO_NPWP", "ALM_NPWP", "JENIS_PPH",
		"PPN", "DISC_JASA", "DISC_SPART", "PERSEN_MATERIAL", "PCT_SELISIH_PL",
		"SLA", "STS_SUPPLY", "SUPPLIER", "STS_EKLAIM", "STS_AUTO_AKSEP",
		"STS_PAYMENT", "STS_AUTOPAYMENT", "STS_TEKNO", "STS_ORDER", "DOKUMENID",
		"APPROVAL",
	}

	for _, name := range []string{
		"bengkel_list", "bengkel_list_search", "bengkel_get",
		"bengkel_find_by_name", "bengkel_find_by_login",
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, selectedColumns(getQuery(name)))
		})
	}
}

// Daftar kolom INSERT harus sama persis dengan urutan yang disusun insertArguments.
//
// Keduanya ditulis terpisah — satu di berkas .sql, satu di berkas .go — dan keduanya
// berisi empat puluh satu nama pada urutan yang sama. Satu yang bergeser akan menyimpan
// nomor NPWP ke kolom SLA tanpa galat apa pun.
func TestInsertColumnCountMatchesArguments(t *testing.T) {
	text := getQuery("bengkel_insert")

	open := strings.Index(text, "(")
	close := strings.Index(text, ")")
	require.Greater(t, close, open, "daftar kolom INSERT tidak terbaca")

	column := strings.Count(text[open:close], ",") + 1
	require.Equal(t, len(insertArguments(sampleWorkshop())), column,
		"jumlah kolom INSERT dan jumlah argumennya berbeda")

	// Jumlah placeholder pada VALUES harus sama pula.
	require.Contains(t, text, ":41")
	require.NotContains(t, text, ":42")
}

// Daftar kolom UPDATE harus sama dengan jumlah argumen updateArguments.
//
// Empat puluh kolom ditulis, ditambah satu penyaring WHERE = empat puluh satu argumen.
func TestUpdateArgumentCountMatchesQuery(t *testing.T) {
	require.Len(t, updateArguments(sampleWorkshop()), 41)
	require.Contains(t, getQuery("bengkel_update"), ":41")
	require.NotContains(t, getQuery("bengkel_update"), ":42")
}

// ID_BENGKEL TIDAK BOLEH ada di klausa SET pernyataan UPDATE.
//
// Ia kunci baris, bukan isian. Uji ini yang membuat penambahannya kelak menjadi
// keputusan sadar, bukan kelalaian saat menyalin daftar kolom dari INSERT di sebelahnya.
func TestUpdateNeverMovesTheKey(t *testing.T) {
	text := strings.ToUpper(getQuery("bengkel_update"))
	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.NotContains(t, setClause, "ID_BENGKEL")

	// Sebagai pembanding: INSERT memang menulisnya, sekali, saat baris dibuat.
	require.Contains(t, strings.ToUpper(getQuery("bengkel_insert")), "ID_BENGKEL")
}

// Keputusan borongan TIDAK boleh menyentuh kolom MAIL.
//
// Sistem lama menulisi `InputBengkel.MAIL := .USER_UPDATE` pada jalur persetujuan —
// menimpa surel bengkel dengan identitas petugas yang menyetujui. Uji ini menjaga
// keputusan untuk tidak membawanya tetap terlihat.
func TestDecisionQueryTouchesOnlyApproval(t *testing.T) {
	text := strings.ToUpper(getQuery("bengkel_set_status"))
	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]

	require.Contains(t, setClause, "APPROVAL")
	require.NotContains(t, setClause, "MAIL")
	require.Equal(t, 1, strings.Count(setClause, "="),
		"keputusan borongan hanya boleh menetapkan satu kolom")
}

// Kedua kueri daftar mengurutkan menurun berdasarkan ID_BENGKEL.
//
// Itu urutan bawaan grid Pega, bukan pilihan: `BrowseMasterHEApprove-Section.xml`
// menyetel kolom pertama — ID_BENGKEL — dengan `pySortType=DESC` dan `pySortOrder=1`.
//
// Uji ini ada karena urutan adalah satu-satunya hal pada layar berpaginasi yang salahnya
// TIDAK terlihat sebagai galat: halaman pertama tetap terisi, angkanya tetap masuk akal,
// dan yang berbeda hanya baris mana yang ada di halaman berapa.
func TestListQueriesOrderByKeyDescending(t *testing.T) {
	for _, name := range []string{"bengkel_list", "bengkel_list_search"} {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(getQuery(name))
			require.Contains(t, text, "ORDER BY ID_BENGKEL DESC")
		})
	}
}

// Penyaring cabang TETAP memuat '0076' yang tertanam.
//
// Ia direplikasi apa adanya karena export tidak memuat satu pun keterangan tentang
// artinya, sehingga mengangkatnya menjadi konfigurasi berarti menebak nilainya untuk
// entitas lain. Uji ini menjaga keputusan itu TERLIHAT: siapa pun yang kelak
// mengangkatnya akan menghapus uji ini dengan sadar, bukan mengubah perilaku diam-diam.
func TestBranchQueryKeepsHardcodedApplicationCode(t *testing.T) {
	require.Contains(t, getQuery("bengkel_branch_list"), "'0076'")
}

// Setiap kueri yang memakai LIKE menyebut ESCAPE secara eksplisit.
//
// Oracle tidak punya karakter pelolos bawaan pada LIKE, sehingga tanda persen yang
// diketik pengguna tetap berlaku sebagai wildcard meski sudah diloloskan di Go.
func TestLikeQueriesDeclareEscape(t *testing.T) {
	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.Contains(upperCase, "LIKE") {
			continue
		}
		require.Containsf(t, upperCase, "ESCAPE",
			"kueri %q memakai LIKE tanpa menyebut ESCAPE", name)
	}
}

// Kata kunci dinormalkan dan karakter wildcard-nya diloloskan.
func TestLikePattern(t *testing.T) {
	for _, c := range []struct{ keyword, pattern string }{
		{"jaya", "%JAYA%"},
		{"  Bengkel Jaya  ", "%BENGKEL JAYA%"},
		// Wildcard diloloskan: pengguna yang mengetik "%" mencari tanda persen, bukan
		// meminta seluruh tabel.
		{"50%", `%50\%%`},
		{"a_b", `%A\_B%`},
		{`c\d`, `%C\\D%`},
	} {
		t.Run(c.keyword, func(t *testing.T) {
			require.Equal(t, c.pattern, likePattern(c.keyword))
		})
	}
}

// selectedColumns mengambil nama kolom pada klausa SELECT sebuah kueri sederhana.
//
// Ia hanya menangani bentuk yang dipakai berkas ini — satu kolom per baris, tanpa
// ekspresi dan tanpa alias — dan itu memang cukup: begitu sebuah kueri pembaca menjadi
// lebih rumit dari itu, uji ini gagal dan memaksa pembacanya melihat kembali.
func selectedColumns(text string) []string {
	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT")
	end := strings.Index(upperCase, "FROM")
	if start < 0 || end < 0 || end < start {
		return nil
	}

	var column []string
	for _, part := range strings.Split(upperCase[start+len("SELECT"):end], ",") {
		if clean := strings.TrimSpace(part); clean != "" {
			column = append(column, clean)
		}
	}
	return column
}
