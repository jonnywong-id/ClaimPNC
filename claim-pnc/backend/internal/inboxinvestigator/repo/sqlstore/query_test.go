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
	"investigator_export",
	"investigasi_ambil",
	"investigasi_perbarui",
	"investigasi_sisip",
	"investigasi_pindahkan_klaim",
	"investigasi_check_table",
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

// HANYA kueri formulir investigasi yang boleh menulis.
//
// # Kenapa daftarnya disebut satu per satu
//
// Modul ini sempat tidak punya jalur tulis sama sekali, dan uji ini melarang SELURUH
// penulisan. Larangan itu dicabut saat formulir kerja Investigator dibangun — tetapi
// dicabut **untuk tiga kueri yang disebut namanya**, bukan untuk modulnya.
//
// Bedanya menentukan. Kueri daftar, kueri ekspor, dan kedua kueri pencacah membaca tabel
// milik ENGINE PEGA (`PC_ASM_FW_GCNMFW_WORK`, `PC_ASSIGN_WORKBASKET`) dan tabel bisnis yang
// ditulis Pega (`JSON_KLAIM`). Satu UPDATE yang menyelinap ke sana melanggar `P-1` tanpa
// menghasilkan galat apa pun — kerusakannya baru terlihat sebagai data yang bertentangan
// antara dua sistem.
//
// Ketiga kueri yang dikecualikan menulis ke tabel yang kepemilikannya JELAS:
//
//	investigasi_perbarui · investigasi_sisip     POOLDATA.TC_PNC_INVESTIGASI — tabel milik
//	                                             aplikasi ini sepenuhnya, tidak pernah
//	                                             ditulis Pega
//	investigasi_pindahkan_klaim                  POOLDATA.T_CLAIM_PNC — sudah ditulis
//	                                             aplikasi ini lewat modul Registrasi
func TestOnlyInvestigationQueriesWrite(t *testing.T) {
	allowed := map[string]bool{
		"investigasi_perbarui":        true,
		"investigasi_sisip":           true,
		"investigasi_pindahkan_klaim": true,
	}
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range query {
		if allowed[name] {
			continue
		}
		uppercase := strings.ToUpper(text)
		for _, verb := range writing {
			require.NotContainsf(t, uppercase, verb,
				"kueri %q menulis (%s) — hanya kueri formulir investigasi yang boleh; "+
					"lihat catatan pada uji ini", name, verb)
		}
	}
}

// Ketiga kueri yang boleh menulis TIDAK boleh menyentuh tabel milik Pega.
//
// Uji ini pasangan dari yang di atas, dan ia yang menjaga pengecualiannya tetap sempit:
// mencantumkan nama kueri ke dalam daftar yang diizinkan tidak boleh sekaligus membuka
// tabel mana pun untuknya.
func TestWritingQueriesNeverTouchPegaTables(t *testing.T) {
	pegaOwned := []string{
		"PC_ASM_FW_GCNMFW_WORK",
		"PC_ASSIGN_WORKBASKET",
		"JSON_KLAIM",
		"T_SURVEYORLIST",
	}

	for _, name := range []string{
		"investigasi_perbarui", "investigasi_sisip", "investigasi_pindahkan_klaim",
	} {
		uppercase := strings.ToUpper(getQuery(name))
		for _, table := range pegaOwned {
			require.NotContainsf(t, uppercase, table,
				"kueri tulis %q menyentuh %s, tabel yang dimiliki Pega (`P-1`)", name, table)
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
		"BUSINESS_LINE",
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
