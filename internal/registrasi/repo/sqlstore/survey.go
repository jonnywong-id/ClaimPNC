package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// SurveyStore menyimpan tab Survey (registrasi.SurveyStore) — lihat survey.sql.
type SurveyStore struct{ db *sql.DB }

// NewSurveyStore membentuk penyimpanan tab Survey.
func NewSurveyStore(db *sql.DB) *SurveyStore { return &SurveyStore{db: db} }

func (s *SurveyStore) exec(ctx context.Context) executor { return executorFrom(ctx, s.db) }

// Objects membaca objek klaim beserta isian surveynya.
func (s *SurveyStore) Objects(ctx context.Context, claimID string) ([]registrasi.SurveyObject, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery("survey_objek_daftar"), strings.TrimSpace(claimID))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca objek survey dari POOLDATA.T_CLAIM_OBJECTLIST: %w", err)
	}
	defer rows.Close()
	var out []registrasi.SurveyObject
	for rows.Next() {
		var seq sql.NullInt64
		var id, name, loc, pick, sloc, stype, sname, slogin, saddr, semail, bcode, bname, mname, mlogin, status, sid, sidm sql.NullString
		if err := rows.Scan(&seq, &id, &name, &loc, &pick, &sloc, &stype, &sname, &slogin, &saddr, &semail, &bcode,
			&bname, &mname, &mlogin, &status, &sid, &sidm); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris objek survey: %w", err)
		}
		out = append(out, registrasi.SurveyObject{
			Seq: int(seq.Int64), ObjectID: trimmed(id), Name: trimmed(name), Location: trimmed(loc),
			Selected: trimmed(pick) == "1", SurveyLocation: trimmed(sloc), SurveyorType: trimmed(stype),
			SurveyorName: trimmed(sname), SurveyorLogin: trimmed(slogin), SurveyorAddr: trimmed(saddr),
			SurveyorEmail: trimmed(semail), BranchCode: trimmed(bcode), BranchName: trimmed(bname),
			MarineName: trimmed(mname), MarineLogin: trimmed(mlogin), Status: trimmed(status),
			SurveyID: trimmed(sid), SurveyIDMarine: trimmed(sidm),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: menelusuri objek survey: %w", err)
	}
	return out, nil
}

// SaveObjects menulis kolom survey setiap objek.
func (s *SurveyStore) SaveObjects(ctx context.Context, claimID string, objects []registrasi.SurveyObject, by string, at time.Time) error {
	exec := s.exec(ctx)
	for _, o := range objects {
		pick := "0"
		if o.Selected {
			pick = "1"
		}
		var approve any
		switch o.Status {
		case registrasi.SurveyInProgress:
			approve = 1
		case registrasi.SurveyRejected:
			approve = 0
		}
		if _, err := exec.ExecContext(ctx, loadQuery("survey_objek_simpan"),
			pick, emptyTextAsNil(o.SurveyLocation), emptyTextAsNil(o.SurveyorType), emptyTextAsNil(o.SurveyorName),
			emptyTextAsNil(o.SurveyorLogin), emptyTextAsNil(o.SurveyorAddr), emptyTextAsNil(o.SurveyorEmail),
			emptyTextAsNil(o.BranchCode), emptyTextAsNil(o.BranchName), emptyTextAsNil(o.MarineName),
			emptyTextAsNil(o.MarineLogin), emptyTextAsNil(o.Status), emptyTextAsNil(o.SurveyID),
			emptyTextAsNil(o.SurveyIDMarine), approve, emptyTextAsNil(by), timeOrNil(at),
			strings.TrimSpace(claimID), o.ObjectID); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan survey objek %s di POOLDATA.T_CLAIM_OBJECTLIST: %w", o.ObjectID, err)
		}
	}
	return nil
}

func (s *SurveyStore) surveyors(ctx context.Context, name string, args ...any) ([]registrasi.SurveyorOption, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery(name), args...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca surveyor dari POOLDATA.V_D_SURVEYORS: %w", err)
	}
	defer rows.Close()
	var out []registrasi.SurveyorOption
	for rows.Next() {
		var id, name, login, addr, branch, bname, email, contact sql.NullString
		if err := rows.Scan(&id, &name, &login, &addr, &branch, &bname, &email, &contact); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris surveyor: %w", err)
		}
		out = append(out, registrasi.SurveyorOption{
			ID: trimmed(id), Name: trimmed(name), Login: trimmed(login), Address: trimmed(addr),
			Branch: trimmed(branch), BranchName: trimmed(bname), Email: trimmed(email), Contact: trimmed(contact),
		})
	}
	return out, rows.Err()
}

// Surveyors membaca pilihan surveyor satu tipe (`ChooseSurveyorType_act`).
func (s *SurveyStore) Surveyors(ctx context.Context, surveyorType string) ([]registrasi.SurveyorOption, error) {
	t := strings.TrimSpace(surveyorType)
	if t == "" || t == registrasi.SurveyorInternal {
		return s.surveyors(ctx, "survey_surveyor_internal")
	}
	master, ok := registrasi.SurveyorMasterType[t]
	if !ok {
		return nil, nil
	}
	return s.surveyors(ctx, "survey_surveyor_tipe", master)
}

// NominatedOptions membaca pilihan Nominated Loss Adjuster.
func (s *SurveyStore) NominatedOptions(ctx context.Context) ([]registrasi.SurveyorOption, error) {
	return s.surveyors(ctx, "survey_surveyor_nominasi")
}

// NextSurveyID menerbitkan nomor survey SRVN.YY.n untuk tahun (WIB) dari at.
func (s *SurveyStore) NextSurveyID(ctx context.Context, at time.Time) (string, error) {
	year := clock.DateWIB(at).Year()
	var next int64
	pattern := fmt.Sprintf("%s.%02d.%%", registrasi.SurveyCasePrefix, year%100)
	if err := s.exec(ctx).QueryRowContext(ctx, loadQuery("survey_nomor_berikut"), pattern).Scan(&next); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: menerbitkan nomor survey dari POOLDATA.T_SURVEYORLIST: %w", err)
	}
	return registrasi.FormatSurveyID(year, next), nil
}

// InsertRecord menyisipkan satu baris T_SURVEYORLIST.
func (s *SurveyStore) InsertRecord(ctx context.Context, r registrasi.SurveyRecord) error {
	if _, err := s.exec(ctx).ExecContext(ctx, loadQuery("survey_sisip"),
		r.CaseID, r.ClaimID, emptyTextAsNil(r.SurveyType), emptyTextAsNil(r.SurveyorName), timeOrNil(r.SurveyDate),
		emptyTextAsNil(r.SurveyLocation), emptyTextAsNil(r.ObjectName), emptyTextAsNil(r.ObjectLocation),
		emptyTextAsNil(r.ObjectID), emptyTextAsNil(r.Index), timeOrNil(r.TreatmentDate), emptyTextAsNil(r.ObjectName),
		emptyTextAsNil(r.Status), timeOrNil(r.At), emptyTextAsNil(r.SurveyLocation), timeOrNil(r.SurveyDate),
		emptyTextAsNil(r.MarineName)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyisipkan survey %s ke POOLDATA.T_SURVEYORLIST: %w", r.CaseID, err)
	}
	return nil
}

// Rows membaca survey tersimpan klaim.
func (s *SurveyStore) Rows(ctx context.Context, keys registrasi.RecordKeys) ([]registrasi.SurveyRow, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery("survey_baris_klaim"), keyArgs(keys)...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca survey klaim dari POOLDATA.T_SURVEYORLIST: %w", err)
	}
	defer rows.Close()
	var out []registrasi.SurveyRow
	for rows.Next() {
		var id, obj, name, status, work sql.NullString
		if err := rows.Scan(&id, &obj, &name, &status, &work); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris survey klaim: %w", err)
		}
		out = append(out, registrasi.SurveyRow{CaseID: trimmed(id), ObjectID: trimmed(obj), SurveyorName: trimmed(name),
			Status: trimmed(status), WorkStatus: trimmed(work)})
	}
	return out, rows.Err()
}

// Cancel menutup satu survey: STS_SURVEY "Batal Survey", PYSTATUSWORK Resolved-Rejected.
func (s *SurveyStore) Cancel(ctx context.Context, caseID string, _ time.Time) error {
	if _, err := s.exec(ctx).ExecContext(ctx, loadQuery("survey_batal"),
		registrasi.SurveyRecordCancelled, registrasi.SurveyWorkRejected, strings.TrimSpace(caseID)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: membatalkan survey %s di POOLDATA.T_SURVEYORLIST: %w", caseID, err)
	}
	return nil
}

// DirectorSurveyor menyatakan salah satu surveyor ber-TRFKOMITE 1 (`SearchCodeDireksi_SQL`).
func (s *SurveyStore) DirectorSurveyor(ctx context.Context, names []string) (bool, error) {
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			continue
		}
		var count int
		if err := s.exec(ctx).QueryRowContext(ctx, loadQuery("survey_direksi"), "%"+strings.TrimSpace(n)+"%").Scan(&count); err != nil {
			return false, fmt.Errorf("registrasi/sqlstore: membaca TRFKOMITE dari POOLDATA.V_D_SURVEYORS: %w", err)
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

// Limit membaca batas jalur otomatis (`GetLimitSurvey`); ok false bila tidak ada baris.
func (s *SurveyStore) Limit(ctx context.Context, businessCode, syariah string) (registrasi.Money, bool, error) {
	var limit sql.NullFloat64
	err := s.exec(ctx).QueryRowContext(ctx, loadQuery("survey_limit"), strings.TrimSpace(businessCode), strings.TrimSpace(syariah)).Scan(&limit)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !limit.Valid) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("registrasi/sqlstore: membaca POOLDATA.LIMIT_LOSSADJUSTER: %w", err)
	}
	return registrasi.Money(limit.Float64 * 100), true, nil
}

// CommitteeMembers membaca anggota komite survey (`EmailKomiteSurvey_sql`).
func (s *SurveyStore) CommitteeMembers(ctx context.Context, line string, belowDirector bool) ([]registrasi.SurveyCommitteeMember, error) {
	name := "survey_komite_anggota"
	if belowDirector {
		name = "survey_komite_anggota_bawah"
	}
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery(name), line)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca anggota komite survey dari POOLDATA.EMAILKOMITE: %w", err)
	}
	defer rows.Close()
	var out []registrasi.SurveyCommitteeMember
	for rows.Next() {
		var op, email sql.NullString
		if err := rows.Scan(&op, &email); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris anggota komite survey: %w", err)
		}
		out = append(out, registrasi.SurveyCommitteeMember{OperatorID: trimmed(op), Email: trimmed(email)})
	}
	return out, rows.Err()
}

// SaveNominated menyisipkan daftar Nominated Loss Adjuster.
func (s *SurveyStore) SaveNominated(ctx context.Context, claimID string, list []registrasi.SurveyorOption) error {
	for _, n := range list {
		if _, err := s.exec(ctx).ExecContext(ctx, loadQuery("survey_nominasi_sisip"),
			strings.TrimSpace(claimID), emptyTextAsNil(n.ID), emptyTextAsNil(n.Name), emptyTextAsNil(n.Login)); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan nominasi ke POOLDATA.NOMINATE_LOSSADJUSTER_PNC: %w", err)
		}
	}
	return nil
}

// SaveCommitteeNote menulis isian modal Transfer Komite ke kepala kasus komite survey.
func (s *SurveyStore) SaveCommitteeNote(ctx context.Context, committeeID string, n registrasi.SurveyCommitteeNote) error {
	if _, err := s.exec(ctx).ExecContext(ctx, loadQuery("survey_komite_catatan"),
		timeOrNil(n.Date), emptyTextAsNil(n.Initial), emptyTextAsNil(n.AnalysisType), emptyTextAsNil(n.Circumstances),
		emptyTextAsNil(n.Nominated), emptyTextAsNil(n.Remarks), emptyTextAsNil(n.Company), emptyTextAsNil(n.ContactPerson),
		emptyTextAsNil(n.OfficePhone), emptyTextAsNil(n.Email), committeeID); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan isian komite survey %s di POOLDATA.TC_PNC_KOMITE: %w", committeeID, err)
	}
	return nil
}

// Request membaca Permintaan Survey terbaru klaim.
func (s *SurveyStore) Request(ctx context.Context, keys registrasi.RecordKeys) (registrasi.SurveyRequest, bool, error) {
	var name, loc, phone, branch, surveyor, email, object sql.NullString
	var date sql.NullTime
	err := s.exec(ctx).QueryRowContext(ctx, loadQuery("survey_permintaan"), keyArgs(keys)...).
		Scan(&name, &loc, &phone, &date, &branch, &surveyor, &email, &object)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.SurveyRequest{}, false, nil
	}
	if err != nil {
		return registrasi.SurveyRequest{}, false, fmt.Errorf("registrasi/sqlstore: membaca POOLDATA.T_REQ_SURVEY: %w", err)
	}
	return registrasi.SurveyRequest{RequestorName: trimmed(name), Location: trimmed(loc), Phone: trimmed(phone),
		Date: date.Time, Branch: trimmed(branch), Surveyor: trimmed(surveyor), Email: trimmed(email),
		ObjectName: trimmed(object)}, true, nil
}

var _ registrasi.SurveyStore = (*SurveyStore)(nil)
