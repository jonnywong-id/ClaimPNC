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

	"claim-pnc/internal/inboxpladlapredla"
)

// Service melayani modul Inbox PLA, DLA, Pre DLA.
type Service struct {
	repoSelector inboxpladlapredla.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxpladlapredla.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox PLA, DLA, Pre DLA.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("inboxpladlapredla/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
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
