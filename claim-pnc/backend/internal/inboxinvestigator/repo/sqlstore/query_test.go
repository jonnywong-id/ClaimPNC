package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxinvestigator"
)

// usedQueries adalah seluruh kueri yang dipanggil kode modul ini.
var usedQueries = []string{
	"investigator_inbox_list",
	"investigator_inbox_search",
	"investigator_inbox_check_table",
	"investigator_inbox_count_waiting",
	"investigator_inbox_count_without_survey",
}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	for _, name := range usedQueries {
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
		"SELECT *":  "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":      "pakai COALESCE",
		"SYSDATE":   "pakai CURRENT_TIMESTAMP",
		"DECODE(":   "pakai CASE WHEN",
		"ROWNUM":    "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":    "pakai POSITION",
		"LISTAGG(":  "pakai STRING_AGG",
		"TO_CHAR(":  "pemformatan tanggal dan angka dilakukan di Go",
		"LPAD(":     "pemformatan angka dilakukan di Go",
		"FROM DUAL": "tidak ada kueri modul ini yang membutuhkannya",
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

// Tidak satu pun kueri modul ini boleh MENULIS.
//
// Layar ini tidak mengubah apa pun: mengambil pekerjaan dari antrean dan mencatat hasil
// investigasi terjadi di layar kerja yang belum dibangun. Selama itu benar, `P-1` terpenuhi
// tanpa negosiasi kepemilikan — Pega tetap satu-satunya penulis tabelnya sendiri (`D-21`).
//
// Uji ini yang menjaganya tetap begitu. Ia akan gagal pada hari seseorang menambahkan
// UPDATE ke berkas .sql tanpa memindahkan kepemilikan tabelnya lebih dulu.
func TestNoQueryWrites(t *testing.T) {
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for _, verb := range writing {
			require.NotContainsf(t, uppercase, verb,
				"kueri %q menulis (%s) — modul ini hanya membaca; lihat banner paket", name, verb)
		}
	}
}

// Kedua kueri daftar HARUS menyaring workbasket dan status kerja.
//
// Keduanya adalah penyaring yang membuat layar ini menjadi Inbox Investigator dan bukan
// daftar seluruh klaim. Yang ketinggalan tidak menghasilkan galat — ia menghasilkan daftar
// yang terlihat masuk akal dan memuat pekerjaan peran lain.
func TestListQueriesFilterByWorkbasketAndStatus(t *testing.T) {
	for _, name := range []string{"investigator_inbox_list", "investigator_inbox_search"} {
		t.Run(name, func(t *testing.T) {
			uppercase := strings.ToUpper(getQuery(name))
			require.Contains(t, uppercase, "PXASSIGNEDOPERATORID",
				"kueri wajib menyaring workbasket")
			require.Contains(t, uppercase, "RESOLVED-COMPLETED",
				"kueri wajib membuang pekerjaan yang sudah selesai")
			require.Contains(t, uppercase, "ASM-FW-GCNMFW-WORK-PNC",
				"kueri wajib membatasi kelas kasus; satu tabel Pega memuat banyak kelas")
			require.Contains(t, uppercase, "ASSIGN-WORKBASKET",
				"kueri wajib membatasi jenis penugasan")
		})
	}
}

// Nama workbasket TIDAK boleh tertulis di dalam teks SQL.
//
// Ia dikirim sebagai parameter meski nilainya konstanta di dalam kode kita sendiri.
// Larangan merangkai nilai ke dalam teks SQL tidak mengenal pengecualian "nilainya toh dari
// kode sendiri" — aturan yang berlaku kadang-kadang bukan aturan
// (`08-TECHNICAL-STRATEGY.md` §4.3).
func TestWorkbasketNameNeverAppearsInSQLText(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text),
			strings.ToUpper(inboxinvestigator.Workbasket),
			"kueri %q menuliskan nama workbasket ke dalam teks SQL; kirim sebagai parameter",
			name)
	}
}

// Ketiga kueri yang barisnya dipindai scanTask HARUS mengembalikan kesepuluh alias yang sama
// pada urutan yang sama.
//
// Satu pemindai melayani ketiganya. Alias yang berbeda urutannya tidak menghasilkan galat
// kompilasi maupun galat runtime — ia menghasilkan Nama Cabang yang muncul di kolom Nama
// Admin, dan tidak ada apa pun di layar yang menandakannya.
func TestScannedQueriesShareTheSameColumnOrder(t *testing.T) {
	expected := []string{
		"REFERENCE",
		"CASE_NUMBER",
		"POLICY_NUMBER",
		"INSURED_NAME",
		"PARTICIPANT_NAME",
		"BUSINESS_NAME",
		"BRANCH_NAME",
		"ADMIN_NAME",
		"REGISTERED_AT",
		"SURVEY_DATE",
	}

	scanned := []string{
		"investigator_inbox_list",
		"investigator_inbox_search",
		"investigator_inbox_check_table",
	}

	for _, name := range scanned {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, aliasOrder(getQuery(name)))
		})
	}
}

// aliasOrder membaca alias `AS <NAMA>` sesuai urutan kemunculannya di teks kueri.
//
// Hanya alias pada tingkat SELECT terluar yang dihitung; subquery di berkas ini tidak
// memakai `AS` sama sekali, sehingga pemindaian sederhana ini cukup dan tidak menuntut
// pengurai SQL.
func aliasOrder(text string) []string {
	var result []string
	for _, line := range strings.Split(strings.ToUpper(text), "\n") {
		index := strings.LastIndex(line, " AS ")
		if index < 0 {
			continue
		}
		alias := strings.TrimSpace(line[index+len(" AS "):])
		alias = strings.TrimSuffix(alias, ",")
		if alias != "" {
			result = append(result, alias)
		}
	}
	return result
}
