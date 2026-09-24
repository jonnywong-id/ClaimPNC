package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxlaporanklaim"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"claim_report_source",
		"claim_report_list_body",
		"claim_report_count_body",
		"claim_report_message_body",
		"claim_report_message_count_body",
		"claim_report_summary_body",
		"claim_report_get_body",
		"claim_report_get_own_body",
		"claim_report_region_list",
		"claim_report_next_sequence",
		"claim_report_insert",
		"claim_report_update",
		"claim_report_check_table",
		"claim_report_check_legacy_table",
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

// sequenceQuery adalah satu-satunya kueri modul ini yang memuat sintaks khas Oracle.
//
// `SELECT seq.NEXTVAL FROM DUAL` tidak punya padanan langsung di PostgreSQL, yang
// memakai `nextval('seq')`. Ini PENGECUALIAN YANG SAMA yang `D-22` dan `D-71` akui untuk
// generator nomor klaim, dengan alasan yang sama: memaksakan portabilitas di sini
// mengorbankan jaminan keunikan nomor — pertukaran yang tidak sepadan.
//
// Ia disebut namanya di sini supaya pengecualiannya TERCATAT, bukan lolos diam-diam.
// Kueri lain yang kelak memakai FROM DUAL akan gagal pada uji di bawah.
const sequenceQuery = "claim_report_next_sequence"

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
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)

		if name != sequenceQuery {
			require.NotContainsf(t, upperCase, "FROM DUAL",
				"kueri %q memakai FROM DUAL; hanya %q yang dikecualikan", name, sequenceQuery)
		}
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, 538 kemunculan
// di seluruh export.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"claim_report_list_body",
		"claim_report_count_body",
		"claim_report_message_body",
		"claim_report_message_count_body",
		"claim_report_summary_body",
		"claim_report_get_body",
		"claim_report_insert",
		"claim_report_update",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Form Input Receive Document MENGISI berkas; ia tidak memindahkannya dan tidak menulis
// ulang jejaknya.
//
// Keenam kolom di bawah karena itu tidak boleh pernah muncul di klausa SET. Satu di
// antaranya yang tergeser sudah cukup merusak: `KODE_CABANG` adalah batas data seluruh
// layar ini, dan menulisnya dari form berarti berkas berpindah cabang tanpa satu pun
// tindakan yang menyatakannya.
func TestUpdateNeverTouchesProtectedColumns(t *testing.T) {
	setClause := getQuery("claim_report_update")
	if at := strings.Index(setClause, "\n WHERE "); at >= 0 {
		setClause = setClause[:at]
	}
	upper := strings.ToUpper(setClause)

	for column, reason := range map[string]string{
		"NO_LAPORAN":     "kunci baris; ia menyaring, tidak pernah berubah",
		"NO_KLAIM":       "terbit saat registrasi (B-2), bukan dari form ini",
		"KODE_CABANG":    "batas data; memindahkan berkas antarcabang bukan tindakan form ini",
		"STS_DISERAHKAN": "perpindahan tahap adalah tindakan tersendiri",
		"DIBUAT_OLEH":    "jejak pembuatan tidak pernah ditulis ulang",
		"DIHAPUS_PADA":   "penghapusan dinyatakan lewat penanda (ADR-0012), bukan di sini",
	} {
		require.NotContainsf(t, upper, column+" ",
			"klausa SET menyentuh %s — %s", column, reason)
	}
}

// Berkas yang sudah ditandai terhapus tidak boleh dapat disunting lewat alamat yang masih
// dipegang peramban seseorang.
func TestUpdateSkipsRowsMarkedDeleted(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("claim_report_update")), "DIHAPUS_PADA IS NULL",
		"penyimpanan tidak menyaring baris yang sudah ditandai terhapus")
}

// Modul ini TIDAK MENULIS satu baris pun ke tabel milik Pega.
//
// Ditetapkan Work Owner 2026-09-19, dan ia bukan sekadar preferensi: menulis ke tabel
// yang dibaca 116 rule Pega selama masa paralel melanggar penulis tunggal per tabel
// (ADR-0004) dan dapat menghentikan produksi yang masih dilayani Pega.
//
// Uji ini membaca SELURUH kueri dan memastikan tidak satu pun pernyataan pengubah
// menyentuh tabel warisan.
func TestNoQueryWritesToLegacyPegaTable(t *testing.T) {
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "}

	// Kedua tabel sekaligus: yang lama tidak boleh kembali, dan yang menggantikannya
	// diisi PROSES LAIN — aplikasi ini hanya membacanya (Work Owner, 2026-09-23).
	readOnly := []string{"PC_ASM_FW_GCNMFW_WORK", "T_CLAIMLIST_ADMIN"}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for _, verb := range writing {
			if !strings.Contains(upperCase, verb) {
				continue
			}
			for _, table := range readOnly {
				require.NotContainsf(t, upperCase, table,
					"kueri %q memuat %q dan menyentuh %s yang hanya boleh dibaca",
					name, strings.TrimSpace(verb), table)
			}
		}
	}
}

// Daftar ditarik dari POOLDATA.T_CLAIMLIST_ADMIN, bukan lagi dari tabel kerja Pega.
//
// Diuji pada SUMBERNYA, bukan pada satu badan kueri: seluruh tab, pencacah, dan ekspor
// menempel pada CTE yang sama, sehingga satu tempat inilah yang menentukan dari mana
// daftar berasal.
// Penanda bind harus MUNCUL dalam urutan menaik di teks kueri, tanpa nomor yang terlewat.
//
// # Kenapa ini invarian yang menentukan, bukan kerapian
//
// Oracle mengikat argumen menurut **urutan kemunculan** penanda di dalam teks kueri, bukan
// menurut angka pada `:n`. Penanda `:22` yang muncul lebih dulu daripada `:1` karena itu
// menerima argumen PERTAMA — dan seluruh argumen sesudahnya ikut bergeser.
//
// Itu benar-benar terjadi pada 2026-09-23: badan pencacah menaruh `:22..:30` di SELECT,
// sebelum `:1..:21` di WHERE. Akibatnya penyaring cabang menerima NULL dan pencacah
// komunikasi menerima kode cabang. Lencana di atas tab menyebut 115 sementara tabel di
// bawahnya kosong — **tanpa satu pun galat**, karena setiap bind tetap terisi sesuatu.
//
// Uji ini tidak dapat membuktikan argumennya benar; ia membuktikan penomorannya tidak lagi
// menyesatkan pembacanya. Itu yang gagal saat itu: penomorannya terbaca benar.
func TestBindMarkersAppearInAscendingOrder(t *testing.T) {
	marker := regexp.MustCompile(`:(\d+)`)

	for name, text := range query {
		seen := map[int]bool{}
		order := make([]int, 0, 40)

		for _, found := range marker.FindAllStringSubmatch(stripComments(text), -1) {
			n, err := strconv.Atoi(found[1])
			require.NoError(t, err)
			if seen[n] {
				continue
			}
			seen[n] = true
			order = append(order, n)
		}
		if len(order) == 0 {
			continue
		}

		for i, n := range order {
			require.Equalf(t, i+1, n,
				"kueri %q: penanda bind ke-%d yang muncul adalah :%d, seharusnya :%d — "+
					"Oracle mengikat menurut urutan kemunculan, bukan menurut nomornya",
				name, i+1, n, i+1)
		}
	}
}

// stripComments membuang baris komentar supaya contoh bind di dalam catatan tidak ikut
// terhitung sebagai penanda yang sesungguhnya.
func stripComments(text string) string {
	line := strings.Split(text, "\n")
	kept := make([]string, 0, len(line))
	for _, l := range line {
		if strings.HasPrefix(strings.TrimSpace(l), "--") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n")
}

// Baris daftar ditarik dari TABEL KERJA PEGA — tabel yang sama dengan yang dibaca layar
// lama, dan satu-satunya yang isinya cocok dengan angkanya.
//
// # Kenapa bukan T_CLAIMLIST_ADMIN
//
// Dibandingkan langsung pada 2026-09-23 dengan saringan Bisnis = NONMBU:
//
//	layar Pega                 ALL 671 · Outstanding 340 · Not Registered 123 · Not Transferred 42
//	tabel kerja Pega           ALL 671 · Outstanding 340 · Not Registered 123 · Not Transferred 42
//	T_CLAIMLIST_ADMIN          142 baris, `kodecabang_1` NULL pada SELURUHNYA
//
// Tabel admin adalah daftar pekerjaan outstanding, bukan daftar laporan yang utuh. Ia tetap
// dibaca sebagai LEFT JOIN untuk empat kolom yang hanya ada di sana.
func TestListIsDrawnFromThePegaWorkTable(t *testing.T) {
	source := strings.ToUpper(getQuery("claim_report_source"))

	require.Contains(t, source, "FROM DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
		"baris daftar tidak lagi ditarik dari tabel kerja Pega")
	require.NotContains(t, source, "FROM POOLDATA.T_CLAIMLIST_ADMIN",
		"tabel admin kembali menjadi tabel penggerak daftar — isinya hanya 5% dari daftar")
	require.Contains(t, source, "LEFT JOIN POOLDATA.T_CLAIMLIST_ADMIN",
		"tabel admin tidak lagi dibaca untuk sts_aktif, aging, kurir, dan keterangan")

	// Penyaring jenis case DIPERTAHANKAN dari kueri lama: tabel ini memuat seluruh case
	// Pega, bukan hanya Receive Document.
	require.Contains(t, source, "ASM-FW-GCNMFW-WORK-RECEIVEDOCUMENT",
		"penyaring pxobjclass hilang saat sumbernya ditukar")
}

// Posisi punya EMPAT keadaan, dan yang keempat tidak masuk tab mana pun.
//
// # Kenapa ini diuji
//
// Pemetaan sebelumnya menaruh seluruh sisanya di `ELSE` sebagai "Not Transferred". Itu
// terbaca benar dan tidak menghasilkan galat apa pun — tetapi tab itu menyebut 208 di
// tempat layar lama menyebut 42, karena 166 baris ber-nomor klaim tetapi belum terkunci
// ikut tersapu ke sana.
//
// Keempat kombinasinya dihitung langsung pada 2026-09-23 dan sama persis dengan layar lama:
// 42 · 123 · 340 · 166, berjumlah 671.
func TestPositionHasFourStatesNotThree(t *testing.T) {
	source := strings.ToUpper(getQuery("claim_report_source"))

	for _, branch := range []string{
		"WHEN W.PNCCASEID IS NOT NULL AND W.STATUSLOCK_1 IS NOT NULL THEN 'OUTSTANDING'",
		"WHEN W.PNCCASEID IS NULL     AND W.STATUSLOCK_1 IS NOT NULL THEN 'NOT REGISTERED'",
		"WHEN W.PNCCASEID IS NULL     AND W.STATUSLOCK_1 IS NULL     THEN 'NOT TRANSFERRED'",
	} {
		require.Containsf(t, source, branch, "cabang posisi hilang: %q", branch)
	}

	// Yang keempat TIDAK boleh dipaksakan ke salah satu tab.
	require.NotContains(t, source, "ELSE 'NOT TRANSFERRED'",
		"baris ber-nomor klaim yang belum terkunci ikut tersapu ke tab Not Transferred")
}

// Baris ber-STS_AKTIF '0' tidak ditampilkan (Work Owner, 2026-09-23), dan yang
// dikecualikan HANYA yang bernilai '0' secara tegas.
//
// Kolomnya nullable. Menyaring dengan `= '1'` akan ikut menyembunyikan baris yang
// penandanya belum ditetapkan — menghilangkan pekerjaan dari layar tanpa seorang pun tahu.
func TestOnlyExplicitlyInactiveRowsAreHidden(t *testing.T) {
	source := strings.ToUpper(getQuery("claim_report_source"))

	require.Contains(t, source, "STS_AKTIF IS NULL OR TRIM(T.STS_AKTIF) <> '0'")
	require.NotContains(t, source, "T.STS_AKTIF = '1'",
		"penyaring menyembunyikan baris yang penandanya belum ditetapkan")
}

// Asal sebuah berkas kini diturunkan dari AWALAN NOMOR, bukan dari tabel asalnya —
// tabelnya sudah satu.
//
// Awalan itu hidup di dua tempat: konstanta domain dan teks SQL. Uji ini yang menjaga
// keduanya tidak berpisah diam-diam; kalau berpisah, setiap berkas terbaca sebagai milik
// Pega dan form membukanya baca-saja tanpa satu pun galat.
func TestOriginIsDerivedFromTheNumberPrefix(t *testing.T) {
	source := getQuery("claim_report_source")

	require.Contains(t, source, inboxlaporanklaim.ReportNumberPrefix+".",
		"awalan nomor pada SQL tidak lagi sama dengan ReportNumberPrefix")
	require.Contains(t, source, "'claimpnc'")
	require.Contains(t, source, "'pega'")
}

// Berkas terbitan aplikasi ini dibaca dari tabelnya sendiri, dan pembacaannya menghormati
// soft delete (ADR-0012).
//
// Jalur ini ada supaya berkas yang BARU DIBUAT dapat dibuka sebelum proses pengisi
// T_CLAIMLIST_ADMIN menyalinnya. Tanpa itu, "Buat Baru" membuka form yang menjawab
// "tidak ditemukan".
func TestOwnReportIsReadFromItsOwnTable(t *testing.T) {
	own := strings.ToUpper(getQuery("claim_report_get_own_body"))

	require.Contains(t, own, "POOLDATA.CPNC_LAPORAN_KLAIM")
	require.NotContains(t, own, "T_CLAIMLIST_ADMIN",
		"pembacaan berkas sendiri ikut bergantung pada tabel yang diisi proses lain")
	require.Contains(t, own, "DIHAPUS_PADA IS NULL",
		"pembacaan berkas sendiri tidak menyaring baris yang ditandai terhapus")
}

// Kedua jalur pembacaan satu berkas dibaca scanDetailRow yang sama, dan ia membaca secara
// POSISI. Kolom yang bergeser tidak menghasilkan galat — hanya kolom yang berisi isi
// kolom sebelahnya.
func TestBothDetailQueriesSelectTheSameColumns(t *testing.T) {
	require.Equal(t,
		selectedColumns(t, getQuery("claim_report_get_body")),
		selectedColumns(t, getQuery("claim_report_get_own_body")),
		"kedua jalur pembacaan berkas memilih kolom yang berbeda")
}

// Penghapusan fisik data bernilai bisnis dilarang (ADR-0012). Layar ini pun tidak
// menghapus apa pun; kalau kelak ia menghapus, itu harus menjadi keputusan sadar — bukan
// satu baris yang menyelinap masuk.
func TestNoQueryDeletesRows(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE FROM",
			"kueri %q menghapus baris", name)
	}
}

// Kedua badan DAFTAR harus memilih kolom yang sama banyak dan sama urutan: keduanya
// dibaca scanRow, yang membaca secara POSISI — satu kolom yang bergeser tidak
// menghasilkan galat, hanya kolom yang berisi isi kolom sebelahnya.
func TestBothListQueriesSelectTheSameColumns(t *testing.T) {
	reference := selectedColumns(t, getQuery("claim_report_list_body"))
	require.NotEmpty(t, reference, "badan daftar tidak memilih satu kolom pun")

	require.Equal(t, reference, selectedColumns(t, getQuery("claim_report_message_body")),
		"badan daftar komunikasi memilih kolom yang berbeda dari badan daftar biasa")
}

// Badan DETAIL memilih lebih banyak kolom, dan itu disengaja — hanya form yang
// membutuhkan isian berkas, dan dua di antaranya berlebar 4.000 karakter.
//
// Yang dijaga uji ini: urutan kolom daftar tetap menjadi AWALAN kolom detail. scanRow dan
// scanDetailRow membaca posisi yang sama untuk kolom yang sama, sehingga menyisipkan
// kolom baru di tengah — bukan di ujung — akan menggeser salah satunya tanpa galat.
func TestDetailQueryExtendsTheListColumnsWithoutReordering(t *testing.T) {
	list := selectedColumns(t, getQuery("claim_report_list_body"))
	detail := selectedColumns(t, getQuery("claim_report_get_body"))

	require.Greater(t, len(detail), len(list), "badan detail tidak memilih kolom tambahan")

	// last_message selalu kolom TERAKHIR pada keduanya; ia dibandingkan terpisah.
	require.Equal(t, "last_message", list[len(list)-1])
	require.Equal(t, "last_message", detail[len(detail)-1])

	require.Equal(t, list[:len(list)-1], detail[:len(list)-1],
		"urutan kolom daftar bukan lagi awalan kolom detail")
}

// Penyaring pada badan daftar dan badan pencacahnya harus sama persis. Bila berbeda,
// lencana di atas tabel menyebut angka yang tidak berhubungan dengan isinya.
func TestCountQueryFiltersExactlyLikeItsListQuery(t *testing.T) {
	pair := map[string]string{
		"claim_report_list_body":    "claim_report_count_body",
		"claim_report_message_body": "claim_report_message_count_body",
	}
	for list, count := range pair {
		require.Equalf(t, whereClause(getQuery(list)), whereClause(getQuery(count)),
			"penyaring %q dan %q berbeda", list, count)
	}
}

// Kedelapan pencacah harus ada, dan namanya harus cocok dengan urutan Scan di
// Summarize. Uji ini menjaga jumlahnya; urutannya dijaga nama aliasnya.
func TestSummaryReturnsAllEightCounters(t *testing.T) {
	text := strings.ToUpper(getQuery("claim_report_summary_body"))
	for _, alias := range []string{
		"AS TOTAL", "AS NOT_TRANSFERRED", "AS UNREGISTERED", "AS OUTSTANDING",
		"AS ACCEPTED", "AS MESSAGE_UNANSWERED", "AS MESSAGE_WAITING", "AS MESSAGE_REPLIED",
	} {
		require.Containsf(t, text, alias, "pencacah %q hilang dari kueri ringkasan", alias)
	}
}

// Ketiga teks posisi ditulis di SQL maupun di Go. Uji ini menjaga keduanya tetap sama —
// tanpa itu, CASE WHEN di basis data dapat menghasilkan teks yang tidak pernah cocok
// dengan pembanding di Go, dan setiap tab akan tampak kosong tanpa satu pun galat.
func TestPositionTextInSQLMatchesDomain(t *testing.T) {
	source := getQuery("claim_report_source")
	for _, position := range []inboxlaporanklaim.Position{
		inboxlaporanklaim.PositionOutstanding,
		inboxlaporanklaim.PositionNotRegistered,
		inboxlaporanklaim.PositionNotTransferred,
	} {
		require.Containsf(t, source, "'"+string(position)+"'",
			"teks posisi %q tidak ada di kueri sumber", position)
	}
}

// Setiap daftar kode selalu dikirim sebagai satu penjaga diikuti empat kode, berapa pun
// panjang aslinya — bentuk kueri tetap, sehingga jumlah bind-nya harus tetap pula.
func TestEveryBusinessLineSendsTheSameNumberOfBinds(t *testing.T) {
	want := len(scopeArguments(inboxlaporanklaim.Filter{}))

	for _, info := range inboxlaporanklaim.ListBusinessLines() {
		got := len(scopeArguments(inboxlaporanklaim.Filter{BusinessLine: info.Line}))
		require.Equalf(t, want, got, "lini %q mengirim %d bind, bukan %d", info.Line, got, want)
	}
}

// Daftar kode yang lebih pendek dari empat dipadatkan dengan mengulang kode terakhirnya.
// `IN ('002','002','002','002')` berarti sama persis dengan `IN ('002')`.
func TestShortCodeListIsPaddedWithoutChangingMeaning(t *testing.T) {
	filter := inboxlaporanklaim.Filter{BusinessLine: inboxlaporanklaim.BusinessLinePA}
	argument := scopeArguments(filter)

	// Enam bind pertama adalah cabang, kanwil, dan kata kunci; daftar Group Panel mulai
	// di bind ketujuh sebagai penjaga.
	guard, codes := argument[6], argument[7:11]
	require.Equal(t, "002", guard, "penjaga Group Panel bukan kode pertamanya")
	for i, code := range codes {
		require.Equalf(t, "002", code, "kode Group Panel ke-%d = %v", i+1, code)
	}
}

// Pilihan "seluruh bisnis" harus mengirim NULL pada setiap penjaga, sehingga tak satu pun
// penyaring lini bisnis berlaku.
func TestAllBusinessLinesSendNilGuards(t *testing.T) {
	argument := scopeArguments(inboxlaporanklaim.Filter{BusinessLine: inboxlaporanklaim.BusinessLineAll})
	for _, index := range []int{6, 11, 16} {
		require.Nilf(t, argument[index], "penjaga pada bind ke-%d tidak NULL", index+1)
	}
}

// selectedColumns mengambil daftar kolom di antara SELECT dan FROM terluar.
func selectedColumns(t *testing.T, text string) []string {
	t.Helper()

	upper := strings.ToUpper(text)
	start := strings.Index(upper, "SELECT ")
	require.GreaterOrEqual(t, start, 0, "kueri tanpa SELECT")

	// FROM terluar adalah yang berada di awal baris; FROM di dalam subkueri selalu
	// menjorok lebih dalam.
	end := strings.Index(text[start:], "\n  FROM ")
	require.Greater(t, end, 0, "kueri tanpa FROM di tingkat terluar")

	var column []string
	depth := 0
	current := strings.Builder{}
	for _, r := range text[start+len("SELECT ") : start+end] {
		switch {
		case r == '(':
			depth++
		case r == ')':
			depth--
		case r == ',' && depth == 0:
			column = append(column, normaliseColumn(current.String()))
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if last := normaliseColumn(current.String()); last != "" {
		column = append(column, last)
	}
	return column
}

// normaliseColumn menyisakan nama alias kolomnya saja.
func normaliseColumn(text string) string {
	clean := strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if clean == "" {
		return ""
	}
	if at := strings.LastIndex(strings.ToUpper(clean), " AS "); at >= 0 {
		return strings.TrimSpace(clean[at+4:])
	}
	if at := strings.LastIndex(clean, "."); at >= 0 {
		return strings.TrimSpace(clean[at+1:])
	}
	return clean
}

// whereClause mengambil badan WHERE tanpa nomor bind, supaya dua kueri yang penyaringnya
// sama tetapi nomor bind-nya berbeda tetap terbaca sama.
func whereClause(text string) string {
	at := strings.Index(text, "\n WHERE ")
	if at < 0 {
		return ""
	}
	body := text[at:]
	if end := strings.Index(body, "\n ORDER BY "); end >= 0 {
		body = body[:end]
	}
	if end := strings.Index(body, "\nOFFSET "); end >= 0 {
		body = body[:end]
	}

	// Nomor bind dibuang: `:12` dan `:30` adalah penyaring yang sama pada kueri yang
	// berbeda panjangnya.
	var clean strings.Builder
	skipping := false
	for _, r := range body {
		if r == ':' {
			skipping = true
			clean.WriteRune(':')
			continue
		}
		if skipping && r >= '0' && r <= '9' {
			continue
		}
		skipping = false
		clean.WriteRune(r)
	}
	return strings.Join(strings.Fields(clean.String()), " ")
}
