package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji
// ini, salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"auto_claim_list",
		"auto_claim_list_by_committee",
		"auto_claim_get",
		"auto_claim_list_initial_locked",
		"auto_claim_insert",
		"auto_claim_update",
		"auto_claim_check_table",
		"auto_claim_business_source_search",
		"auto_claim_client_search",
		"auto_claim_bank_list",
		"auto_claim_bank_by_name",
		"auto_claim_committee",
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

// Disiplin SQL portabel (D-20) hanya bertahan bila ditegakkan perkakas, bukan diingat
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
		"FROM DUAL": "tidak ada padanannya di PostgreSQL",
		"VARCHAR2":  "VARCHAR sudah diterima Oracle maupun PostgreSQL",
	}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}.
//
// Penyaring komite patut diperhatikan khusus: di Pega ia DIRANGKAI dari operator ID
// (`"and KOMITE = '" + OperatorID.pyUserIdentifier + "'"`). Uji ini yang menjaga ia
// tetap terikat.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"auto_claim_list",
		"auto_claim_list_by_committee",
		"auto_claim_get",
		"auto_claim_list_initial_locked",
		"auto_claim_insert",
		"auto_claim_update",
		"auto_claim_business_source_search",
		"auto_claim_client_search",
		"auto_claim_bank_by_name",
	}
	for _, name := range parameterised {
		require.Containsf(t, getQuery(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}

	require.Containsf(t, getQuery("auto_claim_list_by_committee"), ":2",
		"penyaring komite harus terikat, bukan dirangkai ke teks SQL")
}

// Tidak ada DELETE terhadap tabel mana pun. Sistem lama tidak punya satu pun, dan D-66
// melarang penghapusan fisik data bernilai bisnis.
func TestNoDeleteStatement(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "DELETE",
			"kueri %q memuat DELETE; baris yang tidak dipakai ditolak komite, bukan dibuang (D-66)", name)
	}
}

// Keempat tabel acuan HANYA DIBACA (ADR-0004, penulis tunggal per tabel).
//
// Satu-satunya tabel yang boleh ditulis modul ini adalah POOLDATA.M_AUTO_CLAIM_PNC.
func TestOnlyTheMasterTableIsWritten(t *testing.T) {
	readOnlyTables := []string{"POOLDATA.AGENT", "POOLDATA.CLIENT", "GENERAL.LST_BANK_GROUP", "POOLDATA.EMAILKOMITE"}

	for name, text := range query {
		upperCase := strings.ToUpper(text)
		if !strings.HasPrefix(upperCase, "INSERT") && !strings.HasPrefix(upperCase, "UPDATE") {
			continue
		}
		for _, table := range readOnlyTables {
			require.NotContainsf(t, upperCase, table,
				"kueri %q menulis ke %s, padahal tabel itu hanya boleh dibaca (ADR-0004)", name, table)
		}
		require.Containsf(t, upperCase, "POOLDATA.M_AUTO_CLAIM_PNC",
			"kueri tulis %q menyentuh tabel yang bukan milik modul ini", name)
	}
}

// NAMA_PENERIMA TIDAK BOLEH ada di pernyataan UPDATE.
//
// Keputusan Work Owner 2026-09-19, dan kueri lama pun tidak menyebutnya. Uji ini yang
// membuat penambahannya kelak menjadi keputusan sadar, bukan kelalaian saat menyalin
// daftar kolom dari pernyataan INSERT di sebelahnya.
func TestUpdateNeverTouchesReceiverName(t *testing.T) {
	require.NotContains(t, strings.ToUpper(getQuery("auto_claim_update")), "NAMA_PENERIMA")

	// Sebagai pembanding: INSERT memang menulisnya, sekali, saat baris dibuat.
	require.Contains(t, strings.ToUpper(getQuery("auto_claim_insert")), "NAMA_PENERIMA")
}

// Ketiga kueri pembaca master harus menyebut kolom pada URUTAN YANG SAMA.
//
// scanRow membaca ketiganya dengan satu fungsi, berdasarkan POSISI kolom. Satu kolom
// yang bergeser di salah satu kueri akan menaruh nomor rekening ke kolom alamat tanpa
// satu pun galat — dan itu justru pada modul yang menentukan ke mana uang dikirim.
func TestReaderQueriesShareColumnOrder(t *testing.T) {
	expected := []string{
		"INISIALID", "NAMA_PENERIMA", "BANK_PENERIMA", "NO_REKENING", "PCT_MAX",
		"PIC_LAPOR", "EMAIL_LAPOR", "CLAIM_ALLOWED", "ALAMAT_PENERIMA",
		"KOMITE", "APPROVAL", "CLIENTID", "CLIENTNAME",
	}

	for _, name := range []string{"auto_claim_list", "auto_claim_list_by_committee", "auto_claim_get"} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, selectedColumns(getQuery(name)))
		})
	}
}

// selectedColumns mengambil nama kolom pada klausa SELECT sebuah kueri sederhana.
//
// Ia hanya menangani bentuk yang dipakai berkas ini — satu kolom per baris, tanpa
// ekspresi dan tanpa alias — dan itu memang cukup: begitu sebuah kueri pembaca menjadi
// lebih rumit dari itu, uji ini gagal dan memaksa pembacanya melihat kembali.
func selectedColumns(text string) []string {
	upperCase := strings.ToUpper(text)
	start := strings.Index(upperCase, "SELECT")
	end := strings.Index(upperCase, "FROM")
	if start < 0 || end < 0 || end < start {
		return nil
	}

	var column []string
	for _, part := range strings.Split(upperCase[start+len("SELECT"):end], ",") {
		if clean := strings.TrimSpace(part); clean != "" {
			column = append(column, clean)
		}
	}
	return column
}

// Kueri penyetuju komite TETAP memuat 'BONDING' yang tertanam.
//
// Ia direplikasi apa adanya atas keputusan Work Owner 2026-09-19, meski ia persis
// bentuk hardcode yang D-15 perintahkan menjadi master atau konfigurasi. Uji ini
// menjaga keputusan itu TERLIHAT: siapa pun yang kelak mengangkatnya menjadi
// konfigurasi akan menghapus uji ini dengan sadar, bukan mengubah perilaku diam-diam.
func TestCommitteeQueryKeepsHardcodedBusinessType(t *testing.T) {
	require.Contains(t, strings.ToUpper(getQuery("auto_claim_committee")), "'BONDING'")
}

// Kedua kueri lookup menyebut ESCAPE secara eksplisit.
//
// Oracle tidak punya karakter pelolos bawaan pada LIKE, sehingga tanda persen yang
// diketik pengguna tetap berlaku sebagai wildcard meski sudah diloloskan di Go.
func TestLookupQueriesDeclareEscape(t *testing.T) {
	for _, name := range []string{"auto_claim_business_source_search", "auto_claim_client_search"} {
		require.Containsf(t, strings.ToUpper(getQuery(name)), "ESCAPE",
			"kueri %q memakai LIKE tanpa menyebut ESCAPE", name)
	}
}

// Kata kunci dinormalkan sama persis dengan Activity/GetClientName step 3, dan karakter
// wildcard-nya diloloskan.
func TestLookupArguments(t *testing.T) {
	for _, c := range struct2Cases() {
		t.Run(c.keyword, func(t *testing.T) {
			pattern, exact := lookupArguments(c.keyword)
			require.Equal(t, c.pattern, pattern)
			require.Equal(t, c.exact, exact)
		})
	}
}

type lookupCase struct{ keyword, pattern, exact string }

func struct2Cases() []lookupCase {
	return []lookupCase{
		{"pt. abc", "%PT ABC%", "PT ABC"},
		{"  agn001  ", "%AGN001%", "AGN001"},
		// Wildcard diloloskan: pengguna yang mengetik "%" mencari tanda persen, bukan
		// meminta seluruh tabel.
		{"50%", `%50\%%`, "50%"},
		{"a_b", `%A\_B%`, "A_B"},
		{`c\d`, `%C\\D%`, `C\D`},
	}
}
