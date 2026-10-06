package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
)

// namedQueries adalah kesebelas kueri yang wajib ada di berkas .sql.
var namedQueries = []string{
	"list_tasks", "count_tabs",
	"resolve_surveyor", "resolve_members",
	"kpi_by_adjuster", "kpi_by_year",
	"check_tables", "check_columns", "check_new_columns", "check_filled_columns",
	"check_kpi",
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

// helperAliases adalah alias yang BUKAN kolom hasil.
//
// `STEP_RANK` hidup di dalam tampilan sebaris yang memilih langkah terakhir tiap berkas
// survei; ia disaring `WHERE STEP_RANK = 1` dan tidak pernah sampai ke pemanggil. Tanpa
// pengecualian ini, ia terbaca sebagai kolom hasil dan seluruh perbandingan urutan bergeser.
var helperAliases = map[string]bool{"STEP_RANK": true}

// aliasesOf mengambil alias `AS X` yang benar-benar menjadi kolom hasil, berurutan.
func aliasesOf(text string) []string {
	matches := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(text, -1)

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		alias := strings.ToUpper(match[1])
		if helperAliases[alias] {
			continue
		}
		result = append(result, alias)
	}
	return result
}

// availableTabs menyerahkan tab yang BENAR-BENAR dapat dihitung, dalam urutan tampilnya.
func availableTabs() []inboxsurvey.Tab {
	result := []inboxsurvey.Tab{}
	for _, tab := range inboxsurvey.Tabs() {
		if tab.Available() {
			result = append(result, tab)
		}
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

// TestAliasKueriHitungTabSamaDenganTabTersedia mengunci pasangan kolom dengan tab.
//
// `countColumns` dibaca berurutan terhadap tab TERSEDIA di dalam Repo.Counts. Satu kolom yang
// bergeser akan menukar jumlah "belum dibalas ASM" dengan "sudah dibalas ASM" — dua angka yang
// sama-sama masuk akal, sehingga tertukarnya tidak akan disadari siapa pun.
func TestAliasKueriHitungTabSamaDenganTabTersedia(t *testing.T) {
	require.Equal(t, countColumns, aliasesOf(query("count_tabs")))
	require.Len(t, countColumns, len(availableTabs()))
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

// TestKueriDaftarMembacaTabelPenggerakDanHeader mengunci keputusan Work Owner 2026-09-29.
//
// Baris digerakkan `T_SURVEYORLIST`, header klaim diambil dari `POOLDATA.T_CLAIM_PNC`, dan
// keduanya disambung `CLAIMID = PNCCASEID` — persis seperti
// `RDB List/BroswseKlaimByNoSurvey-SQL.xml` menyambungkannya.
func TestKueriDaftarMembacaTabelPenggerakDanHeader(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		text := strings.ToUpper(query(name))

		require.Containsf(t, text, "POOLDATA.T_SURVEYORLIST", "kueri %s", name)
		require.Containsf(t, text, "POOLDATA.T_CLAIM_PNC", "kueri %s", name)
		require.Containsf(t, text, "C.CLAIMID = S.PNCCASEID", "kueri %s", name)
	}
}

// TestTabelCerminTidakDipakai mengunci keputusan Work Owner 2026-09-30.
//
// `POOLDATA.T_CLAIM_SURVEY_DATAPEGA` sempat disambung LEFT JOIN untuk membawa `STATUSWORK`,
// lalu DILEPAS: tabelnya belum pernah terisi satu baris pun, dan keempat kolom yang ditunggu
// diminta langsung ke `T_SURVEYORLIST`.
//
// Uji ini menjaga SELURUH kueri, bukan hanya kueri daftar — satu kueri yang kelak ditambahkan
// dan diam-diam menyambungnya kembali akan bergantung pada tabel yang tidak pernah diisi.
func TestTabelCerminTidakDipakai(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, strings.ToUpper(text), "T_CLAIM_SURVEY_DATAPEGA",
			"kueri %s menyambung tabel cermin yang belum pernah terisi", name)
	}
}

// TestHanyaLangkahTerakhirYangDiambil mengunci perbaikan penggandaan baris.
//
// `T_SURVEYORLIST` adalah jejak perkembangan — diukur di produksi 2026-09-29: 17.641 baris
// untuk 2.448 berkas survei, rata-rata 7,21 langkah, terburuk 176. Tanpa penyaringan ini,
// layar menampilkan tujuh baris untuk setiap satu yang benar.
//
// `LPAD`, bukan `TO_NUMBER`: `to_number` khas Oracle dan melanggar `D-20`, dan ia menjatuhkan
// seluruh layar dengan ORA-01722 pada satu baris warisan yang indeksnya bukan angka.
//
// `NULLS LAST` disebut tegas: bawaan Oracle untuk `DESC` adalah `NULLS FIRST`, sehingga
// tanpanya baris ber-INDEX_SURVEY kosong terpilih sebagai "langkah terakhir".
func TestHanyaLangkahTerakhirYangDiambil(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		text := query(name)

		require.Containsf(t, text, "ROW_NUMBER() OVER (PARTITION BY t.CASEID",
			"kueri %s tidak memilih langkah terakhir", name)
		require.Containsf(t, text, "LPAD(TRIM(t.INDEX_SURVEY), 10, '0') DESC NULLS LAST",
			"kueri %s tidak mengurutkan langkah dengan benar", name)
		require.Containsf(t, text, "s.STEP_RANK = 1",
			"kueri %s tidak menyaring langkah terakhir", name)

		require.NotContainsf(t, strings.ToUpper(text), "TO_NUMBER(T.INDEX_SURVEY",
			"kueri %s memakai TO_NUMBER yang tidak portabel dan dapat gagal", name)
	}
}

// TestTidakSatuPunKueriMembacaTabelDatarMaupunTabelPega mengunci DUA ketetapan sekaligus.
//
// Work Owner 2026-09-28 mencabut dua tabel engine Pega:
//
//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK
//	DATAPEGA.PC_ASSIGN_WORKLIST
//
// Work Owner 2026-09-29 mencabut penggantinya yang pertama, `POOLDATA.T_CLAIMLIST_ADMIN`,
// khusus untuk modul ini — tabel datar itu hanya memuat klaim yang tugasnya berada di antrean
// Admin, sementara survei yang sedang berjalan berarti klaimnya sudah melewati tahap itu.
// `INNER JOIN` ke sana membuang justru baris yang dicari layar ini, tanpa satu pun galat.
//
// Uji ini menjaga SELURUH kueri, bukan hanya kueri daftar: satu kueri yang kelak ditambahkan
// dan diam-diam menembak salah satu tabel itu akan mengembalikan cacat yang sedang dilepas.
func TestTidakSatuPunKueriMembacaTabelDatarMaupunTabelPega(t *testing.T) {
	for name, text := range queries {
		upper := strings.ToUpper(text)

		require.NotContainsf(t, upper, "DATAPEGA.", "kueri %s membaca skema DATAPEGA", name)
		require.NotContainsf(t, upper, "PC_ASSIGN_WORK",
			"kueri %s membaca tabel penugasan Pega", name)
		require.NotContainsf(t, upper, "T_CLAIMLIST_ADMIN",
			"kueri %s membaca tabel datar yang membuang baris survei berjalan", name)
	}
}

// TestPenyaringKelasObjekKerjaTidakLagiDibutuhkan mendokumentasikan apa yang HILANG.
//
// `T_CLAIMLIST_ADMIN` mencampur `Work-PNC` dengan `Work-ReceiveDocument`, sehingga kueri di
// atasnya WAJIB menyaring `PXOBJCLASS`. `POOLDATA.T_CLAIM_PNC` tidak mencampur apa pun — ia
// tabel klaim PNC, dan `PXOBJCLASS` bahkan bukan kolomnya.
//
// Uji ini ada supaya hilangnya penyaring itu terbaca sebagai konsekuensi yang disadari, bukan
// sebagai penyaring yang lupa dibawa saat tabelnya berganti.
func TestPenyaringKelasObjekKerjaTidakLagiDibutuhkan(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		require.NotContainsf(t, strings.ToUpper(query(name)), "PXOBJCLASS",
			"kueri %s menyaring kolom yang tidak ada di T_CLAIM_PNC", name)
	}
}

// TestKolomHeaderMemakaiNamaTabelKlaim mengunci penamaan yang BERBEDA JAUH.
//
// Tidak satu pun nama kolom header sama antara kedua tabel, dan salah satu saja menjatuhkan
// seluruh layar dengan ORA-00904:
//
//	PYID -> CLAIMNO · POLICYNO -> NOPOLIS · DATEOFLOSS_1 -> DATEOFLOSS · USERTEKNIS_1 -> PICTEKNIK
//
// Uji ini menangkapnya tanpa basis data. Tanpa uji ini, satu-satunya cara mengetahuinya adalah
// membuka layarnya di lingkungan yang punya Oracle.
func TestKolomHeaderMemakaiNamaTabelKlaim(t *testing.T) {
	text := query("list_tasks")

	for _, benar := range []string{
		"c.CLAIMNO", "c.NOPOLIS", "c.QQNAME", "c.BUSINESSNAME", "c.PICTEKNIK", "c.DATEOFLOSS",
	} {
		require.Containsf(t, text, benar, "kueri daftar tidak mengambil %s", benar)
	}

	for _, lama := range []string{"PYID", "POLICYNO", "DATEOFLOSS_1", "USERTEKNIS_1", "PZINSKEY"} {
		require.NotContainsf(t, strings.ToUpper(text), lama,
			"kueri daftar masih memakai nama kolom tabel datar: %s", lama)
	}
}

// TestJenisSurveyorDibacaPerJanjiSurvei mengunci sumber kolomnya.
//
// `s.SURVEYTYPE` berlaku PER JANJI SURVEI. Jalur penulisnya terbaca utuh dari
// `ClaimData.SurveyData.SurveyorType` sampai `INSERT_SURVEYORLIST.prc:32`.
//
// Klaim dengan dua janji survei berjenis berbeda akan menampilkan jenis yang SAMA pada kedua
// barisnya bila kolomnya diambil dari sisi klaim. Itu tidak menghasilkan galat, dan tidak
// terlihat di layar.
func TestJenisSurveyorDibacaPerJanjiSurvei(t *testing.T) {
	text := query("list_tasks")

	require.Contains(t, text, "s.SURVEYTYPE")
	require.Contains(t, text, "AS SURVEYOR_TYPE")
	require.NotContains(t, strings.ToUpper(text), "SURVEYORTYPE_1")
}

// TestKolomYangBelumTerisiTidakDibacaKueriDaftar mengunci batas yang disadari.
//
// Kelimanya SUDAH ADA di `POOLDATA.T_SURVEYORLIST` tetapi seluruhnya masih kosong. Membacanya
// sekarang tidak menghasilkan galat apa pun — ia menjawab salah, dan itu lebih berbahaya.
//
// # Kenapa yang diperiksa `S.<kolom>`, bukan nama kolomnya saja
//
// Karena sejak penggantian nama 2026-10-05, nama kolom `ADJUSTER_PIC` **sama persis dengan
// alias** yang dipakai kueri daftar untuk kolom layar "PIC Loss Adjuster":
//
//	s.SURVEYOR_NAME     AS ADJUSTER_PIC
//
// Memeriksa namanya saja akan menolak alias yang memang benar. Yang hendak dicegah adalah
// **membaca kolomnya**, dan itu selalu berbentuk `s.<kolom>` — awalan tabel yang membedakan
// keduanya. Alias menamai kolom LAYAR; `s.` menamai kolom BASIS DATA.
//
// `ADJUSTERSTATUS_1` TIDAK ada di daftar ini, dan itu bukan kelalaian: `STS_SURVEY` terbukti
// membawa domainnya — dan kolom "Status ASM" ternyata bukan itu sama sekali, melainkan
// `LEADER_MEMBER`.
func TestKolomYangBelumTerisiTidakDibacaKueriDaftar(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	for _, kolom := range []string{
		"ADJUSTERACCEPT", "ADJUSTER_PIC", "REFNO", "PYSTATUSWORK", "RESCHEDULE_LOCATION",
	} {
		require.NotContainsf(t, text, "S."+kolom,
			"kueri daftar membaca s.%s sebelum keterisiannya terukur", kolom)
	}

	// Aliasnya JUSTRU harus ada — ia yang menamai kolom layar.
	require.Contains(t, text, "AS ADJUSTER_PIC")
}

// TestKueriPemeriksaMenyebutKelimaKolomYangDitunggu adalah kebalikan uji di atasnya.
//
// # Kenapa LIMA, dan kenapa angkanya pernah berubah dua kali
//
// Daftar ini EMPAT pada 2026-09-29, TIGA setelah `ADJUSTER_PIC` dicoret sebagai asal
// "Appointment No" (nomornya ternyata `CASEID`), lalu LIMA setelah keempat kueri tab tiba dan
// `ADJUSTER_PIC` kembali — kali ini sebagai asal "PIC Loss Adjuster", kolom yang berbeda —
// bersama `RESCHEDULE_LOCATION`.
//
// Setiap perubahan bersandar pada satu pengukuran. Uji ini menjaga daftarnya tetap sama dengan
// apa yang benar-benar ditunggu: menanyakan kolom yang tidak perlu membuat `-periksa`
// melaporkan modul belum siap padahal siap, dan melewatkan yang perlu menyembunyikan
// penghalang.
func TestKueriPemeriksaMenyebutKelimaKolomYangDitunggu(t *testing.T) {
	text := strings.ToUpper(query("check_new_columns"))

	// Nama TANPA akhiran `_1` — itu nama sebenarnya di basis data.
	for _, kolom := range []string{
		"ADJUSTERACCEPT", "REFNO", "PYSTATUSWORK", "ADJUSTER_PIC", "RESCHEDULE_LOCATION",
	} {
		require.Containsf(t, text, kolom, "kueri pemeriksa tidak menanyakan %s", kolom)
	}

	// Lewat KATALOG, bukan parsing: probe parsing bersifat semua-atau-tidak, sehingga satu
	// kolom yang belum ada membuat kolom lain yang sudah ada ikut terbaca belum ada.
	require.Contains(t, text, "ALL_TAB_COLUMNS",
		"kueri pemeriksa harus membaca katalog supaya dapat melapor per kolom")

	// Keberadaan dan keterisian diuji TERPISAH: yang pertama lewat katalog, yang kedua hanya
	// dapat diukur dengan membaca baris. Kolom yang ADA tetapi seluruhnya kosong akan membuat
	// tab Outstanding menampilkan seluruh antrean sebagai "belum dikonfirmasi adjuster".
	require.NotContains(t, strings.ToUpper(query("check_filled_columns")), "WHERE 1 = 0",
		"kueri keterisian tidak boleh disaring habis — ia harus benar-benar membaca baris")
}

// TestPredikatBerkasTutupTerekamDiDomain menjaga aturan yang disepakati Work Owner.
//
// Work Owner menegaskan 2026-09-29: berkas survei yang berjalan adalah yang status Pega-nya
// BUKAN `Resolved-Completed` maupun `Resolved-Rejected`.
//
// Keduanya diserahkan lewat SATU fungsi, bukan dua konstanta lepas: memasang salah satunya dan
// lupa yang lain menghasilkan antrean yang memuat berkas tutup, tanpa satu pun galat.
//
// Nilainya dikunci di sini karena penggantinya sudah diuji dan GAGAL: dari 1.070 berkas yang
// sudah `Resolved-*`, hanya 308 berakhir di `Close Case`/`Reject Case` — dan `Resolved-Rejected`
// tidak pernah sekali pun berpasangan dengan `Reject Case`.
func TestPredikatBerkasTutupTerekamDiDomain(t *testing.T) {
	require.Equal(t,
		[]string{"Resolved-Completed", "Resolved-Rejected"},
		inboxsurvey.ClosedWorkStatuses(),
		"predikat berkas tutup berubah tanpa disengaja")
}

// TestPICLossAdjusterAdalahSurveyorName mengunci identitas yang alias Pega-nya menyamarkan.
//
// Kolom layar "PIC Loss Adjuster" mengikat `.AnaylstRemarks`, dan `BrowseOSLossAdjusterPIC`
// mengisinya dari `SURVEYORNAME_1`. Jadi `SURVEYORNAME_1` = `ADJUSTER_PIC` = `SURVEYOR_NAME`
// — satu kolom, satu orang.
//
// Kolom itu dialiaskan TIGA nama berbeda di tiga rule (`AnaylstRemarks`, `CountryID`,
// `ComplianceRemark`), dan tidak satu pun mencerminkan isinya. Uji ini ada supaya pembaca
// berikutnya tidak menyimpulkan dari alias bahwa ketiganya kolom yang berbeda, lalu meminta
// `SURVEYORNAME_1` ke tabel cermin sebagai kolom keempat yang sebenarnya sudah ada.
func TestPICLossAdjusterAdalahSurveyorName(t *testing.T) {
	text := query("list_tasks")

	require.Contains(t, text, "s.SURVEYOR_NAME     AS ADJUSTER_PIC")

	// Ia juga yang dipakai menyaring cakupan — dan keduanya WAJIB kolom yang sama. Menyaring
	// dengan satu kolom lalu menggambar kolom lain akan menampilkan nama yang tidak
	// menjelaskan kenapa barisnya muncul.
	require.Contains(t, text, "UPPER(TRIM(s.SURVEYOR_NAME))")

	require.NotContains(t, strings.ToUpper(text), "SURVEYORNAME_1")
}

// TestStatusASMAdalahPeranKoasuransi mengunci koreksi 2026-10-03.
//
// # Apa yang dikunci, dan kenapa
//
// Kolom berjudul "Status ASM" **bukan status**. `ASMSTATUS_1` hanya bernilai `LEADER`,
// `MEMBER`, atau kosong — ia peran koasuransi ASM pada klaim itu.
//
// `T_CLAIM_PNC.LEADER_MEMBER` membawanya, diukur di produksi dengan **nol pertentangan** pada
// 17.633 baris. Uji ini mencegah kolomnya kembali diisi `STS_SURVEY`, yang merupakan status
// perkembangan adjuster dan **tidak pernah digambar Pega di grid ini** — `AdjusterStatus_1`
// dialiaskan `"CauseOfLossID"`, dan properti itu bukan salah satu dari 13 sel data.
//
// Uji sebelumnya di tempat ini mengunci yang SEBALIKNYA, lengkap dengan alasan yang terdengar
// meyakinkan. Itu uji yang mengunci perilaku keliru — jenis yang lebih buruk daripada tidak
// ada uji, karena ia membuat kekeliruan tampak disengaja.
func TestStatusASMAdalahPeranKoasuransi(t *testing.T) {
	text := query("list_tasks")

	require.Contains(t, text, "c.LEADER_MEMBER     AS ASM_STATUS")
	require.NotContains(t, text, "AS ASM_STATUS,\n       s.STS_SURVEY")
	require.NotContains(t, strings.ToUpper(text), "ADJUSTERSTATUS_1")
	require.NotContains(t, strings.ToUpper(text), "ASMSTATUS")
}

// TestHanyaTabTersediaPunyaCabangDiKueriDaftar menjaga kedua arah sekaligus.
//
// Tab tersedia TANPA cabang menghasilkan daftar kosong yang terbaca sebagai "tidak ada
// pekerjaan". Tab tak tersedia YANG PUNYA cabang berarti kuerinya membaca kolom yang tidak
// ada, dan seluruh layar jatuh.
func TestHanyaTabTersediaPunyaCabangDiKueriDaftar(t *testing.T) {
	text := query("list_tasks")

	for _, tab := range inboxsurvey.Tabs() {
		penanda := "'" + string(tab) + "'"

		if tab.Available() {
			require.Containsf(t, text, penanda,
				"tab %q tersedia tetapi tidak punya cabang di kueri daftar", tab)
			continue
		}
		require.NotContainsf(t, text, penanda,
			"tab %q belum tersedia tetapi punya cabang di kueri daftar", tab)
	}
}

// TestSetiapTabTakTersediaMenyebutSebabnya mengunci bahwa batasnya TERLIHAT.
//
// Tab yang belum dapat dihitung dan tidak menyebut sebabnya tidak dapat dibedakan dari tab
// kosong, dan tab kosong tidak pernah dilaporkan siapa pun sebagai kerusakan.
func TestSetiapTabTakTersediaMenyebutSebabnya(t *testing.T) {
	hasUnavailable := false

	for _, tab := range inboxsurvey.Tabs() {
		if tab.Available() {
			require.Emptyf(t, inboxsurvey.UnavailableReason(tab),
				"tab %q tersedia tetapi menyebut alasan tak tersedia", tab)
			continue
		}
		hasUnavailable = true
		require.NotEmptyf(t, inboxsurvey.UnavailableReason(tab),
			"tab %q belum tersedia tetapi tidak menyebut sebabnya", tab)
	}

	// Bila kelak keempat kolomnya tiba dan seluruh tab tersedia, uji ini kehilangan isinya.
	// Kegagalan di sini adalah TANDA BAIK: hapus uji ini, jangan longgarkan.
	require.True(t, hasUnavailable,
		"tidak ada lagi tab yang tak tersedia — kolom adjuster tampaknya sudah tiba; "+
			"hidupkan keempat tab dan hapus uji ini")
}

// TestTabBawaanYangDipakaiLayarDapatDihitung mencegah kesan pertama berupa layar kosong.
//
// DefaultTab adalah Outstanding, dan Outstanding termasuk yang belum dapat dihitung. Membuka
// layar di sana membuat setiap pengguna melihat layar tanpa isi lebih dulu — dan kesan itu
// bertahan meski tiga tab lain berisi.
func TestTabBawaanYangDipakaiLayarDapatDihitung(t *testing.T) {
	require.True(t, inboxsurvey.DefaultAvailableTab().Available(),
		"tab bawaan layar tidak dapat dihitung")
}

// TestNilaiBisnisTidakTertanamDiTeksSQL mengunci `D-15`.
//
// Kedua nilai `KOMUNIKASISTATUS` dikirim lewat bind, tidak ditulis di dalam teks SQL. Nilai
// bisnis yang tertanam di dalam SQL adalah bentuk paling sulit ditemukannya saat kebijakannya
// berubah.
func TestNilaiBisnisTidakTertanamDiTeksSQL(t *testing.T) {
	for _, name := range []string{"list_tasks", "count_tabs"} {
		text := query(name)

		require.NotContainsf(t, text, "KOMUNIKASISTATUS = '",
			"kueri %s menanamkan nilai status komunikasi", name)
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

// TestUrutanDaftarMenurun mengunci CATATAN 4 pada berkas .sql.
//
// # Uji ini sempat mengunci yang SEBALIKNYA
//
// Sampai 2026-10-03 ia bernama `TestUrutanDaftarMenaik` dan menolak `DESC`, atas alasan
// "`BrowseLossAdjuster` dan `BrowseInternalSurveyor` keduanya ASC". Alasan itu runtuh begitu
// keempat kueri tab diterima: **kedua kueri itu bukan penggerak grid ini**, dan keempat yang
// sebenarnya memakai `ROW_NUMBER() OVER (ORDER BY a.pxcreatedatetime DESC)`.
//
// Dicatat karena pelajarannya lebih mahal daripada perbaikannya: sebuah uji yang mengunci
// perilaku ke sumber yang salah **lebih buruk daripada tidak ada uji** — ia membuat perilaku
// yang keliru tampak disengaja, dan menolak perbaikannya.
//
// Pemutus serinya `CASEID` saja: sesudah penyaringan langkah terakhir, tepat satu baris
// tersisa per berkas survei, sehingga `INDEX_SURVEY` tidak lagi diperlukan.
func TestUrutanDaftarMenurun(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "ORDER BY S.TGLINPUT DESC NULLS LAST, S.CASEID DESC\n")
}

// TestCauseOfLossDibacaDariCoverageKlaim mengunci asal kolom "Cause Of Loss".
//
// Keempat kueri tab memakai subquery yang sama persis:
//
//	(select causeofloss from pooldata.t_claim_objectcoverage
//	  where a.caseid_1 = claimid fetch next 1 row only)
//
// BUKAN `T_SURVEYORLIST.LOSSTYPE`, yang dipakai sampai 2026-10-03 atas dugaan — dan sempat
// ditandai "menunggu konfirmasi DBA" padahal buktinya ada di kueri yang belum dibuka.
func TestCauseOfLossDibacaDariCoverageKlaim(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "POOLDATA.T_CLAIM_OBJECTCOVERAGE")
	require.Contains(t, text, "FETCH NEXT 1 ROW ONLY")
	require.NotContains(t, text, "S.LOSSTYPE",
		"LOSSTYPE bukan asal Cause Of Loss; lihat keempat kueri tab")
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

// TestNamaKolomTanpaAkhiranPerataanPega mengunci penamaan kolom di `T_SURVEYORLIST`.
//
// # Kenapa uji ini ada
//
// Kolom yang sama punya DUA nama. Di `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` ia berakhiran `_1` —
// artefak perataan objek kerja Pega — sedangkan di `POOLDATA.T_SURVEYORLIST` akhiran itu tidak
// dipakai. Analisis modul ini bersumber dari rule Pega, sehingga nama ber-`_1` adalah bentuk
// yang lebih dulu tertulis di mana-mana.
//
// Itu sudah menggigit sekali: teks yang tampil di layar menyebut `ADJUSTERACCEPT_1` "belum ada
// di POOLDATA.T_SURVEYORLIST" pada hari kolom `ADJUSTERACCEPT` justru baru ditambahkan ke sana.
// Orang yang menambahkannya membaca pesan itu sebagai pekerjaannya tidak terbaca sistem.
//
// Modul dashboardclaim SAH memakai akhiran `_1` karena ia memang membaca tabel datar Pega.
// Uji ini hanya mengikat kueri modul ini.
func TestNamaKolomTanpaAkhiranPerataanPega(t *testing.T) {
	flattened := []string{
		"ADJUSTERACCEPT_1", "ADJUSTERPIC_1", "REFNO_1", "ADJUSTERSTATUS_1", "SURVEYORNAME_1",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, column := range flattened {
			require.NotContainsf(t, upper, column,
				"kueri %s memakai nama Pega %q; di POOLDATA.T_SURVEYORLIST kolomnya TANPA "+
					"akhiran _1", name, column)
		}
	}
}

// TestTidakAdaCarriageReturnDiTeksKueri menutup cacat yang bergantung pada MESIN, bukan kode.
//
// Repository ini ber-`core.autocrlf=true`: berkas `.sql` yang sama ber-LF di satu checkout dan
// ber-CRLF di checkout lain. `\r` yang tersisa ikut terkirim ke Oracle — diperlakukan sebagai
// spasi putih, sehingga kuerinya TETAP JALAN dan cacatnya tidak pernah menampakkan diri selain
// lewat uji yang membandingkan teks.
//
// Itu sudah terjadi: uji urutan `ORDER BY` merah pada satu mesin dan hijau pada mesin lain,
// dengan isi berkas yang sama persis.
func TestTidakAdaCarriageReturnDiTeksKueri(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "\r",
			"kueri %s memuat carriage return; splitByName seharusnya membuangnya", name)
	}
}

// TestKeterisianDiukurUntukKelimaKolom menutup celah yang paling mudah terlewat.
//
// Kolom yang ADA tetapi KOSONG tidak lebih siap daripada kolom yang tidak ada — dan pada tab
// Outstanding ia lebih berbahaya, karena `ADJUSTERACCEPT IS NULL` bernilai benar untuk SELURUH
// antrean. Satu kolom yang tiba tanpa ikut diukur di sini akan terbaca siap.
//
// Kelimanya sudah ada di basis data per 2026-10-03, dan seluruhnya masih kosong.
func TestKeterisianDiukurUntukKelimaKolom(t *testing.T) {
	text := strings.ToUpper(query("check_filled_columns"))

	for _, column := range []string{
		"ADJUSTERACCEPT", "REFNO", "PYSTATUSWORK", "ADJUSTER_PIC", "RESCHEDULE_LOCATION",
	} {
		require.Containsf(t, text, "COUNT(S."+column+")",
			"kolom %s sudah ada di POOLDATA.T_SURVEYORLIST tetapi keterisiannya tidak diukur",
			column)
	}
}
