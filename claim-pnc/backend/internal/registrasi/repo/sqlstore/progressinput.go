package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// ProgressInputStore memenuhi registrasi.ProgressInputStore — modal Input Progress Claim.
type ProgressInputStore struct{ db *sql.DB }

// NewProgressInputStore membentuk penyimpanan modal Input Progress Claim.
func NewProgressInputStore(db *sql.DB) *ProgressInputStore { return &ProgressInputStore{db: db} }

func (s *ProgressInputStore) exec(ctx context.Context) executor { return executorFrom(ctx, s.db) }

// Positions membaca posisi progres klaim (GCNM_PROGRESS_POSISI_PNC).
func (s *ProgressInputStore) Positions(ctx context.Context, claimNumber string) ([]registrasi.ProgressPosition, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery("progres_input_posisi"), strings.TrimSpace(claimNumber))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel GCNM_PROGRESS_POSISI_PNC: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.ProgressPosition
	for rows.Next() {
		var id, caseID, name, status sql.NullString
		var at sql.NullTime
		if err := rows.Scan(&id, &caseID, &name, &status, &at); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel GCNM_PROGRESS_POSISI_PNC: %w", err)
		}
		p := registrasi.ProgressPosition{ID: trimmed(id), CaseID: trimmed(caseID), Name: trimmed(name), Status: trimmed(status)}
		if at.Valid {
			p.Date = wibDate(at)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProgressInputStore) options(ctx context.Context, query, table string, args ...any) ([]registrasi.ProgressOption, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery(query), args...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.ProgressOption
	for rows.Next() {
		var id, name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel %s: %w", table, err)
		}
		out = append(out, registrasi.ProgressOption{ID: trimmed(id), Name: trimmed(name)})
	}
	return out, rows.Err()
}

func (s *ProgressInputStore) texts(ctx context.Context, query, table string, args ...any) ([]string, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery(query), args...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca tabel %s: %w", table, err)
		}
		if t := trimmed(v); t != "" {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// Status1Options membaca Status Progress 1.
func (s *ProgressInputStore) Status1Options(ctx context.Context) ([]registrasi.ProgressOption, error) {
	return s.options(ctx, "progres_input_status1", "GCNM_MST_PROGRESS_KLAIM")
}

// StatusOptions membaca pilihan dropdown Status.
func (s *ProgressInputStore) StatusOptions(ctx context.Context) ([]string, error) {
	return s.texts(ctx, "progres_input_status", "GCNM_PROGRESS_CLAIM")
}

// DocTypes membaca pilihan Tipe Dokumen.
func (s *ProgressInputStore) DocTypes(ctx context.Context) ([]string, error) {
	return s.texts(ctx, "progres_input_tipe_dokumen", "GCNM_MST_PROGRESS")
}

// NeedOptions membaca STATUS PERLU satu Status Progress 1.
func (s *ProgressInputStore) NeedOptions(ctx context.Context, progress1, docType string) ([]registrasi.ProgressOption, error) {
	if t := strings.TrimSpace(docType); t != "" {
		return s.options(ctx, "progres_input_perlu_tipe", "GCNM_MST_PROGRESS", strings.TrimSpace(progress1), t)
	}
	return s.options(ctx, "progres_input_perlu", "GCNM_MST_PROGRESS", strings.TrimSpace(progress1))
}

// NeedName membaca STS_PROGRESS2 satu ID_MST.
func (s *ProgressInputStore) NeedName(ctx context.Context, id string) (string, bool, error) {
	var name sql.NullString
	err := s.exec(ctx).QueryRowContext(ctx, loadQuery("progres_input_nama_perlu"), strings.TrimSpace(id)).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("registrasi/sqlstore: membaca tabel GCNM_MST_PROGRESS: %w", err)
	}
	return trimmed(name), true, nil
}

// InputToday membaca STATUS PROGRESS 2 progres non-AUTO pada tanggal kalender WIB `day`.
func (s *ProgressInputStore) InputToday(ctx context.Context, claimNumber string, day time.Time) ([]string, error) {
	w := day.In(clock.ZoneWIB)
	start := time.Date(w.Year(), w.Month(), w.Day(), 0, 0, 0, 0, clock.ZoneWIB)
	return s.texts(ctx, "progres_input_hari_ini", "GCNM_PROGRESS_CLAIM",
		strings.TrimSpace(claimNumber), wallOrNil(start), wallOrNil(start.AddDate(0, 0, 1)))
}

// progressObject adalah satu baris ObjectList JSONSTATUS_PROGRESS2 — bentuk
// `@GCNM.GetPageJSONString()` atas TempInputStatusProgress2. GET_POSISI_PROGRESS2 hanya
// membaca BranchName.
type progressObject struct {
	BranchName string `json:"BranchName"`
	SerialNo   string `json:"SerialNo"`
	ObjClass   string `json:"pxObjClass"`
}

type progressPage struct {
	ObjClass   string           `json:"pxObjClass"`
	ObjectList []progressObject `json:"ObjectList"`
}

// progressNeedsJSON membentuk JSONSTATUS_PROGRESS2 dari baris STATUS PERLU.
func progressNeedsJSON(needs []registrasi.ProgressNeed) (string, error) {
	page := progressPage{ObjClass: "ASM-FW-GCNMFW-Data-ClaimData", ObjectList: []progressObject{}}
	for _, n := range needs {
		page.ObjectList = append(page.ObjectList, progressObject{BranchName: n.ID, SerialNo: n.Name, ObjClass: "ASM-FW-GCNMFW-Data-Object"})
	}
	raw, err := json.Marshal(page)
	return string(raw), err
}

// Insert menulis satu baris progres (PROGRESS_CLAIM_PNC cabang ELSE).
func (s *ProgressInputStore) Insert(ctx context.Context, p registrasi.ProgressInput) error {
	exec := s.exec(ctx)
	user, err := legacyUser(ctx, exec, p.User)
	if err != nil {
		return err
	}
	doc, err := progressNeedsJSON(p.Needs)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membentuk JSONSTATUS_PROGRESS2: %w", err)
	}
	follow := p.NextFollowUp
	if follow.IsZero() {
		follow = p.At.Add(registrasi.ProgressFollowUpDefault)
	}
	if _, err := exec.ExecContext(ctx, loadQuery("progres_input_sisip"),
		p.ClaimNumber, strings.TrimSpace(p.Remarks), p.Progress1, p.LastNeed().ID, strings.TrimSpace(p.Status),
		wallOrNil(follow), user, p.PositionID, doc, emptyTextAsNil(p.DocType), p.ClaimNumber); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan ke tabel GCNM_PROGRESS_CLAIM: %w", err)
	}
	return nil
}

// BrokerEmail membaca Email Progress Klaim yang tersimpan untuk polis.
func (s *ProgressInputStore) BrokerEmail(ctx context.Context, b registrasi.BrokerEmail) (string, error) {
	var email sql.NullString
	err := s.exec(ctx).QueryRowContext(ctx, loadQuery("progres_input_email_broker"),
		b.PolicyNumber, b.GroupPanel, b.BusinessCode, b.BranchCode).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca tabel MST_EMAIL_BROKER_KLAIM: %w", err)
	}
	return trimmed(email), nil
}

// SaveBrokerEmail adalah INSERTDATABROKEREMAILMASTER: sisip bila belum ada baris polis, ubah
// bila sudah. Seperti prosedurnya, tidak ada yang ditulis bila agen sumber bisnis polis tidak
// tepat satu baris (NO_DATA_FOUND / TOO_MANY_ROWS jatuh ke handler galatnya).
func (s *ProgressInputStore) SaveBrokerEmail(ctx context.Context, b registrasi.BrokerEmail) error {
	email := strings.TrimSpace(b.Email)
	if email == "" {
		return nil
	}
	exec := s.exec(ctx)
	var count int
	if err := exec.QueryRowContext(ctx, loadQuery("progres_input_broker_hitung"),
		b.PolicyNumber, b.GroupPanel, b.BusinessCode, b.BranchCode).Scan(&count); err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca tabel MST_EMAIL_BROKER_KLAIM: %w", err)
	}
	rows, err := exec.QueryContext(ctx, loadQuery("progres_input_broker_agen"), b.PolicyNumber)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca tabel AGENT: %w", err)
	}
	var leader, sumbis sql.NullString
	found := 0
	for rows.Next() {
		if err := rows.Scan(&leader, &sumbis); err != nil {
			_ = rows.Close()
			return fmt.Errorf("registrasi/sqlstore: membaca tabel AGENT: %w", err)
		}
		found++
	}
	_ = rows.Close()
	if found != 1 {
		return nil
	}
	if count <= 0 {
		if _, err := exec.ExecContext(ctx, loadQuery("progres_input_broker_sisip"), b.PolicyNumber, b.BranchCode,
			b.BusinessCode, b.GroupPanel, email, b.ClaimID, emptyTextAsNil(trimmed(leader)), emptyTextAsNil(trimmed(sumbis))); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan ke tabel MST_EMAIL_BROKER_KLAIM: %w", err)
		}
		return nil
	}
	if _, err := exec.ExecContext(ctx, loadQuery("progres_input_broker_ubah"), email, b.PolicyNumber, b.GroupPanel,
		b.BusinessCode, b.BranchCode, trimmed(leader), trimmed(sumbis)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan ke tabel MST_EMAIL_BROKER_KLAIM: %w", err)
	}
	return nil
}

var _ registrasi.ProgressInputStore = (*ProgressInputStore)(nil)
