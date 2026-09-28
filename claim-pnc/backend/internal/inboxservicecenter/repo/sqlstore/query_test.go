package sqlstore

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxservicecenter"
)

// namedQueries adalah seluruh kueri yang dimuat berkas .sql modul ini.
var namedQueries = []string{"list_claims", "count_claims", "check_table"}

func TestSeluruhKueriTerbaca(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong", name)
	}
}

// TestSetiapPenandaBindMunculTepatSekaliDanBerurutan menjaga cacat yang pernah ditemukan di
// modul lain (§63 catatan pengembangan): penanda yang berulang membuat jumlah argumen tidak
// lagi sama dengan jumlah kemunculannya, dan galatnya baru terbaca saat kueri dijalankan
// terhadap basis data sungguhan.
func TestSetiapPenandaBindMunculTepatSekaliDanBerurutan(t *testing.T) {
	expected := map[string]int{"list_claims": 17, "count_claims": 15}

	for name, count := range expected {
		markers := regexp.MustCompile(`:\d+`).FindAllString(query(name), -1)

		want := make([]string, 0, count)
		for i := 1; i <= count; i++ {
			want = append(want, fmt.Sprintf(":%d", i))
		}
		require.Equalf(t, want, markers,
			"kueri %s: penanda harus :1..:%d, masing-masing tepat sekali dan berurutan", name, count)
	}
}

// TestJumlahArgumenSamaDenganJumlahPenanda menutup jarak antara kedua berkas: berkas .sql
// menyebut berapa penanda, dan filterArgs menyebut berapa argumen. Keduanya harus sama.
func TestJumlahArgumenSamaDenganJumlahPenanda(t *testing.T) {
	q := inboxservicecenter.Query{
		Tab:      mustTab(t, inboxservicecenter.TabApproved),
		Approval: inboxservicecenter.ApprovalFilter{Codes: []string{"1"}},
		Keyword:  "abc",
		Caller:   inboxservicecenter.Caller{Login: "PIC"},
	}

	filters := filterArgs(q)
	require.Len(t, filters, 15, "count_claims memakai :1..:15")

	markers := regexp.MustCompile(`:\d+`).FindAllString(query("count_claims"), -1)
	require.Len(t, markers, len(filters))
}

// TestUrutanKolomHasilSamaDenganSELECT menjaga ketiga tempat yang harus sejalan: daftar
// resultColumns, urutan kolom pada SELECT, dan urutan pemindai scanClaim.
//
// Bila salah satu bergeser tanpa yang lain, nilainya akan tertukar antar kolom — dan
// tertukarnya DIAM, karena seluruhnya bertipe teks.
func TestUrutanKolomHasilSamaDenganSELECT(t *testing.T) {
	text := query("list_claims")

	start := strings.Index(text, "SELECT")
	end := strings.Index(text, "FROM")
	require.Greater(t, end, start)

	body := text[start+len("SELECT") : end]

	position := -1
	for _, column := range resultColumns {
		found := strings.Index(body, "k."+column)
		require.GreaterOrEqualf(t, found, 0, "kolom %s tidak ada di SELECT list_claims", column)
		require.Greaterf(t, found, position,
			"kolom %s tidak berurutan seperti resultColumns", column)
		position = found
	}
}

// TestPenyaringDaftarDanHitungSamaPersis menutup kelas cacat yang khas layar berpaginasi:
// penyaring yang berbeda antara kueri daftar dan kueri hitung menghasilkan "halaman 1 dari 7"
// yang halaman ketujuhnya kosong.
func TestPenyaringDaftarDanHitungSamaPersis(t *testing.T) {
	whereOf := func(name string) string {
		text := query(name)
		start := strings.Index(text, "WHERE")
		require.GreaterOrEqual(t, start, 0)

		body := text[start:]
		if cut := strings.Index(body, "ORDER BY"); cut >= 0 {
			body = body[:cut]
		}
		return strings.Join(strings.Fields(body), " ")
	}

	require.Equal(t, whereOf("count_claims"), whereOf("list_claims"))
}

func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, text, forbidden,
				"kueri %s memuat pernyataan yang menulis: %s", name, forbidden)
		}
	}
}

// TestSQLPortabel menjaga `D-20`: satu set SQL yang berjalan di Oracle dan PostgreSQL 17+.
func TestSQLPortabel(t *testing.T) {
	terlarang := []string{"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "FROM DUAL", "SELECT *"}

	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, pola := range terlarang {
			require.NotContainsf(t, text, pola,
				"kueri %s memuat pola tidak portabel: %s", name, pola)
		}
	}

	require.Contains(t, query("list_claims"), "OFFSET :16 ROWS FETCH NEXT :17 ROWS ONLY")
}

func TestKueriPeriksaTidakMembacaSatuBarisPun(t *testing.T) {
	require.Contains(t, query("check_table"), "1 = 0")
	require.Contains(t, query("check_table"), "COUNT(*)")
}

// TestPolaPencarianDiEscape — "100%" tidak boleh menjadi pola yang cocok dengan semuanya.
func TestPolaPencarianDiEscape(t *testing.T) {
	require.Equal(t, "", searchPattern("   "))
	require.Equal(t, "%SC-0001%", searchPattern(" sc-0001 "))
	require.Equal(t, `%100\%%`, searchPattern("100%"))
	require.Equal(t, `%A\_B%`, searchPattern("a_b"))
	require.Equal(t, `%C\\D%`, searchPattern(`c\d`))

	// Escaping hanya bekerja bila SQL menyatakan karakter escape-nya. Keduanya harus cocok:
	// Go meng-escape dengan backslash, SQL wajib menyebut ESCAPE '\'.
	require.Contains(t, query("list_claims"), `ESCAPE '\'`)
}

// TestKataKunciKosongMenjadiNULL menjaga penjaga `:n IS NULL` tetap bekerja.
func TestKataKunciKosongMenjadiNULL(t *testing.T) {
	q := inboxservicecenter.Query{
		Tab:      mustTab(t, inboxservicecenter.TabRegistration),
		Approval: inboxservicecenter.ApprovalFilter{MatchNull: true},
		Caller:   inboxservicecenter.Caller{Login: "pic"},
	}

	args := filterArgs(q)
	require.Nil(t, args[5], ":6 harus NULL saat tidak mencari")
	require.Nil(t, args[10], ":11 harus NULL saat tidak mencari")
}

// TestPenandaStatusPersetujuanPerTab membuktikan ketiga bentuk penyaring yang berbeda
// tersalurkan ke argumen yang benar.
func TestPenandaStatusPersetujuanPerTab(t *testing.T) {
	login := inboxservicecenter.Caller{Login: "pic"}

	t.Run("Registrasi SC menerima NULL, bukan kode", func(t *testing.T) {
		args := filterArgs(inboxservicecenter.Query{
			Tab:      mustTab(t, inboxservicecenter.TabRegistration),
			Approval: inboxservicecenter.ApprovalFilter{MatchNull: true},
			Caller:   login,
		})
		require.Equal(t, "Y", args[0], ":1 menyalakan cabang IS NULL")
		require.Equal(t, "N", args[1], ":2 mematikan cabang IN")
		require.Nil(t, args[2])
		require.Nil(t, args[3])
	})

	t.Run("tab berkode tunggal mengirim kode yang sama dua kali", func(t *testing.T) {
		args := filterArgs(inboxservicecenter.Query{
			Tab:      mustTab(t, inboxservicecenter.TabApproved),
			Approval: inboxservicecenter.ApprovalFilter{Codes: []string{"1"}},
			Caller:   login,
		})
		require.Equal(t, "N", args[0])
		require.Equal(t, "Y", args[1])
		require.Equal(t, "1", args[2])
		require.Equal(t, "1", args[3])
	})

	t.Run("Rejected membawa TLO dan REJECT sekaligus", func(t *testing.T) {
		args := filterArgs(inboxservicecenter.Query{
			Tab:      mustTab(t, inboxservicecenter.TabRejected),
			Approval: inboxservicecenter.ApprovalFilter{Codes: []string{"2", "3"}},
			Caller:   login,
		})
		require.Equal(t, "2", args[2])
		require.Equal(t, "3", args[3])
	})
}

// TestLoginDiseragamkanMenjadiHurufBesar — sisi SQL memakai UPPER(TRIM(k.PIC)).
func TestLoginDiseragamkanMenjadiHurufBesar(t *testing.T) {
	args := filterArgs(inboxservicecenter.Query{
		Tab:      mustTab(t, inboxservicecenter.TabApproved),
		Approval: inboxservicecenter.ApprovalFilter{Codes: []string{"1"}},
		Caller:   inboxservicecenter.Caller{Login: "picservicecenter"},
	})
	require.Equal(t, "PICSERVICECENTER", args[4])
	require.Contains(t, query("list_claims"), "UPPER(TRIM(k.PIC)) = :5")
}

func mustTab(t *testing.T, code string) inboxservicecenter.Tab {
	t.Helper()
	tab, known := inboxservicecenter.FindTab(code)
	require.Truef(t, known, "tab %s tidak terdaftar", code)
	return tab
}
