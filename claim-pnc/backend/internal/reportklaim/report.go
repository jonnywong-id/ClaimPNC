package reportklaim

import "strings"

// Code adalah pengenal satu laporan di dalam katalog.
//
// # Kenapa slug, bukan nomor urut
//
// Ia muncul di URL (`/api/report-klaim/{kode}/ekspor`) dan di berkas uji. Nomor urut
// akan berubah artinya begitu satu panel disisipkan di tengah — dan panel di sistem lama
// memang bertambah dari waktu ke waktu. Slug tidak berubah artinya.
//
// # Kenapa BUKAN idreportKlaim milik Pega
//
// Sistem lama memakai `idreportKlaim` pada dua panel saja (`"1"` Klaim HE, `"2"` Regist
// Simas On Line), dan hanya karena keduanya berbagi satu activity. Ia bukan pengenal
// laporan — 26 panel lainnya tidak punya nilai itu sama sekali. Memakainya berarti
// mengarang nilai untuk 26 panel yang di sistem lama tidak memilikinya.
type Code string

// Group adalah pengelompokan kartu di layar.
//
// # Ia TIDAK ada di sistem lama, dan itu disengaja
//
// Harness lama menumpuk ke-28 panel dalam satu kolom tanpa pengelompokan apa pun. Pada
// layar yang dapat digulir itu berarti mencari satu laporan berarti membaca 28 judul
// berurutan.
//
// Pengelompokan ini karena itu **penambahan yang disadari**, bukan hasil pembacaan rule.
// Ia tidak mengubah satu pun laporan, satu pun penyaring, dan satu pun kolom keluaran —
// hanya urutan bacanya di layar. Urutan asli tetap terjaga di dalam kelompok, dan
// CatalogInPegaOrder mengembalikan urutan harness apa adanya bagi siapa pun yang perlu
// mencocokkannya.
type Group string

const (
	// GroupKlaim: laporan atas berkas klaim secara umum.
	GroupKlaim Group = "klaim"

	// GroupReasuransi: PLA, DLA, dan pengirimannya.
	GroupReasuransi Group = "reasuransi"

	// GroupPenyelesaian: akseptasi, komite, kasir, LOD.
	GroupPenyelesaian Group = "penyelesaian"

	// GroupLiniBisnis: laporan khusus satu lini atau satu mitra.
	GroupLiniBisnis Group = "lini-bisnis"

	// GroupOperasional: mitra, adjuster, compliance, komunikasi, produktivitas.
	GroupOperasional Group = "operasional"
)

// GroupLabel adalah judul kelompok yang dibaca pengguna.
//
// Ia berbahasa Indonesia karena ia teks yang dilihat pengguna (`D-80`), dan tidak ada
// padanannya di XML Pega — kelompoknya memang tidak ada di sana.
func GroupLabel(g Group) string {
	switch g {
	case GroupKlaim:
		return "Klaim"
	case GroupReasuransi:
		return "PLA / DLA"
	case GroupPenyelesaian:
		return "Akseptasi & Penyelesaian"
	case GroupLiniBisnis:
		return "Lini Bisnis & Mitra Kerja Sama"
	case GroupOperasional:
		return "Operasional"
	}
	return string(g)
}

// Source adalah jejak ke rule Pega yang menjadi asal satu laporan.
//
// Ia ikut dikirim ke layar dan ikut diuji. Alasannya bukan kerapian: modul ini menyalin
// 28 kueri beserta ~700 kolom, dan satu-satunya cara menyanggah "kolom ini dari mana"
// adalah menunjuk berkasnya. Tanpa jejak ini, pertanyaan itu hanya dapat dijawab dengan
// menelusuri ulang export 1,1 MiB.
type Source struct {
	// Activity adalah nama activity Pega yang dijalankan tombol Export.
	Activity string

	// SQLRule adalah rule Connect-SQL yang dijalankan activity itu.
	//
	// Kosong berarti laporannya TIDAK memakai Connect-SQL — `PNCAdjusterReport_Act`
	// memakai Report Definition, dan keduanya tidak ada di export (lihat Availability).
	SQLRule []string
}

// Availability menyatakan apakah sebuah laporan dapat dijalankan.
type Availability struct {
	// Ready bernilai false bila laporannya TIDAK dapat dijalankan.
	Ready bool

	// Reason menjelaskan sebabnya bila Ready bernilai false.
	//
	// Ia dikirim ke layar dan ditampilkan pada kartunya. Panel yang tidak dapat
	// dijalankan tetap TAMPIL — sama seperti butir menu yang belum punya layar tetap
	// tampil bertanda "belum tersedia". Menghilangkannya membuat pengguna melaporkan
	// laporan yang "hilang", dan membuat kemajuan migrasi tidak terbaca dari layar.
	Reason string

	// Blocker adalah pengenal penghalangnya pada Risk Analysis, bila ada.
	Blocker string
}

// Ready adalah Availability untuk laporan yang dapat dijalankan.
func Ready() Availability { return Availability{Ready: true} }

// Blocked adalah Availability untuk laporan yang terhalang, beserta sebab dan penghalangnya.
func Blocked(reason, blocker string) Availability {
	return Availability{Ready: false, Reason: reason, Blocker: blocker}
}

// Column adalah satu kolom pada berkas CSV keluaran.
type Column struct {
	// Header adalah judul kolom, disalin APA ADANYA dari `CSVPropHeaders` milik langkah
	// `pxConvertResultsToCSV`. Termasuk salah ketik dan spasi gandanya — berkas ini
	// dibaca ulang oleh berkas kerja dan makro yang sudah ada di sisi pengguna, dan
	// merapikan judul kolom berarti merusak keduanya tanpa ada yang meminta.
	Header string

	// Field adalah nama properti pada `CSVProperties`, dipakai sebagai kunci pengambilan
	// nilai dari baris hasil kueri.
	Field string
}

// Headers mengembalikan judul kolom saja, sesuai urutannya.
func Headers(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Header
	}
	return out
}

// Fields mengembalikan nama properti saja, sesuai urutannya.
func Fields(cols []Column) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Field
	}
	return out
}

// Action adalah satu tombol Export pada sebuah panel.
//
// # Kenapa sebuah panel dapat punya lebih dari satu
//
// Karena satu panel di sistem lama memang punya dua. "REPORT DATA KOMITE" menampilkan
// **dua tombol berdampingan** — "Export Data Approve" dan "Export Data Rejected" —
// yang memanggil activity yang sama dengan `statusapprove` `"1"` versus `"2"`.
//
// Memecahnya menjadi dua panel akan menghasilkan dua kartu berjudul sama di layar, dan
// itu terbaca sebagai cacat. Menggabungkannya menjadi satu tombol akan menghilangkan
// salah satu laporan. Panel karena itu memuat daftar aksi, dan 27 panel lainnya memuat
// tepat satu.
type Action struct {
	// Code membedakan aksi di dalam satu panel; kosong pada panel beraksi tunggal.
	Code string

	// Label adalah tulisan pada tombol, disalin APA ADANYA dari harness — termasuk
	// spasi gandanya pada "Export Data  Asuransi Kredit" dan huruf besar seluruhnya
	// pada "REPORT KOMUNIKASI KLAIM", yang memang menyimpang dari tetangganya.
	Label string

	// FixedParam adalah parameter tetap yang dipasang tombol ini di harness.
	FixedParam map[string]string
}

// Report adalah satu panel pada layar Report Klaim.
type Report struct {
	Code    Code
	Title   string
	Actions []Action
	Group   Group

	// FileBaseName adalah nama berkas CSV tanpa akhiran dan tanpa rentang tanggal.
	//
	// Disalin dari `Param.FileName` milik activity-nya. Rentang tanggal dan nama lini
	// bisnis DITAMBAHKAN saat berkas dibuat, sama seperti sistem lama yang merangkainya
	// dengan `+Param.awal+" - "+Param.akhir`.
	FileBaseName string

	// Uses menyebutkan penyaring bersama mana yang benar-benar dipakai laporan ini.
	//
	// Ia menentukan isian mana yang AKTIF di layar saat kartunya dipilih. Menampilkan
	// seluruh penyaring pada seluruh kartu akan membuat pengguna mengisi rentang tanggal
	// untuk laporan yang kuerinya tidak menerima tanggal sama sekali, lalu menyimpulkan
	// hasilnya salah.
	Uses FilterSet

	Source       Source
	Availability Availability

	// variants menyusun kolom keluaran menurut penyaring yang berlaku.
	//
	// Sembilan dari 28 laporan punya lebih dari satu susunan kolom — sistem lama memilih
	// langkah `pxConvertResultsToCSV` yang berbeda lewat precondition. Ia disimpan
	// tertutup dan dibaca lewat Columns supaya tidak ada pemanggil yang mengambil susunan
	// yang keliru dengan memilih indeks sendiri.
	variants []variant
}

// variant adalah satu susunan kolom beserta syarat berlakunya.
type variant struct {
	// when memutuskan apakah susunan ini yang berlaku. Nil berarti selalu berlaku, dan
	// hanya sah pada susunan TERAKHIR — ia penutup, bukan pilihan.
	when func(Filter) bool

	// why menjelaskan syaratnya dalam kalimat, untuk dibaca manusia dan diuji.
	why string

	columns []Column
}

// Columns mengembalikan susunan kolom yang berlaku untuk penyaring tertentu.
//
// Susunan pertama yang syaratnya terpenuhi yang dipakai — persis urutan precondition
// langkah di activity, yang juga berhenti pada yang pertama cocok.
func (r Report) Columns(f Filter) []Column {
	for _, v := range r.variants {
		if v.when == nil || v.when(f) {
			return v.columns
		}
	}
	return nil
}

// VariantReason menjelaskan susunan kolom mana yang berlaku dan mengapa.
//
// Dipakai uji dan penelusuran; tidak dikirim ke layar.
func (r Report) VariantReason(f Filter) string {
	for _, v := range r.variants {
		if v.when == nil || v.when(f) {
			return v.why
		}
	}
	return ""
}

// HasSingleColumnSet menyatakan apakah laporan ini hanya punya satu susunan kolom.
func (r Report) HasSingleColumnSet() bool { return len(r.variants) == 1 }

// Action mencari aksi menurut kodenya.
//
// Kode kosong pada panel beraksi tunggal mengembalikan satu-satunya aksi itu — supaya
// layar tidak perlu mengetahui panel mana yang beraksi jamak untuk dapat memanggilnya.
// Kode kosong pada panel beraksi jamak DITOLAK: menebak salah satunya berarti memilihkan
// "Approve" atau "Rejected" untuk pengguna, dan keduanya laporan yang berbeda isi.
func (r Report) Action(code string) (Action, bool) {
	if code == "" && len(r.Actions) == 1 {
		return r.Actions[0], true
	}
	for _, a := range r.Actions {
		if a.Code == code {
			return a, true
		}
	}
	return Action{}, false
}

// FileName menyusun nama berkas CSV keluaran.
//
// # Bentuknya mengikuti sistem lama, dengan satu perbedaan yang disengaja
//
// Sistem lama merangkai nama berkas dari potongan ekspresi, dan hasilnya tidak seragam —
// sebagian memuat rentang tanggal, sebagian memuat nama lini bisnis, sebagian tidak
// memuat apa pun. Yang di sini seragam: nama dasar, lalu rentang tanggal bila laporannya
// memang memakai tanggal.
//
// Alasannya praktis. Berkas yang diunduh berkali-kali dengan nama yang sama akan
// bertumpuk menjadi "Laporan Data PLA (3).csv" di folder unduhan, dan tidak ada satu pun
// yang menyatakan periode mana isinya. Itu bukan peniruan yang berguna.
func (r Report) FileName(f Filter) string {
	name := r.FileBaseName
	if r.Uses.Has(FilterDateRange) && !f.From.IsZero() && !f.To.IsZero() {
		name += " " + f.From.Format("20060102") + "-" + f.To.Format("20060102")
	}
	return sanitizeFileName(name) + ".csv"
}

// sanitizeFileName membuang karakter yang tidak sah pada nama berkas.
//
// Nama dasarnya berasal dari rule Pega, bukan dari pengguna — tetapi ia tetap dibersihkan
// di sini. Nama berkas masuk ke header `Content-Disposition`, dan satu tanda kutip atau
// baris baru di sana membuat header itu dapat disisipi (`11-SECURITY.md` §4.1).
func sanitizeFileName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r < 0x20, r == 0x7f:
			// dibuang
		case strings.ContainsRune(`"\/:*?<>|`, r):
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "laporan"
	}
	return out
}
