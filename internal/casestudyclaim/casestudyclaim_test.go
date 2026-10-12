package casestudyclaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/casestudyclaim"
)

// Ambangnya adalah DEFINISI layar ini, bukan sekadar angka.
//
// Uji ini ada supaya perubahan pada angkanya menjadi keputusan sadar: mengubahnya mengubah
// klaim mana yang dianggap layak ditelaah, dan tidak ada apa pun di layar yang menandainya.
func TestAmbangNilaiKlaimLimaMiliar(t *testing.T) {
	// Rp 5.000.000.000 = 500.000.000.000 sen.
	require.Equal(t, int64(500_000_000_000), casestudyclaim.LargeClaimThreshold.MinorUnits())
}

// Pemetaan `A.REINSURER`, disalin dari `CASE` pada RDB List/BrowseClaimStudy-SQL.xml.
func TestPosisiReasuransiDiturunkanDariKode(t *testing.T) {
	require.Equal(t, "Leader", casestudyclaim.ReinsurerRoleOf("1"))
	require.Equal(t, "Member", casestudyclaim.ReinsurerRoleOf("2"))
	require.Equal(t, "Fac In", casestudyclaim.ReinsurerRoleOf("F"))

	// Spasi di sekeliling nilai basis data tidak boleh mengubah hasilnya. Kolomnya
	// VARCHAR2 tanpa penyeragaman, dan satu spasi menggantung adalah keadaan biasa.
	require.Equal(t, "Leader", casestudyclaim.ReinsurerRoleOf(" 1 "))

	// Kode asing menjadi tanda hubung, persis seperti cabang `else '-'` di kueri lama —
	// BUKAN ditampilkan apa adanya. Menampilkannya akan menjadi selisih pada uji
	// kesetaraan tanpa menambah keterangan apa pun.
	require.Equal(t, "-", casestudyclaim.ReinsurerRoleOf("9"))
	require.Equal(t, "-", casestudyclaim.ReinsurerRoleOf(""))
}

// Pemetaan `A.STSKLAIM`, termasuk campuran bahasanya.
//
// "Claim On Progress" dan "Accepted" berbahasa Inggris, "Ditolak" berbahasa Indonesia.
// Campuran itu ADA di kueri Pega dan dipertahankan (`D-13`); uji ini yang menjaganya tidak
// "dirapikan" oleh pembaca berikutnya.
func TestStatusKlaimDiturunkanDariKode(t *testing.T) {
	require.Equal(t, "Claim On Progress", casestudyclaim.ClaimStatusOf("0"))
	require.Equal(t, "Accepted", casestudyclaim.ClaimStatusOf("1"))
	require.Equal(t, "Ditolak", casestudyclaim.ClaimStatusOf("3"))
	require.Equal(t, "-", casestudyclaim.ClaimStatusOf("2"))
	require.Equal(t, "-", casestudyclaim.ClaimStatusOf(""))
}

// Grid Pega punya 24 kolom, dan urutannya menentukan pula urutan kolom berkas CSV.
func TestKolomBerjumlah24DanBerurutanSepertiPega(t *testing.T) {
	columns := casestudyclaim.Columns()
	require.Len(t, columns, 24)

	require.Equal(t, "PNC Case ID", columns[0].Title)
	require.Equal(t, "nomor_klaim", columns[0].Key)
	require.Equal(t, "Remark", columns[23].Title)
	require.Equal(t, casestudyclaim.KindRemark, columns[23].Kind)

	titles := casestudyclaim.ColumnTitles()
	require.Len(t, titles, 24)
	require.Equal(t, "LACK OF DOC/SALVAGE / RECOVERY / SUBROGARATION", titles[19])
}

// Dua kolom bersebelahan yang memang KEMBAR.
//
// Keduanya di Pega terikat properti `.Country` yang sama, dan kuerinya hanya menyediakan
// satu nilai. Uji ini menjaga keduanya tetap ada — menghapus salah satunya akan membuat
// grid baru punya 23 kolom sementara grid lama punya 24.
func TestDuaKolomPenyebabKerugianTetapKembar(t *testing.T) {
	columns := casestudyclaim.Columns()
	require.Equal(t, "Nature of Loss", columns[9].Title)
	require.Equal(t, "Cause of Loss", columns[10].Title)
}

// Columns mengembalikan SALINAN, bukan irisan aslinya.
//
// Pemanggil yang menyortir hasilnya akan menata ulang grid setiap layar sekaligus, dan
// kegagalan seperti itu tidak menghasilkan galat apa pun.
func TestColumnsMengembalikanSalinan(t *testing.T) {
	first := casestudyclaim.Columns()
	first[0].Title = "diubah"

	require.Equal(t, "PNC Case ID", casestudyclaim.Columns()[0].Title)
}

// Isi dropdown Bisnis dibaca dari `Property/StatusReceiver_property.xml`.
//
// Keempat labelnya sama persis dengan nilai `TYPE_BUSINESS` pada master ambang komite.
// Bila salah satunya berubah, penyaringnya masih bekerja tetapi memilih lini yang berbeda
// dari yang dikira pengguna — sehingga kode dan labelnya diperiksa berpasangan.
func TestPilihanBisnisSesuaiPropertyPega(t *testing.T) {
	require.Equal(t, []casestudyclaim.Option{
		{Code: "002", Label: "PA"},
		{Code: "005", Label: "TRAVEL"},
		{Code: "346", Label: "NONMBU"},
		{Code: "003", Label: "BONDING"},
	}, casestudyclaim.BusinessOptions())
}

func TestPilihanBisnisDicariDariKodenya(t *testing.T) {
	scope, known := casestudyclaim.FindBusinessScope("346")
	require.True(t, known)
	require.Equal(t, casestudyclaim.ScopeNonMBU, scope)

	// Kosong berarti "semua", dan itu SAH — di Pega ia `----- Pilih -----`, yang menyetel
	// potongan SQL-nya menjadi teks kosong.
	scope, known = casestudyclaim.FindBusinessScope("")
	require.True(t, known)
	require.Equal(t, casestudyclaim.ScopeAll, scope)

	// Kode asing DITOLAK, bukan diam-diam diartikan "semua". Pada layar berisi klaim di
	// atas Rp 5 miliar, penyaring yang diam-diam melebar bukan selisih yang pantas diam.
	_, known = casestudyclaim.FindBusinessScope("999")
	require.False(t, known)
}

// Isi dropdown Status dibaca dari `Activity/FilterStudyClaim_act-Act.xml`.
//
// Perhatikan SPASI setelah garis miring pada label pertama. Ia ada di export, dan `D-13`
// menetapkan teks layar mengikuti Pega apa adanya — termasuk ketika Pega tidak rapi.
func TestPilihanStatusSesuaiActivityPega(t *testing.T) {
	require.Equal(t, []casestudyclaim.Option{
		{Code: "progress-accept", Label: "CLAIM ON PROGRESS/ ACCEPT"},
		{Code: "reject", Label: "REJECT"},
	}, casestudyclaim.StatusOptions())
}

func TestPilihanStatusDicariDariKodenya(t *testing.T) {
	status, known := casestudyclaim.FindClaimStatus("reject")
	require.True(t, known)
	require.Equal(t, casestudyclaim.StatusRejected, status)

	status, known = casestudyclaim.FindClaimStatus("")
	require.True(t, known)
	require.Equal(t, casestudyclaim.StatusAny, status)

	_, known = casestudyclaim.FindClaimStatus("selesai")
	require.False(t, known)
}

// Ukuran halaman mengikuti `pyPageSize` pada section Pega, bukan angka yang dipilih di sini.
func TestUkuranHalamanMengikutiSectionPega(t *testing.T) {
	require.Equal(t, 20, casestudyclaim.DefaultLimit)
}

func TestNormalizeMemberiNilaiBakuYangMasukAkal(t *testing.T) {
	f := casestudyclaim.Filter{FromYear: " 2024 ", ToYear: " 2025 ", Offset: -5}.Normalize()

	require.Equal(t, "2024", f.FromYear)
	require.Equal(t, "2025", f.ToYear)
	require.Equal(t, casestudyclaim.DefaultLimit, f.Limit)
	require.Equal(t, 0, f.Offset)
}

// Rentang WAJIB terisi, dan penolakannya BUKAN perubahan data.
//
// Di Pega, isian yang kosong menghasilkan `BETWEEN NULL AND NULL` — nol baris tanpa satu
// pun pesan. Kedua sistem menampilkan hasil yang sama; yang berbeda hanyalah pengguna
// diberi tahu sebabnya.
func TestPeriodeKosongDitolak(t *testing.T) {
	require.ErrorIs(t, casestudyclaim.ValidatePeriod("", "2025"), casestudyclaim.ErrPeriodRequired)
	require.ErrorIs(t, casestudyclaim.ValidatePeriod("2024", " "), casestudyclaim.ErrPeriodRequired)
}

func TestPeriodeTerbalikDitolak(t *testing.T) {
	require.ErrorIs(t, casestudyclaim.ValidatePeriod("2026", "2024"), casestudyclaim.ErrPeriodReversed)
}

func TestPeriodeSatuTahunDiterima(t *testing.T) {
	require.NoError(t, casestudyclaim.ValidatePeriod("2024", "2024"))
}

// Tahun diambil pada zona waktu yang DISERAHKAN, bukan dari jam mesin.
//
// Inilah pagar `R-12` pada modul ini: 1 Januari pukul 00.00 WIB adalah 31 Desember pukul
// 17.00 UTC, dan tanpa zona yang tepat penyaringnya akan memilih tahun sebelumnya.
func TestTahunDiambilPadaZonaWaktuYangDiserahkan(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)

	awalTahun := time.Date(2025, 1, 1, 0, 0, 0, 0, wib)
	require.Equal(t, "2025", casestudyclaim.YearOf(awalTahun, wib))

	// Saat yang SAMA, dibaca sebagai UTC, jatuh ke tahun sebelumnya. Itu bukan kesalahan
	// fungsi ini — itulah yang hendak dicegah dengan menyerahkan zonanya.
	require.Equal(t, "2024", casestudyclaim.YearOf(awalTahun, time.UTC))
}

func TestCatatanTerlaluPanjangDitolak(t *testing.T) {
	long := make([]rune, casestudyclaim.MaxRemarkLength+1)
	for i := range long {
		long[i] = 'a'
	}

	require.ErrorIs(t,
		casestudyclaim.ValidateRemark("STD-0001", string(long)),
		casestudyclaim.ErrRemarkTooLong)
}

// Catatan KOSONG diterima, dan itu disengaja: mengosongkan catatan yang salah ketik adalah
// hal yang wajar, dan kolomnya nullable.
func TestCatatanKosongDiterima(t *testing.T) {
	require.NoError(t, casestudyclaim.ValidateRemark("STD-0001", ""))
}

func TestCatatanTanpaNomorKlaimDitolak(t *testing.T) {
	require.ErrorIs(t, casestudyclaim.ValidateRemark("  ", "apa pun"), casestudyclaim.ErrClaimRequired)
}

// Batas panjang dihitung per KARAKTER, bukan per byte.
//
// Kronologi dan catatan telaah ditulis dalam bahasa Indonesia dan kerap memuat tanda kutip
// tipografis atau simbol mata uang. Menghitungnya per byte akan menolak catatan yang
// sebenarnya masih muat.
func TestBatasCatatanDihitungPerKarakter(t *testing.T) {
	// Setiap "é" dua byte, sehingga 2.000 karakter berarti 4.000 byte. Ia tetap diterima:
	// yang dibatasi adalah jumlah karakternya, bukan jumlah byte-nya.
	text := make([]rune, casestudyclaim.MaxRemarkLength)
	for i := range text {
		text[i] = 'é'
	}
	require.NoError(t, casestudyclaim.ValidateRemark("STD-0001", string(text)))
}
