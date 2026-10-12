package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/monitoringslinkojk"
	"claim-pnc/internal/monitoringslinkojk/repo/memory"
	"claim-pnc/internal/monitoringslinkojk/usecase"
)

const portalUtama = "asm"

func newService(t *testing.T) *usecase.Service {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (monitoringslinkojk.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak siap")
			}
			return repo, nil
		},
	})
	require.NoError(t, err)
	return service
}

func request(segment monitoringslinkojk.Segment, filter monitoringslinkojk.Filter) usecase.Request {
	return usecase.Request{
		PortalAlias: portalUtama,
		Caller:      monitoringslinkojk.Caller{Login: "PELAPOR"},
		Segment:     segment,
		Filter:      filter,
	}
}

func date(year int, month time.Month, day int) *time.Time {
	value := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &value
}

func TestNewServiceRequiresRepoSelector(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
}

// ============================================================================
// KATALOG
// ============================================================================

func TestSegmentsReturnsBothWithColumnCounts(t *testing.T) {
	segments := newService(t).Segments()
	require.Len(t, segments, 2)

	require.Equal(t, monitoringslinkojk.SegmentD01, segments[0].Segment)
	require.Equal(t, "Segment Slik D01", segments[0].Label)
	require.Len(t, segments[0].Columns, 38)
	require.Equal(t, 8, segments[0].AvailableCount)

	require.Equal(t, monitoringslinkojk.SegmentF06, segments[1].Segment)
	require.Len(t, segments[1].Columns, 20)
	require.Equal(t, 20, segments[1].AvailableCount,
		"seluruh kolom grid F06 punya alias di GetDataSlinkAllFOG-SQL.xml")
}

func TestDescribeRejectsUnknownSegment(t *testing.T) {
	_, err := newService(t).Describe("D99")
	require.ErrorIs(t, err, monitoringslinkojk.ErrUnknownSegment)
}

// ============================================================================
// VALIDASI
// ============================================================================

func TestSearchRejectsUnknownCaller(t *testing.T) {
	_, err := newService(t).Search(context.Background(), usecase.Request{
		PortalAlias: portalUtama,
		Segment:     monitoringslinkojk.SegmentD01,
	})
	require.ErrorIs(t, err, monitoringslinkojk.ErrCallerUnknown)
}

// Rentang terbalik ditolak dengan PESAN, bukan dilayani dengan senarai kosong.
//
// Nol baris yang tampak wajar adalah keadaan paling berbahaya di layar ini: ia terbaca
// sebagai "tidak ada yang perlu dilaporkan ke OJK".
func TestSearchRejectsReversedDateRange(t *testing.T) {
	_, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			DateOfLoss:            date(2026, time.March, 31),
			DateOfRequestDocument: date(2026, time.March, 1),
		}))

	var validation *monitoringslinkojk.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, monitoringslinkojk.FieldDateOfLoss, validation.Violations[0].Field,
		"pesannya ditempel pada isian \"Dari\" saja; dua isian bertanda merah membuat pengguna menebak")
}

func TestSearchRejectsUnknownBusinessScope(t *testing.T) {
	_, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			BusinessScope: monitoringslinkojk.BusinessScope("MOTOR"),
		}))

	var validation *monitoringslinkojk.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, monitoringslinkojk.FieldBusinessScope, validation.Violations[0].Field)
}

// Seluruh pelanggaran dikembalikan sekaligus, meniru Pega yang menampilkan semua pesan
// bersamaan (`P-5`).
func TestSearchCollectsEveryViolation(t *testing.T) {
	_, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			BusinessScope:         monitoringslinkojk.BusinessScope("MOTOR"),
			DateOfLoss:            date(2026, time.March, 31),
			DateOfRequestDocument: date(2026, time.March, 1),
		}))

	var validation *monitoringslinkojk.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 2)
}

func TestSearchPropagatesPortalFailure(t *testing.T) {
	service := newService(t)
	_, err := service.Search(context.Background(), usecase.Request{
		PortalAlias: "entitas-lain",
		Caller:      monitoringslinkojk.Caller{Login: "PELAPOR"},
		Segment:     monitoringslinkojk.SegmentD01,
	})
	require.Error(t, err)
}

// ============================================================================
// PENCARIAN
// ============================================================================

func TestSearchReturnsEveryRowWhenUnfiltered(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))
	require.NoError(t, err)
	require.Equal(t, 4, page.Total)
	require.Len(t, page.Rows, 4)
}

// "SURETY BOND" MENIADAKAN Asuransi Kredit alih-alih memilih Surety Bond.
//
// Data contoh sengaja memuat satu baris berlini `Bonding`, yang HANYA muncul lewat
// pilihan ini — itulah satu-satunya cara membuktikan sifat meniadakannya.
func TestSearchBusinessScopeExcludesRatherThanSelects(t *testing.T) {
	service := newService(t)

	credit, err := service.Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			BusinessScope: monitoringslinkojk.ScopeCreditInsurance,
		}))
	require.NoError(t, err)
	require.Equal(t, 3, credit.Total)

	surety, err := service.Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			BusinessScope: monitoringslinkojk.ScopeSuretyBond,
		}))
	require.NoError(t, err)
	require.Equal(t, 1, surety.Total)
	require.Equal(t, "PNCN.26.0207", surety.Rows[0].Get("no_klaim"))
}

// Batas atas rentang bersifat INKLUSIF bagi penggunanya: klaim yang diregistrasi pukul
// 23.40 pada hari terakhir tetap ikut.
//
// Ini uji paling penting di berkas ini. Bila batasnya diperlakukan `<=` terhadap tengah
// malam, baris itu hilang dari laporan tanpa satu pun tanda.
func TestSearchIncludesLastDayOfRange(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			DateOfLoss:            date(2026, time.March, 1),
			DateOfRequestDocument: date(2026, time.March, 31),
		}))
	require.NoError(t, err)
	require.Equal(t, 3, page.Total)

	var found bool
	for _, row := range page.Rows {
		if row.Get("no_klaim") == "PNCN.26.0207" {
			found = true
		}
	}
	require.True(t, found,
		"klaim yang diregistrasi 31/03 pukul 23.40 wajib ikut pada rentang yang berakhir 31/03")
}

func TestSearchExcludesRowsOutsideRange(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			DateOfLoss:            date(2026, time.March, 1),
			DateOfRequestDocument: date(2026, time.March, 31),
		}))
	require.NoError(t, err)
	for _, row := range page.Rows {
		require.NotEqual(t, "PNCN.26.0033", row.Get("no_klaim"),
			"klaim Januari tidak boleh ikut pada rentang Maret")
	}
}

func TestSearchPaginates(t *testing.T) {
	service := newService(t)

	first, err := service.Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 1, Size: 2}))
	require.NoError(t, err)
	require.Len(t, first.Rows, 2)
	require.Equal(t, 4, first.Total)

	second, err := service.Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 2, Size: 2}))
	require.NoError(t, err)
	require.Len(t, second.Rows, 2)

	// Halaman di luar jangkauan dijawab kosong, BUKAN galat.
	beyond, err := service.Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 9, Size: 2}))
	require.NoError(t, err)
	require.Empty(t, beyond.Rows)
	require.Equal(t, 4, beyond.Total)
}

// Satu klaim dengan DUA fasilitas kredit tetap menghasilkan dua baris berkunci berbeda.
func TestSearchKeepsBothFacilitiesOfOneClaim(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))
	require.NoError(t, err)
	require.GreaterOrEqual(t, page.Total, 2)

	keys := map[string]bool{}
	for _, row := range page.Rows {
		key := row.Get(monitoringslinkojk.RowKeyColumn)
		require.Falsef(t, keys[key], "kunci baris ganda: %q", key)
		keys[key] = true
	}
}

// Segmen F06 mengembalikan kolom grid yang SAMA dengan D01.
//
// Uji ini sempat menuntut kebalikannya — hanya kolom bersumber, sisanya kosong — ketika
// grid F06 keliru dibangun dari judul berkas ekspor.
// Segmen F06 mengembalikan kolom FASILITAS KREDIT.
func TestSearchF06ReturnsFacilityColumns(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentF06, monitoringslinkojk.Filter{}))
	require.NoError(t, err)
	require.NotEmpty(t, page.Rows)

	row := page.Rows[0]
	for _, column := range monitoringslinkojk.Columns(monitoringslinkojk.SegmentF06) {
		_, exists := row[column.Key]
		require.Truef(t, exists, "kolom grid %q tidak ada pada hasil F06", column.Key)
	}

	require.NotEmpty(t, row.Get("no_klaim"))
	require.NotEmpty(t, row.Get("nomor_rekening_fasilitas"))
}

// Segmen D01 mengembalikan kolom IDENTITAS DEBITUR.
//
// Hanya delapan di antaranya bersumber; sisanya sengaja tidak ada di peta baris, dan
// Row.Get yang menerjemahkannya menjadi sel kosong.
func TestSearchD01ReturnsDebtorColumns(t *testing.T) {
	page, err := newService(t).Search(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))
	require.NoError(t, err)
	require.NotEmpty(t, page.Rows)

	row := page.Rows[0]
	require.NotEmpty(t, row.Get("nomor_cif_debitur"))
	require.Empty(t, row.Get("nama_lengkap"),
		"kolom tanpa sumber tidak boleh diisi tebakan")
}

// ============================================================================
// EKSPOR
// ============================================================================

func TestExportStreamsEveryMatchingRow(t *testing.T) {
	var collected []monitoringslinkojk.Row
	err := newService(t).Export(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{Page: 1, Size: 1}),
		func(row monitoringslinkojk.Row) error {
			collected = append(collected, row)
			return nil
		})
	require.NoError(t, err)
	require.Len(t, collected, 4,
		"ekspor mengabaikan paginasi; ukuran halaman 1 tidak boleh memotongnya")
}

// Galat dari penerima baris menghentikan pembacaan dan diteruskan APA ADANYA.
//
// Itulah yang memungkinkan transport berhenti ketika batas baris ekspor tercapai tanpa
// kesalahannya tersamar sebagai kegagalan basis data.
func TestExportStopsOnEmitError(t *testing.T) {
	stop := errors.New("berhenti")
	count := 0

	err := newService(t).Export(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		func(monitoringslinkojk.Row) error {
			count++
			if count == 2 {
				return stop
			}
			return nil
		})

	require.ErrorIs(t, err, stop)
	require.Equal(t, 2, count)
}

func TestExportRequiresEmitter(t *testing.T) {
	err := newService(t).Export(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}), nil)
	require.Error(t, err)
}

func TestExportValidatesBeforeReading(t *testing.T) {
	called := false
	err := newService(t).Export(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			DateOfLoss:            date(2026, time.March, 31),
			DateOfRequestDocument: date(2026, time.March, 1),
		}),
		func(monitoringslinkojk.Row) error {
			called = true
			return nil
		})

	var validation *monitoringslinkojk.ValidationError
	require.ErrorAs(t, err, &validation)
	require.False(t, called,
		"penyaring cacat harus ditolak SEBELUM satu baris pun dibaca")
}
