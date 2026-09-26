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
		"business_list",
		"rule_list_by_business",
		"rule_get",
		"rule_business_of",
		"coverage_list",
		"coverage_exists",
		"coverage_insert",
		"rule_next_sequence",
		"rule_site",
		"rule_insert",
		"rule_update",
		"business_choice_list",
		"document_type_choice_list",
		"detail_type_doc_choice_list",
		"object_doc_choice_list",
		"rule_check_table",
		"coverage_check_table",
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

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
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

// CURRENT_TIMESTAMP pun tidak dipakai, dan itu lebih ketat daripada disiplin portabilitas.
//
// EDIT_DATE diisi dari seam jam (`F-5`), bukan dari jam basis data. Bila kelak ada yang
// menggantinya dengan CURRENT_TIMESTAMP "supaya lebih ringkas", waktu simpan akan berhenti
// dapat diuji dan akan mengikuti zona waktu server basis data — dua hal yang justru
// dihindari `F-5` dan `R-12`.
func TestSaveTimeComesFromTheApplicationNotTheDatabase(t *testing.T) {
	for _, name := range []string{"rule_insert", "rule_update"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "CURRENT_TIMESTAMP",
			"kueri %q mengambil waktu dari basis data; EDIT_DATE harus datang dari seam jam", name)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai urutan yang
// sama dengan procedure lama adalah syarat agar ID yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan ID yang pernah diterbitkan Pega.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "rule_next_sequence"

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

// Nilai selalu lewat parameter binding.
//
// Ini penting khusus di modul ini: KEDUA kueri layar lama merangkai nilai ke dalam teks SQL
// lewat `{ASIS:...}` — `Select_TYPE_DOCUMENT` pada penyaringnya, dan keenam kueri unggah
// pada daftar jaminannya. Pola itulah yang `11-SECURITY.md` §5 tutup tanpa perkecualian.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"rule_list_by_business",
		"rule_get",
		"rule_business_of",
		"coverage_list",
		"coverage_exists",
		"coverage_insert",
		"rule_insert",
		"rule_update",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// BUSINESSID tidak boleh ikut di-SET saat memperbarui.
//
// Ini bukan kerapian melainkan aturan bisnis: procedure lama pun tidak mengubahnya
// (`PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:39-41`). Sebuah aturan dokumen tidak dapat
// dipindahkan ke lini bisnis lain — ia disalin. Bila kelak ada yang menambahkannya ke
// klausa SET, baris jaminan yang merujuk pasangan (ID, BUSINESSID) akan berhenti cocok
// dengan induknya, dan dokumennya diam-diam berhenti menjadi wajib.
func TestUpdateNeverChangesTheOwningBusiness(t *testing.T) {
	text := strings.ToUpper(getQuery("rule_update"))
	setClause := text[strings.Index(text, "SET")+len("SET") : strings.Index(text, "WHERE")]

	// Nama kolom yang DITULIS dikumpulkan lebih dulu, bukan dicari sebagai potongan teks.
	//
	// Pencarian teks tidak dapat dipakai di sini: `ID =` juga cocok dengan
	// `DOCUMENT_TYPE_ID =`, `OBJECT_DOC_ID =`, dan `DOC_TYPE_DT_ID =` — ketiganya kolom
	// yang memang boleh diubah. Uji versi pertama saya gagal persis karena itu, dan yang
	// keliru pengujiannya, bukan kuerinya.
	assigned := map[string]bool{}
	for _, assignment := range strings.Split(setClause, ",") {
		name, _, found := strings.Cut(assignment, "=")
		if !found {
			continue
		}
		assigned[strings.TrimSpace(name)] = true
	}

	require.NotEmpty(t, assigned, "klausa SET tidak terbaca; uji ini kehilangan gunanya")
	require.False(t, assigned["BUSINESSID"],
		"BUSINESSID tidak boleh diubah: aturan dokumen disalin ke lini bisnis lain, bukan dipindahkan")
	require.False(t, assigned["ID"],
		"ID adalah kunci baris dan dirujuk COVERAGE_DOC_BUSINESS; ia tidak boleh diubah")
}

// `D-66` menetapkan tidak ada penghapusan fisik pada data bernilai bisnis, dan layar Pega
// pun tidak punya tombol hapus sama sekali.
//
// Uji ini lebih ketat daripada padanannya di modul lain, dan itu disengaja: modul Daftar
// Detail Dokumen Travel MEMANG memakai DELETE untuk mengganti daftar anaknya. Di sini tidak
// boleh — tidak ada satu pun DELETE terhadap COVERAGE_DOC_BUSINESS di seluruh export, dan
// menghapus jaminan mengubah dokumen yang tadinya wajib menjadi tidak wajib pada klaim yang
// sedang berjalan.
func TestNoQueryDeletesAnything(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "DELETE",
			"kueri %q menghapus baris; jaminan hanya ditambahkan, tidak pernah dibuang", name)
		require.NotContainsf(t, uppercase, "TRUNCATE",
			"kueri %q mengosongkan tabel", name)
	}
}

// Keempat master rujukan HANYA DIBACA (`P-1`, `D-03`).
//
// Uji ini menjaga batas kepemilikan pada tingkat SQL, bukan hanya pada bentuk antarmuka
// seam-nya: seseorang yang menambahkan INSERT ke salah satu tabel itu di berkas .sql tidak
// akan ditolak kompilator.
func TestReferenceTablesAreNeverWritten(t *testing.T) {
	owned := []string{"BUSINESS", "V_LST_DOC_TYPE", "V_LST_DET_TYPE_DOC", "V_LST_DOC_OBJ"}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		if !strings.Contains(uppercase, "INSERT") && !strings.Contains(uppercase, "UPDATE") {
			continue
		}
		for _, table := range owned {
			require.NotContainsf(t, uppercase, "INTO POOLDATA."+table,
				"kueri %q menulis %s, yang dimiliki modul lain", name, table)
			require.NotContainsf(t, uppercase, "UPDATE POOLDATA."+table,
				"kueri %q menulis %s, yang dimiliki modul lain", name, table)
		}
	}
}

// Baris jaminan milik JALUR KLAIM tidak boleh ikut terbaca maupun tertulis.
//
// Cabang ber-NOKLAIM pada procedure lama dipakai layar "Simpan Doc PA", bukan layar ini.
// Membaca keduanya bercampur akan menampilkan jaminan milik satu klaim tertentu sebagai
// aturan yang berlaku umum; menulisnya akan mencampur keduanya tanpa cara membedakannya
// kembali.
func TestClaimScopedCoverageIsExcluded(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("coverage_list")), "NOKLAIM IS NULL",
		"daftar jaminan harus menyaring baris milik jalur klaim")
	require.Contains(t, strings.ToUpper(getQuery("coverage_exists")), "NOKLAIM IS NULL",
		"pemeriksaan keberadaan harus memakai penyaring yang sama dengan daftarnya")
	require.NotContains(t, strings.ToUpper(getQuery("coverage_insert")), "NOKLAIM",
		"modul ini tidak pernah menulis NOKLAIM")
}

// Grid dan pengambilan satu baris HARUS memakai bentuk join yang sama.
//
// Bila keduanya berbeda, sebuah baris dapat terbuka lewat penyuntingan tetapi tidak pernah
// tampil di grid — atau sebaliknya, tampil di grid lalu menjawab 404 saat dibuka.
//
// Bentuknya meniru kueri lama (`Select_TYPE_DOCUMENT`): tiga master disamakan secara ketat,
// dan HANYA V_LST_DOC_OBJ yang LEFT — karena OBJECT_DOC_ID memang boleh kosong, dan
// procedure lama mengosongkannya dengan sengaja. Versi pertama modul ini memakai LEFT JOIN
// untuk ketiganya; itu penyimpangan saya sendiri, dan dicabut atas keputusan Work Owner
// 2026-09-23 ("seperti aplikasi Pega saja").
func TestRuleQueriesMirrorTheLegacyJoinShape(t *testing.T) {
	for _, name := range []string{"rule_list_by_business", "rule_get"} {
		text := strings.ToUpper(getQuery(name))

		require.Containsf(t, text, "JOIN POOLDATA.BUSINESS",
			"kueri %q harus menyamakan BUSINESS secara ketat seperti kueri lama", name)
		require.Containsf(t, text, "JOIN POOLDATA.V_LST_DOC_TYPE",
			"kueri %q harus menyamakan V_LST_DOC_TYPE secara ketat seperti kueri lama", name)
		require.Containsf(t, text, "JOIN POOLDATA.V_LST_DET_TYPE_DOC",
			"kueri %q harus menyamakan V_LST_DET_TYPE_DOC secara ketat seperti kueri lama", name)

		require.Containsf(t, text, "LEFT JOIN POOLDATA.V_LST_DOC_OBJ",
			"OBJECT_DOC_ID boleh kosong; menyamakannya ketat akan menghilangkan baris "+
				"tanpa objek dokumen, dan itu BUKAN perilaku Pega (kueri %q)", name)
	}
}

// Tiga kolom yang tidak ditulis siapa pun tetap tidak ditulis.
//
// FLAGTYPES menentukan URUTAN dokumen yang dilihat petugas saat meregistrasi klaim;
// mengisinya sembarangan mengubah layar yang tidak sedang dimigrasikan. Ketiganya menunggu
// jawaban Work Owner — lihat migrations/0010 Bagian 1 butir 5.
func TestUnownedColumnsAreLeftAlone(t *testing.T) {
	for _, name := range []string{"rule_insert", "rule_update"} {
		text := strings.ToUpper(getQuery(name))
		for _, column := range []string{"FLAGTYPES", "CREDENTIAL", "DURATION"} {
			require.NotContainsf(t, text, column,
				"kueri %q menulis %s, yang procedure lama pun tidak pernah mengisinya", name, column)
		}
	}
}
