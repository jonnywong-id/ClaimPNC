package sqlstore

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
)

// Uji di berkas ini memakai go-sqlmock: yang diperiksa adalah kueri yang dikirim, argumen
// yang diikat, dan cara baris hasil dipetakan — tanpa Oracle.

var errDB = errors.New("basis data rusak")

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

var day = time.Date(2026, time.January, 15, 13, 0, 0, 0, time.UTC)

func tabOf(t *testing.T, code string) inboxpladla.Tab {
	t.Helper()
	tab, found := inboxpladla.FindTab(code)
	require.True(t, found)
	return tab
}

func queryOf(t *testing.T, code, search string) inboxpladla.Query {
	t.Helper()
	return inboxpladla.Query{
		Tab:            tabOf(t, code),
		Search:         search,
		Caller:         inboxpladla.Caller{Login: "REAS"},
		ReinsurerCodes: []string{"R100"},
	}
}

var scope = inboxpladla.DetailScope{
	ClaimKey: " KEY-1 ", Login: " REAS ", ReinsurerCodes: []string{"R100"},
}

func TestReinsurerCodesMapsAndTrims(t *testing.T) {
	repo, mock := newMock(t)

	codes, err := repo.ReinsurerCodes(context.Background(), "  ")
	require.NoError(t, err)
	require.Nil(t, codes, "login kosong tidak menyentuh basis data")

	mock.ExpectQuery(exact("reinsurer_codes")).WithArgs("REAS").
		WillReturnRows(sqlmock.NewRows([]string{"CODE"}).
			AddRow(" R901 ").AddRow(nil).AddRow("R900"))

	codes, err = repo.ReinsurerCodes(context.Background(), " REAS ")
	require.NoError(t, err)
	require.Equal(t, []string{"R901", "R900"}, codes)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReinsurerCodesFailures(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(exact("reinsurer_codes")).WillReturnError(errDB)
	_, err := repo.ReinsurerCodes(context.Background(), "REAS")
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("reinsurer_codes")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("x", "y"))
	_, err = repo.ReinsurerCodes(context.Background(), "REAS")
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("reinsurer_codes")).
		WillReturnRows(sqlmock.NewRows([]string{"CODE"}).AddRow("R1").RowError(0, errDB))
	_, err = repo.ReinsurerCodes(context.Background(), "REAS")
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func listRows() *sqlmock.Rows {
	return sqlmock.NewRows(listColumns).AddRow(
		" KEY-1 ", " PNC-1 ", " POL-1 ", " PT A ", " FIRE ",
		day, nil, " BUDI ", " 1147 ", " Register ", " PLA/1 ", " catatan ", 7,
	)
}

func TestListBindsLoginSearchAndPaginationPerTab(t *testing.T) {
	cases := []struct {
		tab  string
		name string
		args []driver.Value
	}{
		{inboxpladla.TabPLA, "list_pla", []driver.Value{"REAS", nil, nil,
			"REAS", "REAS", 0, 10}},
		{inboxpladla.TabPLADLA, "list_dla", []driver.Value{"REAS", nil, nil,
			"REAS", 0, 10}},
		{inboxpladla.TabClose, "list_close", []driver.Value{"REAS", nil, nil,
			"REAS", 0, 10}},
	}
	for _, tc := range cases {
		t.Run(tc.tab, func(t *testing.T) {
			repo, mock := newMock(t)
			mock.ExpectQuery(exact(tc.name)).WithArgs(tc.args...).WillReturnRows(listRows())

			page, err := repo.List(context.Background(), queryOf(t, tc.tab, ""),
				inboxpladla.Pagination{})
			require.NoError(t, err)
			require.Equal(t, 7, page.Total)
			require.Equal(t, inboxpladla.Pagination{Page: 1, Size: 10}, page.Pagination)
			require.Equal(t, inboxpladla.Row{
				ClaimKey: "KEY-1", ClaimNo: "PNC-1", PolicyNo: "POL-1", Insured: "PT A",
				BusinessName: "FIRE", RegisterDate: "2026-01-15", LossDate: "",
				PICTeknik: "BUDI", StatusCode: "1147", StatusLabel: "Register",
				AdviceNo: "PLA/1", CloseNote: "catatan",
			}, page.Items[0])
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCommunicationListsBindTheStatusAndAnEscapedSearch(t *testing.T) {
	repo, mock := newMock(t)
	q := queryOf(t, inboxpladla.TabAnswered, "a%b_c\\")
	flag := "1"
	pattern := `%A\%B\_C\\%`

	mock.ExpectQuery(exact("list_komunikasi_sender")).
		WithArgs("REAS", flag, pattern, q.Tab.CommunicationStatus, "REAS", 20, 20).
		WillReturnRows(sqlmock.NewRows(listColumns))

	page, err := repo.List(context.Background(), q, inboxpladla.Pagination{Page: 2, Size: 20})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.Equal(t, 0, page.Total)

	mock.ExpectQuery(exact("list_komunikasi_recipient")).
		WillReturnRows(sqlmock.NewRows(listColumns))
	_, err = repo.List(context.Background(), queryOf(t, inboxpladla.TabInbound, ""),
		inboxpladla.Pagination{})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAndCountsRefuseAnUnknownTab(t *testing.T) {
	repo, _ := newMock(t)
	q := inboxpladla.Query{Tab: inboxpladla.Tab{Code: "xol"}}

	_, err := repo.List(context.Background(), q, inboxpladla.Pagination{})
	require.ErrorContains(t, err, `daftar "xol" tidak dikenal`)

	_, err = repo.Counts(context.Background(), q)
	require.ErrorContains(t, err, `daftar "xol" tidak dikenal`)
}

func TestListFailures(t *testing.T) {
	repo, mock := newMock(t)
	q := queryOf(t, inboxpladla.TabPLA, "")

	mock.ExpectQuery(exact("list_pla")).WillReturnError(errDB)
	_, err := repo.List(context.Background(), q, inboxpladla.Pagination{})
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("list_pla")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.List(context.Background(), q, inboxpladla.Pagination{})
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("list_pla")).WillReturnRows(listRows().RowError(0, errDB))
	_, err = repo.List(context.Background(), q, inboxpladla.Pagination{})
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountsBindTheFilterWithoutPagination(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("count_pla")).
		WithArgs(nil, nil, "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows(countColumns).AddRow(" 1147 ", " Register ", 3))

	counts, err := repo.Counts(context.Background(), queryOf(t, inboxpladla.TabPLA, ""))
	require.NoError(t, err)
	require.Equal(t, []inboxpladla.StatusCount{{Code: "1147", Label: "Register", Total: 3}}, counts)

	q := queryOf(t, inboxpladla.TabInbound, "")
	mock.ExpectQuery(exact("count_komunikasi_recipient")).
		WithArgs(nil, nil, q.Tab.CommunicationStatus, "REAS").
		WillReturnRows(sqlmock.NewRows(countColumns))
	counts, err = repo.Counts(context.Background(), q)
	require.NoError(t, err)
	require.Empty(t, counts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountsFailures(t *testing.T) {
	repo, mock := newMock(t)
	q := queryOf(t, inboxpladla.TabClose, "")

	mock.ExpectQuery(exact("count_close")).WillReturnError(errDB)
	_, err := repo.Counts(context.Background(), q)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("count_close")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Counts(context.Background(), q)
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("count_close")).
		WillReturnRows(sqlmock.NewRows(countColumns).AddRow("1", "x", 1).RowError(0, errDB))
	_, err = repo.Counts(context.Background(), q)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimHeaderMapsTheRowAndHidesForeignClaims(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

	mock.ExpectQuery(exact("detail_claim_header")).
		WithArgs("KEY-1", "REAS", "REAS", "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(
			" KEY-1 ", " PNC-1 ", " POL-1 ", " PT A ", " FIRE ",
			day, day, " BUDI ", " 1147 ", " Register "))

	header, err := repo.ClaimHeader(context.Background(), scope)
	require.NoError(t, err)
	require.Equal(t, inboxpladla.ClaimHeader{
		ClaimKey: "KEY-1", ClaimNo: "PNC-1", PolicyNo: "POL-1", Insured: "PT A",
		BusinessName: "FIRE", RegisterDate: "2026-01-15", LossDate: "2026-01-15",
		PICTeknik: "BUDI", StatusCode: "1147", StatusLabel: "Register",
	}, header)

	mock.ExpectQuery(exact("detail_claim_header")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.ClaimHeader(context.Background(), scope)
	require.ErrorIs(t, err, inboxpladla.ErrRowNotFound)

	mock.ExpectQuery(exact("detail_claim_header")).WillReturnError(errDB)
	_, err = repo.ClaimHeader(context.Background(), scope)
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAdvicesPickTheQueryPerKind(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"NO", "TYPE", "AMOUNT", "ACC", "DATE", "SENT"}

	_, err := repo.Advices(context.Background(), scope, inboxpladla.AdviceKind("XOL"))
	require.ErrorIs(t, err, inboxpladla.ErrAdviceKindUnknown)

	mock.ExpectQuery(exact("detail_advices_dla")).WithArgs("KEY-1", "REAS").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(" DLA/1 ", " Treaty ", " 100.50 ", " AKS-1 ", day, nil))

	rows, err := repo.Advices(context.Background(), scope, inboxpladla.AdviceKindDLA)
	require.NoError(t, err)
	require.Equal(t, []inboxpladla.AdviceRow{{
		Kind: inboxpladla.AdviceKindDLA, No: "DLA/1", Type: "Treaty", Amount: "100.50",
		AcceptanceNo: "AKS-1", AdviceDate: "2026-01-15", SentDate: "",
	}}, rows)

	mock.ExpectQuery(exact("detail_advices_pla")).WillReturnError(errDB)
	_, err = repo.Advices(context.Background(), scope, inboxpladla.AdviceKindPLA)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("detail_advices_pla")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Advices(context.Background(), scope, inboxpladla.AdviceKindPLA)
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("detail_advices_pla")).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("a", "b", "c", "d", day, day).
			RowError(0, errDB))
	_, err = repo.Advices(context.Background(), scope, inboxpladla.AdviceKindPLA)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentsAreScopedToTheAdviceAndCaller(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"ID", "CAT", "SUB", "NAME", "MIME"}

	_, err := repo.Documents(context.Background(), scope, "PLA/1", inboxpladla.AdviceKind("X"))
	require.ErrorIs(t, err, inboxpladla.ErrAdviceKindUnknown)

	_, err = repo.Documents(context.Background(), scope, "  ", inboxpladla.AdviceKindPLA)
	require.ErrorIs(t, err, inboxpladla.ErrDocumentNotFound)

	mock.ExpectQuery(exact("detail_documents")).
		WithArgs("KEY-1", "PLA/1", "PLA", "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(" DOK-1 ", " Dokumen ", " Laporan ", " a.pdf ", " application/pdf "))

	rows, err := repo.Documents(context.Background(), scope, " PLA/1 ", inboxpladla.AdviceKindPLA)
	require.NoError(t, err)
	require.Equal(t, []inboxpladla.DocumentRow{{
		ID: "DOK-1", Category: "Dokumen", SubCategory: "Laporan", Name: "a.pdf",
		MimeType: "application/pdf",
	}}, rows)

	mock.ExpectQuery(exact("detail_documents")).WillReturnError(errDB)
	_, err = repo.Documents(context.Background(), scope, "PLA/1", inboxpladla.AdviceKindPLA)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("detail_documents")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Documents(context.Background(), scope, "PLA/1", inboxpladla.AdviceKindPLA)
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("detail_documents")).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("a", "b", "c", "d", "e").RowError(0, errDB))
	_, err = repo.Documents(context.Background(), scope, "PLA/1", inboxpladla.AdviceKindPLA)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentContentReadsTheBlob(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"NAME", "MIME", "CONTENT"}

	_, err := repo.DocumentContent(context.Background(), scope, " ")
	require.ErrorIs(t, err, inboxpladla.ErrDocumentNotFound)

	mock.ExpectQuery(exact("detail_document_content")).
		WithArgs("DOK-1", "KEY-1", "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows(columns).AddRow(" a.pdf ", " application/pdf ", []byte("isi")))

	content, err := repo.DocumentContent(context.Background(), scope, " DOK-1 ")
	require.NoError(t, err)
	require.Equal(t, inboxpladla.DocumentContent{
		Name: "a.pdf", MimeType: "application/pdf", Content: []byte("isi"),
	}, content)

	mock.ExpectQuery(exact("detail_document_content")).WillReturnRows(sqlmock.NewRows(columns))
	_, err = repo.DocumentContent(context.Background(), scope, "DOK-1")
	require.ErrorIs(t, err, inboxpladla.ErrDocumentNotFound)

	mock.ExpectQuery(exact("detail_document_content")).WillReturnError(errDB)
	_, err = repo.DocumentContent(context.Background(), scope, "DOK-1")
	require.ErrorIs(t, err, errDB)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestConversationsMarkWhatCanBeReplied(t *testing.T) {
	repo, mock := newMock(t)
	columns := []string{"ID", "CREATED", "SENDER", "MSG", "REPLY", "REPLIER", "REPLIED", "STATE"}

	mock.ExpectQuery(exact("detail_conversations")).WithArgs("KEY-1", "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(" K1 ", day, " Budi ", " Halo ", nil, nil, nil, " 0 ").
			AddRow("K2", day, "Budi", "Tanya", " Jawab ", " Siti ", day,
				inboxpladla.CommunicationAnswered))

	rows, err := repo.Conversations(context.Background(), scope)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, inboxpladla.Conversation{
		ID: "K1", CreatedAt: "2026-01-15", SenderName: "Budi", Message: "Halo",
		Answered: false, CanReply: true,
	}, rows[0])
	require.True(t, rows[1].Answered)
	require.False(t, rows[1].CanReply)
	require.Equal(t, "Jawab", rows[1].Reply)
	require.Equal(t, "2026-01-15", rows[1].RepliedAt)

	mock.ExpectQuery(exact("detail_conversations")).WillReturnError(errDB)
	_, err = repo.Conversations(context.Background(), scope)
	require.ErrorIs(t, err, errDB)

	mock.ExpectQuery(exact("detail_conversations")).
		WillReturnRows(sqlmock.NewRows([]string{"A"}).AddRow("x"))
	_, err = repo.Conversations(context.Background(), scope)
	require.ErrorContains(t, err, "memindai baris")

	mock.ExpectQuery(exact("detail_conversations")).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow("a", day, "b", "c", "d", "e", day, "0").RowError(0, errDB))
	_, err = repo.Conversations(context.Background(), scope)
	require.ErrorContains(t, err, "membaca hasil")
	require.NoError(t, mock.ExpectationsWereMet())
}

func replyCommand() inboxpladla.ReplyCommand {
	return inboxpladla.ReplyCommand{
		ClaimKey: "KEY-1", ConversationID: "K1", Message: "Disetujui.",
		Replier:   inboxpladla.Caller{Login: "REAS", Name: "Mitra"},
		RepliedAt: day,
	}
}

func TestReplyUpdatesOnlyAnUnansweredConversation(t *testing.T) {
	repo, mock := newMock(t)
	command := replyCommand()

	mock.ExpectExec(exact("detail_reply")).
		WithArgs("Disetujui.", "REAS", day, "Mitra", inboxpladla.AnsweredStatus,
			"KEY-1", "REAS", "REAS", "K1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Reply(context.Background(), scope, command))

	// Tidak ada baris tersentuh dan percakapannya tidak ada.
	mock.ExpectExec(exact("detail_reply")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(exact("detail_conversation_exists")).
		WithArgs("KEY-1", "K1", "REAS", "REAS").
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(0))
	require.ErrorIs(t, repo.Reply(context.Background(), scope, command),
		inboxpladla.ErrConversationNotFound)

	// Tidak ada baris tersentuh tetapi percakapannya ada — sudah dijawab.
	mock.ExpectExec(exact("detail_reply")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(exact("detail_conversation_exists")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	require.ErrorIs(t, repo.Reply(context.Background(), scope, command),
		inboxpladla.ErrConversationAlreadyAnswered)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplyFailures(t *testing.T) {
	repo, mock := newMock(t)
	command := replyCommand()

	mock.ExpectExec(exact("detail_reply")).WillReturnError(errDB)
	require.ErrorIs(t, repo.Reply(context.Background(), scope, command), errDB)

	mock.ExpectExec(exact("detail_reply")).
		WillReturnResult(sqlmock.NewErrorResult(errDB))
	require.ErrorContains(t, repo.Reply(context.Background(), scope, command),
		"membaca jumlah baris")

	mock.ExpectExec(exact("detail_reply")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(exact("detail_conversation_exists")).WillReturnError(errDB)
	require.ErrorContains(t, repo.Reply(context.Background(), scope, command),
		"detail_conversation_exists")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryPanicsOnAnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`inboxpladla/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { query("tidak_ada") })
}
