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

// waktuUji tetap, supaya `bulanlapor` dapat diperiksa dengan angka pasti.
var waktuUji = time.Date(2026, time.September, 26, 10, 0, 0, 0, time.UTC)

// senderPalsu menggantikan layanan SLIK yang kontraknya belum ada.
type senderPalsu struct {
	dikirim []monitoringslinkojk.Submission
	galat   error
}

func (s *senderPalsu) Send(_ context.Context, submission monitoringslinkojk.Submission) (
	monitoringslinkojk.SubmissionResult, error,
) {
	s.dikirim = append(s.dikirim, submission)
	if s.galat != nil {
		return monitoringslinkojk.SubmissionResult{}, s.galat
	}
	return monitoringslinkojk.SubmissionResult{
		ClientID:      "CLIENT-" + submission.ClaimID,
		TransactionID: "TRX-" + submission.ClaimID,
	}, nil
}

// newWriteService membentuk layanan beserta penyimpanannya, supaya uji dapat memeriksa
// apa yang BENAR-BENAR tersimpan — bukan hanya apa yang dilaporkan.
func newWriteService(t *testing.T, sender monitoringslinkojk.Sender) (
	*usecase.Service, *memory.Repo,
) {
	t.Helper()
	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (monitoringslinkojk.Repo, error) {
			if alias != portalUtama {
				return nil, errors.New("portal tidak siap")
			}
			return repo, nil
		},
		Sender: sender,
	})
	require.NoError(t, err)
	return service, repo
}

// ============================================================================
// PROSES DATA KLAIM
// ============================================================================

// Baris yang tersusun BENAR-BENAR tersimpan, dan `bulanlapor`-nya berbentuk YYYYMM.
func TestProcessStoresRowsWithReportMonth(t *testing.T) {
	service, repo := newWriteService(t, nil)

	outcome, err := service.Process(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}), waktuUji)
	require.NoError(t, err)
	require.Equal(t, 4, outcome.Total())

	stored := repo.Reports()
	require.Len(t, stored, 4)
	for _, entry := range stored {
		require.Equal(t, "202609", entry.ReportMonth,
			"bulanlapor berbentuk YYYYMM, sama dengan FormatDateTime(...,\"yyyyMM\")")
		require.Equal(t, "0", entry.Recovery,
			"jalur proses mengisi recovery nol, seperti InsertAdjustmentListKredit")
	}
}

// Penyusunan KEDUA menandai barisnya `'U'` — pencacah dibaca, bukan diabaikan.
//
// Tanpa ini, klaim yang sudah pernah dilaporkan masuk lagi sebagai klaim BARU, dan OJK
// menerima satu klaim terhitung dua kali tanpa satu pun tanda.
func TestProcessSecondRunMarksUpdate(t *testing.T) {
	service, repo := newWriteService(t, nil)
	ctx := context.Background()
	req := request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{})

	_, err := service.Process(ctx, req, waktuUji)
	require.NoError(t, err)

	outcome, err := service.Process(ctx, req, waktuUji)
	require.NoError(t, err)

	require.Zero(t, outcome.Created)
	require.Equal(t, 4, outcome.Updated)

	// Delapan baris tersimpan, bukan empat: penyusunan ulang MENAMBAH, tidak menimpa —
	// sehingga riwayat pelaporan satu klaim tetap utuh.
	require.Len(t, repo.Reports(), 8)
}

// Penyaring yang tidak menemukan apa pun dijawab galat, bukan "berhasil nol baris".
func TestProcessWithoutSourceReturnsError(t *testing.T) {
	service, _ := newWriteService(t, nil)

	// Rentang tanggal yang jauh di luar data contoh. Sebelumnya dipakai kotak pencarian;
	// kotak itu sudah dicabut supaya layarnya sama dengan Pega.
	dari := time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC)
	sampai := time.Date(1990, time.January, 2, 0, 0, 0, 0, time.UTC)

	_, err := service.Process(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{
			DateOfLoss:            &dari,
			DateOfRequestDocument: &sampai,
		}), waktuUji)

	require.ErrorIs(t, err, monitoringslinkojk.ErrNothingToProcess)
}

// ============================================================================
// UNGGAH
// ============================================================================

// Tunggakan DIKURANGI recovery — persis `.TUNGGAKAN - .RecoveryClaim` di Pega.
//
// Hanya jalur unggah yang melakukannya; jalur proses mengisi recovery nol.
func TestUploadSubtractsRecoveryFromArrears(t *testing.T) {
	service, repo := newWriteService(t, nil)

	outcome, err := service.Upload(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		[]usecase.UploadRow{{
			Line: 1,
			Entry: monitoringslinkojk.ReportEntry{
				ClaimID:    "PNCN.26.9001",
				ContractNo: "KTR-9001",
				Arrears:    "5000",
				Recovery:   "1200",
			},
		}}, waktuUji)

	require.NoError(t, err)
	require.Equal(t, 1, outcome.Total())

	stored := repo.Reports()
	require.Len(t, stored, 1)
	require.Equal(t, "3800", stored[0].Arrears)
}

// Nilai yang tidak terbaca sebagai angka dikembalikan APA ADANYA, bukan menjadi nol.
//
// Nol adalah angka yang tampak sah pada laporan regulator, dan tunggakan yang diam-diam
// menjadi nol tidak akan ditanyakan siapa pun.
func TestUploadKeepsUnparsableArrears(t *testing.T) {
	service, repo := newWriteService(t, nil)

	_, err := service.Upload(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		[]usecase.UploadRow{{
			Line: 1,
			Entry: monitoringslinkojk.ReportEntry{
				ClaimID:    "PNCN.26.9002",
				ContractNo: "KTR-9002",
				Arrears:    "5.000.000",
				Recovery:   "1000",
			},
		}}, waktuUji)

	require.NoError(t, err)
	require.Equal(t, "5.000.000", repo.Reports()[0].Arrears)
}

// Baris cacat ditolak SENDIRIAN; sisanya tetap tersusun.
// Baris tanpa No Klaim TETAP ditulis — Pega tidak memeriksa apa pun.
//
// Uji ini semula menuntut kebalikannya: baris cacat ditolak sendirian dan dilaporkan
// beserta nomor barisnya. Penyaringan itu penambahan di luar Pega, dan dicabut atas
// permintaan Work Owner (2026-09-27). `PNCUploadAutoClaimSlikOJK` menyisipkan setiap
// baris berkas apa adanya.
//
// Yang dijaga uji ini karena itu bukan penyaringan, melainkan KETIADAANNYA — supaya
// penyaringan tidak kembali masuk diam-diam.
func TestUploadWritesEveryRowIncludingIncomplete(t *testing.T) {
	service, repo := newWriteService(t, nil)

	outcome, err := service.Upload(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		[]usecase.UploadRow{
			{Line: 1, Entry: monitoringslinkojk.ReportEntry{
				ClaimID: "PNCN.26.9001", ContractNo: "KTR-9001"}},
			{Line: 2, Entry: monitoringslinkojk.ReportEntry{
				ContractNo: "KTR-9002"}}, // No Klaim kosong
			{Line: 3, Entry: monitoringslinkojk.ReportEntry{
				ClaimID: "PNCN.26.9003", ContractNo: "KTR-9003"}},
		}, waktuUji)

	require.NoError(t, err)
	require.Equal(t, 3, outcome.Total())
	require.Empty(t, outcome.Skipped)
	require.Len(t, repo.Reports(), 3)
}

func TestUploadWithoutRowsIsRejected(t *testing.T) {
	service, _ := newWriteService(t, nil)

	_, err := service.Upload(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}), nil, waktuUji)

	var validation *monitoringslinkojk.ValidationError
	require.ErrorAs(t, err, &validation)
}

// ============================================================================
// PENGIRIMAN KE SLIK
// ============================================================================

// Tanpa seam pengirim, TIDAK ada satu baris pengiriman pun yang tercatat.
//
// Ini uji terpenting di antara aksi kirim: tanpa pemeriksaan di muka, setiap penekanan
// tombol meninggalkan baris yang tidak akan pernah terkirim — dan tabelnya terisi jejak
// palsu yang tidak dapat dibedakan dari pengiriman yang gagal.
func TestSubmitWithoutSenderRecordsNothing(t *testing.T) {
	service, repo := newWriteService(t, nil)

	_, _, err := service.Submit(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		"PNCN.26.0101", "KTR-2026-0101")

	require.ErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured)
	require.Empty(t, repo.Submissions())
}

// Pengiriman yang berhasil menyimpan client ID dan id transaksinya.
func TestSubmitStoresTransactionID(t *testing.T) {
	sender := &senderPalsu{}
	service, repo := newWriteService(t, sender)

	submission, result, err := service.Submit(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		"PNCN.26.0101", "KTR-2026-0101")

	require.NoError(t, err)
	require.Equal(t, int64(1), submission.ID, "nvl(max(id),0)+1 pada tabel kosong")
	require.Equal(t, "TRX-PNCN.26.0101", result.TransactionID)

	stored := repo.Submissions()
	require.Len(t, stored, 1)
	require.True(t, stored[0].Completed)
	require.Equal(t, "CLIENT-PNCN.26.0101", stored[0].Result.ClientID)
}

// Pengiriman yang GAGAL tetap meninggalkan barisnya, tanpa id transaksi.
//
// Baris tanpa id transaksi adalah pengiriman yang tidak pernah sampai — dan tanpa baris
// itu, pengiriman yang gagal tidak dapat dibedakan dari yang tidak pernah dicoba.
func TestSubmitFailureKeepsRecordWithoutTransaction(t *testing.T) {
	sender := &senderPalsu{galat: errors.New("layanan SLIK menolak")}
	service, repo := newWriteService(t, sender)

	_, _, err := service.Submit(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}),
		"PNCN.26.0101", "KTR-2026-0101")

	require.Error(t, err)

	stored := repo.Submissions()
	require.Len(t, stored, 1, "barisnya TIDAK dihapus — ia jejak bahwa pengiriman dicoba")
	require.False(t, stored[0].Completed)
	require.Empty(t, stored[0].Result.TransactionID)
}

// SubmitFiltered mengirim seluruh klaim yang cocok, satu pengiriman per baris.
func TestSubmitFilteredSendsEveryMatchingRow(t *testing.T) {
	sender := &senderPalsu{}
	service, repo := newWriteService(t, sender)

	outcome, err := service.SubmitFiltered(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))

	require.NoError(t, err)
	require.Equal(t, 4, outcome.Created)
	require.Len(t, sender.dikirim, 4)
	require.Len(t, repo.Submissions(), 4)

	// Nomor urutnya BERBEDA untuk setiap pengiriman; `nvl(max(id),0)+1` dibaca ulang
	// setiap kali, bukan dihitung sekali di muka.
	nomor := map[int64]bool{}
	for _, submission := range repo.Submissions() {
		require.Falsef(t, nomor[submission.ID], "nomor pengiriman ganda: %d", submission.ID)
		nomor[submission.ID] = true
	}
}

// Satu pengiriman yang gagal TIDAK menghentikan sisanya.
func TestSubmitFilteredContinuesAfterFailure(t *testing.T) {
	sender := &senderPalsu{galat: errors.New("layanan SLIK menolak")}
	service, _ := newWriteService(t, sender)

	outcome, err := service.SubmitFiltered(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))

	require.NoError(t, err)
	require.Zero(t, outcome.Created)
	require.Len(t, outcome.Skipped, 4, "keempatnya dilaporkan, bukan berhenti di yang pertama")
	require.Len(t, sender.dikirim, 4)
}

// Tanpa seam pengirim, SubmitFiltered pun tidak membaca satu baris pun.
func TestSubmitFilteredWithoutSenderReadsNothing(t *testing.T) {
	service, repo := newWriteService(t, nil)

	_, err := service.SubmitFiltered(context.Background(),
		request(monitoringslinkojk.SegmentD01, monitoringslinkojk.Filter{}))

	require.ErrorIs(t, err, monitoringslinkojk.ErrSenderNotConfigured)
	require.Empty(t, repo.Submissions())
}
