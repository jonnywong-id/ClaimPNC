package dashboardclaim

import "strings"

// Grup adalah grup akses pengguna — `POOLDATA.M_LOGIN_GROUP_PNC.GROUP_ID`.
//
// Namanya sama dengan access group Pega TANPA awalan `GCNMFW:` — keputusan Work Owner
// 2026-10-12. Jadi `GCNMFW:PncPICTeknik` di rule Pega tertulis `PncPICTeknik` di sini.
type Grup string

// Kelima grup yang boleh MEMBUKA CASE dari Dashboard Claim.
//
// # Dari mana daftar ini berasal
//
// `Section/DashboardClaimShow_Sec1-Section.xml` memasang DUA aksi pada sel nomor klaim:
//
//	openAssignment                      membuka klaim untuk DIKERJAKAN
//	showHarness ViewTempDetailClaim     popup rincian, BACA SAJA
//
// Pemisahnya When rule `IsManagerPNC`, yang isinya dibaca dari `pyParametersParamValue`
// — bukan dari `pyConditionString`, yang pada rule ini berbunyi "[Double click to add
// condition]" dan sama sekali tidak mencerminkan isinya:
//
//	AccessGroup.pyAccessGroup = "GCNMFW:Administrators"
//	                          = "GCNMFW:CaseManager"
//	                          = "GCNMFW:PncManagerAdmin"
//	                          = "GCNMFW:PNCKomiteTeknik"
//	OperatorID.pyUserIdentifier = "RATNAGUSNITASARI"
//
// Pada layar Outstanding syaratnya diperluas satu grup lagi:
//
//	IsManagerPNC || AccessGroup.pyAccessGroup == 'GCNMFW:PncPICTeknik'
//
// # Dua hal yang BERBEDA dari Pega, dan keduanya diputuskan
//
//  1. `Administrators` menjadi **IT** — keputusan Work Owner 2026-10-12. Di `M_OTORISASI_PNC`
//     memang `IT` yang berhak atas MENU 73, sedangkan `Administrators` tidak ada di sana
//     sama sekali.
//
//  2. Nama orang yang tertanam di rule (`RATNAGUSNITASARI`) **TIDAK dibawa** — `D-15`
//     menetapkan nilai bisnis tidak boleh di-hardcode, dan nama itu satu dari 24 Operator ID
//     yang harus hilang. Akibatnya orang tersebut kehilangan akses sunting yang hari ini ia
//     punya, KECUALI grupnya sudah termasuk salah satu di bawah.
//
//     Itu selisih terhadap Pega yang disengaja, dan dicatat supaya tidak terbaca sebagai bug
//     saat uji kesetaraan.
//
//  3. `Administrators` ikut diterima sejak 2026-10-12, berdampingan dengan `IT`. Work Owner
//     menyatakan keduanya satu hal yang sama, dan pengukuran `M_LOGIN_GROUP_PNC` hari itu
//     membuktikan KEDUANYA dipakai — `Administrators` 8 anggota, `IT` 2 anggota. Menyebut
//     `IT` saja mencabut akses 8 orang yang hari ini memegangnya.
var grupBolehBukaCase = map[Grup]bool{
	"PncPICTeknik":    true,
	"PNCKomiteTeknik": true,
	"CaseManager":     true,
	"PncManagerAdmin": true,
	"IT":              true,
	"Administrators":  true,
}

// BolehBukaCase menjawab apakah pengguna dengan grup-grup ini boleh membuka klaim untuk
// dikerjakan, bukan sekadar melihat rinciannya.
//
// # Satu pengguna punya BANYAK grup
//
// Diukur di DEV 2026-10-12: 22 dari 31 pengguna punya lebih dari satu grup, terbanyak LIMA.
// Jadi syaratnya "memiliki SALAH SATU grup editable", bukan "grupnya adalah". Menulisnya
// sebagai perbandingan tunggal akan salah pada mayoritas pengguna.
//
// # Gagal-ke-aman
//
// Daftar kosong — pengguna yang tidak punya baris di `M_LOGIN_GROUP_PNC` — menjawab FALSE,
// yaitu baca saja.
//
// Dari dua kemungkinan salah, memberi hak sunting kepada yang tidak berhak jauh lebih mahal
// daripada sebaliknya: yang kedua segera dikeluhkan penggunanya, sedangkan yang pertama tidak
// terlihat sampai ada data yang berubah oleh orang yang semestinya tidak boleh mengubahnya.
func BolehBukaCase(grup []Grup) bool {
	for _, g := range grup {
		if grupBolehBukaCase[Grup(strings.TrimSpace(string(g)))] {
			return true
		}
	}
	return false
}

// GrupBolehBukaCase mengembalikan kelima grup itu — dipakai uji dan dokumentasi.
func GrupBolehBukaCase() []Grup {
	keluar := make([]Grup, 0, len(grupBolehBukaCase))
	for g := range grupBolehBukaCase {
		keluar = append(keluar, g)
	}
	return keluar
}
