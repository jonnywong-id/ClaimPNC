package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxreceivetka"
)

// List mengembalikan isi Inbox Receive TKA satu portal.
//
// Portal dipilih LEBIH DULU, sebelum satu baris pun dibaca. Portal yang tidak dapat dilayani
// harus ditolak sebagai penolakan portal — bukan sebagai kegagalan membaca daftar, yang akan
// membuat pengguna menduga pekerjaannya memang habis.
func (s *Service) List(
	ctx context.Context,
	portalAlias, keyword string,
) (inboxreceivetka.Page, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxreceivetka.Page{}, err
	}

	page, err := repo.List(ctx, inboxreceivetka.Filter{Keyword: keyword}.Clean())
	if err != nil {
		return inboxreceivetka.Page{}, fmt.Errorf("membaca inbox receive TKA: %w", err)
	}
	return page, nil
}
