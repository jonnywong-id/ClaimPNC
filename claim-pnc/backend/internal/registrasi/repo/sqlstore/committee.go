package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// CommitteeStore menyimpan kasus komite klaim PNCN di POOLDATA.T_CLAIM_KOMITE_LIST.
type CommitteeStore struct{ db *sql.DB }

// NewCommitteeStore membentuk penyimpan kasus komite di atas sebuah koneksi.
func NewCommitteeStore(db *sql.DB) *CommitteeStore { return &CommitteeStore{db: db} }

func (s *CommitteeStore) exec(ctx context.Context) executor { return executorFrom(ctx, s.db) }

// NextCaseID menerbitkan KOMITE_ID berikutnya: KMTN- ditambah lima digit.
//
// Nomor diambil dari nomor terbesar yang sudah ada, di dalam transaksi pemanggil. Dua transfer
// yang berlangsung pada saat yang sama dapat memperoleh nomor yang sama; dengan beban komite
// PNCN hari ini peluangnya kecil, dan penggantinya — sequence — menuntut perubahan skema.
func (s *CommitteeStore) NextCaseID(ctx context.Context) (string, error) {
	var next int64
	if err := s.exec(ctx).QueryRowContext(ctx, loadQuery("komite_nomor_berikut")).Scan(&next); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: menerbitkan nomor komite: %w", err)
	}
	if next > 99999 {
		return "", fmt.Errorf("registrasi/sqlstore: nomor komite %d melampaui KOMITE_ID VARCHAR2(10)", next)
	}
	return fmt.Sprintf("%s%05d", registrasi.CommitteeCasePrefix, next), nil
}

// Save menulis seluruh anggota kasus: perbarui-atau-sisip per (KOMITE_ID, NAMAKOMITE, KOMITEKE).
func (s *CommitteeStore) Save(ctx context.Context, c registrasi.CommitteeCase) error {
	exec := s.exec(ctx)
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

// Get membaca satu kasus komite.
func (s *CommitteeStore) Get(ctx context.Context, caseID string) (registrasi.CommitteeCase, error) {
	members, err := s.query(ctx, "komite_ambil", strings.TrimSpace(caseID))
	if err != nil {
		return registrasi.CommitteeCase{}, err
	}
	if len(members) == 0 {
		return registrasi.CommitteeCase{}, registrasi.ErrCommitteeNotFound
	}
	return registrasi.CommitteeCase{ID: members[0].CaseID, ClaimNumber: members[0].ClaimNumber, Members: members}, nil
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
