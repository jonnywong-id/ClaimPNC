package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
)

// namedQueries adalah kedelapan kueri yang wajib ada di berkas .sql.
var namedQueries = []string{
	"list_tasks", "count_tabs",
	"resolve_surveyor", "resolve_members",
	"kpi_by_adjuster", "kpi_by_year",
	"check_tables", "check_columns", "check_kpi",
}

func TestSetiapKueriBernamaAda(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong atau tidak ada", name)
	}
}

func TestTidakAdaKueriTakTerpakaiDiBerkasSQL(t *testing.T) {
	// Kueri yang menganggur adalah kode mati yang kelak dikira siap dipakai.
	require.Len(t, queries, len(namedQueries))
}

// aliasesOf mengambil seluruh alias `AS X` dari sebuah kueri, dalam urutan kemunculannya.
func aliasesOf(text string) []string {
	matches := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(text, -1)

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, strings.ToUpper(match[1]))
	}
	return result
}

// TestAliasKueriDaftarSamaDenganDaftarDanUrutannya menjaga tiga tempat tetap sepadan.
//
// Urutan kolom pada kueri, isi `taskColumns`, dan urutan pembacaan `scanTask` WAJIB sama.
// Satu kolom yang bergeser akan memindahkan nomor polis ke kolom nama tertanggung — dan
// keduanya bertipe teks, sehingga tidak ada satu pun galat yang muncul.
func TestAliasKueriDaftarSamaDenganDaftarDanUrutannya(t *testing.T) {
	require.Equal(t, taskColumns, aliasesOf(query("list_tasks")))
}

// TestAliasKueriHitungTabSamaDenganUrutanTab mengunci pasangan kolom dengan tab.
//
// `countColumns` dibaca berurutan terhadap `inboxsurvey.Tabs()` di dalam Repo.Counts. Satu
// kolom yang bergeser akan menukar jumlah tab Invoice dengan tab Close — dua angka yang
// sama-sama masuk akal, sehingga tertukarnya tidak akan disadari siapa pun.
func TestAliasKueriHitungTabSamaDenganUrutanTab(t *testing.T) {
	require.Equal(t, countColumns, aliasesOf(query("count_tabs")))
	require.Len(t, countColumns, len(inboxsurvey.Tabs()))
}

// TestAliasKueriKPISamaPadaKeduaBentuknya mengunci kedua ringkasan agar sebangun.
//
// Keduanya dibaca `scanKPI` yang SATU. Bila alias salah satunya bergeser, ringkasan per tahun
// akan membaca angka Immediate Advice sebagai Preliminary Advice — tanpa galat, karena
// keduanya bertipe angka.
func TestAliasKueriKPISamaPadaKeduaBentuknya(t *testing.T) {
	require.Equal(t, kpiColumns, aliasesOf(query("kpi_by_adjuster")))
	require.Equal(t, kpiColumns, aliasesOf(query("kpi_by_year")))
}

// TestKueriDaftarMembacaTabelPenggerakDanHeader mengunci keputusan Work Owner 2026-09-28.
//
// Baris digerakkan `T_SURVEYORLIST`, header klaim diambil dari `T_CLAIMLIST_ADMIN`, dan
// keduanya disambung `PNCCASEID = PZINSKEY`. Menggantinya dengan tabel Pega akan membaca
// tabel yang terbukti TIDAK memuat satu pun baris `Work-SurveyClaim`.
func TestKueriDaftarMembacaTabelPenggerakDanHeader(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "POOLDATA.T_SURVEYORLIST")
	require.Contains(t, text, "POOLDATA.T_CLAIMLIST_ADMIN")
	require.Contains(t, text, "K.PZINSKEY = S.PNCCASEID")

	// Tabel Pega TIDAK dibaca sama sekali — itu inti keputusannya.
	require.NotContains(t, text, "DATAPEGA.PC_ASM_FW_GCNMFW_WORK")
}

// TestTidakSatuPunKueriMembacaTabelPega mengunci ketetapan Work Owner 2026-09-28.
//
// DUA tabel engine Pega dicabut sekaligus, dan keduanya diganti `T_CLAIMLIST_ADMIN`:
//
//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK  -> POOLDATA.T_CLAIMLIST_ADMIN
//	DATAPEGA.PC_ASSIGN_WORKLIST     -> POOLDATA.T_CLAIMLIST_ADMIN.PXASSIGNEDOPERATORID
//
// Uji ini menjaga SELURUH kueri, bukan hanya kueri daftar: satu kueri yang kelak ditambahkan
// dan diam-diam menembak skema DATAPEGA akan mengembalikan ketergantungan yang sedang
// dilepas, dan tidak ada apa pun di layar yang menandakannya.
func TestTidakSatuPunKueriMembacaTabelPega(t *testing.T) {
	for name, text := range queries {
		upper := strings.ToUpper(text)

		require.NotContainsf(t, upper, "DATAPEGA.", "kueri %s membaca skema DATAPEGA", name)
		require.NotContainsf(t, upper, "PC_ASSIGN_WORK",
			"kueri %s membaca tabel penugasan Pega", name)
	}
}

// TestKueriMenyaringKelasObjekKerja mengunci pelajaran yang sudah dibayar sekali.
//
// `POOLDATA.T_CLAIMLIST_ADMIN` menampung DUA kelas objek kerja — 870 `Work-PNC` dan 142
// `Work-ReceiveDocument`. Induk sebuah janji survei SELALU `Work-PNC`, terbaca dari
// `BrowseLossAdjuster-SQL.xml` yang menyaring `z.pxobjclass = 'ASM-FW-GCNMFW-Work-PNC'`
// sebelum mencocokkan `a.caseid_1 = z.pzinskey`.
//
// Melupakan penyaring itu tidak menghasilkan galat apa pun — ia hanya mencampur berkas
// penerimaan dokumen ke dalam antrean survei bila ada satu saja `PNCCASEID` yang menunjuk ke
// sana. Pelajaran yang sama sudah dibayar sekali di `inboxmanagerreceivepucl`.
func TestKueriMenyaringKelasObjekKerja(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		require.Containsf(t, query(name), "k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
			"kueri %s tidak menyaring kelas objek kerja", name)
	}
}

// TestJenisSurveyorDibacaPerJanjiSurvei mengunci sumber kolomnya.
//
// `s.SURVEYTYPE` berlaku PER JANJI SURVEI; `k.SURVEYORTYPE_1` pada tabel klaim berlaku per
// klaim. Keduanya memang nilai yang sama — jalur penulisnya terbaca utuh dari
// `ClaimData.SurveyData.SurveyorType` sampai `INSERT_SURVEYORLIST.prc:32` — tetapi
// granularitasnya berbeda.
//
// Klaim dengan dua janji survei berjenis berbeda akan menampilkan jenis yang SAMA pada kedua
// barisnya bila kolomnya diambil dari sisi klaim. Itu tidak menghasilkan galat, dan tidak
// terlihat di layar.
func TestJenisSurveyorDibacaPerJanjiSurvei(t *testing.T) {
	text := query("list_tasks")

	require.Contains(t, text, "s.SURVEYTYPE            AS SURVEYOR_TYPE")
	require.NotContains(t, text, "k.SURVEYORTYPE_1")
}

// TestTabOutstandingMemakaiIsNullBukanTidakSamaDengan adalah uji yang paling mudah terbalik.
//
// `CountOSLostAdjuster` membelah antrean dengan `AdjusterAccept_1 IS NULL` versus `= '1'`.
// Menggantinya menjadi `<> '1'` akan memasukkan baris bernilai `'0'` ke tab Outstanding —
// baris yang di Pega tidak masuk tab mana pun, dan kolomnya terisi wajar sehingga
// tambahannya tidak akan disadari.
func TestTabOutstandingMemakaiIsNullBukanTidakSamaDengan(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "K.ADJUSTERACCEPT_1 IS NULL")
	require.NotContains(t, text, "K.ADJUSTERACCEPT_1 <>")
	require.NotContains(t, text, "K.ADJUSTERACCEPT_1 !=")
}

// TestKetujuhTabAdaDiKueriDaftar memastikan tidak ada tab yang kehilangan cabangnya.
//
// Tab tanpa cabang menghasilkan daftar KOSONG, bukan galat — dan daftar kosong terbaca
// sebagai "tidak ada pekerjaan", yang tidak pernah dilaporkan siapa pun sebagai kerusakan.
func TestKetujuhTabAdaDiKueriDaftar(t *testing.T) {
	text := query("list_tasks")

	for _, tab := range inboxsurvey.Tabs() {
		require.Containsf(t, text, "'"+string(tab)+"'",
			"tab %q tidak punya cabang di kueri daftar", tab)
	}
}

// TestNilaiBisnisTidakTertanamDiTeksSQL mengunci `D-15`.
//
// Ketiga nilai `ADJUSTERSTATUS_1` dan kedua nilai `KOMUNIKASISTATUS` dikirim lewat bind, tidak
// ditulis di dalam teks SQL. Nilai bisnis yang tertanam di dalam SQL adalah bentuk paling
// sulit ditemukannya saat kebijakannya berubah.
func TestNilaiBisnisTidakTertanamDiTeksSQL(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		text := query(name)

		require.NotContainsf(t, text, inboxsurvey.StatusInvoiceFee,
			"kueri %s menanamkan nilai status Invoice", name)
		require.NotContainsf(t, text, inboxsurvey.StatusCloseCase,
			"kueri %s menanamkan nilai status Close", name)
	}
}

// TestPembatasCakupanDipasangDiKeduaSisi mencegah kebocoran antar surveyor.
//
// Tanpa pembatas `|` di kedua sisi, "BUDI" cocok dengan "BUDIONO" — dan seorang surveyor
// melihat pekerjaan surveyor lain yang namanya kebetulan memuat namanya. Layarnya terisi
// wajar, dan tidak ada apa pun yang menandakannya.
func TestPembatasCakupanDipasangDiKeduaSisi(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		require.Containsf(t, query(name), "'|' || UPPER(TRIM(s.SURVEYOR_NAME)) || '|'",
			"kueri %s tidak memasang pembatas cakupan di kedua sisi", name)
	}

	for _, name := range []string{"kpi_by_adjuster", "kpi_by_year"} {
		require.Containsf(t, query(name), "'|' || UPPER(TRIM(d.ADJUSTER)) || '|'",
			"kueri %s tidak memasang pembatas cakupan di kedua sisi", name)
	}
}

// TestKPIDisaringCakupanJuga mencegah papan penilaian menjadi bocor.
//
// `DETAIL_KPI_ADJUSTER` memuat penilaian SELURUH adjuster. Tanpa penyaring cakupan, tab KPI
// berubah menjadi papan peringkat yang tidak pernah diminta siapa pun.
func TestKPIDisaringCakupanJuga(t *testing.T) {
	for _, name := range []string{"kpi_by_adjuster", "kpi_by_year"} {
		require.Containsf(t, strings.ToUpper(query(name)), "INSTR(:1",
			"kueri %s tidak menyaring cakupan", name)
	}
}

// TestRingkasanTahunanDikelompokkanPerTahun mengunci perilaku yang namanya menyesatkan.
//
// `GetSummaryKPIAdjusterKuartal` menyebut "Kuartal" tetapi mengelompokkan
// `to_char(tanggal,'yyyy')` — per TAHUN. Perilakunya yang dibawa, bukan namanya (`P-5`).
func TestRingkasanTahunanDikelompokkanPerTahun(t *testing.T) {
	text := strings.ToUpper(query("kpi_by_year"))

	require.Contains(t, text, "GROUP BY TO_CHAR(D.TANGGAL, 'YYYY')")
	require.NotContains(t, text, "GROUP BY D.ADJUSTER")
}

// TestUrutanDaftarMenaik mengunci CATATAN 3 pada berkas .sql.
//
// `BrowseLossAdjuster` dan `BrowseInternalSurveyor` keduanya `ORDER BY … ASC` — yang tertua
// lebih dulu, urutan antrean kerja. Ia BERBEDA dari inbox lain di aplikasi ini yang menurun,
// dan perbedaannya dibawa (`P-5`).
func TestUrutanDaftarMenaik(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "ORDER BY S.TGLINPUT, S.CASEID, S.INDEX_SURVEY")
	require.NotContains(t, text, "ORDER BY S.TGLINPUT DESC")
}

// TestTidakAdaPerangkaianNilaiKeTeksSQL menutup celah `{ASIS:...}` warisan.
//
// Sistem lama merangkai nilai pengguna ke dalam teks SQL di 538 tempat (§4.5). Tidak satu pun
// dibawa: seluruh nilai di sini lewat bind, termasuk daftar nama surveyor yang dikirim
// sebagai SATU teks berpembatas.
func TestTidakAdaPerangkaianNilaiKeTeksSQL(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola perangkaian warisan", name)
	}
}

// TestSeluruhKueriHanyaMembaca mengunci `P-1`.
//
// Seluruh tabel yang dibaca modul ini milik sistem lama, dan selama masa paralel setiap tabel
// hanya boleh ditulis SATU sistem.
func TestSeluruhKueriHanyaMembaca(t *testing.T) {
	forbidden := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, word := range forbidden {
			require.NotContainsf(t, upper, word,
				"kueri %s memuat pernyataan yang menulis: %s", name, strings.TrimSpace(word))
		}
	}
}
