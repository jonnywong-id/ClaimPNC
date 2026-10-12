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
			// # Penghalangnya HILANG pada 2026-09-30
			//
			// Kueri koneksi keduanya sudah ditulis (`report_mitra_logins`), dan daftar
			// mitra kini dipakai menyaring baris di Go — lihat `mitraKeep`. Laporannya
			// DITOLAK bila daftar itu tidak dapat dibaca, sehingga kekhawatiran di atas
			// (seluruh petugas ikut masuk) tidak dapat terjadi diam-diam.
			//
			// Kolom "Atasan" tetap berupa nama orang yang ditulis di dalam SQL. Itu bukan
			// lagi penghalang: Work Owner menetapkan pada 2026-09-25 bahwa yang sudah
			// sesuai Pega dibiarkan apa adanya, termasuk hardcode-nya. Ia tetap tercatat
			// sebagai salah satu dari 24 Operator ID yang `D-15` haruskan menjadi master.
			Availability: Ready(),
			variants: []variant{
				always("13 kolom", colsMitra),
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
			// Label tombolnya MENYEBUTKAN keterbatasannya, dan itu disengaja.
			//
			// Laporan ini berjalan tanpa penyaring Status Compliance — properti penyaringnya
			// belum punya kolom basis data. Akibatnya berkasnya memuat SELURUH klaim PA pada
			// periode itu, bukan hanya yang berstatus tertentu.
			//
			// Perbedaan sebesar itu tidak boleh hanya tertulis di dokumen. Ia ditaruh di
			// tempat yang pasti terbaca: tombol yang ditekan pengguna.
			Actions:      oneAction("Export Data Compliance (tanpa penyaring status)", nil),
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
			// # Penghalangnya kini DIKETAHUI PERSIS (2026-09-30)
			//
			// Keenam rule Property-nya sudah diterima dan dibaca. Hasilnya menutup satu
			// pertanyaan dan memastikan satu lagi:
			//
			//   TERJAWAB  daftar pilihan "Status Compliance" — `PilihanCompliance`
			//             ber-`pyTableOption = PromptList` dengan empat nilai, kini ada
			//             di `ComplianceStatusOptions`.
			//
			//   PASTI     tidak satu pun dari keenamnya punya `pyColumnInclusion`, dan
			//             kueri katalog DBA atas `PC_ASM_FW_GCNMFW_WORK` mengembalikan nol
			//             kolom ber-`%COMPLIANCE%` maupun `%POSTAUDIT%`. Ditambah
			//             pemeriksaan terakhir: kolom `SURPLUS1` dan `SURPLUS2` ternyata
			//             memuat `isComplianceTransfer` dan `IsTransferAnalisator` — bukan
			//             statusnya. Tidak ada kolom yang menyimpannya.
			//
			// Jadi penghalangnya bukan lagi "belum ketemu kolomnya", melainkan "kolomnya
			// memang belum dibuat". Sebabnya disebut di pesan di bawah dengan nama rule
			// dan tindakan yang persis, supaya kartunya menjadi perintah kerja — bukan
			// keterangan yang masih perlu ditafsirkan.
			// # Dibuka pada 2026-09-30, DENGAN keterbatasan yang dinyatakan
			//
			// Sepuluh dari 12 kolomnya ternyata dapat dibaca dari tabel biasa. Yang tidak:
			// dua kolom komentar dan penyaring statusnya — ketiganya properti klipboard
			// yang belum dioptimasi, dipastikan tiga kali (lihat report_compliance).
			//
			// Menahannya tetap tertutup berarti menahan sepuluh kolom yang sebenarnya siap,
			// menunggu satu tindakan di sistem lain. Menjalankannya diam-diam berarti
			// berkas yang barisnya jauh lebih banyak daripada Pega tanpa ada yang tahu.
			//
			// Jalan tengahnya: dijalankan, dan keterbatasannya ditaruh di LABEL TOMBOL —
			// tempat yang pasti terbaca sebelum berkasnya terunduh.
			//
			// Yang memulihkannya sepenuhnya tetap satu tindakan: "Optimize for Reporting"
			// pada `PilihanCompliance` (kelas `ASM-FW-GCNMFW-Data-ClaimData`). Kedua kolom
			// komentar butuh Declare Index, bukan optimize — keduanya ada di dalam page
			// list `ComplianceList`, dan satu klaim dapat punya banyak barisnya.
			Availability: Ready(),
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
			// # Lepas pada 2026-09-30 — DIREKONSTRUKSI, bukan disalin
			//
			// Kedua Report Definition-nya tetap tidak ada di export (`R-16`). Yang berubah
			// adalah cara menyusunnya: kesepuluh kolomnya sudah lengkap di
			// `PNCAdjusterReport_Act`, dan penyaringnya direkonstruksi dari
			// `RDB List/BrowseInternalSurveyor-SQL.xml` — inbox survei yang BELUM selesai,
			// dibaca terbalik.
			//
			// Daftar kolom `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang diterima dari DBA pada
			// 2026-09-30 yang menutup sisanya: `SURVEYDATE_1`, `KETERANGAN_1`,
			// `ADJUSTERSTATUS_1`, `USERTEKNIS_1`, `SURVEYORNAME_1`.
			//
			// Sejak 2026-10-08 objek kerja itu tidak dibaca lagi: sumbernya langkah terakhir
			// per berkas di `T_SURVEYORLIST` ditambah `T_CLAIM_PNC`. Padanan `SURVEYDATE_1`
			// dan `KETERANGAN_1` BELUM terbukti — lihat SUMBER BARU di report_adjuster.
			//
			// TIGA hal masih berupa simpulan dan mengubah ISI laporan — arti "Close",
			// kolom yang disaring periode, dan rumus "Nilai Reserve Klaim ASM". Ketiganya
			// diuraikan di `report_adjuster` pada reportklaim_operasional.sql, dan hanya
			// dapat dipastikan bila kedua Report Definition itu kelak dikirim.
			Availability: Ready(),
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
