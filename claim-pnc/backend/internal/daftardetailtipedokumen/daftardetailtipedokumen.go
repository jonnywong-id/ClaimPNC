// Package daftardetailtipedokumen adalah inti modul Daftar Detail Tipe Dokumen
// (`F-4`, MENU_ID 41, MENU_PROGRAM `ListDetTypeDocument`).
//
// # Apa yang dimodelkan di sini
//
// Satu baris adalah satu **rincian dokumen klaim**: dokumen apa persisnya yang diminta
// di bawah sebuah tipe dokumen, pada objek apa ia melekat, penyebab kerugian mana yang
// membuatnya diminta, dan — lewat daftar bisnisnya — pada lini bisnis mana ia wajib
// beserta jumlah unggahan minimumnya.
//
// Di sistem lama layarnya `Harness/ListDetTypeDocument-Harness.xml`, dan datanya dibaca
// dari DUA view:
//
//	V_LST_DET_TYPE_DOC          induk — ID · DOC_TYPE_ID · DETAIL_DOCUMENT · STS_INSURED
//	                            DOC_COL_ID · OBJ_DOC · RISK · TGL_EDIT · USER_EDIT
//	                            ditambah TYPE_DOCUMENT · DOC_COL_INFO · OBJ_DOC_DESC
//	V_LST_DET_TYPE_DOC_BISNIS   anak  — ID · DFT_BISNIS_ID · STS_WAJIB · MIN_DOC
//
// Yang pertama mengisi grid layar (`Report Definition/BrowseVLstDetTypeDoc_RD-RD.xml`).
// Yang kedua mengisi grid berulang di dalam formnya, dibaca
// `RDB List/GetLbuDetType-SQL.xml` dengan menjoinnya ke POOLDATA.BUSINESS untuk
// mendapatkan nama bisnisnya.
//
// # Batas modul: ini master TURUNAN, dan ada dua saudara yang mudah tertukar
//
//	LST_DOC_TYPE            <- internal/daftartipedokumen       MENU_ID 40
//	  ├─ V_LST_DET_TYPE_DOC       <- paket ini                  MENU_ID 41
//	  └─ LST_TYPE_DOC_BUSINESS    <- internal/daftartipedokumenbisnis  MENU_ID 42
//
// Yang di atas hanya menyimpan ID dan nama tipe dokumen. Paket ini merujuk ID itu dan
// menambahkan rinciannya. Yang ketiga TIDAK berada di bawah paket ini meski namanya
// mirip: ia tabel yang sama sekali berbeda (`LST_TYPE_DOC_BUSINESS`, relasional penuh,
// punya procedure sendiri) dan justru MEMBACA baris milik paket ini lewat
// `DOC_TYPE_DT_ID`.
//
// Akibat yang mengikat: ID yang sudah terbit TIDAK PERNAH BERUBAH dan TIDAK PERNAH
// DIHAPUS. Selain modul MENU_ID 42, view induknya dibaca **34 rule Pega** — di antaranya
// seluruh jalur validasi unggah dokumen (`Activity/ValidationUploadDocument_act-Act.xml`,
// `Activity/ValidationUploadRegister-Act.xml`, `Activity/RequiredDocument_act-Act.xml`),
// arsip dokumen, dan pencarian klaim.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/ListDetTypeDocument-Harness.xml             layar "Detail Tipe Dokumen"
//	Section/DetTypeDocument-Section.xml                  bingkai layar, tombol Tambah/Refresh
//	Section/BrowseListDetailTypeDocument-Section.xml     grid, form, grid bisnis, Simpan/Ubah
//	Report Definition/BrowseVLstDetTypeDoc_RD-RD.xml     kolom view induk
//	RDB List/GetLbuDetType-SQL.xml                       kolom view anak dan join-nya
//	Activity/CNMSetDetailTypeDocument_act-Act.xml        memuat satu baris ke form
//	Activity/CNMInsertDetailTypeDocument_act-Act.xml     urutan langkah simpan
//	RDB List/UpdateDetTypeDoc-SQL.xml                    pemanggilan procedure penyimpan
//	Database/PEGA_LST_DET_TYPE_DOC.prc                   isi procedure itu
//
// # Yang BERUBAH dari sistem lama: penyimpanan tidak lagi lewat JSON
//
// `Database/PEGA_LST_DET_TYPE_DOC.prc` menyisipkan tepat dua kolom —
// `LST_DET_TYPE_DOC(ID, JSON_DATA)` — dan seluruh isian disimpan sebagai satu dokumen
// JSON hasil `stepPage.getJSON(false)` (`Function/GetPageJSONString-Function.xml`).
// Kedua view membongkar dokumen itu kembali menjadi kolom.
//
// Keputusan Work Owner 2026-09-23: **tabelnya sudah memiliki kolom selain ID dan
// JSON_DATA, dan aplikasi membaca serta menulis kolom itu — bukan JSON.** Arahnya sama
// dengan yang sudah ditempuh Master Penyebab Kerugian (migrasi 0005) dan Daftar Objek
// Dokumen (migrasi 0008).
//
// `D-68` menambahkan alasan yang lebih keras untuk tidak memanggil procedure-nya:
// parameter keluarannya bernama `ErrMsg` tetapi pada jalur BERHASIL ia berisi kalimat
// "Data Sudah Disimpan dengan ID : 1011" (`PEGA_LST_DET_TYPE_DOC.prc:23`), sehingga
// pemanggil tidak dapat membedakan berhasil dari gagal tanpa membaca teks.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package daftardetailtipedokumen

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// DetailType adalah satu rincian dokumen klaim.
//
// Nama fieldnya berbahasa Inggris (`D-80`) dan mengikuti arti kolomnya, bukan singkatan
// kolomnya. Pemetaan nama kolom ke field ada di repo/sqlstore — satu tempat saja.
type DetailType struct {
	// ID adalah kolom ID. Diterbitkan penyimpanan dan TIDAK PERNAH berubah — layar Pega
	// pun menandainya `pyEditOptions=Read-only`
	// (`Section/BrowseListDetailTypeDocument-Section.xml`).
	ID string

	// DocumentTypeID adalah DOC_TYPE_ID, rujukan ke POOLDATA.V_LST_DOC_TYPE milik modul
	// Daftar Tipe Dokumen (MENU_ID 40).
	//
	// Di layar ia autocomplete yang menyimpan `.ID` dan menampilkan `.TYPE_DOCUMENT`.
	// Di sini ia tetap TEKS BEBAS: layar lama tidak memeriksanya sebelum menyimpan, dan
	// `P-5` menetapkan perilaku dipertahankan lebih dulu.
	DocumentTypeID string

	// DocumentTypeName adalah TYPE_DOCUMENT — nama tipe dokumen menurut masternya.
	//
	// HANYA DIBACA, tidak pernah ditulis: ia datang dari V_LST_DOC_TYPE lewat view
	// induk, bukan dari baris ini. Isinya kosong bila DOC_TYPE_ID menunjuk tipe yang
	// sudah tidak ada — dan baris seperti itu tetap harus tampil, bukan disembunyikan.
	//
	// # Kenapa HANYA field ini yang hasil join, sementara dua keterangan lain tersimpan
	//
	// Karena ia SATU-SATUNYA kolom view yang tidak pernah ada di halaman `TempDTDoc`.
	// `Activity/CNMSetDetailTypeDocument_act-Act.xml` mengisi 12 properti — ID,
	// DOC_TYPE_ID, DETAIL_DOCUMENT, STS_INSURED, DOC_COL_ID, DOC_COL_INFO, OBJ_DOC,
	// OBJ_DOC_DESC, RISK, TGL_EDIT, USER_EDIT, DFT_BISNIS_ID — dan `TYPE_DOCUMENT` bukan
	// salah satunya. Halaman itulah yang diserialisasi menjadi dokumen JSON, sehingga apa
	// yang tidak ada di sana tidak mungkin tersimpan.
	DocumentTypeName string

	// Detail adalah DETAIL_DOCUMENT — label layar "Detail Dokumen".
	//
	// Inilah nama yang dibaca petugas, dan inilah yang ditampilkan modul MENU_ID 42
	// sebagai pilihan isian Detail Dokumen.
	Detail string

	// InsuredStatus adalah STS_INSURED — label layar "Status Tertanggung".
	//
	// TEKS BEBAS. Di form Pega ia `pyEditOptions=Auto` tanpa satu pun daftar pilihan,
	// tanpa `pyRequired`, dan tanpa `pyMaxLength` — sama persis dengan STS_PROSES pada
	// modul Daftar Tipe Dokumen, yang namanya juga menyiratkan status padahal bukan.
	InsuredStatus string

	// CauseOfLossID adalah DOC_COL_ID, rujukan ke POOLDATA.M_CAUSE_OF_LOSS.M_COL_ID
	// milik modul Master Penyebab Kerugian.
	//
	// Di layar ia autocomplete atas `ASM-FW-GCNMFW-Int-V_M_CAUSE_OF_LOSS` yang
	// menyimpan `.M_COL_ID` (`pyPropertyTarget = TempDTDoc.DOC_COL_ID`) dan menampilkan
	// `.COL_DESC`.
	CauseOfLossID string

	// CauseOfLossDescription adalah DOC_COL_INFO — keterangan penyebab kerugian.
	//
	// **TERSIMPAN, bukan hasil join.** Inilah isian yang benar-benar dilihat dan diketik
	// petugas: `Section/BrowseListDetailTypeDocument-Section.xml` mengikat isian berlabel
	// "Dokumen kolom ID" ke `TempDTDoc.DOC_COL_INFO`, sementara `DOC_COL_ID` di sampingnya
	// hanya target tersembunyi yang diisi autocomplete (`pyPropertyTarget`).
	//
	// Karena `pyAllowFreeFormInput=true`, keterangan yang TIDAK ADA di master tetap boleh
	// diketik dan tersimpan — dan pada keadaan itu kodenya kosong. Menyusunnya kembali
	// lewat join akan membuang tepat isian itu.
	CauseOfLossDescription string

	// ObjectDocumentID adalah OBJ_DOC, rujukan ke POOLDATA.V_LST_DOC_OBJ milik modul
	// Daftar Objek Dokumen (MENU_ID 43).
	//
	// Di layar ia autocomplete yang menyimpan `.ID` (`pyPropertyTarget =
	// TempDTDoc.OBJ_DOC`) dan menampilkan `.KET_DOC_OBJ`.
	ObjectDocumentID string

	// ObjectDocumentDescription adalah OBJ_DOC_DESC — keterangan objek dokumen.
	//
	// **TERSIMPAN, bukan hasil join**, dengan alasan yang sama seperti
	// CauseOfLossDescription di atas: isian berlabel "Objek Dokumen" terikat ke
	// `TempDTDoc.OBJ_DOC_DESC`, dan `OBJ_DOC` hanya target tersembunyinya.
	ObjectDocumentDescription string

	// Risk adalah RISK — label layar "Resiko".
	//
	// Disimpan sebagai TEKS, bukan angka, karena itulah bentuk kolomnya di view. Bahwa
	// isinya dibaca sebagai bilangan terbukti dari pembacanya:
	// `Activity/SetTypePDFAdjustment-Act.xml` menguji `@toDecimal(.RISK)<=0`.
	//
	// Kosong diterima, dan itu bukan kelalaian: `toDecimal("")` di Pega menghasilkan nol,
	// sehingga baris tanpa resiko berperilaku sama dengan baris beresiko nol. Menolaknya
	// akan mengubah perilaku layar yang tidak sedang dimigrasikan.
	Risk string

	// Businesses adalah aturan per lini bisnis — isi V_LST_DET_TYPE_DOC_BISNIS untuk
	// baris ini.
	//
	// KOSONG berarti rincian dokumen ini belum dikaitkan ke lini bisnis mana pun, dan itu
	// keadaan yang sah — grid utama layar Pega pun hanya membaca view induknya, sehingga
	// baris tanpa bisnis tetap tampil utuh.
	//
	// Senarai ini hanya terisi pada pembacaan SATU baris (Get), tidak pada daftar.
	// Daftar tidak membutuhkannya: gridnya hanya tiga kolom — ID, Tipe Dokumen, dan
	// Detail Dokumen — dan menariknya untuk seluruh baris berarti satu kueri yang
	// hasilnya tidak pernah dilihat siapa pun.
	Businesses []BusinessRule
}

// BusinessRule adalah aturan dokumen pada satu lini bisnis.
//
// Bentuknya mengikuti grid berulang `TempDTDoc.DFT_BISNIS_ID` pada
// `Section/BrowseListDetailTypeDocument-Section.xml`: tiga isian — ID Bisnis, Status
// Wajib, dan Minimum Dokumen.
type BusinessRule struct {
	// BusinessID adalah DFT_BISNIS_ID, rujukan ke POOLDATA.BUSINESS.ID milik GISFW.
	BusinessID string

	// BusinessName adalah NOTE pada POOLDATA.BUSINESS.
	//
	// HANYA DIBACA — `RDB List/GetLbuDetType-SQL.xml` mendapatkannya dengan menjoin
	// BUSINESS, bukan dari baris ini. Akibatnya bila bisnisnya sudah tidak ada di master,
	// namanya kosong sementara ID-nya tetap tersimpan.
	BusinessName string

	// Mandatory adalah STS_WAJIB — label layar "Status Wajib".
	//
	// Boolean di sini; yang tersimpan adalah TEKS. Lihat MandatoryYes di bawah — kolom
	// ini memuat empat nilai yang berbeda di produksi, dan pembacaannya karena itu
	// sengaja longgar sementara penulisannya tegas.
	Mandatory bool

	// MinDocument adalah MIN_DOC — label layar "Minimum Dokumen".
	//
	// Nol berarti tidak ada tuntutan jumlah.
	MinDocument int
}

// Nilai yang DITULIS ke kolom STS_WAJIB.
//
// # Kenapa teks "Ya"/"Tidak", bukan angka 1/0
//
// Keputusan Work Owner 2026-09-23, dan bukti export membenarkannya. Kolom ini dibaca
// tiga activity, dan yang membandingkannya HANYA dengan teks adalah yang membaca view
// milik modul ini:
//
//	Activity/SetTypePDFAdjustment-Act.xml     .STS_WAJIB=="Ya"          hanya teks
//	Activity/ValidationUploadDocument_act     "Ya" "Tidak" "1" "0"      keempatnya
//	Activity/ValidationUploadRegister         "Ya" "Tidak" "1" "0"      keempatnya
//
// Menulis "1" akan membuat `SetTypePDFAdjustment` berhenti mengenali dokumen wajib —
// tanpa satu pun galat, hanya jenis PDF yang salah pilih.
//
// # Kenapa pembacaannya tetap menerima "1" dan "0"
//
// Karena kedua activity yang menerima keempatnya membuktikan data produksi memang
// memuat keduanya. Menolak nilai yang tidak dikenal akan menggagalkan seluruh daftar
// karena satu baris warisan — jauh lebih buruk daripada membacanya sebagai "Tidak".
const (
	MandatoryYes = "Ya"
	MandatoryNo  = "Tidak"
)

// MandatoryText mengubah penanda wajib menjadi teks yang disimpan.
func MandatoryText(mandatory bool) string {
	if mandatory {
		return MandatoryYes
	}
	return MandatoryNo
}

// MandatoryFrom membaca kembali kolom STS_WAJIB menjadi penanda wajib.
//
// Empat nilai dikenali sebagai "wajib", dan sisanya — termasuk kosong dan NULL —
// dijawab "tidak wajib". Alasannya ada di komentar MandatoryYes.
//
// Perbandingannya mengabaikan besar-kecil huruf dan spasi di ujung: kolom bertipe CHAR
// berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan baris
// warisan dapat memuat "YA" maupun "ya".
func MandatoryFrom(stored string) bool {
	switch strings.ToLower(strings.TrimSpace(stored)) {
	case "ya", "1", "y", "true":
		return true
	default:
		return false
	}
}

// Input adalah nilai yang dikirim pengguna dari layar.
//
// Terpisah dari DetailType karena ID tidak pernah berasal dari pengguna: pada penambahan
// ia diterbitkan penyimpanan, pada penyuntingan ia diambil dari jalur URL.
//
// # Dua keterangan ikut dikirim, satu tidak — dan pembedaannya bukan selera
//
// `CauseOfLossDescription` dan `ObjectDocumentDescription` ADA di sini karena keduanya
// isian yang benar-benar diketik petugas dan benar-benar tersimpan. `DocumentTypeName`
// TIDAK ADA karena ia satu-satunya yang hasil join — lihat penjelasan pada field itu.
type Input struct {
	DocumentTypeID string
	Detail         string
	InsuredStatus  string

	// CauseOfLossID dan CauseOfLossDescription dikirim BERPASANGAN.
	//
	// Keterangannya yang diketik petugas; kodenya diisi autocomplete saat sebuah pilihan
	// dipilih. Keterangan di luar master tetap sah — dan pada keadaan itu kodenya kosong,
	// persis seperti `pyAllowFreeFormInput=true` di layar lama. Mengirim kodenya saja akan
	// membuang isian yang sah menjadi baris kosong.
	CauseOfLossID          string
	CauseOfLossDescription string

	// ObjectDocumentID dan ObjectDocumentDescription dikirim BERPASANGAN, dengan alasan
	// yang sama.
	ObjectDocumentID          string
	ObjectDocumentDescription string

	Risk       string
	Businesses []BusinessInput
}

// BusinessInput adalah satu baris grid bisnis yang dikirim layar.
//
// Tanpa BusinessName: namanya milik master dan dibaca lewat join, tidak disimpan di
// baris ini. Berbeda dari modul Daftar Detail Dokumen Travel, yang memang menyimpan nama
// plan dan jaminannya sendiri karena view-nya memaparkan kolom itu.
type BusinessInput struct {
	BusinessID  string
	Mandatory   bool
	MinDocument int
}

// Batas panjang isian.
//
// # Ini BUKAN aturan bisnis, melainkan bentuk kolom
//
// Lebar kolom yang sebenarnya BELUM DIKETAHUI — DDL tabelnya tidak ada di export
// (`R-08`), dan layar Pega tidak memuat satu pun `pyMaxLength` pada isian mana pun.
// Angka di bawah dipilih supaya isian yang jelas keliru — teks sepanjang paragraf yang
// tertempel tanpa sengaja — ditolak dengan pesan yang dapat dibaca petugas, bukan dengan
// ORA-12899 yang tidak dapat dibacanya.
//
// Bila DBA kelak mengabarkan lebar yang sebenarnya, angka di sinilah yang disesuaikan —
// dan hanya di sini.
const (
	MaxDetailLength        = 200
	MaxInsuredStatusLength = 100
	MaxRiskLength          = 50
	MaxReferenceLength     = 50

	// MaxDescriptionLength membatasi kedua keterangan yang TERSIMPAN — DOC_COL_INFO dan
	// OBJ_DOC_DESC.
	//
	// Angkanya disamakan dengan MaxDetailLength karena ketiganya sama sifatnya: teks yang
	// diketik petugas dan dibaca petugas lain, bukan kode.
	MaxDescriptionLength = 200
)

// Nama isian pada laporan pelanggaran, dipakai layar untuk menempatkan pesannya.
const (
	FieldDetail         = "detail_dokumen"
	FieldInsuredStatus  = "status_tertanggung"
	FieldRisk           = "resiko"
	FieldDocumentType   = "id_tipe_dokumen"
	FieldCauseOfLoss    = "id_penyebab_kerugian"
	FieldObjectDocument = "id_objek_dokumen"
	FieldBusiness       = "bisnis"
)

// Violation adalah satu pelanggaran isian.
type Violation struct {
	Field   string
	Message string
}

// ValidationError mengumpulkan SELURUH pelanggaran sekaligus.
//
// Dikumpulkan, bukan dilaporkan satu per satu, meniru perilaku Pega yang menampilkan
// seluruh pesan bersamaan (`11-CROSSCUTTING` §1.2). Pada form dengan tujuh isian,
// mengembalikan satu pelanggaran per percobaan akan sangat menyiksa petugas.
type ValidationError struct {
	Violation []Violation
}

func (e *ValidationError) Error() string {
	if len(e.Violation) == 0 {
		return "daftardetailtipedokumen: isian tidak sah"
	}
	parts := make([]string, 0, len(e.Violation))
	for _, item := range e.Violation {
		parts = append(parts, item.Message)
	}
	return "daftardetailtipedokumen: " + strings.Join(parts, "; ")
}

// Clean memangkas spasi di kedua ujung setiap isian teks dan membuang baris bisnis yang
// seluruhnya kosong.
//
// # Pemangkasannya bukan kerapian
//
// Kolomnya dibaca kembali dengan pemangkasan, karena kolom CHAR berlebar tetap
// memadatkan nilainya dengan spasi tanpa memberi tanda apa pun. Tanpa memangkas saat
// menulis, apa yang disimpan dan apa yang dibaca kembali dapat berbeda — dan selisih itu
// tidak terlihat di layar karena spasi tidak tampak.
//
// # Baris bisnis kosong dibuang, dan itu bukan validasi
//
// Grid di layar selalu menyisakan baris yang baru ditambahkan tetapi belum diisi.
// Menyimpannya berarti menulis aturan yang tidak menunjuk bisnis mana pun, lalu
// membacanya kembali sebagai baris hantu di form berikutnya.
func (i Input) Clean() Input {
	clean := Input{
		DocumentTypeID:            strings.TrimSpace(i.DocumentTypeID),
		Detail:                    strings.TrimSpace(i.Detail),
		InsuredStatus:             strings.TrimSpace(i.InsuredStatus),
		CauseOfLossID:             strings.TrimSpace(i.CauseOfLossID),
		CauseOfLossDescription:    strings.TrimSpace(i.CauseOfLossDescription),
		ObjectDocumentID:          strings.TrimSpace(i.ObjectDocumentID),
		ObjectDocumentDescription: strings.TrimSpace(i.ObjectDocumentDescription),
		Risk:                      strings.TrimSpace(i.Risk),
	}

	clean.Businesses = make([]BusinessInput, 0, len(i.Businesses))
	for _, row := range i.Businesses {
		row.BusinessID = strings.TrimSpace(row.BusinessID)

		// MIN_DOC negatif tidak punya arti apa pun — "paling sedikit minus satu berkas"
		// bukan aturan yang dapat dipenuhi maupun dilanggar. Ia diratakan menjadi nol,
		// bukan ditolak, supaya perlakuannya tetap sejalan dengan layar tanpa validasi.
		if row.MinDocument < 0 {
			row.MinDocument = 0
		}

		// Baris dibuang hanya bila bisnisnya TIDAK dipilih. Status wajib dan jumlah
		// minimum tidak ikut diperiksa: keduanya punya nilai baku yang sah — "Tidak" dan
		// nol — sehingga baris yang bisnisnya terisi selalu bermakna.
		if row.BusinessID == "" {
			continue
		}
		clean.Businesses = append(clean.Businesses, row)
	}
	return clean
}

// Check menjalankan seluruh aturan isian yang dapat diperiksa TANPA menyentuh
// penyimpanan, dan mengembalikan semua pelanggarannya.
//
// Nil berarti isian sah sejauh yang dapat diperiksa di sini. Input sudah harus melewati
// Clean lebih dulu.
//
// # Kenapa nyaris tidak ada yang diperiksa
//
// Karena layar lama pun tidak memeriksa apa pun. Pemeriksaan langsung ke export:
//
//	Section/BrowseListDetailTypeDocument-Section.xml   pyRequired=false pada SELURUH
//	                                                   isiannya — tanpa satu pun
//	                                                   pengecualian
//	(export)                                           nol Rule-Obj-Validate untuk kelas
//	                                                   ASM-FW-GCNMFW-Int-V_LST_DET_TYPE_DOC
//	Database/PEGA_LST_DET_TYPE_DOC.prc                 menyisipkan tanpa memeriksa apa pun
//
// `P-5` menetapkan perilaku dipertahankan lebih dulu, dan Work Owner sudah menetapkan hal
// yang sama untuk empat modul sejenis. Akibat yang sengaja diterima: rincian tanpa detail
// dokumen dapat tersimpan, dan baris itu akan muncul sebagai pilihan KOSONG pada layar
// MENU_ID 42 yang merujuknya.
//
// Yang diperiksa hanyalah PANJANG, dan itu bentuk kolom — bukan aturan bisnis. Lihat
// MaxDetailLength.
//
// # Yang sengaja TIDAK diperiksa di sini
//
// Keberadaan DOC_TYPE_ID, DOC_COL_ID, OBJ_DOC, dan DFT_BISNIS_ID di masternya
// masing-masing. Memeriksanya menuntut membaca empat master lain, sedangkan fungsi ini
// murni dan dapat diuji tanpa apa pun. Layar lama pun tidak memeriksanya: keempat
// isiannya autocomplete yang tetap menerima ketikan di luar daftar.
func (i Input) Check() error {
	var violation []Violation

	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	tooLong := func(value string, limit int, field, label string) {
		if utf8.RuneCountInString(value) > limit {
			violation = append(violation, Violation{
				Field:   field,
				Message: label + " paling panjang " + itoa(limit) + " karakter.",
			})
		}
	}

	tooLong(i.Detail, MaxDetailLength, FieldDetail, "Detail Dokumen")
	tooLong(i.InsuredStatus, MaxInsuredStatusLength, FieldInsuredStatus, "Status Tertanggung")
	tooLong(i.Risk, MaxRiskLength, FieldRisk, "Resiko")
	tooLong(i.DocumentTypeID, MaxReferenceLength, FieldDocumentType, "ID Tipe Dokumen")

	// Yang dibatasi pada kedua rujukan berketerangan adalah KETERANGANNYA, karena itulah
	// yang diketik petugas. Kodenya ikut dibatasi dengan batas kode, bukan batas
	// keterangan — keduanya kolom yang berbeda dan lebarnya tidak sama.
	tooLong(i.CauseOfLossDescription, MaxDescriptionLength, FieldCauseOfLoss, "Dokumen kolom ID")
	tooLong(i.CauseOfLossID, MaxReferenceLength, FieldCauseOfLoss, "Kode Dokumen kolom ID")
	tooLong(i.ObjectDocumentDescription, MaxDescriptionLength, FieldObjectDocument, "Objek Dokumen")
	tooLong(i.ObjectDocumentID, MaxReferenceLength, FieldObjectDocument, "Kode Objek Dokumen")

	for _, row := range i.Businesses {
		if utf8.RuneCountInString(row.BusinessID) > MaxReferenceLength {
			violation = append(violation, Violation{
				Field:   FieldBusiness,
				Message: "ID Bisnis paling panjang " + itoa(MaxReferenceLength) + " karakter: " + row.BusinessID,
			})
			break
		}
	}

	// BISNIS KEMBAR TIDAK DITOLAK, mengikuti grid Pega yang tidak punya satu pun penanda
	// keunikan — tidak ada `pyUnique`, tidak ada validasi. Perlakuan yang sama sudah
	// ditetapkan Work Owner untuk grid sebentuk pada Master COL Simas Online dan Daftar
	// Objek Dokumen.

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// itoa tanpa mengimpor fmt, supaya lapisan domain tetap seringan mungkin.
func itoa(n int) string { return strconv.Itoa(n) }

// Editor adalah jejak siapa yang menyimpan dan kapan.
//
// Ia BUKAN isian pengguna dan tidak pernah datang dari badan permintaan: sistem lama pun
// mengisinya sendiri saat menyimpan — `Activity/CNMInsertDetailTypeDocument_act-Act.xml`
// menetapkan `TempDTDoc.USER_EDIT` dan `TempDTDoc.TGL_EDIT` pada langkah pertamanya,
// sebelum halamannya diserialisasi.
//
// Nilainya dibawa sampai ke repo, bukan dibentuk di sana, karena repo tidak tahu siapa
// yang sedang masuk dan tidak boleh tahu — identitas datang dari sesi, dan sesi adalah
// urusan lapisan transport.
type Editor struct {
	// Identity mengisi USER_EDIT. Sama dengan `OperatorID.pyUserIdentifier` di Pega.
	Identity string

	// At mengisi TGL_EDIT.
	//
	// Waktunya dibawa dari luar, bukan diambil `time.Now()` di dalam repo, supaya seluruh
	// modul membaca jam lewat satu seam yang sama (`F-5`) dan supaya penyimpanan dapat
	// diuji secara deterministik.
	At time.Time
}

// ErrNotFound: baris yang diminta tidak ada di master.
var ErrNotFound = errors.New("daftardetailtipedokumen: detail tipe dokumen tidak ditemukan")

// SequenceDigits adalah lebar nomor urut pada ID.
//
// Angkanya dibaca langsung dari `Database/PEGA_LST_DET_TYPE_DOC.prc:19`:
//
//	id_detype_ins := id_site || lpad(to_Char(LST_DET_TYPE_DOC_SEQ.nextval),4,'0');
//
// EMPAT — kebetulan sama dengan modul Daftar Tipe Dokumen, tetapi TIDAK boleh disalin
// dari sana: Master Status Klaim memakai tiga dan Master Dokumen Travel memakai lima.
// Ketiganya memakai pola yang sama dengan lebar yang berbeda-beda, dan menyalin lebar
// dari modul tetangga akan menghasilkan ID yang tidak dikenali data historis.
const SequenceDigits = 4

// FormatID menyusun ID dari kode situs dan nomor urut, meniru procedure lama persis.
//
// # Kenapa bentuknya direplikasi, bukan diperbaiki
//
// ID ini dirujuk `LST_TYPE_DOC_BUSINESS.DOC_TYPE_DT_ID` milik MENU_ID 42 dan dibaca 34
// rule Pega lain. Mengganti bentuknya akan memutus baris baru dari data historis. `P-5`
// berlaku — bentuk ini tidak ada di daftar 13 perbaikan eksplisit `D-49`.
//
// # Batas yang nyata, dan cacat yang ikut terbawa
//
// Nomor di atas 9999 dikembalikan APA ADANYA, tanpa dipotong, sama seperti LPAD Oracle.
// ID ke-10000 karena itu menjadi satu karakter lebih panjang.
//
// Dipotong menjadi empat digit? Tidak. Itu menghasilkan ID GANDA, yang jauh lebih buruk
// daripada penyisipan yang gagal dengan pesan jelas — dan ID ganda di sini berarti dua
// rincian dokumen berbeda berbagi satu kunci yang dirujuk modul MENU_ID 42.
func FormatID(site string, sequence int64) string {
	digits := strconv.FormatInt(sequence, 10)
	for len(digits) < SequenceDigits {
		digits = "0" + digits
	}
	return strings.TrimSpace(site) + digits
}

// Business adalah satu pilihan pada isian ID Bisnis.
//
// Ia sengaja BUKAN tipe milik modul lain: modul tidak saling mengimpor tipenya, sehingga
// perubahan di satu modul tidak merambat ke modul lain. Yang dibagi adalah tabelnya,
// bukan kodenya.
type Business struct {
	ID   string
	Name string
}

// DocumentTypeOption adalah satu pilihan pada isian ID Tipe Dokumen.
type DocumentTypeOption struct {
	ID   string
	Name string
}

// CauseOfLossOption adalah satu pilihan pada isian Dokumen kolom ID.
type CauseOfLossOption struct {
	ID          string
	Description string
}

// ObjectDocumentOption adalah satu pilihan pada isian Objek Dokumen.
type ObjectDocumentOption struct {
	ID          string
	Description string
}

// Repo adalah seam ke penyimpanan detail tipe dokumen SATU portal.
//
// Pengisinya ada di repo/sqlstore (Oracle) dan repo/memory (pengujian dan pengembangan
// tanpa basis data). Satu instans Repo selalu terikat pada satu basis data entitas —
// pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat kueri (`ADR-0030`
// Opsi 1). Yang memilih instans mana yang melayani satu permintaan adalah RepoSelector.
//
// Tidak ada Delete, dan itu bukan kelalaian: layar Pega tidak punya tombol hapus, dan
// `Database/PEGA_LST_DET_TYPE_DOC.prc` hanya mengenal INSERT dan UPDATE. Menghapus satu
// baris akan membuat setiap aturan MENU_ID 42 yang menyimpan ID itu lewat DOC_TYPE_DT_ID
// kehilangan artinya — persis alasan `ADR-0012` menetapkan master tidak dihapus permanen,
// dan `D-66` melarang penghapusan fisik data bernilai bisnis.
type Repo interface {
	// List mengembalikan seluruh rincian TANPA daftar bisnisnya, terurut seperti grid
	// lama.
	List(ctx context.Context) ([]DetailType, error)

	// Get mengembalikan satu rincian LENGKAP dengan daftar bisnisnya; ErrNotFound bila
	// ID-nya tidak ada.
	Get(ctx context.Context, id string) (DetailType, error)

	// InsertNew menerbitkan ID lalu menyisipkan barisnya beserta seluruh baris bisnisnya,
	// dan mengembalikan baris yang benar-benar tersimpan.
	//
	// Penerbitan ID berada DI DALAM satu operasi repo, bukan dipecah menjadi "ambil
	// nomor" lalu "sisip" di lapisan aplikasi: memecahnya melebarkan jarak antara
	// mengambil nomor urut dan memakainya, dan memaksa lapisan aplikasi mengetahui
	// bentuk kunci yang seharusnya hanya diketahui penyimpanan.
	InsertNew(ctx context.Context, input Input, by Editor) (DetailType, error)

	// Update mengganti isi satu rincian beserta SELURUH daftar bisnisnya; ErrNotFound
	// bila barisnya hilang di antara pemuatan layar dan penyimpanan.
	//
	// Daftar bisnis diganti seluruhnya, bukan ditambal baris demi baris. Alasannya ada di
	// layar: grid di form memang mengirim susunan akhir yang dikehendaki petugas, dan
	// tidak ada satu pun penanda di sana yang menyatakan baris mana yang baru, mana yang
	// berubah, dan mana yang dibuang. Penggantian menyeluruh adalah satu-satunya
	// tafsiran yang tidak menebak.
	Update(ctx context.Context, id string, input Input, by Editor) (DetailType, error)
}

// ReferenceRepo adalah seam BACA-SAJA ke keempat master yang dirujuk layar ini.
//
// Ketiadaan operasi tulis di sini disengaja: keempat tabelnya dimiliki modul lain (`P-1`
// — satu tabel satu penulis), dan batas itu ditegakkan oleh bentuk antarmuka, bukan oleh
// ingatan orang yang menulis kode berikutnya.
//
//	POOLDATA.V_LST_DOC_TYPE       modul Daftar Tipe Dokumen     MENU_ID 40
//	POOLDATA.M_CAUSE_OF_LOSS      modul Master Penyebab Kerugian MENU_ID 20
//	POOLDATA.V_LST_DOC_OBJ        modul Daftar Objek Dokumen    MENU_ID 43
//	POOLDATA.BUSINESS             GISFW                          (`D-03`)
//
// # Kenapa keempatnya di SATU seam
//
// Karena keempatnya dibutuhkan bersamaan oleh SATU form, dan keempatnya gagal dengan cara
// yang sama — daftar pilihannya kosong, sementara isiannya tetap dapat diketik sendiri
// dan penyimpanannya tetap berjalan. Memisahkannya menjadi empat seam akan menghasilkan
// empat selector yang selalu dipilih bersamaan dan empat jalur galat yang ditangani
// dengan cara yang persis sama.
//
// Ini berbeda dari modul Daftar Detail Dokumen Travel, yang memisahkan seam-nya karena
// di sana memang ada perbedaan nyata: satu tabel milik master induk, satu milik GISFW,
// dan keduanya dipakai pada tahap yang berbeda.
type ReferenceRepo interface {
	// ListDocumentTypes mengembalikan pilihan isian ID Tipe Dokumen.
	ListDocumentTypes(ctx context.Context) ([]DocumentTypeOption, error)

	// ListCausesOfLoss mengembalikan pilihan isian Dokumen kolom ID.
	ListCausesOfLoss(ctx context.Context) ([]CauseOfLossOption, error)

	// ListObjectDocuments mengembalikan pilihan isian Objek Dokumen.
	ListObjectDocuments(ctx context.Context) ([]ObjectDocumentOption, error)

	// ListBusinesses mengembalikan pilihan isian ID Bisnis pada grid.
	ListBusinesses(ctx context.Context) ([]Business, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca saat
// permintaan datang — bukan diputuskan sekali ketika aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
//
// # Kenapa modul ini per portal
//
// Dibuktikan dari Pega, bukan diandaikan: `Database/PEGA_LST_DET_TYPE_DOC.prc:11`
// membentuk ID dengan `SELECT ID FROM M_SITE_DATABASE WHERE CURRENT_SITE = '1'` — kode
// situs yang melekat pada basis data tempat procedure berjalan. Mekanismenya identik
// dengan `PEGA_LST_DOC_TYPE.prc:12` dan `PEGA_M_CAUSE_OF_LOSS.prc:11`, dua master yang
// sudah diputuskan per entitas.
type RepoSelector func(portalAlias string) (Repo, error)

// ReferenceRepoSelector memilih ReferenceRepo milik satu portal entitas.
type ReferenceRepoSelector func(portalAlias string) (ReferenceRepo, error)
