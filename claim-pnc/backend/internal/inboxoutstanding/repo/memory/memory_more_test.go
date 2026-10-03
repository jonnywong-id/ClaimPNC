package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
	"claim-pnc/internal/inboxoutstanding/repo/memory"
)

// exportClaim menyusun klaim berjalan untuk uji unduhan.
func exportClaim(id, panel, branch, pic string, registered time.Time) inboxoutstanding.OutstandingClaim {
	return inboxoutstanding.OutstandingClaim{
		ClaimID:       id,
		ClaimNumber:   "PNCN.26." + id,
		GroupPanel:    panel,
		BranchName:    branch,
		TechnicalPIC:  pic,
		ProcessStatus: "New",
		CurrentHolder: "SIAPAPUN",
		RegisteredAt:  registered,
	}
}

func exportIDs(page inboxoutstanding.Page) []string {
	ids := make([]string, 0, len(page.Claims))
	for _, c := range page.Claims {
		ids = append(ids, c.ClaimID)
	}
	return ids
}

var day = time.Date(2026, time.September, 10, 0, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// Data contoh
// ---------------------------------------------------------------------------

// Data contoh memuat keadaan yang mudah membuat layar salah gambar. Diperiksa lewat SELURUH
// isi repo, bukan lewat List, karena sebagian keadaan itu tidak boleh muncul di My Inbox.
func TestSampleDataCoversTheStatesThatBreakScreens(t *testing.T) {
	all := memory.NewRepoWithSamples().Claims()
	require.NotEmpty(t, all)

	var noNumber, noHolder, noReportDate bool
	panels := map[string]bool{}
	for _, c := range all {
		noNumber = noNumber || c.ClaimNumber == ""
		noHolder = noHolder || c.CurrentHolder == ""
		noReportDate = noReportDate || c.ReportDate == nil
		panels[c.GroupPanel] = true
	}
	require.True(t, noNumber, "klaim belum bernomor")
	require.True(t, noHolder, "tugas Workbasket yang belum bertuan")
	require.True(t, noReportDate, "tanggal lapor kosong")
	require.GreaterOrEqual(t, len(panels), 4, "beberapa Group Panel")
}

// Claims menyerahkan salinan: menulisinya tidak mengubah isi repo.
func TestClaimsReturnsACopy(t *testing.T) {
	r := memory.NewRepoWithSamples()
	first := r.Claims()
	first[0].ClaimID = "DIUBAH"

	require.NotEqual(t, "DIUBAH", r.Claims()[0].ClaimID)
}

// Tugas yang BELUM BERTUAN tidak muncul di My Inbox — ia bukan pekerjaan siapa pun.
func TestUnassignedSampleTasksDoNotAppearInMyInbox(t *testing.T) {
	r := memory.NewRepoWithSamples()

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: "BUDISANTOSO",
		Limit:      inboxoutstanding.MaxLimit,
	})
	require.NoError(t, err)
	require.NotEmpty(t, page.Claims)
	for _, c := range page.Claims {
		require.Equal(t, "BUDISANTOSO", c.CurrentHolder)
	}
}

// ---------------------------------------------------------------------------
// Penyaring daftar
// ---------------------------------------------------------------------------

// Klaim warisan yang tertugas ke identitas LAMA tetap muncul bagi pemiliknya.
func TestListMatchesTheLegacyIdentityToo(t *testing.T) {
	legacy := claim("lama", "PNCN.26.0001", "POL-1", "006", "PIC", "JKT", "Komite", 1)
	legacy.CurrentHolder = "OPERATORLAMA"
	other := claim("lain", "PNCN.26.0002", "POL-2", "006", "PIC", "JKT", "Komite", 2)
	other.CurrentHolder = "ORANGLAIN"
	r := repoWith(claim("baru", "PNCN.26.0003", "POL-3", "006", "PIC", "JKT", "Komite", 3), legacy, other)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik, AssignedToLegacy: "operatorlama",
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"baru", "lama"}, exportIDs(page))
}

func TestListFiltersGroupPanelRCVIDAndDocumentStatus(t *testing.T) {
	complete := claim("lengkap", "N1", "P1", "006", "PIC", "JKT", "Komite", 1)
	complete.DocumentComplete = true
	complete.RCVID = "RCV-1"
	incomplete := claim("belum", "N2", "P2", "002", "PIC", "JKT", "Komite", 2)
	incomplete.RCVID = "RCV-2"
	r := repoWith(complete, incomplete)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik, GroupPanel: "002"})
	require.NoError(t, err)
	require.Equal(t, []string{"belum"}, exportIDs(page))

	page, err = r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik, RCVID: "rcv-1"})
	require.NoError(t, err)
	require.Equal(t, []string{"lengkap"}, exportIDs(page))

	page, err = r.List(context.Background(), inboxoutstanding.Filter{
		AssignedTo: pemilik, DocumentStatus: inboxoutstanding.StatusIncomplete,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"belum"}, exportIDs(page))
}

// Dua klaim bertanggal sama diurutkan menurut ClaimID supaya urutannya stabil.
func TestListBreaksTiesByClaimID(t *testing.T) {
	a := claim("B", "N1", "P1", "006", "PIC", "JKT", "Komite", 0)
	b := claim("A", "N2", "P2", "006", "PIC", "JKT", "Komite", 0)
	b.RegisteredAt = a.RegisteredAt
	r := repoWith(a, b)

	page, err := r.List(context.Background(), inboxoutstanding.Filter{AssignedTo: pemilik})
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"}, exportIDs(page))
}

// ---------------------------------------------------------------------------
// Ringkasan
// ---------------------------------------------------------------------------

func TestSummaryCountsCompleteAndIncompleteAndIgnoresTheSelectedStatus(t *testing.T) {
	complete := claim("1", "N1", "P1", "006", "PIC", "JKT", "Komite", 1)
	complete.DocumentComplete = true
	incomplete := claim("2", "N2", "P2", "006", "PIC", "JKT", "Komite", 2)
	foreign := milikOrangLain(claim("3", "N3", "P3", "006", "PIC", "JKT", "Komite", 3))
	r := repoWith(complete, incomplete, foreign)

	summary, err := r.SummarizeDocumentStatus(context.Background(), inboxoutstanding.Filter{
		AssignedTo:     pemilik,
		DocumentStatus: inboxoutstanding.StatusComplete,
	})
	require.NoError(t, err)

	counts := map[inboxoutstanding.DocumentStatus]*int{}
	for _, item := range summary.Status {
		counts[item.Status] = item.Count
	}
	require.Equal(t, 1, *counts[inboxoutstanding.StatusComplete])
	require.Equal(t, 1, *counts[inboxoutstanding.StatusIncomplete])
	require.Equal(t, 2, *counts[inboxoutstanding.StatusAll])
	require.Nil(t, counts[inboxoutstanding.StatusLossAdjuster])
	require.Equal(t, 2, summary.Total)
}

func TestSummaryRequiresAnAssignee(t *testing.T) {
	_, err := memory.NewRepo().SummarizeDocumentStatus(context.Background(),
		inboxoutstanding.Filter{AssignedTo: "  "})
	require.ErrorIs(t, err, inboxoutstanding.ErrAssigneeRequired)
}

// ---------------------------------------------------------------------------
// Identitas lama dan lini bisnis
// ---------------------------------------------------------------------------

func TestLegacyOperatorIsStoredNormalized(t *testing.T) {
	// Repo bernilai nol pun dapat diisi — petanya dibentuk saat pertama dipakai.
	r := &memory.Repo{}
	r.SetLegacyOperator(" budi@contoh ", " budilama ")

	legacy, err := r.LegacyOperatorFor(context.Background(), "BUDI@CONTOH")
	require.NoError(t, err)
	require.Equal(t, "BUDILAMA", legacy)

	none, err := r.LegacyOperatorFor(context.Background(), "tidak-terdaftar")
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestLineBusinessIsLookedUpByNormalizedLogin(t *testing.T) {
	r := &memory.Repo{}
	r.SetLineBusiness(" siti ", inboxoutstanding.LineTravel)

	line, err := r.LineBusinessFor(context.Background(), "SITI")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineTravel, line)

	unknown, err := r.LineBusinessFor(context.Background(), "orang-lain")
	require.NoError(t, err)
	require.Equal(t, inboxoutstanding.LineUnknown, unknown)
}

// ---------------------------------------------------------------------------
// Unduhan
// ---------------------------------------------------------------------------

// Unduhan TIDAK menyaring pemilik pekerjaan, tetapi membuang klaim yang sudah selesai.
func TestExportIgnoresTheOwnerButDropsResolvedClaims(t *testing.T) {
	done := exportClaim("selesai", "006", "JKT", "PIC", day)
	done.ProcessStatus = "Resolved-Completed"
	rejected := exportClaim("ditolak", "006", "JKT", "PIC", day)
	rejected.ProcessStatus = " Resolved-Rejected "
	r := repoWith(exportClaim("jalan", "006", "JKT", "PIC", day), done, rejected)

	page, err := r.Export(context.Background(), inboxoutstanding.ExportFilter{})
	require.NoError(t, err)
	require.Equal(t, []string{"jalan"}, exportIDs(page))
	require.Equal(t, 1, page.Total)
}

func TestExportScopesByLineBusiness(t *testing.T) {
	r := repoWith(
		exportClaim("pa", "002", "JKT", "PIC", day),
		exportClaim("travel", "005", "JKT", "PIC", day.Add(time.Hour)),
		exportClaim("fire", "006", "JKT", "PIC", day.Add(2*time.Hour)),
		exportClaim("asnet", "004", " asnet ", "PIC", day.Add(3*time.Hour)),
		exportClaim("tanpa-pic", "003", "JKT", " ", day.Add(4*time.Hour)),
	)

	for line, want := range map[inboxoutstanding.LineBusiness][]string{
		inboxoutstanding.LinePA:     {"pa"},
		inboxoutstanding.LineTravel: {"travel"},
		// Non-MBU: panel 003/004/006, bukan cabang ASNET, dan PIC teknis terisi.
		inboxoutstanding.LineNonMBU: {"fire"},
		// BONDING tidak dapat ditirukan di memori: seluruh baris diloloskan.
		inboxoutstanding.LineBonding: {"tanpa-pic", "asnet", "fire", "travel", "pa"},
		inboxoutstanding.LineUnknown: {"tanpa-pic", "asnet", "fire", "travel", "pa"},
	} {
		page, err := r.Export(context.Background(), inboxoutstanding.ExportFilter{
			LineBusiness: line, Limit: inboxoutstanding.MaxExportBatch,
		})
		require.NoError(t, err)
		require.Equal(t, want, exportIDs(page), string(line))
	}
}

// Batas atas rentang tanggal EKSKLUSIF, sama seperti SQL-nya.
func TestExportHonoursAHalfOpenDateRange(t *testing.T) {
	r := repoWith(
		exportClaim("sebelum", "006", "JKT", "PIC", day.Add(-time.Second)),
		exportClaim("awal", "006", "JKT", "PIC", day),
		exportClaim("akhir", "006", "JKT", "PIC", day.AddDate(0, 0, 1)),
	)
	from, to := day, day.AddDate(0, 0, 1)

	page, err := r.Export(context.Background(), inboxoutstanding.ExportFilter{From: &from, To: &to})
	require.NoError(t, err)
	require.Equal(t, []string{"awal"}, exportIDs(page))
}

func TestExportPaginatesAfterCountingAndBreaksTiesByClaimID(t *testing.T) {
	r := repoWith(
		exportClaim("B", "006", "JKT", "PIC", day),
		exportClaim("A", "006", "JKT", "PIC", day),
		exportClaim("C", "006", "JKT", "PIC", day.Add(time.Hour)),
	)

	page, err := r.Export(context.Background(), inboxoutstanding.ExportFilter{Limit: 2, Offset: 1})
	require.NoError(t, err)
	require.Equal(t, []string{"A", "B"}, exportIDs(page))
	require.Equal(t, 3, page.Total)

	beyond, err := r.Export(context.Background(), inboxoutstanding.ExportFilter{Offset: 10})
	require.NoError(t, err)
	require.NotNil(t, beyond.Claims)
	require.Empty(t, beyond.Claims)
	require.Equal(t, 3, beyond.Total)
}
