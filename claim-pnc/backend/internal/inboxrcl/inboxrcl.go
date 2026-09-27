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
// Namanya nyaris sama, isinya berbeda sama sekali:
//
//	Inbox RCL/PUCL (61)  antrean BERSAMA `RCLPUCL` — workbasket, tiga tab perjalanan surat
//	Inbox RCL      (62)  antrean PER ORANG — worklist dokter RCL, tahap `RCLDokter`
//
// Pega pun memisahkannya menjadi dua harness, dua section, dan dua Report Definition.
//
// # Artefak Pega yang dibaca
//
// Layar ini LENGKAP di export:
//
//	Harness/RCL_Harness-Harness.xml                     judul layar, 5 judul kolom,
//	                                                    activity pemuat `GetpyUserIdentifierFromTable`
//	Section/InboxRCLDokter_Section-Section.xml          grid, 5 sel berkepala, tautan baris,
//	                                                    parameter `assign = TempOperator.City`
//	Report Definition/InboxRCLDokter_RD-RD.xml          SELURUH penyaring, urutan, isian
//	Activity/GetpyUserIdentifierFromTable-Act.xml       dari mana `TempOperator.City` datang
//	RDB List/GetOperatorID-SQL.xml                      kueri identitas lama
//	Activity/SetAssignmentInboxRCLDoctor_act-Act.xml    aksi klik baris
//	When/IsRCLPA-When.xml                               siapa yang melihat menunya
//	Flow/Register_Flow.xml                              dari mana tugasnya datang (`Assignment12`)
//
// # Apa itu Inbox RCL
//
// Antrean **penolakan yang memerlukan pertimbangan medis**. `CONTEXT.md` mendefinisikan RCL
// Dokter sebagai penolakan klaim lini Personal Accident yang menuntut penilaian dokter, dan
// `Flow/Register_Flow.xml` menempatkannya sebagai `Assignment12` — salah satu dari empat tahap
// penutup jalur analis, ditugaskan ke worklist oleh router `RouterRCLDokter`.
//
// Ia benar-benar Inbox menurut `D-79`: barisnya **tugas** milik satu orang, barisnya
// **hilang** begitu tugasnya selesai (`pyStatusWork != Resolved-Completed`), "hanya milik saya"
// adalah **aturan kewenangan**, dan barisnya menempuh penugasan — bukan data acuan.
//
// # Penyaringnya — keempatnya dari Report Definition, bukan dikarang
//
// `InboxRCLDokter_RD` menyatakan `pyFilterLogic = "A AND B AND C AND D"` atas:
//
//	A  newAssignPage.pxAssignedOperatorID  =   Param.assign
//	B  .pyStatusWork                       !=  "Resolved-Completed"
//	C  .ClaimData.TanggalAnalystSendRCL    IS NOT NULL
//	D  .ClaimData.NamaDokterRCL            =   Param.assign
//
// ditambah gabungan `INNER JOIN Assign-Worklist ON pxRefObjectKey = .pzInsKey` (di sini digantikan tabel datar — lihat di bawah) dan urutan
// `.pxCreateDateTime DESC, .pyID DESC`.
//
// # SATU HAL YANG MEMBEDAKANNYA DARI SELURUH INBOX LAIN: `Param.assign` BUKAN LOGIN
//
// Section-nya mengisi `assign` dengan **`TempOperator.City`**, dan harness-nya memuat
// activity `GetpyUserIdentifierFromTable` sebelum grid digambar. Activity itu menjalankan
// `RDB List/GetOperatorID-SQL.xml`:
//
//	SELECT OLD_OPERATOR_ID AS "City" FROM POOLDATA.T_ACCESS_GROUP_PNC
//	 WHERE OPERATOR_ID = {OperatorID.pyUserIdentifier} AND STS_AKTIF = '1'
//	   AND ACCESS_GROUP IN ('GCNMFW:Administrators','GCNMFW:PNCKomite','GCNMFW:CaseManager')
//	   AND ACCESS_GROUP != 'GCNMFW:ViewClaimPNC'
//
// lalu `TempOperator.City := TempOPID.pxResults(1).City` — TANPA cadangan. Jadi antrean ini
// disaring dengan **identitas LAMA** pemanggil, dan hanya identitas lama yang tercatat pada
// salah satu dari tiga grup akses itu. Pengguna tanpa baris seperti itu melihat antrean
// KOSONG di Pega.
//
// Perilaku itu dibawa apa adanya (`P-5`), dengan satu perbaikan pada cara ia DILAPORKAN:
// antrean yang kosong karena identitas lama tidak ditemukan dinyatakan ke layar sebagai
// keadaan tersendiri, tidak disamarkan sebagai "tidak ada pekerjaan". Lihat IdentityResult.
//
// # SUMBER DATANYA `POOLDATA.T_CLAIMLIST_ADMIN`, BUKAN TABEL PEGA
//
// Work Owner menetapkan 2026-09-27: modul ini tidak membaca `DATAPEGA` lagi. Report
// Definition aslinya menggabung objek kerja dengan worklist; tabel datar menyimpan satu baris
// per klaim beserta pemilik penugasan yang sedang berjalan, sehingga gabungannya hilang.
//
// Tiga isian layar ini — `TanggalAnalystSendRCL` (penyaring C dan kolom "Tanggal Masuk
// Inbox"), `NamaDokterRCL` (penyaring D), dan `KomentarAnalisator` ("Deskripsi Analyst") —
// belum punya kolom di tabel itu, dan dua yang pertama bahkan `unexposed` di Pega. Ketiganya
// diajukan lewat `migrations/0012_claimlist_admin_rcl.up.sql`, dijalankan DBA (`D-63`).
// Lihat kepala `repo/sqlstore/inboxrcl.sql`.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxrcl

import (
	"context"
	"strings"
	"time"
)

// StatusKerjaSelesai adalah nilai `PYSTATUSWORK` yang MENGELUARKAN tugas dari antrean ini.
//
// Penyaringnya `!=`, bukan `=`. Hanya `Resolved-Completed` yang dikecualikan;
// `Resolved-Rejected` TIDAK — klaim yang ditolak tetap muncul. Itu isi Report Definition apa
// adanya, sama seperti `inboxanalystdoctor` dan berbeda dari `inboxoutstanding`.
const StatusKerjaSelesai = "Resolved-Completed"

// LegacyAccessGroups adalah ketiga grup akses yang identitas lamanya boleh menjadi pemilik
// antrean ini, dan grup yang dikecualikan.
//
// Asalnya `TempOperator.AlasanKlaim` pada langkah pertama
// `Activity/GetpyUserIdentifierFromTable-Act.xml`, yang disisipkan ke kueri lewat
// `{Asis:...}`. Di sini nilainya menjadi konstanta dan dikirim lewat parameter binding —
// perangkaian teks SQL tidak dibawa (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// Ketiganya sama dengan grup yang disebut `When/IsRCLPA-When.xml` sebagai penjaga menu:
// `(Administrators OR (PNCKomite AND PA) OR (CaseManager AND PA)) AND NOT ViewClaimPNC`.
// Kesamaan itu bukan kebetulan — dokter RCL di sistem lama adalah anggota komite PA.
var LegacyAccessGroups = []string{
	"GCNMFW:Administrators",
	"GCNMFW:PNCKomite",
	"GCNMFW:CaseManager",
}

// ExcludedAccessGroup dikecualikan meski pemanggil juga memegang salah satu grup di atas.
const ExcludedAccessGroup = "GCNMFW:ViewClaimPNC"

// RCLTask adalah satu baris pada layar — satu tugas penolakan medis yang menunggu.
type RCLTask struct {
	// ClaimID adalah kunci teknis `PZINSKEY`. Tidak digambar sebagai kolom; ia dipakai
	// tautan baris dan sebagai kunci baris di layar.
	ClaimID string

	// ClaimNumber — kolom **"Nomor Case"** <- PYID. Juga kunci urutan kedua, menurun.
	ClaimNumber string

	PolicyNumber string // "No Polis"         <- POLICYNO
	InsuredName  string // "Nama Tertanggung" <- QQNAME

	// SentToRCLAt — kolom **"Tanggal Masuk Inbox"** <- `.ClaimData.TanggalAnalystSendRCL`.
	//
	// Judul kolomnya memang "Tanggal Masuk Inbox", bukan nama propertinya: ia waktu analis
	// mengirim klaim ke dokter RCL. Propertinya TIDAK TEREKSPOS — lihat kepala paket.
	SentToRCLAt time.Time

	// AnalystNote — kolom **"Deskripsi Analyst"** <- `.ClaimData.PUCLStatus.KomentarAnalisator`.
	//
	// Label Report Definition-nya "Komentar Analisator"; judul yang dilihat pengguna di
	// harness "Deskripsi Analyst". Yang dipakai judul harness (`D-13`).
	AnalystNote string

	// RCLDoctor adalah `.ClaimData.NamaDokterRCL`. Tidak digambar; ia penyaring D.
	RCLDoctor string

	// RegisteredAt adalah `PXCREATEDATETIME`, kunci urutan pertama. Tidak digambar —
	// kelima judul kolom di harness tidak memuatnya.
	RegisteredAt time.Time

	// ProcessStatus adalah `PYSTATUSWORK`. Tidak digambar; ia penyaring B.
	ProcessStatus string

	// AssignedOperator adalah `PXASSIGNEDOPERATORID`. Tidak digambar; ia penyaring A.
	AssignedOperator string
}

// Caller adalah identitas pemanggil sebagaimana dibutuhkan modul ini.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK. Ia padanan
	// `OperatorID.pyUserIdentifier`, kunci pencarian identitas lamanya.
	Login string
}

// Filter adalah penyaring dan paginasi yang diminta layar.
//
// Report Definition tidak punya satu pun penyaring yang dapat diubah pengguna. Kotak cari di
// bawah ini adalah TAMBAHAN yang disadari — dinyatakan ke pengguna sebagai selisih terencana,
// sama seperti di Inbox Analyst Doctor.
type Filter struct {
	// Search mencari pada Nomor Case DAN No Polis sekaligus, di SERVER.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi, sama dengan inbox lain di aplikasi ini.
//
// `InboxRCLDokter_RD` memakai `pyMaxRecords = 500`; batas itu TIDAK ditegakkan (`ADR-0011`).
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
//
// Ia hanya MEMBACA. Menyelesaikan tugas RCL Dokter berarti menjalankan Flow Action
// `SendToRCLDokter`, yang menulis objek kerja DAN memindahkan penugasannya — keduanya milik
// Pega selama masa paralel (`P-1`).
type Repo interface {
	// LegacyOperatorFor membaca identitas LAMA seorang petugas — padanan
	// `GetOperatorID-SQL.xml` dengan ketiga grup akses `LegacyAccessGroups`.
	//
	// Petugas yang tidak punya baris mengembalikan string kosong TANPA galat: di Pega ia
	// menghasilkan antrean kosong, bukan kegagalan.
	LegacyOperatorFor(ctx context.Context, loginID string) (string, error)

	// List mengambil satu halaman antrean milik sebuah identitas lama.
	//
	// Operator diserahkan terpisah dari Filter: ia batas kewenangan, bukan penyaring pilihan.
	List(ctx context.Context, operator string, f Filter) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Portal yang tidak dikenal menghasilkan galat — TIDAK PERNAH dialihkan ke koneksi utama
// (`R-20`, `TKT-F6-002`). Barisnya adalah klaim Personal Accident, dan `FR-R2` membatasi
// akses data medis pada peran Analyst Doctor dan RCL Dokter.
type RepoSelector func(portalAlias string) (Repo, error)
