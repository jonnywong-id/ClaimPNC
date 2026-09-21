package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxprogressclaim"
	"claim-pnc/internal/inboxprogressclaim/repo/memory"
	"claim-pnc/internal/inboxprogressclaim/usecase"
)

// today adalah tanggal yang dipakai seluruh uji di berkas ini.
//
// Ia dipilih supaya baris contoh terbelah: dua baris sudah jatuh tempo pada tanggal ini,
// dua belum. Tanpa jam yang tetap, uji region Next Follow Up akan berubah hasilnya
// tergantung kapan ia dijalankan.
var today = time.Date(2026, time.September, 21, 9, 30, 0, 0, time.UTC)

// fixedClock adalah seam Clock yang selalu menjawab tanggal yang sama.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// failingRepo selalu gagal, dipakai memastikan galat penyimpanan tidak tertelan.
type failingRepo struct{}

func (failingRepo) ListClaims(
	context.Context, inboxprogressclaim.ClaimQuery, inboxprogressclaim.Pagination,
) (inboxprogressclaim.ClaimPage, error) {
	return inboxprogressclaim.ClaimPage{}, errors.New("basis data tidak dapat dihubungi")
}

func (failingRepo) ListPICSummary(
	context.Context, inboxprogressclaim.PICQuery,
) ([]inboxprogressclaim.PICSummary, error) {
	return nil, errors.New("basis data tidak dapat dihubungi")
}

// newService membentuk layanan di atas penyimpanan contoh.
func newService(t *testing.T) *usecase.Service {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxprogressclaim.Repo, error) { return store, nil },
		Clock:        fixedClock{at: today},
	})
	require.NoError(t, err)
	return service
}

// list adalah jalan pintas memanggil List dengan pemanggil yang sah.
func list(
	t *testing.T,
	service *usecase.Service,
	input inboxprogressclaim.QueryInput,
	page inboxprogressclaim.Pagination,
) usecase.Listed {
	t.Helper()

	listed, err := service.List(
		context.Background(), "asm",
		inboxprogressclaim.Caller{Login: memory.SampleOwner},
		input, page,
	)
	require.NoError(t, err)
	return listed
}

func TestServiceRequiresItsSeams(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: fixedClock{}})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxprogressclaim.Repo, error) { return nil, nil },
	})
	require.Error(t, err)
}

func TestMetadataDescribesTheWholeScreen(t *testing.T) {
	meta := newService(t).Metadata()

	require.Len(t, meta.Views, 4)
	require.Equal(t, inboxprogressclaim.ViewOutstanding, meta.DefaultView)
	require.Len(t, meta.BusinessLines, 4)
	require.Equal(t, 15, meta.PageSize)
	require.NotEmpty(t, meta.DeadControls)
}

func TestOutstandingReturnsEveryOpenClaim(t *testing.T) {
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewOutstanding},
		inboxprogressclaim.Pagination{})

	require.Equal(t, 4, listed.Claims.Total)
	require.Len(t, listed.Claims.Items, 4)
}

func TestNextFollowUpKeepsOnlyClaimsThatAreDue(t *testing.T) {
	// Dua baris jatuh tempo pada tanggal uji: satu sudah terlewat, satu tepat hari ini.
	// Baris yang tenggatnya masih di depan dan baris yang BELUM PERNAH punya catatan
	// tindak lanjut sama-sama tidak boleh muncul.
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewNextFollowUp},
		inboxprogressclaim.Pagination{})

	require.Equal(t, 2, listed.Claims.Total)

	numbers := []string{}
	for _, item := range listed.Claims.Items {
		numbers = append(numbers, item.ClaimNumber)
	}
	require.ElementsMatch(t, []string{"PNCN.26.0101", "PNCN.26.0102"}, numbers)
}

func TestDueTodayCountsAsDue(t *testing.T) {
	// Batas "hari ini" harus inklusif. Bila ia eksklusif, pekerjaan yang jatuh tempo hari
	// ini baru muncul besok — yaitu ketika ia sudah terlambat.
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewNextFollowUp},
		inboxprogressclaim.Pagination{})

	found := false
	for _, item := range listed.Claims.Items {
		if item.ClaimNumber == "PNCN.26.0102" {
			found = true
		}
	}
	require.True(t, found, "klaim yang jatuh tempo hari ini harus ikut muncul")
}

func TestKeywordSearchesClaimNumberPolicyNumberAndPIC(t *testing.T) {
	service := newService(t)

	byClaim := list(t, service, inboxprogressclaim.QueryInput{Keyword: "0103"},
		inboxprogressclaim.Pagination{})
	require.Equal(t, 1, byClaim.Claims.Total)

	byPolicy := list(t, service, inboxprogressclaim.QueryInput{Keyword: "00102"},
		inboxprogressclaim.Pagination{})
	require.Equal(t, 1, byPolicy.Claims.Total)

	// Kolom ketiga inilah yang mudah terlewat: mengetik nama petugas memunculkan seluruh
	// klaim yang ia tangani.
	byPIC := list(t, service, inboxprogressclaim.QueryInput{Keyword: "SITITEKNIK"},
		inboxprogressclaim.Pagination{})
	require.Equal(t, 1, byPIC.Claims.Total)
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{Keyword: "sititeknik"},
		inboxprogressclaim.Pagination{})

	require.Equal(t, 1, listed.Claims.Total)
}

func TestRowsAreOrderedByProcessDateWithUndatedRowsLast(t *testing.T) {
	// Urutan yang pasti adalah yang membuat paginasi dapat dipercaya: tanpa pemutus seri,
	// satu baris dapat muncul di dua halaman sementara baris lain tidak pernah muncul.
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewOutstanding},
		inboxprogressclaim.Pagination{})

	numbers := []string{}
	for _, item := range listed.Claims.Items {
		numbers = append(numbers, item.ClaimNumber)
	}

	require.Equal(t, []string{
		"PNCN.26.0101", "PNCN.26.0102", "PNCN.26.0103",
		"PNCN.26.0104", // tanpa tanggal proses — harus di belakang
	}, numbers)
}

func TestPaginationCutsAfterOrdering(t *testing.T) {
	service := newService(t)

	first := list(t, service,
		inboxprogressclaim.QueryInput{},
		inboxprogressclaim.Pagination{Page: 1, Size: 2})
	require.Len(t, first.Claims.Items, 2)
	require.Equal(t, 4, first.Claims.Total, "total adalah SELURUH baris, bukan satu halaman")
	require.Equal(t, 2, first.Claims.TotalPages())

	second := list(t, service,
		inboxprogressclaim.QueryInput{},
		inboxprogressclaim.Pagination{Page: 2, Size: 2})
	require.Len(t, second.Claims.Items, 2)
	require.NotEqual(t, first.Claims.Items[0].ClaimNumber, second.Claims.Items[0].ClaimNumber)

	beyond := list(t, service,
		inboxprogressclaim.QueryInput{},
		inboxprogressclaim.Pagination{Page: 9, Size: 2})
	require.Empty(t, beyond.Claims.Items)
	require.Equal(t, 4, beyond.Claims.Total, "total tetap benar di luar jangkauan halaman")
}

func TestPositionsSurviveTheJourney(t *testing.T) {
	// Satu klaim contoh berada di DUA posisi sekaligus — itulah yang menggantikan
	// penggabungan berkoma di GET_POSISI_PROGRESS_PNC.
	listed := list(t, newService(t),
		inboxprogressclaim.QueryInput{Keyword: "0101"},
		inboxprogressclaim.Pagination{})

	require.Len(t, listed.Claims.Items, 1)
	require.Equal(t,
		[]string{"SURVEY", "KOMITE"}, listed.Claims.Items[0].PositionNames())
}

func TestPerPICIsScopedToTheCallerAndOneBusinessLine(t *testing.T) {
	service := newService(t)

	listed := list(t, service, inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
	}, inboxprogressclaim.Pagination{})

	// Baris contoh memuat petugas lain pada lini bisnis yang sama. Ia tidak boleh ikut.
	require.Len(t, listed.PICRows, 1)
	require.Equal(t, memory.SampleOwner, listed.PICRows[0].PIC)
	require.Equal(t, 12, listed.PICRows[0].ClaimCount)

	// Lini bisnis lain menjawab rekap yang berbeda, bukan rekap yang sama.
	travel := list(t, service, inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessTravel),
	}, inboxprogressclaim.Pagination{})
	require.Len(t, travel.PICRows, 1)
	require.Equal(t, 4, travel.PICRows[0].ClaimCount)
}

func TestPerPICHonoursTheDateRange(t *testing.T) {
	service := newService(t)

	inside := list(t, service, inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
		From:     "2026-08-01",
		To:       "2026-08-31",
	}, inboxprogressclaim.Pagination{})
	require.Len(t, inside.PICRows, 1)

	outside := list(t, service, inboxprogressclaim.QueryInput{
		View:     inboxprogressclaim.ViewPerPIC,
		Business: string(inboxprogressclaim.BusinessNonMBU),
		From:     "2026-01-01",
		To:       "2026-01-31",
	}, inboxprogressclaim.Pagination{})
	require.Empty(t, outside.PICRows)
}

func TestEvaluationViewNeverTouchesStorage(t *testing.T) {
	// Region ini kosong di Pega. Memanggil penyimpanan untuknya berarti mengarang kueri
	// yang tidak pernah ada — dan repo yang selalu gagal membuktikan ia memang tidak
	// dipanggil.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxprogressclaim.Repo, error) { return failingRepo{}, nil },
		Clock:        fixedClock{at: today},
	})
	require.NoError(t, err)

	listed, err := service.List(
		context.Background(), "asm",
		inboxprogressclaim.Caller{Login: memory.SampleOwner},
		inboxprogressclaim.QueryInput{View: inboxprogressclaim.ViewEvaluation},
		inboxprogressclaim.Pagination{},
	)

	require.NoError(t, err)
	require.Empty(t, listed.Claims.Items)
	require.Empty(t, listed.PICRows)
}

func TestStorageFailureIsNotSwallowed(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxprogressclaim.Repo, error) { return failingRepo{}, nil },
		Clock:        fixedClock{at: today},
	})
	require.NoError(t, err)

	_, err = service.List(
		context.Background(), "asm",
		inboxprogressclaim.Caller{Login: memory.SampleOwner},
		inboxprogressclaim.QueryInput{}, inboxprogressclaim.Pagination{},
	)
	require.ErrorContains(t, err, "basis data tidak dapat dihubungi")
}

func TestUnknownPortalIsRejected(t *testing.T) {
	// Portal yang tidak dikenal WAJIB menghasilkan galat. Mengembalikan repo portal utama
	// sebagai jalan pintas berarti menampilkan progres klaim satu badan hukum kepada
	// petugas badan hukum lain tanpa satu pun pesan galat (`R-20`).
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxprogressclaim.Repo, error) {
			return nil, errors.New("portal " + alias + " tidak dikenal")
		},
		Clock: fixedClock{at: today},
	})
	require.NoError(t, err)

	_, err = service.List(
		context.Background(), "entitas-lain",
		inboxprogressclaim.Caller{Login: memory.SampleOwner},
		inboxprogressclaim.QueryInput{}, inboxprogressclaim.Pagination{},
	)
	require.ErrorContains(t, err, "tidak dikenal")
}

func TestAppliedFilterIsReportedBack(t *testing.T) {
	// Layar menggambar keadaan penyaringnya dari sini, bukan dari isian yang ia kirim.
	listed := list(t, newService(t), inboxprogressclaim.QueryInput{
		View:    inboxprogressclaim.ViewOutstanding,
		Keyword: "  0101  ",
	}, inboxprogressclaim.Pagination{})

	require.Equal(t, "0101", listed.Query.Keyword, "kata kunci dipangkas sebelum dipakai")
}
