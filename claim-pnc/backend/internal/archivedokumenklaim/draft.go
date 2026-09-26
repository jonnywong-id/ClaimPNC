package archivedokumenklaim

import (
	"strings"
	"time"
)

// Batas panjang isian teks.
//
// Tidak satu pun berasal dari DDL: `T_CLAIM_ARCHIVE_FILE` TIDAK punya berkas DDL di
// export, dan tidak ada satu pun rule yang menyebutkan panjang kolomnya (`R-08`).
// Angkanya karena itu dipasang sebagai pengaman yang longgar — cukup untuk menahan
// tempelan tidak sengaja sepanjang ribuan karakter, tidak cukup ketat untuk menolak isian
// yang sah. Ia diperketat begitu DDL-nya diterima.
const (
	MaxBoxNameLength     = 100
	MaxFillingCodeLength = 100
	MaxCodeLength        = 50
	MaxClaimNumberLength = 50
	MaxNameLength        = 200
)

// MaxSheetCount adalah batas atas Jumlah Lembar.
//
// Sistem lama tidak membatasinya sama sekali, dan kolomnya NUMBER. Batas ini menahan
// salah ketik yang terlihat wajar — "1000000" alih-alih "100" — bukan menahan berkas
// tebal yang sungguhan.
const MaxSheetCount = 100000

// Draft adalah satu formulir Input Data Archive yang sudah sah.
//
// Ia hanya dibentuk lewat NewDraft, sehingga repo yang menerimanya tidak perlu memeriksa
// ulang isinya.
//
// # Mana yang diketik pengguna dan mana yang ikut dari klaim
//
// Enam isian pertama TIDAK diketik: `Activity/SetDataArchiveDokumentCase-Act.xml`
// menyalinnya dari baris klaim yang dipilih pada grid Input Data Archive. Yang diketik
// pengguna hanyalah enam isian terakhir.
type Draft struct {
	// ID adalah ID_ARCHIVE yang hendak diubah. Nol berarti baris baru.
	//
	// Pembedaan ini menggantikan parameter `flags` sistem lama yang bernilai "insert"
	// atau "update" — sebuah penanda terpisah yang dapat bertentangan dengan ID yang
	// menyertainya. Di sini keduanya tidak dapat bertentangan karena hanya ada satu.
	ID int64

	// Keenam isian berikut ikut dari klaim yang dipilih.
	ClaimNumber  string
	PolicyNumber string
	InsuredName  string
	LossDate     *time.Time
	TechnicalPIC string
	GroupPanel   string

	// Keenam isian berikut diketik pengguna pada formulir.
	DocumentReceivedDate *time.Time
	SheetCount           int
	DocumentTypeCode     string
	DocumentKindCode     string
	BoxName              string
	FillingCode          string

	// InputUser adalah pengisi USERINPUT. Ia datang dari identitas pemanggil, bukan dari
	// formulir — isian yang dapat diketik pengguna akan membuat jejak siapa mengarsipkan
	// apa menjadi tidak berarti.
	//
	// Ia HANYA dipakai saat menyisipkan. Prosedur lama tidak mengubahnya saat baris
	// diperbarui, dan perilaku itu dipertahankan: kolom ini menyatakan siapa yang
	// MENGARSIPKAN, bukan siapa yang terakhir menyunting.
	InputUser string

	// BranchCode mengisi kolom KODECABANG.
	//
	// Sistem lama mengisinya dari `TempCabang.KodeCabang`, dan tidak ada satu pun rule
	// di export yang MEMBACA kolom ini kembali. Di sini ia diisi kode cabang pemanggil,
	// yang merupakan pembacaan paling lurus atas nama propertinya. Ia ditandai sebagai
	// simpulan, bukan bacaan, di docs/permintaan-artefak-pega.md.
	BranchCode string
}

// IsNew menyatakan draft ini menyisipkan baris baru, bukan mengubah yang sudah ada.
func (d Draft) IsNew() bool { return d.ID == 0 }

// DraftInput adalah isian mentah dari lapisan transport, sebelum divalidasi.
type DraftInput struct {
	ID int64

	ClaimNumber  string
	PolicyNumber string
	InsuredName  string
	LossDate     *time.Time
	TechnicalPIC string
	GroupPanel   string

	DocumentReceivedDate *time.Time
	SheetCount           int
	DocumentTypeCode     string
	DocumentKindCode     string
	BoxName              string
	FillingCode          string
}

// NewDraft memvalidasi formulir arsip dan membentuk Draft.
//
// Seluruh pelanggaran dikumpulkan sekaligus (`11-CROSSCUTTING.md` §1.2). Pada formulir
// enam isian bedanya kecil, tetapi aturannya berlaku seragam di seluruh aplikasi — dan
// pengecualian yang dibuat "karena formulirnya pendek" adalah pengecualian yang kemudian
// ditiru di formulir panjang.
//
// # Aturan yang TIDAK ada di sistem lama
//
// Sistem lama tidak memeriksa satu isian pun sebelum memanggil prosedurnya. Akibatnya
// nyata: menekan Save To Archive dengan formulir kosong menghasilkan baris arsip tanpa
// nomor klaim, tanpa boks, dan tanpa kode filling — baris yang tidak dapat ditemukan lagi
// lewat pencarian mana pun, dan tidak dapat dihapus karena layar itu tidak punya tombol
// hapus.
//
// Pemeriksaan di sini karena itu dipasang sebagai PENGAMAN, bukan perubahan aturan
// bisnis, dan dicatat di docs/keputusan-implementasi.md supaya tidak terbaca sebagai
// perbaikan diam-diam.
func NewDraft(input DraftInput, caller Caller, branchCode string) (Draft, error) {
	clean := caller.Clean()

	var violations []Violation

	claimNumber := strings.TrimSpace(input.ClaimNumber)
	if claimNumber == "" {
		violations = append(violations, Violation{
			Field:   FieldClaimNumber,
			Message: "Pilih klaim yang berkasnya diarsipkan lebih dulu.",
		})
	} else if len(claimNumber) > MaxClaimNumberLength {
		violations = append(violations, Violation{
			Field:   FieldClaimNumber,
			Message: "No Klaim terlalu panjang.",
		})
	}

	if input.SheetCount <= 0 {
		violations = append(violations, Violation{
			Field:   FieldSheetCount,
			Message: "Isi Jumlah Lembar dengan angka lebih besar dari nol.",
		})
	} else if input.SheetCount > MaxSheetCount {
		violations = append(violations, Violation{
			Field:   FieldSheetCount,
			Message: "Jumlah Lembar tidak wajar. Periksa kembali angkanya.",
		})
	}

	documentType := strings.TrimSpace(input.DocumentTypeCode)
	if documentType == "" {
		violations = append(violations, Violation{
			Field:   FieldDocumentType,
			Message: "Pilih Tipe Dokumen.",
		})
	} else if len(documentType) > MaxCodeLength {
		violations = append(violations, Violation{
			Field:   FieldDocumentType,
			Message: "Kode Tipe Dokumen terlalu panjang.",
		})
	}

	documentKind := strings.TrimSpace(input.DocumentKindCode)
	if documentKind == "" {
		violations = append(violations, Violation{
			Field:   FieldDocumentKind,
			Message: "Pilih Jenis Dokumen.",
		})
	} else if len(documentKind) > MaxCodeLength {
		violations = append(violations, Violation{
			Field:   FieldDocumentKind,
			Message: "Kode Jenis Dokumen terlalu panjang.",
		})
	}

	boxName := strings.TrimSpace(input.BoxName)
	if boxName == "" {
		violations = append(violations, Violation{
			Field:   FieldBoxName,
			Message: "Isi Nama BOX tempat berkas disimpan.",
		})
	} else if len(boxName) > MaxBoxNameLength {
		violations = append(violations, Violation{
			Field:   FieldBoxName,
			Message: "Nama BOX terlalu panjang.",
		})
	}

	fillingCode := strings.TrimSpace(input.FillingCode)
	if fillingCode == "" {
		violations = append(violations, Violation{
			Field:   FieldFillingCode,
			Message: "Isi Kode Filling.",
		})
	} else if len(fillingCode) > MaxFillingCodeLength {
		violations = append(violations, Violation{
			Field:   FieldFillingCode,
			Message: "Kode Filling terlalu panjang.",
		})
	}

	if input.DocumentReceivedDate == nil {
		violations = append(violations, Violation{
			Field:   FieldReceivedDate,
			Message: "Pilih Tgl Terima Dokumen.",
		})
	}

	if clean.Login == "" {
		// Bukan pelanggaran isian melainkan identitas yang tidak terbaca; ia dijawab
		// berbeda oleh transport, sehingga tidak dicampur ke daftar pelanggaran.
		return Draft{}, ErrCallerUnknown
	}

	if err := NewValidationError(violations); err != nil {
		return Draft{}, err
	}

	return Draft{
		ID:                   input.ID,
		ClaimNumber:          claimNumber,
		PolicyNumber:         trimTo(input.PolicyNumber, MaxClaimNumberLength),
		InsuredName:          trimTo(input.InsuredName, MaxNameLength),
		LossDate:             calendarDate(input.LossDate),
		TechnicalPIC:         trimTo(input.TechnicalPIC, MaxNameLength),
		GroupPanel:           trimTo(input.GroupPanel, MaxCodeLength),
		DocumentReceivedDate: calendarDate(input.DocumentReceivedDate),
		SheetCount:           input.SheetCount,
		DocumentTypeCode:     documentType,
		DocumentKindCode:     documentKind,
		BoxName:              boxName,
		FillingCode:          fillingCode,
		InputUser:            clean.Login,
		BranchCode:           strings.TrimSpace(branchCode),
	}, nil
}

// trimTo memangkas spasi lalu memotong panjangnya.
//
// Ia MEMOTONG alih-alih menolak karena keenam isian yang memakainya tidak diketik
// pengguna — semuanya ikut dari baris klaim yang dipilih. Menolak seluruh penyimpanan
// karena nama tertanggung di basis data kebetulan panjang akan membuat berkas klaim itu
// tidak pernah dapat diarsipkan, dan pengguna tidak punya cara memperbaikinya dari layar
// ini.
//
// Pemotongannya sendiri bukan hal baru: `InputRegister_act` memotong `ObjectName` pada
// 3.800 karakter dengan alasan yang sama.
func trimTo(value string, limit int) string {
	clean := strings.TrimSpace(value)
	if len(clean) <= limit {
		return clean
	}
	return clean[:limit]
}
