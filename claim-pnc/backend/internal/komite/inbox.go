package komite

import (
	"context"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
)

// Inbox Komite — daftar pekerjaan milik seorang anggota komite (`D-79`).
//
// # Apa yang digantikan
//
// Harness `InboxKomite_Harness` beserta section `InboxKomite_section` (1,8 MB), yang di
// sana merakit TIGA grid dari tiga kueri berbeda:
//
//   - `RDB List/GetKomitePAOutstanding-SQL.xml`      → Kotak Masuk Komite Outstanding
//   - `RDB List/GetKomitePAditerima-SQL.xml`         → Kotak Masuk Komite Diterima
//   - `RDB List/ShowKomiteTerimaTolakNonMBU-SQL.xml` → Kotak Masuk Komite Ditolak
//
// Ketiganya membaca tabel yang sama dengan penyaring berbeda. Di sini ketiganya menjadi
// SATU kueri berparameter `InboxKind`, sehingga aturan "baris mana milik siapa" hidup di
// satu tempat — bukan tiga salinan yang dapat berbeda pendapat.
//
// # Ini worklist, bukan workbasket — dan itu menentukan segalanya
//
// Penyaring inti kueri lama adalah:
//
//	B.PXASSIGNEDOPERATORID = <operator yang login>
//	A.PYSTATUSWORK        <> 'Resolved-Completed'
//
// Komite ditugaskan ke OPERATOR BERNAMA, bukan ke antrean bersama (`T-7`, `D-26`).
// Akibatnya yang tidak boleh dilupakan: **ketidakhadiran satu orang menghentikan klaim**,
// karena tidak ada orang lain yang berhak mengambil pekerjaan itu. Kolom `STS_ABS` pada
// master tampaknya menjawab ini dan perilakunya belum dirumuskan (`TKT-B07-002`).

// InboxKind adalah kotak mana yang sedang dibuka.
//
// Nilainya berbahasa Indonesia karena ia **kontrak API** — ia muncul apa adanya sebagai
// parameter kueri `?kotak=` — bukan nama internal (`D-80`).
type InboxKind string

const (
	// InboxOutstanding adalah pekerjaan yang MENUNGGU keputusan pemanggil.
	//
	// Inilah satu-satunya kotak yang berisi pekerjaan dalam arti `D-79`: barisnya hilang
	// setelah diputuskan, dan ia punya tenggat yang terbaca dari Aging.
	InboxOutstanding InboxKind = "outstanding"

	// InboxAccepted dan InboxRejected adalah RIWAYAT keputusan pemanggil.
	//
	// Keduanya bukan pekerjaan — barisnya tidak pernah hilang dan tidak punya tenggat.
	// Mereka tetap ada di layar ini karena di sistem lama pun demikian, dan karena
	// anggota komite memakainya untuk menelusuri kembali apa yang pernah ia putuskan.
	InboxAccepted InboxKind = "diterima"
	InboxRejected InboxKind = "ditolak"
)

// Valid menyatakan kotak yang diminta dikenali.
func (k InboxKind) Valid() bool {
	switch k {
	case InboxOutstanding, InboxAccepted, InboxRejected:
		return true
	default:
		return false
	}
}

// InboxKinds mengembalikan seluruh kotak dalam urutan yang ditampilkan layar.
//
// Urutannya mengikuti alur kerja, bukan abjad: yang menunggu lebih dulu, riwayat
// sesudahnya. Anggota komite membuka layar ini untuk mengerjakan sesuatu, bukan untuk
// membaca riwayat.
func InboxKinds() []InboxKind {
	return []InboxKind{InboxOutstanding, InboxAccepted, InboxRejected}
}

// CommitteeCase adalah satu baris di Inbox Komite.
//
// # Nama field TIDAK mengikuti nama property Pega, dan itu disengaja
//
// Section lama menampilkan kolomnya lewat property yang namanya sama sekali tidak
// mencerminkan isinya — persis utang teknis yang `docs/Steering/03-CURRENT-ARCHITECTURE.md`
// §4.2 catat. Yang benar-benar terjadi di `ShowKomiteTerimaTolakNonMBU`:
//
//	.IBNR           → caption "Nilai ASM Share"  → NILAIKLAIM × SHAREASM / 100
//	.pyScore        → caption "Nilai OR ASM"     → NILAIKLAIM × Σ PRSN_* / 100
//	.DraftWordingID → caption "PIC Klaim"        → T_CLAIM_PNC.PICTEKNIK
//	.RejectedCode   → caption "Alasan Reject"    → NOTEKOMITE
//	.StatusKlaim    → caption "Tipe Komite"      → TYPEKOMITE × PAYMENTTYPE
//
// Kelimanya dinamai ulang di sini mengikuti `D-19` dan `CONTEXT.md`. Pemetaan ke kolom
// aslinya ada di repo/sqlstore, satu tempat saja.
type CommitteeCase struct {
	// CaseID adalah nomor case komite — `pyID` pada kelas `ASM-FW-GCNMFW-Work-Komite`.
	// Inilah kunci yang dipakai `T_CLAIM_KOMITE_LIST.KOMITE_ID`.
	CaseID string

	// ClaimNumber adalah nomor klaim yang dikomitekan.
	//
	// Di sistem lama ia diambil dengan `SUBSTR(PNCCASEID, INSTR(PNCCASEID,' ')+1)` —
	// memotong prefix kelas Pega `ASM-FW-GCNMFW-WORK ` yang bocor ke dalam data bisnis
	// (utang teknis §4.1). Pemotongan itu dikerjakan di kueri, dan hasilnya sajalah yang
	// masuk ke sini: prefix Pega tidak pernah sampai ke domain maupun ke layar (`D-22`).
	ClaimNumber string

	PolicyNumber     string
	InsuredName      string
	BusinessName     string
	SourceOfBusiness string
	BranchName       string

	// AssignedOperator adalah pemilik pekerjaan ini: `PXASSIGNEDOPERATORID` pada
	// `PC_ASSIGN_WORKLIST` untuk kotak Outstanding, `PYRESOLVEDUSERID` untuk riwayat.
	AssignedOperator string

	// CommitteeDate adalah kapan jenjang ini mulai menunggu, dan dasar perhitungan Aging.
	//
	// `InboxRegisterKomite_RD` menyebutnya `.Komite.DateOfComitee`, tetapi properti itu
	// BUKAN kolom basis data — diverifikasi ke `ALL_TAB_COLUMNS`, ia tidak ada di antara
	// 186 kolom `PC_ASM_FW_GCNMFW_WORK` dan hidup di blob Pega. Penggantinya diambil dari
	// kedua rule yang menjadi sumber layar ini: `TANGGALKOMITE` bila ada, jatuh ke tanggal
	// pembuatan case bila tidak. Rinciannya di repo/sqlstore/inbox.sql.
	CommitteeDate time.Time

	// CreatedAt adalah `PXCREATEDATETIME` case komitenya.
	CreatedAt time.Time

	// WorkStatus adalah `PYSTATUSWORK` apa adanya.
	//
	// Ia dibawa mentah, tidak diterjemahkan, karena ia status ENGINE Pega — bukan salah
	// satu dari empat konsep status bisnis pada `D-18`. Menerjemahkannya akan
	// menyiratkan ia punya arti bisnis yang sebenarnya tidak ia punya.
	WorkStatus string

	// LegacyOutcome adalah keputusan yang TERCATAT DI PEGA, diturunkan dari
	// `T_CLAIM_KOMITE_LIST.STATUSAPPROVE`.
	//
	// Kueri lama menurunkannya begitu saja:
	//
	//	case when b.STATUSAPPROVE = '1' then 'DITERIMA' else 'DITOLAK' end
	//
	// Perhatikan `else`-nya: baris yang BELUM diputuskan ikut terbaca "DITOLAK" di sana,
	// karena kolom kosong bukan '1'. Itu tidak ditiru. Di sini ketiadaan keputusan
	// menjadi OutcomePending, dan hanya nilai yang benar-benar ada yang menjadi diterima
	// atau ditolak — perbedaan yang menentukan apakah sebuah klaim yang masih menunggu
	// tampak sudah ditolak.
	//
	// Inilah yang menentukan isi kotak Diterima dan Ditolak, persis seperti
	// `GetKomitePAditerima` pada `SetDataKomitePNC_Act`.
	LegacyOutcome Outcome

	// TierCount adalah banyaknya jenjang yang harus menyetujui kasus ini — `KomiteLoop`
	// di sistem lama.
	//
	// # Hari ini ia hampir selalu NOL, dan itu bukan cacat
	//
	// Menghitungnya menuntut dua hal: nilai klaim dan LINI BISNIS. Nilai klaim ada di
	// `T_CLAIM_KOMITE_LIST`. Lini bisnisnya TIDAK ADA di sana — di sistem lama ia
	// diturunkan `Activity/SetEmailKomite-Act.xml`, rule yang mencampur `BusinessType`,
	// pencocokan nama server, dan tiga nama orang (`D-52` mencabut yang terakhir).
	//
	// Menebak pemetaan itu berarti mengarang aturan yang menentukan BERAPA ORANG harus
	// menyetujui sebuah klaim — tepat jenis tebakan yang `D-47` buktikan berbahaya, saat
	// "perbaikan" yang tampak masuk akal hampir menghapus penjenjangan seluruhnya.
	//
	// Karena itu nilainya dibiarkan nol, `TierCountUnknown()` bernilai benar, dan layar
	// menyatakannya apa adanya. Yang ingin mengetahui jumlah jenjang untuk sebuah nilai
	// dapat memakai simulator yang sudah ada di `/api/komite/penjenjangan`, tempat lini
	// bisnisnya disebut eksplisit oleh orang yang tahu.
	TierCount int

	// Progress adalah keadaan penjenjangan kasus ini MENURUT SISTEM BARU — hasil
	// pembacaan tabel keputusan milik aplikasi ini, bukan milik Pega.
	//
	// Ia diisi lapisan usecase, bukan repo, karena ia gabungan dua sumber. Lihat catatan
	// pada Evaluate.
	Progress Progress
}

// AgingDays mengembalikan lama kasus ini menunggu, dalam hari kalender WIB.
//
// # Yang ditiru dan yang diperbaiki
//
// Sistem lama menghitungnya di dalam SQL:
//
//	TRUNC(SYSDATE) - TO_DATE(TO_CHAR(A.PXCREATEDATETIME,'dd/mm/yyyy'),'DD/MM/YYYY')
//
// Bentuknya ditiru — selisih TANGGAL KALENDER, bukan selisih jam — sehingga kasus yang
// masuk kemarin sore dan dibaca pagi ini tetap berumur 1 hari, bukan 0.
//
// Yang DIPERBAIKI: `SYSDATE` adalah jam server basis data dan `TO_CHAR` memakai zona
// sesi, sehingga hasilnya bergeser bergantung tempat kueri dijalankan. Di sini keduanya
// dipotong ke tanggal WIB lewat satu-satunya tempat konversi zona di aplikasi ini
// (`F-5`), sehingga angkanya sama siapa pun yang membukanya dari mana pun.
func (c CommitteeCase) AgingDays(now time.Time) int {
	if c.CommitteeDate.IsZero() {
		return 0
	}
	days := clock.DaysBetween(c.CommitteeDate, now)
	if days < 0 {
		// Tanggal komite di masa depan berarti data yang keliru, bukan umur negatif.
		// Melaporkannya sebagai 0 jauh lebih jujur daripada angka minus yang terbaca
		// seperti hitungan mundur.
		return 0
	}
	return days
}

// Normalized merapikan penulisan sebelum kasus ini dipakai membandingkan apa pun.
//
// Perapian ini menutup cacat data nyata, bukan kerapian kosmetik: `OPERATOR_ID` pada
// `Database/emailkomite.csv` baris ID 4 berakhir dengan BARIS BARU, dan kolom warisan
// lain tersimpan tanpa penyeragaman. Satu spasi di ujung cukup membuat inbox seseorang
// tampak kosong tanpa satu pun galat.
func (c CommitteeCase) Normalized() CommitteeCase {
	c.CaseID = strings.TrimSpace(c.CaseID)
	c.ClaimNumber = strings.TrimSpace(c.ClaimNumber)
	c.PolicyNumber = strings.TrimSpace(c.PolicyNumber)
	c.InsuredName = strings.TrimSpace(c.InsuredName)
	c.BusinessName = strings.TrimSpace(c.BusinessName)
	c.SourceOfBusiness = strings.TrimSpace(c.SourceOfBusiness)
	c.BranchName = strings.TrimSpace(c.BranchName)
	c.AssignedOperator = strings.TrimSpace(c.AssignedOperator)
	c.WorkStatus = strings.TrimSpace(c.WorkStatus)
	return c
}

// BelongsTo menyatakan kasus ini benar-benar milik operator tersebut.
//
// Dipakai sebelum sebuah keputusan dicatat. Penyaring di kueri sudah membatasi apa yang
// TERLIHAT, tetapi pemeriksaan ini membatasi apa yang dapat DILAKUKAN — dan keduanya
// tidak boleh disamakan: nomor case yang terlihat di satu layar dapat dikirim ke endpoint
// mana pun oleh siapa pun yang punya sesi.
//
// Perbandingannya memakai OperatorKey, sehingga besar-kecil huruf dan spasi tepi tidak
// menentukan siapa yang berwenang menyetujui uang.
func (c CommitteeCase) BelongsTo(operator string) bool {
	if strings.TrimSpace(operator) == "" {
		return false
	}
	return OperatorKey(c.AssignedOperator) == OperatorKey(operator)
}

// StatusResolved adalah nilai `PYSTATUSWORK` yang menandai case Pega sudah tuntas.
//
// Ia dipakai apa adanya sebagai penyaring kotak Outstanding, persis seperti
// `A.PYSTATUSWORK <> 'Resolved-Completed'` pada `GetKomitePAOutstanding`.
const StatusResolved = "Resolved-Completed"

// InBox menyatakan kasus ini muncul di kotak tertentu milik operator tertentu.
//
// # Ini definisi kanonik, dan SQL menirunya
//
// Aturan yang sama hidup di dua tempat — di sini, dan di klausa WHERE pada
// repo/sqlstore/inbox.sql. Itu melanggar prinsip "satu aturan satu tempat", dan
// pelanggarannya disengaja: menyaring di Go menuntut seluruh antrean komite dibaca ke
// memori sebelum satu halaman ditampilkan, dan kasus komite tumbuh bersama jumlah klaim.
//
// Yang dikerjakan supaya kedua salinan tidak berbeda pendapat diam-diam: fungsi inilah
// yang MENDEFINISIKAN aturannya, kueri SQL menyebutnya dalam komentar, dan adapter memori
// memakainya langsung — sehingga setiap uji yang berjalan tanpa basis data menguji
// definisi ini, bukan salinan lain.
//
// # Kenapa "sudah saya putuskan" mengeluarkan kasus dari Outstanding
//
// Penugasan komite per ORANG (`T-7`): satu jenjang dimiliki satu operator bernama.
// Menyelesaikan assignment mengeluarkannya dari worklist orang itu di sistem lama, dan
// itulah yang ditiru — jenjang lain tetap menunggu pemiliknya masing-masing.
func (c CommitteeCase) InBox(kind InboxKind, operator string) bool {
	if !c.BelongsTo(operator) {
		return false
	}
	decided := c.Progress.DecidedBy(operator)

	switch kind {
	case InboxOutstanding:
		return !decided && c.WorkStatus != StatusResolved

	case InboxAccepted:
		if decided {
			return c.decidedKindBy(operator) == DecisionApprove
		}
		return c.LegacyOutcome == OutcomeApproved

	case InboxRejected:
		if decided {
			return c.decidedKindBy(operator).Terminal()
		}
		return c.LegacyOutcome == OutcomeRejected || c.LegacyOutcome == OutcomeReturned

	default:
		return false
	}
}

// decidedKindBy mengembalikan keputusan yang orang itu berikan pada kasus ini.
func (c CommitteeCase) decidedKindBy(operator string) DecisionKind {
	key := OperatorKey(operator)
	for _, d := range c.Progress.Decisions {
		if OperatorKey(d.ActorLogin) == key {
			return d.Kind
		}
	}
	return ""
}

// MatchesSearch mencocokkan kasus dengan kotak pencarian "No Komite / No Klaim".
//
// Satu kotak untuk dua kolom, persis seperti prompt pada section lama — pengguna tidak
// selalu tahu yang mana yang sedang ia pegang. Pencocokannya mengandung, bukan diawali:
// nomor klaim sering disalin sebagian dari surel atau percakapan.
func (c CommitteeCase) MatchesSearch(search string) bool {
	needle := strings.ToUpper(strings.TrimSpace(search))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToUpper(c.CaseID), needle) ||
		strings.Contains(strings.ToUpper(c.ClaimNumber), needle)
}

// WithinDateRange mencocokkan kasus dengan "Tgl Input Dari" dan "Tgl Input Sampai".
//
// Inklusif di KEDUA ujung, dihitung terhadap tanggal WIB. Batas eksklusif di ujung atas
// adalah kesalahan paling umum pada penyaring tanggal, dan akibatnya pekerjaan hari itu
// tampak tidak ada.
func (c CommitteeCase) WithinDateRange(from, to time.Time) bool {
	if c.CreatedAt.IsZero() {
		return from.IsZero() && to.IsZero()
	}
	day := clock.DateWIB(c.CreatedAt)

	if !from.IsZero() && day.Before(clock.DateWIB(from)) {
		return false
	}
	if !to.IsZero() && day.After(clock.DateWIB(to)) {
		return false
	}
	return true
}

// DefaultPageSize adalah banyaknya baris per halaman bila pemanggil tidak menyebutnya.
const DefaultPageSize = 25

// MaxPageSize membatasi permintaan yang menyebut batasnya sendiri.
//
// Angkanya 100, mengikuti `docs/Steering/10-API-STRATEGY.md` §4: permintaan yang lebih
// besar DITOLAK, bukan dipenuhi. Ini sekaligus meninggalkan `pyMaxRecords=500` sistem
// lama, yang bukan paginasi melainkan PEMOTONGAN — hasil ke-501 di sana hilang tanpa
// satu pun tanda bahwa ia ada (`T-12`).
const MaxPageSize = 100

// InboxFilter menyaring isi inbox.
type InboxFilter struct {
	// Operator adalah pemilik inbox. Kosong berarti TIDAK ADA HASIL, bukan semua hasil —
	// lihat Normalize.
	Operator string

	// AllOperators mematikan penyaring pemilik: daftarnya menjadi SELURUH antrean komite,
	// bukan milik satu orang.
	//
	// # Ia medan tersendiri, bukan "Operator yang kebetulan kosong"
	//
	// Perbedaannya menentukan. Operator kosong adalah KEGAGALAN — identitas sesi tidak
	// terbaca, atau tidak cocok dengan satu pun `OPERATOR_ID` di data warisan (`ADR-0024`)
	// — dan kegagalan itu dijawab nol baris. Menyatukan keduanya berarti setiap kegagalan
	// pembacaan identitas berubah menjadi "tampilkan seluruh antrean komite perusahaan",
	// dan itu tepat kebalikan dari yang aman.
	//
	// Dengan medan terpisah, tidak-menyaring harus DIMINTA. Ia tidak dapat terjadi karena
	// sesuatu yang lain gagal.
	//
	// # Ia hanya boleh menyala di pengembangan
	//
	// Diminta Work Owner 2026-09-29 supaya isi Inbox Outstanding dapat dilihat sementara
	// pemetaan identitas HCC/HCQ ke `OPERATOR_ID` belum ada. Penyalaannya lewat
	// `KOMITE_TANPA_PENYARING_OPERATOR`, yang MENOLAK berjalan di luar
	// `APP_ENV=development` — lihat platform/config.
	//
	// Selama menyala, layar menampilkan pekerjaan orang lain beserta nama tertanggung dan
	// nomor polisnya. Itu dinyatakan di layar, bukan disembunyikan.
	AllOperators bool

	Kind InboxKind

	// Search mencocokkan "No Komite / No Klaim", persis seperti prompt pencarian pada
	// section lama. Ia SATU kotak untuk dua kolom karena pengguna tidak selalu tahu yang
	// mana yang sedang ia pegang.
	Search string

	// DateFrom dan DateTo menyaring "Tgl Input Dari" dan "Tgl Input Sampai".
	// Keduanya tanggal kalender WIB, inklusif di kedua ujung.
	DateFrom time.Time
	DateTo   time.Time

	Offset int
	Limit  int
}

// Normalize merapikan penyaring dan menerapkan batas yang mengikat.
//
// # Operator kosong tidak dinormalkan menjadi "semua"
//
// Ia dibiarkan kosong, dan repo yang menerjemahkannya menjadi NOL BARIS. Inbox adalah
// daftar pekerjaan SESEORANG; penyaring pemilik yang gagal terisi lalu diam-diam
// diartikan "tampilkan semua" akan membocorkan seluruh antrean komite — termasuk nilai
// klaim dan nama tertanggung — kepada siapa pun yang punya sesi.
//
// Kegagalan yang aman di sini adalah MENAMPILKAN TERLALU SEDIKIT, bukan terlalu banyak.
// Satu-satunya cara memperoleh "semua" adalah MEMINTANYA lewat AllOperators.
func (f InboxFilter) Normalize() InboxFilter {
	f.Operator = OperatorKey(f.Operator)
	f.Search = strings.TrimSpace(f.Search)

	if !f.Kind.Valid() {
		f.Kind = InboxOutstanding
	}
	if !f.DateFrom.IsZero() {
		f.DateFrom = clock.DateWIB(f.DateFrom)
	}
	if !f.DateTo.IsZero() {
		f.DateTo = clock.DateWIB(f.DateTo)
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Limit <= 0 {
		f.Limit = DefaultPageSize
	}
	if f.Limit > MaxPageSize {
		f.Limit = MaxPageSize
	}
	return f
}

// DateRangeInverted menyatakan rentang tanggalnya terbalik.
//
// Ia dikembalikan sebagai pelanggaran validasi, bukan diperbaiki diam-diam dengan
// menukar kedua ujungnya. Pengguna yang salah mengisi harus melihat bahwa ia salah;
// menukarnya diam-diam akan menampilkan hasil yang benar untuk pertanyaan yang tidak ia
// ajukan.
func (f InboxFilter) DateRangeInverted() bool {
	return !f.DateFrom.IsZero() && !f.DateTo.IsZero() && f.DateTo.Before(f.DateFrom)
}

// Validate mengumpulkan SELURUH pelanggaran penyaring sekaligus.
func (f InboxFilter) Validate() error {
	var violations []Violation

	if f.Kind != "" && !f.Kind.Valid() {
		violations = append(violations, Violation{
			Field:   FieldInboxKind,
			Message: "Kotak yang diminta tidak dikenali.",
		})
	}
	if f.DateRangeInverted() {
		violations = append(violations, Violation{
			Field:   FieldDateTo,
			Message: "Tanggal sampai tidak boleh lebih awal daripada tanggal dari.",
		})
	}
	return NewValidationError(violations)
}

// InboxPage adalah satu halaman inbox beserta jumlah seluruh yang cocok.
type InboxPage struct {
	Cases []CommitteeCase

	// Total adalah banyaknya baris yang cocok dengan penyaring — bukan banyaknya baris
	// pada halaman ini.
	//
	// Ia dihitung meski `docs/Steering/10-API-STRATEGY.md` §4 menganjurkan menghindari
	// COUNT(*) pada data besar. Alasannya: inbox komite dibatasi pada pekerjaan MILIK
	// SATU ORANG yang belum selesai, dan itu puluhan baris — bukan puluhan juta. Yang
	// dilarang §4 adalah menghitung seluruh tabel, bukan menghitung antrean satu orang.
	Total int
}

// InboxSummary adalah jumlah baris per kotak, untuk lencana pada tab.
//
// Dihitung dalam satu perjalanan yang sama dengan isi tabelnya, sehingga angka lencana
// tidak dapat berselisih dengan isi tabel di bawahnya.
type InboxSummary struct {
	Outstanding int
	Accepted    int
	Rejected    int
}

// Count mengembalikan jumlah untuk satu kotak.
func (s InboxSummary) Count(kind InboxKind) int {
	switch kind {
	case InboxOutstanding:
		return s.Outstanding
	case InboxAccepted:
		return s.Accepted
	case InboxRejected:
		return s.Rejected
	default:
		return 0
	}
}

// InboxRepo adalah seam ke sumber kasus komite.
//
// # Sumbernya TABEL WARISAN, dan hanya dibaca
//
// Keputusan Work Owner 2026-09-20: inbox dibaca dari tabel milik sistem lama —
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`, `DATAPEGA.PC_ASSIGN_WORKLIST`,
// `POOLDATA.T_CLAIM_KOMITE_LIST`, dan `POOLDATA.T_CLAIM_DATA_RESULTS_AI`.
//
// Keputusan Work Owner 2026-10-08 mencabut `PC_ASM_FW_GCNMFW_WORK` dari daftar itu: tabel
// kerja Pega sudah tidak dipakai, dan tidak satu pun kueri di repo/sqlstore membacanya lagi.
//
// `P-1` terpenuhi tanpa negosiasi kepemilikan karena tidak ada satu pun pernyataan yang
// menulis ke sana. Yang ditulis aplikasi ini adalah tabel keputusannya SENDIRI — lihat
// DecisionRepo.
//
// # Kenapa penyaringan di sini dikerjakan SQL, berbeda dari master ambang
//
// Master ambang berisi 30 baris, sehingga membacanya utuh lalu menyaring di Go tidak
// berbiaya dan membuat aturannya dapat diuji tanpa basis data. Kasus komite berbeda: ia
// tumbuh bersama jumlah klaim, dan membacanya utuh berarti mengirim seluruh antrean
// komite seluruh perusahaan ke memori aplikasi untuk menampilkan satu halaman.
//
// Yang TETAP di Go: penurunan Tipe Komite, perhitungan Aging, dan seluruh aturan
// keputusan. Yang diserahkan ke SQL hanyalah penyaringan dan paginasi — bukan aturan.
type InboxRepo interface {
	ListCases(ctx context.Context, f InboxFilter) (InboxPage, error)

	// Summarize menghitung isi ketiga kotak untuk satu operator dalam satu perjalanan.
	Summarize(ctx context.Context, f InboxFilter) (InboxSummary, error)

	// FindCase mengambil satu kasus tanpa memandang kotaknya.
	//
	// Ia TIDAK menyaring pemilik. Pemeriksaan kepemilikan dikerjakan lapisan usecase
	// lewat BelongsTo, supaya perbedaan antara "tidak ada" dan "bukan milik Anda" dapat
	// dijawab dengan jujur — dan supaya aturannya tidak tersembunyi di dalam WHERE.
	//
	// operator menentukan baris ANGGOTA mana yang dikembalikan bila case itu punya beberapa
	// anggota: baris milik operator itu lebih dulu, lalu anggota yang sedang ditunggu.
	FindCase(ctx context.Context, caseID, operator string) (CommitteeCase, error)
}

// InboxEarliestYear adalah tahun terawal case komite yang ditampilkan inbox.
//
// # Ia bukan angka karangan
//
// `Report Definition/InboxRegisterKomite_RD-RD.xml` menyaring dengan
// `pxYearNumber(.pxCreateDateTime) >= "2024"` sebagai penyaring `F1` pada logika
// `(A) AND B AND F1`. Tanpa penyaring itu layar ini menampilkan pekerjaan yang TIDAK
// pernah terlihat di Pega.
//
// Ukurannya nyata, bukan sepele: pada basis data ASM, penyaring A dan B meloloskan
// **417 baris**, dan penambahan F1 menurunkannya menjadi **189**. Melewatkannya berarti
// layar menampilkan lebih dari dua kali lipat pekerjaan yang seharusnya.
//
// # Kenapa ia konstanta di sini, dan kenapa itu belum selesai
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh di-hardcode. Angka ini melanggarnya,
// dan pelanggarannya disadari: yang menentukan sekarang adalah `P-5` — perilaku
// dipertahankan lebih dulu — dan memindahkannya ke master data berarti mengubah perilaku
// tanpa keputusan tertulis.
//
// Ditaruh di SATU tempat dan dikirim sebagai parameter, bukan ditanam di dalam teks SQL,
// supaya pemindahannya ke master data kelak menyentuh satu baris. Dicatat sebagai
// pertanyaan terbuka di `docs/keputusan-implementasi.md`.
const InboxEarliestYear = 2024

// InboxEarliestCreatedAt mengembalikan batas bawah `PXCREATEDATETIME` untuk inbox.
//
// # Kenapa batas waktu, bukan perbandingan tahun
//
// `EXTRACT(YEAR FROM kolom) >= 2024` membungkus kolomnya dengan fungsi, dan kolom yang
// terbungkus fungsi tidak dapat memakai indeks. Batas bawah waktu memberi hasil yang sama
// persis — tahun 2024 dimulai tepat pada 1 Januari 2024 pukul 00:00 — sambil tetap dapat
// memakai indeks pada tabel yang besar.
//
// UTC, bukan WIB, karena `PXCREATEDATETIME` disimpan Pega dalam GMT (`R-12`, `F-5`).
func InboxEarliestCreatedAt() time.Time {
	return time.Date(InboxEarliestYear, time.January, 1, 0, 0, 0, 0, time.UTC)
}
