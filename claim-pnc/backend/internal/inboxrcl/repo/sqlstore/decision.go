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

	// PIC Teknik klaim — dibaca untuk KEDUA jalur sejak 2026-10-07, bukan hanya jalur yang
	// kembali ke analis.
	//
	// `ASSIGNED_OPERATOR_ID` kini SELALU berisi user teknis (keputusan Work Owner), termasuk
	// sesudah dokter menyetujui. Yang mengeluarkan klaim dari antrean dokter bukan lagi kolom
	// itu melainkan **tahap tugasnya** — penyaring E pada `inboxrcl.sql`. Tanpa penyaring itu
	// perubahan ini akan membuat klaim menetap di Inbox RCL selamanya.
	//
	// `WORKBASKET` tugasnya tetap `RCLPUCL`: klaim memang berpindah ke antrean bersama, dan
	// itu dinyatakan oleh tugasnya — bukan lagi dengan menitipkan nama antrean ke kolom
	// pemilik.
	pic, picErr := technicalPIC(ctx, tx, storedNumber, userTeknis)
	if picErr != nil && !errors.Is(picErr, inboxrcl.ErrTechnicalPICUnknown) {
		return inboxrcl.Outcome{}, picErr
	}

	// SESUDAH DOKTER MENYETUJUI, `ASSIGNED_OPERATOR_ID` berisi ADMIN KLAIM — bukan nama
	// antrean `RCLPUCL`, dan bukan pula PIC Teknik (Work Owner, 2026-10-07).
	//
	// Klaimnya memang berpindah ke antrean bersama RCL/PUCL; yang menyatakan itu adalah
	// **tugasnya** (`WORKBASKET` di bawah tetap `RCLPUCL`), bukan kolom pemilik. Menitipkan
	// nama antrean ke kolom pemilik membuat baris itu tidak dapat dicocokkan dengan satu
	// orang pun — dan itulah yang sudah terbukti mengosongkan ketiga tab Inbox RCL/PUCL
	// ketika kuerinya menyaring literal `'RCLPUCL'` (`inboxrclpucl.sql` butir 1: kolomnya
	// pada data nyata berisi NAMA ORANG).
	//
	// Yang mengeluarkan klaim dari antrean dokter bukan kolom ini melainkan penyaring E —
	// tahap tugasnya sudah bukan `rcl-dokter` lagi.
	admin, adminErr := claimAdmin(ctx, tx, storedNumber)
	if adminErr != nil {
		return inboxrcl.Outcome{}, adminErr
	}

	// Urutan cadangan, dan tidak satu pun boleh kosong: baris ber-`ASSIGNED_OPERATOR_ID`
	// kosong tidak terbaca penyaring A mana pun, sehingga klaimnya hilang dari SETIAP layar
	// tanpa satu pun galat. Admin klaim lebih dulu; bila tidak ada — baris lama atau klaim
	// yang lahir di Pega — PIC Teknik; bila itu pun tidak ada, nama antrean seperti dulu.
	assignee, owner, workbasket := admin, "", inboxrcl.WorkbasketRCLPUCL
	if assignee == "" {
		assignee = pic
	}
	if assignee == "" {
		assignee = inboxrcl.WorkbasketRCLPUCL
	}
	if out.NextQueue == inboxrcl.QueueWorklist {
		// Jalur kembali ke analis MENUNTUT orangnya: tugas worklist tanpa pemilik mandek
		// tanpa ada yang tahu.
		if picErr != nil {
			return inboxrcl.Outcome{}, picErr
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

// claimAdmin membaca admin klaim dari `T_CLAIM_PNC.ADMINKLAIM` — pemilik
// `ASSIGNED_OPERATOR_ID` sesudah dokter menyetujui.
//
// Kosong BUKAN galat, berbeda dengan `technicalPIC`: jalur Setuju punya rantai cadangan
// (PIC Teknik, lalu nama antrean), sehingga menolak keputusannya hanya karena satu kolom
// lama tidak terisi akan memacetkan klaim yang hari ini berjalan normal.
func claimAdmin(ctx context.Context, tx *sql.Tx, number string) (string, error) {
	var admin sql.NullString
	if err := tx.QueryRowContext(ctx, query("decision_claim_admin"), number).Scan(&admin); err != nil &&
		!errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("membaca admin klaim %s: %w", number, err)
	}
	return strings.TrimSpace(admin.String), nil
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
