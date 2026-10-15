package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/inboxcompliance"
)

// DocumentOpened adalah tautan siap buka untuk satu dokumen.
type DocumentOpened struct {
	// Document adalah barisnya, supaya layar dapat menyebut nama berkasnya tanpa
	// mencarinya lagi.
	Document inboxcompliance.Document

	// ViewerURL adalah tautan yang DIBUKA pengguna — tautan bertanda tangan yang sudah
	// dibungkus penampil Office Online, persis seperti `GetLinkViewDoc_Act` langkah 22.
	ViewerURL string
}

// OpenDocument menerbitkan tautan baru untuk satu dokumen klaim.
//
// # Kenapa tautannya diterbitkan ulang setiap kali, bukan disimpan
//
// Karena ia berumur terbatas (`Durasi`, bawaan 3600 detik). Menyimpannya di kolom lalu
// memakainya ulang akan menghasilkan tombol yang bekerja satu jam pertama lalu diam-diam
// berhenti bekerja — kelas cacat yang paling sulit dilaporkan pengguna, karena ia
// kadang berhasil.
//
// # Kenapa dokumennya DICARI ULANG, bukan diterima dari layar
//
// Layar mengirim id dokumen; `StorageID` dan nama berkasnya dibaca dari basis data di
// sini. Menerima keduanya dari layar berarti klien dapat meminta tautan untuk berkas
// milik klaim lain hanya dengan menukar isian — dan layanan dokumen tidak memeriksa
// kepemilikan apa pun (`pyUseAuthentication = false`).
//
// Jadi pemeriksaan bahwa dokumen itu memang milik klaim yang sedang dibuka terjadi DI
// SINI, dan tidak ada tempat lain yang melakukannya.
func (s *Service) OpenDocument(
	ctx context.Context, portalAlias string, caller Caller, reference, documentID string,
) (DocumentOpened, error) {
	if reference == "" || documentID == "" {
		return DocumentOpened{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim atau dokumen yang dibuka tidak disebutkan.",
			}},
		)
	}

	if s.linkerSelector == nil {
		return DocumentOpened{}, inboxcompliance.ErrDocumentServiceMissing
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return DocumentOpened{}, err
	}

	documents, err := repo.FindDocuments(ctx, reference)
	if err != nil {
		return DocumentOpened{}, fmt.Errorf("membaca dokumen klaim: %w", err)
	}

	var found *inboxcompliance.Document
	for i := range documents {
		if documents[i].ID == documentID {
			found = &documents[i]
			break
		}
	}
	if found == nil {
		// Dokumen yang BUKAN milik klaim ini tidak dapat dibedakan dari dokumen yang
		// tidak ada — dan itu disengaja. Membedakannya akan memberi tahu pemanggil
		// bahwa sebuah id itu sah, hanya milik klaim lain.
		return DocumentOpened{}, inboxcompliance.ErrDocumentNotFound
	}

	linker, err := s.linkerSelector(portalAlias)
	if err != nil {
		return DocumentOpened{}, err
	}

	link, err := linker.NewLink(ctx, inboxcompliance.LinkRequest{
		UserInput:       caller.Login,
		StorageID:       found.StorageID,
		FileName:        found.Name,
		DurationSeconds: inboxcompliance.DefaultLinkDurationSeconds,
	})
	if err != nil {
		return DocumentOpened{}, fmt.Errorf("menerbitkan tautan dokumen: %w", err)
	}

	return DocumentOpened{Document: *found, ViewerURL: link.ViewerURL()}, nil
}
