package inboxcompliance_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxcompliance"
)

func TestNewQueryTabBawaan(t *testing.T) {
	t.Parallel()

	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{})

	require.NoError(t, err)
	require.Equal(t, inboxcompliance.TabCompliance, query.Tab.Code,
		"layar yang baru dibuka seharusnya membuka tab Compliance")
	require.Equal(t, inboxcompliance.WorkbasketCompliance, query.Workbasket,
		"antrean yang dibaca seharusnya CompliancePNC, sesuai Flow/Register_Flow.xml:3769")
}

func TestNewQueryMemangkasSpasi(t *testing.T) {
	t.Parallel()

	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: "  " + inboxcompliance.TabCompliance + "  ",
	})

	require.NoError(t, err)
	require.Equal(t, inboxcompliance.TabCompliance, query.Tab.Code)
}

func TestNewQueryTabTidakDikenal(t *testing.T) {
	t.Parallel()

	_, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{Tab: "tidak-ada"})

	var validation *inboxcompliance.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, inboxcompliance.FieldTab, validation.Violations[0].Field)
}

// Post Audit dapat diminta sejak DDL `POOLDATA.T_CLAIM_COMPLIANCE_H` diterima 2026-09-24.
func TestNewQueryTabPostAudit(t *testing.T) {
	t.Parallel()

	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabPostAudit,
	})

	require.NoError(t, err)
	require.Equal(t, inboxcompliance.TabPostAudit, query.Tab.Code)
}

// Kedua tab ada, berurutan, dan keduanya sudah dapat dilayani.
func TestTabs(t *testing.T) {
	t.Parallel()

	tabs := inboxcompliance.Tabs()
	require.Len(t, tabs, 2)

	require.Equal(t, inboxcompliance.TabCompliance, tabs[0].Code,
		"Compliance tab pertama, sesuai urutan di InputCompliance_Section")
	require.Equal(t, inboxcompliance.TabPostAudit, tabs[1].Code)

	for _, tab := range tabs {
		require.Truef(t, tab.Available, "tab %s seharusnya sudah dapat dilayani", tab.Code)
		require.Emptyf(t, tab.Blocker, "tab %s tidak lagi punya penghalang", tab.Code)
		require.NotEmptyf(t, tab.Columns, "tab %s wajib punya kolom", tab.Code)
	}
}

// Mekanisme "tab belum siap" TETAP diuji meski tidak ada tab yang memakainya hari ini.
//
// Ia sengaja dipertahankan, bukan dibuang bersama penghalang Post Audit: modul ini masih
// akan menumbuhkan tab baru, dan yang membedakan "tab tidak dikenal" dari "tab menunggu
// artefak pihak lain" adalah jenis galat ini. Membuang mekanismenya sekarang berarti
// menulisnya lagi dari nol saat dibutuhkan — dan sementara itu, pengguna akan menerima
// "tab tidak dikenal" untuk tab yang sebenarnya ada.
func TestTabNotReadyErrorTetapDapatDibedakan(t *testing.T) {
	t.Parallel()

	pending := inboxcompliance.Tab{
		Code:      "contoh",
		Name:      "Contoh",
		Available: false,
		Blocker:   "menunggu artefak dari pihak lain",
	}

	err := inboxcompliance.NewTabNotReadyError(pending)

	require.ErrorIs(t, err, inboxcompliance.ErrTabNotReady)
	require.Equal(t, "contoh", err.Tab.Code)
	require.NotEmpty(t, err.Blocker)

	var validation *inboxcompliance.ValidationError
	require.False(t, errors.As(error(err), &validation),
		"tab yang belum siap BUKAN galat validasi — permintaannya sudah benar")
}

// Tabs mengembalikan salinan, sehingga pemanggil tidak dapat merusak daftar aslinya.
func TestTabsMengembalikanSalinan(t *testing.T) {
	t.Parallel()

	first := inboxcompliance.Tabs()
	first[0].Name = "diubah"

	require.Equal(t, "Compliance", inboxcompliance.Tabs()[0].Name)
}

func TestPaginationNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         inboxcompliance.Pagination
		wantPage   int
		wantSize   int
		wantOffset int
	}{
		{
			name:     "kosong memakai nilai bawaan",
			in:       inboxcompliance.Pagination{},
			wantPage: 1, wantSize: inboxcompliance.DefaultPageSize, wantOffset: 0,
		},
		{
			name:     "halaman nol dibetulkan, bukan ditolak",
			in:       inboxcompliance.Pagination{Page: 0, Size: 10},
			wantPage: 1, wantSize: 10, wantOffset: 0,
		},
		{
			name:     "halaman negatif dibetulkan",
			in:       inboxcompliance.Pagination{Page: -5, Size: 10},
			wantPage: 1, wantSize: 10, wantOffset: 0,
		},
		{
			name:     "ukuran melebihi batas dijepit ke MaxPageSize",
			in:       inboxcompliance.Pagination{Page: 2, Size: 5000},
			wantPage: 2, wantSize: inboxcompliance.MaxPageSize,
			wantOffset: inboxcompliance.MaxPageSize,
		},
		{
			name:     "halaman ketiga",
			in:       inboxcompliance.Pagination{Page: 3, Size: 20},
			wantPage: 3, wantSize: 20, wantOffset: 40,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			clean := tc.in.Normalize()
			require.Equal(t, tc.wantPage, clean.Page)
			require.Equal(t, tc.wantSize, clean.Size)
			require.Equal(t, tc.wantOffset, tc.in.Offset())
		})
	}
}

func TestPageTotalPages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		total int
		size  int
		want  int
	}{
		{name: "hasil kosong tetap satu halaman", total: 0, size: 25, want: 1},
		{name: "kurang dari satu halaman", total: 7, size: 25, want: 1},
		{name: "tepat satu halaman", total: 25, size: 25, want: 1},
		{name: "sisa satu baris menambah halaman", total: 26, size: 25, want: 2},
		{name: "beberapa halaman", total: 101, size: 25, want: 5},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			page := inboxcompliance.Page{
				Total:      tc.total,
				Pagination: inboxcompliance.Pagination{Page: 1, Size: tc.size},
			}
			require.Equal(t, tc.want, page.TotalPages())
		})
	}
}
