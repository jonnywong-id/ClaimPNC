package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// Tab Survey tahap Choose Surveyor — `Section/TabSurvey_sect.xml`. Lihat registrasi/survey.go.

// SurveyObjectInput adalah isian satu baris grid Tambah Survey yang dikirim layar.
type SurveyObjectInput struct {
	ObjectID       string
	Selected       bool
	SurveyLocation string
	SurveyorType   string
	SurveyorName   string
	SurveyorLogin  string
	SurveyorAddr   string
	SurveyorEmail  string
	BranchCode     string
	BranchName     string
	MarineName     string
	MarineLogin    string
}

// SurveyCommand adalah tombol Simpan / Transfer Survei / Transfer Komite.
type SurveyCommand struct {
	ClaimID string
	TaskID  string
	Objects []SurveyObjectInput

	// CaseSurveyorType adalah `ClaimData.SurveyData.SurveyorType`: Tipe Surveyor yang terakhir
	// diganti (CheckTypeSurveyor_act langkah 3) — SURVEYTYPE baris T_SURVEYORLIST.
	CaseSurveyorType string
}

// SurveyCommitteeCommand adalah modal Transfer Komite (`ClaimComitee`).
type SurveyCommitteeCommand struct {
	SurveyCommand
	Note      registrasi.SurveyCommitteeNote
	Nominated []registrasi.SurveyorOption
	// Manual adalah "Penunjukan Manual Ke Komite" (TempNominated.STATE).
	Manual bool
}

// SurveyCancelCommand adalah tombol "Ya" pada modal Batal Survei.
type SurveyCancelCommand struct {
	ClaimID  string
	TaskID   string
	ObjectID string
}

// SurveyTabView adalah isi tab Survey.
type SurveyTabView struct {
	Objects []registrasi.SurveyObject
	Request registrasi.SurveyRequest
	// HasRequest: ada baris T_REQ_SURVEY.
	HasRequest bool
	// AddVisible: sub-tab Tambah Survey tampil — `AccessGroup.pyAccessGroup!='GCNMFW:PncAdmin'`.
	AddVisible bool
	// CoMember: Sinar Mas anggota koasuransi — jalur Transfer Survei untuk tipe 2/3/4.
	CoMember bool
	// MarineHull: kolom Surveyor Marine tampil.
	MarineHull bool
	// PA: kolom Investigator (lini PA).
	PA bool
}

// SurveyResult adalah hasil satu tombol proses survey.
type SurveyResult struct {
	View      SurveyTabView
	Surveys   []string // nomor survey (SRVN) yang terbit
	Committee []string // nomor kasus komite (KMTN) yang terbit
	Auto      bool     // jalur otomatis (Auto_ChildSurvey_act)
}

// ErrSurveyNotInProgress: Batal Survei hanya untuk objek berstatus Sedang Proses (tombolnya hanya
// tampil pada `.ObjectStatus = 3`).
var ErrSurveyNotInProgress = errors.New("only a survey that is in progress can be cancelled")

func (l *Service) surveyStore() (registrasi.SurveyStore, error) {
	if l.survey == nil {
		return nil, registrasi.ErrSurveyUnavailable
	}
	return l.survey, nil
}

// SurveyTab membaca isi tab Survey.
func (l *Service) SurveyTab(ctx context.Context, claimID string, by Caller) (SurveyTabView, error) {
	store, err := l.surveyStore()
	if err != nil {
		return SurveyTabView{}, err
	}
	claim, err := l.claim.Get(ctx, claimID)
	if err != nil {
		return SurveyTabView{}, err
	}
	return l.surveyView(ctx, store, claim, by)
}

func (l *Service) surveyView(ctx context.Context, store registrasi.SurveyStore, claim registrasi.Claim, by Caller) (SurveyTabView, error) {
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return SurveyTabView{}, err
	}
	req, ok, err := store.Request(ctx, claim.Keys())
	if err != nil {
		return SurveyTabView{}, err
	}
	return SurveyTabView{
		Objects: objects, Request: req, HasRequest: ok,
		AddVisible: !hasRole(by.Roles, registrasi.RoleAdmin),
		CoMember:   registrasi.CoInsuranceMember(claim.Policy),
		MarineHull: registrasi.IsMarineHull(claim.Policy),
		PA:         claim.Policy.Line == registrasi.LinePersonalAccident,
	}, nil
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if strings.EqualFold(strings.TrimSpace(r), role) {
			return true
		}
	}
	return false
}

// SurveyorOptions adalah pilihan surveyor satu tipe (`ChooseSurveyorType_act`).
func (l *Service) SurveyorOptions(ctx context.Context, surveyorType string) ([]registrasi.SurveyorOption, error) {
	store, err := l.surveyStore()
	if err != nil {
		return nil, err
	}
	return store.Surveyors(ctx, surveyorType)
}

// NominatedSurveyorOptions adalah pilihan Nominated Loss Adjuster.
func (l *Service) NominatedSurveyorOptions(ctx context.Context) ([]registrasi.SurveyorOption, error) {
	store, err := l.surveyStore()
	if err != nil {
		return nil, err
	}
	return store.NominatedOptions(ctx)
}

// surveyContext memuat klaim dan tugasnya, lalu memeriksa tahap dan pemiliknya.
func (l *Service) surveyContext(ctx context.Context, claimID, taskID string, by Caller) (registrasi.SurveyStore, registrasi.Claim, error) {
	store, err := l.surveyStore()
	if err != nil {
		return nil, registrasi.Claim{}, err
	}
	claim, task, err := l.loadOpenTask(loadContext{ctx: ctx, taskID: taskID, action: registrasi.ActionInputSurveyor})
	if err != nil {
		return nil, registrasi.Claim{}, err
	}
	if claim.ID != claimID {
		return nil, registrasi.Claim{}, fmt.Errorf("%w: tugas %s bukan milik klaim %s", registrasi.ErrInvalidAction, taskID, claimID)
	}
	if !settlementStages[task.Stage] {
		return nil, registrasi.Claim{}, fmt.Errorf("%w: %q", registrasi.ErrNotAvailableAtStage, task.Stage)
	}
	if !l.canWork(task, by) {
		return nil, registrasi.Claim{}, registrasi.ErrNotTaskOwner
	}
	return store, claim, nil
}

// mergeSurveyInput menumpangkan isian layar ke objek tersimpan. Lokasi, Tipe Surveyor, dan
// surveyor terkunci pada status 1/3 (When `IsStatus`); Pilih tetap dapat diubah.
func mergeSurveyInput(objects []registrasi.SurveyObject, input []SurveyObjectInput) {
	byID := map[string]SurveyObjectInput{}
	for _, in := range input {
		byID[strings.TrimSpace(in.ObjectID)] = in
	}
	for i := range objects {
		in, ok := byID[objects[i].ObjectID]
		if !ok {
			continue
		}
		o := &objects[i]
		o.Selected = in.Selected
		if o.Status == registrasi.SurveyInProgress || o.Status == registrasi.SurveyAwaitingApproval {
			continue
		}
		o.SurveyLocation = strings.TrimSpace(in.SurveyLocation)
		o.SurveyorType = strings.TrimSpace(in.SurveyorType)
		o.SurveyorName = strings.TrimSpace(in.SurveyorName)
		o.SurveyorLogin = strings.TrimSpace(in.SurveyorLogin)
		o.SurveyorAddr = strings.TrimSpace(in.SurveyorAddr)
		o.SurveyorEmail = strings.TrimSpace(in.SurveyorEmail)
		o.BranchCode = strings.TrimSpace(in.BranchCode)
		o.BranchName = strings.TrimSpace(in.BranchName)
		o.MarineName = strings.TrimSpace(in.MarineName)
		o.MarineLogin = strings.TrimSpace(in.MarineLogin)
	}
}

// SaveSurvey adalah tombol Simpan (`PNCSaveButton`: Obj-Save + Commit) — hanya menyimpan grid.
func (l *Service) SaveSurvey(ctx context.Context, p SurveyCommand, by Caller) (SurveyTabView, error) {
	store, claim, err := l.surveyContext(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return SurveyTabView{}, err
	}
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return SurveyTabView{}, err
	}
	mergeSurveyInput(objects, p.Objects)
	now := l.clock.Now().UTC()
	if err := l.unit.Run(ctx, func(ctx context.Context) error {
		return store.SaveObjects(ctx, claim.ID, objects, by.Identity, now)
	}); err != nil {
		return SurveyTabView{}, err
	}
	return l.surveyView(ctx, store, claim, by)
}

func surveyViolation(msg string) error {
	return &registrasi.ValidationError{Violation: []registrasi.Violation{{Code: registrasi.ViolationSurvey, Field: "survey", Message: msg}}}
}

// surveyPremium adalah `ValidationPremi_act` jeniskomite "survey": lolos bila premi LUNAS atau
// tidak ditetapkan, bila BELUM LUNAS pada entitas ASM dengan sumber bisnis KBRU, atau bila ada
// Open Protection premi yang disetujui. Mengembalikan status premi.
func (l *Service) surveyPremium(ctx context.Context, claim registrasi.Claim, now time.Time) (string, error) {
	caseID, err := l.acceptance.PolicyCaseID(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return "", err
	}
	statement, err := l.premium.Statement(ctx, claim.Portal, registrasi.PremiumQuery{
		PolicyNumber: claim.Policy.Number, ProdKe: claim.Policy.ProdKe,
		RequestedAt: claim.Policy.CoverageEnd, CaseID: registrasi.PremiumCaseID(caseID),
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrPremiumUnavailable, err)
	}
	status := registrasi.PremiumStatusOf(statement, now)
	if status == registrasi.PremiumPaid || status == "" {
		return status, nil
	}
	if status == registrasi.PremiumUnpaid && strings.EqualFold(claim.Portal, "ASM") &&
		registrasi.KBRUSourcesOfBusiness[strings.TrimSpace(claim.Policy.SourceOfBusiness)] {
		return status, nil
	}
	ok, err := l.acceptance.OpenProtectionApproved(ctx, claim.Policy.Number, claim.ID, claim.Number)
	if err != nil {
		return "", err
	}
	if ok {
		return status, nil
	}
	return status, surveyViolation(registrasi.MsgSurveyPremiumUnpaid)
}

// newSurvey menerbitkan satu survey SRVN: baris T_SURVEYORLIST dan progres "Auto Create Survey".
func (l *Service) newSurvey(ctx context.Context, store registrasi.SurveyStore, claim registrasi.Claim,
	o registrasi.SurveyObject, caseType, surveyor, marine string, by Caller, now time.Time) (string, error) {
	id, err := store.NextSurveyID(ctx, now)
	if err != nil {
		return "", err
	}
	if err := store.InsertRecord(ctx, registrasi.SurveyRecord{
		CaseID: id, ClaimID: claim.ID, SurveyType: caseType, SurveyorName: surveyor, MarineName: marine,
		SurveyDate: now, SurveyLocation: o.SurveyLocation, ObjectName: o.Name, ObjectLocation: o.Location,
		ObjectID: o.ObjectID, Index: "1", TreatmentDate: claim.DateOfLoss,
		Status: registrasi.SurveyRecordOnProgress, At: now,
	}); err != nil {
		return "", err
	}
	if _, err := l.acceptance.StartProgress(ctx, registrasi.ProgressStart{
		ClaimNumber: claim.Number, CaseID: id, Position: registrasi.SurveyProgressPosition,
		Note: registrasi.SurveyProgressNote, Progress1: registrasi.SurveyProgress1,
		Progress2: registrasi.SurveyProgress2, User: by.Identity, At: now,
	}); err != nil {
		return "", err
	}
	return id, nil
}

// postChildSurvey adalah `Post_ChildSurvey_act`: satu survey SRVN untuk SETIAP objek klaim
// (prekondisi `.pySelected` langkah 2 dinonaktifkan di rule — ditiru apa adanya, keputusan
// Work Owner 2026-10-11), ditambah survey kedua untuk Surveyor Marine (Marine Hull, tipe 2/4).
// Objek berstatus Sedang Proses.
func (l *Service) postChildSurvey(ctx context.Context, store registrasi.SurveyStore, claim registrasi.Claim,
	objects []registrasi.SurveyObject, caseType string, coMember bool, by Caller, now time.Time) ([]string, error) {
	marineHull := registrasi.IsMarineHull(claim.Policy)
	var issued []string
	for i := range objects {
		o := &objects[i]
		if coMember && (o.SurveyorName == "" || (marineHull && o.MarineName == "")) {
			return nil, surveyViolation(registrasi.MsgSurveyAdjusterEmpty)
		}
		id, err := l.newSurvey(ctx, store, claim, *o, caseType, o.SurveyorName, "", by, now)
		if err != nil {
			return nil, err
		}
		o.SurveyID, o.Status = id, registrasi.SurveyInProgress
		issued = append(issued, id)
		if marineHull && (o.SurveyorType == registrasi.SurveyorLossAdjuster || o.SurveyorType == registrasi.SurveyorSurveyAgent) {
			marine, err := l.newSurvey(ctx, store, claim, *o, caseType, o.MarineName, o.MarineName, by, now)
			if err != nil {
				return nil, err
			}
			o.SurveyIDMarine = marine
			issued = append(issued, marine)
		}
	}
	return issued, nil
}

// TransferSurvey adalah tombol Transfer Survei (`InternalSurveyor_act`).
func (l *Service) TransferSurvey(ctx context.Context, p SurveyCommand, by Caller) (SurveyResult, error) {
	store, claim, err := l.surveyContext(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return SurveyResult{}, err
	}
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return SurveyResult{}, err
	}
	mergeSurveyInput(objects, p.Objects)
	now := l.clock.Now().UTC()
	coMember := registrasi.CoInsuranceMember(claim.Policy)

	// Langkah 2–3: cek premi hanya pada jalur anggota koasuransi.
	if coMember {
		if _, err := l.surveyPremium(ctx, claim, now); err != nil {
			return SurveyResult{}, err
		}
	}
	// Langkah 4–5.
	for _, o := range objects {
		if o.Selected && !(o.SurveyorType == registrasi.SurveyorInternal || coMember) {
			return SurveyResult{}, surveyViolation(registrasi.MsgSurveyUseCommittee)
		}
	}
	// Langkah 8–10: setError_act, SetErrorSRV_act.
	rows, err := store.Rows(ctx, claim.Keys())
	if err != nil {
		return SurveyResult{}, err
	}
	if v := registrasi.CheckSurveyObjects(objects, rows); len(v) > 0 {
		return SurveyResult{}, &registrasi.ValidationError{Violation: v}
	}

	var issued []string
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		var err error
		if issued, err = l.postChildSurvey(ctx, store, claim, objects, p.CaseSurveyorType, coMember, by, now); err != nil {
			return err
		}
		claim.ClaimStatus = registrasi.StatusClaimSurvey // langkah 13
		claim.UpdatedBy, claim.UpdatedAt = by.Identity, now
		if err := store.SaveObjects(ctx, claim.ID, objects, by.Identity, now); err != nil {
			return err
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "SURVEY_TRANSFER", Actor: by.Identity, At: now,
			Note: "Transfer Survei — survey " + strings.Join(issued, ", "),
		})
	})
	if err != nil {
		return SurveyResult{}, err
	}
	view, err := l.surveyView(ctx, store, claim, by)
	return SurveyResult{View: view, Surveys: issued}, err
}

// TransferSurveyCommittee adalah tombol Transfer Komite — flow action `ClaimComitee`,
// pasca-proses `ValidationAnalysis`.
func (l *Service) TransferSurveyCommittee(ctx context.Context, p SurveyCommitteeCommand, by Caller) (SurveyResult, error) {
	store, claim, err := l.surveyContext(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return SurveyResult{}, err
	}
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return SurveyResult{}, err
	}
	mergeSurveyInput(objects, p.Objects)
	now := l.clock.Now().UTC()

	// Langkah 1–3: cek premi (selalu).
	premium, err := l.surveyPremium(ctx, claim, now)
	if err != nil {
		return SurveyResult{}, err
	}
	// Langkah 6–9: setError_act, SetErrorSRV_act.
	rows, err := store.Rows(ctx, claim.Keys())
	if err != nil {
		return SurveyResult{}, err
	}
	if v := registrasi.CheckSurveyObjects(objects, rows); len(v) > 0 {
		return SurveyResult{}, &registrasi.ValidationError{Violation: v}
	}
	line := registrasi.SurveyCommitteeLine(claim.Policy) // langkah 9–11

	// Langkah 12: surveyor objek terpilih (yang terakhir menang) dan status Direksi.
	var surveyor, marine, surveyType string
	for _, o := range objects {
		if !o.Selected {
			continue
		}
		surveyor, surveyType = o.SurveyorName, o.SurveyorType
		if o.MarineName != "" {
			marine = o.MarineName
		}
	}
	director, err := store.DirectorSurveyor(ctx, []string{surveyor, marine})
	if err != nil {
		return SurveyResult{}, err
	}
	// Langkah 12.6: total estimasi klaim yang sudah dibuatkan CFS, seluruh objek.
	var total registrasi.Money
	for _, item := range claim.InsuredItem {
		for _, cov := range item.Coverage {
			for _, it := range cov.Item {
				for _, e := range it.Estimation {
					if e.FaceSheet && e.Type == registrasi.EstimateClaim {
						total += e.Converted
					}
				}
			}
		}
	}
	// Langkah 13: batas jalur otomatis.
	// Policy.SyariahStatus dari dokumen polis (sumber yang sama dengan DLA dan Kasir).
	policyDoc, err := l.dla.Policy(ctx, claim.Policy.Number, claim.Policy.ProdKe)
	if err != nil {
		return SurveyResult{}, err
	}
	syariah := "0"
	if policyDoc.Syariah {
		syariah = "1"
	}
	limit, hasLimit, err := store.Limit(ctx, claim.Policy.BusinessCode, syariah)
	if err != nil {
		return SurveyResult{}, err
	}
	// Langkah 25–26: nominasi; flagnominated "1" bila daftar kosong atau memuat surveyor terpilih.
	nominated := len(p.Nominated) == 0
	for _, n := range p.Nominated {
		if strings.TrimSpace(n.Name) == strings.TrimSpace(surveyor) {
			nominated = true
		}
	}
	// Langkah 29: jalur.
	auto := !p.Manual && hasLimit &&
		(surveyType == registrasi.SurveyorSurveyAgent || surveyType == registrasi.SurveyorLossAdjuster) &&
		!director && premium == registrasi.PremiumPaid && total >= limit && nominated

	var members []registrasi.SurveyCommitteeMember
	if !auto {
		if members, err = store.CommitteeMembers(ctx, line, !director); err != nil {
			return SurveyResult{}, err
		}
		if len(members) == 0 {
			return SurveyResult{}, surveyViolation(fmt.Sprintf("%s (no active survey committee member in EMAILKOMITE for %s).",
				registrasi.MsgSurveyNoCommittee, line))
		}
	}

	coMember := registrasi.CoInsuranceMember(claim.Policy)
	result := SurveyResult{Auto: auto}
	err = l.unit.Run(ctx, func(ctx context.Context) error {
		// Langkah 19: Post_ChildSurvey_act untuk seluruh objek.
		issued, err := l.postChildSurvey(ctx, store, claim, objects, p.CaseSurveyorType, coMember, by, now)
		if err != nil {
			return err
		}
		result.Surveys = issued
		claim.ClaimStatus = registrasi.StatusClaimSurvey // langkah 21
		if err := store.SaveNominated(ctx, claim.ID, p.Nominated); err != nil {
			return err
		}
		if auto {
			// Langkah 40: Auto_ChildSurvey_act — survey berikutnya untuk objek terpilih.
			marineHull := registrasi.IsMarineHull(claim.Policy)
			for i := range objects {
				o := &objects[i]
				if !o.Selected {
					continue
				}
				if o.SurveyorName == "" || (marineHull && o.MarineName == "") {
					return surveyViolation(registrasi.MsgSurveyAdjusterEmpty)
				}
				id, err := l.newSurvey(ctx, store, claim, *o, p.CaseSurveyorType, o.SurveyorName, "", by, now)
				if err != nil {
					return err
				}
				o.SurveyID, o.Status = id, registrasi.SurveyInProgress
				result.Surveys = append(result.Surveys, id)
			}
		} else {
			// Langkah 30: Post_ChildKomite_act — satu kasus komite per objek terpilih bukan internal.
			for _, o := range objects {
				if !o.Selected || o.SurveyorType == registrasi.SurveyorInternal {
					continue
				}
				id, err := l.committees.NextCaseID(ctx, now)
				if err != nil {
					return err
				}
				c := registrasi.CommitteeCase{
					ID: id, ClaimID: claim.ID, ClaimNumber: claim.Number, ObjectID: o.ObjectID,
					TransferType: registrasi.SurveyCommitteeTransfer, Line: line,
					Applicant: by.Identity, CreatedBy: by.Identity, UpdatedBy: by.Identity, CreatedAt: now, UpdatedAt: now,
				}
				for k, m := range members {
					c.Members = append(c.Members, registrasi.CommitteeMember{
						CaseID: id, ClaimNumber: claim.Number, Operator: m.OperatorID, Level: k + 1,
						Decision: registrasi.DecisionPending, CaseStatus: registrasi.CommitteeCaseOpen,
						TransferType: registrasi.SurveyCommitteeTransfer, CreatedAt: now,
					})
				}
				if err := l.committees.Save(ctx, c); err != nil {
					return err
				}
				note := p.Note
				note.AnalysisType = "1"
				if note.Date.IsZero() {
					note.Date = now
				}
				if err := store.SaveCommitteeNote(ctx, id, note); err != nil {
					return err
				}
				result.Committee = append(result.Committee, id)
			}
			// Langkah 32: objek terpilih bukan internal menjadi Menunggu Persetujuan.
			for i := range objects {
				o := &objects[i]
				if !o.Selected || (o.Status == "" && o.SurveyorName == "" && o.SurveyorType != registrasi.SurveyorLossAdjuster) {
					continue
				}
				if o.SurveyorType != registrasi.SurveyorInternal {
					o.Status = registrasi.SurveyAwaitingApproval
				}
			}
		}
		claim.UpdatedBy, claim.UpdatedAt = by.Identity, now
		if err := store.SaveObjects(ctx, claim.ID, objects, by.Identity, now); err != nil {
			return err
		}
		if err := l.claim.Save(ctx, claim); err != nil {
			return err
		}
		if err := l.mirrorInbox(ctx, claim); err != nil {
			return err
		}
		path := "MANUAL — komite " + strings.Join(result.Committee, ", ")
		if auto {
			path = "AUTO"
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "SURVEY_TRANSFER_KOMITE", Actor: by.Identity, At: now,
			Note: "Transfer Komite survey (" + path + ") — survey " + strings.Join(result.Surveys, ", "),
		})
	})
	if err != nil {
		return SurveyResult{}, err
	}
	result.View, err = l.surveyView(ctx, store, claim, by)
	return result, err
}

// CancelSurvey adalah tombol "Ya" pada modal Batal Survei (`CancelSurvey`).
func (l *Service) CancelSurvey(ctx context.Context, p SurveyCancelCommand, by Caller) (SurveyTabView, error) {
	store, claim, err := l.surveyContext(ctx, p.ClaimID, p.TaskID, by)
	if err != nil {
		return SurveyTabView{}, err
	}
	objects, err := store.Objects(ctx, claim.ID)
	if err != nil {
		return SurveyTabView{}, err
	}
	index := -1
	for i, o := range objects {
		if o.ObjectID == strings.TrimSpace(p.ObjectID) {
			index = i
		}
	}
	if index < 0 {
		return SurveyTabView{}, fmt.Errorf("%w: objek %s tidak ada", registrasi.ErrInvalidAction, p.ObjectID)
	}
	if objects[index].Status != registrasi.SurveyInProgress {
		return SurveyTabView{}, ErrSurveyNotInProgress
	}
	surveyID := objects[index].SurveyID
	registrasi.CancelSurveyObject(objects, index, claim.Policy.Line == registrasi.LinePersonalAccident)
	now := l.clock.Now().UTC()
	if err := l.unit.Run(ctx, func(ctx context.Context) error {
		// Langkah 6–7: kasus survey ditutup (survey Marine tidak ikut ditutup, seperti Pega).
		if surveyID != "" {
			if err := store.Cancel(ctx, surveyID, now); err != nil {
				return err
			}
		}
		if err := store.SaveObjects(ctx, claim.ID, objects, by.Identity, now); err != nil {
			return err
		}
		return l.audit.Record(ctx, registrasi.AuditTrail{
			ClaimID: claim.ID, ClaimNumber: claim.Number, Event: "SURVEY_BATAL", Actor: by.Identity, At: now,
			Note: "Batal Survei — survey " + surveyID,
		})
	}); err != nil {
		return SurveyTabView{}, err
	}
	return l.surveyView(ctx, store, claim, by)
}
