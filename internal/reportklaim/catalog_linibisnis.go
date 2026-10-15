package reportklaim

// Kode laporan kelompok Lini Bisnis & Mitra Kerja Sama.
const (
	CodeKlaimHE           Code = "klaim-he"
	CodeRegistSimasOnline Code = "regist-simas-online"
)

// catalogLiniBisnis menyusun kedua panel kelompok Lini Bisnis & Mitra Kerja Sama.
//
// Yang menyatukannya: keduanya menyaring SATU lini tertentu, dan penyaringnya tertanam di
// laporannya — bukan dipilih pengguna lewat dropdown "Treaty".
//
// # Empat panel yang DICABUT 2026-10-13
//
// Kelompok ini sempat berisi enam. Keempat berikut dicabut karena **tidak ada di Pega
// produksi**, dan Work Owner menetapkan migrasi mengikuti produksi:
//
//	REPORT KLAIM ASURANSI KREDIT
//	REPORT KLAIM PER BISNIS
//	REPORT KLAIM TRAVELOKA
//	REPORT KLAIM PEGIPEGI
//
// Keempatnya ADA di `Harness/PNCTATReport-Harness.xml`, ber-`pyVisible = ALWAYS`, lengkap
// dengan activity dan rule SQL-nya. Jadi yang berbeda bukan hak akses melainkan VERSI
// harness-nya: export memuat 28 panel sementara produksi menampilkan 23.
//
// Rinciannya di `catatan-pengembangan.md` §61 dan §62.
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
	}
}
