package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya — sebagai
// panik, di produksi.
func TestAllUsedQueriesExist(t *testing.T) {
	used := []string{
		"surveyor_type_list",
		"surveyor_type_get",
		"surveyor_type_site",
		"surveyor_type_next_sequence",
		"surveyor_type_insert",
		"surveyor_type_update",
		"surveyor_type_check_table",
	}
	for _, name := range used {
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
// pernah bertabrakan dengan kode yang pernah diterbitkan Pega.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "surveyor_type_next_sequence"

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
		"surveyor_type_get",
		"surveyor_type_insert",
		"surveyor_type_update",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Tidak ada satu pun jalur yang boleh MENGHAPUS baris master: layar Pega tidak punya
// tombol hapus, procedure lamanya hanya mengenal INSERT dan UPDATE, dan menghapus satu
// tipe akan membuat puluhan baris D_SURVEYORS kehilangan golongannya.
func TestNoQueryDeletes(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "DELETE", "kueri %q menghapus baris", name)
		require.NotContainsf(t, uppercase, "TRUNCATE", "kueri %q mengosongkan tabel", name)
		require.NotContainsf(t, uppercase, "DROP ", "kueri %q membuang objek basis data", name)
	}
}

// Kueri tulis hanya boleh menyentuh M_SURVEYORS. Menyentuh V_M_SURVEYORS berarti menulis
// lewat view yang dibaca rule Pega, dan menyentuh tabel lain melanggar P-1.
func TestWriteQueriesTouchOnlyBaseTable(t *testing.T) {
	for _, name := range []string{"surveyor_type_insert", "surveyor_type_update"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "POOLDATA.M_SURVEYORS",
			"kueri %q harus menulis ke tabel dasarnya", name)
		require.NotContains(t, text, "V_M_SURVEYORS",
			"kueri %q tidak boleh menulis lewat view yang dibaca Pega", name)
	}
}

// Kueri tulis TIDAK menyentuh JSON_DATA sama sekali.
//
// Keputusan Work Owner 2026-09-19: yang ditulis adalah kolom DESCRIPTION, karena itulah
// yang dibaca V_M_SURVEYORS. Menulis JSON_DATA berarti menghidupkan kembali format yang
// sudah diputuskan ditinggalkan (D-02, D-68).
func TestWriteQueriesDoNotTouchJSONColumn(t *testing.T) {
	for _, name := range []string{"surveyor_type_insert", "surveyor_type_update"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "JSON_DATA",
			"kueri %q menyentuh JSON_DATA; kolom itu tidak dipakai lagi", name)
	}
}

// Kueri tulis hanya menyentuh kolom yang memang boleh berubah.
//
// M_SURVEY_ID tidak pernah di-SET — ia dipatok tiga kueri Pega dan disimpan setiap baris
// D_SURVEYORS. OLD_M_SURVEY_ID juga tidak: ia jejak sejarah.
func TestUpdateChangesDescriptionOnly(t *testing.T) {
	text := strings.ToUpper(getQuery("surveyor_type_update"))

	require.Contains(t, text, "SET DESCRIPTION")
	require.NotContains(t, text, "SET M_SURVEY_ID")
	require.NotContains(t, text, "OLD_M_SURVEY_ID",
		"kode lama tidak pernah diubah aplikasi")
}

// Kode dibentuk id_site || lpad(urutan, 3, '0'). Uji ini mengunci pembentukannya,
// termasuk perilaku yang SENGAJA tidak diperbaiki di atas 999.
func TestThreeDigitsMimicsOracleLpad(t *testing.T) {
	require.Equal(t, "001", ThreeDigits(1), "kode pertama pada master hari ini: 1 + 001 = 1001")
	require.Equal(t, "004", ThreeDigits(4), "kode terakhir pada master hari ini")
	require.Equal(t, "011", ThreeDigits(11), "M_SURVEYORS_SEQ berada di 11 pada 2026-09-19")
	require.Equal(t, "099", ThreeDigits(99))
	require.Equal(t, "999", ThreeDigits(999))

	// LPAD Oracle tidak memotong. Kode ke-1000 memang menjadi lima karakter, dan karena
	// M_SURVEY_ID bertipe CHAR(4), penyisipannya akan DITOLAK ORA-12899. Itu cacat skema
	// warisan yang sengaja tidak ditutupi: memotongnya akan menghasilkan kode ganda, yang
	// jauh lebih buruk daripada penyisipan yang gagal dengan pesan jelas.
	require.Equal(t, "1000", ThreeDigits(1000),
		"perilaku ini diwarisi apa adanya; bila diubah, ubah juga catatannya di migrasi 0003")
}

// Nama indeks unik dipakai dua tempat: migrasi 0003 membuatnya, dan kode Go menerjemahkan
// galat bentrok dengan mencocokkan namanya. Bila keduanya berbeda, bentrok nama akan
// muncul sebagai galat 500 di layar.
func TestDescriptionIndexNameMatchesMigration(t *testing.T) {
	require.Equal(t, "UX_M_SURVEYORS_DESC", DescriptionIndexName,
		"harus sama persis dengan nama indeks pada backend/migrations/0003")
}

// Nama kunci utama BUKAN tebakan. Ia dibaca dari ALL_CONSTRAINTS pada 2026-09-19. Salah
// nama membuat kode bentrok muncul sebagai galat 500.
func TestPrimaryKeyNameMatchesDatabase(t *testing.T) {
	require.Equal(t, "M_SURVEYORS_PK", PrimaryKeyName,
		"diverifikasi dari ALL_CONSTRAINTS; jangan diubah tanpa membaca ulang katalog")
}

// Kueri baca mengambil ketiga kolom dengan urutan yang sama dengan yang dipindai
// scanRow. Urutan yang tertukar tidak menghasilkan galat apa pun — ia menghasilkan
// deskripsi yang berisi kode, dan sebaliknya.
func TestReadQueriesSelectColumnsInScanOrder(t *testing.T) {
	for _, name := range []string{"surveyor_type_list", "surveyor_type_get", "surveyor_type_check_table"} {
		text := strings.ToUpper(getQuery(name))

		posisiKode := strings.Index(text, "M_SURVEY_ID")
		posisiDeskripsi := strings.Index(text, "DESCRIPTION")
		posisiKodeLama := strings.Index(text, "OLD_M_SURVEY_ID")

		require.Truef(t, posisiKode >= 0 && posisiDeskripsi >= 0 && posisiKodeLama >= 0,
			"kueri %q harus menyebut ketiga kolom", name)
		require.Lessf(t, posisiKode, posisiDeskripsi,
			"kueri %q: M_SURVEY_ID harus mendahului DESCRIPTION, sesuai urutan Scan", name)
		require.Lessf(t, posisiDeskripsi, posisiKodeLama,
			"kueri %q: DESCRIPTION harus mendahului OLD_M_SURVEY_ID, sesuai urutan Scan", name)
	}
}
