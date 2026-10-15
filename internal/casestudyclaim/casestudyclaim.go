// Package casestudyclaim adalah inti modul **Case Study Claim** (MENU_ID 74).
//
// # Apa yang dimigrasikan
//
//	Harness/PNCStudyClaim-Harness.xml          layar rujukan
//	Section/PNCStudyClaim-Section.xml          24 kolom beserta judulnya, dan tombolnya
//	RDB List/BrowseClaimStudy-SQL.xml          kueri dan keempat penyaringnya
//	Activity/StudyClaim_act-Act.xml            perakitan penyaring sebelum kueri dijalankan
//	Activity/FilterStudyClaim_act-Act.xml      isi dropdown Status
//	Activity/ExportDataCaseStudyClaim-Act.xml  unduhan CSV (pxConvertResultsToCSV)
//	Activity/SaveRemarksRecommendation_act     tombol Save per baris
//	RDB List/SaveRemarksRecommendation_sql     UPDATE-nya
//	Property/StatusReceiver_property.xml       isi dropdown Bisnis
//
// # Apa yang membuat sebuah klaim masuk "Case Study"
//
// Bukan penanda, bukan status, bukan antrean. Satu-satunya syaratnya adalah **NILAI**:
//
//	EXISTS (SELECT … FROM pooldata.t_claim_adjustment
//	         WHERE total_claim * currencyvalue > 5000000000
//	           AND claimid = b.claimid)
//
// Yaitu klaim yang salah satu baris settlement-nya melampaui **Rp 5.000.000.000**. Layar
// ini karena itu bukan Inbox menurut `D-79` — barisnya bukan pekerjaan, tidak hilang
// setelah ditindaklanjuti, dan tidak punya tenggat. Ia layar TELAAH atas klaim besar,
// meski butir menunya berada di bawah kelompok INBOX (`MENU_ID_LEADER 2`).
//
// Ambangnya di-hardcode di dalam rule Pega. `D-15` menetapkan nilai bisnis menjadi master
// data yang dapat diubah tanpa deploy, dan master itu (`F-4`) belum ada — lihat
// LargeClaimThreshold di bawah.
//
// # Alias Pega tidak dibawa masuk
//
// Layar ini termasuk yang terpekat dari utang teknis `03-CURRENT-ARCHITECTURE.md` §4.2:
// dari 24 kolomnya, **21 beralias menyesatkan**. Tidak satu pun dibawa; nama di sini
// mengikuti padanan Inggris (`D-19`, `D-80`), dan judul kolom di layar tetap seperti
// Pega (`D-13`). Pemetaan tiga arahnya ada di columns.go dan di berkas .sql.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package casestudyclaim

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/platform/money"
)

// LargeClaimThreshold adalah ambang nilai yang membuat sebuah klaim masuk layar ini.
//
// # Kenapa ia konstanta, dan kenapa itu BELUM benar
//
// Di Pega ia angka telanjang di dalam teks SQL (`RDB List/BrowseClaimStudy-SQL.xml`).
// `D-15` menetapkan seluruh ambang nilai menjadi **master data yang dapat diubah tanpa
// deploy**, dan `F-4` adalah modul yang akan memilikinya — modul itu belum ada.
//
// Menaruhnya di sini adalah langkah setengah jalan yang disengaja: ia keluar dari teks
// SQL dan menjadi satu nilai bernama yang terlihat, dapat diuji, dan dikirim ke basis
// data lewat parameter binding. Begitu master ambang tiba, yang berubah hanyalah dari
// mana nilai ini dibaca — bukan bentuk kuerinya.
//
// Ia TIDAK dijadikan penyaring yang dapat diubah pengguna. Layar lama tidak punya isian
// itu, dan menambahkannya akan mengubah populasi yang dibandingkan saat `S-8` dijalankan.
var LargeClaimThreshold = money.FromRupiah(5_000_000_000)

// CaseStudyRow adalah satu baris pada layar.
//
// # Pemetaan tiga arah
//
// Kolom "properti Pega" ditulis apa adanya supaya penelusuran balik ke section tetap
// mungkin. Tanda (!) menandai alias yang artinya BERLAWANAN dengan isinya.
//
//	judul kolom di layar         properti Pega             kolom sebenarnya
//	---------------------------  ------------------------  ----------------------------
//	PNC Case ID                  .CaseID               (!) A.NOKLAIM
//	No Polis                     .NoKTP                (!) A.NOPOLIS
//	Nama Tertanggung             .NamaSurveyor         (!) B.QQNAME
//	COB                          .City                 (!) D.NOTE
//	Periode Polis                .AnalystTransferDate  (!) A.THNREGIS
//	Bulan Klaim                  .Email                (!) TO_CHAR(B.REGISTERDATE,'mm')
//	Date of Loss                 .DateOfLoss               A.DATEOFLOSS
//	SOB                          .CityID               (!) B.SOBNAME
//	Leader/Member/Fac In         .AgingAmount          (!) CASE A.REINSURER
//	Nature of Loss               .Country              (!) A.COL_DESC
//	Cause of Loss                .Country              (!) A.COL_DESC  — KOLOM YANG SAMA
//	TSI (100%)                   .AlasanDokterRejectRCL(!) A.TSI
//	ASM SHARE                    .AlasanTerlambat      (!) MAX(ASM_SHARE)
//	Deductible                   .CloseClaimNote       (!) SUM(INDIVIDUAL_RISK_VALUE*CV)
//	Nilai share ASM              .CommentKomiteClosecase(!)SUM(TOTAL_CLAIM*CV*ASM_SHARE/100)
//	NILAI KLAIM 100%             .ComplianceRemark     (!) SUM(TOTAL_CLAIM*CV)
//	ADJUSTER FEE 100% SHARE      .Conveyance           (!) SUM(GROSSVALUE*CV)
//	NILAI KLAIM NET 100%         .UserName             (!) rumus net, lihat .sql
//	NILAI KLAIM NET ASM SHARE    .UserTeknis           (!) net × MAX(ASM_SHARE/100)
//	LACK OF DOC/SALVAGE/…        .CompliancePosAuditByr(!) LOC + salvage
//	Cabang                       .AnalystRemaksInvestigator(!) A.CABANG
//	Status                       .ClaimNo              (!) CASE A.STSKLAIM
//	Kronologi                    .LokasiSurveyor       (!) B.KRONOLOGI
//	Remark                       .AnaylstRemarks           B.REMARKRECOMENDATION
//
// Salah ketik `.AnaylstRemarks` ada di export, bukan di sini.
type CaseStudyRow struct {
	// ClaimNumber — kolom "PNC Case ID". Ia juga KUNCI baris: tombol Save mengirimkannya
	// sebagai parameter `CASE`, dan UPDATE-nya menyaring `WHERE claimno = …`.
	ClaimNumber string

	PolicyNumber string // "No Polis"
	InsuredName  string // "Nama Tertanggung"

	// BusinessName — kolom "COB" (Class of Business), dari `POOLDATA.BUSINESS.NOTE`.
	//
	// Join-nya INNER (`B.BUSINESSCODE = D.ID`), sama seperti Pega: klaim yang kode
	// bisnisnya tidak punya baris master TIDAK muncul sama sekali. Itu direplikasi apa
	// adanya, dan dicatat karena ia cara baris menghilang tanpa satu pun galat.
	BusinessName string

	// PolicyPeriod — kolom "Periode Polis", isinya `A.THNREGIS` yaitu **tahun
	// registrasi**.
	//
	// Judul dan isi tidak sejalan, dan keduanya dipertahankan: judulnya karena `D-13`,
	// isinya karena `P-5`. Kolom inilah yang dibandingkan penyaring rentang tanggal —
	// lihat Filter.From.
	PolicyPeriod string

	// ClaimMonth — kolom "Bulan Klaim", dua digit bulan dari `B.REGISTERDATE`.
	//
	// Ia datang dari tabel yang BERBEDA dengan PolicyPeriod di atas: tahunnya dari
	// `PEGA_DASHBOARDPNC`, bulannya dari `T_CLAIM_PNC`. Keduanya tidak dijamin sepakat,
	// dan Pega pun tidak memeriksanya.
	ClaimMonth string

	LossDate *time.Time // "Date of Loss" — boleh kosong

	BusinessSource string // "SOB" (Source of Business)

	// ReinsurerRole — kolom "Leader/Member/Fac In", diturunkan dari `A.REINSURER`.
	// Lihat ReinsurerRoleOf.
	ReinsurerRole string

	// CauseOfLoss mengisi DUA kolom sekaligus — "Nature of Loss" dan "Cause of Loss".
	//
	// Keduanya di Pega terikat properti yang SAMA (`.Country`), dan kuerinya hanya
	// menyediakan satu nilai (`A.COL_DESC`). Jadi kedua kolom itu memang selalu berisi
	// teks yang sama persis. Itu direplikasi (`P-5`, keputusan Work Owner 2026-09-26) dan
	// dinyatakan terbuka di layar, bukan diperbaiki diam-diam: memilih salah satunya
	// berarti menebak mana yang dimaksud, dan tidak ada bukti yang menjawabnya.
	CauseOfLoss string

	// Sembilan nilai berikut BOLEH KOSONG, dan kosong BERBEDA dari nol.
	//
	// Seluruhnya hasil agregat atas `POOLDATA.T_CLAIM_ADJUSTMENT`. `SUM` atas himpunan
	// kosong mengembalikan NULL, bukan nol — dan pada berkas yang dibuka di pengolah
	// angka, nol adalah angka yang ikut terhitung dalam rata-rata. "Belum ada nilainya"
	// bukan "nilainya nol".
	TSI           *money.Money // "TSI (100%)" — dari A.TSI, bukan agregat
	Deductible    *money.Money // "Deductible"
	ASMShareValue *money.Money // "Nilai share ASM"
	ClaimValue100 *money.Money // "NILAI KLAIM 100%"
	AdjusterFee   *money.Money // "ADJUSTER FEE 100% SHARE"
	NetClaim100   *money.Money // "NILAI KLAIM NET 100%"
	NetClaimASM   *money.Money // "NILAI KLAIM NET ASM SHARE"
	LackOfDoc     *money.Money // "LACK OF DOC/SALVAGE / RECOVERY / SUBROGARATION"

	// ASMSharePercent — kolom "ASM SHARE", persentase dikali 10.000.
	//
	// Empat desimal, bentuk yang sama dengan share spreading pada modul Registrasi:
	// `D-51` menetapkan toleransi spreading diuji pada empat desimal, dan memakai bentuk
	// yang sama di sini membuat kedua layar menampilkan angka yang dapat dibandingkan.
	ASMSharePercent *int64

	BranchName string // "Cabang"

	// ClaimStatus — kolom "Status". Label, bukan kode; lihat ClaimStatusOf.
	ClaimStatus string

	Chronology string // "Kronologi"

	// Remark — kolom "Remark", SATU-SATUNYA sel yang dapat disunting di seluruh layar
	// (`pyEditOptions=Editable` pada `Section/PNCStudyClaim-Section.xml:12814`).
	Remark string
}

// Nilai `A.REINSURER` sebagaimana tersimpan, beserta labelnya di layar.
//
// Diambil dari `CASE` pada `RDB List/BrowseClaimStudy-SQL.xml`. Label berbahasa Inggris
// karena itulah yang tertulis di layar lama (`D-13`).
const (
	reinsurerLeader = "1"
	reinsurerMember = "2"
	reinsurerFacIn  = "F"
)

// ReinsurerRoleOf menurunkan label kolom "Leader/Member/Fac In".
//
// Nilai yang tidak dikenal menjadi tanda hubung, persis seperti cabang `else '-'` pada
// kueri lama. Ia TIDAK ditampilkan apa adanya di sini — berbeda dari DisplayStatus pada
// modul My Inbox — karena Pega pun menyembunyikannya, dan menampilkannya akan menjadi
// selisih pada uji kesetaraan tanpa menambah keterangan apa pun.
func ReinsurerRoleOf(raw string) string {
	switch strings.TrimSpace(raw) {
	case reinsurerLeader:
		return "Leader"
	case reinsurerMember:
		return "Member"
	case reinsurerFacIn:
		return "Fac In"
	default:
		return "-"
	}
}

// Nilai `A.STSKLAIM` sebagaimana tersimpan.
//
// Perhatikan: kolom ini BUKAN `StatusClaim` berkode `1134`–`1166` yang `ADR-0018`
// bicarakan. Ia penanda tiga nilai milik tabel datar `PEGA_DASHBOARDPNC`, dan salah satu
// dari empat konsep status yang `D-18` larang digabung.
const (
	claimStatusOnProgress = "0"
	claimStatusAccepted   = "1"
	claimStatusRejected   = "3"
)

// ClaimStatusOf menurunkan label kolom "Status".
//
// Perhatikan ketimpangan yang direplikasi: `'0'` dan `'1'` berlabel bahasa Inggris
// ("Claim On Progress", "Accepted") sementara `'3'` berlabel bahasa Indonesia
// ("Ditolak"). Campuran itu ada di kueri Pega, dan `D-13` menetapkan teks yang dilihat
// pengguna mengikuti layar lama apa adanya — termasuk ketika layar lama tidak konsisten.
func ClaimStatusOf(raw string) string {
	switch strings.TrimSpace(raw) {
	case claimStatusOnProgress:
		return "Claim On Progress"
	case claimStatusAccepted:
		return "Accepted"
	case claimStatusRejected:
		return "Ditolak"
	default:
		return "-"
	}
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// Total ikut dikembalikan meski `10-API-STRATEGY.md` §4 menyarankan menghindari `COUNT(*)`
// pada tabel besar. Di sini ia murah dan berguna: penyaring ambang Rp 5 miliar memangkas
// populasinya menjadi sangat kecil, dan "berapa klaim besar pada periode ini" justru
// angka yang dicari pengguna layar telaah.
type Page struct {
	Rows  []CaseStudyRow
	Total int
}

// Repo adalah seam ke penyimpanan.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya
// (`08-TECHNICAL-STRATEGY.md` §2 aturan 2). Diisi `repo/sqlstore` terhadap Oracle dan
// `repo/memory` untuk pengujian.
type Repo interface {
	// List membaca satu halaman beserta jumlah seluruh baris yang cocok.
	List(ctx context.Context, f Filter) (Page, error)

	// SaveRemark menuliskan catatan telaah satu klaim.
	//
	// # Ia satu-satunya method tulis modul ini, dan tabelnya MILIK PEGA
	//
	// `POOLDATA.T_CLAIM_PNC` ditulis sistem lama, dan `P-1` menetapkan satu tabel hanya
	// boleh ditulis satu sistem. Penulisan ini karena itu menuntut serah-terima
	// kepemilikan tulis atas SATU KOLOM — `REMARKRECOMENDATION` — lewat `D-63`:
	// permintaan tertulis, persetujuan Work Owner, pelaksanaan DBA.
	//
	// Presedennya sudah ada dan sudah berjalan: modul Inbox Receive TKA menulis kolom
	// `TGLDOKLENGKAP` pada tabel yang sama, dengan pagar yang sama.
	//
	// Nilai kedua `false` berarti nomor klaimnya tidak ditemukan — BUKAN galat. Baris
	// dapat hilang di antara saat daftar dibaca dan saat Save ditekan, dan itu keadaan
	// yang sah yang pantas dijawab dengan "muat ulang daftarnya", bukan dengan 500.
	SaveRemark(ctx context.Context, claimNumber, remark string) (bool, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal
//
// Ketiga tabel yang dibaca modul ini — `PEGA_DASHBOARDPNC`, `T_CLAIM_PNC`, dan
// `T_CLAIM_ADJUSTMENT` — ada di basis data SETIAP entitas (`ADR-0030`). Klaim milik
// Asuransi Sinar Mas dan klaim milik Simas Insurtech tidak pernah berada di tabel yang
// sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Taruhannya di layar ini lebih besar daripada rata-rata:
// barisnya adalah klaim di atas Rp 5 miliar beserta nama tertanggungnya, dan jatuh ke
// koneksi default berarti menampilkannya di layar badan hukum lain tanpa satu pun galat
// (`R-20`, `TKT-F6-002`).
type RepoSelector func(portalAlias string) (Repo, error)
