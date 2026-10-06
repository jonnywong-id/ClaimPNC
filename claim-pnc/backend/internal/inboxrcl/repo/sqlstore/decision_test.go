package sqlstore

import (
	"context"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrcl"
)

// decisionQueries adalah ketujuh kueri decision.sql beserta jumlah penanda bind-nya.
var decisionQueries = map[string]int{
	"decision_lock":            5,
	"decision_technical_pic":   1,
	"decision_claim_key":       1,
	"decision_update_pucl":     11,
	"decision_update_worklist": 5,
	"decision_close_tasks":     3,
	"decision_open_task":       9,
	"decision_insert_history":  3,
}

func TestPenandaBindKueriKeputusanBerurutanDanTunggal(t *testing.T) {
	for name, count := range decisionQueries {
		markers := regexp.MustCompile(`:\d+`).FindAllString(query(name), -1)
		want := make([]string, 0, count)
		for i := 1; i <= count; i++ {
			want = append(want, fmt.Sprintf(":%d", i))
		}
		require.Equalf(t, want, markers, "kueri %s", name)
	}
}

// TestKeputusanTidakMenyentuhTabelPega — `P-1` dan keputusan Work Owner 2026-10-05.
func TestKeputusanTidakMenyentuhTabelPega(t *testing.T) {
	for name := range decisionQueries {
		text := strings.ToUpper(query(name))
		require.NotContainsf(t, text, "DATAPEGA.", "kueri %s", name)
		require.NotContainsf(t, text, "T_ACCESS_GROUP_PNC", "kueri %s", name)
		require.NotContainsf(t, text, "DELETE ", "kueri %s — tidak ada penghapusan fisik (D-66)", name)
		require.NotContainsf(t, text, "||", "kueri %s — tanpa perangkaian", name)
		require.NotContainsf(t, text, "--", "kueri %s — komentar tidak terkirim", name)
	}
}

// TestKueriPenguncianMemakaiPenyaringAntrean — klaim hanya dapat diputus bila masih di
// antrean pemanggil, dengan penyaring yang sama dengan daftar.
func TestKueriPenguncianMemakaiPenyaringAntrean(t *testing.T) {
	text := strings.ToUpper(query("decision_lock"))
	require.Contains(t, text, "UPPER(TRIM(P.CLAIMID)) = UPPER(:1)")
	require.Contains(t, text, "UPPER(TRIM(P.ASSIGNED_OPERATOR_ID)) = UPPER(:2)")
	require.Contains(t, text, "P.STATUS_WORK <> :3")
	require.Contains(t, text, "P.TGL_KIRIM_PUCL IS NOT NULL")
	require.Contains(t, text, "TRIM(P.RCL_PUCL) IN (:4, :5)")
	require.Contains(t, text, "FOR UPDATE")
}

// TestDaftarKerjaHanyaBarisPNCN — baris Pega berkunci awalan kelas tidak pernah tersentuh.
func TestDaftarKerjaHanyaBarisPNCN(t *testing.T) {
	require.Contains(t, query("decision_update_worklist"), "WHERE PZINSKEY = :5")
}

func exactQ(name string) string { return "^" + regexp.QuoteMeta(query(name)) + "$" }

type anyTime struct{}

func (anyTime) Match(v driver.Value) bool { _, ok := v.(time.Time); return ok }

// TestSetujuMenulisSeluruhnyaDalamSatuTransaksi menelusuri urutan pernyataan keputusan Setuju.
func TestSetujuMenulisSeluruhnyaDalamSatuTransaksi(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	at := time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)
	loss := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(exactQ("decision_lock")).
		WithArgs("PNCN.26.31", "JONNY", inboxrcl.StatusKerjaSelesai, "1", "3").
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}).
			AddRow("PNCN.26.31", "1", "0", "1", "1151", nil, loss, at, nil, "JONNY"))
	mock.ExpectExec(exactQ("decision_update_pucl")).
		WithArgs(inboxrcl.StatusClaimRCL, inboxrcl.StatusKlaimActive, "0", nil, nil,
			at, at, at, nil, inboxrcl.WorkbasketRCLPUCL, "PNCN.26.31").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_update_worklist")).
		WithArgs(inboxrcl.StatusClaimRCL, at, nil, inboxrcl.StageNameRCLPUCL, "PNCN.26.31").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(exactQ("decision_claim_key")).WithArgs("PNCN.26.31").
		WillReturnRows(sqlmock.NewRows([]string{"k"}).AddRow("KUNCI-KLAIM"))
	mock.ExpectExec(exactQ("decision_close_tasks")).
		WithArgs(at, inboxrcl.TicketSendToPUCL, "PNCN.26.31").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_open_task")).
		WithArgs(sqlmock.AnyArg(), "KUNCI-KLAIM", "PNCN.26.31", inboxrcl.StageRCLPUCL,
			inboxrcl.QueueWorkbasket, inboxrcl.WorkbasketRCLPUCL, nil, at, nil).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_insert_history")).
		WithArgs("PNCN.26.31", inboxrcl.HistorySentToDoctor, "JONNY").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	out, err := NewRepo(db).Decide(context.Background(), inboxrcl.DecisionCommand{
		Operator: "JONNY", ClaimNumber: "PNCN.26.31", Decision: inboxrcl.DecisionApprove, At: at,
	})
	require.NoError(t, err)
	require.Equal(t, inboxrcl.StageRCLPUCL, out.NextStage)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestTidakSetujuKembaliKePICTeknik — tugas worklist bertuan PIC Teknik, riwayat dua baris.
func TestTidakSetujuKembaliKePICTeknik(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	at := time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(exactQ("decision_lock")).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}).
			AddRow("PNC-2067", "1", "0", "0", "AKTIF", nil, nil, nil, nil, ""))
	mock.ExpectQuery(exactQ("decision_technical_pic")).WithArgs("PNC-2067").
		WillReturnRows(sqlmock.NewRows([]string{"p"}).AddRow("PICTEKNIK01"))
	mock.ExpectExec(exactQ("decision_update_pucl")).
		WithArgs(inboxrcl.StatusClaimAnalyst, inboxrcl.StatusKlaimActive, nil, "0", nil,
			at, nil, nil, "Diagnosa dijamin.", "PICTEKNIK01", "PNC-2067").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_update_worklist")).
		WithArgs(inboxrcl.StatusClaimAnalyst, nil, "PICTEKNIK01", inboxrcl.StageNameSendToAnalyst, "PNC-2067").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(exactQ("decision_claim_key")).WithArgs("PNC-2067").
		WillReturnRows(sqlmock.NewRows([]string{"k"}))
	mock.ExpectExec(exactQ("decision_close_tasks")).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(exactQ("decision_open_task")).
		WithArgs(sqlmock.AnyArg(), "ASM-FW-GCNMFW-WORK PNC-2067", "PNC-2067", inboxrcl.StageSendToAnalyst,
			inboxrcl.QueueWorklist, nil, "PICTEKNIK01", anyTime{}, anyTime{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_insert_history")).
		WithArgs("ASM-FW-GCNMFW-WORK PNC-2067", inboxrcl.HistorySentToDoctor, "JONNY").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(exactQ("decision_insert_history")).
		WithArgs("ASM-FW-GCNMFW-WORK PNC-2067", inboxrcl.HistorySentToMSIG, "JONNY").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	_, err = NewRepo(db).Decide(context.Background(), inboxrcl.DecisionCommand{
		Operator: "JONNY", ClaimNumber: "PNC-2067", Decision: inboxrcl.DecisionDisagree,
		DoctorReason: "  Diagnosa dijamin.  ", At: at,
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestKlaimDiLuarAntreanTidakMenulisApaPun — baris tidak terkunci, transaksi digulung balik.
func TestKlaimDiLuarAntreanTidakMenulisApaPun(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(exactQ("decision_lock")).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}))
	mock.ExpectRollback()

	_, err = NewRepo(db).Decide(context.Background(), inboxrcl.DecisionCommand{
		Operator: "JONNY", ClaimNumber: "PNCN.26.28", Decision: inboxrcl.DecisionApprove, At: time.Now(),
	})
	require.ErrorIs(t, err, inboxrcl.ErrClaimNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestPICTeknikKosongMenolak — klaim tidak dipindahkan ke worklist tanpa pemilik.
func TestPICTeknikKosongMenolak(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(exactQ("decision_lock")).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}).
			AddRow("PNCN.26.40", "3", "0", nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectQuery(exactQ("decision_technical_pic")).
		WillReturnRows(sqlmock.NewRows([]string{"p"}).AddRow(nil))
	mock.ExpectRollback()

	_, err = NewRepo(db).Decide(context.Background(), inboxrcl.DecisionCommand{
		Operator: "JONNY", ClaimNumber: "PNCN.26.40", Decision: inboxrcl.DecisionBackMSIG, At: time.Now(),
	})
	require.ErrorIs(t, err, inboxrcl.ErrTechnicalPICUnknown)
	require.NoError(t, mock.ExpectationsWereMet())
}
