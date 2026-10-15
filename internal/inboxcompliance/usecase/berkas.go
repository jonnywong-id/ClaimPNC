package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxcompliance"
)

// UploadInput adalah satu berkas yang diunggah petugas.
type UploadInput struct {
	// Reference adalah `PZINSKEY` klaimnya.
	Reference string

	// FileName adalah nama berkas apa adanya dari peramban. Pembersihannya dikerjakan
	// adapter, meniru `@pxReplaceAllViaRegex(param.Filename,"[^a-zA-Z0-9]","")`.
	FileName string

	// Extension adalah ekstensi berkas, untuk pemetaan tipe MIME.
	Extension string

	// Category dan SubCategory mengisi kolom `CATEGORY` dan `SUB_CATEGORY`.
	Category    string
	SubCategory string

	// Note mengisi `ATTACHNOTE`. Boleh kosong.
	Note string

	// Content adalah isi berkas.
	Content []byte
}

// UploadDocument mengunggah satu berkas lalu mencatatnya pada klaim.
//
// # Urutannya: LAYANAN dulu, basis data kemudian
//
// Berkas diunggah ke penyimpanan lebih dulu; barisnya ditulis setelah `ImageID` terbit.
// Urutan itu disengaja dan sama dengan Pega (`InsertDokumenPNC` memanggil REST di langkah
// 18, lalu menulis basis data di langkah 19 dan seterusnya).
//
// Bila yang kedua gagal, yang tertinggal adalah **berkas di penyimpanan tanpa baris** —
// tidak terlihat siapa pun, tidak merusak apa pun, dan dapat diunggah ulang. Kebalikannya
// jauh lebih buruk: baris yang menunjuk berkas yang tidak pernah ada, tampil di grid
// dengan tombol Lihat yang selalu gagal.
func (s *Service) UploadDocument(
	ctx context.Context, portalAlias string, caller Caller, input UploadInput,
) (inboxcompliance.Document, error) {
	if strings.TrimSpace(input.Reference) == "" || strings.TrimSpace(input.FileName) == "" {
		return inboxcompliance.Document{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim atau nama berkas tidak disebutkan.",
			}},
		)
	}
	if len(input.Content) == 0 {
		return inboxcompliance.Document{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Berkas yang diunggah kosong.",
			}},
		)
	}

	if s.storeSelector == nil {
		return inboxcompliance.Document{}, inboxcompliance.ErrDocumentServiceMissing
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxcompliance.Document{}, err
	}

	// Klaimnya WAJIB sedang di antrean Compliance. Tanpa ini, berkas dapat ditempelkan
	// ke klaim mana pun yang nomornya diketahui — termasuk yang sudah selesai.
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return inboxcompliance.Document{}, err
	}
	claim, found, err := repo.FindInQueue(ctx, query, input.Reference)
	if err != nil {
		return inboxcompliance.Document{},
			fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return inboxcompliance.Document{}, inboxcompliance.ErrClaimNotInQueue
	}

	store, err := s.storeSelector(portalAlias)
	if err != nil {
		return inboxcompliance.Document{}, err
	}

	hasil, err := store.Upload(ctx, inboxcompliance.UploadRequest{
		UserInput:    caller.Login,
		ClaimNumber:  claim.CaseID,
		FileName:     input.FileName,
		MimeTypeHint: input.Extension,
		Content:      input.Content,
	})
	if err != nil {
		return inboxcompliance.Document{}, fmt.Errorf("mengunggah berkas: %w", err)
	}

	tersimpan, err := repo.SaveDocument(ctx, inboxcompliance.Document{
		Name:           input.FileName,
		MimeType:       input.Extension,
		Category:       input.Category,
		SubCategory:    input.SubCategory,
		Note:           input.Note,
		StorageID:      hasil.StorageID,
		ClaimReference: input.Reference,
		UploadedBy:     caller.Login,
	})
	if err != nil {
		return inboxcompliance.Document{}, fmt.Errorf("mencatat lampiran klaim: %w", err)
	}

	return tersimpan, nil
}

// DeleteDocument menghapus satu lampiran klaim.
//
// # Urutannya: RIWAYAT, basis data, lalu penyimpanan
//
// Penghapusan di sini FISIK — `D-66` tidak diberlakukan pada `DATA_ATTACHFILE`, atas
// persetujuan Work Owner (2026-10-07). Lihat kueri `delete_attachment` untuk ketiga
// alasannya.
//
// Karena barisnya benar-benar hilang, satu baris riwayat ditulis **lebih dulu**. Itu yang
// menutup kerugian utama hapus fisik: tanpanya, penghapusan lampiran klaim tidak tercatat
// di mana pun — dan `D-59` menjadikan jejak audit satu-satunya kontrol pengimbang karena
// tidak ada pemisahan tugas.
//
// Riwayat ditulis SEBELUM, bukan sesudah. Bila ia ditulis sesudah dan gagal, yang hilang
// adalah jejak penghapusan yang sudah terjadi — dan itu tidak dapat dipulihkan.
func (s *Service) DeleteDocument(
	ctx context.Context, portalAlias string, caller Caller, reference, documentID string,
) error {
	if strings.TrimSpace(reference) == "" || strings.TrimSpace(documentID) == "" {
		return inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim atau dokumen yang dihapus tidak disebutkan.",
			}},
		)
	}

	if s.storeSelector == nil {
		return inboxcompliance.ErrDocumentServiceMissing
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return err
	}

	documents, err := repo.FindDocuments(ctx, reference)
	if err != nil {
		return fmt.Errorf("membaca dokumen klaim: %w", err)
	}

	var target *inboxcompliance.Document
	for i := range documents {
		if documents[i].ID == documentID {
			target = &documents[i]
			break
		}
	}
	if target == nil {
		return inboxcompliance.ErrDocumentNotFound
	}

	// Jejak lebih dulu — lihat catatan di atas.
	if err := repo.AppendHistory(ctx, inboxcompliance.HistoryEntry{
		Reference:  reference,
		RecordedAt: s.clock.Now(),
		Note:       "Hapus dokumen " + target.Name,
		By:         caller.Login,
	}); err != nil {
		return fmt.Errorf("menulis jejak penghapusan dokumen: %w", err)
	}

	terhapus, err := repo.DeleteDocument(ctx, reference, documentID)
	if err != nil {
		return fmt.Errorf("menghapus lampiran klaim: %w", err)
	}
	if !terhapus {
		// Barisnya ada saat dibaca tetapi tidak saat dihapus — berarti ada yang
		// menghapusnya lebih dulu. Bukan galat teknis.
		return inboxcompliance.ErrDocumentNotFound
	}

	store, err := s.storeSelector(portalAlias)
	if err != nil {
		return err
	}

	// Berkasnya dihapus TERAKHIR.
	//
	// Bila ia gagal, yang tertinggal adalah berkas yatim di penyimpanan — tidak terlihat
	// pengguna mana pun, karena barisnya sudah hilang. Kebalikannya lebih buruk: berkas
	// hilang sementara barisnya tetap, menghasilkan tombol Lihat yang selalu gagal.
	//
	// Kegagalannya TIDAK membatalkan penghapusan baris, dan itu disengaja: pengguna
	// sudah tidak melihat dokumennya, dan memunculkannya kembali akan membingungkan.
	// Yang terjadi adalah galat dibawa naik supaya tercatat.
	if err := store.Delete(ctx, inboxcompliance.DeleteRequest{
		ObjectPath: target.StorageID,
	}); err != nil {
		return fmt.Errorf(
			"baris lampiran sudah dihapus, tetapi berkasnya gagal dihapus dari "+
				"penyimpanan: %w", err)
	}

	return nil
}
