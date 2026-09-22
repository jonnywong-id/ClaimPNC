// Package inboxclaimtreatyprop adalah inti modul Inbox Claim Treaty Prop.
//
// # Layar apa ini
//
// Menu `MENU_ID 54` "Inbox Claim Treaty Prop" pada POOLDATA.M_MENU_APLIKASI_PNC, yang
// menunjuk harness `InboxClaimTreaty_Harness`. Ia berada di kelompok menu `2` (Proses
// Produksi), urutan 1144, dan diotorisasi untuk grup `IT` pada POOLDATA.M_OTORISASI_PNC.
//
// Isinya **daftar pekerjaan klaim treaty proporsional** — klaim yang masuk lewat jalur
// treaty inward, tempat ASM menjadi penanggung ulang atas klaim milik perusahaan asuransi
// lain (Ceding Co). Per `D-79` ia benar-benar Inbox, bukan layar data acuan: barisnya
// diambil dari DATAPEGA.PC_ASSIGN_WORKLIST dan PC_ASSIGN_WORKBASKET, dan hilang begitu
// penugasannya selesai. Karena itu modul ini milik `U-3`, bukan `U-6`.
//
// Menu `MENU_ID 55` "Inbox Claim Treaty Non Prop" (`InboxClaimNonProp_Harness`) adalah
// layar SAUDARA yang berdiri sendiri dan TIDAK dibangun di sini.
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxClaimTreaty_Harness-Harness.xml   pembungkus layar (Data-Portal)
//	Section/InboxClaimTreaty_Section-Section.xml   3 kontainer, 5 grid, 1 tombol, 2 checkbox
//	Activity/GetDataTreatyin_Act-Act.xml           pemilih kueri per peran + "See All Claim"
//	Activity/GetDataTreatyin_Actkomite-Act.xml     varian komite dari activity yang sama
//	Activity/GetDataInboxTreaty_act-Act.xml        penentu mode komite (lihat di bawah)
//	RDB List/GetClaimTreaty_SQL-SQL.xml            worklist milik pemanggil
//	RDB List/GetClaimTreatyAllAdmin_SQL-SQL.xml    worklist seluruh petugas
//	RDB List/GetClaimTreatyTeknik_SQL-SQL.xml      workbasket `TreatyinPNCTeknik`
//	Report Definition/WorkListKomite2-RD.xml       antrean komite (terhalang, lihat tab.go)
//	Report Definition/InboxKomiteTreaty_RD-RD.xml  antrean komite (terhalang, lihat tab.go)
//
// # Layar lama punya DUA MODE, dan pemisahnya bukan tombol
//
// `Section/InboxClaimTreaty_Section-Section.xml` menjaga kontainernya dengan
// `InputData.CARI13`:
//
//	CARI13 != 'tampil'   kontainer "Claim Treatyin In Progress"
//	CARI13 == 'tampil'   kontainer "Histori Klaim Treaty" + "Komite Treaty ASM"
//
// `CARI13` TIDAK disetel oleh salah satu activity pemuat grid. Ia disetel
// `Activity/GetDataInboxTreaty_act-Act.xml` langkah 3, sesudah langkah 2 menjalankan
// report atas kelas `ASM-FW-GCNMFW-Int-EMAILKOMITE`, dengan prakondisi
//
//	@equalsIgnoreCase(.OPERATOR_ID, Inputdata.CARI10)
//
// Artinya `CARI13 = 'tampil'` berarti **pemanggil adalah anggota komite** menurut
// POOLDATA.EMAILKOMITE.OPERATOR_ID — bukan sakelar tampilan, melainkan pemeriksaan peran.
// Pembacaan itu sejalan dengan yang sudah ditetapkan modul Master Surveyors: kolom
// `OPERATOR_ID` pada EMAILKOMITE memang identitas anggota komite
// (`internal/mastersurveyors/committee/resolver.go`).
//
// # Kenapa modul ini punya TIGA tab, bukan dua mode
//
// Karena kedua mode itu bersama-sama hanya memuat TIGA sumber data yang berbeda, dan satu
// di antaranya digambar dua kali:
//
//	kontainer "Work List Treatyin Propotional"  GetClaimTreaty_SQL / ...AllAdmin_SQL
//	kontainer "Treaty Klaim"        (mode 1)    GetClaimTreatyTeknik_SQL
//	kontainer "Work Teknik Treatyin" (mode 2)   GetClaimTreatyTeknik_SQL   <- kueri yang SAMA
//	kontainer "Komite Treaty ASM"   (mode 2)    dua Report Definition
//
// "Treaty Klaim" dan "Work Teknik Treatyin" adalah grid yang sama dengan judul berbeda,
// dipasok kueri yang sama persis, dibedakan hanya oleh mode. Membawa keduanya berarti dua
// tab yang isinya dijamin identik. Yang dibawa karena itu satu tab, dengan judul
// "Work Teknik Treatyin".
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxclaimtreatyprop/          aturan modul + seam          ← paket ini
//	inboxclaimtreatyprop/usecase/  orkestrasi: rakit tab, isi satu tab
//	inboxclaimtreatyprop/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxclaimtreatyprop/http/     lapisan transport modul ini  — handler, dto, rute
package inboxclaimtreatyprop

import (
	"context"
	"strings"
)

// WorkItem adalah satu baris pekerjaan klaim treaty proporsional.
//
// # Kenapa satu bentuk untuk ketiga tab
//
// Karena begitulah sistem lama menyusunnya: ketiga kueri mengisi halaman klipboard berkelas
// `ASM-FW-GISFW-Data-Search` yang sama, dengan alias `CARI1`…`CARI15`. Yang berbeda antar
// tab hanyalah kolom mana yang terlihat — dan itu ditetapkan Tab.Columns, bukan oleh bentuk
// barisnya.
//
// # Kenapa nama isian di sini tidak mirip nama properti Pega
//
// Karena nama properti di layar ini tidak menyatakan apa pun: seluruh kolomnya bernama
// `CARI` ditambah nomor urut. Nomor itu pun TIDAK konsisten antar kueri — lihat LossDate.
// Yang dipakai di sini adalah padanan Inggris dari `CONTEXT.md` sesuai `D-19` dan `D-80`;
// pemetaan lengkapnya ada di repo/sqlstore/inboxclaimtreatyprop.sql.
type WorkItem struct {
	// Reference adalah kunci teknis Pega — `PZINSKEY` (alias `CARI4`).
	//
	// Ia dikirim ke layar tetapi TIDAK pernah digambar sebagai kolom; yang memakainya
	// adalah tombol rincian klaim.
	Reference string

	// WorkKey adalah kunci objek kerja yang ditunjuk penugasan — `PXREFOBJECTKEY`
	// (alias `CARI1`). Ia yang dipakai bergabung ke POOLDATA.JSON_KLAIM.IDPEGA.
	//
	// Tidak digambar. Ia dibawa karena penyaring `LIKE '%CLMP%'` pada ketiga kueri
	// bekerja atas kolom ini, dan penyimpanan memori harus dapat meniru penyaring itu.
	WorkKey string

	// ClaimID adalah nomor yang dibaca pengguna di kolom pertama — `PXREFOBJECTINSNAME`
	// (alias `CARI2`), berjudul "Claim ID".
	ClaimID string

	// AssignedOperator adalah petugas yang memegang penugasan ini —
	// `PXASSIGNEDOPERATORID` (alias `CARI3`).
	//
	// Tidak digambar di grid mana pun. Ia dibawa karena tab "Work List Treatyin
	// Propotional" menyaring menurut kolom ini, dan karena tanpa membawanya penyimpanan
	// memori tidak dapat meniru penyaring yang sama.
	AssignedOperator string

	// MasterID — `JSON_VALUE(DATA_JSONBLOB, '$.IDMaster')` (alias `CARI15`), berjudul
	// "ID Master".
	MasterID string

	// PolicyNumber — `JSON_KLAIM.NOPOLIS` (alias `CARI5`), berjudul "Policy No".
	PolicyNumber string

	// LossDate adalah Tanggal Kejadian — `JSON_VALUE(DATA_JSONBLOB, '$.DateOfLoss')`.
	//
	// # Kenapa TEKS, bukan time.Time
	//
	// Karena `JSON_VALUE` mengembalikan teks, dan bentuk teks di dalam blob itu tidak
	// dapat diperiksa: tidak ada DDL, dan isi `JSON_KLAIM` belum pernah dilihat (`R-08`).
	// Mengubahnya menjadi tanggal di sini berarti menebak formatnya untuk seluruh baris
	// historis. Pemformatannya dikerjakan layar, dan hanya bila bentuknya memang dikenali.
	//
	// # Alias kolomnya BERBEDA antar kueri, dan di sanalah cacatnya
	//
	// `GetClaimTreaty_SQL` dan `GetClaimTreatyAllAdmin_SQL` mengaliaskannya `CARI10`,
	// sedangkan `GetClaimTreatyTeknik_SQL` mengaliaskannya **`CARI13`**. Ketiga grid di
	// section terikat ke `.CARI10`. Akibatnya kolom "Date Of Loss" pada grid yang dipasok
	// kueri Teknik **selalu kosong** di sistem lama.
	//
	// Itu DIPERBAIKI di sini, dan perbaikannya disetujui Work Owner 2026-09-21 sebagai
	// selisih terencana `P-5` — lihat PlannedDifferences.
	LossDate string

	// BusinessName — `JSON_VALUE(DATA_JSONBLOB, '$.QuotationData.BusinessName')`
	// (alias `CARI6`).
	//
	// Judulnya BERBEDA antar grid pada layar lama: "Business Name" pada kedua grid
	// worklist, "Class Of Business" pada grid komite. Keduanya isi yang sama; perbedaan
	// judul dipertahankan lewat Tab.Columns (`D-13`).
	BusinessName string

	// BusinessSource — `JSON_VALUE(DATA_JSONBLOB, '$.QuotationData.SobName')`
	// (alias `CARI7`), berjudul "Source Of Business".
	BusinessSource string

	// CedingCompany adalah perusahaan asuransi yang mengalihkan risikonya kepada ASM —
	// `JSON_VALUE(DATA_JSONBLOB, '$.QuotationData.CedingCoName')` (alias `CARI8`),
	// berjudul "Ceding Co Name".
	CedingCompany string

	// InsuredName — `JSON_VALUE(DATA_JSONBLOB, '$.InsuredName')` (alias `CARI9`),
	// berjudul "Insured Name".
	InsuredName string

	// Subjectivity — `JSON_VALUE(DATA_JSONBLOB, '$.IsSubjectivity')` (alias `CARI14`).
	//
	// HANYA kueri Teknik yang membawanya; kedua kueri worklist tidak memuatnya sama
	// sekali. Karena itu kolomnya hanya digambar pada tab Work Teknik Treatyin, dan pada
	// tab lain isian ini memang kosong — bukan hilang.
	Subjectivity string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Itulah yang dibandingkan dengan `PXASSIGNEDOPERATORID` pada
	// `GetClaimTreaty_SQL`. Memakai NIK di sini akan membuat tab pertama tampak kosong
	// bagi setiap pengguna.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// TechnicalWorkbasket adalah operator workbasket yang memegang antrean teknik treaty.
//
// Nilainya literal di `RDB List/GetClaimTreatyTeknik_SQL-SQL.xml`:
// `where PXASSIGNEDOPERATORID='TreatyinPNCTeknik'`. Ia BUKAN nama orang melainkan akun
// fungsional, sehingga menuliskannya di sini tidak melanggar `D-67` — yang dilarang `D-15`
// adalah nilai bisnis yang berubah, sedangkan ini kunci antrean yang menentukan tab mana
// yang dibaca.
//
// Ia tetap dikumpulkan sebagai konstanta, bukan disebar ke dalam SQL dan penyimpanan
// memori masing-masing, supaya keduanya tidak dapat berselisih tanpa ketahuan.
const TechnicalWorkbasket = "TreatyinPNCTeknik"

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Karena itu ukuran halaman grid pada `Section/InboxClaimTreaty_Section-Section.xml`, sama
// dengan Inbox Admin. Ia tidak dikarang dan tidak disamakan dengan modul yang memakai 20.
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
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari parameter
// query yang mudah salah ketik, dan menolak seluruh permintaan karena `halaman=0` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna.
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
type Page struct {
	Items []WorkItem

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

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

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat catatan paginasi di
// repo/sqlstore/inboxclaimtreatyprop.sql.
//
// Keduanya tetap menghasilkan Page dengan arti yang sama, sehingga uji aturan modul yang
// berjalan di atas memori menyatakan hal yang benar tentang yang berjalan di Oracle.
func Slice(all []WorkItem, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []WorkItem{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Items = all[offset:end]
	return result
}

// Repo adalah seam ke antrean klaim treaty proporsional SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Tidak ada satu pun operasi yang menulis
//
// Itu bukan kelalaian melainkan batas yang ditetapkan Work Owner 2026-09-21. Tombol
// "Create Claim Treaty Prop" di layar lama memanggil `CreateInputKlaimTreaty`, yang membuat
// objek kerja baru di tabel milik Pega. Selama masa paralel tabel itu tetap dimiliki Pega
// (`P-1`), sehingga tombolnya digambar tetapi menolak dengan alasan — lihat http/routes.go.
//
// Operasi yang tidak tersedia di seam ini tidak dapat dipakai kode yang ditulis kemudian
// tanpa keputusan sadar.
type Repo interface {
	// List mengembalikan SATU HALAMAN baris yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan nama tertanggung
// dan Ceding Co satu badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat
// (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
