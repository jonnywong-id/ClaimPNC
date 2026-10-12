package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/masterpenolakan"
)

// ServiceKomite adalah pintu masuk seluruh perkara Master Penolakan Komite.
//
// Ia terpisah dari Service meski keduanya berada di paket yang sama, karena keduanya
// memilih penyimpanan yang berbeda (RepoSelector versus RepoSelectorKomite) dan tabelnya
// tidak punya satu pun kolom yang menghubungkannya.
//
// Ia TIDAK memegang Clock: POOLDATA.MST_REJECTED_KOMITE tidak punya kolom waktu maupun
// kolom pelaku sama sekali — hanya IDMASTER dan NOTEMASTER. Menambahkannya menuntut
// perubahan skema, yang menempuh persetujuan Work Owner dan pelaksanaan DBA (`D-63`).
// Akibatnya perubahan pada master ini tidak meninggalkan jejak siapa dan kapan; itu
// keterbatasan yang dicatat, bukan ditambal dengan kolom yang dikarang.
type ServiceKomite struct {
	repoSelector masterpenolakan.RepoSelectorKomite
}

// OptionsKomite adalah bahan pembentuk ServiceKomite.
type OptionsKomite struct {
	// RepoSelector memilih penyimpanan milik satu portal entitas. Wajib.
	RepoSelector masterpenolakan.RepoSelectorKomite
}

// NewServiceKomite membentuk layanan dan menolak bahan yang tidak lengkap.
func NewServiceKomite(o OptionsKomite) (*ServiceKomite, error) {
	if o.RepoSelector == nil {
		return nil, errors.New("masterpenolakan/usecase: RepoSelector komite wajib diisi")
	}
	return &ServiceKomite{repoSelector: o.RepoSelector}, nil
}

// List mengembalikan seluruh penolakan komite milik satu portal.
func (l *ServiceKomite) List(ctx context.Context, portalAlias string) ([]masterpenolakan.CommitteeRejection, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// Get mengembalikan satu penolakan komite milik satu portal.
func (l *ServiceKomite) Get(ctx context.Context, portalAlias, id string) (masterpenolakan.CommitteeRejection, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}
	return repo.Get(ctx, id)
}

// Create menyisipkan satu penolakan komite baru dan mengembalikan baris tersimpannya.
//
// ID tidak diterima dari pemanggil: ia diturunkan dari isi tabel portal yang
// bersangkutan. Nomor urut karena itu berdiri sendiri per entitas — dua portal dapat
// memiliki ID yang sama untuk catatan yang berbeda, persis seperti sistem lama, karena
// setiap entitas punya basis datanya sendiri (ADR-0030).
func (l *ServiceKomite) Create(ctx context.Context, portalAlias string, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}
	return repo.InsertNew(ctx, clean)
}

// Update menyimpan perubahan catatan pada baris yang sudah ada.
//
// ID diambil dari jalur URL, tidak pernah dari badan permintaan — dua sumber untuk satu
// nilai berarti keduanya dapat berbeda, dan yang mana yang menang menjadi pertanyaan yang
// tidak perlu ada.
func (l *ServiceKomite) Update(ctx context.Context, portalAlias, id string, input masterpenolakan.InputKomite) (masterpenolakan.CommitteeRejection, error) {
	repo, err := l.repoSelector(portalAlias)
	if err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}

	clean := input.Clean()
	if err := clean.Check(); err != nil {
		return masterpenolakan.CommitteeRejection{}, err
	}
	return repo.Update(ctx, id, clean)
}

// EnsurePortalReady memeriksa portal dapat dilayani tanpa menyentuh satu baris pun.
func (l *ServiceKomite) EnsurePortalReady(portalAlias string) error {
	if _, err := l.repoSelector(portalAlias); err != nil {
		return fmt.Errorf("masterpenolakan/usecase: portal %q tidak dapat dilayani: %w", portalAlias, err)
	}
	return nil
}
