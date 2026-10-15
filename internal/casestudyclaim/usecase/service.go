// Package usecase mengorkestrasi modul Case Study Claim.
//
// Isinya tiga hal, dan ketiganya alasan ia layak dipisahkan dari transport:
//
//  1. MEMILIH PENYIMPANAN menurut portal sebelum satu baris pun dibaca. Layar ini memuat
//     nama tertanggung pada klaim di atas Rp 5 miliar; salah portal berarti data satu
//     badan hukum tampil di layar badan hukum lain (`R-20`).
//  2. MEMERIKSA rentang periode sebelum kueri dijalankan, sehingga daftar dan unduhan
//     menolak dengan alasan yang sama.
//  3. MENCATAT siapa yang menyimpan catatan telaah. Tabelnya tidak punya kolom pelaku,
//     sehingga log adalah satu-satunya tempat identitas itu tersimpan.
package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/casestudyclaim"
)

// Service melayani layar Case Study Claim pada portal yang sedang dibuka.
type Service struct {
	repos  casestudyclaim.RepoSelector
	logger *slog.Logger
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector WAJIB. Ia yang memilih basis data entitas mana yang dibaca.
	RepoSelector casestudyclaim.RepoSelector

	// Logger WAJIB, dan bukan sekadar kelengkapan.
	//
	// `POOLDATA.T_CLAIM_PNC` tidak punya kolom yang mencatat SIAPA yang mengubah
	// `REMARKRECOMENDATION` maupun kapan. Selama modul jejak audit (`S-5`) belum ada, log
	// aplikasi adalah satu-satunya tempat perubahan itu meninggalkan jejak.
	//
	// Itu BUKAN pengganti jejak audit: `D-28` menuntut setiap perubahan bernilai bisnis
	// tercatat permanen beserta pelakunya, dan `D-59` menjadikan jejak audit satu-satunya
	// kontrol pengimbang karena tidak ada pemisahan tugas. Log dapat berputar, dan
	// retensinya bukan retensi audit. Kekurangan itu dicatat, bukan ditutupi.
	Logger *slog.Logger
}

// NewService membentuk service.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, fmt.Errorf("casestudyclaim/usecase: pemilih repo wajib diisi")
	}
	if o.Logger == nil {
		return nil, fmt.Errorf("casestudyclaim/usecase: logger wajib diisi")
	}
	return &Service{repos: o.RepoSelector, logger: o.Logger}, nil
}

// Query adalah permintaan daftar dari layar.
//
// Ia TIDAK memuat identitas pemanggil, dan itu bukan kelalaian: berbeda dari My Inbox,
// layar ini tidak menyaring menurut siapa yang membukanya. Pega pun tidak — kueri lama
// tidak punya satu pun penyaring operator. Yang membatasi apa yang terlihat adalah portal,
// dan kelak izin menu (`TKT-F3-005`).
type Query struct {
	// PortalAlias adalah entitas yang sedang dibuka, diambil dari header `X-Portal` yang
	// sudah diperiksa middleware. Permintaan tanpa portal DITOLAK, tidak pernah dilayani
	// portal utama sebagai cadangan (`R-20`).
	PortalAlias string

	// FromYear dan ToYear adalah rentang tahun registrasi, empat digit.
	//
	// Tahunnya sudah diturunkan transport dari tanggal yang dipilih pengguna — lihat
	// casestudyclaim.Filter.FromYear soal kenapa hanya tahunnya yang dipakai.
	FromYear string
	ToYear   string

	Business casestudyclaim.BusinessScope
	Status   casestudyclaim.ClaimStatusFilter

	Limit  int
	Offset int
}

// filter menyusun penyaring penyimpanan dari permintaan layar.
//
// Satu tempat, supaya daftar dan unduhan TIDAK dapat menyimpang: keduanya wajib menyaring
// populasi yang sama, kalau tidak berkas yang diunduh berisi klaim yang berbeda dari yang
// baru saja dilihat pengguna — dan tidak ada apa pun di berkas itu yang menandainya.
func (q Query) filter() casestudyclaim.Filter {
	return casestudyclaim.Filter{
		FromYear: q.FromYear,
		ToYear:   q.ToYear,
		Business: q.Business,
		Status:   q.Status,
		Limit:    q.Limit,
		Offset:   q.Offset,
	}
}

// List membaca satu halaman klaim telaah.
func (s *Service) List(ctx context.Context, q Query) (casestudyclaim.Page, error) {
	if err := casestudyclaim.ValidatePeriod(q.FromYear, q.ToYear); err != nil {
		// Dikembalikan APA ADANYA, tidak dibungkus: transport mengenalinya dengan
		// errors.Is dan menjawabnya 422, bukan 500.
		return casestudyclaim.Page{}, err
	}

	repo, err := s.repos(q.PortalAlias)
	if err != nil {
		return casestudyclaim.Page{}, err
	}

	page, err := repo.List(ctx, q.filter())
	if err != nil {
		return casestudyclaim.Page{}, fmt.Errorf("casestudyclaim/usecase: membaca daftar klaim: %w", err)
	}
	return page, nil
}

// Caller adalah identitas petugas yang menyimpan catatan telaah.
//
// Ia TIDAK menentukan apa yang boleh disimpan — layar ini tidak punya batas data per
// pengguna. Ia dibawa semata supaya penyimpanannya tercatat; lihat Options.Logger.
type Caller struct {
	Login string
}

// SaveRemark menyimpan catatan telaah satu klaim.
//
// # Urutannya, dan kenapa begitu
//
//  1. Isian diperiksa lebih dulu, sehingga catatan yang terlalu panjang ditolak sebelum
//     menyentuh basis data — bukan dijawab ORA-12899 yang sampai ke pengguna sebagai
//     "terjadi kesalahan sistem".
//  2. Penyimpanan dipilih menurut portal. Menyimpan ke portal yang salah berarti menulis
//     catatan telaah ke klaim badan hukum lain.
//  3. Baris yang tidak ditemukan dijawab ErrClaimNotFound, bukan galat teknis: baris dapat
//     hilang di antara saat daftar dibaca dan saat Save ditekan.
//
// # Tanpa kunci idempotensi, dan itu tidak berbahaya di sini
//
// `10-API-STRATEGY.md` §7 menuntut idempotensi untuk aksi yang menimbulkan akibat di luar
// sistem. Penyimpanan ini **sudah idempoten dengan sendirinya**: ia menimpa satu kolom
// dengan nilai yang sama. Menekan tombol dua kali menghasilkan keadaan yang persis sama,
// dan tidak ada baris yang berganda.
func (s *Service) SaveRemark(
	ctx context.Context,
	portalAlias string,
	caller Caller,
	claimNumber, remark string,
) error {
	if err := casestudyclaim.ValidateRemark(claimNumber, remark); err != nil {
		return err
	}

	repo, err := s.repos(portalAlias)
	if err != nil {
		return err
	}

	saved, err := repo.SaveRemark(ctx, claimNumber, remark)
	if err != nil {
		return fmt.Errorf("casestudyclaim/usecase: menyimpan catatan telaah: %w", err)
	}
	if !saved {
		return casestudyclaim.ErrClaimNotFound
	}

	// Satu-satunya tempat perubahan ini tercatat beserta pelakunya. ISI catatannya TIDAK
	// ikut dicatat — ia dapat memuat keterangan klaim nasabah, dan log disimpan lebih
	// longgar daripada basis data (`11-CROSSCUTTING.md` §2.4).
	s.logger.Info("catatan telaah Case Study Claim disimpan",
		slog.String("portal", portalAlias),
		slog.String("oleh", caller.Login),
		slog.String("nomor_klaim", claimNumber),
		slog.Bool("dikosongkan", remark == ""),
	)
	return nil
}
