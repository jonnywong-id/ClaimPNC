package reportklaim

// Kode laporan kelompok Operasional.
const (
	CodeMitra           Code = "mitra"
	CodeProduksiKlaimPA Code = "produksi-klaim-pa"
	CodeCompliance      Code = "compliance"
	CodeAdjuster        Code = "adjuster"
	CodeKomunikasiKlaim Code = "komunikasi-klaim"
)

// catalogOperasional menyusun kelima panel kelompok Operasional.
//
// Yang menyatukannya: kelimanya melaporkan CARA KERJA penanganan klaim — siapa yang
// mengerjakan, berapa lama, dan apa yang dipercakapkan — bukan nilai uang klaimnya.
func catalogOperasional() []Report {
	return []Report{
		{
			Code:         CodeMitra,
			Title:        "REPORT MITRA",
			Group:        GroupOperasional,
			Actions:      oneAction("Export Data Mitra", nil),
			FileBaseName: "Laporan Mitra",
			Uses:         FilterDateRange,
			Source: Source{
				Activity: "PNCMitraReport_Act",
				SQLRule:  []string{"ExportDetailMitraReport"},
			},
			// # Terhalang, dan DB Link di sini MENENTUKAN BARIS — bukan sekadar kolom
			//
			// `ExportDetailMitraReport` menggabung `general.lst_mitra@asmd` sebagai INNER
			// JOIN: `a.userassign = b.login_aplikasi`. Gabungan itu bukan pelengkap kolom
			// melainkan PENYARING — ia membatasi laporan pada petugas yang terdaftar
			// sebagai mitra.
			//
			// Di modul lain, kolom yang bersumber DB Link cukup dikosongkan dan ditandai
			// (`inboxautoclaim`, "No Ref Bank"). Perlakuan itu TIDAK berlaku di sini:
			// membuang gabungannya akan memasukkan seluruh petugas ke dalam laporan
			// produktivitas mitra. Berkasnya tetap terbit, angkanya tetap masuk akal, dan
			// isinya bukan yang diminta — kelas cacat yang paling sulit ketahuan.
			//
			// # Penghalangnya BERUBAH pada 2026-09-24, dan belum hilang
			//
			// Work Owner memutuskan hari itu: yang berupa sub-query tetap memakai DB Link,
			// **selain itu memakai koneksi langsung** ke basis datanya — dikonfigurasi per
			// portal lewat `ANEKA_<PORTAL_ALIAS>_*` (lihat keputusan-implementasi.md §49).
			//
			// Gabungan di laporan ini BUKAN sub-query, sehingga ia masuk golongan kedua:
			// daftar mitra dibaca lewat koneksi kedua, lalu dipakai menyaring. Jalurnya
			// sudah tersedia di konfigurasi dan perakitan; kuerinya belum ditulis.
			//
			// Satu hal lagi yang harus diselesaikan bersamaan: kolom "Atasan" di kueri
			// aslinya adalah **nama orang yang ditulis sebagai literal** di dalam SQL.
			// Ia salah satu dari 24 Operator ID hardcode yang `D-15` haruskan menjadi
			// master data, dan tidak ada sumber penggantinya di export.
			Availability: Blocked(
				"Daftar mitra dibaca lewat koneksi kedua portal (ANEKA_<PORTAL_ALIAS>_*); "+
					"kuerinya belum ditulis. Kolom \"Atasan\" juga masih berupa nama orang yang "+
					"ditulis langsung di dalam SQL dan belum punya master penggantinya.",
				"R-03",
			),
			variants: []variant{
				always("13 kolom; kolom bersumber DB Link dikosongkan (R-03)", colsMitra),
			},
		},
		{
			Code:         CodeProduksiKlaimPA,
			Title:        "REPORT PRODUKSI KLAIM PA",
			Group:        GroupOperasional,
			Actions:      oneAction("Export Data Produksi Klaim PA", nil),
			FileBaseName: "Laporan Produksi Klaim PA",
			Uses:         FilterDateRange,
			Source: Source{
				Activity: "ReportProduksiPA_act",
				// Lima kueri pencacah, satu per kelompok risiko. Ia bukan satu kueri
				// dengan lima kolom: tiap kelompok dihitung terpisah lalu disandingkan.
				SQLRule: []string{
					"CountCoverageID",
					"CountCoverageResikoB",
					"CountCoverageResikoD",
					"CountCoverageResikoMC",
					"CountCoverageResikoLainnya",
				},
			},
			Availability: Ready(),
			variants: []variant{
				always("15 kolom", colsProduksiPA),
			},
		},
		{
			Code:         CodeCompliance,
			Title:        "REPORT COMPLIANCE",
			Group:        GroupOperasional,
			Actions:      oneAction("Export Data Compliance", nil),
			FileBaseName: "Laporan Compliance",
			// Radio "Status Compliance" dipakai HANYA oleh panel ini — di seluruh
			// harness, `TempAdjComp.ComplianceStatus` tidak muncul di panel lain mana pun.
			//
			// Lini bisnisnya TIDAK dapat dipilih: activity-nya memasang
			// `Param.GroupPanel = "002"` sebagai nilai tetap. Laporan ini memang laporan
			// Personal Accident saja, dan dropdown "Treaty" tidak berpengaruh padanya.
			Uses: FilterDateRange | FilterComplianceStatus,
			Source: Source{
				Activity: "PNCComplianceReport_Act",
				SQLRule:  []string{"BroswseKlaimByRegisterDate"},
			},
			// # Terhalang: isinya hidup di klipboard Pega, bukan di tabel
			//
			// Activity-nya membuka setiap klaim satu per satu (`Obj-Open-By-Handle`) lalu
			// membaca `TempWorkPage.ClaimData.ComplianceList(<LAST>)` — daftar pemeriksaan
			// kepatuhan yang tersimpan sebagai properti objek kerja Pega. Tujuh dari 12
			// kolomnya berasal dari sana, termasuk isi catatan compliance, tanggalnya, dan
			// statusnya — yang justru menjadi penyaring laporan ini.
			//
			// Properti klipboard yang tidak dioptimasi TIDAK punya kolom basis data;
			// pencarian ke seluruh `RDB List/` dan `Database/` tidak menemukan satu pun
			// padanannya. Ini keadaan yang sama dengan sembilan isian pada modul RCL/PUCL:
			// bukan kolom yang belum ditemukan, melainkan kolom yang memang tidak ada.
			//
			// Menjalankannya tetap akan menghasilkan berkas — berisi kolom klaim yang
			// terbaca dan tujuh kolom kosong, disaring oleh status yang tidak dapat
			// dibaca. Berkas seperti itu lebih buruk daripada tidak ada: ia tampak sah.
			Availability: Blocked(
				"Isi pemeriksaan compliance tersimpan sebagai properti klipboard Pega, bukan "+
					"sebagai kolom basis data. Menunggu properti itu diekspos sebagai kolom, atau "+
					"modul Compliance memiliki tabelnya sendiri.",
				"R-16",
			),
			variants: []variant{
				always("12 kolom", colsCompliance),
			},
		},
		{
			Code:         CodeAdjuster,
			Title:        "REPORT ADJUSTER",
			Group:        GroupOperasional,
			Actions:      oneAction("Export Data Adjuster", nil),
			FileBaseName: "Laporan Adjuster",
			Uses:         FilterDateRange,
			Source: Source{
				Activity: "PNCAdjusterReport_Act",
				// Kosong dengan sengaja: laporan ini TIDAK memakai Connect-SQL sama
				// sekali. Ia memanggil `pxRetrieveReportData` atas dua Report Definition.
				SQLRule: nil,
			},
			// # Terhalang: kedua Report Definition-nya tidak ada di export
			//
			// `InboxSurveyClose_rd` dan `InboxInternalSurveyClose_rd` — keduanya dirujuk
			// `PNCAdjusterReport_Act`, dan tidak satu pun ada di antara 57 Report
			// Definition yang dikirim. Tanpa keduanya, tidak diketahui tabel mana yang
			// dibaca, penyaring apa yang berlaku, dan urutan barisnya bagaimana.
			//
			// Ia contoh langsung `R-16`: tujuh tipe rule tidak diaudit `19-GAP` sama
			// sekali, dan Report Definition salah satunya.
			Availability: Blocked(
				"Report Definition InboxSurveyClose_rd dan InboxInternalSurveyClose_rd tidak ada "+
					"di export Pega. Menunggu export ulang berbasis Product rule.",
				"R-16",
			),
			variants: []variant{
				always("10 kolom", colsAdjuster),
			},
		},
		{
			Code:  CodeKomunikasiKlaim,
			Title: "REPORT DATA KOMUNIKASI KLAIM",
			Group: GroupOperasional,
			// Label tombolnya seluruhnya huruf besar dan tidak berawalan "Export",
			// menyimpang dari 28 tombol lainnya. Disalin apa adanya.
			Actions:      oneAction("REPORT KOMUNIKASI KLAIM", nil),
			FileBaseName: "Report Inbox Komunikasi",
			// `ReportDataInboxKomunikasi_Klaim` tidak menerima satu pun parameter — ia
			// mengambil SELURUH percakapan. Tidak ada isian yang aktif pada kartunya.
			Uses: 0,
			Source: Source{
				Activity: "PNCReportDataKominukasiAdjusterKlaim",
				SQLRule:  []string{"ReportDataInboxKomunikasi_Klaim"},
			},
			Availability: Ready(),
			variants: []variant{
				always("7 kolom", colsKomunikasiKlaim),
			},
		},
	}
}
