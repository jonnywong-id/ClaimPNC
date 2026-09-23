package inboxlaporanklaim

import (
	"context"
	"strings"
	"time"
)

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6), dan seluruh layar ini
// bergantung padanya — batas datanya cabang, dan ketiga tab komunikasi menyaring
// percakapan menurut pengirimnya.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`
	// di sistem lama. Ia yang dicocokkan ke `pxcreateoperator` dan ke `sender` pada
	// percakapan, bukan NIK.
	Login string

	// Name dipakai saat membuat berkas baru: `CreateNewCaseRCV` mengisi Sender dengan
	// `OperatorID.pyUserName`.
	//
	// CELAH YANG DISADARI — nomor telepon tidak ikut. Activity yang sama juga mengisi
	// TelpPengirim dengan `OperatorID.pyTelephone`, dan kontrak HCC/HCQ **tidak
	// memuat nomor telepon sama sekali**: `auth.Profile` berisi NIK, nama, login, surel,
	// perusahaan, cabang, kode cabang, dan jabatan — tidak lebih.
	//
	// Kolomnya karena itu TIDAK dibuat di basis data. Membuat kolom yang selamanya
	// kosong menjadikannya tampak "belum diisi" padahal sumbernya memang tidak ada, dan
	// itu pertanyaan yang akan terus berulang. Bila kelak nomor telepon dibutuhkan, yang
	// harus bertambah lebih dulu adalah kontrak identitasnya.
	Name string

	// # Kenapa TIDAK ada BranchCode di sini
	//
	// Versi pertama modul ini membawa `auth.User.BranchCode` — kode cabang dari profil
	// HCC/HCQ (`EmpResponse.Placement.BranchCode`) — lalu memakainya sebagai penyaring
	// terhadap kolom `kodecabang_1`. Keduanya **sistem kode yang berbeda**, dan akibatnya
	// tidak satu pun baris cocok: daftar tampil KOSONG tanpa satu pun pesan galat.
	//
	// Kode cabang yang dipakai layar ini adalah `POOLDATA.BRANCH.ID`, dan sistem lama
	// menurunkannya lewat tiga tabel dan dua DB Link
	// (`RDB List/GetIDCabang-SQL.xml`):
	//
	//	login petugas -> HRDASM.V_HRD_MST.login_aplikasi
	//	              -> NIK
	//	              -> LST_USER_ASURANSI.cab_id
	//	              -> BRANCH.oldid
	//	              -> BRANCH.id            <- inilah yang dibandingkan
	//
	// Perhatikan sambungan terakhir: ia lewat `oldid`, BUKAN `id`. Tidak ada satu pun
	// jalan pintas dari profil HCC/HCQ ke nilai itu.
	//
	// Karena itu cabang TIDAK lagi dibawa di sini; ia diselesaikan BranchResolver dari
	// Login. Membawanya sebagai field yang tampak sudah benar adalah persis yang membuat
	// cacat ini luput — nilainya ada, bentuknya masuk akal, dan hasilnya salah diam-diam.
}

// BranchResolver menerjemahkan login petugas menjadi kode cabang klaimnya.
//
// # Kenapa ia seam tersendiri
//
// Karena penurunannya menyentuh dua basis data lain lewat DB Link, dan `D-25` menetapkan
// seluruh DB Link kelak diganti pemanggilan API (`R-03`). Menaruhnya di balik seam berarti
// penggantian itu kelak tidak menyentuh satu baris pun aturan modul ini.
//
// # Kegagalan BUKAN galat
//
// Nilai kedua false berarti cabangnya tidak dapat ditentukan — petugas non-karyawan tidak
// ada di HRD, dan DB Link dapat sedang tidak dapat dihubungi. Pemanggil WAJIB
// memperlakukannya sebagai "tanpa batas cabang", bukan sebagai "tidak ada berkas".
//
// Sistem lama memang berperilaku sebaliknya: ia merangkai `branch where ID=''` lalu
// menampilkan daftar kosong. Perilaku itu TIDAK direplikasi, dan itu keputusan sadar —
// `ID=''` bukan aturan bisnis melainkan akibat perangkaian teks `{ASIS:}` yang tidak
// pernah memeriksa hasilnya. Daftar kosong yang tidak menjelaskan dirinya adalah kegagalan
// yang paling mahal ditemukan.
type BranchResolver interface {
	Resolve(ctx context.Context, login string) (code string, resolved bool, err error)
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{
		Login:      strings.TrimSpace(c.Login),
		Name:       strings.TrimSpace(c.Name),
	}
}

// Region adalah satu pilihan pada dropdown "Pilih Kanwil".
//
// Sumbernya kolom `basterritory` pada POOLDATA.BRANCH, yang dipakai `tempQuery.Remark`
// untuk menyaring cabang mana saja yang termasuk sebuah kanwil.
type Region struct {
	Code string
	Name string
}

// Repo adalah seam ke penyimpanan laporan klaim SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di
// tingkat kueri (`ADR-0030` Opsi 1). Tidak ada satu pun kueri di baliknya yang menyaring
// menurut entitas, dan memang tidak boleh ada.
//
// # Kenapa ada Update, tetapi tidak ada Delete
//
// Update ada karena form **Input Receive Document** mengisi berkas yang sudah dibuat
// tombol "Buat Baru" — itulah assignment tunggal pada `Flow/InputReceiveDocument.xml`,
// dan tanpa Update tombol itu hanya menerbitkan berkas kosong yang tidak dapat diapa-apakan.
//
// Delete TIDAK ada, dan tidak akan ada: `ADR-0012` melarang penghapusan fisik data
// bernilai bisnis. Operasi yang tidak tersedia di seam ini tidak dapat dipakai kode yang
// ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan satu halaman hasil beserta jumlah seluruh baris yang cocok.
	//
	// Keduanya dikembalikan bersama, bukan lewat dua panggilan terpisah, supaya angka
	// "menampilkan 11–20 dari 57" tidak dapat berasal dari dua saat yang berbeda.
	List(ctx context.Context, filter Filter, page Pagination) (Page, error)

	// Summarize mengembalikan kedelapan pencacah di atas daftar.
	//
	// Penyaring yang sama dengan daftar ikut diberlakukan — kecuali Category, yang
	// memang tidak berlaku: pencacahnya justru menghitung setiap kategori sekaligus.
	Summarize(ctx context.Context, filter Filter) (Summary, error)

	// ListRegions mengembalikan isi dropdown "Pilih Kanwil".
	ListRegions(ctx context.Context) ([]Region, error)

	// Get mengembalikan satu berkas; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (ClaimReport, error)

	// Insert menyimpan berkas baru dan mengembalikan baris yang benar-benar tersimpan.
	//
	// Nomornya diterbitkan di dalam operasi ini, bukan diminta lebih dulu lalu dipakai
	// beberapa langkah kemudian: jarak antara mengambil nomor dan memakainya adalah
	// jarak yang membuat dua penambahan bersamaan menerima nomor yang sama.
	Insert(ctx context.Context, report ClaimReport) (ClaimReport, error)

	// Update menyimpan isian form ke atas berkas yang sudah ada.
	//
	// ErrNotFound bila berkasnya hilang di antara pemuatan form dan penyimpanannya.
	//
	// Pengisi seam WAJIB menolak baris milik Pega dengan ErrReadOnlyOrigin. Ia bukan
	// kenyamanan tampilan: selama masa paralel, tepat satu sistem yang menulis sebuah
	// baris (`ADR-0004`, `P-1`), dan `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` dibaca 116 rule
	// Pega yang masih melayani produksi.
	//
	// Report sudah harus melewati Detail.Clean dan Detail.Check.
	Update(ctx context.Context, report ClaimReport) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti membaca berkas satu badan
// hukum dari basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Ia ada supaya umur berkas dan tanggal berkas baru dapat diuji secara deterministik.
// Seluruh waktu yang dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja
// dan tidak pernah dengan menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}
