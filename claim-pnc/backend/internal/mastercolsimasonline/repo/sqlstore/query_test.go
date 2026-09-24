package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"cause_of_loss_list",
		"cause_of_loss_get",
		"cause_of_loss_business_list",
		"cause_of_loss_site",
		"cause_of_loss_next_sequence",
		"cause_of_loss_insert",
		"cause_of_loss_update",
		"cause_of_loss_business_update",
		"cause_of_loss_business_insert",
		"business_list",
		"cause_of_loss_check_table",
		"cause_of_loss_business_check_table",
		"business_check_table",
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
		"LPAD(":    "pemformatan angka dilakukan di Go",
		"MERGE ":   "sintaksnya berbeda jauh antara Oracle dan PostgreSQL; pakai UPDATE lalu INSERT",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, uppercase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai urutan yang
// sama dengan procedure lama adalah syarat agar kode yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan kode yang pernah diterbitkan Pega. Perlakuannya sama dengan
// modul Master Status Klaim dan dengan generator nomor klaim pada `ADR-0005`.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "cause_of_loss_next_sequence"

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
		"cause_of_loss_get",
		"cause_of_loss_business_list",
		"cause_of_loss_insert",
		"cause_of_loss_update",
		"cause_of_loss_business_update",
		"cause_of_loss_business_insert",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Kolom M_COL_ID tidak boleh ikut di-SET saat memperbarui: ia kunci baris, dirujuk
// D_CAUSE_OF_LOSS.M_COL_ID pada data yang sudah berjalan.
// Dicocokkan sebagai NAMA KOLOM UTUH, bukan sebagai substring.
//
// Pencocokan substring dipakai di sini sampai 2026-09-23 dan langsung menghasilkan positif
// palsu begitu kuerinya mengisi NAME_M_COL_ID: nama itu MEMUAT "M_COL_ID", sehingga uji
// ini gagal atas kueri yang sebenarnya benar. Pelajarannya sama dengan `@contains` pada
// toleransi spreading sistem lama, yang meloloskan 199.99.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	assigned := assignedColumns(getQuery("cause_of_loss_update"))

	require.NotContains(t, assigned, "M_COL_ID",
		"M_COL_ID hanya boleh menyaring di WHERE, tidak pernah di-SET")

	// Keduanya justru HARUS ada: deskripsi ditulis ke dua kolom sekaligus.
	require.Contains(t, assigned, "NAME_M_COL_ID")
	require.Contains(t, assigned, "COL_DESC")
}

// assignedColumns mengembalikan nama kolom yang benar-benar di-SET sebuah UPDATE.
//
// Ia mengurai klausa SET menjadi daftar nama, bukan memperlakukannya sebagai teks —
// sehingga "M_COL_ID" tidak pernah cocok dengan "NAME_M_COL_ID".
func assignedColumns(text string) []string {
	upper := strings.ToUpper(text)

	start := strings.Index(upper, " SET ")
	if start < 0 {
		return nil
	}
	body := upper[start+len(" SET "):]
	if end := strings.Index(body, "WHERE"); end >= 0 {
		body = body[:end]
	}

	var name []string
	for _, assignment := range strings.Split(body, ",") {
		before, _, found := strings.Cut(assignment, "=")
		if !found {
			continue
		}
		if column := strings.TrimSpace(before); column != "" {
			name = append(name, column)
		}
	}
	return name
}

// `D-66` menetapkan tidak ada penghapusan fisik pada data bernilai bisnis, dan secara
// khusus mencabut pola hapus-lalu-sisip-ulang. Uji ini penegaknya: pemetaan bisnis
// diganti dengan menandai, bukan dengan DELETE.
//
// Gejala bila aturan ini dilanggar sangat halus — layarnya tetap bekerja persis sama —
// sehingga tidak ada yang akan menangkapnya selain uji seperti ini.
func TestNoPhysicalDeleteAnywhere(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memakai DELETE; D-66 menetapkan penghapusan dinyatakan lewat penanda", name)
	}
}

// Modul ini membaca dan menulis PASANGAN TABEL JALUR ONLINE — bukan master COL biasa.
//
// Uji ini adalah penegak koreksi 2026-09-23, dan ia ada karena kekeliruannya TIDAK
// menimbulkan galat: M_CAUSE_OF_LOSS dan M_CAUSE_OF_LOSS_ONLINE punya nama kolom yang sama
// persis, sehingga kueri yang menunjuk tabel salah tetap berjalan dan hanya menampilkan
// baris milik master yang lain. Tanpa uji ini, tidak ada apa pun yang akan menangkapnya.
func TestQueriesTargetTheOnlineTables(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)

		// Dicocokkan sebagai NAMA UTUH: "M_CAUSE_OF_LOSS" adalah awalan
		// "M_CAUSE_OF_LOSS_ONLINE", sehingga pencocokan substring akan menuduh setiap
		// kueri yang benar. Pelajaran yang sama dengan `@contains` pada toleransi
		// spreading sistem lama, yang meloloskan 199.99.
		for _, wrong := range []string{
			"POOLDATA.M_CAUSE_OF_LOSS ",
			"POOLDATA.M_CAUSE_OF_LOSS\n",
			"POOLDATA.M_CAUSE_OF_LOSS_BUSINESS",
			"POOLDATA.V_M_CAUSE_OF_LOSS",
		} {
			require.NotContainsf(t, uppercase+"\n", wrong,
				"kueri %q menunjuk %s; layar Simas Online memakai pasangan tabel _ONLINE", name, strings.TrimSpace(wrong))
		}
	}
}

// Urutan yang dipakai adalah MILIK JALUR ONLINE.
//
// M_CAUSE_SEQ dan M_CAUSE_SEQ_ONLINE keduanya ada dan keduanya memasok master yang
// BERBEDA. Tertukar berarti kode yang diterbitkan layar ini bertabrakan dengan deret milik
// master COL biasa — dan tabrakannya baru terlihat setelah datanya menumpuk.
func TestSequenceIsTheOnlineOne(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_next_sequence"))
	require.Contains(t, text, "M_CAUSE_SEQ_ONLINE.NEXTVAL")
}

// Pemetaan bisnis dibaca LANGSUNG dari tabelnya, bukan dengan mem-parse JSON.
//
// Uji ini semula menuntut kebalikannya — JSON_TABLE atas OLD_M_COL_ID — karena tabel
// pemetaannya disangka tidak ada. Yang tidak ada adalah M_CAUSE_OF_LOSS_BUSINESS, tabel
// yang direncanakan migrasi 0004; POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL sudah ada dan
// sudah berisi 282 baris untuk 55 induk.
//
// Membaca langsung dari tabel adalah yang diminta Work Owner 2026-09-22 ("langsung ke
// database, tidak ke json"), dan sekaligus yang benar: hasil JSON_TABLE itu SELALU KOSONG
// karena tidak satu pun OLD_M_COL_ID berisi JSON.
func TestBusinessReaderReadsTheDetailTableDirectly(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_list"))

	require.Contains(t, text, "POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL")
	require.NotContains(t, text, "JSON_TABLE",
		"pemetaan dibaca dari kolom, bukan dari dokumen JSON")
	require.NotContains(t, text, "OLD_M_COL_ID",
		"kolom itu milik master COL biasa dan tidak pernah berisi JSON di sini")
}

// Baris pemetaan dikenali menurut NAMANYA, bukan menurut ID maupun posisinya.
//
// ID gugur karena boleh NULL — nama yang diketik bebas tidak punya ID sama sekali, dan
// pada data hari ini memang ada satu baris seperti itu. Posisi gugur karena tabelnya tidak
// punya kolom urutan.
func TestBusinessUpsertMatchesByName(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_update"))
	whereClause := text[strings.Index(text, "WHERE"):]

	require.Contains(t, whereClause, "NOTE", "penyaringnya wajib memakai nama bisnis")
	require.NotContains(t, whereClause, "URUTAN",
		"tabel ini tidak punya kolom urutan")
}

// Urutan tampilan mengikuti nama, dan itu keterbatasan yang DIKETAHUI.
//
// POOLDATA.M_CAUSE_OF_LOSS_ONLINE_DETAIL tidak punya kolom urutan, sehingga susunan baris
// yang disimpan pengguna TIDAK dapat dipertahankan. Mengurutkan menurut ID akan
// menempatkan baris tanpa ID di satu ujung — lebih membingungkan lagi.
//
// Uji ini memagari pilihan itu supaya ia tidak diam-diam berubah.
func TestBusinessOrderFollowsName(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_list"))
	require.Contains(t, text, "ORDER BY NOTE")
	require.NotContains(t, text, "ORDER BY ID")
}

// POOLDATA.BUSINESS tidak boleh di-join pada pembacaan pemetaan.
//
// Namanya sudah tersimpan di NAMA_BISNIS, dan join apa pun akan gagal menemukan baris
// yang memang tidak punya ID — persis kelas cacat yang kueri lama punya.
func TestBusinessReaderDoesNotJoinTheGISFWTable(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_list"))
	require.NotContains(t, text, "POOLDATA.BUSINESS",
		"nama dibaca dari NAMA_BISNIS, bukan dari tabel milik GISFW")
}

// Master bisnis milik GISFW (`D-03`) dan hanya boleh DIBACA. Uji ini menegakkan batas
// kepemilikan itu di tingkat kueri, bukan hanya di komentar.
func TestBusinessTableIsNeverWritten(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		if !strings.Contains(uppercase, "POOLDATA.BUSINESS") {
			continue
		}
		for _, write := range []string{"INSERT INTO POOLDATA.BUSINESS", "UPDATE POOLDATA.BUSINESS"} {
			require.NotContainsf(t, uppercase, write,
				"kueri %q menulis POOLDATA.BUSINESS; tabel itu milik GISFW dan hanya dibaca (D-03)", name)
		}
	}
}

// Kueri pemeriksa tabel tidak boleh mengambil satu baris pun — ia dijalankan terhadap
// produksi pada mode periksa.
func TestCheckTableFetchesNoRows(t *testing.T) {
	for _, name := range []string{
		"cause_of_loss_check_table",
		"cause_of_loss_business_check_table",
		"business_check_table",
	} {
		require.Containsf(t, getQuery(name), "1 = 0", "kueri %q harus tidak mengambil baris", name)
	}
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0'). Termasuk perilakunya di atas 999, yang
// SENGAJA dibiarkan menghasilkan kode lebih panjang alih-alih dipotong menjadi kode
// ganda — lihat catatan pada fungsinya.
func TestThreeDigitsMirrorsOracleLPAD(t *testing.T) {
	require.Equal(t, "001", ThreeDigits(1))
	require.Equal(t, "099", ThreeDigits(99))
	require.Equal(t, "134", ThreeDigits(134))
	require.Equal(t, "999", ThreeDigits(999))
	require.Equal(t, "1000", ThreeDigits(1000), "di atas 999 LPAD tidak memotong, dan kita pun tidak")
}

// Isian opsional yang kosong disimpan sebagai NULL, bukan teks kosong. Dua bentuk
// "tidak diisi" berarti keduanya harus sama-sama diingat setiap kueri sesudahnya.
func TestEmptyOptionalValueBecomesNull(t *testing.T) {
	require.Nil(t, nullIfEmpty(""))
	require.Nil(t, nullIfEmpty("   "))
	require.Equal(t, "SO-001", nullIfEmpty("SO-001"))
}
