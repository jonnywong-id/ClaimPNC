// Package detailpenyebab adalah inti modul Detail Penyebab Kerugian.
//
// # Apa yang dimodelkan di sini
//
// Sebuah **Detail Penyebab Kerugian** adalah satu rincian di bawah sebuah **Master
// Penyebab Kerugian**. Master menyebut sebab kerugiannya secara umum — kebakaran,
// kecelakaan, kehilangan — dan detail inilah yang memecahnya menjadi butir-butir yang
// benar-benar dipilih petugas saat klaim diregistrasi.
//
// Satu baris menyimpan: ID-nya, master yang menaunginya, deskripsi kerugiannya, kode
// kehilangannya, status aktifnya, dan daftar **lini bisnis** tempat detail itu berlaku.
//
// # Hubungan induk-anak, dan arah yang benar membacanya
//
//	POOLDATA.V_M_CAUSE_OF_LOSS    induk — M_COL_ID, COL_DESC, OLD_M_COL_ID
//	         ^
//	         | M_COL_ID
//	POOLDATA.V_D_CAUSE_OF_LOSS    modul INI — D_COL_ID, M_COL_ID, DESCRIPTION,
//	         ^                                 LOSS_CODE, STS_AKTIF, OLD_D_COL_ID
//	         | D_COL_ID
//	POOLDATA.V_D_CAUSE_OF_LOSS_BUSINESS   lini bisnis per detail — D_COL_ID, BISNISID
//
// Induknya adalah **Master Penyebab Kerugian** (MENU_ID 20, `CauseOfLossInbox`) yang
// BELUM dibangun. Modul ini karena itu hanya MEMBACA induknya, sebagai daftar pilihan —
// tidak menulis satu baris pun ke sana. Saat modul induk dibangun kelak, ia memiliki
// tabelnya; modul ini tetap pembaca (ADR-0004, penulis tunggal per tabel).
//
// # Bentuk penyimpanannya tidak biasa, dan itu menentukan seluruh modul
//
// Yang DITULIS dan yang DIBACA bukan objek yang sama:
//
//	ditulis   POOLDATA.D_CAUSE_OF_LOSS     D_COL_ID (kunci) · JSONDATA (CLOB)
//	dibaca    POOLDATA.V_D_CAUSE_OF_LOSS   enam kolom, view atas tabel di atas
//
// Buktinya `Database/PEGA_D_CAUSE_OF_LOSS.prc:22,33` — satu-satunya jalur tulisnya:
//
//	INSERT INTO POOLDATA.D_CAUSE_OF_LOSS(D_COL_ID,JSONDATA) VALUES(id_dcol_ins, …);
//	UPDATE POOLDATA.D_CAUSE_OF_LOSS SET JSONDATA = DataPega WHERE D_COL_ID = IDPega;
//
// sedangkan `RDB List/QueryGetAllDataCauseOfLoss-SQL.xml:92-100` membacanya lewat view
// berkolom. Jadi seluruh isi form selain D_COL_ID hidup di dalam `JSONDATA` sebagai satu
// dokumen JSON, dan view itulah yang membentangkannya menjadi kolom.
//
// Dokumen JSON-nya ditulis `@GCNM.GetPageJSONString()`
// (`Activity/CNMInsertDetailCauseOfLoss_act-Act.xml:397`) — fungsi yang sama yang dipakai
// Master Pasal Kerugian, isinya satu baris Java `stepPage.getJSON(false)`. `false` berarti
// tanpa metadata Pega, sehingga yang tersimpan adalah properti page `TempDcol` apa adanya.
//
// Akibatnya bagi modul ini: **domain tidak mengenal JSON sama sekali**. Pembongkaran dan
// penyusunan dokumen itu ada di repo, karena ia bentuk penyimpanan — sama seperti nama
// kolom, yang juga tidak pernah bocor ke berkas ini.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DetailCauseOfLoss-Harness.xml             layar "Detail Penyebab Kerugian" (MENU_ID 38)
//	Section/ListDetailCauseOfLoss-Section.xml         pembungkus defer-load, tanpa isi
//	Section/BrowseDetailCauseOfLoss-Section.xml       isi sebenarnya — grid, form, Simpan, Ubah, Cari
//	Activity/CNMInsertDetailCauseOfLoss_act-Act.xml   urutan langkah simpan
//	Activity/CNMSetDetailCauseOfLoss_act-Act.xml      urutan langkah muat satu baris ke form
//	RDB List/QueryGetAllDataCauseOfLoss-SQL.xml       keenam kolom view, urut DESCRIPTION
//	RDB List/UpdateDCauseOfLoss-SQL.xml               panggil PEGA_D_CAUSE_OF_LOSS
//	RDB List/GetLBUID_SQL-SQL.xml                     lini bisnis satu detail
//	RDB List/BrowseCOLByBisnis_Sql-SQL.xml            pencarian menurut lini bisnis
//	Report Definition/SelectVDCauseOfLoss_RD-RD.xml   pemuat form — keenam properti
//	Report Definition/BrowseVMCauseOfLoss_RD-RD.xml   daftar pilihan master penyebab
//	Report Definition/BrowseBusiness_RD-RD.xml        daftar pilihan lini bisnis
//	Database/PEGA_D_CAUSE_OF_LOSS.prc                 sisip ATAU perbarui menurut D_COL_ID
//
// # Satu rule yang HILANG, dan apa akibatnya
//
// **`BrowseVDCauseOfLoss_RD` tidak ada di export** (`R-16`). Ia Report Definition yang
// mengisi grid utama — `Section/BrowseDetailCauseOfLoss-Section.xml:8051,8418,8582,8673,8815`
// menyebutnya lima kali, dan pencarian atas seluruh export hanya menemukan section itu
// sendiri.
//
// Cakupan daftarnya karena itu **rekonstruksi**, bukan bacaan. Dasarnya kuat: dua rule
// lain atas kelas yang SAMA (`ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS`) membaca view yang sama
// dengan keenam kolom yang sama —
// `QueryGetAllDataCauseOfLoss-SQL.xml` (`order by DESCRIPTION`) dan
// `SelectVDCauseOfLoss_RD-RD.xml` (`pyMaxRecords=500`) — dan keempat kolom yang benar-benar
// tampil di grid adalah bagian dari keenamnya.
//
// Yang TIDAK dapat dipastikan tanpa rule itu: apakah grid lamanya menyaring sesuatu
// (misalnya hanya yang aktif). Modul ini **tidak menyaring** — sama seperti
// `QueryGetAllDataCauseOfLoss` — sehingga baris tidak aktif tetap terlihat dan statusnya
// terbaca di kolom Status Aktif. Menyembunyikannya akan membuat petugas menganggapnya
// hilang, dan tidak ada tombol mana pun untuk memunculkannya kembali.
//
// # Penamaan ulang yang disengaja (D-19)
//
// Nama kolomnya menyebut "cause of loss", tetapi tiga di antaranya tidak menyebutkan
// isinya sama sekali. Nama itu TIDAK dibawa:
//
//	D_COL_ID      -> ID           kunci baris
//	OLD_D_COL_ID  -> LegacyID     ID pada sistem sebelum Pega; bukan penampung lain
//	M_COL_ID      -> MasterID     kunci induk di V_M_CAUSE_OF_LOSS
//	COL_DESC      -> MasterLabel  sebutan induk; hanya untuk ditampilkan
//	DESCRIPTION   -> Description  berlabel "Deskripsi Kerugian" di layar
//	LOSS_CODE     -> LossCode     berlabel "Kode Kehilangan"
//	STS_AKTIF     -> Active       berlabel "Status Aktif"
//	BISNISID[]    -> Business[]   daftar lini bisnis
//
// # Satu nama yang memikul DUA arti, dan ia mudah sekali tertukar
//
// `OLD_D_COL_ID` adalah **kolom data** — ID warisan sebuah baris, dimuat ke form oleh
// `Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1297-1299`.
//
// Tetapi pada JALUR SIMPAN, properti dengan nama yang sama dipakai sebagai **saluran
// pengangkut dokumen JSON**: `CNMInsertDetailCauseOfLoss_act-Act.xml:394-397` mengisi
// `InputData.OLD_D_COL_ID` dengan `@GCNM.GetPageJSONString()`, lalu
// `RDB List/UpdateDCauseOfLoss-SQL.xml:83` menyerahkannya sebagai `Datapega`.
//
// Keduanya TIDAK bertabrakan karena berada di page yang berbeda — `TempDcol` untuk data,
// `InputData` untuk pengangkut — tetapi membacanya sepintas akan menyimpulkan ID warisan
// tertimpa JSON setiap kali disimpan. Ia tidak.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package detailpenyebab

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// Nilai kolom STS_AKTIF — isian berlabel "Status Aktif".
//
// # Daftar pilihannya TIDAK ada di export, dan ini yang dipakai sebagai gantinya
//
// Isiannya `pxDropdown` ber-`pyListSource = associated`
// (`Section/BrowseDetailCauseOfLoss-Section.xml:2123`), artinya pilihannya datang dari
// Rule-Obj-Property `STS_AKTIF` sendiri — dan **tidak ada satu pun direktori Properties di
// export** (`R-16`).
//
// Yang dipakai karena itu adalah nilai yang terbukti dari PEMAKAIANNYA, bukan dari daftar
// pilihannya. Seluruh perbandingan `STS_AKTIF` di export berbunyi `= '1'` — 14 kemunculan
// huruf besar dan 14 huruf kecil, tidak satu pun membandingkannya dengan nilai lain.
// Modul Master Supplier di aplikasi ini sudah memodelkannya demikian
// (`mastersupplier.ActiveYes`/`ActiveNo`), dan menyimpang darinya akan membuat dua layar
// menuliskan arti yang berbeda ke dalam kolom yang sama.
const (
	// ActiveYes — baris berlaku. Inilah satu-satunya nilai yang pernah dibandingkan.
	ActiveYes = "1"

	// ActiveNo — baris tidak berlaku.
	//
	// Ia adalah lawan dari ActiveYes, bukan nilai yang terbaca dari export. Tidak ada satu
	// pun rule yang membandingkan `STS_AKTIF = '0'`; yang ada hanyalah penyaring `= '1'`,
	// sehingga apa pun selain "1" sudah berperilaku sebagai tidak aktif.
	ActiveNo = "0"
)

// ActiveOption adalah satu pilihan pada dropdown Status Aktif.
type ActiveOption struct {
	Code  string
	Label string
}

// ActiveOptions mengembalikan kedua pilihan Status Aktif.
//
// Ia fungsi, bukan variabel paket, supaya pemanggil tidak dapat menyunting isinya. Daftar
// yang dapat diubah dari luar berarti satu layar dapat mengubah pilihan layar lain.
func ActiveOptions() []ActiveOption {
	return []ActiveOption{
		{Code: ActiveYes, Label: "Aktif"},
		{Code: ActiveNo, Label: "Tidak Aktif"},
	}
}

// IsActive menyatakan apakah sebuah nilai STS_AKTIF berarti berlaku.
//
// Ia MEMAAFKAN nilai yang tidak dikenal — apa pun selain "1" berarti tidak aktif — persis
// seperti penyaring `STS_AKTIF = '1'` di sistem lama. Menolaknya akan membuat baris lama
// bernilai lain gagal dibaca, padahal di Pega baris itu hanya tidak ikut tersaring.
func IsActive(code string) bool { return strings.TrimSpace(code) == ActiveYes }

// ActiveLabel menurunkan sebutan status dari nilainya.
//
// Nilai KOSONG dibedakan dari nilai "0". Keduanya sama-sama berarti tidak aktif bagi
// penyaring, tetapi di layar keduanya berbeda artinya bagi petugas: "0" adalah pilihan
// yang pernah diambil seseorang, sedangkan kosong berarti baris itu lahir sebelum isiannya
// ada — dan tabelnya tidak punya constraint NOT NULL yang diketahui (`R-08`).
func ActiveLabel(code string) string {
	switch strings.TrimSpace(code) {
	case ActiveYes:
		return "Aktif"
	case "":
		return "Belum diisi"
	default:
		return "Tidak Aktif"
	}
}

// Business adalah satu lini bisnis tempat sebuah detail penyebab kerugian berlaku.
//
// Asalnya `POOLDATA.BUSINESS`, tabel milik ruleset GISFW yang dimiliki sistem lain.
// Aplikasi ini HANYA MEMBACA-nya (ADR-0004, penulis tunggal per tabel). Master Pasal
// Kerugian membaca tabel yang sama lewat seam-nya sendiri; keduanya sengaja tidak berbagi
// kode, supaya perubahan di satu modul tidak menyeret modul lain.
type Business struct {
	// ID adalah kolom POOLDATA.BUSINESS.ID, tersimpan sebagai `$.BISNISID[].ID`.
	//
	// Inilah yang dijoinkan view lini bisnis:
	// `RDB List/GetLBUID_SQL-SQL.xml:33` berbunyi
	// `from V_D_CAUSE_OF_LOSS_BUSINESS a, BUSINESS b where a.BISNISID = b.ID`.
	ID string

	// Name adalah kolom POOLDATA.BUSINESS.NOTE, tersimpan sebagai `$.BISNISID[].Note`.
	//
	// Ia ikut disimpan meski dapat diturunkan dari ID, karena itulah yang dilakukan
	// `Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1784-1785` saat memuat form, dan
	// dokumen JSON yang ditulis ulang harus memuat bentuk yang sama agar view yang
	// membentangkannya tidak berubah artinya.
	Name string
}

// CauseOfLossDetail adalah satu baris Detail Penyebab Kerugian.
type CauseOfLossDetail struct {
	// ID adalah kunci baris, kolom D_COL_ID.
	//
	// Ia diterbitkan penyimpanan, **tidak pernah diketik pengguna** — isiannya
	// `pyReadOnly = true` (`Section/BrowseDetailCauseOfLoss-Section.xml:1475`) — dan tidak
	// pernah berubah setelah baris lahir.
	//
	// Bentuknya: kode situs disambung nomor urut empat digit berpadding nol, dari
	// `Database/PEGA_D_CAUSE_OF_LOSS.prc:19`:
	//
	//	id_dcol_ins := id_site || lpad(to_Char(D_CAUSE_SEQ.nextval),4,'0');
	ID string

	// LegacyID adalah kolom OLD_D_COL_ID — ID baris ini pada sistem sebelum Pega.
	//
	// Ia TIDAK digambar di form mana pun, tetapi DIMUAT dan DISIMPAN ulang
	// (`Activity/CNMSetDetailCauseOfLoss_act-Act.xml:1297-1299`), sehingga membuangnya akan
	// menghapus tautan ke sistem lama pada setiap baris yang pernah disunting.
	//
	// Baca peringatan pada banner paket sebelum menyimpulkan apa pun dari namanya.
	LegacyID string

	// MasterID adalah kunci induk, kolom M_COL_ID di V_M_CAUSE_OF_LOSS.
	//
	// Ia BOLEH kosong. Isiannya tidak bertanda wajib, dan jalur simpannya tidak
	// memeriksanya — lihat Input.Check.
	MasterID string

	// MasterLabel adalah sebutan induk, kolom COL_DESC.
	//
	// Ia **tidak tersimpan** di baris ini: ia dibaca dari induknya saat ditampilkan.
	// Di layar lama ia isi `pyPrompt` pada autocomplete
	// (`Section/BrowseDetailCauseOfLoss-Section.xml:1817`), dan di pencarian ia sub-kueri
	// (`RDB List/BrowseCOLByBisnis_Sql-SQL.xml:64`):
	//
	//	(select col_desc from v_m_cause_of_loss where m_col_id = a.m_col_id) as "pyNote"
	//
	// Kosong berarti induknya tidak ditemukan — baris yatim, yang mungkin saja ada karena
	// tidak ada foreign key yang diketahui (`R-08`).
	MasterLabel string

	// Description adalah Deskripsi Kerugian, kolom DESCRIPTION.
	//
	// Inilah kolom yang mengurutkan daftar di sistem lama (`order by DESCRIPTION`).
	Description string

	// LossCode adalah Kode Kehilangan, kolom LOSS_CODE.
	LossCode string

	// Active adalah Status Aktif, kolom STS_AKTIF. Lihat ActiveYes dan ActiveNo.
	Active string

	// Business adalah lini bisnis tempat detail ini berlaku, `$.BISNISID[]`.
	//
	// Ia TIDAK ikut dimuat pada daftar — grid lamanya pun tidak menampilkannya — dan hanya
	// dibaca saat satu baris dibuka. Lihat Repo.List dan Repo.Get.
	Business []Business
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// ID tidak ada di sini: ia diterbitkan penyimpanan saat menambah, dan datang dari jalur
// URL saat mengubah. Menerimanya dari badan permintaan berarti membuka kemungkinan badan
// dan jalur menyebut baris yang berbeda.
//
// MasterLabel juga tidak ada: ia milik induk, dan menerimanya dari layar berarti sebutan
// yang ditampilkan dapat berbeda dari sebutan yang sebenarnya tersimpan di induknya.
//
// LegacyID ADA di sini, meski tidak digambar di layar mana pun. Alasannya ada pada
// CauseOfLossDetail.LegacyID: ia dimuat dan disimpan ulang oleh sistem lama, dan tanpa
// jalur untuk mengembalikannya, setiap penyimpanan akan mengosongkannya.
type Input struct {
	LegacyID    string
	MasterID    string
	Description string
	LossCode    string
	Active      string
	Business    []Business
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("detailpenyebab: detail penyebab kerugian tidak ditemukan")

	// ErrIDTaken: D_COL_ID yang akan disisipkan sudah dipakai baris lain.
	//
	// Ia bukan kesalahan pengguna — nomornya diterbitkan penyimpanan — melainkan tanda
	// deret nomornya sudah pernah dipakai, misalnya karena sequence di-reset.
	ErrIDTaken = errors.New("detailpenyebab: ID detail penyebab kerugian sudah dipakai")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "detailpenyebab: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian, dan membuang baris bisnis kosong.
//
// Dipisahkan dari Check supaya nilai yang TERSIMPAN adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// # Kenapa baris bisnis kosong dibuang di sini
//
// Isian Bisnis adalah repeating grid (`pyPageListProperty = TempDcol.BISNISID`,
// `Section/BrowseDetailCauseOfLoss-Section.xml:3251`). Menekan ikon tambah baris
// menerbitkan baris kosong seketika, dan pengguna yang berpindah pikiran meninggalkannya
// tanpa isi. Pega menyimpannya apa adanya ke dalam JSON; di sini ia dibuang, karena baris
// tanpa nama dan tanpa kode tidak menunjuk apa pun.
//
// Ini SELISIH TERENCANA terhadap Pega, dan satu-satunya yang diambil atas isian Bisnis.
// Perlakuan yang sama dipakai Master Pasal Kerugian atas isian yang bentuknya sama.
func (i Input) Clean() Input {
	clean := Input{
		LegacyID:    strings.TrimSpace(i.LegacyID),
		MasterID:    strings.TrimSpace(i.MasterID),
		Description: strings.TrimSpace(i.Description),
		LossCode:    strings.TrimSpace(i.LossCode),
		Active:      strings.TrimSpace(i.Active),
	}

	for _, business := range i.Business {
		tidy := Business{
			ID:   strings.TrimSpace(business.ID),
			Name: strings.TrimSpace(business.Name),
		}
		if tidy.ID == "" && tidy.Name == "" {
			continue
		}
		clean.Business = append(clean.Business, tidy)
	}
	return clean
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Layar lamanya TIDAK memeriksa apa pun — nol aturan
//
// Itu bukan kesimpulan dari membaca sepintas. `CNMInsertDetailCauseOfLoss_act` diperiksa
// seluruhnya, dan kelima langkahnya adalah:
//
//	langkah 1  Property-Set  InputData.D_COL_ID := @if(TempDcol.D_COL_ID!="", …, "UnknownID")
//	langkah 2  Property-Set  InputData.OLD_D_COL_ID := @GCNM.GetPageJSONString()
//	langkah 3  RDB-List      UpdateDCauseOfLoss  -> PEGA_D_CAUSE_OF_LOSS
//	langkah 4  Property-Set  TempDcol.pyNote := OutputData.STS_AKTIF   (pesan dari procedure)
//	langkah 5  Property-Set  TempDcol.pyLabel := "Close"
//
// **Nol `Property-Set-Messages`, nol precondition, nol isian ber-`pyRequired = true`.**
// Baris yang seluruh isiannya kosong pun tersimpan.
//
// # Kenapa tidak ditambahkan aturan sendiri
//
// `P-5` menetapkan perilaku dipertahankan lebih dulu dan diperbaiki kemudian, dan
// preseden di aplikasi ini sudah ada: pada Master Pasal Kerugian, tiga aturan yang
// diusulkan — nomor wajib unik, isi wajib diisi, bisnis wajib dari master — **ditolak Work
// Owner pada 2026-09-19** dengan jawaban "coba jalankan secara as is".
//
// Yang diberlakukan di sini hanyalah satu pemeriksaan yang TIDAK menolak isian pengguna
// melainkan menolak nilai yang tidak mungkin berasal dari layar: Status Aktif di luar
// daftar pilihannya. Alasannya ada di bawah.
//
// # Satu-satunya aturan: Status Aktif harus salah satu pilihannya
//
// Isiannya dropdown dengan dua pilihan, sehingga nilai di luar keduanya tidak dapat
// dikirim layar ini. Yang ditolak karena itu bukan kesalahan ketik petugas melainkan
// permintaan yang datang dari luar layar — dan membiarkannya lewat berarti menulis nilai
// yang tidak dapat ditampilkan kembali oleh dropdown yang sama.
//
// **Kosong tetap diterima.** Isiannya tidak wajib, dan baris lama yang kolomnya belum
// pernah diisi harus tetap dapat disimpan ulang tanpa dipaksa memilih — memaksanya berarti
// mengubah data yang tidak diminta siapa pun untuk diubah.
//
// Panjang maksimum tidak diberlakukan. Lebar kolomnya tidak diketahui (`R-08`), dan
// menolak berdasarkan angka yang dikarang berarti menolak isian yang sebenarnya diterima
// basis data.
func (i Input) Check() error {
	var violation []Violation

	if i.Active != "" && i.Active != ActiveYes && i.Active != ActiveNo {
		violation = append(violation, Violation{
			Field: "status_aktif",
			Message: "Status Aktif hanya dapat bernilai Aktif atau Tidak Aktif. " +
				"Pilih salah satunya dari daftar.",
		})
	}

	if len(violation) == 0 {
		return nil
	}
	return &ValidationError{Violation: violation}
}

// SequenceWidth adalah lebar nomor urut pada D_COL_ID, dipadatkan dengan nol di depan.
//
// **Empat**, dari `Database/PEGA_D_CAUSE_OF_LOSS.prc:19`:
//
//	id_dcol_ins := id_site || lpad(to_Char(D_CAUSE_SEQ.nextval),4,'0');
//
// Perhatikan induknya memakai **tiga** (`PEGA_M_CAUSE_OF_LOSS.prc:19`, `lpad(...,3,'0')`).
// Keduanya sengaja tidak disatukan: kedua deret itu memang berbeda lebarnya, dan
// menyeragamkannya akan menerbitkan ID yang tidak sebentuk dengan yang sudah ada.
const SequenceWidth = 4

// FormatID menyusun D_COL_ID dari kode situs dan nomor urut.
//
// Nomor urut yang melampaui empat digit TIDAK dipotong — `lpad` pada Oracle pun tidak
// memotong, ia hanya berhenti memadatkan. ID yang lebih panjang lebih baik daripada ID
// yang terpotong dan bertabrakan dengan baris lain.
func FormatID(site string, sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	if padding := SequenceWidth - len(digits); padding > 0 {
		digits = strings.Repeat("0", padding) + digits
	}
	return strings.TrimSpace(site) + digits
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan pencarian.
//
// Kedua Report Definition yang menyuapi isiannya memakai `pyMaxRecords` yang besar —
// `BrowseVMCauseOfLoss_RD` 500 dan `BrowseBusiness_RD` 200 — lalu menyaringnya di peramban.
// Angka di sini lebih kecil karena ia melayani daftar pilihan yang hanya akan diambil
// satu; lima ratus baris di dalam satu daftar turun bukan sesuatu yang dapat dibaca
// manusia.
//
// Batasnya di sini, bukan di frontend: memotong di peramban berarti barisnya sudah
// terlanjur dibaca, dikirim, dan diurai.
const MaxLookupRows = 50

// MinLookupKeyword adalah panjang minimum kata kunci pencarian.
//
// Angkanya sama dengan Master Pasal Kerugian dan Master Auto Claim, dan alasannya sama:
// kata kunci kosong membuat `like '%%'` menarik seluruh tabel, sedangkan dua huruf sudah
// membuat hasilnya bermakna tanpa membuat pencarian terasa rewel.
const MinLookupKeyword = 2

// Filter mempersempit daftar Detail Penyebab Kerugian.
//
// # Asalnya panel "Cari Data" di layar lama
//
// Panel itu punya dua isian dan dua tombol — `Cari Data` dan `Clear Pencarian`
// (`Section/BrowseDetailCauseOfLoss-Section.xml:5397,6069,6394`) — dan keduanya terikat
// pada page `TempSearchBisnis`: `.M_COL_ID` (`:5721`) dan `.D_COL_ID` (`:5797`).
//
// Kuerinya `RDB List/BrowseCOLByBisnis_Sql-SQL.xml`, dan ia menyaring lewat
// `{ASIS:TempSearchBisnis.DESCRIPTION}` — potongan klausa SQL yang **dirangkai dari nilai
// property**, persis utang teknis 4.5 pada Steering. Klausa itu tidak dibawa: penyaringnya
// di sini adalah parameter, bukan teks yang disambung.
type Filter struct {
	// Keyword mempersempit pada Deskripsi Kerugian, Kode Kehilangan, dan ID.
	//
	// Ia lebih luas daripada panel lama, yang hanya menyaring menurut lini bisnis. Alasan:
	// grid lamanya menampilkan ID, Deskripsi Kerugian, Status Aktif, dan Kode Kehilangan —
	// dan pencarian yang tidak menjangkau kolom yang terlihat akan terbaca sebagai cacat.
	Keyword string

	// MasterID mempersempit pada satu induk, padanan `TempSearchBisnis.M_COL_ID`.
	MasterID string

	// BusinessID mempersempit pada satu lini bisnis, padanan penyaring `EXISTS` atas
	// V_D_CAUSE_OF_LOSS_BUSINESS pada `BrowseCOLByBisnis_Sql`.
	BusinessID string
}

// Repo adalah seam ke penyimpanan Detail Penyebab Kerugian SATU portal.
//
// # TIDAK ADA Delete, dan itu bukan kelalaian
//
// Layar lamanya memang tidak punya tombolnya: tombol yang ada hanya `Simpan`, `Ubah`,
// `Cari Data`, dan `Clear Pencarian`
// (`Section/BrowseDetailCauseOfLoss-Section.xml:4981,10276,6069,6394`). `pyDeleteSQL` pada
// `RDB List/UpdateDCauseOfLoss-SQL.xml:6` kosong, dan tidak ada satu pun activity penghapus
// di export.
//
// Modul ini karena itu **tidak bertentangan dengan `D-66`** — berbeda dari Master Pasal
// Kerugian, yang tombol hapusnya nyata dan memaksa penghapusan fisik.
//
// Yang harus disadari: baris yang tidak lagi dipakai dinyatakan lewat **Status Aktif**,
// bukan dibuang. Itulah gunanya kolom itu, dan itulah satu-satunya cara menonaktifkan
// sebuah detail.
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	//
	// Daftar lini bisnis TIDAK ikut dimuat. Grid layar lama pun tidak menampilkannya —
	// kolomnya hanya ID, Deskripsi Kerugian, Status Aktif, dan Kode Kehilangan — dan
	// memuatnya berarti satu pembacaan view lini bisnis untuk setiap baris yang tidak
	// terlihat siapa pun.
	//
	// MasterLabel IKUT dimuat meski juga tidak ada di grid lama, karena pencarian lama pun
	// mengambilnya (`BrowseCOLByBisnis_Sql` sub-kueri `as "pyNote"`) dan tanpa itu kolom
	// induk pada layar baru tidak dapat diisi.
	List(ctx context.Context, filter Filter) ([]CauseOfLossDetail, error)

	// Get mengembalikan satu baris LENGKAP dengan daftar lini bisnisnya; ErrNotFound bila
	// tidak ada.
	//
	// Padanan `CNMSetDetailCauseOfLoss_act`: langkah 3 memuat keenam kolom lewat
	// `SelectVDCauseOfLoss_RD`, lalu langkah terakhir membaca lini bisnisnya lewat
	// `GetLBUID_SQL` dan menyusunnya menjadi daftar.
	Get(ctx context.Context, id string) (CauseOfLossDetail, error)

	// Insert menyimpan satu baris baru dan mengembalikannya beserta ID yang diterbitkan.
	//
	// Nomor D_COL_ID diterbitkan DI DALAM operasi ini, bukan dipecah menjadi "hitung" lalu
	// "sisip" di lapisan aplikasi: jarak di antara keduanya adalah lubang balapan.
	//
	// Input sudah harus bersih dan lolos Check.
	Insert(ctx context.Context, in Input) (CauseOfLossDetail, error)

	// Update menyimpan perubahan pada baris yang sudah ada.
	//
	// D_COL_ID tidak pernah ikut berubah — procedure lama pun hanya memakainya sebagai
	// penyaring `WHERE`. ErrNotFound bila barisnya hilang di antara pemuatan layar dan
	// penyimpanan.
	Update(ctx context.Context, id string, in Input) (CauseOfLossDetail, error)
}

// LookupRepo adalah seam ke dua master yang hanya DIBACA modul ini.
//
// Keduanya disatukan dalam satu interface karena keduanya menjawab pertanyaan yang sama —
// "apa pilihan yang tersedia untuk isian ini" — dan keduanya selalu berasal dari koneksi
// entitas yang sama.
type LookupRepo interface {
	// SearchMaster mencari Master Penyebab Kerugian menurut sebutan atau kodenya.
	//
	// Asalnya autocomplete pada isian "ID Master Kerugian"
	// (`Section/BrowseDetailCauseOfLoss-Section.xml:1814-1819`), yang bersumber
	// `Report Definition/BrowseVMCauseOfLoss_RD-RD.xml` atas kelas
	// `ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS`. Nilai yang disetel `.M_COL_ID`, yang
	// ditampilkan `.COL_DESC`.
	SearchMaster(ctx context.Context, keyword string) ([]MasterOption, error)

	// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
	//
	// Asalnya autocomplete pada grid "Bisnis"
	// (`Section/BrowseDetailCauseOfLoss-Section.xml:3819`), bersumber
	// `Report Definition/BrowseBusiness_RD-RD.xml` atas kelas `ASM-FW-GISFW-Int-BUSINESS`
	// — yaitu POOLDATA.BUSINESS.
	SearchBusiness(ctx context.Context, keyword string) ([]Business, error)
}

// MasterOption adalah satu pilihan pada isian "ID Master Kerugian".
//
// Ia BUKAN Business dengan nama lain: keduanya berasal dari tabel yang berbeda, dan
// menyatukannya akan membuat perubahan pada salah satunya menyeret yang lain.
type MasterOption struct {
	// ID adalah kolom M_COL_ID.
	ID string

	// Label adalah kolom COL_DESC.
	Label string
}

// Store menyatukan kedua seam yang dipakai layanan modul ini.
//
// Keduanya tetap DIDEKLARASIKAN terpisah karena dapat berubah sendiri-sendiri. Yang
// disatukan hanyalah CARA MEMILIHNYA: keduanya selalu berasal dari koneksi entitas yang
// sama, sehingga dua pemilih terpisah hanya akan membuka kemungkinan keduanya menunjuk
// entitas yang berbeda.
type Store interface {
	Repo
	LookupRepo
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
type RepoSelector func(portalAlias string) (Store, error)
