package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxadmin"
	"claim-pnc/internal/inboxadmin/repo/memory"
	"claim-pnc/internal/inboxadmin/usecase"
)

// fixedClock adalah jam tetap, supaya hitungan Aging dapat diuji.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// now adalah hari yang dipakai seluruh uji di berkas ini.
var now = time.Date(2026, time.September, 20, 8, 0, 0, 0, time.UTC)

// caller adalah pemilik baris contoh pada tab yang ScopedToCaller.
var caller = inboxadmin.Caller{Login: memory.SampleOwner}

// newService membentuk layanan di atas contoh bawaan adapter memori.
func newService(t *testing.T) *usecase.Service {
	t.Helper()

	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return store, nil },
		Clock:        fixedClock{at: now},
	})
	require.NoError(t, err)
	return service
}

func list(t *testing.T, service *usecase.Service, input inboxadmin.QueryInput) usecase.Listed {
	t.Helper()

	listed, err := service.List(context.Background(), "utama", caller, input,
		inboxadmin.Pagination{Page: 1, Size: 50})
	require.NoError(t, err)
	return listed
}

func TestServiceRefusesToStartWithoutItsSeams(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: fixedClock{}})
	require.Error(t, err)

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return nil, nil },
	})
	require.Error(t, err)
}

func TestMetadataCarriesEveryTabAndTheDisabledOnes(t *testing.T) {
	meta := newService(t).Metadata()

	require.Len(t, meta.Tabs, 8)
	require.Len(t, meta.DisabledTabs, 3)
	require.Equal(t, inboxadmin.TabAllCaseAdmin, meta.DefaultTab)
	require.Len(t, meta.BusinessLines, 5)
}

func TestMetadataNeedsNoDatabase(t *testing.T) {
	// Daftar tab adalah bentuk LAYAR, bukan data entitas. Memanggilnya tidak boleh
	// menyentuh repo — kalau tidak, membuka layar saat basis data entitas sedang tidak
	// tersedia akan gagal tanpa alasan yang berhubungan.
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) {
			t.Fatal("Metadata tidak boleh memilih repo")
			return nil, nil
		},
		Clock: fixedClock{at: now},
	})
	require.NoError(t, err)

	require.NotEmpty(t, service.Metadata().Tabs)
}

func TestDefaultTabShowsOnlyRowsOwnedByTheCaller(t *testing.T) {
	// Contoh memuat satu baris milik ADMINLAIN dengan sengaja. Bila ia muncul, penyaring
	// kepemilikan tidak terpasang — dan itu berarti setiap petugas melihat pekerjaan
	// petugas lain.
	listed := list(t, newService(t), inboxadmin.QueryInput{})

	require.Equal(t, inboxadmin.TabAllCaseAdmin, listed.Query.Tab.Code)
	require.NotEmpty(t, listed.Page.Items)

	for _, item := range listed.Page.Items {
		require.NotEqual(t, "PNC-8899", item.CaseID,
			"baris milik pengguna lain bocor ke antrean pemanggil")
	}
}

func TestBusinessFilterFollowsTheOldPredicate(t *testing.T) {
	service := newService(t)

	all := list(t, service, inboxadmin.QueryInput{Tab: inboxadmin.TabAll})
	require.Equal(t, 4, all.Page.Total)

	pa := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Business: string(inboxadmin.BusinessPA),
	})
	require.Equal(t, 1, pa.Page.Total)
	require.Equal(t, "PNC-8802", pa.Page.Items[0].CaseID)

	travel := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Business: string(inboxadmin.BusinessTravel),
	})
	require.Equal(t, 1, travel.Page.Total)
}

func TestNonMBUExcludesBondingEvenWhenItsGroupPanelMatches(t *testing.T) {
	// Inilah bagian yang paling mudah salah dibaca: NONMBU BUKAN "selain PA dan Travel".
	// Baris contoh ber-Group Panel `009` termasuk keempat panel Non-MBU, tetapi
	// BUSINESSGROUPID-nya `10015` sehingga ia justru DIBUANG — dan ia satu-satunya yang
	// muncul di saringan BONDING.
	service := newService(t)

	nonMBU := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Business: string(inboxadmin.BusinessNonMBU),
	})
	for _, item := range nonMBU.Page.Items {
		require.NotEqual(t, "PNC-8804", item.CaseID, "baris Bonding bocor ke saringan Non-MBU")
	}

	bonding := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Business: string(inboxadmin.BusinessBonding),
	})
	require.Equal(t, 1, bonding.Page.Total)
	require.Equal(t, "PNC-8804", bonding.Page.Items[0].CaseID)
}

func TestKeywordOnlyTouchesCaseIDAndPolicyNumber(t *testing.T) {
	// Jangkauan kotak cari di sistem lama hanya `a.pyid` dan `a.policyno`. Mencari nama
	// tertanggung tidak pernah membuahkan hasil, dan layar menyatakannya sebagai
	// keterbatasan alih-alih membiarkannya dilaporkan sebagai cacat.
	service := newService(t)

	byCaseID := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Keyword: "PNC-8802",
	})
	require.Equal(t, 1, byCaseID.Page.Total)

	byPolicy := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Keyword: "16.005.2026",
	})
	require.Equal(t, 1, byPolicy.Page.Total)

	byInsuredName := list(t, service, inboxadmin.QueryInput{
		Tab: inboxadmin.TabAll, Keyword: "Aneka Sejahtera",
	})
	require.Equal(t, 0, byInsuredName.Page.Total)
}

func TestCourierSeparatesBothUnregisteredTabs(t *testing.T) {
	service := newService(t)

	normal := list(t, service, inboxadmin.QueryInput{Tab: inboxadmin.TabUnregisteredRCV})
	require.Equal(t, 1, normal.Page.Total)
	require.Equal(t, "RCV-2201", normal.Page.Items[0].CaseID)

	online := list(t, service, inboxadmin.QueryInput{Tab: inboxadmin.TabRCVOnline})
	require.Equal(t, 1, online.Page.Total)
	require.Equal(t, "RCV-2202", online.Page.Items[0].CaseID)
}

func TestAgingIsFilledFromTheServiceClock(t *testing.T) {
	listed := list(t, newService(t), inboxadmin.QueryInput{Tab: inboxadmin.TabAll})

	found := false
	for _, item := range listed.Page.Items {
		if item.CaseID != "PNCN.26.0007" {
			continue
		}
		found = true

		// Tanggal lapor 2026-09-02, dibaca 2026-09-20.
		require.NotNil(t, item.ReportAgingDays)
		require.Equal(t, 18, *item.ReportAgingDays)
	}
	require.True(t, found, "baris contoh tidak ditemukan")
}

func TestRCLPUCLTabIgnoresFiltersItDoesNotSupport(t *testing.T) {
	listed := list(t, newService(t), inboxadmin.QueryInput{
		Tab:      inboxadmin.TabRCLPUCL,
		Keyword:  "tidak-ada",
		Business: string(inboxadmin.BusinessPA),
	})

	// Kata kunci dibuang, sehingga barisnya tetap muncul.
	require.Equal(t, 1, listed.Page.Total)
	require.Empty(t, listed.Query.Keyword)
	require.Equal(t, inboxadmin.BusinessAll, listed.Query.Business)
}

func TestDisabledTabIsRefusedBeforeTouchingTheRepo(t *testing.T) {
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) {
			t.Fatal("tab yang tidak dibangun tidak boleh sampai ke repo")
			return nil, nil
		},
		Clock: fixedClock{at: now},
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "utama", caller,
		inboxadmin.QueryInput{Tab: "6"}, inboxadmin.Pagination{})

	var validation *inboxadmin.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestUnknownPortalFails(t *testing.T) {
	// Portal yang tidak dikenal WAJIB menghasilkan galat. Mengembalikan repo portal utama
	// sebagai jalan pintas berarti menampilkan antrean satu badan hukum kepada petugas
	// badan hukum lain tanpa satu pun pesan galat (`R-20`).
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) {
			return nil, errors.New("portal tidak dikenal")
		},
		Clock: fixedClock{at: now},
	})
	require.NoError(t, err)

	_, err = service.List(context.Background(), "entah", caller,
		inboxadmin.QueryInput{}, inboxadmin.Pagination{})
	require.Error(t, err)
}

func TestPaginationCutsInTheApplicationNotTheRepo(t *testing.T) {
	// Keputusan Work Owner 2026-09-20: paginasi direplikasi apa adanya. Repo menyerahkan
	// SELURUH baris, dan totalnya dihitung dari seluruh baris itu — bukan dari satu
	// halaman.
	store := memory.NewSampleStore()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxadmin.Repo, error) { return store, nil },
		Clock:        fixedClock{at: now},
	})
	require.NoError(t, err)

	listed, err := service.List(context.Background(), "utama", caller,
		inboxadmin.QueryInput{Tab: inboxadmin.TabAll},
		inboxadmin.Pagination{Page: 1, Size: 2})
	require.NoError(t, err)

	require.Len(t, listed.Page.Items, 2)
	require.Equal(t, 4, listed.Page.Total)
	require.Equal(t, 2, listed.Page.TotalPages())
}
