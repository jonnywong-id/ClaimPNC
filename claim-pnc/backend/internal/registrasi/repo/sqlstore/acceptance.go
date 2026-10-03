package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// AcceptanceStore memenuhi registrasi.AcceptanceSource.
type AcceptanceStore struct {
	db *sql.DB
}

// NewAcceptanceStore membentuk penyimpanan akseptasi.
func NewAcceptanceStore(db *sql.DB) *AcceptanceStore { return &AcceptanceStore{db: db} }

// NextNumber menerbitkan nomor akseptasi seperti `PLA_DLA.prc` TIPE ALOD:
// KODE || TAHUN || ID_SITE || LPAD(ACCEPTLOD_SEQ, 15, '0').
func (s *AcceptanceStore) NextNumber(ctx context.Context, year int) (string, error) {
	tx, ok := txFrom(ctx)
	if !ok {
		return "", errors.New("registrasi/sqlstore: nomor akseptasi hanya boleh diterbitkan di dalam transaksi")
	}
	var site sql.NullString
	if err := tx.QueryRowContext(ctx, loadQuery("akseptasi_site")).Scan(&site); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca site aktif: %w", err)
	}
	var counter int64
	if err := tx.QueryRowContext(ctx, loadQuery("akseptasi_urut")).Scan(&counter); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca ACCEPTLOD_SEQ: %w", err)
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	yy := fmt.Sprintf("%02d", year%100)
	if _, err := tx.ExecContext(ctx, loadQuery("akseptasi_nomor_sisip"),
		strings.ToUpper(hex.EncodeToString(key)), registrasi.AcceptanceCode, trimmed(site), yy,
		strconv.FormatInt(counter, 10)); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: mencatat nomor akseptasi: %w", err)
	}
	return registrasi.PLANumber(registrasi.AcceptanceCode, year, trimmed(site), counter), nil
}

// Save menulis isian akseptasi satu baris adjustment: kolom yang sudah ada, lalu tujuh
// kolom migrasi 0013. Di dalam UnitOfWork, sesudah ClaimStore.Save.
func (s *AcceptanceStore) Save(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line registrasi.SettlementLine) error {
	exec := executorFrom(ctx, s.db)
	a := line.Acceptance
	keys := []any{claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq)}
	values := []any{
		emptyTextAsNil(a.Form.LODStatus), emptyTextAsNil(line.AcceptedNo), wallOrNil(a.AcceptedAt),
		emptyTextAsNil(a.Form.ReceiverID), emptyTextAsNil(a.ReceiverName),
		wallOrNil(a.Form.PrintDate), wallOrNil(a.Form.ReceiveDate),
	}
	if _, err := exec.ExecContext(ctx, loadQuery("akseptasi_simpan"), append(values, keys...)...); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan akseptasi: %w", err)
	}
	var lodValue any
	if a.Form.HasLODValue {
		lodValue = int64(a.Form.LODValue)
	}
	fields := []any{
		wallOrNil(a.Form.PayableDate), wallOrNil(a.Form.AnalystReceiveDate), lodValue,
		emptyTextAsNil(a.Form.Type), emptyTextAsNil(a.Form.CommitteeName),
		emptyTextAsNil(a.Form.Remark), emptyTextAsNil(a.Form.MinutesNote),
	}
	if _, err := exec.ExecContext(ctx, loadQuery("akseptasi_isian_simpan"), append(fields, keys...)...); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan isian akseptasi (kolom migrasi 0013): %w", err)
	}
	return nil
}

// RecordLODPrint memenuhi registrasi.AcceptanceSource.
func (s *AcceptanceStore) RecordLODPrint(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, printedAt time.Time, lodType string) error {
	if _, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("lod_cetak_simpan"),
		wallOrNil(printedAt), emptyTextAsNil(lodType), claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq)); err != nil {
		return fmt.Errorf("registrasi/sqlstore: mencatat Print LOD: %w", err)
	}
	return nil
}

// SetLODType memenuhi registrasi.AcceptanceSource.
func (s *AcceptanceStore) SetLODType(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, lodType string) error {
	res, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("lod_tipe_simpan"),
		emptyTextAsNil(lodType), claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq))
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menyimpan Tipe LOD: %w", err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return fmt.Errorf("registrasi/sqlstore: baris adjustment %s/%d/%d tidak ditemukan", objectID, coverageSeq, adjustmentSeq)
	}
	return nil
}

// Fields membaca tujuh isian akseptasi migrasi 0013 ke baris.
func (s *AcceptanceStore) Fields(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int, line *registrasi.SettlementLine) error {
	var (
		payable, analyst                 sql.NullTime
		lodValue                         sql.NullInt64
		kind, committee, remark, minutes sql.NullString
	)
	err := s.db.QueryRowContext(ctx, loadQuery("akseptasi_isian_ambil"),
		claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq)).
		Scan(&payable, &analyst, &lodValue, &kind, &committee, &remark, &minutes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: membaca isian akseptasi (kolom migrasi 0013): %w", err)
	}
	f := &line.Acceptance.Form
	f.PayableDate, f.AnalystReceiveDate = wallWIB(payable.Time), wallWIB(analyst.Time)
	f.LODValue, f.HasLODValue = registrasi.Money(lodValue.Int64), lodValue.Valid
	f.Type, f.CommitteeName = trimmed(kind), trimmed(committee)
	f.Remark, f.MinutesNote = remark.String, minutes.String
	return nil
}

// OtherDLA membaca DLA adjustment lain yang sudah disetujui LOD (`ValidationDLA_Act`).
func (s *AcceptanceStore) OtherDLA(ctx context.Context, claimID, objectID string, coverageSeq, adjustmentSeq int) ([]registrasi.AcceptanceDLAState, error) {
	rows, err := executorFrom(ctx, s.db).QueryContext(ctx, loadQuery("akseptasi_dla_lain"),
		claimID, objectID, strconv.Itoa(coverageSeq), strconv.Itoa(adjustmentSeq))
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca DLA adjustment lain: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []registrasi.AcceptanceDLAState
	for rows.Next() {
		var number, printed, sent sql.NullString
		if err := rows.Scan(&number, &printed, &sent); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris DLA: %w", err)
		}
		out = append(out, registrasi.AcceptanceDLAState{
			Number: trimmed(number), Printed: trimmed(printed) == "1", Sent: trimmed(sent) == "1",
		})
	}
	return out, rows.Err()
}

// AddHistory menulis satu baris riwayat klaim.
func (s *AcceptanceStore) AddHistory(ctx context.Context, caseID, note, user string, at time.Time) error {
	_, err := executorFrom(ctx, s.db).ExecContext(ctx, loadQuery("riwayat_sisip"),
		caseID, wallOrNil(at), registrasi.Truncate(note, 150), user)
	if err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis riwayat klaim: %w", err)
	}
	return nil
}

// progressJSON adalah JSONSTATUS_PROGRESS2 kategori UPDATE, persis keluaran
// PROGRESS_CLAIM_PNC.prc (diperiksa terhadap baris "Auto Akseptasi By LOD" di Oracle).
func progressJSON(progress2 string) string {
	return `{"pxObjClass":"ASM-FW-GCNMFW-Data-ClaimData","ObjectList":[ {"BranchName":"` +
		strings.ReplaceAll(progress2, `"`, `\"`) + `","pxObjClass":"ASM-FW-GCNMFW-Data-Object" }]}`
}

// OpenPosition memenuhi registrasi.AcceptanceSource.
func (s *AcceptanceStore) OpenPosition(ctx context.Context, claimNumber, position string) (string, error) {
	var id sql.NullString
	err := executorFrom(ctx, s.db).QueryRowContext(ctx, loadQuery("progres_posisi_terbuka"), claimNumber, position).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca posisi progres %s: %w", position, err)
	}
	return trimmed(id), nil
}

// legacyUser adalah USER_INPUT progres: identitas lama operator (GetOperatorIDs) bila ada.
func legacyUser(ctx context.Context, exec executor, user string) (string, error) {
	var old sql.NullString
	switch err := exec.QueryRowContext(ctx, loadQuery("operator_lama"), user).Scan(&old); {
	case err == nil:
		if v := trimmed(old); v != "" {
			return v, nil
		}
	case !errors.Is(err, sql.ErrNoRows):
		return "", fmt.Errorf("registrasi/sqlstore: membaca identitas lama pengguna: %w", err)
	}
	return user, nil
}

// StartProgress menjalankan PNCInsertProgressClaim kategori INSERT (PROGRESS_CLAIM_PNC.prc).
func (s *AcceptanceStore) StartProgress(ctx context.Context, p registrasi.ProgressStart) (string, error) {
	exec := executorFrom(ctx, s.db)
	user, err := legacyUser(ctx, exec, p.User)
	if err != nil {
		return "", err
	}
	var id int64
	if err := exec.QueryRowContext(ctx, loadQuery("progres_posisi_berikut"), p.ClaimNumber).Scan(&id); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: menomori posisi progres: %w", err)
	}
	if _, err := exec.ExecContext(ctx, loadQuery("progres_posisi_sisip"),
		id, p.ClaimNumber, emptyTextAsNil(p.CaseID), registrasi.ProgressOnProgress, wallOrNil(p.At), p.Position); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membuka posisi progres %s: %w", p.Position, err)
	}
	if _, err := exec.ExecContext(ctx, loadQuery("progres_sisip_buka"),
		p.ClaimNumber, p.Note, p.Progress1, p.Progress2, wallOrNil(p.At.Add(7*24*time.Hour)), user, id,
		progressJSON(p.Progress2), p.ClaimNumber); err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: menulis progres klaim: %w", err)
	}
	return strconv.FormatInt(id, 10), nil
}

// AddProgress menjalankan PNCInsertProgressClaim kategori UPDATE (PROGRESS_CLAIM_PNC.prc).
func (s *AcceptanceStore) AddProgress(ctx context.Context, p registrasi.ProgressUpdate) error {
	exec := executorFrom(ctx, s.db)
	user, err := legacyUser(ctx, exec, p.User)
	if err != nil {
		return err
	}

	var position any
	if id := strings.TrimSpace(p.PositionID); id != "" {
		position = id
		if _, err := exec.ExecContext(ctx, loadQuery("progres_posisi_selesai"),
			p.Position, wallOrNil(p.At), p.ClaimNumber, id); err != nil {
			return fmt.Errorf("registrasi/sqlstore: menutup posisi progres: %w", err)
		}
	}
	if _, err := exec.ExecContext(ctx, loadQuery("progres_sisip"),
		p.ClaimNumber, p.Note, p.Progress1, p.Progress2, user, position, progressJSON(p.Progress2),
		p.ClaimNumber); err != nil {
		return fmt.Errorf("registrasi/sqlstore: menulis progres klaim: %w", err)
	}
	return nil
}

// PolicyCaseID membaca Policy.CaseID polis pada PRODKE klaim: T_GENERAL lebih dulu, dokumen
// POLICYDATA sebagai cadangan.
func (s *AcceptanceStore) PolicyCaseID(ctx context.Context, policyNumber, prodKe string) (string, error) {
	number, prodKe := strings.TrimSpace(policyNumber), strings.TrimSpace(prodKe)
	for _, name := range []string{"premi_polis_caseid", "premi_polis_caseid_dokumen"} {
		var v sql.NullString
		err := s.db.QueryRowContext(ctx, loadQuery(name), number, prodKe).Scan(&v)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("registrasi/sqlstore: membaca CaseID polis: %w", err)
		}
		if id := trimmed(v); id != "" {
			return id, nil
		}
	}
	return "", nil
}

// OpenProtectionApproved memeriksa Open Protection premi yang disetujui.
func (s *AcceptanceStore) OpenProtectionApproved(ctx context.Context, policyNumber, claimID, claimNumber string) (bool, error) {
	var n int64
	if err := s.db.QueryRowContext(ctx, loadQuery("premi_open_protection"), policyNumber, claimID, claimNumber).Scan(&n); err != nil {
		return false, fmt.Errorf("registrasi/sqlstore: membaca Open Protection premi: %w", err)
	}
	return n > 0, nil
}

// TravelClientName membaca CLIENTNAME agen leader polis.
func (s *AcceptanceStore) TravelClientName(ctx context.Context, policyNumber string) (string, error) {
	var v sql.NullString
	err := s.db.QueryRowContext(ctx, loadQuery("premi_agen_travel"), policyNumber).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: membaca agen polis Travel: %w", err)
	}
	return trimmed(v), nil
}

// wallOrNil menulis waktu sebagai jam dinding WIB untuk kolom DATE warisan.
func wallOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	w := t.In(clock.ZoneWIB)
	return time.Date(w.Year(), w.Month(), w.Day(), w.Hour(), w.Minute(), w.Second(), 0, time.UTC)
}

var _ registrasi.AcceptanceSource = (*AcceptanceStore)(nil)
