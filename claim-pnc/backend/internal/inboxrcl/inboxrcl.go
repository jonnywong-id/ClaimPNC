// Package inboxrcl adalah inti modul Inbox RCL.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `MENU_ID 62` berbunyi **"Inbox RCL"**, dan `Harness/RCL_Harness-Harness.xml` memuat judul
// yang sama persis (`pyCaption Inbox RCL`). `D-81` menetapkan nama modul mengikuti nama yang
// dipakai Work Owner.
//
// # JANGAN tertukar dengan Inbox RCL/PUCL (`MENU_ID 61`, modul `inboxrclpucl`)
//
//	Inbox RCL/PUCL (61)  antrean BERSAMA `RCLPUCL` — tiga tab perjalanan surat
//	Inbox RCL      (62)  antrean PER ORANG — dokter RCL, tahap `RCLDokter`
//
// Keduanya kini membaca tabel yang SAMA, `POOLDATA.TC_PNC_PUCL`, dengan penyaring berbeda.
//
// # Artefak Pega yang dibaca
//
//	Harness/RCL_Harness-Harness.xml                  judul layar, 5 judul kolom
//	Section/InboxRCLDokter_Section-Section.xml       grid, 5 sel berkepala, tautan baris
//	Report Definition/InboxRCLDokter_RD-RD.xml       penyaring, urutan, isian
//	Flow Action/SendToRCLDokter-FA.xml               layar kerja = section `RCLDokter`
//	Section/RCLDokter-Section.xml                    layar kerja dokter RCL, dua mode
//	Activity/RouterRCLDokter-act.xml                 `Param.AssignTo := ClaimData.NamaDokterRCL`
//	Flow/Register_Flow.xml                           tahap `Assignment12`
//
// # Apa itu Inbox RCL
//
// Antrean **penolakan yang memerlukan pertimbangan medis**: klaim yang dikirim analis ke
// dokter RCL. Ia benar-benar Inbox menurut `D-79` — barisnya tugas milik satu orang, dan
// barisnya hilang begitu tugasnya selesai.
//
// # SUMBER DATANYA `POOLDATA.TC_PNC_PUCL`
//
// Work Owner menetapkan 2026-10-05: daftar dan layar kerja `RCLDokter` membaca
// `POOLDATA.TC_PNC_PUCL` — tabel yang menyimpan `ClaimData.PUCLStatus` satu baris per klaim,
// dan yang juga dibaca Inbox RCL/PUCL. Keempat penyaring `InboxRCLDokter_RD`
// (`A AND B AND C AND D`) dipetakan ke kolomnya:
//
//	A  newAssignPage.pxAssignedOperatorID = assign  ->  ASSIGNED_OPERATOR_ID = operator
//	B  .pyStatusWork != "Resolved-Completed"         ->  STATUS_WORK <> 'Resolved-Completed'
//	C  .ClaimData.TanggalAnalystSendRCL IS NOT NULL  ->  TGL_KIRIM_PUCL IS NOT NULL
//	D  .ClaimData.NamaDokterRCL = assign             ->  RCL_PUCL IN ('1','3')
//
// Dasar pemetaan C: `Activity/SendToPUCL-Act.xml` langkah 8 mengisi `TanggalAnalystSendRCL`
// dan `PUCLStatus.TanggalKirimPUCL` dengan `@CurrentDateTime()` pada langkah yang sama.
//
// Dasar pemetaan D: tidak ada kolom nama dokter. `RouterRCLDokter` menugaskan klaim ke
// `NamaDokterRCL`, sehingga dokter = pemilik penugasan (sudah dijaga A). Yang tersisa dari D
// adalah "klaim ini memang punya dokter RCL" — hanya jalur RCL (`1`) dan MSIG (`3`) yang
// melewati dokter; PUCL (`2`) tidak. Tanpa saringan itu, baris PUCL milik orang yang sama
// ikut masuk antrean ini.
//
// # Operator = `POOLDATA.M_LOGIN_PNC.LOGIN_ID`
//
// Di Pega `assign = TempOperator.City` (identitas lama dari `T_ACCESS_GROUP_PNC`). Work Owner
// menetapkan 2026-10-05: tabel itu tidak dipakai lagi; login aktif di `M_LOGIN_PNC` dipakai
// langsung.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxrcl

import (
	"context"
	"strings"
	"time"
)

// StatusKerjaSelesai adalah nilai `STATUS_WORK` yang MENGELUARKAN tugas dari antrean ini.
//
// Hanya `Resolved-Completed` yang dikecualikan; `Resolved-Rejected` TIDAK — isi Report
// Definition apa adanya.
const StatusKerjaSelesai = "Resolved-Completed"

// LoginAktif adalah nilai `M_LOGIN_PNC.ACTIVE_STATUS` bagi login yang masih berlaku.
const LoginAktif = "1"

// Mode adalah `PUCLStatus.RCL_PUCL` — jalur yang menentukan isi layar kerja `RCLDokter`.
type Mode string

// Ketiga nilai `RCL_PUCL`. Hanya RCL dan MSIG yang melewati dokter RCL.
const (
	ModeRCL  Mode = "1"
	ModePUCL Mode = "2"
	ModeMSIG Mode = "3"
)

// DoctorModes adalah nilai `RCL_PUCL` yang termasuk antrean ini — padanan penyaring D.
var DoctorModes = []Mode{ModeRCL, ModeMSIG}

// IsDoctorMode menyatakan apakah sebuah mode melewati dokter RCL.
func IsDoctorMode(m Mode) bool {
	for _, d := range DoctorModes {
		if m == d {
			return true
		}
	}
	return false
}

// RCLTask adalah satu baris pada layar — satu tugas penolakan medis yang menunggu.
type RCLTask struct {
	// ClaimNumber — kolom **"Nomor Case"** <- `TC_PNC_PUCL.CLAIMID` (nomor klaim, mis.
	// `PNCN.26.31`). Juga kunci urutan kedua, menurun, dan kunci layar kerja.
	ClaimNumber string

	PolicyNumber string // "No Polis"         <- POLICY_NO
	InsuredName  string // "Nama Tertanggung" <- QQ_NAME

	// SentToRCLAt — kolom **"Tanggal Masuk Inbox"** <- `TGL_KIRIM_PUCL`.
	SentToRCLAt time.Time

	// AnalystNote — kolom **"Deskripsi Analyst"** <- `KOMENTAR_ANALISATOR`.
	AnalystNote string

	// Mode adalah `RCL_PUCL`. Tidak digambar; penyaring D.
	Mode Mode

	// RegisteredAt adalah `TGL_CREATE_PUCL`, kunci urutan pertama (padanan `.pxCreateDateTime`).
	RegisteredAt time.Time

	// ProcessStatus adalah `STATUS_WORK`. Tidak digambar; penyaring B.
	ProcessStatus string

	// AssignedOperator adalah `ASSIGNED_OPERATOR_ID`. Tidak digambar; penyaring A.
	AssignedOperator string
}

// RCLDetail adalah isi layar kerja `RCLDokter` untuk satu klaim — seluruhnya dari
// `TC_PNC_PUCL`.
type RCLDetail struct {
	ClaimNumber  string
	PolicyNumber string
	InsuredName  string

	// Mode menentukan isi layar (`Section/RCLDokter-Section.xml`):
	//
	//	ModeRCL   Catatan dari Analyst · Alasan Klaim Ditolak/RCL · tombol Setuju / Tidak Setuju
	//	ModeMSIG  Catatan dari Analyst · Alasan Klaim MSIG        · tombol Back / Submit
	Mode Mode

	// AnalystNote — "Catatan dari Analyst" <- `KOMENTAR_ANALISATOR`.
	AnalystNote string

	// Reason — "Alasan Klaim Ditolak/RCL" (mode RCL) atau "Alasan Klaim MSIG" (mode MSIG).
	// Keduanya properti yang sama di Pega, `PUCLStatus.Keterangan2` <- `KETERANGAN2`.
	Reason string

	// DoctorReason — "Alasan Dokter" pada layar Tidak Setuju/Back
	// (`sendToAnalystTolakRCL_sect`) <- `ALASAN_DOKTER_REJECT_RCL`.
	DoctorReason string

	StatusClaim      string // STATUS_CLAIM
	ProcessStatus    string // STATUS_WORK
	AssignedOperator string // ASSIGNED_OPERATOR_ID
	SentToRCLAt      time.Time
}

// Caller adalah identitas pemanggil sebagaimana dibutuhkan modul ini.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — kunci pencarian di
	// `M_LOGIN_PNC.LOGIN_ID`.
	Login string
}

// Filter adalah penyaring dan paginasi yang diminta layar.
//
// Report Definition tidak punya penyaring yang dapat diubah pengguna; kotak cari adalah
// tambahan yang sama dengan inbox lain.
type Filter struct {
	// Search mencari pada Nomor Case DAN No Polis sekaligus, di SERVER.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi, sama dengan inbox lain di aplikasi ini.
const (
	DefaultLimit   = 25
	MaxLimit       = 100
	PegaMaxRecords = 500
)

// Normalize mengembalikan filter dengan nilai yang dijamin masuk akal.
func (f Filter) Normalize() Filter {
	f.Search = strings.TrimSpace(f.Search)

	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	return f
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Tasks []RCLTask
	Total int
}

// Repo adalah seam ke penyimpanan antrean RCL Dokter.
type Repo interface {
	// OperatorFor membaca `LOGIN_ID` seorang petugas dari `M_LOGIN_PNC`. Login yang tidak
	// ada atau tidak aktif mengembalikan string kosong TANPA galat.
	OperatorFor(ctx context.Context, loginID string) (string, error)

	// List mengambil satu halaman antrean milik seorang operator.
	List(ctx context.Context, operator string, f Filter) (Page, error)

	// Detail mengambil isi layar kerja satu klaim milik seorang operator.
	//
	// Klaim yang tidak ada, BUKAN milik operator itu, atau tidak lolos penyaring antrean
	// mengembalikan ErrClaimNotFound — layar kerja hanya terbuka bagi klaim yang memang
	// tampil di antrean pemanggil.
	Detail(ctx context.Context, operator, claimNumber string) (RCLDetail, error)

	// Decide menjalankan keputusan dokter RCL atas satu klaim di antrean operator — padanan
	// `SendToPUCL` — dalam SATU transaksi: baris TC_PNC_PUCL dikunci, Plan dijalankan,
	// TC_PNC_PUCL dan T_CLAIMLIST_ADMIN ditulis, tugas dipindahkan, riwayat dicatat.
	//
	// Klaim yang tidak lagi di antrean operator (sudah diputus di tab lain, atau milik orang
	// lain) mengembalikan ErrClaimNotFound dan TIDAK menulis apa pun.
	Decide(ctx context.Context, cmd DecisionCommand) (Outcome, error)
}

// DecisionCommand adalah satu keputusan dokter RCL.
type DecisionCommand struct {
	Operator     string // LOGIN_ID pemanggil — pemilik penugasan dan pelaku riwayat
	ClaimNumber  string
	Decision     Decision
	DoctorReason string // isian "Alasan Dokter"; hanya dipakai Tidak Setuju/Back
	At           time.Time
}

// RepoSelector memilih Repo milik satu portal entitas. Portal yang tidak dikenal
// menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
