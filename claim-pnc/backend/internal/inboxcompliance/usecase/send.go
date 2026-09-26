package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxcompliance"
)

// Caller adalah identitas petugas yang mengirim klaim ke Post Audit.
//
// # Kenapa ia dibawa, padahal tabelnya tidak menyimpannya
//
// `POOLDATA.T_CLAIM_COMPLIANCE_H` tidak punya kolom yang mencatat SIAPA yang mengirim —
// hanya kapan. Identitasnya karena itu tidak dapat disimpan bersama barisnya.
//
// Ia tetap dibawa sampai ke sini supaya pengirimannya tercatat di LOG. Itu bukan pengganti
// jejak audit yang sebenarnya: `D-28` menuntut setiap perubahan bernilai bisnis tercatat
// permanen dengan pelakunya, dan `D-59` menjadikan jejak audit satu-satunya kontrol
// pengimbang karena tidak ada pemisahan tugas. Log aplikasi tidak memenuhi keduanya — ia
// dapat berputar, dan retensinya bukan retensi audit.
//
// Kekurangan itu dicatat sebagai temuan: **tabelnya perlu kolom pengirim**, atau modul
// `S-5` harus sudah ada sebelum jalur ini dipakai di produksi.
type Caller struct {
	// Login adalah nama pengguna yang diketik saat masuk.
	Login string
}

// Sent adalah hasil pengiriman satu klaim ke Post Audit.
type Sent struct {
	// Entry adalah baris yang tersimpan, lengkap dengan nomor yang terbit.
	Entry inboxcompliance.PostAuditEntry

	// Claim adalah klaim yang dikirim, sebagaimana terbaca dari antrean.
	Claim inboxcompliance.WorkItem
}

// SendToPostAudit meneruskan satu klaim dari antrean Compliance ke Post Audit.
//
// # Urutannya, dan kenapa begitu
//
//  1. Klaimnya dicari DI ANTREAN, bukan di seluruh tabel klaim. Pengiriman atas klaim yang
//     tidak sedang diperiksa ditolak — meniru Pega, yang menuntut assignment-nya ada.
//  2. Barisnya disusun dari klaim itu, bukan dari isian layar. Nama tertanggung dan nomor
//     polis karena itu tidak mungkin berbeda dari klaimnya.
//  3. Waktunya diambil dari seam Clock, bukan dari layar. Layar Pega pun menampilkan waktu
//     pengiriman, bukan tanggal yang dipilih.
//
// # Yang BELUM ada, dan disadari
//
// Tidak ada kunci idempotensi. `10-API-STRATEGY.md` §7 menuntutnya untuk aksi yang
// menimbulkan akibat, dan di sini akibatnya nyata: menekan tombol dua kali menghasilkan DUA
// baris Post Audit untuk satu klaim, dan tabelnya tidak punya constraint unik yang akan
// menolaknya.
//
// Yang menahannya hari ini hanyalah layar, yang menonaktifkan tombolnya selama permintaan
// berjalan — penahan yang tidak berlaku bagi permintaan yang datang langsung. Menambahkan
// penyaring "sudah pernah dikirim" akan mengarang aturan yang tidak ada sumbernya: di Pega,
// satu klaim memang dapat punya lebih dari satu pemeriksaan Post Audit, dan `CPL-1` sampai
// `CPL-19` di layar Pega tidak memberi tahu apakah sebagiannya merujuk klaim yang sama.
func (s *Service) SendToPostAudit(
	ctx context.Context,
	portalAlias string,
	caller Caller,
	input inboxcompliance.PostAuditInput,
) (Sent, error) {
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return Sent{}, err
	}

	if input.Reference == "" {
		return Sent{}, inboxcompliance.NewValidationError([]inboxcompliance.Violation{{
			Field:   inboxcompliance.FieldReference,
			Message: "Klaim yang dikirim tidak disebutkan.",
		}})
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Sent{}, err
	}

	claim, found, err := repo.FindInQueue(ctx, query, input.Reference)
	if err != nil {
		return Sent{}, fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return Sent{}, inboxcompliance.ErrClaimNotInQueue
	}

	entry, err := inboxcompliance.NewPostAuditEntry(input, claim, s.clock.Now())
	if err != nil {
		return Sent{}, err
	}

	saved, err := repo.CreatePostAudit(ctx, entry)
	if err != nil {
		return Sent{}, fmt.Errorf("menyimpan baris Post Audit: %w", err)
	}

	if s.logger != nil {
		// Satu-satunya tempat pengirimannya tercatat beserta pelakunya, karena tabelnya
		// tidak punya kolom untuk itu. Lihat catatan di tipe Caller.
		s.logger.Info(
			"klaim dikirim ke Post Audit",
			slog.String("portal", portalAlias),
			slog.String("oleh", caller.Login),
			slog.String("nomor_case", saved.CaseID),
			slog.String("klaim", saved.ClaimNumber),
			slog.Bool("ada_catatan", saved.Remarks != ""),
		)
	}

	return Sent{Entry: saved, Claim: claim}, nil
}
