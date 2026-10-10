package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/dashboardclaim"
)

// FindClaimDetail membaca rincian satu klaim beserta dokumen JSON-nya.
//
// Menggantikan langkah pertama `setDataViewKlaim_Act` — lihat rincian.sql.
func (r *Repo) FindClaimDetail(
	ctx context.Context,
	claimID string,
) (dashboardclaim.ClaimDetail, error) {
	key := strings.TrimSpace(claimID)
	if key == "" {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}

	var (
		nomor, status, statusKlaim, pic, admin, dokumen sql.NullString
		kunci                                           sql.NullString
		didaftarkan                                     sql.NullTime
	)

	err := r.db.QueryRowContext(ctx, query("klaim_rincian"), key).Scan(
		&nomor, &kunci, &status, &statusKlaim, &pic, &admin, &didaftarkan, &dokumen)

	if errors.Is(err, sql.ErrNoRows) {
		return dashboardclaim.ClaimDetail{}, dashboardclaim.ErrClaimNotFound
	}
	if err != nil {
		if needsDBA(err) {
			return dashboardclaim.ClaimDetail{}, fmt.Errorf(
				"%w: %v", dashboardclaim.ErrAssignmentUnavailable, err)
		}
		return dashboardclaim.ClaimDetail{}, fmt.Errorf(
			"dashboardclaim/sqlstore: membaca rincian klaim: %w", err)
	}

	document, err := uraiDokumen(text(dokumen))
	if err != nil {
		return dashboardclaim.ClaimDetail{}, err
	}

	return dashboardclaim.ClaimDetail{
		ClaimID:       text(kunci),
		ClaimNumber:   text(nomor),
		ProcessStatus: text(status),
		ClaimStatus:   text(statusKlaim),
		TechnicalPIC:  text(pic),
		AdminPNC:      text(admin),
		RegisteredAt:  waktuISO(didaftarkan),
		Document:      document,
	}, nil
}

// uraiDokumen mengurai isi kolom `DATA_JSON`.
//
// Tiga keadaan yang DIBEDAKAN, dan pembedaannya yang penting:
//
//  1. kolomnya NULL atau kosong  → peta kosong, bukan galat. Klaim yang belum punya baris di
//     JSON_KLAIM memang begitu, dan ia tetap dapat dibuka.
//  2. terurai                    → petanya dikembalikan apa adanya.
//  3. ADA tetapi tidak terurai   → GALAT. Menelannya menjadi rincian kosong memberitahu
//     pengguna "klaim ini belum diisi" — pernyataan yang salah dan tidak dapat dibantah
//     dari layar.
func uraiDokumen(raw string) (map[string]any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]any{}, nil
	}

	var document map[string]any
	if err := json.Unmarshal([]byte(trimmed), &document); err != nil {
		// Isinya TIDAK ikut dilaporkan: dokumen klaim memuat data nasabah (`D-69`), dan galat
		// yang mengutip isinya akan menuliskannya ke log.
		return nil, fmt.Errorf("%w: %v", dashboardclaim.ErrClaimDetailUnreadable, err)
	}
	if document == nil {
		// `null` terurai tanpa galat menjadi peta nil. Dibedakan dari kosong supaya pemanggil
		// tidak perlu menjaga nil.
		return map[string]any{}, nil
	}
	return document, nil
}

// waktuISO memformat stempel waktu, atau kosong bila tidak ada.
//
// Lapisan HTTP yang memformatnya menurut zona tampilan; di sini bentuknya baku supaya adapter
// memori dan adapter SQL menghasilkan teks yang sama.
func waktuISO(t sql.NullTime) string {
	if !t.Valid || t.Time.IsZero() {
		return ""
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// CheckClaimDetailReadable membuktikan kedua tabel rincian terbaca.
//
// Dipanggil `claimpnc -periksa`. Ia menyebut kolomnya satu per satu — termasuk `DATA_JSON`,
// kolom yang berdampingan dengan `DATA_JSONBLOB` pada tabel yang sama. Probe yang hanya
// membuktikan tabelnya ada akan lulus terhadap kolom yang salah di antara keduanya.
func (r *Repo) CheckClaimDetailReadable(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, query("klaim_rincian_check")); err != nil {
		return fmt.Errorf(
			"rincian klaim tidak terbaca akun aplikasi — periksa POOLDATA.T_CLAIMLIST_ADMIN "+
				"dan POOLDATA.JSON_KLAIM(DATA_JSON): %w", err)
	}
	return nil
}
