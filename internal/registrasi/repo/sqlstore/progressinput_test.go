package sqlstore

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/registrasi"
)

// PROGRESS_CLAIM_PNC cabang ELSE: POSISIID posisi terpilih, STATUS_PROGRESS2 baris terakhir,
// JSON berisi seluruh baris STATUS PERLU, NEXT_FOLLOWUP sysdate+7 bila kosong.
func TestProgressInputInsert(t *testing.T) {
	db, mock := be4DB(t)
	at := time.Date(2026, 10, 11, 2, 0, 0, 0, time.UTC)
	p := registrasi.ProgressInput{ClaimNumber: "PNCN.26.43", PositionID: "2", Progress1: "003",
		Needs:  []registrasi.ProgressNeed{{ID: "20", Name: "A"}, {ID: "21", Name: "B"}},
		Status: "SUDAH BERHASIL DIHUBUNGKAN", Remarks: " r ", DocType: "ALL", User: "NIK1", At: at}
	doc := `{"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData","ObjectList":[` +
		`{"BranchName":"20","SerialNo":"A","pxObjClass":"ASM-FW-GCNMFW-Data-Object"},` +
		`{"BranchName":"21","SerialNo":"B","pxObjClass":"ASM-FW-GCNMFW-Data-Object"}]}`
	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow("OLD1"))
	mock.ExpectExec(be4Q("progres_input_sisip")).
		WithArgs("PNCN.26.43", "r", "003", "21", "SUDAH BERHASIL DIHUBUNGKAN", wallOrNil(at.Add(7*24*time.Hour)),
			"OLD1", "2", doc, "ALL", "PNCN.26.43").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewProgressInputStore(db).Insert(context.Background(), p))

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectExec(be4Q("progres_input_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, NewProgressInputStore(db).Insert(context.Background(), p), "GCNM_PROGRESS_CLAIM")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProgressInputReads(t *testing.T) {
	db, mock := be4DB(t)
	s := NewProgressInputStore(db)
	ctx := context.Background()

	mock.ExpectQuery(be4Q("progres_input_posisi")).WithArgs("PNCN.1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "c", "p", "s", "d"}).
			AddRow("1", "ASM-FW-GCNMFW-WORK PNCN.1", "REGISTER", "Done", time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)))
	pos, err := s.Positions(ctx, "PNCN.1")
	require.NoError(t, err)
	require.Equal(t, "REGISTER", pos[0].Name)
	require.False(t, pos[0].Open())
	require.Equal(t, time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC), pos[0].Date)

	mock.ExpectQuery(be4Q("progres_input_perlu_tipe")).WithArgs("003", "LIABILITY").
		WillReturnRows(sqlmock.NewRows([]string{"id", "n"}).AddRow("30", " X "))
	opts, err := s.NeedOptions(ctx, "003", " LIABILITY ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.ProgressOption{{ID: "30", Name: "X"}}, opts)
	mock.ExpectQuery(be4Q("progres_input_perlu")).WithArgs("004").WillReturnRows(sqlmock.NewRows([]string{"id", "n"}))
	_, err = s.NeedOptions(ctx, "004", "")
	require.NoError(t, err)

	mock.ExpectQuery(be4Q("progres_input_nama_perlu")).WithArgs("59").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow("HASIL"))
	name, ok, err := s.NeedName(ctx, "59")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "HASIL", name)
	mock.ExpectQuery(be4Q("progres_input_nama_perlu")).WillReturnRows(sqlmock.NewRows([]string{"n"}))
	_, ok, err = s.NeedName(ctx, "x")
	require.NoError(t, err)
	require.False(t, ok)

	// Hari WIB: 11 Okt 00:00 sampai 12 Okt 00:00 jam dinding.
	day := time.Date(2026, 10, 10, 20, 0, 0, 0, time.UTC) // 11 Okt 03:00 WIB
	mock.ExpectQuery(be4Q("progres_input_hari_ini")).
		WithArgs("PNCN.1", time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)).
		WillReturnRows(sqlmock.NewRows([]string{"t"}).AddRow("A, B").AddRow(nil))
	today, err := s.InputToday(ctx, "PNCN.1", day)
	require.NoError(t, err)
	require.Equal(t, []string{"A, B"}, today)

	mock.ExpectQuery(be4Q("progres_input_status")).WillReturnError(be4Boom)
	_, err = s.StatusOptions(ctx)
	require.ErrorContains(t, err, "GCNM_PROGRESS_CLAIM")
	require.NoError(t, mock.ExpectationsWereMet())
}

// INSERTDATABROKEREMAILMASTER: sisip bila belum ada, ubah bila ada, diam bila agen tidak tepat satu.
func TestProgressInputSaveBrokerEmail(t *testing.T) {
	db, mock := be4DB(t)
	s := NewProgressInputStore(db)
	ctx := context.Background()
	b := registrasi.BrokerEmail{Email: "e@x.id", PolicyNumber: "POL", GroupPanel: "003", BusinessCode: "10", BranchCode: "100", ClaimID: "K1"}

	require.NoError(t, s.SaveBrokerEmail(ctx, registrasi.BrokerEmail{}))

	mock.ExpectQuery(be4Q("progres_input_broker_hitung")).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	mock.ExpectQuery(be4Q("progres_input_broker_agen")).WithArgs("POL").WillReturnRows(sqlmock.NewRows([]string{"l", "i"}).AddRow("L1", "S1"))
	mock.ExpectExec(be4Q("progres_input_broker_sisip")).WithArgs("POL", "100", "10", "003", "e@x.id", "K1", "L1", "S1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.SaveBrokerEmail(ctx, b))

	mock.ExpectQuery(be4Q("progres_input_broker_hitung")).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(1))
	mock.ExpectQuery(be4Q("progres_input_broker_agen")).WillReturnRows(sqlmock.NewRows([]string{"l", "i"}).AddRow("L1", "S1"))
	mock.ExpectExec(be4Q("progres_input_broker_ubah")).WithArgs("e@x.id", "POL", "003", "10", "100", "L1", "S1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.SaveBrokerEmail(ctx, b))

	mock.ExpectQuery(be4Q("progres_input_broker_hitung")).WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(0))
	mock.ExpectQuery(be4Q("progres_input_broker_agen")).WillReturnRows(sqlmock.NewRows([]string{"l", "i"}))
	require.NoError(t, s.SaveBrokerEmail(ctx, b))
	require.NoError(t, mock.ExpectationsWereMet())
}
