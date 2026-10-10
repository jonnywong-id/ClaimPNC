package inboxcompliance

import (
	"fmt"
	"time"
)

// Nama berkas surat, ditiru apa adanya dari `DownloadPDFReject` langkah 6.
//
// # Hanya SATU surat yang benar-benar terbit — koreksi atas catatan sebelumnya
//
// Catatan versi pertama di sini menyebut dua surat diterbitkan sekaligus. Itu keliru, dan
// sebab kekeliruannya layak dicatat: `grep DownloadPDFReject` ikut menangkap
// `DownloadPDFRejectRefund` karena yang satu awalan yang lain, sehingga
// `Section/FormRejectClaim_section-Section.xml` terbaca seolah memanggil keduanya.
// Pembacaan langsung `pyActivity`-nya menunjukkan ia hanya memanggil **satu**:
// `DownloadPDFRejectRefund`.
//
// Jalurnya karena itu tertutup:
//
//	tombol "Generate PDF"  ->  DownloadPDFRejectRefund
//	                           langkah 7: DownloadPDFReject(FileType="RejectRefund")
//	                           langkah 9: PrintRejectCompliancePDF
//
// Keduanya merender templat yang SAMA — `HTML/RejectRefundLetter-HTML.xml` — dengan nama
// berkas yang sama. Cabang `FileType=”` pada `DownloadPDFReject`, yang akan menerbitkan
// `Surat_Penolakan.pdf` dari `HTML/RejectLetter-HTML.xml`, **tidak pernah terpanggil dari
// mana pun di export**. Templat itu mati.
//
// Pega melampirkannya dua kali karena Compliance di sana adalah work object tersendiri
// (`ASM-FW-GCNMFW-Work-Compliance`) yang terpisah dari klaim induknya: satu salinan pada
// klaim induk berkategori `RejectClaim`, satu lagi pada work object Compliance berkategori
// `RejectLetter`. Sistem baru tidak punya pemisahan itu — Compliance adalah tahap klaim,
// bukan kasus sendiri — sehingga yang tersimpan **satu dokumen**, berkategori
// RejectLetterCategory. Ini perbedaan bentuk penyimpanan, bukan perbedaan isi surat.
const (
	// RejectLetterFileName ditiru persis, termasuk garis bawahnya.
	RejectLetterFileName = "Surat_Penolakan_dan_Penarikan_Dana.pdf"

	// RejectLetterCategory adalah kategori dokumennya — `local.Category` pada
	// `PrintRejectCompliancePDF` langkah 1.
	RejectLetterCategory = "RejectLetter"
)

// namaBulanSurat dipakai menggambar "dd MMMM YYYY" pada kepala surat.
//
// Pega memakai `@DateTime.FormatDateTime(..., "dd MMMM YYYY", strTimeZone, strLocale)`
// dengan locale Indonesia, dan suratnya berbahasa Indonesia — jadi nama bulannya juga.
var namaBulanSurat = [12]string{
	"Januari", "Februari", "Maret", "April", "Mei", "Juni",
	"Juli", "Agustus", "September", "Oktober", "November", "Desember",
}

// wibSurat adalah zona yang dipakai kepala surat.
//
// Dipasang tetap +7, bukan dibaca dari basis data zona sistem: mesin produksi belum tentu
// punya basis data itu, dan surat yang tanggalnya bergeser sehari lebih buruk daripada
// surat yang zonanya tidak dapat dikonfigurasi. WIB tidak mengenal waktu musim panas,
// sehingga offset tetap memang benar sepanjang tahun.
var wibSurat = time.FixedZone("WIB", 7*60*60)

// NewRejectLetterDate menggambar tanggal surat — "dd MMMM YYYY" dalam WIB.
func NewRejectLetterDate(now time.Time) string {
	t := now.In(wibSurat)
	return fmt.Sprintf("%02d %s %04d", t.Day(), namaBulanSurat[int(t.Month())-1], t.Year())
}

// NewRejectLetterNumber merakit nomor surat.
//
// # Bentuknya ditiru PERSIS, termasuk yang terbaca aneh
//
// `DownloadPDFReject` langkah 1 merakitnya dari empat properti bernama menyesatkan —
// `AcceptanceBank`, `AcceptanceCurrency`, `AcceptanceDate`, `AcceptanceNo` — yang
// seluruhnya berisi potongan WAKTU SISTEM, bukan data bank maupun akseptasi:
//
//	AcceptanceBank      "mmss"            -> nomor urut surat
//	AcceptanceCurrency  "MM"              -> bulan
//	AcceptanceDate      "YYYY"            -> tahun
//	AcceptanceNo        "dd MMMM YYYY"    -> tanggal surat
//
// Hasilnya `{mmss}/CL.AHID.ASM/{MM}/{yyyy}`.
//
// Segmen pertama adalah MENIT dan DETIK saat surat dibuat — bukan pencacah. Akibatnya
// nomor surat TIDAK unik: dua surat yang terbit pada menit dan detik yang sama di bulan
// yang sama bernomor sama persis, dan nomornya tidak pernah naik berurutan. Ditiru apa
// adanya (`P-5`), bukan "diperbaiki" menjadi sequence — menggantinya mengubah penomoran
// surat yang keluar ke pihak luar, dan itu keputusan Work Owner.
//
// Pola yang sama sudah ditemukan di jalur RCL/PUCL (`NewLetterNumber`), sehingga ini bukan
// kekhilafan satu rule melainkan kebiasaan yang berulang di sistem lama.
func NewRejectLetterNumber(now time.Time) string {
	t := now.In(wibSurat)
	return fmt.Sprintf("%02d%02d/CL.AHID.ASM/%02d/%04d", t.Minute(), t.Second(), int(t.Month()), t.Year())
}

// RejectLetterDocument adalah seluruh bahan yang dibutuhkan untuk menggambar suratnya.
//
// Seluruh isiannya TEKS yang sudah jadi, bukan tanggal atau angka. Perendernya tidak boleh
// memutuskan bentuk tanggal maupun pemisah ribuan — itu keputusan yang sudah diambil di
// lapisan atas, dan menaruhnya di dua tempat berarti dua jawaban yang dapat berbeda.
//
// # Isinya PERSIS sebanyak merge field templat, tidak lebih
//
// Hanya enam. Sebelumnya tipe ini juga membawa kedelapan isian form dan grid Alasan,
// karena saya kira surat seharusnya memuatnya. **Work Owner menetapkan sebaliknya pada
// 2026-10-08:** templat Pega sudah merupakan format yang sesuai, dan tidak boleh diubah.
//
// Pembacaan `HTML/RejectRefundLetter-HTML.xml` memang hanya memuat enam merge field —
// seluruhnya di bawah ini. Baris a sampai e dan daftar Alasan **tergambar kosong** di
// templat, dan memang begitu adanya: suratnya dicetak lalu dilengkapi tangan, sebagaimana
// `Rp._____________` pada paragraf pengembalian dana sejak awal menunjukkannya.
//
// Karena itu tipe ini memuat **hanya yang digambar**. Isian form tetap diterima API
// (formnya ada di Pega) tetapi tidak sampai ke sini — pembuangannya terjadi di satu
// tempat yang tercatat, bukan tersebar sebagai medan yang tidak pernah dipakai.
type RejectLetterDocument struct {
	// LetterDate — "Jakarta, <LetterDate>".
	LetterDate string

	// LetterNumber — "No. <LetterNumber>".
	LetterNumber string

	// RecipientCompany dan RecipientPerson adalah baris "Kepada Yth.".
	//
	// Dua isian, bukan satu, karena templat MEMBEDAKANNYA — dan bukan hanya di baris
	// alamat. Pada "Tembusan" ia memilih kalimat yang berbeda:
	//
	//	Customer_P.pyFirstName kosong  ->  "Direktur <Company>"
	//	Customer_C.pyCompany   kosong  ->  "Pihak Sdr. <Person>"
	//
	// Menggabungkan keduanya menjadi satu nama akan menghilangkan pilihan itu, dan surat
	// yang menyapa badan usaha dengan "Pihak Sdr." salah alamat secara harfiah.
	//
	// Keduanya datang dari snapshot polis (`Policy.CIFData`). Klaim hanya membawa `QQNAME`
	// tanpa penanda perorangan atau badan usaha, sehingga untuk sekarang pemanggil mengisi
	// salah satunya dan membiarkan yang lain kosong — lihat catatan di lapisan usecase.
	RecipientCompany string
	RecipientPerson  string

	// RecipientAddress adalah alamat tertanggung.
	//
	// TERGAMBAR KOSONG untuk sekarang. Templat mengambilnya dari
	// `Policy.CIFData.AddressList(1).ASMAddress` — alamat pada snapshot polis, yang tidak
	// punya kolom di `T_CLAIM_PNC`. `REPORTADDRESS` yang ada di tabel itu adalah alamat
	// PELAPOR, bukan alamat tertanggung, dan memakainya berarti mengarang.
	RecipientAddress string

	// SubjectName adalah nama yang muncul pada baris "Perihal" dan "Lampiran".
	//
	// Dari `ObjectList(1).ObjectName` — objek pertanggungan pertama, bukan isian form.
	// `RejectPage.AcceptanceStream` di Pega pun dibaca dari klaim, bukan dari isian "Nama
	// Pasien" yang disunting petugas.
	SubjectName string
}

// RejectLetterRenderer adalah seam ke pembentuk PDF.
//
// Dideklarasikan di sisi pemakai, seperti seam lain di modul ini: domain menyatakan apa
// yang dibutuhkannya, dan paket `rejectpdf` yang memenuhinya. Itu yang membuat alur
// suratnya dapat diuji tanpa membentuk PDF sungguhan.
type RejectLetterRenderer interface {
	Render(RejectLetterDocument) ([]byte, error)
}

// RejectPrefill adalah isian yang diambilkan dari klaim saat form dibuka.
//
// Hanya TIGA dari empat isian `AutoFillFormReject_Pre`. Yang keempat —
// `TanggalSelesaiRawatInap` — tidak punya kolom di `T_CLAIM_PNC`; lihat kueri
// `find_reject_prefill`.
type RejectPrefill struct {
	IncidentDate  *time.Time
	IncidentPlace string
	PatientName   string
}
