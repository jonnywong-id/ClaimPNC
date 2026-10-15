// Package documentlink menghubungkan seam inboxmanager.DocumentLinker ke modul Dokumen
// Penunjang: alamat lampiran dibaca dari metadata GENERAL.T_STORAGE_IMAGE, dan diperpanjang
// lewat layanan penyimpanan bila kosong atau sudah lewat EXPDATE
// (dokumenpenunjang.Service.Tautan).
//
// Modul Inbox Manager tidak mengimpor modul lain secara langsung; adapter inilah satu-satunya
// titik sambungnya — preseden `registrasi/repo/dokumenlink`.
package documentlink

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/dokumenpenunjang"
	dokumenpenunjangusecase "claim-pnc/internal/dokumenpenunjang/usecase"
	"claim-pnc/internal/inboxmanager"
)

// Linker mengisi inboxmanager.DocumentLinker.
type Linker struct {
	service *dokumenpenunjangusecase.Service
}

// New membentuk Linker di atas layanan dokumen penunjang.
func New(service *dokumenpenunjangusecase.Service) *Linker { return &Linker{service: service} }

// Link mengembalikan alamat lampiran yang masih berlaku.
//
// Lampiran yang tidak tercatat dijawab alamat kosong, bukan galat — bagi pemanggil artinya
// sama: tidak ada yang dapat dibuka.
func (l *Linker) Link(ctx context.Context, portal, imageID, by string) (string, error) {
	if l == nil || l.service == nil {
		return "", inboxmanager.ErrDocumentLinkUnavailable
	}
	doc, err := l.service.Tautan(ctx, portal, imageID, by)
	if errors.Is(err, dokumenpenunjang.ErrTidakDitemukan) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("%w: %v", inboxmanager.ErrDocumentLinkUnavailable, err)
	}
	return doc.URL, nil
}

var _ inboxmanager.DocumentLinker = (*Linker)(nil)
