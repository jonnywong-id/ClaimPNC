package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladla"
	"claim-pnc/internal/inboxpladla/repo/memory"
	"claim-pnc/internal/inboxpladla/usecase"
)

// errBoom adalah galat penyimpanan buatan.
var errBoom = errors.New("penyimpanan rusak")

// failingRepo meneruskan seluruh panggilan ke penyimpanan memori, kecuali satu operasi
// yang sengaja digagalkan.
type failingRepo struct {
	inboxpladla.Repo
	failOn string
}

func (f failingRepo) fail(name string) bool { return f.failOn == name }

func (f failingRepo) ReinsurerCodes(ctx context.Context, login string) ([]string, error) {
	if f.fail("codes") {
		return nil, errBoom
	}
	return f.Repo.ReinsurerCodes(ctx, login)
}

func (f failingRepo) List(
	ctx context.Context, q inboxpladla.Query, p inboxpladla.Pagination,
) (inboxpladla.Page, error) {
	if f.fail("list") {
		return inboxpladla.Page{}, errBoom
	}
	return f.Repo.List(ctx, q, p)
}

func (f failingRepo) Counts(
	ctx context.Context, q inboxpladla.Query,
) ([]inboxpladla.StatusCount, error) {
	if f.fail("counts") {
		return nil, errBoom
	}
	return f.Repo.Counts(ctx, q)
}

func (f failingRepo) ClaimHeader(
	ctx context.Context, s inboxpladla.DetailScope,
) (inboxpladla.ClaimHeader, error) {
	if f.fail("header") {
		return inboxpladla.ClaimHeader{}, errBoom
	}
	return f.Repo.ClaimHeader(ctx, s)
}

func (f failingRepo) Advices(
	ctx context.Context, s inboxpladla.DetailScope, k inboxpladla.AdviceKind,
) ([]inboxpladla.AdviceRow, error) {
	if f.fail("advices-" + string(k)) {
		return nil, errBoom
	}
	return f.Repo.Advices(ctx, s, k)
}

func (f failingRepo) Documents(
	ctx context.Context, s inboxpladla.DetailScope, no string, k inboxpladla.AdviceKind,
) ([]inboxpladla.DocumentRow, error) {
	if f.fail("documents") {
		return nil, errBoom
	}
	return f.Repo.Documents(ctx, s, no, k)
}

func (f failingRepo) Conversations(
	ctx context.Context, s inboxpladla.DetailScope,
) ([]inboxpladla.Conversation, error) {
	if f.fail("conversations") {
		return nil, errBoom
	}
	return f.Repo.Conversations(ctx, s)
}

// sampleCaller adalah mitra pada data contoh.
var sampleCaller = inboxpladla.Caller{Login: memory.SampleReinsurerLogin, Name: "Mitra Contoh"}

const sampleKey = "ASM-FW-GCNMFW-WORK PNC-2001"

func newService(t *testing.T, repo inboxpladla.Repo, logs *bytes.Buffer) *usecase.Service {
	t.Helper()
	var logger *slog.Logger
	if logs != nil {
		logger = slog.New(slog.NewJSONHandler(logs, nil))
	}
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxpladla.Repo, error) {
			if alias != "ASM" {
				return nil, errors.New("portal tidak siap")
			}
			return repo, nil
		},
		Logger: logger,
	})
	require.NoError(t, err)
	return service
}

func TestNewServiceRequiresARepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

func TestMetadataListsTheTabs(t *testing.T) {
	meta := newService(t, memory.NewSampleStore(), nil).Metadata()

	require.Equal(t, inboxpladla.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxpladla.Tabs(), meta.Tabs)
	require.Equal(t, inboxpladla.PlannedDifferences, meta.PlannedDifferences)
}

func TestListReturnsTheRowsAndLogsTheOpening(t *testing.T) {
	logs := &bytes.Buffer{}
	listed, err := newService(t, memory.NewSampleStore(), logs).List(
		context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: inboxpladla.TabPLA}, inboxpladla.Pagination{},
	)

	require.NoError(t, err)
	require.Equal(t, inboxpladla.TabPLA, listed.Query.Tab.Code)
	require.NotEmpty(t, listed.Page.Items)
	require.Contains(t, logs.String(), "daftar PLA/DLA reasuradur dibuka")
	require.Contains(t, logs.String(), `"kode_reasuradur":["R100"]`)
}

func TestListWithoutLoggerStillWorks(t *testing.T) {
	listed, err := newService(t, memory.NewSampleStore(), nil).List(
		context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: inboxpladla.TabPLA}, inboxpladla.Pagination{},
	)
	require.NoError(t, err)
	require.NotEmpty(t, listed.Page.Items)
}

// "xol" ditolak sebagai tab TAK DIKENAL, bukan sebagai "bukan daftar klaim".
//
// Tampilan itu dicabut 2026-10-09: wadahnya di Pega bersyarat `TempView.CityID==7`,
// dan nilai itu tidak pernah dapat tercapai. Lihat TestTheXOLViewIsNotOfferedAtAll.
func TestListRefusesTheXOLView(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.List(context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: "xol"}, inboxpladla.Pagination{})
	var validation *inboxpladla.ValidationError
	require.ErrorAs(t, err, &validation)

	_, err = service.Counts(context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: "xol"})
	require.ErrorAs(t, err, &validation)
}

func TestListRejectsAnUnknownTab(t *testing.T) {
	_, err := newService(t, memory.NewSampleStore(), nil).List(
		context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: "tidak-ada"}, inboxpladla.Pagination{})

	var validation *inboxpladla.ValidationError
	require.ErrorAs(t, err, &validation)
}

func TestTheCallerMustBeKnownAndARegisteredReinsurer(t *testing.T) {
	logs := &bytes.Buffer{}
	service := newService(t, memory.NewSampleStore(), logs)

	_, err := service.List(context.Background(), "ASM", inboxpladla.Caller{Login: "  "},
		inboxpladla.QueryInput{}, inboxpladla.Pagination{})
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)

	_, err = service.List(context.Background(), "ASM", inboxpladla.Caller{Login: "PEGAWAI"},
		inboxpladla.QueryInput{}, inboxpladla.Pagination{})
	require.ErrorIs(t, err, inboxpladla.ErrCallerNotAReinsurer)
	require.Contains(t, logs.String(), "pemanggil bukan reasuradur terdaftar")

	// Tanpa pencatat, penolakannya tetap sama.
	_, err = newService(t, memory.NewSampleStore(), nil).List(
		context.Background(), "ASM", inboxpladla.Caller{Login: "PEGAWAI"},
		inboxpladla.QueryInput{}, inboxpladla.Pagination{})
	require.ErrorIs(t, err, inboxpladla.ErrCallerNotAReinsurer)
}

func TestAPortalThatIsNotReadyIsReportedAsIs(t *testing.T) {
	_, err := newService(t, memory.NewSampleStore(), nil).List(
		context.Background(), "ASI", sampleCaller,
		inboxpladla.QueryInput{}, inboxpladla.Pagination{})
	require.EqualError(t, err, "portal tidak siap")
}

func TestStorageFailuresAreWrappedWithTheirContext(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		failOn string
		call   func(*usecase.Service) error
		prefix string
	}{
		{"codes", func(s *usecase.Service) error {
			_, err := s.List(ctx, "ASM", sampleCaller,
				inboxpladla.QueryInput{}, inboxpladla.Pagination{})
			return err
		}, "mencari kode reasuradur pemanggil"},
		{"list", func(s *usecase.Service) error {
			_, err := s.List(ctx, "ASM", sampleCaller,
				inboxpladla.QueryInput{Tab: inboxpladla.TabPLA}, inboxpladla.Pagination{})
			return err
		}, "mengambil isi daftar pla"},
		{"counts", func(s *usecase.Service) error {
			_, err := s.Counts(ctx, "ASM", sampleCaller,
				inboxpladla.QueryInput{Tab: inboxpladla.TabPLA})
			return err
		}, "mengambil tabel ringkas pla"},
		{"header", func(s *usecase.Service) error {
			_, err := s.Detail(ctx, "ASM", sampleCaller, sampleKey)
			return err
		}, "mengambil keterangan klaim"},
		{"advices-PLA", func(s *usecase.Service) error {
			_, err := s.Detail(ctx, "ASM", sampleCaller, sampleKey)
			return err
		}, "mengambil daftar PLA"},
		{"advices-DLA", func(s *usecase.Service) error {
			_, err := s.Detail(ctx, "ASM", sampleCaller, sampleKey)
			return err
		}, "mengambil daftar DLA"},
		{"conversations", func(s *usecase.Service) error {
			_, err := s.Detail(ctx, "ASM", sampleCaller, sampleKey)
			return err
		}, "mengambil riwayat komunikasi"},
		{"documents", func(s *usecase.Service) error {
			_, err := s.Documents(ctx, "ASM", sampleCaller, sampleKey,
				"PLA/2026/2001-R1", inboxpladla.AdviceKindPLA)
			return err
		}, "mengambil dokumen PLA"},
	}

	for _, tc := range cases {
		t.Run(tc.failOn, func(t *testing.T) {
			repo := failingRepo{Repo: memory.NewSampleStore(), failOn: tc.failOn}
			err := tc.call(newService(t, repo, nil))
			require.ErrorIs(t, err, errBoom)
			require.Contains(t, err.Error(), tc.prefix)
		})
	}
}

func TestCountsFollowTheList(t *testing.T) {
	counts, err := newService(t, memory.NewSampleStore(), nil).Counts(
		context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Tab: inboxpladla.TabPLA})
	require.NoError(t, err)
	require.NotEmpty(t, counts)

	_, err = newService(t, memory.NewSampleStore(), nil).Counts(
		context.Background(), "ASM", inboxpladla.Caller{}, inboxpladla.QueryInput{})
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)
}

func TestDetailReturnsAllFourPartsAndTheirColumns(t *testing.T) {
	logs := &bytes.Buffer{}
	detail, err := newService(t, memory.NewSampleStore(), logs).
		Detail(context.Background(), "ASM", sampleCaller, " "+sampleKey+" ")

	require.NoError(t, err)
	require.Equal(t, "PNC-2001", detail.Claim.Header.ClaimNo)
	require.NotEmpty(t, detail.Claim.PLA)
	require.Equal(t, inboxpladla.AdviceColumns(inboxpladla.AdviceKindPLA), detail.AdviceColumnsPLA)
	require.Equal(t, inboxpladla.AdviceColumns(inboxpladla.AdviceKindDLA), detail.AdviceColumnsDLA)
	require.Equal(t, inboxpladla.DocumentColumns(), detail.DocumentColumns)
	require.Equal(t, inboxpladla.ConversationColumns(), detail.ConversationColumns)
	require.NotEmpty(t, detail.DocumentColumns)
	require.NotEmpty(t, detail.ConversationColumns)
	require.Contains(t, logs.String(), "rincian klaim dibuka reasuradur")
	require.NotContains(t, logs.String(), "PNC-2001", "nomor klaim tidak dicatat")
}

func TestDetailRefusesAnEmptyKeyAndAnUnknownCaller(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.Detail(context.Background(), "ASM", sampleCaller, "   ")
	require.ErrorIs(t, err, inboxpladla.ErrRowNotFound)

	_, err = service.Detail(context.Background(), "ASM", inboxpladla.Caller{}, sampleKey)
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)

	_, err = service.Detail(context.Background(), "ASM", sampleCaller,
		"ASM-FW-GCNMFW-WORK PNC-2008")
	require.ErrorIs(t, err, inboxpladla.ErrRowNotFound)
}

func TestDocumentsChecksTheKindFirst(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	_, err := service.Documents(context.Background(), "ASM", sampleCaller, sampleKey,
		"PLA/2026/2001-R1", inboxpladla.AdviceKind("XOL"))
	require.ErrorIs(t, err, inboxpladla.ErrAdviceKindUnknown)

	_, err = service.Documents(context.Background(), "ASM", inboxpladla.Caller{}, sampleKey,
		"PLA/2026/2001-R1", inboxpladla.AdviceKindPLA)
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)

	rows, err := service.Documents(context.Background(), "ASM", sampleCaller, sampleKey,
		"PLA/2026/2001-R1", inboxpladla.AdviceKindPLA)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "DOK-01", rows[0].ID)
}

func TestDocumentContentIsServedAndLogged(t *testing.T) {
	logs := &bytes.Buffer{}
	service := newService(t, memory.NewSampleStore(), logs)

	content, err := service.DocumentContent(context.Background(), "ASM", sampleCaller,
		sampleKey, "DOK-01")
	require.NoError(t, err)
	require.Equal(t, "laporan-kerugian.pdf", content.Name)
	require.Contains(t, logs.String(), "dokumen klaim diunduh reasuradur")

	_, err = service.DocumentContent(context.Background(), "ASM", sampleCaller,
		sampleKey, "DOK-03")
	require.ErrorIs(t, err, inboxpladla.ErrDocumentNotFound)

	_, err = service.DocumentContent(context.Background(), "ASM", inboxpladla.Caller{},
		sampleKey, "DOK-01")
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)

	// Tanpa pencatat isi dokumen tetap terserah.
	content, err = newService(t, memory.NewSampleStore(), nil).DocumentContent(
		context.Background(), "ASM", sampleCaller, sampleKey, "DOK-01")
	require.NoError(t, err)
	require.NotEmpty(t, content.Content)
}

func TestReplyValidatesStoresAndLogs(t *testing.T) {
	now := time.Date(2026, time.February, 8, 9, 0, 0, 0, time.UTC)
	logs := &bytes.Buffer{}
	store := memory.NewSampleStore()
	service := newService(t, store, logs)

	// Balasan kosong ditolak sebelum penyimpanan disentuh.
	err := service.Reply(context.Background(), "ASM", sampleCaller, sampleKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01"}, now)
	var validation *inboxpladla.ValidationError
	require.ErrorAs(t, err, &validation)

	// Portal tidak siap.
	err = service.Reply(context.Background(), "ASI", sampleCaller, sampleKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "ok"}, now)
	require.EqualError(t, err, "portal tidak siap")

	// Percakapan milik pihak lain.
	err = service.Reply(context.Background(), "ASM", sampleCaller, sampleKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-04", Message: "ok"}, now)
	require.ErrorIs(t, err, inboxpladla.ErrConversationNotFound)
	require.NotContains(t, logs.String(), "balasan komunikasi ditulis")

	err = service.Reply(context.Background(), "ASM", sampleCaller, sampleKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "Nilai disetujui."}, now)
	require.NoError(t, err)
	require.Contains(t, logs.String(), "balasan komunikasi ditulis reasuradur")
	require.NotContains(t, logs.String(), "Nilai disetujui.", "isi balasan tidak dicatat")

	conversations, err := store.Conversations(context.Background(), inboxpladla.DetailScope{
		ClaimKey: sampleKey, Login: memory.SampleReinsurerLogin, ReinsurerCodes: []string{"R100"},
	})
	require.NoError(t, err)
	require.Equal(t, "Nilai disetujui.", conversations[0].Reply)
}

func TestReplyWithoutLoggerStillStores(t *testing.T) {
	now := time.Date(2026, time.February, 8, 9, 0, 0, 0, time.UTC)
	err := newService(t, memory.NewSampleStore(), nil).Reply(
		context.Background(), "ASM", sampleCaller, sampleKey,
		inboxpladla.ReplyInput{ConversationID: "KOM-01", Message: "Disetujui."}, now)
	require.NoError(t, err)
}

func TestRejectActionNamesTheButton(t *testing.T) {
	err := newService(t, memory.NewSampleStore(), nil).
		RejectAction(string(inboxpladla.ActionDownloadAllDLA))

	var notAvailable *inboxpladla.NotAvailableError
	require.ErrorAs(t, err, &notAvailable)
	require.ErrorIs(t, err, inboxpladla.ErrWriteNotAvailable)
	require.NotEmpty(t, notAvailable.Reason())
}

// ── Tabel "Status / Jumlah" — satu baris per DAFTAR ─────────────────────────────

// Tabelnya menyebut KEENAM daftar, berapa pun isinya — termasuk yang kosong.
//
// Pega menggambar keenam barisnya meski seluruh angkanya nol (tangkapan layar Work Owner
// 2026-10-09 memperlihatkan persis itu), dan tabel yang menghilangkan barisnya saat nol
// membuat pengguna menyimpulkan daftarnya tidak ada — bukan bahwa daftarnya kosong.
func TestListCountsNamesEverySixListsEvenTheEmptyOnes(t *testing.T) {
	counts, err := newService(t, memory.NewSampleStore(), nil).ListCounts(
		context.Background(), "ASM", sampleCaller, inboxpladla.QueryInput{})
	require.NoError(t, err)

	wanted := []string{}
	for _, tab := range inboxpladla.Tabs() {
		wanted = append(wanted, tab.Code)
	}
	require.Len(t, wanted, 6, "layar ini punya enam daftar klaim")

	got := []string{}
	for _, row := range counts {
		got = append(got, row.Code)
		require.NotEmpty(t, row.Name, "nama daftar dipakai sebagai label kolom Status")
		require.GreaterOrEqual(t, row.Total, 0)
	}
	require.Equal(t, wanted, got, "urutannya mengikuti urutan daftar, bukan besar angkanya")
}

// Angka di tabel WAJIB sama dengan jumlah baris daftarnya.
//
// Inilah satu-satunya pernyataan yang menjaga janji di ListCounts: keduanya memakai
// pernyataan yang sama, sehingga selisih di antaranya berarti ada yang menempuh jalan
// lain. Selisih seperti itu tidak terlihat sebagai galat — hanya sebagai angka yang tidak
// cocok dengan tabel di sebelahnya.
func TestListCountsAgreeWithTheListItself(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	counts, err := service.ListCounts(
		context.Background(), "ASM", sampleCaller, inboxpladla.QueryInput{})
	require.NoError(t, err)

	for _, row := range counts {
		listed, err := service.List(
			context.Background(), "ASM", sampleCaller,
			inboxpladla.QueryInput{Tab: row.Code}, inboxpladla.Pagination{})
		require.NoError(t, err, "daftar %s", row.Code)
		require.Equal(t, listed.Page.Total, row.Total, "daftar %s", row.Code)
	}
}

// Pencarian IKUT dibawa ke setiap daftar.
//
// Tanpa ini, angka di samping daftar menyatakan populasi yang berbeda dari tabel yang
// sedang dilihat pengguna.
func TestListCountsCarryTheSearchIntoEveryList(t *testing.T) {
	service := newService(t, memory.NewSampleStore(), nil)

	semua, err := service.ListCounts(
		context.Background(), "ASM", sampleCaller, inboxpladla.QueryInput{})
	require.NoError(t, err)

	totalSemua := 0
	for _, row := range semua {
		totalSemua += row.Total
	}
	require.Positive(t, totalSemua, "premis: data contoh harus berisi sesuatu")

	tanpaHasil, err := service.ListCounts(
		context.Background(), "ASM", sampleCaller,
		inboxpladla.QueryInput{Search: "TIDAK-ADA-KLAIM-BERNOMOR-INI"})
	require.NoError(t, err)
	require.Len(t, tanpaHasil, len(semua), "barisnya tetap enam meski tak satu pun cocok")

	for _, row := range tanpaHasil {
		require.Zero(t, row.Total, "daftar %s", row.Code)
	}
}

// Pemanggil yang bukan mitra terdaftar DITOLAK, bukan menerima enam angka nol.
//
// Enam nol tidak dapat dibedakan dari "mitra ini memang belum punya klaim", dan layar
// akan menggambar tabel yang rapi untuk orang yang semestinya tidak melihat layar itu.
func TestListCountsRejectAnUnknownCaller(t *testing.T) {
	_, err := newService(t, memory.NewSampleStore(), nil).ListCounts(
		context.Background(), "ASM", inboxpladla.Caller{}, inboxpladla.QueryInput{})
	require.ErrorIs(t, err, inboxpladla.ErrCallerUnknown)
}
