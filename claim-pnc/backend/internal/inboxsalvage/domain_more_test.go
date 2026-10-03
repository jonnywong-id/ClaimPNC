package inboxsalvage_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

var siti = inboxsalvage.Caller{Login: "SITIRAHAYU"}

func TestValidationErrorMessageListsEveryViolation(t *testing.T) {
	require.Equal(t, "inboxsalvage: isian tidak sah",
		inboxsalvage.NewValidationError(nil).Error())

	err := inboxsalvage.NewValidationError([]inboxsalvage.Violation{
		{Field: "a", Message: "satu"},
		{Field: "b", Message: "dua"},
	})
	require.Equal(t, "inboxsalvage: a: satu; b: dua", err.Error())
}

func TestOffsetAndSliceCutTheRequestedPage(t *testing.T) {
	require.Equal(t, 0, inboxsalvage.Pagination{}.Offset())
	require.Equal(t, 40, inboxsalvage.Pagination{Page: 3, Size: 20}.Offset())

	rows := []inboxsalvage.Row{{ClaimNo: "1"}, {ClaimNo: "2"}, {ClaimNo: "3"}}

	page := inboxsalvage.Slice(rows, inboxsalvage.Pagination{Page: 2, Size: 2})
	require.Equal(t, 3, page.Total)
	require.Equal(t, []inboxsalvage.Row{{ClaimNo: "3"}}, page.Items, "halaman terakhir dipotong")

	beyond := inboxsalvage.Slice(rows, inboxsalvage.Pagination{Page: 5, Size: 2})
	require.Equal(t, 3, beyond.Total)
	require.NotNil(t, beyond.Items)
	require.Empty(t, beyond.Items)
}

func TestSoldStatusFollowsTheDetailPanelMapping(t *testing.T) {
	require.Equal(t, "Terjual", inboxsalvage.SoldStatusOf("1"))
	require.Equal(t, "Tidak terjual", inboxsalvage.SoldStatusOf("0"))
	require.Equal(t, "Waiting approval", inboxsalvage.SoldStatusOf(" 3 "))
	require.Equal(t, "Belum terjual", inboxsalvage.SoldStatusOf(""))
	// Kode 2 tidak dipetakan panel detail, sama seperti di Pega.
	require.Equal(t, "-", inboxsalvage.SoldStatusOf("2"))
}

func TestPositionLabelComesFromTheTabThatFiltersTheCode(t *testing.T) {
	require.Equal(t, "", inboxsalvage.PositionLabelOf("  "))
	require.Equal(t, "Salvage Balai Lelang", inboxsalvage.PositionLabelOf("1"))
	// Kode tanpa daftar dikembalikan apa adanya, bukan disembunyikan.
	require.Equal(t, "6", inboxsalvage.PositionLabelOf(" 6 "))
}

func TestDetailKeyFollowsTheQueryFamily(t *testing.T) {
	require.Equal(t, inboxsalvage.DetailKeySubmission,
		inboxsalvage.DetailKeyOf(inboxsalvage.FamilySalvage))
	require.Equal(t, inboxsalvage.DetailKeyClaim, inboxsalvage.DetailKeyOf(inboxsalvage.FamilyClaim))
	require.Equal(t, inboxsalvage.DetailKeyClaim,
		inboxsalvage.DetailKeyOf(inboxsalvage.FamilyClaimObject))
}

func TestTabLookupsRejectUnknownCodes(t *testing.T) {
	_, found := inboxsalvage.FindAnyTab("tidak-ada")
	require.False(t, found)

	_, found = inboxsalvage.TabForLegacy("", "99")
	require.False(t, found, "tipe2 yang tidak dikenal tidak jatuh ke tipe")

	_, found = inboxsalvage.TabForLegacy("", "")
	require.False(t, found)

	_, found = inboxsalvage.TabForLegacy("99", "")
	require.False(t, found)
}

func TestQueryForTabRejectsUnknownCallerAndOverlongSearchAndDropsUnusableSearch(t *testing.T) {
	tab, _ := inboxsalvage.FindTab(inboxsalvage.TabHistori)

	_, err := inboxsalvage.NewQueryForTab(tab, inboxsalvage.QueryInput{}, inboxsalvage.Caller{})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)

	_, err = inboxsalvage.NewQueryForTab(tab,
		inboxsalvage.QueryInput{Search: strings.Repeat("x", 101)}, siti)
	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxsalvage.FieldSearch, validation.Violations[0].Field)

	// Daftar tanpa kotak pencarian membuang kata kuncinya.
	q, err := inboxsalvage.NewQueryForTab(inboxsalvage.Tab{Code: "x"},
		inboxsalvage.QueryInput{Search: "PNC"}, siti)
	require.NoError(t, err)
	require.Empty(t, q.Search)
}

func TestMatchesWithoutSearchAndPartialPICSearch(t *testing.T) {
	require.True(t, inboxsalvage.Query{}.Matches(inboxsalvage.Row{}))

	exactOnly := inboxsalvage.Query{
		Tab: inboxsalvage.Tab{SearchExact: true}, Search: "SITI",
	}
	require.False(t, exactOnly.Matches(inboxsalvage.Row{ClaimNo: "X", PIC: "SITI"}),
		"tab cocok-persis tanpa PIC tidak mencocokkan PIC")

	partialPIC := inboxsalvage.Query{
		Tab: inboxsalvage.Tab{SearchByPIC: true}, Search: "siti",
	}
	require.True(t, partialPIC.Matches(inboxsalvage.Row{ClaimNo: "X", PIC: "SITIRAHAYU"}))
	require.False(t, partialPIC.Matches(inboxsalvage.Row{ClaimNo: "X", PIC: "BUDI"}))
}

func TestAuctionStatusTreatsUnreadableValuesAsFilled(t *testing.T) {
	require.Equal(t, inboxsalvage.AuctionSold, inboxsalvage.AuctionStatusOf("bukan-angka"))
	require.Equal(t, inboxsalvage.AuctionUnsold, inboxsalvage.AuctionStatusOf("0,00"))
}

// Tanggal input SESUDAH hari ini tidak menghasilkan aging negatif.
func TestAgingNeverGoesNegative(t *testing.T) {
	require.Equal(t, "0 day", inboxsalvage.AgingOf("2026-09-30", "2026-09-25"))
}

// TodayWIB selalu berbentuk tanggal ISO, sehingga dapat dibaca AgingOf.
func TestTodayWIBIsAnISODateCloseToNow(t *testing.T) {
	today := inboxsalvage.TodayWIB()
	parsed, err := time.Parse(inboxsalvage.DateLayout, today)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), parsed, 48*time.Hour)
}

func TestParseUploadIgnoresBlankAndDuplicateHeadersAndShortRows(t *testing.T) {
	items, err := inboxsalvage.ParseUpload(strings.NewReader(
		string(rune(0xFEFF)) + "Item,,ITEM,Quantity\nBesi,abaikan,lain\nKayu,,,3\n"))
	require.NoError(t, err)
	require.Equal(t, []inboxsalvage.DetailItem{
		{Name: "Besi"},
		{Name: "Kayu", Quantity: "3"},
	}, items, "kolom ganda memakai yang pertama; baris pendek tetap terbaca")
}

func TestParseUploadReportsAMalformedHeaderAndRow(t *testing.T) {
	_, err := inboxsalvage.ParseUpload(strings.NewReader("\"Item\n"))
	require.Error(t, err)
	require.False(t, errors.Is(err, inboxsalvage.ErrUploadEmpty))

	_, err = inboxsalvage.ParseUpload(strings.NewReader("Item\n\"Besi\n"))
	require.Error(t, err)
}

func TestParseUploadRejectsMoreThanTheMaximumRows(t *testing.T) {
	_, err := inboxsalvage.ParseUpload(strings.NewReader(
		"Item\n" + strings.Repeat("Besi\n", 501)))
	require.ErrorIs(t, err, inboxsalvage.ErrUploadTooManyRows)
}

func TestFormRejectsUnknownStatusUpdateWithoutIDAndBadItems(t *testing.T) {
	input := validForm()
	input.Mode = inboxsalvage.FormModeUpdate
	input.Status = "99"
	input.Items = []inboxsalvage.DetailItem{
		{Quantity: "abc"},
		{Name: "Besi", Quantity: "1,5"},
	}

	_, err := inboxsalvage.NewForm(input, siti)
	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)

	messages := []string{}
	for _, violation := range validation.Violations {
		messages = append(messages, violation.Message)
	}
	require.Contains(t, messages, "Status Salvage tidak dikenal.")
	require.Contains(t, messages,
		"Pengajuan yang diubah tidak dikenali. Muat ulang halaman lalu ulangi.")
	require.Contains(t, messages, "Nama Item pada baris 1 harus diisi.")
	require.Contains(t, messages, "Jumlah Item pada baris 1 harus berupa angka.")
}

func TestFormRejectsMoreThanTheMaximumItems(t *testing.T) {
	input := validForm()
	for i := 0; i < 501; i++ {
		input.Items = append(input.Items, inboxsalvage.DetailItem{Name: "Besi"})
	}

	_, err := inboxsalvage.NewForm(input, siti)
	var validation *inboxsalvage.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxsalvage.FieldFormItems, validation.Violations[0].Field)
	require.Contains(t, validation.Violations[0].Message, "melebihi 500 baris")
}

func TestFormRejectsAnUnknownCaller(t *testing.T) {
	_, err := inboxsalvage.NewForm(validForm(), inboxsalvage.Caller{Login: " "})
	require.ErrorIs(t, err, inboxsalvage.ErrCallerUnknown)
}

func TestFormAcceptsAKnownStatus(t *testing.T) {
	input := validForm()
	input.Status = inboxsalvage.StatusOptions()[0].Code

	form, err := inboxsalvage.NewForm(input, siti)
	require.NoError(t, err)
	require.Equal(t, input.Status, form.Status)
}
