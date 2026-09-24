package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// samplePage adalah paginasi lengkap, dipakai memanggil penyusun argumen tiap kueri.
func samplePage() inboxmanagerreceivepucl.Pagination {
	return inboxmanagerreceivepucl.Pagination{Page: 3, Size: 50}
}

// sampleQuery menyusun permintaan untuk sebuah tab.
func sampleQuery(t *testing.T, code string) inboxmanagerreceivepucl.Query {
	t.Helper()

	tab, found := inboxmanagerreceivepucl.FindTab(code)
	require.Truef(t, found, "tab %s tidak ditemukan", code)

	return inboxmanagerreceivepucl.Query{
		Tab:    tab,
		Caller: inboxmanagerreceivepucl.Caller{Login: "PENYELIACONTOH"},
	}
}

type planCase struct {
	plan  plan
	query inboxmanagerreceivepucl.Query
}

// allPlans mengembalikan ketiga rencana kueri beserta permintaannya.
//
// Ketiganya disebut lengkap, bukan disimpulkan dari daftar tab: daftar yang dihitung sendiri
// oleh uji akan ikut salah bila pemilihan kuerinya salah.
func allPlans(t *testing.T) map[string]planCase {
	t.Helper()

	cases := []struct {
		label string
		tab   string
	}{
		{"receive_pa", inboxmanagerreceivepucl.TabReceivePA},
		{"receive_nonmbu", inboxmanagerreceivepucl.TabReceiveNonMBU},
		{"rclpucl", inboxmanagerreceivepucl.TabRCLPUCL},
	}

	result := map[string]planCase{}
	for _, c := range cases {
		q := sampleQuery(t, c.tab)
		selected, err := planFor(q)
		require.NoErrorf(t, err, "%s belum punya kueri", c.label)
		result[c.label] = planCase{plan: selected, query: q}
	}

	return result
}

func TestEverySelectableTabHasQuery(t *testing.T) {
	for _, tab := range inboxmanagerreceivepucl.Tabs() {
		if tab.Blocked {
			continue
		}
		_, err := planFor(inboxmanagerreceivepucl.Query{Tab: tab})
		require.NoErrorf(t, err, "tab %s (%s) belum punya kueri", tab.Code, tab.Name)
	}
}

func TestEveryTabPicksItsOwnQuery(t *testing.T) {
	// Ketiga tab dilayani kueri yang BERBEDA-BEDA. Kalau ada dua yang sama, salah satu
	// penyaringnya pasti hilang — dan yang paling mungkin hilang adalah penyaring kelas
	// objek kerja, yang berarti berkas penerimaan dokumen bercampur dengan klaim.
	plans := allPlans(t)

	seen := map[string]string{}
	for label, entry := range plans {
		if previous, clash := seen[entry.plan.name]; clash {
			t.Fatalf("%s dan %s memakai kueri yang sama (%s)",
				previous, label, entry.plan.name)
		}
		seen[entry.plan.name] = label
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { query("kueri_yang_tidak_pernah_ada") })
}

func TestEveryListQueryReturnsTheSameAliases(t *testing.T) {
	// Satu pemindai melayani ketiga kueri. Begitu satu kueri memakai alias yang berbeda
	// atau urutannya bergeser, pemindai itu memasukkan nilai ke isian yang salah — dan
	// akibatnya BUKAN galat, melainkan kolom yang tertukar di layar.
	//
	// Di modul ini bahayanya lebih besar daripada biasa: sembilan dari tujuh belas kolom
	// dikirim sebagai `CAST(NULL …)` pada salah satu kuerinya, sehingga pergeseran satu
	// posisi menghasilkan kolom yang kosong — bentuk yang mudah dikira "data memang belum
	// diisi".
	//
	// Hanya alias di UJUNG baris kolom yang dihitung. Tanpa syarat itu,
	// `CAST(NULL AS VARCHAR2(100))` ikut terbaca sebagai alias.
	alias := regexp.MustCompile(`(?im)\bAS\s+([A-Z_]+)\s*,?\s*$`)

	for _, name := range listQueries {
		found := []string{}
		for _, match := range alias.FindAllStringSubmatch(query(name), -1) {
			found = append(found, strings.ToUpper(match[1]))
		}
		require.Equalf(t, resultColumns, found,
			"alias kueri %s berbeda dari resultColumns", name)
	}
}

func TestEveryQueryFiltersItsOwnWorkClass(t *testing.T) {
	// Kedua kelas objek kerja hidup di SATU tabel dan hanya dibedakan PXOBJCLASS. Kueri
	// yang lupa menyaringnya mencampur berkas penerimaan dokumen dengan klaim — dan
	// keduanya punya PYID, POLICYNO, serta QQNAME, sehingga hasilnya tidak menghasilkan
	// satu pun galat.
	for _, name := range receiveQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper,
			strings.ToUpper(inboxmanagerreceivepucl.WorkClassReceiveDocument),
			"kueri %s tidak menyaring kelas berkas penerimaan dokumen", name)
		require.NotContainsf(t, upper,
			"= '"+strings.ToUpper(inboxmanagerreceivepucl.WorkClassClaim)+"'",
			"kueri %s menyaring kelas klaim; ia tidak seharusnya", name)
	}

	upper := strings.ToUpper(query("list_rclpucl"))
	require.Contains(t, upper,
		strings.ToUpper(inboxmanagerreceivepucl.WorkClassClaim),
		"kueri RCL/PUCL tidak menyaring kelas klaim")
	require.NotContains(t, upper,
		strings.ToUpper(inboxmanagerreceivepucl.WorkClassReceiveDocument),
		"kueri RCL/PUCL menyaring kelas berkas penerimaan dokumen")
}

func TestReceiveQueriesReadTheAssignmentTableOfTheirOwn(t *testing.T) {
	// Kedua tab Receive membaca penugasan PER ORANG; tab RCL/PUCL membaca antrean BERSAMA.
	// Kueri yang membaca tabel yang salah mengembalikan nol baris tanpa satu pun galat —
	// terbaca persis seperti antrean yang memang kosong.
	for _, name := range receiveQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "DATAPEGA.PC_ASSIGN_WORKLIST",
			"kueri %s tidak menggabung tabel penugasan per orang", name)
		require.NotContainsf(t, upper, "DATAPEGA.PC_ASSIGN_WORKBASKET",
			"kueri %s menggabung antrean bersama; ia tidak seharusnya", name)
	}

	upper := strings.ToUpper(query("list_rclpucl"))
	require.Contains(t, upper, "DATAPEGA.PC_ASSIGN_WORKBASKET",
		"kueri RCL/PUCL tidak menggabung antrean bersama")
	require.NotContains(t, upper, "DATAPEGA.PC_ASSIGN_WORKLIST",
		"kueri RCL/PUCL menggabung penugasan per orang")
}

func TestOnlyReceiveQueriesFilterTheGroupPanel(t *testing.T) {
	// Group Panel adalah pengganti `.ReceiveDocument.TypeOfClaim` yang tidak punya kolom.
	// Ia HANYA berlaku pada kedua tab Receive; membocorkannya ke tab RCL/PUCL akan
	// menyaring klaim menurut lini bisnis yang layar lama tidak pernah saring.
	for _, name := range receiveQueries {
		require.Containsf(t, strings.ToUpper(query(name)), "GROUPPANEL_1",
			"kueri %s tidak menyaring Group Panel", name)
	}
	require.NotContains(t, strings.ToUpper(query("list_rclpucl")), "GROUPPANEL_1",
		"kueri RCL/PUCL menyaring Group Panel; ia tidak seharusnya")
}

func TestTheTwoReceiveQueriesFilterTheGroupPanelInOppositeDirections(t *testing.T) {
	// Keduanya mengikat NILAI yang sama (kode Group Panel PA) dan berbeda hanya pada ARAH
	// pembandingnya. Bila keduanya memakai arah yang sama, satu tab akan menampilkan isi
	// tab yang lain — dan kolomnya identik, sehingga tidak ada apa pun di layar yang
	// menandakannya.
	pa := strings.ToUpper(query("list_receive_pa"))
	nonMBU := strings.ToUpper(query("list_receive_non_mbu"))

	require.Contains(t, pa, "W.GROUPPANEL_1 = :1",
		"kueri PA tidak membandingkan Group Panel dengan kesamaan")
	require.Contains(t, nonMBU, "W.GROUPPANEL_1 <> :1",
		"kueri NONMBU tidak membandingkan Group Panel dengan ketidaksamaan")
}

func TestReceiveQueriesBindTheGroupPanelInsteadOfWritingIt(t *testing.T) {
	// Kode Group Panel dikirim sebagai BIND, bukan ditulis di dalam SQL. Nilainya karena
	// itu hanya hidup di satu tempat — konstanta domain — dan penyimpanan memori membaca
	// konstanta yang sama lewat penerjemah yang sama.
	plans := allPlans(t)

	for _, label := range []string{"receive_pa", "receive_nonmbu"} {
		entry := plans[label]
		require.Equalf(t,
			inboxmanagerreceivepucl.GroupPanelPA,
			entry.plan.args(samplePage())[0],
			"%s wajib mengikat kode Group Panel PA", label)
	}

	for _, name := range receiveQueries {
		require.NotContainsf(t, query(name),
			"'"+inboxmanagerreceivepucl.GroupPanelPA+"'",
			"kueri %s menulis kode Group Panel langsung di dalam SQL", name)
	}
}

func TestRCLPUCLQueryBindsTheWorkbasketAndCompletedStatus(t *testing.T) {
	// Keduanya dikirim sebagai bind. Nama antrean bersama BUKAN sekadar kerapian: ia
	// penyaring yang TIDAK ADA di Report Definition-nya dan diambil dari dua kueri Pega
	// lain, sehingga ia harus hidup di satu tempat yang dapat ditunjuk saat ditinjau ulang.
	plans := allPlans(t)
	entry := plans["rclpucl"]
	args := entry.plan.args(samplePage())

	require.Equal(t, inboxmanagerreceivepucl.WorkStatusCompleted, args[0],
		"kueri RCL/PUCL tidak mengikat status kerja yang dikecualikan")
	require.Equal(t, inboxmanagerreceivepucl.RCLPUCLWorkbasket, args[1],
		"kueri RCL/PUCL tidak mengikat akun antrean bersama")

	require.NotContains(t, query("list_rclpucl"),
		"'"+inboxmanagerreceivepucl.RCLPUCLWorkbasket+"'",
		"kueri RCL/PUCL menulis nama antrean langsung di dalam SQL")
}

func TestRCLPUCLQueryTranslatesTheTrackCodeWithoutElse(t *testing.T) {
	// Penerjemahan `RCL_PUCL_1` ditulis sebagai CASE tanpa ELSE, persis seperti
	// `GetReminderPUCL-SQL.xml`. Menambahkan ELSE akan mengisi sel dengan teks untuk jalur
	// yang tidak dikenali — dan sel kosong adalah jawaban yang benar untuk keadaan itu.
	text := query("list_rclpucl")

	require.Contains(t, text, "'"+inboxmanagerreceivepucl.TrackRCL+"'",
		"kueri RCL/PUCL tidak menerjemahkan kode jalur RCL")
	require.Contains(t, text, "'"+inboxmanagerreceivepucl.TrackPUCL+"'",
		"kueri RCL/PUCL tidak menerjemahkan kode jalur PUCL")
	require.NotContains(t, strings.ToUpper(text), "ELSE",
		"kueri RCL/PUCL memakai ELSE; sistem lama tidak")
}

func TestReminderOnlyFiltersAreNotCarried(t *testing.T) {
	// Ketiga penyaring ini milik JOB PENGINGAT (`ReminderPUCL-SQL.xml`), bukan milik
	// antrean inbox. Membawanya akan menyembunyikan klaim yang suratnya belum dicetak —
	// padahal justru itu yang menunggu tindakan.
	upper := strings.ToUpper(query("list_rclpucl"))

	require.NotContains(t, upper, "TANGGALCETAKDOKUMENPUCL_1 IS NOT NULL",
		"kueri RCL/PUCL membawa penyaring job pengingat")
	require.NotContains(t, upper, "PUCLAPPROVE_1",
		"kueri RCL/PUCL membawa penyaring job pengingat")
	require.NotContains(t, upper, "MSIG_1",
		"kueri RCL/PUCL membawa penyaring job pengingat")
}

func TestBindCountMatchesSuppliedArguments(t *testing.T) {
	// Argumen yang kurang menghasilkan ORA-01008 saat permintaan pertama datang; argumen
	// yang berlebih menghasilkan ORA-01036. Keduanya hanya terlihat di produksi bila tidak
	// diuji di sini, karena kuerinya tidak pernah dijalankan saat kompilasi.
	//
	// Di modul ini jumlah bind-nya BERBEDA antar kueri — tiga untuk Receive, empat untuk
	// RCL/PUCL — sehingga uji ini yang paling mungkin menangkap kekeliruan saat kueri
	// keempat ditambahkan kelak.
	bind := regexp.MustCompile(`:(\d+)`)

	for label, entry := range allPlans(t) {
		text := query(entry.plan.name)

		highest := 0
		for _, match := range bind.FindAllStringSubmatch(text, -1) {
			index, err := strconv.Atoi(match[1])
			require.NoError(t, err)
			if index > highest {
				highest = index
			}
		}

		args := entry.plan.args(samplePage())
		require.Lenf(t, args, highest,
			"%s memakai bind tertinggi :%d tetapi menyiapkan %d argumen",
			label, highest, len(args))
	}
}

func TestPaginationArgumentsFollowTheRequestedPage(t *testing.T) {
	// Offset dan ukuran diambil dari paginasi yang SUDAH dinormalkan. Mengambilnya dari
	// yang diminta akan membuat `halaman=0` menghasilkan offset negatif — yang di Oracle
	// bukan galat melainkan halaman pertama, sehingga cacatnya tidak pernah terlihat.
	page := inboxmanagerreceivepucl.Pagination{Page: 3, Size: 10}

	for label, entry := range allPlans(t) {
		args := entry.plan.args(page)
		require.Equalf(t, 20, args[len(args)-2], "%s: offset halaman ketiga", label)
		require.Equalf(t, 10, args[len(args)-1], "%s: ukuran halaman", label)
	}
}

func TestNoValueIsConcatenatedIntoSQL(t *testing.T) {
	// `GetDataPUCLRCLForDailyReport-SQL.xml` menyisipkan `{TempRCLPUCLReport.AlasanKlaim}`
	// dan saudaranya langsung ke teks SQL sebagai batas tanggal. Larangan perangkaian
	// berlaku penuh di sini (`08-TECHNICAL-STRATEGY.md` §4.3).
	for name, text := range queries {
		require.NotContainsf(t, text, "{ASIS", "kueri %s memuat pola {ASIS:…} warisan", name)
		require.NotContainsf(t, text, "{Temp", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "{Inputdata", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "{Operator", "kueri %s menyisipkan properti Pega", name)
		require.NotContainsf(t, text, "%s", "kueri %s tampak dirangkai lewat fmt", name)
		require.NotContainsf(t, text, "||", "kueri %s merangkai teks di dalam SQL", name)
	}
}

func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	// Daftar terlarang `08-TECHNICAL-STRATEGY.md` §4.3.
	//
	// `TRUNC(` dan `TO_DATE(` patut diperhatikan khusus di modul ini: kueri lama
	// `GetDataPUCLRCLForDailyReport` memakai keduanya untuk membandingkan tanggal, dan
	// modul ini tidak membawa perbandingan itu sama sekali.
	forbidden := []string{
		"NVL(", "SYSDATE", "DECODE(", "ROWNUM", "INSTR(", "LISTAGG(",
		"FROM DUAL", "ADD_MONTHS(", "MONTHS_BETWEEN(", "TO_CHAR(", "TO_DATE(",
		"TRUNC(", "SELECT *",
	}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, pattern := range forbidden {
			require.NotContainsf(t, upper, pattern,
				"kueri %s memakai %s yang tidak portabel", name, pattern)
		}
	}
}

func TestQueriesNeverWrite(t *testing.T) {
	// SELURUH tabel yang dibaca modul ini milik sistem lama. Menulis satu saja melanggar
	// `P-1`, dan akibatnya bukan galat melainkan dua sistem yang saling menimpa.
	writing := []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range queries {
		upper := strings.ToUpper(text)
		for _, verb := range writing {
			require.NotContainsf(t, upper, verb,
				"kueri %s tampak menulis (%s); modul ini hanya membaca",
				name, strings.TrimSpace(verb))
		}
	}
}

func TestEveryListQueryPaginatesAndOrders(t *testing.T) {
	// Di sini halaman dipotong basis data. Kueri yang lupa memaginasi akan menarik seluruh
	// antrean ke memori aplikasi, dan kueri yang lupa mengurutkan membuat satu baris
	// muncul di dua halaman sekaligus.
	for _, name := range listQueries {
		upper := strings.ToUpper(query(name))
		require.Containsf(t, upper, "OFFSET ", "kueri %s tidak memaginasi", name)
		require.Containsf(t, upper, "FETCH NEXT", "kueri %s tidak membatasi jumlah baris", name)
		require.Containsf(t, upper, "ORDER BY", "kueri %s tidak menetapkan urutan", name)
		require.Containsf(t, upper, "COUNT(*) OVER ()",
			"kueri %s tidak membawa jumlah seluruh baris", name)
	}
}

func TestQueriesTouchOnlyTheExpectedTables(t *testing.T) {
	// Daftar tabel yang boleh disentuh ditulis tegas. Tanpa ini, satu gabungan tambahan
	// yang ditambahkan kemudian dapat menarik data dari tabel yang belum pernah ditinjau
	// kepemilikannya (`P-1`) maupun kewenangan bacanya.
	allowed := []string{
		"DATAPEGA.PC_ASM_FW_GCNMFW_WORK",
		"DATAPEGA.PC_ASSIGN_WORKLIST",
		"DATAPEGA.PC_ASSIGN_WORKBASKET",
		"POOLDATA.T_CLAIM_RECIVEDCLAIM",
	}

	table := regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+([A-Z_]+\.[A-Z_]+)`)

	for name, text := range queries {
		for _, match := range table.FindAllStringSubmatch(text, -1) {
			found := strings.ToUpper(match[1])
			require.Containsf(t, allowed, found,
				"kueri %s menyentuh tabel di luar daftar: %s", name, found)
		}
	}
}

func TestQueriesNeverCrossDBLink(t *testing.T) {
	// DB Link tidak punya padanan di PostgreSQL dan diganti pemanggilan API (`D-25`).
	// Satu `@` yang masuk diam-diam akan menggagalkan perpindahan basis data tanpa satu
	// pun tanda sampai cutover.
	for name, text := range queries {
		require.NotContainsf(t, text, "@ASMD", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SIMASNET", "kueri %s menembus DB Link", name)
		require.NotContainsf(t, text, "@SMI", "kueri %s menembus DB Link", name)
	}
}

func TestBothCheckQueriesReturnExactlyOneRow(t *testing.T) {
	// Mode `-periksa` memindai satu nilai dari tiap kueri. Kueri yang memilih KOLOM dengan
	// `WHERE 1 = 0` tidak mengembalikan baris sama sekali, sehingga pemindaiannya
	// menghasilkan sql.ErrNoRows — terbaca sebagai kegagalan hak baca padahal tabelnya
	// justru terbaca dengan baik.
	for _, name := range []string{"check_receive", "check_rclpucl"} {
		require.Containsf(t, strings.ToUpper(query(name)), "COUNT(*)",
			"kueri %s tidak menjamin satu baris hasil", name)
	}
}
