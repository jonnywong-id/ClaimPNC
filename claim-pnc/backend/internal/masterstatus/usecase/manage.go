// Package usecase mengorkestrasi pengelolaan Master Status Klaim: melihat daftar,
// membuka satu baris, menambah, dan mengubah.
//
// Lapisan orkestrasi: ia memanggil seam yang dideklarasikan paket masterstatus dan
// tidak tahu apa pun soal HTTP maupun SQL.
//
// # Empat aksi, dan satu yang sengaja tidak ada
//
// Tidak ada Hapus. Itu bukan kelalaian: layar Pega pun tidak punya tombol hapus
// (`Section/BrowseStatusClaim-Section.xml` — `pyDeleteActivityExists=false`), dan
// `Database/PEGA_M_STS_CLAIM.prc` hanya mengenal INSERT dan UPDATE. Menghapus satu
// baris master akan membuat setiap klaim lama yang menyimpan kode itu kehilangan
// artinya — persis alasan `ADR-0012` menetapkan master tidak dihapus permanen.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/masterstatus"
)

// Service mengelola master status klaim di atas seam penyimpanan PER PORTAL.
type Service struct {
	repoSelector masterstatus.RepoSelector
}

// Options adalah bahan pembentuk Service.
type Options struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterstatus.RepoSelector
}

// NewService membentuk Service dan menolak bahan yang tidak lengkap — kegagalannya
// terjadi saat start, bukan saat pengguna pertama membuka layar.
func NewService(o Options) (*Service, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterstatus/usecase: RepoSelector wajib diisi")
	}
	return &Service{repoSelector: o.RepoSelector}, nil
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
func (l *Service) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterstatus/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}

// Daftar mengembalikan seluruh status klaim, terurut menurut kode.
//
// Tidak dipaginasi, dan itu keputusan yang diambil dengan angka: isinya 33 baris dan
// bertambah beberapa baris per tahun. Report Definition lama pun memuat seluruhnya
// sekaligus dengan batas `pyMaxRecords=500`. Menambahkan paginasi server di sini akan
// menambah kerumitan yang tidak menyelesaikan satu pun masalah nyata; penyaringan dan
// pengurutan cukup dikerjakan di layar atas 33 baris yang sudah di tangan.
func (l *Service) List(ctx context.Context, portalAlias string) ([]masterstatus.ClaimStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	list, err := repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("masterstatus/usecase: membaca daftar status: %w", err)
	}
	return list, nil
}

// Ambil mengembalikan satu status klaim.
//
// Ia menggantikan `SetStsClaimValue_act(lscid)`, yang menjalankan Report Definition
// `SelectVStsClaim_RD` lalu menyalin hasilnya ke halaman `TempStsClaim`.
func (l *Service) Get(ctx context.Context, portalAlias, code string) (masterstatus.ClaimStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	status, err := repo.Get(ctx, trim(code))
	if err != nil {
		if errors.Is(err, masterstatus.ErrNotFound) {
			return masterstatus.ClaimStatus{}, err
		}
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/usecase: membaca status %q: %w", code, err)
	}
	return status, nil
}

// Tambah menyisipkan status baru dan mengembalikannya lengkap dengan kode yang dibuat
// penyimpanan.
//
// Kode TIDAK diterima dari pemanggil. Sistem lama pun demikian: layar mengirim sentinel
// `"UnknownID"` dan procedure yang menentukan kodenya
// (`Activity/CNMUpdateStsclaim_act-Act.xml` → `PEGA_M_STS_CLAIM`). Menerima kode dari
// luar akan membuat dua status berbeda dapat memperebutkan nomor yang sama.
func (l *Service) Create(ctx context.Context, portalAlias, label string) (masterstatus.ClaimStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	if err := masterstatus.NewValidationError(masterstatus.CheckLabel(label)); err != nil {
		return masterstatus.ClaimStatus{}, err
	}
	if err := ensureLabelNotTaken(ctx, repo, label, ""); err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	status, err := repo.Insert(ctx, trim(label))
	if err != nil {
		// Kedua galat di bawah datang dari penegakan di basis data, yang menang atas
		// pemeriksaan di atas bila dua permintaan tiba bersamaan. Ia diteruskan apa
		// adanya supaya pengguna melihat sebab yang benar.
		if errors.Is(err, masterstatus.ErrLabelTaken) || errors.Is(err, masterstatus.ErrCodeTaken) {
			return masterstatus.ClaimStatus{}, err
		}
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/usecase: menambah status: %w", err)
	}
	return status, nil
}

// Ubah mengganti label status yang sudah ada.
//
// Hanya label yang dapat berubah. Kode bersifat tetap seumur hidup baris itu — layar
// Pega menandainya read-only (`pyEditOptions=Read-only`), dan mengubahnya akan
// memutus setiap klaim lama yang menyimpan kode tersebut.
func (l *Service) Update(ctx context.Context, portalAlias, code, label string) (masterstatus.ClaimStatus, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterstatus.ClaimStatus{}, err
	}
	code = trim(code)

	if err := masterstatus.NewValidationError(masterstatus.CheckLabel(label)); err != nil {
		return masterstatus.ClaimStatus{}, err
	}
	// Kode diperiksa lebih dulu supaya mengubah status yang tidak ada dijawab "tidak
	// ditemukan", bukan "label sudah dipakai" yang menyesatkan.
	if _, err := repo.Get(ctx, code); err != nil {
		if errors.Is(err, masterstatus.ErrNotFound) {
			return masterstatus.ClaimStatus{}, err
		}
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/usecase: membaca status %q: %w", code, err)
	}
	if err := ensureLabelNotTaken(ctx, repo, label, code); err != nil {
		return masterstatus.ClaimStatus{}, err
	}

	status, err := repo.Update(ctx, code, trim(label))
	if err != nil {
		if errors.Is(err, masterstatus.ErrNotFound) || errors.Is(err, masterstatus.ErrLabelTaken) {
			return masterstatus.ClaimStatus{}, err
		}
		return masterstatus.ClaimStatus{}, fmt.Errorf("masterstatus/usecase: mengubah status %q: %w", code, err)
	}
	return status, nil
}

// ensureLabelNotTaken menolak label yang sudah dipakai status lain.
//
// kecualiKode dikosongkan saat menambah, dan diisi kode yang sedang diubah saat
// mengubah — tanpa itu, menyimpan ulang status tanpa mengubah labelnya akan ditolak
// karena bentrok dengan dirinya sendiri.
//
// Pemeriksaan ini adalah KENYAMANAN, bukan jaminan: dua permintaan yang tiba bersamaan
// dapat sama-sama lolos di sini. Jaminannya ada di indeks unik basis data
// (`UX_M_STS_CLAIM_LABEL` pada migrasi 0002), dan galatnya diterjemahkan kembali
// menjadi ErrLabelTaken oleh repo. Yang di sini hanya membuat pesannya tiba lebih
// cepat dan lebih jelas.
func ensureLabelNotTaken(ctx context.Context, repo masterstatus.Repo, label, exceptCode string) error {
	list, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("masterstatus/usecase: memeriksa keunikan label: %w", err)
	}

	wanted := masterstatus.LabelKey(label)
	for _, s := range list {
		if s.Code == exceptCode {
			continue
		}
		if masterstatus.LabelKey(s.Label) == wanted {
			return masterstatus.ErrLabelTaken
		}
	}
	return nil
}

// trim membuang spasi tepi dari masukan pengguna sebelum ia menyentuh penyimpanan.
func trim(s string) string { return strings.TrimSpace(s) }
