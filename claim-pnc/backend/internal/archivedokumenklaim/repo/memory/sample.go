package memory

import (
	_ "embed"
	"time"

	"claim-pnc/internal/archivedokumenklaim"
	"claim-pnc/internal/platform/sampledata"
)

// sampleJSON memuat seluruh data contoh paket ini; lihat platform/sampledata.
//
//go:embed sample.json
var sampleJSON []byte

// Data contoh untuk mode pengembangan tanpa basis data.
//
// # Kenapa isinya karangan, dan kenapa itu justru yang benar
//
// `D-64` menetapkan staging memakai salinan data produksi apa adanya — dan aturan kerja
// proyek melarang menuliskan data nasabah ke berkas yang di-commit (`D-69`). Data di sini
// karena itu SELURUHNYA karangan: nomor polis, nama tertanggung, dan nomor klaimnya tidak
// menunjuk siapa pun.
//
// Yang ditiru bukan isinya melainkan BENTUKNYA — termasuk empat keadaan yang paling
// mudah luput saat menggambar layar:
//
//	berkas yang sudah dikirim ke layanan Arsip     (CABANGSTATUS '1', ada tanggal kirim)
//	berkas yang belum dikirim                      (CABANGSTATUS '0', tanggal kirim kosong)
//	berkas lini PA dan Travel                      (menguji saringan jabatan)
//	berkas yang kode tipe dokumennya tidak ada di master (nama kosong, LEFT JOIN)

// SampleFiles adalah isi awal grid ARCHIVE FILE KLAIM.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Lini Personal Accident — ia HILANG dari daftar kirim ke cabang bagi
// petugas berjabatan PA maupun NONMBU. Lihat archivedokumenklaim.BranchScope.
// Lini Travel — hilang bagi petugas berjabatan TRAVEL maupun NONMBU.
// Kode tipe dokumennya TIDAK ADA di master, sehingga kedua nama dokumennya
// kosong. Baris seperti ini nyata di data warisan, dan layar harus tetap
// menggambarnya alih-alih menyembunyikannya.
func SampleFiles() []archivedokumenklaim.ArchiveFile {
	return sampledata.Must[[]archivedokumenklaim.ArchiveFile](sampleJSON, "SampleFiles")
}

// SampleClaims adalah calon klaim pada bagian Input Data Archive.
//
// Catatan pada isinya, yang kini tersimpan di sample.json:
//
// Klaim yang BELUM tutup. Sistem lama tidak melarang mengarsipkan berkasnya,
// dan larangan itu tidak ditambahkan di sini.
func SampleClaims() []archivedokumenklaim.ClaimCandidate {
	return sampledata.Must[[]archivedokumenklaim.ClaimCandidate](sampleJSON, "SampleClaims")
}

// SampleDocumentTypes adalah pilihan Tipe Dokumen.
func SampleDocumentTypes() []archivedokumenklaim.DocumentTypeOption {
	return sampledata.Must[[]archivedokumenklaim.DocumentTypeOption](sampleJSON, "SampleDocumentTypes")
}

// SampleDocumentKinds adalah pilihan Jenis Dokumen beserta tipe pemiliknya.
func SampleDocumentKinds() []archivedokumenklaim.DocumentKindOption {
	return sampledata.Must[[]archivedokumenklaim.DocumentKindOption](sampleJSON, "SampleDocumentKinds")
}

// NewSampleRepo membentuk repo memori berisi seluruh data contoh.
func NewSampleRepo() *Repo {
	return NewRepo(Options{
		Files:  SampleFiles(),
		Claims: SampleClaims(),
		Types:  SampleDocumentTypes(),
		Kinds:  SampleDocumentKinds(),
	})
}

// day menyusun satu tanggal kalender UTC.
func day(year int, month time.Month, dayOfMonth int) *time.Time {
	moment := time.Date(year, month, dayOfMonth, 0, 0, 0, 0, time.UTC)
	return &moment
}
