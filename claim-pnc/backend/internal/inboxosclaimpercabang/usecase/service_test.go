package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/repo/memory"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/platform/clock"
)

const primaryPortal = "ASM"

// fixedNow adalah jam uji. Ia dipatok jauh setelah tanggal contoh, sehingga baris tertua pada
// contoh memang melewati ambang 180 hari — dan tetap melewatinya tahun depan.
func fixedNow() *clock.Fixed {
	return clock.FixedAt(time.Date(2026, time.September, 28, 5, 0, 0, 0, time.UTC))
}

func selectorFor(store inboxosclaimpercabang.Repo) inboxosclaimpercabang.RepoSelector {
	return func(alias string) (inboxosclaimpercabang.Repo, error) {
		if alias != primaryPortal {
			return nil, errors.New("portal tidak tersedia: " + alias)
		}
		return store, nil
	}
}

func newService(t *testing.T, store inboxosclaimpercabang.Repo) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: selectorFor(store),
		Clock:        fixedNow(),
	})
	require.NoError(t, err)
	return service
}

// callerAt membentuk pemanggil dengan kode cabang RINCI — yang dikirim HCQ, bukan yang dipakai
// baris klaim. Contoh: "078" untuk CILEGON, yang barisnya berkode "100099".
func callerAt(detailBranch string) inboxosclaimpercabang.Caller {
	return inboxosclaimpercabang.Caller{
		Login:            "PETUGAS1",
		DetailBranchCode: detailBranch,
	}
}

func firstPage() inboxosclaimpercabang.Pagination {
	return inboxosclaimpercabang.Pagination{Page: 1, Size: 100}
}

func TestServiceRefusesToBeBuiltWithoutItsSeams(t *testing.T) {
	// Keduanya WAJIB, dan kegagalannya harus terjadi saat aplikasi dirakit — bukan saat
	// pengguna membuka layar. Clock yang nil membuat setiap baris berumur nol hari, dan nol
	// hari adalah angka yang tampak wajar.
	base := usecase.Options{
		RepoSelector: selectorFor(memory.NewSampleStore()),
		Clock:        fixedNow(),
	}

	withoutRepo := base
	withoutRepo.RepoSelector = nil
	_, err := usecase.NewService(withoutRepo)
	require.Error(t, err)

	withoutClock := base
	withoutClock.Clock = nil
	_, err = usecase.NewService(withoutClock)
	require.Error(t, err)
}

func TestListRefusesCallerWithoutIdentity(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{DetailBranchCode: "078"}, firstPage())

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrCallerUnknown)
}

func TestListRefusesCallerWithoutBranchInsteadOfShowingNothing(t *testing.T) {
	// Ini keputusan terpenting di modul ini, dan ia meniru sistem lama alih-alih menyimpang
	// darinya. `OutstandingperCabang_PreAct` langkah 2 menampilkan pesan ketika cabang
	// pemanggil kosong — bukan grid kosong.
	//
	// Daftar kosong dan "Anda belum punya cabang" TERLIHAT SAMA di layar, padahal yang
	// pertama berarti tidak ada pekerjaan dan yang kedua berarti layar tidak dapat bekerja.
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{Login: "MITRA1"}, firstPage())

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)
}

func TestIdentityIsCheckedBeforeBranch(t *testing.T) {
	// Urutan pemeriksaan menentukan pesan mana yang sampai ke pengguna, dan keduanya punya
	// tindak lanjut yang berbeda: masuk ulang, versus menghubungi Tim IT.
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{}, firstPage())

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrCallerUnknown)
	require.NotErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)
}

func TestListScopesToTheCallerBranch(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed, err := service.List(context.Background(), primaryPortal,
		callerAt("078"), firstPage())
	require.NoError(t, err)

	require.Equal(t, "100099", listed.Query.Branch.Code)
	require.Equal(t, "CILEGON", listed.Query.Branch.Name,
		"nama cabang harus datang dari master, bukan dari profil sesi")

	for _, item := range listed.Page.Items {
		require.Equal(t, "100099", item.BranchCode)
	}
}

func TestListFillsAgingFromTheClock(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed, err := service.List(context.Background(), primaryPortal,
		callerAt("078"), firstPage())
	require.NoError(t, err)

	byNumber := map[string]inboxosclaimpercabang.WorkItem{}
	for _, item := range listed.Page.Items {
		byNumber[item.ClaimNumber] = item
	}

	// PNC-9001 terdaftar 15 Januari 2024; terhadap jam uji 28 September 2026 itu 987 hari.
	require.Equal(t, 987, byNumber["PNC-9001"].AgingDays)
	require.True(t, byNumber["PNC-9001"].NeedsAttention(),
		"klaim berumur jauh di atas 180 hari harus ditandai")

	// PNC-9002 masih muda — yang menandainya MERAH adalah progres mandek, bukan umur.
	require.Less(t, byNumber["PNC-9002"].AgingDays, inboxosclaimpercabang.AgingThreshold)
	require.True(t, byNumber["PNC-9002"].NeedsAttention())
	require.True(t, byNumber["PNC-9002"].ProgressStalled)
}

func TestUnknownDetailBranchCodeStopsTheScreenInsteadOfShowingNothing(t *testing.T) {
	// Kode rinci yang tidak dikenal master TIDAK dapat diteruskan sebagai penyaring: ia bukan
	// kode yang dipakai baris klaim, dan memakainya apa adanya menghasilkan nol baris.
	//
	// Nol baris dan "cabang Anda tidak dikenal" TERLIHAT SAMA di layar, padahal yang pertama
	// berarti tidak ada pekerjaan dan yang kedua berarti layar tidak dapat bekerja sama
	// sekali. Karena itu jawabannya pesan, bukan grid kosong.
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), primaryPortal,
		callerAt("999"), firstPage())

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)
}

func TestClaimBranchCodeIsNotAcceptedAsTheDetailCode(t *testing.T) {
	// Kekeliruan yang paling mungkin terjadi: meneruskan `BranchCode` profil (6 digit) ke
	// tempat kode rinci (3 digit), karena keduanya sama-sama disebut "cabang".
	//
	// Bila itu diam-diam lolos, layar menampilkan baris cabang yang salah — tepat kelas cacat
	// yang tidak menghasilkan satu pun galat (`R-20`).
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), primaryPortal,
		callerAt("100099"), firstPage())

	require.ErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)
}

func TestListRejectsUnknownPortal(t *testing.T) {
	// Portal yang tidak dikenal WAJIB gagal, bukan jatuh ke portal utama — jatuh ke koneksi
	// bawaan berarti menampilkan klaim satu badan hukum kepada petugas badan hukum lain
	// tanpa satu pun pesan galat (`R-20`).
	service := newService(t, memory.NewSampleStore())

	_, err := service.List(context.Background(), "SMI", callerAt("078"), firstPage())
	require.Error(t, err)
}

func TestListCarriesThePlannedDifferences(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	listed, err := service.List(context.Background(), primaryPortal,
		callerAt("078"), firstPage())
	require.NoError(t, err)
	require.NotEmpty(t, listed.PlannedDifferences,
		"selisih terencana harus sampai ke layar, bukan berhenti di komentar kode")
}

func TestExportSessionAppliesTheSameGuardsAsTheList(t *testing.T) {
	// Berkas ekspor memuat nama tertanggung, nomor polis, dan nilai uang. Pemeriksaannya
	// tidak boleh lebih longgar daripada layarnya.
	service := newService(t, memory.NewSampleStore())

	_, err := service.BeginExport(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{DetailBranchCode: "078"})
	require.ErrorIs(t, err, inboxosclaimpercabang.ErrCallerUnknown)

	_, err = service.BeginExport(context.Background(), primaryPortal,
		inboxosclaimpercabang.Caller{Login: "MITRA1"})
	require.ErrorIs(t, err, inboxosclaimpercabang.ErrBranchUnknown)

	_, err = service.BeginExport(context.Background(), "SMI", callerAt("078"))
	require.Error(t, err)
}

func TestExportFillsAgingAndDominantFactors(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	session, err := service.BeginExport(context.Background(), primaryPortal,
		callerAt("078"))
	require.NoError(t, err)
	require.Equal(t, "CILEGON", session.Query.Branch.Name)

	page, err := session.Page(context.Background(), firstPage())
	require.NoError(t, err)
	require.Len(t, page.Items, 3)

	byNumber := map[string]inboxosclaimpercabang.ExportRow{}
	for _, item := range page.Items {
		byNumber[item.ClaimNumber] = item
	}

	require.Equal(t, 987, byNumber["PNC-9001"].AgingDays)
	require.Equal(t, "Kelalaian Tertanggung, Dokumen Tidak Lengkap",
		byNumber["PNC-9002"].DominantFactors)
	require.Equal(t, "-", byNumber["PNC-9001"].DominantFactors,
		"klaim tanpa faktor dominan ditulis '-', mengikuti berkas lama")
}

func TestExportPagesDoNotRepeatRows(t *testing.T) {
	service := newService(t, memory.NewSampleStore())

	session, err := service.BeginExport(context.Background(), primaryPortal,
		callerAt("078"))
	require.NoError(t, err)

	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		result, err := session.Page(context.Background(),
			inboxosclaimpercabang.Pagination{Page: page, Size: 1})
		require.NoError(t, err)
		require.Equal(t, 3, result.Total)

		for _, item := range result.Items {
			require.Falsef(t, seen[item.ClaimNumber],
				"%s muncul di lebih dari satu potong ekspor", item.ClaimNumber)
			seen[item.ClaimNumber] = true
		}
	}
	require.Len(t, seen, 3)
}
