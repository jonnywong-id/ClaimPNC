package inboxbandinghargasalvage

import "strings"

// Berkas ini memuat SATU aturan, dan aturan itu melanggar `D-15`.
//
// ============================================================================
// APA YANG DILANGGAR, DAN ATAS KEPUTUSAN SIAPA
// ============================================================================
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode, dan nama orang sebagai
// penentu perilaku adalah tepat yang dilarangnya — keempat nama di bawah termasuk 24 Operator
// ID yang `F-4` hapus.
//
// Akibatnya dinyatakan lebih dulu kepada Work Owner pada 2026-09-29, beserta tiga alternatif:
// memakai identitas pemanggil apa adanya, membuang penyaringnya sama sekali, atau menundanya
// sampai `F-4` menyediakan peran. **Work Owner memilih meniru Pega apa adanya.** Keputusan itu
// dihormati dan diterapkan penuh.
//
// Preseden yang sama sudah pernah ditempuh dan tercatat di `keputusan-implementasi.md` §41.15
// (`IsGCNMUser`): keberatan diajukan dengan bukti, Work Owner menegaskan pilihannya, dan
// hasilnya diterapkan tanpa dikurangi.
//
// ============================================================================
// KENAPA SELURUHNYA DIKUMPULKAN DI SATU BERKAS
// ============================================================================
//
// Supaya hari `F-4` selesai, yang perlu dihapus adalah SATU berkas dan satu pemanggilan —
// bukan empat nama yang tersebar di kueri, pencacah, dan uji. Sampai hari itu, berkas ini
// pula satu-satunya tempat yang perlu dibaca untuk mengetahui siapa melihat antrean siapa.
//
// Nama Operator ID BOLEH ditulis lengkap (`D-69`), dan di sini ia memang harus: tanpa namanya,
// tiket `F-4` tidak dapat menunjuk hardcode mana yang dihapus.

// Operator ID yang menentukan perilaku di layar ini.
//
// Keempatnya dibaca dari dua activity, dan keduanya TIDAK sepakat:
//
//	Activity/SetReqSalvage_Act-Act.xml langkah 8             hanya MARIATRIELSA
//	Activity/GCNMCountRequestSalvage_act-Act.xml langkah 5   MARIATRIELSA atau WULANINDRIPAAT
//
// Yang berlaku di sini adalah versi DAFTAR-nya, karena Work Owner memutuskan pencacah
// disamakan dengan daftarnya (lihat PlannedDifferences butir 4). `operatorWulan` karena itu
// tidak ikut dipakai, dan ia tetap dicatat supaya selisih antara kedua activity itu tidak
// hilang dari ingatan saat seseorang membandingkan keduanya kelak.
const (
	// operatorMaria melihat antrean operatorBambang, bukan antreannya sendiri.
	operatorMaria = "MARIATRIELSA"

	// operatorBambang adalah komite yang antreannya diwakilkan, sekaligus komite yang harus
	// memutus LEBIH DULU sebelum operatorDaniel melihat barisnya.
	operatorBambang = "BAMBANGSETIADJIGUNAWAN"

	// operatorDaniel hanya melihat baris yang operatorBambang sudah selesai menanganinya.
	operatorDaniel = "DANIELLISWANDI"

	// operatorWulan TIDAK dipakai — lihat catatan di atas. Ia ada di pencacah Pega saja.
	operatorWulan = "WULANINDRIPAAT"
)

// Reviewer menyatakan antrean SIAPA yang dilihat seorang pemanggil, dan giliran siapa yang
// harus lewat lebih dulu.
//
// Ia hasil terjemahan tiga langkah `Activity/SetReqSalvage_Act-Act.xml`:
//
//	langkah 7  TempLaporan.BranchName := OperatorID.pyUserIdentifier
//	langkah 8  BILA pemanggil MARIATRIELSA -> BranchName := "BAMBANGSETIADJIGUNAWAN"
//	langkah 9  BILA BranchName == "DANIELLISWANDI"
//	           -> TempLaporan.NoteKomite := "AND NOT EXISTS (SELECT 1
//	                FROM POOLDATA.T_CLAIM_CHEKER_SALVAGE C
//	               WHERE C.IDDETAILSALVAGE = A.IDDETAILSALVAGE
//	                 AND C.NAMAKOMITE = 'BAMBANGSETIADJIGUNAWAN'
//	                 AND C.STATUSAPPROVE IS NULL)"
//
// Langkah 9 itu bukan penyaring kepemilikan melainkan **urutan giliran**: ia menyembunyikan
// baris yang komite sebelumnya belum memutuskannya. Membacanya sebagai "penyaring milik
// sendiri" akan membuat seseorang menyimpulkan ia dapat dibuang begitu saja.
type Reviewer struct {
	// Name adalah nilai yang dicocokkan ke `T_CLAIM_CHEKER_SALVAGE.NAMAKOMITE`.
	//
	// Biasanya sama dengan login pemanggil. Ia BERBEDA hanya bagi operatorMaria.
	Name string

	// WaitFor adalah komite yang harus memutus lebih dulu.
	//
	// Kosong berarti tidak ada giliran yang ditunggu, dan itu yang berlaku bagi hampir
	// semua orang. Terisi hanya bagi operatorDaniel.
	WaitFor string

	// Delegated menyatakan pemanggil sedang melihat antrean ORANG LAIN.
	//
	// Ia bukan detail teknis: layar menampilkannya sebagai keterangan, supaya petugas tidak
	// menyimpulkan antreannya sendiri kosong padahal yang ia lihat memang milik orang lain.
	Delegated bool
}

// ReviewerFor menyerahkan antrean yang dilihat seorang pemanggil.
//
// Perbandingan namanya TIDAK peka huruf besar-kecil dan spasinya dipangkas. Di Pega ia
// dibandingkan apa adanya, sehingga login yang tersimpan dengan huruf kecil tidak akan cocok
// dan orangnya melihat antrean yang salah — tanpa satu pun galat. Itu bukan perilaku bisnis
// yang perlu direplikasi; ia akibat dari membandingkan teks tanpa menormalkannya.
func ReviewerFor(caller Caller) Reviewer {
	login := strings.ToUpper(strings.TrimSpace(caller.Login))

	reviewer := Reviewer{Name: login}

	// Langkah 8 — satu orang melihat antrean orang lain.
	if login == operatorMaria {
		reviewer.Name = operatorBambang
		reviewer.Delegated = true
	}

	// Langkah 9 — satu orang menunggu giliran orang lain.
	//
	// Diuji terhadap Name, bukan terhadap login, persis seperti di Pega: langkah 9 di sana
	// membaca `TempLaporan.BranchName` yang baru saja mungkin ditimpa langkah 8. Perbedaannya
	// tidak pernah terlihat hari ini — operatorMaria menjadi operatorBambang, bukan
	// operatorDaniel — tetapi menirunya di tempat yang sama menjaga perilakunya tetap setara
	// bila isi namanya kelak berubah.
	if reviewer.Name == operatorDaniel {
		reviewer.WaitFor = operatorBambang
	}

	return reviewer
}

// DelegationNotice menyatakan bahwa pemanggil sedang melihat antrean orang lain, atau kosong
// bila tidak.
//
// Teksnya menyebut nama komitenya. Itu disengaja: tanpa nama, keterangan "Anda melihat antrean
// petugas lain" justru menimbulkan pertanyaan baru, dan `D-69` membolehkan Operator ID ditulis.
func (r Reviewer) DelegationNotice() string {
	if !r.Delegated {
		return ""
	}
	return "Anda sedang melihat antrean banding milik " + r.Name +
		", bukan antrean Anda sendiri. Aturan itu tertanam di layar lama dan ditiru apa " +
		"adanya; ia akan diganti peran dari master data (F-4)."
}

// QueueNotice menyatakan bahwa sebagian baris ditahan menunggu komite lain, atau kosong bila
// tidak.
func (r Reviewer) QueueNotice() string {
	if r.WaitFor == "" {
		return ""
	}
	return "Banding yang belum diputus " + r.WaitFor + " tidak ditampilkan di sini — " +
		"ia baru muncul setelah giliran itu lewat."
}

// PlanDecision melengkapi sebuah keputusan dengan ketiga langkah tambahannya.
//
// Ketiganya diturunkan dari `Activity/ApprovalCheckerSalvage-Act.xml`, dan SELURUHNYA
// bergantung pada aturan bernama orang — karena itu ia tinggal di berkas ini, bukan di
// decide.go. Hari `F-4` selesai, satu berkas inilah yang dibongkar.
//
// # Ketiga langkah, dan prakondisinya di Pega
//
//	langkah  6  status "0" DAN komite BAMBANG  -> tutup pula baris komite DANIEL
//	langkah  7  status "0"                     -> tandai dokumen "Reject Checker"
//	langkah  9  status "1" DAN komite DANIEL   -> terapkan harga ke DETAIL_PNC_SALVAGE
//
// # Akibat yang harus dibaca apa adanya, bukan diperhalus
//
// Bagi komite mana pun SELAIN kedua nama itu, menyetujui **tidak menerapkan harga**. Langkah
// 9 menuntut `TempInsert.AgentID == "DANIELLISWANDI"`, dan bagi pengguna lain syarat itu
// tidak pernah benar — langkah 8 dan 9 mengarahkannya melewati penerapan harga.
//
// Artinya tombol Approve, bagi hampir semua orang, hanya MENCATAT putusan. Itu perilaku
// layar lama apa adanya (`P-5`), dan ia dinyatakan ke pengguna lewat DecisionResult —
// bukan disembunyikan di balik pesan "berhasil disimpan" yang menyiratkan lebih.
//
// # Rantai dua jenjang yang menjelaskan kenapa begitu
//
// Kedua nama itu membentuk urutan: BAMBANG memutus lebih dulu, DANIEL menyusul (lihat
// Reviewer.WaitFor). Harga baru diterapkan ketika jenjang TERAKHIR menyetujui, dan penolakan
// di jenjang pertama menutup jenjang berikutnya sekaligus — tidak ada gunanya menanyakan
// persetujuan atas harga yang sudah ditolak.
func PlanDecision(command DecisionCommand) DecisionCommand {
	planned := command
	komite := normalizeOperator(command.Reviewer.Name)
	menolak := command.Status == DecisionRejected

	// Langkah 7 — berlaku bagi SIAPA PUN yang menolak.
	planned.MarkDocument = menolak

	// Langkah 6 — penolakan oleh jenjang pertama menutup jenjang berikutnya.
	if menolak && komite == operatorBambang {
		planned.CascadeTo = operatorDaniel
	}

	// Langkah 9 — harga hanya diterapkan oleh jenjang terakhir.
	planned.ApplyPrice = !menolak && komite == operatorDaniel

	return planned
}

// normalizeOperator menyeragamkan nama sebelum dibandingkan, sama seperti ReviewerFor.
func normalizeOperator(name string) string {
	return strings.ToUpper(strings.TrimSpace(name))
}
