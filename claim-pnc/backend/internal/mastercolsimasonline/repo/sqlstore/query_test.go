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
		"cause_of_loss_business_deactivate_all",
		"cause_of_loss_business_activate",
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
		"cause_of_loss_business_deactivate_all",
		"cause_of_loss_business_activate",
		"cause_of_loss_business_insert",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Kolom M_COL_ID tidak boleh ikut di-SET saat memperbarui: ia kunci baris, dirujuk
// D_CAUSE_OF_LOSS.M_COL_ID pada data yang sudah berjalan.
func TestUpdateNeverChangesRowKey(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_update"))
	setClause := text[strings.Index(text, "SET"):strings.Index(text, "WHERE")]
	require.NotContains(t, setClause, "M_COL_ID",
		"M_COL_ID hanya boleh menyaring di WHERE, tidak pernah di-SET")
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

// Setiap pembaca pemetaan bisnis WAJIB menyaring baris yang sudah ditandai tidak aktif.
//
// Ini konsekuensi langsung soft delete yang `09-DATABASE-STRATEGY.md` §8.1 sebut: satu
// kueri yang lupa akan menampilkan data yang seharusnya sudah hilang — kelas cacat baru
// yang tidak ada di sistem lama.
func TestBusinessReaderFiltersInactiveRows(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_list"))
	require.Contains(t, text, "STS_AKTIF",
		"pembaca pemetaan bisnis wajib menyaring baris yang ditandai tidak aktif")
}

// Baris pemetaan dikenali menurut POSISINYA di grid, bukan menurut nama maupun ID.
//
// Keduanya gugur karena bukti: BISNISID boleh NULL (nama yang diketik bebas tidak punya
// ID), dan NAMA_BISNIS tidak unik (grid Pega tidak punya penanda keunikan sama sekali,
// sehingga satu bisnis boleh dipilih dua kali).
//
// Mencocokkan menurut nama akan mengenai dua baris sekaligus saat namanya kembar;
// mencocokkan menurut ID akan menyisipkan baris kembar setiap kali disimpan ulang.
// Keduanya gejalanya baru terlihat setelah data menumpuk.
func TestBusinessUpsertMatchesByRowPosition(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_activate"))
	whereClause := text[strings.Index(text, "WHERE"):]

	require.Contains(t, whereClause, "URUTAN", "penyaringnya wajib memakai posisi baris")
	require.NotContains(t, whereClause, "NAMA_BISNIS",
		"nama tidak boleh menjadi penyaring: ia boleh kembar")
	require.NotContains(t, whereClause, "BISNISID",
		"BISNISID tidak boleh menjadi penyaring: ia boleh NULL")
}

// Urutan baris disimpan dan dibaca kembali dari kolomnya sendiri.
//
// Mengurutkan menurut BISNISID akan menempatkan seluruh baris tanpa ID di satu ujung,
// dan layar menampilkan urutan yang berbeda dari yang baru saja disimpan pengguna.
func TestBusinessOrderComesFromItsOwnColumn(t *testing.T) {
	text := strings.ToUpper(getQuery("cause_of_loss_business_list"))
	require.Contains(t, text, "ORDER BY URUTAN")
	require.NotContains(t, text, "ORDER BY BISNISID")
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
