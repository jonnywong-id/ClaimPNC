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

	"claim-pnc/internal/reportkpi"
)

// Uji pengisi seam SQL terhadap basis data tiruan.
//
// Yang diuji: kueri mana yang dikirim, argumen bind-nya, pemetaan kolom ke struct, dan
// setiap jalur galat (Query, Scan, Rows.Err).

var errBoom = errors.New("basis data mati")

// newMockRepo membentuk Repo di atas sqlmock dan memastikan seluruh harapan terpenuhi.
func newMockRepo(t *testing.T) (*Repo, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
		_ = db.Close()
	})
	// Koneksi yang SAMA dipasang sebagai keduanya, supaya uji yang tidak mempersoalkan
	// pembagian koneksi tidak perlu memasang dua mock. Pembagiannya sendiri diuji
	// tersendiri oleh TestHolidaysDibacaDiKoneksiKedua, yang memakai DUA mock berbeda —
	// dan di situlah ia benar-benar terbukti.
	return NewRepo(db, db), mock
}

// sqlOf mengubah teks kueri bernama menjadi pola regexp yang persis.
func sqlOf(name string) string { return regexp.QuoteMeta(query(name)) }

// picSQLOf sama dengan sqlOf untuk kueri yang penyaring lininya sudah terpasang.
func picSQLOf(name string, line reportkpi.BusinessLine) string {
	return regexp.QuoteMeta(picQuery(name, line))
}

func adjusterQuery(reportType reportkpi.ReportType, adjuster string) reportkpi.Query {
	return reportkpi.Query{
		ReportType: reportType,
		Adjuster:   adjuster,
		Range:      reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	}
}

func adminQuery(group reportkpi.AdminGroup) reportkpi.AdminQuery {
	return reportkpi.AdminQuery{
		Group: group,
		Range: reportkpi.DateRange{From: "2026-03-01", To: "2026-03-31"},
	}
}

func picSpan() reportkpi.PICQuery {
	return reportkpi.PICQuery{
		Line: reportkpi.LineNonMBU,
		From: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	}
}

// brokenRows adalah satu baris berkolom tunggal — pemindaian ke banyak tujuan pasti gagal.
func brokenRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"X"}).AddRow("x")
}

// --- Tab Adjuster ---

func TestSummaryMapsRowsAndBindsFilter(t *testing.T) {
	repo, mock := newMockRepo(t)

	rows := sqlmock.NewRows(summaryColumns).
		AddRow("PT A", "FINAL", 1.0, 2.0, nil, 4.0, 5.0, 1.5, 2.5, 3.5, 4.5)
	mock.ExpectQuery(sqlOf("summary")).
		WithArgs("FINAL", "FINAL", "PT A", "PT A", "2026-03-01", "2026-03-31").
		WillReturnRows(rows)

	result, err := repo.Summary(context.Background(), adjusterQuery(reportkpi.TypeFinal, "PT A"))
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "PT A", result[0].Adjuster)
	require.Equal(t, reportkpi.TypeFinal, result[0].ReportType)
	require.Equal(t, reportkpi.NewScore(1), result[0].Scores[reportkpi.ComponentSurvey])
	// NULL menjadi nilai kosong, bukan nol.
	require.Equal(t, reportkpi.EmptyScore(), result[0].Scores[reportkpi.ComponentPreliminaryAdvice])
	require.Equal(t, reportkpi.NewScore(4.5), result[0].Scores[reportkpi.ComponentTotal])
}

// Tipe ALL dan adjuster kosong dikirim sebagai NULL.
func TestSummaryBindsAllTypeAndEmptyAdjusterAsNull(t *testing.T) {
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(sqlOf("summary")).
		WithArgs(nil, nil, nil, nil, "2026-03-01", "2026-03-31").
		WillReturnRows(sqlmock.NewRows(summaryColumns))

	result, err := repo.Summary(context.Background(), adjusterQuery(reportkpi.TypeAll, ""))
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Empty(t, result)
}

func TestSummaryErrorPaths(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("summary")).WillReturnError(errBoom)
		_, err := repo.Summary(context.Background(), adjusterQuery(reportkpi.TypeFinal, ""))
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("summary")).WillReturnRows(brokenRows())
		_, err := repo.Summary(context.Background(), adjusterQuery(reportkpi.TypeFinal, ""))
		require.ErrorContains(t, err, "memindai ringkasan")
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("summary")).WillReturnRows(
			sqlmock.NewRows(summaryColumns).
				AddRow("PT A", "FINAL", 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0).
				RowError(0, errBoom))
		_, err := repo.Summary(context.Background(), adjusterQuery(reportkpi.TypeFinal, ""))
		require.ErrorIs(t, err, errBoom)
	})
}

func TestDetailMapsRowsAndPaginates(t *testing.T) {
	repo, mock := newMockRepo(t)

	scored := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows(detailColumns).
		AddRow("PT A", "CASE-1", "OUTSTANDING", scored,
			1.0, 2.0, 3.0, 4.0, 5.0, 1.0, 2.0, 3.0, 4.0, int64(7)).
		AddRow("PT A", "CASE-2", "OUTSTANDING", nil,
			nil, nil, nil, nil, nil, nil, nil, nil, nil, int64(7))
	// Halaman 2 berukuran 3 → offset 3.
	mock.ExpectQuery(sqlOf("detail")).
		WithArgs("OUTSTANDING", "OUTSTANDING", nil, nil, "2026-03-01", "2026-03-31", 3, 3).
		WillReturnRows(rows)

	page, err := repo.Detail(context.Background(),
		adjusterQuery(reportkpi.TypeOutstanding, ""), reportkpi.Pagination{Page: 2, Size: 3})
	require.NoError(t, err)
	require.Equal(t, 7, page.Total)
	require.Len(t, page.Rows, 2)
	require.Equal(t, "CASE-1", page.Rows[0].CaseID)
	require.Equal(t, "2026-03-04", page.Rows[0].ScoredOn)
	require.Equal(t, reportkpi.NewScore(5), page.Rows[0].Scores[reportkpi.ComponentProgress])
	require.Empty(t, page.Rows[1].ScoredOn, "tanggal NULL tetap kosong")
	require.Equal(t, reportkpi.EmptyScore(), page.Rows[1].Scores[reportkpi.ComponentTotal])
}

// Halaman tanpa baris mengembalikan senarai kosong, bukan nil.
func TestDetailEmptyPageHasEmptyRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("detail")).
		WithArgs("FINAL", "FINAL", "PT B", "PT B", "2026-03-01", "2026-03-31", 0, reportkpi.DefaultPageSize).
		WillReturnRows(sqlmock.NewRows(detailColumns))

	page, err := repo.Detail(context.Background(),
		adjusterQuery(reportkpi.TypeFinal, "PT B"), reportkpi.Pagination{})
	require.NoError(t, err)
	require.NotNil(t, page.Rows)
	require.Empty(t, page.Rows)
	require.Zero(t, page.Total)
}

func TestDetailErrorPaths(t *testing.T) {
	q := adjusterQuery(reportkpi.TypeFinal, "")
	t.Run("query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("detail")).WillReturnError(errBoom)
		_, err := repo.Detail(context.Background(), q, reportkpi.Pagination{})
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("detail")).WillReturnRows(brokenRows())
		_, err := repo.Detail(context.Background(), q, reportkpi.Pagination{})
		require.ErrorContains(t, err, "memindai rincian")
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("detail")).WillReturnRows(
			sqlmock.NewRows(detailColumns).
				AddRow("PT A", "C", "FINAL", nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, 1).
				RowError(0, errBoom))
		_, err := repo.Detail(context.Background(), q, reportkpi.Pagination{})
		require.ErrorIs(t, err, errBoom)
	})
}

// Nama kosong dan NULL tidak masuk dropdown, dan kuerinya TIDAK diberi parameter.
//
// `WithArgs()` tanpa argumen itu bagian dari yang diuji: rule Pega pengisinya tidak punya
// parameter satu pun, dan menambahkan penyaring di sini adalah kekeliruan yang pernah
// membuat dropdown kosong pada periode tanpa penilaian.
func TestAdjustersSkipsEmptyNames(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("adjusters")).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"ADJUSTER"}).
			AddRow("PT A").AddRow(nil).AddRow("").AddRow("PT B"))

	names, err := repo.Adjusters(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"PT A", "PT B"}, names)
}

func TestAdjustersErrorPaths(t *testing.T) {
	t.Run("query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("adjusters")).WillReturnError(errBoom)
		_, err := repo.Adjusters(context.Background())
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("scan", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("adjusters")).WillReturnRows(
			sqlmock.NewRows([]string{"A", "B"}).AddRow("x", "y"))
		_, err := repo.Adjusters(context.Background())
		require.ErrorContains(t, err, "memindai adjuster")
	})
	t.Run("rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("adjusters")).WillReturnRows(
			sqlmock.NewRows([]string{"ADJUSTER"}).AddRow("PT A").RowError(0, errBoom))
		_, err := repo.Adjusters(context.Background())
		require.ErrorIs(t, err, errBoom)
	})
}

func TestCheckSourceReadsCountAndTypes(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("check_source")).
		WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(42))
	mock.ExpectQuery(sqlOf("check_distinct_types")).
		WillReturnRows(sqlmock.NewRows([]string{"TIPE"}).
			AddRow("FINAL").AddRow(nil).AddRow("OUTSTANDING"))

	state, err := repo.CheckSource(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, state.Rows)
	require.Equal(t, []string{"FINAL", "OUTSTANDING"}, state.Types)
}

func TestCheckSourceErrorPaths(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("check_source")).WillReturnError(errBoom)
		_, err := repo.CheckSource(context.Background())
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, reportkpi.SourceTable)
	})
	t.Run("types query", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("check_source")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(sqlOf("check_distinct_types")).WillReturnError(errBoom)
		_, err := repo.CheckSource(context.Background())
		require.ErrorIs(t, err, errBoom)
	})
	t.Run("types scan", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("check_source")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(sqlOf("check_distinct_types")).WillReturnRows(
			sqlmock.NewRows([]string{"A", "B"}).AddRow("x", "y"))
		_, err := repo.CheckSource(context.Background())
		require.ErrorContains(t, err, "memindai nilai TIPE")
	})
	t.Run("types rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("check_source")).
			WillReturnRows(sqlmock.NewRows([]string{"N"}).AddRow(1))
		mock.ExpectQuery(sqlOf("check_distinct_types")).WillReturnRows(
			sqlmock.NewRows([]string{"TIPE"}).AddRow("FINAL").RowError(0, errBoom))
		_, err := repo.CheckSource(context.Background())
		require.ErrorIs(t, err, errBoom)
	})
}

// Kueri bernama yang tidak ada adalah cacat pemrograman — ia panik.
func TestQueryPanicsOnUnknownName(t *testing.T) {
	require.PanicsWithValue(t,
		`reportkpi/sqlstore: kueri "tidak_ada" tidak ditemukan di berkas .sql`,
		func() { _ = query("tidak_ada") })
}

// Nilai yang lebih sedikit daripada komponen tidak membuat pemetaan panik.
func TestToScoresStopsAtShorterInput(t *testing.T) {
	result := toScores(nil)
	require.Empty(t, result)
}

// --- Tab KPI Admin ---
//
// Sejak 2026-10-09 kueri tab ini mengembalikan BARIS, bukan angka jadi. Pencacahan,
// tangga nilai, dan selisih hari kerja dikerjakan Go — sehingga tiruan di bawah menyiapkan
// baris mentah beserta kalender liburnya, bukan hasil hitungan.
//
// Kalender libur ikut ditiru: `Holidays` dipanggil setiap kali selisih hari kerja dihitung,
// dan tanpa tiruannya kueri itu tidak terjawab.

// liburKosong menyiapkan jawaban kalender libur yang tidak memuat satu tanggal pun.
func liburKosong(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(sqlOf("holidays")).
		WillReturnRows(sqlmock.NewRows([]string{"TANGGAL"}))
}

// Kartu skor NON-MBU dihitung dari baris, dan leader dibedakan dari member.
func TestAdminTotalsNonMBUMenghitungDariBaris(t *testing.T) {
	repo, mock := newMockRepo(t)

	// Senin 2026-03-02 sebagai awal; selisih hari kerjanya ditentukan tanggal akhir.
	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	hariSama := senin                    // 0 hari kerja -> TIDAK melanggar
	duaHari := senin.AddDate(0, 0, 2)    // Rabu, 2 hari kerja -> melanggar (> 1)
	sehari := senin.AddDate(0, 0, 1)     // Selasa, 1 hari kerja -> TIDAK melanggar

	mock.ExpectQuery(sqlOf("admin_rows_nonmbu")).
		WithArgs("2026-03-01", "2026-03-31").
		WillReturnRows(sqlmock.NewRows([]string{"REINSURER", "AWAL", "AKHIR"}).
			AddRow("1", senin, duaHari).  // leader, melanggar
			AddRow("1", senin, sehari).   // leader, tidak
			AddRow("2", senin, duaHari).  // member, melanggar
			AddRow("2", senin, hariSama)) // member, tidak
	liburKosong(mock)

	totals, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupNonMBU))
	require.NoError(t, err)

	require.Equal(t, reportkpi.NewScore(1), totals.LeaderOverSLA)
	require.Equal(t, reportkpi.NewScore(2), totals.LeaderTotal)
	require.Equal(t, reportkpi.NewScore(1), totals.MemberOverSLA)
	require.Equal(t, reportkpi.NewScore(2), totals.MemberTotal)

	// 1 dari 2 = 50% -> di atas 2% -> nilai 0 pada kedua kelompok.
	require.Equal(t, reportkpi.NewScore(0), totals.LeaderScore)
	require.Equal(t, reportkpi.NewScore(0), totals.MemberScore)

	// Isian PA tidak tersentuh.
	require.False(t, totals.RegisterTotal.Present)
}

// Hari libur MENGURANGI selisih, sehingga klaim yang tadinya melanggar menjadi tidak.
//
// Inilah alasan kalender libur tetap dibaca dari basis data meski hitungannya pindah ke Go.
func TestAdminTotalsNonMBUMemperhitungkanHariLibur(t *testing.T) {
	repo, mock := newMockRepo(t)

	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	rabu := senin.AddDate(0, 0, 2)
	selasa := senin.AddDate(0, 0, 1)

	mock.ExpectQuery(sqlOf("admin_rows_nonmbu")).
		WillReturnRows(sqlmock.NewRows([]string{"REINSURER", "AWAL", "AKHIR"}).
			AddRow("1", senin, rabu))
	mock.ExpectQuery(sqlOf("holidays")).
		WillReturnRows(sqlmock.NewRows([]string{"TANGGAL"}).AddRow(selasa))

	totals, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupNonMBU))
	require.NoError(t, err)

	// Dua hari kerja dikurangi satu hari libur = satu -> tidak lagi melewati ambang.
	require.Equal(t, reportkpi.NewScore(0), totals.LeaderOverSLA)
	require.Equal(t, reportkpi.NewScore(1), totals.LeaderTotal)
}

// Kartu skor PA membaca TIGA kueri, dan yang ketiga tanpa parameter.
func TestAdminTotalsPAMembacaTigaKueri(t *testing.T) {
	repo, mock := newMockRepo(t)

	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	selasa := senin.AddDate(0, 0, 1)

	mock.ExpectQuery(sqlOf("admin_rows_pa_register")).
		WithArgs("2026-03-01", "2026-03-31").
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}).
			AddRow("K1", senin, selasa). // 1 hari kerja -> melanggar (> 0)
			AddRow("K2", senin, senin))  // 0 hari kerja -> tidak
	mock.ExpectQuery(sqlOf("admin_rows_pa_payment")).
		WithArgs("2026-03-01", "2026-03-31").
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}).
			AddRow("K1", senin, selasa).
			AddRow("K1", senin, selasa)) // klaim yang SAMA, dihitung sekali
	mock.ExpectQuery(sqlOf("admin_rows_pa_payment_total")).
		WithArgs().
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}).
			AddRow("K9", senin, selasa))
	liburKosong(mock)

	totals, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupPA))
	require.NoError(t, err)

	require.Equal(t, reportkpi.NewScore(1), totals.RegisterOverSLA)
	require.Equal(t, reportkpi.NewScore(2), totals.RegisterTotal)
	require.Equal(t, reportkpi.NewScore(1), totals.PaymentOverSLA,
		"dua baris milik satu klaim dihitung SEKALI")
	require.Equal(t, reportkpi.NewScore(1), totals.PaymentTotal)
	require.False(t, totals.LeaderTotal.Present)
}

// Tanggal terima LOD yang kosong jatuh ke tanggal akseptasi, sehingga umurnya NOL.
func TestAdminTotalsPALODKosongJatuhKeAkseptasi(t *testing.T) {
	repo, mock := newMockRepo(t)
	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(sqlOf("admin_rows_pa_register")).
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}))
	mock.ExpectQuery(sqlOf("admin_rows_pa_payment")).
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}).
			AddRow("K1", nil, senin)) // LOD kosong
	mock.ExpectQuery(sqlOf("admin_rows_pa_payment_total")).
		WillReturnRows(sqlmock.NewRows([]string{"CLAIM", "AWAL", "AKHIR"}))
	liburKosong(mock)

	totals, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupPA))
	require.NoError(t, err)
	require.Equal(t, reportkpi.NewScore(0), totals.PaymentOverSLA,
		"umur nol tidak pernah terhitung melanggar")
}

func TestAdminTotalsErrors(t *testing.T) {
	t.Run("nonmbu", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("admin_rows_nonmbu")).WillReturnError(errBoom)
		_, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupNonMBU))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "admin_rows_nonmbu")
	})
	t.Run("pa", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("admin_rows_pa_register")).WillReturnError(errBoom)
		_, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupPA))
		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "admin_rows_pa_register")
	})
	t.Run("kalender libur", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
		mock.ExpectQuery(sqlOf("admin_rows_nonmbu")).
			WillReturnRows(sqlmock.NewRows([]string{"REINSURER", "AWAL", "AKHIR"}).
				AddRow("1", senin, senin))
		mock.ExpectQuery(sqlOf("holidays")).WillReturnError(errBoom)
		_, err := repo.AdminTotals(context.Background(), adminQuery(reportkpi.AdminGroupNonMBU))
		require.Error(t, err)
	})
}

func TestAdminDetailNonMBUMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	rabu := senin.AddDate(0, 0, 2)

	mock.ExpectQuery(sqlOf("admin_detail_nonmbu")).
		WithArgs("2026-03-01", "2026-03-31", 0, 2).
		WillReturnRows(sqlmock.NewRows([]string{"a", "b", "c", "d", "e", "f", "g"}).
			AddRow("K1", "P1", "BIS", senin, rabu, "leader", int64(9)))
	liburKosong(mock)

	page, err := repo.AdminDetail(context.Background(),
		adminQuery(reportkpi.AdminGroupNonMBU), reportkpi.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Equal(t, 9, page.Total)
	require.Equal(t, reportkpi.AdminDetailRow{
		ClaimNumber: "K1", PolicyNumber: "P1", BusinessName: "BIS",
		RegisterDate: "2026-03-02", TransferDate: "2026-03-04", TeamFlag: "leader",
		RegisterAging: reportkpi.NewScore(2),
	}, page.Rows[0])
}

func TestAdminDetailPAMapsRows(t *testing.T) {
	repo, mock := newMockRepo(t)
	senin := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	rabu := senin.AddDate(0, 0, 2)

	mock.ExpectQuery(sqlOf("admin_detail_pa")).
		WithArgs("2026-03-01", "2026-03-31", 0, reportkpi.DefaultPageSize).
		WillReturnRows(sqlmock.NewRows(
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}).
			AddRow("K2", "P2", senin, senin, "ADM", senin, rabu, nil, rabu,
				"Close", int64(1)))
	liburKosong(mock)

	page, err := repo.AdminDetail(context.Background(),
		adminQuery(reportkpi.AdminGroupPA), reportkpi.Pagination{})
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)

	baris := page.Rows[0]
	require.Equal(t, "K2", baris.ClaimNumber)
	require.Equal(t, "ADM", baris.AdminName)

	// Terima LOD kosong -> ditampilkan sebagai tanggal akseptasi, dan umurnya nol.
	require.Equal(t, "2026-03-04", baris.LODReceiveDate)
	require.Equal(t, reportkpi.NewScore(0), baris.PaymentAging)
	require.Equal(t, "SLA", baris.PaymentSLA)

	// Dua hari kerja -> melewati ambang grid (> 1).
	require.Equal(t, reportkpi.NewScore(2), baris.RegisterAging)
	require.Equal(t, "TIDAK SLA", baris.RegisterSLA)
	require.Equal(t, "Close", baris.ClaimStatus)
}

func TestAdminDetailErrorPaths(t *testing.T) {
	for _, tc := range []struct {
		group reportkpi.AdminGroup
		name  string
		cols  int
	}{
		{reportkpi.AdminGroupNonMBU, "admin_detail_nonmbu", 7},
		{reportkpi.AdminGroupPA, "admin_detail_pa", 11},
	} {
		t.Run(string(tc.group)+" query", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			mock.ExpectQuery(sqlOf(tc.name)).WillReturnError(errBoom)
			_, err := repo.AdminDetail(context.Background(), adminQuery(tc.group), reportkpi.Pagination{})
			require.ErrorIs(t, err, errBoom)
		})
		t.Run(string(tc.group)+" scan", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			mock.ExpectQuery(sqlOf(tc.name)).WillReturnRows(brokenRows())
			_, err := repo.AdminDetail(context.Background(), adminQuery(tc.group), reportkpi.Pagination{})
			require.ErrorContains(t, err, "memindai rincian")
		})
		t.Run(string(tc.group)+" rows", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			cols := make([]string, tc.cols)
			values := make([]driver.Value, tc.cols)
			for i := range cols {
				cols[i] = string(rune('a' + i))
			}
			mock.ExpectQuery(sqlOf(tc.name)).WillReturnRows(
				sqlmock.NewRows(cols).AddRow(values...).RowError(0, errBoom))
			_, err := repo.AdminDetail(context.Background(), adminQuery(tc.group), reportkpi.Pagination{})
			require.ErrorIs(t, err, errBoom)
		})
	}
}

// --- Tab KPI PIC Teknik ---

func TestPICsMapsLeaderFlagAndSkipsBlankOperator(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("pic_list")).
		WithArgs("NONMBU").
		WillReturnRows(sqlmock.NewRows([]string{"OPERATOR_ID", "STS_LEADER"}).
			AddRow(" PICSATU ", "true").
			AddRow("  ", "TRUE").
			AddRow("PICDUA", "FALSE").
			AddRow("PICTIGA", nil))

	pics, err := repo.PICs(context.Background(), reportkpi.LineNonMBU)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.PICProfile{
		{OperatorID: "PICSATU", Leader: true},
		{OperatorID: "PICDUA"},
		{OperatorID: "PICTIGA"},
	}, pics)
}

func TestBandsMapsRowsAndBindsNote(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("bands")).
		WithArgs(reportkpi.JobSLA, reportkpi.TeamLeader, reportkpi.TeamLeader).
		WillReturnRows(sqlmock.NewRows([]string{"JOB", "VALUE", "BOTTOM", "TOP", "NOTE"}).
			AddRow(" SLA KLAIM ", 5.0, 0.0, 20.0, " LEADER "))

	bands, err := repo.Bands(context.Background(), reportkpi.JobSLA, reportkpi.TeamLeader)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.Band{
		{Job: "SLA KLAIM", Value: 5, Bottom: 0, Top: 20, Note: "LEADER"},
	}, bands)
}

func TestThresholdDaysBranches(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		// Catatan dikirim dua kali — lihat TestTidakAdaPenandaBindBerulang.
		mock.ExpectQuery(sqlOf("threshold_days")).
			WithArgs(reportkpi.JobAnalysis, nil, nil).
			WillReturnRows(sqlmock.NewRows([]string{"DAY"}).AddRow(10.0))
		days, err := repo.ThresholdDays(context.Background(), reportkpi.JobAnalysis, "")
		require.NoError(t, err)
		require.Equal(t, reportkpi.NewScore(10), days)
	})
	t.Run("no rows", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("threshold_days")).
			WillReturnRows(sqlmock.NewRows([]string{"DAY"}))
		days, err := repo.ThresholdDays(context.Background(), reportkpi.JobProgress, "")
		require.NoError(t, err)
		require.Equal(t, reportkpi.EmptyScore(), days)
	})
	t.Run("null", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("threshold_days")).
			WillReturnRows(sqlmock.NewRows([]string{"DAY"}).AddRow(nil))
		days, err := repo.ThresholdDays(context.Background(), reportkpi.JobSLA, reportkpi.TeamMember)
		require.NoError(t, err)
		require.Equal(t, reportkpi.EmptyScore(), days)
	})
	t.Run("error", func(t *testing.T) {
		repo, mock := newMockRepo(t)
		mock.ExpectQuery(sqlOf("threshold_days")).WillReturnError(errBoom)
		days, err := repo.ThresholdDays(context.Background(), reportkpi.JobSLA, "")
		require.ErrorIs(t, err, errBoom)
		require.Equal(t, reportkpi.EmptyScore(), days)
	})
}

func TestHolidaysSkipsNullDates(t *testing.T) {
	repo, mock := newMockRepo(t)
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	holiday := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sqlOf("holidays")).
		WithArgs(from, to).
		WillReturnRows(sqlmock.NewRows([]string{"D"}).AddRow(holiday).AddRow(nil))

	days, err := repo.Holidays(context.Background(), from, to)
	require.NoError(t, err)
	require.Equal(t, []time.Time{holiday}, days)
}

func TestProgressCountsUsesLineFilter(t *testing.T) {
	repo, mock := newMockRepo(t)
	span := picSpan()
	mock.ExpectQuery(picSQLOf("progress_counts", reportkpi.LineNonMBU)).
		WithArgs(span.From, span.To).
		WillReturnRows(sqlmock.NewRows([]string{"PIC", "TOTAL", "ONTIME"}).
			AddRow(" PICSATU ", 10.0, 9.0))

	counts, err := repo.ProgressCounts(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.ProgressCount{{PIC: "PICSATU", Total: 10, OnTime: 9}}, counts)
}

func TestAnalysisSpansMapsDates(t *testing.T) {
	repo, mock := newMockRepo(t)
	span := picSpan()
	start := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sqlOf("analysis_spans")).
		WithArgs(span.From, span.To).
		WillReturnRows(sqlmock.NewRows([]string{"PIC", "S", "E"}).AddRow("PICSATU", start, end))

	spans, err := repo.AnalysisSpans(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.DateSpan{{PIC: "PICSATU", Start: start, End: end}}, spans)
}

func TestAcceptanceSpansMapsDates(t *testing.T) {
	repo, mock := newMockRepo(t)
	span := picSpan()
	lod := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	committee := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	accepted := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sqlOf("acceptance_spans")).
		WithArgs(span.From, span.To).
		WillReturnRows(sqlmock.NewRows([]string{"PIC", "TEAM", "L", "C", "A"}).
			AddRow("PICSATU", " LEADER ", lod, committee, accepted))

	spans, err := repo.AcceptanceSpans(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.AcceptanceSpan{{
		PIC: "PICSATU", Team: "LEADER", ReceiveLOD: lod,
		CommitteeDate: committee, AcceptanceDate: accepted,
	}}, spans)
}

func TestClosureSpansMapsDates(t *testing.T) {
	repo, mock := newMockRepo(t)
	span := picSpan()
	registered := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	closed := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(sqlOf("closure_spans")).
		WithArgs(span.From, span.To).
		WillReturnRows(sqlmock.NewRows([]string{"PIC", "TEAM", "R", "C"}).
			AddRow("PICDUA", "MEMBER", registered, closed))

	spans, err := repo.ClosureSpans(context.Background(), span)
	require.NoError(t, err)
	require.Equal(t, []reportkpi.ClosureSpan{{
		PIC: "PICDUA", Team: "MEMBER", RegisterDate: registered, CloseDate: closed,
	}}, spans)
}

// Seluruh pembaca tab PIC Teknik meneruskan galat Query, Scan, dan Rows.Err apa adanya.
func TestPICReadersPropagateErrors(t *testing.T) {
	span := picSpan()
	day := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)

	readers := []struct {
		name    string
		pattern string
		good    func() *sqlmock.Rows
		call    func(r *Repo) error
	}{
		{"pic_list", sqlOf("pic_list"),
			func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A", "B"}).AddRow("P", "TRUE") },
			func(r *Repo) error { _, err := r.PICs(context.Background(), reportkpi.LinePA); return err }},
		{"bands", sqlOf("bands"),
			func() *sqlmock.Rows {
				return sqlmock.NewRows([]string{"A", "B", "C", "D", "E"}).AddRow("J", 1.0, 0.0, 1.0, "")
			},
			func(r *Repo) error { _, err := r.Bands(context.Background(), "J", ""); return err }},
		{"holidays", sqlOf("holidays"),
			func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A"}).AddRow(day) },
			func(r *Repo) error { _, err := r.Holidays(context.Background(), day, day); return err }},
		{"progress_counts", picSQLOf("progress_counts", span.Line),
			func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A", "B", "C"}).AddRow("P", 1.0, 1.0) },
			func(r *Repo) error { _, err := r.ProgressCounts(context.Background(), span); return err }},
		{"analysis_spans", sqlOf("analysis_spans"),
			func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A", "B", "C"}).AddRow("P", day, day) },
			func(r *Repo) error { _, err := r.AnalysisSpans(context.Background(), span); return err }},
		{"acceptance_spans", sqlOf("acceptance_spans"),
			func() *sqlmock.Rows {
				return sqlmock.NewRows([]string{"A", "B", "C", "D", "E"}).AddRow("P", "T", day, day, day)
			},
			func(r *Repo) error { _, err := r.AcceptanceSpans(context.Background(), span); return err }},
		{"closure_spans", sqlOf("closure_spans"),
			func() *sqlmock.Rows { return sqlmock.NewRows([]string{"A", "B", "C", "D"}).AddRow("P", "T", day, day) },
			func(r *Repo) error { _, err := r.ClosureSpans(context.Background(), span); return err }},
	}

	for _, reader := range readers {
		t.Run(reader.name+" query", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			mock.ExpectQuery(reader.pattern).WillReturnError(errBoom)
			require.ErrorIs(t, reader.call(repo), errBoom)
		})
		t.Run(reader.name+" scan", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			// Kolom tunggal untuk pembaca yang kolomnya tunggal pun gagal: Holidays memindai
			// ke NullTime, dan teks bukan tanggal.
			rows := sqlmock.NewRows([]string{"A", "B", "C", "D", "E", "F"}).
				AddRow("x", "x", "x", "x", "x", "x")
			mock.ExpectQuery(reader.pattern).WillReturnRows(rows)
			require.Error(t, reader.call(repo))
		})
		t.Run(reader.name+" rows", func(t *testing.T) {
			repo, mock := newMockRepo(t)
			mock.ExpectQuery(reader.pattern).WillReturnRows(reader.good().RowError(0, errBoom))
			require.ErrorIs(t, reader.call(repo), errBoom)
		})
	}
}

func TestCheckBandsDetectsDirectionGapsAndOverlaps(t *testing.T) {
	repo, mock := newMockRepo(t)
	// Catatan dikirim DUA KALI: kuerinya memakai dua penanda untuk satu nilai, karena
	// driver mengikat menurut urutan kemunculan. Lihat TestTidakAdaPenandaBindBerulang.
	mock.ExpectQuery(sqlOf("bands")).
		WithArgs(reportkpi.JobProgress, nil, nil).
		WillReturnRows(sqlmock.NewRows([]string{"JOB", "VALUE", "BOTTOM", "TOP", "NOTE"}).
			AddRow("P", 5.0, 0.0, 20.0, nil).
			AddRow("P", 4.0, 20.0, 25.0, nil).  // bersambung
			AddRow("P", 3.0, 26.5, 30.0, nil).  // lubang 25 → 26.5
			AddRow("P", 1.0, 29.0, 100.0, nil)) // tumpang tindih 29 < 30

	state, err := repo.CheckBands(context.Background(), reportkpi.JobProgress)
	require.NoError(t, err)
	require.Equal(t, reportkpi.JobProgress, state.Job)
	require.Equal(t, 4, state.Bands)
	require.True(t, state.Descending)
	require.Equal(t, []string{
		"lubang antara 25 dan 26.5",
		"tumpang tindih antara 29 dan 30",
	}, state.Gaps)
}

func TestCheckBandsWithFewerThanTwoBands(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("bands")).
		WillReturnRows(sqlmock.NewRows([]string{"JOB", "VALUE", "BOTTOM", "TOP", "NOTE"}).
			AddRow("A", 1.0, 0.0, 100.0, nil))

	state, err := repo.CheckBands(context.Background(), reportkpi.JobAnalysis)
	require.NoError(t, err)
	require.Equal(t, 1, state.Bands)
	require.False(t, state.Descending)
	require.Empty(t, state.Gaps)
}

func TestCheckBandsPropagatesError(t *testing.T) {
	repo, mock := newMockRepo(t)
	mock.ExpectQuery(sqlOf("bands")).WillReturnError(errBoom)
	_, err := repo.CheckBands(context.Background(), reportkpi.JobAnalysis)
	require.ErrorIs(t, err, errBoom)
}
