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

	"claim-pnc/internal/dashboardclaim"
)

var (
	claimColumns = []string{
		"CLAIM_ID", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME", "BUSINESS_NAME",
		"BUSINESS_SOURCE", "BRANCH_NAME", "TECHNICAL_PIC", "ADMIN_PNC", "CLAIM_STATUS",
		"CLAIM_STATUS_LABEL", "PROCESS_STATUS", "LOSS_DATE", "REPORT_DATE", "REGISTERED_AT",
	}
	surveyColumns = []string{
		"SURVEY_ID", "SURVEY_NUMBER", "CLAIM_NUMBER", "POLICY_NUMBER", "INSURED_NAME",
		"REFERENCE_NUMBER", "SURVEYOR_NAME", "TECHNICAL_PIC", "SURVEY_LOCATION",
		"SURVEY_STATUS", "PROCESS_STATUS", "ADJUSTER_PIC", "SCHEDULED_AT", "ASSIGNED_AT",
	}
)

func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return NewRepo(db), mock
}

func exact(name string) string { return regexp.QuoteMeta(query(name)) }

// Hitungan klaim berjalan mengirim penyaring dengan urutan bind kueri itu sendiri.
func TestCountOutstandingBindsFilters(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Kotak cari dikirim apa adanya (sudah dipangkas), polanya dibesarkan dan wildcard-nya
	// di-escape; paginasi tidak ikut pada kueri hitung.
	mock.ExpectQuery(exact("outstanding_count")).
		WithArgs("pol_1", `%POL\_1%`, `%POL\_1%`, "PA", "PA", "PA", "PA", "PA",
			// Kedua belas penanda panel penyaring kosong: ujinya hanya mengisi kotak cari.
			nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(12))

	total, err := repo.CountOutstanding(context.Background(),
		dashboardclaim.Filter{Business: dashboardclaim.BusinessPA, Search: " pol_1 ", Limit: 5, Offset: 10})
	require.NoError(t, err)
	require.Equal(t, 12, total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountOutstandingError(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("outstanding_count")).
		WithArgs(nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
			nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil).
		WillReturnError(sql.ErrConnDone)

	_, err := repo.CountOutstanding(context.Background(), dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "dashboardclaim/sqlstore: menghitung klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Daftar klaim berjalan memetakan setiap kolom, termasuk NULL menjadi teks kosong dan
// tanggal nil.
func TestListOutstandingMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	loss := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	// Tanggal lapor kini T_CLAIM_PNC.RECEIVEDATE — kolom DATE, bukan teks bergaya Pega.
	report := time.Date(2026, time.September, 2, 0, 0, 0, 0, time.UTC)
	registered := time.Date(2026, time.September, 2, 3, 0, 0, 0, time.UTC)

	// Delapan penyaring lama ditambah dua belas penanda panel — Outstanding sendiri yang
	// memilikinya; kueri survei tetap delapan.
	filters := []driver.Value{nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil}
	mock.ExpectQuery(exact("outstanding_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(30))
	mock.ExpectQuery(exact("outstanding_list")).WithArgs(append(filters, 25, 25)...).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID1", "PNCN.26.0001", "POL", "Tertanggung", "Bisnis", "Sumber", "Cabang",
				"PIC", "Admin", "1147", "Register", "Open", loss, report, registered).
			AddRow("ID2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := repo.ListOutstanding(context.Background(), dashboardclaim.Filter{Offset: 25})
	require.NoError(t, err)
	require.Equal(t, 30, page.Total)
	require.Equal(t, []dashboardclaim.ClaimRow{
		{
			ClaimID: "ID1", ClaimNumber: "PNCN.26.0001", PolicyNumber: "POL", InsuredName: "Tertanggung",
			BusinessName: "Bisnis", BusinessSource: "Sumber", BranchName: "Cabang", TechnicalPIC: "PIC",
			AdminPNC: "Admin", ClaimStatusCode: "1147", ClaimStatusLabel: "Register",
			ProcessStatus: "Open", LossDate: &loss, ReportDate: &report, RegisteredAt: registered,
		},
		{ClaimID: "ID2"},
	}, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOutstandingErrors(t *testing.T) {
	filters := []driver.Value{nil, nil, nil, "ALL", "ALL", "ALL", "ALL", "ALL",
		nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil}
	ctx := context.Background()

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).WillReturnError(sql.ErrConnDone)
	_, err := repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), "menghitung klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).WillReturnError(sql.ErrTxDone)
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrTxDone)
	require.Contains(t, err.Error(), "membaca daftar klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID", "N", "P", "I", "B", "S", "C", "T", "A", "K", "L", "O", "bukan-tanggal", nil, nil))
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris klaim")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("outstanding_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("outstanding_list")).
		WillReturnRows(sqlmock.NewRows(claimColumns).
			AddRow("ID", "N", "P", "I", "B", "S", "C", "T", "A", "K", "L", "O", nil, nil, nil).
			RowError(0, boom))
	_, err = repo.ListOutstanding(ctx, dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), "menutup daftar klaim berjalan")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Hitungan survei memilih kueri menurut jenis surveyor, dengan urutan bind penyaring umum.
func TestCountSurveyPicksQueryPerKind(t *testing.T) {
	cases := map[dashboardclaim.SurveyorType]string{
		dashboardclaim.SurveyorAdjuster: "loss_adjuster_count",
		dashboardclaim.SurveyorInternal: "internal_surveyor_count",
	}
	for kind, name := range cases {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(exact(name)).
			WithArgs("TRAVEL", "TRAVEL", "TRAVEL", "TRAVEL", "TRAVEL", nil, nil, nil).
			WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(4))

		total, err := repo.CountSurvey(context.Background(), kind,
			dashboardclaim.Filter{Business: dashboardclaim.BusinessTravel})
		require.NoError(t, err)
		require.Equal(t, 4, total)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func TestSurveyUnknownKindTouchesNothing(t *testing.T) {
	repo, mock := newMockRepo(t)

	_, err := repo.CountSurvey(context.Background(), "9", dashboardclaim.Filter{})
	require.EqualError(t, err, `dashboardclaim/sqlstore: jenis surveyor "9" tidak dikenal`)
	_, err = repo.ListSurvey(context.Background(), "9", dashboardclaim.Filter{})
	require.EqualError(t, err, `dashboardclaim/sqlstore: jenis surveyor "9" tidak dikenal`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountSurveyError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).WillReturnError(sql.ErrConnDone)

	_, err := repo.CountSurvey(context.Background(), dashboardclaim.SurveyorAdjuster, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.Contains(t, err.Error(), `menghitung survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSurveyMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	scheduled := time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC)
	assigned := time.Date(2026, time.September, 10, 3, 0, 0, 0, time.UTC)

	filters := []driver.Value{"ALL", "ALL", "ALL", "ALL", "ALL", nil, nil, nil}
	mock.ExpectQuery(exact("internal_surveyor_count")).WithArgs(filters...).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(2))
	mock.ExpectQuery(exact("internal_surveyor_list")).WithArgs(append(filters, 0, 10)...).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S1", "SRV-1", "PNCN.26.0001", "POL", "Tertanggung", "REF", "Surveyor",
				"PIC", "Jakarta", "Scheduled", "Open", "PIC Adj", scheduled, assigned).
			AddRow("S2", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	page, err := repo.ListSurvey(context.Background(), dashboardclaim.SurveyorInternal,
		dashboardclaim.Filter{Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 2, page.Total)
	require.Equal(t, []dashboardclaim.SurveyRow{
		{
			SurveyID: "S1", SurveyNumber: "SRV-1", ClaimNumber: "PNCN.26.0001", PolicyNumber: "POL",
			InsuredName: "Tertanggung", ReferenceNumber: "REF", SurveyorName: "Surveyor",
			TechnicalPIC: "PIC", SurveyLocation: "Jakarta", SurveyStatus: "Scheduled",
			ProcessStatus: "Open", AdjusterPIC: "PIC Adj", ScheduledAt: &scheduled, AssignedAt: assigned,
		},
		{SurveyID: "S2"},
	}, page.Rows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListSurveyErrors(t *testing.T) {
	ctx := context.Background()
	kind := dashboardclaim.SurveyorAdjuster

	repo, mock := newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).WillReturnError(sql.ErrConnDone)
	_, err := repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).WillReturnError(sql.ErrTxDone)
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, sql.ErrTxDone)
	require.Contains(t, err.Error(), `membaca daftar survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())

	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S", "N", "C", "P", "I", "R", "S", "T", "L", "St", "O", "A", "bukan-tanggal", nil))
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "membaca baris survei")
	require.NoError(t, mock.ExpectationsWereMet())

	boom := errors.New("koneksi putus")
	repo, mock = newMockRepo(t)
	mock.ExpectQuery(exact("loss_adjuster_count")).
		WillReturnRows(sqlmock.NewRows([]string{"TOTAL"}).AddRow(1))
	mock.ExpectQuery(exact("loss_adjuster_list")).
		WillReturnRows(sqlmock.NewRows(surveyColumns).
			AddRow("S", "N", "C", "P", "I", "R", "S", "T", "L", "St", "O", "A", nil, nil).
			RowError(0, boom))
	_, err = repo.ListSurvey(ctx, kind, dashboardclaim.Filter{})
	require.ErrorIs(t, err, boom)
	require.Contains(t, err.Error(), `menutup daftar survei "2"`)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kolom kepala rincian, sesuai urutan SELECT pada `klaim_rincian`.
//
// Ditulis sekali dan dipakai kedua uji rincian: saat kolom isian "Registrasi" ditambahkan,
// dua daftar terpisah akan berbeda diam-diam, dan yang gagal hanyalah salah satunya.
var kolomKepalaRincian = []string{
	"NOMOR_KLAIM", "KLAIM_ID", "STATUS_PROSES", "STATUS_KLAIM",
	"PIC_TEKNIK", "ADMIN_PNC", "DIDAFTARKAN_PADA", "GROUP_PANEL", "JENIS_BISNIS", "CATATAN_INPUTOR",
	"TANGGAL_TERIMA_DOKUMEN", "TANGGAL_KEJADIAN", "TANGGAL_KELUAR_RAWAT_INAP", "TANGGAL_LAPOR",
	"TELEPON_PELAPOR", "STATUS_PELAPOR", "STATUS_PELAPOR_LAIN", "LOKASI_KEJADIAN",
	"KELURAHAN", "TERTANGGUNG_TKI", "CATATAN_KE_ANALYST", "CATATAN_KE_PIC_TEKNIS", "EMAIL_PELAPOR",
}

// Rincian klaim membaca KEPALA lalu kesembilan bagiannya, dan bagian kosong bukan galat.
//
// Sejak 2026-10-12 isinya datang dari tabel relasional, bukan dari dokumen JSON. Uji ini
// menjaga dua hal sekaligus: urutan pembacaannya, dan bahwa klaim tanpa isi tetap terbaca.
func TestFindClaimDetailReadsSections(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("KUNCI-1", "KUNCI-1").
		WillReturnRows(sqlmock.NewRows(kolomKepalaRincian).
			AddRow("PNCN.26.0001", "KUNCI-1", "Open", "1147", "PIC", "Admin", nil, "006", "Fire", "catatan",
				nil, nil, nil, nil, "0812", "Suami", "", "Lokasi", "Kelurahan", "0", "ke analis", "ke pic", "surel"))

	// Enam bagian ditaut kunci Pega, satu — progres — ditaut NOMOR klaim.
	mock.ExpectQuery(exact("rincian_penerima")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "ADDRESS", "NAMEOFBANK", "NOACCOUNT"}).
			AddRow("Penerima", "Alamat", "Bank Contoh", "123"))
	mock.ExpectQuery(exact("rincian_objek")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"OBJECTNAME", "LOKASI", "OBJECTPARTICIPANTSTATUS", "OBJECTIDCARD",
			"DATEOFBIRTH", "OBJECTJOB", "TYPENAME", "CHASISSNUMBER",
		}))
	mock.ExpectQuery(exact("rincian_coverage")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"COVERAGENAME", "COVERAGEID", "SUMTSI", "CURRENCY", "CAUSEOFLOSS", "OBJECTNAME",
		}))
	mock.ExpectQuery(exact("rincian_adjustment")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"NOAKSEPTASI", "TGLAKSEPTASI", "STATUSAKSEPTASI", "PAYMENTTYPE", "RECEIVERNAME",
			"CURRENCY", "NILAIAKSEPTASI", "PROPOSE_VALUE", "TOTAL_CLAIM",
			"TRANSFER_CASHIER_DATE", "CLAIM_STATUS_PAID",
		}))
	mock.ExpectQuery(exact("rincian_survey")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"SURVEYDATE", "SURVEYOR_NAME", "OBJECT_NAME", "RESCHEDULE_LOCATION",
			"STS_SURVEY", "INDEX_SURVEY", "REFNO", "KETERANGAN", "LOCATION_SURVEY", "LOSSTYPE",
		}))
	mock.ExpectQuery(exact("rincian_lampiran")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"CATEGORY", "ATTACHNAME", "INPUTDATE", "INPUTOPERATOR",
		}))
	mock.ExpectQuery(exact("rincian_progress")).WithArgs("PNCN.26.0001").
		WillReturnRows(sqlmock.NewRows([]string{
			"TGL_INPUT", "STATUS_PROGRESS1", "STATUS", "KETERANGAN", "USER_INPUT", "NAMA_TAHAP",
			"NEXT_FOLLOWUP",
		}))
	// Dua himpunan isian tunggal. Keduanya dijawab KOSONG di sini dengan sengaja: klaim yang
	// belum pernah diminta survei adalah keadaan yang sah, dan uji ini menjaga bahwa keadaan
	// itu tidak menjadi galat.
	mock.ExpectQuery(exact("rincian_permintaan_survey")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"REQ_NAME", "LOCATION", "NOTELP", "DATE_SURVEY",
			"BRANCH", "SURVEYOR", "EMAIL_SURVEYOR", "OBJECTNAME",
		}))
	mock.ExpectQuery(exact("rincian_pucl")).WithArgs("KUNCI-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"TGL_TERIMA_DOKUMEN_PUCL", "ALASAN_DOKTER_REJECT_RCL",
		}))

	detail, err := repo.FindClaimDetail(context.Background(), "KUNCI-1")
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0001", detail.ClaimNumber)

	// Isian tab "Registrasi" ikut terbaca dari kepala — bukan dari kueri tambahan.
	//
	// Dijaga di sini supaya kolomnya tidak dapat dilepas diam-diam: kueri tetap berjalan tanpa
	// ketiga belas kolom itu, dan yang hilang hanyalah isi layar.
	require.Equal(t, "0812", detail.Registration.ReporterPhone)
	require.Equal(t, "Suami", detail.Registration.ReporterRelation)
	require.Equal(t, "Lokasi", detail.Registration.LossLocation)
	require.Equal(t, "surel", detail.Registration.ReporterEmail)

	// Permintaan survei dan PUCL yang kosong menghasilkan isian kosong, BUKAN galat.
	require.Empty(t, detail.SurveyRequest.RequestorName)
	require.Nil(t, detail.PUCL.DocumentCompleteAt)
	require.Len(t, detail.Receivers, 1)
	require.Equal(t, "Penerima", detail.Receivers[0].Name)
	require.Empty(t, detail.Items, "bagian kosong bukan galat")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Kegagalan membaca SATU bagian dikembalikan, tidak ditelan menjadi bagian kosong.
//
// Menelannya akan memberitahu pengguna "klaim ini belum diisi" — pernyataan yang salah dan
// tidak dapat dibantah dari layar.
func TestFindClaimDetailPropagatesSectionFailure(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("KUNCI-2", "KUNCI-2").
		WillReturnRows(sqlmock.NewRows(kolomKepalaRincian).
			AddRow("PNCN.26.0002", "KUNCI-2", "Open", "1147", "PIC", "Admin", nil, "002", "PAYDI_PA", nil,
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))
	mock.ExpectQuery(exact("rincian_penerima")).WithArgs("KUNCI-2").
		WillReturnError(errors.New("ora-00942"))

	_, err := repo.FindClaimDetail(context.Background(), "KUNCI-2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "rincian_penerima")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Klaim yang tidak ada dijawab ErrClaimNotFound, bukan galat teknis.
func TestFindClaimDetailMissingClaim(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("TIDAK-ADA", "TIDAK-ADA").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.FindClaimDetail(context.Background(), "TIDAK-ADA")
	require.ErrorIs(t, err, dashboardclaim.ErrClaimNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Rincian dapat dicari lewat NOMOR klaim, dan bagiannya tetap memakai KUNCI PEGA.
//
// # Cacat yang dijaga di sini
//
// Halaman detail memakai nomor klaim di alamatnya (`D-22`: kunci teknis Pega tidak pernah
// menjadi alamat). Sampai 2026-10-12 kueri kepalanya hanya mencari `PZINSKEY`, sehingga
// halaman itu menjawab "Klaim tidak ditemukan" untuk setiap klaim — dilaporkan Work Owner
// pada `PNC-13`.
//
// Perbaikannya menambah cacat kedua yang lebih sulit terlihat: bila bagian-bagiannya tetap
// dicari memakai nilai yang DIKIRIM pemanggil, kepalanya terbaca tetapi seluruh isinya kosong,
// dan layar menyatakan "klaim ini belum punya isi rincian" — pernyataan yang salah dan tidak
// dapat dibantah dari layar.
//
// Uji ini menjaga keduanya sekaligus: dicari lewat nomor, dan bagiannya memakai kunci hasil.
func TestFindClaimDetailAcceptsClaimNumberAndResolvesKeyForSections(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Pemanggil mengirim NOMOR; kedua bind menerima nomor yang sama.
	mock.ExpectQuery(exact("klaim_rincian")).WithArgs("PNC-13", "PNC-13").
		WillReturnRows(sqlmock.NewRows(kolomKepalaRincian).
			AddRow("PNC-13", "ASM-FW-GCNMFW-WORK PNC-13", "Open", "1147", "PIC", "Admin", nil,
				"006", "Fire", "catatan",
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil))

	// Enam bagian memakai KUNCI PEGA dari hasil — bukan "PNC-13" yang dikirim.
	kunci := "ASM-FW-GCNMFW-WORK PNC-13"
	mock.ExpectQuery(exact("rincian_penerima")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{"NAME", "ADDRESS", "NAMEOFBANK", "NOACCOUNT"}))
	mock.ExpectQuery(exact("rincian_objek")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"OBJECTNAME", "LOKASI", "OBJECTPARTICIPANTSTATUS", "OBJECTIDCARD",
			"DATEOFBIRTH", "OBJECTJOB", "TYPENAME", "CHASISSNUMBER",
		}))
	mock.ExpectQuery(exact("rincian_coverage")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"COVERAGENAME", "COVERAGEID", "SUMTSI", "CURRENCY", "CAUSEOFLOSS", "OBJECTNAME",
		}))
	mock.ExpectQuery(exact("rincian_adjustment")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"NOAKSEPTASI", "TGLAKSEPTASI", "STATUSAKSEPTASI", "PAYMENTTYPE", "RECEIVERNAME",
			"CURRENCY", "NILAIAKSEPTASI", "PROPOSE_VALUE", "TOTAL_CLAIM",
			"TRANSFER_CASHIER_DATE", "CLAIM_STATUS_PAID",
		}))
	mock.ExpectQuery(exact("rincian_survey")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"SURVEYDATE", "SURVEYOR_NAME", "OBJECT_NAME", "RESCHEDULE_LOCATION",
			"STS_SURVEY", "INDEX_SURVEY", "REFNO", "KETERANGAN", "LOCATION_SURVEY", "LOSSTYPE",
		}))
	mock.ExpectQuery(exact("rincian_lampiran")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"CATEGORY", "ATTACHNAME", "INPUTDATE", "INPUTOPERATOR",
		}))

	// Progres memakai NOMOR — satu-satunya bagian yang begitu, dan itu terukur.
	mock.ExpectQuery(exact("rincian_progress")).WithArgs("PNC-13").
		WillReturnRows(sqlmock.NewRows([]string{
			"TGL_INPUT", "STATUS_PROGRESS1", "STATUS", "KETERANGAN", "USER_INPUT", "NAMA_TAHAP",
			"NEXT_FOLLOWUP",
		}))

	mock.ExpectQuery(exact("rincian_permintaan_survey")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"REQ_NAME", "LOCATION", "NOTELP", "DATE_SURVEY",
			"BRANCH", "SURVEYOR", "EMAIL_SURVEYOR", "OBJECTNAME",
		}))
	mock.ExpectQuery(exact("rincian_pucl")).WithArgs(kunci).
		WillReturnRows(sqlmock.NewRows([]string{
			"TGL_TERIMA_DOKUMEN_PUCL", "ALASAN_DOKTER_REJECT_RCL",
		}))

	detail, err := repo.FindClaimDetail(context.Background(), "PNC-13")
	require.NoError(t, err)
	require.Equal(t, "PNC-13", detail.ClaimNumber)
	require.Equal(t, kunci, detail.ClaimID)
	require.NoError(t, mock.ExpectationsWereMet())
}
