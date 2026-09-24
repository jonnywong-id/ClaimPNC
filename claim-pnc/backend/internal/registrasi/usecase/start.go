package usecase

import (
	"context"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// StartCommand membuka klaim baru dari sebuah polis.
type StartCommand struct {
	PolicyNumber string
	Portal       string

	// RCVID menautkan klaim ke berkas laporan asalnya.
	//
	// Di Pega, tombol Register Klaim pada form Input Receive Document memanggil
	// CreateRegisterKlaimPNC, yang membuat case klaim DARI berkas RCV yang sedang
	// dibuka. Tautan itu bukan hiasan: kolom NOKLAIM pada baris RCV diisi dari sini,
	// dan itulah yang memindahkan berkasnya keluar dari tab Not Transferred.
	//
	// Kosong berarti klaim dimulai langsung dari layar registrasi, tanpa berkas RCV.
	RCVID string
}

// StartResult adalah klaim yang baru dibuka beserta tugas pertamanya.
type StartResult struct {
	Claim registrasi.Claim
	Task  registrasi.Task
}

// Start membuka klaim baru pada tahap pertama alur Register.
//
// # Yang sengaja BELUM terjadi di sini
//
// Nomor klaim belum terbit. Di sistem lama pun demikian: shape `Start1` menuju
// `View Polis`, dan penerbitan nomor berada di ujung tahap `Input Register`. Menerbitkan
// nomor lebih awal berarti setiap polis yang dibuka sekadar untuk dilihat ikut memakan
// satu nomor yang tidak dapat ditarik kembali.
func (l *Service) Start(ctx context.Context, p StartCommand, by Caller) (StartResult, error) {
	policyNumber := strings.TrimSpace(p.PolicyNumber)
	if policyNumber == "" {
		return StartResult{}, fmt.Errorf("registrasi/usecase: nomor polis wajib diisi")
	}

	policy, err := l.policy.Get(ctx, policyNumber)
	if err != nil {
		return StartResult{}, err
	}

	now := l.clock.Now().UTC()
	firstStage, ok := l.flow.Stage(l.flow.Start)
	if !ok {
		return StartResult{}, fmt.Errorf("%w: %q", registrasi.ErrUnknownStage, l.flow.Start)
	}

	claim := registrasi.Claim{
		ID:     l.id.New(),
		Portal: p.Portal,
		Policy: policy,

		Currency: policy.Currency,

		// Tautan ke berkas laporan asalnya; kosong bila klaim dimulai tanpa RCV.
		RCVID: strings.TrimSpace(p.RCVID),

		ProcessStatus:          registrasi.ProcessRunning,
		ClaimStatus:            "",
		ClaimFlag:              registrasi.FlagUnset,
		ProgressPositionStatus: registrasi.PositionInProgress,

		CurrentStage: firstStage.ID,
		CreatedBy:    by.Identity,
		CreatedAt:    now,
		UpdatedBy:    by.Identity,
		UpdatedAt:    now,
	}

	recipients, err := l.assigner.Assign(ctx, firstStage, claim, by.Identity)
	if err != nil {
		return StartResult{}, fmt.Errorf("registrasi/usecase: menentukan penerima tahap awal: %w", err)
	}
	task := registrasi.NewTask(l.id.New(), claim, firstStage, recipients, now)

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if err := l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID,
			Event:   "KLAIM_DIBUKA",
			Actor:   by.Identity,
			At:      now,
			Note:    "Polis " + policy.Number + " dibuka pada tahap " + firstStage.Name,
		}); err != nil {
			return err
		}

		// Berkas laporan asalnya ditandai DISERAHKAN, sehingga ia berpindah dari
		// "Not Transferred" ke "Not Registered".
		//
		// Ia berada DI DALAM transaksi yang sama dengan pembuatan klaim, dan itu
		// disengaja: klaim yang lahir tanpa berkasnya berpindah adalah keadaan yang
		// tampak seperti tombol tidak bekerja, dan petugas akan menekannya lagi.
		//
		// Nomor klaim belum ada di sini — ia terbit di ujung Input Register (`ADR-0009`).
		// Karena itu yang terisi baru TRANSFERASM; NOKLAIM menyusul di SaveRegister.
		if claim.RCVID == "" {
			return nil
		}
		return l.reportLink.MarkHandedOver(ctx, claim.RCVID, now)
	})
	if err != nil {
		return StartResult{}, err
	}

	return StartResult{Claim: claim, Task: task}, nil
}
