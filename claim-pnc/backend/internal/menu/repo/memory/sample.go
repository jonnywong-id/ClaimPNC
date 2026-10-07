package memory

import (
	_ "embed"

	"claim-pnc/internal/menu"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

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
//
//   - ENAM MENU_PROGRAM menunjuk harness yang TIDAK ADA di export Pega:
//     InboxCloseClaim_Harness, InboxOutstanding_Harness, InboxRequestSalvage,
//     LostAdjuster_harness, PNCViewClaim, dan ReportProduksiPA_harnes. Ini memperjelas
//     `K-33`, yang menyebut sebagian harness target tidak ikut diekspor.
//
//     Angkanya SEMBILAN sampai 2026-09-22, ketika `DataMemberReas` dan
//     `DetailMasterPasalAI` menyusul masuk, lalu TUJUH sampai 2026-09-28 ketika
//     `InboxServiceCenter` menyusul — beserta empat section tab, sembilan activity, dan
//     delapan rule SQL-nya.
//
//     Harness yang ada TIDAK berarti layarnya dapat dibangun. `DetailMasterPasalAI`
//     (MENU_ID 36) membuktikannya: harness-nya hanya menggambar judul, lalu menyerahkan
//     seluruh isinya ke sebuah Section yang sampai hari ini masih hilang — sehingga
//     tabel dan kolom yang dikelolanya tetap tidak diketahui. Lihat
//     `docs/permintaan-artefak-pega.md` §1.
//
// Urutan di bawah mengikuti MENU_SEQUENCE.
func SampleItems() []menu.Item { return sampledata.Must[[]menu.Item](sampleJSON, "SampleItems") }

// SampleGroups adalah isi POOLDATA.M_LOGIN_GROUP_PNC.
//
// Disalin dari `Database/m_login_group_pnc.csv`, yang saat diterima memuat SATU baris.
// Sedikitnya isi itu bukan kelalaian pembacaan — tabelnya memang baru diisi contoh.
func SampleGroups() map[string][]string {
	return sampledata.Must[map[string][]string](sampleJSON, "SampleGroups")
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
	return sampledata.Must[map[string][]int](sampleJSON, "SampleGrants")
}
