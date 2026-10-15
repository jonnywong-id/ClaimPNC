package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	dipakai := []string{
		"claim_status_list",
		"claim_status_get",
		"claim_status_site",
		"claim_status_next_sequence",
		"claim_status_insert",
		"claim_status_update",
		"claim_status_check_table",
	}
	for _, name := range dipakai {
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
	terlarang := map[string]string{
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
		for pola, alasan := range terlarang {
			require.NotContainsf(t, uppercase, pola,
				"kueri %q memakai %q — %s", name, pola, alasan)
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
// generator nomor klaim pada ADR-0005.
//
// Uji ini memagari pengecualian itu supaya ia tidak menyebar: kueri KEDUA yang memakai
// FROM DUAL akan membuat uji ini gagal.
func TestFromDualOnlyInSequenceQuery(t *testing.T) {
	const exempted = "claim_status_next_sequence"

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
	berparameter := []string{
		"claim_status_get",
		"claim_status_insert",
		"claim_status_update",
	}
	for _, name := range berparameter {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Modul ini menulis, dan yang ditulisnya adalah tabel milik sistem lama. Tidak ada satu
// pun jalur yang boleh MENGHAPUS baris master: layar Pega tidak punya tombol hapus, dan
// ADR-0012 melarang master dihapus permanen karena klaim lama merujuknya.
func TestNoQueryDeletes(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "DELETE", "kueri %q menghapus baris", name)
		require.NotContainsf(t, uppercase, "TRUNCATE", "kueri %q mengosongkan tabel", name)
		require.NotContainsf(t, uppercase, "DROP ", "kueri %q membuang objek basis data", name)
	}
}

// Kueri tulis hanya boleh menyentuh M_STS_CLAIM. Menyentuh V_STS_CLAIM akan menulis ke
// view yang dibaca 23 rule Pega, dan menyentuh tabel lain melanggar P-1.
func TestWriteQueriesTouchOnlyBaseTable(t *testing.T) {
	for _, name := range []string{"claim_status_insert", "claim_status_update"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "POOLDATA.M_STS_CLAIM",
			"kueri %q harus menulis ke tabel dasarnya", name)
		require.NotContains(t, text, "V_STS_CLAIM",
			"kueri %q tidak boleh menulis lewat view yang dibaca Pega", name)
	}
}

// Kode dibentuk id_site || lpad(urutan, 3, '0'). Uji ini mengunci pembentukannya,
// termasuk perilaku yang SENGAJA tidak diperbaiki di atas 999.
func TestThreeDigitsMimicsOracleLpad(t *testing.T) {
	require.Equal(t, "001", ThreeDigits(1))
	require.Equal(t, "099", ThreeDigits(99))
	require.Equal(t, "134", ThreeDigits(134), "kode pertama pada master hari ini: 1 + 134 = 1134")
	require.Equal(t, "166", ThreeDigits(166), "kode terakhir pada master hari ini")
	require.Equal(t, "999", ThreeDigits(999))

	// LPAD Oracle tidak memotong. Kode ke-1000 memang menjadi lima karakter, dan itu
	// cacat skema warisan yang sengaja tidak ditutupi: memotongnya akan menghasilkan
	// kode ganda, yang jauh lebih buruk daripada kode yang kepanjangan.
	require.Equal(t, "1000", ThreeDigits(1000),
		"perilaku ini diwarisi apa adanya; bila diubah, ubah juga catatannya di README")
}

// Nama indeks unik dipakai dua tempat: migrasi 0002 membuatnya, dan kode Go
// menerjemahkan galat bentrok dengan mencocokkan namanya. Bila keduanya berbeda, bentrok
// label akan muncul sebagai galat 500 di layar.
func TestLabelIndexNameMatchesMigration(t *testing.T) {
	require.Equal(t, "UX_M_STS_CLAIM_LABEL", LabelIndexName,
		"harus sama persis dengan nama indeks pada backend/migrations/0002")
}

// Nama kunci utama BUKAN tebakan. Ia dibaca dari ALL_CONSTRAINTS pada 2026-09-17, dan
// ternyata terbalik dari pola penamaan yang dipakai migrasi 0001 — `M_STS_CLAIM_PK`,
// bukan `PK_M_STS_CLAIM`. Salah nama membuat kode bentrok muncul sebagai galat 500.
func TestPrimaryKeyNameMatchesDatabase(t *testing.T) {
	require.Equal(t, "M_STS_CLAIM_PK", PrimaryKeyName,
		"diverifikasi dari ALL_CONSTRAINTS; jangan diubah tanpa membaca ulang katalog")
}
