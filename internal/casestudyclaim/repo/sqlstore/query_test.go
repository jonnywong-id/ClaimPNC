package sqlstore

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
)

// Uji di berkas ini berjalan TANPA basis data.
//
// Yang diperiksa adalah bentuk SQL dan pengikatan parameternya — dua hal yang bila salah
// menghasilkan kegagalan yang mahal: galat pengikatan di produksi, atau lebih buruk,
// penyaring ambang yang tidak berlaku tanpa satu pun galat, sehingga layar telaah klaim
// besar menampilkan SELURUH klaim.

// queryDaftar adalah kedua kueri baca yang wajib menyaring populasi yang sama.
var queryDaftar = []string{"case_study_list", "case_study_count"}

// querySemua mencakup pula pernyataan tulis dan pemeriksaan tabel.
var querySemua = append(append([]string{}, queryDaftar...),
	"case_study_save_remark", "case_study_check_table")

func TestSeluruhKueriTerbaca(t *testing.T) {
	for _, name := range querySemua {
		require.NotEmpty(t, query(name), "kueri %s kosong", name)
	}
}

func TestKomentarTidakIkutDikirimKeBasisData(t *testing.T) {
	for _, name := range querySemua {
		require.NotContains(t, query(name), "-- ", "komentar baris tidak boleh ikut terkirim")
	}
}

// Inilah uji yang menjaga janji terpenting layar ini: hanya klaim BESAR yang muncul.
//
// Tanpa penyaring ini, "Case Study Claim" menampilkan seluruh klaim pada rentang tahun yang
// dipilih — terisi, tampak wajar, dan salah tanpa satu pun galat.
func TestKeduaKueriMenyaringAmbangNilaiKlaim(t *testing.T) {
	for _, name := range queryDaftar {
		text := query(name)
		require.Contains(t, text, "POOLDATA.T_CLAIM_ADJUSTMENT adj",
			"kueri %s kehilangan tabel settlement", name)
		require.Contains(t, text, "adj.TOTAL_CLAIM * adj.CURRENCYVALUE >",
			"kueri %s kehilangan penyaring ambang", name)
	}
}

// Ambang diuji terhadap SATU baris settlement, bukan terhadap jumlahnya.
//
// `EXISTS (… WHERE total_claim*currencyvalue > …)` benar bila ADA SATU baris yang
// melampauinya. Menggantinya dengan `SUM(…) > …` akan memasukkan klaim yang totalnya besar
// tetapi terpecah menjadi beberapa baris kecil — populasi yang BERBEDA dari layar lama.
func TestAmbangDiujiPerBarisSettlementBukanJumlahnya(t *testing.T) {
	for _, name := range queryDaftar {
		text := query(name)
		require.Contains(t, text, "AND EXISTS ( SELECT 1", "kueri %s", name)

		// Bagian EXISTS tidak boleh memakai SUM. Diperiksa pada potongan di sekitar
		// penyaring ambang, bukan di seluruh kueri — kueri daftar memang penuh SUM pada
		// kolom nilainya.
		index := strings.Index(text, "AND EXISTS ( SELECT 1")
		require.GreaterOrEqual(t, index, 0)
		require.NotContains(t, text[index:], "SUM(",
			"kueri %s: ambang tidak boleh diuji terhadap jumlah", name)
	}
}

// Rentang tahun ikut menyaring pada KEDUA kueri.
//
// Bila hanya salah satunya, bilah halaman menyebut jumlah yang berbeda dari yang dapat
// ditelusuri pengguna — dan tidak ada galat yang muncul.
func TestKeduaKueriMenyaringRentangTahun(t *testing.T) {
	for _, name := range queryDaftar {
		require.Contains(t, query(name), "a.THNREGIS BETWEEN :1 AND :2", "kueri %s", name)
	}
}

// Keempat cakupan bisnis terbaca apa adanya dari kueri lama.
//
// Kode bisnisnya di-hardcode di Pega dan di sini pun — keduanya menunggu master `F-4`
// (`D-15`). Uji ini yang menjaga daftarnya tidak berubah diam-diam sebelum master itu ada.
func TestCakupanBisnisSesuaiKueriLama(t *testing.T) {
	for _, name := range queryDaftar {
		text := query(name)

		// NONMBU: empat Group Panel, dikurangi lima kode bisnis.
		require.Contains(t, text, "b.GROUPPANEL IN ('003', '004', '006', '009')", "kueri %s", name)
		require.Contains(t, text,
			"b.BUSINESSCODE NOT IN ('10145', '10168', '10165', '10164', '10053')", "kueri %s", name)

		// PA dan TRAVEL: satu Group Panel masing-masing.
		require.Contains(t, text, "( :5 = '002' AND b.GROUPPANEL = '002' )", "kueri %s", name)
		require.Contains(t, text, "( :6 = '005' AND b.GROUPPANEL = '005' )", "kueri %s", name)

		// BONDING: Group Panel yang SAMA dengan sebagian NONMBU, dibedakan hanya oleh
		// kesepuluh kode bisnisnya.
		require.Contains(t, text, "'10076', '10077', '10007', '10011', '10083'", "kueri %s", name)
		require.Contains(t, text, "'10141', '10131', '10126', '10055', '10075'", "kueri %s", name)
	}
}

// Penyaring status memakai `<> '3'`, bukan `IN ('0','1')`.
//
// Perbedaannya nyata: nilai asing apa pun ikut lolos ke "CLAIM ON PROGRESS/ ACCEPT", persis
// seperti di Pega. Merapikannya menjadi daftar tertutup akan menghilangkan baris yang di
// layar lama terlihat.
func TestPenyaringStatusMeniruOperatorKueriLama(t *testing.T) {
	for _, name := range queryDaftar {
		text := query(name)
		require.Contains(t, text, "AND a.STSKLAIM <> '3'", "kueri %s", name)
		require.Contains(t, text, "AND a.STSKLAIM =  '3'", "kueri %s", name)
		require.NotContains(t, text, "a.STSKLAIM IN (", "kueri %s", name)
	}
}

// Ketiga tabel dibaca, dan join-nya INNER — sama seperti koma pada kueri lama.
//
// Mengubahnya menjadi LEFT JOIN akan MENAMBAH baris yang di layar lama tidak pernah muncul:
// klaim yang kode bisnisnya tidak punya baris di `POOLDATA.BUSINESS`.
func TestKeduaKueriMembacaTigaTabelDenganInnerJoin(t *testing.T) {
	for _, name := range queryDaftar {
		text := query(name)
		require.Contains(t, text, "POOLDATA.PEGA_DASHBOARDPNC a", "kueri %s", name)
		require.Contains(t, text, "INNER JOIN POOLDATA.T_CLAIM_PNC b", "kueri %s", name)
		require.Contains(t, text, "INNER JOIN POOLDATA.BUSINESS d", "kueri %s", name)
		require.NotContains(t, text, "LEFT JOIN", "kueri %s", name)
	}
}

// Paginasi menuntut urutan pasti; Pega tidak punya ORDER BY sama sekali.
//
// Tanpa ini, dua halaman berturut-turut dapat memuat baris yang sama dan melewatkan yang
// lain — dan tidak ada apa pun di layar yang menandakannya.
func TestKueriDaftarPunyaUrutanPasti(t *testing.T) {
	text := query("case_study_list")
	require.Contains(t, text, "ORDER BY a.NOKLAIM, b.CLAIMID")
	require.Contains(t, text, "OFFSET :12 ROWS FETCH NEXT :13 ROWS ONLY")

	// Lapisan LUAR ikut mengurutkan. Tanpa itu, Oracle tidak menjamin urutan hasil akhir
	// meski lapisan dalamnya sudah terurut.
	require.Contains(t, text, "ORDER BY base.CLAIM_NUMBER, base.CLAIM_KEY")
}

// Nilai uang dikembalikan dalam SATUAN TERKECIL, sebagai bilangan bulat.
//
// Inilah yang membuat pembacaannya tidak dapat kehilangan ketepatan — lihat readMinorUnits.
// Menghapus `* 100` akan membuat nilai berdesimal sampai ke Go sebagai float64 dan ditolak,
// atau lebih buruk, dibulatkan diam-diam.
func TestNilaiUangDikembalikanDalamSatuanTerkecil(t *testing.T) {
	text := query("case_study_list")

	for _, column := range []string{
		"TSI_MINOR", "DEDUCTIBLE_MINOR", "ASM_SHARE_VALUE_MINOR", "CLAIM_VALUE_MINOR",
		"ADJUSTER_FEE_MINOR", "LACK_OF_DOC_MINOR", "NET_CLAIM_MINOR", "NET_CLAIM_ASM_MINOR",
	} {
		require.Contains(t, text, column, "kolom %s hilang", column)
	}

	// Persentase dikali 10.000, empat desimal (`D-51`).
	require.Contains(t, text, "ROUND(MAX(adj.ASM_SHARE) * 10000)")
}

// Rumus NET disalin dari kueri lama, termasuk tanda setiap sukunya.
//
// Satu tanda yang tertukar mengubah nilai klaim bersih tanpa membuat satu pun baris hilang
// — kelas cacat yang hanya terlihat bila angkanya dicocokkan satu per satu.
func TestRumusNilaiKlaimNetDisalinApaAdanya(t *testing.T) {
	text := query("case_study_list")

	require.Contains(t, text, "SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE)")
	require.Contains(t, text, "- SUM(adj.INDIVIDUAL_RISK_VALUE * adj.CURRENCYVALUE)")
	require.Contains(t, text, "- SUM((COALESCE(adj.LOC, 0) / 100) * adj.TOTAL_CLAIM * adj.CURRENCYVALUE)")
	require.Contains(t, text, "+ SUM(COALESCE(adj.NILAI_SALVAGE_A * adj.CURRENCYVALUE, 0))")
	require.Contains(t, text, "- SUM(adj.GROSSVALUE * adj.CURRENCYVALUE)")

	// NET ASM SHARE memakai MAX(ASM_SHARE/100) atas seluruh baris — BERBEDA dari kolom
	// "Nilai share ASM" yang memakai ASM_SHARE per baris di dalam SUM. Keduanya memang
	// berbeda di kueri lama.
	require.Contains(t, text, "MAX(adj.ASM_SHARE / 100)")
	require.Contains(t, text, "SUM(adj.TOTAL_CLAIM * adj.CURRENCYVALUE * (adj.ASM_SHARE / 100))")
}

// Pernyataan tulis menyentuh SATU kolom pada SATU tabel.
//
// Tabelnya milik Pega (`P-1`). Uji ini yang menjaga lingkup penulisannya tidak melebar
// tanpa keputusan — menambah satu kolom di sini berarti menambah satu kolom pada
// serah-terima kepemilikan tulis (`D-63`).
func TestPernyataanTulisHanyaMenyentuhSatuKolom(t *testing.T) {
	text := query("case_study_save_remark")

	require.Contains(t, text, "UPDATE POOLDATA.T_CLAIM_PNC")
	require.Contains(t, text, "SET REMARKRECOMENDATION = :1")
	require.Contains(t, text, "WHERE CLAIMNO = :2")

	require.Equal(t, 1, strings.Count(text, "SET "), "hanya satu klausa SET")
	require.Equal(t, 1, strings.Count(strings.ToUpper(text), "UPDATE "), "hanya satu UPDATE")
	require.NotContains(t, strings.ToUpper(text), "DELETE")
}

// Modul ini TIDAK boleh punya pernyataan tulis lain.
//
// Penulisan ke tabel milik Pega adalah pengecualian yang sudah dinegosiasikan untuk SATU
// kolom. Pernyataan tulis kedua yang masuk diam-diam akan melewati negosiasi itu.
func TestTidakAdaPernyataanTulisLain(t *testing.T) {
	for _, name := range querySemua {
		if name == "case_study_save_remark" {
			continue
		}
		text := strings.ToUpper(query(name))
		for _, statement := range []string{"UPDATE ", "INSERT ", "DELETE ", "MERGE "} {
			require.NotContains(t, text, statement,
				"kueri %s tidak boleh menulis (%s)", name, strings.TrimSpace(statement))
		}
	}
}

// Tiap penanda bernomor muncul TEPAT SEKALI di dalam satu kueri.
//
// Ini uji yang menjaga ORA-01008 tidak kembali: driver mengikat argumen menurut urutan
// KEMUNCULAN penanda, bukan menurut nomornya. Penanda yang dipakai ulang lolos seluruh uji
// sampai kuerinya benar-benar menyentuh Oracle.
func TestPenandaBernomorTidakDiulangDalamSatuKueri(t *testing.T) {
	for _, name := range querySemua {
		seen := map[string]int{}
		for _, mark := range regexp.MustCompile(`:\d+`).FindAllString(query(name), -1) {
			seen[mark]++
		}
		for mark, count := range seen {
			require.Equalf(t, 1, count,
				"kueri %s memakai %s sebanyak %d kali; tiap kemunculan wajib bernomor sendiri",
				name, mark, count)
		}
	}
}

// Jumlah argumen yang dikirim Go sama dengan jumlah penanda di dalam SQL.
//
// Selisih satu di sini menghasilkan galat pengikatan yang hanya muncul saat kueri benar-
// benar dijalankan terhadap Oracle — yaitu di produksi.
func TestJumlahArgumenSamaDenganJumlahPenanda(t *testing.T) {
	args := filterArgs(casestudyclaim.Filter{FromYear: "2024", ToYear: "2025"}.Normalize())
	require.Len(t, args, 11, "sebelas penyaring, sebelum offset dan limit")

	count := len(regexp.MustCompile(`:\d+`).FindAllString(query("case_study_count"), -1))
	require.Equal(t, len(args), count, "case_study_count")

	list := len(regexp.MustCompile(`:\d+`).FindAllString(query("case_study_list"), -1))
	require.Equal(t, len(args)+2, list, "case_study_list menambah offset dan limit")
}

// Penyaring yang tidak dipilih dikirim sebagai NULL, sehingga `:n IS NULL` membuatnya tidak
// mempersempit apa pun.
func TestPenyaringKosongDikirimSebagaiNULL(t *testing.T) {
	args := filterArgs(casestudyclaim.Filter{FromYear: "2024", ToYear: "2025"}.Normalize())

	require.Equal(t, "2024", args[0], "tahun awal TIDAK pernah NULL")
	require.Equal(t, "2025", args[1], "tahun akhir TIDAK pernah NULL")

	for i := 2; i <= 9; i++ {
		require.Nilf(t, args[i], "argumen ke-%d harus NULL saat penyaringnya kosong", i+1)
	}
}

// Penyaring yang dipilih dikirim BERKALI-KALI — sekali per kemunculan penandanya.
func TestPenyaringTerpilihDikirimSebanyakKemunculannya(t *testing.T) {
	args := filterArgs(casestudyclaim.Filter{
		FromYear: "2024",
		ToYear:   "2025",
		Business: casestudyclaim.ScopeNonMBU,
		Status:   casestudyclaim.StatusRejected,
	}.Normalize())

	// Lima kali untuk bisnis (:3 … :7).
	for i := 2; i <= 6; i++ {
		require.Equal(t, "346", args[i], "argumen ke-%d", i+1)
	}
	// Tiga kali untuk status (:8 … :10).
	for i := 7; i <= 9; i++ {
		require.Equal(t, "reject", args[i], "argumen ke-%d", i+1)
	}
}

// Ambang dikirim dalam RUPIAH, bukan sen.
//
// Kolom yang dibandingkan (`TOTAL_CLAIM * CURRENCYVALUE`) bersatuan rupiah, dan kueri lama
// membandingkannya terhadap `5000000000`. Mengirim sen akan menaikkan ambangnya seratus
// kali lipat — layar tetap terisi, hanya jauh lebih sedikit, tanpa satu pun galat.
func TestAmbangDikirimDalamRupiah(t *testing.T) {
	args := filterArgs(casestudyclaim.Filter{FromYear: "2024", ToYear: "2025"}.Normalize())
	require.Equal(t, int64(5_000_000_000), args[10])
}

// Rentang yang kosong DITOLAK sebelum kueri dijalankan.
func TestPermintaanTanpaPeriodeDitolak(t *testing.T) {
	_, err := (&Repo{}).List(nil, casestudyclaim.Filter{}) //nolint:staticcheck // ctx tidak dipakai sebelum penolakan
	require.ErrorIs(t, err, casestudyclaim.ErrPeriodRequired)
}

func TestPenyimpananTanpaNomorKlaimDitolak(t *testing.T) {
	_, err := (&Repo{}).SaveRemark(nil, "  ", "apa pun") //nolint:staticcheck // ctx tidak dipakai sebelum penolakan
	require.ErrorIs(t, err, casestudyclaim.ErrClaimRequired)
}

// SELECT * dilarang — `08-TECHNICAL-STRATEGY.md` §4.3.
func TestTidakAdaSelectBintang(t *testing.T) {
	for _, name := range querySemua {
		require.NotContains(t, query(name), "SELECT *", "kueri %s", name)
	}
}

// Pola Oracle-khas yang dilarang demi portabilitas ke PostgreSQL (`D-20`).
//
// `TO_CHAR` ikut dilarang di modul ini. Kueri lama memakainya untuk kolom "Bulan Klaim";
// penggantinya `EXTRACT`, yang portabel DAN membaca nilai tersimpan tanpa konversi zona
// waktu.
func TestTidakAdaPolaSQLYangDilarang(t *testing.T) {
	forbidden := []string{
		"NVL(", "ROWNUM", "SYSDATE", "DECODE(", "FROM DUAL", "(+)", "LISTAGG", "TO_CHAR",
	}

	for _, name := range querySemua {
		text := strings.ToUpper(query(name))
		for _, pattern := range forbidden {
			require.NotContains(t, text, pattern,
				"kueri %s memakai pola terlarang %s", name, pattern)
		}
	}
}

// Nomor bulan ditulis dua digit, sama persis dengan `to_char(…,'mm')` di Pega.
func TestNomorBulanDitulisDuaDigit(t *testing.T) {
	three := int64(3)
	require.Equal(t, "03", monthText(&three))

	twelve := int64(12)
	require.Equal(t, "12", monthText(&twelve))

	require.Equal(t, "", monthText(nil))

	// Nilai di luar rentang ditulis APA ADANYA, bukan dipaksa masuk rentang: kolom
	// sumbernya bertipe tanggal, sehingga nilai seperti ini berarti ada yang tidak
	// dipahami — dan itu harus terlihat.
	strange := int64(13)
	require.Equal(t, "13", monthText(&strange))
}

// Bentuk apa pun yang diserahkan driver untuk kolom NUMBER dibaca menjadi angka yang sama.
//
// `go-ora` menyerahkan NUMBER sebagai int64, float64, atau string bergantung presisinya,
// dan kolom hasil `ROUND(...)` tidak punya presisi yang dideklarasikan. Ketiganya harus
// menghasilkan nilai yang identik.
func TestNilaiUangDibacaDariBentukApaPunYangDiserahkanDriver(t *testing.T) {
	expected := money.FromRupiah(8_000_000_000)

	for name, raw := range map[string]any{
		"int64":   int64(800_000_000_000),
		"float64": float64(800_000_000_000),
		"string":  "800000000000",
		"bytes":   []byte(" 800000000000 "),
	} {
		amount, err := readMinorUnits(raw)
		require.NoErrorf(t, err, "bentuk %s", name)
		require.NotNilf(t, amount, "bentuk %s", name)
		require.Equalf(t, expected, *amount, "bentuk %s", name)
	}
}

// NULL menghasilkan nil, BUKAN nol.
//
// `SUM` atas himpunan kosong mengembalikan NULL, dan "belum ada nilainya" berbeda dari
// "nilainya nol" — terutama pada berkas yang dibuka di pengolah angka.
func TestNilaiUangKosongTetapKosong(t *testing.T) {
	amount, err := readMinorUnits(nil)
	require.NoError(t, err)
	require.Nil(t, amount)
}

// Nilai berdesimal yang lolos ke Go DITOLAK, bukan dibulatkan diam-diam.
//
// SQL sudah membulatkannya; bila yang sampai ke sini masih berdesimal, ada yang tidak
// sesuai dugaan — dan pembulatan diam adalah cara selisih rupiah masuk tanpa ada yang
// menyadarinya.
func TestNilaiBerdesimalDitolak(t *testing.T) {
	_, err := readMinorUnits(1234.56)
	require.Error(t, err)
	require.Contains(t, err.Error(), "berdesimal")
}

// Penolakan Oracle atas nilai yang melebihi lebar kolom dikenali dari NOMOR galatnya.
//
// # Kenapa uji ini ada
//
// `MaxRemarkLength` adalah DUGAAN: DDL `REMARKRECOMENDATION` belum pernah diterima
// (`R-08`), dan Work Owner memutuskan tidak mengejarnya — modul berjalan seperti Pega,
// yang juga tidak membatasi panjangnya.
//
// Pengenalan inilah yang membuat keputusan itu aman. Bila dugaannya terlalu longgar,
// pengguna tetap diberi tahu bahwa catatannya terlalu panjang; tanpanya ia menerima
// "terjadi kesalahan pada sistem" setelah mengetik satu halaman penuh.
func TestPenolakanLebarKolomDikenali(t *testing.T) {
	require.True(t, isValueTooLarge(
		errors.New("ORA-12899: value too large for column")))

	// Pesan berbahasa lain tetap dikenali: yang dicocokkan nomornya, bukan teksnya.
	require.True(t, isValueTooLarge(
		errors.New("ORA-12899: nilai terlalu besar untuk kolom")))
}

// Galat lain TIDAK boleh tersamar menjadi "catatan terlalu panjang".
//
// Menyamarkan kegagalan koneksi atau hak akses sebagai kesalahan isian akan menyuruh
// pengguna memperpendek catatannya berulang kali atas masalah yang tidak ada hubungannya.
func TestGalatLainTidakDikiraPenolakanLebarKolom(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("ORA-01017: invalid username/password"),
		errors.New("ORA-00942: table or view does not exist"),
		errors.New("dial tcp: connection refused"),
	} {
		require.False(t, isValueTooLarge(err), "%v", err)
	}
}
