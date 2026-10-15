// Package usecase mengorkestrasi modul Inbox PLA, DLA, Pre DLA.
//
// Tiga operasi:
//
//	Metadata   menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List       mengambil isi satu antrean
//	Documents  mengambil isi grid "Detail PLA List" / "Detail DLA List" satu klaim
//
// Ekspor TIDAK menjadi operasi keempat: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// # TIDAK ADA operasi yang MENULIS, dan itu keputusan Work Owner
//
// Layar lama punya tiga tombol yang menulis — "Send", "Upload File Penunjang", dan
// "Print Pre DLA". Ketiganya tidak dibangun, dan Work Owner memutuskannya pada 2026-09-26
// setelah keberatan berikut disampaikan.
//
// "Send" di Pega (`Activity/UpdateDetailPLA2-Act.xml`) menjalankan EMPAT hal berurutan:
//
//  1. mengumpulkan lampiran dan MENGIRIM EMAIL ke reasuradur (`ASMSendsEmailAttachments`)
//  2. memperbarui master reasuransi (`UpdateEmailReas` -> `Database/UPDATEREAS.prc`)
//  3. menyisipkan dokumennya (`InsertDokumenPLADLA`)
//  4. menandai dokumen terkirim (`UpdatePLAList` -> `ISKIRIM = '1'`)
//
// Hanya langkah 4 yang dapat dikerjakan sekarang; tiga yang pertama menembak SMTP dan
// penyimpanan dokumen yang belum tersambung. Mengerjakan langkah 4 sendirian adalah
// pilihan terburuk dari ketiganya: barisnya HILANG dari antrean — itulah yang disaring
// `ISKIRIM` — padahal tidak satu pun surat sampai ke reasuradur, dan tidak ada apa pun
// yang tersisa untuk memberi tahu siapa pun bahwa ia belum terkirim.
//
// Ketiadaannya dinyatakan ke pengguna lewat PlannedDifferences dan lewat tombol yang
// menjawab alasan, bukan disamarkan.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/inboxpladlapredla"
)

// Service melayani modul Inbox PLA, DLA, Pre DLA.
type Service struct {
	repoSelector inboxpladlapredla.RepoSelector
	logger       *slog.Logger

	// notifier mengirim surat PLA/DLA. Nil berarti "SEND" belum dapat dijalankan, dan
	// tombolnya menjawab alasan yang menyebutkan itu.
	notifier inboxpladlapredla.Notifier

	// composeBody menyusun badan surat. Nil berarti badan kosong.
	composeBody BodyComposer
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxpladlapredla.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger

	// Notifier boleh nil.
	//
	// Nil BUKAN kegagalan diam: tombol "SEND" menjawab alasan yang menyebut sambungan
	// surelnya belum disiapkan. Itu berbeda dari surat yang dicoba lalu gagal, dan
	// pengguna harus dapat membedakan keduanya.
	Notifier inboxpladlapredla.Notifier

	// ComposeBody menyusun badan surat. Nil berarti badan kosong.
	ComposeBody BodyComposer
}

// NewService membentuk layanan modul Inbox PLA, DLA, Pre DLA.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxpladlapredla/usecase: RepoSelector wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		logger:       o.Logger,
		notifier:     o.Notifier,
		composeBody:  o.ComposeBody,
	}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi daftar.
type Metadata struct {
	// Tabs adalah ketiga daftar beserta kolom dan judul penyaringnya.
	Tabs []inboxpladlapredla.Tab

	// DefaultTab adalah daftar yang terbuka pertama kali.
	DefaultTab string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:               inboxpladlapredla.Tabs(),
		DefaultTab:         inboxpladlapredla.DefaultTab,
		PlannedDifferences: inboxpladlapredla.PlannedDifferences,
	}
}

// Listed adalah isi satu antrean beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxpladlapredla.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim:
	// daftar yang diminta kosong menjadi daftar bawaan, dan layar harus tahu daftar mana
	// yang sebenarnya dijawab.
	Query inboxpladlapredla.Query
}

// List mengambil isi satu antrean.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	input inboxpladlapredla.QueryInput,
	page inboxpladlapredla.Pagination,
) (Listed, error) {
	query, err := inboxpladlapredla.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf(
			"mengambil isi antrean %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan.
	//
	// Setiap baris layar ini memuat nama tertanggung DAN nomor polis — keduanya data
	// nasabah (`D-69`). Ketiga daftarnya bersama dan tidak ada satu pun penyaring
	// berbasis pengguna, sehingga selama `TKT-F3-004` belum selesai, jejak inilah
	// satu-satunya yang menyatakan siapa yang membukanya (`D-59`).
	//
	// Kata kunci pencarian TIDAK ikut dicatat: ia dapat memuat nomor klaim, dan nomor
	// klaim adalah data nasabah. Yang dicatat adalah APAKAH pengguna mencari, dan
	// APAKAH ia membatasi tanggalnya.
	if s.logger != nil {
		s.logger.Info(
			"antrean PLA/DLA/Pre DLA dibuka",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("daftar", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Bool("mencari", query.Search != ""),
			slog.Bool("membatasi_tanggal", query.From != nil || query.To != nil),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Documented adalah isi grid rincian satu klaim.
type Documented struct {
	// Tab adalah daftar yang rinciannya diminta, lengkap dengan kolomnya.
	Tab inboxpladlapredla.Tab

	// ClaimKey adalah kunci klaim yang diminta, setelah dipangkas.
	ClaimKey string

	// Items adalah barisnya. Kosong BUKAN galat — lihat Documents.
	Items []inboxpladlapredla.Document
}

// Documents mengambil isi grid "Detail PLA List" / "Detail DLA List" satu klaim.
//
// # Kenapa ia TIDAK memeriksa apakah klaimnya ada di antrean pemanggil
//
// Karena grid rincian di Pega pun tidak: `GetPLAList` menyaring `claimid` saja, tanpa satu
// pun penyaring daftar ikut. Klaim yang dokumennya baru saja dikirim petugas lain — dan
// karena itu sudah keluar dari antrean — tetap harus dapat dibuka rinciannya; menolaknya
// akan membuat layar gagal justru pada keadaan yang paling sering terjadi.
//
// Batas yang tetap berlaku adalah PORTAL: kuncinya dicari di basis data entitas yang
// sedang dipilih, dan kunci milik entitas lain menghasilkan "tidak ditemukan" — bukan
// diam-diam dilayani koneksi lain (`R-20`).
//
// # Tab Pre DLA ditolak, bukan dijawab daftar kosong
//
// Ia tidak punya grid rincian di Pega. Daftar kosong akan terbaca sebagai "klaim ini
// belum punya Pre-DLA", padahal setiap baris di tab itu pasti punya — pesan yang salah
// justru pada keadaan yang tidak pernah terjadi.
func (s *Service) Documents(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	tabCode string,
	claimKey string,
) (Documented, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Documented{}, inboxpladlapredla.ErrCallerUnknown
	}

	tab, known := inboxpladlapredla.FindTab(tabCode)
	if !known {
		return Documented{}, inboxpladlapredla.NewValidationError(
			[]inboxpladlapredla.Violation{{
				Field:   inboxpladlapredla.FieldTab,
				Message: "Daftar tidak dikenal. Pilih PLA, DLA, atau Pre DLA.",
			}})
	}

	if !tab.HasDocuments() {
		return Documented{}, inboxpladlapredla.ErrDocumentsNotOnTab
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Documented{}, err
	}

	items, err := repo.Documents(ctx, tab, claimKey)
	if err != nil {
		if errors.Is(err, inboxpladlapredla.ErrRowNotFound) {
			return Documented{}, err
		}
		return Documented{}, fmt.Errorf(
			"mengambil rincian %s: %w", tab.Code, err)
	}

	if s.logger != nil {
		// Kunci klaim TIDAK dicatat: ia memuat nomor klaim, dan nomor klaim adalah data
		// nasabah (`D-69`). Yang dicatat adalah bahwa rincian dibuka, pada daftar mana,
		// dan berapa dokumen yang terlihat — cukup untuk menelusuri pola pemakaian tanpa
		// menyalin data nasabah ke dalam log.
		s.logger.Info("rincian PLA/DLA dibuka",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("daftar", tab.Code),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.Int("dokumen", len(items)))
	}

	return Documented{Tab: tab, ClaimKey: claimKey, Items: items}, nil
}

// Printable adalah isi panel "Print Pre DLA" satu klaim.
type Printable struct {
	// Tab selalu tab Pre DLA, lengkap dengan PrintColumns-nya.
	Tab inboxpladlapredla.Tab

	// ClaimKey adalah kunci klaim yang diminta, setelah dipangkas.
	ClaimKey string

	// Items adalah barisnya. Kosong BUKAN galat.
	//
	// Kosong punya DUA sebab yang sama-sama sah: klaimnya belum punya Pre-DLA, atau
	// Pre-DLAnya ada tetapi belum satu pun lampirannya cocok. Keduanya tidak dibedakan
	// di sini — yang dibedakan hanyalah keduanya dari kunci klaim yang keliru, dan itu
	// dijawab ErrRowNotFound.
	Items []inboxpladlapredla.PreDLADocument
}

// PrintList mengambil isi panel "Print Pre DLA" satu klaim.
//
// Namanya sengaja BUKAN Print. Tombolnya bernama "Print Pre DLA", tetapi yang dilakukannya
// adalah MEMBUKA DAFTAR — tidak ada satu pun berkas yang dihasilkan, baik di Pega maupun
// di sini. Menamainya Print akan membuat pemanggil berikutnya mengira mesin dokumen
// (`D-11`) sudah tersambung di jalur ini.
//
// Ia MEMBACA saja. Tombol "Kirim Pre DLA" di dalam panelnya menulis ke `T_PREDLALIST`, dan
// tabel itu masih dimiliki Pega selama masa paralel (`P-1`) — tombolnya karena itu
// menjawab alasan, sama seperti tombol tulis lain di modul ini.
func (s *Service) PrintList(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	claimKey string,
) (Printable, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Printable{}, inboxpladlapredla.ErrCallerUnknown
	}

	tab, known := inboxpladlapredla.FindTab("pre-dla")
	if !known {
		return Printable{}, fmt.Errorf(
			"inboxpladlapredla/usecase: tab Pre DLA tidak terdaftar")
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Printable{}, err
	}

	items, err := repo.PrintPreDLA(ctx, claimKey)
	if err != nil {
		if errors.Is(err, inboxpladlapredla.ErrRowNotFound) {
			return Printable{}, err
		}
		return Printable{}, fmt.Errorf("mengambil panel Print Pre DLA: %w", err)
	}

	if s.logger != nil {
		// Kunci klaim TIDAK dicatat — ia memuat nomor klaim, dan nomor klaim adalah data
		// nasabah (`D-69`).
		s.logger.Info("panel Print Pre DLA dibuka",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.Int("dokumen", len(items)))
	}

	return Printable{Tab: tab, ClaimKey: claimKey, Items: items}, nil
}

// SendPreDLA menandai satu Pre-DLA sebagai terkirim.
//
// # Ia satu-satunya operasi TULIS di modul ini
//
// Menggantikan `Activity/SetTglKirimPreDLA_Act-Act.xml`. Konsekuensi `P-1` — dua sistem
// menulis satu tabel — diterima Work Owner pada 2026-09-27 setelah keberatannya
// disampaikan, dan dipagari di sisi kueri: baris yang sudah terkirim tidak disentuh.
//
// # Kenapa ia TIDAK mengirim surat apa pun
//
// Karena tombolnya memang tidak mengirim surat. Namanya "Kirim Pre DLA", tetapi yang
// dikerjakan `SetTglKirimPreDLA_Act` hanyalah menandai baris dan mengisi tanggalnya —
// satu `RDB-Save`, tidak lebih. Yang mengirim surat adalah tombol "SEND" di grid rincian,
// dan itu operasi yang sama sekali berbeda.
func (s *Service) SendPreDLA(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	claimKey string,
	adviceNo string,
) error {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return inboxpladlapredla.ErrCallerUnknown
	}

	if strings.TrimSpace(adviceNo) == "" {
		return inboxpladlapredla.NewValidationError(
			[]inboxpladlapredla.Violation{{
				Field:   inboxpladlapredla.FieldAdviceNo,
				Message: "Nomor Pre-DLA tidak disebutkan.",
			}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	berubah, err := repo.MarkPreDLASent(ctx, claimKey, adviceNo)
	if err != nil {
		if errors.Is(err, inboxpladlapredla.ErrRowNotFound) {
			return err
		}
		return fmt.Errorf("menandai Pre-DLA terkirim: %w", err)
	}

	if !berubah {
		return inboxpladlapredla.ErrPreDLAAlreadySent
	}

	if s.logger != nil {
		// Nomor Pre-DLA DICATAT, dan itu berbeda dari operasi baca di modul ini.
		//
		// Ia perubahan bernilai bisnis — `D-28` mewajibkannya tercatat, dan `D-59`
		// menjadikan jejak audit satu-satunya kontrol pengimbang karena tidak ada
		// pemisahan tugas. Tanpa nomornya, catatan itu tidak dapat menjawab "yang mana".
		//
		// Kunci klaim tetap TIDAK dicatat: ia memuat nomor klaim, dan nomor klaim adalah
		// data nasabah (`D-69`). Nomor Pre-DLA bukan.
		s.logger.Info("Pre-DLA ditandai terkirim",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("no_pre_dla", strings.TrimSpace(adviceNo)),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias))
	}

	return nil
}

// SendResult adalah hasil satu pengiriman "SEND".
type SendResult struct {
	Tab      inboxpladlapredla.Tab
	AdviceNo string

	// Recipients adalah alamat yang benar-benar dikirimi.
	Recipients []string

	// Attachments adalah jumlah berkas yang ikut.
	Attachments int
}

// BodyComposer menyusun badan surat.
//
// Ia seam tersendiri, terpisah dari Notifier, karena keduanya berubah karena sebab yang
// berbeda: badan surat berubah ketika isinya dipersoalkan, pengiriman berubah ketika
// infrastrukturnya berubah. Menyatukannya membuat perubahan kalimat menyentuh kode yang
// membuka soket.
type BodyComposer func(
	tab inboxpladlapredla.Tab,
	advice inboxpladlapredla.SendableAdvice,
	claim inboxpladlapredla.ClaimSummary,
	attachments int,
) string

// SendAdvice mengirim surat PLA/DLA ke reasuradur, lalu menandai dokumennya terkirim.
//
// # URUTANNYA adalah isi utama fungsi ini
//
//	1  baca dokumen + klaimnya
//	2  tolak bila alamat kosong, atau sudah terkirim
//	3  ambil lampiran
//	4  KIRIM SURAT                      <- menyentuh dunia luar
//	5  bila (4) berhasil: tandai terkirim + perbarui master, dalam satu transaksi
//
// Pega menempuhnya terbalik — menandai lebih dulu, mengirim belakangan. Urutan itu tidak
// dibawa; alasannya ada di inboxpladlapredla/send.go.
//
// # Kegagalan SETELAH surat terkirim
//
// Bila langkah 5 gagal, suratnya sudah sampai dan tidak dapat ditarik. Yang dikembalikan
// karena itu adalah galat yang **menyebutkan bahwa suratnya sudah terkirim** — supaya
// pengguna tidak menekan tombolnya lagi dan mengirim surat kedua.
//
// Barisnya tetap di antrean, dan itu memang keadaan yang benar: dokumennya belum tercatat
// terkirim. Yang salah adalah catatannya, bukan suratnya.
func (s *Service) SendAdvice(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	tabCode string,
	claimKey string,
	adviceNo string,
) (SendResult, error) {
	return s.SendAdviceWith(ctx, portalAlias, caller, tabCode, claimKey, adviceNo, nil)
}

// SendAdviceWith sama dengan SendAdvice, ditambah berkas yang dilampirkan di DEPAN lampiran
// klaim — dipakai SEND ALL PLA untuk melampirkan PDF PLA-nya sendiri, padanan berkas kategori
// PLA hasil `AttachAsPDFC` yang dilampirkan `UpdateDetailPLA2` (Obj-Browse + ASMCollectAttachments).
func (s *Service) SendAdviceWith(
	ctx context.Context,
	portalAlias string,
	caller inboxpladlapredla.Caller,
	tabCode string,
	claimKey string,
	adviceNo string,
	extra []inboxpladlapredla.Attachment,
) (SendResult, error) {
	var hasil SendResult

	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return hasil, inboxpladlapredla.ErrCallerUnknown
	}

	if s.notifier == nil {
		return hasil, fmt.Errorf(
			"%w: pengirim surat belum dipasang", inboxpladlapredla.ErrNotifierUnavailable)
	}

	tab, known := inboxpladlapredla.FindTab(tabCode)
	if !known {
		return hasil, inboxpladlapredla.NewValidationError(
			[]inboxpladlapredla.Violation{{
				Field:   inboxpladlapredla.FieldTab,
				Message: "Daftar tidak dikenal. Pilih PLA atau DLA.",
			}})
	}
	if !tab.HasDocuments() {
		return hasil, inboxpladlapredla.ErrDocumentsNotOnTab
	}

	if strings.TrimSpace(adviceNo) == "" {
		return hasil, inboxpladlapredla.NewValidationError(
			[]inboxpladlapredla.Violation{{
				Field:   inboxpladlapredla.FieldAdviceNo,
				Message: "Nomor dokumen tidak disebutkan.",
			}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return hasil, err
	}

	// --- 1. baca dokumen dan klaimnya -------------------------------------------
	advice, claim, err := repo.AdviceForSending(ctx, tab, claimKey, adviceNo)
	if err != nil {
		if errors.Is(err, inboxpladlapredla.ErrRowNotFound) {
			return hasil, err
		}
		return hasil, fmt.Errorf("membaca dokumen untuk dikirim: %w", err)
	}

	// --- 2. tolak sebelum apa pun terjadi ----------------------------------------
	if err := advice.CanBeSent(); err != nil {
		return hasil, err
	}

	penerima := advice.Recipients()
	if len(penerima) == 0 {
		// Kolomnya terisi, tetapi tidak satu pun isinya berbentuk alamat surel.
		//
		// Itu berbeda dari kolom kosong, dan pesannya harus berbeda pula: yang pertama
		// menyuruh mengisi, yang kedua menyuruh memperbaiki.
		return hasil, inboxpladlapredla.NewValidationError(
			[]inboxpladlapredla.Violation{{
				Field: inboxpladlapredla.FieldAdviceNo,
				Message: "Alamat surel reasuradur tidak dapat dibaca sebagai alamat " +
					"yang sah. Perbaiki lewat master reasuransi.",
			}})
	}

	// --- 3. ambil lampiran -------------------------------------------------------
	lampiran, err := repo.AttachmentsForClaim(
		ctx, claimKey, inboxpladlapredla.AttachmentCategory(tab))
	if err != nil {
		return hasil, fmt.Errorf("mengambil lampiran: %w", err)
	}
	if len(extra) > 0 {
		lampiran = append(append([]inboxpladlapredla.Attachment{}, extra...), lampiran...)
	}

	// --- 4. kirim suratnya --------------------------------------------------------
	badan := s.composeBody
	if badan == nil {
		badan = func(
			_ inboxpladlapredla.Tab,
			_ inboxpladlapredla.SendableAdvice,
			_ inboxpladlapredla.ClaimSummary,
			_ int,
		) string {
			return ""
		}
	}

	surat := inboxpladlapredla.Letter{
		To:          penerima,
		Subject:     inboxpladlapredla.Subject(tab, claim),
		HTMLBody:    badan(tab, advice, claim, len(lampiran)),
		Attachments: lampiran,
	}

	if err := s.notifier.SendAdvice(ctx, surat); err != nil {
		if errors.Is(err, inboxpladlapredla.ErrNotifierUnavailable) {
			return hasil, err
		}
		return hasil, fmt.Errorf("%w: %s", inboxpladlapredla.ErrLetterNotSent, err)
	}

	// Surat SUDAH terkirim mulai baris ini. Tidak ada jalan kembali.
	if s.logger != nil {
		s.logger.Info("surat PLA/DLA terkirim",
			slog.String("modul", "inbox-pla-dla-pre-dla"),
			slog.String("daftar", tab.Code),
			slog.String("no_dokumen", strings.TrimSpace(adviceNo)),
			slog.Int("penerima", len(penerima)),
			slog.Int("lampiran", len(lampiran)),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias))
	}

	hasil = SendResult{
		Tab:         tab,
		AdviceNo:    strings.TrimSpace(adviceNo),
		Recipients:  penerima,
		Attachments: len(lampiran),
	}

	// --- 5. tandai terkirim -------------------------------------------------------
	berubah, err := repo.MarkAdviceSent(ctx, tab, claimKey, adviceNo, advice)
	if err != nil {
		// Suratnya sudah terkirim; catatannya yang gagal. Galatnya menyebut keduanya.
		return hasil, fmt.Errorf(
			"%w: surat SUDAH terkirim ke %d penerima, tetapi penandaannya gagal — "+
				"jangan menekan tombol ini lagi sebelum memeriksa dokumennya: %s",
			inboxpladlapredla.ErrSentButNotMarked, len(penerima), err)
	}
	if !berubah {
		// Dokumennya ditandai orang lain di antara langkah 2 dan 5. Suratnya terlanjur
		// terkirim, dan kemungkinan besar dua kali.
		return hasil, fmt.Errorf(
			"%w: surat SUDAH terkirim, tetapi dokumennya ternyata sudah ditandai "+
				"terkirim oleh pihak lain — periksa apakah reasuradur menerima dua surat",
			inboxpladlapredla.ErrSentButNotMarked)
	}

	return hasil, nil
}
