package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Uji di berkas ini TIDAK menyentuh basis data. Yang diperiksa adalah kesesuaian antara
// teks SQL, daftar alias di query.go, dan pemindai di inboxsalvage.go — tiga tempat yang
// mudah bergeser satu sama lain tanpa ada yang gagal sampai kuerinya benar-benar dijalankan
// terhadap Oracle.

func TestEveryQueryNamedInTheCodeExists(t *testing.T) {
	names := []string{
		"list_claim", "list_claim_search",
		"list_claim_object", "list_claim_object_search",
		"list_claim_buyback", "list_claim_buyback_search",
		"list_salvage",
		"count_claim_status", "count_claim_buyback",
		"count_salvage_status", "count_salvage_detail_unsold",
		"next_salvage_id", "insert_salvage", "update_salvage",
		"count_salvage_detail_for", "insert_salvage_detail",
		"check_salvage_columns",
	}

	for _, name := range names {
		require.NotPanics(t, func() { query(name) }, "kueri %q tidak ada", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)))
	}
}

// Komentar TIDAK ikut dikirim ke basis data: yang dibaca DBA adalah berkasnya, bukan jejak
// di jurnal basis data.
func TestCommentsAreStrippedFromTheStatementsSent(t *testing.T) {
	for name, text := range queries {
		for _, line := range strings.Split(text, "\n") {
			require.False(t, strings.HasPrefix(strings.TrimSpace(line), "--"),
				"kueri %q masih membawa baris komentar", name)
		}
	}
}

// Urutan alias di SQL WAJIB sama dengan daftar di query.go, dan daftar itu pula yang
// menjadi urutan pemindai. Satu kolom yang tersisip di tengah akan menggeser SELURUH isi
// baris tanpa satu pun galat — kelas cacat yang tidak menghasilkan tanda apa pun.
func TestAliasOrderInSQLMatchesTheListsInGo(t *testing.T) {
	cases := []struct {
		queries []string
		aliases []string
	}{
		{[]string{"list_claim", "list_claim_search"}, claimColumns},
		{
			[]string{
				"list_claim_object", "list_claim_object_search",
				"list_claim_buyback", "list_claim_buyback_search",
			},
			claimObjectColumns,
		},
		{[]string{"list_salvage"}, salvageColumns},
	}

	for _, testCase := range cases {
		for _, name := range testCase.queries {
			require.Equal(t, testCase.aliases, aliasesOf(query(name)),
				"urutan alias kueri %q bergeser", name)
		}
	}
}

// Setiap kueri daftar WAJIB mengembalikan jumlah seluruh baris lewat `COUNT(*) OVER ()`.
//
// Tanpa itu, paginasi kehilangan totalnya dan layar menggambar "halaman 1 dari 1" pada
// daftar yang sebenarnya punya ribuan baris.
func TestEveryListQueryReturnsTheTotalRowCount(t *testing.T) {
	for _, name := range append(listQueries, "list_salvage") {
		require.Contains(t, query(name), "COUNT(*) OVER ()",
			"kueri %q tidak menghitung total barisnya", name)
		require.Contains(t, query(name), "TOTAL_ROWS")
	}
}

// Paginasi dipotong DI BASIS DATA, bukan setelah baris sampai ke aplikasi.
func TestEveryListQueryPagesInTheDatabase(t *testing.T) {
	for _, name := range append(listQueries, "list_salvage") {
		text := query(name)
		require.Contains(t, text, "OFFSET", "kueri %q tidak memotong halaman", name)
		require.Contains(t, text, "FETCH NEXT", "kueri %q tidak membatasi halaman", name)
	}
}

// Penyaring lini bisnis WAJIB ada di setiap kueri berbasis klaim.
//
// Kueri yang kehilangannya akan menampilkan klaim Personal Accident dan Travel — lini yang
// tidak mengenal salvage sama sekali — tanpa satu pun galat.
func TestClaimQueriesAlwaysFilterTheBusinessLines(t *testing.T) {
	for _, name := range businessFilterQueries {
		text := query(name)
		for _, panel := range inboxsalvage.GroupPanels {
			require.Contains(t, text, "'"+panel+"'",
				"kueri %q kehilangan Group Panel %s", name, panel)
		}
		for _, code := range inboxsalvage.ExcludedBizCode {
			require.Contains(t, text, "'"+code+"'",
				"kueri %q kehilangan pengecualian %s", name, code)
		}
	}
}

// TIDAK ADA nilai yang dirangkai ke dalam teks SQL.
//
// Sistem lama menyisipkan isi kotak pencarian langsung ke dalam kueri lewat pola
// `{ASIS:…}` di LIMA tempat. Uji ini menjaga polanya tidak pernah terbawa.
func TestNoQueryCarriesTheLegacyStringSplicingPattern(t *testing.T) {
	for name, text := range queries {
		require.NotContains(t, text, "{ASIS",
			"kueri %q membawa pola perangkaian SQL sistem lama", name)
		require.NotContains(t, text, "||'%'||",
			"kueri %q merangkai pola pencarian di dalam SQL", name)
	}
}

// Setiap pencarian memakai `ESCAPE`, supaya `%` yang DIKETIK pengguna tidak berlaku sebagai
// wildcard — terutama pada ketiga daftar yang pencariannya cocok persis.
func TestEverySearchQueryDeclaresAnEscapeCharacter(t *testing.T) {
	searchQueries := []string{
		"list_claim_search", "list_claim_object_search",
		"list_claim_buyback_search", "list_salvage",
	}

	for _, name := range searchQueries {
		text := query(name)
		likeCount := strings.Count(text, "LIKE")
		escapeCount := strings.Count(text, "ESCAPE")
		require.Equal(t, likeCount, escapeCount,
			"kueri %q punya %d LIKE tetapi %d ESCAPE", name, likeCount, escapeCount)
	}
}

// Kedua cabang pencarian keluarga C digabung dengan ATAU, bukan DAN.
//
// Penyaring tab Checker di sistem lama berbunyi `and (a.noklaim = '…' or a.pic = '…')` —
// satu kotak mencari dua kolom. Menggantinya dengan DAN akan membuat pencarian nomor klaim
// hanya menemukan baris yang PIC-nya kebetulan sama dengan nomor klaim itu, yakni tidak
// pernah.
func TestTheTwoSearchBranchesAreJoinedWithOr(t *testing.T) {
	text := query("list_salvage")
	require.Regexp(t, regexp.MustCompile(`\(\s*UPPER\(a\.NOKLAIM\)[\s\S]*?OR[\s\S]*?a\.PIC`),
		text, "kedua cabang pencarian tidak digabung dengan OR")
}

// Kueri pembaruan TIDAK BOLEH menulis `IDSALVAGE`.
//
// Inilah butir 12 daftar perbaikan `P-5` (`D-49` #9): procedure lama menimpa kunci barisnya
// sendiri dengan nilai kosong pada cabang pembaruan, sehingga barisnya hilang dari seluruh
// daftar. Uji ini menjaga perbaikannya tetap berlaku.
func TestUpdateNeverWritesTheSalvageIDColumn(t *testing.T) {
	text := query("update_salvage")

	setClause := text
	if index := strings.Index(text, "WHERE"); index >= 0 {
		setClause = text[:index]
	}

	require.NotContains(t, setClause, "IDSALVAGE =",
		"pembaruan tidak boleh menimpa kunci barisnya sendiri")
	require.Contains(t, text, "WHERE IDSALVAGE = :10",
		"kunci pencarinya tetap ID salvage")
}

// Bind pada pernyataan simpan WAJIB sejumlah argumen yang disusun insertArgs. Selisih satu
// saja menghasilkan galat bind yang menyebut nomor, bukan menyebut isian mana yang hilang.
func TestInsertArgumentCountMatchesTheBindsInBothStatements(t *testing.T) {
	form := inboxsalvage.Form{Caller: inboxsalvage.Caller{Login: "SITIRAHAYU"}}
	args := insertArgs(form, "101")

	require.Len(t, args, 22, "insertArgs menyusun 22 bind")

	for _, name := range []string{"insert_salvage", "update_salvage"} {
		require.Equal(t, 22, highestBind(query(name)),
			"kueri %q memakai nomor bind tertinggi yang berbeda", name)
	}
}

// Nomor bind harus BERURUTAN tanpa lompatan. Lompatan berarti satu argumen tidak pernah
// dipakai, dan argumen yang tidak dipakai biasanya berarti isian yang tidak tersimpan.
func TestBindNumbersAreContiguousInEveryStatement(t *testing.T) {
	for name, text := range queries {
		highest := highestBind(text)
		if highest == 0 {
			continue
		}

		// `update_salvage` memang melewati satu nomor di klausa SET — `:10` dipakai di
		// WHERE — sehingga seluruh nomor tetap terpakai, hanya letaknya berbeda.
		for number := 1; number <= highest; number++ {
			require.Contains(t, text, ":"+strconv.Itoa(number),
				"kueri %q melewatkan bind :%d", name, number)
		}
	}
}

// Awalan kunci Pega DIIKAT, bukan ditulis di dalam SQL — supaya terbaca sebagai nilai yang
// kelak berubah (`D-22`, `D-71`), bukan sebagai bagian kuerinya.
func TestThePegaWorkKeyPrefixIsBoundNotWrittenIntoTheSQL(t *testing.T) {
	require.NotContains(t, query("list_salvage"), "ASM-FW-GCNMFW-WORK",
		"awalan kunci Pega tidak boleh tertanam di dalam teks SQL")

	// Dan SPASI di ujungnya adalah bagian nilainya. Tanpa spasi itu gabungan tidak pernah
	// cocok, dan daftarnya kosong tanpa satu pun galat.
	require.True(t, strings.HasSuffix(PegaWorkKeyPrefix, " "),
		"awalan kunci Pega wajib berakhir dengan spasi")
}

// Tidak satu pun kueri memakai konstruksi khas Oracle yang `D-20` larang.
func TestNoQueryUsesTheForbiddenOraclePatterns(t *testing.T) {
	forbidden := []string{"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "SELECT *"}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContains(t, upper, pattern,
				"kueri %q memakai pola terlarang %q", name, pattern)
		}
	}
}

// aliasesOf membaca urutan alias pada klausa SELECT TERLUAR sebuah kueri.
//
// # Kenapa kedalaman kurung dihitung
//
// Karena dua kueri di berkas ini memuat sub-kueri berkolom sendiri — nama objek pertama
// klaim, dan kedua agregat detail salvage. Berhenti pada `FROM` yang PERTAMA ditemukan akan
// berhenti di dalam sub-kueri itu, sehingga separuh alias terluar tidak pernah terbaca dan
// ujinya gagal karena alasan yang bukan yang sedang diujinya.
//
// Yang dicari adalah `FROM` pada kedalaman NOL, dan alias yang dikumpulkan hanyalah yang
// berada pada kedalaman itu pula — alias di dalam sub-kueri bukan kolom hasil.
func aliasesOf(text string) []string {
	pattern := regexp.MustCompile(`(?i)^AS\s+([A-Za-z_][A-Za-z0-9_]*)`)

	found := []string{}
	depth := 0

	for index := 0; index < len(text); index++ {
		switch text[index] {
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}

		if depth != 0 {
			continue
		}

		// Batas kata di kiri, supaya `… AS` di dalam nama kolom tidak ikut tertangkap.
		if index > 0 && !isWordBoundary(text[index-1]) {
			continue
		}

		rest := text[index:]
		if match := pattern.FindStringSubmatch(rest); match != nil {
			found = append(found, strings.ToUpper(match[1]))
			index += len(match[0]) - 1
			continue
		}

		// `FROM` pada kedalaman nol menutup klausa SELECT terluar.
		if len(rest) >= 5 && strings.EqualFold(rest[:4], "FROM") &&
			isWordBoundary(rest[4]) {
			return found
		}
	}

	return found
}

// isWordBoundary menyatakan sebuah bita bukan bagian dari kata.
func isWordBoundary(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
		return false
	default:
		return true
	}
}

// highestBind mengembalikan nomor bind tertinggi yang dipakai sebuah kueri.
func highestBind(text string) int {
	pattern := regexp.MustCompile(`:(\d+)`)

	highest := 0
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		if number, err := strconv.Atoi(match[1]); err == nil && number > highest {
			highest = number
		}
	}
	return highest
}
