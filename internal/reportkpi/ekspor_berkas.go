package reportkpi

// Susunan berkas ekspor, DIAMBIL DARI PEGA — bukan diturunkan dari susunan kolom layar.
//
// # Kenapa ia tidak boleh diturunkan dari grid
//
// Sampai 2026-10-09 berkas ekspor disusun dari `Grid.Columns` ditambah daftar komponen,
// dengan alasan "isi berkas dan isi layar tidak boleh berselisih". Alasan itu terdengar
// benar dan ternyata keliru: **di Pega keduanya memang berbeda**, dan perbedaannya
// disengaja.
//
// Buktinya ada di parameter pemanggilan `pxConvertResultsToCSV`:
//
//	CSVPropHeaders   baris judul berkas, ditulis pemanggil
//	CSVProperties    properti mana yang ditulis, dan dalam urutan apa
//	FileName         nama berkasnya
//
// Ketiganya ditetapkan terpisah dari grid. Pada grid Summary, misalnya, kolom `Status`
// DIGAMBAR tetapi TIDAK diekspor, dan kolom terakhirnya tertulis `KATEGORY` di berkas
// sementara gridnya menulis `KATEGORI`.
//
// Seluruh susunan di berkas ini terbaca di `docs/spesifikasi-ekspor-kpi-pega.md`, yang
// dibangun otomatis dari `Activity/*.xml`.
type ExportSpec struct {
	// FileLabel adalah nama berkas sebagaimana Pega menyetelnya, apa adanya.
	//
	// Ia dipakai sebagai dasar nama unduhan. Pega tidak menambahkan akhiran apa pun;
	// di sini akhiran `.csv` ditambahkan karena peramban membutuhkannya untuk membuka
	// berkasnya dengan program yang benar.
	FileLabel string

	// Headers adalah baris judul berkas, APA ADANYA seperti `CSVPropHeaders` — termasuk
	// salah ketiknya. `D-13` berlaku untuk berkas yang dibaca pengguna, bukan hanya layar.
	Headers []string

	// Fields adalah kunci data KAMI, sejajar satu-ke-satu dengan Headers.
	//
	// Ia sengaja dipisah dari Headers: nama properti Pega — `UserTeknisGroup`, `ProdKe`,
	// `CityID` — tidak ada hubungannya dengan isinya (utang teknis §4.2), dan memakainya
	// sebagai kunci di sini hanya memindahkan kekacauan itu ke sistem baru.
	Fields []string
}

// ExportSummaryAdjuster adalah berkas "LAPORAN SUMMARY KPI ADJUSTER" — 11 kolom.
//
// Dua hal yang BERBEDA dari grid di layar, dan keduanya disengaja:
//
//	kolom `Status`  DIGAMBAR di grid, TIDAK ADA di berkas
//	kolom terakhir  tertulis `KATEGORY` di berkas, `KATEGORI` di grid
//
// Keduanya diperiksa ke `Activity/PNCReportKPIAdjuster_act-Act.xml`. Yang kedua salah
// ketik, dan salah ketik itu ikut dibawa — pengguna sudah membacanya begitu bertahun-tahun,
// dan berkas yang judulnya "diperbaiki" tidak dapat dibandingkan dengan berkas lama.
func ExportSummaryAdjuster() ExportSpec {
	fields := []string{FieldAdjusterName}
	headers := []string{"ADJUSTER"}

	// Kesembilan komponen mengikuti urutan Components(), yang sudah disamakan dengan grid
	// Pega. Judulnya pun sama persis dengan judul kolom berkas, sehingga tidak ditulis
	// ulang di sini — satu daftar, bukan dua yang harus dijaga kesamaannya.
	for _, component := range components {
		fields = append(fields, component.Code)
		headers = append(headers, component.Label)
	}

	fields = append(fields, FieldCategory)
	headers = append(headers, "KATEGORY")

	return ExportSpec{
		FileLabel: "LAPORAN SUMMARY KPI ADJUSTER",
		Headers:   headers,
		Fields:    fields,
	}
}

// ExportPICScorecard adalah berkas "Laporan KPI" tab KPI PIC Teknik — 6 kolom.
//
// # Berkas ini SEBELUMNYA tidak ada sama sekali
//
// Tab KPI PIC Teknik di Pega punya DUA tombol ekspor, dan kami hanya membangun yang kedua:
//
//	Export Data KPI  di sebelah Cari              -> berkas ini, 6 kolom
//	Export Data KPI  di sebelah Pilih Data KPI    -> empat kumpulan data klaim mentah
//
// Yang pertama mengekspor PENILAIANNYA — persis isi grid di layar. Ia terlewat karena
// seluruh perhatian tertuju pada "Pilih Data KPI", yang memang lebih rumit.
//
// # Kenapa judul kolom keduanya berbeda dari grid
//
// Grid menulis `KATEGORI`, berkas menulis `KETERANGAN`. Keduanya menunjuk hal yang sama —
// nama komponen KPI — dan keduanya ditulis apa adanya (`D-13`), karena itulah yang dibaca
// pengguna selama ini di masing-masing tempat.
func ExportPICScorecard() ExportSpec {
	return ExportSpec{
		// Pega menyusunnya `"Laporan KPI "+" , "+Param.awal+" - "+Param.akhir`, sehingga
		// ada DUA spasi sebelum koma. Bentuk `Param.awal` tidak terbaca di export, dan
		// periodenya karena itu ditambahkan pemanggil — lihat picLaporanFilename.
		FileLabel: "Laporan KPI",
		Headers: []string{
			"PIC", "KETERANGAN", "TOTAL DATA", "JUMLAH TERCAPAI", "TERCAPAI (%)", "NILAI",
		},
		Fields: []string{
			FieldPICName, FieldPICMetric, FieldPICTotal,
			FieldPICAchieved, FieldPICPercent, FieldPICValue,
		},
	}
}
