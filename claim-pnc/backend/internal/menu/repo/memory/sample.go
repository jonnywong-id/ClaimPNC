package memory

import "claim-pnc/internal/menu"

// SampleItems adalah isi POOLDATA.M_MENU_APLIKASI_PNC untuk APP_DESC = 'CLAIM PNC'.
//
// DISALIN APA ADANYA dari `Database/m_menu_aplikasi_pnc.csv` yang diterima bersama
// `CREATE_MENU.sql` — bukan susunan sendiri. Itu membuat menu yang terlihat saat
// pengembangan sama persis dengan menu produksi, termasuk keanehannya.
//
// Dua keanehan yang sengaja ikut disalin, karena keduanya nyata:
//
//   - MENU_ID 83 "Report Adjuster" adalah daun TANPA MENU_PROGRAM. Ia menempel pada
//     kelompok REPORT tetapi tidak menuju layar mana pun.
//   - Sembilan MENU_PROGRAM menunjuk harness yang TIDAK ADA di export Pega:
//     DataMemberReas, DetailMasterPasalAI, InboxCloseClaim_Harness,
//     InboxOutstanding_Harness, InboxRequestSalvage, InboxServiceCenter,
//     LostAdjuster_harness, PNCViewClaim, dan ReportProduksiPA_harnes. Ini memperjelas
//     `K-33`, yang menyebut sebagian harness target tidak ikut diekspor.
//
// Urutan di bawah mengikuti MENU_SEQUENCE.
func SampleItems() []menu.Item {
	return []menu.Item{
		{ID: 1, Description: "MASTER", Program: "", ParentID: nil, Sequence: 1},
		{ID: 2, Description: "INBOX", Program: "", ParentID: nil, Sequence: 2},
		{ID: 3, Description: "VIEW", Program: "", ParentID: nil, Sequence: 3},
		{ID: 4, Description: "REPORT", Program: "", ParentID: nil, Sequence: 4},
		{ID: 11, Description: "Master Status Klaim", Program: "StatusClaimInbox", ParentID: parent(1), Sequence: 1101},
		{ID: 12, Description: "Master Rekening", Program: "MasterRekening", ParentID: parent(1), Sequence: 1102},
		{ID: 13, Description: "Master PIC Teknik", Program: "UserTeknisInbox", ParentID: parent(1), Sequence: 1103},
		{ID: 14, Description: "Master Tipe Surveyors", Program: "SurveyorsInbox", ParentID: parent(1), Sequence: 1104},
		{ID: 15, Description: "Master Surveyors", Program: "DetailSurveyorsInbox", ParentID: parent(1), Sequence: 1105},
		{ID: 16, Description: "Master Recovery", Program: "MasterRecovery", ParentID: parent(1), Sequence: 1106},
		{ID: 17, Description: "Master Masking", Program: "MasterProteksiVisibilityData", ParentID: parent(1), Sequence: 1107},
		{ID: 18, Description: "Master Dominan Factor", Program: "DetailDominanFactor", ParentID: parent(1), Sequence: 1108},
		{ID: 19, Description: "Master XOL", Program: "DetailMasterXOL", ParentID: parent(1), Sequence: 1109},
		{ID: 20, Description: "Master Penyebab Kerugian", Program: "CauseOfLossInbox", ParentID: parent(1), Sequence: 1110},
		{ID: 21, Description: "Master COL SIMAS ONLNE", Program: "CauseOfLossInboxSimasOnline", ParentID: parent(1), Sequence: 1111},
		{ID: 22, Description: "Master Dokumen Travel", Program: "BrowseMasterDocumentTravel_Harness", ParentID: parent(1), Sequence: 1112},
		{ID: 23, Description: "Master Status Progress 1", Program: "StatusProgress", ParentID: parent(1), Sequence: 1113},
		{ID: 24, Description: "Master Status Progress 2", Program: "StatusProgress2", ParentID: parent(1), Sequence: 1114},
		{ID: 25, Description: "Master Penolakan Klaim", Program: "PNC_MasterTolakKlaim", ParentID: parent(1), Sequence: 1115},
		{ID: 26, Description: "Master Auto Claim", Program: "AutoKlaim", ParentID: parent(1), Sequence: 1116},
		{ID: 27, Description: "Master Pasal Kerugian", Program: "DetailMasterPasalRejected", ParentID: parent(1), Sequence: 1117},
		{ID: 28, Description: "Master Bengkel", Program: "BengkelHE", ParentID: parent(1), Sequence: 1118},
		{ID: 29, Description: "Master Supplier", Program: "MasterSupplier", ParentID: parent(1), Sequence: 1119},
		{ID: 30, Description: "Master Panel", Program: "MasterPanel_HE", ParentID: parent(1), Sequence: 1120},
		{ID: 31, Description: "Master Sparepart", Program: "SparePart_HE", ParentID: parent(1), Sequence: 1121},
		{ID: 32, Description: "Master Grouping Sparepart", Program: "GroupingSparePart_HE", ParentID: parent(1), Sequence: 1122},
		{ID: 33, Description: "Master Kategori Sparepart", Program: "GCNMCatSparepart", ParentID: parent(1), Sequence: 1123},
		{ID: 34, Description: "Master Tipe Sparepart", Program: "GCNMMasterSparepartType", ParentID: parent(1), Sequence: 1124},
		{ID: 35, Description: "Master Reas", Program: "DataMemberReas", ParentID: parent(1), Sequence: 1125},
		{ID: 36, Description: "Master Pasal AI", Program: "DetailMasterPasalAI", ParentID: parent(1), Sequence: 1126},
		{ID: 37, Description: "Master Login", Program: "MasterLoginSurvey", ParentID: parent(1), Sequence: 1127},
		{ID: 38, Description: "Detail Penyebab Kerugian", Program: "DetailCauseOfLoss", ParentID: parent(1), Sequence: 1128},
		{ID: 39, Description: "Daftar Detail Dokumen Travel", Program: "ListDocumentTravel", ParentID: parent(1), Sequence: 1129},
		{ID: 40, Description: "Daftar Tipe Dokumen", Program: "ListDocumentTypeInbox", ParentID: parent(1), Sequence: 1130},
		{ID: 41, Description: "Daftar Detail Tipe Dokumen", Program: "ListDetTypeDocument", ParentID: parent(1), Sequence: 1131},
		{ID: 42, Description: "Daftar Tipe Dokumen Bisnis", Program: "DetTypeDocumenBisnis", ParentID: parent(1), Sequence: 1132},
		{ID: 43, Description: "Daftar Objek Dokumen", Program: "ListDocumentObject", ParentID: parent(1), Sequence: 1133},
		{ID: 44, Description: "Inbox PLA, DLA, Pre DLA", Program: "InboxPLA_harness", ParentID: parent(2), Sequence: 1134},
		{ID: 45, Description: "Inbox PLA DLA", Program: "InboxPLADLA", ParentID: parent(2), Sequence: 1135},
		{ID: 46, Description: "Inbox Service Center", Program: "InboxServiceCenter", ParentID: parent(2), Sequence: 1136},
		{ID: 47, Description: "Inbox Compliance", Program: "inboxCompliance_Harness", ParentID: parent(2), Sequence: 1137},
		{ID: 48, Description: "Inbox Investigator", Program: "InboxInvestigator_Harness", ParentID: parent(2), Sequence: 1138},
		{ID: 49, Description: "Inbox Receive TKA", Program: "InboxTKA_Harness", ParentID: parent(2), Sequence: 1139},
		{ID: 50, Description: "My Work", Program: "InboxSurvey_Harness", ParentID: parent(2), Sequence: 1140},
		{ID: 51, Description: "My Inbox", Program: "InboxRegister_Harness", ParentID: parent(2), Sequence: 1141},
		{ID: 52, Description: "Inbox Komite", Program: "InboxKomite_Harness", ParentID: parent(2), Sequence: 1142},
		{ID: 53, Description: "Inbox XOL", Program: "Inbox_XOL_Harness", ParentID: parent(2), Sequence: 1143},
		{ID: 54, Description: "Inbox Claim Treaty Prop", Program: "InboxClaimTreaty_Harness", ParentID: parent(2), Sequence: 1144},
		{ID: 55, Description: "Inbox Claim Treaty Non Prop", Program: "InboxClaimNonProp_Harness", ParentID: parent(2), Sequence: 1145},
		{ID: 56, Description: "Inbox Manager Receive / PUCL", Program: "ReceiveDoucument_Harness", ParentID: parent(2), Sequence: 1146},
		{ID: 57, Description: "Inbox Manager Admin", Program: "InboxManagerAdmin_Harness", ParentID: parent(2), Sequence: 1147},
		{ID: 58, Description: "Inbox Manager", Program: "UserInbox_Harness", ParentID: parent(2), Sequence: 1148},
		{ID: 59, Description: "Inbox Close Claim", Program: "InboxCloseClaim_Harness", ParentID: parent(2), Sequence: 1149},
		{ID: 60, Description: "Inbox Analyst Doctor", Program: "inboxAnalystDoctor_Harness", ParentID: parent(2), Sequence: 1150},
		{ID: 61, Description: "Inbox RCL/PUCL", Program: "RCLPUCL_Harness", ParentID: parent(2), Sequence: 1151},
		{ID: 62, Description: "Inbox RCL", Program: "RCL_Harness", ParentID: parent(2), Sequence: 1152},
		{ID: 63, Description: "Inbox Admin", Program: "PNCInboxAdmin", ParentID: parent(2), Sequence: 1153},
		{ID: 64, Description: "Inbox Laporan Klaim", Program: "InboxRCVApp_Harness", ParentID: parent(2), Sequence: 1154},
		{ID: 65, Description: "Inbox Progress Claim", Program: "ProgressClaim_Harness", ParentID: parent(2), Sequence: 1155},
		{ID: 66, Description: "Inbox Open Protection", Program: "InputProtection_Harness", ParentID: parent(2), Sequence: 1156},
		{ID: 67, Description: "Input Req Protection", Program: "InputReqProtection_Harness", ParentID: parent(2), Sequence: 1157},
		{ID: 68, Description: "Inbox Auto Claim", Program: "InboxAutoClaim", ParentID: parent(2), Sequence: 1158},
		{ID: 69, Description: "Inbox OS Claim per Cabang", Program: "OutstandingKlaimperCabang_Harness", ParentID: parent(2), Sequence: 1159},
		{ID: 70, Description: "Inbox Komunikasi Cabang", Program: "InboxKomunikasiCabang", ParentID: parent(2), Sequence: 1160},
		{ID: 71, Description: "Inbox Salvage", Program: "InboxSalvage", ParentID: parent(2), Sequence: 1161},
		{ID: 72, Description: "Inbox Banding Harga Salvage", Program: "InboxRequestSalvage", ParentID: parent(2), Sequence: 1162},
		{ID: 73, Description: "Dashboard Claim", Program: "DashboardClaim_Harness", ParentID: parent(2), Sequence: 1163},
		{ID: 74, Description: "Case Study Claim", Program: "PNCStudyClaim", ParentID: parent(2), Sequence: 1164},
		{ID: 75, Description: "View Claim", Program: "PNCViewClaim", ParentID: parent(3), Sequence: 1165},
		{ID: 76, Description: "View History Claim", Program: "PNCSearchKlaim", ParentID: parent(3), Sequence: 1166},
		{ID: 77, Description: "Archive Dokumen Klaim", Program: "PNCArchiveDokumen", ParentID: parent(3), Sequence: 1167},
		{ID: 78, Description: "Monitoring SLINK OJK", Program: "MonitoringSLINKOJK", ParentID: parent(2), Sequence: 1168},
		{ID: 79, Description: "Inbox Outstanding", Program: "InboxOutstanding_Harness", ParentID: parent(2), Sequence: 1169},
		{ID: 80, Description: "Lost Adjuster", Program: "LostAdjuster_harness", ParentID: parent(2), Sequence: 1170},
		{ID: 81, Description: "View Policy", Program: "ViewPolis", ParentID: parent(3), Sequence: 1171},
		{ID: 82, Description: "Laporan Hasil AI", Program: "Har_LaporanHasilAI", ParentID: parent(4), Sequence: 1172},
		{ID: 83, Description: "Report Adjuster", Program: "", ParentID: parent(4), Sequence: 1173},
		{ID: 84, Description: "Report KPI PNC", Program: "ReportKPIHarness", ParentID: parent(4), Sequence: 1174},
		{ID: 85, Description: "Report Klaim", Program: "PNCTATReport", ParentID: parent(4), Sequence: 1175},
		{ID: 86, Description: "Report Produksi Klaim PA", Program: "ReportProduksiPA_harnes", ParentID: parent(4), Sequence: 1176},
	}
}

// SampleGroups adalah isi POOLDATA.M_LOGIN_GROUP_PNC.
//
// Disalin dari `Database/m_login_group_pnc.csv`, yang saat diterima memuat SATU baris.
// Sedikitnya isi itu bukan kelalaian pembacaan — tabelnya memang baru diisi contoh.
func SampleGroups() map[string][]string {
	return map[string][]string{
		"JONNY": {"IT"},
	}
}

// SampleGrants adalah isi POOLDATA.M_OTORISASI_PNC.
//
// Disalin dari `Database/m_otorisasi_pnc.csv`. Dua subjek, dan pembagiannya menguji dua
// jalur yang berbeda sekaligus:
//
//	IT     MENU_ID 11..81 — seluruh MASTER, INBOX, dan VIEW; TANPA satu pun kelompoknya
//	JONNY  MENU_ID 4 dan 82..86 — kelompok REPORT beserta anaknya
//
// Bahwa group `IT` tidak diberi izin atas MENU_ID 1..4 adalah alasan langsung aturan
// "kelompok tampil bila ada anaknya yang tampil" di menu.BuildTree. Menuntut kelompok
// punya baris izin sendiri akan menghapus seluruh menu group IT.
func SampleGrants() map[string][]int {
	return map[string][]int{
		"IT": {
			11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22,
			23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34,
			35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46,
			47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
			59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
			71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81,
		},
		"JONNY": {
			4, 82, 83, 84, 85, 86,
		},
	}
}
