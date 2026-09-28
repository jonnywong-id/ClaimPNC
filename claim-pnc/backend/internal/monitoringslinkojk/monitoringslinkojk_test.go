package monitoringslinkojk_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
)

// Segmen kosong jatuh ke D01 — segmen yang terbuka lebih dulu di layar lama.
func TestParseSegmentDefaultsToD01(t *testing.T) {
	segment, known := monitoringslinkojk.ParseSegment("")
	require.True(t, known)
	require.Equal(t, monitoringslinkojk.SegmentD01, segment)
}

func TestParseSegmentAcceptsBothSegments(t *testing.T) {
	for _, raw := range []string{"D01", "d01", " f06 ", "F06"} {
		segment, known := monitoringslinkojk.ParseSegment(raw)
		require.Truef(t, known, "segmen %q seharusnya dikenal", raw)
		require.True(t, segment.Valid())
	}
}

func TestParseSegmentRejectsUnknown(t *testing.T) {
	_, known := monitoringslinkojk.ParseSegment("D02")
	require.False(t, known)
}

// Ketiga pilihan Business Name dikenal, termasuk pilihan kosong yang berarti "seluruhnya".
func TestParseBusinessScope(t *testing.T) {
	for raw, want := range map[string]monitoringslinkojk.BusinessScope{
		"":              monitoringslinkojk.ScopeAll,
		"AS. KREDIT":    monitoringslinkojk.ScopeCreditInsurance,
		"as. kredit":    monitoringslinkojk.ScopeCreditInsurance,
		" SURETY BOND ": monitoringslinkojk.ScopeSuretyBond,
	} {
		scope, valid := monitoringslinkojk.ParseBusinessScope(raw)
		require.Truef(t, valid, "pilihan %q seharusnya sah", raw)
		require.Equal(t, want, scope)
	}

	_, valid := monitoringslinkojk.ParseBusinessScope("MOTOR")
	require.False(t, valid)
}

// Paginasi dirapikan di lapisan modul, bukan di transport.
func TestFilterNormalizeFillsPaging(t *testing.T) {
	filter := monitoringslinkojk.Filter{Page: 0, Size: 0}.Normalize()
	require.Equal(t, 1, filter.Page)
	require.Equal(t, monitoringslinkojk.DefaultPageSize, filter.Size)

	oversized := monitoringslinkojk.Filter{Page: 3, Size: 100_000}.Normalize()
	require.Equal(t, monitoringslinkojk.MaxPageSize, oversized.Size,
		"ukuran halaman wajib dipagari supaya satu permintaan tidak menarik puluhan juta baris")

	require.Equal(t, 2*monitoringslinkojk.DefaultPageSize,
		monitoringslinkojk.Filter{Page: 3, Size: monitoringslinkojk.DefaultPageSize}.
			Normalize().Offset())
}

// Rentang terbalik HARUS terdeteksi.
//
// Sistem lama tidak memeriksanya dan menjawabnya dengan nol baris yang tampak wajar —
// dan pada layar pemantauan laporan regulator, nol baris terbaca sebagai "tidak ada yang
// perlu dilaporkan".
func TestFilterDateRangeReversed(t *testing.T) {
	early := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)

	require.True(t, monitoringslinkojk.Filter{
		DateOfLoss:            &late,
		DateOfRequestDocument: &early,
	}.DateRangeReversed())

	require.False(t, monitoringslinkojk.Filter{
		DateOfLoss:            &early,
		DateOfRequestDocument: &late,
	}.DateRangeReversed())

	// Satu batas saja bukan rentang terbalik; layar lama pun membolehkannya.
	require.False(t, monitoringslinkojk.Filter{DateOfLoss: &late}.DateRangeReversed())
	require.False(t, monitoringslinkojk.Filter{}.DateRangeReversed())
}

// ============================================================================
// KATALOG KOLOM
// ============================================================================

// Jumlah kolom dikunci pada angka yang terbaca dari export.
//
// Angka ini BUKAN sekadar penjaga regresi: ia yang menyatakan bahwa layar barunya
// menggambar kolom sebanyak layar lamanya. Bila sebuah kolom laporan regulator hilang,
// tidak ada apa pun di layar yang menandakannya — uji inilah satu-satunya yang menyalak.
func TestColumnCounts(t *testing.T) {
	require.Len(t, monitoringslinkojk.Columns(monitoringslinkojk.SegmentD01), 20,
		"grid D01 di Sec_SegmentD01_1 punya 20 kolom")
	require.Len(t, monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentD01), 27,
		"CSVPropHeaders pada ExportDataSlinkD01 memuat 27 judul")
	// Grid F06 memakai daftar kolom yang SAMA dengan D01 — terbukti dari layar Pega yang
	// berjalan dan dari alias `GetDataSlinkAllFOG-SQL.xml`.
	//
	// Uji ini sempat menuntut 38, karena grid F06 dibangun dari judul BERKAS EKSPOR.
	// Angka 38 tetap dijaga — tetapi pada ExportHeaders, tempatnya yang benar.
	require.Len(t, monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06), 20,
		"grid F06 memakai kolom yang sama dengan grid D01")
	require.Len(t, monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentF06), 38,
		"CSVPropHeaders pada ExportDataSlinkFOG memuat 38 judul")
}

// Kedua grid identik — bukan mirip.
//
// Bila kelak salah satunya disunting sendirian, uji ini menyalak. Di Pega keduanya satu
// daftar yang sama, dan perbedaan di antara keduanya adalah cacat, bukan keputusan.
func TestBothGridsUseTheSameColumns(t *testing.T) {
	require.Equal(t,
		monitoringslinkojk.Columns(monitoringslinkojk.SegmentD01),
		monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06))
}

// ============================================================================
// EKSPOR F06 MISALIGN — DIREPLIKASI ATAS KEPUTUSAN WORK OWNER
// ============================================================================

// Berkas ekspor F06 SENGAJA tidak sejajar: 38 judul, 34 kolom data.
//
// Keputusan Work Owner 2026-09-26 sesudah selisihnya disampaikan beserta akibatnya.
// Uji ini menjaganya tetap begitu — tanpa uji, "merapikannya" adalah suntingan satu baris
// yang tidak akan ketahuan siapa pun sampai berkasnya dibandingkan dengan keluaran Pega.
func TestF06ExportReplicatesPegaMisalignment(t *testing.T) {
	headers := monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentF06)
	slots := monitoringslinkojk.ExportSlots(monitoringslinkojk.SegmentF06)

	require.Len(t, headers, 38, "CSVPropHeaders pada ExportDataSlinkFOG memuat 38 judul")
	require.Len(t, slots, 34, "CSVProperties pada aktivitas yang sama memuat 34 nama")
	require.NotEqual(t, len(headers), len(slots),
		"ketidaksejajaran inilah yang direplikasi; menyamakannya menghapusnya")
}

// Slot keenam adalah DUA nama properti yang tersambung tanpa koma di sumbernya.
//
// Ia yang menyebabkan seluruh slot sesudahnya bergeser terhadap judulnya, dan ia harus
// tetap ada sebagai SATU slot — memecahnya menjadi dua akan menggeser berkasnya kembali.
func TestF06ExportKeepsGluedPropertyName(t *testing.T) {
	slots := monitoringslinkojk.ExportSlots(monitoringslinkojk.SegmentF06)
	require.Equal(t, "ASMGenderASMDateOfBirth", slots[5].Header)
	require.Empty(t, slots[5].Key, "properti sambungan itu tidak pernah ada; slotnya kosong")
}

// Hanya TIGA slot yang terisi, dan ketiganya tercetak di bawah judul yang SALAH.
//
// Angka dan posisinya dikunci di sini supaya akibat replikasi ini terbaca dari uji, bukan
// hanya dari komentar.
func TestF06ExportHasThreePopulatedSlots(t *testing.T) {
	headers := monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentF06)
	slots := monitoringslinkojk.ExportSlots(monitoringslinkojk.SegmentF06)

	populated := map[int]string{}
	for i, slot := range slots {
		if slot.Key != "" {
			populated[i] = slot.Key
		}
	}

	require.Len(t, populated, 3)
	require.Equal(t, "alamat", populated[6])
	require.Equal(t, "kode_kantor_cabang", populated[32])
	require.Equal(t, "operasi_data", populated[33])

	// Inilah pergeserannya, dinyatakan sebagai angka: nilai Alamat tercetak di bawah
	// judul "Jenis Kelamin".
	require.Equal(t, "Jenis Kelamin", headers[6])
	require.Equal(t, "Perjanjian Pisah Harta", headers[32])
	require.Equal(t, "Melanggar BMPK", headers[33])
}

// Ekspor D01 SEJAJAR — 27 judul, 27 kolom data. Yang misalign hanya F06.
func TestD01ExportIsAligned(t *testing.T) {
	require.Len(t,
		monitoringslinkojk.ExportSlots(monitoringslinkojk.SegmentD01),
		len(monitoringslinkojk.ExportHeaders(monitoringslinkojk.SegmentD01)))
}

// SELURUH kolom grid F06 bersumber — tidak ada lagi kolom penanda kosong.
//
// Uji ini sempat menuntut kebalikannya: hanya 8 dari 38 yang bersumber. Itu berlaku
// ketika grid F06 keliru dibangun dari judul berkas ekspor. Sejak gridnya dikoreksi
// menjadi kedua puluh kolom yang sama dengan D01, setiap kolomnya punya alias di
// `GetDataSlinkAllFOG-SQL.xml`.
//
// Kekosongan itu belum hilang — ia pindah ke tempatnya yang benar, yaitu berkas ekspor
// F06, dan dijaga TestF06ExportIsIntentionallyMisaligned.
func TestEveryF06GridColumnIsSourced(t *testing.T) {
	all := monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06)
	available := monitoringslinkojk.AvailableColumns(monitoringslinkojk.SegmentF06)

	require.Len(t, available, len(all),
		"tidak boleh ada kolom grid F06 yang tanpa sumber")
}

// Seluruh kolom grid D01 bersumber. Bila satu saja tidak, pemetaannya ke
// GetDataSlinkAllFOGF06 sudah putus.
func TestD01ColumnsAllAvailable(t *testing.T) {
	require.Len(t,
		monitoringslinkojk.AvailableColumns(monitoringslinkojk.SegmentD01),
		len(monitoringslinkojk.Columns(monitoringslinkojk.SegmentD01)))
}

// Kunci kolom GRID wajib unik — ia dipakai sebagai kunci peta pada setiap baris, dan
// kunci ganda membuat satu kolom diam-diam menimpa kolom lain.
func TestGridColumnKeysUnique(t *testing.T) {
	for _, segment := range []monitoringslinkojk.Segment{
		monitoringslinkojk.SegmentD01,
		monitoringslinkojk.SegmentF06,
	} {
		seen := map[string]bool{}
		for _, column := range monitoringslinkojk.Columns(segment) {
			require.Falsef(t, seen[column.Key],
				"segmen %s: kunci kolom ganda %q", segment, column.Key)
			seen[column.Key] = true
		}
	}
}

// Kunci pada katalog EKSPOR D01 sengaja TIDAK unik: `Keterangan` muncul dua kali di
// `CSVPropHeaders`, dan itu direplikasi. Uji ini mengunci fakta tersebut supaya penyunting
// berikutnya tidak "merapikannya" tanpa menyadari bahwa duplikatnya memang ada di Pega.
func TestD01ExportRepeatsKeteranganTwice(t *testing.T) {
	count := 0
	for _, slot := range monitoringslinkojk.ExportSlots(monitoringslinkojk.SegmentD01) {
		if slot.Key == "keterangan" {
			count++
		}
	}
	require.Equal(t, 2, count,
		"ExportDataSlinkD01 memuat kolom Keterangan dua kali; itu ada di sumbernya")
}

// Setiap slot ekspor wajib punya judul; kunci boleh kosong.
//
// Kunci yang kosong berarti slotnya memang tidak punya sumber — 31 dari 34 slot F06
// begitu keadaannya, dan itu direplikasi dari Pega.
func TestExportSlotsHaveHeaders(t *testing.T) {
	for _, segment := range []monitoringslinkojk.Segment{
		monitoringslinkojk.SegmentD01,
		monitoringslinkojk.SegmentF06,
	} {
		for i, slot := range monitoringslinkojk.ExportSlots(segment) {
			require.NotEmptyf(t, slot.Header,
				"segmen %s: slot ke-%d tanpa nama properti", segment, i+1)
		}
		for i, header := range monitoringslinkojk.ExportHeaders(segment) {
			require.NotEmptyf(t, header,
				"segmen %s: judul ke-%d kosong", segment, i+1)
		}
	}
}

// Judul kolom yang salah ketik di Pega DIPERTAHANKAN — berkas CSV ini dibaca ulang oleh
// berkas kerja yang sudah ada di sisi pelapor.
func TestTypoHeadersPreserved(t *testing.T) {
	var found bool
	for _, column := range monitoringslinkojk.Columns(monitoringslinkojk.SegmentD01) {
		if column.Key == "kode_kolektibilitas" {
			require.Equal(t, "Kode Kelektibilitas", column.Header,
				"salah ketik pada Sec_SegmentD01_1 sengaja dipertahankan")
			found = true
		}
	}
	require.True(t, found)
}

// Nama berkas ekspor di Pega TERTUKAR, dan ketertukaran itu DIREPLIKASI.
//
// Keputusan Work Owner 2026-09-26. Uji ini menyatakan ketertukarannya disengaja —
// tanpanya, siapa pun yang membacanya akan "membetulkannya" dan mengira sedang
// memperbaiki salah ketik.
func TestFileNamesMatchPega(t *testing.T) {
	require.Equal(t, "Laporan F06 SLIK OJK",
		monitoringslinkojk.FileName(monitoringslinkojk.SegmentD01),
		"ekspor segmen D01 memang bernama F06 di Pega; jangan dibetulkan")
	require.Equal(t, "Laporan SLIK OJK D01",
		monitoringslinkojk.FileName(monitoringslinkojk.SegmentF06),
		"ekspor segmen F06 memang bernama D01 di Pega; jangan dibetulkan")
}

// Berkas contoh unggahan disalin UTUH: 25 kolom, sama dengan `CSVPropHeaders` di Pega.
//
// # Kenapa uji ini menyebut angkanya
//
// Versi pertama modul ini memotongnya menjadi 6 kolom, atas bacaan keliru bahwa berkasnya
// tidak sejajar. Yang misalign adalah `ExportDataSlinkFOG` (38 judul, 34 properti);
// berkas format ini justru sejajar — `CSVPropHeaders` dan `CSVProperties` sama-sama 25.
//
// Angka 25 di sini yang mencegah pemotongan itu terulang tanpa ada yang menyadarinya.
func TestTemplateMatchesPega(t *testing.T) {
	require.Len(t, monitoringslinkojk.TemplateColumns, 25,
		"CSVPropHeaders pada DownloadFileCSVFormatSlikOJK memuat 25 judul")
	require.Len(t, monitoringslinkojk.TemplateSample,
		len(monitoringslinkojk.TemplateColumns),
		"baris contoh wajib sejajar dengan judulnya")

	require.Equal(t, "PolicyNo", monitoringslinkojk.TemplateColumns[0])
	require.Equal(t, "OperasiData", monitoringslinkojk.TemplateColumns[24])

	// Enam sel pertama terisi; sembilan belas sisanya kosong — persis seperti berkas
	// aslinya, yang hanya men-set keenam properti pertama.
	for i := 0; i < 6; i++ {
		require.NotEmptyf(t, monitoringslinkojk.TemplateSample[i],
			"sel contoh ke-%d seharusnya terisi", i+1)
	}
	for i := 6; i < 25; i++ {
		require.Emptyf(t, monitoringslinkojk.TemplateSample[i],
			"sel contoh ke-%d seharusnya kosong", i+1)
	}
}

// Row.Get mengembalikan teks kosong untuk kolom yang tidak ada, bukan panik.
func TestRowGetMissingKey(t *testing.T) {
	var empty monitoringslinkojk.Row
	require.Equal(t, "", empty.Get("apa_saja"))

	row := monitoringslinkojk.Row{"no_klaim": "PNCN.26.0001"}
	require.Equal(t, "PNCN.26.0001", row.Get("no_klaim"))
	require.Equal(t, "", row.Get("nama_lengkap"))
}
