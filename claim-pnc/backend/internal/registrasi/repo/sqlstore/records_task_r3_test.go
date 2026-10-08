package sqlstore

import (
	"context"
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

var r3Keys = registrasi.RecordKeys{Number: "PNCN.26.1", ID: "K1", Prefixed: "ASM-FW-GCNMFW-WORK PNCN.26.1"}

func TestClaimRecordsSurveys(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("survey_daftar")).WithArgs(r3Keys.Number, r3Keys.ID, r3Keys.Prefixed).
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).
			AddRow(" SV1 ", " INT ", " Surveyor ", at, " Lok ", " Obj ", " ObjLok ", " 1 ", " DONE ", " cat ", at).
			AddRow("SV2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	got, err := r.Surveys(context.Background(), r3Keys)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, registrasi.Survey{
		CaseID: "SV1", Type: "INT", SurveyorName: "Surveyor", Date: at, SurveyLocation: "Lok", ObjectName: "Obj",
		ObjectLocation: "ObjLok", Index: "1", Status: "DONE", Note: "cat", InputDate: at,
	}, got[0])
	require.True(t, got[1].Date.IsZero())
	r3RowFailures(t, mock, "survey_daftar", 11, func() error {
		_, err := r.Surveys(context.Background(), r3Keys)
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Jenis dokumen: kode bisnis kosong tanpa kueri; coverage dikelompokkan per jenis.
func TestClaimRecordsDocumentTypes(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	ctx := context.Background()

	got, err := r.DocumentTypes(ctx, " ")
	require.NoError(t, err)
	require.Nil(t, got)

	mock.ExpectQuery(be4Q("dokumen_coverage_daftar")).WithArgs("10140").
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(" 14901 ", " C1 ").AddRow("14901", "C2"))
	mock.ExpectQuery(be4Q("dokumen_jenis_daftar")).WithArgs("10140").
		WillReturnRows(sqlmock.NewRows(be4Cols(7)).AddRow(" REGISTER ", " 10064 ", " 14901 ", " PELAPORAN ", " 1 ", " OTHERS ", " 1 ").
			AddRow("SURVEY", "10065", "14903", "LAPORAN", "0", "", "0"))
	got, err = r.DocumentTypes(ctx, " 10140 ")
	require.NoError(t, err)
	require.Equal(t, []registrasi.DocumentType{
		{Category: "REGISTER", CategoryID: "10064", ID: "14901", Name: "PELAPORAN", RequiredRaw: "1",
			ObjectDocID: "OTHERS", MinDoc: "1", Coverage: []string{"C1", "C2"}},
		{Category: "SURVEY", CategoryID: "10065", ID: "14903", Name: "LAPORAN", RequiredRaw: "0", MinDoc: "0"},
	}, got)

	// Galat pada pembacaan coverage.
	mock.ExpectQuery(be4Q("dokumen_coverage_daftar")).WillReturnError(be4Boom)
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorContains(t, err, "membaca coverage dokumen")
	mock.ExpectQuery(be4Q("dokumen_coverage_daftar")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorContains(t, err, "membaca baris coverage dokumen")
	mock.ExpectQuery(be4Q("dokumen_coverage_daftar")).
		WillReturnRows(sqlmock.NewRows(be4Cols(2)).AddRow(nil, nil).RowError(0, be4Boom))
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorContains(t, err, "menelusuri coverage dokumen")

	// Galat pada pembacaan jenis dokumen.
	coverageOK := func() {
		mock.ExpectQuery(be4Q("dokumen_coverage_daftar")).WillReturnRows(sqlmock.NewRows(be4Cols(2)))
	}
	coverageOK()
	mock.ExpectQuery(be4Q("dokumen_jenis_daftar")).WillReturnError(be4Boom)
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorContains(t, err, "membaca jenis dokumen")
	coverageOK()
	mock.ExpectQuery(be4Q("dokumen_jenis_daftar")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorContains(t, err, "membaca baris jenis dokumen")
	coverageOK()
	mock.ExpectQuery(be4Q("dokumen_jenis_daftar")).
		WillReturnRows(sqlmock.NewRows(be4Cols(7)).AddRow(make([]driver.Value, 7)...).RowError(0, be4Boom))
	_, err = r.DocumentTypes(ctx, "10140")
	require.ErrorIs(t, err, be4Boom)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRecordsAttachments(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("lampiran_daftar")).WithArgs(r3Keys.Number, r3Keys.ID, r3Keys.Prefixed).
		WillReturnRows(sqlmock.NewRows(be4Cols(9)).
			AddRow(" 1 ", " lod.pdf ", " pdf ", " n ", " 10064 ", " 14901 ", " IMG ", " NIK1 ", at).
			AddRow("2", nil, nil, nil, nil, nil, nil, nil, nil))
	got, err := r.Attachments(context.Background(), r3Keys)
	require.NoError(t, err)
	require.Equal(t, registrasi.Attachment{
		ID: "1", Name: "lod.pdf", MimeType: "pdf", Note: "n", Category: "10064", SubCategory: "14901",
		// INPUTDATE dibaca sebagai jam dinding WIB.
		ImageID: "IMG", UploadedBy: "NIK1", UploadedAt: time.Date(2026, 6, 9, 0, 0, 0, 0, clock.ZoneWIB),
	}, got[0])
	require.True(t, got[1].UploadedAt.IsZero())
	r3RowFailures(t, mock, "lampiran_daftar", 9, func() error {
		_, err := r.Attachments(context.Background(), r3Keys)
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestClaimRecordsProgress(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	next := at.Add(24 * time.Hour)
	mock.ExpectQuery(be4Q("progres_daftar")).WithArgs(r3Keys.Number, r3Keys.ID, r3Keys.Prefixed).
		WillReturnRows(sqlmock.NewRows(be4Cols(10)).
			AddRow(int64(3), at, " 014 ", " Akseptasi ", " 60 ", " Done ", " n ", next, " NIK1 ", " 5 ").
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	got, err := r.Progress(context.Background(), r3Keys)
	require.NoError(t, err)
	require.Equal(t, registrasi.ProgressEntry{
		Seq: 3, InputAt: at, Status1: "014", Status1Name: "Akseptasi", Status2: "60", Status2Name: "Done",
		Note: "n", NextFollowUp: next, InputBy: "NIK1", PositionID: "5",
	}, got[0])
	require.Equal(t, registrasi.ProgressEntry{}, got[1])
	r3RowFailures(t, mock, "progres_daftar", 10, func() error {
		_, err := r.Progress(context.Background(), r3Keys)
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Komunikasi memakai kunci klaim dua kali (dua cabang UNION).
func TestClaimRecordsCommunications(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("komunikasi_daftar")).
		WithArgs(r3Keys.Number, r3Keys.ID, r3Keys.Prefixed, r3Keys.Number, r3Keys.ID, r3Keys.Prefixed).
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).
			AddRow(" KM1 ", int64(7), at, " NIK1 ", " Budi ", " halo ", " ya ", " Ani ", at, " 1 ", " SENDTOINPUTOR ").
			AddRow(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	got, err := r.Communications(context.Background(), r3Keys)
	require.NoError(t, err)
	require.Equal(t, registrasi.Communication{
		CaseID: "KM1", ID: "7", SentAt: at, Sender: "NIK1", SenderName: "Budi", Message: "halo",
		Reply: "ya", ReplierName: "Ani", RepliedAt: at, Status: "1", Channel: registrasi.ChannelSendToInputor,
	}, got[0])
	require.Equal(t, registrasi.Communication{}, got[1])
	r3RowFailures(t, mock, "komunikasi_daftar", 11, func() error {
		_, err := r.Communications(context.Background(), r3Keys)
		return err
	})
	require.NoError(t, mock.ExpectationsWereMet())
}

// Catatan "Kirim ke Inputor": kunci klaim di CASEID dan CASECLAIM, nama kosong menjadi NULL.
func TestClaimRecordsAddCommunication(t *testing.T) {
	db, mock := be4DB(t)
	r := NewClaimRecords(db)
	c := registrasi.NewCommunication{
		ClaimID: "K1", ClaimNumber: "PNCN.26.1", Sender: "NIK1", Message: "lengkapi KTP",
		Recipient: "ADMIN1", Channel: registrasi.ChannelSendToInputor, Status: registrasi.CommunicationStatusOpen,
	}
	mock.ExpectExec(be4Q("komunikasi_sisip")).
		WithArgs("K1", "K1", "NIK1", nil, "lengkapi KTP", "0", "ADMIN1", "SENDTOINPUTOR").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, r.AddCommunication(context.Background(), c))

	mock.ExpectExec(be4Q("komunikasi_sisip")).WillReturnError(be4Boom)
	err := r.AddCommunication(context.Background(), c)
	require.ErrorIs(t, err, be4Boom)
	require.ErrorContains(t, err, "menyimpan komunikasi klaim PNCN.26.1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func r3TaskRow(rows *sqlmock.Rows, id, owner string, created time.Time, completed any) *sqlmock.Rows {
	return rows.AddRow(id, "K1", "PNCN.26.1", registrasi.StageInputRegister, "WORKLIST", nil, owner, created, created, completed, nil)
}

func TestTaskStoreGetAndOpen(t *testing.T) {
	db, mock := be4DB(t)
	s := NewTaskStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(be4Q("tugas_ambil")).WithArgs("T1").WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T1", "NIK1", at, at))
	task, err := s.Get(ctx, "T1")
	require.NoError(t, err)
	require.Equal(t, "T1", task.ID)
	require.Equal(t, "PNCN.26.1", task.ClaimNumber)
	require.Equal(t, registrasi.QueueKind("WORKLIST"), task.Queue)
	require.Equal(t, "NIK1", task.Owner)
	require.NotNil(t, task.ClaimedAt)
	require.NotNil(t, task.CompletedAt)
	require.False(t, task.Open())

	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	_, err = s.Get(ctx, "T1")
	require.ErrorIs(t, err, registrasi.ErrTaskNotFound)
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnError(be4Boom)
	_, err = s.Get(ctx, "T1")
	require.ErrorContains(t, err, "membaca tugas")

	mock.ExpectQuery(be4Q("tugas_terbuka_klaim")).WithArgs("K1").WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T2", "", at, nil))
	task, err = s.OpenTaskForClaim(ctx, "K1")
	require.NoError(t, err)
	require.True(t, task.Open())
	require.Nil(t, task.CompletedAt)
	mock.ExpectQuery(be4Q("tugas_terbuka_klaim")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	_, err = s.OpenTaskForClaim(ctx, "K1")
	require.ErrorIs(t, err, registrasi.ErrTaskNotFound)
	mock.ExpectQuery(be4Q("tugas_terbuka_klaim")).WillReturnError(be4Boom)
	_, err = s.OpenTaskForClaim(ctx, "K1")
	require.ErrorContains(t, err, "membaca tugas terbuka klaim")
	require.NoError(t, mock.ExpectationsWereMet())
}

func r3Task() registrasi.Task {
	return registrasi.Task{
		ID: "T1", ClaimID: "K1", ClaimNumber: "PNCN.26.1", Stage: registrasi.StageInputRegister,
		Queue: registrasi.QueueKind("WORKLIST"), Owner: "NIK1", CreatedAt: time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC),
	}
}

// Simpan: pembaruan yang mengenai baris selesai; bila tidak, tugas baru disisipkan.
func TestTaskStoreSave(t *testing.T) {
	db, mock := be4DB(t)
	s := NewTaskStore(db)
	ctx := context.Background()
	task := r3Task()

	mock.ExpectExec(be4Q("tugas_perbarui")).WithArgs("PNCN.26.1", "NIK1", nil, nil, nil, "T1", "NIK1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(ctx, task))

	mock.ExpectExec(be4Q("tugas_perbarui")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(be4Q("tugas_ambil")).WithArgs("T1").WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	mock.ExpectExec(be4Q("tugas_sisip")).
		WithArgs("PNCN.26.1", "NIK1", nil, nil, nil, "T1", "K1", registrasi.StageInputRegister, "WORKLIST", nil, task.CreatedAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, s.Save(ctx, task))
	require.NoError(t, mock.ExpectationsWereMet())
}

// Baris yang tidak terbarui karena sudah selesai atau dimiliki orang lain ditolak.
func TestTaskStoreSaveConflicts(t *testing.T) {
	db, mock := be4DB(t)
	s := NewTaskStore(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	task := r3Task()

	notUpdated := func() { mock.ExpectExec(be4Q("tugas_perbarui")).WillReturnResult(sqlmock.NewResult(0, 0)) }

	notUpdated()
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T1", "NIK1", at, at))
	require.ErrorIs(t, s.Save(ctx, task), registrasi.ErrTaskAlreadyDone)

	notUpdated()
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T1", "LAIN", at, nil))
	require.ErrorIs(t, s.Save(ctx, task), registrasi.ErrTaskAlreadyClaimed)

	// Baris ada, terbuka, pemilik sama (tidak ada yang berubah): selesai tanpa sisip.
	notUpdated()
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T1", "NIK1", at, nil))
	require.NoError(t, s.Save(ctx, task))

	notUpdated()
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnError(be4Boom)
	require.ErrorIs(t, s.Save(ctx, task), be4Boom)

	mock.ExpectExec(be4Q("tugas_perbarui")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(ctx, task), "memperbarui tugas")

	mock.ExpectExec(be4Q("tugas_perbarui")).WillReturnResult(sqlmock.NewErrorResult(be4Boom))
	require.ErrorContains(t, s.Save(ctx, task), "membaca jumlah baris tugas")

	notUpdated()
	mock.ExpectQuery(be4Q("tugas_ambil")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	mock.ExpectExec(be4Q("tugas_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, s.Save(ctx, task), "menyisipkan tugas")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Inbox menggabungkan milik sendiri, antrean, dan tahap grup tanpa duplikat, urut waktu.
func TestTaskStoreInbox(t *testing.T) {
	db, mock := be4DB(t)
	s := NewTaskStore(db)
	ctx := context.Background()
	early := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(be4Q("tugas_inbox_milik_saya")).WithArgs("NIK1").
		WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T3", "NIK1", late, nil))
	mock.ExpectQuery(be4Q("tugas_inbox_antrean")).WithArgs("WB1").
		WillReturnRows(r3TaskRow(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T3", "NIK1", late, nil), "T2", "", late, nil))
	mock.ExpectQuery(be4Q("tugas_inbox_tahap_grup")).WithArgs("ST1").
		WillReturnRows(r3TaskRow(sqlmock.NewRows(be4Cols(11)), "T1", "", early, nil))
	got, err := s.Inbox(ctx, "NIK1", []string{"WB1"}, []string{"ST1"})
	require.NoError(t, err)
	ids := []string{}
	for _, task := range got {
		ids = append(ids, task.ID)
	}
	require.Equal(t, []string{"T1", "T2", "T3"}, ids)

	mock.ExpectQuery(be4Q("tugas_inbox_milik_saya")).WillReturnError(be4Boom)
	_, err = s.Inbox(ctx, "NIK1", nil, nil)
	require.ErrorContains(t, err, "membaca inbox")

	mock.ExpectQuery(be4Q("tugas_inbox_milik_saya")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	mock.ExpectQuery(be4Q("tugas_inbox_antrean")).WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))
	_, err = s.Inbox(ctx, "NIK1", []string{"WB1"}, nil)
	require.ErrorContains(t, err, "membaca baris inbox")

	mock.ExpectQuery(be4Q("tugas_inbox_milik_saya")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	mock.ExpectQuery(be4Q("tugas_inbox_antrean")).WillReturnRows(sqlmock.NewRows(be4Cols(11)))
	mock.ExpectQuery(be4Q("tugas_inbox_tahap_grup")).
		WillReturnRows(sqlmock.NewRows(be4Cols(11)).AddRow(make([]driver.Value, 11)...).RowError(0, be4Boom))
	_, err = s.Inbox(ctx, "NIK1", []string{"WB1"}, []string{"ST1"})
	require.ErrorContains(t, err, "menelusuri inbox")
	require.NoError(t, mock.ExpectationsWereMet())
}
