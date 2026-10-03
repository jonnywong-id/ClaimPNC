package sqlstore

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/komite"
)

// step adalah satu kueri yang dijalankan FindTransfer, beserta baris jawabannya.
type step struct {
	name string
	rows func() *sqlmock.Rows
}

const legacyKey = "ASM-FW-GCNMFW-WORK PNC-1"

var lineColumns = []string{
	"CLAIM", "OBJ", "COV", "ACC", "ACCAT", "CUR", "PAY", "GROSS", "PROPOSE", "ACCEPTED",
	"SALVAGE", "SHAREPCT", "SHAREVAL", "RISK", "EXGRATIA", "NOTES", "CAUSE", "CURCODE", "FEE",
	"ADJID", "TOTAL", "LOC", "RISKTYPE", "RISKPCT", "EST", "INTERIM",
}

func lineValues() []any {
	return []any{
		legacyKey, "O1", "C1", "AKS-1", at, "IDR", "1", "100.00", "90.00", "80.00",
		"5.00", "50", "40.00", "1.00", "1", "catatan", "Banjir", "IDR", "2.00",
		"ADJ-1", "100.00", "10", "R", "5", "70.00", "0",
	}
}

func lineRows() *sqlmock.Rows {
	return sqlmock.NewRows(lineColumns).AddRow(toValues(lineValues())...)
}

var entryColumns = []string{"MEMBER", "TIER", "STATUS", "NOTE", "AT", "CASE"}

func entryRows() *sqlmock.Rows {
	return sqlmock.NewRows(entryColumns).AddRow(" Budi ", "2", " Setuju ", " ok ", at, " K-1 ")
}

var committeeColumns = []string{
	"NAME", "TIER", "KIND", "PAY", "NOTE", "VALUE", "SHARE", "DECIDED", "APPROVE", "COUNT", "CREATED",
}

func committeeRows() *sqlmock.Rows {
	return sqlmock.NewRows(committeeColumns).
		AddRow("Budi", "x", "2", "2", "catatan", "1000.00", "50", at, "1", 1, at)
}

func caseRows(key string) func() *sqlmock.Rows {
	return func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"PANEL", "BUSINESS", "KEY"}).AddRow("006", "Fire", key)
	}
}

var claimColumns = []string{
	"DOL", "REG", "LOC", "CHRONO", "STATUS", "RECOMM", "SHARE", "COINS", "CUR", "GRATIA", "COUNT",
	"NO", "POLICY", "INSURED", "BUSINESS", "BRANCH", "SOB", "ROLE", "PANEL", "CURCODE", "PRODKE",
}

func claimRows() *sqlmock.Rows {
	return sqlmock.NewRows(claimColumns).AddRow(
		at, at, "Jakarta", "kronologi", "1147", "setuju", "50", "PT Ko", "IDR", "0", 1,
		"PNC-1", "POL-1", "PT A", "FIRE", "Jakarta", "Direct", "Leader", "006", "IDR", "1")
}

var coverageColumns = []string{
	"OBJ", "COV", "OBJNAME", "COVNAME", "CAUSE", "TSI", "CUR", "CIRCUM", "EXTENT", "LIAB",
	"REMARKS", "DIAG", "INITIAL", "DATE",
}

func coverageRows() *sqlmock.Rows {
	return sqlmock.NewRows(coverageColumns).AddRow(
		"O1", "C1", "Gudang", "Kebakaran", "Api", "500.00", "IDR", "c", "e", "l", "r", "d", "i", at)
}

func spreadingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"OBJ", "COV", "TREATY", "NAME", "PCT"}).
		AddRow("O1", "C1", "10001", "OR", "100")
}

func factorRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"NAME"}).AddRow(" Banjir ").AddRow(" ")
}

func attachmentRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"ID", "NAME", "NOTE", "CAT", "BY", "AT"}).
		AddRow(" A1 ", " foto.png ", " n ", " Foto ", " budi ", at)
}

func policyRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"START", "END"}).AddRow("20260101T000000.000 GMT", "20261231")
}

func coinsuranceRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"LEADER", "NAME", "PCT"}).AddRow(" TRUE ", " PT Ko ", " 40 ")
}

func facOfferRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"NAME", "PCT"}).AddRow(" Reas ", " 10 ")
}

// driverValue adalah nilai baris tiruan sqlmock.
type driverValue = driver.Value

func toValues(values []any) []driverValue {
	out := make([]driverValue, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func legacySteps() []step {
	return []step{
		{"transfer_lines", lineRows},
		{"transfer_members", entryRows},
		{"transfer_committee", committeeRows},
		{"transfer_case", caseRows(legacyKey)},
		{"transfer_claim", claimRows},
		{"transfer_coverages", coverageRows},
		{"transfer_spreading", spreadingRows},
		{"transfer_dominant_factors", factorRows},
		{"transfer_attachments", attachmentRows},
		{"transfer_history_legacy", entryRows},
		{"transfer_policy_dokumen", policyRows},
		{"transfer_coinsurance", coinsuranceRows},
		{"transfer_fac_offer", facOfferRows},
	}
}

func expectSteps(mock sqlmock.Sqlmock, steps []step) {
	for _, s := range steps {
		mock.ExpectQuery(exact(s.name)).WillReturnRows(s.rows())
	}
}

func TestFindTransferReadsEveryPartOfALegacyCase(t *testing.T) {
	db, mock := newMock(t)
	repo := NewTransferRepo(db)

	detail, err := repo.FindTransfer(context.Background(), "  ")
	require.NoError(t, err)
	require.Equal(t, komite.TransferDetail{}, detail)

	expectSteps(mock, legacySteps())
	detail, err = repo.FindTransfer(context.Background(), " K-1 ")
	require.NoError(t, err)

	require.Len(t, detail.Lines, 1)
	line := detail.Lines[0]
	require.Equal(t, "PNC-1", line.ClaimNumber, "awalan kunci Pega dibuang")
	require.Equal(t, mustMoney(t, "80.00"), line.AcceptedValue)
	require.Equal(t, mustMoney(t, "70.00"), line.Estimation)
	require.True(t, line.HasEstimation)
	require.True(t, line.ExGratia)
	require.Equal(t, "Banjir", line.CauseOfLoss)

	require.True(t, detail.HasCommitteeRecord)
	require.Equal(t, 0, detail.Committee.Tier, "jenjang yang tidak terbaca menjadi nol")
	require.Equal(t, "Interim", detail.Committee.Kind)
	require.Equal(t, mustMoney(t, "1000.00"), detail.Committee.ClaimValue)
	require.Equal(t, komite.OutcomeApproved, detail.Committee.Outcome)

	require.Equal(t, "006", detail.GroupPanel)
	require.Equal(t, "Fire", detail.BusinessType)
	require.Equal(t, []komite.CommitteeEntry{{
		CaseID: "K-1", MemberName: "Budi", Tier: 2, Status: "Setuju", Note: "ok", DecidedAt: at,
	}}, detail.Members)

	require.True(t, detail.HasClaim)
	require.Equal(t, "POL-1", detail.Claim.PolicyNumber)
	require.Equal(t, "Leader", detail.Claim.CoinsRole)
	require.Len(t, detail.Coverages, 1)
	require.Equal(t, mustMoney(t, "500.00"), detail.Coverages[0].SumInsured)
	require.Equal(t, []komite.SpreadingShare{{
		ObjectID: "O1", CoverageID: "C1", TreatyType: "10001", TreatyName: "OR", Percent: "100",
	}}, detail.Spreading)
	require.Equal(t, []string{"Banjir"}, detail.DominantFactors)
	require.Equal(t, []komite.Attachment{{
		ID: "A1", Name: "foto.png", Note: "n", Category: "Foto", InputBy: "budi", InputAt: at,
	}}, detail.Attachments)
	require.Len(t, detail.History, 1)

	require.True(t, detail.HasPolicy)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), detail.Policy.Start)
	require.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), detail.Policy.End)
	require.Equal(t, []komite.CoinsuranceShare{{Name: "PT Ko", Leader: true, Percent: "40"}},
		detail.Policy.Coinsurance)
	require.Equal(t, []komite.FacOffer{{ReinsurerName: "Reas", Percent: "10"}}, detail.Policy.FacOffers)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindTransferOfANewCaseUsesItsOwnHeader(t *testing.T) {
	db, mock := newMock(t)
	repo := NewTransferRepo(db)

	noCommittee := func() *sqlmock.Rows {
		return sqlmock.NewRows(committeeColumns).
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, 0, nil)
	}
	noClaim := func() *sqlmock.Rows {
		values := make([]any, len(claimColumns))
		values[10] = 0
		return sqlmock.NewRows(claimColumns).AddRow(toValues(values)...)
	}
	noPolicy := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"START", "END"}) }
	empty := func(columns ...string) func() *sqlmock.Rows {
		return func() *sqlmock.Rows { return sqlmock.NewRows(columns) }
	}

	expectSteps(mock, []step{
		{"transfer_lines", empty(lineColumns...)},
		{"transfer_members", empty(entryColumns...)},
		{"transfer_committee", noCommittee},
		{"transfer_case", caseRows("")},
		{"transfer_case_new", caseRows(legacyKey)},
		{"transfer_claim", noClaim},
		{"transfer_coverages", empty(coverageColumns...)},
		{"transfer_spreading", empty("A", "B", "C", "D", "E")},
		{"transfer_dominant_factors", empty("A")},
		{"transfer_attachments", empty("A", "B", "C", "D", "E", "F")},
		{"transfer_history_new", empty(entryColumns...)},
	})

	detail, err := repo.FindTransfer(context.Background(), "KMTN.26.0001")
	require.NoError(t, err)
	require.False(t, detail.HasCommitteeRecord)
	require.False(t, detail.HasClaim)
	require.False(t, detail.HasPolicy, "tanpa nomor polis, polis tidak dibaca")
	require.Empty(t, detail.Lines)

	// Kasus baru tanpa kunci klaim berhenti sesudah kepala case.
	expectSteps(mock, []step{
		{"transfer_lines", empty(lineColumns...)},
		{"transfer_members", empty(entryColumns...)},
		{"transfer_committee", noCommittee},
		{"transfer_case", caseRows("")},
		{"transfer_case_new", caseRows("")},
	})
	detail, err = repo.FindTransfer(context.Background(), "KMTN-26-0002")
	require.NoError(t, err)
	require.Equal(t, "006", detail.GroupPanel)
	require.False(t, detail.HasClaim)

	// Dokumen tanpa periode: T_GENERAL (DATE jam dinding WIB) menjadi cadangan.
	wib := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	steps := append(legacySteps()[:10],
		step{"transfer_policy_dokumen", noPolicy},
		step{"transfer_policy", func() *sqlmock.Rows {
			return sqlmock.NewRows([]string{"START", "END"}).AddRow(wib, wib)
		}},
		step{"transfer_coinsurance", empty("A", "B", "C")},
		step{"transfer_fac_offer", empty("A", "B")})
	expectSteps(mock, steps)
	detail, err = repo.FindTransfer(context.Background(), "K-1")
	require.NoError(t, err)
	require.True(t, detail.HasPolicy)
	require.Equal(t, time.Date(2025, 12, 31, 17, 0, 0, 0, time.UTC), detail.Policy.Start)

	expectSteps(mock, append(legacySteps()[:10], step{"transfer_policy_dokumen", noPolicy}))
	mock.ExpectQuery(exact("transfer_policy")).WillReturnError(errDB)
	_, err = repo.FindTransfer(context.Background(), "K-1")
	require.ErrorContains(t, err, "membaca T_GENERAL polis")

	// Polis yang tidak ada di dokumen maupun T_GENERAL tetap membaca koasuransi dan fac offer,
	// tetapi HasPolicy palsu.
	steps = append(legacySteps()[:10],
		step{"transfer_policy_dokumen", noPolicy},
		step{"transfer_policy", noPolicy},
		step{"transfer_coinsurance", empty("A", "B", "C")},
		step{"transfer_fac_offer", empty("A", "B")})
	expectSteps(mock, steps)
	detail, err = repo.FindTransfer(context.Background(), "K-1")
	require.NoError(t, err)
	require.False(t, detail.HasPolicy)
	require.True(t, detail.Policy.Start.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindTransferStopsAtTheFirstFailingQuery(t *testing.T) {
	steps := legacySteps()
	for i := range steps {
		t.Run(steps[i].name, func(t *testing.T) {
			db, mock := newMock(t)
			expectSteps(mock, steps[:i])
			mock.ExpectQuery(exact(steps[i].name)).WillReturnError(errDB)

			_, err := NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
			require.ErrorIs(t, err, errDB)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}

	t.Run("transfer_case_new", func(t *testing.T) {
		db, mock := newMock(t)
		expectSteps(mock, steps[:3])
		mock.ExpectQuery(exact("transfer_case")).WillReturnRows(caseRows("")())
		mock.ExpectQuery(exact("transfer_case_new")).WillReturnError(errDB)
		_, err := NewTransferRepo(db).FindTransfer(context.Background(), "KMTN.1")
		require.ErrorContains(t, err, "membaca kepala case komite")
	})
}

func TestFindTransferRejectsUnreadableRows(t *testing.T) {
	bad := func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A"}).AddRow("x") }
	steps := legacySteps()
	for i := range steps {
		if steps[i].name == "transfer_case" || steps[i].name == "transfer_policy_dokumen" {
			continue
		}
		t.Run(steps[i].name, func(t *testing.T) {
			db, mock := newMock(t)
			expectSteps(mock, steps[:i])
			mock.ExpectQuery(exact(steps[i].name)).WillReturnRows(bad())

			_, err := NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
			require.Error(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindTransferReportsIterationErrors(t *testing.T) {
	steps := legacySteps()
	for i := range steps {
		switch steps[i].name {
		case "transfer_case", "transfer_policy_dokumen", "transfer_committee", "transfer_claim":
			continue
		}
		t.Run(steps[i].name, func(t *testing.T) {
			db, mock := newMock(t)
			expectSteps(mock, steps[:i])
			mock.ExpectQuery(exact(steps[i].name)).WillReturnRows(steps[i].rows().RowError(0, errDB))

			_, err := NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
			require.ErrorIs(t, err, errDB)
		})
	}
}

func TestFindTransferRejectsBadMoney(t *testing.T) {
	steps := legacySteps()

	// Setiap kolom uang pada rincian transfer diperiksa satu per satu.
	for _, column := range []int{7, 8, 9, 10, 12, 13, 18, 20, 24, 25} {
		values := lineValues()
		values[column] = "bukan uang"
		db, mock := newMock(t)
		mock.ExpectQuery(exact("transfer_lines")).
			WillReturnRows(sqlmock.NewRows(lineColumns).AddRow(toValues(values)...))
		_, err := NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
		require.ErrorContains(t, err, "membaca baris transfer", lineColumns[column])
	}

	db, mock := newMock(t)
	expectSteps(mock, steps[:2])
	mock.ExpectQuery(exact("transfer_committee")).WillReturnRows(sqlmock.NewRows(committeeColumns).
		AddRow("Budi", "1", "1", "1", "n", "bukan uang", "50", at, "1", 1, at))
	_, err := NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
	require.ErrorContains(t, err, "nilai klaim komite")

	db, mock = newMock(t)
	expectSteps(mock, steps[:5])
	mock.ExpectQuery(exact("transfer_coverages")).WillReturnRows(sqlmock.NewRows(coverageColumns).
		AddRow("O1", "C1", "a", "b", "c", "bukan uang", "IDR", "", "", "", "", "", "", at))
	_, err = NewTransferRepo(db).FindTransfer(context.Background(), "K-1")
	require.ErrorContains(t, err, "TSI coverage")
}

func TestParsePegaTimeAndCaseKind(t *testing.T) {
	require.Equal(t, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), parsePegaTime("20260102T030405"))
	require.True(t, parsePegaTime(" ").IsZero())
	require.True(t, parsePegaTime("bukan tanggal").IsZero())
	require.True(t, isNewCase(" kmtn.1 "))
	require.False(t, isNewCase("K-1"))
}

func TestTransferCheckTables(t *testing.T) {
	db, mock := newMock(t)
	repo := NewTransferRepo(db)

	mock.ExpectQuery(exact("transfer_check_table")).WillReturnRows(sqlmock.NewRows([]string{"X"}))
	require.NoError(t, repo.CheckTables(context.Background()))
	mock.ExpectQuery(exact("transfer_check_table")).WillReturnError(errDB)
	require.ErrorContains(t, repo.CheckTables(context.Background()), "memeriksa tabel rincian transfer")
	require.NoError(t, mock.ExpectationsWereMet())
}
