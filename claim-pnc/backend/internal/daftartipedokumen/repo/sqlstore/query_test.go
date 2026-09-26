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
		"document_type_list",
		"document_type_get",
		"document_type_site",
		"document_type_next_sequence",
		"document_type_insert",
		"document_type_update",
		"document_type_check_table",
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
// TGL_EDIT diisi dari seam jam (`F-5`), bukan dari jam basis data. Bila kelak ada yang
// menggantinya dengan CURRENT_TIMESTAMP "supaya lebih ringkas", waktu simpan akan berhenti
// dapat diuji dan akan mengikuti zona waktu server basis data — dua hal yang justru
// dihindari `F-5` dan `R-12`.
func TestSaveTimeComesFromTheApplicationNotTheDatabase(t *testing.T) {
	for _, name := range []string{"document_type_insert", "document_type_update"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "CURRENT_TIMESTAMP",
			"kueri %q mengambil waktu dari basis data; TGL_EDIT harus datang dari seam jam", name)
	}
}

// FROM DUAL dilarang di mana pun KECUALI satu kueri.
//
// Pengecualian ini disengaja dan terbatas: NEXTVAL menuntutnya, dan memakai urutan yang
// sama dengan procedure lama adalah syarat agar ID yang diterbitkan aplikasi ini tidak
// pernah bertabrakan dengan ID yang pernah diterbitkan Pega. Perlakuannya sama dengan modul
// master lain dan dengan generator nomor klaim pada `ADR-0005`.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "document_type_next_sequence"

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
	parameterised := []string{
		"document_type_get",
		"document_type_insert",
		"document_type_update",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Kolom ID tidak boleh ikut di-SET saat memperbarui: ia kunci baris, dirujuk dua master
// turunan (V_LST_DET_TYPE_DOC, LST_TYPE_DOC_BUSINESS) dan dokumen klaim yang sudah
// terunggah.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	text := strings.ToUpper(getQuery("document_type_update"))
	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.NotContains(t, setClause, "ID =",
		"ID hanya boleh menyaring di WHERE, tidak pernah di-SET")
	require.NotContains(t, setClause, "OLD_ID",
		"OLD_ID adalah jejak sejarah; modul ini tidak pernah menulisnya")
}

// `D-66` menetapkan tidak ada penghapusan fisik pada data bernilai bisnis, dan layar Pega
// pun tidak punya tombol hapus. Uji ini penegaknya pada tingkat SQL.
func TestNoQueryDeletesAnything(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q menghapus baris; master tidak pernah dihapus (`D-66`, `ADR-0012`)", name)
	}
}

// Tabel dasarnya yang dibaca dan ditulis, bukan view-nya.
//
// View V_LST_DOC_TYPE tetap dibaca sekurang-kurangnya sepuluh rule Pega selama masa
// paralel; menulis lewatnya berarti bergantung pada definisi yang dimiliki pihak lain.
// Membaca lewat view, menulis ke tabel dasar.
//
// Uji ini semula menuntut kebalikannya — seluruh kueri menyentuh tabel dasar — karena
// migrasi `0005_daftar_tipe_dokumen` direncanakan memindahkan isi JSON_DATA menjadi kolom
// bernama. Migrasi itu belum dijalankan, dan tabel dasarnya sampai sekarang hanya punya
// ID, OLD_ID, dan JSON_DATA: setiap pembacaan gagal dengan `ORA-00904: "TGL_EDIT"` dan
// layarnya tidak dapat dibuka sama sekali.
//
// Keputusan Work Owner 2026-09-22: ikuti Pega dan baca langsung dari basis data lewat view
// yang sudah memaparkan kolomnya. Menulis tetap ke tabel dasar — view berisi ekspresi JSON
// tidak dapat ditulisi.
func TestReadsUseTheViewAndWritesUseTheBaseTable(t *testing.T) {
	read := []string{"document_type_list", "document_type_get", "document_type_check_table"}
	write := []string{"document_type_insert", "document_type_update"}

	for _, name := range read {
		text := strings.ToUpper(query[name])
		require.Containsf(t, text, "POOLDATA.V_LST_DOC_TYPE",
			"kueri baca %q harus membaca view: tabel dasarnya belum punya kolomnya", name)
	}

	for _, name := range write {
		text := strings.ToUpper(query[name])
		require.NotContainsf(t, text, "V_LST_DOC_TYPE",
			"kueri tulis %q menyentuh view, padahal view berekspresi JSON tidak dapat ditulisi", name)
		require.Containsf(t, text, "POOLDATA.LST_DOC_TYPE",
			"kueri tulis %q harus menulis tabel dasarnya", name)
	}
}
