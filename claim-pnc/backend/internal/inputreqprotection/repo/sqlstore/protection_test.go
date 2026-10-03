package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inputreqprotection"
)

// Uji adapter Oracle memakai sqlmock: yang diperiksa adalah kueri yang dikirim, argumennya,
// pemetaan baris, dan setiap cabang galat.

var errDB = errors.New("ORA-03113: koneksi terputus")

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

// q mengubah teks kueri bernama menjadi pola regex yang cocok persis.
func q(name string) string { return regexp.QuoteMeta(query(name)) }

var protectionColumns = []string{
	"OPEN_PROTECTION_ID", "POLICY_NO", "CLAIM_NO", "ID_CLAIM", "PROTECTION_TYPE_ID",
	"PROTECTION_TYPE_NAME", "CREATE_DATE", "NOTES", "APPROVAL_STATUS", "CREATED_BY",
	"OLD_DATA", "NEW_DATA", "OBJECT_NAME", "BRANCH_NAME",
}

var created = time.Date(2026, time.September, 23, 3, 0, 0, 0, time.UTC)

func dolRow() []driver.Value {
	return []driver.Value{
		" OPCN.26.0004 ", "POL-5", "PNCN.26.0009", "PNCN.26.0009", "7",
		"Perubahan DOL", created, "catatan", nil, "TEKNIK",
		"2026-08-14", "2026-08-17", "OBJ", " CAB ",
	}
}

func TestListCountsThenReadsPageWithoutSearch(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(q("protection_count")).
		WithArgs(nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(3))
	mock.ExpectQuery(q("protection_list")).
		WithArgs(nil, nil, nil, nil, 0, inputreqprotection.DefaultLimit).
		WillReturnRows(sqlmock.NewRows(protectionColumns).
			AddRow(dolRow()...).
			AddRow("OPC-201", nil, nil, nil, "8", nil, nil, nil, nil, nil, "12002", "COL-1", nil, nil).
			AddRow("OPC-202", "POL-1", nil, nil, "1", "General", created, "x", nil, "U", "abaikan", "abaikan", nil, nil))

	page, err := NewRepo(db).List(context.Background(), inputreqprotection.Filter{})
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)
	require.Len(t, page.Protections, 3)

	first := page.Protections[0]
	require.Equal(t, "OPCN.26.0004", first.Number)
	require.Equal(t, "POL-5", first.PolicyNumber)
	require.Equal(t, "PNCN.26.0009", first.ClaimReference)
	require.Equal(t, "Perubahan DOL", first.TypeName)
	require.Equal(t, created, first.InputDate)
	require.Equal(t, created, first.CreatedAt)
	require.Equal(t, "CAB", first.ChangeDetail.BranchName)
	require.Equal(t, "2026-08-14", first.ChangeDetail.LossDateBefore.Format("2006-01-02"))
	require.Equal(t, "2026-08-17", first.ChangeDetail.LossDateAfter.Format("2006-01-02"))

	// Tipe '8' memetakan OLD_DATA/NEW_DATA ke penyebab kerugian; kolom NULL menjadi kosong.
	second := page.Protections[1]
	require.Equal(t, "12002", second.ChangeDetail.CauseOfLossID)
	require.Equal(t, "COL-1", second.ChangeDetail.CauseOfLossMasterID)
	require.Equal(t, "", second.PolicyNumber)
	require.True(t, second.InputDate.IsZero())

	// Tipe lain mengabaikan isi kedua kolom.
	require.Equal(t, inputreqprotection.ChangeDetail{}, page.Protections[2].ChangeDetail)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSendsEscapedSearchPatternFourTimes(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(q("protection_count")).
		WithArgs("POL_50%", "%POL50%", "%POL50%", "%POL50%").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT"}).AddRow(0))
	mock.ExpectQuery(q("protection_list")).
		WithArgs("POL_50%", "%POL50%", "%POL50%", "%POL50%", 10, 5).
		WillReturnRows(sqlmock.NewRows(protectionColumns))

	page, err := NewRepo(db).List(context.Background(),
		inputreqprotection.Filter{Search: " pol_50% ", Limit: 5, Offset: 10})
	require.NoError(t, err)
	require.Zero(t, page.Total)
	require.Empty(t, page.Protections)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("hitungan gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("protection_count")).WillReturnError(errDB)
		_, err := NewRepo(db).List(ctx, inputreqprotection.Filter{})
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "menghitung permintaan proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("kueri daftar gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("protection_count")).WillReturnRows(sqlmock.NewRows([]string{"C"}).AddRow(1))
		mock.ExpectQuery(q("protection_list")).WillReturnError(errDB)
		_, err := NewRepo(db).List(ctx, inputreqprotection.Filter{})
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "membaca permintaan proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pemindaian gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("protection_count")).WillReturnRows(sqlmock.NewRows([]string{"C"}).AddRow(1))
		mock.ExpectQuery(q("protection_list")).
			WillReturnRows(sqlmock.NewRows([]string{"OPEN_PROTECTION_ID"}).AddRow("X"))
		_, err := NewRepo(db).List(ctx, inputreqprotection.Filter{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "memindai permintaan proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("galat baris", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectQuery(q("protection_count")).WillReturnRows(sqlmock.NewRows([]string{"C"}).AddRow(1))
		mock.ExpectQuery(q("protection_list")).
			WillReturnRows(sqlmock.NewRows(protectionColumns).AddRow(dolRow()...).RowError(0, errDB))
		_, err := NewRepo(db).List(ctx, inputreqprotection.Filter{})
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "membaca baris permintaan proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetMapsRowNotFoundAndError(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	mock.ExpectQuery(q("protection_get")).WithArgs("OPCN.26.0004").
		WillReturnRows(sqlmock.NewRows(protectionColumns).AddRow(dolRow()...))
	p, err := NewRepo(db).Get(ctx, "  opcn.26.0004 ")
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0004", p.Number)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_get")).WithArgs("X").WillReturnRows(sqlmock.NewRows(protectionColumns))
	_, err = NewRepo(db).Get(ctx, "x")
	require.ErrorIs(t, err, inputreqprotection.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_get")).WillReturnError(errDB)
	_, err = NewRepo(db).Get(ctx, "x")
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "memindai permintaan proteksi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHasDuplicateBindsExceptNumberTwice(t *testing.T) {
	ctx := context.Background()
	day := time.Date(2026, time.September, 23, 0, 0, 0, 0, time.UTC)
	key := inputreqprotection.DuplicateKey{PolicyNumber: " pol-1 ", Type: " 1 ", Day: day}

	db, mock := newMock(t)
	mock.ExpectQuery(q("protection_duplicate")).
		WithArgs("POL-1", "1", nil, nil, day).
		WillReturnRows(sqlmock.NewRows([]string{"C"}).AddRow(2))
	dup, err := NewRepo(db).HasDuplicate(ctx, key, "  ")
	require.NoError(t, err)
	require.True(t, dup)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_duplicate")).
		WithArgs("POL-1", "1", "OPCN.26.0001", "OPCN.26.0001", day).
		WillReturnRows(sqlmock.NewRows([]string{"C"}).AddRow(0))
	dup, err = NewRepo(db).HasDuplicate(ctx, key, " opcn.26.0001 ")
	require.NoError(t, err)
	require.False(t, dup)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_duplicate")).WillReturnError(errDB)
	_, err = NewRepo(db).HasDuplicate(ctx, key, "")
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "memeriksa proteksi ganda")
	require.NoError(t, mock.ExpectationsWereMet())
}

// pegaClaim adalah klaim warisan Pega dengan DOL tercatat.
func pegaClaim() inputreqprotection.Claim {
	dol := time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC)
	return inputreqprotection.Claim{
		Number: "PNC-1865", PegaID: "ASM-FW-GCNMFW-WORK PNC-1865", PolicyNumber: "POL-1",
		LossDate: &dol, CauseOfLoss: "12002", ObjectName: "OBJ", BranchName: "CAB",
	}
}

func TestCreateIssuesSequenceAndInsertsDerivedColumns(t *testing.T) {
	db, mock := newMock(t)
	after := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	wib := time.FixedZone("WIB", 7*60*60)
	at := time.Date(2026, time.September, 23, 10, 0, 0, 0, wib)
	draft := inputreqprotection.Draft{
		ClaimNumber: "PNC-1865", Type: "7", Note: "catatan",
		Change: inputreqprotection.ChangeRequest{LossDateAfter: &after},
	}

	mock.ExpectBegin()
	mock.ExpectQuery(q("protection_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(42))
	mock.ExpectExec(q("protection_insert")).
		WithArgs("OPCN.26.0042", "POL-1", "PNC-1865", "ASM-FW-GCNMFW-WORK PNC-1865", "7",
			at.UTC(), "ADMIN", "catatan", "2026-08-14", "2026-08-17", "OBJ", "CAB").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	p, err := NewRepo(db).Create(context.Background(), draft, pegaClaim(), "ADMIN", at)
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0042", p.Number)
	require.Equal(t, "POL-1", p.PolicyNumber)
	require.Equal(t, inputreqprotection.AcceptPending, p.AcceptStatus)
	require.Equal(t, "ADMIN", p.CreatedBy)
	require.Equal(t, at, p.CreatedAt)
	require.Equal(t, "12002", p.ChangeDetail.CauseOfLossID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateCauseOfLossTypeStoresCauseColumns(t *testing.T) {
	// Tipe '8' menyimpan penyebab kerugian klaim dan pilihan pengguna; isian kosong menjadi NULL.
	db, mock := newMock(t)
	at := time.Date(2026, time.September, 23, 3, 0, 0, 0, time.UTC)
	claim := inputreqprotection.Claim{Number: "PNCN.26.0007", CauseOfLoss: "Kebakaran"}
	draft := inputreqprotection.Draft{ClaimNumber: "PNCN.26.0007", Type: "8", Note: "n",
		Change: inputreqprotection.ChangeRequest{CauseOfLossAfter: "COL-2"}}

	mock.ExpectBegin()
	mock.ExpectQuery(q("protection_next_sequence")).
		WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(1))
	mock.ExpectExec(q("protection_insert")).
		WithArgs("OPCN.26.0001", nil, "PNCN.26.0007", "PNCN.26.0007", "8",
			at, "U", "n", "Kebakaran", "COL-2", nil, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err := NewRepo(db).Create(context.Background(), draft, claim, "U", at)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateErrorBranchesRollBack(t *testing.T) {
	ctx := context.Background()
	draft := inputreqprotection.Draft{ClaimNumber: "PNC-1865", Type: "1", Note: "n"}
	at := time.Date(2026, time.September, 23, 3, 0, 0, 0, time.UTC)

	t.Run("transaksi gagal dibuka", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin().WillReturnError(errDB)
		_, err := NewRepo(db).Create(ctx, draft, pegaClaim(), "U", at)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "membuka transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sequence gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("protection_next_sequence")).WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := NewRepo(db).Create(ctx, draft, pegaClaim(), "U", at)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "menerbitkan nomor proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("penyisipan gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("protection_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(1))
		// Tipe '1' tidak menyimpan OLD_DATA/NEW_DATA.
		mock.ExpectExec(q("protection_insert")).
			WithArgs("OPCN.26.0001", "POL-1", "PNC-1865", "ASM-FW-GCNMFW-WORK PNC-1865", "1",
				at, "U", "n", nil, nil, "OBJ", "CAB").
			WillReturnError(errDB)
		mock.ExpectRollback()
		_, err := NewRepo(db).Create(ctx, draft, pegaClaim(), "U", at)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "menyimpan permintaan proteksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("commit gagal", func(t *testing.T) {
		db, mock := newMock(t)
		mock.ExpectBegin()
		mock.ExpectQuery(q("protection_next_sequence")).
			WillReturnRows(sqlmock.NewRows([]string{"NEXTVAL"}).AddRow(1))
		mock.ExpectExec(q("protection_insert")).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit().WillReturnError(errDB)
		_, err := NewRepo(db).Create(ctx, draft, pegaClaim(), "U", at)
		require.ErrorIs(t, err, errDB)
		require.Contains(t, err.Error(), "menyelesaikan transaksi")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateRereadsRowAfterSuccess(t *testing.T) {
	db, mock := newMock(t)
	draft := inputreqprotection.Draft{ClaimNumber: "PNC-1865", Type: "1", Note: "baru"}

	mock.ExpectExec(q("protection_update")).
		WithArgs("POL-1", "PNC-1865", "ASM-FW-GCNMFW-WORK PNC-1865", "1", "baru",
			nil, nil, "OBJ", "CAB", "OPCN.26.0004").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(q("protection_get")).WithArgs("OPCN.26.0004").
		WillReturnRows(sqlmock.NewRows(protectionColumns).AddRow(dolRow()...))

	p, err := NewRepo(db).Update(context.Background(), " opcn.26.0004", draft, pegaClaim(), "U", created)
	require.NoError(t, err)
	require.Equal(t, "OPCN.26.0004", p.Number)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateDistinguishesWhyNoRowWasTouched(t *testing.T) {
	ctx := context.Background()
	draft := inputreqprotection.Draft{ClaimNumber: "PNC-1865", Type: "1", Note: "n"}

	accepted := dolRow()
	accepted[8] = "1"
	locked := dolRow()

	cases := []struct {
		name string
		rows *sqlmock.Rows
		want error
	}{
		{"sudah diakseptasi", sqlmock.NewRows(protectionColumns).AddRow(accepted...), inputreqprotection.ErrAccepted},
		{"sudah tertaut klaim", sqlmock.NewRows(protectionColumns).AddRow(locked...), inputreqprotection.ErrLocked},
		{"tidak ada", sqlmock.NewRows(protectionColumns), inputreqprotection.ErrNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, mock := newMock(t)
			mock.ExpectExec(q("protection_update")).WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectQuery(q("protection_get")).WillReturnRows(c.rows)
			_, err := NewRepo(db).Update(ctx, "OPCN.26.0004", draft, pegaClaim(), "U", created)
			require.ErrorIs(t, err, c.want)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUpdateExecAndRowsAffectedErrors(t *testing.T) {
	ctx := context.Background()
	draft := inputreqprotection.Draft{ClaimNumber: "PNC-1865", Type: "1", Note: "n"}

	db, mock := newMock(t)
	mock.ExpectExec(q("protection_update")).WillReturnError(errDB)
	_, err := NewRepo(db).Update(ctx, "X", draft, pegaClaim(), "U", created)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "menyunting permintaan proteksi")
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectExec(q("protection_update")).WillReturnResult(sqlmock.NewErrorResult(errDB))
	_, err = NewRepo(db).Update(ctx, "X", draft, pegaClaim(), "U", created)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "membaca jumlah baris tersentuh")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTypeRepoSkipsEmptyCodesAndTrims(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(q("protection_type_list")).
		WillReturnRows(sqlmock.NewRows([]string{"PROTECTION_TYPE_ID", "PROTECTION_TYPE_NAME"}).
			AddRow(" 1 ", "General ").
			AddRow(nil, "Tanpa kode").
			AddRow("  ", "Spasi").
			AddRow("9", nil))

	types, err := NewTypeRepo(db).ListTypes(context.Background())
	require.NoError(t, err)
	require.Equal(t, []inputreqprotection.ProtectionType{
		{ID: "1", Name: "General"},
		{ID: "9", Name: ""},
	}, types)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTypeRepoErrorBranches(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	mock.ExpectQuery(q("protection_type_list")).WillReturnError(errDB)
	_, err := NewTypeRepo(db).ListTypes(ctx)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "membaca master tipe proteksi")
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_type_list")).
		WillReturnRows(sqlmock.NewRows([]string{"PROTECTION_TYPE_ID"}).AddRow("1"))
	_, err = NewTypeRepo(db).ListTypes(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "memindai master tipe proteksi")
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("protection_type_list")).
		WillReturnRows(sqlmock.NewRows([]string{"A", "B"}).AddRow("1", "x").RowError(0, errDB))
	_, err = NewTypeRepo(db).ListTypes(ctx)
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "membaca baris master tipe proteksi")
	require.NoError(t, mock.ExpectationsWereMet())
}

var claimColumns = []string{"CLAIMID", "NOPOLIS", "QQNAME", "DATEOFLOSS", "CAUSEOFLOSS", "BRANCHNAME", "OBJECTNAME"}

func TestFindClaimMatchesBothClaimIDForms(t *testing.T) {
	db, mock := newMock(t)
	dol := time.Date(2026, time.August, 14, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(q("claim_find")).
		WithArgs("PNC-1865", "ASM-FW-GCNMFW-WORK PNC-1865").
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow(" ASM-FW-GCNMFW-WORK PNC-1865 ", "POL-1", "TERTANGGUNG", dol, "12002", "CAB", "OBJ"))

	c, err := NewClaimRepo(db).FindClaim(context.Background(), " pnc-1865 ")
	require.NoError(t, err)
	require.Equal(t, inputreqprotection.Claim{
		Number: "PNC-1865", PegaID: "ASM-FW-GCNMFW-WORK PNC-1865", PolicyNumber: "POL-1",
		InsuredName: "TERTANGGUNG", LossDate: &dol, CauseOfLoss: "12002",
		BranchName: "CAB", ObjectName: "OBJ",
	}, c)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindClaimWithoutLossDateNotFoundAndError(t *testing.T) {
	ctx := context.Background()

	db, mock := newMock(t)
	mock.ExpectQuery(q("claim_find")).
		WillReturnRows(sqlmock.NewRows(claimColumns).AddRow("PNCN.26.0008", nil, nil, nil, nil, nil, nil))
	c, err := NewClaimRepo(db).FindClaim(ctx, "PNCN.26.0008")
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0008", c.Number)
	require.Equal(t, "PNCN.26.0008", c.PegaID)
	require.Nil(t, c.LossDate)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("claim_find")).WillReturnRows(sqlmock.NewRows(claimColumns))
	_, err = NewClaimRepo(db).FindClaim(ctx, "X")
	require.ErrorIs(t, err, inputreqprotection.ErrClaimNotFound)
	require.NoError(t, mock.ExpectationsWereMet())

	db, mock = newMock(t)
	mock.ExpectQuery(q("claim_find")).WillReturnError(errDB)
	_, err = NewClaimRepo(db).FindClaim(ctx, "X")
	require.ErrorIs(t, err, errDB)
	require.Contains(t, err.Error(), "mencari klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestChangeDetailHelpers(t *testing.T) {
	// Tanggal tak terbaca menjadi nil, bukan galat; teks kosong menjadi nil.
	require.Nil(t, bacaTanggal("14/08/2026"))
	require.Nil(t, bacaTanggal("  "))
	require.Equal(t, "", formatTanggal(nil))

	d := decodeChangeDetail("7", "rusak", "")
	require.Nil(t, d.LossDateBefore)
	require.Nil(t, d.LossDateAfter)

	lama, baru := encodeChangeDetail(" 7 ", inputreqprotection.ChangeDetail{})
	require.Nil(t, lama)
	require.Nil(t, baru)

	require.Nil(t, nullIfEmpty("   "))
	require.Equal(t, "a", nullIfEmpty(" a "))
}

func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`inputreqprotection/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { query("tidak_ada") })
}

func TestSplitByNameDropsCommentsAndEmptyBodies(t *testing.T) {
	got := splitByName("-- kepala\nSELECT 0\n-- name: a\n-- komentar\nSELECT 1\n\n-- name: kosong\n-- hanya komentar\n-- name: b\nSELECT 2")
	require.Equal(t, map[string]string{"a": "SELECT 1", "b": "SELECT 2"}, got)
}
