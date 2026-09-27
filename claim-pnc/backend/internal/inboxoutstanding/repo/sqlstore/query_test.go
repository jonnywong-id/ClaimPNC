package sqlstore

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

// Uji di berkas ini berjalan TANPA basis data.
//
// Yang diperiksa adalah bentuk SQL dan pengikatan parameternya — dua hal yang bila salah
// menghasilkan kegagalan yang mahal: galat pengikatan di produksi, atau lebih buruk,
// penyaring pemilik pekerjaan yang tidak berlaku tanpa satu pun galat.

// kueriModul adalah kedua kueri DAFTAR, dipakai hampir seluruh uji di bawah.
var kueriModul = []string{"my_inbox_list", "my_inbox_count"}

// kueriUnduhan adalah kedua kueri EXPORT.
//
// Ia daftar tersendiri karena syaratnya memang BERBEDA — export tidak menyaring pemilik
// pekerjaan sama sekali. Menggabungkannya dengan kueriModul akan memaksa salah satu uji
// dilonggarkan, dan yang dilonggarkan pasti penjaga penyaring pemilik.
var kueriUnduhan = []string{"my_inbox_export", "my_inbox_export_count"}

// kueriSemua dipakai uji struktural yang berlaku untuk seluruh kueri modul ini.
var kueriSemua = append(append([]string{}, kueriModul...),
	append(kueriUnduhan, "legacy_operator_for", "line_business_for")...)

func TestKeduaKueriTerbaca(t *testing.T) {
	for _, name := range kueriSemua {
		require.NotEmpty(t, query(name), "kueri %s kosong", name)
	}
}

func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range kueriSemua {
		require.NotContains(t, query(name), "-- ", "komentar baris tidak boleh ikut terkirim")
	}
}

// Inilah uji yang menjaga janji terpenting layar ini: daftarnya TERIKAT pada pemiliknya.
//
// Tanpa penyaring ini, layar bernama "My Inbox" menampilkan pekerjaan SELURUH operator —
// terisi, tampak wajar, dan salah tanpa satu pun galat.
func TestKeduaKueriMenyaringPemilikPekerjaan(t *testing.T) {
	for _, name := range kueriModul {
		// DUA identitas untuk satu orang: login baru berbentuk email, klaim warisan
		// tertugas ke nama operator Pega. Lihat Filter.AssignedToLegacy.
		require.Contains(t, query(name), "UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)",
			"kueri %s kehilangan penyaring pemilik", name)
	}
}

// Kueri UNDUHAN justru TIDAK boleh menyaring pemilik pekerjaan.
//
// Ini kebalikan uji di atas, dan sengaja ditulis sebagai uji tersendiri: export pernah
// memanggil ulang daftar sehingga mewarisi penyaring itu, dan akibatnya petugas yang
// inbox-nya kosong mengunduh berkas kosong — padahal di Pega berkasnya tetap berisi.
func TestKueriUnduhanTidakMenyaringPemilikPekerjaan(t *testing.T) {
	for _, name := range kueriUnduhan {
		require.NotContains(t, query(name), "PXASSIGNEDOPERATORID) IN",
			"kueri %s tidak boleh menyaring pemilik", name)
		require.NotContains(t, query(name), "PXASSIGNEDOPERATORID) =",
			"kueri %s tidak boleh menyaring pemilik", name)
	}
}

// Repo MENOLAK permintaan tanpa pemilik, bukan menjalankannya tanpa penyaring.
func TestPermintaanTanpaPemilikDitolak(t *testing.T) {
	_, err := (&Repo{}).List(nil, inboxoutstanding.Filter{}) //nolint:staticcheck // ctx tidak dipakai sebelum penolakan
	require.ErrorIs(t, err, inboxoutstanding.ErrAssigneeRequired)
}

// Tiga penyaring `BrowseInboxOutstanding1` TIDAK boleh kembali.
//
// Ketiganya berasal dari kueri layar LAIN — dashboard — dan `InboxRegister_RD` tidak
// memilikinya. Membawanya berarti menyaring lebih ketat daripada layar aslinya: klaim yang
// di Pega terlihat akan hilang di sini, tanpa galat yang menandainya.
func TestPenyaringDariKueriYangSalahTidakTerbawa(t *testing.T) {
	for _, name := range kueriModul {
		text := query(name)
		require.NotContains(t, text, "PXFLOWNAME", "kueri %s", name)
		require.NotContains(t, text, "FixCorrespondence", "kueri %s", name)
		require.NotContains(t, text, "ASNET", "kueri %s", name)
	}
}

func TestPolaPencarianDiseragamkanMenjadiHurufBesar(t *testing.T) {
	require.Equal(t, "%PNCN.26%", searchPattern("pncn.26"))
	require.Equal(t, "", searchPattern("   "))
}

// Karakter khusus LIKE di-escape supaya pencarian tidak berubah menjadi pola liar.
//
// Tanpa ini, mengetik "%" akan mencocokkan SELURUH baris — dan pengguna tidak punya cara
// menduga mengapa.
func TestKarakterKhususLIKEDiEscape(t *testing.T) {
	require.Equal(t, `%100\%%`, searchPattern("100%"))
	require.Equal(t, `%A\_B%`, searchPattern("a_b"))
	require.Equal(t, `%C\\D%`, searchPattern(`c\d`))
}

// Escaping hanya bekerja bila SQL menyatakan karakter escape-nya. Keduanya harus cocok:
// Go meng-escape dengan backslash, SQL wajib menyebut ESCAPE '\'.
func TestSQLMenyatakanKarakterEscapeYangSamaDenganGo(t *testing.T) {
	for _, name := range kueriSemua {
		text := query(name)
		require.Equal(t, strings.Count(text, "LIKE"), strings.Count(text, `ESCAPE '\'`),
			"kueri %s: setiap LIKE wajib menyebut ESCAPE '\\'", name)
	}
}

func TestPenyaringKosongDikirimSebagaiNULL(t *testing.T) {
	args := filterArgs(inboxoutstanding.Filter{AssignedTo: "BUDI"}.Normalize())

	// Empat belas argumen: DUA identitas + satu penanda pencarian + tiga pola + empat
	// penyaring opsional yang masing-masing dikirim dua kali.
	require.Len(t, args, 14)

	require.Equal(t, "BUDI", args[0], "identitas login TIDAK pernah NULL")

	// Identitas lama yang kosong DIULANGI dengan identitas sekarang, bukan dikirim NULL.
	// Hasil kueri sama, tetapi maksudnya terbaca tanpa menalar perilaku NULL pada IN.
	require.Equal(t, "BUDI", args[1], "identitas lama kosong diulangi, bukan NULL")

	for i, a := range args[2:] {
		require.Nil(t, a, "argumen ke-%d harus NULL saat penyaringnya kosong", i+3)
	}
}

// Identitas lama benar-benar sampai ke kueri.
//
// Tanpa uji ini, sebuah cacat yang membuang AssignedToLegacy tidak akan tertangkap: daftar
// tetap terisi bagi petugas yang identitasnya tidak pernah berganti — yaitu 9 dari 29
// operator pada data ASM — dan kosong bagi 20 sisanya tanpa satu pun galat.
func TestIdentitasLamaIkutDikirimKeKueri(t *testing.T) {
	args := filterArgs(inboxoutstanding.Filter{
		AssignedTo:       "orang@contoh.co.id",
		AssignedToLegacy: "NamaOperatorLama",
	}.Normalize())

	require.Equal(t, "ORANG@CONTOH.CO.ID", args[0])
	require.Equal(t, "NAMAOPERATORLAMA", args[1], "identitas lama diseragamkan huruf besar")
}

func TestPenyaringOpsionalDiseragamkanMenjadiHurufBesar(t *testing.T) {
	args := filterArgs(inboxoutstanding.Filter{
		AssignedTo: "budi",
		GroupPanel: "002",
		RCVID:      "rcv-1",
		Stage:      "komite",
		BranchCode: "jkt",
	}.Normalize())

	require.Len(t, args, 14)

	require.Equal(t, "BUDI", args[0])
	// Tiap penyaring opsional dikirim DUA KALI karena penandanya muncul dua kali di dalam
	// SQL. Lihat kepala outstanding.sql.
	require.Equal(t, "002", args[6])
	require.Equal(t, "002", args[7])
	require.Equal(t, "RCV-1", args[8])
	require.Equal(t, "RCV-1", args[9])
	require.Equal(t, "KOMITE", args[10])
	require.Equal(t, "KOMITE", args[11])
	require.Equal(t, "JKT", args[12])
	require.Equal(t, "JKT", args[13])
}

// Tiap penanda bernomor muncul TEPAT SEKALI di dalam satu kueri.
//
// Ini uji yang menjaga ORA-01008 tidak kembali. Kueri ini sempat memakai
// `:5 IS NULL OR … = :5`, dan Oracle menolaknya: driver mengikat argumen menurut urutan
// KEMUNCULAN penanda, bukan menurut nomornya.
//
// Cacat itu lolos seluruh uji sampai kuerinya benar-benar menyentuh Oracle — tidak satu pun
// kueri lain di repo ini mengulang penanda, sehingga polanya tidak pernah teruji.
func TestPenandaBernomorTidakDiulangDalamSatuKueri(t *testing.T) {
	for _, name := range kueriSemua {
		seen := map[string]int{}
		for _, mark := range regexp.MustCompile(`:\d+`).FindAllString(bodyOnly(query(name)), -1) {
			seen[mark]++
		}
		for mark, count := range seen {
			require.Equalf(t, 1, count,
				"kueri %s memakai %s sebanyak %d kali; tiap kemunculan wajib bernomor sendiri",
				name, mark, count)
		}
	}
}

// bodyOnly membuang baris komentar supaya nomor yang disebut di dalam penjelasan tidak
// ikut terhitung sebagai penanda.
func bodyOnly(statement string) string {
	var kept []string
	for _, line := range strings.Split(statement, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// Kedua kueri wajib punya syarat WHERE yang sama persis selain paginasi.
//
// Bila keduanya menyimpang, pengguna melihat jumlah yang berbeda dari yang dapat
// ditelusurinya — dan tidak ada galat yang muncul.
func TestSyaratSamaPadaKeduaKueri(t *testing.T) {
	list, count := query("my_inbox_list"), query("my_inbox_count")

	// Kelimanya disalin dari `Report Definition/InboxRegister_RD-RD.xml`.
	for _, condition := range []string{
		"k.PXOBJCLASS = 'ASM-FW-GCNMFW-Work-PNC'",
		// DUA identitas, bukan satu — lihat Filter.AssignedToLegacy.
		"UPPER(TRIM(k.PXASSIGNEDOPERATORID)) IN (:1, :2)",
		"k.PYSTATUSWORK NOT IN ('Resolved-Completed', 'Resolved-Rejected')",
		"UPPER(TRIM(k.GROUPPANEL_1)) = :8",
		"UPPER(TRIM(k.PNCCASEID)) = :10",
	} {
		require.Contains(t, list, condition, "my_inbox_list")
		require.Contains(t, count, condition, "my_inbox_count")
	}
}

// Kedua kueri membaca tabel yang benar.
//
// Modul ini sempat dibangun di atas `CPNC_KLAIM` — tabel rancangan yang belum pernah
// dibuat dan modul penulisnya belum dipasang, sehingga layar akan selalu kosong TANPA
// GALAT. Uji ini menjaga kekeliruan itu tidak kembali diam-diam.
func TestKeduaKueriMembacaTabelYangBenar(t *testing.T) {
	for _, name := range kueriModul {
		text := query(name)
		require.Contains(t, text, "POOLDATA.T_CLAIMLIST_ADMIN", "kueri %s", name)
		require.NotContains(t, text, "CPNC_KLAIM", "kueri %s", name)
		require.NotContains(t, text, "CPNC_TUGAS", "kueri %s", name)
	}
}

// Tabel datar berarti TANPA JOIN — dan tanpa join, persoalan INNER versus LEFT gugur.
//
// Di sistem lama kueri ini menempuh empat tabel, dan inner join-nya membuat klaim tanpa
// assignment hilang serta klaim dengan dua assignment muncul dua kali.
func TestKueriTidakMemakaiJoinSamaSekali(t *testing.T) {
	for _, name := range kueriModul {
		text := strings.ToUpper(query(name))
		require.NotContains(t, text, " JOIN ", "kueri %s", name)
		require.Equal(t, 1, strings.Count(text, "FROM "), "kueri %s membaca satu tabel", name)
	}
}

// SELECT * dilarang — `08-TECHNICAL-STRATEGY.md` §4.3.
func TestTidakAdaSelectBintang(t *testing.T) {
	for _, name := range kueriSemua {
		require.NotContains(t, query(name), "SELECT *", "kueri %s", name)
	}
}

// Pola Oracle-khas yang dilarang demi portabilitas ke PostgreSQL (`D-20`).
func TestTidakAdaPolaSQLYangDilarang(t *testing.T) {
	forbidden := []string{"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "FROM DUAL", "(+)", "LISTAGG"}

	for _, name := range kueriSemua {
		text := strings.ToUpper(query(name))
		for _, pattern := range forbidden {
			require.NotContains(t, text, pattern, "kueri %s memakai pola terlarang %s", name, pattern)
		}
	}
}
