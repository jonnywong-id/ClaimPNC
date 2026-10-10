package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxcompliance"
)

// ListDocumentsInCategory menyerahkan lampiran klaim pada satu kategori.
//
// Melayani tombol **"Lihat dokumen"** pada tab Dokumen. Dibaca SAAT DITEKAN, bukan ikut
// pada pembukaan form: sebagian besar kategori kosong, dan memuat seluruhnya setiap kali
// form dibuka berarti membaca banyak yang tidak pernah dilihat.
//
// `category` kosong berarti SELURUH lampiran klaim. Itu bukan kelonggaran yang tidak
// disengaja — tombol "Ubah Kategori Dok" membutuhkannya untuk menampilkan dokumen yang
// akan dipindahkan, termasuk yang kategorinya belum terdaftar di master.
func (s *Service) ListDocumentsInCategory(
	ctx context.Context, portalAlias, reference, category string,
) ([]inboxcompliance.Document, error) {
	if strings.TrimSpace(reference) == "" {
		return nil, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim tidak disebutkan.",
			}},
		)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return nil, err
	}

	documents, err := repo.FindDocuments(ctx, reference)
	if err != nil {
		return nil, fmt.Errorf("membaca dokumen klaim: %w", err)
	}

	cari := strings.TrimSpace(category)
	if cari == "" {
		return documents, nil
	}

	// Penyaringan di sini, bukan di kueri, dan itu disengaja: kuerinya sudah ada dan
	// sudah teruji, dipakai tiga jalur lain, dan jumlah lampiran satu klaim kecil.
	// Menambah kueri kedua yang hampir sama berarti dua tempat yang dapat menyimpang.
	cocok := []inboxcompliance.Document{}
	for _, d := range documents {
		if d.Category == cari {
			cocok = append(cocok, d)
		}
	}
	return cocok, nil
}

// ChangeDocumentCategory memindahkan satu lampiran ke kategori lain.
//
// Melayani tombol **"Ubah Kategori Dok"**.
//
// # Kenapa jauh lebih pendek daripada Pega
//
// `SetCategoryAttachment` menempuh sembilan belas langkah dengan empat Commit, karena
// model lampiran Pega terpecah di tiga tempat — `Link-Attachment`, halaman `WorkAttach`,
// dan `TEMP_DATA_ATTACHMENT` — dan kategori yang sama harus ditulis ke ketiganya.
//
// Di sini lampirannya satu baris, dan kategorinya satu kolom. Tidak ada yang perlu
// disinkronkan, sehingga tidak ada yang dapat gagal separuh jalan.
//
// # Jejaknya ditulis, dan ditulis LEBIH DULU
//
// Perpindahan kategori mengubah **apa yang dianggap lengkap**: dokumen yang tadinya
// memenuhi satu kategori wajib kini memenuhi kategori lain, dan kategori asalnya kembali
// kosong. Itu perubahan bernilai bisnis, dan `D-59` menjadikan jejak audit satu-satunya
// kontrol pengimbang karena tidak ada pemisahan tugas.
//
// Ditulis sebelum, bukan sesudah — alasan yang sama dengan DeleteDocument: jejak yang
// gagal ditulis setelah perpindahan terjadi tidak dapat dipulihkan.
func (s *Service) ChangeDocumentCategory(
	ctx context.Context, portalAlias string, caller Caller,
	reference, documentID, category string,
) error {
	if strings.TrimSpace(reference) == "" ||
		strings.TrimSpace(documentID) == "" ||
		strings.TrimSpace(category) == "" {
		return inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim, dokumen, atau kategori tujuan tidak disebutkan.",
			}},
		)
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

	// Kategori yang SUDAH sama bukan galat, dan juga bukan pekerjaan.
	//
	// Dijawab berhasil tanpa menulis apa pun: menolaknya akan membingungkan petugas yang
	// menekan tombol dua kali, sedangkan menulis jejak untuk perpindahan yang tidak
	// terjadi mengotori satu-satunya kontrol pengimbang yang kita punya.
	if target.Category == category {
		return nil
	}

	if err := repo.AppendHistory(ctx, inboxcompliance.HistoryEntry{
		Reference:  reference,
		RecordedAt: s.clock.Now(),
		Note: "Ubah kategori dokumen " + target.Name +
			" dari " + kategoriTerbaca(target.Category) + " ke " + category,
		By: caller.Login,
	}); err != nil {
		return fmt.Errorf("menulis jejak perubahan kategori dokumen: %w", err)
	}

	dipindahkan, err := repo.UpdateDocumentCategory(ctx, reference, documentID, category)
	if err != nil {
		return fmt.Errorf("memindahkan kategori dokumen: %w", err)
	}
	if !dipindahkan {
		// Barisnya ada saat dibaca tetapi tidak saat dipindahkan — berarti ada yang
		// menghapusnya lebih dulu. Bukan galat teknis.
		return inboxcompliance.ErrDocumentNotFound
	}

	return nil
}

// kategoriTerbaca menggambar kategori asal pada jejak audit.
//
// Kategori kosong ditulis sebagai "(tanpa kategori)", bukan dibiarkan kosong: jejak yang
// berbunyi "dari  ke 10064" tidak dapat dibaca siapa pun setahun kemudian.
func kategoriTerbaca(kategori string) string {
	if strings.TrimSpace(kategori) == "" {
		return "(tanpa kategori)"
	}
	return kategori
}
