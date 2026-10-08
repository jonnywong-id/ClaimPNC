package registrasi

import "strings"

// # PIC Teknik klaim — ClaimData.UserTeknis
//
// Di Pega, tahap yang dirutekan `PNCTeknikRouter` (Choose Surveyor, Send To Analis, Send To
// PIC Teknik) dipegang PIC Teknik KLAIMNYA: dari 338 baris Choose Surveyor di
// POOLDATA.T_CLAIMLIST_ADMIN, pemegang tugasnya sama dengan USERTEKNIS_1 pada 320 baris
// (2026-09-28).
//
// PIC itu dipilih `Activity/getRandomTeam_act-Act.xml` beserta turunannya
// (`GetRandomTeamClaimLeader`, `GetRandomTeamGroup_act`, `GetRandomTeam2_act`) dan prosedur
// `Database/POOLDATA.GETDATA_PICTEKNIK.sql`. Keputusan Work Owner 2026-10-08: aturannya
// disamakan dengan Pega, TERMASUK kondisi yang di Pega tertulis langsung di rule (nama
// petugas, kode sumber bisnis, kode cabang). Itu pengecualian sadar terhadap `D-15`
// sampai aturannya dipindah ke master.
//
// Pembagiannya: PlanTechnicalPIC memutuskan JALUR mana yang berlaku (berkas ini, dapat
// diuji tanpa basis data), dan pengisi seam Assigner menjalankan jalur itu terhadap
// POOLDATA.MST_USER_TEKNIK.
//
// Absensi PIC (`ServiceGetDataAbsenPIC`, langkah 15) dibaca lewat seam AttendanceSource:
// kandidat NONMBU diperiksa berurutan, yang `RuleTimeIn`-nya memuat "000000" dilewati, dan
// akhir pekan/hari libur menghentikan pemilihan tanpa PIC.
//
// # Yang sengaja TIDAK dibawa
//
//   - Rotasi Team A/B lewat `pooldata.pega_dashboardpnc_refresh` (`RandomOver1M_act`,
//     `RandomUnder1M_act`). Ia hanya mengisi label `ClaimData.UserTeknisGroup`, tidak
//     menyaring kueri PIC, dan A maupun B sama-sama menjadi komite NONMBUAB; tabel flag-nya
//     masih ditulis Pega (`P-1`). Menunggu keputusan Work Owner.
//   - Cabang host dev (`pegadev`) pada `DownloadClaimFaceSheet_act` step 6 — perilaku
//     berdasarkan hostname dilarang (Cross-Cutting §3.4).
//
// Kandidat alamat Gmail pada `GetRandomTeam2_act` step 7 DIBAWA atas keputusan Work Owner
// 2026-10-08, sebagai pengecualian sadar terhadap `D-67`.

// technicalPICOf membaca PIC Teknik klaim. Tanda `-` adalah isian "belum ada PIC" pada data
// klaim, bukan nama petugas, sehingga diperlakukan sama dengan kosong — tanpa ini tugas
// tahap teknis diberikan kepada operator bernama "-" dan router tidak pernah memilih ulang.
func technicalPICOf(claim Claim) string {
	pic := strings.TrimSpace(claim.TechnicalPIC)
	if pic == "-" {
		return ""
	}
	return pic
}

// HasTechnicalPIC menyatakan klaim sudah punya PIC Teknik (bukan kosong dan bukan `-`).
func HasTechnicalPIC(claim Claim) bool { return technicalPICOf(claim) != "" }

// AssignedTechnicalPIC mengembalikan PIC Teknik klaim bila tahap itu dirutekan ke PIC Teknik.
// Kosong berarti router harus memilih sendiri.
func AssignedTechnicalPIC(stage Stage, claim Claim) string {
	if stage.Router != RouterPNCTechnical {
		return ""
	}
	return technicalPICOf(claim)
}

// AdoptTechnicalPIC mencatat penerima tugas tahap teknis sebagai PIC Teknik klaim, bila
// klaim belum punya PIC (kosong atau `-`). PIC yang sudah ada tidak pernah ditimpa.
func AdoptTechnicalPIC(claim *Claim, stage Stage, to Assignee) {
	if stage.Router != RouterPNCTechnical || technicalPICOf(*claim) != "" {
		return
	}
	if operator := strings.TrimSpace(to.Operator); operator != "" {
		claim.TechnicalPIC = operator
	}
}

// TechnicalPICPool adalah kelompok petugas di POOLDATA.MST_USER_TEKNIK yang dipakai sebuah
// jalur pemilihan.
type TechnicalPICPool string

const (
	// PoolNone: tidak ada kandidat — klaim dibiarkan tanpa PIC (antrean ServicePNC).
	PoolNone TechnicalPICPool = ""
	// PoolNonMBU: `BrowsePICRandomTeam` / `BrowsePICRandomTeam2`.
	PoolNonMBU TechnicalPICPool = "NONMBU"
	// PoolProcedure: `POOLDATA.GETDATA_PICTEKNIK` (lini PA dan Travel). Cabangnya dipilih
	// JABATAN OPERATOR yang sedang bekerja (`OperatorID.pyPosition`), bukan lini klaim:
	// jabatan "PA" memilih petugas PA bernama menurut TKI (PATechnicalPIC), jabatan lain
	// memilih beban paling ringan TRAVEL.
	PoolProcedure TechnicalPICPool = "PROSEDUR"
)

// PositionPA adalah nilai `OperatorID.pyPosition` yang memilih cabang PA pada
// GETDATA_PICTEKNIK (`txtBusinessType = 'PA'`).
const PositionPA = "PA"

// Petugas yang di Pega tertulis langsung di rule. Dipertahankan atas keputusan Work Owner
// 2026-10-08 — lihat catatan berkas.
const (
	picPATKI        = "DIBADYASANTI"           // GETDATA_PICTEKNIK.sql:19
	picPANonTKI     = "ESTHERSIMBOLON"         // GETDATA_PICTEKNIK.sql:31
	picBranch100639 = "DHARMANTORAHARDJO"      // getRandomTeam_act step 5
	picPHK          = "BAMBANGSETIADJIGUNAWAN" // DownloadClaimFaceSheet_act step 12
	adminJoni       = "JONI_1"                 // getRandomTeam_act step 4, DownloadClaimFaceSheet_act step 8
	branchDharmanto = "100639"

	// ExcludedTechnicalPIC dikecualikan dari kueri NONMBU (`BrowsePICRandomTeam*`).
	ExcludedTechnicalPIC = "ELLENSUPRIYATI"
)

// LargeTechnicalClaim adalah batas Rp 1 miliar yang memisahkan `counter_quota` dari
// `counter_quota2` (`GetRandomTeam2_act` step 8–9).
const LargeTechnicalClaim Money = 1_000_000_000 * 100

// creditBusinessCodes adalah Asuransi Kredit — `getRandomTeam_act` step 6 dan 17.
var creditBusinessCodes = map[string]bool{
	"10165": true, "10164": true, "10168": true, "10053": true, "10145": true, "10075": true,
}

// TechnicalPICPlan adalah jalur pemilihan PIC Teknik sebuah klaim.
type TechnicalPICPlan struct {
	// Operator terisi berarti PIC sudah pasti dan tidak perlu kueri maupun penambahan beban.
	Operator string

	Pool TechnicalPICPool

	// TKI memilih petugas PA pada PoolProcedure — `ClaimData.TKI` (T_CLAIM_PNC.STS_TKI).
	TKI bool

	// TeamC menyaring NONMBU ke `TEAM_GROUP = 'C'`: polis Fac In atau ASM sebagai member.
	TeamC bool

	// Preferred adalah kandidat tetap per sumber bisnis (`GetRandomTeam2_act` step 4–7). Bila
	// terisi, kueri NONMBU dilewati dan kandidat pertama yang dipakai.
	Preferred []string

	// Large memakai COUNTER_QUOTA2 (estimasi > Rp 1 miliar), bukan COUNTER_QUOTA.
	Large bool
}

// PlanTechnicalPIC menentukan jalur pemilihan PIC Teknik, mengikuti urutan
// `getRandomTeam_act`. estimateIDR adalah estimasi klaim dalam rupiah
// (`ClaimEstimate × DollarCurrencyVal`).
func PlanTechnicalPIC(claim Claim, estimateIDR Money) TechnicalPICPlan {
	p := claim.Policy
	admin := strings.TrimSpace(claim.CreatedBy)

	// Step 3: entitas SIMASNET — PIC Teknik = admin klaim.
	if CashierCompany(claim.Portal) == "SIMASNET" && admin != "" {
		return TechnicalPICPlan{Operator: admin}
	}
	// Step 4.
	if admin == adminJoni {
		return TechnicalPICPlan{Operator: admin}
	}
	// Step 5.
	if strings.TrimSpace(p.BranchCode) == branchDharmanto {
		return TechnicalPICPlan{Operator: picBranch100639}
	}
	// Step 6 → BB / CC / KREDIT.
	switch {
	case p.Line == LinePersonalAccident || p.Line == LineTravel:
		// Step 20 (PA) dan step 25 (Travel) SAMA-SAMA mensyaratkan
		// `pyWorkPage.Quotation.BusinessType == "PA"` — salinan Policy.Quotation oleh
		// `ShowCoverage`. Bila tidak terpenuhi, aktivitasnya keluar tanpa PIC (kode 6).
		// Dibawa apa adanya, termasuk akibatnya pada lini Travel.
		if strings.TrimSpace(p.BusinessType) != "PA" {
			return TechnicalPICPlan{}
		}
		return TechnicalPICPlan{Pool: PoolProcedure, TKI: claim.TKI}
	case isCreditOrBonding(p):
		// Step 17: PIC Asuransi Kredit/Bonding = admin.
		if admin == "" {
			return TechnicalPICPlan{}
		}
		return TechnicalPICPlan{Operator: admin}
	}

	// Jalur NONMBU (step 10–15).
	plan := TechnicalPICPlan{Pool: PoolNonMBU}
	fac := strings.TrimSpace(p.TypeOfCoins) == "F"
	member := asmIsMember(claim)
	plan.TeamC = fac || member

	// GetRandomTeam2_act step 1 melompati kandidat tetap untuk Fac In; kandidat tetap hanya
	// berlaku bila ASM member (local.member = "1").
	if !fac && member {
		plan.Preferred = preferredTechnicalPIC(p)
	}

	// Step 8–9 hanya punya `< 1M` dan `> 1M`: tepat Rp 1 miliar tidak menghasilkan kandidat.
	if len(plan.Preferred) == 0 && estimateIDR == LargeTechnicalClaim {
		return TechnicalPICPlan{}
	}
	plan.Large = estimateIDR > LargeTechnicalClaim
	return plan
}

// FaceSheetTechnicalPIC adalah penimpaan PIC Teknik oleh `DownloadClaimFaceSheet_act`
// sesudah pemilihan: admin JONI_1 (step 8) dan jaminan PA PHK (step 12, kode jaminan objek
// pertama). Kosong berarti tidak ada penimpaan.
func FaceSheetTechnicalPIC(claim Claim) string {
	if strings.TrimSpace(claim.CreatedBy) == adminJoni {
		return adminJoni
	}
	if len(claim.InsuredItem) > 0 && len(claim.InsuredItem[0].Coverage) > 0 &&
		PHKCoverages[strings.TrimSpace(claim.InsuredItem[0].Coverage[0].ID)] {
		return picPHK
	}
	return ""
}

// PATechnicalPIC adalah petugas PA yang dipilih `GETDATA_PICTEKNIK` untuk jabatan PA.
func PATechnicalPIC(tki bool) string {
	if tki {
		return picPATKI
	}
	return picPANonTKI
}

func isCreditOrBonding(p Policy) bool {
	if creditBusinessCodes[strings.TrimSpace(p.BusinessCode)] {
		return true
	}
	switch strings.TrimSpace(p.BusinessType) {
	case "Bonding", "BondingKBG":
		return true
	}
	return false
}

// asmIsMember: ada baris CoinsList dengan Leader "false" dan nama memuat ASURANSI SINAR MAS
// (`GetRandomTeam2_act` step 2, `GetRandomTeamGroup_act` step 2).
//
// Klaim hanya menyimpan posisi koasuransi yang sudah diturunkan (DeriveCoinsurance), bukan
// CoinsList-nya. Peran MEMBER di sana berarti baris leader-nya perusahaan lain, yang pada
// polis koasuransi berarti ASM tercatat sebagai baris non-leader.
func asmIsMember(claim Claim) bool {
	return strings.EqualFold(strings.TrimSpace(claim.Policy.Coinsurance.Role), "MEMBER")
}

// preferredTechnicalPIC adalah kandidat tetap per sumber bisnis — `GetRandomTeam2_act`
// step 4–7, yang pertama cocok yang berlaku. TempNoPolis.CaseID = SourceOfBusiness,
// TempNoPolis.ClientID = BusinessCode (`getRandomTeam_act` step 9).
func preferredTechnicalPIC(p Policy) []string {
	sob := strings.TrimSpace(p.SourceOfBusiness)
	code := strings.TrimSpace(p.BusinessCode)
	in := func(v string, set ...string) bool {
		for _, s := range set {
			if v == s {
				return true
			}
		}
		return false
	}
	switch {
	case in(code, "10003017", "10029") || in(sob, "10005351", "10006255"):
		// "PIC Tony untuk bisnis marinehull / sumbis HOWDEN/AON".
		return []string{"TONY"}
	case in(sob, "10001551", "10000952", "10013070", "10000942"):
		// "PIC Jose untuk sumbis IBS".
		return []string{"YOSECHRISTOFER"}
	case sob == "10053930":
		// "PIC Jose untuk sumbis Asuransi TOTAL".
		return []string{"YOSECHRISTOFER", "TONY"}
	case in(sob, "10001889", "10012144", "10001199", "10001173", "10049026") ||
		strings.Contains(p.SourceOfBusinessName, "KBRU"):
		// "PIC Hendry untuk sumbis MIR, KBRU, DSR, MARSH & JLT" — kandidat pertama berupa
		// alamat Gmail, dibawa apa adanya atas keputusan Work Owner 2026-10-08.
		return []string{"BERITAARIELIEZERTARIGAN@GMAIL.COM", "TONY"}
	}
	return nil
}

// Attendance adalah jawaban `ServiceGetDataAbsenPIC` untuk satu petugas pada satu tanggal —
// field yang dibaca `getRandomTeam_act` step 15.
type Attendance struct {
	Day        string // nama hari, mis. "SABTU"
	Holiday    string // "1" = hari libur
	RuleTimeIn string // jam masuk aturan; memuat "000000" berarti tidak dijadwalkan masuk
	TimeIn     string
}

// Closed: akhir pekan atau hari libur — step 15.7 melompat ke EX, pemilihan berhenti TANPA
// PIC.
func (a Attendance) Closed() bool {
	return strings.Contains(a.Day, "SABTU") || strings.Contains(a.Day, "MINGGU") || a.Holiday == "1"
}

// Absent: step 15.8 melewati petugas ini (`RuleTimeIn` memuat "000000") ke kandidat berikutnya.
func (a Attendance) Absent() bool { return strings.Contains(a.RuleTimeIn, "000000") }

// AttendanceApplies: Pega hanya mengurai jawaban absensi (step 15.6) bila nomor polis ada dan
// tanggal kejadian berada di dalam periode polis. Selain itu jawabannya diabaikan dan
// kandidat pertama yang dipakai.
func AttendanceApplies(claim Claim) bool {
	p := claim.Policy
	if strings.TrimSpace(p.Number) == "" {
		return false
	}
	dol := claim.DateOfLoss
	return !dol.Before(p.CoverageStart) && !p.CoverageEnd.Before(dol)
}
