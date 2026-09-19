package usecase

import (
	"context"
	"errors"
	"fmt"

	"claim-pnc/internal/registrasi"
)

// Inbox mengembalikan pekerjaan yang menunggu seorang pengguna (`D-79`).
//
// Isinya dua kelompok yang sengaja disatukan, persis seperti di sistem lama: tugas
// Worklist yang sudah menjadi miliknya, dan tugas Workbasket yang belum bertuan pada
// antrean yang ia berwenang. Yang kedua belum menjadi pekerjaannya — ia baru menjadi
// pekerjaannya setelah diambil.
func (l *Service) Inbox(ctx context.Context, by Caller) ([]registrasi.Task, error) {
	return l.task.Inbox(ctx, by.Identity, by.Workbasket)
}

// ClaimTask menjadikan pemanggil pemilik sebuah tugas Workbasket.
//
// Dua orang yang menekan tombol ini pada tugas yang sama adalah kejadian biasa; yang
// kedua menerima ErrTaskAlreadyClaimed. Penguncian sesungguhnya ada di penyimpanan —
// pemeriksaan di sini menjawab kasus yang sudah terlihat tanpa menyentuh basis data.
func (l *Service) ClaimTask(ctx context.Context, taskID string, by Caller) (registrasi.Task, error) {
	var result registrasi.Task

	err := l.unit.Run(ctx, func(ctx context.Context) error {
		task, err := l.task.Get(ctx, taskID)
		if err != nil {
			return err
		}
		if err := task.Get(by.Identity, l.clock.Now()); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		result = task
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     task.ClaimID,
			ClaimNumber: task.ClaimNumber,
			Event:       "TUGAS_DIAMBIL",
			Actor:       by.Identity,
			At:          l.clock.Now().UTC(),
			Note:        "Tahap " + task.Stage,
		})
	})
	if err != nil {
		return registrasi.Task{}, err
	}
	return result, nil
}

// CompleteCommand menutup tahap yang aturan isiannya BUKAN milik modul ini.
//
// Delapan dari sembilan tahap alur Register berada dalam keadaan itu: View Polis milik
// `B-1`, Input Estimasi milik `B-5`, Choose Surveyor milik `B-8`, dan empat tahap penutup
// milik `B-11`. Modul ini memindahkan klaim di antara mereka; ia tidak memeriksa isinya.
//
// Yang tetap diperiksa di sini ada empat, dan keempatnya milik alur: tugas masih terbuka,
// pemanggil adalah pemiliknya, klaim benar-benar berada di tahap itu, dan tindakan yang
// dikirim memang penutup tahap tersebut.
type CompleteCommand struct {
	TaskID string

	// Action adalah nama penutup tahap — nama Flow Action di sistem lama.
	Action string

	// Return menandai tahap ditutup dengan tombol Back. Ia hanya berpengaruh pada
	// tahap yang alurnya memang punya gerbang Back.
	Return bool
}

// CompleteResult adalah keadaan klaim setelah sebuah tahap ditutup.
type CompleteResult struct {
	Claim         registrasi.Claim
	NextTask      *registrasi.Task
	DecisionTrace []string
}

// CompleteStage menutup satu tahap dan memindahkan klaim ke tahap berikutnya.
func (l *Service) CompleteStage(ctx context.Context, p CompleteCommand, by Caller) (CompleteResult, error) {
	claim, task, err := l.loadOpenTask(loadContext{
		ctx:    ctx,
		taskID: p.TaskID,
		action: p.Action,
	})
	if err != nil {
		return CompleteResult{}, err
	}
	if claim.CurrentStage == registrasi.StageInputRegister {
		// Tahap Input Register punya jalannya sendiri karena ia membawa isian dan
		// gerbang validasi. Menutupnya lewat jalur umum akan melewatkan keduanya.
		return CompleteResult{}, fmt.Errorf("%w: tahap Input Register ditutup lewat SimpanRegister",
			registrasi.ErrInvalidAction)
	}

	now := l.clock.Now().UTC()
	claim.RequestReturn = p.Return
	if p.Return {
		claim.ClaimStatus = registrasi.StatusReturned
	}
	claim.UpdatedBy = by.Identity
	claim.UpdatedAt = now

	newTask, trace, err := l.advance(advanceContext{
		ctx:    ctx,
		claim:  &claim,
		task:   &task,
		caller: by,
		action: p.Action,
		now:    now,
	})
	if err != nil {
		return CompleteResult{}, err
	}

	err = l.unit.Run(ctx, func(ctx context.Context) error {
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.task.Save(ctx, task); err != nil {
			return err
		}
		if newTask != nil {
			if err := l.task.Save(ctx, *newTask); err != nil {
				return err
			}
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID:     claim.ID,
			ClaimNumber: claim.Number,
			Event:       "TAHAP_DITUTUP",
			Actor:       by.Identity,
			At:          now,
			Note:        task.Stage + " → " + traceNote(trace),
		})
	})
	if err != nil {
		return CompleteResult{}, err
	}

	return CompleteResult{Claim: claim, NextTask: newTask, DecisionTrace: trace}, nil
}

// ViewClaim mengembalikan klaim beserta tugas terbukanya dan jalur tahap yang akan
// dilaluinya.
func (l *Service) ViewClaim(ctx context.Context, claimID string, by Caller) (ClaimSummary, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return ClaimSummary{}, err
	}

	summary := ClaimSummary{Claim: claim}

	task, err := l.task.OpenTaskForClaim(ctx, claim.ID)
	switch {
	case err == nil:
		summary.Task = &task
	case errors.Is(err, registrasi.ErrTaskNotFound):
		// Klaim yang sudah selesai tidak punya tugas terbuka. Itu keadaan yang sah.
	default:
		return ClaimSummary{}, err
	}

	if claim.CurrentStage != "" {
		path, err := l.flow.Path(claim.CurrentStage, registrasi.FlowContext{
			Claim:       claim,
			CallerRoles: by.Roles,
		})
		if err != nil {
			return ClaimSummary{}, err
		}
		summary.Path = path
	}

	return summary, nil
}

// ClaimSummary adalah klaim beserta keadaan alurnya.
type ClaimSummary struct {
	Claim registrasi.Claim

	// Task bernilai nil bila klaim sudah selesai.
	Task *registrasi.Task

	// Path adalah rangkaian tahap yang akan dilalui klaim dari tahap sekarang, dengan
	// data klaim yang berlaku sekarang. Ia gambaran, bukan janji: data yang berubah
	// mengubah jalurnya, dan itu memang perilaku sistem lama.
	Path []string
}
