// Package inboxclaimtreatynonprop adalah inti modul Inbox Claim Treaty Non Prop.
//
// # Layar apa ini
//
// Menu `MENU_ID 55` "Inbox Claim Treaty Non Prop" pada POOLDATA.M_MENU_APLIKASI_PNC, yang
// menunjuk harness `InboxClaimNonProp_Harness`. Ia berada di kelompok menu `2` (Proses
// Produksi), urutan 1145 — tepat sesudah `MENU_ID 54` "Inbox Claim Treaty Prop".
//
// Isinya **daftar pekerjaan klaim treaty NON-proporsional**: klaim yang masuk lewat jalur
// treaty inward non-proporsional, tempat ASM menanggung kerugian di atas batas tertentu
// (Excess of Loss) atas klaim milik perusahaan asuransi lain (Ceding Co). Per `D-79` ia
// benar-benar Inbox: barisnya diambil dari DATAPEGA.PC_ASSIGN_WORKLIST dan
// PC_ASSIGN_WORKBASKET, dan hilang begitu penugasannya selesai. Karena itu modul ini milik
// `U-3`, bukan `U-6`.
//
// # Ia BUKAN salinan modul Prop
//
// Modul `inboxclaimtreatyprop` adalah layar saudaranya, dan keduanya memang mirip di layar.
// Di bawah permukaan keduanya berbeda pada hal-hal yang menentukan kuerinya:
//
//	                    Prop                        Non Prop (paket ini)
//	penanda objek kerja PXREFOBJECTKEY LIKE %CLMP%  PXREFOBJECTINSNAME LIKE 'CLMNP-%'
//	jumlah tabel        2                           3
//	asal kolom bisnis   JSON_VALUE(DATA_JSONBLOB)   kolom PC_ASM_FW_GCNMFW_WORK
//	varian tab pertama  2 (biasa, See All)          3 (biasa, See All, TBA)
//	kolom tambahan      —                           Status, Aging, Create/Last Update Operator
//
// Menyatukan keduanya karena judulnya mirip akan memaksa satu kueri melayani dua bentuk
// data yang tidak sama, dan itulah cacat yang justru sedang ditinggalkan (utang teknis
// 4.6 — duplikasi per lini bisnis yang disatukan asal-asalan sama buruknya dengan yang
// digandakan asal-asalan).
//
// # Asal setiap aturan di modul ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/InboxClaimNonProp_Harness-Harness.xml  pembungkus layar (Data-Portal)
//	Section/InboxClaimNonProp_Harness-Section.xml  3 kontainer grid, checkbox, tombol
//	Activity/GetWorkCNP_Act-Act.xml                pemuat layar; mengisi ketiga grid
//	Activity/GetDataTreatyinNonProp_Act-Act.xml    pemilih kueri tab Admin (3 varian)
//	Activity/GenerateClaimNonPropCSV-Act.xml       tombol ekspor
//	Activity/CreateClaimTNonProp_Act-Act.xml       tombol buat klaim (tidak dibawa, lihat http)
//	RDB List/GetKlaimNonPropAdmin_SQL-SQL.xml      worklist milik pemanggil
//	RDB List/GetKlaimNonPropAdminALL_SQL-SQL.xml   worklist seluruh petugas
//	RDB List/GetKlaimNonPropAdminTBA_SQL-SQL.xml   worklist yang polisnya belum terbit
//	RDB List/GetInboxListCNP_SQL-SQL.xml           workbasket `TreatyinPNCTeknik`
//
// # Bagaimana layar lama memuat ketiga gridnya
//
// `GetWorkCNP_Act` dijalankan saat layar dibuka, dan urutannya menjelaskan bentuk modul ini:
//
//	langkah 1  Inputdata.CARI10 := OperatorID.pyUserIdentifier
//	langkah 2  report FilterEmailKomiteWithLimit atas ASM-FW-GCNMFW-Int-EMAILKOMITE
//	langkah 3  InputData.CARI13 := "tampil"   bila pemanggil anggota komite
//	langkah 4  InputData.CARI13 := "tampil"   bila pemanggil operator tertentu (lihat bawah)
//	langkah …  RDB-List GetInboxListCNP_SQL     -> ListCaseInboxTeknik
//	langkah …  RDB-List KmtGetInboxListCNP_SQL  -> ListCaseInboxKomite
//	langkah 7  call GetDataTreatyinNonProp_Act  -> ListCaseInbox   (tab Admin)
//
// Ketiga halaman klipboard itulah ketiga tab di sini.
//
// # Dua hal di layar lama yang TIDAK dibawa, dan keduanya bukan selera
//
// **Nama orang menentukan kewenangan.** `GetWorkCNP_Act` langkah 4 membuka antrean komite
// bila `OperatorID.pyUserIdentifier` sama dengan satu Operator ID yang ditanam di dalam
// rule. Itu salah satu dari 24 Operator ID hardcode yang `D-15` tetapkan menjadi master
// data. Kewenangan di sini diturunkan dari peran, bukan dari nama — dan selama tabel peran
// belum dapat diisi (`TKT-F3-004`), yang berlaku adalah keadaan yang dijelaskan di
// http/routes.go, bukan sebuah nama yang disalin ke sini.
//
// **Fragmen SQL dirangkai dari string.** `GetWorkCNP_Act` menyusun potongan klausa seperti
// `"AND a.PXASSIGNEDOPERATORID = '" + OperatorID.pyUserIdentifier + "' AND …"` lalu
// menyisipkannya ke teks kueri. Itu pola `{ASIS:…}` yang menjadi celah injeksi (utang teknis
// 4.5), dan larangan perangkaian (`08-TECHNICAL-STRATEGY.md` §4.3) tidak dikecualikan oleh
// keputusan mana pun: yang direplikasi adalah perilaku bisnis, bukan celahnya.
//
// # Yang DILARANG masuk ke paket ini
//
// HTTP, SQL, driver basis data, dan bentuk JSON wire. Paket ini hanya boleh mengimpor
// pustaka standar dan seam lintas modul.
//
// # Susunan subpaket
//
//	inboxclaimtreatynonprop/          aturan modul + seam          ← paket ini
//	inboxclaimtreatynonprop/usecase/  orkestrasi: rakit tab, isi satu tab, ekspor
//	inboxclaimtreatynonprop/repo/     pengisi seam penyimpanan     — sqlstore, memory
//	inboxclaimtreatynonprop/http/     lapisan transport modul ini  — handler, dto, rute
package inboxclaimtreatynonprop

import (
	"context"
	"strings"
)

// WorkItem adalah satu baris pekerjaan klaim treaty non-proporsional.
//
// # Kenapa satu bentuk untuk kedua tab yang terisi
//
// Karena begitulah sistem lama menyusunnya: kedua kueri mengisi halaman klipboard berkelas
// `ASM-FW-GISFW-Data-Search` yang sama, dengan alias `CARI10`…`CARI24`. Yang berbeda antar
// tab hanyalah kolom mana yang terlihat — dan itu ditetapkan Tab.Columns, bukan oleh bentuk
// barisnya.
//
// # Kenapa nama isian di sini tidak mirip nama properti Pega
//
// Karena nama properti di layar ini tidak menyatakan apa pun: seluruh kolomnya bernama
// `CARI` ditambah nomor urut, dan nomornya BERBEDA dari layar Prop meski artinya sama —
// `CARI10` di sini adalah kunci teknis, sedangkan di layar Prop ia Tanggal Kejadian.
// Yang dipakai di sini adalah padanan Inggris dari `CONTEXT.md` sesuai `D-19` dan `D-80`;
// pemetaan lengkapnya ada di repo/sqlstore/inboxclaimtreatynonprop.sql.
type WorkItem struct {
	// Reference adalah kunci teknis Pega — `a.PZINSKEY` (alias `CARI10`).
	//
	// Ia dikirim ke layar tetapi TIDAK pernah digambar sebagai kolom; yang memakainya
	// adalah tombol rincian klaim.
	Reference string

	// ClaimID adalah nomor yang dibaca pengguna — `a.PXREFOBJECTINSNAME` (alias `CARI11`),
	// berjudul "No Klaim".
	//
	// Ia sekaligus KOLOM PENYARING modul ini: ketiga kueri Admin dan kueri Teknik
	// menyaring `PXREFOBJECTINSNAME LIKE 'CLMNP-%'` atas kolom yang sama. Itu berbeda dari
	// layar Prop, yang menyaring kunci objek kerjanya.
	ClaimID string

	// AssignedOperator adalah petugas atau akun antrean yang memegang penugasan ini —
	// `a.PXASSIGNEDOPERATORID`.
	//
	// Ia TIDAK dipilih satu pun kueri lama dan tidak digambar di grid mana pun. Ia dibawa
	// karena tab Admin menyaring menurut kolom ini, dan karena tanpa membawanya penyimpanan
	// memori tidak dapat meniru penyaring yang sama.
	AssignedOperator string

	// MasterID — `b.MASTERID` pada DATAPEGA.PC_ASM_FW_GCNMFW_WORK (alias `CARI19`),
	// berjudul "MasterID".
	//
	// Ia BUKAN isian yang sama dengan JSONMasterID di bawah, meski keduanya terbaca sebagai
	// "master id". Lihat catatan di sana.
	MasterID string

	// JSONMasterID — `c.data_json.IDMaster` pada POOLDATA.JSON_KLAIM (alias `CARI23`),
	// berjudul "ID Master".
	//
	// # Kenapa DUA isian untuk hal yang terdengar sama
	//
	// Karena layar lama memang menampilkan keduanya berdampingan, dan keduanya diambil dari
	// TEMPAT YANG BERBEDA: yang satu kolom pada objek kerja, yang lain nilai di dalam blob
	// JSON klaim. Menyatukannya berarti memutuskan salah satu yang benar — keputusan yang
	// tidak dapat diambil tanpa melihat isi kedua sumbernya, dan isi JSON_KLAIM belum pernah
	// dilihat (`R-08`).
	//
	// Bila kelak terbukti keduanya selalu sama, menyatukannya adalah perubahan sepele. Bila
	// ternyata berbeda, menyatukannya sekarang akan menyembunyikan ketidakcocokan data yang
	// justru perlu ketahuan.
	//
	// HANYA ketiga kueri Admin yang membawanya; kueri Teknik tidak memuatnya sama sekali.
	JSONMasterID string

	// PolicyNumber — `c.NOPOLIS` pada POOLDATA.JSON_KLAIM (alias `CARI18`), berjudul
	// "Policy No".
	//
	// Ia yang menentukan arti "TBA": kueri `GetKlaimNonPropAdminTBA_SQL` menyaring
	// `c.NOPOLIS IS NULL` — klaim treaty yang sudah masuk tetapi polisnya belum terbit.
	PolicyNumber string

	// LossDate adalah Tanggal Kejadian — `c.data_json.DateOfLoss` (alias `CARI22`).
	//
	// # Kenapa TEKS, bukan time.Time
	//
	// Karena notasi titik Oracle atas kolom JSON mengembalikan teks, dan bentuk teks di
	// dalam blob itu tidak dapat diperiksa: tidak ada DDL, dan isi `JSON_KLAIM` belum pernah
	// dilihat (`R-08`). Mengubahnya menjadi tanggal di sini berarti menebak formatnya untuk
	// seluruh baris historis. Pemformatannya dikerjakan layar, dan hanya bila bentuknya
	// memang dikenali.
	LossDate string

	// BusinessName — `b.BUSINESSNAME` (alias `CARI15`).
	//
	// Judulnya BERBEDA antar grid pada layar lama: "Business Name" pada grid Admin,
	// "Class of Business" pada grid Teknik dan Komite. Keduanya isi yang sama; perbedaan
	// judul dipertahankan lewat Tab.Columns (`D-13`).
	BusinessName string

	// BusinessSource — `b.SOBNAME` (alias `CARI14`), berjudul "Source of Business".
	BusinessSource string

	// CedingCompany adalah perusahaan asuransi yang mengalihkan risikonya kepada ASM —
	// `b.CEDINGCONAME` (alias `CARI16`), berjudul "Ceding Co Name".
	CedingCompany string

	// InsuredName — `b.INSUREDNAME` (alias `CARI17`), berjudul "Insured Name".
	InsuredName string

	// Status adalah tahap yang ditampilkan grid — alias `CARI13`.
	//
	// # Ia LITERAL di dalam kueri, bukan kolom
	//
	// `GetKlaimNonPropAdmin_SQL` dan kedua variannya memilih `'Estimation'` sebagai teks
	// tetap; `GetInboxListCNP_SQL` memilih `'Acceptation'`. Tidak ada satu pun kolom status
	// yang dibaca.
	//
	// Artinya kolom ini TIDAK menyatakan status klaim yang sebenarnya — ia menyatakan
	// ANTREAN MANA baris ini berasal. Dua klaim dengan status bisnis berbeda akan
	// menampilkan teks yang sama selama keduanya berada di antrean yang sama.
	//
	// Itu dipertahankan apa adanya (`P-5`), dan justru karena mudah disalahpahami, sifatnya
	// dinyatakan ke pengguna lewat PlannedDifferences alih-alih hanya dicatat di sini.
	// Perhatikan pula: keempat kode status klaim yang sebenarnya ada 33 (`1134`–`1166`,
	// `R-06`), dan tak satu pun dari kedua teks di atas termasuk di dalamnya.
	Status string

	// AgingDays adalah umur pekerjaan dalam HARI KALENDER —
	// `TRUNC(SYSDATE) - TRUNC(b.PXCREATEDATETIME)` (alias `CARI20`), berjudul "Aging".
	//
	// # Hari kalender, bukan hari kerja
	//
	// Pengurangan dua tanggal Oracle menghasilkan selisih hari kalender apa adanya: akhir
	// pekan dan hari libur ikut terhitung. Ia karena itu BUKAN TAT — perhitungan TAT
	// memotong hari kerja lewat `GET_WORKING_HOURS` dan `HRD_LBR` (`D-50`), dan tidak satu
	// pun dari keduanya disentuh layar ini.
	//
	// Nilainya dihitung terhadap tanggal server basis data. Ia karena itu berubah sendiri
	// setiap hari tanpa ada yang menyentuh datanya — dan itu memang yang dikehendaki.
	AgingDays int

	// CreateOperator adalah petugas yang membuat PENUGASAN ini — `a.PXCREATEOPNAME`
	// (alias `CARI12`), berjudul "Create Operator".
	CreateOperator string

	// LastUpdateOperator adalah petugas yang terakhir mengubah OBJEK KERJANYA —
	// `b.PXUPDATEOPERATOR` (alias `CARI24`), berjudul "Last Update Operator".
	//
	// Perhatikan keduanya datang dari TABEL YANG BERBEDA: yang satu dari baris penugasan,
	// yang lain dari objek kerja klaimnya. Keduanya dapat menyebut orang yang berbeda, dan
	// itu bukan ketidakcocokan data.
	LastUpdateOperator string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama.
	//
	// Itulah yang disetel ke `Inputdata.CARI10` oleh `GetDataTreatyinNonProp_Act` langkah 1
	// lalu dibandingkan dengan `PXASSIGNEDOPERATORID`. Memakai NIK di sini akan membuat tab
	// Admin tampak kosong bagi setiap pengguna.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// ClaimPrefix adalah penanda yang membedakan objek kerja klaim treaty non-proporsional dari
// objek kerja lain di tabel penugasan yang sama.
//
// Keempat kueri memakainya: `PXREFOBJECTINSNAME LIKE 'CLMNP-%'`. Ia dikumpulkan sebagai
// konstanta supaya SQL dan penyimpanan memori tidak dapat berselisih tanpa ketahuan.
//
// # Kenapa awalan, bukan potongan di tengah
//
// Karena kueri lama memang memakai `'CLMNP-%'` dengan jangkar di depan — berbeda dari layar
// Prop yang memakai `'%CLMP%'` tanpa jangkar. Perbedaan itu penting justru di modul INI:
// `CLMP` adalah awalan dari `CLMNP` bukan sebaliknya, sehingga penyaring longgar milik layar
// Prop akan ikut menangkap baris non-proporsional. Penyaring di sini yang ketat.
const ClaimPrefix = "CLMNP-"

// CommitteeClaimPrefix adalah awalan objek kerja KOMITE non-proporsional.
//
// `GetWorkCNP_Act` menyusun fragmen penyaring tab Admin sebagai
// `(PXREFOBJECTINSNAME LIKE 'KMTNP-%' OR PXREFOBJECTINSNAME LIKE 'CLMNP-%')`, sementara
// ketiga kueri Admin yang BENAR-BENAR dijalankan hanya menyaring `'CLMNP-%'`.
//
// Fragmen itu karena itu TIDAK BERPENGARUH: kueri yang membacanya tidak ada di export
// (lihat TabCommittee), dan ketiga kueri Admin memuat penyaringnya sendiri secara tetap.
// Awalan ini dicatat supaya keberadaannya di export punya keterangan — bukan supaya dipakai.
const CommitteeClaimPrefix = "KMTNP-"

// TechnicalWorkbasket adalah operator workbasket yang memegang antrean teknik treaty.
//
// Nilainya literal di `RDB List/GetInboxListCNP_SQL-SQL.xml`:
// `AND a.PXASSIGNEDOPERATORID = 'TreatyinPNCTeknik'`. Ia BUKAN nama orang melainkan akun
// fungsional, sehingga menuliskannya di sini tidak melanggar `D-67` — yang dilarang `D-15`
// adalah nilai bisnis yang berubah, sedangkan ini kunci antrean yang menentukan tab mana
// yang dibaca.
//
// Perhatikan ia SAMA dengan akun antrean layar Prop. Kedua layar berbagi satu antrean teknik
// dan dipisahkan hanya oleh awalan nomor klaimnya.
const TechnicalWorkbasket = "TreatyinPNCTeknik"

// Teks status yang ditampilkan tiap antrean.
//
// Keduanya LITERAL di dalam kueri lama, bukan kolom — lihat WorkItem.Status. Ia dikumpulkan
// di sini supaya SQL dan penyimpanan memori memakai teks yang sama persis; ejaannya
// dipertahankan apa adanya, termasuk "Acceptation" yang bukan bentuk baku bahasa Inggris.
const (
	// StatusEstimation dipakai ketiga kueri tab Admin.
	StatusEstimation = "Estimation"

	// StatusAcceptation dipakai kueri tab Teknik.
	StatusAcceptation = "Acceptation"
)

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 25
//
// Karena itu ukuran halaman grid pada `Section/InboxClaimNonProp_Harness-Section.xml`, sama
// dengan Inbox Admin dan dengan layar Prop. Ia tidak dikarang dan tidak disamakan dengan
// modul yang memakai 20.
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
// repo/sqlstore/inboxclaimtreatynonprop.sql.
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

// Repo adalah seam ke antrean klaim treaty non-proporsional SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans Repo selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat KONEKSI, bukan di tingkat
// kueri (`ADR-0030`). Tidak ada satu pun kueri di baliknya yang menyaring menurut entitas,
// dan memang tidak boleh ada.
//
// # Tidak ada satu pun operasi yang menulis
//
// Itu bukan kelalaian melainkan batas yang sama dengan modul Prop. Tombol "Create Claim
// Treaty Non Prop" di layar lama memanggil `CreateClaimTNonProp_Act`, yang membuat objek
// kerja baru di tabel milik Pega. Selama masa paralel tabel itu tetap dimiliki Pega
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
