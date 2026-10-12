// Package masterpasal adalah inti modul Master Pasal Kerugian.
//
// # Apa yang dimodelkan di sini
//
// Sebuah **Pasal Kerugian** adalah satu butir ketentuan polis yang dirujuk saat klaim
// dinilai — entah sebagai jaminan yang menanggung, sebagai pengecualian yang menolak,
// atau sebagai pemberitahuan. Daftar baku butir-butir itulah yang dikelola modul ini.
//
// Satu baris menyimpan: nomor pasalnya, isi pasalnya, keterangan singkatnya, kategorinya,
// dan daftar **lini bisnis** tempat pasal itu berlaku.
//
// # Bentuk penyimpanannya tidak biasa, dan itu menentukan seluruh modul
//
// Tabelnya hanya punya TIGA kolom:
//
//	POOLDATA.V_M_DATA_PASAL   IDDATA (kunci) · IDPASAL (No Pasal) · JSONPASAL (CLOB)
//
// Seluruh isi form selain No Pasal hidup di dalam `JSONPASAL` sebagai satu dokumen JSON.
// Itu bukan tafsiran: `Activity/CNMInsertPasalDataMaster-Act.xml` langkah 4 mengisinya
// dengan `@GCNM.GetPageJSONString()`, dan fungsi itu — `Function/GetPageJSONString-Function.xml` —
// isinya satu baris Java:
//
//	String retValue = stepPage.getJSON(false);
//
// `false` berarti tanpa metadata Pega, sehingga yang tersimpan adalah objek JSON polos
// berisi properti page `TempPasalCol` apa adanya. Kunci-kuncinya terbukti dari
// `RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml` yang membacanya dengan `json_value`.
//
// Akibatnya bagi modul ini: **domain tidak mengenal JSON sama sekali**. Pembongkaran dan
// penyusunan dokumen itu ada di repo, karena ia bentuk penyimpanan — sama seperti nama
// kolom, yang juga tidak pernah bocor ke berkas ini.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DetailMasterPasalRejected-Harness.xml   layar "Detail Pasal Kerugian" (MENU_ID 27)
//	Section/GridDetailMasterPasalRejected-Section.xml  pembungkus — judul, Tambah, Refresh
//	Section/BrowsePasalDeatailMaster-Section.xml       isi sebenarnya — grid, form, Simpan, Delete
//	Activity/CNMInsertPasalDataMaster-Act.xml          urutan langkah simpan DAN hapus
//	Activity/PNCGetListPasalDataCOL_Act-Act.xml        urutan langkah daftar dan muat satu baris
//	RDB List/GetDataCOLByPasalBisnis_Sql-SQL.xml       daftar — json_value atas JSONPASAL
//	RDB List/BrowseCOLByPasalDataBisnis_Sql-SQL.xml    daftar lini bisnis satu pasal
//	RDB List/UpdateDPasalDataMaster-SQL.xml            panggil Pega_D_Pasal_Master
//	RDB List/DeleteDataPasalDataMaster-SQL.xml         DELETE fisik
//	Database/PEGA_D_PASAL_MASTER.prc                   sisip ATAU perbarui menurut IDDATA
//	Report Definition/BrowseBusiness_RD-RD.xml         sumber autocomplete lini bisnis
//
// # Penamaan ulang yang disengaja (D-19)
//
// Kelas Pega-nya `ASM-FW-GCNMFW-Int-V_D_CAUSE_OF_LOSS` — kelas **Detail Cause of Loss**,
// bukan kelas pasal. Propertinya ikut terbawa dari sana, dan tidak satu pun namanya
// menyebutkan isinya. Nama itu TIDAK dibawa:
//
//	M_COL_ID       -> Number       No Pasal; bukan ID cause of loss
//	DESCRIPTION    -> Text         Isi Pasal; berlabel "ISI PASAL" di grid
//	OLD_D_COL_ID   -> Description  Deskripsi; bukan ID lama apa pun
//	OLD_M_COL_ID   -> ID           kunci baris (IDDATA); bukan ID lama apa pun
//	pyCountry      -> Category     kode kategori; tidak ada urusan dengan negara
//	LOSS_CODE      -> CategoryLabel  sebutan kategori; bukan kode kerugian
//	BISNISID[].ID  -> Business[].ID    kode lini bisnis pada POOLDATA.BUSINESS
//	BISNISID[].Note -> Business[].Name nama lini bisnis
//
// Perhatikan dua yang paling mudah tertukar. `DESCRIPTION` adalah **Isi Pasal** sedangkan
// `OLD_D_COL_ID` adalah **Deskripsi** — dugaan yang wajar justru terbalik, dan buktinya
// urutan kolom grid pada `BrowsePasalDeatailMaster-Section.xml`:
//
//	No Pasal | ISI PASAL | Deskripsi | Kategori | Action
//	.M_COL_ID | .DESCRIPTION | .OLD_D_COL_ID | .LOSS_CODE | tombol
//
// Dan `LOSS_CODE` memikul DUA arti di dua kueri yang berbeda: di
// `GetDataCOLByPasalBisnis_Sql` ia sebutan kategori dari JSON, sedangkan di
// `BrowseCOLByPasalDataBisnis_Sql` ia `POOLDATA.BUSINESS.NOTE` — nama lini bisnis. Satu
// nama, dua isi yang tidak berhubungan sama sekali.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpasal

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// Kode kategori pasal — kolom `pyCountry` di dalam JSONPASAL.
//
// Ketiganya dibaca dari `Activity/CNMInsertPasalDataMaster-Act.xml` langkah 1, yang
// MENURUNKAN sebutannya dari kode yang dipilih pengguna:
//
//	TempPasalCol.LOSS_CODE := @if(...JaminanPengecualianApproval==1, "Jaminan Polis",
//	                          @if(...JaminanPengecualianApproval==2, "Pengecualian",
//	                                                                 "Notifikasi"))
//
// # Kenapa "Notifikasi" berkode KOSONG, bukan "3"
//
// Karena ekspresi di atas memakai cabang `else`, bukan `when == 3`. Daftar pilihan
// dropdown-nya sendiri hidup di Rule-Obj-Property `JaminanPengecualianApproval`, dan
// **tidak ada satu pun direktori Properties di export** (`R-16`) — jadi kode aslinya tidak
// terbaca di mana pun.
//
// Mengarang kode "3" berarti menebak, dan tebakan itu akan terbukti salah tanpa satu pun
// galat: baris lama berkode lain akan tampak benar di layar tetapi tersimpan ulang dengan
// kode yang berbeda. Yang dipakai karena itu adalah cabang `else` apa adanya — satu-satunya
// yang benar-benar terbukti.
//
// Baris lama yang berkode apa pun selain "1" dan "2" tetap terbaca "Notifikasi", persis
// seperti di Pega, dan kodenya TIDAK ditimpa selama pengguna tidak menyentuh dropdown-nya
// (lihat ClauseForm di frontend).
const (
	// CategoryPolicyCoverage — "Jaminan Polis". Pasal yang menanggung.
	CategoryPolicyCoverage = "1"

	// CategoryException — "Pengecualian". Pasal yang menolak.
	CategoryException = "2"

	// CategoryNotification — "Notifikasi". Cabang `else`; kodenya memang kosong.
	CategoryNotification = ""
)

// Category adalah satu pilihan pada dropdown Kategori.
type Category struct {
	// Code mengisi `$.pyCountry`.
	Code string

	// Label mengisi `$.LOSS_CODE`, dan itulah yang tampil di kolom Kategori.
	Label string
}

// Categories mengembalikan ketiga pilihan Kategori, urut seperti ekspresi aslinya.
//
// Ia fungsi, bukan variabel paket, supaya pemanggil tidak dapat menyunting isinya. Daftar
// yang dapat diubah dari luar berarti satu layar dapat mengubah pilihan layar lain.
func Categories() []Category {
	return []Category{
		{Code: CategoryPolicyCoverage, Label: "Jaminan Polis"},
		{Code: CategoryException, Label: "Pengecualian"},
		{Code: CategoryNotification, Label: "Notifikasi"},
	}
}

// CategoryLabel menurunkan sebutan kategori dari kodenya.
//
// Ia MEMAAFKAN kode yang tidak dikenal — apa pun selain "1" dan "2" menjadi "Notifikasi" —
// persis seperti cabang `else` pada ekspresi aslinya. Menolaknya akan membuat baris lama
// berkode lain menampilkan sel kosong yang tampak seperti cacat layar.
func CategoryLabel(code string) string {
	switch strings.TrimSpace(code) {
	case CategoryPolicyCoverage:
		return "Jaminan Polis"
	case CategoryException:
		return "Pengecualian"
	default:
		return "Notifikasi"
	}
}

// Business adalah satu lini bisnis tempat sebuah pasal berlaku.
//
// Asalnya `POOLDATA.BUSINESS`, tabel milik ruleset GISFW yang dimiliki sistem lain.
// Aplikasi ini HANYA MEMBACA-nya (ADR-0004, penulis tunggal per tabel).
type Business struct {
	// ID adalah kolom POOLDATA.BUSINESS.ID, tersimpan sebagai `$.BISNISID[].ID`.
	//
	// Ia BOLEH kosong. Autocomplete layar lama ber-`pyAllowFreeFormInput=true`
	// (`Section/BrowsePasalDeatailMaster-Section.xml`), sehingga nama yang diketik bebas
	// tersimpan tanpa kode. Perilakunya dipertahankan atas keputusan Work Owner
	// 2026-09-19 — lihat Input.Check.
	ID string

	// Name adalah kolom POOLDATA.BUSINESS.NOTE, tersimpan sebagai `$.BISNISID[].Note`.
	Name string
}

// Clause adalah satu baris Master Pasal Kerugian — POOLDATA.V_M_DATA_PASAL.
type Clause struct {
	// ID adalah kunci baris, kolom IDDATA.
	//
	// Ia diterbitkan penyimpanan, tidak pernah diketik pengguna, dan tidak pernah berubah
	// setelah baris lahir. Perhatikan ia BUKAN No Pasal: satu-satunya yang menjadikan dua
	// baris berbeda adalah kolom ini.
	ID string

	// Number adalah No Pasal yang diketik pengguna, kolom IDPASAL.
	//
	// Ia TIDAK dijamin unik. Kuncinya IDDATA, dan tidak ada satu pun constraint yang
	// diketahui melarang dua baris bernomor pasal sama (DDL-nya belum ada — `R-08`).
	// Pega pun tidak memeriksanya. Keputusan Work Owner 2026-09-19: dipertahankan apa
	// adanya.
	Number string

	// Text adalah Isi Pasal, `$.DESCRIPTION`. Di grid berlabel "ISI PASAL".
	Text string

	// Description adalah Deskripsi, `$.OLD_D_COL_ID`.
	Description string

	// Category adalah kode kategori, `$.pyCountry`.
	Category string

	// CategoryLabel adalah sebutan kategori, `$.LOSS_CODE`.
	//
	// Ia DITURUNKAN dari Category, bukan diterima dari layar — lihat CategoryLabel.
	// Ia tetap ikut disimpan, sama seperti di Pega, karena tabelnya dibaca sistem lain
	// yang mengambil teksnya dari sana alih-alih menurunkannya lagi.
	CategoryLabel string

	// Business adalah lini bisnis tempat pasal ini berlaku, `$.BISNISID[]`.
	//
	// Boleh kosong: Pega tidak mewajibkannya, dan satu-satunya isian wajib di layar lama
	// adalah No Pasal.
	Business []Business
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// ID tidak ada di sini: ia diterbitkan penyimpanan saat menambah, dan datang dari jalur
// URL saat mengubah. Menerimanya dari badan permintaan berarti membuka kemungkinan badan
// dan jalur menyebut baris yang berbeda.
//
// CategoryLabel juga tidak ada: ia turunan, dan menerimanya dari layar berarti sebutan
// yang tersimpan dapat berbeda dari kodenya.
type Input struct {
	Number      string
	Text        string
	Description string
	Category    string
	Business    []Business
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterpasal: pasal kerugian tidak ditemukan")

	// ErrIDTaken: IDDATA yang akan disisipkan sudah dipakai baris lain.
	//
	// Ia bukan kesalahan pengguna — nomornya diterbitkan penyimpanan — melainkan tanda
	// dua penambahan berjalan bersamaan.
	ErrIDTaken = errors.New("masterpasal: ID pasal kerugian sudah dipakai")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Modul ini hari ini hanya punya satu aturan, sehingga bentuk jamaknya tampak berlebihan.
// Ia tetap dipakai supaya bentuk respons galatnya sama persis dengan modul master lain —
// dan supaya aturan kedua, bila kelak ditambahkan, tidak menuntut perubahan kontrak.
type ValidationError struct {
	Violation []Violation
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterpasal: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian, dan membuang baris bisnis kosong.
//
// Dipisahkan dari Check supaya nilai yang TERSIMPAN adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// # Kenapa baris bisnis kosong dibuang di sini
//
// Layar lama memakai repeating grid ber-ikon tambah baris
// (`pzPegaDefaultGridIcons`). Menekan ikonnya menerbitkan baris kosong seketika, dan
// pengguna yang menekannya lalu berpindah pikiran meninggalkan baris tanpa isi. Pega
// menyimpannya apa adanya ke dalam JSON; di sini ia dibuang, karena baris tanpa nama dan
// tanpa kode tidak menunjuk apa pun dan hanya menambah panjang dokumen.
//
// Ini SELISIH TERENCANA terhadap Pega, dan satu-satunya yang diambil atas isian Bisnis.
func (i Input) Clean() Input {
	clean := Input{
		Number:      strings.TrimSpace(i.Number),
		Text:        strings.TrimSpace(i.Text),
		Description: strings.TrimSpace(i.Description),
		Category:    strings.TrimSpace(i.Category),
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
// # Hanya SATU aturan, dan itu memang seluruh isi layar lama
//
// `Activity/CNMInsertPasalDataMaster-Act.xml` memeriksa tepat satu hal:
//
//	langkah 1  TempPasalCol.M_COL_ID := @If(M_COL_ID != "", M_COL_ID, "UnknownID")
//	langkah 3  bila InputData.M_COL_ID == "UnknownID" -> lompat ke EXT
//	langkah 8  Property-Set-Messages "Silahkan ISI No Pasal Terlebih Dahulu"
//	           pada isian TempPasalCol.M_COL_ID
//
// Tidak ada pemeriksaan lain. Isi Pasal boleh kosong, Deskripsi boleh kosong, Kategori
// boleh tidak dipilih, daftar Bisnis boleh kosong, dan No Pasal boleh kembar.
//
// # Tiga aturan yang SENGAJA TIDAK ditambahkan
//
// Ketiganya sempat diusulkan dan ketiganya ditolak Work Owner pada 2026-09-19 dengan
// jawaban yang sama — "coba jalankan secara as is":
//
//  1. No Pasal wajib unik. Tidak diberlakukan; dua baris bernomor sama tetap dapat
//     tersimpan, persis seperti hari ini.
//  2. Isi Pasal wajib diisi. Tidak diberlakukan.
//  3. Bisnis wajib dipilih dari master, bukan diketik bebas. Tidak diberlakukan; lihat
//     Business.ID.
//
// Panjang maksimum pun tidak diberlakukan. Lebar kolom IDPASAL tidak diketahui (`R-08`),
// dan menolak berdasarkan angka yang dikarang berarti menolak isian yang sebenarnya
// diterima basis data. Bila basis data menolaknya, penolakannya datang dari sana dan
// terbaca sebagai galat teknis — kekurangan yang diterima sampai DDL-nya tiba.
func (i Input) Check() error {
	if i.Number == "" {
		return &ValidationError{Violation: []Violation{{
			Field: "no_pasal",
			// Teksnya mengikuti layar lama apa adanya (`D-13`), termasuk ejaan
			// "Silahkan" yang sudah dikenal petugas. Memperbaikinya menjadi "Silakan"
			// membuat pesan yang sama terbaca berbeda di dua sistem selama masa paralel.
			Message: "Silahkan ISI No Pasal Terlebih Dahulu.",
		}}}
	}
	return nil
}

// FormatID menyusun IDDATA dari nomor urut berikutnya.
//
// Angka polos tanpa awalan dan tanpa pemadatan lebar, direplikasi dari
// `Database/PEGA_D_PASAL_MASTER.prc:11-14`:
//
//	select max(TO_NUMBER(IDDATA)) INTO JUM_PASAL from V_M_DATA_PASAL;
//	JUM_PASAL := JUM_PASAL+1;
//	INSERT INTO POOLDATA.V_M_DATA_PASAL(IDDATA,...) VALUES(to_char(JUM_PASAL),...)
//
// `to_char` tanpa format mask tidak memberi nol di depan, sehingga bentuk "01" atau "001"
// justru akan menyimpang darinya.
func FormatID(sequence int) string { return strconv.Itoa(sequence) }

// NextSequence mencari nomor urut berikutnya yang belum dipakai.
//
// # DUA cacat procedure lama yang diperbaiki di sini
//
// Keduanya selisih terencana, dan keduanya menyangkut nomor yang diterbitkan sistem —
// bukan nilai yang diketik pengguna, sehingga tidak ada yang berubah di layar.
//
//  1. **`max()` TANPA `NVL`.** `PEGA_D_PASAL_MASTER.prc:11` menulis
//     `select max(TO_NUMBER(IDDATA)) INTO JUM_PASAL` — tanpa `NVL(...,0)` yang dipakai
//     procedure sejenis. Pada tabel KOSONG hasilnya NULL, `NULL+1` tetap NULL, dan
//     `to_char(NULL)` menyisipkan IDDATA kosong. Baris pertama pada tabel kosong karena
//     itu lahir tanpa kunci. Di sini nomor pertamanya 1.
//
//  2. **`TO_NUMBER` atas kolom teks.** Ia gagal dengan ORA-01722 begitu SATU baris saja
//     memuat nilai yang bukan angka — dan karena kolomnya bertipe teks tanpa constraint
//     yang diketahui (`R-08`), tidak ada yang mencegahnya. Menghitungnya di Go membuat
//     baris semacam itu dilewati alih-alih menghentikan seluruh penambahan.
//
// ID yang tidak dapat ditafsirkan sebagai angka DIABAIKAN saat mencari yang terbesar,
// tetapi tetap dihitung sebagai terpakai — baris lama dapat memuat apa saja, dan
// menabraknya lebih buruk daripada melewatinya.
func NextSequence(used []string) string {
	taken := make(map[string]bool, len(used))
	highest := 0
	for _, id := range used {
		clean := strings.TrimSpace(id)
		taken[clean] = true
		if number, err := strconv.Atoi(clean); err == nil && number > highest {
			highest = number
		}
	}

	// Perulangan ini nyaris selalu berhenti pada percobaan pertama. Ia ada untuk tabel
	// yang sudah memuat ID berbentuk lain, supaya baris baru tidak menabraknya.
	for number := highest + 1; ; number++ {
		if candidate := FormatID(number); !taken[candidate] {
			return candidate
		}
	}
}

// MaxLookupRows membatasi banyaknya baris yang dikembalikan pencarian lini bisnis.
//
// `Report Definition/BrowseBusiness_RD-RD.xml` memakai `pyMaxRecords=200`. Angka di sini
// lebih kecil karena ia melayani daftar pilihan yang hanya akan diambil satu — 200 baris
// di dalam satu daftar turun bukan sesuatu yang dapat dibaca manusia.
//
// Batasnya di sini, bukan di frontend: memotong di peramban berarti barisnya sudah
// terlanjur dibaca, dikirim, dan diurai.
const MaxLookupRows = 50

// MinLookupKeyword adalah panjang minimum kata kunci pencarian lini bisnis.
//
// Angkanya sama dengan modul Master Auto Claim, dan alasannya sama: kata kunci kosong
// membuat `like '%%'` menarik seluruh tabel, sedangkan dua huruf sudah membuat hasilnya
// bermakna tanpa membuat pencarian terasa rewel.
const MinLookupKeyword = 2

// Repo adalah seam ke penyimpanan Master Pasal Kerugian SATU portal.
//
// # Kenapa ada Delete di sini, padahal modul master lain tidak punya
//
// Karena layar lamanya PUNYA tombolnya, dan tombol itu menghapus barisnya secara fisik:
//
//	Section/BrowsePasalDeatailMaster-Section.xml   tombol "Delete"
//	  -> CNMInsertPasalDataMaster(DeleteFlag="1")
//	  -> RDB List/DeleteDataPasalDataMaster-SQL.xml
//	     Delete from POOLDATA.V_M_DATA_PASAL where IDDATA = {InputData.OLD_M_COL_ID}
//
// # Dan itu BERTENTANGAN dengan D-66 — secara sadar
//
// `D-66` menetapkan **soft delete menyeluruh**: tidak ada `DELETE` fisik pada data
// bernilai bisnis di sistem baru, dan penghapusan dinyatakan lewat penanda.
//
// Tabel ini tidak punya kolom penanda terhapus, dan menambahnya menempuh `D-63`
// (permintaan tertulis tim pengembang → persetujuan Work Owner → pelaksanaan DBA).
// Tiga jalan keluar diajukan ke Work Owner pada 2026-09-19 — penanda di dalam JSON,
// meminta kolom baru ke DBA, atau menunda tombolnya — dan Work Owner memilih
// **"coba jalankan secara as is"**, yaitu `DELETE` fisik seperti Pega.
//
// Keputusan itu dihormati dan dijalankan. Yang TIDAK dilakukan adalah menyembunyikannya:
// ia menyupersede `D-66` untuk tabel ini saja, tercatat di
// `docs/keputusan-implementasi.md`, dan konsekuensinya dinyatakan di muka —
// **baris yang dihapus tidak dapat dipulihkan, dan tidak meninggalkan jejak apa pun**,
// karena tabel ini juga tidak punya kolom pencatat siapa dan kapan.
type Repo interface {
	// List mengembalikan seluruh pasal kerugian.
	//
	// Daftar lini bisnis TIDAK ikut dimuat. Grid layar lama pun tidak menampilkannya —
	// kolomnya hanya No Pasal, Isi Pasal, Deskripsi, dan Kategori — dan memuatnya berarti
	// satu pembacaan master lini bisnis untuk setiap baris yang tidak terlihat siapa pun.
	List(ctx context.Context) ([]Clause, error)

	// Get mengembalikan satu pasal LENGKAP dengan daftar lini bisnisnya; ErrNotFound bila
	// tidak ada.
	//
	// Padanan `PNCGetListPasalDataCOL_Act` pada jalur `Param.idstatusp` berisi IDDATA —
	// langkah 7 membaca lini bisnisnya, langkah 10 menyusunnya menjadi daftar, langkah 11
	// mengisi keempat isian form.
	Get(ctx context.Context, id string) (Clause, error)

	// Insert menyimpan satu pasal baru.
	//
	// Nomor IDDATA diturunkan dari isi tabel DI DALAM operasi ini, bukan dipecah menjadi
	// "hitung" lalu "sisip" di lapisan aplikasi: jarak di antara keduanya adalah lubang
	// balapan yang justru sedang ditutup.
	//
	// Input sudah harus bersih dan lolos Check.
	Insert(ctx context.Context, in Input) (Clause, error)

	// Update menyimpan perubahan pada pasal yang sudah ada.
	//
	// IDDATA tidak pernah ikut berubah — procedure lama pun hanya memakainya sebagai
	// penyaring `WHERE`. ErrNotFound bila barisnya hilang di antara pemuatan layar dan
	// penyimpanan.
	Update(ctx context.Context, id string, in Input) (Clause, error)

	// Delete membuang satu baris secara FISIK; ErrNotFound bila barisnya sudah tidak ada.
	//
	// Baca peringatan pada doc comment Repo sebelum memakainya.
	Delete(ctx context.Context, id string) error
}

// LookupRepo adalah seam ke POOLDATA.BUSINESS — master lini bisnis milik sistem lain.
//
// Ia terpisah dari Repo karena menjawab pertanyaan yang berbeda — "apa pilihan yang
// tersedia" alih-alih "apa isi master ini" — dan karena tabelnya tidak pernah ditulis
// modul ini. Keduanya tetap dipilih bersama lewat RepoSelector, karena keduanya selalu
// berasal dari koneksi entitas yang sama.
type LookupRepo interface {
	// SearchBusiness mencari lini bisnis menurut nama atau kodenya.
	//
	// Asalnya autocomplete pada `Section/BrowsePasalDeatailMaster-Section.xml`, yang
	// bersumber `Report Definition/BrowseBusiness_RD-RD.xml` atas kelas
	// `ASM-FW-GISFW-Int-BUSINESS` — yaitu tabel POOLDATA.BUSINESS. Kolom yang ditampilkan
	// `.Note`, dan yang ikut disetel saat dipilih `.ID`.
	SearchBusiness(ctx context.Context, keyword string) ([]Business, error)
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
