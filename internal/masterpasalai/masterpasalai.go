// Package masterpasalai adalah inti modul Master Pasal AI.
//
// # Apa yang dimodelkan di sini
//
// Daftar **wording polis** yang dipakai penilaian AI atas sebuah klaim — satu baris per
// butir ketentuan, terdiri dari nomor pasalnya, ayatnya, dan kejadian yang dicakupnya.
//
// # Modul ini TIDAK MENULIS, dan itu terbukti dari layar lamanya
//
// `Section/DetailMasterPasalAI_sect.xml` ber-`pyEditingMode = readOnly` (`:4405`), dan satu-
// satunya tombol yang ada di sana adalah **Cari** (`:1833`) dan **Refresh** (`:2222`). Tidak
// ada Tambah, Simpan, Ubah, maupun Hapus — dipastikan dengan menelusuri `pyDeleteActivity`
// dan `pyAppendActivity` yang seluruhnya kosong, serta `pyGridDeleteActivityExists` dan
// `pyGridAppendActivityExists` yang keduanya `false`.
//
// Bentuknya karena itu mengikuti modul Master Reas: satu method `List` pada seam Repo, satu
// rute `GET`, tanpa `Get`, `Insert`, `Update`, maupun `Delete`.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DetailMasterPasalAI-harness.xml     layar "Detail Pasal AI" (MENU_ID 36) — kerangka
//	Section/DetailMasterPasalAI_sect.xml        isi layar — grid, kotak cari, dua tombol
//	Activity/GetListPasalAI_act.xml             pengisi grid — penyaring, paginasi, cacah
//	RDB List/GetListDataPasalAI_SQL.xml         kuerinya — tabel, kolom, urutan
//	RDB List/CountDataPasalAI_SQL.xml           cacah baris yang cocok
//
// # Penamaan ulang yang WAJIB dilakukan di sini (D-19)
//
// Ketiga kolom layar lama terikat properti klipboard yang namanya **tidak ada hubungannya
// sama sekali** dengan isinya:
//
//	No Pasal   .City       ← bukan kota
//	Ayat       .CityID     ← bukan kode kota
//	Kejadian   .District   ← bukan kecamatan
//
// Itu bukan salah baca. Section ini Save-As berlapis — `BrowseDetailSuveryors` (2017) →
// `BrowsePasalDeatailMaster` (2022) → section ini (2023) — dan properti layar surveyor lama
// dipakai ulang apa adanya. Persis utang teknis 4.2 pada Steering
// (`a.LOCATION AS "RISKLOCATION"`).
//
// Nama kolom basis datanya baru terbaca dari activity, di dalam ekspresi penyaringnya
// (`Activity/GetListPasalAI_act.xml:548`):
//
//	WP_ID        -> ID          kunci baris; tidak digambar grid
//	WP_PASAL     -> Number      No Pasal
//	WP_AYAT      -> Paragraph   Ayat
//	WP_KEJADIAN  -> Event       Kejadian
//
// Tabelnya `POOLDATA.MST_PASAL_AI`, terbaca dari
// `RDB List/GetListDataPasalAI_SQL.xml`. Alias yang dipakai kueri itu — `"CaseID"`,
// `"City"`, `"CityID"`, `"District"` — TIDAK dibawa: ia dipilih agar cocok dengan properti
// klipboard warisan, bukan karena ada hubungannya dengan isinya.
//
// Awalan `WP` hampir pasti **Wording Polis**, sejalan dengan kolom `PENGGUNAAN_WORDING_POLIS`
// dan `WORDING_POLIS` pada `POOLDATA.T_CLAIM_DATA_RESULTS_AI`. Itu **dugaan yang masuk akal,
// bukan hal yang terbukti** — tidak ada satu pun rule di export yang menyatakannya, dan ia
// TIDAK dipakai untuk menurunkan apa pun.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpasalai

import (
	"context"
	"strings"
)

// PageSize adalah banyaknya baris per halaman.
//
// **25**, dibaca dari `Activity/GetListPasalAI_act.xml:874`:
//
//	Pagination.PageSize := 25
//
// # Kenapa BUKAN 30, meski section menyebut angka itu
//
// `Section/DetailMasterPasalAI_sect.xml:4472` memang memuat `pyPageSize = 30` di dalam
// `pyGridProps`. Angka itu **tidak berlaku**: grid-nya ber-`pyPageMode = None` (`:4488`),
// sehingga ia tidak memaginasi apa pun sendiri. Paginasinya dikerjakan di server lewat
// `FirstRow`/`LastRow` yang dihitung activity, lalu ditampilkan section `ButtonPagingInbox`.
//
// Angka 30 itu setelan mati — sekelompok dengan `BrowseVDSurveyors_RD` yang juga tertinggal
// di `pyGridProps` dari layar surveyor. Pada section hasil Save-As berlapis, setelan yang ADA
// belum tentu setelan yang BERLAKU.
//
// # Ia milik domain, bukan frontend
//
// Berbeda dari seluruh layar master lain di aplikasi ini, yang memaginasi di peramban dengan
// ukuran halaman sebagai prop layar. Di sini paginasinya memang sudah di server sejak di
// Pega, sehingga angkanya menentukan JENDELA YANG DIBACA dari basis data — bukan sekadar
// berapa baris yang digambar.
const PageSize = 25

// FirstPage adalah nomor halaman pertama.
//
// Satu, bukan nol: `Pagination.FirstRow := ((.CurrentIndex-1) * .PageSize) + 1`
// (`Activity/GetListPasalAI_act.xml:895`) hanya benar bila `CurrentIndex` mulai dari 1.
const FirstPage = 1

// Clause adalah satu baris Master Pasal AI — `POOLDATA.MST_PASAL_AI`.
type Clause struct {
	// ID adalah kolom WP_ID — kunci baris.
	//
	// Ia TIDAK digambar grid: layar lamanya hanya punya tiga kolom. Tetapi kueri lamanya
	// tetap memilihnya (`WP_ID AS "CaseID"`) dan **mengurutkan dengannya**
	// (`ORDER BY WP_ID`), sehingga ia bagian dari apa yang dibaca — bukan tambahan.
	//
	// Ia ikut dikirim ke layar sebagai kunci baris tabel. Tanpa itu, kunci harus disusun
	// dari gabungan No Pasal dan Ayat — dan tidak ada satu pun constraint yang diketahui
	// melarang dua baris berpasangan sama (DDL-nya belum ada, `R-08`).
	ID string

	// Number adalah kolom WP_PASAL — di grid berlabel "No Pasal".
	Number string

	// Paragraph adalah kolom WP_AYAT — di grid berlabel "Ayat".
	Paragraph string

	// Event adalah kolom WP_KEJADIAN — di grid berlabel "Kejadian".
	//
	// Ia satu-satunya yang digambar `pxTextArea`, bukan `pxTextInput`
	// (`Section/DetailMasterPasalAI_sect.xml:4145`), dan kolomnya paling lebar — 283 px
	// berbanding 48 dan 47. Isinya karena itu teks panjang, bukan sandi.
	//
	// **Artinya belum terbukti.** Tidak ada master, rule, maupun keterangan di export yang
	// menjelaskan apa yang dicatat di sana. Layar menampilkannya apa adanya.
	Event string
}

// IsEmpty menyatakan ketiga isian baris ini kosong seluruhnya.
//
// Dipakai penyimpanan memori untuk membuang baris contoh yang tidak menunjuk apa pun. Ia
// TIDAK dipakai sebagai aturan bisnis: tidak ada satu pun pemeriksaan semacam itu di layar
// lama, dan modul ini tidak menulis apa pun sehingga tidak ada yang perlu ditolak.
func (c Clause) IsEmpty() bool {
	return strings.TrimSpace(c.Number) == "" &&
		strings.TrimSpace(c.Paragraph) == "" &&
		strings.TrimSpace(c.Event) == ""
}

// Filter adalah penyaring daftar.
//
// Hanya DUA hal, dan keduanya terbukti dari layar lama: satu kata kunci, dan nomor halaman.
// Tidak ada penyaring status — tabelnya tidak dikenal punya kolom semacam itu, dan layarnya
// tidak bertab (`pyTabbedHeader = false`).
type Filter struct {
	// Keyword adalah isi kotak "Cari".
	//
	// Di layar lama ia properti `TempSearch.Country`
	// (`Section/DetailMasterPasalAI_sect.xml:1521`) — sekali lagi nama warisan yang tidak
	// ada hubungannya dengan negara.
	//
	// Satu kata kunci dicocokkan ke KETIGA kolom sekaligus, digabung `OR`
	// (`Activity/GetListPasalAI_act.xml:548`). Bukan tiga isian terpisah.
	Keyword string

	// Page adalah nomor halaman yang diminta, mulai dari FirstPage.
	//
	// Nilai di bawah FirstPage dinaikkan ke FirstPage oleh Clean, bukan ditolak: nomor
	// halaman datang dari tautan paginasi, dan halaman yang tidak masuk akal berarti
	// tautannya basi — bukan pengguna melakukan kesalahan.
	Page int
}

// Clean memangkas spasi kata kunci dan menormalkan nomor halaman.
//
// Dipisahkan dari pemakaiannya supaya nilai yang DIPAKAI MENCARI adalah nilai yang sudah
// dipangkas — bukan nilai mentah yang kebetulan lolos karena spasinya ikut dicocokkan.
func (f Filter) Clean() Filter {
	clean := Filter{
		Keyword: strings.TrimSpace(f.Keyword),
		Page:    f.Page,
	}
	if clean.Page < FirstPage {
		clean.Page = FirstPage
	}
	return clean
}

// Offset mengembalikan banyaknya baris yang dilewati sebelum halaman ini.
//
// Padanan `Pagination.FirstRow := ((.CurrentIndex-1) * .PageSize) + 1`
// (`Activity/GetListPasalAI_act.xml:895`), dikurangi satu karena `OFFSET` menghitung dari
// nol sedangkan `FirstRow` menghitung dari satu.
//
// Filter sudah harus melewati Clean; halaman di bawah FirstPage menghasilkan 0.
func (f Filter) Offset() int {
	if f.Page <= FirstPage {
		return 0
	}
	return (f.Page - FirstPage) * PageSize
}

// Page adalah satu halaman hasil pencarian.
//
// Total ikut dikembalikan karena layar lama pun memuatnya lewat kueri TERPISAH —
// `CountDataPasalAI` (`Activity/GetListPasalAI_act.xml:663`), yang hasilnya masuk ke
// `Pagination.TotalData` (`:1000`). Tanpa angka itu, paginator tidak dapat menghitung ada
// berapa halaman.
type Page struct {
	// Clause adalah baris pada halaman ini, paling banyak PageSize.
	Clause []Clause

	// Total adalah cacah SELURUH baris yang cocok dengan kata kuncinya — bukan cacah baris
	// pada halaman ini.
	Total int

	// Number adalah nomor halaman yang benar-benar dikembalikan.
	//
	// Ia belum tentu sama dengan yang diminta: permintaan halaman di luar jangkauan
	// dikembalikan sebagai halaman kosong, dan nomornya dikembalikan apa adanya supaya
	// layar dapat menyatakan halaman mana yang sedang dilihat.
	Number int
}

// PageCount menghitung banyaknya halaman dari Total.
//
// Nol baris tetap menghasilkan SATU halaman — halaman pertama yang kosong — karena layar
// selalu menggambar paginator, dan "halaman 1 dari 0" tidak dapat dibaca siapa pun.
func (p Page) PageCount() int {
	if p.Total <= 0 {
		return 1
	}
	count := p.Total / PageSize
	if p.Total%PageSize != 0 {
		count++
	}
	return count
}

// Repo adalah seam ke penyimpanan Master Pasal AI SATU portal.
//
// Satu method saja, dan itu memang seluruh kemampuan layar lamanya. Tidak ada `Get` — layar
// lamanya tidak punya jalur membuka satu baris, dan ketiga kolomnya sudah muat di dalam grid
// sehingga tidak ada yang tersisa untuk ditampilkan.
type Repo interface {
	// List mengembalikan satu halaman hasil beserta cacah seluruh baris yang cocok.
	//
	// Keduanya dikembalikan bersama, bukan lewat dua method: di Pega pun keduanya dijalankan
	// berurutan di dalam SATU activity (`CountDataPasalAI` lalu `GetListDataPasalAI`), dan
	// memisahkannya membuka kemungkinan cacah dan isinya dibaca dari keadaan yang berbeda.
	//
	// Filter sudah harus melewati Clean.
	List(ctx context.Context, filter Filter) (Page, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan wording polis satu
// badan hukum kepada pengguna badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
