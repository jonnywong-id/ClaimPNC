// Package dokumenpenunjang menangani unggah dan daftar dokumen penunjang sebuah KLAIM.
//
// # Apa yang digantikannya di Pega
//
// Kedua layar Open Protection punya tombol unggah yang di Pega adalah flow action
// `SetUploadDocPUCL` (kelas `ASM-FW-GCNMFW-Work-OpenProtection`), dipasang dua kali di
// `Section/InputProtectionSection-Section.xml:8634`, `:8777` dan dua kali di
// `Section/AcceptProtectionSection-Section.xml:8812`, `:8906`.
//
// Rantainya:
//
//	SetUploadDocPUCL  →  pre  SetPreAttachmentPNC       Page-New Data-WorkAttach-File
//	                  →  sec  SetUploadDoc_Detl         → ASMAttachContentScreen
//	                  →  post SaveAttachmentOPPNC       → GCNMSaveAttachments
//	                                                    → SetCategoryAttachment
//
// Jadi di Pega **kedua layar itu memakai mesin lampiran Pega sendiri**
// (`Data-WorkAttach-File`, `SaveAllAttachments`), bukan layanan penyimpanan internal.
// Berkasnya mendarat di tabel engine `PC_LINK_ATTACHMENT` / `PC_DATA_WORKATTACH`.
//
// # Kenapa sistem baru TIDAK menirunya
//
// `D-16` sudah memutuskan sebaliknya, dan keputusan itu mendahului kemiripan per layar:
//
//	"Dokumen disimpan lewat API storage internal Sinarmas yang sudah berjalan. Database
//	 aplikasi hanya menyimpan metadata dan referensi. Tiga mekanisme penyimpanan yang ada
//	 sekarang disatukan menjadi satu jalur."
//
// Meniru mesin lampiran Pega berarti membangun mesin lampiran keempat justru ketika
// keputusannya adalah menyatukan yang tiga. Jalur yang dipakai karena itu adalah jalur yang
// Pega pakai di tempat LAIN untuk hal yang sama — `Activity/InsertDokumenPNC-Act.xml`, yang
// mengunggah ke layanan penyimpanan internal lalu mencatat metadatanya.
//
// Konsekuensinya dinyatakan di muka: pada uji kesetaraan, dokumen yang diunggah dari kedua
// layar ini **tidak akan muncul di tabel lampiran Pega**. Itu selisih yang direncanakan,
// bukan cacat.
//
// # Dokumennya menempel pada KLAIM, bukan pada proteksi
//
// Work Owner menetapkan *"Ke klaim, seperti Pega"*. Itu juga yang dituntut bentuk datanya:
// `T_CLAIM_OPENPROTECTION` tidak punya satu pun kolom dokumen, sedangkan tabel metadata
// penyimpanan punya `NO_CLAIM` yang **NOT NULL**.
//
// # Yang BELUM dikerjakan, dan itu dinyatakan bukan disembunyikan
//
// `InsertDokumenPNC` memanggil `Convert_Avif` lebih dulu untuk PNG dan JPG, sehingga gambar
// tersimpan sebagai AVIF. Konversi itu Connect REST tersendiri (`KonversiAvif`) dan BELUM
// dibawa. Akibatnya gambar tersimpan pada format aslinya — lebih besar, tetapi tidak salah.
package dokumenpenunjang

import (
	"fmt"
	"strings"
	"time"
)

// NamaAplikasi adalah kunci pencarian folder penyimpanan.
//
// Pega menyetelnya sebagai konstanta `"KLAIMPNC"` di
// `Activity/InsertDokumenPNC-Act.xml:2836`, lalu memakainya mencari nama folder:
//
//	select NAMA_FOLDER AS CARI1 FROM general.T_FOLDER_STORAGE WHERE APLIKASI = 'KLAIMPNC'
//
// Ia BUKAN nama folder itu sendiri — yang dikirim ke layanan penyimpanan adalah hasil
// pencariannya. Membedakan keduanya penting: menyamakannya akan mengirim `KLAIMPNC` sebagai
// nama aplikasi penyimpanan, dan berkasnya mendarat di tempat yang salah.
const NamaAplikasi = "KLAIMPNC"

// JenisPenyimpanan mengisi kolom `STORAGE`, yang Pega tulis sebagai literal `'standard'`
// pada `RDB List/InsertDataPNCStorage-SQL.xml`.
const JenisPenyimpanan = "standard"

// TanpaKlaim adalah nilai yang dipakai ketika nomor klaim kosong.
//
// Kolom `NO_CLAIM` **NOT NULL** (diperiksa langsung ke katalog 2026-09-26), dan Pega
// menambalnya di `Activity/InsertDokumenPNC-Act.xml:3829`:
//
//	DocAPI.NoClaim := @If(DocAPI.NoClaim=="", "-", DocAPI.NoClaim)
//
// Ditiru apa adanya. Tanpa ini, unggahan tanpa klaim gagal dengan galat constraint yang
// tidak memberi tahu sebabnya.
const TanpaKlaim = "-"

// BatasUkuranBerkas membatasi satu unggahan.
//
// Pega tidak menyatakan batas apa pun — berkasnya dikirim sebagai base64 di dalam satu
// muatan JSON. Batas ini karena itu **kemampuan baru**, bukan peniruan, dan alasannya
// operasional: base64 membengkakkan isi sekitar sepertiga, dan muatan JSON dirakit di
// memori. Tanpa batas, satu berkas besar dapat menghabiskan memori satu instans.
//
// Angkanya belum ditetapkan Work Owner; 20 MiB dipilih sebagai nilai awal yang lapang untuk
// foto dan dokumen pindaian, dan ia **parameter**, bukan aturan bisnis.
const BatasUkuranBerkas = 20 << 20

// Document adalah satu dokumen yang sudah tersimpan.
type Document struct {
	// ImageID adalah kunci yang DITERBITKAN layanan penyimpanan, bukan oleh kita.
	ImageID string

	// FileName adalah nama berkas sebagaimana tersimpan — sudah dibersihkan, lihat
	// BersihkanNamaBerkas. Ia BUKAN nama yang diketik pengguna.
	FileName string

	// URL adalah alamat publik berkasnya, dan ia BERMASA BERLAKU.
	URL       string
	ExpiresAt *time.Time

	// Folder adalah folder penyimpanan yang dipakai, dikembalikan layanan.
	Folder string

	// ClaimNumber adalah klaim yang dokumen ini menempel padanya.
	ClaimNumber string

	// DocumentType mengisi `TYPEIMAGE` — jenis dokumen menurut master, bila dipilih.
	DocumentType string

	UploadedAt *time.Time
}

// Kedaluwarsa menyatakan URL-nya sudah lewat masa berlaku pada waktu tertentu.
//
// Dipisahkan sebagai method, bukan dihitung di layar, karena dua layar memakainya dan
// keduanya harus sepakat. URL tanpa masa berlaku diperlakukan sebagai MASIH berlaku:
// 128.379 baris produksi memuat sebagian tanpa `EXPDATE`, dan memperlakukannya kedaluwarsa
// akan menyembunyikan dokumen yang sebenarnya dapat dibuka.
func (d Document) Kedaluwarsa(saat time.Time) bool {
	if d.ExpiresAt == nil {
		return false
	}
	return saat.After(*d.ExpiresAt)
}

// UploadRequest adalah satu permintaan unggah.
type UploadRequest struct {
	// ClaimNumber boleh kosong; ia menjadi TanpaKlaim saat disimpan.
	ClaimNumber string

	// FileName adalah nama asli yang diketik pengguna, LENGKAP dengan ekstensinya —
	// ekstensi itulah yang menentukan MimeType.
	FileName string

	// DocumentType mengisi `TYPEIMAGE`; boleh kosong.
	DocumentType string

	// Content adalah isi berkas apa adanya. Penyandian base64 dikerjakan adapter, bukan
	// pemanggil: ia detail protokol layanan penyimpanan, bukan bagian dari permintaan.
	Content []byte

	// By adalah login pengunggah, mengisi `UserInput`.
	By string
}

// BersihkanNamaBerkas membuang setiap karakter selain huruf dan angka.
//
// Ini tiruan tepat dari `Activity/InsertDokumenPNC-Act.xml:1597`:
//
//	@pxReplaceAllViaRegex(param.Filename, "[^a-zA-Z0-9]", "")
//
// # Ya, ini ikut membuang titik ekstensinya — dan itu memang perilaku Pega
//
//	"Foto Kerugian.pdf"  ->  "FotoKerugianpdf"
//
// Terlihat seperti cacat, dan mungkin memang cacat; tetapi jenis berkasnya TIDAK hilang —
// ia dikirim terpisah sebagai `MimeType`, dan layanan penyimpanan yang menentukan nama
// akhir berkasnya.
//
// `P-5` menetapkan perilaku dipertahankan lebih dulu, dan ini bukan cacat yang menghitung
// uang (`R-19`), sehingga ia TIDAK masuk daftar perbaikan `D-49`. Bila Work Owner
// menghendaki nama asli dipertahankan, perubahannya satu fungsi ini saja — dan itu sebabnya
// ia berdiri sendiri, bukan tersebar di adapter.
func BersihkanNamaBerkas(nama string) string {
	var b strings.Builder
	b.Grow(len(nama))
	for _, r := range nama {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// EkstensiDari mengambil ekstensi berkas tanpa titik, dalam huruf kecil.
//
// Dipisahkan dari TipeMedia karena BersihkanNamaBerkas menghapus titiknya: ekstensi harus
// diambil dari nama ASLI, sebelum pembersihan. Urutan itu mudah terbalik, dan terbaliknya
// tidak menghasilkan galat — hanya `MimeType` yang selalu jatuh ke nilai bawaan.
func EkstensiDari(nama string) string {
	titik := strings.LastIndex(nama, ".")
	if titik < 0 || titik == len(nama)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(nama[titik+1:]))
}

// TipeMedia memetakan ekstensi berkas ke tipe media.
//
// Peta ini disalin apa adanya dari `Activity/InsertDokumenPNC-Act.xml:3744`, sebuah rantai
// `@If` lima belas cabang. Urutannya tidak penting, isinya penting — menambah atau
// mengurangi satu baris mengubah apa yang diterima layanan penyimpanan.
//
// Cabang terakhirnya pun ditiru: ekstensi yang tidak dikenal menjadi `"application/<ext>"`,
// bukan `application/octet-stream`. Itu menghasilkan tipe media yang sering tidak sah
// (`application/xyz`), tetapi layanan penyimpanan sudah menerimanya selama ini, dan
// menggantinya adalah perubahan perilaku yang tidak diminta.
func TipeMedia(ekstensi string) string {
	switch strings.ToUpper(strings.TrimSpace(ekstensi)) {
	case "PNG":
		return "image/png"
	case "JPG", "JPEG":
		return "image/jpeg"
	case "AVIF":
		return "image/avif"
	case "TXT":
		return "text/plain"
	case "DOC":
		return "application/msword"
	case "DOCX":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "PDF":
		return "application/pdf"
	case "EML":
		return "message/rfc822"
	case "RAR":
		return "application/vnd.rar"
	case "ZIP":
		return "application/zip"
	case "CSV":
		return "text/csv"
	case "XLS":
		return "application/vnd.ms-excel"
	case "XLSX":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "PPT":
		return "application/vnd.ms-powerpoint"
	case "PPTX":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	default:
		return "application/" + strings.TrimSpace(ekstensi)
	}
}

// FolderTanggal menyusun folder tujuan dari waktu unggah.
//
// Pega merakitnya di `Activity/InsertDokumenPNC-Act.xml:3652`:
//
//	"Doc/" + @substring(@CurrentDateTime(),0,4) + "/" + @substring(@CurrentDateTime(),4,6) + "/"
//
// `@CurrentDateTime()` berbentuk `YYYYMMDDThhmmss.mmm GMT`, sehingga potongan 0–4 adalah
// tahun dan 4–6 adalah bulan. Hasilnya `Doc/2026/09/`.
//
// # Zona waktunya sengaja WIB, dan itu SELISIH dari Pega
//
// Pega memakai GMT, sehingga unggahan antara pukul 00:00 dan 07:00 WIB pada tanggal 1 jatuh
// ke folder BULAN SEBELUMNYA. Itu hanya soal penempatan berkas, bukan nilai bisnis, tetapi
// `F-5` menetapkan seluruh tanggal yang dilihat manusia dihitung terhadap tanggal WIB — dan
// folder ini dilihat manusia yang mencari berkas.
//
// Selisihnya kecil dan sengaja; dicatat supaya tidak terbaca sebagai kelalaian.
func FolderTanggal(saat time.Time) string {
	return fmt.Sprintf("Doc/%04d/%02d/", saat.Year(), int(saat.Month()))
}

// EkstensiHasilKonversi adalah ekstensi yang menggantikan ekstensi gambar setelah dikonversi.
//
// Nilainya `"Avif"` apa adanya, bukan `"image/avif"`: Pega menyetel
// `Param.MimeType := "Avif"` (`Activity/InsertDokumenPNC-Act.xml:2399`), dan barulah
// kemudian peta tipe media mengubahnya menjadi `image/avif`. Urutan itu ditiru, sebab
// melompatinya akan melewatkan satu-satunya tempat peta itu diterapkan.
const EkstensiHasilKonversi = "Avif"

// PerluKonversi menyatakan sebuah berkas dikirim ke layanan konversi lebih dulu.
//
// Empat ekstensi, dan PDF salah satunya — meski hasilnya tetap PDF. Prakondisinya disalin
// apa adanya dari `Activity/Convert_Avif-Act.xml:672` dan
// `Activity/InsertDokumenPNC-Act.xml:2177`:
//
//	@toUpperCase(param.MimeType)=="PNG" || =="JPG" || =="JPEG" || =="PDF"
//
// Untuk PDF, layanan itu memampatkan berkasnya; tipenya TIDAK berubah. Yang berubah tipe
// hanyalah gambar — lihat EkstensiSetelahKonversi.
func PerluKonversi(ekstensi string) bool {
	switch strings.ToUpper(strings.TrimSpace(ekstensi)) {
	case "PNG", "JPG", "JPEG", "PDF":
		return true
	default:
		return false
	}
}

// EkstensiSetelahKonversi menyatakan ekstensi yang berlaku sesudah konversi berhasil.
//
// # PDF tetap PDF, dan itu bukan kelalaian
//
// Pega menyetel `Param.MimeType := "Avif"` HANYA pada cabang PNG/JPG/JPEG
// (`Activity/InsertDokumenPNC-Act.xml:2322`). Cabang PDF (`:2401`) mengganti isinya saja
// dan membiarkan tipenya. Menyeragamkannya akan menandai PDF sebagai `image/avif` — dan
// peramban menolak membukanya.
func EkstensiSetelahKonversi(ekstensi string) string {
	switch strings.ToUpper(strings.TrimSpace(ekstensi)) {
	case "PNG", "JPG", "JPEG":
		return EkstensiHasilKonversi
	default:
		return ekstensi
	}
}

// NomorKlaimUntukPenyimpanan menambal nomor klaim yang kosong.
func NomorKlaimUntukPenyimpanan(nomor string) string {
	if rapi := strings.TrimSpace(nomor); rapi != "" {
		return rapi
	}
	return TanpaKlaim
}
