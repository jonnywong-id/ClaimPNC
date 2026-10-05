package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxrcl"
)

// MaxDoctorReasonLength adalah panjang kolom ALASAN_DOKTER_REJECT_RCL — VARCHAR2(4000).
const MaxDoctorReasonLength = 4000

// Decide menjalankan keputusan dokter RCL atas satu klaim di antrean pemanggil — padanan
// tombol layar kerja `RCLDokter` dan tombol Kirim layar Alasan Dokter (`SendToPUCL`).
//
// Kewenangannya sama dengan antrean: klaim hanya dapat diputus bila tampil di antrean
// pemanggil. Login yang tidak dikenal atau tidak aktif menghasilkan ErrClaimNotFound.
func (s *Service) Decide(
	ctx context.Context,
	portalAlias string,
	caller inboxrcl.Caller,
	claimNumber string,
	rawDecision string,
	doctorReason string,
	at time.Time,
) (inboxrcl.Outcome, error) {
	if caller.Login == "" {
		return inboxrcl.Outcome{}, inboxrcl.ErrCallerUnknown
	}

	decision, ok := inboxrcl.ParseDecision(rawDecision)
	if !ok {
		return inboxrcl.Outcome{}, inboxrcl.ErrUnknownDecision
	}

	// Kolom tujuannya 4.000 karakter (byte, VARCHAR2 tanpa CHAR). Dipotong, bukan ditolak:
	// textarea Pega tidak membatasi panjang, dan menolak membuat tombol Kirim gagal tanpa
	// sebab yang terlihat pengguna.
	reason := strings.TrimSpace(doctorReason)
	if len(reason) > MaxDoctorReasonLength {
		reason = truncateBytes(reason, MaxDoctorReasonLength)
	}

	repo, err := s.repoSelector(portalAlias)
	if err != nil {
		return inboxrcl.Outcome{}, err
	}

	operator, err := repo.OperatorFor(ctx, caller.Login)
	if err != nil {
		return inboxrcl.Outcome{}, fmt.Errorf("mencari login pemanggil di M_LOGIN_PNC: %w", err)
	}
	if operator == "" {
		return inboxrcl.Outcome{}, inboxrcl.ErrClaimNotFound
	}

	out, err := repo.Decide(ctx, inboxrcl.DecisionCommand{
		Operator:     operator,
		ClaimNumber:  claimNumber,
		Decision:     decision,
		DoctorReason: reason,
		At:           at,
	})
	if err != nil {
		if errors.Is(err, inboxrcl.ErrClaimNotFound) ||
			errors.Is(err, inboxrcl.ErrDecisionNotAllowed) ||
			errors.Is(err, inboxrcl.ErrTechnicalPICUnknown) {
			return inboxrcl.Outcome{}, err
		}
		return inboxrcl.Outcome{}, fmt.Errorf("menjalankan keputusan RCL Dokter: %w", err)
	}
	return out, nil
}

// truncateBytes memotong teks paling banyak n byte tanpa membelah satu karakter UTF-8.
func truncateBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut]
}
