package sqlstore

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// Uji kueri tab KPI PIC Teknik.
//
// Seluruhnya memeriksa TEKS kueri, bukan hasilnya terhadap basis data — dan itu memang yang
// dapat diperiksa tanpa Oracle. Yang diuji adalah hal-hal yang bila salah TIDAK menimbulkan
// galat: penyaring yang hilang, dialek yang tidak portabel, dan penyaring lini yang
// meloloskan semua baris.

// picQueryNames adalah ketujuh kueri tab ini.
var picQueryNames = []string{
	"pic_list", "bands", "threshold_days", "holidays",
	"progress_counts", "analysis_spans", "acceptance_spans", "closure_spans",
}

// lineScopedQueries adalah kueri yang WAJIB menyaring lini bisnis.
//
// Satu saja yang tidak menyaring akan menampilkan angka SELURUH lini kepada orang yang
// meminta satu lini — dan itu tidak terlihat sebagai galat, hanya sebagai angka yang lebih
// besar.
var lineScopedQueries = []string{
	"progress_counts", "analysis_spans", "acceptance_spans", "closure_spans",
}

func TestSeluruhKueriPICTeknikAda(t *testing.T) {
	for _, name := range picQueryNames {
		require.NotEmpty(t, query(name), name)
	}
}

// Keempat kueri berbasis periode WAJIB memuat penanda penyaring lini.
func TestKueriPICBerlingkupLiniMemuatPenanda(t *testing.T) {
	for _, name := range lineScopedQueries {
		require.Containsf(t, query(name), lineFilterMarker,
			"kueri %s tidak menyaring lini bisnis", name)
	}
}

// Penanda benar-benar TERGANTI, dan penyaringnya berbeda tiap lini.
func TestPenandaLiniTergantiDanBerbedaTiapLini(t *testing.T) {
	seen := map[string]bool{}

	for _, line := range reportkpi.BusinessLines() {
		text := picQuery("closure_spans", line.Code)
		require.NotContainsf(t, text, lineFilterMarker,
			"penanda masih tertinggal pada lini %s", line.Code)
		require.Falsef(t, seen[text], "lini %s memakai penyaring yang sama dengan lini lain",
			line.Code)
		seen[text] = true
	}

	require.Len(t, seen, 4)
}

// Lini yang TIDAK dikenal menolak semua baris, bukan meloloskan semuanya.
//
// Ini penjaga terakhir. Penyaring kosong akan menampilkan seluruh lini kepada peminta satu
// lini — kegagalan yang tidak terlihat. `AND 1 = 0` membuatnya terlihat seketika.
func TestLiniTidakDikenalMenolakSemuaBaris(t *testing.T) {
	text := picQuery("closure_spans", reportkpi.BusinessLine("MBU"))
	require.Contains(t, text, "AND 1 = 0")
}

// Penyaring Non-MBU dan Bonding memang BERBENTUK BEDA, bukan dua nilai pada kolom yang sama.
//
// Non-MBU menyaring GROUP_PANEL sekaligus MENGECUALIKAN empat kode kelompok bisnis; Bonding
// justru menyaring keempat kode itu dan tidak menyentuh GROUP_PANEL sama sekali. Itu sebab
// penyaringnya tidak dapat dijadikan parameter.
func TestPenyaringNonMBUDanBondingBerbentukBeda(t *testing.T) {
	nonMBU := lineFilter(reportkpi.LineNonMBU)
	bonding := lineFilter(reportkpi.LineBonding)

	require.Contains(t, nonMBU, "GROUP_PANEL IN ('003', '004', '006')")
	require.Contains(t, nonMBU, "GROUPBISNISID NOT IN ('09', '11', '16', '25')")

	require.Contains(t, bonding, "GROUPBISNISID IN ('09', '11', '16', '25')")
	require.NotContains(t, bonding, "GROUP_PANEL",
		"Bonding TIDAK menyentuh GROUP_PANEL — itu perilaku lama")
}

// Keempat kode kelompok bisnis Bonding sama persis dengan yang dikecualikan Non-MBU.
//
// Keduanya saling meniadakan di sistem lama. Satu kode yang berbeda membuat sebagian klaim
// tidak terhitung di lini mana pun — dan tidak ada yang akan menyadarinya.
func TestKodeBondingSamaDenganYangDikecualikanNonMBU(t *testing.T) {
	const codes = "('09', '11', '16', '25')"
	require.Contains(t, lineFilter(reportkpi.LineNonMBU), "NOT IN "+codes)
	require.Contains(t, lineFilter(reportkpi.LineBonding), "IN "+codes)
}

// Tidak ada satu pun pola SQL terlarang `D-20` di kueri tab ini.
func TestKueriPICTidakMemakaiPolaTerlarang(t *testing.T) {
	// TO_CHAR SENGAJA dikecualikan: ia dipakai pada kueri hari libur untuk membandingkan
	// NAMA HARI, bukan untuk memformat tampilan. Larangannya menyangkut pemformatan.
	forbidden := []string{"SELECT *", "ROWNUM", "NVL(", "SYSDATE", "DECODE("}

	for _, name := range picQueryNames {
		text := strings.ToUpper(query(name))
		for _, pattern := range forbidden {
			require.NotContainsf(t, text, pattern,
				"kueri %s memakai pola terlarang %s", name, pattern)
		}
	}
}

// Rentang tanggal SETENGAH TERBUKA, dan tidak ada TRUNC pada kolom.
//
// Kueri lama memakai `trunc(kolom) >= … and trunc(kolom) <= …`, yang mematikan index dan
// memaksa pemindaian tabel penuh. Penggantinya mencakup hari yang sama persis.
func TestRentangTanggalSetengahTerbukaTanpaTrunc(t *testing.T) {
	for _, name := range lineScopedQueries {
		text := strings.ToUpper(query(name))
		require.Containsf(t, text, "INTERVAL '1' DAY",
			"kueri %s tidak memakai rentang setengah terbuka", name)
		require.NotContainsf(t, text, "TRUNC(",
			"kueri %s masih memakai TRUNC pada kolom", name)
	}
}

// Kueri pita menyaring TIPE='PIC' dan membuang NILAI kosong.
//
// Keduanya penajaman terhadap kueri lama, dan keduanya menutup kegagalan senyap: baris
// ADJUSTER bernama JOB yang sama akan mengubah nilai seseorang, dan baris ber-NILAI kosong
// akan terbaca sebagai nol.
func TestKueriPitaMenyaringTipeDanNilaiKosong(t *testing.T) {
	text := query("bands")
	require.Contains(t, text, "TIPE = 'PIC'")
	require.Contains(t, text, "NILAI IS NOT NULL")
	require.Contains(t, text, "ORDER BY ID",
		"urutan tetap membuat titik batas pita dapat ditentukan")
}

// Kueri hari libur mengecualikan akhir pekan DI SISI KUERI, seperti CheckHoliday_SQL.
//
// Bila dikecualikan di Go saja, libur yang jatuh pada Sabtu atau Minggu akan terpotong dua
// kali — sekali sebagai akhir pekan, sekali sebagai libur.
func TestKueriHariLiburMengecualikanAkhirPekan(t *testing.T) {
	text := strings.ToUpper(query("holidays"))
	for _, day := range []string{"SABTU", "MINGGU", "SATURDAY", "SUNDAY"} {
		require.Containsf(t, text, day, "nama hari %s tidak dikecualikan", day)
	}
}

// Ketiga penyaring pembersih pada kueri progres dipertahankan apa adanya.
//
// Ketiganya membuang pembaruan yang dibuat SISTEM, bukan orang — dan menilai orang atas
// pembaruan yang dibuat sistem jelas keliru.
func TestPenyaringPembersihProgresDipertahankan(t *testing.T) {
	text := query("progress_counts")
	require.Contains(t, text, "'%AUTO%'")
	require.Contains(t, text, "'%SYSTEM%'")
	require.Contains(t, text, "a.PIC <> 'ASNET'")
	require.Contains(t, text, "a.STSKLAIM NOT IN ('1', '2', '3')")
}

// Kueri Analisa hanya menilai adjustment PERTAMA tiap klaim.
func TestKueriAnalisaHanyaAdjustmentPertama(t *testing.T) {
	text := query("analysis_spans")
	require.Contains(t, text, "MIN(x.ADJUSTMENTID)")
	require.Contains(t, text, "ANALYST_TFKOMITEDATE IS NOT NULL")
}

// Kueri Akseptasi mengembalikan KETIGA tanggal, dan tidak memilih di dalam SQL.
//
// Pemilihannya aturan bisnis — ia hidup di domain, bukan di dalam kueri.
func TestKueriAkseptasiMengembalikanKetigaTanggal(t *testing.T) {
	text := query("acceptance_spans")
	require.Contains(t, text, "b.RECEIVEDATELOD")
	require.Contains(t, text, "b.ACCEPTANCE_DATECOMITEE")
	require.Contains(t, text, "b.TGLAKSEPTASI")
	require.Contains(t, text, "a.LEADER_MEMBER")
	require.NotContains(t, strings.ToUpper(text), "CASE WHEN",
		"pemilihan pasangan tanggal TIDAK dikerjakan di SQL")
}
