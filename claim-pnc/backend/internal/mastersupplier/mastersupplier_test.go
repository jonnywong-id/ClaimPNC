package mastersupplier_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/mastersupplier"
)

// validInput adalah isian yang seluruhnya sah, dipakai sebagai titik awal.
//
// Setiap uji mengubah SATU hal darinya, sehingga yang diperiksa memang hal itu dan bukan
// kebetulan isian lain juga tidak lengkap.
func validInput() mastersupplier.Input {
	return mastersupplier.Input{
		Name:            "Supplier Uji",
		Address:         "Jalan Uji Nomor 1",
		City:            "Jakarta Pusat",
		BranchName:      "Cabang Uji",
		Country:         "Indonesia",
		Phone:           "021-0000000",
		ContactPerson:   "Narahubung Uji",
		PartnerStatus:   "1",
		SupplyType:      mastersupplier.SupplyTypeOther,
		TermOfPayment:   "30",
		TermOfDelivery:  "7",
		Bank:            "Bank Uji",
		AccountNumber:   "1234567890",
		SupplierType:    "1",
		ActiveRequested: mastersupplier.ActiveYes,
	}
}

func TestValidInputPasses(t *testing.T) {
	require.NoError(t, validInput().Clean().Check())
}

// Kelima belas isian wajib dibaca langsung dari `pyRequired=true` pada
// `Section/CreateMasterSupplier_Sec-Section.xml`.
//
// Uji ini menjaga daftarnya persis: mewajibkan lebih banyak akan menolak penambahan yang
// hari ini diterima, dan mewajibkan lebih sedikit akan meloloskan baris yang layar lamanya
// tolak.
func TestRequiredFieldsRejectEmpty(t *testing.T) {
	blank := map[string]func(*mastersupplier.Input){
		"nama":             func(i *mastersupplier.Input) { i.Name = "" },
		"alamat":           func(i *mastersupplier.Input) { i.Address = "" },
		"kota":             func(i *mastersupplier.Input) { i.City = "" },
		"nama_cabang":      func(i *mastersupplier.Input) { i.BranchName = "" },
		"negara":           func(i *mastersupplier.Input) { i.Country = "" },
		"telepon":          func(i *mastersupplier.Input) { i.Phone = "" },
		"contact_person":   func(i *mastersupplier.Input) { i.ContactPerson = "" },
		"status_rekanan":   func(i *mastersupplier.Input) { i.PartnerStatus = "" },
		"status_supply":    func(i *mastersupplier.Input) { i.SupplyType = "" },
		"term_of_payment":  func(i *mastersupplier.Input) { i.TermOfPayment = "" },
		"term_of_delivery": func(i *mastersupplier.Input) { i.TermOfDelivery = "" },
		"bank":             func(i *mastersupplier.Input) { i.Bank = "" },
		"no_account":       func(i *mastersupplier.Input) { i.AccountNumber = "" },
		"jenis_supplier":   func(i *mastersupplier.Input) { i.SupplierType = "" },
		"status_aktif":     func(i *mastersupplier.Input) { i.ActiveRequested = "" },
	}

	require.Len(t, blank, 15, "layar lama menandai tepat lima belas isian wajib")

	for field, clear := range blank {
		t.Run(field, func(t *testing.T) {
			input := validInput()
			clear(&input)

			var invalid *mastersupplier.ValidationError
			require.ErrorAs(t, input.Clean().Check(), &invalid)
			require.Len(t, invalid.Violation, 1)
			require.Equal(t, field, invalid.Violation[0].Field)
		})
	}
}

// Isian ber-`pyRequired=false` harus tetap lolos saat kosong.
func TestOptionalFieldsAcceptEmpty(t *testing.T) {
	input := validInput()
	input.PostalCode = ""
	input.Fax = ""
	input.Email = ""
	input.TaxNumber = ""
	input.Note = ""
	input.AccountName = ""
	input.BankBranch = ""
	input.AutoPayment = ""

	require.NoError(t, input.Clean().Check())
}

// SELURUH pelanggaran dikembalikan sekaligus, bukan yang pertama saja (P-5).
//
// Pada form berisi lima belas isian wajib, mengembalikannya satu per satu berarti lima
// belas kali bolak-balik untuk satu form kosong.
func TestCheckReportsEveryViolationAtOnce(t *testing.T) {
	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, mastersupplier.Input{}.Clean().Check(), &invalid)
	require.Len(t, invalid.Violation, 15)
}

// Spasi di kedua ujung dipangkas SEBELUM diperiksa, sehingga isian berisi spasi saja
// ditolak sebagai kosong — bukan lolos karena kebetulan panjangnya bukan nol.
func TestCleanTrimsBeforeCheck(t *testing.T) {
	input := validInput()
	input.Name = "   "

	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, input.Clean().Check(), &invalid)
	require.Equal(t, "nama", invalid.Violation[0].Field)
}

func TestLengthLimitRejectsTooLong(t *testing.T) {
	input := validInput()
	input.Name = string(make([]byte, mastersupplier.MaxNameLength+1))

	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, input.Clean().Check(), &invalid)
	require.Contains(t, invalid.Violation[0].Message, "paling panjang")
}

// # SUPPLIER_HE adalah turunan JENIS_STATUS, dan keduanya saling membalik
//
// Arah pertama dari `Activity/CreateNewMasterSupplier_post` step 7, arah kebalikannya dari
// `Activity/GetDataSupplier_pre` step 6.3.
func TestHeavyEquipmentDerivation(t *testing.T) {
	require.Equal(t, mastersupplier.SupplyTypeHeavyEquipment,
		mastersupplier.DeriveHeavyEquipment("1"))
	require.Equal(t, mastersupplier.SupplyTypeOther,
		mastersupplier.DeriveHeavyEquipment("0"))

	// Nilai yang TIDAK dikenal diperlakukan sebagai bukan-HE, bukan diteruskan apa adanya:
	// yang menentukan perilaku sistem hilir adalah perbandingan dengan "1", dan apa pun
	// selain itu memang bukan HE.
	require.Equal(t, mastersupplier.SupplyTypeOther,
		mastersupplier.DeriveHeavyEquipment("entah"))
	require.Equal(t, mastersupplier.SupplyTypeOther,
		mastersupplier.DeriveHeavyEquipment(""))

	require.Equal(t, mastersupplier.SupplyTypeHeavyEquipment,
		mastersupplier.DeriveSupplyType("1"))
	require.Equal(t, mastersupplier.SupplyTypeOther,
		mastersupplier.DeriveSupplyType(""))
}

// Cacat sistem lama yang TIDAK dibawa: Pega hanya punya cabang "bila" tanpa cabang
// "selain itu", sehingga supplier HE yang diubah menjadi bukan-HE tetap tersimpan sebagai
// HE dan perubahannya hilang tanpa satu pun tanda.
//
// Di sini kedua arah selalu ditulis, dan uji ini yang menjaganya.
func TestHeavyEquipmentAlwaysWritten(t *testing.T) {
	require.NotEqual(t, mastersupplier.DeriveHeavyEquipment("1"),
		mastersupplier.DeriveHeavyEquipment("0"),
		"menonaktifkan HE harus benar-benar mengubah nilainya")
}

// Bentuk ID meniru `Database/PEGA_M_SUPPLIER.prc:21` — kode situs ditambah nomor urut
// SEBELAS digit bertambal nol.
func TestComposeID(t *testing.T) {
	require.Equal(t, "0100000000001",
		mastersupplier.ComposeID("01", 1, mastersupplier.SequenceWidth))
	require.Equal(t, "0100000123456",
		mastersupplier.ComposeID("01", 123456, mastersupplier.SequenceWidth))

	// Spasi pada kode situs dipangkas: kolomnya dapat bertipe CHAR berlebar tetap (R-08),
	// dan kunci yang membawa spasi di tengahnya tidak akan pernah cocok saat dicari.
	require.Equal(t, "0100000000001",
		mastersupplier.ComposeID("  01  ", 1, mastersupplier.SequenceWidth))
}

// Nomor urut yang MELAMPAUI lebar tidak dipotong.
//
// `LPAD` Oracle memotongnya dari kanan, sehingga urutan yang melampaui sebelas digit akan
// menghasilkan kunci yang bertabrakan dengan urutan lain — diam-diam. Di sini kuncinya
// dibiarkan tumbuh: ia menjadi lebih panjang, dan itu terlihat.
func TestComposeIDDoesNotTruncate(t *testing.T) {
	long := mastersupplier.ComposeID("01", 123456789012, mastersupplier.SequenceWidth)
	require.Equal(t, "01123456789012", long)
	require.Len(t, long, 14)
}

// TGL_INSERT ditulis sebagai tanggal WIB berformat dd/MM/yyyy.
//
// Bentuknya meniru `@DateTime.FormatDateTime(...,"dd/MM/yyyy","in_ID","Asia/Jakarta")` pada
// kedua activity penyimpan.
func TestFormatJakartaDate(t *testing.T) {
	at := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	require.Equal(t, "20/09/2026", mastersupplier.FormatJakartaDate(at))
}

// Konversi WIB terjadi SEBELUM tanggalnya diambil, bukan sesudah.
//
// Ini kasus yang `R-12` catat dan yang paling mudah salah: pukul 17:30 UTC sudah pukul
// 00:30 WIB keesokan harinya. Mengambil tanggal dari waktu UTC akan menuliskan tanggal
// KEMARIN pada setiap penyimpanan yang dilakukan selepas pukul lima sore waktu Jakarta —
// sepertiga hari kerja terakhir setiap hari.
func TestFormatJakartaDateCrossesMidnight(t *testing.T) {
	beforeMidnightWIB := time.Date(2026, 9, 20, 16, 59, 0, 0, time.UTC)
	afterMidnightWIB := time.Date(2026, 9, 20, 17, 30, 0, 0, time.UTC)

	require.Equal(t, "20/09/2026", mastersupplier.FormatJakartaDate(beforeMidnightWIB))
	require.Equal(t, "21/09/2026", mastersupplier.FormatJakartaDate(afterMidnightWIB))
}

// Waktu yang datang dalam zona lain tetap dikonversi ke WIB, bukan dibaca apa adanya.
func TestFormatJakartaDateNormalisesZone(t *testing.T) {
	tokyo := time.FixedZone("JST", 9*60*60)
	at := time.Date(2026, 9, 21, 1, 0, 0, 0, tokyo) // 2026-09-20 16:00 UTC = 23:00 WIB

	require.Equal(t, "20/09/2026", mastersupplier.FormatJakartaDate(at))
}

// PROTEKSI_ID buatan aplikasi ini harus dapat dibedakan dari ID work object Pega, dan
// harus menyebut supplier-nya.
func TestComposeApprovalID(t *testing.T) {
	at := time.Date(2026, 9, 20, 3, 4, 5, 0, time.UTC)
	id := mastersupplier.ComposeApprovalID("0100000000001", at)

	require.Equal(t, "SUP.0100000000001.20260920030405", id)
}

// Waktunya dipakai dalam UTC, bukan WIB.
//
// Ia kunci teknis, bukan nilai yang dibaca pengguna: memakai zona waktu setempat pada
// sebuah kunci berarti kuncinya berulang setiap kali zona waktunya bergeser.
func TestComposeApprovalIDUsesUTC(t *testing.T) {
	tokyo := time.FixedZone("JST", 9*60*60)
	at := time.Date(2026, 9, 20, 12, 0, 0, 0, tokyo) // 03:00 UTC

	require.Equal(t, "SUP.X.20260920030000", mastersupplier.ComposeApprovalID("X", at))
}

// Dua permintaan atas supplier yang sama pada detik yang berbeda harus berbeda kuncinya.
func TestComposeApprovalIDChangesOverTime(t *testing.T) {
	first := time.Date(2026, 9, 20, 3, 0, 0, 0, time.UTC)
	second := first.Add(time.Second)

	require.NotEqual(t,
		mastersupplier.ComposeApprovalID("0100000000001", first),
		mastersupplier.ComposeApprovalID("0100000000001", second))
}

// OneViolation membungkus satu pelanggaran dalam bentuk yang SAMA dengan pelanggaran
// isian lain, supaya layar menampilkannya menempel pada isiannya.
func TestOneViolation(t *testing.T) {
	var invalid *mastersupplier.ValidationError
	require.ErrorAs(t, mastersupplier.OneViolation("nama", "pesan uji"), &invalid)
	require.Equal(t, "nama", invalid.Violation[0].Field)
	require.Equal(t, "pesan uji", invalid.Violation[0].Message)
}

// Batas panjang diulang di `SupplierForm.tsx`.
//
// Uji ini TIDAK dapat membaca berkas itu, dan tidak berpura-pura bisa. Yang dijaganya
// adalah angkanya tidak berubah diam-diam di sisi Go: bila salah satu berubah, uji ini
// gagal dan yang mengubahnya diingatkan bahwa ada tempat KEDUA yang harus ikut berubah.
func TestLengthLimitsAreStable(t *testing.T) {
	require.Equal(t, 100, mastersupplier.MaxNameLength)
	require.Equal(t, 250, mastersupplier.MaxAddressLength)
	require.Equal(t, 100, mastersupplier.MaxCityLength)
	require.Equal(t, 10, mastersupplier.MaxPostalLength)
	require.Equal(t, 30, mastersupplier.MaxPhoneLength)
	require.Equal(t, 100, mastersupplier.MaxEmailLength)
	require.Equal(t, 30, mastersupplier.MaxTaxIDLength)
	require.Equal(t, 20, mastersupplier.MaxCodeLength)
	require.Equal(t, 50, mastersupplier.MaxShortLength)
	require.Equal(t, 30, mastersupplier.MaxAccountLength)
	require.Equal(t, 500, mastersupplier.MaxNoteLength)
}

// Sandi yang artinya terbukti dari export TIDAK boleh bertambah tanpa bukti baru.
//
// Ketiganya — dan hanya ketiganya — punya arti yang terbaca dari percabangan activity.
// STS_REKANAN dan JENIS_SUPPLIER sengaja KOSONG: Master Bengkel punya bukti untuk kolom
// bernama mirip, tetapi itu kolom tabel lain pada modul lain.
func TestDefaultCodeOptionOnlyHoldsProvenValues(t *testing.T) {
	known := mastersupplier.DefaultCodeOption()

	require.Empty(t, known.PartnerStatus, "arti sandi STS_REKANAN tidak ada di export")
	require.Empty(t, known.SupplierType, "arti sandi JENIS_SUPPLIER tidak ada di export")

	require.Len(t, known.SupplyType, 2)
	require.Len(t, known.Active, 2)
	require.Len(t, known.AutoPayment, 2)
}
