package archivedokumenklaim_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/archivedokumenklaim"
)

func TestOffsetDanTotalPages(t *testing.T) {
	require.Equal(t, 40, archivedokumenklaim.Pagination{Page: 3, Size: 20}.Offset())
	require.Equal(t, 0, archivedokumenklaim.Pagination{Page: -1, Size: 500}.Offset())

	page := archivedokumenklaim.ArchivePage{Total: 41, Pagination: archivedokumenklaim.Pagination{Size: 20}}
	require.Equal(t, 3, page.TotalPages())

	page.Total = 40
	require.Equal(t, 2, page.TotalPages())

	page.Total = 0
	require.Equal(t, 1, page.TotalPages(), "hasil kosong tetap satu halaman")
}

func TestValidationErrorMenyebutSetiapField(t *testing.T) {
	err := archivedokumenklaim.NewValidationError([]archivedokumenklaim.Violation{
		{Field: "a", Message: "satu"},
		{Field: "b", Message: "dua"},
	})
	require.EqualError(t, err, "archivedokumenklaim: validasi gagal — a: satu; b: dua")
	require.NoError(t, archivedokumenklaim.NewValidationError(nil))
}

func violationFields(t *testing.T, err error) []string {
	t.Helper()
	var validation *archivedokumenklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	fields := make([]string, 0, len(validation.Violations))
	for _, v := range validation.Violations {
		fields = append(fields, v.Field)
	}
	return fields
}

func TestNewCriteriaCabangGalat(t *testing.T) {
	// Kolom yang tidak dikenal ditolak — tetapi hanya bila kata kuncinya terisi; tanpa
	// kata kunci, kolom tidak punya arti apa pun.
	_, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Column: "lain", Keyword: "PNC-1",
	})
	require.Equal(t, []string{archivedokumenklaim.FieldSearchColumn}, violationFields(t, err))

	_, err = archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{Keyword: "  "})
	require.Equal(t, []string{archivedokumenklaim.FieldKeyword}, violationFields(t, err))

	from := time.Date(2024, 2, 10, 15, 0, 0, 0, time.UTC)
	to := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	_, err = archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		From: &from, To: &to,
	})
	require.Equal(t, []string{archivedokumenklaim.FieldTo}, violationFields(t, err))
}

// Kolom dibuang bila kata kuncinya kosong, dan tanggal dipotong menjadi tanggal kalender.
//
// Kolom terpilih tanpa nilai yang dicari akan membuat kueri menyaring kolom itu dengan
// teks kosong — mengembalikan nol baris tanpa alasan yang terbaca pengguna.
func TestNewCriteriaMembuangKolomTanpaKataKunci(t *testing.T) {
	from := time.Date(2024, 2, 1, 15, 30, 0, 0, time.UTC)

	keyword, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Column:  string(archivedokumenklaim.ColumnClaimNumber),
		Keyword: " pnc-1 ", From: &from, To: &from,
	})
	require.NoError(t, err)
	require.Equal(t, archivedokumenklaim.ColumnClaimNumber, keyword.Column)
	require.Equal(t, "PNC-1", keyword.Keyword)
	require.Equal(t, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), *keyword.From)

	dated, err := archivedokumenklaim.NewCriteria(archivedokumenklaim.CriteriaInput{
		Column: string(archivedokumenklaim.ColumnClaimNumber), From: &from, To: &from,
	})
	require.NoError(t, err)
	require.Empty(t, dated.Keyword)
	require.Empty(t, string(dated.Column))
	require.Equal(t, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), *dated.From)
}

func TestNewClaimCriteria(t *testing.T) {
	_, err := archivedokumenklaim.NewClaimCriteria("no_klaim", "  ")
	require.Equal(t, []string{archivedokumenklaim.FieldClaimKeyword}, violationFields(t, err))

	criteria, err := archivedokumenklaim.NewClaimCriteria(" nama_tertanggung ", " Budi ")
	require.NoError(t, err)
	require.Equal(t, archivedokumenklaim.ClaimByInsuredName, criteria.Type)
	require.Equal(t, "Budi", criteria.Value, "huruf tidak dibesarkan")
}

func validInput() archivedokumenklaim.DraftInput {
	received := time.Date(2024, 5, 20, 9, 0, 0, 0, time.UTC)
	return archivedokumenklaim.DraftInput{
		ClaimNumber:          "PNC-1",
		DocumentReceivedDate: &received,
		SheetCount:           3,
		DocumentTypeCode:     "0001",
		DocumentKindCode:     "000101",
		BoxName:              "BOX",
		FillingCode:          "FIL",
	}
}

func TestNewDraftMengumpulkanSeluruhPelanggaranPanjang(t *testing.T) {
	long := strings.Repeat("x", 201)
	input := archivedokumenklaim.DraftInput{
		ClaimNumber:      long,
		SheetCount:       archivedokumenklaim.MaxSheetCount + 1,
		DocumentTypeCode: long,
		DocumentKindCode: long,
		BoxName:          long,
		FillingCode:      long,
	}

	_, err := archivedokumenklaim.NewDraft(input, archivedokumenklaim.Caller{Login: "U"}, "")
	require.Equal(t, []string{
		archivedokumenklaim.FieldClaimNumber,
		archivedokumenklaim.FieldSheetCount,
		archivedokumenklaim.FieldDocumentType,
		archivedokumenklaim.FieldDocumentKind,
		archivedokumenklaim.FieldBoxName,
		archivedokumenklaim.FieldFillingCode,
		archivedokumenklaim.FieldReceivedDate,
	}, violationFields(t, err))
}

// Identitas yang tidak terbaca dijawab tersendiri, bahkan bila isiannya sah.
func TestNewDraftTanpaLoginDitolak(t *testing.T) {
	_, err := archivedokumenklaim.NewDraft(validInput(), archivedokumenklaim.Caller{Login: " "}, "")
	require.ErrorIs(t, err, archivedokumenklaim.ErrCallerUnknown)
}

// Isian yang ikut dari klaim dipotong, bukan ditolak.
func TestNewDraftMemotongIsianDariKlaim(t *testing.T) {
	input := validInput()
	input.InsuredName = strings.Repeat("N", 250)
	input.PolicyNumber = " POL-1 "
	loss := time.Date(2024, 5, 1, 22, 0, 0, 0, time.UTC)
	input.LossDate = &loss

	draft, err := archivedokumenklaim.NewDraft(input,
		archivedokumenklaim.Caller{Login: " USER "}, " 01 ")
	require.NoError(t, err)
	require.Len(t, draft.InsuredName, archivedokumenklaim.MaxNameLength)
	require.Equal(t, "POL-1", draft.PolicyNumber)
	require.Equal(t, "USER", draft.InputUser)
	require.Equal(t, "01", draft.BranchCode)
	require.Equal(t, time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), *draft.LossDate)
	require.True(t, draft.IsNew())
}
