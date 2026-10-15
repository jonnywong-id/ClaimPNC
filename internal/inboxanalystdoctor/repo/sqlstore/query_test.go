package sqlstore

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// namedQueries adalah ketiga kueri yang wajib ada di berkas .sql.
var namedQueries = []string{"list_tasks", "check_tables", "check_columns"}

func TestSetiapKueriBernamaAda(t *testing.T) {
	for _, name := range namedQueries {
		require.NotEmptyf(t, query(name), "kueri %s kosong atau tidak ada", name)
	}
}

func TestTidakAdaKueriTakTerpakaiDiBerkasSQL(t *testing.T) {
	// Kueri yang menganggur adalah kode mati yang kelak dikira siap dipakai.
	require.Len(t, queries, len(namedQueries))
}

// TestAliasKueriSamaDenganDaftarDanUrutannya menjaga tiga tempat tetap sepadan.
//
// Urutan kolom pada kueri, isi `taskColumns`, dan urutan pembacaan `scanTask` WAJIB sama.
// Satu kolom yang bergeser akan memindahkan nomor polis ke kolom nama tertanggung — dan
// keduanya bertipe teks, sehingga tidak ada satu pun galat yang muncul.
func TestAliasKueriSamaDenganDaftarDanUrutannya(t *testing.T) {
	text := query("list_tasks")

	aliases := regexp.MustCompile(`(?i)\bAS\s+([A-Z_]+)`).FindAllStringSubmatch(text, -1)
	require.Len(t, aliases, len(taskColumns))

	for i, match := range aliases {
		require.Equalf(t, taskColumns[i], strings.ToUpper(match[1]),
			"alias ke-%d tidak sepadan dengan taskColumns", i+1)
	}
}

// TestKueriDaftarMenyaringKelasObjekKerja mengunci CATATAN 1 pada berkas .sql.
//
// `PC_ASM_FW_GCNMFW_WORK` menampung DUA kelas objek kerja. Report Definition tidak
// menyaringnya karena engine Pega yang melakukannya; SQL langsung harus menuliskannya.
// Melupakannya mencampur berkas penerimaan dokumen ke dalam antrean medis, tanpa galat.
func TestKueriDaftarMenyaringKelasObjekKerja(t *testing.T) {
	require.Contains(t, query("list_tasks"), "'ASM-FW-GCNMFW-Work-PNC'")
}

// TestKueriDaftarMemakaiGabunganDalamKeWorklist mengunci bentuk gabungannya.
//
// `pyJoinType = INNER` pada Report Definition. Menggantinya dengan `EXISTS` akan mengubah
// jumlah baris yang terlihat pengguna — klaim dengan dua penugasan terbuka muncul dua kali di
// Pega, dan itu perilaku yang dibawa (`P-5`).
func TestKueriDaftarMemakaiGabunganDalamKeWorklist(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "INNER JOIN DATAPEGA.PC_ASSIGN_WORKLIST")
	require.Contains(t, text, "A.PXREFOBJECTKEY = W.PZINSKEY")
	require.NotContains(t, text, "EXISTS")
}

// TestKueriDaftarMemakaiWorklistBukanWorkbasket menjaga model penugasannya.
//
// Report Definition menggabung ke `Assign-Worklist` — antrean PER ORANG. Tab RCL/PUCL pada
// modul lain memakai `Assign-Workbasket`, antrean bersama, dan keduanya mudah tertukar saat
// menyalin kueri antarmodul.
func TestKueriDaftarMemakaiWorklistBukanWorkbasket(t *testing.T) {
	text := strings.ToUpper(query("list_tasks"))

	require.Contains(t, text, "PC_ASSIGN_WORKLIST")
	require.NotContains(t, text, "PC_ASSIGN_WORKBASKET")
}

// TestPenyaringStatusMemakaiTidakSamaDengan adalah uji yang paling mudah terbalik.
//
// Filter C berbunyi `!=`. Satu tanda yang salah membalik seluruh isi layar: yang tampil
// menjadi tugas yang sudah tuntas, dan tidak ada apa pun yang menandakannya.
func TestPenyaringStatusMemakaiTidakSamaDengan(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), "W.PYSTATUSWORK <> :3")
}

// TestResolvedRejectedTidakIkutDikecualikan menjaga perbedaan terhadap Inbox Outstanding.
func TestResolvedRejectedTidakIkutDikecualikan(t *testing.T) {
	require.NotContains(t, query("list_tasks"), "Resolved-Rejected")
}

// TestUrutanMenurunDenganPemutusSeri mengunci kedua `pySortType = DESC`.
func TestUrutanMenurunDenganPemutusSeri(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"ORDER BY W.PXCREATEDATETIME DESC, W.PZINSKEY DESC")
}

// TestPaginasiDikerjakanBasisData menjaga halaman dipotong sebelum baris meninggalkannya.
func TestPaginasiDikerjakanBasisData(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"OFFSET :7 ROWS FETCH NEXT :8 ROWS ONLY")
}

// TestJumlahBarisDihitungFungsiJendela menjaga satu perjalanan, bukan dua.
func TestJumlahBarisDihitungFungsiJendela(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), "COUNT(*) OVER ()")
}

// TestSeluruhNilaiLewatParameterBinding adalah uji keamanan, bukan uji gaya.
//
// Kueri lama sistem Pega menyisipkan nilai pengguna langsung ke teks SQL lewat penanda
// `{ASIS:…}` — 538 kemunculan di seluruh export. Larangan perangkaian
// (`08-TECHNICAL-STRATEGY.md` §4.3) tidak dikecualikan oleh keputusan mana pun.
//
// Yang diperiksa: setiap penanda bind `:n` yang dipakai berurutan dari 1, dan tidak ada
// penanda gaya lain yang menandakan perangkaian.
func TestSeluruhNilaiLewatParameterBinding(t *testing.T) {
	text := query("list_tasks")

	markers := regexp.MustCompile(`:\d+`).FindAllString(text, -1)
	require.NotEmpty(t, markers)

	seen := map[string]bool{}
	for _, marker := range markers {
		seen[marker] = true
	}
	for _, expected := range []string{":1", ":2", ":3", ":4", ":5", ":6", ":7", ":8"} {
		require.Truef(t, seen[expected], "penanda bind %s tidak dipakai", expected)
	}

	require.NotContains(t, text, "{ASIS", "perangkaian gaya Pega tidak boleh terbawa")

	// TIDAK ADA perangkaian sama sekali di kueri ini, termasuk untuk wildcard LIKE.
	//
	// Pola `'%' || UPPER(:4) || '%'` sempat dipakai di sini, dan ia membawa cacat yang
	// berbeda dari injeksi: `:4` menjadi muncul TIGA kali, sementara driver menghitung
	// setiap kemunculan sebagai satu variabel yang harus diikat. Hasilnya ORA-01008.
	// Wildcard-nya kini dibentuk `likePattern` di Go.
	require.NotContains(t, text, "||",
		"tidak boleh ada perangkaian apa pun ke teks SQL; wildcard dibentuk di Go")
}

// TestPenandaBindTidakPernahBerulang adalah uji yang menangkap cacat kedua layar ini.
//
// `database/sql` mengirim argumen menurut POSISI, dan driver go-ora menghitung setiap
// kemunculan `:n` sebagai satu variabel yang harus diikat — bukan sebagai rujukan ke
// variabel yang sama. Satu penanda yang ditulis dua kali karena itu menuntut lebih banyak
// ikatan daripada yang dikirim pemanggil, dan Oracle menjawab:
//
//	ORA-01008: not all variables bound
//
// Cacat ini TIDAK terlihat dari membaca kode, tidak tertangkap `go vet`, dan tidak
// tertangkap uji sqlmock — ia hanya muncul saat kueri benar-benar dijalankan. Di layar ini
// ia bahkan tersembunyi di belakang ORA-00904 sampai penyebab pertamanya diperbaiki.
//
// Yang diperiksa: setiap penanda muncul TEPAT SEKALI, dan penomorannya menaik tanpa lubang.
func TestPenandaBindTidakPernahBerulang(t *testing.T) {
	for name := range queries {
		markers := regexp.MustCompile(`:\d+`).FindAllString(query(name), -1)

		counted := map[string]int{}
		for _, marker := range markers {
			counted[marker]++
		}
		for marker, times := range counted {
			require.Equalf(t, 1, times,
				"kueri %s memakai penanda %s sebanyak %d kali; "+
					"setiap kemunculan menuntut satu ikatan tersendiri (ORA-01008)",
				name, marker, times)
		}

		for i := 1; i <= len(markers); i++ {
			require.Containsf(t, markers, ":"+strconv.Itoa(i),
				"kueri %s melompati penanda :%d", name, i)
		}
	}
}

// TestLikePatternMembungkusDanMelepasKarakterKhusus menjaga kotak cari tetap jujur.
//
// Tanpa escape, pencarian "100%" berubah menjadi pola yang mencocokkan APA SAJA — pengguna
// mengira ia menemukan sesuatu, padahal ia hanya mematikan penyaringnya sendiri.
func TestLikePatternMembungkusDanMelepasKarakterKhusus(t *testing.T) {
	require.Equal(t, "", likePattern("   "),
		"kata kunci kosong tidak boleh menjadi pola yang cocok dengan semuanya")
	require.Equal(t, "%PNC-1%", likePattern(" pnc-1 "),
		"pola diseragamkan huruf besar karena sisi SQL memakai UPPER(...)")
	require.Equal(t, `%100\%%`, likePattern("100%"))
	require.Equal(t, `%A\_B%`, likePattern("a_b"))
	require.Equal(t, `%C\D%`, likePattern(`c\d`))
}

// TestPencarianDimatikanSaatKataKunciNULL menjaga kotak cari kosong tidak memaksa pemindaian.
//
// `LIKE '%%'` kebetulan cocok dengan semuanya, tetapi memaksa basis data memeriksa setiap
// baris alih-alih melewati predikatnya.
func TestPencarianDimatikanSaatKataKunciNULL(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), ":4 IS NULL")
}

// TestPerbandinganOperatorTidakPekaHurufBesarKecil mengunci CATATAN 3.
func TestPerbandinganOperatorTidakPekaHurufBesarKecil(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")),
		"UPPER(A.PXASSIGNEDOPERATORID) = UPPER(:2)")
}

// TestTidakAdaPernyataanYangMenulis menjaga `P-1`.
//
// Seluruh tabel yang dibaca modul ini milik Pega selama masa paralel. Satu `UPDATE` yang
// lolos ke sini berarti dua sistem menulis tabel yang sama — dan kegagalannya tidak
// menghasilkan galat, hanya data yang berubah sendiri.
func TestTidakAdaPernyataanYangMenulis(t *testing.T) {
	for _, name := range namedQueries {
		text := strings.ToUpper(query(name))
		for _, forbidden := range []string{"INSERT ", "UPDATE ", "DELETE ", "MERGE ", "TRUNCATE "} {
			require.NotContainsf(t, text, forbidden,
				"kueri %s memuat pernyataan yang menulis: %s", name, forbidden)
		}
	}
}

// TestKueriPeriksaTidakMembacaSatuBarisPun menjaga `-periksa` tetap murah.
func TestKueriPeriksaTidakMembacaSatuBarisPun(t *testing.T) {
	for _, name := range []string{"check_tables", "check_columns"} {
		require.Containsf(t, query(name), "1 = 0",
			"kueri %s harus menolak seluruh baris", name)
	}
}

// TestKueriPeriksaKolomMenutupSeluruhKolomYangDipakai.
//
// Versi sebelumnya memeriksa DUA kolom tebakan lalu berhenti. Keduanya ternyata memang tidak
// ada, dan karena pemeriksaannya berhenti pada temuan pertama, kolom lain tidak pernah
// sempat terperiksa sama sekali.
func TestKueriPeriksaKolomMenutupSeluruhKolomYangDipakai(t *testing.T) {
	text := strings.ToUpper(query("check_columns"))

	for _, column := range []string{
		"W.PYID", "W.POLICYNO", "W.QQNAME", "W.BRANCHNAME", "W.PYORIGUSERID",
		"W.USERTEKNIS_1", "W.PXCREATEDATETIME", "W.PYSTATUSWORK",
		"A.PXTASKLABEL", "A.PXASSIGNEDOPERATORID",
	} {
		require.Containsf(t, text, column,
			"kolom %s dipakai kueri daftar tetapi tidak ikut diperiksa", column)
	}
}

// TestKolomYangTerbuktiTidakAdaTidakDipakaiLagi adalah uji yang mencegah kambuhnya cacat ini.
//
// `ISCOMPLIANCETRANSFER_1` dan `ANALYSTDOCTORREMAKS_1` adalah nama TEBAKAN yang mengikuti
// konvensi `_1`. Katalog Oracle membuktikan keduanya tidak ada, dan selama keduanya dipakai,
// layar ini gagal ORA-00904 pada SETIAP permintaan — bukan sesekali, dan bukan hanya untuk
// sebagian pengguna.
//
// Menyalin kueri dari modul lain adalah cara paling mudah mengembalikannya tanpa sengaja.
func TestKolomYangTerbuktiTidakAdaTidakDipakaiLagi(t *testing.T) {
	for name := range queries {
		text := strings.ToUpper(query(name))
		require.NotContainsf(t, text, "ISCOMPLIANCETRANSFER",
			"kueri %s memakai kolom yang terbukti tidak ada di Oracle", name)
		require.NotContainsf(t, text, "ANALYSTDOCTORREMAKS",
			"kueri %s memakai kolom yang terbukti tidak ada di Oracle", name)
	}
}

// TestAntreanDikenaliDariLabelTahapPenugasan mengunci pengganti penyaring utamanya.
//
// `Flow/Register_Flow.xml` `Assignment13` menamai tahap itu "Analyst Doctor", dan Pega
// menyimpan nama itu sebagai `PXTASKLABEL`. Tanpa penyaring ini, kueri mengembalikan SELURUH
// tugas worklist milik pemanggil sebagai tugas medis — tanpa satu pun galat.
func TestAntreanDikenaliDariLabelTahapPenugasan(t *testing.T) {
	require.Contains(t, strings.ToUpper(query("list_tasks")), "A.PXTASKLABEL = :1")
}

// TestKomentarTidakIkutDikirimKeBasisData menjaga pemecah berkas .sql bekerja.
//
// Berkas ini memuat komentar kepala yang panjang. Bila pemecahnya tidak membuangnya, seluruh
// penjelasan itu ikut terkirim pada setiap permintaan.
func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range namedQueries {
		require.NotContainsf(t, query(name), "--",
			"kueri %s masih memuat baris komentar", name)
	}
}
