package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxmanager"
	"claim-pnc/internal/inboxmanager/repo/memory"
	"claim-pnc/internal/inboxmanager/usecase"
	"claim-pnc/internal/platform/clock"
)

type fakeLinker struct {
	url   string
	err   error
	calls []string
}

func (f *fakeLinker) Link(_ context.Context, portal, imageID, by string) (string, error) {
	f.calls = append(f.calls, portal+"|"+imageID+"|"+by)
	return f.url, f.err
}

type fakeNotifier struct {
	err     error
	notices []inboxmanager.PaymentNotice
}

func (f *fakeNotifier) NotifyPaymentChecked(_ context.Context, n inboxmanager.PaymentNotice) error {
	f.notices = append(f.notices, n)
	return f.err
}

func paymentStore() *memory.Store {
	store := memory.NewStrictStore()
	store.SetQueue(inboxmanager.TabPaymentAkseptasi, []inboxmanager.QueueRow{{Key: "AKS-1"}})
	store.SetPaymentDetail(inboxmanager.PaymentDetail{
		AcceptedNo: "AKS-1", ClaimID: "CLAIM-1", ClaimNo: "C-1", InsuredName: "Tertanggung",
		PaymentTypeCode: "2", PICEmail: "pic@contoh.test", AdjustmentFound: true,
		Documents: []inboxmanager.PaymentDocument{
			{Key: "1", ImageID: "IMG-1"},
			{Key: "2", HasFile: true},
		},
	})
	return store
}

func newPaymentService(
	t *testing.T, store *memory.Store, linker inboxmanager.DocumentLinker, notifier inboxmanager.PaymentNotifier,
) *usecase.Service {
	t.Helper()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector:         func(string) (inboxmanager.Repo, error) { return store, nil },
		LineBusinessSelector: func(string) (inboxmanager.LineBusinessRepo, error) { return store, nil },
		Clock:                clock.FixedAt(time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)),
		PaymentRules: inboxmanager.PaymentRules{PaymentTypeName: func(code string) string {
			if code == "2" {
				return "Interim"
			}
			return ""
		}},
		DocumentLinker:  linker,
		PaymentNotifier: notifier,
	})
	require.NoError(t, err)
	return service
}

var mgr = inboxmanager.Caller{Login: "MGR"}

func TestPaymentDetailMenamaiTipePembayaran(t *testing.T) {
	service := newPaymentService(t, paymentStore(), nil, nil)

	detail, err := service.PaymentDetail(context.Background(), portalUtama, "AKS-1", mgr)
	require.NoError(t, err)
	require.Equal(t, "Interim", detail.PaymentType)

	_, err = service.PaymentDetail(context.Background(), portalUtama, "AKS-1", inboxmanager.Caller{})
	require.ErrorIs(t, err, inboxmanager.ErrCallerUnknown)

	_, err = service.PaymentDetail(context.Background(), portalUtama, "AKS-X", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrPaymentNotFound)
}

func TestPaymentDocumentLink(t *testing.T) {
	ctx := context.Background()
	linker := &fakeLinker{url: "https://penyimpanan.contoh.test/berkas"}
	service := newPaymentService(t, paymentStore(), linker, nil)

	url, err := service.PaymentDocumentLink(ctx, portalUtama, "AKS-1", "1", mgr)
	require.NoError(t, err)
	require.Equal(t, linker.url, url)
	require.Equal(t, []string{"ASM|IMG-1|MGR"}, linker.calls)

	// Lampiran lama tidak punya tautan; kunci di luar daftar tidak menunjuk apa pun.
	_, err = service.PaymentDocumentLink(ctx, portalUtama, "AKS-1", "2", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)
	_, err = service.PaymentDocumentLink(ctx, portalUtama, "AKS-1", "9", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)

	// Dokumen yang tidak tercatat di layanan penyimpanan.
	linker.url = ""
	_, err = service.PaymentDocumentLink(ctx, portalUtama, "AKS-1", "1", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)

	// Tanpa linker.
	service = newPaymentService(t, paymentStore(), nil, nil)
	_, err = service.PaymentDocumentLink(ctx, portalUtama, "AKS-1", "1", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrDocumentLinkUnavailable)
}

func TestPaymentDocument(t *testing.T) {
	store := paymentStore()
	store.SetPaymentDocumentContent("AKS-1", "2", inboxmanager.PaymentDocumentContent{
		Name: "lama.pdf", MimeType: "application/pdf", Content: []byte("%PDF"),
	})
	service := newPaymentService(t, store, nil, nil)

	content, err := service.PaymentDocument(context.Background(), portalUtama, "AKS-1", "2", mgr)
	require.NoError(t, err)
	require.Equal(t, []byte("%PDF"), content.Content)

	_, err = service.PaymentDocument(context.Background(), portalUtama, "AKS-1", "1", mgr)
	require.ErrorIs(t, err, inboxmanager.ErrPaymentDocumentNotFound)
}

func TestSavePaymentMemberiTahuPIC(t *testing.T) {
	store := paymentStore()
	notifier := &fakeNotifier{}
	service := newPaymentService(t, store, nil, notifier)

	result, err := service.SavePayment(context.Background(), portalUtama, usecase.SavePaymentInput{
		AcceptedNo: "AKS-1", Status: "2", Remark: "nilai belum sesuai",
	}, mgr)
	require.NoError(t, err)
	require.Equal(t, inboxmanager.PaymentSaveResult{Changed: 1, AdjustmentMarked: true, Notified: true}, result)

	require.Equal(t, []inboxmanager.PaymentNotice{{
		To: "pic@contoh.test", ClaimNo: "C-1", InsuredName: "Tertanggung", AcceptedNo: "AKS-1",
		Status: "2", Remark: "nilai belum sesuai", DecidedBy: "MGR",
	}}, notifier.notices)

	// Nomornya keluar dari antrean; SIMPAN kedua tidak mengubah apa pun dan tidak bersurel.
	result, err = service.SavePayment(context.Background(), portalUtama, usecase.SavePaymentInput{
		AcceptedNo: "AKS-1", Status: "2", Remark: "lagi",
	}, mgr)
	require.NoError(t, err)
	require.Zero(t, result.Changed)
	require.Len(t, notifier.notices, 1)
}

// Surel yang gagal tidak membatalkan keputusan yang sudah tertulis.
func TestSavePaymentSurelGagalTetapTersimpan(t *testing.T) {
	store := paymentStore()
	service := newPaymentService(t, store, nil, &fakeNotifier{err: errors.New("SMTP mati")})

	result, err := service.SavePayment(context.Background(), portalUtama, usecase.SavePaymentInput{
		AcceptedNo: "AKS-1", Status: "0",
	}, mgr)
	require.NoError(t, err)
	require.Equal(t, 1, result.Changed)
	require.False(t, result.Notified)
	require.Len(t, store.PaymentSaves(), 1)

	// Tanpa pengirim surel pun sama.
	service = newPaymentService(t, store, nil, nil)
	result, err = service.SavePayment(context.Background(), portalUtama, usecase.SavePaymentInput{
		AcceptedNo: "AKS-1", Status: "0",
	}, mgr)
	require.NoError(t, err)
	require.Equal(t, 1, result.Changed, "Sedang Proses Komite membiarkan barisnya menunggu")
	require.False(t, result.Notified)
}

func TestSavePaymentSetujuDitahan(t *testing.T) {
	store := paymentStore()
	service := newPaymentService(t, store, nil, &fakeNotifier{})

	_, err := service.SavePayment(context.Background(), portalUtama, usecase.SavePaymentInput{
		AcceptedNo: "AKS-1", Status: "1",
	}, mgr)
	require.ErrorIs(t, err, inboxmanager.ErrApproveBlocked)
	require.Empty(t, store.PaymentSaves())
}
