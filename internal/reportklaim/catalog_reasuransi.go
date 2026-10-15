package reportklaim

// Kode laporan kelompok PLA / DLA.
const (
	CodePLA           Code = "pla"
	CodeDLA           Code = "dla"
	CodePengirimanPLA Code = "pengiriman-pla"
	CodePengirimanDLA Code = "pengiriman-dla"
)

// catalogReasuransi menyusun keempat panel kelompok PLA / DLA.
//
// Urutan PLA lalu DLA bukan selera: ia urutan pemberitahuan ke koasuransi dan reasuransi
// itu sendiri — PLA atas nilai estimasi, DLA atas nilai akseptasi (`CONTEXT.md`). Panel
// "pengiriman" melaporkan pengirimannya, bukan penerbitannya.
func catalogReasuransi() []Report {
	return []Report{
		{
			Code:         CodePLA,
			Title:        "REPORT DATA PLA",
			Group:        GroupReasuransi,
			Actions:      oneAction("Export Data PLA", nil),
			FileBaseName: "Laporan Data PLA",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataPLA_act",
				SQLRule:  []string{"ExportDataPLA"},
			},
			Availability: Ready(),
			variants: []variant{
				always("12 kolom", colsPLA),
			},
		},
		{
			Code:         CodeDLA,
			Title:        "REPORT DATA DLA",
			Group:        GroupReasuransi,
			Actions:      oneAction("Export Data DLA", nil),
			FileBaseName: "Laporan Data DLA",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataDLA_act",
				SQLRule:  []string{"ExportDataDLA"},
			},
			Availability: Ready(),
			variants: []variant{
				always("16 kolom", colsDLA),
			},
		},
		{
			Code:         CodePengirimanPLA,
			Title:        "REPORT DATA PENGIRIMAN PLA",
			Group:        GroupReasuransi,
			Actions:      oneAction("Export Data Pengiriman PLA", nil),
			FileBaseName: "Laporan Data Pengiriman PLA",
			// # Tanpa penyaring lini bisnis, dan itu MENGOREKSI cacat sistem lama
			//
			// `ExportDataPengirimanPLA` memuat `{ASIS:TempLaporan.UserTeknis}` seperti
			// ketiga tetangganya — tetapi `PNCReportDataPengirimanPLA_act` **tidak
			// pernah mengisi properti itu** dengan potongan penyaring apa pun. Yang
			// tersisip karenanya adalah nilai yang KEBETULAN masih tertinggal di
			// klipboard dari laporan yang dijalankan sebelumnya.
			//
			// Akibatnya di sistem lama: hasil laporan ini bergantung pada laporan apa
			// yang dibuka pengguna sebelum membukanya. Itu bukan perilaku yang ditiru —
			// ia tidak dapat diuji, tidak dapat dijelaskan ke pengguna, dan tidak dapat
			// direproduksi dua kali berturut-turut.
			Uses: FilterDateRange,
			Source: Source{
				Activity: "PNCReportDataPengirimanPLA_act",
				SQLRule:  []string{"ExportDataPengirimanPLA"},
			},
			Availability: Ready(),
			variants: []variant{
				always("12 kolom", colsPengirimanPLA),
			},
		},
		{
			Code:         CodePengirimanDLA,
			Title:        "REPORT DATA PENGIRIMAN DLA",
			Group:        GroupReasuransi,
			Actions:      oneAction("Export Data Pengiriman DLA", nil),
			FileBaseName: "Laporan Data Pengiriman DLA",
			// Berbeda dari ketiga tetangganya, kuerinya TIDAK menerima penyaring lini
			// bisnis — `ExportDataPengirimanDLA` hanya menerima rentang tanggal.
			Uses: FilterDateRange,
			Source: Source{
				Activity: "PNCReportDataPengirimanDLA_act",
				SQLRule:  []string{"ExportDataPengirimanDLA"},
			},
			Availability: Ready(),
			// Judul kolom kedelapan berbunyi "Statur Kirim" di export — salah ketik yang
			// dipertahankan, lihat Column.Header.
			variants: []variant{
				always("9 kolom", colsPengirimanDLA),
			},
		},
	}
}
