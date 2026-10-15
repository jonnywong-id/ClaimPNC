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

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

func newMock(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

// workItemColumns adalah alias kolom kueri daftar sesuai urutan pemindai.
var workItemColumns = []string{
	"REFERENCE", "CASE_ID", "POLICY_NUMBER", "CLAIM_NUMBER", "INSURED_NAME", "LOSS_DATE",
	"GROUP_PANEL", "SENDER_NAME", "DOCUMENT_RECEIVED_AT", "SHEET_COUNT", "INBOX_ENTRY_AT",
	"ANALYST_NOTE", "TRACK", "TRACK_STATUS", "LETTER_PRINTED_AT", "CLAIM_AGE",
	"EXPIRY_STATUS", "CLAIM_SCREEN_READY", "TOTAL_ROWS",
}

func TestListReceiveScansRowsAndDerivesClaimType(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(query("list_receive")).
		WithArgs(50, 50).
		WillReturnRows(sqlmock.NewRows(workItemColumns).
			AddRow("K1", "RCV-1", "POL", "PNCN.26.1", "Nama", "2026-09-01",
				inboxmanagerreceivepucl.GroupPanelPA, "Pengirim", "03/09/2026", nil,
				"2026-09-03", nil, nil, nil, nil, nil, nil, "1", 7).
			AddRow("K2", "RCV-2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
				nil, nil, nil, nil, nil, 7))

	page, err := repo.List(context.Background(), sampleQuery(t, inboxmanagerreceivepucl.TabReceive),
		inboxmanagerreceivepucl.Pagination{Page: 2, Size: 50})
	require.NoError(t, err)
	require.Equal(t, 7, page.Total)
	require.Equal(t, inboxmanagerreceivepucl.Pagination{Page: 2, Size: 50}, page.Pagination)
	require.Len(t, page.Items, 2)
	require.Equal(t, "RCV-1", page.Items[0].CaseID)
	require.Equal(t, "POL", page.Items[0].PolicyNumber)
	require.Equal(t,
		inboxmanagerreceivepucl.ClaimTypeOf(inboxmanagerreceivepucl.GroupPanelPA),
		page.Items[0].ClaimType)
	require.Empty(t, page.Items[1].ClaimType, "Group Panel NULL tidak diterjemahkan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRCLPUCLBindsStatusAndWorkbasket(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(query("list_rclpucl")).
		WithArgs(inboxmanagerreceivepucl.WorkStatusCompleted,
			inboxmanagerreceivepucl.RCLPUCLWorkbasket, 0, inboxmanagerreceivepucl.DefaultPageSize).
		WillReturnRows(sqlmock.NewRows(workItemColumns))

	page, err := repo.List(context.Background(), sampleQuery(t, inboxmanagerreceivepucl.TabRCLPUCL),
		inboxmanagerreceivepucl.Pagination{})
	require.NoError(t, err)
	require.Empty(t, page.Items)
	require.NotNil(t, page.Items)
	require.Zero(t, page.Total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListRejectsTabWithoutQuery(t *testing.T) {
	repo, mock := newMock(t)

	_, err := repo.List(context.Background(),
		inboxmanagerreceivepucl.Query{Tab: inboxmanagerreceivepucl.Tab{Code: "9"}},
		inboxmanagerreceivepucl.Pagination{})
	require.EqualError(t, err, `inboxmanagerreceivepucl/sqlstore: tab "9" belum punya kueri`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListWrapsQueryScanAndRowsErrors(t *testing.T) {
	q := sampleQuery(t, inboxmanagerreceivepucl.TabReceive)

	t.Run("query", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(query("list_receive")).WillReturnError(errors.New("ora-1"))
		_, err := repo.List(context.Background(), q, inboxmanagerreceivepucl.Pagination{})
		require.ErrorContains(t, err, "menjalankan kueri list_receive: ora-1")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(query("list_receive")).
			WillReturnRows(sqlmock.NewRows([]string{"SATU"}).AddRow("x"))
		_, err := repo.List(context.Background(), q, inboxmanagerreceivepucl.Pagination{})
		require.ErrorContains(t, err, "membaca baris kueri list_receive")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows", func(t *testing.T) {
		repo, mock := newMock(t)
		row := make([]driver.Value, len(workItemColumns))
		mock.ExpectQuery(query("list_receive")).
			WillReturnRows(sqlmock.NewRows(workItemColumns).
				AddRow(row...).
				RowError(0, errors.New("putus")))
		_, err := repo.List(context.Background(), q, inboxmanagerreceivepucl.Pagination{})
		require.ErrorContains(t, err, "menelusuri hasil kueri list_receive: putus")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDocumentScansEveryColumnInOrder(t *testing.T) {
	repo, mock := newMock(t)

	values := make([]driver.Value, len(documentColumns))
	for i, name := range documentColumns {
		values[i] = "v-" + name
	}
	values[3] = "006"

	mock.ExpectQuery(query("detail_receive_document")).
		WithArgs("K1").
		WillReturnRows(sqlmock.NewRows(documentColumns).AddRow(values...))

	doc, err := repo.Document(context.Background(), "K1")
	require.NoError(t, err)
	require.Equal(t, inboxmanagerreceivepucl.ReceiveDocument{
		Reference:        "v-REFERENCE",
		CaseID:           "v-CASE_ID",
		ClaimNumber:      "v-CLAIM_NUMBER",
		ClaimType:        inboxmanagerreceivepucl.ClaimTypeOf("006"),
		GroupPanel:       "006",
		WorkStatus:       "v-WORK_STATUS",
		CreatedAt:        "v-CREATED_AT",
		ReceivedAt:       "v-RECEIVED_AT",
		SenderName:       "v-SENDER_NAME",
		SenderEmail:      "v-SENDER_EMAIL",
		SenderPhone:      "v-SENDER_PHONE",
		CourierName:      "v-COURIER_NAME",
		InsuredName:      "v-INSURED_NAME",
		PolicyNumber:     "v-POLICY_NUMBER",
		LossDate:         "v-LOSS_DATE",
		ReferenceNumber:  "v-REFERENCE_NUMBER",
		InsuredEmail:     "v-INSURED_EMAIL",
		LossLocation:     "v-LOSS_LOCATION",
		DriverLicence:    "v-DRIVER_LICENCE",
		Chronology:       "v-CHRONOLOGY",
		DamageDetail:     "v-DAMAGE_DETAIL",
		TransferReason:   "v-TRANSFER_REASON",
		EmailSubject:     "v-EMAIL_SUBJECT",
		NotRegisteredNot: "v-NOT_REGISTERED_NOTE",
	}, doc)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentBlankGroupPanelLeavesClaimTypeEmpty(t *testing.T) {
	repo, mock := newMock(t)

	values := make([]driver.Value, len(documentColumns))
	values[1] = "RCV-9"
	values[3] = "  "
	mock.ExpectQuery(query("detail_receive_document")).
		WithArgs("K9").
		WillReturnRows(sqlmock.NewRows(documentColumns).AddRow(values...))

	doc, err := repo.Document(context.Background(), "K9")
	require.NoError(t, err)
	require.Equal(t, "RCV-9", doc.CaseID)
	require.Empty(t, doc.ClaimType)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentNotFoundAndFailure(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(query("detail_receive_document")).WithArgs("X").
		WillReturnError(sql.ErrNoRows)
	_, err := repo.Document(context.Background(), "X")
	require.ErrorIs(t, err, inboxmanagerreceivepucl.ErrDocumentNotFound)

	mock.ExpectQuery(query("detail_receive_document")).WithArgs("Y").
		WillReturnError(errors.New("ora-2"))
	_, err = repo.Document(context.Background(), "Y")
	require.ErrorContains(t, err, "membaca berkas penerimaan dokumen: ora-2")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableRunsBothChecks(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(query("check_receive")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(query("check_rclpucl")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	require.NoError(t, repo.CheckTable(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckTableReportsWhichTableFailed(t *testing.T) {
	repo, mock := newMock(t)
	mock.ExpectQuery(query("check_receive")).WillReturnError(errors.New("ora-942"))
	err := repo.CheckTable(context.Background())
	require.Regexp(t, regexp.MustCompile(`^membaca DATAPEGA\.PC_ASSIGN_WORKLIST, POOLDATA\.T_CLAIMLIST_ADMIN.*ora-942$`), err.Error())
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMock(t)
	mock.ExpectQuery(query("check_receive")).WillReturnRows(sqlmock.NewRows([]string{"X"}).AddRow(1))
	mock.ExpectQuery(query("check_rclpucl")).WillReturnError(errors.New("ora-942"))
	require.EqualError(t, repo.CheckTable(context.Background()),
		"membaca POOLDATA.TC_PNC_PUCL atau POOLDATA.T_CLAIM_PNC: ora-942")
	require.NoError(t, mock.ExpectationsWereMet())
}
