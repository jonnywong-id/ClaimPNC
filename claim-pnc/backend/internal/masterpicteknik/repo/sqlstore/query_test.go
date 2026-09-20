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
		"technician_list",
		"technician_get",
		"technician_insert",
		"technician_update",
		"technician_check_table",
		"technician_check_view",
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

// Tidak boleh ada satu pun stored procedure dipanggil dari sini (`D-02`). Aturannya mudah
// dilanggar justru di modul ini, karena penulisan lamanya memang lewat
// POOLDATA.PEGA_MST_USER_TEKNIS.
func TestQueriesNeverCallStoredProcedure(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		require.NotContainsf(t, uppercase, "PEGA_MST_USER_TEKNIS",
			"kueri %q memanggil procedure lama; logikanya sudah naik ke Go (D-02, D-68)", name)
		require.NotContainsf(t, uppercase, "BEGIN ",
			"kueri %q memuat blok PL/SQL; logika bisnis tidak tinggal di basis data", name)
	}
}

// Nilai TIDAK PERNAH dirangkai ke dalam teks SQL. Rule lama menyisipkan
// `{ASIS:InputCOL.OPERATOR_ID}` langsung ke WHERE — celah injeksi yang persis tidak boleh
// terulang (utang teknis 4.5).
func TestQueriesUseBindParametersOnly(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, text, "{ASIS:",
			"kueri %q merangkai nilai ke dalam teks SQL", name)
		require.NotContainsf(t, text, "||'",
			"kueri %q merangkai nilai ke dalam teks SQL", name)
	}
}

// Daftar HARUS menyaring petugas aktif. Penyaring ini adalah keputusan Work Owner
// 2026-09-19 yang meniru `STS_AKTIF = '1'` pada Report Definition lama — menghapusnya akan
// menampilkan petugas nonaktif di grid tanpa ada yang menyadarinya.
func TestListQueryFiltersActiveOnly(t *testing.T) {
	list := strings.ToUpper(getQuery("technician_list"))
	require.Contains(t, list, "WHERE STS_AKTIF =")
	// Sandinya dikirim sebagai parameter, bukan ditulis di dalam teks SQL, supaya ia hidup
	// di satu tempat saja — masterpicteknik.ActiveCode.
	require.NotContains(t, list, "STS_AKTIF = '1'")
}

// Ambil satu baris TIDAK boleh menyaring status aktif. Tanpa aturan ini, petugas yang
// telanjur dinonaktifkan menjadi tidak terjangkau sama sekali dan tidak dapat diaktifkan
// kembali — `GetMasterPICTeknis` pun tidak menyaringnya.
func TestGetQueryDoesNotFilterActive(t *testing.T) {
	require.NotContains(t, strings.ToUpper(getQuery("technician_get")), "STS_AKTIF =")
}

// Daftar dibaca dari VIEW karena TOTAL_JOB tidak ada di tabelnya; penulisan selalu ke
// TABEL. Tertukarnya keduanya akan membuat aplikasi menulis ke objek yang tidak dapat
// ditulis, atau membaca daftar tanpa kolom beban kerja.
func TestListReadsViewWhileWritesTargetTable(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("technician_list")), "POOLDATA.V_MST_USER_TEKNIS")
	require.Contains(t, strings.ToUpper(getQuery("technician_list")), "TOTAL_JOB")

	for _, name := range []string{"technician_get", "technician_insert", "technician_update"} {
		uppercase := strings.ToUpper(getQuery(name))
		require.Containsf(t, uppercase, "POOLDATA.MST_USER_TEKNIK", "kueri %q harus menyentuh tabel", name)
		require.NotContainsf(t, uppercase, "V_MST_USER_TEKNIS", "kueri %q tidak boleh menyentuh view", name)
	}
}

// GROUPPANEL tidak pernah ditulis — procedure lama pun tidak, pada cabang INSERT maupun
// UPDATE. Menuliskannya berarti menambah perilaku yang tidak pernah ada.
func TestWritesNeverTouchPanelGroup(t *testing.T) {
	for _, name := range []string{"technician_insert", "technician_update"} {
		require.NotContainsf(t, strings.ToUpper(getQuery(name)), "GROUPPANEL",
			"kueri %q menulis GROUPPANEL", name)
	}
}

// OPERATOR_ID tidak pernah ikut diubah. Ia kunci alami yang disimpan setiap klaim;
// memindahkannya memutus rujukan penugasan pada data lama.
func TestUpdateNeverChangesOperatorID(t *testing.T) {
	update := strings.ToUpper(getQuery("technician_update"))
	setClause := update[strings.Index(update, "SET"):strings.Index(update, "WHERE")]
	require.NotContains(t, setClause, "OPERATOR_ID")
}
