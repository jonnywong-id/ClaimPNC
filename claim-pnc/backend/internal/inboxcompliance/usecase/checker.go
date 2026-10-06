package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"claim-pnc/internal/inboxcompliance"
)

// Form Compliance Checker — membuka satu klaim dari antrean, dan menyimpan keputusannya.
//
// # Kenapa kedua operasi ini ada di modul Inbox Compliance, bukan di modul tersendiri
//
// Karena di Pega pun begitu. Form ini satu-satunya flow action pada assignment yang
// menunggu di workbasket `CompliancePNC`, dan assignment itulah yang menjadi baris tab
// Compliance. Memisahkannya menjadi modul sendiri akan memutus satu alur menjadi dua, lalu
// menuntut keduanya sepakat tentang klaim mana yang sedang diperiksa.

// CheckerOpened adalah form yang terbuka beserta bahan menggambarnya.
type CheckerOpened struct {
	// Case adalah klaimnya beserta keputusan yang sudah pernah disimpan, bila ada.
	Case inboxcompliance.CheckerCase

	// Choices adalah keempat Pilihan Compliance beserta labelnya.
	//
	// Ia datang dari server, bukan disalin ke layar, dengan alasan yang sama seperti
	// daftar tab: keempat nilainya adalah HASIL PEMBACAAN
	// `Property/PilihanCompliance_property.xml`, dan tempat pembacaan itu tercatat adalah
	// backend. Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat.
	Choices []inboxcompliance.Choice
}

// OpenChecker membuka form Compliance Checker atas satu klaim.
//
// # Kenapa klaimnya dicari DI ANTREAN, bukan di seluruh tabel klaim
//
// Meniru Pega: form ini adalah flow action di atas sebuah assignment. Tanpa assignment yang
// menunggu, tidak ada yang dapat dibuka — dan itu bukan rincian teknis melainkan kewenangan.
// Klaim yang sudah selesai diperiksa, atau yang sedang menunggu di antrean lain, bukan
// pekerjaan petugas Compliance.
//
// Akibatnya galatnya ErrClaimNotInQueue, bukan "tidak ditemukan". Pembedaan itu yang membuat
// petugas menyegarkan daftarnya alih-alih mencari klaim yang sebenarnya ada.
func (s *Service) OpenChecker(
	ctx context.Context, portalAlias string, reference string,
) (CheckerOpened, error) {
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return CheckerOpened{}, err
	}

	if reference == "" {
		return CheckerOpened{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim yang dibuka tidak disebutkan.",
			}},
		)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return CheckerOpened{}, err
	}

	claim, found, err := repo.FindInQueue(ctx, query, reference)
	if err != nil {
		return CheckerOpened{}, fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return CheckerOpened{}, inboxcompliance.ErrClaimNotInQueue
	}

	// Aging diisi di sini, bukan di repo, dengan alasan yang sama seperti pada daftar:
	// hitungan waktu tunggu adalah aturan bisnis yang `D-50` tetapkan ditulis ulang di Go.
	claim = claim.WithElapsed(s.clock.Now())

	opened := CheckerOpened{
		Case:    inboxcompliance.CheckerCase{Claim: claim},
		Choices: inboxcompliance.Choices(),
	}

	decision, decided, err := repo.FindDecision(ctx, reference)
	if err != nil {
		return CheckerOpened{}, fmt.Errorf("membaca keputusan Compliance: %w", err)
	}
	if decided {
		opened.Case.Decision = &decision
	}

	return opened, nil
}

// Decided adalah hasil penyimpanan satu keputusan Compliance.
type Decided struct {
	// Decision adalah keputusan yang tersimpan.
	Decision inboxcompliance.Decision

	// Claim adalah klaim yang diputuskan, sebagaimana terbaca dari antrean.
	Claim inboxcompliance.WorkItem

	// PostAudit adalah baris Post Audit yang ikut terbit, HANYA pada ChoicePostAudit.
	//
	// Pointer supaya layar dapat membedakan "Post Audit terbit, ini nomornya" dari tiga
	// pilihan lain yang tidak menerbitkan apa pun.
	PostAudit *inboxcompliance.PostAuditEntry
}

// SubmitDecision menyimpan keputusan Compliance atas satu klaim — padanan tombol
// "Simpan Data" pada `Section/ComplianceChecker-Section.xml`.
//
// # Urutan yang ditiru dari SetComplianceResult
//
//	langkah 1      baca PilihanCompliance
//	langkah 2      CPLValidDate := now           bila pilihan 1
//	langkah 5      TanggalKirimPostAudit := now  bila pilihan 2
//	langkah 6–8    InsertHistoryClaimPNC dengan keterangan per pilihan
//	langkah 9      Obj-Save
//	langkah 10     pxAddChildWork kelas Work-Compliance  bila pilihan 2
//
// Langkah 10 itulah yang di sini menjadi CreatePostAudit: child case `Work-Compliance`
// adalah baris Post Audit, dan awalan `CPL-` pada nomornya memang singkatan kelas itu.
//
// # Akibat yang BELUM sampai ke klaim, dan kenapa
//
// `SetComplianceResult` juga menulis ke KLAIMNYA — `StatusClaim := "1151"`,
// `.ComplianceStatus` pada tiap Objek Pertanggungan, dan dua Ticket rule
// (`SendtoAnalysator` bila PA, `SendToPICTravel` bila Travel). Tidak satu pun dikerjakan di
// sini.
//
// Alasannya `P-1`: ketiganya menyentuh `T_CLAIM_PNC` dan `PC_ASM_FW_GCNMFW_WORK`, yang
// selama masa paralel masih dimiliki Pega. Menulisnya dari sini berarti dua sistem menulis
// satu tabel dengan aturan validasi berbeda — kelas kerusakan data yang `P-1` ada untuk
// mencegahnya.
//
// Konsekuensinya HARUS dinyatakan, bukan dibiarkan terlihat sebagai fitur yang jalan:
// keputusan tersimpan dan terbaca di aplikasi baru, tetapi **klaimnya di Pega tidak
// berpindah status dan tidak keluar dari antrean**. Sampai kepemilikan tabelnya berpindah
// (`D-63`, menempuh permintaan tertulis → persetujuan Work Owner → pelaksanaan DBA), form
// ini mencatat keputusan — ia belum menjalankan alurnya.
func (s *Service) SubmitDecision(
	ctx context.Context,
	portalAlias string,
	caller Caller,
	input inboxcompliance.DecisionInput,
) (Decided, error) {
	query, err := inboxcompliance.NewQuery(inboxcompliance.QueryInput{
		Tab: inboxcompliance.TabCompliance,
	})
	if err != nil {
		return Decided{}, err
	}

	if input.Reference == "" {
		return Decided{}, inboxcompliance.NewValidationError(
			[]inboxcompliance.Violation{{
				Field:   inboxcompliance.FieldReference,
				Message: "Klaim yang diputuskan tidak disebutkan.",
			}},
		)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return Decided{}, err
	}

	claim, found, err := repo.FindInQueue(ctx, query, input.Reference)
	if err != nil {
		return Decided{}, fmt.Errorf("mencari klaim di antrean Compliance: %w", err)
	}
	if !found {
		return Decided{}, inboxcompliance.ErrClaimNotInQueue
	}

	decision, err := inboxcompliance.NewDecision(input, claim, caller.Login, s.clock.Now())
	if err != nil {
		return Decided{}, err
	}

	if err := repo.SaveDecision(ctx, decision); err != nil {
		return Decided{}, fmt.Errorf("menyimpan keputusan Compliance: %w", err)
	}

	result := Decided{Decision: decision, Claim: claim}

	// Hanya ChoicePostAudit yang menerbitkan baris Post Audit — meniru langkah 10
	// `SetComplianceResult`, yang prasyaratnya `local.pilihan == 2`.
	//
	// Catatan yang dibawa ke barisnya adalah Remarks, bukan Note. Itu bukan tebakan:
	// langkah 5 menyalin `.ClaimData.ComplianceRemark` ke
	// `childPageCompliance.ComplianceRemarks`, dan kolom `REMARKS` pada
	// `T_CLAIM_COMPLIANCE_H` itulah yang tampil sebagai "Catatan" di tab Post Audit.
	if decision.Choice == inboxcompliance.ChoicePostAudit {
		entry, err := inboxcompliance.NewPostAuditEntry(
			inboxcompliance.PostAuditInput{
				Reference: input.Reference,
				Remarks:   decision.Remarks,
			},
			claim,
			decision.DecidedAt,
		)
		if err != nil {
			return Decided{}, err
		}

		// Urutannya SENGAJA: keputusan disimpan lebih dulu, baris Post Audit menyusul.
		//
		// Keduanya TIDAK berada dalam satu transaksi, dan itu kekurangan yang disadari.
		// Bila yang kedua gagal, keputusan tersimpan tanpa baris Post Audit — keadaan
		// yang dapat dilihat petugas dan dapat diulang, karena form masih terbuka.
		// Kebalikannya jauh lebih buruk: baris Post Audit terbit tanpa jejak siapa yang
		// memutuskan.
		//
		// Membungkus keduanya menuntut kepemilikan transaksi di lapisan aplikasi
		// (`08-TECHNICAL-STRATEGY.md` §4.5), yang seam Repo modul ini belum punya.
		saved, err := repo.CreatePostAudit(ctx, entry)
		if err != nil {
			return Decided{}, fmt.Errorf("menerbitkan baris Post Audit: %w", err)
		}
		result.PostAudit = &saved
	}

	if s.logger != nil {
		// Satu-satunya jejak yang menyebut pelakunya sampai `S-5` ada. Keputusan ini
		// menyangkut uang — `ChoiceFraud` menolak klaim — sehingga ketiadaan jejak audit
		// yang sebenarnya lebih berat akibatnya di sini daripada di jalur mana pun lain
		// di modul ini (`D-59`).
		label := decision.Choice
		if choice, known := inboxcompliance.FindChoice(decision.Choice); known {
			label = choice.Label
		}

		s.logger.Info(
			"keputusan Compliance disimpan",
			slog.String("portal", portalAlias),
			slog.String("oleh", caller.Login),
			slog.String("klaim", decision.Reference),
			slog.String("pilihan", decision.Choice),
			slog.String("pilihan_label", label),
			slog.Bool("post_audit_terbit", result.PostAudit != nil),
		)
	}

	return result, nil
}
