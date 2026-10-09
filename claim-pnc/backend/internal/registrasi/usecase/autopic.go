package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// AutoPICAgent adalah nama pelaku pada jejak audit dan kolom pengubah klaim — nama entri
// agent di `Agents/TATReportAgent-Agents.xml` yang menjalankan `TransferAllCaseNotAssigned`.
const AutoPICAgent = "AutoPICAgent"

// AutoPICResult merangkum satu putaran agent.
type AutoPICResult struct {
	Examined int // tugas ServicePNC yang ditemukan
	Assigned int // klaim yang kini ber-PIC Teknik
	Moved    int // tugas yang ikut dipindahkan ke PIC itu
	Skipped  int // tidak ada kandidat, atau sudah ditangani instans/proses lain
	Failures []AutoPICFailure
}

// AutoPICFailure adalah satu klaim yang gagal diproses; klaim lain tetap diproses.
type AutoPICFailure struct {
	ClaimNumber string
	Err         error
}

// AssignUnassignedTechnicalPIC adalah agent `AutoPICAgent` (`TransferAllCaseNotAssigned`):
// untuk setiap tugas yang diparkir di ServicePNC karena klaimnya belum punya PIC Teknik,
// PIC dipilih dengan aturan yang sama dengan saat Claim Face Sheet (registrasi.PlanTechnicalPIC)
// lalu dicatat di klaim.
//
// # Yang berbeda dari Pega, atas keputusan Work Owner 2026-10-09
//
// Pega hanya mengisi `ClaimData.UserTeknis`; tugasnya tetap di worklist ServicePNC. Di sini
// tugas IKUT DIPINDAHKAN ke PIC itu — tetapi hanya tugas tahap yang memang dirutekan ke PIC
// Teknik (PNCTeknikRouter). Tugas tahap lain yang juga diparkir di ServicePNC, seperti
// RCL Dokter yang menunggu dokter dipilih, tetap di tempatnya: PIC Teknik bukan penerimanya.
//
// # Pemanggil
//
// Agent berjalan tanpa operator, sehingga jabatan pemanggilnya kosong — prosedur PA/Travel
// (`GETDATA_PICTEKNIK`) karena itu memakai cabang TRAVEL, sama seperti agent Pega yang
// berjalan tanpa `pyPosition` PA.
//
// # Satu transaksi per klaim
//
// Tugasnya dikunci lebih dulu (LockUnassigned) dan syaratnya diperiksa ulang di dalam
// kunci, sehingga dua instans yang berjalan bersamaan tidak memilih PIC dua kali untuk klaim
// yang sama. Kegagalan satu klaim tidak menghentikan klaim berikutnya.
func (l *Service) AssignUnassignedTechnicalPIC(ctx context.Context) (AutoPICResult, error) {
	if l.unassigned == nil {
		return AutoPICResult{}, errors.New("registrasi/usecase: agent PIC Teknik otomatis tidak terpasang (UnassignedTasks kosong)")
	}
	technical, ok := l.flow.Stage(registrasi.StageSendToTechnicalPIC)
	if !ok {
		return AutoPICResult{}, fmt.Errorf("registrasi/usecase: tahap %q tidak ada di flow", registrasi.StageSendToTechnicalPIC)
	}
	tasks, err := l.unassigned.UnassignedTechnicalTasks(ctx)
	if err != nil {
		return AutoPICResult{}, err
	}

	result := AutoPICResult{Examined: len(tasks)}
	for _, t := range tasks {
		outcome, err := l.assignOneUnassigned(ctx, t.ID, technical)
		switch {
		case err != nil:
			result.Failures = append(result.Failures, AutoPICFailure{ClaimNumber: t.ClaimNumber, Err: err})
		case outcome == autoPICSkipped:
			result.Skipped++
		default:
			result.Assigned++
			if outcome == autoPICMoved {
				result.Moved++
			}
		}
	}
	return result, nil
}

type autoPICOutcome int

const (
	autoPICSkipped autoPICOutcome = iota
	autoPICAssigned
	autoPICMoved
)

func (l *Service) assignOneUnassigned(ctx context.Context, taskID string, technical registrasi.Stage) (autoPICOutcome, error) {
	outcome := autoPICSkipped
	err := l.unit.Run(ctx, func(ctx context.Context) error {
		task, err := l.unassigned.LockUnassigned(ctx, taskID)
		if errors.Is(err, registrasi.ErrTaskNotFound) {
			return nil // sudah diproses atau sudah berpindah sejak daftar dibaca
		}
		if err != nil {
			return err
		}
		claim, err := l.claim.Get(ctx, task.ClaimID)
		if err != nil {
			return err
		}
		if registrasi.HasTechnicalPIC(claim) || strings.TrimSpace(claim.Policy.Number) == "" {
			return nil
		}

		to, err := l.assigner.Assign(ctx, technical, claim, "")
		if err != nil {
			return fmt.Errorf("registrasi/usecase: memilih PIC Teknik klaim %s: %w", claim.Number, err)
		}
		pic := strings.TrimSpace(to.Operator)
		if pic == "" || pic == registrasi.OperatorUnassigned {
			// Tidak ada kandidat. Transaksi tetap disimpan: rotasi tim yang sudah berjalan
			// ikut tercatat, sama dengan Pega.
			return nil
		}

		now := l.clock.Now().UTC()
		claim.TechnicalPIC = pic
		claim.UpdatedBy = AutoPICAgent
		claim.UpdatedAt = now
		outcome = autoPICAssigned
		note := "PIC Teknik " + pic + " dipilih agent"
		if stage, ok := l.flow.Stage(task.Stage); ok && stage.Router == registrasi.RouterPNCTechnical {
			if err := l.unassigned.Reassign(ctx, task.ID, pic); err != nil {
				return fmt.Errorf("registrasi/usecase: memindahkan tugas klaim %s: %w", claim.Number, err)
			}
			outcome = autoPICMoved
			note += "; tugas " + task.Stage + " dipindahkan dari " + registrasi.OperatorUnassigned
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "PIC_TEKNIS_OTOMATIS",
			Actor: AutoPICAgent, At: now, Note: note,
		})
	})
	if err != nil {
		return autoPICSkipped, err
	}
	return outcome, nil
}
