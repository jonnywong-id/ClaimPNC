package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxsurvey"
	"claim-pnc/internal/inboxsurvey/repo/memory"
	"claim-pnc/internal/inboxsurvey/usecase"
)

// stubDirectory mengembalikan identitas atau galat yang ditentukan.
type stubDirectory struct {
	identity inboxsurvey.SurveyorIdentity
	err      error
}

func (s stubDirectory) ResolveSurveyor(context.Context, string) (inboxsurvey.SurveyorIdentity, error) {
	return s.identity, s.err
}

// failingRepo selalu gagal.
type failingRepo struct{ err error }

func (f failingRepo) List(context.Context, inboxsurvey.SurveyorIdentity, inboxsurvey.Filter) (inboxsurvey.Page, error) {
	return inboxsurvey.Page{}, f.err
}

func (f failingRepo) Counts(context.Context, inboxsurvey.SurveyorIdentity) ([]inboxsurvey.TabCount, error) {
	return nil, f.err
}

func (f failingRepo) KPI(context.Context, inboxsurvey.SurveyorIdentity, inboxsurvey.KPIFilter) ([]inboxsurvey.KPIRow, error) {
	return nil, f.err
}

func sampleService(t *testing.T) *usecase.Service {
	t.Helper()
	store := memory.NewSampleStore()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxsurvey.Repo, error) {
			if alias != "ASM" {
				return nil, errors.New("repo belum siap")
			}
			return store, nil
		},
		DirectorySelector: func(alias string) (inboxsurvey.Directory, error) {
			if alias == "BAD" {
				return nil, errors.New("direktori belum siap")
			}
			return store, nil
		},
	})
	require.NoError(t, err)
	return svc
}

func serviceWith(t *testing.T, dir inboxsurvey.Directory, repo inboxsurvey.Repo) *usecase.Service {
	t.Helper()
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector:      func(string) (inboxsurvey.Repo, error) { return repo, nil },
		DirectorySelector: func(string) (inboxsurvey.Directory, error) { return dir, nil },
	})
	require.NoError(t, err)
	return svc
}

var leader = inboxsurvey.Caller{Login: memory.SampleLeaderLogin}

func TestNewServiceRequiresBothSelectors(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.ErrorContains(t, err, "RepoSelector wajib diisi")

	_, err = usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxsurvey.Repo, error) { return nil, nil },
	})
	require.ErrorContains(t, err, "DirectorySelector wajib diisi")
}

func TestMetadataDescribesTabsColumnsAndDefaults(t *testing.T) {
	meta := sampleService(t).Metadata()

	require.Len(t, meta.Columns, 13)

	// "Appointment No" TERSEDIA sejak 2026-10-03. Ia diturunkan dari CASEID — persis seperti
	// Pega memotongnya dari kunci objek kerja — sehingga tidak menunggu kolom dari siapa pun.
	require.Equal(t, "appointment_no", meta.Columns[0].Key)
	require.True(t, meta.Columns[0].Available)

	// Yang masih tertahan tinggal "Reference No": kolom REFNO sudah ada tetapi masih kosong.
	require.Equal(t, "reference_no", meta.Columns[1].Key)
	require.False(t, meta.Columns[1].Available)
	require.Len(t, meta.Tabs, len(inboxsurvey.Tabs()))
	for _, tab := range meta.Tabs {
		require.Equal(t, tab.Key.Available(), tab.Available)
		require.Equal(t, inboxsurvey.UnavailableReason(tab.Key), tab.UnavailableReason)
	}
	require.Len(t, meta.KPIColumns, 9)
	require.Equal(t, inboxsurvey.DefaultAvailableTab(), meta.DefaultTab)
	require.Equal(t, inboxsurvey.DefaultLimit, meta.PageSize)
	require.Len(t, meta.PlannedDifferences, 6)
	require.Len(t, meta.Limitations, 10)

	cols := usecase.Columns()
	cols[0].Key = "rusak"
	require.Equal(t, "appointment_no", usecase.Columns()[0].Key)
	kpi := usecase.KPIColumns()
	kpi[0].Key = "rusak"
	require.Equal(t, "penjadwalan_survey", usecase.KPIColumns()[0].Key)
}

func TestListNormalizesFilterAndReturnsIdentity(t *testing.T) {
	listed, err := sampleService(t).List(context.Background(), "ASM", leader,
		inboxsurvey.Filter{Tab: inboxsurvey.TabNotAnswered, Limit: 500})
	require.NoError(t, err)
	require.Equal(t, memory.SampleLeaderLogin, listed.Identity.Login)
	require.Equal(t, inboxsurvey.MaxLimit, listed.Filter.Limit)
	require.Equal(t, inboxsurvey.TabNotAnswered, listed.Filter.Tab)
}

func TestCountsAndKPIReturnIdentity(t *testing.T) {
	svc := sampleService(t)

	counted, err := svc.Counts(context.Background(), "ASM", leader)
	require.NoError(t, err)
	require.Equal(t, memory.SampleLeaderLogin, counted.Identity.Login)
	require.NotEmpty(t, counted.Counts)

	scored, err := svc.KPI(context.Background(), "ASM", leader, inboxsurvey.KPIFilter{Kind: "?"})
	require.NoError(t, err)
	require.Equal(t, inboxsurvey.KPIOutstanding, scored.Filter.Kind)
}

func TestPrepareRejectsUnknownCallerAndOutsider(t *testing.T) {
	svc := sampleService(t)

	_, err := svc.List(context.Background(), "ASM", inboxsurvey.Caller{}, inboxsurvey.Filter{})
	require.ErrorIs(t, err, inboxsurvey.ErrCallerUnknown)

	_, err = svc.Counts(context.Background(), "ASM",
		inboxsurvey.Caller{Login: memory.SampleOutsiderLogin})
	require.ErrorIs(t, err, inboxsurvey.ErrNotSurveyor)
}

func TestPreparePropagatesSelectorErrors(t *testing.T) {
	svc := sampleService(t)

	_, err := svc.KPI(context.Background(), "BAD", leader, inboxsurvey.KPIFilter{})
	require.EqualError(t, err, "direktori belum siap")

	_, err = svc.KPI(context.Background(), "ASI", leader, inboxsurvey.KPIFilter{})
	require.EqualError(t, err, "repo belum siap")
}

func TestPrepareWrapsDirectoryFailureAndRejectsEmptyScope(t *testing.T) {
	boom := errors.New("ldap mati")
	_, err := serviceWith(t, stubDirectory{err: boom}, failingRepo{}).
		List(context.Background(), "ASM", leader, inboxsurvey.Filter{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "menerjemahkan identitas surveyor")

	_, err = serviceWith(t, stubDirectory{identity: inboxsurvey.SurveyorIdentity{Login: "X"}},
		failingRepo{}).List(context.Background(), "ASM", leader, inboxsurvey.Filter{})
	require.ErrorIs(t, err, inboxsurvey.ErrNotSurveyor)
}

func TestRepoFailuresAreWrapped(t *testing.T) {
	boom := errors.New("kueri gagal")
	dir := stubDirectory{identity: inboxsurvey.SurveyorIdentity{Login: "X", Scope: []string{"X"}}}
	svc := serviceWith(t, dir, failingRepo{err: boom})

	_, err := svc.List(context.Background(), "ASM", leader, inboxsurvey.Filter{})
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "mengambil antrean survei")

	_, err = svc.Counts(context.Background(), "ASM", leader)
	require.ErrorContains(t, err, "menghitung isi tab antrean survei")

	_, err = svc.KPI(context.Background(), "ASM", leader, inboxsurvey.KPIFilter{})
	require.ErrorContains(t, err, "mengambil ringkasan KPI adjuster")
}

// TestKolomPenggantiDitandai mengunci keadaan ketiga sebuah kolom.
//
// # Kenapa uji ini ada
//
// `Available` hanya membedakan terisi dari kosong. Ia tidak dapat menyatakan keadaan yang
// paling berbahaya di antara keduanya: **terisi, tampak wajar, dan bukan angka yang sama
// dengan Pega**.
//
// Ketiga kolom di bawah persis begitu. Pega membacanya dari kolom objek kerja yang
// `POOLDATA.T_SURVEYORLIST` tidak punya sama sekali, sehingga modul ini menggambar pengganti.
// Tanpa penanda, selisihnya tidak terlihat siapa pun — dan selisih yang tidak terlihat tidak
// pernah dilaporkan.
func TestKolomPenggantiDitandai(t *testing.T) {
	// `status_asm` SEMPAT ada di daftar ini dan KELUAR pada 2026-10-03: usulan Work Owner
	// memakai `T_CLAIM_PNC.LEADER_MEMBER` terbukti setara dengan `ASMSTATUS_1` — nol
	// pertentangan pada 17.633 baris — sehingga kolomnya bukan pengganti lagi, melainkan
	// setara. Dicatat supaya tidak dimasukkan kembali.
	pengganti := map[string]bool{
		"pic_loss_adjuster": true, // Pega: ADJUSTERPIC_1 · di sini: SURVEYOR_NAME
		"location":          true, // Pega: RescheduleLocation_1 · di sini: LOCATION_SURVEY
	}

	for _, c := range usecase.Columns() {
		if pengganti[c.Key] {
			require.Truef(t, c.Substitute, "kolom %s menggambar pengganti tetapi tidak ditandai", c.Key)
			require.Truef(t, c.Available, "kolom %s terisi, jadi harus tetap tersedia", c.Key)
			require.Containsf(t, c.Note, "PENGGANTI",
				"keterangan kolom %s harus menyebut kolom Pega yang digantikannya", c.Key)
			continue
		}
		require.Falsef(t, c.Substitute, "kolom %s ditandai pengganti tanpa alasan", c.Key)
	}
}
