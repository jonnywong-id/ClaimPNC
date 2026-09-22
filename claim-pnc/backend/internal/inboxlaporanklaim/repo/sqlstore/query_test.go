package sqlstore

import (
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
		"claim_report_region_list",
		"claim_report_next_sequence",
		"claim_report_insert",
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
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
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

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for _, verb := range writing {
			if !strings.Contains(upperCase, verb) {
				continue
			}
			require.NotContainsf(t, upperCase, "PC_ASM_FW_GCNMFW_WORK",
				"kueri %q memuat %q dan menyentuh tabel milik Pega", name, strings.TrimSpace(verb))
		}
	}
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

// Kolom yang dibaca ketiga badan kueri harus sama banyak dan sama urutan: scanRow
// membacanya secara posisi, dan satu kolom yang bergeser TIDAK menghasilkan galat —
// hanya kolom yang berisi isi kolom sebelahnya.
func TestAllReadingQueriesSelectTheSameColumns(t *testing.T) {
	reference := selectedColumns(t, getQuery("claim_report_list_body"))
	require.NotEmpty(t, reference, "badan daftar tidak memilih satu kolom pun")

	for _, name := range []string{"claim_report_message_body", "claim_report_get_body"} {
		require.Equalf(t, reference, selectedColumns(t, getQuery(name)),
			"kueri %q memilih kolom yang berbeda dari badan daftar", name)
	}
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
