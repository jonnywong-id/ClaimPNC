package reportklaim

// Kode laporan kelompok Akseptasi & Penyelesaian.
const (
	CodeKasirSudahBayar Code = "kasir-sudah-bayar"
	CodeKasirBelumBayar Code = "kasir-belum-bayar"
	CodePendingLOD      Code = "pending-lod"
	CodeAkseptasi       Code = "akseptasi"
	CodeOSKomite        Code = "os-komite"
	CodeOSBelumKomite   Code = "os-belum-komite"
	CodeKomite          Code = "komite"
)

// Kode aksi pada panel REPORT DATA KOMITE, satu-satunya panel bertombol dua.
const (
	ActionKomiteApprove  = "approve"
	ActionKomiteRejected = "rejected"
)

// catalogPenyelesaian menyusun ketujuh panel kelompok Akseptasi & Penyelesaian.
func catalogPenyelesaian() []Report {
	return []Report{
		{
			Code:  CodeKasirSudahBayar,
			Title: "REPORT DATA KASIR SUDAH BAYAR",
			Group: GroupPenyelesaian,
			// Panel ini dan tetangganya berbagi satu activity, dibedakan HANYA oleh
			// `STSKASIR` — `"1"` sudah bayar, `"0"` belum. Keduanya panel tersendiri di
			// harness, bukan dua tombol pada satu panel, dan karena itu dua entri di sini.
			Actions:      oneAction("Export Data Kasir Sudah Bayar", map[string]string{"STSKASIR": "1"}),
			FileBaseName: "Data Transfer Kasir Sudah Bayar",
			// # Penyaringnya ada, hanya tidak terlihat di kueri
			//
			// `GetBrowseTransferDataToKasir` tampak tanpa penyaring karena ketiganya
			// disisipkan lewat `{ASIS:}` dari halaman `TempDataDetailKasir`, bukan lewat
			// bind. `Activity/ExportTransferKeKasir-Act.xml` mengisi ketiganya:
			//
			//	.City        status bayar — `CLAIM_STATUS_PAID is (not) null`
			//	.Country     rentang tanggal atas TRANSFER_CASHIER_DATE
			//	.UserTeknis  Group Panel menurut pilihan lini bisnis
			//
			// Membaca kuerinya saja akan menyimpulkan laporan ini tidak punya penyaring —
			// dan kartunya akan tampil tanpa satu pun isian, lalu mengunduh seluruh
			// riwayat transfer kasir sejak sistem berdiri.
			Uses: FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "ExportTransferKeKasir",
				SQLRule:  []string{"GetBrowseTransferDataToKasir"},
			},
			Availability: Ready(),
			variants: []variant{
				always("9 kolom", colsKasir),
			},
		},
		{
			Code:         CodeKasirBelumBayar,
			Title:        "REPORT DATA KASIR BELUM BAYAR",
			Group:        GroupPenyelesaian,
			Actions:      oneAction("Export Data Kasir Belum Bayar", map[string]string{"STSKASIR": "0"}),
			FileBaseName: "Data Transfer Kasir Belum Bayar",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "ExportTransferKeKasir",
				SQLRule:  []string{"GetBrowseTransferDataToKasir"},
			},
			Availability: Ready(),
			variants: []variant{
				always("9 kolom", colsKasir),
			},
		},
		{
			Code:         CodePendingLOD,
			Title:        "REPORT DATA PENDING LOD",
			Group:        GroupPenyelesaian,
			Actions:      oneAction("Export Data Pending LOD", nil),
			FileBaseName: "Data Pending LOD",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "ExportDataPendingLOD",
				SQLRule:  []string{"BrowseDataPendingLOD"},
			},
			Availability: Ready(),
			variants: []variant{
				always("11 kolom", colsPendingLOD),
			},
		},
		{
			Code:         CodeAkseptasi,
			Title:        "REPORT AKSEPTASI",
			Group:        GroupPenyelesaian,
			Actions:      oneAction("Export Data Akseptasi", nil),
			FileBaseName: "Laporan Akseptasi Non MBU",
			// Kotak centang di panel ini TIDAK menyaring baris — ia memilih susunan
			// kolom. Lihat FilterDetail.
			Uses: FilterDateRange | FilterDetail,
			Source: Source{
				Activity: "ReportAkseptasiNonMBU",
				SQLRule:  []string{"ReportAkseptasiNonMBU", "GetFileOnPc_link_attachmentGCNM"},
			},
			Availability: Ready(),
			variants: []variant{
				detailVariant("kotak centang dicentang — 42 kolom, susunan rinci", colsAkseptasiRinci),
				always("kotak centang kosong — 26 kolom, susunan ringkas", colsAkseptasiRingkas),
			},
		},
		{
			Code:         CodeOSKomite,
			Title:        "REPORT OS KOMITE",
			Group:        GroupPenyelesaian,
			Actions:      oneAction("Export Data OS Komite", nil),
			FileBaseName: "Report OS Komite Klaim Non MBU",
			Uses:         FilterDateRange | FilterDetail,
			Source: Source{
				Activity: "ReportOSKomiteNonMBU",
				SQLRule:  []string{"GetOSKomiteNonMBU"},
			},
			Availability: Ready(),
			variants: []variant{
				detailVariant("kotak centang dicentang — 43 kolom, susunan rinci", colsOSKomiteRinci),
				always("kotak centang kosong — 27 kolom, susunan ringkas", colsOSKomiteRingkas),
			},
		},
		{
			Code:  CodeOSBelumKomite,
			Title: "REPORT OS BELUM KOMITE",
			Group: GroupPenyelesaian,
			// Label tombolnya berbunyi "Export Data OS Klaim", tidak sejalan dengan judul
			// panelnya. Disalin apa adanya — itu yang dibaca pengguna selama ini.
			Actions:      oneAction("Export Data OS Klaim", nil),
			FileBaseName: "Report OS Belum Komite Klaim Non MBU",
			Uses:         FilterDateRange | FilterDetail,
			Source: Source{
				Activity: "ReportOSBlmKomiteNonMBU",
				SQLRule:  []string{"GetOSBlmKomiteNonMBU"},
			},
			Availability: Ready(),
			variants: []variant{
				detailVariant("kotak centang dicentang — 36 kolom, susunan rinci", colsOSBelumKomiteRinci),
				always("kotak centang kosong — 21 kolom, susunan ringkas", colsOSBelumKomiteRingkas),
			},
		},
		{
			Code:  CodeKomite,
			Title: "REPORT DATA KOMITE",
			Group: GroupPenyelesaian,
			// SATU-SATUNYA panel bertombol dua di seluruh layar. Keduanya memanggil
			// activity yang sama dan dibedakan oleh `statusapprove`.
			//
			// Urutannya dibalik dari harness — di sana "Rejected" tergambar lebih dulu.
			// Approve didahulukan di sini karena ia jalur yang normal; yang ditolak adalah
			// kekecualian. Itu satu-satunya perbedaan, dan tidak menyentuh isi berkas.
			Actions: []Action{
				{Code: ActionKomiteApprove, Label: "Export Data Approve", FixedParam: map[string]string{"statusapprove": "1"}},
				{Code: ActionKomiteRejected, Label: "Export Data Rejected", FixedParam: map[string]string{"statusapprove": "2"}},
			},
			FileBaseName: "Laporan Data Komite",
			Uses:         FilterDateRange | FilterBusinessLine,
			Source: Source{
				Activity: "PNCReportDataKomites_act",
				// Sama seperti panel Close Klaim: nama kueri yang berakhiran NONMBU yang
				// memastikan susunan kolom mana yang berlaku.
				SQLRule: []string{"ExportDataCloseKlaim", "ExportDataKomitesKlaimNONMBU"},
			},
			Availability: Ready(),
			variants: []variant{
				nonMBUVariant("Non-MBU — 80 kolom, kueri ExportDataKomitesKlaimNONMBU", colsKomite),
				always("selain Non-MBU — 11 kolom, kueri ExportDataCloseKlaim", colsKomiteRingkas),
			},
		},
	}
}
