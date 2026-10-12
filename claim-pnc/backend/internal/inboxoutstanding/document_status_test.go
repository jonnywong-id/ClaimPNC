package inboxoutstanding_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxoutstanding"
)

func TestFindDocumentStatusAcceptsKnownCodesCaseInsensitively(t *testing.T) {
	status, found := inboxoutstanding.FindDocumentStatus("  LENGKAP ")
	require.True(t, found)
	require.Equal(t, inboxoutstanding.StatusComplete, status)

	status, found = inboxoutstanding.FindDocumentStatus("tka")
	require.True(t, found)
	require.Equal(t, inboxoutstanding.StatusTKA, status)
}

// Kode tak dikenal DITOLAK, tidak diam-diam diartikan "semua".
func TestFindDocumentStatusRejectsUnknownCodes(t *testing.T) {
	status, found := inboxoutstanding.FindDocumentStatus("hampir-lengkap")
	require.False(t, found)
	require.Empty(t, status)
}

func TestDocumentStatusTitleFollowsTheLegacyLabel(t *testing.T) {
	require.Equal(t, "Complete documents", inboxoutstanding.StatusComplete.Title())
	require.Equal(t, "Documents not complete", inboxoutstanding.StatusIncomplete.Title())
	require.Equal(t, "ALL Case", inboxoutstanding.StatusAll.Title())

	// Status yang tidak dikenal digambar apa adanya.
	require.Equal(t, "entah", inboxoutstanding.DocumentStatus("entah").Title())
}

// SELURUH sembilan tab dapat dihitung sejak 2026-10-11.
//
// Tidak satu pun bergabung karena kodenya berubah; seluruhnya karena sumber datanya tiba:
// `TKA_1` ditambahkan dan diisi · `ISPENDINGCLOSE` diisi · `SURVEYORTYPE_1` ternyata sudah
// ada sejak awal · `TEMPORARY_CLOSESEMENTARAPNC` dibaca · dan `M_KOMUNIKASI_PNC`
// disambungkan lewat `T_SURVEYORLIST` sebagai jembatan SRV -> PNC.
//
// Uji ini sengaja menyebut kesembilannya satu per satu, bukan menghitung panjang
// `statusOrder`. Menambah tab baru karena itu memaksa keputusan sadar di sini: apakah
// sumber datanya sudah ada, atau ia dikirim tanpa angka.
func TestSeluruhSembilanTabDapatDihitung(t *testing.T) {
	for _, status := range []inboxoutstanding.DocumentStatus{
		inboxoutstanding.StatusAll,
		inboxoutstanding.StatusComplete,
		inboxoutstanding.StatusIncomplete,
		inboxoutstanding.StatusTemporaryClose,
		inboxoutstanding.StatusDeadlineTemporaryClose,
		inboxoutstanding.StatusLossAdjuster,
		inboxoutstanding.StatusInternalSurveyor,
		inboxoutstanding.StatusCommunication,
		inboxoutstanding.StatusTKA,
	} {
		require.True(t, status.Countable(), status)
	}

	// Mekanismenya tetap ada dan tetap diuji: kode tak dikenal TIDAK dapat dihitung.
	require.False(t, inboxoutstanding.DocumentStatus("entah").Countable())
}

// Tab yang tidak dapat dihitung dikirim TANPA jumlah — bukan nol — dan urutannya mengikuti
// layar Pega.
func TestBuildSummaryKeepsTheLegacyOrderAndLeavesUncountableTabsWithoutACount(t *testing.T) {
	summary := inboxoutstanding.BuildSummary(map[inboxoutstanding.DocumentStatus]int{
		inboxoutstanding.StatusComplete: 3,
		inboxoutstanding.StatusAll:      5,

		// Kode tak dikenal: diabaikan, tidak menambah tab dan tidak menimpa apa pun.
		inboxoutstanding.DocumentStatus("entah"): 9,
	}, 5)

	require.Equal(t, 5, summary.Total)
	require.Len(t, summary.Status, 9)

	codes := make([]inboxoutstanding.DocumentStatus, 0, len(summary.Status))
	for _, item := range summary.Status {
		codes = append(codes, item.Status)
	}
	// "ALL Case" di DEPAN, sisanya urutan Pega — keputusan Work Owner 2026-10-05.
	// Lihat catatan pada statusOrder; di export ia berada di urutan ketujuh.
	require.Equal(t, []inboxoutstanding.DocumentStatus{
		inboxoutstanding.StatusAll,
		inboxoutstanding.StatusComplete,
		inboxoutstanding.StatusIncomplete,
		inboxoutstanding.StatusTemporaryClose,
		inboxoutstanding.StatusDeadlineTemporaryClose,
		inboxoutstanding.StatusLossAdjuster,
		inboxoutstanding.StatusInternalSurveyor,
		inboxoutstanding.StatusCommunication,
		inboxoutstanding.StatusTKA,
	}, codes)

	require.Equal(t, "ALL Case", summary.Status[0].Label)
	require.NotNil(t, summary.Status[0].Count)
	require.Equal(t, 5, *summary.Status[0].Count)

	require.Equal(t, "Complete documents", summary.Status[1].Label)
	require.NotNil(t, summary.Status[1].Count)
	require.Equal(t, 3, *summary.Status[1].Count)

	// Dapat dihitung tetapi tidak diserahkan penyimpanan: tetap tanpa jumlah.
	require.Equal(t, inboxoutstanding.StatusIncomplete, summary.Status[2].Status)
	require.Nil(t, summary.Status[2].Count)

	// Tab yang angkanya tidak diserahkan penyimpanan tetap TANPA jumlah — seluruhnya,
	// bukan hanya satu. Itu yang mencegah lencana nol muncul di tab yang belum dihitung.
	for i := 3; i < len(summary.Status); i++ {
		require.Nil(t, summary.Status[i].Count, summary.Status[i].Status)
	}

	// Kode tak dikenal tidak menyelinap masuk sebagai tab kesepuluh.
	require.Len(t, summary.Status, 9)
}

func TestNormalizeLineBusinessRecognisesTheFourLines(t *testing.T) {
	require.Equal(t, inboxoutstanding.LinePA, inboxoutstanding.NormalizeLineBusiness(" pa "))
	require.Equal(t, inboxoutstanding.LineTravel, inboxoutstanding.NormalizeLineBusiness("travel"))
	require.Equal(t, inboxoutstanding.LineNonMBU, inboxoutstanding.NormalizeLineBusiness("NonMBU"))
	require.Equal(t, inboxoutstanding.LineBonding, inboxoutstanding.NormalizeLineBusiness("BONDING"))

	// Nilai asing menjadi tanpa cakupan, bukan ditolak.
	require.Equal(t, inboxoutstanding.LineUnknown, inboxoutstanding.NormalizeLineBusiness("MARINE"))
	require.Equal(t, inboxoutstanding.LineUnknown, inboxoutstanding.NormalizeLineBusiness(""))
}

func TestExportFilterNormalizeClampsTheBatchAndOffset(t *testing.T) {
	from := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)

	clean := inboxoutstanding.ExportFilter{
		LineBusiness: "travel", From: &from, Limit: 0, Offset: -4,
	}.Normalize()
	require.Equal(t, inboxoutstanding.LineTravel, clean.LineBusiness)
	require.Equal(t, inboxoutstanding.DefaultLimit, clean.Limit)
	require.Zero(t, clean.Offset)
	require.Equal(t, &from, clean.From)

	big := inboxoutstanding.ExportFilter{Limit: 5000, Offset: 10}.Normalize()
	require.Equal(t, inboxoutstanding.MaxExportBatch, big.Limit)
	require.Equal(t, 10, big.Offset)
	require.Equal(t, inboxoutstanding.LineUnknown, big.LineBusiness)
}

// ---------------------------------------------------------------------------
// Sembilan label, delapan tab — Deadline dan TKA berbagi satu slot
// ---------------------------------------------------------------------------

// Di Pega keduanya memakai `TempVisibility.Email==9` dan dibedakan `pyPosition`, sehingga
// seorang pengguna tidak pernah melihat keduanya. Padanan `pyPosition` di sini adalah
// `M_LOGIN_PNC.LINE_BUSINESS`.

func TestPetugasPATidakMelihatTabDeadlineMelainkanTKA(t *testing.T) {
	ringkasan := inboxoutstanding.BuildSummary(nil, 0).
		ForLineBusiness(inboxoutstanding.LinePA)

	kode := kodeTab(ringkasan)
	require.Contains(t, kode, inboxoutstanding.StatusTKA)
	require.NotContains(t, kode, inboxoutstanding.StatusDeadlineTemporaryClose)
	require.Len(t, kode, 8)
}

func TestPetugasSelainPAMelihatDeadlineBukanTKA(t *testing.T) {
	for _, lini := range []inboxoutstanding.LineBusiness{
		inboxoutstanding.LineTravel,
		inboxoutstanding.LineNonMBU,
		inboxoutstanding.LineBonding,
	} {
		ringkasan := inboxoutstanding.BuildSummary(nil, 0).ForLineBusiness(lini)

		kode := kodeTab(ringkasan)
		require.Contains(t, kode, inboxoutstanding.StatusDeadlineTemporaryClose, lini)
		require.NotContains(t, kode, inboxoutstanding.StatusTKA, lini)
		require.Len(t, kode, 8, lini)
	}
}

// Lini bisnis yang tidak terbaca jatuh ke perilaku MAYORITAS, bukan ke pengecualian.
// Menebak ke arah TKA akan menyembunyikan Deadline dari hampir semua orang setiap kali
// pembacaan lini bisnis gagal — dan kegagalan itu tidak memunculkan galat apa pun.
func TestLiniBisnisTidakDikenaliJatuhKeDeadline(t *testing.T) {
	ringkasan := inboxoutstanding.BuildSummary(nil, 0).
		ForLineBusiness(inboxoutstanding.LineUnknown)

	kode := kodeTab(ringkasan)
	require.Contains(t, kode, inboxoutstanding.StatusDeadlineTemporaryClose)
	require.NotContains(t, kode, inboxoutstanding.StatusTKA)
}

// Menyaring tab TIDAK boleh menyentuh total — total mengisi tab "ALL Case" dan wajib
// cocok dengan paginasi grid.
func TestPenyaringTabTidakMengubahTotalMaupunUrutan(t *testing.T) {
	asal := inboxoutstanding.BuildSummary(map[inboxoutstanding.DocumentStatus]int{
		inboxoutstanding.StatusAll: 7,
	}, 7)

	hasil := asal.ForLineBusiness(inboxoutstanding.LinePA)

	require.Equal(t, 7, hasil.Total)
	require.Equal(t, inboxoutstanding.StatusAll, hasil.Status[0].Status,
		"ALL Case tetap paling depan setelah penyaringan")
	require.Equal(t, inboxoutstanding.StatusComplete, hasil.Status[1].Status)
}

func kodeTab(s inboxoutstanding.Summary) []inboxoutstanding.DocumentStatus {
	kode := make([]inboxoutstanding.DocumentStatus, 0, len(s.Status))
	for _, item := range s.Status {
		kode = append(kode, item.Status)
	}
	return kode
}
