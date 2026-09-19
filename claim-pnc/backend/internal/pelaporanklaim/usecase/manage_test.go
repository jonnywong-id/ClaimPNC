package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/pelaporanklaim"
	"claim-pnc/internal/pelaporanklaim/repo/memory"
	"claim-pnc/internal/pelaporanklaim/usecase"
	"claim-pnc/internal/platform/clock"
)

// testNow adalah waktu tetap yang dipakai seluruh uji di berkas ini.
//
// Waktu tetap, bukan time.Now(): uji yang hasilnya bergantung pada jam dinding akan lulus
// hari ini dan gagal pada hari lain tanpa ada yang berubah di kode.
var testNow = time.Date(2026, time.September, 18, 7, 30, 0, 0, time.UTC)

func testService(t *testing.T, initial ...pelaporanklaim.ClaimReport) (*usecase.Service, *clock.Fixed) {
	t.Helper()

	clock := clock.FixedAt(testNow)
	service, err := usecase.NewService(usecase.Options{
		Repo:  memory.NewRepo(initial...),
		Clock: clock,
	})
	require.NoError(t, err)
	return service, clock
}

var testRecorder = usecase.Recorder{Login: "petugas.penerimaan", BranchCode: "100081"}

func testReport() pelaporanklaim.ClaimReport {
	return pelaporanklaim.ClaimReport{
		ReporterName: "Bagas Prasetya",
		SenderEmail:  "bagas@contoh.example",
		PolicyNumber: "CONTOH-PL-000117",
		InsuredName:  "PT Harapan Sentosa",
		Chronology:   "Api muncul dari panel listrik lantai satu.",
	}
}

// Bahan yang tidak lengkap ditolak saat start, bukan saat pengguna pertama membuka layar.
func TestServiceRejectsIncompleteOptions(t *testing.T) {
	_, err := usecase.NewService(usecase.Options{Clock: clock.FixedAt(testNow)})
	require.Error(t, err, "seam penyimpanan wajib diisi")

	_, err = usecase.NewService(usecase.Options{Repo: memory.NewRepo()})
	require.Error(t, err, "seam jam wajib diisi")
}

// Nomor laporan dibuat penyimpanan, TIDAK pernah diterima dari pemanggil.
//
// Menerima nomor dari luar akan membuat dua laporan memperebutkan nomor yang sama — dan
// yang kalah menimpa yang menang tanpa galat apa pun.
func TestNumberIssuedByStoreNotByCaller(t *testing.T) {
	service, _ := testService(t)

	sent := testReport()
	sent.Number = "NOMOR-KARANGAN"

	saved, err := service.Record(context.Background(), sent, testRecorder)
	require.NoError(t, err)
	require.NotEqual(t, "NOMOR-KARANGAN", saved.Number)
	require.NotEmpty(t, saved.Number)
}

// Empat field diisi sistem dan menimpa apa pun yang dikirim klien.
//
// Pencatat dan waktu datang dari sesi dan dari jam. Membiarkannya dikirim klien berarti
// siapa pun dapat mengaku mencatat atas nama orang lain — dan karena `D-59` menghapus
// pemisahan tugas, jejak siapa-mencatat-apa adalah satu-satunya kontrol yang tersisa.
func TestRecorderAndTimesComeFromSystemNotClient(t *testing.T) {
	service, _ := testService(t)

	sent := testReport()
	sent.CreatedBy = "orang.lain"
	sent.CreatedAt = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)

	saved, err := service.Record(context.Background(), sent, testRecorder)
	require.NoError(t, err)

	require.Equal(t, "petugas.penerimaan", saved.CreatedBy,
		"pencatat datang dari sesi, bukan dari badan permintaan")
	require.Equal(t, testNow, saved.CreatedAt)
	require.Equal(t, testNow, saved.UpdatedAt)
	require.Equal(t, "100081", saved.BranchCode)
}

// Laporan baru SELALU mulai dari tahap paling awal.
//
// Membiarkan klien menyatakan dirinya "sudah ditransfer" akan melewati satu-satunya
// langkah yang mencatat kapan dan oleh siapa transfer itu terjadi.
func TestNewReportAlwaysStartsAtEarliestStage(t *testing.T) {
	service, _ := testService(t)

	sent := testReport()
	sent.Transferred = true
	sent.ClaimNumber = "PNCN.26.9999"
	sent.Outcome = pelaporanklaim.OutcomeAccepted

	saved, err := service.Record(context.Background(), sent, testRecorder)
	require.NoError(t, err)

	require.False(t, saved.Transferred)
	require.Empty(t, saved.ClaimNumber)
	require.Equal(t, pelaporanklaim.OutcomeNone, saved.Outcome)
	require.Equal(t, pelaporanklaim.StageNotTransferred, saved.Stage())
}

// Kode cabang dari form dipakai bila profil pengguna tidak membawanya.
//
// Pengguna non-karyawan tidak punya cabang: `POOLDATA.M_LOGIN_PNC` tidak memuatnya.
// Tanpa jalan mundur ini, seluruh laporan yang dicatat broker akan tercatat tanpa cabang.
func TestBranchCodeFromFormUsedWhenProfileHasNone(t *testing.T) {
	service, _ := testService(t)

	sent := testReport()
	sent.BranchCode = "100099"

	saved, err := service.Record(context.Background(), sent,
		usecase.Recorder{Login: "broker.contoh"})
	require.NoError(t, err)
	require.Equal(t, "100099", saved.BranchCode)
}

// Profil pengguna MENANG atas isian form bila keduanya terisi.
//
// Cabang menentukan siapa yang kelak boleh melihat laporan ini (`11-SECURITY.md` §3.2).
// Membiarkan form menimpanya berarti petugas dapat mencatat laporan ke cabang lain.
func TestBranchCodeFromProfileBeatsForm(t *testing.T) {
	service, _ := testService(t)

	sent := testReport()
	sent.BranchCode = "100099" // cabang lain, dikirim dari form

	saved, err := service.Record(context.Background(), sent, testRecorder)
	require.NoError(t, err)
	require.Equal(t, "100081", saved.BranchCode,
		"form tidak boleh memindahkan laporan ke cabang lain")
}

// Validasi berjalan sebelum apa pun menyentuh penyimpanan.
func TestRecordRejectsReportWithoutReporterName(t *testing.T) {
	service, _ := testService(t)

	_, err := service.Record(context.Background(), pelaporanklaim.ClaimReport{}, testRecorder)

	var validation *pelaporanklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Len(t, validation.Violations, 1)
	require.Equal(t, pelaporanklaim.FieldReporterName, validation.Violations[0].Field)
}

// Transfer menandai laporan sudah dikirim ke ASM pusat, dan menolak transfer kedua.
//
// Transfer ulang TIDAK dibiarkan lolos diam-diam: menimpa tanggal transfer berarti
// menghapus kapan laporan itu benar-benar dikirim.
func TestTransferRejectsSecondTransfer(t *testing.T) {
	service, clock := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)

	clock.Advance(2 * time.Hour)
	moved, err := service.Transfer(context.Background(), recorded.Number)
	require.NoError(t, err)
	require.True(t, moved.Transferred)
	require.NotNil(t, moved.TransferredAt)
	require.Equal(t, testNow.Add(2*time.Hour), *moved.TransferredAt)
	require.Equal(t, pelaporanklaim.StageNotRegistered, moved.Stage())

	clock.Advance(time.Hour)
	_, err = service.Transfer(context.Background(), recorded.Number)
	require.ErrorIs(t, err, pelaporanklaim.ErrAlreadyTransferred)

	// Tanggal transfer yang pertama harus BERTAHAN.
	again, err := service.Get(context.Background(), recorded.Number)
	require.NoError(t, err)
	require.Equal(t, testNow.Add(2*time.Hour), *again.TransferredAt,
		"transfer kedua tidak boleh menimpa tanggal transfer yang pertama")
}

// Penautan klaim menandai laporan sudah diregistrasi, dan menolak penautan kedua.
//
// Satu laporan melahirkan satu klaim. Menautkannya ke klaim kedua akan membuat dua klaim
// mengaku berasal dari laporan yang sama, dan tidak ada apa pun sesudahnya yang dapat
// membedakan mana yang benar.
func TestLinkClaimRejectsSecondLink(t *testing.T) {
	service, clock := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)

	clock.Advance(3 * time.Hour)
	linked, err := service.LinkClaim(context.Background(), recorded.Number, "PNCN.26.0148")
	require.NoError(t, err)
	require.Equal(t, "PNCN.26.0148", linked.ClaimNumber)
	require.Equal(t, pelaporanklaim.StageRegistered, linked.Stage())

	_, err = service.LinkClaim(context.Background(), recorded.Number, "PNCN.26.0999")
	require.ErrorIs(t, err, pelaporanklaim.ErrAlreadyRegistered)
}

// Registrasi mengandaikan laporannya sudah sampai ASM.
//
// Laporan yang diregistrasi tanpa pernah ditransfer — mungkin karena dicatat langsung di
// pusat — ditandai ditransfer pada saat yang sama. Tanpa itu, ia akan tetap terbaca
// "belum ditransfer" padahal klaimnya sudah berjalan.
func TestLinkClaimAlsoMarksTransferWhenMissing(t *testing.T) {
	service, clock := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)
	require.False(t, recorded.Transferred)

	clock.Advance(time.Hour)
	linked, err := service.LinkClaim(context.Background(), recorded.Number, "PNCN.26.0148")
	require.NoError(t, err)

	require.True(t, linked.Transferred)
	require.NotNil(t, linked.TransferredAt)
	require.Equal(t, testNow.Add(time.Hour), *linked.TransferredAt)
}

// Nomor klaim kosong ditolak sebagai galat validasi, bukan disimpan sebagai penautan
// kosong yang memindahkan tahap tanpa isi.
func TestLinkClaimRejectsEmptyClaimNumber(t *testing.T) {
	service, _ := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)

	_, err = service.LinkClaim(context.Background(), recorded.Number, "   ")

	var validation *pelaporanklaim.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, pelaporanklaim.FieldClaimNumber, validation.Violations[0].Field)
}

// Laporan yang sudah diregistrasi TIDAK dapat diubah lagi.
//
// Sejak klaim terbit, yang berlaku adalah data klaimnya. Membiarkan laporannya tetap
// dapat disunting berarti dua versi nomor polis hidup berdampingan, dan tidak ada yang
// tahu mana yang dipakai.
func TestRegisteredReportCannotBeEdited(t *testing.T) {
	service, _ := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)
	_, err = service.LinkClaim(context.Background(), recorded.Number, "PNCN.26.0148")
	require.NoError(t, err)

	edited := testReport()
	edited.ReporterName = "Nama Lain"
	_, err = service.Update(context.Background(), recorded.Number, edited)
	require.ErrorIs(t, err, pelaporanklaim.ErrAlreadyRegistered)

	_, err = service.Transfer(context.Background(), recorded.Number)
	require.ErrorIs(t, err, pelaporanklaim.ErrAlreadyRegistered)
}

// Update TIDAK dapat memindahkan tahap.
//
// Perpindahan tahap punya jalurnya sendiri — Transfer dan LinkClaim — supaya ia tidak
// dapat terjadi sebagai efek samping penyuntingan biasa.
func TestUpdateCannotMoveStage(t *testing.T) {
	service, clock := testService(t)

	recorded, err := service.Record(context.Background(), testReport(), testRecorder)
	require.NoError(t, err)

	clock.Advance(time.Hour)
	edited := testReport()
	edited.ReporterName = "Nama Baru"
	edited.Transferred = true
	edited.ClaimNumber = "PNCN.26.9999"
	edited.Outcome = pelaporanklaim.OutcomeAccepted
	edited.CreatedBy = "orang.lain"

	updated, err := service.Update(context.Background(), recorded.Number, edited)
	require.NoError(t, err)

	require.Equal(t, "Nama Baru", updated.ReporterName, "isi yang boleh berubah memang berubah")
	require.False(t, updated.Transferred)
	require.Empty(t, updated.ClaimNumber)
	require.Equal(t, pelaporanklaim.OutcomeNone, updated.Outcome)
	require.Equal(t, "petugas.penerimaan", updated.CreatedBy, "pencatat tidak ikut berubah")
	require.Equal(t, testNow, updated.CreatedAt, "waktu pencatatan tidak ikut berubah")
	require.Equal(t, testNow.Add(time.Hour), updated.UpdatedAt)
}

// Laporan yang tidak ada dijawab "tidak ditemukan" pada keempat jalurnya.
func TestMissingReportAnsweredNotFound(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()

	_, err := service.Get(ctx, "LPK.00.9999")
	require.ErrorIs(t, err, pelaporanklaim.ErrNotFound)

	_, err = service.Update(ctx, "LPK.00.9999", testReport())
	require.ErrorIs(t, err, pelaporanklaim.ErrNotFound)

	_, err = service.Transfer(ctx, "LPK.00.9999")
	require.ErrorIs(t, err, pelaporanklaim.ErrNotFound)

	_, err = service.LinkClaim(ctx, "LPK.00.9999", "PNCN.26.0148")
	require.ErrorIs(t, err, pelaporanklaim.ErrNotFound)
}

// Tahap karangan ditolak, bukan diabaikan diam-diam.
//
// Penyaring yang diabaikan akan mengembalikan SELURUH baris, dan pemanggilnya mengira ia
// sudah tersaring — pada layar berisi data nasabah, itu berarti petugas melihat baris yang
// tidak ia minta.
func TestUnknownStageFilterRejected(t *testing.T) {
	service, _ := testService(t)

	_, err := service.List(context.Background(), pelaporanklaim.Filter{Stage: "KARANGAN"})

	var validation *pelaporanklaim.ValidationError
	require.ErrorAs(t, err, &validation)
}

// Ringkasan menghitung SELURUH tahap, tidak terpengaruh penyaring tahap yang sedang aktif.
//
// Angkanya dipakai lencana seluruh tab sekaligus. Menghormati penyaring tahap akan membuat
// setiap tab melaporkan dirinya sendiri sebagai satu-satunya yang berisi.
func TestSummaryIgnoresStageFilter(t *testing.T) {
	service, _ := testService(t, memory.SampleReports()...)
	ctx := context.Background()

	all, err := service.List(ctx, pelaporanklaim.Filter{})
	require.NoError(t, err)

	single, err := service.List(ctx, pelaporanklaim.Filter{
		Stage: pelaporanklaim.StageNotTransferred,
	})
	require.NoError(t, err)

	require.Equal(t, all.Summary, single.Summary,
		"ringkasan harus sama apa pun tab yang sedang terbuka")

	// Halaman-nya justru HARUS berbeda — itu yang membedakan penyaring yang bekerja dari
	// penyaring yang diabaikan.
	require.Less(t, single.Page.Total, all.Page.Total)
}

// Kelima tahap selalu ada di ringkasan, termasuk yang jumlahnya nol.
//
// Tab yang menghilang saat kosong membuat pengguna mengira tabnya tidak ada.
func TestSummaryAlwaysHasAllFiveStages(t *testing.T) {
	service, _ := testService(t)

	result, err := service.List(context.Background(), pelaporanklaim.Filter{})
	require.NoError(t, err)

	for _, stage := range []pelaporanklaim.Stage{
		pelaporanklaim.StageNotTransferred,
		pelaporanklaim.StageNotRegistered,
		pelaporanklaim.StageRegistered,
		pelaporanklaim.StageAccepted,
		pelaporanklaim.StageRejected,
	} {
		count, exists := result.Summary[stage]
		require.True(t, exists, "%s harus ada di ringkasan walau kosong", stage)
		require.Zero(t, count)
	}
}

// Laporan contoh mencakup kelima tahap.
//
// Tanpa contoh untuk kelimanya, tab yang kosong tidak dapat dibedakan antara "memang
// kosong" dan "rusak" saat layar dicoba tanpa basis data.
func TestSampleReportsCoverAllFiveStages(t *testing.T) {
	service, _ := testService(t, memory.SampleReports()...)

	result, err := service.List(context.Background(), pelaporanklaim.Filter{})
	require.NoError(t, err)

	for stage, count := range result.Summary {
		require.Positive(t, count, "tahap %s tidak punya satu pun contoh", stage)
	}
}

// Pencarian mencocokkan lima field yang dipakai orang mencari.
func TestSearchMatchesFiveFields(t *testing.T) {
	service, _ := testService(t, memory.SampleReports()...)
	ctx := context.Background()

	cases := []struct {
		name   string
		search string
	}{
		{"nomor laporan", "LPK.00.0001"},
		{"nomor klaim", "PNCN.26.0148"},
		{"nomor polis", "CONTOH-PL-000117"},
		{"nama tertanggung", "Harapan Sentosa"},
		{"nama pelapor", "Bagas"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result, err := service.List(ctx, pelaporanklaim.Filter{Search: c.search})
			require.NoError(t, err)
			require.Positive(t, result.Page.Total, "pencarian %q tidak menemukan apa pun", c.search)
		})
	}
}

// Pencarian tidak peduli besar-kecil huruf.
func TestSearchIsCaseInsensitive(t *testing.T) {
	service, _ := testService(t, memory.SampleReports()...)
	ctx := context.Background()

	upper, err := service.List(ctx, pelaporanklaim.Filter{Search: "HARAPAN SENTOSA"})
	require.NoError(t, err)
	lower, err := service.List(ctx, pelaporanklaim.Filter{Search: "harapan sentosa"})
	require.NoError(t, err)

	require.Equal(t, upper.Page.Total, lower.Page.Total)
	require.Positive(t, upper.Page.Total)
}

// Paginasi memotong hasil tetapi TIDAK memotong jumlah.
//
// Tanpa pembedaan ini, layar tidak dapat mengetahui masih ada halaman berikutnya.
func TestPaginationTrimsRowsNotTotal(t *testing.T) {
	service, _ := testService(t, memory.SampleReports()...)
	ctx := context.Background()

	first, err := service.List(ctx, pelaporanklaim.Filter{Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Page.Reports, 2)
	require.Equal(t, 6, first.Page.Total, "jumlah adalah seluruh yang cocok")

	second, err := service.List(ctx, pelaporanklaim.Filter{Limit: 2, Offset: 2})
	require.NoError(t, err)
	require.Len(t, second.Page.Reports, 2)
	require.Equal(t, 6, second.Page.Total)

	require.NotEqual(t, first.Page.Reports[0].Number,
		second.Page.Reports[0].Number, "halaman kedua harus berisi baris lain")
}

// Urutan daftar tetap di antara dua permintaan.
//
// Tanpa pemutus seri, dua baris berwaktu sama dapat bertukar tempat dan membuat paginasi
// melewatkan baris — pengguna tidak akan pernah melihat baris yang terlewat itu.
func TestListOrderStableBetweenTwoRequests(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()

	// Tiga laporan dicatat pada waktu yang SAMA PERSIS — jamnya tidak digeser. Inilah
	// keadaan yang membuat pengurutan tanpa pemutus seri menjadi tidak stabil.
	for i := 0; i < 3; i++ {
		_, err := service.Record(ctx, testReport(), testRecorder)
		require.NoError(t, err)
	}

	first, err := service.List(ctx, pelaporanklaim.Filter{})
	require.NoError(t, err)
	second, err := service.List(ctx, pelaporanklaim.Filter{})
	require.NoError(t, err)

	require.Len(t, first.Page.Reports, 3)
	for i := range first.Page.Reports {
		require.Equal(t, first.Page.Reports[i].Number, second.Page.Reports[i].Number,
			"urutan berubah di antara dua permintaan yang identik")
	}
}
