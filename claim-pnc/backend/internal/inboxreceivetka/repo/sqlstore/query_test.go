package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxreceivetka"
)

// usedQueries adalah seluruh kueri yang dipanggil kode modul ini.
var usedQueries = []string{
	"tka_inbox_list",
	"tka_inbox_search",
	"tka_inbox_find_one",
	"tka_claim_set_document_date",
	"tka_inbox_check_table",
	"tka_inbox_check_claim_column",
	"tka_inbox_count_waiting",
	"tka_inbox_count_pega_only",
	"tka_inbox_count_orphan_claim",
	"tka_inbox_count_missing_participant",
	"tka_inbox_sample_registered_on",
}

// scannedQueries adalah kueri yang barisnya dibaca scanTask.
var scannedQueries = []string{
	"tka_inbox_list",
	"tka_inbox_search",
	"tka_inbox_find_one",
	"tka_inbox_check_table",
}

// listQueries adalah kueri yang mengembalikan daftar untuk layar.
var listQueries = []string{"tka_inbox_list", "tka_inbox_search"}

// Seluruh kueri yang dipanggil kode harus benar-benar ada di berkas .sql. Tanpa uji ini,
// salah ketik nama kueri baru ketahuan saat pengguna memanggil endpointnya.
func TestAllUsedQueriesExist(t *testing.T) {
	for _, name := range usedQueries {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() { _ = getQuery(name) })
			require.NotEmpty(t, strings.TrimSpace(getQuery(name)))
		})
	}
}

func TestMissingQueryPanics(t *testing.T) {
	require.Panics(t, func() { _ = getQuery("kueri_yang_tidak_pernah_ada") })
}

// Disiplin SQL portabel (`D-20`) hanya bertahan bila ditegakkan perkakas, bukan diingat
// orang. Uji ini adalah penegaknya sampai pemeriksaan pola SQL berjalan di CI.
func TestQueriesFollowPortableSQLDiscipline(t *testing.T) {
	forbidden := map[string]string{
		"SELECT *":   "kolom harus disebut namanya; kolom baru tidak boleh diam-diam mengubah perilaku",
		"NVL(":       "pakai COALESCE",
		"SYSDATE":    "pakai CURRENT_TIMESTAMP",
		"DECODE(":    "pakai CASE WHEN",
		"ROWNUM":     "pakai OFFSET ... FETCH NEXT ... ROWS ONLY",
		"INSTR(":     "pakai POSITION",
		"LISTAGG(":   "pakai STRING_AGG",
		"TO_CHAR(":   "pemformatan tanggal dan angka dilakukan di Go",
		"TO_DATE(":   "penguraian tanggal dilakukan di Go; lihat parseRegisterDate",
		"TO_NUMBER(": "penguraian angka dilakukan di Go",
		"TRUNC(":     "pemotongan tanggal dilakukan di Go; lihat Completion.Clean",
		"LPAD(":      "pemformatan angka dilakukan di Go",
		"FROM DUAL":  "tidak ada kueri modul ini yang membutuhkannya",
	}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for pattern, reason := range forbidden {
			require.NotContainsf(t, uppercase, pattern,
				"kueri %q memakai %q — %s", name, pattern, reason)
		}
		require.NotContainsf(t, uppercase, "(+)",
			"kueri %q memakai outer join gaya Oracle; pakai LEFT JOIN", name)
	}
}

// Modul ini MENULIS, tetapi hanya dengan satu UPDATE ke satu tabel.
//
//   - tanpa DELETE dan TRUNCATE  — `D-66` menetapkan tidak ada penghapusan fisik atas data
//     bernilai bisnis
//   - tanpa INSERT               — modul ini tidak pernah membuat pekerjaan, hanya
//     menyelesaikannya
//   - tanpa MERGE                — ia dapat menyisipkan diam-diam bila barisnya tidak ada,
//     dan itu justru keadaan yang harus DITOLAK (ErrClaimMissing)
func TestQueriesOnlyWriteWithUpdate(t *testing.T) {
	forbidden := []string{"INSERT ", "DELETE ", "MERGE ", "TRUNCATE "}

	for name, text := range query {
		uppercase := strings.ToUpper(text)
		for _, verb := range forbidden {
			require.NotContainsf(t, uppercase, verb,
				"kueri %q memakai %s — modul ini hanya boleh UPDATE; lihat banner berkas .sql",
				name, verb)
		}
	}
}

// Tabel engine Pega TIDAK boleh ditulis, dan uji ini yang menjaganya.
//
// # Kenapa ini uji tersendiri, bukan sekadar catatan
//
// Pega menyimpan nilai sebenarnya di BLOB kasus dan menyalinnya ke kolom. Menulis kolomnya
// dari luar akan tertimpa tanpa satu pun tanda begitu Pega menyimpan kasus itu lagi — dan
// kegagalannya berupa isian petugas yang hilang diam-diam, bukan galat yang terlihat.
//
// Ia akan gagal pada hari seseorang menambahkan `UPDATE DATAPEGA...` demi membuat barisnya
// hilang lebih cepat. Bila itu memang dikehendaki, keputusannya menempuh `P-1` dan `D-63`
// lebih dulu — bukan lewat satu baris SQL.
func TestPegaWorkTableIsNeverWritten(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		if !strings.Contains(uppercase, "UPDATE ") {
			continue
		}
		require.NotContainsf(t, uppercase, "UPDATE DATAPEGA",
			"kueri %q menulis ke tabel engine Pega; nilainya akan tertimpa tanpa tanda", name)
	}
}

// Setiap UPDATE wajib punya klausa WHERE.
//
// Uji paling sederhana di berkas ini, dan yang paling mahal bila hilang. `UPDATE` tanpa
// `WHERE` pada `T_CLAIM_PNC` akan menuliskan satu tanggal yang sama ke SELURUH klaim di
// entitas itu — puluhan juta baris data nilai klaim — dan pernyataannya berhasil tanpa satu
// pun galat.
func TestEveryUpdateHasWhereClause(t *testing.T) {
	for name, text := range query {
		uppercase := strings.ToUpper(text)
		if !strings.Contains(uppercase, "UPDATE ") {
			continue
		}
		require.Containsf(t, uppercase, "WHERE",
			"kueri %q memakai UPDATE tanpa WHERE — ia akan menyentuh seluruh tabel", name)
	}
}

// UPDATE-nya wajib menyertakan penjaga `TGLDOKLENGKAP IS NULL`.
//
// Ia yang menggantikan kunci idempotensi (`10-API-STRATEGY.md` §7): tanpa penjaga itu, dua
// permintaan bersamaan sama-sama berhasil dan surelnya terkirim dua kali.
func TestUpdateGuardsAgainstDoubleFill(t *testing.T) {
	uppercase := strings.ToUpper(getQuery("tka_claim_set_document_date"))
	require.Contains(t, uppercase, "TGLDOKLENGKAP IS NULL",
		"UPDATE wajib menolak baris yang tanggalnya sudah terisi")
}

// Ketiga penyaring Report Definition WAJIB ada pada setiap kueri daftar.
//
// Ketiganya yang membuat layar ini menjadi Inbox Receive TKA dan bukan daftar seluruh klaim.
// Yang ketinggalan tidak menghasilkan galat — ia menghasilkan daftar yang terlihat masuk
// akal dan memuat pekerjaan yang bukan haknya.
func TestListQueriesCarryAllThreeFilters(t *testing.T) {
	for _, name := range append(append([]string{}, listQueries...), "tka_inbox_find_one") {
		t.Run(name, func(t *testing.T) {
			uppercase := strings.ToUpper(getQuery(name))
			require.Contains(t, uppercase, "W.TKA_1 = '1'",
				"penyaring A: penanda klaim TKA")
			require.Contains(t, uppercase, "W.TANGGALDOKLENGKAP IS NULL",
				"penyaring B sisi Pega")
			require.Contains(t, uppercase, "C.TGLDOKLENGKAP IS NULL",
				"penyaring B sisi aplikasi ini — yang membuat barisnya hilang setelah Submit")
			require.Contains(t, uppercase, "W.PYSTATUSWORK <>",
				"penyaring C: pekerjaan yang sudah selesai dibuang")
			require.Contains(t, uppercase, "ASM-FW-GCNMFW-WORK-PNC",
				"kueri wajib membatasi kelas kasus; satu tabel Pega memuat banyak kelas")
		})
	}
}

// Kedua gabungan WAJIB berupa LEFT JOIN.
//
// Bukan gaya penulisan. Gabungan INNER akan membuang pekerjaan yang klaimnya belum ada di
// tabel bisnis — pekerjaan yang HILANG dari layar tanpa satu pun tanda, dan yang justru
// ditampilkan Pega. Yang dipilih sebaliknya: barisnya tetap tampil, dan penolakannya terjadi
// saat Submit dengan pesan yang menjelaskan sebabnya.
func TestScannedQueriesUseLeftJoin(t *testing.T) {
	for _, name := range scannedQueries {
		t.Run(name, func(t *testing.T) {
			uppercase := strings.ToUpper(getQuery(name))
			require.Equal(t, 2, strings.Count(uppercase, "LEFT JOIN"),
				"kedua gabungan wajib LEFT; INNER akan menyembunyikan pekerjaan")
			require.NotContains(t, uppercase, "INNER JOIN",
				"gabungan INNER menyembunyikan baris yang klaimnya tidak ditemukan")
		})
	}
}

// Tidak satu pun kueri boleh mengunci baris pada tabel engine Pega.
//
// `SELECT ... FOR UPDATE` di sana akan menahan baris yang sedang dilayani aplikasi lama, dan
// kunci yang ditahan permintaan kita dapat menghentikan alur kerja Pega. Penguncian tidak
// dibutuhkan: penjaga pada UPDATE yang mengamankan pengisian ganda.
func TestNoQueryLocksThePegaWorkTable(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text), "FOR UPDATE",
			"kueri %q mengunci baris; modul ini mengandalkan penjaga pada UPDATE", name)
	}
}

// Nilai status pekerjaan TIDAK boleh tertulis di dalam teks SQL.
//
// Ia dikirim sebagai parameter meski nilainya konstanta di dalam kode kita sendiri.
// Larangan merangkai nilai ke dalam teks SQL tidak mengenal pengecualian "nilainya toh dari
// kode sendiri" — aturan yang berlaku kadang-kadang bukan aturan
// (`08-TECHNICAL-STRATEGY.md` §4.3).
func TestResolvedStatusNeverAppearsInSQLText(t *testing.T) {
	for name, text := range query {
		require.NotContainsf(t, strings.ToUpper(text),
			strings.ToUpper(inboxreceivetka.ResolvedWorkStatus),
			"kueri %q menuliskan nilai status ke dalam teks SQL; kirim sebagai parameter",
			name)
	}
}

// Keempat kueri yang barisnya dipindai scanTask HARUS mengembalikan kedelapan alias yang
// sama pada urutan yang sama.
//
// Satu pemindai melayani keempatnya. Alias yang berbeda urutannya tidak menghasilkan galat
// kompilasi maupun galat runtime — ia menghasilkan Nama Peserta yang muncul di kolom Nama
// Tertanggung, dan tidak ada apa pun di layar yang menandakannya.
func TestScannedQueriesShareTheSameColumnOrder(t *testing.T) {
	expected := []string{
		"REFERENCE",
		"CLAIM_KEY",
		"CLAIM_NUMBER",
		"POLICY_NUMBER",
		"INSURED_NAME",
		"PARTICIPANT_NAME",
		"DATE_OF_LOSS",
		"REGISTERED_ON",
	}

	for _, name := range scannedQueries {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, expected, aliasOrder(getQuery(name)))
		})
	}
}

// Urutan daftar WAJIB memakai tanggal registrasi, bukan kolom lain.
//
// Ia kolom yang SAMA dengan yang Report Definition pakai (`pySortOrder = 1`), dan
// mengurutkannya sebagai teks benar karena formatnya `yyyymmdd` berlebar tetap.
func TestListQueriesOrderByRegistrationDate(t *testing.T) {
	for _, name := range listQueries {
		t.Run(name, func(t *testing.T) {
			require.Contains(t, strings.ToUpper(getQuery(name)),
				"ORDER BY W.REGISTERDATE_1, W.PYID",
				"urutannya wajib sama dengan Report Definition, dengan pemutus yang unik")
		})
	}
}

// aliasOrder membaca alias `AS <NAMA>` sesuai urutan kemunculannya di teks kueri.
//
// Hanya alias pada tingkat SELECT terluar yang dihitung; kueri yang dipindai di berkas ini
// tidak memakai subquery ber-`AS` sama sekali, sehingga pemindaian sederhana ini cukup dan
// tidak menuntut pengurai SQL.
func aliasOrder(text string) []string {
	var result []string
	for _, line := range strings.Split(strings.ToUpper(text), "\n") {
		index := strings.LastIndex(line, " AS ")
		if index < 0 {
			continue
		}
		alias := strings.TrimSpace(line[index+len(" AS "):])
		alias = strings.TrimSuffix(alias, ",")
		if alias != "" {
			result = append(result, alias)
		}
	}
	return result
}
