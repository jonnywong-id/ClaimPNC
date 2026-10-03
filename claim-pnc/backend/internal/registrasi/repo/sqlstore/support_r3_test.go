package sqlstore

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// r3ID adalah pembuat pengenal tetap.
type r3ID string

func (i r3ID) New() string { return string(i) }

// Nomor klaim: tahun WIB, nomor terakhir ditambah satu, hanya di dalam transaksi.
func TestNumberIssuerIssue(t *testing.T) {
	db, mock := be4DB(t)
	_, err := NewNumberIssuer(db).Issue(context.Background(), time.Now())
	require.ErrorContains(t, err, "nomor klaim hanya boleh diterbitkan di dalam transaksi")

	ctx, _ := r3Tx(t, db, mock)
	at := time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC) // 2027 WIB
	mock.ExpectQuery(be4Q("nomor_terakhir_tahun")).WithArgs("PNCN.27.%").WillReturnRows(sqlmock.NewRows([]string{"n"}).AddRow(int64(41)))
	number, err := NewNumberIssuer(db).Issue(ctx, at)
	require.NoError(t, err)
	require.Equal(t, "PNCN.27.42", number)

	mock.ExpectQuery(be4Q("nomor_terakhir_tahun")).WillReturnError(be4Boom)
	_, err = NewNumberIssuer(db).Issue(ctx, at)
	require.ErrorContains(t, err, "membaca nomor terakhir tahun 2027")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuditRecorderRecord(t *testing.T) {
	db, mock := be4DB(t)
	r := NewAuditRecorder(db, r3ID("AUD-1"))
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, clock.ZoneWIB)
	long := strings.Repeat("x", 2100)
	mock.ExpectExec(be4Q("audit_sisip")).
		WithArgs("AUD-1", nil, "PNCN.26.1", "TUGAS_DIAMBIL", "NIK1", at.UTC(), long[:2000]).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, r.Record(context.Background(), registrasi.AuditTrail{
		ClaimNumber: "PNCN.26.1", Event: "TUGAS_DIAMBIL", Actor: "NIK1", At: at, Note: long,
	}))
	mock.ExpectExec(be4Q("audit_sisip")).WillReturnError(be4Boom)
	require.ErrorContains(t, r.Record(context.Background(), registrasi.AuditTrail{}), "merekam jejak audit")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifierSend(t *testing.T) {
	db, mock := be4DB(t)
	n := NewNotifier(db, r3ID("NTF-1"))
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)
	mock.ExpectExec(be4Q("notifikasi_sisip")).
		WithArgs("NTF-1", "LARGE_LOSS", "PNCN.26.1", nil, "a@x, b@x", int64(500), "1", at).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, n.Send(context.Background(), registrasi.Notification{
		Kind: "LARGE_LOSS", ClaimNumber: "PNCN.26.1", Recipients: []string{"a@x", "b@x"}, RupiahValue: 500,
		Revision: true, At: at,
	}))
	mock.ExpectExec(be4Q("notifikasi_sisip")).
		WithArgs("NTF-1", "", nil, "POL", "", int64(0), nil, at).
		WillReturnError(be4Boom)
	require.ErrorContains(t, n.Send(context.Background(), registrasi.Notification{PolicyNumber: "POL", At: at}), "mencatat pemberitahuan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTrimLimit(t *testing.T) {
	require.Equal(t, "abc", trim("abc", 3))
	require.Equal(t, "ab", trim("abc", 2))
}

// Laporan diserahkan: kosong ditolak, satu baris berubah selesai, nol baris dijelaskan.
func TestReportLinkMarkHandedOver(t *testing.T) {
	db, mock := be4DB(t)
	p := NewClaimReportLink(db)
	ctx := context.Background()
	at := time.Date(2026, 6, 9, 0, 0, 0, 0, time.UTC)

	require.ErrorContains(t, p.MarkHandedOver(ctx, " ", at), "nomor laporan kosong")

	mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WithArgs(at, "RCV1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, p.MarkHandedOver(ctx, " RCV1 ", at))

	mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WillReturnError(be4Boom)
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "menandai laporan RCV1 diserahkan")
	mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WillReturnResult(sqlmock.NewErrorResult(be4Boom))
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "membaca jumlah baris laporan")

	// Nol baris: sudah diserahkan = tidak apa-apa (idempoten).
	state := func(ours, handed, numbered int) {
		mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(be4Q("laporan_keadaan")).WithArgs("RCV1").
			WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(ours, handed, numbered))
	}
	state(1, 1, 0)
	require.NoError(t, p.MarkHandedOver(ctx, "RCV1", at))
	state(0, 0, 0)
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "dimiliki sistem lama")
	state(1, 0, 1)
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "tidak dapat diperbarui")

	mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(be4Q("laporan_keadaan")).WillReturnRows(sqlmock.NewRows(be4Cols(3)))
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "laporan RCV1 tidak ditemukan")
	mock.ExpectExec(be4Q("laporan_tandai_diserahkan")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(be4Q("laporan_keadaan")).WillReturnError(be4Boom)
	require.ErrorContains(t, p.MarkHandedOver(ctx, "RCV1", at), "membaca keadaan laporan RCV1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReportLinkAttachClaimNumber(t *testing.T) {
	db, mock := be4DB(t)
	p := NewClaimReportLink(db)
	ctx := context.Background()

	require.ErrorContains(t, p.AttachClaimNumber(ctx, "", "PNCN.26.1"), "nomor laporan kosong")
	require.ErrorContains(t, p.AttachClaimNumber(ctx, "RCV1", " "), "nomor klaim kosong untuk laporan RCV1")

	mock.ExpectExec(be4Q("laporan_pasang_nomor_klaim")).WithArgs("PNCN.26.1", "RCV1").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, p.AttachClaimNumber(ctx, "RCV1", " PNCN.26.1 "))

	mock.ExpectExec(be4Q("laporan_pasang_nomor_klaim")).WillReturnError(be4Boom)
	require.ErrorContains(t, p.AttachClaimNumber(ctx, "RCV1", "PNCN.26.1"), "memasang nomor klaim pada laporan RCV1")
	mock.ExpectExec(be4Q("laporan_pasang_nomor_klaim")).WillReturnResult(sqlmock.NewErrorResult(be4Boom))
	require.ErrorContains(t, p.AttachClaimNumber(ctx, "RCV1", "PNCN.26.1"), "membaca jumlah baris laporan")

	// Nol baris karena belum diserahkan: ditolak dengan penjelasan.
	mock.ExpectExec(be4Q("laporan_pasang_nomor_klaim")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(be4Q("laporan_keadaan")).WillReturnRows(sqlmock.NewRows(be4Cols(3)).AddRow(1, 0, 0))
	require.ErrorContains(t, p.AttachClaimNumber(ctx, "RCV1", "PNCN.26.1"), "belum ditandai diserahkan")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReportLinkSnapshot(t *testing.T) {
	db, mock := be4DB(t)
	p := NewClaimReportLink(db)
	ctx := context.Background()

	_, err := p.Snapshot(ctx, " ")
	require.ErrorContains(t, err, "nomor laporan kosong")

	loss := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(be4Q("laporan_isi")).WithArgs("RCV1").
		WillReturnRows(sqlmock.NewRows(be4Cols(10)).
			AddRow(loss, "2026-06-06", " Budi ", " 08 ", " b@x ", " Gudang ", " Kronologi ", 1234.5, " POL ", " PNCN.26.1 "))
	s, err := p.Snapshot(ctx, " RCV1 ")
	require.NoError(t, err)
	require.Equal(t, registrasi.ClaimReportSnapshot{
		DateOfLoss: loss, ReportDate: time.Date(2026, 6, 6, 0, 0, 0, 0, clock.ZoneWIB),
		ReporterName: "Budi", ReporterPhone: "08", ReporterEmail: "b@x", Location: "Gudang", Chronology: "Kronologi",
		EstimateValue: 123450, PolicyNumber: "POL", ClaimNumber: "PNCN.26.1",
	}, s)

	mock.ExpectQuery(be4Q("laporan_isi")).WillReturnRows(sqlmock.NewRows(be4Cols(10)))
	_, err = p.Snapshot(ctx, "RCV1")
	require.ErrorContains(t, err, "laporan RCV1 tidak ditemukan")
	mock.ExpectQuery(be4Q("laporan_isi")).WillReturnError(be4Boom)
	_, err = p.Snapshot(ctx, "RCV1")
	require.ErrorContains(t, err, "membaca isi laporan RCV1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestParseReportDate(t *testing.T) {
	require.True(t, parseReportDate("  ").IsZero())
	require.True(t, parseReportDate("06/06/2026").IsZero())
	require.Equal(t, time.Date(2026, 6, 6, 0, 0, 0, 0, clock.ZoneWIB), parseReportDate("2026-06-06"))
	require.Equal(t, time.Date(2026, 6, 6, 13, 5, 0, 0, clock.ZoneWIB), parseReportDate("2026-06-06 13:05:00"))
	require.True(t, parseReportDate("2026-06-06T13:05:00Z").Equal(time.Date(2026, 6, 6, 13, 5, 0, 0, time.UTC)))
}
