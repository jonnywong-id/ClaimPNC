package inboxosclaimpercabang

import (
	"context"
	"time"

	"claim-pnc/internal/platform/clock"
)

// Repo adalah seam ke daftar klaim outstanding SATU portal entitas.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
//
// # Tidak ada satu pun operasi yang menulis
//
// Itu bukan kelalaian melainkan batas. Layar lamanya tidak punya aksi tulis sama sekali, dan
// seluruh tabel yang dibacanya milik Pega selama masa paralel (`P-1`). Operasi yang tidak
// tersedia di seam ini tidak dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// BranchOf menerjemahkan kode cabang RINCI milik pemanggil menjadi cabang yang
	// barisnya ia lihat.
	//
	// Ia menggantikan `RDB List/GetNamaCabangTelepon-SQL.xml`, yang membaca tabel yang
	// sama tetapi dengan kolom kunci yang berbeda:
	//
	//	kueri lama   select branchname from pooldata.branch where id    = {OperatorID.pyTelephone}
	//	di sini      select id, branchname from pooldata.branch where oldid = :1
	//
	// # Kenapa ada terjemahan sama sekali
	//
	// Karena keduanya ruang kode yang BERBEDA, dan menyamakannya membuat layar kosong
	// tanpa satu pun galat. Terukur langsung terhadap basis data:
	//
	//	yang datang dari HCQ   Placement.DetailBranchCode   "001"     3 digit
	//	yang dipakai klaim     T_CLAIM_PNC.BRANCHCODE       "100081"  6 digit
	//
	// Work Owner menetapkan (2026-09-28) `DetailBranchCode` sama dengan
	// `POOLDATA.BRANCH.OLDID`, sehingga kode klaimnya diperoleh dengan
	// `SELECT id FROM pooldata.branch WHERE oldid = <DetailBranchCode>`.
	//
	// Diperiksa terhadap `GENERAL.LST_DET_CABANG@asmd`, jalur terjemahan yang lain:
	// **791 sepakat, 0 berselisih**. Jalur lokal ini dipilih karena hasilnya sama dan ia
	// tidak menyentuh DB Link sama sekali (`D-25`).
	//
	// # Kenapa kode dan nama dikembalikan bersama
	//
	// Karena keduanya datang dari BARIS YANG SAMA. Mengambilnya lewat dua pemanggilan
	// membuka kemungkinan judul layar menyebut cabang yang berbeda dari cabang barisnya —
	// kelas cacat yang tidak menghasilkan satu pun galat.
	//
	// Nilai kedua false berarti kode itu tidak dikenal master cabang. Itu BUKAN galat: ia
	// keadaan yang wajar bagi pengguna non-karyawan, yang di `POOLDATA.M_LOGIN_PNC` memang
	// tidak punya kolom cabang sama sekali.
	BranchOf(ctx context.Context, detailBranchCode string) (Branch, bool, error)

	// ListForExport mengembalikan satu halaman baris BERKAS, yang lebih lebar daripada baris
	// grid.
	//
	// Ia terpisah dari List dengan dua alasan yang keduanya nyata: nilai uangnya dihitung
	// berbeda (lihat WorkItem.EstimationValue), dan 24 kolom treaty-nya menuntut gabungan
	// lewat DB Link yang tidak boleh dibayar setiap kali layar dibuka.
	//
	// Ia tetap berhalaman. Ekspor menulis tiap potong langsung ke jawaban, sehingga memori
	// tetap datar berapa pun jumlah barisnya.
	ListForExport(ctx context.Context, query Query, page Pagination) (ExportPage, error)

	// FindDetail mengambil seluruh isi popup untuk SATU klaim milik cabang pada query.
	//
	// # Cabang ikut menjadi penyaring, dan itu bukan kehati-hatian berlebihan
	//
	// Layar lamanya tidak menyaring karena popup hanya dapat dibuka dari baris yang sudah
	// tampil. Endpoint HTTP tidak punya pembatas itu: nomor klaim dapat ditebak atau
	// diperoleh dari mana saja. Tanpa penyaring ini, popup menjadi jalan memutar yang
	// membocorkan nama tertanggung dan nilai uang antarbadan hukum (`R-20`).
	//
	// Nilai kedua false berarti klaim itu tidak ada, atau ada tetapi bukan milik cabang
	// tersebut. Keduanya sengaja tidak dibedakan: membedakannya memberi tahu pemanggil
	// bahwa sebuah nomor klaim memang ada di cabang lain.
	FindDetail(ctx context.Context, query Query, claimNumber string) (Detail, bool, error)

	// DominantFactors mengembalikan faktor dominan setiap klaim outstanding satu cabang,
	// dikunci nomor internal klaim (`T_CLAIM_PNC.CLAIMID`).
	//
	// Ia terpisah dari ListForExport karena tidak ada bentuk agregasi teks yang berjalan di
	// Oracle 19c maupun PostgreSQL 17+ — `LISTAGG` hanya ada di yang pertama, `STRING_AGG`
	// hanya di yang kedua. Perangkaiannya karena itu dikerjakan pemanggil.
	//
	// Nilainya sudah berurut sesuai `idx_dominanfactor`, sehingga pemanggil cukup
	// merangkainya tanpa mengurutkan ulang dengan aturan yang dapat berselisih.
	DominantFactors(ctx context.Context, query Query) (map[string][]string, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan nomor polis dan
// nama tertanggung satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan
// galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)

// Clock adalah seam ke waktu.
//
// Ia ada supaya kolom Aging dapat diuji secara deterministik. Seluruh waktu yang
// dihasilkannya UTC; pengubahan ke WIB terjadi di satu tempat saja dan tidak pernah dengan
// menambahkan tujuh jam secara manual (`F-5`).
type Clock interface {
	Now() time.Time
}

// AgingDaysSince menghitung umur klaim dalam hari kalender WIB.
//
// # Kenapa hitungannya di sini, bukan di dalam kueri
//
// Kueri lama memakai `TRUNC(SYSDATE) - TRUNC(c.registerdate)`. `TRUNC` tidak portabel, dan
// padanan yang dianjurkan `09-DATABASE-STRATEGY.md` §4 — `CAST(x AS DATE)` — **tidak
// memangkas jam di Oracle**. Diukur langsung terhadap basis data:
//
//	CAST(CURRENT_TIMESTAMP AS DATE) - CAST(registerdate AS DATE)  ->  0.8758…
//	TRUNC(SYSDATE)                  - TRUNC(registerdate)         ->  1
//
// Baris itu klaim yang terdaftar KEMARIN. Dipindai ke bilangan bulat, bentuk pertama
// menjadikannya **0 hari**. Modul lain di repo ini memakai bentuk pertama; itu dilaporkan
// terpisah dan tidak disentuh dari sini.
//
// Menghitungnya di Go menutup ketiganya sekaligus: portabel tanpa perkecualian, benar di
// sekitar tengah malam karena batas harinya WIB dan bukan jam server basis data, dan bebas
// dari pembulatan yang tidak terlihat.
//
// Umur negatif dikembalikan sebagai 0. Tanggal registrasi di masa depan memang ada pada data
// warisan, dan umur negatif bukan jawaban yang berarti bagi siapa pun yang membacanya.
func AgingDaysSince(registeredAt *time.Time, now time.Time) int {
	if registeredAt == nil || registeredAt.IsZero() {
		return 0
	}

	from := clock.DateWIB(*registeredAt)
	to := clock.DateWIB(now)
	if !to.After(from) {
		return 0
	}

	// Pembagian jam, bukan `Sub().Hours()/24` yang dibulatkan: keduanya sudah tengah malam
	// WIB, sehingga selisihnya selalu kelipatan 24 jam persis.
	return int(to.Sub(from).Hours() / 24)
}
