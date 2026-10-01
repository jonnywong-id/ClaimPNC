package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
	"claim-pnc/internal/inboxrclpucl/usecase"
)

// pegaPalsu merekam tindakan yang diteruskan ke layanan Pega.
type pegaPalsu struct {
	diterima []inboxrclpucl.ClaimActionKind
}

func (p *pegaPalsu) Perform(_ context.Context, cmd inboxrclpucl.ClaimActionCommand) error {
	p.diterima = append(p.diterima, cmd.Kind)
	return nil
}

func layanan(t *testing.T, store *memory.Store, pega inboxrclpucl.ClaimActions) *usecase.Service {
	t.Helper()
	s, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return store, nil },
		Actions:      pega,
	})
	require.NoError(t, err)
	return s
}

// klaimDiAntrean mencari klaim yang BENAR-BENAR tergambar di tab Kelengkapan Dokumen dan
// salah satu tombol Kirim-nya muncul.
//
// Dicari dari DAFTARNYA, bukan dari baris contoh mana pun, karena yang hendak dibuktikan
// adalah perpindahan antrean — dan klaim yang tidak pernah ada di antrean tidak dapat
// membuktikan apa pun tentang keluarnya dari antrean.
//
// Kedua tombol Kirim diterima karena keduanya menempuh jalur yang sama; mana yang muncul
// ditentukan lini bisnis klaimnya — PA memunculkan "Kirim Ke Analyst", Travel memunculkan
// "Kirim ke PIC Teknik".
func klaimDiAntrean(
	t *testing.T,
	s *usecase.Service,
	store *memory.Store,
) (inboxrclpucl.ClaimDetail, inboxrclpucl.ClaimActionKind) {
	t.Helper()

	listed, err := s.List(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"},
		inboxrclpucl.QueryInput{Tab: inboxrclpucl.TabKelengkapanDokumen},
		inboxrclpucl.Pagination{})
	require.NoError(t, err)

	for _, item := range listed.Page.Items {
		detail, err := store.Detail(context.Background(), item.Reference)
		if err != nil {
			continue
		}
		for _, kind := range []inboxrclpucl.ClaimActionKind{
			inboxrclpucl.ActionSendToAnalyst,
			inboxrclpucl.ActionSendToPICTeknik,
		} {
			if detail.Allows(kind) {
				return detail, kind
			}
		}
	}
	t.Fatal("tidak ada klaim di tab Kelengkapan Dokumen yang tombol Kirim-nya muncul")
	return inboxrclpucl.ClaimDetail{}, ""
}

// adaDiAntrean menjawab apakah klaim masih tergambar di tab Kelengkapan Dokumen.
func adaDiAntrean(t *testing.T, s *usecase.Service, reference string) bool {
	t.Helper()
	listed, err := s.List(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"},
		inboxrclpucl.QueryInput{Tab: inboxrclpucl.TabKelengkapanDokumen},
		inboxrclpucl.Pagination{})
	require.NoError(t, err)
	for _, item := range listed.Page.Items {
		if item.Reference == reference {
			return true
		}
	}
	return false
}

func TestKirimKeAnalystMengeluarkanKlaimDariAntreanPUCL(t *testing.T) {
	// Inilah yang diminta Work Owner berkali-kali: klaimnya BENAR-BENAR pindah, bukan sekadar
	// menjawab "berhasil". Yang dibuktikan di sini bukan pemanggilan melainkan AKIBATNYA —
	// penandanya berubah, sehingga klaim tidak lagi menjadi pekerjaan PUCL.
	store := memory.NewSampleStore()
	pega := &pegaPalsu{}
	s := layanan(t, store, pega)

	detail, kind := klaimDiAntrean(t, s, store)
	require.True(t, adaDiAntrean(t, s, detail.Reference),
		"prasyarat: klaimnya memang ada di antrean sebelum tombolnya ditekan")

	require.NoError(t, s.PerformAction(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"}, detail.Reference, kind))

	require.False(t, adaDiAntrean(t, s, detail.Reference),
		"klaim yang sudah dikirim TIDAK boleh tersisa di antrean PUCL")
}

func TestKeduaTombolKirimTIDAKMenungguLayananPega(t *testing.T) {
	// Keduanya menandai klaim selesai di `TC_PNC_PUCL` — tabel milik aplikasi ini. Meneruskannya
	// ke layanan Pega berarti tombolnya menjawab 503 selama layanan itu belum dibangun,
	// padahal tidak ada satu pun yang dibutuhkan dari Pega untuk perpindahan ini.
	for _, kind := range []inboxrclpucl.ClaimActionKind{
		inboxrclpucl.ActionSendToAnalyst,
		inboxrclpucl.ActionSendToPICTeknik,
	} {
		store := memory.NewSampleStore()
		pega := &pegaPalsu{}
		s := layanan(t, store, pega)

		detail, _ := klaimDiAntrean(t, s, store)
		if !detail.Allows(kind) {
			continue
		}

		require.NoErrorf(t, s.PerformAction(context.Background(), "asm",
			inboxrclpucl.Caller{Login: "petugascontoh"}, detail.Reference, kind),
			"tindakan %s", kind)
		require.Emptyf(t, pega.diterima,
			"tindakan %s tidak boleh diteruskan ke layanan Pega", kind)
	}
}

func TestKirimTetapBERHASILTanpaLayananPegaSamaSekali(t *testing.T) {
	// Penjaga paling langsung terhadap keluhan "masih tidak bisa": tanpa pengisi seam Pega
	// sama sekali — keadaan nyata hari ini — tombolnya tetap menjalankan tindakannya.
	store := memory.NewSampleStore()
	s := layanan(t, store, nil)

	detail, kind := klaimDiAntrean(t, s, store)
	require.NoError(t, s.PerformAction(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"}, detail.Reference, kind))
}

func TestTindakanLainTETAPMenempuhLayananPega(t *testing.T) {
	store := memory.NewSampleStore()
	pega := &pegaPalsu{}
	s := layanan(t, store, pega)

	detail, _ := klaimDiAntrean(t, s, store)
	require.True(t, detail.Allows(inboxrclpucl.ActionPrintLetter))

	require.NoError(t, s.PerformAction(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"},
		detail.Reference, inboxrclpucl.ActionPrintLetter))
	require.Equal(t, []inboxrclpucl.ClaimActionKind{inboxrclpucl.ActionPrintLetter},
		pega.diterima, "Download Dokumen membuat PDF dan mengirim email — itu milik Pega")
}
