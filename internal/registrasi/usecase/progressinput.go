package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// Modal Input Progress Claim — lihat registrasi/progressinput.go.

// ProgressInputForm adalah isi awal modal.
type ProgressInputForm struct {
	Positions   []registrasi.ProgressPosition
	Status1     []registrasi.ProgressOption
	Statuses    []string
	DocTypes    []string
	EmailBroker string
}

// ProgressInputCommand adalah tombol Submit modal.
type ProgressInputCommand struct {
	ClaimID      string
	PositionID   string
	Progress1    string
	NeedIDs      []string
	Status       string
	Remarks      string
	NextFollowUp time.Time // tanggal kalender WIB; nol bila kosong
	DocType      string
	EmailBroker  string
}

// ProgressInputResult adalah hasil Submit.
type ProgressInputResult struct {
	// Warnings: langkah susulan yang gagal tanpa membatalkan progres (simpan master email broker).
	Warnings []string
}

func (l *Service) progressInputStore() (registrasi.ProgressInputStore, error) {
	if l.progressInput == nil {
		return nil, registrasi.ErrProgressInputUnavailable
	}
	return l.progressInput, nil
}

func brokerKey(claim registrasi.Claim) registrasi.BrokerEmail {
	return registrasi.BrokerEmail{
		PolicyNumber: strings.TrimSpace(claim.Policy.Number), GroupPanel: string(claim.Policy.Line),
		BusinessCode: strings.TrimSpace(claim.Policy.BusinessCode), BranchCode: strings.TrimSpace(claim.Policy.BranchCode),
		ClaimID: claim.ID,
	}
}

// ProgressInputForm membaca isi awal modal Input Progress Claim.
func (l *Service) ProgressInputForm(ctx context.Context, claimID string) (ProgressInputForm, error) {
	store, err := l.progressInputStore()
	if err != nil {
		return ProgressInputForm{}, err
	}
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return ProgressInputForm{}, err
	}
	var f ProgressInputForm
	if f.Positions, err = store.Positions(ctx, claim.Number); err != nil {
		return ProgressInputForm{}, err
	}
	if f.Status1, err = store.Status1Options(ctx); err != nil {
		return ProgressInputForm{}, err
	}
	if f.Statuses, err = store.StatusOptions(ctx); err != nil {
		return ProgressInputForm{}, err
	}
	if f.DocTypes, err = store.DocTypes(ctx); err != nil {
		return ProgressInputForm{}, err
	}
	if f.EmailBroker, err = store.BrokerEmail(ctx, brokerKey(claim)); err != nil {
		return ProgressInputForm{}, err
	}
	return f, nil
}

// ProgressNeedOptions membaca STATUS PERLU satu Status Progress 1 (BrowseStsProgress2_act2 /
// BrowseStsProgressDoc).
func (l *Service) ProgressNeedOptions(ctx context.Context, progress1, docType string) ([]registrasi.ProgressOption, error) {
	store, err := l.progressInputStore()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(progress1) == "" {
		return nil, nil
	}
	if strings.TrimSpace(progress1) != registrasi.ProgressDocumentStatus {
		docType = ""
	}
	return store.NeedOptions(ctx, progress1, docType)
}

// InputProgress menjalankan ValidationInputProgressClaim_act.
func (l *Service) InputProgress(ctx context.Context, p ProgressInputCommand, by Caller) (ProgressInputResult, error) {
	store, err := l.progressInputStore()
	if err != nil {
		return ProgressInputResult{}, err
	}
	claim, err := l.claim.Get(ctx, p.ClaimID)
	if err != nil {
		return ProgressInputResult{}, err
	}

	positions, err := store.Positions(ctx, claim.Number)
	if err != nil {
		return ProgressInputResult{}, err
	}
	var position *registrasi.ProgressPosition
	for i := range positions {
		if positions[i].ID == strings.TrimSpace(p.PositionID) {
			position = &positions[i]
		}
	}
	if position == nil {
		return ProgressInputResult{}, &registrasi.ValidationError{Violation: []registrasi.Violation{{
			Code: registrasi.ViolationProgress, Field: "posisi", Message: registrasi.MsgProgressPositionMissing}}}
	}
	if !position.Open() {
		return ProgressInputResult{}, &registrasi.ValidationError{Violation: []registrasi.Violation{{
			Code: registrasi.ViolationProgress, Field: "posisi", Message: registrasi.MsgProgressPositionClosed}}}
	}

	now := l.clock.Now().UTC()
	today := now.In(clock.ZoneWIB)
	in := registrasi.ProgressInput{
		ClaimNumber: claim.Number, PositionID: position.ID, Progress1: strings.TrimSpace(p.Progress1),
		Status: strings.TrimSpace(p.Status), Remarks: strings.TrimSpace(p.Remarks), NextFollowUp: p.NextFollowUp,
		User: by.Identity, At: now,
	}
	if in.Progress1 == registrasi.ProgressDocumentStatus {
		in.DocType = strings.TrimSpace(p.DocType)
	}

	// ValidationProgressClaim_act1 langkah 4: setiap baris STATUS PERLU wajib dipilih dari master.
	var violations []registrasi.Violation
	for _, id := range p.NeedIDs {
		id = strings.TrimSpace(id)
		name, ok := "", false
		if id != "" {
			if name, ok, err = store.NeedName(ctx, id); err != nil {
				return ProgressInputResult{}, err
			}
		}
		if !ok {
			id = ""
		}
		in.Needs = append(in.Needs, registrasi.ProgressNeed{ID: id, Name: name})
	}
	violations = append(violations, registrasi.CheckProgressInput(in, p.EmailBroker, today)...)

	// Langkah 11: STATUS PERLU baris terakhir sudah diinput hari ini.
	if last := in.LastNeed(); last.ID != "" {
		inputs, err := store.InputToday(ctx, claim.Number, now)
		if err != nil {
			return ProgressInputResult{}, err
		}
		if registrasi.AlreadyInputToday(last.Name, inputs) {
			violations = append([]registrasi.Violation{{Code: registrasi.ViolationProgress, Field: "status_perlu",
				Message: registrasi.MsgProgressAlreadyToday}}, violations...)
		}
	}
	if len(violations) > 0 {
		return ProgressInputResult{}, &registrasi.ValidationError{Violation: violations}
	}

	if err := l.unit.Run(ctx, func(ctx context.Context) error {
		if err := store.Insert(ctx, in); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "PROGRES_INPUT", Actor: by.Identity, At: now,
			Note: fmt.Sprintf("Input Progress Claim — posisi %s, status progress 1 %s, status progress 2 %s",
				position.Name, in.Progress1, in.LastNeed().ID),
		})
	}); err != nil {
		return ProgressInputResult{}, err
	}

	// SaveMasterEmailBrokerKlaim berjalan sesudah modal ditutup; galat prosedurnya tidak
	// pernah dibaca Pega, sehingga kegagalannya tidak membatalkan progres.
	var result ProgressInputResult
	b := brokerKey(claim)
	b.Email = p.EmailBroker
	if err := store.SaveBrokerEmail(ctx, b); err != nil {
		result.Warnings = append(result.Warnings, "Email progress klaim tidak tersimpan ke master: "+err.Error())
	}
	return result, nil
}
