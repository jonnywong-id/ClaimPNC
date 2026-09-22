package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxautoclaim"
)

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestEveryUsedQueryExists(t *testing.T) {
	usedNames := []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
		"auto_claim_batch_count",
		"auto_claim_batch_count_by_company",
		"auto_claim_company_list",
		"auto_claim_company_summary",
		"auto_claim_line_list",
		"auto_claim_line_list_succeeded",
		"auto_claim_line_list_failed",
		"auto_claim_line_count",
		"auto_claim_line_count_succeeded",
		"auto_claim_line_count_failed",
		"auto_claim_export",
		"auto_claim_export_succeeded",
		"auto_claim_export_failed",
		"auto_claim_batch_exists",
		"auto_claim_receiver",
		"auto_claim_policy",
		"auto_claim_batch_number_used",
		"auto_claim_line_insert",
		"auto_claim_check_table",
		"auto_claim_check_master",
		"auto_claim_check_currency",
	}
	for _, name := range usedNames {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = kueriAneka(name) })
			require.NotEmpty(t, strings.TrimSpace(kueriAneka(name)))
		})
	}
}

// kueriAneka mengambil kueri tab ANEKA, dan menyebut tabnya TERANG-TERANGAN — bukan lewat
// DefaultSource.
//
// Sebagian besar uji di bawah memeriksa BENTUK kueri, dan bentuknya sama untuk ketiga tab
// karena ketiganya berasal dari satu templat. Yang memeriksa ketiganya sekaligus adalah
// uji yang menelusuri seluruh isi peta resolved.
//
// Sebelumnya ia memakai DefaultSource, dan ketika tab bawaan berpindah ke Asuransi Kredit
// seluruh uji di bawah diam-diam berganti memeriksa tabel lain — namanya tetap "Aneka"
// sementara isinya bukan. Uji yang menyebut tabnya sendiri tidak dapat berubah arti
// karena keputusan di tempat lain.
func kueriAneka(name string) string {
	return getQueryFor(inboxautoclaim.SourceAneka, name)
}

// Setiap tab menghasilkan kueri yang LENGKAP dan BERBEDA tabelnya.
//
// Ini penjaga langsung atas cara ketiga tab dibangun: satu templat, diisi dari enum
// tertutup. Bila placeholder-nya salah ketik atau satu tab kehilangan isian, kuerinya
// akan berisi teks placeholder mentah yang ditolak Oracle dengan galat sintaks — dan galat itu
// baru muncul saat pengguna membuka tab tersebut.
func TestEverySourceResolvesToItsOwnTable(t *testing.T) {
	terlihat := map[string]bool{}

	for _, source := range inboxautoclaim.AllSource() {
		info, exists := source.Info()
		require.Truef(t, exists, "tab %q tidak punya tabel", source)
		require.False(t, terlihat[info.Table], "dua tab tidak boleh berbagi tabel yang sama")
		terlihat[info.Table] = true

		teks := getQueryFor(source, "auto_claim_batch_list")
		require.NotContains(t, teks, "{{", "masih ada placeholder yang belum diisi")
		require.Contains(t, teks, info.Table)
		require.Contains(t, teks, "A."+info.CompanyColumn)
	}

	// Tab Kredit memakai kolom yang BERBEDA. Bila ini pernah gagal, ketiga tab diam-diam
	// membaca kolom yang sama dan tab Kredit tidak akan mengembalikan satu baris pun.
	require.Contains(t, getQueryFor(inboxautoclaim.SourceKredit, "auto_claim_batch_list"), "A.AGENID")
	require.NotContains(t, getQueryFor(inboxautoclaim.SourceKredit, "auto_claim_batch_list"), "A.INISIALID")
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = kueriAneka("kueri_yang_tidak_pernah_ada") })
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
	}

	for name, text := range resolved {
		upperCase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, upperCase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, upperCase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Tidak ada DB Link yang tersisa.
//
// Kueri ekspor Pega mengambil "No Ref Bank" lewat
// `gl.t_claim_asuransi_credit@asmd.sinarmas.co.id`. D-25 mengganti seluruh DB Link
// dengan API, dan API penggantinya belum ada (R-03) — jadi kolomnya kosong, BUKAN
// diambil lewat DB Link "sementara". Uji ini yang menjaga jalan pintas itu tidak masuk.
func TestQueriesUseNoDatabaseLink(t *testing.T) {
	for name, text := range resolved {
		require.NotContainsf(t, strings.ToUpper(text), "@ASMD",
			"kueri %q memakai DB Link; D-25 menggantinya dengan API", name)
	}
}

// Tidak ada pemanggilan stored procedure (D-02).
//
// POOLDATA.INSERT_AUTOCLAIM adalah godaan nyata di modul ini: sistem lama memakainya
// untuk menulis ke tabel yang sama, dan memanggilnya akan terasa "setara". Ia tetap
// tidak boleh dipakai — source-nya pun tidak ada di export, sehingga yang dipanggil
// adalah logika yang tidak pernah dibaca siapa pun di tim ini.
func TestQueriesCallNoStoredProcedure(t *testing.T) {
	for name, text := range resolved {
		upperCase := strings.ToUpper(text)
		require.NotContainsf(t, upperCase, "INSERT_AUTOCLAIM",
			"kueri %q memanggil stored procedure; D-02 melarangnya", name)
		require.NotContainsf(t, upperCase, "BEGIN ",
			"kueri %q memakai blok PL/SQL; ia tidak portabel", name)
	}
}

// Nilai selalu lewat parameter binding. Kueri yang merangkai nilai ke dalam teks SQL
// adalah celah injeksi — pola yang diwarisi sistem lama lewat {ASIS:...}, dan penyaring
// nama perusahaan pada layar INI justru salah satu contohnya.
func TestQueriesUseParameterBinding(t *testing.T) {
	parameterised := []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
		"auto_claim_batch_count_by_company",
		"auto_claim_line_list",
		"auto_claim_line_list_succeeded",
		"auto_claim_line_list_failed",
		"auto_claim_export",
		"auto_claim_export_succeeded",
		"auto_claim_export_failed",
		"auto_claim_batch_exists",
		"auto_claim_receiver",
		"auto_claim_policy",
		"auto_claim_batch_number_used",
		"auto_claim_line_insert",
	}
	for _, name := range parameterised {
		require.Containsf(t, kueriAneka(name), ":1",
			"kueri %q harus memakai parameter binding", name)
	}
}

// Nomor bind WAJIB berurutan 1..N menurut urutan kemunculannya.
//
// # Kenapa uji ini ada
//
// Ia lahir dari cacat sungguhan. `auto_claim_batch_list` sempat memakai `:1` DUA KALI
// untuk nilai yang sama, lalu `:2` dan `:3`, sementara kodenya mengirim tiga nilai.
// Terlihat rapi, dan salah: driver mengikat menurut URUTAN KEMUNCULAN, bukan menurut
// nomornya. Bind terakhir tidak pernah terisi, dan Oracle menolak seluruh pernyataan:
//
//	ORA-01008: not all variables bound
//
// # Kenapa tidak ada uji lain yang menangkapnya
//
// Karena tidak ada yang menguraikan SQL. Penyimpanan memori tidak memakai kueri ini sama
// sekali, dan `TestQueriesUseParameterBinding` hanya memastikan `:1` ADA — bukan bahwa
// penomorannya masuk akal. Cacatnya baru muncul saat layar dibuka terhadap Oracle
// sungguhan, di tangan pengguna.
//
// Uji ini menutup seluruh kelasnya sekaligus: nomor yang melompat, nomor yang terbalik,
// dan nomor yang diulang. Ketiganya menghasilkan ORA-01008 yang sama membingungkannya.
func TestQueryBindsAreNumberedInOrder(t *testing.T) {
	pattern := regexp.MustCompile(`:(\d+)`)

	for name, text := range resolved {
		match := pattern.FindAllStringSubmatch(text, -1)
		if len(match) == 0 {
			continue
		}

		t.Run(name, func(t *testing.T) {
			for position, m := range match {
				number, err := strconv.Atoi(m[1])
				require.NoError(t, err)
				require.Equalf(t, position+1, number,
					"bind ke-%d tertulis %s; nomor bind wajib berurutan 1..N menurut "+
						"urutan kemunculan. Nilai yang sama yang dipakai dua kali tetap "+
						"menuntut DUA nomor, dan kodenya mengirimkannya dua kali",
					position+1, m[0])
			}
		})
	}
}

// Tidak satu pun kueri boleh memuat nama perusahaan sebagai teks di dalam SQL.
//
// Kueri Pega aslinya merangkai penyaringnya lewat `{ASIS:...CaseID}` — teks SQL yang
// dibentuk di luar kueri lalu disisipkan mentah. Uji ini memastikan pola itu tidak
// kembali lewat "perbaikan" yang terasa praktis.
func TestCompanyFilterUsesCodeNotName(t *testing.T) {
	text := strings.ToUpper(kueriAneka("auto_claim_batch_list_by_company"))

	// Yang diperiksa: penyaringnya membandingkan KOLOM KODE lewat bind. Nomor bind-nya
	// TIDAK dipaku — ia bergeser secara sah setiap kali bind lain ditambah atau dikurangi
	// di depannya, dan uji yang memakunya akan merah pada perubahan yang benar.
	require.Regexp(t, `A\.INISIALID = :\d+`, text,
		"penyaring perusahaan memakai KODE lewat bind, bukan nama")
	require.NotContains(t, text, "NAMA_PENERIMA =",
		"nama perusahaan tidak pernah menjadi penyaring")
}

// Tidak satu pun kueri menyisipkan potongan SQL yang dibentuk di luar dirinya.
//
// `{ASIS:TemporaryInboxKasirAutoClaim.CaseID}` pada BrowseClaimSPKAutoClaim adalah
// contoh persisnya: nilai properti klipboard disisipkan MENTAH ke dalam teks SQL. Di
// sistem baru, setiap variasi penyaring adalah kueri tersendiri di berkas ini.
func TestNoQueryBuildsSQLFromText(t *testing.T) {
	for name, text := range resolved {
		require.NotContainsf(t, text, "{ASIS", "kueri %q mewarisi pola {ASIS:...}", name)
		require.NotContainsf(t, text, "%s", "kueri %q dirangkai dengan fmt; pakai bind", name)
	}
}

// Join ke master WAJIB LEFT.
//
// Kueri Pega aslinya memakai INNER JOIN gaya lama:
// `FROM POOLDATA.TMP_BATCH_AUTO_CLAIM a, POOLDATA.M_AUTO_CLAIM_PNC b
//
//	WHERE a.INISIALID = b.INISIALID`.
//
// Akibatnya batch yang kode perusahaannya tidak ada di master HILANG dari layar tanpa
// satu pun tanda — padahal baris seperti itu justru yang tidak akan pernah berhasil
// diproses. Ini selisih yang DISENGAJA terhadap Pega, dicatat di
// keputusan-implementasi.md §18, dan uji ini menjaganya tidak "dirapikan" kembali.
func TestMasterJoinStaysLeft(t *testing.T) {
	for _, name := range []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(kueriAneka(name))
			require.Contains(t, text, "LEFT JOIN POOLDATA.M_AUTO_CLAIM_PNC")
		})
	}
}

// Daftar perusahaan dibaca dari MASTER, bukan dari tabel batch.
//
// `BrowseCompanyClaimCredit-SQL.xml` membaca M_AUTO_CLAIM_PNC apa adanya. Perusahaan
// yang baru didaftarkan dan belum punya batch satu pun TETAP muncul di penyaring —
// tanpa itu, petugas tidak dapat memastikan batch-nya memang belum ada; ia hanya
// melihat namanya hilang.
func TestCompanyListReadsMasterOnly(t *testing.T) {
	text := strings.ToUpper(kueriAneka("auto_claim_company_list"))
	require.Contains(t, text, "POOLDATA.M_AUTO_CLAIM_PNC")
	require.NotContains(t, text, "TMP_BATCH_AUTO_CLAIM",
		"daftar perusahaan tidak menyentuh tabel batch")
}

// Baris yang BELUM diproses bukan baris gagal.
//
// Kueri export sistem lama menegaskannya:
// "AND TMP_MESSAGE!='Sukses Klaim' and TMP_MESSAGE is not null". Tanpa IS NOT NULL,
// setiap baris yang menunggu proses ikut terhitung gagal — dan angka di grid menjadi
// salah tanpa satu pun galat.
func TestFailedFilterExcludesUnprocessedRows(t *testing.T) {
	for _, name := range []string{
		"auto_claim_line_list_failed",
		"auto_claim_line_count_failed",
		"auto_claim_export_failed",
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(kueriAneka(name))
			require.Contains(t, text, "A.TMP_MESSAGE IS NOT NULL")
			require.Contains(t, text, "A.TMP_MESSAGE <> :3")
		})
	}
}

// Penyisipan WAJIB mengisi TGLPROSES.
//
// Ia bagian PRIMARY KEY (NOPOLIS, TGLPROSES, TGLKEJADIAN) menurut
// Database/CREATE_TABLE_1.sql. Rekonstruksi pertama modul ini TIDAK mengisinya, dan
// setiap penyisipan akan ditolak ORA-01400 — cacat yang hanya ketahuan saat menembak
// basis data sungguhan, tidak pernah saat menguji dengan penyimpanan memori.
func TestInsertFillsProcessedDate(t *testing.T) {
	text := strings.ToUpper(kueriAneka("auto_claim_line_insert"))
	require.Contains(t, text, "TGLPROSES")
	require.Contains(t, text, "CURRENT_TIMESTAMP")
}

// Penyisipan MENGISI ketiga kolom penanda — dan itu memang perilaku Pega.
//
// Versi pertama modul ini menegakkan kebalikannya: IDPEGA, NOAKSEPTASI, dan TMP_MESSAGE
// harus NULL supaya batch baru terambil pemrosesan. Itu benar untuk baris yang LOLOS,
// dan salah untuk baris yang GAGAL.
//
// `Activity/InsertKlaimToTable_Other-Act.xml` mengisi ketiganya dengan pesan galat yang
// sama ketika sebuah baris tidak lolos pemeriksaan polis. Baris itu tetap disisipkan —
// justru supaya terlihat petugas di grid dan ikut keluar di berkas ekspor GAGAL.
//
// Yang menjaga batch tetap terambil pemrosesan bukan ketiadaan kolomnya di kueri,
// melainkan NILAI NULL yang dikirim untuk baris yang lolos.
func TestInsertCarriesFailureMarkers(t *testing.T) {
	text := strings.ToUpper(kueriAneka("auto_claim_line_insert"))
	columns := text[strings.Index(text, "("):strings.Index(text, "VALUES")]

	for _, marker := range []string{"IDPEGA", "NOAKSEPTASI", "TMP_MESSAGE"} {
		require.Containsf(t, columns, marker,
			"kolom %s diisi pesan galat untuk baris yang gagal", marker)
	}

	// PROGRESS tetap tidak disentuh: tidak satu pun activity unggahan menulisinya, dan
	// GroupingAutoClaim2 menuntut `progress is null OR progress='0'`.
	require.NotContains(t, columns, "PROGRESS",
		"PROGRESS tidak pernah ditulis saat unggah")
}

// Paginasi memakai OFFSET ... FETCH NEXT, bukan ROWNUM (D-20, 09-DATABASE-STRATEGY.md §3.3).
//
// Kueri Pega aslinya memakai ROWNUM berlapis. Ini SELISIH PERILAKU, bukan pemeliharaan:
// ROWNUM dihitung sebelum ORDER BY pada pola tertentu, sehingga halaman lama dan halaman
// baru dapat berisi baris yang berbeda. Dicatat sebagai selisih yang diketahui.
func TestPagedQueriesUsePortablePagination(t *testing.T) {
	for _, name := range []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
		"auto_claim_line_list",
		"auto_claim_line_list_succeeded",
		"auto_claim_line_list_failed",
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(kueriAneka(name))
			require.Contains(t, text, "OFFSET")
			require.Contains(t, text, "FETCH NEXT")
		})
	}
}

// Kueri berpaginasi WAJIB punya ORDER BY.
//
// OFFSET tanpa urutan yang ditetapkan tidak menjamin apa pun: basis data boleh
// mengembalikan baris dalam urutan berbeda pada setiap permintaan, sehingga berpindah
// halaman dapat melewatkan atau menggandakan baris.
func TestPagedQueriesAreOrdered(t *testing.T) {
	for _, name := range []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
		"auto_claim_line_list",
		"auto_claim_line_list_succeeded",
		"auto_claim_line_list_failed",
	} {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, strings.ToUpper(kueriAneka(name)), "ORDER BY")
		})
	}
}

// Daftar batch diurutkan MENURUN.
//
// `ORDER BY BATCH DESC` pada BrowseClaimSPKAutoClaim. Versi pertama modul ini
// mengurutkan menaik, sehingga batch yang baru saja diunggah petugas berada di halaman
// TERAKHIR — pada perusahaan dengan puluhan batch, ia tidak akan menemukannya.
func TestBatchListShowsNewestFirst(t *testing.T) {
	for _, name := range []string{"auto_claim_batch_list", "auto_claim_batch_list_by_company"} {
		t.Run(name, func(t *testing.T) {
			text := strings.ToUpper(kueriAneka(name))
			require.Contains(t, text, "DESC", "batch terbaru harus tampil lebih dulu")
		})
	}
}

// Kueri pemeriksa tabel tidak boleh mengambil satu baris pun — ia dijalankan terhadap
// produksi pada mode periksa.
func TestCheckTableFetchesNoRows(t *testing.T) {
	for _, name := range []string{
		"auto_claim_check_table",
		"auto_claim_check_master",
		"auto_claim_check_currency",
	} {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, kueriAneka(name), "1 = 0")
		})
	}
}

// Ringkasan memakai PENGELOMPOKAN yang sama dengan kueri hitung grid.
//
// Ini yang menjamin angka pada panel ringkasan dan total paginasi grid selalu cocok.
// Bila salah satunya diubah tanpa yang lain, pengguna melihat "27" lalu mengeklik dan
// mendapat jumlah baris yang berbeda — tanpa satu pun galat.
func TestSummaryGroupsTheSameWayAsTheGrid(t *testing.T) {
	summary := strings.ToUpper(kueriAneka("auto_claim_company_summary"))
	count := strings.ToUpper(kueriAneka("auto_claim_batch_count"))

	// Dibandingkan per BAGIAN, bukan sebagai satu untai panjang: keduanya boleh berbeda
	// lekuk barisnya, dan yang wajib sama adalah kolom pengelompokannya.
	for _, bagian := range []string{
		"A.INISIALID, A.BATCH, A.USERINPUT",
		"EXTRACT(YEAR FROM A.TGLPROSES)",
		"EXTRACT(MONTH FROM A.TGLPROSES)",
		"EXTRACT(DAY FROM A.TGLPROSES)",
	} {
		require.Containsf(t, count, bagian, "premis uji ini: kueri hitung grid mengelompokkan menurut %s", bagian)
		require.Containsf(t, summary, bagian, "ringkasan wajib mengelompokkan menurut %s juga", bagian)
	}
}

// Ringkasan dihitung dari TABEL BATCH TAB INI, lalu namanya diambil dari master.
//
// Arahnya menentukan apa yang dilihat pengguna, dan pernah salah dua kali:
//
//	dari master (FULL OUTER JOIN) -> ketiga tab menampilkan daftar perusahaan yang SAMA,
//	                                 penuh baris berjumlah 0 yang bila diklik menghasilkan
//	                                 grid kosong. Dilaporkan sebagai "penyaring tidak
//	                                 berfungsi" (2026-09-20)
//	dari batch (sekarang)         -> tiap tab memuat perusahaannya sendiri
//
// LEFT JOIN, bukan INNER seperti Pega: batch yang kodenya tidak ada di master tetap
// terhitung, sebab ia tetap tampil di grid dan justru baris itulah yang tidak akan pernah
// berhasil diproses.
func TestSummaryCountsFromBatchTableNotMaster(t *testing.T) {
	text := strings.ToUpper(kueriAneka("auto_claim_company_summary"))

	require.Contains(t, text, "LEFT JOIN POOLDATA.M_AUTO_CLAIM_PNC")
	require.NotContains(t, text, "FULL OUTER JOIN",
		"master tidak boleh menjadi sisi yang dipertahankan; perusahaan tanpa batch tidak ditampilkan")
	require.NotContains(t, text, "COALESCE(R.JUMLAH_BATCH",
		"tidak ada lagi baris berjumlah NULL yang perlu dijadikan 0")
}

// Kueri berpaginasi WAJIB punya urutan yang menentukan satu susunan tunggal.
//
// `OFFSET … FETCH NEXT` memotong hasil menurut urutan. Bila ada baris yang seri, basis
// data boleh menyusunnya berbeda pada tiap eksekusi — dan halaman 2 lalu dapat mengulang
// baris halaman 1 sementara baris lain tidak pernah muncul, tanpa galat apa pun.
//
// Daftar batch mengurutkan BATCH DESC, dan BATCH berulang antar perusahaan dan antar
// tanggal. Ketiga kolom pemecah serinya karena itu bagian dari kebenaran kueri, bukan
// kerapian.
func TestPagedQueriesOrderDeterministically(t *testing.T) {
	for _, name := range []string{"auto_claim_batch_list", "auto_claim_batch_list_by_company"} {
		t.Run(name, func(t *testing.T) {
			text := kueriAneka(name)
			require.Contains(t, text, "OFFSET", "premis uji ini: kueri ini berpaginasi")

			// Keempatnya bersama-sama adalah kunci GROUP BY, sehingga tidak ada dua baris
			// hasil yang dapat seri.
			for _, kolom := range []string{"A.BATCH DESC", "A.INISIALID", "MIN(A.TGLPROSES) DESC", "A.USERINPUT"} {
				require.Containsf(t, text, kolom,
					"%s: urutannya harus memuat %s supaya paginasinya tidak dapat mengulang baris", name, kolom)
			}
		})
	}
}

// Pengelompokan per HARI tidak boleh memakai CAST(... AS DATE).
//
// # Kenapa uji ini ada
//
// Tipe DATE Oracle MEMBAWA JAM. `CAST(timestamp AS DATE)` karena itu tidak memotong apa
// pun di Oracle, sementara cast yang sama memotong ke hari di PostgreSQL — satu kueri,
// dua perilaku, persis yang dilarang D-20.
//
// Akibatnya bukan teoretis: satu hari kalender terpecah menjadi satu kelompok per detik
// yang berbeda, sehingga grid menampilkan baris yang kembar persis — seluruh kolomnya
// sama, termasuk tanggalnya, karena Go memformat hanya sampai hari. Terukur 38 baris di
// tab Asuransi Kredit dan 2 di ANEKA sebelum diperbaiki.
//
// Penggantinya EXTRACT, yang berarti sama di kedua basis data.
func TestDayGroupingDoesNotRelyOnDateCast(t *testing.T) {
	berkelompokPerHari := []string{
		"auto_claim_batch_list",
		"auto_claim_batch_list_by_company",
		"auto_claim_batch_count",
		"auto_claim_batch_count_by_company",
		"auto_claim_company_summary",
	}
	for _, name := range berkelompokPerHari {
		t.Run(name, func(t *testing.T) {
			text := kueriAneka(name)

			require.NotContains(t, text, "CAST(A.TGLPROSES AS DATE)",
				"DATE Oracle membawa jam; cast ini tidak memotong ke hari di sana")
			for _, bagian := range []string{
				"EXTRACT(YEAR FROM A.TGLPROSES)",
				"EXTRACT(MONTH FROM A.TGLPROSES)",
				"EXTRACT(DAY FROM A.TGLPROSES)",
			} {
				require.Containsf(t, text, bagian,
					"%s: pengelompokan per hari harus memakai %s", name, bagian)
			}
		})
	}
}
