package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxrcl"
)

// Decide menjalankan keputusan dokter RCL dalam SATU transaksi — lihat decision.sql.
func (r *Repo) Decide(ctx context.Context, cmd inboxrcl.DecisionCommand) (inboxrcl.Outcome, error) {
	number := strings.TrimSpace(cmd.ClaimNumber)
	operator := strings.TrimSpace(cmd.Operator)
	if number == "" || operator == "" {
		return inboxrcl.Outcome{}, inboxrcl.ErrClaimNotFound
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxrcl.Outcome{}, fmt.Errorf("memulai transaksi keputusan RCL: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	state, storedNumber, userTeknis, err := lockForDecision(ctx, tx, number, operator)
	if err != nil {
		return inboxrcl.Outcome{}, err
	}

	out, err := inboxrcl.Plan(state, cmd.Decision, cmd.DoctorReason, cmd.At)
	if err != nil {
		return inboxrcl.Outcome{}, err
	}

	// Pemilik tahap berikutnya.
	assignee, owner, workbasket := inboxrcl.WorkbasketRCLPUCL, "", inboxrcl.WorkbasketRCLPUCL
	if out.NextQueue == inboxrcl.QueueWorklist {
		pic, err := technicalPIC(ctx, tx, storedNumber, userTeknis)
		if err != nil {
			return inboxrcl.Outcome{}, err
		}
		assignee, owner, workbasket = pic, pic, ""
	}

	if _, err := tx.ExecContext(ctx, query("decision_update_pucl"),
		nilIfEmpty(out.StatusClaim),
		nilIfEmpty(out.StatusKlaim),
		nilIfEmpty(out.StatusCase),
		nilIfEmpty(out.PUCLApprove),
		nilIfZero(out.LetterPrintedAt),
		nilIfZero(out.SentToPUCLAt),
		nilIfZero(out.AnalystSentAt),
		nilIfZero(out.ClaimAgeFrom),
		nilIfEmpty(out.DoctorReason),
		assignee,
		storedNumber,
	); err != nil {
		return inboxrcl.Outcome{}, fmt.Errorf("menulis TC_PNC_PUCL klaim %s: %w", number, err)
	}

	if _, err := tx.ExecContext(ctx, query("decision_update_worklist"),
		nilIfEmpty(out.StatusClaim),
		nilIfZero(out.AnalystSentAt),
		nilIfEmpty(owner),
		out.NextStageName,
		storedNumber,
	); err != nil {
		return inboxrcl.Outcome{}, fmt.Errorf("menulis T_CLAIMLIST_ADMIN klaim %s: %w", number, err)
	}

	if err := moveTask(ctx, tx, storedNumber, out, owner, workbasket, cmd.At); err != nil {
		return inboxrcl.Outcome{}, err
	}

	for _, note := range out.History {
		if _, err := tx.ExecContext(ctx, query("decision_insert_history"),
			inboxrcl.HistoryKey(storedNumber), note, strings.ToUpper(operator),
		); err != nil {
			return inboxrcl.Outcome{}, fmt.Errorf("menulis riwayat klaim %s: %w", number, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return inboxrcl.Outcome{}, fmt.Errorf("menyimpan keputusan RCL klaim %s: %w", number, err)
	}
	return out, nil
}

// lockForDecision mengunci dan membaca baris TC_PNC_PUCL. Nol baris berarti klaim tidak lagi
// di antrean pemanggil.
func lockForDecision(
	ctx context.Context, tx *sql.Tx, number, operator string,
) (inboxrcl.PUCLState, string, string, error) {
	var (
		caseID, mode, statusCase, puclApprove, statusKlaim, doctorReason, userTeknis sql.NullString
		dischargedAt, dateOfLoss, letterPrintedAt                                    sql.NullTime
	)
	err := tx.QueryRowContext(ctx, query("decision_lock"),
		number,
		operator,
		inboxrcl.StatusKerjaSelesai,
		string(inboxrcl.ModeRCL),
		string(inboxrcl.ModeMSIG),
	).Scan(
		&caseID, &mode, &statusCase, &puclApprove, &statusKlaim,
		&dischargedAt, &dateOfLoss, &letterPrintedAt, &doctorReason, &userTeknis,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return inboxrcl.PUCLState{}, "", "", inboxrcl.ErrClaimNotFound
	case err != nil:
		return inboxrcl.PUCLState{}, "", "", fmt.Errorf("mengunci TC_PNC_PUCL klaim %s: %w", number, err)
	}

	state := inboxrcl.PUCLState{
		Mode:         inboxrcl.Mode(strings.TrimSpace(mode.String)),
		StatusCase:   strings.TrimSpace(statusCase.String),
		PUCLApprove:  strings.TrimSpace(puclApprove.String),
		StatusKlaim:  strings.TrimSpace(statusKlaim.String),
		DoctorReason: doctorReason.String,
	}
	if dischargedAt.Valid {
		state.DischargedAt = dischargedAt.Time.UTC()
	}
	if dateOfLoss.Valid {
		state.DateOfLoss = dateOfLoss.Time.UTC()
	}
	if letterPrintedAt.Valid {
		state.LetterPrintedAt = letterPrintedAt.Time.UTC()
	}
	return state, strings.TrimSpace(caseID.String), strings.TrimSpace(userTeknis.String), nil
}

// technicalPIC membaca PIC Teknik dari T_CLAIM_PNC.PICTEKNIK. Bila kosong, dipakai
// TC_PNC_PUCL.USER_TEKNIS — kolom tabel yang sama yang menyimpan PIC Teknik klaim itu.
func technicalPIC(ctx context.Context, tx *sql.Tx, number, userTeknis string) (string, error) {
	var pic sql.NullString
	if err := tx.QueryRowContext(ctx, query("decision_technical_pic"), number).Scan(&pic); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("membaca PIC Teknik klaim %s: %w", number, err)
	}
	if value := strings.TrimSpace(pic.String); value != "" {
		return value, nil
	}
	if value := strings.TrimSpace(userTeknis); value != "" {
		return value, nil
	}
	return "", inboxrcl.ErrTechnicalPICUnknown
}

// moveTask menutup tugas terbuka klaim dan membuka tugas tahap berikutnya.
func moveTask(
	ctx context.Context, tx *sql.Tx,
	number string, out inboxrcl.Outcome, owner, workbasket string, at time.Time,
) error {
	var key sql.NullString
	err := tx.QueryRowContext(ctx, query("decision_claim_key"), number).Scan(&key)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("membaca kunci tugas klaim %s: %w", number, err)
	}
	claimKey := strings.TrimSpace(key.String)
	if claimKey == "" {
		// Klaim yang belum pernah punya tugas di aplikasi ini — kunci yang sama dengan riwayat.
		claimKey = inboxrcl.HistoryKey(number)
	}

	now := at.UTC()
	if _, err := tx.ExecContext(ctx, query("decision_close_tasks"), now, out.Ticket, number); err != nil {
		return fmt.Errorf("menutup tugas terbuka klaim %s: %w", number, err)
	}

	var takenAt any
	if owner != "" {
		takenAt = now
	}
	if _, err := tx.ExecContext(ctx, query("decision_open_task"),
		newTaskID(),
		claimKey,
		number,
		out.NextStage,
		out.NextQueue,
		nilIfEmpty(workbasket),
		nilIfEmpty(owner),
		now,
		takenAt,
	); err != nil {
		return fmt.Errorf("membuka tugas %s klaim %s: %w", out.NextStageName, number, err)
	}
	return nil
}

// nilIfZero mengirim waktu nol sebagai NULL.
func nilIfZero(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}

// newTaskID membangkitkan pengenal tugas 128 bit dalam heksadesimal — bentuk yang sama dengan
// modul alur yang mengisi tabel yang sama.
func newTaskID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("inboxrcl/sqlstore: sumber acak tidak tersedia: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}
