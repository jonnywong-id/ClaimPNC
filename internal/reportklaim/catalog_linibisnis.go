package reportklaim

// Kode laporan kelompok Lini Bisnis & Mitra Kerja Sama.
const (
	CodeKlaimHE             Code = "klaim-he"
	CodeRegistSimasOnline   Code = "regist-simas-online"
	CodeKlaimAsuransiKredit Code = "klaim-asuransi-kredit"
	CodeKlaimPerBisnis      Code = "klaim-per-bisnis"
	CodeKlaimTraveloka      Code = "klaim-traveloka"
	CodeKlaimPegiPegi       Code = "klaim-pegipegi"
)

// catalogLiniBisnis menyusun keenam panel kelompok Lini Bisnis & Mitra Kerja Sama.
//
// Yang menyatukannya: keenamnya menyaring SATU lini atau satu mitra tertentu, dan
// penyaringnya tertanam di laporannya — bukan dipilih pengguna lewat dropdown "Treaty".
func catalogLiniBisnis() []Report {
	return []Report{
		{
			Code:  CodeKlaimHE,
			Title: "REPORT KLAIM HE",
			Group: GroupLiniBisnis,
			// HE = Heavy Equipment (`CONTEXT.md`). Panel ini berbagi activity dengan
			// "Regist Simas On Line" di bawahnya, dibedakan HANYA oleh `idreportKlaim`.
			Actions:      oneAction("Export Data Klaim HE", map[string]string{"idreportKlaim": "1"}),
			FileBaseName: "Laporan Klaim HE",
			// Rentang tanggalnya OPSIONAL, dan itu terbaca dari rule-nya sendiri:
			// potongan penyaringnya dibungkus `@If((dari=="" && sampai==""),"", ...)`,
			// sehingga keduanya kosong berarti seluruh periode. Lihat Validate — modul
			// ini tetap mewajibkannya bila panelnya memakai FilterDateRange, dan panel
			// ini memang memakainya: menjalankan laporan tanpa batas periode atas data
			// puluhan juta baris (`D-10`) bukan kemudahan, melainkan kueri yang tidak
			// selesai.
			Uses: FilterDateRange,
			Source: Source{
				Activity: "PNCReportKlaimHE_act",
				SQLRule:  []string{"BrowseKlaimHE"},
			},
			Availability: Ready(),
			variants: []variant{
				always("27 kolom", colsKlaimHE),
			},
		},
		{
			Code:         CodeRegistSimasOnline,
			Title:        "REPORT DATA REGIST SIMAS ON LINE",
			Group:        GroupLiniBisnis,
			Actions:      oneAction("Export Data Regist Simas On Line", map[string]string{"idreportKlaim": "2"}),
			FileBaseName: "Laporan Regist Simas On Line",
			// `GetDataRegistBySimasOnline` tidak menerima satu pun parameter.
			Uses: 0,
			Source: Source{
				Activity: "PNCReportKlaimHE_act",
				SQLRule:  []string{"GetDataRegistBySimasOnline"},
			},
			Availability: Ready(),
			variants: []variant{
				always("12 kolom", colsRegistSimasOnline),
			},
		},
		{
			Code:  CodeKlaimAsuransiKredit,
			Title: "REPORT KLAIM ASURANSI KREDIT",
			Group: GroupLiniBisnis,
			// Spasi ganda pada "Klaim  Asuransi" ada di harness, bukan di sini.
			Actions:      oneAction("Export Data Klaim  Asuransi Kredit", nil),
			FileBaseName: "Laporan Klaim Asuransi Kredit",
			// Rentang tanggalnya atas `tglproses`, bukan tanggal registrasi.
			Uses: FilterDateRange,
			Source: Source{
				Activity: "PNCReportKlaimKredit_act",
				SQLRule:  []string{"GetDataKlaimAsuransiKredit", "GetClientNameAutoKlaim"},
			},
			Availability: Ready(),
			// Tiga kolom saja — dan itu memang seluruh isinya di export.
			variants: []variant{
				always("3 kolom", colsKlaimAsuransiKredit),
			},
		},
		{
			Code:         CodeKlaimPerBisnis,
			Title:        "REPORT KLAIM PER BISNIS",
			Group:        GroupLiniBisnis,
			Actions:      oneAction("Export Data", nil),
			FileBaseName: "DATA KLAIM",
			// SATU-SATUNYA panel yang memakai autocomplete "Bisnis", dan ia WAJIB diisi:
			// `BrowseDataClaimBusiness` menyaring `TempLaporan.CountryID` tanpa cabang
			// "bila kosong". Dibiarkan kosong, kuerinya membandingkan dengan string
			// kosong dan berkasnya keluar tanpa satu baris pun — tanpa satu pun tanda
			// bahwa yang kurang adalah isiannya.
			Uses: FilterBusinessCode,
			Source: Source{
				Activity: "ExportDataClaimBusiness",
				SQLRule:  []string{"BrowseDataClaimBusiness"},
			},
			Availability: Ready(),
			variants: []variant{
				always("14 kolom", colsKlaimPerBisnis),
			},
		},
		{
			Code:  CodeKlaimTraveloka,
			Title: "REPORT KLAIM TRAVELOKA",
			Group: GroupLiniBisnis,
			// Panel ini dan PegiPegi berbagi satu activity, dibedakan oleh `tipe`, yang
			// di dalam activity menjadi potongan penyaring nomor polis:
			//
			//	TRVLK  AND a.nopolis LIKE '%T%'
			//	PEGI   and a.nopolis like '122N%'
			Actions:      oneAction("Export Data Traveloka", map[string]string{"tipe": "TRVLK"}),
			FileBaseName: "DATA KLAIM TRAVELOKA",
			Uses:         FilterDateRange,
			Source: Source{
				Activity: "ExportDataClaimTravel",
				SQLRule:  []string{"BrowseDataClaimTraveloka"},
			},
			Availability: Ready(),
			variants: []variant{
				always("6 kolom", colsKlaimTravel),
			},
		},
		{
			Code:         CodeKlaimPegiPegi,
			Title:        "REPORT KLAIM PEGIPEGI",
			Group:        GroupLiniBisnis,
			Actions:      oneAction("Export Data PegiPegi", map[string]string{"tipe": "PEGI"}),
			FileBaseName: "DATA KLAIM PEGIPEGI",
			Uses:         FilterDateRange,
			Source: Source{
				Activity: "ExportDataClaimTravel",
				SQLRule:  []string{"BrowseDataClaimTraveloka"},
			},
			Availability: Ready(),
			variants: []variant{
				always("6 kolom", colsKlaimTravel),
			},
		},
	}
}
