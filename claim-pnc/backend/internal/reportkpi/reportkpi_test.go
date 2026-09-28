package reportkpi_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/reportkpi"
)

// caller adalah identitas pemanggil yang sah, dipakai hampir seluruh uji di sini.
func caller() reportkpi.Caller {
	return reportkpi.Caller{Login: "PENYELIACONTOH"}
}

// validInput adalah isian yang lolos seluruh aturan, dipakai sebagai titik tolak.
func validInput() reportkpi.QueryInput {
	return reportkpi.QueryInput{
		ReportType: "FINAL",
		From:       "2026-03-01",
		To:         "2026-03-31",
	}
}

func TestPermintaanYangSahDiterima(t *testing.T) {
	query, err := reportkpi.NewQuery(validInput(), caller())
	require.NoError(t, err)

	require.Equal(t, reportkpi.TypeFinal, query.ReportType)
	require.Equal(t, "2026-03-01", query.Range.From)
	require.Equal(t, "2026-03-31", query.Range.To)
	require.Empty(t, query.Adjuster, "adjuster kosong berarti seluruh adjuster")
	require.Equal(t, "PENYELIACONTOH", query.Caller.Login)
}

// Identitas yang tidak terbaca ditolak SEBELUM apa pun yang lain diperiksa.
//
// Bukan karena penyaringnya membutuhkan identitas — laporannya tidak disaring per
// pengguna — melainkan karena pembukaannya wajib tercatat atas nama seseorang (`D-59`).
func TestIdentitasYangTidakTerbacaDitolak(t *testing.T) {
	_, err := reportkpi.NewQuery(validInput(), reportkpi.Caller{Login: "   "})
	require.ErrorIs(t, err, reportkpi.ErrCallerUnknown)
}

// SELURUH pelanggaran dilaporkan sekaligus (`P-5`, `11-CROSSCUTTING.md` §1.1).
//
// Pengguna yang menekan "Cari" dengan form kosong melanggar tiga hal, dan diberi tahu
// ketiganya dalam satu kali jalan — bukan satu, lalu satu lagi.
func TestFormKosongMelaporkanKetigaPelanggaranSekaligus(t *testing.T) {
	_, err := reportkpi.NewQuery(reportkpi.QueryInput{}, caller())

	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 3)

	fields := map[string]string{}
	for _, v := range validation.Violations {
		fields[v.Field] = v.Message
	}
	require.Contains(t, fields, reportkpi.FieldReportType)
	require.Contains(t, fields, reportkpi.FieldDateFrom)
	require.Contains(t, fields, reportkpi.FieldDateTo)
}

// "Belum diisi" dan "salah bentuk" dibedakan: tindakan penggunanya berbeda.
func TestPesanMembedakanBelumDiisiDariSalahBentuk(t *testing.T) {
	input := validInput()
	input.From = ""
	input.To = "31/03/2026" // bentuk layar lama, bukan bentuk kontrak API

	_, err := reportkpi.NewQuery(input, caller())

	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)

	pesan := map[string]string{}
	for _, v := range validation.Violations {
		pesan[v.Field] = v.Message
	}
	require.Contains(t, pesan[reportkpi.FieldDateFrom], "wajib diisi")
	require.Contains(t, pesan[reportkpi.FieldDateTo], "tidak terbaca")
}

// Tanggal yang tidak ada di kalender ditolak di sini, bukan diserahkan ke basis data.
func TestTanggalYangTidakAdaDitolak(t *testing.T) {
	input := validInput()
	input.From = "2026-02-30"

	_, err := reportkpi.NewQuery(input, caller())

	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, reportkpi.FieldDateFrom, validation.Violations[0].Field)
}

func TestPeriodeTerbalikDitolak(t *testing.T) {
	input := validInput()
	input.From = "2026-03-31"
	input.To = "2026-03-01"

	_, err := reportkpi.NewQuery(input, caller())

	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1,
		"urutan diperiksa hanya bila kedua tanggal terbaca")
	require.Equal(t, reportkpi.FieldDateTo, validation.Violations[0].Field)
}

// Periode satu hari SAH: batas bawah dan batas atas boleh sama.
//
// Diuji khusus karena pemeriksaan urutan yang keliru memakai `!Before` alih-alih `After`
// akan menolaknya — dan periode satu hari adalah pemakaian yang wajar.
func TestPeriodeSatuHariDiterima(t *testing.T) {
	input := validInput()
	input.From = "2026-03-15"
	input.To = "2026-03-15"

	_, err := reportkpi.NewQuery(input, caller())
	require.NoError(t, err)
}

// Tipe report dibakukan menjadi huruf besar — itulah bentuk yang dibandingkan dengan isi
// kolom `TIPE`.
func TestTipeReportDibakukanMenjadiHurufBesar(t *testing.T) {
	input := validInput()
	input.ReportType = "  outstanding  "

	query, err := reportkpi.NewQuery(input, caller())
	require.NoError(t, err)
	require.Equal(t, reportkpi.TypeOutstanding, query.ReportType)
}

func TestTipeReportYangTidakDikenalDitolakDenganMenyebutPilihannya(t *testing.T) {
	input := validInput()
	input.ReportType = "SELESAI"

	_, err := reportkpi.NewQuery(input, caller())

	var validation *reportkpi.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Contains(t, validation.Violations[0].Message, "OUTSTANDING")
	require.Contains(t, validation.Violations[0].Message, "FINAL")
	require.Contains(t, validation.Violations[0].Message, "ALL")
}

// ALL memecah menjadi dua tipe; dua lainnya menghasilkan dirinya sendiri.
//
// Inilah yang menggantikan `UNION ALL` di sistem lama, dan yang membuat satu adjuster
// tampil dua baris pada tipe ALL.
func TestTipeAllMemecahMenjadiDuaTipe(t *testing.T) {
	require.Equal(t,
		[]reportkpi.ReportType{reportkpi.TypeOutstanding, reportkpi.TypeFinal},
		reportkpi.TypeAll.SplitsIntoTypes())

	require.Equal(t,
		[]reportkpi.ReportType{reportkpi.TypeFinal},
		reportkpi.TypeFinal.SplitsIntoTypes())
}

// Kesembilan komponen ada, berurutan, dan tidak ada yang kembar.
//
// Urutannya mengikuti urutan SELECT kueri lama, yang sama dengan urutan parameter
// `INSERT_KPIADJUSTER.prc`. Kedua sumber itu sepakat, dan kesepakatan itulah yang
// menjadikan urutan ini fakta.
func TestKomponenLengkapDanTidakKembar(t *testing.T) {
	components := reportkpi.Components()
	require.Len(t, components, 9)

	codes := map[string]bool{}
	columns := map[string]bool{}
	for _, c := range components {
		require.NotEmpty(t, c.Code)
		require.NotEmpty(t, c.Label)
		require.NotEmpty(t, c.Column)

		require.Falsef(t, codes[c.Code], "kode komponen kembar: %s", c.Code)
		require.Falsef(t, columns[c.Column], "kolom komponen kembar: %s", c.Column)
		codes[c.Code] = true
		columns[c.Column] = true
	}

	require.Equal(t, reportkpi.ComponentSurvey, components[0].Code,
		"PENJADWALAN SURVEY adalah komponen pertama")
	require.Equal(t, reportkpi.ComponentTotal, components[8].Code,
		"NILAI adalah yang terakhir")
}

// Components() mengembalikan SALINAN.
//
// Pemanggil di lapisan transport menyusunnya menjadi jawaban JSON; slice yang dibagikan
// dapat diubah tanpa sengaja oleh salah satunya, dan akibatnya baru terlihat di layar.
func TestComponentsMengembalikanSalinan(t *testing.T) {
	first := reportkpi.Components()
	first[0].Label = "DIUBAH"

	require.NotEqual(t, "DIUBAH", reportkpi.Components()[0].Label)
}

// Judul kolom mengikuti caption layar lama APA ADANYA (`D-13`).
//
// Kesembilannya terverifikasi ada di `Section/ReportKPI_Section-Section.xml`. Diperiksa
// supaya "perbaikan" ejaan atau kapitalisasi tidak masuk diam-diam.
func TestJudulKomponenMengikutiCaptionLayarLama(t *testing.T) {
	expected := map[string]string{
		reportkpi.ComponentSurvey:            "PENJADWALAN SURVEY",
		reportkpi.ComponentImmediateAdvice:   "IMMEDIATE ADVICE",
		reportkpi.ComponentPreliminaryAdvice: "PRELIMINARY ADVICE",
		reportkpi.ComponentInterimReport:     "INTERIM REPORT",
		reportkpi.ComponentProgress:          "UPDATE PROGRESS",
		reportkpi.ComponentCommunication:     "TANGGAPAN KOMUNIKASI",
		reportkpi.ComponentPropose:           "PROPOSE ADJUSTMENT",
		reportkpi.ComponentFinalReport:       "FINAL REPORT",
		reportkpi.ComponentTotal:             "NILAI",
	}

	for _, c := range reportkpi.Components() {
		require.Equalf(t, expected[c.Code], c.Label,
			"judul komponen %s berubah dari caption layar lama", c.Code)
	}
}

// Nilai yang tidak ada BERBEDA dari nilai nol.
//
// Keduanya terlihat sama bila disimpan sebagai float telanjang, dan pada laporan penilaian
// kinerja perbedaannya nyata: yang pertama berarti belum dinilai, yang kedua berarti tidak
// mendapat poin.
func TestNilaiKosongBerbedaDariNilaiNol(t *testing.T) {
	kosong := reportkpi.EmptyScore()
	require.False(t, kosong.Present)

	nol := reportkpi.NewScore(0)
	require.True(t, nol.Present)
	require.Equal(t, float64(0), nol.Value)
}

// Ketiga tab ada; dua di antaranya terhalang dan MENYEBUTKAN sebabnya.
//
// Tab terhalang tanpa sebab hanya memindahkan pekerjaan menebak kepada pembacanya — dan
// yang membacanya adalah penguji yang sedang memutuskan apakah ini cacat atau bukan.
func TestTabTerhalangMenyebutkanSebabnya(t *testing.T) {
	tabs := reportkpi.Tabs()
	require.Len(t, tabs, 3)

	terhalang := 0
	for _, tab := range tabs {
		require.NotEmpty(t, tab.Code)
		require.NotEmpty(t, tab.Title)

		if !tab.Blocked {
			require.NotEmptyf(t, tab.Grids, "tab %s tidak punya grid", tab.Code)
			continue
		}
		terhalang++
		require.NotEmptyf(t, tab.BlockedReason,
			"tab %s terhalang tanpa menyebut sebabnya", tab.Code)
	}
	require.Equal(t, 0, terhalang,
		"ketiga tab sudah dibangun; tidak ada lagi yang terhalang")
}

// Tab KPI Admin punya kartu skor dan grid rinciannya.
//
// Kartu skor sengaja TANPA kolom tetap: ia bukan tabel berbaris-baris melainkan satu kartu
// berisi metrik berurutan, dan metriknya berbeda antar kelompok. Yang menyusunnya adalah
// BuildScorecard, bukan daftar kolom.
func TestTabAdminPunyaKartuSkorDanRincian(t *testing.T) {
	tab := reportkpi.AdminTab()
	require.False(t, tab.Blocked)
	require.Len(t, tab.Grids, 2)

	require.Empty(t, reportkpi.AdminGridFor(reportkpi.GridScorecard).Columns,
		"kartu skor tidak punya kolom tetap")
	require.NotEmpty(t, reportkpi.AdminGridFor(reportkpi.GridAdminDetail).Columns)
}

// Kolom grid rincian BERBEDA antar kelompok, dan tidak satu pun kelompok melihat kolom
// milik kelompok lain.
//
// Yang paling mudah keliru: judul "Tgl Terima Dokumen" ada di KEDUA kelompok tetapi
// menunjuk kolom basis data yang berbeda. Bila penyaringnya gagal, salah satu kelompok akan
// melihat judul itu DUA KALI — dan salah satunya berisi tanggal yang salah.
func TestKolomRincianAdminBerbedaAntarKelompok(t *testing.T) {
	grid := reportkpi.AdminGridFor(reportkpi.GridAdminDetail)

	judul := func(group reportkpi.AdminGroup) []string {
		result := []string{}
		for _, c := range grid.ColumnsForGroup(group) {
			result = append(result, c.Title)
		}
		return result
	}

	nonMBU := judul(reportkpi.AdminGroupNonMBU)
	pa := judul(reportkpi.AdminGroupPA)

	require.Contains(t, nonMBU, "BUSINESS")
	require.Contains(t, nonMBU, "Flag")
	require.NotContains(t, nonMBU, "Tgl Terima LOD")

	require.Contains(t, pa, "Tgl Terima LOD")
	require.Contains(t, pa, "Status SLA Pembayaran Klaim")
	require.NotContains(t, pa, "BUSINESS")
	require.NotContains(t, pa, "Flag")

	// Judul yang sama tidak boleh muncul dua kali pada satu kelompok.
	for _, daftar := range [][]string{nonMBU, pa} {
		terlihat := map[string]bool{}
		for _, title := range daftar {
			require.Falsef(t, terlihat[title], "judul kolom kembar: %s", title)
			terlihat[title] = true
		}
	}
}

func TestTabBawaanAdalahTabYangDibangun(t *testing.T) {
	tab, found := reportkpi.FindTab(reportkpi.DefaultTabKPI)
	require.True(t, found)
	require.False(t, tab.Blocked, "tab bawaan tidak boleh tab yang terhalang")
	require.Equal(t, reportkpi.TabAdjuster, tab.Code)
}

// Tab KPI Adjuster punya kedua grid, dan keduanya menyebut kolom tetapnya.
func TestTabAdjusterPunyaKeduaGrid(t *testing.T) {
	grids := reportkpi.AdjusterTab().Grids
	require.Len(t, grids, 2)

	codes := []string{grids[0].Code, grids[1].Code}
	require.Contains(t, codes, reportkpi.GridSummary)
	require.Contains(t, codes, reportkpi.GridDetail)

	for _, grid := range grids {
		require.NotEmptyf(t, grid.Title, "grid %s tidak punya judul", grid.Code)
		require.NotEmptyf(t, grid.Columns, "grid %s tidak punya kolom tetap", grid.Code)

		// Kolom komponen TIDAK boleh ikut di sini: keduanya memakai kesembilan komponen
		// yang sama, dan menuliskannya di grid berarti dua daftar yang dapat berselisih.
		for _, column := range grid.Columns {
			_, isComponent := reportkpi.FindComponent(column.Key)
			require.Falsef(t, isComponent,
				"kolom komponen %s tidak boleh ditulis ulang di grid %s",
				column.Key, grid.Code)
		}
	}
}

// Kolom TIPE pada grid Summary HANYA muncul pada tipe report ALL.
//
// Ini meniru Pega apa adanya, dan buktinya dua rule yang berbeda:
//
//	GetSummaryKPIAdjuster-SQL.xml     SELECT adjuster …                 tanpa kolom tipe
//	GetSummaryKPIAdjusterALL-SQL.xml  SELECT adjuster, 'OUTSTANDING' …  DENGAN kolom tipe
//
// Menggambarnya pada tipe tunggal akan menambah kolom yang tidak ada di layar lama, dan
// isinya pun tidak berarti apa-apa di sana — seluruh barisnya bernilai sama.
func TestKolomTipeHanyaMunculPadaTipeGabungan(t *testing.T) {
	var summary reportkpi.Grid
	for _, grid := range reportkpi.AdjusterTab().Grids {
		if grid.Code == reportkpi.GridSummary {
			summary = grid
		}
	}
	require.NotEmpty(t, summary.Columns)

	judul := func(columns []reportkpi.Column) []string {
		result := make([]string, 0, len(columns))
		for _, c := range columns {
			result = append(result, c.Title)
		}
		return result
	}

	require.Equal(t, []string{"ADJUSTER", "TIPE"},
		judul(summary.ColumnsFor(reportkpi.TypeAll)),
		"tipe ALL menggabungkan dua kelompok, sehingga penandanya wajib ada")

	for _, tunggal := range []reportkpi.ReportType{
		reportkpi.TypeOutstanding, reportkpi.TypeFinal,
	} {
		require.Equalf(t, []string{"ADJUSTER"}, judul(summary.ColumnsFor(tunggal)),
			"kolom TIPE tidak boleh digambar pada tipe tunggal %s", tunggal)
	}
}

// Grid Detail TIDAK punya kolom yang muncul-hilang.
//
// Diperiksa supaya penanda `OnlyOnCombinedType` tidak menyebar ke grid yang tidak
// membutuhkannya — kolomnya di sana berasal dari perhitungan clipboard Pega, bukan dari
// dua kueri yang berbeda.
func TestGridDetailTidakPunyaKolomYangMunculHilang(t *testing.T) {
	for _, grid := range reportkpi.AdjusterTab().Grids {
		if grid.Code != reportkpi.GridDetail {
			continue
		}
		require.Len(t, grid.ColumnsFor(reportkpi.TypeFinal), len(grid.Columns))
		require.Len(t, grid.ColumnsFor(reportkpi.TypeAll), len(grid.Columns))
	}
}

// Selisih terencana menyebut hal-hal yang PALING MUNGKIN dilaporkan sebagai kerusakan.
//
// Yang tidak dinyatakan di muka akan dilaporkan sebagai cacat, dan menelusurinya kembali
// jauh lebih mahal daripada menuliskannya sekarang (`D-54`).
func TestSelisihTerencanaMenyebutYangPalingMungkinDilaporkanSebagaiCacat(t *testing.T) {
	require.NotEmpty(t, reportkpi.PlannedDifferences)

	joined := strings.ToLower(strings.Join(reportkpi.PlannedDifferences, " "))
	for _, topik := range []string{
		"membaca saja", // layar tidak menghitung ulang
		"dropdown",     // isi dropdown adjuster berbeda sumbernya
		"periode",      // periode menjadi wajib
		"dua baris",    // tipe ALL menghasilkan dua baris per adjuster
		"nilai",        // NILAI kolom tersendiri, bukan jumlah
		"tanda hubung", // nilai kosong bukan 0
		"dipaginasi",   // rincian dipaginasi di server
	} {
		require.Containsf(t, joined, topik,
			"selisih terencana belum menyebut %q", topik)
	}
}
