package inboxautoclaim

import (
	"fmt"
	"strings"
)

// Berkas CSV yang dihasilkan kedua tombol ekspor.
//
// Nama berkas dan judul kolomnya DISALIN APA ADANYA dari
// Activity/REPORT_AUTO_CLAIM_ACT-Act.xml, yang memanggil pxConvertResultsToCSV dua kali
// dengan parameter berbeda menurut precondition Local.TYPE:
//
//	Local.TYPE=="BERHASIL"
//	  FileName        = "Laporan Hasil Klaim"
//	  CSVPropHeaders  = "Inisial, No Polis, No Klaim, No Ref Bank, No Aksep,Currency, Nilai Klaim, No Objek, Keterangan"
//	  CSVProperties   = "THEINSURED,NOPOLIS,IDPEGA,WARRANTYNO,EDMNO,STATUSBUSINESS,BRANCHCODE,CLIENTID,FLAGEDMBATAL"
//
//	Local.TYPE=="GAGAL"
//	  FileName        = "Laporan Hasil Gagal Klaim"
//	  CSVPropHeaders  = "Inisial, No Polis, No Klaim, No Aksep,Currency, Nilai Klaim, No Objek, Keterangan"
//	  CSVProperties   = "THEINSURED,NOPOLIS,IDPEGA,EDMNO,STATUSBUSINESS,BRANCHCODE,CLIENTID,FLAGEDMBATAL"
//
// Berkas GAGAL punya SATU kolom lebih sedikit — "No Ref Bank" tidak ada di sana. Itu
// bukan kekeliruan pembacaan; kedua daftar memang berbeda panjang di export, dan
// perbedaannya dipertahankan supaya berkas yang diunduh petugas tetap sama bentuknya.
//
// # Isi tiap kolom kini TERBUKTI, bukan lagi dugaan
//
// Versi pertama berkas ini menandai dua kolom sebagai DUGAAN karena kueri yang mengisinya
// belum ada di export. Kueri itu kemudian diterima
// (InboxAutoClaim/BrowseReportClaimSPK_AutoClaim-SQL.xml), dan ia membuktikan **kedua
// dugaan itu keliru**:
//
//	kolom          properti Pega    isi sebenarnya           dugaan lama   benar?
//	Inisial        THEINSURED       INISIALID                INISIALID     ya
//	No Polis       NOPOLIS          NOPOLIS                  NOPOLIS       ya
//	No Klaim       IDPEGA           IDPEGA, prefix dipangkas  IDPEGA utuh  hampir
//	No Ref Bank    WARRANTYNO       gl.t_claim_asuransi_credit@asmd  KEYWORD  TIDAK
//	No Aksep       EDMNO            NOAKSEPTASI              NOAKSEPTASI   ya
//	Currency       STATUSBUSINESS   lookup POOLDATA.CURRENCY  CURRENCY(id) hampir
//	Nilai Klaim    BRANCHCODE       NILAIKLAIM               NILAIKLAIM    ya
//	No Objek       CLIENTID         COL_ID                   PRODKE        TIDAK
//	Keterangan     FLAGEDMBATAL     TMP_MESSAGE              TMP_MESSAGE   ya
//
// Ini alasan konkret mengapa dugaan tidak boleh diam-diam naik derajat menjadi fakta:
// dua dari sembilan salah, dan keduanya baru terlihat setelah kuerinya dibaca.
//
// # Penamaan warisan yang menyesatkan dan sengaja tidak diperbaiki di sini
//
// Judul kolomnya tidak cocok dengan isinya, dan sekarang terbukti lebih jauh melenceng
// daripada yang terlihat semula: "No Objek" berisi KODE PENYEBAB KERUGIAN, "Currency"
// diisi properti bernama STATUSBUSINESS, dan "Nilai Klaim" diisi properti bernama
// BRANCHCODE.
//
// D-19 menyuruh menamai ulang, tetapi yang dinamai ulang adalah NAMA DI DALAM KODE,
// bukan judul kolom pada berkas yang sudah dipakai pihak ketiga. Berkas ini dibaca
// perusahaan rekanan di luar Sinarmas; mengubah judulnya adalah perubahan kontrak
// keluaran, bukan penggantian nama. Penamaan ulangnya karena itu berhenti di batas
// berkas: nama field Go bersih, judul kolom CSV tetap seperti aslinya.
//
// # Satu kolom yang TIDAK DAPAT diisi hari ini
//
// "No Ref Bank" diambil lewat DB Link:
//
//	(select no_ref_bank from gl.t_claim_asuransi_credit@asmd.sinarmas.co.id
//	  where no_aksep = a.noakseptasi)
//
// D-25 mengganti seluruh DB Link dengan pemanggilan API, dan API penggantinya belum ada
// (R-03). Kolomnya karena itu **tetap ada di berkas tetapi selalu kosong**, dan itu
// dinyatakan di muka sebagai selisih yang sudah diketahui — bukan ditemukan sebagai
// kejutan saat gerbang 1.
//
// Kolomnya tidak dibuang karena membuang kolom mengubah bentuk berkas bagi pembacanya di
// luar Sinarmas; kolom kosong masih dapat dibaca, kolom hilang menggeser seluruh kolom
// sesudahnya.
const (
	// ExportFileNameSucceeded adalah nama berkas ekspor baris berhasil.
	ExportFileNameSucceeded = "Laporan Hasil Klaim"

	// ExportFileNameFailed adalah nama berkas ekspor baris gagal.
	ExportFileNameFailed = "Laporan Hasil Gagal Klaim"
)

// pegaClaimPrefix adalah awalan kunci teknis Pega yang menempel di depan nomor klaim.
//
// Panjangnya tepat 19 karakter, dan itulah sebabnya kueri ekspor memangkas mulai posisi
// 20 (Oracle menghitung dari 1). Lihat StripClaimPrefix.
const pegaClaimPrefix = "ASM-FW-GCNMFW-WORK "

// StripClaimPrefix membuang awalan kunci teknis Pega dari nomor klaim.
//
// # Apa yang dilakukan kueri aslinya
//
//	case when idpega like '%PNC%' then substr(IDPEGA,20,30) else idpega end
//
// Yaitu: bila nilainya memuat "PNC", ambil karakter ke-20 sampai 30 karakter berikutnya.
// Angka 20 bukan angka sembarang — ia panjang awalan "ASM-FW-GCNMFW-WORK " ditambah satu,
// sehingga yang tersisa persis nomor klaimnya. Ini kebocoran kunci teknis Pega ke data
// bisnis yang dicatat sebagai utang teknis 4.1 dan dijawab D-22/D-71.
//
// # Kenapa saya TIDAK menyalin syaratnya apa adanya
//
// Syarat `like '%PNC%'` juga cocok dengan nilai yang **tidak** berawalan Pega. Pada nilai
// yang lebih pendek dari 20 karakter, `substr(...,20,30)` mengembalikan **teks kosong** —
// nomor klaimnya hilang dari berkas tanpa satu pun tanda.
//
// Hari ini itu tidak pernah terjadi, karena nilai yang memuat "PNC" di tabel ini selalu
// nomor klaim berawalan Pega. Tetapi D-71 menetapkan nomor klaim sistem baru berbentuk
// `PNCN.YY.xxxx` — yang memuat "PNC" dan **jauh lebih pendek dari 20 karakter**. Menyalin
// syaratnya apa adanya berarti setiap nomor klaim buatan sistem baru terbit kosong di
// berkas yang dikirim ke perusahaan rekanan.
//
// Karena itu syaratnya diganti menjadi "berawalan prefix Pega", bukan "memuat PNC".
//
// # Kenapa ini tetap setara dengan Pega
//
// Untuk **setiap** nilai yang benar-benar ada di tabel hari ini, kedua aturan memberi
// hasil yang sama persis:
//
//	nilai                            like '%PNC%'   berawalan prefix   hasil
//	ASM-FW-GCNMFW-WORK PNC-1865      ya             ya                 PNC-1865 (sama)
//	Penerima klaim tidak ditemukan   tidak          tidak              utuh (sama)
//	No Polis tidak di temukan        tidak          tidak              utuh (sama)
//
// Selisih hanya muncul pada nilai yang **tidak pernah dibuat Pega**. Jadi ini bukan
// perubahan perilaku terhadap data yang dibandingkan pada gerbang 1, melainkan penutupan
// lubang yang akan terbuka begitu klaim bernomor baru masuk ke jalur ini.
func StripClaimPrefix(claimID string) string {
	if !strings.HasPrefix(claimID, pegaClaimPrefix) {
		return claimID
	}

	sisa := claimID[len(pegaClaimPrefix):]

	// Batas 30 karakter dari substr(...,20,30) dipertahankan. Ia tidak pernah menggigit
	// pada data nyata — nomor klaim terpanjang jauh di bawah itu — tetapi membuangnya
	// berarti berkas dapat memuat nilai lebih panjang daripada yang pernah dihasilkan
	// sistem lama, dan itu selisih yang tidak ada gunanya.
	if len(sisa) > 30 {
		sisa = sisa[:30]
	}

	return sisa
}

// exportHeaderSucceeded adalah judul kolom berkas BERHASIL — sembilan kolom.
//
// Spasi setelah koma pada CSVPropHeaders asli DIBUANG: ia artefak penulisan daftar di
// Pega, bukan bagian nama kolom, dan membawanya akan membuat setiap judul kolom di
// berkas berawalan spasi.
var exportHeaderSucceeded = []string{
	"Inisial", "No Polis", "No Klaim", "No Ref Bank", "No Aksep",
	"Currency", "Nilai Klaim", "No Objek", "Keterangan",
}

// exportHeaderFailed adalah judul kolom berkas GAGAL — delapan kolom, tanpa "No Ref Bank".
var exportHeaderFailed = []string{
	"Inisial", "No Polis", "No Klaim", "No Aksep",
	"Currency", "Nilai Klaim", "No Objek", "Keterangan",
}

// ExportSpec adalah bentuk satu berkas ekspor: namanya, judul kolomnya, dan cara satu
// baris domain dipetakan menjadi satu baris berkas.
type ExportSpec struct {
	// FileName tanpa akhiran .csv; yang menambahkannya lapisan transport.
	FileName string

	Header []string

	// Row memetakan satu Line menjadi satu baris berkas.
	Row func(Line) []string
}

// ExportSpecFor memilih bentuk berkas sesuai hasil yang diminta.
//
// ResultAll tidak punya berkas: layar lama hanya menyediakan EXPORT BERHASIL dan EXPORT
// GAGAL, dan menambahkan "ekspor semua" berarti menerbitkan berkas yang tidak pernah ada
// — dengan judul kolom yang harus dikarang, karena kedua berkas yang ada pun berbeda.
func ExportSpecFor(result Result) (ExportSpec, error) {
	switch result {
	case ResultSucceeded:
		return ExportSpec{
			FileName: ExportFileNameSucceeded,
			Header:   exportHeaderSucceeded,
			Row: func(l Line) []string {
				return []string{
					l.CompanyCode,               // Inisial       <- INISIALID
					l.PolicyNo,                  // No Polis      <- NOPOLIS
					StripClaimPrefix(l.ClaimID), // No Klaim      <- IDPEGA dipangkas
					l.BankRefNo,                 // No Ref Bank   <- DB Link, KOSONG (R-03)
					l.AcceptanceNo,              // No Aksep      <- NOAKSEPTASI
					l.CurrencyCode,              // Currency      <- lookup CURRENCY
					l.ClaimAmount,               // Nilai Klaim   <- NILAIKLAIM
					l.CauseOfLoss,               // No Objek      <- COL_ID
					l.Message,                   // Keterangan    <- TMP_MESSAGE
				}
			},
		}, nil

	case ResultFailed:
		return ExportSpec{
			FileName: ExportFileNameFailed,
			Header:   exportHeaderFailed,
			Row: func(l Line) []string {
				return []string{
					l.CompanyCode,               // Inisial
					l.PolicyNo,                  // No Polis
					StripClaimPrefix(l.ClaimID), // No Klaim
					l.AcceptanceNo,              // No Aksep
					l.CurrencyCode,              // Currency
					l.ClaimAmount,               // Nilai Klaim
					l.CauseOfLoss,               // No Objek
					l.Message,                   // Keterangan
				}
			},
		}, nil

	default:
		return ExportSpec{}, fmt.Errorf(
			"%w: ekspor hanya tersedia untuk hasil %q dan %q", ErrUnknownResult,
			ResultSucceeded, ResultFailed)
	}
}
