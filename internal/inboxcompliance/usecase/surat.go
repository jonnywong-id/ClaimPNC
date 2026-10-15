package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxcompliance"
)

// RejectLetterInput adalah isian form Download Dokumen Reject apa adanya dari layar.
//
// Tanggalnya TEKS, bukan time.Time. Dua alasan: isian "Tanggal Keluar Rawat Inap" tidak
// punya sumber basis data sehingga diketik bebas, dan satu-satunya tujuan nilai ini adalah
// digambar ke surat. Menguraikannya menjadi tanggal lalu menggambarnya kembali hanya
// menambah satu tempat yang dapat menolak masukan yang Pega terima.
type RejectLetterInput struct {
	// Reference adalah `PZINSKEY` klaimnya.
	Reference string

	Recipient     string
	Position      string
	PatientName   string
	IncidentPlace string
	IncidentDate  string
	DischargeDate string
	PaidAmount    string
	PaymentDate   string

	// Reasons adalah grid "Alasan" apa adanya. Penomoran dan pembuangan baris kosong
	// dikerjakan RejectLetter.NumberedReasons, bukan di sini.
	Reasons []string
}

// RejectLetterIssued adalah hasil penerbitan surat.
type RejectLetterIssued struct {
	// Document adalah baris lampiran yang baru tercatat.
	Document inboxcompliance.Document

	// Replaced bernilai true bila surat sebelumnya dengan nama yang sama dihapus.
	//
	// Dibawa keluar supaya layar dapat mengatakan "surat diperbarui" alih-alih "surat
	// dibuat" — dan supaya perilaku ganti-bukan-tambah ini terlihat, bukan tersembunyi.
	Replaced bool
}

// GenerateRejectLetter menerbitkan Surat Penolakan lalu melampirkannya pada klaim.
//
// # Ia bukan unduhan — koreksi atas pembacaan pertama
//
// Tombol "Generate PDF" di Pega tidak mengirim berkas ke peramban. Ia MELAMPIRKAN surat
// ke klaim: `DownloadPDFReject` langkah 8-25 dan `PrintRejectCompliancePDF` langkah 3-19
// menghapus lampiran bernama sama, memasang yang baru, lalu menetapkan kategorinya.
//
// Karena itu alur di sini mengikuti UploadDocument, bukan OpenDocument — suratnya menjadi
// dokumen klaim yang muncul di grid Dokumen dan dapat dibuka ulang kapan saja.
//
// # Mengganti, bukan menumpuk
//
// Surat lama bernama sama DIHAPUS lebih dulu, persis seperti kedua activity itu. Tanpa
// penghapusan, menekan tombol tiga kali meninggalkan tiga surat berbeda nomor pada klaim
// yang sama — dan nomor surat berasal dari menit-detik (lihat NewRejectLetterNumber),
// sehingga ketiganya tampak sah dan tidak ada yang tahu mana yang berlaku.
//
// Penghapusannya memakai DeleteDocument yang sudah ada, bukan jalur sendiri. Itu yang
// menjamin jejak auditnya ikut tertulis — `D-59` menjadikan jejak audit satu-satunya
// kontrol pengimbang.
//
// # Urutan yang dipilih, dan kenapa
//
//  1. klaim harus ada di antrean Compliance
//  2. baca pra-isi untuk nama objek pertanggungan
//  3. bentuk PDF
//  4. hapus surat lama
//  5. unggah dan catat yang baru
//
// PDF dibentuk SEBELUM yang lama dihapus. Bila pembentukannya gagal — isian merusak tata
// letak, misalnya — klaim masih memegang surat lamanya. Urutan sebaliknya meninggalkan
// klaim tanpa surat sama sekali karena kegagalan yang belum tentu permanen.
func (s *Service) GenerateRejectLetter(
	ctx context.Context, portalAlias string, caller Caller, input RejectLetterInput,
) (RejectLetterIssued, error) {
	if strings.TrimSpace(input.Reference) == "" {
		return RejectLetterIssued{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim tidak disebutkan.",
			}},
		)
	}

	if s.letterRenderer == nil {
		return RejectLetterIssued{}, inboxcompliance.ErrLetterRendererMissing
	}
	if s.storeSelector == nil {
		return RejectLetterIssued{}, inboxcompliance.ErrDocumentServiceMissing
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return RejectLetterIssued{}, err
	}

	// Klaimnya WAJIB sedang di antrean Compliance — alasan yang sama dengan UploadDocument:
	// tanpa ini, surat penolakan dapat ditempelkan ke klaim mana pun yang nomornya
	// diketahui, termasuk yang sudah dibayar.
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return RejectLetterIssued{}, err
	}
	claim, found, err := repo.FindInQueue(ctx, query, input.Reference)
	if err != nil {
		return RejectLetterIssued{},
			fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return RejectLetterIssued{}, inboxcompliance.ErrClaimNotInQueue
	}

	prefill, err := repo.FindRejectPrefill(ctx, input.Reference)
	if err != nil {
		return RejectLetterIssued{}, fmt.Errorf("membaca pra-isi surat penolakan: %w", err)
	}

	berkas, err := s.letterRenderer.Render(s.buildRejectLetter(claim, prefill, input))
	if err != nil {
		return RejectLetterIssued{}, fmt.Errorf("membentuk surat penolakan: %w", err)
	}

	diganti, err := s.hapusSuratLama(ctx, portalAlias, caller, input.Reference)
	if err != nil {
		return RejectLetterIssued{}, err
	}

	store, err := s.storeSelector(portalAlias)
	if err != nil {
		return RejectLetterIssued{}, err
	}

	hasil, err := store.Upload(ctx, inboxcompliance.UploadRequest{
		UserInput:    caller.Login,
		ClaimNumber:  claim.CaseID,
		FileName:     inboxcompliance.RejectLetterFileName,
		MimeTypeHint: "pdf",
		Content:      berkas,
	})
	if err != nil {
		return RejectLetterIssued{}, fmt.Errorf("mengunggah surat penolakan: %w", err)
	}

	tersimpan, err := repo.SaveDocument(ctx, inboxcompliance.Document{
		Name:           inboxcompliance.RejectLetterFileName,
		MimeType:       "pdf",
		Category:       inboxcompliance.RejectLetterCategory,
		StorageID:      hasil.StorageID,
		ClaimReference: input.Reference,
		UploadedBy:     caller.Login,
	})
	if err != nil {
		return RejectLetterIssued{}, fmt.Errorf("mencatat surat penolakan: %w", err)
	}

	return RejectLetterIssued{Document: tersimpan, Replaced: diganti}, nil
}

// buildRejectLetter merakit bahan surat dari klaim dan pra-isi.
//
// # Inilah satu-satunya tempat isian form DIBUANG, dan itu disengaja
//
// Parameter `input` sengaja TIDAK dipakai. Kedelapan isian form beserta grid Alasan tidak
// sampai ke surat, karena templat Pega tidak punya merge field untuk satu pun di
// antaranya — baris a sampai e dan daftar alasan tergambar kosong untuk diisi tangan.
// Work Owner menetapkan templat itu diikuti apa adanya (2026-10-08).
//
// Isiannya tetap DITERIMA di API dan tetap ada di layar, karena formnya ada di Pega dan
// `D-13` menetapkan layar mengikuti Pega. Yang tidak terjadi hanyalah pemakaiannya.
//
// Pembuangan itu dikumpulkan di SATU fungsi, bukan dibiarkan sebagai medan yang tidak
// pernah dibaca di `RejectLetterDocument`. Bedanya nyata: medan yang tidak terpakai
// terbaca seperti kelalaian dan mengundang seseorang "memperbaikinya", sedangkan
// parameter yang dibuang di satu tempat beserta alasannya adalah keputusan yang terbaca.
//
// Bila kelak Work Owner memutuskan surat harus memuatnya, yang berubah hanya fungsi ini
// dan perendernya — kontrak API, layar, dan ujinya tidak perlu disentuh.
//
// # Sumber keenam isian yang benar-benar digambar
//
//	dari KLAIM     nama penerima surat (`QQNAME`)
//	dari PRA-ISI   nama objek pertanggungan — baris "Perihal" dan "Lampiran"
//	dari JAM       tanggal dan nomor surat
//
// Baris Perihal memakai nama dari basis data, BUKAN isian "Nama Pasien". Itu bukan
// kelalaian: `RejectPage.AcceptanceStream` di Pega dibaca dari
// `ClaimData.ObjectList(1).ObjectName`, terpisah dari `RejectCompliance.NamaPasien`.
func (s *Service) buildRejectLetter(
	claim inboxcompliance.WorkItem,
	prefill inboxcompliance.RejectPrefill,
	_ RejectLetterInput,
) inboxcompliance.RejectLetterDocument {
	now := s.clock.Now()

	return inboxcompliance.RejectLetterDocument{
		LetterDate:   inboxcompliance.NewRejectLetterDate(now),
		LetterNumber: inboxcompliance.NewRejectLetterNumber(now),

		// `QQNAME` ditaruh pada isian BADAN USAHA, dan yang perorangan dibiarkan kosong.
		//
		// Bukan karena tertanggungnya pasti badan usaha, melainkan karena klaim tidak
		// membawa penandanya: `T_CLAIM_PNC` punya `QQNAME` tanpa kolom yang menyatakan
		// perorangan atau badan usaha — pembedaan itu hidup di snapshot polis
		// (`Policy.CIFData.Customer_C` versus `Customer_P`).
		//
		// Dipilih bentuk badan usaha karena dua hal yang dapat diperiksa: surat ini
		// menyediakan isian "Up" dan "Jabatan", yang hanya bermakna bila penerimanya
		// organisasi; dan "Tembusan" templatnya menyebut nama-nama perusahaan.
		//
		// Ini TETAP dugaan terbaik, bukan fakta, dan karena itu dicatat sebagai pertanyaan
		// terbuka untuk Work Owner. Perbaikannya kecil begitu penandanya tersedia —
		// satu isian berpindah — karena perendernya sudah menangani kedua bentuk.
		RecipientCompany: claim.InsuredName,

		// Alamat tertanggung ada di snapshot polis, tidak di tabel klaim. Tergambar kosong
		// beserta tempatnya; lihat catatan paket rejectpdf.
		RecipientAddress: "",

		SubjectName: prefill.PatientName,
	}
}

// hapusSuratLama membuang surat penolakan sebelumnya bila ada.
//
// Mengembalikan true bila benar-benar ada yang dihapus. Tidak adanya surat lama BUKAN
// galat — penekanan pertama memang tidak punya pendahulu.
func (s *Service) hapusSuratLama(
	ctx context.Context, portalAlias string, caller Caller, reference string,
) (bool, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return false, err
	}

	documents, err := repo.FindDocuments(ctx, reference)
	if err != nil {
		return false, fmt.Errorf("membaca dokumen klaim: %w", err)
	}

	var diganti bool
	for _, d := range documents {
		if d.Name != inboxcompliance.RejectLetterFileName {
			continue
		}
		// Lewat DeleteDocument, bukan repo.DeleteDocument langsung: yang pertama menulis
		// jejak audit dan membuang berkasnya dari penyimpanan, yang kedua hanya membuang
		// barisnya dan meninggalkan berkas yatim.
		if err := s.DeleteDocument(ctx, portalAlias, caller, reference, d.ID); err != nil {
			return false, fmt.Errorf("menghapus surat penolakan sebelumnya: %w", err)
		}
		diganti = true
	}

	return diganti, nil
}
