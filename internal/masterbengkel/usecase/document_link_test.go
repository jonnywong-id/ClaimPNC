package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/masterbengkel"
	"claim-pnc/internal/masterbengkel/repo/memory"
	"claim-pnc/internal/masterbengkel/usecase"
)

// Membuka dokumen yang sudah terunggah — padanan tombol lihat dokumen pada
// `Section/ApprovalMasterBengkelHE`.
//
// Yang diuji di sini BUKAN penerbitan alamatnya (itu milik modul dokumen penunjang),
// melainkan tiga keputusan yang memang milik modul ini: kapan layanan penyimpanan
// disentuh, kapan TIDAK disentuh, dan keadaan mana yang dibedakan dari mana.

// fakeLinker berdiri di tempat penerbit alamat.
//
// Ia mencatat panggilannya supaya uji dapat menegaskan bahwa layanan penyimpanan TIDAK
// disentuh pada kasus yang sudah dapat dijawab tanpa menyentuhnya — pemeriksaan yang
// tidak dapat dilakukan hanya dengan melihat nilai baliknya.
type fakeLinker struct {
	fakeUploader

	url       string
	expiresAt time.Time
	err       error

	calls []linkCall
}

type linkCall struct{ portal, imageID, by string }

func (l *fakeLinker) Link(
	_ context.Context,
	portal, imageID, by string,
) (masterbengkel.DocumentLink, error) {
	l.calls = append(l.calls, linkCall{portal: portal, imageID: imageID, by: by})
	if l.err != nil {
		return masterbengkel.DocumentLink{}, l.err
	}
	return masterbengkel.DocumentLink{URL: l.url, ExpiresAt: l.expiresAt}, nil
}

// linkService merakit layanan dengan pengunggah yang JUGA dapat menerbitkan alamat —
// persis seperti adapter sungguhan, yang mengisi kedua seam sekaligus.
func linkService(t *testing.T, linker *fakeLinker) (*usecase.Service, *memory.Repo) {
	t.Helper()

	repo := memory.NewSampleRepo()
	service, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (masterbengkel.Store, error) { return repo, nil },
		Clock:        fixedClock{at: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)},
		Uploader:     linker,
	})
	require.NoError(t, err)
	return service, repo
}

// unggah menaruh satu dokumen pada sebuah bengkel lewat jalur yang sesungguhnya.
func unggah(t *testing.T, service *usecase.Service, workshopID string) {
	t.Helper()

	_, err := service.UploadDocument(context.Background(), "ASM",
		usecase.Actor{Login: "PENGUJI"},
		masterbengkel.UploadInput{
			WorkshopID: workshopID,
			FileName:   "bukti-kerja-sama.pdf",
			Content:    []byte("isi berkas"),
		})
	require.NoError(t, err)
}

func TestDocumentLinkIssuesAddressForStoredFile(t *testing.T) {
	linker := &fakeLinker{
		fakeUploader: fakeUploader{imageID: "img-9001"},
		url:          "https://penyimpanan.contoh/berkas/img-9001",
		expiresAt:    time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC),
	}
	service, repo := linkService(t, linker)
	id := firstWorkshopID(t, repo)
	unggah(t, service, id)

	link, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.NoError(t, err)
	require.Equal(t, "https://penyimpanan.contoh/berkas/img-9001", link.URL)
	require.Equal(t, time.Date(2026, 9, 30, 11, 0, 0, 0, time.UTC), link.ExpiresAt)

	// IMAGEID yang diteruskan adalah milik dokumen yang benar-benar tertaut, bukan ID
	// bengkelnya — keduanya mudah tertukar dan akibatnya tidak terlihat di layar.
	require.Len(t, linker.calls, 1)
	require.Equal(t, "img-9001", linker.calls[0].imageID)
	require.Equal(t, "ASM", linker.calls[0].portal)

	// Pelakunya adalah yang MEMBUKA, bukan yang mengunggah: memperpanjang alamat menempuh
	// pencatatan izin akses atas namanya.
	require.Equal(t, "PEMBUKA", linker.calls[0].by)
}

// Dokumen warisan Pega: barisnya ada, `IMAGEID`-nya tidak pernah terisi.
//
// Ia dibedakan dari "belum punya dokumen" — nama berkasnya terpampang di layar, sehingga
// menjawab "tidak ada dokumen" akan bertentangan dengan apa yang pengguna lihat.
func TestDocumentLinkRejectsLegacyRowWithoutFile(t *testing.T) {
	linker := &fakeLinker{url: "https://tidak-boleh-dipakai"}
	service, repo := linkService(t, linker)
	id := firstWorkshopID(t, repo)

	require.NoError(t, repo.SaveDocument(context.Background(), id, masterbengkel.Document{
		ID:      "260000000042",
		ImageID: "", // persis keadaan seluruh dokumen warisan
		Name:    "dokumen-lama.pdf",
	}))

	_, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.ErrorIs(t, err, masterbengkel.ErrDocumentLinkEmpty)

	// Layanan penyimpanan TIDAK disentuh: jawabannya sudah pasti sejak metadata dibaca,
	// dan memintakan alamat untuk berkas yang tidak pernah ada hanya menambah satu
	// panggilan jaringan yang pasti gagal.
	require.Empty(t, linker.calls)
}

func TestDocumentLinkSaysNotFoundWhenNothingAttached(t *testing.T) {
	service, repo := linkService(t, &fakeLinker{url: "https://contoh"})
	id := firstWorkshopID(t, repo)

	_, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.ErrorIs(t, err, masterbengkel.ErrDocumentNotFound)
}

// Alamat kosong tanpa galat diperlakukan sebagai berkas yang tidak dapat dibuka.
//
// Tanpa pemeriksaan ini layar menerima 200 lalu membuka tab kosong, dan pengguna tidak
// punya cara tahu bahwa yang gagal adalah penyimpanannya, bukan peramban mereka.
func TestDocumentLinkRejectsEmptyAddress(t *testing.T) {
	linker := &fakeLinker{fakeUploader: fakeUploader{imageID: "img-9002"}, url: "   "}
	service, repo := linkService(t, linker)
	id := firstWorkshopID(t, repo)
	unggah(t, service, id)

	_, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.ErrorIs(t, err, masterbengkel.ErrDocumentLinkEmpty)
}

func TestDocumentLinkForwardsUpstreamFailure(t *testing.T) {
	linker := &fakeLinker{
		fakeUploader: fakeUploader{imageID: "img-9003"},
		err:          masterbengkel.ErrDocumentLinkRenewFailed,
	}
	service, repo := linkService(t, linker)
	id := firstWorkshopID(t, repo)
	unggah(t, service, id)

	_, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.ErrorIs(t, err, masterbengkel.ErrDocumentLinkRenewFailed)
}

// Pengunggah yang tidak dapat menerbitkan alamat menyisakan jalur buka yang mati, dan ia
// harus menolak dengan sebab yang jelas — bukan memanggil nil.
//
// `fakeUploader` sengaja TIDAK mengimplement DocumentLinker, sehingga rakitan ini adalah
// keadaan yang benar-benar mungkin terjadi, bukan yang dikarang untuk uji ini.
func TestDocumentLinkRejectsWhenUploaderCannotIssueAddress(t *testing.T) {
	service, repo := documentService(t)
	id := firstWorkshopID(t, repo)
	unggah(t, service, id)

	_, err := service.DocumentLink(context.Background(), "ASM", id,
		usecase.Actor{Login: "PEMBUKA"})
	require.ErrorIs(t, err, masterbengkel.ErrDocumentLinkUnsupported)
	require.False(t, errors.Is(err, masterbengkel.ErrDocumentLinkEmpty),
		"cacat perakitan tidak boleh terbaca sebagai berkas yang tidak tersimpan")
}
