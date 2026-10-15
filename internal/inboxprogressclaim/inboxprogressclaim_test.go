package inboxprogressclaim_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
)

// caller adalah pemanggil yang sah, dipakai hampir seluruh uji di berkas ini.
var caller = inboxprogressclaim.Caller{Login: "ADMINKLAIM"}

func TestPaginationCorrectsInsteadOfRejecting(t *testing.T) {
	// Halaman dan ukuran datang dari parameter query yang mudah salah ketik. Menolak
	// seluruh permintaan karena `halaman=0` akan membuat layar gagal tanpa alasan yang
	// terbaca pengguna.
	cases := []struct {
		name  string
		given inboxprogressclaim.Pagination
		want  inboxprogressclaim.Pagination
	}{
		{
			name:  "kosong menjadi halaman pertama berukuran bawaan",
			given: inboxprogressclaim.Pagination{},
			want: inboxprogressclaim.Pagination{
				Page: 1, Size: inboxprogressclaim.DefaultPageSize,
			},
		},
		{
			name:  "halaman negatif dibetulkan",
			given: inboxprogressclaim.Pagination{Page: -3, Size: 10},
			want:  inboxprogressclaim.Pagination{Page: 1, Size: 10},
		},
		{
			name: "ukuran di atas batas dipangkas, bukan ditolak diam-diam",
			given: inboxprogressclaim.Pagination{
				Page: 2, Size: inboxprogressclaim.MaxPageSize + 50,
			},
			want: inboxprogressclaim.Pagination{
				Page: 2, Size: inboxprogressclaim.MaxPageSize,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.given.Normalize())
		})
	}
}

func TestDefaultPageSizeFollowsPega(t *testing.T) {
	// 15, bukan 25 seperti Inbox Admin. Angkanya dibaca dari `.PageSize` pada
	// Activity/GetDataProgressClaim-Act.xml, dan ukuran halaman di sistem lama memang
	// berbeda-beda per layar.
	require.Equal(t, 15, inboxprogressclaim.DefaultPageSize)
}

func TestOffsetSkipsWholePages(t *testing.T) {
	require.Equal(t, 0, inboxprogressclaim.Pagination{Page: 1, Size: 15}.Offset())
	require.Equal(t, 15, inboxprogressclaim.Pagination{Page: 2, Size: 15}.Offset())
	require.Equal(t, 30, inboxprogressclaim.Pagination{Page: 3, Size: 15}.Offset())
}

func TestTotalPagesIsNeverZero(t *testing.T) {
	// Layar tidak boleh pernah menggambar "halaman 1 dari 0" saat hasilnya kosong.
	empty := inboxprogressclaim.ClaimPage{
		Total:      0,
		Pagination: inboxprogressclaim.Pagination{Page: 1, Size: 15},
	}
	require.Equal(t, 1, empty.TotalPages())

	exact := inboxprogressclaim.ClaimPage{
		Total:      30,
		Pagination: inboxprogressclaim.Pagination{Page: 1, Size: 15},
	}
	require.Equal(t, 2, exact.TotalPages())

	remainder := inboxprogressclaim.ClaimPage{
		Total:      31,
		Pagination: inboxprogressclaim.Pagination{Page: 1, Size: 15},
	}
	require.Equal(t, 3, remainder.TotalPages())
}

func TestPositionCollectorsKeepOrderAndEmptyValues(t *testing.T) {
	// Padanan antarkolom bersifat positional: nilai ke-n pada `posisi` sepadan dengan
	// nilai ke-n pada kedua status. Membuang nilai kosong akan menggeser seluruh nilai
	// sesudahnya dan memasangkan status sebuah posisi dengan nama posisi yang lain.
	row := inboxprogressclaim.ClaimRow{
		Positions: []inboxprogressclaim.Position{
			{Name: "SURVEY", Status1: "Berjalan", Status2: ""},
			{Name: "KOMITE", Status1: "", Status2: "Berkas lengkap"},
		},
	}

	require.Equal(t, []string{"SURVEY", "KOMITE"}, row.PositionNames())
	require.Equal(t, []string{"Berjalan", ""}, row.Status1List())
	require.Equal(t, []string{"", "Berkas lengkap"}, row.Status2List())
}

func TestPositionCollectorsHandleNoPositions(t *testing.T) {
	row := inboxprogressclaim.ClaimRow{}

	require.Empty(t, row.PositionNames())
	require.Empty(t, row.Status1List())
	require.Empty(t, row.Status2List())
}

func TestCallerIsTrimmed(t *testing.T) {
	require.Equal(t,
		inboxprogressclaim.Caller{Login: "ADMINKLAIM"},
		inboxprogressclaim.Caller{Login: "  ADMINKLAIM  "}.Clean(),
	)
}

func TestViewsCarryTheFourRegionsOfTheOldScreen(t *testing.T) {
	views := inboxprogressclaim.Views()
	require.Len(t, views, 4)

	codes := []string{}
	for _, view := range views {
		codes = append(codes, view.Code)
	}
	require.Equal(t, []string{
		inboxprogressclaim.ViewOutstanding,
		inboxprogressclaim.ViewNextFollowUp,
		inboxprogressclaim.ViewPerPIC,
		inboxprogressclaim.ViewEvaluation,
	}, codes)
}

func TestViewsCannotBeMutatedByCaller(t *testing.T) {
	first := inboxprogressclaim.Views()
	first[0].Name = "diubah"

	require.NotEqual(t, "diubah", inboxprogressclaim.Views()[0].Name)
}

func TestOutstandingDrawsRegisterDateTwice(t *testing.T) {
	// Ini bukan salah salin. Section/ProgressClaim_Section-Section.xml benar-benar
	// mengikat `.DateForAging` di dua sel grid terpisah pada region yang sama, dan
	// keputusan Work Owner 2026-09-21 adalah mereplikasinya apa adanya.
	//
	// Uji ini ada supaya kolom kedua tidak "dirapikan" oleh orang berikutnya yang
	// menganggapnya cacat — bila memang akan dihapus, ia dihapus lewat keputusan, bukan
	// lewat kerapian.
	view, found := inboxprogressclaim.FindView(inboxprogressclaim.ViewOutstanding)
	require.True(t, found)

	drawn := 0
	keys := map[string]bool{}
	for _, column := range view.Columns {
		if column.Field == inboxprogressclaim.FieldRegisterDate {
			drawn++
		}
		require.Falsef(t, keys[column.Key], "kunci kolom %s muncul dua kali", column.Key)
		keys[column.Key] = true
	}

	require.Equal(t, 2, drawn, "kolom DateForAging harus digambar dua kali")
	require.Len(t, view.Columns, 13)
}

func TestNextFollowUpDropsTheEarliestFollowUpColumn(t *testing.T) {
	// Kolom `KomiteApproveDate` nol kemunculan di region Next Follow Up.
	view, found := inboxprogressclaim.FindView(inboxprogressclaim.ViewNextFollowUp)
	require.True(t, found)
	require.Len(t, view.Columns, 12)

	for _, column := range view.Columns {
		require.NotEqual(t, inboxprogressclaim.FieldEarliestFollowUp, column.Field)
	}
}

func TestEveryColumnExplainsWhatItActuallyHolds(t *testing.T) {
	// Judul kolom memakai alias Pega yang sebagian menyesatkan — "District" berisi nama
	// tertanggung. Keterangan itulah satu-satunya tempat artinya terbaca pengguna, jadi ia
	// tidak boleh kosong.
	for _, view := range inboxprogressclaim.Views() {
		for _, column := range view.Columns {
			require.NotEmptyf(t, column.Title,
				"kolom %s pada bagian %s tanpa judul", column.Key, view.Code)
			require.NotEmptyf(t, column.Description,
				"kolom %s pada bagian %s tanpa keterangan", column.Key, view.Code)
		}
	}
}

func TestMisleadingAliasesAreKeptAsTitles(t *testing.T) {
	// Keputusan Work Owner 2026-09-21. Uji ini menahan godaan menamainya sesuai isi:
	// bila judulnya kelak diubah, ia diubah lewat keputusan baru, bukan diam-diam.
	view, _ := inboxprogressclaim.FindView(inboxprogressclaim.ViewOutstanding)

	titles := map[string]string{}
	for _, column := range view.Columns {
		titles[column.Field] = column.Title
	}

	require.Equal(t, "CaseID", titles[inboxprogressclaim.FieldClaimNumber])
	require.Equal(t, "ClaimNo", titles[inboxprogressclaim.FieldPolicyNumber])
	require.Equal(t, "District", titles[inboxprogressclaim.FieldInsuredName])
}

func TestEvaluationViewCarriesNoColumns(t *testing.T) {
	// Region ini KOSONG di Pega: ia punya judul dan kerangka tabel, tetapi nol properti
	// terikat dan nol activity pengisi. Mengarang kolom untuknya berarti mengarang layar
	// yang tidak pernah ada.
	view, found := inboxprogressclaim.FindView(inboxprogressclaim.ViewEvaluation)
	require.True(t, found)
	require.Equal(t, inboxprogressclaim.KindEmpty, view.Kind)
	require.Empty(t, view.Columns)
}

func TestDeadControlsBelongOnlyToClaimViews(t *testing.T) {
	// Dropdown lini bisnis mati pada kedua region klaim tetapi HIDUP pada rekap per PIC.
	// Bila penandanya tertukar, layar akan menyembunyikan penyaring yang sebenarnya
	// bekerja.
	require.Len(t, inboxprogressclaim.DeadControlsFor(inboxprogressclaim.ViewOutstanding), 2)
	require.Len(t, inboxprogressclaim.DeadControlsFor(inboxprogressclaim.ViewNextFollowUp), 1)
	require.Empty(t, inboxprogressclaim.DeadControlsFor(inboxprogressclaim.ViewPerPIC))

	perPIC, _ := inboxprogressclaim.FindView(inboxprogressclaim.ViewPerPIC)
	require.True(t, perPIC.SupportsBusinessFilter)

	outstanding, _ := inboxprogressclaim.FindView(inboxprogressclaim.ViewOutstanding)
	require.False(t, outstanding.SupportsBusinessFilter)
}

func TestBusinessLinesFollowTheExportDefinition(t *testing.T) {
	// Keempatnya, dan TANPA pilihan "semua": rekap per PIC mencocokkan nilai yang sama ke
	// MST_USER_TEKNIK.TYPE_BUSINESS, sehingga tanpa lini bisnis tidak ada petugas yang
	// cocok.
	require.Equal(t, []inboxprogressclaim.BusinessLine{
		inboxprogressclaim.BusinessNonMBU,
		inboxprogressclaim.BusinessTravel,
		inboxprogressclaim.BusinessBonding,
		inboxprogressclaim.BusinessPA,
	}, inboxprogressclaim.BusinessLines())

	_, valid := inboxprogressclaim.ParseBusinessLine("")
	require.False(t, valid, "kosong bukan berarti semua di layar ini")

	_, valid = inboxprogressclaim.ParseBusinessLine("MBU")
	require.False(t, valid)

	line, valid := inboxprogressclaim.ParseBusinessLine("  nonmbu  ")
	require.True(t, valid)
	require.Equal(t, inboxprogressclaim.BusinessNonMBU, line)
}

func TestNewQueryDefaultsToOutstanding(t *testing.T) {
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{}, caller)
	require.NoError(t, err)
	require.Equal(t, inboxprogressclaim.ViewOutstanding, query.View.Code)
}

func TestNewQueryRejectsUnknownCaller(t *testing.T) {
	_, err := inboxprogressclaim.NewQuery(
		inboxprogressclaim.QueryInput{}, inboxprogressclaim.Caller{Login: "   "})
	require.ErrorIs(t, err, inboxprogressclaim.ErrCallerUnknown)
}

func TestNewQueryRejectsUnknownView(t *testing.T) {
	_, err := inboxprogressclaim.NewQuery(
		inboxprogressclaim.QueryInput{View: "approval"}, caller)

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxprogressclaim.FieldView, validation.Violations[0].Field)
}

func TestKeywordIsDroppedOnViewsThatCannotSearch(t *testing.T) {
	// Kata kunci yang tampak aktif padahal tidak menyaring apa pun lebih buruk daripada
	// kotak cari yang tidak ada: pengguna menyimpulkan datanya kosong.
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
		Keyword:  "PNCN.26",
	}, caller)

	require.NoError(t, err)
	require.Empty(t, query.Keyword)
}

func TestBusinessLineIsRequiredOnlyOnPerPIC(t *testing.T) {
	_, err := inboxprogressclaim.NewQuery(
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewPerPIC}, caller)

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxprogressclaim.FieldBusiness, validation.Violations[0].Field)

	// Region klaim tidak menuntutnya, dan tidak pula memakainya.
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewOutstanding,
		Business: string(inboxprogressclaim.BusinessPA),
	}, caller)
	require.NoError(t, err)
	require.Empty(t, string(query.Business))
}

func TestDateRangeIsValidatedOnlyWhereItApplies(t *testing.T) {
	// Tanggal yang tidak terbaca pada region klaim DIABAIKAN, bukan ditolak: kontrolnya
	// memang mati di sana, dan menolaknya akan membuat layar gagal karena isian yang tidak
	// berpengaruh apa pun.
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View: inboxprogressclaim.ViewOutstanding,
		From: "bukan-tanggal",
	}, caller)
	require.NoError(t, err)
	require.Nil(t, query.From)

	_, err = inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
		From:     "bukan-tanggal",
	}, caller)

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxprogressclaim.FieldFrom, validation.Violations[0].Field)
}

func TestReversedDateRangeIsRejectedInsteadOfSwapped(t *testing.T) {
	// Menukarnya berarti menjawab pertanyaan yang tidak diajukan, dan pengguna tidak
	// pernah tahu bahwa yang ia ketik bukan yang ia lihat.
	_, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
		From:     "2026-09-30",
		To:       "2026-09-01",
	}, caller)

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxprogressclaim.FieldTo, validation.Violations[0].Field)
}

func TestAllViolationsAreReportedTogether(t *testing.T) {
	// Meniru perilaku Pega yang menampilkan semua pesan sekaligus (`P-5`). Tanpa ini,
	// pengguna memperbaiki satu isian, mengirim, lalu diberi tahu ada yang kedua.
	_, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View: inboxprogressclaim.ViewPerPIC,
		From: "bukan-tanggal",
		To:   "juga-bukan",
	}, caller)

	var validation *inboxprogressclaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 3)
}

func TestClaimQueryTruncatesTodayToItsDate(t *testing.T) {
	// Tanpa pemangkasan, "jatuh tempo hari ini" berarti "jatuh tempo sebelum jam sekian
	// hari ini", sehingga baris yang sama muncul dan menghilang tergantung kapan layar
	// dibuka.
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{}, caller)
	require.NoError(t, err)

	sore := time.Date(2026, time.September, 21, 16, 45, 12, 0, time.UTC)
	require.Equal(t,
		time.Date(2026, time.September, 21, 0, 0, 0, 0, time.UTC),
		query.ClaimQuery(sore).Today,
	)
}

func TestPICQueryCarriesCallerAndFilters(t *testing.T) {
	query, err := inboxprogressclaim.NewQuery(inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessTravel),
		From:     "2026-09-01",
		To:       "2026-09-30",
	}, caller)
	require.NoError(t, err)

	picQuery := query.PICQuery(time.Date(2026, time.September, 21, 8, 0, 0, 0, time.UTC))

	require.Equal(t, inboxprogressclaim.BusinessTravel, picQuery.Business)
	require.Equal(t, "ADMINKLAIM", picQuery.Caller.Login)
	require.Equal(t, 2026, picQuery.From.Year())
	require.Equal(t, time.September, picQuery.To.Month())
	require.Equal(t, 30, picQuery.To.Day())
}
