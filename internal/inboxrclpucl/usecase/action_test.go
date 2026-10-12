package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/inboxrclpucl/repo/memory"
	"claim-pnc/internal/inboxrclpucl/suratpdf"
	"claim-pnc/internal/inboxrclpucl/usecase"
	"claim-pnc/internal/platform/clock"
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
		Letters:      suratpdf.Renderer{},

		// Jam BERHENTI supaya tanggal dan nomor surat dapat diuji. Waktunya dipilih yang
		// menjawab satu pertanyaan sekaligus: 14.05 WIB membuktikan jamnya 12-jam.
		Clock: clock.FixedAt(time.Date(2026, 10, 2, 7, 5, 9, 0, time.UTC)),
	})
	require.NoError(t, err)
	return s
}

// aksi menjalankan satu tindakan TANPA isian — bentuk yang dipakai "Download Dokumen" dan
// "Tolak Klaim", keduanya tidak menyentuh kedua isian Penerimaan Dokumen.
func aksi(
	t *testing.T,
	s *usecase.Service,
	reference string,
	kind inboxrclpucl.ClaimActionKind,
) (*inboxrclpucl.Document, error) {
	t.Helper()
	return s.PerformAction(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"}, reference, kind,
		inboxrclpucl.ReceiptInput{})
}

// isianLengkap adalah kedua isian wajib Penerimaan Dokumen, terisi sah.
func isianLengkap() inboxrclpucl.ReceiptInput {
	return inboxrclpucl.ReceiptInput{
		Note:       "Dokumen sudah lengkap, diteruskan.",
		CompleteAt: "02/10/2026 09:30",
	}
}

// aksiIsi menjalankan satu tindakan BESERTA kedua isian wajibnya.
//
// Dipakai "Save" dan kedua tombol Kirim. Ketiganya membawa isian karena di Pega ketiganya
// mem-posting form yang sama — lihat Service.PerformAction.
func aksiIsi(
	t *testing.T,
	s *usecase.Service,
	reference string,
	kind inboxrclpucl.ClaimActionKind,
	in inboxrclpucl.ReceiptInput,
) (*inboxrclpucl.Document, error) {
	t.Helper()
	return s.PerformAction(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"}, reference, kind, in)
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

// klaimTolakDiAntrean mencari klaim di tab Kelengkapan Dokumen yang tombol "Tolak Klaim"-nya
// muncul — yaitu klaim jalur RCL (`RCL_PUCL = 1`).
//
// Dicari dari DAFTARNYA, sama alasannya dengan klaimDiAntrean: yang hendak dibuktikan adalah
// klaim KELUAR dari antrean, dan klaim yang tidak pernah ada di antrean tidak membuktikan
// apa pun tentang itu.
func klaimTolakDiAntrean(
	t *testing.T,
	s *usecase.Service,
	store *memory.Store,
) inboxrclpucl.ClaimDetail {
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
		if detail.Allows(inboxrclpucl.ActionRejectClaim) {
			return detail
		}
	}
	t.Fatal(`tidak ada klaim di tab Kelengkapan Dokumen yang tombol "Tolak Klaim"-nya muncul`)
	return inboxrclpucl.ClaimDetail{}
}

func TestTolakKlaimMENUTUPKlaimDanMengeluarkannyaDariAntrean(t *testing.T) {
	// Yang dibuktikan di sini AKIBATNYA, bukan pemanggilannya:
	//
	//   1. klaimnya benar-benar hilang dari antrean RCL/PUCL;
	//   2. status kerjanya menjadi `Resolved-Rejected` — padanan ASMForceCaseClose;
	//   3. TIDAK ada tahap tujuan, berbeda dari kedua tombol Kirim.
	//
	// Butir 3 adalah pembeda pokoknya terhadap "Kirim Ke Analyst", dan ia yang paling mudah
	// hilang bila kelak kedua jalur disatukan menjadi satu fungsi berparameter.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail := klaimTolakDiAntrean(t, s, store)
	require.True(t, adaDiAntrean(t, s, detail.Reference),
		"prasyarat: klaimnya memang ada di antrean sebelum tombolnya ditekan")

	_, err := aksi(t, s, detail.Reference, inboxrclpucl.ActionRejectClaim)
	require.NoError(t, err)

	require.False(t, adaDiAntrean(t, s, detail.Reference),
		"klaim yang sudah ditolak TIDAK boleh tersisa di antrean RCL/PUCL")

	_, dipindahkan := store.MovedTaskOf(detail.Reference)
	require.False(t, dipindahkan,
		`"Tolak Klaim" menutup kasusnya; ia tidak memindahkan klaim ke tahap mana pun`)

	riwayat := store.History(detail.Reference)
	require.Len(t, riwayat, 1)
	require.Equal(t, inboxrclpucl.HistoryNoteRejectClaim, riwayat[0].Note)
	require.Equal(t, "petugascontoh", riwayat[0].Caller)
}

func TestTolakKlaimTIDAKMenungguLayananPega(t *testing.T) {
	// Sebelum 2026-10-06 tombol ini menjawab 503 "dikerjakan di Pega". Ketiga tulisannya
	// seluruhnya pada tabel milik aplikasi ini, sehingga tidak ada satu pun yang dibutuhkan
	// dari Pega — lihat Repo.RejectClaim.
	store := memory.NewSampleStore()
	pega := &pegaPalsu{}
	s := layanan(t, store, pega)

	detail := klaimTolakDiAntrean(t, s, store)

	_, err := aksi(t, s, detail.Reference, inboxrclpucl.ActionRejectClaim)
	require.NoError(t, err)
	require.Empty(t, pega.diterima,
		`"Tolak Klaim" tidak boleh diteruskan ke layanan Pega`)
}

func TestTolakKlaimBERHASILTanpaLayananPegaSamaSekali(t *testing.T) {
	// `Actions` nil adalah keadaan yang SAH — layanan Pega belum dibangun. Tombol ini harus
	// tetap bekerja di sana, bukan menjawab ErrPegaServiceUnavailable.
	store := memory.NewSampleStore()
	s := layanan(t, store, nil)

	detail := klaimTolakDiAntrean(t, s, store)

	_, err := aksi(t, s, detail.Reference, inboxrclpucl.ActionRejectClaim)
	require.NoError(t, err)
	require.False(t, adaDiAntrean(t, s, detail.Reference))
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

	_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.NoError(t, err)

	require.False(t, adaDiAntrean(t, s, detail.Reference),
		"klaim yang sudah dikirim TIDAK boleh tersisa di antrean PUCL")
}

func TestKeduaTombolKirimTIDAKMenungguLayananPega(t *testing.T) {
	// Keduanya menandai klaim selesai di `TC_PNC_PUCL` — tabel milik aplikasi ini.
	// Meneruskannya ke layanan Pega berarti tombolnya menjawab 503 selama layanan itu belum
	// dibangun, padahal tidak ada satu pun yang dibutuhkan dari Pega untuk perpindahan ini.
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

		_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
		require.NoErrorf(t, err, "tindakan %s", kind)
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
	_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.NoError(t, err)
}

func TestKirimMenyimpanCatatanUntukAnalyst(t *testing.T) {
	// Cacat yang ditemukan 2026-10-02 saat rantai dua belas activity ditelusuri: tombol Kirim
	// MEMBUANG catatan yang diketik petugas, karena badan permintaannya hanya dibaca untuk
	// "save". Di Pega tidak demikian — Finish Assignment mem-posting form yang sama, dan
	// `PUCLPost` langkah 10 menuliskan `KomentarPUCL` bersama penandaan klaimnya.
	//
	// Kegagalannya SENYAP: layar menjawab berhasil, klaimnya pindah, dan Analyst menerima
	// pekerjaan tanpa satu kata pun tentang apa yang berubah.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)

	in := inboxrclpucl.ReceiptInput{
		Note:       "Polis dan kuitansi asli sudah diterima 2 Oktober.",
		CompleteAt: "02/10/2026 09:30",
	}
	_, err := aksiIsi(t, s, detail.Reference, kind, in)
	require.NoError(t, err)

	sesudah, err := store.Detail(context.Background(), detail.Reference)
	require.NoError(t, err)
	require.Equal(t, in.Note, sesudah.DocumentReceipt.PUCLNote,
		"catatan yang diketik petugas WAJIB tersimpan oleh tombol Kirim, bukan hanya oleh Save")
}

func TestKirimMENOLAKIsianWajibYangKosong(t *testing.T) {
	// Kedua isian `pyRequired` di section, dan Finish Assignment di Pega menolak form yang
	// salah satunya kosong. Ditiru (`P-5`).
	//
	// Yang dijaga bukan hanya penolakannya melainkan BENTUKNYA: keduanya dikembalikan
	// SEKALIGUS, bukan satu per satu, supaya petugas tidak menekan tombolnya dua kali untuk
	// mengetahui dua hal — lihat ReceiptInput.Validate.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)

	_, err := aksiIsi(t, s, detail.Reference, kind, inboxrclpucl.ReceiptInput{})
	require.Error(t, err)

	var invalid *inboxrclpucl.ValidationError
	require.ErrorAs(t, err, &invalid)
	require.Len(t, invalid.Violations, 2,
		"kedua pelanggaran dikembalikan sekaligus")

	// Dan klaimnya TIDAK boleh bergerak. Penolakan yang tetap memindahkan klaim jauh lebih
	// buruk daripada tidak menolak sama sekali.
	require.True(t, adaDiAntrean(t, s, detail.Reference),
		"klaim yang isiannya ditolak WAJIB tetap di antrean")
}

func TestKirimMelampirkanSuratSepertiDownloadDokumen(t *testing.T) {
	// `PUCLPost` langkah 27 memanggil `AttachAsPDFC` berprekondisi `1==1` — SELALU, berapa
	// pun `param.Status`. Jadi surat itu terbit pada tombol Kirim pula, bukan hanya pada
	// "Download Dokumen".
	//
	// Sebelum 2026-10-02 modul ini hanya menerbitkannya pada "Download Dokumen", dan
	// alasannya templat `SuratPUCL` yang belum ada. Templatnya sudah diterima.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)

	sebelum, err := store.Documents(context.Background(), detail.Reference)
	require.NoError(t, err)

	doc, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.NoError(t, err)
	require.NotNil(t, doc, "tombol Kirim menerbitkan surat pula")

	sesudah, err := store.Documents(context.Background(), detail.Reference)
	require.NoError(t, err)
	require.Len(t, sesudah, len(sebelum)+1,
		"suratnya wajib MELAMPIR ke klaim, bukan sekadar dikembalikan")
}

func TestNamaBerkasSuratMengikutiJalurKlaim(t *testing.T) {
	// `PUCLPost` langkah 20–22 menamainya BERBEDA per jalur, dan jalur ketiga pernah
	// terlewat: klaim `RCL_PUCL = '3'` mendapat `PUCL.pdf`, padahal Pega memberinya
	// `Notification.pdf`.
	require.Equal(t, inboxrclpucl.LetterFileNamePUCL,
		inboxrclpucl.LetterFileNameFor(inboxrclpucl.TrackPUCL))
	require.Equal(t, inboxrclpucl.LetterFileNameRCL,
		inboxrclpucl.LetterFileNameFor(inboxrclpucl.TrackRCL))
	require.Equal(t, inboxrclpucl.LetterFileNameNotification,
		inboxrclpucl.LetterFileNameFor(inboxrclpucl.TrackNotification))

	// Satu baris produksi memang berkode kosong. Ia jatuh ke PUCL.pdf — layar ini antrean
	// PUCL, dan berkas bernama sesuatu lebih berguna daripada tindakan yang gagal.
	require.Equal(t, inboxrclpucl.LetterFileNamePUCL, inboxrclpucl.LetterFileNameFor(""))
}

func TestKirimMEMINDAHKANKlaimKeTahapSendToAnalis(t *testing.T) {
	// INILAH yang diminta Work Owner berkali-kali, dan yang selama ini tidak terjadi:
	// klaimnya BENAR-BENAR pindah, bukan sekadar berubah penanda.
	//
	// Di Pega, `PUCLPost` langkah 17 melepas ticket `SendtoAnalysator`, dan ticket itu
	// menempel pada shape `Send To Analis` (`Assignment5`) — lompatan lateral. Tugas lama
	// ditutup Finish Assignment.
	//
	// Yang dijaga BUKAN pemanggilannya melainkan AKIBATNYA: ada tugas baru, pada tahap yang
	// benar, antrean yang benar, dan bertuan. Tugas Worklist tanpa pemilik tidak muncul di
	// inbox siapa pun — klaimnya hilang dari setiap layar tanpa satu pun galat.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)

	_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.NoError(t, err)

	tugas, ada := store.MovedTaskOf(detail.Reference)
	require.True(t, ada, "tombol Kirim WAJIB membuka tugas baru, bukan sekadar menandai")

	// Ketiganya wajib sama persis dengan modul `registrasi`: tugas dibaca inbox modul lain
	// yang mencocokkan kolom TAHAP sebagai teks.
	require.Equal(t, "kirim-analis", tugas.Stage)
	require.Equal(t, "WORKLIST", tugas.Queue)
	require.NotEmpty(t, tugas.Owner, "tugas Worklist WAJIB bertuan sejak lahir (D-26)")
}

func TestKirimDITOLAKSaatPICTeknikTidakDiketahui(t *testing.T) {
	// Tanpa PIC Teknik, tugas yang dibuat tidak akan bertuan — dan tugas Worklist tanpa
	// pemilik tidak muncul di inbox siapa pun. Klaimnya keluar dari antrean PUCL dan tidak
	// sampai ke mana-mana.
	//
	// Menolak membuat sebabnya terbaca, dan klaimnya tetap di tempatnya.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)
	store.ClearTechnicalPIC(detail.Reference)

	_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.ErrorIs(t, err, inboxrclpucl.ErrTechnicalPICUnknown)

	_, ada := store.MovedTaskOf(detail.Reference)
	require.False(t, ada, "tidak boleh ada tugas yang dibuka saat pemiliknya tidak diketahui")
}

func TestKirimMenulisRiwayatKlaim(t *testing.T) {
	// `PUCLPost` langkah 35 memanggil `InsertHistoryClaimPNC` dengan
	// `statusNote = "Send by PUCL to Analyst"`.
	//
	// Catatan kami sebelumnya menyatakan riwayat tidak dapat ditulis karena "tidak ada tabel
	// riwayat". **Keliru** — `Database/PEGA_JSON_INSERT_HISTORY_CLAIM_PNC.prc` menunjuk
	// `LIST_HISTORY_CLAIM_PNC`, tabel bisnis POOLDATA.
	//
	// Yang dijaga bukan hanya adanya baris melainkan TEKSNYA, karena ia yang terbaca orang
	// saat menelusuri klaim — dan `D-59` menjadikan jejak audit kontrol pengimbang tunggal.
	store := memory.NewSampleStore()
	s := layanan(t, store, &pegaPalsu{})

	detail, kind := klaimDiAntrean(t, s, store)

	_, err := aksiIsi(t, s, detail.Reference, kind, isianLengkap())
	require.NoError(t, err)

	riwayat := store.History(detail.Reference)
	require.Len(t, riwayat, 1, "tepat satu baris riwayat per tindakan")
	require.Equal(t, inboxrclpucl.HistoryNoteFor(kind), riwayat[0].Note)
	require.Equal(t, "petugascontoh", riwayat[0].Caller)
}

func TestTeksRiwayatDisalinAPAADANYADariPega(t *testing.T) {
	// Ketiganya mudah "dirapikan" tanpa sadar, dan ketiganya mengubah data yang tersimpan.
	//
	// Spasi di ujung teks cetak ADA di Pega. Huruf besar/kecil pada kedua teks Kirim pun
	// berbeda, karena sumbernya berbeda: yang satu dari rangkaian tombol, yang satu dari
	// `<statusNote>` `PUCLPost` langkah 35.
	require.Equal(t, "Wait for Complete PUCL Document ",
		inboxrclpucl.HistoryNoteFor(inboxrclpucl.ActionPrintLetter),
		"spasi di ujungnya ADA di Pega dan tidak boleh dipangkas")
	require.Equal(t, "Send by PUCL to Analyst",
		inboxrclpucl.HistoryNoteFor(inboxrclpucl.ActionSendToAnalyst),
		"huruf BESAR pada Send")
	require.Equal(t, "send by PUCL to PIC Teknis",
		inboxrclpucl.HistoryNoteFor(inboxrclpucl.ActionSendToPICTeknik),
		"huruf kecil pada send — berbeda dari di atas, dan itu memang begitu di Pega")

	// "Tolak Klaim" MENULIS riwayat sejak 2026-10-06, dan teksnya pun disalin apa adanya —
	// `<statusNote>` `PUCLPost` langkah 36.
	//
	// Di Pega langkah itu berprekondisi `local.isCFS=="1"`, sehingga klaim yang ditolak
	// sebelum pernah diakseptasi tidak meninggalkan jejak apa pun. Di sini ia ditulis
	// SELALU — selisih yang disengaja, dinyatakan pada HistoryNoteRejectClaim.
	require.Equal(t, "RCL and Close Claim",
		inboxrclpucl.HistoryNoteFor(inboxrclpucl.ActionRejectClaim))

	// Satu tindakan TIDAK menulis riwayat, dan kosong di sini yang menyatakannya: "Save"
	// tidak memanggil `PUCLPost` sama sekali.
	require.Empty(t, inboxrclpucl.HistoryNoteFor(inboxrclpucl.ActionSave))
}

func TestDownloadDokumenMenerbitkanSuratDanMelampirkannya(t *testing.T) {
	// Keluhan Work Owner 2026-10-02: "pas download surat filenya masuk di lihat dokumen".
	//
	// Yang dijaga TIGA hal, dan ketiganya pernah tidak terjadi:
	//
	//   1. surat BENAR-BENAR terbit — berkas PDF, bukan hanya penanda;
	//   2. ia MELAMPIR ke klaim, sehingga terbaca di daftar dokumen;
	//   3. ia TIDAK menempuh layanan Pega, yang belum dibangun.
	//
	// Butir kedua yang paling mudah terlewat: surat yang terbit tetapi tidak melampir akan
	// terasa benar saat tombolnya ditekan, dan hilang begitu layarnya ditutup.
	store := memory.NewSampleStore()
	pega := &pegaPalsu{}
	s := layanan(t, store, pega)

	detail, _ := klaimDiAntrean(t, s, store)
	require.True(t, detail.Allows(inboxrclpucl.ActionPrintLetter))

	sebelum, err := store.Documents(context.Background(), detail.Reference)
	require.NoError(t, err)

	doc, err := aksi(t, s, detail.Reference, inboxrclpucl.ActionPrintLetter)
	require.NoError(t, err)
	require.NotNil(t, doc, "surat wajib terbit, bukan hanya penandanya")

	require.Contains(t, []string{
		inboxrclpucl.LetterFileNamePUCL,
		inboxrclpucl.LetterFileNameRCL,
	}, doc.Name, "nama berkasnya mengikuti Pega")

	sesudah, err := store.Documents(context.Background(), detail.Reference)
	require.NoError(t, err)
	require.Len(t, sesudah, len(sebelum)+1,
		"suratnya wajib MELAMPIR, bukan sekadar dikembalikan ke peramban")

	require.Empty(t, pega.diterima,
		"Download Dokumen tidak lagi menempuh layanan Pega")
}

func TestDownloadDokumenTetapMemindahkanTabMeskiSuratGagalTerbit(t *testing.T) {
	// Tanpa perender surat — keadaan modul ini sebelum templat `SuratPUCL` diterima —
	// tombolnya TETAP memindahkan klaim antartab.
	//
	// Itu bukan kelonggaran: perpindahan tab itulah yang menghambat petugas, dan
	// menggagalkan seluruh tindakan karena berkasnya gagal akan menahan klaimnya di tab
	// pertama selamanya.
	store := memory.NewSampleStore()
	s, err := usecase.NewService(usecase.Options{
		RepoSelector: func(string) (inboxrclpucl.Repo, error) { return store, nil },
		Letters:      nil,
	})
	require.NoError(t, err)

	detail, _ := klaimDiAntrean(t, s, store)

	doc, err := aksi(t, s, detail.Reference, inboxrclpucl.ActionPrintLetter)
	require.NoError(t, err, "tindakannya TIDAK boleh gagal karena berkasnya gagal")
	require.Nil(t, doc, "tanpa perender, tidak ada surat yang dapat dikembalikan")

	// Penandanya tetap tertulis — itulah yang memindahkan klaim antartab, dan penyaring tab
	// pertama adalah `TGL_CETAK_DOKUMEN_PUCL IS NULL`.
	sesudah, err := store.Detail(context.Background(), detail.Reference)
	require.NoError(t, err)
	require.NotEmpty(t, sesudah.Reference, "klaimnya tetap terbaca sesudah tindakan")

	require.False(t, adaDiTab(t, s, inboxrclpucl.TabCetakSurat, detail.Reference),
		"klaim yang suratnya sudah ditandai cetak TIDAK boleh tersisa di tab Cetak Surat")
}

// adaDiTab menjawab apakah klaim tergambar di sebuah tab.
func adaDiTab(t *testing.T, s *usecase.Service, tab, reference string) bool {
	t.Helper()
	listed, err := s.List(context.Background(), "asm",
		inboxrclpucl.Caller{Login: "petugascontoh"},
		inboxrclpucl.QueryInput{Tab: tab},
		inboxrclpucl.Pagination{})
	require.NoError(t, err)
	for _, item := range listed.Page.Items {
		if item.Reference == reference {
			return true
		}
	}
	return false
}

func TestNomorSuratMemakaiJamDuaBelasJam(t *testing.T) {
	// `PUCLPost` merakit nomor surat dengan `@CurrentDate("hh","WIB")`, dan `hh` pada format
	// Java adalah jam 01–12. Surat yang terbit pukul 14.05 WIB karena itu bernomor berawalan
	// `0205`, bukan `1405`.
	//
	// Itu ditiru apa adanya (`P-5`). Uji ini ada supaya tidak ada yang "memperbaikinya"
	// menjadi 24 jam tanpa menyadari ia mengubah nomor surat yang keluar ke cabang.
	siang := time.Date(2026, 10, 2, 7, 5, 9, 0, time.UTC) // 14.05.09 WIB
	require.Equal(t, "020509/Notification.CL.AHID.ASM/10/2026",
		inboxrclpucl.NewLetterNumber(siang, inboxrclpucl.LetterCategory))

	// Tengah malam WIB menjadi 12, bukan 00 — itu pun perilaku `hh`.
	tengahMalam := time.Date(2026, 10, 1, 17, 0, 0, 0, time.UTC) // 00.00.00 WIB
	require.Equal(t, "120000/Notification.CL.AHID.ASM/10/2026",
		inboxrclpucl.NewLetterNumber(tengahMalam, inboxrclpucl.LetterCategory))

	require.Equal(t, "02 Oktober 2026", inboxrclpucl.NewLetterDate(siang))
}
