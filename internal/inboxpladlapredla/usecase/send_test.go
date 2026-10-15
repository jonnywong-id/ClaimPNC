package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxpladlapredla"
	"claim-pnc/internal/inboxpladlapredla/repo/memory"
	"claim-pnc/internal/inboxpladlapredla/usecase"
)

// Uji di berkas ini memeriksa satu hal yang tidak dapat dilihat dari membaca kode:
// **urutan** antara mengirim surat dan menandai dokumennya.
//
// Urutan itu satu-satunya pagar terhadap kelas kegagalan terburuk modul ini — baris yang
// hilang dari antrean padahal tidak satu pun surat sampai ke reasuradur. Kegagalan itu
// tidak menghasilkan galat, tidak meninggalkan jejak, dan baru diketahui ketika
// reasuradur menanyakannya.

const kunciKlaim = "ASM-FW-GCNMFW-WORK PNC-1001"

// notifierPalsu merekam surat tanpa mengirimnya.
type notifierPalsu struct {
	surat []inboxpladlapredla.Letter
	galat error
}

func (n *notifierPalsu) SendAdvice(
	_ context.Context,
	letter inboxpladlapredla.Letter,
) error {
	if n.galat != nil {
		return n.galat
	}
	n.surat = append(n.surat, letter)
	return nil
}

func layanan(
	t *testing.T,
	store *memory.Store,
	notifier inboxpladlapredla.Notifier,
) *usecase.Service {
	t.Helper()

	svc, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxpladlapredla.Repo, error) { return store, nil },
		Notifier:     notifier,
		ComposeBody: func(
			_ inboxpladlapredla.Tab,
			_ inboxpladlapredla.SendableAdvice,
			_ inboxpladlapredla.ClaimSummary,
			_ int,
		) string {
			return "<p>badan</p>"
		},
	})
	require.NoError(t, err)
	return svc
}

func contoh(t *testing.T) *memory.Store {
	t.Helper()

	store := memory.NewSampleStore()
	store.Now = func() time.Time { return time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC) }
	store.SeedAttachments([]memory.Attachment{
		{
			ClaimKey: kunciKlaim, Category: "PLA",
			Name: "dokumen-pendukung.pdf", MIMEType: "application/pdf",
			Content: []byte("isi berkas"),
		},
		{
			// Lampiran KOSONG tidak ikut — berkas kosong yang sampai ke reasuradur
			// tampak seperti dokumen rusak.
			ClaimKey: kunciKlaim, Category: "PLA",
			Name: "kosong.pdf", Content: nil,
		},
		{
			// Kategori lain tidak ikut pada surat PLA.
			ClaimKey: kunciKlaim, Category: "DLA",
			Name: "punya-dla.pdf", Content: []byte("x"),
		},
	})
	return store
}

func pemanggil() inboxpladlapredla.Caller {
	return inboxpladlapredla.Caller{Login: "penguji"}
}

func nomorPLA(t *testing.T, store *memory.Store) string {
	t.Helper()

	tab, ok := inboxpladlapredla.FindTab("pla")
	require.True(t, ok)

	docs, err := store.Documents(context.Background(), tab, kunciKlaim)
	require.NoError(t, err)
	require.NotEmpty(t, docs)

	for _, d := range docs {
		if d.Sent != "1" {
			return d.AdviceNo
		}
	}
	t.Fatal("tidak ada PLA yang belum terkirim di data contoh")
	return ""
}

// Surat terkirim LEBIH DULU, dan barulah dokumennya ditandai.
func TestTheLetterGoesOutBeforeTheDocumentIsMarked(t *testing.T) {
	store := contoh(t)
	notifier := &notifierPalsu{}
	svc := layanan(t, store, notifier)

	no := nomorPLA(t, store)

	hasil, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim, no)
	require.NoError(t, err)

	require.Len(t, notifier.surat, 1, "surat tidak terkirim")
	require.Equal(t, 1, hasil.Attachments,
		"lampiran kosong dan lampiran kategori lain seharusnya tidak ikut")
	require.NotEmpty(t, hasil.Recipients)

	// Suratnya membawa subjek dan lampirannya.
	require.Contains(t, notifier.surat[0].Subject, "PLA Supporting Document Claim")
	require.Equal(t, "dokumen-pendukung.pdf", notifier.surat[0].Attachments[0].Name)

	// Dan dokumennya benar-benar ditandai.
	tab, _ := inboxpladlapredla.FindTab("pla")
	docs, err := store.Documents(context.Background(), tab, kunciKlaim)
	require.NoError(t, err)

	for _, d := range docs {
		if d.AdviceNo == no {
			require.Equal(t, "1", d.Sent)
			require.Equal(t, "2026-03-10", d.SentDate)
			return
		}
	}
	t.Fatal("dokumennya hilang dari daftar")
}

// **Surat yang GAGAL tidak menandai apa pun.**
//
// Ini uji terpenting di modul ini. Bila ia gagal, satu baris dapat hilang dari antrean
// tanpa satu pun surat sampai ke reasuradur — dan tidak ada galat, tidak ada tanda, dan
// tidak ada yang mengetahuinya sampai reasuradur menanyakannya.
func TestAFailedLetterMarksNothing(t *testing.T) {
	store := contoh(t)
	notifier := &notifierPalsu{galat: errors.New("relay menolak")}
	svc := layanan(t, store, notifier)

	no := nomorPLA(t, store)

	_, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim, no)
	require.ErrorIs(t, err, inboxpladlapredla.ErrLetterNotSent)

	tab, _ := inboxpladlapredla.FindTab("pla")
	docs, err := store.Documents(context.Background(), tab, kunciKlaim)
	require.NoError(t, err)

	for _, d := range docs {
		if d.AdviceNo == no {
			require.NotEqual(t, "1", d.Sent,
				"dokumen DITANDAI terkirim padahal suratnya gagal")
			require.Empty(t, d.SentDate)
			return
		}
	}
	t.Fatal("dokumennya hilang dari daftar")
}

// Tanpa pengirim surat, TIDAK ada yang dikirim dan tidak ada yang ditandai.
//
// Ia berbeda dari surat yang dicoba lalu gagal, dan galatnya pun berbeda: yang ini tidak
// dapat ditolong dengan mencoba lagi.
func TestWithoutANotifierNothingHappens(t *testing.T) {
	store := contoh(t)
	svc := layanan(t, store, nil)

	no := nomorPLA(t, store)

	_, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim, no)
	require.ErrorIs(t, err, inboxpladlapredla.ErrNotifierUnavailable)

	tab, _ := inboxpladlapredla.FindTab("pla")
	docs, err := store.Documents(context.Background(), tab, kunciKlaim)
	require.NoError(t, err)
	for _, d := range docs {
		if d.AdviceNo == no {
			require.NotEqual(t, "1", d.Sent)
		}
	}
}

// Dokumen yang sudah terkirim TIDAK dikirimi surat kedua.
func TestAnAlreadySentAdviceSendsNoSecondLetter(t *testing.T) {
	store := contoh(t)
	notifier := &notifierPalsu{}
	svc := layanan(t, store, notifier)

	no := nomorPLA(t, store)

	_, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim, no)
	require.NoError(t, err)
	require.Len(t, notifier.surat, 1)

	// Penekanan kedua: tidak ada surat tambahan.
	_, err = svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pla", kunciKlaim, no)
	require.ErrorIs(t, err, inboxpladlapredla.ErrAdviceAlreadySent)
	require.Len(t, notifier.surat, 1, "surat KEDUA terkirim ke reasuradur")
}

// Tab Pre DLA tidak punya grid rincian, sehingga tidak punya tombol Send.
func TestThePreDLATabCannotSendLetters(t *testing.T) {
	store := contoh(t)
	notifier := &notifierPalsu{}
	svc := layanan(t, store, notifier)

	_, err := svc.SendAdvice(
		context.Background(), "ASM", pemanggil(), "pre-dla", kunciKlaim, "PRE/2026/0001")
	require.ErrorIs(t, err, inboxpladlapredla.ErrDocumentsNotOnTab)
	require.Empty(t, notifier.surat)
}
