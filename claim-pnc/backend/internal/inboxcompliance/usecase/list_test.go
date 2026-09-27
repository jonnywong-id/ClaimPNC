package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
	"claim-pnc/internal/inboxcompliance/repo/memory"
	"claim-pnc/internal/inboxcompliance/usecase"
	"claim-pnc/internal/platform/clock"
)

// fixedClock adalah jam tetap, sehingga hitungan Aging dapat diuji secara deterministik.
type fixedClock struct{ at time.Time }

func (c fixedClock) Now() time.Time { return c.at }

// Rabu 2026-09-23 pukul 09.00 WIB.
//
// Sengaja di tengah pekan: berkas ini menguji orkestrasi, bukan pemotongan akhir pekan.
// Basis hari Senin akan membuat "26 jam yang lalu" jatuh pada hari Minggu, sehingga Aging
// yang diuji di sini ikut memotong 24 jam — dan kegagalannya akan terbaca seperti cacat
// pengisian Aging, padahal ia pemotongan akhir pekan yang memang benar. Pemotongan itu
// diuji tersendiri di inboxcompliance/aging_test.go.
var now = time.Date(2026, time.September, 23, 9, 0, 0, 0, clock.ZoneWIB)

const portal = "utama"

func newService(t *testing.T, store *memory.Store) *usecase.Service {
	t.Helper()

	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxcompliance.Repo, error) {
			if alias != portal {
				return nil, errors.New("portal tidak dikenal: " + alias)
			}
			return store, nil
		},
		Clock: fixedClock{at: now},
	})
	require.NoError(t, err)

	return service
}

// sampleStore menyusun antrean yang isinya terkendali, bukan NewSampleStore yang
// tanggalnya relatif terhadap jam nyata.
func sampleStore() *memory.Store {
	sent := now.Add(-26 * time.Hour)

	return memory.NewStore(
		memory.Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-3 * time.Hour),
			Item: inboxcompliance.WorkItem{
				CaseID: "PNC-2", ComplianceSentDate: &sent,
			},
		},
		memory.Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-5 * time.Hour),
			Item:       inboxcompliance.WorkItem{CaseID: "PNC-1"},
		},
		memory.Row{
			Workbasket: inboxcompliance.WorkbasketCompliance,
			CreatedAt:  now.Add(-2 * time.Hour),
			Resolved:   true,
			Item:       inboxcompliance.WorkItem{CaseID: "PNC-SELESAI"},
		},
		memory.Row{
			Workbasket: "RCLPUCL",
			CreatedAt:  now.Add(-time.Hour),
			Item:       inboxcompliance.WorkItem{CaseID: "PNC-ANTREAN-LAIN"},
		},

		// Baris tab Post Audit. Ia TIDAK punya workbasket dan TIDAK punya status —
		// tabelnya memang tidak menyimpan keduanya.
		memory.Row{
			Tab: inboxcompliance.TabPostAudit,
			Item: inboxcompliance.WorkItem{
				CaseID:            "PNC-PA-1",
				Reference:         "ASM-FW-GCNMFW-WORK PNC-PA-1",
				PolicyNumber:      "POL-CONTOH-0009",
				InsuredName:       "Tertanggung Contoh Sembilan",
				PostAuditSentDate: timePtr(now.Add(-72 * time.Hour)),
				ComplianceRemarks: "Contoh catatan compliance",
			},
		},
	)
}

func timePtr(at time.Time) *time.Time { return &at }

func TestListMenyaringAntreanDanStatus(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	require.Equal(t, 2, listed.Page.Total,
		"klaim yang sudah selesai dan baris antrean lain tidak boleh ikut")

	require.Equal(t, []string{"PNC-2", "PNC-1"},
		[]string{listed.Page.Items[0].CaseID, listed.Page.Items[1].CaseID},
		"urutannya PXCREATEDATETIME DESC, sesuai Report Definition")
}

// Aging diisi usecase, bukan repo, dan memakai satu jam yang sama untuk seluruh baris.
func TestListMengisiAging(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	// PNC-2 dikirim 26 jam lalu pada hari kerja, sehingga tidak ada yang dipotong.
	require.NotNil(t, listed.Page.Items[0].AgingHours)
	require.InDelta(t, 26.0, *listed.Page.Items[0].AgingHours, 0.001)
	require.Equal(t, "1 days 2 hours ago", listed.Page.Items[0].AgingLabel())

	// PNC-1 tidak punya Tanggal Kirim Compliance, sehingga Aging-nya KOSONG — bukan
	// "0 hours ago" (`P-5` butir 13).
	require.Nil(t, listed.Page.Items[1].AgingHours)
	require.Empty(t, listed.Page.Items[1].AgingLabel())
}

func TestListMemaginasiDiRepo(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{},
		inboxcompliance.Pagination{Page: 2, Size: 1},
	)
	require.NoError(t, err)

	require.Len(t, listed.Page.Items, 1)
	require.Equal(t, "PNC-1", listed.Page.Items[0].CaseID)
	require.Equal(t, 2, listed.Page.Total, "total adalah seluruh baris, bukan isi halaman")
	require.Equal(t, 2, listed.Page.TotalPages())
}

// Tab Post Audit membaca baris yang BERBEDA dari tab Compliance, bukan baris yang sama
// dengan penyaring berbeda — keduanya memang membaca tabel yang berbeda.
func TestListTabPostAudit(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{Tab: inboxcompliance.TabPostAudit},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	require.Equal(t, 1, listed.Page.Total,
		"baris tab Compliance tidak boleh bocor ke tab Post Audit")
	require.Equal(t, "PNC-PA-1", listed.Page.Items[0].CaseID)
	require.Equal(t, "Contoh catatan compliance", listed.Page.Items[0].ComplianceRemarks)
	require.NotNil(t, listed.Page.Items[0].PostAuditSentDate)

	// Tab Post Audit tidak punya kolom Aging, dan tabelnya pun tidak menyimpan Tanggal
	// Kirim Compliance — sehingga Aging-nya kosong, bukan nol.
	require.Nil(t, listed.Page.Items[0].AgingHours)
}

// Baris tab Post Audit tidak boleh bocor ke tab Compliance, dan sebaliknya.
func TestListMemisahkanBarisAntarTab(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	compliance, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{Tab: inboxcompliance.TabCompliance},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	for _, item := range compliance.Page.Items {
		require.NotEqual(t, "PNC-PA-1", item.CaseID)
	}
}

// Portal yang tidak dikenal DITOLAK, tidak pernah dialihkan ke portal utama sebagai
// cadangan (`R-20`).
func TestListMenolakPortalTidakDikenal(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())

	_, err := service.List(
		context.Background(), "entitas-lain",
		inboxcompliance.QueryInput{},
		inboxcompliance.Pagination{},
	)

	require.Error(t, err)
}

func TestMetadata(t *testing.T) {
	t.Parallel()

	service := newService(t, sampleStore())
	meta := service.Metadata()

	require.Equal(t, inboxcompliance.TabCompliance, meta.DefaultTab)
	require.Len(t, meta.Tabs, 2)
	require.Len(t, meta.Tabs[0].Columns, 8,
		"tab Compliance punya delapan kolom data, sesuai InputComplianceDtl_Section")
	require.Len(t, meta.Tabs[1].Columns, 7,
		"tab Post Audit punya tujuh kolom, sesuai pyCaption di InputPostAuditDtl_Section")
}

func TestNewServiceMenolakBahanKosong(t *testing.T) {
	t.Parallel()

	_, err := usecase.NewService(usecase.Options{Clock: fixedClock{at: now}})
	require.Error(t, err, "RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxcompliance.Repo, error) { return nil, nil },
	})
	require.Error(t, err, "Clock wajib diisi")
}

// Tab Post Audit diurutkan menurut Nomor Case sebagai TEKS, menurun — meniru Pega.
//
// Layar Pega yang berjalan menampilkan `CPL-3, CPL-2, CPL-19, CPL-17, CPL-16, CPL-15,
// CPL-1`. Itu hanya masuk akal bila teksnya yang dibandingkan; mengurutkan angkanya akan
// menaruh `CPL-19` paling atas.
//
// Uji ini memakai urutan yang sama persis, sehingga ia gagal begitu seseorang
// "merapikannya" menjadi urutan angka.
func TestListPostAuditMengurutkanNomorCaseSebagaiTeks(t *testing.T) {
	t.Parallel()

	pega := []string{"CPL-3", "CPL-2", "CPL-19", "CPL-17", "CPL-16", "CPL-15", "CPL-1"}

	// Sengaja dimasukkan dalam urutan acak supaya yang diuji benar-benar pengurutannya.
	rows := make([]memory.Row, 0, len(pega))
	for _, code := range []string{"CPL-1", "CPL-19", "CPL-3", "CPL-15", "CPL-2", "CPL-17", "CPL-16"} {
		rows = append(rows, memory.Row{
			Tab:  inboxcompliance.TabPostAudit,
			Item: inboxcompliance.WorkItem{CaseID: code},
		})
	}

	service := newService(t, memory.NewStore(rows...))

	listed, err := service.List(
		context.Background(), portal,
		inboxcompliance.QueryInput{Tab: inboxcompliance.TabPostAudit},
		inboxcompliance.Pagination{},
	)
	require.NoError(t, err)

	got := make([]string, 0, len(listed.Page.Items))
	for _, item := range listed.Page.Items {
		got = append(got, item.CaseID)
	}

	require.Equal(t, pega, got)
}
