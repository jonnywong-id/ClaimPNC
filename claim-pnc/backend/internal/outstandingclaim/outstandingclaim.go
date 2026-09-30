// Package outstandingclaim adalah inti modul Outstanding Claim.
//
// # Layar apa ini
//
// Rincian satu klaim treaty proporsional. Di Pega ia BUKAN butir menu melainkan **Flow
// Action** `OutstandingClaim` pada kelas `ASM-FW-GCNMFW-Work-ClaimTreaty`, yang dijalankan
// ketika pengguna mengklik nomor klaim di Inbox Claim Treaty Prop (`MENU_ID 54`). Layarnya
// adalah `Section/OutstandingClaim-Section.xml`.
//
// Karena pintunya satu — nomor klaim di inbox — modul ini tidak punya butir menu sendiri,
// dan memang tidak boleh punya.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Flow Action/OutstandingClaim-FA.xml       pintu masuknya; menunjuk section di bawah
//	Section/OutstandingClaim-Section.xml      97 isian + 10 grid, lihat section.go
//	Activity/ProteksiData_act-Act.xml         pra-aksi; MENAMAI ketiga page list utama
//	Activity/CheeckNoRNM_Act-Act.xml          pra-aksi; mengisi Treaty Group ID
//	RDB List/GetTreatyGroupID-SQL.xml         satu-satunya kueri yang dipanggil pra-aksi
//
// `SetDateOutstanding` — pra-aksi ketiga yang disebut Flow Action — **tidak ada di export**
// (`R-16`). Namanya menyiratkan ia menyetel tanggal saat layar dibuka; apa yang disetelnya
// tidak diketahui, dan modul ini tidak menebaknya.
//
// # Dari mana ISINYA dibaca, dan kenapa bukan dari kolom tabel
//
// Section mengikat hampir seluruh isiannya ke `.ClaimData.*` — satu halaman klipboard yang
// di basis data tersimpan sebagai **satu dokumen JSON**, `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`.
// Itu bukan dugaan: ketiga kueri Inbox Claim Treaty Prop sudah membaca `$.IDMaster`,
// `$.InsuredName`, `$.DateOfLoss`, dan `$.QuotationData.*` dari dokumen yang sama untuk
// klaim yang sama. Jalur `$.X` pada dokumen itu adalah `.ClaimData.X` di klipboard.
//
// Dokumennya dibaca UTUH lalu diurai di Go, bukan dipetik dengan `JSON_VALUE` per isian.
// Tiga alasan:
//
//	satu perjalanan          97 isian dan 10 grid berarti 10 `JSON_TABLE` bila dipetik SQL
//	kegagalan yang terbaca   jalur yang salah pada `JSON_TABLE` mengembalikan KOSONG diam-diam;
//	                         di sini isian yang tidak ditemukan dapat dibedakan dari yang kosong
//	portabilitas             sepuluh `JSON_TABLE` harus berperilaku sama di Oracle dan
//	                         PostgreSQL 17 (`D-20`, `D-24`); satu kolom CLOB tidak
//
// Yang TIDAK datang dari dokumen itu adalah keadaan objek kerjanya — nomor klaim, status,
// dan petugas pengubah terakhir — dan ketiganya dibaca sebagai kolom tabel dari
// `DATAPEGA.PC_ASM_FW_GCNMFW_WORK`.
//
// # Satu blok yang TIDAK dapat dibaca sama sekali
//
// Delapan isian pada blok Treaty Information terikat ke `.TreatyInMaster.*`, halaman berkelas
// `ASM-FW-GISFW-Int-TREATY_IN`. Ia halaman TERSENDIRI pada objek kerja, bukan bagian
// `.ClaimData`, sehingga tidak ikut tersimpan di dokumen JSON di atas.
//
// Tidak satu pun rule di export memuatnya: seluruh export hanya menyebut `TREATY_IN` di dua
// berkas, dan keduanya memakainya — tidak ada yang mengisinya. Kedua pra-aksi Flow Action pun
// bukan pemuat data; `ProteksiData_act` seluruhnya validasi, dan `CheeckNoRNM_Act` hanya
// mengambil Treaty Group ID.
//
// Kedelapan isian itu karena itu **digambar dengan alasan terhalang**, bukan dihilangkan dan
// bukan ditebak dari kolom yang kebetulan mirip — lihat section.go. Menebaknya berarti
// menampilkan pita share reasuransi milik treaty lain sebagai milik treaty ini, dan
// kesalahan seperti itu tidak menghasilkan satu pun galat.
//
// # Layar ini MEMBACA SAJA
//
// Flow Action aslinya menulis: ia menyimpan kembali objek kerja `ASM-FW-GCNMFW-Work-ClaimTreaty`
// beserta seluruh page list di dalamnya. Selama masa paralel, tabel objek kerja dan
// `POOLDATA.JSON_KLAIM` dimiliki Pega (`P-1`), sehingga tidak satu pun operasi di seam Repo
// menulis — dan memang tidak boleh ada.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor pustaka
// standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	outstandingclaim/          aturan modul + seam          ← paket ini
//	outstandingclaim/usecase/  orkestrasi: rakit satu rincian
//	outstandingclaim/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	outstandingclaim/http/     lapisan transport modul ini  — handler, dto, rute
package outstandingclaim

import (
	"context"
	"strings"
)

// Detail adalah satu klaim treaty beserta SELURUH isi layar rinciannya.
//
// Ia dibagi menjadi keadaan objek kerja, isian skalar, dan sepuluh grid — persis pembagian
// yang dipakai section. Bentuk gridnya senarai, bukan peta, karena urutan barisnya bermakna:
// urutan itulah yang dipakai Pega menghitung total di baris terakhirnya.
type Detail struct {
	// ── Keadaan objek kerja, dari DATAPEGA.PC_ASM_FW_GCNMFW_WORK ──

	// ClaimID adalah nomor klaim yang dibaca pengguna — `PYID`, mis. `CLMP-70`.
	//
	// Ia pula kunci yang dipakai membuka layar ini, dan itu disengaja: alamatnya terbaca
	// orang dan dapat disalin ke percakapan.
	ClaimID string

	// Reference adalah kunci teknis Pega — `PZINSKEY`.
	//
	// Tidak digambar. Ia dibawa supaya beralih memakainya sebagai kunci alamat kelak tidak
	// menuntut perubahan kontrak.
	Reference string

	// StatusWork adalah `PYSTATUSWORK` — status ALUR KERJA Pega ("New", "Pending", …).
	//
	// Ia BUKAN Status Klaim berkode `1134`–`1166` milik master `V_STS_CLAIM`; keduanya
	// konsep berbeda (`D-18`). Nilainya tidak diterjemahkan ke label master mana pun.
	StatusWork string

	// LastUpdateOperator adalah `PXUPDATEOPERATOR` — petugas yang terakhir mengubahnya.
	LastUpdateOperator string

	// ── Isian skalar, dari POOLDATA.JSON_KLAIM.DATA_JSONBLOB ──

	// Values memuat setiap isian skalar, dikunci nama isian pada kontrak API.
	//
	// # Kenapa PETA, bukan struct berisi 97 field
	//
	// Karena isian di layar ini tidak punya tipe yang berbeda-beda: seluruhnya digambar
	// sebagai teks baca-saja, dan 90 dari 97 di antaranya bertanda `Read-only` di section.
	// Struct berisi 97 field bertipe sama hanya memindahkan daftar yang sama ke tempat
	// kedua — daftar yang sudah ada di section.go dan wajib sama dengannya.
	//
	// Kuncinya DIJAGA: NewDetail menolak kunci yang tidak dikenal section.go, sehingga
	// peta ini tidak dapat menampung isian yang tidak pernah digambar.
	Values map[string]string

	// ── Grid ──

	// Grids memuat setiap grid, dikunci kode grid pada section.go.
	//
	// Grid yang sumbernya kosong di dokumen JSON tetap ADA di peta ini dengan nol baris.
	// Bedanya dengan grid yang tidak ada kuncinya sama sekali bermakna: yang pertama
	// berarti "tidak ada isinya", yang kedua berarti "belum dapat dibaca".
	Grids map[string][]GridRow
}

// GridRow adalah satu baris grid.
//
// Isinya peta dari kunci kolom ke teksnya, dengan alasan yang sama seperti Detail.Values:
// setiap sel digambar sebagai teks, dan bentuk barisnya berbeda-beda antar grid.
type GridRow map[string]string

// Get mengembalikan isian bernama tertentu, atau teks kosong bila tidak ada.
//
// Ia ada supaya pemanggil tidak perlu memeriksa keberadaan kunci di setiap tempat — dan
// supaya tidak ada yang tergoda mengakses peta langsung lalu memakai nilai nol tanpa sadar.
func (d Detail) Get(field string) string {
	if d.Values == nil {
		return ""
	}
	return d.Values[field]
}

// Rows mengembalikan baris sebuah grid, atau senarai kosong bila gridnya tidak terisi.
func (d Detail) Rows(grid string) []GridRow {
	if d.Grids == nil {
		return nil
	}
	return d.Grids[grid]
}

// Query adalah permintaan satu rincian yang sudah tervalidasi.
type Query struct {
	// ClaimID adalah nomor klaim, mis. `CLMP-70`.
	ClaimID string

	// Caller adalah identitas pemanggil.
	//
	// Ia TIDAK menyaring apa pun — lihat NewQuery — tetapi dibawa supaya setiap pembukaan
	// rincian dapat dicatat dengan pelakunya.
	Caller Caller
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk — `OperatorID.pyUserIdentifier`.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
//
// # Kenapa rincian TIDAK disaring menurut pemanggil
//
// Karena di Pega pun tidak. Flow Action `OutstandingClaim` dijalankan atas assignment yang
// sedang dibuka, dan siapa pun yang dapat membuka assignment itu melihat seluruh isinya —
// tidak ada satu pun prakondisi berbasis operator di kedua pra-aksinya.
//
// Konsekuensinya diterima dengan sadar dan tidak disembunyikan: nomor klaim di sini
// BERURUTAN (`CLMP-70`, `CLMP-71`, …), sehingga siapa pun yang sudah masuk dapat membaca
// rincian klaim treaty mana pun di portalnya hanya dengan menaikkan angkanya. Yang membatasi
// bukan modul ini melainkan pemeriksaan kewenangan menu — `TKT-F3-005` — yang belum ada.
//
// Itu sebabnya setiap pembukaan DICATAT beserta pelakunya di lapisan usecase. Pencatatan
// bukan kendali, dan tidak diklaim sebagai kendali; ia yang membuat penyalahgunaannya dapat
// ditelusuri setelah terjadi.
func NewQuery(claimID string, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	clean := strings.TrimSpace(claimID)
	if clean == "" {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldClaimID,
			Message: "Nomor klaim wajib diisi.",
		}})
	}

	return Query{ClaimID: clean, Caller: cleanCaller}, nil
}

// Repo adalah seam ke rincian klaim treaty pada SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat kueri
// (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas, dan
// memang tidak boleh ada.
//
// # Tidak ada satu pun operasi yang menulis
//
// Flow Action aslinya menyimpan kembali objek kerja beserta seluruh page list di dalamnya.
// Selama masa paralel, tabel itu dimiliki Pega (`P-1`). Operasi yang tidak tersedia di seam
// ini tidak dapat dipakai kode yang ditulis kemudian tanpa keputusan sadar.
type Repo interface {
	// Find mengembalikan satu rincian klaim.
	//
	// Klaim yang tidak ada menghasilkan ErrNotFound, bukan Detail kosong: Detail kosong
	// terbaca di layar sebagai "klaim tanpa isi", padahal yang benar adalah "klaim tidak
	// ditemukan".
	Find(ctx context.Context, q Query) (Detail, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan nama tertanggung,
// nilai klaim, dan pembagian reasuransi satu badan hukum kepada petugas badan hukum lain
// tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
