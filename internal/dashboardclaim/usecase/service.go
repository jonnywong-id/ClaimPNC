// Package usecase mengorkestrasi layar Dashboard Claim.
//
// Isinya satu hal yang tidak boleh hidup di transport maupun di repo: MENJAGA KEEMPAT ANGKA
// DAN ISI TELUSURNYA MENYARING POPULASI YANG SAMA.
//
// Ringkasan dan telusur dijalankan dua permintaan HTTP yang terpisah, atas dua kueri yang
// berbeda, dan salah satunya bahkan dilayani modul lain. Tanpa satu tempat yang menyusun
// penyaringnya, keduanya dapat menyimpang — dan pengguna membaca "247" pada kartu lalu
// menemukan jumlah baris yang lain saat menelusurinya. Kegagalan seperti itu tidak
// menghasilkan galat apa pun.
package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dashboardclaim"
)

// Service membaca ringkasan dan telusur Dashboard Claim pada portal yang sedang dibuka.
type Service struct {
	claims dashboardclaim.RepoSelector
	closed dashboardclaim.ClosedClaimReader

	// assignments OPSIONAL — kosong berarti PEMINDAHAN PIC belum terpasang.
	//
	// Terpisah dari transfers, dan pemisahannya bukan kerapian: yang satu menulis tabel
	// milik aplikasi ini, yang lain menulis tabel milik Pega. Keduanya dapat terpasang
	// sendiri-sendiri, dan hak basis datanya pun diminta terpisah.
	assignments dashboardclaim.AssignmentWriterSelector
	clock       Clock

	// picReaders OPSIONAL — kosong berarti daftar PIC Teknik tidak dapat dibaca.
	//
	// Ia melayani modal Transfer, yang di layar lama berupa DAFTAR petugas, bukan isian
	// bebas (`Section/PNCTransferManagement_sec-Section.xml`).
	picReaders dashboardclaim.TechnicalPICReaderSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan menurut portal entitas.
	RepoSelector dashboardclaim.RepoSelector

	// Assignments memilih penulis PIC Teknik. BOLEH kosong — lihat Service.
	Assignments dashboardclaim.AssignmentWriterSelector

	// Clock wajib bila Assignments diisi.
	//
	// IDs sudah TIDAK ada: pengenal permintaan hanya dibutuhkan antrean, dan antreannya
	// dicabut saat jalur ini dibuat mengikuti Pega.
	Clock Clock

	// ClosedClaim membaca tile CLOSE CLAIM dari modul yang sudah memilikinya.
	//
	// Ia WAJIB diisi. Membiarkannya kosong lalu menjawab nol akan menampilkan "0 klaim
	// tutup" pada layar manajerial — angka yang tidak dapat dibedakan dari keadaan
	// benar-benar kosong oleh siapa pun yang membacanya.
	ClosedClaim dashboardclaim.ClosedClaimReader

	// TechnicalPIC membaca daftar PIC Teknik untuk modal Transfer. BOLEH kosong.
	TechnicalPIC dashboardclaim.TechnicalPICReaderSelector
}

// NewService membentuk service dan menolak Options yang tidak lengkap.
//
// Penolakan terjadi saat aplikasi START, bukan saat pengguna membuka layar: perakitan yang
// kurang harus gagal keras di awal, bukan gagal diam-diam ketika seseorang sedang bekerja
// (`12-CROSSCUTTING` §3.1).
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("dashboardclaim/usecase: RepoSelector wajib diisi")
	}
	if o.ClosedClaim == nil {
		return nil, errors.New("dashboardclaim/usecase: ClosedClaim wajib diisi")
	}
	if o.Assignments != nil {
		// Diperiksa saat START, bukan saat tombolnya ditekan: perakitan yang kurang harus
		// gagal keras di awal, bukan saat seseorang sedang bekerja.
		//
		// IDs tidak lagi dituntut — pengenal permintaan hanya dibutuhkan antrean, dan
		// antreannya dicabut. Clock tetap: hasil pemindahan membawa WAKTU-nya.
		if o.Clock == nil {
			return nil, errors.New("dashboardclaim/usecase: Clock wajib diisi bila Assignments dipasang")
		}
	}
	return &Service{
		claims:      o.RepoSelector,
		closed:      o.ClosedClaim,
		assignments: o.Assignments,
		clock:       o.Clock,
		picReaders:  o.TechnicalPIC,
	}, nil
}

// Query adalah permintaan dari layar.
type Query struct {
	// PortalAlias adalah entitas yang sedang dibuka, diambil dari header `X-Portal` yang
	// sudah diperiksa middleware. Ia menentukan BASIS DATA mana yang dibaca.
	//
	// Permintaan tanpa portal DITOLAK, tidak pernah dilayani portal utama sebagai cadangan
	// (`R-20`).
	PortalAlias string

	Filter dashboardclaim.Filter
}

// CountsResult adalah keempat angka beserta penyaring yang menghasilkannya.
type CountsResult struct {
	Counts dashboardclaim.Counts

	// Filter ikut dikembalikan supaya LAYAR DAPAT MENYATAKANNYA.
	//
	// Keempat angka nol punya dua sebab yang tampak sama di layar: memang tidak ada
	// pekerjaan, atau penyaring lini bisnis mempersempitnya sampai habis. Menyebutkan
	// penyaring yang berlaku membedakan keduanya.
	Filter dashboardclaim.Filter
}

// Counts menghitung keempat kartu.
//
// # Kenapa keempatnya dibaca berurutan, bukan bersamaan
//
// Menjalankannya paralel akan menghemat waktu dinding, tetapi membuka empat koneksi
// sekaligus per pembukaan layar. Pool-nya 20 koneksi per instance (`09-DATABASE-STRATEGY`
// §7), dan layar ini dibuka banyak orang pada jam yang sama — empat kali lipat kebutuhan
// koneksi adalah harga yang tidak sepadan dengan penghematan pada beban serendah ini
// (200–300 pengguna, `D-10`).
//
// Bila pengukuran nyata kelak menunjukkan sebaliknya, perubahannya terbatas di fungsi ini.
func (s *Service) Counts(ctx context.Context, q Query) (CountsResult, error) {
	filter := q.Filter.Normalize().CountFilter()

	// Penyimpanan dipilih menurut portal SEBELUM apa pun dibaca. Galat di sini tidak
	// dialihkan ke koneksi mana pun sebagai cadangan — ia dikembalikan apa adanya.
	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return CountsResult{}, err
	}

	var counts dashboardclaim.Counts

	if counts.Outstanding, err = claims.CountOutstanding(ctx, filter); err != nil {
		return CountsResult{}, fmt.Errorf("dashboardclaim/usecase: menghitung klaim berjalan: %w", err)
	}

	// CLOSE CLAIM dibaca lewat seam, bukan kueri modul ini — lihat catatan paket domain.
	// Galatnya DIKEMBALIKAN, tidak ditelan menjadi nol: angka nol yang sebenarnya kegagalan
	// tidak dapat dibedakan dari nol yang benar.
	if counts.CloseClaim, err = s.closed.Count(ctx, q.PortalAlias, filter); err != nil {
		return CountsResult{}, fmt.Errorf("dashboardclaim/usecase: menghitung klaim tutup: %w", err)
	}

	if counts.LossAdjuster, err = claims.CountSurvey(ctx, dashboardclaim.SurveyorAdjuster, filter); err != nil {
		return CountsResult{}, fmt.Errorf("dashboardclaim/usecase: menghitung survei loss adjuster: %w", err)
	}

	if counts.InternalSurveyor, err = claims.CountSurvey(ctx, dashboardclaim.SurveyorInternal, filter); err != nil {
		return CountsResult{}, fmt.Errorf("dashboardclaim/usecase: menghitung survei internal: %w", err)
	}

	return CountsResult{Counts: counts, Filter: filter}, nil
}

// HoldingResult adalah satu halaman tab Inbox Tampungan PIC.
type HoldingResult struct {
	Page   dashboardclaim.HoldingPage
	Filter dashboardclaim.Filter
}

// Holding membaca satu halaman tab Inbox Tampungan PIC.
//
// Penyaring lini bisnis sengaja DIBUANG sebelum dikirim ke penyimpanan: tab ini tidak
// mengenalnya, dan membiarkannya terbawa akan membuat pengguna yang sebelumnya memilih
// "Personal Accident" pada tab sebelah melihat penampungan yang tampak kosong tanpa sebab
// yang terlihat di layar.
func (s *Service) Holding(ctx context.Context, q Query) (HoldingResult, error) {
	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return HoldingResult{}, err
	}

	filter := q.Filter.Normalize()
	filter.Business = dashboardclaim.BusinessAll

	page, err := claims.ListHolding(ctx, filter)
	if err != nil {
		return HoldingResult{}, fmt.Errorf("dashboardclaim/usecase: membaca daftar klaim tampungan: %w", err)
	}
	return HoldingResult{Page: page, Filter: filter}, nil
}

// ListQuery adalah permintaan telusur satu tile.
type ListQuery struct {
	PortalAlias string
	Tile        dashboardclaim.Tile
	Filter      dashboardclaim.Filter
}

// ListResult adalah satu halaman telusur.
//
// Hanya SATU dari kedua halaman yang terisi, ditentukan Shape. Menyatukannya menjadi satu
// bentuk baris akan memaksa kolom yang tidak dimiliki salah satunya menjadi kosong — dan
// kolom kosong di layar tidak dapat dibedakan dari data yang hilang.
type ListResult struct {
	Tile  dashboardclaim.Tile
	Shape dashboardclaim.RowShape

	Claims  dashboardclaim.ClaimPage
	Surveys dashboardclaim.SurveyPage

	Filter dashboardclaim.Filter
}

// List membaca satu halaman telusur untuk tile yang diminta.
func (s *Service) List(ctx context.Context, q ListQuery) (ListResult, error) {
	if _, known := dashboardclaim.ParseTile(string(q.Tile)); !known {
		return ListResult{}, dashboardclaim.ErrTileNotFound
	}

	filter := q.Filter.Normalize()
	result := ListResult{Tile: q.Tile, Shape: q.Tile.Shape(), Filter: filter}

	// CLOSE CLAIM dilayani seam, dan karena itu TIDAK menyentuh RepoSelector sama sekali.
	// Memilih repo lebih dulu akan menolak permintaan pada portal yang repo-nya belum siap,
	// padahal tile ini tidak membacanya.
	if q.Tile == dashboardclaim.TileCloseClaim {
		page, err := s.closed.List(ctx, q.PortalAlias, filter)
		if err != nil {
			return ListResult{}, fmt.Errorf("dashboardclaim/usecase: membaca daftar klaim tutup: %w", err)
		}
		result.Claims = page
		return result, nil
	}

	claims, err := s.claims(q.PortalAlias)
	if err != nil {
		return ListResult{}, err
	}

	if kind, isSurvey := dashboardclaim.SurveyorTypeFor(q.Tile); isSurvey {
		page, err := claims.ListSurvey(ctx, kind, filter)
		if err != nil {
			return ListResult{}, fmt.Errorf("dashboardclaim/usecase: membaca daftar survei: %w", err)
		}
		result.Surveys = page
		return result, nil
	}

	page, err := claims.ListOutstanding(ctx, filter)
	if err != nil {
		return ListResult{}, fmt.Errorf("dashboardclaim/usecase: membaca daftar klaim berjalan: %w", err)
	}
	result.Claims = page
	return result, nil
}
