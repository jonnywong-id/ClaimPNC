package usecase

import (
	"context"

	"claim-pnc/internal/registrasi"
)

// SurveyView adalah isi tab Survey.
type SurveyView struct {
	Survey []registrasi.Survey
}

// DocumentView adalah isi tab Unggah Dokumen.
type DocumentView struct {
	Category   []registrasi.DocumentCategory
	Attachment []registrasi.Attachment
}

// ProgressView adalah isi tab Progress Claim & Komunikasi.
type ProgressView struct {
	Progress      []registrasi.ProgressEntry
	Communication []registrasi.Communication
}

// Surveys membaca tab Survey sebuah klaim.
func (l *Service) Surveys(ctx context.Context, claimID string) (SurveyView, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return SurveyView{}, err
	}
	list, err := l.records.Surveys(ctx, claim.Keys())
	if err != nil {
		return SurveyView{}, err
	}
	return SurveyView{Survey: list}, nil
}

// Documents membaca tab Unggah Dokumen sebuah klaim.
func (l *Service) Documents(ctx context.Context, claimID string) (DocumentView, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return DocumentView{}, err
	}
	types, err := l.records.DocumentTypes(ctx, claim.Policy.BusinessCode)
	if err != nil {
		return DocumentView{}, err
	}
	files, err := l.records.Attachments(ctx, claim.Keys())
	if err != nil {
		return DocumentView{}, err
	}
	return DocumentView{Category: registrasi.DocumentChecklist(claim, types, files), Attachment: files}, nil
}

// ProgressRecords membaca tab Progress Claim & Komunikasi sebuah klaim.
func (l *Service) ProgressRecords(ctx context.Context, claimID string) (ProgressView, error) {
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return ProgressView{}, err
	}
	keys := claim.Keys()
	progress, err := l.records.Progress(ctx, keys)
	if err != nil {
		return ProgressView{}, err
	}
	talk, err := l.records.Communications(ctx, keys)
	if err != nil {
		return ProgressView{}, err
	}
	return ProgressView{Progress: progress, Communication: talk}, nil
}
