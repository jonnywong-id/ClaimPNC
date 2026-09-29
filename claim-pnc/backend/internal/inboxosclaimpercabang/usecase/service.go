// Package usecase mengorkestrasi modul Inbox OS Claim per Cabang.
//
// Satu operasi saja: List — mengambil isi layar untuk cabang pemanggil.
//
// Ekspor TIDAK menjadi operasi kedua. Ia memanggil List berulang kali, halaman demi halaman,
// dan menuliskan hasilnya langsung ke jawaban; menaruhnya di sini akan memaksa seluruh baris
// berkumpul di memori lebih dulu — persis yang dilarang
// `15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 aturan 6. Perakitannya ada di http/export.go.
//
// Tidak ada operasi yang menulis. Layar ini membaca; seluruh tabelnya milik Pega selama masa
// paralel (`P-1`).
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxosclaimpercabang"
)

// Service melayani modul Inbox OS Claim per Cabang.
type Service struct {
	repoSelector inboxosclaimpercabang.RepoSelector
	clock        inboxosclaimpercabang.Clock
	logger       *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	RepoSelector inboxosclaimpercabang.RepoSelector

	// Clock dipakai menghitung kolom Aging. Wajib, dengan alasan yang sama: jam nil akan
	// membuat setiap baris berumur nol hari, dan nol hari adalah angka yang tampak wajar.
	Clock inboxosclaimpercabang.Clock

	// Logger boleh nil; bila nil, peringatannya tidak ditulis dan tidak ada yang gagal
	// karenanya.
	Logger *slog.Logger
}

// NewService membentuk layanan modul Inbox OS Claim per Cabang.
func NewService(o Options) (*Service, error) {
	switch {
	case o.RepoSelector == nil:
		return nil, errors.New("inboxosclaimpercabang/usecase: RepoSelector wajib diisi")
	case o.Clock == nil:
		return nil, errors.New("inboxosclaimpercabang/usecase: Clock wajib diisi")
	}

	return &Service{
		repoSelector: o.RepoSelector,
		clock:        o.Clock,
		logger:       o.Logger,
	}, nil
}

// Listed adalah isi layar beserta cabang yang benar-benar dipakai.
type Listed struct {
	// Page adalah satu halaman baris beserta jumlah seluruh baris yang cocok.
	Page inboxosclaimpercabang.Page

	// Query adalah cabang yang benar-benar dipakai, lengkap dengan namanya.
	//
	// Layar menggambar judulnya dari sini — "( CABANG <nama> )" — bukan dari profil sesi.
	// Nama yang dipakai judul dan kode yang dipakai penyaring harus berasal dari satu
	// jawaban yang sama; mengambil keduanya dari tempat berbeda membuat judul dapat menyebut
	// cabang yang bukan cabang barisnya.
	Query inboxosclaimpercabang.Query

	// PlannedDifferences adalah selisih terhadap Pega yang sudah diputuskan.
	//
	// Ia dikirim ke layar, bukan disimpan sebagai komentar, supaya pengguna yang
	// membandingkan kedua layar berdampingan memperoleh jawaban alih-alih melaporkannya
	// sebagai kerusakan (`D-54`).
	PlannedDifferences []string
}

// List mengambil isi layar untuk cabang pemanggil.
//
// Urutan langkahnya menentukan pesan galat mana yang sampai ke pengguna, jadi ia tetap:
// identitas, lalu cabang, baru isinya. Memeriksa cabang lebih dulu akan menjawab "cabang
// tidak diketahui" kepada pengguna yang sebenarnya sesinya yang belum lengkap.
func (s *Service) List(
	ctx context.Context,
	portalAlias string,
	caller inboxosclaimpercabang.Caller,
	page inboxosclaimpercabang.Pagination,
) (Listed, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return Listed{}, inboxosclaimpercabang.ErrCallerUnknown
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Listed{}, err
	}

	branch, resolved, err := repo.BranchOf(ctx, clean.DetailBranchCode)
	if err != nil {
		return Listed{}, fmt.Errorf("menentukan cabang pemanggil: %w", err)
	}
	if !resolved {
		// Dicatat, bukan hanya ditolak. Ia punya DUA sebab yang berbeda jauh, dan log
		// inilah satu-satunya yang membedakannya: pemanggil tidak membawa kode cabang sama
		// sekali (pengguna non-karyawan), atau kodenya ada tetapi tidak dikenal master
		// cabang — yang kedua itu tanda paling awal bahwa kode dari HCQ berada di ruang
		// kode yang berbeda.
		if s.logger != nil {
			s.logger.Info(
				"layar OS per cabang dibuka tanpa cabang yang dapat ditentukan",
				slog.String("modul", "inbox-os-claim-per-cabang"),
				slog.String("pemanggil", clean.Login),
				slog.String("kode_cabang_detail", clean.DetailBranchCode),
				slog.String("portal", portalAlias),
			)
		}
		return Listed{}, inboxosclaimpercabang.ErrBranchUnknown
	}

	query := inboxosclaimpercabang.Query{Branch: branch}

	result, err := repo.List(ctx, query, page)
	if err != nil {
		return Listed{}, fmt.Errorf(
			"mengambil klaim outstanding cabang %q: %w", branch.Code, err)
	}

	// Aging diisi DI SINI, bukan di dalam kueri. Alasannya di
	// inboxosclaimpercabang.AgingDaysSince: padanan portabel `TRUNC` tidak memangkas jam di
	// Oracle, dan hasilnya menjadi nol hari untuk klaim yang terdaftar kemarin.
	//
	// Urutan barisnya tidak terpengaruh: penyimpanan sudah mengurutkannya menurut tanggal
	// registrasi menaik, yang menghasilkan urutan umur menurun yang sama persis.
	now := s.clock.Now()
	for i := range result.Items {
		result.Items[i].AgingDays = inboxosclaimpercabang.AgingDaysSince(
			result.Items[i].RegisterDate, now)
	}

	return Listed{
		Page:               result,
		Query:              query,
		PlannedDifferences: inboxosclaimpercabang.PlannedDifferences,
	}, nil
}
