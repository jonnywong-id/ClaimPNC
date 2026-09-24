// Package usecase mengorkestrasi modul Inbox Komunikasi Cabang.
//
// Empat operasi:
//
//	Metadata   menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List       mengambil isi satu tab beserta kedua pencacahnya
//	Detail     mengambil isi layar "Detail Komunikasi" satu percakapan
//	Export     TIDAK ada di sini — lihat catatan di bawah
//
// Ekspor tidak menjadi operasi kelima: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// Tidak ada operasi yang menulis. Layar lama punya EMPAT tindakan yang menulis — Kirim
// Pesan, Balas, Selesai Komunikasi, dan Tambah — dan seluruhnya menyentuh tabel yang masih
// dimiliki Pega selama masa paralel (`P-1`). Satu di antaranya bahkan tidak dapat
// direplikasi sama sekali: activity di balik tombol "Balas" tidak ada di export mana pun.
//
// # Di mana batas cabang diselesaikan, dan kenapa di sini
//
// Di paket INI, bukan di domain dan bukan di penyimpanan.
//
//   - Bukan di domain, karena penurunannya menyentuh DB Link dan karena itu dapat gagal;
//     lapisan domain tidak boleh mengenal I/O apa pun.
//   - Bukan di penyimpanan, karena keputusan tentang ARTI kegagalannya adalah aturan bisnis
//     — dan aturan yang dititipkan ke dua pengisi seam akan diputuskan dua kali.
//
// Yang ada di domain adalah `ResolveBranch`, yang menerjemahkan HASIL penurunan menjadi
// batas data. Yang ada di sini adalah pemanggilan seam-nya dan penanganan galatnya.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxkomunikasicabang"
)

// Service melayani modul Inbox Komunikasi Cabang.
type Service struct {
	repoSelector inboxkomunikasicabang.RepoSelector
	branch       inboxkomunikasicabang.BranchResolver
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxkomunikasicabang.RepoSelector

	// BranchResolver menerjemahkan login petugas menjadi kode cabangnya.
	//
	// Ia WAJIB: tanpa penerjemah, batas data layar ini tidak dapat ditentukan sama sekali,
	// dan satu-satunya jalan yang tersisa adalah menampilkan percakapan siapa saja.
	// Membiarkannya nil lalu "menanganinya nanti" adalah persis cara batas data menghilang
	// tanpa ada yang menyadarinya.
	BranchResolver inboxkomunikasicabang.BranchResolver

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Komunikasi Cabang.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxkomunikasicabang/usecase: RepoSelector wajib diisi")
	}
	if o.BranchResolver == nil {
		return nil, errors.New("inboxkomunikasicabang/usecase: BranchResolver wajib diisi")
	}
	return &Service{
		repoSelector: o.RepoSelector,
		branch:       o.BranchResolver,
		logger:       o.Logger,
	}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi kotak percakapan.
type Metadata struct {
	// Tabs adalah kedua tab beserta kolomnya.
	Tabs []inboxkomunikasicabang.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// ExportColumns adalah kolom berkas ekspor.
	//
	// Dikirim supaya layar dapat menyebutkan isi berkasnya SEBELUM diunduh — ia memuat tiga
	// kolom yang tidak ada di tabel mana pun, dan perbedaan itu tidak boleh baru ketahuan
	// setelah berkasnya dibuka.
	ExportColumns []inboxkomunikasicabang.Column

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	columns := make([]inboxkomunikasicabang.Column, len(inboxkomunikasicabang.ExportColumns))
	copy(columns, inboxkomunikasicabang.ExportColumns)

	return Metadata{
		Tabs:               inboxkomunikasicabang.Tabs(),
		DefaultTab:         inboxkomunikasicabang.DefaultTab,
		ExportColumns:      columns,
		PlannedDifferences: inboxkomunikasicabang.PlannedDifferences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxkomunikasicabang.Page

	// Summary adalah kedua pencacah di atas grid.
	Summary inboxkomunikasicabang.Summary

	// Query adalah permintaan setelah divalidasi, termasuk batas cabang yang dipakai.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim: tab
	// yang diminta kosong menjadi tab bawaan, dan layar harus tahu tab mana yang sebenarnya
	// dijawab. Batas cabangnya ikut, supaya layar dapat menyatakan petugas sedang melihat
	// percakapan siapa.
	Query inboxkomunikasicabang.Query
}

// List mengambil isi satu tab beserta kedua pencacahnya.
//
// # Kenapa pencacah diambil BERSAMA daftar, bukan lewat permintaan terpisah
//
// Karena keduanya digambar berdampingan di layar lama, dan angka yang berasal dari dua saat
// yang berbeda akan saling bertentangan di mata penggunanya — tabel menampilkan satu baris
// sementara lencananya menyebut nol.
//
// Sistem lama pun mengambil keduanya dalam satu perjalanan:
// `PNCCountKomunikasiCabang_Act` menjalankan kedua pencacah berurutan dalam satu activity.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxkomunikasicabang.Caller,
	input inboxkomunikasicabang.QueryInput,
	page inboxkomunikasicabang.Pagination,
) (Listed, error) {
	cleanCaller := caller.Clean()

	filter, err := s.resolveBranch(ctx, cleanCaller)
	if err != nil {
		return Listed{}, err
	}

	query, err := inboxkomunikasicabang.NewQuery(input, filter, cleanCaller)
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

	summary, err := repo.Summarize(ctx, filter)
	if err != nil {
		return Listed{}, fmt.Errorf("menghitung percakapan: %w", err)
	}

	s.logOpen("kotak komunikasi cabang dibuka", portalAlias, query, len(result.Items))

	return Listed{Page: result, Summary: summary, Query: query}, nil
}

// Detailed adalah isi layar detail beserta batas cabang yang dipakai membacanya.
//
// # Kenapa batas cabangnya ikut dikembalikan
//
// Karena lapisan transport membutuhkannya untuk menyatakan ke pengguna percakapan SIAPA yang
// sedang dilihat — dan meminta ulang lewat ResolveBranchFor berarti menembus DB Link DUA
// KALI untuk satu permintaan.
//
// Itu bukan kehalusan: penerjemahan cabang melewati dua DB Link dengan batas waktu lima
// detik masing-masing, sehingga permintaan yang seharusnya satu perjalanan jaringan menjadi
// dua. Versi pertama modul ini melakukannya, dan diperbaiki sebelum dipakai.
type Detailed struct {
	Detail inboxkomunikasicabang.ConversationDetail

	// Branch adalah batas data yang dipakai, sudah diselesaikan.
	Branch inboxkomunikasicabang.BranchFilter
}

// Detail mengambil isi layar "Detail Komunikasi" satu percakapan.
//
// Batas cabang diselesaikan ulang di sini, tidak diwarisi dari permintaan daftar: layar
// detail dapat dibuka lewat alamatnya langsung, tanpa pernah membuka daftarnya. Batas yang
// hanya berlaku bila daftarnya dibuka lebih dulu bukan batas sama sekali.
func (s *Service) Detail(
	ctx context.Context,
	portalAlias string,
	caller inboxkomunikasicabang.Caller,
	input inboxkomunikasicabang.DetailInput,
) (Detailed, error) {
	cleanCaller := caller.Clean()

	id, err := inboxkomunikasicabang.NewDetailRequest(input, cleanCaller)
	if err != nil {
		return Detailed{}, err
	}

	filter, err := s.resolveBranch(ctx, cleanCaller)
	if err != nil {
		return Detailed{}, err
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Detailed{}, err
	}

	detail, err := repo.Detail(ctx, id, filter)
	if err != nil {
		// "Tidak ditemukan" diteruskan APA ADANYA, tidak dibungkus: lapisan transport
		// memetakannya ke 404 dengan keterangan yang menyebut portal, dan pembungkusan akan
		// membuat `errors.Is` di sana gagal mengenalinya.
		if errors.Is(err, inboxkomunikasicabang.ErrConversationNotFound) {
			return Detailed{}, err
		}
		return Detailed{}, fmt.Errorf("mengambil percakapan %s: %w", id, err)
	}

	// Pembukaan satu percakapan dicatat TERPISAH dari pembukaan daftar, dan nomornya ikut.
	//
	// Alasannya bukan kelengkapan: daftar menampilkan potongan pesan, sementara layar ini
	// menampilkan seluruh utas beserta lampirannya. Yang harus dapat ditelusuri adalah
	// PERCAKAPAN MANA yang dibuka, bukan sekadar bahwa seseorang membuka layarnya.
	if s.logger != nil {
		s.logger.Info(
			"detail komunikasi cabang dibuka",
			slog.String("modul", "inbox-komunikasi-cabang"),
			slog.String("pemanggil", cleanCaller.Login),
			slog.String("portal", portalAlias),
			slog.String("komunikasi", detail.ID),
			slog.Int("pesan", len(detail.Messages)),
			slog.Int("lampiran", len(detail.Attachments)),
			slog.Bool("cabang_terbaca", filter.Resolved),
		)
	}

	return Detailed{Detail: detail, Branch: filter}, nil
}

// resolveBranch menurunkan batas data dari login pemanggil.
//
// # Tiga keadaan, dua jawaban
//
//	cabang ditemukan       -> batas cabangnya
//	tidak terdaftar di HRD -> batas KANTOR PUSAT, Resolved=false   (`P-5`)
//	sumbernya tidak terbaca -> GALAT, layar ditutup
//
// Kedua baris pertama menghasilkan daftar; yang ketiga tidak. Pembedaan itulah yang membuat
// "DB Link mati" tidak terbaca sebagai "Anda petugas kantor pusat" — dua keadaan yang
// tampilannya akan sama persis bila digabungkan, dan hanya satu yang benar.
func (s *Service) resolveBranch(
	ctx context.Context, caller inboxkomunikasicabang.Caller,
) (inboxkomunikasicabang.BranchFilter, error) {
	if caller.Login == "" {
		return inboxkomunikasicabang.BranchFilter{}, inboxkomunikasicabang.ErrCallerUnknown
	}

	code, resolved, err := s.branch.Resolve(ctx, caller.Login)
	if err != nil {
		// Sebab aslinya DIBUNGKUS dengan `%w` supaya log menyebut DB Link atau tabel mana
		// yang gagal, sementara peramban hanya menerima pesan umum.
		return inboxkomunikasicabang.BranchFilter{}, fmt.Errorf("%w: %w",
			inboxkomunikasicabang.ErrBranchUnreadable, err)
	}

	filter := inboxkomunikasicabang.ResolveBranch(code, resolved)

	// Petugas yang cabangnya TIDAK terbaca dicatat tersendiri, setiap kali.
	//
	// Ia sedang melihat percakapan kantor pusat karena sistem lama memperlakukannya begitu
	// (`P-5`), dan itu pelebaran batas data yang tidak menghasilkan satu pun galat. Jejak
	// inilah satu-satunya hal yang dapat menjawab "siapa saja yang terkena" bila kelak
	// keputusan itu ditinjau ulang.
	if !filter.Resolved && s.logger != nil {
		s.logger.Warn(
			"kode cabang tidak terbaca; pemanggil dilayani sebagai kantor pusat",
			slog.String("modul", "inbox-komunikasi-cabang"),
			slog.String("pemanggil", caller.Login),
			slog.String("keputusan", "P-5 2026-09-24"),
		)
	}

	return filter, nil
}

// logOpen mencatat pembukaan daftar.
//
// SETIAP pembukaan dicatat, bukan hanya yang mencurigakan. Isi layar ini adalah percakapan
// antarpetugas tentang klaim yang sedang berjalan, dan selama pemeriksaan peran belum ada
// (`TKT-F3-004`), jejak inilah satu-satunya hal yang menyatakan siapa yang membukanya —
// `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang justru untuk keadaan ini.
func (s *Service) logOpen(
	message, portalAlias string, query inboxkomunikasicabang.Query, rows int,
) {
	if s.logger == nil {
		return
	}
	s.logger.Info(
		message,
		slog.String("modul", "inbox-komunikasi-cabang"),
		slog.String("tab", query.Tab.Code),
		slog.String("pemanggil", query.Caller.Login),
		slog.String("portal", portalAlias),
		slog.String("batas_cabang", query.Branch.Code),
		slog.Bool("kantor_pusat", query.Branch.HeadOffice),
		slog.Bool("cabang_terbaca", query.Branch.Resolved),
		slog.Int("baris", rows),
	)
}
