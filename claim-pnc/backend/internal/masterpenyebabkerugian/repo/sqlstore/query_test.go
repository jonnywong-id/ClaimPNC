package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	used := []string{
		"cause_of_loss_list",
		"cause_of_loss_get",
		"cause_of_loss_site",
		"cause_of_loss_next_sequence",
		"cause_of_loss_insert",
		"cause_of_loss_update",
		"cause_of_loss_check_table",
		"cause_of_loss_count_pending_json",
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

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
//
// FROM DUAL TIDAK ada di daftar terlarang, dan itu disengaja: modul ini memakai urutan
// basis data, dan NEXTVAL menuntutnya. Ia diisolasi di satu kueri — dijaga uji tersendiri
// di bawah, supaya isolasi itu tidak diam-diam meluas.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":    "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":        "pakai COALESCE",
		"SYSDATE":     "pakai CURRENT_TIMESTAMP",
		"DECODE(":     "pakai CASE WHEN",
		"ROWNUM":      "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":      "pakai POSITION",
		"LISTAGG(":    "pakai STRING_AGG",
		"TO_CHAR(":    "pemformatan tanggal dan angka dilakukan di Go",
		"TO_NUMBER(":  "pembacaan ID sebagai bilangan dilakukan di Go",
		"LPAD(":       "pemformatan angka dilakukan di Go — lihat ThreeDigits",
		"CONNECT BY":  "tidak portabel",
		"KEEP (DENSE": "tidak portabel",
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

// FROM DUAL hanya boleh ada di kueri urutan. Ia satu-satunya bentuk khas Oracle di modul
// ini, dan perlakuannya sama dengan generator nomor klaim pada `ADR-0005`: diisolasi di
// satu tempat supaya perpindahan ke PostgreSQL kelak menyentuh satu kueri, bukan tersebar.
func TestFromDualIsolatedToSequenceQuery(t *testing.T) {
	for name, text := range query {
		if name == "cause_of_loss_next_sequence" {
			require.Contains(t, strings.ToUpper(text), "FROM DUAL")
			continue
		}
		require.NotContainsf(t, strings.ToUpper(text), "FROM DUAL",
			"kueri %q memakai FROM DUAL; percabangan dialek hanya boleh di kueri urutan", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat `{ASIS:...}`, 538 kemunculan
// (`docs/Steering/11-SECURITY.md` §5).
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterized := []string{
		"cause_of_loss_get",
		"cause_of_loss_insert",
		"cause_of_loss_update",
	}
	for _, name := range parameterized {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Tidak ada satu pun jalur yang boleh MENGHAPUS baris master.
//
// Alasannya bukan selera: `Database/PEGA_M_CAUSE_OF_LOSS.prc` hanya mengenal INSERT dan
// UPDATE, layar Pega tidak punya tombol hapus, dan menghapus satu golongan akan membuat
// setiap baris `D_CAUSE_OF_LOSS` yang menyimpan `M_COL_ID` itu kehilangan induknya
// (`ADR-0012`).
func TestNoQueryDeletes(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "DELETE", "kueri %q menghapus baris", name)
		require.NotContainsf(t, uppercase, "TRUNCATE", "kueri %q mengosongkan tabel", name)
		require.NotContainsf(t, uppercase, "DROP ", "kueri %q membuang objek basis data", name)
	}
}

// Kueri tulis hanya boleh menyentuh M_CAUSE_OF_LOSS. Menyentuh D_CAUSE_OF_LOSS berarti
// modul golongan ikut mengubah data RINCIAN — pelanggaran batas modul sekaligus
// pelanggaran `P-1`, karena tabel itu dimiliki layar MENU_ID 38 dan bukan layar ini.
func TestWriteQueriesTouchOnlyMasterTable(t *testing.T) {
	for _, name := range []string{"cause_of_loss_insert", "cause_of_loss_update"} {
		text := strings.ToUpper(getQuery(name))
		require.Contains(t, text, "POOLDATA.M_CAUSE_OF_LOSS",
			"kueri %q harus menulis ke tabel masternya", name)
		require.NotContains(t, text, "D_CAUSE_OF_LOSS ",
			"kueri %q tidak boleh menyentuh tabel rincian", name)
	}
}

// Modul ini TIDAK menulis JSON_DATA lagi — keputusan Work Owner 2026-09-20. Kolom itu
// hanya boleh muncul di kueri PEMANTAU, yang membacanya untuk melaporkan berapa baris
// belum ikut dipindahkan migrasi 0005.
func TestJSONColumnOnlyReadNeverWritten(t *testing.T) {
	for _, name := range []string{"cause_of_loss_insert", "cause_of_loss_update"} {
		require.NotContains(t, strings.ToUpper(getQuery(name)), "JSON_DATA",
			"kueri %q tidak boleh menulis dokumen JSON", name)
	}
	require.Contains(t, strings.ToUpper(getQuery("cause_of_loss_count_pending_json")), "JSON_DATA",
		"kueri pemantau justru harus membacanya")
}

// Modul ini membaca TABEL DASAR, bukan view. Membaca view berarti bergantung pada
// definisi yang dimiliki pihak lain, dan apa yang kita tulis belum tentu sama dengan apa
// yang kita baca kembali.
func TestQueriesReadBaseTableNotView(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "V_M_CAUSE_OF_LOSS",
			"kueri %q membaca view; modul ini membaca tabel dasarnya", name)
	}
}

// Kueri daftar SENGAJA tanpa ORDER BY — pengurutannya dikerjakan di Go supaya perilakunya
// sama di Oracle maupun PostgreSQL, termasuk saat kolomnya CHAR berpadding.
func TestListQueryHasNoOrdering(t *testing.T) {
	require.NotContains(t, strings.ToUpper(getQuery("cause_of_loss_list")), "ORDER BY",
		"urutan dikerjakan masterpenyebabkerugian.SortByID supaya portabel")
}

// ThreeDigits meniru lpad(to_char(seq), 3, '0') persis, termasuk saat bilangannya
// menembus batas — perilakunya tidak ditambal, karena pada titik yang sama penyisipannya
// sendiri sudah ditolak basis data.
func TestThreeDigits(t *testing.T) {
	cases := map[int64]string{
		1:    "001",
		9:    "009",
		10:   "010",
		99:   "099",
		100:  "100",
		999:  "999",
		1000: "1000", // empat digit: ID menjadi lima karakter dan penyisipan ditolak
	}
	for input, expected := range cases {
		require.Equalf(t, expected, ThreeDigits(input), "ThreeDigits(%d)", input)
	}
}
