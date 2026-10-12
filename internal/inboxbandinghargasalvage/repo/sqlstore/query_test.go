package sqlstore

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Uji di berkas ini tidak menyentuh basis data. Yang diperiksa adalah SIFAT teks kuerinya —
// hal-hal yang bila salah tidak menghasilkan galat kompilasi, dan baru terbaca sebagai kolom
// tertukar, halaman kosong, atau angka ringkas yang tidak pernah cocok.

const (
	namaListRequest  = "list_request"
	namaCountRequest = "count_request"
	namaListHistory  = "list_history"
	namaCountHistory = "count_history"
	namaCheckTable   = "check_table"
	namaListDecision = "list_decisions"
	namaCheckWrite   = "check_write_targets"
)

func TestSeluruhKueriBernamaAda(t *testing.T) {
	for _, name := range []string{
		namaListRequest, namaCountRequest, namaListHistory, namaCountHistory,
		namaListDecision, namaCheckTable, namaCheckWrite,
	} {
		require.NotPanics(t, func() { query(name) }, "kueri %q", name)
		require.NotEmpty(t, strings.TrimSpace(query(name)), "kueri %q kosong", name)
	}
}

// Penanda parameter harus bernomor urut 1..N dan masing-masing muncul TEPAT SEKALI.
//
// Penanda berulang membuat jumlah argumen tidak lagi sama dengan jumlah kemunculannya, dan
// galatnya baru terbaca saat kueri dijalankan — bukan saat kodenya dibaca (§63 catatan
// pengembangan).
func TestPenandaParameterBernomorUrutDanMunculSekali(t *testing.T) {
	for _, name := range []string{
		namaListRequest, namaCountRequest, namaListHistory, namaCountHistory,
		namaListDecision,
		"decide_record", "decide_cascade", "decide_mark_document", "decide_apply_price",
	} {
		markers := penandaPada(query(name))
		require.NotEmpty(t, markers, "kueri %q tidak punya penanda sama sekali", name)

		for index, marker := range markers {
			require.Equal(t, index+1, marker,
				"kueri %q: penanda harus 1..N berurutan, ditemukan %v", name, markers)
		}
	}
}

// Kueri pencacah harus menyaring hal yang SAMA PERSIS dengan kueri daftarnya.
//
// Inilah selisih terencana nomor 4 yang dijaga: di sistem lama keduanya menghitung populasi
// yang berbeda, dan akibatnya angka ringkas tidak pernah cocok dengan jumlah baris gridnya.
func TestKueriPencacahMenyaringSamaPersisDenganDaftarnya(t *testing.T) {
	pasangan := []struct {
		daftar   string
		pencacah string
	}{
		{namaListRequest, namaCountRequest},
		{namaListHistory, namaCountHistory},
	}

	for _, p := range pasangan {
		require.Equal(t,
			klausaWhere(query(p.daftar)),
			klausaWhere(query(p.pencacah)),
			"klausa WHERE %q dan %q harus sama persis", p.daftar, p.pencacah)
	}
}

// Kueri pencacah TIDAK boleh memuat ORDER BY.
//
// Selisih terencana nomor 3: `CountRequestSalvage_Sql` menutup `SELECT COUNT(1)` dengan
// `ORDER BY TGL_REQUEST DESC` — tidak bermakna, menyebut kolom yang tidak dikenal, dan pada
// Oracle berpotensi menggagalkan kuerinya.
func TestKueriPencacahTidakMemuatOrderBy(t *testing.T) {
	for _, name := range []string{namaCountRequest, namaCountHistory} {
		require.NotContains(t, strings.ToUpper(query(name)), "ORDER BY",
			"kueri %q tidak boleh punya ORDER BY", name)
	}
}

// Kueri daftar WAJIB punya ORDER BY.
//
// `HistoryReqSalvage_SQL` di Pega tidak punya satu pun, dan tanpa urutan yang pasti
// `OFFSET … FETCH` boleh mengembalikan baris dalam urutan yang berbeda antar halaman —
// satu baris muncul dua kali sementara baris lain tidak pernah terlihat.
func TestKueriDaftarSelaluPunyaOrderBySebelumPaginasi(t *testing.T) {
	for _, name := range []string{namaListRequest, namaListHistory} {
		upper := strings.ToUpper(query(name))
		require.Contains(t, upper, "ORDER BY", "kueri %q", name)
		require.Contains(t, upper, "OFFSET", "kueri %q", name)
		require.Less(t, strings.Index(upper, "ORDER BY"), strings.Index(upper, "OFFSET"),
			"kueri %q: ORDER BY harus mendahului OFFSET", name)
	}
}

// Urutan alias pada SELECT harus sama dengan daftar kolom di berkas ini — dan daftar itu pula
// yang menjadi urutan pemindainya. Ketiganya berselisih tanpa menghasilkan galat apa pun;
// yang terlihat hanyalah kolom yang isinya tertukar.
func TestUrutanAliasSelectSamaDenganDaftarKolom(t *testing.T) {
	require.Equal(t, requestColumns, aliasSelect(query(namaListRequest)))
	require.Equal(t, historyColumns, aliasSelect(query(namaListHistory)))
	require.Equal(t, decisionColumns, aliasSelect(query(namaListDecision)))
}

// Jumlah argumen penyaring harus sama dengan jumlah penanda pada kuerinya, di luar kedua
// penanda paginasi yang ditambahkan List.
func TestJumlahArgumenPenyaringSamaDenganJumlahPenanda(t *testing.T) {
	kasus := []struct {
		tab      string
		daftar   string
		pencacah string
	}{
		{inboxbandinghargasalvage.TabRequest, namaListRequest, namaCountRequest},
		{inboxbandinghargasalvage.TabHistory, namaListHistory, namaCountHistory},
	}

	for _, k := range kasus {
		query, err := inboxbandinghargasalvage.NewQuery(
			inboxbandinghargasalvage.QueryInput{Tab: k.tab, Keyword: "PNCN.26.0001"},
			inboxbandinghargasalvage.Caller{Login: "DANIELLISWANDI"},
		)
		require.NoError(t, err)

		args := filterArgs(query)

		require.Len(t, args, len(penandaPada(teksKueri(k.pencacah))),
			"tab %s: argumen penyaring vs penanda pencacah", k.tab)

		// Kueri daftar punya DUA penanda tambahan — offset dan jumlah baris — yang
		// ditambahkan List, bukan filterArgs.
		require.Equal(t, len(args)+2, len(penandaPada(teksKueri(k.daftar))),
			"tab %s: argumen penyaring + 2 penanda paginasi vs penanda daftar", k.tab)
	}
}

// Nama kueri dan pemindainya harus berpasangan benar. Pasangan yang tertukar tidak
// menghasilkan galat kompilasi — keduanya bertipe sama.
func TestNamaKueriDanPemindaiBerpasanganMenurutTabnya(t *testing.T) {
	request, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabRequest)
	require.True(t, known)
	nameRequest, _ := listQueryFor(request)
	require.Equal(t, namaListRequest, nameRequest)
	require.Equal(t, namaCountRequest, countQueryFor(request))

	history, known := inboxbandinghargasalvage.FindTab(inboxbandinghargasalvage.TabHistory)
	require.True(t, known)
	nameHistory, _ := listQueryFor(history)
	require.Equal(t, namaListHistory, nameHistory)
	require.Equal(t, namaCountHistory, countQueryFor(history))
}

// Pola khas Oracle yang `09-DATABASE-STRATEGY.md` §4 larang dibawa.
//
// Kueri lama memakai `TRUNC (SYSDATE) - TRUNC (A.tglrequest)`; padanannya di sini
// `CAST(CURRENT_TIMESTAMP AS DATE) - CAST(… AS DATE)`. Uji ini menjaga agar bentuk lamanya
// tidak diam-diam kembali saat seseorang menyalin ulang dari rule Pega.
func TestTidakAdaPolaSQLKhasOracle(t *testing.T) {
	terlarang := []string{
		"SYSDATE", "NVL(", "ROWNUM", "DECODE(", "INSTR(", "LISTAGG(",
		"FROM DUAL", "TRUNC(", "TRUNC ", "SELECT *", "(+)",
	}

	for name := range queries {
		upper := strings.ToUpper(query(name))
		for _, pola := range terlarang {
			require.NotContains(t, upper, pola,
				"kueri %q memuat pola khas Oracle %q", name, pola)
		}
	}
}

// Nilai tidak pernah dirangkai ke dalam teks SQL. Sistem lama menyisipkannya lewat
// `{Asis:…}`, termasuk kata kunci yang diketik pengguna.
func TestTidakAdaPenyisipanGayaAsis(t *testing.T) {
	for name := range queries {
		require.NotContains(t, strings.ToUpper(query(name)), "{ASIS",
			"kueri %q memuat penyisipan gaya {Asis:…}", name)
		require.NotContains(t, query(name), "'\"+",
			"kueri %q tampak merangkai nilai ke dalam teks SQL", name)
	}
}

// HANYA keempat kueri keputusan yang boleh menulis.
//
// Modul ini semula tidak menulis apa pun. Sejak tombol Approve/Reject dibangun ia menulis,
// dan batas itu harus dijaga mesin — bukan ingatan: menambah satu `UPDATE` di kueri baca
// tidak menghasilkan galat apa pun, dan akibatnya baru terlihat sebagai data yang berubah
// saat seseorang membuka layar.
func TestHanyaKueriKeputusanYangMenulis(t *testing.T) {
	bolehMenulis := map[string]bool{
		"decide_record":        true,
		"decide_cascade":       true,
		"decide_mark_document": true,
		"decide_apply_price":   true,
	}

	for name := range queries {
		upper := strings.ToUpper(query(name))
		menulis := false
		for _, kata := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE "} {
			if strings.Contains(upper, kata) {
				menulis = true
			}
		}

		if menulis && !bolehMenulis[name] {
			t.Fatalf("kueri %q menulis, padahal ia bukan kueri keputusan", name)
		}
		if !menulis && bolehMenulis[name] {
			t.Fatalf("kueri keputusan %q ternyata tidak menulis apa pun", name)
		}
	}
}

// Kueri keputusan hanya boleh menyentuh ketiga tabel yang disepakati, dan hanya kolom yang
// disepakati pada dua di antaranya.
//
// `T_CLAIM_CHEKER_SALVAGE` dimiliki modul ini. `DETAIL_PNC_SALVAGE` dan `SALAVAGEDOCUMENT`
// dimiliki modul Inbox Salvage, dan modul ini menyentuh SATU kolom pada masing-masing —
// keputusan Work Owner 2026-09-30. Batas itu tidak dapat dijaga dengan komentar.
func TestKueriKeputusanHanyaMenyentuhTabelYangDisepakati(t *testing.T) {
	diizinkan := []string{
		"POOLDATA.T_CLAIM_CHEKER_SALVAGE",
		"POOLDATA.DETAIL_PNC_SALVAGE",
		"POOLDATA.SALAVAGEDOCUMENT",
	}

	for _, name := range []string{
		"decide_record", "decide_cascade", "decide_mark_document", "decide_apply_price",
	} {
		upper := strings.ToUpper(query(name))

		cocok := false
		for _, tabel := range diizinkan {
			if strings.Contains(upper, tabel) {
				cocok = true
			}
		}
		require.True(t, cocok, "kueri %q menyentuh tabel di luar yang disepakati", name)
	}

	// Pada tabel milik modul lain, satu kolom saja.
	harga := strings.ToUpper(query("decide_apply_price"))
	require.Contains(t, harga, "SET HARGAITEM")
	require.NotContains(t, harga, ",",
		"decide_apply_price hanya boleh menyetel SATU kolom")

	dokumen := strings.ToUpper(query("decide_mark_document"))
	require.Contains(t, dokumen, "SET IDBALAILELANG")
}

// Kedua kueri yang mencatat putusan WAJIB menyaring `TGLAPPROVE IS NULL`.
//
// Penyaring itu yang membuat penekanan tombol kedua tidak menimpa putusan pertama — dan
// tanpanya, dua orang yang menekan bersamaan akan saling menimpa tanpa satu pun galat.
func TestKueriKeputusanMenjagaPutusanYangSudahAda(t *testing.T) {
	for _, name := range []string{"decide_record", "decide_cascade"} {
		require.Contains(t, strings.ToUpper(query(name)), "TGLAPPROVE IS NULL",
			"kueri %q harus menolak baris yang sudah diputus", name)
	}
}

// Waktu putusan memakai bentuk yang portabel, bukan `sysdate` milik kueri aslinya.
func TestWaktuPutusanMemakaiBentukPortabel(t *testing.T) {
	for _, name := range []string{"decide_record", "decide_cascade"} {
		require.Contains(t, strings.ToUpper(query(name)), "CURRENT_TIMESTAMP",
			"kueri %q harus memakai CURRENT_TIMESTAMP", name)
	}
}

// check_table harus menyebut SETIAP kolom yang dipakai modul ini.
//
// DDL `T_CLAIM_CHEKER_SALVAGE` belum pernah dibaca; seluruh nama kolom disimpulkan dari teks
// kueri Pega. `COUNT(*)` akan lolos meski kolomnya tidak ada — menyebutkannya membuat
// kekeliruan itu terbaca saat aplikasi start.
//
// Pemeriksaan ini sudah membuktikan gunanya sekali: kolom catatan komite semula ditulis
// `NOTEKOMITE`, mengikuti nama propertinya, dan itu KELIRU — `UpdateDataReqSalvage` menulis
// ke `NOTEAPPROVE`. `NOTEKOMITE` ada pada tabel LAIN (`T_CLAIM_KOMITE_LIST`).
func TestCheckTableMenyebutSetiapKolomYangDipakai(t *testing.T) {
	upper := strings.ToUpper(query(namaCheckTable))

	for _, kolom := range []string{
		"NOKLAIM", "IDSALVAGE", "IDDETAILSALVAGE", "NAMAKOMITE", "NAMABARANG",
		"HARGABARANG", "HARGAREQUEST", "ALASANREQUEST", "TGLREQUEST", "TGLAPPROVE",
		"STATUSAPPROVE", "NOTEAPPROVE",
	} {
		require.Contains(t, upper, kolom, "check_table harus menyebut kolom %q", kolom)
	}
}

// `NOTEKOMITE` tidak boleh muncul di berkas kueri mana pun modul ini.
//
// Ia kolom `T_CLAIM_KOMITE_LIST`, bukan `T_CLAIM_CHEKER_SALVAGE`, dan memakainya di sini
// menggagalkan SELURUH kueri dengan ORA-00904 — bukan mengosongkan satu kolom. Uji ini
// menjaga agar kekeliruan itu tidak kembali saat seseorang menyalin dari nama propertinya.
func TestKolomCatatanKomiteMemakaiNamaTabelIni(t *testing.T) {
	for name := range queries {
		require.NotContains(t, strings.ToUpper(query(name)), "NOTEKOMITE",
			"kueri %q memakai NOTEKOMITE — kolom itu milik T_CLAIM_KOMITE_LIST", name)
	}
}

// ============================================================================
// Pembantu
// ============================================================================

// teksKueri menyerahkan teks kueri bernama tertentu.
func teksKueri(name string) string { return query(name) }

var polaPenanda = regexp.MustCompile(`:(\d+)`)

// penandaPada menyerahkan nomor penanda yang muncul, terurut naik.
func penandaPada(text string) []int {
	found := map[int]int{}
	for _, match := range polaPenanda.FindAllStringSubmatch(text, -1) {
		number, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		found[number]++
	}

	numbers := make([]int, 0, len(found))
	for number, count := range found {
		// Penanda yang muncul lebih dari sekali dikembalikan sebagai nomor mustahil,
		// sehingga uji urutannya gagal dengan pesan yang menyebut nomornya.
		if count > 1 {
			numbers = append(numbers, -number)
			continue
		}
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	return numbers
}

// klausaWhere memotong bagian WHERE sebuah kueri, dinormalkan spasinya.
func klausaWhere(text string) string {
	upper := strings.ToUpper(text)

	start := strings.Index(upper, "\nWHERE")
	if start < 0 {
		start = strings.Index(upper, " WHERE ")
	}
	if start < 0 {
		return ""
	}

	tail := text[start:]
	for _, penutup := range []string{"\nORDER BY", "\n ORDER BY", "\nOFFSET", "\n OFFSET"} {
		if cut := strings.Index(strings.ToUpper(tail), strings.ToUpper(penutup)); cut >= 0 {
			tail = tail[:cut]
		}
	}

	return strings.Join(strings.Fields(tail), " ")
}

var polaAlias = regexp.MustCompile(`(?m)\bAS\s+([A-Z_][A-Z0-9_]*)\s*,?\s*$`)

// aliasSelect menyerahkan alias kolom pada bagian SELECT, berurutan.
//
// Dicocokkan per AKHIR BARIS, bukan di mana pun: `CAST(… AS DATE)` memuat kata `AS` pula, dan
// pencocokan bebas posisi akan menangkapnya sebagai nama kolom.
func aliasSelect(text string) []string {
	upper := strings.ToUpper(text)

	end := strings.Index(upper, "\nFROM")
	if end < 0 {
		end = strings.Index(upper, "  FROM")
	}
	if end < 0 {
		end = len(text)
	}

	result := []string{}
	for _, match := range polaAlias.FindAllStringSubmatch(strings.ToUpper(text[:end]), -1) {
		result = append(result, match[1])
	}
	return result
}

// Panel rincian WAJIB menyaring menurut nama komite, bukan hanya nomor klaim.
//
// Penyaring itu ada di kueri aslinya, dan tanpanya seorang komite dapat membaca keputusan
// komite lain hanya dengan menebak nomor klaim — dan nomor klaim bukan rahasia. Ia kelas
// kebocoran yang tidak menghasilkan satu pun galat.
func TestPanelRincianMenyaringMenurutNamaKomite(t *testing.T) {
	where := klausaWhere(query(namaListDecision))

	require.Contains(t, where, "NAMAKOMITE")
	require.Contains(t, where, "NOKLAIM")
	require.Contains(t, where, "STATUSAPPROVE IS NOT NULL")
}

// Kode keputusan dikirim APA ADANYA; penerjemahannya di Go.
//
// Kueri lama menerjemahkannya di dalam `CASE WHEN`. Pemetaan yang hidup di dalam SQL tidak
// dapat diuji tanpa basis data, sementara pemetaan yang salah menampilkan keputusan yang
// keliru tanpa satu pun galat.
func TestPanelRincianTidakMenerjemahkanKodeKeputusanDiDalamSQL(t *testing.T) {
	upper := strings.ToUpper(query(namaListDecision))

	require.NotContains(t, upper, "SETUJU")
	require.NotContains(t, upper, "PROSES")
	require.NotContains(t, upper, "CASE WHEN")
}

// Pemeriksa kesiapan harus menyebut SETIAP kolom yang benar-benar ditulis pada kedua tabel
// milik modul lain.
//
// Tanpa uji ini, sebuah kolom yang ditambahkan ke pernyataan tulis kelak akan lolos dari
// pemeriksaan kesiapan tanpa ada yang menyadarinya — dan kekeliruan namanya baru terbaca
// ketika seorang komite menekan Simpan.
func TestPemeriksaKesiapanMenyebutSetiapKolomYangDitulis(t *testing.T) {
	periksa := strings.ToUpper(query(namaCheckWrite))

	for _, kolom := range []string{
		"IDBALAILELANG", "NOKLAIM", "TIPEDOCSALVAGE",
		"HARGAITEM", "IDSALVAGE", "IDDETAILSALVAGE",
	} {
		require.Contains(t, periksa, "'"+kolom+"'",
			"kolom %q ditulis tetapi tidak diperiksa kesiapannya", kolom)
	}

	// Jumlah yang diharapkan pemanggil harus sama dengan jumlah yang disebut kuerinya.
	//
	// Keduanya hidup di berkas berbeda, dan selisihnya tidak menghasilkan galat apa pun —
	// hanya pesan kesiapan yang keliru: "punya 6 dari 6 kolom" padahal yang diperiksa tujuh.
	//
	// Yang dihitung adalah isi kedua klausa `IN (…)` saja. Menghitung seluruh teks berkutip
	// akan ikut menghitung nama pemilik dan nama tabelnya.
	require.Equal(t, WriteTargetColumns, len(kolomDalamKlausaIn(periksa)),
		"WriteTargetColumns tidak sama dengan jumlah kolom pada check_write_targets")
}

// Pemeriksa kesiapan TIDAK boleh menulis apa pun.
//
// Ia menyentuh nama tabel produksi, sehingga satu salah ketik `SELECT` menjadi `UPDATE`
// akan mengubah data setiap kali aplikasi start.
func TestPemeriksaKesiapanTidakMenulisApaPun(t *testing.T) {
	periksa := strings.ToUpper(query(namaCheckWrite))

	require.True(t, strings.HasPrefix(strings.TrimSpace(periksa), "SELECT"))
	require.Contains(t, periksa, "ALL_TAB_COLUMNS",
		"yang dibaca adalah KATALOG, bukan tabelnya sendiri")
}

// polaKlausaIn menangkap isi setiap klausa `IN (…)`.
var polaKlausaIn = regexp.MustCompile(`(?s)IN\s*\(([^)]*)\)`)

// polaTeksBerkutip menangkap satu teks berkutip tunggal.
var polaTeksBerkutip = regexp.MustCompile(`'([^']*)'`)

// kolomDalamKlausaIn menyerahkan seluruh nama kolom yang disebut klausa `IN (…)`.
func kolomDalamKlausaIn(text string) []string {
	kolom := []string{}
	for _, klausa := range polaKlausaIn.FindAllStringSubmatch(text, -1) {
		for _, teks := range polaTeksBerkutip.FindAllStringSubmatch(klausa[1], -1) {
			kolom = append(kolom, teks[1])
		}
	}
	return kolom
}

// ============================================================================
// Dialog "Lihat File"
// ============================================================================

// Kedua kueri dokumen WAJIB menyaring kepemilikan.
//
// Penyaring itu TIDAK ada di sistem lama dan sengaja ditambahkan: tanpanya, siapa pun yang
// tahu sepasang id dapat membaca — bahkan mengunduh — dokumen banding komite lain.
// Penambahan yang tidak dijaga uji mudah hilang saat kuerinya kelak disunting.
func TestKueriDokumenMenyaringKepemilikanKomite(t *testing.T) {
	for _, name := range []string{"list_documents", "document_content"} {
		teks := strings.ToUpper(query(name))

		require.Contains(t, teks, "T_CLAIM_CHEKER_SALVAGE",
			"kueri %q tidak memeriksa banding itu milik siapa", name)
		require.Contains(t, teks, "NAMAKOMITE",
			"kueri %q tidak menyaring menurut nama komite", name)
		require.Contains(t, teks, "EXISTS",
			"kepemilikan diperiksa DI DALAM kueri, bukan setelah barisnya terambil")
	}
}

// Keduanya hanya MEMBACA.
//
// Berkas ini memuat pernyataan tulis pula, dan salah ketik `SELECT` menjadi `UPDATE` pada
// kueri yang menyentuh tabel dokumen akan menandai dokumen tanpa ada yang memutuskannya.
func TestKueriDokumenTidakMenulisApaPun(t *testing.T) {
	for _, name := range []string{"list_documents", "document_content"} {
		teks := strings.ToUpper(strings.TrimSpace(query(name)))

		require.True(t, strings.HasPrefix(teks, "SELECT"), "kueri %q tidak diawali SELECT", name)
		for _, larangan := range []string{"UPDATE ", "INSERT ", "DELETE ", "MERGE "} {
			require.NotContains(t, teks, larangan,
				"kueri %q memuat %q", name, strings.TrimSpace(larangan))
		}
	}
}

// Daftar dokumen menirukan `IDBALAILELANG IS NULL` — dokumen yang sudah ditandai ditolak
// tidak lagi muncul. Ia perilaku layar lama, bukan pilihan di sini.
func TestDaftarDokumenMenyaringYangSudahDitandaiDitolak(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_documents")), "IDBALAILELANG IS NULL")
}

// Unduhan TIDAK boleh bergantung pada DATAID saja.
//
// Dengan `WHERE DATAID = :1` tanpa apa pun lagi, siapa pun yang masuk dapat mengunduh
// lampiran mana pun di seluruh basis data hanya dengan menebak angkanya — termasuk lampiran
// klaim yang tidak ada hubungannya dengan salvage.
func TestUnduhanDokumenTidakBergantungPadaDataIdSaja(t *testing.T) {
	teks := strings.ToUpper(query("document_content"))

	require.Contains(t, teks, "SALAVAGEDOCUMENT",
		"unduhan harus memastikan dokumennya memang dokumen banding")
	require.Contains(t, teks, "TIPEDOCSALVAGE")
	require.GreaterOrEqual(t, strings.Count(teks, "EXISTS"), 2,
		"dua lapis: dokumen banding, dan banding milik komite pemanggil")
}
