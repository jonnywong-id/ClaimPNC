package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsalvage"
)

// Uji di berkas ini menjalankan Repo terhadap sqlmock: kueri yang dikirim, argumennya,
// pemetaan baris ke struct, dan setiap cabang galat.

var errBoom = errors.New("basis data mati")

// todayPattern mencocokkan kueri tanggal basis data yang ditulis langsung di kode Go.
const todayPattern = `TO_CHAR\(CURRENT_DATE, 'YYYY-MM-DD'\) FROM DUAL`

// newMock membentuk repo di atas sqlmock dengan pencocok regexp.
func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// exact mengubah teks kueri bernama menjadi pola regexp yang cocok persis.
func exact(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

// tabQuery menyusun permintaan atas daftar mana pun, termasuk yang tidak ditawarkan layar.
func tabQuery(t *testing.T, code, search, login string) inboxsalvage.Query {
	t.Helper()
	tab, known := inboxsalvage.FindAnyTab(code)
	require.Truef(t, known, "daftar %q tidak ada", code)
	q, err := inboxsalvage.NewQueryForTab(
		tab, inboxsalvage.QueryInput{Tab: code, Search: search}, inboxsalvage.Caller{Login: login})
	require.NoError(t, err)
	return q
}

var firstPage = inboxsalvage.Pagination{Page: 1, Size: 20}

// ── Daftar keluarga A dan B ─────────────────────────────────────────────────────

func TestListOutstandingSendsClosedStatusesAndMapsLossDate(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_claim")).
		WithArgs("Resolved-Completed", "Resolved-Rejected", 0, 20).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("PNC-1", "PIC1", "Fire", "2026-09-01", 7).
			AddRow("PNC-2", nil, nil, nil, 7))

	page, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabOutstanding, "", "SITI"), firstPage)
	require.NoError(t, err)
	require.Equal(t, 7, page.Total)
	require.Len(t, page.Items, 2)
	require.Equal(t, inboxsalvage.Row{
		Reference: "PNC-1", ClaimNo: "PNC-1", PIC: "PIC1",
		BusinessName: "Fire", LossDate: "2026-09-01",
	}, page.Items[0])
	require.Empty(t, page.Items[1].ObjectName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOutstandingWithSearchEscapesTheWildcards(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_claim_search")).
		WithArgs("Resolved-Completed", "Resolved-Rejected", `%PNC\_1\%%`, 20, 20).
		WillReturnRows(sqlmock.NewRows(claimColumns))

	page, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabOutstanding, "PNC_1%", "SITI"),
		inboxsalvage.Pagination{Page: 2, Size: 20})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Equal(t, 0, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimObjectMapsTheFourthColumnToObjectName(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_claim_object")).
		WithArgs("3", 0, 20).
		WillReturnRows(sqlmock.NewRows(claimObjectColumns).
			AddRow("PNC-3", "PIC", "Marine", "Kapal", 1))

	page, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabEkonomis, "", "SITI"), firstPage)
	require.NoError(t, err)
	require.Equal(t, "Kapal", page.Items[0].ObjectName)
	require.Empty(t, page.Items[0].LossDate, "keluarga B tidak membawa tanggal kejadian")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimObjectSearchSendsStatusAndPattern(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_claim_object_search")).
		WithArgs("5", "%PNC%", 0, 20).
		WillReturnRows(sqlmock.NewRows(claimObjectColumns))

	_, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabTBA, "PNC", "SITI"), firstPage)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListBuybackUsesItsOwnQueriesWithAndWithoutSearch(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_claim_buyback")).
		WithArgs(0, 20).
		WillReturnRows(sqlmock.NewRows(claimObjectColumns))
	mock.ExpectQuery(exact("list_claim_buyback_search")).
		WithArgs("%X%", 0, 20).
		WillReturnRows(sqlmock.NewRows(claimObjectColumns))

	_, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabBuyback, "", "SITI"), firstPage)
	require.NoError(t, err)
	_, err = repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabBuyback, "X", "SITI"), firstPage)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListClaimReportsQueryScanAndRowsErrors(t *testing.T) {
	q := func(t *testing.T) inboxsalvage.Query {
		return tabQuery(t, inboxsalvage.TabEkonomis, "", "SITI")
	}

	t.Run("kueri gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_claim_object")).WillReturnError(errBoom)

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorIs(t, err, errBoom)
		require.Contains(t, err.Error(), "list_claim_object")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pemindaian gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_claim_object")).
			WillReturnRows(sqlmock.NewRows(claimObjectColumns).
				AddRow("PNC-1", "P", "B", "O", "bukan-angka"))

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorContains(t, err, "memindai baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("hasil gagal dibaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_claim_object")).
			WillReturnRows(sqlmock.NewRows(claimObjectColumns).
				AddRow("PNC-1", "P", "B", "O", 1).RowError(0, errBoom))

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca hasil")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── Daftar keluarga C ───────────────────────────────────────────────────────────

func salvageRow(id, claim, input string, accepted, note any) []driver.Value {
	return []driver.Value{
		id, claim, input, "PIC1", "Besi Tua",
		"Gudang", "2", "1000", "a@b.c", "remark",
		"ACC-1", "3", accepted, "500", note,
		int64(2),
	}
}

func TestListSalvageMapsRowsAndDerivesStatusesFromTheDatabaseDate(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_salvage")).
		WithArgs(PegaWorkKeyPrefix, "3", "%", patternAll, patternNever, 0, 20).
		WillReturnRows(sqlmock.NewRows(salvageColumns).
			AddRow(salvageRow("11", "PNC-1", "2026-09-15 08:00:00", "5100", nil)...).
			AddRow(salvageRow("12", "PNC-2", "", "0", "minta turun")...))
	mock.ExpectQuery(todayPattern).
		WillReturnRows(sqlmock.NewRows([]string{"TODAY"}).AddRow("2026-09-25"))

	page, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabChecker, "", "SITI"), firstPage)
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Len(t, page.Items, 2)

	first := page.Items[0]
	require.Equal(t, "11", first.Reference)
	require.Equal(t, "11", first.SalvageID)
	require.Equal(t, "PNC-1", first.ClaimNo)
	require.Equal(t, "2026-09-15", first.InputDate, "bagian waktu dipotong")
	require.Equal(t, inboxsalvage.AuctionSold, first.AuctionStatus)
	require.Equal(t, inboxsalvage.SubmissionNew, first.SubmissionType)
	require.Equal(t, "10 day", first.Aging)
	require.Equal(t, "ACC-1", first.AcceptanceNo)
	require.Equal(t, "3", first.TransferStatus)
	require.Equal(t, "500", first.RequestValue)
	require.Equal(t, "Gudang", first.SalvageLocation)
	require.Equal(t, "2", first.Quantity)
	require.Equal(t, "1000", first.EstimateValue)
	require.Empty(t, first.Note, "kolom Catatan selalu kosong")

	second := page.Items[1]
	require.Equal(t, inboxsalvage.AuctionUnsold, second.AuctionStatus)
	require.Equal(t, inboxsalvage.SubmissionRequest, second.SubmissionType)
	require.Equal(t, "minta turun", second.RequestNote)
	require.Empty(t, second.Aging, "tanggal kosong menghasilkan aging kosong")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tab Request Balai Lelang menyaring PIC = pemanggil, dan karakter wildcard pada
// identitasnya dilepaskan.
func TestListSalvageFiltersOwnerAndSearchesBothColumnsWhereTheTabSaysSo(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_salvage")).
		WithArgs(PegaWorkKeyPrefix, "7", `SITI\_A`, "%PNC%", patternNever, 0, 20).
		WillReturnRows(sqlmock.NewRows(salvageColumns))
	mock.ExpectQuery(todayPattern).
		WillReturnRows(sqlmock.NewRows([]string{"TODAY"}).AddRow("2026-09-25"))

	page, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabRequestBalai, "PNC", "SITI_A"), firstPage)
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSalvageWithoutTransferFilterSendsTheMatchAllPattern(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("list_salvage")).
		WithArgs(PegaWorkKeyPrefix, patternAll, patternAll, patternAll, patternNever, 0, 20).
		WillReturnRows(sqlmock.NewRows(salvageColumns))
	mock.ExpectQuery(todayPattern).
		WillReturnRows(sqlmock.NewRows([]string{"TODAY"}).AddRow("2026-09-25"))

	_, err := repo.List(context.Background(),
		tabQuery(t, inboxsalvage.TabHistori, "", "SITI"), firstPage)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSalvageReportsEveryFailure(t *testing.T) {
	q := func(t *testing.T) inboxsalvage.Query {
		return tabQuery(t, inboxsalvage.TabBalaiLelang, "", "SITI")
	}

	t.Run("kueri gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_salvage")).WillReturnError(errBoom)

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "list_salvage")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("tanggal basis data gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_salvage")).WillReturnRows(sqlmock.NewRows(salvageColumns))
		mock.ExpectQuery(todayPattern).WillReturnError(errBoom)

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca tanggal basis data")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pemindaian gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		bad := salvageRow("1", "PNC-1", "2026-09-01", "0", nil)
		bad[15] = "bukan-angka"
		mock.ExpectQuery(exact("list_salvage")).
			WillReturnRows(sqlmock.NewRows(salvageColumns).AddRow(bad...))
		mock.ExpectQuery(todayPattern).
			WillReturnRows(sqlmock.NewRows([]string{"TODAY"}).AddRow("2026-09-25"))

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorContains(t, err, "memindai baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("hasil gagal dibaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("list_salvage")).
			WillReturnRows(sqlmock.NewRows(salvageColumns).
				AddRow(salvageRow("1", "PNC-1", "2026-09-01", "0", nil)...).
				RowError(0, errBoom))
		mock.ExpectQuery(todayPattern).
			WillReturnRows(sqlmock.NewRows([]string{"TODAY"}).AddRow("2026-09-25"))

		_, err := repo.List(context.Background(), q(t), firstPage)
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca hasil")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSearchPatternsFollowTheTabRules(t *testing.T) {
	exactTab := inboxsalvage.Tab{SearchExact: true}
	exactPIC := inboxsalvage.Tab{SearchExact: true, SearchByPIC: true}
	partialPIC := inboxsalvage.Tab{SearchByPIC: true}

	claim, pic := searchPatterns(inboxsalvage.Query{})
	require.Equal(t, patternAll, claim)
	require.Equal(t, patternNever, pic, "tanpa pencarian, cabang PIC dimatikan")

	claim, pic = searchPatterns(inboxsalvage.Query{Tab: exactTab, Search: "A%"})
	require.Equal(t, `A\%`, claim)
	require.Equal(t, patternNever, pic)

	claim, pic = searchPatterns(inboxsalvage.Query{Tab: exactPIC, Search: "A"})
	require.Equal(t, "A", claim)
	require.Equal(t, "A", pic)

	claim, pic = searchPatterns(inboxsalvage.Query{Tab: partialPIC, Search: "A"})
	require.Equal(t, "%A%", claim)
	require.Equal(t, "%A%", pic)

	claim, pic = searchPatterns(inboxsalvage.Query{Search: "A"})
	require.Equal(t, "%A%", claim)
	require.Equal(t, patternNever, pic)
}

// ── Panel Detail Salvage ────────────────────────────────────────────────────────

var detailHeaderColumns = []string{
	"ID", "CLAIM", "INPUT", "TYPE", "QTY", "EST", "LOC", "GA", "STS",
	"ACCDATE", "ACCNO", "REMARK", "CUR", "OBJNAME", "OBJID", "COVNAME", "COVID",
	"ACCVAL", "EMAIL", "OFFER", "WINNER", "AUCTION", "SURNAME", "SURPHONE",
	"SUREMAIL", "JABO", "LEGACY", "BIZ",
}

func detailHeaderRow(id, claim, business any) []driver.Value {
	return []driver.Value{
		id, claim, "2026-09-01 10:00:00", "Besi Tua", "3", "1000", "Gudang",
		"2026-09-02 00:00:00", "1", "2026-09-03", "ACC-9", "catatan", "IDR",
		"Panel", "OBJ-1", "Property", "COV-1", "2500", "a@b.c", "3000", "Budi",
		"2026-09-04", "Surveyor", "0812", "s@b.c", "1", "0", business,
	}
}

var itemColumns = []string{
	"NAME", "COUNT", "UNIT", "TOTAL", "SOLD", "WINNER", "ACCNO", "ACCVAL", "REMARK",
}

var historyColumns = []string{"INPUT", "NO", "PIC", "MIN", "STS", "HASACC", "ID"}

func TestDetailReadsHeaderItemsAndHistory(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("detail_header")).
		WithArgs(PegaWorkKeyPrefix, "77").
		WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
			AddRow(detailHeaderRow("77", "PNC-1", "Fire")...))
	mock.ExpectQuery(exact("detail_items")).
		WithArgs("PNC-1", "77").
		WillReturnRows(sqlmock.NewRows(itemColumns).
			AddRow("Besi", 2, "kg", "10", "1", "Budi", "ACC", "9", "rem").
			AddRow("Kayu", 1, nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectQuery(exact("salvage_history_of_claim")).
		WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows(historyColumns).
			AddRow("2026-09-01 08:00:00", " PNC-1 ", " PIC ", " 100 ", "2", "1", " 77 "))

	detail, err := repo.Detail(context.Background(), " 77 ")
	require.NoError(t, err)

	require.True(t, detail.HasSubmission)
	require.Equal(t, "77", detail.SalvageID)
	require.Equal(t, "PNC-1", detail.ClaimNo)
	require.Equal(t, "Fire", detail.BusinessName)
	require.Equal(t, "2026-09-01", detail.InputDate)
	require.Equal(t, "2026-09-02", detail.TransferGADate)
	require.Equal(t, "1", detail.TransferStatus)
	require.Equal(t, inboxsalvage.PositionLabelOf("1"), detail.Position)
	require.Equal(t, "2026-09-03", detail.AcceptanceDate)
	require.Equal(t, "2026-09-04", detail.AuctionDate)
	require.Equal(t, "OBJ-1", detail.ObjectID)
	require.Equal(t, "COV-1", detail.CoverageID)
	require.Equal(t, "Surveyor", detail.SurveyorName)
	require.True(t, detail.InJabodetabek)
	require.False(t, detail.LegacyBeforeJuly2023, "\"0\" bukan penanda benar")

	require.Len(t, detail.Items, 2)
	require.Equal(t, inboxsalvage.DetailBarang{
		Name: "Besi", Count: 2, Unit: "kg", TotalValue: "10",
		SoldStatus: "Terjual", WinnerName: "Budi", AcceptanceNo: "ACC",
		AcceptedValue: "9", Remark: "rem",
	}, detail.Items[0])
	require.Equal(t, "Belum terjual", detail.Items[1].SoldStatus)

	require.Equal(t, []inboxsalvage.HistoryRow{{
		SalvageID: "77", InputDate: "2026-09-01", ClaimNo: "PNC-1", PIC: "PIC",
		MinimumValue: "100", Position: inboxsalvage.HistoryAccepted,
	}}, detail.History)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailWithEmptyIDIsNotFoundWithoutTouchingTheDatabase(t *testing.T) {
	repo, mock := newMock(t)

	_, err := repo.Detail(context.Background(), "   ")
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailReportsEveryFailure(t *testing.T) {
	t.Run("kepala tidak ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns))

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kepala gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).WillReturnError(errBoom)

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "detail_header")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("barang gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
				AddRow(detailHeaderRow("1", "PNC-1", "Fire")...))
		mock.ExpectQuery(exact("detail_items")).WillReturnError(errBoom)

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "detail_items")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("barang gagal dipindai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
				AddRow(detailHeaderRow("1", "PNC-1", "Fire")...))
		mock.ExpectQuery(exact("detail_items")).
			WillReturnRows(sqlmock.NewRows(itemColumns).
				AddRow("Besi", "bukan-angka", "kg", "1", "1", "", "", "", ""))

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorContains(t, err, "memindai baris")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("barang gagal dibaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
				AddRow(detailHeaderRow("1", "PNC-1", "Fire")...))
		mock.ExpectQuery(exact("detail_items")).
			WillReturnRows(sqlmock.NewRows(itemColumns).
				AddRow("Besi", 1, "kg", "1", "1", "", "", "", "").RowError(0, errBoom))

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca hasil")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("riwayat gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
				AddRow(detailHeaderRow("1", "PNC-1", "Fire")...))
		mock.ExpectQuery(exact("detail_items")).WillReturnRows(sqlmock.NewRows(itemColumns))
		mock.ExpectQuery(exact("salvage_history_of_claim")).WillReturnError(errBoom)

		_, err := repo.Detail(context.Background(), "1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca riwayat salvage klaim")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestHistoryReportsScanAndRowsErrors(t *testing.T) {
	t.Run("pemindaian gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("salvage_history_of_claim")).
			WillReturnRows(sqlmock.NewRows([]string{"SATU"}).AddRow("x"))

		_, err := repo.historyOfClaim(context.Background(), "PNC-1")
		require.ErrorContains(t, err, "memindai riwayat salvage")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("hasil gagal dibaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("salvage_history_of_claim")).
			WillReturnRows(sqlmock.NewRows(historyColumns).
				AddRow("2026-09-01", "PNC-1", "P", "1", "1", "0", "1").RowError(0, errBoom))

		_, err := repo.historyOfClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── Panel yang dibuka dari baris klaim ──────────────────────────────────────────

var claimHeaderColumns = []string{"NO", "PIC", "BIZ", "LOSS"}

func expectClaimHeader(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(exact("claim_header")).
		WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows(claimHeaderColumns).
			AddRow(" PNC-1 ", " PICKLAIM ", " Fire ", "2026-08-01 00:00:00"))
}

func expectEmptyHistory(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(exact("salvage_history_of_claim")).
		WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows(historyColumns))
}

func TestDetailByClaimWithoutSubmissionReturnsTheClaimOnly(t *testing.T) {
	repo, mock := newMock(t)

	expectClaimHeader(mock)
	expectEmptyHistory(mock)
	mock.ExpectQuery(exact("latest_salvage_of_claim")).
		WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(nil))

	detail, err := repo.DetailByClaim(context.Background(), " PNC-1 ")
	require.NoError(t, err)
	require.False(t, detail.HasSubmission)
	require.Equal(t, "PNC-1", detail.ClaimNo)
	require.Equal(t, "PICKLAIM", detail.PIC)
	require.Equal(t, "Fire", detail.BusinessName)
	require.Equal(t, "2026-08-01", detail.LossDate)
	require.Empty(t, detail.Items)
	require.NotNil(t, detail.Items)
	require.Empty(t, detail.History)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailByClaimTreatsNoRowsFromTheAggregateAsNoSubmission(t *testing.T) {
	repo, mock := newMock(t)

	expectClaimHeader(mock)
	expectEmptyHistory(mock)
	mock.ExpectQuery(exact("latest_salvage_of_claim")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}))

	detail, err := repo.DetailByClaim(context.Background(), "PNC-1")
	require.NoError(t, err)
	require.False(t, detail.HasSubmission)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailByClaimWithSubmissionKeepsTheClaimFields(t *testing.T) {
	repo, mock := newMock(t)

	expectClaimHeader(mock)
	mock.ExpectQuery(exact("salvage_history_of_claim")).
		WithArgs("PNC-1").
		WillReturnRows(sqlmock.NewRows(historyColumns).
			AddRow("2026-09-01", "PNC-1", "P", "1", "6", "0", "9"))
	mock.ExpectQuery(exact("latest_salvage_of_claim")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow(" 9 "))
	mock.ExpectQuery(exact("detail_header")).
		WithArgs(PegaWorkKeyPrefix, "9").
		WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
			AddRow(detailHeaderRow("9", "PNC-1", nil)...))
	mock.ExpectQuery(exact("detail_items")).
		WithArgs("PNC-1", "9").
		WillReturnRows(sqlmock.NewRows(itemColumns).
			AddRow("Besi", 1, "kg", "1", "0", "", "", "", ""))

	detail, err := repo.DetailByClaim(context.Background(), "PNC-1")
	require.NoError(t, err)
	require.True(t, detail.HasSubmission)
	require.Equal(t, "9", detail.SalvageID)
	require.Equal(t, "PICKLAIM", detail.PIC, "PIC berasal dari klaim")
	require.Equal(t, "2026-08-01", detail.LossDate)
	require.Equal(t, "Fire", detail.BusinessName, "nama bisnis kosong diambil dari klaim")
	require.Len(t, detail.Items, 1)
	require.Equal(t, "Tidak terjual", detail.Items[0].SoldStatus)
	require.Len(t, detail.History, 1)
	require.Equal(t, inboxsalvage.HistoryWaived, detail.History[0].Position)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pengajuan yang tercatat sebagai terakhir tetapi tidak terbaca kepalanya tidak
// menggagalkan panel: klaimnya tetap ditampilkan.
func TestDetailByClaimFallsBackToTheClaimWhenTheSubmissionVanished(t *testing.T) {
	repo, mock := newMock(t)

	expectClaimHeader(mock)
	expectEmptyHistory(mock)
	mock.ExpectQuery(exact("latest_salvage_of_claim")).
		WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("9"))
	mock.ExpectQuery(exact("detail_header")).
		WillReturnRows(sqlmock.NewRows(detailHeaderColumns))

	detail, err := repo.DetailByClaim(context.Background(), "PNC-1")
	require.NoError(t, err)
	require.False(t, detail.HasSubmission)
	require.Equal(t, "PNC-1", detail.ClaimNo)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailByClaimReportsEveryFailure(t *testing.T) {
	t.Run("nomor kosong", func(t *testing.T) {
		repo, mock := newMock(t)
		_, err := repo.DetailByClaim(context.Background(), "  ")
		require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("klaim tidak ada", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("claim_header")).
			WillReturnRows(sqlmock.NewRows(claimHeaderColumns))

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("klaim gagal dibaca", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("claim_header")).WillReturnError(errBoom)

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "membaca klaim")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("riwayat gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		expectClaimHeader(mock)
		mock.ExpectQuery(exact("salvage_history_of_claim")).WillReturnError(errBoom)

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pengajuan terakhir gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		expectClaimHeader(mock)
		expectEmptyHistory(mock)
		mock.ExpectQuery(exact("latest_salvage_of_claim")).WillReturnError(errBoom)

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "mencari pengajuan terakhir")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kepala pengajuan gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		expectClaimHeader(mock)
		expectEmptyHistory(mock)
		mock.ExpectQuery(exact("latest_salvage_of_claim")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("9"))
		mock.ExpectQuery(exact("detail_header")).WillReturnError(errBoom)

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("barang gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		expectClaimHeader(mock)
		expectEmptyHistory(mock)
		mock.ExpectQuery(exact("latest_salvage_of_claim")).
			WillReturnRows(sqlmock.NewRows([]string{"ID"}).AddRow("9"))
		mock.ExpectQuery(exact("detail_header")).
			WillReturnRows(sqlmock.NewRows(detailHeaderColumns).
				AddRow(detailHeaderRow("9", "PNC-1", "Fire")...))
		mock.ExpectQuery(exact("detail_items")).WillReturnError(errBoom)

		_, err := repo.DetailByClaim(context.Background(), "PNC-1")
		require.ErrorIs(t, err, errBoom)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── Tabel ringkas ───────────────────────────────────────────────────────────────

func TestCountsRunsOneQueryPerVisibleRowWithTheRightBinds(t *testing.T) {
	repo, mock := newMock(t)

	total := func(n int) *sqlmock.Rows { return sqlmock.NewRows([]string{"TOTAL"}).AddRow(n) }

	// Urutan mengikuti inboxsalvage.CountRows().
	//
	// Baris PERTAMA — Outstanding — memakai kueri yang BERBEDA: penanda salvage yang sama
	// dengan daftarnya, DITAMBAH penyaring status pekerjaan. Hanya baris ini yang
	// menyaring status pekerjaan. Lihat inboxsalvage.CountRow.ExcludesClosedWork.
	mock.ExpectQuery(exact("count_claim_outstanding")).
		WithArgs("3", "5", "Resolved-Completed", "Resolved-Rejected").
		WillReturnRows(total(1))
	mock.ExpectQuery(exact("count_claim_status")).WithArgs("3", "3").WillReturnRows(total(2))
	mock.ExpectQuery(exact("count_salvage_status")).WithArgs("4", "4", patternAll).
		WillReturnRows(total(3))
	mock.ExpectQuery(exact("count_salvage_status")).WithArgs("1", "1", patternAll).
		WillReturnRows(total(4))
	mock.ExpectQuery(exact("count_claim_status")).WithArgs("4", "4").WillReturnRows(total(5))
	mock.ExpectQuery(exact("count_claim_status")).WithArgs("5", "5").WillReturnRows(total(6))
	mock.ExpectQuery(exact("count_claim_status")).WithArgs("1", "1").WillReturnRows(total(7))
	mock.ExpectQuery(exact("count_claim_buyback")).WithoutArgs().WillReturnRows(total(8))
	mock.ExpectQuery(exact("count_salvage_status")).WithArgs("1", "6", patternAll).
		WillReturnRows(total(9))

	counts, err := repo.Counts(context.Background(), inboxsalvage.Caller{Login: "SITI"})
	require.NoError(t, err)

	definitions := inboxsalvage.CountRows()
	require.Len(t, counts, len(definitions))
	for index, count := range counts {
		require.Equal(t, definitions[index].Label, count.Label)
		require.Equal(t, definitions[index].Tab, count.Tab)
		require.Equal(t, index+1, count.Total)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountsStopsAtTheFirstFailure(t *testing.T) {
	repo, mock := newMock(t)
	// Baris pertama yang dijalankan adalah Outstanding, dan kuerinya tersendiri.
	mock.ExpectQuery(exact("count_claim_outstanding")).WillReturnError(errBoom)

	_, err := repo.Counts(context.Background(), inboxsalvage.Caller{Login: "SITI"})
	require.ErrorIs(t, err, errBoom)
	require.ErrorContains(t, err, "count_claim_outstanding")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Baris pencacah yang tidak digambar tetap harus menyusun kuerinya dengan benar.
func TestHiddenCounterRowsBuildTheirOwnQueries(t *testing.T) {
	var request, unsold inboxsalvage.CountRow
	for _, row := range inboxsalvage.AllCountRows() {
		switch row.Label {
		case "Request Balai Lelang":
			request = row
		case "Tidak Terjual":
			unsold = row
		}
	}

	repo, mock := newMock(t)
	mock.ExpectQuery(exact("count_salvage_status")).WithArgs("7", "7", `SITI\%`).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(4))
	mock.ExpectQuery(exact("count_salvage_detail_unsold")).WithArgs(UnsoldDetailStatus).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(5))

	got, err := repo.countOne(context.Background(), request, inboxsalvage.Caller{Login: "SITI%"})
	require.NoError(t, err)
	require.Equal(t, 4, got)

	got, err = repo.countOne(context.Background(), unsold, inboxsalvage.Caller{Login: "SITI"})
	require.NoError(t, err)
	require.Equal(t, 5, got)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLastOrFirst(t *testing.T) {
	require.Equal(t, "", lastOrFirst(nil))
	require.Equal(t, "3", lastOrFirst([]string{"3"}))
	require.Equal(t, "5", lastOrFirst([]string{"3", "5"}))
}

// ── Penyimpanan ─────────────────────────────────────────────────────────────────

func newForm(mode inboxsalvage.FormMode, id string, items ...inboxsalvage.DetailItem) inboxsalvage.Form {
	return inboxsalvage.Form{
		Mode:           mode,
		SalvageID:      id,
		ClaimNo:        "PNC-1",
		ObjectName:     "Panel",
		CoverageName:   "PAR",
		SalvageType:    "Besi Tua",
		InputDate:      "2026-09-25",
		Quantity:       "2,5",
		InJabodetabek:  true,
		TransferStatus: inboxsalvage.TransferStatusOnSubmit,
		Caller:         inboxsalvage.Caller{Login: "SITI"},
		Items:          items,
	}
}

// insertBinds adalah ke-22 bind yang diharapkan untuk newForm.
func insertBinds(id string) []driver.Value {
	return []driver.Value{
		"PNC-1", "2026-09-25", "Besi Tua", "2.5", nil, nil, "3", nil, "SITI", id,
		nil, "Panel", nil, "PAR", nil, nil, nil, nil, nil, nil, nil, "1",
	}
}

func TestCreateInsertsTheSubmissionAndItsItemsInOneTransaction(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(exact("next_salvage_id")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(int64(42)))
	mock.ExpectExec(exact("insert_salvage")).
		WithArgs(insertBinds("42")...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(exact("count_salvage_detail_for")).
		WithArgs("42", "PNC-1").
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(2))
	mock.ExpectExec(exact("insert_salvage_detail")).
		WithArgs("42", "PNC-1", "PNC-1/42/3", "Besi", "kg", "catatan", "1.5", "3",
			"catatan", NewDetailStatus, EmptyAcceptanceNo).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exact("insert_salvage_detail")).
		WithArgs("42", "PNC-1", "PNC-1/42/4", "Kayu", nil, nil, nil, "4",
			nil, NewDetailStatus, EmptyAcceptanceNo).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeInsert, "",
		inboxsalvage.DetailItem{Name: "Besi", Quantity: "1,5", Unit: "kg", Remarks: "catatan"},
		inboxsalvage.DetailItem{Name: "Kayu"},
	))
	require.NoError(t, err)
	require.Equal(t, "42", id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUpdateKeepsTheSalvageIDAndSkipsItemsWhenThereAreNone(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(exact("update_salvage")).
		WithArgs(insertBinds("17")...).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	id, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeUpdate, "17"))
	require.NoError(t, err)
	require.Equal(t, "17", id)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Pembaruan yang tidak menyentuh satu baris pun berarti pengajuannya tidak ada.
func TestCreateUpdateTouchingNoRowIsNotFoundAndRollsBack(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectExec(exact("update_salvage")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	_, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeUpdate, "17"))
	require.ErrorIs(t, err, inboxsalvage.ErrRowNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateRollsBackOnEveryFailure(t *testing.T) {
	item := inboxsalvage.DetailItem{Name: "Besi"}

	t.Run("transaksi gagal dimulai", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errBoom)

		_, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeInsert, ""))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "memulai transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nomor berikutnya gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("next_salvage_id")).WillReturnError(errBoom)
		mock.ExpectRollback()

		_, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeInsert, ""))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "next_salvage_id")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sisip gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("next_salvage_id")).
			WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(int64(1)))
		mock.ExpectExec(exact("insert_salvage")).WillReturnError(errBoom)
		mock.ExpectRollback()

		_, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeInsert, ""))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "insert_salvage")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("hitung detail gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("next_salvage_id")).
			WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(int64(1)))
		mock.ExpectExec(exact("insert_salvage")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(exact("count_salvage_detail_for")).WillReturnError(errBoom)
		mock.ExpectRollback()

		_, err := repo.Create(context.Background(),
			newForm(inboxsalvage.FormModeInsert, "", item))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "count_salvage_detail_for")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sisip detail gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(exact("next_salvage_id")).
			WillReturnRows(sqlmock.NewRows([]string{"NEXT"}).AddRow(int64(1)))
		mock.ExpectExec(exact("insert_salvage")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery(exact("count_salvage_detail_for")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(0))
		mock.ExpectExec(exact("insert_salvage_detail")).WillReturnError(errBoom)
		mock.ExpectRollback()

		_, err := repo.Create(context.Background(),
			newForm(inboxsalvage.FormModeInsert, "", item))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "insert_salvage_detail baris 1")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectExec(exact("update_salvage")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errBoom)

		_, err := repo.Create(context.Background(), newForm(inboxsalvage.FormModeUpdate, "5"))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "menyimpan transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// ── Pemeriksaan kesiapan dan pembantu ───────────────────────────────────────────

func TestReadyReturnsTheNumberOfColumnsFound(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("check_salvage_columns")).
		WillReturnRows(sqlmock.NewRows([]string{"FOUND"}).AddRow(9))

	found, err := repo.Ready(context.Background())
	require.NoError(t, err)
	require.Equal(t, 9, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReadyReportsTheFailure(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("check_salvage_columns")).WillReturnError(sql.ErrConnDone)

	_, err := repo.Ready(context.Background())
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.ErrorContains(t, err, "check_salvage_columns")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestValueHelpers(t *testing.T) {
	require.Equal(t, "1", jabodetabekFlag(true))
	require.Equal(t, "0", jabodetabekFlag(false))

	require.Nil(t, nullIfEmpty("  "))
	require.Equal(t, "x", nullIfEmpty("x"))

	require.Nil(t, numberOrNull(" "))
	require.Equal(t, "1.25", numberOrNull(" 1,25 "))

	require.Equal(t, "2026-09-01", dateOnly(" 2026-09-01 10:00:00 "))
	require.Equal(t, "2026-09", dateOnly("2026-09"))

	require.True(t, flagIsSet(" 1 "))
	require.False(t, flagIsSet("0"))

	require.Equal(t, `a\\b\%c\_d`, escapeLike(`a\b%c_d`))
}

// Nama kueri yang tidak ada adalah cacat pemrograman yang harus terlihat seketika.
func TestUnknownQueryNamePanics(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxsalvage/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { query("tidak_ada") })
}
