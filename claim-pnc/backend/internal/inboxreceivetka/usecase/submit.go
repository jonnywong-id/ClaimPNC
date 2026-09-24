package usecase

import (
	"context"
	"log/slog"

	"claim-pnc/internal/inboxreceivetka"
	"claim-pnc/internal/platform/logging"
)

// CompletionResult adalah hasil satu penekanan tombol Submit.
//
// Ia membedakan TIGA keadaan yang berbeda akibatnya bagi pengguna, dan ketiganya tidak
// boleh terbaca sama:
//
//	tersimpan + terkirim        pekerjaan selesai seluruhnya
//	tersimpan + tidak dicoba    pemberitahuan memang belum dipasang di lingkungan ini
//	tersimpan + gagal kirim     tanggalnya aman, tetapi ada yang perlu dikabari secara lain
//
// Menyatukan dua yang terakhir akan membuat lingkungan pengembangan yang memang tidak punya
// server surel terus-menerus menampilkan peringatan gagal — dan peringatan yang selalu
// muncul berhenti dibaca.
type CompletionResult struct {
	// Task adalah baris yang barusan terisi, dibaca di dalam transaksi yang sama.
	Task inboxreceivetka.Task

	// NotificationAttempted bernilai false bila Notifier memang tidak dipasang.
	NotificationAttempted bool

	// NotificationSent bernilai true hanya bila surelnya benar-benar terkirim.
	NotificationSent bool
}

// Complete mengisi tanggal kelengkapan dokumen satu klaim TKA.
//
// # Urutan langkahnya, dan kenapa persis begitu
//
//  1. pilih portal                     galat portal harus terbaca sebagai galat portal
//  2. bersihkan dan validasi masukan   tanggal kosong ditolak sebelum menyentuh basis data
//  3. repo.Complete                    SATU transaksi: kunci, periksa, tulis DUA tabel
//  4. kirim pemberitahuan              DI LUAR transaksi
//
// # Langkah 4 berada DI LUAR transaksi, dan itu menyimpang dari sistem lama
//
// `Activity/SubmitTanggalLengkapTKA-Act.xml` memanggil `SendEmailNotification` pada langkah
// ke-8 dan baru `Commit` pada langkah ke-9 — surelnya dikirim sementara transaksi basis
// data masih terbuka.
//
// Itu tidak dibawa, dan alasannya ditulis tegas di Steering:
// `10-API-STRATEGY.md` §8.2 melarang pemanggilan sistem eksternal berada di dalam transaksi
// basis data, karena kegagalan jaringan akan menahan kunci baris. Pada layar ini akibatnya
// nyata: server surel yang menggantung akan menahan kunci pada `T_CLAIM_PNC` — tabel yang
// dibaca 116 rule Pega yang sedang melayani produksi.
//
// # Akibatnya, surel yang gagal TIDAK membatalkan penyimpanan
//
// Ini perbedaan perilaku yang disengaja dan harus dinyatakan, bukan ditemukan. Di sistem
// lama, kegagalan sebelum `Commit` berarti tanggalnya tidak tersimpan sama sekali dan
// pengguna mengulang dari awal. Di sini tanggalnya sudah aman, dan yang gagal hanyalah
// pemberitahuannya — layar mengatakan persis itu.
//
// Pilihan ini berpihak pada pekerjaan pengguna. Kehilangan tanggal yang sudah diketik
// karena server surel sedang mati adalah kerugian yang lebih besar daripada satu surel yang
// harus dikirim ulang, dan kegagalannya tercatat lengkap di log.
func (s *Service) Complete(
	ctx context.Context,
	portalAlias string,
	one inboxreceivetka.Completion,
) (CompletionResult, error) {
	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return CompletionResult{}, err
	}

	cleaned := one.Clean()
	if err := cleaned.Validate(); err != nil {
		// Dikembalikan APA ADANYA, tanpa dibungkus: transport memetakannya ke kode HTTP
		// dengan errors.Is, dan pembungkusan di sini hanya menambah kalimat yang tidak
		// dibaca siapa pun.
		return CompletionResult{}, err
	}

	saved, err := repo.Complete(ctx, cleaned)
	if err != nil {
		return CompletionResult{}, err
	}

	log := logging.From(ctx, s.logger)

	// Peristiwa bisnis dicatat sebagai INFO, dan nomor klaimnya ikut — itulah satu-satunya
	// cara menelusuri "siapa mengisi tanggal apa" sebelum modul Jejak Audit (`S-5`) ada.
	//
	// Yang TIDAK ikut dicatat: nama tertanggung dan nama peserta. Keduanya data nasabah,
	// dan `11-CROSSCUTTING.md` §2.4 melarangnya masuk log.
	log.Info("tanggal kelengkapan dokumen TKA diisi",
		slog.String("modul", "inbox-receive-tka"),
		slog.String("portal", portalAlias),
		slog.String("nomor_klaim", saved.ClaimNumber),
		slog.Time("tanggal_dokumen_lengkap", cleaned.CompletedAt),
	)

	result := CompletionResult{Task: saved}
	if s.notifier == nil {
		return result, nil
	}
	result.NotificationAttempted = true

	notice := inboxreceivetka.Notice{
		ClaimNumber:     saved.ClaimNumber,
		PolicyNumber:    saved.PolicyNumber,
		InsuredName:     saved.InsuredName,
		ParticipantName: saved.ParticipantName,
		DateOfLoss:      saved.DateOfLoss,
		CompletedAt:     cleaned.CompletedAt,
	}
	if err := s.notifier.NotifyDocumentCompleted(ctx, notice); err != nil {
		// WARN, bukan ERROR: penyimpanannya berhasil, dan yang gagal adalah akibat
		// sampingannya. Mencatatnya sebagai ERROR akan menyamakannya dengan kegagalan yang
		// membuat pengguna kehilangan pekerjaan (`11-CROSSCUTTING.md` §2.2).
		log.Warn("pemberitahuan kelengkapan dokumen TKA gagal dikirim",
			slog.String("modul", "inbox-receive-tka"),
			slog.String("nomor_klaim", saved.ClaimNumber),
			slog.String("galat", err.Error()),
		)
		return result, nil
	}

	result.NotificationSent = true
	return result, nil
}
