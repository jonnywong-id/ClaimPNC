package sqlstore

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// TestTidakAdaKueriMenyentuhDATAPEGA menahan kedua tabel lama kembali diam-diam.
//
// Ia uji terpenting di berkas ini. Ketetapan Work Owner 2026-09-28 memindahkan sumber
// dashboard dari `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` + `PC_ASSIGN_WORKLIST` ke satu tabel datar,
// dan pelanggaran paling mudah terjadi bukan lewat penulisan ulang melainkan lewat SATU join
// yang ditambahkan "sementara" saat sebuah kolom terasa kurang.
func TestTidakAdaKueriMenyentuhDATAPEGA(t *testing.T) {
	for name, text := range queries {
		lower := strings.ToLower(text)

		require.NotContains(t, lower, "datapega",
			"kueri %q menyentuh skema DATAPEGA; sumber modul ini POOLDATA.T_CLAIMLIST_ADMIN", name)
		require.NotContains(t, lower, "pc_asm_fw_gcnmfw_work",
			"kueri %q membaca tabel kerja Pega", name)
		require.NotContains(t, lower, "pc_assign_worklist",
			"kueri %q membaca tabel penugasan Pega", name)
	}
}

// TestSetiapPenandaBindMunculTepatSekaliDanBerurutan menahan kelas cacat yang HANYA muncul di
// Oracle.
//
// Driver mengikat argumen menurut urutan kemunculan penanda, bukan menurut nomornya. Penanda
// yang dipakai dua kali karena itu menuntut dua argumen, dan bila lupa kuerinya gagal
// ORA-01008 — sementara seluruh uji memori tetap hijau. Cacat itu sudah pernah menggigit modul
// Inbox RCL (`catatan-pengembangan.md` §63.2).
func TestSetiapPenandaBindMunculTepatSekaliDanBerurutan(t *testing.T) {
	marker := regexp.MustCompile(`:(\d+)`)

	for name, text := range queries {
		found := marker.FindAllStringSubmatch(text, -1)
		if len(found) == 0 {
			continue
		}

		numbers := make([]int, 0, len(found))
		seen := map[int]int{}
		for _, match := range found {
			number, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			numbers = append(numbers, number)
			seen[number]++
		}

		for number, count := range seen {
			require.Equalf(t, 1, count,
				"kueri %q memakai penanda :%d sebanyak %d kali. Driver mengikat menurut "+
					"urutan kemunculan, bukan menurut nomor penanda — penanda berulang "+
					"menuntut argumen berulang dan gagal ORA-01008 bila lupa.",
				name, number, count)
		}

		ascending := make([]int, len(numbers))
		copy(ascending, numbers)
		sort.Ints(ascending)
		require.Equalf(t, ascending, numbers,
			"penanda bind pada kueri %q tidak menaik: %v", name, numbers)

		require.Equalf(t, 1, numbers[0],
			"penanda bind pada kueri %q tidak dimulai dari :1", name)
		require.Equalf(t, len(numbers), numbers[len(numbers)-1],
			"penanda bind pada kueri %q berlubang: %v", name, numbers)
	}
}

// TestKueriDashboardMenyaringKelasKasus menahan penyaring PXOBJCLASS dihapus.
//
// Tabel sumbernya memuat lebih dari satu kelas kasus — 872 baris Work-PNC dan 142 baris
// Work-ReceiveDocument saat diperiksa 2026-09-28. Tanpa penyaring ini baris Receive Document
// ikut terhitung sebagai klaim, dan karena keduanya sama-sama punya PYID tidak ada yang
// tampak salah.
func TestKueriDashboardMenyaringKelasKasus(t *testing.T) {
	for _, name := range []string{
		"count_outstanding", "dashboard_os_pic", "dashboard_os_business_group",
	} {
		text := query(name)
		require.Containsf(t, text, "PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
			"kueri %q tidak menyaring kelas kasus", name)
	}
}

// TestKueriOutstandingMenyaringStatusKerja menahan cacat kueri lama kembali.
//
// `GetBisnisGroupDashboardOS` di Pega TIDAK menyaring status kerja sama sekali, sehingga
// dashboard berjudul "Outstanding" ikut menghitung klaim yang sudah selesai. Penyaringnya
// ditambahkan, dan uji ini yang menjaganya tetap ada pada KETIGA kueri.
func TestKueriOutstandingMenyaringStatusKerja(t *testing.T) {
	for _, name := range []string{
		"count_outstanding", "dashboard_os_pic", "dashboard_os_business_group",
	} {
		text := query(name)
		require.Containsf(t, text,
			"PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')",
			"kueri %q tidak menyaring status kerja — dashboard Outstanding akan ikut "+
				"menghitung klaim yang sudah selesai", name)
	}
}

// TestKueriOutstandingTidakMenggabungDaftarObjek menahan cacat penghitungan ganda.
//
// `CountOutstandingManager` di Pega menggabung `t_claim_objectlist` tanpa memilih satu kolom
// pun darinya, sehingga klaim berobjek banyak terhitung berkali-kali pada pencacah yang
// menamai dirinya jumlah klaim.
func TestKueriOutstandingTidakMenggabungDaftarObjek(t *testing.T) {
	for _, name := range []string{
		"count_outstanding", "dashboard_os_pic", "dashboard_os_business_group",
	} {
		require.NotContainsf(t, strings.ToLower(query(name)), "t_claim_objectlist",
			"kueri %q menggabung daftar objek; klaim berobjek banyak akan terhitung "+
				"berkali-kali", name)
	}
}

// TestSetiapKeputusanMenyaringStatusMenunggu menahan keputusan orang lain tertimpa.
//
// Tanpa penyaring ini, penyelia yang membuka daftar lama akan menimpa keputusan orang lain —
// dan jumlah baris yang berubah akan tetap sama dengan jumlah yang dipilih, sehingga tidak ada
// yang tahu.
func TestSetiapKeputusanMenyaringStatusMenunggu(t *testing.T) {
	guards := map[string]string{
		"decide_bengkel":            "TRIM(APPROVAL) = '0'",
		"decide_panel":              "TRIM(APPROVAL) = '0'",
		"decide_nomor_rangka":       "STS_AKSEP IS NULL",
		"decide_sparepart":          "TRIM(APPROVAL) = '0'",
		"decide_kategori_sparepart": "TRIM(APPROVAL) = '0'",
		"decide_tipe_sparepart":     "TRIM(APPROVAL) = '0'",
		"decide_grouping_sparepart": "TRIM(APPROVAL) = '0'",
		"decide_payment_akseptasi":  "TRIM(STSAPP) = '0'",
		"decide_penolakan_klaim":    "TRIM(STATUS) = '0'",
	}

	for name, guard := range guards {
		require.Containsf(t, query(name), guard,
			"pernyataan %q tidak menyaring status menunggu — ia akan menimpa keputusan "+
				"orang lain tanpa satu pun tanda", name)
	}
}

// TestSetiapAntreanPunyaPencacahDanPernyataanKeputusan memastikan kesembilan antrean lengkap.
//
// Antrean yang punya daftar tetapi tidak punya pencacah akan hilang dari kepala layar; yang
// punya pencacah tetapi tidak punya pernyataan keputusan akan menggambar tombol yang selalu
// gagal.
func TestSetiapAntreanPunyaPencacahDanPernyataanKeputusan(t *testing.T) {
	for _, tab := range inboxmanager.QueueTabs() {
		listName, known := queueQueries[tab.Code]
		require.Truef(t, known, "tab antrean %q (%s) tidak punya kueri daftar",
			tab.Code, tab.Name)
		require.NotPanicsf(t, func() { query(listName) },
			"kueri daftar %q tidak ada di berkas .sql", listName)

		// Antrean BACA-SAJA berhenti di sini: ia memang tidak punya pernyataan keputusan,
		// dan menuntutnya akan memaksa kami menulis jalur tulis yang sectionnya di Pega tidak
		// punya tombolnya. Lihat "Approval Progress Klaim".
		if !tab.Decision.Decidable {
			continue
		}

		decision := inboxmanager.Decision{Tab: tab, Verdict: inboxmanager.VerdictReject}
		key := "X"
		if tab.Code == inboxmanager.TabNomorRangka {
			key = "a|b|c|d"
		}

		name, args, err := decideArgs(decision, key)
		require.NoErrorf(t, err, "tab antrean %q tidak punya pernyataan keputusan", tab.Code)
		require.NotEmpty(t, args)
		require.NotPanicsf(t, func() { query(name) },
			"pernyataan keputusan %q tidak ada di berkas .sql", name)
	}
}

// TestJumlahArgumenKeputusanSamaDenganJumlahPenanda menahan ORA-01008 pada jalur TULIS.
//
// Ia lebih penting daripada padanannya di jalur baca: kegagalan bind pada pernyataan keputusan
// terjadi SETELAH penyelia menekan tombol, dan yang dilihatnya hanyalah galat tanpa keterangan
// atas pekerjaan yang ia kira sudah selesai.
func TestJumlahArgumenKeputusanSamaDenganJumlahPenanda(t *testing.T) {
	marker := regexp.MustCompile(`:(\d+)`)

	for _, tab := range inboxmanager.QueueTabs() {
		if !tab.Decision.Decidable {
			continue
		}

		key := "X"
		if tab.Code == inboxmanager.TabNomorRangka {
			key = "a|b|c|d"
		}

		name, args, err := decideArgs(inboxmanager.Decision{
			Tab:     tab,
			Verdict: inboxmanager.VerdictReject,
			Reason:  "alasan",
			Caller:  inboxmanager.Caller{Login: "JONNY"},
		}, key)
		require.NoError(t, err)

		markers := marker.FindAllString(query(name), -1)
		require.Lenf(t, args, len(markers),
			"pernyataan %q punya %d penanda bind tetapi dikirimi %d argumen",
			name, len(markers), len(args))
	}
}

// TestKueriAntreanMengembalikanKunciDanKolomSebanyakTabnya menahan pergeseran kolom.
//
// Setiap kueri antrean mengembalikan ROW_KEY lebih dulu, lalu satu kolom per kolom tab DALAM
// URUTAN YANG SAMA. Satu kolom yang bergeser akan menaruh nama bengkel ke kolom cabang tanpa
// satu pun galat.
func TestKueriAntreanMengembalikanKunciDanKolomSebanyakTabnya(t *testing.T) {
	column := regexp.MustCompile(`(?i)\bAS\s+(ROW_KEY|COL\d+)\b`)

	for _, tab := range inboxmanager.QueueTabs() {
		name := queueQueries[tab.Code]
		aliases := column.FindAllStringSubmatch(query(name), -1)

		require.Lenf(t, aliases, len(tab.Columns)+1,
			"kueri %q mengembalikan %d kolom beralias, sedangkan tab %q punya %d kolom "+
				"ditambah satu kunci",
			name, len(aliases), tab.Name, len(tab.Columns))

		require.Equalf(t, "ROW_KEY", strings.ToUpper(aliases[0][1]),
			"kolom pertama kueri %q bukan ROW_KEY", name)
	}
}

// TestTidakAdaPolaSQLTerlarang menahan gaya SQL yang menghalangi portabilitas.
//
// Daftarnya mengikuti `08-TECHNICAL-STRATEGY.md` §4.3 dan `09-DATABASE-STRATEGY.md` §4. Ia
// menggigit modul ini secara khusus pada `TO_CHAR` dan `TRUNC`: kueri Pega yang digantikannya
// memakai keduanya pada kolom tanggal, dan keduanya menyentuh kolomnya pada setiap baris
// sehingga index tidak terpakai.
func TestTidakAdaPolaSQLTerlarang(t *testing.T) {
	forbidden := []struct {
		pattern *regexp.Regexp
		why     string
	}{
		{regexp.MustCompile(`(?i)\bNVL\s*\(`), "pakai COALESCE"},
		{regexp.MustCompile(`(?i)\bSYSDATE\b`), "pakai CURRENT_TIMESTAMP"},
		{regexp.MustCompile(`(?i)\bDECODE\s*\(`), "pakai CASE WHEN"},
		{regexp.MustCompile(`(?i)\bROWNUM\b`), "pakai OFFSET ... FETCH NEXT"},
		{regexp.MustCompile(`(?i)\bINSTR\s*\(`), "pakai POSITION"},
		{regexp.MustCompile(`(?i)\bLISTAGG\s*\(`), "pakai STRING_AGG"},
		{regexp.MustCompile(`(?i)\bFROM\s+DUAL\b`), "hilangkan klausa FROM"},
		{regexp.MustCompile(`(?i)\bTO_CHAR\s*\(`), "format tanggal dan angka di Go"},
		{regexp.MustCompile(`(?i)\bTRUNC\s*\(`), "pakai selang setengah terbuka"},
		{regexp.MustCompile(`(?i)SELECT\s+\*`), "sebutkan nama kolom"},
	}

	for name, text := range queries {
		for _, rule := range forbidden {
			// `SELECT z.*` pada subkueri dikecualikan: ia menyebut seluruh kolom sebuah
			// subkueri yang daftar kolomnya SUDAH disebut satu per satu tepat di atasnya,
			// dan bentuk itu dibawa apa adanya dari kueri Pega.
			if rule.pattern.String() == `(?i)SELECT\s+\*` &&
				strings.Contains(text, "SELECT z.*") {
				continue
			}
			require.NotRegexpf(t, rule.pattern, text,
				"kueri %q memakai pola SQL terlarang — %s", name, rule.why)
		}
	}
}

// TestTidakAdaNilaiDirangkaiKeTeksSQL menahan pola `{ASIS:…}` warisan kembali.
//
// Sembilan kueri layar ini di Pega menyisipkan potongan SQL langsung ke teksnya. Pola itu
// risiko injeksi sekaligus penghalang portabilitas (`03-CURRENT-ARCHITECTURE.md` §4.5).
func TestTidakAdaNilaiDirangkaiKeTeksSQL(t *testing.T) {
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %q masih memuat pola ASIS", name)
		require.NotContainsf(t, text, "' || '", "kueri %q merangkai teks SQL", name)
	}
}

// TestKueriLiniBisnisSamaDenganModulLain menahan dua modul membaca kolom yang sama dengan cara
// berbeda.
//
// Ejaan `LINE_BUSINESS` BERGARIS BAWAH, dan migrasi yang menulisnya tanpa garis bawah sudah
// dicabut sebelum dijalankan justru karena itu.
func TestKueriLiniBisnisSamaDenganModulLain(t *testing.T) {
	text := query("line_business_for")

	require.Contains(t, text, "LINE_BUSINESS")
	require.Contains(t, text, "POOLDATA.M_LOGIN_PNC")
	require.Contains(t, text, "UPPER(TRIM(p.LOGIN_ID)) = :1")
	require.NotContains(t, text, "LINEBUSINESS")
}
