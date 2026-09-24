// Package usecase mengorkestrasi modul Inbox Manager Receive / PUCL.
//
// Dua operasi:
//
//	Metadata  menyerahkan daftar tab, kolomnya, dan selisih terencana yang berlaku
//	List      mengambil isi satu tab
//
// Ekspor TIDAK menjadi operasi ketiga: ia memanggil List berulang kali, halaman demi
// halaman, dan menuliskan hasilnya langsung ke jawaban. Menaruhnya di sini akan memaksa
// seluruh baris berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// Tidak ada operasi yang menulis. Layar ini membaca dua antrean; keduanya masih dimiliki
// Pega selama masa paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// Service melayani modul Inbox Manager Receive / PUCL.
type Service struct {
	repoSelector inboxmanagerreceivepucl.RepoSelector
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxmanagerreceivepucl.RepoSelector

	// Logger boleh nil; bila nil, jejaknya tidak ditulis dan tidak ada yang gagal karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox Manager Receive / PUCL.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New(
			"inboxmanagerreceivepucl/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector, logger: o.Logger}, nil
}

// Metadata adalah keterangan layar yang tidak bergantung isi antrean.
type Metadata struct {
	// Tabs adalah ketiga tab beserta kolomnya.
	Tabs []inboxmanagerreceivepucl.Tab

	// DefaultTab adalah tab yang terbuka pertama kali.
	DefaultTab string

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	//
	// Ia dikirim ke layar, bukan disimpan sebagai komentar, supaya pengguna yang
	// membandingkan kedua layar berdampingan memperoleh jawaban alih-alih melaporkannya
	// sebagai kerusakan. Di layar ini butirnya delapan, dan tiga di antaranya menyangkut
	// isian yang benar-benar berbeda isinya dari Pega — bukan sekadar berbeda tampilannya.
	PlannedDifferences []string
}

// Metadata menyerahkan keterangan layar.
//
// Ia tidak menyentuh basis data sama sekali dan tidak bergantung portal: daftar tab dan
// kolomnya sama di seluruh entitas, karena ia bentuk layar, bukan data entitas.
func (s *Service) Metadata() Metadata {
	return Metadata{
		Tabs:               inboxmanagerreceivepucl.Tabs(),
		DefaultTab:         inboxmanagerreceivepucl.DefaultTab,
		PlannedDifferences: inboxmanagerreceivepucl.PlannedDifferences,
	}
}

// Listed adalah isi satu tab beserta permintaan yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxmanagerreceivepucl.Page

	// Query adalah permintaan setelah divalidasi.
	//
	// Layar menggambar judul dan kolomnya dari sini, bukan dari isian yang ia kirim: tab
	// yang diminta kosong menjadi tab bawaan, dan layar harus tahu tab mana yang
	// sebenarnya dijawab.
	Query inboxmanagerreceivepucl.Query
}

// List mengambil isi satu tab.
//
// Paginasi dikerjakan penyimpanan, bukan di sini: pengisi SQL memotongnya di basis data
// dengan `OFFSET … FETCH NEXT`, dan menariknya ke sini akan memaksa seluruh baris melewati
// memori aplikasi lebih dulu.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxmanagerreceivepucl.Caller,
	input inboxmanagerreceivepucl.QueryInput,
	page inboxmanagerreceivepucl.Pagination,
) (Listed, error) {
	query, err := inboxmanagerreceivepucl.NewQuery(input, caller)
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
	// Layar ini berbeda dari modul inbox lain: tidak satu pun tabnya menyaring menurut
	// pemanggil, sehingga seluruh barisnya memuat nomor polis dan nama tertanggung milik
	// pekerjaan orang lain. Di modul lain yang dicatat hanya saat penyaring kepemilikan
	// DILEPAS; di sini penyaring itu memang tidak pernah ada.
	//
	// Selama pemeriksaan peran belum ada (`TKT-F3-004`), jejak inilah satu-satunya hal yang
	// menyatakan siapa yang membuka pandangan penyelia — dan `D-59` menjadikan jejak audit
	// satu-satunya kontrol pengimbang justru untuk keadaan seperti ini.
	if s.logger != nil {
		s.logger.Info(
			"pandangan penyelia antrean receive/RCL-PUCL dibuka",
			slog.String("modul", "inbox-manager-receive-pucl"),
			slog.String("tab", query.Tab.Code),
			slog.String("pemanggil", query.Caller.Login),
			slog.String("portal", portalAlias),
			slog.Int("baris", len(result.Items)),
		)
	}

	return Listed{Page: result, Query: query}, nil
}
