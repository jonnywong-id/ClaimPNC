package reportklaim

// Kode laporan kelompok Klaim.
const (
	CodeTAT                 Code = "tat"
	CodeKlaimHarian         Code = "klaim-harian"
	CodeRejectKlaim         Code = "reject-klaim"
	CodeCloseKlaim          Code = "close-klaim"
	CodeTemporaryCloseKlaim Code = "temporary-close-klaim"
	CodeAIKlaim             Code = "ai-klaim"
)

// catalogKlaim menyusun keenam panel kelompok Klaim.
func catalogKlaim() []Report {
	return []Report{
		{
			Code:         CodeTAT,
			Title:        "REPORT TAT",
			Group:        GroupKlaim,
			Actions:      oneAction("Export Data TAT", nil),
			FileBaseName: "Laporan TAT",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCTATReport1_Act",
				// Ketiganya dipakai satu laporan: kueri induk mengambil klaim menurut
				// tanggal registrasi, dua sisanya mengambil PLA dan DLA per klaim.
				SQLRule: []string{"BroswseKlaimByRegisterDate", "BroswsePLA_SQL", "BroswseDLA_SQL"},
			},
			Availability: Ready(),
			// Rentang tanggalnya adalah TANGGAL REGISTRASI, bukan tanggal kejadian:
			// `trunc(a.registerdate) >= ... and trunc(a.registerdate) <= ...`.
			// Nama rule-nya menyebutkannya apa adanya — "ByRegisterDate".
			variants: []variant{
				personalVariant(
					"lini PA atau Travel — 52 kolom, memuat DOB, usia, diagnosa, dan sifat kerugian",
					colsTATPersonal,
				),
				{
					when:    func(f Filter) bool { return f.BusinessLine == BusinessLineBonding },
					why:     "lini Bonding — 34 kolom",
					columns: colsTATBonding,
				},
				always("Non-MBU atau seluruh lini — 39 kolom", colsTATNonMBU),
			},
		},
		{
			Code:         CodeKlaimHarian,
			Title:        "REPORT KLAIM HARIAN",
			Group:        GroupKlaim,
			Actions:      oneAction("Export Data Klaim Harian", nil),
			FileBaseName: "Laporan Harian",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportHarian_act1",
				SQLRule:  []string{"BroswseKlaimperday"},
			},
			Availability: Ready(),
			// # Satu susunan kolom, padahal activity-nya punya dua
			//
			// `PNCReportHarian_act1` memuat DUA langkah `pxConvertResultsToCSV`: satu
			// berkolom 30 bernama "Laporan Harian", satu berkolom 10 bernama
			// "Laporan TAT". Yang dipakai di sini adalah yang 30 kolom — namanya yang
			// cocok dengan panelnya.
			//
			// **Syarat pemilih antara keduanya tidak dapat ditetapkan dari export.**
			// Keduanya berdeskripsi sama persis (".csv Report NonMBU per hari"),
			// bersumber halaman yang sama, dan precondition terdekatnya menguji isi
			// BARIS (`...Email=="10145"`) — sesuatu yang tidak dapat memilih susunan
			// kolom sebuah berkas. Menebaknya berarti separuh unduhan keluar dengan 10
			// kolom tanpa ada yang tahu sebabnya.
			//
			// Pertanyaannya ditujukan ke Tim Pega; sampai dijawab, panel ini selalu
			// menghasilkan susunan 30 kolom.
			variants: []variant{
				always("30 kolom — susunan \"Laporan Harian\"", colsKlaimHarian),
			},
		},
		{
			Code:    CodeRejectKlaim,
			Title:   "REPORT DATA REJECT KLAIM",
			Group:   GroupKlaim,
			Actions: oneAction("Export Data Reject Klaim", nil),
			// # Nama berkas TIDAK ditiru apa adanya, dan ini satu-satunya alasannya
			//
			// `Param.FileName` pada activity ini berbunyi "Laporan Data Close" — nama
			// laporan LAIN. Ia salin-tempel dari `PNCReportDataClose_act`, terbukti dari
			// susunan kolomnya yang berbeda sama sekali.
			//
			// Menirunya berarti tombol "Export Data Reject Klaim" mengunduh berkas
			// bernama "Laporan Data Close.csv", lalu bertumpuk di folder unduhan dengan
			// berkas laporan Close yang sebenarnya — dua laporan berbeda dengan satu
			// nama. Itu bukan kesetaraan perilaku yang berguna, dan bukan salah satu dari
			// 13 butir `D-49` yang harus dipertahankan.
			FileBaseName: "Laporan Data Reject Klaim",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataReject_act",
				SQLRule:  []string{"ExportDataRejectKlaim"},
			},
			Availability: Ready(),
			variants: []variant{
				always("33 kolom", colsRejectKlaim),
			},
		},
		{
			Code:         CodeCloseKlaim,
			Title:        "REPORT DATA CLOSE KLAIM",
			Group:        GroupKlaim,
			Actions:      oneAction("Export Data Close Klaim", nil),
			FileBaseName: "Laporan Data Close",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataClose_act",
				// DUA kueri, dan nama keduanya yang memastikan susunan kolom mana yang
				// berlaku: yang berakhiran NONMBU menghasilkan berkas 81 kolom.
				SQLRule: []string{"ExportDataCloseKlaim", "ExportDataCloseKlaimNONMBU"},
			},
			Availability: Ready(),
			variants: []variant{
				nonMBUVariant("Non-MBU — 81 kolom, kueri ExportDataCloseKlaimNONMBU", colsCloseKlaim),
				always("selain Non-MBU — 11 kolom, kueri ExportDataCloseKlaim", colsCloseKlaimRingkas),
			},
		},
		{
			Code:  CodeTemporaryCloseKlaim,
			Title: "REPORT DATA TEMPORARY CLOSE KLAIM",
			Group: GroupKlaim,
			// Panel ini berbagi activity dengan panel di atasnya dan dibedakan HANYA oleh
			// parameter `temp`. Tanpa mencatatnya, keduanya terbaca sebagai duplikat.
			Actions:      oneAction("Export Data Temporary Close Claim", map[string]string{"temp": "temp"}),
			FileBaseName: "Laporan Data Temporary Close",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataClose_act",
				SQLRule:  []string{"ExportDataCloseKlaim", "ExportDataCloseKlaimNONMBU"},
			},
			Availability: Ready(),
			variants: []variant{
				nonMBUVariant("Non-MBU — 81 kolom", colsCloseKlaim),
				always("selain Non-MBU — 11 kolom", colsCloseKlaimRingkas),
			},
		},
		{
			Code:         CodeAIKlaim,
			Title:        "REPORT DATA AI KLAIM",
			Group:        GroupKlaim,
			Actions:      oneAction("Export DATA AI KLAIM", nil),
			FileBaseName: "Data AI Klaim",
			// Kuerinya tidak menerima rentang tanggal sama sekali — `GetBrowseDataAIPA`
			// tanpa satu pun parameter. Menampilkan isian tanggal pada kartunya akan
			// membuat pengguna mengisinya, lalu menyimpulkan hasilnya salah.
			Uses: FilterBusinessLine,
			Source: Source{
				Activity: "ExportHasilDataAIKlaim",
				SQLRule:  []string{"GetBrowseDataAIPA"},
			},
			Availability: Ready(),
			// # Kolom ke-14 memang tidak berjudul
			//
			// `CSVProperties` menyebut 14 properti, `CSVPropHeaders` hanya 13 judul.
			// Kolom terakhir (`District`) karena itu keluar tanpa judul di sistem lama,
			// dan di sini pun demikian — diisi judul karangan akan membuat berkasnya
			// berbeda dari yang selama ini diterima penggunanya.
			variants: []variant{
				always("14 kolom; kolom terakhir tanpa judul, mengikuti export", colsAIKlaim),
			},
		},
	}
}
