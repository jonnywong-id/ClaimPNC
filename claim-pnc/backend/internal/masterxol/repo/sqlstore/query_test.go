package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// usedQuery adalah seluruh nama kueri yang benar-benar dipanggil kode di paket ini.
// Tanpa uji ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
var usedQuery = []string{
	"xol_list",
	"xol_get",
	"xol_business_list",
	"xol_layer_list",
	"xol_reas_list",
	"xol_lock_master_ids",
	"xol_lock_layer_ids",
	"xol_insert_master",
	"xol_update_master",
	"xol_business_count",
	"xol_insert_business",
	"xol_insert_layer",
	"xol_update_layer",
	"xol_reas_count",
	"xol_insert_reas",
	"xol_update_reas",
	"xol_delete_master",
	"xol_delete_business_of_master",
	"xol_delete_reas_of_master",
	"xol_delete_layer_of_master",
	"xol_delete_business",
	"xol_delete_layer",
	"xol_delete_reas_of_layer",
	"xol_delete_reas",
	"xol_layer_owner",
	"xol_submit_committee",
	"xol_year_list",
	"xol_business_group_list",
	"xol_orphan_count",
	"xol_check_table",
}

func TestAllUsedQueriesExist(t *testing.T) {
	for _, name := range usedQuery {
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
		"SELECT *":    "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":        "pakai COALESCE",
		"SYSDATE":     "pakai CURRENT_TIMESTAMP",
		"DECODE(":     "pakai CASE WHEN",
		"ROWNUM":      "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":      "pakai POSITION",
		"LISTAGG(":    "pakai STRING_AGG",
		"TO_CHAR(":    "pemformatan tanggal dan angka dilakukan di Go",
		"TO_NUMBER(":  "pembacaan nomor sebagai bilangan dilakukan di Go — lihat lessByNumber",
		"LPAD(":       "pemformatan angka dilakukan di Go",
		"FROM DUAL":   "modul ini tidak memakai sequence, jadi tidak ada alasan memerlukannya",
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

// Kolom LIMIT wajib selalu dikutip.
//
// Di Oracle ia bukan kata cadangan sehingga dapat ditulis polos — dan justru itu yang
// membuat kelalaiannya tidak akan ketahuan sampai cutover ke PostgreSQL, tempat `LIMIT`
// ADALAH kata cadangan dan kuerinya gagal. Uji ini menangkapnya sekarang, bukan nanti.
func TestKolomLimitSelaluDikutip(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		if !strings.Contains(uppercase, "LIMIT") {
			continue
		}
		// Setiap kemunculan LIMIT yang berdiri sendiri harus berupa "LIMIT", kecuali
		// CONVERT_LIMIT yang namanya tidak bertabrakan dengan kata kunci mana pun.
		cleaned := strings.ReplaceAll(uppercase, `"LIMIT"`, "")
		cleaned = strings.ReplaceAll(cleaned, "CONVERT_LIMIT", "")
		require.NotContainsf(t, cleaned, "LIMIT",
			"kueri %q memakai kolom LIMIT tanpa tanda kutip; ia kata cadangan di PostgreSQL", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — dan modul ini mewarisi DUA di antaranya: pernyataan UPDATE yang
// dirangkai dari catatan pengguna di `Activity/UpdateStatusMasterKomitexol-Act.xml`, dan
// potongan klausa WHERE yang disisipkan lewat `{Asis:}` di `GetDataBisnisXol_Sql`.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterized := []string{
		"xol_get",
		"xol_business_list",
		"xol_layer_list",
		"xol_reas_list",
		"xol_insert_master",
		"xol_update_master",
		"xol_insert_layer",
		"xol_update_layer",
		"xol_insert_reas",
		"xol_update_reas",
		"xol_delete_master",
		"xol_layer_owner",
		"xol_submit_committee",
		"xol_business_group_list",
	}
	for _, name := range parameterized {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Penyaring grup bisnis wajib mengikat TIGA pola.
//
// Bentuk kuerinya tetap, dan polanya yang berubah — itulah yang menggantikan perangkaian
// klausa WHERE lewat `{Asis:TempXOL.AgentID}`. Bila salah satu bind hilang, kodenya akan
// mengirim tiga nilai ke kueri yang hanya menerima dua dan gagal saat dijalankan.
func TestPenyaringBisnisMengikatTigaPola(t *testing.T) {
	text := getQuery("xol_business_group_list")
	for _, bind := range []string{":1", ":2", ":3"} {
		require.Containsf(t, text, bind, "penyaring grup bisnis kehilangan parameter %s", bind)
	}
	require.NotContains(t, text, "{", "tidak boleh ada sisa pola penyisipan teks gaya Pega")
}

// Kueri tulis hanya boleh menyentuh keempat tabel MST_XOL_*.
//
// Tiga tabel yang HANYA DIBACA — BUSINESS, BUSINESSGROUP, PROPORTIONALARRG — dan
// M_TREATYYEAR tetap milik sistem lain (`P-1`). Menulis ke salah satunya berarti modul
// master XOL ikut mengubah data milik modul lain.
func TestKueriTulisHanyaMenyentuhTabelXOL(t *testing.T) {
	readOnlyTable := []string{
		"POOLDATA.BUSINESS ",
		"POOLDATA.BUSINESSGROUP",
		"POOLDATA.PROPORTIONALARRG",
		"POOLDATA.M_TREATYYEAR",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		writes := strings.HasPrefix(uppercase, "INSERT") ||
			strings.HasPrefix(uppercase, "UPDATE") ||
			strings.HasPrefix(uppercase, "DELETE")
		if !writes {
			continue
		}
		for _, table := range readOnlyTable {
			require.NotContainsf(t, uppercase, table,
				"kueri tulis %q menyentuh tabel milik sistem lain: %s", name, table)
		}
		require.Containsf(t, uppercase, "POOLDATA.MST_XOL_",
			"kueri tulis %q harus menyentuh salah satu tabel MST_XOL_*", name)
	}
}

// Penyisipan bergantung pada kunci baris untuk menutup cacat balapan `max+1` yang
// diwarisi `Database/INSERT_UPDATE_MST_XOL.prc:16` dan `:73`.
func TestInsertPathLocksRows(t *testing.T) {
	for _, name := range []string{"xol_lock_master_ids", "xol_lock_layer_ids"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "FOR UPDATE",
			"tanpa kunci ini, dua penyimpanan bersamaan dapat menghasilkan nomor kembar")
	}
}

// Kueri daftar SENGAJA tanpa ORDER BY — pengurutannya numerik dan dikerjakan di Go.
// Menambahkan ORDER BY di sini akan mengurutkan sebagai TEKS, sehingga "10010" mendahului
// "1009" dan daftarnya tampak melompat-lompat.
func TestKueriDaftarTanpaPengurutanTeks(t *testing.T) {
	for _, name := range []string{"xol_list", "xol_year_list"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "ORDER BY",
			"urutan kueri %q dikerjakan di Go supaya numerik dan portabel", name)
	}
}

// Kaskade hapus induk menempuh urutan yang mengikat: reas lebih dulu, baru lapisannya.
// Membalik keduanya menghilangkan satu-satunya cara mengetahui IDLAYER mana yang milik
// induk itu — dan reas-nya tertinggal sebagai baris yatim, persis cacat yang diperbaiki.
func TestHapusReasIndukBersandarPadaTabelLayer(t *testing.T) {
	text := strings.ToUpper(getQuery("xol_delete_reas_of_master"))
	require.Contains(t, text, "POOLDATA.MST_XOL_LAYER",
		"penghapusan reas satu induk harus mencari IDLAYER-nya lewat tabel layer")
}
