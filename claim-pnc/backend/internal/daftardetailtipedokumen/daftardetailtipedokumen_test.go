package daftardetailtipedokumen_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/daftardetailtipedokumen"
)

// Lebar nomor urut EMPAT, dibaca dari `Database/PEGA_LST_DET_TYPE_DOC.prc:19`.
//
// Uji ini ada karena modul master di aplikasi ini memakai pola ID yang sama dengan lebar
// yang BERBEDA-BEDA — tiga pada Master Status Klaim, empat di sini dan pada Daftar Tipe
// Dokumen, lima pada Master Dokumen Travel. Menyalin lebar dari modul tetangga adalah
// kekeliruan yang paling mungkin terjadi sekaligus yang paling sulit terlihat: ID yang
// dihasilkan tetap "masuk akal" di layar padahal tidak dikenali data historis.
func TestIDFollowsTheFourDigitShapeOfTheOldProcedure(t *testing.T) {
	require.Equal(t, 4, daftardetailtipedokumen.SequenceDigits)

	require.Equal(t, "10001", daftardetailtipedokumen.FormatID("1", 1))
	require.Equal(t, "10042", daftardetailtipedokumen.FormatID("1", 42))
	require.Equal(t, "19999", daftardetailtipedokumen.FormatID("1", 9999))
}

// Nomor di atas 9999 dikembalikan APA ADANYA, tidak dipotong — sama seperti LPAD Oracle.
//
// Memotongnya akan menghasilkan ID GANDA, dan ID ganda di sini berarti dua rincian dokumen
// berbeda berbagi satu kunci yang dirujuk `LST_TYPE_DOC_BUSINESS.DOC_TYPE_DT_ID` milik
// modul MENU_ID 42. Penyisipan yang gagal dengan pesan jelas jauh lebih baik daripada itu.
func TestSequenceBeyondFourDigitsIsNotTruncated(t *testing.T) {
	require.Equal(t, "110000", daftardetailtipedokumen.FormatID("1", 10000))
	require.Equal(t, "1123456", daftardetailtipedokumen.FormatID("1", 123456))
}

// Kode situs ikut dipangkas: ia dibaca dari kolom yang dapat berisi padding.
func TestSiteCodeIsTrimmedBeforeBeingJoined(t *testing.T) {
	require.Equal(t, "10001", daftardetailtipedokumen.FormatID(" 1 ", 1))
}

// STS_WAJIB DITULIS sebagai teks "Ya"/"Tidak", bukan angka.
//
// Uji ini mengunci keputusan Work Owner 2026-09-23 beserta buktinya:
// `Activity/SetTypePDFAdjustment-Act.xml` membandingkan `.STS_WAJIB=="Ya"` dan TIDAK
// mengenali "1". Menulis angka akan membuatnya berhenti mengenali dokumen wajib — tanpa
// satu pun galat, hanya jenis PDF yang salah pilih.
func TestMandatoryIsStoredAsIndonesianTextNotDigits(t *testing.T) {
	require.Equal(t, "Ya", daftardetailtipedokumen.MandatoryText(true))
	require.Equal(t, "Tidak", daftardetailtipedokumen.MandatoryText(false))

	require.Equal(t, "Ya", daftardetailtipedokumen.MandatoryYes)
	require.Equal(t, "Tidak", daftardetailtipedokumen.MandatoryNo)
}

// PEMBACAAN-nya menerima keempat nilai yang benar-benar ada di produksi.
//
// `ValidationUploadDocument_act` dan `ValidationUploadRegister` sama-sama membandingkan
// kolom ini dengan "Ya", "Tidak", "1", DAN "0" — bukti langsung bahwa data lamanya
// bercampur. Menolak yang tidak dikenal akan menggagalkan seluruh daftar karena satu baris
// warisan.
func TestMandatoryIsReadLenientlyBecauseLegacyRowsAreMixed(t *testing.T) {
	for _, stored := range []string{"Ya", "ya", "YA", " Ya ", "1", "y", "true"} {
		require.True(t, daftardetailtipedokumen.MandatoryFrom(stored), "nilai %q seharusnya dibaca wajib", stored)
	}
	for _, stored := range []string{"Tidak", "tidak", "0", "", "   ", "entah"} {
		require.False(t, daftardetailtipedokumen.MandatoryFrom(stored), "nilai %q seharusnya dibaca tidak wajib", stored)
	}
}

// Clean memangkas spasi tepi SELURUH isian teks.
//
// Bukan kerapian: kolom bertipe CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa
// memberi tanda apa pun, sehingga tanpa pemangkasan apa yang disimpan dan apa yang dibaca
// kembali dapat berbeda — dan selisih itu tidak terlihat di layar.
func TestCleanTrimsEveryTextField(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{
		DocumentTypeID:            "  10001  ",
		Detail:                    "\tFormulir Laporan Kerugian\n",
		InsuredStatus:             "  Tertanggung  ",
		CauseOfLossID:             " 1001 ",
		CauseOfLossDescription:    "  Contoh Golongan A  ",
		ObjectDocumentID:          " 10002 ",
		ObjectDocumentDescription: "\tPolis Asli\n",
		Risk:                      "  0  ",
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "  003  ", Mandatory: true, MinDocument: 1},
		},
	}.Clean()

	require.Equal(t, "10001", cleaned.DocumentTypeID)
	require.Equal(t, "Formulir Laporan Kerugian", cleaned.Detail)
	require.Equal(t, "Tertanggung", cleaned.InsuredStatus)
	require.Equal(t, "1001", cleaned.CauseOfLossID)
	require.Equal(t, "Contoh Golongan A", cleaned.CauseOfLossDescription)
	require.Equal(t, "10002", cleaned.ObjectDocumentID)
	require.Equal(t, "Polis Asli", cleaned.ObjectDocumentDescription)
	require.Equal(t, "0", cleaned.Risk)
	require.Len(t, cleaned.Businesses, 1)
	require.Equal(t, "003", cleaned.Businesses[0].BusinessID)
}

// Keterangan TANPA kode tetap diterima, dan itu bukan kelonggaran melainkan tiruan.
//
// Ketiga autocomplete di layar lama ber-`pyAllowFreeFormInput=true`, sehingga keterangan
// di luar master boleh diketik — dan pada keadaan itu `pyPropertyTarget` tidak pernah
// terisi, sehingga kodenya kosong. Karena DOC_COL_INFO dan OBJ_DOC_DESC punya kolomnya
// sendiri, isian itu TERSIMPAN utuh.
//
// Uji ini yang menjaga keduanya tidak kelak "dirapikan" menjadi wajib berpasangan.
func TestDescriptionWithoutACodeIsAccepted(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{
		CauseOfLossID:             "",
		CauseOfLossDescription:    "Keterangan yang diketik sendiri",
		ObjectDocumentID:          "",
		ObjectDocumentDescription: "Objek yang diketik sendiri",
	}.Clean()

	require.NoError(t, cleaned.Check())
	require.Empty(t, cleaned.CauseOfLossID)
	require.Equal(t, "Keterangan yang diketik sendiri", cleaned.CauseOfLossDescription)
	require.Empty(t, cleaned.ObjectDocumentID)
	require.Equal(t, "Objek yang diketik sendiri", cleaned.ObjectDocumentDescription)
}

// Keterangan dibatasi dengan batas KETERANGAN, kodenya dengan batas KODE.
//
// Keduanya kolom yang berbeda dan lebarnya tidak sama; memakai satu batas untuk keduanya
// akan menolak keterangan yang panjangnya wajar hanya karena kodenya pendek.
func TestDescriptionAndCodeUseTheirOwnLengthLimits(t *testing.T) {
	// Keterangan sepanjang batas kode + 1 — sah, karena batas keterangan jauh lebih besar.
	require.NoError(t, daftardetailtipedokumen.Input{
		CauseOfLossDescription: strings.Repeat("x", daftardetailtipedokumen.MaxReferenceLength+1),
	}.Clean().Check())

	// Kode sepanjang batas kode + 1 — ditolak.
	require.Error(t, daftardetailtipedokumen.Input{
		CauseOfLossID: strings.Repeat("x", daftardetailtipedokumen.MaxReferenceLength+1),
	}.Clean().Check())
}

// Isian kosong TETAP kosong setelah Clean — ia tidak diganti nilai bawaan apa pun.
//
// Uji ini mengunci `P-5` pada tingkat domain: layar lama tidak memuat satu pun
// `pyRequired=true` pada isian mana pun (`Section/BrowseListDetailTypeDocument-Section.xml`),
// dan `Database/PEGA_LST_DET_TYPE_DOC.prc` menyisipkan tanpa memeriksa apa pun.
//
// Bila kelak layar ini diperketat, uji inilah yang akan gagal lebih dulu — dan itu memang
// gunanya: perubahan kebijakan harus terlihat, bukan menyelinap.
func TestCleanKeepsEmptyFieldsEmpty(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{}.Clean()

	require.Empty(t, cleaned.DocumentTypeID)
	require.Empty(t, cleaned.Detail)
	require.Empty(t, cleaned.InsuredStatus)
	require.Empty(t, cleaned.CauseOfLossID)
	require.Empty(t, cleaned.CauseOfLossDescription)
	require.Empty(t, cleaned.ObjectDocumentID)
	require.Empty(t, cleaned.ObjectDocumentDescription)
	require.Empty(t, cleaned.Risk)
	require.NoError(t, cleaned.Check())
}

// Baris grid yang bisnisnya belum dipilih dibuang, dan itu bukan validasi.
//
// Grid di layar selalu menyisakan baris yang baru ditambahkan tetapi belum diisi.
// Menyimpannya berarti menulis aturan yang tidak menunjuk bisnis mana pun, lalu membacanya
// kembali sebagai baris hantu di form berikutnya.
func TestCleanDropsBusinessRowsWithoutABusiness(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "003", Mandatory: true, MinDocument: 1},
			{BusinessID: "   ", Mandatory: true, MinDocument: 9},
			{BusinessID: "", Mandatory: false, MinDocument: 0},
			{BusinessID: "006"},
		},
	}.Clean()

	require.Len(t, cleaned.Businesses, 2)
	require.Equal(t, "003", cleaned.Businesses[0].BusinessID)
	require.Equal(t, "006", cleaned.Businesses[1].BusinessID)
}

// Baris yang bisnisnya terisi TIDAK dibuang meski status wajib dan jumlah minimumnya
// bernilai bawaan — keduanya memang punya arti pada nilai bawaannya.
func TestCleanKeepsBusinessRowsThatOnlyHaveABusiness(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "005"},
		},
	}.Clean()

	require.Len(t, cleaned.Businesses, 1)
	require.False(t, cleaned.Businesses[0].Mandatory)
	require.Equal(t, 0, cleaned.Businesses[0].MinDocument)
}

// MIN_DOC negatif diratakan menjadi nol, bukan ditolak.
//
// "Paling sedikit minus satu berkas" bukan aturan yang dapat dipenuhi maupun dilanggar,
// dan meratakannya menjaga perlakuan tetap sejalan dengan layar yang tidak memvalidasi.
func TestCleanFlattensNegativeMinimumToZero(t *testing.T) {
	cleaned := daftardetailtipedokumen.Input{
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "003", MinDocument: -5},
		},
	}.Clean()

	require.Len(t, cleaned.Businesses, 1)
	require.Equal(t, 0, cleaned.Businesses[0].MinDocument)
}

// Yang diperiksa Check hanyalah PANJANG, dan itu bentuk kolom — bukan aturan bisnis.
func TestCheckRejectsOnlyOverlongFields(t *testing.T) {
	err := daftardetailtipedokumen.Input{
		Detail: strings.Repeat("x", daftardetailtipedokumen.MaxDetailLength+1),
	}.Clean().Check()

	var validationError *daftardetailtipedokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 1)
	require.Equal(t, daftardetailtipedokumen.FieldDetail, validationError.Violation[0].Field)
}

// Panjangnya dihitung dalam RUNE, bukan byte.
//
// Satu huruf beraksen memakan dua byte, dan menghitung byte akan membuat batas terasa
// berubah-ubah bagi pengguna — teks 200 huruf ditolak hanya karena sebagian hurufnya
// beraksen.
func TestCheckCountsRunesNotBytes(t *testing.T) {
	// 200 rune, tetapi 400 byte.
	err := daftardetailtipedokumen.Input{
		Detail: strings.Repeat("é", daftardetailtipedokumen.MaxDetailLength),
	}.Clean().Check()

	require.NoError(t, err)
}

// SELURUH pelanggaran dikumpulkan sekaligus, bukan yang pertama saja.
//
// Meniru layar Pega yang menampilkan semua pesannya bersamaan. Pada form dengan tujuh
// isian, melaporkan satu pelanggaran per percobaan akan sangat menyiksa petugas.
func TestCheckCollectsEveryViolationAtOnce(t *testing.T) {
	err := daftardetailtipedokumen.Input{
		Detail:        strings.Repeat("x", daftardetailtipedokumen.MaxDetailLength+1),
		InsuredStatus: strings.Repeat("x", daftardetailtipedokumen.MaxInsuredStatusLength+1),
		Risk:          strings.Repeat("9", daftardetailtipedokumen.MaxRiskLength+1),
	}.Clean().Check()

	var validationError *daftardetailtipedokumen.ValidationError
	require.ErrorAs(t, err, &validationError)
	require.Len(t, validationError.Violation, 3)
}

// Kode rujukan yang TIDAK ADA di master tetap diterima.
//
// Layar lama memakai autocomplete yang menerima ketikan di luar daftar, dan procedure
// penyimpannya tidak memeriksa apa pun. Menolaknya di sini adalah perubahan perilaku,
// bukan perbaikan — dan akan menghalangi petugas memperbaiki baris warisan yang rujukannya
// memang sudah hilang.
func TestCheckAcceptsReferencesThatDoNotExistInAnyMaster(t *testing.T) {
	err := daftardetailtipedokumen.Input{
		DocumentTypeID:   "TIDAK-ADA",
		CauseOfLossID:    "TIDAK-ADA",
		ObjectDocumentID: "TIDAK-ADA",
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "TIDAK-ADA"},
		},
	}.Clean().Check()

	require.NoError(t, err)
}

// Bisnis kembar TIDAK ditolak, mengikuti grid Pega yang tidak punya satu pun penanda
// keunikan.
func TestCheckAcceptsDuplicateBusinessRows(t *testing.T) {
	err := daftardetailtipedokumen.Input{
		Businesses: []daftardetailtipedokumen.BusinessInput{
			{BusinessID: "003", Mandatory: true, MinDocument: 1},
			{BusinessID: "003", Mandatory: false, MinDocument: 2},
		},
	}.Clean().Check()

	require.NoError(t, err)
}

// Resiko kosong diterima.
//
// `toDecimal("")` di Pega menghasilkan nol, sehingga baris tanpa resiko berperilaku sama
// dengan baris beresiko nol pada `Activity/SetTypePDFAdjustment-Act.xml`. Menolaknya akan
// mengubah perilaku layar yang tidak sedang dimigrasikan.
func TestCheckAcceptsEmptyRisk(t *testing.T) {
	require.NoError(t, daftardetailtipedokumen.Input{Risk: ""}.Clean().Check())
}
