package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ClaimRecords membaca catatan pendamping klaim dari tabel warisan.
type ClaimRecords struct {
	db *sql.DB
}

// NewClaimRecords membentuk pembaca catatan pendamping klaim.
func NewClaimRecords(db *sql.DB) *ClaimRecords { return &ClaimRecords{db: db} }

func keyArgs(k registrasi.RecordKeys) []any {
	return []any{k.Number, k.ID, k.Prefixed}
}

func trimmed(s sql.NullString) string { return strings.TrimSpace(s.String) }

// Surveys membaca hasil survey klaim.
func (r *ClaimRecords) Surveys(ctx context.Context, keys registrasi.RecordKeys) ([]registrasi.Survey, error) {
	rows, err := r.db.QueryContext(ctx, loadQuery("survey_daftar"), keyArgs(keys)...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca survey: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.Survey
	for rows.Next() {
		var caseID, kind, name, surveyLoc, object, objectLoc, index, status, note sql.NullString
		var date, input sql.NullTime
		if err := rows.Scan(&caseID, &kind, &name, &date, &surveyLoc, &object, &objectLoc,
			&index, &status, &note, &input); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris survey: %w", err)
		}
		s := registrasi.Survey{
			CaseID: trimmed(caseID), Type: trimmed(kind), SurveyorName: trimmed(name),
			SurveyLocation: trimmed(surveyLoc), ObjectName: trimmed(object), ObjectLocation: trimmed(objectLoc),
			Index: trimmed(index), Status: trimmed(status), Note: trimmed(note),
		}
		if date.Valid {
			s.Date = date.Time
		}
		if input.Valid {
			s.InputDate = input.Time
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// DocumentTypes membaca jenis dokumen satu lini bisnis beserta coverage pewajibnya.
func (r *ClaimRecords) DocumentTypes(ctx context.Context, businessCode string) ([]registrasi.DocumentType, error) {
	code := strings.TrimSpace(businessCode)
	if code == "" {
		return nil, nil
	}

	coverage := map[string][]string{}
	crows, err := r.db.QueryContext(ctx, loadQuery("dokumen_coverage_daftar"), code)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca coverage dokumen: %w", err)
	}
	for crows.Next() {
		var id, cov sql.NullString
		if err := crows.Scan(&id, &cov); err != nil {
			_ = crows.Close()
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris coverage dokumen: %w", err)
		}
		coverage[trimmed(id)] = append(coverage[trimmed(id)], trimmed(cov))
	}
	if err := crows.Err(); err != nil {
		_ = crows.Close()
		return nil, fmt.Errorf("registrasi/sqlstore: menelusuri coverage dokumen: %w", err)
	}
	_ = crows.Close()

	rows, err := r.db.QueryContext(ctx, loadQuery("dokumen_jenis_daftar"), code)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca jenis dokumen: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.DocumentType
	for rows.Next() {
		var category, categoryID, id, name, required, objectDoc, minDoc sql.NullString
		if err := rows.Scan(&category, &categoryID, &id, &name, &required, &objectDoc, &minDoc); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris jenis dokumen: %w", err)
		}
		result = append(result, registrasi.DocumentType{
			Category: trimmed(category), CategoryID: trimmed(categoryID), ID: trimmed(id),
			Name: trimmed(name), RequiredRaw: trimmed(required), ObjectDocID: trimmed(objectDoc),
			MinDoc: trimmed(minDoc), Coverage: coverage[trimmed(id)],
		})
	}
	return result, rows.Err()
}

// Attachments membaca daftar berkas yang sudah diunggah untuk klaim.
func (r *ClaimRecords) Attachments(ctx context.Context, keys registrasi.RecordKeys) ([]registrasi.Attachment, error) {
	rows, err := r.db.QueryContext(ctx, loadQuery("lampiran_daftar"), keyArgs(keys)...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca lampiran: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.Attachment
	for rows.Next() {
		var id, name, mime, note, category, sub, image, by sql.NullString
		var at sql.NullTime
		if err := rows.Scan(&id, &name, &mime, &note, &category, &sub, &image, &by, &at); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris lampiran: %w", err)
		}
		a := registrasi.Attachment{
			ID: trimmed(id), Name: trimmed(name), MimeType: trimmed(mime), Note: trimmed(note),
			Category: trimmed(category), SubCategory: trimmed(sub), ImageID: trimmed(image), UploadedBy: trimmed(by),
		}
		if at.Valid {
			a.UploadedAt = at.Time
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// Progress membaca riwayat progres klaim, terbaru lebih dulu.
func (r *ClaimRecords) Progress(ctx context.Context, keys registrasi.RecordKeys) ([]registrasi.ProgressEntry, error) {
	rows, err := r.db.QueryContext(ctx, loadQuery("progres_daftar"), keyArgs(keys)...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca progres: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.ProgressEntry
	for rows.Next() {
		var seq sql.NullInt64
		var s1, s1Name, s2, s2Name, note, by, position sql.NullString
		var at, next sql.NullTime
		if err := rows.Scan(&seq, &at, &s1, &s1Name, &s2, &s2Name, &note, &next, &by, &position); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris progres: %w", err)
		}
		e := registrasi.ProgressEntry{
			Seq: int(seq.Int64), Status1: trimmed(s1), Status1Name: trimmed(s1Name),
			Status2: trimmed(s2), Status2Name: trimmed(s2Name), Note: trimmed(note),
			InputBy: trimmed(by), PositionID: trimmed(position),
		}
		if at.Valid {
			e.InputAt = at.Time
		}
		if next.Valid {
			e.NextFollowUp = next.Time
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// Communications membaca percakapan klaim, terbaru lebih dulu.
func (r *ClaimRecords) Communications(ctx context.Context, keys registrasi.RecordKeys) ([]registrasi.Communication, error) {
	args := append(keyArgs(keys), keyArgs(keys)...)
	rows, err := r.db.QueryContext(ctx, loadQuery("komunikasi_daftar"), args...)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca komunikasi: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.Communication
	for rows.Next() {
		var caseID, sender, senderName, message, reply, replier, status sql.NullString
		var id sql.NullInt64
		var sent, replied sql.NullTime
		if err := rows.Scan(&caseID, &id, &sent, &sender, &senderName, &message, &reply,
			&replier, &replied, &status); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris komunikasi: %w", err)
		}
		c := registrasi.Communication{
			CaseID: trimmed(caseID), Sender: trimmed(sender), SenderName: trimmed(senderName),
			Message: trimmed(message), Reply: trimmed(reply), ReplierName: trimmed(replier), Status: trimmed(status),
		}
		if id.Valid {
			c.ID = strconv.FormatInt(id.Int64, 10)
		}
		if sent.Valid {
			c.SentAt = sent.Time
		}
		if replied.Valid {
			c.RepliedAt = replied.Time
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
