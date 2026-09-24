// Package inboxautoclaimhttp adalah lapisan transport modul Inbox Auto Claim.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `inboxautoclaimhttp` supaya tidak menutupi `net/http`.
package inboxautoclaimhttp

import "claim-pnc/internal/inboxautoclaim"

// BatchDTO adalah satu baris grid inbox yang dikirim ke peramban.
//
// Terpisah dari inboxautoclaim.Batch supaya perubahan internal tidak bocor ke klien dan
// sebaliknya (08-TECHNICAL-STRATEGY.md §2 aturan 4).
//
// Nama field-nya berbahasa Indonesia karena ia KONTRAK API, bukan nama internal (D-80),
// dan kata-katanya mengikuti judul kolom pada layar lama supaya penguji yang membandingkan
// kedua layar tidak perlu menerjemahkan apa pun.
type BatchDTO struct {
	KodePerusahaan string `json:"kode_perusahaan"`

	// NamaPerusahaan dapat kosong bila kodenya tidak ada di Master Auto Claim.
	NamaPerusahaan string `json:"nama_perusahaan"`

	Batch string `json:"batch"`

	// TanggalProses adalah TGLPROSES, dan ia bagian KUNCI PENGELOMPOKAN grid — bukan
	// sekadar kolom tampilan.
	//
	// `BrowseClaimSPKAutoClaim` mengelompokkan menurut
	// `(BATCH, INISIALID, USERINPUT, NAMA_PENERIMA, tanggal TGLPROSES)`, sehingga satu
	// nomor batch yang diunggah pada dua tanggal berbeda tampil sebagai DUA baris.
	// Menghilangkan kolom ini membuat kedua baris itu tampak kembar tanpa sebab.
	TanggalProses string `json:"tanggal_proses"`

	JumlahUpload   int `json:"jumlah_upload"`
	JumlahProses   int `json:"jumlah_proses"`
	JumlahBerhasil int `json:"jumlah_berhasil"`
	JumlahGagal    int `json:"jumlah_gagal"`

	// JumlahBelumProses dihitung server, bukan di layar. Bila layar menghitungnya
	// sendiri, dua tempat memegang satu aturan yang sama.
	JumlahBelumProses int `json:"jumlah_belum_proses"`

	UserUpload string `json:"user_upload"`
}

// LineDTO adalah satu baris rincian batch.
type LineDTO struct {
	NomorPolis string `json:"nomor_polis"`
	ProdKe     string `json:"prod_ke"`
	NomorKlaim string `json:"nomor_klaim"`
	NomorAksep string `json:"nomor_aksep"`

	// MataUang adalah KODE yang dibaca pengguna ("IDR"), bukan id yang tersimpan di
	// kolom CURRENCY.
	//
	// Layar Pega menampilkan hasil lookup ke POOLDATA.CURRENCY
	// (`BrowseClaimSPK_detail_AutoClaim-SQL.xml`), dan versi pertama modul ini
	// menampilkan id-nya — pengguna melihat "1", bukan "IDR".
	MataUang string `json:"mata_uang"`

	NilaiKlaim       string `json:"nilai_klaim"`
	PenyebabKerugian string `json:"penyebab_kerugian"`
	TanggalKejadian  string `json:"tanggal_kejadian"`
	TanggalLapor     string `json:"tanggal_lapor"`

	// TanggalProses adalah TGLPROSES — kolom kedua pada grid rincian Pega.
	TanggalProses string `json:"tanggal_proses"`

	Catatan string `json:"catatan"`
	Keyword string `json:"keyword"`

	// NamaObjek dan FlagTidakBayar adalah OBJECTNAME dan FLAGTIDAKBAYAR.
	//
	// Keduanya tidak tampil di grid rincian Pega dan tidak pernah terbaca dari kueri
	// mana pun — keberadaannya baru diketahui dari DDL. Keduanya dibawa apa adanya
	// supaya isinya dapat diperiksa saat gerbang 1, bukan karena artinya sudah
	// dipahami. Lihat keputusan-implementasi.md §18.
	NamaObjek      string `json:"nama_objek"`
	FlagTidakBayar string `json:"flag_tidak_bayar"`

	// Keterangan adalah TMP_MESSAGE apa adanya. Kosong berarti belum diproses.
	Keterangan string `json:"keterangan"`

	// Hasil menyatakan keadaan baris dalam tiga nilai yang dapat dibaca mesin:
	// "berhasil", "gagal", atau "belum". Layar memakainya untuk memilih lencana, alih-alih
	// mencocokkan teks Keterangan — teks pesan berubah-ubah, keadaan tidak.
	Hasil string `json:"hasil"`
}

// CompanyDTO adalah satu pilihan pada penyaring Nama Perusahaan.
type CompanyDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// PageDTO menyebut halaman yang sedang ditampilkan.
//
// TotalHalaman ikut dikirim walau dapat dihitung layar dari Total dan Ukuran. Sebabnya
// pembulatan: layar yang menghitungnya sendiri akan salah pada sisa halaman terakhir
// bila pembulatannya keliru, dan kesalahannya hanya muncul pada jumlah baris tertentu —
// jenis cacat yang lolos pengujian tangan.
type PageDTO struct {
	Halaman      int `json:"halaman"`
	Ukuran       int `json:"ukuran"`
	Total        int `json:"total"`
	TotalHalaman int `json:"total_halaman"`
}

// BatchListResponse adalah jawaban GET /api/inbox-auto-claim.
type BatchListResponse struct {
	Batch []BatchDTO `json:"batch"`
	Page  PageDTO    `json:"paginasi"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan
	// hukum, "data siapa ini" tidak boleh hanya diandaikan (R-20).
	Portal string `json:"portal"`
}

// LineListResponse adalah jawaban GET /api/inbox-auto-claim/{kode}/{batch}.
type LineListResponse struct {
	KodePerusahaan string    `json:"kode_perusahaan"`
	Batch          string    `json:"batch"`
	Baris          []LineDTO `json:"baris"`
	Page           PageDTO   `json:"paginasi"`
	Portal         string    `json:"portal"`
}

// CompanyListResponse adalah jawaban GET /api/inbox-auto-claim/perusahaan.
type CompanyListResponse struct {
	Perusahaan []CompanyDTO `json:"perusahaan"`
	Portal     string       `json:"portal"`
}

// CompanySummaryDTO adalah satu irisan grafik sekaligus satu baris tabel ringkasan.
type CompanySummaryDTO struct {
	Kode string `json:"kode"`

	// Nama dapat kosong bila kodenya tidak ada di Master Auto Claim; irisannya tetap ada,
	// sama seperti barisnya tetap ada di grid.
	Nama string `json:"nama"`

	// JumlahBatch memakai satuan yang SAMA dengan satu baris grid, sehingga angka di sini
	// dan total paginasi setelah disaring selalu cocok.
	JumlahBatch int `json:"jumlah_batch"`
}

// SummaryResponse adalah jawaban GET /api/inbox-auto-claim/ringkasan.
type SummaryResponse struct {
	Perusahaan []CompanySummaryDTO `json:"perusahaan"`

	// Total adalah baris "All" pada tabel ringkasan. Ia dihitung server, bukan
	// dijumlahkan layar — lihat catatan pada inboxautoclaim.Summary.
	Total  int    `json:"total"`
	Portal string `json:"portal"`
}

// UploadBatchDTO menyebut satu batch yang terbentuk dari sebuah unggahan.
type UploadBatchDTO struct {
	KodePerusahaan string `json:"kode_perusahaan"`
	NamaPerusahaan string `json:"nama_perusahaan"`
	Batch          string `json:"batch"`
	JumlahBaris    int    `json:"jumlah_baris"`

	// JumlahLolos dan JumlahBertanda adalah hasil PEMERIKSAAN SAAT UNGGAH, bukan hasil
	// pemrosesan menjadi klaim.
	//
	// Perbedaannya penting dan mudah tertukar: baris "lolos" di sini berarti lolos
	// pemeriksaan polis dan MENUNGGU diproses — ia belum menjadi klaim apa pun. Nama
	// field-nya sengaja tidak memakai kata "berhasil"/"gagal" supaya tidak terbaca
	// sebagai hasil akhir.
	JumlahLolos    int `json:"jumlah_lolos"`
	JumlahBertanda int `json:"jumlah_bertanda"`
}

// UploadRejectedDTO adalah satu baris berkas yang TIDAK disisipkan sama sekali.
type UploadRejectedDTO struct {
	Baris      int    `json:"baris"`
	NomorPolis string `json:"nomor_polis"`
	Pesan      string `json:"pesan"`
}

// UploadResponse adalah jawaban POST /api/inbox-auto-claim/unggah.
//
// Ia membedakan TIGA keadaan, dan ketiganya harus terlihat pengguna:
//
//	lolos     disisipkan, menunggu diproses
//	bertanda  disisipkan beserta pesan galatnya — persis perilaku Pega
//	ditolak   TIDAK disisipkan karena perusahaannya tidak dapat diturunkan dari polis
//
// Menggabungkan yang kedua dan ketiga akan menyesatkan: yang bertanda masih ada di
// tabel dan masih dapat dilihat di grid, yang ditolak tidak ada di mana pun. Pengguna
// yang mengira keduanya sama akan mencari baris yang tidak pernah tersimpan.
type UploadResponse struct {
	Batch       []UploadBatchDTO    `json:"batch"`
	JumlahBaris int                 `json:"jumlah_baris"`
	Ditolak     []UploadRejectedDTO `json:"ditolak"`
	Portal      string              `json:"portal"`
}

// UploadTemplateResponse menyebut bentuk berkas yang diterima.
//
// Ia dilayani server, tidak disalin ke frontend, supaya daftar kolomnya hidup di SATU
// tempat. Frontend yang memuat daftarnya sendiri akan menjadi tempat kedua yang harus
// diingat ketika flow action Pega yang asli akhirnya tiba dan judul kolomnya ternyata
// berbeda.
type UploadTemplateResponse struct {
	KolomWajib    []string `json:"kolom_wajib"`
	KolomOpsional []string `json:"kolom_opsional"`
	BatasBaris    int      `json:"batas_baris"`
}

// TabDTO adalah satu tab pada layar.
//
// Daftarnya dilayani SERVER, tidak disalin ke frontend. Sebabnya sama dengan daftar
// kolom unggahan: tab adalah pengetahuan tentang tabel mana yang ada, dan itu milik
// backend. Layar yang memuat daftarnya sendiri akan menjadi tempat kedua yang harus
// diingat begitu ada tab yang ditambah atau namanya berubah.
type TabDTO struct {
	// Kode adalah nilai yang dikirim balik sebagai ?sumber=.
	Kode string `json:"kode"`

	// Label adalah teks tab, disalin dari pyCaption harness Pega.
	Label string `json:"label"`

	// Tabel adalah nama tabel Oracle yang dibaca tab ini, ditampilkan sebagai keterangan
	// sumber data di bawah judul grid.
	//
	// Ia dikirim server, bukan diketik di layar. Sebelum ada tab, layar menuliskan
	// "POOLDATA.TMP_BATCH_AUTO_CLAIM" sebagai teks tetap — keterangan yang menjadi SALAH
	// begitu tab lain terbuka, dan salah dengan cara yang tidak terlihat: petugas yang
	// menelusuri selisih angka akan menanyakan tabel yang bukan sumbernya kepada DBA.
	Tabel string `json:"tabel"`
}

// TabListResponse adalah jawaban GET /api/inbox-auto-claim/tab.
type TabListResponse struct {
	Tab []TabDTO `json:"tab"`

	// Bawaan adalah tab yang terbuka lebih dulu.
	Bawaan string `json:"bawaan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — {kode, pesan} — ditambah `detail` untuk pelanggaran
// per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan mencocokkan
// teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
//
// Nama kuncinya `kolom`, mengikuti masterstatusprogres — bukan `field` seperti
// masterstatus. Ketidakseragaman itu sudah ada dan sudah ditampung klien di
// api/client.ts; menambah bentuk KETIGA akan memperburuknya. Penyeragamannya TKT-F1-004.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// Nilai kolom `hasil` pada LineDTO.
const (
	resultLabelSucceeded = "berhasil"
	resultLabelFailed    = "gagal"
	resultLabelPending   = "belum"
)

// toBatchDTO mengubah satu baris domain menjadi bentuk yang dikirim ke peramban.
func toBatchDTO(b inboxautoclaim.Batch) BatchDTO {
	return BatchDTO{
		KodePerusahaan:    b.CompanyCode,
		NamaPerusahaan:    b.CompanyName,
		Batch:             b.BatchNumber,
		TanggalProses:     b.ProcessedDate,
		JumlahUpload:      b.Uploaded,
		JumlahProses:      b.Processed,
		JumlahBerhasil:    b.Succeeded,
		JumlahGagal:       b.Failed,
		JumlahBelumProses: b.Pending(),
		UserUpload:        b.UploadedBy,
	}
}

// toBatchListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai [] dan bukan null — layar yang menerima null harus menjaganya sendiri, dan
// satu layar yang lupa akan gagal saat tabelnya masih kosong.
func toBatchListDTO(list []inboxautoclaim.Batch) []BatchDTO {
	result := make([]BatchDTO, 0, len(list))
	for _, b := range list {
		result = append(result, toBatchDTO(b))
	}
	return result
}

func toLineDTO(l inboxautoclaim.Line) LineDTO {
	hasil := resultLabelPending
	switch {
	case l.Succeeded():
		hasil = resultLabelSucceeded
	case l.Processed():
		hasil = resultLabelFailed
	}

	return LineDTO{
		NomorPolis:       l.PolicyNo,
		ProdKe:           l.ProductSeq,
		NomorKlaim:       l.ClaimID,
		NomorAksep:       l.AcceptanceNo,
		MataUang:         l.CurrencyCode,
		NilaiKlaim:       l.ClaimAmount,
		PenyebabKerugian: l.CauseOfLoss,
		TanggalKejadian:  l.DateOfLoss,
		TanggalLapor:     l.ReportDate,
		TanggalProses:    l.ProcessedDate,
		Catatan:          l.Note,
		Keyword:          l.Keyword,
		NamaObjek:        l.ObjectName,
		FlagTidakBayar:   l.FlagNoPayout,
		Keterangan:       l.Message,
		Hasil:            hasil,
	}
}

func toLineListDTO(list []inboxautoclaim.Line) []LineDTO {
	result := make([]LineDTO, 0, len(list))
	for _, l := range list {
		result = append(result, toLineDTO(l))
	}
	return result
}

func toSummaryListDTO(list []inboxautoclaim.CompanySummary) []CompanySummaryDTO {
	result := make([]CompanySummaryDTO, 0, len(list))
	for _, c := range list {
		result = append(result, CompanySummaryDTO{
			Kode: c.Code, Nama: c.Name, JumlahBatch: c.BatchCount,
		})
	}
	return result
}

func toCompanyListDTO(list []inboxautoclaim.Company) []CompanyDTO {
	result := make([]CompanyDTO, 0, len(list))
	for _, c := range list {
		result = append(result, CompanyDTO{Kode: c.Code, Nama: c.Name})
	}
	return result
}

// toPageDTO menyusun keterangan halaman.
func toPageDTO(request inboxautoclaim.PageRequest, total int) PageDTO {
	clean := request.Clean()

	// Pembulatan ke atas tanpa pembagian pecahan. Halaman terakhir yang hanya berisi
	// satu baris tetap dihitung sebagai satu halaman penuh.
	totalPage := 0
	if clean.Size > 0 {
		totalPage = (total + clean.Size - 1) / clean.Size
	}

	return PageDTO{
		Halaman:      clean.Number,
		Ukuran:       clean.Size,
		Total:        total,
		TotalHalaman: totalPage,
	}
}
