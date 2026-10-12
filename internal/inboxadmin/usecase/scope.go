package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxadmin"
)

// resolveScope menentukan batas data pemanggil — lihat inboxadmin/scope.go.
//
// Repo yang tidak memenuhi ScopeRepo (tiruan di pengujian) tidak dibatasi. Repo nyata —
// SQL maupun memori — memenuhinya, dan pemenuhannya dijaga saat kompilasi.
func (s *Service) resolveScope(
	ctx context.Context, repo inboxadmin.Repo, caller inboxadmin.Caller, region string,
) (inboxadmin.Scope, inboxadmin.Viewer, error) {
	reader, ok := repo.(inboxadmin.ScopeRepo)
	if !ok {
		return inboxadmin.Scope{}, inboxadmin.Viewer{}, nil
	}

	groups, err := reader.GroupsOf(ctx, caller.Login)
	if err != nil {
		return inboxadmin.Scope{}, inboxadmin.Viewer{}, fmt.Errorf("membaca access group: %w", err)
	}
	manager := inboxadmin.IsManager(groups)

	branch := ""
	if !manager {
		code, found, err := reader.BranchOfLogin(ctx, caller.Login)
		if err != nil {
			// Galat, bukan "tidak dibatasi": cabang yang tidak dapat dibaca karena DB Link
			// mati tidak sama dengan petugas kantor pusat. Membuka seluruh antrean pada
			// kegagalan berarti menampilkan antrean cabang lain tanpa satu pun pesan.
			return inboxadmin.Scope{}, inboxadmin.Viewer{}, fmt.Errorf("membaca cabang petugas: %w", err)
		}
		if found {
			branch = code
		}
	}

	scope, viewer := inboxadmin.ResolveScope(manager, branch, region)
	return scope, viewer, nil
}

// ViewerInfo adalah keterangan batas data pemanggil beserta isi dropdown kanwil.
type ViewerInfo struct {
	Viewer inboxadmin.Viewer

	// Regions terisi hanya bagi manajer — hanya mereka yang dapat memilih kanwil.
	Regions []string
}

// Viewer mengembalikan batas data pemanggil, untuk menggambar layar.
func (s *Service) Viewer(ctx context.Context, portalAlias string, caller inboxadmin.Caller) (ViewerInfo, error) {
	clean := caller.Clean()
	if clean.Login == "" {
		return ViewerInfo{}, inboxadmin.ErrCallerUnknown
	}
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return ViewerInfo{}, err
	}
	_, viewer, err := s.resolveScope(ctx, repo, clean, "")
	if err != nil {
		return ViewerInfo{}, err
	}
	info := ViewerInfo{Viewer: viewer, Regions: []string{}}
	if reader, ok := repo.(inboxadmin.ScopeRepo); ok && viewer.Manager {
		if info.Regions, err = reader.Regions(ctx); err != nil {
			return ViewerInfo{}, fmt.Errorf("membaca daftar kanwil: %w", err)
		}
	}
	return info, nil
}
