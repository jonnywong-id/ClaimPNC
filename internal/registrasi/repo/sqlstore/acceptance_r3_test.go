package sqlstore

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// r3Tx membuka transaksi tiruan dan menitipkannya di context, seperti UnitOfWork.Run.
func r3Tx(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) (context.Context, *sql.Tx) {
	t.Helper()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	return context.WithValue(context.Background(), txKey, tx), tx
}

// Nomor akseptasi hanya terbit di dalam transaksi.
func TestAcceptanceNextNumberNeedsTransaction(t *testing.T) {
	db, mock := be4DB(t)
	_, err := NewAcceptanceStore(db).NextNumber(context.Background(), 2026)
	require.ErrorContains(t, err, "hanya boleh diterbitkan di dalam transaksi")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Nomor ALOD: kode + tahun + site + urutan 15 digit, dan barisnya dicatat.
func TestAcceptanceNextNumberIssues(t *testing.T) {
	db, mock := be4DB(t)
	ctx, _ := r3Tx(t, db, mock)
	mock.ExpectQuery(be4Q("akseptasi_site")).WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow(" 1 "))
	mock.ExpectQuery(be4Q("akseptasi_urut")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(42))
	mock.ExpectExec(be4Q("akseptasi_nomor_sisip")).
		WithArgs(sqlmock.AnyArg(), registrasi.AcceptanceCode, "1", "26", "42").
		WillReturnResult(sqlmock.NewResult(0, 1))

	number, err := NewAcceptanceStore(db).NextNumber(ctx, 2026)
	require.NoError(t, err)
	require.Equal(t, registrasi.PLANumber(registrasi.AcceptanceCode, 2026, "1", 42), number)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Setiap langkah penerbitan nomor yang gagal menghentikannya dengan galat berkonteks.
func TestAcceptanceNextNumberFailures(t *testing.T) {
	for step, want := range map[int]string{0: "membaca site aktif", 1: "membaca ACCEPTLOD_SEQ", 2: "mencatat nomor akseptasi"} {
		db, mock := be4DB(t)
		ctx, _ := r3Tx(t, db, mock)
		site := mock.ExpectQuery(be4Q("akseptasi_site"))
		if step == 0 {
			site.WillReturnError(be4Boom)
		} else {
			site.WillReturnRows(sqlmock.NewRows([]string{"s"}).AddRow("1"))
			seq := mock.ExpectQuery(be4Q("akseptasi_urut"))
			if step == 1 {
				seq.WillReturnError(be4Boom)
			} else {
				seq.WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(1))
				mock.ExpectExec(be4Q("akseptasi_nomor_sisip")).WillReturnError(be4Boom)
			}
		}
		_, err := NewAcceptanceStore(db).NextNumber(ctx, 2026)
		require.ErrorIs(t, err, be4Boom)
		require.ErrorContains(t, err, want)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func r3AcceptedLine() registrasi.SettlementLine {
	day := time.Date(2026, 6, 9, 0, 0, 0, 0, clock.ZoneWIB)
	return registrasi.SettlementLine{
		AcceptedNo: "A26100001",
		Acceptance: registrasi.Acceptance{
			AcceptedAt: day, ReceiverName: "Budi",
			Form: registrasi.AcceptanceForm{
				LODStatus: "1", ReceiverID: "R1", ReceiveDate: day, PayableDate: day,
				LODValue: 5_000, HasLODValue: true, Type: "2", CommitteeName: "KOMITE02",
				Remark: "OK", MinutesNote: "BA",
			},
		},
	}
}

// Simpan menulis kolom lama lalu tujuh kolom migrasi 0013 dengan kunci baris yang sama.
func TestAcceptanceSave(t *testing.T) {
	db, mock := be4DB(t)
	wall := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(be4Q("akseptasi_simpan")).
		WithArgs("1", "A26100001", wall, "R1", "Budi", nil, wall, "K1", "OBJ1", "2", "3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("akseptasi_isian_simpan")).
		WithArgs(wall, nil, int64(5_000), "2", "KOMITE02", "OK", "BA", "K1", "OBJ1", "2", "3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).Save(context.Background(), "K1", "OBJ1", 2, 3, r3AcceptedLine()))
	require.NoError(t, mock.ExpectationsWereMet())

	// Tanpa nilai LOD, kolomnya ditulis NULL.
	db, mock = be4DB(t)
	line := r3AcceptedLine()
	line.Acceptance.Form.HasLODValue = false
	mock.ExpectExec(be4Q("akseptasi_simpan")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("akseptasi_isian_simpan")).
		WithArgs(wall, nil, nil, "2", "KOMITE02", "OK", "BA", "K1", "OBJ1", "2", "3").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).Save(context.Background(), "K1", "OBJ1", 2, 3, line))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceSaveFailures(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectExec(be4Q("akseptasi_simpan")).WillReturnError(be4Boom)
	err := NewAcceptanceStore(db).Save(context.Background(), "K1", "O", 1, 1, r3AcceptedLine())
	require.ErrorIs(t, err, be4Boom)
	require.ErrorContains(t, err, "menyimpan akseptasi")

	db, mock = be4DB(t)
	mock.ExpectExec(be4Q("akseptasi_simpan")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("akseptasi_isian_simpan")).WillReturnError(be4Boom)
	err = NewAcceptanceStore(db).Save(context.Background(), "K1", "O", 1, 1, r3AcceptedLine())
	require.ErrorIs(t, err, be4Boom)
	require.ErrorContains(t, err, "kolom migrasi 0013")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceRecordLODPrint(t *testing.T) {
	db, mock := be4DB(t)
	at := time.Date(2026, 6, 9, 3, 0, 0, 0, time.UTC) // 10:00 WIB
	mock.ExpectExec(be4Q("lod_cetak_simpan")).
		WithArgs(time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC), "4", "K1", "O", "1", "2").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).RecordLODPrint(context.Background(), "K1", "O", 1, 2, at, "4"))

	mock.ExpectExec(be4Q("lod_cetak_simpan")).WillReturnError(be4Boom)
	err := NewAcceptanceStore(db).RecordLODPrint(context.Background(), "K1", "O", 1, 2, time.Time{}, "")
	require.ErrorContains(t, err, "mencatat Print LOD")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Fields membaca tujuh isian; baris yang belum ada tidak mengubah apa pun.
func TestAcceptanceFields(t *testing.T) {
	db, mock := be4DB(t)
	payable := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("akseptasi_isian_ambil")).WithArgs("K1", "O", "1", "2").
		WillReturnRows(sqlmock.NewRows(be4Cols(7)).AddRow(payable, nil, int64(700), " 2 ", " KOMITE02 ", "catatan", "BA"))
	var line registrasi.SettlementLine
	require.NoError(t, NewAcceptanceStore(db).Fields(context.Background(), "K1", "O", 1, 2, &line))
	f := line.Acceptance.Form
	require.Equal(t, registrasi.Money(700), f.LODValue)
	require.True(t, f.HasLODValue)
	require.Equal(t, "2", f.Type)
	require.Equal(t, "KOMITE02", f.CommitteeName)
	require.Equal(t, "catatan", f.Remark)
	require.Equal(t, "BA", f.MinutesNote)
	require.Equal(t, 9, f.PayableDate.In(clock.ZoneWIB).Day())
	require.True(t, f.AnalystReceiveDate.IsZero())

	mock.ExpectQuery(be4Q("akseptasi_isian_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(7)))
	untouched := registrasi.SettlementLine{AcceptedNo: "X"}
	require.NoError(t, NewAcceptanceStore(db).Fields(context.Background(), "K1", "O", 1, 2, &untouched))
	require.Equal(t, registrasi.SettlementLine{AcceptedNo: "X"}, untouched)

	mock.ExpectQuery(be4Q("akseptasi_isian_ambil")).WillReturnError(be4Boom)
	err := NewAcceptanceStore(db).Fields(context.Background(), "K1", "O", 1, 2, &line)
	require.ErrorContains(t, err, "membaca isian akseptasi")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceOtherDLA(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectQuery(be4Q("akseptasi_dla_lain")).WithArgs("K1", "O", "1", "2").
		WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(" D1 ", "1", "0").AddRow("D2", nil, "1"))
	got, err := NewAcceptanceStore(db).OtherDLA(context.Background(), "K1", "O", 1, 2)
	require.NoError(t, err)
	require.Equal(t, []registrasi.AcceptanceDLAState{
		{Number: "D1", Printed: true, Sent: false},
		{Number: "D2", Printed: false, Sent: true},
	}, got)

	for _, mode := range []be4Mode{be4Fail, be4RowErr} {
		be4Expect(mock, be4Step{name: "akseptasi_dla_lain", cols: 3}, mode)
		_, err = NewAcceptanceStore(db).OtherDLA(context.Background(), "K1", "O", 1, 2)
		require.ErrorIs(t, err, be4Boom, "mode %d", mode)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

// be4ScanFail memakai satu kolom sehingga Scan gagal tanpa be4Boom; ia diuji tersendiri.
func TestAcceptanceOtherDLAScanFailure(t *testing.T) {
	db, mock := be4DB(t)
	be4Expect(mock, be4Step{name: "akseptasi_dla_lain"}, be4ScanFail)
	_, err := NewAcceptanceStore(db).OtherDLA(context.Background(), "K1", "O", 1, 2)
	require.ErrorContains(t, err, "membaca baris DLA")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceAddHistory(t *testing.T) {
	db, mock := be4DB(t)
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(be4Q("riwayat_sisip")).
		WithArgs("C1", time.Date(2026, 6, 9, 7, 0, 0, 0, time.UTC), string(long[:150]), "U1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).AddHistory(context.Background(), "C1", string(long), "U1", at))

	mock.ExpectExec(be4Q("riwayat_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, NewAcceptanceStore(db).AddHistory(context.Background(), "C1", "n", "U1", at), "menulis riwayat klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProgressJSONEscapesQuotes(t *testing.T) {
	got := progressJSON(`Cabang "A"`)
	require.Contains(t, got, `"BranchName":"Cabang \"A\""`)
	require.Contains(t, got, `"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData"`)
}

func TestAcceptanceOpenPosition(t *testing.T) {
	db, mock := be4DB(t)
	mock.ExpectQuery(be4Q("progres_posisi_terbuka")).WithArgs("PNCN.26.1", "AKSEPTASI").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(" 9 "))
	id, err := NewAcceptanceStore(db).OpenPosition(context.Background(), "PNCN.26.1", "AKSEPTASI")
	require.NoError(t, err)
	require.Equal(t, "9", id)

	mock.ExpectQuery(be4Q("progres_posisi_terbuka")).WillReturnRows(sqlmock.NewRows([]string{"id"}))
	id, err = NewAcceptanceStore(db).OpenPosition(context.Background(), "PNCN.26.1", "AKSEPTASI")
	require.NoError(t, err)
	require.Empty(t, id)

	mock.ExpectQuery(be4Q("progres_posisi_terbuka")).WillReturnError(be4Boom)
	_, err = NewAcceptanceStore(db).OpenPosition(context.Background(), "PNCN.26.1", "AKSEPTASI")
	require.ErrorContains(t, err, "membaca posisi progres AKSEPTASI")
	require.NoError(t, mock.ExpectationsWereMet())
}

func r3ProgressStart() registrasi.ProgressStart {
	return registrasi.ProgressStart{
		ClaimNumber: "PNCN.26.1", CaseID: "C1", Position: "AKSEPTASI", Note: "catat",
		Progress1: "014", Progress2: "60", User: "NIK1",
		At: time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}
}

// StartProgress memakai identitas lama bila ada, menomori posisi, lalu menulis dua baris.
func TestAcceptanceStartProgress(t *testing.T) {
	db, mock := be4DB(t)
	p := r3ProgressStart()
	mock.ExpectQuery(be4Q("operator_lama")).WithArgs("NIK1").WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow(" LAMA1 "))
	mock.ExpectQuery(be4Q("progres_posisi_berikut")).WithArgs("PNCN.26.1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(5)))
	mock.ExpectExec(be4Q("progres_posisi_sisip")).
		WithArgs(int64(5), "PNCN.26.1", "C1", registrasi.ProgressOnProgress, sqlmock.AnyArg(), "AKSEPTASI").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("progres_sisip_buka")).
		WithArgs("PNCN.26.1", "catat", "014", "60", time.Date(2026, 6, 16, 7, 0, 0, 0, time.UTC), "LAMA1", int64(5), progressJSON("60"), "PNCN.26.1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	id, err := NewAcceptanceStore(db).StartProgress(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, "5", id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceStartProgressFailures(t *testing.T) {
	// Identitas lama gagal dibaca.
	db, mock := be4DB(t)
	mock.ExpectQuery(be4Q("operator_lama")).WillReturnError(be4Boom)
	_, err := NewAcceptanceStore(db).StartProgress(context.Background(), r3ProgressStart())
	require.ErrorContains(t, err, "membaca identitas lama pengguna")

	// Penomoran posisi gagal; identitas lama kosong memakai pengguna sendiri.
	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}).AddRow(""))
	mock.ExpectQuery(be4Q("progres_posisi_berikut")).WillReturnError(be4Boom)
	_, err = NewAcceptanceStore(db).StartProgress(context.Background(), r3ProgressStart())
	require.ErrorContains(t, err, "menomori posisi progres")

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectQuery(be4Q("progres_posisi_berikut")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(1)))
	mock.ExpectExec(be4Q("progres_posisi_sisip")).WillReturnError(be4Boom)
	_, err = NewAcceptanceStore(db).StartProgress(context.Background(), r3ProgressStart())
	require.ErrorContains(t, err, "membuka posisi progres AKSEPTASI")

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectQuery(be4Q("progres_posisi_berikut")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(1)))
	mock.ExpectExec(be4Q("progres_posisi_sisip")).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("progres_sisip_buka")).
		WithArgs("PNCN.26.1", "catat", "014", "60", sqlmock.AnyArg(), "NIK1", int64(1), sqlmock.AnyArg(), "PNCN.26.1").
		WillReturnError(be4Boom)
	_, err = NewAcceptanceStore(db).StartProgress(context.Background(), r3ProgressStart())
	require.ErrorContains(t, err, "menulis progres klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAcceptanceAddProgress(t *testing.T) {
	db, mock := be4DB(t)
	p := registrasi.ProgressUpdate{
		ClaimNumber: "PNCN.26.1", PositionID: " 5 ", Position: "AKSEPTASI", Note: "n",
		Progress1: "014", Progress2: "60", User: "NIK1", At: time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}
	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectExec(be4Q("progres_posisi_selesai")).WithArgs("AKSEPTASI", sqlmock.AnyArg(), "PNCN.26.1", "5").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(be4Q("progres_sisip")).
		WithArgs("PNCN.26.1", "n", "014", "60", "NIK1", "5", progressJSON("60"), "PNCN.26.1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).AddProgress(context.Background(), p))

	// Tanpa posisi: tidak ada posisi yang ditutup, POSISIID ditulis NULL.
	noPos := p
	noPos.PositionID = ""
	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectExec(be4Q("progres_sisip")).
		WithArgs("PNCN.26.1", "n", "014", "60", "NIK1", nil, sqlmock.AnyArg(), "PNCN.26.1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, NewAcceptanceStore(db).AddProgress(context.Background(), noPos))

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnError(be4Boom)
	require.ErrorIs(t, NewAcceptanceStore(db).AddProgress(context.Background(), p), be4Boom)

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectExec(be4Q("progres_posisi_selesai")).WillReturnError(be4Boom)
	require.ErrorContains(t, NewAcceptanceStore(db).AddProgress(context.Background(), p), "menutup posisi progres")

	mock.ExpectQuery(be4Q("operator_lama")).WillReturnRows(sqlmock.NewRows([]string{"o"}))
	mock.ExpectExec(be4Q("progres_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, NewAcceptanceStore(db).AddProgress(context.Background(), noPos), "menulis progres klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Tiga pembacaan premi: nilai, baris tidak ada, dan galat.
func TestAcceptancePremiumReads(t *testing.T) {
	db, mock := be4DB(t)
	s := NewAcceptanceStore(db)
	ctx := context.Background()

	// CaseID pada PRODKE klaim: T_GENERAL lebih dulu, dokumen POLICYDATA sebagai cadangan.
	mock.ExpectQuery(be4Q("premi_polis_caseid")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(" CASE1 "))
	v, err := s.PolicyCaseID(ctx, "POL", "1")
	require.NoError(t, err)
	require.Equal(t, "CASE1", v)
	mock.ExpectQuery(be4Q("premi_polis_caseid")).WillReturnRows(sqlmock.NewRows([]string{"c"}))
	mock.ExpectQuery(be4Q("premi_polis_caseid_dokumen")).WithArgs("POL", "1").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow("CASE2"))
	v, err = s.PolicyCaseID(ctx, "POL", "1")
	require.NoError(t, err)
	require.Equal(t, "CASE2", v)
	mock.ExpectQuery(be4Q("premi_polis_caseid")).WillReturnRows(sqlmock.NewRows([]string{"c"}))
	mock.ExpectQuery(be4Q("premi_polis_caseid_dokumen")).WillReturnRows(sqlmock.NewRows([]string{"c"}))
	v, err = s.PolicyCaseID(ctx, "POL", "1")
	require.NoError(t, err)
	require.Empty(t, v)
	mock.ExpectQuery(be4Q("premi_polis_caseid")).WillReturnError(be4Boom)
	_, err = s.PolicyCaseID(ctx, "POL", "1")
	require.ErrorContains(t, err, "membaca CaseID polis")

	mock.ExpectQuery(be4Q("premi_open_protection")).WithArgs("POL", "K1", "N1").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(2)))
	ok, err := s.OpenProtectionApproved(ctx, "POL", "K1", "N1")
	require.NoError(t, err)
	require.True(t, ok)
	mock.ExpectQuery(be4Q("premi_open_protection")).WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(0)))
	ok, err = s.OpenProtectionApproved(ctx, "POL", "K1", "N1")
	require.NoError(t, err)
	require.False(t, ok)
	mock.ExpectQuery(be4Q("premi_open_protection")).WillReturnError(be4Boom)
	_, err = s.OpenProtectionApproved(ctx, "POL", "K1", "N1")
	require.ErrorContains(t, err, "membaca Open Protection premi")

	mock.ExpectQuery(be4Q("premi_agen_travel")).WithArgs("POL").WillReturnRows(sqlmock.NewRows([]string{"c"}).AddRow(" AGEN "))
	name, err := s.TravelClientName(ctx, "POL")
	require.NoError(t, err)
	require.Equal(t, "AGEN", name)
	mock.ExpectQuery(be4Q("premi_agen_travel")).WillReturnRows(sqlmock.NewRows([]string{"c"}))
	name, err = s.TravelClientName(ctx, "POL")
	require.NoError(t, err)
	require.Empty(t, name)
	mock.ExpectQuery(be4Q("premi_agen_travel")).WillReturnError(be4Boom)
	_, err = s.TravelClientName(ctx, "POL")
	require.ErrorContains(t, err, "membaca agen polis Travel")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestWallOrNil(t *testing.T) {
	require.Nil(t, wallOrNil(time.Time{}))
	got := wallOrNil(time.Date(2026, 6, 9, 20, 30, 15, 99, time.UTC))
	require.Equal(t, time.Date(2026, 6, 10, 3, 30, 15, 0, time.UTC), got)
	var _ driver.Value = got
}
