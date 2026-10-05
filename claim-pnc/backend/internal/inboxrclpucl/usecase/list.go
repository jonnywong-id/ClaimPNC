// Package usecase mengorkestrasi modul Inbox RCL/PUCL.
//
// Tiga operasi:
//
//	Metadata     menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List         mengambil isi satu tab
//	DailyReport  mengambil satu halaman laporan harian pada rentang tanggal
//
// Ekspor grid TIDAK menjadi operasi keempat: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go,
// dan DailyReport dipakai dengan pola yang sama.
//
// Tidak ada operasi yang menulis. Layar lama punya dua tindakan yang menulis — mencetak
// surat PUCL/RCL dan mengirim Reminder PUCL — dan keduanya menyentuh tabel yang masih
// dimiliki Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"claim-pnc/internal/inboxrclpucl"
	"claim-pnc/internal/platform/clock"
)

// Service melayani modul Inbox RCL/PUCL.
type Service struct {
	repoSelector inboxrclpucl.RepoSelector
	actions      inboxrclpucl.ClaimActions
	letters      inboxrclpucl.LetterRenderer
	clock        clock.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxrclpucl.RepoSelector

	// Actions adalah pengisi tindakan tulis. Boleh nil.
	//
	// Nil BUKAN kekeliruan tatanan melainkan keadaan yang sah: layanan Pega yang
	// menjalankannya belum dibangun (`permintaan-artefak-pega.md` §12). Permintaan tindakan
	// lalu dijawab ErrPegaServiceUnavailable — bukan panik, dan bukan layar yang gagal
	// digambar.
	Actions inboxrclpucl.ClaimActions

	// Letters membentuk PDF surat RCL/PUCL. Boleh nil.
	//
	// Nil berarti "Download Dokumen" hanya memindahkan klaim antartab tanpa menerbitkan
	// berkas — perilaku modul ini sebelum templat `SuratPUCL` diterima 2026-10-02. Ia tidak
	// menggagalkan tindakannya, karena perpindahan tab itulah yang menghambat petugas.
	Letters inboxrclpucl.LetterRenderer

	// Clock boleh nil; bila nil dipakai jam sistem.
	//
	// Ia dibutuhkan surat, bukan daftar: tanggal dan nomor surat lahir saat tombolnya
	// ditekan. Seam-nya ada supaya keduanya dapat diuji tanpa menebak waktu.
	Clock clock.Clock

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox RCL/PUCL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxrclpucl/usecase: RepoSelector wajib diisi")
	}
	jam := o.Clock
	if jam == nil {
		jam = clock.System{}
	}
	return &Service{
		repoSelector: o.RepoSelector,
		actions:      o.Actions,
		letters:      o.Letters,
		clock:        jam,
		logger:       o.Logger,
	}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah ketiga tab beserta kolomnya.
	Tabs []inboxrclpucl.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// ReportColumns adalah kolom berkas laporan harian.
	//
	// Ia dikirim ke layar meski layar tidak menggambarnya sebagai tabel: yang membacanya
	// adalah keterangan di dekat tombol unduh, supaya pengguna tahu isi berkasnya berbeda
	// dari tabel yang sedang dilihatnya SEBELUM mengunduh — bukan setelah membukanya.
	ReportColumns []inboxrclpucl.Column

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	//
	// Tiap butir membawa ringkasan satu kalimat DAN rinciannya. Layar menggambar
	// ringkasannya, dan membuka rinciannya hanya bila diminta — lihat
	// `inboxrclpucl.Difference`.
	PlannedDifferences []inboxrclpucl.Difference
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	columns := make([]inboxrclpucl.Column, len(inboxrclpucl.DailyReportColumns))
	copy(columns, inboxrclpucl.DailyReportColumns)

	// Salinan, bukan senarai aslinya — sama alasannya dengan Tabs(): pemanggil tidak boleh
	// dapat mengubah daftar selisih dengan menulisi hasilnya.
	differences := make([]inboxrclpucl.Difference, len(inboxrclpucl.PlannedDifferences))
	copy(differences, inboxrclpucl.PlannedDifferences)

	return Metadata{
		Tabs:               inboxrclpucl.Tabs(),
		DefaultTab:         inboxrclpucl.DefaultTab,
		ReportColumns:      columns,
		PlannedDifferences: differences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxrclpucl.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim: tab
	// yang diminta kosong menjadi tab bawaan, dan layar harus tahu tab mana yang
	// sebenarnya dijawab.
	Query inboxrclpucl.Query
}

// List mengambil isi satu tab.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	input inboxrclpucl.QueryInput,
	page inboxrclpucl.Pagination,
) (Listed, error) {
	query, err := inboxrclpucl.NewQuery(input, caller)
	if err != nil {
		return Listed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf("mengambil isi tab %s: %w", query.Tab.Code, err)
	}

	// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan.
	//
	// Antrean layar ini BERSAMA: penyaringnya akun `RCLPUCL`, bukan pengguna yang login,
	// sehingga setiap petugas melihat daftar yang sama dan seluruh barisnya memuat nomor
	// polis serta nama tertanggung milik pekerjaan bersama.
	//
	// Selama pemeriksaan peran belum ada (`TKT-F3-004`), jejak inilah satu-satunya hal yang
	// menyatakan siapa yang membukanya — dan `D-59` menjadikan jejak audit satu-satunya
	// kontrol pengimbang justru untuk keadaan seperti ini.
	if s.logger != nil {
		s.logger.Info(
			"antrean RCL/PUCL dibuka",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("tab", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}

// Detail mengambil isi layar kerja RCL/PUCL untuk satu klaim.
//
// # Kenapa ia TIDAK memeriksa apakah klaimnya ada di antrean pemanggil
//
// Karena layar kerja dibuka dengan KUNCI, bukan lewat antrean — begitu pula di Pega, tempat
// tautannya mengirim `inskey` dan tidak satu pun penyaring antrean ikut. Klaim yang sudah
// berpindah antrean sejak daftarnya dimuat tetap harus dapat dibuka; menolaknya akan
// membuat layar gagal justru pada keadaan yang paling sering terjadi.
//
// Batas yang tetap berlaku adalah PORTAL: kuncinya dicari di basis data entitas yang sedang
// dipilih, dan kunci milik entitas lain menghasilkan "tidak ditemukan" — bukan diam-diam
// dilayani koneksi lain (`R-20`).
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	reference string,
) (inboxrclpucl.ClaimDetail, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return inboxrclpucl.ClaimDetail{}, inboxrclpucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(reference)
	if key == "" {
		return inboxrclpucl.ClaimDetail{}, inboxrclpucl.NewValidationError(
			[]inboxrclpucl.Violation{{
				Field:   inboxrclpucl.FieldReference,
				Message: "Kunci klaim tidak disebutkan.",
			}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxrclpucl.ClaimDetail{}, err
	}

	detail, err := repo.Detail(ctx, key)
	if err != nil {
		// "Tidak ditemukan" diteruskan APA ADANYA, tidak dibungkus: lapisan transport
		// memetakannya ke 404 dengan keterangan yang menyebut portal, dan pembungkusan
		// akan membuat `errors.Is` di sana gagal mengenalinya.
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
			return inboxrclpucl.ClaimDetail{}, err
		}
		return inboxrclpucl.ClaimDetail{}, fmt.Errorf(
			"mengambil layar kerja klaim %s: %w", key, err)
	}

	// Pembukaan satu klaim dicatat TERPISAH dari pembukaan daftar, dan kuncinya ikut.
	//
	// Alasannya bukan kelengkapan: daftar menampilkan ringkasan, sementara layar ini
	// menampilkan bahan surat berisi nama peserta dan jumlah tagihan satu klaim tertentu.
	// Yang harus dapat ditelusuri adalah KLAIM MANA yang dibuka, bukan sekadar bahwa
	// seseorang membuka layarnya.
	if s.logger != nil {
		s.logger.Info(
			"layar kerja RCL/PUCL dibuka",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.String("klaim", detail.ClaimNumber),
		)
	}

	return detail, nil
}

// Reported adalah satu halaman laporan harian beserta permintaan yang dipakai.
type Reported struct {
	Rows []inboxrclpucl.DailyReportRow

	// Total adalah jumlah SELURUH baris laporan yang cocok, bukan yang ada di halaman ini.
	Total int

	// Pagination adalah paginasi yang benar-benar dipakai setelah dibetulkan.
	Pagination inboxrclpucl.Pagination

	// Request adalah permintaan setelah divalidasi.
	Request inboxrclpucl.ReportRequest
}

// DailyReport mengambil satu halaman laporan harian.
//
// # Kenapa ia operasi tersendiri, bukan penyaring pada List
//
// Karena isinya memang bukan isi tabel. Kuerinya berbeda, kolomnya berbeda, dan
// penyaringnya jauh lebih longgar — ia bahkan memuat klaim yang sudah selesai dan klaim
// Personal Accident yang tidak pernah masuk antrean RCL/PUCL. Lihat
// inboxrclpucl.DailyReportRow.
//
// Menjadikannya penyaring pada List akan membuat layar mengira ia dapat menampilkan
// hasilnya di tabel yang sama, padahal kolomnya pun tidak sama.
func (s *Service) DailyReport(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	input inboxrclpucl.ReportInput,
	page inboxrclpucl.Pagination,
) (Reported, error) {
	request, err := inboxrclpucl.NewReportRequest(input, caller)
	if err != nil {
		return Reported{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Reported{}, err
	}

	clean := page.Normalize()
	rows, total, err := repo.DailyReport(ctx, request.Range, clean)
	if err != nil {
		return Reported{}, fmt.Errorf(
			"mengambil laporan harian %s..%s: %w",
			request.Range.From, request.Range.To, err)
	}

	// Laporan dicatat TERPISAH dari pembukaan tab, dan rentang tanggalnya ikut.
	//
	// Alasannya bukan kelengkapan: berkas laporan dapat diunduh dan dibawa keluar, dan ia
	// memuat nomor polis serta nama tertanggung dalam jumlah yang ditentukan penggunanya
	// sendiri lewat rentang tanggal. Rentang yang lebar adalah hal yang harus dapat
	// ditelusuri setelahnya.
	if s.logger != nil {
		s.logger.Info(
			"laporan harian RCL/PUCL diminta",
			slog.String("modul", "inbox-rcl-pucl"),
			slog.String("pemanggil", request.Caller.Login),
			slog.String("portal", portalAlias),
			slog.String("dari", request.Range.From),
			slog.String("sampai", request.Range.To),
			slog.Int("baris", len(rows)),
			slog.Int("total", total),
		)
	}

	return Reported{
		Rows:       rows,
		Total:      total,
		Pagination: clean,
		Request:    request,
	}, nil
}

// Documents mengembalikan dokumen yang terlampir pada satu klaim.
//
// Ia TIDAK mencatat ke log, berbeda dari Detail. Alasannya: yang bernilai ditelusuri adalah
// siapa membuka ISI dokumen, bukan siapa melihat daftar namanya — dan itu dicatat
// DocumentContent. Mencatat keduanya menggandakan baris tanpa menambah jawaban.
func (s *Service) Documents(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	caseNumber string,
) ([]inboxrclpucl.Document, error) {
	repo, key, err := s.documentAccess(portalAlias, caller, caseNumber)
	if err != nil {
		return nil, err
	}

	documents, err := repo.Documents(ctx, key)
	if err != nil {
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("mengambil daftar dokumen klaim %s: %w", key, err)
	}
	return documents, nil
}

// DocumentContent mengembalikan isi satu dokumen.
func (s *Service) DocumentContent(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	caseNumber, documentID string,
) (inboxrclpucl.DocumentContent, error) {
	repo, key, err := s.documentAccess(portalAlias, caller, caseNumber)
	if err != nil {
		return inboxrclpucl.DocumentContent{}, err
	}

	id := strings.TrimSpace(documentID)
	if id == "" {
		return inboxrclpucl.DocumentContent{}, inboxrclpucl.NewValidationError(
			[]inboxrclpucl.Violation{{
				Field:   inboxrclpucl.FieldReference,
				Message: "Dokumen yang diminta tidak disebutkan.",
			}})
	}

	document, err := repo.DocumentContent(ctx, key, id)
	if err != nil {
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) ||
			errors.Is(err, inboxrclpucl.ErrDocumentNotFound) {
			return inboxrclpucl.DocumentContent{}, err
		}
		return inboxrclpucl.DocumentContent{}, fmt.Errorf(
			"mengambil isi dokumen %s klaim %s: %w", id, key, err)
	}

	// Pengambilan ISI dokumen dicatat, dan kuncinya ikut.
	//
	// Alasannya sama dengan pencatatan Detail, dan lebih kuat: yang berpindah tangan di sini
	// adalah BERKAS milik nasabah — kuitansi, surat keterangan, kadang dokumen medis pada
	// lini PA. Yang harus dapat ditelusuri adalah siapa mengambil dokumen mana, bukan
	// sekadar bahwa seseorang membuka layarnya.
	if s.logger != nil {
		s.logger.Info("dokumen klaim RCL/PUCL diambil",
			"portal", portalAlias,
			"login", caller.Clean().Login,
			"klaim", key,
			"dokumen", id,
		)
	}

	return document, nil
}

// documentAccess memeriksa pemanggil dan nomor case, lalu memilih penyimpanan portalnya.
//
// Ia dipisah karena KEDUA rute dokumen memerlukan pemeriksaan yang sama persis, dan
// menyalinnya dua kali membuat salah satu dapat tertinggal saat yang lain diubah — tepat pada
// pemeriksaan yang memutuskan siapa boleh mengambil berkas milik nasabah.
func (s *Service) documentAccess(
	portalAlias string,
	caller inboxrclpucl.Caller,
	caseNumber string,
) (inboxrclpucl.Repo, string, error) {
	if caller.Clean().Login == "" {
		return nil, "", inboxrclpucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(caseNumber)
	if key == "" {
		return nil, "", inboxrclpucl.NewValidationError(
			[]inboxrclpucl.Violation{{
				Field:   inboxrclpucl.FieldReference,
				Message: "Kunci klaim tidak disebutkan.",
			}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, "", err
	}
	return repo, key, nil
}

// PerformAction menjalankan satu tindakan pada satu klaim.
//
// # Urutannya: baca dulu, baru kirim
//
// Klaimnya dibaca lebih dulu karena tiga hal yang dibutuhkan tindakan ini hanya ada di sana —
// ketiga parameter tersembunyi, catatan untuk Analyst, dan syarat apakah tombolnya memang
// digambar. Mengirimkannya dari layar akan membuat sisi peladen memercayai nilai yang dapat
// disusun siapa pun.
func (s *Service) PerformAction(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	reference string,
	kind inboxrclpucl.ClaimActionKind,
	input inboxrclpucl.ReceiptInput,
) (*inboxrclpucl.Document, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return nil, inboxrclpucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(reference)
	if key == "" {
		return nil, inboxrclpucl.NewValidationError([]inboxrclpucl.Violation{{
			Field:   inboxrclpucl.FieldReference,
			Message: "Kunci klaim tidak disebutkan.",
		}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	detail, err := repo.Detail(ctx, key)
	if err != nil {
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("membaca klaim %s sebelum tindakan %s: %w", key, kind, err)
	}

	// Syaratnya diturunkan dari tombolnya, bukan ditulis ulang — lihat ClaimDetail.Allows.
	if !detail.Allows(kind) {
		return nil, inboxrclpucl.ErrActionNotAvailable
	}

	// KEDUA tombol Kirim ditangani SENDIRI, tanpa menunggu layanan Pega.
	//
	// Yang keduanya lakukan adalah menandai klaim selesai dikerjakan PUCL, dan penandanya
	// hidup di `TC_PNC_PUCL` — tabel milik aplikasi ini. Klaimnya kembali kepada PIC Teknik
	// yang SUDAH memegang baris penugasannya; lihat Repo.ReturnToAnalyst.
	//
	// Ketiga tindakan lain tetap menempuh Pega, dan bukan karena kehati-hatian melainkan
	// karena isinya: "Download Dokumen" membuat PDF dan mengirim email berlampiran, "Tolak
	// Klaim" dan "Save" menyentuh isian yang tidak dibaca layar ini.
	// "Save" ditangani SENDIRI pula, dan ia menyimpan apa yang DIKETIK petugas.
	//
	// Di layar lama tombolnya menempuh `SaveInputRegisterDetail2`, yang berakhir pada
	// `Obj-Save` — dan `Obj-Save` menyimpan seluruh objek kerja, termasuk isian yang baru saja
	// diposkan form. Activity itu sendiri TIDAK menyebut satu pun isian Penerimaan Dokumen;
	// yang disebutnya adalah Pengkinian Data (KTP, email, telepon) dan `RemarkRecommendation`.
	//
	// Jadi yang perlu ditiru bukan activity-nya melainkan AKIBATNYA: kedua isian wajib
	// tersimpan. Lihat inboxrclpucl.ReceiptInput.
	if kind == inboxrclpucl.ActionSave {
		if err := repo.SaveReceipt(ctx, key, input, cleanCaller.Login); err != nil {
			if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
				return nil, err
			}
			var invalid *inboxrclpucl.ValidationError
			if errors.As(err, &invalid) {
				return nil, err
			}
			return nil, fmt.Errorf("menyimpan isian klaim %s: %w", key, err)
		}
		s.logAction(portalAlias, cleanCaller.Login, detail.ClaimNumber, kind)
		return nil, nil
	}

	// "Download Dokumen" — menandai surat tercetak, lalu MENERBITKAN suratnya.
	//
	// Keduanya dikerjakan sendiri, tanpa menempuh layanan Pega. Yang pertama memindahkan
	// klaim dari tab "Cetak Surat" ke "Kelengkapan Dokumen" — itulah yang menghambat
	// petugas bila tidak dikerjakan. Yang kedua melampirkan PDF suratnya sehingga ia muncul
	// di "Lihat Dokumen", persis seperti `PUCLPost` melampirkannya di Pega.
	//
	// Urutannya penting: penandaan DULU, penerbitan kemudian. Bila penerbitannya gagal,
	// klaimnya tetap berpindah tab dan petugas dapat mencetak ulang — sedangkan urutan
	// terbalik membuat kegagalan penandaan meninggalkan surat tanpa klaim yang berpindah.
	if kind == inboxrclpucl.ActionPrintLetter {
		if err := repo.MarkLetterPrinted(ctx, key, cleanCaller.Login); err != nil {
			if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
				return nil, err
			}
			return nil, fmt.Errorf("menandai surat klaim %s sudah dicetak: %w", key, err)
		}
		s.logAction(portalAlias, cleanCaller.Login, detail.ClaimNumber, kind)
		s.recordHistory(ctx, repo, portalAlias, detail, kind, cleanCaller.Login)

		doc, err := s.issueLetter(ctx, repo, detail, cleanCaller.Login)
		if err != nil {
			// Kegagalan menerbitkan surat TIDAK membatalkan tindakannya. Klaimnya sudah
			// berpindah tab, dan mengembalikan galat di sini akan membuat petugas menekan
			// tombolnya lagi — tanpa akibat, karena penandaannya sudah terjadi.
			s.logLetterFailure(portalAlias, cleanCaller.Login, detail.ClaimNumber, err)
			return nil, nil
		}
		return doc, nil
	}

	// KEDUA tombol Kirim menempuh TIGA langkah, dan URUTANNYA bagian dari kebenarannya.
	//
	// # Urutannya DIBALIK pada 2026-10-05, dan ini sebabnya
	//
	// Sampai saat itu urutannya: simpan isian → tandai selesai di PUCL → pindahkan tahap.
	// Ketiganya pernyataan TERPISAH, masing-masing menutup transaksinya sendiri — dan
	// langkah ketiga dapat MENOLAK. Ketika ia menolak, dua langkah pertama sudah terlanjur
	// tersimpan, sehingga klaimnya:
	//
	//	TC_PNC_PUCL.PUCL_APPROVE = '1'   -> keluar dari SELURUH tab RCL/PUCL
	//	tidak ada tugas Send To Analis   -> belum menjadi pekerjaan siapa pun
	//
	// Klaimnya HILANG DARI SETIAP LAYAR — tepat kegagalan yang dikhawatirkan catatan pada
	// Repo.MoveToSendToAnalyst, hanya saja lubangnya bukan di dalam fungsi itu melainkan di
	// ANTARA ketiga pemanggilan ini. Itu benar-benar terjadi pada klaim `PNCN.26.31`, yang
	// PIC Tekniknya belum ditetapkan sehingga perpindahannya ditolak.
	//
	// Sekarang yang DAPAT MENOLAK dikerjakan lebih dulu:
	//
	//	1. validasi isian   — tanpa menulis apa pun
	//	2. pindahkan tahap  — satu-satunya langkah yang dapat menolak karena keadaan klaim
	//	3. tandai PUCL selesai
	//	4. simpan isian
	//
	// Penolakan pada langkah 2 kini tidak meninggalkan satu pun tulisan, dan klaimnya tetap
	// berada di antrean RCL/PUCL tempat petugas dapat menemukannya kembali.
	//
	// # Yang BELUM dijamin, dan dinyatakan di sini supaya tidak terbaca sebagai jaminan
	//
	// Keempatnya masih pernyataan terpisah. Kegagalan BASIS DATA pada langkah 3 atau 4 —
	// bukan penolakan, melainkan sambungan putus — tetap meninggalkan klaim yang sudah
	// berpindah tetapi penandanya belum dicabut. Akibatnya klaim tergambar di DUA tempat
	// sekaligus, dan itu dipilih dengan sadar: terlihat dua kali dapat diperbaiki, hilang
	// sama sekali tidak. Menutupnya sepenuhnya menuntut keempatnya berada dalam SATU
	// transaksi, dan itu menuntut Repo meneruskan transaksi antar-pemanggilan — perubahan
	// yang menyentuh seluruh antarmukanya.
	//
	// # Kenapa isian tetap disimpan oleh tombol Kirim, bukan hanya oleh "Save"
	//
	// `PUCLPost` langkah 10 menulis `KomentarPUCL` ke baris `AdjustmentList` bersamaan dengan
	// penandaan "Setuju" — artinya catatan yang diketik petugas memang ikut tersimpan oleh
	// tombol Kirim. Dan keduanya `pyRequired` di section, sehingga Finish Assignment di Pega
	// MENOLAK form yang salah satunya kosong.
	//
	// Validasinya dipanggil DI SINI, bukan hanya di dalam Repo.SaveReceipt, justru supaya ia
	// berjalan sebelum satu pun tulisan terjadi. Aturannya tetap hidup di satu tempat —
	// ReceiptInput.Validate — dan SaveReceipt tetap memanggilnya untuk pemanggil lain.
	if kind == inboxrclpucl.ActionSendToAnalyst || kind == inboxrclpucl.ActionSendToPICTeknik {
		if _, err := input.Validate(); err != nil {
			return nil, err
		}

		// Perpindahan tahap — langkah yang BENAR-BENAR memindahkan klaim.
		//
		// Ia meniru `SetTicket(SendtoAnalysator)` pada `PUCLPost` langkah 17, ditambah
		// penutupan tugas lama yang di Pega dikerjakan Finish Assignment. Tanpanya klaim
		// hanya berubah penanda: keluar dari layar ini, tetapi tidak menjadi pekerjaan
		// siapa pun.
		//
		// Kegagalannya DIKEMBALIKAN, berbeda dari riwayat dan surat. Keduanya pelengkap;
		// yang ini tujuan tombolnya. Melaporkan berhasil sementara klaimnya tidak bergerak
		// adalah kegagalan senyap yang sudah pernah terjadi di layar ini.
		if err := repo.MoveToSendToAnalyst(ctx, key, cleanCaller.Login); err != nil {
			// ErrAlreadyWithAnalyst ikut diteruskan APA ADANYA: ia keadaan yang dapat
			// dijelaskan ke petugas, bukan kegagalan teknis. Membungkusnya akan
			// menguburnya menjadi 500 dan menghilangkan sebabnya.
			if errors.Is(err, inboxrclpucl.ErrClaimNotFound) ||
				errors.Is(err, inboxrclpucl.ErrTechnicalPICUnknown) ||
				errors.Is(err, inboxrclpucl.ErrAlreadyWithAnalyst) {
				return nil, err
			}
			return nil, fmt.Errorf(
				"memindahkan klaim %s ke tahap Send To Analis: %w", key, err)
		}

		// Klaimnya SUDAH berpindah. Kedua penulisan berikut melepaskannya dari antrean
		// RCL/PUCL dan menyimpan isian petugas — keduanya menyusul, bukan mendahului.
		if err := repo.ReturnToAnalyst(ctx, key, cleanCaller.Login); err != nil {
			if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
				return nil, err
			}
			return nil, fmt.Errorf("menjalankan tindakan %s pada klaim %s: %w", kind, key, err)
		}

		if err := repo.SaveReceipt(ctx, key, input, cleanCaller.Login); err != nil {
			if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
				return nil, err
			}
			var invalid *inboxrclpucl.ValidationError
			if errors.As(err, &invalid) {
				return nil, err
			}
			return nil, fmt.Errorf("menyimpan isian klaim %s sesudah %s: %w", key, kind, err)
		}

		s.logAction(portalAlias, cleanCaller.Login, detail.ClaimNumber, kind)
		s.recordHistory(ctx, repo, portalAlias, detail, kind, cleanCaller.Login)

		// Surat diterbitkan di sini pula, bukan hanya pada "Download Dokumen".
		//
		// `PUCLPost` langkah 27 memanggil `AttachAsPDFC` berprekondisi `1==1` — SELALU,
		// berapa pun `param.Status`. Nama berkasnya ditentukan langkah 20–24 dari jalur
		// klaimnya (`RCL_PUCL`), bukan dari tombol yang ditekan.
		//
		// `detail` sengaja dipakai apa adanya meski dibaca sebelum penyimpanan di atas:
		// surat ini tidak memuat satu pun dari kedua isian itu — lihat issueLetter.
		doc, err := s.issueLetter(ctx, repo, detail, cleanCaller.Login)
		if err != nil {
			s.logLetterFailure(portalAlias, cleanCaller.Login, detail.ClaimNumber, err)
			return nil, nil
		}
		return doc, nil
	}

	if s.actions == nil {
		return nil, inboxrclpucl.ErrPegaServiceUnavailable
	}

	err = s.actions.Perform(ctx, inboxrclpucl.ClaimActionCommand{
		Kind:         kind,
		CaseNumber:   detail.ClaimNumber,
		IDObject:     detail.ActionParameters.IDObject,
		IDCoverage:   detail.ActionParameters.IDCoverage,
		IDAdjustment: detail.ActionParameters.IDAdjustment,
		Note:         detail.DocumentReceipt.PUCLNote,
		Caller:       cleanCaller.Login,
	})
	if err != nil {
		// Ketidaktersediaan diteruskan APA ADANYA supaya transport dapat mengenalinya dan
		// menjawab dengan kalimat yang menyebut siapa yang harus bertindak.
		if errors.Is(err, inboxrclpucl.ErrPegaServiceUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("menjalankan tindakan %s pada klaim %s: %w", kind, key, err)
	}

	s.logAction(portalAlias, cleanCaller.Login, detail.ClaimNumber, kind)
	return nil, nil
}

// logAction mencatat satu tindakan yang MENGUBAH klaim, beserta pelakunya.
//
// Dipakai KEDUA jalur tulis — yang ditangani sendiri maupun yang menempuh Pega — supaya
// jejaknya tidak bergantung pada jalur mana yang kebetulan dipakai. Jejak ini bukan
// kelengkapan melainkan satu-satunya cara mengetahui siapa mengerjakan apa pada klaim mana:
// `D-59` menjadikan jejak audit kontrol pengimbang tunggal, karena tidak ada pemisahan tugas.
func (s *Service) logAction(portalAlias, login, claimNumber string, kind inboxrclpucl.ClaimActionKind) {
	if s.logger == nil {
		return
	}
	s.logger.Info("tindakan klaim RCL/PUCL dijalankan",
		"portal", portalAlias,
		"login", login,
		"klaim", claimNumber,
		"tindakan", string(kind),
	)
}

// DocumentCategories mengembalikan pilihan kolom "Category" pada dialog unggah.
//
// TIDAK dicatat ke jejak audit, dan itu disengaja: ia tidak menyentuh satu pun data klaim —
// isinya master jenis dokumen yang sama bagi setiap petugas. Mencatatnya akan menenggelamkan
// catatan pembukaan klaim yang justru menjadi kontrol pengimbang `D-59`.
func (s *Service) DocumentCategories(
	ctx context.Context,
	portalAlias string,
) ([]inboxrclpucl.DocumentCategory, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	categories, err := repo.DocumentCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("membaca kategori dokumen: %w", err)
	}
	return categories, nil
}

// issueLetter membentuk PDF surat RCL/PUCL lalu melampirkannya ke klaim.
//
// # Kenapa ia melampirkan, bukan sekadar mengembalikan berkasnya
//
// Karena di Pega surat itu MENJADI LAMPIRAN KLAIM — `PUCLPost` menempuh `AttachAsPDFC` lalu
// `SetUploadDocument`, dan sesudahnya ia terbaca di daftar lampiran. Mengembalikannya ke
// peramban saja akan membuat berkasnya hilang begitu petugas menutup tabnya, dan klaim yang
// suratnya sudah terbit tidak dapat dibedakan dari yang belum.
//
// # Satu selisih yang disengaja: yang senama TIDAK dibuang
//
// `PUCLPost` membuang lampiran bernama sama lebih dulu ("search attachment for delete if
// same name"), sehingga satu klaim hanya punya satu `PUCL.pdf`. Itu TIDAK ditiru: `D-66`
// menetapkan tidak ada penghapusan fisik data bernilai bisnis, dan
// `POOLDATA.DATA_ATTACHFILE` tidak punya kolom penanda hapus — sehingga penghapusan lunak
// pun belum mungkin.
//
// Akibatnya mencetak ulang menambah baris baru, bukan menimpa. Daftarnya terurut terbaru di
// atas, jadi yang berlaku tetap yang teratas.
func (s *Service) issueLetter(
	ctx context.Context,
	repo inboxrclpucl.Repo,
	detail inboxrclpucl.ClaimDetail,
	caller string,
) (*inboxrclpucl.Document, error) {
	if s.letters == nil {
		return nil, errors.New("perender surat tidak dipasang")
	}

	now := s.clock.Now()
	nama := inboxrclpucl.LetterFileNameFor(detail.Letter.Track)

	isi, err := s.letters.Render(inboxrclpucl.LetterDocument{
		LetterDate: inboxrclpucl.NewLetterDate(now),

		// Nomor surat memuat KATEGORI lampirannya, dan di Pega itu `Param.category` —
		// parameter yang dikirim tombolnya. Nilainya tetap `Notification`.
		LetterNumber: inboxrclpucl.NewLetterNumber(now, inboxrclpucl.LetterCategory),

		Recipient: detail.Letter.InsuredParty,

		// Ketiga isian berikut sengaja KOSONG, dan barisnya tetap digambar. Lihat suratpdf.
		RecipientAddress: "",
		ContractNumber:   "",
		BusinessUnit:     "",

		SumInsured:   detail.Letter.SumInsured,
		PolicyNumber: detail.Letter.PolicyNumber,
		InsuredName:  detail.Letter.InsuredName,
		BillAmount:   detail.Letter.BillAmount,
		LossDate:     detail.Letter.LossDate,

		Subject:     detail.Letter.Subject,
		OpeningNote: detail.Letter.OpeningNote,
		BodyNote:    detail.Letter.BodyNote,
		ClosingNote: detail.Letter.ClosingNote,
	})
	if err != nil {
		return nil, fmt.Errorf("membentuk surat klaim %s: %w", detail.ClaimNumber, err)
	}

	doc, err := repo.AddDocument(ctx, detail.Reference, inboxrclpucl.UploadedDocument{
		Name: nama,

		// Akhiran telanjang, bukan jenis media: itulah bentuk yang dipakai seluruh baris
		// `ATTACHMIMETYPE` yang ditulis Pega (§158.2).
		MimeType: "pdf",
		Category: inboxrclpucl.LetterCategory,
		Content:  isi,
	}, caller)
	if err != nil {
		return nil, fmt.Errorf("melampirkan surat klaim %s: %w", detail.ClaimNumber, err)
	}

	if s.logger != nil {
		s.logger.Info("surat rcl/pucl diterbitkan",
			"klaim", detail.ClaimNumber,
			"berkas", nama,
			"dokumen", doc.ID,
			"oleh", caller)
	}
	return &doc, nil
}

// recordHistory menulis satu baris riwayat klaim, meniru `InsertHistoryClaimPNC`.
//
// # Kenapa kegagalannya TIDAK dikembalikan
//
// Karena tindakannya sendiri sudah berhasil dan klaimnya sudah berpindah. Mengembalikan galat
// di sini akan membuat petugas menekan tombolnya lagi — tanpa akibat pada penandaan, tetapi
// menambah satu baris riwayat lagi. Kegagalannya dicatat sebagai peringatan.
//
// # Kenapa tetap dikerjakan meski "hanya" riwayat
//
// `D-59` menetapkan tidak ada pemisahan tugas, sehingga jejak audit menjadi **satu-satunya
// kontrol pengimbang**. Baris ini yang menjawab "siapa meneruskan klaim ini, kapan" pada
// sistem yang tidak mencegah siapa pun melakukannya.
func (s *Service) recordHistory(
	ctx context.Context,
	repo inboxrclpucl.Repo,
	portalAlias string,
	detail inboxrclpucl.ClaimDetail,
	kind inboxrclpucl.ClaimActionKind,
	caller string,
) {
	// Teks kosong berarti tindakan ini memang TIDAK menulis riwayat di Pega — "Tolak Klaim"
	// dan "Save". Lihat inboxrclpucl.HistoryNoteFor.
	note := inboxrclpucl.HistoryNoteFor(kind)
	if note == "" {
		return
	}

	if err := repo.RecordHistory(ctx, detail.Reference, note, caller); err != nil {
		if s.logger != nil {
			s.logger.Warn("riwayat klaim rcl/pucl gagal ditulis",
				"portal", portalAlias,
				"oleh", caller,
				"klaim", detail.ClaimNumber,
				"tindakan", string(kind),
				"sebab", err)
		}
	}
}

// logLetterFailure mencatat surat yang gagal terbit.
//
// Dicatat sebagai peringatan, bukan galat: tindakannya sendiri berhasil, dan klaimnya sudah
// berpindah tab. Yang gagal hanya berkasnya, dan petugas dapat mencetak ulang.
func (s *Service) logLetterFailure(portalAlias, login, claimNumber string, err error) {
	if s.logger == nil {
		return
	}
	s.logger.Warn("surat rcl/pucl gagal diterbitkan",
		"portal", portalAlias,
		"oleh", login,
		"klaim", claimNumber,
		"sebab", err)
}

// UploadDocument melampirkan satu berkas ke klaim// UploadDocument melampirkan satu berkas ke klaim — tombol "Unggah Dokumen".
//
// # Kenapa ia operasi tersendiri, bukan salah satu `aksi`
//
// Karena muatannya berbeda jenis: keempat tindakan lain mengirim JSON kecil, yang ini mengirim
// BERKAS. Memaksanya ke jalur yang sama berarti satu alamat yang menerima dua bentuk badan
// permintaan, dan pemanggil harus menebak mana yang berlaku.
//
// Ia juga satu-satunya yang mengembalikan sesuatu — baris dokumen yang baru tersimpan —
// sehingga layar dapat menggambarnya tanpa menarik ulang seluruh daftar.
func (s *Service) UploadDocument(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	reference string,
	upload inboxrclpucl.UploadedDocument,
) (inboxrclpucl.Document, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return inboxrclpucl.Document{}, inboxrclpucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(reference)
	if key == "" {
		return inboxrclpucl.Document{}, inboxrclpucl.NewValidationError(
			[]inboxrclpucl.Violation{{
				Field:   inboxrclpucl.FieldReference,
				Message: "Kunci klaim tidak disebutkan.",
			}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxrclpucl.Document{}, err
	}

	doc, err := repo.AddDocument(ctx, key, upload, cleanCaller.Login)
	if err != nil {
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
			return inboxrclpucl.Document{}, err
		}
		var invalid *inboxrclpucl.ValidationError
		if errors.As(err, &invalid) {
			return inboxrclpucl.Document{}, err
		}
		return inboxrclpucl.Document{},
			fmt.Errorf("mengunggah dokumen klaim %s: %w", key, err)
	}

	// Unggahan DICATAT bersama nama berkasnya. Ia menambah data pada klaim, dan `D-59`
	// menjadikan jejak audit satu-satunya kontrol pengimbang.
	if s.logger != nil {
		s.logger.Info("dokumen klaim RCL/PUCL diunggah",
			"portal", portalAlias,
			"login", cleanCaller.Login,
			"klaim", key,
			"dokumen", doc.ID,
			"berkas", doc.Name,
			"bita", len(upload.Content),
		)
	}
	return doc, nil
}
