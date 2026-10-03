package usecase

import (
	"context"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// AreaOptions membaca satu tingkat daftar pilihan wilayah kejadian.
//
// Ia hanya membaca master; tidak ada klaim maupun tugas yang disentuh, sehingga pemanggil
// tidak perlu sedang memegang tugas apa pun.
func (l *Service) AreaOptions(ctx context.Context, level registrasi.AreaLevel, parent string) ([]registrasi.AreaOption, error) {
	return l.area.Options(ctx, level, parent)
}

// CauseOfLossOptions membaca pilihan Penyebab Kerugian sebuah kode bisnis polis. Seperti
// AreaOptions, ia hanya membaca master.
func (l *Service) CauseOfLossOptions(ctx context.Context, businessCode string) ([]registrasi.CauseOfLossOption, error) {
	return l.causeOfLoss.CauseOfLossOptions(ctx, businessCode)
}

// SaveDraft menyimpan isian Input Register TANPA menutup tahapnya — tombol Save.
//
// # Apa bedanya dengan SaveRegister
//
// Tombol Save pada layar tahap Pega menyimpan work object tanpa menjalankan flow action:
// validasinya tidak dijalankan, tahapnya tidak berpindah, dan tidak ada pemberitahuan
// yang terbit. Petugas dapat menyimpan isian yang belum lengkap lalu melanjutkannya nanti.
// Yang menutup tahap — dan menjalankan seluruh gerbang validasi — tetap tombol Next
// (SaveRegister).
//
// Pemeriksaan kepemilikan tugas tetap berlaku: menyimpan tanpa maju pun mengubah klaim,
// dan klaim hanya boleh diubah pemegang tugasnya.
func (l *Service) SaveDraft(ctx context.Context, p RegisterCommand, by Caller) (registrasi.Claim, error) {
	claim, task, err := l.loadOpenTask(loadContext{
		ctx:           ctx,
		taskID:        p.TaskID,
		requiredStage: registrasi.StageInputRegister,
	})
	if err != nil {
		return registrasi.Claim{}, err
	}
	if !l.canWork(task, by) {
		return registrasi.Claim{}, registrasi.ErrNotTaskOwner
	}

	now := l.clock.Now().UTC()
	p.Return = false
	applyInput(&claim, p, by, now)

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "INPUT_REGISTER_DISIMPAN",
			Actor:       by.Identity,
			At:          now,
			Note:        "Isian Input Register disimpan tanpa menutup tahap",
		})
	})
	if err != nil {
		return registrasi.Claim{}, fmt.Errorf("registrasi/usecase: menyimpan isian: %w", err)
	}
	return claim, nil
}
