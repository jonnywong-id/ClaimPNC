// Package riwayatklaim adalah inti modul View History Claim.
//
// # Layar apa ini
//
// Menu `MENU_ID 76` "View History Claim" pada POOLDATA.M_MENU_APLIKASI_PNC, yang menunjuk
// harness `PNCSearchKlaim`. Judul yang dibaca pengguna di sistem lama adalah
// **"VIEW HISTORY CLAIM"**.
//
// Isinya **pencarian riwayat klaim lintas seluruh klaim** — pengguna memilih satu tipe
// pencarian, mengisi satu nilai, lalu menekan Cari. Ia BUKAN inbox: barisnya bukan
// pekerjaan, tidak hilang setelah dikerjakan, tidak punya tenggat, dan "hanya milik saya"
// tidak berlaku padanya. Pembedaan itu ditetapkan `D-79`, dan itulah sebabnya modul ini
// berdiri sendiri alih-alih menjadi varian layar inbox.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/PNCSearchKlaim-Harness.xml              pembungkus layar; judul layar
//	Section/PNCSearchKlaim-Section.xml              3 isian, tombol Cari, 2 grid, kondisi tampil
//	Activity/PNCSearchHistoryKlaim_Act-Act.xml      pemilih kueri per tipe + penyiapan parameter
//	Activity/InsertLogProteksiDataKlaimMasking-Act.xml  gerbang proteksi data + kuota
//	Activity/CekmaskingDataPerLoginUserKlaim-Act.xml    pembacaan master proteksi
//	Database/UPDATE_LOG_PROTEKSI.prc                isi log proteksi
//	RDB List/BroswseKlaimByPolicyNo-SQL.xml         tipe 1  No Polis
//	RDB List/BroswseKlaimByName-SQL.xml             tipe 2  Nama Customer
//	RDB List/BroswseKlaimByObjectName-SQL.xml       tipe 3  Nama Objek
//	RDB List/BroswseKlaimByPLANo-SQL.xml            tipe 4  No PLA
//	RDB List/BroswseKlaimByDLANo-SQL.xml            tipe 5  No DLA
//	RDB List/BroswseKlaimByDOL-SQL.xml              tipe 6  Tgl Kejadian
//	RDB List/BroswseKlaimByKlaimNo-SQL.xml          tipe 7  No Klaim
//	RDB List/BroswseKlaimByAcceptedNo-SQL.xml       tipe 8  No Akseptasi
//	RDB List/BroswseKlaimByBirthDate-SQL.xml        tipe 9  Tanggal Lahir
//	RDB List/BroswseKlaimByNoSurvey-SQL.xml         tipe 11 No Survey
//	RDB List/AmbilDataKlaimDenganNoRekening-SQL.xml tipe 12 No Rekening
//	RDB List/BroswseKlaimByIdBalaiLelang-SQL.xml    tipe 13 ID Balai Lelang
//
// # Kenapa nama field di sini tidak mirip nama properti Pega
//
// Karena nama properti Pega di layar ini MENYESATKAN SECARA AKTIF, dan bukan sekadar
// singkatan yang tidak lazim. Grid lama menampilkan `.EDMNO` untuk **No Klaim**,
// `.THEINSURED` untuk **Posisi Klaim**, dan `.SOBNAME` untuk **PIC Teknis** — tak satu pun
// berarti seperti namanya. Pemetaan lengkapnya ada di repo/sqlstore/riwayatklaim.sql,
// satu-satunya tempat ketiganya dapat dibandingkan berdampingan.
//
// Membawa nama itu ke sistem baru berarti mewariskan kekacauan yang justru menjadi alasan
// migrasi (`03-CURRENT-ARCHITECTURE.md` §4.2). Yang dipakai di sini adalah padanan Inggris
// dari `CONTEXT.md` sesuai `D-19` dan `D-80`.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	riwayatklaim/          aturan modul + seam          ← paket ini
//	riwayatklaim/usecase/  orkestrasi: buka layar, cari
//	riwayatklaim/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	riwayatklaim/http/     lapisan transport modul ini  — handler, dto, rute
//
// # Yang BUKAN urusan paket ini
//
// Apa yang terjadi setelah tombol "Lihat Detail Klaim" ditekan. Di sistem lama ia membuka
// layar rincian lewat `setDataViewKlaim_Act` — dan layar itu punya menunya sendiri,
// `MENU_ID 75` "View Claim" (`PNCViewClaim`), sehingga ia modul tersendiri. Modul ini
// mengetahui KEBERADAAN klaimnya (ia menyusun rujukannya, lihat ClaimHistory.Reference)
// tetapi tidak mengetahui isinya.
package riwayatklaim

import (
	"context"
	"strings"
	"time"
)

// ClaimHistory adalah satu baris hasil pencarian riwayat klaim.
//
// # Kenapa sebagian isian dapat kosong
//
// Karena kedua belas kueri sistem lama TIDAK mengembalikan kolom yang sama. Hanya
// pencarian Tanggal Lahir yang membawa nama peserta dan tanggal lahirnya; hanya pencarian
// ID Balai Lelang yang membawa nomor akseptasi dan nomor balai lelang. Kolom yang tidak
// dibawa kueri yang sedang dipakai bernilai kosong, dan layar menyembunyikannya — bukan
// menampilkan kolom kosong yang membuat pengguna menduga datanya hilang.
//
// Isian mana yang terisi untuk tipe pencarian mana dinyatakan SearchType.ExtraColumns,
// sehingga layar tidak perlu menebaknya.
type ClaimHistory struct {
	// Reference adalah kunci teknis Pega — `CLAIMID` pada T_CLAIM_PNC, berbentuk
	// "ASM-FW-GCNMFW-WORK <nomor>" pada klaim warisan.
	//
	// Ia dibawa karena tombol "Lihat Detail Klaim" membutuhkannya, BUKAN untuk
	// ditampilkan. `03-CURRENT-ARCHITECTURE.md` §4.1 menyebut bocornya nama kelas Pega
	// ke data bisnis sebagai utang teknis; ia tidak dibawa ke layar, dan `D-22` menetapkan
	// klaim terbitan sistem baru tidak pernah menulis awalan itu lagi.
	Reference string

	// Number adalah Nomor Klaim yang dibaca pengguna — `CLAIMNO`, yang di grid lama
	// bernama `.EDMNO`.
	//
	// Dua format hidup berdampingan selama masa paralel: `PNC-xxxx` dari Pega dan
	// `PNCN.YY.xxxx` dari sistem baru (`D-71`). Keduanya muncul di daftar yang sama.
	Number string

	// PolicyNumber — `NOPOLIS`.
	PolicyNumber string

	// InsuredName adalah Nama Tertanggung — `QQNAME`.
	InsuredName string

	// LossDate adalah Tanggal Kejadian, di grid lama bernama `.STARTDATE` padahal
	// kolomnya `DATEOFLOSS`.
	//
	// Nil bila kolomnya kosong di basis data. Pointer, bukan time.Time kosong: tanggal
	// nol tahun 1 tidak dapat dibedakan dari "belum diisi" saat ditampilkan.
	LossDate *time.Time

	// BusinessName — `BUSINESSNAME`, lini bisnis klaim.
	BusinessName string

	// BranchName — `BRANCHNAME`, cabang yang menangani.
	BranchName string

	// WorkStatus adalah Status alur kerja — `STATUSWORK`, di grid lama bernama
	// `.STATUSBUSINESS`.
	WorkStatus string

	// ClaimPosition adalah Posisi Klaim — hasil pencarian `v_sts_claim.lsc_note`
	// berdasarkan `StatusClaim`. Di grid lama bernama `.THEINSURED`.
	//
	// Ia KOSONG pada pencarian Nama Objek, dan itu perilaku sistem lama yang
	// direplikasi sadar: kueri tipe 3 adalah satu-satunya yang tidak membawa subquery
	// v_sts_claim. Keputusan Work Owner 2026-09-20 menetapkan ketiga cacat layar ini
	// direplikasi apa adanya, bukan diperbaiki.
	ClaimPosition string

	// CloseDate adalah Tanggal Close — `CLOSECLAIMDATE`, di grid lama `.ENDDATE`.
	CloseDate *time.Time

	// CloseNote adalah Catatan Close — `CLOSECLAIMNOTE`, di grid lama `.FLAGEDMBATAL`.
	// Namanya di Pega menyebut "flag batal"; isinya catatan bebas.
	CloseNote string

	// TechnicalPIC adalah PIC Teknis — `PICTEKNIK`, di grid lama `.SOBNAME`.
	TechnicalPIC string

	// AcceptanceNumber adalah No Akseptasi — hanya terisi pada pencarian ID Balai
	// Lelang, dari `detail_pnc_salvage.noakseptasi`. Di grid lama `.OLDPOLICYNO`.
	AcceptanceNumber string

	// AuctionHouseID adalah No Balai Lelang — hanya terisi pada pencarian ID Balai
	// Lelang, dari `detail_pnc_salvage.idbalailelang`. Di grid lama `.WARRANTYNO`.
	AuctionHouseID string

	// InsuredItemName adalah Nama Objek — hanya terisi pada pencarian Tanggal Lahir,
	// dari `t_person.FullName`. Di grid lama `.SOBLEADER1`.
	//
	// Istilahnya mengikuti `CONTEXT.md`: objek pertanggungan, bukan "Object" yang
	// bertabrakan dengan makna pemrograman.
	InsuredItemName string

	// BirthDate adalah Tanggal Lahir peserta — hanya terisi pada pencarian Tanggal
	// Lahir, dari `t_person.ASMDateOfBirth`. Di grid lama `.EDMDATE`.
	BirthDate *time.Time
}

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa modul ini memaginasi sementara sistem lama tidak
//
// Tak satu pun dari kedua belas kueri lama menyetel `MaxRecords`, sehingga seluruh baris
// yang cocok ditarik sekaligus ke halaman klipboard. Terhadap `T_CLAIM_PNC` yang berisi
// puluhan juta baris (`D-10`), pencarian "Nama Customer" berpola `%a%` akan menarik hampir
// seluruh tabel.
//
// Paginasi di sini karena itu **perubahan perilaku yang disadari**, bukan pemeliharaan —
// `09-DATABASE-STRATEGY.md` §6.3 menyatakannya harus diuji per layar, tidak diasumsikan
// setara.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK,
// bukan dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari
// parameter query yang mudah salah ketik, dan menolak seluruh permintaan karena
// `halaman=0` akan membuat layar gagal tanpa alasan yang terbaca pengguna. Ukuran yang
// melebihi batas dipotong ke batas, bukan dibiarkan.
func (p Pagination) Normalize() Pagination {
	clean := p
	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
//
// Keduanya dikembalikan bersama, bukan lewat dua panggilan terpisah, supaya angka
// "menampilkan 21–40 dari 57" tidak dapat berasal dari dua saat yang berbeda.
type Page struct {
	Claims []ClaimHistory
	Total  int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan, bukan yang
	// diminta. Layar menggambar penomoran halamannya dari sini.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
	size := p.Pagination.Normalize().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
	// di sistem lama. Ia yang dicocokkan ke kolom LOGIN pada master proteksi data,
	// bukan NIK.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// Repo adalah seam ke penyimpanan riwayat klaim SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut
// entitas, dan memang tidak boleh ada.
//
// # Kenapa hanya ada satu operasi, dan tidak ada satu pun yang menulis
//
// Karena layar ini tidak mengubah apa pun. Ia membaca riwayat; registrasi klaim
// (`B-2`), akseptasi (`B-10`), dan penutupan klaim terjadi di layar lain. Operasi yang
// tidak tersedia di seam ini tidak dapat dipakai kode yang ditulis kemudian tanpa
// keputusan sadar.
type Repo interface {
	// Search menjalankan kueri milik satu tipe pencarian dan mengembalikan satu halaman
	// hasil beserta jumlah seluruh baris yang cocok.
	Search(ctx context.Context, criteria Criteria, page Pagination) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti membaca riwayat klaim satu
// badan hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Ia ada supaya waktu pencatatan pemakaian gerbang dapat diuji secara deterministik.
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan
// tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
