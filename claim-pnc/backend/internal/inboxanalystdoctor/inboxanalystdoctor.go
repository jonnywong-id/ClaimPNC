// Package inboxanalystdoctor adalah inti modul Inbox Analyst Doctor.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari layar lamanya sendiri: butir menu
// `MENU_ID 60` pada `Database/m_menu_aplikasi_pnc.csv` berbunyi **"Inbox Analyst Doctor"**,
// dan `Harness/inboxAnalystDoctor_Harness-Harness.xml` memuat judul yang sama persis di
// dalam layarnya (`pyCaption Inbox Analyst Doctor`). `D-81` menetapkan nama modul mengikuti
// nama yang dipakai Work Owner.
//
// # Artefak Pega yang dibaca
//
// Berbeda dari `inboxcloseclaim` dan `inboxoutstanding` yang harness-nya hilang, layar ini
// LENGKAP di export — dan itu membuat hampir seluruh bentuknya terbaca dari bukti, bukan
// disusun ulang:
//
//	Harness/inboxAnalystDoctor_Harness-Harness.xml      judul layar, 8 judul kolom
//	Section/InboxAnalystDoctor_Section-Section.xml      grid, 8 sel kepala, 1 tautan baris
//	Report Definition/InboxAnalystDoctor_RD-RD.xml      SELURUH penyaring, urutan, isian
//	When/IsAnalystDoctor-When.xml                       siapa yang melihat menunya
//	Flow/Register_Flow.xml                              dari mana tugasnya datang
//	Flow Action/SendAnalystDoctor-FA.xml                aksi yang mengeluarkan tugas
//
// # Apa itu Inbox Analyst Doctor
//
// Antrean **penilaian medis**. `CONTEXT.md` mendefinisikan Analyst Doctor sebagai tenaga
// medis yang menilai klaim kecelakaan diri dan kesehatan, dan `Flow/Register_Flow.xml`
// menempatkannya sebagai `Assignment13` — salah satu dari empat tahap penutup jalur analis.
//
// Ia benar-benar Inbox menurut `D-79`, dan keempat cirinya terpenuhi: barisnya adalah
// **tugas** milik satu orang (penyaring `pxAssignedOperatorID`), barisnya **hilang** begitu
// tugasnya selesai (penyaring `pyStatusWork != Resolved-Completed`), "hanya milik saya"
// adalah **aturan kewenangan** dan bukan sekadar penyaring, dan barisnya menempuh
// penugasan — bukan data acuan.
//
// # Penyaringnya — ketiganya dari Report Definition, bukan dikarang
//
// `InboxAnalystDoctor_RD` menyatakan `pyFilterLogic = "A AND B AND C"` atas:
//
//	A  .ClaimData.isComplianceTransfer  =   "2"
//	B  newAssignPage.pxAssignedOperatorID =  Param.assign
//	C  .pyStatusWork                    !=  "Resolved-Completed"
//
// ditambah gabungan `INNER JOIN Assign-Worklist ON pxRefObjectKey = .pzInsKey` dan urutan
// `.pxCreateDateTime DESC, .pzInsKey DESC`.
//
// # SATU HAL YANG HARUS DISADARI SEBELUM MEMBACA SISA BERKAS INI
//
// Dua properti yang dipakai layar ini ditandai Pega sendiri sebagai TIDAK TEREKSPOS:
//
//	<pzPropertyType>unexposed</pzPropertyType>
//
// yaitu `.ClaimData.isComplianceTransfer` — **penyaring utamanya** — dan
// `.ClaimData.AnalystDoctorRemaks`, kolom "Komentar dari PIC Teknis". Properti tak terekspos
// hidup di dalam blob Pega, bukan sebagai kolom SQL; Pega tetap dapat menyaringnya karena ia
// memuat blob lalu menyaring di memori, dan Go tidak dapat menempuh jalan itu.
//
// Work Owner menjawab 2026-09-23: **nilainya langsung di-set 2**. Jadi ia memang nilai yang
// tersimpan, dan yang dibutuhkan hanyalah kolomnya. Nama kolom yang dipakai di sini
// mengikuti konvensi `_1` yang berlaku pada seluruh properti `ClaimData` lain di tabel yang
// sama (`STATUSKLAIM_1`, `RCL_PUCL_1`, `DATEOFLOSS_1`, `LAMAKLAIM_1`), dan **wajib
// dikonfirmasi DBA** — lihat kepala `repo/sqlstore/inboxanalystdoctor.sql`.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxanalystdoctor

import (
	"context"
	"strings"
	"time"
)

// TransferAnalystDoctor adalah nilai `ClaimData.isComplianceTransfer` yang menempatkan
// sebuah klaim di antrean ini.
//
// # Kenapa ia konstanta, bukan angka di dalam SQL
//
// `D-15` menetapkan tidak ada nilai bisnis yang boleh tertanam berulang kali di dalam kode.
// Di sini alasannya lebih tajam lagi: nilai yang sama menempatkan klaim di antrean yang
// BERBEDA, dan satu digit yang salah memindahkan seluruh isi layar tanpa satu pun galat.
//
//	"1"  Compliance      — `When/IsCompliance-When.xml`, dipakai `InboxRegisterCompliance_RD`
//	"2"  Analyst Doctor  — `Report Definition/InboxAnalystDoctor_RD-RD.xml`
//
// Keduanya ditulis berdampingan di sini justru supaya perbedaannya tidak dapat terlewat saat
// membaca. `TransferCompliance` sendiri TIDAK dipakai modul ini; ia ada sebagai pembanding.
const (
	TransferCompliance    = "1"
	TransferAnalystDoctor = "2"
)

// StatusKerjaSelesai adalah nilai `PYSTATUSWORK` yang MENGELUARKAN tugas dari antrean ini.
//
// Penyaringnya `!=`, bukan `=`. Satu tanda yang salah di sini membalik seluruh isi layar:
// yang tampil menjadi tugas yang sudah tuntas, dan tidak ada apa pun di layar yang
// menandakannya.
//
// Perhatikan ia HANYA menyebut `Resolved-Completed`. `Resolved-Rejected` TIDAK dikecualikan,
// sehingga klaim yang ditolak TETAP muncul di antrean ini. Itu perilaku Report Definition-nya
// apa adanya, dan berbeda dari `inboxoutstanding` yang mengecualikan keduanya — perbedaan
// yang dibawa (`P-5`), bukan diseragamkan.
const StatusKerjaSelesai = "Resolved-Completed"

// AnalystDoctorTask adalah satu baris pada layar — satu tugas penilaian medis yang menunggu.
//
// Field di sini adalah apa yang dibaca layar, bukan salinan utuh klaim. Modul ini antrean
// pemantauan; isi klaim yang sesungguhnya milik modul `registrasi` dan modul nilai.
type AnalystDoctorTask struct {
	// ClaimID adalah kunci teknis `PZINSKEY`.
	//
	// Isinya berbentuk `ASM-FW-GCNMFW-WORK PNC-xxxx`: nama kelas internal Pega tertanam di
	// dalam kunci data bisnis — utang teknis §4.1 yang `D-22` dan `D-71` hapus untuk klaim
	// baru. Ia TIDAK digambar sebagai kolom; yang memakainya hanyalah pengurutan pemutus
	// seri, persis seperti `pySortOrder` kedua pada Report Definition.
	ClaimID string

	// ClaimNumber — kolom **"Nomor Case"** <- PYID.
	//
	// Judul kolomnya memang "Nomor Case", bukan "Nomor Klaim". Itu teks layar lama dan
	// `D-13` menetapkan teks yang dilihat pengguna mengikutinya apa adanya.
	ClaimNumber string

	PolicyNumber string // "No Polis"           <- POLICYNO
	InsuredName  string // "Nama Tertanggung"   <- QQNAME
	BranchName   string // "Nama Cabang"        <- BRANCHNAME
	AdminName    string // "Nama Admin"         <- PYORIGUSERID

	// TechnicalPICNote — kolom **"Komentar dari PIC Teknis"**.
	//
	// Asalnya `.ClaimData.AnalystDoctorRemaks`, dan properti itu **tidak terekspos** — lihat
	// kepala paket. Kolomnya tetap DIGAMBAR, tidak dihilangkan: isian yang belum terbawa
	// harus terlihat, bukan tersamar sebagai layar yang sudah setara. Preseden yang sama
	// dipakai `inboxmanagerreceivepucl` pada "Jumlah Lembar Dokumen".
	//
	// Catatan penamaan: harness memuat DUA rule caption yang nyaris sama — "Komentar dari
	// PIC Teknis" dan "Komentar dari PIC Tekniks". Yang kedua salah ketik. Yang dipakai
	// bentuk yang benar; salah ketiknya tidak ikut dibawa, sejalan dengan `Broswse*` dan
	// `Complience` yang juga tidak dibawa (§4.7).
	TechnicalPICNote string

	// TechnicalPIC adalah `.ClaimData.UserTeknis`, label RD "Nama PIC Teknik".
	//
	// # Ia dibaca tetapi TIDAK digambar sebagai kolom
	//
	// Report Definition menyediakannya, tetapi harness TIDAK punya caption untuknya —
	// kedelapan judul kolom di sana tidak memuat "Nama PIC Teknik". `D-13` menetapkan
	// tampilan mengikuti layar lama, sehingga menggambarnya berarti menambah kolom yang
	// tidak pernah ada.
	//
	// Ia tetap dibaca dan tetap dikirim ke layar karena RD memang mengambilnya, dan karena
	// kolom "Komentar dari PIC Teknis" di sebelahnya menjadi jauh lebih berguna bila
	// diketahui PIC mana yang menuliskannya.
	TechnicalPIC string

	// RegisteredAt — kolom **"Tanggal Pendaftaran"** <- PXCREATEDATETIME.
	//
	// Ia juga kunci urutan pertama, menurun. Bukan `REGISTERDATE_1`: keduanya ada di tabel
	// dan mudah tertukar, dan yang dibaca Report Definition adalah `.pxCreateDateTime`.
	RegisteredAt time.Time

	// ProcessStatus adalah `PYSTATUSWORK` — status ALUR KERJA, salah satu dari empat konsep
	// status yang `ADR-0018` larang digabung.
	//
	// Tidak digambar sebagai kolom; ia yang menentukan sebuah baris ADA di layar ini sama
	// sekali. Dibawa supaya jawaban API dapat menjelaskan dirinya sendiri saat ditelusuri.
	ProcessStatus string

	// AssignedOperator adalah `PXASSIGNEDOPERATORID` dari baris worklist-nya.
	//
	// Ia sama dengan pemanggil pada keadaan biasa — kuerinya memang menyaring dengan itu.
	// Dibawa karena tanpanya, antrean kosong tidak dapat dibedakan dari antrean milik orang
	// lain saat menelusuri satu keluhan.
	AssignedOperator string
}

// DurationDays adalah kolom **"Lama Waktu Klaim"** — sudah berapa hari tugas ini menunggu.
//
// # Dihitung, bukan dibaca dari kolom
//
// Report Definition layar ini **tidak mengambil** satu pun properti durasi; kedelapan
// isiannya tidak memuatnya. Kolom `LAMAKLAIM_1` memang ADA di tabel dengan 86 nilai berbeda
// (`docs/kolom-t-claimlist-admin.md` §B.3), tetapi TIDAK dibaca layar ini, dan artinya —
// dihitung sampai kapan, oleh siapa, dan diperbarui kapan — tidak terbaca dari export mana
// pun. Memakainya berarti menampilkan angka yang tidak dapat dipertanggungjawabkan.
//
// Preseden menghitungnya sudah ada dan sudah disetujui: `inboxcloseclaim.DurationDays`,
// keputusan Work Owner 2026-09-23. Bedanya di sini klaimnya MASIH BERJALAN, sehingga
// hitungannya berhenti di hari ini, bukan di tanggal tutup.
//
// # Kenapa terhadap TANGGAL, bukan selisih jam dibagi 24
//
// Tugas yang masuk pukul 23.00 dan dilihat pukul 01.00 keesokan harinya sudah berumur SATU
// HARI bagi pengguna, meski selisihnya dua jam. Membagi selisih jam akan mengembalikan nol.
//
// # Kenapa zona waktu ikut masuk
//
// Waktu disimpan UTC (`08-TECHNICAL-STRATEGY.md` §4.4) sedangkan "hari" yang dimaksud
// pengguna adalah hari WIB. Tanpa konversi, tugas yang masuk antara pukul 00.00 dan 07.00
// WIB dihitung satu hari lebih tua. Keduanya diserahkan pemanggil lewat parameter, bukan
// dibaca dari jam sistem — supaya dapat diuji tanpa bergantung mesin (`F-5`).
func (t AnalystDoctorTask) DurationDays(now time.Time, location *time.Location) int {
	if t.RegisteredAt.IsZero() {
		return 0
	}
	if location == nil {
		location = time.UTC
	}

	start := t.RegisteredAt.In(location)
	finish := now.In(location)

	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDay := time.Date(finish.Year(), finish.Month(), finish.Day(), 0, 0, 0, 0, location)

	days := int(endDay.Sub(startDay).Hours() / 24)
	if days < 0 {
		// Tanggal pendaftaran di masa depan adalah data yang cacat, bukan durasi negatif.
		// Ia ditampilkan nol; yang memperbaikinya adalah datanya, bukan layar ini.
		return 0
	}
	return days
}

// Caller adalah identitas pemanggil sebagaimana dibutuhkan modul ini.
//
// # Kenapa modul ini menuntutnya, sementara sebagian inbox lain tidak
//
// Karena penyaring B pada Report Definition membandingkan `pxAssignedOperatorID` dengan
// `Param.assign` — antrean ini milik SATU ORANG, bukan antrean bersama. Tanpa identitas,
// tidak ada antrean yang dapat ditampilkan; menampilkan seluruhnya akan memperlihatkan tugas
// medis milik petugas lain.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk, bukan NIK.
	//
	// Itulah yang dicocokkan ke `PC_ASSIGN_WORKLIST.PXASSIGNEDOPERATORID`. Memakai NIK di
	// sini akan membuat antrean tampak kosong bagi setiap pengguna, dan kosong adalah
	// jawaban yang tidak pernah ditanyakan siapa pun.
	Login string
}

// Filter adalah penyaring dan paginasi yang diminta layar.
//
// # Kenapa hanya SATU penyaring, dan itu bukan kelalaian
//
// `InboxAnalystDoctor_RD` tidak punya satu pun penyaring yang dapat diubah pengguna: ketiga
// filternya tetap, dan `pyEnableFilters` beserta saudaranya tidak menyalakan panel penyaring
// apa pun. Layar lamanya memang hanya daftar.
//
// Kotak cari di bawah ini karena itu **tambahan yang disadari**, bukan hasil pembacaan — dan
// ia dinyatakan ke pengguna sebagai selisih terencana, bukan disamarkan. Alasannya: paginasi
// di sistem baru dikerjakan basis data, sehingga tanpa pencarian sisi server pengguna
// kehilangan kemampuan menemukan satu klaim yang di Pega selalu ada di halaman klipboard
// yang sama.
type Filter struct {
	// Search mencari pada Nomor Case DAN No Polis sekaligus.
	//
	// Pencarian dikerjakan SERVER, bukan peramban: menyaring satu halaman dari sepuluh akan
	// memberi tahu pengguna bahwa sesuatu tidak ada padahal ia ada di halaman lain.
	Search string

	Limit  int
	Offset int
}

// Batas paginasi.
//
// # Kenapa 25, sementara Pega memakai 50
//
// `InboxAnalystDoctor_RD` menetapkan `pyPageSize = 50` dan `pyMaxRecords = 500`. Angka 50
// itu TIDAK dipakai sebagai bawaan di sini: ia ukuran halaman KLIPBOARD Pega — seluruh baris
// ditarik lebih dulu, dipotong di 500, lalu dinomori di memori. Di sini halamannya dipotong
// basis data sebelum baris meninggalkannya, sehingga angkanya menjawab pertanyaan yang
// berbeda.
//
// Yang dipakai 25, sama dengan inbox lain di aplikasi ini, supaya ukuran halaman tidak
// berbeda-beda antarlayar tanpa alasan. Batas maksimumnya mengikuti `10-API-STRATEGY.md` §4:
// permintaan yang lebih besar DITOLAK, bukan dipenuhi.
const (
	DefaultLimit = 25
	MaxLimit     = 100

	// PegaMaxRecords adalah `pyMaxRecords` layar lama.
	//
	// Ia TIDAK ditegakkan di sini — `ADR-0011` mencatat batas 500 pada 54 dari 56 laporan
	// sebagai pemotongan diam-diam, bukan paginasi. Angkanya disimpan supaya selisihnya
	// dapat dinyatakan ke pengguna, bukan supaya ditiru.
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
	Tasks []AnalystDoctorTask
	Total int
}

// Repo adalah seam ke penyimpanan tugas Analyst Doctor.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya. Diisi
// `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// Ia hanya MEMBACA, dan ketiadaan method tulis di sini disengaja. Menyelesaikan tugas
// Analyst Doctor berarti menjalankan Flow Action `SendAnalystDoctor`, yang menulis objek
// kerja DAN memindahkan penugasannya — keduanya milik Pega selama masa paralel (`P-1`).
// Membawanya ke sini menuntut modul alur kerja tersendiri, bukan satu method di antrean.
type Repo interface {
	// List mengambil satu halaman antrean milik seorang operator.
	//
	// Operator diserahkan terpisah dari Filter dengan sengaja: ia bukan penyaring yang
	// dipilih pengguna melainkan batas kewenangan, dan menaruhnya di dalam Filter akan
	// membuatnya terlihat seperti sesuatu yang boleh dikosongkan.
	List(ctx context.Context, operator string, f Filter) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas, bukan satu database bersama dengan
// penanda entitas. Klaim milik Asuransi Sinar Mas dan klaim milik Simas Insurtech karena itu
// tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan tugas medis satu
// badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`). Di layar ini taruhannya lebih besar daripada di inbox mana pun: barisnya
// menyangkut klaim Personal Accident, dan `FR-R2` membatasi akses data medis.
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu (`F-5`).
//
// Waktu TIDAK pernah dibaca langsung dari jam sistem di dalam modul:
// `08-TECHNICAL-STRATEGY.md` §4.4 menetapkan pembacaan waktu terpusat, dan seam ini yang
// membuat kolom "Lama Waktu Klaim" dapat diuji secara deterministik.
type Clock interface {
	Now() time.Time
}
