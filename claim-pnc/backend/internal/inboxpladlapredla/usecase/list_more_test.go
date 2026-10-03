package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/usecase"
)

// repoPalsu mengembalikan nilai yang sudah ditentukan untuk setiap operasi seam.
type repoPalsu struct {
	listErr        error
	documentsErr   error
	printErr       error
	markPreChanged bool
	markPreErr     error
	advice         inboxpladlapredla.SendableAdvice
	claim          inboxpladlapredla.ClaimSummary
	adviceErr      error
	attachments    []inboxpladlapredla.Attachment
	attachmentsErr error
	markChanged    bool
	markErr        error
}

func (r *repoPalsu) List(
	context.Context, inboxpladlapredla.Query, inboxpladlapredla.Pagination,
) (inboxpladlapredla.Page, error) {
	return inboxpladlapredla.Page{}, r.listErr
}

func (r *repoPalsu) Documents(
	context.Context, inboxpladlapredla.Tab, string,
) ([]inboxpladlapredla.Document, error) {
	return nil, r.documentsErr
}

func (r *repoPalsu) AdviceForSending(
	context.Context, inboxpladlapredla.Tab, string, string,
) (inboxpladlapredla.SendableAdvice, inboxpladlapredla.ClaimSummary, error) {
	return r.advice, r.claim, r.adviceErr
}

func (r *repoPalsu) AttachmentsForClaim(
	context.Context, string, string,
) ([]inboxpladlapredla.Attachment, error) {
	return r.attachments, r.attachmentsErr
}

func (r *repoPalsu) MarkAdviceSent(
	context.Context, inboxpladlapredla.Tab, string, string, inboxpladlapredla.SendableAdvice,
) (bool, error) {
	return r.markChanged, r.markErr
}

func (r *repoPalsu) MarkPreDLASent(context.Context, string, string) (bool, error) {
	return r.markPreChanged, r.markPreErr
}

func (r *repoPalsu) PrintPreDLA(
	context.Context, string,
) ([]inboxpladlapredla.PreDLADocument, error) {
	return nil, r.printErr
}

var errSelector = errors.New("portal belum siap")

// layananPalsu membentuk layanan di atas repo palsu, dengan log ditulis ke penyangga.
func layananPalsu(
	t *testing.T,
	repo inboxpladlapredla.Repo,
	notifier inboxpladlapredla.Notifier,
	composeBody usecase.BodyComposer,
) (*usecase.Service, *bytes.Buffer) {
	t.Helper()

	log := &bytes.Buffer{}
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxpladlapredla.Repo, error) {
			if alias != "ASM" {
				return nil, errSelector
			}
			return repo, nil
		},
		Logger:      slog.New(slog.NewTextHandler(log, nil)),
		Notifier:    notifier,
		ComposeBody: composeBody,
	})
	require.NoError(t, err)
	return svc, log
}

// layananContoh membentuk layanan ber-log di atas penyimpanan memori contoh.
func layananContoh(
	t *testing.T,
	notifier inboxpladlapredla.Notifier,
) (*usecase.Service, *bytes.Buffer) {
	t.Helper()
	store := contoh(t)
	log := &bytes.Buffer{}
	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(alias string) (inboxpladlapredla.Repo, error) {
			if alias != "ASM" {
				return nil, errSelector
			}
			return store, nil
		},
		Logger:   slog.New(slog.NewTextHandler(log, nil)),
		Notifier: notifier,
	})
	require.NoError(t, err)
	return svc, log
}

func TestNewServiceRequiresARepoSelector(t *testing.T) {
	svc, err := usecase.NewService(usecase.Options{})
	require.Error(t, err)
	require.Nil(t, svc)
}

// Metadata menyerahkan tiga tab, tab bawaan, dan selisih terencana.
func TestMetadataDescribesTheScreen(t *testing.T) {
	svc, _ := layananContoh(t, nil)

	meta := svc.Metadata()
	require.Len(t, meta.Tabs, 3)
	require.Equal(t, inboxpladlapredla.DefaultTab, meta.DefaultTab)
	require.Equal(t, inboxpladlapredla.PlannedDifferences, meta.PlannedDifferences)
}

// Pembukaan antrean dicatat, tanpa kata kunci pencariannya.
func TestListLogsTheOpeningWithoutTheKeyword(t *testing.T) {
	svc, log := layananContoh(t, nil)

	listed, err := svc.List(context.Background(), "ASM", pemanggil(),
		inboxpladlapredla.QueryInput{Tab: "pla", Search: "PNC-1001"},
		inboxpladlapredla.Pagination{})
	require.NoError(t, err)
	require.Equal(t, "pla", listed.Query.Tab.Code)
	require.Equal(t, 1, listed.Page.Total)
	require.Equal(t, kunciKlaim, listed.Page.Items[0].ClaimKey)

	require.Contains(t, log.String(), "antrean PLA/DLA/Pre DLA dibuka")
	require.Contains(t, log.String(), "mencari=true")
	require.NotContains(t, log.String(), "PNC-1001")
}

// Galat dari permintaan, pemilih portal, dan penyimpanan diteruskan.
func TestListPropagatesErrors(t *testing.T) {
	ctx := context.Background()
	repo := &repoPalsu{listErr: errors.New("oracle mati")}
	svc, _ := layananPalsu(t, repo, nil, nil)

	_, err := svc.List(ctx, "ASM", inboxpladlapredla.Caller{},
		inboxpladlapredla.QueryInput{}, inboxpladlapredla.Pagination{})
	require.ErrorIs(t, err, inboxpladlapredla.ErrCallerUnknown)

	_, err = svc.List(ctx, "ASI", pemanggil(),
		inboxpladlapredla.QueryInput{}, inboxpladlapredla.Pagination{})
	require.ErrorIs(t, err, errSelector)

	_, err = svc.List(ctx, "ASM", pemanggil(),
		inboxpladlapredla.QueryInput{Tab: "dla"}, inboxpladlapredla.Pagination{})
	require.ErrorIs(t, err, repo.listErr)
	require.Contains(t, err.Error(), "mengambil isi antrean dla")
}

// Rincian dibuka dan dicatat tanpa kunci klaim.
func TestDocumentsReturnsTheGridAndLogsIt(t *testing.T) {
	svc, log := layananContoh(t, nil)

	documented, err := svc.Documents(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim)
	require.NoError(t, err)
	require.Equal(t, "pla", documented.Tab.Code)
	require.Equal(t, kunciKlaim, documented.ClaimKey)
	require.Len(t, documented.Items, 2)

	require.Contains(t, log.String(), "rincian PLA/DLA dibuka")
	require.Contains(t, log.String(), "dokumen=2")
	require.NotContains(t, log.String(), "PNC-1001")
}

func TestDocumentsRejectsAndPropagates(t *testing.T) {
	ctx := context.Background()

	svc, _ := layananPalsu(t, &repoPalsu{}, nil, nil)

	_, err := svc.Documents(ctx, "ASM", inboxpladlapredla.Caller{Login: " "}, "pla", "k")
	require.ErrorIs(t, err, inboxpladlapredla.ErrCallerUnknown)

	_, err = svc.Documents(ctx, "ASM", pemanggil(), "entah", "k")
	var validation *inboxpladlapredla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxpladlapredla.FieldTab, validation.Violations[0].Field)

	_, err = svc.Documents(ctx, "ASM", pemanggil(), "pre-dla", "k")
	require.ErrorIs(t, err, inboxpladlapredla.ErrDocumentsNotOnTab)

	_, err = svc.Documents(ctx, "ASI", pemanggil(), "pla", "k")
	require.ErrorIs(t, err, errSelector)

	notFound, _ := layananPalsu(t,
		&repoPalsu{documentsErr: inboxpladlapredla.ErrRowNotFound}, nil, nil)
	_, err = notFound.Documents(ctx, "ASM", pemanggil(), "dla", "k")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	broken, _ := layananPalsu(t, &repoPalsu{documentsErr: errors.New("putus")}, nil, nil)
	_, err = broken.Documents(ctx, "ASM", pemanggil(), "dla", "k")
	require.EqualError(t, err, "mengambil rincian dla: putus")
}

// Panel Print Pre DLA dibuka dan dicatat.
func TestPrintListReturnsThePanelAndLogsIt(t *testing.T) {
	svc, log := layananContoh(t, nil)

	printable, err := svc.PrintList(context.Background(), "ASM", pemanggil(), kunciKlaim)
	require.NoError(t, err)
	require.Equal(t, "pre-dla", printable.Tab.Code)
	require.Equal(t, kunciKlaim, printable.ClaimKey)
	require.Len(t, printable.Items, 1)
	require.Equal(t, "PRE/2026/0001", printable.Items[0].AdviceNo)

	require.Contains(t, log.String(), "panel Print Pre DLA dibuka")
	require.NotContains(t, log.String(), "PNC-1001")
}

func TestPrintListRejectsAndPropagates(t *testing.T) {
	ctx := context.Background()
	svc, _ := layananPalsu(t, &repoPalsu{}, nil, nil)

	_, err := svc.PrintList(ctx, "ASM", inboxpladlapredla.Caller{}, "k")
	require.ErrorIs(t, err, inboxpladlapredla.ErrCallerUnknown)

	_, err = svc.PrintList(ctx, "ASI", pemanggil(), "k")
	require.ErrorIs(t, err, errSelector)

	notFound, _ := layananPalsu(t,
		&repoPalsu{printErr: inboxpladlapredla.ErrRowNotFound}, nil, nil)
	_, err = notFound.PrintList(ctx, "ASM", pemanggil(), "k")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	broken, _ := layananPalsu(t, &repoPalsu{printErr: errors.New("putus")}, nil, nil)
	_, err = broken.PrintList(ctx, "ASM", pemanggil(), "k")
	require.EqualError(t, err, "mengambil panel Print Pre DLA: putus")
}

// Penandaan Pre-DLA berhasil dicatat beserta nomornya.
func TestSendPreDLAMarksAndLogsTheNumber(t *testing.T) {
	svc, log := layananContoh(t, nil)

	err := svc.SendPreDLA(context.Background(), "ASM", pemanggil(), kunciKlaim,
		" PRE/2026/0001 ")
	require.NoError(t, err)
	require.Contains(t, log.String(), "Pre-DLA ditandai terkirim")
	require.Contains(t, log.String(), "no_pre_dla=PRE/2026/0001")

	// Penekanan kedua menyatakan sudah terkirim.
	err = svc.SendPreDLA(context.Background(), "ASM", pemanggil(), kunciKlaim,
		"PRE/2026/0001")
	require.ErrorIs(t, err, inboxpladlapredla.ErrPreDLAAlreadySent)
}

func TestSendPreDLARejectsAndPropagates(t *testing.T) {
	ctx := context.Background()
	svc, _ := layananPalsu(t, &repoPalsu{}, nil, nil)

	require.ErrorIs(t, svc.SendPreDLA(ctx, "ASM", inboxpladlapredla.Caller{}, "k", "n"),
		inboxpladlapredla.ErrCallerUnknown)

	err := svc.SendPreDLA(ctx, "ASM", pemanggil(), "k", "  ")
	var validation *inboxpladlapredla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxpladlapredla.FieldAdviceNo, validation.Violations[0].Field)

	require.ErrorIs(t, svc.SendPreDLA(ctx, "ASI", pemanggil(), "k", "n"), errSelector)

	notFound, _ := layananPalsu(t,
		&repoPalsu{markPreErr: inboxpladlapredla.ErrRowNotFound}, nil, nil)
	require.ErrorIs(t, notFound.SendPreDLA(ctx, "ASM", pemanggil(), "k", "n"),
		inboxpladlapredla.ErrRowNotFound)

	broken, _ := layananPalsu(t, &repoPalsu{markPreErr: errors.New("putus")}, nil, nil)
	require.EqualError(t, broken.SendPreDLA(ctx, "ASM", pemanggil(), "k", "n"),
		"menandai Pre-DLA terkirim: putus")
}

func sendableFake() *repoPalsu {
	return &repoPalsu{
		advice: inboxpladlapredla.SendableAdvice{
			AdviceNo: "PLA/1", Email: "a@contoh.example", Reinsurer: "Reas",
		},
		claim:       inboxpladlapredla.ClaimSummary{ClaimNo: "PNC-1"},
		markChanged: true,
	}
}

// Isian yang salah ditolak sebelum penyimpanan disentuh.
func TestSendAdviceRejectsBadInput(t *testing.T) {
	ctx := context.Background()
	notifier := &notifierPalsu{}
	svc, _ := layananPalsu(t, sendableFake(), notifier, nil)

	_, err := svc.SendAdvice(ctx, "ASM", inboxpladlapredla.Caller{}, "pla", "k", "n")
	require.ErrorIs(t, err, inboxpladlapredla.ErrCallerUnknown)

	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "entah", "k", "n")
	var validation *inboxpladlapredla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxpladlapredla.FieldTab, validation.Violations[0].Field)

	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", " ")
	require.ErrorAs(t, err, &validation)
	require.Equal(t, inboxpladlapredla.FieldAdviceNo, validation.Violations[0].Field)

	_, err = svc.SendAdvice(ctx, "ASI", pemanggil(), "pla", "k", "n")
	require.ErrorIs(t, err, errSelector)

	require.Empty(t, notifier.surat)
}

// Galat pembacaan dokumen dan lampiran menghentikan pengiriman sebelum surat dikirim.
func TestSendAdviceStopsBeforeTheLetterOnReadErrors(t *testing.T) {
	ctx := context.Background()

	notFound := sendableFake()
	notFound.adviceErr = inboxpladlapredla.ErrRowNotFound
	notifier := &notifierPalsu{}
	svc, _ := layananPalsu(t, notFound, notifier, nil)
	_, err := svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	require.ErrorIs(t, err, inboxpladlapredla.ErrRowNotFound)

	broken := sendableFake()
	broken.adviceErr = errors.New("putus")
	svc, _ = layananPalsu(t, broken, notifier, nil)
	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	require.EqualError(t, err, "membaca dokumen untuk dikirim: putus")

	empty := sendableFake()
	empty.advice.Email = "bukan alamat"
	svc, _ = layananPalsu(t, empty, notifier, nil)
	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	var validation *inboxpladlapredla.ValidationError
	require.ErrorAs(t, err, &validation)
	require.Contains(t, validation.Violations[0].Message, "tidak dapat dibaca")

	noAttachments := sendableFake()
	noAttachments.attachmentsErr = errors.New("lampiran rusak")
	svc, _ = layananPalsu(t, noAttachments, notifier, nil)
	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	require.EqualError(t, err, "mengambil lampiran: lampiran rusak")

	require.Empty(t, notifier.surat)
}

// Tanpa penyusun badan, suratnya tetap dikirim dengan badan kosong.
func TestSendAdviceWithoutABodyComposerSendsAnEmptyBody(t *testing.T) {
	notifier := &notifierPalsu{}
	repo := sendableFake()
	repo.attachments = []inboxpladlapredla.Attachment{{Name: "a.pdf", Content: []byte("x")}}
	svc, log := layananPalsu(t, repo, notifier, nil)

	hasil, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "dla", "k", " PLA/1 ")
	require.NoError(t, err)
	require.Equal(t, "PLA/1", hasil.AdviceNo)
	require.Equal(t, "dla", hasil.Tab.Code)
	require.Equal(t, []string{"a@contoh.example"}, hasil.Recipients)
	require.Equal(t, 1, hasil.Attachments)

	require.Len(t, notifier.surat, 1)
	require.Empty(t, notifier.surat[0].HTMLBody)
	require.Contains(t, notifier.surat[0].Subject, "DLA Supporting Document Claim")
	require.Contains(t, log.String(), "surat PLA/DLA terkirim")
}

// Pengirim yang belum siap diteruskan apa adanya, bukan dibungkus "surat tidak terkirim".
func TestSendAdviceKeepsNotifierUnavailableAsIs(t *testing.T) {
	notifier := &notifierPalsu{
		galat: errors.Join(inboxpladlapredla.ErrNotifierUnavailable, errors.New("host"))}
	svc, _ := layananPalsu(t, sendableFake(), notifier, nil)

	_, err := svc.SendAdvice(context.Background(), "ASM", pemanggil(), "pla", "k", "n")
	require.ErrorIs(t, err, inboxpladlapredla.ErrNotifierUnavailable)
	require.NotErrorIs(t, err, inboxpladlapredla.ErrLetterNotSent)
}

// Penandaan yang gagal SETELAH surat terkirim menyebut bahwa suratnya sudah terkirim.
func TestSendAdviceReportsALetterSentButNotMarked(t *testing.T) {
	ctx := context.Background()

	failing := sendableFake()
	failing.markErr = errors.New("commit gagal")
	notifier := &notifierPalsu{}
	svc, _ := layananPalsu(t, failing, notifier, nil)

	hasil, err := svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	require.ErrorIs(t, err, inboxpladlapredla.ErrSentButNotMarked)
	require.Contains(t, err.Error(), "SUDAH terkirim ke 1 penerima")
	require.Contains(t, err.Error(), "commit gagal")
	require.Equal(t, 0, hasil.Attachments)
	require.Equal(t, []string{"a@contoh.example"}, hasil.Recipients)

	raced := sendableFake()
	raced.markChanged = false
	svc, _ = layananPalsu(t, raced, notifier, nil)
	_, err = svc.SendAdvice(ctx, "ASM", pemanggil(), "pla", "k", "n")
	require.ErrorIs(t, err, inboxpladlapredla.ErrSentButNotMarked)
	require.Contains(t, err.Error(), "sudah ditandai terkirim oleh pihak lain")

	require.Len(t, notifier.surat, 2)
}
