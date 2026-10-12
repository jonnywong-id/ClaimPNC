package masterbengkel

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Dokumen lampiran sebuah bengkel — unggah dan lihat.
//
// # Rantai aslinya di Pega, dibaca utuh
//
//	Section/BrowseMasterHE            tombol "Upload Document"
//	  -> Flow Action UploadDocument            @baseclass 01-01-89
//	       -> Section UploadDocument           satu pemilih berkas, label "Nama File"
//	       -> Activity SaveFilePenunjang       Call GCNMUploadResult64, 2x Property-Set
//	            -> GCNMUploadResult64          Java: pega_rules_utilities.uploadFile
//	                                           menaruh FileBase64, FileName, FileExt, FileSize
//
//	Section/ApprovalMasterBengkelHE   tombol lihat dokumen
//	  -> Flow Action ViewDocumentMasterBengkel @baseclass 01-01-89
//	       -> Section ViewAttachment           @baseclass 01-01-93
//	       -> Activity GetDetailDocument       GetIDDokumenBengkel -> GetAttachmentFromDB_Sql
//
// Penyimpanannya menempuh `PNCSaveAttachmentToDB` -> `SaveAttachmentToDB_Sql` ->
// `POOLDATA.SET_ATTACHMENT_64BIT`, yang menerbitkan `DATAID` lalu menaruhnya di
// `BENGKEL_HE.DOKUMENID`.
//
// # Satu lubang yang Pega tinggalkan, dan ke mana isinya dialihkan
//
// Di sistem lama isi berkasnya TIDAK pernah tersimpan, dan itu terbaca utuh dari rantainya:
//
//   - `Activity/PNCSaveAttachmentToDB` MENYETEL isinya di halaman
//     (`InsertAttachment.ATTACHFILE = Param.attachfile`),
//   - tetapi `RDB List/SaveAttachmentToDB_Sql` memanggil prosedurnya dengan sembilan
//     masukan — dan `ATTACHFILE` **bukan salah satunya**,
//   - `Database/SET_ATTACHMENT_64BIT.prc:30-31` pun menyisipkan sepuluh kolom tanpa
//     `ATTACHFILE`,
//   - sehingga `GetAttachmentFromDB_Sql`, yang membacanya lewat `base64encode(attachfile)`,
//     selalu menemukan kosong.
//
// Berkasnya dibaca dari peramban, dibawa melintasi tiga rule, lalu jatuh di pemanggilan
// prosedurnya.
//
// Yang TIDAK dilakukan di sini adalah menambal kolom `ATTACHFILE`. Isi berkas dikirim ke
// **layanan penyimpanan internal** (`D-16`) lewat jalur yang sudah terpasang di aplikasi ini
// — modul `dokumenpenunjang`, tiruan rantai `InsertDokumenPNC` Pega — dan yang disimpan di
// barisnya hanyalah `IMAGEID`-nya. Itu persis kolom yang `SET_ATTACHMENT_64BIT` memang
// sediakan dan memang isi (`tIMAGEID`), sehingga bentuk barisnya tidak bergeser sedikit pun
// dari Pega.
//
// Jalur yang sama dipakai Master Panel, Master Sparepart, Open Protection, dan registrasi,
// lewat adapter `repo/dokumenlink` masing-masing. Membangunnya ulang di sini berarti dua
// tiruan aturan Pega yang sama, dan keduanya akan menyimpang pada perbaikan pertama yang
// hanya diterapkan di salah satunya.
//
// Ini **selisih terencana**, bukan perbaikan diam-diam: di uji kesetaraan, dokumen yang
// diunggah sistem baru dapat dibuka, sedangkan dokumen Pega tidak.
//
// # Yang TIDAK dibawa
//
// `INPUTOPERATOR` tetap diisi, tetapi `CATEGORY` dan `SUB_CATEGORY` dibiarkan kosong —
// `Section/UploadDocument` tidak punya isian untuk keduanya, sehingga mengisinya berarti
// mengarang. `IDPEGA` juga kosong: ia kunci teknis Pega (`pyWorkPage.pzInsKey`) yang `D-22`
// larang dibawa.

// Batas ukuran dan jenis berkas.
//
// # Keduanya KEPUTUSAN BARU, bukan salinan
//
// Sistem lama tidak punya satu pun pemeriksaan: `GCNMUploadResult64` menghitung ukuran dalam
// KB lalu menaruhnya di parameter, dan angka itu tidak pernah dibandingkan dengan apa pun.
// Tidak ada daftar jenis berkas yang diizinkan.
//
// Membiarkannya tanpa batas berarti membuka lubang baru di sistem baru, sehingga batasnya
// ditetapkan di sini — dan dinyatakan sebagai selisih, bukan disembunyikan. Angkanya milik
// modul ini, bukan aturan bisnis, dan menaikkannya tidak menyentuh satu pun aturan klaim.
const (
	// MaxDocumentBytes membatasi satu berkas pada 10 MiB.
	MaxDocumentBytes = 10 << 20

	// MaxDocumentNameLength mengikuti panjang terpakai `ATTACHNAME`; belum dibaca dari
	// katalog (`R-08`) sehingga ia asumsi, dan ditandai begitu.
	MaxDocumentNameLength = 200
)

// AllowedDocumentExtension adalah jenis berkas yang diterima.
//
// Daftarnya ditetapkan modul ini — lihat MaxDocumentBytes. Isinya mengikuti apa yang wajar
// dilampirkan pada berkas bengkel: bukti kerja sama, NPWP, rekening, dan foto bengkel.
var AllowedDocumentExtension = map[string]string{
	".pdf":  "application/pdf",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".csv":  "text/csv",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
}

// Kesalahan dokumen.
var (
	// ErrDocumentEmpty berarti berkasnya nol byte. Pega menerimanya tanpa keberatan; di
	// sini ia ditolak, sebab berkas kosong hanya menghasilkan lampiran yang menyesatkan.
	ErrDocumentEmpty = errors.New("masterbengkel: berkas kosong")

	// ErrDocumentTooLarge berarti melewati MaxDocumentBytes.
	ErrDocumentTooLarge = errors.New("masterbengkel: berkas melebihi batas ukuran")

	// ErrDocumentTypeNotAllowed berarti jenisnya di luar AllowedDocumentExtension.
	ErrDocumentTypeNotAllowed = errors.New("masterbengkel: jenis berkas tidak diizinkan")

	// ErrDocumentNotFound berarti `DOKUMENID`-nya kosong, atau barisnya tidak ada di
	// `POOLDATA.DATA_ATTACHFILE`.
	ErrDocumentNotFound = errors.New("masterbengkel: dokumen tidak ditemukan")
)

// Document adalah satu baris `POOLDATA.DATA_ATTACHFILE`.
//
// Nama fieldnya bahasa Inggris (`D-80`); nama kolomnya tetap seperti di basis data, dan
// pemetaannya ada di adapter SQL.
type Document struct {
	// ID adalah `DATAID` — kunci yang diterbitkan saat unggah, dan yang disimpan di
	// `BENGKEL_HE.DOKUMENID`.
	ID string

	// ImageID adalah kunci berkasnya di layanan penyimpanan, mengisi kolom `IMAGEID`.
	//
	// Tanpa ini barisnya hanya metadata yang menunjuk ke ketiadaan — dan itulah keadaan
	// SELURUH dokumen warisan Pega, yang jalur unggahnya tidak pernah mengisinya.
	ImageID string

	Name     string
	Note     string
	MimeType string

	UploadedBy string
	UploadedAt time.Time
}

// HasFile membedakan dokumen yang berkasnya benar-benar ada dari dokumen warisan yang hanya
// metadata.
//
// Layar memerlukannya: menawarkan tombol buka untuk baris tanpa `IMAGEID` hanya
// menghasilkan kegagalan, dan pengguna tidak punya cara tahu sebabnya.
func (d Document) HasFile() bool { return strings.TrimSpace(d.ImageID) != "" }

// UploadInput adalah satu permintaan unggah, apa adanya dari transport.
type UploadInput struct {
	// WorkshopID adalah `ID_BENGKEL` yang dilampiri.
	WorkshopID string

	// FileName adalah nama berkas asli dari peramban.
	FileName string

	// Content adalah isi berkasnya.
	Content []byte
}

// Clean merapikan masukan tanpa menilai kesahihannya.
func (i UploadInput) Clean() UploadInput {
	i.WorkshopID = strings.TrimSpace(i.WorkshopID)
	i.FileName = strings.TrimSpace(i.FileName)
	return i
}

// Check memeriksa masukan, dan mengembalikan SELURUH pelanggaran sekaligus.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: layar lama menampilkan
// seluruh pesan bersamaan.
func (i UploadInput) Check() error {
	var violation []Violation

	switch {
	case i.FileName == "":
		violation = append(violation, Violation{
			Field:   "nama_berkas",
			Message: "Nama berkas wajib ada.",
		})
	case len([]rune(i.FileName)) > MaxDocumentNameLength:
		violation = append(violation, Violation{
			Field: "nama_berkas",
			Message: fmt.Sprintf("Nama berkas paling panjang %d karakter.",
				MaxDocumentNameLength),
		})
	default:
		if _, ok := AllowedDocumentExtension[DocumentExtension(i.FileName)]; !ok {
			violation = append(violation, Violation{
				Field:   "nama_berkas",
				Message: "Jenis berkas tidak diizinkan. Yang diterima: " + allowedExtensionList() + ".",
			})
		}
	}

	switch {
	case len(i.Content) == 0:
		violation = append(violation, Violation{
			Field:   "berkas",
			Message: "Berkas kosong.",
		})
	case len(i.Content) > MaxDocumentBytes:
		violation = append(violation, Violation{
			Field: "berkas",
			Message: fmt.Sprintf("Ukuran berkas melebihi batas %d MB.",
				MaxDocumentBytes>>20),
		})
	}

	if len(violation) == 0 {
		return nil
	}
	return &ValidationError{Violation: violation}
}

// DocumentExtension mengembalikan akhiran berkas dalam huruf kecil, termasuk titiknya.
func DocumentExtension(fileName string) string {
	return strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
}

// DocumentMimeType memetakan nama berkas ke tipe media.
//
// Ia TIDAK memercayai tipe media yang dikirim peramban: nilai itu berasal dari klien dan
// dapat disetel apa saja. Yang dipakai adalah akhiran berkas yang sudah lolos daftar izin.
func DocumentMimeType(fileName string) string {
	return AllowedDocumentExtension[DocumentExtension(fileName)]
}

func allowedExtensionList() string {
	list := make([]string, 0, len(AllowedDocumentExtension))
	for ext := range AllowedDocumentExtension {
		list = append(list, strings.TrimPrefix(ext, "."))
	}
	// Urutannya dibuat tetap supaya pesan galatnya tidak berubah-ubah antarpermintaan —
	// map Go tidak menjamin urutan, dan pesan yang berubah membuat ujinya rapuh.
	sort.Strings(list)
	return strings.Join(list, ", ")
}

// ComposeDocumentID menyusun `DATAID` persis seperti `SET_ATTACHMENT_64BIT`.
//
// Bentuknya, dibaca dari `Database/SET_ATTACHMENT_64BIT.prc:18-26`:
//
//	count_ATTACH := ATTACHFILE_SEQ.nextval
//	INSERT INTO C_COUNTER_ATTACHMENT (KEY, YEAR, RUNNO) VALUES (pkey, to_char(sysdate,'yy'), count_ATTACH)
//	select year || lpad(runno,10,'0') into tDATAID from C_COUNTER_ATTACHMENT where key = pkey
//
// Jadi `DATAID` = dua digit tahun + nomor urut yang diratakan nol sampai sepuluh digit.
// Tahunnya diambil dari jam basis data (`sysdate`), sehingga dokumen yang diunggah di
// sekitar pergantian tahun mengikuti jam server — bertaut dengan `R-12`, sama seperti
// nomor klaim pada `D-71`.
func ComposeDocumentID(year string, sequence int64) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := documentSequenceWidth - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(year) + number
}

// documentSequenceWidth adalah lebar `lpad(runno,10,'0')` pada prosedur aslinya.
const documentSequenceWidth = 10

// Clock adalah seam waktu (`F-5`).
//
// Ia dibutuhkan karena DATAID memuat dua digit TAHUN. Prosedur lama mengambilnya dari
// `sysdate` — jam basis data — sedangkan modul ini melarang SYSDATE dan TO_CHAR di dalam
// SQL, sehingga tahunnya disusun di Go. Tanpa seam, ujinya hanya dapat menegaskan tahun
// yang sedang berjalan, dan akan gagal sendiri pada 1 Januari.
type Clock interface {
	Now() time.Time
}

// DocumentIDSource menerbitkan `DATAID` berikutnya.
//
// Ia seam tersendiri, terpisah dari IDSource milik nomor bengkel: keduanya memakai sequence
// yang berbeda (`ATTACHFILE_SEQ` versus `BENGKEL_HE_SEQ`) dan dipanggil pada saat yang
// berbeda. Menyatukannya akan membuat satu kegagalan menjatuhkan keduanya.
type DocumentIDSource interface {
	NextDocumentID(ctx context.Context) (string, error)
}

// DocumentRepo adalah seam ke penyimpan dokumen.
//
// Ia dideklarasikan di sini — di paket yang MEMAKAINYA — bukan di adapter, sehingga bentuk
// yang dibutuhkan domain menjadi yang menentukan.
type DocumentRepo interface {
	// SaveDocument mencatat satu baris lampiran, lalu menautkannya ke bengkel.
	//
	// Keduanya satu transaksi: baris lampiran tanpa tautan adalah dokumen yatim yang tidak
	// dapat ditemukan siapa pun, dan tautan tanpa baris lampiran adalah `DOKUMENID` yang
	// menunjuk ke ketiadaan. `D-68` menempatkan kepemilikan transaksi di Go.
	SaveDocument(ctx context.Context, workshopID string, document Document) error

	// FindDocument mengembalikan satu dokumen menurut `DATAID`.
	FindDocument(ctx context.Context, documentID string) (Document, error)
}

// UploadFailure menggolongkan kegagalan unggah menurut APA YANG BOLEH DILAKUKAN PENGGUNA.
//
// Golongannya sengaja SAMA PERSIS dengan Master Sparepart dan Master Panel: pengguna yang
// menemui kegagalan yang sama di dua layar tidak seharusnya membaca dua kalimat berbeda.
//
// "Unggah gagal" tanpa penggolongan membuat pengguna menebak apakah ia boleh mengulang, dan
// pada unggahan menebak salah berakibat nyata — mengulang unggahan yang separuh berhasil
// menghasilkan berkas ganda di layanan penyimpanan.
type UploadFailure int

const (
	// UploadInvalid: isi permintaannya salah — perbaiki lalu ulangi.
	UploadInvalid UploadFailure = iota + 1
	// UploadTooLarge: berkas melampaui batas ukuran.
	UploadTooLarge
	// UploadUnavailable: layanan hulu gagal dan belum ada yang tersimpan — aman diulang.
	UploadUnavailable
	// UploadHalfDone: berkas sudah terkirim tetapi catatannya gagal — JANGAN diulang.
	UploadHalfDone
	// UploadMisconfigured: salah konfigurasi di sisi aplikasi.
	UploadMisconfigured
)

// DocumentUploadError adalah galat unggah beserta golongan dan pesan siap tampil.
type DocumentUploadError struct {
	Kind    UploadFailure
	Message string
	Err     error
}

func (e *DocumentUploadError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// Unwrap membuka galat aslinya supaya errors.Is tetap bekerja menembus terjemahan.
func (e *DocumentUploadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// DocumentFile adalah satu berkas yang hendak diunggah ke layanan penyimpanan.
type DocumentFile struct {
	// Portal menentukan basis data entitas tujuannya (`D-75`).
	Portal string

	// FileName adalah nama asli yang dipilih pengguna, LENGKAP dengan ekstensinya —
	// ekstensi itulah yang menentukan tipe medianya di hilir.
	FileName string

	// Content adalah isi berkas apa adanya. Penyandian base64 urusan adapter penyimpanan,
	// bukan urusan modul ini.
	Content []byte

	// By adalah login pengunggah.
	By string
}

// DocumentUploader adalah seam ke layanan penyimpanan internal (`D-16`).
//
// Ia mengembalikan IMAGEID saja, karena hanya itu yang disimpan modul ini. URL dan masa
// berlakunya dimiliki modul dokumen penunjang dan dibaca dari sana saat dibutuhkan —
// menyalinnya ke sini akan membuat dua tempat menyimpan masa berlaku yang sama, dan yang
// satu pasti basi lebih dulu.
type DocumentUploader interface {
	Upload(ctx context.Context, f DocumentFile) (imageID string, err error)
}
