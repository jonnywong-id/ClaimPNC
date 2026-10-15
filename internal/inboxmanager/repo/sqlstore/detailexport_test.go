package sqlstore

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
)

// TestCalendarDaysSamaDenganOracle mengunci TAT pada angka yang Oracle hitung.
//
// Kedua angka harapan dibaca langsung dari Oracle 2026-10-14:
//
//	DATE '2020-05-12' - DATE '1000-01-01'  = 372674
//	DATE '2020-05-12' - DATE '1990-01-01'  = 11089
//
// Yang pertama adalah jebakannya: "Tgl Aksep" berisi tanggal penanda 01/01/1000, dan hitungan
// `time.Sub` jenuh di 106751 sementara hitungan Gregorian murni menjawab 372679.
func TestCalendarDaysSamaDenganOracle(t *testing.T) {
	process := time.Date(2020, 5, 12, 8, 58, 49, 0, time.FixedZone("+07", 7*3600))

	require.Equal(t, 372674, calendarDays(time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC), process))
	require.Equal(t, 11089, calendarDays(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), process))
	require.Equal(t, 0, calendarDays(process, process))
	require.Equal(t, -1, calendarDays(process, process.AddDate(0, 0, -1)))
	// Pergantian kalender: 04/10/1582 Julian langsung disusul 15/10/1582 Gregorian.
	require.Equal(t, 1, calendarDays(
		time.Date(1582, 10, 4, 0, 0, 0, 0, time.UTC),
		time.Date(1582, 10, 15, 0, 0, 0, 0, time.UTC)))
}

// TestMultiplyDecimalSepertiBigDecimal mengunci perkalian kolom "… OR".
func TestMultiplyDecimalSepertiBigDecimal(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"1500000", "0.25", "375000.00"},
		{"71453050", "0", "0"},
		{"-2.5", "4", "-10.0"},
		{"0.1", "0.02", "0.002"},
		{"", "0.5", "0.0"}, // @toDecimal("") dibaca nol
		{"", "", "0"},
		{"abc", "1", ""},
		{"1.2.3", "1", ""},
	}
	for _, c := range cases {
		require.Equalf(t, c.want, multiplyDecimal(c.a, c.b), "%s × %s", c.a, c.b)
	}
}

type fakeFee struct {
	unit float64
	ok   bool
}

func (f fakeFee) Fee(float64) (float64, bool) { return f.unit, f.ok }

type fakeProgress struct{}

func (fakeProgress) Positions(raw string) string { return "TAHAP:" + raw }

func TestAdjusterFee(t *testing.T) {
	raw := func(rows, direct, interp, base string) detailRaw {
		return detailRaw{text: map[string]string{
			"FeeJumlahBaris": rows, "FeeLangsung": direct,
			"FeeJumlahBarisInterpolasi": interp, "FeeDasarTotalClaim": base,
		}}
	}

	require.Empty(t, adjusterFee(raw("0", "", "", ""), nil), "tanpa settlement: kosong")
	require.Equal(t, "33330000", adjusterFee(raw("1", "33330000", "0", "0"), nil),
		"seluruhnya bertipe '4': tidak butuh tangga")
	require.Empty(t, adjusterFee(raw("1", "0", "1", "74195000"), nil),
		"butuh interpolasi tetapi tangga tidak ada: kosong, bukan nol")
	require.Equal(t, "3500", adjusterFee(raw("3", "500", "2", "74195000"),
		fakeFee{unit: 1500, ok: true}), "angka interpolasi terjumlah sekali per baris bukan-'4'")
	require.Empty(t, adjusterFee(raw("1", "0", "1", "1"), fakeFee{ok: false}))
}

func TestDeriveDetailRow(t *testing.T) {
	raw := detailRaw{
		text: map[string]string{
			"CaseID":              "PNC-1011",
			"AnalystDoctorRemaks": "HENDRIK, NG",
			"ProvinceID":          "74,195,000",
			"CountryID":           "71453050.5",
			"Password":            "1000.5",
			"RCV_ID":              "200",
			"ReportAddress":       "",
			"StatusEkonomis":      "10",
			"FaktorOR":            "0.25",
			"FeeJumlahBaris":      "1",
			"FeeLangsung":         "1500.75",
			"ProgresJSON":         "{}",
		},
		date: map[string]time.Time{
			"TglDateOfLoss":       time.Date(2018, 4, 23, 0, 0, 0, 0, time.UTC),
			"TglProses":           time.Date(2020, 5, 12, 8, 58, 49, 0, time.UTC),
			"TglAksep":            time.Date(2018, 7, 23, 0, 0, 0, 0, time.UTC),
			"TglUpdateProgres":    time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
			"TglKirimKomite":      time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
			"TglJatuhTempoKomite": time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		},
	}

	row := deriveDetailRow(raw, nil, fakeProgress{})

	require.Equal(t, "PNC-1011", row["CaseID"])
	require.Equal(t, "HENDRIK  NG", row["AnalystDoctorRemaks"], "koma teks menjadi spasi")
	require.Equal(t, "74 195 000", row["ProvinceID"])
	require.Equal(t, "71453050,5", row["CountryID"], "pemisah desimal menjadi koma")

	// "… OR": empat dikalikan faktor, yang kelima faktornya sendiri.
	require.Equal(t, "250,125", row["ReportDescription"])
	require.Equal(t, "50,00", row["StatusClaim"])
	require.Equal(t, "0,00", row["TelpTertanggung"], "operan kosong dibaca nol")
	require.Equal(t, "2,50", row["UserAdmin"])
	require.Equal(t, "0,25", row["OldAddress"])

	require.Equal(t, "1500,75", row["ReporterName"])
	require.Equal(t, "TAHAP:{}", row["AlasanTerlambat"])

	require.Equal(t, "23/04/2018", row["StatusWork"])
	require.Equal(t, "04", row["ClaimNo"])
	require.Equal(t, "2018", row["CloseClaimNote"])
	require.Equal(t, "12/05/2020", row["UserTeknisEmail"])
	require.Equal(t, "23/07/2018", row["DollarCurrencyVal"])
	require.Equal(t, "659", row["UserName"], "TAT dalam hari kalender")
	require.Empty(t, row["UserTeknis"], "tanggal reject kosong")
	require.Equal(t, "01/09/26", row["ReceiverClaim"], "tahun dua digit seperti 'DD/MM/YY'")

	// Judulnya tampak tertukar di berkas lama, dan itu dibawa apa adanya.
	require.Equal(t, "30/09/2026", row["view_ktp"], "\"Tgl Kirim\" berisi NEXT_FOLLOWUP")
	require.Equal(t, "02/09/2026", row["view_email"], "\"Tgl Jatuh Tempo\" berisi TGL_INPUT")

	for name := range inboxmanager.DetailExportEmptyColumns {
		require.Contains(t, row, name)
		require.Empty(t, row[name])
	}
}

// TestDeriveDetailRowMengisiSetiapKolom memastikan setiap kolom berkas punya sumber: kolom
// yang tidak diisi deriveDetailRow sama sekali berarti satu properti terlupa.
func TestDeriveDetailRowMengisiSetiapKolom(t *testing.T) {
	row := deriveDetailRow(detailRaw{text: map[string]string{}, date: map[string]time.Time{}},
		fakeFee{}, fakeProgress{})
	for _, column := range inboxmanager.DetailExportColumns {
		_, exists := row[column.Property]
		require.Truef(t, exists, "kolom %q (%s) tidak diisi", column.Title, column.Property)
	}
}

func TestDetailExportMengikatArgumenDanMengalirkanBaris(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	repo := NewRepo(db).WithDetailRules(inboxmanager.DetailRules{
		NewFeeScale: func(bands []inboxmanager.FeeBand) inboxmanager.FeeScale {
			require.Len(t, bands, 1)
			return fakeFee{unit: 100, ok: true}
		},
		NewProgressNames: func(names map[string]string) inboxmanager.ProgressNames {
			require.Equal(t, map[string]string{"02": "SURVEY"}, names)
			return fakeProgress{}
		},
	})

	mock.ExpectQuery(exact("detail_fee_scale")).
		WillReturnRows(sqlmock.NewRows([]string{"IndexFee", "LossAmount", "Fee"}).AddRow(1, 10, 5))
	mock.ExpectQuery(exact("detail_progress_names")).
		WillReturnRows(sqlmock.NewRows([]string{"IDMst", "Nama"}).AddRow("02", "SURVEY"))

	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	args := append(lineFlags("PA"), 0, from, until)
	mock.ExpectQuery(exact("detail_export")).WithArgs(values(args)...).
		WillReturnRows(sqlmock.NewRows([]string{"CaseID", "TglDateOfLoss", "CountryID"}).
			AddRow("PNC-1", time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC), "1.5").
			AddRow("PNC-2", nil, nil))

	q, err := inboxmanager.NewDetailExportQuery(
		inboxmanager.DetailExportInput{From: "2026-09-01", Until: "2026-09-30"},
		inboxmanager.Caller{Login: "MGR", LineBusiness: "PA"}, "ASM")
	require.NoError(t, err)

	got := []inboxmanager.DetailExportRow{}
	require.NoError(t, repo.DetailExport(context.Background(), q,
		func(row inboxmanager.DetailExportRow) error {
			got = append(got, row)
			return nil
		}))

	require.Len(t, got, 2)
	require.Equal(t, "PNC-1", got[0]["CaseID"])
	require.Equal(t, "03/09/2026", got[0]["StatusWork"])
	require.Equal(t, "1,5", got[0]["CountryID"])
	require.Empty(t, got[1]["StatusWork"])
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDetailExportSimasnetTanpaSaringanLini mengunci langkah "Jika SIMASNET, remove filter
// bisnis": di portal ASI bendera "seluruh lini" menyala meski petugas punya lini bisnis.
func TestDetailExportSimasnetTanpaSaringanLini(t *testing.T) {
	repo, mock := newMock(t)

	mock.ExpectQuery(exact("detail_export")).
		WithArgs(values(append(lineFlags(""), 1, time.Time{}, time.Time{}))...).
		WillReturnRows(sqlmock.NewRows([]string{"CaseID"}))

	q, err := inboxmanager.NewDetailExportQuery(inboxmanager.DetailExportInput{},
		inboxmanager.Caller{Login: "MGR", LineBusiness: "NONMBU"}, "asi")
	require.NoError(t, err)
	require.True(t, q.AllLines)

	require.NoError(t, repo.DetailExport(context.Background(), q,
		func(inboxmanager.DetailExportRow) error { return nil }))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDetailExportGalat(t *testing.T) {
	q, err := inboxmanager.NewDetailExportQuery(inboxmanager.DetailExportInput{},
		inboxmanager.Caller{Login: "MGR"}, "ASM")
	require.NoError(t, err)

	t.Run("kueri gagal", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_export")).WillReturnError(errors.New("ORA-00904: X"))

		err := repo.DetailExport(context.Background(), q,
			func(inboxmanager.DetailExportRow) error { return nil })
		require.ErrorIs(t, err, inboxmanager.ErrSourceUnavailable)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("emit gagal menghentikan pembacaan", func(t *testing.T) {
		repo, mock := newMock(t)
		mock.ExpectQuery(exact("detail_export")).
			WillReturnRows(sqlmock.NewRows([]string{"CaseID"}).AddRow("A").AddRow("B"))

		stop := errors.New("putus")
		calls := 0
		err := repo.DetailExport(context.Background(), q,
			func(inboxmanager.DetailExportRow) error { calls++; return stop })
		require.ErrorIs(t, err, stop)
		require.Equal(t, 1, calls)
	})

	t.Run("tangga dan master tidak terbaca: kolomnya kosong, ekspor jalan", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
		require.NoError(t, err)
		t.Cleanup(func() { _ = db.Close() })
		repo := NewRepo(db).WithDetailRules(inboxmanager.DetailRules{
			NewFeeScale:      func([]inboxmanager.FeeBand) inboxmanager.FeeScale { return fakeFee{} },
			NewProgressNames: func(map[string]string) inboxmanager.ProgressNames { return fakeProgress{} },
		})
		mock.ExpectQuery(exact("detail_fee_scale")).WillReturnError(errors.New("ORA-00942"))
		mock.ExpectQuery(exact("detail_progress_names")).WillReturnError(errors.New("ORA-00942"))
		mock.ExpectQuery(exact("detail_export")).
			WillReturnRows(sqlmock.NewRows([]string{"CaseID", "ProgresJSON"}).AddRow("A", "{}"))

		var row inboxmanager.DetailExportRow
		require.NoError(t, repo.DetailExport(context.Background(), q,
			func(r inboxmanager.DetailExportRow) error { row = r; return nil }))
		require.Empty(t, row["AlasanTerlambat"])
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
