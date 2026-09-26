package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/reportklaim"
	"claim-pnc/internal/reportklaim/repo/memory"
	"claim-pnc/internal/reportklaim/usecase"
)

const portalUji = "ASM"

// jamTetap membuat hasil uji tidak bergantung pada saat ia dijalankan.
type jamTetap struct{ pada time.Time }

func (j jamTetap) Now() time.Time { return j.pada }

// pencatat merekam pemanggilan jejak audit.
type pencatat struct {
	dipanggil int
	terakhir  struct {
		caller reportklaim.Caller
		portal string
		report reportklaim.Report
		rows   int
	}
}

func (p *pencatat) ReportDownloaded(
	_ context.Context,
	caller reportklaim.Caller,
	portal string,
	report reportklaim.Report,
	_ reportklaim.Filter,
	rows int,
) {
	p.dipanggil++
	p.terakhir.caller = caller
	p.terakhir.portal = portal
	p.terakhir.report = report
	p.terakhir.rows = rows
}

func layanan(t *testing.T, audit reportklaim.AuditSink) *usecase.Service {
	t.Helper()
	s, err := usecase.NewService(usecase.Options{
		RepoSelector: memory.Selector(portalUji),
		Clock:        jamTetap{pada: time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)},
		Audit:        audit,
	})
	require.NoError(t, err)
	return s
}

func rentangSah() reportklaim.Filter {
	return reportklaim.Filter{
		From: time.Date(2026, 9, 1, 0, 0, 0, 0, clock.ZoneWIB),
		To:   time.Date(2026, 9, 30, 0, 0, 0, 0, clock.ZoneWIB),
	}
}

// Rakitan yang setengah jadi harus gagal saat start, bukan saat pengguna sedang bekerja.
func TestBahanYangTidakLengkapDitolakSaatPerakitan(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: jamTetap{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "RepoSelector")

	_, err = usecase.NewService(usecase.Options{RepoSelector: memory.Selector(portalUji)})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Clock")
}

// Panel yang terhalang tetap ikut di katalog. Menghilangkannya membuat pengguna
// melaporkan laporan yang "hilang", dan membuat kemajuan migrasi tidak terbaca dari layar.
func TestKatalogTetapMemuatPanelYangTerhalang(t *testing.T) {
	daftar := layanan(t, nil).Catalog()
	require.Len(t, daftar, 28)

	terhalang := 0
	for _, r := range daftar {
		if !r.Availability.Ready {
			terhalang++
		}
	}
	require.Equal(t, 3, terhalang)
}

func TestEksporMengalirkanBarisDanMenghitungnya(t *testing.T) {
	var diterima []reportklaim.Row
	report, rows, err := layanan(t, nil).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodePLA,
		Filter:      rentangSah(),
	}, func(row reportklaim.Row) error {
		diterima = append(diterima, row)
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "REPORT DATA PLA", report.Title)
	require.Equal(t, len(diterima), rows, "jumlah yang dilaporkan sama dengan yang terkirim")
	require.NotEmpty(t, diterima)
	require.Equal(t, "PNCN.26.0001", diterima[0].Value("CaseID"))
}

// Laporan yang tidak ada dan laporan yang terhalang DIBEDAKAN: yang satu salah ketik,
// yang lain penghalang migrasi. Menyatukannya membuat penghalang dicari di tempat salah.
func TestLaporanTidakDikenalDibedakanDariYangTerhalang(t *testing.T) {
	s := layanan(t, nil)
	buang := func(reportklaim.Row) error { return nil }

	_, _, err := s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji, Code: "tidak-ada", Filter: rentangSah(),
	}, buang)
	require.ErrorIs(t, err, reportklaim.ErrUnknownReport)

	_, _, err = s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji, Code: reportklaim.CodeAdjuster, Filter: rentangSah(),
	}, buang)
	require.ErrorIs(t, err, reportklaim.ErrReportNotReady)
}

// Katalog diperiksa SEBELUM penyaring: laporan yang belum siap dijawab begitu, bukan
// dijawab "tanggal wajib diisi" yang membuat pengguna mengisinya berkali-kali.
func TestLaporanTerhalangDitolakSebelumPenyaringDiperiksa(t *testing.T) {
	_, _, err := layanan(t, nil).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodeAdjuster,
		Filter:      reportklaim.Filter{}, // tanggal sengaja kosong
	}, func(reportklaim.Row) error { return nil })

	require.ErrorIs(t, err, reportklaim.ErrReportNotReady)

	var validasi *reportklaim.ValidationError
	require.False(t, errors.As(err, &validasi), "bukan galat validasi")
}

func TestPenyaringYangTidakSahDitolakSebelumBasisDataDisentuh(t *testing.T) {
	dipanggil := false
	_, _, err := layanan(t, nil).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodeTAT,
		Filter:      reportklaim.Filter{}, // tanggal wajib, sengaja kosong
	}, func(reportklaim.Row) error {
		dipanggil = true
		return nil
	})

	var validasi *reportklaim.ValidationError
	require.ErrorAs(t, err, &validasi)
	require.Len(t, validasi.Violation, 1)
	require.Equal(t, "dari", validasi.Violation[0].Field)
	require.False(t, dipanggil, "tidak satu baris pun boleh terkirim")
}

// Panel bertombol dua menolak permintaan tanpa kode tombol — memilihkan salah satunya
// berarti mengunduh laporan yang berbeda isi dari yang diminta.
func TestPanelBertombolDuaMenolakPermintaanTanpaTombol(t *testing.T) {
	s := layanan(t, nil)
	buang := func(reportklaim.Row) error { return nil }

	_, _, err := s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji, Code: reportklaim.CodeKomite, Filter: rentangSah(),
	}, buang)
	require.ErrorIs(t, err, reportklaim.ErrUnknownAction)

	_, _, err = s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodeKomite,
		Action:      reportklaim.ActionKomiteApprove,
		Filter:      rentangSah(),
	}, buang)
	require.NoError(t, err)
}

// Nilai milik TOMBOL tidak boleh datang dari pemanggil. Membiarkannya berarti
// membiarkan pemanggil memilih antara komite yang MENYETUJUI dan yang MENOLAK.
func TestParameterTombolDiisiUlangDariKatalog(t *testing.T) {
	var terlihat reportklaim.Filter
	audit := &pencatatFilter{lihat: func(f reportklaim.Filter) { terlihat = f }}

	s, err := usecase.NewService(usecase.Options{
		RepoSelector: memory.Selector(portalUji),
		Clock:        jamTetap{},
		Audit:        audit,
	})
	require.NoError(t, err)

	_, _, err = s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodeKomite,
		Action:      reportklaim.ActionKomiteRejected,
		Filter: reportklaim.Filter{
			From:       rentangSah().From,
			To:         rentangSah().To,
			FixedParam: map[string]string{"statusapprove": "1"}, // dikirim dari luar
		},
	}, func(reportklaim.Row) error { return nil })
	require.NoError(t, err)

	require.Equal(t, "2", terlihat.FixedParam["statusapprove"],
		"nilai dari luar harus ditimpa nilai milik tombol Rejected")
}

type pencatatFilter struct{ lihat func(reportklaim.Filter) }

func (p *pencatatFilter) ReportDownloaded(
	_ context.Context, _ reportklaim.Caller, _ string,
	_ reportklaim.Report, f reportklaim.Filter, _ int,
) {
	p.lihat(f)
}

// Portal yang tidak dikenal DITOLAK, tidak pernah dialihkan ke portal utama sebagai
// cadangan — itu berarti membaca data satu badan hukum dari basis data badan hukum lain
// tanpa satu pun pesan galat (`R-20`).
func TestPortalTidakDikenalDitolak(t *testing.T) {
	_, _, err := layanan(t, nil).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: "PORTAL-LAIN",
		Code:        reportklaim.CodePLA,
		Filter:      rentangSah(),
	}, func(reportklaim.Row) error { return nil })

	require.Error(t, err)
	require.Contains(t, err.Error(), "PORTAL-LAIN")
}

// Jejak audit dicatat SETELAH seluruh baris terkirim. Unduhan yang gagal di tengah bukan
// unduhan, dan mencatatnya sebagai berhasil membuat jejak audit berbohong.
func TestJejakAuditDicatatSetelahBerhasil(t *testing.T) {
	audit := &pencatat{}
	s := layanan(t, audit)

	_, rows, err := s.Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Caller:      reportklaim.Caller{Login: "PETUGAS", Name: "Petugas Uji"},
		Code:        reportklaim.CodePLA,
		Filter:      rentangSah(),
	}, func(reportklaim.Row) error { return nil })

	require.NoError(t, err)
	require.Equal(t, 1, audit.dipanggil)
	require.Equal(t, rows, audit.terakhir.rows)
	require.Equal(t, "PETUGAS", audit.terakhir.caller.Login)
	require.Equal(t, portalUji, audit.terakhir.portal)
	require.Equal(t, reportklaim.CodePLA, audit.terakhir.report.Code)
}

func TestUnduhanYangGagalDiTengahTidakDicatatSebagaiBerhasil(t *testing.T) {
	audit := &pencatat{}
	gagal := errors.New("sambungan terputus")

	_, _, err := layanan(t, audit).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji,
		Code:        reportklaim.CodePLA,
		Filter:      rentangSah(),
	}, func(reportklaim.Row) error { return gagal })

	require.ErrorIs(t, err, gagal)
	require.Zero(t, audit.dipanggil, "unduhan yang gagal bukan unduhan")
}

// Modul dapat dirakit TANPA pencatat, dan ketiadaannya tidak boleh menjatuhkan
// permintaan — hanya membuat jejaknya tidak tercatat.
func TestTanpaPencatatTetapBerjalan(t *testing.T) {
	_, rows, err := layanan(t, nil).Export(t.Context(), usecase.ExportRequest{
		PortalAlias: portalUji, Code: reportklaim.CodePLA, Filter: rentangSah(),
	}, func(reportklaim.Row) error { return nil })

	require.NoError(t, err)
	require.Positive(t, rows)
}

func TestPilihanBisnisDibacaDariPortalYangDiminta(t *testing.T) {
	s := layanan(t, nil)

	pilihan, err := s.BusinessOptions(t.Context(), portalUji)
	require.NoError(t, err)
	require.NotEmpty(t, pilihan)
	require.NotEmpty(t, pilihan[0].Code)
	require.NotEmpty(t, pilihan[0].Name)

	_, err = s.BusinessOptions(t.Context(), "PORTAL-LAIN")
	require.Error(t, err)
}
