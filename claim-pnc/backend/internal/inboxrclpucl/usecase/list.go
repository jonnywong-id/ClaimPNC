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
)

// Service melayani modul Inbox RCL/PUCL.
type Service struct {
	repoSelector inboxrclpucl.RepoSelector
	actions      inboxrclpucl.ClaimActions
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

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox RCL/PUCL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxrclpucl/usecase: RepoSelector wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		actions:      o.Actions,
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

// SendToAnalyst menjalankan tindakan "Kirim Ke Analyst" pada satu klaim.
//
// # Urutannya: baca dulu, baru kirim
//
// Klaimnya dibaca lebih dulu karena tiga hal yang dibutuhkan tindakan ini hanya ada di sana —
// ketiga parameter tersembunyi, catatan untuk Analyst, dan syarat apakah tombolnya memang
// digambar. Mengirimkannya dari layar akan membuat sisi peladen memercayai nilai yang dapat
// disusun siapa pun.
func (s *Service) SendToAnalyst(
	ctx context.Context,
	portalAlias string,
	caller inboxrclpucl.Caller,
	reference string,
) error {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return inboxrclpucl.ErrCallerUnknown
	}

	key := strings.TrimSpace(reference)
	if key == "" {
		return inboxrclpucl.NewValidationError([]inboxrclpucl.Violation{{
			Field:   inboxrclpucl.FieldReference,
			Message: "Kunci klaim tidak disebutkan.",
		}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	detail, err := repo.Detail(ctx, key)
	if err != nil {
		if errors.Is(err, inboxrclpucl.ErrClaimNotFound) {
			return err
		}
		return fmt.Errorf("membaca klaim %s sebelum kirim ke analyst: %w", key, err)
	}

	// Syaratnya diturunkan dari tombolnya, bukan ditulis ulang — lihat
	// `inboxrclpucl.ClaimDetail.CanSendToAnalyst`.
	if !detail.CanSendToAnalyst() {
		return inboxrclpucl.ErrActionNotAvailable
	}

	if s.actions == nil {
		return inboxrclpucl.ErrPegaServiceUnavailable
	}

	err = s.actions.SendToAnalyst(ctx, inboxrclpucl.SendToAnalystCommand{
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
			return err
		}
		return fmt.Errorf("mengirim klaim %s ke analyst: %w", key, err)
	}

	// Tindakan yang MENGUBAH klaim dicatat, dan pelakunya ikut.
	//
	// Ia satu-satunya jalur tulis modul ini. Jejaknya karena itu bukan kelengkapan melainkan
	// satu-satunya cara mengetahui siapa meneruskan klaim mana — `D-59` menjadikan jejak audit
	// kontrol pengimbang tunggal, karena tidak ada pemisahan tugas.
	if s.logger != nil {
		s.logger.Info("klaim RCL/PUCL dikirim ke Analyst",
			"portal", portalAlias,
			"login", cleanCaller.Login,
			"klaim", detail.ClaimNumber,
		)
	}
	return nil
}
