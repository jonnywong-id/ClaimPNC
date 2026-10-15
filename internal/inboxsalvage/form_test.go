package inboxsalvage_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// validForm adalah isian terkecil yang sah, dipakai sebagai titik awal setiap uji yang
// mengubah satu hal saja.
func validForm() inboxsalvage.FormInput {
	return inboxsalvage.FormInput{
		ClaimNo:      "PNC-2044",
		ObjectName:   "Panel Listrik",
		CoverageName: "Property All Risk",
		SalvageType:  "Besi Tua",
		InputDate:    "2026-09-25",
	}
}

func TestFormAcceptsTheSmallestValidSubmission(t *testing.T) {
	form, err := inboxsalvage.NewForm(validForm(), inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.NoError(t, err)

	require.Equal(t, inboxsalvage.FormModeInsert, form.Mode)
	require.Equal(t, "SITIRAHAYU", form.Caller.Login)

	// Pengajuan baru LANGSUNG masuk antrean checker di portal ASM. Nilainya terbaca dari
	// komentar langkah 6 `SetStsSalvagePNC_act` apa adanya.
	require.Equal(t, inboxsalvage.TransferStatusOnSubmit, form.TransferStatus)
	require.Equal(t, "3", form.TransferStatus)
}

// Kedua pesan ini ditulis PERSIS seperti di `GCNMNewSalvage_act` langkah 4 dan 5 (`D-13`).
// Mengubah satu huruf pun membuat pengguna yang terbiasa dengan Pega membacanya sebagai
// pesan yang berbeda.
func TestFormRepeatsTheTwoLegacyMessagesWordForWord(t *testing.T) {
	input := validForm()
	input.ObjectName = ""
	input.CoverageName = ""

	_, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})

	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)

	messages := []string{}
	for _, violation := range validation.Violations {
		messages = append(messages, violation.Message)
	}

	require.Contains(t, messages, "Nama object harus diisi")
	require.Contains(t, messages, "Nama coverage harus diisi")
}

// SELURUH pelanggaran dikumpulkan, bukan yang pertama saja (`P-5`). Form ini punya tujuh
// belas isian, dan menyampaikannya satu per satu akan menyiksa pengguna.
func TestFormCollectsEveryViolationAtOnce(t *testing.T) {
	_, err := inboxsalvage.NewForm(inboxsalvage.FormInput{
		MinimumValue: "bukan angka",
	}, inboxsalvage.Caller{Login: "SITIRAHAYU"})

	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)

	fields := map[string]bool{}
	for _, violation := range validation.Violations {
		fields[violation.Field] = true
	}

	require.True(t, fields[inboxsalvage.FieldFormClaimNo])
	require.True(t, fields[inboxsalvage.FieldFormObjectName])
	require.True(t, fields[inboxsalvage.FieldFormCoverageName])
	require.True(t, fields[inboxsalvage.FieldFormSalvageType])
	require.True(t, fields[inboxsalvage.FieldFormMinimum])
	require.GreaterOrEqual(t, len(validation.Violations), 5)
}

// Nomor klaim DIBESARKAN hurufnya, mengikuti `@toUpperCase` pada langkah 2
// `GCNMNewSalvage_act`.
//
// Tanpa itu, baris yang nomor klaimnya diketik huruf kecil tidak akan pernah tergabung ke
// `T_CLAIM_PNC` — gabungannya berbentuk perangkaian teks, yang peka huruf besar-kecil.
func TestFormUppercasesTheClaimNumber(t *testing.T) {
	input := validForm()
	input.ClaimNo = "  pnc-2044  "

	form, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.NoError(t, err)
	require.Equal(t, "PNC-2044", form.ClaimNo)
}

// Isian bernilai angka yang KOSONG diterima — sistem lama pun menerimanya, dan procedure
// menuliskannya sebagai NULL. Yang ditolak hanyalah isian terisi yang bukan angka.
func TestFormAcceptsEmptyMoneyFieldsButRejectsNonNumericOnes(t *testing.T) {
	caller := inboxsalvage.Caller{Login: "SITIRAHAYU"}

	input := validForm()
	input.MinimumValue = ""
	input.Quantity = ""
	_, err := inboxsalvage.NewForm(input, caller)
	require.NoError(t, err)

	input = validForm()
	input.MinimumValue = "4.500.000"
	_, err = inboxsalvage.NewForm(input, caller)
	require.Error(t, err, "pemisah ribuan bukan angka yang dapat dibaca")

	// Koma desimal diterima dan TIDAK diubah menjadi bilangan pecahan di sini — `D-51`
	// menetapkan nilai uang disimpan presisi penuh.
	input = validForm()
	input.MinimumValue = "4500000,50"
	form, err := inboxsalvage.NewForm(input, caller)
	require.NoError(t, err)
	require.Equal(t, "4500000,50", form.MinimumValue,
		"nilainya diteruskan apa adanya, bukan dibulatkan lebih dulu")
}

// Mode ubah TANPA ID salvage DITOLAK.
//
// Diteruskan, ia akan jatuh ke cabang sisip procedure dan menerbitkan baris BARU alih-alih
// memperbarui yang ada — pengajuan ganda tanpa satu pun galat.
func TestUpdateWithoutASalvageIDIsRejected(t *testing.T) {
	input := validForm()
	input.Mode = inboxsalvage.FormModeUpdate

	_, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.Error(t, err)

	input.SalvageID = "101"
	form, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.NoError(t, err)
	require.Equal(t, "101", form.SalvageID)
}

func TestFormRejectsAnUnknownSalvageStatus(t *testing.T) {
	input := validForm()
	input.Status = "99"

	_, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.Error(t, err)

	// Kedua kode yang sah datang dari `SetDataSalavage_act` langkah 5.
	for _, option := range inboxsalvage.StatusOptions() {
		input.Status = option.Code
		_, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
		require.NoError(t, err, "kode %s seharusnya sah", option.Code)
	}
}

// Baris kosong DIBUANG, bukan ditolak. Berkas CSV yang disunting di Excel hampir selalu
// membawa baris kosong di ujungnya.
func TestEmptyDetailRowsAreDroppedRatherThanRejected(t *testing.T) {
	input := validForm()
	input.Items = []inboxsalvage.DetailItem{
		{Name: "Besi siku", Quantity: "10", Unit: "batang"},
		{},
		{Name: "  ", Quantity: "  "},
		{Name: "Kabel", Quantity: "5", Unit: "roll"},
	}

	form, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})
	require.NoError(t, err)
	require.Len(t, form.Items, 2)
	require.Equal(t, "Besi siku", form.Items[0].Name)
	require.Equal(t, "Kabel", form.Items[1].Name)
}

// Baris yang TIDAK kosong tetapi kehilangan nama item ditolak, dan pesannya menyebut baris
// KE BERAPA — bukan sekadar "ada baris yang salah".
func TestADetailRowWithoutANameNamesItsRowNumber(t *testing.T) {
	input := validForm()
	input.Items = []inboxsalvage.DetailItem{
		{Name: "Besi siku", Quantity: "10"},
		{Quantity: "5", Unit: "roll"},
	}

	_, err := inboxsalvage.NewForm(input, inboxsalvage.Caller{Login: "SITIRAHAYU"})

	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Violations[0].Message, "baris 2")
}

func TestParseUploadReadsTheFourColumnsLegacyReads(t *testing.T) {
	berkas := strings.NewReader(
		"Item,Quantity,Satuan,REMARKS\n" +
			"Besi siku,10,batang,kondisi baik\n" +
			"Kabel tembaga,5,roll,\n")

	items, err := inboxsalvage.ParseUpload(berkas)
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.Equal(t, "Besi siku", items[0].Name)
	require.Equal(t, "10", items[0].Quantity)
	require.Equal(t, "batang", items[0].Unit)
	require.Equal(t, "kondisi baik", items[0].Remarks)

	require.Equal(t, "Kabel tembaga", items[1].Name)
	require.Equal(t, "", items[1].Remarks)
}

// Judul kolom TIDAK peka huruf besar-kecil, dan itu bukan kelonggaran yang dikarang:
// sistem lama sendiri menuliskan `REMARKS` seluruhnya kapital sementara tiga kolom lain
// tidak.
func TestUploadColumnTitlesAreCaseInsensitiveAndTrimmed(t *testing.T) {
	berkas := strings.NewReader(
		" ITEM , quantity ,SATUAN,remarks\n" +
			"Besi siku,10,batang,-\n")

	items, err := inboxsalvage.ParseUpload(berkas)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Besi siku", items[0].Name)
	require.Equal(t, "10", items[0].Quantity)
}

// Penanda urutan bita yang Excel tuliskan di awal berkas DIBUANG.
//
// Tanpa itu, judul kolom pertama terbaca sebagai sesuatu yang tidak pernah cocok — dan
// berkas yang benar ditolak dengan alasan kolom wajib tidak ada.
func TestUploadSurvivesTheByteOrderMarkExcelWrites(t *testing.T) {
	berkas := strings.NewReader("\ufeffItem,Quantity\nBesi siku,10\n")

	items, err := inboxsalvage.ParseUpload(berkas)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Besi siku", items[0].Name)
}

// Hanya `Item` yang wajib. Ketiga kolom lain boleh tidak ada sama sekali — `UploadDetail
// Salvage` menyalin keempatnya tanpa memeriksa satu pun.
func TestUploadNeedsOnlyTheItemColumn(t *testing.T) {
	items, err := inboxsalvage.ParseUpload(strings.NewReader("Item\nBesi siku\n"))
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "", items[0].Quantity)

	_, err = inboxsalvage.ParseUpload(strings.NewReader("Quantity,Satuan\n10,batang\n"))
	require.ErrorIs(t, err, inboxsalvage.ErrUploadColumnMissing)
}

// Berkas yang disimpan Excel dengan setelan Indonesia memakai TITIK KOMA. Ia terbaca
// sebagai satu kolom, dan ditolak dengan alasan kolom wajib tidak ada — bukan diterima
// diam-diam sebagai nol baris.
func TestUploadRejectsASemicolonSeparatedFileRatherThanReadingItAsEmpty(t *testing.T) {
	berkas := strings.NewReader("Item;Quantity;Satuan\nBesi siku;10;batang\n")

	_, err := inboxsalvage.ParseUpload(berkas)
	require.ErrorIs(t, err, inboxsalvage.ErrUploadColumnMissing)
}

func TestUploadRejectsAnEmptyFile(t *testing.T) {
	_, err := inboxsalvage.ParseUpload(strings.NewReader(""))
	require.ErrorIs(t, err, inboxsalvage.ErrUploadEmpty)
}
