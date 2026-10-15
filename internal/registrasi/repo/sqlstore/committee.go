package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// CommitteeStore menyimpan kasus komite klaim PNCN di POOLDATA.T_CLAIM_KOMITE_LIST.
type CommitteeStore struct{ db *sql.DB }

// NewCommitteeStore membentuk penyimpan kasus komite di atas sebuah koneksi.
func NewCommitteeStore(db *sql.DB) *CommitteeStore { return &CommitteeStore{db: db} }

func (s *CommitteeStore) exec(ctx context.Context) executor { return executorFrom(ctx, s.db) }

// NextCaseID menerbitkan KOMITE_ID berikutnya: `KMTN.YY.n`, deret per tahun WIB.
//
// Tidak ada sequence: nomor diambil dari nomor terbesar tahun itu di TC_PNC_KOMITE, di dalam
// transaksi pemanggil — pola yang sama dengan PNCN dan RCVN. Dua transfer serentak dapat
// memperoleh nomor yang sama; primary key KOMITE_ID membuat yang kedua gagal (transaksinya
// dibatalkan seluruhnya) alih-alih menyimpan nomor ganda.
func (s *CommitteeStore) NextCaseID(ctx context.Context, at time.Time) (string, error) {
	year := clock.DateWIB(at).Year()
	pattern := fmt.Sprintf("%s.%02d.%%", registrasi.CommitteeCasePrefix, year%100)
	var next int64
	if err := s.exec(ctx).QueryRowContext(ctx, loadQuery("komite_nomor_berikut"), pattern).Scan(&next); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: menerbitkan nomor komite: %w", err)
	}
	return registrasi.FormatCommitteeCaseID(year, next)
}

// Save menulis kepala kasus ke TC_PNC_KOMITE, lalu seluruh anggotanya ke T_CLAIM_KOMITE_LIST:
// perbarui-atau-sisip per KOMITE_ID dan per (KOMITE_ID, NAMAKOMITE, KOMITEKE).
func (s *CommitteeStore) Save(ctx context.Context, c registrasi.CommitteeCase) error {
	exec := s.exec(ctx)
	head := []any{
		c.ClaimID, c.ClaimNumber, c.ObjectID, strconv.Itoa(c.CoverageSeq), strconv.Itoa(c.AdjustmentSeq),
		emptyTextAsNil(c.PaymentType), emptyTextAsNil(c.TransferType), emptyTextAsNil(c.Line), emptyTextAsNil(c.Band),
		emptyTextAsNil(c.Currency), int64(c.Rate), int64(c.AdjustmentValue), int64(c.Value),
		len(c.Members), c.Level(), emptyTextAsNil(c.Outcome()), c.Status(), emptyTextAsNil(c.Applicant),
		timeOrNil(c.CreatedAt), emptyTextAsNil(c.CreatedBy), timeOrNil(c.UpdatedAt), emptyTextAsNil(c.UpdatedBy),
		timeOrNil(c.DecidedAt), c.ID,
	}
	if err := upsert(ctx, exec, "komite_kepala_perbarui", head, "komite_kepala_sisip", head); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan kepala komite %s: %w", c.ID, err)
	}
	for _, m := range c.Members {
		values := []any{
			c.ClaimNumber, m.CaseStatus, m.Decision, emptyTextAsNil(m.Note), timeOrNil(m.CreatedAt),
			timeOrNil(m.DecidedAt), m.TransferType, emptyTextAsNil(m.PaymentType), int64(m.ShareASM), int64(m.Value),
			c.ID, m.Operator, strconv.Itoa(m.Level),
		}
		if err := upsert(ctx, exec, "komite_perbarui", values, "komite_sisip", values); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menyimpan komite %s jenjang %d: %w", c.ID, m.Level, err)
		}
	}
	return nil
}

// Get membaca satu kasus komite: kepala dan anggotanya.
func (s *CommitteeStore) Get(ctx context.Context, caseID string) (registrasi.CommitteeCase, error) {
	id := strings.TrimSpace(caseID)
	var (
		key, claimID, number, objectID, coverage, adjustment, payment, transfer, line, band sql.NullString
		currency, applicant, createdBy, updatedBy                                           sql.NullString
		rate, adjValue, value                                                               sql.NullInt64
		created, updated, decided                                                           sql.NullTime
	)
	err := s.exec(ctx).QueryRowContext(ctx, loadQuery("komite_kepala_ambil"), id).Scan(
		&key, &claimID, &number, &objectID, &coverage, &adjustment, &payment, &transfer, &line, &band,
		&currency, &rate, &adjValue, &value, &applicant, &created, &createdBy, &updated, &updatedBy, &decided)
	if errors.Is(err, sql.ErrNoRows) {
		return registrasi.CommitteeCase{}, registrasi.ErrCommitteeNotFound
	}
	if err != nil {
		return registrasi.CommitteeCase{}, fmt.Errorf("registrasi/sqlstore: membaca kepala komite: %w", err)
	}
	members, err := s.query(ctx, "komite_ambil", id)
	if err != nil {
		return registrasi.CommitteeCase{}, err
	}
	coverageSeq, _ := strconv.Atoi(trimmed(coverage))
	adjustmentSeq, _ := strconv.Atoi(trimmed(adjustment))
	return registrasi.CommitteeCase{
		ID: trimmed(key), ClaimID: trimmed(claimID), ClaimNumber: trimmed(number), ObjectID: trimmed(objectID),
		CoverageSeq: coverageSeq, AdjustmentSeq: adjustmentSeq, PaymentType: trimmed(payment),
		TransferType: trimmed(transfer), Line: trimmed(line), Band: trimmed(band), Currency: trimmed(currency),
		Rate: registrasi.ExchangeRate(rate.Int64), AdjustmentValue: registrasi.Money(adjValue.Int64),
		Value: registrasi.Money(value.Int64), Applicant: trimmed(applicant),
		CreatedAt: created.Time, CreatedBy: trimmed(createdBy), UpdatedAt: updated.Time, UpdatedBy: trimmed(updatedBy),
		DecidedAt: decided.Time, Members: members,
	}, nil
}

// Pending mengembalikan baris anggota yang sedang ditunggu dan milik operator itu.
func (s *CommitteeStore) Pending(ctx context.Context, operator string) ([]registrasi.CommitteeMember, error) {
	return s.query(ctx, "komite_tertunda", operator)
}

func (s *CommitteeStore) query(ctx context.Context, name string, arg string) ([]registrasi.CommitteeMember, error) {
	rows, err := s.exec(ctx).QueryContext(ctx, loadQuery(name), arg)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca komite: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []registrasi.CommitteeMember
	for rows.Next() {
		var (
			id, claim, operator, level, decision, caseStatus, note, transfer, payment sql.NullString
			share, value                                                              sql.NullInt64
			created, decided                                                          sql.NullTime
		)
		if err := rows.Scan(&id, &claim, &operator, &level, &decision, &caseStatus, &note, &transfer,
			&payment, &share, &value, &created, &decided); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris komite: %w", err)
		}
		n, _ := strconv.Atoi(strings.TrimSpace(level.String))
		result = append(result, registrasi.CommitteeMember{
			CaseID: trimmed(id), ClaimNumber: trimmed(claim), Operator: trimmed(operator), Level: n,
			Decision: trimmed(decision), CaseStatus: trimmed(caseStatus), Note: note.String,
			TransferType: trimmed(transfer), PaymentType: trimmed(payment),
			ShareASM: registrasi.Percent(share.Int64), Value: registrasi.Money(value.Int64),
			CreatedAt: created.Time, DecidedAt: decided.Time,
		})
	}
	return result, rows.Err()
}
