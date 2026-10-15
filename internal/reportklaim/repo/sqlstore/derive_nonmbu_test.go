package sqlstore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportklaim"
)

func TestBulanDanTahunDuaEmpatDigit(t *testing.T) {
	d := bacaTanggalCSV("07/03/2026")

	require.Equal(t, "03", bulanDuaDigit(d), "nol di depan ikut, seperti TO_CHAR(...,'mm')")
	require.Equal(t, "2026", tahunEmpatDigit(d))
	require.Equal(t, "07/03/26", tanggalDuaDigitTahun(d))
}

// Tanggal kosong menghasilkan sel kosong pada ketiganya — bukan "00", bukan "0001".
func TestBulanTahunTanggalKosongTetapKosong(t *testing.T) {
	require.Empty(t, bulanDuaDigit(bacaTanggalCSV("")))
	require.Empty(t, tahunEmpatDigit(bacaTanggalCSV("")))
	require.Empty(t, tanggalDuaDigitTahun(bacaTanggalCSV("")))
}

// Kolom TAT pada susunan rinci adalah selisih HARI KALENDER, bukan hari kerja. Akhir pekan
// ikut terhitung, dan itu memang yang dilakukan sumbernya.
func TestSelisihHariKalenderTidakMemotongAkhirPekan(t *testing.T) {
	// Jumat 2026-09-04 sampai Senin 2026-09-07 = 3 hari kalender, 1 hari kerja.
	require.Equal(t, "3", selisihHariKalender(
		bacaTanggalCSV("04/09/2026"), bacaTanggalCSV("07/09/2026")))
}

func TestSelisihHariKalenderTanpaTanggalKosong(t *testing.T) {
	require.Empty(t, selisihHariKalender(bacaTanggalCSV(""), bacaTanggalCSV("07/09/2026")))
	require.Empty(t, selisihHariKalender(bacaTanggalCSV("07/09/2026"), bacaTanggalCSV("")))
}

// Hari yang sama menghasilkan 0, bukan sel kosong — nol berarti "tidak ada jeda".
func TestSelisihHariKalenderHariYangSamaNol(t *testing.T) {
	require.Equal(t, "0", selisihHariKalender(
		bacaTanggalCSV("07/09/2026"), bacaTanggalCSV("07/09/2026")))
}

func tanggaFeeContoh() *reportklaim.FeeScale {
	return reportklaim.NewFeeScale([]reportklaim.FeeBand{
		{Index: 1, LossAmount: 100, Fee: 10},
		{Index: 2, LossAmount: 200, Fee: 30},
	})
}

// Rumus fee: baris bertipe Adjuster Fee dipakai apa adanya, baris lain memakai hasil
// interpolasi — dan hasil interpolasi itu terjumlah SEKALI PER BARIS bukan-'4'.
func TestFeeAdjusterMenjumlahInterpolasiSekaliPerBaris(t *testing.T) {
	row := reportklaim.Row{
		"FeeJumlahBaris":            "3",
		"FeeJumlahBarisInterpolasi": "2",
		"FeeLangsung":               "500",
		"FeeDasarTotalClaim":        "150", // → interpolasi 20
	}

	require.Equal(t, "540", feeAdjuster(row, tanggaFeeContoh()), "500 + 2 × 20")
}

// Tanpa baris bukan-'4', tangga fee tidak disentuh sama sekali.
func TestFeeAdjusterTanpaBarisInterpolasiTidakMemakaiTangga(t *testing.T) {
	row := reportklaim.Row{
		"FeeJumlahBaris":            "1",
		"FeeJumlahBarisInterpolasi": "0",
		"FeeLangsung":               "500",
		"FeeDasarTotalClaim":        "",
	}

	require.Equal(t, "500", feeAdjuster(row, reportklaim.UnavailableFeeScale()))
}

// Tidak ada satu pun baris settlement → sel KOSONG, bukan "0".
//
// Sumbernya menghasilkan NULL pada keadaan itu, dan nol pada kolom fee terbaca sebagai
// "adjuster tidak dibayar" alih-alih "tidak ada datanya".
func TestFeeAdjusterTanpaBarisSamaSekaliMenghasilkanSelKosong(t *testing.T) {
	row := reportklaim.Row{"FeeJumlahBaris": "0"}
	require.Empty(t, feeAdjuster(row, tanggaFeeContoh()))

	require.Empty(t, feeAdjuster(reportklaim.Row{}, tanggaFeeContoh()))
}

// Tangga yang tidak terbaca sementara ADA baris yang membutuhkannya → sel kosong.
// Menuliskan hanya bagian bertipe '4' akan menghasilkan angka yang lebih kecil dari
// seharusnya tanpa satu pun tanda.
func TestFeeAdjusterTanpaTanggaTidakMenuliskanSebagian(t *testing.T) {
	row := reportklaim.Row{
		"FeeJumlahBaris":            "2",
		"FeeJumlahBarisInterpolasi": "1",
		"FeeLangsung":               "500",
		"FeeDasarTotalClaim":        "150",
	}

	require.Empty(t, feeAdjuster(row, reportklaim.UnavailableFeeScale()))
}

// closeDerive pada lini SELAIN Non-MBU hanya mengisi singkatan bulan — bukan sebelas kolom
// susunan rinci. Mencampurkannya akan menambah kolom yang tidak ada di berkas ringkas.
func TestCloseDeriveRingkasHanyaSingkatanBulan(t *testing.T) {
	hitung, err := closeDerive(t.Context(), NewRepo(nil, nil),
		reportklaim.Filter{BusinessLine: reportklaim.BusinessLinePA})
	require.NoError(t, err)

	row := reportklaim.Row{"BulanCloseTanggal": "07/09/2026"}
	hitung(row)

	require.Equal(t, "SEP", row.Value("FlagASO"))
	require.NotContains(t, row, "ReporterName", "kolom susunan rinci tidak boleh muncul")
	require.NotContains(t, row, "AlasanTerlambat")
}

// Dan pada Non-MBU ia mengisi susunan rinci, bukan singkatan bulan.
func TestCloseDeriveRinciMengisiKolomTurunannya(t *testing.T) {
	hitung, err := closeDerive(t.Context(), NewRepo(nil, nil),
		reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU})
	require.NoError(t, err)

	row := reportklaim.Row{
		"StatusWork":            "07/03/2026", // tanggal kejadian
		"UserTeknisGroup":       "01/01/2025", // tanggal mulai polis
		"DollarCurrencyVal":     "10/04/2026", // tanggal akseptasi
		"UserTeknisEmail":       "20/04/2026", // tanggal proses
		"UserTeknis":            "05/05/2026", // tanggal reject
		"ReportDescription":     "1234",
		"TanggalRegistrasiTeks": "20260307",
	}
	hitung(row)

	require.Equal(t, "03", row.Value("ClaimNo"), "bulan DOL")
	require.Equal(t, "2026", row.Value("CloseClaimNote"), "tahun DOL")
	require.Equal(t, "2025", row.Value("RefNo"), "tahun mulai polis")
	require.Equal(t, "04", row.Value("CompliancePosAuditByr"))
	require.Equal(t, "2026", row.Value("ComplianceRemark"))
	require.Equal(t, "05", row.Value("Conveyance"))
	require.Equal(t, "2026", row.Value("Country"))
	require.Equal(t, "10", row.Value("UserName"), "TAT: 10/04 → 20/04")
	require.Equal(t, "07/03/2026", row.Value("NewNoKTP"))

	// Empat kolom "… OR" berisi nilai yang SAMA, seperti di sumbernya.
	for _, name := range append([]string{"ReportDescription"}, closeNonMBUSalinanNilaiOR...) {
		require.Equal(t, "1234", row.Value(name), "kolom %s", name)
	}

	require.NotContains(t, row, "FlagASO", "singkatan bulan bukan kolom susunan rinci")
}

// komiteDerive pada Non-MBU mengosongkan kolom hari kerja saat kalender tidak tersedia —
// bukan mengisinya dengan angka yang dihitung tanpa hari libur.
func TestKomiteDeriveKosongkanHariKerjaSaatKalenderTidakAda(t *testing.T) {
	hitung, err := komiteDerive(t.Context(), NewRepo(nil, nil),
		reportklaim.Filter{BusinessLine: reportklaim.BusinessLineNonMBU})
	require.NoError(t, err)

	row := reportklaim.Row{
		"CountryID":            "07/03/2026",
		"RCV_ID":               "01/01/2025",
		"Conveyance":           "10/04/2026",
		"Country":              "01/02/2026",
		"TanggalCloseUntukTAT": "10/04/2026",
		"TanggalTerimaLOD":     "01/03/2026",
		"TanggalTransferKasir": "10/03/2026",
	}
	hitung(row)

	require.Equal(t, "03", row.Value("AreaClaimId"))
	require.Equal(t, "2026", row.Value("ContractNo"))
	require.Equal(t, "2025", row.Value("Remark"))

	// "BULAN CLOSE" dan "TAHUN CLOSE" TIDAK diisi — alias sumbernya salah ketik, dan
	// cacat Pega dibiarkan seperti Pega.
	require.NotContains(t, row, "CopyFrom")
	require.NotContains(t, row, "CreateFrom")

	require.Empty(t, row.Value("ExGratiaNote"), "kalender tidak tersedia")
	require.Empty(t, row.Value("FlagASO"))
}

// Kolom hari kerja terisi ketika kalendernya ada.
func TestHariKerjaSelMemotongAkhirPekanDanLibur(t *testing.T) {
	libur := reportklaim.NewHolidayCalendar([]time.Time{
		bacaTanggalCSV("07/09/2026"), // Senin
	})

	// Jumat 04/09 → Selasa 08/09: Sabtu, Minggu, dan Senin libur tidak dihitung.
	require.Equal(t, "1", hariKerjaSel("04/09/2026", "08/09/2026", libur))
}

func TestOSKomiteDeriveMengisiTahunPeriodePolis(t *testing.T) {
	row := reportklaim.Row{"RCV_ID": "01/01/2025"}
	osKomiteDerive(row)

	require.Equal(t, "2025", row.Value("Remark"))
}
