package sqlstore

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

func TestSaveCommitteeNote(t *testing.T) {
	db, mock := be4DB(t)
	n := registrasi.CommitteeNote{
		Circumstances: "a", ExtentOfLoss: "b", LegalLiability: "c", Remarks: "d", RemarkInvestigation: "e",
		Diagnose: "f", DiagnoseCode: "g", DiagnoseDesc: "h", Receiver: "1", InitialName: "TIDAK-DITULIS",
	}
	mock.ExpectExec(be4Q("coverage_catatan_komite")).
		WithArgs("a", "b", "c", "d", "e", "f", "g", "h", "1", "K1", 2, 3).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewClaimStore(db).SaveCommitteeNote(context.Background(), "K1", 2, 3, n))

	mock.ExpectExec(be4Q("coverage_catatan_komite")).WillReturnResult(sqlmock.NewResult(0, 0))
	err := NewClaimStore(db).SaveCommitteeNote(context.Background(), "K1", 9, 9, n)
	require.ErrorIs(t, err, registrasi.ErrInvalidAction)

	mock.ExpectExec(be4Q("coverage_catatan_komite")).WillReturnError(be4Boom)
	require.ErrorIs(t, NewClaimStore(db).SaveCommitteeNote(context.Background(), "K1", 1, 1, n), be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())
}
